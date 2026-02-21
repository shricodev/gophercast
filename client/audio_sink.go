package client

import (
	"fmt"
	"io"

	"github.com/ebitengine/oto/v3"
)

// SystemAudioSink plays PCM audio through the system audio output using oto.
type SystemAudioSink struct {
	otoCtx    *oto.Context
	player    *oto.Player
	pipeR     *io.PipeReader
	pipeW     *io.PipeWriter
	initiated bool
}

// NewSystemAudioSink creates a new SystemAudioSink.
func NewSystemAudioSink() *SystemAudioSink {
	return &SystemAudioSink{}
}

// Init initializes the audio context and player for the given sample rate and channels.
func (s *SystemAudioSink) Init(sampleRate, channels int) error {
	if s.initiated {
		s.Close()
	}

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
	s.pipeR, s.pipeW = io.Pipe()
	s.player = otoCtx.NewPlayer(s.pipeR)
	s.player.Play()
	s.initiated = true

	return nil
}

// Write writes PCM audio bytes to the player pipe.
func (s *SystemAudioSink) Write(p []byte) (int, error) {
	if s.pipeW == nil {
		return 0, fmt.Errorf("audio sink not initialized")
	}
	return s.pipeW.Write(p)
}

// Close stops playback and releases resources.
func (s *SystemAudioSink) Close() error {
	if s.pipeW != nil {
		s.pipeW.Close()
	}
	if s.player != nil {
		s.player.Close()
	}
	if s.pipeR != nil {
		s.pipeR.Close()
	}
	s.initiated = false
	return nil
}
