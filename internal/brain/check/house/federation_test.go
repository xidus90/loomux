package house

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/brain/wiki"
	"github.com/xidus90/loomux/internal/config"
)

// cites is one page whose single `sources[]` entry names a resource. The
// other fields stay zero: `wrong-direction` reads the resource and nothing
// else, and a fixture carrying more would let a second rule of this file
// decide a test that means to measure the first.
func cites(relative, resource string) wiki.WikiPage {
	return wiki.WikiPage{
		Relative: relative,
		Sources:  []wiki.Source{{Resource: resource}},
	}
}

// vault lays out a signpost area and returns its two roots. They are two
// and not one on purpose: catalog links resolve against the *wiki* root
// (Python's `_catalog_targets(context.root)`,
// `src/brain/wiki/lint.py:474`), while the hub pointer is built from the
// area *path* (`src/brain/cli.py:1520`). A test that used one for both
// would pass over the one area of the real registry whose wiki lies
// outside the vault.
func vault(t *testing.T) (areaPath, wikiPath string) {
	t.Helper()
	areaPath = t.TempDir()
	return areaPath, filepath.Join(areaPath, "90 Wiki")
}

// signpost is the registered signpost area of the tests below.
func signpost(areaPath, wikiPath string) config.Area {
	return config.Area{
		Scope:    "knowledge",
		Path:     areaPath,
		WikiPath: wikiPath,
		Signpost: true,
		Shared:   true,
	}
}

// linkTo spells an absolute path the way a catalog line reaches it: as a
// target relative to the wiki root, with `/` separators and percent
// escapes for the blanks the link pattern refuses.
func linkTo(wikiPath, target string) string {
	rel, err := filepath.Rel(wikiPath, target)
	if err != nil {
		return target
	}
	return strings.ReplaceAll(filepath.ToSlash(rel), " ", "%20")
}

func TestASharedAreaMustNotCiteAProject(t *testing.T) {
	// Verbund §6.2 against architecture §5.7.5: projects read shared
	// areas, shared areas read no project. The rule is what makes the
	// prose enforceable.
	areas := []config.Area{
		{Scope: "engineering/python", Path: "p", WikiPath: "p",
			Shared: true},
		{Scope: "project/ultra-brain", Path: "u", WikiPath: "u"},
	}
	bundles := map[string][]wiki.WikiPage{
		"engineering/python": {
			cites("a.md", "brain://project/ultra-brain/topics/y"),
		},
	}
	got := Federation(bundles, areas, config.ArtifactLookup{})
	if !hasRule(got, "wrong-direction") {
		t.Fatalf("a project citation was not reported: %v", got)
	}
	if sev(got, "wrong-direction") != check.Error {
		t.Fatalf("severity = %v, want error",
			sev(got, "wrong-direction"))
	}
}

func TestWrongDirectionNamesTheWholeReference(t *testing.T) {
	// `_cited_scope` matches against the *shared* scopes alone
	// (`src/brain/cli.py:1552` hands `shared_scopes` in), so a reference
	// no shared scope owns comes back uncut -- and Python says why at
	// `src/brain/wiki/lint.py:394-396`: guessing where the scope ends
	// would put a name in the message nobody can look up.
	areas := []config.Area{
		{Scope: "knowledge", Path: "k", WikiPath: "k", Shared: true},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {cites("a.md", "brain://project/space/topics/y")},
	}
	got := messagesOf(Federation(bundles, areas, config.ArtifactLookup{}), "wrong-direction")
	if len(got) != 1 {
		t.Fatalf("messages = %v, want one", got)
	}
	if !strings.Contains(got[0], `"project/space/topics/y"`) {
		t.Fatalf("message does not carry the whole reference: %q",
			got[0])
	}
}

func TestWrongDirectionCarriesTheAreaItJudged(t *testing.T) {
	// `check.Finding.Scope` exists for a run that covers several areas
	// (the comment on `check.Finding.Scope`), and this is the first function that
	// knows the scope: `Page` and `Bundle` are handed one bundle without
	// its name.
	areas := []config.Area{
		{Scope: "engineering/craft", Path: "c", WikiPath: "c",
			Shared: true},
	}
	bundles := map[string][]wiki.WikiPage{
		"engineering/craft": {cites("a.md", "brain://project/space/y")},
	}
	got := Federation(bundles, areas, config.ArtifactLookup{})
	if len(got) != 1 || got[0].Scope != "engineering/craft" {
		t.Fatalf("findings = %+v, want one on engineering/craft", got)
	}
	if got[0].Axis != check.AxisHouse {
		t.Fatalf("axis = %v, want house", got[0].Axis)
	}
}

