package config_test

import (
	"aide/cli/internal/platform/config"
	"aide/cli/internal/testutil"
	"path/filepath"
	"testing"
)

func TestLoadAppliesDefaults(t *testing.T) {
	home := testutil.TempAideHome(t)
	path := testutil.WriteConfig(t, "settings: {}\n")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Settings.Concurrency != 5 {
		t.Errorf("Concurrency = %d, want 5", cfg.Settings.Concurrency)
	}
	if cfg.Settings.TimeoutSeconds != 120 {
		t.Errorf("TimeoutSeconds = %d, want 120", cfg.Settings.TimeoutSeconds)
	}
	if cfg.Settings.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.Settings.LogLevel)
	}
	if cfg.Settings.LogFormat != "text" {
		t.Errorf("LogFormat = %q, want text", cfg.Settings.LogFormat)
	}
	if cfg.Agent.RunInterval != "30m" {
		t.Errorf("RunInterval = %q, want 30m", cfg.Agent.RunInterval)
	}
	if want := filepath.Join(home, "data"); cfg.Settings.DataDir != want {
		t.Errorf("DataDir = %q, want %q", cfg.Settings.DataDir, want)
	}
}

func TestLoadOverridesDefaults(t *testing.T) {
	testutil.TempAideHome(t)
	path := testutil.WriteConfig(t, `settings:
  concurrency: 3
  timeout_seconds: 45
  log_level: debug
sources:
  jira:
    enabled: true
  github:
    enabled: false
`)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Settings.Concurrency != 3 {
		t.Errorf("Concurrency = %d, want 3", cfg.Settings.Concurrency)
	}
	if cfg.Settings.TimeoutSeconds != 45 {
		t.Errorf("TimeoutSeconds = %d, want 45", cfg.Settings.TimeoutSeconds)
	}
	if cfg.Settings.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.Settings.LogLevel)
	}
	if len(cfg.Sources) != 2 {
		t.Errorf("Sources len = %d, want 2", len(cfg.Sources))
	}
}

func TestLoadValidationBounds(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{"zero concurrency", "settings:\n  concurrency: 0\n", true},
		{"negative concurrency", "settings:\n  concurrency: -1\n", true},
		{"zero timeout", "settings:\n  timeout_seconds: 0\n", true},
		{"valid", "settings:\n  concurrency: 1\n  timeout_seconds: 1\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TempAideHome(t)
			path := testutil.WriteConfig(t, tt.yaml)
			_, err := config.Load(path)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := config.Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("expected error loading missing file")
	}
}

func TestResolvePaths(t *testing.T) {
	c := &config.Config{}
	c.Settings.DataDir = "relative/data"
	c.ResolvePaths("/base/dir")
	if want := filepath.FromSlash("/base/dir/relative/data"); c.Settings.DataDir != want {
		t.Errorf("relative DataDir = %q, want %q", c.Settings.DataDir, want)
	}

	abs := &config.Config{}
	abs.Settings.DataDir = filepath.FromSlash("/absolute/data")
	abs.ResolvePaths("/base/dir")
	if want := filepath.FromSlash("/absolute/data"); abs.Settings.DataDir != want {
		t.Errorf("absolute DataDir = %q, want %q", abs.Settings.DataDir, want)
	}
}

func TestEnabledSources(t *testing.T) {
	c := &config.Config{Sources: map[string]config.Source{
		"a": {Enabled: true},
		"b": {Enabled: false},
		"c": {Enabled: true},
	}}
	enabled := c.EnabledSources()
	if len(enabled) != 2 {
		t.Fatalf("EnabledSources len = %d, want 2", len(enabled))
	}
	if _, ok := enabled["b"]; ok {
		t.Error("disabled source b should not be returned")
	}
}

func TestRunIntervalDuration(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"15m", "15m0s"},
		{"2h", "2h0m0s"},
		{"", "30m0s"},
		{"garbage", "30m0s"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := config.AgentConfig{RunInterval: tt.in}.RunIntervalDuration().String()
			if got != tt.want {
				t.Errorf("RunIntervalDuration(%q) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}
