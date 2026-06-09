package main

import (
	"aide/cli/internal/config"
	"aide/cli/internal/render"
	"aide/cli/internal/store"

	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show past run history",
	RunE:  historyExecute,
}

func init() {
	rootCmd.AddCommand(historyCmd)
}

func historyExecute(_ *cobra.Command, _ []string) error {
	return withStore(func(_ *config.Config, s *store.Store) error {
		return render.PrintHistory(s)
	})
}
