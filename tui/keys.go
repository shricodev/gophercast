package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"

	"github.com/shricodev/gophercast/pkg/types"
)

const keyD = "d"

func (m *model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m.handleCtrlCKey()

	case tea.KeyEnter:
		return m.handleEnterKey()

	case tea.KeyEsc:
		return m.handleEscKey()

	case tea.KeySpace:
		return m.handleSpaceKey()

	case tea.KeyRunes:
		switch msg.String() {
		case keyD:
			return m.handleDKey()
		}
	}

	return m, nil
}

func (m *model) handleCtrlCKey() (tea.Model, tea.Cmd) {
	if m.state == screenDownloadStarting && m.downloader != nil {
		if !m.isShuttingDown {
			m.isShuttingDown = true
			m.shutdownMessage = "Gracefully shutting down... Please wait for the download to finish."
			return m, shutdown()
		}

		return m, nil
	}

	if m.state == screenLobby || m.state == screenStreamTracks {
		if m.audioServer != nil {
			m.audioServer.Stop()
			m.audioServer = nil
		}
		return m, tea.Quit
	}

	return m, tea.Quit
}

func (m *model) handleEnterKey() (tea.Model, tea.Cmd) {
	switch m.state {
	case screenMenu:
		return m.handleMenuEnter()

	case screenChooseTracksOptions:
		return m.handleChooseTracksOptionsEnter()

	case screenPickDir:
		return m.handlePickDirEnter()

	case screenInputYoutube:
		return m.handleYouTubeEnter()

	case screenInputYoutubePlaylist:
		return m.handleYouTubePlaylistEnter()

	case screenChooseTracks:
		return m.handleChooseTracksEnter()

	case screenLobby:
		return m.handleLobbyEnter()
	}
	return m, nil
}

