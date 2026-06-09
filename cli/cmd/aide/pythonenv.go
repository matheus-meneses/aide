package main

import (
	"aide/cli/internal/updater"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	pythonVersion = "3.12.7"
	pythonRelease = "20241016"
	pythonBaseURL = "https://github.com/indygreg/python-build-standalone/releases/download/" + pythonRelease
)

func ensurePython(base string) (string, error) {
	pythonDir := filepath.Join(base, "python")
	pythonBin := filepath.Join(pythonDir, "bin", "python3")
	if runtime.GOOS == "windows" {
		pythonBin = filepath.Join(pythonDir, "python.exe")
	}

	if info, err := os.Stat(pythonBin); err == nil && !info.IsDir() {
		fmt.Printf("  [=] Standalone Python already installed (%s)\n", pythonBin)
		return pythonBin, nil
	}

	tarballName, err := pythonTarballName()
	if err != nil {
		return "", err
	}

	url := pythonBaseURL + "/" + tarballName
	fmt.Printf("  [+] Downloading Python %s...\n", pythonVersion)

	tmpFile, err := os.CreateTemp("", "aide-python-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if err := updater.DownloadFile(url, tmpFile, true); err != nil {
		return "", fmt.Errorf("downloading python: %w", err)
	}
	tmpFile.Close()

	fmt.Println("  [+] Extracting Python...")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", fmt.Errorf("creating base dir: %w", err)
	}

	if err := execCmdSilent("tar", "-xzf", tmpFile.Name(), "-C", base); err != nil {
		return "", fmt.Errorf("extracting python tarball: %w", err)
	}

	if _, err := os.Stat(pythonBin); err != nil {
		return "", fmt.Errorf("python binary not found after extraction at %s", pythonBin)
	}

	fmt.Printf("  [+] Python %s installed\n", pythonVersion)
	return pythonBin, nil
}

func pythonTarballName() (string, error) {
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

	return fmt.Sprintf("cpython-%s+%s-%s-%s-install_only.tar.gz", pythonVersion, pythonRelease, arch, platform), nil
}

func execCmdSilent(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Env = os.Environ()
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return nil
}
