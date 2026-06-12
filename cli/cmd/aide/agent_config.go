package main

import (
	"aide/cli/internal/config"
	"aide/cli/internal/keychain"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

var agentConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure the agent's LLM endpoint and schedule",
	RunE:  agentConfigExecute,
}

func agentConfigExecute(_ *cobra.Command, _ []string) error {
	cfg, err := config.LoadRaw(cfgFile)
	if err != nil {
		return err
	}

	fmt.Println("Configure agent mode. The agent talks only to the endpoint you set here.")
	fmt.Println()

	llmURL := cfg.Agent.LLMURL
	if err := survey.AskOne(&survey.Input{
		Message: "LLM URL (any OpenAI-compatible endpoint)",
		Default: llmURL,
		Help:    "e.g. http://localhost:11434/v1 for Ollama, or a hosted provider's base URL",
	}, &llmURL); err != nil {
		return err
	}

	llmModel := cfg.Agent.LLMModel
	if err := survey.AskOne(&survey.Input{
		Message: "Model",
		Default: llmModel,
		Help:    "e.g. llama3.1, gpt-4o-mini",
	}, &llmModel); err != nil {
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

	existingPlainKey := strings.TrimSpace(cfg.Agent.LLMAPIKey)

	cfg.Agent.LLMURL = strings.TrimSpace(llmURL)
	cfg.Agent.LLMModel = strings.TrimSpace(llmModel)
	cfg.Agent.RunInterval = strings.TrimSpace(runInterval)
	cfg.Agent.BriefingTimes = parseBriefingTimes(briefings)
	cfg.Agent.LLMAPIKey = ""

	if err := cfg.Save(cfgFile); err != nil {
		return err
	}

	if err := promptAgentAPIKey(existingPlainKey); err != nil {
		return err
	}

	fmt.Println("\n✓ Agent configured.")
	fmt.Println("  Verify connectivity:  aide agent status")
	fmt.Println("  Start the agent:      aide agent start")
	return nil
}

func promptAgentAPIKey(existingPlainKey string) error {
	hasKey := false
	if cred, err := keychain.GetAll("agent"); err == nil {
		if v, ok := cred.Fields["llm_api_key"]; ok && v != "" {
			hasKey = true
		}
	}

	if existingPlainKey != "" && !hasKey {
		if err := keychain.SetField("agent", "llm_api_key", existingPlainKey); err != nil {
			return fmt.Errorf("migrating API key to keychain: %w", err)
		}
		hasKey = true
		fmt.Println("  Moved your existing API key out of config.yaml and into the keychain.")
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
		return err
	}
	if !setKey {
		return nil
	}

	var key string
	if err := survey.AskOne(&survey.Password{
		Message: "LLM API key",
	}, &key); err != nil {
		return err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		fmt.Println("  (no key entered, leaving keychain unchanged)")
		return nil
	}
	if err := keychain.SetField("agent", "llm_api_key", key); err != nil {
		return fmt.Errorf("storing API key: %w", err)
	}
	fmt.Println("  API key stored in keychain.")
	return nil
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
