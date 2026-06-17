package plugin

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func yamlIndexServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchIndexParsesPlugins(t *testing.T) {
	t.Setenv("GH_TOKEN", "test-token") // short-circuit authToken's shell-out to gh
	srv := yamlIndexServer(t, `plugins:
  jira:
    latest: 1.2.0
    description: Jira issues
    icon: "📋"
    versions:
      - version: 1.2.0
        manifest_url: https://example.test/jira/plugin.yaml
`)

	idx, err := FetchIndex(srv.URL)
	if err != nil {
		t.Fatalf("FetchIndex: %v", err)
	}
	entry, ok := idx.Plugins["jira"]
	if !ok {
		t.Fatalf("expected jira in index, got %v", idx.Plugins)
	}
	if entry.Latest != "1.2.0" || entry.Icon != "📋" {
		t.Fatalf("unexpected entry: %+v", entry)
	}
}

func TestFetchIndexHTTPError(t *testing.T) {
	t.Setenv("GH_TOKEN", "test-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	if _, err := FetchIndex(srv.URL); err == nil {
		t.Fatal("expected error on HTTP 500, got nil")
	}
}

func TestMergedIndexTagsSources(t *testing.T) {
	t.Setenv("GH_TOKEN", "test-token")
	base := yamlIndexServer(t, `plugins:
  jira:
    latest: 1.0.0
    description: builtin jira
    versions:
      - version: 1.0.0
        manifest_url: https://example.test/jira.yaml
`)
	private := yamlIndexServer(t, `plugins:
  jira:
    latest: 9.9.9
    description: private jira override attempt
    versions:
      - version: 9.9.9
        manifest_url: https://example.test/evil.yaml
  internal:
    latest: 2.0.0
    description: private only
    versions:
      - version: 2.0.0
        manifest_url: https://example.test/internal.yaml
`)
	t.Setenv("AIDE_REGISTRY_URL", base.URL)

	idx, err := MergedIndex([]string{private.URL})
	if err != nil {
		t.Fatalf("MergedIndex: %v", err)
	}

	jira, ok := idx.Plugins["jira"]
	if !ok {
		t.Fatal("expected jira from base registry")
	}
	if jira.Source != SourceBuiltin {
		t.Fatalf("jira source = %q, want %q", jira.Source, SourceBuiltin)
	}
	if jira.Latest != "1.0.0" {
		t.Fatalf("builtin jira was overwritten by private registry: latest=%q", jira.Latest)
	}

	internal, ok := idx.Plugins["internal"]
	if !ok {
		t.Fatal("expected internal from private registry")
	}
	if internal.Source != SourcePrivate {
		t.Fatalf("internal source = %q, want %q", internal.Source, SourcePrivate)
	}
}

func writeTarGz(t *testing.T, entries map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for name, content := range entries {
		hdr := &tar.Header{Name: name, Mode: 0o600, Size: int64(len(content)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header: %v", err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write body: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	path := filepath.Join(t.TempDir(), "artifact.tar.gz")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	return path
}

func TestExtractTarGzRejectsTraversal(t *testing.T) {
	archive := writeTarGz(t, map[string]string{"../escape.txt": "pwned"})
	dest := t.TempDir()
	err := extractTarGz(archive, dest)
	if err == nil || !strings.Contains(err.Error(), "path traversal") {
		t.Fatalf("extractTarGz error = %v, want path traversal rejection", err)
	}
}

func TestExtractTarGzExtractsRegularFile(t *testing.T) {
	archive := writeTarGz(t, map[string]string{"plugin.yaml": "name: demo"})
	dest := t.TempDir()
	if err := extractTarGz(archive, dest); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "plugin.yaml"))
	if err != nil {
		t.Fatalf("reading extracted file: %v", err)
	}
	if string(got) != "name: demo" {
		t.Fatalf("extracted content = %q", string(got))
	}
}
