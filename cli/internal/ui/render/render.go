package render

import (
	"aide/cli/internal/persistence/store"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

var out io.Writer = os.Stdout

func SetOutput(w io.Writer) { out = w }

func fprintf(format string, a ...any) {
	fmt.Fprintf(out, format, a...)
}

func fprintln(a ...any) {
	fmt.Fprintln(out, a...)
}

func newTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
}

func relativeAge(isoStr string) string {
	t, err := time.Parse(time.RFC3339, isoStr)
	if err != nil {
		return "?"
	}
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}
}

func daysSince(isoStr string) int {
	t, err := time.Parse(time.RFC3339, isoStr)
	if err != nil {
		return 0
	}
	return int(time.Since(t).Hours() / 24)
}

func stripPrefix(title string) string {
	prefixes := []string{"Review MR: ", "Assigned MR: ", "Work Item: ", "Authored Item: ", "Grant: ", "Certification: ", "Meeting: "}
	for _, p := range prefixes {
		if strings.HasPrefix(title, p) {
			return strings.TrimPrefix(title, p)
		}
	}
	return title
}

func hyperlink(url, text string) string {
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}

func buildSparkline(data []store.DailyCount) string {
	if len(data) == 0 {
		return ""
	}
	blocks := []rune("▁▂▃▄▅▆▇█")
	minVal, maxVal := data[0].Count, data[0].Count
	for _, d := range data {
		if d.Count < minVal {
			minVal = d.Count
		}
		if d.Count > maxVal {
			maxVal = d.Count
		}
	}

	var sb strings.Builder
	spread := maxVal - minVal
	for _, d := range data {
		idx := 0
		if spread > 0 {
			idx = int(math.Round(float64(d.Count-minVal) / float64(spread) * float64(len(blocks)-1)))
		}
		sb.WriteRune(blocks[idx])
	}
	return sb.String()
}

func buildMetricSparkline(data []store.DailyMetric) string {
	if len(data) == 0 {
		return ""
	}
	blocks := []rune("▁▂▃▄▅▆▇█")
	minVal, maxVal := data[0].Value, data[0].Value
	for _, d := range data {
		if d.Value < minVal {
			minVal = d.Value
		}
		if d.Value > maxVal {
			maxVal = d.Value
		}
	}

	var sb strings.Builder
	spread := maxVal - minVal
	for _, d := range data {
		idx := 0
		if spread > 0 {
			idx = int(math.Round((d.Value - minVal) / spread * float64(len(blocks)-1)))
		}
		sb.WriteRune(blocks[idx])
	}
	return sb.String()
}
