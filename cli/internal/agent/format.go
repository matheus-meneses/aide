package agent

import (
	"aide/cli/internal/agent/tools"
	"aide/cli/internal/persistence/store"
	"fmt"
)

func formatItem(item store.Item) string {
	title := sanitizeUntrusted(item.Title)
	detail := sanitizeUntrusted(item.Detail)
	member := sanitizeUntrusted(item.Member)

	line := fmt.Sprintf("- [%s] %s", item.Priority, title)
	if member != "" {
		line += " (assigned: " + member + ")"
	}
	if item.EntryDate != "" {
		dateLabel := tools.HumanizeDate(item.EntryDate)
		line += " | " + dateLabel
		if item.Category == "event" && dateLabel != "TODAY" && detail != "" {
			line += " " + detail
		}
	}
	if item.Category != "event" || tools.HumanizeDate(item.EntryDate) == "TODAY" {
		if detail != "" {
			line += " | " + detail
		}
	}
	if item.Link != "" {
		line += " | link: " + sanitizeUntrusted(item.Link)
	}
	return line + "\n"
}
