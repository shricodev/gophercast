package tui

import (
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) updateBubbles(msg tea.Msg) []tea.Cmd {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch m.state {
	case screenMenu:
		m.initialScreenList, cmd = m.initialScreenList.Update(msg)
		cmds = append(cmds, cmd)

	case screenPickDir:
		m.filePicker, cmd = m.filePicker.Update(msg)
		cmds = append(cmds, cmd)

	case screenInputYoutube, screenInputYoutubePlaylist:
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)

	case screenDownloadStarting:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)

	case screenLobby:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case screenChooseTracksOptions:
		m.chooseTracksOptionsList, cmd = m.chooseTracksOptionsList.Update(msg)
		cmds = append(cmds, cmd)

	case screenChooseTracks:
		m.selectedTracksList, cmd = m.selectedTracksList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return cmds
}
