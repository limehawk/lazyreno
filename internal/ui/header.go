package ui

import "charm.land/lipgloss/v2"

var tabNames = []string{"PRs", "Repos", "Jobs", "Status"}

func RenderHeader(theme *Theme, activeTab int, width int) string {
	var tabs []string
	for i, name := range tabNames {
		label := "[" + string(rune('1'+i)) + "] " + name
		if i == activeTab {
			tabs = append(tabs, theme.ActiveTab.Render(label))
		} else {
			tabs = append(tabs, theme.InactiveTab.Render(label))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	return lipgloss.NewStyle().Width(width).Render(row)
}
