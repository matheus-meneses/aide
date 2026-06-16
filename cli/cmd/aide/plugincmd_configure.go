package main

import (
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/plugin"
	"aide/cli/internal/setup/provision"
	"aide/cli/internal/ui/prompt"
	"aide/cli/internal/ui/widgets"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

func pluginConfigureExecute(_ *cobra.Command, args []string) error {
	cfg, err := loadRawConfig()
	if err != nil {
		return err
	}
	if cfg.Sources == nil {
		cfg.Sources = make(map[string]config.Source)
	}

	mgr := plugin.NewManager()

	name, err := resolveConfigureName(mgr, cfg.Sources, args)
	if err != nil {
		return err
	}

	if _, exists := cfg.Sources[name]; exists {
		return reconfigureSource(mgr, cfg, name)
	}
	return addSource(mgr, name)
}

func resolveConfigureName(mgr *plugin.Manager, sources map[string]config.Source, args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	return prompt.PickPlugin(mgr, sources)
}

func addSource(mgr *plugin.Manager, name string) error {
	m, err := mgr.Get(name)
	if err != nil {
		return fmt.Errorf("plugin '%s' is not installed — run 'aide plugin install %s' first", name, name)
	}

	widgets.Printf("\nSetting up %s...\n\n", name)

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

	widgets.Printf("\n✓ Source '%s' enabled.\n", name)
	widgets.Printf("  Run 'aide run --source %s' to test.\n", name)
	return nil
}

func reconfigureSource(mgr *plugin.Manager, cfg *config.Config, name string) error {
	src := cfg.Sources[name]

	m, err := mgr.Get(name)
	if err != nil {
		return fmt.Errorf("plugin '%s' is not installed — run 'aide plugin install %s' first", name, name)
	}

	widgets.Printf("\nReconfiguring %s...\n\n", name)

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

	widgets.Printf("\n✓ Source '%s' updated.\n", name)
	widgets.Printf("  Run 'aide run --source %s' to test.\n", name)
	return nil
}
