package runner

import "testing"

func TestParseTextLevel(t *testing.T) {
	const source = "sailpoint"
	cases := []struct {
		name      string
		line      string
		wantOK    bool
		wantLevel string
		wantMsg   string
	}{
		{
			name:      "debug with timestamp and scope prefix",
			line:      "2026-06-17T22:53:58Z [debug] sailpoint: Starting browser...",
			wantOK:    true,
			wantLevel: "debug",
			wantMsg:   "Starting browser...",
		},
		{
			name:      "warn level",
			line:      "2026-06-17T22:54:00Z [warn] sailpoint: slow response",
			wantOK:    true,
			wantLevel: "warn",
			wantMsg:   "slow response",
		},
		{
			name:      "error level",
			line:      "2026-06-17T22:54:01Z [error] sailpoint: auth failed",
			wantOK:    true,
			wantLevel: "error",
			wantMsg:   "auth failed",
		},
		{
			name:      "info without timestamp",
			line:      "[info] sailpoint: ready",
			wantOK:    true,
			wantLevel: "info",
			wantMsg:   "ready",
		},
		{
			name:      "no scope prefix keeps full message",
			line:      "2026-06-17T22:54:02Z [debug] connecting",
			wantOK:    true,
			wantLevel: "debug",
			wantMsg:   "connecting",
		},
		{
			name:      "uppercase level normalized",
			line:      "2026-06-17T22:54:03Z [ERROR] sailpoint: boom",
			wantOK:    true,
			wantLevel: "error",
			wantMsg:   "boom",
		},
		{
			name:   "non-level bracket ignored",
			line:   "[1234] some library output",
			wantOK: false,
		},
		{
			name:   "traceback ignored",
			line:   "Traceback (most recent call last):",
			wantOK: false,
		},
		{
			name:   "plain text ignored",
			line:   "no brackets here",
			wantOK: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			scope, level, msg, ok := parseTextLevel(c.line, source)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if !c.wantOK {
				return
			}
			if scope != source {
				t.Errorf("scope = %q, want %q", scope, source)
			}
			if level != c.wantLevel {
				t.Errorf("level = %q, want %q", level, c.wantLevel)
			}
			if msg != c.wantMsg {
				t.Errorf("msg = %q, want %q", msg, c.wantMsg)
			}
		})
	}
}
