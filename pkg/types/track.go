package types

import (
	"errors"
	"time"
)

type Track struct {
	Title     string
	Path      Path
	Duration  time.Duration
	Source    Source
	IsPlaying bool
}

func (t Track) Validate() error {
	if !t.Source.IsValid() {
		return errors.New("invalid source type:" + t.Source.String())
	}

	if !t.Path.Exists() {
		return errors.New("invalid path:" + t.Path.String())
	}

	return nil
}
