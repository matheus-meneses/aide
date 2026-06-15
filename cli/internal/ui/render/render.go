package render

import (
	"fmt"
	"io"
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
