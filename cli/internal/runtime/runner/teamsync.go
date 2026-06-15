package runner

import (
	"aide/cli/internal/persistence/store"
	"aide/cli/internal/platform/config"
	"encoding/json"
)

func SyncTeamFromConfig(cfg *config.Config, s *store.Store) error {
	if len(cfg.Team) == 0 {
		return nil
	}

	members := make([]store.Member, 0, len(cfg.Team))
	for _, t := range cfg.Team {
		aliasesJSON := "[]"
		if len(t.Aliases) > 0 {
			b, err := json.Marshal(t.Aliases)
			if err == nil {
				aliasesJSON = string(b)
			}
		}
		members = append(members, store.Member{
			Name:         t.Name,
			Email:        t.Email,
			Aliases:      aliasesJSON,
			Role:         t.Role,
			Department:   t.Department,
			Branch:       t.Branch,
			Registration: t.Registration,
			ManagerRef:   t.Manager,
			Source:       "config",
		})
	}

	return s.Team.Upsert(members)
}
