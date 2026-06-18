package api

import (
	"aide/cli/internal/agent"
	"aide/cli/internal/runtime/plugin"
	"aide/cli/internal/setup/provision"
	"encoding/json"
	"net/http"
)

func (h *handlers) handleProviders(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, provision.Providers())
}

func (h *handlers) handlePlugins(w http.ResponseWriter, _ *http.Request) {
	items, err := provision.ListPlugins(h.a.ConfigPath())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type fieldDTO struct {
	Key      string     `json:"key"`
	Label    string     `json:"label"`
	Type     string     `json:"type"`
	Default  string     `json:"default"`
	Required bool       `json:"required"`
	Fields   []fieldDTO `json:"fields,omitempty"`
}

func toFieldDTO(f plugin.Field) fieldDTO {
	dto := fieldDTO{Key: f.Key, Label: f.Label, Type: f.Type, Default: f.Default, Required: f.Required}
	for _, sub := range f.Fields {
		dto.Fields = append(dto.Fields, toFieldDTO(sub))
	}
	return dto
}

type credDTO struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Secret bool   `json:"secret"`
}

type manifestDTO struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Config      []fieldDTO `json:"config"`
	Credentials []credDTO  `json:"credentials"`
}

func (h *handlers) handlePluginManifest(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	m, err := provision.PluginManifest(name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	dto := manifestDTO{Name: m.Name, Description: m.Description}
	for _, f := range m.Config {
		dto.Config = append(dto.Config, toFieldDTO(f))
	}
	for _, c := range m.Credentials {
		dto.Credentials = append(dto.Credentials, credDTO{Key: c.Key, Label: c.Label, Secret: c.Secret})
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *handlers) handleInstallPlugin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                    string `json:"name"`
		Version                 string `json:"version"`
		AcknowledgeCapabilities bool   `json:"acknowledge_capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "plugin name required"})
		return
	}
	if !req.AcknowledgeCapabilities {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "plugin capabilities must be acknowledged before install"})
		return
	}

	go func() {
		h.a.PublishProgress("install_progress", "Installing "+req.Name+"…")
		if _, err := provision.InstallPlugin(detachedCtx(), h.a.ConfigPath(), req.Name, req.Version, req.AcknowledgeCapabilities); err != nil {
			h.a.PublishProgress("install_error", err.Error())
			return
		}
		h.a.PublishProgress("install_done", req.Name)
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

func (h *handlers) handleUpdatePlugin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                    string `json:"name"`
		AcknowledgeCapabilities bool   `json:"acknowledge_capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "plugin name required"})
		return
	}
	if !req.AcknowledgeCapabilities {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "plugin capabilities must be acknowledged before update"})
		return
	}

	go func() {
		h.a.PublishProgress("install_progress", "Updating "+req.Name+"…")
		if _, err := provision.UpdatePlugin(detachedCtx(), h.a.ConfigPath(), req.Name, req.AcknowledgeCapabilities); err != nil {
			h.a.PublishProgress("install_error", err.Error())
			return
		}
		h.a.PublishProgress("install_done", req.Name)
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

func (h *handlers) handleAddSource(w http.ResponseWriter, r *http.Request) {
	var in provision.SourceInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if err := provision.AddSource(h.a.ConfigPath(), in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.a.ReloadConfig(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) handleRemoveSource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source name required"})
		return
	}
	if err := provision.RemoveSource(h.a.ConfigPath(), req.Name); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.a.ReloadConfig(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) handleSetLLM(w http.ResponseWriter, r *http.Request) {
	var in provision.LLMInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if err := provision.SetLLM(h.a.ConfigPath(), in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.a.ReloadConfig(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
		BaseURL  string `json:"base_url"`
		Model    string `json:"model"`
		APIKey   string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.APIKey == "" {
		req.APIKey = h.a.StoredAPIKey()
	}
	if err := agent.TestLLM(req.Provider, req.BaseURL, req.Model, req.APIKey); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
