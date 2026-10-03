package privacy_test

import (
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/config"
)

// nestedWorld registers the shape found on this machine on 2026-09-29: the
// open area "hub", whose tree is its own wiki, holds the wiki of the
// read-only `local_only` area "project/inner", whose source tree lies
// elsewhere. It answers the registry and legacy directories and the three
// trees, spelt as the registry spells them.
func nestedWorld(t *testing.T) (registryDir, legacyDir, hub, innerPath, innerWiki string) {
	t.Helper()
	root := t.TempDir()
	registryDir = filepath.Join(root, "state")
	legacyDir = filepath.Join(root, "legacy")
	hub = filepath.ToSlash(filepath.Join(root, "vault", "hub"))
	innerWiki = hub + "/inner"
	innerPath = filepath.ToSlash(filepath.Join(root, "inner-src"))
	writeFile(t, filepath.Join(hub, ".loomux", "config.toml"), "[area]\nscope = \"hub\"\n\n[privacy]\nmode = \"manual_cloud\"\n")
	writeFile(t, filepath.Join(innerWiki, "page.md"), "# Page\n")
	writeFile(t, filepath.Join(innerPath, "source.md"), "# Source\n")
	inner := config.Area{Scope: "project/inner", Path: innerPath, ReadOnly: true}
	writeFile(t, filepath.Join(config.ManifestDir(inner, legacyDir), ".loomux", "config.toml"),
		"[area]\nscope = \"project/inner\"\n\n[privacy]\nmode = \"local_only\"\n")
	writeFile(t, filepath.Join(registryDir, "registry.toml"),
		"[[area]]\nscope = \"project/inner\"\npath = \""+innerPath+"\"\nwiki = \""+innerWiki+"\"\nreadonly = true\n\n"+
			"[[area]]\nscope = \"hub\"\npath = \""+hub+"\"\nwiki = \""+hub+"\"\n")
	return registryDir, legacyDir, hub, innerPath, innerWiki
}

// A visible area carries the trees of every hidden area, source tree and
// wiki both, so that a path reached through an enclosing area can be asked
// about. The local channel hides nothing and carries nothing.
func TestVisibleAreasCarryTheTreesOfTheHiddenAreas(t *testing.T) {
	registryDir, legacyDir, _, innerPath, innerWiki := nestedWorld(t)
	cloud, err := privacy.VisibleAreas(registryDir, legacyDir, "hub", privacy.ChannelCloud)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{innerPath, innerWiki}; !reflect.DeepEqual(cloud[0].Hidden, want) {
		t.Errorf("cloud: Hidden = %v, want %v", cloud[0].Hidden, want)
	}
	local, err := privacy.VisibleAreas(registryDir, legacyDir, "all", privacy.ChannelLocal)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range local {
		if len(entry.Hidden) != 0 {
			t.Errorf("local: %s carries %v", entry.Area.Scope, entry.Hidden)
		}
	}
}

// A hidden area without a wiki hides its source tree alone; the empty wiki
// value names no tree and must not become the working directory.
func TestAHiddenAreaWithoutAWikiHidesItsSourceTree(t *testing.T) {
	registryDir, legacyDir := twoAreas(t)
	cloud, err := privacy.VisibleAreas(registryDir, legacyDir, "all", privacy.ChannelCloud)
	if err != nil {
		t.Fatal(err)
	}
	if len(cloud[0].Hidden) != 1 || !strings.HasSuffix(cloud[0].Hidden[0], "/repo-1") {
		t.Errorf("Hidden = %v, want the source tree of project/alpha alone", cloud[0].Hidden)
	}
}

func TestConceals(t *testing.T) {
	registryDir, legacyDir, _, _, _ := nestedWorld(t)
	cloud, err := privacy.VisibleAreas(registryDir, legacyDir, "hub", privacy.ChannelCloud)
	if err != nil {
		t.Fatal(err)
	}
	local, err := privacy.VisibleAreas(registryDir, legacyDir, "hub", privacy.ChannelLocal)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		relative string
		cloud    bool
	}{
		{"inner/page.md", true},
		{"inner/missing.md", true},
		{"inner", true},
		{"inner/sub/deeper.md", true},
		{"top.md", false},
		{".", false},
		{"inner-neu/page.md", false},
	} {
		if got := cloud[0].Conceals(tc.relative); got != tc.cloud {
			t.Errorf("cloud: Conceals(%q) = %v, want %v", tc.relative, got, tc.cloud)
		}
		if local[0].Conceals(tc.relative) {
			t.Errorf("local: Conceals(%q) = true", tc.relative)
		}
	}
}

// Within decides on the file system where the root exists: another spelling
// of the same directory is inside, a sibling that only shares a prefix is not.
func TestWithinAnExistingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "space")
	writeFile(t, filepath.Join(root, "a.md"), "")
	writeFile(t, filepath.Join(root+"-neu", "a.md"), "")
	for _, tc := range []struct {
		path string
		want bool
	}{
		{root, true},
		{filepath.Join(root, "a.md"), true},
		{filepath.Join(root, "absent", "b.md"), true},
		{root + string(filepath.Separator), true},
		{filepath.Join(root+"-neu", "a.md"), false},
		{filepath.Dir(root), false},
		{filepath.VolumeName(root) + string(filepath.Separator), false},
	} {
		if got := privacy.Within(root, tc.path); got != tc.want {
			t.Errorf("Within(%q, %q) = %v, want %v", root, tc.path, got, tc.want)
		}
	}
}

// Where the file system folds case, a path spelt in another case lies in the
// root all the same. Only Windows is asked: the test needs a folding file
// system, and that is the one this project runs on.
func TestWithinFollowsTheFileSystemNotTheSpelling(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("needs a file system that folds case")
	}
	root := filepath.Join(t.TempDir(), "space")
	writeFile(t, filepath.Join(root, "a.md"), "")
	if !privacy.Within(root, strings.ToUpper(filepath.Join(root, "a.md"))) {
		t.Error("an upper-case spelling of a file in the root is not within it")
	}
	link := filepath.Join(t.TempDir(), "link")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, root).CombinedOutput(); err != nil {
		t.Fatalf("mklink: %v: %s", err, out)
	}
	if !privacy.Within(root, filepath.Join(link, "a.md")) {
		t.Error("a file reached through a junction to the root is not within it")
	}
}

// A root that is not there is compared by its cleaned spelling: nothing
// under it exists either, so identity has nothing to go on.
func TestWithinAnAbsentRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "gone")
	for _, tc := range []struct {
		path string
		want bool
	}{
		{root, true},
		{filepath.Join(root, "x", "y.md"), true},
		{filepath.Join(root, "x", "..", "..", "other"), false},
		{root + "-neu", false},
		{"relative/path", false},
	} {
		if got := privacy.Within(root, tc.path); got != tc.want {
			t.Errorf("Within(%q, %q) = %v, want %v", root, tc.path, got, tc.want)
		}
	}
}
