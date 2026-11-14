package target

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config/schema"
)

func TestProjectAndGlobalNameTheirFilesAndKeys(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	p, g := Project(root), Global(state)
	if p.Global || p.Path != filepath.Join(root, ".loomux", "config.toml") || len(p.Keys) != len(schema.Keys()) {
		t.Errorf("project: %+v", p)
	}
	if !g.Global || g.Path != filepath.Join(state, "config.toml") || len(g.Keys) != len(schema.GlobalKeys()) {
		t.Errorf("global: %+v", g)
	}
}

func TestReadOfAFileThatIsNotThereIsEmpty(t *testing.T) {
	text, err := Project(t.TempDir()).Read()
	if err != nil || text != "" {
		t.Fatalf("%q %v", text, err)
	}
	_, tgt := projectTarget(t, "[commit]\nlanguage = \"de\"\n")
	if text, err := tgt.Read(); err != nil || text != "[commit]\nlanguage = \"de\"\n" {
		t.Fatalf("%q %v", text, err)
	}
}

func TestEntriesNameTheGlobalFileByItsPath(t *testing.T) {
	_, tgt := projectTarget(t, "")
	entries, err := tgt.Entries("[commit]\nlanguage = \"de\"\n")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		found = found || (e.Key.ID() == "commit.language" && e.Input == "de")
	}
	if !found {
		t.Fatalf("no commit.language = de in %d entries", len(entries))
	}
	global := Global(t.TempDir())
	if _, err := global.Entries("[model\n"); err == nil || !strings.Contains(err.Error(), global.Path) {
		t.Fatalf("err = %v, want the global path named", err)
	}
	if _, err := tgt.Entries("[model\n"); err == nil || strings.Contains(err.Error(), tgt.Path) {
		t.Fatalf("err = %v, want the short project name", err)
	}
}

func TestLookupFindsKnownKeysOnly(t *testing.T) {
	tgt := Project(t.TempDir())
	if key, ok := tgt.Lookup("commit.language"); !ok || key.ID() != "commit.language" {
		t.Errorf("%v %v", key, ok)
	}
	if _, ok := tgt.Lookup("nope.nothing"); ok {
		t.Error("an unknown key was found")
	}
}

func TestChangeAndUnsetEditOneLine(t *testing.T) {
	tgt := Project(t.TempDir())
	next, err := tgt.Change("[commit]\nlanguage = \"en\"\n", "commit.language", "de")
	if err != nil || next != "[commit]\nlanguage = \"de\"\n" {
		t.Fatalf("%q %v", next, err)
	}
	// The default is never written; the line goes instead.
	if next, err = tgt.Change(next, "commit.language", "en"); err != nil || strings.Contains(next, "language") {
		t.Fatalf("%q %v", next, err)
	}
	next, err = tgt.Unset("[commit]\nlanguage = \"de\"\nthreshold = 4\n", "commit.language")
	if err != nil || next != "[commit]\nthreshold = 4\n" {
		t.Fatalf("%q %v", next, err)
	}
}

func TestChangeAndUnsetRefuseWhatTheyCannotEdit(t *testing.T) {
	tgt := Project(t.TempDir())
	for _, id := range []string{"nope.nothing", "policy.paths.rules", "verify"} {
		if _, err := tgt.Change("", id, "x"); err == nil {
			t.Errorf("Change %s: no error", id)
		}
		if _, err := tgt.Unset("", id); err == nil {
			t.Errorf("Unset %s: no error", id)
		}
	}
	if _, err := tgt.Change("", "verify.timeout", "soon"); err == nil {
		t.Error("a malformed value was taken")
	}
	if _, err := tgt.Change("[commit\n", "commit.language", "de"); err == nil {
		t.Error("a file that is no TOML was edited")
	}
	if _, err := tgt.Change("[commit]\nlanguage = \"de\"\nlanguage = \"en\"\n", "commit.language", "fr"); err == nil {
		t.Error("an ambiguous file was set")
	}
	if _, err := tgt.Unset("[commit]\nlanguage = \"de\"\nlanguage = \"en\"\n", "commit.language"); err == nil {
		t.Error("an ambiguous file was edited")
	}
}

func TestChangeOfABrainKeyNeedsAnArea(t *testing.T) {
	tgt := Project(t.TempDir())
	key := ""
	for _, k := range tgt.Keys {
		if k.Module == schema.Brain && k.ID() != "area.scope" && k.Kind != schema.Table && k.Kind != schema.TableList && !k.Named() {
			key = k.ID()
			break
		}
	}
	if key == "" {
		t.Skip("no plain brain key")
	}
	if _, err := tgt.Change("", key, "x"); err == nil || !strings.Contains(err.Error(), "area.scope") {
		t.Fatalf("err = %v, want a hint to set area.scope", err)
	}
	if _, err := tgt.Change("[commit\n", key, "x"); err == nil || !strings.Contains(err.Error(), "not valid TOML") {
		t.Fatalf("err = %v, want the parse named", err)
	}
}

func TestValidatedHoldsTheGlobalFileToItsReaders(t *testing.T) {
	g := Global(t.TempDir())
	for name, text := range map[string]string{
		"model":    "[model\n",
		"endpoint": "[model]\nendpoint = \"http://example.com:1234/v1\"\n",
		"search":   "[search]\nbackbone = \"nope\"\n",
		"policy":   "[[policy.paths.rules]]\nmatch = [\"a\"]\n",
		"guard":    "[guard]\nmode = \"strict\"\n",
	} {
		if _, err := g.Validated(text); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	if next, err := g.Validated("[search]\nbackbone = \"cpu\"\n"); err != nil || next == "" {
		t.Errorf("%q %v", next, err)
	}
	if _, err := Project(t.TempDir()).Validated("[commit\n"); err == nil {
		t.Error("a project file that is no TOML passed")
	}
}