func TestASharedAreaMayCiteAnotherSharedArea(t *testing.T) {
	// The direction the rule permits. Both spellings of a hit are here:
	// the scope itself, and a path below it.
	areas := []config.Area{
		{Scope: "knowledge", Path: "k", WikiPath: "k", Shared: true},
		{Scope: "engineering/python", Path: "p", WikiPath: "p",
			Shared: true},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {
			cites("a.md", "brain://engineering/python/topics/y"),
			cites("b.md", "brain://engineering/python"),
		},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("a citation between shared areas was reported: %v", got)
	}
}

func TestASiblingSpeltBelowASharedScopeIsNoHit(t *testing.T) {
	// `engineering/pythonx` merely begins with a shared scope; it is a
	// separate area, and citing it is the forbidden direction. Python
	// asks `parts[: len(scope_parts)] == scope_parts` and so answers the
	// same. Without the separator in the prefix test the rule would fall
	// silent here, and the whole suite stays green -- `names()` has this
	// case, `citedScope` had none.
	areas := []config.Area{
		{Scope: "engineering/python", Path: "p", WikiPath: "p",
			Shared: true},
		{Scope: "engineering/pythonx", Path: "x", WikiPath: "x"},
	}
	bundles := map[string][]wiki.WikiPage{
		"engineering/python": {
			cites("a.md", "brain://engineering/pythonx/topics/y"),
		},
	}
	got := Federation(bundles, areas, config.ArtifactLookup{})
	if len(got) != 1 || got[0].Rule != "wrong-direction" {
		t.Fatalf("a citation to a same-prefixed area went unreported: %v",
			got)
	}
}

func TestAProjectAreaIsNotAskedAboutItsDirection(t *testing.T) {
	// Verbund §6.2 binds `shared` areas alone; a project citing a project
	// is the ordinary case and not this rule's business.
	areas := []config.Area{
		{Scope: "project/ecoflow", Path: "e", WikiPath: "e"},
	}
	bundles := map[string][]wiki.WikiPage{
		"project/ecoflow": {cites("a.md", "brain://project/space/y")},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("a project bundle was judged: %v", got)
	}
}

func TestASourceWithoutASchemeStaysInsideItsArea(t *testing.T) {
	// Verbund §6.2: "Eine Quelle ohne Schema zeigt in den eigenen Bereich
	// und ist nie betroffen." A `https://` reference names no area of
	// this federation either.
	areas := []config.Area{
		{Scope: "knowledge", Path: "k", WikiPath: "k", Shared: true},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {
			cites("a.md", "10 Rohquellen/x.md"),
			cites("b.md", "https://example.invalid/x"),
		},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("a citation with no brain:// form was reported: %v", got)
	}
}

func TestABareBrainReferenceNamesNoScope(t *testing.T) {
	// `brain://` alone leaves netloc and path empty, and Python's
	// `_cited_scope` answers None for it (`src/brain/wiki/lint.py:397`,
	// `joined or None`). Reporting it would name the empty string as the
	// area a page must not cite.
	areas := []config.Area{
		{Scope: "knowledge", Path: "k", WikiPath: "k", Shared: true},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {cites("a.md", "brain://"), cites("b.md", "")},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("a reference naming no scope was reported: %v", got)
	}
}

func TestAScaffoldFileIsNoSubjectOfWrongDirection(t *testing.T) {
	// `judgedInBundle` draws the same line the five other bundle rules
	// draw, and Python drops the scaffold names before any rule sees one
	// (`src/brain/wiki/lint.py:582-586`).
	areas := []config.Area{
		{Scope: "knowledge", Path: "k", WikiPath: "k", Shared: true},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {cites("_schema.md", "brain://project/space/y")},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("a scaffold file was judged: %v", got)
	}
}

func TestAnUnreadBundleIsNotACleanBundle(t *testing.T) {
	// The asymmetry the two rules carry around a missing map entry. A
	// shared area nobody read yields no pages and therefore no findings
	// -- a silence with no candidates behind it. The signpost falls the
	// other way, one test below.
	areas := []config.Area{
		{Scope: "knowledge", Path: "k", WikiPath: "k", Shared: true},
	}
	if got := Federation(map[string][]wiki.WikiPage{}, areas, config.ArtifactLookup{}); got !=
		nil {
		t.Fatalf("an unread shared area produced findings: %v", got)
	}
}

