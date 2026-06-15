//go:build linux

package keychain

import (
	"aide/cli/internal/platform/xdg"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func credFilePath() string {
	return filepath.Join(xdg.AideHome(), "credentials.json")
}

func readCredFile() (map[string]string, error) {
	data, err := os.ReadFile(credFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	var m map[string]string
	if json.Unmarshal(data, &m) != nil {
		m = make(map[string]string)
	}
	return m, nil
}

func writeCredFile(m map[string]string) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	path := credFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func kcStore(service, data string) error {
	if _, err := exec.LookPath("secret-tool"); err == nil {
		cmd := exec.Command("secret-tool", "store", "--label=aide", "service", service, "account", accountDefault)
		cmd.Stdin = bytes.NewBufferString(data)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	m, err := readCredFile()
	if err != nil {
		return fmt.Errorf("reading credentials file: %w", err)
	}
	m[service] = data
	return writeCredFile(m)
}

func kcGet(service string) (string, error) {
	if _, err := exec.LookPath("secret-tool"); err == nil {
		out, err := exec.Command("secret-tool", "lookup", "service", service, "account", accountDefault).Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
	}
	m, err := readCredFile()
	if err != nil {
		return "", fmt.Errorf("reading credentials file: %w", err)
	}
	val, ok := m[service]
	if !ok {
		return "", fmt.Errorf("credential not found for %s", service)
	}
	return val, nil
}

func kcDelete(service string) error {
	if _, err := exec.LookPath("secret-tool"); err == nil {
		if err := exec.Command("secret-tool", "clear", "service", service, "account", accountDefault).Run(); err == nil {
			return nil
		}
	}
	m, err := readCredFile()
	if err != nil {
		return fmt.Errorf("reading credentials file: %w", err)
	}
	delete(m, service)
	return writeCredFile(m)
}

func kcList() ([]string, error) {
	m, err := readCredFile()
	if err != nil {
		return nil, err
	}
	prefix := ServicePrefix()
	var sources []string
	for key := range m {
		if strings.HasPrefix(key, prefix) {
			sources = append(sources, strings.TrimPrefix(key, prefix))
		}
	}
	return sources, nil
}
