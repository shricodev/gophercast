package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

func (m *model) View() string {
	if m.err != nil {
		return docStyle.Render("Error: " + m.err.Error())
	}

	bannerPath := filepath.Join("assets", "banner.txt")
	content, err := os.ReadFile(bannerPath)
	if err != nil {
		content = []byte(`
█▀▀ █▀█ █▀█ █ █ █▀▀ █▀█ █▀▀ ▄▀█ █▀ ▀█▀
█▄█ █▄█ █▀▀ █▀█ ██▄ █▀▄ █▄▄ █▀█ ▄█  █ 
		`)
	}

	message := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("Built with 🤍 by Shrijal Acharya @shricodev")

	header := color.CyanString(lipgloss.JoinVertical(lipgloss.Left, string(content), message))

	var body string
	switch m.state {
	case screenMenu:
		body = m.initialScreenList.View()

	case screenPickDir:
		label := "Choose a directory:"
		if m.pickMode == "file" {
			label = "Choose an audio file:"
		}
		body = lipgloss.JoinVertical(lipgloss.Left,
			label,
			"",
			m.filePicker.View(),
		)

	case screenInputYoutube:
		body = lipgloss.JoinVertical(lipgloss.Left,
			"Enter YouTube URL:",
			"",
			m.textInput.View(),
			"",
			helpStyle.Render("Enter: Confirm, Esc: Back"),
		)

	case screenInputYoutubePlaylist:
		body = lipgloss.JoinVertical(lipgloss.Left,
			"Enter YouTube Playlist URL:",
			"",
			m.textInput.View(),
			"",
			helpStyle.Render("Enter: Confirm, Esc: Back"),
		)

	case screenChooseTracksOptions:
		body = m.chooseTracksOptionsList.View()

	case screenChooseTracks:
		// log.Println("items title:", m.selectedTracksList.Title)
		// log.Println("items:", len(m.selectedTracksList.Items()))
		body = m.selectedTracksList.View()

	case screenDownloadStarting:
		var message string
		var components []string

		if m.isShuttingDown {
			shutdownStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))

			components = append(components, shutdownStyle.Render(m.shutdownMessage))

			if m.showDownloadProgress && m.downloadProgress.Total > 0 {
				components = append(components, "")
				components = append(
					components,
					fmt.Sprintf(
						"Completing downloads: %d/%d",
						m.downloadProgress.Completed,
						m.downloadProgress.Total,
					),
				)

				if m.downloadProgress.Current != "" {
					components = append(
						components,
						fmt.Sprintf("Current: %s", m.downloadProgress.Current),
					)
				}

				if len(m.downloadProgress.Failed) > 0 {
					components = append(
						components,
						fmt.Sprintf("Failed: %d", len(m.downloadProgress.Failed)),
					)
				}

				components = append(components, "")
				components = append(components, m.progress.View())
			}
			components = append(components, "")
			components = append(
				components,
				helpStyle.Render("Please wait... Downloads will complete gracefully."),
			)
		} else {
			message = "Starting work, please wait..."
			if m.youtubeURL != "" {
				message = "Downloading YouTube video, please wait..."
			} else if m.youtubePlaylistURL != "" {
				message = "Downloading YouTube playlist, please wait..."
			}

			components = append(components, m.spinner.View()+" "+message)

			if m.showDownloadProgress && m.downloadProgress.Total > 0 {
				components = append(components, "")
				components = append(components, fmt.Sprintf("Progress: %d/%d completed",
					m.downloadProgress.Completed, m.downloadProgress.Total))

				if m.downloadProgress.Current != "" {
					components = append(components, fmt.Sprintf("Downloading: %s", m.downloadProgress.Current))
				}

				if len(m.downloadProgress.Failed) > 0 {
					components = append(components, fmt.Sprintf("Failed: %d", len(m.downloadProgress.Failed)))
				}

				components = append(components, "")
				components = append(components, m.progress.View())
			}

			components = append(components, "")
			components = append(
				components,
				helpStyle.Render("Press Ctrl + C for graceful shutdown"),
			)
		}

		body = lipgloss.JoinVertical(lipgloss.Left, components...)

	case screenLobby:
		var components []string

		components = append(components, m.spinner.View()+" Lobby - Waiting for clients")

		addrs := m.getServerAddresses()
		if len(addrs) > 0 {
			components = append(components, "")
			components = append(components, fmt.Sprintf("Server: %s", addrs[0]))
		}

		trackCount := 0
		if m.selectedTracks != nil {
			trackCount = m.selectedTracks.Len()
		}
		components = append(components, fmt.Sprintf("Tracks: %d selected", trackCount))

		components = append(components, "")
		components = append(components, fmt.Sprintf("Connected clients (%d):", len(m.connectedClients)))
		for i, c := range m.connectedClients {
			name := c.Name
			if name == "" {
				name = "(unnamed)"
			}
			components = append(components, fmt.Sprintf("  %d. %s - %s", i+1, name, c.Addr))
		}

		if len(m.connectedClients) == 0 {
			components = append(components, "  (none)")
		}

		components = append(components, "")
		if len(m.connectedClients) > 0 {
			components = append(components, helpStyle.Render("Enter: Start playback | Esc: Back | Ctrl+C: Quit"))
		} else {
			components = append(components, helpStyle.Render("Waiting for clients to connect... | Esc: Back | Ctrl+C: Quit"))
		}

		body = lipgloss.JoinVertical(lipgloss.Left, components...)

	case screenStreamTracks:
		var components []string

		if m.currentTrackTitle != "" {
			components = append(components, fmt.Sprintf("Now Playing: %s", m.currentTrackTitle))
		} else {
			components = append(components, "Streaming...")
		}

		minutes := int(m.streamElapsed.Minutes())
		seconds := int(m.streamElapsed.Seconds()) % 60
		components = append(components, fmt.Sprintf("Elapsed: %d:%02d", minutes, seconds))

		components = append(components, "")
		components = append(components, fmt.Sprintf("Connected clients (%d):", len(m.connectedClients)))
		for i, c := range m.connectedClients {
			name := c.Name
			if name == "" {
				name = "(unnamed)"
			}
			components = append(components, fmt.Sprintf("  %d. %s - %s", i+1, name, c.Addr))
		}
		if len(m.connectedClients) == 0 {
			components = append(components, "  (none)")
		}

		components = append(components, "")
		components = append(components, helpStyle.Render("Ctrl+C: Stop and exit"))

		body = lipgloss.JoinVertical(lipgloss.Left, components...)
	}

	full := lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", "")
	return docStyle.Render(full)
}
