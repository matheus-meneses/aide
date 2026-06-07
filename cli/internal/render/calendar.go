package render

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"aide/cli/internal/store"
)

func init() {
	registerSource("outlook", &outlookPlugin{})
}

type outlookPlugin struct{}

func (o *outlookPlugin) Classify(item store.Item) string {
	if strings.HasPrefix(item.Title, "Meeting:") {
		return "Calendar"
	}
	return "Inbox"
}

func (o *outlookPlugin) RenderSection(heading string, items []store.Item) {
	if heading == "Calendar" {
		renderCalendar(items)
	} else {
		(&defaultPlugin{}).RenderSection(heading, items)
	}
}

func renderCalendar(items []store.Item) {
	today := time.Now().Format("2006-01-02")

	filtered := items[:0]
	for _, item := range items {
		if item.EntryDate >= today {
			filtered = append(filtered, item)
		}
	}
	items = filtered

	sort.Slice(items, func(i, j int) bool {
		if items[i].EntryDate != items[j].EntryDate {
			return items[i].EntryDate < items[j].EntryDate
		}
		return items[i].Detail < items[j].Detail
	})

	type dayGroup struct {
		date  string
		items []store.Item
	}
	var groups []dayGroup
	groupMap := make(map[string]int)

	hasToday := false
	for _, item := range items {
		d := item.EntryDate
		if d == today {
			hasToday = true
		}
		if idx, ok := groupMap[d]; ok {
			groups[idx].items = append(groups[idx].items, item)
		} else {
			groupMap[d] = len(groups)
			groups = append(groups, dayGroup{date: d, items: []store.Item{item}})
		}
	}

	if !hasToday {
		todayGroup := dayGroup{date: today, items: nil}
		groups = append([]dayGroup{todayGroup}, groups...)
	}

	for _, g := range groups {
		label := formatDayLabel(g.date, today)
		fprintf(" │\n")
		fprintf(" │  ── %s ─────────────────────────────────────────\n", label)

		if len(g.items) == 0 {
			fprintf(" │    (no meetings)\n")
			continue
		}

		conflicts := detectConflicts(g.items)

		for i, item := range g.items {
			title := stripPrefix(item.Title)
			if len(title) > 42 {
				title = title[:39] + "..."
			}
			member := item.Member
			if len(member) > 22 {
				member = member[:19] + "..."
			}
			detail := item.Detail
			marker := ""
			if conflicts[i] {
				marker = " [!]"
			}
			padding := 42 - len([]rune(title))
			if padding < 0 {
				padding = 0
			}
			displayTitle := title
			if item.Link != "" {
				displayTitle = hyperlink(item.Link, title)
			}
			fprintf(" │    %s%s  %-22s  %s%s\n", displayTitle, strings.Repeat(" ", padding), member, detail, marker)
		}
	}
}

func formatDayLabel(dateStr, today string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	dayName := t.Format("Mon")
	datePart := t.Format("02/01")
	if dateStr == today {
		return fmt.Sprintf("Today (%s %s)", dayName, datePart)
	}
	return fmt.Sprintf("%s %s", dayName, datePart)
}

func detectConflicts(items []store.Item) map[int]bool {
	conflicts := make(map[int]bool)
	type interval struct {
		start int
		end   int
	}

	intervals := make([]interval, len(items))
	for i, item := range items {
		startMin, duration := parseTimeDetail(item.Detail)
		if startMin >= 0 {
			intervals[i] = interval{start: startMin, end: startMin + duration}
		} else {
			intervals[i] = interval{start: -1, end: -1}
		}
	}

	for i := 0; i < len(intervals); i++ {
		if intervals[i].start < 0 {
			continue
		}
		for j := i + 1; j < len(intervals); j++ {
			if intervals[j].start < 0 {
				continue
			}
			if intervals[i].start < intervals[j].end && intervals[j].start < intervals[i].end {
				conflicts[i] = true
				conflicts[j] = true
			}
		}
	}
	return conflicts
}

func parseTimeDetail(detail string) (startMinutes int, durationMinutes int) {
	if len(detail) < 5 {
		return -1, 0
	}

	timePart := detail
	durPart := ""
	if idx := strings.Index(detail, "("); idx > 0 {
		timePart = strings.TrimSpace(detail[:idx])
		endIdx := strings.Index(detail, ")")
		if endIdx > idx {
			durPart = detail[idx+1 : endIdx]
		}
	}

	parts := strings.Split(timePart, ":")
	if len(parts) != 2 {
		return -1, 0
	}
	h := 0
	m := 0
	fmt.Sscanf(parts[0], "%d", &h)
	fmt.Sscanf(parts[1], "%d", &m)
	startMinutes = h*60 + m

	durationMinutes = 0
	if strings.Contains(durPart, "h") {
		var dh, dm int
		fmt.Sscanf(durPart, "%dh%dm", &dh, &dm)
		durationMinutes = dh*60 + dm
	} else if strings.HasSuffix(durPart, "m") {
		var dm int
		fmt.Sscanf(durPart, "%dm", &dm)
		durationMinutes = dm
	}
	if durationMinutes == 0 {
		durationMinutes = 30
	}

	return startMinutes, durationMinutes
}
