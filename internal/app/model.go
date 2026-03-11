package app

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/limehawk/lazyreno/internal/backend"
	"github.com/limehawk/lazyreno/internal/config"
	"github.com/limehawk/lazyreno/internal/ui"
)

type Model struct {
	cfg      *config.Config
	renovate *backend.RenovateClient
	github   *backend.GitHubClient
	cache    *backend.Cache

	width  int
	height int

	// Shared data
	repos          []string
	reposWithPRs   []string // short names of repos that have open PRs
	prs            []backend.PR
	pendingPRs     []backend.PR
	pendingPRCount int
	status         *backend.SystemStatus
	jobs           []backend.Job

	// Selected repo (cycles with [ / ])
	selectedRepoIdx int

	// Filtered PRs for the currently selected repo.
	filteredPRs []backend.PR

	// UI state
	keys           KeyMap
	help           help.Model
	spinner        spinner.Model
	lastUpdate     time.Time
	showRepos      bool // overlay toggle
	confirmText    string
	confirmAction  func() tea.Cmd
	flashText      string
	flashIsError   bool
	flashExpiry    time.Time
	focusedPanel   int // 0=table, 1=detail

	// All repos overlay
	allRepoList list.Model

	// PR table (full width)
	prTable table.Model

	// Detail viewport (bottom-left bento panel)
	detailView viewport.Model

	err error
}

// newSidebarList creates a compact single-line list with default delegate.
func newSidebarList(title string) list.Model {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.SetSpacing(0)

	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(ui.Accent).
		Foreground(ui.Accent).
		Padding(0, 0, 0, 1)
	d.Styles.NormalTitle = lipgloss.NewStyle().
		Padding(0, 0, 0, 2)

	l := list.New(nil, d, 0, 0)
	l.Title = title
	l.DisableQuitKeybindings()
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)
	l.SetShowTitle(true)

	styles := l.Styles
	styles.TitleBar = lipgloss.NewStyle()
	styles.Title = lipgloss.NewStyle().Foreground(ui.Title).Bold(true)
	l.Styles = styles

	return l
}

// newPRTable creates the table for the PR list.
func newPRTable() table.Model {
	cols := []table.Column{
		{Title: "", Width: 1},
		{Title: "Title", Width: 40},
		{Title: "Type", Width: 7},
		{Title: "Age", Width: 8},
	}

	return table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithStyles(prTableStyles()),
	)
}

// prTableStyles builds table.Styles from theme colors.
func prTableStyles() table.Styles {
	return table.Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(ui.Accent).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ui.Border).
			BorderBottom(true),
		Cell: lipgloss.NewStyle().
			Padding(0, 1),
		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(ui.SelectedFG).
			Background(ui.SelectedBG).
			Padding(0, 1),
	}
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
	hs := help.DefaultDarkStyles()
	hs.ShortKey = ui.ShortcutKey
	hs.FullKey = ui.ShortcutKey
	hs.ShortDesc = ui.Dim
	hs.ShortSeparator = lipgloss.NewStyle().Foreground(ui.Border)
	h.Styles = hs

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(ui.Accent)

	return Model{
		cfg:         cfg,
		renovate:    renovate,
		github:      gh,
		cache:       backend.NewCache(30 * time.Second),
		keys:        GlobalKeys,
		help:        h,
		spinner:     sp,
		allRepoList: newSidebarList("Repos"),
		prTable:     newPRTable(),
		detailView:  viewport.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchRepos(),
		m.fetchStatus(),
		m.fetchJobQueue(),
		m.tickCmd(),
		m.spinner.Tick,
	)
}

// selectedRepo returns the short name of the currently selected repo, or "".
func (m Model) selectedRepo() string {
	if len(m.reposWithPRs) == 0 {
		return ""
	}
	if m.selectedRepoIdx >= len(m.reposWithPRs) {
		return m.reposWithPRs[0]
	}
	return m.reposWithPRs[m.selectedRepoIdx]
}

func (m Model) selectedRepoFull() string {
	repo := m.selectedRepo()
	if repo == "" {
		return ""
	}
	return m.cfg.GitHub.Owner + "/" + repo
}

