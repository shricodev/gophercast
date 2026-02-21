package client

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/shricodev/gophercast/pkg/protocol"
)

// AudioClient connects to a GopherCast server and plays audio.
type AudioClient struct {
	conn      *websocket.Conn
	sink      AudioSink
	name      string
	state     protocol.ServerState
	startAtNs int64
	nextSeq   uint32
	bufferMu  sync.Mutex
	buffer    []*protocol.AudioFrame
	playing   atomic.Bool
}

// NewAudioClient dials the server and sends a hello message.
func NewAudioClient(serverURL, name string, sink AudioSink) (*AudioClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to server: %w", err)
	}

	c := &AudioClient{
		conn: conn,
		sink: sink,
		name: name,
	}

	hello, err := protocol.MarshalEnvelope(protocol.MsgHello, protocol.HelloMsg{Name: name})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("marshal hello: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, hello); err != nil {
		conn.Close()
		return nil, fmt.Errorf("send hello: %w", err)
	}

	return c, nil
}

// Start is the main loop that reads messages from the server.
func (c *AudioClient) Start() error {
	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return nil
			}
			return fmt.Errorf("read message: %w", err)
		}

		switch msgType {
		case websocket.TextMessage:
			if err := c.handleControl(data); err != nil {
				return err
			}
		case websocket.BinaryMessage:
			c.handleAudioFrame(data)
		}
	}
}

// handleControl processes a JSON control message.
func (c *AudioClient) handleControl(data []byte) error {
	env, err := protocol.ParseEnvelope(data)
	if err != nil {
		return fmt.Errorf("parse control: %w", err)
	}

	switch env.Type {
	case protocol.MsgStartPlayback:
		var msg protocol.StartPlaybackMsg
		if err := json.Unmarshal(env.Data, &msg); err != nil {
			return fmt.Errorf("unmarshal start_playback: %w", err)
		}
		return c.handleStartPlayback(msg)

	case protocol.MsgStopPlayback:
		var msg protocol.StopPlaybackMsg
		if err := json.Unmarshal(env.Data, &msg); err != nil {
			return fmt.Errorf("unmarshal stop_playback: %w", err)
		}
		fmt.Printf("Playback stopped: %s\n", msg.Reason)
		c.playing.Store(false)
		c.sink.Close()
		if msg.Reason == "playlist_ended" || msg.Reason == "user_stopped" {
			return nil
		}

	case protocol.MsgTrackChange:
		var msg protocol.TrackChangeMsg
		if err := json.Unmarshal(env.Data, &msg); err != nil {
			return fmt.Errorf("unmarshal track_change: %w", err)
		}
		return c.handleTrackChange(msg)

	case protocol.MsgReject:
		var msg protocol.RejectMsg
		if err := json.Unmarshal(env.Data, &msg); err != nil {
			return fmt.Errorf("unmarshal reject: %w", err)
		}
		return fmt.Errorf("rejected by server: %s", msg.Reason)

	case protocol.MsgClientList:
		// Informational — ignore on client side

	case protocol.MsgServerState:
		var msg protocol.ServerStateMsg
		if err := json.Unmarshal(env.Data, &msg); err != nil {
			return fmt.Errorf("unmarshal server_state: %w", err)
		}
		c.state = msg.State
	}

	return nil
}

// handleStartPlayback initializes the audio sink and schedules playback.
func (c *AudioClient) handleStartPlayback(msg protocol.StartPlaybackMsg) error {
	fmt.Printf("Starting playback: %s (sample rate: %d, channels: %d)\n",
		msg.TrackTitle, msg.SampleRate, msg.Channels)

	if err := c.sink.Init(msg.SampleRate, msg.Channels); err != nil {
		return fmt.Errorf("init audio sink: %w", err)
	}

	c.startAtNs = msg.StartAtNs
	c.nextSeq = 0

	c.bufferMu.Lock()
	c.buffer = nil
	c.bufferMu.Unlock()

	go c.waitAndPlay()
	return nil
}

// handleTrackChange handles track transitions.
func (c *AudioClient) handleTrackChange(msg protocol.TrackChangeMsg) error {
	fmt.Printf("Track change: %s\n", msg.TrackTitle)

	c.playing.Store(false)
	c.startAtNs = msg.StartAtNs
	c.nextSeq = 0

	c.bufferMu.Lock()
	c.buffer = nil
	c.bufferMu.Unlock()

	if err := c.sink.Init(msg.SampleRate, msg.Channels); err != nil {
		return fmt.Errorf("re-init audio sink: %w", err)
	}

	go c.waitAndPlay()
	return nil
}

// waitAndPlay sleeps until startAtNs, then flushes the buffer and starts playing.
func (c *AudioClient) waitAndPlay() {
	startTime := time.Unix(0, c.startAtNs)
	sleepDuration := time.Until(startTime)
	if sleepDuration > 0 {
		time.Sleep(sleepDuration)
	}

	c.bufferMu.Lock()
	buffered := c.buffer
	c.buffer = nil
	c.bufferMu.Unlock()

	// Sort buffered frames by sequence number
	sort.Slice(buffered, func(i, j int) bool {
		return buffered[i].SeqNum < buffered[j].SeqNum
	})

	for _, frame := range buffered {
		c.sink.Write(frame.Payload)
		c.nextSeq = frame.SeqNum + 1
	}

	c.playing.Store(true)
}

// handleAudioFrame processes a binary audio frame.
func (c *AudioClient) handleAudioFrame(data []byte) {
	frame, err := protocol.UnmarshalAudioFrame(data)
	if err != nil {
		return
	}

	if !c.playing.Load() {
		c.bufferMu.Lock()
		c.buffer = append(c.buffer, frame)
		c.bufferMu.Unlock()
		return
	}

	if frame.SeqNum != c.nextSeq && c.nextSeq > 0 {
		fmt.Printf("frame gap: expected seq %d, got %d\n", c.nextSeq, frame.SeqNum)
	}

	c.sink.Write(frame.Payload)
	c.nextSeq = frame.SeqNum + 1
}

// Close closes the connection and the audio sink.
func (c *AudioClient) Close() {
	c.playing.Store(false)
	if c.conn != nil {
		c.conn.Close()
	}
	if c.sink != nil {
		c.sink.Close()
	}
}
