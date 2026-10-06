package apply

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
)

// The write barrier, ported from `_gate`, `_write`, `_record_case`,
// `_remove`, `_touch` and `_preflight` (apply.py:467-633). Where a test
// carries over one of test_apply.py, its comment names it and its line.

// newPlace builds a vault with a wiki and a review centre below it, the
// shape `_resolve` hands `_Place`, and returns the place and the vault.
func newPlace(t *testing.T) (*place, string) {
	t.Helper()
	vault := t.TempDir()
	for _, dir := range []string{"wiki", filepath.Join("review", "c1")} {
		if err := os.MkdirAll(filepath.Join(vault, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return &place{anchor: vault, wiki: filepath.Join(vault, "wiki")}, vault
}

// seam swaps one of the barrier's function variables for the test's
// duration, the way the Python tests monkeypatch `_resolved` and `_is_link`.
func seam[T any](t *testing.T, slot *T, value T) {
	t.Helper()
	saved := *slot
	*slot = value
	t.Cleanup(func() { *slot = saved })
}

// linkNamed makes isLink answer true for every path whose last name is name.
func linkNamed(t *testing.T, name string) {
	t.Helper()
	seam(t, &isLink, func(path string) bool { return filepath.Base(path) == name })
}

// refused asserts a gate refusal carrying phrase, and that it is one: the
// caller turns a refusal into its own error kind and lets an OS error pass.
func refused(t *testing.T, err error, phrase string) {
	t.Helper()
	var refusal *gateError
	if !errors.As(err, &refusal) {
		t.Fatalf("want a gate refusal containing %q, got %v", phrase, err)
	}
	if !strings.Contains(err.Error(), phrase) {
		t.Fatalf("want %q in the refusal, got %q", phrase, err.Error())
	}
}

func absent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s is on disk: %v", path, err)
	}
}

func writeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// test_a_path_outside_the_vault_never_reaches_the_disk (test_apply.py:1498).
func TestGateRefusesAPathOutsideTheVault(t *testing.T) {
	p, vault := newPlace(t)
	outside := filepath.Join(t.TempDir(), "fremd.md")
	err := p.write(outside, "x")
	refused(t, err, outside+": not inside the vault "+vault)
	absent(t, outside)
	if len(p.touched) != 0 {
		t.Fatalf("a refused write was recorded: %v", p.touched)
	}
}

// test_the_vault_root_is_not_a_write_target (test_apply.py:1505).
func TestGateRefusesTheVaultRootItself(t *testing.T) {
	p, vault := newPlace(t)
	refused(t, p.remove(vault), vault+": the vault root is not a write target")
	if _, err := os.Stat(vault); err != nil {
		t.Fatalf("the vault is gone: %v", err)
	}
}

// test_a_declared_scaffold_write_must_name_a_scaffold_file (test_apply.py:1512).
func TestGateADeclaredScaffoldWriteMustNameAScaffoldFile(t *testing.T) {
	p, vault := newPlace(t)
	page := filepath.Join(vault, "wiki", "thema.md")
	refused(t, p.writeScaffold(page, "x"),
		page+": declared a scaffold write, but thema.md is not a scaffold file")
	absent(t, page)
}

// test_the_gate_refuses_a_scaffold_path_a_call_site_forgot_to_declare
// (test_apply.py:1520), and the fix-round-4 target tests one level down
// (test_apply.py:1223-1253): case, register, subdirectory. Then the aliases
// of fix round 5 (test_apply.py:1450-1475): trailing dot, stream, the
// register's trailing dot -- plus a trailing blank and a long s, which
// `casefold` folds onto `s` and which `_normalised` therefore refuses too.
func TestGateRefusesEveryScaffoldSpelling(t *testing.T) {
	for _, name := range []string{
		"audit.md",
		"Audit.md",
		"_identities.tsv",
		filepath.Join("topics", "log.md"),
		filepath.Join("log.md", "seite.md"),
		"audit.md.",
		"audit.md:zone",
		"_identities.tsv.",
		"index.md ",
		"_schema.md. .",
		"_identitieſ.tsv",
		"LOG.MD:$DATA",
	} {
		t.Run(name, func(t *testing.T) {
			p, vault := newPlace(t)
			path := filepath.Join(vault, "wiki", name)
			refused(t, p.write(path, "PWN"), path+": a scaffold file, not a page a case may change")
			if len(p.touched) != 0 {
				t.Fatalf("a refused write was recorded: %v", p.touched)
			}
		})
	}
}

// A name that only resembles a scaffold file is a page like any other.
func TestGateLetsANameThatOnlyResemblesAScaffoldFilePass(t *testing.T) {
	for _, name := range []string{"audit.md.bak", "my-log.md", "index.markdown", "a.md"} {
		t.Run(name, func(t *testing.T) {
			p, vault := newPlace(t)
			if err := p.write(filepath.Join(vault, "wiki", name), "x"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// test_a_short_name_alias_of_the_register_is_refused (test_apply.py:1532):
// the alias is modelled through the resolver, so the rule is measured on
// every platform; place_windows_test.go measures a real one.
func TestGateRefusesAShortNameThatResolvesToTheRegister(t *testing.T) {
	p, vault := newPlace(t)
	short := filepath.Join(vault, "_IDENT~1.TSV")
	real := resolvePath
	seam(t, &resolvePath, func(path string) (string, error) {
		if filepath.Base(path) == "_IDENT~1.TSV" {
			return filepath.Join(vault, "_identities.tsv"), nil
		}
		return real(path)
	})
	refused(t, p.write(short, "PWN"), short+": a scaffold file, not a page a case may change")
	absent(t, short)
}

// test_a_link_on_the_way_to_the_target_is_refused (test_apply.py:1197).
func TestGateRefusesALinkOnTheWay(t *testing.T) {
	p, vault := newPlace(t)
	topics := filepath.Join(vault, "wiki", "topics")
	linkNamed(t, "topics")
	page := filepath.Join(topics, "thema.md")
	refused(t, p.write(page, "x"), topics+": topics is a link, not a directory")
	absent(t, page)
}

// test_a_link_at_a_write_target_is_refused (test_apply.py:1256), through
// the preflight that finds it before anything is written: a link at any of
// the four fixed write targets, or at the wiki above two of them.
func TestPreflightRefusesALinkAtAFixedWriteTarget(t *testing.T) {
	for _, name := range []string{"log.md", "audit.md", "_identities.tsv", "wiki", "c1"} {
		t.Run(name, func(t *testing.T) {
			p, vault := newPlace(t)
			linkNamed(t, name)
			refused(t, p.preflight(filepath.Join(vault, "review", "c1")), "is a link, not a directory")
		})
	}
}

// test_a_linked_audit_stops_before_the_page_and_the_register
// (test_apply.py:1558): the preflight answers before a single byte is
// written, and a clean preflight writes nothing either.
func TestPreflightChecksAllFourPathsAndWritesNothing(t *testing.T) {
	p, vault := newPlace(t)
	caseDir := filepath.Join(vault, "review", "c1")
	var gated []string
	real := resolvePath
	seam(t, &resolvePath, func(path string) (string, error) {
		gated = append(gated, path)
		return real(path)
	})
	if err := p.preflight(caseDir); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		filepath.Join(vault, "wiki", "log.md"),
		filepath.Join(vault, "wiki", "audit.md"),
		filepath.Join(vault, "_identities.tsv"),
		caseDir,
	} {
		if !slices.Contains(gated, want) {
			t.Errorf("preflight never walked %s", want)
		}
	}
	for _, name := range []string{"log.md", "audit.md"} {
		absent(t, filepath.Join(vault, "wiki", name))
	}
	absent(t, filepath.Join(vault, "_identities.tsv"))
	if len(p.touched) != 0 {
		t.Fatalf("preflight recorded a write: %v", p.touched)
	}
}

// The register beside the vault root is a scaffold write, the case
// directory is not: a preflight with the two swapped would pass a case
// directory named after a scaffold file.
func TestPreflightRefusesACaseDirectoryWithAScaffoldName(t *testing.T) {
	p, vault := newPlace(t)
	caseDir := filepath.Join(vault, "review", "audit.md")
	refused(t, p.preflight(caseDir), caseDir+": a scaffold file, not a page a case may change")
}

// test_the_case_directory_is_not_removed_from_outside_the_vault
// (test_apply.py:1477): a component that resolves out of the vault is
// refused although it is no link.
func TestGateRefusesAComponentThatResolvesOutOfTheVault(t *testing.T) {
	p, vault := newPlace(t)
	outside := filepath.Join(t.TempDir(), "anderswo", "review")
	real := resolvePath
	seam(t, &resolvePath, func(path string) (string, error) {
		if filepath.Base(path) == "review" {
			return outside, nil
		}
		return real(path)
	})
	caseDir := filepath.Join(vault, "review", "c1")
	refused(t, p.remove(caseDir), filepath.Join(vault, "review")+": resolves to a path not inside the vault")
	if _, err := os.Stat(caseDir); err != nil {
		t.Fatalf("the case directory is gone: %v", err)
	}
}

// junction makes link a directory junction to target on Windows and a
// symlink elsewhere, as the brief asks. A junction is what `is_symlink`
// misses, so it is the fixture the containment arm exists for.
func junction(t *testing.T, target, link string) {
	t.Helper()
	if runtime.GOOS != "windows" {
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("this machine does not let the test make a symlink: %v", err)
		}
		return
	}
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Skipf("mklink /J failed: %v (%s)", err, out)
	}
}

// A real junction out of the vault: `is_symlink` is false for it, so the
// containment arm refuses it; a symlink elsewhere takes the link arm first.
// Either way nothing lands outside.
func TestGateRefusesARealJunctionOutOfTheVault(t *testing.T) {
	p, vault := newPlace(t)
	outside := t.TempDir()
	linked := filepath.Join(vault, "wiki", "topics")
	junction(t, outside, linked)
	page := filepath.Join(linked, "thema.md")
	want := linked + ": resolves to a path not inside the vault"
	if runtime.GOOS != "windows" {
		want = linked + ": topics is a link, not a directory"
	}
	refused(t, p.write(page, "x"), want)
	absent(t, filepath.Join(outside, "thema.md"))
}

// A junction that points back inside the vault stays uncaught, and on
// purpose: `_gate`'s docstring names it (apply.py:585-586). Nothing lands
// outside the vault through it.
func TestGateLetsAJunctionInsideTheVaultPass(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("a symlink is a link to `is_symlink`; only a junction reaches this arm")
	}
	p, vault := newPlace(t)
	inside := filepath.Join(vault, "wiki", "real")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(vault, "wiki", "alias")
	junction(t, inside, linked)
	if err := p.write(filepath.Join(linked, "thema.md"), "x"); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(inside, "thema.md")); got != "x" {
		t.Fatalf("the write did not land in the junction's target: %q", got)
	}
}

// A walked `..` is resolved like every other component, so climbing out is
// refused and climbing back in is not.
func TestGateWalksADotDotComponentRatherThanCleaningIt(t *testing.T) {
	p, vault := newPlace(t)
	out := vault + string(filepath.Separator) + ".." + string(filepath.Separator) + "x.md"
	refused(t, p.write(out, "x"), vault+string(filepath.Separator)+".."+": resolves to a path not inside the vault")
	back := filepath.Join(vault, "wiki") + string(filepath.Separator) + ".." +
		string(filepath.Separator) + "wiki" + string(filepath.Separator) + "x.md"
	if err := p.write(back, "x"); err != nil {
		t.Fatal(err)
	}
}

// A resolver that cannot answer -- a cycle of links -- is no refusal of the
// gate's own but a failure the caller reports as it is.
func TestGatePassesOnAResolverFailure(t *testing.T) {
	p, vault := newPlace(t)
	cycle := errors.New("its links lead in a circle")
	seam(t, &resolvePath, func(string) (string, error) { return "", cycle })
	for _, err := range []error{
		p.write(filepath.Join(vault, "wiki", "a.md"), "x"),
		p.writeScaffold(filepath.Join(vault, "_identities.tsv"), "x"),
	} {
		if !errors.Is(err, cycle) {
			t.Fatalf("want the resolver's error, got %v", err)
		}
	}
}

// The resolver may fail on the component walk alone, after the anchor
// resolved; that is passed on as well.
func TestGatePassesOnAResolverFailureMidWalk(t *testing.T) {
	p, vault := newPlace(t)
	cycle := errors.New("its links lead in a circle")
	real := resolvePath
	seam(t, &resolvePath, func(path string) (string, error) {
		if filepath.Base(path) == "loop" {
			return "", cycle
		}
		return real(path)
	})
	if err := p.write(filepath.Join(vault, "wiki", "loop", "a.md"), "x"); !errors.Is(err, cycle) {
		t.Fatalf("want the resolver's error, got %v", err)
	}
	seam(t, &resolvePath, func(path string) (string, error) {
		if path == vault {
			return "", cycle
		}
		return real(path)
	})
	if err := p.write(filepath.Join(vault, "wiki", "a.md"), "x"); !errors.Is(err, cycle) {
		t.Fatalf("want the resolver's error on the anchor, got %v", err)
	}
	seam(t, &resolvePath, func(path string) (string, error) {
		if path == p.wiki {
			return "", cycle
		}
		return real(path)
	})
	if err := p.write(filepath.Join(vault, "wiki", "a.md"), "x"); !errors.Is(err, cycle) {
		t.Fatalf("want the resolver's error on the wiki, got %v", err)
	}
}

// A swap that fails leaves the record standing.
func TestWriteRecordsTheFileWhenTheSwapFails(t *testing.T) {
	p, vault := newPlace(t)
	locked := errors.New("held open")
	seam(t, &replaceText, func(string, string) error { return locked })
	if err := p.write(filepath.Join(vault, "wiki", "thema.md"), "x"); !errors.Is(err, locked) {
		t.Fatalf("want the swap's error, got %v", err)
	}
	if want := []string{"wiki/thema.md"}; !slices.Equal(p.touched, want) {
		t.Fatalf("touched = %v, want %v", p.touched, want)
	}
}

// `_anchor` (apply.py:449-464): a wiki outside the vault, which `_resolve`
// admits only as a bundle, is the tree a write under it is measured
// against. `_relative` then has no vault-relative answer and records the
// path as it is.
func TestGateMeasuresAWriteUnderAnOutsideWikiAgainstThatWiki(t *testing.T) {
	p, _ := newPlace(t)
	p.wiki = t.TempDir()
	page := filepath.Join(p.wiki, "thema.md")
	if err := p.write(page, "x"); err != nil {
		t.Fatal(err)
	}
	if want := []string{filepath.ToSlash(page)}; !slices.Equal(p.touched, want) {
		t.Fatalf("touched = %v, want %v", p.touched, want)
	}
	if err := p.writeScaffold(filepath.Join(p.wiki, "log.md"), "x"); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(filepath.Dir(p.wiki), "daneben.md")
	refused(t, p.write(outside, "x"), "not inside the vault")
}

// test_gate_authorizes_external_registers_and_denies_other_external_writes
// (test_apply.py:1929): a registered area's register outside the vault is
// the one outside path a scaffold write may name.
func TestGateAuthorisesARegisteredExternalRegisterOnly(t *testing.T) {
	p, _ := newPlace(t)
	code := t.TempDir()
	register := filepath.Join(code, "_identities.tsv")
	p.registers = []string{register}

	if err := p.writeScaffold(register, "header\n"); err != nil {
		t.Fatalf("a registered external register was refused: %v", err)
	}
	if got := readFile(t, register); got != "header\n" {
		t.Fatalf("register = %q", got)
	}
	refused(t, p.write(register, "x"), "not inside the vault")
	refused(t, p.writeScaffold(filepath.Join(code, "src", "_identities.tsv"), "x"), "not inside the vault")
	refused(t, p.writeScaffold(filepath.Join(code, "src", "algo.py"), "x"), "not inside the vault")
	unregistered := filepath.Join(t.TempDir(), "_identities.tsv")
	refused(t, p.writeScaffold(unregistered, "x"), "not inside the vault")
}

// The register is recognised by what it resolves to as well as by its
// spelling, as `_is_external_register` compares both.
func TestGateRecognisesAnExternalRegisterByItsResolvedPath(t *testing.T) {
	p, _ := newPlace(t)
	code := t.TempDir()
	p.registers = []string{filepath.Join(code, "_identities.tsv")}
	sep := string(filepath.Separator)
	spelled := code + sep + "sub" + sep + ".." + sep + "_identities.tsv"
	if err := p.writeScaffold(spelled, "x"); err != nil {
		t.Fatalf("a register spelled another way was refused: %v", err)
	}
}

// `_gate_external_register`: the exemption skips the walk, so the path
// itself must be no link.
func TestGateRefusesAnExternalRegisterThatIsALink(t *testing.T) {
	p, _ := newPlace(t)
	register := filepath.Join(t.TempDir(), "_identities.tsv")
	p.registers = []string{register}
	linkNamed(t, "_identities.tsv")
	refused(t, p.writeScaffold(register, "x"), register+": is a link, not a regular register file")
}

// Only a name that normalises to the register takes the exemption.
func TestGateGivesTheExemptionToTheRegisterNameAlone(t *testing.T) {
	p, _ := newPlace(t)
	log := filepath.Join(t.TempDir(), "log.md")
	p.registers = []string{log}
	refused(t, p.writeScaffold(log, "x"), "not inside the vault")
}

// An external register that cannot be resolved is no match.
func TestGateReadsAnUnresolvableRegisterAsNoMatch(t *testing.T) {
	p, _ := newPlace(t)
	code := t.TempDir()
	register := filepath.Join(code, "_identities.tsv")
	p.registers = []string{filepath.Join(code, "other", "_identities.tsv")}
	real := resolvePath
	seam(t, &resolvePath, func(path string) (string, error) {
		if strings.Contains(path, "other") {
			return "", errors.New("unresolvable")
		}
		return real(path)
	})
	refused(t, p.writeScaffold(register, "x"), "not inside the vault")
}

// `_touch` (apply.py:511-529): the record comes before the write, so a
// write that fails half-way still names the file.
func TestWriteRecordsTheFileBeforeTheWrite(t *testing.T) {
	p, vault := newPlace(t)
	// A directory where the page should be: the read before the write fails.
	page := filepath.Join(vault, "wiki", "thema.md")
	if err := os.MkdirAll(page, 0o755); err != nil {
		t.Fatal(err)
	}
	err := p.write(page, "x")
	var refusal *gateError
	if err == nil || errors.As(err, &refusal) {
		t.Fatalf("want an OS error, got %v", err)
	}
	if want := []string{"wiki/thema.md"}; !slices.Equal(p.touched, want) {
		t.Fatalf("touched = %v, want %v", p.touched, want)
	}
}

// The same, one step later: the parent cannot be created.
func TestWriteRecordsTheFileWhenItsParentCannotBeMade(t *testing.T) {
	p, vault := newPlace(t)
	blocker := filepath.Join(vault, "wiki", "topics")
	writeFile(t, blocker, "a file, not a directory")
	page := filepath.Join(blocker, "thema.md")
	if err := p.write(page, "x"); err == nil {
		t.Fatal("a write below a file went through")
	}
	if want := []string{"wiki/topics/thema.md"}; !slices.Equal(p.touched, want) {
		t.Fatalf("touched = %v, want %v", p.touched, want)
	}
}

// The write itself goes through `lock.ReplaceText`, creates what is
// missing, and keeps the bytes as given.
func TestWriteCreatesTheFileAndItsParent(t *testing.T) {
	p, vault := newPlace(t)
	page := filepath.Join(vault, "wiki", "neu", "thema.md")
	if err := p.write(page, "a\r\nb\n"); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, page); got != "a\r\nb\n" {
		t.Fatalf("page = %q", got)
	}
	if want := []string{"wiki/neu/thema.md"}; !slices.Equal(p.touched, want) {
		t.Fatalf("touched = %v, want %v", p.touched, want)
	}
}

