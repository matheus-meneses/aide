package plugin

import (
	"aide/cli/internal/platform/clog"
	"aide/cli/internal/platform/xdg"
	"aide/cli/internal/security/keychain"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// RegistryTokenService is the keychain "source" under which a private registry
// token is stored (field "token"), enabling access to private GitHub registries.
const RegistryTokenService = "registry"

const defaultRegistryRepo = "matheus-meneses/aide-plugins"

var registryVersionOverride string

func SetRegistryVersion(v string) { registryVersionOverride = v }

func registryRepo() string {
	if r := os.Getenv("AIDE_REGISTRY_REPO"); r != "" {
		return r
	}
	return defaultRegistryRepo
}

func registryVersion() string {
	if registryVersionOverride != "" {
		return registryVersionOverride
	}
	return os.Getenv("AIDE_REGISTRY_VERSION")
}

func DefaultRegistryURL() string {
	if u := os.Getenv("AIDE_REGISTRY_URL"); u != "" {
		return u
	}
	repo := registryRepo()
	if v := registryVersion(); v != "" {
		return fmt.Sprintf("https://github.com/%s/releases/download/%s/index.yaml", repo, v)
	}
	return fmt.Sprintf("https://github.com/%s/releases/latest/download/index.yaml", repo)
}

var registryClient = sync.OnceValue(newRegistryClient)

// newRegistryClient builds the HTTPS client used for registry index and asset
// downloads. TLS verification stays on by default; to reach a self-hosted
// registry served by an internal CA, point AIDE_REGISTRY_CA_BUNDLE at a PEM
// file and those roots are trusted on top of the system pool.
func newRegistryClient() *http.Client {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if bundle := os.Getenv("AIDE_REGISTRY_CA_BUNDLE"); bundle != "" {
		if pem, err := os.ReadFile(bundle); err == nil { //nolint:gosec // G703: path is operator-provided config (env var), not untrusted input
			pool, perr := x509.SystemCertPool()
			if perr != nil || pool == nil {
				pool = x509.NewCertPool()
			}
			if pool.AppendCertsFromPEM(pem) {
				tlsCfg.RootCAs = pool
			} else {
				clog.Warn("AIDE_REGISTRY_CA_BUNDLE %s contained no usable certificates", bundle)
			}
		} else {
			clog.Warn("could not read AIDE_REGISTRY_CA_BUNDLE %s: %v", bundle, err)
		}
	}
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
			Proxy:           http.ProxyFromEnvironment,
		},
	}
}

type Index struct {
	Plugins map[string]PluginEntry `yaml:"plugins"`
}

type PluginEntry struct {
	Latest      string         `yaml:"latest"`
	Description string         `yaml:"description,omitempty"`
	Icon        string         `yaml:"icon,omitempty"`
	Source      string         `yaml:"source,omitempty"`
	Versions    []VersionEntry `yaml:"versions"`
}

const (
	SourceBuiltin = "builtin"
	SourcePrivate = "private"
)

type VersionEntry struct {
	Version     string                 `yaml:"version"`
	ManifestURL string                 `yaml:"manifest_url"`
	Artifacts   map[string]ArtifactRef `yaml:"artifacts"`
}

type ArtifactRef struct {
	URL       string `yaml:"url"`
	SHA256    string `yaml:"sha256"`
	Signature string `yaml:"signature"`
}

func authToken() string {
	if t := os.Getenv("GH_TOKEN"); t != "" {
		return t
	}
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t
	}
	if cred, err := keychain.GetAll(RegistryTokenService); err == nil {
		if t := strings.TrimSpace(cred.Fields["token"]); t != "" {
			return t
		}
	}
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func httpGetAsset(rawURL string) (*http.Response, error) {
	token := authToken()
	getURL := rawURL
	accept := ""
	if token != "" {
		if apiURL, ok := githubAssetAPIURL(rawURL, token); ok {
			getURL = apiURL
			accept = "application/octet-stream"
		}
	}
	req, err := http.NewRequest(http.MethodGet, getURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	return registryClient().Do(req)
}

func githubAssetAPIURL(rawURL, token string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host != "github.com" {
		return "", false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 6 || parts[2] != "releases" {
		return "", false
	}
	owner, repo := parts[0], parts[1]
	file := parts[len(parts)-1]
	var releaseAPI string
	switch {
	case parts[3] == "download":
		releaseAPI = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repo, parts[4])
	case parts[3] == "latest" && parts[4] == "download":
		releaseAPI = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	default:
		return "", false
	}
	assetURL, err := lookupAssetURL(releaseAPI, file, token)
	if err != nil {
		return "", false
	}
	return assetURL, true
}

func lookupAssetURL(releaseAPI, file, token string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, releaseAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := registryClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("release lookup returned HTTP %d", resp.StatusCode)
	}
	var rel struct {
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	for _, a := range rel.Assets {
		if a.Name == file {
			return a.URL, nil
		}
	}
	return "", fmt.Errorf("asset %q not found in release", file)
}

func FetchIndex(registryURL string) (*Index, error) {
	resp, err := httpGetAsset(registryURL)
	if err != nil {
		return nil, fmt.Errorf("fetching index: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading index: %w", err)
	}
	var idx Index
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing index: %w", err)
	}
	if idx.Plugins == nil {
		idx.Plugins = make(map[string]PluginEntry)
	}
	return &idx, nil
}

