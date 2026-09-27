package load

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestAnOverlayTakesOnlyItsOwnFilesFromTheTop(t *testing.T) {
	overlay := overlayFS{
		base: fstest.MapFS{
			"flow.toml":             {Data: []byte("base flow")},
			"instructions/draft.md": {Data: []byte("base draft")},
			"questions/approve.md":  {Data: []byte("base question")},
		},
		top: fstest.MapFS{
			"questions/approve.md":  {Data: []byte("top question")},
			"instructions/draft.md": {Data: []byte("not listed")},
		},
		files: []string{"questions/approve.md"},
	}
	for name, want := range map[string]string{
		"questions/approve.md":  "top question",
		"instructions/draft.md": "base draft",
		"flow.toml":             "base flow",
	} {
		raw, err := fs.ReadFile(overlay, name)
		if err != nil || string(raw) != want {
			t.Fatalf("%s = %q, %v; want %q", name, raw, err, want)
		}
	}
	if _, err := fs.ReadFile(overlay, "questions/nope.md"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v", err)
	}
}
