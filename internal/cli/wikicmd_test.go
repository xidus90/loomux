package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/testlock"
)

// wikiEntry is a registry entry for scope with the given wiki path, plus any
// extra lines.
func wikiEntry(scope, path, wiki, extra string) string {
	entry := "[[area]]\nscope = " + strconv.Quote(scope) + "\npath = " + strconv.Quote(filepath.ToSlash(path)) + "\n"
	if wiki != "" {
		entry += "wiki = " + strconv.Quote(filepath.ToSlash(wiki)) + "\n"
	}
	return entry + extra
}

func TestWikiInitWritesTheFrameAndNamesEveryFile(t *testing.T) {
	base := t.TempDir()
	wiki := filepath.Join(base, "wiki")
	checkWorld(t, wikiEntry("project/a", base, wiki, ""))
	code, out, errOut := run("wiki", "init", "--scope", "project/a")
	if code != 0 || errOut != "" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 5 || lines[0] != filepath.Join(wiki, "_schema.md") {
		t.Fatalf("out %q", out)
	}
	// A second run writes nothing and so names nothing.
	if code, out, _ := run("wiki", "init", "--scope", "project/a"); code != 0 || out != "" {
		t.Fatalf("second run: code %d, out %q", code, out)
	}
}

func TestWikiInitRefusesAnAreaItCannotScaffold(t *testing.T) {
	base := t.TempDir()
	checkWorld(t, wikiEntry("project/a", base, "", ""))
	for _, c := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"wiki", "init", "--scope", "project/b"}, 1, "error: no area named 'project/b' in the registry\n"},
		{[]string{"wiki", "init", "--scope", "project/a"}, 1, "error: area 'project/a' declares no wiki path; add `wiki = ...` to its entry\n"},
		{[]string{"wiki", "init"}, 2, "error: the following arguments are required: --scope\n"},
		{[]string{"wiki", "init", "--scope", "project/a", "extra"}, 2, "error: unrecognized arguments: extra\n"},
		{[]string{"wiki", "init", "--bogus"}, 2, "flag provided but not defined: -bogus"},
		{[]string{"wiki"}, 2, "error: 'loomux wiki' needs init, types or retype\n"},
		{[]string{"wiki", "gate"}, 2, "error: unknown wiki command \"gate\"\n"},
	} {
		code, out, errOut := run(c.args...)
		if code != c.code || out != "" || !strings.Contains(errOut, c.want) {
			t.Errorf("%v: code %d, out %q, err %q", c.args, code, out, errOut)
		}
	}
}

