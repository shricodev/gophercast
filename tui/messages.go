package tui

import (
	"fmt"
	"net"

	tea "github.com/charmbracelet/bubbletea"
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
	case clientConnectedMsg:
		return m.handleClientConnected(msg)
	case trackChangedMsg:
		return m.handleTrackChanged(msg)
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
		return m, m.startAudioServer()
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
		// m.audioServer.Stop()
		m.audioServer = nil
	}

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

func (m *model) handleServerStarted(msg serverStartedMsg) (tea.Model, tea.Cmd) {
	m.serverPort = msg.port
	m.logger.Logger.Info("server started on port %d", m.serverPort)

	m.printNetworkInfo()

	return m, nil
}

func (m *model) handleServerError(msg serverErrorMsg) (tea.Model, tea.Cmd) {
	m.serverError = msg.err
	m.logger.Logger.Error("server error: %v", m.serverError)
	return m, nil
}

func (m *model) printNetworkInfo() {
	fmt.Println("\n=== GopherCast Server Started ===")
	fmt.Printf("Server running on port: %d\n", m.serverPort)

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		m.logger.Logger.Info("error getting network interfaces: %v", err)
		return
	}

	fmt.Println("\nClients can connect to:")
	for _, addrs := range addrs {
		if ipnet, ok := addrs.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				fmt.Printf("  http://%s:%d\n", ipnet.IP.String(), m.serverPort)
				fmt.Printf("  ws://%s:%d/ws\n", ipnet.IP.String(), m.serverPort)
			}
		}
	}

	fmt.Printf("  http://localhost:%d (local only)\n", m.serverPort)
	fmt.Printf("  http://127.0.0.1:%d (local only)\n", m.serverPort)
	fmt.Println("\n================================")
}

func (m *model) handleClientConnected(msg clientConnectedMsg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *model) handleTrackChanged(msg trackChangedMsg) (tea.Model, tea.Cmd) {
	return m, nil
}
