package main

import (
	"aide/cli/internal/clog"
	"aide/cli/internal/config"
	"aide/cli/internal/ui"
	"aide/cli/internal/updater"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

var (
	verbose   bool
	logFormat string
	verifySSL bool
	caBundle  string
)

var (
	resolvedLevel  = "info"
	resolvedFormat = "text"
)

func logLevel() string {
	return resolvedLevel
}

func logFormatValue() string {
	return resolvedFormat
}

// resolveLogging applies precedence flag > env > config > default for the log
// level and format, sourcing config defaults so the CLI honors settings.yaml.
func resolveLogging(cmd *cobra.Command) (level, format string) {
	flagLevel := ""
	if verbose {
		flagLevel = "debug"
	}
	flagFormat := ""
	if cmd.Flags().Changed("log-format") {
		flagFormat = logFormat
	}

	cfgLevel, cfgFormat := "", ""
	if cfg, err := config.LoadRaw(cfgFile); err == nil {
		cfgLevel = cfg.Settings.LogLevel
		cfgFormat = cfg.Settings.LogFormat
	}

	return clog.Resolve(flagLevel, flagFormat, cfgLevel, cfgFormat)
}

func verifySSLValue() bool {
	return verifySSL
}

func caBundleValue() string {
	return caBundle
}

var rootCmd = &cobra.Command{
	Use:           "aide",
	Short:         "Aide - your personal work assistant",
	Long:          "Aide orchestrates data collection, provides insights, and assists with daily work management.",
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		resolvedLevel, resolvedFormat = resolveLogging(cmd)
		clog.Configure(resolvedLevel, resolvedFormat)
	},
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
	rootCmd.PersistentFlags().BoolVar(&verifySSL, "verify-ssl", true, "verify TLS certificates for plugin network requests")
	rootCmd.PersistentFlags().StringVar(&caBundle, "ca-bundle", "", "path to a CA bundle (PEM) plugins use to verify TLS")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		ui.PrintError("%s", err)
		os.Exit(1)
	}
}
