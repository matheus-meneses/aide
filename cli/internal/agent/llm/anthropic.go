package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	anthropicVersion   = "2023-06-01"
	anthropicMaxTokens = 4096
)

type anthropicClient struct {
	baseURL string
	model   string
	apiKey  string
	client  *http.Client
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
	return &anthropicClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 3 * time.Minute},
	}
}

func (c *anthropicClient) Model() string {
	return c.model
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

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", anthropicVersion)
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling LLM: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
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
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var event anthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
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
	}

	if err := scanner.Err(); err != nil {
		return full.String(), anthropicUsageToUsage(usage), fmt.Errorf("reading stream: %w", err)
	}

	return full.String(), anthropicUsageToUsage(usage), nil
}

func (c *anthropicClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/v1/models", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("anthropic-version", anthropicVersion)
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to LLM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LLM returned status %d", resp.StatusCode)
	}
	return nil
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
