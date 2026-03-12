package backend

import "time"

// PR represents a GitHub pull request created by Renovate.
type PR struct {
	Number     int
	Title      string
	URL        string
	Branch     string
	Base       string
	State      string // "open", "closed", "merged"
	Mergeable  bool
	ChecksPass bool
	UpdateType string // "major", "minor", "patch", "digest", "pin", ""
	CreatedAt  time.Time
	Repo       string // "owner/name"
	Labels     []string
}

// Repo represents a repository managed by Renovate.
type Repo struct {
	FullName  string // "owner/name"
	OpenPRs   int
	LastRunAt *time.Time
	LastRunOK bool
}

// Job represents a Renovate job (queued, running, or completed).
type Job struct {
	ID        string
	Repo      string // "owner/name"
	Status    string // "running", "pending", "failed", "success"
	StartedAt *time.Time
	Duration  time.Duration
	Trigger   string
}

// JobLog is a single parsed line from Renovate JSONL logs.
type JobLog struct {
	Level   string // "INFO", "WARN", "ERROR", "DEBUG"
	Message string
	Time    time.Time
}

// SystemStatus represents the Renovate CE system state.
type SystemStatus struct {
	Version      string
	BootTime     time.Time
	Uptime       time.Duration
	Enabled      map[string]bool // feature flags
	QueueSize    int
	RunningJob   int
	FailedJobs   int
	LastFinished *Job
}
