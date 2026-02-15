# Quick Reference: Adding a New Tool

This ties together most concepts from the other guides. To add a new tool:

## 1. Create `internal/tools/<name>/model.go`

```go
package mytool

import (
    "github.com/charmbracelet/bubbles/help"
    "github.com/charmbracelet/bubbles/key"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "github.com/ryan-rushton/rig/internal/messages"
    "github.com/ryan-rushton/rig/internal/registry"
    "github.com/ryan-rushton/rig/internal/styles"
)

func init() {
    registry.Register(registry.Tool{
        ID:          "my-tool",
        Name:        "my-tool",
        Description: "Does something useful",
        New:         func() tea.Model { return New() },
    })
}

type keyMap struct {
    bindings []key.Binding
}

func (k keyMap) ShortHelp() []key.Binding  { return k.bindings }
func (k keyMap) FullHelp() [][]key.Binding { return nil }

var defaultKeys = keyMap{bindings: []key.Binding{
    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc/q", "back")),
}}

type Model struct {
    help help.Model
    // your state here
}

func New() Model {
    h := help.New()
    h.Styles.ShortKey = lipgloss.NewStyle().Foreground(styles.DimGray).Italic(true).Bold(true)
    h.Styles.ShortDesc = styles.Help
    h.Styles.ShortSeparator = styles.Help

    return Model{help: h}
}

func (m Model) Init() tea.Cmd {
    return nil // or start async work
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c":
            return m, tea.Quit
        case "q", "esc":
            return m, func() tea.Msg { return messages.BackMsg{} }
        }
    }
    return m, nil
}

func (m Model) View() string {
    content := styles.Title.Render("My Tool") + "\n\n"
    content += "Hello, world!\n"
    content += "\n" + m.help.View(defaultKeys)
    return styles.Box.Render(content)
}
```

## 2. Create `cmd/mytool.go`

```go
package cmd

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/spf13/cobra"

    "github.com/ryan-rushton/rig/internal/messages"
    mytool "github.com/ryan-rushton/rig/internal/tools/mytool"
)

func init() {
    rootCmd.AddCommand(&cobra.Command{
        Use:   "my-tool",
        Short: "Does something useful",
        RunE: func(cmd *cobra.Command, args []string) error {
            p := tea.NewProgram(messages.Standalone(mytool.New()), tea.WithAltScreen())
            _, err := p.Run()
            return err
        },
    })
}
```

## 3. That's it

There is no step 3. The `init()` registration means the tool automatically:
- Appears on the home screen
- Works via `rig my-tool` from the CLI
- Navigates back correctly (BackMsg → home screen or exit, depending on launch mode)

No changes to `app.go`, `home.go`, or any existing code needed. The registry pattern handles it all.
