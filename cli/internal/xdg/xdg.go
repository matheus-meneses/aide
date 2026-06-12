package xdg

import (
	"os"
	"path/filepath"
)

func AideHome() string {
	if h := os.Getenv("AIDE_HOME"); h != "" {
		return h
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aide")
}
