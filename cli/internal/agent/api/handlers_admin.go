package api

import (
	"aide/cli/internal/agent"
	"aide/cli/internal/runtime/updater"
	"encoding/json"
	"errors"
	"net/http"
	"runtime"
	"strconv"
	"strings"
)

func (h *handlers) handleMemory(w http.ResponseWriter, _ *http.Request) {
	mem, err := h.a.Store().Memory.LoadLast()
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no memory stored yet"})
		return
	}
	writeJSON(w, http.StatusOK, mem)
}

func (h *handlers) handleExec(w http.ResponseWriter, r *http.Request) {
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

	result := h.a.ExecuteCommand(r.Context(), req.Command)
	writeJSON(w, http.StatusOK, result)
}

func (h *handlers) handleStats(w http.ResponseWriter, _ *http.Request) {
	summary, err := h.a.Store().Tokens.Stats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	daily, _ := h.a.Store().Tokens.DailyStats(7)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"summary": summary,
		"daily":   daily,
	})
}

func (h *handlers) handleAck(w http.ResponseWriter, r *http.Request) {
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
	if err := h.a.Store().Acks.Add(req.Fingerprint, req.Title); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}

func (h *handlers) handleWhoami(w http.ResponseWriter, _ *http.Request) {
	profile, err := h.a.Store().Profile.All()
	if err != nil || len(profile) == 0 {
		writeJSON(w, http.StatusOK, map[string]string{})
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (h *handlers) handleSetWhoami(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string `json:"name"`
		Email         string `json:"email"`
		PreferredName string `json:"preferred_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if err := h.a.Store().Profile.SetIdentity(req.Name, req.Email, req.PreferredName); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	profile, _ := h.a.Store().Profile.All()
	writeJSON(w, http.StatusOK, profile)
}

func (h *handlers) handleNotifications(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	events := h.a.Bus().Ring().Recent(limit)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  len(events),
	})
}

func handleReady(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleVersion serves the running version immediately. Update information is
// served from an in-memory cache (empty on a cold start) and a throttled
// background refresh is kicked off, so the call never blocks on GitHub.
func handleVersion(w http.ResponseWriter, _ *http.Request) {
	info, _ := updater.CachedUpgradeInfo()
	updater.RefreshUpgradeInfoAsync(agent.Version)
	writeVersionInfo(w, info)
}

// handleVersionCheck forces a synchronous GitHub check and refreshes the cache.
// It backs the explicit "Check for updates" action in the About tab.
func handleVersionCheck(w http.ResponseWriter, _ *http.Request) {
	writeVersionInfo(w, updater.RefreshUpgradeInfo(agent.Version))
}

func writeVersionInfo(w http.ResponseWriter, info updater.UpgradeInfo) {
	method := updater.DetectMethod(agent.Version)
	writeJSON(w, http.StatusOK, map[string]any{
		"current":          agent.Version,
		"latest":           info.Latest,
		"update_available": info.UpdateAvailable,
		"update_url":       updater.InstallURL(),
		"can_self_update":  method.CanSelfUpdate(),
		"notes":            info.Notes,
		"release_url":      info.ReleaseURL,
		"platform":         runtime.GOOS + "/" + runtime.GOARCH,
	})
}

func (h *handlers) handleUpdate(w http.ResponseWriter, _ *http.Request) {
	method := updater.DetectMethod(agent.Version)
	if !method.CanSelfUpdate() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "this installation can't update itself"})
		return
	}

	go func() {
		h.a.PublishProgress("update_progress", "Starting update…")
		prog := updater.Progress(func(line string) {
			h.a.PublishProgress("update_progress", line)
		})
		res, err := updater.Apply(detachedCtx(), agent.Version, method, prog)
		if err != nil {
			if errors.Is(err, updater.ErrUpToDate) {
				h.a.PublishProgress("update_done", agent.Version)
				return
			}
			h.a.PublishProgress("update_error", err.Error())
			return
		}
		h.a.PublishProgress("update_done", res.Version)
		if res.RestartNow {
			h.a.RequestRestart()
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

func (h *handlers) handleRuntime(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"native_notifications": h.a.NativeNotifications(),
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
