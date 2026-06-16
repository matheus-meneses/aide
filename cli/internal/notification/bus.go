package notification

import (
	"aide/cli/internal/agent/events"
	"fmt"
)

type BusNotifier struct {
	Bus *events.EventBus
}

func (n *BusNotifier) Notify(title, body string) error {
	n.Bus.Publish(events.Event{
		Type:     "notification",
		Priority: "urgent",
		Data:     fmt.Sprintf(`{"title":%q,"body":%q}`, title, body),
	})
	return nil
}
