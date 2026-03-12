package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	descStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 2 }
func (d itemDelegate) Spacing() int                              { return 1 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd  { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	var checkbox string
	if i.selected {
		checkbox = selectedStyle.Render("[x]")
	} else {
		checkbox = "[ ]"
	}

	title := i.Title()
	desc := i.Description()

	if index == m.Index() {
		titleLine := fmt.Sprintf("%s %s", checkbox, title)
		descLine := fmt.Sprintf("  %s", desc)

		_, _ = fmt.Fprint(w, focusedStyle.Render(fmt.Sprintf("> %s\n%s", titleLine, descLine)))
	} else {
		var titleLine string
		if i.selected {
			titleLine = fmt.Sprintf("  %s %s", checkbox, selectedStyle.Render(title))
		} else {
			titleLine = fmt.Sprintf("  %s %s", checkbox, title)
		}
		descLine := descStyle.Render(fmt.Sprintf("  %s", desc))
		_, _ = fmt.Fprintf(w, "%s\n%s", titleLine, descLine)
	}
}
