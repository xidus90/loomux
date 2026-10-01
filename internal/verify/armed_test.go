package verify

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/detect"
)

func armedAt(t *testing.T, text string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "armed.toml"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// keyOf is the key of the planned job with this kind, stack and area.
func keyOf(t *testing.T, jobs []Job, kind, stack, area string) (key, name string) {
	t.Helper()
	for _, j := range jobs {
		if j.Kind == kind && j.Stack == stack && j.Area == area {
			return LaneKey(j), j.Name
		}
	}
	t.Fatalf("no %s/%s in %s among %s", kind, stack, area, names(jobs))
	return "", ""
}

// A second area renames the lane (lint/go becomes lint/go@.), and a key built
// from the name would drop the lane back into probation without a word.
func TestTheKeyOfALaneIsTheSameWithOneAreaAndWithTwo(t *testing.T) {
	const src = "[verify.project]\nlint = \"echo hi\"\n"
	one := effFor(t, src, goOnly)
	two := effFor(t, src, detect.Facts{Stacks: []string{"go"}, Areas: map[string][]string{"go": {".", "tools"}}})
	req := Request{Kinds: []string{"lint", "graph"}, Scope: ScopeCheck}
	ready := env(t.TempDir())
	ready.GraphReady = func(string) (bool, string) { return true, "" }
	jobsOne, err := Plan(one, req, ready)
	if err != nil {
		t.Fatal(err)
	}
	jobsTwo, err := Plan(two, req, ready)
	if err != nil {
		t.Fatal(err)
	}
	keyOne, nameOne := keyOf(t, jobsOne, "lint", "go", ".")
	keyTwo, nameTwo := keyOf(t, jobsTwo, "lint", "go", ".")
	if nameOne != "lint/go" || nameTwo != "lint/go@." {
		t.Fatalf("the world must rename the lane: %q, %q", nameOne, nameTwo)
	}
	if keyOne != "lint/go@." || keyTwo != keyOne {
		t.Fatalf("keys %q and %q, want lint/go@. twice", keyOne, keyTwo)
	}
	if key, _ := keyOf(t, jobsTwo, "lint", "go", "tools"); key != "lint/go@tools" {
		t.Errorf("the second area: %q", key)
	}
	if key, _ := keyOf(t, jobsOne, "lint", "project", "."); key != "lint/project@." {
		t.Errorf("the project lane: %q", key)
	}
	// The graph lane's name carries no area at all; its key still does.
	if key, name := keyOf(t, jobsTwo, "graph", "go", "."); key != "graph/go@." || name != "graph/go" {
		t.Errorf("the graph lane: key %q, name %q", key, name)
	}
	if got := LaneKey(Job{Kind: "lint", Stack: "go", Area: `tools\gen`}); got != "lint/go@tools/gen" {
		t.Errorf("an area is spelt with slashes: %q", got)
	}
}

func TestWithoutTheFileEveryLaneIsArmed(t *testing.T) {
	set, err := ReadArmed(t.TempDir())
	if err != nil || set.Exists || len(set.Keys) != 0 {
		t.Fatalf("%+v %v", set, err)
	}
	if !set.Arms(Job{Kind: "lint", Stack: "go", Area: "."}) {
		t.Fatal("no file must arm every lane")
	}
}

func TestTheFileArmsOnlyTheLanesItNames(t *testing.T) {
	set, err := ReadArmed(armedAt(t, "armed = [\n  \"test/go@.\",\n  \"lint/go@.\",\n  \"lint/go@.\",\n]\n"))
	if err != nil || !set.Exists || !slices.Equal(set.Keys, []string{"lint/go@.", "test/go@."}) {
		t.Fatalf("%+v %v", set, err)
	}
	if !set.Arms(Job{Kind: "lint", Stack: "go", Area: "."}) || set.Arms(Job{Kind: "lint", Stack: "go", Area: "tools"}) {
		t.Fatalf("%+v arms the wrong lanes", set)
	}
	empty, err := ReadArmed(armedAt(t, "# a comment\narmed = []\n"))
	if err != nil || !empty.Exists || empty.Arms(Job{Kind: "lint", Stack: "go", Area: "."}) {
		t.Fatalf("an empty list arms nothing: %+v %v", empty, err)
	}
}

// A broken file disarms nothing. The empty file is the nearest wrong
// neighbour of `armed = []`: a file a merge or an editor truncated.
func TestAnUnreadableFileArmsEveryLane(t *testing.T) {
	for name, text := range map[string]string{
		"empty":            "",
		"only a comment":   "# nothing\n",
		"conflict markers": "<<<<<<< HEAD\narmed = []\n=======\narmed = [\"lint/go@.\"]\n>>>>>>> theirs\n",
		"a string":         "armed = \"lint/go@.\"\n",
		"a number inside":  "armed = [1]\n",
		"an unknown key":   "armed = []\nother = true\n",
		"a table":          "[armed]\nlint = true\n",
	} {
		set, err := ReadArmed(armedAt(t, text))
		if err == nil || !strings.HasPrefix(err.Error(), ".loomux/armed.toml ") || !strings.HasSuffix(err.Error(), "; every lane is armed") {
			t.Errorf("%s: err %v", name, err)
		}
		if set.Exists || !set.Arms(Job{Kind: "lint", Stack: "go", Area: "."}) {
			t.Errorf("%s: %+v must arm every lane", name, set)
		}
	}
	// A directory in the file's place cannot be read either.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "armed.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if set, err := ReadArmed(root); err == nil || set.Exists {
		t.Errorf("a directory: %+v %v", set, err)
	}
	// Two unknown keys are named in a fixed order.
	_, err := ReadArmed(armedAt(t, "zeta = 1\nalpha = 2\narmed = []\n"))
	if err == nil || !strings.Contains(err.Error(), "alpha") {
		t.Errorf("unknown keys: %v", err)
	}
}

