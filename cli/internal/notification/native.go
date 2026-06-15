package notification

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Native shows an OS-level notification so alerts surface even when the
// app window is closed but the process is resident in the menu bar.
func Native(title, body string) {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf("display notification %s with title %s", osaQuote(body), osaQuote(title))
		_ = exec.Command("osascript", "-e", script).Start()
	case "linux":
		_ = exec.Command("notify-send", title, body).Start()
	}
}

func osaQuote(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", " ")
	return "\"" + s + "\""
}

// MacNotifier fires native OS notifications via Native.
type MacNotifier struct{}

func (n *MacNotifier) Notify(title, body string) error {
	Native(title, body)
	return nil
}
