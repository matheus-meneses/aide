package main

import (
	"aide/cli/internal/plugin"
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Plugin development toolkit (scaffold, test, validate, package)",
}

var devJSON bool

var (
	devNewRuntime     string
	devNewDescription string
	devNewCategories  []string
	devNewCredentials []string
	devNewNetwork     []string
	devNewBrowser     bool
	devNewDir         string
	devNewYes         bool
)

var (
	devTestAction      string
	devTestConfig      string
	devTestSecrets     []string
	devTestInteractive bool
)

var devPackageOut string

var devNewCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Scaffold a new plugin (flag-driven; prompts as fallback)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  devNewExecute,
}

var devTestCmd = &cobra.Command{
	Use:   "test [path]",
	Short: "Run a local plugin without installing it",
	Args:  cobra.MaximumNArgs(1),
	RunE:  devTestExecute,
}

var devValidateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate a plugin manifest and layout",
	Args:  cobra.MaximumNArgs(1),
	RunE:  devValidateExecute,
}

var devPackageCmd = &cobra.Command{
	Use:   "package [path]",
	Short: "Build a release artifact and print its registry index entry",
	Args:  cobra.MaximumNArgs(1),
	RunE:  devPackageExecute,
}

var devSchemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Print the plugin.yaml contract as JSON Schema",
	RunE:  devSchemaExecute,
}

func init() {
	devCmd.PersistentFlags().BoolVar(&devJSON, "json", false, "emit machine-readable JSON output")

	devNewCmd.Flags().StringVar(&devNewRuntime, "runtime", "", "plugin runtime: python or go")
	devNewCmd.Flags().StringVar(&devNewDescription, "description", "", "one-line description")
	devNewCmd.Flags().StringSliceVar(&devNewCategories, "category", nil, "category (repeatable)")
	devNewCmd.Flags().StringArrayVar(&devNewCredentials, "credential", nil, "credential key[:label][:secret] (repeatable)")
	devNewCmd.Flags().StringSliceVar(&devNewNetwork, "network", nil, "allowed outbound host (repeatable)")
	devNewCmd.Flags().BoolVar(&devNewBrowser, "browser", false, "declare a browser (Playwright) capability")
	devNewCmd.Flags().StringVar(&devNewDir, "dir", "", "target directory (default ./<name>)")
	devNewCmd.Flags().BoolVar(&devNewYes, "yes", false, "skip prompts and overwrite confirmation")

	devTestCmd.Flags().StringVar(&devTestAction, "action", "scrape", "action to invoke: scrape, describe, query")
	devTestCmd.Flags().StringVar(&devTestConfig, "config", "", "config as JSON string or @file.json")
	devTestCmd.Flags().StringArrayVar(&devTestSecrets, "secret", nil, "secret key=value (repeatable)")
	devTestCmd.Flags().BoolVar(&devTestInteractive, "interactive", false, "run interactively (browser auth flows)")

	devPackageCmd.Flags().StringVar(&devPackageOut, "out", "dist", "output directory for the artifact")

	devCmd.AddCommand(devNewCmd)
	devCmd.AddCommand(devTestCmd)
	devCmd.AddCommand(devValidateCmd)
	devCmd.AddCommand(devPackageCmd)
	devCmd.AddCommand(devSchemaCmd)
	rootCmd.AddCommand(devCmd)
}

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// ensureRuntimeQuiet prepares the plugin runtime, redirecting build progress
// (pip/go build chatter) to stderr when quiet is set so --json output keeps a
// clean stdout for machine consumers.
func ensureRuntimeQuiet(ctx context.Context, m *plugin.Manifest, quiet bool) error {
	if !quiet {
		return plugin.EnsureRuntime(ctx, m)
	}
	orig := os.Stdout
	os.Stdout = os.Stderr
	defer func() { os.Stdout = orig }()
	return plugin.EnsureRuntime(ctx, m)
}

var allowedCategories = []string{"absence", "approval", "metric", "alert", "task", "event"}

type scaffoldCred struct {
	Key    string
	Label  string
	Secret bool
}

type scaffoldData struct {
	Name          string
	Runtime       string
	Description   string
	IsPython      bool
	ClassName     string
	FirstCategory string
	CategoriesCSV string
	CategoriesPy  string
	NetworkCSV    string
	Credentials   []scaffoldCred
	Browser       bool
}

