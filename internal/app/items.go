package app

// AllRepoItem wraps a repo name for the repos overlay.
type AllRepoItem struct {
	Name string
}

func (i AllRepoItem) FilterValue() string { return i.Name }
func (i AllRepoItem) Title() string       { return i.Name }
func (i AllRepoItem) Description() string { return "" }
