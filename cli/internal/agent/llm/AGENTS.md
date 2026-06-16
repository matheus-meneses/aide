# agent/llm

## Purpose

Provider-agnostic LLM client used by the agent for chat, streaming chat, reachability checks, and token accounting.
This package isolates all HTTP/wire details of the supported providers behind one interface so the rest of `agent`
depends on types, not vendors.

## Key Types

- `LLM` — interface: `Chat`, `ChatStream`, `Ping`, `Model`.
- `ChatMessage`, `Usage`, `StreamCallback` — shared request/response value types.
- `Provider` + `ProviderOpenAI` / `ProviderLiteLLM` / `ProviderAnthropic` constants.

## Entry points

- `NewLLM(provider, baseURL, model, apiKey)` — constructs the right client for the provider.
- `NormalizeProvider`, `SupportedProviders`, `DefaultBaseURL` — provider name/URL helpers (also used by `cmd/aide`).

## Files

| File           | Responsibility                                            |
|----------------|-----------------------------------------------------------|
| `llm.go`       | Interface, value types, provider constants, `NewLLM`      |
| `openai.go`    | OpenAI/LiteLLM-compatible client (`/chat/completions`)    |
| `anthropic.go` | Anthropic Messages API client (`/v1/messages`)            |

## Invariants

- Imports stdlib only — never imports `agent` (no import cycle).
- Provider client constructors (`newOpenAIClient`, `newAnthropicClient`) stay unexported.
- Streaming clients invoke the `StreamCallback` per delta and return the full text + `Usage`.

## Relations

- Used by: `internal/agent`, `cmd/aide` (provider/URL helpers).
