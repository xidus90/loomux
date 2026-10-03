package cli

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// checkedPage is a page none of the three axes has anything to say about: a
// type the vocabulary knows, one complete source and a trust family.
func checkedPage(title string) string {
	return "---\ntitle: " + title + "\ndescription: d\ntype: Topic\n" +
		"sources:\n  - id: s1\n    resource: brain://x/y\n" +
		"    doc_id: d1\n    content_hash: h1\n    revision: 1\n" +
		"generated:\n  by: test\n---\n\n# " + title + "\n"
}

// checkedArea lays out a bundle of two linked pages with its catalog and its
// log under base, so that no rule speaks for a reason the caller did not ask
// about, and answers the registry entry for it.
func checkedArea(t *testing.T, base, scope string, extra string) (string, string) {
	t.Helper()
	root := filepath.Join(base, strings.ReplaceAll(scope, "/", "-"))
	wiki := filepath.Join(root, "wiki")
	writeFile(t, filepath.Join(wiki, "index.md"),
		"# c\n\n* [a](a.md) -- first\n* [b](b.md) -- second\n")
	writeFile(t, filepath.Join(wiki, "log.md"), "# log\n\n## 2026-09-01\n")
	writeFile(t, filepath.Join(wiki, "a.md"), checkedPage("a")+"\n[b](b.md)\n")
	writeFile(t, filepath.Join(wiki, "b.md"), checkedPage("b")+"\n[a](a.md)\n")
	entry := "[[area]]\nscope = " + strconv.Quote(scope) +
		"\npath = " + strconv.Quote(filepath.ToSlash(root)) +
		"\nwiki = " + strconv.Quote(filepath.ToSlash(wiki)) + "\n" + extra
	return wiki, entry
}

// checkWorld registers the given entries in a state directory of its own and
// keeps the old one empty, so no run reads this machine's registry.
func checkWorld(t *testing.T, entries ...string) {
	t.Helper()
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	t.Setenv("LOOMUX_STATE_DIR", state)
	writeFile(t, filepath.Join(state, "registry.toml"), strings.Join(entries, "\n"))
}

func TestBrainCheckFileOnACleanPageIsGreen(t *testing.T) {
	base := t.TempDir()
	wiki, entry := checkedArea(t, base, "project/a", "")
	checkWorld(t, entry)
	code, out, errOut := run("brain", "check", "file", filepath.Join(wiki, "a.md"))
	if code != 0 || out != "" || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestBrainCheckFileReportsAnErrorWithExitOne(t *testing.T) {
	base := t.TempDir()
	wiki, entry := checkedArea(t, base, "project/a", "")
	checkWorld(t, entry)
	page := writeFile(t, filepath.Join(wiki, "c.md"), "---\ntitle: c\n---\n\n# c\n")
	code, out, _ := run("brain", "check", "file", page)
	if code != 1 {
		t.Fatalf("code %d, want 1; out %q", code, out)
	}
	if !strings.Contains(out, "[error] okf/type-missing c.md: ") {
		t.Fatalf("out %q names no missing type on c.md", out)
	}
}

func TestBrainCheckFileShowsNotesOnlyWhenAsked(t *testing.T) {
	base := t.TempDir()
	wiki, entry := checkedArea(t, base, "project/a", "")
	checkWorld(t, entry)
	// No `generated`: the trust family is a note, the one degree the default
	// run leaves out.
	page := writeFile(t, filepath.Join(wiki, "c.md"), strings.Replace(
		checkedPage("c"), "generated:\n  by: test\n", "", 1))
	_, quiet, _ := run("brain", "check", "file", page)
	if strings.Contains(quiet, "[note]") {
		t.Fatalf("a note without --notes: %q", quiet)
	}
	for _, args := range [][]string{
		{"brain", "check", "file", "--notes", page},
		{"brain", "check", "file", page, "--notes"},
	} {
		code, out, _ := run(args...)
		if code != 0 || !strings.Contains(out, "[note] okf/no-trust-family c.md: ") {
			t.Fatalf("%v: code %d, out %q", args, code, out)
		}
	}
}

func TestBrainCheckFileRefusesWhatItCannotRead(t *testing.T) {
	base := t.TempDir()
	wiki, entry := checkedArea(t, base, "project/a", "")
	checkWorld(t, entry)
	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"no path", []string{"brain", "check", "file"}, "error: 'loomux brain check file' needs a path\n"},
		{"missing", []string{"brain", "check", "file", filepath.Join(wiki, "gone.md")}, "error: no such file: "},
		{"directory", []string{"brain", "check", "file", wiki}, " is not a file\n"},
	} {
		code, out, errOut := run(c.args...)
		if code != 2 || out != "" || !strings.Contains(errOut, c.want) {
			t.Errorf("%s: code %d, out %q, err %q", c.name, code, out, errOut)
		}
	}
}

