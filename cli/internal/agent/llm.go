package agent

import (
	"context"
	"fmt"
	"strings"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type StreamCallback func(chunk string)

type LLM interface {
	Chat(ctx context.Context, messages []ChatMessage) (string, *Usage, error)
	ChatStream(ctx context.Context, messages []ChatMessage, cb StreamCallback) (string, *Usage, error)
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
