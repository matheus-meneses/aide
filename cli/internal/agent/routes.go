package agent

import (
	"fmt"
	"io/fs"
	"net/http"
)

func (a *Agent) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/events", a.bus.ServeSSE)
	mux.HandleFunc("GET /api/notifications", a.handleNotifications)
	mux.HandleFunc("POST /api/chat", a.handleChat(a.bus))
	mux.HandleFunc("GET /api/items", a.handleItems)
	mux.HandleFunc("GET /api/today", a.handleToday)
	mux.HandleFunc("GET /api/status", a.handleStatus)
	mux.HandleFunc("GET /api/memory", a.handleMemory)
	mux.HandleFunc("POST /api/exec", a.handleExec)
	mux.HandleFunc("GET /api/stats", a.handleStats)
	mux.HandleFunc("POST /api/ack", a.handleAck)
	mux.HandleFunc("GET /api/whoami", a.handleWhoami)
	mux.HandleFunc("POST /api/whoami", a.handleSetWhoami)
	mux.HandleFunc("GET /api/sessions", a.handleSessions)
	mux.HandleFunc("GET /api/sessions/{id}", a.handleSessionMessages)
	mux.HandleFunc("GET /api/version", handleVersion)
	mux.HandleFunc("GET /api/runtime", a.handleRuntime)

	mux.HandleFunc("GET /api/setup/status", a.handleSetupStatus)
	mux.HandleFunc("POST /api/setup/bootstrap", a.handleSetupBootstrap)
	mux.HandleFunc("POST /api/setup/complete", a.handleSetupComplete)
	mux.HandleFunc("GET /api/providers", a.handleProviders)
	mux.HandleFunc("GET /api/plugins", a.handlePlugins)
	mux.HandleFunc("GET /api/plugins/{name}/manifest", a.handlePluginManifest)
	mux.HandleFunc("POST /api/plugins/install", a.handleInstallPlugin)
	mux.HandleFunc("POST /api/sources", a.handleAddSource)
	mux.HandleFunc("POST /api/sources/remove", a.handleRemoveSource)
	mux.HandleFunc("POST /api/llm", a.handleSetLLM)
	mux.HandleFunc("POST /api/test-connection", a.handleTestConnection)

	mux.HandleFunc("GET /api/config", a.handleConfigSnapshot)
	mux.HandleFunc("GET /api/sources", a.handleListSources)
	mux.HandleFunc("POST /api/sources/toggle", a.handleToggleSource)
	mux.HandleFunc("POST /api/plugins/uninstall", a.handleUninstallPlugin)
	mux.HandleFunc("POST /api/agent/schedule", a.handleSetSchedule)
	mux.HandleFunc("POST /api/settings", a.handleSetSettings)
	mux.HandleFunc("GET /api/team", a.handleGetTeam)
	mux.HandleFunc("POST /api/team", a.handleSetTeam)
	mux.HandleFunc("GET /api/registries", a.handleListRegistries)
	mux.HandleFunc("POST /api/registries/add", a.handleAddRegistry)
	mux.HandleFunc("POST /api/registries/remove", a.handleRemoveRegistry)
	mux.HandleFunc("POST /api/registries/refresh", a.handleRefreshRegistries)

	a.registerStaticRoutes(mux)
}

func (a *Agent) registerStaticRoutes(mux *http.ServeMux) {
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<h1>Aide</h1><p>Frontend not built. Run: cd cli/internal/agent/frontend && npm run build</p>")
		})
		return
	}

	fileServer := http.FileServer(http.FS(distFS))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			if _, err := fs.Stat(distFS, r.URL.Path[1:]); err != nil {
				r.URL.Path = "/"
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}
