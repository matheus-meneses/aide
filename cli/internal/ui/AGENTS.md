# ui

## Purpose

Everything presentation: the embedded web UI/HTTP server, terminal rendering, interactive prompts,
and small terminal widgets. `ui` sits near the top of the dependency DAG — things depend on it only
from `cmd`.

## Packages

- `webui` — HTTP server + embedded Vite/React frontend (`//go:embed frontend/dist`), `/api/open`,
  `/api/logs`. Mounts agent routes through `webui.Options.RegisterAPI` so it never imports `agent`.
  - `frontend/` — Vite/React/TypeScript app (built to `frontend/dist`).
- `render` — terminal and structured output rendering for CLI commands.
- `prompt` — interactive terminal prompts (select, confirm) built on survey/bubbletea.
- `widgets` — reusable terminal widgets (spinner, tables); formerly `internal/ui`.

## Dependency rules (depguard)

- **May import:** the Go standard library, third-party libs, `platform/*`, `persistence/*`,
  `security/*`, `runtime/*`.
- **Must NOT import:** `agent`, `setup`, `notification`. Agent functionality is wired in by `cmd`
  via the `RegisterAPI` registrar (`agentapi.Register`), keeping `ui` and `agent` decoupled.
