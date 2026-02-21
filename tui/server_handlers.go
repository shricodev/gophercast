package tui

import (
	"fmt"
	"net"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shricodev/gophercast/server"
)

func (m *model) startAudioServer() tea.Cmd {
	return func() tea.Msg {
		if m.selectedTracks == nil || len(*m.selectedTracks) == 0 {
			return serverErrorMsg{err: fmt.Errorf("no tracks selected for streaming")}
		}

		port, err := m.findAvailablePort(m.serverPort)
		if err != nil {
			return serverErrorMsg{err: fmt.Errorf("no available ports found")}
		}

		m.audioServer = server.NewAudioServer(m.selectedTracks, port, m.logger)

		if err := m.audioServer.ListenAndServe(); err != nil {
			return serverErrorMsg{err: fmt.Errorf("server failed to start: %w", err)}
		}

		return serverStartedMsg{port: m.audioServer.Port()}
	}
}

func (m *model) startPlayback() tea.Cmd {
	return func() tea.Msg {
		if m.audioServer == nil {
			return serverErrorMsg{err: fmt.Errorf("no server running")}
		}

		if err := m.audioServer.StartPlayback(); err != nil {
			return serverErrorMsg{err: fmt.Errorf("failed to start playback: %w", err)}
		}

		return playbackStartedMsg{}
	}
}

func (m *model) listenForClientUpdates() tea.Cmd {
	return func() tea.Msg {
		if m.audioServer == nil {
			return nil
		}

		select {
		case clients, ok := <-m.audioServer.ClientChangeChan():
			if !ok {
				return nil
			}
			return clientListUpdateMsg{clients: clients}
		}
	}
}

func (m *model) listenForStreamEvents() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(500 * time.Millisecond)
		if m.audioServer == nil {
			return nil
		}
		return streamTickMsg{
			elapsed: m.audioServer.PlaybackElapsed(),
			track:   m.audioServer.CurrentTrackTitle(),
		}
	}
}

func (m *model) findAvailablePort(startPort int) (int, error) {
	for port := startPort; port < startPort+100; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			_ = ln.Close()
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports found")
}

func (m *model) stopAudioServer() tea.Cmd {
	return func() tea.Msg {
		if m.audioServer != nil {
			if err := m.audioServer.Stop(); err != nil {
				m.logger.Error("error stopping the server", "error", err)
			}
			m.audioServer = nil
		}
		return nil
	}
}
