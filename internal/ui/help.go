package ui

import "charm.land/lipgloss/v2"

type HelpEntry struct {
	Key         string
	Description string
}

func RenderHelp(entries []HelpEntry, width, height int) string {
	var lines []string
	for _, e := range entries {
		line := HelpKey.Render(e.Key) + "  " + HelpValue.Render(e.Description)
		lines = append(lines, line)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		FocusedBorder.
			Width(40).
			Height(len(entries)+2).
			Padding(1, 2).
			Render(content),
	)
}
