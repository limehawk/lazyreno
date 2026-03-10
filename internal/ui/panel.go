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
	if p.Width < 4 || p.Height < 3 {
		return ""
	}

	borderColor := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	if p.Focused {
		borderColor = lipgloss.NewStyle().Foreground(Accent)
		titleStyle = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	}

	innerWidth := p.Width - 2
	innerHeight := p.Height - 2

	// Build top border: ╭─ Title ────────╮
	topLeft := borderColor.Render("╭")
	topRight := borderColor.Render("╮")
	bottomLeft := borderColor.Render("╰")
	bottomRight := borderColor.Render("╯")
	hBar := borderColor.Render("─")
	vBar := borderColor.Render("│")

	var topLine string
	if p.Title != "" {
		titleStr := titleStyle.Render(" " + p.Title + " ")
		titleVisualWidth := lipgloss.Width(titleStr)
		remainingBars := innerWidth - titleVisualWidth
		if remainingBars < 0 {
			remainingBars = 0
		}
		leftBars := 1
		rightBars := remainingBars - leftBars
		if rightBars < 0 {
			rightBars = 0
		}
		topLine = topLeft + strings.Repeat(hBar, leftBars) + titleStr + strings.Repeat(hBar, rightBars) + topRight
	} else {
		topLine = topLeft + strings.Repeat(hBar, innerWidth) + topRight
	}

	// Build bottom border: ╰────────────────╯
	bottomLine := bottomLeft + strings.Repeat(hBar, innerWidth) + bottomRight

	// Pad/truncate content to fit inner dimensions
	contentLines := strings.Split(p.Content, "\n")
	for len(contentLines) < innerHeight {
		contentLines = append(contentLines, "")
	}
	if len(contentLines) > innerHeight {
		contentLines = contentLines[:innerHeight]
	}

	// Build middle rows with vertical borders
	var rows []string
	rows = append(rows, topLine)
	for _, line := range contentLines {
		lineWidth := lipgloss.Width(line)
		padding := innerWidth - lineWidth
		if padding < 0 {
			// Truncate if line is too wide — crude but functional
			line = runeSlice(line, innerWidth-1) + "…"
			padding = 0
		}
		rows = append(rows, vBar+line+strings.Repeat(" ", padding)+vBar)
	}
	rows = append(rows, bottomLine)

	return strings.Join(rows, "\n")
}

// runeSlice returns up to n visible characters from a string,
// being careful about multi-byte runes. Does not handle ANSI well
// but is a reasonable fallback.
func runeSlice(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
