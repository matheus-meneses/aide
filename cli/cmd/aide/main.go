package main

import (
	"aide/cli/internal/platform/clog"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/updater"
	"aide/cli/internal/ui/widgets"
	"errors"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

var (
	verbose      bool
	logFormat    string
	logLevelFlag string
	quiet        bool
	noColor      bool
	assumeYes    bool
	verifySSL    bool
	caBundle     string
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
	switch {
	case cmd.Flags().Changed("log-level"):
		flagLevel = logLevelFlag
	case verbose:
		flagLevel = "debug"
	case quiet:
		flagLevel = "error"
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
	Use:   "aide",
	Short: "Aide - your personal work assistant",
	Long: `Aide is a local-first work assistant. It collects data from the tools you
use, surfaces what needs attention, and helps you manage your day — all on
your machine.

Quickstart:
  aide ui                       launch the web UI and autonomous agent (easiest)

Or drive it from the terminal:
  aide plugin install <name>    add a data source plugin
  aide plugin configure         connect it as a source
  aide run                      collect data and generate a briefing
  aide report                   view your latest briefing`,
	Example: `  aide ui
  aide plugin install github
  aide plugin configure
  aide run
  aide report`,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		if noColor {
			widgets.SetColorEnabled(false)
		}
		widgets.SetQuiet(quiet)
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
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("aide {{.Version}}\n")

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", config.DefaultConfigPath(), "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug-level logging")
	rootCmd.PersistentFlags().StringVar(&logLevelFlag, "log-level", "", "log level: debug, info, warn, error")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log output format: text or json")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress incidental output (errors still shown)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&assumeYes, "yes", "y", false, "assume yes for all confirmation prompts")
	rootCmd.PersistentFlags().BoolVar(&verifySSL, "verify-ssl", true, "verify TLS certificates for plugin network requests")
	rootCmd.PersistentFlags().StringVar(&caBundle, "ca-bundle", "", "path to a CA bundle (PEM) plugins use to verify TLS")
}

// registerGroups organizes the root help output into logical sections instead
// of one flat alphabetical list.
func registerGroups() {
	rootCmd.AddGroup(
		&cobra.Group{ID: "setup", Title: "Setup & Configuration:"},
		&cobra.Group{ID: "work", Title: "Daily Work:"},
		&cobra.Group{ID: "ecosystem", Title: "Plugins:"},
		&cobra.Group{ID: "system", Title: "System:"},
	)
	groupOf := map[string]string{
		"init": "setup", "config": "setup", "credential": "setup", "tls": "setup",
		"ui": "work", "run": "work", "report": "work", "stats": "work", "history": "work",
		"diff": "work", "agent": "work", "team": "work",
		"plugin":  "ecosystem",
		"version": "system", "prune": "system", "dev": "system", "whoami": "system",
	}
	for _, c := range rootCmd.Commands() {
		if g, ok := groupOf[c.Name()]; ok {
			c.GroupID = g
		}
	}
	rootCmd.SetHelpCommandGroupID("system")
	rootCmd.SetCompletionCommandGroupID("system")
}

func main() {
	registerGroups()
	err := rootCmd.Execute()
	if err == nil {
		return
	}
	if errors.Is(err, errCanceled) {
		os.Exit(130)
	}
	widgets.PrintError("%s", err)
	os.Exit(1)
}
