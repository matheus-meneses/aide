package updater

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Method string

const (
	MethodDev             Method = "dev"
	MethodScript          Method = "script"
	MethodHomebrewFormula Method = "homebrew-formula"
	MethodHomebrewCask    Method = "homebrew-cask"
	MethodManualApp       Method = "manual-app"
	MethodUnknown         Method = "unknown"
)

// CanSelfUpdate reports whether Apply can update this installation in place
// without the user running manual commands.
func (m Method) CanSelfUpdate() bool {
	switch m {
	case MethodScript, MethodHomebrewFormula, MethodHomebrewCask, MethodManualApp:
		return true
	default:
		return false
	}
}

// DetectMethod classifies how the running binary was installed so the updater
// can pick a safe upgrade path. version is the build-stamped version ("dev" for
// local builds, which are never auto-updated).
func DetectMethod(version string) Method {
	if version == "" || version == "dev" {
		return MethodDev
	}

	exe, err := os.Executable()
	if err != nil {
		return MethodUnknown
	}
	if resolved, rErr := filepath.EvalSymlinks(exe); rErr == nil {
		exe = resolved
	}

	if isHomebrewPath(exe) {
		return MethodHomebrewFormula
	}

	if appBundlePath(exe) != "" {
		if caskInstalled() {
			return MethodHomebrewCask
		}
		return MethodManualApp
	}

	return MethodScript
}

// isHomebrewPath reports whether p lives inside a Homebrew Cellar/prefix, which
// means brew owns the binary and we must upgrade through brew rather than
// overwriting the file.
func isHomebrewPath(p string) bool {
	if strings.Contains(p, "/Cellar/aide/") {
		return true
	}
	for _, env := range []string{os.Getenv("HOMEBREW_CELLAR"), os.Getenv("HOMEBREW_PREFIX")} {
		if env != "" && strings.HasPrefix(p, strings.TrimRight(env, "/")+"/") {
			return true
		}
	}
	if prefix := brewPrefix(); prefix != "" && strings.HasPrefix(p, prefix+"/") {
		return true
	}
	return false
}

// appBundlePath returns the enclosing "*.app" bundle path for exe, or "" when
// exe is not inside a macOS application bundle.
func appBundlePath(exe string) string {
	const marker = ".app/Contents/MacOS/"
	idx := strings.Index(exe, marker)
	if idx < 0 {
		return ""
	}
	return exe[:idx+len(".app")]
}

// brewBin locates the brew executable, falling back to the standard install
// locations because GUI apps launched from Finder inherit a minimal PATH.
func brewBin() string {
	if p, err := exec.LookPath("brew"); err == nil {
		return p
	}
	for _, p := range []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"} {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

func brewPrefix() string {
	brew := brewBin()
	if brew == "" {
		return ""
	}
	out, err := exec.Command(brew, "--prefix").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func caskInstalled() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	brew := brewBin()
	if brew == "" {
		return false
	}
	out, err := exec.Command(brew, "list", "--cask", "--versions", "aide").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}
