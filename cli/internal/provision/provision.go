// Package provision holds the headless configuration logic shared by the CLI
// commands and the agent's HTTP API. Functions here never assume a TTY and
// never print to stdout; they read and write config.yaml, the OS keychain, and
// the installed plugin set so both surfaces behave identically.
//
// The package is organized by domain:
//   - provision_sources.go:    sources and their credentials
//   - provision_llm.go:        agent LLM/provider settings
//   - provision_plugins.go:    plugin catalog, install, manifest
//   - provision_team.go:       team roster
//   - provision_registries.go: plugin registries
//   - provision_settings.go:   general settings, schedule, config snapshot
package provision
