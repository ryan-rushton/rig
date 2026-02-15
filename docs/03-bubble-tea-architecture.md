# Bubble Tea: The Elm Architecture

## The Core Loop

Bubble Tea implements the [Elm Architecture](https://guide.elm-lang.org/architecture/), which has three parts:

```
┌──────────────────────────────────────────┐
│                                          │
│  Model ──→ View() ──→ Terminal Output    │
│    ↑                        │            │
│    │                        ↓            │
│  Update() ←── Messages ← User Input     │
│                                          │
└──────────────────────────────────────────┘
```

1. **Model** — the data (a struct with all your state)
2. **Update** — handles messages and returns a new model + optional command
3. **View** — renders the model to a string for the terminal

The framework calls `View()` after every `Update()` to re-render. You never mutate state and re-render manually — you return new state and the framework handles it.

### Why This Architecture?

- **Predictable** — state only changes in `Update`, making bugs easy to trace
- **Testable** — send a message, check the resulting model, no mocking needed
- **No race conditions** — `Update` runs on a single goroutine; async work happens in commands and results come back as messages

---

## The `tea.Model` Interface

```go
type Model interface {
    Init() Cmd
    Update(Msg) (Model, Cmd)
    View() string
}
```

### Init

Called once when the program starts. Returns a command to run initial async work:

```go
func (m Model) Init() tea.Cmd {
    return tea.Batch(fetchBranches, m.spinner.Tick, m.stopwatch.Start())
}
```

The home screen checks for updates on init:

```go
func (m Model) Init() tea.Cmd {
    if m.version == "dev" {
        return nil          // No update check in dev mode
    }
    return checkForUpdate(m.version)
}
```

### Update

The heart of the program. Receives a message, returns updated model and optional command:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case branchesLoadedMsg:
        if msg.err != nil {
            m = showError(m, msg.err)
        } else {
            m.state = stateBrowse
            m.branches = msg.branches
        }
        return m, nil       // nil command = nothing more to do

    case tea.KeyMsg:
        return m.handleKey(msg)
    }
    return m, nil
}
```

Important: `Update` returns `(tea.Model, tea.Cmd)`, not `(Model, tea.Cmd)`. The return type is the **interface**, not your concrete type. Go handles this implicitly — your `Model` satisfies `tea.Model`, so returning it works.

### View

Pure function — takes the model state and returns a string to display:

```go
func (m Model) View() string {
    var content string

    switch m.state {
    case stateLoading:
        elapsed := fmt.Sprintf("%.2fs", m.stopwatch.Elapsed().Seconds())
        content = m.spinner.View() + " " + styles.Dimmed.Render(m.processingMsg) +
            "  " + styles.Subtitle.Render(elapsed)

    case stateBrowse:
        content = styles.Title.Render("Git Branch Manager") + "\n\n"
        for i, b := range m.branches {
            // ... render each branch
        }
        content += "\n" + m.help.View(browseKeys)  // Uses bubbles/help component
    }

    return styles.Box.Render(content)
}
```

`View()` should **never** have side effects. It's called frequently and should only read from the model, never modify it.

---

Next: [Messages and Commands](./04-messages-and-commands.md)
