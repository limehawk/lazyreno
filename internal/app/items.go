package app

import "fmt"

// RepoItem wraps a repo name + PR count for the sidebar list.
// Used with list.DefaultDelegate (shows Title + optional Description).
type RepoItem struct {
	Name    string
	PRCount int
}

func (i RepoItem) FilterValue() string { return i.Name }
func (i RepoItem) Title() string       { return i.Name }
func (i RepoItem) Description() string { return fmt.Sprintf("%d PRs", i.PRCount) }

// AllRepoItem wraps a repo name for the Repos tab sidebar.
type AllRepoItem struct {
	Name string
}

func (i AllRepoItem) FilterValue() string { return i.Name }
func (i AllRepoItem) Title() string       { return i.Name }
func (i AllRepoItem) Description() string { return "" }