func TestWithWithoutAndMissingKeepTheKeysSortedAndOnce(t *testing.T) {
	set := ArmedSet{}.With("test/go@.", "lint/go@.", "test/go@.")
	if !set.Exists || !slices.Equal(set.Keys, []string{"lint/go@.", "test/go@."}) {
		t.Fatalf("%+v", set)
	}
	if got := set.Without("lint/go@.", "nope"); !got.Exists || !slices.Equal(got.Keys, []string{"test/go@."}) {
		t.Fatalf("%+v", got)
	}
	if !slices.Equal(set.Keys, []string{"lint/go@.", "test/go@."}) {
		t.Fatalf("Without changed its receiver: %+v", set)
	}
	// Taking a key out of a project without the file leaves a file: the one
	// that arms what is left. It does not stay "no file, every lane armed".
	if got := (ArmedSet{}).Without("lint/go@."); !got.Exists || len(got.Keys) != 0 {
		t.Fatalf("Without on the zero value: %+v", got)
	}
	if got := set.Missing([]string{"test/go@.", "types/go@.", "coverage/go@.", "types/go@."}); !slices.Equal(got, []string{"coverage/go@.", "types/go@."}) {
		t.Fatalf("missing %v", got)
	}
}

func TestTheTextIsOneKeyPerLine(t *testing.T) {
	const head = "# Lanes that are armed: a red run of one of them fails the gate.\n" +
		"# Written by the pre-commit gate; a human edits it through `loomux gate`.\n"
	if got := (ArmedSet{Exists: true}).Text(); got != head+"armed = []\n" {
		t.Fatalf("empty: %q", got)
	}
	got := ArmedSet{}.With("test/go@.", "lint/python@.").Text()
	if got != head+"armed = [\n  \"lint/python@.\",\n  \"test/go@.\",\n]\n" {
		t.Fatalf("two keys: %q", got)
	}
	// What is written reads back as it was, a key with a quote included.
	odd := ArmedSet{}.With(`lint/go@a"b`)
	back, err := ReadArmed(armedAt(t, odd.Text()))
	if err != nil || !slices.Equal(back.Keys, odd.Keys) {
		t.Fatalf("%+v %v", back, err)
	}
}

