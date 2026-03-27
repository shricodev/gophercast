package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/shricodev/gophercast/internal/logger"
	"github.com/shricodev/gophercast/pkg/protocol"
	"github.com/shricodev/gophercast/pkg/types"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// AudioServer manages WebSocket client connections and streams audio.
type AudioServer struct {
	mu         sync.RWMutex
	state      protocol.ServerState
	playlist   *types.Playlist
	trackIndex int

	clients      map[string]*Client
	nextClientID atomic.Uint64

	httpServer *http.Server
	listener   net.Listener
	port       int
	logger     *logger.Logger

	ctx    context.Context
	cancel context.CancelFunc

	clientChangeCh chan []protocol.ClientInfo
	playbackDoneCh chan struct{}

	currentTrackTitle string
	playbackStart     time.Time
}

func NewAudioServer(playlist *types.Playlist, port int, log *logger.Logger) *AudioServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &AudioServer{
		state:          protocol.StateLobby,
		playlist:       playlist,
		clients:        make(map[string]*Client),
		port:           port,
		logger:         log,
		ctx:            ctx,
		cancel:         cancel,
		clientChangeCh: make(chan []protocol.ClientInfo, 1),
		playbackDoneCh: make(chan struct{}),
	}
}

func (s *AudioServer) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("listen on port %d: %w", s.port, err)
	}
	s.listener = ln
	s.port = ln.Addr().(*net.TCPAddr).Port

	s.httpServer = &http.Server{Handler: mux}

	go func() {
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.logger.Error("http server error", "error", err)
		}
	}()

	s.logger.ServerStarted(s.port)
	return nil
}

// registerClient adds a client to the server, or rejects it if playback is in progress.
// Returns true if the client was accepted.
func (s *AudioServer) registerClient(client *Client) bool {
	s.mu.Lock()
	if s.state == protocol.StatePlaying {
		s.mu.Unlock()
		rejectData, _ := protocol.MarshalEnvelope(protocol.MsgReject, protocol.RejectMsg{
			Reason: "playback already in progress, connect during lobby",
		})
		client.conn.WriteMessage(websocket.TextMessage, rejectData)
		client.conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "rejected"),
		)
		client.conn.Close()
		return false
	}
	s.clients[client.id] = client
	s.mu.Unlock()

	s.logger.ClientConnected(client.id, client.addr)
	s.broadcastClientList()
	s.pushClientChange()
	return true
}

// unregisterClient removes a client from the server and cleans up its channels.
func (s *AudioServer) unregisterClient(client *Client) {
	s.mu.Lock()
	if _, ok := s.clients[client.id]; ok {
		delete(s.clients, client.id)
		close(client.sendCtrl)
		close(client.sendAudio)
	}
	s.mu.Unlock()

	s.logger.ClientDisconnected(client.id, "disconnected")
	s.broadcastClientList()
	s.pushClientChange()
}

func (s *AudioServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("websocket upgrade failed", "error", err)
		return
	}

	id := fmt.Sprintf("client-%d", s.nextClientID.Add(1))
	client := &Client{
		id:        id,
		addr:      r.RemoteAddr,
		conn:      conn,
		sendCtrl:  make(chan []byte, 64),
		sendAudio: make(chan []byte, 256),
		server:    s,
	}

	if !s.registerClient(client) {
		return
	}

	go client.readMessages()
	go client.writeMessages()
}

func (s *AudioServer) StartPlayback() error {
	s.mu.Lock()
	if s.state != protocol.StateLobby {
		s.mu.Unlock()
		return fmt.Errorf("cannot start playback: server is in %s state", s.state)
	}

	if s.playlist == nil || s.playlist.Len() == 0 {
		s.mu.Unlock()
		return fmt.Errorf("no tracks to play")
	}

	s.state = protocol.StatePlaying
	s.trackIndex = 0
	s.mu.Unlock()

	go s.streamPlaylist()
	return nil
}

func (s *AudioServer) Stop() error {
	s.cancel()

	// route stop through the channel so writeMessages sends the ws close frame.
	// closing sendCtrl after is what causes writeMessages to exit cleanly.
	s.broadcastControl(protocol.MsgStopPlayback, protocol.StopPlaybackMsg{
		Reason: "user_stopped",
	})

	s.mu.Lock()
	for id, client := range s.clients {
		close(client.sendCtrl)
		close(client.sendAudio)
		delete(s.clients, id)
	}
	s.mu.Unlock()

	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *AudioServer) ClientChangeChan() <-chan []protocol.ClientInfo {
	return s.clientChangeCh
}

func (s *AudioServer) PlaybackDoneChan() <-chan struct{} {
	return s.playbackDoneCh
}

// Clients returns a snapshot of connected clients (safe to call from any goroutine).
func (s *AudioServer) Clients() []protocol.ClientInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := make([]protocol.ClientInfo, 0, len(s.clients))
	for _, c := range s.clients {
		clients = append(clients, c.Info())
	}
	return clients
}

func (s *AudioServer) Port() int {
	return s.port
}

func (s *AudioServer) State() protocol.ServerState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *AudioServer) CurrentTrackTitle() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentTrackTitle
}

func (s *AudioServer) PlaybackElapsed() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.playbackStart.IsZero() {
		return 0
	}
	return time.Since(s.playbackStart)
}

// pushClientChange sends the updated client list to the TUI (drops if nobody's listening).
func (s *AudioServer) pushClientChange() {
	clients := s.Clients()
	select {
	case s.clientChangeCh <- clients:
	default:
	}
}
