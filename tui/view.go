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
█▀▀ █▀█ █▀█ █░█ █▀▀ █▀█ █▀▀ ▄▀█ █▀ ▀█▀
█▄█ █▄█ █▀▀ █▀█ ██▄ █▀▄ █▄▄ █▀█ ▄█ ░█░
		`)
	}

	message := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("Built with 🤍 by Shrijal Acharya @shricodev")

	header := color.CyanString(string(content) + "\n" + message)

	var body string
	switch m.state {
	case screenMenu:
		body = m.list.View()
	case screenPickDir:
		body = "Choose a directory:\n\n" + m.filePicker.View() + "\n" +
			helpStyle.Render("↑/↓: Navigate, Enter: Select, d: Select current, Esc: Back")
	case screenInputYoutube:
		body = "Enter YouTube URL:\n\n" + m.textInput.View() + "\n\n" +
			helpStyle.Render("Press Enter to confirm, Esc to go back")
	case screenInputPlaylist:
		body = "Enter YouTube Playlist URL:\n\n" + m.textInput.View() + "\n\n" +
			helpStyle.Render("Press Enter to confirm, Esc to go back")
	case screenAppStarting:
		body = m.spinner.View() + " " + "Starting work, please wait..."
	case screenAppRunning:
		body = "Server running!\n\n" + helpStyle.Render("Press Ctrl + C to exit")
	}

	full := header + "\n\n" + body + "\n\n"
	return docStyle.Render(full)
}
