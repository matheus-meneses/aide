package plugin

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestCacheIndexRoundTrip(t *testing.T) {
	t.Setenv("AIDE_HOME", t.TempDir())
	want := &Index{Plugins: map[string]PluginEntry{
		"jira": {Latest: "1.0.0", Source: SourceBuiltin, Versions: []VersionEntry{{Version: "1.0.0"}}},
	}}

	if err := CacheIndex(want); err != nil {
		t.Fatalf("CacheIndex: %v", err)
	}
	got, err := LoadCachedIndex()
	if err != nil {
		t.Fatalf("LoadCachedIndex: %v", err)
	}
	entry, ok := got.Plugins["jira"]
	if !ok {
		t.Fatalf("cached index missing jira: %+v", got.Plugins)
	}
	if entry.Latest != "1.0.0" || entry.Source != SourceBuiltin {
		t.Fatalf("round-trip entry mismatch: %+v", entry)
	}
}

func TestLoadCachedIndexMissing(t *testing.T) {
	t.Setenv("AIDE_HOME", t.TempDir())
	if _, err := LoadCachedIndex(); err == nil {
		t.Fatal("expected error loading absent cache, got nil")
	}
}

func TestIndexHasSource(t *testing.T) {
	tagged := &Index{Plugins: map[string]PluginEntry{"a": {Source: SourceBuiltin}}}
	if !indexHasSource(tagged) {
		t.Fatal("indexHasSource = false for tagged index, want true")
	}
	untagged := &Index{Plugins: map[string]PluginEntry{"a": {}}}
	if indexHasSource(untagged) {
		t.Fatal("indexHasSource = true for untagged index, want false")
	}
	if indexHasSource(&Index{Plugins: map[string]PluginEntry{}}) {
		t.Fatal("indexHasSource = true for empty index, want false")
	}
}

func TestCachedOrFreshIndexUsesTaggedCache(t *testing.T) {
	t.Setenv("AIDE_HOME", t.TempDir())
	t.Setenv("GH_TOKEN", "test-token")
	// Point the default registry at a server that fails, proving the tagged
	// cache short-circuits the network entirely.
	t.Setenv("AIDE_REGISTRY_URL", "http://127.0.0.1:0")

	cached := &Index{Plugins: map[string]PluginEntry{
		"jira": {Latest: "3.3.3", Source: SourceBuiltin},
	}}
	if err := CacheIndex(cached); err != nil {
		t.Fatalf("CacheIndex: %v", err)
	}

	idx, err := CachedOrFreshIndex(nil)
	if err != nil {
		t.Fatalf("CachedOrFreshIndex: %v", err)
	}
	if idx.Plugins["jira"].Latest != "3.3.3" {
		t.Fatalf("expected cached entry, got %+v", idx.Plugins["jira"])
	}
}

func TestCachedOrFreshIndexReMergesStaleCache(t *testing.T) {
	t.Setenv("AIDE_HOME", t.TempDir())
	t.Setenv("GH_TOKEN", "test-token")
	base := yamlIndexServer(t, `plugins:
  jira:
    latest: 1.0.0
    versions:
      - version: 1.0.0
        manifest_url: https://example.test/jira.yaml
`)
	t.Setenv("AIDE_REGISTRY_URL", base.URL)

	stale := &Index{Plugins: map[string]PluginEntry{
		"jira": {Latest: "0.0.1"}, // no Source tag => stale
	}}
	if err := CacheIndex(stale); err != nil {
		t.Fatalf("CacheIndex: %v", err)
	}

	idx, err := CachedOrFreshIndex(nil)
	if err != nil {
		t.Fatalf("CachedOrFreshIndex: %v", err)
	}
	jira := idx.Plugins["jira"]
	if jira.Source != SourceBuiltin {
		t.Fatalf("stale cache was not re-merged: source=%q", jira.Source)
	}
	if jira.Latest != "1.0.0" {
		t.Fatalf("re-merged entry should reflect fresh fetch, got %+v", jira)
	}
}

