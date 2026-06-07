package render

import (
	"strings"

	"aide/cli/internal/store"
)

func init() {
	registerSource("sailpoint", &sailpointPlugin{})
}

type sailpointPlugin struct{}

func (s *sailpointPlugin) Classify(item store.Item) string {
	switch {
	case strings.HasPrefix(item.Title, "Grant:"):
		return "Access approvals pending"
	case strings.HasPrefix(item.Title, "Certification:"):
		return "Certifications"
	default:
		if item.Category != "" {
			return item.Category
		}
		return "Items"
	}
}

func (s *sailpointPlugin) RenderSection(heading string, items []store.Item) {
	(&defaultPlugin{}).RenderSection(heading, items)
}
