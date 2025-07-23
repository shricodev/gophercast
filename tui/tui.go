package tui

import (
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
	chooseTracks    = "Choose Tracks"

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
	screenChooseTracksOptions
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

		for {
			select {
			case progress := <-progressChan:
				_ = progress
				// log.Printf("Downloading: %s", progress.Current)
			case result := <-resultChan:
				if result.err != nil {
					return errMsg{err: result.err}
				}
				return downloadCompleteMsg{
					tracks: types.Playlist{result.track},
					path:   result.track.Path,
				}
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

		for {
			select {
			case progress := <-progressChan:
				_ = progress
			// log.Println(progress)

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

				// log.Printf("Download complete: %v", tracks)
				return downloadCompleteMsg{
					tracks: tracks,
					path:   dirPath,
				}
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

	mo := finalModel.(*model)
	if mo.downloader != nil {
		mo.downloader.Shutdown()
	}

	return mo.dirToMp3Path, mo.youtubePlaylistURL, mo.youtubeURL
}
