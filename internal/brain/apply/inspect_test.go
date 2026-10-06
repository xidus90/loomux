package apply

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// isThere is isFile for a test that fails on an error.
func isThere(t *testing.T, path string) bool {
	t.Helper()
	there, err := isFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return there
}

// A path the system refuses to inspect is no absent file: a decision that
// takes it for one goes on over something it did not look at. The tests below
// make one path unstatable and ask each reader of isFile to stop at it.

func TestIsFileTellsAnAbsenceFromAPathThatCannotBeInspected(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.md")
	writeFile(t, file, "x")
	for _, row := range []struct {
		path string
		want bool
	}{{file, true}, {dir, false}, {filepath.Join(dir, "fehlt.md"), false}} {
		if got, err := isFile(row.path); got != row.want || err != nil {
			t.Errorf("isFile(%q) = %v, %v; want %v, nil", row.path, got, err, row.want)
		}
	}
	// A NUL byte is refused the same way on every platform, and is no
	// fs.ErrNotExist.
	path := filepath.Join(dir, "a\x00b.md")
	got, err := isFile(path)
	if got || err == nil || errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), "cannot be inspected") {
		t.Fatalf("isFile(%q) = %v, %v; want false and the inspection error", path, got, err)
	}
}

// A register that cannot be inspected, skipped, would leave its doc_ids
// unknown to the source guard.
func TestResolveSourcesStopsAtARegisterThatCannotBeInspected(t *testing.T) {
	areas := []config.Area{{Scope: "one", Path: t.TempDir() + "\x00x"}}
	got, err := resolveSources(areas, config.ArtifactLookup{}, []string{"A"})
	if err == nil || got != nil || !strings.Contains(err.Error(), "cannot be inspected") {
		t.Fatalf("resolveSources = %+v, %v; want the inspection error", got, err)
	}
}

func TestResolveStopsAtABundleSchemaThatCannotBeInspected(t *testing.T) {
	_, casePath, areas := newVault(t)
	areas[0].WikiPath = filepath.Join(t.TempDir(), "wiki")
	denied := errors.New("access denied")
	seam(t, &statPath, func(path string) (fs.FileInfo, error) {
		if filepath.Base(path) == "_schema.md" {
			return nil, &fs.PathError{Op: "stat", Path: path, Err: denied}
		}
		return os.Stat(path)
	})
	_, err := resolve(casePath, caseOf("knowledge"), areas)
	if !errors.Is(err, denied) || strings.Contains(err.Error(), "not a scaffolded bundle") {
		t.Fatalf("resolve = %v; want the inspection error, not the refusal", err)
	}
}

func TestTargetPathStopsAtAPageThatCannotBeInspected(t *testing.T) {
	r := resolvedVault(t)
	denied := unstatable(t, filepath.Join(r.wiki, "topics", "thema.md"))
	_, err := targetPath(r, "topics/thema.md")
	if !errors.Is(err, denied) || strings.Contains(err.Error(), "is gone") {
		t.Fatalf("targetPath = %v; want the inspection error, not \"gone\"", err)
	}
}

// The source guard is skipped for a source that cannot be looked at, and an
// approval would certify a source that changed after the case was formed.
func TestTheSourceGuardStopsAtASourceThatCannotBeInspected(t *testing.T) {
	v := newAppVault(t)
	before := readFile(t, v.page())
	denied := unstatable(t, v.source())
	_, err := v.run()
	stopped[*ApplyError](t, err, "cannot be inspected")
	if !errors.Is(err, denied) || readFile(t, v.page()) != before {
		t.Fatalf("err = %v, or the page was written", err)
	}
}

func TestApproveStopsAtAProposalThatCannotBeInspected(t *testing.T) {
	v := newAppVault(t)
	denied := unstatable(t, v.proposal())
	_, err := v.run()
	stopped[*ApplyError](t, err, "cannot be inspected")
	if !errors.Is(err, denied) || strings.Contains(err.Error(), "no proposal to approve") {
		t.Fatalf("err = %v; want the inspection error, not \"no proposal\"", err)
	}
}

func TestSameFileStopsAtAPathThatCannotBeInspected(t *testing.T) {
	dir := t.TempDir()
	one, other := filepath.Join(dir, "a.md"), filepath.Join(dir, "b.md")
	writeFile(t, one, "x")
	writeFile(t, other, "x")
	for _, path := range []string{one, other} {
		denied := unstatable(t, path)
		if same, err := sameFile(one, other); same || !errors.Is(err, denied) {
			t.Errorf("sameFile with %q unstatable = %v, %v; want false and the inspection error", path, same, err)
		}
	}
}

// A rejection that cannot tell whether the page is there would otherwise
// advance the registers and close the case without the page's sources.
func TestRejectStopsAtAPageThatCannotBeInspected(t *testing.T) {
	j := newRejection(t)
	page := filepath.Join(j.r.wiki, "topics", "thema.md")
	register := withSource(t, &j)
	denied := unstatable(t, page)
	_, err := j.run(t)
	if !errors.Is(err, denied) {
		t.Fatalf("reject = %v; want the inspection error", err)
	}
	if readFile(t, register) != rejectRegister || !isThere(t, filepath.Join(j.r.directory, "case.toml")) {
		t.Fatal("the register moved or the case went although nothing was decided")
	}
}

func TestClaimHeadingsStopAtAProposalThatCannotBeInspected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proposal.md")
	denied := unstatable(t, path)
	if claims, err := claimHeadings(path); claims != nil || !errors.Is(err, denied) {
		t.Fatalf("claimHeadings = %v, %v; want the inspection error", claims, err)
	}
}

func TestAProtocolThatCannotBeInspectedIsNotReplacedByAFreshOne(t *testing.T) {
	p, vault := newPlace(t)
	path := filepath.Join(vault, "wiki", "audit.md")
	writeFile(t, path, "bisher\n")
	denied := unstatable(t, path)
	if err := p.appendProtocol(path, "neu\n"); !errors.Is(err, denied) {
		t.Fatalf("appendProtocol = %v; want the inspection error", err)
	}
	if got := readFile(t, path); got != "bisher\n" {
		t.Fatalf("audit.md = %q, want it untouched", got)
	}
}

// A path through a regular file is a file that is not there, as on Windows:
// Linux answers ENOTDIR for it.
func TestIsFileTakesAPathThroughARegularFileForAbsent(t *testing.T) {
	file := filepath.Join(t.TempDir(), "a.md")
	writeFile(t, file, "x")
	if got, err := isFile(filepath.Join(file, "b.md")); got || err != nil {
		t.Fatalf("isFile = %v, %v; want false, nil", got, err)
	}
}