func TestTheSignpostMustNameEveryRegisteredArea(t *testing.T) {
	// Verbund §6.1: "Damit meldet sich ein Projekt nicht nur an, es muss
	// sich anmelden." The catalog names one of the two areas.
	areaPath, wikiPath := vault(t)
	named := filepath.Join(areaPath, "92 Engineering", "python")
	forgotten := filepath.Join(areaPath, "91 Projekte", "space")
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "engineering/python", Path: named, WikiPath: named},
		{Scope: "project/space", Path: forgotten, WikiPath: forgotten},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {catalog(
			linkTo(wikiPath, filepath.Join(named, "index.md")))},
	}
	got := Federation(bundles, areas, config.ArtifactLookup{})
	if len(got) != 1 {
		t.Fatalf("findings = %+v, want one", got)
	}
	if got[0].Rule != "unlisted-area" || got[0].Severity != check.Error {
		t.Fatalf("finding = %+v, want unlisted-area as an error", got[0])
	}
	if got[0].Scope != "knowledge" || got[0].Relative != "index.md" {
		t.Fatalf("finding = %+v, want it on the signpost catalog",
			got[0])
	}
	if !strings.Contains(got[0].Message, `"project/space"`) {
		t.Fatalf("message does not name the area: %q", got[0].Message)
	}
	if !strings.Contains(got[0].Message, forgotten) {
		t.Fatalf("message does not name the expected target: %q",
			got[0].Message)
	}
}

func TestTheSignpostItselfIsNotExpected(t *testing.T) {
	// Verbund §6.1: "Der `signpost`-Bereich selbst wird nicht erwartet --
	// er ist der Ort, an dem man bereits steht."
	areaPath, wikiPath := vault(t)
	areas := []config.Area{signpost(areaPath, wikiPath)}
	bundles := map[string][]wiki.WikiPage{"knowledge": {catalog()}}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("the signpost was expected to link itself: %v", got)
	}
}

func TestAnAreaWithoutAWikiIsNotExpected(t *testing.T) {
	// Verbund §6.1 asks for "jeden registrierten Bereich mit
	// `wiki_path`", and `_expected_targets` skips the others
	// (`src/brain/cli.py:1516-1517`): there is nothing to link to yet.
	areaPath, wikiPath := vault(t)
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "project/nowiki", Path: filepath.Join(areaPath, "n")},
	}
	bundles := map[string][]wiki.WikiPage{"knowledge": {catalog()}}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("an area without a wiki was expected: %v", got)
	}
}

func TestAWikiIsNamedByAnyPageBelowIt(t *testing.T) {
	// The directory arm of `_names`: a wiki is a folder and the signpost
	// may point at any page inside it
	// (`src/brain/wiki/lint.py:489-491`).
	areaPath, wikiPath := vault(t)
	other := filepath.Join(areaPath, "92 Engineering", "python")
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "engineering/python", Path: other, WikiPath: other},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {catalog(linkTo(wikiPath,
			filepath.Join(other, "topics", "deep.md")))},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("a page below the wiki did not name it: %v", got)
	}
}

func TestASiblingSpeltBelowAWikiDoesNotNameIt(t *testing.T) {
	// The other direction of the same arm, and the reason the prefix is
	// taken by path component and not by string: `python-old` starts with
	// `python` as text and is a different area.
	areaPath, wikiPath := vault(t)
	other := filepath.Join(areaPath, "92 Engineering", "python")
	sibling := filepath.Join(areaPath, "92 Engineering", "python-old")
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "engineering/python", Path: other, WikiPath: other},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {catalog(linkTo(wikiPath,
			filepath.Join(sibling, "index.md")))},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 1 {
		t.Fatalf("a sibling folder passed for the wiki: %v", got)
	}
}

func TestAHubPointerNamesAnAreaWhoseWikiLiesElsewhere(t *testing.T) {
	// The case the hub pointer exists for, and the only kind in the real
	// registry: an area that keeps its wiki in its own repository, which no
	// link from an Obsidian vault reaches. The
	// pointer is `<signpost.Path>/<layout.hub>/<last scope segment>.md`.
	areaPath, wikiPath := vault(t)
	write(t, areaPath, ".loomux/config.toml",
		"[area]\nscope = \"knowledge\"\n\n[layout]\nhub = \"91 P\"\n")
	elsewhere := t.TempDir()
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "project/ultra-brain", Path: elsewhere,
			WikiPath: filepath.Join(elsewhere, "docs", "wiki")},
	}
	hub := filepath.Join(areaPath, "91 P", "ultra-brain.md")
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {catalog(linkTo(wikiPath, hub))},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("the hub pointer did not name the area: %v", got)
	}
}

