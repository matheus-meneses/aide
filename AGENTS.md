# AGENTS.md — aide

## Identity

You are working on **aide**: a personal work-assistant CLI written in Go, with a Python plugin
subsystem and an embedded React UI. The codebase is polyglot; each language has its own guardrails.

Rules that apply regardless of language:

- Write idiomatic code for the language at hand (Go interfaces, Python dataclasses/pydantic, TS
  functional components with hooks).
- Errors are explicit: Go wraps with `fmt.Errorf("context: %w", err)`, Python raises typed
  exceptions, TypeScript propagates typed `Error` objects.
- No narrating comments. Only write comments that explain *why*, never *what*. Do not say
  "increment counter", "return result", or anything the code already expresses.
- Propagate `context.Context` in Go wherever it is available; never swallow cancellations.
- Challenge questionable requirements instead of blindly complying. If a decision seems wrong,
  say so and propose the correct approach.
- `//nolint`, `# noqa`, and `eslint-disable` are last resorts. Fix the root cause first. If a
  suppression is unavoidable, add a brief explanation alongside it.

---

## Repo map

```
aide/
├── Makefile              polyglot verify gate (make verify)
├── .editorconfig         cross-editor formatting baseline
├── .pre-commit-config.yaml
├── cli/                  Go core — single static binary
│   ├── cmd/aide/         Cobra entry-point commands (thin shells, logic in internal/)
│   ├── cmd/aide-app/     desktop shell (webview) wrapping the same agent + webui
│   └── internal/
│       ├── agent/        autonomous brain: scheduler, tools, chat, SSE event bus
│       │   └── llm/      provider-agnostic LLM clients (OpenAI/LiteLLM, Anthropic)
│       ├── webui/        HTTP server + embedded Vite/React UI + /api/open + /api/logs
│       │   └── frontend/ Vite/React UI (built → embedded via //go:embed)
│       ├── clog/         scoped logging sink (stderr/file + live log subscribers)
│       ├── config/       AIDE_HOME-rooted config loading
│       ├── keychain/     per-OS credential storage (macOS, Linux, Windows)
│       ├── plugin/       plugin lifecycle: resolve, install, execute, sandbox
│       ├── procctl/      process-group management for subprocess cleanup
│       ├── prompt/       LLM prompt assembly and context injection
│       ├── render/       terminal and structured output rendering
│       ├── runner/       parallel scrape scheduling and result normalisation
│       ├── store/        SQLite persistence for items, metrics, team, sessions
│       ├── updater/      self-update from GitHub releases
│       └── xdg/          platform-specific data/config/cache paths
├── sdk/
│   ├── python/           aide-sdk Python package (BaseScraper, models, runtime)
│   └── go/               aide-sdk-go: plugin.Serve + Handler for Go-runtime plugins
└── bin/                  compiled binary output
```

The `aide-plugins` repo is a sibling directory (`../aide-plugins/`). Plugins are **not** part of
this repo.

---

## The three runtimes

### 1. Go CLI

A single static binary built with `go build ./cmd/aide`. Cobra commands in `cmd/aide/` are thin
wrappers; all business logic lives in `internal/`. Key conventions:

- Error propagation: `fmt.Errorf("pkg/function: %w", err)`. Compare with `errors.Is`/`errors.As`.
- Logging: `log` (stdlib) is used in `internal/agent`; `log/slog` is the target for **new**
  structured logging elsewhere. Do not mix them in the same package.
- Config keys and plugin/source names use `snake_case`.
- OS-specific files use build-tag suffixes: `sandbox_darwin.go`, `keychain_linux.go`, etc.

### Verbose logging flags

Two global flags are available on every command:

| Flag | Short | Default | Effect |
|------|-------|---------|--------|
| `--verbose` | `-v` | off | Sets log level to `debug`. Without this flag, level is `info`. |
| `--log-format` | — | `text` | `text` (human-readable) or `json` (one JSON object per line). |

Example:
```
aide -v run --source jira            # debug-level text output to stderr
aide -v --log-format json run        # debug-level JSON lines to stderr
aide run --source jira               # info-level only (default)
```

The runner passes `log_level` and `log_format` to every plugin via `Request.Context`. Plugins
read these values automatically through the SDK (`self.log` in Python, `plugin.Log` in Go).

#### Canonical log line format

**text** (default):
```
<RFC3339> [<level>] <scope>: <message>
```
Example: `2026-06-09T21:30:00Z [debug] jira: Connecting to Jira...`

**json**:
```json
{"ts":"2026-06-09T21:30:00Z","level":"debug","scope":"jira","msg":"Connecting to Jira..."}
```
Keys always in order: `ts`, `level`, `scope`, `msg`. `scope` is omitted when empty.

