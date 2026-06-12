package main

import (
	"aide/cli/internal/config"
	"aide/cli/internal/store"
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

var stdinReader = bufio.NewReader(os.Stdin)

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("no config found at %s — run 'aide init' to set up", cfgFile)
		}
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return cfg, nil
}

func withStore(fn func(cfg *config.Config, s *store.Store) error) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	s, err := store.Open(cfg.Settings.DataDir)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer s.Close()

	return fn(cfg, s)
}
