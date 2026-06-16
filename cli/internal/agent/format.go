package agent

import (
	"aide/cli/internal/agent/tools"
	"aide/cli/internal/persistence/store"
	"fmt"
)

func formatItem(item store.Item) string {
	line := fmt.Sprintf("- [%s] %s", item.Priority, item.Title)
	if item.Member != "" {
		line += " (assigned: " + item.Member + ")"
	}
	if item.EntryDate != "" {
		dateLabel := tools.HumanizeDate(item.EntryDate)
		line += " | " + dateLabel
		if item.Category == "event" && dateLabel != "TODAY" && item.Detail != "" {
			line += " " + item.Detail
		}
	}
	if item.Category != "event" || tools.HumanizeDate(item.EntryDate) == "TODAY" {
		if item.Detail != "" {
			line += " | " + item.Detail
		}
	}
	if item.Link != "" {
		line += " | link: " + item.Link
	}
	return line + "\n"
}
