package store

import (
	"github.com/davecheney/junk/clock"
	"time"
)

func ServerTime() time.Time {
	return clock.Monotonic.Now()
}