func TestCachedOrFreshIndexFetchesWhenNoCache(t *testing.T) {
	t.Setenv("AIDE_HOME", t.TempDir())
	t.Setenv("GH_TOKEN", "test-token")
	base := yamlIndexServer(t, `plugins:
  internal:
    latest: 2.0.0
    versions:
      - version: 2.0.0
        manifest_url: https://example.test/internal.yaml
`)
	t.Setenv("AIDE_REGISTRY_URL", base.URL)

	idx, err := CachedOrFreshIndex(nil)
	if err != nil {
		t.Fatalf("CachedOrFreshIndex: %v", err)
	}
	if idx.Plugins["internal"].Latest != "2.0.0" {
		t.Fatalf("expected fresh fetch entry, got %+v", idx.Plugins)
	}
	if _, err := LoadCachedIndex(); err != nil {
		t.Fatalf("fresh fetch should have populated the cache: %v", err)
	}
}

func TestVerifySHA256(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.bin")
	content := []byte("registry artifact bytes")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	sum := sha256.Sum256(content)
	digest := hex.EncodeToString(sum[:])

	if err := verifySHA256(path, digest); err != nil {
		t.Fatalf("verifySHA256 with correct digest: %v", err)
	}
	if err := verifySHA256(path, ""); err != nil {
		t.Fatalf("empty expected digest should skip verification: %v", err)
	}
	if err := verifySHA256(path, "deadbeef"); err == nil {
		t.Fatal("verifySHA256 with wrong digest should error")
	}
}

func TestDefaultRegistryURL(t *testing.T) {
	SetRegistryVersion("")
	t.Cleanup(func() { SetRegistryVersion("") })

	t.Setenv("AIDE_REGISTRY_URL", "https://example.test/custom/index.yaml")
	if got := DefaultRegistryURL(); got != "https://example.test/custom/index.yaml" {
		t.Fatalf("explicit override = %q", got)
	}

	t.Setenv("AIDE_REGISTRY_URL", "")
	t.Setenv("AIDE_REGISTRY_REPO", "acme/plugins")
	t.Setenv("AIDE_REGISTRY_VERSION", "v1.2.3")
	if got := DefaultRegistryURL(); got != "https://github.com/acme/plugins/releases/download/v1.2.3/index.yaml" {
		t.Fatalf("pinned version URL = %q", got)
	}

	t.Setenv("AIDE_REGISTRY_VERSION", "")
	if got := DefaultRegistryURL(); got != "https://github.com/acme/plugins/releases/latest/download/index.yaml" {
		t.Fatalf("latest URL = %q", got)
	}
}

func selfSignedCertPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "aide-test-registry"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func registryTLSConfig(t *testing.T, c *http.Client) *tls.Config {
	t.Helper()
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", c.Transport)
	}
	return tr.TLSClientConfig
}

func TestNewRegistryClientSecureByDefault(t *testing.T) {
	t.Setenv("AIDE_REGISTRY_CA_BUNDLE", "")
	cfg := registryTLSConfig(t, newRegistryClient())
	if cfg.InsecureSkipVerify {
		t.Fatal("registry client must verify TLS by default")
	}
	if cfg.RootCAs != nil {
		t.Fatal("RootCAs should be nil (system pool) without a CA bundle")
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Fatalf("MinVersion = %x, want TLS 1.2", cfg.MinVersion)
	}
}

func TestNewRegistryClientLoadsCABundle(t *testing.T) {
	bundle := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(bundle, selfSignedCertPEM(t), 0o600); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	t.Setenv("AIDE_REGISTRY_CA_BUNDLE", bundle)

	cfg := registryTLSConfig(t, newRegistryClient())
	if cfg.InsecureSkipVerify {
		t.Fatal("CA bundle path must not disable verification")
	}
	if cfg.RootCAs == nil {
		t.Fatal("RootCAs should be populated from AIDE_REGISTRY_CA_BUNDLE")
	}
}

func TestNewRegistryClientIgnoresUnreadableBundle(t *testing.T) {
	t.Setenv("AIDE_REGISTRY_CA_BUNDLE", filepath.Join(t.TempDir(), "missing.pem"))
	cfg := registryTLSConfig(t, newRegistryClient())
	if cfg.InsecureSkipVerify {
		t.Fatal("unreadable bundle must not disable verification")
	}
	if cfg.RootCAs != nil {
		t.Fatal("unreadable bundle should leave RootCAs at system default (nil)")
	}
}
