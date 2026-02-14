package messages

import tea "github.com/charmbracelet/bubbletea"

// BackMsg is sent by tools when they want to return to the home screen.
type BackMsg struct{}

// ToolSelectedMsg is sent by the home screen when a tool is selected.
type ToolSelectedMsg struct {
	ID string
}

// standalone wraps a tool model so that BackMsg causes a quit instead of
// navigating back — used when a tool is launched directly via CLI.
type standalone struct {
	inner tea.Model
}

// Standalone wraps a model for direct CLI invocation.
func Standalone(m tea.Model) tea.Model {
	return standalone{inner: m}
}

func (s standalone) Init() tea.Cmd {
	return s.inner.Init()
}

func (s standalone) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(BackMsg); ok {
		return s, tea.Quit
	}
	m, cmd := s.inner.Update(msg)
	s.inner = m
	return s, cmd
}

func (s standalone) View() string {
	return s.inner.View()
}
