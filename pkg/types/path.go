package types

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Path represents a file path.
type Path string

func NewPath(s string) (Path, error) {
	abs, err := filepath.Abs(s)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	return Path(abs), nil
}

func NewPathInFS(fsys fs.FS, relativePath string, root Path) (Path, error) {
	if _, err := fs.Stat(fsys, relativePath); err != nil {
		return "", fmt.Errorf("path does not exist in the filesystem: %w", err)
	}

	fullPath := filepath.Join(root.String(), relativePath)
	abs, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
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
