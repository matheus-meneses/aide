package webui

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
)

func registerOpen(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/open", handleOpenExternal)
}

func handleOpenExternal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	parsed, err := url.Parse(req.URL)
	if err != nil {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}
	switch parsed.Scheme {
	case "http", "https", "mailto":
	default:
		http.Error(w, "unsupported url scheme", http.StatusBadRequest)
		return
	}

	openExternal(req.URL)
	w.WriteHeader(http.StatusNoContent)
}

func openExternal(target string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", target).Start() //nolint:errcheck // fire-and-forget
	case "linux":
		exec.Command("xdg-open", target).Start() //nolint:errcheck // fire-and-forget
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start() //nolint:errcheck // fire-and-forget
	}
}
