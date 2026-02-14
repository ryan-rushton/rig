package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ryan-rushton/rig/internal/home"
	"github.com/ryan-rushton/rig/internal/messages"
	"github.com/ryan-rushton/rig/internal/tools/gitbranch"
)

// Model is the top-level application model that manages screen transitions.
type Model struct {
	current tea.Model
	version string
}

func New(version string) Model {
	return Model{
		current: home.New(version),
		version: version,
	}
}

func (m Model) Init() tea.Cmd {
	return m.current.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.BackMsg:
		h := home.New(m.version)
		m.current = h
		return m, h.Init()

	case messages.ToolSelectedMsg:
		if msg.ID == "git-branch" {
			gb := gitbranch.New()
			m.current = gb
			return m, gb.Init()
		}
	}

	updated, cmd := m.current.Update(msg)
	m.current = updated
	return m, cmd
}

func (m Model) View() string {
	return m.current.View()
}