func MergedIndex(userRegistries []string) (*Index, error) {
	base, err := FetchIndex(DefaultRegistryURL())
	if err != nil {
		if len(userRegistries) == 0 {
			return nil, fmt.Errorf("fetching registry %s: %w", DefaultRegistryURL(), err)
		}
		clog.Warn("skipping default registry %s: %v", DefaultRegistryURL(), err)
		base = &Index{Plugins: make(map[string]PluginEntry)}
	}
	for name, entry := range base.Plugins {
		entry.Source = SourceBuiltin
		base.Plugins[name] = entry
	}
	for _, url := range userRegistries {
		extra, err := FetchIndex(url)
		if err != nil {
			clog.Warn("skipping registry %s: %v", url, err)
			continue
		}
		for name, entry := range extra.Plugins {
			if _, exists := base.Plugins[name]; !exists {
				entry.Source = SourcePrivate
				base.Plugins[name] = entry
			}
		}
	}
	return base, nil
}

// ResolveIndex returns the plugin registry index, preferring a fresh network
// fetch (which it then caches) and falling back to the on-disk cache when the
// network is unavailable. Use it when freshness matters, e.g. installs/updates.
func ResolveIndex(userRegistries []string) (*Index, error) {
	idx, err := MergedIndex(userRegistries)
	if err != nil {
		clog.Warn("registry fetch failed (%v); falling back to cache", err)
		cached, cacheErr := LoadCachedIndex()
		if cacheErr != nil {
			return nil, fmt.Errorf("registry unavailable and no cache: %w", err)
		}
		return cached, nil
	}
	if err := CacheIndex(idx); err != nil {
		clog.Warn("could not cache registry: %v", err)
	}
	return idx, nil
}

// CachedOrFreshIndex returns the cached index when it has entries, otherwise it
// fetches and caches a fresh one. Use it on latency-sensitive paths (listing,
// install) where a slightly stale catalog is acceptable. A cache missing the
// builtin/private source tags is treated as stale and re-merged, so older
// caches self-heal instead of leaving the UI unable to distinguish sources.
func CachedOrFreshIndex(userRegistries []string) (*Index, error) {
	if idx, err := LoadCachedIndex(); err == nil && len(idx.Plugins) > 0 && indexHasSource(idx) {
		return idx, nil
	}
	idx, err := MergedIndex(userRegistries)
	if err != nil {
		return nil, err
	}
	if err := CacheIndex(idx); err != nil {
		clog.Warn("could not cache registry: %v", err)
	}
	return idx, nil
}

func indexHasSource(idx *Index) bool {
	for _, entry := range idx.Plugins {
		if entry.Source != "" {
			return true
		}
	}
	return false
}

func CacheIndex(idx *Index) error {
	path := filepath.Join(xdg.AideHome(), "registry-cache.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}
	data, err := yaml.Marshal(idx)
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

func LoadCachedIndex() (*Index, error) {
	path := filepath.Join(xdg.AideHome(), "registry-cache.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading cache: %w", err)
	}
	var idx Index
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing cache: %w", err)
	}
	if idx.Plugins == nil {
		idx.Plugins = make(map[string]PluginEntry)
	}
	return &idx, nil
}

func verifySHA256(path, expected string) error {
	if expected == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != expected {
		return fmt.Errorf("sha256 mismatch: expected %s got %s", expected, got)
	}
	return nil
}
