package client

import (
	"fmt"
	"os"
	"time"
)

// FileSink writes raw PCM audio data to a file. Mostly useful for debugging.
type FileSink struct {
	path string
	file *os.File
}

func NewFileSink(path string) *FileSink {
	return &FileSink{path: path}
}

func (s *FileSink) Init(sampleRate, channels int) error {
	f, err := os.Create(s.path)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	s.file = f
	return nil
}

func (s *FileSink) Write(p []byte) (int, error) {
	if s.file == nil {
		return 0, fmt.Errorf("file sink not initialized")
	}
	return s.file.Write(p)
}

func (s *FileSink) Latency() time.Duration {
	return 0
}

func (s *FileSink) Close() error {
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}
