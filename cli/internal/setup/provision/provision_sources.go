package provision

import (
	"aide/cli/internal/platform/clog"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/plugin"
	"aide/cli/internal/security/keychain"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type SourceInput struct {
	Name        string            `json:"name"`
	Config      map[string]any    `json:"config"`
	Credentials map[string]string `json:"credentials"`
}

// AddSource validates against the installed plugin manifest, stores credentials
// in the OS keychain, and writes the source into config.yaml.
func AddSource(cfgPath string, in SourceInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("source name is required")
	}

	m, err := plugin.NewManager().Get(in.Name)
	if err != nil {
		return fmt.Errorf("plugin %q is not installed", in.Name)
	}

	typed := make(map[string]any, len(in.Config))
	known := make(map[string]plugin.Field, len(m.Config))
	for _, f := range m.Config {
		known[f.Key] = f
		coerced := coerceConfigValue(f, in.Config[f.Key])
		if isEmptyValue(coerced) {
			if f.Required {
				return fmt.Errorf("missing required config field %q", f.Key)
			}
			continue
		}
		typed[f.Key] = coerced
	}
	for k, v := range in.Config {
		if _, ok := known[k]; ok {
			continue
		}
		typed[k] = v
	}

	for _, c := range m.Credentials {
		if hasStoredCredential(in.Name, c.Key) {
			continue
		}
		if v, ok := in.Credentials[c.Key]; !ok || strings.TrimSpace(v) == "" {
			return fmt.Errorf("missing required credential %q", c.Key)
		}
	}

	if err := SetCredentials(in.Name, in.Credentials); err != nil {
		return err
	}

	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return err
	}
	if cfg.Sources == nil {
		cfg.Sources = make(map[string]config.Source)
	}

	existing := cfg.Sources[in.Name]
	existing.Enabled = true
	existing.Config = typed
	cfg.Sources[in.Name] = existing
	return cfg.Save(cfgPath)
}

// hasStoredCredential reports whether a credential field is already saved in the
// keychain, so reconfiguring a source does not force re-entering secrets.
func hasStoredCredential(source, key string) bool {
	cred, err := keychain.GetAll(source)
	if err != nil {
		return false
	}
	v, ok := cred.Fields[key]
	return ok && strings.TrimSpace(v) != ""
}

// coerceConfigValue converts a UI- or API-supplied value into the type declared
// by the plugin manifest field, so config.yaml honors the plugin contract
// (integers stay integers, string_list becomes a real list, etc.).
func coerceConfigValue(f plugin.Field, v any) any {
	switch f.Type {
	case "integer":
		switch x := v.(type) {
		case nil:
			return coerceConfigValue(f, f.Default)
		case int:
			return x
		case float64:
			return int(x)
		case string:
			s := strings.TrimSpace(x)
			if s == "" {
				if f.Default == "" {
					return nil
				}
				s = f.Default
			}
			if n, err := strconv.Atoi(s); err == nil {
				return n
			}
			return s
		default:
			return x
		}
	case "string_list":
		return toStringList(v)
	case "object_list":
		return toObjectList(f, v)
	default:
		switch x := v.(type) {
		case nil:
			if f.Default != "" {
				return f.Default
			}
			return nil
		case string:
			return strings.TrimSpace(x)
		default:
			return x
		}
	}
}

// toObjectList normalizes an object_list config value into a slice of maps. The
// web UI submits the value as a JSON-encoded string, while the CLI and existing
// config.yaml supply it as a real slice; both are accepted. Each row is trimmed
// against the field's declared sub-fields, defaults are applied, and rows that
// are missing a required sub-field are dropped.
func toObjectList(f plugin.Field, v any) []map[string]any {
	var rows []map[string]any
	switch x := v.(type) {
	case nil:
		return nil
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil
		}
		if err := json.Unmarshal([]byte(s), &rows); err != nil {
			return nil
		}
	case []map[string]any:
		rows = x
	case []any:
		for _, e := range x {
			if m, ok := e.(map[string]any); ok {
				rows = append(rows, m)
			}
		}
	default:
		return nil
	}

	if len(f.Fields) == 0 {
		return rows
	}

	var out []map[string]any
	for _, row := range rows {
		if norm := normalizeObjectRow(f, row); norm != nil {
			out = append(out, norm)
		}
	}
	return out
}

