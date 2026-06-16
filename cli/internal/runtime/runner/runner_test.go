package runner

import (
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/plugin"
	"testing"
)

func testRunner() *Runner {
	cfg := &config.Config{
		Sources: map[string]config.Source{
			"jira":   {Enabled: true},
			"github": {Enabled: false},
			"slack":  {Enabled: true},
		},
		Team: []config.TeamMember{
			{Name: "Alice", Aliases: []string{"al", "a.lice"}},
		},
	}
	cfg.Settings.Concurrency = 1
	return &Runner{cfg: cfg}
}

func TestValidateFilter(t *testing.T) {
	r := testRunner()
	tests := []struct {
		name    string
		filter  []string
		wantErr bool
	}{
		{"empty filter", nil, false},
		{"all known enabled", []string{"jira", "slack"}, false},
		{"unknown source", []string{"jira", "nope"}, true},
		{"disabled source", []string{"github"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.ValidateFilter(tt.filter)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestResolveSources(t *testing.T) {
	r := testRunner()

	all := r.resolveSources(nil)
	if len(all) != 2 {
		t.Fatalf("empty filter: got %d enabled sources, want 2", len(all))
	}
	if _, ok := all["github"]; ok {
		t.Error("disabled source github should not be resolved")
	}

	subset := r.resolveSources([]string{"jira"})
	if len(subset) != 1 {
		t.Fatalf("filter [jira]: got %d, want 1", len(subset))
	}

	unknown := r.resolveSources([]string{"nope"})
	if len(unknown) != 0 {
		t.Fatalf("filter [nope]: got %d, want 0", len(unknown))
	}
}

func TestNormalizeResponse(t *testing.T) {
	r := testRunner()

	resp := &plugin.Response{
		Entries: []plugin.Entry{
			{Title: "Issue", Member: "al", Category: "bug", Metadata: map[string]any{"web_url": "https://x/1"}},
			{Title: "Throughput", Metadata: map[string]any{"mode": "metric", "metric_value": 3.0}},
		},
		Metrics:     []plugin.Metric{{Name: "latency", Value: 1.5}},
		TeamMembers: []plugin.TeamMember{{Name: "Bob", Email: "bob@x"}},
	}

	items, metrics, members := r.normalizeResponse("jira", resp)

	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].Member != "Alice" {
		t.Errorf("member alias not resolved: got %q, want Alice", items[0].Member)
	}
	if items[0].Source != "jira" {
		t.Errorf("source = %q, want jira", items[0].Source)
	}
	if items[0].Link != "https://x/1" {
		t.Errorf("link from metadata web_url not applied: %q", items[0].Link)
	}
	if items[0].Fingerprint == "" {
		t.Error("fingerprint should be set")
	}

	if len(metrics) != 2 {
		t.Fatalf("metrics = %d, want 2 (one inline mode=metric + one resp.Metrics)", len(metrics))
	}
	if len(members) != 1 || members[0].Name != "Bob" || members[0].Source != "jira" {
		t.Fatalf("unexpected members: %+v", members)
	}
}
