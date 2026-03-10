package app

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
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
	showHelp     bool
	confirmMsg   string
	confirmFn    func() tea.Cmd
	flash        *ui.FlashMessage
	focusedPanel int // 0=sidebar, 1=main, 2=detail

	// Per-tab cursors
	sidebarCursor int
	mainCursor    int

	err error
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

	return Model{
		cfg:      cfg,
		renovate: renovate,
		github:   gh,
		cache:    backend.NewCache(30 * time.Second),
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
	prsByRepo := m.groupPRsByRepo()
	repoOrder := m.getReposWithPRs(prsByRepo)

	if m.sidebarCursor >= len(repoOrder) {
		return nil
	}
	selectedRepo := repoOrder[m.sidebarCursor]
	fullName := m.cfg.GitHub.Owner + "/" + selectedRepo
	prs := prsByRepo[fullName]
	if m.mainCursor >= len(prs) {
		return nil
	}
	pr := prs[m.mainCursor]
	return &pr
}

func (m Model) getSafePRsForSelectedRepo() []backend.PR {
	prsByRepo := m.groupPRsByRepo()
	repoOrder := m.getReposWithPRs(prsByRepo)

	if m.sidebarCursor >= len(repoOrder) {
		return nil
	}
	selectedRepo := repoOrder[m.sidebarCursor]
	fullName := m.cfg.GitHub.Owner + "/" + selectedRepo

	var safe []backend.PR
	for _, pr := range prsByRepo[fullName] {
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

func (m Model) getSidebarLen() int {
	switch m.activeTab {
	case TabPRs:
		prsByRepo := m.groupPRsByRepo()
		return len(m.getReposWithPRs(prsByRepo))
	case TabRepos:
		return len(m.repos)
	case TabJobs:
		return len(m.jobs)
	default:
		return 0
	}
}

func (m Model) getMainLen() int {
	switch m.activeTab {
	case TabPRs:
		prsByRepo := m.groupPRsByRepo()
		repoOrder := m.getReposWithPRs(prsByRepo)
		if m.sidebarCursor < len(repoOrder) {
			fullName := m.cfg.GitHub.Owner + "/" + repoOrder[m.sidebarCursor]
			return len(prsByRepo[fullName])
		}
	}
	return 0
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
