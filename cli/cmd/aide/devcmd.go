package main

import (
	"aide/cli/internal/devtool"
	"aide/cli/internal/runtime/plugin"
	"aide/cli/internal/runtime/runner"
	"aide/cli/internal/ui/widgets"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Plugin development toolkit (scaffold, test, validate, package)",
}

func devTestCABundle() string {
	if cb := caBundleValue(); cb != "" {
		return cb
	}
	if verifySSLValue() {
		return runner.SystemTrustBundle()
	}
	return ""
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

func devNewExecute(_ *cobra.Command, args []string) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		if stdinIsTerminal() && !assumeYes {
			name = promptLine("Plugin name (snake_case): ")
		}
		if name == "" {
			return fmt.Errorf("plugin name is required")
		}
	}

	rt := devNewRuntime
	if rt == "" {
		if stdinIsTerminal() && !assumeYes {
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

	creds, err := devtool.ParseCredentials(devNewCredentials)
	if err != nil {
		return err
	}

	data := devtool.NewScaffoldData(name, rt, devNewDescription, categories, devNewNetwork, creds, devNewBrowser)

	targetDir := devNewDir
	if targetDir == "" {
		targetDir = "./" + name
	}
	if entries, statErr := os.ReadDir(targetDir); statErr == nil && len(entries) > 0 {
		if !assumeYes && (!stdinIsTerminal() || !confirm(fmt.Sprintf("%s is not empty — write into it?", targetDir))) {
			return fmt.Errorf("target directory %s is not empty (use --yes to overwrite)", targetDir)
		}
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating target dir: %w", err)
	}

	files := devtool.ScaffoldFiles(data)
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
		if err := os.WriteFile(full, []byte(files[fn]), 0o600); err != nil {
			return fmt.Errorf("writing %s: %w", fn, err)
		}
		created = append(created, full)
	}

	if devJSON {
		return printJSON(map[string]any{"ok": true, "dir": targetDir, "created": created})
	}
	widgets.Printf("Scaffolded %s plugin %q in %s\n", rt, name, targetDir)
	for _, c := range created {
		widgets.Printf("  + %s\n", c)
	}
	widgets.Printf("\nNext: edit the scraper, then run\n  aide dev test %s\n", targetDir)
	return nil
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
			"ca_bundle":  devTestCABundle(),
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
	widgets.Printf("OK  action=%s  entries=%d  team=%d  metrics=%d\n",
		result.Action, len(result.Entries), len(result.TeamMembers), len(result.Metrics))
	for _, e := range result.Entries {
		widgets.Printf("  [%s] %s — %s (%s)\n", e.Category, e.Member, e.Title, e.EntryDate)
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

func devValidateExecute(_ *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	errs := devtool.Validate(abs)
	ok := len(errs) == 0
	if devJSON {
		_ = printJSON(map[string]any{"ok": ok, "errors": errs})
		if !ok {
			os.Exit(1)
		}
		return nil
	}
	if ok {
		widgets.Println("Manifest is valid.")
		return nil
	}
	widgets.Printf("Validation failed (%d issue(s)):\n", len(errs))
	for _, e := range errs {
		widgets.Printf("  - %s: %s\n", e.Field, e.Message)
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

	res, err := devtool.BuildPackage(abs, devPackageOut, m, artifactKey)
	if err != nil {
		return err
	}

	if devJSON {
		return printJSON(map[string]any{
			"ok":           true,
			"tarball":      res.Tarball,
			"manifest":     res.Manifest,
			"sha256":       res.SHA256,
			"artifact_key": res.ArtifactKey,
			"index_entry":  res.IndexEntry,
		})
	}
	widgets.Printf("Packaged %s@%s\n  artifact: %s\n  manifest: %s\n  sha256:   %s\n\nRegistry index entry:\n\n%s",
		m.Name, m.Version, res.Tarball, res.Manifest, res.SHA256, res.IndexEntry)
	return nil
}

func devSchemaExecute(_ *cobra.Command, _ []string) error {
	return printJSON(devtool.Schema())
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
