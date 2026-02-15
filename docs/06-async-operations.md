# Async Operations

All async operations in rig follow the same pattern:

```
User action → startAsync() → [stateProcessing + spinner] → result message → new state
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
    m.startedAt = time.Now()
    m.spinnerFrame = 0
    return m, tea.Batch(cmd, tick())    // Run the command AND start the spinner
}
```

**3. The spinner ticks update the animation:**
```go
case tickMsg:
    if m.state == stateLoading || m.state == stateProcessing {
        m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
        return m, tick()    // Keep ticking
    }
    return m, nil           // Stop ticking when no longer in a loading state
```

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

## The Spinner

The spinner is a manual animation using Unicode braille characters:

```go
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
```

Each `tickMsg` (every 100ms) advances to the next frame. This is a simpler alternative to using the `spinner` component from the `bubbles` library.

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
    content += "\n" + styles.Help.Render("any key to dismiss")
    return styles.Box.BorderForeground(styles.Red).Render(content)
}
```

This pattern avoids adding error states to the state machine while giving a clean UX.

---

Next: [Styling and CLI](./07-styling-and-cli.md)
