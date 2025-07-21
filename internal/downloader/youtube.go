package downloader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shricodev/gophercast/internal/utils"
	"github.com/shricodev/gophercast/pkg/types"
)

var (
	errDownloaderShutdown error = errors.New("downloader is shutdown")
	errDownloadCancelled  error = errors.New("download cancelled")
	errOperationCancelled error = errors.New("operation cancelled")
)

// DownloadConfig holds the configuration for downloading videos.
type DownloadConfig struct {
	OutputDir types.Path
	Quality   string
	Format    string
	Workers   int
}

// DefaultConfig returns a default download configuration.
func DefaultConfig() *DownloadConfig {
	homeDir, _ := os.UserHomeDir()
	return &DownloadConfig{
		OutputDir: types.Path(filepath.Join(homeDir, ".gophercast", "downloads")),
		Quality:   "192",
		Format:    "mp3",
		Workers:   5,
	}
}

// Downloader is a YouTube downloader.
type Downloader struct {
	config     *DownloadConfig
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	isShutdown bool

	progressCallback ProgressCallback
	currentProgress  DownloadProgress
}

// NewDownloader creates a new Downloader instance.
func NewDownloader(config *DownloadConfig) *Downloader {
	ctx, cancel := context.WithCancel(context.Background())
	return &Downloader{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

// DownloadVideo downloads a single YouTube video.
func (d *Downloader) DownloadVideo(url string) (types.Track, error) {
	if d.IsShutdown() {
		return types.Track{}, errDownloaderShutdown
	}

	if err := utils.CheckYtDlp(); err != nil {
		return types.Track{}, err
	}

	if err := utils.EnsureDir(d.config.OutputDir); err != nil {
		return types.Track{}, fmt.Errorf("failed to create output directory: %w", err)
	}

	type ytDlpInfo struct {
		Title    string `json:"title"`
		Duration int    `json:"duration"`
	}

	var info ytDlpInfo
	infoCmd := exec.CommandContext(d.ctx, "yt-dlp", "--no-warnings", "--dump-single-json", url)
	infoBytes, err := infoCmd.Output()
	if err != nil {
		if d.ctx.Err() == context.Canceled {
			return types.Track{}, errDownloadCancelled
		}
		return types.Track{}, fmt.Errorf("failed to get video info: %w", err)
	}

	if err := json.Unmarshal(infoBytes, &info); err != nil {
		return types.Track{}, fmt.Errorf("failed to parse JSON from yt-dlp: %w", err)
	}

	sanitizedTitle := utils.SanitizeFilename(info.Title)

	outputPath := filepath.Join(d.config.OutputDir.String(), sanitizedTitle+"."+d.config.Format)
	if _, err := os.Stat(outputPath); err == nil {
		return types.Track{
			Title:     info.Title,
			Path:      types.Path(outputPath),
			Source:    types.SourceYoutube,
			Duration:  time.Second * time.Duration(info.Duration),
			IsPlaying: false,
		}, nil
	}

	cmd := exec.CommandContext(d.ctx, "yt-dlp",
		"--extract-audio",
		"--audio-format", d.config.Format,
		"--audio-quality", d.config.Quality,
		"-o", outputPath,
		url)

	if err := cmd.Run(); err != nil {
		if d.ctx.Err() == context.Canceled {
			return types.Track{}, errDownloadCancelled
		}
		return types.Track{}, fmt.Errorf("failed to download video: %w", err)
	}

	track := types.Track{
		Title:     info.Title,
		Path:      types.Path(outputPath),
		Source:    types.SourceYoutube,
		Duration:  time.Second * time.Duration(info.Duration),
		IsPlaying: false,
	}

	return track, nil
}

// DownloadPlaylist downloads an entire YouTube playlist.
func (d *Downloader) DownloadPlaylist(url string) (*types.Playlist, error) {
	if d.IsShutdown() {
		return nil, errDownloaderShutdown
	}

	if err := utils.CheckYtDlp(); err != nil {
		return nil, err
	}

	playlistInfo, err := d.getPlaylistInfo(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist info: %w", err)
	}

	if len(playlistInfo.Entries) == 0 {
		return nil, fmt.Errorf("playlist is empty or could not extract video URLs")
	}

	sanitizedPlaylistTitle := utils.SanitizeFilename(playlistInfo.Title)
	playlistDir := types.Path(filepath.Join(d.config.OutputDir.String(), sanitizedPlaylistTitle))

	if err = utils.EnsureDir(types.Path(playlistDir)); err != nil {
		return nil, fmt.Errorf("failed to create playlist directory: %w", err)
	}

	d.mu.Lock()
	d.currentProgress = DownloadProgress{
		Total:  len(playlistInfo.Entries),
		Failed: make([]string, 0),
	}
	d.mu.Unlock()

	tracks, err := d.downloadVideosConcurrently(playlistInfo.Entries, playlistDir)
	if err != nil {
		return nil, fmt.Errorf("failed to download playlist: %w", err)
	}

	return tracks, nil
}

// SetProgressCallback sets the progress callback function.
func (d *Downloader) SetProgressCallback(progressCallback ProgressCallback) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.progressCallback = progressCallback
}

// Shutdown gracefully shuts down the downloader, waiting for all active downloads to complete.
func (d *Downloader) Shutdown() {
	d.mu.Lock()
	if d.isShutdown {
		d.mu.Unlock()
		return
	}

	d.isShutdown = true
	d.mu.Unlock()

	d.cancel()
	d.wg.Wait()
}

// IsShutdown returns true if the downloader is in a shutdown state.
func (d *Downloader) IsShutdown() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.isShutdown
}