Level ordering: `debug=10`, `info=20`, `warn=30`, `error=40`. Only messages at or above the
configured threshold are emitted. All output goes to **stderr**; stdout remains reserved for the
plugin protocol JSON.

### 2. Python plugin subprocess

Each plugin runs in its own `.venv`, invoked by `cli/internal/plugin` via stdin/stdout JSON.

Protocol (one round-trip per invocation):
- CLI sends `{"action": "<action>", "config": {...}, "secrets": {...}, ...}` on **stdin**.
- Plugin responds with a single JSON object on **stdout**: `{"protocol_version": "1", "ok": true,
  "entries": [...], "team_members": [...], "metrics": [...]}` or `{"ok": false, "error": "..."}`.

**STDOUT IS RESERVED for the protocol.** The `aide_sdk` runtime redirects `sys.stdout` to stderr
at startup. All plugin logging must go to stderr.

Actions: `describe`, `scrape`, `render`, `query`.

A plugin may also set `runtime: go` instead of `python`. The host then runs a compiled binary at
`<plugin_dir>/bin/<entrypoint.go.binary>` (see `runtime_go.go`) over the identical JSON protocol —
no venv is built. Go plugins use the `sdk/go` package (`plugin.Serve(Handler)`, `plugin.Log`) and
publish one artifact per platform in the registry under `go/<goos>_<goarch>` keys.

### 3. Embedded React UI

`cli/internal/webui/frontend/` is a Vite/React app with TypeScript strict mode.

- Built by `npm run build` → `frontend/dist/`.
- Embedded into the Go binary via `//go:embed frontend/dist` in `internal/webui`.
- Served by the `internal/webui` HTTP server at the root path; `internal/agent` domain routes are mounted via the
  `webui.Options.RegisterAPI` registrar (so `webui` and `agent` stay decoupled).
- Communicates with the backend over SSE (`/api/events`, `/api/logs`) and fetch (`/api/chat`, `/api/items`, `/api/open`, …).

---

## Verify gate

Before committing or opening a PR, run:

```
make fmt     # gofumpt + ruff format + prettier
make verify  # go-lint, go-test, py-lint, py-type, fe-lint
```

Sub-targets:

| Target    | Tool                  | Scope                            |
|-----------|-----------------------|----------------------------------|
| `go-lint` | golangci-lint v2      | `cli/` (config: `.golangci.yml`) |
| `go-test` | `go test -race ./...` | `cli/`                           |
| `py-lint` | ruff                  | `sdk/python/`                    |
| `py-type` | mypy                  | `sdk/python/aide_sdk/`           |
| `fe-lint` | tsc --noEmit + eslint | `cli/internal/webui/frontend/`   |

Optional (not in default verify): `make go-vuln` (govulncheck), `make py-test`.

---

## Commit convention

```
<type>(<scope>): <description>
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`, `build`, `ci`.

Scopes: `agent`, `config`, `keychain`, `plugin`, `runner`, `store`, `updater`, `xdg`, `prompt`,
`render`, `procctl`, `frontend`, `sdk`, `cli`.

Body: wrap at 72 chars. Reference issues/PRs with `Closes #N` or `Refs #N`.

---

## Testing

- Go: stdlib `testing` + `-race` flag. Prefer table-driven tests. Mock at interface boundaries.
- Python: `pytest` under `sdk/python/tests/`. Validate pydantic models with edge cases.
- React: (not yet wired) — add `vitest` tests alongside components as the UI grows.
- Plugins: `aide dev test <path>` runs a plugin's `scrape` action in place without installing it
  (builds the venv / Go binary on demand). Use `--json` for a machine-readable
  `{ok, entries, ..., logs, exit_code}` result and `-v` for debug logs. `aide dev validate <path>`
  checks the manifest. These are flag-driven and `--json`-capable for autonomous agent loops, and
  replace the old `aide scrape` flow.

---

## What lives where

| Question                            | Answer                                                  |
|-------------------------------------|---------------------------------------------------------|
| Where is a plugin executed?         | `cli/internal/plugin/plugin.go` → `Execute()`           |
| Where does the sandbox policy live? | `cli/internal/plugin/sandbox_*.go`                      |
| Where are scrape results stored?    | `cli/internal/store/` (SQLite via `store.Store`)        |
| Where is the LLM prompt assembled?  | `cli/internal/prompt/`                                  |
| Where are credentials stored?       | OS keychain via `cli/internal/keychain/`                |
| How does the runner schedule work?  | `cli/internal/runner/runner.go` — semaphore concurrency |
