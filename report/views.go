package report

import (
	"encoding/json"

	"github.com/swissinfo-ch/lstn/ev"
)

// Views implements the Report interface
// It generates a json representation of the views per content id
type Views struct {
	Filter        func(*ev.Ev) bool // filter function
	Cutoff        int               // minimum number of views to be included in the report
	EstimatedSize int               // estimated size of the map
}

// Generate returns a json representation of the views per content id
func (v *Views) Generate(events <-chan *ev.Ev) ([]byte, error) {
	cidViews := make(map[uint32]uint32, v.EstimatedSize)
	for e := range events {
		if v.Filter(e) {
			cidViews[e.Cid]++
		}
	}
	// remove content ids with less than v.Cutoff views
	for cid, views := range cidViews {
		if views < uint32(v.Cutoff) {
			delete(cidViews, cid)
		}
	}
	return json.Marshal(cidViews)
}