func TestAHubPointerIsComparedByEquality(t *testing.T) {
	// The `.md` arm of `_names`, and Python's own example for why it is
	// not the wider question (`src/brain/wiki/lint.py:492-493`):
	// `Path("p.md/deeper.md").is_relative_to(Path("p.md"))` is True, so a
	// path merely spelt below the page would pass for a link to it.
	areaPath, wikiPath := vault(t)
	write(t, areaPath, ".loomux/config.toml",
		"[area]\nscope = \"knowledge\"\n\n[layout]\nhub = \"91 P\"\n")
	elsewhere := t.TempDir()
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "project/ultra-brain", Path: elsewhere,
			WikiPath: filepath.Join(elsewhere, "docs", "wiki")},
	}
	below := filepath.Join(areaPath, "91 P", "ultra-brain.md", "d.md")
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {catalog(linkTo(wikiPath, below))},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 1 {
		t.Fatalf("a path below the hub page passed for it: %v", got)
	}
}

func TestWithoutAHubTheWikiPathIsTheOnlyName(t *testing.T) {
	// Verbund §5.1 lets a federation run without a hub folder, and
	// `hub_layout` answers None when the key is unsaid
	// (`src/brain/manifest.py:110-112`). The manifest is missing
	// altogether here, which this reader cannot tell from an unsaid key
	// and answers the same way.
	areaPath, wikiPath := vault(t)
	elsewhere := t.TempDir()
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "project/ultra-brain", Path: elsewhere,
			WikiPath: filepath.Join(elsewhere, "docs", "wiki")},
	}
	hub := filepath.Join(areaPath, "91 P", "ultra-brain.md")
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {catalog(linkTo(wikiPath, hub))},
	}
	got := messagesOf(Federation(bundles, areas, config.ArtifactLookup{}), "unlisted-area")
	if len(got) != 1 {
		t.Fatalf("messages = %v, want one", got)
	}
	if strings.Contains(got[0], "ultra-brain.md") {
		t.Fatalf("a hub pointer was expected without a hub folder: %q",
			got[0])
	}
}

func TestWithoutASignpostNothingIsExpected(t *testing.T) {
	// Verbund §5.1: "Kein `signpost`-Bereich ist zulässig -- dann gibt es
	// keinen Wegweiser und `unlisted-area` prüft nichts."
	other := t.TempDir()
	areas := []config.Area{
		{Scope: "project/space", Path: other, WikiPath: other},
	}
	if got := Federation(map[string][]wiki.WikiPage{}, areas, config.ArtifactLookup{}); len(
		got) != 0 {
		t.Fatalf("a federation without a signpost was judged: %v", got)
	}
}

func TestEverySignpostIsAsked(t *testing.T) {
	// Verbund §5.1 calls a second signpost an error *in the registry*.
	// `config.ReadRegistry` refuses one, but `Federation` takes its areas
	// from the caller. Taking the first would let the slice order
	// decide whose catalog goes unchecked, and the spec says that
	// question "hätte keine Antwort" -- so both are asked, and the
	// `Scope` of the finding tells them apart.
	firstPath, firstWiki := vault(t)
	secondPath, secondWiki := vault(t)
	missing := t.TempDir()
	first := signpost(firstPath, firstWiki)
	second := signpost(secondPath, secondWiki)
	second.Scope = "engineering/craft"
	areas := []config.Area{
		first, second,
		{Scope: "project/space", Path: missing, WikiPath: missing},
	}
	// The first catalog names both of its two expected areas; the second
	// forgets `project/space`. Only the second may report, and its
	// finding has to carry its own scope.
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {catalog(
			linkTo(firstWiki, secondWiki),
			linkTo(firstWiki, missing))},
		"engineering/craft": {catalog(linkTo(secondWiki, firstWiki))},
	}
	got := Federation(bundles, areas, config.ArtifactLookup{})
	if len(got) != 1 {
		t.Fatalf("findings = %+v, want one -- the second signpost", got)
	}
	if got[0].Scope != "engineering/craft" {
		t.Fatalf("finding = %+v, want it on the second signpost",
			got[0])
	}
}

