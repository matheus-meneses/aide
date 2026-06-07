# keychain

## Purpose

Secure credential storage using macOS Keychain (`/usr/bin/security` CLI). Stores per-source credential maps as JSON generic passwords.

## Exported API

- `SetField(source, key, value string) error` — upsert a single credential field
- `GetAll(source string) (*Credential, error)` — retrieve all fields for a source
- `DeleteSource(source string) error` — remove entire credential
- `DeleteField(source, key string) error` — remove one field (loads, deletes, re-saves)
- `List() ([]string, error)` — list all stored source names

## Key Types

- `Credential` — `Fields map[string]string`

## Important Invariants

- Service name format: `aide/<source>` (e.g. `aide/jira`). Account: `aide`.
- Payload is JSON-encoded `map[string]string` stored as the password field.
- `SetField` loads existing credential, merges the field, then writes back (read-modify-write).
- `List` parses `security dump-keychain` output looking for `aide/` service prefixes.

## Pitfalls

- macOS-only — will fail on Linux/Windows (no fallback implementation).
- `security` CLI prompts for access permission on first use per app.
- Concurrent `SetField` calls for the same source can race (no locking).
- Empty `Fields` map after `DeleteField` leaves a keychain entry with `{}` payload.

## Relations

- Used by: `runner` (inject credentials as env vars), `prompt` (store credentials during setup), `cmd/aide` (credential commands)
- No internal dependencies (leaf package, macOS system dependency only).
