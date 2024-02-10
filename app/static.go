package app

import (
	"net/http"
)

// handleStatic is the HTTP handler for static files
func (a *App) handleStatic(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=3600")
	http.ServeFile(w, r, "static/"+r.URL.Path)
}
