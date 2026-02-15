package home

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ryan-rushton/rig/internal/messages"
	"github.com/ryan-rushton/rig/internal/registry"
	"github.com/ryan-rushton/rig/internal/styles"
	"github.com/ryan-rushton/rig/internal/updater"
)

// Model is the home screen model.
type Model struct {
	cursor    int
	version   string
	updateTag string
	updating  bool
	updated   bool
	updateErr string
}

func New(version string) Model {
	return Model{version: version}
}

func checkForUpdate(version string) tea.Cmd {
	return func() tea.Msg {
		latest, err := updater.LatestRelease()
		if err != nil || !updater.IsNewer(version, latest) {
			return nil
		}
		return messages.UpdateAvailableMsg{Tag: latest}
	}
}

func runUpdate(tag string) tea.Cmd {
	return func() tea.Msg {
		err := updater.DownloadAndReplace(tag)
		return messages.UpdateFinishedMsg{Err: err}
	}
}

func (m Model) Init() tea.Cmd {
	if m.version == "dev" {
		return nil
	}
	return checkForUpdate(m.version)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.UpdateAvailableMsg:
		m.updateTag = msg.Tag
		return m, nil

	case messages.UpdateFinishedMsg:
		m.updating = false
		if msg.Err != nil {
			m.updateErr = msg.Err.Error()
		} else {
			m.updated = true
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "u" && m.updateTag != "" && !m.updating && !m.updated {
			m.updating = true
			m.updateErr = ""
			return m, runUpdate(m.updateTag)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(registry.All())-1 {
				m.cursor++
			}
		case "enter", " ":
			all := registry.All()
			if m.cursor < len(all) {
				selected := all[m.cursor]
				return m, func() tea.Msg {
					return messages.ToolSelectedMsg{ID: selected.ID}
				}
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	content := styles.Title.Render("rig") + "\n"
	content += styles.Subtitle.Render("Ryan's TUI Toolkit") + "\n\n"

	// Update banner
	switch {
	case m.updateErr != "":
		content += styles.Err.Render(fmt.Sprintf("Update failed: %s", m.updateErr)) + "\n\n"
	case m.updated:
		content += styles.Success.Render(fmt.Sprintf("Updated! Restart rig to use %s", m.updateTag)) + "\n\n"
	case m.updating:
		content += styles.UpdateBanner.Render(fmt.Sprintf("Updating to %s...", m.updateTag)) + "\n\n"
	case m.updateTag != "":
		content += styles.UpdateBanner.Render(
			fmt.Sprintf("Update available: %s (press u to update)", m.updateTag),
		) + "\n\n"
	}

	for i, t := range registry.All() {
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

	helpText := "↑↓/jk navigate  enter select  q quit"
	if m.updateTag != "" && !m.updating && !m.updated {
		helpText = "↑↓/jk navigate  enter select  u update  q quit"
	}
	content += "\n" + styles.Help.Render(helpText)

	return styles.Box.Render(content)
}
