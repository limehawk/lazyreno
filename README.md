# lazyreno

A lazy-style TUI dashboard for self-hosted [Renovate CE](https://github.com/mend/renovate-ce-ee). Monitor PRs, repos, jobs, and system status from your terminal.

## Features

- **3-column layout** — Sidebar repos, PR table + detail, system status + jobs
- **PR management** — Merge, bulk-merge safe (minor/patch), close PRs with branch cleanup
- **Job monitoring** — Live view of running/pending Renovate jobs
- **System status** — Renovate version, uptime, queue depth
- Vim-style navigation, context-sensitive footer hints, keyboard-driven

## Install

### AUR (Arch Linux)

```bash
yay -S lazyreno
```

### From source

```bash
git clone https://github.com/limehawk/lazyreno.git
cd lazyreno
cargo build --release
cp target/release/lazyreno ~/.local/bin/
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
accent = "cyan"
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

### Navigation

| Key | Action |
|-----|--------|
| `j`/`k` | Navigate up/down |
| `h`/`l` | Switch panel |
| `g`/`G` | Jump to top/bottom |
| `Ctrl+u`/`Ctrl+d` | Half page up/down |
| `Tab`/`Shift+Tab` | Cycle panel focus |
| `Enter` | Select / drill in |

### Actions

| Key | Action |
|-----|--------|
| `m` | Merge selected PR |
| `M` | Merge all safe PRs |
| `x` | Close PR + delete branch |
| `o` | Open PR in browser |
| `s` | Trigger Renovate sync |
| `P` | Purge finished jobs |

### Other

| Key | Action |
|-----|--------|
| `a` | All repos overlay |
| `?` | Help |
| `R` | Refresh |
| `q` | Quit |

## License

MIT
