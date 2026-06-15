package agent

import (
	"aide/cli/internal/agent/events"
	"aide/cli/internal/agent/tools"
	"context"
	"time"
)

func (a *Agent) registerDefaultTools() {
	a.tools = tools.NewToolRegistry()
	tools.RegisterBuiltins(a.tools, a)
}

func (a *Agent) Scrape(ctx context.Context, sources []string) (tools.ScrapeResult, error) {
	r, err := a.runScrape(ctx, sources)
	if err != nil {
		return tools.ScrapeResult{}, err
	}
	return tools.ScrapeResult{
		SourcesTotal:  r.SourcesTotal,
		SourcesOK:     r.SourcesOK,
		SourcesFailed: r.SourcesFailed,
	}, nil
}

func (a *Agent) LastRun() time.Time {
	return a.getLastRun()
}

func (a *Agent) Bus() *events.EventBus {
	return a.bus
}

func (a *Agent) PostMessage(content, timestamp string) {
	a.postToChatAndSSE(content, timestamp)
}

// NotifyOS surfaces an urgent notification through the OS-level notifier (the
// desktop app's native notifier when present, osascript/notify-send otherwise).
// It only fires when the host delivers native notifications; otherwise the web
// UI already shows the bus event as a browser notification.
func (a *Agent) NotifyOS(title, body string) {
	if !a.NativeNotifications() {
		return
	}
	if err := a.notifier.Notify(title, body); err != nil {
		alog.Warn("os notification: %v", err)
	}
}
