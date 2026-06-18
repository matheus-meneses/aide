package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	Name         string       `yaml:"name"`
	Version      string       `yaml:"version"`
	Runtime      string       `yaml:"runtime"`
	Description  string       `yaml:"description"`
	Icon         string       `yaml:"icon,omitempty"`
	Categories   []string     `yaml:"categories"`
	Entrypoint   Entrypoint   `yaml:"entrypoint"`
	Requirements string       `yaml:"requirements"`
	Config       []Field      `yaml:"config"`
	Credentials  []Credential `yaml:"credentials"`
	Capabilities Capabilities `yaml:"capabilities"`
	Render       RenderSpec   `yaml:"render"`
	Tools        []ToolSpec   `yaml:"tools"`

	Dir string `yaml:"-"`
}

type Entrypoint struct {
	Python struct {
		Script string `yaml:"script"`
	} `yaml:"python"`
	Go struct {
		Binary string `yaml:"binary"`
	} `yaml:"go"`
}

type Field struct {
	Key      string  `yaml:"key"`
	Label    string  `yaml:"label"`
	Required bool    `yaml:"required"`
	Default  string  `yaml:"default"`
	Type     string  `yaml:"type"`
	Fields   []Field `yaml:"fields,omitempty"`
}

type Credential struct {
	Key    string `yaml:"key"`
	Label  string `yaml:"label"`
	Secret bool   `yaml:"secret"`
}

type Capabilities struct {
	Network    []string     `yaml:"network"`
	Filesystem []FileAccess `yaml:"filesystem"`
	Browser    bool         `yaml:"browser"`
}

type FileAccess struct {
	Read  string `yaml:"read,omitempty"`
	Write string `yaml:"write,omitempty"`
}

// fsPaths splits the declared filesystem capabilities into read and write
// path lists for the sandbox policy.
func (c Capabilities) fsPaths() (reads, writes []string) {
	for _, f := range c.Filesystem {
		if f.Read != "" {
			reads = append(reads, f.Read)
		}
		if f.Write != "" {
			writes = append(writes, f.Write)
		}
	}
	return reads, writes
}

type RenderSpec struct {
	Custom bool `yaml:"custom"`
}

type ToolSpec struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Params      map[string]string `yaml:"params"`
}

func LoadManifest(dir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "plugin.yaml"))
	if err != nil {
		return nil, fmt.Errorf("reading plugin.yaml in %s: %w", dir, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing plugin.yaml in %s: %w", dir, err)
	}
	m.Dir = dir
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest in %s: %w", dir, err)
	}
	return &m, nil
}

func (m *Manifest) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if !ValidName(m.Name) {
		return fmt.Errorf("name %q must contain only letters, digits, '.', '_' or '-'", m.Name)
	}
	if m.Version == "" {
		return fmt.Errorf("version is required")
	}
	if m.Runtime != "python" && m.Runtime != "go" {
		return fmt.Errorf("runtime must be 'python' or 'go', got %q", m.Runtime)
	}
	if m.Runtime == "python" && m.Entrypoint.Python.Script == "" {
		return fmt.Errorf("entrypoint.python.script is required for python runtime")
	}
	if m.Runtime == "go" && m.Entrypoint.Go.Binary == "" {
		return fmt.Errorf("entrypoint.go.binary is required for go runtime")
	}
	return nil
}
