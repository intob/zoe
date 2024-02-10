package report

import (
	"encoding/json"

	"github.com/swissinfo-ch/zoe/ev"
)

// Subset is a report that returns a subset of events
// based on a filter and a limit.
type Subset struct {
	Filter func(*ev.Ev) bool // filter function
	Limit  int               // maximum number of events to be included in the report
}

// Generate returns a json representation of the subset of events
func (s *Subset) Generate(events <-chan *ev.Ev) (*Result, error) {
	raw := make([]*ev.Ev, 0, s.Limit)

	// get the subset of events
	for e := range events {
		if s.Filter(e) {
			raw = append(raw, e)
			if len(raw) >= s.Limit {
				break
			}
		}
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	return &Result{
		Content:     data,
		ContentType: "application/json",
	}, nil
}
