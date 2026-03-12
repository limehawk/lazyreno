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
	width  int
	height int

	// Shared data
	repos          []string
	prs            []backend.PR
	prsByRepo      map[string][]backend.PR // indexed on PR fetch completion
	pendingPRs     []backend.PR
	pendingPRCount int
	prBatchQueue   [][]string // remaining batches to fetch
	status         *backend.SystemStatus
	jobs           []backend.Job

	// Filtered PRs for the currently selected repo.
	filteredPRs []backend.PR

	// UI state
	keys          KeyMap
	help          help.Model
	spinner       spinner.Model
	lastUpdate    time.Time
	showRepos     bool // overlay toggle
	confirmText   string
	confirmAction func() tea.Cmd
	flashText     string
	flashIsError  bool
	flashExpiry   time.Time
	focusedPanel  int // 0=sidebar, 1=table, 2=detail

	// Cached layout (recalculated on WindowSizeMsg and help toggle)
	cachedSidebarW int
	cachedRightW   int
	cachedDetailW  int
	cachedSystemW  int
	cachedJobsW    int

	// Sidebar: repos with open PRs
	repoList list.Model

	// All repos overlay
	allRepoList list.Model

	// PR table (right column, top)
	prTable table.Model

	// Detail viewport (right column, bottom-left bento panel)
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

func newPRTable() table.Model {
	cols := []table.Column{
		{Title: "", Width: 3},
		{Title: "Title", Width: 40},
		{Title: "Type", Width: 7},
		{Title: "Age", Width: 8},
	}

	return table.New(
		table.WithColumns(cols),
		table.WithFocused(false),
		table.WithStyles(prTableStyles()),
	)
}

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
			Foreground(ui.Accent),
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
		keys:        GlobalKeys,
		help:        h,
		spinner:     sp,
		repoList:    newSidebarList("Repos"),
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

func (m Model) getSelectedPR() *backend.PR {
	idx := m.prTable.Cursor()
	if idx < 0 || idx >= len(m.filteredPRs) {
		return nil
	}
	pr := m.filteredPRs[idx]
	return &pr
}

func (m Model) getSafePRsForSelectedRepo() []backend.PR {
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
	for _, pr := range m.prsByRepo[fullName] {
		if backend.IsSafeToMerge(pr) {
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

func (m *Model) rebuildRepoList() tea.Cmd {
	prevIdx := m.repoList.Index()
	prsByRepo := m.groupPRsByRepo()
	repoOrder := m.getReposWithPRs(prsByRepo)

	items := make([]list.Item, len(repoOrder))
	for i, repo := range repoOrder {
		fullName := m.cfg.GitHub.Owner + "/" + repo
		items[i] = RepoItem{Name: repo, PRCount: len(prsByRepo[fullName])}
	}
	m.repoList.Title = fmt.Sprintf("Repos (%d)", len(items))
	cmd := m.repoList.SetItems(items)
	if prevIdx < len(items) {
		m.repoList.Select(prevIdx)
	}
	return cmd
}

func (m *Model) rebuildPRTable() {
	sel := m.repoList.SelectedItem()
	if sel == nil {
		m.filteredPRs = nil
		m.prTable.SetRows(nil)
		return
	}
	ri, ok := sel.(RepoItem)
	if !ok {
		m.filteredPRs = nil
		m.prTable.SetRows(nil)
		return
	}
	fullName := m.cfg.GitHub.Owner + "/" + ri.Name

	prevCursor := m.prTable.Cursor()
	m.filteredPRs = m.prsByRepo[fullName]

	if prevCursor < len(m.filteredPRs) {
		m.prTable.SetCursor(prevCursor)
	}
	m.stampPRTableCursor()
}

func prToRow(pr backend.PR, selected bool) table.Row {
	dot := ui.PRStatusDot(pr.ChecksPass, pr.Mergeable)
	indicator := "  " + dot
	if selected {
		indicator = ui.AccentText.Render("▸") + " " + dot
	}

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

	return table.Row{indicator, title, updateType, age}
}

func (m *Model) stampPRTableCursor() {
	cursor := m.prTable.Cursor()
	rows := make([]table.Row, len(m.filteredPRs))
	for i, pr := range m.filteredPRs {
		rows[i] = prToRow(pr, i == cursor)
	}
	m.prTable.SetRows(rows)
}

func (m *Model) rebuildAllRepoList() tea.Cmd {
	prevIdx := m.allRepoList.Index()
	items := make([]list.Item, len(m.repos))
	for i, repo := range m.repos {
		items[i] = AllRepoItem{Name: repo}
	}
	m.allRepoList.Title = fmt.Sprintf("All Repos (%d)", len(items))
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

func (m *Model) resizeLists() {
	bodyHeight := m.bodyHeight()
	if bodyHeight < 1 {
		return
	}

	sidebarW, rightW := m.panelWidths()

	// Sidebar: full height
	sidebarInnerW, sidebarInnerH := ui.InnerSize(sidebarW, bodyHeight)
	m.repoList.SetSize(sidebarInnerW, sidebarInnerH)

	// Right column: table top ~60%, bento bottom ~40%
	tableH := bodyHeight * 60 / 100
	bentoH := bodyHeight - tableH

	tableInnerW, tableInnerH := ui.InnerSize(rightW, tableH)
	m.prTable.SetWidth(tableInnerW)
	m.prTable.SetHeight(tableInnerH)
	m.updatePRTableColumns(tableInnerW)

	// Detail viewport in bottom-left bento panel
	detailW, _, _ := m.bentoPanelWidths(rightW)
	detailInnerW, detailInnerH := ui.InnerSize(detailW, bentoH)
	m.detailView.SetWidth(detailInnerW)
	m.detailView.SetHeight(detailInnerH - 1)

	// Repos overlay
	m.allRepoList.SetSize(m.width-4, bodyHeight-4)
}

func (m *Model) updatePRTableColumns(innerW int) {
	fixedWidth := 3 + 7 + 8 + 8
	titleWidth := innerW - fixedWidth
	if titleWidth < 15 {
		titleWidth = 15
	}

	m.prTable.SetColumns([]table.Column{
		{Title: "", Width: 3},
		{Title: "Title", Width: titleWidth},
		{Title: "Type", Width: 7},
		{Title: "Age", Width: 8},
	})
}

func (m Model) bodyHeight() int {
	header := ui.RenderStatusBar("", "", m.width)
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

func (m Model) panelWidths() (sidebar, right int) {
	w := m.width
	switch {
	case w >= 160:
		sidebar = 28
	case w >= 120:
		sidebar = 24
	case w >= 80:
		sidebar = 22
	default:
		sidebar = 20
	}
	right = w - sidebar
	if right < 40 {
		right = 40
	}
	return
}

func (m *Model) recalcLayout() {
	m.cachedSidebarW, m.cachedRightW = m.panelWidths()
	m.cachedDetailW, m.cachedSystemW, m.cachedJobsW = m.bentoPanelWidths(m.cachedRightW)
}

func (m Model) bentoPanelWidths(rightW int) (detail, system, jobs int) {
	detail = rightW * 45 / 100
	system = rightW * 22 / 100
	jobs = rightW - detail - system
	if detail < 20 {
		detail = 20
	}
	if system < 14 {
		system = 14
	}
	if jobs < 14 {
		jobs = 14
	}
	return
}
