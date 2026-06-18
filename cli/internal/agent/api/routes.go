package api

import (
	"aide/cli/internal/agent"
	"net/http"
)

// Register mounts the agent's HTTP API onto mux. It replaces the former
// (*agent.Agent).RegisterRoutes, keeping the HTTP adapter out of the core.
func Register(a *agent.Agent, mux *http.ServeMux) {
	h := &handlers{a: a}

	mux.HandleFunc("GET /api/events", h.serveSSE)
	mux.HandleFunc("GET /api/notifications", h.handleNotifications)
	mux.HandleFunc("POST /api/chat", h.handleChat)
	mux.HandleFunc("GET /api/items", h.handleItems)
	mux.HandleFunc("GET /api/today", h.handleToday)
	mux.HandleFunc("GET /api/events/next", h.handleNextEvent)
	mux.HandleFunc("GET /api/events/upcoming", h.handleUpcomingEvents)
	mux.HandleFunc("POST /api/ui/command", h.handleUICommand)
	mux.HandleFunc("POST /api/ui/sync", h.handleUISync)
	mux.HandleFunc("GET /api/status", h.handleStatus)
	mux.HandleFunc("GET /api/memory", h.handleMemory)
	mux.HandleFunc("POST /api/exec", h.handleExec)
	mux.HandleFunc("GET /api/stats", h.handleStats)
	mux.HandleFunc("POST /api/ack", h.handleAck)
	mux.HandleFunc("GET /api/whoami", h.handleWhoami)
	mux.HandleFunc("POST /api/whoami", h.handleSetWhoami)
	mux.HandleFunc("GET /api/sessions", h.handleSessions)
	mux.HandleFunc("GET /api/sessions/{id}", h.handleSessionMessages)
	mux.HandleFunc("GET /api/ready", handleReady)
	mux.HandleFunc("GET /api/version", handleVersion)
	mux.HandleFunc("GET /api/version/check", handleVersionCheck)
	mux.HandleFunc("POST /api/update", h.handleUpdate)
	mux.HandleFunc("GET /api/runtime", h.handleRuntime)

	mux.HandleFunc("GET /api/setup/status", h.handleSetupStatus)
	mux.HandleFunc("POST /api/setup/bootstrap", h.handleSetupBootstrap)
	mux.HandleFunc("POST /api/setup/complete", h.handleSetupComplete)
	mux.HandleFunc("GET /api/providers", h.handleProviders)
	mux.HandleFunc("GET /api/plugins", h.handlePlugins)
	mux.HandleFunc("GET /api/plugins/{name}/manifest", h.handlePluginManifest)
	mux.HandleFunc("POST /api/plugins/install", h.handleInstallPlugin)
	mux.HandleFunc("POST /api/plugins/update", h.handleUpdatePlugin)
	mux.HandleFunc("POST /api/sources", h.handleAddSource)
	mux.HandleFunc("POST /api/sources/remove", h.handleRemoveSource)
	mux.HandleFunc("POST /api/llm", h.handleSetLLM)
	mux.HandleFunc("POST /api/test-connection", h.handleTestConnection)

	mux.HandleFunc("GET /api/config", h.handleConfigSnapshot)
	mux.HandleFunc("GET /api/sources", h.handleListSources)
	mux.HandleFunc("POST /api/sources/toggle", h.handleToggleSource)
	mux.HandleFunc("POST /api/plugins/uninstall", h.handleUninstallPlugin)
	mux.HandleFunc("POST /api/agent/schedule", h.handleSetSchedule)
	mux.HandleFunc("POST /api/settings", h.handleSetSettings)
	mux.HandleFunc("GET /api/team", h.handleGetTeam)
	mux.HandleFunc("POST /api/team", h.handleAddTeamMember)
	mux.HandleFunc("PUT /api/team/{id}", h.handleUpdateTeamMember)
	mux.HandleFunc("DELETE /api/team/{id}", h.handleDeleteTeamMember)
	mux.HandleFunc("GET /api/registries", h.handleListRegistries)
	mux.HandleFunc("POST /api/registries/add", h.handleAddRegistry)
	mux.HandleFunc("POST /api/registries/remove", h.handleRemoveRegistry)
	mux.HandleFunc("POST /api/registries/refresh", h.handleRefreshRegistries)
}
