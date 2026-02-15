package testchanged

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/stopwatch"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ryan-rushton/rig/internal/messages"
	"github.com/ryan-rushton/rig/internal/registry"
	"github.com/ryan-rushton/rig/internal/styles"
)

func init() {
	registry.Register(registry.Tool{
		ID:          "test-changed",
		Name:        "test-changed",
		Description: "Run tests for changed files vs merge base",
		New:         func() tea.Model { return New() },
	})
}

type viewState int

const (
	stateLoading viewState = iota
	stateBrowse
	stateRunning
	stateResults
)

type keyMap struct {
	bindings []key.Binding
}

func (k keyMap) ShortHelp() []key.Binding  { return k.bindings }
func (k keyMap) FullHelp() [][]key.Binding { return nil }

var browseEmptyKeys = keyMap{bindings: []key.Binding{
	key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc/q", "back")),
}}

var browseKeys = keyMap{bindings: []key.Binding{
	key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run")),
	key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc/q", "back")),
}}

var resultsKeys = keyMap{bindings: []key.Binding{
	key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rerun")),
	key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc/q", "back")),
}}

var dismissKeys = keyMap{bindings: []key.Binding{
	key.NewBinding(key.WithKeys("any"), key.WithHelp("any key", "dismiss")),
}}

// Messages used by this tool.
type targetsLoadedMsg struct {
	runner  string
	targets []string
	err     error
}

type testDoneMsg struct {
	err error
}

// discoveredTarget groups a target with which runner found it.
type discoveredTarget struct {
	runner string
	target string
}

// Model is the test-changed TUI model.
type Model struct {
	state      viewState
	targets    []discoveredTarget
	cursor     int
	output     []string
	maxOutput  int
	viewport   viewport.Model
	errSplash  string
	spinner    spinner.Model
	stopwatch  stopwatch.Model
	help       help.Model
	loadingMsg string
	exitCode   int
	runnerName string
	finishedIn time.Duration
}

func New() Model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = styles.Selected

	sw := stopwatch.NewWithInterval(100 * time.Millisecond)

	vp := viewport.New(80, 30)

	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(styles.DimGray).Italic(true).Bold(true)
	h.Styles.ShortDesc = styles.Help
	h.Styles.ShortSeparator = styles.Help

	return Model{
		state:      stateLoading,
		maxOutput:  500,
		spinner:    s,
		stopwatch:  sw,
		viewport:   vp,
		help:       h,
		loadingMsg: "Detecting default branch...",
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadTargets, m.spinner.Tick, m.stopwatch.Start())
}

// startAsync transitions into a waiting state, resets the timer, and
// batches the command with the spinner/stopwatch. Must be used as: return startAsync(m, ...).
func startAsync(m Model, state viewState, label string, cmd tea.Cmd) (Model, tea.Cmd) {
	m.state = state
	m.loadingMsg = label
	return m, tea.Batch(cmd, m.spinner.Tick, m.stopwatch.Reset(), m.stopwatch.Start())
}

func showError(m Model, err error) Model {
	m.state = stateBrowse
	m.errSplash = err.Error()
	return m
}

func loadTargets() tea.Msg {
	branch, err := detectDefaultBranch()
	if err != nil {
		return targetsLoadedMsg{err: fmt.Errorf("detect default branch: %w", err)}
	}

	base, err := mergeBase(branch)
	if err != nil {
		return targetsLoadedMsg{err: fmt.Errorf("merge base: %w", err)}
	}

	files, err := changedFiles(base)
	if err != nil {
		return targetsLoadedMsg{err: fmt.Errorf("changed files: %w", err)}
	}

	var targets []string
	runnerName := ""
	for _, r := range allRunners() {
		if r.Detect() {
			found := r.FindTargets(files)
			if len(found) > 0 {
				runnerName = r.Name()
				targets = found
				break
			}
		}
	}

	return targetsLoadedMsg{runner: runnerName, targets: targets}
}

type testBatchMsg struct {
	lines []string
	err   error
}

