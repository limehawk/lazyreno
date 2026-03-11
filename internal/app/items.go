package app

import (
	"fmt"
	"strings"

	"github.com/limehawk/lazyreno/internal/backend"
)

// RepoItem wraps a repo name + PR count for the sidebar list.
// Used with list.DefaultDelegate (shows Title + optional Description).
type RepoItem struct {
	Name    string
	PRCount int
}

func (i RepoItem) FilterValue() string { return i.Name }
func (i RepoItem) Title() string       { return i.Name }
func (i RepoItem) Description() string { return fmt.Sprintf("%d PRs", i.PRCount) }

// AllRepoItem wraps a repo name for the repos overlay.
type AllRepoItem struct {
	Name string
}

func (i AllRepoItem) FilterValue() string { return i.Name }
func (i AllRepoItem) Title() string       { return i.Name }
func (i AllRepoItem) Description() string { return "" }

// JobItem wraps a job for the jobs sidebar list.
type JobItem struct {
	Job backend.Job
}

func (i JobItem) FilterValue() string { return i.Job.Repo }
func (i JobItem) Title() string {
	if parts := strings.SplitN(i.Job.Repo, "/", 2); len(parts) == 2 {
		return parts[1]
	}
	return i.Job.Repo
}
func (i JobItem) Description() string { return i.Job.Status }

