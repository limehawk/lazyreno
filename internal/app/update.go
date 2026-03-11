package app

import (
	"fmt"
	"os/exec"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetWidth(msg.Width)
		m.resizeLists()
		m.syncTableFocus()
		return m, nil

	case tea.KeyMsg:
		// handled below

	default:
		// non-key, non-special messages fall through
	}

	// Inline confirm intercepts next keypress.
	if m.confirmText != "" {
		if msg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(msg, key.NewBinding(key.WithKeys("y"))) {
				action := m.confirmAction
				m.confirmText = ""
				m.confirmAction = nil
				return m, action()
			}
			m.confirmText = ""
			m.confirmAction = nil
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Repos overlay intercepts all keys when shown.
		if m.showRepos {
			var cmd tea.Cmd
			if key.Matches(msg, GlobalKeys.Repos) || key.Matches(msg, GlobalKeys.Escape) || key.Matches(msg, GlobalKeys.Quit) {
				m.showRepos = false
				return m, nil
			}
			m.allRepoList, cmd = m.allRepoList.Update(msg)
			return m, cmd
		}

		switch {
		case key.Matches(msg, GlobalKeys.Quit):
			return m, tea.Quit
		case key.Matches(msg, GlobalKeys.Help):
			m.help.ShowAll = !m.help.ShowAll
			m.resizeLists()
			return m, nil
		case key.Matches(msg, GlobalKeys.Repos):
			m.showRepos = !m.showRepos
			return m, nil
		case key.Matches(msg, GlobalKeys.FocusNext):
			m.focusNext()
			return m, nil
		case key.Matches(msg, GlobalKeys.FocusPrev):
			m.focusPrev()
			return m, nil
		case key.Matches(msg, GlobalKeys.Refresh):
			return m, tea.Batch(m.fetchRepos(), m.fetchStatus(), m.fetchJobQueue(), m.spinner.Tick)
		}

		// Sync/purge available globally when renovate is configured.
		if m.renovate != nil {
			switch {
			case key.Matches(msg, GlobalKeys.Sync):
				return m, func() tea.Msg {
					err := m.renovate.TriggerSync()
					return SyncTriggeredMsg{Err: err}
				}
			case key.Matches(msg, GlobalKeys.Purge):
				m.showConfirm("Purge all failed jobs? [y/n]", func() tea.Cmd {
					return func() tea.Msg {
						err := m.renovate.PurgeFailedJobs()
						return PurgeResultMsg{Err: err}
					}
				})
				return m, nil
			}
		}

		// Delegate to focused-panel handling
		return m.updateFocusedPanel(msg)

	// Data messages
	case ReposFetchedMsg:
		if msg.Err != nil {
			m.setFlash(msg.Err.Error(), true)
		} else {
			m.repos = msg.Repos
			m.pendingPRs = nil
			batches := makeBatches(msg.Repos, maxConcurrentFetches)
			if len(batches) == 0 {
				m.pendingPRCount = 0
			} else {
				m.pendingPRCount = len(batches[0])
				m.prBatchQueue = batches[1:]
			}
			cmd1 := m.rebuildAllRepoList()
			if len(batches) > 0 {
				return m, tea.Batch(cmd1, m.fetchPRBatch(batches[0]))
			}
			return m, cmd1
		}
	case PRsFetchedMsg:
		if msg.Err != nil {
			m.setFlash(msg.Err.Error(), true)
			m.pendingPRCount--
		} else {
			m.pendingPRs = append(m.pendingPRs, msg.PRs...)
			m.pendingPRCount--
		}
		if m.pendingPRCount <= 0 {
			if len(m.prBatchQueue) > 0 {
				next := m.prBatchQueue[0]
				m.prBatchQueue = m.prBatchQueue[1:]
				m.pendingPRCount = len(next)
				return m, m.fetchPRBatch(next)
			}
			m.prs = m.pendingPRs
			m.pendingPRs = nil
			m.lastUpdate = time.Now()
			cmd1 := m.rebuildRepoList()
			m.rebuildPRTable()
			m.updateDetailView()
			return m, cmd1
		}
	case SystemStatusFetchedMsg:
		if msg.Err != nil {
			m.setFlash(msg.Err.Error(), true)
		} else {
			m.status = msg.Status
		}
	case JobQueueFetchedMsg:
		if msg.Err != nil {
			m.setFlash(msg.Err.Error(), true)
		} else {
			m.jobs = msg.Jobs
		}
	case MergePRResultMsg:
		if msg.Err != nil {
			m.setFlash("Merge failed: "+msg.Err.Error(), true)
		} else {
			m.setFlash(fmt.Sprintf("Merged #%d", msg.Number), false)
			m.prs = removePR(m.prs, msg.Repo, msg.Number)
			cmd1 := m.rebuildRepoList()
			m.rebuildPRTable()
			m.updateDetailView()
			return m, cmd1
		}
	case ClosePRResultMsg:
		if msg.Err != nil {
			m.setFlash("Close failed: "+msg.Err.Error(), true)
		} else {
			m.setFlash(fmt.Sprintf("Closed #%d", msg.Number), false)
			m.prs = removePR(m.prs, msg.Repo, msg.Number)
			cmd1 := m.rebuildRepoList()
			m.rebuildPRTable()
			m.updateDetailView()
			return m, cmd1
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
		return m, tea.Batch(
			m.fetchRepos(),
			m.fetchStatus(),
			m.fetchJobQueue(),
			m.tickCmd(),
			m.spinner.Tick,
		)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) updateFocusedPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Arrow keys for panel navigation.
	if key.Matches(msg, GlobalKeys.Right) {
		if m.focusedPanel < m.maxPanel() {
			m.focusNext()
		}
		return m, nil
	}
	if key.Matches(msg, GlobalKeys.Left) {
		if m.focusedPanel > 0 {
			m.focusPrev()
		}
		return m, nil
	}

	switch m.focusedPanel {
	case 0: // sidebar
		if key.Matches(msg, GlobalKeys.Enter) {
			m.focusNext()
			return m, nil
		}
		prevIdx := m.repoList.Index()
		m.repoList, cmd = m.repoList.Update(msg)
		if m.repoList.Index() != prevIdx {
			m.rebuildPRTable()
			m.updateDetailView()
		}
		return m, cmd
	case 1: // PR table
		if key.Matches(msg, key.NewBinding(key.WithKeys("m", "M", "c", "o", "enter"))) {
			return m.handlePRActions(msg)
		}
		prevCursor := m.prTable.Cursor()
		m.prTable, cmd = m.prTable.Update(msg)
		if m.prTable.Cursor() != prevCursor {
			m.stampPRTableCursor()
			m.updateDetailView()
		}
		return m, cmd
	case 2: // detail viewport
		m.detailView, cmd = m.detailView.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) showConfirm(message string, fn func() tea.Cmd) {
	m.confirmText = message
	m.confirmAction = fn
}

func (m *Model) handlePRActions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	selectedPR := m.getSelectedPR()
	if selectedPR == nil {
		return m, nil
	}

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("m", "enter"))):
		pr := *selectedPR
		m.showConfirm(fmt.Sprintf("Merge #%d into %s? [y/n]", pr.Number, pr.Base), func() tea.Cmd {
			return func() tea.Msg {
				owner, repo := splitRepo(pr.Repo)
				mergeable, _, err := m.github.GetPRMergeability(owner, repo, pr.Number)
				if err != nil {
					return MergePRResultMsg{Repo: pr.Repo, Number: pr.Number, Err: fmt.Errorf("mergeability check failed: %w", err)}
				}
				if !mergeable {
					return MergePRResultMsg{Repo: pr.Repo, Number: pr.Number, Err: fmt.Errorf("PR #%d is not mergeable", pr.Number)}
				}
				err = m.github.MergePR(owner, repo, pr.Number)
				return MergePRResultMsg{Repo: pr.Repo, Number: pr.Number, Err: err}
			}
		})
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("M"))):
		safePRs := m.getSafePRsForSelectedRepo()
		if len(safePRs) == 0 {
			m.setFlash("No safe PRs to merge", true)
			return m, nil
		}
		m.showConfirm(fmt.Sprintf("Merge %d safe PRs? [y/n]", len(safePRs)), func() tea.Cmd {
			var cmds []tea.Cmd
			for _, pr := range safePRs {
				pr := pr
				cmds = append(cmds, func() tea.Msg {
					owner, repo := splitRepo(pr.Repo)
					mergeable, _, err := m.github.GetPRMergeability(owner, repo, pr.Number)
					if err != nil {
						return MergePRResultMsg{Repo: pr.Repo, Number: pr.Number, Err: fmt.Errorf("check failed: %w", err)}
					}
					if !mergeable {
						return MergePRResultMsg{Repo: pr.Repo, Number: pr.Number, Err: fmt.Errorf("PR #%d not mergeable", pr.Number)}
					}
					err = m.github.MergePR(owner, repo, pr.Number)
					return MergePRResultMsg{Repo: pr.Repo, Number: pr.Number, Err: err}
				})
			}
			return tea.Batch(cmds...)
		})
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
		pr := *selectedPR
		m.showConfirm(fmt.Sprintf("Close #%d and delete branch? [y/n]", pr.Number), func() tea.Cmd {
			return func() tea.Msg {
				owner, repo := splitRepo(pr.Repo)
				err := m.github.ClosePR(owner, repo, pr.Number, pr.Branch)
				return ClosePRResultMsg{Repo: pr.Repo, Number: pr.Number, Err: err}
			}
		})
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("o"))):
		return m, tea.ExecProcess(exec.Command("xdg-open", selectedPR.URL), nil)
	}

	return m, nil
}

// syncTableFocus ensures the PR table focus matches the current panel state.
func (m *Model) syncTableFocus() {
	if m.focusedPanel == 1 {
		m.prTable.Focus()
	} else {
		m.prTable.Blur()
	}
}

func (m *Model) maxPanel() int {
	return 2 // sidebar, table, detail
}

func (m *Model) focusNext() {
	max := m.maxPanel()
	m.focusedPanel = (m.focusedPanel + 1) % (max + 1)
	m.syncTableFocus()
}

func (m *Model) focusPrev() {
	max := m.maxPanel()
	m.focusedPanel = (m.focusedPanel + max) % (max + 1)
	m.syncTableFocus()
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

const maxConcurrentFetches = 5

func makeBatches(repos []string, size int) [][]string {
	var batches [][]string
	for i := 0; i < len(repos); i += size {
		end := i + size
		if end > len(repos) {
			end = len(repos)
		}
		batches = append(batches, repos[i:end])
	}
	return batches
}

func (m Model) fetchPRBatch(repos []string) tea.Cmd {
	var cmds []tea.Cmd
	for _, repo := range repos {
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
