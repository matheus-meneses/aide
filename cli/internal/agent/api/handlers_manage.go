package api

import (
	"aide/cli/internal/platform/config"
	"aide/cli/internal/setup/provision"
	"encoding/json"
	"net/http"
)

func (h *handlers) handleConfigSnapshot(w http.ResponseWriter, _ *http.Request) {
	snap, err := provision.ConfigSnapshot(h.a.ConfigPath())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (h *handlers) handleListSources(w http.ResponseWriter, _ *http.Request) {
	sources, err := provision.ListSources(h.a.ConfigPath())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sources)
}

func (h *handlers) handleToggleSource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source name required"})
		return
	}
	if err := provision.SetSourceEnabled(h.a.ConfigPath(), req.Name, req.Enabled); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.respondReload(w)
}

func (h *handlers) handleUninstallPlugin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "plugin name required"})
		return
	}
	if err := provision.UninstallPlugin(h.a.ConfigPath(), req.Name); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.respondReload(w)
}

func (h *handlers) handleSetSchedule(w http.ResponseWriter, r *http.Request) {
	var in provision.ScheduleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if err := provision.SetSchedule(h.a.ConfigPath(), in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.respondReload(w)
}

func (h *handlers) handleSetSettings(w http.ResponseWriter, r *http.Request) {
	var in provision.GeneralSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if err := provision.SetGeneralSettings(h.a.ConfigPath(), in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.respondReload(w)
}

func (h *handlers) handleGetTeam(w http.ResponseWriter, _ *http.Request) {
	members, err := provision.GetTeam(h.a.ConfigPath())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, members)
}

func (h *handlers) handleSetTeam(w http.ResponseWriter, r *http.Request) {
	var members []config.TeamMember
	if err := json.NewDecoder(r.Body).Decode(&members); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if err := provision.SetTeam(h.a.ConfigPath(), members); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.respondReload(w)
}

func (h *handlers) handleListRegistries(w http.ResponseWriter, _ *http.Request) {
	registries, err := provision.ListRegistries(h.a.ConfigPath())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, registries)
}

func (h *handlers) handleAddRegistry(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL   string `json:"url"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "registry url required"})
		return
	}
	if err := provision.AddRegistry(h.a.ConfigPath(), req.URL, req.Token); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) handleRemoveRegistry(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "registry url required"})
		return
	}
	if err := provision.RemoveRegistry(h.a.ConfigPath(), req.URL); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) handleRefreshRegistries(w http.ResponseWriter, _ *http.Request) {
	count, err := provision.RefreshCatalog(h.a.ConfigPath())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"plugins": count})
}

// respondReload reloads the live config (refreshing the runner, LLM, tools, and
// team) after a successful write and reports the outcome.
func (h *handlers) respondReload(w http.ResponseWriter) {
	if err := h.a.ReloadConfig(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
