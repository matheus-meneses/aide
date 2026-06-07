package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (a *Agent) StartAutonomous(ctx context.Context, port int) error {
	a.loadMemory()

	interval := a.cfg.Agent.RunIntervalDuration()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	go func() {
		a.runAgentCycle(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.runAgentCycle(ctx)
			}
		}
	}()

	go func() {
		time.Sleep(2 * time.Second)
		a.checkIdentityOnStart()
	}()

	if len(a.cfg.Agent.BriefingTimes) > 0 {
		go a.runBriefingScheduler(ctx)
	}

	return a.Serve(ctx, port)
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
			for _, bt := range a.cfg.Agent.BriefingTimes {
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
	events, _ := a.store.Items.TodayEvents()

	var body strings.Builder
	body.WriteString("Good morning! Here's your briefing:\n")
	if len(events) > 0 {
		body.WriteString(fmt.Sprintf("- %d meetings today\n", len(events)))
	}
	total := 0
	for source, count := range counts {
		body.WriteString(fmt.Sprintf("- %d open %s items\n", count, source))
		total += count
	}
	if total == 0 && len(events) == 0 {
		body.WriteString("- No open items or meetings. Clean slate!")
	}

	if a.bus != nil {
		a.bus.Publish(Event{
			Type:     "briefing",
			Priority: "normal",
			Data:     fmt.Sprintf(`{"title":"Daily Briefing","body":%q}`, body.String()),
		})
	}

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
