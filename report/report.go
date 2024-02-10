package report

import (
	"time"

	"github.com/swissinfo-ch/zoe/ev"
)

type Report interface {
	Generate(<-chan *ev.Ev) ([]byte, error)
}

func YoungerThan(e *ev.Ev, d time.Duration) bool {
	return e.Time > uint32(time.Now().Add(-d).Unix())
}
