# agent

## Purpose

Autonomous AI agent that runs on a timer, uses LLM tool-calling to decide what actions to take (scrape, diff, notify), and serves a web UI with real-time event streaming.

## Key Types

- `Agent` — Central orchestrator: holds config, store, runner, LLM client, notifier, event bus, tool registry.
- `LLMClient` — HTTP streaming client for the configured LLM API (SSE format); endpoint set via `agent.llm` config or `AIDE_LLM_URL` env var.
- `EventBus` — In-memory pub/sub for SSE events with subscriber channels.
- `Event` — SSE event with type, data (JSON string), and timestamp.
- `ToolRegistry` / `Tool` — Agent's available actions (scrape, diff, notify_user, send_message, check_items, check_today, check_health, done).
- `Notifier` interface — `MacNotifier`, `BusNotifier`, `MultiNotifier`, `NoopNotifier`.
- `StatusResult` — Agent health check response.

## Architecture

```
Timer tick → runAgentCycle → build context → LLM → parse tool calls → execute tools → loop until "done"
                                                                          ↓
                                                                   EventBus.Publish
                                                                          ↓
                                                                   ServeSSE → browser
```

## Files

| File | Responsibility |
|------|---------------|
| `agent.go` | Agent struct, constructor, status check |
| `loop.go` | `StartAutonomous`, `runAgentCycle`, timer, system prompt, tool-call parsing |
| `context.go` | Builds system prompt context (state, rules, ack list, briefing schedule) |
| `tools.go` | Tool definitions and `postToChatAndSSE` |
| `server.go` | HTTP server, routes, CORS, embedded frontend |
| `handlers.go` | REST handlers (chat, items, today, status, ack, sessions, stats, memory, whoami, exec) |
| `sse.go` | EventBus (pub/sub), ServeSSE, BusNotifier |
| `notify.go` | MacNotifier (osascript), MultiNotifier, NoopNotifier |
| `llm.go` | LLM streaming client |
| `exec.go` | Slash command execution (/scrape, /status, /stats, /ack, /memory) |
| `diff.go` | ComputeDiff between scrape runs |
| `embed.go` | `//go:embed frontend/dist` for static assets |

## Important Invariants

- HTTP server binds to `127.0.0.1` only (security: no network exposure).
- CORS restricted to `localhost` / `127.0.0.1` origins.
- EventBus channel capacity is 64; Publish drops events (with `default` case) when full.
- Max 10 tool calls per agent cycle (`maxActionsPerCycle`).
- Agent memory is saved to store after each cycle.
- `notify_user` publishes one SSE event (with fingerprint); `MacNotifier` is macOS-only (`osascript`); `BusNotifier` publishes to the event bus for web UI delivery on all platforms.
- Chat sessions use `"web-default"` as the persistent session ID.
- Acknowledged alerts (24h window) are included in LLM context to prevent re-notification.

## Pitfalls

- `EventBus.Publish` silently drops events on full buffers — consider the EventRing improvement.
- Two SSE event sources in the frontend were consolidated to a single connection via `useSSE`.
- `chatSessions` is an in-memory map that never evicts — potential memory leak for long-running agents.
- LLM response parsing uses regex for tool calls in markdown code blocks — fragile against format changes.
- The `handleRefresh` handler is defined but never registered on the mux.

## Relations

- Depends on: `config`, `store`, `runner`, `keychain`, `updater`
- Used by: `cmd/aide` (agent command)
