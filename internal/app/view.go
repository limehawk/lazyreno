package app

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/limehawk/lazyreno/internal/ui"
)

func (m Model) View() tea.View {
	view := func(s string) tea.View {
		v := tea.NewView(s)
		v.AltScreen = true
		return v
	}

	if m.width == 0 {
		return view("Loading...")
	}

	if m.confirmForm != nil {
		return view(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.confirmForm.View()))
	}

	// Repos overlay
	if m.showRepos {
		return view(m.viewReposOverlay())
	}

	// Status bar
	updatedAgo := ""
	if !m.lastUpdate.IsZero() {
		d := time.Since(m.lastUpdate).Truncate(time.Second)
		updatedAgo = "updated " + d.String() + " ago"
	}
	header := ui.RenderStatusBar(m.spinner.View(), updatedAgo, m.width)

	// Help bar
	helpBar := m.help.View(m.keys)

	var bottomLines []string
	if m.flashText != "" && time.Now().Before(m.flashExpiry) {
		style := ui.SuccessText
		if m.flashIsError {
			style = ui.ErrorText
		}
		bottomLines = append(bottomLines, style.Render(m.flashText))
	}
	bottomLines = append(bottomLines, helpBar)
	bottom := lipgloss.JoinVertical(lipgloss.Left, bottomLines...)

	bodyHeight := m.height - lipgloss.Height(header) - lipgloss.Height(bottom)
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	body := m.viewDashboard(bodyHeight)

	return view(lipgloss.JoinVertical(lipgloss.Left, header, body, bottom))
}

func (m Model) viewDashboard(height int) string {
	leftWidth, rightWidth := m.panelWidths()

	// Left column: stacked panels
	systemH := 5
	jobsH := 8
	prListH := height - systemH - jobsH
	if prListH < 6 {
		prListH = 6
	}

	prSidebar := ui.WrapListInPanel(
		m.repoList.View(),
		m.focusedPanel == 0, leftWidth, prListH,
	)

	systemPanel := ui.RenderPanel(
		"System", m.renderStatusBox(),
		false, leftWidth, systemH,
	)

	jobsPanel := ui.WrapListInPanel(
		m.jobList.View(),
		false, leftWidth, jobsH,
	)

	leftCol := lipgloss.JoinVertical(lipgloss.Left, prSidebar, systemPanel, jobsPanel)

	// Right column: PR table top, detail bottom
	tableH := height * 55 / 100
	detailH := height - tableH

	tablePanel := ui.RenderPanel(
		"", m.prTable.View(),
		m.focusedPanel == 1, rightWidth, tableH,
	)

	detailPanel := ui.RenderPanel(
		"Details", m.detailView.View(),
		m.focusedPanel == 2, rightWidth, detailH,
	)

	rightCol := lipgloss.JoinVertical(lipgloss.Left, tablePanel, detailPanel)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)
}

func (m Model) viewReposOverlay() string {
	content := m.allRepoList.View()
	overlay := ui.RenderPanel("All Repos (press 2 or esc to close)", content, true, m.width-2, m.height-2)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
}
