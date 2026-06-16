package widgets

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	Out io.Writer = os.Stdout
	Err io.Writer = os.Stderr

	colorEnabled = detectColor()
	quiet        bool
)

func detectColor() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("AIDE_NO_COLOR") != "" {
		return false
	}
	return isTerminal(os.Stdout)
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// SetOutput overrides the writers and re-evaluates color support; intended for
// tests and for callers that pipe output elsewhere.
func SetOutput(stdout, stderr io.Writer) {
	Out = stdout
	Err = stderr
	colorEnabled = os.Getenv("NO_COLOR") == "" && os.Getenv("AIDE_NO_COLOR") == "" && isTerminal(stdout)
}

// ColorEnabled reports whether styled output is active.
func ColorEnabled() bool { return colorEnabled }

// SetColorEnabled forces color output on or off, overriding auto-detection.
func SetColorEnabled(enabled bool) { colorEnabled = enabled }

// SetQuiet toggles suppression of incidental informational output. Errors,
// warnings, and primary command output are always shown.
func SetQuiet(q bool) { quiet = q }

// Quiet reports whether incidental output is suppressed.
func Quiet() bool { return quiet }

const (
	SymCheck = "✓"
	SymCross = "✗"
	SymDot   = "•"
	SymArrow = "→"
	SymWarn  = "⚠"
	SymInfo  = "ℹ"
)

var (
	styleHeading = lipgloss.NewStyle().Bold(true)
	styleKey     = lipgloss.NewStyle().Bold(true)
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styleInfo    = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	styleMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func paint(s lipgloss.Style, text string) string {
	if !colorEnabled {
		return text
	}
	return s.Render(text)
}

func Bold(text string) string    { return paint(styleHeading, text) }
func Muted(text string) string   { return paint(styleMuted, text) }
func Success(text string) string { return paint(styleSuccess, text) }
func Warn(text string) string    { return paint(styleWarn, text) }
func Danger(text string) string  { return paint(styleError, text) }
func Info(text string) string    { return paint(styleInfo, text) }

// Heading renders a bold title followed by a blank line.
func Heading(title string) {
	if quiet {
		return
	}
	fmt.Fprintln(Out, paint(styleHeading, title))
}

func line(w io.Writer, sym string, style lipgloss.Style, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "%s %s\n", paint(style, sym), msg)
}

func PrintSuccess(format string, args ...any) { line(Out, SymCheck, styleSuccess, format, args...) }
func PrintWarn(format string, args ...any)    { line(Out, SymWarn, styleWarn, format, args...) }

func PrintInfo(format string, args ...any) {
	if quiet {
		return
	}
	line(Out, SymInfo, styleInfo, format, args...)
}

// PrintError writes a styled error line to stderr.
func PrintError(format string, args ...any) { line(Err, SymCross, styleError, format, args...) }

// Print, Printf, and Println write plain text to the configured stdout.
func Print(args ...any)                 { fmt.Fprint(Out, args...) }
func Printf(format string, args ...any) { fmt.Fprintf(Out, format, args...) }
func Println(args ...any)               { fmt.Fprintln(Out, args...) }

// KeyValue prints an aligned "key: value" pair with the key emphasized.
func KeyValue(key, value string) {
	fmt.Fprintf(Out, "  %s %s\n", paint(styleKey, key+":"), value)
}

// Bullet prints an indented bullet line.
func Bullet(format string, args ...any) {
	if quiet {
		return
	}
	fmt.Fprintf(Out, "  %s %s\n", paint(styleMuted, SymDot), fmt.Sprintf(format, args...))
}

// Table renders headers + rows in aligned columns; headers are emphasized.
func Table(headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(Out, 0, 0, 2, ' ', 0)
	styled := make([]string, len(headers))
	for i, h := range headers {
		styled[i] = paint(styleHeading, h)
	}
	fmt.Fprintln(tw, strings.Join(styled, "\t"))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	_ = tw.Flush()
}
