//go:build darwin

package keychain

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func kcStore(service, data string) error {
	cmd := exec.Command(
		"/usr/bin/security", "add-generic-password",
		"-s", service,
		"-a", accountDefault,
		"-w", data,
		"-U",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("keychain store failed: %s", strings.TrimSpace(stderr.String()))
	}
	return nil
}

func kcGet(service string) (string, error) {
	cmd := exec.Command("/usr/bin/security", "find-generic-password", "-s", service, "-w")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("credential not found for %s", service)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func kcDelete(service string) error {
	cmd := exec.Command("/usr/bin/security", "delete-generic-password", "-s", service)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("keychain delete failed: %s", strings.TrimSpace(stderr.String()))
	}
	return nil
}

func kcList() ([]string, error) {
	cmd := exec.Command("/usr/bin/security", "dump-keychain")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("keychain dump failed: %w", err)
	}

	prefix := ServicePrefix()
	var sources []string
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "\"svce\"") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		if !strings.HasPrefix(val, prefix) {
			continue
		}
		name := strings.TrimPrefix(val, prefix)
		found := false
		for _, s := range sources {
			if s == name {
				found = true
				break
			}
		}
		if !found {
			sources = append(sources, name)
		}
	}
	return sources, nil
}
