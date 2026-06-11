package main

import (
	"aide/cli/internal/clog"
	"aide/cli/internal/config"
	"aide/cli/internal/keychain"
	"aide/cli/internal/plugin"
	"aide/cli/internal/prompt"
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage aide plugins",
}

var (
	pluginListAvailable  bool
	pluginRegistryURL    string
	pluginRegistryVersion string
)

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins (--available to show registry catalog)",
	RunE:  pluginListExecute,
}

var pluginInstallLocal string
var pluginInstallYes bool

var pluginInstallCmd = &cobra.Command{
	Use:   "install [name[@version]]",
	Short: "Install a plugin from the registry (or --local <path> for local dev)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  pluginInstallExecute,
}

var pluginRemoveYes bool

var pluginRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an installed plugin",
	Args:  cobra.ExactArgs(1),
	RunE:  pluginRemoveExecute,
}

var pluginUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Refresh the registry cache",
	RunE:  pluginUpdateExecute,
}

var pluginAuthCmd = &cobra.Command{
	Use:   "auth <source>",
	Short: "Authenticate a browser-based source interactively",
	Args:  cobra.ExactArgs(1),
	RunE:  pluginAuthExecute,
}

func init() {
	pluginListCmd.Flags().BoolVar(&pluginListAvailable, "available", false, "show available plugins from registry cache")
	pluginInstallCmd.Flags().StringVar(&pluginRegistryURL, "registry", "", "extra registry URL to include in merge")
	pluginInstallCmd.Flags().StringVar(&pluginRegistryVersion, "registry-version", "", "registry release version/tag to pull the index from (default: latest)")
	pluginInstallCmd.Flags().StringVar(&pluginInstallLocal, "local", "", "install from a local directory instead of the registry")
	pluginInstallCmd.Flags().BoolVar(&pluginInstallYes, "yes", false, "skip confirmation prompt (local installs only)")
	pluginUpdateCmd.Flags().StringVar(&pluginRegistryURL, "registry", "", "extra registry URL to include in merge")
	pluginUpdateCmd.Flags().StringVar(&pluginRegistryVersion, "registry-version", "", "registry release version/tag to pull the index from (default: latest)")
	pluginRemoveCmd.Flags().BoolVar(&pluginRemoveYes, "yes", false, "skip confirmation")

	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginRemoveCmd)
	pluginCmd.AddCommand(pluginUpdateCmd)
	pluginCmd.AddCommand(pluginAuthCmd)
	rootCmd.AddCommand(pluginCmd)
}

func pluginListExecute(_ *cobra.Command, _ []string) error {
	if pluginListAvailable {
		return pluginListAvailableExecute()
	}
	return pluginListInstalledExecute()
}

