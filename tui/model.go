package tui

import (
	"os"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/shricodev/gophercast/internal/downloader"
	"github.com/shricodev/gophercast/pkg/types"
)

type model struct {
	state   screen
	list    list.Model
	spinner spinner.Model

	filePicker filepicker.Model
	textInput  textinput.Model

	dirPath            types.Path
	youtubeURL         string
	youtubePlaylistURL string
	downloadedTracks   types.Playlist
	downloadConfig     *downloader.DownloadConfig

	err error
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		switch key {
		case "ctrl+c":
			return m, tea.Quit

		case "enter":
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
						m.dirPath = types.Path(selectedPath)
						m.state = screenAppStarting
						return m, tea.Batch(m.spinner.Tick, run)
					}
				}
			case screenInputYoutube:
				if m.textInput.Value() != "" {
					m.youtubeURL = m.textInput.Value()
					m.state = screenAppStarting
					return m, tea.Batch(m.spinner.Tick, downloadYouTubeVideo(m.youtubeURL, m.downloadConfig))
				}
			case screenInputPlaylist:
				if m.textInput.Value() != "" {
					m.youtubePlaylistURL = m.textInput.Value()
					m.state = screenAppStarting
					return m, tea.Batch(m.spinner.Tick, downloadYouTubePlaylist(m.youtubePlaylistURL, m.downloadConfig))
				}
			}

		case "d":
			if m.state == screenPickDir {
				currentDir := m.filePicker.CurrentDirectory
				if currentDir != "" {
					m.dirPath = types.Path(currentDir)
					m.state = screenAppStarting
					return m, tea.Batch(m.spinner.Tick, run)
				}
			}

		case "esc":
			if m.state != screenMenu && m.state != screenAppRunning {
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
		m.filePicker.SetHeight(msg.Height - 40)

	case appStartedMsg:
		m.state = screenAppRunning
		return m, nil

	case downloadCompleteMsg:
		m.downloadedTracks = msg.tracks
		m.dirPath = msg.path
		m.state = screenAppRunning
		return m, nil

	case errMsg:
		m.err = msg
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

// InitialModel returns the initial model for the application.
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
	fp.ShowHidden = false

	homeDir, err := os.UserHomeDir()
	if err == nil {
		fp.CurrentDirectory = homeDir
	}

	ti := textinput.New()
	ti.CharLimit = 500
	ti.Width = 50

	s := spinner.New()
	s.Spinner = spinner.Monkey
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		list:           l,
		state:          screenMenu,
		filePicker:     fp,
		textInput:      ti,
		spinner:        s,
		downloadConfig: downloader.DefaultConfig(),
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
