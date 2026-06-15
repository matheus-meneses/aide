package render

import (
	"aide/cli/internal/persistence/store"
	"sort"
	"strings"
)

type defaultPlugin struct{}

func (d *defaultPlugin) Classify(item store.Item) string {
	if item.Category != "" {
		return item.Category
	}
	return "Items"
}

func (d *defaultPlugin) RenderSection(_ string, items []store.Item) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].FirstSeenAt > items[j].FirstSeenAt
	})

	for _, item := range items {
		title := truncateRunes(stripPrefix(item.Title), 50)
		member := truncateRunes(item.Member, 25)
		rightCol := relativeAge(item.FirstSeenAt)

		// Pad on the visible (plain) rune width so OSC-8 hyperlink escape
		// sequences never skew column alignment.
		displayTitle := title
		if item.Link != "" {
			displayTitle = hyperlink(item.Link, title)
		}
		fprintf(" │    %s%s  %s%s  %6s\n",
			displayTitle, padRight(title, 50),
			member, padRight(member, 25),
			rightCol)
	}
}

// truncateRunes shortens s to at most max runes, appending an ellipsis when
// truncated. It counts runes (not bytes) so multibyte characters are never cut
// mid-sequence.
func truncateRunes(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	if maxRunes <= 3 {
		return string(r[:maxRunes])
	}
	return string(r[:maxRunes-3]) + "..."
}

// padRight returns the spaces needed to pad s (measured in runes) to width.
func padRight(s string, width int) string {
	n := width - len([]rune(s))
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}
