package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"aide/cli/internal/config"
	"aide/cli/internal/prompt"
	"aide/cli/internal/registry"
)

func sourceAddExecute(cmd *cobra.Command, args []string) error {
	reg := registry.Load()
	cfg, err := config.LoadRaw(cfgFile)
	if err != nil {
		return err
	}

	var name string
	if len(args) == 1 {
		name = args[0]
		if _, exists := cfg.Sources[name]; exists {
			return fmt.Errorf("source '%s' already configured", name)
		}
	} else {
		if cfg.Sources == nil {
			cfg.Sources = make(map[string]config.Source)
		}
		picked, err := prompt.PickSource(reg, cfg.Sources)
		if err != nil {
			return err
		}
		name = picked
	}

	def := reg.GetSource(name)
	if def == nil {
		return fmt.Errorf("source '%s' not found in registry", name)
	}

	fmt.Printf("\nSetting up %s...\n\n", name)

	sourceCfg, err := prompt.ConfigureSource(def)
	if err != nil {
		return err
	}

	sourceCfg["credentials_env"] = fmt.Sprintf("AIDE_%s", strings.ToUpper(name))

	if cfg.Sources == nil {
		cfg.Sources = make(map[string]config.Source)
	}
	cfg.Sources[name] = config.Source{
		Enabled: true,
		Config:  sourceCfg,
	}

	if err := prompt.SetupCredentials(def, name); err != nil {
		return fmt.Errorf("credential setup failed, source not saved: %w", err)
	}

	if err := cfg.Save(cfgFile); err != nil {
		return err
	}

	fmt.Printf("\n✓ Source '%s' enabled.\n", name)
	fmt.Printf("  Run 'aide run --source %s' to test.\n", name)
	return nil
}
