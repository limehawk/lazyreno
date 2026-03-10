package ui

import "strings"

// RenderPanel wraps content in a bordered panel with a title.
// It uses lipgloss rounded borders instead of hand-drawn box characters.
func RenderPanel(title, content string, focused bool, width, height int) string {
	if width < 4 || height < 3 {
		return ""
	}

	style := PanelBorder(focused)

	// The border + padding consume space. We set the inner dimensions
	// so the outer dimensions match the requested width/height.
	// Border takes 2 cols (left+right), padding(0,1) takes 2 cols.
	innerWidth := width - style.GetHorizontalFrameSize()
	innerHeight := height - style.GetVerticalFrameSize()
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	style = style.Width(innerWidth).Height(innerHeight)

	// Set the title in the top border if provided.
	if title != "" {
		titleStr := PanelTitle(" "+title+" ", focused)
		style = style.BorderTop(true).
			SetString(titleStr)
	}

	return style.Render(content)
}

// RenderPanelAround wraps already-rendered content (e.g. from list.View())
// in a bordered panel. The content is placed as-is; only padding lines are
// added to fill the panel height.
func RenderPanelAround(title, content string, focused bool, width, height int) string {
	if width < 4 || height < 3 {
		return ""
	}

	style := PanelBorder(focused)

	innerWidth := width - style.GetHorizontalFrameSize()
	innerHeight := height - style.GetVerticalFrameSize()
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Pad content to fill height.
	lines := strings.Split(content, "\n")
	for len(lines) < innerHeight {
		lines = append(lines, "")
	}
	if len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}
	padded := strings.Join(lines, "\n")

	style = style.Width(innerWidth).Height(innerHeight)

	if title != "" {
		titleStr := PanelTitle(" "+title+" ", focused)
		style = style.SetString(titleStr)
	}

	return style.Render(padded)
}

// WrapListInPanel wraps a list.View() output in a bordered panel.
// The list should already be sized to fit the inner dimensions.
func WrapListInPanel(title, listView string, focused bool, width, height int) string {
	return RenderPanelAround(title, listView, focused, width, height)
}

// InnerSize returns the inner width and height for a panel of the given outer dimensions.
func InnerSize(width, height int) (int, int) {
	style := PanelBorder(false)
	iw := width - style.GetHorizontalFrameSize()
	ih := height - style.GetVerticalFrameSize()
	if iw < 1 {
		iw = 1
	}
	if ih < 1 {
		ih = 1
	}
	return iw, ih
}
