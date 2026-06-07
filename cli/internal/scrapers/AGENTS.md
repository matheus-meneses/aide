# scrapers

## Purpose

Embeds the Python scraper source tree at build time and provides utilities to extract it to disk and list available scrapers.

## Exported API

- `FS embed.FS` — the full embedded filesystem (`//go:embed all:embedded`)
- `ListAvailable() []string` — returns scraper names (`.py` files under `embedded/sources/`, excluding `_` prefixed)
- `ExtractTo(dir string) error` — walks `embedded/` and writes all files/dirs to `dir`

## Important Invariants

- The `embedded/` directory must exist at build time with the full scraper tree (sources, framework, requirements.txt, registry.yaml fallback).
- `ExtractTo` overwrites existing files — it's used during `aide init` to ensure scrapers are current.
- `ListAvailable` strips the `.py` extension to return clean source names.
- File permissions: directories get 0o755, files get 0o644.

## Pitfalls

- Embedding happens at compile time — scraper changes require rebuilding the Go binary.
- Large embedded trees increase binary size significantly.
- `FS.ReadFile("embedded/registry.yaml")` is used as fallback by `initcmd` when Nexus download fails.

## Relations

- Used by: `cmd/aide` (init command for extraction), `initcmd` (embedded registry fallback)
- No internal dependencies (leaf package).
