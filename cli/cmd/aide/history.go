package main

import (
	"github.com/spf13/cobra"

	"aide/cli/internal/config"
	"aide/cli/internal/render"
	"aide/cli/internal/store"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show past run history",
	RunE:  historyExecute,
}

func init() {
	rootCmd.AddCommand(historyCmd)
}

func historyExecute(cmd *cobra.Command, args []string) error {
	return withStore(func(cfg *config.Config, s *store.Store) error {
		return render.PrintHistory(s)
	})
}
