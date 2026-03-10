package app

import (
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/limehawk/lazyreno/internal/backend"
	"github.com/limehawk/lazyreno/internal/ui"

	"charm.land/bubbles/v2/list"
)

// --- Repo delegate (sidebar) ------------------------------------------------

type repoDelegate struct {
	theme *ui.Theme
}

func (d repoDelegate) Height() int                              { return 1 }
func (d repoDelegate) Spacing() int                             { return 0 }
func (d repoDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd  { return nil }
func (d repoDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ri, ok := item.(RepoItem)
	if !ok {
		return
	}
	width := m.Width()
	if width <= 0 {
		return
	}

	selected := index == m.Index()

	countStr := fmt.Sprintf("%2d", ri.PRCount)
	nameWidth := width - 5 // prefix(2) + space(1) + count(2)
	if nameWidth < 4 {
		nameWidth = 4
	}
	name := ansi.Truncate(ri.Name, nameWidth, "…")
	padded := fmt.Sprintf("%-*s", nameWidth, name) // pad plain text first

	if selected {
		fmt.Fprintf(w, "%s%s %s",
			d.theme.AccentText.Render("> "),
			d.theme.AccentText.Render(padded),
			d.theme.Dim.Render(countStr))
	} else {
		fmt.Fprintf(w, "  %s %s", padded, d.theme.Dim.Render(countStr))
	}
}

// --- All-repo delegate (Repos tab sidebar) ----------------------------------

type allRepoDelegate struct {
	theme *ui.Theme
}

func (d allRepoDelegate) Height() int                              { return 1 }
func (d allRepoDelegate) Spacing() int                             { return 0 }
func (d allRepoDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd  { return nil }
func (d allRepoDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ri, ok := item.(AllRepoItem)
	if !ok {
		return
	}
	width := m.Width()
	if width <= 0 {
		return
	}

	selected := index == m.Index()

	nameWidth := width - 2 // prefix(2)
	if nameWidth < 4 {
		nameWidth = 4
	}
	name := ansi.Truncate(ri.Name, nameWidth, "…")

	if selected {
		fmt.Fprintf(w, "%s%s",
			d.theme.AccentText.Render("> "),
			d.theme.AccentText.Render(name))
	} else {
		fmt.Fprintf(w, "  %s", name)
	}
}

// --- PR delegate (main panel) -----------------------------------------------

type prDelegate struct {
	theme *ui.Theme
}

func (d prDelegate) Height() int                              { return 1 }
func (d prDelegate) Spacing() int                             { return 0 }
func (d prDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd  { return nil }
func (d prDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	pi, ok := item.(PRItem)
	if !ok {
		return
	}
	width := m.Width()
	if width <= 0 {
		return
	}

	selected := index == m.Index()

	pr := pi.PR
	age := backend.RelativeTime(pr.CreatedAt)
	updateType := pr.UpdateType
	if updateType == "" {
		updateType = "dep"
	}

	typeStyle := d.theme.Dim
	if updateType == "major" {
		typeStyle = d.theme.WarningText
	}

	// Layout: "● title  type     age"
	typeWidth := 7
	ageWidth := 7
	// prefix(2) + title + "  "(2) + type(7) + "  "(2) + age(7) = 20 overhead
	titleWidth := width - 20
	if titleWidth < 10 {
		titleWidth = 10
	}

	title := ansi.Truncate(pr.Title, titleWidth, "…")
	// Pad plain text before styling to avoid ANSI width miscalculation
	paddedTitle := fmt.Sprintf("%-*s", titleWidth, title)
	paddedType := fmt.Sprintf("%-*s", typeWidth, updateType)
	paddedAge := fmt.Sprintf("%*s", ageWidth, age)

	if selected {
		fmt.Fprintf(w, "%s%s  %s  %s",
			d.theme.AccentText.Render("> "),
			d.theme.AccentText.Render(paddedTitle),
			typeStyle.Render(paddedType),
			d.theme.Dim.Render(paddedAge))
	} else {
		fmt.Fprintf(w, "  %s  %s  %s",
			paddedTitle,
			typeStyle.Render(paddedType),
			d.theme.Dim.Render(paddedAge))
	}
}

// --- Job delegate (Jobs tab sidebar) ----------------------------------------

type jobDelegate struct {
	theme *ui.Theme
}

func (d jobDelegate) Height() int                              { return 1 }
func (d jobDelegate) Spacing() int                             { return 0 }
func (d jobDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd  { return nil }
func (d jobDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ji, ok := item.(JobItem)
	if !ok {
		return
	}
	width := m.Width()
	if width <= 0 {
		return
	}

	selected := index == m.Index()

	job := ji.Job
	statusLabel := job.Status
	// prefix(2) + icon(1) + space(1) + name + space(1) + status(~8) = overhead ~13
	statusWidth := len(statusLabel)
	nameWidth := width - 5 - statusWidth // prefix(2) + icon(1) + 2 spaces
	if nameWidth < 4 {
		nameWidth = 4
	}

	repoShort := ansi.Truncate(ji.Title(), nameWidth, "…")
	paddedName := fmt.Sprintf("%-*s", nameWidth, repoShort)

	statusIcon := "◌"
	switch job.Status {
	case "running":
		statusIcon = d.theme.AccentText.Render("●")
	case "pending":
		statusIcon = d.theme.Dim.Render("◌")
	case "failed":
		statusIcon = d.theme.ErrorText.Render("✗")
	case "success":
		statusIcon = d.theme.SuccessText.Render("✓")
	}

	if selected {
		fmt.Fprintf(w, "%s%s %s %s",
			d.theme.AccentText.Render("> "),
			statusIcon,
			d.theme.AccentText.Render(paddedName),
			d.theme.Dim.Render(statusLabel))
	} else {
		fmt.Fprintf(w, "  %s %s %s",
			statusIcon,
			paddedName,
			d.theme.Dim.Render(statusLabel))
	}
}
