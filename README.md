<div align="center">

# lazyreno

A lazy-style TUI dashboard for self-hosted [Renovate CE](https://github.com/mend/renovate-ce-ee).

Monitor PRs, repos, jobs, and system status — all from your terminal.

![demo](demo.gif)

[![AUR](https://img.shields.io/aur/version/lazyreno)](https://aur.archlinux.org/packages/lazyreno)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Rust](https://img.shields.io/badge/rust-2024_edition-orange.svg)](https://www.rust-lang.org/)

</div>

---

## Why

Renovate CE generates a flood of dependency PRs across repos. Clicking through GitHub to review, merge, and close them is tedious. lazyreno gives you a single keyboard-driven view of everything — PRs grouped by repo, bulk-merge safe updates, monitor job queues, trigger syncs — without leaving the terminal.

## Features

- **3-panel bento layout** — repo sidebar, PR table + detail, system status + jobs
- **Bulk merge** — merge all safe (minor/patch, mergeable, checks passing) PRs in one keystroke
- **PR management** — merge, close with branch cleanup, open in browser
- **Job monitoring** — live view of running/pending Renovate jobs with queue depth
- **System status** — Renovate version, uptime, last finished job
- **Vim navigation** — `hjkl`, `g`/`G`, `Ctrl+u`/`Ctrl+d`, context-sensitive hints
- **Fuzzy repo filter** — `a` opens an overlay to search across all repos
- **1Password integration** — `op://` secret references resolved automatically
- **Auto-refresh** — configurable polling interval

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

### Releases

Pre-built binaries available on the [releases page](https://github.com/limehawk/lazyreno/releases).

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

Secrets can be [1Password secret references](https://developer.1password.com/docs/cli/secret-references/) — resolved automatically via the `op` CLI:

```toml
[renovate]
secret = "op://Dev/Renovate CE API/credential"

[github]
token = "op://Dev/My GitHub Token/token"
```

### Environment variables

Override config values with environment variables:

```bash
export LAZYRENO_RENOVATE_SECRET="your-renovate-api-secret"
export LAZYRENO_GITHUB_TOKEN="your-github-token"  # or GITHUB_TOKEN
```

### Renovate CE API setup

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
|---|---|
| `j` / `k` | Navigate up / down |
| `h` / `l` | Switch panel |
| `g` / `G` | Jump to top / bottom |
| `Ctrl+u` / `Ctrl+d` | Half page up / down |
| `Tab` / `Shift+Tab` | Cycle panel focus |
| `Enter` | Select / drill in |

### Actions

| Key | Action |
|---|---|
| `m` | Merge selected PR |
| `M` | Merge all safe PRs in repo |
| `x` | Close PR + delete branch |
| `o` | Open PR in browser |
| `s` | Trigger Renovate sync |
| `P` | Purge finished jobs |

### General

| Key | Action |
|---|---|
| `a` | All repos overlay (fuzzy filter) |
| `R` | Force refresh |
| `?` | Help |
| `q` | Quit |

## License

[MIT](LICENSE)
