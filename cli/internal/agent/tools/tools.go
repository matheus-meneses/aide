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
	// Parameters is the human-readable parameter hint used by the prompt-JSON
	// fallback and Describe(). InputSchema is the JSON Schema sent to providers
	// via native function-calling.
	Parameters  string
	InputSchema json.RawMessage
	Execute     func(ctx context.Context, params map[string]string) (string, error)
}

// Definition is the provider-neutral description of a tool. The agent converts
// it into the llm package's tool type, so this package stays free of any llm or
// provider dependency.
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

// Definitions returns every registered tool as a provider-neutral Definition,
// sorted by name so the catalog sent to the model is deterministic.
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

// objectSchema builds a JSON Schema object from a map of property name to
// description. All properties are typed as strings, matching the tool Execute
// contract (map[string]string).
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

// sourceSchema builds the scrape tool's object schema with a single "source"
// string property constrained to enum (when non-empty). This lets native
// function-calling reject unknown source names up front instead of relying on
// the description as a hint, so the model cannot invent a non-existent source.
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
