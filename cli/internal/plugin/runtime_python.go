package plugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
		systemBrowsersPath := filepath.Join(os.Getenv("HOME"), "Library", "Caches", "ms-playwright")
		env = append(env, "PLAYWRIGHT_BROWSERS_PATH="+systemBrowsersPath)
	}
	if interactive {
		env = append(env, "AIDE_INTERACTIVE=1")
	} else {
		env = append(env, "AIDE_INTERACTIVE=0")
	}
	cmd.Env = env
	return cmd, nil
}
