package provision

import (
	"aide/cli/internal/platform/config"
	"aide/cli/internal/security/keychain"
	"fmt"
	"strings"
)

// AgentSnapshot is the agent's LLM/schedule configuration without any secrets.
type AgentSnapshot struct {
	Provider      string           `json:"provider"`
	BaseURL       string           `json:"base_url"`
	Model         string           `json:"model"`
	RunInterval   string           `json:"run_interval"`
	BriefingTimes []string         `json:"briefing_times"`
	HasAPIKey     bool             `json:"has_api_key"`
	UserContext   string           `json:"user_context"`
	Preferences   AgentPreferences `json:"preferences"`
}

// AgentPreferences mirrors config.AgentPreferences for the secret-free snapshot
// and the preferences write endpoint.
type AgentPreferences struct {
	Notifications            string `json:"notifications"`
	MaxNotificationsPerCycle int    `json:"max_notifications_per_cycle"`
	Tone                     string `json:"tone"`
}

// Snapshot is a sanitized view of config.yaml used to prefill the management UI.
// It never contains API keys or stored credential values.
type Snapshot struct {
	Settings   config.Settings  `json:"settings"`
	Agent      AgentSnapshot    `json:"agent"`
	Sources    []SourceSnapshot `json:"sources"`
	Registries []string         `json:"registries"`
}

// ConfigSnapshot returns a secret-free view of the current configuration.
func ConfigSnapshot(cfgPath string) (*Snapshot, error) {
	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return nil, err
	}

	hasKey := strings.TrimSpace(cfg.Agent.LLMAPIKey) != ""
	if !hasKey {
		if cred, err := keychain.GetAll("agent"); err == nil {
			hasKey = strings.TrimSpace(cred.Fields["llm_api_key"]) != ""
		}
	}

	sources, err := ListSources(cfgPath)
	if err != nil {
		return nil, err
	}

	settings := cfg.Settings
	if settings.LogLevel == "" {
		settings.LogLevel = "info"
	}
	if settings.LogFormat == "" {
		settings.LogFormat = "text"
	}
	if settings.AutoUpdate == "" {
		settings.AutoUpdate = config.AutoUpdateNotify
	}

	return &Snapshot{
		Settings: settings,
		Agent: AgentSnapshot{
			Provider:      cfg.Agent.LLMProvider,
			BaseURL:       cfg.Agent.LLMURL,
			Model:         cfg.Agent.LLMModel,
			RunInterval:   cfg.Agent.RunInterval,
			BriefingTimes: cfg.Agent.BriefingTimes,
			HasAPIKey:     hasKey,
			UserContext:   cfg.Agent.UserContext,
			Preferences: AgentPreferences{
				Notifications:            cfg.Agent.Preferences.NotificationLevel(),
				MaxNotificationsPerCycle: cfg.Agent.Preferences.MaxNotificationsPerCycle,
				Tone:                     cfg.Agent.Preferences.Tone,
			},
		},
		Sources:    sources,
		Registries: cfg.Registries,
	}, nil
}

// ScheduleInput configures how often the agent re-collects and when it briefs.
type ScheduleInput struct {
	RunInterval   string   `json:"run_interval"`
	BriefingTimes []string `json:"briefing_times"`
}

// SetSchedule updates the run interval and briefing times in config.yaml.
func SetSchedule(cfgPath string, in ScheduleInput) error {
	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return err
	}
	if ri := strings.TrimSpace(in.RunInterval); ri != "" {
		cfg.Agent.RunInterval = ri
	}
	if in.BriefingTimes != nil {
		cfg.Agent.BriefingTimes = in.BriefingTimes
	}
	return cfg.Save(cfgPath)
}

// SetUserContext writes the user's free-text context block (the "about me /
// how to help me" layer) to config.yaml. An empty value clears it.
func SetUserContext(cfgPath, context string) error {
	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return err
	}
	cfg.Agent.UserContext = strings.TrimSpace(context)
	return cfg.Save(cfgPath)
}

// SetAgentPreferences writes the user's behavior preferences (notification
// level, max notifications per cycle, tone) to config.yaml. An empty/zero field
// clears that preference (falling back to the built-in default).
func SetAgentPreferences(cfgPath string, in AgentPreferences) error {
	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return err
	}
	level := strings.TrimSpace(in.Notifications)
	if level != "" && !config.ValidNotificationLevel(level) {
		return fmt.Errorf("notifications must be one of: silent, urgent_only, normal, all")
	}
	if in.MaxNotificationsPerCycle < 0 {
		return fmt.Errorf("max_notifications_per_cycle must be >= 0")
	}
	cfg.Agent.Preferences = config.AgentPreferences{
		Notifications:            level,
		MaxNotificationsPerCycle: in.MaxNotificationsPerCycle,
		Tone:                     strings.TrimSpace(in.Tone),
	}
	return cfg.Save(cfgPath)
}

// GeneralSettingsInput holds the editable general settings. data_dir is
// intentionally omitted: changing it would require reopening the store.
type GeneralSettingsInput struct {
	Concurrency    int    `json:"concurrency"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	VerifySSL      *bool  `json:"verify_ssl"`
	CABundle       string `json:"ca_bundle"`
	LogLevel       string `json:"log_level"`
	LogFormat      string `json:"log_format"`
	AutoUpdate     string `json:"auto_update"`
}

// SetGeneralSettings writes runtime settings (concurrency, timeout, TLS, logging)
// to config.yaml, validating numeric bounds.
func SetGeneralSettings(cfgPath string, in GeneralSettingsInput) error {
	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return err
	}

	if in.Concurrency > 0 {
		cfg.Settings.Concurrency = in.Concurrency
	}
	if in.TimeoutSeconds > 0 {
		cfg.Settings.TimeoutSeconds = in.TimeoutSeconds
	}
	if cfg.Settings.Concurrency < 1 {
		return fmt.Errorf("concurrency must be >= 1")
	}
	if cfg.Settings.TimeoutSeconds < 1 {
		return fmt.Errorf("timeout_seconds must be >= 1")
	}

	cfg.Settings.TLS.VerifySSL = in.VerifySSL
	cfg.Settings.TLS.CABundle = strings.TrimSpace(in.CABundle)

	if lvl := strings.TrimSpace(in.LogLevel); lvl != "" {
		cfg.Settings.LogLevel = lvl
	}
	if f := strings.TrimSpace(in.LogFormat); f != "" {
		cfg.Settings.LogFormat = f
	}
	if au := strings.TrimSpace(in.AutoUpdate); au != "" {
		if !config.ValidAutoUpdate(au) {
			return fmt.Errorf("auto_update must be one of: off, notify, auto")
		}
		cfg.Settings.AutoUpdate = au
	}

	return cfg.Save(cfgPath)
}
