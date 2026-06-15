//go:build linux

package sandbox

import (
	"fmt"
	"os/exec"
)

func Wrap(cmd *exec.Cmd, p Policy) error {
	if _, err := exec.LookPath("bwrap"); err != nil {
		fmt.Fprintf(cmd.Stderr.(interface{ Write([]byte) (int, error) }), "warning: bwrap not found, running %s unsandboxed\n", p.Name)
		return nil
	}

	bwrapArgs := []string{
		"--ro-bind", "/", "/",
		"--bind", p.Dir, p.Dir,
		"--dev", "/dev",
		"--proc", "/proc",
	}
	if len(p.Network) == 0 {
		bwrapArgs = append(bwrapArgs, "--unshare-net")
	}
	bwrapArgs = append(bwrapArgs, "--")
	bwrapArgs = append(bwrapArgs, cmd.Args...)

	cmd.Path, _ = exec.LookPath("bwrap")
	cmd.Args = append([]string{"bwrap"}, bwrapArgs...)
	return nil
}
