// Package types is a collection of custom types
package types

import (
	"errors"
	"os"
	"path/filepath"
)

type Path string

func NewPath(s string) (Path, error) {
	abs, err := filepath.Abs(s)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(abs); err != nil {
		return "", errors.New("invalid path: " + err.Error())
	}

	return Path(abs), nil
}

func (p Path) String() string {
	return string(p)
}

func (p Path) Exists() bool {
	_, err := os.Stat(p.String())
	return err == nil
}

func (p Path) IsDir() bool {
	info, err := os.Stat(p.String())
	return err == nil && info.IsDir()
}
