package ui

import "charm.land/lipgloss/v2"

func RenderConfirm(message string, width, height int) string {
	content := Bold.Render(message) + "\n\n" + Dim.Render("[y/N]")

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		FocusedBorder.
			Width(lipgloss.Width(message)+6).
			Padding(1, 2).
			Render(content),
	)
}
