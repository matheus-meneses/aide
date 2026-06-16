package webui

import (
	"fmt"
	"io/fs"
	"net/http"
)

func registerStatic(mux *http.ServeMux) {
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<h1>aide</h1><p>Frontend not built. Run: cd cli/internal/ui/webui/frontend && npm run build</p>")
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
