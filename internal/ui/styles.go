package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme holds all colors and pre-built styles, adapted for light or dark
// terminal backgrounds.
type Theme struct {
	// Raw colors
	Accent         color.Color
	DimColor       color.Color
	ErrorColor     color.Color
	WarningColor   color.Color
	SuccessColor   color.Color
	UnfocusedBorder color.Color

	// Pre-built styles
	Bold        lipgloss.Style
	Dim         lipgloss.Style
	AccentText  lipgloss.Style
	ErrorText   lipgloss.Style
	WarningText lipgloss.Style
	SuccessText lipgloss.Style
	ActiveTab   lipgloss.Style
	InactiveTab lipgloss.Style
}

// NewTheme builds a Theme using lipgloss.LightDark to pick colors appropriate
// for the terminal background.
func NewTheme(hasDarkBG bool) Theme {
	ld := lipgloss.LightDark(hasDarkBG)

	accent := ld(lipgloss.Color("#0097b2"), lipgloss.Color("#00d7ff"))
	dim := ld(lipgloss.Color("#888888"), lipgloss.Color("#6c6c6c"))
	errColor := ld(lipgloss.Color("#cc3333"), lipgloss.Color("#ff5f5f"))
	warnColor := ld(lipgloss.Color("#cc8800"), lipgloss.Color("#ffaf00"))
	successColor := ld(lipgloss.Color("#22aa44"), lipgloss.Color("#5fff87"))
	unfocused := ld(lipgloss.Color("#aaaaaa"), lipgloss.Color("#585858"))

	return Theme{
		Accent:          accent,
		DimColor:        dim,
		ErrorColor:      errColor,
		WarningColor:    warnColor,
		SuccessColor:    successColor,
		UnfocusedBorder: unfocused,

		Bold:       lipgloss.NewStyle().Bold(true),
		Dim:        lipgloss.NewStyle().Foreground(dim),
		AccentText: lipgloss.NewStyle().Foreground(accent).Bold(true),
		ErrorText:  lipgloss.NewStyle().Foreground(errColor),
		WarningText: lipgloss.NewStyle().Foreground(warnColor),
		SuccessText: lipgloss.NewStyle().Foreground(successColor),

		ActiveTab: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Padding(0, 1),

		InactiveTab: lipgloss.NewStyle().
			Foreground(dim).
			Padding(0, 1),
	}
}

// PanelBorder returns a lipgloss style with a rounded border.
// Focused panels get the accent color, unfocused get a dim gray.
func (t *Theme) PanelBorder(focused bool) lipgloss.Style {
	c := t.UnfocusedBorder
	if focused {
		c = t.Accent
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c).
		Padding(0, 1)
}

// PanelTitle returns a styled title string for a panel header.
func (t *Theme) PanelTitle(title string, focused bool) string {
	c := t.UnfocusedBorder
	if focused {
		c = t.Accent
	}
	return lipgloss.NewStyle().
		Foreground(c).
		Bold(focused).
		Render(title)
}
