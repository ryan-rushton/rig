package styles

import "github.com/charmbracelet/lipgloss"

var (
	Purple  = lipgloss.Color("#7C3AED")
	Gray    = lipgloss.Color("#9CA3AF")
	DimGray = lipgloss.Color("#4B5563")
	Green   = lipgloss.Color("#10B981")
	Red     = lipgloss.Color("#EF4444")

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Purple)

	Subtitle = lipgloss.NewStyle().
		Foreground(Gray)

	Selected = lipgloss.NewStyle().
		Foreground(Purple).
		Bold(true)

	Dimmed = lipgloss.NewStyle().
		Foreground(DimGray)

	Success = lipgloss.NewStyle().
		Foreground(Green)

	Err = lipgloss.NewStyle().
		Foreground(Red)

	Help = lipgloss.NewStyle().
		Foreground(DimGray).
		Italic(true)

	CurrentBranch = lipgloss.NewStyle().
		Foreground(Green).
		Bold(true)

	Remote = lipgloss.NewStyle().
		Foreground(Gray)

	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(DimGray).
		Padding(1, 2)
)