func pluginListInstalledExecute() error {
	mgr := plugin.NewManager()
	groups, err := mgr.GroupByCategory()
	if err != nil {
		return fmt.Errorf("listing plugins: %w", err)
	}

	if len(groups) == 0 {
		fmt.Println("No plugins installed.")
		fmt.Println("Run 'aide plugin install <name>' to install a plugin.")
		return nil
	}

	cats := make([]string, 0, len(groups))
	for c := range groups {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	fmt.Println("Installed plugins:")
	for _, cat := range cats {
		fmt.Printf("\n  [%s]\n", cat)
		for _, m := range groups[cat] {
			fmt.Printf("    %-20s %s  (%s)\n", m.Name, m.Version, m.Runtime)
			if m.Description != "" {
				fmt.Printf("    %s\n", m.Description)
			}
		}
	}
	return nil
}

func pluginListAvailableExecute() error {
	idx, err := plugin.LoadCachedIndex()
	if err != nil {
		return fmt.Errorf("no registry cache found — run 'aide plugin update' to fetch: %w", err)
	}

	if len(idx.Plugins) == 0 {
		fmt.Println("Registry is empty.")
		return nil
	}

	names := make([]string, 0, len(idx.Plugins))
	for n := range idx.Plugins {
		names = append(names, n)
	}
	sort.Strings(names)

	fmt.Println("Available plugins:")
	for _, name := range names {
		entry := idx.Plugins[name]
		fmt.Printf("  %-20s %s\n", name, entry.Latest)
	}
	return nil
}

func pluginInstallExecute(_ *cobra.Command, args []string) error {
	if pluginInstallLocal != "" {
		consent := func(m *plugin.Manifest) bool {
			if pluginInstallYes {
				return true
			}
			fmt.Printf("\nPlugin: %s@%s (local)\n", m.Name, m.Version)
			if m.Description != "" {
				fmt.Printf("Description: %s\n", m.Description)
			}
			return confirm("Install this plugin?")
		}
		m, err := plugin.InstallLocal(context.Background(), pluginInstallLocal, consent)
		if err != nil {
			return err
		}
		if pluginInstallYes {
			fmt.Printf("  [+] %s installed (--yes: skipping config wizard)\n", m.Name)
			return nil
		}
		return runConfigWizard(m)
	}

	nameVersion := ""
	if len(args) > 0 {
		nameVersion = args[0]
	}

	cfg, cfgErr := loadConfig()
	if cfgErr != nil {
		return cfgErr
	}

	extraRegistries := cfg.Registries
	if pluginRegistryURL != "" {
		extraRegistries = append(extraRegistries, pluginRegistryURL)
	}
	if pluginRegistryVersion != "" {
		plugin.SetRegistryVersion(pluginRegistryVersion)
	}

	clog.Info("fetching registry")
	idx, idxErr := plugin.MergedIndex(extraRegistries)
	if idxErr != nil {
		clog.Warn("registry fetch failed (%v), trying cache", idxErr)
		idx, idxErr = plugin.LoadCachedIndex()
		if idxErr != nil {
			return fmt.Errorf("registry unavailable and no cache: %w", idxErr)
		}
	}

	var name, version string
	if nameVersion == "" {
		selected, selErr := selectPluginInteractive(idx)
		if selErr != nil {
			return selErr
		}
		name = selected
	} else {
		name, version, _ = strings.Cut(nameVersion, "@")
	}

	consent := func(m *plugin.Manifest) bool {
		fmt.Printf("\nPlugin: %s@%s\n", m.Name, m.Version)
		if m.Description != "" {
			fmt.Printf("Description: %s\n", m.Description)
		}
		if len(m.Capabilities.Network) > 0 {
			fmt.Printf("Network access: %s\n", strings.Join(m.Capabilities.Network, ", "))
		}
		if len(m.Capabilities.Filesystem) > 0 {
			paths := make([]string, 0, len(m.Capabilities.Filesystem))
			for _, f := range m.Capabilities.Filesystem {
				if f.Read != "" {
					paths = append(paths, "r:"+f.Read)
				}
				if f.Write != "" {
					paths = append(paths, "w:"+f.Write)
				}
			}
			fmt.Printf("Filesystem access: %s\n", strings.Join(paths, ", "))
		}
		return confirm("Install this plugin?")
	}

	m, err := plugin.Install(context.Background(), idx, name, version, consent)
	if err != nil {
		return err
	}
	return runConfigWizard(m)
}

func selectPluginInteractive(idx *plugin.Index) (string, error) {
	if idx == nil || len(idx.Plugins) == 0 {
		return "", fmt.Errorf("no plugins available — run 'aide plugin update' to refresh the registry")
	}
	if !isInteractive() {
		return "", fmt.Errorf("no plugin name given; pass a name (e.g. 'aide plugin install jira') or run in a terminal")
	}

	installed := map[string]bool{}
	if list, err := plugin.NewManager().List(); err == nil {
		for _, m := range list {
			installed[m.Name] = true
		}
	}

	names := make([]string, 0, len(idx.Plugins))
	for n := range idx.Plugins {
		names = append(names, n)
	}
	sort.Strings(names)

	choices := make([]prompt.Choice, len(names))
	for i, n := range names {
		entry := idx.Plugins[n]
		desc := entry.Latest
		if entry.Description != "" {
			desc += " — " + entry.Description
		}
		c := prompt.Choice{Title: n, Desc: desc}
		if installed[n] {
			c.Tag = "installed"
		}
		choices[i] = c
	}

	i, err := prompt.Select("Select a plugin to install", choices)
	if err != nil {
		if errors.Is(err, prompt.ErrCancelled) {
			return "", fmt.Errorf("installation cancelled")
		}
		return "", err
	}
	return names[i], nil
}

func runConfigWizard(m *plugin.Manifest) error {
	if len(m.Config) == 0 && len(m.Credentials) == 0 {
		fmt.Printf("\nPlugin %s installed with no configuration required.\n", m.Name)
		return nil
	}

	fmt.Printf("\n─── Configure %s ───────────────────────────────\n", m.Name)
	fmt.Println("Press Enter to skip optional fields.")
	fmt.Println()

	sourceName := promptLine(fmt.Sprintf("Source name [%s]: ", m.Name))
	if sourceName == "" {
		sourceName = m.Name
	}

	cfgValues := map[string]any{}
	for _, field := range m.Config {
		val := promptField(field, "  ")
		if val != nil {
			cfgValues[field.Key] = val
		} else if field.Required {
			fmt.Printf("  [!] %s is required — you can set it later in config.yaml\n", field.Key)
		}
	}

	if len(m.Credentials) > 0 {
		fmt.Println()
		for _, cred := range m.Credentials {
			label := cred.Label
			if label == "" {
				label = cred.Key
			}
			var val string
			if cred.Secret {
				val = promptSecret(fmt.Sprintf("  %s (secret): ", label))
			} else {
				val = promptLine(fmt.Sprintf("  %s: ", label))
			}
			if val != "" {
				if err := keychain.SetField(sourceName, cred.Key, val); err != nil {
					fmt.Printf("  [!] Could not save %s to keychain: %v\n", cred.Key, err)
					fmt.Printf("  [!] Storing in config.yaml instead (not recommended)\n")
					if cfgValues["credentials"] == nil {
						cfgValues["credentials"] = map[string]string{}
					}
					cfgValues["credentials"].(map[string]string)[cred.Key] = val
				} else {
					fmt.Printf("  [+] %s saved to keychain\n", cred.Key)
				}
			}
		}
	}

	configPath := config.DefaultConfigPath()
	cfg, err := config.LoadRaw(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Sources == nil {
		cfg.Sources = map[string]config.Source{}
	}

	if _, exists := cfg.Sources[sourceName]; exists {
		fmt.Printf("\n  [!] Source %q already exists in config.yaml — overwrite?\n", sourceName)
		if !confirm("Overwrite?") {
			fmt.Println("  Skipped. Run 'aide config source add' to configure manually.")
			return nil
		}
	}

	src := config.Source{
		Enabled: true,
		Config:  cfgValues,
	}
	if m.Name != sourceName {
		src.Plugin = m.Name
	}

	cfg.Sources[sourceName] = src

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("\n  [+] Source %q added to config.yaml\n", sourceName)
	fmt.Printf("      Run: aide run --source %s\n", sourceName)
	return nil
}

func promptLine(prompt string) string {
	fmt.Print(prompt)
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		return ""
	}
	line = strings.TrimSpace(line)
	cleaned := strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, line)
	return cleaned
}