func TestASignpostWithoutACatalogNamesNobody(t *testing.T) {
	// The other half of the asymmetry above. Python's
	// `_catalog_targets` returns an empty tuple when the file is absent
	// (`src/brain/wiki/lint.py:475-476`), and `any()` over nothing is
	// False, so every expected area is reported. A bundle nobody read
	// reaches this rule as the same absence, and it cannot tell the two
	// apart -- it errs towards speaking.
	areaPath, wikiPath := vault(t)
	other := t.TempDir()
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "project/space", Path: other, WikiPath: other},
	}
	withPages := map[string][]wiki.WikiPage{
		"knowledge": {{Relative: "topics/a.md"}},
	}
	if got := Federation(withPages, areas, config.ArtifactLookup{}); len(got) != 1 {
		t.Fatalf("a bundle without a catalog named an area: %v", got)
	}
	if got := Federation(nil, areas, config.ArtifactLookup{}); len(got) != 1 {
		t.Fatalf("an unread signpost bundle named an area: %v", got)
	}
}

func TestOnlyTheRootCatalogIsTheSignpost(t *testing.T) {
	// Deliberately narrower than `orphan` two files over, which counts
	// every `index.md` as a link source. Python reads `root / "index.md"`
	// here (`src/brain/wiki/lint.py:474`), and a catalog in `topics/` is
	// the listing of that folder, not the entrance to the federation.
	areaPath, wikiPath := vault(t)
	other := t.TempDir()
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "project/space", Path: other, WikiPath: other},
	}
	nested := wiki.WikiPage{
		Relative: "topics/index.md",
		Links:    []string{linkTo(wikiPath, other)},
	}
	bundles := map[string][]wiki.WikiPage{"knowledge": {nested}}
	got := Federation(bundles, areas, config.ArtifactLookup{})
	if len(got) != 1 {
		t.Fatalf("a nested catalog stood in for the signpost: %v", got)
	}
	// The finding names `index.md` even where no such page was read: the
	// repair is a line in that file, and the nested catalog it did read
	// is not where it belongs.
	if got[0].Relative != "index.md" {
		t.Fatalf("finding = %+v, want it on index.md", got[0])
	}
}

func TestASchemeInTheSignpostNamesNoArea(t *testing.T) {
	// A `brain://` target must name no area at all, and the fixture is
	// built so that reading it as a path would: `brain://project/space/
	// index.md` leaves the path `/space/index.md`, which lands inside
	// this area's own wiki -- where the registration puts the nested
	// area Verbund §3.2 allows.
	areaPath, wikiPath := vault(t)
	nested := filepath.Join(wikiPath, "space")
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "project/space", Path: nested, WikiPath: nested},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {catalog("brain://project/space/index.md")},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 1 {
		t.Fatalf("a brain:// target was read as a path: %v", got)
	}
}

func TestAnEmptyTargetNamesNoEnclosingArea(t *testing.T) {
	// A target with no path must name nothing, and a fixture proves it
	// only where reading it as a path would name something. Joined onto
	// the wiki root, the empty path *is* the wiki root -- which lies
	// inside every area that encloses this one, and the real
	// registration has such an area: `hub` holds the vault folder that
	// four project wikis sit in.
	areaPath, wikiPath := vault(t)
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "hub", Path: areaPath, WikiPath: areaPath},
	}
	bundles := map[string][]wiki.WikiPage{"knowledge": {catalog("?")}}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 1 {
		t.Fatalf("an empty target named the enclosing area: %v", got)
	}
}

func TestALinkIsNoCitation(t *testing.T) {
	// Verbund §6.2 on why the signpost is not itself a violation: "Er
	// trägt Links, keine `sources`-Einträge -- Namen, keine Aussagen.
	// Nichts Projektinternes wandert dadurch nach oben." The page here
	// is an ordinary one, because a catalog is scaffold and would be
	// exempt for the other reason as well.
	areas := []config.Area{
		{Scope: "knowledge", Path: "k", WikiPath: "k", Shared: true},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {{
			Relative: "a.md",
			Links:    []string{"brain://project/space/topics/y"},
		}},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("a link was judged as a citation: %v", got)
	}
}

func TestASignpostWithoutAWikiPathIsNotAsked(t *testing.T) {
	// Every catalog target resolves against the wiki root, so without one
	// there is nothing to resolve against. Python never reaches the rule
	// in that state either: `_lint_targets` only lints areas whose
	// `wiki_path` is set (`src/brain/cli.py:1489`).
	areaPath, _ := vault(t)
	other := t.TempDir()
	bare := signpost(areaPath, "")
	areas := []config.Area{
		bare,
		{Scope: "project/space", Path: other, WikiPath: other},
	}
	bundles := map[string][]wiki.WikiPage{"knowledge": {catalog()}}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("a signpost without a wiki was asked: %v", got)
	}
}

