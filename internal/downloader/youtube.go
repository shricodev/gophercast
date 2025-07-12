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
	"sync"

	"github.com/shricodev/gophercast/pkg/types"
)

type DownloadConfig struct {
	OutputDir types.Path
	Quality   string
	Format    string
	Workers   int
}

func DefaultConfig() *DownloadConfig {
	homeDir, _ := os.UserHomeDir()
	return &DownloadConfig{
		OutputDir: types.Path(filepath.Join(homeDir, ".gophercast", "downloads")),
		Quality:   "192",
		Format:    "mp3",
		Workers:   5,
	}
}

type PlaylistVideo struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	ID    string `json:"id"`
}

type PlaylistInfo struct {
	Title   string          `json:"title"`
	Entries []PlaylistVideo `json:"entries"`
}

type DownloadProgress struct {
	Completed int
	Total     int
	Current   string
	Failed    []string
}

type ProgressCallback func(progress DownloadProgress)

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

	outputPath := filepath.Join(config.OutputDir.String(), sanitizedTitle+"."+config.Format)

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

	playlistInfo, err := getPlaylistInfo(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist info: %w", err)
	}

	if len(playlistInfo.Entries) == 0 {
		return nil, fmt.Errorf("playlist is empty or could not extract video URLs")
	}

	sanitizedPlaylistTitle := sanitizeFilename(playlistInfo.Title)
	playlistDir := types.Path(filepath.Join(config.OutputDir.String(), sanitizedPlaylistTitle))

	if err = ensureDir(types.Path(playlistDir)); err != nil {
		return nil, fmt.Errorf("failed to create playlist directory: %w", err)
	}

	tracks, err := downloadVideosConcurrently(playlistInfo.Entries, playlistDir, config)
	if err != nil {
		return nil, fmt.Errorf("failed to download playlist: %w", err)
	}

	return tracks, nil
}

func DownloadPlaylistWithProgress(url string, config *DownloadConfig, progressCallback ProgressCallback) (*types.Playlist, error) {
	if err := checkYtDlp(); err != nil {
		return nil, err
	}

	playlistInfo, err := getPlaylistInfo(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist info: %w", err)
	}

	if len(playlistInfo.Entries) == 0 {
		return nil, fmt.Errorf("playlist is empty or could not extract video URLs")
	}

	sanitizedPlaylistTitle := sanitizeFilename(playlistInfo.Title)
	playlistDir := types.Path(filepath.Join(config.OutputDir.String(), sanitizedPlaylistTitle))

	if err = ensureDir(types.Path(playlistDir)); err != nil {
		return nil, fmt.Errorf("failed to create playlist directory: %w", err)
	}

	tracks, err := downloadVideosConcurrentlyWithProgress(playlistInfo.Entries, playlistDir, config, progressCallback)
	if err != nil {
		return nil, fmt.Errorf("failed to download playlist: %w", err)
	}

	return tracks, nil
}

func downloadVideosConcurrently(videos []PlaylistVideo, outputDir types.Path, config *DownloadConfig) (*types.Playlist, error) {
	jobs := make(chan PlaylistVideo, len(videos))
	results := make(chan downloadResult, len(videos))

	var wg sync.WaitGroup
	for range config.Workers {
		wg.Add(1)
		go worker(&wg, jobs, results, outputDir, config)
	}

	for _, video := range videos {
		jobs <- video
	}

	close(jobs)

	wg.Wait()

	close(results)

	tracks := make(types.Playlist, 0, len(videos))
	errors := make([]string, 0, len(videos))

	for result := range results {
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("failed to download %s: %v", result.video.Title, result.err))
		} else {
			tracks = append(tracks, result.track)
		}
	}

	if len(tracks) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("failed to download playlist: %s", strings.Join(errors, "\n"))
	}

	return &tracks, nil
}

