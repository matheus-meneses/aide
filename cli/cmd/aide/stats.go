package main

import (
	"aide/cli/internal/persistence/store"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/ui/widgets"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show token usage statistics",
	RunE:  statsExecute,
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

func statsExecute(_ *cobra.Command, _ []string) error {
	return withStore(func(_ *config.Config, s *store.Store) error {
		summary, err := s.Tokens.Stats()
		if err != nil {
			return fmt.Errorf("querying stats: %w", err)
		}

		widgets.Println("Token Usage Statistics")
		widgets.Println("─────────────────────────")
		widgets.Printf("  Today:     %s tokens\n", formatTokens(summary.TodayTokens))
		widgets.Printf("  This week: %s tokens\n", formatTokens(summary.WeekTokens))
		widgets.Printf("  Avg/day:   %s tokens\n", formatTokens(summary.AvgPerDay))
		widgets.Printf("  Calls:     %d (7d)\n", summary.TotalCalls)
		widgets.Println()

		if len(summary.BySource) > 0 {
			widgets.Println("  By source (7d):")
			sources := make([]string, 0, len(summary.BySource))
			for src := range summary.BySource {
				sources = append(sources, src)
			}
			sort.Strings(sources)
			for _, src := range sources {
				widgets.Printf("    %-8s %s tokens\n", src, formatTokens(summary.BySource[src]))
			}
		}

		daily, err := s.Tokens.DailyStats(7)
		if err == nil && len(daily) > 0 {
			widgets.Println()
			widgets.Println("  Daily breakdown:")
			for _, d := range daily {
				total := d.Agent + d.Chat
				if total > 0 {
					widgets.Printf("    %s  agent: %s  chat: %s\n", d.Date, formatTokens(d.Agent), formatTokens(d.Chat))
				}
			}
		}

		return nil
	})
}

func formatTokens(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
