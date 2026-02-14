package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ryan-rushton/rig/internal/app"
	"github.com/spf13/cobra"
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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
