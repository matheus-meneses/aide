package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func (h *handlers) serveSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	lastID := uint64(0)
	if idStr := r.Header.Get("Last-Event-ID"); idStr != "" {
		if parsed, err := strconv.ParseUint(idStr, 10, 64); err == nil {
			lastID = parsed
		}
	}

	if lastID > 0 {
		missed := h.a.Bus().Ring().Since(lastID)
		for _, event := range missed {
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Type, data)
		}
		flusher.Flush()
	}

	ch, unsub := h.a.Bus().Subscribe()
	defer unsub()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Type, data)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}
