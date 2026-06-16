package main

import (
	"aide/cli/internal/runtime/updater"
	"aide/cli/internal/ui/widgets"
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var updateCheckOnly bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update aide to the latest version",
	Long: `Check for a newer release and update aide in place.

The update method is chosen automatically based on how aide was installed
(install script, Homebrew, or the macOS app), so you always run the same
command regardless of channel.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		current := version

		rel, err := updater.LatestUpgrade(current)
		if err != nil {
			return fmt.Errorf("checking for updates: %w", err)
		}
		if !updater.IsNewer(rel.Tag, current) {
			widgets.Printf("aide is up to date (%s).\n", current)
			return nil
		}

		widgets.Printf("A new version is available: %s (current: %s)\n", rel.Tag, current)
		if rel.Notes != "" {
			widgets.Println()
			widgets.Println(rel.Notes)
			widgets.Println()
		}
		if updateCheckOnly {
			return nil
		}

		method := updater.DetectMethod(current)
		if !method.CanSelfUpdate() {
			widgets.Printf("This build can't update itself. Download the latest release at %s\n", rel.URL)
			return nil
		}

		if err := requireConfirm("Update now?"); err != nil {
			return err
		}

		prog := updater.Progress(func(line string) { widgets.Printf("  %s\n", line) })
		res, err := updater.Apply(context.Background(), current, method, prog)
		if err != nil {
			if errors.Is(err, updater.ErrUpToDate) {
				widgets.Printf("aide is up to date (%s).\n", current)
				return nil
			}
			return err
		}

		if res.RestartNow {
			widgets.Printf("\nUpdate staged. Restart the app to finish.\n")
		} else {
			widgets.Printf("\nUpdated to %s. Restart aide to use the new version.\n", res.Version)
		}
		return nil
	},
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "only check for updates; don't apply")
	rootCmd.AddCommand(updateCmd)
}
