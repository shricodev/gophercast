package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shricodev/gophercast/pkg/types"
)

// EnsureDir creates directory if it doesn't exist.
func EnsureDir(path types.Path) error {
	return os.MkdirAll(path.String(), 0755)
}

// SanitizeFilename cleans up filenames for safe storage.
func SanitizeFilename(filename string) string {
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	filename = strings.ToLower(filename)

	filename = strings.ReplaceAll(filename, " ", "_")

	reg := regexp.MustCompile(`[^a-zA-Z0-9\-_.]`)
	filename = reg.ReplaceAllString(filename, "")

	reg = regexp.MustCompile(`_+`)
	filename = reg.ReplaceAllString(filename, "_")

	filename = strings.Trim(filename, "_")

	return filename
}

// UnsanitizeFilename reverses SanitizeFilename.
func UnsanitizeFilename(filename string) string {
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	filename = strings.ReplaceAll(filename, "_", " ")
	filename = strings.ReplaceAll(filename, "-", " ")

	filename = strings.ToTitle(filename)

	return filename
}

// CheckYtDlp verifies yt-dlp is installed.
func CheckYtDlp() error {
	_, err := exec.LookPath("yt-dlp")
	if err != nil {
		return fmt.Errorf("yt-dlp not found: %w\nPlease install with: pip install yt-dlp", err)
	}
	return nil
}
