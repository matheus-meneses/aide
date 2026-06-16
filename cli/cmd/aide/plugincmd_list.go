package main

import (
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/plugin"
	"aide/cli/internal/ui/widgets"
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
	manifests, err := mgr.List()
	if err != nil {
		return fmt.Errorf("listing plugins: %w", err)
	}

	if len(manifests) == 0 {
		widgets.Println("No plugins installed.")
		widgets.Println("Run 'aide plugin install <name>' to install a plugin.")
		return nil
	}

	sources := map[string]config.Source{}
	if cfg, cfgErr := loadRawConfig(); cfgErr == nil {
		sources = cfg.Sources
	}

	sort.Slice(manifests, func(i, j int) bool { return manifests[i].Name < manifests[j].Name })

	widgets.Println("Installed plugins:")
	for _, m := range manifests {
		widgets.Printf("  %-20s %-10s %s\n", m.Name, m.Version, pluginStatusLabel(sources, m.Name))
		if m.Description != "" {
			widgets.Printf("    %s\n", m.Description)
		}
	}

	if !anyConfigured(sources) {
		widgets.Println("\nRun 'aide plugin configure <name>' to set one up.")
	}
	return nil
}

func pluginStatusLabel(sources map[string]config.Source, name string) string {
	src, ok := sources[name]
	if !ok {
		return "[not configured]"
	}
	if src.Enabled {
		return "[enabled]"
	}
	return "[disabled]"
}

func anyConfigured(sources map[string]config.Source) bool {
	return len(sources) > 0
}

func pluginListAvailableExecute() error {
	idx, err := plugin.LoadCachedIndex()
	if err != nil {
		return fmt.Errorf("no registry cache found — run 'aide plugin registry refresh' to fetch: %w", err)
	}

	if len(idx.Plugins) == 0 {
		widgets.Println("Registry is empty.")
		return nil
	}

	names := make([]string, 0, len(idx.Plugins))
	for n := range idx.Plugins {
		names = append(names, n)
	}
	sort.Strings(names)

	widgets.Println("Available plugins:")
	for _, name := range names {
		entry := idx.Plugins[name]
		widgets.Printf("  %-20s %s\n", name, entry.Latest)
	}
	return nil
}
