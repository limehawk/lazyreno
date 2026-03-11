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

	// Status bar: lazyreno ⣾  ▸ repo-name (3)             updated 12s ago
	updatedAgo := ""
	if !m.lastUpdate.IsZero() {
		d := time.Since(m.lastUpdate).Truncate(time.Second)
		updatedAgo = "updated " + d.String() + " ago"
	}
	header := ui.RenderStatusBar(m.spinner.View(), m.repoInfo(), updatedAgo, m.width)

	// Bottom bar: either confirm prompt or help
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
	// Table takes top ~60%, bento grid takes bottom ~40%
	tableH := height * 60 / 100
	bentoH := height - tableH

	// Full-width PR table
	tablePanel := ui.RenderPanel(
		"", m.prTable.View(),
		m.focusedPanel == 0, m.width, tableH,
	)

	// Bottom bento: Detail | System | Jobs
	detailW, systemW, jobsW := m.bentoPanelWidths()

	detailPanel := ui.RenderPanel(
		"Details", m.detailView.View(),
		m.focusedPanel == 1, detailW, bentoH,
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

	return lipgloss.JoinVertical(lipgloss.Left, tablePanel, bentoRow)
}

func (m Model) viewReposOverlay() string {
	content := m.allRepoList.View()
	overlay := ui.RenderPanel("All Repos (press 2 or esc to close)", content, true, m.width-2, m.height-2)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
}
