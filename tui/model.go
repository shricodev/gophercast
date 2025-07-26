package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
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

	initialScreenList       list.Model
	chooseTracksOptionsList list.Model
	selectedTracksList      list.Model

	spinner    spinner.Model
	progress   progress.Model
	filePicker filepicker.Model
	textInput  textinput.Model

	dirToMp3Path       types.Path
	youtubeURL         string
	youtubePlaylistURL string

	youtubeDownloadPath         types.Path
	youtubePlaylistDownloadPath types.Path

	downloadedTracks *types.Playlist
	selectedTracks   *types.Playlist

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
func (m *model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model accordingly.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:

		// We check for 'esc' here because it's a special case. As soon as the
		// user presses 'esc', we need to return from this entire function, or
		// else the msg 'esc' is passed to the updateBubbles function below the
		// switch statement, which eventually closes the running bubble and the
		// entire application instead of moving the state to the set state.
		if msg.Type == tea.KeyEsc {
			return m.handleEscKey()
		}
		model, cmd := m.handleKeyMsg(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		if model != m {
			return model, tea.Batch(cmds...)
		}

	case tea.WindowSizeMsg:
		model, cmd := m.handleWindowSizeMsg(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		if model != m {
			return model, tea.Batch(cmds...)
		}

	default:
		model, cmd := m.handleCustomMessages(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		if model != m {
			return model, tea.Batch(cmds...)
		}
	}

	bubbleCmds := m.updateBubbles(msg)
	cmds = append(cmds, bubbleCmds...)

	return m, tea.Batch(cmds...)
}

// InitialModel returns the initial model for the TUI.
func InitialModel() *model {
	initialScreenChoices := []list.Item{
		item{title: directory, desc: "Pick a directory with mp3 files"},
		item{title: youtube, desc: "Provide a YouTube video URL"},
		item{title: youtubePlaylist, desc: "Provide a YouTube playlist URL"},
	}

	chooseTracksOptions := []list.Item{
		item{title: chooseTracksAuto, desc: "Select and run the mp3 files in random"},
		item{title: chooseTracksManually, desc: "Select the mp3 files manually"},
	}

	initialScreenChoicesList := list.New(initialScreenChoices, list.NewDefaultDelegate(), 0, 0)
	initialScreenChoicesList.Title = "Select source"

	chooseTracksOptionsList := list.New(chooseTracksOptions, list.NewDefaultDelegate(), 0, 0)
	chooseTracksOptionsList.Title = "Track selection method"

	selectedTracksList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)

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
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}

	return &model{
		state: screenMenu,

		initialScreenList:       initialScreenChoicesList,
		chooseTracksOptionsList: chooseTracksOptionsList,
		selectedTracksList:      selectedTracksList,

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

func newSelectedTracksList(tracks *types.Playlist) list.Model {
	items := make([]list.Item, len(*tracks))
	for i, track := range *tracks {
		items[i] = item{title: track.Title, desc: track.Path.String(), selected: false}
	}

	l := list.New(items, itemDelegate{}, 80, 20)

	l.Title = chooseTracks

	l.SetFilteringEnabled(true)
	l.SetShowStatusBar(true)

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys(tea.KeySpace.String()), key.WithHelp("Space", "toggle select")),
			key.NewBinding(key.WithKeys(tea.KeyEnter.String()), key.WithHelp("Enter", "confirm selection")),
		}
	}

	// fmt.Printf("DEBUG: Created list with title: '%s' and %d items\n", l.Title, len(l.Items()))
	return l
}

// item is an item in the list that holds the title and the description. This
// is what's shown in the initial page when the application runs with the serve
// subcommand
type item struct {
	title, desc string
	selected    bool
}

// Title satisfies the list.Item interface
func (i item) Title() string { return i.title }

// Description satisfies the list.Item interface
func (i item) Description() string { return i.desc }

// FilterValue satisfies the list.Item interface
func (i item) FilterValue() string { return i.title }

type (
	shutdownCompleteMsg struct{}
	downloadProgressMsg struct {
		progress *downloader.DownloadProgress
	}
	downloadCompleteMsg struct {
		tracks types.Playlist
		path   types.Path
	}
	downloadVideoResult struct {
		track types.Track
		err   error
	}
	downloadPlaylistResult struct {
		tracks *types.Playlist
		err    error
	}
)

// errMsg represents an error message.
type errMsg struct{ err error }

// Implements the error interface.
func (e errMsg) Error() string {
	return e.err.Error()
}
