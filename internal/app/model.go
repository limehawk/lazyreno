package app

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/limehawk/lazyreno/internal/backend"
	"github.com/limehawk/lazyreno/internal/config"
	"github.com/limehawk/lazyreno/internal/ui"
)

const (
	TabPRs   = 0
	TabRepos = 1
	tabCount = 2
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
	repos          []string
	prs            []backend.PR
	pendingPRs     []backend.PR
	pendingPRCount int // number of PR fetch responses expected
	status         *backend.SystemStatus

	// Filtered PRs for the currently selected repo (maps to prTable rows).
	filteredPRs []backend.PR

	// UI state
	keys         KeyMap
	help         help.Model
	confirmForm  *huh.Form
	confirmFn    func() tea.Cmd
	confirmed    *bool
	flashText    string
	flashIsError bool
	flashExpiry  time.Time
	focusedPanel int // 0=sidebar, 1=main, 2=detail

	// Sidebar lists (use default delegate)
	repoList    list.Model // PRs tab sidebar
	allRepoList list.Model // Repos tab sidebar

	// PR table (main panel on PRs tab)
	prTable table.Model

	// Detail viewport
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
	s := help.DefaultDarkStyles()
	s.ShortKey = ui.ShortcutKey
	s.FullKey = ui.ShortcutKey
	s.ShortDesc = ui.Dim
	s.ShortSeparator = lipgloss.NewStyle().Foreground(ui.Border)
	h.Styles = s

	return Model{
		cfg:         cfg,
		renovate:    renovate,
		github:      gh,
		cache:       backend.NewCache(30 * time.Second),
		keys:        GlobalKeys,
		help:        h,
		repoList:    newSidebarList("Repos"),
		allRepoList: newSidebarList("Repos"),
		prTable:     newPRTable(),
		detailView: viewport.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchRepos(),
		m.fetchStatus(),
		m.tickCmd(),
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

func (m *Model) rebuildRepoList() tea.Cmd {
	prevIdx := m.repoList.Index()
	prsByRepo := m.groupPRsByRepo()
	repoOrder := m.getReposWithPRs(prsByRepo)

	items := make([]list.Item, len(repoOrder))
	for i, repo := range repoOrder {
		fullName := m.cfg.GitHub.Owner + "/" + repo
		items[i] = RepoItem{Name: repo, PRCount: len(prsByRepo[fullName])}
	}
	m.repoList.Title = fmt.Sprintf("Repos (%d open)", len(items))
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
	status := "  "
	if pr.ChecksPass && pr.Mergeable {
		status = "ok"
	} else if !pr.ChecksPass {
		status = "!!"
	} else if !pr.Mergeable {
		status = "xx"
	}

	updateType := pr.UpdateType
	if updateType == "" {
		updateType = "dep"
	}

	return table.Row{status, pr.Title, updateType, backend.RelativeTime(pr.CreatedAt)}
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

	mergeStatus := ui.ErrorText.Render("!! conflict")
	if pr.Mergeable {
		mergeStatus = ui.SuccessText.Render("ok mergeable")
	}
	checkStatus := ui.ErrorText.Render("!! failing")
	if pr.ChecksPass {
		checkStatus = ui.SuccessText.Render("ok passing")
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
		ui.ShortcutKey.Render("[m]"), ui.ShortcutKey.Render("[c]"),
		ui.ShortcutKey.Render("[o]"),
	)

	m.detailView.SetContent(content)
}

func (m Model) renderStatusBox() string {
	if m.renovate == nil {
		return ui.WarningText.Render("Renovate CE not configured") + "\n" +
			ui.Dim.Render("Set LAZYRENO_RENOVATE_URL and LAZYRENO_RENOVATE_SECRET")
	}
	if m.status == nil {
		return ui.Dim.Render("Connecting to Renovate CE...")
	}

	s := m.status
	uptime := s.Uptime.Truncate(time.Minute).String()

	lines := []string{
		fmt.Sprintf("%s  %s", ui.Dim.Render("System"), ui.Bold.Render("v"+s.Version)),
		fmt.Sprintf("%s  %s", ui.Dim.Render("Uptime"), uptime),
		fmt.Sprintf("%s  %d", ui.Dim.Render("Queue "), s.QueueSize),
	}

	if s.LastFinished != nil {
		lf := s.LastFinished
		statusStyle := ui.SuccessText
		if lf.Status == "failed" {
			statusStyle = ui.ErrorText
		}
		_, repo := splitRepo(lf.Repo)
		jobLine := fmt.Sprintf("%s  %s", ui.Dim.Render("Last  "), repo)
		if lf.Duration > 0 {
			jobLine += fmt.Sprintf("  %s  %s",
				lf.Duration.Truncate(time.Second).String(),
				statusStyle.Render(lf.Status))
		} else {
			jobLine += "  " + statusStyle.Render(lf.Status)
		}
		lines = append(lines, jobLine)
	}

	return strings.Join(lines, "\n")
}

func (m *Model) resizeLists() {
	bodyHeight := m.bodyHeight()
	if bodyHeight < 1 {
		return
	}

	sidebarWidth, mainWidth, detailWidth := m.panelWidths()

	sidebarInnerW, sidebarInnerH := ui.InnerSize(sidebarWidth, bodyHeight)
	mainInnerW, mainInnerH := ui.InnerSize(mainWidth, bodyHeight)

	m.repoList.SetSize(sidebarInnerW, sidebarInnerH)
	m.allRepoList.SetSize(sidebarInnerW, sidebarInnerH)

	m.prTable.SetWidth(mainInnerW)
	m.prTable.SetHeight(mainInnerH)
	m.updatePRTableColumns(mainInnerW)

	if detailWidth > 0 {
		detailInnerW, detailInnerH := ui.InnerSize(detailWidth, bodyHeight)
		m.detailView.SetWidth(detailInnerW)
		m.detailView.SetHeight(detailInnerH - 1) // -1 for title line
	}

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
	header := ui.RenderHeader(m.activeTab, m.width)
	helpBar := m.help.View(TabKeyMap{KeyMap: m.keys, tab: m.activeTab})

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
	w := m.width

	switch {
	case w >= 180:
		sidebar = w * 20 / 100
		detail = w * 30 / 100
		main = w - sidebar - detail
	case w >= 140:
		sidebar = w * 22 / 100
		detail = w * 28 / 100
		main = w - sidebar - detail
	case w >= 100:
		sidebar = 25
		detail = w * 25 / 100
		main = w - sidebar - detail
	case w >= 80:
		sidebar = 25
		detail = 0
		main = w - sidebar
	default:
		sidebar = 20
		detail = 0
		main = w - sidebar
	}

	if sidebar < 18 {
		sidebar = 18
	}
	if detail > 0 && detail < 25 {
		detail = 25
	}
	if main < 20 {
		main = 20
	}
	return
}
