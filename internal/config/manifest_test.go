package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/testlock"
)

func TestTheVocabularyComesFromTheManifest(t *testing.T) {
	dir := t.TempDir()
	write(t, manifestIn(t, dir), "[area]\nscope = \"k\"\n\n[wiki]\ntypes = [\"Decision\"]\n")
	m, err := ReadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, core := range []string{"Source", "Topic", "Entity", "Synthesis"} {
		if !m.KnowsType(core) {
			t.Errorf("core type %q unknown; the four types of the schema are always valid", core)
		}
	}
	if !m.KnowsType("Decision") {
		t.Error("declared type not honoured")
	}
	if m.KnowsType("component") {
		t.Error("`component` was in the hardcoded Go list and must not survive it")
	}
}

func TestATypeIsCompareRatherThanFolded(t *testing.T) {
	// `rank_of` asks `page_type in CORE_TYPES` on the exact string
	// (`src/brain/wiki/types.py`), so `topic` in lower case is unknown to
	// Python and `Architecure` stands out -- which is the whole reason
	// that catalog exists. `KnowsType` used to lower and trim, and was
	// therefore the *weaker* of the two: design 9.3 asks for the
	// opposite, and a page could be spelled wrong here and pass.
	//
	// Measured before the change: not one page of the eight registered
	// wikis carries a type that differs from a `types.py` spelling in
	// case or in padding, so no real verdict moves with this.
	m := &Manifest{DeclaredTypes: []string{"playbook"}}
	for _, known := range []string{"Decision", "Open Question", "Topic", "playbook"} {
		if !m.KnowsType(known) {
			t.Errorf("%q is spelled as the catalog spells it and was refused", known)
		}
	}
	for _, wrong := range []string{"decision", "open question", "topic", "Playbook", " Topic ", "TOPIC"} {
		if m.KnowsType(wrong) {
			t.Errorf("%q is not how the catalog spells it and was accepted", wrong)
		}
	}
}

func TestADeclaredTypeIsNotTrimmedEither(t *testing.T) {
	// The declared half of `rank_of` is the same literal `in`, and a
	// manifest may write padding into its array. Measured on the Python
	// side: `rank_of("playbook", frozenset({" playbook "}))` is unknown
	// and `rank_of(" playbook ", frozenset({" playbook "}))` is
	// `declared` -- so the padding belongs to the name, and trimming one
	// side would let two different declarations answer for each other.
	m := &Manifest{DeclaredTypes: []string{" playbook "}}
	if !m.KnowsType(" playbook ") {
		t.Error("the declared spelling itself was refused")
	}
	if m.KnowsType("playbook") {
		t.Error("a type the manifest did not declare was accepted after trimming")
	}
}

func TestUntouchedDaysDefaultsTo180(t *testing.T) {
	dir := t.TempDir()
	write(t, manifestIn(t, dir), "[area]\nscope = \"k\"\n")
	m, err := ReadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.UntouchedDays != 180 {
		t.Fatalf("UntouchedDays = %d, want 180 (Scheibe 3 §4)", m.UntouchedDays)
	}
}

func TestADeclaredTypeOutsideTheCatalogIsKnown(t *testing.T) {
	// `Decision` above does not test the declared path: it is a CORE type
	// and stays known even when [wiki] types is ignored entirely --
	// measured, with the loop over DeclaredTypes cut out. `playbook` is in
	// none of the three sets of types.py, so only the declaration can carry it.
	dir := t.TempDir()
	write(t, manifestIn(t, dir), "[area]\nscope = \"k\"\n\n[wiki]\ntypes = [\"playbook\"]\n")
	m, err := ReadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !m.KnowsType("playbook") {
		t.Error("a type only the area declares must be known")
	}
	if m.KnowsType("runsheet") {
		t.Error("an undeclared type outside the catalog must stay unknown")
	}
}

