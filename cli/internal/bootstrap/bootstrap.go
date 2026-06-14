package bootstrap

import (
	"aide/cli/internal/config"
	"aide/cli/internal/plugin"
	"aide/cli/internal/pyenv"
	"aide/cli/internal/xdg"
	"fmt"
	"os"
	"path/filepath"
)

type ProgressFunc func(msg string)

func report(progress ProgressFunc, format string, args ...any) {
	if progress != nil {
		progress(fmt.Sprintf(format, args...))
	}
}

func ConfigPath() string {
	return filepath.Join(xdg.AideHome(), "config.yaml")
}

func setupMarkerPath() string {
	return filepath.Join(xdg.AideHome(), ".setup-complete")
}

// SetupCompleted reports whether the first-run wizard has been finished. A
// previously configured LLM also counts as complete so existing installs are
// never bounced back into setup.
func SetupCompleted() bool {
	if _, err := os.Stat(setupMarkerPath()); err == nil {
		return true
	}
	if cfg, err := config.LoadRaw(ConfigPath()); err == nil {
		if cfg.Agent.LLMModel != "" && cfg.Agent.LLMURL != "" {
			return true
		}
	}
	return false
}

// MarkSetupComplete records that the first-run wizard has been finished.
func MarkSetupComplete() error {
	if err := os.MkdirAll(xdg.AideHome(), 0o755); err != nil {
		return fmt.Errorf("creating aide home: %w", err)
	}
	if err := os.WriteFile(setupMarkerPath(), []byte("ok\n"), 0o600); err != nil {
		return fmt.Errorf("writing setup marker: %w", err)
	}
	return nil
}

// NeedsSetup reports whether first-run setup is required. The LLM/agent config
// is optional and can be filled in later, so completion is tracked explicitly
// rather than inferred from whether a model is set.
func NeedsSetup() bool {
	return !SetupCompleted()
}

// PythonReady reports whether the standalone Python runtime is present.
func PythonReady() bool {
	_, err := os.Stat(pyenv.BinPath(xdg.AideHome()))
	return err == nil
}

// Ensure creates the aide home layout, installs the standalone Python runtime,
// caches the plugin registry, and writes a default config if none exists. It is
// safe to call repeatedly; existing artifacts are left untouched.
func Ensure(progress ProgressFunc) error {
	base := xdg.AideHome()
	dataDir := filepath.Join(base, "data")
	pluginsDir := filepath.Join(base, "plugins")
	cfgPath := filepath.Join(base, "config.yaml")

	report(progress, "Preparing %s", base)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return fmt.Errorf("creating plugins dir: %w", err)
	}

	if _, err := pyenv.Ensure(base, pyenv.ProgressFunc(progress)); err != nil {
		return fmt.Errorf("python runtime: %w", err)
	}

	var registries []string
	if cfg, err := config.LoadRaw(cfgPath); err == nil {
		registries = cfg.Registries
	}

	report(progress, "Fetching plugin registry...")
	if idx, err := plugin.MergedIndex(registries); err != nil {
		report(progress, "Registry fetch failed (continuing offline): %v", err)
	} else if err := plugin.CacheIndex(idx); err != nil {
		report(progress, "Could not cache registry: %v", err)
	} else {
		report(progress, "Registry cached (%d plugins available)", len(idx.Plugins))
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if err := os.WriteFile(cfgPath, []byte(defaultConfig(base)), 0o600); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}
		report(progress, "Generated config.yaml")
	}

	return nil
}

// EnsureConfigScaffold creates the aide home layout and a default config file
// if none exists, without downloading the Python runtime. It lets the desktop
// app boot and serve the setup UI before the heavier bootstrap runs.
func EnsureConfigScaffold() error {
	base := xdg.AideHome()
	if err := os.MkdirAll(filepath.Join(base, "data"), 0o755); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(base, "plugins"), 0o755); err != nil {
		return fmt.Errorf("creating plugins dir: %w", err)
	}
	cfgPath := filepath.Join(base, "config.yaml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if err := os.WriteFile(cfgPath, []byte(defaultConfig(base)), 0o600); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}
	}
	return nil
}

func defaultConfig(base string) string {
	return fmt.Sprintf(`settings:
  concurrency: 5
  timeout_seconds: 120
  data_dir: "%s/data"

team:
  - name: "Your Name"
    aliases: ["you@example.com"]

sources: {}

agent:
  run_interval: "30m"
  briefing_times: ["08:00"]
  llm_model: ""
  llm_url: ""

registries: []
`, base)
}
