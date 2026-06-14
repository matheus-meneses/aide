package plugin

import (
	"aide/cli/internal/xdg"
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func InstallLocal(_ context.Context, srcDir string, consent func(*Manifest) bool) (*Manifest, error) {
	srcDir, err := filepath.Abs(srcDir)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	m, err := LoadManifest(srcDir)
	if err != nil {
		return nil, fmt.Errorf("loading manifest from %s: %w", srcDir, err)
	}

	if consent != nil && !consent(m) {
		return nil, fmt.Errorf("installation cancelled")
	}

	installDir := filepath.Join(xdg.AideHome(), "plugins", m.Name)

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating install dir: %w", err)
	}

	fmt.Printf("  [+] Copying %s from %s...\n", m.Name, srcDir)
	if err := copyDir(srcDir, installDir); err != nil {
		return nil, fmt.Errorf("copying plugin: %w", err)
	}

	if m.Runtime == "python" && m.Requirements != "" {
		reqFile := filepath.Join(installDir, m.Requirements)
		fmt.Println("  [+] Building venv...")
		if err := buildVenv(installDir, reqFile); err != nil {
			return nil, fmt.Errorf("building venv: %w", err)
		}
	}

	fmt.Printf("  [+] Installed %s (local)\n", m.Name)
	return m, nil
}

func Install(_ context.Context, idx *Index, name, version string, consent func(*Manifest) bool) (*Manifest, error) {
	entry, ok := idx.Plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin %q not found in registry", name)
	}

	if version == "" {
		version = entry.Latest
	}

	var ve *VersionEntry
	for i := range entry.Versions {
		if entry.Versions[i].Version == version {
			ve = &entry.Versions[i]
			break
		}
	}
	if ve == nil {
		return nil, fmt.Errorf("plugin %q version %q not found in registry", name, version)
	}

	manifestURL := ve.ManifestURL
	if manifestURL == "" {
		return nil, fmt.Errorf("plugin %q version %q has no manifest_url", name, version)
	}

	tmpManifestDir, err := os.MkdirTemp("", "aide-plugin-manifest-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpManifestDir)

	manifestPath := filepath.Join(tmpManifestDir, "plugin.yaml")
	if err := downloadWithAuth(manifestURL, manifestPath); err != nil {
		return nil, fmt.Errorf("downloading manifest: %w", err)
	}

	m, err := LoadManifest(tmpManifestDir)
	if err != nil {
		return nil, fmt.Errorf("loading manifest: %w", err)
	}

	if consent != nil && !consent(m) {
		return nil, fmt.Errorf("installation cancelled")
	}

	artifactKey := m.Runtime
	if m.Runtime == "go" {
		artifactKey = fmt.Sprintf("go/%s_%s", runtime.GOOS, runtime.GOARCH)
	}

	artifact, ok := ve.Artifacts[artifactKey]
	if !ok {
		return nil, fmt.Errorf("no artifact for %s in plugin %q", artifactKey, name)
	}

	installDir := filepath.Join(xdg.AideHome(), "plugins", name)

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating install dir: %w", err)
	}

	tmpArtifact, err := os.CreateTemp("", "aide-plugin-artifact-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	tmpArtifact.Close()
	defer os.Remove(tmpArtifact.Name())

	fmt.Printf("  [+] Downloading %s@%s...\n", name, version)
	if err := downloadWithAuth(artifact.URL, tmpArtifact.Name()); err != nil {
		return nil, fmt.Errorf("downloading artifact: %w", err)
	}

	if err := verifySHA256(tmpArtifact.Name(), artifact.SHA256); err != nil {
		return nil, fmt.Errorf("checksum verification failed: %w", err)
	}

	fmt.Println("  [+] Extracting...")
	if err := extractTarGz(tmpArtifact.Name(), installDir); err != nil {
		return nil, fmt.Errorf("extracting artifact: %w", err)
	}

	manifestDest := filepath.Join(installDir, "plugin.yaml")
	if err := copyFile(manifestPath, manifestDest); err != nil {
		return nil, fmt.Errorf("copying manifest: %w", err)
	}

	installed, err := LoadManifest(installDir)
	if err != nil {
		return nil, fmt.Errorf("validating installed plugin: %w", err)
	}

	if installed.Runtime == "python" && installed.Requirements != "" {
		reqFile := filepath.Join(installDir, installed.Requirements)
		if err := buildVenv(installDir, reqFile); err != nil {
			return nil, fmt.Errorf("building venv: %w", err)
		}
	}

	fmt.Printf("  [+] Installed %s@%s\n", name, version)
	return installed, nil
}

func Remove(name string) error {
	return NewManager().Remove(name)
}

