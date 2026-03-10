package app

import (
	"fmt"
	"os/exec"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.hasDarkBG = msg.IsDark()
		m.rebuildTheme()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetWidth(msg.Width)
		m.resizeLists()
		return m, nil

	case tea.KeyMsg:
		// Confirmation form intercepts all messages when active
		// (handled below in the confirmForm block)

	default:
		// non-key, non-special messages fall through
	}

	// Forward messages to the confirm form when active.
	if m.confirmForm != nil {
		model, cmd := m.confirmForm.Update(msg)
		m.confirmForm = model.(*huh.Form)
		if m.confirmForm.State == huh.StateCompleted {
			if m.confirmed {
				actionCmd := m.confirmFn()
				m.confirmForm = nil
				m.confirmFn = nil
				return m, tea.Batch(cmd, actionCmd)
			}
			m.confirmForm = nil
			m.confirmFn = nil
			return m, cmd
		}
		if m.confirmForm.State == huh.StateAborted {
			m.confirmForm = nil
			m.confirmFn = nil
			return m, cmd
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, GlobalKeys.Quit):
			return m, tea.Quit
		case key.Matches(msg, GlobalKeys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case key.Matches(msg, GlobalKeys.Tab1):
			m.activeTab = TabPRs
			m.focusedPanel = 0
			return m, nil
		case key.Matches(msg, GlobalKeys.Tab2):
			m.activeTab = TabRepos
			m.focusedPanel = 0
			return m, nil
		case key.Matches(msg, GlobalKeys.Tab3):
			m.activeTab = TabJobs
			m.focusedPanel = 0
			return m, nil
		case key.Matches(msg, GlobalKeys.Tab4):
			m.activeTab = TabStatus
			m.focusedPanel = 0
			return m, nil
		case key.Matches(msg, GlobalKeys.NextTab):
			m.activeTab = (m.activeTab + 1) % 4
			m.focusedPanel = 0
			return m, nil
		case key.Matches(msg, GlobalKeys.PrevTab):
			m.activeTab = (m.activeTab + 3) % 4
			m.focusedPanel = 0
			return m, nil
		case key.Matches(msg, GlobalKeys.FocusNext):
			maxPanel := 2
			if m.activeTab != TabPRs {
				maxPanel = 1
			}
			if m.activeTab == TabStatus {
				maxPanel = 0
			}
			m.focusedPanel = (m.focusedPanel + 1) % (maxPanel + 1)
			return m, nil
		case key.Matches(msg, GlobalKeys.FocusPrev):
			maxPanel := 2
			if m.activeTab != TabPRs {
				maxPanel = 1
			}
			if m.activeTab == TabStatus {
				maxPanel = 0
			}
			m.focusedPanel = (m.focusedPanel + maxPanel) % (maxPanel + 1)
			return m, nil
		case key.Matches(msg, GlobalKeys.Refresh):
			return m, tea.Batch(m.fetchRepos(), m.fetchStatus(), m.fetchJobQueue())
		}

		// Delegate to tab-specific + focused-panel handling
		return m.updateActiveTab(msg)

	// Data messages
	case ReposFetchedMsg:
		if msg.Err != nil {
			m.setFlash(msg.Err.Error(), true)
		} else {
			m.repos = msg.Repos
			m.prs = nil
			cmd1 := m.rebuildAllRepoList()
			return m, tea.Batch(cmd1, m.fetchAllPRs())
		}
	case PRsFetchedMsg:
		if msg.Err != nil {
			m.setFlash(msg.Err.Error(), true)
		} else {
			m.prs = append(m.prs, msg.PRs...)
			cmd1 := m.rebuildRepoList()
			cmd2 := m.rebuildPRList()
			m.updateDetailView()
			return m, tea.Batch(cmd1, cmd2)
		}
	case JobQueueFetchedMsg:
		if msg.Err != nil {
			m.setFlash(msg.Err.Error(), true)
		} else {
			m.jobs = msg.Jobs
			cmd := m.rebuildJobList()
			return m, cmd
		}
	case SystemStatusFetchedMsg:
		if msg.Err != nil {
			m.setFlash(msg.Err.Error(), true)
		} else {
			m.status = msg.Status
			m.updateStatusView()
		}
	case MergePRResultMsg:
		if msg.Err != nil {
			m.setFlash("Merge failed: "+msg.Err.Error(), true)
		} else {
			m.setFlash(fmt.Sprintf("Merged #%d", msg.Number), false)
			m.prs = removePR(m.prs, msg.Repo, msg.Number)
			cmd1 := m.rebuildRepoList()
			cmd2 := m.rebuildPRList()
			m.updateDetailView()
			return m, tea.Batch(cmd1, cmd2)
		}
	case ClosePRResultMsg:
		if msg.Err != nil {
			m.setFlash("Close failed: "+msg.Err.Error(), true)
		} else {
			m.setFlash(fmt.Sprintf("Closed #%d", msg.Number), false)
			m.prs = removePR(m.prs, msg.Repo, msg.Number)
			cmd1 := m.rebuildRepoList()
			cmd2 := m.rebuildPRList()
			m.updateDetailView()
			return m, tea.Batch(cmd1, cmd2)
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
		// Don't clear existing data — let it stay visible while refreshing.
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
	var cmd tea.Cmd

	switch m.activeTab {
	case TabPRs:
		// Forward keys to the focused list.
		if m.focusedPanel == 0 {
			prevIdx := m.repoList.Index()
			m.repoList, cmd = m.repoList.Update(msg)
			if m.repoList.Index() != prevIdx {
				// Sidebar selection changed — rebuild PR list.
				cmd2 := m.rebuildPRList()
				m.updateDetailView()
				return m, tea.Batch(cmd, cmd2)
			}
			return m, cmd
		} else if m.focusedPanel == 1 {
			prevIdx := m.prList.Index()
			m.prList, cmd = m.prList.Update(msg)
			if m.prList.Index() != prevIdx {
				m.updateDetailView()
			}
			// Check for PR-specific actions.
			return m.handlePRActions(msg)
		} else {
			// Detail pane: forward to viewport for scrolling.
			m.detailView, cmd = m.detailView.Update(msg)
			return m, cmd
		}

	case TabRepos:
		if m.focusedPanel == 0 {
			m.allRepoList, cmd = m.allRepoList.Update(msg)
			return m, cmd
		}
		// Main panel in Repos tab is just informational, no list.
		return m, nil

	case TabJobs:
		if m.focusedPanel == 0 {
			m.jobList, cmd = m.jobList.Update(msg)
			// Check for job-specific actions.
			return m.handleJobActions(msg)
		}
		// Main panel in Jobs tab is informational.
		return m, nil

	case TabStatus:
		// Status tab: forward to viewport for scrolling.
		m.statusView, cmd = m.statusView.Update(msg)
		// Check for status-specific actions.
		return m.handleStatusActions(msg)
	}

	return m, nil
}

func (m *Model) showConfirm(message string, fn func() tea.Cmd) tea.Cmd {
	m.confirmed = false
	m.confirmFn = fn
	m.confirmForm = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(message).
				Affirmative("Yes").
				Negative("No").
				Value(&m.confirmed),
		),
	).WithWidth(40)
	return m.confirmForm.Init()
}

