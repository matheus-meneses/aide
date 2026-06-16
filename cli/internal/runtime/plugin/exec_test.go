package plugin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestBuildCmdDoesNotLeakSecretsIntoEnv pins the runner invariant documented in
// internal/runtime/runner/AGENTS.md: credentials are passed to plugins on stdin,
// never as environment variables.
func TestBuildCmdDoesNotLeakSecretsIntoEnv(t *testing.T) {
	const secret = "s3cr3t-token-do-not-leak"

	m := &Manifest{Name: "acme", Runtime: "python", Dir: t.TempDir()}
	m.Entrypoint.Python.Script = "main.py"

	req := &Request{Action: "scrape", Secrets: map[string]string{"api_token": secret}}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	cmd, err := buildCmd(context.Background(), m, reqJSON, false)
	if err != nil {
		t.Fatalf("buildCmd: %v", err)
	}

	for _, e := range cmd.Env {
		if strings.Contains(e, secret) {
			t.Fatalf("secret leaked into plugin environment: %q", e)
		}
	}

	if !strings.Contains(string(reqJSON), secret) {
		t.Fatal("secret should be present in the stdin request payload")
	}
}
