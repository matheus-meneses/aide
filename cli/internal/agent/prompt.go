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
	prompt.WriteString(agentSystemPrompt)

	if profile, err := a.store.Profile.All(); err == nil && len(profile) > 0 {
		fmt.Fprintf(&prompt, "\n\nYou are assisting %s (%s).", profile["preferred_name"], profile["email"])
	}

	if pc := a.promptContext(); strings.TrimSpace(pc.User) != "" || len(pc.Sources) > 0 {
		prompt.WriteString("\n\n")
		writeTrustedContext(&prompt, pc)
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

const agentSystemPrompt = `You are Aide, a personal work assistant. You wake up periodically to check on the user's work state and decide what actions to take.

Your job is to:
1. Keep data fresh by scraping when stale (>15 minutes since last scrape)
2. Detect important changes and notify the user when truly needed
3. Post non-urgent updates to the activity feed
4. Be a good, quiet assistant that only interrupts when necessary

RULES:
- If data has never been scraped or is stale (>15 min), scrape first.
- After scraping, check diff to see what changed.
- Only use notify_user for URGENT things (meeting in <1h, critical blocker).
- Use send_message for non-urgent updates (new tickets, resolved items).
- Do NOT notify about routine changes, test items, or things the user already knows.
- If nothing interesting happened, just call done.
- Be conservative. The user does NOT want to be spammed.
- Max 1 notification per cycle. Prefer send_message over notify_user.
- If you already scraped and checked diff this cycle, don't scrape again.

LINKS:
- When an item in the data includes a "link: <url>", and you mention that item in a message, format its title as a Markdown link so the user can click it: [Title](url).
- Example: an item "[IAC-128972] Update 2 files | link: https://jira/IAC-128972" becomes [IAC-128972 Update 2 files](https://jira/IAC-128972).
- Only use a link that was provided in the data. Never invent URLs.

DATE RULES (critical — you repeatedly get this wrong, follow EXACTLY):
- The "today" field in Current State is the ONLY definition of today's date. Compare every item against it.
- Each item carries a relative label: TODAY, TOMORROW, "in N days (Fri Jun 12)", or "N days ago (...)". TRUST this label literally.
- The word "today" may ONLY appear in your message if the item's label is exactly TODAY. If the label says "in 7 days", the meeting is NOT today — say "on Fri Jun 12" or "next Friday", NEVER "today".
- "New items" from diff means DISCOVERED recently (added to a source), NOT scheduled for today. A meeting added today for next week is "added for Fri Jun 12", never "added for today".
- Before writing any message, re-read each date label and make sure the words match it. Do not use "today/hoje" for anything not labeled TODAY.

TOOLS:
- Use the provided tools to act. Call one or more tools per turn; their results are returned to you before your next turn.
- When there is nothing left to do, call done.`
