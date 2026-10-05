package apply

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/config"
)

// Where a case lies and what it names, ported from `_resolve`, `_wiki`,
// `_layout`, `_is_bundle`, `_check_target`, `_target` and
// `_resolve_sources` (apply.py:155-186, :361-446, :656-676, :814-900).
// Where a test carries over one of test_apply.py, its comment names it.

const (
	testReview = "80 Review"
	testWiki   = "90 Wiki"
)

// vaultManifest is the vault's declaration; `.loomux/config.toml` stands
// where Python's fixture writes `.brain.toml`.
func vaultManifest(review string) string {
	return "[area]\nscope = \"knowledge\"\n\n[layout]\nwiki = \"" + testWiki + "\"\nreview = \"" + review + "\"\n"
}

// newVault builds a vault with a manifest, a wiki holding `topics/thema.md`
// and one case directory in the review centre, and returns the vault, the
// case file and the registry that names the vault's own area.
func newVault(t *testing.T) (string, string, []config.Area) {
	t.Helper()
	vault := t.TempDir()
	writeFile(t, filepath.Join(vault, ".loomux", "config.toml"), vaultManifest(testReview))
	writeFile(t, filepath.Join(vault, testWiki, "topics", "thema.md"), "---\ntitle: x\n---\n")
	casePath := filepath.Join(vault, testReview, "knowledge", "c1", "case.toml")
	writeFile(t, casePath, "")
	areas := []config.Area{{Scope: "knowledge", Path: vault, WikiPath: filepath.Join(vault, testWiki)}}
	return vault, casePath, areas
}

func caseOf(area string) maintenance.Case {
	c := testCase()
	c.Area = area
	c.Target = "topics/thema.md"
	return c
}

func TestResolveFindsVaultReviewCentreAndWiki(t *testing.T) {
	vault, casePath, areas := newVault(t)
	got, err := resolve(casePath, caseOf("knowledge"), areas)
	if err != nil {
		t.Fatal(err)
	}
	want := resolved{
		vault:     vault,
		review:    filepath.Join(vault, testReview),
		wiki:      filepath.Join(vault, testWiki),
		directory: filepath.Dir(casePath),
		area:      areas[0],
	}
	if got != want {
		t.Fatalf("resolve = %+v, want %+v", got, want)
	}
}

// The nearest marker decides, not the outermost one.
func TestResolveTakesTheNearestVault(t *testing.T) {
	outer := t.TempDir()
	writeFile(t, filepath.Join(outer, ".loomux", "config.toml"), vaultManifest("elsewhere"))
	vault := filepath.Join(outer, "inner")
	writeFile(t, filepath.Join(vault, ".loomux", "config.toml"), vaultManifest(testReview))
	casePath := filepath.Join(vault, testReview, "c1", "case.toml")
	areas := []config.Area{{Scope: "knowledge", Path: vault, WikiPath: filepath.Join(vault, testWiki)}}
	got, err := resolve(casePath, caseOf("knowledge"), areas)
	if err != nil {
		t.Fatal(err)
	}
	if got.vault != vault {
		t.Fatalf("vault = %s, want %s", got.vault, vault)
	}
}

// test_a_case_with_no_manifest_above_it_is_refused (test_apply.py:483).
func TestResolveRefusesACaseWithNoVaultAbove(t *testing.T) {
	stray := filepath.Join(t.TempDir(), "stray", "fall", "case.toml")
	_, err := resolve(stray, caseOf("knowledge"), nil)
	refused(t, err, stray+": no area declaration above this case; there is no vault to write")
}

// A marker that is a directory is no marker, as `is_file()` has it.
func TestResolveSkipsAMarkerThatIsNoFile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "v", ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := resolve(filepath.Join(root, "v", "r", "c1", "case.toml"), caseOf("knowledge"), nil)
	refused(t, err, "there is no vault to write")
}

// test_an_unreadable_manifest_is_refused (test_apply.py:696).
func TestResolveRefusesAnUnreadableManifest(t *testing.T) {
	for name, text := range map[string]string{
		"broken":   "[area\n",
		"no scope": "[area]\n",
	} {
		t.Run(name, func(t *testing.T) {
			vault, casePath, areas := newVault(t)
			writeFile(t, filepath.Join(vault, ".loomux", "config.toml"), text)
			_, err := resolve(casePath, caseOf("knowledge"), areas)
			refused(t, err, filepath.Join(vault, ".loomux", "config.toml"))
		})
	}
}

