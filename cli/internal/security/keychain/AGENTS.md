# keychain

## Purpose

Secure per-source credential storage with per-OS backends. Stores credential maps as JSON, keyed
by a service name derived from `$AIDE_HOME` so sandbox and production environments don't collide.

## Exported API

- `SetField(source, key, value string) error` — upsert one credential field (read-modify-write)
- `GetAll(source string) (*Credential, error)` — retrieve all fields for a source
- `DeleteSource(source string) error` — remove all credentials for a source
- `DeleteField(source, key string) error` — remove one field (deletes whole entry if empty)
- `List() ([]string, error)` — list all stored source names
- `ServicePrefix() string` — returns the prefix used for storage (derived from `AIDE_HOME`)

## Key Types

- `Credential` — `Fields map[string]string`

## Service-name prefix

The service prefix is **not** a hardcoded `aide/`; it is derived dynamically:

```go
func ServicePrefix() string {
    base := filepath.Base(xdg.AideHome())
    name := strings.TrimLeft(base, ".")
    if name == "" { name = "aide" }
    return name + "/"
}
```

For a default install (`~/.aide`): prefix = `aide/` → service = `aide/<source>`.
For a sandbox (`~/.aide-sandbox`): prefix = `aide-sandbox/` → service = `aide-sandbox/<source>`.

This ensures sandbox and production credentials never collide in the OS keychain.

## Per-OS backends

| OS      | Build tag   | Backend                                          |
|---------|-------------|--------------------------------------------------|
| macOS   | `darwin`    | `/usr/bin/security` CLI (macOS Keychain)         |
| Linux   | `linux`     | Flat JSON file at `$AIDE_HOME/credentials.json`  |
| Windows | `windows`   | Flat JSON file at `$AIDE_HOME\credentials.json`  |

The stale "macOS-only, fails on Linux/Windows" claim is incorrect. All three platforms have
implementations. Linux and Windows use a JSON file rather than a system keychain.

## Important Invariants

- `SetField` always reads the existing credential first, then merges and writes back.
- `DeleteField` removes the whole entry (calls `DeleteSource`) when the last field is removed,
  rather than leaving an entry with an empty `{}` payload.
- Account name for macOS Keychain entries: `"aide"` (constant).
- Concurrent `SetField` calls for the same source can race (no mutex); callers should serialize.

## Relations

- Depends on: `xdg` (for `AideHome()`)
- Used by: `plugin` (`ScopedSecrets`), `prompt` (credential setup wizard), `cmd/aide`
  (credential subcommands)
