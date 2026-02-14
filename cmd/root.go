package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ryan-rushton/rig/internal/app"
)

var rootCmd = &cobra.Command{
	Use:   "rig",
	Short: "Ryan's TUI toolkit",
	Long:  "rig - a personal TUI toolkit for custom workflows and tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(app.New(), tea.WithAltScreen())
		_, err := p.Run()
		return err
	},
}

// SetVersion sets the version string shown by --version.
func SetVersion(v string) {
	rootCmd.Version = v
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
