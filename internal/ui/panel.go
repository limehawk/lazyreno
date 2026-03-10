package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Panel renders content inside a bordered box with a title.
// Focused panels get the accent border.
type Panel struct {
	Title   string
	Content string
	Focused bool
	Width   int
	Height  int
}

func (p Panel) View() string {
	style := UnfocusedBorder
	if p.Focused {
		style = FocusedBorder
	}

	// Account for border (2 chars each side)
	innerWidth := p.Width - 2
	innerHeight := p.Height - 2
	if innerWidth < 0 {
		innerWidth = 0
	}
	if innerHeight < 0 {
		innerHeight = 0
	}

	rendered := style.
		Width(innerWidth).
		Height(innerHeight).
		Render(lipgloss.Place(
			innerWidth, innerHeight,
			lipgloss.Left, lipgloss.Top,
			p.Content,
		))

	// Overlay title onto top border
	if p.Title != "" {
		titleStr := " " + p.Title + " "
		lines := strings.SplitN(rendered, "\n", 2)
		if len(lines) >= 1 {
			topBorder := []rune(lines[0])
			titleRunes := []rune(titleStr)
			// Place title starting at position 2 (after corner + border char)
			insertAt := 2
			if insertAt+len(titleRunes) < len(topBorder) {
				for i, r := range titleRunes {
					topBorder[insertAt+i] = r
				}
			}
			lines[0] = string(topBorder)
			rendered = strings.Join(lines, "\n")
		}
	}

	return rendered
}
