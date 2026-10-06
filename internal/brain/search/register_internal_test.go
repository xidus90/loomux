package search

import (
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/config"
)

// A register that cannot be inspected fails the search. Read as empty it
// made every hit look unregistered, and the note then advised a reindex that
// cannot help.
func TestAssembleRefusesARegisterThatCannotBeInspected(t *testing.T) {
	area := privacy.VisibleArea{
		Area:     config.Area{Scope: "project/odd", Path: filepath.Join(t.TempDir(), "odd"), ReadOnly: true},
		Manifest: &config.Manifest{},
	}
	stateDir := filepath.Join(t.TempDir(), "a\x00b")
	answer, err := assemble(nil, []privacy.VisibleArea{area}, stateDir, 5)
	if err == nil || answer != nil {
		t.Fatalf("got %+v, %v", answer, err)
	}
}
