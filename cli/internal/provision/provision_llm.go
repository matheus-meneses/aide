package provision

import (
	"aide/cli/internal/config"
	"aide/cli/internal/keychain"
	"fmt"
	"strings"
)

type ProviderInfo struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	DefaultURL string `json:"default_url"`
}

func Providers() []ProviderInfo {
	return []ProviderInfo{
		{ID: "openai", Label: "OpenAI-compatible (OpenAI, Azure, vLLM, Ollama, …)", DefaultURL: "https://api.openai.com/v1"},
		{ID: "litellm", Label: "LiteLLM proxy", DefaultURL: ""},
		{ID: "anthropic", Label: "Anthropic (Claude native API)", DefaultURL: "https://api.anthropic.com"},
	}
}

type LLMInput struct {
	Provider      string   `json:"provider"`
	BaseURL       string   `json:"base_url"`
	Model         string   `json:"model"`
	APIKey        string   `json:"api_key"`
	RunInterval   string   `json:"run_interval"`
	BriefingTimes []string `json:"briefing_times"`
}

// SetLLM writes the agent LLM settings to config.yaml and stores the API key in
// the keychain (never in the config file).
func SetLLM(cfgPath string, in LLMInput) error {
	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return err
	}

	provider := strings.TrimSpace(in.Provider)
	if provider == "" {
		provider = "openai"
	}
	cfg.Agent.LLMProvider = provider
	cfg.Agent.LLMURL = strings.TrimSpace(in.BaseURL)
	cfg.Agent.LLMModel = strings.TrimSpace(in.Model)
	cfg.Agent.LLMAPIKey = ""
	if ri := strings.TrimSpace(in.RunInterval); ri != "" {
		cfg.Agent.RunInterval = ri
	}
	if len(in.BriefingTimes) > 0 {
		cfg.Agent.BriefingTimes = in.BriefingTimes
	}

	if err := cfg.Save(cfgPath); err != nil {
		return err
	}

	if key := strings.TrimSpace(in.APIKey); key != "" {
		if err := keychain.SetField("agent", "llm_api_key", key); err != nil {
			return fmt.Errorf("storing API key: %w", err)
		}
	}
	return nil
}
