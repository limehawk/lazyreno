package app

import (
	"fmt"
	"os/exec"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/limehawk/lazyreno/internal/ui"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Confirmation dialog intercepts all keys
		if m.confirmMsg != "" {
			if msg.String() == "y" || msg.String() == "Y" {
				cmd := m.confirmFn()
				m.confirmMsg = ""
				m.confirmFn = nil
				return m, cmd
			}
			m.confirmMsg = ""
			m.confirmFn = nil
			return m, nil
		}

		// Help overlay intercepts escape
		if m.showHelp {
			if key.Matches(msg, GlobalKeys.Help, GlobalKeys.Escape) {
				m.showHelp = false
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, GlobalKeys.Quit):
			return m, tea.Quit
		case key.Matches(msg, GlobalKeys.Help):
			m.showHelp = true
			return m, nil
		case key.Matches(msg, GlobalKeys.Tab1):
			m.activeTab = TabPRs
			m.sidebarCursor = 0
			m.mainCursor = 0
		case key.Matches(msg, GlobalKeys.Tab2):
			m.activeTab = TabRepos
			m.sidebarCursor = 0
			m.mainCursor = 0
		case key.Matches(msg, GlobalKeys.Tab3):
			m.activeTab = TabJobs
			m.sidebarCursor = 0
			m.mainCursor = 0
		case key.Matches(msg, GlobalKeys.Tab4):
			m.activeTab = TabStatus
		case key.Matches(msg, GlobalKeys.NextTab):
			m.activeTab = (m.activeTab + 1) % 4
			m.sidebarCursor = 0
			m.mainCursor = 0
		case key.Matches(msg, GlobalKeys.PrevTab):
			m.activeTab = (m.activeTab + 3) % 4
			m.sidebarCursor = 0
			m.mainCursor = 0
		case key.Matches(msg, GlobalKeys.FocusNext):
			m.focusedPanel = (m.focusedPanel + 1) % 3
		case key.Matches(msg, GlobalKeys.FocusPrev):
			m.focusedPanel = (m.focusedPanel + 2) % 3
		case key.Matches(msg, GlobalKeys.Refresh):
			m.prs = nil
			return m, tea.Batch(m.fetchRepos(), m.fetchStatus(), m.fetchJobQueue())
		}

		// Delegate to tab-specific handling
		return m.updateActiveTab(msg)

	// Data messages
	case ReposFetchedMsg:
		if msg.Err != nil {
			m.setFlash(msg.Err.Error(), true)
		} else {
			m.repos = msg.Repos
			m.prs = nil // clear before refetch
			return m, m.fetchAllPRs()
		}
	case PRsFetchedMsg:
		if msg.Err != nil {
			m.setFlash(msg.Err.Error(), true)
		} else {
			m.prs = append(m.prs, msg.PRs...)
		}
	case JobQueueFetchedMsg:
		if msg.Err != nil {
			m.setFlash(msg.Err.Error(), true)
		} else {
			m.jobs = msg.Jobs
		}
	case SystemStatusFetchedMsg:
		if msg.Err != nil {
			m.setFlash(msg.Err.Error(), true)
		} else {
			m.status = msg.Status
		}
	case MergePRResultMsg:
		if msg.Err != nil {
			m.setFlash("Merge failed: "+msg.Err.Error(), true)
		} else {
			m.setFlash(fmt.Sprintf("Merged #%d", msg.Number), false)
			m.prs = removePR(m.prs, msg.Repo, msg.Number)
		}
	case ClosePRResultMsg:
		if msg.Err != nil {
			m.setFlash("Close failed: "+msg.Err.Error(), true)
		} else {
			m.setFlash(fmt.Sprintf("Closed #%d", msg.Number), false)
			m.prs = removePR(m.prs, msg.Repo, msg.Number)
		}
	case SyncTriggeredMsg:
		if msg.Err != nil {
			m.setFlash("Sync failed: "+msg.Err.Error(), true)
		} else {
			m.setFlash("Sync triggered", false)
		}
	case PurgeResultMsg:
		if msg.Err != nil {
			m.setFlash("Purge failed: "+msg.Err.Error(), true)
		} else {
			m.setFlash("Failed jobs purged", false)
		}
	case TickMsg:
		m.prs = nil
		return m, tea.Batch(
			m.fetchRepos(),
			m.fetchStatus(),
			m.fetchJobQueue(),
			m.tickCmd(),
		)
	}

	return m, nil
}

