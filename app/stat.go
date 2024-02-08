package app

import (
	"encoding/json"
	"net/http"
)

func (a *App) handleGetStat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=60")
	type stat struct {
		FileSize                int64  `json:"fileSize"`
		CurrentReportEventCount uint32 `json:"currentReportEventCount"`
		LastReportEventCount    uint32 `json:"lastReportEventCount"`
		LastReportDuration      string `json:"lastReportDuration"`
		LastReportTime          int32  `json:"lastReportTime"`
	}
	s := &stat{
		FileSize:                a.reportRunner.FileSize(),
		CurrentReportEventCount: a.reportRunner.CurrentReportEventCount(),
		LastReportEventCount:    a.reportRunner.LastReportEventCount(),
		LastReportDuration:      a.reportRunner.LastReportDuration().String(),
		LastReportTime:          int32(a.reportRunner.LastReportTime().Unix()),
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		panic(err)
	}
	w.Write(data)
}
