package tui

import tea "github.com/charmbracelet/bubbletea"

func (m *model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	hor, ver := docStyle.GetFrameSize()

	listWidth := max(msg.Width-hor, 40)
	listHeight := max(msg.Height-ver-30, 10)

	m.initialScreenList.SetSize(listWidth, listHeight)
	m.chooseTracksOptionsList.SetSize(listWidth, listHeight)
	m.selectedTracksList.SetSize(listWidth, listHeight)

	return m, nil
}
