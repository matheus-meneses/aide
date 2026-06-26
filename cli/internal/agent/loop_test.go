package agent

import (
	"aide/cli/internal/agent/llm"
	"aide/cli/internal/agent/tools"
	"aide/cli/internal/testutil"
	"context"
	"encoding/json"
	"testing"
)

type stubLLM struct {
	results   []*llm.ChatResult
	always    *llm.ChatResult
	calls     int
	lastTools []llm.ToolDefinition
	chatReply string
	chatErr   error
	lastChat  []llm.ChatMessage
}

func (s *stubLLM) Chat(_ context.Context, messages []llm.ChatMessage) (string, *llm.Usage, error) {
	s.lastChat = messages
	return s.chatReply, nil, s.chatErr
}

func (s *stubLLM) ChatStream(context.Context, []llm.ChatMessage, llm.StreamCallback) (string, *llm.Usage, error) {
	return "", nil, nil
}

func (s *stubLLM) ChatWithTools(_ context.Context, _ []llm.ChatMessage, td []llm.ToolDefinition) (*llm.ChatResult, error) {
	s.lastTools = td
	i := s.calls
	s.calls++
	if s.always != nil {
		return s.always, nil
	}
	if i < len(s.results) {
		return s.results[i], nil
	}
	return &llm.ChatResult{}, nil
}

func (s *stubLLM) ListModels(context.Context) ([]string, error) { return nil, nil }
func (s *stubLLM) Ping() error                                  { return nil }
func (s *stubLLM) Model() string                                { return "stub" }

func newLoopTestAgent(t *testing.T, stub llm.LLM, reg *tools.ToolRegistry) *Agent {
	t.Helper()
	return &Agent{
		store: testutil.OpenStore(t),
		llm:   stub,
		tools: reg,
		clock: realClock{},
	}
}

func scrapeCountingRegistry(counter *int) *tools.ToolRegistry {
	reg := tools.NewToolRegistry()
	reg.Register(&tools.Tool{
		Name: "scrape",
		Execute: func(context.Context, map[string]string) (string, error) {
			*counter++
			return "ok", nil
		},
	})
	return reg
}

func TestRunAgentCycleExecutesToolThenStopsOnDone(t *testing.T) {
	var scraped int
	reg := scrapeCountingRegistry(&scraped)
	stub := &stubLLM{results: []*llm.ChatResult{
		{ToolCalls: []llm.ToolCall{{ID: "1", Name: "scrape", Arguments: json.RawMessage(`{"source":"jira"}`)}}},
		{ToolCalls: []llm.ToolCall{{ID: "2", Name: "done"}}},
	}}

	a := newLoopTestAgent(t, stub, reg)
	a.runAgentCycle(context.Background())

	if scraped != 1 {
		t.Fatalf("scrape executed %d times, want 1", scraped)
	}
	if stub.calls != 2 {
		t.Fatalf("model called %d times, want 2", stub.calls)
	}
	if len(stub.lastTools) == 0 {
		t.Fatal("expected tool definitions to be passed to the model")
	}
}

func TestRunAgentCycleFallbackParsesPromptJSON(t *testing.T) {
	var scraped int
	reg := scrapeCountingRegistry(&scraped)
	stub := &stubLLM{results: []*llm.ChatResult{
		{Content: `{"tool":"scrape","params":{}}`},
		{Content: `{"tool":"done","params":{}}`},
	}}

	a := newLoopTestAgent(t, stub, reg)
	a.runAgentCycle(context.Background())

	if scraped != 1 {
		t.Fatalf("fallback scrape executed %d times, want 1", scraped)
	}
}

func TestRunAgentCycleRespectsMaxActions(t *testing.T) {
	var scraped int
	reg := scrapeCountingRegistry(&scraped)
	stub := &stubLLM{always: &llm.ChatResult{
		ToolCalls: []llm.ToolCall{{ID: "x", Name: "scrape", Arguments: json.RawMessage(`{}`)}},
	}}

	a := newLoopTestAgent(t, stub, reg)
	a.runAgentCycle(context.Background())

	if scraped != maxActionsPerCycle {
		t.Fatalf("scrape executed %d times, want %d", scraped, maxActionsPerCycle)
	}
	if stub.calls != maxActionsPerCycle {
		t.Fatalf("model called %d times, want %d", stub.calls, maxActionsPerCycle)
	}
}
