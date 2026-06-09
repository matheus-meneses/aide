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
	mux.HandleFunc("GET /api/sessions", a.handleSessions)
	mux.HandleFunc("GET /api/sessions/{id}", a.handleSessionMessages)
	mux.HandleFunc("GET /api/version", handleVersion)

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
