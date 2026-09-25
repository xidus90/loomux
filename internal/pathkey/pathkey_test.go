package pathkey

import (
	"runtime"
	"testing"
)

// Windows and macOS compare paths without regard to case by default; on
// Linux Repo and repo are two directories, and a record of one is no record
// of the other.
func TestKeysFoldCaseWhereTheFileSystemDoes(t *testing.T) {
	for goos, folds := range map[string]bool{"windows": true, "darwin": true, "linux": false, "freebsd": false} {
		if got := KeyOn(goos, "/srv/Repo/.git") == KeyOn(goos, "/srv/repo/.git"); got != folds {
			t.Errorf("%s: Repo and repo equal %v, want %v", goos, got, folds)
		}
		if KeyOn(goos, "/srv/repo/") != KeyOn(goos, "/srv/repo") {
			t.Errorf("%s: a trailing slash set one path apart", goos)
		}
	}
}

func TestSameAndKeyFollowThisPlatform(t *testing.T) {
	if Key("/a/B") != KeyOn(runtime.GOOS, "/a/B") {
		t.Error("Key differs from KeyOn for this platform")
	}
	if !Same("/a/b/", "/a/b") {
		t.Error("Same set a trailing slash apart")
	}
	if Same("/a/b", "/a/c") {
		t.Error("Same took two paths for one")
	}
}
