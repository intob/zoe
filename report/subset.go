package report

import (
	"encoding/json"

	"github.com/swissinfo-ch/lstn/ev"
)

// Subset is a report that returns a subset of events
// based on a filter and a limit.
type Subset struct {
	Filter func(*ev.Ev) bool // filter function
	Limit  int               // maximum number of events to be included in the report
}

// Generate returns a json representation of the subset of events
func (s *Subset) Generate(events <-chan *ev.Ev) ([]byte, error) {
	data := make([]*ev.Ev, 0, s.Limit)
	for e := range events {
		if s.Filter(e) {
			data = append(data, e)
			if len(data) >= s.Limit {
				break
			}
		}
	}
	return json.Marshal(data)
}
