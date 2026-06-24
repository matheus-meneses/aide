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

// baseClient holds the HTTP plumbing shared by every provider client. Concrete
// clients embed it so request execution, streaming, and Model() are defined once.
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

// postJSON marshals body, issues a POST to url with the given headers, and
// returns the live response on 2xx. Non-2xx responses are drained and turned
// into an error that includes the body, so callers never leak the connection.
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

// get issues a GET to url with the given headers and returns an error on any
// non-2xx status. Used by reachability checks.
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

// scanSSE reads a Server-Sent Events stream, invoking onData with the payload of
// each `data:` line. onData returning errStopSSE ends the scan cleanly; any other
// error aborts and is returned to the caller.
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

// errStopSSE signals scanSSE to stop reading without surfacing an error (e.g. on
// the OpenAI "[DONE]" sentinel).
var errStopSSE = fmt.Errorf("sse stream complete")
