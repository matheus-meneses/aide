//go:build darwin

package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Wrap(cmd *exec.Cmd, p Policy) error {
	sandboxExec, err := exec.LookPath("sandbox-exec")
	if err != nil {
		fmt.Fprintf(cmd.Stderr.(interface{ Write([]byte) (int, error) }), "warning: sandbox-exec not found, running %s unsandboxed\n", p.Name)
		return nil //nolint:nilerr // missing sandbox-exec is a soft warning; plugin runs unsandboxed
	}

	profile := buildDarwinProfile(p)
	originalArgs := cmd.Args
	cmd.Path = sandboxExec
	cmd.Args = append([]string{"sandbox-exec", "-p", profile}, originalArgs...)
	return nil
}

func buildDarwinProfile(p Policy) string {
	var b strings.Builder
	b.WriteString("(version 1)\n(deny default)\n")
	b.WriteString("(allow process*)\n")
	b.WriteString("(allow file-read*)\n")
	fmt.Fprintf(&b, "(allow file-write* (subpath %q))\n", p.Dir)
	for _, w := range p.Writes {
		fmt.Fprintf(&b, "(allow file-write* (subpath %q))\n", w)
	}
	if p.Browser {
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
	if len(p.Network) > 0 {
		b.WriteString("(allow network*)\n")
	}
	return b.String()
}
