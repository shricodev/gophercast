package tui

import (
	"log"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/shricodev/gophercast/internal/downloader"
	"github.com/shricodev/gophercast/internal/playlist"
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
	screenLobby
	screenStreamTracks
)

var (
	docStyle  = lipgloss.NewStyle().Margin(1, 1)
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(1, 0)
)

// downloadYouTubeVideo starts a YouTube video download using the channel pattern.
func downloadYouTubeVideo(url string, d *downloader.Downloader, events chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		d.SetProgressCallback(func(progress downloader.DownloadProgress) {
			select {
			case events <- downloadProgressMsg{progress: &progress}:
			default:
			}
		})

		go func() {
			track, err := d.DownloadVideo(url)
			if err != nil {
				events <- errMsg{err: err}
				return
			}
			events <- downloadCompleteMsg{
				tracks: &types.Playlist{track},
				path:   track.Path,
			}
		}()

		return nil
	}
}

// downloadYouTubePlaylist starts a YouTube playlist download using the channel pattern.
func downloadYouTubePlaylist(url string, d *downloader.Downloader, events chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		d.SetProgressCallback(func(progress downloader.DownloadProgress) {
			select {
			case events <- downloadProgressMsg{progress: &progress}:
			default:
			}
		})

		go func() {
			tracks, err := d.DownloadPlaylist(url)
			if err != nil {
				events <- errMsg{err: err}
				return
			}

			var dirPath types.Path
			var trackList types.Playlist
			if tracks != nil {
				trackList = *tracks
				if trackList.Len() > 0 {
					audioPath := trackList[0].Path
					dirPath = types.Path(filepath.Dir(audioPath.String()))
				}
			}

			events <- downloadCompleteMsg{
				tracks: &trackList,
				path:   dirPath,
			}
		}()

		return nil
	}
}

// buildPlaylistFromDir scans a directory for MP3 files in the background.
func buildPlaylistFromDir(dirPath types.Path, events chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		go func() {
			tracks, err := playlist.BuildPlaylistFromDir(dirPath)
			if err != nil {
				events <- errMsg{err: err}
				return
			}
			events <- downloadCompleteMsg{
				tracks: tracks,
				path:   dirPath,
			}
		}()

		return nil
	}
}

// listenForDownloadEvents blocks on the events channel and returns messages.
func listenForDownloadEvents(events chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-events
		if !ok {
			return nil
		}
		return msg
	}
}

// Start starts the Bubble Tea TUI.
func Start() (types.Path, string, string) {
	m := InitialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

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
