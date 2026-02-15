package gitbranch

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ryan-rushton/rig/internal/messages"
	"github.com/ryan-rushton/rig/internal/registry"
	"github.com/ryan-rushton/rig/internal/styles"
)

func init() {
	registry.Register(registry.Tool{
		ID:          "git-branch",
		Name:        "git-branch",
		Description: "Rename git branches (local and remote)",
		New:         func() tea.Model { return New() },
	})
}

type viewState int

const (
	stateLoading viewState = iota
	stateBrowse
	stateEdit
	stateCreate
	stateConfirmRemote
	stateProcessing
	stateResult
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type branchesLoadedMsg struct {
	branches []Branch
	err      error
}

type renameResultMsg struct {
	localOk  bool
	remoteOk bool
	err      error
}

type deleteResultMsg struct{ err error }
type createResultMsg struct{ err error }
type checkoutResultMsg struct{ err error }

// Model is the git branch editor TUI model.
type Model struct {
	state         viewState
	branches      []Branch
	cursor        int
	input         textinput.Model
	editing       Branch
	didRemote     bool
	result        string
	errSplash     string // non-empty = show error splash; any key dismisses it
	confirmIdx    int    // 0 = yes, 1 = no
	startedAt     time.Time
	spinnerFrame  int
	processingMsg string
	// delete staging — first d marks, second d confirms
	deleteStaged    bool
	deleteStagedIdx int
}

func New() Model {
	ti := textinput.New()
	ti.CharLimit = 200
	ti.Width = 50

	return Model{
		state:     stateLoading,
		input:     ti,
		startedAt: time.Now(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchBranches, tick())
}

func fetchBranches() tea.Msg {
	branches, err := getBranches()
	return branchesLoadedMsg{branches: branches, err: err}
}

// startAsync transitions into a waiting state, resets the timer, and
// batches the git command with the ticker. Returns the updated model
// and batched command — must be used as: return startAsync(m, ...).
func startAsync(m Model, state viewState, label string, cmd tea.Cmd) (Model, tea.Cmd) {
	m.state = state
	m.processingMsg = label
	m.startedAt = time.Now()
	m.spinnerFrame = 0
	return m, tea.Batch(cmd, tick())
}

// showError keeps the model in stateBrowse but sets the splash message.
func showError(m Model, err error) Model {
	m.state = stateBrowse
	m.errSplash = err.Error()
	return m
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
		if m.state == stateLoading || m.state == stateProcessing {
			m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
			return m, tick()
		}
		return m, nil

	case branchesLoadedMsg:
		if msg.err != nil {
			m = showError(m, msg.err)
		} else {
			m.state = stateBrowse
			m.branches = msg.branches
			if m.cursor >= len(m.branches) && len(m.branches) > 0 {
				m.cursor = len(m.branches) - 1
			}
		}
		return m, nil

	case renameResultMsg:
		if msg.err != nil {
			if msg.localOk {
				// Local succeeded but remote failed — show partial result.
				m.state = stateResult
				remote, _ := splitUpstream(m.editing.Upstream)
				m.result = styles.Success.Render("✓") + " Renamed " +
					styles.Dimmed.Render(m.editing.Name) + " → " +
					styles.Selected.Render(m.input.Value()) + "\n" +
					styles.Err.Render("✗") + " Remote " +
					styles.Remote.Render(remote) + " update failed: " +
					styles.Err.Render(msg.err.Error())
			} else {
				m = showError(m, msg.err)
			}
		} else {
			m.state = stateResult
			lines := []string{
				styles.Success.Render("✓") + " Renamed " +
					styles.Dimmed.Render(m.editing.Name) + " → " +
					styles.Selected.Render(m.input.Value()),
			}
			if m.didRemote && msg.remoteOk {
				remote, _ := splitUpstream(m.editing.Upstream)
				lines = append(lines,
					styles.Success.Render("✓")+" Updated remote "+
						styles.Remote.Render(remote+"/"+m.input.Value()),
				)
			}
			m.result = strings.Join(lines, "\n")
		}
		return m, nil

	case deleteResultMsg:
		if msg.err != nil {
			m = showError(m, msg.err)
		} else {
			m.state = stateResult
			m.result = styles.Success.Render("✓") + " Deleted " +
				styles.Dimmed.Render(m.editing.Name)
		}
		return m, nil

	case createResultMsg:
		if msg.err != nil {
			m = showError(m, msg.err)
		} else {
			m.state = stateResult
			m.result = styles.Success.Render("✓") + " Created " +
				styles.Selected.Render(m.input.Value())
		}
		return m, nil

	case checkoutResultMsg:
		if msg.err != nil {
			m = showError(m, msg.err)
			return m, nil
		}
		// Reload so the current-branch indicator updates.
		return startAsync(m, stateLoading, "Loading branches...", fetchBranches)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateBrowse:
		// Any key other than d clears delete staging.
		if msg.String() != "d" {
			m.deleteStaged = false
		}

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
			if m.cursor < len(m.branches)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.branches) > 0 {
				b := m.branches[m.cursor]
				if b.IsCurrent {
					break
				}
				m.editing = b
				return startAsync(m, stateProcessing, "Switching branch...", m.cmdCheckout(b.Name))
			}
		case "e":
			if len(m.branches) > 0 {
				m.editing = m.branches[m.cursor]
				m.input.SetValue(m.editing.Name)
				m.input.Focus()
				m.input.CursorEnd()
				m.state = stateEdit
			}
		case "c":
			m.input.SetValue("")
			m.input.Focus()
			m.state = stateCreate
		case "d":
			if len(m.branches) == 0 {
				break
			}
			b := m.branches[m.cursor]
			if b.IsCurrent {
				break
			}
			if m.deleteStaged && m.deleteStagedIdx == m.cursor {
				m.editing = b
				m.deleteStaged = false
				return startAsync(m, stateProcessing, "Deleting branch...", m.cmdDelete(b.Name))
			}
			m.deleteStaged = true
			m.deleteStagedIdx = m.cursor
		case "r":
			return startAsync(m, stateLoading, "Loading branches...", fetchBranches)
		}

	case stateEdit:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.state = stateBrowse
			m.input.Blur()
			return m, nil
		case "enter":
			newName := strings.TrimSpace(m.input.Value())
			if newName == "" || newName == m.editing.Name {
				m.state = stateBrowse
				m.input.Blur()
				return m, nil
			}
			if m.editing.HasRemote {
				m.state = stateConfirmRemote
				m.confirmIdx = 0
				return m, nil
			}
			return startAsync(m, stateProcessing, "Renaming branch...", m.cmdRenameLocal(newName))
		default:
			var inputCmd tea.Cmd
			m.input, inputCmd = m.input.Update(msg)
			return m, inputCmd
		}

	case stateCreate:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.state = stateBrowse
			m.input.Blur()
			return m, nil
		case "enter":
			newName := strings.TrimSpace(m.input.Value())
			if newName == "" || m.branchExists(newName) {
				return m, nil
			}
			return startAsync(m, stateProcessing, "Creating branch...", m.cmdCreate(newName))
		default:
			var inputCmd tea.Cmd
			m.input, inputCmd = m.input.Update(msg)
			return m, inputCmd
		}

	case stateConfirmRemote:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.state = stateBrowse
			return m, nil
		case "left", "h", "shift+tab":
			m.confirmIdx = 0
		case "right", "l", "tab":
			m.confirmIdx = 1
		case "y":
			newName := strings.TrimSpace(m.input.Value())
			m.didRemote = true
			return startAsync(m, stateProcessing, "Renaming branch...", m.cmdRenameAll(newName))
		case "n":
			newName := strings.TrimSpace(m.input.Value())
			m.didRemote = false
			return startAsync(m, stateProcessing, "Renaming branch...", m.cmdRenameLocal(newName))
		case "enter", " ":
			newName := strings.TrimSpace(m.input.Value())
			if m.confirmIdx == 0 {
				m.didRemote = true
				return startAsync(m, stateProcessing, "Renaming branch...", m.cmdRenameAll(newName))
			}
			m.didRemote = false
			return startAsync(m, stateProcessing, "Renaming branch...", m.cmdRenameLocal(newName))
		}

	case stateResult:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			return m, func() tea.Msg { return messages.BackMsg{} }
		default:
			m.cursor = 0
			return startAsync(m, stateLoading, "Loading branches...", fetchBranches)
		}
	}

	return m, nil
}

