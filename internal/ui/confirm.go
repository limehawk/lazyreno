package ui

import "charm.land/lipgloss/v2"

func RenderConfirm(message string, width, height int) string {
	content := Bold.Render(message) + "\n\n" + Dim.Render("[y/N]")

	panelWidth := lipgloss.Width(message) + 8
	if panelWidth < 30 {
		panelWidth = 30
	}

	panel := RenderPanel("Confirm", content, true, panelWidth, 7)

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		panel,
	)
}
