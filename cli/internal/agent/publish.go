package agent

import (
	"aide/cli/internal/agent/events"
	"fmt"
	"time"
)

func (a *Agent) postToChatAndSSE(content, timestamp string) {
	a.ensureWebSession()
	if err := a.store.Chat.InsertMessage("web-default", "assistant", content, timestamp); err != nil {
		alog.Warn("failed to persist chat message: %v", err)
	}

	if a.bus != nil {
		a.bus.Publish(events.Event{
			Type: "chat_message",
			Data: fmt.Sprintf(`{"role":"assistant","content":%q,"timestamp":%q}`, content, timestamp),
		})
	}
}

// PublishProgress emits a progress event on the bus, used by the HTTP API
// adapter for long-running setup/install operations.
func (a *Agent) PublishProgress(eventType, msg string) {
	if a.bus != nil {
		a.bus.Publish(events.Event{Type: eventType, Data: fmt.Sprintf(`{"message":%q}`, msg)})
	}
}

// PublishUICommand emits a ui_command event consumed by the desktop shell (to
// show/quit the window) and the main web UI (to navigate to a view).
func (a *Agent) PublishUICommand(action, view string) {
	if a.bus != nil {
		a.bus.Publish(events.Event{Type: "ui_command", Data: fmt.Sprintf(`{"action":%q,"view":%q}`, action, view)})
	}
}

func (a *Agent) ensureWebSession() {
	if err := a.store.Chat.CreateSession("web-default", time.Now().UTC().Format(time.RFC3339)); err != nil {
		alog.Warn("failed to ensure web session: %v", err)
	}
}
