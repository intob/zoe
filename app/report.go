package app

import "net/http"

func (a *App) handleGetReport(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=60")
	w.Write(report)
}
