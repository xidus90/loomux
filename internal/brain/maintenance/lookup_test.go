package maintenance_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/brain/maintenance"
)

// caseAt lays one empty case file at a slash-separated place beneath root and
// answers its directory. Empty is enough: neither the walk nor the lookup
// reads what a case says.
func caseAt(t *testing.T, root, relative string) string {
	t.Helper()
	writeUnder(t, root, relative+"/case.toml", "")
	return filepath.Join(root, filepath.FromSlash(relative))
}

// The order is Python's `sorted(rglob)` on Windows: the parts of the path
// compared one by one, each in lower case. A byte-order walk puts `Zeta-case`
// first; a sort over the joined strings puts `alpha-later` before
// `alpha/...`, because the dash sorts before either separator.
func TestCaseFilesSortsByLowerCasedParts(t *testing.T) {
	root := t.TempDir()
	zeta := caseAt(t, root, "project-a/Zeta-case")
	alpha := caseAt(t, root, "project-a/alpha-case")
	nested := caseAt(t, root, "project-b/alpha/topic-1")
	later := caseAt(t, root, "project-b/alpha-later")

	got := maintenance.CaseFiles(root)
	want := []string{
		filepath.Join(alpha, "case.toml"),
		filepath.Join(zeta, "case.toml"),
		filepath.Join(nested, "case.toml"),
		filepath.Join(later, "case.toml"),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("CaseFiles = %q, want %q", got, want)
	}
}

// A directory that happens to be called `case.toml` holds no case, and a
// review centre that does not exist yet holds none either.
func TestCaseFilesSkipsDirectoriesAndAMissingRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "x", "case.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := maintenance.CaseFiles(root); len(got) != 0 {
		t.Fatalf("CaseFiles = %q, want none", got)
	}
	if got := maintenance.CaseFiles(filepath.Join(root, "missing")); len(got) != 0 {
		t.Fatalf("CaseFiles of a missing root = %q, want none", got)
	}
}

func TestFindCaseFindsTheDirectoryAtAnyDepth(t *testing.T) {
	root := t.TempDir()
	caseAt(t, root, "project-a/other")
	want := caseAt(t, root, "project-a/moved/wanted")
	got, err := maintenance.FindCase(root, "wanted")
	if err != nil {
		t.Fatalf("FindCase: %v", err)
	}
	if got != want {
		t.Fatalf("FindCase = %q, want %q", got, want)
	}
}

func TestFindCaseRefusesWithoutAReviewCentre(t *testing.T) {
	root := filepath.Join(t.TempDir(), "95 Prüfzentrum")
	_, err := maintenance.FindCase(root, "x")
	want := "no case named 'x'; there is no review centre at " + root + " yet"
	if err == nil || err.Error() != want {
		t.Fatalf("FindCase error = %v, want %q", err, want)
	}
}

// A file where the review centre should be is no review centre either:
// `Path.is_dir()`, not `exists()`.
func TestFindCaseRefusesAFileAsTheReviewCentre(t *testing.T) {
	root := filepath.Join(t.TempDir(), "review")
	if err := os.WriteFile(root, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := maintenance.FindCase(root, "x"); err == nil {
		t.Fatal("FindCase accepted a file as the review centre")
	}
}

// The id is compared as a name, never globbed, and quoted the way Python's
// `!r` quotes it.
func TestFindCaseComparesNamesAndQuotesLikeRepr(t *testing.T) {
	root := t.TempDir()
	caseAt(t, root, "project-a/topic-1")
	for identifier, quoted := range map[string]string{
		"topic-*": "'topic-*'",
		"it's":    `"it's"`,
		"..":      "'..'",
		"Topic-1": "'Topic-1'",
	} {
		_, err := maintenance.FindCase(root, identifier)
		want := "no case named " + quoted + " in the review centre at " + root
		if err == nil || err.Error() != want {
			t.Fatalf("FindCase(%q) error = %v, want %q", identifier, err, want)
		}
	}
}

// Two directories carrying one id are refused, named in the order the
// listing shows them.
func TestFindCaseRefusesAnAmbiguousID(t *testing.T) {
	root := t.TempDir()
	second := caseAt(t, root, "Zeta/dup")
	first := caseAt(t, root, "alpha/dup")
	_, err := maintenance.FindCase(root, "dup")
	want := "no case named 'dup' unambiguously; " + first + " and " + second + " both carry it"
	if err == nil || err.Error() != want {
		t.Fatalf("FindCase error = %v, want %q", err, want)
	}
}

func TestAreaManifestAnswersTheDeclaredArea(t *testing.T) {
	w := newWorld(t)
	w.addArea(t, areaOptions{Scope: "project/a", PrivacyMode: "local_only"})
	manifest := maintenance.AreaManifest(w.Areas, w.Lookup(), "project/a")
	if manifest == nil || manifest.PrivacyMode != "local_only" {
		t.Fatalf("AreaManifest = %+v, want the local_only declaration", manifest)
	}
}

// An unregistered area and one without a declaration both answer nothing,
// which the caller reads as closed.
func TestAreaManifestAnswersNothingForAnUndeclaredArea(t *testing.T) {
	w := newWorld(t)
	w.addArea(t, areaOptions{Scope: "project/a", NoManifest: true})
	for _, scope := range []string{"project/a", "project/unknown"} {
		if manifest := maintenance.AreaManifest(w.Areas, w.Lookup(), scope); manifest != nil {
			t.Fatalf("AreaManifest(%q) = %+v, want nil", scope, manifest)
		}
	}
}

// A declaration that does not read closes the area too, rather than opening
// it for want of an answer.
func TestAreaManifestAnswersNothingForABrokenDeclaration(t *testing.T) {
	w := newWorld(t)
	w.addArea(t, areaOptions{Scope: "project/a", BrokenManifest: true})
	if manifest := maintenance.AreaManifest(w.Areas, w.Lookup(), "project/a"); manifest != nil {
		t.Fatalf("AreaManifest = %+v, want nil for a broken declaration", manifest)
	}
}
