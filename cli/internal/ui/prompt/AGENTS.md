# prompt

## Purpose

Interactive terminal UI for guided source configuration. Selection menus ("terminal iteration") are built on `charmbracelet/bubbletea`; text/confirm/password inputs still use `survey` pending migration to `charmbracelet/huh`.

## Exported API

- `Select(header string, choices []Choice) (int, error)` — bubbletea arrow-key menu. Returns the chosen index or `ErrCancelled`. Requires a TTY; callers must guard non-interactive contexts.
- `Choice{Title, Desc, Tag}` — one menu row (`Tag` renders as a badge, e.g. `[installed]`).
- `ErrCancelled` — returned by `Select` when the user aborts (esc / q / ctrl+c).
- `PickPlugin(mgr *plugin.Manager, configured map[string]config.Source) (string, error)` — selects an unconfigured installed plugin via `Select`.
- `ConfigurePlugin(m *plugin.Manifest) (map[string]any, error)` — guided field prompts with defaults (survey).
- `SetupPluginCredentials(m *plugin.Manifest, sourceName string) error` — credential prompts stored to keychain (survey, masked for secrets).

## Important Invariants

- `PickPlugin` filters out already-configured plugins — returns an error if all are configured.
- `Select` navigation: ↑/↓ or k/j, home/g and end/G to jump, enter to choose, esc/q/ctrl+c to cancel. Long lists scroll within a fixed window with "more above/below" markers.
- Optional fields show a confirm prompt before asking for a value.
- Secret credentials use masked input.

## Pitfalls

- All interactive functions require a TTY — they fail in CI/pipe contexts. The `aide plugin install` picker guards this with an `isInteractive()` check and a "pass a name" hint.
- `survey` is in maintenance mode; the remaining input/confirm/password prompts should move to `charmbracelet/huh` (bubbletea-based) to retire the dependency.

## Relations

- Depends on: `plugin`, `config`, `keychain`, `bubbletea`, `lipgloss`
- Used by: `cmd/aide` (source add, plugin install picker)
