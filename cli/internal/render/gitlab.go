package render

import (
	"strings"

	"aide/cli/internal/store"
)

func init() {
	registerSource("gitlab", &gitlabPlugin{})
}

type gitlabPlugin struct{}

func (g *gitlabPlugin) Classify(item store.Item) string {
	switch {
	case strings.HasPrefix(item.Title, "Review MR:"):
		return "MRs waiting for your review"
	case strings.HasPrefix(item.Title, "Assigned MR:"):
		return "MRs assigned to you"
	case strings.HasPrefix(item.Title, "Work Item:"):
		return "Work items assigned to you"
	case strings.HasPrefix(item.Title, "Authored Item:"):
		return "Work items authored by you"
	default:
		if item.Category != "" {
			return item.Category
		}
		return "Items"
	}
}

func (g *gitlabPlugin) RenderSection(heading string, items []store.Item) {
	(&defaultPlugin{}).RenderSection(heading, items)
}