func TestACatalogTargetWithASchemeNamesNoArea(t *testing.T) {
	// Python skips a target carrying a scheme and one with no path at all
	// (`src/brain/wiki/lint.py:480-481`). `brain://` is not clickable in
	// Obsidian, which is the whole reason the signpost uses relative
	// paths (Verbund §4).
	areaPath, wikiPath := vault(t)
	other := t.TempDir()
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "project/space", Path: other, WikiPath: other},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {catalog("brain://project/space/index.md", "?")},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 1 {
		t.Fatalf("a scheme or an empty target named an area: %v", got)
	}
}

func TestTheSignpostIsReadWithItsEscapesUndone(t *testing.T) {
	// Verbund §4: blanks in a target are percent-encoded because the link
	// pattern refuses a raw one, "auch nicht in Winkelklammern". A rule
	// comparing the encoded text would report every area of this vault.
	areaPath, wikiPath := vault(t)
	other := filepath.Join(areaPath, "92 Engineering", "python")
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "engineering/python", Path: other, WikiPath: other},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {catalog("../92%20Engineering/python/index.md")},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("an encoded blank hid the area: %v", got)
	}
}

func TestTheTwoFederationRulesSpeakInOneRun(t *testing.T) {
	// Both rules on one input, so that neither can carry the wrong axis
	// or degree and go unmeasured because nothing made it fire. The order
	// is Python's `RULES` (`src/brain/wiki/lint.py:543,546`):
	// `unlisted_area` before `wrong_direction`.
	areaPath, wikiPath := vault(t)
	other := t.TempDir()
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "project/space", Path: other, WikiPath: other},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {
			catalog(),
			cites("a.md", "brain://project/space/topics/y"),
		},
	}
	got := Federation(bundles, areas, config.ArtifactLookup{})
	if len(got) != 2 {
		t.Fatalf("findings = %+v, want two", got)
	}
	if got[0].Rule != "unlisted-area" ||
		got[1].Rule != "wrong-direction" {
		t.Fatalf("order = %s, %s", got[0].Rule, got[1].Rule)
	}
	for _, f := range got {
		if f.Axis != check.AxisHouse || f.Severity != check.Error {
			t.Fatalf("finding = %+v, want a house error", f)
		}
	}
}

func TestABrokenManifestSilencesTheSignpost(t *testing.T) {
	// The hub folder is read out of the signpost's manifest, and without
	// it every area whose wiki lies outside the vault loses its second
	// name. A declaration that cannot be read must therefore stop the
	// rule, not shrink it: the run would otherwise go red at the areas
	// that did nothing wrong instead of at the one line that did --
	// which is what `hub_layout`'s own docstring refuses
	// (`src/brain/manifest.py:97-108`).
	//
	// Measured against one area of the real registry before this fixture
	// was written: a broken manifest at the vault root reported that area,
	// which the signpost names correctly.
	//
	// `TestWithoutAHubTheWikiPathIsTheOnlyName` is the counter-probe: a
	// manifest that is simply absent leaves the rule running.
	areaPath, wikiPath := vault(t)
	write(t, areaPath, ".loomux/config.toml", "[area\nscope = \"knowledge\"\n")
	elsewhere := t.TempDir()
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "project/ultra-brain", Path: elsewhere,
			WikiPath: filepath.Join(elsewhere, "docs", "wiki")},
	}
	bundles := map[string][]wiki.WikiPage{"knowledge": {catalog()}}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("an unreadable manifest let the rule speak: %v", got)
	}
}

func TestAMalformedEscapeStillNamesAScope(t *testing.T) {
	// Measured on both sides: Python's `urlsplit` never refuses a target,
	// so `brain://project/space/%zz` splits into netloc `project` and
	// path `/space/%zz` and `_cited_scope` hands the scope back --
	// Python reports. A resolver that gave up on the malformed escape
	// would let a guard of this severity fail silent, and a run of the
	// two sides over one page would disagree.
	areas := []config.Area{
		{Scope: "knowledge", Path: "k", WikiPath: "k", Shared: true},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {cites("a.md", "brain://project/space/%zz")},
	}
	got := messagesOf(Federation(bundles, areas, config.ArtifactLookup{}), "wrong-direction")
	if len(got) != 1 {
		t.Fatalf("messages = %v, want one", got)
	}
	if !strings.Contains(got[0], `"project/space/%zz"`) {
		t.Fatalf("message does not carry the reference: %q", got[0])
	}
}

