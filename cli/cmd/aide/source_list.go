package main

import (
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/plugin"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

func sourceListExecute(_ *cobra.Command, _ []string) error {
	cfg, err := config.LoadRaw(cfgFile)
	if err != nil {
		return err
	}

	mgr := plugin.NewManager()
	manifests, _ := mgr.List()

	manifestMap := make(map[string]*plugin.Manifest)
	for _, m := range manifests {
		manifestMap[m.Name] = m
	}

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
			if m, ok := manifestMap[name]; ok && m.Description != "" {
				desc = " - " + m.Description
			}
			fmt.Printf("  %-22s [%s]%s\n", name, status, desc)
		}
	}

	var unconfigured []string
	for _, m := range manifests {
		if !configuredSet[m.Name] {
			unconfigured = append(unconfigured, m.Name)
		}
	}
	sort.Strings(unconfigured)

	if len(unconfigured) > 0 {
		fmt.Println("\nInstalled (not yet configured):")
		for _, name := range unconfigured {
			m := manifestMap[name]
			fmt.Printf("  %-22s %s\n", name, m.Description)
		}
		fmt.Println("\nRun 'aide config source add' to set up a source.")
	}

	return nil
}
