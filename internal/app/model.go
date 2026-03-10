package app

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/limehawk/lazyreno/internal/backend"
	"github.com/limehawk/lazyreno/internal/config"
	"github.com/limehawk/lazyreno/internal/ui"
)

const (
	TabPRs    = 0
	TabRepos  = 1
	TabJobs   = 2
	TabStatus = 3
)

type Model struct {
	cfg      *config.Config
	renovate *backend.RenovateClient
	github   *backend.GitHubClient
	cache    *backend.Cache

	activeTab int
	width     int
	height    int

	// Shared data
	repos  []string
	prs    []backend.PR
	jobs   []backend.Job
	status *backend.SystemStatus

	// UI state
	keys         KeyMap
	help         help.Model
	confirmMsg   string
	confirmFn    func() tea.Cmd
	flashText    string
	flashIsError bool
	flashExpiry  time.Time
	focusedPanel int // 0=sidebar, 1=main, 2=detail

	// Bubble lists
	repoList    list.Model // PRs tab sidebar
	prList      list.Model // PRs tab main panel
	allRepoList list.Model // Repos tab sidebar
	jobList     list.Model // Jobs tab sidebar

	// Detail / status viewport
	detailView viewport.Model
	statusView viewport.Model

	err error
}

func newList(delegate list.ItemDelegate, title string) list.Model {
	l := list.New(nil, delegate, 0, 0)
	l.Title = title
	l.DisableQuitKeybindings()
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)
	l.SetShowTitle(false)
	// Remove built-in key handling for j/k so we stay consistent.
	// The list bubble uses its own j/k by default which is what we want.
	return l
}

func NewModel(cfg *config.Config) Model {
	var renovate *backend.RenovateClient
	if cfg.Renovate.URL != "" && cfg.Renovate.Secret != "" {
		renovate = backend.NewRenovateClient(cfg.Renovate.URL, cfg.Renovate.Secret)
	}

	var gh *backend.GitHubClient
	if cfg.GitHub.Token != "" {
		gh = backend.NewGitHubClient(cfg.GitHub.Token)
	}

	h := help.New()
	s := help.DefaultDarkStyles()
	s.ShortKey = lipgloss.NewStyle().Foreground(ui.Accent).Bold(true)
	s.FullKey = lipgloss.NewStyle().Foreground(ui.Accent).Bold(true)
	h.Styles = s

	return Model{
		cfg:         cfg,
		renovate:    renovate,
		github:      gh,
		cache:       backend.NewCache(30 * time.Second),
		keys:        GlobalKeys,
		help:        h,
		repoList:    newList(repoDelegate{}, "Repos"),
		prList:      newList(prDelegate{}, "Pull Requests"),
		allRepoList: newList(allRepoDelegate{}, "Repos"),
		jobList:     newList(jobDelegate{}, "Queue"),
		detailView:  viewport.New(),
		statusView:  viewport.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchRepos(),
		m.fetchStatus(),
		m.fetchJobQueue(),
		m.tickCmd(),
	)
}

func (m Model) getSelectedPR() *backend.PR {
	sel := m.prList.SelectedItem()
	if sel == nil {
		return nil
	}
	if pi, ok := sel.(PRItem); ok {
		pr := pi.PR
		return &pr
	}
	return nil
}

func (m Model) getSafePRsForSelectedRepo() []backend.PR {
	// Get the currently selected repo from sidebar.
	sel := m.repoList.SelectedItem()
	if sel == nil {
		return nil
	}
	ri, ok := sel.(RepoItem)
	if !ok {
		return nil
	}
	fullName := m.cfg.GitHub.Owner + "/" + ri.Name

	var safe []backend.PR
	for _, pr := range m.prs {
		if pr.Repo == fullName && backend.IsSafeToMerge(pr) {
			safe = append(safe, pr)
		}
	}
	return safe
}

func (m Model) groupPRsByRepo() map[string][]backend.PR {
	prsByRepo := make(map[string][]backend.PR)
	for _, pr := range m.prs {
		prsByRepo[pr.Repo] = append(prsByRepo[pr.Repo], pr)
	}
	return prsByRepo
}

func (m Model) getReposWithPRs(prsByRepo map[string][]backend.PR) []string {
	var repoOrder []string
	for _, repo := range m.repos {
		fullName := m.cfg.GitHub.Owner + "/" + repo
		if prs, ok := prsByRepo[fullName]; ok && len(prs) > 0 {
			repoOrder = append(repoOrder, repo)
		}
	}
	return repoOrder
}

func splitRepo(fullName string) (string, string) {
	parts := strings.SplitN(fullName, "/", 2)
	return parts[0], parts[1]
}

func removePR(prs []backend.PR, repo string, number int) []backend.PR {
	result := make([]backend.PR, 0, len(prs))
	for _, pr := range prs {
		if !(pr.Repo == repo && pr.Number == number) {
			result = append(result, pr)
		}
	}
	return result
}

// rebuildRepoList rebuilds the sidebar repo list from current data.
func (m *Model) rebuildRepoList() tea.Cmd {
	prsByRepo := m.groupPRsByRepo()
	repoOrder := m.getReposWithPRs(prsByRepo)

	items := make([]list.Item, len(repoOrder))
	for i, repo := range repoOrder {
		fullName := m.cfg.GitHub.Owner + "/" + repo
		items[i] = RepoItem{Name: repo, PRCount: len(prsByRepo[fullName])}
	}
	return m.repoList.SetItems(items)
}

