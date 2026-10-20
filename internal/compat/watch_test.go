package compat

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func entryKeys(es []Entry) []string {
	var ks []string
	for _, e := range es {
		ks = append(ks, e.Key())
	}
	return ks
}

// withAddon is the keys, in list order, of the entries that run one of names.
func withAddon(list []Entry, names ...string) []string {
	var ks []string
	for _, e := range list {
		for _, a := range e.Addons {
			if slices.Contains(names, a.Name) {
				ks = append(ks, e.Key())
				break
			}
		}
	}
	return ks
}

func TestWatchedEntries(t *testing.T) {
	list, _ := Load()
	all := entryKeys(list)
	gdunit := withAddon(list, "gdUnit4", "NanoCoverage")
	gut := withAddon(list, "GUT")
	both := withAddon(list, "gdUnit4", "NanoCoverage", "GUT")
	if len(gdunit) == 0 || len(gut) == 0 {
		t.Fatal("the list lacks gdUnit4 or GUT entries")
	}
	cases := []struct {
		name  string
		paths []string
		want  []string
	}{
		{"godotsuite", []string{"internal/verify/godotsuite/gd/x.gd"}, gdunit},
		{"gutsuite", []string{"internal/verify/gutsuite/a.go"}, gut},
		{"presets", []string{"internal/verify/presets.toml"}, all},
		{"frameworks", []string{"internal/verify/frameworks.toml"}, all},
		{"script", []string{"ci/compat-plan.sh"}, all},
		{"compat code", []string{"internal/compat/run.go"}, all},
		{"compat test", []string{"internal/compat/run_test.go"}, nil},
		{"docs", []string{"docs/en/support.md"}, nil},
		{"two rules, once each, list order", []string{"internal/verify/gutsuite/a.go", "internal/verify/godotsuite/b.go"}, both},
	}
	for _, c := range cases {
		got, err := WatchedEntries(list, c.paths)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestWatchedEntriesBySelector(t *testing.T) {
	list, _ := Load()
	w := []Watch{{Paths: []string{"p/**"}, Stacks: []string{"go"}}}
	got, err := matchWatches(list, w, []string{"p/a/b"})
	if err != nil || len(got) == 0 {
		t.Fatalf("stack watch selected %v, err %v", got, err)
	}
	for _, k := range got {
		if !strings.HasPrefix(k, "go/") {
			t.Errorf("stack watch selected %s", k)
		}
	}
}

func TestWatchedEntriesBadPattern(t *testing.T) {
	list, _ := Load()
	if _, err := matchWatches(list, []Watch{{Paths: []string{"[x"}, All: true}}, []string{"a"}); err == nil {
		t.Error("a malformed glob passed")
	}
	if _, err := matchWatches(list, []Watch{{Paths: []string{"a"}, Except: []string{"[x"}, All: true}}, []string{"a"}); err == nil {
		t.Error("a malformed except glob passed")
	}
}

func TestWatchLoadErrors(t *testing.T) {
	head := "[[watch]]\npaths = [\"a\"]\nall = true\n\n[[watch]]\n"
	cases := map[string]string{
		"no selector":   head + "paths = [\"a\"]\n",
		"two selectors": head + "paths = [\"a\"]\nall = true\nstacks = [\"go\"]\n",
		"empty paths":   head + "paths = []\nall = true\n",
		"unknown key":   head + "paths = [\"a\"]\nall = true\nbogus = 1\n",
		"unknown stack": head + "paths = [\"a\"]\nstacks = [\"cobol\"]\n",
		"unknown addon": head + "paths = [\"a\"]\naddons = [\"Nope\"]\n",
		"all = false":   head + "paths = [\"a\"]\nall = false\n",
		"bad glob":      head + "paths = [\"[x\"]\nall = true\n",
		"bad except":    head + "paths = [\"a\"]\nexcept = [\"[x\"]\nall = true\n",
	}
	if _, err := decodeWatches([]byte("[[watch]]\npaths = \"a\"\nall = true\n"), "compat.toml"); err == nil {
		t.Error("paths of the wrong type passed")
	}
	for name, doc := range cases {
		_, err := decodeWatches([]byte(doc), "compat.toml")
		if err == nil || !strings.Contains(err.Error(), "watch 2") {
			t.Errorf("%s: err = %v, want one naming watch 2", name, err)
		}
	}
	if _, err := decodeWatches([]byte("[[watch"), "compat.toml"); err == nil {
		t.Error("broken TOML passed")
	}
}

func TestChangedSinceSelectsWatched(t *testing.T) {
	list, _ := Load()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	put(t, dir, listPath, string(listText))
	put(t, dir, "a.txt", "1")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-qm", "one")
	put(t, dir, "ci/compat-plan.sh", "2")
	put(t, dir, "internal/compat/run_test.go", "2")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-qm", "two")
	_, en, err := ChangedSince(dir, "HEAD~1")
	if err != nil || !reflect.DeepEqual(en, entryKeys(list)) {
		t.Errorf("entries = %v, err %v; want all", en, err)
	}
}