func (m *Model) updateActiveTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Navigation
	switch {
	case key.Matches(msg, GlobalKeys.Up):
		if m.focusedPanel == 0 && m.sidebarCursor > 0 {
			m.sidebarCursor--
			m.mainCursor = 0
		} else if m.focusedPanel == 1 && m.mainCursor > 0 {
			m.mainCursor--
		}
	case key.Matches(msg, GlobalKeys.Down):
		if m.focusedPanel == 0 {
			m.sidebarCursor++
		} else if m.focusedPanel == 1 {
			m.mainCursor++
		}
	case key.Matches(msg, GlobalKeys.Top):
		if m.focusedPanel == 0 {
			m.sidebarCursor = 0
			m.mainCursor = 0
		} else if m.focusedPanel == 1 {
			m.mainCursor = 0
		}
	case key.Matches(msg, GlobalKeys.Bottom):
		if m.focusedPanel == 0 {
			m.sidebarCursor = m.getSidebarLen() - 1
			m.mainCursor = 0
		} else if m.focusedPanel == 1 {
			m.mainCursor = m.getMainLen() - 1
		}
	}

	// Clamp cursors
	maxSidebar := m.getSidebarLen() - 1
	if maxSidebar < 0 {
		maxSidebar = 0
	}
	if m.sidebarCursor > maxSidebar {
		m.sidebarCursor = maxSidebar
	}
	if m.sidebarCursor < 0 {
		m.sidebarCursor = 0
	}
	maxMain := m.getMainLen() - 1
	if maxMain < 0 {
		maxMain = 0
	}
	if m.mainCursor > maxMain {
		m.mainCursor = maxMain
	}
	if m.mainCursor < 0 {
		m.mainCursor = 0
	}

	// Tab-specific actions
	switch m.activeTab {
	case TabPRs:
		return m.updatePRsTab(msg)
	case TabJobs:
		return m.updateJobsTab(msg)
	case TabStatus:
		return m.updateStatusTab(msg)
	}

	return m, nil
}

func (m *Model) updatePRsTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.focusedPanel != 1 {
		return m, nil
	}

	selectedPR := m.getSelectedPR()
	if selectedPR == nil {
		return m, nil
	}

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("m"))):
		pr := *selectedPR
		m.confirmMsg = fmt.Sprintf("Merge #%d into %s?", pr.Number, pr.Base)
		m.confirmFn = func() tea.Cmd {
			return func() tea.Msg {
				owner, repo := splitRepo(pr.Repo)
				err := m.github.MergePR(owner, repo, pr.Number)
				return MergePRResultMsg{Repo: pr.Repo, Number: pr.Number, Err: err}
			}
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("M"))):
		safePRs := m.getSafePRsForSelectedRepo()
		if len(safePRs) == 0 {
			m.setFlash("No safe PRs to merge", true)
			return m, nil
		}
		m.confirmMsg = fmt.Sprintf("Merge %d safe PRs?", len(safePRs))
		m.confirmFn = func() tea.Cmd {
			var cmds []tea.Cmd
			for _, pr := range safePRs {
				pr := pr
				cmds = append(cmds, func() tea.Msg {
					owner, repo := splitRepo(pr.Repo)
					err := m.github.MergePR(owner, repo, pr.Number)
					return MergePRResultMsg{Repo: pr.Repo, Number: pr.Number, Err: err}
				})
			}
			return tea.Batch(cmds...)
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
		pr := *selectedPR
		m.confirmMsg = fmt.Sprintf("Close #%d and delete branch?", pr.Number)
		m.confirmFn = func() tea.Cmd {
			return func() tea.Msg {
				owner, repo := splitRepo(pr.Repo)
				err := m.github.ClosePR(owner, repo, pr.Number, pr.Branch)
				return ClosePRResultMsg{Repo: pr.Repo, Number: pr.Number, Err: err}
			}
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("o"))):
		return m, tea.ExecProcess(exec.Command("xdg-open", selectedPR.URL), nil)
	}

	return m, nil
}

