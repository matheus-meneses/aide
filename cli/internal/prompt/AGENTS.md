# prompt

## Purpose

Interactive terminal UI for guided source configuration. Wraps the `survey` library to present selection menus, text inputs with defaults, and password prompts driven by registry metadata.

## Exported API

- `PickSource(reg *registry.Registry, configured map[string]config.Source) (string, error)` — arrow-key selection of unconfigured sources
- `ConfigureSource(def *registry.SourceDef) (map[string]any, error)` — guided field prompts with defaults, type-aware (json, list, confirm for optional)
- `SetupCredentials(def *registry.SourceDef, sourceName string) error` — credential prompts stored to keychain

## Important Invariants

- `PickSource` filters out already-configured sources — returns error if all sources are configured.
- Optional fields show a confirm prompt before asking for value.
- `type: "json"` fields are validated as valid JSON before accepting.
- `type: "list"` fields are split by comma into `[]string` for YAML storage.
- Secret credentials use `survey.Password` (masked input).

## Pitfalls

- Depends on terminal being interactive (TTY) — will fail in CI/pipe contexts.
- `survey` package is in maintenance mode; future replacement with `charmbracelet/huh` is planned.
- If registry has no unconfigured sources, the error message must be user-friendly (handled in cmd layer).

## Relations

- Depends on: `registry`, `keychain`, `config`
- Used by: `cmd/aide` (source add command)
