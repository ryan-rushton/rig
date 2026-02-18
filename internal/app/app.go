package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ryan-rushton/rig/internal/home"
	"github.com/ryan-rushton/rig/internal/messages"
	"github.com/ryan-rushton/rig/internal/registry"
)

// Model is the top-level application model that manages screen transitions.
type Model struct {
	current    tea.Model
	version    string
	windowSize tea.WindowSizeMsg
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
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.windowSize = ws
	}

	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case messages.BackMsg:
		h := home.New(m.version)
		m.current = h
		return m, tea.Batch(h.Init(), func() tea.Msg { return m.windowSize })

	case messages.ToolSelectedMsg:
		if t := registry.Get(msg.ID); t != nil {
			tool := t.New()
			m.current = tool
			return m, tea.Batch(tool.Init(), func() tea.Msg { return m.windowSize })
		}
	}

	updated, cmd := m.current.Update(msg)
	m.current = updated
	return m, cmd
}

func (m Model) View() string {
	return m.current.View()
}
