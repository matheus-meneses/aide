package render

import (
	"sort"
	"strings"

	"aide/cli/internal/store"
)

type defaultPlugin struct{}

func (d *defaultPlugin) Classify(item store.Item) string {
	if item.Category != "" {
		return item.Category
	}
	return "Items"
}

func (d *defaultPlugin) RenderSection(heading string, items []store.Item) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].FirstSeenAt > items[j].FirstSeenAt
	})

	for _, item := range items {
		title := stripPrefix(item.Title)
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		member := item.Member
		if len(member) > 25 {
			member = member[:22] + "..."
		}
		rightCol := relativeAge(item.FirstSeenAt)
		padding := 50 - len([]rune(title))
		if padding < 0 {
			padding = 0
		}
		displayTitle := title
		if item.Link != "" {
			displayTitle = hyperlink(item.Link, title)
		}
		fprintf(" │    %s%s  %-25s  %6s\n", displayTitle, strings.Repeat(" ", padding), member, rightCol)
	}
}
