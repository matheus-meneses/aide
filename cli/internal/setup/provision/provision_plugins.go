package provision

import (
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/plugin"
	"aide/cli/internal/security/keychain"
	"context"
	"fmt"
	"sort"
)

type PluginListItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Runtime     string `json:"runtime,omitempty"`
	Installed   bool   `json:"installed"`
	Configured  bool   `json:"configured"`
}

// ListPlugins merges installed plugins with the cached registry so the UI can
// show what is available and what is already set up.
func ListPlugins(cfgPath string) ([]PluginListItem, error) {
	mgr := plugin.NewManager()
	installed, err := mgr.List()
	if err != nil {
		return nil, err
	}

	configured := map[string]bool{}
	var registries []string
	if cfg, err := config.LoadRaw(cfgPath); err == nil {
		for name := range cfg.Sources {
			configured[name] = true
		}
		registries = cfg.Registries
	}

	items := map[string]*PluginListItem{}
	for _, m := range installed {
		items[m.Name] = &PluginListItem{
			Name:        m.Name,
			Description: m.Description,
			Runtime:     m.Runtime,
			Installed:   true,
			Configured:  configured[m.Name],
		}
	}

	idx, err := plugin.LoadCachedIndex()
	if err != nil || len(idx.Plugins) == 0 {
		if fresh, ferr := plugin.MergedIndex(registries); ferr == nil {
			idx = fresh
			_ = plugin.CacheIndex(fresh)
		}
	}
	if idx != nil {
		for name, entry := range idx.Plugins {
			if _, ok := items[name]; ok {
				continue
			}
			items[name] = &PluginListItem{
				Name:        name,
				Description: entry.Description,
				Installed:   false,
			}
		}
	}

	out := make([]PluginListItem, 0, len(items))
	for _, it := range items {
		out = append(out, *it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// PluginManifest returns the installed plugin manifest so the UI can render its
// config and credential schema.
func PluginManifest(name string) (*plugin.Manifest, error) {
	return plugin.NewManager().Get(name)
}

// InstallPlugin downloads and installs a plugin from the registry into the
// aide home, building its runtime (Python venv) as needed.
func InstallPlugin(ctx context.Context, name, version string) (*plugin.Manifest, error) {
	idx, err := plugin.LoadCachedIndex()
	if err != nil {
		idx, err = plugin.MergedIndex(nil)
		if err != nil {
			return nil, fmt.Errorf("loading registry: %w", err)
		}
	}
	return plugin.Install(ctx, idx, name, version, func(*plugin.Manifest) bool { return true })
}

// UninstallPlugin removes a source (if configured), its stored credentials, and
// the installed plugin directory, so the CLI and app uninstall identically.
func UninstallPlugin(cfgPath, name string) error {
	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return err
	}
	if _, ok := cfg.Sources[name]; ok {
		delete(cfg.Sources, name)
		if err := cfg.Save(cfgPath); err != nil {
			return err
		}
	}
	_ = keychain.DeleteSource(name)

	if err := plugin.NewManager().Remove(name); err != nil {
		return fmt.Errorf("removing plugin %q: %w", name, err)
	}
	return nil
}
