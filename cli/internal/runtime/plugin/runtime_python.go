package plugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func pythonCmdOpts(ctx context.Context, m *Manifest, reqJSON []byte, interactive bool) (*exec.Cmd, error) {
	pythonBin := filepath.Join(m.Dir, ".venv", "bin", "python")
	if runtime.GOOS == "windows" {
		pythonBin = filepath.Join(m.Dir, ".venv", "Scripts", "python.exe")
	}
	script := m.Entrypoint.Python.Script
	if script == "" {
		return nil, fmt.Errorf("plugin %s: no python entrypoint script", m.Name)
	}
	cmd := exec.CommandContext(ctx, pythonBin, script)
	cmd.Dir = m.Dir
	cmd.Stdin = newBytesReader(reqJSON)
	env := os.Environ()
	if m.Capabilities.Browser {
		if p := playwrightBrowsersPath(); p != "" {
			env = append(env, "PLAYWRIGHT_BROWSERS_PATH="+p)
		}
	}
	if interactive {
		env = append(env, "AIDE_INTERACTIVE=1")
	} else {
		env = append(env, "AIDE_INTERACTIVE=0")
	}
	cmd.Env = env
	if sdkPath := os.Getenv("AIDE_SDK_PATH"); sdkPath != "" {
		existing := ""
		for _, e := range env {
			if strings.HasPrefix(e, "PYTHONPATH=") {
				existing = strings.TrimPrefix(e, "PYTHONPATH=")
				break
			}
		}
		if existing != "" {
			cmd.Env = append(cmd.Env, "PYTHONPATH="+sdkPath+string(os.PathListSeparator)+existing)
		} else {
			cmd.Env = append(cmd.Env, "PYTHONPATH="+sdkPath)
		}
	}
	return cmd, nil
}

// playwrightBrowsersPath returns the OS-specific default ms-playwright browser
// cache directory, matching where `playwright install` places browsers during
// venv build. Returns "" when the location can't be resolved.
func playwrightBrowsersPath() string {
	switch runtime.GOOS {
	case "windows":
		if base := os.Getenv("LOCALAPPDATA"); base != "" {
			return filepath.Join(base, "ms-playwright")
		}
		return ""
	case "darwin", "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "Caches", "ms-playwright")
		}
		return filepath.Join(home, ".cache", "ms-playwright")
	default:
		return ""
	}
}
