// Package tui launches the tui with BubbleTea
package tui

import (
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/shricodev/gophercast/internal/downloader"
	"github.com/shricodev/gophercast/pkg/types"
)

var (
	docStyle  = lipgloss.NewStyle().Margin(1, 1)
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(1, 0)
)

type appStartedMsg struct{}

type downloadCompleteMsg struct {
	tracks types.Playlist
	path   types.Path
}

type errMsg struct{ err error }

// Implements the error interface.
func (e errMsg) Error() string {
	return e.err.Error()
}

func run() tea.Msg {
	// perform the actual work
	time.Sleep(2 * time.Second)
	return appStartedMsg{}
}

func downloadYouTubeVideo(url string, config *downloader.DownloadConfig) tea.Cmd {
	return func() tea.Msg {
		track, err := downloader.DownloadVideo(url, config)
		if err != nil {
			return errMsg{err}
		}

		return downloadCompleteMsg{
			tracks: types.Playlist{track},
			path:   track.Path,
		}
	}
}

func downloadYouTubePlaylist(url string, config *downloader.DownloadConfig) tea.Cmd {
	return func() tea.Msg {
		tracks, err := downloader.DownloadPlaylist(url, config)
		if err != nil {
			return errMsg{err}
		}

		var dirPath types.Path
		if len(tracks) > 0 {
			dirPath = tracks[0].Path
		}

		return downloadCompleteMsg{
			tracks: tracks,
			path:   dirPath,
		}
	}
}

func Start() (types.Path, string, string) {
	m := InitialModel()
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}

	mo := finalModel.(model)
	return mo.dirPath, mo.youtubePlaylistURL, mo.youtubeURL
}
