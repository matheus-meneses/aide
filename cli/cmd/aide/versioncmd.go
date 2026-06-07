package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print aide version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("╭───────────────────────────────────────╮")
		fmt.Printf("│  aide %-33s│\n", version)
		fmt.Printf("│  platform: %-28s│\n", runtime.GOOS+"/"+runtime.GOARCH)
		fmt.Println("╰───────────────────────────────────────╯")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
