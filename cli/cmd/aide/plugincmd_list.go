package main

import (
	"aide/cli/internal/plugin"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

func pluginListExecute(_ *cobra.Command, _ []string) error {
	if pluginListAvailable {
		return pluginListAvailableExecute()
	}
	return pluginListInstalledExecute()
}

func pluginListInstalledExecute() error {
	mgr := plugin.NewManager()
	groups, err := mgr.GroupByCategory()
	if err != nil {
		return fmt.Errorf("listing plugins: %w", err)
	}

	if len(groups) == 0 {
		fmt.Println("No plugins installed.")
		fmt.Println("Run 'aide plugin install <name>' to install a plugin.")
		return nil
	}

	cats := make([]string, 0, len(groups))
	for c := range groups {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	fmt.Println("Installed plugins:")
	for _, cat := range cats {
		fmt.Printf("\n  [%s]\n", cat)
		for _, m := range groups[cat] {
			fmt.Printf("    %-20s %s  (%s)\n", m.Name, m.Version, m.Runtime)
			if m.Description != "" {
				fmt.Printf("    %s\n", m.Description)
			}
		}
	}
	return nil
}

func pluginListAvailableExecute() error {
	idx, err := plugin.LoadCachedIndex()
	if err != nil {
		return fmt.Errorf("no registry cache found — run 'aide plugin update' to fetch: %w", err)
	}

	if len(idx.Plugins) == 0 {
		fmt.Println("Registry is empty.")
		return nil
	}

	names := make([]string, 0, len(idx.Plugins))
	for n := range idx.Plugins {
		names = append(names, n)
	}
	sort.Strings(names)

	fmt.Println("Available plugins:")
	for _, name := range names {
		entry := idx.Plugins[name]
		fmt.Printf("  %-20s %s\n", name, entry.Latest)
	}
	return nil
}
