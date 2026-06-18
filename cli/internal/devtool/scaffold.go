// Package devtool holds the plugin development toolkit logic (scaffolding,
// validation, packaging and the manifest schema). The cobra commands under
// cmd/aide/devcmd.go are thin wrappers around this package, keeping CLI
// transport concerns out of the reusable logic.
package devtool

import (
	"strings"
	"text/template"
)

// Cred describes a scaffolded credential declaration.
type Cred struct {
	Key    string
	Label  string
	Secret bool
}

// ScaffoldData is the template context for generating a new plugin.
type ScaffoldData struct {
	Name          string
	Runtime       string
	Description   string
	IsPython      bool
	ClassName     string
	FirstCategory string
	CategoriesCSV string
	CategoriesPy  string
	NetworkCSV    string
	Credentials   []Cred
	Browser       bool
}

// ParseCredentials parses repeatable key[:label][:secret] credential flags.
func ParseCredentials(raw []string) ([]Cred, error) {
	creds := make([]Cred, 0, len(raw))
	for _, r := range raw {
		parts := strings.Split(r, ":")
		if parts[0] == "" {
			return nil, &ParseError{Value: r, Hint: "expected key[:label][:secret]"}
		}
		c := Cred{Key: parts[0], Label: parts[0]}
		if len(parts) >= 2 && parts[1] != "" {
			c.Label = parts[1]
		}
		if len(parts) >= 3 && parts[2] == "secret" {
			c.Secret = true
		}
		creds = append(creds, c)
	}
	return creds, nil
}

// ParseError reports an invalid scaffold flag value.
type ParseError struct {
	Value string
	Hint  string
}

func (e *ParseError) Error() string {
	return "invalid credential " + e.Value + " (" + e.Hint + ")"
}

// NewScaffoldData assembles the template context, deriving the class name and
// the various comma-joined renderings from the inputs.
func NewScaffoldData(name, runtime, description string, categories, network []string, creds []Cred, browser bool) ScaffoldData {
	return ScaffoldData{
		Name:          name,
		Runtime:       runtime,
		Description:   description,
		IsPython:      runtime == "python",
		ClassName:     ToClassName(name),
		FirstCategory: categories[0],
		CategoriesCSV: strings.Join(categories, ", "),
		CategoriesPy:  quoteJoin(categories),
		NetworkCSV:    quoteJoin(network),
		Credentials:   creds,
		Browser:       browser,
	}
}

// ScaffoldFiles renders the file set for a new plugin keyed by relative path.
func ScaffoldFiles(d ScaffoldData) map[string]string {
	files := map[string]string{
		"plugin.yaml": renderTemplate(manifestTmpl, d),
		"AGENTS.md":   renderTemplate(agentsTmpl, d),
	}
	if d.IsPython {
		files["__main__.py"] = renderTemplate(pyMainTmpl, d)
		files["scraper.py"] = renderTemplate(pyScraperTmpl, d)
		files["requirements.txt"] = "aide-plugin-sdk>=0.1.0\n"
	} else {
		files["main.go"] = renderTemplate(goMainTmpl, d)
		files["go.mod"] = renderTemplate(goModTmpl, d)
	}
	return files
}

func renderTemplate(tmpl string, d ScaffoldData) string {
	t := template.Must(template.New("f").Parse(tmpl))
	var b strings.Builder
	if err := t.Execute(&b, d); err != nil {
		return "ERROR: " + err.Error()
	}
	return b.String()
}

func quoteJoin(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = "\"" + s + "\""
	}
	return strings.Join(quoted, ", ")
}

// ToClassName converts a snake/kebab plugin name into a PascalCase scraper class.
func ToClassName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '_' || r == '-' || r == ' ' })
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		b.WriteString(p[1:])
	}
	cls := b.String()
	if cls == "" {
		cls = "Scraper"
	}
	return cls + "Scraper"
}

const manifestTmpl = `name: {{.Name}}
version: 0.1.0
runtime: {{.Runtime}}
description: "{{.Description}}"
icon: ""
categories: [{{.CategoriesCSV}}]
entrypoint:
{{- if .IsPython}}
  python:
    script: __main__.py
{{- else}}
  go:
    binary: {{.Name}}
{{- end}}
{{- if .IsPython}}
requirements: requirements.txt
{{- end}}
{{- if .Credentials}}
credentials:
{{- range .Credentials}}
  - { key: {{.Key}}, label: "{{.Label}}"{{if .Secret}}, secret: true{{end}} }
{{- end}}
{{- end}}
capabilities:
  network: [{{.NetworkCSV}}]
  filesystem: []
{{- if .Browser}}
  browser: true
{{- end}}
`

const pyMainTmpl = `from aide_sdk.runtime import serve

from scraper import {{.ClassName}}

if __name__ == "__main__":
    serve({{.ClassName}})
`

const pyScraperTmpl = `from __future__ import annotations

from datetime import date
from typing import Any, ClassVar

from aide_sdk import BaseScraper, ScraperEntry


class {{.ClassName}}(BaseScraper):
    name = "{{.Name}}"
    version = "0.1.0"
    categories: ClassVar[list[str]] = [{{.CategoriesPy}}]

    def scrape(self, config: dict[str, Any], secrets: dict[str, Any]) -> list[ScraperEntry]:
        self.log.info("scraping {{.Name}}")
        return [
            ScraperEntry(
                member="example",
                category="{{.FirstCategory}}",
                title="Hello from {{.Name}}",
                entry_date=date.today(),
            )
        ]
`

const goMainTmpl = `package main

import sdk "github.com/matheus-meneses/aide-sdk-go"

type handler struct{}

func (handler) Handle(req *sdk.Request) (*sdk.Response, error) {
	sdk.Log.Infof("scraping {{.Name}}")
	return &sdk.Response{
		OK: true,
		Entries: []any{
			map[string]any{
				"member":     "example",
				"category":   "{{.FirstCategory}}",
				"title":      "Hello from {{.Name}}",
				"entry_date": "2026-01-01",
			},
		},
	}, nil
}

func main() { sdk.Serve(handler{}) }
`

const goModTmpl = `module {{.Name}}

go 1.26

require github.com/matheus-meneses/aide-sdk-go v0.1.0
`

const agentsTmpl = "# AGENTS.md — {{.Name}}\n\n" +
	"An aide plugin ({{.Runtime}} runtime). The host runs this plugin as a sandboxed\n" +
	"subprocess and exchanges a single JSON object over stdin/stdout.\n\n" +
	"## Contract\n\n" +
	"- stdin: `{ \"action\": \"scrape\", \"config\": {...}, \"secrets\": {...} }`\n" +
	"- stdout: `{ \"protocol_version\": \"1\", \"ok\": true, \"entries\": [...] }` or `{ \"ok\": false, \"error\": \"...\" }`\n" +
	"- **stdout is reserved for the protocol.** Log only via the SDK logger (stderr).\n" +
	"- Declare every outbound host in `capabilities.network` — undeclared hosts are blocked.\n" +
	"- Never write secrets to disk or logs.\n\n" +
	"## Entry shape\n\n" +
	"```\n" +
	"member: str, category: one of [{{.CategoriesCSV}} ...], title: str,\n" +
	"entry_date: YYYY-MM-DD, priority: info|warning|critical (optional),\n" +
	"detail/link/metadata: optional\n" +
	"```\n\n" +
	"## Dev loop\n\n" +
	"```sh\n" +
	"aide dev validate .     # check the manifest\n" +
	"aide dev test .         # run scrape locally and print entries\n" +
	"aide dev test . --json  # machine-readable result\n" +
	"```\n"
