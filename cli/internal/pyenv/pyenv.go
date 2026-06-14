package pyenv

import (
	"aide/cli/internal/updater"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	Version = "3.12.7"
	release = "20241016"
	baseURL = "https://github.com/indygreg/python-build-standalone/releases/download/" + release
)

type ProgressFunc func(msg string)

func report(progress ProgressFunc, format string, args ...any) {
	if progress != nil {
		progress(fmt.Sprintf(format, args...))
	}
}

func BinPath(base string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(base, "python", "python.exe")
	}
	return filepath.Join(base, "python", "bin", "python3")
}

func Ensure(base string, progress ProgressFunc) (string, error) {
	pythonBin := BinPath(base)

	if info, err := os.Stat(pythonBin); err == nil && !info.IsDir() {
		report(progress, "Python runtime already installed")
		return pythonBin, nil
	}

	tarballName, err := tarballName()
	if err != nil {
		return "", err
	}

	url := baseURL + "/" + tarballName
	report(progress, "Downloading Python %s...", Version)

	tmpFile, err := os.CreateTemp("", "aide-python-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if err := updater.DownloadFile(url, tmpFile, false); err != nil {
		return "", fmt.Errorf("downloading python: %w", err)
	}
	tmpFile.Close()

	report(progress, "Extracting Python runtime...")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", fmt.Errorf("creating base dir: %w", err)
	}

	if err := run("tar", "-xzf", tmpFile.Name(), "-C", base); err != nil {
		return "", fmt.Errorf("extracting python tarball: %w", err)
	}

	if _, err := os.Stat(pythonBin); err != nil {
		return "", fmt.Errorf("python binary not found after extraction at %s", pythonBin)
	}

	report(progress, "Python %s installed", Version)
	return pythonBin, nil
}

func tarballName() (string, error) {
	var arch, platform string

	switch runtime.GOARCH {
	case "arm64":
		arch = "aarch64"
	case "amd64":
		arch = "x86_64"
	default:
		return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	switch runtime.GOOS {
	case "darwin":
		platform = "apple-darwin"
	case "linux":
		platform = "unknown-linux-gnu"
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	return fmt.Sprintf("cpython-%s+%s-%s-%s-install_only.tar.gz", Version, release, arch, platform), nil
}

func run(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Env = os.Environ()
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return nil
}
