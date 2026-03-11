package ui

import "charm.land/lipgloss/v2"

func RenderStatusBar(spinnerView string, lastUpdated string, width int) string {
	left := Bold.Render("lazyreno")
	if spinnerView != "" {
		left += " " + spinnerView
	}
	right := Dim.Render(lastUpdated)

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	spacer := lipgloss.NewStyle().Width(gap).Render("")
	bar := lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right)
	return lipgloss.NewStyle().Width(width).Render(bar)
}
