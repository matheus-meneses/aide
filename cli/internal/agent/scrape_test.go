package agent

import (
	"aide/cli/internal/runtime/runner"
	"context"
	"fmt"
	"testing"
)

type fakeScraper struct {
	known map[string]bool
	ran   bool
}

func (f *fakeScraper) ValidateFilter(filter []string) error {
	for _, n := range filter {
		if !f.known[n] {
			return fmt.Errorf("unknown or disabled sources: %s", n)
		}
	}
	return nil
}

func (f *fakeScraper) Run(_ context.Context, _ []string) (*runner.RunResult, error) {
	f.ran = true
	return &runner.RunResult{}, nil
}

// TestRunScrapeRejectsUnknownSource pins guard 1C / the runner AGENTS.md
// invariant: scraping an unknown source fails fast instead of silently
// skipping, and the scraper is never run when validation fails.
func TestRunScrapeRejectsUnknownSource(t *testing.T) {
	fs := &fakeScraper{known: map[string]bool{"jira": true}}
	a := &Agent{scraper: fs, clock: realClock{}}

	if _, err := a.runScrape(context.Background(), []string{"definitely-not-a-source"}); err == nil {
		t.Fatal("expected error for unknown source, got nil")
	}
	if fs.ran {
		t.Fatal("scraper Run should not be called when filter validation fails")
	}
}

func TestRunScrapeAllowsKnownSource(t *testing.T) {
	fs := &fakeScraper{known: map[string]bool{"jira": true}}
	a := &Agent{scraper: fs, clock: realClock{}}

	if _, err := a.runScrape(context.Background(), []string{"jira"}); err != nil {
		t.Fatalf("unexpected error for known source: %v", err)
	}
	if !fs.ran {
		t.Fatal("scraper Run should be called for a valid source")
	}
}
