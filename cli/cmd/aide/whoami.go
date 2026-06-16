package main

import (
	"aide/cli/internal/persistence/store"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/ui/widgets"
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	whoamiSet      bool
	whoamiName     string
	whoamiEmail    string
	whoamiNickname string
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show or set your identity",
	RunE:  whoamiExecute,
}

func init() {
	whoamiCmd.Flags().BoolVar(&whoamiSet, "set", false, "Re-run identity setup")
	whoamiCmd.Flags().StringVar(&whoamiName, "name", "", "Full name (non-interactive)")
	whoamiCmd.Flags().StringVar(&whoamiEmail, "email", "", "Email (non-interactive)")
	whoamiCmd.Flags().StringVar(&whoamiNickname, "nickname", "", "How Aide should call you (non-interactive)")
	rootCmd.AddCommand(whoamiCmd)
}

func whoamiExecute(_ *cobra.Command, _ []string) error {
	return withStore(func(_ *config.Config, s *store.Store) error {
		if whoamiName != "" || whoamiEmail != "" || whoamiNickname != "" {
			if whoamiName == "" {
				return fmt.Errorf("--name is required when setting identity non-interactively")
			}
			if err := s.Profile.SetIdentity(whoamiName, whoamiEmail, whoamiNickname); err != nil {
				return err
			}
			widgets.PrintSuccess("Identity saved for %s.", whoamiName)
			return nil
		}

		if !whoamiSet {
			profile, err := s.Profile.All()
			if err == nil && len(profile) > 0 {
				fmt.Printf("Name:           %s\n", profile["name"])
				fmt.Printf("Email:          %s\n", profile["email"])
				fmt.Printf("Preferred name: %s\n", profile["preferred_name"])
				return nil
			}
		}

		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Full name: ")
		name, _ := reader.ReadString('\n')
		name = strings.TrimSpace(name)

		fmt.Print("Email: ")
		email, _ := reader.ReadString('\n')
		email = strings.TrimSpace(email)

		fmt.Print("How should Aide call you? ")
		preferred, _ := reader.ReadString('\n')
		preferred = strings.TrimSpace(preferred)

		if err := s.Profile.SetIdentity(name, email, preferred); err != nil {
			return err
		}
		if preferred == "" {
			if fields := strings.Fields(name); len(fields) > 0 {
				preferred = fields[0]
			} else {
				preferred = "there"
			}
		}

		fmt.Printf("\nSaved! Hi %s.\n", preferred)
		return nil
	})
}
