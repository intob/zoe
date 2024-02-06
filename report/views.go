package report

import (
	"encoding/json"

	"github.com/swissinfo-ch/lstn/ev"
)

// Views is a report that counts the number of views per content id
type Views struct {
	Filter func(*ev.Ev) bool // filter function
	Cutoff int               // minimum number of views to be included in the report
}

func (v *Views) Generate(events <-chan *ev.Ev) ([]byte, error) {
	cidViews := make(map[uint32]uint32)
	for e := range events {
		if v.Filter(e) {
			cidViews[e.Cid]++
		}
	}
	// apply cutoff
	for cid, views := range cidViews {
		if views < uint32(v.Cutoff) {
			delete(cidViews, cid)
		}
	}
	return json.Marshal(cidViews)
}
