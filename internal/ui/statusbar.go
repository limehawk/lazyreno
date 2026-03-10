package ui

import (
	"time"

	"charm.land/lipgloss/v2"
)

type FlashMessage struct {
	Text      string
	IsError   bool
	ExpiresAt time.Time
}

func RenderStatusBar(context string, keyHints string, flash *FlashMessage, width int) string {
	left := Dim.Render(context)
	right := Dim.Render(keyHints)

	if flash != nil && time.Now().Before(flash.ExpiresAt) {
		style := SuccessText
		if flash.IsError {
			style = ErrorText
		}
		left = style.Render(flash.Text)
	}

	// Account for 1-char padding on each side
	innerWidth := width - 2
	if innerWidth < 0 {
		innerWidth = 0
	}

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := innerWidth - leftWidth - rightWidth
	if gap < 0 {
		gap = 0
	}

	bar := left + lipgloss.NewStyle().Width(gap).Render("") + right

	return StatusBar.
		Width(innerWidth).
		MaxHeight(1).
		Render(bar)
}
