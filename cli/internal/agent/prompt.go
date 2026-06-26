package agent

import (
	"aide/cli/internal/agent/llm"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (a *Agent) buildAgentMessages(state agentState) []llm.ChatMessage {
	stateJSON, _ := json.Marshal(state)

	var prompt strings.Builder
	prompt.WriteString(untrustedDataGuardrail)
	prompt.WriteString("\n\n")
	prompt.WriteString(promptPrecedencePreamble)
	prompt.WriteString("\n\n")
	prompt.WriteString(agentRolePrompt)
	prompt.WriteString("\n\n")
	prompt.WriteString(agentCoreRules)
	prompt.WriteString("\n\n")
	prompt.WriteString(agentDefaultBehavior)

	if profile, err := a.store.Profile.All(); err == nil && len(profile) > 0 {
		fmt.Fprintf(&prompt, "\n\nYou are assisting %s (%s).", profile["preferred_name"], profile["email"])
	}

	if pc := a.promptContext(); pc.hasTrustedContext(true) {
		prompt.WriteString("\n\n")
		writeTrustedContext(&prompt, pc, true)
	}

	if mem := a.getLastMemory(); mem != "" {
		prompt.WriteString("\n\n## Previous Session\n")
		prompt.WriteString(sanitizeUntrusted(mem))
		prompt.WriteString("\n")
	}

	fmt.Fprintf(&prompt, "\n\n## TODAY IS %s, time now %s\n", state.Today, state.Time)
	prompt.WriteString("Items show dates in these same formats. An item whose date equals " + time.Now().Format("2006-01-02") + " (or is labeled TODAY) is happening today. Any other date is NOT today.\n")

	prompt.WriteString("\n## Current State\n")
	prompt.WriteString(fenceUntrusted(sanitizeUntrusted(string(stateJSON))))

	if acks, err := a.store.Acks.ListActive(); err == nil && len(acks) > 0 {
		openItems, _ := a.store.Items.QueryOpen("", "", "")
		todayEvents, _ := a.store.Items.TodayEvents()

		visibleFP := make(map[string]bool)
		for _, item := range openItems {
			visibleFP[item.Fingerprint] = true
		}
		for _, ev := range todayEvents {
			visibleFP[ev.Fingerprint] = true
		}

		var relevant []string
		for _, ack := range acks {
			if visibleFP[ack.Fingerprint] {
				relevant = append(relevant, ack.Title)
			}
		}

		if len(relevant) > 0 {
			prompt.WriteString("\n## Already Acknowledged (do NOT notify about these)\n")
			var acked strings.Builder
			for _, title := range relevant {
				acked.WriteString("- " + sanitizeUntrusted(title) + "\n")
			}
			prompt.WriteString(fenceUntrusted(acked.String()))
		}
	}

	return []llm.ChatMessage{
		{Role: "system", Content: prompt.String()},
		{Role: "user", Content: "Run your cycle now: review the state and call the tools you need. Call done when finished."},
	}
}
