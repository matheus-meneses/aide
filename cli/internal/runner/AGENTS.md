# runner

## Purpose

Orchestrates concurrent execution of Python scrapers, collects JSON output, upserts results into the store, and records health/metrics.

## Key Types

- `Runner` — Holds config, store, and a log writer.
- `RunResult` — Aggregate outcome: run ID, source counts, per-source results.
- `SourceResult` — Per-source execution result (entries, errors, timing, new items).
- `ScraperOutput` — JSON schema expected from Python scraper stdout.

## Important Invariants

- Scrapers are invoked as `python -m framework.runner <source> --config <json>` inside `cfg.Settings.ScrapersDir`.
- Concurrency is limited by a semaphore sized to `cfg.Settings.Concurrency`.
- Each source runs with `context.WithTimeout` and process-group kill on cancel.
- `ValidateFilter()` must be called before `Run()` if the user supplies `--source` flags — unknown or disabled names are rejected.
- Credentials are injected as environment variables (`PREFIX_KEY=value`) when source config specifies `credentials_env`.

## Protocol (Python → Go)

Scraper stdout must be a JSON array of `ScraperOutput`:
```json
[{"title": "...", "member": "...", "category": "...", "metadata": {"web_url": "...", "mode": "metric|item", ...}}]
```
- `mode: "metric"` → `store.RecordMetric` (not upserted as item)
- Otherwise → `store.UpsertItems`

stderr is streamed to the runner's log writer.

## Pitfalls

- `UpdateRun` error is now logged but not propagated (non-fatal).
- If a scraper returns zero entries, no items are resolved (empty-scrape guard in store).
- Process-group kill uses `syscall.Kill(-pid, SIGKILL)` — macOS/Linux only.
- Timeout includes process startup time; slow Python imports count against it.

## Relations

- Depends on: `config`, `store`, `keychain`
- Used by: `agent` (tool), `cmd/aide` (run command)
