# lazyreno — TUI Dashboard for Self-Hosted Renovate CE

A lazy-style TUI for monitoring and managing a self-hosted Renovate Community Edition instance. Shows open PRs across all repos, job queue status, repo dependency health, and system status. Merge PRs, trigger syncs, and retry jobs without leaving the terminal.

## Context

Limehawk runs Renovate CE on Dokploy (Hetzner) at `reno.limehawk.dev`. It manages ~15 repos via a GitHub App. There is no web dashboard for Renovate CE — this TUI fills that gap.

The tool will be open-sourced for other self-hosted Renovate CE users.

## Stack

- **Language:** Go
- **TUI framework:** Bubbletea (Elm architecture) + Lipgloss (styling) + Bubbles (components)
- **GitHub client:** `go-github` (for distribution — no `gh` CLI dependency)
- **HTTP client:** `net/http` for Renovate CE REST API
- **Config:** TOML (`~/.config/lazyreno/config.toml`) + env var overrides
- **Distribution:** Single binary via `go install` and GitHub Releases

## Architecture

```
lazyreno binary
  ├── GitHub API (go-github) ─── PR data, checks, merge/close actions
  ├── Renovate CE API (REST) ─── system status, job queue, repo stats, logs
  └── Config (~/.config/lazyreno/config.toml + env vars)
```

### Data Flow

- On startup: fetch orgs/repos from Renovate API and open PRs from GitHub API in parallel.
- Background polling at configurable interval (default 30s), updates both sources.
- Actions (merge, sync, retry) are fire-and-forget with optimistic UI update, confirmed on next poll.
- All API responses cached in-memory, invalidated on poll or manual refresh (`R`).

### Auth

| Secret | Env var | Fallback |
|--------|---------|----------|
| Renovate CE API secret | `LAZYRENO_RENOVATE_SECRET` | None (required) |
| GitHub token | `LAZYRENO_GITHUB_TOKEN` | `GITHUB_TOKEN` |

Config file stores non-secret values (URL, owner, UI preferences). Secrets are env-only.

## Renovate CE API

Requires these env vars on the Renovate CE server (already enabled):

| Env var | Purpose |
|---------|---------|
| `MEND_RNV_API_ENABLED=true` | Enables API |
| `MEND_RNV_API_ENABLE_SYSTEM=true` | System status, job queue, sync triggers |
| `MEND_RNV_API_ENABLE_REPORTING=true` | Repo stats, org stats, LibYears |
| `MEND_RNV_API_ENABLE_JOBS=true` | Job logs per repo |
| `RENOVATE_REPOSITORY_CACHE=enabled` | Required for reporting data collection |

### Endpoints Used

**System endpoints** (require `MEND_RNV_API_ENABLE_SYSTEM=true`):

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/system/v1/status` | System status (version, bootTime, enabled features) |
| GET | `/system/v1/tasks/queue` | Queue depth, pending count |
| GET | `/system/v1/jobs/queue` | Up to 100 jobs (running + pending arrays) |
| GET | `/system/v1/jobs/logs/{jobId}` | Logs for a specific job by ID |
| POST | `/system/v1/sync` | Trigger app sync (discover new repos) |
| POST | `/system/v1/jobs/add` | Enqueue job for a repo (`{ "repository": "org/repo" }`) |
| POST | `/system/v1/jobs/purge` | Purge failed jobs |

**Reporting endpoints** (require `MEND_RNV_API_ENABLE_REPORTING=true`):

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/orgs` | List all orgs with metadata |
| GET | `/api/v1/orgs/{org}` | Org stats (repo counts) |
| GET | `/api/v1/orgs/{org}/-/repos` | Repos for an org (note `/-/` separator) |
| GET | `/api/v1/repos/{orgRepo}` | Per-repo stats — `{orgRepo}` is a single slug like `limehawk/gruman-law-website` |

