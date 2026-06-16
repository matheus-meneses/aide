package main

import (
	"aide/cli/internal/setup/provision"
	"aide/cli/internal/ui/widgets"
	"encoding/json"

	"github.com/spf13/cobra"
)

var (
	registryAddToken string
	registryListJSON bool
)

var pluginRegistryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Manage the plugin registries the catalog is merged from",
}

var registryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured plugin registries",
	RunE:  registryListExecute,
}

var registryAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a plugin registry (use --token for a private registry)",
	Args:  cobra.ExactArgs(1),
	RunE:  registryAddExecute,
}

var registryRemoveCmd = &cobra.Command{
	Use:   "remove <url>",
	Short: "Remove a plugin registry",
	Args:  cobra.ExactArgs(1),
	RunE:  registryRemoveExecute,
}

var registryRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Re-fetch and cache the merged plugin catalog",
	RunE:  registryRefreshExecute,
}

func init() {
	registryAddCmd.Flags().StringVar(&registryAddToken, "token", "", "auth token for a private registry (stored in the OS keychain)")
	registryListCmd.Flags().BoolVar(&registryListJSON, "json", false, "output as JSON")
	pluginRegistryCmd.AddCommand(registryListCmd)
	pluginRegistryCmd.AddCommand(registryAddCmd)
	pluginRegistryCmd.AddCommand(registryRemoveCmd)
	pluginRegistryCmd.AddCommand(registryRefreshCmd)
}

func registryListExecute(_ *cobra.Command, _ []string) error {
	registries, err := provision.ListRegistries(cfgFile)
	if err != nil {
		return err
	}
	if registryListJSON {
		enc := json.NewEncoder(widgets.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(registries)
	}
	if len(registries) == 0 {
		widgets.PrintInfo("No custom registries configured (the default registry is always included).")
		return nil
	}
	widgets.Heading("Registries")
	for _, r := range registries {
		widgets.Bullet("%s", r)
	}
	return nil
}

func registryAddExecute(_ *cobra.Command, args []string) error {
	if err := provision.AddRegistry(cfgFile, args[0], registryAddToken); err != nil {
		return err
	}
	widgets.PrintSuccess("Registry added: %s", args[0])
	widgets.PrintInfo("Run 'aide plugin registry refresh' to update the catalog.")
	return nil
}

func registryRemoveExecute(_ *cobra.Command, args []string) error {
	if err := provision.RemoveRegistry(cfgFile, args[0]); err != nil {
		return err
	}
	widgets.PrintSuccess("Registry removed: %s", args[0])
	return nil
}

func registryRefreshExecute(_ *cobra.Command, _ []string) error {
	count, err := provision.RefreshCatalog(cfgFile)
	if err != nil {
		return err
	}
	widgets.PrintSuccess("Catalog refreshed: %d plugin(s) available.", count)
	return nil
}
