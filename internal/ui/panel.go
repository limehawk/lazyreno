package ui


// RenderPanel wraps static content in a bordered panel with a title line.
// Used for non-list content (detail views, info panels, status).
// The title is rendered as the first line inside the panel.
func RenderPanel(theme *Theme, title, content string, focused bool, width, height int) string {
	if width < 4 || height < 3 {
		return ""
	}

	style := theme.PanelBorder(focused)

	// In lipgloss v2, Width/Height set the total outer dimension.
	style = style.Width(width).Height(height)

	if title != "" {
		titleStr := theme.PanelTitle(title, focused)
		content = titleStr + "\n" + content
	}

	return style.Render(content)
}

// WrapListInPanel wraps a list.View() output in a bordered panel.
// The list handles its own title — this just adds the border frame.
func WrapListInPanel(theme *Theme, title, listView string, focused bool, width, height int) string {
	if width < 4 || height < 3 {
		return ""
	}

	style := theme.PanelBorder(focused)

	// In lipgloss v2, Width/Height set the total outer dimension.
	style = style.Width(width).Height(height)

	// The list bubble manages its own content. We don't pad or trim
	// here — just wrap in a border frame.
	_ = title // list panels: title is managed by the list.Model itself.

	return style.Render(listView)
}

// InnerSize returns the content width and height available inside a panel.
func InnerSize(theme *Theme, width, height int) (int, int) {
	style := theme.PanelBorder(false)
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
