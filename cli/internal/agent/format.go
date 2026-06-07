package agent

import (
	"fmt"
	"time"

	"aide/cli/internal/store"
)

func formatItem(item store.Item) string {
	line := fmt.Sprintf("- [%s] %s", item.Priority, item.Title)
	if item.Member != "" {
		line += " (assigned: " + item.Member + ")"
	}
	if item.EntryDate != "" {
		dateLabel := humanizeDate(item.EntryDate)
		line += " | " + dateLabel
		if item.Category == "event" && dateLabel != "TODAY" && item.Detail != "" {
			line += " " + item.Detail
		}
	}
	if item.Category != "event" || humanizeDate(item.EntryDate) == "TODAY" {
		if item.Detail != "" {
			line += " | " + item.Detail
		}
	}
	if item.Link != "" {
		line += " | link: " + item.Link
	}
	return line + "\n"
}

func formatToolItem(item store.Item) string {
	line := fmt.Sprintf("[%s/%s] %s", item.Source, item.Category, item.Title)
	if item.EntryDate != "" {
		line += " (" + humanizeDate(item.EntryDate) + ")"
	}
	if item.Detail != "" {
		line += " | " + item.Detail
	}
	if item.Link != "" {
		line += " | link: " + item.Link
	}
	return line
}

func humanizeDate(dateStr string) string {
	t, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		return dateStr
	}

	now := time.Now()
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	eventMidnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	days := int(eventMidnight.Sub(todayMidnight).Hours() / 24)

	switch {
	case days == 0:
		return "TODAY"
	case days == 1:
		return "TOMORROW"
	case days == -1:
		return "YESTERDAY (" + t.Format("Mon Jan 2") + ")"
	case days > 1:
		return fmt.Sprintf("in %d days (%s)", days, t.Format("Mon Jan 2"))
	default:
		return fmt.Sprintf("%d days ago (%s)", -days, t.Format("Mon Jan 2"))
	}
}
