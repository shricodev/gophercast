package tui

import (
	"fmt"
	"net"

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

		// for now, just test it with one audio file
		m.audioServer = server.NewAudioServer(m.selectedTracks.Current().Path.String())

		go func() {
			m.audioServer.Run()
		}()

		return serverStartedMsg{port: port}
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

func (m *model) sopAudioServer() tea.Cmd {
	return func() tea.Msg {
		if m.audioServer != nil {
			// if err := m.audioServer.Stop(); err != nil {
			// 	m.logger.Logger.Error("error stopping the server: %v", err)
			// }
			m.audioServer = nil
		}
		return nil
	}
}
