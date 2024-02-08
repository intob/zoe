package app

import (
	"encoding/json"
	"net/http"
)

// Stat is a JSON-serializable struct for the /stat endpoint.
type Stat struct {
	FileSize                int64  `json:"fileSize"`                // in bytes
	CurrentReportEventCount uint32 `json:"currentReportEventCount"` // number of events in the current report so far
	LastReportEventCount    uint32 `json:"lastReportEventCount"`    // number of events in the last report
	LastReportDuration      string `json:"lastReportDuration"`      // duration of the last report
	LastReportTime          int64  `json:"lastReportTime"`          // Unix timestamp of the last report
}

// handleGetStat is the HTTP handler for the /stat endpoint.
func (a *App) handleGetStat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	s := &Stat{
		FileSize:                a.reportRunner.FileSize(),
		CurrentReportEventCount: a.reportRunner.CurrentReportEventCount(),
		LastReportEventCount:    a.reportRunner.LastReportEventCount(),
		LastReportDuration:      a.reportRunner.LastReportDuration().String(),
		LastReportTime:          a.reportRunner.LastReportTime().Unix(),
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		panic(err)
	}
	w.Write(data)
}
