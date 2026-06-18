package api

import (
	"aide/cli/internal/agent"
	"aide/cli/internal/runtime/updater"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func decodeJSON(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decoding body %q: %v", rr.Body.String(), err)
	}
	return out
}

func TestHandleReady(t *testing.T) {
	rr := httptest.NewRecorder()
	handleReady(rr, httptest.NewRequest(http.MethodGet, "/api/ready", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := decodeJSON(t, rr)
	if ok, _ := body["ok"].(bool); !ok {
		t.Fatalf("ready body = %v, want ok:true", body)
	}
}

func TestHandleVersionDev(t *testing.T) {
	// agent.Version defaults to "dev"; a dev build never reports an update and
	// the call must not block on the network.
	if agent.Version != "dev" {
		t.Skipf("agent.Version = %q, expected dev default", agent.Version)
	}
	rr := httptest.NewRecorder()
	handleVersion(rr, httptest.NewRequest(http.MethodGet, "/api/version", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := decodeJSON(t, rr)
	if body["current"] != "dev" {
		t.Fatalf("current = %v, want dev", body["current"])
	}
	if body["update_available"] != false {
		t.Fatalf("update_available = %v, want false", body["update_available"])
	}
	if body["can_self_update"] != false {
		t.Fatalf("can_self_update = %v, want false for dev", body["can_self_update"])
	}
}

func TestWriteVersionInfoFieldMapping(t *testing.T) {
	info := updater.UpgradeInfo{
		Latest:          "v9.9.9",
		UpdateAvailable: true,
		Notes:           "shiny new things",
		ReleaseURL:      "https://example.test/releases/v9.9.9",
	}
	rr := httptest.NewRecorder()
	writeVersionInfo(rr, info)

	body := decodeJSON(t, rr)
	if body["current"] != agent.Version {
		t.Fatalf("current = %v, want %q", body["current"], agent.Version)
	}
	if body["latest"] != "v9.9.9" {
		t.Fatalf("latest = %v, want v9.9.9", body["latest"])
	}
	if body["update_available"] != true {
		t.Fatalf("update_available = %v, want true", body["update_available"])
	}
	if body["notes"] != "shiny new things" {
		t.Fatalf("notes = %v", body["notes"])
	}
	if body["release_url"] != "https://example.test/releases/v9.9.9" {
		t.Fatalf("release_url = %v", body["release_url"])
	}
	if body["update_url"] != updater.InstallURL() {
		t.Fatalf("update_url = %v, want %q", body["update_url"], updater.InstallURL())
	}
	if body["platform"] != runtime.GOOS+"/"+runtime.GOARCH {
		t.Fatalf("platform = %v", body["platform"])
	}
}
