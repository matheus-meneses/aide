package prompt

import (
	"aide/cli/internal/config"
	"aide/cli/internal/keychain"
	"aide/cli/internal/plugin"
	"fmt"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

func PickPlugin(mgr *plugin.Manager, configured map[string]config.Source) (string, error) {
	manifests, err := mgr.List()
	if err != nil {
		return "", fmt.Errorf("listing plugins: %w", err)
	}

	var names []string
	var choices []Choice
	for _, m := range manifests {
		if _, exists := configured[m.Name]; !exists {
			names = append(names, m.Name)
			choices = append(choices, Choice{Title: m.Name, Desc: m.Description})
		}
	}

	if len(choices) == 0 {
		fmt.Println("All installed plugins are already configured.")
		fmt.Println("\nYour configured sources:")
		for name := range configured {
			fmt.Printf("  - %s\n", name)
		}
		fmt.Println("\nInstall new plugins with:")
		fmt.Println("  aide plugin install <name>")
		return "", fmt.Errorf("nothing to add")
	}

	i, err := Select("Select a plugin to configure", choices)
	if err != nil {
		return "", err
	}
	return names[i], nil
}

// ConfigurePlugin runs the guided configuration for a plugin's declared config
// schema. When existing is non-nil its values are used as defaults, enabling
// reconfiguration of an already-set-up source. Fields of type "object_list" are
// collected as repeated groups of their nested fields.
// PickConfiguredSource prompts the user to choose one of the already-configured
// sources by name.
func PickConfiguredSource(configured map[string]config.Source) (string, error) {
	if len(configured) == 0 {
		return "", fmt.Errorf("no sources configured — run 'aide config source add' first")
	}

	names := make([]string, 0, len(configured))
	for name := range configured {
		names = append(names, name)
	}
	sort.Strings(names)

	choices := make([]Choice, 0, len(names))
	for _, name := range names {
		tag := "enabled"
		if !configured[name].Enabled {
			tag = "disabled"
		}
		choices = append(choices, Choice{Title: name, Tag: tag})
	}

	i, err := Select("Select a source to configure", choices)
	if err != nil {
		return "", err
	}
	return names[i], nil
}

func ConfigurePlugin(m *plugin.Manifest, existing map[string]any) (map[string]any, error) {
	cfg := make(map[string]any)

	for _, field := range m.Config {
		var prev any
		if existing != nil {
			prev = existing[field.Key]
		}

		switch field.Type {
		case "object_list":
			value, ok, err := promptObjectList(field, prev)
			if err != nil {
				return nil, err
			}
			if ok {
				cfg[field.Key] = value
			}
		default:
			value, ok, err := promptScalar(field, prev)
			if err != nil {
				return nil, err
			}
			if ok {
				cfg[field.Key] = value
			}
		}
	}

	return cfg, nil
}

func promptScalar(field plugin.Field, existing any) (string, bool, error) {
	def := field.Default
	if s, ok := existing.(string); ok && s != "" {
		def = s
	}

	if !field.Required && def == "" {
		var confirm bool
		_ = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Configure %s?", field.Label),
			Default: false,
		}, &confirm)
		if !confirm {
			return "", false, nil
		}
	}

	var opts []survey.AskOpt
	if field.Required && def == "" {
		opts = append(opts, survey.WithValidator(survey.Required))
	}

	var value string
	if err := survey.AskOne(&survey.Input{
		Message: field.Label,
		Default: def,
	}, &value, opts...); err != nil {
		return "", false, err
	}

	if value == "" {
		value = def
	}
	if value == "" {
		return "", false, nil
	}
	return value, true, nil
}

func promptObjectList(field plugin.Field, existing any) ([]map[string]any, bool, error) {
	items := coerceObjectList(existing)

	if len(items) > 0 {
		fmt.Printf("\nCurrent %s:\n", field.Label)
		for i, it := range items {
			fmt.Printf("  %d. %s\n", i+1, summarizeEntry(field, it))
		}
		idx, err := Select(fmt.Sprintf("Configure %s", field.Label), []Choice{
			{Title: "Keep these and add more"},
			{Title: "Replace all"},
			{Title: "Keep as-is"},
		})
		if err != nil {
			return nil, false, err
		}
		switch idx {
		case 1:
			items = nil
		case 2:
			return items, len(items) > 0, nil
		}
	} else if !field.Required {
		var confirm bool
		_ = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Configure %s?", field.Label),
			Default: false,
		}, &confirm)
		if !confirm {
			return nil, false, nil
		}
	}

	for {
		if len(items) > 0 {
			var more bool
			_ = survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("Add another %s?", strings.ToLower(field.Label)),
				Default: false,
			}, &more)
			if !more {
				break
			}
		}

		entry := make(map[string]any)
		for _, nf := range field.Fields {
			var opts []survey.AskOpt
			if nf.Required && nf.Default == "" {
				opts = append(opts, survey.WithValidator(survey.Required))
			}
			var v string
			if err := survey.AskOne(&survey.Input{
				Message: nf.Label,
				Default: nf.Default,
			}, &v, opts...); err != nil {
				return nil, false, err
			}
			if v == "" {
				v = nf.Default
			}
			if v != "" {
				entry[nf.Key] = v
			}
		}
		if len(entry) > 0 {
			items = append(items, entry)
		}
	}

	return items, len(items) > 0, nil
}

func summarizeEntry(field plugin.Field, entry map[string]any) string {
	var parts []string
	for _, nf := range field.Fields {
		if v, ok := entry[nf.Key]; ok {
			parts = append(parts, fmt.Sprintf("%s=%v", nf.Key, v))
		}
	}
	if len(parts) == 0 {
		keys := make([]string, 0, len(entry))
		for k := range entry {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s=%v", k, entry[k]))
		}
	}
	return strings.Join(parts, ", ")
}

func coerceObjectList(existing any) []map[string]any {
	raw, ok := existing.([]any)
	if !ok {
		return nil
	}
	var out []map[string]any
	for _, e := range raw {
		if m := coerceStringMap(e); m != nil {
			out = append(out, m)
		}
	}
	return out
}

func coerceStringMap(v any) map[string]any {
	switch m := v.(type) {
	case map[string]any:
		return m
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			out[fmt.Sprintf("%v", k)] = val
		}
		return out
	default:
		return nil
	}
}

func SetupPluginCredentials(m *plugin.Manifest, sourceName string) error {
	if len(m.Credentials) == 0 {
		return nil
	}

	fmt.Println("\nCredentials:")

	for _, cred := range m.Credentials {
		var value string
		var err error

		if cred.Secret {
			err = survey.AskOne(&survey.Password{
				Message: cred.Label,
			}, &value, survey.WithValidator(survey.Required))
		} else {
			err = survey.AskOne(&survey.Input{
				Message: cred.Label,
			}, &value, survey.WithValidator(survey.Required))
		}

		if err != nil {
			return err
		}

		if err := keychain.SetField(sourceName, cred.Key, value); err != nil {
			return fmt.Errorf("storing credential '%s': %w", cred.Key, err)
		}
	}

	return nil
}