func normalizeObjectRow(f plugin.Field, row map[string]any) map[string]any {
	out := make(map[string]any, len(f.Fields))
	for _, sub := range f.Fields {
		s := ""
		if raw, ok := row[sub.Key]; ok && raw != nil {
			s = strings.TrimSpace(fmt.Sprintf("%v", raw))
		}
		if s == "" {
			s = strings.TrimSpace(sub.Default)
		}
		if s == "" {
			if sub.Required {
				return nil
			}
			continue
		}
		out[sub.Key] = s
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func toStringList(v any) []string {
	var out []string
	switch x := v.(type) {
	case nil:
		return nil
	case []string:
		for _, e := range x {
			if s := strings.TrimSpace(e); s != "" {
				out = append(out, s)
			}
		}
	case []any:
		for _, e := range x {
			if s := strings.TrimSpace(fmt.Sprintf("%v", e)); s != "" {
				out = append(out, s)
			}
		}
	case string:
		for _, part := range strings.FieldsFunc(x, func(r rune) bool { return r == '\n' || r == ',' }) {
			if s := strings.TrimSpace(part); s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

func isEmptyValue(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(x) == ""
	case []string:
		return len(x) == 0
	case []any:
		return len(x) == 0
	case []map[string]any:
		return len(x) == 0
	}
	return false
}

// RemoveSource deletes a source from config.yaml and its stored credentials.
func RemoveSource(cfgPath, name string) error {
	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return err
	}
	if _, ok := cfg.Sources[name]; !ok {
		return fmt.Errorf("source %q not configured", name)
	}
	delete(cfg.Sources, name)
	if err := cfg.Save(cfgPath); err != nil {
		return err
	}
	if err := keychain.DeleteSource(name); err != nil {
		clog.Warn("could not delete credentials for %q: %v", name, err)
	}
	return nil
}

type SourceSnapshot struct {
	Name           string         `json:"name"`
	Plugin         string         `json:"plugin,omitempty"`
	Enabled        bool           `json:"enabled"`
	Config         map[string]any `json:"config,omitempty"`
	HasCredentials bool           `json:"has_credentials"`
}

// ListSources returns each configured source with its enabled flag, config
// values, and whether credentials are stored, without exposing secret values.
func ListSources(cfgPath string) ([]SourceSnapshot, error) {
	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return nil, err
	}
	out := make([]SourceSnapshot, 0, len(cfg.Sources))
	for name, src := range cfg.Sources {
		cred, _ := keychain.GetAll(name)
		out = append(out, SourceSnapshot{
			Name:           name,
			Plugin:         src.Plugin,
			Enabled:        src.Enabled,
			Config:         src.Config,
			HasCredentials: cred != nil && len(cred.Fields) > 0,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// SetSourceEnabled toggles whether a configured source participates in scrapes.
func SetSourceEnabled(cfgPath, name string, enabled bool) error {
	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return err
	}
	src, ok := cfg.Sources[name]
	if !ok {
		return fmt.Errorf("source %q not configured", name)
	}
	src.Enabled = enabled
	cfg.Sources[name] = src
	return cfg.Save(cfgPath)
}

func SetCredentials(source string, fields map[string]string) error {
	for key, val := range fields {
		if strings.TrimSpace(val) == "" {
			continue
		}
		if err := keychain.SetField(source, key, val); err != nil {
			return fmt.Errorf("storing credential %q: %w", key, err)
		}
	}
	return nil
}
