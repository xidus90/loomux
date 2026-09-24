package serve

import (
	"context"
	"time"
)

// The pace of the self-update: a minute after start, so the pass does not
// compete with the first requests, and daily after that.
const (
	UpdateFirst    = time.Minute
	UpdateInterval = 24 * time.Hour
)

// UpdateLoop runs update once first has passed and then every interval, until
// ctx ends. It is not part of the upkeep: the upkeep holds the first answer
// until its pass is done, and an update must hold none.
func UpdateLoop(ctx context.Context, update func(context.Context), first, interval time.Duration, after func(time.Duration) <-chan time.Time) {
	wait := first
	for {
		select {
		case <-ctx.Done():
		case <-after(wait):
		}
		// A select with both ready picks at random; a clock that fires as the
		// service stops must not start a pass the stop has already ended.
		if ctx.Err() != nil {
			return
		}
		update(ctx)
		wait = interval
	}
}
