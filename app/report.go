package app

import "net/http"

// handleGetReport is the HTTP handler for the /r endpoint.
func (a *App) handleGetReportResult(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "missing name query parameter", http.StatusBadRequest)
		return
	}
	result, exists := a.reportRunner.Result(name)
	if !exists {
		http.Error(w, "report not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", result.ContentType)
	w.Write(result.Content)
}
