//go:build !windows

package child

import (
	"testing"
	"time"
)

func TestRunKillsTheProcessGroup(t *testing.T) {
	r := Run(Spec{Argv: []string{"sh", "-c", "sleep 60 & sleep 60"}, Timeout: 300 * time.Millisecond})
	if !r.TimedOut {
		t.Fatalf("%+v", r)
	}
}
