package agent

import (
	"aide/cli/internal/agent/events"
	"aide/cli/internal/notification"
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

	a.runAgentCycle(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.runAgentCycle(ctx)
		case <-a.reschedule:
			if next := a.getConfig().Agent.RunIntervalDuration(); next > 0 && next != interval {
				interval = next
				ticker.Reset(interval)
				alog.Info("run interval updated to %s", interval)
			}
		}
	}
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
	lastDate := time.Now().Format("2006-01-02")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			today := now.Format("2006-01-02")
			if today != lastDate {
				firedToday = make(map[string]bool)
				lastDate = today
			}

			currentTime := now.Format("15:04")
			for _, bt := range a.getConfig().Agent.BriefingTimes {
				if firedToday[bt] {
					continue
				}
				if currentTime == bt || (now.After(parseTime(bt)) && now.Before(parseTime(bt).Add(45*time.Second))) {
					firedToday[bt] = true
					a.publishBriefing()
				}
			}
		}
	}
}

func parseTime(hhmm string) time.Time {
	now := time.Now()
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

	notification.Native("Daily Briefing", body.String())

	a.postToChatAndSSE(body.String(), time.Now().UTC().Format(time.RFC3339))
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

	now := time.Now().UTC().Format(time.RFC3339)
	a.postToChatAndSSE(msg, now)
}
