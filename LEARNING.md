# Learning Go and Bubble Tea with Rig

This guide teaches Go fundamentals and the Bubble Tea TUI framework using the rig codebase as a living example. Every concept is illustrated with real code from this repo.

## Guides

### Go Language

1. **[Go Fundamentals](docs/01-go-fundamentals.md)** — entry points, value types, zero values, modules, packages, and imports
2. **[Types, Functions, and Error Handling](docs/02-types-functions-errors.md)** — structs, interfaces, receivers, closures, slices, `init()`, and concurrency

### Bubble Tea Framework

3. **[The Elm Architecture](docs/03-bubble-tea-architecture.md)** — the Model-Update-View loop and the `tea.Model` interface
4. **[Messages and Commands](docs/04-messages-and-commands.md)** — how events flow and async work is triggered
5. **[Composing Models](docs/05-composing-models.md)** — parent-child delegation, the standalone wrapper, and embedded components
6. **[Async Operations](docs/06-async-operations.md)** — the startAsync pattern, bubbles spinner/stopwatch components, and the error splash overlay

### Supporting Tools

7. **[Styling and CLI](docs/07-styling-and-cli.md)** — Lipgloss terminal styling and Cobra CLI routing

### Practical

8. **[Testing](docs/08-testing.md)** — test conventions, table-driven tests, and testing Bubble Tea models
9. **[Gotchas](docs/09-gotchas.md)** — value semantics bugs, closure capture, nil slices, and other pitfalls
10. **[Adding a New Tool](docs/10-adding-a-tool.md)** — step-by-step recipe that ties all the concepts together

## Further Reading

- [A Tour of Go](https://go.dev/tour/) — official interactive tutorial
- [Effective Go](https://go.dev/doc/effective_go) — idiomatic Go patterns
- [Bubble Tea docs](https://github.com/charmbracelet/bubbletea) — framework docs and examples
- [Lipgloss docs](https://github.com/charmbracelet/lipgloss) — styling reference
- [Cobra docs](https://cobra.dev/) — CLI framework
- [Go by Example](https://gobyexample.com/) — concise examples of Go features
