package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
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
			p := tea.NewProgram(messages.Standalone(gitbranch.New()), tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
	})
}
