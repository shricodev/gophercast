package client

import "time"

// AudioSink is the interface for audio output backends.
type AudioSink interface {
	Init(sampleRate, channels int) error
	Write(p []byte) (int, error)
	Close() error
	// Latency returns the estimated audio output pipeline latency.
	// This is the delay between writing PCM data and sound exiting the speaker.
	Latency() time.Duration
}
