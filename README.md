# rig

**Ryan's jig** — a personal TUI toolkit for custom dev workflows, built with Go and [Charm](https://charm.sh/).

So far this has been heavily developed by Claude Code's Opus 4.6, but I am getting it to teach me go in the process. Check out **[LEARNING.md](LEARNING.md)**

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/ryan-rushton/rig/main/install.sh | sh
```

This downloads the latest release binary to `~/.local/bin`. Override the location with `INSTALL_DIR`:

```bash
INSTALL_DIR=/usr/local/bin sh install.sh
```

Or install from source:

```bash
go install github.com/ryan-rushton/rig@latest
```

Check your version:

```bash
rig --version
```

## Usage

Launch the home screen and pick a tool interactively:

```bash
rig
```

Or run a tool directly:

```bash
rig git-branch    # also: rig gb
rig test-changed  # also: rig tc
```

## Tools

### `git-branch` / `gb`

Interactive git branch manager — checkout, rename, create, and delete branches.

| Key         | Action                                            |
| ----------- | ------------------------------------------------- |
| `j` / `↓`   | Move down                                         |
| `k` / `↑`   | Move up                                           |
| `enter`     | Checkout selected branch                          |
| `e`         | Rename selected branch                            |
| `c`         | Create a new branch                               |
| `dd`        | Delete branch (first `d` stages, second confirms) |
| `r`         | Refresh branch list                               |
| `esc` / `q` | Back / quit                                       |

When renaming a branch that has a remote tracking branch, you'll be prompted whether to also rename it on the remote. Git errors (e.g. uncommitted changes blocking a checkout) are shown in a dismissible splash.

### `test-changed` / `tc`

Detects files changed vs the merge base with the default branch and runs affected tests. Supports Go and Bazel projects.

| Key         | Action               |
| ----------- | -------------------- |
| `j` / `↓`   | Move down            |
| `k` / `↑`   | Move up              |
| `enter`     | Run tests            |
| `r`         | Re-run / refresh     |
| `esc` / `q` | Back / quit          |

## Development

```bash
go run .            # run home screen
go run . gb         # run a tool directly
go build ./...      # verify all packages compile
go test ./...       # run tests
```

Releases use [CalVer](https://calver.org/) (`YYYY.MM.DD`) and are published automatically on every push to main via GoReleaser. Binaries are built for darwin and linux (amd64 + arm64).

See [CLAUDE.md](./CLAUDE.md) for architecture notes and instructions on adding new tools.
