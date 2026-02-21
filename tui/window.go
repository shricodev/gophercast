package tui

import tea "github.com/charmbracelet/bubbletea"

func (m *model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.windowWidth = msg.Width
	m.windowHeight = msg.Height

	hor, ver := docStyle.GetFrameSize()

	listWidth := max(msg.Width-hor, 40)
	listHeight := max(msg.Height-ver-headerHeight, 10)

	m.initialScreenList.SetSize(listWidth, listHeight)
	m.chooseTracksOptionsList.SetSize(listWidth, listHeight)
	m.selectedTracksList.SetSize(listWidth, listHeight)

	// File picker: subtract header, the "Choose a directory:" label, and help text (~4 extra lines)
	fpHeight := max(msg.Height-ver-headerHeight-4, 5)
	m.filePicker.Height = fpHeight

	return m, nil
}
