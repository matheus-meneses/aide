package plugin

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
)

const ProtocolVersion = "1"

var (
	VerifySSL = true
	CABundle  = ""
)

// TLSConfig returns a *tls.Config reflecting the CLI-resolved TLS policy:
// verification disabled when VerifySSL is false, or a custom CA bundle
// trusted when CABundle points at a readable PEM file.
func TLSConfig() *tls.Config {
	cfg := &tls.Config{}
	if !VerifySSL {
		cfg.InsecureSkipVerify = true
		return cfg
	}
	if CABundle != "" {
		if pem, err := os.ReadFile(CABundle); err == nil {
			pool, perr := x509.SystemCertPool()
			if perr != nil || pool == nil {
				pool = x509.NewCertPool()
			}
			pool.AppendCertsFromPEM(pem)
			cfg.RootCAs = pool
		}
	}
	return cfg
}

type Request struct {
	ProtocolVersion string            `json:"protocol_version"`
	Action          string            `json:"action"`
	Config          map[string]any    `json:"config,omitempty"`
	Secrets         map[string]string `json:"secrets,omitempty"`
	Context         map[string]any    `json:"context,omitempty"`
	Heading         string            `json:"heading,omitempty"`
	Items           []map[string]any  `json:"items,omitempty"`
	Name            string            `json:"name,omitempty"`
	Params          map[string]any    `json:"params,omitempty"`
}

type Response struct {
	ProtocolVersion string         `json:"protocol_version"`
	OK              bool           `json:"ok"`
	Entries         []any          `json:"entries,omitempty"`
	TeamMembers     []any          `json:"team_members,omitempty"`
	Metrics         []any          `json:"metrics,omitempty"`
	Lines           []string       `json:"lines,omitempty"`
	Text            string         `json:"text,omitempty"`
	Error           string         `json:"error,omitempty"`
}

type Handler interface {
	Handle(req *Request) (*Response, error)
}

func Serve(h Handler) {
	var req Request
	dec := json.NewDecoder(os.Stdin)
	if err := dec.Decode(&req); err != nil {
		emit(&Response{ProtocolVersion: ProtocolVersion, OK: false, Error: fmt.Sprintf("decode request: %v", err)})
		os.Exit(1)
	}

	Log = NewLogger(
		stringFromContext(req.Context, "log_level"),
		stringFromContext(req.Context, "log_format"),
		"",
	)
	VerifySSL = boolFromContext(req.Context, "verify_ssl", true)
	CABundle = stringFromContext(req.Context, "ca_bundle")

	resp, err := h.Handle(&req)
	if err != nil {
		emit(&Response{ProtocolVersion: ProtocolVersion, OK: false, Error: err.Error()})
		os.Exit(1)
	}

	if resp.ProtocolVersion == "" {
		resp.ProtocolVersion = ProtocolVersion
	}
	emit(resp)
}

func emit(resp *Response) {
	json.NewEncoder(os.Stdout).Encode(resp)
}
