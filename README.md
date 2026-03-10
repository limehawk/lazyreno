# lazyreno

A lazy-style TUI dashboard for self-hosted [Renovate CE](https://github.com/mend/renovate-ce-ee).

## Features

- **PRs tab** — View and merge Renovate PRs across all repos. Bulk-merge safe (minor/patch) PRs.
- **Repos tab** — Browse all repos managed by Renovate.
- **Jobs tab** — Monitor the Renovate job queue. Retry failed jobs.
- **Status tab** — System overview, trigger sync, purge failed jobs.
- Vim-style navigation, fuzzy filtering, keyboard-driven.

## Install

### AUR (Arch Linux)

```bash
yay -S lazyreno
```

### Go

```bash
go install github.com/limehawk/lazyreno/cmd/lazyreno@latest
```

Or download a binary from [Releases](https://github.com/limehawk/lazyreno/releases).

## Configuration

Create `~/.config/lazyreno/config.toml`:

```toml
[renovate]
url = "https://your-renovate-instance.example.com"
secret = "your-renovate-api-secret"

[github]
owner = "your-github-org"
token = "your-github-token"

[ui]
refresh_interval = "30s"
```

### 1Password

Secrets can be [1Password secret references](https://developer.1password.com/docs/cli/secret-references/) — they'll be resolved automatically via the `op` CLI:

```toml
[renovate]
secret = "op://Dev/Renovate CE API/credential"

[github]
token = "op://Dev/My GitHub Token/token"
```

### Environment variables

Secrets can also be set via environment variables, which override config file values:

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