// test_an_unchanged_case_file_is_not_reported_as_dirty (test_apply.py:1638):
// a write that changes no byte is withdrawn from the record, and a file
// whose bytes differ only in line endings is changed.
func TestWriteWithdrawsTheRecordOfANoOp(t *testing.T) {
	p, vault := newPlace(t)
	log := filepath.Join(vault, "wiki", "log.md")
	writeFile(t, log, "zeile\r\n")
	if err := p.writeScaffold(log, "zeile\r\n"); err != nil {
		t.Fatal(err)
	}
	if len(p.touched) != 0 {
		t.Fatalf("an unchanged file was recorded: %v", p.touched)
	}
	if err := p.writeScaffold(log, "zeile\n"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"wiki/log.md"}; !slices.Equal(p.touched, want) {
		t.Fatalf("touched = %v, want %v", p.touched, want)
	}
}

// The record keeps the write order, one entry per write that changed bytes.
func TestTouchedKeepsTheWriteOrder(t *testing.T) {
	p, vault := newPlace(t)
	for _, step := range []func() error{
		func() error { return p.write(filepath.Join(vault, "wiki", "thema.md"), "x") },
		func() error { return p.writeScaffold(filepath.Join(vault, "_identities.tsv"), "x") },
		func() error { return p.writeScaffold(filepath.Join(vault, "wiki", "log.md"), "x") },
		func() error { return p.writeScaffold(filepath.Join(vault, "wiki", "audit.md"), "x") },
		func() error { return p.remove(filepath.Join(vault, "review", "c1")) },
	} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"wiki/thema.md", "_identities.tsv", "wiki/log.md", "wiki/audit.md", "review/c1"}
	if !slices.Equal(p.touched, want) {
		t.Fatalf("touched = %v, want %v", p.touched, want)
	}
}

