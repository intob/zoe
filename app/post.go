package app

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/swissinfo-ch/lstn/ev"
)

// Handle event input
func (a *App) handlePost(w http.ResponseWriter, r *http.Request) {
	evType, ok := ev.EvType_value[r.Header.Get("X_TYPE")]
	if !ok {
		http.Error(w, "invalid header X_TYPE, must be one of LOAD, UNLOAD or TIME", http.StatusBadRequest)
		return
	}
	usr, err := strconv.ParseUint(r.Header.Get("X_USR"), 10, 32)
	if err != nil {
		http.Error(w, fmt.Errorf("err to parse uint32 in header X_USR: %w", err).Error(), http.StatusBadRequest)
		return
	}
	sess, err := strconv.ParseUint(r.Header.Get("X_SESS"), 10, 32)
	if err != nil {
		http.Error(w, fmt.Errorf("err to parse uint32 in header X_SESS: %w", err).Error(), http.StatusBadRequest)
		return
	}
	cid, err := strconv.ParseUint(r.Header.Get("X_CID"), 10, 32)
	if err != nil {
		http.Error(w, fmt.Errorf("err to parse uint32 in header X_CID: %w", err).Error(), http.StatusBadRequest)
		return
	}
	e := &ev.Ev{
		Time:   uint32(time.Now().Unix()),
		EvType: ev.EvType(evType),
		Usr:    uint32(usr),
		Sess:   uint32(sess),
		Cid:    uint32(cid),
	}
	switch e.EvType {
	case ev.EvType_UNLOAD:
		scrolled, err := strconv.ParseFloat(r.Header.Get("X_SCROLLED"), 32)
		if err != nil {
			http.Error(w, fmt.Errorf("failed to parse X_SCROLLED: %w", err).Error(), http.StatusBadRequest)
			return
		}
		scrolled32 := float32(scrolled)
		e.Scrolled = &scrolled32
	case ev.EvType_TIME:
		pageSeconds, err := strconv.ParseUint(r.Header.Get("X_PAGE_SECONDS"), 10, 32)
		if err != nil {
			http.Error(w, fmt.Errorf("failed to parse X_PAGE_SECONDS: %w", err).Error(), http.StatusBadRequest)
			return
		}
		pageSeconds32 := uint32(pageSeconds)
		e.PageSeconds = &pageSeconds32
	}
	a.events <- e
}