func (m Model) branchExists(name string) bool {
	for _, b := range m.branches {
		if b.Name == name {
			return true
		}
	}
	return false
}

func (m Model) cmdCheckout(name string) tea.Cmd {
	return func() tea.Msg {
		return checkoutResultMsg{err: checkoutBranch(name)}
	}
}

func (m Model) cmdRenameLocal(newName string) tea.Cmd {
	oldName := m.editing.Name
	return func() tea.Msg {
		err := renameBranch(oldName, newName)
		return renameResultMsg{localOk: err == nil, err: err}
	}
}

func (m Model) cmdRenameAll(newName string) tea.Cmd {
	oldName := m.editing.Name
	upstream := m.editing.Upstream
	return func() tea.Msg {
		if err := renameBranch(oldName, newName); err != nil {
			return renameResultMsg{err: err}
		}
		remote, branch := splitUpstream(upstream)
		err := renameRemoteBranch(remote, branch, newName)
		return renameResultMsg{localOk: true, remoteOk: err == nil, err: err}
	}
}

func (m Model) cmdDelete(name string) tea.Cmd {
	return func() tea.Msg {
		return deleteResultMsg{err: deleteBranch(name)}
	}
}

func (m Model) cmdCreate(name string) tea.Cmd {
	return func() tea.Msg {
		return createResultMsg{err: createBranch(name)}
	}
}

