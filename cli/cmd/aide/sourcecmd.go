package main

import (
	"github.com/spf13/cobra"
)

var sourceRemoveYes bool

var sourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Manage configured sources",
}

var sourceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured and available sources",
	RunE:  sourceListExecute,
}

var sourceAddCmd = &cobra.Command{
	Use:           "add [name]",
	Short:         "Add and configure a new source interactively",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          sourceAddExecute,
}

var sourceRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a source from configuration",
	Args:  cobra.ExactArgs(1),
	RunE:  sourceRemoveExecute,
}

var sourceEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a source",
	Args:  cobra.ExactArgs(1),
	RunE:  sourceEnableExecute,
}

var sourceDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a source",
	Args:  cobra.ExactArgs(1),
	RunE:  sourceDisableExecute,
}

var sourceSetCmd = &cobra.Command{
	Use:   "set <name> <key> <value>",
	Short: "Set a config value for a source",
	Args:  cobra.ExactArgs(3),
	RunE:  sourceSetExecute,
}

func init() {
	sourceRemoveCmd.Flags().BoolVar(&sourceRemoveYes, "yes", false, "Skip confirmation prompt")

	sourceCmd.AddCommand(sourceListCmd)
	sourceCmd.AddCommand(sourceAddCmd)
	sourceCmd.AddCommand(sourceRemoveCmd)
	sourceCmd.AddCommand(sourceEnableCmd)
	sourceCmd.AddCommand(sourceDisableCmd)
	sourceCmd.AddCommand(sourceSetCmd)
	configCmd.AddCommand(sourceCmd)
}
