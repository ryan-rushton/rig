# Messages and Commands

## Messages

Messages are how events flow through the system. They can be any Go type:

```go
// A message with data
type branchesLoadedMsg struct {
    branches []Branch
    err      error
}

// A simple message — no fields needed
type deleteResultMsg struct{ err error }

// Messages from other packages
type BackMsg struct{}
type ToolSelectedMsg struct{ ID string }
```

### Built-in Messages

Bubble Tea provides some built-in message types:

- **`tea.KeyMsg`** — keyboard input. Use `msg.String()` to get the key name (`"enter"`, `"ctrl+c"`, `"q"`, etc.)
- **`tea.WindowSizeMsg`** — terminal was resized
- **`tea.QuitMsg`** — the program should exit (returned by `tea.Quit`)

---

## Commands

Commands are how you do async work. A command is a function that returns a message:

```go
// Simple command — wraps a synchronous call
func fetchBranches() tea.Msg {
    branches, err := getBranches()
    return branchesLoadedMsg{branches: branches, err: err}
}

// Command factory — returns a command (closure) that captures parameters
func (m Model) cmdCheckout(name string) tea.Cmd {
    return func() tea.Msg {
        return checkoutResultMsg{err: checkoutBranch(name)}
    }
}
```

Note the difference:
- `fetchBranches` has the signature `func() tea.Msg` — it **is** a `tea.Cmd`
- `cmdCheckout` **returns** a `tea.Cmd` — it's a factory that creates commands with captured parameters

### Special Commands

```go
tea.Quit         // Exits the program
tea.Batch(a, b)  // Runs multiple commands concurrently
tea.Tick(d, fn)  // Sends a message after a duration
```

---

Next: [Composing Models](./05-composing-models.md)
