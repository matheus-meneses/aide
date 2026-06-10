package main

import (
	"aide/cli/internal/config"
	"aide/cli/internal/updater"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

var (
	verbose   bool
	logFormat string
)

func logLevel() string {
	if verbose {
		return "debug"
	}
	return "info"
}

func logFormatValue() string {
	if logFormat == "json" {
		return "json"
	}
	return "text"
}

var rootCmd = &cobra.Command{
	Use:   "aide",
	Short: "Aide - your personal work assistant",
	Long:  "Aide orchestrates data collection, provides insights, and assists with daily work management.",
	PersistentPostRun: func(cmd *cobra.Command, _ []string) {
		if cmd.Name() == "version" || cmd.Name() == "init" {
			return
		}
		updater.CheckOnce(version)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", config.DefaultConfigPath(), "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug-level logging")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log output format: text or json")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
