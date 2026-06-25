package api

import (
	"aide/cli/internal/persistence/store"
	"aide/cli/internal/setup/provision"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
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

func (h *handlers) handleSetUserContext(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Context string `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if err := provision.SetUserContext(h.a.ConfigPath(), req.Context); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.respondReload(w)
}

func (h *handlers) handleSetSourceContext(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Context string `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source name required"})
		return
	}
	if err := provision.SetSourceContext(h.a.ConfigPath(), req.Name, req.Context); err != nil {
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

type teamMemberRequest struct {
	Name                string   `json:"name"`
	Email               string   `json:"email"`
	Aliases             []string `json:"aliases"`
	Role                string   `json:"role"`
	Department          string   `json:"department"`
	Branch              string   `json:"branch"`
	Registration        string   `json:"registration"`
	ManagerID           *int64   `json:"manager_id"`
	ManagerRegistration string   `json:"manager_registration"`
}

type teamMemberResponse struct {
	ID                  int64    `json:"id"`
	Name                string   `json:"name"`
	Email               string   `json:"email"`
	Aliases             []string `json:"aliases"`
	Role                string   `json:"role"`
	Department          string   `json:"department"`
	Branch              string   `json:"branch"`
	Registration        string   `json:"registration"`
	ManagerID           *int64   `json:"manager_id"`
	ManagerRegistration string   `json:"manager_registration"`
	Source              string   `json:"source"`
}

func (req teamMemberRequest) toMember() store.Member {
	aliases := "[]"
	if len(req.Aliases) > 0 {
		if b, err := json.Marshal(req.Aliases); err == nil {
			aliases = string(b)
		}
	}
	return store.Member{
		Name:                strings.TrimSpace(req.Name),
		Email:               req.Email,
		Aliases:             aliases,
		Role:                req.Role,
		Department:          req.Department,
		Branch:              req.Branch,
		Registration:        req.Registration,
		ManagerID:           req.ManagerID,
		ManagerRegistration: req.ManagerRegistration,
	}
}

func toTeamResponse(m store.Member) teamMemberResponse {
	var aliases []string
	if m.Aliases != "" {
		_ = json.Unmarshal([]byte(m.Aliases), &aliases)
	}
	return teamMemberResponse{
		ID:                  m.ID,
		Name:                m.Name,
		Email:               m.Email,
		Aliases:             aliases,
		Role:                m.Role,
		Department:          m.Department,
		Branch:              m.Branch,
		Registration:        m.Registration,
		ManagerID:           m.ManagerID,
		ManagerRegistration: m.ManagerRegistration,
		Source:              m.Source,
	}
}

func wouldCycle(members []store.Member, memberID, managerID int64) bool {
	parent := make(map[int64]*int64, len(members))
	for _, m := range members {
		parent[m.ID] = m.ManagerID
	}
	for cur := managerID; ; {
		if cur == memberID {
			return true
		}
		next, ok := parent[cur]
		if !ok || next == nil {
			return false
		}
		cur = *next
	}
}

func (h *handlers) handleGetTeam(w http.ResponseWriter, _ *http.Request) {
	members, err := h.a.Store().Team.All()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	out := make([]teamMemberResponse, 0, len(members))
	for _, m := range members {
		out = append(out, toTeamResponse(m))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) handleAddTeamMember(w http.ResponseWriter, r *http.Request) {
	var req teamMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	created, err := h.a.Store().Team.Add(req.toMember())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, toTeamResponse(created))
}

func (h *handlers) handleUpdateTeamMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req teamMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if req.ManagerID != nil && *req.ManagerID == id {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "a member cannot manage themselves"})
		return
	}
	if req.ManagerID != nil {
		members, err := h.a.Store().Team.All()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if wouldCycle(members, id, *req.ManagerID) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manager selection would create a cycle"})
			return
		}
	}
	if err := h.a.Store().Team.Update(id, req.toMember()); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) handleDeleteTeamMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.a.Store().Team.Delete(id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
