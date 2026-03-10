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

	// Confirmation form
	if m.confirmForm != nil {
		return view(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.confirmForm.View()))
	}

	header := ui.RenderHeader(&m.theme, m.activeTab, m.width)

	// Flash message line (shown above help bar when active)
	var flashLine string
	if m.flashText != "" && time.Now().Before(m.flashExpiry) {
		style := m.theme.SuccessText
		if m.flashIsError {
			style = m.theme.ErrorText
		}
		flashLine = style.Render(m.flashText)
	}

	// Help bar via the help bubble
	helpBar := m.help.View(m.keys)

	// Build the bottom section: optional flash + help bar
	var bottomLines []string
	if flashLine != "" {
		bottomLines = append(bottomLines, flashLine)
	}
	bottomLines = append(bottomLines, helpBar)
	bottom := lipgloss.JoinVertical(lipgloss.Left, bottomLines...)

	// Body height = total - header - bottom
	bodyHeight := m.height - lipgloss.Height(header) - lipgloss.Height(bottom)
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	var body string
	switch m.activeTab {
	case TabPRs:
		body = m.viewPRs(bodyHeight)
	case TabRepos:
		body = m.viewRepos(bodyHeight)
	case TabJobs:
		body = m.viewJobs(bodyHeight)
	case TabStatus:
		body = m.viewStatus(bodyHeight)
	}

	return view(lipgloss.JoinVertical(lipgloss.Left, header, body, bottom))
}

func (m Model) viewPRs(height int) string {
	sidebarWidth, mainWidth, detailWidth := m.panelWidths()

	// Sidebar: repo list wrapped in a panel (list has its own title).
	sidebar := ui.WrapListInPanel(
		&m.theme, "", m.repoList.View(),
		m.focusedPanel == 0, sidebarWidth, height,
	)

	// Main: PR list wrapped in a panel.
	main := ui.WrapListInPanel(
		&m.theme, "", m.prList.View(),
		m.focusedPanel == 1, mainWidth, height,
	)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)

	if detailWidth > 0 {
		detail := ui.RenderPanel(
			&m.theme, "Details", m.detailView.View(),
			m.focusedPanel == 2, detailWidth, height,
		)
		panels = lipgloss.JoinHorizontal(lipgloss.Top, panels, detail)
	}

	return panels
}

func (m Model) viewRepos(height int) string {
	sidebarWidth := m.width * 25 / 100
	if sidebarWidth < 22 {
		sidebarWidth = 22
	}
	if sidebarWidth > 40 {
		sidebarWidth = 40
	}
	mainWidth := m.width - sidebarWidth

	sidebar := ui.WrapListInPanel(
		&m.theme, "", m.allRepoList.View(),
		m.focusedPanel == 0, sidebarWidth, height,
	)

	// Main: repo info
	mainContent := m.theme.Dim.Render("Select a repo")
	sel := m.allRepoList.SelectedItem()
	if sel != nil {
		if ri, ok := sel.(AllRepoItem); ok {
			fullName := m.cfg.GitHub.Owner + "/" + ri.Name

			prCount := 0
			for _, pr := range m.prs {
				if pr.Repo == fullName {
					prCount++
				}
			}

			prCountStr := m.theme.SuccessText.Render(fmt.Sprintf("%d", prCount))
			if prCount > 0 {
				prCountStr = m.theme.WarningText.Render(fmt.Sprintf("%d", prCount))
			}

			mainContent = fmt.Sprintf(
				"%s  %s\n%s  %s",
				m.theme.Dim.Render("Repository:"),
				m.theme.Bold.Render(fullName),
				m.theme.Dim.Render("Open PRs:  "),
				prCountStr,
			)
		}
	}

	main := ui.RenderPanel(
		&m.theme,
		"Repository Info",
		mainContent,
		m.focusedPanel == 1,
		mainWidth,
		height,
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
}

func (m Model) viewJobs(height int) string {
	sidebarWidth := m.width * 25 / 100
	if sidebarWidth < 22 {
		sidebarWidth = 22
	}
	if sidebarWidth > 40 {
		sidebarWidth = 40
	}
	mainWidth := m.width - sidebarWidth

	sidebar := ui.WrapListInPanel(
		&m.theme, "", m.jobList.View(),
		m.focusedPanel == 0, sidebarWidth, height,
	)

	mainContent := m.theme.Dim.Render("Select a job")
	sel := m.jobList.SelectedItem()
	if sel != nil {
		if ji, ok := sel.(JobItem); ok {
			job := ji.Job
			mainContent = fmt.Sprintf(
				"%s  %s\n%s  %s\n%s  %s\n\n%s  %s",
				m.theme.Dim.Render("Job:   "), job.ID,
				m.theme.Dim.Render("Repo:  "), m.theme.Bold.Render(job.Repo),
				m.theme.Dim.Render("Status:"), job.Status,
				m.theme.AccentText.Render("[r]"), "retry  "+m.theme.AccentText.Render("[p]")+" purge failed",
			)
		}
	}

	main := ui.RenderPanel(
		&m.theme,
		"Job Details",
		mainContent,
		m.focusedPanel == 1,
		mainWidth,
		height,
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
}

func (m Model) viewStatus(height int) string {
	return ui.RenderPanel(
		&m.theme, "System Status", m.statusView.View(),
		true, m.width, height,
	)
}
