package keychain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

const (
	servicePrefix  = "aide/"
	accountDefault = "aide"
)

type Credential struct {
	Fields map[string]string
}

func SetField(source, key, value string) error {
	cred, _ := GetAll(source)
	if cred == nil {
		cred = &Credential{Fields: make(map[string]string)}
	}
	cred.Fields[key] = value
	return store(source, cred)
}

func GetAll(source string) (*Credential, error) {
	service := servicePrefix + source
	raw, err := getPassword(service)
	if err != nil {
		return nil, err
	}

	fields := make(map[string]string)
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return nil, fmt.Errorf("corrupted credential data for %s: %w", source, err)
	}

	return &Credential{Fields: fields}, nil
}

func DeleteSource(source string) error {
	service := servicePrefix + source
	cmd := exec.Command("/usr/bin/security", "delete-generic-password", "-s", service)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("keychain delete failed: %s", strings.TrimSpace(stderr.String()))
	}
	return nil
}

func DeleteField(source, key string) error {
	cred, err := GetAll(source)
	if err != nil {
		return err
	}

	delete(cred.Fields, key)

	if len(cred.Fields) == 0 {
		return DeleteSource(source)
	}

	return store(source, cred)
}

func List() ([]string, error) {
	cmd := exec.Command("/usr/bin/security", "dump-keychain")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("keychain dump failed: %w", err)
	}

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
		if !strings.HasPrefix(val, servicePrefix) {
			continue
		}
		name := strings.TrimPrefix(val, servicePrefix)
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

func store(source string, cred *Credential) error {
	data, err := json.Marshal(cred.Fields)
	if err != nil {
		return fmt.Errorf("encoding credentials: %w", err)
	}

	service := servicePrefix + source
	cmd := exec.Command("/usr/bin/security", "add-generic-password",
		"-s", service,
		"-a", accountDefault,
		"-w", string(data),
		"-U",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("keychain store failed: %s", strings.TrimSpace(stderr.String()))
	}
	return nil
}

func getPassword(service string) (string, error) {
	cmd := exec.Command("/usr/bin/security", "find-generic-password", "-s", service, "-w")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("credential not found for %s", service)
	}
	return strings.TrimSpace(stdout.String()), nil
}
