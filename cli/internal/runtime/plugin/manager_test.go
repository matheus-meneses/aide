package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func writePluginDir(t *testing.T, root, name, manifest string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func TestManagerListEmptyRoot(t *testing.T) {
	mgr := &Manager{root: filepath.Join(t.TempDir(), "does-not-exist")}
	manifests, err := mgr.List()
	if err != nil {
		t.Fatalf("List on absent root: %v", err)
	}
	if len(manifests) != 0 {
		t.Fatalf("expected no manifests, got %d", len(manifests))
	}
}

func TestManagerListSkipsCorruptManifests(t *testing.T) {
	root := t.TempDir()
	writePluginDir(t, root, "good", `name: good
version: 1.0.0
runtime: python
entrypoint:
  python:
    script: main.py
`)
	writePluginDir(t, root, "broken-yaml", "::: not valid yaml :::\n")
	writePluginDir(t, root, "missing-runtime", `name: missing-runtime
version: 1.0.0
`)
	// A stray non-directory entry must be ignored, not treated as a plugin.
	if err := os.WriteFile(filepath.Join(root, "README.txt"), []byte("hi"), 0o600); err != nil {
		t.Fatalf("write stray file: %v", err)
	}

	mgr := &Manager{root: root}
	manifests, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(manifests) != 1 {
		t.Fatalf("expected only the valid plugin, got %d: %+v", len(manifests), manifests)
	}
	if manifests[0].Name != "good" {
		t.Fatalf("listed plugin = %q, want good", manifests[0].Name)
	}
}
