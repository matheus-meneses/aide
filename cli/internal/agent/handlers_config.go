package agent

import (
	"aide/cli/internal/keychain"
	"aide/cli/internal/provision"
	"encoding/json"
	"net/http"
)

func (a *Agent) handleProviders(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, provision.Providers())
}

func (a *Agent) handlePlugins(w http.ResponseWriter, _ *http.Request) {
	items, err := provision.ListPlugins(a.configPathOrDefault())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type fieldDTO struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Type     string `json:"type"`
	Default  string `json:"default"`
	Required bool   `json:"required"`
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

func (a *Agent) handlePluginManifest(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	m, err := provision.PluginManifest(name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	dto := manifestDTO{Name: m.Name, Description: m.Description}
	for _, f := range m.Config {
		dto.Config = append(dto.Config, fieldDTO{Key: f.Key, Label: f.Label, Type: f.Type, Default: f.Default, Required: f.Required})
	}
	for _, c := range m.Credentials {
		dto.Credentials = append(dto.Credentials, credDTO{Key: c.Key, Label: c.Label, Secret: c.Secret})
	}
	writeJSON(w, http.StatusOK, dto)
}

func (a *Agent) handleInstallPlugin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "plugin name required"})
		return
	}

	go func() {
		a.publishProgress("install_progress", "Installing "+req.Name+"…")
		if _, err := provision.InstallPlugin(detachedCtx(), req.Name, req.Version); err != nil {
			a.publishProgress("install_error", err.Error())
			return
		}
		a.publishProgress("install_done", req.Name)
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

func (a *Agent) handleAddSource(w http.ResponseWriter, r *http.Request) {
	var in provision.SourceInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if err := provision.AddSource(a.configPathOrDefault(), in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := a.ReloadConfig(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *Agent) handleRemoveSource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source name required"})
		return
	}
	if err := provision.RemoveSource(a.configPathOrDefault(), req.Name); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := a.ReloadConfig(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *Agent) handleSetLLM(w http.ResponseWriter, r *http.Request) {
	var in provision.LLMInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if err := provision.SetLLM(a.configPathOrDefault(), in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := a.ReloadConfig(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *Agent) handleTestConnection(w http.ResponseWriter, r *http.Request) {
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
		req.APIKey = a.storedAPIKey()
	}
	if err := TestLLM(req.Provider, req.BaseURL, req.Model, req.APIKey); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Agent) storedAPIKey() string {
	if cfg := a.getConfig(); cfg != nil && cfg.Agent.LLMAPIKey != "" {
		return cfg.Agent.LLMAPIKey
	}
	if cred, err := keychain.GetAll("agent"); err == nil {
		return cred.Fields["llm_api_key"]
	}
	return ""
}
