package agent

import (
	"aide/cli/internal/agent/tools"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/testutil"
	"strings"
	"testing"
)

func TestRenderPreferences(t *testing.T) {
	if got := renderPreferences(config.AgentPreferences{}, true); got != "" {
		t.Fatalf("default prefs should render nothing, got %q", got)
	}

	all := renderPreferences(config.AgentPreferences{Notifications: config.NotifyAll}, true)
	if !strings.Contains(all, "all noteworthy") {
		t.Fatalf("notify=all missing directive: %q", all)
	}

	if chat := renderPreferences(config.AgentPreferences{Notifications: config.NotifyAll}, false); chat != "" {
		t.Fatalf("chat path must omit notification directives, got %q", chat)
	}

	withTone := renderPreferences(config.AgentPreferences{Notifications: config.NotifyAll, Tone: "formal"}, false)
	if !strings.Contains(withTone, "formal tone") {
		t.Fatalf("tone directive missing: %q", withTone)
	}

	maxPerCycle := renderPreferences(config.AgentPreferences{MaxNotificationsPerCycle: 3}, true)
	if !strings.Contains(maxPerCycle, "at most 3") {
		t.Fatalf("max directive missing: %q", maxPerCycle)
	}
}

func TestBuildAgentMessages_LayerPrecedence(t *testing.T) {
	a := &Agent{
		store: testutil.OpenStore(t),
		tools: tools.NewToolRegistry(),
		clock: realClock{},
		cfg: &config.Config{
			Agent: config.AgentConfig{
				UserContext: "I am a tech lead.",
				Preferences: config.AgentPreferences{Notifications: config.NotifyAll, Tone: "formal"},
			},
		},
	}

	state := agentState{
		Today:      "Monday, 2026-06-15 (Mon Jun 15)",
		Time:       "09:00",
		ItemCounts: map[string]int{"jira": 3},
	}
	prompt := a.buildAgentMessages(state)[0].Content

	if !strings.HasPrefix(prompt, untrustedDataGuardrail) {
		t.Fatal("guardrail must remain the first, highest-priority text even with aggressive preferences")
	}

	for _, want := range []string{
		promptPrecedencePreamble,
		"DEFAULT BEHAVIOR",
		"USER PREFERENCES & CONTEXT",
		"all noteworthy",
		"formal tone",
		"I am a tech lead.",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}

	if strings.Index(prompt, "USER PREFERENCES & CONTEXT") < strings.Index(prompt, "DEFAULT BEHAVIOR") {
		t.Fatal("user preferences must appear after the default behavior they override")
	}
}