func devNewExecute(_ *cobra.Command, args []string) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		if isInteractive() && !devNewYes {
			name = promptLine("Plugin name (snake_case): ")
		}
		if name == "" {
			return fmt.Errorf("plugin name is required")
		}
	}

	rt := devNewRuntime
	if rt == "" {
		if isInteractive() && !devNewYes {
			rt = promptLine("Runtime [python/go] (default python): ")
		}
		if rt == "" {
			rt = "python"
		}
	}
	if rt != "python" && rt != "go" {
		return fmt.Errorf("runtime must be 'python' or 'go', got %q", rt)
	}

	categories := devNewCategories
	if len(categories) == 0 {
		categories = []string{"task"}
	}

	creds, err := parseScaffoldCredentials(devNewCredentials)
	if err != nil {
		return err
	}

	data := scaffoldData{
		Name:          name,
		Runtime:       rt,
		Description:   devNewDescription,
		IsPython:      rt == "python",
		ClassName:     toClassName(name),
		FirstCategory: categories[0],
		CategoriesCSV: strings.Join(categories, ", "),
		CategoriesPy:  quoteJoin(categories),
		NetworkCSV:    quoteJoin(devNewNetwork),
		Credentials:   creds,
		Browser:       devNewBrowser,
	}

	targetDir := devNewDir
	if targetDir == "" {
		targetDir = "./" + name
	}
	if entries, statErr := os.ReadDir(targetDir); statErr == nil && len(entries) > 0 {
		if !devNewYes && !(isInteractive() && confirm(fmt.Sprintf("%s is not empty — write into it?", targetDir))) {
			return fmt.Errorf("target directory %s is not empty (use --yes to overwrite)", targetDir)
		}
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating target dir: %w", err)
	}

	files := scaffoldFiles(data)
	created := make([]string, 0, len(files))
	names := make([]string, 0, len(files))
	for fn := range files {
		names = append(names, fn)
	}
	sort.Strings(names)
	for _, fn := range names {
		full := filepath.Join(targetDir, fn)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("creating dir for %s: %w", fn, err)
		}
		if err := os.WriteFile(full, []byte(files[fn]), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", fn, err)
		}
		created = append(created, full)
	}

	if devJSON {
		return printJSON(map[string]any{"ok": true, "dir": targetDir, "created": created})
	}
	fmt.Printf("Scaffolded %s plugin %q in %s\n", rt, name, targetDir)
	for _, c := range created {
		fmt.Printf("  + %s\n", c)
	}
	fmt.Printf("\nNext: edit the scraper, then run\n  aide dev test %s\n", targetDir)
	return nil
}

