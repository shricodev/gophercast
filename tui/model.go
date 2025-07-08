package tui

import (
	"os"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

	err error
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		switch m.state {
		case screenMenu:
			if msg.String() == "enter" {
				selectedItem := m.list.SelectedItem().(item)
				switch selectedItem.title {
				case "Directory":
					m.state = screenPickDir
					return m, m.filePicker.Init()
				case "YouTube":
					m.state = screenInputYoutube
					m.textInput.Placeholder = "Enter YouTube URL..."
					m.textInput.Focus()
					return m, textinput.Blink
				case "Playlist":
					m.state = screenInputPlaylist
					m.textInput.Placeholder = "Enter YouTube playlist URL..."
					m.textInput.Focus()
					return m, textinput.Blink
				}
			}
		case screenPickDir:
			if msg.String() == "enter" {
				selectedPath := m.filePicker.Path
				if selectedPath != "" {
					if info, err := os.Stat(selectedPath); err == nil && info.IsDir() {
						m.dirPath = types.Path(selectedPath)
						m.state = screenAppStarting
						return m, tea.Batch(m.spinner.Tick, run)
					}
				}
			} else if msg.String() == "d" {
				currentDir := m.filePicker.CurrentDirectory
				if currentDir != "" {
					m.dirPath = types.Path(currentDir)
					m.state = screenAppStarting
					return m, tea.Batch(m.spinner.Tick, run)
				}
			}
		case screenInputYoutube:
			if msg.String() == "enter" {
				if m.textInput.Value() != "" {
					m.youtubeURL = m.textInput.Value()
					m.state = screenAppStarting
					return m, tea.Batch(m.spinner.Tick, run)
				}
			}
		case screenInputPlaylist:
			if msg.String() == "enter" {
				if m.textInput.Value() != "" {
					m.youtubePlaylistURL = m.textInput.Value()
					m.state = screenAppStarting
					return m, tea.Batch(m.spinner.Tick, run)
				}
			}
		}

		if msg.String() == "esc" && m.state != screenMenu {
			m.state = screenMenu
			return m, nil
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-10)
		// m.filePicker.SetHeight(10)

	case appStartedMsg:
		m.state = screenAppRunning
		return m, nil

	case errMsg:
		m.err = msg
		return m, tea.Quit
	}

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

func InitialModel() model {
	items := []list.Item{
		item{title: "Directory", desc: "Pick a directory with mp3 files"},
		item{title: "YouTube", desc: "Provide a YouTube video URL"},
		item{title: "Playlist", desc: "Provide a YouTube playlist URL"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select source"

	// Set up file picker with initial directory
	fp := filepicker.New()
	fp.DirAllowed = true
	fp.FileAllowed = false
	fp.ShowHidden = false
	fp.SetHeight(10)

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
		list:       l,
		state:      screenMenu,
		filePicker: fp,
		textInput:  ti,
		spinner:    s,
	}
}