// streamLines returns a tea.Cmd that sends output lines one at a time,
// allowing the TUI to render progressively.
func streamLines(runner string, targets []string) tea.Cmd {
	return func() tea.Msg {
		var r TestRunner
		for _, candidate := range allRunners() {
			if candidate.Name() == runner {
				r = candidate
				break
			}
		}
		if r == nil {
			return testDoneMsg{err: fmt.Errorf("runner %q not found", runner)}
		}

		cmd := r.RunTests(targets)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return testDoneMsg{err: err}
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			return testDoneMsg{err: err}
		}

		if err := cmd.Start(); err != nil {
			return testDoneMsg{err: err}
		}

		combined := io.MultiReader(stdout, stderr)
		var lines []string
		scanner := bufio.NewScanner(combined)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		exitErr := cmd.Wait()
		return testBatchMsg{lines: lines, err: exitErr}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Error splash intercepts all key presses and clears itself.
	if m.errSplash != "" {
		if _, ok := msg.(tea.KeyMsg); ok {
			m.errSplash = ""
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Reserve space for the header (status line + blank line) and footer (help line).
		headerHeight := 4
		m.viewport.Width = msg.Width - 6 // account for box border + padding
		m.viewport.Height = msg.Height - headerHeight - 6
		return m, nil

	case targetsLoadedMsg:
		if msg.err != nil {
			m = showError(m, msg.err)
			return m, nil
		}
		m.state = stateBrowse
		m.runnerName = msg.runner
		m.targets = make([]discoveredTarget, len(msg.targets))
		for i, t := range msg.targets {
			m.targets[i] = discoveredTarget{runner: msg.runner, target: t}
		}
		return m, nil

	case testBatchMsg:
		m.output = append(m.output, msg.lines...)
		m.state = stateResults
		m.finishedIn = m.stopwatch.Elapsed()
		if msg.err != nil {
			m.exitCode = 1
		} else {
			m.exitCode = 0
		}
		m.viewport.SetContent(colorizeOutput(m.output))
		m.viewport.GotoBottom()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Route viewport messages when viewing results.
	if m.state == stateResults {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	// Route spinner and stopwatch messages when in async states.
	if m.state == stateLoading || m.state == stateRunning {
		var cmd tea.Cmd
		var cmds []tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		m.stopwatch, cmd = m.stopwatch.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateBrowse:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			return m, func() tea.Msg { return messages.BackMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.targets)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.targets) > 0 {
				targets := make([]string, len(m.targets))
				for i, t := range m.targets {
					targets[i] = t.target
				}
				m.output = nil
				return startAsync(m, stateRunning, "Running tests...", streamLines(m.runnerName, targets))
			}
		case "r":
			m.targets = nil
			m.cursor = 0
			return startAsync(m, stateLoading, "Detecting default branch...", loadTargets)
		}

	case stateRunning:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case stateResults:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			return m, func() tea.Msg { return messages.BackMsg{} }
		case "r":
			m.targets = nil
			m.cursor = 0
			m.output = nil
			return startAsync(m, stateLoading, "Detecting default branch...", loadTargets)
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

const tailLines = 30

func colorizeOutput(lines []string) string {
	var b strings.Builder
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "ok"):
			b.WriteString(styles.Success.Render(line))
		case strings.HasPrefix(line, "FAIL"):
			b.WriteString(styles.Err.Render(line))
		case strings.Contains(line, "--- PASS"):
			b.WriteString(styles.Success.Render(line))
		case strings.Contains(line, "--- FAIL"):
			b.WriteString(styles.Err.Render(line))
		default:
			b.WriteString(line)
		}
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m Model) View() string {
	// Error splash takes over the whole view; any key will clear it.
	if m.errSplash != "" {
		content := styles.Title.Render("Error") + "\n\n"
		content += styles.Err.Render(m.errSplash) + "\n"
		content += "\n" + m.help.View(dismissKeys)
		return styles.Box.
			BorderForeground(styles.Red).
			Render(content)
	}

	var content string

	switch m.state {
	case stateLoading:
		elapsed := fmt.Sprintf("%.2fs", m.stopwatch.Elapsed().Seconds())
		content = m.spinner.View() + " " + styles.Dimmed.Render(m.loadingMsg) +
			"  " + styles.Subtitle.Render(elapsed)

	case stateBrowse:
		content = styles.Title.Render("Test Changed Files") + "\n\n"

		if len(m.targets) == 0 {
			content += styles.Dimmed.Render("No affected test targets found.") + "\n"
			content += "\n" + m.help.View(browseEmptyKeys)
		} else {
			content += styles.Subtitle.Render(
				fmt.Sprintf("Found %d target(s) via %s runner:", len(m.targets), m.runnerName),
			) + "\n\n"

			for i, t := range m.targets {
				cursor := "  "
				nameStyle := styles.Dimmed
				if i == m.cursor {
					cursor = styles.Selected.Render("> ")
					nameStyle = styles.Selected
				}
				content += cursor + nameStyle.Render(t.target) + "\n"
			}

			content += "\n" + m.help.View(browseKeys)
		}

	case stateRunning:
		elapsed := fmt.Sprintf("%.2fs", m.stopwatch.Elapsed().Seconds())
		content = m.spinner.View() + " " + styles.Dimmed.Render(m.loadingMsg) +
			"  " + styles.Subtitle.Render(elapsed) + "\n\n"

		// Show tail of output collected so far.
		if len(m.output) > 0 {
			start := len(m.output) - tailLines
			if start < 0 {
				start = 0
			}
			for _, line := range m.output[start:] {
				content += line + "\n"
			}
		}

	case stateResults:
		elapsed := fmt.Sprintf("%.2fs", m.finishedIn.Seconds())
		if m.exitCode == 0 {
			content = styles.Success.Render("✓ Tests passed") + "  " +
				styles.Subtitle.Render(elapsed) + "\n\n"
		} else {
			content = styles.Err.Render("✗ Tests failed") + "  " +
				styles.Subtitle.Render(elapsed) + "\n\n"
		}

		content += m.viewport.View()

		if len(m.output) > m.viewport.Height {
			content += "\n" + styles.Dimmed.Render(
				fmt.Sprintf("(%d%% — ↑↓/jk to scroll)", int(m.viewport.ScrollPercent()*100)),
			)
		}

		content += "\n" + m.help.View(resultsKeys)
	}

	return styles.Box.Render(content)
}
