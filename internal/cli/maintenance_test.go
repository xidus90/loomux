package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/config"
)

// reconcileOptions is everything a case here varies about the one machine it
// stands on. The zero value is the ordinary pass: one registered source, no
// page deriving from it, a declared review centre and the manifest's own
// privacy mode.
type reconcileOptions struct {
	// Sources is how many registered sources the area holds. 0 means one.
	Sources int

	// Cite writes one wiki page deriving from every source, which is what
	// turns a changed source into a case. Without it the area has no wiki at
	// all and a pass raises nothing.
	Cite bool

	// NoReview leaves `[layout] review` undeclared -- the state
	// ErrNoReviewCentre is about.
	NoReview bool

	// PrivacyMode is `[privacy] mode`. Left out, the manifest's default
	// applies, which is `manual_cloud`.
	PrivacyMode string
}

// reconcileWorld is the machine a reconcile pass runs on: a state directory
// with a registry, one registered area that declares its wiki and its review
// centre, and a register whose digests no longer describe the files -- which
// is what makes a pass see a change at all.
type reconcileWorld struct {
	State  string
	Area   string
	Review string
}

func newReconcileWorld(t *testing.T, opts reconcileOptions) reconcileWorld {
	t.Helper()
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	area := filepath.Join(tmp, "area")
	t.Setenv("LOOMUX_STATE_DIR", state)
	// Without this a case would rewrite the real qmd configuration of whoever
	// runs the suite -- a trap this suite has walked into once.
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	sources := opts.Sources
	if sources == 0 {
		sources = 1
	}
	identities := map[string]identity.Identity{}
	var cites strings.Builder
	for i := 1; i <= sources; i++ {
		relative := fmt.Sprintf("src/note%d.md", i)
		full := filepath.Join(area, filepath.FromSlash(relative))
		writeFile(t, full, "# Note\n\nWhat it said before.\n")
		digest, err := identity.ContentHash(full)
		if err != nil {
			t.Fatalf("ContentHash: %v", err)
		}
		// Rewritten longer than what was hashed: the stat cache compares
		// modification time and size, and a rewrite of the same length within
		// the clock's resolution could hide behind it.
		writeFile(t, full, "# Note\n\nWhat it says now, at a length of its own.\n")
		docID := fmt.Sprintf("%026d", i)
		identities[relative] = identity.Identity{
			DocID: docID, Relative: relative, ContentHash: digest, Revision: 1,
		}
		cites.WriteString("  - resource: " + relative + "\n    doc_id: " + docID + "\n")
	}
	// The register lands where a writable area keeps its artefacts, which is
	// the area itself.
	writeFile(t, filepath.Join(area, "_identities.tsv"), identity.RenderIdentities(identities))

	if opts.Cite {
		writeFile(t, filepath.Join(area, "wiki", "page.md"),
			"---\ntype: note\ntitle: Page\nsources:\n"+cites.String()+"---\n\n# Page\n\nWhat the sources say.\n")
	}

	writeFile(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"project/a\"\npath = "+strconv.Quote(filepath.ToSlash(area))+
			"\nwiki = "+strconv.Quote(filepath.ToSlash(filepath.Join(area, "wiki")))+"\n")

	declaration := "[area]\nscope = \"project/a\"\n\n[layout]\nwiki = \"wiki\"\n"
	if !opts.NoReview {
		declaration += "review = \"review\"\n"
	}
	if opts.PrivacyMode != "" {
		declaration += "\n[privacy]\nmode = " + config.QuoteTOML(opts.PrivacyMode) + "\n"
	}
	writeFile(t, filepath.Join(area, ".loomux", "config.toml"),
		declaration+"\n[index]\ninclude = [\"**/*.md\"]\n")

	return reconcileWorld{State: state, Area: area, Review: filepath.Join(area, "review")}
}

