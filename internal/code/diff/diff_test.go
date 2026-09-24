package diff

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
)

func TestParseNameStatus(t *testing.T) {
	in := "M\x00a.go\x00R087\x00old name.go\x00neu/ä.go\x00D\x00gone.go\x00A\x00new.go\x00C100\x00x.go\x00y.go\x00T\x00link\x00"
	got, err := ParseNameStatus(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	want := []File{
		{Status: Modified, Path: "a.go"},
		{Status: Renamed, OldPath: "old name.go", Path: "neu/ä.go"},
		{Status: Deleted, Path: "gone.go"},
		{Status: Added, Path: "new.go"},
		{Status: Copied, OldPath: "x.go", Path: "y.go"},
		{Status: TypeChanged, Path: "link"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestParseNameStatusWithoutTrailingNUL(t *testing.T) {
	got, err := ParseNameStatus(strings.NewReader("M\x00a.go"))
	if err != nil || !reflect.DeepEqual(got, []File{{Status: Modified, Path: "a.go"}}) {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestParseNameStatusRejects(t *testing.T) {
	for name, in := range map[string]string{
		"unknown status":   "X\x00a.go\x00",
		"missing path":     "M\x00",
		"cut after status": "M",
		"rename one path":  "R100\x00a.go\x00",
		"empty status":     "\x00a.go\x00",
	} {
		if _, err := ParseNameStatus(strings.NewReader(in)); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

// An unmerged path has no single post-image to read; the error names it.
func TestParseNameStatusRejectsAnUnmergedPath(t *testing.T) {
	_, err := ParseNameStatus(strings.NewReader("M\x00a.go\x00U\x00calc/calc.go\x00"))
	if !errors.Is(err, ErrUnmerged) || err.Error() != "unresolved conflict in calc/calc.go" {
		t.Fatalf("got %v, want ErrUnmerged", err)
	}
}

// An unmerged status without its path is cut short like any other.
func TestParseNameStatusRejectsAnUnmergedStatusWithoutItsPath(t *testing.T) {
	_, err := ParseNameStatus(strings.NewReader("U\x00"))
	if err == nil || errors.Is(err, ErrUnmerged) {
		t.Fatalf("got %v, want the missing path", err)
	}
}

func TestParseNameStatusEmpty(t *testing.T) {
	got, err := ParseNameStatus(strings.NewReader(""))
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestParseNameStatusPassesAReadError(t *testing.T) {
	boom := errors.New("boom")
	if _, err := ParseNameStatus(iotest.ErrReader(boom)); !errors.Is(err, boom) {
		t.Fatalf("got %v, want %v", err, boom)
	}
}

func TestStatusMarshalsAsItsLetter(t *testing.T) {
	b, err := json.Marshal(File{Status: Renamed, Path: "b", OldPath: "a"})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"status":"R","path":"b","old_path":"a"}`; string(b) != want {
		t.Fatalf("got %s, want %s", b, want)
	}
}

func TestApplyHunks(t *testing.T) {
	files := []File{{Status: Modified, Path: "a.go"}, {Status: Deleted, Path: "gone.go"}, {Status: Modified, Path: "my file.go"}, {Status: Modified, Path: "ä.go"}, {Status: Modified, Path: "untouched.go"}}
	patch := strings.Join([]string{
		"diff --git a/a.go b/a.go",
		"index 1..2 100644",
		"--- a/a.go",
		"+++ b/a.go",
		"@@ -3 +3 @@ func A() {",
		"-\told()",
		"+\tnew()",
		"@@ -10,2 +9,0 @@",
		"--- a removed line that looks like a header",
		"-x",
		"@@ -1,0 +1,2 @@",
		"+package a",
		"+",
		"diff --git a/gone.go b/gone.go",
		"deleted file mode 100644",
		"--- a/gone.go",
		"+++ /dev/null",
		"@@ -1,2 +0,0 @@",
		"-package gone",
		"-",
		"diff --git a/my file.go b/my file.go",
		"--- a/my file.go\t",
		"+++ b/my file.go\t",
		"@@ -5 +5 @@",
		"-a",
		"+b",
		`diff --git "a/\303\244.go" "b/\303\244.go"`,
		`--- "a/\303\244.go"`,
		`+++ "b/\303\244.go"`,
		"@@ -2 +2 @@",
		"-a",
		"+b",
		"\\ No newline at end of file",
		"",
	}, "\n")
	if err := ApplyHunks(files, strings.NewReader(patch)); err != nil {
		t.Fatal(err)
	}
	a := files[0].Hunks
	if len(a) != 3 || a[0].From != 3 || a[0].To != 3 || a[1].From != 9 || a[1].To != 9 || a[2].From != 1 || a[2].To != 2 {
		t.Fatalf("a.go hunks %+v", a)
	}
	if !reflect.DeepEqual(a[0].Lines, []string{"-\told()", "+\tnew()"}) {
		t.Fatalf("a.go first hunk lines %q", a[0].Lines)
	}
	if a[1].Lines[0] != "--- a removed line that looks like a header" {
		t.Fatalf("header-like removed line lost: %q", a[1].Lines)
	}
	if g := files[1].Hunks; len(g) != 1 || g[0].From != 1 || g[0].To != 1 {
		t.Fatalf("deleted file hunk %+v", g)
	}
	if len(files[2].Hunks) != 1 || len(files[3].Hunks) != 1 || files[4].Hunks != nil {
		t.Fatalf("space/umlaut/untouched: %+v", files[2:])
	}
	if got := files[3].Hunks[0].Lines; !reflect.DeepEqual(got, []string{"-a", "+b"}) {
		t.Fatalf("no-newline marker counted as a line: %q", got)
	}
}

func TestApplyHunksSkipsFilesNotListed(t *testing.T) {
	files := []File{{Status: Modified, Path: "b.go"}}
	patch := "diff --git a/other.go b/other.go\r\n--- a/other.go\r\n+++ b/other.go\r\n@@ -1 +1 @@\r\n-x\r\n+y\r\n" +
		"diff --git a/b.go b/b.go\r\n--- a/b.go\r\n+++ b/b.go\r\n@@ -4,0 +5 @@\r\n+z"
	if err := ApplyHunks(files, strings.NewReader(patch)); err != nil {
		t.Fatal(err)
	}
	h := files[0].Hunks
	if len(h) != 1 || h[0].From != 5 || h[0].To != 5 || !reflect.DeepEqual(h[0].Lines, []string{"+z"}) {
		t.Fatalf("b.go hunks %+v", h)
	}
}

func TestApplyHunksSkipsANoNewlineMarkerInsideAHunk(t *testing.T) {
	files := []File{{Status: Modified, Path: "a.go"}}
	patch := "--- a/a.go\n+++ b/a.go\n@@ -7 +7 @@\n-old\n\\ No newline at end of file\n+new\n\\ No newline at end of file\n"
	if err := ApplyHunks(files, strings.NewReader(patch)); err != nil {
		t.Fatal(err)
	}
	if h := files[0].Hunks; len(h) != 1 || !reflect.DeepEqual(h[0].Lines, []string{"-old", "+new"}) {
		t.Fatalf("hunks %+v", h)
	}
}

func TestApplyHunksEndsOnAHeaderWithoutNewline(t *testing.T) {
	files := []File{{Status: Added, Path: "empty.go"}}
	if err := ApplyHunks(files, strings.NewReader("--- /dev/null\n+++ b/empty.go")); err != nil {
		t.Fatal(err)
	}
	if files[0].Hunks != nil {
		t.Fatalf("hunks %+v", files[0].Hunks)
	}
}

func TestApplyHunksCaps(t *testing.T) {
	var b strings.Builder
	b.WriteString("--- a/big.go\n+++ b/big.go\n")
	for h := 0; h < 10; h++ {
		fmt.Fprintf(&b, "@@ -%d,30 +%d,30 @@\n", h*100+1, h*100+1)
		for i := 0; i < 30; i++ {
			b.WriteString("-o\n")
		}
		for i := 0; i < 30; i++ {
			b.WriteString("+n\n")
		}
	}
	files := []File{{Status: Modified, Path: "big.go"}}
	if err := ApplyHunks(files, strings.NewReader(b.String())); err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, h := range files[0].Hunks {
		if len(h.Lines) > MaxHunkLines {
			t.Fatalf("hunk over cap: %d", len(h.Lines))
		}
		if len(h.Lines)+h.Omitted != 60 {
			t.Fatalf("lines %d + omitted %d != 60", len(h.Lines), h.Omitted)
		}
		total += len(h.Lines)
	}
	if total != MaxFileLines || len(files[0].Hunks) != 10 || files[0].Hunks[9].To != 930 {
		t.Fatalf("file cap: total %d, hunks %d", total, len(files[0].Hunks))
	}
}

func TestApplyHunksRejectsABrokenHeader(t *testing.T) {
	files := []File{{Status: Modified, Path: "a.go"}}
	for _, h := range []string{"@@ -1 +x @@", "@@ nothing @@", "@@ -1 +2,y @@", "@@ -x +1 @@", "@@ -1"} {
		if err := ApplyHunks(files, strings.NewReader("--- a/a.go\n+++ b/a.go\n"+h+"\n")); err == nil {
			t.Errorf("%q: want error", h)
		}
	}
	for _, h := range []string{`+++ "b/\x.go"`, `--- "a/\x.go"`} {
		if err := ApplyHunks(files, strings.NewReader(h+"\n")); err == nil {
			t.Errorf("%q bad quoting: want error", h)
		}
	}
}

func TestApplyHunksPassesAReadError(t *testing.T) {
	boom := errors.New("boom")
	if err := ApplyHunks(nil, iotest.ErrReader(boom)); !errors.Is(err, boom) {
		t.Fatalf("got %v, want %v", err, boom)
	}
}

func TestApplyHunksFollowsARenameToItsNewPath(t *testing.T) {
	files := []File{{Status: Renamed, Path: "new.go"}}
	patch := "diff --git a/old.go b/new.go\nsimilarity index 90%\nrename from old.go\nrename to new.go\n--- a/old.go\n+++ b/new.go\n@@ -2 +2 @@\n-a\n+b\n"
	if err := ApplyHunks(files, strings.NewReader(patch)); err != nil {
		t.Fatal(err)
	}
	if h := files[0].Hunks; len(h) != 1 || h[0].From != 2 {
		t.Fatalf("new.go hunks %+v, want the one hunk under the +++ path", h)
	}
}

func TestHeaderPathStripsOnlyAPrefixFollowedByAName(t *testing.T) {
	for in, want := range map[string]string{"b/x.go": "x.go", "x.go": "x.go", "a": "a", "b/": "b/", "/dev/null": ""} {
		if got, err := headerPath(in); err != nil || got != want {
			t.Errorf("headerPath(%q) = %q, %v, want %q", in, got, err, want)
		}
	}
}

func TestParseHunkHeaderNeedsMinusThenPlus(t *testing.T) {
	if _, _, _, err := parseHunkHeader("@@ +1 -1 @@"); err == nil {
		t.Error("swapped ranges: want an error")
	}
	if h, _, _, err := parseHunkHeader("@@ -1 +4"); err != nil || h.From != 4 {
		t.Errorf("header without the closing @@ = %+v, %v, want From 4", h, err)
	}
}

// With diff.interHunkContext set, git fuses near hunks even under
// --unified=0: the context line between them counts in both header counts,
// and the next file's headers must still be read as headers.
func TestApplyHunksCountsAContextLineOfAMergedHunk(t *testing.T) {
	files := []File{{Status: Modified, Path: "f"}, {Status: Modified, Path: "g"}}
	patch := "diff --git a/f b/f\n--- a/f\n+++ b/f\n@@ -1,3 +1,3 @@\n-a\n+A\n b\n-c\n+C\n" +
		"diff --git a/g b/g\n--- a/g\n+++ b/g\n@@ -2 +2 @@\n-2\n+X\n"
	if err := ApplyHunks(files, strings.NewReader(patch)); err != nil {
		t.Fatal(err)
	}
	if h := files[0].Hunks; len(h) != 1 || h[0].From != 1 || h[0].To != 3 || !reflect.DeepEqual(h[0].Lines, []string{"-a", "+A", "-c", "+C"}) {
		t.Fatalf("f hunks %+v", h)
	}
	if h := files[1].Hunks; len(h) != 1 || h[0].From != 2 || h[0].To != 2 || !reflect.DeepEqual(h[0].Lines, []string{"-2", "+X"}) {
		t.Fatalf("g hunks %+v", h)
	}
}
