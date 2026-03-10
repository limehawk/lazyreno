package ui

import "charm.land/lipgloss/v2"

var (
	// Accent color — bright cyan.
	Accent = lipgloss.Color("#00d7ff")

	// Text styles
	Bold       = lipgloss.NewStyle().Bold(true)
	Dim        = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	AccentText = lipgloss.NewStyle().Foreground(Accent).Bold(true)

	// Semantic colors
	ErrorText   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f")) // bright red
	WarningText = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaf00")) // bright yellow/orange
	SuccessText = lipgloss.NewStyle().Foreground(lipgloss.Color("#5fff87")) // bright green

	// Tab bar
	ActiveTab = lipgloss.NewStyle().
			Bold(true).
			Foreground(Accent).
			Padding(0, 1)

	InactiveTab = lipgloss.NewStyle().
			Foreground(lipgloss.Color("242")).
			Padding(0, 1)
)

// PanelBorder returns a lipgloss style with a rounded border.
// Focused panels get the accent color, unfocused get a dim gray.
func PanelBorder(focused bool) lipgloss.Style {
	color := lipgloss.Color("240")
	if focused {
		color = Accent
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Padding(0, 1)
}

// PanelTitle returns a styled title string for a panel header.
func PanelTitle(title string, focused bool) string {
	color := lipgloss.Color("240")
	if focused {
		color = Accent
	}
	return lipgloss.NewStyle().
		Foreground(color).
		Bold(focused).
		Render(title)
}