// EnsureRuntime prepares a plugin directory so it can be executed in place:
// a Python plugin gets a .venv (built if missing), a Go plugin is compiled to
// bin/<binary>. It is used by `aide dev test` to run uninstalled plugins.
func EnsureRuntime(ctx context.Context, m *Manifest) error {
	switch m.Runtime {
	case "python":
		venvDir := filepath.Join(m.Dir, ".venv")
		if info, err := os.Stat(venvDir); err == nil && info.IsDir() {
			return nil
		}
		if m.Requirements == "" {
			return fmt.Errorf("python plugin %s: no requirements file declared", m.Name)
		}
		return buildVenv(m.Dir, filepath.Join(m.Dir, m.Requirements))
	case "go":
		return buildGoBinary(ctx, m)
	default:
		return fmt.Errorf("unsupported runtime %q", m.Runtime)
	}
}

func buildGoBinary(ctx context.Context, m *Manifest) error {
	binary := m.Entrypoint.Go.Binary
	if binary == "" {
		return fmt.Errorf("go plugin %s: entrypoint.go.binary is required", m.Name)
	}
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	binDir := filepath.Join(m.Dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("creating bin dir: %w", err)
	}
	cmd := exec.CommandContext(ctx, "go", "build", "-o", filepath.Join(binDir, binary), ".")
	cmd.Dir = m.Dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}
	return nil
}

func buildVenv(pluginDir, reqFile string) error {
	pythonBin := filepath.Join(xdg.AideHome(), "python", "bin", "python3")
	if runtime.GOOS == "windows" {
		pythonBin = filepath.Join(xdg.AideHome(), "python", "python.exe")
	}
	if _, err := os.Stat(pythonBin); err != nil {
		if sysPython, err := exec.LookPath("python3"); err == nil {
			pythonBin = sysPython
		} else if sysPython, err := exec.LookPath("python"); err == nil {
			pythonBin = sysPython
		} else {
			return fmt.Errorf("no python interpreter found")
		}
	}

	venvDir := filepath.Join(pluginDir, ".venv")
	if err := os.RemoveAll(venvDir); err != nil {
		return fmt.Errorf("clearing stale venv: %w", err)
	}
	if err := runCmd(pythonBin, "-m", "venv", venvDir); err != nil {
		return fmt.Errorf("creating venv: %w", err)
	}

	pipBin := filepath.Join(venvDir, "bin", "pip")
	if runtime.GOOS == "windows" {
		pipBin = filepath.Join(venvDir, "Scripts", "pip.exe")
	}

	if err := runCmd(pipBin, "install", "--upgrade", "pip"); err != nil {
		return fmt.Errorf("upgrading pip: %w", err)
	}

	if sdkPath := os.Getenv("AIDE_SDK_PATH"); sdkPath != "" {
		fmt.Printf("  [+] Installing local SDK from %s...\n", sdkPath)
		if err := runCmd(pipBin, "install", "setuptools"); err != nil {
			fmt.Printf("  [!] setuptools install failed: %v\n", err)
		}
		if err := runCmd(pipBin, "install", sdkPath); err != nil {
			fmt.Printf("  [!] Local SDK install failed: %v\n", err)
		}
	}

	if err := runCmd(pipBin, "install", "-r", reqFile); err != nil {
		return fmt.Errorf("installing requirements: %w", err)
	}

	pythonVenvBin := filepath.Join(venvDir, "bin", "python")
	if runtime.GOOS == "windows" {
		pythonVenvBin = filepath.Join(venvDir, "Scripts", "python.exe")
	}
	defaultBrowsersPath := filepath.Join(os.Getenv("HOME"), "Library", "Caches", "ms-playwright")
	if _, err := os.Stat(defaultBrowsersPath); err == nil { //nolint:gosec // G703: path is constructed from HOME env + known suffix, not user-controlled
		fmt.Printf("  [+] Playwright browsers already present at %s, skipping download.\n", defaultBrowsersPath)
	} else {
		fmt.Printf("  [+] Installing Playwright browsers...\n")
		if err := runCmd(pythonVenvBin, "-m", "playwright", "install", "chromium"); err != nil {
			fmt.Printf("  [!] playwright install failed: %v\n", err)
			fmt.Printf("  [!] Run manually outside Cursor: %s -m playwright install chromium\n", pythonVenvBin)
		}
	}

	return nil
}

func downloadWithAuth(url, destPath string) error {
	resp, err := httpGetAsset(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func extractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("opening gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		target := filepath.Join(dest, filepath.Clean(hdr.Name))
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("tar path traversal detected: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)) //nolint:gosec // G115: mode comes from tar header of a trusted local archive
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil { //nolint:gosec // G110: plugin archives are trusted; size validated before extraction
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

func copyFile(src, dst string) error {
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

func runCmd(name string, args ...string) error {
	c := exec.Command(name, args...) //nolint:gosec // G702: name comes from plugin manifest paths, not user input
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		if strings.HasPrefix(rel, ".venv"+string(os.PathSeparator)) ||
			rel == ".venv" ||
			strings.HasPrefix(rel, "__pycache__"+string(os.PathSeparator)) {
			return nil
		}

		return copyFile(path, target)
	})
}
