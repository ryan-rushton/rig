# Styling and CLI

## Lipgloss: Terminal Styling

[Lipgloss](https://github.com/charmbracelet/lipgloss) is the styling library from the Charm team. It works like CSS for the terminal.

### Defining Styles

Styles are created with a builder pattern:

```go
// internal/styles/styles.go
var (
    Pink    = lipgloss.Color("#FF2E97")
    Green   = lipgloss.Color("#39FF14")
    Red     = lipgloss.Color("#FF3131")
    Cyan    = lipgloss.Color("#00F0FF")

    Title = lipgloss.NewStyle().
        Bold(true).
        Foreground(Cyan)

    Help = lipgloss.NewStyle().
        Foreground(DimGray).
        Italic(true)

    Box = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(Cyan).
        Padding(1, 2)
)
```

### Using Styles

Styles are applied with `.Render()`:

```go
styles.Title.Render("Git Branch Manager")    // Bold cyan text
styles.Err.Render("something went wrong")    // Red text
styles.Box.Render(content)                   // Rounded border with padding
```

### Inline Style Overrides

You can create one-off variations:

```go
styles.Box.
    BorderForeground(styles.Red).    // Override just the border colour
    Render(content)
```

### Centralised Styles

Rig keeps all styles in `internal/styles/styles.go`. This means:
- Consistent colours across all tools
- Easy to change the colour scheme in one place
- No magic colour strings scattered through the codebase

---

## Cobra: CLI Routing

[Cobra](https://github.com/spf13/cobra) handles command-line argument parsing and subcommands.

### Root Command

```go
// cmd/root.go
var rootCmd = &cobra.Command{
    Use:   "rig",
    Short: "Ryan's Jig TUI toolkit",
    RunE: func(cmd *cobra.Command, args []string) error {
        p := tea.NewProgram(app.New(version), tea.WithAltScreen())
        _, err := p.Run()
        return err
    },
}
```

When you run `rig` with no subcommand, `RunE` fires and launches the full TUI.

### Subcommands

Each tool has a subcommand registered via `init()`:

```go
// cmd/gitbranch.go
func init() {
    rootCmd.AddCommand(&cobra.Command{
        Use:     "git-branch",
        Aliases: []string{"gb"},
        Short:   "Edit git branch names",
        RunE: func(cmd *cobra.Command, args []string) error {
            p := tea.NewProgram(
                messages.Standalone(gitbranch.New()),
                tea.WithAltScreen(),
            )
            _, err := p.Run()
            return err
        },
    })
}
```

Note the `Aliases: []string{"gb"}` — this lets users type `rig gb` as a shortcut.

### `tea.WithAltScreen()`

This option tells Bubble Tea to use the terminal's alternate screen buffer. When the program exits, the terminal restores its previous content. Without this, TUI output would remain in the scrollback.

---

Next: [Testing](./08-testing.md)
