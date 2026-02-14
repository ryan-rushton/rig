package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ryan-rushton/rig/internal/updater"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "update",
		Short: "Check for and apply updates",
		Long:  "Checks GitHub for a newer release and replaces the current binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			if version == "dev" {
				fmt.Println("Skipping update check (dev build)")
				return nil
			}

			fmt.Println("Checking for updates...")

			latest, err := updater.LatestRelease()
			if err != nil {
				return fmt.Errorf("checking for updates: %w", err)
			}

			if !updater.IsNewer(version, latest) {
				fmt.Printf("Already up to date (%s)\n", version)
				return nil
			}

			fmt.Printf("Update available: %s → %s\n", version, latest)
			fmt.Println("Downloading...")

			if err := updater.DownloadAndReplace(latest); err != nil {
				return fmt.Errorf("updating: %w", err)
			}

			fmt.Printf("Updated to %s! Restart rig to use the new version.\n", latest)
			return nil
		},
	})
}
