# agent

## Purpose

Autonomous AI agent that runs on a timer, uses LLM tool-calling to decide what actions to take (scrape, diff, notify),
and serves a web UI with real-time event streaming.

## Subpackages

`agent` is the concept domain root. It is split into cohesive subpackages so the core stays small:

- `agent` (this dir) — `Agent` orchestrator: config, store, runner, LLM client, notifier, event bus,
  tool registry; scheduler, chat, cycle, slash-command exec. Exposes accessors used by `agent/api`
  (`Bus()`, `StreamChat()`, `ExecuteCommand()`, `PublishProgress()`, `ConfigPath()`, `StoredAPIKey()`).
- `agent/events` — **leaf**: `Event`, `EventRing`, `EventBus` (in-memory SSE pub/sub). Imports `platform` only.
- `agent/llm` — provider-agnostic chat client (`Chat`, `ChatStream`, `ChatWithTools`, `Ping`, `Model`); OpenAI/LiteLLM + Anthropic.
- `agent/tools` — `Tool` / `ToolRegistry` + builtins, behind a `Capabilities` interface implemented by `*Agent`
  (avoids a core↔tools cycle).
- `agent/api` — HTTP REST handlers + `Register(a *agent.Agent, mux *http.ServeMux)`; mounted by `webui` via `RegisterAPI`.

Notifications moved out to the top-level `notification` concept (`MacNotifier`, `BusNotifier`, `MultiNotifier`,
`NoopNotifier`), which depends on the `agent/events` leaf.

## Architecture

```
Timer tick → runAgentCycle → build messages + tool catalog → LLM (native tool-calling)
                → execute returned tool calls → append tool results → loop until no tool calls / "done"
                                                                          ↓
                                                                   EventBus.Publish
                                                                          ↓
                                                                   ServeSSE → browser
```

The autonomous loop uses provider-native function-calling: `runAgentCycle` builds a running
`[]llm.ChatMessage` conversation (system + state) plus a tool catalog from `tools.Registry.Definitions()`,
then calls `llm.ChatWithTools` each turn, executes every returned tool call, and feeds results back as
`tool`-role messages until the model stops calling tools or calls `done`. If a model returns plain text
instead of tool calls, the loop falls back to parsing the legacy prompt-JSON (`{"tool","params"}`) once.

The wire protocol is each vendor's native function-calling spec (not a homegrown one); see the
protocol references in [agent/llm/AGENTS.md](llm/AGENTS.md) for the OpenAI, Anthropic, and JSON Schema links.

## Files

| File              | Responsibility                                                                       |
|-------------------|--------------------------------------------------------------------------------------|
| `agent.go`        | Agent struct, constructor (`New`), status check, LLM wiring                          |
| `capabilities.go` | Accessors exposed to `agent/tools` and `agent/api` (`Bus`, `ConfigPath`, …)          |
| `scheduler.go`    | `StartAutonomous(ctx)` — schedule + briefing loops; blocks on `ctx.Done()` (no HTTP) |
| `cycle.go`        | `runAgentCycle` — native multi-turn tool-calling loop; threads tool results back     |
| `think.go`        | One `ChatWithTools` turn + token accounting; tool-definition snapshot, arg coercion, prompt-JSON fallback |
| `context.go` / `prompt.go` | Build system prompt context (state, rules, ack list, briefing schedule)     |
| `chat.go` / `sessions.go` | `StreamChat` (transport-free), in-memory chat sessions (`"web-default"`)      |
| `exec.go`         | Slash command execution (/scrape, /status, /stats, /ack, /memory)                   |
| `publish.go`      | `PublishProgress` and SSE event emission helpers                                     |
| `format.go`       | Output formatting helpers (`formatItem` sanitizes scraped fields)                   |
| `guardrail.go`    | Non-overridable untrusted-data guardrail text, fence markers, `fenceUntrusted` / `sanitizeUntrusted` |
| `version.go`      | Build-time `Version` variable                                                        |

Code that moved out of this directory: `agent/events` (SSE bus), `agent/tools` (registry + builtins +
`ComputeDiff`), `agent/api` (REST handlers + `Register`), and the top-level `notification` concept
(`MacNotifier`, `BusNotifier`, `MultiNotifier`, `NoopNotifier`).

The HTTP server, embedded frontend/static serving, `POST /api/open`, and the desktop-only
`GET`/`DELETE /api/logs` (file tail + prune) live in the `internal/ui/webui` package. `cmd` runs `agent.StartAutonomous(ctx)` and
`webui.Serve(ctx, webui.Options{RegisterAPI: func(mux){ agentapi.Register(a, mux) }})`, so `agent`
never imports `ui` and `ui` never imports `agent`.

## Important Invariants

- HTTP server binds to `127.0.0.1` only (security: no network exposure).
- CORS restricted to `localhost` / `127.0.0.1` origins.
- EventBus channel capacity is 64; Publish drops events (with `default` case) when full.
- Max 10 tool calls per agent cycle (`maxActionsPerCycle`).
- Agent memory is saved to store after each cycle.
- `notify_user` publishes one SSE event (with fingerprint); `MacNotifier` is macOS-only (`osascript`); `BusNotifier`
  publishes to the event bus for web UI delivery on all platforms.
- Chat sessions use `"web-default"` as the persistent session ID.
- Acknowledged alerts (24h window) are included in LLM context to prevent re-notification.
- **Prompt-injection guardrail (non-overridable):** all scraped/external data injected into LLM prompts — both
  chat `BuildContext` and the autonomous `buildAgentPrompt` (Current State, Actions, acknowledged titles,
  previous-session memory) — must be wrapped in `BEGIN/END UNTRUSTED DATA` fences and preceded by
  `untrustedDataGuardrail`, which is always the highest-priority system text. Scraped fields are run through
  `sanitizeUntrusted` so an item cannot forge a fence and break out. Any new prompt path that surfaces scraped
  content must reuse these helpers; do not interpolate raw item text into a prompt.

## Pitfalls

- `EventBus.Publish` silently drops events on full buffers — consider the EventRing improvement.
- Two SSE event sources in the frontend were consolidated to a single connection via `useSSE`.
- `chatSessions` is an in-memory map that never evicts — potential memory leak for long-running agents.
- Tool calls now arrive via native function-calling; `parseToolCall` (prompt-JSON) survives only as the degraded fallback for models that ignore tools.
- The `handleRefresh` handler is defined but never registered on the mux.

## Relations

- Depends on: `platform/config`, `platform/clog`, `persistence/store`, `runtime/runner`,
  `security/keychain`, `notification`, `agent/events`, `agent/llm`, `agent/tools`
- Must **not** import: `ui`, `setup` (enforced by depguard; `agent/api` may import `setup`)
- Used by: `cmd/aide`, `cmd/aide-app`; `ui/webui` mounts `agent/api` routes via the `RegisterAPI` registrar
