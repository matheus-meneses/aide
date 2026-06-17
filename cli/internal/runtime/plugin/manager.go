package plugin

import (
	"aide/cli/internal/platform/clog"
	"aide/cli/internal/platform/xdg"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var validPluginName = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// ValidName reports whether name is a bare plugin name safe to join onto the
// plugins root without escaping it (no path separators, not "." or "..").
func ValidName(name string) bool {
	return name != "." && name != ".." && validPluginName.MatchString(name)
}

// safeDir resolves a plugin directory under the plugins root, rejecting any
// name that could escape it via path separators or relative components.
func (mgr *Manager) safeDir(name string) (string, error) {
	if !ValidName(name) {
		return "", fmt.Errorf("invalid plugin name %q", name)
	}
	return filepath.Join(mgr.root, name), nil
}

// Remove deletes an installed plugin's directory. The name must be a bare
// plugin name (no path separators) to avoid escaping the plugins root.
func (mgr *Manager) Remove(name string) error {
	dir, err := mgr.safeDir(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("plugin %q is not installed", name)
		}
		return fmt.Errorf("locating plugin %q: %w", name, err)
	}
	return os.RemoveAll(dir)
}

type Manager struct {
	root string
}

func NewManager() *Manager {
	return &Manager{root: filepath.Join(xdg.AideHome(), "plugins")}
}

func (mgr *Manager) List() ([]*Manifest, error) {
	entries, err := os.ReadDir(mgr.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading plugins dir: %w", err)
	}
	var manifests []*Manifest
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(mgr.root, e.Name())
		m, err := LoadManifest(dir)
		if err != nil {
			clog.Warn("skipping plugin %q: %v", e.Name(), err)
			continue
		}
		manifests = append(manifests, m)
	}
	return manifests, nil
}

func (mgr *Manager) Get(name string) (*Manifest, error) {
	dir, err := mgr.safeDir(name)
	if err != nil {
		return nil, err
	}
	m, err := LoadManifest(dir)
	if err != nil {
		return nil, fmt.Errorf("plugin %q not found: %w", name, err)
	}
	return m, nil
}

func (mgr *Manager) GroupByCategory() (map[string][]*Manifest, error) {
	manifests, err := mgr.List()
	if err != nil {
		return nil, err
	}
	groups := make(map[string][]*Manifest)
	for _, m := range manifests {
		for _, cat := range m.Categories {
			groups[cat] = append(groups[cat], m)
		}
		if len(m.Categories) == 0 {
			groups["uncategorized"] = append(groups["uncategorized"], m)
		}
	}
	return groups, nil
}

func (mgr *Manager) GroupByRuntime() (map[string][]*Manifest, error) {
	manifests, err := mgr.List()
	if err != nil {
		return nil, err
	}
	groups := make(map[string][]*Manifest)
	for _, m := range manifests {
		groups[m.Runtime] = append(groups[m.Runtime], m)
	}
	return groups, nil
}
