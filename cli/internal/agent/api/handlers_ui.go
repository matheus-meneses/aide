package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (h *handlers) handleUICommand(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
		View   string `json:"view"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action required"})
		return
	}
	h.a.PublishUICommand(req.Action, req.View)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) handleUISync(w http.ResponseWriter, _ *http.Request) {
	go func() {
		h.a.PublishProgress("sync_progress", "Sync started")
		res, err := h.a.Scrape(detachedCtx(), nil)
		if err != nil {
			h.a.PublishProgress("sync_error", err.Error())
			return
		}
		h.a.PublishProgress("sync_done", fmt.Sprintf("%d/%d sources updated", res.SourcesOK, res.SourcesTotal))
	}()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}