func (m Model) getSelectedPR() *backend.PR {
	idx := m.prTable.Cursor()
	if idx < 0 || idx >= len(m.filteredPRs) {
		return nil
	}
	pr := m.filteredPRs[idx]
	return &pr
}

func (m Model) getSafePRsForSelectedRepo() []backend.PR {
	fullName := m.selectedRepoFull()
	if fullName == "" {
		return nil
	}
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

func (m *Model) rebuildRepoList() {
	prsByRepo := m.groupPRsByRepo()
	m.reposWithPRs = m.getReposWithPRs(prsByRepo)
	if m.selectedRepoIdx >= len(m.reposWithPRs) && len(m.reposWithPRs) > 0 {
		m.selectedRepoIdx = len(m.reposWithPRs) - 1
	}
}

func (m *Model) rebuildPRTable() {
	fullName := m.selectedRepoFull()
	if fullName == "" {
		m.filteredPRs = nil
		m.prTable.SetRows(nil)
		return
	}

	prevCursor := m.prTable.Cursor()
	m.filteredPRs = nil
	for _, pr := range m.prs {
		if pr.Repo == fullName {
			m.filteredPRs = append(m.filteredPRs, pr)
		}
	}

	rows := make([]table.Row, len(m.filteredPRs))
	for i, pr := range m.filteredPRs {
		rows[i] = prToRow(pr)
	}
	m.prTable.SetRows(rows)
	if prevCursor < len(rows) {
		m.prTable.SetCursor(prevCursor)
	}
}

func prToRow(pr backend.PR) table.Row {
	dot := ui.PRStatusDot(pr.ChecksPass, pr.Mergeable)

	title := pr.Title
	if backend.IsSafeToMerge(pr) {
		title = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).Render(title)
	}

	updateType := pr.UpdateType
	if updateType == "" {
		updateType = "dep"
	}
	updateType = ui.PRTypeStyle(updateType)

	age := backend.RelativeTime(pr.CreatedAt)
	ageStyle := lipgloss.NewStyle().Foreground(ui.PRAgeForeground(pr.CreatedAt))
	age = ageStyle.Render(age)

	return table.Row{dot, title, updateType, age}
}

func (m *Model) rebuildAllRepoList() tea.Cmd {
	prevIdx := m.allRepoList.Index()
	items := make([]list.Item, len(m.repos))
	for i, repo := range m.repos {
		items[i] = AllRepoItem{Name: repo}
	}
	m.allRepoList.Title = fmt.Sprintf("Repos (%d)", len(items))
	cmd := m.allRepoList.SetItems(items)
	if prevIdx < len(items) {
		m.allRepoList.Select(prevIdx)
	}
	return cmd
}

func (m *Model) updateDetailView() {
	pr := m.getSelectedPR()
	if pr == nil {
		m.detailView.SetContent(ui.Dim.Render("No PR selected"))
		return
	}

	checkDot := ui.PRStatusDot(pr.ChecksPass, true)
	checkLabel := "passing"
	if !pr.ChecksPass {
		checkLabel = "failing"
	}
	mergeDot := ui.PRStatusDot(true, pr.Mergeable)
	mergeLabel := "mergeable"
	if !pr.Mergeable {
		mergeDot = ui.PRStatusDot(true, false)
		mergeLabel = "conflict"
	}

	content := fmt.Sprintf("%s  %s\n%s → %s\n%s %s  %s %s\n%s  %s\n\n%s merge  %s close  %s open",
		ui.Bold.Render(fmt.Sprintf("#%d", pr.Number)),
		pr.Title,
		ui.Dim.Render(pr.Branch), pr.Base,
		checkDot, checkLabel, mergeDot, mergeLabel,
		ui.PRTypeStyle(pr.UpdateType), backend.RelativeTime(pr.CreatedAt),
		ui.ShortcutKey.Render("[m]"), ui.ShortcutKey.Render("[c]"),
		ui.ShortcutKey.Render("[o]"),
	)

	m.detailView.SetContent(content)
}

