package main

import (
	"aide/cli/internal/config"
	"aide/cli/internal/provision"
	"aide/cli/internal/render"
	"aide/cli/internal/runner"
	"aide/cli/internal/store"
	"aide/cli/internal/ui"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	teamListView   string
	teamListSource string
)

var (
	teamEmail        string
	teamRole         string
	teamDepartment   string
	teamBranch       string
	teamRegistration string
	teamManager      string
	teamAliases      string
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

var teamAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a team member to config",
	Args:  cobra.ExactArgs(1),
	RunE:  teamAddExecute,
}

var teamEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit an existing team member (only provided flags change)",
	Args:  cobra.ExactArgs(1),
	RunE:  teamEditExecute,
}

var teamRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a team member from config",
	Args:  cobra.ExactArgs(1),
	RunE:  teamRemoveExecute,
}

func registerTeamMemberFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&teamEmail, "email", "", "member email")
	cmd.Flags().StringVar(&teamRole, "role", "", "member role")
	cmd.Flags().StringVar(&teamDepartment, "department", "", "member department")
	cmd.Flags().StringVar(&teamBranch, "branch", "", "member branch")
	cmd.Flags().StringVar(&teamRegistration, "registration", "", "member registration id")
	cmd.Flags().StringVar(&teamManager, "manager", "", "manager name or registration")
	cmd.Flags().StringVar(&teamAliases, "aliases", "", "comma-separated aliases")
}

func init() {
	teamListCmd.Flags().StringVar(&teamListView, "view", "tree", "output view: tree or flat")
	teamListCmd.Flags().StringVar(&teamListSource, "source", "", "filter by source (config, rh_portal, …)")
	registerTeamMemberFlags(teamAddCmd)
	registerTeamMemberFlags(teamEditCmd)
	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamSyncCmd)
	teamCmd.AddCommand(teamAddCmd)
	teamCmd.AddCommand(teamEditCmd)
	teamCmd.AddCommand(teamRemoveCmd)
	rootCmd.AddCommand(teamCmd)
}

func parseAliases(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if a := strings.TrimSpace(part); a != "" {
			out = append(out, a)
		}
	}
	return out
}

func syncTeamToStore() error {
	return withStore(func(cfg *config.Config, s *store.Store) error {
		return runner.SyncTeamFromConfig(cfg, s)
	})
}

func teamAddExecute(_ *cobra.Command, args []string) error {
	name := args[0]
	members, err := provision.GetTeam(cfgFile)
	if err != nil {
		return err
	}
	for _, m := range members {
		if m.Name == name {
			return fmt.Errorf("team member %q already exists (use 'aide team edit')", name)
		}
	}
	members = append(members, config.TeamMember{
		Name:         name,
		Email:        teamEmail,
		Role:         teamRole,
		Department:   teamDepartment,
		Branch:       teamBranch,
		Registration: teamRegistration,
		Manager:      teamManager,
		Aliases:      parseAliases(teamAliases),
	})
	if err := provision.SetTeam(cfgFile, members); err != nil {
		return err
	}
	if err := syncTeamToStore(); err != nil {
		return err
	}
	ui.PrintSuccess("Added team member %q.", name)
	return nil
}

func teamEditExecute(cmd *cobra.Command, args []string) error {
	name := args[0]
	members, err := provision.GetTeam(cfgFile)
	if err != nil {
		return err
	}
	idx := -1
	for i, m := range members {
		if m.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("team member %q not found", name)
	}

	f := cmd.Flags()
	if f.Changed("email") {
		members[idx].Email = teamEmail
	}
	if f.Changed("role") {
		members[idx].Role = teamRole
	}
	if f.Changed("department") {
		members[idx].Department = teamDepartment
	}
	if f.Changed("branch") {
		members[idx].Branch = teamBranch
	}
	if f.Changed("registration") {
		members[idx].Registration = teamRegistration
	}
	if f.Changed("manager") {
		members[idx].Manager = teamManager
	}
	if f.Changed("aliases") {
		members[idx].Aliases = parseAliases(teamAliases)
	}

	if err := provision.SetTeam(cfgFile, members); err != nil {
		return err
	}
	if err := syncTeamToStore(); err != nil {
		return err
	}
	ui.PrintSuccess("Updated team member %q.", name)
	return nil
}

func teamRemoveExecute(_ *cobra.Command, args []string) error {
	name := args[0]
	members, err := provision.GetTeam(cfgFile)
	if err != nil {
		return err
	}
	filtered := members[:0]
	found := false
	for _, m := range members {
		if m.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, m)
	}
	if !found {
		return fmt.Errorf("team member %q not found", name)
	}
	if err := provision.SetTeam(cfgFile, filtered); err != nil {
		return err
	}
	if err := syncTeamToStore(); err != nil {
		return err
	}
	ui.PrintSuccess("Removed team member %q.", name)
	return nil
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
