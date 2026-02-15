# Composing Models

Rig demonstrates several ways to compose Bubble Tea models together.

## Parent-Child: The App Model

`app.Model` owns the currently active screen and delegates all messages to it:

```go
type Model struct {
    current tea.Model    // Could be home.Model or any tool model
    version string
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case messages.BackMsg:
        // Tool wants to go back → swap to home screen
        h := home.New(m.version)
        m.current = h
        return m, h.Init()

    case messages.ToolSelectedMsg:
        // Home selected a tool → swap to tool screen
        if t := registry.Get(msg.ID); t != nil {
            tool := t.New()
            m.current = tool
            return m, tool.Init()
        }
    }

    // All other messages → delegate to the active screen
    updated, cmd := m.current.Update(msg)
    m.current = updated
    return m, cmd
}

func (m Model) View() string {
    return m.current.View()    // Just render whatever screen is active
}
```

This is the **composite model pattern**. The parent handles navigation messages and delegates everything else.

---

## The Standalone Wrapper

When running a tool directly from the CLI, there's no home screen to go back to. The `standalone` wrapper solves this:

```go
type standalone struct {
    inner tea.Model
}

func Standalone(m tea.Model) tea.Model {
    return standalone{inner: m}
}

func (s standalone) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if _, ok := msg.(BackMsg); ok {
        return s, tea.Quit     // Convert "go back" into "exit"
    }
    m, cmd := s.inner.Update(msg)
    s.inner = m
    return s, cmd
}
```

This is the **decorator pattern** — it wraps a model and changes one behaviour (BackMsg) while passing everything else through.

---

## Embedded Components

The tools use several components from the `charmbracelet/bubbles` library:

```go
type Model struct {
    // ...
    input     textinput.Model    // Text input for renaming/creating
    spinner   spinner.Model      // Animated loading indicator
    stopwatch stopwatch.Model    // Elapsed time tracking
    help      help.Model         // Formatted help text from key bindings
}
```

Each component follows the same pattern — initialize in `New()`, route messages through `Update()`, render with `View()`:

```go
func New() Model {
    ti := textinput.New()
    ti.CharLimit = 200

    s := spinner.New()
    s.Spinner = spinner.MiniDot
    s.Style = styles.Selected

    sw := stopwatch.NewWithInterval(100 * time.Millisecond)

    return Model{input: ti, spinner: s, stopwatch: sw}
}
```

When the tool is in edit mode, it delegates key events to the input component:

```go
case stateEdit:
    switch msg.String() {
    case "enter":
        // Handle submission
    case "esc":
        m.state = stateBrowse
    default:
        // Let the text input handle all other keys
        var inputCmd tea.Cmd
        m.input, inputCmd = m.input.Update(msg)
        return m, inputCmd
    }
```

For the spinner and stopwatch, messages are routed after the type switch when in async states:

```go
if m.state == stateLoading || m.state == stateProcessing {
    var cmd tea.Cmd
    var cmds []tea.Cmd
    m.spinner, cmd = m.spinner.Update(msg)
    cmds = append(cmds, cmd)
    m.stopwatch, cmd = m.stopwatch.Update(msg)
    cmds = append(cmds, cmd)
    return m, tea.Batch(cmds...)
}
```

Notice the pattern: intercept special keys (`enter`, `esc`), and delegate everything else to the sub-component. The sub-component returns an updated copy and possibly a command.

---

Next: [Async Operations](./06-async-operations.md)
