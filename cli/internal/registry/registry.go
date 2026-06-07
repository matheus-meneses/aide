package registry

import (
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

type Registry struct {
	Sources map[string]SourceDef `yaml:"sources"`
}

type SourceDef struct {
	Description string       `yaml:"description"`
	Categories  []string     `yaml:"categories"`
	Fields      []Field      `yaml:"fields"`
	Credentials []Credential `yaml:"credentials"`
}

type Field struct {
	Key      string `yaml:"key"`
	Label    string `yaml:"label"`
	Hint     string `yaml:"hint,omitempty"`
	Required bool   `yaml:"required"`
	Default  string `yaml:"default,omitempty"`
	Type     string `yaml:"type,omitempty"`
}

type Credential struct {
	Key    string `yaml:"key"`
	Label  string `yaml:"label"`
	Hint   string `yaml:"hint,omitempty"`
	Secret bool   `yaml:"secret,omitempty"`
}

func Load() *Registry {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".aide", "registry.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		return &Registry{Sources: map[string]SourceDef{}}
	}

	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return &Registry{Sources: map[string]SourceDef{}}
	}

	if reg.Sources == nil {
		reg.Sources = map[string]SourceDef{}
	}
	return &reg
}

func LoadFrom(path string) *Registry {
	data, err := os.ReadFile(path)
	if err != nil {
		return &Registry{Sources: map[string]SourceDef{}}
	}

	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return &Registry{Sources: map[string]SourceDef{}}
	}

	if reg.Sources == nil {
		reg.Sources = map[string]SourceDef{}
	}
	return &reg
}

func (r *Registry) GetSource(name string) *SourceDef {
	src, ok := r.Sources[name]
	if !ok {
		return nil
	}
	return &src
}

func (r *Registry) ListSources() []string {
	names := make([]string, 0, len(r.Sources))
	for name := range r.Sources {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *Registry) Options() []string {
	names := r.ListSources()
	options := make([]string, 0, len(names))
	for _, name := range names {
		src := r.Sources[name]
		options = append(options, name+" - "+src.Description)
	}
	return options
}
