package provision

import (
	"aide/cli/internal/runtime/plugin"
	"reflect"
	"testing"
)

func queriesField() plugin.Field {
	return plugin.Field{
		Key:      "queries",
		Type:     "object_list",
		Required: true,
		Fields: []plugin.Field{
			{Key: "name", Required: true},
			{Key: "jql", Required: true},
			{Key: "mode", Default: "items"},
		},
	}
}

func TestCoerceObjectListFromJSONString(t *testing.T) {
	f := queriesField()
	got := coerceConfigValue(f, `[{"name":"Open","jql":"status = Open"}]`)

	want := []map[string]any{
		{"name": "Open", "jql": "status = Open", "mode": "items"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("coerceConfigValue() = %#v, want %#v", got, want)
	}
}

func TestCoerceObjectListDropsIncompleteRows(t *testing.T) {
	f := queriesField()
	got := coerceConfigValue(f, `[{"name":"NoJQL"},{"name":"Good","jql":"assignee = me"}]`)

	want := []map[string]any{
		{"name": "Good", "jql": "assignee = me", "mode": "items"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("coerceConfigValue() = %#v, want %#v", got, want)
	}
}

func TestCoerceObjectListEmptyIsRequiredEmpty(t *testing.T) {
	f := queriesField()
	for _, in := range []any{"", "[]", nil, `[{"name":"only"}]`} {
		got := coerceConfigValue(f, in)
		if !isEmptyValue(got) {
			t.Fatalf("coerceConfigValue(%v) = %#v, expected empty", in, got)
		}
	}
}

func TestCoerceObjectListFromSlice(t *testing.T) {
	f := queriesField()
	in := []any{
		map[string]any{"name": "Metric", "jql": "project = X", "mode": "metric"},
	}
	got := coerceConfigValue(f, in)

	want := []map[string]any{
		{"name": "Metric", "jql": "project = X", "mode": "metric"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("coerceConfigValue() = %#v, want %#v", got, want)
	}
}