// PlaylistVideo represents a single video in a YouTube playlist.
type PlaylistVideo struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	ID    string `json:"id"`
}

// PlaylistInfo represents information about a YouTube playlist.
type PlaylistInfo struct {
	Title   string          `json:"title"`
	Entries []PlaylistVideo `json:"entries"`
}

// DownloadProgress represents the progress of a download operation.
type DownloadProgress struct {
	Completed int
	Total     int
	Current   string
	Failed    []string
}

// ProgressCallback is a function type for reporting download progress.
type ProgressCallback func(progress DownloadProgress)

// downloadVideosConcurrently downloads multiple videos in parallel.
func (d *Downloader) downloadVideosConcurrently(
	videos []PlaylistVideo,
	outputDir types.Path,
) (*types.Playlist, error) {
	jobsCh := make(chan PlaylistVideo, len(videos))
	resultsCh := make(chan downloadResult, len(videos))
	startedCh := make(chan bool, len(videos))

	for range d.config.Workers {
		d.wg.Add(1)
		go d.worker(jobsCh, resultsCh, outputDir, startedCh)
	}

	go func() {
		defer close(jobsCh)
		for _, video := range videos {
			select {
			case <-d.ctx.Done():
				return
			case jobsCh <- video:
			}
		}
	}()

	var tracks types.Playlist
	var errors []string
	completed := 0
	started := 0

	for completed < len(videos) {
		select {
		case <-d.ctx.Done():
			// Only wait for downloads that have actually started
			expectedResults := started
			d.wg.Wait()
			close(resultsCh)

			for result := range resultsCh {
				completed++
				if result.err == nil {
					tracks = append(tracks, result.track)
				}
				if completed >= expectedResults {
					break
				}
			}

			return &tracks, fmt.Errorf(
				"download interrupted: %d/%d completed",
				len(tracks),
				len(videos),
			)

		case <-startedCh:
			started++

		case result := <-resultsCh:
			completed++
			d.mu.Lock()
			d.currentProgress.Completed = completed

			if result.err != nil {
				errorMsg := fmt.Sprintf("failed to download %s: %s", result.video.Title, result.err)
				errors = append(errors, errorMsg)
				d.currentProgress.Failed = append(d.currentProgress.Failed, errorMsg)
			} else {
				tracks = append(tracks, result.track)
			}

			if d.progressCallback != nil {
				d.progressCallback(d.currentProgress)
			}
			d.mu.Unlock()
		}
	}

	close(resultsCh)
	d.wg.Wait()

	if len(tracks) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("failed to download playlist: %s", strings.Join(errors, "\n"))
	}

	return &tracks, nil
}

