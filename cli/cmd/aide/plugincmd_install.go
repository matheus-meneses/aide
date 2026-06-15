package main

import (
	"aide/cli/internal/platform/clog"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/plugin"
	"aide/cli/internal/security/keychain"
	"aide/cli/internal/ui/prompt"
	"aide/cli/internal/ui/widgets"
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

func pluginInstallExecute(_ *cobra.Command, args []string) error {
	if pluginInstallLocal != "" {
		consent := func(m *plugin.Manifest) bool {
			if assumeYes {
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
		if assumeYes {
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

	sp := widgets.NewSpinner("Fetching registry…")
	sp.Start()
	idx, idxErr := plugin.MergedIndex(extraRegistries)
	sp.Stop()
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

	configPath := cfgFile
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
	b, err := term.ReadPassword(int(syscall.Stdin)) //nolint:unconvert // syscall.Stdin is a Handle on Windows; int() keeps this cross-platform
	fmt.Println()
	if err != nil {
		return promptLine("")
	}
	return strings.TrimSpace(string(b))
}
