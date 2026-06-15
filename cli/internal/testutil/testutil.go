// Package testutil provides shared helpers for the Go test suite: scratch
// AIDE_HOME directories, config/manifest fixtures, and ready-to-use stores.
// It is imported only from _test.go files (depguard excludes tests, so the
// import DAG enforced on production code is unaffected).
package testutil

import (
	"aide/cli/internal/persistence/store"
	"os"
	"path/filepath"
	"testing"
)

// TempAideHome points AIDE_HOME at a fresh temp dir for the duration of the
// test and returns the path. xdg.AideHome() and config defaults resolve here.
func TempAideHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("AIDE_HOME", dir)
	return dir
}

// WriteConfig writes yaml to a config.yaml in a fresh temp dir and returns its
// path, suitable for config.Load.
func WriteConfig(t *testing.T, yaml string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

// WriteManifest writes yaml to plugin.yaml inside dir (created if needed) and
// returns the manifest path.
func WriteManifest(t *testing.T, dir, yaml string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, "plugin.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}

// OpenStore opens a SQLite store rooted in a fresh temp dir and registers
// cleanup. Generalizes the openTestStore helper from the store package tests.
func OpenStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
