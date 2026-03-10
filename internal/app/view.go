package app

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/limehawk/lazyreno/internal/backend"
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

	// Help overlay
	if m.showHelp {
		return view(ui.RenderHelp(m.helpEntries(), m.width, m.height))
	}

	// Confirmation dialog
	if m.confirmMsg != "" {
		return view(ui.RenderConfirm(m.confirmMsg, m.width, m.height))
	}

	header := ui.RenderHeader(m.activeTab, m.width)

	// Status bar
	context := fmt.Sprintf("%d repos  %d PRs", len(m.repos), len(m.prs))
	keyHints := "? help  q quit  R refresh"
	statusBar := ui.RenderStatusBar(context, keyHints, m.flash, m.width)

	// Body height = total - header - status bar
	bodyHeight := m.height - lipgloss.Height(header) - lipgloss.Height(statusBar)
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

	return view(lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar))
}

func (m Model) viewPRs(height int) string {
	// Group PRs by repo
	prsByRepo := m.groupPRsByRepo()
	repoOrder := m.getReposWithPRs(prsByRepo)

	// Sidebar: repos with PR counts
	var sidebarLines []string
	for i, repo := range repoOrder {
		fullName := m.cfg.GitHub.Owner + "/" + repo
		prs := prsByRepo[fullName]
		prefix := "  "
		if i == m.sidebarCursor {
			prefix = ui.AccentText.Render("● ")
		}
		sidebarLines = append(sidebarLines, fmt.Sprintf("%s%-18s %d", prefix, truncate(repo, 18), len(prs)))
	}

	sidebarWidth := 26
	detailWidth := 28
	mainWidth := m.width - sidebarWidth - detailWidth
	if m.width < 100 {
		detailWidth = 0
		mainWidth = m.width - sidebarWidth
	}

	sidebar := ui.Panel{
		Title:   fmt.Sprintf("Repos (%d open)", len(repoOrder)),
		Content: strings.Join(sidebarLines, "\n"),
		Focused: m.focusedPanel == 0,
		Width:   sidebarWidth,
		Height:  height,
	}

	// Main: PRs for selected repo
	var mainLines []string
	selectedRepo := ""
	if m.sidebarCursor < len(repoOrder) {
		selectedRepo = repoOrder[m.sidebarCursor]
	}
	if selectedRepo != "" {
		fullName := m.cfg.GitHub.Owner + "/" + selectedRepo
		for i, pr := range prsByRepo[fullName] {
			prefix := "  "
			if i == m.mainCursor {
				prefix = ui.AccentText.Render("● ")
			}
			age := backend.RelativeTime(pr.CreatedAt)
			updateType := pr.UpdateType
			if updateType == "" {
				updateType = "dep"
			}
			mainLines = append(mainLines, fmt.Sprintf("%s%-30s %5s %5s", prefix, truncate(pr.Title, 30), updateType, age))
		}
	}

	main := ui.Panel{
		Title:   "Pull Requests",
		Content: strings.Join(mainLines, "\n"),
		Focused: m.focusedPanel == 1,
		Width:   mainWidth,
		Height:  height,
	}

	panels := lipgloss.JoinHorizontal(lipgloss.Top, sidebar.View(), main.View())

	if detailWidth > 0 {
		var detailContent string
		if selectedRepo != "" {
			fullName := m.cfg.GitHub.Owner + "/" + selectedRepo
			prs := prsByRepo[fullName]
			if m.mainCursor < len(prs) {
				pr := prs[m.mainCursor]
				mergeable := "✗ conflict"
				if pr.Mergeable {
					mergeable = "✓ mergeable"
				}
				checks := "✗ failing"
				if pr.ChecksPass {
					checks = "✓ passing"
				}
				detailContent = fmt.Sprintf(
					"#%d %s\n\nBranch: %s\nBase:   %s\nChecks: %s\nMerge:  %s\nAge:    %s\nType:   %s\n\n[m]erge [c]lose\n[o]pen in browser",
					pr.Number, truncate(pr.Title, 22), pr.Branch, pr.Base, checks, mergeable,
					backend.RelativeTime(pr.CreatedAt), pr.UpdateType,
				)
			}
		}

		detail := ui.Panel{
			Title:   "Details",
			Content: detailContent,
			Focused: m.focusedPanel == 2,
			Width:   detailWidth,
			Height:  height,
		}
		panels = lipgloss.JoinHorizontal(lipgloss.Top, panels, detail.View())
	}

	return panels
}

