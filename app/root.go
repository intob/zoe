package app

import (
	"fmt"
	"net/http"
	"text/template"
)

func (a *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "max-age=3600")

	t, err := template.ParseFiles("assets/index.html")
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Commit      string
		ReportNames []string
	}{
		Commit:      a.commit,
		ReportNames: a.reportNames,
	}

	err = t.Execute(w, data)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
