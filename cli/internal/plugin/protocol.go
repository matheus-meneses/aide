package plugin

import (
	"encoding/json"
	"fmt"
)

const ProtocolVersion = "1"

type Request struct {
	ProtocolVersion string            `json:"protocol_version"`
	Action          string            `json:"action"`
	Config          map[string]any    `json:"config,omitempty"`
	Secrets         map[string]string `json:"secrets,omitempty"`
	Context         map[string]any    `json:"context,omitempty"`
	Heading         string            `json:"heading,omitempty"`
	Items           []Entry           `json:"items,omitempty"`
	Name            string            `json:"name,omitempty"`
	Params          map[string]any    `json:"params,omitempty"`
}

type Entry struct {
	Member    string         `json:"member"`
	Category  string         `json:"category"`
	Title     string         `json:"title"`
	Detail    string         `json:"detail"`
	EntryDate string         `json:"entry_date"`
	Priority  string         `json:"priority"`
	Link      string         `json:"link,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type TeamMember struct {
	Name                string `json:"name"`
	Email               string `json:"email"`
	Role                string `json:"role"`
	Department          string `json:"department"`
	Branch              string `json:"branch"`
	Registration        string `json:"registration"`
	ManagerRegistration string `json:"manager_registration"`
}

type Metric struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type Response struct {
	ProtocolVersion string       `json:"protocol_version"`
	OK              bool         `json:"ok"`
	Entries         []Entry      `json:"entries,omitempty"`
	TeamMembers     []TeamMember `json:"team_members,omitempty"`
	Metrics         []Metric     `json:"metrics,omitempty"`
	Lines           []string     `json:"lines,omitempty"`
	Text            string       `json:"text,omitempty"`
	Error           string       `json:"error,omitempty"`
}

func Parse(stdout []byte) (*Response, error) {
	var resp Response
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return nil, fmt.Errorf("parsing plugin response: %w", err)
	}
	return &resp, nil
}
