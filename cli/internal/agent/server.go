package agent

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var Version = "dev"

func (a *Agent) Serve(ctx context.Context, port int) error {
	a.bus = NewEventBus()
	a.SetNotifier(&BusNotifier{Bus: a.bus})
	a.sessions.startJanitor(ctx)

	mux := http.NewServeMux()
	a.registerRoutes(mux)

	handler := corsMiddleware(mux)

	server := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() { //nolint:gosec // G118: this goroutine handles graceful shutdown and intentionally creates a fresh timeout context
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("agent server shutdown: %v", err)
		}
	}()

	url := fmt.Sprintf("http://localhost:%d", port)
	log.Printf("Agent web UI available at %s", url)
	go openBrowser(url)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Vary", "Origin")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		if focusExistingChromeTab(url) {
			return
		}
		exec.Command("open", url).Start() //nolint:errcheck // fire-and-forget browser open
	case "linux":
		exec.Command("xdg-open", url).Start() //nolint:errcheck // fire-and-forget browser open
	}
}

func focusExistingChromeTab(url string) bool {
	match := strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
	script := fmt.Sprintf(`tell application "System Events"
	if not (exists process "Google Chrome") then return "notrunning"
end tell
tell application "Google Chrome"
	repeat with theWin in windows
		set tabCount to count of tabs of theWin
		repeat with j from 1 to tabCount
			if (URL of tab j of theWin) contains "%s" then
				set active tab index of theWin to j
				set index of theWin to 1
				activate
				return "found"
			end if
		end repeat
	end repeat
end tell
return "notfound"`, match)

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "found"
}
