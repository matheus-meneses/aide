package api

import (
	"aide/cli/internal/setup/bootstrap"
	"context"
	"net/http"
)

func detachedCtx() context.Context { return context.Background() }

func (h *handlers) handleSetupStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{
		"needs_setup":  bootstrap.NeedsSetup(),
		"python_ready": bootstrap.PythonReady(),
	})
}

func (h *handlers) handleSetupComplete(w http.ResponseWriter, _ *http.Request) {
	if err := bootstrap.MarkSetupComplete(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "complete"})
}

func (h *handlers) handleSetupBootstrap(w http.ResponseWriter, _ *http.Request) {
	go func() {
		err := bootstrap.Ensure(func(msg string) {
			h.a.PublishProgress("setup_progress", msg)
		})
		if err != nil {
			h.a.PublishProgress("setup_error", err.Error())
			return
		}
		h.a.PublishProgress("setup_done", "ready")
	}()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}
