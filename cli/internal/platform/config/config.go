package config

import (
	"aide/cli/internal/platform/xdg"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Settings   Settings          `yaml:"settings"`
	Sources    map[string]Source `yaml:"sources"`
	Agent      AgentConfig       `yaml:"agent"`
	Registries []string          `yaml:"registries,omitempty"`
}

type AgentConfig struct {
	RunInterval   string   `yaml:"run_interval"`
	BriefingTimes []string `yaml:"briefing_times"`
	LLMProvider   string   `yaml:"llm_provider,omitempty"`
	LLMModel      string   `yaml:"llm_model"`
	LLMURL        string   `yaml:"llm_url"`
	LLMAPIKey     string   `yaml:"llm_api_key,omitempty"`
	UserContext   string   `yaml:"user_context,omitempty"`
}

func (a AgentConfig) RunIntervalDuration() time.Duration {
	d, err := time.ParseDuration(a.RunInterval)
	if err != nil {
		return 30 * time.Minute
	}
	return d
}

type Settings struct {
	Concurrency    int    `yaml:"concurrency" json:"concurrency"`
	TimeoutSeconds int    `yaml:"timeout_seconds" json:"timeout_seconds"`
	DataDir        string `yaml:"data_dir" json:"data_dir"`
	LogLevel       string `yaml:"log_level,omitempty" json:"log_level"`
	LogFormat      string `yaml:"log_format,omitempty" json:"log_format"`
	AutoUpdate     string `yaml:"auto_update,omitempty" json:"auto_update"`
	TLS            TLS    `yaml:"tls,omitempty" json:"tls"`
}

type TLS struct {
	VerifySSL *bool  `yaml:"verify_ssl,omitempty" json:"verify_ssl"`
	CABundle  string `yaml:"ca_bundle,omitempty" json:"ca_bundle"`
}

// Auto-update modes control how aide reacts to a newer release being available.
const (
	AutoUpdateOff    = "off"    // never check
	AutoUpdateNotify = "notify" // check and surface a banner (default)
	AutoUpdateAuto   = "auto"   // check and apply self-applicable updates automatically
)

// ValidAutoUpdate reports whether mode is a recognized auto-update mode.
func ValidAutoUpdate(mode string) bool {
	switch mode {
	case AutoUpdateOff, AutoUpdateNotify, AutoUpdateAuto:
		return true
	default:
		return false
	}
}

type Source struct {
	Plugin  string         `yaml:"plugin,omitempty"`
	Enabled bool           `yaml:"enabled"`
	Config  map[string]any `yaml:"config,omitempty"`
	Context string         `yaml:"context,omitempty"`
	TLS     *TLS           `yaml:"tls,omitempty"`
}

func Load(path string) (*Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving config path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", absPath, err)
	}

	aideHome := xdg.AideHome()
	cfg := &Config{
		Settings: Settings{
			Concurrency:    5,
			TimeoutSeconds: 120,
			DataDir:        filepath.Join(aideHome, "data"),
			LogLevel:       "info",
			LogFormat:      "text",
			AutoUpdate:     AutoUpdateNotify,
		},
		Agent: AgentConfig{
			RunInterval:   "30m",
			BriefingTimes: []string{"08:00"},
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	cfg.ResolvePaths(filepath.Dir(absPath))

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Settings.Concurrency < 1 {
		return fmt.Errorf("concurrency must be >= 1")
	}
	if c.Settings.TimeoutSeconds < 1 {
		return fmt.Errorf("timeout_seconds must be >= 1")
	}
	return nil
}

func (c *Config) ResolvePaths(basedir string) {
	home, _ := os.UserHomeDir()
	resolve := func(p string) string {
		if strings.HasPrefix(p, "~/") {
			p = filepath.Join(home, p[2:])
		}
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(basedir, p)
	}
	c.Settings.DataDir = resolve(c.Settings.DataDir)
	if c.Settings.TLS.CABundle != "" {
		c.Settings.TLS.CABundle = resolve(c.Settings.TLS.CABundle)
	}
	for name, src := range c.Sources {
		if src.TLS != nil && src.TLS.CABundle != "" {
			src.TLS.CABundle = resolve(src.TLS.CABundle)
			c.Sources[name] = src
		}
	}
}

func (c *Config) EnabledSources() map[string]Source {
	enabled := make(map[string]Source)
	for name, src := range c.Sources {
		if src.Enabled {
			enabled[name] = src
		}
	}
	return enabled
}

func DefaultConfigPath() string {
	return filepath.Join(xdg.AideHome(), "config.yaml")
}

func LoadRaw(path string) (*Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving config path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", absPath, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

func (c *Config) Save(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(absPath, buf.Bytes(), 0o600)
}