func testCase() maintenance.Case {
	return maintenance.Case{
		ID: "c1", Area: "vault", Target: "thema.md", TargetHash: "sha256:00",
		State: "in_review", Trigger: "source_change", Weight: "normal",
		Created: time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC),
	}
}

// `_record_case`: through the gate, recorded, withdrawn when unchanged.
func TestRecordCaseWritesThroughTheGate(t *testing.T) {
	p, vault := newPlace(t)
	path := filepath.Join(vault, "review", "c1", "case.toml")
	if err := p.recordCase(path, testCase()); err != nil {
		t.Fatal(err)
	}
	if err := p.recordCase(path, testCase()); err != nil {
		t.Fatal(err)
	}
	if want := []string{"review/c1/case.toml"}; !slices.Equal(p.touched, want) {
		t.Fatalf("touched = %v, want %v", p.touched, want)
	}
	if !strings.Contains(readFile(t, path), `id = "c1"`) {
		t.Fatal("case.toml was not written")
	}
	linkNamed(t, "case.toml")
	refused(t, p.recordCase(path, testCase()), "is a link")
}

// A case file WriteCase refuses leaves its record standing, as a failed
// write does.
func TestRecordCaseKeepsTheRecordOfAFailedWrite(t *testing.T) {
	p, vault := newPlace(t)
	broken := testCase()
	broken.State = "unknown"
	if err := p.recordCase(filepath.Join(vault, "review", "c1", "case.toml"), broken); err == nil {
		t.Fatal("a case outside the vocabulary was written")
	}
	if want := []string{"review/c1/case.toml"}; !slices.Equal(p.touched, want) {
		t.Fatalf("touched = %v, want %v", p.touched, want)
	}
}

