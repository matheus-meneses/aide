package main

import (
	"aide/cli/internal/config"
	"aide/cli/internal/plugin"
	"aide/cli/internal/prompt"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

func sourceConfigureExecute(_ *cobra.Command, args []string) error {
	cfg, err := config.LoadRaw(cfgFile)
	if err != nil {
		return err
	}

	var name string
	if len(args) == 1 {
		name = args[0]
	} else {
		name, err = prompt.PickConfiguredSource(cfg.Sources)
		if err != nil {
			return err
		}
	}

	src, exists := cfg.Sources[name]
	if !exists {
		return fmt.Errorf("source '%s' not configured — run 'aide config source add %s' first", name, name)
	}

	mgr := plugin.NewManager()
	m, err := mgr.Get(name)
	if err != nil {
		return fmt.Errorf("plugin '%s' is not installed — run 'aide plugin install %s' first", name, name)
	}

	fmt.Printf("\nReconfiguring %s...\n\n", name)

	sourceCfg, err := prompt.ConfigurePlugin(m, src.Config)
	if err != nil {
		return err
	}
	src.Config = sourceCfg
	cfg.Sources[name] = src

	if len(m.Credentials) > 0 {
		var update bool
		_ = survey.AskOne(&survey.Confirm{
			Message: "Update stored credentials?",
			Default: false,
		}, &update)
		if update {
			if err := prompt.SetupPluginCredentials(m, name); err != nil {
				return fmt.Errorf("credential setup failed, changes not saved: %w", err)
			}
		}
	}

	if err := cfg.Save(cfgFile); err != nil {
		return err
	}

	fmt.Printf("\n✓ Source '%s' updated.\n", name)
	fmt.Printf("  Run 'aide run --source %s' to test.\n", name)
	return nil
}
