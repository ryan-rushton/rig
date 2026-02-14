# rig

**Ryan's jig** — a personal TUI toolkit for custom dev workflows, built with Go and [Charm](https://charm.sh/).

## Install

```bash
go install github.com/ryan-rushton/rig@latest
```

Or build from source:

```bash
git clone https://github.com/ryan-rushton/rig
cd rig
go install .
```

## Usage

Launch the home screen and pick a tool interactively:

```bash
rig
```

Or run a tool directly:

```bash
rig git-branch   # also: rig gb
```

## Tools

### `git-branch` / `gb`

Interactive editor for git branch names. Rename a local branch and optionally push the rename to the remote in one flow.

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `e` / `enter` | Rename selected branch |
| `r` | Refresh branch list |
| `esc` / `q` | Back / quit |

When renaming a branch that has a remote tracking branch, you'll be prompted whether to also delete the old remote branch and push the renamed one.

## Development

```bash
go run .            # run home screen
go run . gb         # run a tool directly
go build ./...      # verify all packages compile
```

See [CLAUDE.md](./CLAUDE.md) for architecture notes and instructions on adding new tools.
