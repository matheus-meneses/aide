package agent

import (
	"context"
	"fmt"
	"strings"
)

type Tool struct {
	Name        string
	Description string
	Parameters  string
	Execute     func(ctx context.Context, params map[string]string) (string, error)
}

type ToolRegistry struct {
	tools map[string]*Tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]*Tool)}
}

func (r *ToolRegistry) Register(t *Tool) {
	r.tools[t.Name] = t
}

func (r *ToolRegistry) Get(name string) (*Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *ToolRegistry) Describe() string {
	var b strings.Builder
	for _, t := range r.tools {
		b.WriteString(fmt.Sprintf("- %s: %s", t.Name, t.Description))
		if t.Parameters != "" {
			b.WriteString(fmt.Sprintf(" Params: %s", t.Parameters))
		}
		b.WriteString("\n")
	}
	return b.String()
}