// downloadSingleVideoFromYtPlaylist downloads a single video from a playlist.
func (d *Downloader) downloadSingleVideoFromYtPlaylist(
	video PlaylistVideo,
	outputDir types.Path,
) (types.Track, error) {
	if d.IsShutdown() {
		return types.Track{}, errDownloadCancelled
	}

	videoURL := fmt.Sprintf("https://youtube.com/watch?v=%s", video.ID)

	type ytDlpInfo struct {
		Duration int `json:"duration"`
	}

	var info ytDlpInfo
	infoCmd := exec.CommandContext(d.ctx, "yt-dlp", "--no-warnings", "--dump-single-json", videoURL)
	infoBytes, err := infoCmd.Output()
	if err != nil {
		if d.ctx.Err() == context.Canceled {
			return types.Track{}, errDownloadCancelled
		}
		return types.Track{}, fmt.Errorf("failed to get video info: %w", err)
	}

	if err := json.Unmarshal(infoBytes, &info); err != nil {
		return types.Track{}, fmt.Errorf("failed to parse video info: %w", err)
	}

	sanitizedTitle := utils.SanitizeFilename(video.Title)
	outputPath := filepath.Join(outputDir.String(), sanitizedTitle+"."+d.config.Format)
	if _, err := os.Stat(outputPath); err == nil {
		return types.Track{
			Title:     video.Title,
			Path:      types.Path(outputPath),
			Source:    types.SourceYoutubePlaylist,
			Duration:  time.Second * time.Duration(info.Duration),
			IsPlaying: false,
		}, nil
	}

	cmd := exec.CommandContext(d.ctx, "yt-dlp",
		"--extract-audio",
		"--audio-format", d.config.Format,
		"--audio-quality", d.config.Quality,
		"-o", outputPath,
		videoURL)

	if err := cmd.Run(); err != nil {
		if d.ctx.Err() == context.Canceled {
			return types.Track{}, errDownloadCancelled
		}
		return types.Track{}, fmt.Errorf("failed to download video: %w", err)
	}

	track := types.Track{
		Title:     video.Title,
		Path:      types.Path(outputPath),
		Source:    types.SourceYoutubePlaylist,
		Duration:  time.Second * time.Duration(info.Duration),
		IsPlaying: false,
	}

	return track, nil
}

// getPlaylistInfo fetches information about a YouTube playlist.
func (d *Downloader) getPlaylistInfo(url string) (*PlaylistInfo, error) {
	cmd := exec.CommandContext(d.ctx, "yt-dlp",
		"--flat-playlist",
		"--no-warnings",
		"--dump-single-json",
		url,
	)

	jsonOutput, err := cmd.Output()
	if err != nil {
		if d.ctx.Err() == context.Canceled {
			return nil, errOperationCancelled
		}
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

// downloadResult represents the result of a single video download.
type downloadResult struct {
	track types.Track
	video PlaylistVideo
	err   error
}

// worker is a concurrent worker for downloading videos.
func (d *Downloader) worker(
	jobs <-chan PlaylistVideo,
	results chan<- downloadResult,
	outputDir types.Path,
	started chan<- bool,
) {
	defer d.wg.Done()

	for {
		select {
		case video, ok := <-jobs:
			if !ok {
				return
			}

			// Signal that we've started this download
			select {
			case started <- true:
			case <-d.ctx.Done():
				return
			}

			d.mu.Lock()
			d.currentProgress.Current = video.Title
			if d.progressCallback != nil {
				d.progressCallback(d.currentProgress)
			}
			d.mu.Unlock()

			track, err := d.downloadSingleVideoFromYtPlaylist(video, outputDir)

			select {
			case results <- downloadResult{
				track: track,
				video: video,
				err:   err,
			}:
			case <-d.ctx.Done():
				return
			}

		case <-d.ctx.Done():
			return
		}
	}
}
