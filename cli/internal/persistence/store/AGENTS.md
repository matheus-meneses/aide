# store

## Purpose

SQLite persistence layer for all application data: scraped items, run history, source health, metrics, chat sessions, agent memory, user profile, acknowledged alerts, and LLM token usage.

## Key Types

- `Store` — Wraps `*sql.DB` with `SetMaxOpenConns(1)` for SQLite safety.
- `Item` — Core work item with fingerprint-based identity, status lifecycle (`open` → `resolved`).
- `Run` — Aggregate record of a scrape execution (timestamps, OK/failed counts).
- `SourceHealth` — Latest health snapshot per source.
- `ChatSession` / `ChatMessage` — Persistent chat history for the web UI.
- `AgentMemory` — JSON snapshot of agent's working memory.
- `TokenSummary` — Aggregated LLM token usage stats.
- `PruneResult` — Rows deleted per table during retention cleanup.

## Important Invariants

- Items are uniquely identified by `fingerprint` (SHA-256 of source+link or source+title+member).
- `UpsertItems` only resolves items when `len(items) > 0` — an empty scrape result MUST NOT mass-resolve existing items.
- Migrations run sequentially on `Open()` and are versioned (table `schema_version`). Never skip or reorder migrations.
- `SetMaxOpenConns(1)` — SQLite does not support concurrent writers; all writes serialize through this single connection.
- `Prune` respects event category items (never prunes future-dated events).

## Pitfalls

- All timestamp columns use RFC3339 strings, not Unix epochs. Parse with `time.Parse(time.RFC3339, ...)`.
- `QueryRow().Scan()` errors must be checked — previously some were swallowed.
- `Exec` errors in `Prune` are now propagated; callers should handle them.
- `Fingerprint()` is a free function, not a method — call it before building `Item` structs.

## Relations

- Used by: `runner`, `render`, `agent`, `cmd/aide`
- Depends on: `modernc.org/sqlite` (pure-Go SQLite driver)
