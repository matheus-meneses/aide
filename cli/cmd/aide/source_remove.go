package main

import (
	"aide/cli/internal/config"
	"aide/cli/internal/keychain"
	"fmt"

	"github.com/spf13/cobra"
)

func sourceRemoveExecute(_ *cobra.Command, args []string) error {
	name := args[0]
	cfg, err := config.LoadRaw(cfgFile)
	if err != nil {
		return err
	}

	if _, exists := cfg.Sources[name]; !exists {
		return fmt.Errorf("source '%s' not found", name)
	}

	if err := requireConfirm(fmt.Sprintf("Remove source '%s' from config?", name)); err != nil {
		return err
	}

	delete(cfg.Sources, name)

	if err := cfg.Save(cfgFile); err != nil {
		return err
	}

	fmt.Printf("Source '%s' removed.\n", name)

	if _, err := keychain.GetAll(name); err == nil {
		if confirm(fmt.Sprintf("Also delete stored credentials for '%s'?", name)) {
			if err := keychain.DeleteSource(name); err != nil {
				fmt.Printf("Warning: failed to delete credentials: %v\n", err)
			} else {
				fmt.Printf("Credentials for '%s' deleted.\n", name)
			}
		}
	}

	return nil
}
