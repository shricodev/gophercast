// Package tui provides the terminal user interface for Gophercast.
// It is responsible for rendering the UI and handling user input.
package tui

import (
	"os"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/shricodev/gophercast/internal/downloader"
	"github.com/shricodev/gophercast/internal/logger"
	"github.com/shricodev/gophercast/pkg/types"
)

// model represents the state of the TUI.
type model struct {
	state screen

	list       list.Model
	spinner    spinner.Model
	progress   progress.Model
	filePicker filepicker.Model
	textInput  textinput.Model

	dirToMp3           types.Path
	youtubeURL         string
	youtubePlaylistURL string

	downloadedTracks types.Playlist
	downloader       *downloader.Downloader
	downloadConfig   *downloader.DownloadConfig
	downloadProgress *downloader.DownloadProgress

	isShuttingDown       bool
	shutdownMessage      string
	showDownloadProgress bool

	logger *logger.Logger

	err error
}

// Init initializes the TUI model.
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model accordingly.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		switch key {
		case tea.KeyCtrlC.String():
			if m.state == screenAppStarting && m.downloader != nil {
				if !m.isShuttingDown {
					m.isShuttingDown = true
					m.shutdownMessage = "Gracefully shutting down... Please wait for the download to finish."
					return m, shutdown(m.downloader)
				}

				return m, nil
			}

			return m, tea.Quit

		case tea.KeyEnter.String():
			switch m.state {
			case screenMenu:
				selectedItem := m.list.SelectedItem().(item)
				switch selectedItem.title {
				case directory:
					m.state = screenPickDir
					return m, m.filePicker.Init()
				case youtube:
					m.state = screenInputYoutube
					m.textInput.Placeholder = "Enter YouTube URL..."
					m.textInput.Focus()
					return m, textinput.Blink
				case youtubePlaylist:
					m.state = screenInputPlaylist
					m.textInput.Placeholder = "Enter YouTube playlist URL..."
					m.textInput.Focus()
					return m, textinput.Blink
				}
			case screenPickDir:
				selectedPath := m.filePicker.Path
				if selectedPath != "" {
					if info, err := os.Stat(selectedPath); err == nil && info.IsDir() {
						m.dirToMp3 = types.Path(selectedPath)
						m.state = screenAppStarting
						return m, tea.Batch(m.spinner.Tick, run)
					}
				}
			case screenInputYoutube:
				if m.textInput.Value() != "" {
					m.youtubeURL = m.textInput.Value()
					m.state = screenAppStarting

					config := downloader.DefaultConfig()
					downloader := downloader.NewDownloader(config)

					return m, tea.Batch(m.spinner.Tick, downloadYouTubeVideo(m.youtubeURL, downloader))
				}
			case screenInputPlaylist:
				if m.textInput.Value() != "" {
					m.youtubePlaylistURL = m.textInput.Value()
					m.state = screenAppStarting
					m.showDownloadProgress = true

					config := downloader.DefaultConfig()
					downloader := downloader.NewDownloader(config)

					return m, tea.Batch(m.spinner.Tick, downloadYouTubePlaylist(m.youtubePlaylistURL, downloader))
				}
			}

		case "d":
			if m.state == screenPickDir {
				currentDir := m.filePicker.CurrentDirectory
				if currentDir != "" {
					m.dirToMp3 = types.Path(currentDir)
					m.state = screenAppStarting
					return m, tea.Batch(m.spinner.Tick, run)
				}
			}

		case tea.KeyEsc.String():
			switch m.state {
			case screenPickDir, screenInputYoutube, screenInputPlaylist:
				m.state = screenMenu
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		hor, ver := docStyle.GetFrameSize()

		// Removing -30 after removing the vertical margin seems to work well
		// with the UI. This is a completely arbitrary value.
		m.list.SetSize(msg.Width-hor, msg.Height-ver-30)

		// Here again, arbitrary values. Subtracting 40 from the height makes
		// it look nice in the UI.
		// FIX: When the window is resize, or the font is changed, the height
		// of the picker grows super huge (might be an issue with bubble
		// itself, or I'm not correct).
		m.filePicker.SetHeight(msg.Height - 40)

		m.progress.Width = msg.Width - hor - 40

	case downloadProgressMsg:
		m.downloadProgress = msg.progress
		if m.downloadProgress.Total > 0 {
			percentage := float64(m.downloadProgress.Completed) / float64(m.downloadProgress.Total)
			return m, m.progress.SetPercent(percentage)
		}

	case downloadCompleteMsg:
		m.downloadedTracks = msg.tracks
		m.dirToMp3 = msg.path
		m.state = screenAppRunning
		m.showDownloadProgress = false

		if m.downloader != nil {
			m.downloader.Shutdown()
			m.downloader = nil
		}
		return m, nil

	case appStartedMsg:
		m.state = screenAppRunning
		return m, nil

	case shutdownCompleteMsg:
		return m, tea.Quit

	case errMsg:
		m.err = msg
		if m.downloader != nil {
			m.downloader.Shutdown()
			m.downloader = nil
		}
		return m, tea.Quit
	}

	// Update bubble components
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch m.state {
	case screenMenu:
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	case screenPickDir:
		m.filePicker, cmd = m.filePicker.Update(msg)
		cmds = append(cmds, cmd)
	case screenInputYoutube, screenInputPlaylist:
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	case screenAppStarting:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// InitialModel returns the initial model for the TUI.
func InitialModel() model {
	items := []list.Item{
		item{title: directory, desc: "Pick a directory with mp3 files"},
		item{title: youtube, desc: "Provide a YouTube video URL"},
		item{title: youtubePlaylist, desc: "Provide a YouTube playlist URL"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select source"

	fp := filepicker.New()
	fp.DirAllowed = true
	fp.FileAllowed = false

	// fp.ShowHidden is set to true because by default we save the files in the
	// ~/.gophercast/downloads directory.
	fp.ShowHidden = true

	homeDir, err := os.UserHomeDir()
	if err == nil {
		fp.CurrentDirectory = homeDir
	}

	ti := textinput.New()
	ti.CharLimit = 500
	ti.Width = 50

	s := spinner.New()
	s.Spinner = spinner.Globe
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	p := progress.New(progress.WithDefaultScaledGradient())

	downloadConfig := downloader.DefaultConfig()
	d := downloader.NewDownloader(downloadConfig)

	logger, err := logger.New(logger.Config{
		Level:      logger.LevelInfo,
		Output:     "stdout",
		TimeFormat: "15:04:05",
	})
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	return model{
		state: screenMenu,

		list:       l,
		spinner:    s,
		progress:   p,
		filePicker: fp,
		textInput:  ti,

		downloadConfig:   downloadConfig,
		downloader:       d,
		downloadProgress: &downloader.DownloadProgress{},

		logger: logger,
	}
}

// shutdown gracefully shuts down the downloader.
func shutdown(d *downloader.Downloader) tea.Cmd {
	return func() tea.Msg {
		d.Shutdown()
		return shutdownCompleteMsg{}
	}
}

// item is an item in the list that holds the title and the description. This
// is what's shown in the initial page when the application runs with the serve
// subcommand
type item struct {
	title, desc string
}

// Title satisfies the list.Item interface
func (i item) Title() string { return i.title }

// Description satisfies the list.Item interface
func (i item) Description() string { return i.desc }

// FilterValue satisfies the list.Item interface
func (i item) FilterValue() string { return i.title }

type (
	appStartedMsg       struct{}
	shutdownCompleteMsg struct{}
	downloadProgressMsg struct {
		progress *downloader.DownloadProgress
	}
	downloadCompleteMsg struct {
		tracks types.Playlist
		path   types.Path
	}
)

// errMsg represents an error message.
type errMsg struct{ err error }

// Implements the error interface.
func (e errMsg) Error() string {
	return e.err.Error()
}