// `_remove`: the directory is gone and recorded.
func TestRemoveDeletesTheCaseDirectory(t *testing.T) {
	p, vault := newPlace(t)
	caseDir := filepath.Join(vault, "review", "c1")
	writeFile(t, filepath.Join(caseDir, "case.toml"), "x")
	if err := p.remove(caseDir); err != nil {
		t.Fatal(err)
	}
	absent(t, caseDir)
	if want := []string{"review/c1"}; !slices.Equal(p.touched, want) {
		t.Fatalf("touched = %v, want %v", p.touched, want)
	}
}

// test_a_half_finished_deletion_is_reported (test_apply.py:1685): the
// deletion is recorded before it runs and its failure comes back.
func TestRemoveRecordsAHalfFinishedDeletion(t *testing.T) {
	p, vault := newPlace(t)
	inUse := errors.New("directory in use")
	seam(t, &removeAll, func(string) error { return inUse })
	if err := p.remove(filepath.Join(vault, "review", "c1")); !errors.Is(err, inUse) {
		t.Fatalf("want the deletion's error, got %v", err)
	}
	if want := []string{"review/c1"}; !slices.Equal(p.touched, want) {
		t.Fatalf("touched = %v, want %v", p.touched, want)
	}
}

// `rmtree` raises on a directory that is not there; `os.RemoveAll` would
// answer nil and report a deletion that never happened.
func TestRemoveRefusesADirectoryThatIsNotThere(t *testing.T) {
	p, vault := newPlace(t)
	if err := p.remove(filepath.Join(vault, "review", "fehlt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want a missing directory to be an error, got %v", err)
	}
	if len(p.touched) != 0 {
		t.Fatalf("a deletion that never ran was recorded: %v", p.touched)
	}
}

