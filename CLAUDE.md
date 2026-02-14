# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build ./...             # verify all packages compile
go test ./...              # run all tests
golangci-lint run ./...    # lint (v2, config in .golangci.yml)
go run . [args]            # run without building
go install .               # install to $GOPATH/bin
go mod tidy                # sync dependencies
```

Imports are formatted with `goimports` using `github.com/ryan-rushton/rig` as the local prefix (three groups: stdlib, third-party, local). VS Code is configured to run this on save.

## Architecture

**rig** is a personal TUI monorepo built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). It has two entry modes:

- `rig` — launches the home screen TUI where tools are listed and selected interactively
- `rig <tool>` — launches a tool directly, bypassing the home screen

CLI routing uses [cobra](https://github.com/spf13/cobra) (`cmd/` package). The version string is injected via `-ldflags` at build time (see `.goreleaser.yml`); it defaults to `"dev"` for local builds.

### Screen management

`internal/app/app.go` holds the top-level `Model` that owns the currently active screen. Screen transitions happen via two message types in `internal/messages/messages.go`:

- `ToolSelectedMsg{ID}` — emitted by the home screen when a tool is selected; `app.Model` swaps the active model to the matching tool
- `BackMsg{}` — emitted by tools when the user quits; `app.Model` swaps back to a fresh home screen

When a tool is launched directly via CLI (`cmd/gitbranch.go`), it is wrapped in `messages.Standalone(...)`, which converts `BackMsg` into `tea.Quit` instead of navigating.

### Async git operations

Tools that call git use Bubble Tea commands (async `tea.Cmd`) so the UI never blocks. A shared `startAsync` helper in the tool model sets up a ticker-driven spinner + elapsed timer (`s.ms` format). All git errors are shown in a dismissible splash overlay (`errSplash` field) rather than a separate error state — any keypress clears it and returns to browse.

**Important:** `startAsync` must be a free function returning `(Model, tea.Cmd)`, not a pointer-receiver method. Using `return m, m.startAsync(...)` with a pointer receiver is a bug — Go evaluates the `m` copy before the pointer mutation runs.

### Adding a new tool

1. Create `internal/tools/<name>/model.go` — implement `tea.Model`; send `messages.BackMsg{}` on quit/back
2. Create `cmd/<name>.go` — cobra subcommand calling `tea.NewProgram(messages.Standalone(<name>.New()), tea.WithAltScreen())`
3. Add an entry to `tools` in `internal/home/home.go`
4. Add a case to the `ToolSelectedMsg` switch in `internal/app/app.go`

### Shared packages

- `internal/styles/` — lipgloss color palette and reusable styles; use these rather than defining new colors inline
- `internal/messages/` — shared message types and the `Standalone` wrapper

## CI/CD

- **CI** (`.github/workflows/ci.yml`): builds and tests on PRs and pushes to main
- **Release** (`.github/workflows/release.yml`): on push to main, generates a CalVer tag (`YYYY.MM.DD`), creates a GitHub release with cross-compiled binaries via GoReleaser
- **Dependabot** (`.github/dependabot.yml`): weekly updates for Go modules and GitHub Actions
