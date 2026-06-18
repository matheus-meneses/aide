package updater

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.2.0", "v0.1.0", true},
		{"v0.1.0", "v0.2.0", false},
		{"v0.2.0", "v0.2.0", false},
		{"v0.2.0-rc.9", "v0.2.0-rc.8", true},
		{"v0.2.0-rc.8", "v0.2.0-rc.9", false},
		{"v0.2.0-rc.10", "v0.2.0-rc.9", true},
		{"v0.2.0-rc.8", "v0.2.0-rc.8", false},
		{"v0.2.0", "v0.2.0-rc.9", true},
		{"v0.2.0-rc.9", "v0.2.0", false},
		{"v0.2.0-rc.1", "v0.1.0", true},
		{"v0.1.0", "v0.2.0-rc.1", false},
		{"", "v0.1.0", false},
	}
	for _, c := range cases {
		if got := IsNewer(c.latest, c.current); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestComputeUpgradeInfoDev(t *testing.T) {
	got := computeUpgradeInfo("dev")
	if got != (UpgradeInfo{}) {
		t.Fatalf("computeUpgradeInfo(dev) = %+v, want zero value (no network)", got)
	}
}

func TestRefreshUpgradeInfoAsyncDevNoOp(t *testing.T) {
	upgradeMu.Lock()
	upgradeCache = nil
	upgradeChecked = time.Time{}
	upgradeInFlight = false
	upgradeMu.Unlock()

	RefreshUpgradeInfoAsync("dev")

	if _, ok := CachedUpgradeInfo(); ok {
		t.Fatal("dev build should not populate the upgrade cache")
	}
	upgradeMu.RLock()
	inFlight := upgradeInFlight
	upgradeMu.RUnlock()
	if inFlight {
		t.Fatal("dev build should not start a background refresh")
	}
}

func TestShouldCheckMarkCheckedThrottle(t *testing.T) {
	t.Setenv("AIDE_HOME", t.TempDir())

	if !shouldCheck() {
		t.Fatal("shouldCheck should be true when no throttle file exists")
	}

	markChecked()
	if shouldCheck() {
		t.Fatal("shouldCheck should be false immediately after markChecked")
	}

	// A timestamp older than the throttle window must re-open the gate.
	stale := time.Now().Add(-throttleWindow - time.Hour).Unix()
	path := filepath.Join(aideHome(), throttleFile)
	if err := os.WriteFile(path, []byte(strconv.FormatInt(stale, 10)), 0o600); err != nil {
		t.Fatalf("write stale throttle: %v", err)
	}
	if !shouldCheck() {
		t.Fatal("shouldCheck should be true once the throttle window elapses")
	}
}

func TestDetectMethodDev(t *testing.T) {
	if got := DetectMethod("dev"); got != MethodDev {
		t.Fatalf("DetectMethod(dev) = %q, want %q", got, MethodDev)
	}
	if got := DetectMethod(""); got != MethodDev {
		t.Fatalf("DetectMethod(empty) = %q, want %q", got, MethodDev)
	}
}

func TestInstallURL(t *testing.T) {
	t.Setenv("AIDE_RELEASE_URL", "https://example.test/aide")
	if got := InstallURL(); got != "https://example.test/aide/install.sh" {
		t.Fatalf("InstallURL = %q", got)
	}

	t.Setenv("AIDE_RELEASE_URL", "")
	if got := InstallURL(); got != defaultReleaseBaseURL+"/install.sh" {
		t.Fatalf("default InstallURL = %q", got)
	}
}
