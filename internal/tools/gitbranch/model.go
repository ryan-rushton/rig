package gitbranch

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ryan-rushton/rig/internal/messages"
	"github.com/ryan-rushton/rig/internal/styles"
)

type viewState int

const (
	stateLoading viewState = iota
	stateBrowse
	stateEdit
	stateConfirmRemote
	stateProcessing
	stateResult
	stateError
)

type branchesLoadedMsg struct {
	branches []Branch
	err      error
}

type renameResultMsg struct {
	localOk  bool
	remoteOk bool
	err      error
}

// Model is the git branch editor TUI model.
type Model struct {
	state      viewState
	branches   []Branch
	cursor     int
	input      textinput.Model
	editing    Branch
	didRemote  bool
	result     string
	errMsg     string
	confirmIdx int // 0 = yes, 1 = no
}

func New() Model {
	ti := textinput.New()
	ti.CharLimit = 200
	ti.Width = 50

	return Model{
		state: stateLoading,
		input: ti,
	}
}

func (m Model) Init() tea.Cmd {
	return fetchBranches
}

func fetchBranches() tea.Msg {
	branches, err := getBranches()
	return branchesLoadedMsg{branches: branches, err: err}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case branchesLoadedMsg:
		if msg.err != nil {
			m.state = stateError
			m.errMsg = msg.err.Error()
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
			m.state = stateError
			m.errMsg = msg.err.Error()
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
			if m.cursor < len(m.branches)-1 {
				m.cursor++
			}
		case "enter", "e":
			if len(m.branches) > 0 {
				m.editing = m.branches[m.cursor]
				m.input.SetValue(m.editing.Name)
				m.input.Focus()
				m.input.CursorEnd()
				m.state = stateEdit
			}
		case "r":
			m.state = stateLoading
			return m, fetchBranches
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
			m.state = stateProcessing
			return m, m.cmdRenameLocal(newName)
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
			m.didRemote = true
			m.state = stateProcessing
			return m, m.cmdRenameAll(strings.TrimSpace(m.input.Value()))
		case "n":
			m.didRemote = false
			m.state = stateProcessing
			return m, m.cmdRenameLocal(strings.TrimSpace(m.input.Value()))
		case "enter", " ":
			newName := strings.TrimSpace(m.input.Value())
			if m.confirmIdx == 0 {
				m.didRemote = true
				m.state = stateProcessing
				return m, m.cmdRenameAll(newName)
			}
			m.didRemote = false
			m.state = stateProcessing
			return m, m.cmdRenameLocal(newName)
		}

	case stateResult, stateError:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			return m, func() tea.Msg { return messages.BackMsg{} }
		default:
			m.state = stateLoading
			m.cursor = 0
			return m, fetchBranches
		}
	}

	return m, nil
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

// splitUpstream splits "origin/feature/foo" into ("origin", "feature/foo").
func splitUpstream(upstream string) (remote, branch string) {
	idx := strings.Index(upstream, "/")
	if idx < 0 {
		return upstream, upstream
	}
	return upstream[:idx], upstream[idx+1:]
}

func (m Model) View() string {
	var content string

	switch m.state {
	case stateLoading:
		content = styles.Dimmed.Render("Loading branches...")

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

				remote := ""
				if b.HasRemote {
					remote = "  " + styles.Remote.Render("["+b.Upstream+"]")
				}

				content += fmt.Sprintf("%s%s%s%s\n",
					cursor,
					prefix,
					nameStyle.Render(fmt.Sprintf("%-40s", b.Name)),
					remote,
				)
			}
		}

		content += "\n" + styles.Help.Render("↑↓/jk navigate  e/enter rename  r refresh  esc/q back")

	case stateEdit:
		content = styles.Title.Render("Rename Branch") + "\n\n"
		content += styles.Dimmed.Render("Old: ") + styles.Subtitle.Render(m.editing.Name) + "\n"
		content += styles.Dimmed.Render("New: ") + m.input.View() + "\n"
		content += "\n" + styles.Help.Render("enter confirm  esc cancel")

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
		content = styles.Dimmed.Render("Renaming branch...")

	case stateResult:
		content = styles.Title.Render("Done") + "\n\n"
		content += m.result + "\n"
		content += "\n" + styles.Help.Render("any key to refresh  esc/q back")

	case stateError:
		content = styles.Title.Render("Error") + "\n\n"
		content += styles.Err.Render(m.errMsg) + "\n"
		content += "\n" + styles.Help.Render("any key to retry  esc/q back")
	}

	return styles.Box.Render(content)
}