func TestTheLayoutTripleIsRead(t *testing.T) {
	dir := t.TempDir()
	write(t, manifestIn(t, dir),
		"[area]\nscope = \"k\"\n\n[layout]\nwiki = \"docs/wiki\"\nhub = \"hub\"\nreview = \"docs/review\"\n\n[wiki]\nuntouched_days = 30\n")
	m, err := ReadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.LayoutWiki != "docs/wiki" || m.LayoutHub != "hub" || m.LayoutReview != "docs/review" {
		t.Errorf("layout = %q/%q/%q, want docs/wiki/hub/docs/review", m.LayoutWiki, m.LayoutHub, m.LayoutReview)
	}
	if m.UntouchedDays != 30 {
		t.Errorf("UntouchedDays = %d; a declared value must beat the default", m.UntouchedDays)
	}
}

func TestAMissingManifestIsAnError(t *testing.T) {
	// Not a Manifest with empty fields: the caller would then lint an area
	// that never declared itself against the bare catalog and call that
	// vocabulary the area's own.
	dir := t.TempDir()
	m, err := ReadManifest(dir)
	if err == nil {
		t.Fatalf("ReadManifest = %+v, want an error when no manifest exists", m)
	}
	if !strings.Contains(err.Error(), "config.toml") {
		t.Errorf("error %q does not name the place searched", err)
	}
}

func TestBrokenTomlIsAnError(t *testing.T) {
	dir := t.TempDir()
	write(t, manifestIn(t, dir), "[area\nscope = \"k\"\n")
	m, err := ReadManifest(dir)
	if err == nil {
		t.Fatalf("ReadManifest = %+v, want an error for a file that is not TOML", m)
	}
	if !strings.Contains(err.Error(), "config.toml") {
		t.Errorf("error %q does not name the broken file", err)
	}
}

// A config.toml that is a regular file and does not read is an error: a
// caller that took the failure for "nothing declared" would fall back to
// defaults the area never chose.
func TestAConfigTomlThatCannotBeReadIsAnErrorNotAnAbsence(t *testing.T) {
	dir := t.TempDir()
	path := manifestIn(t, dir)
	write(t, path, "[area]\nscope = \"k\"\n\n[privacy]\nmode = \"local_only\"\n")
	testlock.Lock(t, path)
	m, err := ReadManifest(dir)
	if err == nil {
		t.Fatalf("ReadManifest = %+v, want an error for a config.toml that cannot be read", m)
	}
	if errors.Is(err, ErrNoManifest) {
		t.Errorf("error %q is ErrNoManifest; a file that exists is not an absent declaration", err)
	}
	if !strings.Contains(err.Error(), "config.toml") {
		t.Errorf("error %q does not name the file that cannot be read", err)
	}
}

// A directory named config.toml is not a file to is_file(), so it declares
// nothing.
func TestAConfigTomlDirectoryIsNoManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := readManifestErr(t, dir); !errors.Is(err, ErrNoManifest) {
		t.Errorf("error %v, want ErrNoManifest for a config.toml directory", err)
	}
}

// write fails the test instead of returning the error, so a fixture that
// cannot be written is reported as itself and not as a missing manifest.
func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// manifestIn creates the .loomux directory under dir and answers the path of
// the manifest inside it, which the test then writes.
func manifestIn(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, manifestNames[0])
}

func TestAMissingManifestIsToldFromABrokenOne(t *testing.T) {
	// The two errors have to be distinguishable without reading their
	// text. `house/unlisted-area` runs on an area that never declared a
	// hub folder and must *not* run on one whose declaration it could
	// not read -- and a caller matching on a message would break the day
	// the wording changes.
	missing := readManifestErr(t, t.TempDir())
	if !errors.Is(missing, ErrNoManifest) {
		t.Errorf("a missing manifest is not ErrNoManifest: %v", missing)
	}
	dir := t.TempDir()
	write(t, manifestIn(t, dir), "[area\n")
	broken := readManifestErr(t, dir)
	if errors.Is(broken, ErrNoManifest) {
		t.Errorf("a broken manifest passes for a missing one: %v", broken)
	}
}

