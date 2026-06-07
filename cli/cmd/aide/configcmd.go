package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"aide/cli/internal/config"
	"aide/cli/internal/keychain"
	"aide/cli/internal/registry"
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

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configCheckCmd)
	rootCmd.AddCommand(configCmd)
}

func configShowExecute(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadRaw(cfgFile)
	if err != nil {
		return err
	}

	reg := registry.Load()

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
		if def := reg.GetSource(name); def != nil {
			desc = def.Description
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

func configCheckExecute(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadRaw(cfgFile)
	if err != nil {
		return fmt.Errorf("config load failed: %w", err)
	}

	reg := registry.Load()
	issues := 0

	fmt.Print("Checking sources...\n\n")
	for _, name := range sortedSourceNames(cfg.Sources) {
		src := cfg.Sources[name]
		def := reg.GetSource(name)
		if def == nil {
			fmt.Printf("  [WARN] %s - not in registry\n", name)
			issues++
			continue
		}

		if !src.Enabled {
			fmt.Printf("  [OFF]  %s\n", name)
			continue
		}

		sourceOK := true

		if len(def.Credentials) > 0 {
			_, credErr := keychain.GetAll(name)
			if credErr != nil {
				fmt.Printf("  [WARN] %s - credentials missing (run: aide credential set %s)\n", name, name)
				issues++
				sourceOK = false
			}
		}

		if sourceOK {
			for _, field := range def.Fields {
				if field.Required {
					val, _ := src.Config[field.Key]
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

	if issues == 0 {
		fmt.Println("\n  All checks passed.")
	} else {
		fmt.Printf("\n  %d issue(s) found.\n", issues)
		return fmt.Errorf("%d configuration issue(s) found", issues)
	}

	return nil
}
