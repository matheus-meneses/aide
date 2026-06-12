//go:build !windows

package procctl

import (
	"os/exec"
	"syscall"
)

func Configure(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}
}

func Kill(cmd *exec.Cmd) error {
	if cmd.Process != nil {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	return nil
}
