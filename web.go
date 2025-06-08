//go:generate sh -c "cd web && pnpm install && npm run build"
package proxy

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed web/build/client/*
var webFs embed.FS

// webHandler returns an http.Handler for serving web requests
func webHandler() http.Handler {
	staticFS, _ := fs.Sub(webFs, "web/build/client")
	fileServer := http.FileServer(http.FS(staticFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath := r.URL.Path

		// For paths without file extensions (likely SPA routes)
		if path.Ext(reqPath) == "" && reqPath != "/" {
			trimmedPath := strings.TrimPrefix(reqPath, "/")

			// Check if the file exists in the static filesystem
			if _, err := staticFS.Open(trimmedPath); err != nil {
				// File doesn't exist - serve index.html for client-side routing
				r.URL.Path = "/"
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Serve the actual file
		fileServer.ServeHTTP(w, r)
	})
}
