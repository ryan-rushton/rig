package home

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ryan-rushton/rig/internal/messages"
	"github.com/ryan-rushton/rig/internal/registry"
	"github.com/ryan-rushton/rig/internal/styles"
	"github.com/ryan-rushton/rig/internal/updater"
)

type keyMap struct {
	Navigate key.Binding
	Select   key.Binding
	Update   key.Binding
	Quit     key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Navigate, k.Select, k.Update, k.Quit}
}
func (k keyMap) FullHelp() [][]key.Binding { return nil }

func newKeys() keyMap {
	return keyMap{
		Navigate: key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓/jk", "navigate")),
		Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Update:   key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update"), key.WithDisabled()),
		Quit:     key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
}

// Model is the home screen model.
type Model struct {
	cursor    int
	version   string
	updateTag string
	updating  bool
	updated   bool
	updateErr string
	help      help.Model
	keys      keyMap
}

func New(version string) Model {
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(styles.DimGray).Italic(true).Bold(true)
	h.Styles.ShortDesc = styles.Help
	h.Styles.ShortSeparator = styles.Help

	return Model{
		version: version,
		help:    h,
		keys:    newKeys(),
	}
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
		m.keys.Update.SetEnabled(true)
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
			m.keys.Update.SetEnabled(false)
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

		paddedName := fmt.Sprintf("%-22s", t.Name)
		content += fmt.Sprintf("%s%s %s\n",
			cursor,
			nameStyle.Render(paddedName),
			descStyle.Render(t.Description),
		)
	}

	content += "\n" + m.help.View(m.keys)

	return styles.Box.Render(content)
}
