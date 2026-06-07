# registry

## Purpose

Loads and provides typed access to the source marketplace catalog (`~/.aide/registry.yaml`). Describes available scrapers, their config fields, required credentials, and display metadata.

## Key Types

- `Registry` — Root: `Sources map[string]SourceDef`
- `SourceDef` — Description, categories, fields, credentials for one scraper
- `Field` — Config field metadata: key, label, type, required flag, default value, hint
- `Credential` — Credential field metadata: key, label, secret flag, hint

## Exported API

- `Load() *Registry` — loads from `~/.aide/registry.yaml`; returns empty registry on error
- `LoadFrom(path string) *Registry` — explicit path
- `(Registry) GetSource(name string) *SourceDef`
- `(Registry) ListSources() []string` — sorted names
- `(Registry) Options() []string` — `"name - description"` for interactive prompts

## Important Invariants

- `Load()` never returns nil — always at least an empty `Registry{}`.
- `Field.Type` can be: `"string"`, `"url"`, `"number"`, `"list"`, `"json"`.
- `Credential.Secret` determines whether prompt masks input.
- The file is versioned and deployed to Nexus alongside binaries; `aide init` downloads it.

## Pitfalls

- `registry.Credential` (metadata schema) is different from `keychain.Credential` (stored secrets).
- If the registry file is missing or malformed, all operations proceed with no sources available — no error surfaced to the user.
- Adding a new source requires updating both the registry YAML and adding a Python scraper.

## Relations

- Used by: `prompt` (interactive source setup), `cmd/aide` (config check, source add/list)
- No internal dependencies (leaf package).
