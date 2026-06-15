package webui

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"aide/cli/internal/clog"
)

var wlog = clog.New("webui")

type Options struct {
	Port        int
	NoBrowser   bool
	RegisterAPI func(*http.ServeMux)
}

func Serve(ctx context.Context, opts Options) error {
	mux := http.NewServeMux()
	if opts.RegisterAPI != nil {
		opts.RegisterAPI(mux)
	}
	registerOpen(mux)
	registerLogs(mux)
	registerStatic(mux)

	handler := corsMiddleware(mux)

	server := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", opts.Port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() { //nolint:gosec // G118: this goroutine handles graceful shutdown and intentionally creates a fresh timeout context
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			wlog.Error("web server shutdown: %v", err)
		}
	}()

	url := fmt.Sprintf("http://localhost:%d", opts.Port)
	wlog.Info("web UI available at %s", url)
	if !opts.NoBrowser {
		go openBrowser(url)
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}