func TestWikiInitReportsAFileItCannotWrite(t *testing.T) {
	base := t.TempDir()
	wiki := filepath.Join(base, "wiki")
	// A directory in the place of the log: the write fails and names it.
	if err := os.MkdirAll(filepath.Join(wiki, "log.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	checkWorld(t, wikiEntry("project/a", base, wiki, ""))
	code, out, errOut := run("wiki", "init", "--scope", "project/a")
	if code != 1 || out != "" || !strings.Contains(errOut, "log.md: cannot be written") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestWikiCommandsRefuseAnUnreadableRegistry(t *testing.T) {
	checkWorld(t, "this is not TOML {{{\n")
	for _, args := range [][]string{
		{"wiki", "init", "--scope", "a"},
		{"wiki", "types"},
		{"wiki", "retype", "--scope", "a", "--from", "x", "--to", "y"},
	} {
		code, out, errOut := run(args...)
		if code != 1 || out != "" || !strings.HasPrefix(errOut, "error: ") {
			t.Errorf("%v: code %d, out %q, err %q", args, code, out, errOut)
		}
	}
}

func TestWikiTypesPrintsTheCensus(t *testing.T) {
	base := t.TempDir()
	_, a := checkedArea(t, base, "project/a", "")
	checkWorld(t, a)
	code, out, errOut := run("wiki", "types")
	if code != 0 || errOut != "" || out != "Topic [origin]: 2 (project/a: 2)\n" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if code, _, _ := run("wiki", "types", "extra"); code != 2 {
		t.Fatalf("a positional argument: code %d", code)
	}
}

func TestWikiTypesStopsAtABrokenDeclaration(t *testing.T) {
	base := t.TempDir()
	_, a := checkedArea(t, base, "project/a", "")
	checkWorld(t, a)
	writeFile(t, filepath.Join(base, "project-a", ".loomux", "config.toml"), "[area\n")
	code, out, errOut := run("wiki", "types")
	if code != 1 || out != "" || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestWikiRetypeRenamesAndNamesThePages(t *testing.T) {
	base := t.TempDir()
	wiki, a := checkedArea(t, base, "project/a", "")
	checkWorld(t, a)
	code, out, errOut := run("wiki", "retype", "--scope", "project/a", "--from", "Topic", "--to", "Decision")
	want := filepath.Join(wiki, "a.md") + "\n" + filepath.Join(wiki, "b.md") + "\n"
	if code != 0 || errOut != "" || out != want {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestWikiRetypeWarnsAboutAnUnknownTarget(t *testing.T) {
	base := t.TempDir()
	_, a := checkedArea(t, base, "project/a", "")
	checkWorld(t, a)
	code, out, errOut := run("wiki", "retype", "--scope", "project/a", "--from", "Topic", "--to", "Topik")
	if code != 0 || out == "" || errOut != "warning: 'Topik' is neither core, catalogue, origin nor declared "+
		"in area 'project/a''s manifest; `loomux lint` will flag it as missing-type\n" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestWikiRetypeRefusesWhatItMustNotWrite(t *testing.T) {
	base := t.TempDir()
	_, a := checkedArea(t, base, "project/a", "readonly = true\n")
	_, b := checkedArea(t, base, "project/b", "")
	checkWorld(t, a, b)
	writeFile(t, filepath.Join(base, "project-b", ".loomux", "config.toml"), "[area\n")
	for _, c := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"wiki", "retype", "--scope", "project/a", "--from", "Topic", "--to", "Decision"}, 1,
			"error: area 'project/a' is read-only; its bundle cannot be renamed\n"},
		{[]string{"wiki", "retype", "--scope", "project/b", "--from", "Topic", "--to", "Decision"}, 1, "error: "},
		{[]string{"wiki", "retype", "--scope", "project/c", "--from", "Topic", "--to", "Decision"}, 1,
			"error: no area named 'project/c' in the registry\n"},
		{[]string{"wiki", "retype", "--scope", "project/a", "--from", "Topic"}, 2,
			"error: the following arguments are required: --to\n"},
	} {
		code, out, errOut := run(c.args...)
		if code != c.code || out != "" || !strings.HasPrefix(errOut, c.want) {
			t.Errorf("%v: code %d, out %q, err %q", c.args, code, out, errOut)
		}
	}
}

func TestWikiRetypeNamesNoPageWhenTheRunBreaks(t *testing.T) {
	base := t.TempDir()
	wiki, a := checkedArea(t, base, "project/a", "")
	checkWorld(t, a)
	testlock.Lock(t, filepath.Join(wiki, "b.md"))
	code, out, errOut := run("wiki", "retype", "--scope", "project/a", "--from", "Topic", "--to", "Decision")
	if code != 1 || out != "" || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

// A read-only area takes no write from init either, the same line retype
// draws: a frame written into it would be the first forbidden write.
func TestWikiInitRefusesAReadOnlyArea(t *testing.T) {
	base := t.TempDir()
	wiki := filepath.Join(base, "wiki")
	checkWorld(t, wikiEntry("project/a", base, wiki, "readonly = true\n"))
	code, out, errOut := run("wiki", "init", "--scope", "project/a")
	if code != 1 || out != "" || errOut != "error: area 'project/a' is read-only; no bundle is written into it\n" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if _, err := os.Stat(wiki); !os.IsNotExist(err) {
		t.Fatalf("init wrote into a read-only area: %v", err)
	}
}
