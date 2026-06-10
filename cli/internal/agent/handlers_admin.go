package agent

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func (a *Agent) handleMemory(w http.ResponseWriter, _ *http.Request) {
	mem, err := a.store.Memory.LoadLast()
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no memory stored yet"})
		return
	}
	writeJSON(w, http.StatusOK, mem)
}

func (a *Agent) handleExec(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxChatBodyBytes)
	var req struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.Command == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "command required"})
		return
	}

	result := a.executeCommand(r.Context(), req.Command)
	writeJSON(w, http.StatusOK, result)
}

func (a *Agent) handleStats(w http.ResponseWriter, _ *http.Request) {
	summary, err := a.store.Tokens.Stats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	daily, _ := a.store.Tokens.DailyStats(7)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"summary": summary,
		"daily":   daily,
	})
}

func (a *Agent) handleAck(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Fingerprint string `json:"fingerprint"`
		Title       string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Fingerprint == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "fingerprint required"})
		return
	}
	if req.Title == "" {
		req.Title = req.Fingerprint
	}
	if err := a.store.Acks.Add(req.Fingerprint, req.Title); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}

func (a *Agent) handleWhoami(w http.ResponseWriter, _ *http.Request) {
	profile, err := a.store.Profile.All()
	if err != nil || len(profile) == 0 {
		writeJSON(w, http.StatusOK, map[string]string{})
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (a *Agent) handleNotifications(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	events := a.bus.Ring().Recent(limit)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  len(events),
	})
}

func handleVersion(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"current":    Version,
		"update_url": "https://raw.githubusercontent.com/matheus-meneses/aide/main/assets/deploy/install.sh",
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
