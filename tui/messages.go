package tui

import tea "github.com/charmbracelet/bubbletea"

func (m *model) handleCustomMessages(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case downloadProgressMsg:
		return m.handleDownloadProgress(msg)
	case downloadCompleteMsg:
		return m.handleDownloadComplete(msg)
	case shutdownCompleteMsg:
		return m.handleShutdownComplete()
	case errMsg:
		return m.handleError(msg)
	}

	return m, nil
}

func (m *model) handleDownloadProgress(msg downloadProgressMsg) (tea.Model, tea.Cmd) {
	m.downloadProgress = msg.progress
	if m.downloadProgress.Total > 0 {
		percentage := float64(m.downloadProgress.Completed) / float64(m.downloadProgress.Total)
		return m, m.progress.SetPercent(percentage)
	}
	return m, nil
}

func (m *model) handleDownloadComplete(msg downloadCompleteMsg) (tea.Model, tea.Cmd) {
	m.downloadedTracks = msg.tracks

	// When the download completes, by default all tracks
	// are selected.
	m.selectedTracks = m.downloadedTracks

	if m.youtubeURL != "" {
		m.youtubeDownloadPath = msg.path
		m.state = screenStreamTracks
	} else if m.youtubePlaylistURL != "" {
		m.youtubePlaylistDownloadPath = msg.path
		m.state = screenChooseTracksOptions
	} else if m.dirToMp3Path != "" {
		m.dirToMp3Path = msg.path
		m.state = screenChooseTracksOptions
	}

	m.showDownloadProgress = false

	if m.downloader != nil {
		m.downloader.Shutdown()
		m.downloader = nil
	}
	return m, nil
}

func (m *model) handleShutdownComplete() (tea.Model, tea.Cmd) {
	return m, tea.Quit
}

func (m *model) handleError(msg errMsg) (tea.Model, tea.Cmd) {
	m.err = msg
	if m.downloader != nil {
		m.downloader.Shutdown()
		m.downloader = nil
	}
	return m, tea.Quit
}
