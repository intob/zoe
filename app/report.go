package app

import "net/http"

// handleGetReport is the HTTP handler for the /r endpoint.
func (a *App) handleGetReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "missing name query parameter", http.StatusBadRequest)
		return
	}
	report, exists := a.reportRunner.Results(name)
	if !exists {
		http.Error(w, "report not found", http.StatusNotFound)
		return
	}
	w.Write(report)
}
