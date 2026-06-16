//go:build linux

package sandbox

import (
	"fmt"
	"os/exec"
)

func Wrap(cmd *exec.Cmd, p Policy) error {
	bwrapPath, _ := exec.LookPath("bwrap")
	if bwrapPath == "" {
		fmt.Fprintf(cmd.Stderr.(interface{ Write([]byte) (int, error) }), "warning: bwrap not found, running %s unsandboxed\n", p.Name)
		return nil
	}

	cmd.Path = bwrapPath
	cmd.Args = append([]string{"bwrap"}, buildBwrapArgs(p, cmd.Args)...)
	return nil
}

// buildBwrapArgs assembles the bwrap argument list for a policy: the whole
// filesystem is read-only, the plugin dir and any declared write paths are
// writable, and the network namespace is unshared unless network is requested.
// Kept pure so the policy-to-args mapping is unit testable without bwrap.
func buildBwrapArgs(p Policy, childArgs []string) []string {
	args := []string{
		"--ro-bind", "/", "/",
		"--bind", p.Dir, p.Dir,
		"--dev", "/dev",
		"--proc", "/proc",
	}
	for _, w := range p.Writes {
		args = append(args, "--bind", w, w)
	}
	if len(p.Network) == 0 {
		args = append(args, "--unshare-net")
	}
	args = append(args, "--")
	args = append(args, childArgs...)
	return args
}
