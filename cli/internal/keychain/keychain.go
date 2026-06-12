package keychain

import (
	"aide/cli/internal/xdg"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

type Credential struct {
	Fields map[string]string
}

func ServicePrefix() string {
	base := filepath.Base(xdg.AideHome())
	name := strings.TrimLeft(base, ".")
	if name == "" {
		name = "aide"
	}
	return name + "/"
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
	service := ServicePrefix() + source
	raw, err := kcGet(service)
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
	service := ServicePrefix() + source
	return kcDelete(service)
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
	return kcList()
}

func store(source string, cred *Credential) error {
	data, err := json.Marshal(cred.Fields)
	if err != nil {
		return fmt.Errorf("encoding credentials: %w", err)
	}
	service := ServicePrefix() + source
	return kcStore(service, string(data))
}

const accountDefault = "aide"
