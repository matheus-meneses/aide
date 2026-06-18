package main

import (
	"aide/cli/internal/persistence/store"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/ui/render"
	"aide/cli/internal/ui/widgets"
	"encoding/json"
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
	Short: "Add a team member",
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
	Short: "Remove a team member",
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
	teamListCmd.Flags().StringVar(&teamListSource, "source", "", "filter by source (manual, rh_portal, …)")
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

func aliasesJSON(s string) string {
	al := parseAliases(s)
	if len(al) == 0 {
		return "[]"
	}
	b, err := json.Marshal(al)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func teamAddExecute(_ *cobra.Command, args []string) error {
	name := args[0]
	return withStore(func(_ *config.Config, s *store.Store) error {
		existing, err := s.Team.All()
		if err != nil {
			return err
		}
		for _, m := range existing {
			if m.Name == name {
				return fmt.Errorf("team member %q already exists (use 'aide team edit')", name)
			}
		}
		if _, err := s.Team.Add(store.Member{
			Name:                name,
			Email:               teamEmail,
			Aliases:             aliasesJSON(teamAliases),
			Role:                teamRole,
			Department:          teamDepartment,
			Branch:              teamBranch,
			Registration:        teamRegistration,
			ManagerRef:          teamManager,
			ManagerRegistration: teamManager,
			Source:              "manual",
		}); err != nil {
			return err
		}
		widgets.PrintSuccess("Added team member %q.", name)
		return nil
	})
}

func teamEditExecute(cmd *cobra.Command, args []string) error {
	name := args[0]
	return withStore(func(_ *config.Config, s *store.Store) error {
		members, err := s.Team.All()
		if err != nil {
			return err
		}
		var target *store.Member
		for i := range members {
			if members[i].Name == name {
				target = &members[i]
				break
			}
		}
		if target == nil {
			return fmt.Errorf("team member %q not found", name)
		}

		f := cmd.Flags()
		if f.Changed("email") {
			target.Email = teamEmail
		}
		if f.Changed("role") {
			target.Role = teamRole
		}
		if f.Changed("department") {
			target.Department = teamDepartment
		}
		if f.Changed("branch") {
			target.Branch = teamBranch
		}
		if f.Changed("registration") {
			target.Registration = teamRegistration
		}
		if f.Changed("manager") {
			target.ManagerRef = teamManager
			target.ManagerRegistration = teamManager
			target.ManagerID = nil
		}
		if f.Changed("aliases") {
			target.Aliases = aliasesJSON(teamAliases)
		}

		if err := s.Team.Update(target.ID, *target); err != nil {
			return err
		}
		widgets.PrintSuccess("Updated team member %q.", name)
		return nil
	})
}

func teamRemoveExecute(_ *cobra.Command, args []string) error {
	name := args[0]
	return withStore(func(_ *config.Config, s *store.Store) error {
		members, err := s.Team.All()
		if err != nil {
			return err
		}
		id := int64(-1)
		for _, m := range members {
			if m.Name == name {
				id = m.ID
				break
			}
		}
		if id < 0 {
			return fmt.Errorf("team member %q not found", name)
		}
		if err := s.Team.Delete(id); err != nil {
			return err
		}
		widgets.PrintSuccess("Removed team member %q.", name)
		return nil
	})
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
		widgets.Printf("Re-resolved %d manager relationships.\n", n)
		return nil
	})
}
