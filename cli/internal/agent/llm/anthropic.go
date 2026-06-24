package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	anthropicVersion   = "2023-06-01"
	anthropicMaxTokens = 4096
)

type anthropicClient struct {
	baseClient
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Stream    bool               `json:"stream"`
}

type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
	Usage   *anthropicUsage         `json:"usage,omitempty"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicStreamEvent struct {
	Type    string `json:"type"`
	Message *struct {
		Usage *anthropicUsage `json:"usage"`
	} `json:"message"`
	Delta *struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Usage *anthropicUsage `json:"usage"`
}

func newAnthropicClient(baseURL, model, apiKey string) *anthropicClient {
	if baseURL == "" {
		baseURL = DefaultBaseURL(string(ProviderAnthropic))
	}
	return &anthropicClient{baseClient: newBaseClient(baseURL, model, apiKey)}
}

func (c *anthropicClient) authHeaders() map[string]string {
	h := map[string]string{"anthropic-version": anthropicVersion}
	if c.apiKey != "" {
		h["x-api-key"] = c.apiKey
	}
	return h
}

func splitSystem(messages []ChatMessage) (string, []anthropicMessage) {
	var system []string
	out := make([]anthropicMessage, 0, len(messages))
	for _, m := range messages {
		if m.Role == "system" {
			system = append(system, m.Content)
			continue
		}
		role := m.Role
		if role != "assistant" {
			role = "user"
		}
		out = append(out, anthropicMessage{Role: role, Content: m.Content})
	}
	return strings.Join(system, "\n\n"), out
}

func (c *anthropicClient) doRequest(ctx context.Context, messages []ChatMessage, stream bool) (*http.Response, error) {
	system, msgs := splitSystem(messages)
	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: anthropicMaxTokens,
		System:    system,
		Messages:  msgs,
		Stream:    stream,
	}
	return c.postJSON(ctx, c.baseURL+"/v1/messages", c.authHeaders(), reqBody, stream)
}

func (c *anthropicClient) Chat(ctx context.Context, messages []ChatMessage) (string, *Usage, error) {
	resp, err := c.doRequest(ctx, messages, false)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	var result anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil, fmt.Errorf("decoding response: %w", err)
	}

	var text strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}

	return text.String(), anthropicUsageToUsage(result.Usage), nil
}

func (c *anthropicClient) ChatStream(ctx context.Context, messages []ChatMessage, cb StreamCallback) (string, *Usage, error) {
	resp, err := c.doRequest(ctx, messages, true)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	var full strings.Builder
	usage := &anthropicUsage{}

	scanErr := scanSSE(resp.Body, func(data string) error {
		event, ok := decodeAnthropicStreamEvent(data)
		if !ok {
			return nil
		}

		switch event.Type {
		case "message_start":
			if event.Message != nil && event.Message.Usage != nil {
				usage.InputTokens = event.Message.Usage.InputTokens
			}
		case "content_block_delta":
			if event.Delta != nil && event.Delta.Type == "text_delta" && event.Delta.Text != "" {
				full.WriteString(event.Delta.Text)
				if cb != nil {
					cb(event.Delta.Text)
				}
			}
		case "message_delta":
			if event.Usage != nil {
				usage.OutputTokens = event.Usage.OutputTokens
			}
		case "message_stop":
		}
		return nil
	})
	if scanErr != nil {
		return full.String(), anthropicUsageToUsage(usage), scanErr
	}

	return full.String(), anthropicUsageToUsage(usage), nil
}

func decodeAnthropicStreamEvent(data string) (anthropicStreamEvent, bool) {
	var event anthropicStreamEvent
	if json.Unmarshal([]byte(data), &event) != nil {
		return anthropicStreamEvent{}, false
	}
	return event, true
}

func (c *anthropicClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.get(ctx, c.baseURL+"/v1/models", c.authHeaders())
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicBlockMessage struct {
	Role    string              `json:"role"`
	Content []anthropicReqBlock `json:"content"`
}

type anthropicReqBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
}

type anthropicToolsRequest struct {
	Model     string                  `json:"model"`
	MaxTokens int                     `json:"max_tokens"`
	System    string                  `json:"system,omitempty"`
	Messages  []anthropicBlockMessage `json:"messages"`
	Tools     []anthropicTool         `json:"tools,omitempty"`
}

type anthropicToolsResponse struct {
	Content []anthropicRespBlock `json:"content"`
	Usage   *anthropicUsage      `json:"usage,omitempty"`
}

type anthropicRespBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

func (c *anthropicClient) ChatWithTools(ctx context.Context, messages []ChatMessage, tools []ToolDefinition) (*ChatResult, error) {
	system, msgs := toAnthropicBlockMessages(messages)
	reqBody := anthropicToolsRequest{
		Model:     c.model,
		MaxTokens: anthropicMaxTokens,
		System:    system,
		Messages:  msgs,
	}
	if len(tools) > 0 {
		reqBody.Tools = toAnthropicTools(tools)
	}

	resp, err := c.postJSON(ctx, c.baseURL+"/v1/messages", c.authHeaders(), reqBody, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result anthropicToolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	out := &ChatResult{Usage: anthropicUsageToUsage(result.Usage)}
	var text strings.Builder
	for _, block := range result.Content {
		switch block.Type {
		case "text":
			text.WriteString(block.Text)
		case "tool_use":
			input := block.Input
			if len(input) == 0 {
				input = json.RawMessage("{}")
			}
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: input,
			})
		}
	}
	out.Content = text.String()
	return out, nil
}

func toAnthropicTools(tools []ToolDefinition) []anthropicTool {
	out := make([]anthropicTool, 0, len(tools))
	for _, t := range tools {
		schema := t.Parameters
		if len(schema) == 0 {
			schema = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out = append(out, anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		})
	}
	return out
}

func toAnthropicBlockMessages(messages []ChatMessage) (string, []anthropicBlockMessage) {
	var system []string
	out := make([]anthropicBlockMessage, 0, len(messages))

	for _, m := range messages {
		switch {
		case m.Role == "system":
			system = append(system, m.Content)

		case m.Role == "tool":
			out = append(out, anthropicBlockMessage{
				Role: "user",
				Content: []anthropicReqBlock{{
					Type:      "tool_result",
					ToolUseID: m.ToolCallID,
					Content:   m.Content,
				}},
			})

		case m.Role == "assistant" && len(m.ToolCalls) > 0:
			blocks := make([]anthropicReqBlock, 0, len(m.ToolCalls)+1)
			if m.Content != "" {
				blocks = append(blocks, anthropicReqBlock{Type: "text", Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				input := tc.Arguments
				if len(input) == 0 {
					input = json.RawMessage("{}")
				}
				blocks = append(blocks, anthropicReqBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: input,
				})
			}
			out = append(out, anthropicBlockMessage{Role: "assistant", Content: blocks})

		default:
			role := m.Role
			if role != "assistant" {
				role = "user"
			}
			out = append(out, anthropicBlockMessage{
				Role:    role,
				Content: []anthropicReqBlock{{Type: "text", Text: m.Content}},
			})
		}
	}

	return strings.Join(system, "\n\n"), out
}

func anthropicUsageToUsage(u *anthropicUsage) *Usage {
	if u == nil {
		return nil
	}
	return &Usage{
		PromptTokens:     u.InputTokens,
		CompletionTokens: u.OutputTokens,
		TotalTokens:      u.InputTokens + u.OutputTokens,
	}
}
