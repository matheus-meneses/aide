package provision

import (
	"aide/cli/internal/platform/config"
)

// GetTeam returns the team roster declared in config.yaml.
func GetTeam(cfgPath string) ([]config.TeamMember, error) {
	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return nil, err
	}
	return cfg.Team, nil
}

// SetTeam replaces the team roster in config.yaml. Callers should follow up with
// a config reload so the runtime store is re-synced from the new roster.
func SetTeam(cfgPath string, members []config.TeamMember) error {
	cfg, err := config.LoadRaw(cfgPath)
	if err != nil {
		return err
	}
	cfg.Team = members
	return cfg.Save(cfgPath)
}
