package agent

import (
	"aide/cli/internal/agent/events"
	"aide/cli/internal/agent/llm"
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

	messages := a.buildAgentMessages(state)
	toolDefs := a.toolDefinitions()
	var history []string

	for i := 0; i < maxActionsPerCycle; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		result, err := a.think(ctx, messages, toolDefs)
		if err != nil {
			alog.Error("LLM error: %v", err)
			break
		}

		calls := result.ToolCalls
		fallback := false
		if len(calls) == 0 {
			if fb, ok := fallbackToolCall(result.Content); ok {
				calls = []llm.ToolCall{fb}
				fallback = true
			}
		}

		if len(calls) == 0 {
			if strings.TrimSpace(result.Content) != "" {
				alog.Debug("agent: %s", result.Content)
			}
			break
		}

		messages = appendAssistantTurn(messages, result.Content, calls, fallback)

		stop := false
		for _, call := range calls {
			if call.Name == "done" {
				if reason := argString(call.Arguments, "reason"); reason != "" {
					alog.Debug("done: %s", reason)
				}
				stop = true
				break
			}

			params := argsToParams(call.Arguments)
			alog.Info("action: %s(%v)", call.Name, params)

			out, execErr := a.executeTool(ctx, call.Name, params)
			if execErr != nil {
				history = append(history, fmt.Sprintf("Called %s -> ERROR: %v", call.Name, execErr))
				alog.Error("tool error: %v", execErr)
				if a.bus != nil {
					a.bus.Publish(events.Event{
						Type:     "cycle_error",
						Priority: "silent",
						Data:     fmt.Sprintf(`{"tool":%q,"error":%q}`, call.Name, execErr.Error()),
					})
				}
				messages = appendToolResult(messages, call, fallback, "ERROR: "+execErr.Error())
				continue
			}

			history = append(history, fmt.Sprintf("Called %s -> %s", call.Name, out))
			messages = appendToolResult(messages, call, fallback, out)
		}

		if stop {
			break
		}
	}

	a.saveMemory(history)
}

func appendAssistantTurn(messages []llm.ChatMessage, content string, calls []llm.ToolCall, fallback bool) []llm.ChatMessage {
	msg := llm.ChatMessage{Role: "assistant", Content: content}
	if !fallback {
		msg.ToolCalls = calls
	}
	return append(messages, msg)
}

func appendToolResult(messages []llm.ChatMessage, call llm.ToolCall, fallback bool, content string) []llm.ChatMessage {
	safe := fenceUntrusted(sanitizeUntrusted(content))
	if fallback {
		return append(messages, llm.ChatMessage{
			Role:    "user",
			Content: fmt.Sprintf("Result of %s:\n%s", call.Name, safe),
		})
	}
	return append(messages, llm.ChatMessage{
		Role:       "tool",
		ToolCallID: call.ID,
		Name:       call.Name,
		Content:    safe,
	})
}

func (a *Agent) saveMemory(history []string) {
	if len(history) == 0 {
		history = []string{"No actions taken"}
	}

	summary := fmt.Sprintf(
		"Cycle at %s | %s",
		a.clock.Now().Format("Monday 15:04"),
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
	now := a.clock.Now()
	state := agentState{
		Today: now.Format("Monday, 2006-01-02 (Mon Jan 2)"),
		Time:  now.Format("15:04"),
	}

	lastRun := a.getLastRun()
	if !lastRun.IsZero() {
		state.LastScrape = lastRun.Format("15:04")
		state.MinutesSinceScrape = int(now.Sub(lastRun).Minutes())
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
