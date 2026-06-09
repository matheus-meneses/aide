package main

import (
	"aide/cli/internal/config"
	"aide/cli/internal/plugin"
	"aide/cli/internal/xdg"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Setup aide home directory, install Python runtime, and fetch the plugin registry",
	RunE:  initExecute,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func aideHome() string {
	return xdg.AideHome()
}

func initExecute(_ *cobra.Command, _ []string) error {
	base := aideHome()
	dataDir := filepath.Join(base, "data")
	pluginsDir := filepath.Join(base, "plugins")
	configPath := filepath.Join(base, "config.yaml")

	fmt.Printf("Initializing aide in %s\n", base)

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}
	fmt.Println("  [+] Created data/")

	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return fmt.Errorf("creating plugins dir: %w", err)
	}
	fmt.Println("  [+] Created plugins/")

	fmt.Println("  [+] Setting up Python runtime...")
	if _, err := ensurePython(base); err != nil {
		fmt.Printf("  [!] Python setup failed: %v\n", err)
		fmt.Println("  [!] Plugin installation will fall back to system Python.")
	}

	var registries []string
	if _, err := os.Stat(configPath); err == nil {
		if cfg, err := config.Load(configPath); err == nil {
			registries = cfg.Registries
		}
	}

	fmt.Println("  [+] Fetching plugin registry...")
	idx, err := plugin.MergedIndex(registries)
	if err != nil {
		fmt.Printf("  [!] Registry fetch failed (will work offline): %v\n", err)
	} else {
		if cacheErr := plugin.CacheIndex(idx); cacheErr != nil {
			fmt.Printf("  [!] Could not cache registry: %v\n", cacheErr)
		} else {
			fmt.Printf("  [+] Registry cached (%d plugins available)\n", len(idx.Plugins))
		}
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(defaultConfig(base)), 0o600); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}
		fmt.Println("  [+] Generated config.yaml")
	} else {
		fmt.Println("  [=] config.yaml already exists, skipped")
	}

	fmt.Println("\nDone!")
	fmt.Println("  Install plugins:  aide plugin install <name>")
	fmt.Println("  Add a source:     aide config source add")
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