// test_the_module_calls_no_write_primitive_outside_the_gate
// (test_apply.py:1367): the primitives that touch the disk may be called
// from the barrier's own wrappers and nowhere else in the package, so a new
// write site has to go through the gate to exist at all.
//
// Two tiers, each primitive with the functions that may call it. The disk
// calls belong to the writers below the gate; replaceIfChanged is such a
// writer and does not gate itself, so it is a primitive in its own right,
// callable only from the gated methods -- as Python lists write_if_changed.
func TestNoWritePrimitiveIsCalledOutsideTheBarrier(t *testing.T) {
	below := map[string]bool{"replaceIfChanged": true, "place.recordCase": true, "place.remove": true}
	gated := map[string]bool{
		"place.write": true, "place.writeScaffold": true,
		"place.recordCase": true, "place.remove": true,
	}
	primitives := map[string]map[string]bool{"replaceIfChanged": gated}
	for _, call := range []string{
		"lock.ReplaceText", "maintenance.WriteCase",
		"os.WriteFile", "os.Create", "os.OpenFile",
		"os.Remove", "os.RemoveAll", "os.Rename",
		"os.Mkdir", "os.MkdirAll", "removeAll", "replaceText",
	} {
		primitives[call] = below
	}
	// The third tier renames below the state directory and never into the
	// vault or a wiki, which is all the gate measures: recoverStock puts a
	// read-only area's stock back from aside before a register is written
	// there, and derives its target itself.
	for _, call := range []string{"recoverDir", "lock.Recover"} {
		primitives[call] = map[string]bool{"recoverStock": true}
	}
	offenders := primitiveCallsOutside(t, primitives)
	if len(offenders) != 0 {
		t.Fatalf("write primitives outside the barrier:\n%s", strings.Join(offenders, "\n"))
	}
}