func TestBrainCheckFileNamesAnUnreadablePathInItsOwnWords(t *testing.T) {
	checkWorld(t)
	// Two names Windows refuses before it looks for the file: a NUL byte
	// fails the full-path lookup, an angle bracket the Stat. Neither is
	// "does not exist", and neither message may say so.
	for _, name := range []string{"bad\x00name.md", "bad<name.md"} {
		code, out, errOut := run("brain", "check", "file", name)
		if code != 2 || out != "" || !strings.HasPrefix(errOut, "error: ") ||
			strings.Contains(errOut, "no such file") {
			t.Errorf("%q: code %d, err %q", name, code, errOut)
		}
	}
}

func TestBrainCheckBundleChecksOneRegisteredArea(t *testing.T) {
	base := t.TempDir()
	wiki, entry := checkedArea(t, base, "project/a", "")
	checkWorld(t, entry)
	code, out, errOut := run("brain", "check", "bundle", "--scope", "project/a")
	if code != 0 || out != "" || errOut != "" {
		t.Fatalf("clean: code %d, out %q, err %q", code, out, errOut)
	}
	writeFile(t, filepath.Join(wiki, "lonely.md"), checkedPage("lonely"))
	code, out, _ = run("brain", "check", "bundle", "--scope", "project/a")
	if code != 1 || !strings.Contains(out, "[error] house/orphan project/a/lonely.md: ") {
		t.Fatalf("orphan: code %d, out %q", code, out)
	}
}

func TestBrainCheckBundleRefusesWithoutAKnownScope(t *testing.T) {
	base := t.TempDir()
	_, entry := checkedArea(t, base, "project/a", "")
	checkWorld(t, entry)
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"brain", "check", "bundle"}, "error: 'loomux brain check bundle' needs --scope <area>\n"},
		{[]string{"brain", "check", "bundle", "--scope", "project/b"}, "error: no area named 'project/b' in the registry\n"},
	} {
		code, out, errOut := run(c.args...)
		if code != 2 || out != "" || errOut != c.want {
			t.Errorf("%v: code %d, out %q, err %q", c.args, code, out, errOut)
		}
	}
}

func TestBrainCheckAllJudgesTheFederation(t *testing.T) {
	base := t.TempDir()
	_, project := checkedArea(t, base, "project/a", "")
	shared, sharedEntry := checkedArea(t, base, "engineering/x", "shared = true\n")
	writeFile(t, filepath.Join(shared, "a.md"), strings.Replace(checkedPage("a"),
		"brain://x/y", "brain://project/a/topics/y", 1)+"\n[b](b.md)\n")
	checkWorld(t, project, sharedEntry)
	code, out, _ := run("brain", "check", "all")
	if code != 1 || !strings.Contains(out, "[error] house/wrong-direction engineering/x/a.md: ") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestBrainCheckAllRefusesAnUnreadableRegistry(t *testing.T) {
	checkWorld(t, "this is not TOML {{{\n")
	code, out, errOut := run("brain", "check", "all")
	if code != 2 || out != "" || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestBrainCheckRefusesAnUnknownWidthOrFlag(t *testing.T) {
	checkWorld(t)
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"brain", "check"}, "error: 'loomux brain check' needs file, bundle or all\n"},
		{[]string{"brain", "check", "code"}, "error: unknown width \"code\"\n"},
		{[]string{"brain", "check", "pages"}, "error: unknown width \"pages\"\n"},
		{[]string{"brain", "check", "all", "--lane", "x"}, "flag provided but not defined: -lane"},
	} {
		code, out, errOut := run(c.args...)
		if code != 2 || out != "" || !strings.Contains(errOut, c.want) {
			t.Errorf("%v: code %d, out %q, err %q", c.args, code, out, errOut)
		}
	}
}

// `bundle` is one area. `--scope all` names every area, which is the width
// `all`; running only the first of them, or panicking on a registry without
// any wiki, would both answer a question nobody asked.
func TestBrainCheckBundleRefusesScopeAll(t *testing.T) {
	base := t.TempDir()
	_, a := checkedArea(t, base, "project/a", "")
	_, b := checkedArea(t, base, "project/b", "")
	for _, entries := range [][]string{{a, b}, {"[[area]]\nscope = \"project/n\"\npath = \"n\"\n"}} {
		checkWorld(t, entries...)
		code, out, errOut := run("brain", "check", "bundle", "--scope", "all")
		if code != 2 || out != "" || !strings.Contains(errOut, "loomux brain check all") {
			t.Fatalf("code %d, out %q, err %q", code, out, errOut)
		}
	}
}