// readManifestErr is the error of a read that has to fail.
func readManifestErr(t *testing.T, dir string) error {
	t.Helper()
	m, err := ReadManifest(dir)
	if err == nil {
		t.Fatalf("ReadManifest(%q) = %+v, want an error", dir, m)
	}
	return err
}

// hubCases are the values `hub_layout` was measured against on the Python
// side, with its answer beside each. Refused values carry the fragment of
// the complaint that says *why*, because the two reasons send the reader
// to different repairs: one is a spelling, the other a place.
var hubCases = []struct {
	value  string
	want   string
	reason string
}{
	{value: "", want: ""},
	{value: "91 Projekte", want: "91 Projekte"},
	{value: ".", want: "."},
	{value: "a/b", want: "a/b"},
	{value: "a/./b", want: "a/./b"},
	// Drive-relative and absolute are not the same thing: neither
	// flavour calls `C:hub` absolute, so Python lets it through.
	{value: "C:hub", want: "C:hub"},
	{value: `..\evil`, reason: "forward slashes"},
	{value: `docs\hub`, reason: "forward slashes"},
	{value: "/srv/hub", reason: "inside the vault"},
	{value: "//server/share", reason: "inside the vault"},
	{value: "C:/hub", reason: "inside the vault"},
	{value: "../x", reason: "inside the vault"},
	{value: "a/../b", reason: "inside the vault"},
	{value: "a/..", reason: "inside the vault"},
	// The forms the second mutation round asked for, every answer
	// measured against `hub_layout` and none of them guessed. A drive is
	// exactly one character before `:/`, whatever that character is --
	// pathlib asks for no letter -- so the first three are refused and
	// the last two are ordinary relative paths.
	{value: "c:/hub", reason: "inside the vault"},
	{value: "1:/x", reason: "inside the vault"},
	{value: "#:/x", reason: "inside the vault"},
	{value: "ä:/x", reason: "inside the vault"},
	// A drive and a separator and nothing else is still rooted, and it
	// is the shortest value that is: the length test has to admit it.
	{value: "C:/", reason: "inside the vault"},
	// And a drive without a separator is the other edge of that test:
	// `PureWindowsPath("C:").is_absolute()` is false, so Python keeps
	// it and so must this. Without the case the length guard can be
	// tightened by one and no test notices.
	{value: "C:", want: "C:"},
	{value: ":/x", want: ":/x"},
	{value: "ab:/c", want: "ab:/c"},
	// `ab/c` has a letter at the front and a slash in the third place
	// and is no drive at all; `a..b` and `...` carry `..` as text and
	// not as a path component.
	{value: "ab/c", want: "ab/c"},
	{value: "a..b", want: "a..b"},
	{value: "...", want: "..."},
}

func TestTheHubValueIsRefusedTheWayPythonRefusesIt(t *testing.T) {
	// `src/brain/manifest.py:96-130`, measured value by value rather than
	// read: a backslash first, so that the complaint names the spelling
	// rather than the place -- a posix split hides both `..` and the root
	// behind one, and neither flavour calls `\srv\hub` absolute.
	//
	// There is no root check, unlike `[layout] wiki`: a vault keeping its
	// hub pages at its own root is unusual, not broken
	// (`src/brain/manifest.py:127-129`).
	for _, c := range hubCases {
		m := &Manifest{LayoutHub: c.value}
		got, err := m.HubLayout()
		if c.reason == "" {
			if err != nil || got != c.want {
				t.Errorf("HubLayout(%q) = %q, %v; want %q, no error",
					c.value, got, err, c.want)
			}
			continue
		}
		if err == nil {
			t.Errorf("HubLayout(%q) = %q, want a refusal", c.value, got)
			continue
		}
		if !strings.Contains(err.Error(), c.reason) {
			t.Errorf("HubLayout(%q) refused with %q, want the reason %q",
				c.value, err, c.reason)
		}
		if got != "" {
			t.Errorf("HubLayout(%q) refused and still answered %q",
				c.value, got)
		}
	}
}

