package devtool

import (
	"aide/cli/internal/runtime/plugin"
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PackageResult describes the artifacts produced by BuildPackage.
type PackageResult struct {
	Tarball     string `json:"tarball"`
	Manifest    string `json:"manifest"`
	SHA256      string `json:"sha256"`
	ArtifactKey string `json:"artifact_key"`
	IndexEntry  string `json:"index_entry"`
}

// BuildPackage produces the release tarball, copies the manifest, hashes the
// artifact and renders the registry index entry. artifactKey identifies the
// artifact slot (e.g. "python" or "go/darwin_arm64"); the caller computes it
// after preparing any runtime build.
func BuildPackage(abs, outDir string, m *plugin.Manifest, artifactKey string) (*PackageResult, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating out dir: %w", err)
	}
	tarball := filepath.Join(outDir, fmt.Sprintf("%s-%s.tar.gz", m.Name, m.Version))
	manifestAsset := filepath.Join(outDir, fmt.Sprintf("%s-%s.plugin.yaml", m.Name, m.Version))

	if err := createTarGz(abs, tarball); err != nil {
		return nil, fmt.Errorf("packaging: %w", err)
	}
	if err := copyOut(filepath.Join(abs, "plugin.yaml"), manifestAsset); err != nil {
		return nil, fmt.Errorf("copying manifest: %w", err)
	}
	digest, err := sha256File(tarball)
	if err != nil {
		return nil, fmt.Errorf("hashing artifact: %w", err)
	}

	iconLine := ""
	if m.Icon != "" {
		iconLine = fmt.Sprintf("    icon: %q\n", m.Icon)
	}
	indexEntry := fmt.Sprintf(`plugins:
  %s:
    latest: %s
    description: "%s"
%s    versions:
      - version: %s
        manifest_url: "https://<host>/%s-%s.plugin.yaml"
        artifacts:
          %s:
            url: "https://<host>/%s-%s.tar.gz"
            sha256: "%s"
`, m.Name, m.Version, m.Description, iconLine, m.Version, m.Name, m.Version, artifactKey, m.Name, m.Version, digest)

	return &PackageResult{
		Tarball:     tarball,
		Manifest:    manifestAsset,
		SHA256:      digest,
		ArtifactKey: artifactKey,
		IndexEntry:  indexEntry,
	}, nil
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

	root, err := os.OpenRoot(srcDir)
	if err != nil {
		return err
	}
	defer root.Close()

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
		f, err := root.Open(rel)
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
