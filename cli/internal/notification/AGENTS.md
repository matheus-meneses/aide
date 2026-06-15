# notification

## Purpose

Delivers user-facing alerts through multiple channels: native OS notifications and the in-app event
bus consumed by the web UI.

## Files

- `notification.go` — `Notifier` interface, `MultiNotifier` (fan-out), `NoopNotifier` (default).
- `native.go` — `Native(title, body)` + `MacNotifier` (macOS `osascript`, Linux `notify-send`).
- `bus.go` — `BusNotifier`, publishing notification events onto the `agent/events` bus.

## Dependency rules (depguard)

- **May import:** the Go standard library, `platform/*`, and the `agent/events` leaf.
- **Must NOT import:** `persistence`, `security`, `runtime`, `setup`, `ui`, or the rest of `agent`
  (only `agent/events`). This keeps `agent` → `notification` → `agent/events` acyclic: the core
  `agent` package depends on `notification`, while `notification` reaches only the events leaf.
