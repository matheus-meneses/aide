package main

import (
	"aide/cli/internal/runtime/plugin"
	"aide/cli/internal/runtime/updater"
	"aide/cli/internal/ui/widgets"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var pluginUpdateCheck bool

var pluginUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update installed plugins to the latest registry version",
	Long: "Update one plugin (by name) or every installed plugin (no name) to the " +
		"latest version published in the configured registries. Use --check to only " +
		"report what would update without changing anything.",
	Args: cobra.MaximumNArgs(1),
	RunE: pluginUpdateExecute,
}

type pluginUpgrade struct {
	name      string
	installed string
	latest    string
}

func pluginUpdateExecute(cmd *cobra.Command, args []string) error {
	cfg, cfgErr := loadConfig()
	if cfgErr != nil {
		return cfgErr
	}

	sp := widgets.NewSpinner("Fetching registry…")
	sp.Start()
	idx, idxErr := plugin.ResolveIndex(cfg.Registries)
	sp.Stop()
	if idxErr != nil {
		return idxErr
	}

	installed, err := plugin.NewManager().List()
	if err != nil {
		return fmt.Errorf("listing installed plugins: %w", err)
	}

	want := ""
	if len(args) > 0 {
		want = args[0]
	}

	var upgrades []pluginUpgrade
	for _, m := range installed {
		if want != "" && m.Name != want {
			continue
		}
		entry, ok := idx.Plugins[m.Name]
		if !ok {
			continue
		}
		if updater.IsNewer(entry.Latest, m.Version) {
			upgrades = append(upgrades, pluginUpgrade{name: m.Name, installed: m.Version, latest: entry.Latest})
		}
	}
	sort.Slice(upgrades, func(i, j int) bool { return upgrades[i].name < upgrades[j].name })

	if want != "" && !pluginIsInstalled(installed, want) {
		return fmt.Errorf("plugin %q is not installed", want)
	}

	if len(upgrades) == 0 {
		if want != "" {
			widgets.Printf("%s is already up to date.\n", want)
		} else {
			widgets.Println("All plugins are up to date.")
		}
		return nil
	}

	widgets.Println("Updates available:")
	for _, u := range upgrades {
		widgets.Printf("  %-20s %s → %s\n", u.name, u.installed, u.latest)
	}
	if pluginUpdateCheck {
		return nil
	}

	var failed []string
	for _, u := range upgrades {
		consent := func(m *plugin.Manifest) bool {
			if assumeYes {
				return true
			}
			widgets.Printf("\nUpdate %s: %s → %s\n", u.name, u.installed, u.latest)
			if m.Description != "" {
				widgets.Printf("Description: %s\n", m.Description)
			}
			printPluginCapabilities(m)
			return confirm("Apply this update?")
		}
		if _, err := plugin.Install(cmd.Context(), idx, u.name, u.latest, consent); err != nil {
			widgets.PrintWarn("failed to update %s: %v", u.name, err)
			failed = append(failed, u.name)
			continue
		}
		widgets.PrintSuccess("Updated %s to %s.", u.name, u.latest)
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to update: %s", strings.Join(failed, ", "))
	}
	return nil
}

func printPluginCapabilities(m *plugin.Manifest) {
	if len(m.Capabilities.Network) > 0 {
		widgets.Printf("Network access: %s\n", strings.Join(m.Capabilities.Network, ", "))
	}
	if len(m.Capabilities.Filesystem) > 0 {
		paths := make([]string, 0, len(m.Capabilities.Filesystem))
		for _, f := range m.Capabilities.Filesystem {
			if f.Read != "" {
				paths = append(paths, "r:"+f.Read)
			}
			if f.Write != "" {
				paths = append(paths, "w:"+f.Write)
			}
		}
		widgets.Printf("Filesystem access: %s\n", strings.Join(paths, ", "))
	}
}

func pluginIsInstalled(installed []*plugin.Manifest, name string) bool {
	for _, m := range installed {
		if m.Name == name {
			return true
		}
	}
	return false
}
