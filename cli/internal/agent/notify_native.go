package agent

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// nativeNotify shows an OS-level notification so alerts surface even when the
// app window is closed but the process is resident in the menu bar.
func nativeNotify(title, body string) {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf("display notification %s with title %s", osaQuote(body), osaQuote(title))
		_ = exec.Command("osascript", "-e", script).Start() //nolint:errcheck // best-effort UX notification
	case "linux":
		_ = exec.Command("notify-send", title, body).Start() //nolint:errcheck // best-effort UX notification
	}
}

func osaQuote(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", " ")
	return "\"" + s + "\""
}

// MacNotifier fires native OS notifications via nativeNotify.
type MacNotifier struct{}

func (n *MacNotifier) Notify(title, body string) error {
	nativeNotify(title, body)
	return nil
}

// MultiNotifier fans a notification out to several notifiers.
type MultiNotifier struct {
	notifiers []Notifier
}

func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

func (m *MultiNotifier) Notify(title, body string) error {
	var firstErr error
	for _, n := range m.notifiers {
		if err := n.Notify(title, body); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
