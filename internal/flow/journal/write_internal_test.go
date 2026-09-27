package journal

import (
	"errors"
	"testing"
)

// A disk that refuses a write to an open file cannot be arranged from a test,
// so the wrapping is checked on its own.
func TestClosedWrapsAFailedWrite(t *testing.T) {
	if err := closed("run.jsonl", nil); err != nil {
		t.Fatalf("no failure, no error: %v", err)
	}
	err := closed("run.jsonl", errors.New("disk full"))
	if err == nil || err.Error() != "writing run.jsonl: disk full" {
		t.Fatalf("got %v", err)
	}
}
