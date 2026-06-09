package main

import (
	"aide/cli/internal/config"
	"aide/cli/internal/store"
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

		fmt.Println("Token Usage Statistics")
		fmt.Println("─────────────────────────")
		fmt.Printf("  Today:     %s tokens\n", formatTokens(summary.TodayTokens))
		fmt.Printf("  This week: %s tokens\n", formatTokens(summary.WeekTokens))
		fmt.Printf("  Avg/day:   %s tokens\n", formatTokens(summary.AvgPerDay))
		fmt.Printf("  Calls:     %d (7d)\n", summary.TotalCalls)
		fmt.Println()

		if len(summary.BySource) > 0 {
			fmt.Println("  By source (7d):")
			sources := make([]string, 0, len(summary.BySource))
			for src := range summary.BySource {
				sources = append(sources, src)
			}
			sort.Strings(sources)
			for _, src := range sources {
				fmt.Printf("    %-8s %s tokens\n", src, formatTokens(summary.BySource[src]))
			}
		}

		daily, err := s.Tokens.DailyStats(7)
		if err == nil && len(daily) > 0 {
			fmt.Println()
			fmt.Println("  Daily breakdown:")
			for _, d := range daily {
				total := d.Agent + d.Chat
				if total > 0 {
					fmt.Printf("    %s  agent: %s  chat: %s\n", d.Date, formatTokens(d.Agent), formatTokens(d.Chat))
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
