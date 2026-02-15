package testchanged

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

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

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

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
	state        viewState
	targets      []discoveredTarget
	cursor       int
	output       []string
	maxOutput    int
	scrollOffset int
	errSplash    string
	startedAt    time.Time
	spinnerFrame int
	loadingMsg   string
	exitCode     int
	runnerName   string
	finishedIn   time.Duration
}

func New() Model {
	return Model{
		state:      stateLoading,
		maxOutput:  500,
		startedAt:  time.Now(),
		loadingMsg: "Detecting default branch...",
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadTargets, tick())
}

// startAsync transitions into a waiting state, resets the timer, and
// batches the command with the ticker. Must be used as: return startAsync(m, ...).
func startAsync(m Model, state viewState, label string, cmd tea.Cmd) (Model, tea.Cmd) {
	m.state = state
	m.loadingMsg = label
	m.startedAt = time.Now()
	m.spinnerFrame = 0
	return m, tea.Batch(cmd, tick())
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
	case tickMsg:
		if m.state == stateLoading || m.state == stateRunning {
			m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
			return m, tick()
		}
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
		m.finishedIn = time.Since(m.startedAt)
		if msg.err != nil {
			m.exitCode = 1
		} else {
			m.exitCode = 0
		}
		// Auto-scroll to bottom.
		m.scrollOffset = maxScroll(len(m.output))
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
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
				m.scrollOffset = 0
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
		case "up", "k":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "down", "j":
			m.scrollOffset++
			max := maxScroll(len(m.output))
			if m.scrollOffset > max {
				m.scrollOffset = max
			}
		case "r":
			m.targets = nil
			m.cursor = 0
			m.output = nil
			return startAsync(m, stateLoading, "Detecting default branch...", loadTargets)
		}
	}

	return m, nil
}

const visibleLines = 30

func maxScroll(total int) int {
	if total <= visibleLines {
		return 0
	}
	return total - visibleLines
}

func (m Model) elapsed() string {
	return fmt.Sprintf("%.2fs", time.Since(m.startedAt).Seconds())
}

func (m Model) View() string {
	// Error splash takes over the whole view; any key will clear it.
	if m.errSplash != "" {
		content := styles.Title.Render("Error") + "\n\n"
		content += styles.Err.Render(m.errSplash) + "\n"
		content += "\n" + styles.Help.Render("any key to dismiss")
		return styles.Box.
			BorderForeground(styles.Red).
			Render(content)
	}

	var content string

	switch m.state {
	case stateLoading:
		spinner := styles.Selected.Render(spinnerFrames[m.spinnerFrame])
		content = spinner + " " + styles.Dimmed.Render(m.loadingMsg) +
			"  " + styles.Subtitle.Render(m.elapsed())

	case stateBrowse:
		content = styles.Title.Render("Test Changed Files") + "\n\n"

		if len(m.targets) == 0 {
			content += styles.Dimmed.Render("No affected test targets found.") + "\n"
			content += "\n" + styles.Help.Render("r refresh  esc/q back")
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

			content += "\n" + styles.Help.Render("enter run  r refresh  esc/q back")
		}

	case stateRunning:
		spinner := styles.Selected.Render(spinnerFrames[m.spinnerFrame])
		content = spinner + " " + styles.Dimmed.Render(m.loadingMsg) +
			"  " + styles.Subtitle.Render(m.elapsed()) + "\n\n"

		// Show tail of output collected so far.
		if len(m.output) > 0 {
			start := len(m.output) - visibleLines
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

		if len(m.output) > 0 {
			end := m.scrollOffset + visibleLines
			if end > len(m.output) {
				end = len(m.output)
			}
			start := m.scrollOffset
			if start > end {
				start = end
			}
			for _, line := range m.output[start:end] {
				// Colorize pass/fail lines.
				switch {
				case strings.HasPrefix(line, "ok"):
					content += styles.Success.Render(line) + "\n"
				case strings.HasPrefix(line, "FAIL"):
					content += styles.Err.Render(line) + "\n"
				case strings.Contains(line, "--- PASS"):
					content += styles.Success.Render(line) + "\n"
				case strings.Contains(line, "--- FAIL"):
					content += styles.Err.Render(line) + "\n"
				default:
					content += line + "\n"
				}
			}

			if len(m.output) > visibleLines {
				content += "\n" + styles.Dimmed.Render(
					fmt.Sprintf("(%d/%d lines — ↑↓/jk to scroll)", end, len(m.output)),
				)
			}
		}

		content += "\n" + styles.Help.Render("r rerun  esc/q back")
	}

	return styles.Box.Render(content)
}
