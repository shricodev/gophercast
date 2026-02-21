package client

// AudioSink is the interface for audio output backends.
type AudioSink interface {
	Init(sampleRate, channels int) error
	Write(p []byte) (int, error)
	Close() error
}
