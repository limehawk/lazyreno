# lazyreno

A lazy-style TUI dashboard for self-hosted [Renovate CE](https://github.com/mend/renovate-ce-ee).

## Features

- **PRs tab** — View and merge Renovate PRs across all repos. Bulk-merge safe (minor/patch) PRs.
- **Repos tab** — Browse all repos managed by Renovate.
- **Jobs tab** — Monitor the Renovate job queue. Retry failed jobs.
- **Status tab** — System overview, trigger sync, purge failed jobs.
- Vim-style navigation, fuzzy filtering, keyboard-driven.

## Install

```bash
go install github.com/limehawk/lazyreno/cmd/lazyreno@latest
```

Or download a binary from [Releases](https://github.com/limehawk/lazyreno/releases).

## Configuration

Create `~/.config/lazyreno/config.toml`:

```toml
[renovate]
url = "https://your-renovate-instance.example.com"

[github]
owner = "your-github-org"

[ui]
refresh_interval = "30s"
```

Set secrets via environment variables:

```bash
export LAZYRENO_RENOVATE_SECRET="your-renovate-api-secret"
export LAZYRENO_GITHUB_TOKEN="your-github-token"  # or GITHUB_TOKEN
```

### Renovate CE API Setup

Enable the APIs on your Renovate CE instance:

```
MEND_RNV_API_ENABLED=true
MEND_RNV_API_ENABLE_SYSTEM=true
MEND_RNV_API_ENABLE_REPORTING=true
MEND_RNV_API_ENABLE_JOBS=true
RENOVATE_REPOSITORY_CACHE=enabled
```

## Keybindings

| Key | Action |
|-----|--------|
| `1`-`4` | Switch tabs |
| `j`/`k` | Navigate up/down |
| `Tab` | Cycle panel focus |
| `m` | Merge selected PR |
| `M` | Merge all safe PRs |
| `c` | Close PR + delete branch |
| `s` | Trigger Renovate sync |
| `r` | Retry failed job |
| `R` | Refresh all data |
| `?` | Help |
| `q` | Quit |

## License

MIT
