package render

import (
	"aide/cli/internal/store"
)

func printDiffReport(newItems, resolved []store.Item) {
	fprintf("\n Diff (last 24h)\n")

	if len(newItems) == 0 && len(resolved) == 0 {
		fprintln(" No changes detected.")
		fprintln()
		return
	}

	if len(newItems) > 0 {
		fprintf("\n + NEW (%d)\n", len(newItems))
		for _, item := range newItems {
			title := stripPrefix(item.Title)
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			fprintf("   [%s]  %-50s  %s\n", item.Source, title, item.Member)
		}
	}

	if len(resolved) > 0 {
		fprintf("\n - RESOLVED (%d)\n", len(resolved))
		for _, item := range resolved {
			title := stripPrefix(item.Title)
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			age := relativeAge(item.FirstSeenAt)
			fprintf("   [%s]  %-50s  %s  (lived %s)\n", item.Source, title, item.Member, age)
		}
	}

	fprintln()
}
