package agent

import (
	"aide/cli/internal/agent/events"
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
	return cfg.Agent.LLMModel != "" && cfg.Agent.LLMURL != ""
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
				a.publishBriefing()
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

func (a *Agent) publishBriefing() {
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

	if a.bus != nil {
		a.bus.Publish(events.Event{
			Type:     "briefing",
			Priority: "normal",
			Data:     fmt.Sprintf(`{"title":"Daily Briefing","body":%q}`, body.String()),
		})
	}

	if err := a.notifier.Notify("Daily Briefing", body.String()); err != nil {
		alog.Warn("briefing notification: %v", err)
	}

	a.postToChatAndSSE(body.String(), a.clock.Now().UTC().Format(time.RFC3339))
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
