package client

import (
	"fmt"
	"os"
)

// FileSink writes raw PCM audio data to a file.
type FileSink struct {
	path string
	file *os.File
}

// NewFileSink creates a new FileSink that writes to the given path.
func NewFileSink(path string) *FileSink {
	return &FileSink{path: path}
}

// Init opens the output file.
func (s *FileSink) Init(sampleRate, channels int) error {
	f, err := os.Create(s.path)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	s.file = f
	return nil
}

// Write writes PCM bytes to the file.
func (s *FileSink) Write(p []byte) (int, error) {
	if s.file == nil {
		return 0, fmt.Errorf("file sink not initialized")
	}
	return s.file.Write(p)
}

// Close closes the output file.
func (s *FileSink) Close() error {
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}
