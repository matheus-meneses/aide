package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnthropicChatWithToolsParsesToolUse(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"content":[{"type":"text","text":"ok"},{"type":"tool_use","id":"tu_1","name":"scrape","input":{"source":"jira"}}],"usage":{"input_tokens":5,"output_tokens":6}}`)
	}))
	defer srv.Close()

	c := newAnthropicClient(srv.URL, "claude", "key")
	res, err := c.ChatWithTools(context.Background(),
		[]ChatMessage{{Role: "user", Content: "go"}},
		[]ToolDefinition{{Name: "scrape", Parameters: json.RawMessage(`{"type":"object","properties":{}}`)}},
	)
	if err != nil {
		t.Fatalf("ChatWithTools: %v", err)
	}
	if _, ok := gotBody["tools"]; !ok {
		t.Fatalf("request did not include tools: %v", gotBody)
	}
	if res.Content != "ok" {
		t.Fatalf("content = %q, want %q", res.Content, "ok")
	}
	if len(res.ToolCalls) != 1 || res.ToolCalls[0].Name != "scrape" || res.ToolCalls[0].ID != "tu_1" {
		t.Fatalf("tool calls: %+v", res.ToolCalls)
	}
	if !strings.Contains(string(res.ToolCalls[0].Arguments), "jira") {
		t.Fatalf("arguments: %s", res.ToolCalls[0].Arguments)
	}
	if res.Usage == nil || res.Usage.TotalTokens != 11 {
		t.Fatalf("usage: %+v", res.Usage)
	}
}

func TestToAnthropicBlockMessagesThreadsToolResults(t *testing.T) {
	msgs := []ChatMessage{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "hi"},
		{Role: "assistant", ToolCalls: []ToolCall{{ID: "tu_1", Name: "scrape", Arguments: json.RawMessage(`{"source":"jira"}`)}}},
		{Role: "tool", ToolCallID: "tu_1", Name: "scrape", Content: "done"},
	}

	system, out := toAnthropicBlockMessages(msgs)
	if system != "sys" {
		t.Fatalf("system = %q, want %q", system, "sys")
	}
	if len(out) != 3 {
		t.Fatalf("want 3 block messages, got %d", len(out))
	}
	if out[1].Role != "assistant" || len(out[1].Content) != 1 || out[1].Content[0].Type != "tool_use" || out[1].Content[0].ID != "tu_1" {
		t.Fatalf("assistant tool_use block wrong: %+v", out[1])
	}
	if out[2].Role != "user" || out[2].Content[0].Type != "tool_result" || out[2].Content[0].ToolUseID != "tu_1" || out[2].Content[0].Content != "done" {
		t.Fatalf("tool_result block wrong: %+v", out[2])
	}
}
