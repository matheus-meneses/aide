//go:build linux

package sandbox

import (
	"strings"
	"testing"
)

func joined(args []string) string { return strings.Join(args, " ") }

func TestBuildBwrapArgsBaseline(t *testing.T) {
	p := Policy{Name: "acme", Dir: "/plugins/acme"}
	args := buildBwrapArgs(p, []string{"/plugins/acme/bin/run"})
	s := joined(args)

	if !strings.Contains(s, "--ro-bind / /") {
		t.Errorf("root should be read-only bound:\n%s", s)
	}
	if !strings.Contains(s, "--bind /plugins/acme /plugins/acme") {
		t.Errorf("plugin dir should be writable:\n%s", s)
	}
	if !strings.Contains(s, "--unshare-net") {
		t.Errorf("network namespace must be unshared with no network policy:\n%s", s)
	}
	if args[len(args)-1] != "/plugins/acme/bin/run" {
		t.Errorf("child args must be appended last, got %q", args[len(args)-1])
	}
	if !strings.Contains(s, "-- /plugins/acme/bin/run") {
		t.Errorf("child args must follow the -- separator:\n%s", s)
	}
}

func TestBuildBwrapArgsGrantsDeclaredWrites(t *testing.T) {
	p := Policy{Name: "acme", Dir: "/plugins/acme", Writes: []string{"/data/out", "/var/cache/acme"}}
	s := joined(buildBwrapArgs(p, []string{"run"}))

	for _, w := range p.Writes {
		if !strings.Contains(s, "--bind "+w+" "+w) {
			t.Errorf("declared write %q not bound:\n%s", w, s)
		}
	}
}

func TestBuildBwrapArgsNetworkToggle(t *testing.T) {
	p := Policy{Name: "acme", Dir: "/plugins/acme", Network: []string{"api.example.com"}}
	s := joined(buildBwrapArgs(p, []string{"run"}))
	if strings.Contains(s, "--unshare-net") {
		t.Errorf("network namespace must not be unshared when network is requested:\n%s", s)
	}
}
