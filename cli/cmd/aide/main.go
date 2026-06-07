package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"aide/cli/internal/config"
	"aide/cli/internal/updater"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "aide",
	Short: "Aide - your personal work assistant",
	Long:  "Aide orchestrates data collection, provides insights, and assists with daily work management.",
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if cmd.Name() == "version" || cmd.Name() == "init" {
			return
		}
		updater.CheckOnce(version)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", config.DefaultConfigPath(), "config file path")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
