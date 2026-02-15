# Go Fundamentals

This covers core Go language concepts, illustrated with code from the rig codebase.

## The Entry Point

Every Go program starts with `package main` and a `main()` function. Here's rig's:

```go
// main.go
package main

import "github.com/ryan-rushton/rig/cmd"

var version = "dev"

func main() {
    cmd.SetVersion(version)
    cmd.Execute()
}
```

Key things to notice:
- **`package main`** — this is special. Only `package main` can have a `main()` function, and it's what `go run` and `go build` look for
- **`var version = "dev"`** — a package-level variable. At build time, the real version is injected via linker flags (`-ldflags`), but for local development it defaults to `"dev"`
- **Exported vs unexported** — Go uses capitalisation instead of keywords like `public`/`private`. `SetVersion` is exported (capital S), so other packages can call it. If it were `setVersion`, only the `cmd` package itself could use it

## Value Types vs Reference Types

Go is a **value-type language**. When you assign a struct to a new variable or pass it to a function, you get a **copy**. This is different from languages like JavaScript or Python where objects are references.

```go
// This creates a COPY of the model — changes to `m2` don't affect `m`
m := Model{cursor: 0}
m2 := m
m2.cursor = 5
// m.cursor is still 0!
```

This has important consequences for Bubble Tea (covered in [gotchas](./08-gotchas.md)).

## Zero Values

In Go, variables always have a value — there's no `undefined` or `null` for basic types. Uninitialized variables get their **zero value**:

| Type | Zero Value |
|------|-----------|
| `int` | `0` |
| `string` | `""` |
| `bool` | `false` |
| `[]T` (slice) | `nil` |
| `*T` (pointer) | `nil` |
| `struct` | all fields are zero-valued |

This is used throughout rig. For example, `home.New()` doesn't need to explicitly set `cursor: 0` — it's already zero:

```go
func New(version string) Model {
    return Model{version: version}
    // cursor is 0, updateTag is "", updating is false — all zero values
}
```

---

## Project Structure and Modules

### Go Modules (`go.mod`)

The `go.mod` file is Go's equivalent of `package.json`. It defines the module path and dependencies:

```
module github.com/ryan-rushton/rig

go 1.24.2

require (
    github.com/charmbracelet/bubbles v1.0.0
    github.com/charmbracelet/bubbletea v1.3.10
    github.com/charmbracelet/lipgloss v1.1.0
    github.com/spf13/cobra v1.10.2
)
```

- **`module github.com/ryan-rushton/rig`** — this is the module's import path. All internal packages are imported relative to this (e.g. `github.com/ryan-rushton/rig/internal/styles`)
- **`go.sum`** — the lockfile (like `package-lock.json`), auto-generated
- **`go mod tidy`** — adds missing and removes unused dependencies

### Directory Layout

```
rig/
├── main.go                      # Entry point
├── go.mod                       # Dependencies
├── cmd/                         # CLI commands (cobra)
│   ├── root.go
│   ├── gitbranch.go
│   └── testchanged.go
└── internal/                    # Private packages
    ├── app/                     # Top-level screen manager
    ├── home/                    # Home screen
    ├── messages/                # Shared message types
    ├── registry/                # Tool registry
    ├── styles/                  # Shared lipgloss styles
    ├── updater/                 # Self-update logic
    └── tools/
        ├── gitbranch/           # Git branch manager tool
        └── testchanged/         # Changed-test runner tool
```

The **`internal/`** directory is special in Go: packages inside `internal/` can only be imported by code within the same module. This is enforced by the compiler — it's Go's way of keeping implementation details private.

---

## Packages and Imports

### Package Basics

Every `.go` file starts with a `package` declaration. All files in the same directory must use the same package name:

```go
// internal/styles/styles.go
package styles
```

The package name is typically the same as the directory name. When you import a package, you use the full module path but refer to it by its package name:

```go
import "github.com/ryan-rushton/rig/internal/styles"

// Usage: styles.Title, styles.Purple, etc.
```

### Import Groups

Go convention (enforced by `goimports`) is to group imports into three blocks, separated by blank lines:

```go
import (
    // 1. Standard library
    "fmt"
    "strings"
    "time"

    // 2. Third-party packages
    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    // 3. Local packages (same module)
    "github.com/ryan-rushton/rig/internal/messages"
    "github.com/ryan-rushton/rig/internal/styles"
)
```

### Import Aliases

Notice `tea "github.com/charmbracelet/bubbletea"` — this creates an alias. Instead of writing `bubbletea.Model`, you write `tea.Model`. This is common when a package name is long or when you want a more descriptive name.

### Unused Imports

Go **won't compile** if you have an unused import. This keeps code clean but can be annoying during development. Tools like `goimports` automatically add and remove imports.

---

Next: [Types, Functions, and Error Handling](./02-types-functions-errors.md)
