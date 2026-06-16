package main

import (
	"aide/cli/internal/platform/clog"
	"aide/cli/internal/runtime/plugin"
	"aide/cli/internal/setup/provision"
	"aide/cli/internal/ui/widgets"
	"fmt"

	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage aide plugins",
}

var (
	pluginListAvailable   bool
	pluginRegistryURL     string
	pluginRegistryVersion string
)

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins (--available to show registry catalog)",
	RunE:  pluginListExecute,
}

var pluginInstallLocal string

var pluginInstallCmd = &cobra.Command{
	Use:   "install [name[@version]]",
	Short: "Install a plugin from the registry (or --local <path> for local dev)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  pluginInstallExecute,
}

var pluginRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an installed plugin",
	Args:  cobra.ExactArgs(1),
	RunE:  pluginRemoveExecute,
}

var pluginUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Refresh the registry cache (alias of 'aide registry refresh')",
	Aliases: []string{"refresh"},
	RunE:    pluginUpdateExecute,
}

var pluginAuthCmd = &cobra.Command{
	Use:   "auth <source>",
	Short: "Authenticate a browser-based source interactively",
	Args:  cobra.ExactArgs(1),
	RunE:  pluginAuthExecute,
}

func init() {
	pluginListCmd.Flags().BoolVar(&pluginListAvailable, "available", false, "show available plugins from registry cache")
	pluginInstallCmd.Flags().StringVar(&pluginRegistryURL, "registry", "", "extra registry URL to include in merge")
	pluginInstallCmd.Flags().StringVar(&pluginRegistryVersion, "registry-version", "", "registry release version/tag to pull the index from (default: latest)")
	pluginInstallCmd.Flags().StringVar(&pluginInstallLocal, "local", "", "install from a local directory instead of the registry")
	pluginUpdateCmd.Flags().StringVar(&pluginRegistryURL, "registry", "", "extra registry URL to include in merge")
	pluginUpdateCmd.Flags().StringVar(&pluginRegistryVersion, "registry-version", "", "registry release version/tag to pull the index from (default: latest)")

	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginRemoveCmd)
	pluginCmd.AddCommand(pluginUpdateCmd)
	pluginCmd.AddCommand(pluginAuthCmd)
	rootCmd.AddCommand(pluginCmd)
}

func pluginRemoveExecute(_ *cobra.Command, args []string) error {
	name := args[0]
	if err := requireConfirm(fmt.Sprintf("Remove plugin '%s' (and its source + stored credentials)?", name)); err != nil {
		return err
	}
	if err := provision.UninstallPlugin(cfgFile, name); err != nil {
		return err
	}
	widgets.PrintSuccess("Plugin '%s' removed.", name)
	return nil
}

func pluginUpdateExecute(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	extraRegistries := cfg.Registries
	if pluginRegistryURL != "" {
		extraRegistries = append(extraRegistries, pluginRegistryURL)
	}
	if pluginRegistryVersion != "" {
		plugin.SetRegistryVersion(pluginRegistryVersion)
	}

	clog.Info("fetching registry")
	idx, err := plugin.MergedIndex(extraRegistries)
	if err != nil {
		return fmt.Errorf("fetching registry: %w", err)
	}

	if err := plugin.CacheIndex(idx); err != nil {
		return fmt.Errorf("caching index: %w", err)
	}

	clog.Info("registry updated: %d plugins available", len(idx.Plugins))
	return nil
}
