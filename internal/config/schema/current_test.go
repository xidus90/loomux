package schema

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func byID(t *testing.T, text string) map[string]Entry {
	t.Helper()
	entries, err := Current(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(Keys()) {
		t.Fatalf("%d entries for %d keys", len(entries), len(Keys()))
	}
	out := map[string]Entry{}
	for i, e := range entries {
		if want := Keys()[i].ID(); e.Key.ID() != want {
			t.Fatalf("entry %d is %s, want %s", i, e.Key.ID(), want)
		}
		out[e.Key.ID()] = e
	}
	return out
}

func TestCurrentTellsWhereEachValueComesFrom(t *testing.T) {
	text := "[commit]\nlanguage = \"de\"\n[[policy.paths.rules]]\nmatch = [\"a\"]\nreason = \"r\"\n[[policy.paths.rules]]\nmatch = [\"b\"]\nreason = \"r\"\n[verify.go.lint]\ncommands = []\n"
	got := byID(t, text)
	if e := got["commit.language"]; e.Origin != Set || e.Value != `"de"` || e.Input != "de" {
		t.Errorf("language %+v", e)
	}
	if e := got["commit.threshold"]; e.Origin != Default || e.Value != "2" {
		t.Errorf("threshold %+v", e)
	}
	if e := got["policy.paths.rules"]; e.Origin != Set || e.Count != 2 || e.Value != "map[match:[a] reason:r]\nmap[match:[b] reason:r]" {
		t.Errorf("paths %+v", e)
	}
	if e := got["commit.allow"]; e.Origin != Unset || e.Count != 0 {
		t.Errorf("allow %+v", e)
	}
	if e := got["layout.hub"]; e.Origin != Unset || e.Value != "" || e.Input != "" {
		t.Errorf("hub %+v", e)
	}
	// The verify tables come from the presets, which the schema default only
	// mirrors; the preset is the truer answer.
	if e := got["verify.profiles"]; e.Origin != Preset || e.Value != "" {
		t.Errorf("profiles %+v", e)
	}
}

func TestInputFormIsWhatAHumanTypes(t *testing.T) {
	got := byID(t, "[layout]\nwiki = 'docs\\wiki \"x\"'\n[index]\ninclude = [\"a\", \"b\"]\n[verify]\ntimeout = 30\n[modules]\nbrain = false\n")
	for id, want := range map[string]string{
		"layout.wiki":      `docs\wiki "x"`,
		"index.include":    "a, b",
		"commit.threshold": "2",
		"verify.timeout":   "30",
		"modules.brain":    "false",
		"commit.language":  "en",
		"worktree.mirror":  "",
	} {
		if e := got[id]; e.Input != want {
			t.Errorf("%s input %q, want %q", id, e.Input, want)
		}
	}
	for id, want := range map[string]string{
		"layout.wiki":    `"docs\\wiki \"x\""`,
		"index.include":  `["a", "b"]`,
		"verify.timeout": "30",
		"modules.brain":  "false",
	} {
		if e := got[id]; e.Value != want || e.Origin != Set {
			t.Errorf("%s value %q (%s), want %q", id, e.Value, e.Origin, want)
		}
	}
}

// The encoder writes a table as a [v] section, not as v = {…}; a set table
// is therefore shown the way Go prints the decoded map.
func TestASetTableIsShownOnOneLine(t *testing.T) {
	var b strings.Builder
	if err := toml.NewEncoder(&b).Encode(map[string]any{"v": map[string]any{"a": true}}); err != nil || !strings.HasPrefix(b.String(), "[v]") {
		t.Fatalf("the encoder now writes %q; literal may drop its table fallback", b.String())
	}
	got := byID(t, "[model.roles]\ndescribe = true\n[verify.profiles]\nedit = [\"lint\"]\n")
	if e := got["model.roles"]; e.Origin != Set || e.Value != "map[describe:true]" || strings.Contains(e.Value, "\n") {
		t.Errorf("roles %+v", e)
	}
	if e := got["verify.profiles"]; e.Origin != Set || e.Value != "map[edit:[lint]]" {
		t.Errorf("profiles %+v", e)
	}
}

func TestAValueWhereATableBelongsLeavesItsKeysUnfound(t *testing.T) {
	got := byID(t, "commit = 1\n")
	if e := got["commit.language"]; e.Origin != Default {
		t.Errorf("language %+v", e)
	}
}

func TestCurrentRefusesBrokenTOML(t *testing.T) {
	if _, err := Current("[x"); err == nil {
		t.Fatal("want an error")
	}
}
