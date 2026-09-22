package apply

import (
	"errors"
	"slices"
	"testing"
)

// A caller reads what an abort left behind through one interface, whatever
// kind stopped the run: the three kinds embed ApplyError by value, so
// errors.As on *ApplyError misses them, while the promoted method reaches
// all four.
func TestDirtyFilesReachesEveryKind(t *testing.T) {
	dirty := []string{"95 Prüfzentrum/x/case.toml", "90 Wiki/audit.md"}
	for _, err := range []error{
		&ApplyError{Msg: "plain", Dirty: dirty},
		&TargetMoved{ApplyError{Msg: "target", Dirty: dirty}},
		&SourceMoved{ApplyError{Msg: "source", Dirty: dirty}},
		&ProposalRefused{ApplyError{Msg: "refused", Dirty: dirty}},
	} {
		var d interface{ DirtyFiles() []string }
		if !errors.As(err, &d) || !slices.Equal(d.DirtyFiles(), dirty) {
			t.Fatalf("%v: DirtyFiles not reached", err)
		}
	}
}
