package main

import (
	"aide/cli/internal/persistence/store"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/ui/widgets"
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

var stdinReader = bufio.NewReader(os.Stdin)

// errCanceled signals that the user declined a prompt; main() maps it to a
// non-zero exit without printing a scary error line.
var errCanceled = errors.New("canceled")

func stdinIsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// confirm asks a yes/no question. It returns true immediately when --yes is
// set, and refuses (returns false) when stdin is not a terminal and --yes was
// not given, so destructive actions never silently proceed or hang in scripts.
func confirm(prompt string) bool {
	if assumeYes {
		return true
	}
	if !stdinIsTerminal() {
		widgets.PrintError("%q needs confirmation; re-run with --yes for non-interactive use", prompt)
		return false
	}
	widgets.Printf("%s [y/N]: ", prompt)
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

// requireConfirm is like confirm but returns errCanceled when the user
// declines, so callers can propagate a non-zero exit code.
func requireConfirm(prompt string) error {
	if confirm(prompt) {
		return nil
	}
	return errCanceled
}

func loadConfig() (*config.Config, error) {
	return wrapConfigLoad(config.Load(cfgFile))
}

// loadRawConfig reads the config without injecting defaults or resolving paths
// (for edit-then-Save round trips) while sharing loadConfig's friendly
// not-found error message.
func loadRawConfig() (*config.Config, error) {
	return wrapConfigLoad(config.LoadRaw(cfgFile))
}

func wrapConfigLoad(cfg *config.Config, err error) (*config.Config, error) {
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
