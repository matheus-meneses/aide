package scrapers

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed all:embedded
var FS embed.FS

func ListAvailable() []string {
	entries, err := fs.ReadDir(FS, "embedded/sources")
	if err != nil {
		return nil
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".py") {
			continue
		}
		if strings.HasPrefix(name, "_") {
			continue
		}
		names = append(names, strings.TrimSuffix(name, ".py"))
	}
	sort.Strings(names)
	return names
}

func ExtractTo(dir string) error {
	return fs.WalkDir(FS, "embedded", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel("embedded", path)
		if rel == "." {
			return nil
		}

		dest := filepath.Join(dir, rel)

		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}

		data, err := FS.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}

		return os.WriteFile(dest, data, 0o644)
	})
}
