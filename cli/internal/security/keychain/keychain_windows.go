//go:build windows

package keychain

import (
	"aide/cli/internal/platform/xdg"
	"encoding/json"
	"fmt"
	"os"
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
	if err := json.Unmarshal(data, &m); err != nil {
		return make(map[string]string), nil
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
	m, err := readCredFile()
	if err != nil {
		return fmt.Errorf("reading credentials file: %w", err)
	}
	m[service] = data
	return writeCredFile(m)
}

func kcGet(service string) (string, error) {
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
