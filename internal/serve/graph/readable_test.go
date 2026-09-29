package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/config"
)

// The graph tools read files under an area's path by relative path, so a
// hidden tree inside that path has to be kept out of what they may read, as
// the brain tools keep it out: nesting does not lift local_only.
func TestReadableKeepsAHiddenTreeOutOfAnEnclosingArea(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	area := privacy.VisibleArea{Area: config.Area{Scope: "hub", Path: root}, Hidden: []string{inner}}
	keep := readable(area)
	if keep("inner/page.go") {
		t.Error("a file inside the hidden tree is readable")
	}
	if !keep("top.go") {
		t.Error("a file outside the hidden tree is not readable")
	}
	if !readable(privacy.VisibleArea{Area: config.Area{Scope: "hub", Path: root}})("inner/page.go") {
		t.Error("with nothing hidden, the file is not readable")
	}
}
