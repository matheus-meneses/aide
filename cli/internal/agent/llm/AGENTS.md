# agent/llm

## Purpose

Provider-agnostic LLM client used by the agent for chat, streaming chat, reachability checks, and token accounting.
This package isolates all HTTP/wire details of the supported providers behind one interface so the rest of `agent`
depends on types, not vendors.

## Key Types

- `LLM` — interface: `Chat`, `ChatStream`, `ChatWithTools`, `Ping`, `Model`.
- `ChatMessage`, `Usage`, `StreamCallback` — shared request/response value types. `ChatMessage` also carries optional `ToolCalls` (assistant turns) and `ToolCallID`/`Name` (role `"tool"` results) for the function-calling path.
- `ToolDefinition` (name/description + JSON-Schema `Parameters`), `ToolCall` (id/name + raw JSON `Arguments`), `ChatResult` (text + tool calls + usage) — native function-calling value types.
- `Provider` + `ProviderOpenAI` / `ProviderLiteLLM` / `ProviderAnthropic` constants.

## Entry points

- `NewLLM(provider, baseURL, model, apiKey)` — constructs the right client for the provider.
- `NormalizeProvider`, `SupportedProviders`, `DefaultBaseURL` — provider name/URL helpers (also used by `cmd/aide`).

## Files

| File           | Responsibility                                                        |
|----------------|-----------------------------------------------------------------------|
| `llm.go`       | Interface, value types, provider constants, `NewLLM`                  |
| `transport.go` | Shared HTTP plumbing: `baseClient` (embedded), `postJSON`, `scanSSE`  |
| `openai.go`    | OpenAI/LiteLLM-compatible client (`/chat/completions`)                |
| `anthropic.go` | Anthropic Messages API client (`/v1/messages`)                        |

## Invariants

- Imports stdlib only — never imports `agent` (no import cycle).
- Provider client constructors (`newOpenAIClient`, `newAnthropicClient`) stay unexported.
- Streaming clients invoke the `StreamCallback` per delta and return the full text + `Usage`.
- Both clients embed `baseClient` and share one transport (`postJSON`/`scanSSE`); request execution, SSE scanning, and `Model()` are defined once, not per provider.
- `ChatWithTools` is non-streaming and translates the neutral `ChatMessage`/`ToolCall` model into each vendor's wire shape (OpenAI `tool_calls` + `role:"tool"`; Anthropic `tool_use`/`tool_result` content blocks). `Chat`/`ChatStream` are unchanged.

## Protocol references

`ChatWithTools` implements each provider's native function-calling protocol rather than a homegrown convention. Tool schemas are JSON Schema objects.

- OpenAI / LiteLLM function calling (`tools`, `tool_calls`, `role:"tool"`): https://platform.openai.com/docs/guides/function-calling
- Anthropic tool use (`tools` with `input_schema`, `tool_use` / `tool_result` content blocks): https://platform.claude.com/docs/en/agents-and-tools/tool-use/overview
- JSON Schema (tool `parameters` / `input_schema`): https://json-schema.org/

## Relations

- Used by: `internal/agent`, `cmd/aide` (provider/URL helpers).