func (m *Model) handlePRActions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	selectedPR := m.getSelectedPR()
	if selectedPR == nil {
		return m, nil
	}

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("m"))):
		pr := *selectedPR
		cmd := m.showConfirm(fmt.Sprintf("Merge #%d into %s?", pr.Number, pr.Base), func() tea.Cmd {
			return func() tea.Msg {
				owner, repo := splitRepo(pr.Repo)
				err := m.github.MergePR(owner, repo, pr.Number)
				return MergePRResultMsg{Repo: pr.Repo, Number: pr.Number, Err: err}
			}
		})
		return m, cmd

	case key.Matches(msg, key.NewBinding(key.WithKeys("M"))):
		safePRs := m.getSafePRsForSelectedRepo()
		if len(safePRs) == 0 {
			m.setFlash("No safe PRs to merge", true)
			return m, nil
		}
		cmd := m.showConfirm(fmt.Sprintf("Merge %d safe PRs?", len(safePRs)), func() tea.Cmd {
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
		})
		return m, cmd

	case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
		pr := *selectedPR
		cmd := m.showConfirm(fmt.Sprintf("Close #%d and delete branch?", pr.Number), func() tea.Cmd {
			return func() tea.Msg {
				owner, repo := splitRepo(pr.Repo)
				err := m.github.ClosePR(owner, repo, pr.Number, pr.Branch)
				return ClosePRResultMsg{Repo: pr.Repo, Number: pr.Number, Err: err}
			}
		})
		return m, cmd

	case key.Matches(msg, key.NewBinding(key.WithKeys("o"))):
		return m, tea.ExecProcess(exec.Command("xdg-open", selectedPR.URL), nil)
	}

	return m, nil
}

func (m *Model) handleJobActions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.renovate == nil {
		return m, nil
	}

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
		sel := m.jobList.SelectedItem()
		if sel == nil {
			return m, nil
		}
		ji, ok := sel.(JobItem)
		if !ok {
			return m, nil
		}
		job := ji.Job
		return m, func() tea.Msg {
			err := m.renovate.AddJob(job.Repo)
			return SyncTriggeredMsg{Err: err}
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("p"))):
		cmd := m.showConfirm("Purge all failed jobs?", func() tea.Cmd {
			return func() tea.Msg {
				err := m.renovate.PurgeFailedJobs()
				return PurgeResultMsg{Err: err}
			}
		})
		return m, cmd
	}

	return m, nil
}

func (m *Model) handleStatusActions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		cmd := m.showConfirm("Purge all failed jobs?", func() tea.Cmd {
			return func() tea.Msg {
				err := m.renovate.PurgeFailedJobs()
				return PurgeResultMsg{Err: err}
			}
		})
		return m, cmd
	}

	return m, nil
}

func (m *Model) setFlash(text string, isError bool) {
	m.flashText = text
	m.flashIsError = isError
	m.flashExpiry = time.Now().Add(5 * time.Second)
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
