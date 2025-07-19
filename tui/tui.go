package tui

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/shricodev/gophercast/internal/downloader"
	"github.com/shricodev/gophercast/pkg/types"
)

const (
	directory       = "Directory"
	youtube         = "YouTube"
	youtubePlaylist = "YouTube Playlist"

	chooseTracksManually = "Manually"
	chooseTracksAuto     = "Auto"
)

type screen int

const (
	screenMenu screen = iota
	screenPickDir
	screenInputYoutube
	screenInputYoutubePlaylist
	screenDownloadStarting
	screenDownloadComplete
	screenChooseTracks
	screenStreamTracks
)

var (
	docStyle  = lipgloss.NewStyle().Margin(1, 1)
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(1, 0)
)

// downloadYouTubeVideo downloads a YouTube video and returns a message.
func downloadYouTubeVideo(url string, d *downloader.Downloader) tea.Cmd {
	return func() tea.Msg {
		progressChan := make(chan downloader.DownloadProgress, 1)
		d.SetProgressCallback(func(progress downloader.DownloadProgress) {
			select {
			case progressChan <- progress:
			default:
			}
		})

		resultChan := make(chan downloadVideoResult, 1)
		go func() {
			track, err := d.DownloadVideo(url)
			resultChan <- downloadVideoResult{
				track: track,
				err:   err,
			}
		}()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for {
			select {
			case progress := <-progressChan:
				fmt.Println(progress)
			case result := <-resultChan:
				if result.err != nil {
					return errMsg{err: result.err}
				}
				return downloadCompleteMsg{
					tracks: types.Playlist{result.track},
					path:   result.track.Path,
				}
			case <-ctx.Done():
				return errMsg{ctx.Err()}
			}
		}
	}
}

// downloadYouTubePlaylist downloads a YouTube playlist and returns a message.
func downloadYouTubePlaylist(url string, d *downloader.Downloader) tea.Cmd {
	return func() tea.Msg {
		progressChan := make(chan downloader.DownloadProgress, 1)
		d.SetProgressCallback(func(progress downloader.DownloadProgress) {
			select {
			case progressChan <- progress:
			default:
			}
		})

		resultChan := make(chan downloadPlaylistResult, 1)
		go func() {
			tracks, err := d.DownloadPlaylist(url)
			resultChan <- downloadPlaylistResult{
				tracks: tracks,
				err:    err,
			}
		}()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for {
			select {
			case progress := <-progressChan:
				_ = progress
			case result := <-resultChan:
				if result.err != nil {
					return errMsg{result.err}
				}

				var dirPath types.Path
				var tracks types.Playlist
				if result.tracks != nil {
					tracks = *result.tracks
					if tracks.Len() > 0 {
						audioPath := tracks[0].Path
						dirPath = types.Path(filepath.Dir(audioPath.String()))
					}
				}

				return downloadCompleteMsg{
					tracks: tracks,
					path:   dirPath,
				}
			case <-ctx.Done():
				return errMsg{ctx.Err()}
			}
		}
	}
}

// Start starts the Bubble Tea TUI.
func Start() (types.Path, string, string) {
	m := InitialModel()
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}

	mo := finalModel.(model)
	if mo.downloader != nil {
		mo.downloader.Shutdown()
	}

	return mo.dirToMp3, mo.youtubePlaylistURL, mo.youtubeURL
}

type downloadVideoResult struct {
	track types.Track
	err   error
}

type downloadPlaylistResult struct {
	tracks *types.Playlist
	err    error
}
