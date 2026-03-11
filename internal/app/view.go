package app

import (
	"fmt"
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

	// Bottom bar: confirm prompt or help
	var bottom string
	if m.confirmText != "" {
		bottom = ui.WarningText.Render(m.confirmText)
	} else {
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
		bottom = lipgloss.JoinVertical(lipgloss.Left, bottomLines...)
	}

	bodyHeight := m.height - lipgloss.Height(header) - lipgloss.Height(bottom)
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	body := m.viewDashboard(bodyHeight)

	return view(lipgloss.JoinVertical(lipgloss.Left, header, body, bottom))
}

func (m Model) viewDashboard(height int) string {
	sidebarW, rightW := m.cachedSidebarW, m.cachedRightW

	// Left: sidebar (full height)
	sidebar := ui.WrapListInPanel(
		m.repoList.View(),
		m.focusedPanel == 0, sidebarW, height,
	)

	// Right: table top ~60%, bento bottom ~40%
	tableH := height * 60 / 100
	bentoH := height - tableH

	tablePanel := ui.RenderPanel(
		"", m.prTable.View(),
		m.focusedPanel == 1, rightW, tableH,
	)

	// Bottom bento: Detail | System | Jobs
	detailW, systemW, jobsW := m.cachedDetailW, m.cachedSystemW, m.cachedJobsW

	detailPanel := ui.RenderPanel(
		"Details", m.detailView.View(),
		m.focusedPanel == 2, detailW, bentoH,
	)

	systemContent := m.renderStatusBox()
	if m.renovate != nil {
		systemContent += fmt.Sprintf("\n\n%s sync  %s purge",
			ui.ShortcutKey.Render("[s]"), ui.ShortcutKey.Render("[p]"))
	}
	systemPanel := ui.RenderPanel(
		"System", systemContent,
		false, systemW, bentoH,
	)

	jobsTitle := "Jobs"
	if len(m.jobs) > 0 {
		jobsTitle = fmt.Sprintf("Jobs (%d)", len(m.jobs))
	}
	jobsPanel := ui.RenderPanel(
		jobsTitle, m.renderJobsPanel(),
		false, jobsW, bentoH,
	)

	bentoRow := lipgloss.JoinHorizontal(lipgloss.Top, detailPanel, systemPanel, jobsPanel)
	rightCol := lipgloss.JoinVertical(lipgloss.Left, tablePanel, bentoRow)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, rightCol)
}

func (m Model) viewReposOverlay() string {
	content := m.allRepoList.View()
	overlay := ui.RenderPanel("All Repos (press 2 or esc to close)", content, true, m.width-2, m.height-2)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
}
