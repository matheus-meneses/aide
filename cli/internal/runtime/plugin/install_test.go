package plugin

import (
	"context"
	"strings"
	"testing"
)

func TestInstallRejectsInvalidName(t *testing.T) {
	idx := &Index{Plugins: map[string]PluginEntry{}}
	_, err := Install(context.Background(), idx, "../escape", "", nil)
	if err == nil || !strings.Contains(err.Error(), "invalid plugin name") {
		t.Fatalf("Install with invalid name error = %v, want invalid plugin name", err)
	}
}

func TestInstallUnknownPlugin(t *testing.T) {
	idx := &Index{Plugins: map[string]PluginEntry{}}
	_, err := Install(context.Background(), idx, "ghost", "", nil)
	if err == nil || !strings.Contains(err.Error(), "not found in registry") {
		t.Fatalf("Install of unknown plugin error = %v, want not found in registry", err)
	}
}

func TestInstallMissingVersion(t *testing.T) {
	idx := &Index{Plugins: map[string]PluginEntry{
		"jira": {
			Latest:   "1.0.0",
			Versions: []VersionEntry{{Version: "1.0.0", ManifestURL: "https://example.test/jira.yaml"}},
		},
	}}
	_, err := Install(context.Background(), idx, "jira", "9.9.9", nil)
	if err == nil || !strings.Contains(err.Error(), "version \"9.9.9\" not found") {
		t.Fatalf("Install of missing version error = %v, want version not found", err)
	}
}

func TestInstallMissingManifestURL(t *testing.T) {
	idx := &Index{Plugins: map[string]PluginEntry{
		"jira": {
			Latest:   "1.0.0",
			Versions: []VersionEntry{{Version: "1.0.0"}},
		},
	}}
	_, err := Install(context.Background(), idx, "jira", "", nil)
	if err == nil || !strings.Contains(err.Error(), "no manifest_url") {
		t.Fatalf("Install with no manifest_url error = %v, want no manifest_url", err)
	}
}
