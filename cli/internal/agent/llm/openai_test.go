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

func TestOpenAIChatWithToolsParsesToolCalls(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"scrape","arguments":"{\"source\":\"jira\"}"}}]}}],"usage":{"prompt_tokens":3,"completion_tokens":4,"total_tokens":7}}`)
	}))
	defer srv.Close()

	c := newOpenAIClient(srv.URL, "gpt-test", "key")
	res, err := c.ChatWithTools(context.Background(),
		[]ChatMessage{{Role: "user", Content: "go"}},
		[]ToolDefinition{{Name: "scrape", Description: "d", Parameters: json.RawMessage(`{"type":"object","properties":{}}`)}},
	)
	if err != nil {
		t.Fatalf("ChatWithTools: %v", err)
	}
	if _, ok := gotBody["tools"]; !ok {
		t.Fatalf("request did not include tools: %v", gotBody)
	}
	if len(res.ToolCalls) != 1 {
		t.Fatalf("want 1 tool call, got %d", len(res.ToolCalls))
	}
	tc := res.ToolCalls[0]
	if tc.Name != "scrape" || tc.ID != "call_1" {
		t.Fatalf("unexpected tool call: %+v", tc)
	}
	if !strings.Contains(string(tc.Arguments), "jira") {
		t.Fatalf("arguments: %s", tc.Arguments)
	}
	if res.Usage == nil || res.Usage.TotalTokens != 7 {
		t.Fatalf("usage: %+v", res.Usage)
	}
}

func TestOpenAIPostJSONNon200ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "boom")
	}))
	defer srv.Close()

	c := newOpenAIClient(srv.URL, "m", "")
	_, _, err := c.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "x"}})
	if err == nil {
		t.Fatal("expected error on 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("error should mention status: %v", err)
	}
}

func TestListModelsParsesData(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"object":"list","data":[{"id":"gpt-4o-mini"},{"id":""},{"id":"llama3.1"}]}`)
	}))
	defer srv.Close()

	models, err := ListModels(context.Background(), "litellm", srv.URL, "secret")
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if gotPath != "/models" {
		t.Fatalf("path = %q, want /models", gotPath)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("auth = %q, want Bearer secret", gotAuth)
	}
	if len(models) != 2 || models[0] != "gpt-4o-mini" || models[1] != "llama3.1" {
		t.Fatalf("models = %v, want [gpt-4o-mini llama3.1]", models)
	}
}

func TestListModelsUnsupportedProvider(t *testing.T) {
	_, err := ListModels(context.Background(), "anthropic", "https://api.anthropic.com", "")
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("error should explain lack of support: %v", err)
	}
}

func TestListModelsNon200ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, "nope")
	}))
	defer srv.Close()

	_, err := ListModels(context.Background(), "openai", srv.URL, "")
	if err == nil {
		t.Fatal("expected error on 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("error should mention status: %v", err)
	}
}

func TestOpenAIChatStreamScansSSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"He\"}}]}\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"llo\"}}]}\n")
		io.WriteString(w, "data: [DONE]\n")
	}))
	defer srv.Close()

	c := newOpenAIClient(srv.URL, "m", "")
	var chunks []string
	full, _, err := c.ChatStream(context.Background(), []ChatMessage{{Role: "user", Content: "x"}}, func(s string) {
		chunks = append(chunks, s)
	})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	if full != "Hello" {
		t.Fatalf("full = %q, want %q", full, "Hello")
	}
	if len(chunks) != 2 {
		t.Fatalf("want 2 streamed chunks, got %v", chunks)
	}
}
