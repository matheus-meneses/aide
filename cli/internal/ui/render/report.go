package render

import (
	"aide/cli/internal/persistence/store"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/runner"
	"fmt"
	"sort"
	"strings"
	"time"
)

func PrintRunSummary(result *runner.RunResult) {
	printRunSummaryTable(result)
}

func PrintReport(s *store.Store, cfg *config.Config, member, category string) error {
	items, err := s.Items.QueryOpen("", member, category)
	if err != nil {
		return err
	}
	printItemsReport(items, sourcePluginMap(cfg))
	return nil
}

func PrintDiff(s *store.Store, source string) error {
	since := time.Now().UTC().Add(-24 * time.Hour)

	newItems, err := s.Items.RecentlyDiscovered(source, since)
	if err != nil {
		return err
	}
	resolved, err := s.Items.RecentlyResolved(source, since)
	if err != nil {
		return err
	}
	printDiffReport(newItems, resolved)
	return nil
}

func PrintSources(cfg *config.Config, s *store.Store) error {
	health, err := s.Runs.AllHealth()
	if err != nil {
		return err
	}
	printSourcesTable(cfg, health)
	return nil
}

func PrintHistory(s *store.Store) error {
	runs, err := s.Runs.History(20)
	if err != nil {
		return err
	}
	printHistoryTable(runs)
	return nil
}

func printItemsReport(items []store.Item, pluginMap map[string]string) {
	if len(items) == 0 {
		fprintln("No open items.")
		return
	}

	var actionRequired []store.Item
	var informational []store.Item

	for _, item := range items {
		if item.Priority == "critical" || item.Priority == "warning" {
			actionRequired = append(actionRequired, item)
		} else {
			informational = append(informational, item)
		}
	}

	fprintf("\n Report (%d open items)\n", len(items))
	printItemSummaryBar(items)

	if len(actionRequired) > 0 {
		fprintf("\n ┌─ ACTION REQUIRED (%d) ─────────────────────────────────────────\n", len(actionRequired))
		printGroupedItems(actionRequired, pluginMap)
	}

	if len(informational) > 0 {
		fprintf("\n ┌─ INFORMATIONAL (%d) ────────────────────────────────────────────\n", len(informational))
		printGroupedItems(informational, pluginMap)
	}

	fprintln()
}

func printItemSummaryBar(items []store.Item) {
	sourceCounts := make(map[string]int)
	priorityCounts := make(map[string]int)
	for _, item := range items {
		sourceCounts[item.Source]++
		priorityCounts[item.Priority]++
	}

	var parts []string
	for src, count := range sourceCounts {
		parts = append(parts, fmt.Sprintf("%s: %d", src, count))
	}
	sort.Strings(parts)

	fprintf(" Sources: %s\n", strings.Join(parts, " | "))

	if c := priorityCounts["critical"]; c > 0 {
		fprintf(" Critical: %d", c)
	}
	if w := priorityCounts["warning"]; w > 0 {
		fprintf(" Warning: %d", w)
	}
	if i := priorityCounts["info"]; i > 0 {
		fprintf(" Info: %d", i)
	}
	fprintln()
}

type sectionEntry struct {
	source  string
	heading string
	items   []store.Item
}

func printGroupedItems(items []store.Item, pluginMap map[string]string) {
	sectionMap := make(map[string]*sectionEntry)
	var sectionOrder []string

	for _, item := range items {
		plugin := pluginFor(item.Source, pluginMap)
		heading := plugin.Classify(item)
		mapKey := item.Source + "|" + heading
		if _, exists := sectionMap[mapKey]; !exists {
			sectionMap[mapKey] = &sectionEntry{source: item.Source, heading: heading}
			sectionOrder = append(sectionOrder, mapKey)
		}
		sectionMap[mapKey].items = append(sectionMap[mapKey].items, item)
	}

	sort.SliceStable(sectionOrder, func(i, j int) bool {
		si := sectionMap[sectionOrder[i]]
		sj := sectionMap[sectionOrder[j]]
		if si.source != sj.source {
			return si.source < sj.source
		}
		return si.heading < sj.heading
	})

	for _, mapKey := range sectionOrder {
		sec := sectionMap[mapKey]
		fprintf(" │\n")
		fprintf(" ├─ %s / %s (%d)\n", strings.ToUpper(sec.source), sec.heading, len(sec.items))

		plugin := pluginFor(sec.source, pluginMap)
		plugin.RenderSection(sec.heading, sec.items)
	}
	fprintf(" │\n")
}
