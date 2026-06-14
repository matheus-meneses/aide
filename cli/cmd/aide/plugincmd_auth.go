package main

import (
	"aide/cli/internal/plugin"
	"fmt"

	"github.com/spf13/cobra"
)

func pluginAuthExecute(cmd *cobra.Command, args []string) error {
	sourceName := args[0]
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	src, ok := cfg.Sources[sourceName]
	if !ok {
		return fmt.Errorf("source %q not found in config", sourceName)
	}
	pluginName := src.Plugin
	if pluginName == "" {
		pluginName = sourceName
	}
	mgr := plugin.NewManager()
	m, err := mgr.Get(pluginName)
	if err != nil {
		return fmt.Errorf("loading plugin %q: %w", pluginName, err)
	}
	if !m.Capabilities.Browser {
		return fmt.Errorf("plugin %q does not use a browser — no auth flow needed", pluginName)
	}
	secrets, err := plugin.ScopedSecrets(sourceName, m)
	if err != nil {
		return fmt.Errorf("loading secrets: %w", err)
	}
	req := &plugin.Request{
		Action:  "scrape",
		Config:  src.Config,
		Secrets: secrets,
	}
	fmt.Printf("Opening browser for %s authentication...\n", sourceName)
	fmt.Println("Complete the login in the browser window, then return here.")
	_, stderr, err := plugin.ExecuteInteractive(cmd.Context(), m, req)
	if stderr != "" {
		fmt.Print(stderr)
	}
	if err != nil {
		return fmt.Errorf("auth failed: %w", err)
	}
	fmt.Printf("Authentication for %s saved successfully.\n", sourceName)
	return nil
}
