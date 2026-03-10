package ui

import "charm.land/lipgloss/v2"

var (
	// Accent color — cyan by default.
	Accent = lipgloss.Color("6") // ANSI cyan

	// Panel borders
	FocusedBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Accent)

	UnfocusedBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")) // dim gray

	// Text styles
	Bold       = lipgloss.NewStyle().Bold(true)
	Dim        = lipgloss.NewStyle().Faint(true)
	AccentText = lipgloss.NewStyle().Foreground(Accent)

	// Semantic colors
	ErrorText   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	WarningText = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	SuccessText = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green

	// Status bar
	StatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("8")).
			Padding(0, 1)

	// Tab bar
	ActiveTab = lipgloss.NewStyle().
			Bold(true).
			Foreground(Accent).
			Padding(0, 1)

	InactiveTab = lipgloss.NewStyle().
			Faint(true).
			Padding(0, 1)

	// Help overlay
	HelpKey   = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	HelpValue = lipgloss.NewStyle().Faint(true)
)
