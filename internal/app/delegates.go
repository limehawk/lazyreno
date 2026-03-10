package app

import (
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/limehawk/lazyreno/internal/backend"
	"github.com/limehawk/lazyreno/internal/ui"

	"charm.land/bubbles/v2/list"
)

// --- Repo delegate (sidebar) ------------------------------------------------

type repoDelegate struct{}

func (d repoDelegate) Height() int                              { return 1 }
func (d repoDelegate) Spacing() int                             { return 0 }
func (d repoDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd  { return nil }
func (d repoDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ri, ok := item.(RepoItem)
	if !ok {
		return
	}
	if m.Width() <= 0 {
		return
	}

	selected := index == m.Index()
	prefix := "  "
	nameStyle := lipgloss.NewStyle()
	countStyle := ui.Dim
	if selected {
		prefix = ui.AccentText.Render("● ")
		nameStyle = ui.AccentText
	}

	countStr := fmt.Sprintf("%2d", ri.PRCount)
	nameWidth := m.Width() - 5 // "● " (2) + " XX" (3)
	if nameWidth < 4 {
		nameWidth = 4
	}
	name := ansi.Truncate(ri.Name, nameWidth, "…")

	fmt.Fprintf(w, "%s%-*s %s", prefix, nameWidth, nameStyle.Render(name), countStyle.Render(countStr))
}

// --- All-repo delegate (Repos tab sidebar) ----------------------------------

type allRepoDelegate struct{}

func (d allRepoDelegate) Height() int                              { return 1 }
func (d allRepoDelegate) Spacing() int                             { return 0 }
func (d allRepoDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd  { return nil }
func (d allRepoDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ri, ok := item.(AllRepoItem)
	if !ok {
		return
	}
	if m.Width() <= 0 {
		return
	}

	selected := index == m.Index()
	prefix := "  "
	style := lipgloss.NewStyle()
	if selected {
		prefix = ui.AccentText.Render("● ")
		style = ui.AccentText
	}

	nameWidth := m.Width() - 2
	if nameWidth < 4 {
		nameWidth = 4
	}
	name := ansi.Truncate(ri.Name, nameWidth, "…")
	fmt.Fprintf(w, "%s%s", prefix, style.Render(name))
}

// --- PR delegate (main panel) -----------------------------------------------

type prDelegate struct{}

func (d prDelegate) Height() int                              { return 1 }
func (d prDelegate) Spacing() int                             { return 0 }
func (d prDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd  { return nil }
func (d prDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	pi, ok := item.(PRItem)
	if !ok {
		return
	}
	if m.Width() <= 0 {
		return
	}

	selected := index == m.Index()
	prefix := "  "
	titleStyle := lipgloss.NewStyle()
	if selected {
		prefix = ui.AccentText.Render("● ")
		titleStyle = ui.AccentText
	}

	pr := pi.PR
	age := backend.RelativeTime(pr.CreatedAt)
	updateType := pr.UpdateType
	if updateType == "" {
		updateType = "dep"
	}

	typeStyle := ui.Dim
	if updateType == "major" {
		typeStyle = ui.WarningText
	}

	// Layout: "● title  type     age"
	typeWidth := 7
	ageWidth := 7
	metaWidth := typeWidth + 2 + ageWidth // "  " separator + fields
	titleWidth := m.Width() - 2 - metaWidth - 2 // prefix + meta + spacing
	if titleWidth < 10 {
		titleWidth = 10
	}

	title := ansi.Truncate(pr.Title, titleWidth, "…")
	fmt.Fprintf(w, "%s%-*s  %-*s  %*s",
		prefix,
		titleWidth, titleStyle.Render(title),
		typeWidth, typeStyle.Render(updateType),
		ageWidth, ui.Dim.Render(age),
	)
}

// --- Job delegate (Jobs tab sidebar) ----------------------------------------

type jobDelegate struct{}

func (d jobDelegate) Height() int                              { return 1 }
func (d jobDelegate) Spacing() int                             { return 0 }
func (d jobDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd  { return nil }
func (d jobDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ji, ok := item.(JobItem)
	if !ok {
		return
	}
	if m.Width() <= 0 {
		return
	}

	selected := index == m.Index()
	prefix := "  "
	if selected {
		prefix = ui.AccentText.Render("● ")
	}

	job := ji.Job
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

	repoShort := ji.Title()
	nameWidth := m.Width() - 12 // prefix + icon + space + status
	if nameWidth < 4 {
		nameWidth = 4
	}
	repoShort = ansi.Truncate(repoShort, nameWidth, "…")
	fmt.Fprintf(w, "%s%s %-*s %s", prefix, statusIcon, nameWidth, repoShort, ui.Dim.Render(job.Status))
}
