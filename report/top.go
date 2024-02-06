package report

import (
	"encoding/json"
	"sort"

	"github.com/swissinfo-ch/lstn/ev"
)

// Assuming ev is a package that defines Ev and other related types

type Top struct {
	Filter func(*ev.Ev) bool // filter function
	N      int               // number of top content ids to return
}

func (t *Top) Generate(events <-chan *ev.Ev) ([]byte, error) {
	cidViews := make(map[uint32]uint32)
	for e := range events {
		if t.Filter(e) {
			cidViews[e.Cid]++
		}
	}

	// Convert map to slice of kv pairs to sort
	type kv struct {
		Cid   uint32
		Views uint32
	}
	var sortedViews []kv
	for cid, views := range cidViews {
		sortedViews = append(sortedViews, kv{Cid: cid, Views: views})
	}

	// Sort by views in descending order
	sort.Slice(sortedViews, func(i, j int) bool {
		return sortedViews[i].Views > sortedViews[j].Views
	})

	// Select top N content ids
	topN := make(map[uint32]uint32)
	for i, kv := range sortedViews {
		if i >= t.N {
			break
		}
		topN[kv.Cid] = kv.Views
	}

	return json.Marshal(topN)
}
