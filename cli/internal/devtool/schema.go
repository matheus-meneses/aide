package devtool

// Schema returns the plugin.yaml contract as a JSON Schema (draft-07) document.
func Schema() map[string]any {
	return map[string]any{
		"$schema":              "http://json-schema.org/draft-07/schema#",
		"title":                "aide plugin.yaml",
		"type":                 "object",
		"required":             []string{"name", "version", "runtime", "entrypoint"},
		"additionalProperties": false,
		"properties": map[string]any{
			"name":        map[string]any{"type": "string", "description": "snake_case identifier, matches the directory"},
			"version":     map[string]any{"type": "string"},
			"runtime":     map[string]any{"type": "string", "enum": []string{"python", "go"}},
			"description": map[string]any{"type": "string"},
			"categories": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string", "enum": AllowedCategories},
			},
			"entrypoint": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"python": map[string]any{"type": "object", "properties": map[string]any{"script": map[string]any{"type": "string"}}},
					"go":     map[string]any{"type": "object", "properties": map[string]any{"binary": map[string]any{"type": "string"}}},
				},
			},
			"requirements": map[string]any{"type": "string", "description": "pip requirements file (python only)"},
			"config": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"key"},
					"properties": map[string]any{
						"key":      map[string]any{"type": "string"},
						"label":    map[string]any{"type": "string"},
						"required": map[string]any{"type": "boolean"},
						"default":  map[string]any{"type": "string"},
						"type":     map[string]any{"type": "string", "enum": []string{"string", "integer", "string_list", "object_list"}},
					},
				},
			},
			"credentials": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"key"},
					"properties": map[string]any{
						"key":    map[string]any{"type": "string"},
						"label":  map[string]any{"type": "string"},
						"secret": map[string]any{"type": "boolean"},
					},
				},
			},
			"capabilities": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"network":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"filesystem": map[string]any{"type": "array"},
					"browser":    map[string]any{"type": "boolean"},
				},
			},
			"render": map[string]any{"type": "object", "properties": map[string]any{"custom": map[string]any{"type": "boolean"}}},
			"tools": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":       "object",
					"properties": map[string]any{"name": map[string]any{"type": "string"}, "description": map[string]any{"type": "string"}, "params": map[string]any{"type": "object"}},
				},
			},
		},
		"x-protocol-actions": []string{"describe", "scrape", "render", "query"},
		"x-entry-priorities": []string{"info", "warning", "critical"},
	}
}
