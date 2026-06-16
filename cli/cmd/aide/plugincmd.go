package main

import (
	"aide/cli/internal/setup/provision"
	"aide/cli/internal/ui/render"
	"aide/cli/internal/ui/widgets"
	"fmt"

	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Install, configure, and manage data-source plugins",
}

var (
	pluginListAvailable   bool
	pluginRegistryURL     string
	pluginRegistryVersion string
	pluginInstallLocal    string
)

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins and their configured/enabled status",
	RunE:  pluginListExecute,
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install [name[@version]]",
	Short: "Install a plugin from the registry (or --local <path> for local dev)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  pluginInstallExecute,
}

var pluginConfigureCmd = &cobra.Command{
	Use:           "configure [name]",
	Aliases:       []string{"add", "edit", "reconfigure"},
	Short:         "Configure a plugin as a source interactively (settings + credentials)",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          pluginConfigureExecute,
}

var pluginEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a configured plugin",
	Args:  cobra.ExactArgs(1),
	RunE:  sourceEnableExecute,
}

var pluginDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a configured plugin",
	Args:  cobra.ExactArgs(1),
	RunE:  sourceDisableExecute,
}

var pluginSetCmd = &cobra.Command{
	Use:   "set <name> <key> <value>",
	Short: "Set a config value for a configured plugin",
	Args:  cobra.ExactArgs(3),
	RunE:  sourceSetExecute,
}

var pluginRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an installed plugin (and its config + stored credentials)",
	Args:  cobra.ExactArgs(1),
	RunE:  pluginRemoveExecute,
}

var pluginStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show configured plugins with their health from run history",
	RunE:  pluginStatusExecute,
}

func init() {
	pluginListCmd.Flags().BoolVar(&pluginListAvailable, "available", false, "show the registry catalog instead of installed plugins")
	pluginInstallCmd.Flags().StringVar(&pluginRegistryURL, "registry", "", "extra registry URL to include in merge")
	pluginInstallCmd.Flags().StringVar(&pluginRegistryVersion, "registry-version", "", "registry release version/tag to pull the index from (default: latest)")
	pluginInstallCmd.Flags().StringVar(&pluginInstallLocal, "local", "", "install from a local directory instead of the registry")

	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginConfigureCmd)
	pluginCmd.AddCommand(pluginEnableCmd)
	pluginCmd.AddCommand(pluginDisableCmd)
	pluginCmd.AddCommand(pluginSetCmd)
	pluginCmd.AddCommand(pluginRemoveCmd)
	pluginCmd.AddCommand(pluginStatusCmd)
	pluginCmd.AddCommand(pluginRegistryCmd)
	rootCmd.AddCommand(pluginCmd)
}

func pluginRemoveExecute(_ *cobra.Command, args []string) error {
	name := args[0]
	if err := requireConfirm(fmt.Sprintf("Remove plugin '%s' (and its config + stored credentials)?", name)); err != nil {
		return err
	}
	if err := provision.UninstallPlugin(cfgFile, name); err != nil {
		return err
	}
	widgets.PrintSuccess("Plugin '%s' removed.", name)
	return nil
}

func pluginStatusExecute(_ *cobra.Command, _ []string) error {
	return withStore(render.PrintSources)
}
