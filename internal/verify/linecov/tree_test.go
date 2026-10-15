package linecov

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestScanTreeSkipsGitAndOtherCheckouts(t *testing.T) {
	root := writeTree(t, map[string]string{
		"core/a.gd":                     "",
		".git/config":                   "",
		".claude/worktrees/w/.git":      "gitdir: elsewhere",
		".claude/worktrees/w/core/a.gd": "",
		"vendor/lib/.git/HEAD":          "",
		"vendor/lib/x.c":                "",
		"z.txt":                         "",
	})
	tree, err := ScanTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"core/a.gd", "z.txt"}; !reflect.DeepEqual(tree.Files, want) {
		t.Fatalf("%q", tree.Files)
	}
	if !tree.InNested(".claude/worktrees/w/core/a.gd") || !tree.InNested("vendor/lib") || tree.InNested("vendor/libx/a.c") {
		t.Fatal("nested checkouts not recognised")
	}
	if !tree.Has("core/a.gd") || tree.Has("core/b.gd") {
		t.Fatal("Has")
	}
}

// A root reached through a link is walked where the link points, and keeps
// the spelling it was given.
func TestScanTreeWalksARootThatIsALink(t *testing.T) {
	target := writeTree(t, map[string]string{"core/a.gd": ""})
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(target, link); err != nil {
		// Windows refuses symlinks without a privilege; a junction needs none.
		if runtime.GOOS != "windows" || exec.Command("cmd", "/c", "mklink", "/J", link, target).Run() != nil {
			t.Skip("no links here:", err)
		}
	}
	tree, err := ScanTree(link)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"core/a.gd"}; !reflect.DeepEqual(tree.Files, want) || tree.Root != link {
		t.Fatalf("%q %s", tree.Files, tree.Root)
	}
}

func TestScanTreeFailsOnAMissingRoot(t *testing.T) {
	if _, err := ScanTree(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("no error")
	}
}

func TestResolveMapsEveryPathFormToTheRoot(t *testing.T) {
	root := writeTree(t, map[string]string{
		"core/a.gd": "", "src/app.py": "", "src/main/java/com/x/Calc.java": "",
		"a/com/y/D.java": "", "b/com/y/D.java": "",
		"sub/.git": "", "sub/s.c": "",
	})
	tree, err := ScanTree(root)
	if err != nil {
		t.Fatal(err)
	}
	abs := filepath.ToSlash(tree.Root)
	for _, row := range []struct {
		rec  Record
		want Resolution
	}{
		{Record{Path: "res://core/a.gd"}, Resolution{Rel: "core/a.gd", Outcome: Resolved}},
		{Record{Path: `core\a.gd`}, Resolution{Rel: "core/a.gd", Outcome: Resolved}},
		{Record{Path: abs + "/core/a.gd"}, Resolution{Rel: "core/a.gd", Outcome: Resolved}},
		{Record{Path: "/usr/include/stdio.h"}, Resolution{Rel: "/usr/include/stdio.h", Outcome: Outside}},
		{Record{Path: "../elsewhere/x.c"}, Resolution{Rel: "../elsewhere/x.c", Outcome: Outside}},
		{Record{Path: "app.py", Bases: []string{"lib", "src"}}, Resolution{Rel: "src/app.py", Outcome: Resolved}},
		{Record{Path: "app.py", Bases: []string{abs + "/src"}}, Resolution{Rel: "src/app.py", Outcome: Resolved}},
		{Record{Path: "gone.py", Bases: []string{"src"}}, Resolution{Rel: "gone.py", Outcome: Missing}},
		{Record{Path: "com/x/Calc.java", Suffix: true}, Resolution{Rel: "src/main/java/com/x/Calc.java", Outcome: Resolved}},
		{Record{Path: "com/x/Gone.java", Suffix: true}, Resolution{Rel: "com/x/Gone.java", Outcome: Missing}},
		{Record{Path: "com/y/D.java", Suffix: true}, Resolution{Rel: "com/y/D.java", Outcome: Ambiguous, Candidates: []string{"a/com/y/D.java", "b/com/y/D.java"}}},
		{Record{Path: "sub/s.c"}, Resolution{Rel: "sub/s.c", Outcome: Nested}},
	} {
		if got := tree.Resolve(row.rec); !reflect.DeepEqual(got, row.want) {
			t.Errorf("%+v: %+v", row.rec, got)
		}
	}
}

