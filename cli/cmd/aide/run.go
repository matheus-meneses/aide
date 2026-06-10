package main

import (
	"aide/cli/internal/config"
	"aide/cli/internal/render"
	"aide/cli/internal/runner"
	"aide/cli/internal/store"
	"fmt"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run scrapers and collect data",
	RunE:  runExecute,
}

func init() {
	runCmd.Flags().StringSlice("source", nil, "run specific sources (comma-separated)")
	runCmd.Flags().Int("concurrency", 0, "override parallel execution limit")
	rootCmd.AddCommand(runCmd)
}

func runExecute(cmd *cobra.Command, _ []string) error {
	sources, _ := cmd.Flags().GetStringSlice("source")
	concurrency, _ := cmd.Flags().GetInt("concurrency")

	return withStore(func(cfg *config.Config, s *store.Store) error {
		if concurrency > 0 {
			cfg.Settings.Concurrency = concurrency
		}

		if err := runner.SyncTeamFromConfig(cfg, s); err != nil {
			return fmt.Errorf("syncing team from config: %w", err)
		}

		r := runner.New(cfg, s)
		r.SetLogLevel(logLevel())
		r.SetLogFormat(logFormatValue())
		r.SetVerifySSL(verifySSLValue())

		if err := r.ValidateFilter(sources); err != nil {
			return err
		}

		result, err := r.Run(cmd.Context(), sources)
		if err != nil {
			return fmt.Errorf("run failed: %w", err)
		}

		render.PrintRunSummary(result)

		if result.SourcesFailed > 0 {
			return fmt.Errorf("%d of %d sources failed", result.SourcesFailed, result.SourcesTotal)
		}
		return nil
	})
}
