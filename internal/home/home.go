package home

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ryan-rushton/rig/internal/messages"
	"github.com/ryan-rushton/rig/internal/styles"
)

type tool struct {
	ID          string
	Name        string
	Description string
}

var tools = []tool{
	{
		ID:          "git-branch",
		Name:        "git-branch",
		Description: "Rename git branches (local and remote)",
	},
}

// Model is the home screen model.
type Model struct {
	cursor int
}

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(tools)-1 {
				m.cursor++
			}
		case "enter", " ":
			selected := tools[m.cursor]
			return m, func() tea.Msg {
				return messages.ToolSelectedMsg{ID: selected.ID}
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	content := styles.Title.Render("rig") + "\n"
	content += styles.Subtitle.Render("Ryan's TUI Toolkit") + "\n\n"

	for i, t := range tools {
		cursor := "  "
		nameStyle := lipgloss.NewStyle()
		descStyle := styles.Dimmed

		if i == m.cursor {
			cursor = styles.Selected.Render("> ")
			nameStyle = styles.Selected
			descStyle = styles.Subtitle
		}

		content += fmt.Sprintf("%s%-22s %s\n",
			cursor,
			nameStyle.Render(t.Name),
			descStyle.Render(t.Description),
		)
	}

	content += "\n" + styles.Help.Render("↑↓/jk navigate  enter select  q quit")

	return styles.Box.Render(content)
}
