//go:build !darwin

package updater

import (
	"context"
	"fmt"
)

// applyAppUpdate is only meaningful on macOS, where the desktop app ships.
func applyAppUpdate(_ context.Context, _ Method, _ Release, _ Progress) error {
	return fmt.Errorf("app self-update is only supported on macOS")
}
