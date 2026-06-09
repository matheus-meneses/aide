package plugin

import (
	"aide/cli/internal/keychain"
)

func ScopedSecrets(sourceName string, m *Manifest) (map[string]string, error) {
	if len(m.Credentials) == 0 {
		return nil, nil
	}
	cred, err := keychain.GetAll(sourceName)
	if err != nil {
		return nil, nil //nolint:nilerr // missing credentials is not fatal; plugin runs with empty secrets
	}
	scoped := make(map[string]string, len(m.Credentials))
	for _, c := range m.Credentials {
		if val, ok := cred.Fields[c.Key]; ok {
			scoped[c.Key] = val
		}
	}
	return scoped, nil
}
