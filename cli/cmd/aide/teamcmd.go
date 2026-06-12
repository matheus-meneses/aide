package main

import (
	"aide/cli/internal/config"
	"aide/cli/internal/render"
	"aide/cli/internal/store"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	teamListView   string
	teamListSource string
)

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "View and manage team members",
}

var teamListCmd = &cobra.Command{
	Use:           "list",
	Short:         "List team members",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          teamListExecute,
}

var teamSyncCmd = &cobra.Command{
	Use:           "sync",
	Short:         "Re-resolve manager relationships from stored data",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          teamSyncExecute,
}

func init() {
	teamListCmd.Flags().StringVar(&teamListView, "view", "tree", "output view: tree or flat")
	teamListCmd.Flags().StringVar(&teamListSource, "source", "", "filter by source (config, rh_portal, …)")
	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamSyncCmd)
	rootCmd.AddCommand(teamCmd)
}

func teamListExecute(_ *cobra.Command, _ []string) error {
	return withStore(func(_ *config.Config, s *store.Store) error {
		members, err := s.Team.All()
		if err != nil {
			return err
		}

		if teamListSource != "" {
			filtered := members[:0]
			for _, m := range members {
				if m.Source == teamListSource {
					filtered = append(filtered, m)
				}
			}
			members = filtered
		}

		render.PrintTeamList(members, teamListView)
		return nil
	})
}

func teamSyncExecute(_ *cobra.Command, _ []string) error {
	return withStore(func(_ *config.Config, s *store.Store) error {
		n, err := s.Team.ReresolveManagers()
		if err != nil {
			return err
		}
		fmt.Printf("Re-resolved %d manager relationships.\n", n)
		return nil
	})
}
