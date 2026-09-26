package benchsearch

import (
	"fmt"
	"time"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

// Operation is one step of the search chain whose latency is measured.
type Operation struct {
	Name string
	Call func() error
}

// MeasureLatency times every operation once cold and repeat times warm.
// The cold run stands apart: folded into the median it would hide both.
func MeasureLatency(ops []Operation, repeat int, clock func() time.Time) ([]benchreport.Timing, error) {
	if repeat < 1 {
		return nil, fmt.Errorf("repeat must be at least 1, got %d", repeat)
	}
	timings := make([]benchreport.Timing, 0, len(ops))
	for _, op := range ops {
		cold, err := once(op, clock)
		if err != nil {
			return nil, err
		}
		warm := make([]float64, 0, repeat)
		for range repeat {
			ms, err := once(op, clock)
			if err != nil {
				return nil, err
			}
			warm = append(warm, ms)
		}
		timings = append(timings, benchreport.Summarize(op.Name, cold, warm))
	}
	return timings, nil
}

// once times a single call in milliseconds.
func once(op Operation, clock func() time.Time) (float64, error) {
	started := clock()
	if err := op.Call(); err != nil {
		return 0, err
	}
	return benchreport.MS(clock().Sub(started)), nil
}
