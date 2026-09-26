package benchsearch

import (
	"errors"
	"testing"
	"time"
)

func TestMeasureLatencyRunsColdThenRepeatWarm(t *testing.T) {
	calls := 0
	ops := []Operation{{Name: "keyword", Call: func() error { calls++; return nil }}}
	got, err := MeasureLatency(ops, 3, tick(2*time.Millisecond))
	if err != nil || calls != 4 || got[0].ColdMS != 2 || len(got[0].WarmMS) != 3 {
		t.Fatalf("got %+v calls=%d err=%v", got, calls, err)
	}
	if got[0].Name != "keyword" || got[0].MedianMS != 2 {
		t.Fatalf("got %+v", got)
	}
}

func TestMeasureLatencyRefusesRepeatBelowOne(t *testing.T) {
	if _, err := MeasureLatency(nil, 0, tick(0)); err == nil || err.Error() != "repeat must be at least 1, got 0" {
		t.Fatalf("err = %v", err)
	}
}

func TestMeasureLatencyStopsAtAFailingOperation(t *testing.T) {
	boom := errors.New("read failed")
	// The cold run and a warm run fail on different calls, so both places
	// that can end the measurement are held.
	for _, failAt := range []int{1, 2} {
		calls := 0
		ops := []Operation{
			{Name: "read", Call: func() error {
				calls++
				if calls == failAt {
					return boom
				}
				return nil
			}},
			{Name: "never", Call: func() error { t.Fatal("ran after a failure"); return nil }},
		}
		got, err := MeasureLatency(ops, 3, tick(time.Millisecond))
		if !errors.Is(err, boom) || got != nil || calls != failAt {
			t.Fatalf("failAt %d: got %+v calls=%d err=%v", failAt, got, calls, err)
		}
	}
}
