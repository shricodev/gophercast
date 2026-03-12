package client

import "time"

// AudioSink is the interface for audio output backends.
type AudioSink interface {
	Init(sampleRate, channels int) error
	Write(p []byte) (int, error)
	Close() error
	// Latency is how long it takes from writing PCM to hearing it out the speaker.
	Latency() time.Duration
}
