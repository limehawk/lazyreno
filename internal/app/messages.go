package app

import "github.com/limehawk/lazyreno/internal/backend"

// Data fetch results — sent from async commands to Update().
type PRsFetchedMsg struct {
	PRs []backend.PR
	Err error
}

type ReposFetchedMsg struct {
	Repos []string
	Err   error
}

type JobQueueFetchedMsg struct {
	Jobs []backend.Job
	Err  error
}

type SystemStatusFetchedMsg struct {
	Status *backend.SystemStatus
	Err    error
}

type JobHistoryFetchedMsg struct {
	Repo string
	Jobs []backend.Job
	Err  error
}

// Action results
type MergePRResultMsg struct {
	Repo   string
	Number int
	Err    error
}

type ClosePRResultMsg struct {
	Repo   string
	Number int
	Err    error
}

type SyncTriggeredMsg struct {
	Err error
}

type PurgeResultMsg struct {
	Err error
}

// Timer tick for background polling.
type TickMsg struct{}

// Flash message for status bar.
type FlashMsg struct {
	Text    string
	IsError bool
}
