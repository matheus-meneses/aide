package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"aide/cli/internal/updater"
)

const (
	pythonVersion = "3.12.7"
	pythonRelease = "20241016"
	pythonBaseURL = "https://github.com/indygreg/python-build-standalone/releases/download/" + pythonRelease
)

func setupScraperVenv(base, scrapersDir string) error {
	venvDir := filepath.Join(scrapersDir, ".venv")

	pythonPath, err := ensurePython(base)
	if err != nil {
		return fmt.Errorf("setting up python: %w", err)
	}

	if _, err := os.Stat(venvDir); err == nil {
		fmt.Println("  [~] Removing old venv...")
		os.RemoveAll(venvDir)
	}

	fmt.Println("  [+] Creating Python venv...")
	if err := execCmd(pythonPath, "-m", "venv", venvDir); err != nil {
		return fmt.Errorf("creating venv: %w", err)
	}

	pipBin := filepath.Join(venvDir, "bin", "pip")
	if runtime.GOOS == "windows" {
		pipBin = filepath.Join(venvDir, "Scripts", "pip.exe")
	}

	fmt.Println("  [+] Installing Python dependencies...")
	reqFile := filepath.Join(scrapersDir, "requirements.txt")
	if err := execCmd(pipBin, "install", "--upgrade", "pip"); err != nil {
		return fmt.Errorf("upgrading pip: %w", err)
	}
	if err := execCmd(pipBin, "install", "-r", reqFile); err != nil {
		return fmt.Errorf("installing requirements: %w", err)
	}

	pythonBin := filepath.Join(venvDir, "bin", "python")
	if runtime.GOOS == "windows" {
		pythonBin = filepath.Join(venvDir, "Scripts", "python.exe")
	}

	fmt.Println("  [+] Installing Playwright chromium...")
	if err := execCmd(pythonBin, "-m", "playwright", "install", "chromium"); err != nil {
		fmt.Printf("  [!] Playwright install failed (non-fatal): %v\n", err)
	}

	return nil
}

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

func execCmd(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = append(os.Environ(),
		"NODE_TLS_REJECT_UNAUTHORIZED=0",
		"PYTHONHTTPSVERIFY=0",
		"PIP_TRUSTED_HOST=pypi.org files.pythonhosted.org pypi.python.org",
	)
	return c.Run()
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
