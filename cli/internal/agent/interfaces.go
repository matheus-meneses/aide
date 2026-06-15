package agent

import (
	"aide/cli/internal/agent/events"
	"aide/cli/internal/runtime/runner"
	"context"
)

// Scraper runs source scrapers and reports per-source results. The production
// implementation is *runner.Runner; tests inject a fake to drive the agent
// cycle without spawning real plugins.
type Scraper interface {
	Run(ctx context.Context, sources []string) (*runner.RunResult, error)
}

// Publisher fans agent events out to subscribers. *events.EventBus satisfies it;
// it is the write-only seam the core depends on, while the HTTP layer uses the
// concrete bus for subscription and replay.
type Publisher interface {
	Publish(events.Event)
}
