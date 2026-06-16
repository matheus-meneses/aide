// Package app is the composition root for the agent runtime. It wires the
// long-lived dependency stack (store -> runner -> agent) so the CLI
// (`aide agent start`) and the desktop binary (`aide-app`) construct the agent
// identically instead of duplicating the chain.
package app

import (
	"aide/cli/internal/agent"
	"aide/cli/internal/persistence/store"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/runner"
	"fmt"
)

// Stack bundles the agent runtime and its owned resources. Close releases the
// store; call it once the agent is no longer needed.
type Stack struct {
	Agent  *agent.Agent
	Store  *store.Store
	Runner *runner.Runner
}

// Close releases resources owned by the stack.
func (s *Stack) Close() error {
	if s == nil || s.Store == nil {
		return nil
	}
	return s.Store.Close()
}

// New builds the agent runtime from cfg. level and format are the resolved
// logging settings (callers compute them from flags or config) and version is
// the build string stamped onto the agent. On error no resources leak.
func New(cfg *config.Config, level, format, version string) (*Stack, error) {
	s, err := store.Open(cfg.Settings.DataDir)
	if err != nil {
		return nil, fmt.Errorf("opening store: %w", err)
	}

	r := runner.New(cfg, s)
	r.SetLogLevel(level)
	r.SetLogFormat(format)

	a, err := agent.New(cfg, s, r)
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("creating agent: %w", err)
	}
	agent.Version = version

	return &Stack{Agent: a, Store: s, Runner: r}, nil
}