// A report spelling the path in another case finds no file and says so; only
// the root prefix of an absolute path is compared without case on Windows.
func TestResolveComparesTheRootPrefixWithoutCaseOnlyWhereTheSystemDoes(t *testing.T) {
	root := writeTree(t, map[string]string{"core/a.gd": ""})
	tree, err := ScanTree(root)
	if err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(filepath.ToSlash(tree.Root)) + "/core/a.gd"
	old := foldCase
	t.Cleanup(func() { foldCase = old })
	foldCase = true
	if got := tree.Resolve(Record{Path: upper}); got.Outcome != Resolved {
		t.Fatalf("fold: %+v", got)
	}
	foldCase = false
	if got := tree.Resolve(Record{Path: upper}); got.Outcome != Outside {
		t.Fatalf("no fold: %+v", got)
	}
	if got := tree.Resolve(Record{Path: "Core/A.gd"}); got.Outcome != Missing {
		t.Fatalf("case of the relative part: %+v", got)
	}
}

// A root at the top of the file system already ends in a separator.
func TestResolveTreatsARootAtTheTopOfTheFileSystem(t *testing.T) {
	root, p := "/", "/x/y.c"
	if runtime.GOOS == "windows" {
		root, p = `C:\`, "C:/x/y.c"
	}
	tree := &Tree{Root: root, has: map[string]bool{"x/y.c": true}}
	if got := tree.Resolve(Record{Path: p}); got.Outcome != Resolved || got.Rel != "x/y.c" {
		t.Fatalf("%+v", got)
	}
}

func TestFoldCaseDefaultsToTheSystemsBehaviour(t *testing.T) {
	if foldCase != (runtime.GOOS == "windows") {
		t.Fatal(foldCase)
	}
}

func TestScanTreeKeepsRootFilesAndLeavesOutAGitFileAndSymlinks(t *testing.T) {
	root := writeTree(t, map[string]string{".git": "gitdir: x", "x.c": "", "d/y.c": ""})
	if err := os.Symlink(filepath.Join(root, "x.c"), filepath.Join(root, "link.c")); err != nil {
		t.Log("no symlinks here:", err)
	}
	tree, err := ScanTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"d/y.c", "x.c"}; !reflect.DeepEqual(tree.Files, want) {
		t.Fatalf("%q", tree.Files)
	}
	if got := tree.Resolve(Record{Path: "x.c", Suffix: true}); got.Outcome != Resolved || got.Rel != "x.c" {
		t.Fatalf("%+v", got)
	}
}

func TestResolveSeesTheBareRootAndExactCaseWithoutFolding(t *testing.T) {
	tree, err := ScanTree(writeTree(t, map[string]string{"core/a.gd": ""}))
	if err != nil {
		t.Fatal(err)
	}
	abs := filepath.ToSlash(tree.Root)
	if got := tree.Resolve(Record{Path: "C:/"}); got.Outcome != Outside {
		t.Fatalf("bare drive root: %+v", got)
	}
	old := foldCase
	t.Cleanup(func() { foldCase = old })
	foldCase = false
	if got := tree.Resolve(Record{Path: abs + "/core/a.gd"}); got.Outcome != Resolved {
		t.Fatalf("exact case: %+v", got)
	}
}

func TestScanTreeFailsWhenTheRootCannotBeMadeAbsolute(t *testing.T) {
	old := absPath
	t.Cleanup(func() { absPath = old })
	absPath = func(string) (string, error) { return "", os.ErrNotExist }
	if _, err := ScanTree("."); err == nil {
		t.Fatal("no error")
	}
}