// splitUpstream splits "origin/feature/foo" into ("origin", "feature/foo").
func splitUpstream(upstream string) (remote, branch string) {
	idx := strings.Index(upstream, "/")
	if idx < 0 {
		return upstream, upstream
	}
	return upstream[:idx], upstream[idx+1:]
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
		content = spinner + " " + styles.Dimmed.Render(m.processingMsg) +
			"  " + styles.Subtitle.Render(m.elapsed())

	case stateBrowse:
		content = styles.Title.Render("Git Branch Manager") + "\n\n"

		if len(m.branches) == 0 {
			content += styles.Dimmed.Render("No branches found.")
		} else {
			for i, b := range m.branches {
				cursor := "  "
				nameStyle := lipgloss.NewStyle()

				if b.IsCurrent {
					nameStyle = styles.CurrentBranch
				}
				if i == m.cursor {
					cursor = styles.Selected.Render("> ")
					nameStyle = styles.Selected
				}

				prefix := "  "
				if b.IsCurrent {
					prefix = styles.CurrentBranch.Render("* ")
				}

				deleteMarker := ""
				if m.deleteStaged && m.deleteStagedIdx == i {
					deleteMarker = "  " + styles.Err.Render("d again to delete")
					nameStyle = styles.Err
					if i == m.cursor {
						cursor = styles.Err.Render("> ")
					}
				}

				remote := ""
				if b.HasRemote && deleteMarker == "" {
					remote = "  " + styles.Remote.Render("["+b.Upstream+"]")
				}

				content += fmt.Sprintf("%s%s%s%s%s\n",
					cursor,
					prefix,
					nameStyle.Render(fmt.Sprintf("%-40s", b.Name)),
					remote,
					deleteMarker,
				)
			}
		}

		content += "\n" + styles.Help.Render("enter checkout  e rename  c create  dd delete  r refresh  esc/q back")

	case stateEdit:
		content = styles.Title.Render("Rename Branch") + "\n\n"
		content += styles.Dimmed.Render("Old: ") + styles.Subtitle.Render(m.editing.Name) + "\n"
		content += styles.Dimmed.Render("New: ") + m.input.View() + "\n"
		content += "\n" + styles.Help.Render("enter confirm  esc cancel")

	case stateCreate:
		newName := m.input.Value()
		content = styles.Title.Render("New Branch") + "\n\n"
		content += styles.Dimmed.Render("Name: ") + m.input.View() + "\n\n"

		switch {
		case newName == "":
			content += styles.Dimmed.Render("enter a branch name")
		case m.branchExists(newName):
			content += styles.Err.Render("✗ branch already exists")
		default:
			content += styles.Success.Render("✓ name available")
		}

		content += "\n\n" + styles.Help.Render("enter create  esc cancel")

	case stateConfirmRemote:
		newName := strings.TrimSpace(m.input.Value())
		remote, _ := splitUpstream(m.editing.Upstream)
		newUpstream := remote + "/" + newName

		content = styles.Title.Render("Update Remote?") + "\n\n"
		content += fmt.Sprintf("Also rename %s\n         → %s?\n\n",
			styles.Remote.Render(m.editing.Upstream),
			styles.Selected.Render(newUpstream),
		)

		yesStyle := styles.Dimmed
		noStyle := styles.Dimmed
		if m.confirmIdx == 0 {
			yesStyle = styles.Selected
		} else {
			noStyle = styles.Selected
		}
		content += fmt.Sprintf("  %s    %s\n",
			yesStyle.Render("[ Yes ]"),
			noStyle.Render("[ No ]"),
		)
		content += "\n" + styles.Help.Render("←→/hl select  enter confirm  y/n shortcut  esc cancel")

	case stateProcessing:
		spinner := styles.Selected.Render(spinnerFrames[m.spinnerFrame])
		content = spinner + " " + styles.Dimmed.Render(m.processingMsg) +
			"  " + styles.Subtitle.Render(m.elapsed())

	case stateResult:
		content = styles.Title.Render("Done") + "\n\n"
		content += m.result + "\n"
		content += "\n" + styles.Help.Render("any key to refresh  esc/q back")
	}

	return styles.Box.Render(content)
}
