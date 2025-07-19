// Package types provides the data structures for Gophercast.
package types

import (
	"fmt"
	"time"
)

// Track represents a single audio track.
type Track struct {
	Title     string
	Path      Path
	Duration  time.Duration
	Source    Source
	IsPlaying bool
}

func (t Track) Validate() error {
	if !t.Source.IsValid() {
		return fmt.Errorf("invalid source type: %s", t.Source.String())
	}

	if !t.Path.Exists() {
		return fmt.Errorf("invalid path: %s", t.Path.String())
	}

	return nil
}
