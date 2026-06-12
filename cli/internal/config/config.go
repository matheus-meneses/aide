package config

import (
	"aide/cli/internal/xdg"
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
	Team       []TeamMember      `yaml:"team"`
	Sources    map[string]Source `yaml:"sources"`
	Agent      AgentConfig       `yaml:"agent"`
	Registries []string          `yaml:"registries,omitempty"`
}

type AgentConfig struct {
	RunInterval   string   `yaml:"run_interval"`
	BriefingTimes []string `yaml:"briefing_times"`
	LLMModel      string   `yaml:"llm_model"`
	LLMURL        string   `yaml:"llm_url"`
	LLMAPIKey     string   `yaml:"llm_api_key"`
}

func (a AgentConfig) RunIntervalDuration() time.Duration {
	d, err := time.ParseDuration(a.RunInterval)
	if err != nil {
		return 30 * time.Minute
	}
	return d
}

type Settings struct {
	Concurrency    int    `yaml:"concurrency"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	DataDir        string `yaml:"data_dir"`
	TLS            TLS    `yaml:"tls,omitempty"`
}

type TLS struct {
	VerifySSL *bool  `yaml:"verify_ssl,omitempty"`
	CABundle  string `yaml:"ca_bundle,omitempty"`
}

type TeamMember struct {
	Name         string   `yaml:"name"`
	Aliases      []string `yaml:"aliases"`
	Email        string   `yaml:"email"`
	Role         string   `yaml:"role"`
	Department   string   `yaml:"department"`
	Branch       string   `yaml:"branch"`
	Registration string   `yaml:"registration"`
	Manager      string   `yaml:"manager"`
}

type Source struct {
	Plugin  string         `yaml:"plugin,omitempty"`
	Enabled bool           `yaml:"enabled"`
	Config  map[string]any `yaml:"config,omitempty"`
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

func (c *Config) ResolveMember(alias string) string {
	for _, member := range c.Team {
		if member.Name == alias {
			return member.Name
		}
		for _, a := range member.Aliases {
			if a == alias {
				return member.Name
			}
		}
	}
	return alias
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
