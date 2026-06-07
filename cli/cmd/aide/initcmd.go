package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"aide/cli/internal/scrapers"
	"aide/cli/internal/updater"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Setup ~/.aide/ directory structure with scrapers, venv, and config",
	RunE:  initExecute,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func aideHome() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aide")
}

func initExecute(cmd *cobra.Command, args []string) error {
	base := aideHome()
	dataDir := filepath.Join(base, "data")
	scrapersDir := filepath.Join(base, "scrapers")
	configPath := filepath.Join(base, "config.yaml")
	registryPath := filepath.Join(base, "registry.yaml")

	fmt.Printf("Initializing aide in %s\n", base)

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}
	fmt.Println("  [+] Created data/")

	fmt.Println("  [+] Extracting scrapers...")
	if err := os.MkdirAll(scrapersDir, 0o755); err != nil {
		return fmt.Errorf("creating scrapers dir: %w", err)
	}
	if err := scrapers.ExtractTo(scrapersDir); err != nil {
		return fmt.Errorf("extracting scrapers: %w", err)
	}
	fmt.Println("  [+] Scrapers extracted")

	if err := setupScraperVenv(base, scrapersDir); err != nil {
		return err
	}

	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		fmt.Println("  [+] Downloading source registry...")
		regURL := fmt.Sprintf("%s/%s/registry.yaml", updater.NexusBaseURL, version)
		if dlErr := updater.DownloadToPath(regURL, registryPath); dlErr != nil {
			fmt.Printf("  [!] Registry download failed, using embedded fallback: %v\n", dlErr)
			writeEmbeddedRegistry(registryPath)
		}
	} else {
		fmt.Println("  [=] registry.yaml already exists")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(defaultConfig(base)), 0o644); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}
		fmt.Println("  [+] Generated config.yaml")
	} else {
		fmt.Println("  [=] config.yaml already exists, skipped")
	}

	fmt.Println("\nDone! Run 'aide config source add' to configure your first source.")
	return nil
}

func writeEmbeddedRegistry(path string) {
	data, err := scrapers.FS.ReadFile("embedded/registry.yaml")
	if err != nil {
		return
	}
	os.WriteFile(path, data, 0o644)
}

func defaultConfig(base string) string {
	return fmt.Sprintf(`settings:
  concurrency: 5
  timeout_seconds: 120
  data_dir: "%s/data"
  scrapers_dir: "%s/scrapers"
  python_bin: "%s/scrapers/.venv/bin/python"

team:
  - name: "Your Name"
    aliases: ["your.email@company.com"]

sources: {}

agent:
  run_interval: "30m"
  briefing_times: ["08:00"]
  llm_model: "AWS_ANTHROPIC-CLAUDE-SONNET-4.6"
  llm_url: "https://inter.genai.local/api/v1"
`, base, base, base)
}
