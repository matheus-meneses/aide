//go:build windows

package sandbox

import (
	"fmt"
	"os/exec"
)

func Wrap(cmd *exec.Cmd, p Policy) error {
	fmt.Fprintf(cmd.Stderr.(interface{ Write([]byte) (int, error) }), "warning: sandboxing not supported on Windows, running %s unsandboxed\n", p.Name)
	return nil
}
