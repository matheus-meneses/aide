package agent

import (
	"aide/cli/internal/agent/events"
	"aide/cli/internal/agent/llm"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (a *Agent) StartAutonomous(ctx context.Context) error {
	a.loadMemory()
	a.sessions.startJanitor(ctx)

	a.schedMu.Lock()
	a.autoCtx = ctx
	a.reschedule = make(chan struct{}, 1)
	a.schedMu.Unlock()

	go a.runScheduleLoop(ctx)

	go func() {
		time.Sleep(2 * time.Second)
		a.checkIdentityOnStart()
	}()

	a.maybeStartBriefingScheduler(ctx)

	<-ctx.Done()
	return nil
}

func (a *Agent) runScheduleLoop(ctx context.Context) {
	interval := a.getConfig().Agent.RunIntervalDuration()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	a.startCycleIfConfigured(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.maybeRunCycle(ctx)
		case <-a.reschedule:
			if next := a.getConfig().Agent.RunIntervalDuration(); next > 0 && next != interval {
				interval = next
				ticker.Reset(interval)
				alog.Info("run interval updated to %s", interval)
			}
			a.startCycleIfConfigured(ctx)
		}
	}
}

// llmConfigured reports whether a model is set up. Until it is, the autonomous
// loop stays idle instead of hammering an empty endpoint (the symptom is
// repeated `unsupported protocol scheme ""` LLM errors), so `aide ui` can run
// before the in-browser setup wizard has been completed.
func (a *Agent) llmConfigured() bool {
	cfg := a.getConfig()
	return cfg != nil && cfg.Agent.LLMModel != "" && cfg.Agent.LLMURL != ""
}

// maybeRunCycle runs an agent cycle when a model is configured, otherwise it
// logs the idle state once and returns.
func (a *Agent) maybeRunCycle(ctx context.Context) {
	if !a.llmConfigured() {
		a.schedMu.Lock()
		already := a.idleLogged
		a.idleLogged = true
		a.schedMu.Unlock()
		if !already {
			alog.Info("autonomous loop idle: no model configured yet — finish setup to start")
		}
		return
	}
	a.schedMu.Lock()
	a.idleLogged = false
	a.schedMu.Unlock()
	a.runAgentCycle(ctx)
}

// startCycleIfConfigured kicks the first cycle as soon as a model is configured
// (at startup or right after the setup wizard saves), then defers to the ticker.
func (a *Agent) startCycleIfConfigured(ctx context.Context) {
	if !a.llmConfigured() {
		a.maybeRunCycle(ctx)
		return
	}
	a.schedMu.Lock()
	already := a.autoCycleStarted
	a.autoCycleStarted = true
	a.schedMu.Unlock()
	if already {
		return
	}
	a.maybeRunCycle(ctx)
}

// signalReschedule asks the running schedule loop to re-read the configured run
// interval (and reset its ticker) and lazily starts the briefing scheduler if it
// was not running. Safe to call before StartAutonomous (it becomes a no-op).
func (a *Agent) signalReschedule() {
	a.schedMu.Lock()
	ch := a.reschedule
	ctx := a.autoCtx
	a.schedMu.Unlock()

	if ch != nil {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	if ctx != nil {
		a.maybeStartBriefingScheduler(ctx)
	}
}

func (a *Agent) maybeStartBriefingScheduler(ctx context.Context) {
	if len(a.getConfig().Agent.BriefingTimes) == 0 {
		return
	}
	a.schedMu.Lock()
	if a.briefingStarted {
		a.schedMu.Unlock()
		return
	}
	a.briefingStarted = true
	a.schedMu.Unlock()

	go a.runBriefingScheduler(ctx)
}

func (a *Agent) runBriefingScheduler(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	firedToday := make(map[string]bool)
	lastDate := a.clock.Now().Format("2006-01-02")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := a.clock.Now()
			today := now.Format("2006-01-02")
			if today != lastDate {
				firedToday = make(map[string]bool)
				lastDate = today
			}

			for _, bt := range dueBriefings(now, a.getConfig().Agent.BriefingTimes, firedToday) {
				firedToday[bt] = true
				a.publishBriefing(ctx)
			}
		}
	}
}

// dueBriefings returns the configured briefing times that should fire at now and
// have not already fired today. Kept pure so scheduling decisions are unit
// testable without real timers.
func dueBriefings(now time.Time, times []string, firedToday map[string]bool) []string {
	current := now.Format("15:04")
	var due []string
	for _, bt := range times {
		if firedToday[bt] {
			continue
		}
		target := parseTime(now, bt)
		if current == bt || (now.After(target) && now.Before(target.Add(45*time.Second))) {
			due = append(due, bt)
		}
	}
	return due
}

