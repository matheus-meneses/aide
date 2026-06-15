package agent

import (
	"aide/cli/internal/agent/llm"
	"aide/cli/internal/persistence/store"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	maxContextTokens = 24000
	tokensPerChar    = 4
)

func EstimateTokens(text string) int {
	return len(text) / tokensPerChar
}

func BuildContext(s *store.Store) (string, error) {
	var b strings.Builder

	profile, _ := s.Profile.All()
	if name := profile["preferred_name"]; name != "" {
		email := profile["email"]
		fmt.Fprintf(&b, "You are Aide, a personal work assistant for %s (%s).\n", name, email)
	} else {
		b.WriteString("You are Aide, a personal work assistant.\n")
	}
	b.WriteString("Today is " + time.Now().Format("Monday, January 2, 2006 15:04") + ".\n")
	b.WriteString("Below is the user's current operational data.\n\n")

	openItems, err := s.Items.QueryOpen("", "", "")
	if err != nil {
		return "", fmt.Errorf("querying open items: %w", err)
	}

	prioritized := prioritizeItems(openItems)
	budget := maxContextTokens - 2000

	grouped := make(map[string][]store.Item)
	usedTokens := EstimateTokens(b.String())

	for _, item := range prioritized {
		line := formatItem(item)
		lineTokens := EstimateTokens(line)
		if usedTokens+lineTokens > budget {
			break
		}
		usedTokens += lineTokens
		grouped[item.Source] = append(grouped[item.Source], item)
	}

	sourceOrder := []string{"outlook", "jira", "gitlab", "sailpoint", "rh_management_portal"}
	written := make(map[string]bool)

	for _, source := range sourceOrder {
		items, ok := grouped[source]
		if !ok {
			continue
		}
		written[source] = true
		fmt.Fprintf(&b, "## %s (%d items)\n", source, len(items))
		for _, item := range items {
			b.WriteString(formatItem(item))
		}
		b.WriteString("\n")
	}

	for source, items := range grouped {
		if written[source] {
			continue
		}
		fmt.Fprintf(&b, "## %s (%d items)\n", source, len(items))
		for _, item := range items {
			b.WriteString(formatItem(item))
		}
		b.WriteString("\n")
	}

	if len(prioritized) > len(flatGrouped(grouped)) {
		fmt.Fprintf(&b, "(%d items omitted due to context limits)\n\n", len(prioritized)-len(flatGrouped(grouped)))
	}

	health, err := s.Runs.AllHealth()
	if err == nil && len(health) > 0 {
		b.WriteString("## Source Health\n")
		for _, h := range health {
			fmt.Fprintf(&b, "- %s: %s (last run: %s, entries: %d)\n", h.Source, h.Status, h.LastRun, h.EntriesCount)
		}
		b.WriteString("\n")
	}

	metrics, err := s.Metrics.Latest("")
	if err == nil && len(metrics) > 0 {
		b.WriteString("## Latest Metrics\n")
		for _, m := range metrics {
			fmt.Fprintf(&b, "- %s/%s: %.0f\n", m.Source, m.Name, m.Value)
		}
		b.WriteString("\n")
	}

	counts, err := s.Items.CountOpenBySource()
	if err == nil && len(counts) > 0 {
		b.WriteString("## Summary\n")
		total := 0
		for source, count := range counts {
			fmt.Fprintf(&b, "- %s: %d open\n", source, count)
			total += count
		}
		fmt.Fprintf(&b, "- Total: %d open items\n", total)
		b.WriteString("\n")
	}

	if teamMembers, err := s.Team.All(); err == nil && len(teamMembers) > 0 {
		b.WriteString("## Your Team\n")
		byID := make(map[int64]store.Member, len(teamMembers))
		for _, m := range teamMembers {
			byID[m.ID] = m
		}
		for _, m := range teamMembers {
			line := fmt.Sprintf("- %s", m.Name)
			if m.Registration != "" {
				line += fmt.Sprintf(" (reg: %s)", m.Registration)
			}
			if m.Role != "" {
				line += fmt.Sprintf(", role: %s", m.Role)
			}
			if m.Department != "" {
				line += fmt.Sprintf(", dept: %s", m.Department)
			}
			if m.ManagerID != nil {
				if mgr, ok := byID[*m.ManagerID]; ok {
					line += fmt.Sprintf(", manager: %s", mgr.Name)
				}
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("IMPORTANT: All the user's current data is listed above. You already have everything you need. Do NOT say you will search, look up, or fetch anything. Answer directly and immediately from the data above.\n")
	b.WriteString("Be concise and actionable. When referencing items that have links, include the URL as a markdown link [title](url).\n")
	b.WriteString("If no relevant data exists for the user's question, say so clearly instead of pretending to search.\n")
	b.WriteString("CRITICAL DATE RULE: Only notify about meetings/events whose date label is TODAY. Items marked TOMORROW or any other future date are NOT happening now regardless of their time. Never say a future meeting is 'ongoing' or 'started'.\n")

	return b.String(), nil
}

func prioritizeItems(items []store.Item) []store.Item {
	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")

	type scored struct {
		item  store.Item
		score int
	}

	var scoredItems []scored
	for _, item := range items {
		s := 0

		switch item.EntryDate {
		case today:
			s += 100
		case tomorrow:
			s += 80
		}

		if item.Category == "event" {
			s += 50
		}

		switch item.Priority {
		case "critical":
			s += 40
		case "high":
			s += 30
		case "medium":
			s += 20
		case "low":
			s += 5
		default:
			s += 10
		}

		switch item.Source {
		case "outlook":
			s += 15
		case "jira":
			s += 10
		}

		scoredItems = append(scoredItems, scored{item: item, score: s})
	}

	sort.Slice(scoredItems, func(i, j int) bool {
		return scoredItems[i].score > scoredItems[j].score
	})

	result := make([]store.Item, len(scoredItems))
	for i, si := range scoredItems {
		result[i] = si.item
	}
	return result
}

func flatGrouped(grouped map[string][]store.Item) []store.Item {
	var all []store.Item
	for _, items := range grouped {
		all = append(all, items...)
	}
	return all
}

func TrimHistory(history []llm.ChatMessage, maxTokens int) []llm.ChatMessage {
	if len(history) <= 3 {
		return history
	}

	total := 0
	for _, m := range history {
		total += EstimateTokens(m.Content)
	}

	if total <= maxTokens {
		return history
	}

	system := history[0]
	messages := history[1:]

	keepLast := 6
	if keepLast > len(messages) {
		keepLast = len(messages)
	}

	kept := messages[len(messages)-keepLast:]
	dropped := messages[:len(messages)-keepLast]

	if len(dropped) == 0 {
		return history
	}

	var summary strings.Builder
	summary.WriteString("Previous conversation summary:\n")
	for _, m := range dropped {
		prefix := "User"
		if m.Role == "assistant" {
			prefix = "Assistant"
		}
		content := m.Content
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		fmt.Fprintf(&summary, "- %s: %s\n", prefix, content)
	}

	result := []llm.ChatMessage{system}
	result = append(result, llm.ChatMessage{Role: "system", Content: summary.String()})
	result = append(result, kept...)
	return result
}
