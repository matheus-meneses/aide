package main

import (
	"aide/cli/internal/platform/xdg"
	"aide/cli/internal/setup/bootstrap"
	"aide/cli/internal/ui/widgets"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Setup aide home directory, install Python runtime, and fetch the plugin registry",
	RunE:  initExecute,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func aideHome() string {
	return xdg.AideHome()
}

func initExecute(_ *cobra.Command, _ []string) error {
	widgets.Printf("Initializing aide in %s\n", aideHome())

	if err := bootstrap.Ensure(func(msg string) {
		widgets.Printf("  [+] %s\n", msg)
	}); err != nil {
		return err
	}

	widgets.Println("\nDone!")
	widgets.Println("  Launch the UI:    aide ui")
	widgets.Println("  Install plugins:  aide plugin install <name>")
	widgets.Println("  Connect a source: aide plugin configure")
	return nil
}
