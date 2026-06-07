package main

import (
	"github.com/spf13/cobra"

	"aide/cli/internal/config"
	"aide/cli/internal/render"
	"aide/cli/internal/store"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show new and resolved items since last run",
	RunE:  diffExecute,
}

func init() {
	diffCmd.Flags().String("source", "", "filter by source")
	rootCmd.AddCommand(diffCmd)
}

func diffExecute(cmd *cobra.Command, args []string) error {
	source, _ := cmd.Flags().GetString("source")

	return withStore(func(cfg *config.Config, s *store.Store) error {
		return render.PrintDiff(s, source)
	})
}
