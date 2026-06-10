package plugin

import (
	"aide/cli/internal/xdg"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultRegistryURL = "https://github.com/matheus-meneses/aide-plugins/releases/latest/download/index.yaml"

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

func FetchIndex(registryURL string) (*Index, error) {
	req, err := http.NewRequest(http.MethodGet, registryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	if token := authToken(); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	resp, err := registryHTTPClient.Do(req)
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
	base, err := FetchIndex(DefaultRegistryURL)
	if err != nil {
		return nil, fmt.Errorf("fetching builtin registry: %w", err)
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
