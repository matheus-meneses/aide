# config

## Purpose

Loads, validates, and provides typed access to the YAML configuration file (`~/.aide/config.yaml`). Supplies sensible defaults for all settings.

## Key Types

- `Config` — Root config struct containing `Settings`, `Team`, `Sources`, and `Agent`.
- `Settings` — Concurrency, timeout, paths to data/scrapers/python, and global `TLS`.
- `Source` — Per-source config with `Enabled` flag, opaque `Config map[string]any`, free-text `Context` (guidance injected into LLM prompts), and optional `TLS` override.
- `TLS` — `VerifySSL *bool` (nil = unset, so defaults/overrides compose) and `CABundle` PEM path. Set globally under `settings.tls` and/or per source under `sources.<name>.tls`. The `runner` resolves the effective value (flag > per-source > global > secure default).
- `AgentConfig` — Autonomous agent settings (run interval, briefing times, LLM config, and `UserContext` free-text shaping the assistant).
- `TeamMember` — Name + aliases for member resolution across sources.

## Important Invariants

- `Load()` always applies defaults before returning. `LoadRaw()` does not.
- `Concurrency` must be >= 1, `TimeoutSeconds` must be >= 1 (enforced by `Load`).
- `EnabledSources()` filters the sources map to only those with `Enabled: true`.
- `RunIntervalDuration()` falls back to 30m on parse failure, never panics.
- `Save()` writes back to disk (used by `source add` command).

## Pitfalls

- `Source.Config` is `map[string]any` — values come from YAML unmarshaling; numeric types may be `int` or `float64` depending on the YAML content.
- `ResolvePaths` mutates the receiver in place (expands `~`/relative for `data_dir` and every `ca_bundle`).
- Never assume config paths are absolute until after `Load` (which calls `ResolvePaths`).

## Relations

- Used by: `runner`, `render`, `prompt`, `cmd/aide`
- No internal dependencies (leaf package).
