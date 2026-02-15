# Types, Functions, and Error Handling

## Structs

Structs are Go's primary data type — like classes but without inheritance. Here's the git branch model:

```go
// internal/tools/gitbranch/model.go
type Model struct {
    state         viewState
    branches      []Branch
    cursor        int
    input         textinput.Model    // An embedded Bubble Tea component
    editing       Branch
    didRemote     bool
    result        string
    errSplash     string
    confirmIdx    int
    startedAt     time.Time
    spinnerFrame  int
    processingMsg string
    deleteStaged    bool
    deleteStagedIdx int
}
```

Key observations:
- All fields are **lowercase** (unexported) — only code within the `gitbranch` package can access them directly
- `textinput.Model` is a struct from another package embedded as a field
- `[]Branch` is a **slice** (dynamic array) of Branch structs

## Custom Types

Go lets you define new types based on existing ones. This is used for state enums:

```go
type viewState int

const (
    stateLoading viewState = iota   // = 0
    stateBrowse                     // = 1
    stateEdit                       // = 2
    stateCreate                     // = 3
    stateConfirmRemote              // = 4
    stateProcessing                 // = 5
    stateResult                     // = 6
)
```

- **`type viewState int`** — creates a new type that's backed by `int` but is a distinct type. You can't accidentally mix it with regular ints
- **`iota`** — auto-incrementing constant generator. First value is 0, each subsequent const increments by 1

## Interfaces

Interfaces in Go are **implicit** — a type satisfies an interface if it has all the required methods. You don't need to declare `implements`.

The most important interface in rig is `tea.Model`:

```go
// From the bubbletea package
type Model interface {
    Init() Cmd
    Update(Msg) (Model, Cmd)
    View() string
}
```

Every screen in rig implements these three methods, so they all satisfy `tea.Model` without ever writing `implements tea.Model`. The compiler checks it for you when you try to use the struct where a `tea.Model` is expected.

The `TestRunner` interface in testchanged is another example:

```go
type TestRunner interface {
    Name() string
    Detect() bool
    FindTargets(files []string) []string
    RunTests(targets []string) *exec.Cmd
}
```

Both `GoRunner` and `BazelRunner` structs implement all four methods, so they both satisfy `TestRunner` automatically.

---

## Functions, Methods, and Receivers

### Regular Functions

```go
func tick() tea.Cmd {
    return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}
```

### Methods (Value Receivers)

Methods are functions attached to a type via a **receiver**. In rig, most methods use value receivers:

```go
func (m Model) View() string {
    // m is a COPY of the Model
    // This method cannot modify the original Model
}
```

The `(m Model)` before the function name is the receiver. With a **value receiver**, `m` is a copy. This is the correct choice for Bubble Tea models because the framework expects `Update` to return a new model value.

### Methods (Pointer Receivers)

Pointer receivers get a reference to the original value:

```go
func (m *Model) SetName(name string) {
    m.Name = name  // Modifies the original Model
}
```

Rig deliberately avoids pointer receivers on Bubble Tea models (see [gotchas](./08-gotchas.md) for why).

### Free Functions vs Methods

Some functions are deliberately **not** methods. `startAsync` is a great example:

```go
// Free function — takes Model as a parameter, returns modified copy
func startAsync(m Model, state viewState, label string, cmd tea.Cmd) (Model, tea.Cmd) {
    m.state = state
    m.processingMsg = label
    m.startedAt = time.Now()
    m.spinnerFrame = 0
    return m, tea.Batch(cmd, tick())
}
```

Why not a method? Because of how it's called:

```go
// Correct — startAsync receives a copy of m, modifies it, returns the modified copy
return startAsync(m, stateProcessing, "Renaming...", someCmd)
```

If this were a pointer method `m.startAsync(...)`, there would be a subtle bug (explained in [gotchas](./08-gotchas.md)).

### Multiple Return Values

Go functions can return multiple values. This is used everywhere for error handling:

```go
func getBranches() ([]Branch, error) {
    // Returns both the result AND an error
}

// Caller handles both
branches, err := getBranches()
if err != nil {
    return nil, err
}
```

---

## Error Handling

Go doesn't have exceptions. Instead, functions return errors as values:

```go
func renameBranch(oldName, newName string) error {
    var buf bytes.Buffer
    cmd := exec.Command("git", "branch", "-m", oldName, newName)
    cmd.Stderr = &buf
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("rename branch: %s", strings.TrimSpace(buf.String()))
    }
    return nil   // nil means "no error"
}
```

Key patterns:

1. **Check immediately** — always check `err != nil` right after the call
2. **Wrap errors** — `fmt.Errorf("context: %s", err)` adds context about what operation failed
3. **Return nil** — returning `nil` for the error means success

### The `error` Interface

`error` is just an interface with one method:

```go
type error interface {
    Error() string
}
```

Any type with an `Error() string` method satisfies it. This is used in tests:

```go
type errForTest string

func (e errForTest) Error() string { return string(e) }

// Usage in tests:
msg := branchesLoadedMsg{err: errForTest("git failed")}
```

---

## Control Flow and Type Switches

### Regular Switch

Go's `switch` doesn't need `break` statements — cases don't fall through by default:

```go
switch msg.String() {
case "ctrl+c":
    return m, tea.Quit
case "q", "esc":                          // Multiple values in one case
    return m, func() tea.Msg { return messages.BackMsg{} }
case "up", "k":
    if m.cursor > 0 {
        m.cursor--
    }
case "down", "j":
    if m.cursor < len(m.branches)-1 {
        m.cursor++
    }
}
```