// rebuildPRList rebuilds the PR list for the currently selected repo.
func (m *Model) rebuildPRList() tea.Cmd {
	sel := m.repoList.SelectedItem()
	if sel == nil {
		return m.prList.SetItems(nil)
	}
	ri, ok := sel.(RepoItem)
	if !ok {
		return m.prList.SetItems(nil)
	}
	fullName := m.cfg.GitHub.Owner + "/" + ri.Name

	var items []list.Item
	for _, pr := range m.prs {
		if pr.Repo == fullName {
			items = append(items, PRItem{PR: pr})
		}
	}
	return m.prList.SetItems(items)
}

// rebuildAllRepoList rebuilds the Repos tab sidebar.
func (m *Model) rebuildAllRepoList() tea.Cmd {
	items := make([]list.Item, len(m.repos))
	for i, repo := range m.repos {
		items[i] = AllRepoItem{Name: repo}
	}
	return m.allRepoList.SetItems(items)
}

// rebuildJobList rebuilds the Jobs tab list.
func (m *Model) rebuildJobList() tea.Cmd {
	items := make([]list.Item, len(m.jobs))
	for i, job := range m.jobs {
		items[i] = JobItem{Job: job}
	}
	return m.jobList.SetItems(items)
}

// updateDetailView updates the detail viewport content for the selected PR.
func (m *Model) updateDetailView() {
	pr := m.getSelectedPR()
	if pr == nil {
		m.detailView.SetContent(ui.Dim.Render("No PR selected"))
		return
	}

	mergeStatus := ui.ErrorText.Render("✗ conflict")
	if pr.Mergeable {
		mergeStatus = ui.SuccessText.Render("✓ mergeable")
	}
	checkStatus := ui.ErrorText.Render("✗ failing")
	if pr.ChecksPass {
		checkStatus = ui.SuccessText.Render("✓ passing")
	}

	content := fmt.Sprintf("%s\n\n%s\n\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n\n%s merge  %s close\n%s open in browser",
		ui.Bold.Render(fmt.Sprintf("#%d", pr.Number)),
		pr.Title,
		ui.Dim.Render("Branch:"), pr.Branch,
		ui.Dim.Render("Base:  "), pr.Base,
		ui.Dim.Render("Checks:"), checkStatus,
		ui.Dim.Render("Merge: "), mergeStatus,
		ui.Dim.Render("Age:   "), backend.RelativeTime(pr.CreatedAt),
		ui.Dim.Render("Type:  "), pr.UpdateType,
		ui.AccentText.Render("[m]"), ui.AccentText.Render("[c]"),
		ui.AccentText.Render("[o]"),
	)

	m.detailView.SetContent(content)
}

// updateStatusView updates the status viewport content.
func (m *Model) updateStatusView() {
	var lines []string

	if m.renovate == nil {
		lines = []string{
			"",
			ui.WarningText.Render("  Renovate CE not configured"),
			"",
			ui.Dim.Render("  Set LAZYRENO_RENOVATE_URL and LAZYRENO_RENOVATE_SECRET"),
		}
	} else if m.status == nil {
		lines = []string{"", ui.Dim.Render("  Connecting to Renovate CE...")}
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
		lines = append(lines, "  "+ui.AccentText.Render("[s]")+" sync now   "+
			ui.AccentText.Render("[p]")+" purge failed")
	}

	m.statusView.SetContent(strings.Join(lines, "\n"))
}

// resizeLists updates all list and viewport dimensions based on terminal size.
func (m *Model) resizeLists() {
	bodyHeight := m.bodyHeight()
	if bodyHeight < 1 {
		return
	}

	sidebarWidth, mainWidth, detailWidth := m.panelWidths()

	sidebarInnerW, sidebarInnerH := ui.InnerSize(sidebarWidth, bodyHeight)
	mainInnerW, mainInnerH := ui.InnerSize(mainWidth, bodyHeight)

	m.repoList.SetSize(sidebarInnerW, sidebarInnerH)
	m.prList.SetSize(mainInnerW, mainInnerH)
	m.allRepoList.SetSize(sidebarInnerW, sidebarInnerH)
	m.jobList.SetSize(sidebarInnerW, sidebarInnerH)

	if detailWidth > 0 {
		detailInnerW, detailInnerH := ui.InnerSize(detailWidth, bodyHeight)
		m.detailView.SetWidth(detailInnerW)
		m.detailView.SetHeight(detailInnerH)
	}

	statusInnerW, statusInnerH := ui.InnerSize(m.width, bodyHeight)
	m.statusView.SetWidth(statusInnerW)
	m.statusView.SetHeight(statusInnerH)
}

func (m Model) bodyHeight() int {
	header := ui.RenderHeader(m.activeTab, m.width)
	helpBar := m.help.View(m.keys)

	var bottomLines []string
	if m.flashText != "" && time.Now().Before(m.flashExpiry) {
		bottomLines = append(bottomLines, m.flashText)
	}
	bottomLines = append(bottomLines, helpBar)
	bottom := lipgloss.JoinVertical(lipgloss.Left, bottomLines...)

	h := m.height - lipgloss.Height(header) - lipgloss.Height(bottom)
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) panelWidths() (sidebar, main, detail int) {
	sidebar = 28
	detail = 30
	main = m.width - sidebar - detail
	if m.width < 120 {
		detail = 0
		main = m.width - sidebar
	}
	if m.width < 80 {
		sidebar = 22
		main = m.width - sidebar
	}
	if main < 10 {
		main = 10
	}
	return
}