func TestTheCitedScopeIsNotDecoded(t *testing.T) {
	// `urlsplit` hands the path back as written; only `_catalog_targets`
	// unquotes, and `_cited_scope` never does
	// (`src/brain/wiki/lint.py:389-397`). A reader that decoded here
	// would print `a b` where the page says `a%20b`, and the reader
	// could not search the page for the string the finding names.
	areas := []config.Area{
		{Scope: "knowledge", Path: "k", WikiPath: "k", Shared: true},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {cites("a.md", "brain://project/x/a%20b")},
	}
	got := messagesOf(Federation(bundles, areas, config.ArtifactLookup{}), "wrong-direction")
	if len(got) != 1 || !strings.Contains(got[0], `"project/x/a%20b"`) {
		t.Fatalf("messages = %v, want the reference as written", got)
	}
}

func TestAQueryOrFragmentIsNoPartOfTheScope(t *testing.T) {
	// `urlsplit` takes the fragment off the whole reference and then the
	// query, so neither reaches `netloc + path`. It is not cosmetic: a
	// reference into a shared area with either appended would otherwise
	// match no shared scope and be reported, which is the opposite
	// verdict. `url.Parse` did this for free; a hand-written split has
	// to do it on purpose.
	areas := []config.Area{
		{Scope: "knowledge", Path: "k", WikiPath: "k", Shared: true},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {
			cites("a.md", "brain://knowledge?x=1"),
			cites("b.md", "brain://knowledge#section"),
		},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("a query or fragment hid the shared scope: %v", got)
	}
}

func TestTheSchemeIsReadWithoutRegardToCase(t *testing.T) {
	// `urlsplit` folds the scheme, so `BRAIN://` names a scope on the
	// Python side and is reported there. The hand-written split has to
	// fold it too, or a guard of this severity goes silent on a spelling
	// nothing forbids.
	areas := []config.Area{
		{Scope: "knowledge", Path: "k", WikiPath: "k", Shared: true},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {cites("a.md", "BRAIN://project/space/y")},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 1 {
		t.Fatalf("an upper-case scheme was not read: %v", got)
	}
}

func TestATrailingSlashIsNoPartOfTheScope(t *testing.T) {
	// Python strips at both ends (`.strip("/")`). The verdict survives a
	// front-only trim on its own -- `knowledge/` still passes the prefix
	// arm against `knowledge/` -- and the claim that it does not was
	// wrong: the mutant that trims only the front outlived the first
	// form of this fixture. What the trailing trim decides is the
	// message, and the message is the name a reader has to look up, so
	// the second page below is the half that holds it.
	areas := []config.Area{
		{Scope: "knowledge", Path: "k", WikiPath: "k", Shared: true},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {
			cites("a.md", "brain://knowledge/"),
			cites("b.md", "brain://project/space/"),
		},
	}
	got := messagesOf(Federation(bundles, areas, config.ArtifactLookup{}), "wrong-direction")
	if len(got) != 1 {
		t.Fatalf("messages = %v, want one -- the shared scope is not "+
			"a finding whatever it is spelt with", got)
	}
	if !strings.Contains(got[0], `"project/space"`) {
		t.Fatalf("message carries the trailing slash: %q", got[0])
	}
}

func TestABrokenManifestDoesNotSilenceWrongDirection(t *testing.T) {
	// The two rules are independent, and only one of them reads a
	// manifest. A signpost whose declaration cannot be read stops
	// `unlisted-area` alone; the citations of that same area are still
	// judged, because nothing about them depends on the hub folder.
	areaPath, wikiPath := vault(t)
	write(t, areaPath, ".loomux/config.toml", "[area\n")
	other := t.TempDir()
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "project/space", Path: other, WikiPath: other},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {cites("a.md", "brain://project/space/topics/y")},
	}
	got := Federation(bundles, areas, config.ArtifactLookup{})
	if len(got) != 1 || got[0].Rule != "wrong-direction" {
		t.Fatalf("findings = %+v, want one wrong-direction", got)
	}
}

