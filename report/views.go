package report

import (
	"encoding/json"
	"time"

	"github.com/swissinfo-ch/zoe/ev"
)

// Views implements the Report interface
// It generates a json representation of the views (loads) per content id
type Views struct {
	Cutoff        int              // minimum number of views to be included in the report
	EstimatedSize int              // estimated size of the map
	MinEvTime     func() time.Time // func that returns earliest time for events to be included in the report
}

// Generate returns a json representation of the views per content id
func (v *Views) Generate(events <-chan *ev.Ev) (*Result, error) {
	minEvTime := uint32(v.MinEvTime().Unix())
	cidViews := make(map[uint32]uint32, v.EstimatedSize)

	// range over events and count views per content id
	for e := range events {
		if e.Time < minEvTime {
			// events are ordered by time, so we can break here
			break
		}
		if e.EvType == ev.EvType_LOAD {
			cidViews[e.Cid]++
		}
	}

	// remove content ids with less than v.Cutoff views
	for cid, views := range cidViews {
		if views < uint32(v.Cutoff) {
			delete(cidViews, cid)
		}
	}

	data, err := json.Marshal(cidViews)
	if err != nil {
		return nil, err
	}

	return &Result{
		Content:     data,
		ContentType: "application/json",
	}, nil
}
