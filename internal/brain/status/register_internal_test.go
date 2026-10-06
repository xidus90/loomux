package status

import (
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
)

// A register that cannot be inspected aborts the status. Read as empty it
// dropped the area from the findings without a word: no document the engine
// misses, no content held under several paths.
func TestARegisterThatCannotBeInspectedAbortsTheRegisterFindings(t *testing.T) {
	visible := privacy.VisibleArea{
		Area:     config.Area{Scope: "project/odd", Path: filepath.Join(t.TempDir(), "odd"), ReadOnly: true},
		Manifest: &config.Manifest{},
	}
	stateDir := filepath.Join(t.TempDir(), "a\x00b")
	if lines, err := unfindable(visible, search.NewFakePort(), stateDir); err == nil || lines != nil {
		t.Fatalf("unfindable: got %q, %v", lines, err)
	}
	if lines, err := sharedHashes([]privacy.VisibleArea{visible}, stateDir); err == nil || lines != nil {
		t.Fatalf("sharedHashes: got %q, %v", lines, err)
	}
}
