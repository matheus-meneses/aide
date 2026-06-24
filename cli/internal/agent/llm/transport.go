package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	requestTimeout = 3 * time.Minute
	sseBufferLimit = 1024 * 1024
)

type baseClient struct {
	baseURL string
	model   string
	apiKey  string
	client  *http.Client
}

func newBaseClient(baseURL, model, apiKey string) baseClient {
	return baseClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: requestTimeout},
	}
}

func (c *baseClient) Model() string {
	return c.model
}

func (c *baseClient) postJSON(ctx context.Context, url string, headers map[string]string, body any, stream bool) (*http.Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling LLM: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(errBody))
	}

	return resp, nil
}

func (c *baseClient) get(ctx context.Context, url string, headers map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
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

func scanSSE(r io.Reader, onData func(data string) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), sseBufferLimit)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if err := onData(data); err != nil {
			if errors.Is(err, errStopSSE) {
				return nil
			}
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading stream: %w", err)
	}
	return nil
}

var errStopSSE = fmt.Errorf("sse stream complete")
