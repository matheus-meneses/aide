package agent

import (
	"aide/cli/internal/config"
	"aide/cli/internal/provision"
	"encoding/json"
	"net/http"
)

func (a *Agent) handleConfigSnapshot(w http.ResponseWriter, _ *http.Request) {
	snap, err := provision.ConfigSnapshot(a.configPathOrDefault())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (a *Agent) handleListSources(w http.ResponseWriter, _ *http.Request) {
	sources, err := provision.ListSources(a.configPathOrDefault())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sources)
}

func (a *Agent) handleToggleSource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source name required"})
		return
	}
	if err := provision.SetSourceEnabled(a.configPathOrDefault(), req.Name, req.Enabled); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	a.respondReload(w)
}

func (a *Agent) handleUninstallPlugin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "plugin name required"})
		return
	}
	if err := provision.UninstallPlugin(a.configPathOrDefault(), req.Name); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	a.respondReload(w)
}

func (a *Agent) handleSetSchedule(w http.ResponseWriter, r *http.Request) {
	var in provision.ScheduleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if err := provision.SetSchedule(a.configPathOrDefault(), in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	a.respondReload(w)
}

func (a *Agent) handleSetSettings(w http.ResponseWriter, r *http.Request) {
	var in provision.GeneralSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if err := provision.SetGeneralSettings(a.configPathOrDefault(), in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	a.respondReload(w)
}

func (a *Agent) handleGetTeam(w http.ResponseWriter, _ *http.Request) {
	members, err := provision.GetTeam(a.configPathOrDefault())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, members)
}

func (a *Agent) handleSetTeam(w http.ResponseWriter, r *http.Request) {
	var members []config.TeamMember
	if err := json.NewDecoder(r.Body).Decode(&members); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if err := provision.SetTeam(a.configPathOrDefault(), members); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	a.respondReload(w)
}

func (a *Agent) handleListRegistries(w http.ResponseWriter, _ *http.Request) {
	registries, err := provision.ListRegistries(a.configPathOrDefault())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, registries)
}

func (a *Agent) handleAddRegistry(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL   string `json:"url"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "registry url required"})
		return
	}
	if err := provision.AddRegistry(a.configPathOrDefault(), req.URL, req.Token); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *Agent) handleRemoveRegistry(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "registry url required"})
		return
	}
	if err := provision.RemoveRegistry(a.configPathOrDefault(), req.URL); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *Agent) handleRefreshRegistries(w http.ResponseWriter, _ *http.Request) {
	count, err := provision.RefreshCatalog(a.configPathOrDefault())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"plugins": count})
}

// respondReload reloads the live config (refreshing the runner, LLM, tools, and
// team) after a successful write and reports the outcome.
func (a *Agent) respondReload(w http.ResponseWriter) {
	if err := a.ReloadConfig(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
