package agent

import (
	"fmt"
	"log"
	"time"
)

func (a *Agent) postToChatAndSSE(content, timestamp string) {
	a.ensureWebSession()
	if err := a.store.Chat.InsertMessage("web-default", "assistant", content, timestamp); err != nil {
		log.Printf("[agent] failed to persist chat message: %v", err)
	}

	if a.bus != nil {
		a.bus.Publish(Event{
			Type: "chat_message",
			Data: fmt.Sprintf(`{"role":"assistant","content":%q,"timestamp":%q}`, content, timestamp),
		})
	}
}

func (a *Agent) ensureWebSession() {
	if err := a.store.Chat.CreateSession("web-default", time.Now().UTC().Format(time.RFC3339)); err != nil {
		log.Printf("[agent] failed to ensure web session: %v", err)
	}
}
