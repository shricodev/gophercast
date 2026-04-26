package tui

import (
	"fmt"
	"net"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shricodev/gophercast/internal/downloader"
)

func (m *model) handleCustomMessages(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case downloadProgressMsg:
		return m.handleDownloadProgress(msg)
	case downloadCompleteMsg:
		return m.handleDownloadComplete(msg)
	case shutdownInitiatedMsg:
		return m.handleShutdownInitiated()
	case serverStartedMsg:
		return m.handleServerStarted(msg)
	case serverErrorMsg:
		return m.handleServerError(msg)
	case clientListUpdateMsg:
		return m.handleClientListUpdate(msg)
	case playbackStartedMsg:
		return m.handlePlaybackStarted()
	case playbackStoppedMsg:
		return m.handlePlaybackStopped(msg)
	case streamTickMsg:
		return m.handleStreamTick(msg)
	case errMsg:
		return m.handleError(msg)
	}

	return m, nil
}

func (m *model) handleDownloadProgress(msg downloadProgressMsg) (tea.Model, tea.Cmd) {
	m.downloadProgress = msg.progress
	cmds := []tea.Cmd{listenForDownloadEvents(m.downloadEvents)}
	if m.downloadProgress.Total > 0 {
		percentage := float64(m.downloadProgress.Completed) / float64(m.downloadProgress.Total)
		cmds = append(cmds, m.progress.SetPercent(percentage))
	}
	return m, tea.Batch(cmds...)
}

func (m *model) handleDownloadComplete(msg downloadCompleteMsg) (tea.Model, tea.Cmd) {
	m.downloadedTracks = msg.tracks

	// default to all tracks selected
	m.selectedTracks = m.downloadedTracks

	if m.youtubeURL != "" {
		m.youtubeDownloadPath = msg.path
		m.state = screenLobby
		return m, tea.Batch(m.spinner.Tick, m.startAudioServer())
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

func (m *model) handleShutdownInitiated() (tea.Model, tea.Cmd) {
	if m.downloader != nil {
		m.downloader.Shutdown()
		m.downloader = nil
	}

	if m.audioServer != nil {
		m.audioServer.Stop()
		m.audioServer = nil
	}

	if m.logger != nil {
		m.logger.Close()
	}

	return m, tea.Quit
}

func (m *model) handleError(msg errMsg) (tea.Model, tea.Cmd) {
	m.err = msg
	m.logger.Error("download error", "error", msg.err)

	// reset state so the user can pick a different source....
	m.youtubeURL = ""
	m.youtubePlaylistURL = ""
	m.dirToMp3Path = ""
	m.showDownloadProgress = false
	m.downloadProgress = &downloader.DownloadProgress{}
	m.state = screenMenu
	return m, nil
}

func (m *model) handleServerStarted(msg serverStartedMsg) (tea.Model, tea.Cmd) {
	m.serverPort = msg.port
	m.logger.ServerStarted(m.serverPort)
	return m, m.listenForClientUpdates()
}

func (m *model) handleServerError(msg serverErrorMsg) (tea.Model, tea.Cmd) {
	m.serverError = msg.err
	m.logger.Error("server error", "error", m.serverError)
	return m, nil
}

func (m *model) handleClientListUpdate(msg clientListUpdateMsg) (tea.Model, tea.Cmd) {
	m.connectedClients = msg.clients

	// if everyone disconnected mid-stream, just quit
	if m.state == screenStreamTracks && len(m.connectedClients) == 0 {
		if m.audioServer != nil {
			m.audioServer.Stop()
			m.audioServer = nil
		}
		return m, tea.Quit
	}

	return m, m.listenForClientUpdates()
}

func (m *model) handlePlaybackStarted() (tea.Model, tea.Cmd) {
	m.state = screenStreamTracks
	return m, tea.Batch(m.listenForStreamEvents(), m.listenForPlaybackDone())
}

func (m *model) handlePlaybackStopped(msg playbackStoppedMsg) (tea.Model, tea.Cmd) {
	m.currentTrackTitle = ""
	m.streamElapsed = 0

	if m.audioServer != nil {
		m.audioServer.Stop()
		m.audioServer = nil
	}

	return m, tea.Quit
}

func (m *model) handleStreamTick(msg streamTickMsg) (tea.Model, tea.Cmd) {
	m.streamElapsed = msg.elapsed
	m.currentTrackTitle = msg.track
	if m.audioServer != nil && m.state == screenStreamTracks {
		return m, m.listenForStreamEvents()
	}
	return m, nil
}

func (m *model) getServerAddresses() []string {
	var addrs []string
	ifaces, err := net.InterfaceAddrs()
	if err != nil {
		return []string{fmt.Sprintf("ws://localhost:%d/ws", m.serverPort)}
	}

	for _, addr := range ifaces {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				addrs = append(addrs, fmt.Sprintf("ws://%s:%d/ws", ipnet.IP.String(), m.serverPort))
			}
		}
	}

	if len(addrs) == 0 {
		addrs = append(addrs, fmt.Sprintf("ws://localhost:%d/ws", m.serverPort))
	}

	return addrs
}
