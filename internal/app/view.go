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

	if m.confirmForm != nil {
		return view(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.confirmForm.View()))
	}

	header := ui.RenderHeader(m.activeTab, m.width)

	var flashLine string
	if m.flashText != "" && time.Now().Before(m.flashExpiry) {
		style := ui.SuccessText
		if m.flashIsError {
			style = ui.ErrorText
		}
		flashLine = style.Render(m.flashText)
	}

	helpBar := m.help.View(m.keys)

	var bottomLines []string
	if flashLine != "" {
		bottomLines = append(bottomLines, flashLine)
	}
	bottomLines = append(bottomLines, helpBar)
	bottom := lipgloss.JoinVertical(lipgloss.Left, bottomLines...)

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

	sidebar := ui.WrapListInPanel(
		m.repoList.View(),
		m.focusedPanel == 0, sidebarWidth, height,
	)

	main := ui.RenderPanel(
		"", m.prTable.View(),
		m.focusedPanel == 1, mainWidth, height,
	)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)

	if detailWidth > 0 {
		detail := ui.RenderPanel(
			"Details", m.detailView.View(),
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
		m.allRepoList.View(),
		m.focusedPanel == 0, sidebarWidth, height,
	)

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
	sidebarWidth := m.width * 25 / 100
	if sidebarWidth < 22 {
		sidebarWidth = 22
	}
	if sidebarWidth > 40 {
		sidebarWidth = 40
	}
	mainWidth := m.width - sidebarWidth

	sidebar := ui.WrapListInPanel(
		m.jobList.View(),
		m.focusedPanel == 0, sidebarWidth, height,
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
				ui.ShortcutKey.Render("[r]"), "retry  "+ui.ShortcutKey.Render("[p]")+" purge failed",
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
	return ui.RenderPanel(
		"System Status", m.statusView.View(),
		true, m.width, height,
	)
}