func (m *Model) updateJobsTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.renovate == nil {
		return m, nil
	}

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
		if m.focusedPanel == 0 && m.sidebarCursor < len(m.jobs) {
			job := m.jobs[m.sidebarCursor]
			return m, func() tea.Msg {
				err := m.renovate.AddJob(job.Repo)
				if err != nil {
					return SyncTriggeredMsg{Err: err}
				}
				return SyncTriggeredMsg{Err: nil}
			}
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("p"))):
		m.confirmMsg = "Purge all failed jobs?"
		m.confirmFn = func() tea.Cmd {
			return func() tea.Msg {
				err := m.renovate.PurgeFailedJobs()
				return PurgeResultMsg{Err: err}
			}
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) updateStatusTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.renovate == nil {
		return m, nil
	}

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("s"))):
		return m, func() tea.Msg {
			err := m.renovate.TriggerSync()
			return SyncTriggeredMsg{Err: err}
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("p"))):
		m.confirmMsg = "Purge all failed jobs?"
		m.confirmFn = func() tea.Cmd {
			return func() tea.Msg {
				err := m.renovate.PurgeFailedJobs()
				return PurgeResultMsg{Err: err}
			}
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) setFlash(text string, isError bool) {
	m.flash = &ui.FlashMessage{
		Text:      text,
		IsError:   isError,
		ExpiresAt: time.Now().Add(5 * time.Second),
	}
}

// Async commands
func (m Model) fetchRepos() tea.Cmd {
	return func() tea.Msg {
		if m.github == nil || m.cfg.GitHub.Owner == "" {
			return ReposFetchedMsg{Err: nil}
		}
		repos, err := m.github.ListOwnerRepos(m.cfg.GitHub.Owner)
		return ReposFetchedMsg{Repos: repos, Err: err}
	}
}

func (m Model) fetchAllPRs() tea.Cmd {
	if m.github == nil || m.cfg.GitHub.Owner == "" {
		return nil
	}

	var cmds []tea.Cmd
	for _, repo := range m.repos {
		repo := repo
		cmds = append(cmds, func() tea.Msg {
			prs, err := m.github.ListOpenPRs(m.cfg.GitHub.Owner, repo)
			return PRsFetchedMsg{PRs: prs, Err: err}
		})
	}
	return tea.Batch(cmds...)
}

func (m Model) fetchStatus() tea.Cmd {
	return func() tea.Msg {
		if m.renovate == nil {
			return SystemStatusFetchedMsg{Err: nil}
		}
		status, err := m.renovate.GetStatus()
		return SystemStatusFetchedMsg{Status: status, Err: err}
	}
}

func (m Model) fetchJobQueue() tea.Cmd {
	return func() tea.Msg {
		if m.renovate == nil {
			return JobQueueFetchedMsg{Err: nil}
		}
		jobs, err := m.renovate.GetJobQueue()
		return JobQueueFetchedMsg{Jobs: jobs, Err: err}
	}
}

func (m Model) tickCmd() tea.Cmd {
	d, _ := time.ParseDuration(m.cfg.UI.RefreshInterval)
	if d == 0 {
		d = 30 * time.Second
	}
	return tea.Tick(d, func(time.Time) tea.Msg {
		return TickMsg{}
	})
}