func (m Model) renderStatusBox() string {
	if m.renovate == nil {
		return ui.WarningText.Render("Not configured")
	}
	if m.status == nil {
		return ui.Dim.Render("Connecting...")
	}

	s := m.status
	uptime := s.Uptime.Truncate(time.Minute).String()

	return fmt.Sprintf("%s %s\n%s %s\n%s %d",
		ui.Dim.Render("CE"), ui.Bold.Render("v"+s.Version),
		ui.Dim.Render("Up"), uptime,
		ui.Dim.Render("Q"), s.QueueSize)
}

func (m Model) renderJobsPanel() string {
	if len(m.jobs) == 0 && (m.status == nil || m.status.LastFinished == nil) {
		return ui.Dim.Render("No jobs")
	}

	var lines []string
	for _, job := range m.jobs {
		_, repo := splitRepo(job.Repo)
		dot := ui.JobStatusDot(job.Status)
		lines = append(lines, fmt.Sprintf("%s %s  %s", dot, repo, ui.Dim.Render(job.Status)))
	}

	// Append last-finished job if not already in queue.
	if m.status != nil && m.status.LastFinished != nil {
		lf := m.status.LastFinished
		found := false
		for _, j := range m.jobs {
			if j.ID == lf.ID {
				found = true
				break
			}
		}
		if !found {
			_, repo := splitRepo(lf.Repo)
			dot := ui.JobStatusDot(lf.Status)
			dur := ""
			if lf.Duration > 0 {
				dur = "  " + lf.Duration.Truncate(time.Second).String()
			}
			lines = append(lines, fmt.Sprintf("%s %s  %s%s", dot, repo, ui.Dim.Render(lf.Status), dur))
		}
	}

	return strings.Join(lines, "\n")
}

// repoInfo returns a styled string for the status bar: "▸ repo-name (N)"
func (m Model) repoInfo() string {
	repo := m.selectedRepo()
	if repo == "" {
		if len(m.repos) == 0 {
			return ""
		}
		return ui.Dim.Render("no PRs")
	}

	prCount := 0
	fullName := m.selectedRepoFull()
	for _, pr := range m.prs {
		if pr.Repo == fullName {
			prCount++
		}
	}

	return fmt.Sprintf("%s %s %s",
		ui.AccentText.Render("▸"),
		ui.Bold.Render(repo),
		ui.Dim.Render(fmt.Sprintf("(%d)", prCount)),
	)
}

func (m *Model) resizeLists() {
	bodyHeight := m.bodyHeight()
	if bodyHeight < 1 {
		return
	}

	// Table takes top ~60%, bento takes bottom ~40%
	tableH := bodyHeight * 60 / 100
	bentoH := bodyHeight - tableH

	// Table is full width
	tableInnerW, tableInnerH := ui.InnerSize(m.width, tableH)
	m.prTable.SetWidth(tableInnerW)
	m.prTable.SetHeight(tableInnerH)
	m.updatePRTableColumns(tableInnerW)

	// Detail viewport in bottom-left bento panel
	detailW, _, _ := m.bentoPanelWidths()
	detailInnerW, detailInnerH := ui.InnerSize(detailW, bentoH)
	m.detailView.SetWidth(detailInnerW)
	m.detailView.SetHeight(detailInnerH - 1) // -1 for title

	// Repos overlay
	m.allRepoList.SetSize(m.width-4, bodyHeight-4)
}

func (m *Model) updatePRTableColumns(innerW int) {
	fixedWidth := 1 + 7 + 8 + 8 // dot(1) + type(7) + age(8) + padding(8)
	titleWidth := innerW - fixedWidth
	if titleWidth < 15 {
		titleWidth = 15
	}

	m.prTable.SetColumns([]table.Column{
		{Title: "", Width: 1},
		{Title: "Title", Width: titleWidth},
		{Title: "Type", Width: 7},
		{Title: "Age", Width: 8},
	})
}

func (m Model) bodyHeight() int {
	header := ui.RenderStatusBar("", "", "", m.width)
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

func (m Model) bentoPanelWidths() (detail, system, jobs int) {
	w := m.width
	detail = w * 45 / 100
	system = w * 22 / 100
	jobs = w - detail - system
	if detail < 20 {
		detail = 20
	}
	if system < 15 {
		system = 15
	}
	if jobs < 15 {
		jobs = 15
	}
	return
}
