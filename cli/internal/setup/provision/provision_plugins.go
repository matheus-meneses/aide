package provision

import (
	"aide/cli/internal/platform/clog"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/plugin"
	"aide/cli/internal/runtime/updater"
	"aide/cli/internal/security/keychain"
	"context"
	"fmt"
	"sort"
)

type PluginListItem struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	Runtime          string `json:"runtime,omitempty"`
	Icon             string `json:"icon,omitempty"`
	Source           string `json:"source,omitempty"`
	Installed        bool   `json:"installed"`
	Configured       bool   `json:"configured"`
	InstalledVersion string `json:"installed_version,omitempty"`
	LatestVersion    string `json:"latest_version,omitempty"`
	UpdateAvailable  bool   `json:"update_available"`
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

	idx, err := plugin.CachedOrFreshIndex(registries)
	if err != nil {
		clog.Warn("could not resolve plugin registry: %v", err)
		idx = nil
	}

	items := map[string]*PluginListItem{}
	for _, m := range installed {
		item := &PluginListItem{
			Name:             m.Name,
			Description:      m.Description,
			Runtime:          m.Runtime,
			Icon:             m.Icon,
			Installed:        true,
			Configured:       configured[m.Name],
			InstalledVersion: m.Version,
		}
		if idx != nil {
			if entry, ok := idx.Plugins[m.Name]; ok {
				item.Source = entry.Source
				item.LatestVersion = entry.Latest
				item.UpdateAvailable = entry.Latest != "" && updater.IsNewer(entry.Latest, m.Version)
				if item.Icon == "" {
					item.Icon = entry.Icon
				}
			}
		}
		items[m.Name] = item
	}

	if idx != nil {
		for name, entry := range idx.Plugins {
			if _, ok := items[name]; ok {
				continue
			}
			items[name] = &PluginListItem{
				Name:        name,
				Description: entry.Description,
				Icon:        entry.Icon,
				Source:      entry.Source,
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
// aide home, building its runtime (Python venv) as needed. The caller must set
// ackCapabilities to confirm the user has acknowledged the plugin's declared
// network/filesystem capabilities; installation is refused otherwise.
func InstallPlugin(ctx context.Context, cfgPath, name, version string, ackCapabilities bool) (*plugin.Manifest, error) {
	if !ackCapabilities {
		return nil, fmt.Errorf("plugin capabilities must be acknowledged before install")
	}
	var registries []string
	if cfg, err := config.LoadRaw(cfgPath); err == nil {
		registries = cfg.Registries
	}
	idx, err := plugin.CachedOrFreshIndex(registries)
	if err != nil {
		return nil, fmt.Errorf("loading registry: %w", err)
	}
	return plugin.Install(ctx, idx, name, version, func(*plugin.Manifest) bool { return true })
}

// UpdatePlugin re-installs the plugin at the registry's latest version,
// rebuilding its runtime. Config and stored credentials are untouched (they
// live in config.yaml and the keychain, not the plugin directory). It refuses
// when the plugin is not installed or already at the latest version. Because an
// update can change a plugin's declared capabilities, callers must confirm via
// ackCapabilities, exactly as for a fresh install.
func UpdatePlugin(ctx context.Context, cfgPath, name string, ackCapabilities bool) (*plugin.Manifest, error) {
	if !ackCapabilities {
		return nil, fmt.Errorf("plugin capabilities must be acknowledged before update")
	}
	m, err := plugin.NewManager().Get(name)
	if err != nil {
		return nil, fmt.Errorf("plugin %q is not installed", name)
	}
	installedVersion := m.Version

	var registries []string
	if cfg, err := config.LoadRaw(cfgPath); err == nil {
		registries = cfg.Registries
	}
	idx, err := plugin.ResolveIndex(registries)
	if err != nil {
		return nil, err
	}

	entry, ok := idx.Plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin %q not found in any registry", name)
	}
	if !updater.IsNewer(entry.Latest, installedVersion) {
		return nil, fmt.Errorf("plugin %q is already up to date (%s)", name, installedVersion)
	}
	return plugin.Install(ctx, idx, name, entry.Latest, func(*plugin.Manifest) bool { return true })
}

// Updatable returns the installed plugins that have a newer version available
// in the configured registries, so callers can update all at once.
func Updatable(cfgPath string) ([]PluginListItem, error) {
	items, err := ListPlugins(cfgPath)
	if err != nil {
		return nil, err
	}
	var out []PluginListItem
	for _, it := range items {
		if it.UpdateAvailable {
			out = append(out, it)
		}
	}
	return out, nil
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
	if err := keychain.DeleteSource(name); err != nil {
		clog.Warn("could not delete credentials for %q: %v", name, err)
	}

	if err := plugin.NewManager().Remove(name); err != nil {
		return fmt.Errorf("removing plugin %q: %w", name, err)
	}
	return nil
}
