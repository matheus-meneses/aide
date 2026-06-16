package api

import (
	"aide/cli/internal/agent"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type chatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

func (h *handlers) handleChat(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxChatBodyBytes)

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, `{"error":"message required"}`, http.StatusBadRequest)
		return
	}

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

	err := h.a.StreamChat(ctx, agent.ChatRequest{Message: req.Message, SessionID: req.SessionID}, func(chunk string) {
		data, _ := json.Marshal(map[string]string{"content": chunk})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	})

	switch {
	case errors.Is(err, agent.ErrLLMNotConfigured):
		errData, _ := json.Marshal(map[string]string{
			"error": "No AI model is configured yet, so I can't answer questions. Configure the agent to connect a model.",
			"code":  "llm_not_configured",
		})
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", errData)
	case err != nil:
		errData, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", errData)
	default:
		fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	}
	flusher.Flush()
}
