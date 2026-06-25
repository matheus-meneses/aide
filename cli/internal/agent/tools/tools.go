package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type Tool struct {
	Name        string
	Description string
	Parameters  string
	InputSchema json.RawMessage
	Execute     func(ctx context.Context, params map[string]string) (string, error)
}

type Definition struct {
	Name        string
	Description string
	InputSchema json.RawMessage
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

func (r *ToolRegistry) Definitions() []Definition {
	defs := make([]Definition, 0, len(r.tools))
	for _, t := range r.tools {
		schema := t.InputSchema
		if len(schema) == 0 {
			schema = emptyObjectSchema()
		}
		defs = append(defs, Definition{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		})
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })
	return defs
}

func emptyObjectSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

type schemaField struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

func objectSchema(props map[string]string) json.RawMessage {
	schema := struct {
		Type       string                 `json:"type"`
		Properties map[string]schemaField `json:"properties"`
	}{
		Type:       "object",
		Properties: make(map[string]schemaField, len(props)),
	}
	for name, desc := range props {
		schema.Properties[name] = schemaField{Type: "string", Description: desc}
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		return emptyObjectSchema()
	}
	return raw
}

func sourceSchema(desc string, enum []string) json.RawMessage {
	schema := struct {
		Type       string                 `json:"type"`
		Properties map[string]schemaField `json:"properties"`
	}{
		Type: "object",
		Properties: map[string]schemaField{
			"source": {Type: "string", Description: desc, Enum: enum},
		},
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		return emptyObjectSchema()
	}
	return raw
}

func (r *ToolRegistry) Describe() string {
	var b strings.Builder
	for _, t := range r.tools {
		fmt.Fprintf(&b, "- %s: %s", t.Name, t.Description)
		if t.Parameters != "" {
			fmt.Fprintf(&b, " Params: %s", t.Parameters)
		}
		b.WriteString("\n")
	}
	return b.String()
}