func parseScaffoldCredentials(raw []string) ([]scaffoldCred, error) {
	creds := make([]scaffoldCred, 0, len(raw))
	for _, r := range raw {
		parts := strings.Split(r, ":")
		if parts[0] == "" {
			return nil, fmt.Errorf("invalid --credential %q (expected key[:label][:secret])", r)
		}
		c := scaffoldCred{Key: parts[0], Label: parts[0]}
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

func devTestExecute(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	m, err := plugin.LoadManifest(abs)
	if err != nil {
		return devTestFail(fmt.Sprintf("loading manifest: %v", err))
	}

	if err := ensureRuntimeQuiet(cmd.Context(), m, devJSON); err != nil {
		return devTestFail(fmt.Sprintf("preparing runtime: %v", err))
	}

	cfgMap, err := parseConfigFlag(devTestConfig)
	if err != nil {
		return devTestFail(err.Error())
	}
	secrets, err := parseSecretFlags(devTestSecrets)
	if err != nil {
		return devTestFail(err.Error())
	}

	req := &plugin.Request{
		Action:  devTestAction,
		Config:  cfgMap,
		Secrets: secrets,
		Context: map[string]any{
			"log_level":  logLevel(),
			"log_format": logFormatValue(),
			"verify_ssl": verifySSLValue(),
		},
	}

	var resp *plugin.Response
	var stderr string
	if devTestInteractive {
		resp, stderr, err = plugin.ExecuteInteractive(cmd.Context(), m, req)
	} else {
		resp, stderr, err = plugin.Execute(cmd.Context(), m, req)
	}

	result := devTestResult{Action: devTestAction, Logs: stderr}
	if err != nil {
		result.OK = false
		result.Error = err.Error()
	} else {
		result.OK = resp.OK
		result.Entries = resp.Entries
		result.TeamMembers = resp.TeamMembers
		result.Metrics = resp.Metrics
		result.Text = resp.Text
		result.Error = resp.Error
	}
	if !result.OK {
		result.ExitCode = 1
	}

	if devJSON {
		_ = printJSON(result)
		if !result.OK {
			os.Exit(1)
		}
		return nil
	}

	if stderr != "" {
		fmt.Fprint(os.Stderr, stderr)
		if !strings.HasSuffix(stderr, "\n") {
			fmt.Fprintln(os.Stderr)
		}
	}
	if !result.OK {
		return fmt.Errorf("plugin returned error: %s", result.Error)
	}
	fmt.Printf("OK  action=%s  entries=%d  team=%d  metrics=%d\n",
		result.Action, len(result.Entries), len(result.TeamMembers), len(result.Metrics))
	for _, e := range result.Entries {
		fmt.Printf("  [%s] %s — %s (%s)\n", e.Category, e.Member, e.Title, e.EntryDate)
	}
	return nil
}

type devTestResult struct {
	OK          bool                `json:"ok"`
	Action      string              `json:"action"`
	Entries     []plugin.Entry      `json:"entries"`
	TeamMembers []plugin.TeamMember `json:"team_members"`
	Metrics     []plugin.Metric     `json:"metrics"`
	Text        string              `json:"text,omitempty"`
	Error       string              `json:"error,omitempty"`
	Logs        string              `json:"logs,omitempty"`
	ExitCode    int                 `json:"exit_code"`
}

func devTestFail(msg string) error {
	if devJSON {
		_ = printJSON(devTestResult{OK: false, Action: devTestAction, Error: msg, ExitCode: 1})
		os.Exit(1)
	}
	return fmt.Errorf("%s", msg)
}

func parseConfigFlag(s string) (map[string]any, error) {
	if s == "" {
		return nil, nil
	}
	data := []byte(s)
	if strings.HasPrefix(s, "@") {
		b, err := os.ReadFile(s[1:])
		if err != nil {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
		data = b
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}
	return m, nil
}

func parseSecretFlags(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid --secret %q (expected key=value)", p)
		}
		m[k] = v
	}
	return m, nil
}

type validationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func devValidateExecute(_ *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	var errs []validationError
	add := func(field, msg string) { errs = append(errs, validationError{Field: field, Message: msg}) }

	m, loadErr := plugin.LoadManifest(abs)
	if loadErr != nil {
		add("manifest", loadErr.Error())
	} else {
		if m.Runtime == "python" {
			if m.Entrypoint.Python.Script == "" {
				add("entrypoint.python.script", "required for python runtime")
			} else if _, statErr := os.Stat(filepath.Join(abs, m.Entrypoint.Python.Script)); statErr != nil {
				add("entrypoint.python.script", fmt.Sprintf("file %s not found", m.Entrypoint.Python.Script))
			}
			if m.Requirements == "" {
				add("requirements", "required for python runtime")
			} else if _, statErr := os.Stat(filepath.Join(abs, m.Requirements)); statErr != nil {
				add("requirements", fmt.Sprintf("file %s not found", m.Requirements))
			}
		}
		if m.Runtime == "go" && m.Entrypoint.Go.Binary == "" {
			add("entrypoint.go.binary", "required for go runtime")
		}
		for _, c := range m.Categories {
			if !contains(allowedCategories, c) {
				add("categories", fmt.Sprintf("%q is not one of %s", c, strings.Join(allowedCategories, ", ")))
			}
		}
	}

	if loadErr == nil && m.Runtime == "python" {
		if ruff, lookErr := exec.LookPath("ruff"); lookErr == nil {
			rc := exec.Command(ruff, "check", abs)
			if out, runErr := rc.CombinedOutput(); runErr != nil {
				add("ruff", strings.TrimSpace(string(out)))
			}
		}
	}

	ok := len(errs) == 0
	if devJSON {
		_ = printJSON(map[string]any{"ok": ok, "errors": errs})
		if !ok {
			os.Exit(1)
		}
		return nil
	}
	if ok {
		fmt.Println("Manifest is valid.")
		return nil
	}
	fmt.Printf("Validation failed (%d issue(s)):\n", len(errs))
	for _, e := range errs {
		fmt.Printf("  - %s: %s\n", e.Field, e.Message)
	}
	return fmt.Errorf("%d validation error(s)", len(errs))
}

func devPackageExecute(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	m, err := plugin.LoadManifest(abs)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	artifactKey := m.Runtime
	if m.Runtime == "go" {
		if err := ensureRuntimeQuiet(cmd.Context(), m, devJSON); err != nil {
			return fmt.Errorf("building go binary: %w", err)
		}
		artifactKey = fmt.Sprintf("go/%s_%s", runtime.GOOS, runtime.GOARCH)
	}

	if err := os.MkdirAll(devPackageOut, 0o755); err != nil {
		return fmt.Errorf("creating out dir: %w", err)
	}
	tarball := filepath.Join(devPackageOut, fmt.Sprintf("%s-%s.tar.gz", m.Name, m.Version))
	manifestAsset := filepath.Join(devPackageOut, fmt.Sprintf("%s-%s.plugin.yaml", m.Name, m.Version))

	if err := createTarGz(abs, tarball); err != nil {
		return fmt.Errorf("packaging: %w", err)
	}
	if err := copyOut(filepath.Join(abs, "plugin.yaml"), manifestAsset); err != nil {
		return fmt.Errorf("copying manifest: %w", err)
	}
	digest, err := sha256File(tarball)
	if err != nil {
		return fmt.Errorf("hashing artifact: %w", err)
	}

	indexEntry := fmt.Sprintf(`plugins:
  %s:
    latest: %s
    description: "%s"
    versions:
      - version: %s
        manifest_url: "https://<host>/%s-%s.plugin.yaml"
        artifacts:
          %s:
            url: "https://<host>/%s-%s.tar.gz"
            sha256: "%s"
`, m.Name, m.Version, m.Description, m.Version, m.Name, m.Version, artifactKey, m.Name, m.Version, digest)

	if devJSON {
		return printJSON(map[string]any{
			"ok":           true,
			"tarball":      tarball,
			"manifest":     manifestAsset,
			"sha256":       digest,
			"artifact_key": artifactKey,
			"index_entry":  indexEntry,
		})
	}
	fmt.Printf("Packaged %s@%s\n  artifact: %s\n  manifest: %s\n  sha256:   %s\n\nRegistry index entry:\n\n%s",
		m.Name, m.Version, tarball, manifestAsset, digest, indexEntry)
	return nil
}

func devSchemaExecute(_ *cobra.Command, _ []string) error {
	schema := map[string]any{
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
				"items": map[string]any{"type": "string", "enum": allowedCategories},
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
	return printJSON(schema)
}

func scaffoldFiles(d scaffoldData) map[string]string {
	files := map[string]string{
		"plugin.yaml": renderTemplate(manifestTmpl, d),
		"AGENTS.md":   renderTemplate(agentsTmpl, d),
	}
	if d.IsPython {
		files["__main__.py"] = renderTemplate(pyMainTmpl, d)
		files["scraper.py"] = renderTemplate(pyScraperTmpl, d)
		files["requirements.txt"] = "aide-sdk\n"
	} else {
		files["main.go"] = renderTemplate(goMainTmpl, d)
		files["go.mod"] = renderTemplate(goModTmpl, d)
	}
	return files
}

func renderTemplate(tmpl string, d scaffoldData) string {
	t := template.Must(template.New("f").Parse(tmpl))
	var b strings.Builder
	if err := t.Execute(&b, d); err != nil {
		return "ERROR: " + err.Error()
	}
	return b.String()
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func createTarGz(srcDir, outPath string) error {
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()
	gw := gzip.NewWriter(out)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if rel == ".venv" || strings.HasPrefix(rel, ".venv"+string(os.PathSeparator)) ||
			strings.Contains(rel, "__pycache__") || strings.HasSuffix(rel, ".pyc") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if info.IsDir() {
			hdr.Name += "/"
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, f)
		f.Close()
		return copyErr
	})
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyOut(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func quoteJoin(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = "\"" + s + "\""
	}
	return strings.Join(quoted, ", ")
}

func toClassName(name string) string {
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

const agentsTmpl = `# AGENTS.md — {{.Name}}

An aide plugin ({{.Runtime}} runtime). The host runs this plugin as a sandboxed
subprocess and exchanges a single JSON object over stdin/stdout.

## Contract

- stdin: ` + "`{ \"action\": \"scrape\", \"config\": {...}, \"secrets\": {...} }`" + `
- stdout: ` + "`{ \"protocol_version\": \"1\", \"ok\": true, \"entries\": [...] }`" + ` or ` + "`{ \"ok\": false, \"error\": \"...\" }`" + `
- **stdout is reserved for the protocol.** Log only via the SDK logger (stderr).
- Declare every outbound host in ` + "`capabilities.network`" + ` — undeclared hosts are blocked.
- Never write secrets to disk or logs.

## Entry shape

` + "```" + `
member: str, category: one of [{{.CategoriesCSV}} ...], title: str,
entry_date: YYYY-MM-DD, priority: info|warning|critical (optional),
detail/link/metadata: optional
` + "```" + `

## Dev loop

` + "```sh" + `
aide dev validate .     # check the manifest
aide dev test .         # run scrape locally and print entries
aide dev test . --json  # machine-readable result
` + "```" + `
`
