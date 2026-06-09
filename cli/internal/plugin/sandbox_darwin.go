//go:build darwin

package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Wrap(cmd *exec.Cmd, m *Manifest) error {
	if m.Capabilities.Browser {
		return nil
	}
	if _, err := exec.LookPath("sandbox-exec"); err != nil {
		fmt.Fprintf(cmd.Stderr.(interface{ Write([]byte) (int, error) }), "warning: sandbox-exec not found, running %s unsandboxed\n", m.Name)
		return nil //nolint:nilerr // missing sandbox-exec is a soft warning; plugin runs unsandboxed
	}

	profile := buildDarwinProfile(m)
	originalArgs := cmd.Args
	cmd.Path, _ = exec.LookPath("sandbox-exec")
	cmd.Args = append([]string{"sandbox-exec", "-p", profile}, originalArgs...)
	return nil
}

func buildDarwinProfile(m *Manifest) string {
	var b strings.Builder
	b.WriteString("(version 1)\n(deny default)\n")
	b.WriteString("(allow process*)\n")
	b.WriteString("(allow file-read*)\n")
	fmt.Fprintf(&b, "(allow file-write* (subpath %q))\n", m.Dir)
	if m.Capabilities.Browser {
		b.WriteString("(allow mach*)\n")
		b.WriteString("(allow ipc*)\n")
		b.WriteString("(allow sysctl*)\n")
		tmpDir := os.TempDir()
		if resolved, err := filepath.EvalSymlinks(tmpDir); err == nil {
			tmpDir = resolved
		}
		fmt.Fprintf(&b, "(allow file-write* (subpath %q))\n", tmpDir)
		b.WriteString("(allow file-write* (subpath \"/private/var/folders\"))\n")
		fmt.Fprintf(&b, "(allow file-write* (subpath %q))\n", os.Getenv("HOME")+"/Library/Application Support/Chromium")
		fmt.Fprintf(&b, "(allow file-write* (subpath %q))\n", os.Getenv("HOME")+"/Library/Caches/Chromium")
	}
	if len(m.Capabilities.Network) > 0 {
		b.WriteString("(allow network*)\n")
	}
	return b.String()
}
