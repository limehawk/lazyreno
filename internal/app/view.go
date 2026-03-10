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

	// Confirmation dialog
	if m.confirmMsg != "" {
		return view(ui.RenderConfirm(m.confirmMsg, m.width, m.height))
	}

	header := ui.RenderHeader(m.activeTab, m.width)

	// Flash message line (shown above help bar when active)
	var flashLine string
	if m.flashText != "" && time.Now().Before(m.flashExpiry) {
		style := ui.SuccessText
		if m.flashIsError {
			style = ui.ErrorText
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

	// Sidebar: repo list wrapped in a panel
	sidebar := ui.WrapListInPanel(
		fmt.Sprintf("Repos (%d open)", len(m.repoList.Items())),
		m.repoList.View(),
		m.focusedPanel == 0,
		sidebarWidth,
		height,
	)

	// Main: PR list wrapped in a panel
	// Add a footer with totals.
	prView := m.prList.View()
	totalPRs := len(m.prs)
	mergeableCount := 0
	for _, pr := range m.prs {
		if pr.Mergeable {
			mergeableCount++
		}
	}
	_ = totalPRs
	_ = mergeableCount

	main := ui.WrapListInPanel(
		"Pull Requests",
		prView,
		m.focusedPanel == 1,
		mainWidth,
		height,
	)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)

	if detailWidth > 0 {
		detail := ui.WrapListInPanel(
			"Details",
			m.detailView.View(),
			m.focusedPanel == 2,
			detailWidth,
			height,
		)
		panels = lipgloss.JoinHorizontal(lipgloss.Top, panels, detail)
	}

	return panels
}

func (m Model) viewRepos(height int) string {
	sidebarWidth := 28
	mainWidth := m.width - sidebarWidth
	if m.width < 80 {
		sidebarWidth = 22
		mainWidth = m.width - sidebarWidth
	}

	sidebar := ui.WrapListInPanel(
		fmt.Sprintf("Repos (%d)", len(m.repos)),
		m.allRepoList.View(),
		m.focusedPanel == 0,
		sidebarWidth,
		height,
	)

	// Main: repo info
	mainContent := ui.Dim.Render("Select a repo")
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

			prCountStr := ui.SuccessText.Render(fmt.Sprintf("%d", prCount))
			if prCount > 0 {
				prCountStr = ui.WarningText.Render(fmt.Sprintf("%d", prCount))
			}

			mainContent = fmt.Sprintf(
				"%s  %s\n%s  %s",
				ui.Dim.Render("Repository:"),
				ui.Bold.Render(fullName),
				ui.Dim.Render("Open PRs:  "),
				prCountStr,
			)
		}
	}

	main := ui.RenderPanel(
		"Repository Info",
		mainContent,
		m.focusedPanel == 1,
		mainWidth,
		height,
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
}

func (m Model) viewJobs(height int) string {
	sidebarWidth := 28
	mainWidth := m.width - sidebarWidth
	if m.width < 80 {
		sidebarWidth = 22
		mainWidth = m.width - sidebarWidth
	}

	sidebar := ui.WrapListInPanel(
		fmt.Sprintf("Queue (%d)", len(m.jobs)),
		m.jobList.View(),
		m.focusedPanel == 0,
		sidebarWidth,
		height,
	)

	mainContent := ui.Dim.Render("Select a job")
	sel := m.jobList.SelectedItem()
	if sel != nil {
		if ji, ok := sel.(JobItem); ok {
			job := ji.Job
			mainContent = fmt.Sprintf(
				"%s  %s\n%s  %s\n%s  %s\n\n%s  %s",
				ui.Dim.Render("Job:   "), job.ID,
				ui.Dim.Render("Repo:  "), ui.Bold.Render(job.Repo),
				ui.Dim.Render("Status:"), job.Status,
				ui.AccentText.Render("[r]"), "retry  "+ui.AccentText.Render("[p]")+" purge failed",
			)
		}
	}

	main := ui.RenderPanel(
		"Job Details",
		mainContent,
		m.focusedPanel == 1,
		mainWidth,
		height,
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
}

func (m Model) viewStatus(height int) string {
	return ui.WrapListInPanel(
		"System Status",
		m.statusView.View(),
		true,
		m.width,
		height,
	)
}
