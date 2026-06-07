package agent

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

type LLMClient struct {
	baseURL string
	model   string
	apiKey  string
	client  *http.Client
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type StreamCallback func(chunk string)

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Usage   *Usage       `json:"usage,omitempty"`
}

type chatChoice struct {
	Message ChatMessage `json:"message"`
	Delta   *chatDelta  `json:"delta,omitempty"`
}

type chatDelta struct {
	Content string `json:"content"`
}

type streamChunk struct {
	Choices []chatChoice `json:"choices"`
	Usage   *Usage       `json:"usage,omitempty"`
}

func NewLLMClient(baseURL, model, apiKey string) *LLMClient {
	return &LLMClient{
		baseURL: baseURL,
		model:   model,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 3 * time.Minute},
	}
}

func (c *LLMClient) Model() string {
	return c.model
}

func (c *LLMClient) doRequest(ctx context.Context, messages []ChatMessage, stream bool) (*http.Response, error) {
	reqBody := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   stream,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
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

func (c *LLMClient) Chat(ctx context.Context, messages []ChatMessage) (string, *Usage, error) {
	resp, err := c.doRequest(ctx, messages, false)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", result.Usage, fmt.Errorf("LLM returned no choices")
	}

	return result.Choices[0].Message.Content, result.Usage, nil
}

func (c *LLMClient) ChatStream(ctx context.Context, messages []ChatMessage, cb StreamCallback) (string, *Usage, error) {
	resp, err := c.doRequest(ctx, messages, true)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	var full strings.Builder
	var usage *Usage
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if chunk.Usage != nil {
			usage = chunk.Usage
		}

		if len(chunk.Choices) == 0 {
			continue
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
	}

	if err := scanner.Err(); err != nil {
		return full.String(), usage, fmt.Errorf("reading stream: %w", err)
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

func (c *LLMClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
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
