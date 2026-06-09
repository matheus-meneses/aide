package main

import (
	"aide/cli/internal/config"
	"aide/cli/internal/store"
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var whoamiSet bool

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show or set your identity",
	RunE:  whoamiExecute,
}

func init() {
	whoamiCmd.Flags().BoolVar(&whoamiSet, "set", false, "Re-run identity setup")
	rootCmd.AddCommand(whoamiCmd)
}

func whoamiExecute(_ *cobra.Command, _ []string) error {
	return withStore(func(_ *config.Config, s *store.Store) error {
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
		if preferred == "" {
			if fields := strings.Fields(name); len(fields) > 0 {
				preferred = fields[0]
			} else {
				preferred = "there"
			}
		}

		if err := s.Profile.Set("name", name); err != nil {
			return err
		}
		if err := s.Profile.Set("email", email); err != nil {
			return err
		}
		if err := s.Profile.Set("preferred_name", preferred); err != nil {
			return err
		}

		fmt.Printf("\nSaved! Hi %s.\n", preferred)
		return nil
	})
}
