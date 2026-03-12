package server

import (
	"time"

	"github.com/shricodev/gophercast/pkg/protocol"
)

// broadcastControl sends a control message to all connected clients.
func (s *AudioServer) broadcastControl(msgType protocol.MessageType, data any) {
	envelope, err := protocol.MarshalEnvelope(msgType, data)
	if err != nil {
		s.logger.Error("failed to marshal broadcast", "type", string(msgType), "error", err)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		select {
		case client.sendCtrl <- envelope:
		default:
			s.logger.Warn("dropping control message for slow client", "client", client.id)
		}
	}
}

// broadcastAudioFrame sends a binary audio frame to all connected clients.
func (s *AudioServer) broadcastAudioFrame(frame *protocol.AudioFrame) {
	data := frame.MarshalBinary()

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		select {
		case client.sendAudio <- data:
		default:
			// Drop frame for slow client — audio must not block
		}
	}
}

// sendControlTo sends a control message to a single client.
func (s *AudioServer) sendControlTo(client *Client, msgType protocol.MessageType, data any) {
	envelope, err := protocol.MarshalEnvelope(msgType, data)
	if err != nil {
		s.logger.Error("failed to marshal control message", "type", string(msgType), "error", err)
		return
	}

	select {
	case client.sendCtrl <- envelope:
	default:
		s.logger.Warn("dropping control message for slow client", "client", client.id)
	}
}

// broadcastPlaybackStart sends per-client start/track-change messages with
// adjusted start times to compensate for each client's audio pipeline latency.
// The target time T is when sound should exit all speakers simultaneously.
// Each client's start_at_ns is set to T - audioLatencyNs so that despite
// different pipeline latencies, sound arrives at speakers at the same moment.
func (s *AudioServer) broadcastPlaybackStart(msgType protocol.MessageType, trackTitle string, sampleRate, channels int, leadTime time.Duration) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Find the maximum audio latency across all clients.
	var maxLatencyNs int64
	for _, client := range s.clients {
		if client.audioLatencyNs > maxLatencyNs {
			maxLatencyNs = client.audioLatencyNs
		}
	}

	// Target time: when sound should exit all speakers.
	// Add maxLatency so the highest-latency client still has enough lead time.
	targetTime := time.Now().Add(leadTime + time.Duration(maxLatencyNs))

	for _, client := range s.clients {
		// Each client starts writing PCM at targetTime - its own latency.
		// The client with the highest latency starts earliest.
		clientStartNs := targetTime.Add(-time.Duration(client.audioLatencyNs)).UnixNano()

		var data any
		if msgType == protocol.MsgStartPlayback {
			data = protocol.StartPlaybackMsg{
				TrackTitle: trackTitle,
				SampleRate: sampleRate,
				Channels:   channels,
				StartAtNs:  clientStartNs,
			}
		} else {
			data = protocol.TrackChangeMsg{
				TrackTitle: trackTitle,
				SampleRate: sampleRate,
				Channels:   channels,
				StartAtNs:  clientStartNs,
			}
		}

		envelope, err := protocol.MarshalEnvelope(msgType, data)
		if err != nil {
			s.logger.Error("failed to marshal playback message", "client", client.id, "error", err)
			continue
		}

		select {
		case client.sendCtrl <- envelope:
		default:
			s.logger.Warn("dropping control message for slow client", "client", client.id)
		}
	}
}

// broadcastClientList sends the current client list to all clients.
func (s *AudioServer) broadcastClientList() {
	clients := s.Clients()
	s.broadcastControl(protocol.MsgClientList, protocol.ClientListMsg{
		Clients: clients,
	})
}
