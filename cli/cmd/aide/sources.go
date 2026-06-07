package main

import (
	"github.com/spf13/cobra"

	"aide/cli/internal/config"
	"aide/cli/internal/render"
	"aide/cli/internal/store"
)

var sourcesCmd = &cobra.Command{
	Use:   "sources",
	Short: "List available scrapers and their health status",
	RunE:  sourcesExecute,
}

func init() {
	rootCmd.AddCommand(sourcesCmd)
}

func sourcesExecute(cmd *cobra.Command, args []string) error {
	return withStore(func(cfg *config.Config, s *store.Store) error {
		return render.PrintSources(cfg, s)
	})
}
