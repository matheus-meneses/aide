package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ChatMessage is the provider-neutral conversation turn. ToolCalls/ToolCallID/
// Name carry function-calling state for the ChatWithTools path; the plain Chat/
// ChatStream paths leave them empty. Provider clients translate these into the
// vendor wire format, so ToolCalls is never serialized directly.
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"-"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ToolDefinition describes a callable tool exposed to the model. Parameters is a
// JSON Schema object describing the tool's arguments.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

// ToolCall is a single tool invocation requested by the model. Arguments is the
// raw JSON object of arguments as produced by the provider.
type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

// ChatResult is the structured outcome of a ChatWithTools turn: assistant text
// (may be empty when the model only calls tools) and any requested tool calls.
type ChatResult struct {
	Content   string
	ToolCalls []ToolCall
	Usage     *Usage
}

type StreamCallback func(chunk string)

type LLM interface {
	Chat(ctx context.Context, messages []ChatMessage) (string, *Usage, error)
	ChatStream(ctx context.Context, messages []ChatMessage, cb StreamCallback) (string, *Usage, error)
	ChatWithTools(ctx context.Context, messages []ChatMessage, tools []ToolDefinition) (*ChatResult, error)
	Ping() error
	Model() string
}

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderLiteLLM   Provider = "litellm"
	ProviderAnthropic Provider = "anthropic"
)

func NormalizeProvider(provider string) Provider {
	p := Provider(strings.ToLower(strings.TrimSpace(provider)))
	if p == "" {
		return ProviderOpenAI
	}
	return p
}

func SupportedProviders() []Provider {
	return []Provider{ProviderOpenAI, ProviderLiteLLM, ProviderAnthropic}
}

func DefaultBaseURL(provider string) string {
	switch NormalizeProvider(provider) {
	case ProviderOpenAI:
		return "https://api.openai.com/v1"
	case ProviderAnthropic:
		return "https://api.anthropic.com"
	default:
		return ""
	}
}

func NewLLM(provider, baseURL, model, apiKey string) (LLM, error) {
	switch NormalizeProvider(provider) {
	case ProviderOpenAI, ProviderLiteLLM:
		return newOpenAIClient(baseURL, model, apiKey), nil
	case ProviderAnthropic:
		return newAnthropicClient(baseURL, model, apiKey), nil
	default:
		return nil, fmt.Errorf("unknown llm provider %q (supported: openai, litellm, anthropic)", provider)
	}
}
