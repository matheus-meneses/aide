package plugin

import (
	"aide/cli/internal/testutil"
	"strings"
	"testing"
)

func TestManagerRejectsPathTraversal(t *testing.T) {
	mgr := &Manager{root: t.TempDir()}

	bad := []string{"..", ".", "../etc", "foo/bar", "a/../../b", "", "with space"}
	for _, name := range bad {
		t.Run("get/"+name, func(t *testing.T) {
			if _, err := mgr.Get(name); err == nil || !strings.Contains(err.Error(), "invalid plugin name") {
				t.Fatalf("Get(%q) error = %v, want invalid plugin name", name, err)
			}
		})
		t.Run("remove/"+name, func(t *testing.T) {
			if err := mgr.Remove(name); err == nil || !strings.Contains(err.Error(), "invalid plugin name") {
				t.Fatalf("Remove(%q) error = %v, want invalid plugin name", name, err)
			}
		})
	}
}

func TestManagerValidNameNotInstalled(t *testing.T) {
	mgr := &Manager{root: t.TempDir()}
	// A syntactically valid name that isn't installed must fail with a
	// not-found style error, never the invalid-name guard.
	if err := mgr.Remove("totally-valid_name.v2"); err == nil || strings.Contains(err.Error(), "invalid plugin name") {
		t.Fatalf("Remove of valid-but-missing name error = %v", err)
	}
}

func TestManifestValidate(t *testing.T) {
	tests := []struct {
		name    string
		m       Manifest
		wantErr bool
	}{
		{"valid python", manifest("acme", "1.0.0", "python"), false},
		{"valid go", goManifest("acme", "1.0.0"), false},
		{"missing name", Manifest{Version: "1", Runtime: "python"}, true},
		{"bad name slash", manifest("ac/me", "1.0.0", "python"), true},
		{"bad name dotdot", manifest("..", "1.0.0", "python"), true},
		{"missing version", Manifest{Name: "acme", Runtime: "python"}, true},
		{"bad runtime", manifest("acme", "1.0.0", "ruby"), true},
		{"python missing script", Manifest{Name: "acme", Version: "1", Runtime: "python"}, true},
		{"go missing binary", Manifest{Name: "acme", Version: "1", Runtime: "go"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.m.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParse(t *testing.T) {
	resp, err := Parse([]byte(`{"protocol_version":"1","ok":true,"entries":[{"title":"x"}]}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !resp.OK || len(resp.Entries) != 1 || resp.Entries[0].Title != "x" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	if _, err := Parse([]byte(`{not json`)); err == nil {
		t.Fatal("expected error parsing malformed JSON")
	}
}

func TestLoadManifest(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteManifest(t, dir, `name: acme
version: 1.2.3
runtime: python
entrypoint:
  python:
    script: main.py
`)

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if m.Name != "acme" || m.Version != "1.2.3" || m.Dir != dir {
		t.Fatalf("unexpected manifest: %+v", m)
	}

	bad := t.TempDir()
	testutil.WriteManifest(t, bad, "name: ac/me\nversion: 1\nruntime: python\n")
	if _, err := LoadManifest(bad); err == nil {
		t.Fatal("expected error loading manifest with invalid name")
	}

	if _, err := LoadManifest(t.TempDir()); err == nil {
		t.Fatal("expected error loading manifest from empty dir")
	}
}

func manifest(name, version, rt string) Manifest {
	m := Manifest{Name: name, Version: version, Runtime: rt}
	m.Entrypoint.Python.Script = "main.py"
	return m
}

func goManifest(name, version string) Manifest {
	m := Manifest{Name: name, Version: version, Runtime: "go"}
	m.Entrypoint.Go.Binary = "plugin"
	return m
}
