package ui

// RenderPanel wraps static content in a bordered panel with a title line.
func RenderPanel(title, content string, focused bool, width, height int) string {
	if width < 4 || height < 3 {
		return ""
	}

	style := PanelBorder(focused).Width(width).Height(height)

	if title != "" {
		content = PanelTitle(title, focused) + "\n" + content
	}

	return style.Render(content)
}

// WrapListInPanel wraps a list.View() output in a bordered panel.
func WrapListInPanel(listView string, focused bool, width, height int) string {
	if width < 4 || height < 3 {
		return ""
	}

	style := PanelBorder(focused).Width(width).Height(height)
	return style.Render(listView)
}

// InnerSize returns the content width and height available inside a panel.
func InnerSize(width, height int) (int, int) {
	style := InactiveBorder
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
