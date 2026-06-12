package render

import (
	"aide/cli/internal/config"
	"aide/cli/internal/store"
	"fmt"
	"sort"
)

func printSourcesTable(cfg *config.Config, health []store.SourceHealth) {
	fprintf("\n Sources\n\n")

	w := newTabWriter()
	fmt.Fprintf(w, "  SOURCE\tENABLED\tSTATUS\tLAST RUN\tENTRIES\tDURATION\n")
	fmt.Fprintf(w, "  ------\t-------\t------\t--------\t-------\t--------\n")

	healthMap := make(map[string]store.SourceHealth)
	for _, h := range health {
		healthMap[h.Source] = h
	}

	names := make([]string, 0, len(cfg.Sources))
	for name := range cfg.Sources {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		src := cfg.Sources[name]
		enabled := "no"
		if src.Enabled {
			enabled = "yes"
		}

		status := "-"
		lastRun := "-"
		entries := "-"
		duration := "-"

		if h, ok := healthMap[name]; ok {
			status = h.Status
			if h.LastRun != "" && len(h.LastRun) >= 19 {
				lastRun = h.LastRun[:19]
			}
			entries = fmt.Sprintf("%d", h.EntriesCount)
			duration = fmt.Sprintf("%dms", h.DurationMs)
		}

		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\n", name, enabled, status, lastRun, entries, duration)
	}

	w.Flush()
	fprintln()
}
