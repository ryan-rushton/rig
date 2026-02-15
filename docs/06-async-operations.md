# Async Operations

All async operations in rig follow the same pattern:

```
User action → startAsync() → [stateProcessing + spinner/stopwatch] → result message → new state
```

## Step by Step

**1. User triggers an action:**
```go
case "enter":
    return startAsync(m, stateProcessing, "Switching branch...", m.cmdCheckout(b.Name))
```

**2. `startAsync` sets up the waiting state:**
```go
func startAsync(m Model, state viewState, label string, cmd tea.Cmd) (Model, tea.Cmd) {
    m.state = state
    m.processingMsg = label
    return m, tea.Batch(cmd, m.spinner.Tick, m.stopwatch.Reset(), m.stopwatch.Start())
}
```

The `tea.Batch` starts the git command, the spinner animation, and the elapsed stopwatch all concurrently. The `spinner` and `stopwatch` are components from the `charmbracelet/bubbles` library — they manage their own internal tick messages.

**3. The spinner and stopwatch update themselves:**
```go
// After the type switch in Update, route messages to sub-components:
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

Unlike the old hand-rolled approach (manual tick messages, frame counters), the bubbles components handle their own timing internally. You just route unhandled messages to them when in an async state.

**4. The async result arrives:**
```go
case checkoutResultMsg:
    if msg.err != nil {
        m = showError(m, msg.err)
        return m, nil
    }
    // Success — reload branches to show the updated state
    return startAsync(m, stateLoading, "Loading branches...", fetchBranches)
```

---

## The Spinner and Stopwatch

Both are initialized in `New()`:

```go
s := spinner.New()
s.Spinner = spinner.MiniDot       // Braille dot animation
s.Style = styles.Selected         // Purple bold

sw := stopwatch.NewWithInterval(100 * time.Millisecond)  // Updates 10x/sec
```

In `View()`, rendering is straightforward:

```go
case stateLoading:
    elapsed := fmt.Sprintf("%.2fs", m.stopwatch.Elapsed().Seconds())
    content = m.spinner.View() + " " + styles.Dimmed.Render(m.processingMsg) +
        "  " + styles.Subtitle.Render(elapsed)
```

`m.spinner.View()` returns the current animation frame. `m.stopwatch.Elapsed()` returns the `time.Duration` since the stopwatch was last started.

---

## Error Handling: The Splash Pattern

Errors don't get their own state — they're shown as an overlay:

```go
func showError(m Model, err error) Model {
    m.state = stateBrowse         // Stay in browse state
    m.errSplash = err.Error()     // Set the overlay text
    return m
}
```

In `Update`, the error splash intercepts ALL key presses:

```go
if m.errSplash != "" {
    if _, ok := msg.(tea.KeyMsg); ok {
        m.errSplash = ""      // Any key dismisses
        return m, nil
    }
}
```

In `View`, the splash takes over the entire display:

```go
if m.errSplash != "" {
    content := styles.Title.Render("Error") + "\n\n"
    content += styles.Err.Render(m.errSplash) + "\n"
    content += "\n" + m.help.View(dismissKeys)
    return styles.Box.BorderForeground(styles.Red).Render(content)
}
```

This pattern avoids adding error states to the state machine while giving a clean UX.

---

Next: [Styling and CLI](./07-styling-and-cli.md)
