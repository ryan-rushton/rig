package home

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
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
	viewport  viewport.Model
	width     int
	height    int
}

func New(version string) Model {
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(styles.DimGray).Italic(true).Bold(true)
	h.Styles.ShortDesc = styles.Help
	h.Styles.ShortSeparator = styles.Help

	vp := viewport.New(80, 20)
	vp.KeyMap = viewport.KeyMap{}

	return Model{
		version:  version,
		help:     h,
		keys:     newKeys(),
		viewport: vp,
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 6    // border(2) + padding(4)
		m.viewport.Height = msg.Height - 10 // border(2) + padding(2) + banner(4) + help+blank(2)
		return m, nil

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
		case "q", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				ensureCursorVisible(&m.viewport, m.cursor)
			}
		case "down", "j":
			if m.cursor < len(registry.All())-1 {
				m.cursor++
				ensureCursorVisible(&m.viewport, m.cursor)
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

func ensureCursorVisible(vp *viewport.Model, cursor int) {
	if cursor < vp.YOffset {
		vp.SetYOffset(cursor)
	} else if cursor >= vp.YOffset+vp.Height {
		vp.SetYOffset(cursor - vp.Height + 1)
	}
}

func (m Model) View() string {
	banner := "█▀█ █ █▀▀\n█▀▄ █ █ █\n▀ ▀ ▀ ▀▀▀"
	content := styles.Title.Render(banner) + "\n"
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

	all := registry.All()
	var listContent strings.Builder
	for i, t := range all {
		cursor := "  "
		nameStyle := lipgloss.NewStyle()
		descStyle := styles.Dimmed

		if i == m.cursor {
			cursor = styles.Selected.Render("> ")
			nameStyle = styles.Selected
			descStyle = styles.Subtitle
		}

		paddedName := fmt.Sprintf("%-22s", t.Name)
		listContent.WriteString(fmt.Sprintf("%s%s %s",
			cursor,
			nameStyle.Render(paddedName),
			descStyle.Render(t.Description),
		))
		if i < len(all)-1 {
			listContent.WriteByte('\n')
		}
	}

	m.viewport.SetContent(listContent.String())
	content += m.viewport.View()

	if len(all) > m.viewport.Height {
		content += "\n" + styles.Dimmed.Render(
			fmt.Sprintf("(%d%% — ↑↓/jk to scroll)", int(m.viewport.ScrollPercent()*100)),
		)
	}

	content += "\n" + m.help.View(m.keys)

	return styles.Box.Render(content)
}
