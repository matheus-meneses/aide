package plugin

import (
	"aide/cli/internal/platform/xdg"
	"fmt"
	"os"
	"path/filepath"
)

// Remove deletes an installed plugin's directory. The name must be a bare
// plugin name (no path separators) to avoid escaping the plugins root.
func (mgr *Manager) Remove(name string) error {
	if name == "" || name != filepath.Base(name) || name == "." || name == ".." {
		return fmt.Errorf("invalid plugin name %q", name)
	}
	dir := filepath.Join(mgr.root, name)
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
			continue
		}
		manifests = append(manifests, m)
	}
	return manifests, nil
}

func (mgr *Manager) Get(name string) (*Manifest, error) {
	dir := filepath.Join(mgr.root, name)
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
