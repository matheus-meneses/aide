package main

import (
	"aide/cli/internal/ui/widgets"
	"runtime"

	"github.com/spf13/cobra"
)

var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print aide version",
	Run: func(_ *cobra.Command, _ []string) {
		widgets.Println("╭───────────────────────────────────────╮")
		widgets.Printf("│  aide %-33s│\n", version)
		widgets.Printf("│  platform: %-28s│\n", runtime.GOOS+"/"+runtime.GOARCH)
		widgets.Println("╰───────────────────────────────────────╯")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
