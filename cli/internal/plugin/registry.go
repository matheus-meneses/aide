package plugin

import (
	"aide/cli/internal/xdg"
	"crypto/sha256"
	"crypto/tls"
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
	"time"

	"gopkg.in/yaml.v3"
)

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
	repo := registryRepo()
	if v := registryVersion(); v != "" {
		return fmt.Sprintf("https://github.com/%s/releases/download/%s/index.yaml", repo, v)
	}
	return fmt.Sprintf("https://github.com/%s/releases/latest/download/index.yaml", repo)
}

var registryHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // G402: InsecureSkipVerify supports self-hosted registries at non-public URLs
		Proxy:           http.ProxyFromEnvironment,
	},
}

type Index struct {
	Plugins map[string]PluginEntry `yaml:"plugins"`
}

type PluginEntry struct {
	Latest      string         `yaml:"latest"`
	Description string         `yaml:"description,omitempty"`
	Versions    []VersionEntry `yaml:"versions"`
}

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
	return registryHTTPClient.Do(req)
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
	resp, err := registryHTTPClient.Do(req)
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
		fmt.Fprintf(os.Stderr, "warning: skipping default registry %s: %v\n", DefaultRegistryURL(), err)
		base = &Index{Plugins: make(map[string]PluginEntry)}
	}
	for _, url := range userRegistries {
		extra, err := FetchIndex(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping registry %s: %v\n", url, err)
			continue
		}
		for name, entry := range extra.Plugins {
			if _, exists := base.Plugins[name]; !exists {
				base.Plugins[name] = entry
			}
		}
	}
	return base, nil
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
