package client

import (
	"fmt"
	"io"
	"time"

	"github.com/ebitengine/oto/v3"
)

// rough estimate for the total latency on Linux (oto buffer + PipeWire/ALSA overhead)
const defaultOtoLatency = 50 * time.Millisecond

// SystemAudioSink plays PCM audio through the system audio output using oto.
type SystemAudioSink struct {
	otoCtx *oto.Context
	player *oto.Player
	pipeR  *io.PipeReader
	pipeW  *io.PipeWriter
}

func NewSystemAudioSink() *SystemAudioSink {
	return &SystemAudioSink{}
}

func (s *SystemAudioSink) Init(sampleRate, channels int) error {
	s.closePlayer()

	// oto context can only be created once per process, so reuse it
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

func (s *SystemAudioSink) Write(p []byte) (int, error) {
	if s.pipeW == nil {
		return 0, fmt.Errorf("audio sink not initialized")
	}
	return s.pipeW.Write(p)
}

// closePlayer tears down the pipe and player, but keeps the oto context alive.
func (s *SystemAudioSink) closePlayer() {
	if s.pipeW != nil {
		s.pipeW.Close()
		s.pipeW = nil
	}
	if s.player != nil {
		s.player = nil
	}
	if s.pipeR != nil {
		s.pipeR.Close()
		s.pipeR = nil
	}
}

func (s *SystemAudioSink) Latency() time.Duration {
	return defaultOtoLatency
}

func (s *SystemAudioSink) Close() error {
	s.closePlayer()
	return nil
}
