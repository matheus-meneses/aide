//go:build linux

package plugin

import (
	"fmt"
	"os/exec"
)

func Wrap(cmd *exec.Cmd, m *Manifest) error {
	if _, err := exec.LookPath("bwrap"); err != nil {
		fmt.Fprintf(cmd.Stderr.(interface{ Write([]byte) (int, error) }), "warning: bwrap not found, running %s unsandboxed\n", m.Name)
		return nil
	}

	bwrapArgs := []string{
		"--ro-bind", "/", "/",
		"--bind", m.Dir, m.Dir,
		"--dev", "/dev",
		"--proc", "/proc",
	}
	if len(m.Capabilities.Network) == 0 {
		bwrapArgs = append(bwrapArgs, "--unshare-net")
	}
	bwrapArgs = append(bwrapArgs, "--")
	bwrapArgs = append(bwrapArgs, cmd.Args...)

	cmd.Path, _ = exec.LookPath("bwrap")
	cmd.Args = append([]string{"bwrap"}, bwrapArgs...)
	return nil
}
