package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"aide/cli/internal/config"
	"aide/cli/internal/keychain"
)

func sourceRemoveExecute(cmd *cobra.Command, args []string) error {
	name := args[0]
	cfg, err := config.LoadRaw(cfgFile)
	if err != nil {
		return err
	}

	if _, exists := cfg.Sources[name]; !exists {
		return fmt.Errorf("source '%s' not found", name)
	}

	if !sourceRemoveYes && !confirm(fmt.Sprintf("Remove source '%s' from config?", name)) {
		fmt.Println("Aborted.")
		return nil
	}

	delete(cfg.Sources, name)

	if err := cfg.Save(cfgFile); err != nil {
		return err
	}

	fmt.Printf("Source '%s' removed.\n", name)

	if _, err := keychain.GetAll(name); err == nil {
		if sourceRemoveYes || confirm(fmt.Sprintf("Also delete stored credentials for '%s'?", name)) {
			if err := keychain.DeleteSource(name); err != nil {
				fmt.Printf("Warning: failed to delete credentials: %v\n", err)
			} else {
				fmt.Printf("Credentials for '%s' deleted.\n", name)
			}
		}
	}

	return nil
}
