package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type webChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

const maxChatBodyBytes = 1 << 20

func (a *Agent) handleChat(_ *EventBus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxChatBodyBytes)

		var req webChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
			return
		}

		if req.Message == "" {
			http.Error(w, `{"error":"message required"}`, http.StatusBadRequest)
			return
		}

		if req.SessionID == "" {
			req.SessionID = "default"
		}

		sess := a.sessions.getOrCreate(req.SessionID)
		sess.mu.Lock()
		defer sess.mu.Unlock()

		if len(sess.history) == 0 {
			sysCtx, err := BuildContext(a.store)
			if err != nil {
				http.Error(w, `{"error":"context build failed"}`, http.StatusInternalServerError)
				return
			}
			sess.history = []ChatMessage{{Role: "system", Content: sysCtx}}

			if persisted, err := a.store.Chat.LoadMessages(req.SessionID); err == nil {
				for _, m := range persisted {
					if m.Role == "user" || m.Role == "assistant" {
						sess.history = append(sess.history, ChatMessage{Role: m.Role, Content: m.Content})
					}
				}
			}
		}

		sess.history = append(sess.history, ChatMessage{Role: "user", Content: req.Message})
		sess.history = TrimHistory(sess.history, 30000)

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher.Flush()

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()

		now := time.Now().UTC().Format(time.RFC3339)
		if err := a.store.Chat.InsertMessage(req.SessionID, "user", req.Message, now); err != nil {
			log.Printf("[agent] failed to persist user message: %v", err)
		}

		full, usage, err := a.llm.ChatStream(ctx, sess.history, func(chunk string) {
			data, _ := json.Marshal(map[string]string{"content": chunk})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		})
		if err != nil {
			errData, _ := json.Marshal(map[string]string{"error": err.Error()})
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", errData)
			flusher.Flush()
			sess.history = sess.history[:len(sess.history)-1]
			return
		}

		if usage != nil {
			if err := a.store.Tokens.Record("chat", a.llm.Model(), usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens); err != nil {
				log.Printf("[agent] failed to record token usage: %v", err)
			}
		}

		sess.history = append(sess.history, ChatMessage{Role: "assistant", Content: full})

		if err := a.store.Chat.InsertMessage(req.SessionID, "assistant", full, time.Now().UTC().Format(time.RFC3339)); err != nil {
			log.Printf("[agent] failed to persist assistant message: %v", err)
		}

		fmt.Fprintf(w, "event: done\ndata: {}\n\n")
		flusher.Flush()
	}
}

func (a *Agent) handleSessions(w http.ResponseWriter, _ *http.Request) {
	sessions, err := a.store.Chat.ListSessions(20)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (a *Agent) handleSessionMessages(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session id required"})
		return
	}

	msgs, err := a.store.Chat.LoadMessages(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}
