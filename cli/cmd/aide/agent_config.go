package main

import (
	"aide/cli/internal/agent/llm"
	"aide/cli/internal/security/keychain"
	"aide/cli/internal/setup/provision"
	"aide/cli/internal/ui/widgets"
	"context"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

const manualModelEntry = "Other (type the name manually)"

var providerOptions = []struct {
	id    string
	label string
}{
	{"openai", "OpenAI-compatible (OpenAI, Azure, vLLM, Ollama, …)"},
	{"litellm", "LiteLLM proxy"},
	{"anthropic", "Anthropic (Claude native API)"},
}

func promptProvider(current string) (string, error) {
	current = string(llm.NormalizeProvider(current))

	labels := make([]string, len(providerOptions))
	defaultLabel := providerOptions[0].label
	for i, opt := range providerOptions {
		labels[i] = opt.label
		if opt.id == current {
			defaultLabel = opt.label
		}
	}

	var chosen string
	if err := survey.AskOne(&survey.Select{
		Message: "LLM provider",
		Options: labels,
		Default: defaultLabel,
	}, &chosen); err != nil {
		return "", err
	}

	for _, opt := range providerOptions {
		if opt.label == chosen {
			return opt.id, nil
		}
	}
	return "openai", nil
}

var agentConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure the agent's LLM endpoint and schedule",
	RunE:  agentConfigExecute,
}

func agentConfigExecute(_ *cobra.Command, _ []string) error {
	cfg, err := loadRawConfig()
	if err != nil {
		return err
	}

	widgets.Println("Configure agent mode. The agent talks only to the endpoint you set here.")
	widgets.Println()

	provider, err := promptProvider(cfg.Agent.LLMProvider)
	if err != nil {
		return err
	}

	llmURL := cfg.Agent.LLMURL
	if llmURL == "" {
		llmURL = llm.DefaultBaseURL(provider)
	}
	if err := survey.AskOne(&survey.Input{
		Message: "LLM base URL",
		Default: llmURL,
		Help:    "OpenAI/LiteLLM: the API base (…/v1). Anthropic: https://api.anthropic.com",
	}, &llmURL); err != nil {
		return err
	}

	apiKey, err := promptAgentAPIKey(strings.TrimSpace(cfg.Agent.LLMAPIKey))
	if err != nil {
		return err
	}

	llmModel, err := promptModel(provider, llmURL, effectiveAPIKey(apiKey, cfg.Agent.LLMAPIKey), cfg.Agent.LLMModel)
	if err != nil {
		return err
	}

	runInterval := cfg.Agent.RunInterval
	if runInterval == "" {
		runInterval = "30m"
	}
	if err := survey.AskOne(&survey.Input{
		Message: "Run interval (how often the background agent re-collects)",
		Default: runInterval,
	}, &runInterval); err != nil {
		return err
	}

	briefings := strings.Join(cfg.Agent.BriefingTimes, ", ")
	if briefings == "" {
		briefings = "08:00"
	}
	if err := survey.AskOne(&survey.Input{
		Message: "Daily briefing times (comma-separated, 24h)",
		Default: briefings,
	}, &briefings); err != nil {
		return err
	}

	if err := provision.SetLLM(cfgFile, provision.LLMInput{
		Provider:      provider,
		BaseURL:       llmURL,
		Model:         llmModel,
		APIKey:        apiKey,
		RunInterval:   runInterval,
		BriefingTimes: parseBriefingTimes(briefings),
	}); err != nil {
		return err
	}

	widgets.Println("\n✓ Agent configured.")
	widgets.Println("  Verify connectivity:  aide agent status")
	widgets.Println("  Start the agent:      aide agent start")
	return nil
}

// promptAgentAPIKey returns the API key to store (empty to leave unchanged).
// An existing plaintext key in config is migrated to the keychain transparently.
func promptAgentAPIKey(existingPlainKey string) (string, error) {
	hasKey := false
	if cred, err := keychain.GetAll("agent"); err == nil {
		if v, ok := cred.Fields["llm_api_key"]; ok && v != "" {
			hasKey = true
		}
	}

	if existingPlainKey != "" && !hasKey {
		return existingPlainKey, nil
	}

	message := "Set an LLM API key? (stored in your OS keychain)"
	if hasKey {
		message = "An API key is already stored. Replace it?"
	}

	var setKey bool
	if err := survey.AskOne(&survey.Confirm{
		Message: message,
		Default: !hasKey,
	}, &setKey); err != nil {
		return "", err
	}
	if !setKey {
		return "", nil
	}

	var key string
	if err := survey.AskOne(&survey.Password{
		Message: "LLM API key",
	}, &key); err != nil {
		return "", err
	}
	return strings.TrimSpace(key), nil
}

// effectiveAPIKey resolves the key to use when probing the provider for models:
// a freshly entered key wins, then the keychain, then a legacy plaintext key.
func effectiveAPIKey(entered, existingPlain string) string {
	if k := strings.TrimSpace(entered); k != "" {
		return k
	}
	if cred, err := keychain.GetAll("agent"); err == nil {
		if v, ok := cred.Fields["llm_api_key"]; ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	return strings.TrimSpace(existingPlain)
}

// promptModel offers a list fetched from the provider when reachable, and
// always falls back to free-text entry.
func promptModel(provider, baseURL, apiKey, current string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	models, err := llm.ListModels(ctx, provider, baseURL, apiKey)
	if err != nil || len(models) == 0 {
		if err != nil {
			widgets.Println("  Could not list models from the provider (" + err.Error() + "). Enter it manually.")
		}
		return promptModelManual(current)
	}

	options := append(append([]string{}, models...), manualModelEntry)
	defaultChoice := manualModelEntry
	for _, m := range models {
		if m == current {
			defaultChoice = m
			break
		}
	}

	var chosen string
	if err := survey.AskOne(&survey.Select{
		Message: "Model",
		Options: options,
		Default: defaultChoice,
	}, &chosen); err != nil {
		return "", err
	}
	if chosen == manualModelEntry {
		return promptModelManual(current)
	}
	return chosen, nil
}

func promptModelManual(current string) (string, error) {
	model := current
	if err := survey.AskOne(&survey.Input{
		Message: "Model",
		Default: model,
		Help:    "e.g. llama3.1, gpt-4o-mini",
	}, &model); err != nil {
		return "", err
	}
	return model, nil
}

func parseBriefingTimes(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' '
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