// wikiCases are the values `wiki_layout` was measured against on the
// Python side, with its answer beside each. The table is `hubCases` plus
// the fourth refusal and minus the one value the two readers answer
// differently: `.` is an ordinary hub folder and is the repository root
// for a wiki.
//
// Every answer was read off `src/brain/manifest.py:60-93` running, not
// derived from it.
var wikiCases = []struct {
	value  string
	want   string
	reason string
}{
	{value: "", want: ""},
	{value: "docs/wiki", want: "docs/wiki"},
	{value: "docs/wiki/", want: "docs/wiki/"},
	{value: "a/./b", want: "a/./b"},
	// A blank is a component like any other: `PurePosixPath(" ").parts`
	// is `(' ',)`, so the root test below must not trim before it counts.
	{value: " ", want: " "},
	{value: "C:hub", want: "C:hub"},
	{value: ":/x", want: ":/x"},
	{value: "ab:/c", want: "ab:/c"},
	{value: "a..b", want: "a..b"},
	{value: "...", want: "..."},
	{value: `docs\wiki`, reason: "forward slashes"},
	{value: `\srv\wiki`, reason: "forward slashes"},
	{value: "/srv/wiki", reason: "inside the repository"},
	{value: "//srv/x", reason: "inside the repository"},
	{value: "C:/x", reason: "inside the repository"},
	{value: "1:/x", reason: "inside the repository"},
	{value: "..", reason: "inside the repository"},
	{value: "docs/../x", reason: "inside the repository"},
	{value: "x/..", reason: "inside the repository"},
	// The fourth refusal, the one `hub_layout` does not have. All three
	// forms leave `PurePosixPath(...).parts` empty, and an empty value
	// does not reach it -- it is the unsaid key and answers "" above.
	{value: ".", reason: "not be the repository root"},
	{value: "./", reason: "not be the repository root"},
	{value: "./.", reason: "not be the repository root"},
	// The three the cross-mutation round asked for. Two of them fire
	// two tests at once and so fix an order that would otherwise be
	// free: without them the backslash test may trade places with the
	// escape test, and the escape test with the root test, and every
	// remaining case still passes.
	//
	// The third is not one of those, and the commit that added them
	// said it was. Measured: `hubEscapes` answers false for a lone
	// backslash -- it carries no leading slash, no drive and no `..`
	// component -- so that value fires the backslash test alone. It
	// stays because it is the shortest value that does, and because
	// the claim is cheaper to correct here than to leave standing.
	{value: `/srv\x`, reason: "forward slashes"},
	{value: `\`, reason: "forward slashes"},
	{value: "/", reason: "inside the repository"},
}

func TestTheWikiValueIsRefusedTheWayPythonRefusesIt(t *testing.T) {
	for _, c := range wikiCases {
		m := &Manifest{LayoutWiki: c.value}
		got, err := m.WikiLayout()
		if c.reason == "" {
			if err != nil || got != c.want {
				t.Errorf("WikiLayout(%q) = %q, %v; want %q, no error",
					c.value, got, err, c.want)
			}
			continue
		}
		if err == nil {
			t.Errorf("WikiLayout(%q) = %q, want a refusal", c.value, got)
			continue
		}
		if !strings.Contains(err.Error(), c.reason) {
			t.Errorf("WikiLayout(%q) refused with %q, want the reason %q",
				c.value, err, c.reason)
		}
		if got != "" {
			t.Errorf("WikiLayout(%q) refused and still answered %q",
				c.value, got)
		}
	}
}

func TestAReadOnlyAreaKeepsItsManifestInTheStateDirectory(t *testing.T) {
	// `manifest_path` (src/brain/registry.py:128-134) asks
	// `area_artifact_dir` (`:120-126`), which answers
	// `area_state_dir(state_dir, scope)` for a read-only area and
	// `area.path` for every other. Three of the nine registered areas are
	// read-only, and all three have a manifest in that directory today --
	// counted by listing `%LOCALAPPDATA%\brain\areas\*\.brain.toml`.
	state := filepath.Join("S", "tate")
	owned := Area{Scope: "project/ultra-brain", Path: "P"}
	if got := ManifestDir(owned, state); got != "P" {
		t.Errorf("ManifestDir(owned) = %q, want the area path", got)
	}
	borrowed := Area{Scope: "project/iam-wiki", Path: "P", ReadOnly: true}
	want := filepath.Join(state, "areas", "project-iam-wiki")
	if got := ManifestDir(borrowed, state); got != want {
		t.Errorf("ManifestDir(readonly) = %q, want %q", got, want)
	}
}

func TestAScopeIsFlattenedTheWayTheStateDirectoryIsNamed(t *testing.T) {
	// `_flat` (src/brain/paths.py:38-39) is
	// `re.sub(r"[^A-Za-z0-9_.-]+", "-", scope).strip("-")`, and every
	// answer below was measured against it rather than read off the
	// pattern. The runs of unsafe characters collapse to one dash, and
	// the dashes at both ends come off afterwards -- so a scope of
	// nothing but separators flattens to the empty string, which is the
	// case `read_registry` refuses (`:35`) and this function does not.
	cases := map[string]string{
		"knowledge":          "knowledge",
		"project/iam-wiki":   "project-iam-wiki",
		"engineering/python": "engineering-python",
		"a b":                "a-b",
		"//x//":              "x",
		"-x-":                "x",
		"///":                "",
		"Ä/b":                "b",
		// The values the mutation round asked for. Everything above
		// answers the same whether a *run* of unsafe characters
		// collapses to one dash or every character becomes its own,
		// because none of them carries two in a row. These do, and
		// the `+` of `_UNSAFE` is what they measure.
		"a  b":       "a-b",
		"project//x": "project-x",
		// A dash is *in* the class, so it is not replaced and not
		// part of a run -- the blanks around it become two more.
		"a - b": "a---b",
	}
	for scope, want := range cases {
		if got := flat(scope); got != want {
			t.Errorf("flat(%q) = %q, want %q", scope, got, want)
		}
	}
}

func TestReadManifestCheckLanes(t *testing.T) {
	dir := t.TempDir()
	tomlContent := `
[area]
scope = "project/test"

[check]
lanes = ["gofmt", "pytest"]
`
	if err := os.WriteFile(manifestIn(t, dir), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.Lanes) != 2 {
		t.Fatalf("expected 2 lanes, got %d", len(m.Lanes))
	}
	if m.Lanes[0].Name != "gofmt" || m.Lanes[1].Name != "pytest" {
		t.Errorf("unexpected lanes: %+v", m.Lanes)
	}
}

func TestReadManifestCheckLanesTable(t *testing.T) {
	dir := t.TempDir()
	tomlContent := `
[area]
scope = "project/test"

[check.lanes]
gofmt = "gofmt -l ."
ruff = "uv run ruff check"
`
	if err := os.WriteFile(manifestIn(t, dir), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.Lanes) != 2 {
		t.Fatalf("expected 2 lanes, got %d", len(m.Lanes))
	}
	if m.Lanes[0].Name != "gofmt" || m.Lanes[0].Command != "gofmt -l ." {
		t.Errorf("unexpected lane 0: %+v", m.Lanes[0])
	}
	if m.Lanes[1].Name != "ruff" || m.Lanes[1].Command != "uv run ruff check" {
		t.Errorf("unexpected lane 1: %+v", m.Lanes[1])
	}
}

func TestReadManifestPrivacyDefault(t *testing.T) {
	dir := t.TempDir()
	tomlContent := `
[area]
scope = "project/test"
`
	if err := os.WriteFile(manifestIn(t, dir), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.PrivacyMode != "manual_cloud" {
		t.Errorf("expected default PrivacyMode manual_cloud, got %q", m.PrivacyMode)
	}
	if len(m.NeverGlobs) != 0 {
		t.Errorf("expected empty NeverGlobs, got %v", m.NeverGlobs)
	}
}

func TestReadManifestPrivacyExplicit(t *testing.T) {
	dir := t.TempDir()
	tomlContent := `
[area]
scope = "project/test"

[privacy]
mode = "local_only"
never = ["secrets/**", "keys/*.pem"]
`
	if err := os.WriteFile(manifestIn(t, dir), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.PrivacyMode != "local_only" {
		t.Errorf("expected PrivacyMode local_only, got %q", m.PrivacyMode)
	}
	if len(m.NeverGlobs) != 2 || m.NeverGlobs[0] != "secrets/**" || m.NeverGlobs[1] != "keys/*.pem" {
		t.Errorf("unexpected NeverGlobs: %v", m.NeverGlobs)
	}
}

func TestReadManifestPrivacyInvalidMode(t *testing.T) {
	dir := t.TempDir()
	tomlContent := `
[area]
scope = "project/test"

[privacy]
mode = "invalid_mode"
`
	if err := os.WriteFile(manifestIn(t, dir), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadManifest(dir)
	if err == nil {
		t.Fatal("expected error for invalid privacy mode, got nil")
	}
	if !strings.Contains(err.Error(), "mode must be one of") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestReadManifestPrivacyAutomaticCloud(t *testing.T) {
	dir := t.TempDir()
	tomlContent := `
[area]
scope = "project/test"

[privacy]
mode = "automatic_cloud"
`
	if err := os.WriteFile(manifestIn(t, dir), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.PrivacyMode != "automatic_cloud" {
		t.Errorf("expected PrivacyMode automatic_cloud, got %q", m.PrivacyMode)
	}
}

func TestReadManifestIndexConfig(t *testing.T) {
	dir := t.TempDir()
	tomlContent := `
[area]
scope = "project/test"

[index]
include = ["**/*.md", "README.txt"]
exclude = ["tmp/**"]
unsearched = ["secret/**"]
`
	if err := os.WriteFile(manifestIn(t, dir), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.IndexInclude) != 2 || m.IndexInclude[0] != "**/*.md" {
		t.Errorf("expected IndexInclude [**/*.md, README.txt], got %v", m.IndexInclude)
	}
	if len(m.IndexExclude) != 1 || m.IndexExclude[0] != "tmp/**" {
		t.Errorf("expected IndexExclude [tmp/**], got %v", m.IndexExclude)
	}
	if len(m.IndexUnsearched) != 1 || m.IndexUnsearched[0] != "secret/**" {
		t.Errorf("expected IndexUnsearched [secret/**], got %v", m.IndexUnsearched)
	}
}

func TestTheWikiValuesTheBarrierHeldAreStillJudgedSo(t *testing.T) {
	// Moved from the barrier when it began to read through this package:
	// a drive is one character followed by ":/", and a value of nothing but
	// dots and slashes names the repository root.
	for value, kept := range map[string]bool{
		"C:x": true, "ab:/c": true, ":/x": true, "1:/w": false, "\u00c4:/w": false,
		"C:/": false, "C:/w": false, "/srv/w": false, "//": false,
		".": false, "./": false, ".//": false, "./.": false, "a/../..": false,
	} {
		got, err := (&Manifest{LayoutWiki: value}).WikiLayout()
		if kept && (err != nil || got != value) {
			t.Errorf("WikiLayout(%q) = %q, %v; wanted it kept", value, got, err)
		}
		if !kept && err == nil {
			t.Errorf("WikiLayout(%q) = %q; wanted a refusal", value, got)
		}
	}
}
