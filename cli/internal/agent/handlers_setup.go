package agent

import (
	"aide/cli/internal/bootstrap"
	"context"
	"fmt"
	"net/http"
)

func detachedCtx() context.Context { return context.Background() }

func (a *Agent) publishProgress(eventType, msg string) {
	if a.bus != nil {
		a.bus.Publish(Event{Type: eventType, Data: fmt.Sprintf(`{"message":%q}`, msg)})
	}
}

func (a *Agent) handleSetupStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{
		"needs_setup":  bootstrap.NeedsSetup(),
		"python_ready": bootstrap.PythonReady(),
	})
}

func (a *Agent) handleSetupComplete(w http.ResponseWriter, _ *http.Request) {
	if err := bootstrap.MarkSetupComplete(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "complete"})
}

func (a *Agent) handleSetupBootstrap(w http.ResponseWriter, _ *http.Request) {
	go func() {
		err := bootstrap.Ensure(func(msg string) {
			a.publishProgress("setup_progress", msg)
		})
		if err != nil {
			a.publishProgress("setup_error", err.Error())
			return
		}
		a.publishProgress("setup_done", "ready")
	}()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}
