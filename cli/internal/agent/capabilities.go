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
