package render

import (
	"fmt"
	"time"

	"aide/cli/internal/store"
)

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	secs := int(d.Seconds())
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	return fmt.Sprintf("%dm%ds", secs/60, secs%60)
}

func printHistoryTable(runs []store.Run) {
	if len(runs) == 0 {
		fprintln("No run history.")
		return
	}

	fprintf("\n Run history (last %d)\n\n", len(runs))

	w := newTabWriter()
	fmt.Fprintf(w, "  RUN ID\tSTARTED\tDURATION\tOK\tFAILED\tTOTAL\n")
	fmt.Fprintf(w, "  ------\t-------\t--------\t--\t------\t-----\n")

	for _, r := range runs {
		id := r.ID
		if len(id) > 8 {
			id = id[:8]
		}
		started := r.StartedAt
		if len(started) > 19 {
			started = started[:19]
		}
		duration := "-"
		if r.FinishedAt != "" {
			startT, err1 := time.Parse(time.RFC3339, r.StartedAt)
			endT, err2 := time.Parse(time.RFC3339, r.FinishedAt)
			if err1 == nil && err2 == nil {
				duration = formatDuration(endT.Sub(startT))
			}
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%d\t%d\t%d\n", id, started, duration, r.SourcesOK, r.SourcesFailed, r.SourcesTotal)
	}

	w.Flush()
	fprintln()
}
