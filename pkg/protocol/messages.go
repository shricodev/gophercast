package protocol

import "encoding/json"

// MessageType identifies the type of a control message.
type MessageType string

const (
	MsgHello         MessageType = "hello"
	MsgServerState   MessageType = "server_state"
	MsgClientList    MessageType = "client_list"
	MsgStartPlayback MessageType = "start_playback"
	MsgStopPlayback  MessageType = "stop_playback"
	MsgTrackChange   MessageType = "track_change"
	MsgReject        MessageType = "reject"
)

type ServerState string

const (
	StateLobby   ServerState = "lobby"
	StatePlaying ServerState = "playing"
	StateStopped ServerState = "stopped"
)

// Envelope is the outer wrapper for all control messages.
type Envelope struct {
	Type MessageType     `json:"type"`
	Data json.RawMessage `json:"data"`
}

// HelloMsg is sent by the client immediately after connecting.
type HelloMsg struct {
	Name           string `json:"name"`
	AudioLatencyNs int64  `json:"audio_latency_ns,omitempty"`
}

type ServerStateMsg struct {
	State ServerState `json:"state"`
}

type ClientInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Addr string `json:"addr"`
}

// ClientListMsg is broadcast from server to all clients when the client list changes.
type ClientListMsg struct {
	Clients []ClientInfo `json:"clients"`
}

// StartPlaybackMsg is sent from server to all clients to begin playback.
type StartPlaybackMsg struct {
	TrackTitle string `json:"track_title"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	StartAtNs  int64  `json:"start_at_ns"`
}

// StopPlaybackMsg is sent from server to all clients to stop playback.
type StopPlaybackMsg struct {
	Reason string `json:"reason"`
}

// TrackChangeMsg is sent from server to all clients when the track changes.
type TrackChangeMsg struct {
	TrackTitle string `json:"track_title"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	StartAtNs  int64  `json:"start_at_ns"`
}

// RejectMsg is sent from server to a client to reject a late connection.
type RejectMsg struct {
	Reason string `json:"reason"`
}
