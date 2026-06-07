package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"aide/cli/internal/config"
	"aide/cli/internal/registry"
)

func sourceListExecute(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadRaw(cfgFile)
	if err != nil {
		return err
	}

	reg := registry.Load()
	configuredSet := make(map[string]bool)

	if len(cfg.Sources) > 0 {
		fmt.Println("Configured:")
		configuredNames := make([]string, 0, len(cfg.Sources))
		for name := range cfg.Sources {
			configuredNames = append(configuredNames, name)
		}
		sort.Strings(configuredNames)
		for _, name := range configuredNames {
			src := cfg.Sources[name]
			configuredSet[name] = true
			status := "disabled"
			if src.Enabled {
				status = "enabled"
			}
			desc := ""
			if def := reg.GetSource(name); def != nil {
				desc = " - " + def.Description
			}
			fmt.Printf("  %-22s [%s]%s\n", name, status, desc)
		}
	}

	names := reg.ListSources()
	var unconfigured []string
	for _, name := range names {
		if !configuredSet[name] {
			unconfigured = append(unconfigured, name)
		}
	}
	if len(unconfigured) > 0 {
		fmt.Println("\nAvailable:")
		for _, name := range unconfigured {
			def := reg.GetSource(name)
			fmt.Printf("  %-22s %s\n", name, def.Description)
		}
		fmt.Println("\nRun 'aide config source add' to set up a source.")
	}

	return nil
}