**Jobs endpoints** (require `MEND_RNV_API_ENABLE_JOBS=true`):

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/repos/{orgRepo}/-/jobs` | Paginated job history for a repo (includes failed jobs) |
| GET | `/api/v1/repos/{orgRepo}/-/jobs/{jobId}` | Logs for a specific job in a repo |

All reporting and jobs endpoints use cursor-based pagination via `Link` header / `x-next-cursor`.

Auth: `Authorization: Bearer <MEND_RNV_SERVER_API_SECRET>` header on all requests.

### Data Source Responsibilities

- **GitHub API** is the canonical source for PR data (title, status, checks, mergeable state, merge actions). GitHub is always authoritative for PR operations.
- **Renovate CE API** is the source for job queue, job history, repo management state, and system status. It also returns PR metadata in repo stats, but we use this only for supplementary info (e.g., PR counts on the Repos tab) — never to contradict GitHub.
- **Failed jobs** are NOT in the queue endpoint (which only has running + pending). Failed jobs come from the job history endpoint (`/api/v1/repos/{orgRepo}/-/jobs`) filtered by non-success status.
- **Job logs** are returned as JSON Lines (newline-delimited JSON objects). The TUI parses each line and extracts `level` and `msg` fields for display, color-coding by log level.

## Views

Four tabs: `[1] PRs  [2] Repos  [3] Jobs  [4] Status`

### Tab 1: PRs (default)

Sidebar: repos with open PR count, sorted by count descending. Main: PRs for selected repo. Detail: selected PR info.

```
╭─ Repos (4 open) ──────╮╭─ Pull Requests ─────────────────────╮╭─ Details ──────────────╮
│ ● gruman-law       2   ││  ● shadcn to v4          minor  2d ││ #8 shadcn to v4        │
│   mill-mama        6   ││  ● eslint to v10         major  2d ││                        │
│   american-cable   1   ││                                     ││ Branch: renovate/shadcn│
│   peertitle        1   ││                                     ││ Base:   main           │
│   elite-suites     1   ││                                     ││ Checks: ✓ passing      │
│   twofold          1   ││                                     ││ Merge:  ✓ mergeable    │
│   5lime-scanner    2   ││                                     ││ Age:    2 days         │
│                        ││                                     ││ Type:   minor          │
│                        ││                                     ││                        │
│                        ││                                     ││ [m]erge [c]lose        │
│                        ││                                     ││ [o]pen in browser      │
├────────────────────────┤├─────────────────────────────────────┤├────────────────────────┤
│ /filter  [M]erge safe  ││ 14 PRs total  3 mergeable          ││ [?] help               │
╰────────────────────────╯╰─────────────────────────────────────╯╰────────────────────────╯
```

PR list item format:
```
 ● chore(deps): update shadcn to v4    minor  2d
   limehawk/gruman-law-website          ✓ mergeable
```

### Tab 2: Repos

Sidebar: all managed repos. Main: repo info (last run, status, open PRs, config). Detail: dependency stats (LibYears, outdated counts).

```
╭─ Repos ────────────────╮╭─ Repository Info ─────────────────────╮╭─ Stats ──────────────╮
│ ● gruman-law           ││ Last run:    10m ago                   ││ LibYears: 2.3        │
│   mill-mama            ││ Status:      ✓ up to date              ││ Outdated: 4 deps     │
│   american-cable       ││ Open PRs:    2                         ││ Major:    1          │
│   peertitle            ││ Config:      global (config.js)        ││ Minor:    2          │
│   ...                  ││ Schedule:    default                   ││ Patch:    1          │
╰────────────────────────╯╰────────────────────────────────────────╯╰──────────────────────╯
```

### Tab 3: Jobs

Sidebar: job queue grouped by status (running/pending/failed). Main: log output for selected job. Detail: job context.

```
╭─ Queue (3) ────────────╮╭─ Job Log ─────────────────────────────╮╭─ Context ────────────╮
│ ● gruman-law  running  ││ INFO: Repository started               ││ Repo: gruman-law     │
│   mill-mama   pending  ││ INFO: Using config from global          ││ Trigger: scheduled   │
│   peertitle   pending  ││ INFO: Found 2 updates                  ││ Duration: 12s        │
│ ─── failed ──────────  ││ INFO: Branch renovate/shadcn created   ││ Started: 2m ago      │
│   5lime-scan  ✗ error  ││ INFO: PR #8 created                    ││                      │
│                        ││                                        ││ [r]etry [p]urge      │
╰────────────────────────╯╰────────────────────────────────────────╯╰──────────────────────╯
```

Log view: auto-scroll (follow mode) by default. `f` toggles follow. Log levels color-coded: ERROR red, WARN yellow, INFO default, DEBUG dim.

### Tab 4: Status

Full-width system overview. No sidebar.

```
╭─ System Status ─────────────────────────────────────────────────────────────────────────╮
│                                                                                         │
│  Renovate CE v14.1.0          API: ✓ connected       Uptime: 4d 12h                    │
│  URL: reno.limehawk.dev       Jobs: 3 queued         Last sync: 5m ago                 │
│  Repos: 14 managed            Failed: 1              Next sync: 25m                    │
│                                                                                         │
│  ─── Config ────────────────────────────────────────────────────────────────────────     │
│  platformAutomerge: false     automerge: minor/patch  ignoreTests: true                 │
│  dependencyDashboard: true    schedule: default        repoCache: enabled               │
│                                                                                         │
│  [s]ync now   [p]urge failed                                                            │
╰─────────────────────────────────────────────────────────────────────────────────────────╯
```

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `1`-`4` / `[` `]` | Switch tabs |
| `Tab` / `Shift+Tab` | Cycle panel focus |
| `j` / `k` | Move down/up in lists |
| `g` / `G` | Jump to top/bottom |
| `h` / `l` | Move focus left/right between panels |
| `/` | Filter active list (fuzzy) |
| `Esc` | Back, cancel, close overlay, clear filter |
| `?` | Toggle help overlay |
| `R` | Force refresh all data |
| `q` | Quit |
| `ctrl+d` / `ctrl+u` | Half-page scroll |

### Context-Specific

| Context | Key | Action | Confirmation |
|---------|-----|--------|-------------|
| PR list | `m` | Merge selected PR | `Merge #8 into main? [y/N]` |
| PR list | `M` | Merge all safe PRs for selected repo (see below) | `Merge 3 safe PRs? [y/N]` |
| PR list | `c` | Close selected PR (also deletes the Renovate branch) | `Close #8 and delete branch? [y/N]` |
| PR list | `o` | Open PR in browser | No |
| Jobs list | `r` | Retry failed job (enqueues new job for same repo via `/system/v1/jobs/add`) | No |
| Jobs list | `p` | Purge all failed jobs | `Purge 2 failed jobs? [y/N]` |
| Status | `s` | Trigger Renovate sync | No |

