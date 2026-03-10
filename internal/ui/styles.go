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

	// Unused legacy — kept for reference, panel.go builds borders manually now
	FocusedBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Accent)

	UnfocusedBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
)
