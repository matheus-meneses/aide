package main

import (
	"aide/cli/internal/persistence/store"
	"aide/cli/internal/platform/config"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var pruneDry bool

var pruneCmd = &cobra.Command{
	Use:   "prune [days]",
	Short: "Delete old data, keeping N days of history (default 7)",
	Long:  "Removes resolved items, old messages, metrics, and other data older than N days (default 7). Future events are always kept.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  pruneExecute,
}

func init() {
	pruneCmd.Flags().BoolVar(&pruneDry, "dry", false, "Show what would be deleted without deleting")
	rootCmd.AddCommand(pruneCmd)
}

func printPruneResult(header string, result *store.PruneResult) {
	fmt.Println(header)
	fmt.Printf("  Items:         %d\n", result.Items)
	fmt.Printf("  Messages:      %d\n", result.Messages)
	fmt.Printf("  Sessions:      %d\n", result.Sessions)
	fmt.Printf("  Memories:      %d\n", result.Memories)
	fmt.Printf("  Metrics:       %d\n", result.Metrics)
	fmt.Printf("  Runs:          %d\n", result.Runs)
	fmt.Printf("  Acks:          %d\n", result.Acks)
	fmt.Printf("  Token records: %d\n", result.Tokens)
}

func pruneTotal(result *store.PruneResult) int64 {
	return result.Items + result.Messages + result.Sessions + result.Memories +
		result.Metrics + result.Runs + result.Acks + result.Tokens
}

func pruneExecute(_ *cobra.Command, args []string) error {
	days := 7
	if len(args) > 0 {
		d, err := strconv.Atoi(args[0])
		if err != nil || d < 1 {
			return fmt.Errorf("days must be a positive integer")
		}
		days = d
	}

	return withStore(func(_ *config.Config, s *store.Store) error {
		counts, err := s.Maintenance.PruneCounts(days)
		if err != nil {
			return fmt.Errorf("computing prune counts: %w", err)
		}

		if pruneDry {
			printPruneResult(fmt.Sprintf("Would prune (keeping %d days, dry run):", days), counts)
			return nil
		}

		total := pruneTotal(counts)
		if total == 0 {
			fmt.Printf("Nothing to prune (keeping %d days).\n", days)
			return nil
		}

		printPruneResult(fmt.Sprintf("About to prune (keeping %d days):", days), counts)
		if err := requireConfirm(fmt.Sprintf("Delete %d records?", total)); err != nil {
			return err
		}

		result, err := s.Maintenance.Prune(days)
		if err != nil {
			return fmt.Errorf("prune failed: %w", err)
		}

		printPruneResult(fmt.Sprintf("Pruned (kept %d days):", days), result)
		return nil
	})
}