// caseDir is the one case directory the pass landed, so a case can look at
// what stands on disk without knowing the id the run computed.
func (w reconcileWorld) caseDir(t *testing.T) string {
	t.Helper()
	scope := filepath.Join(w.Review, "project-a")
	entries, err := os.ReadDir(scope)
	if err != nil {
		t.Fatalf("ReadDir %s: %v", scope, err)
	}
	if len(entries) != 1 {
		t.Fatalf("%s holds %d entries, want 1", scope, len(entries))
	}
	return filepath.Join(scope, entries[0].Name())
}

// A case is the result of the command, not a failure of it: the pass ran to
// the end, and the reviewer is owed the queue it produced.
func TestReconcileExitsZeroWithCases(t *testing.T) {
	world := newReconcileWorld(t, reconcileOptions{Cite: true})
	code, out, errOut := run("reconcile")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(out, "1 Quelle geprüft, 1 davon gehasht") {
		t.Fatalf("stdout does not carry the count: %q", out)
	}
	if !strings.Contains(out, "\n1 Fall\n") {
		t.Fatalf("stdout does not carry the case count: %q", out)
	}
	name := filepath.Base(world.caseDir(t))
	if !strings.Contains(out, "  "+name+"\tproject/a\tpage.md\tsource_changed\n") {
		t.Fatalf("stdout does not name the case and its place: %q", out)
	}
}

// Nothing derives from the changed sources, so nothing is under review -- and
// the counts read as German rather than as a defect in the count.
func TestReconcileCountsSeveralSourcesAndNoCases(t *testing.T) {
	newReconcileWorld(t, reconcileOptions{Sources: 2})
	code, out, errOut := run("reconcile")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(out, "2 Quellen geprüft, 2 davon gehasht") {
		t.Fatalf("stdout does not carry the count: %q", out)
	}
	if !strings.Contains(out, "\n0 Fälle\n") {
		t.Fatalf("stdout does not carry the case count: %q", out)
	}
}

// A vault that declares no review centre has nowhere to put a case, and the
// reference makes no exception of that failure: one `error:` line, exit 1.
func TestReconcileWithoutAReviewCentreIsRed(t *testing.T) {
	newReconcileWorld(t, reconcileOptions{Cite: true, NoReview: true})
	code, _, errOut := run("reconcile")
	if code != 1 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(errOut, "[layout]") || !strings.Contains(errOut, "review") {
		t.Fatalf("stderr does not name the missing declaration: %q", errOut)
	}
}

// A broken case file is the one finding that scores the run: reconcile stepped
// over it and opened a second case beside it, so two entries now stand for one
// page and only a person can settle which of them counts.
func TestReconcileReportsUnreadableCaseFiles(t *testing.T) {
	world := newReconcileWorld(t, reconcileOptions{Cite: true})
	broken := filepath.Join(world.Review, "project-a", "handmade", "case.toml")
	writeFile(t, broken, "this is not TOML {{{\n")
	code, out, errOut := run("reconcile")
	if code != 1 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(errOut, "unreadable case: "+broken) {
		t.Fatalf("stderr does not name the broken case: %q", errOut)
	}
	if !strings.Contains(out, "1 Quelle geprüft") {
		t.Fatalf("stdout lost the report over the broken case: %q", out)
	}
}

// A file lying in the review centre is not a scope directory and carries no
// case. The listing steps over it rather than losing the addresses of every
// case that does stand there.
func TestReconcileStepsOverAStrayFileInTheReviewCentre(t *testing.T) {
	world := newReconcileWorld(t, reconcileOptions{Cite: true})
	writeFile(t, filepath.Join(world.Review, "read-me.txt"), "a note a reviewer left\n")
	code, out, errOut := run("reconcile")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	name := filepath.Base(world.caseDir(t))
	if !strings.Contains(out, "  "+name+"\t") {
		t.Fatalf("stdout does not name the case: %q", out)
	}
}

