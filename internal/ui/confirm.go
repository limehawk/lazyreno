package ui

import "charm.land/lipgloss/v2"

func RenderConfirm(message string, width, height int) string {
	content := Bold.Render(message) + "\n\n" + Dim.Render("[y/N]")

	panelWidth := lipgloss.Width(message) + 8
	if panelWidth < 30 {
		panelWidth = 30
	}

	panel := Panel{
		Title:   "Confirm",
		Content: content,
		Focused: true,
		Width:   panelWidth,
		Height:  7,
	}

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		panel.View(),
	)
}
