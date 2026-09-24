//go:build windows

package query

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// shortName is the 8.3 alias the file system keeps for path, and the test is
// skipped where this volume keeps none. A CI runner's %TEMP% is such an alias
// (C:\Users\RUNNER~1\...), while git answers the long name for the same
// directory, so a hook's index spelt one way must be accepted against the
// git directory spelt the other.
func shortName(t *testing.T, path string) string {
	t.Helper()
	wide, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]uint16, windows.MAX_PATH)
	length, err := windows.GetShortPathName(wide, &buffer[0], uint32(len(buffer)))
	if err != nil || length == 0 || int(length) > len(buffer) {
		t.Skipf("no short name for %q: %v", path, err)
	}
	short := windows.UTF16ToString(buffer[:length])
	if strings.EqualFold(short, path) {
		t.Skipf("this volume keeps no 8.3 alias for %q", path)
	}
	return short
}

// A drive-relative path is one the resolver refuses to place; resolved keeps
// it as spelt, so the comparison it feeds cannot accept it by accident.
func TestResolvedKeepsWhatTheResolverRefuses(t *testing.T) {
	if got := resolved(`C:rel\..\index`); got != `C:index` {
		t.Fatalf("got %q", got)
	}
}

func TestIndexFileForAcceptsItsIndexSpeltWithAShortName(t *testing.T) {
	long := filepath.Join(t.TempDir(), "a directory with a long name")
	if err := os.Mkdir(long, 0o755); err != nil {
		t.Fatal(err)
	}
	short := shortName(t, long)
	git(t, long, "init", "-q")
	if err := os.WriteFile(filepath.Join(long, ".git", "index"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for name, index := range map[string]string{
		"an index that exists":         filepath.Join(short, ".git", "index"),
		"a hook's index not yet there": filepath.Join(short, ".git", "next-index-1.lock"),
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("GIT_INDEX_FILE", index)
			if got, ok := indexFileFor(long); !ok || got != index {
				t.Fatalf("%q %v, want the index as git handed it", got, ok)
			}
		})
	}
}
