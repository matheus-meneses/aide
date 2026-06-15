//go:build cgo

package main

import (
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

type wailsNotifier struct {
	svc *notifications.NotificationService
	seq atomic.Uint64
}

func (n *wailsNotifier) Notify(title, body string) error {
	if title == "" {
		title = "Aide"
	}
	return n.svc.SendNotification(notifications.NotificationOptions{
		ID:    strconv.FormatUint(n.seq.Add(1), 10),
		Title: title,
		Body:  body,
	})
}

func isBundled() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	return strings.Contains(exe, ".app/Contents/MacOS/")
}
