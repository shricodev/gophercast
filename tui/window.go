package tui

import tea "github.com/charmbracelet/bubbletea"

func (m *model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.windowWidth = msg.Width
	m.windowHeight = msg.Height

	hor, ver := docStyle.GetFrameSize()

	listWidth := max(msg.Width-hor, 40)
	listHeight := max(msg.Height-ver-headerHeight, 10)

	// cap small menus so the help text isn't weirdly far from the items
	menuHeight := min(listHeight, len(m.initialScreenList.Items())*3+7)
	optsHeight := min(listHeight, len(m.chooseTracksOptionsList.Items())*3+7)

	m.initialScreenList.SetSize(listWidth, menuHeight)
	m.chooseTracksOptionsList.SetSize(listWidth, optsHeight)
	m.selectedTracksList.SetSize(listWidth, listHeight)

	fpHeight := max(msg.Height-ver-headerHeight-4, 5)
	m.filePicker.SetHeight(fpHeight)

	return m, nil
}
