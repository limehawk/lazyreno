package app

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	prsByRepo := m.groupPRsByRepo()
	repoOrder := m.getReposWithPRs(prsByRepo)

	// Dynamic widths
	sidebarWidth := 28
	detailWidth := 30
	mainWidth := m.width - sidebarWidth - detailWidth
	if m.width < 120 {
		detailWidth = 0
		mainWidth = m.width - sidebarWidth
	}
	if m.width < 80 {
		sidebarWidth = 22
		mainWidth = m.width - sidebarWidth
	}
	sidebarInner := sidebarWidth - 4 // borders + padding

	// Sidebar: repos with PR counts
	var sidebarLines []string
	for i, repo := range repoOrder {
		fullName := m.cfg.GitHub.Owner + "/" + repo
		prs := prsByRepo[fullName]
		prefix := "  "
		if i == m.sidebarCursor {
			prefix = ui.AccentText.Render("● ")
		}
		nameWidth := sidebarInner - 5 // space for " XX"
		line := fmt.Sprintf("%s%s %s",
			prefix,
			padRight(truncate(repo, nameWidth), nameWidth),
			ui.Dim.Render(fmt.Sprintf("%2d", len(prs))),
		)
		sidebarLines = append(sidebarLines, line)
	}

	sidebar := ui.Panel{
		Title:   fmt.Sprintf("Repos (%d open)", len(repoOrder)),
		Content: strings.Join(sidebarLines, "\n"),
		Focused: m.focusedPanel == 0,
		Width:   sidebarWidth,
		Height:  height,
	}

	// Main: PRs for selected repo
	mainInner := mainWidth - 4
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
			// Color the update type
			typeStr := ui.Dim.Render(updateType)
			if updateType == "major" {
				typeStr = ui.WarningText.Render(updateType)
			}
			titleWidth := mainInner - 18 // space for type + age
			if titleWidth < 20 {
				titleWidth = 20
			}
			line := fmt.Sprintf("%s%s  %s  %s",
				prefix,
				padRight(truncate(pr.Title, titleWidth), titleWidth),
				typeStr,
				ui.Dim.Render(fmt.Sprintf("%7s", age)),
			)
			mainLines = append(mainLines, line)
		}
	}

	// Footer line
	totalPRs := len(m.prs)
	mergeableCount := 0
	for _, pr := range m.prs {
		if pr.Mergeable {
			mergeableCount++
		}
	}
	footer := ui.Dim.Render(fmt.Sprintf(" %d PRs total  %d mergeable", totalPRs, mergeableCount))

	mainContent := strings.Join(mainLines, "\n")
	if len(mainLines) < height-3 && footer != "" {
		// Add footer at bottom
		padding := height - 3 - len(mainLines) - 1
		if padding > 0 {
			mainContent += strings.Repeat("\n", padding)
		}
		mainContent += "\n" + footer
	}

	main := ui.Panel{
		Title:   "Pull Requests",
		Content: mainContent,
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
				mergeStatus := ui.ErrorText.Render("✗ conflict")
				if pr.Mergeable {
					mergeStatus = ui.SuccessText.Render("✓ mergeable")
				}
				checkStatus := ui.ErrorText.Render("✗ failing")
				if pr.ChecksPass {
					checkStatus = ui.SuccessText.Render("✓ passing")
				}
				titleWidth := detailWidth - 4
				detailContent = fmt.Sprintf(
					"%s\n%s\n\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n\n%s  %s\n%s",
					ui.Bold.Render(fmt.Sprintf("#%d", pr.Number)),
					truncate(pr.Title, titleWidth),
					ui.Dim.Render("Branch:"), pr.Branch,
					ui.Dim.Render("Base:  "), pr.Base,
					ui.Dim.Render("Checks:"), checkStatus,
					ui.Dim.Render("Merge: "), mergeStatus,
					ui.Dim.Render("Age:   "), backend.RelativeTime(pr.CreatedAt),
					ui.Dim.Render("Type:  "), pr.UpdateType,
					ui.AccentText.Render("[m]"), "merge",
					ui.AccentText.Render("[c]")+" close  "+ui.AccentText.Render("[o]")+" open",
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
	sidebarWidth := 28
	mainWidth := m.width - sidebarWidth
	sidebarInner := sidebarWidth - 4

	var sidebarLines []string
	for i, repo := range m.repos {
		prefix := "  "
		if i == m.sidebarCursor {
			prefix = ui.AccentText.Render("● ")
		}
		sidebarLines = append(sidebarLines, fmt.Sprintf("%s%s", prefix, truncate(repo, sidebarInner-2)))
	}

	sidebar := ui.Panel{
		Title:   fmt.Sprintf("Repos (%d)", len(m.repos)),
		Content: strings.Join(sidebarLines, "\n"),
		Focused: m.focusedPanel == 0,
		Width:   sidebarWidth,
		Height:  height,
	}

	mainContent := ui.Dim.Render("Select a repo")
	if m.sidebarCursor < len(m.repos) {
		repo := m.repos[m.sidebarCursor]
		fullName := m.cfg.GitHub.Owner + "/" + repo

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
	sidebarWidth := 28
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
			fmt.Sprintf("%s%s %s  %s", prefix, statusIcon, padRight(truncate(repoShort, 14), 14), ui.Dim.Render(job.Status)))
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

	mainContent := ui.Dim.Render("Select a job")
	if m.sidebarCursor < len(m.jobs) {
		job := m.jobs[m.sidebarCursor]
		mainContent = fmt.Sprintf(
			"%s  %s\n%s  %s\n%s  %s\n\n%s  %s",
			ui.Dim.Render("Job:   "), job.ID,
			ui.Dim.Render("Repo:  "), ui.Bold.Render(job.Repo),
			ui.Dim.Render("Status:"), job.Status,
			ui.AccentText.Render("[r]"), "retry  "+ui.AccentText.Render("[p]")+" purge failed",
		)
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
			"",
			ui.Dim.Render("  Set LAZYRENO_RENOVATE_URL and LAZYRENO_RENOVATE_SECRET"),
		}
	} else if m.status == nil {
		lines = append(lines, "", ui.Dim.Render("  Connecting to Renovate CE..."))
	} else {
		s := m.status
		connected := ui.SuccessText.Render("✓ connected")
		uptime := s.Uptime.Truncate(time.Minute).String()

		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  %s %s          %s %s       %s %s",
			ui.Dim.Render("Renovate CE"), ui.Bold.Render(s.Version),
			ui.Dim.Render("API:"), connected,
			ui.Dim.Render("Uptime:"), uptime))
		lines = append(lines, fmt.Sprintf("  %s %d            %s %d",
			ui.Dim.Render("Jobs:"), s.QueueSize,
			ui.Dim.Render("Failed:"), s.FailedJobs))
		lines = append(lines, "")
		divWidth := m.width - 6
		if divWidth < 0 {
			divWidth = 0
		}
		lines = append(lines, ui.Dim.Render("  "+strings.Repeat("─", divWidth)))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  %s sync now   %s purge failed",
			ui.AccentText.Render("[s]"),
			ui.AccentText.Render("[p]")))
	}

	return ui.Panel{
		Title:   "System Status",
		Content: strings.Join(lines, "\n"),
		Focused: true,
		Width:   m.width,
		Height:  height,
	}.View()
}


func truncate(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}

func padRight(s string, width int) string {
	visWidth := lipgloss.Width(s)
	if visWidth >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visWidth)
}
