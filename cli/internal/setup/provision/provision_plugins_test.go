package provision

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func registryServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// isolatedHome points AIDE_HOME at a fresh temp dir and short-circuits the
// registry auth shell-out, so plugin/registry helpers stay hermetic.
func isolatedHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("AIDE_HOME", home)
	t.Setenv("GH_TOKEN", "test-token")
	return home
}

func TestListPluginsReturnsRegistryCatalog(t *testing.T) {
	isolatedHome(t)
	base := registryServer(t, `plugins:
  jira:
    latest: 1.0.0
    description: Track Jira issues
    icon: "📋"
    versions:
      - version: 1.0.0
        manifest_url: https://example.test/jira.yaml
`)
	t.Setenv("AIDE_REGISTRY_URL", base.URL)

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	items, err := ListPlugins(cfgPath)
	if err != nil {
		t.Fatalf("ListPlugins: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 catalog item, got %d (%+v)", len(items), items)
	}
	got := items[0]
	if got.Name != "jira" || got.Installed || got.Description != "Track Jira issues" {
		t.Fatalf("unexpected item: %+v", got)
	}
}

func TestUpdatePluginNotInstalled(t *testing.T) {
	isolatedHome(t)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	_, err := UpdatePlugin(context.Background(), cfgPath, "ghost", true)
	if err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("UpdatePlugin error = %v, want not-installed", err)
	}
}

func TestUpdatePluginRequiresAck(t *testing.T) {
	isolatedHome(t)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if _, err := UpdatePlugin(context.Background(), cfgPath, "anything", false); err == nil {
		t.Fatal("expected error when capabilities are not acknowledged")
	}
}

func TestUpdatablEmptyWhenNothingInstalled(t *testing.T) {
	home := isolatedHome(t)
	if err := os.MkdirAll(filepath.Join(home, "plugins"), 0o755); err != nil {
		t.Fatalf("mkdir plugins: %v", err)
	}
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	out, err := Updatable(cfgPath)
	if err != nil {
		t.Fatalf("Updatable: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no updatable plugins, got %+v", out)
	}
}
