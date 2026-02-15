package styles

import "github.com/charmbracelet/lipgloss"

var (
	Pink    = lipgloss.Color("#FF2E97")
	Gray    = lipgloss.Color("#8A8F98")
	DimGray = lipgloss.Color("#3D4250")
	Green   = lipgloss.Color("#39FF14")
	Red     = lipgloss.Color("#FF3131")
	Cyan    = lipgloss.Color("#00F0FF")

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Cyan)

	Subtitle = lipgloss.NewStyle().
			Foreground(Cyan)

	Selected = lipgloss.NewStyle().
			Foreground(Pink).
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
		BorderForeground(Cyan).
		Padding(1, 2)

	UpdateBanner = lipgloss.NewStyle().
			Foreground(Cyan).
			Bold(true)
)
