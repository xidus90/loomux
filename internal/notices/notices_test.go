package notices

import (
	"testing"

	devnotices "github.com/xidus90/loomux/internal/dev/notices"
)

// TestTheNoticeIsCurrent fails when a dependency, a grammar or the Zipf
// notice changed and NOTICE.md was not written again: run loomux dev notices.
func TestTheNoticeIsCurrent(t *testing.T) {
	want, err := devnotices.Render("../..", devnotices.GoCommand)
	if err != nil {
		t.Fatal(err)
	}
	if Text() != want {
		t.Fatal("internal/notices/NOTICE.md is stale; run loomux dev notices")
	}
}
