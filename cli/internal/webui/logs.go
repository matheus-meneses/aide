package webui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"aide/cli/internal/clog"
)

func registerLogs(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/logs", handleLogStream)
}

func handleLogStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	for _, entry := range clog.Recent() {
		writeLogEntry(w, entry)
	}
	flusher.Flush()

	ch, unsub := clog.Subscribe()
	defer unsub()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			writeLogEntry(w, entry)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func writeLogEntry(w http.ResponseWriter, entry clog.LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
}
