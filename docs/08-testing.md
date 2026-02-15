# Testing in Go

## Test File Convention

Test files end in `_test.go` and live alongside the code they test:

```
internal/tools/gitbranch/
├── model.go          # Implementation
├── git.go            # Git operations
└── model_test.go     # Tests
```

## Test Functions

Test functions start with `Test` and take `*testing.T`:

```go
func TestBrowseNavigation(t *testing.T) {
    m := modelWithBranches(twoBranches)

    // Press "down"
    r, _ := m.Update(key('j'))
    m = r.(Model)

    if m.cursor != 1 {
        t.Errorf("cursor = %d, want 1", m.cursor)
    }
}
```

## Table-Driven Tests

Go convention for testing multiple cases:

```go
func TestCursorBounds(t *testing.T) {
    tests := []struct {
        name   string
        key    tea.KeyMsg
        start  int
        want   int
    }{
        {"down from 0", key('j'), 0, 1},
        {"up from 0 stays", key('k'), 0, 0},
        {"down from last stays", key('j'), 1, 1},
        {"up from 1", key('k'), 1, 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            m := modelWithBranches(twoBranches)
            m.cursor = tt.start
            r, _ := m.Update(tt.key)
            got := r.(Model).cursor
            if got != tt.want {
                t.Errorf("cursor = %d, want %d", got, tt.want)
            }
        })
    }
}
```

## Testing Bubble Tea Models

The pattern for testing Bubble Tea models:

1. **Create a model** with known state
2. **Send a message** via `Update()`
3. **Type-assert** the result back to your concrete type
4. **Check the state** changed as expected

```go
// Helper to create test messages
func key(r rune) tea.KeyMsg {
    return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// Helper to create a model in a known state
func modelWithBranches(branches []Branch) Model {
    m := New()
    m.state = stateBrowse
    m.branches = branches
    return m
}

// The test
func TestEditMode(t *testing.T) {
    m := modelWithBranches(twoBranches)
    r, _ := m.Update(key('e'))
    m = r.(Model)

    if m.state != stateEdit {
        t.Fatalf("state = %v, want stateEdit", m.state)
    }
    if m.input.Value() != "main" {
        t.Errorf("input = %q, want %q", m.input.Value(), "main")
    }
}
```

## Running Tests

```bash
go test ./...                           # All tests
go test ./internal/tools/gitbranch/     # One package
go test -v ./internal/tools/gitbranch/  # Verbose output
go test -run TestBrowse ./...           # Tests matching a pattern
```

---

Next: [Gotchas and Lessons Learned](./09-gotchas.md)
