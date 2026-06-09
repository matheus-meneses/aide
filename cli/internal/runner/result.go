package runner

import (
	"aide/cli/internal/plugin"
)

type ScraperEntry struct {
	Source    string         `json:"source"`
	Member    string         `json:"member"`
	Category  string         `json:"category"`
	Title     string         `json:"title"`
	Detail    string         `json:"detail"`
	EntryDate string         `json:"entry_date"`
	Priority  string         `json:"priority"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type TeamMemberRaw struct {
	Name                string `json:"name"`
	Email               string `json:"email"`
	Role                string `json:"role"`
	Department          string `json:"department"`
	Branch              string `json:"branch"`
	Registration        string `json:"registration"`
	ManagerRegistration string `json:"manager_registration"`
}

type ScraperPayload struct {
	Entries     []ScraperEntry  `json:"entries"`
	TeamMembers []TeamMemberRaw `json:"team_members,omitempty"`
}

type SourceResult struct {
	Source      string
	Entries     []ScraperEntry
	TeamMembers []TeamMemberRaw
	PluginResp  *plugin.Response
	Error       error
	DurationMs  int64
	Stderr      string
	NewItems    int
}

type RunResult struct {
	RunID         string
	SourcesTotal  int
	SourcesOK     int
	SourcesFailed int
	Results       []SourceResult
}