### Type Switches

Type switches check the dynamic type of an interface value. This is how Bubble Tea dispatches messages:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {     // msg is REBOUND to its concrete type in each case
    case tickMsg:
        // msg is now type tickMsg, not tea.Msg
        m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
        return m, tick()

    case branchesLoadedMsg:
        // msg is now type branchesLoadedMsg — you can access msg.branches, msg.err
        if msg.err != nil {
            m = showError(m, msg.err)
        }

    case tea.KeyMsg:
        // msg is now type tea.KeyMsg — you can call msg.String()
        return m.handleKey(msg)
    }
    return m, nil
}
```

The `msg := msg.(type)` syntax rebinds `msg` to its concrete type within each `case` branch. This is called a **type assertion** within a switch.

### Type Assertions (standalone)

Outside of switches, you can check a single type:

```go
if _, ok := msg.(tea.KeyMsg); ok {
    // msg is a tea.KeyMsg
    m.errSplash = ""
    return m, nil
}
```

The `ok` boolean tells you whether the assertion succeeded, avoiding a panic if the type doesn't match.

---

## Closures and First-Class Functions

Functions are first-class values in Go — they can be stored in variables, passed as arguments, and returned from other functions.

### Closures as Commands

Bubble Tea commands are functions that return a message. Closures capture variables from their enclosing scope:

```go
func (m Model) cmdRenameLocal(newName string) tea.Cmd {
    oldName := m.editing.Name       // Capture current value
    return func() tea.Msg {         // Return a closure
        err := renameBranch(oldName, newName)
        return renameResultMsg{localOk: err == nil, err: err}
    }
}
```

Why capture `oldName` in a local variable? Because `m` is a value receiver (copy), but by the time the closure runs (asynchronously), you want to be sure you have the right value. Capturing into a local variable makes the intent explicit.

### Function Fields in Structs

The registry stores a function as a struct field:

```go
type Tool struct {
    ID          string
    Name        string
    Description string
    New         func() tea.Model    // A function that creates a new tool model
}
```

This is used to create tool instances lazily:

```go
if t := registry.Get(msg.ID); t != nil {
    tool := t.New()    // Call the stored function to create a model
    m.current = tool
}
```

### Anonymous Functions as Messages

Emitting a message often uses an inline anonymous function:

```go
return m, func() tea.Msg {
    return messages.BackMsg{}
}
```

This wraps the message in a `tea.Cmd` (which has the signature `func() tea.Msg`). The function executes asynchronously and delivers the message back to the `Update` loop.

---

## Slices and Iteration

### Slices

Slices are Go's dynamic arrays. They're used throughout rig:

```go
var tools []Tool                    // Declare a nil slice
tools = append(tools, newTool)     // Append returns a new slice

branches := make([]Branch, 0, 10)  // Pre-allocate capacity
```

- `nil` slices are valid — `len(nil)` is `0`, and `append` works on them
- `append` may allocate a new backing array, so always use its return value

### Range Loops

```go
for i, b := range m.branches {
    // i = index, b = copy of the element
    cursor := "  "
    if i == m.cursor {
        cursor = styles.Selected.Render("> ")
    }
}
```

If you only need the index:
```go
for i := range tools {
    if tools[i].ID == id {
        return &tools[i]    // Return pointer to the actual element
    }
}
```

Notice the registry uses `for i := range tools` and then `tools[i]` instead of `for _, t := range tools`. This is because `range` gives you a **copy** of each element, but the registry wants to return a **pointer** to the original element.

---

## The `init()` Function

`init()` is a special function that runs automatically when a package is loaded. Each package can have multiple `init()` functions. They run after all package-level variables are initialized.

Rig uses `init()` for tool self-registration:

```go
// internal/tools/gitbranch/model.go
func init() {
    registry.Register(registry.Tool{
        ID:          "git-branch",
        Name:        "git-branch",
        Description: "Rename git branches (local and remote)",
        New:         func() tea.Model { return New() },
    })
}
```

This pattern means:
1. Each tool registers itself just by being imported
2. No central "list of all tools" to maintain
3. Adding a new tool doesn't require modifying the registry

The tools get imported via the `cmd/` package, which imports each tool's package for its cobra subcommand. Go guarantees `init()` runs before `main()`.

---

## Concurrency: Goroutines and Commands

### Goroutines

Go has lightweight concurrency via goroutines (started with `go`). However, rig doesn't use goroutines directly — it uses Bubble Tea's command system instead, which manages concurrency for you.

### `tea.Cmd`

A command is just a function that returns a message:

```go
type Cmd func() Msg
```

When you return a `tea.Cmd` from `Update`, Bubble Tea runs it in a **goroutine** for you and delivers the result back to `Update`. This keeps async operations simple and safe.

```go
// This function returns a tea.Cmd — a function that Bubble Tea will run async
func fetchBranches() tea.Msg {
    branches, err := getBranches()              // This blocks (runs git command)
    return branchesLoadedMsg{branches, err}     // Result delivered to Update
}

// In Init():
func (m Model) Init() tea.Cmd {
    return tea.Batch(fetchBranches, tick())    // Run both concurrently
}
```

### `tea.Batch`

`tea.Batch` runs multiple commands concurrently:

```go
return m, tea.Batch(cmd, tick())
```

This starts both the git command and the spinner ticker at the same time. Results arrive independently as messages to `Update`.

---

Next: [Bubble Tea Architecture](./03-bubble-tea-architecture.md)