### "Merge Safe" Logic (`M` key)

A PR is considered "safe to auto-merge" when ALL of these are true:
1. **Update type is minor or patch** — determined by Renovate's labels on the PR (e.g., `renovate/minor`, `renovate/patch`). Falls back to parsing the PR branch name prefix (`renovate/`) and title patterns (`update ... to v1.2.3`).
2. **PR is mergeable** — GitHub reports `mergeable: true` (no conflicts).
3. **All checks pass** — no failing or pending status checks.

Major updates, PRs with conflicts, or PRs with failing checks are never included in bulk merge.

### GitHub API Rate Limiting

At ~15 repos with 30s polling, the TUI makes ~30 API calls per poll cycle (list PRs + check statuses). GitHub's authenticated rate limit is 5,000/hour — this uses ~3,600/hour at default polling. The client includes:
- Respect for `X-RateLimit-Remaining` header — backs off when low.
- Conditional requests via `If-None-Match` / ETags to reduce quota consumption.
- Configurable poll interval to adjust if needed.

## Config

`~/.config/lazyreno/config.toml`:

```toml
[renovate]
url = "https://reno.limehawk.dev"
# secret: LAZYRENO_RENOVATE_SECRET env var

[github]
# token: LAZYRENO_GITHUB_TOKEN or GITHUB_TOKEN env var
owner = "limehawk"  # default owner to filter; Renovate API discovers all orgs regardless

[ui]
accent = "cyan"
refresh_interval = "30s"
```

## Project Structure

```
lazyreno/
  cmd/
    lazyreno/
      main.go                # Entry point, config loading, tea.NewProgram()
  internal/
    app/
      model.go               # Root model — active tab, sub-models, shared state
      update.go              # Root Update() — tab switching, global keys, message routing
      view.go                # Root View() — composes header + panels + status bar
      keys.go                # Global keybinding definitions
    ui/
      styles.go              # Lipgloss styles — borders, colors, accent, dim
      panel.go               # Generic focusable panel with border rendering
      header.go              # Tab bar rendering
      statusbar.go           # Status bar — context, key hints, flash messages
      help.go                # Help overlay
    tabs/
      prs/
        model.go             # PR tab model
        update.go            # PR tab update logic
        view.go              # PR tab view
        keys.go              # PR-specific keybindings (m, M, c, o)
      repos/
        model.go
        update.go
        view.go
      jobs/
        model.go
        update.go
        view.go
        keys.go              # Jobs-specific keybindings (r, p)
      status/
        model.go
        update.go
        view.go
        keys.go              # Status-specific keybindings (s, p)
    backend/
      github.go              # go-github client — list PRs, checks, merge, close
      renovate.go            # Renovate CE API client — all endpoints
      types.go               # Shared domain types (Repo, PR, Job, Status)
      cache.go               # In-memory cache with TTL, invalidation on action
    config/
      config.go              # TOML loading + env var override logic
```

## Error Handling

- API connection failures: flash message in status bar (red), auto-retry on next poll.
- Merge conflicts or failed merges: modal with error message and suggested action.
- Renovate CE unreachable: degrade gracefully — GitHub tabs still work, Renovate tabs show "disconnected".
- Invalid config: exit with clear error message on startup.

## Responsive Layout

- Wide terminal (>120 cols): full 3-panel layout.
- Medium (80-120): detail panel collapses, main panel expands.
- Minimum width: 80 cols. Below that, show a "terminal too narrow" message.
- Status tab always full-width regardless of terminal size.

## v2 Candidates (not in v1)

- Narrow terminal (<80 cols) sidebar collapse behavior.
- LibYears display in Repos tab (requires async cron computation).
- Prometheus `/metrics` integration.
- Configurable accent color.
- Schedule display in Status tab (would require parsing Renovate config, not exposed via API).

## Visual Style

Per the lazy-style TUI prompt:
- Terminal default background, one accent color (configurable, default cyan).
- Rounded box-drawing borders. Focused panel = bright accent border, unfocused = dim.
- Dense layout, 1-space panel padding, truncate with `...` in lists.
- Right-align ages, counts, statuses. Left-align names.
- Semantic colors: red = error/failed, yellow = warning/major update, green = success/mergeable.
