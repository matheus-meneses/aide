package agent

import (
	"aide/cli/internal/persistence/store"
	"aide/cli/internal/testutil"
	"strings"
	"testing"
	"time"
)

func TestBuildContext_TrustedContextOutsideFence(t *testing.T) {
	s := testutil.OpenStore(t)

	if _, _, err := s.Items.Upsert("jira", []store.Item{{
		Fingerprint: "fp-1",
		Source:      "jira",
		Category:    "task",
		Priority:    "high",
		Title:       "Fix login",
	}}); err != nil {
		t.Fatalf("seeding item: %v", err)
	}

	pc := PromptContext{
		User:    "I am a tech lead; prioritize incidents.",
		Sources: map[string]string{"jira": "Only surface tickets assigned to me."},
	}
	out, err := BuildContext(s, time.Now(), pc)
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	for _, want := range []string{pc.User, pc.Sources["jira"], "About the user", "Source guidance"} {
		if !strings.Contains(out, want) {
			t.Fatalf("context output missing %q\n---\n%s", want, out)
		}
	}

	fenceIdx := strings.Index(out, untrustedBegin)
	if fenceIdx < 0 {
		t.Fatal("untrusted fence missing")
	}
	if idx := strings.Index(out, pc.User); idx < 0 || idx > fenceIdx {
		t.Fatalf("user context must sit outside (before) the untrusted fence: user at %d, fence at %d", idx, fenceIdx)
	}
	if idx := strings.Index(out, pc.Sources["jira"]); idx < 0 || idx > fenceIdx {
		t.Fatalf("source guidance must sit outside (before) the untrusted fence: at %d, fence at %d", idx, fenceIdx)
	}
}

func TestBuildContext_NoContextOmitsHeaders(t *testing.T) {
	s := testutil.OpenStore(t)

	out, err := BuildContext(s, time.Now(), PromptContext{})
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	for _, unwanted := range []string{"About the user", "Source guidance"} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("empty context should not render %q", unwanted)
		}
	}
}

func TestOrderedSources_ByCountThenName(t *testing.T) {
	grouped := map[string][]store.Item{
		"alpha": make([]store.Item, 1),
		"bravo": make([]store.Item, 3),
		"delta": make([]store.Item, 3),
	}
	got := orderedSources(grouped)
	want := []string{"bravo", "delta", "alpha"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("orderedSources = %v, want %v", got, want)
	}
}
