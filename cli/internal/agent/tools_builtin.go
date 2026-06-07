package agent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
)

func sha256Sum(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:8]
}

func (a *Agent) registerDefaultTools() {
	a.tools = NewToolRegistry()

	a.tools.Register(&Tool{
		Name:        "scrape",
		Description: "Run scrapers to fetch fresh data from sources.",
		Parameters:  `{"source": "optional, one of: outlook, jira, gitlab, sailpoint, rh_management_portal. Omit to scrape all."}`,
		Execute: func(ctx context.Context, params map[string]string) (string, error) {
			var sources []string
			if s := params["source"]; s != "" {
				sources = []string{s}
			}
			result, err := a.runScrape(ctx, sources)
			if err != nil {
				return "", err
			}
			msg := fmt.Sprintf("Scraped %d sources: %d OK, %d failed.", result.SourcesTotal, result.SourcesOK, result.SourcesFailed)
			if result.SourcesFailed > 0 {
				msg += " Some sources failed; data may be incomplete."
			}
			if a.bus != nil {
				a.bus.Publish(Event{
					Type:     "scrape_complete",
					Priority: "silent",
					Data:     fmt.Sprintf(`{"sources_total":%d,"sources_ok":%d,"sources_failed":%d}`, result.SourcesTotal, result.SourcesOK, result.SourcesFailed),
				})
			}
			return msg, nil
		},
	})

	a.tools.Register(&Tool{
		Name:        "diff",
		Description: "Check what changed since the last cycle. Shows new and resolved items.",
		Parameters:  `{}`,
		Execute: func(ctx context.Context, params map[string]string) (string, error) {
			since := a.getLastRun()
			if since.IsZero() {
				since = time.Now().Add(-30 * time.Minute)
			}
			d, err := ComputeDiff(a.store, since)
			if err != nil {
				return "", err
			}
			if len(d.NewItems) == 0 && len(d.ResolvedItems) == 0 {
				return "No changes detected.", nil
			}
			var b strings.Builder
			if len(d.NewItems) > 0 {
				b.WriteString(fmt.Sprintf("NEW (%d):\n", len(d.NewItems)))
				for _, item := range d.NewItems {
					b.WriteString("  - " + formatToolItem(item) + "\n")
				}
			}
			if len(d.ResolvedItems) > 0 {
				b.WriteString(fmt.Sprintf("RESOLVED (%d):\n", len(d.ResolvedItems)))
				for _, item := range d.ResolvedItems {
					b.WriteString("  - " + formatToolItem(item) + "\n")
				}
			}
			return b.String(), nil
		},
	})

	a.tools.Register(&Tool{
		Name:        "notify_user",
		Description: "Send a browser notification to the user. Use ONLY for urgent/important things that need immediate attention.",
		Parameters:  `{"title": "required, 3-5 words", "body": "required, max 12 words", "fingerprint": "optional, item fingerprint for ack tracking"}`,
		Execute: func(ctx context.Context, params map[string]string) (string, error) {
			title := params["title"]
			body := params["body"]
			fingerprint := params["fingerprint"]
			if title == "" || body == "" {
				return "", fmt.Errorf("title and body are required")
			}

			fp := fingerprint
			if fp == "" {
				fp = fmt.Sprintf("%x", sha256Sum(title+body))
			}

			if a.bus != nil {
				a.bus.Publish(Event{
					Type:     "notification",
					Priority: "urgent",
					Data:     fmt.Sprintf(`{"title":%q,"body":%q,"fingerprint":%q}`, title, body, fp),
				})
			}

			now := time.Now().UTC().Format(time.RFC3339)
			chatContent := fmt.Sprintf("**%s**\n\n%s\n\n---\n_Notified at %s_",
				title, body, time.Now().Format("15:04"))
			a.postToChatAndSSE(chatContent, now)

			return "Notification sent.", nil
		},
	})

	a.tools.Register(&Tool{
		Name:        "send_message",
		Description: "Post a message to the web UI activity feed. Use for non-urgent updates the user might want to see later.",
		Parameters:  `{"content": "required, the message to display", "fingerprint": "optional, item fingerprint for ack tracking"}`,
		Execute: func(ctx context.Context, params map[string]string) (string, error) {
			content := params["content"]
			fingerprint := params["fingerprint"]
			if content == "" {
				return "", fmt.Errorf("content is required")
			}

			fp := fingerprint
			if fp == "" {
				fp = fmt.Sprintf("%x", sha256Sum(content))
			}

			if a.bus != nil {
				a.bus.Publish(Event{
					Type:     "notification",
					Priority: "normal",
					Data:     fmt.Sprintf(`{"title":"Aide","body":%q,"fingerprint":%q}`, content, fp),
				})
			}

			now := time.Now().UTC().Format(time.RFC3339)
			a.postToChatAndSSE(content, now)

			return "Message posted to activity feed.", nil
		},
	})

	a.tools.Register(&Tool{
		Name:        "check_items",
		Description: "Query current open items. Optionally filter by source.",
		Parameters:  `{"source": "optional, filter by source name"}`,
		Execute: func(ctx context.Context, params map[string]string) (string, error) {
			source := params["source"]
			items, err := a.store.Items.QueryOpen(source, "", "")
			if err != nil {
				return "", err
			}
			if len(items) == 0 {
				return "No open items.", nil
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("%d open items:\n", len(items)))
			limit := 15
			if len(items) < limit {
				limit = len(items)
			}
			for _, item := range items[:limit] {
				b.WriteString("  - " + formatToolItem(item) + "\n")
			}
			if len(items) > 15 {
				b.WriteString(fmt.Sprintf("  ... and %d more\n", len(items)-15))
			}
			return b.String(), nil
		},
	})

	a.tools.Register(&Tool{
		Name:        "check_today",
		Description: "Get today's calendar events/meetings.",
		Parameters:  `{}`,
		Execute: func(ctx context.Context, params map[string]string) (string, error) {
			events, err := a.store.Items.TodayEvents()
			if err != nil {
				return "", err
			}
			if len(events) == 0 {
				return "No meetings today.", nil
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("%d meetings today:\n", len(events)))
			for _, ev := range events {
				b.WriteString(fmt.Sprintf("  - %s %s\n", ev.Detail, ev.Title))
			}
			return b.String(), nil
		},
	})

	a.tools.Register(&Tool{
		Name:        "check_health",
		Description: "Check the health status of all data sources (last run time, errors).",
		Parameters:  `{}`,
		Execute: func(ctx context.Context, params map[string]string) (string, error) {
			health, err := a.store.Runs.AllHealth()
			if err != nil {
				return "", err
			}
			if len(health) == 0 {
				return "No health data available. Sources have never been scraped.", nil
			}
			var b strings.Builder
			for _, h := range health {
				b.WriteString(fmt.Sprintf("  - %s: %s (last: %s, entries: %d)\n", h.Source, h.Status, h.LastRun, h.EntriesCount))
			}
			return b.String(), nil
		},
	})

	a.tools.Register(&Tool{
		Name:        "done",
		Description: "Stop acting for this cycle. Use when there is nothing else to do.",
		Parameters:  `{"reason": "optional, why you are stopping"}`,
		Execute: func(ctx context.Context, params map[string]string) (string, error) {
			return "cycle complete", nil
		},
	})
}