// A closed area is decided by hand: stage 3a asks no local model, so the case
// carries no proposal and says so in the listing.
func TestReconcileMarksAClosedAreasCaseManual(t *testing.T) {
	newReconcileWorld(t, reconcileOptions{Cite: true, PrivacyMode: "local_only"})
	code, out, errOut := run("reconcile")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(out, "\tsource_changed\tmanuell\n") {
		t.Fatalf("stdout does not mark the case manual: %q", out)
	}
}

// `case.toml` is synchronised vault content and gets hand-edited. The
// directory name is what addresses a case, so the listing prints that and not
// the field, which may have parted ways with it.
func TestReconcileNamesACaseByItsDirectoryAndNotByItsField(t *testing.T) {
	world := newReconcileWorld(t, reconcileOptions{Cite: true})
	if code, _, errOut := run("reconcile"); code != 0 {
		t.Fatalf("first pass: exit = %d, stderr = %s", code, errOut)
	}
	directory := world.caseDir(t)
	file := filepath.Join(directory, "case.toml")
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	name := filepath.Base(directory)
	edited := strings.Replace(string(raw), strconv.Quote(name), strconv.Quote("renamed-by-hand"), 1)
	if edited == string(raw) {
		t.Fatalf("case.toml does not carry its id %q: %s", name, raw)
	}
	writeFile(t, file, edited)

	code, out, errOut := run("reconcile")
	if code != 0 {
		t.Fatalf("second pass: exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(out, "  "+name+"\t") {
		t.Fatalf("stdout does not address the case by its directory: %q", out)
	}
	if strings.Contains(out, "renamed-by-hand") {
		t.Fatalf("stdout prints the hand-edited field: %q", out)
	}
}

// The state model of this stage is the environment, and no command of it takes
// a directory on the command line.
func TestReconcileRefusesAnUnknownFlag(t *testing.T) {
	newReconcileWorld(t, reconcileOptions{})
	if code, _, _ := run("reconcile", "--state-dir", "somewhere"); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

// A registry that cannot be read leaves nothing to reconcile.
func TestReconcileIsRedOnABrokenRegistry(t *testing.T) {
	world := newReconcileWorld(t, reconcileOptions{})
	writeFile(t, filepath.Join(world.State, "registry.toml"), "this is not TOML {{{\n")
	code, _, errOut := run("reconcile")
	if code != 1 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(errOut, "error:") {
		t.Fatalf("stderr does not carry the error: %q", errOut)
	}
}

// A register that cannot be read is not an area without sources: reading it as
// empty would report a clean area, which is the one answer that must not be
// guessed.
func TestReconcileIsRedWhenThePassBreaks(t *testing.T) {
	world := newReconcileWorld(t, reconcileOptions{Cite: true})
	register := filepath.Join(world.Area, "_identities.tsv")
	if err := os.Remove(register); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := os.MkdirAll(register, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	code, _, errOut := run("reconcile")
	if code != 1 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(errOut, "error:") {
		t.Fatalf("stderr does not carry the error: %q", errOut)
	}
}

// The stamps of a pass are UTC, whatever zone the machine keeps. Nothing
// downstream folds the zone away: `CaseID` takes the date into the directory
// name and `pytext.IsoFormat` prints the offset it is handed, so a local clock
// would name a case differently from the reference for two hours of every day.
//
// `time.Local` is swapped rather than a clock seam added: `time.Now` reads
// that variable at the moment of the call, so the swap reaches the command
// without the command growing a parameter that only a test would use.
func TestReconcileStampsACaseInUTC(t *testing.T) {
	world := newReconcileWorld(t, reconcileOptions{Cite: true})
	saved := time.Local
	time.Local = time.FixedZone("two hours east", 2*60*60)
	t.Cleanup(func() { time.Local = saved })

	if code, _, errOut := run("reconcile"); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	raw, err := os.ReadFile(filepath.Join(world.caseDir(t), "case.toml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	created := ""
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "created = ") {
			created = line
		}
	}
	if !strings.HasSuffix(created, "+00:00") {
		t.Fatalf("created is not stamped in UTC: %q", created)
	}
}
