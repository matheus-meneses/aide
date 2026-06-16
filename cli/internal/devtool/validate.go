package devtool

import (
	"aide/cli/internal/runtime/plugin"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AllowedCategories enumerates the entry categories a plugin may declare.
var AllowedCategories = []string{"absence", "approval", "metric", "alert", "task", "event"}

// ValidationError is a single manifest/layout problem.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Validate checks a plugin directory's manifest and layout, returning the list
// of problems found (empty means valid). It also runs `ruff check` for Python
// plugins when ruff is on PATH.
func Validate(abs string) []ValidationError {
	var errs []ValidationError
	add := func(field, msg string) { errs = append(errs, ValidationError{Field: field, Message: msg}) }

	m, loadErr := plugin.LoadManifest(abs)
	if loadErr != nil {
		add("manifest", loadErr.Error())
		return errs
	}

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
		if !contains(AllowedCategories, c) {
			add("categories", fmt.Sprintf("%q is not one of %s", c, strings.Join(AllowedCategories, ", ")))
		}
	}

	if m.Runtime == "python" {
		if ruff, lookErr := exec.LookPath("ruff"); lookErr == nil {
			rc := exec.Command(ruff, "check", abs)
			if out, runErr := rc.CombinedOutput(); runErr != nil {
				add("ruff", strings.TrimSpace(string(out)))
			}
		}
	}

	return errs
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