// primitiveCallsOutside lists every call of a primitive, in the package's
// non-test files, from a function the primitive does not allow.
func primitiveCallsOutside(t *testing.T, primitives map[string]map[string]bool) []string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var offenders []string
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			caller := funcName(fn)
			ast.Inspect(fn, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if allowed, primitive := primitives[callee(call.Fun)]; primitive && !allowed[caller] {
					offenders = append(offenders, name+":"+caller+" calls "+callee(call.Fun))
				}
				return true
			})
		}
	}
	return offenders
}

// funcName names a function as `Func`, a method as `Receiver.Method`, so a
// free function cannot borrow a method's allowance by sharing its name.
func funcName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	receiver := fn.Recv.List[0].Type
	if star, ok := receiver.(*ast.StarExpr); ok {
		receiver = star.X
	}
	if ident, ok := receiver.(*ast.Ident); ok {
		return ident.Name + "." + fn.Name.Name
	}
	return fn.Name.Name
}

// callee names a called function as `pkg.Func` or `Func`.
func callee(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		if x, ok := f.X.(*ast.Ident); ok {
			return x.Name + "." + f.Sel.Name
		}
	}
	return ""
}

// isLink is `Path.is_symlink`: a symlink, where the machine lets the test
// make one, and not a plain directory.
func TestIsLinkAnswersForASymlinkAndNotForADirectory(t *testing.T) {
	dir := t.TempDir()
	if isLink(dir) {
		t.Fatal("a directory counted as a link")
	}
	if isLink(filepath.Join(dir, "fehlt")) {
		t.Fatal("a missing path counted as a link")
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Skipf("this machine does not let the test make a symlink: %v", err)
	}
	if !isLink(link) {
		t.Fatal("a symlink did not count as a link")
	}
}
