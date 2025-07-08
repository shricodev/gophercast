// Package tui launches the tui with BubbleTea
package tui

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mbndr/figlet4go"

	"github.com/shricodev/gophercast/pkg/types"
)

var (
	docStyle  = lipgloss.NewStyle().Margin(1, 2)
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(1, 0)
)

type screen int

const (
	screenMenu screen = iota
	screenPickDir
	screenInputYoutube
	screenInputPlaylist
	screenServerStarting
	screenServerRunning
)

type item struct {
	title, desc string
}

func (i item) Title() string { return i.title }

func (i item) Description() string { return i.desc }

func (i item) FilterValue() string { return i.title }

type serverStartedMsg struct{}

type errMsg struct{ err error }

// Implements the error interface.
func (e errMsg) Error() string {
	return e.err.Error()
}

type model struct {
	state   screen
	list    list.Model
	spinner spinner.Model

	filePicker filepicker.Model
	textInput  textinput.Model

	dirPath            types.Path
	youtubeURL         string
	youtubePlaylistURL string

	width  int
	height int
	err    error
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
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
						m.state = screenServerStarting
						return m, tea.Batch(m.spinner.Tick, startServer)
					}
				}
			} else if msg.String() == "d" {
				currentDir := m.filePicker.CurrentDirectory
				if currentDir != "" {
					m.dirPath = types.Path(currentDir)
					m.state = screenServerStarting
					return m, tea.Batch(m.spinner.Tick, startServer)
				}
			}
		case screenInputYoutube:
			if msg.String() == "enter" {
				if m.textInput.Value() != "" {
					m.youtubeURL = m.textInput.Value()
					m.state = screenServerStarting
					return m, tea.Batch(m.spinner.Tick, startServer)
				}
			}
		case screenInputPlaylist:
			if msg.String() == "enter" {
				if m.textInput.Value() != "" {
					m.youtubePlaylistURL = m.textInput.Value()
					m.state = screenServerStarting
					return m, tea.Batch(m.spinner.Tick, startServer)
				}
			}
		}

		if msg.String() == "esc" && m.state != screenMenu {
			m.state = screenMenu
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-10)
		m.filePicker.SetHeight(10)

	case serverStartedMsg:
		m.state = screenServerRunning
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
	case screenServerStarting:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.err != nil {
		return "Error: " + m.err.Error()
	}

	var s string

	options := figlet4go.NewRenderOptions()
	options.FontName = "standard"
	options.FontColor = []figlet4go.Color{figlet4go.ColorCyan}
	figlet := figlet4go.NewAsciiRender()
	rendered, _ := figlet.RenderOpts("Gophercast", options)
	s += rendered + "\n"

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("Built with 🤍 by @shricodev")
	s += footer + "\n\n"

	switch m.state {
	case screenMenu:
		s += m.list.View()
	case screenPickDir:
		s += "Choose a directory:\n\n" + m.filePicker.View() + "\n" + helpStyle.Render("↑/↓: Navigate, Enter: Select, d: Select current, Esc: Back")
	case screenInputYoutube:
		s += "Enter YouTube URL:\n\n" + m.textInput.View() + "\n\n" + helpStyle.Render("Press Enter to confirm, Esc to go back")
	case screenInputPlaylist:
		s += "Enter YouTube Playlist URL:\n\n" + m.textInput.View() + "\n\n" + helpStyle.Render("Press Enter to confirm, Esc to go back")
	case screenServerStarting:
		s += m.spinner.View() + " Starting work, please wait..."
	case screenServerRunning:
		s += "Server running!\n\nCtrl+C to quit."
	}

	return docStyle.Render(s)
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

	// Set initial directory to ~/Music if it exists, otherwise /home/<username>
	homeDir, err := os.UserHomeDir()
	if err == nil {
		musicDir := filepath.Join(homeDir, "Music")
		if _, err := os.Stat(musicDir); err == nil {
			fp.CurrentDirectory = musicDir
		} else {
			fp.CurrentDirectory = homeDir
		}
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

func startServer() tea.Msg {
	// perform the actual work
	time.Sleep(2 * time.Second)
	return serverStartedMsg{}
}

func Run() (types.Path, string, string) {
	m := InitialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}

	mo := finalModel.(model)
	return mo.dirPath, mo.youtubePlaylistURL, mo.youtubeURL
}
