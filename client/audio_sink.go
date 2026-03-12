package client

import (
	"fmt"
	"io"
	"time"

	"github.com/ebitengine/oto/v3"
)

// defaultOtoLatency is the estimated audio pipeline latency for oto on Linux.
// Includes oto internal buffer + OS audio subsystem (PipeWire/PulseAudio/ALSA).
const defaultOtoLatency = 50 * time.Millisecond

// SystemAudioSink plays PCM audio through the system audio output using oto.
type SystemAudioSink struct {
	otoCtx *oto.Context
	player *oto.Player
	pipeR  *io.PipeReader
	pipeW  *io.PipeWriter
}

// NewSystemAudioSink creates a new SystemAudioSink.
func NewSystemAudioSink() *SystemAudioSink {
	return &SystemAudioSink{}
}

// Init initializes the audio context (once) and creates a new player.
func (s *SystemAudioSink) Init(sampleRate, channels int) error {
	// Close previous player/pipe if any
	s.closePlayer()

	// Create the oto context only once — it cannot be recreated
	if s.otoCtx == nil {
		op := &oto.NewContextOptions{
			SampleRate:   sampleRate,
			ChannelCount: channels,
			Format:       oto.FormatSignedInt16LE,
		}

		otoCtx, readyChan, err := oto.NewContext(op)
		if err != nil {
			return fmt.Errorf("create oto context: %w", err)
		}
		<-readyChan
		s.otoCtx = otoCtx
	}

	s.pipeR, s.pipeW = io.Pipe()
	s.player = s.otoCtx.NewPlayer(s.pipeR)
	s.player.Play()

	return nil
}

// Write writes PCM audio bytes to the player pipe.
func (s *SystemAudioSink) Write(p []byte) (int, error) {
	if s.pipeW == nil {
		return 0, fmt.Errorf("audio sink not initialized")
	}
	return s.pipeW.Write(p)
}

// closePlayer stops the current player and pipe without destroying the context.
func (s *SystemAudioSink) closePlayer() {
	if s.pipeW != nil {
		s.pipeW.Close()
		s.pipeW = nil
	}
	if s.player != nil {
		s.player.Close()
		s.player = nil
	}
	if s.pipeR != nil {
		s.pipeR.Close()
		s.pipeR = nil
	}
}

// Latency returns the estimated audio output latency.
func (s *SystemAudioSink) Latency() time.Duration {
	return defaultOtoLatency
}

// Close stops playback and releases resources.
func (s *SystemAudioSink) Close() error {
	s.closePlayer()
	return nil
}