func TestAnUnusableHubSilencesTheSignpostToo(t *testing.T) {
	// The other half of the same defect as the broken manifest above. A
	// declaration that reads fine and names a hub folder outside the
	// vault builds a pointer that can hit nothing, and every area whose
	// wiki lies outside the vault is then reported -- the same wrong
	// accusation at the same grade, through a second door. Python
	// refuses the value instead of using it
	// (`src/brain/manifest.py:96-130`), so the rule never runs there
	// either.
	//
	// `TestAHubPointerNamesAnAreaWhoseWikiLiesElsewhere` is the
	// counter-probe: a usable hub folder leaves the rule running.
	areaPath, wikiPath := vault(t)
	write(t, areaPath, ".loomux/config.toml",
		"[area]\nscope = \"knowledge\"\n\n[layout]\nhub = \"../x\"\n")
	elsewhere := t.TempDir()
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "project/ultra-brain", Path: elsewhere,
			WikiPath: filepath.Join(elsewhere, "docs", "wiki")},
	}
	bundles := map[string][]wiki.WikiPage{"knowledge": {catalog()}}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("an unusable hub folder let the rule speak: %v", got)
	}
}

func TestOnlyASubjectIsJudgedThoughTheMapHoldsMore(t *testing.T) {
	// The subjects say who is asked, the map says what was read, and the
	// two are not the same question. A bundle run hands in one bundle
	// today, so a rule that took the map for the subject list would
	// answer the same -- until a caller hands in more than it asks
	// about, which is the input this test is.
	areas := []config.Area{
		{Scope: "shared/one", Path: "a", WikiPath: "a", Shared: true},
		{Scope: "shared/two", Path: "b", WikiPath: "b", Shared: true},
		{Scope: "project/x", Path: "c", WikiPath: "c"},
	}
	bundles := map[string][]wiki.WikiPage{
		"shared/one": {cites("a.md", "brain://project/x/y")},
		"shared/two": {cites("b.md", "brain://project/x/y")},
	}
	got := messagesOf(
		FederationFor(areas[:1], bundles, areas, config.ArtifactLookup{}), "wrong-direction")
	if len(got) != 1 {
		t.Fatalf("messages = %v, want the subject's one", got)
	}
	scopes := map[string]bool{}
	for _, f := range FederationFor(areas[:1], bundles, areas, config.ArtifactLookup{}) {
		scopes[f.Scope] = true
	}
	if scopes["shared/two"] {
		t.Fatal("an area that was read but not asked about was judged")
	}
}

func TestAReferenceThatIsOnlyAFragmentOrAQueryNamesNoScope(t *testing.T) {
	// `urlsplit` cuts a fragment and a query even when they start the
	// reference, which leaves nothing to name. Found by the mutation round
	// of 2026-09-23: only cuts further in were held.
	areas := []config.Area{{Scope: "knowledge", Path: "k", WikiPath: "k", Shared: true}}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {cites("a.md", "brain:#x"), cites("b.md", "brain:?q")},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("a reference naming nothing was reported: %v", got)
	}
}

func TestAWikiIsNamedWithoutRegardToCase(t *testing.T) {
	// Windows compares paths without regard to case, and so do Python's
	// `WindowsPath` and the sweep's `wiki.LinkNames`: a registry entry
	// spelt in another case than the catalog link is the same folder.
	areaPath, wikiPath := vault(t)
	other := filepath.Join(areaPath, "92 Engineering", "python")
	areas := []config.Area{
		signpost(areaPath, wikiPath),
		{Scope: "engineering/python", Path: other, WikiPath: strings.ToUpper(other)},
	}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {catalog(linkTo(wikiPath, filepath.Join(other, "index.md")))},
	}
	if got := Federation(bundles, areas, config.ArtifactLookup{}); len(got) != 0 {
		t.Fatalf("a link spelt in another case did not name the wiki: %v", got)
	}
}

func TestAReadOnlySignpostReadsItsHubFromTheStateDirectory(t *testing.T) {
	// A read-only area keeps its declaration under the state directory
	// (`config.ManifestDir`); the one in its own tree is not the one
	// that counts, and here there is none.
	areaPath, wikiPath := vault(t)
	state := t.TempDir()
	write(t, state, "areas/knowledge/.loomux/config.toml",
		"[area]\nscope = \"knowledge\"\n\n[layout]\nhub = \"91 P\"\n")
	elsewhere := t.TempDir()
	post := signpost(areaPath, wikiPath)
	post.ReadOnly = true
	areas := []config.Area{post, {Scope: "project/ultra-brain", Path: elsewhere,
		WikiPath: filepath.Join(elsewhere, "docs", "wiki")}}
	bundles := map[string][]wiki.WikiPage{
		"knowledge": {catalog(linkTo(wikiPath, filepath.Join(areaPath, "91 P", "ultra-brain.md")))},
	}
	lookup := config.ArtifactLookup{Primary: state}
	if got := Federation(bundles, areas, lookup); len(got) != 0 {
		t.Fatalf("the hub of the state directory's declaration was not read: %v", got)
	}
}
