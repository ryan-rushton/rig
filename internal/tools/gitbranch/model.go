package gitbranch

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/stopwatch"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
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

type keyMap struct {
	bindings []key.Binding
}

func (k keyMap) ShortHelp() []key.Binding  { return k.bindings }
func (k keyMap) FullHelp() [][]key.Binding { return nil }

var browseKeys = keyMap{bindings: []key.Binding{
	key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "checkout")),
	key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "rename")),
	key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "create")),
	key.NewBinding(key.WithKeys("d"), key.WithHelp("dd", "delete")),
	key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc/q", "back")),
}}

var editKeys = keyMap{bindings: []key.Binding{
	key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}}

var createKeys = keyMap{bindings: []key.Binding{
	key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "create")),
	key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}}

var confirmRemoteKeys = keyMap{bindings: []key.Binding{
	key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←→/hl", "select")),
	key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	key.NewBinding(key.WithKeys("y", "n"), key.WithHelp("y/n", "shortcut")),
	key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}}

var resultKeys = keyMap{bindings: []key.Binding{
	key.NewBinding(key.WithKeys("any"), key.WithHelp("any key", "refresh")),
	key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc/q", "back")),
}}

var dismissKeys = keyMap{bindings: []key.Binding{
	key.NewBinding(key.WithKeys("any"), key.WithHelp("any key", "dismiss")),
}}

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
	spinner       spinner.Model
	stopwatch     stopwatch.Model
	help          help.Model
	processingMsg string
	viewport      viewport.Model
	width         int
	height        int
	// delete staging — first d marks, second d confirms
	deleteStaged    bool
	deleteStagedIdx int
}

func New() Model {
	ti := textinput.New()
	ti.CharLimit = 200
	ti.Width = 50

	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = styles.Selected

	sw := stopwatch.NewWithInterval(100 * time.Millisecond)

	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(styles.DimGray).Italic(true).Bold(true)
	h.Styles.ShortDesc = styles.Help
	h.Styles.ShortSeparator = styles.Help

	vp := viewport.New(80, 20)
	vp.KeyMap = viewport.KeyMap{}

	return Model{
		state:     stateLoading,
		input:     ti,
		spinner:   s,
		stopwatch: sw,
		help:      h,
		viewport:  vp,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchBranches, m.spinner.Tick, m.stopwatch.Start())
}

func fetchBranches() tea.Msg {
	branches, err := getBranches()
	return branchesLoadedMsg{branches: branches, err: err}
}

// startAsync transitions into a waiting state, resets the timer, and
// batches the git command with the spinner/stopwatch. Returns the updated model
// and batched command — must be used as: return startAsync(m, ...).
func startAsync(m Model, state viewState, label string, cmd tea.Cmd) (Model, tea.Cmd) {
	m.state = state
	m.processingMsg = label
	return m, tea.Batch(cmd, m.spinner.Tick, m.stopwatch.Reset(), m.stopwatch.Start())
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// border(2) + padding(2) horizontal on each side
		m.viewport.Width = msg.Width - 6
		// border(2) + padding(2) + title+blank(2) + help+blank(2) = 8
		m.viewport.Height = msg.Height - 8
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

	// Route spinner and stopwatch messages when in async states.
	if m.state == stateLoading || m.state == stateProcessing {
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
		// Any key other than d clears delete staging.
		if msg.String() != "d" {
			m.deleteStaged = false
		}

		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return messages.BackMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				ensureCursorVisible(&m.viewport, m.cursor)
			}
		case "down", "j":
			if m.cursor < len(m.branches)-1 {
				m.cursor++
				ensureCursorVisible(&m.viewport, m.cursor)
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

func ensureCursorVisible(vp *viewport.Model, cursor int) {
	if cursor < vp.YOffset {
		vp.SetYOffset(cursor)
	} else if cursor >= vp.YOffset+vp.Height {
		vp.SetYOffset(cursor - vp.Height + 1)
	}
}

// splitUpstream splits "origin/feature/foo" into ("origin", "feature/foo").
func splitUpstream(upstream string) (remote, branch string) {
	before, after, ok := strings.Cut(upstream, "/")
	if !ok {
		return upstream, upstream
	}
	return before, after
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
		content = m.spinner.View() + " " + styles.Dimmed.Render(m.processingMsg) +
			"  " + styles.Subtitle.Render(elapsed)

	case stateBrowse:
		content = styles.Title.Render("Git Branch Manager") + "\n\n"

		if len(m.branches) == 0 {
			content += styles.Dimmed.Render("No branches found.")
		} else {
			var listContent strings.Builder
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

				listContent.WriteString(fmt.Sprintf("%s%s%s%s%s",
					cursor,
					prefix,
					nameStyle.Render(fmt.Sprintf("%-40s", b.Name)),
					remote,
					deleteMarker,
				))
				if i < len(m.branches)-1 {
					listContent.WriteByte('\n')
				}
			}

			m.viewport.SetContent(listContent.String())
			content += m.viewport.View()

			if len(m.branches) > m.viewport.Height {
				content += "\n" + styles.Dimmed.Render(
					fmt.Sprintf("(%d%% — ↑↓/jk to scroll)", int(m.viewport.ScrollPercent()*100)),
				)
			}
		}

		content += "\n" + m.help.View(browseKeys)

	case stateEdit:
		content = styles.Title.Render("Rename Branch") + "\n\n"
		content += styles.Dimmed.Render("Old: ") + styles.Subtitle.Render(m.editing.Name) + "\n"
		content += styles.Dimmed.Render("New: ") + m.input.View() + "\n"
		content += "\n" + m.help.View(editKeys)

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

		content += "\n\n" + m.help.View(createKeys)

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
		content += "\n" + m.help.View(confirmRemoteKeys)

	case stateProcessing:
		elapsed := fmt.Sprintf("%.2fs", m.stopwatch.Elapsed().Seconds())
		content = m.spinner.View() + " " + styles.Dimmed.Render(m.processingMsg) +
			"  " + styles.Subtitle.Render(elapsed)

	case stateResult:
		content = styles.Title.Render("Done") + "\n\n"
		content += m.result + "\n"
		content += "\n" + m.help.View(resultKeys)
	}

	return styles.Box.Render(content)
}
