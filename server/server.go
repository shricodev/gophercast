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
	register     chan *Client
	unregister   chan *Client
	nextClientID atomic.Uint64

	httpServer *http.Server
	listener   net.Listener
	port       int
	logger     *logger.Logger

	ctx    context.Context
	cancel context.CancelFunc

	clientChangeCh chan []protocol.ClientInfo

	currentTrackTitle string
	playbackStart     time.Time
}

// NewAudioServer creates a new AudioServer instance.
func NewAudioServer(playlist *types.Playlist, port int, log *logger.Logger) *AudioServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &AudioServer{
		state:          protocol.StateLobby,
		playlist:       playlist,
		clients:        make(map[string]*Client),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		port:           port,
		logger:         log,
		ctx:            ctx,
		cancel:         cancel,
		clientChangeCh: make(chan []protocol.ClientInfo, 1),
	}
}

// ListenAndServe starts the HTTP server and the hub goroutine.
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

	go s.runHub()
	go func() {
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.logger.Error("http server error", "error", err)
		}
	}()

	s.logger.ServerStarted(s.port)
	return nil
}

// runHub processes client register/unregister events.
func (s *AudioServer) runHub() {
	for {
		select {
		case <-s.ctx.Done():
			return

		case client := <-s.register:
			s.mu.Lock()
			if s.state == protocol.StatePlaying {
				s.mu.Unlock()
				// Write reject directly to the wire — the write goroutine
				// may not be running yet so the channel route is unreliable.
				rejectData, _ := protocol.MarshalEnvelope(protocol.MsgReject, protocol.RejectMsg{
					Reason: "playback already in progress, connect during lobby",
				})
				client.conn.WriteMessage(websocket.TextMessage, rejectData)
				client.conn.WriteMessage(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, "rejected"),
				)
				client.conn.Close()
				continue
			}
			s.clients[client.id] = client
			s.mu.Unlock()

			s.logger.ClientConnected(client.id, client.addr)
			s.broadcastClientList()
			s.pushClientChange()

		case client := <-s.unregister:
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
	}
}

// handleWebSocket upgrades HTTP connections to WebSocket and registers clients.
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

	s.register <- client

	go client.readMessages()
	go client.writeMessages()
}

// StartPlayback transitions from lobby to playing and begins streaming.
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

// Stop gracefully shuts down the audio server.
func (s *AudioServer) Stop() error {
	s.cancel()

	s.broadcastControl(protocol.MsgStopPlayback, protocol.StopPlaybackMsg{
		Reason: "user_stopped",
	})

	s.mu.Lock()
	for _, client := range s.clients {
		client.conn.Close()
	}
	s.mu.Unlock()

	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// ClientChangeChan returns a channel that receives client list updates.
func (s *AudioServer) ClientChangeChan() <-chan []protocol.ClientInfo {
	return s.clientChangeCh
}

// Clients returns a thread-safe snapshot of connected clients.
func (s *AudioServer) Clients() []protocol.ClientInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := make([]protocol.ClientInfo, 0, len(s.clients))
	for _, c := range s.clients {
		clients = append(clients, c.Info())
	}
	return clients
}

// Port returns the port the server is listening on.
func (s *AudioServer) Port() int {
	return s.port
}

// State returns the current server state.
func (s *AudioServer) State() protocol.ServerState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// CurrentTrackTitle returns the title of the currently playing track.
func (s *AudioServer) CurrentTrackTitle() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentTrackTitle
}

// PlaybackElapsed returns how long the current track has been playing.
func (s *AudioServer) PlaybackElapsed() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.playbackStart.IsZero() {
		return 0
	}
	return time.Since(s.playbackStart)
}

// pushClientChange sends the current client list to the TUI channel (non-blocking).
func (s *AudioServer) pushClientChange() {
	clients := s.Clients()
	select {
	case s.clientChangeCh <- clients:
	default:
	}
}
