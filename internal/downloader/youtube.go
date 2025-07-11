// Package downloader handles YouTube downloads: videos and playlists
package downloader

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shricodev/gophercast/internal/playlist"
	"github.com/shricodev/gophercast/pkg/types"
)

type DownloadConfig struct {
	OutputDir string
	Quality   string
	Format    string
}

func DefaultConfig() *DownloadConfig {
	homeDir, _ := os.UserHomeDir()
	return &DownloadConfig{
		OutputDir: filepath.Join(homeDir, ".gophercast", "downloads"),
		Quality:   "192",
		Format:    "mp3",
	}
}

// DownloadVideo downloads a single YouTube video
func DownloadVideo(url string, config *DownloadConfig) (types.Track, error) {
	if err := checkYtDlp(); err != nil {
		return types.Track{}, err
	}

	if err := ensureDir(config.OutputDir); err != nil {
		return types.Track{}, fmt.Errorf("failed to create output directory: %w", err)
	}

	infoCmd := exec.Command("yt-dlp", "--get-title", url)
	titleBytes, err := infoCmd.Output()
	if err != nil {
		return types.Track{}, fmt.Errorf("failed to get video info: %w", err)
	}

	title := strings.TrimSpace(string(titleBytes))
	sanitizedTitle := sanitizeFilename(title)

	outputPath := filepath.Join(config.OutputDir, sanitizedTitle+"."+config.Format)

	cmd := exec.Command("yt-dlp",
		"--extract-audio",
		"--audio-format", config.Format,
		"--audio-quality", config.Quality,
		"-o", outputPath,
		url)

	if err := cmd.Run(); err != nil {
		return types.Track{}, fmt.Errorf("failed to download video: %w", err)
	}

	track := types.Track{
		Title:  title,
		Path:   types.Path(outputPath),
		Source: types.SourceYoutube,
	}

	return track, nil
}

func DownloadPlaylist(url string, config *DownloadConfig) (*types.Playlist, error) {
	if err := checkYtDlp(); err != nil {
		return nil, err
	}

	infoCmd := exec.Command("yt-dlp",
		"--flat-playlist",
		"--no-warnings",
		"--dump-single-json",
		url,
	)
	jsonOutput, err := infoCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist info: %w", err)
	}

	var info struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(jsonOutput, &info); err != nil {
		return nil, fmt.Errorf("failed to parse playlist JSON from yt-dlp: %w", err)
	}
	playlistTitle := info.Title
	if playlistTitle == "" {
		return nil, fmt.Errorf("could not find playlist title in yt-dlp output")
	}

	sanitizedPlaylistTitle := sanitizeFilename(playlistTitle)

	playlistDir := filepath.Join(config.OutputDir, sanitizedPlaylistTitle)
	if err := ensureDir(playlistDir); err != nil {
		return nil, fmt.Errorf("failed to create playlist directory: %w", err)
	}

	outputTemplate := filepath.Join(playlistDir, "%(title)s.%(ext)s")
	cmd := exec.Command("yt-dlp",
		"--extract-audio",
		"--audio-format", config.Format,
		"--audio-quality", config.Quality,
		"-o", outputTemplate,
		url)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to download playlist: %w", err)
	}

	return playlist.BuildPlaylistFromDir(types.Path(playlistDir))
}

// sanitizeFilename cleans up filenames for safe storage
func sanitizeFilename(filename string) string {
	filename = strings.ToLower(filename)

	filename = strings.ReplaceAll(filename, " ", "_")

	reg := regexp.MustCompile(`[^a-zA-Z0-9\-_.]`)
	filename = reg.ReplaceAllString(filename, "")

	reg = regexp.MustCompile(`_+`)
	filename = reg.ReplaceAllString(filename, "_")

	filename = strings.Trim(filename, "_")

	return filename
}

// ensureDir creates directory if it doesn't exist
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// checkYtDlp verifies yt-dlp is installed
func checkYtDlp() error {
	_, err := exec.LookPath("yt-dlp")
	if err != nil {
		return fmt.Errorf("yt-dlp not found: %v\nPlease install with: pip install yt-dlp", err)
	}
	return nil
}
