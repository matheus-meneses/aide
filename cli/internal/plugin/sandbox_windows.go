//go:build windows

package plugin

import (
	"fmt"
	"os/exec"
)

func Wrap(cmd *exec.Cmd, m *Manifest) error {
	fmt.Fprintf(cmd.Stderr.(interface{ Write([]byte) (int, error) }), "warning: sandboxing not supported on Windows, running %s unsandboxed\n", m.Name)
	return nil
}
