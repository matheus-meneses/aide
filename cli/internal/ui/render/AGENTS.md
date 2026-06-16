# render

## Purpose

Terminal output formatting for CLI reports: run summaries, item reports, diffs, source health tables, and run history.

## Exported API

All exported symbols are functions (no exported types):
- `SetOutput(w io.Writer)` — redirect output (default: os.Stdout)
- `PrintRunSummary(*runner.RunResult)` — post-scrape table
- `PrintReport(*store.Store, member, category)` — open items grouped by source
- `PrintDiff(*store.Store, source)` — 24h new vs resolved items
- `PrintSources(*config.Config, *store.Store)` — source health overview
- `PrintHistory(*store.Store)` — recent run history with duration

## Important Invariants

- Duration rendering uses proper `time.Parse` of RFC3339 timestamps (not string manipulation).
- Source-specific formatting uses an internal `sourcePlugin` interface with plugins for `gitlab`, `sailpoint`, `outlook` registered at `init()`.
- All output goes through package-level `fprintf`/`fprintln` helpers that target the configured writer.
- `tabwriter` is used for aligned columns.

## Pitfalls

- `sourcePlugin` is unexported — new source display logic requires adding a new file in this package.
- ANSI hyperlinks (`\033]8;;url\a`) may not render in all terminals.

## Relations

- Depends on: `config`, `runner`, `store`
- Used by: `cmd/aide` (report, diff, stats, sources, history commands)
