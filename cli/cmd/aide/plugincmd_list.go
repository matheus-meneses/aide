package main

import (
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/plugin"
	"aide/cli/internal/runtime/updater"
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

	var idx *plugin.Index
	if cached, err := plugin.LoadCachedIndex(); err == nil {
		idx = cached
	}

	updates := 0
	widgets.Println("Installed plugins:")
	for _, m := range manifests {
		widgets.Printf("  %-20s %-10s %s%s\n", m.Name, m.Version, pluginStatusLabel(sources, m.Name), pluginUpdateLabel(idx, m))
		if m.Description != "" {
			widgets.Printf("    %s\n", m.Description)
		}
		if idx != nil {
			if entry, ok := idx.Plugins[m.Name]; ok && updater.IsNewer(entry.Latest, m.Version) {
				updates++
			}
		}
	}

	if updates > 0 {
		widgets.Printf("\n%d update(s) available — run 'aide plugin update' to apply.\n", updates)
	}
	if !anyConfigured(sources) {
		widgets.Println("\nRun 'aide plugin configure <name>' to set one up.")
	}
	return nil
}

func pluginUpdateLabel(idx *plugin.Index, m *plugin.Manifest) string {
	if idx == nil {
		return ""
	}
	entry, ok := idx.Plugins[m.Name]
	if !ok || !updater.IsNewer(entry.Latest, m.Version) {
		return ""
	}
	return fmt.Sprintf(" (update → %s)", entry.Latest)
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
