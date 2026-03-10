package ui

import "charm.land/lipgloss/v2"

// ANSI colors — inherits from terminal theme automatically.
var (
	Red     = lipgloss.Color("1")
	Green   = lipgloss.Color("2")
	Yellow  = lipgloss.Color("3")
	Blue    = lipgloss.Color("4")
	Magenta = lipgloss.Color("5")
	Cyan    = lipgloss.Color("6")
	White   = lipgloss.Color("7")
	Gray    = lipgloss.Color("8")

	BrightRed   = lipgloss.Color("9")
	BrightGreen = lipgloss.Color("10")
	BrightBlue  = lipgloss.Color("12")
	BrightCyan  = lipgloss.Color("14")
	BrightWhite = lipgloss.Color("15")
)

// Pre-built styles.
var (
	Bold        = lipgloss.NewStyle().Bold(true)
	Dim         = lipgloss.NewStyle().Foreground(Gray)
	AccentText  = lipgloss.NewStyle().Foreground(Blue).Bold(true)
	ErrorText   = lipgloss.NewStyle().Foreground(Red)
	WarningText = lipgloss.NewStyle().Foreground(Yellow)
	SuccessText = lipgloss.NewStyle().Foreground(Green)

	ActiveTab = lipgloss.NewStyle().
			Bold(true).
			Foreground(Blue).
			Padding(0, 1)

	InactiveTab = lipgloss.NewStyle().
			Foreground(Gray).
			Padding(0, 1)

	ActiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Blue).
			Padding(0, 1)

	InactiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Gray).
			Padding(0, 1)

	ShortcutKey = lipgloss.NewStyle().Foreground(Cyan).Bold(true)
)

// PanelBorder returns the border style for a panel.
func PanelBorder(focused bool) lipgloss.Style {
	if focused {
		return ActiveBorder
	}
	return InactiveBorder
}

// PanelTitle returns a styled title string.
func PanelTitle(title string, focused bool) string {
	if focused {
		return AccentText.Render(title)
	}
	return Dim.Render(title)
}
