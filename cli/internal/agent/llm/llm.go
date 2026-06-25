package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

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

type ToolDefinition struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

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
