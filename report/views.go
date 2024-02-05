package report

import (
	"encoding/json"

	"github.com/swissinfo-ch/lstn/ev"
)

// Views is a report that counts the number of views per content id
type Views struct {
	Filter func(*ev.Ev) bool
}

func (v *Views) Generate(events <-chan *ev.Ev) ([]byte, error) {
	cidViews := make(map[uint32]uint32)
	for e := range events {
		if v.Filter(e) {
			_, exists := cidViews[e.Cid]
			if !exists {
				cidViews[e.Cid] = 1
			} else {
				cidViews[e.Cid]++
			}
		}
	}
	return json.Marshal(cidViews)
}
