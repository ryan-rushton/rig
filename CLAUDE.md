# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -o ./bin/rig .   # build binary
go run . [args]            # run without building
go install .               # install to $GOPATH/bin
go build ./...             # verify all packages compile
go mod tidy                # sync dependencies
```

There are no tests yet. When added, run them with `go test ./...`.

## Architecture

**rig** is a personal TUI monorepo built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). It has two entry modes:

- `rig` — launches the home screen TUI where tools are listed and selected interactively
- `rig <tool>` — launches a tool directly, bypassing the home screen

### Screen management

`internal/app/app.go` holds the top-level `Model` that owns the currently active screen. Screen transitions happen via two message types in `internal/messages/messages.go`:

- `ToolSelectedMsg{ID}` — emitted by the home screen when a tool is selected; `app.Model` swaps the active model to the matching tool
- `BackMsg{}` — emitted by tools when the user quits; `app.Model` swaps back to a fresh home screen

When a tool is launched directly via CLI (`cmd/gitbranch.go`), it is wrapped in `messages.Standalone(...)`, which converts `BackMsg` into `tea.Quit` instead of navigating.

### Adding a new tool

1. Create `internal/tools/<name>/model.go` — implement `tea.Model`; send `messages.BackMsg{}` on quit/back
2. Create `cmd/<name>.go` — cobra subcommand calling `tea.NewProgram(messages.Standalone(<name>.New()), tea.WithAltScreen())`
3. Add an entry to `tools` in `internal/home/home.go`
4. Add a case to the `ToolSelectedMsg` switch in `internal/app/app.go`

### Shared packages

- `internal/styles/` — lipgloss colour palette and reusable styles; use these rather than defining new colours inline
- `internal/messages/` — shared message types; add new cross-tool messages here