func downloadVideosConcurrentlyWithProgress(videos []PlaylistVideo, outputDir types.Path, config *DownloadConfig, progressCallback ProgressCallback) (*types.Playlist, error) {
	jobs := make(chan PlaylistVideo, len(videos))
	results := make(chan downloadResult, len(videos))

	progress := DownloadProgress{
		Total:  len(videos),
		Failed: make([]string, 0, len(videos)),
	}

	var wg sync.WaitGroup
	for range config.Workers {
		wg.Add(1)
		go workerWithProgress(&wg, jobs, results, outputDir, config, progressCallback)
	}

	for _, video := range videos {
		jobs <- video
	}

	close(jobs)

	wg.Wait()

	close(results)

	tracks := make(types.Playlist, 0, len(videos))
	errors := make([]string, 0, len(videos))

	for result := range results {
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("failed to download %s: %v", result.video.Title, result.err))
			progress.Failed = append(progress.Failed, result.video.Title)
		} else {
			tracks = append(tracks, result.track)
		}
		progress.Completed++
		if progressCallback != nil {
			progressCallback(progress)
		}
	}

	if len(tracks) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("failed to download playlist: %s", strings.Join(errors, "\n"))
	}

	return &tracks, nil
}

func downloadSingleVideo(video PlaylistVideo, outputDir types.Path, config *DownloadConfig) (types.Track, error) {
	videoURL := fmt.Sprintf("https://youtube.com/watch?v=%s", video.ID)

	sanitizedTitle := sanitizeFilename(video.Title)
	outputPath := filepath.Join(outputDir.String(), sanitizedTitle+"."+config.Format)

	cmd := exec.Command("yt-dlp",
		"--extract-audio",
		"--audio-format", config.Format,
		"--audio-quality", config.Quality,
		"-o", outputPath,
		videoURL)

	if err := cmd.Run(); err != nil {
		return types.Track{}, fmt.Errorf("failed to download video: %w", err)
	}

	track := types.Track{
		Title:  video.Title,
		Path:   types.Path(outputPath),
		Source: types.SourceYoutube,
	}

	return track, nil
}

func getPlaylistInfo(url string) (*PlaylistInfo, error) {
	cmd := exec.Command("yt-dlp",
		"--flat-playlist",
		"--no-warnings",
		"--dump-single-json",
		url,
	)

	jsonOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist info: %w", err)
	}

	var info PlaylistInfo
	if err := json.Unmarshal(jsonOutput, &info); err != nil {
		return nil, fmt.Errorf("failed to parse playlist JSON from yt-dlp: %w", err)
	}

	if info.Title == "" {
		return nil, fmt.Errorf("could not find playlist title in yt-dlp output")
	}

	return &info, nil
}

type downloadResult struct {
	track types.Track
	video PlaylistVideo
	err   error
}

func worker(wg *sync.WaitGroup, jobs <-chan PlaylistVideo, results chan<- downloadResult, outputDir types.Path, config *DownloadConfig) {
	defer wg.Done()

	for video := range jobs {
		track, err := downloadSingleVideo(video, outputDir, config)
		results <- downloadResult{
			track: track,
			video: video,
			err:   err,
		}
	}
}

func workerWithProgress(wg *sync.WaitGroup, jobs <-chan PlaylistVideo, results chan<- downloadResult, outputDir types.Path, config *DownloadConfig, progressCallback ProgressCallback) {
	defer wg.Done()

	for video := range jobs {
		if progressCallback != nil {
			progress := DownloadProgress{
				Current: video.Title,
			}
			progressCallback(progress)
		}

		track, err := downloadSingleVideo(video, outputDir, config)
		results <- downloadResult{
			track: track,
			video: video,
			err:   err,
		}
	}
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
func ensureDir(path types.Path) error {
	return os.MkdirAll(path.String(), 0755)
}

// checkYtDlp verifies yt-dlp is installed
func checkYtDlp() error {
	_, err := exec.LookPath("yt-dlp")
	if err != nil {
		return fmt.Errorf("yt-dlp not found: %v\nPlease install with: pip install yt-dlp", err)
	}
	return nil
}
