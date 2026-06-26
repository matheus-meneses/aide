package agent

import (
	"aide/cli/internal/platform/config"
	"context"
	"errors"
	"strings"
	"testing"
)

func configuredAgent(t *testing.T, stub *stubLLM) *Agent {
	t.Helper()
	a := newLoopTestAgent(t, stub, nil)
	a.cfg = &config.Config{}
	a.cfg.Agent.LLMModel = "stub-model"
	a.cfg.Agent.LLMURL = "http://stub.local"
	return a
}

func TestBriefingBodyDeterministicWhenLLMUnavailable(t *testing.T) {
	a := newLoopTestAgent(t, &stubLLM{chatReply: "SHOULD NOT BE USED"}, nil)

	body := a.briefingBody(context.Background())

	if !strings.Contains(body, "Clean slate") {
		t.Fatalf("expected deterministic briefing, got: %q", body)
	}
	if strings.Contains(body, "SHOULD NOT BE USED") {
		t.Fatalf("LLM was used despite being unconfigured: %q", body)
	}
}

func TestBriefingBodyUsesLLMWhenConfigured(t *testing.T) {
	a := configuredAgent(t, &stubLLM{chatReply: "  - Urgent: ship the release  "})

	body := a.briefingBody(context.Background())

	if body != "- Urgent: ship the release" {
		t.Fatalf("expected trimmed LLM body, got: %q", body)
	}
}

func TestBriefingBodyFallsBackWhenLLMErrors(t *testing.T) {
	a := configuredAgent(t, &stubLLM{chatReply: "ignored", chatErr: errors.New("boom")})

	body := a.briefingBody(context.Background())

	if !strings.Contains(body, "Clean slate") {
		t.Fatalf("expected deterministic fallback on LLM error, got: %q", body)
	}
	if strings.Contains(body, "ignored") {
		t.Fatalf("errored LLM output leaked into briefing: %q", body)
	}
}

func TestBriefingBodyFallsBackWhenLLMEmpty(t *testing.T) {
	a := configuredAgent(t, &stubLLM{chatReply: "   "})

	body := a.briefingBody(context.Background())

	if !strings.Contains(body, "Clean slate") {
		t.Fatalf("expected deterministic fallback on empty LLM output, got: %q", body)
	}
}

func TestSynthesizeBriefingPromptCarriesGuardrails(t *testing.T) {
	stub := &stubLLM{chatReply: "summary"}
	a := configuredAgent(t, stub)

	if _, err := a.synthesizeBriefing(context.Background()); err != nil {
		t.Fatalf("synthesizeBriefing: %v", err)
	}
	if len(stub.lastChat) == 0 {
		t.Fatal("no messages sent to the LLM")
	}

	system := stub.lastChat[0].Content
	if !strings.HasPrefix(system, promptPrecedencePreamble) {
		t.Fatal("system prompt does not lead with the precedence preamble")
	}
	for _, want := range []string{agentCoreRules, untrustedDataGuardrail, untrustedBegin} {
		if !strings.Contains(system, want) {
			t.Fatalf("system prompt missing a required guardrail segment: %q", want)
		}
	}
}
