# runner

## Purpose

Orchestrates concurrent execution of plugin scrapers, collects JSON responses, normalises results,
and upserts entries into the store.

## Key Types

- `Runner` — holds config, store, and a log writer (defaults to `os.Stderr`).
- `RunResult` — aggregate outcome: run ID, per-source counts, slice of `SourceResult`.
- `SourceResult` — per-source result: entries, team members, plugin response, timing, stderr.

## How it works

1. `Run(ctx, filterSources)` resolves enabled sources from config (filtered by `filterSources` if
   non-empty).
2. A run record is inserted into `store.Runs` before dispatch.
3. Each source is launched in a goroutine, throttled by a semaphore sized to
   `cfg.Settings.Concurrency`.
4. Per source, `executeSource` calls `plugin.NewManager().Get(pluginName)` to resolve the plugin,
   loads scoped secrets via `plugin.ScopedSecrets()`, then calls `plugin.Execute(ctx, m, req)` with
   `req.Action = "scrape"`.
5. The plugin sends a single JSON object (protocol v1) on stdout; `plugin.Execute` parses the
   `plugin.Response` struct and returns it.
6. Entries, metrics, and team members are extracted from `resp.Entries`, `resp.Metrics`,
   `resp.TeamMembers`, then upserted into the store.
7. `store.Runs.Update` records the final counts; errors there are logged, not propagated.

## Plugin protocol (Python → Go)

The runner sends a JSON object on stdin via `plugin.Execute`:

```json
{
  "action": "scrape",
  "config": { ... },
  "secrets": { ... },
  "context": {
    "data_dir": "/path/to/data",
    "log_level": "info",
    "log_format": "text",
    "verify_ssl": true
  }
}
```

`context` keys:

| Key | Values | Default | Meaning |
|-----|--------|---------|---------|
| `data_dir` | path string | from config | plugin working directory |
| `log_level` | `debug` \| `info` \| `warn` \| `error` | `info` | logging threshold; set to `debug` by `-v` |
| `log_format` | `text` \| `json` | `text` | log line format; set by `--log-format` |
| `verify_ssl` | `true` \| `false` | `true` | verify TLS certificates for plugin network requests; set by `--verify-ssl` |

The plugin replies with **one JSON object** on stdout:

```json
{
  "protocol_version": "1",
  "ok": true,
  "entries": [{"member":"...","category":"...","title":"...","priority":"info",...}],
  "team_members": [...],
  "metrics": [{"name":"...","value":0.0}]
}
```

or `{"ok":false,"error":"..."}` on failure.

This is the **current** protocol. The legacy `python -m framework.runner <source> --config <json>`
invocation and JSON-array stdout format are obsolete — they do not exist in the current codebase.

## Important Invariants

- `ValidateFilter()` must be called before `Run()` when the caller supplies source names;
  unknown or disabled sources return an error.
- Each source runs with `context.WithTimeout` derived from `cfg.Settings.TimeoutSeconds`; the
  plugin process is killed on context cancellation by `plugin.Execute`.
- Credentials are loaded by `plugin.ScopedSecrets(name, m)` and passed in `Request.Secrets` on
  stdin. They are never injected as environment variables.
- If a plugin returns `ok=false`, the source is counted as failed.
- Plugin stderr is streamed to the runner's log writer prefixed with `[sourceName]`.

## Relations

- Depends on: `config`, `store`, `plugin`, `keychain` (indirectly via `plugin.ScopedSecrets`)
- Used by: `internal/agent` (scrape tool), `cmd/aide` (run command)
