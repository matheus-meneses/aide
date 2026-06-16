package main

import (
	"aide/cli/internal/ui/render"

	"github.com/spf13/cobra"
)

var sourcesCmd = &cobra.Command{
	Use:   "sources",
	Short: "List available scrapers and their health status",
	RunE:  sourcesExecute,
}

func init() {
	rootCmd.AddCommand(sourcesCmd)
}

func sourcesExecute(_ *cobra.Command, _ []string) error {
	return withStore(render.PrintSources)
}
