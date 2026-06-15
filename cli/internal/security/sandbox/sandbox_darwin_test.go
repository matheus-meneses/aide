//go:build darwin

package sandbox

import (
	"strings"
	"testing"
)

func TestBuildDarwinProfileBaseline(t *testing.T) {
	p := Policy{Name: "acme", Dir: "/plugins/acme"}
	prof := buildDarwinProfile(p)

	if !strings.Contains(prof, "(deny default)") {
		t.Error("profile must start from deny-default")
	}
	if !strings.Contains(prof, `(allow file-write* (subpath "/plugins/acme"))`) {
		t.Errorf("plugin dir not writable in profile:\n%s", prof)
	}
	if strings.Contains(prof, "(allow network*)") {
		t.Error("network must not be allowed when Policy.Network is empty")
	}
}

func TestBuildDarwinProfileGrantsDeclaredWrites(t *testing.T) {
	p := Policy{Name: "acme", Dir: "/plugins/acme", Writes: []string{"/data/out", "/tmp/cache"}}
	prof := buildDarwinProfile(p)

	for _, w := range p.Writes {
		if !strings.Contains(prof, `(allow file-write* (subpath "`+w+`"))`) {
			t.Errorf("declared write %q not granted:\n%s", w, prof)
		}
	}
}

func TestBuildDarwinProfileNetworkToggle(t *testing.T) {
	p := Policy{Name: "acme", Dir: "/plugins/acme", Network: []string{"api.example.com"}}
	if !strings.Contains(buildDarwinProfile(p), "(allow network*)") {
		t.Error("network should be allowed when Policy.Network is non-empty")
	}
}

// TestBuildDarwinProfileBrowserStaysSandboxed pins the 1B fix: browser plugins
// are no longer bypassed; they get a real (deny default) profile with the
// extra mach/ipc allowances rather than running unsandboxed.
func TestBuildDarwinProfileBrowserStaysSandboxed(t *testing.T) {
	prof := buildDarwinProfile(Policy{Name: "browser", Dir: "/plugins/browser", Browser: true})

	if !strings.Contains(prof, "(deny default)") {
		t.Error("browser profile must still be deny-default (no bypass)")
	}
	for _, want := range []string{"(allow mach*)", "(allow ipc*)"} {
		if !strings.Contains(prof, want) {
			t.Errorf("browser profile missing %q:\n%s", want, prof)
		}
	}
}