// A `.loomux/config.toml` without [area] is policy only and marks no vault:
// the walk goes on past one between the case and the vault.
func TestResolveWalksPastAPolicyOnlyConfig(t *testing.T) {
	policy := "[policy]\nx = 1\n"
	t.Run("between", func(t *testing.T) {
		vault, casePath, areas := newVault(t)
		writeFile(t, filepath.Join(vault, testReview, ".loomux", "config.toml"), policy)
		got, err := resolve(casePath, caseOf("knowledge"), areas)
		if err != nil {
			t.Fatal(err)
		}
		if got.vault != vault {
			t.Fatalf("vault = %s, want %s", got.vault, vault)
		}
	})
	t.Run("alone", func(t *testing.T) {
		stray := filepath.Join(t.TempDir(), "r", "c1", "case.toml")
		writeFile(t, filepath.Join(filepath.Dir(filepath.Dir(stray)), ".loomux", "config.toml"), policy)
		_, err := resolve(stray, caseOf("knowledge"), nil)
		refused(t, err, "there is no vault to write")
	})
}

// `Path(".").parents` is empty: a case directory of `.` asks no ancestor,
// not even the working directory.
func TestNearestVaultAsksNothingAboveTheDot(t *testing.T) {
	if dir, manifest, err := nearestVault("."); dir != "" || manifest != nil || err != nil {
		t.Fatalf("nearestVault(.) = %q, %v, %v; want nothing", dir, manifest, err)
	}
}

// test_a_vault_whose_manifest_declares_no_review_centre_is_refused
// (test_apply.py:623).
func TestResolveRefusesAVaultWithoutAReviewCentre(t *testing.T) {
	vault, casePath, areas := newVault(t)
	writeFile(t, filepath.Join(vault, ".loomux", "config.toml"), "[area]\nscope = \"knowledge\"\n")
	_, err := resolve(casePath, caseOf("knowledge"), areas)
	refused(t, err, "declares no [layout] review; there is nowhere to go")
}

