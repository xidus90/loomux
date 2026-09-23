package run

import (
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// dangling is an area whose registration names a wiki that is not there --
// the shape the registry is in right now for `project/ultra-brain`, whose
// wiki path points into a checkout that has no `docs/wiki` on its branch.
func dangling(t *testing.T) config.Area {
	t.Helper()
	base := t.TempDir()
	return config.Area{Scope: "project/p", Path: base,
		WikiPath: filepath.Join(base, "gone")}
}

func TestTargetsRefusesANamedAreaWhoseWikiIsNotThere(t *testing.T) {
	a := dangling(t)
	_, err := Targets([]config.Area{a}, "project/p")
	if err == nil {
		t.Fatal("no error for a named area whose wiki is not there")
	}
	want := "area \"project/p\" has no wiki at " + a.WikiPath +
		"; run `loomux wiki init --scope project/p` first"
	if err.Error() != want {
		t.Errorf("err = %q, want %q", err, want)
	}
}

func TestTargetsRefusesANamedAreaThatDeclaresNoWiki(t *testing.T) {
	a := config.Area{Scope: "project/p", Path: t.TempDir()}
	_, err := Targets([]config.Area{a}, "project/p")
	if err == nil {
		t.Fatal("no error for a named area that declares no wiki")
	}
	want := "area \"project/p\" declares no wiki path; " +
		"add `wiki = ...` to its entry"
	if err.Error() != want {
		t.Errorf("err = %q, want %q", err, want)
	}
}

func TestTargetsRefusesAnUnregisteredScope(t *testing.T) {
	_, err := Targets(nil, "project/nope")
	want := "no area named \"project/nope\" in the registry"
	if err == nil || err.Error() != want {
		t.Errorf("err = %v, want %q", err, want)
	}
}

func TestTargetsRefusesASweepWhoseAreaWikiIsNotThere(t *testing.T) {
	// Python's `_existing` guards the sweep as well as the named path
	// (`src/brain/cli.py:1487-1490`): a dangling wiki yields no pages,
	// and every rule would read that silence as a clean bundle.
	a := dangling(t)
	if _, err := Targets([]config.Area{a}, "all"); err == nil {
		t.Fatal("no error for a sweep over an area whose wiki is not there")
	}
}

func TestTargetsSkipsAnAreaThatDeclaresNoWikiInTheSweep(t *testing.T) {
	// The asymmetry is Python's and deliberate: naming an area without a
	// wiki is a mistake worth a message, skipping one in the sweep is
	// not -- most areas will never carry a bundle
	// (`src/brain/cli.py:1469-1471`).
	a := config.Area{Scope: "project/p", Path: t.TempDir()}
	got, err := Targets([]config.Area{a}, "all")
	if err != nil {
		t.Fatalf("err = %v, want none", err)
	}
	if len(got) != 0 {
		t.Errorf("got %+v, want no targets", got)
	}
}

func TestTargetsRefusesAWikiThatIsAFile(t *testing.T) {
	// The input-space probe beside the mutants: `is_dir()` refuses a
	// plain file too, and a walk over one yields the same silence a
	// missing directory does.
	base := t.TempDir()
	wiki := filepath.Join(base, "wiki")
	write(t, wiki, "not a directory\n")
	a := config.Area{Scope: "project/p", Path: base, WikiPath: wiki}
	if _, err := Targets([]config.Area{a}, "project/p"); err == nil {
		t.Fatal("no error for a wiki path that is a file")
	}
}

func TestTargetsAnswersTheAreasItAccepts(t *testing.T) {
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	a := area(t, base, "project/p")
	got, err := Targets([]config.Area{a}, "project/p")
	if err != nil {
		t.Fatalf("err = %v, want none", err)
	}
	if len(got) != 1 || got[0].Scope != "project/p" {
		t.Errorf("got %+v, want the one named area", got)
	}
	if got, err = Targets([]config.Area{a}, "all"); err != nil ||
		len(got) != 1 {
		t.Errorf("sweep gave %+v, %v; want the one area", got, err)
	}
}

func TestTargetsFindsANamedAreaBehindAnother(t *testing.T) {
	// The named lookup walks past the entries it was not asked about.
	// Without a second area in front, the skip is never taken and a
	// `Targets` that answered the *first* entry whatever its scope
	// would pass every other test in this file.
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	first := area(t, filepath.Join(base, "one"), "project/first")
	second := area(t, filepath.Join(base, "two"), "project/second")
	got, err := Targets([]config.Area{first, second}, "project/second")
	if err != nil {
		t.Fatalf("err = %v, want none", err)
	}
	if len(got) != 1 || got[0].Scope != "project/second" {
		t.Errorf("got %+v, want the second area", got)
	}
}
