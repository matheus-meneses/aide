package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type openAIClient struct {
	baseClient
}

type oaiChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type oaiChatResponse struct {
	Choices []oaiChatChoice `json:"choices"`
	Usage   *Usage          `json:"usage,omitempty"`
}

type oaiChatChoice struct {
	Message ChatMessage `json:"message"`
	Delta   *oaiDelta   `json:"delta,omitempty"`
}

type oaiDelta struct {
	Content string `json:"content"`
}

type oaiStreamChunk struct {
	Choices []oaiChatChoice `json:"choices"`
	Usage   *Usage          `json:"usage,omitempty"`
}

func newOpenAIClient(baseURL, model, apiKey string) *openAIClient {
	return &openAIClient{baseClient: newBaseClient(baseURL, model, apiKey)}
}

func (c *openAIClient) authHeaders() map[string]string {
	if c.apiKey == "" {
		return nil
	}
	return map[string]string{"Authorization": "Bearer " + c.apiKey}
}

func (c *openAIClient) doRequest(ctx context.Context, messages []ChatMessage, stream bool) (*http.Response, error) {
	reqBody := oaiChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   stream,
	}
	return c.postJSON(ctx, c.baseURL+"/chat/completions", c.authHeaders(), reqBody, stream)
}

func (c *openAIClient) Chat(ctx context.Context, messages []ChatMessage) (string, *Usage, error) {
	resp, err := c.doRequest(ctx, messages, false)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	var result oaiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", result.Usage, fmt.Errorf("LLM returned no choices")
	}

	return result.Choices[0].Message.Content, result.Usage, nil
}

func (c *openAIClient) ChatStream(ctx context.Context, messages []ChatMessage, cb StreamCallback) (string, *Usage, error) {
	resp, err := c.doRequest(ctx, messages, true)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	var full strings.Builder
	var usage *Usage

	scanErr := scanSSE(resp.Body, func(data string) error {
		if data == "[DONE]" {
			return errStopSSE
		}

		chunk, ok := decodeOAIStreamChunk(data)
		if !ok {
			return nil
		}

		if chunk.Usage != nil {
			usage = chunk.Usage
		}

		if len(chunk.Choices) == 0 {
			return nil
		}

		choice := chunk.Choices[0]
		content := ""
		if choice.Delta != nil {
			content = choice.Delta.Content
		}

		if content != "" {
			full.WriteString(content)
			if cb != nil {
				cb(content)
			}
		}
		return nil
	})
	if scanErr != nil {
		return full.String(), usage, scanErr
	}

	if usage == nil {
		chars := full.Len()
		usage = &Usage{
			CompletionTokens: chars / 4,
			TotalTokens:      chars / 4,
		}
	}

	return full.String(), usage, nil
}

type oaiTool struct {
	Type     string          `json:"type"`
	Function oaiToolFunction `json:"function"`
}

type oaiToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type oaiToolCall struct {
	ID       string              `json:"id"`
	Type     string              `json:"type"`
	Function oaiToolCallFunction `json:"function"`
}

type oaiToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type oaiMessage struct {
	Role       string        `json:"role"`
	Content    string        `json:"content,omitempty"`
	ToolCalls  []oaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
	Name       string        `json:"name,omitempty"`
}

type oaiToolsRequest struct {
	Model      string       `json:"model"`
	Messages   []oaiMessage `json:"messages"`
	Tools      []oaiTool    `json:"tools,omitempty"`
	ToolChoice string       `json:"tool_choice,omitempty"`
	Stream     bool         `json:"stream"`
}

type oaiToolsResponse struct {
	Choices []struct {
		Message struct {
			Content   string        `json:"content"`
			ToolCalls []oaiToolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage *Usage `json:"usage,omitempty"`
}

func (c *openAIClient) ChatWithTools(ctx context.Context, messages []ChatMessage, tools []ToolDefinition) (*ChatResult, error) {
	reqBody := oaiToolsRequest{
		Model:    c.model,
		Messages: toOAIMessages(messages),
	}
	if len(tools) > 0 {
		reqBody.Tools = toOAITools(tools)
		reqBody.ToolChoice = "auto"
	}

	resp, err := c.postJSON(ctx, c.baseURL+"/chat/completions", c.authHeaders(), reqBody, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result oaiToolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("LLM returned no choices")
	}

	msg := result.Choices[0].Message
	out := &ChatResult{Content: msg.Content, Usage: result.Usage}
	for _, tc := range msg.ToolCalls {
		args := tc.Function.Arguments
		if strings.TrimSpace(args) == "" {
			args = "{}"
		}
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: json.RawMessage(args),
		})
	}
	return out, nil
}

func toOAITools(tools []ToolDefinition) []oaiTool {
	out := make([]oaiTool, 0, len(tools))
	for _, t := range tools {
		params := t.Parameters
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out = append(out, oaiTool{
			Type: "function",
			Function: oaiToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}
	return out
}

func toOAIMessages(messages []ChatMessage) []oaiMessage {
	out := make([]oaiMessage, 0, len(messages))
	for _, m := range messages {
		om := oaiMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
			Name:       m.Name,
		}
		for _, tc := range m.ToolCalls {
			args := string(tc.Arguments)
			if strings.TrimSpace(args) == "" {
				args = "{}"
			}
			om.ToolCalls = append(om.ToolCalls, oaiToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: oaiToolCallFunction{
					Name:      tc.Name,
					Arguments: args,
				},
			})
		}
		out = append(out, om)
	}
	return out
}

func decodeOAIStreamChunk(data string) (oaiStreamChunk, bool) {
	var chunk oaiStreamChunk
	if json.Unmarshal([]byte(data), &chunk) != nil {
		return oaiStreamChunk{}, false
	}
	return chunk, true
}

func (c *openAIClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.get(ctx, c.baseURL+"/models", c.authHeaders())
}
