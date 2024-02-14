package app

import (
	"encoding/json"
	"net/http"

	"github.com/intob/jfmt"
)

// Status is a JSON-serializable struct for the /stat endpoint.
type Status struct {
	FileSize                int64  `json:"fileSize"`                // in bytes
	CurrentReportEventCount uint32 `json:"currentReportEventCount"` // number of events in the current report so far
	LastReportEventCount    uint32 `json:"lastReportEventCount"`    // number of events in the last report
	LastReportDuration      string `json:"lastReportDuration"`      // duration of the last report
	LastReportTime          int64  `json:"lastReportTime"`          // Unix timestamp of the last report
	Commit                  string `json:"commit"`                  // Git commit hash
	NumCPU                  int    `json:"numCPU"`                  // number of CPU cores
}

// handleGetStatus is the HTTP handler for the /stat endpoint.
func (a *App) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	s := &Status{
		FileSize:                a.reportRunner.FileSize(),
		CurrentReportEventCount: a.reportRunner.CurrentReportEventCount(),
		LastReportEventCount:    a.reportRunner.LastReportEventCount(),
		LastReportDuration:      jfmt.FmtDuration(a.reportRunner.LastReportDuration()),
		LastReportTime:          a.reportRunner.LastReportTime().Unix(),
		Commit:                  a.commit,
		NumCPU:                  a.numCPU,
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		panic(err)
	}
	w.Write(data)
}
