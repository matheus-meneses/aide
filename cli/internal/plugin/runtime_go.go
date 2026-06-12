package plugin

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

func goCmd(ctx context.Context, m *Manifest, reqJSON []byte) (*exec.Cmd, error) {
	binary := m.Entrypoint.Go.Binary
	if binary == "" {
		return nil, fmt.Errorf("plugin %s: no go entrypoint binary", m.Name)
	}
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	binPath := filepath.Join(m.Dir, "bin", binary)
	cmd := exec.CommandContext(ctx, binPath)
	cmd.Dir = m.Dir
	cmd.Stdin = newBytesReader(reqJSON)
	return cmd, nil
}
