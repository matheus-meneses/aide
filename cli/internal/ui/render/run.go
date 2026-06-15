package render

import (
	"aide/cli/internal/runtime/runner"
	"fmt"
)

func printRunSummaryTable(result *runner.RunResult) {
	fprintf("\n Run %s\n", result.RunID[:8])
	fprintf(" Sources: %d total, %d ok, %d failed\n\n", result.SourcesTotal, result.SourcesOK, result.SourcesFailed)

	if len(result.Results) == 0 {
		return
	}

	w := newTabWriter()
	fmt.Fprintf(w, "  SOURCE\tSTATUS\tENTRIES\tNEW\tDURATION\n")
	fmt.Fprintf(w, "  ------\t------\t-------\t---\t--------\n")

	var failures []string
	for _, r := range result.Results {
		status := "OK"
		entries := fmt.Sprintf("%d", len(r.Entries))
		newCol := fmt.Sprintf("+%d", r.NewItems)
		if r.Error != nil {
			status = "FAIL"
			entries = "-"
			newCol = "-"
			failures = append(failures, fmt.Sprintf("  [%s] %s", r.Source, r.Error))
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%dms\n", r.Source, status, entries, newCol, r.DurationMs)
	}

	w.Flush()

	if len(failures) > 0 {
		fprintf("\n  Errors:\n")
		for _, f := range failures {
			fprintln(f)
		}
	}
	fprintln()
}
