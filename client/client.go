package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/shricodev/gophercast/pkg/protocol"
)

// Sentinel errors for clean shutdown handling.
var (
	ErrDisconnected = errors.New("disconnected")
	ErrRejected     = errors.New("rejected")
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
	closed    atomic.Bool

	// drift correction state
	drift          driftCorrector
	sampleRate     int
	bytesPerSample int

	// audioLatencyNs is the estimated audio pipeline latency reported to the server.
	audioLatencyNs int64
}

// NewAudioClient dials the server and sends a hello. Pass latencyOverride > 0 to skip auto-detection.
func NewAudioClient(serverURL, name string, sink AudioSink, latencyOverride time.Duration) (*AudioClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to server: %w", err)
	}

	latency := sink.Latency()
	if latencyOverride > 0 {
		latency = latencyOverride
	}

	c := &AudioClient{
		conn:           conn,
		sink:           sink,
		name:           name,
		audioLatencyNs: latency.Nanoseconds(),
	}

	hello, err := protocol.MarshalEnvelope(protocol.MsgHello, protocol.HelloMsg{
		Name:           name,
		AudioLatencyNs: c.audioLatencyNs,
	})
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

func (c *AudioClient) Start() error {
	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			return c.classifyReadError(err)
		}

		switch msgType {
		case websocket.TextMessage:
			if err := c.handleControl(data); err != nil {
				if errors.Is(err, ErrDisconnected) || errors.Is(err, ErrRejected) {
					return err
				}
				return err
			}
		case websocket.BinaryMessage:
			c.handleAudioFrame(data)
		}
	}
}

func (c *AudioClient) classifyReadError(err error) error {
	// user pressed ctrl+c, we already called Close()
	if c.closed.Load() {
		return nil
	}

	// normal ws close handshake
	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		return nil
	}

	// server closed abruptly (rejected, crashed, etc.)
	if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
		return ErrDisconnected
	}

	// connection reset by peer
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return ErrDisconnected
	}

	// triggered when we call conn.Close() ourselves
	if strings.Contains(err.Error(), "use of closed network connection") {
		return nil
	}

	if strings.Contains(err.Error(), "unexpected EOF") {
		return ErrDisconnected
	}

	return fmt.Errorf("connection error: %w", err)
}

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
		c.playing.Store(false)
		c.sink.Close()
		return ErrDisconnected

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
		return fmt.Errorf("%w: %s", ErrRejected, msg.Reason)

	case protocol.MsgClientList:
		// Informational

	case protocol.MsgServerState:
		var msg protocol.ServerStateMsg
		if err := json.Unmarshal(env.Data, &msg); err != nil {
			return fmt.Errorf("unmarshal server_state: %w", err)
		}
		c.state = msg.State
	}

	return nil
}

// handleStartPlayback kicks off sink init in a goroutine so the read loop stays unblocked.
// oto context creation can be slow on some systems.
func (c *AudioClient) handleStartPlayback(msg protocol.StartPlaybackMsg) error {
	fmt.Printf("Now playing: %s\n", msg.TrackTitle)

	c.startAtNs = msg.StartAtNs
	c.nextSeq = 0
	c.sampleRate = msg.SampleRate
	c.bytesPerSample = msg.Channels * 2

	c.bufferMu.Lock()
	c.buffer = nil
	c.bufferMu.Unlock()

	go func() {
		if err := c.sink.Init(msg.SampleRate, msg.Channels); err != nil {
			fmt.Printf("Error initializing audio: %v\n", err)
			return
		}
		c.waitAndPlay(msg.SampleRate, msg.Channels)
	}()
	return nil
}

func (c *AudioClient) handleTrackChange(msg protocol.TrackChangeMsg) error {
	fmt.Printf("Now playing: %s\n", msg.TrackTitle)

	c.playing.Store(false)
	c.startAtNs = msg.StartAtNs
	c.nextSeq = 0
	c.sampleRate = msg.SampleRate
	c.bytesPerSample = msg.Channels * 2

	c.bufferMu.Lock()
	c.buffer = nil
	c.bufferMu.Unlock()

	go func() {
		if err := c.sink.Init(msg.SampleRate, msg.Channels); err != nil {
			fmt.Printf("Error re-initializing audio: %v\n", err)
			return
		}
		c.waitAndPlay(msg.SampleRate, msg.Channels)
	}()
	return nil
}

func (c *AudioClient) waitAndPlay(sampleRate, channels int) {
	startTime := time.Unix(0, c.startAtNs)
	sleepDuration := time.Until(startTime)
	if sleepDuration > 0 {
		time.Sleep(sleepDuration)
	}

	// anchor drift corrector to the server's start time so all clients stay in sync
	c.drift.reset(sampleRate, channels, startTime)

	c.bufferMu.Lock()
	buffered := c.buffer
	c.buffer = nil
	c.bufferMu.Unlock()

	sort.Slice(buffered, func(i, j int) bool {
		return buffered[i].SeqNum < buffered[j].SeqNum
	})

	for _, frame := range buffered {
		c.sink.Write(frame.Payload)
		c.drift.written(len(frame.Payload))
		c.nextSeq = frame.SeqNum + 1
	}

	c.playing.Store(true)
}

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

	shouldCheck := frame.SeqNum%driftCheckInterval == 0
	payload := c.drift.correct(frame.Payload, shouldCheck)

	c.sink.Write(payload)
	c.drift.written(len(payload))
	c.nextSeq = frame.SeqNum + 1
}

func (c *AudioClient) Close() {
	c.closed.Store(true)
	c.playing.Store(false)
	if c.conn != nil {
		c.conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		c.conn.Close()
	}
	if c.sink != nil {
		c.sink.Close()
	}
}