func promptField(field plugin.Field, indent string) any {
	label := field.Label
	if label == "" {
		label = field.Key
	}
	required := ""
	if field.Required {
		required = " (required)"
	}

	switch field.Type {
	case "string_list":
		var items []string
		fmt.Printf("%s%s%s (one per line, empty to stop):\n", indent, label, required)
		for i := 1; ; i++ {
			val := promptLine(fmt.Sprintf("%s  [%d]: ", indent, i))
			if val == "" {
				break
			}
			items = append(items, val)
		}
		if len(items) == 0 {
			return nil
		}
		return items

	case "object_list":
		var items []map[string]any
		fmt.Printf("%s%s%s:\n", indent, label, required)
		if len(field.Fields) > 0 {
			subLabels := make([]string, 0, len(field.Fields))
			for _, f := range field.Fields {
				l := f.Label
				if l == "" {
					l = f.Key
				}
				subLabels = append(subLabels, l)
			}
			fmt.Printf("%s  (each entry: %s)\n", indent, strings.Join(subLabels, " → "))
		}
		for i := 1; ; i++ {
			firstSub := field.Fields[0]
			firstLabel := firstSub.Label
			if firstLabel == "" {
				firstLabel = firstSub.Key
			}
			pivot := promptLine(fmt.Sprintf("%s  [%d] %s (empty to stop): ", indent, i, firstLabel))
			if pivot == "" {
				break
			}
			item := map[string]any{firstSub.Key: pivot}
			for _, sub := range field.Fields[1:] {
				subLabel := sub.Label
				if subLabel == "" {
					subLabel = sub.Key
				}
				hint := ""
				if sub.Default != "" {
					hint = fmt.Sprintf(" [%s]", sub.Default)
				}
				val := promptLine(fmt.Sprintf("%s      %s%s: ", indent, subLabel, hint))
				if val == "" {
					val = sub.Default
				}
				if val != "" {
					item[sub.Key] = val
				}
			}
			items = append(items, item)
		}
		if len(items) == 0 {
			return nil
		}
		return items

	case "integer":
		hint := ""
		if field.Default != "" {
			hint = fmt.Sprintf(" [%s]", field.Default)
		}
		val := promptLine(fmt.Sprintf("%s%s%s%s: ", indent, label, required, hint))
		if val == "" {
			val = field.Default
		}
		if val == "" {
			return nil
		}
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
		return val

	default:
		hint := ""
		if field.Default != "" {
			hint = fmt.Sprintf(" [%s]", field.Default)
		}
		val := promptLine(fmt.Sprintf("%s%s%s%s: ", indent, label, required, hint))
		if val == "" {
			val = field.Default
		}
		if val == "" {
			return nil
		}
		return val
	}
}