// test_a_manifest_review_outside_the_area_is_refused (test_apply.py:1189),
// and every other spelling `_layout` refuses.
func TestResolveRefusesAReviewCentreNotRelativeToTheArea(t *testing.T) {
	values := []string{"/woanders", `\\woanders`, "../woanders", `a\..\b`, "a/../b"}
	if runtime.GOOS == "windows" {
		values = append(values, "C:/woanders", "C:woanders")
	}
	for _, value := range values {
		t.Run(value, func(t *testing.T) {
			vault, casePath, areas := newVault(t)
			writeFile(t, filepath.Join(vault, ".loomux", "config.toml"),
				"[area]\nscope = \"knowledge\"\n[layout]\nreview = '"+value+"'\n")
			_, err := resolve(casePath, caseOf("knowledge"), areas)
			refused(t, err, "[layout] review must be relative to the area, found '"+strings.ReplaceAll(value, `\`, `\\`)+"'")
		})
	}
}

// test_a_case_outside_any_declared_review_centre_is_refused (test_apply.py:466).
func TestResolveRefusesACaseOutsideTheReviewCentre(t *testing.T) {
	vault, _, areas := newVault(t)
	stray := filepath.Join(vault, "98 Anderswo", "knowledge", "fall", "case.toml")
	_, err := resolve(stray, caseOf("knowledge"), areas)
	refused(t, err, "not inside the review centre "+filepath.Join(vault, testReview))
}

// test_a_case_naming_an_area_the_registry_does_not_know_is_refused_by_name
// (test_apply.py:539).
func TestResolveRefusesAnAreaTheRegistryDoesNotKnow(t *testing.T) {
	_, casePath, areas := newVault(t)
	_, err := resolve(casePath, caseOf("fremd"), areas)
	refused(t, err, "case belongs to area 'fremd', which the registry does not register")
}

// test_an_area_registered_without_a_wiki_is_refused (test_apply.py:547).
func TestResolveRefusesAnAreaWithoutAWiki(t *testing.T) {
	_, casePath, areas := newVault(t)
	areas[0].WikiPath = ""
	_, err := resolve(casePath, caseOf("knowledge"), areas)
	refused(t, err, "area 'knowledge' is registered with no wiki; there is nowhere to go")
}

// test_a_case_of_another_registered_area_is_written_into_that_areas_wiki
// (test_apply.py:524): the registry, not the vault's manifest, names the wiki.
func TestResolveTakesTheWikiOfTheCasesAreaFromTheRegistry(t *testing.T) {
	vault, casePath, areas := newVault(t)
	other := filepath.Join(vault, "91 Projekte", "x")
	areas = append(areas, config.Area{Scope: "project/x", Path: vault, WikiPath: other})
	got, err := resolve(casePath, caseOf("project/x"), areas)
	if err != nil {
		t.Fatal(err)
	}
	if got.wiki != other || got.area.Scope != "project/x" {
		t.Fatalf("resolve = %+v, want the wiki %s of project/x", got, other)
	}
}

// test_a_wiki_outside_the_vault_is_refused (test_apply.py:556).
func TestResolveRefusesAWikiOutsideTheVault(t *testing.T) {
	vault, casePath, areas := newVault(t)
	foreign := filepath.Join(t.TempDir(), "fremd")
	areas[0].WikiPath = foreign
	_, err := resolve(casePath, caseOf("knowledge"), areas)
	refused(t, err, "the wiki of area 'knowledge' is "+foreign+", outside the vault "+vault+" and not a scaffolded bundle")
}

// test_a_wiki_outside_the_vault_that_is_a_scaffolded_bundle_is_approvable
// (test_apply.py:571).
func TestResolveAcceptsAScaffoldedBundleOutsideTheVault(t *testing.T) {
	_, casePath, areas := newVault(t)
	bundle := filepath.Join(t.TempDir(), "project-x", "docs", "wiki")
	writeFile(t, filepath.Join(bundle, "_schema.md"), "# Schema\n")
	areas[0].WikiPath = bundle
	got, err := resolve(casePath, caseOf("knowledge"), areas)
	if err != nil {
		t.Fatal(err)
	}
	if got.wiki != bundle {
		t.Fatalf("wiki = %s, want %s", got.wiki, bundle)
	}
}

// `_schema.md` has to be a file; a directory of that name scaffolds nothing.
func TestResolveRefusesABundleWhoseSchemaIsNoFile(t *testing.T) {
	_, casePath, areas := newVault(t)
	bundle := filepath.Join(t.TempDir(), "wiki")
	if err := os.MkdirAll(filepath.Join(bundle, "_schema.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	areas[0].WikiPath = bundle
	_, err := resolve(casePath, caseOf("knowledge"), areas)
	refused(t, err, "not a scaffolded bundle")
}

// test_a_registered_wiki_that_climbs_out_of_the_vault_is_refused
// (test_apply.py:1176).
func TestResolveRefusesARegisteredWikiThatClimbsOut(t *testing.T) {
	vault, casePath, areas := newVault(t)
	areas[0].WikiPath = filepath.ToSlash(filepath.Join(vault, testWiki)) + "/../.."
	_, err := resolve(casePath, caseOf("knowledge"), areas)
	refused(t, err, "outside the vault")
}

// test_a_wiki_that_resolves_out_of_the_vault_is_refused (test_apply.py:1267).
func TestResolveRefusesAWikiThatResolvesOutOfTheVault(t *testing.T) {
	_, casePath, areas := newVault(t)
	outside := filepath.Join(t.TempDir(), "anderswo", "wiki")
	real := resolvePath
	seam(t, &resolvePath, func(path string) (string, error) {
		if filepath.Base(path) == testWiki {
			return outside, nil
		}
		return real(path)
	})
	_, err := resolve(casePath, caseOf("knowledge"), areas)
	refused(t, err, "outside the vault")
}

// A resolver failure is the disk's error, not a refusal, and is passed on.
func TestResolvePassesOnAResolverFailure(t *testing.T) {
	for _, name := range []string{testWiki, ""} {
		t.Run("fails at "+name, func(t *testing.T) {
			vault, casePath, areas := newVault(t)
			boom := errors.New("boom")
			real := resolvePath
			seam(t, &resolvePath, func(path string) (string, error) {
				if filepath.Base(path) == name || (name == "" && path == vault) {
					return "", boom
				}
				return real(path)
			})
			if _, err := resolve(casePath, caseOf("knowledge"), areas); !errors.Is(err, boom) {
				t.Fatalf("want the resolver's error, got %v", err)
			}
		})
	}
}

// test_a_target_with_a_newline_is_refused, test_a_tab_in_the_target_is_refused,
// test_a_unicode_line_separator_in_the_target_is_refused (test_apply.py:945,
// :962, :1160), and U+200E, a format character the Go form let through.
func TestCheckTargetRefusesControlCharacters(t *testing.T) {
	for _, char := range []string{"\n", "\t", "\x7f", "\u0085", "\u2028", "\u2029", "\u200b", "\u200e"} {
		target := "topics/x" + char + ".md"
		refused(t, checkTarget(target), "target contains a control character")
	}
}

// test_a_target_of_only_whitespace_is_refused (test_apply.py:1053).
func TestCheckTargetRefusesAnEmptyTarget(t *testing.T) {
	for _, target := range []string{"", "   ", "\u3000", "\x1c"} {
		refused(t, checkTarget(target), "case target is empty")
	}
}

// test_a_target_with_surrounding_whitespace_is_refused (test_apply.py:1169).
func TestCheckTargetRefusesSurroundingWhitespace(t *testing.T) {
	refused(t, checkTarget(" topics/thema.md "), "' topics/thema.md ': target has surrounding whitespace")
	refused(t, checkTarget("topics/thema.md\u00a0"), "target has surrounding whitespace")
}

func TestCheckTargetAcceptsAPlainTarget(t *testing.T) {
	if err := checkTarget("topics/Thema mit Leerzeichen.md"); err != nil {
		t.Fatal(err)
	}
}

func resolvedVault(t *testing.T) resolved {
	t.Helper()
	_, casePath, areas := newVault(t)
	r, err := resolve(casePath, caseOf("knowledge"), areas)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestTargetPathNamesThePageInTheWiki(t *testing.T) {
	r := resolvedVault(t)
	for _, target := range []string{"topics/thema.md", `topics\thema.md`, "./topics//thema.md"} {
		got, err := targetPath(r, target)
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(r.wiki, "topics", "thema.md"); got != want {
			t.Fatalf("targetPath(%q) = %s, want %s", target, got, want)
		}
	}
}

// test_a_target_pointing_out_of_the_wiki_is_refused and
// test_an_absolute_target_is_refused (test_apply.py:877, :885); `a/..` is the
// spelling the Go form, which looked for "../" alone, let through.
func TestTargetPathRefusesATargetOutsideTheWiki(t *testing.T) {
	targets := []string{"../../../anderswo.md", "/woanders.md", `\woanders.md`,
		"topics/..", `topics\..\thema.md`, ".."}
	if runtime.GOOS == "windows" {
		targets = append(targets, "C:/woanders.md", "C:woanders.md")
	}
	for _, target := range targets {
		_, err := targetPath(resolvedVault(t), target)
		refused(t, err, target+": target is not inside the wiki")
	}
}

// test_a_scaffold_file_may_not_be_the_target_of_a_case and its neighbours
// (test_apply.py:1223, :1235, :1242, :1248).
func TestTargetPathRefusesAScaffoldFile(t *testing.T) {
	for _, target := range []string{"audit.md", "Audit.md", "_identities.tsv", "topics/log.md", "log.md/x.md", "audit.md."} {
		_, err := targetPath(resolvedVault(t), target)
		refused(t, err, "a scaffold file, not a page a case may change")
	}
}

// test_a_page_that_resolves_out_of_the_wiki_is_refused (test_apply.py:992).
func TestTargetPathRefusesAPageThatResolvesOutOfTheWiki(t *testing.T) {
	r := resolvedVault(t)
	outside := filepath.Join(t.TempDir(), "anderswo", "seite.md")
	real := resolvePath
	seam(t, &resolvePath, func(path string) (string, error) {
		if filepath.Base(path) == "thema.md" {
			return outside, nil
		}
		return real(path)
	})
	_, err := targetPath(r, "topics/thema.md")
	refused(t, err, "topics/thema.md: target resolves to a path not inside the wiki")
}

func TestTargetPathPassesOnAResolverFailure(t *testing.T) {
	for _, name := range []string{"thema.md", testWiki} {
		t.Run(name, func(t *testing.T) {
			r := resolvedVault(t)
			boom := errors.New("boom")
			real := resolvePath
			seam(t, &resolvePath, func(path string) (string, error) {
				if filepath.Base(path) == name {
					return "", boom
				}
				return real(path)
			})
			if _, err := targetPath(r, "topics/thema.md"); !errors.Is(err, boom) {
				t.Fatalf("want the resolver's error, got %v", err)
			}
		})
	}
}

// test_a_link_on_the_way_to_the_target_is_refused (test_apply.py:1197).
func TestTargetPathRefusesALinkOnTheWay(t *testing.T) {
	for _, name := range []string{"topics", "thema.md"} {
		r := resolvedVault(t)
		linkNamed(t, name)
		_, err := targetPath(r, "topics/thema.md")
		refused(t, err, "topics/thema.md: "+name+" is a link, not a page")
	}
}

// A leading `./` names no component: pathlib drops `.` from `parts`
// (apply.py:890-899), so the walk never asks whether the wiki itself is a
// link -- that is the registration's business, not the target's.
func TestTargetPathWalksNoDotComponent(t *testing.T) {
	r := resolvedVault(t)
	wiki := filepath.Clean(r.wiki)
	seam(t, &isLink, func(path string) bool { return filepath.Clean(path) == wiki })
	if _, err := targetPath(r, "./topics/thema.md"); err != nil {
		t.Fatalf("targetPath: %v", err)
	}
	// An empty target has no parts either (`PurePosixPath("").parts` is
	// empty): it names the wiki itself, which is no page -- not a link.
	_, err := targetPath(r, "")
	refused(t, err, r.wiki+": the case's target page is gone")
}

// test_a_vanished_target_page_is_refused (test_apply.py:702); in Python the
// check follows `_target` at its one caller, `_apply`.
func TestTargetPathRefusesAVanishedPage(t *testing.T) {
	r := resolvedVault(t)
	for _, target := range []string{"topics/weg.md", "topics"} {
		_, err := targetPath(r, target)
		refused(t, err, filepath.Join(r.wiki, filepath.FromSlash(target))+": the case's target page is gone")
	}
}

func writeRegister(t *testing.T, dir string, rows ...identity.Identity) string {
	t.Helper()
	identities := map[string]identity.Identity{}
	for _, row := range rows {
		identities[row.Relative] = row
	}
	path := filepath.Join(dir, registerName)
	writeFile(t, path, identity.RenderIdentities(identities))
	return path
}

func row(docID, relative string) identity.Identity {
	return identity.Identity{DocID: docID, Relative: relative, ContentHash: "sha256:" + docID, Revision: 1}
}

// `_resolve_sources` looks in the register of every registered area, so a
// source of a second area outside the vault is found in that area's own
// register; the Go form read the vault's alone.
func TestResolveSourcesFindsASourceInASecondAreaOutsideTheVault(t *testing.T) {
	vault, other := t.TempDir(), t.TempDir()
	vaultRegister := writeRegister(t, vault, row("A", "notes/a.md"))
	otherRegister := writeRegister(t, other, row("B", "src/b.go"))
	areas := []config.Area{{Scope: "knowledge", Path: vault}, {Scope: "project/x", Path: other}}
	got, err := resolveSources(areas, config.ArtifactLookup{Primary: t.TempDir()}, []string{"A", "B"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]sourceFile{
		"A": {docID: "A", relative: "notes/a.md", path: filepath.Join(vault, "notes", "a.md"), readFrom: vaultRegister, register: vaultRegister},
		"B": {docID: "B", relative: "src/b.go", path: filepath.Join(other, "src", "b.go"), readFrom: otherRegister, register: otherRegister},
	}
	if len(got) != len(want) || got["A"] != want["A"] || got["B"] != want["B"] {
		t.Fatalf("resolveSources = %+v, want %+v", got, want)
	}
}

// test_a_source_the_register_forgot_does_not_block_the_approval
// (test_apply.py:910): an unknown doc_id is skipped, not refused.
func TestResolveSourcesSkipsAnUnknownDocID(t *testing.T) {
	vault := t.TempDir()
	writeRegister(t, vault, row("A", "a.md"))
	got, err := resolveSources([]config.Area{{Scope: "knowledge", Path: vault}}, config.ArtifactLookup{}, []string{"A", "Z"})
	if err != nil {
		t.Fatal(err)
	}
	if _, found := got["Z"]; found || len(got) != 1 {
		t.Fatalf("resolveSources = %+v, want A alone", got)
	}
}

// The first area in registry order that knows a doc_id wins.
func TestResolveSourcesTakesTheFirstAreaThatKnowsTheDocID(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	writeRegister(t, first, row("A", "first.md"))
	writeRegister(t, second, row("A", "second.md"))
	areas := []config.Area{{Scope: "one", Path: first}, {Scope: "two", Path: second}}
	got, err := resolveSources(areas, config.ArtifactLookup{}, []string{"A"})
	if err != nil {
		t.Fatal(err)
	}
	if got["A"].relative != "first.md" {
		t.Fatalf("resolveSources = %+v, want first.md", got)
	}
}

// Once every doc_id is found no further register is read, so a broken one
// after it costs nothing -- as `return resolved` inside the loop has it.
func TestResolveSourcesStopsOnceEverythingIsFound(t *testing.T) {
	first, broken := t.TempDir(), t.TempDir()
	writeRegister(t, first, row("A", "a.md"))
	writeFile(t, filepath.Join(broken, registerName), "header\nnot four fields\n")
	areas := []config.Area{{Scope: "one", Path: first}, {Scope: "two", Path: broken}}
	if _, err := resolveSources(areas, config.ArtifactLookup{}, []string{"A"}); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveSources(areas, config.ArtifactLookup{}, []string{"A", "Z"}); err == nil {
		t.Fatal("a broken register that has to be read is an error")
	}
}

// A register that is not a regular file is skipped, as `is_file()` has it,
// and no doc_id reads no register at all.
func TestResolveSourcesSkipsARegisterThatIsNoFile(t *testing.T) {
	area := t.TempDir()
	if err := os.MkdirAll(filepath.Join(area, registerName), 0o755); err != nil {
		t.Fatal(err)
	}
	areas := []config.Area{{Scope: "one", Path: area}, {Scope: "two", Path: t.TempDir()}}
	got, err := resolveSources(areas, config.ArtifactLookup{}, []string{"A"})
	if err != nil || len(got) != 0 {
		t.Fatalf("resolveSources = %+v, %v; want nothing", got, err)
	}
	if got, err := resolveSources(areas, config.ArtifactLookup{}, nil); err != nil || len(got) != 0 {
		t.Fatalf("resolveSources = %+v, %v; want nothing", got, err)
	}
}

// A read-only area keeps its register in the state directory
// (`area_artifact_dir`), while the source stays below the area.
func TestResolveSourcesReadsAReadOnlyAreasRegisterFromTheStateDirectory(t *testing.T) {
	area, state := t.TempDir(), t.TempDir()
	ro := config.Area{Scope: "project/ro", Path: area, ReadOnly: true}
	lookup := config.ArtifactLookup{Primary: state}
	dir := config.ManifestDir(ro, lookup.Primary)
	register := writeRegister(t, dir, row("A", "docs/a.md"))
	got, err := resolveSources([]config.Area{ro}, lookup, []string{"A"})
	if err != nil {
		t.Fatal(err)
	}
	want := sourceFile{docID: "A", relative: "docs/a.md", path: filepath.Join(area, "docs", "a.md"), readFrom: register, register: register}
	if got["A"] != want {
		t.Fatalf("resolveSources = %+v, want %+v", got["A"], want)
	}
	if registers := registersOf([]config.Area{ro}, lookup); len(registers) != 1 || registers[0] != register {
		t.Fatalf("registersOf = %v, want [%s]", registers, register)
	}
}

// registersOf names every area's register, whether it exists or not, as
// `_is_external_register` lists them.
func TestRegistersOfNamesEveryArea(t *testing.T) {
	areas := []config.Area{{Scope: "one", Path: "C:/a"}, {Scope: "two", Path: "C:/b"}}
	got := registersOf(areas, config.ArtifactLookup{})
	want := []string{filepath.Join("C:/a", registerName), filepath.Join("C:/b", registerName)}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("registersOf = %v, want %v", got, want)
	}
}

// Within one register the row whose relative sorts first wins where two
// share a doc_id: Python takes the first in file order, and the register
// reader hands back a map, which keeps none.
func TestResolveSourcesPicksDuplicatesInSortedOrder(t *testing.T) {
	area := t.TempDir()
	writeRegister(t, area, row("A", "z.md"), row("A", "b.md"), row("A", "m.md"))
	for range 5 {
		got, err := resolveSources([]config.Area{{Scope: "one", Path: area}}, config.ArtifactLookup{}, []string{"A"})
		if err != nil {
			t.Fatal(err)
		}
		if got["A"].relative != "b.md" {
			t.Fatalf("resolveSources = %+v, want b.md", got)
		}
	}
}
