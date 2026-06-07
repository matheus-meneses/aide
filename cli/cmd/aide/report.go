package main

import (
	"github.com/spf13/cobra"

	"aide/cli/internal/config"
	"aide/cli/internal/render"
	"aide/cli/internal/store"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Show latest consolidated report",
	RunE:  reportExecute,
}

func init() {
	reportCmd.Flags().String("member", "", "filter by team member")
	reportCmd.Flags().String("category", "", "filter by category")
	rootCmd.AddCommand(reportCmd)
}

func reportExecute(cmd *cobra.Command, args []string) error {
	member, _ := cmd.Flags().GetString("member")
	category, _ := cmd.Flags().GetString("category")

	return withStore(func(cfg *config.Config, s *store.Store) error {
		return render.PrintReport(s, member, category)
	})
}