func promptSecret(prompt string) string {
	fmt.Print(prompt)
	b, err := term.ReadPassword(syscall.Stdin)
	fmt.Println()
	if err != nil {
		return promptLine("")
	}
	return strings.TrimSpace(string(b))
}

func pluginRemoveExecute(_ *cobra.Command, args []string) error {
	name := args[0]
	if !pluginRemoveYes && !confirm(fmt.Sprintf("Remove plugin '%s'?", name)) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := plugin.Remove(name); err != nil {
		return err
	}
	fmt.Printf("Plugin '%s' removed.\n", name)
	return nil
}

func pluginUpdateExecute(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	extraRegistries := cfg.Registries
	if pluginRegistryURL != "" {
		extraRegistries = append(extraRegistries, pluginRegistryURL)
	}
	if pluginRegistryVersion != "" {
		plugin.SetRegistryVersion(pluginRegistryVersion)
	}

	clog.Info("fetching registry")
	idx, err := plugin.MergedIndex(extraRegistries)
	if err != nil {
		return fmt.Errorf("fetching registry: %w", err)
	}

	if err := plugin.CacheIndex(idx); err != nil {
		return fmt.Errorf("caching index: %w", err)
	}

	clog.Info("registry updated: %d plugins available", len(idx.Plugins))
	return nil
}

func pluginAuthExecute(cmd *cobra.Command, args []string) error {
	sourceName := args[0]
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	src, ok := cfg.Sources[sourceName]
	if !ok {
		return fmt.Errorf("source %q not found in config", sourceName)
	}
	pluginName := src.Plugin
	if pluginName == "" {
		pluginName = sourceName
	}
	mgr := plugin.NewManager()
	m, err := mgr.Get(pluginName)
	if err != nil {
		return fmt.Errorf("loading plugin %q: %w", pluginName, err)
	}
	if !m.Capabilities.Browser {
		return fmt.Errorf("plugin %q does not use a browser — no auth flow needed", pluginName)
	}
	secrets, err := plugin.ScopedSecrets(sourceName, m)
	if err != nil {
		return fmt.Errorf("loading secrets: %w", err)
	}
	req := &plugin.Request{
		Action:  "scrape",
		Config:  src.Config,
		Secrets: secrets,
	}
	fmt.Printf("Opening browser for %s authentication...\n", sourceName)
	fmt.Println("Complete the login in the browser window, then return here.")
	_, stderr, err := plugin.ExecuteInteractive(cmd.Context(), m, req)
	if stderr != "" {
		fmt.Print(stderr)
	}
	if err != nil {
		return fmt.Errorf("auth failed: %w", err)
	}
	fmt.Printf("Authentication for %s saved successfully.\n", sourceName)
	return nil
}
