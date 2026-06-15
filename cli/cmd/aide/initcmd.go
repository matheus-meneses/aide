package main

import (
	"aide/cli/internal/platform/xdg"
	"aide/cli/internal/setup/bootstrap"
	"fmt"

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
	fmt.Printf("Initializing aide in %s\n", aideHome())

	if err := bootstrap.Ensure(func(msg string) {
		fmt.Printf("  [+] %s\n", msg)
	}); err != nil {
		return err
	}

	fmt.Println("\nDone!")
	fmt.Println("  Install plugins:  aide plugin install <name>")
	fmt.Println("  Add a source:     aide config source add")
	return nil
}
