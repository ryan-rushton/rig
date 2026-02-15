# Gotchas and Lessons Learned

## 1. Value Semantics in Bubble Tea

This is the most important gotcha. Since Go structs are values, `Update` works with copies:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // m is a COPY of the model
    m.cursor++      // Modifies the copy
    return m, nil   // Returns the modified copy
}
```

This is correct. The framework takes your returned copy and uses it going forward.

**The bug to avoid:** Don't use pointer receivers on Bubble Tea models unless you understand the implications:

```go
// BAD — pointer method modifies the original while Go evaluates the return
func (m *Model) badStartAsync(cmd tea.Cmd) tea.Cmd {
    m.state = stateProcessing
    return tea.Batch(cmd, tick())
}

// This is buggy:
return m, m.badStartAsync(someCmd)
// Go evaluates `m` (copy) BEFORE `m.badStartAsync` runs,
// so the returned model has the OLD state!
```

That's why `startAsync` is a free function:

```go
// GOOD — free function, no pointer confusion
func startAsync(m Model, state viewState, label string, cmd tea.Cmd) (Model, tea.Cmd) {
    m.state = state
    return m, tea.Batch(cmd, tick())
}

// Clean usage:
return startAsync(m, stateProcessing, "Loading...", fetchBranches)
```

---

## 2. Capturing Loop Variables in Closures

When creating closures inside loops, be careful about variable capture:

```go
// CAREFUL — in older Go versions, `b` was shared across iterations
for _, b := range branches {
    go func() {
        process(b)    // Might get the wrong branch!
    }()
}

// SAFE — capture explicitly
for _, b := range branches {
    b := b    // Shadow with a new variable
    go func() {
        process(b)    // Correct — each goroutine has its own copy
    }()
}
```

Note: Go 1.22+ fixed this for `for` loops, but it's good to understand the pattern.

---

## 3. `nil` Slices vs Empty Slices

```go
var s []string       // nil slice — s == nil, len(s) == 0
s = []string{}       // empty slice — s != nil, len(s) == 0
s = make([]string, 0) // also empty — s != nil
```

They behave the same for most operations (`len`, `append`, `range`), but they're different for JSON encoding and nil checks. In rig, nil slices are used by default (zero values).

---

## 4. Always Handle the Error

Go won't compile if you ignore a return value that you've captured. But it will let you ignore return values entirely:

```go
os.Remove("file.txt")              // Compiles but ignores error — BAD
err := os.Remove("file.txt")      // Won't compile unless you use err
_ = os.Remove("file.txt")         // Explicitly ignoring — OK if intentional
```

---

## 5. The Blank Identifier `_`

When you don't need a value, use `_`:

```go
_, err := p.Run()          // Don't need the model, just the error
for _, b := range branches // Don't need the index
if _, ok := msg.(KeyMsg)   // Don't need the value, just checking the type
```

---

## 6. String Building for Views

Go strings are immutable. Building strings with `+=` in a loop creates many allocations. For small UIs (like rig's views), this is fine. For large-scale string building, use `strings.Builder`:

```go
var b strings.Builder
for _, line := range lines {
    b.WriteString(line)
    b.WriteByte('\n')
}
return b.String()
```

Rig uses `+=` because the views are small and the simplicity is worth it.