func (m Model) viewRepos(height int) string {
	sidebarWidth := 26
	mainWidth := m.width - sidebarWidth

	var sidebarLines []string
	for i, repo := range m.repos {
		prefix := "  "
		if i == m.sidebarCursor {
			prefix = ui.AccentText.Render("● ")
		}
		sidebarLines = append(sidebarLines, fmt.Sprintf("%s%s", prefix, truncate(repo, 22)))
	}

	sidebar := ui.Panel{
		Title:   "Repos",
		Content: strings.Join(sidebarLines, "\n"),
		Focused: m.focusedPanel == 0,
		Width:   sidebarWidth,
		Height:  height,
	}

	mainContent := "Select a repo"
	if m.sidebarCursor < len(m.repos) {
		repo := m.repos[m.sidebarCursor]
		fullName := m.cfg.GitHub.Owner + "/" + repo

		prCount := 0
		for _, pr := range m.prs {
			if pr.Repo == fullName {
				prCount++
			}
		}
		mainContent = fmt.Sprintf("Repository:  %s\nOpen PRs:    %d", fullName, prCount)
	}

	main := ui.Panel{
		Title:   "Repository Info",
		Content: mainContent,
		Focused: m.focusedPanel == 1,
		Width:   mainWidth,
		Height:  height,
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar.View(), main.View())
}

func (m Model) viewJobs(height int) string {
	sidebarWidth := 26
	mainWidth := m.width - sidebarWidth

	var sidebarLines []string
	for i, job := range m.jobs {
		prefix := "  "
		if i == m.sidebarCursor {
			prefix = ui.AccentText.Render("● ")
		}
		statusIcon := "◌"
		switch job.Status {
		case "running":
			statusIcon = ui.AccentText.Render("●")
		case "pending":
			statusIcon = ui.Dim.Render("◌")
		case "failed":
			statusIcon = ui.ErrorText.Render("✗")
		case "success":
			statusIcon = ui.SuccessText.Render("✓")
		}
		repoShort := job.Repo
		if parts := strings.SplitN(job.Repo, "/", 2); len(parts) == 2 {
			repoShort = parts[1]
		}
		sidebarLines = append(sidebarLines,
			fmt.Sprintf("%s%s %-14s %s", prefix, statusIcon, truncate(repoShort, 14), job.Status))
	}

	if len(m.jobs) == 0 {
		sidebarLines = append(sidebarLines, ui.Dim.Render("  No jobs in queue"))
	}

	sidebar := ui.Panel{
		Title:   fmt.Sprintf("Queue (%d)", len(m.jobs)),
		Content: strings.Join(sidebarLines, "\n"),
		Focused: m.focusedPanel == 0,
		Width:   sidebarWidth,
		Height:  height,
	}

	mainContent := "Select a job"
	if m.sidebarCursor < len(m.jobs) {
		job := m.jobs[m.sidebarCursor]
		mainContent = fmt.Sprintf("Job:    %s\nRepo:   %s\nStatus: %s\n\n[r]etry  [p]urge failed",
			job.ID, job.Repo, job.Status)
	}

	main := ui.Panel{
		Title:   "Job Details",
		Content: mainContent,
		Focused: m.focusedPanel == 1,
		Width:   mainWidth,
		Height:  height,
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar.View(), main.View())
}

func (m Model) viewStatus(height int) string {
	var lines []string

	if m.renovate == nil {
		lines = []string{
			"",
			ui.WarningText.Render("  Renovate CE not configured"),
			ui.Dim.Render("  Set LAZYRENO_RENOVATE_URL and LAZYRENO_RENOVATE_SECRET"),
		}
	} else if m.status == nil {
		lines = append(lines, ui.Dim.Render("  Connecting to Renovate CE..."))
	} else {
		s := m.status
		connected := ui.SuccessText.Render("✓ connected")
		uptime := s.Uptime.Truncate(time.Minute).String()

		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  Renovate CE %s          API: %s       Uptime: %s",
			ui.Bold.Render(s.Version), connected, uptime))
		lines = append(lines, fmt.Sprintf("  Jobs: %d queued            Failed: %d",
			s.QueueSize, s.FailedJobs))
		lines = append(lines, "")
		lines = append(lines, ui.Dim.Render("  "+strings.Repeat("─", max(m.width-6, 0))))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  %s  %s",
			ui.AccentText.Render("[s]ync now"),
			ui.AccentText.Render("[p]urge failed")))
	}

	return ui.Panel{
		Title:   "System Status",
		Content: strings.Join(lines, "\n"),
		Focused: true,
		Width:   m.width,
		Height:  height,
	}.View()
}

func (m Model) helpEntries() []ui.HelpEntry {
	return []ui.HelpEntry{
		{"1-4", "Switch tabs"},
		{"[ ]", "Prev/next tab"},
		{"Tab", "Cycle panel focus"},
		{"j/k", "Move up/down"},
		{"h/l", "Move left/right"},
		{"/", "Filter"},
		{"R", "Refresh"},
		{"m", "Merge PR"},
		{"M", "Merge safe PRs"},
		{"c", "Close PR"},
		{"o", "Open in browser"},
		{"s", "Trigger sync"},
		{"r", "Retry job"},
		{"p", "Purge failed"},
		{"?", "Toggle help"},
		{"q", "Quit"},
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}
