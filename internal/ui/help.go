package ui

import "charm.land/lipgloss/v2"

type HelpEntry struct {
	Key         string
	Description string
}

func RenderHelp(entries []HelpEntry, width, height int) string {
	var lines []string
	for _, e := range entries {
		line := HelpKey.Render(padKeyRight(e.Key, 6)) + "  " + HelpValue.Render(e.Description)
		lines = append(lines, line)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	panelWidth := 44
	panelHeight := len(entries) + 4

	panel := Panel{
		Title:   "Help",
		Content: content,
		Focused: true,
		Width:   panelWidth,
		Height:  panelHeight,
	}

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		panel.View(),
	)
}

func padKeyRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + "        "[:width-w]
}
