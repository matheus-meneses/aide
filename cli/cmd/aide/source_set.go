package main

import (
	"aide/cli/internal/config"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func sourceEnableExecute(_ *cobra.Command, args []string) error {
	return toggleSource(args[0], true)
}

func sourceDisableExecute(_ *cobra.Command, args []string) error {
	return toggleSource(args[0], false)
}

func toggleSource(name string, enabled bool) error {
	cfg, err := config.LoadRaw(cfgFile)
	if err != nil {
		return err
	}

	src, exists := cfg.Sources[name]
	if !exists {
		return fmt.Errorf("source '%s' not found", name)
	}

	src.Enabled = enabled
	cfg.Sources[name] = src

	if err := cfg.Save(cfgFile); err != nil {
		return err
	}

	state := "disabled"
	if enabled {
		state = "enabled"
	}
	fmt.Printf("Source '%s' %s.\n", name, state)
	return nil
}

func sourceSetExecute(_ *cobra.Command, args []string) error {
	name, key, value := args[0], args[1], args[2]

	cfg, err := config.LoadRaw(cfgFile)
	if err != nil {
		return err
	}

	src, exists := cfg.Sources[name]
	if !exists {
		return fmt.Errorf("source '%s' not found", name)
	}

	if src.Config == nil {
		src.Config = make(map[string]any)
	}

	var parsed any
	if json.Unmarshal([]byte(value), &parsed) == nil {
		src.Config[key] = parsed
	} else {
		src.Config[key] = value
	}

	cfg.Sources[name] = src

	if err := cfg.Save(cfgFile); err != nil {
		return err
	}

	fmt.Printf("Source '%s': %s = %v\n", name, key, src.Config[key])
	return nil
}
