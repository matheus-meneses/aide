package agent

import (
	"aide/cli/internal/agent/events"
	"context"
	"fmt"
	"strings"
	"time"
)

const maxActionsPerCycle = 10

type agentState struct {
	Today              string         `json:"today"`
	Time               string         `json:"time"`
	LastScrape         string         `json:"last_scrape"`
	MinutesSinceScrape int            `json:"minutes_since_scrape"`
	ItemCounts         map[string]int `json:"item_counts"`
	TodayEvents        int            `json:"today_events"`
	RecentActions      []string       `json:"recent_actions,omitempty"`
}

func (a *Agent) loadMemory() {
	mem, err := a.store.Memory.LoadLast()
	if err != nil {
		alog.Debug("no previous memory found, starting fresh")
		return
	}

	a.setLastMemory(mem.Content)
	if mem.LastScrapeAt != "" {
		t, err := time.Parse(time.RFC3339, mem.LastScrapeAt)
		if err == nil {
			a.setLastRun(t)
			alog.Debug("restored last scrape time: %s", t.Format("15:04"))
		}
	}
	alog.Debug("loaded memory: %s", mem.Content)
}

func (a *Agent) runAgentCycle(ctx context.Context) {
	if err := a.store.Acks.Prune(); err != nil {
		alog.Warn("failed to prune acks: %v", err)
	}
	state := a.observeState()
	var history []string

	for i := 0; i < maxActionsPerCycle; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		call := a.think(ctx, state, history)
		if call.Tool == "" || call.Tool == "done" {
			if call.Reason != "" {
				alog.Debug("done: %s", call.Reason)
			}
			break
		}

		alog.Info("action: %s(%v) — %s", call.Tool, call.Params, call.Reason)

		result, err := a.executeTool(ctx, call.Tool, call.Params)
		if err != nil {
			entry := fmt.Sprintf("Called %s -> ERROR: %v", call.Tool, err)
			history = append(history, entry)
			alog.Error("tool error: %v", err)
			if a.bus != nil {
				a.bus.Publish(events.Event{
					Type:     "cycle_error",
					Priority: "silent",
					Data:     fmt.Sprintf(`{"tool":%q,"error":%q}`, call.Tool, err.Error()),
				})
			}
			continue
		}

		entry := fmt.Sprintf("Called %s -> %s", call.Tool, result)
		history = append(history, entry)
	}

	a.saveMemory(history)
}

func (a *Agent) saveMemory(history []string) {
	if len(history) == 0 {
		history = []string{"No actions taken"}
	}

	summary := fmt.Sprintf(
		"Cycle at %s | %s",
		time.Now().Format("Monday 15:04"),
		strings.Join(history, " | "),
	)

	lastRun := a.getLastRun()
	lastScrape := ""
	if !lastRun.IsZero() {
		lastScrape = lastRun.UTC().Format(time.RFC3339)
	}

	if err := a.store.Memory.Save(lastScrape, summary); err != nil {
		alog.Warn("failed to save memory: %v", err)
		return
	}

	if err := a.store.Memory.Prune(5); err != nil {
		alog.Warn("failed to prune memories: %v", err)
	}
	a.setLastMemory(summary)
}

func (a *Agent) observeState() agentState {
	now := time.Now()
	state := agentState{
		Today: now.Format("Monday, 2006-01-02 (Mon Jan 2)"),
		Time:  now.Format("15:04"),
	}

	lastRun := a.getLastRun()
	if !lastRun.IsZero() {
		state.LastScrape = lastRun.Format("15:04")
		state.MinutesSinceScrape = int(time.Since(lastRun).Minutes())
	} else {
		state.LastScrape = "never"
		state.MinutesSinceScrape = 9999
	}

	if mem := a.getLastMemory(); mem != "" {
		state.RecentActions = []string{mem}
	}

	counts, err := a.store.Items.CountOpenBySource()
	if err == nil {
		state.ItemCounts = counts
	}

	events, err := a.store.Items.TodayEvents()
	if err == nil {
		state.TodayEvents = len(events)
	}

	return state
}
