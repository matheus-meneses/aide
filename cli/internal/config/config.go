package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Settings Settings          `yaml:"settings"`
	Team     []TeamMember      `yaml:"team"`
	Sources  map[string]Source `yaml:"sources"`
	Agent    AgentConfig       `yaml:"agent"`
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
	ScrapersDir    string `yaml:"scrapers_dir"`
	PythonBin      string `yaml:"python_bin"`
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
	Enabled bool           `yaml:"enabled"`
	Config  map[string]any `yaml:"config"`
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

	home, _ := os.UserHomeDir()
	aideHome := filepath.Join(home, ".aide")

	cfg := &Config{
		Settings: Settings{
			Concurrency:    5,
			TimeoutSeconds: 120,
			DataDir:        filepath.Join(aideHome, "data"),
			ScrapersDir:    filepath.Join(aideHome, "scrapers"),
			PythonBin:      filepath.Join(aideHome, "scrapers", ".venv", "bin", "python"),
		},
		Agent: AgentConfig{
			RunInterval:   "30m",
			BriefingTimes: []string{"08:00"},
			LLMModel:      "AWS_ANTHROPIC-CLAUDE-SONNET-4.6",
			LLMURL:        "https://inter.genai.local/api/v1",
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
	c.Settings.ScrapersDir = resolve(c.Settings.ScrapersDir)
	c.Settings.PythonBin = resolve(c.Settings.PythonBin)
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
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aide", "config.yaml")
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

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(absPath, data, 0o644)
}
