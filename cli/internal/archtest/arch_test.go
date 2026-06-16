package archtest

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

const internalPrefix = "aide/cli/internal/"

// forbidden maps a concept to the set of other concepts it must never import.
// It mirrors the dependency DAG documented across the AGENTS.md files and the
// depguard rules in .golangci.yml, enforcing them from the real import graph so
// the guardrails fail CI on drift instead of living only in prose.
var forbidden = map[string][]string{
	"platform":     {"persistence", "security", "runtime", "setup", "notification", "agent", "agent/events", "ui"},
	"persistence":  {"platform", "security", "runtime", "setup", "notification", "agent", "agent/events", "ui"},
	"agent/events": {"persistence", "security", "runtime", "setup", "notification", "agent", "ui"},
	"security":     {"persistence", "runtime", "setup", "notification", "agent", "agent/events", "ui"},
	"runtime":      {"setup", "notification", "agent", "agent/events", "ui"},
	"setup":        {"notification", "agent", "agent/events", "ui"},
	"notification": {"persistence", "security", "runtime", "setup", "agent", "ui"},
	"agent":        {"ui"},
	"ui":           {"agent", "setup", "notification"},
}

// concept reduces an internal import path tail (already stripped of
// internalPrefix) to its concept domain. agent/events is a leaf and is kept
// distinct from the rest of agent.
func concept(tail string) string {
	if tail == "agent/events" || strings.HasPrefix(tail, "agent/events/") {
		return "agent/events"
	}
	if i := strings.IndexByte(tail, '/'); i >= 0 {
		return tail[:i]
	}
	return tail
}

func internalDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// thisFile = .../cli/internal/archtest/arch_test.go
	return filepath.Dir(filepath.Dir(thisFile))
}

func TestInternalImportGraph(t *testing.T) {
	root := internalDir(t)
	fset := token.NewFileSet()

	type violation struct{ from, to, file string }
	var violations []violation

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		srcConcept := concept(filepath.ToSlash(filepath.Dir(rel)))
		deny, watched := forbidden[srcConcept]
		if !watched {
			return nil
		}

		f, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if perr != nil {
			t.Errorf("parse %s: %v", rel, perr)
			return nil
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(p, internalPrefix) {
				continue
			}
			target := concept(strings.TrimPrefix(p, internalPrefix))
			if target == srcConcept {
				continue
			}
			for _, banned := range deny {
				if target == banned {
					violations = append(violations, violation{srcConcept, target, rel})
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal tree: %v", err)
	}

	if len(violations) > 0 {
		msgs := make([]string, 0, len(violations))
		for _, v := range violations {
			msgs = append(msgs, v.from+" -> "+v.to+" ("+v.file+")")
		}
		sort.Strings(msgs)
		t.Fatalf("dependency DAG violations (see AGENTS.md):\n  %s", strings.Join(msgs, "\n  "))
	}
}
