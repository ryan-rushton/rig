package cmd

import (
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/ryan-rushton/rig/internal/messages"
	"github.com/ryan-rushton/rig/internal/tools/gitbranch"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:     "git-branch",
		Aliases: []string{"gb"},
		Short:   "Edit git branch names",
		Long:    "Interactive TUI for renaming git branches locally and on remote",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := tea.NewProgram(messages.Standalone(gitbranch.New()))
			_, err := p.Run()
			return err
		},
	})
}
