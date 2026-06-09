package render

import (
	"aide/cli/internal/store"
	"fmt"
	"strings"
)

func printStatsReport(openItems []store.Item, resolvedCounts map[string]int, history []store.DailyCount, avgResAge float64, days int, latestMetrics []store.Metric, metricHistories map[string][]store.DailyMetric, pluginMap map[string]string) {
	fprintf("\n Stats (last %d days)\n", days)

	if len(latestMetrics) > 0 {
		fprintf("\n Metrics\n\n")
		w := newTabWriter()
		fmt.Fprintf(w, "  SOURCE\tMETRIC\tCURRENT\tTREND\tDELTA\n")
		fmt.Fprintf(w, "  ------\t------\t-------\t-----\t-----\n")

		for _, m := range latestMetrics {
			key := m.Source + "|" + m.Name
			spark := ""
			deltaStr := ""
			if hist, ok := metricHistories[key]; ok && len(hist) > 1 {
				spark = buildMetricSparkline(hist)
				first := hist[0].Value
				last := hist[len(hist)-1].Value
				d := last - first
				switch {
				case d > 0:
					deltaStr = fmt.Sprintf("+%.0f", d)
				case d < 0:
					deltaStr = fmt.Sprintf("%.0f", d)
				default:
					deltaStr = "="
				}
			}
			fmt.Fprintf(w, "  %s\t%s\t%.0f\t%s\t%s\n", m.Source, m.Name, m.Value, spark, deltaStr)
		}
		w.Flush()
	}

	fprintf("\n Age Analysis\n\n")
	type ageInfo struct {
		section string
		count   int
		oldest  int
		total   int
	}
	ageMap := make(map[string]*ageInfo)
	var ageOrder []string

	for _, item := range openItems {
		plugin := pluginFor(item.Source, pluginMap)
		heading := plugin.Classify(item)
		key := strings.ToUpper(item.Source) + " / " + heading
		if _, exists := ageMap[key]; !exists {
			ageMap[key] = &ageInfo{section: key}
			ageOrder = append(ageOrder, key)
		}
		ai := ageMap[key]
		ai.count++
		ageDays := daysSince(item.FirstSeenAt)
		ai.total += ageDays
		if ageDays > ai.oldest {
			ai.oldest = ageDays
		}
	}

	w := newTabWriter()
	fmt.Fprintf(w, "  SECTION\tCOUNT\tOLDEST\tAVG\n")
	fmt.Fprintf(w, "  -------\t-----\t------\t---\n")
	for _, key := range ageOrder {
		ai := ageMap[key]
		avg := 0
		if ai.count > 0 {
			avg = ai.total / ai.count
		}
		fmt.Fprintf(w, "  %s\t%d\t%dd\t%dd\n", ai.section, ai.count, ai.oldest, avg)
	}
	w.Flush()

	if len(history) > 1 {
		fprintf("\n History\n\n")
		spark := buildSparkline(history)
		current := 0
		if len(history) > 0 {
			current = history[len(history)-1].Count
		}
		first := 0
		if len(history) > 0 {
			first = history[0].Count
		}
		delta := current - first
		deltaStr := fmt.Sprintf("%+d", delta)
		if delta == 0 {
			deltaStr = "="
		}
		fprintf("  Open items: %s  %d (%s)\n", spark, current, deltaStr)
	}

	fprintf("\n Velocity\n\n")
	w2 := newTabWriter()
	fmt.Fprintf(w2, "  SOURCE\tRESOLVED\tPER DAY\tAVG RESOLUTION AGE\n")
	fmt.Fprintf(w2, "  ------\t--------\t-------\t------------------\n")

	totalResolved := 0
	for source, count := range resolvedCounts {
		totalResolved += count
		perDay := float64(count) / float64(days)
		fmt.Fprintf(w2, "  %s\t%d\t%.1f\t%.0fd\n", source, count, perDay, avgResAge)
	}
	if totalResolved == 0 {
		fmt.Fprintf(w2, "  (no items resolved in this period)\t\t\t\n")
	}
	w2.Flush()

	fprintln()
}
