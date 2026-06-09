package plugin

import (
	"aide/cli/internal/procctl"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

func Execute(ctx context.Context, m *Manifest, req *Request) (*Response, string, error) {
	return execute(ctx, m, req, false)
}

func ExecuteInteractive(ctx context.Context, m *Manifest, req *Request) (*Response, string, error) {
	return execute(ctx, m, req, true)
}

func execute(ctx context.Context, m *Manifest, req *Request, interactive bool) (*Response, string, error) {
	req.ProtocolVersion = ProtocolVersion

	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, "", fmt.Errorf("marshaling request: %w", err)
	}

	cmd, err := buildCmd(ctx, m, reqJSON, interactive)
	if err != nil {
		return nil, "", err
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	procctl.Configure(cmd)

	if err := Wrap(cmd, m); err != nil {
		return nil, "", fmt.Errorf("sandbox wrap: %w", err)
	}

	runErr := cmd.Run()
	stderr := stderrBuf.String()

	if runErr != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = runErr.Error()
		}
		return nil, stderr, fmt.Errorf("%s", msg)
	}

	resp, err := Parse(stdoutBuf.Bytes())
	if err != nil {
		return nil, stderr, fmt.Errorf("parsing output: %w", err)
	}
	return resp, stderr, nil
}

func buildCmd(ctx context.Context, m *Manifest, reqJSON []byte, interactive bool) (*exec.Cmd, error) {
	switch m.Runtime {
	case "python":
		return pythonCmdOpts(ctx, m, reqJSON, interactive)
	case "go":
		return goCmd(ctx, m, reqJSON)
	default:
		return nil, fmt.Errorf("unsupported runtime %q", m.Runtime)
	}
}

func newBytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
