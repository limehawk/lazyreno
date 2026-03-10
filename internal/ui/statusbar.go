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

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	spacer := lipgloss.NewStyle().Width(gap).Render("")

	return StatusBar.Width(width).Render(
		lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right),
	)
}
