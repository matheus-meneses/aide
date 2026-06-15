package main

import (
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/plugin"
	"aide/cli/internal/setup/provision"
	"aide/cli/internal/ui/prompt"
	"fmt"

	"github.com/spf13/cobra"
)

func sourceAddExecute(_ *cobra.Command, args []string) error {
	cfg, err := config.LoadRaw(cfgFile)
	if err != nil {
		return err
	}

	mgr := plugin.NewManager()

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
		picked, err := prompt.PickPlugin(mgr, cfg.Sources)
		if err != nil {
			return err
		}
		name = picked
	}

	m, err := mgr.Get(name)
	if err != nil {
		return fmt.Errorf("plugin '%s' is not installed — run 'aide plugin install %s' first", name, name)
	}

	fmt.Printf("\nSetting up %s...\n\n", name)

	sourceCfg, err := prompt.ConfigurePlugin(m, nil)
	if err != nil {
		return err
	}

	creds, err := prompt.CollectPluginCredentials(m)
	if err != nil {
		return fmt.Errorf("credential setup failed, source not saved: %w", err)
	}

	if err := provision.AddSource(cfgFile, provision.SourceInput{
		Name:        name,
		Config:      sourceCfg,
		Credentials: creds,
	}); err != nil {
		return err
	}

	fmt.Printf("\n✓ Source '%s' enabled.\n", name)
	fmt.Printf("  Run 'aide run --source %s' to test.\n", name)
	return nil
}
