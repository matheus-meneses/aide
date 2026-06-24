package tools

import (
	"aide/cli/internal/agent/events"
	"aide/cli/internal/persistence/store"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/plugin"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ScrapeResult struct {
	SourcesTotal  int
	SourcesOK     int
	SourcesFailed int
}

// Capabilities is the subset of agent behaviour the built-in tools depend on.
// The agent core implements it, keeping this package free of a cyclic import.
type Capabilities interface {
	Scrape(ctx context.Context, sources []string) (ScrapeResult, error)
	LastRun() time.Time
	Store() *store.Store
	Config() *config.Config
	Bus() *events.EventBus
	PostMessage(content, timestamp string)
	NotifyOS(title, body string)
}

func sha256Sum(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:8]
}

func RegisterBuiltins(reg *ToolRegistry, c Capabilities) {
	mgr := plugin.NewManager()
	cfg := c.Config()

	sourceNames := make([]string, 0, len(cfg.Sources))
	for name, src := range cfg.Sources {
		if src.Enabled {
			sourceNames = append(sourceNames, name)
		}
	}

	sourceParam := "optional, scrape all if omitted"
	if len(sourceNames) > 0 {
		sourceParam = fmt.Sprintf("optional, one of: %s. Omit to scrape all.", strings.Join(sourceNames, ", "))
	}

	reg.Register(&Tool{
		Name:        "scrape",
		Description: "Run scrapers to fetch fresh data from sources.",
		Parameters:  fmt.Sprintf(`{"source": "%s"}`, sourceParam),
		InputSchema: objectSchema(map[string]string{"source": sourceParam}),
		Execute: func(ctx context.Context, params map[string]string) (string, error) {
			var sources []string
			if s := params["source"]; s != "" {
				sources = []string{s}
			}
			result, err := c.Scrape(ctx, sources)
			if err != nil {
				return "", err
			}
			msg := fmt.Sprintf("Scraped %d sources: %d OK, %d failed.", result.SourcesTotal, result.SourcesOK, result.SourcesFailed)
			if result.SourcesFailed > 0 {
				msg += " Some sources failed; data may be incomplete."
			}
			if bus := c.Bus(); bus != nil {
				bus.Publish(events.Event{
					Type:     "scrape_complete",
					Priority: "silent",
					Data:     fmt.Sprintf(`{"sources_total":%d,"sources_ok":%d,"sources_failed":%d}`, result.SourcesTotal, result.SourcesOK, result.SourcesFailed),
				})
			}
			return msg, nil
		},
	})

	for sourceName, src := range cfg.Sources {
		if !src.Enabled {
			continue
		}
		pluginName := src.Plugin
		if pluginName == "" {
			pluginName = sourceName
		}
		m, err := mgr.Get(pluginName)
		if err != nil {
			continue
		}
		for _, toolSpec := range m.Tools {
			sourceName := sourceName
			m := m
			src := src
			toolSpec := toolSpec
			paramsJSON, _ := json.Marshal(toolSpec.Params)

			toolName := toolSpec.Name
			if _, exists := reg.Get(toolName); exists {
				toolName = sourceName + "_" + toolSpec.Name
			}

			reg.Register(&Tool{
				Name:        toolName,
				Description: toolSpec.Description,
				Parameters:  string(paramsJSON),
				InputSchema: objectSchema(toolSpec.Params),
				Execute: func(ctx context.Context, params map[string]string) (string, error) {
					secrets, _ := plugin.ScopedSecrets(sourceName, m)
					paramAny := make(map[string]any, len(params))
					for k, v := range params {
						paramAny[k] = v
					}
					req := &plugin.Request{
						Action:  "query",
						Name:    toolSpec.Name,
						Params:  paramAny,
						Secrets: secrets,
						Config:  src.Config,
					}
					resp, _, err := plugin.Execute(ctx, m, req)
					if err != nil {
						return "", err
					}
					if !resp.OK {
						return "", fmt.Errorf("%s", resp.Error)
					}
					return resp.Text, nil
				},
			})
		}
	}

	reg.Register(&Tool{
		Name:        "diff",
		Description: "Check what changed since the last cycle. Shows new and resolved items.",
		Parameters:  `{}`,
		Execute: func(_ context.Context, _ map[string]string) (string, error) {
			since := c.LastRun()
			if since.IsZero() {
				since = time.Now().Add(-30 * time.Minute)
			}
			d, err := ComputeDiff(c.Store(), since)
			if err != nil {
				return "", err
			}
			if len(d.NewItems) == 0 && len(d.ResolvedItems) == 0 {
				return "No changes detected.", nil
			}
			var b strings.Builder
			if len(d.NewItems) > 0 {
				fmt.Fprintf(&b, "NEW (%d):\n", len(d.NewItems))
				for _, item := range d.NewItems {
					b.WriteString("  - " + formatToolItem(item) + "\n")
				}
			}
			if len(d.ResolvedItems) > 0 {
				fmt.Fprintf(&b, "RESOLVED (%d):\n", len(d.ResolvedItems))
				for _, item := range d.ResolvedItems {
					b.WriteString("  - " + formatToolItem(item) + "\n")
				}
			}
			return b.String(), nil
		},
	})

	reg.Register(&Tool{
		Name:        "notify_user",
		Description: "Send a browser notification to the user. Use ONLY for urgent/important things that need immediate attention.",
		Parameters:  `{"title": "required, 3-5 words", "body": "required, max 12 words", "fingerprint": "optional, item fingerprint for ack tracking"}`,
		InputSchema: objectSchema(map[string]string{
			"title":       "required, 3-5 words",
			"body":        "required, max 12 words",
			"fingerprint": "optional, item fingerprint for ack tracking",
		}),
		Execute: func(_ context.Context, params map[string]string) (string, error) {
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

			if bus := c.Bus(); bus != nil {
				bus.Publish(events.Event{
					Type:     "notification",
					Priority: "urgent",
					Data:     fmt.Sprintf(`{"title":%q,"body":%q,"fingerprint":%q}`, title, body, fp),
				})
			}

			c.NotifyOS(title, body)

			now := time.Now().UTC().Format(time.RFC3339)
			chatContent := fmt.Sprintf("**%s**\n\n%s\n\n---\n_Notified at %s_",
				title, body, time.Now().Format("15:04"))
			c.PostMessage(chatContent, now)

			return "Notification sent.", nil
		},
	})

	reg.Register(&Tool{
		Name:        "send_message",
		Description: "Post a message to the web UI activity feed. Use for non-urgent updates the user might want to see later.",
		Parameters:  `{"content": "required, the message to display", "fingerprint": "optional, item fingerprint for ack tracking"}`,
		InputSchema: objectSchema(map[string]string{
			"content":     "required, the message to display",
			"fingerprint": "optional, item fingerprint for ack tracking",
		}),
		Execute: func(_ context.Context, params map[string]string) (string, error) {
			content := params["content"]
			fingerprint := params["fingerprint"]
			if content == "" {
				return "", fmt.Errorf("content is required")
			}

			fp := fingerprint
			if fp == "" {
				fp = fmt.Sprintf("%x", sha256Sum(content))
			}

			if bus := c.Bus(); bus != nil {
				bus.Publish(events.Event{
					Type:     "notification",
					Priority: "normal",
					Data:     fmt.Sprintf(`{"title":"Aide","body":%q,"fingerprint":%q}`, content, fp),
				})
			}

			now := time.Now().UTC().Format(time.RFC3339)
			c.PostMessage(content, now)

			return "Message posted to activity feed.", nil
		},
	})

	reg.Register(&Tool{
		Name:        "check_items",
		Description: "Query current open items. Optionally filter by source.",
		Parameters:  `{"source": "optional, filter by source name"}`,
		InputSchema: objectSchema(map[string]string{"source": "optional, filter by source name"}),
		Execute: func(_ context.Context, params map[string]string) (string, error) {
			source := params["source"]
			items, err := c.Store().Items.QueryOpen(source, "", "")
			if err != nil {
				return "", err
			}
			if len(items) == 0 {
				return "No open items.", nil
			}
			var b strings.Builder
			fmt.Fprintf(&b, "%d open items:\n", len(items))
			limit := 15
			if len(items) < limit {
				limit = len(items)
			}
			for _, item := range items[:limit] {
				b.WriteString("  - " + formatToolItem(item) + "\n")
			}
			if len(items) > 15 {
				fmt.Fprintf(&b, "  ... and %d more\n", len(items)-15)
			}
			return b.String(), nil
		},
	})

	reg.Register(&Tool{
		Name:        "check_today",
		Description: "Get today's calendar events/meetings.",
		Parameters:  `{}`,
		Execute: func(_ context.Context, _ map[string]string) (string, error) {
			todayEvents, err := c.Store().Items.TodayEvents()
			if err != nil {
				return "", err
			}
			if len(todayEvents) == 0 {
				return "No meetings today.", nil
			}
			var b strings.Builder
			fmt.Fprintf(&b, "%d meetings today:\n", len(todayEvents))
			for _, ev := range todayEvents {
				fmt.Fprintf(&b, "  - %s %s\n", ev.Detail, ev.Title)
			}
			return b.String(), nil
		},
	})

	reg.Register(&Tool{
		Name:        "check_health",
		Description: "Check the health status of all data sources (last run time, errors).",
		Parameters:  `{}`,
		Execute: func(_ context.Context, _ map[string]string) (string, error) {
			health, err := c.Store().Runs.AllHealth()
			if err != nil {
				return "", err
			}
			if len(health) == 0 {
				return "No health data available. Sources have never been scraped.", nil
			}
			var b strings.Builder
			for _, h := range health {
				fmt.Fprintf(&b, "  - %s: %s (last: %s, entries: %d)\n", h.Source, h.Status, h.LastRun, h.EntriesCount)
			}
			return b.String(), nil
		},
	})

	reg.Register(&Tool{
		Name:        "done",
		Description: "Stop acting for this cycle. Use when there is nothing else to do.",
		Parameters:  `{"reason": "optional, why you are stopping"}`,
		InputSchema: objectSchema(map[string]string{"reason": "optional, why you are stopping"}),
		Execute: func(_ context.Context, _ map[string]string) (string, error) {
			return "cycle complete", nil
		},
	})
}

func formatToolItem(item store.Item) string {
	line := fmt.Sprintf("[%s/%s] %s", item.Source, item.Category, item.Title)
	if item.EntryDate != "" {
		line += " (" + HumanizeDate(item.EntryDate) + ")"
	}
	if item.Detail != "" {
		line += " | " + item.Detail
	}
	if item.Link != "" {
		line += " | link: " + item.Link
	}
	return line
}

// HumanizeDate renders an ISO date (YYYY-MM-DD) relative to today (TODAY,
// TOMORROW, "in N days", etc.). It is the single shared implementation used by
// both the tool output and the agent's item formatting.
func HumanizeDate(dateStr string) string {
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
