# lazyreno

TUI dashboard for self-hosted Renovate CE. Shows PRs, repos, jobs, and system status.

## Stack

- Go 1.24+
- TUI: bubbletea + lipgloss + bubbles
- GitHub: go-github
- Config: BurntSushi/toml
- Tests: go test (standard library)

## Commands

- Build: `go build ./cmd/lazyreno`
- Test: `go test ./...`
- Run: `go run ./cmd/lazyreno`
- Lint: `golangci-lint run`

## Architecture

Elm architecture (bubbletea). Root model in `internal/app/` dispatches to tab models in `internal/tabs/`.
Backend clients in `internal/backend/` — one for Renovate CE REST API, one for GitHub API.
Config in `internal/config/` — TOML file + env var overrides.

## Rules

- Always use `bun`/`bunx` for any JS tooling (not relevant here but inherited from org)
- No secrets in git — use env vars for tokens
- TDD — write failing test first, then implement
- Follow the lazy-style TUI conventions (vim keys, dense panels, box-drawing borders)
