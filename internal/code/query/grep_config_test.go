package query

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/config"
)

// writeGrepLimit sets [grep] max_filesize in the project's manifest.
func writeGrepLimit(t *testing.T, root string, limit int64) {
	t.Helper()
	path := config.ManifestPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, fmt.Appendf(nil, "[grep]\nmax_filesize = %d\n", limit), 0o644); err != nil {
		t.Fatal(err)
	}
}

// wantSettings asserts the record carries the limit and the ignore case of
// the record itself.
func wantSettings(t *testing.T, root string, limit int64) {
	t.Helper()
	rec, ok := corpus.Read(root)
	if !ok {
		t.Fatal("no corpus record")
	}
	if want := corpus.CanonicalSettings(limit, rec.IgnoreCase); !bytes.Equal(rec.Settings, want) {
		t.Fatalf("settings = %q, want the limit %d", rec.Settings, limit)
	}
}

func TestConfigLimitSetsTheBuild(t *testing.T) {
	root := grepFixture(t)
	writeGrepLimit(t, root, 16)
	if _, _, err := BuildWith(root, BuildOptions{}, nil); err != nil {
		t.Fatal(err)
	}
	wantSettings(t, root, 16)
}

func TestFlagBeatsConfigOnBuild(t *testing.T) {
	root := grepFixture(t)
	writeGrepLimit(t, root, 16)
	unlimited := int64(0)
	if _, _, err := BuildWith(root, BuildOptions{MaxFilesize: &unlimited}, nil); err != nil {
		t.Fatal(err)
	}
	wantSettings(t, root, 0)
}

func TestConfigChangeRebuilds(t *testing.T) {
	root := grepFixture(t)
	writeGrepLimit(t, root, 16)
	if _, _, err := BuildWith(root, BuildOptions{}, nil); err != nil {
		t.Fatal(err)
	}
	writeGrepLimit(t, root, 0)
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "used" {
		t.Fatalf("index = %q, want used after the rebuild for the new limit", res.Index)
	}
	wantSettings(t, root, 0)
}
