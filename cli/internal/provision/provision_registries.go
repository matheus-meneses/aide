package provision

import (
	"aide/cli/internal/config"
	"aide/cli/internal/keychain"
	"aide/cli/internal/plugin"
	"fmt"
	"strings"
)

// ListRegistries returns the user-configured plugin registry URLs.
func ListRegistries(cfgPath string) ([]string, error) {
	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return nil, err
	}
	return cfg.Registries, nil
}

// AddRegistry appends a registry URL (deduplicated) and, when a token is given,
// stores it in the keychain so private GitHub registries can be fetched.
func AddRegistry(cfgPath, registryURL, token string) error {
	registryURL = strings.TrimSpace(registryURL)
	if registryURL == "" {
		return fmt.Errorf("registry URL is required")
	}

	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return err
	}
	for _, existing := range cfg.Registries {
		if existing == registryURL {
			if t := strings.TrimSpace(token); t != "" {
				return keychain.SetField(plugin.RegistryTokenService, "token", t)
			}
			return nil
		}
	}
	cfg.Registries = append(cfg.Registries, registryURL)
	if err := cfg.Save(cfgPath); err != nil {
		return err
	}
	if t := strings.TrimSpace(token); t != "" {
		return keychain.SetField(plugin.RegistryTokenService, "token", t)
	}
	return nil
}

// RemoveRegistry drops a registry URL from config.yaml.
func RemoveRegistry(cfgPath, registryURL string) error {
	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return err
	}
	filtered := cfg.Registries[:0]
	found := false
	for _, existing := range cfg.Registries {
		if existing == registryURL {
			found = true
			continue
		}
		filtered = append(filtered, existing)
	}
	if !found {
		return fmt.Errorf("registry %q not configured", registryURL)
	}
	cfg.Registries = filtered
	return cfg.Save(cfgPath)
}

// RefreshCatalog re-fetches the merged plugin index (default + user registries)
// and caches it, returning the number of plugins discovered.
func RefreshCatalog(cfgPath string) (int, error) {
	var registries []string
	if cfg, err := config.LoadRaw(cfgPath); err == nil {
		registries = cfg.Registries
	}
	idx, err := plugin.MergedIndex(registries)
	if err != nil {
		return 0, fmt.Errorf("refreshing catalog: %w", err)
	}
	if err := plugin.CacheIndex(idx); err != nil {
		return 0, fmt.Errorf("caching catalog: %w", err)
	}
	return len(idx.Plugins), nil
}
