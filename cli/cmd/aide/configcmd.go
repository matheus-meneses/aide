package main

import (
	"aide/cli/internal/config"
	"aide/cli/internal/keychain"
	"aide/cli/internal/plugin"
	"aide/cli/internal/provision"
	"aide/cli/internal/ui"
	"fmt"
	"sort"
	"strconv"

	"github.com/spf13/cobra"
)

func sortedSourceNames(sources map[string]config.Source) []string {
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and manage aide configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	RunE:  configShowExecute,
}

var configCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate configuration and check source readiness",
	RunE:  configCheckExecute,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a general setting (concurrency, timeout_seconds, verify_ssl, ca_bundle, log_level, log_format)",
	Args:  cobra.ExactArgs(2),
	RunE:  configSetExecute,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configCheckCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}

func configSetExecute(_ *cobra.Command, args []string) error {
	key, value := args[0], args[1]

	snap, err := provision.ConfigSnapshot(cfgFile)
	if err != nil {
		return err
	}

	in := provision.GeneralSettingsInput{
		Concurrency:    snap.Settings.Concurrency,
		TimeoutSeconds: snap.Settings.TimeoutSeconds,
		VerifySSL:      snap.Settings.TLS.VerifySSL,
		CABundle:       snap.Settings.TLS.CABundle,
		LogLevel:       snap.Settings.LogLevel,
		LogFormat:      snap.Settings.LogFormat,
	}

	switch key {
	case "concurrency":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("concurrency must be an integer: %w", err)
		}
		in.Concurrency = n
	case "timeout_seconds", "timeout":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("timeout_seconds must be an integer: %w", err)
		}
		in.TimeoutSeconds = n
	case "verify_ssl":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("verify_ssl must be true or false: %w", err)
		}
		in.VerifySSL = &b
	case "ca_bundle":
		in.CABundle = value
	case "log_level":
		in.LogLevel = value
	case "log_format":
		in.LogFormat = value
	default:
		return fmt.Errorf("unknown setting %q (valid: concurrency, timeout_seconds, verify_ssl, ca_bundle, log_level, log_format)", key)
	}

	if err := provision.SetGeneralSettings(cfgFile, in); err != nil {
		return err
	}
	ui.PrintSuccess("Set %s = %s", key, value)
	return nil
}

func configShowExecute(_ *cobra.Command, _ []string) error {
	cfg, err := config.LoadRaw(cfgFile)
	if err != nil {
		return err
	}

	mgr := plugin.NewManager()

	fmt.Println("╭─ Agent ─────────────────────────────────╮")
	fmt.Printf("│  run_interval:   %-24s│\n", cfg.Agent.RunInterval)
	fmt.Printf("│  briefing_times: %-24v│\n", cfg.Agent.BriefingTimes)
	fmt.Printf("│  llm_model:      %-24s│\n", cfg.Agent.LLMModel)
	fmt.Println("╰──────────────────────────────────────────╯")

	fmt.Println("\n╭─ Sources ────────────────────────────────╮")
	if len(cfg.Sources) == 0 {
		fmt.Println("│  (none configured)                       │")
		fmt.Println("│  Run 'aide config source add' to start   │")
	}
	for _, name := range sortedSourceNames(cfg.Sources) {
		src := cfg.Sources[name]
		status := "OFF"
		if src.Enabled {
			status = " ON"
		}
		desc := ""
		if m, err := mgr.Get(name); err == nil && m.Description != "" {
			desc = m.Description
		}
		fmt.Printf("│  [%s] %-15s %s\n", status, name, desc)
	}
	fmt.Println("╰──────────────────────────────────────────╯")

	if len(cfg.Team) > 0 {
		fmt.Println("\n╭─ Team ───────────────────────────────────╮")
		for _, m := range cfg.Team {
			fmt.Printf("│  %-40s│\n", m.Name)
		}
		fmt.Println("╰──────────────────────────────────────────╯")
	}

	return nil
}

func configCheckExecute(_ *cobra.Command, _ []string) error {
	cfg, err := config.LoadRaw(cfgFile)
	if err != nil {
		return fmt.Errorf("config load failed: %w", err)
	}

	mgr := plugin.NewManager()
	issues := 0

	fmt.Print("Checking sources...\n\n")
	for _, name := range sortedSourceNames(cfg.Sources) {
		src := cfg.Sources[name]
		m, pluginErr := mgr.Get(name)
		if pluginErr != nil {
			fmt.Printf("  [WARN] %s - plugin not installed (run: aide plugin install %s)\n", name, name)
			issues++
			continue
		}

		if !src.Enabled {
			fmt.Printf("  [OFF]  %s\n", name)
			continue
		}

		sourceOK := true

		if len(m.Credentials) > 0 {
			_, credErr := keychain.GetAll(name)
			if credErr != nil {
				fmt.Printf("  [WARN] %s - credentials missing (run: aide credential set %s)\n", name, name)
				issues++
				sourceOK = false
			}
		}

		if sourceOK {
			for _, field := range m.Config {
				if field.Required {
					val := src.Config[field.Key]
					if val == nil || val == "" {
						fmt.Printf("  [WARN] %s - missing required field '%s'\n", name, field.Key)
						issues++
						sourceOK = false
					}
				}
			}
		}

		if sourceOK {
			fmt.Printf("  [OK]   %s\n", name)
		}
	}

	if issues != 0 {
		fmt.Printf("\n  %d issue(s) found.\n", issues)
		return fmt.Errorf("%d configuration issue(s) found", issues)
	}

	fmt.Println("\n  All checks passed.")
	return nil
}
