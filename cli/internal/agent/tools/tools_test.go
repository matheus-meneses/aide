package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSourceSchemaConstrainsValues(t *testing.T) {
	raw := sourceSchema("optional, one of: jira, confluence", []string{"jira", "confluence"})

	var schema struct {
		Type       string `json:"type"`
		Properties map[string]struct {
			Type string   `json:"type"`
			Enum []string `json:"enum"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	if schema.Type != "object" {
		t.Fatalf("type = %q, want object", schema.Type)
	}
	src, ok := schema.Properties["source"]
	if !ok {
		t.Fatal("missing source property")
	}
	if src.Type != "string" {
		t.Fatalf("source type = %q, want string", src.Type)
	}
	if strings.Join(src.Enum, ",") != "jira,confluence" {
		t.Fatalf("enum = %v, want [jira confluence]", src.Enum)
	}
}

func TestSourceSchemaOmitsEmptyEnum(t *testing.T) {
	raw := sourceSchema("optional, scrape all if omitted", nil)
	if strings.Contains(string(raw), "enum") {
		t.Fatalf("empty enum should be omitted, got %s", raw)
	}
}

func TestDefinitionsAreSortedAndCarrySchema(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&Tool{Name: "scrape", Description: "s", InputSchema: sourceSchema("d", []string{"jira"})})
	reg.Register(&Tool{Name: "done", Description: "d"})

	defs := reg.Definitions()
	if len(defs) != 2 {
		t.Fatalf("want 2 definitions, got %d", len(defs))
	}
	if defs[0].Name != "done" || defs[1].Name != "scrape" {
		t.Fatalf("definitions not sorted by name: %v, %v", defs[0].Name, defs[1].Name)
	}
	if len(defs[0].InputSchema) == 0 || !strings.Contains(string(defs[0].InputSchema), "object") {
		t.Fatalf("done should get a default object schema, got %s", defs[0].InputSchema)
	}
	if !strings.Contains(string(defs[1].InputSchema), "jira") {
		t.Fatalf("scrape schema should carry enum, got %s", defs[1].InputSchema)
	}
}