func (m *model) handleMenuEnter() (tea.Model, tea.Cmd) {
	selectedItem := m.initialScreenList.SelectedItem().(item)
	switch selectedItem.title {
	case singleFile:
		m.pickMode = "file"
		m.filePicker.FileAllowed = true
		m.filePicker.DirAllowed = false
		m.filePicker.AllowedTypes = []string{".mp3"}
		m.state = screenPickDir
		return m, m.filePicker.Init()

	case directory:
		m.pickMode = "dir"
		m.filePicker.DirAllowed = true
		m.filePicker.FileAllowed = false
		m.filePicker.AllowedTypes = nil
		m.state = screenPickDir
		return m, m.filePicker.Init()

	case youtube:
		m.state = screenInputYoutube
		m.textInput.Placeholder = "Enter YouTube URL..."
		m.textInput.Focus()
		return m, textinput.Blink

	case youtubePlaylist:
		m.state = screenInputYoutubePlaylist
		m.textInput.Placeholder = "Enter YouTube playlist URL..."
		m.textInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m *model) handleChooseTracksOptionsEnter() (tea.Model, tea.Cmd) {
	selectedItem := m.chooseTracksOptionsList.SelectedItem().(item)
	switch selectedItem.title {
	case chooseTracksAuto:
		m.state = screenLobby
		return m, tea.Batch(m.spinner.Tick, m.startAudioServer())

	case chooseTracksManually:
		m.selectedTracks = &types.Playlist{}
		m.selectedTracksList = newSelectedTracksList(m.downloadedTracks)

		m.state = screenChooseTracks
		return m, nil
	}
	return m, nil
}

func (m *model) handlePickDirEnter() (tea.Model, tea.Cmd) {
	// single file selection happens in updateBubbles via DidSelectFile
	if m.pickMode == "file" {
		return m, nil
	}

	selectedPath := m.filePicker.Path
	if selectedPath == "" {
		return m, nil
	}

	info, err := os.Stat(selectedPath)
	if err != nil {
		return m, nil
	}

	if info.IsDir() {
		m.dirToMp3Path = types.Path(selectedPath)
		m.state = screenDownloadStarting
		return m, tea.Batch(
			m.spinner.Tick,
			buildPlaylistFromDir(m.dirToMp3Path, m.downloadEvents),
			listenForDownloadEvents(m.downloadEvents),
		)
	}

	return m, nil
}

// handleFileSelected builds a one-track playlist and jumps straight to the lobby.
func (m *model) handleFileSelected(path string) tea.Cmd {
	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	track := types.Track{
		Title:  title,
		Path:   types.Path(path),
		Source: types.SourceLocalFile,
	}
	playlist := types.Playlist{track}
	m.downloadedTracks = &playlist
	m.selectedTracks = &playlist
	m.state = screenLobby
	return tea.Batch(m.spinner.Tick, m.startAudioServer())
}

func (m *model) handleYouTubeEnter() (tea.Model, tea.Cmd) {
	if m.textInput.Value() != "" {
		m.youtubeURL = m.textInput.Value()
		m.state = screenDownloadStarting

		return m, tea.Batch(
			m.spinner.Tick,
			downloadYouTubeVideo(m.youtubeURL, m.downloader, m.downloadEvents),
			listenForDownloadEvents(m.downloadEvents),
		)
	}
	return m, nil
}

func (m *model) handleYouTubePlaylistEnter() (tea.Model, tea.Cmd) {
	if m.textInput.Value() != "" {
		m.youtubePlaylistURL = m.textInput.Value()
		m.showDownloadProgress = true
		m.state = screenDownloadStarting
		return m, tea.Batch(
			m.spinner.Tick,
			downloadYouTubePlaylist(m.youtubePlaylistURL, m.downloader, m.downloadEvents),
			listenForDownloadEvents(m.downloadEvents),
		)
	}
	return m, nil
}

func (m *model) handleChooseTracksEnter() (tea.Model, tea.Cmd) {
	selectedTracks := types.Playlist{}
	for i, itm := range m.selectedTracksList.Items() {
		listItem, ok := itm.(item)
		if !ok || !listItem.selected {
			continue
		}

		selectedTracks = append(selectedTracks, (*m.downloadedTracks)[i])
	}

	m.selectedTracks = &selectedTracks
	m.state = screenLobby
	return m, tea.Batch(m.spinner.Tick, m.startAudioServer())
}

func (m *model) handleLobbyEnter() (tea.Model, tea.Cmd) {
	if m.audioServer != nil && len(m.connectedClients) > 0 {
		return m, m.startPlayback()
	}
	return m, nil
}

func (m *model) handleSpaceKey() (tea.Model, tea.Cmd) {
	if m.state == screenChooseTracks {
		if itm, ok := m.selectedTracksList.SelectedItem().(item); ok {
			newItem := item{
				title:    itm.title,
				desc:     itm.desc,
				selected: !itm.selected,
			}
			index := m.selectedTracksList.Index()
			m.selectedTracksList.SetItem(index, newItem)
		}
	}
	return m, nil
}

func (m *model) handleDKey() (tea.Model, tea.Cmd) {
	if m.state == screenPickDir && m.pickMode != "file" {
		currentDir := m.filePicker.CurrentDirectory
		if currentDir != "" {
			m.dirToMp3Path = types.Path(currentDir)
			m.state = screenDownloadStarting
			return m, tea.Batch(
				m.spinner.Tick,
				buildPlaylistFromDir(m.dirToMp3Path, m.downloadEvents),
				listenForDownloadEvents(m.downloadEvents),
			)
		}
	}

	return m, nil
}

func (m *model) handleEscKey() (tea.Model, tea.Cmd) {
	switch m.state {
	case screenPickDir, screenInputYoutube, screenInputYoutubePlaylist, screenChooseTracksOptions:
		m.state = screenMenu
		return m, nil
	case screenChooseTracks:
		m.state = screenChooseTracksOptions
		return m, nil
	case screenLobby:
		if m.audioServer != nil {
			m.audioServer.Stop()
			m.audioServer = nil
		}
		m.connectedClients = nil
		m.state = screenChooseTracksOptions
		return m, nil
	}
	return m, nil
}