func TestWriteArmedReplacesTheFileWithLF(t *testing.T) {
	root := t.TempDir()
	if err := WriteArmed(root, ArmedSet{}.With("lint/go@.")); err != nil {
		t.Fatal(err)
	}
	if err := WriteArmed(root, ArmedSet{}.With("lint/go@.", "test/go@.")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".loomux", "armed.toml"))
	if err != nil || bytes.Contains(data, []byte("\r")) || string(data) != (ArmedSet{}).With("lint/go@.", "test/go@.").Text() {
		t.Fatalf("%q %v", data, err)
	}
	entries, _ := os.ReadDir(filepath.Join(root, ".loomux"))
	if len(entries) != 1 {
		t.Fatalf("a temporary file stayed behind: %v", entries)
	}
}

func TestWriteArmedReportsWhatItCannotWrite(t *testing.T) {
	// .loomux is a file: the directory cannot be made.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".loomux"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteArmed(root, ArmedSet{Exists: true}); err == nil {
		t.Fatal("a file in place of .loomux went through")
	}
	// armed.toml is a directory: the swap cannot land.
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "armed.toml", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteArmed(root, ArmedSet{Exists: true}); err == nil {
		t.Fatal("a directory in place of the file went through")
	}
}

func TestHookArmsReadsTheCallNotTheComment(t *testing.T) {
	for text, want := range map[string]bool{
		"#!/bin/sh\n\"/opt/loomux\" check precommit --arm\ncode=$?\nexit \"$code\"\n": true,
		"#!/bin/sh\nloomux check precommit --root . --arm\n":                          true,
		"#!/bin/sh\nexec \"/opt/loomux\" check precommit\n":                           false,
		"#!/bin/sh\n# loomux check precommit --arm\nsh ci/gate.sh\n":                  false,
		"#!/bin/sh\nloomux gate status --arm\n":                                       false,
		"#!/bin/sh\nloomux check precommit --armed\n":                                 false,
		// The flag ends where the shell ends a word: at a separator, a quote,
		// the end of the line.
		"loomux check precommit --arm; code=$?\n":         true,
		"loomux check precommit --arm&&echo ok\n":         true,
		"loomux check precommit --arm||exit 1\n":          true,
		"(loomux check precommit --arm)\n":                true,
		"loomux check precommit --arm|tee log\n":          true,
		"loomux check precommit \"--arm\"\n":              true,
		"loomux check precommit '--arm'\n":                true,
		"loomux check precommit --arm \"$@\"\n":           true,
		"loomux check precommit --arm":                    true,
		"loomux check precommit --arm\r\n":                true,
		"echo \"a #b\" && loomux check precommit --arm\n": true,
		"loomux check precommit --note 'fix #1' --arm\n":  true,
		"loomux check precommit --note a#1 --arm\n":       true,
		"loomux check precommit --note 'x' # --arm\n":     false,
		"loomux check stop;# precommit --arm\n":           false,
		// A # after a character of several bytes is within a word.
		"loomux check precommit --note=voilà#1 --arm\n": true,
		"loomux check precommit aą#1 --arm\n":           true,
		// A quoted string is one word: a hook that prints the call does not
		// make it, and --arm counts only in the command of the call.
		"echo \"loomux check precommit --arm\"\n":           false,
		"echo 'loomux check precommit --arm' >> log\n":      false,
		"loomux check precommit; echo \"--arm\"\n":          false,
		"loomux check precommit && echo --arm\n":            false,
		"echo --arm; loomux check precommit\n":              false,
		"loomux check \"precommit\" --arm\n":                true,
		"echo ok; loomux check precommit --arm\n":           true,
		"loomux check precommit --arm # arms green lanes\n": true,
		"loomux check precommit --note \"a; b\" --arm\n":    true,
		"loomux check precommit --show # --arm\n":           false,
		"loomux check precommit --show #--arm\n":            false,
		"loomux check precommit --show\t# --arm\n":          false,
		"loomux check precommit --show; # --arm\n":          false,
		"loomux check stop # precommit --arm\n":             false,
		"  # loomux check precommit --arm\n":                false,
		"loomux check precommit --arm=false\n":              false,
		"":                                                  false,
	} {
		if got := HookArms(text); got != want {
			t.Errorf("HookArms(%q) = %v, want %v", text, got, want)
		}
	}
}
