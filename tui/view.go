package tui

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

func (m model) View() string {
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
		body = m.list.View()
	case screenPickDir:
		body = lipgloss.JoinVertical(lipgloss.Left,
			"Choose a directory:",
			"",
			m.filePicker.View(),
			helpStyle.Render("↑/↓ j/k: Navigate, Enter: Select, d: Select cwd, Esc: Back"),
		)
	case screenInputYoutube:
		body = lipgloss.JoinVertical(lipgloss.Left,
			"Enter YouTube URL:",
			"",
			m.textInput.View(),
			"",
			helpStyle.Render("Enter: Confirm, Esc: Back"),
		)
	case screenInputPlaylist:
		body = lipgloss.JoinVertical(lipgloss.Left,
			"Enter YouTube Playlist URL:",
			"",
			m.textInput.View(),
			"",
			helpStyle.Render("Enter: Confirm, Esc: Back"),
		)
	case screenAppStarting:
		body = m.spinner.View() + " " + "Starting work, please wait..."
	case screenAppRunning:
		body = lipgloss.JoinVertical(lipgloss.Left,
			"Server running!",
			"",
			helpStyle.Render("Press Ctrl + C to exit"),
		)
	}

	full := lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", "")
	return docStyle.Render(full)
}