func parseTime(now time.Time, hhmm string) time.Time {
	parts := strings.SplitN(hhmm, ":", 2)
	if len(parts) != 2 {
		return now
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, now.Location())
}

// briefingInstruction asks the model to synthesize the briefing strictly from
// the context BuildContext already assembled (which carries the untrusted-data
// guardrail and the date rules), so the synthesized path stays safe.
const briefingInstruction = "Write my daily briefing now. Using ONLY the data above, produce a concise, " +
	"prioritized summary: lead with anything urgent or happening TODAY (events first, then " +
	"high/critical open items), then briefly note what else is open by area. Use short markdown " +
	"bullets, include [title](url) links where available, and keep it under ~12 lines. If there " +
	"are no open items and no events today, say it's a clean slate. Do not invent anything or say " +
	"you will look something up. Follow the CRITICAL DATE RULE."

func (a *Agent) publishBriefing(ctx context.Context) {
	body := a.briefingBody(ctx)

	if a.bus != nil {
		a.bus.Publish(events.Event{
			Type:     "briefing",
			Priority: "normal",
			Data:     fmt.Sprintf(`{"title":"Daily Briefing","body":%q}`, body),
		})
	}

	if a.notifier != nil {
		if err := a.notifier.Notify("Daily Briefing", body); err != nil {
			alog.Warn("briefing notification: %v", err)
		}
	}

	a.postToChatAndSSE(body, a.clock.Now().UTC().Format(time.RFC3339))
}

// briefingBody returns an LLM-synthesized briefing when a model is configured
// and reachable, and the deterministic template otherwise. Any failure (no
// model, build error, LLM error, or empty output) falls back deterministically
// so a briefing always goes out.
func (a *Agent) briefingBody(ctx context.Context) string {
	if a.llmConfigured() && a.getLLM() != nil {
		body, err := a.synthesizeBriefing(ctx)
		switch {
		case err != nil:
			alog.Warn("briefing synthesis failed, using deterministic fallback: %v", err)
		case strings.TrimSpace(body) == "":
			alog.Warn("briefing synthesis returned empty output, using deterministic fallback")
		default:
			return body
		}
	}
	return a.deterministicBriefing()
}

func (a *Agent) synthesizeBriefing(ctx context.Context) (string, error) {
	sysCtx, err := BuildContext(a.store, a.clock.Now(), a.promptContext())
	if err != nil {
		return "", fmt.Errorf("building context: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	client := a.getLLM()
	full, usage, err := client.Chat(ctx, []llm.ChatMessage{
		{Role: "system", Content: sysCtx},
		{Role: "user", Content: briefingInstruction},
	})
	if err != nil {
		return "", err
	}

	if usage != nil {
		if err := a.store.Tokens.Record("briefing", client.Model(), usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens); err != nil {
			alog.Warn("failed to record briefing token usage: %v", err)
		}
	}

	return strings.TrimSpace(full), nil
}

func (a *Agent) deterministicBriefing() string {
	counts, _ := a.store.Items.CountOpenBySource()
	todayEvents, _ := a.store.Items.TodayEvents()

	var body strings.Builder
	body.WriteString("Good morning! Here's your briefing:\n")
	if len(todayEvents) > 0 {
		fmt.Fprintf(&body, "- %d meetings today\n", len(todayEvents))
	}
	total := 0
	for source, count := range counts {
		fmt.Fprintf(&body, "- %d open %s items\n", count, source)
		total += count
	}
	if total == 0 && len(todayEvents) == 0 {
		body.WriteString("- No open items or meetings. Clean slate!")
	}
	return body.String()
}

func (a *Agent) checkIdentityOnStart() {
	profile, _ := a.store.Profile.All()
	if len(profile) > 0 && profile["preferred_name"] != "" {
		return
	}

	msg := "Hey! I'm **Aide**, your personal work assistant.\n\n" +
		"I don't know your name yet. Tell me who you are so I can personalize your experience.\n\n" +
		"Use the command: `/whoami set <Your Name> <email> <nickname>`\n\n" +
		"For example: `/whoami set John Doe john@company.com John`"

	now := a.clock.Now().UTC().Format(time.RFC3339)
	a.postToChatAndSSE(msg, now)
}
