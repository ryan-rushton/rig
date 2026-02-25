package cmd

import (
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/ryan-rushton/rig/internal/messages"
	"github.com/ryan-rushton/rig/internal/tools/testchanged"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:     "test-changed",
		Aliases: []string{"tc"},
		Short:   "Run tests for files changed vs merge base",
		Long:    "Detect changed files compared to the merge-base with the default branch and run affected tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := tea.NewProgram(messages.Standalone(testchanged.New()))
			_, err := p.Run()
			return err
		},
	})
}
