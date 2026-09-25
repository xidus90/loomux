// Package config reads the registry and the manifest in which an area
// declares itself.
//
// There are two manifest readers, and the difference is deliberate.
// ReadDeclaration checks a declaration whole and refuses it; brain and the
// write barrier read through it. ReadManifest supplies the few values the
// lint and the post-edit hook need and checks almost nothing, because both
// read any error as "no manifest" and a stricter reader would switch their
// wiki lane off without a word.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
)

// DefaultUntouchedDays is the value a manifest that says nothing gets:
// `src/brain/manifest.py:53` reads `wiki.get("untouched_days", 180)`, and
// `models.Manifest.untouched_days` (src/brain/models.py:20) defaults to the
// same number. `DEFAULT_UNTOUCHED_DAYS` in src/brain/wiki/lint.py is a
// different place with the same number -- the fallback for a bundle with no
// manifest at all (cli.py:1538, gate.py:146) -- and not what a manifest reader
// mirrors.
const DefaultUntouchedDays = 180

// ErrNoManifest is the sentinel of the one error that is not a defect: no
// declaration was found at all. An area may legitimately declare nothing, and
// a caller then falls back to the defaults; a file that exists and cannot be
// read is a different thing entirely, and `house/unlisted-area` has to act on
// the difference. Distinguishing them by message would break the day the
// wording changes, so the sentinel carries it instead.
var ErrNoManifest = errors.New("no manifest found")

// The one manifest name. loomux cut the two spellings of ultra-brain
// (`.ultra-brain/config.toml`, `.brain.toml`) on 2026-09-14;
// `guard.declarationIn` repeats this name for the write barrier.
var manifestNames = []string{filepath.Join(".loomux", "config.toml")}

// knownTypes is the union of CORE_TYPES, CATALOGUE_TYPES and ORIGIN_TYPES in
// src/brain/wiki/types.py, spelled as types.py spells them and asked the
// same way `rank_of` asks: letter for letter. Membership here therefore
// means what `rank_of(...) is not Rank.UNKNOWN` means there.
//
// The spelling is the rule, not an accident of it. types.py opens with the
// reason -- "a typo, `Architecure` should stand out, and without any
// expectation at all it does not" -- and a fold would take back exactly
// that. This map and `KnowsType` folded case and padding until design
// step 8, which made Go silent about a `type: topic` that Python calls an
// error: the weaker of the two sides, where design 9.3 asks for no weaker.
//
// The fold's justification named `pkg/wiki/lint.go` as a caller that
// forces it. That file does not call `KnowsType` -- it carries its own
// `defaultValidTypes` and lowers before consulting it. The one caller in
// the tree is `house.unknownType` (`pkg/check/house/page.go`), which now
// hands the value over untouched.
//
// The four schema types the brief spells `source`, `topic`, `entity`,
// `synthesis` are spelt `Source`, `Topic`, `Entity`, `Synthesis` here,
// because that is how types.py spells them.
var knownTypes = map[string]bool{
	// CORE_TYPES
	"Architecture":  true,
	"Decision":      true,
	"Open Question": true,
	"Reference":     true,
	// CATALOGUE_TYPES
	"API Endpoint":   true,
	"Data Model":     true,
	"Metric":         true,
	"Runbook":        true,
	"Glossary Entry": true,
	// ORIGIN_TYPES
	"Source":    true,
	"Topic":     true,
	"Entity":    true,
	"Synthesis": true,
}

// LaneConfig describes one verification lane declared under [check] lanes.
type LaneConfig struct {
	Name    string
	Command string
}

// Manifest holds the part of an area's declaration the Go checks read.
type Manifest struct {
	// Path is the file the declaration was read from; refusals name it.
	Path            string
	Scope           string
	DeclaredTypes   []string
	UntouchedDays   int
	LayoutWiki      string
	LayoutHub       string
	LayoutReview    string
	LayoutInbox     string
	Lanes           []LaneConfig
	PrivacyMode     string
	NeverGlobs      []string
	IndexInclude    []string
	IndexExclude    []string
	IndexUnsearched []string
	// OnMerge is the area's consent to have merges recorded (`[maintenance]
	// on_merge`); MergeBranch the branch whose merges count, "main" when the
	// declaration names none, as the reference's reader has it.
	OnMerge     bool
	MergeBranch string
}

// DefaultMergeBranch is the branch whose merges count where a declaration
// names none, as `manifest.py:41-43` of the reference has it.
const DefaultMergeBranch = "main"

// mergeBranch answers the declared branch, or the default for an unsaid one.
func mergeBranch(declared string) string {
	if declared == "" {
		return DefaultMergeBranch
	}
	return declared
}

// manifestFile is the wire shape of the file. It is separate from Manifest
// because the two are not the same thing: the file may leave keys out, and the
// defaults are applied after decoding.
type manifestFile struct {
	Area struct {
		Scope string `toml:"scope"`
	} `toml:"area"`
	Wiki struct {
		Types         []string `toml:"types"`
		UntouchedDays int      `toml:"untouched_days"`
	} `toml:"wiki"`
	Layout struct {
		Wiki   string `toml:"wiki"`
		Hub    string `toml:"hub"`
		Review string `toml:"review"`
	} `toml:"layout"`
	Check struct {
		Lanes toml.Primitive `toml:"lanes"`
	} `toml:"check"`
	Privacy struct {
		Mode  string   `toml:"mode"`
		Never []string `toml:"never"`
	} `toml:"privacy"`
	Index struct {
		Include    []string `toml:"include"`
		Exclude    []string `toml:"exclude"`
		Unsearched []string `toml:"unsearched"`
	} `toml:"index"`
	Maintenance struct {
		OnMerge bool   `toml:"on_merge"`
		Branch  string `toml:"branch"`
	} `toml:"maintenance"`
}

// ReadManifest reads the manifest of the area rooted at repoRoot.
func ReadManifest(repoRoot string) (*Manifest, error) {
	return readManifestAmong(repoRoot, manifestNames)
}

// readManifestAmong reads the first of names below dir that is a regular
// file, for ReadManifest. It decodes into the typed wire shape and checks
// only the privacy mode: the lint and the post-edit hook read a declaration
// as "none" on any error, and a stricter reader would switch their wiki lane
// off silently. The write barrier refuses a broken area declaration visibly.
func readManifestAmong(dir string, names []string) (*Manifest, error) {
	for _, name := range names {
		path := filepath.Join(dir, name)
		// Read first and stat only on failure, which is the order that makes
		// the distinction Python draws: `registry.manifest_path` picks this
		// file by `is_file()` and `read_manifest` then fails on it, so a
		// regular file that does not read -- denied permissions, a lock another
		// process holds -- is an error here too, and never ErrNoManifest. Taken
		// for an absence, it would open a closed area wherever the caller's next
		// step is a visibility decision: a `local_only` area would fall back to
		// the defaults it never chose. The strictness is half of that repair;
		// the other half is privacy.VisibleManifest, because a caller that
		// discards the error is back where it started.
		//
		// Anything but a regular file counts as no manifest, and `is_file()` is
		// again the rule: a directory of that name declares nothing, as does a
		// name that does not exist.
		data, err := os.ReadFile(path)
		if err != nil {
			if info, statErr := os.Stat(path); statErr == nil && info.Mode().IsRegular() {
				return nil, fmt.Errorf("%s: cannot be read: %w", path, err)
			}
			continue
		}
		file := manifestFile{}
		// Set before decoding: toml overwrites only the keys the file names,
		// so an unsaid `untouched_days` keeps this value.
		file.Wiki.UntouchedDays = DefaultUntouchedDays
		file.Privacy.Mode = "manual_cloud"
		meta, err := toml.Decode(string(data), &file)
		if err != nil {
			return nil, fmt.Errorf("%s: not valid TOML: %w", path, err)
		}
		if file.Privacy.Mode != "local_only" && file.Privacy.Mode != "manual_cloud" && file.Privacy.Mode != "automatic_cloud" {
			return nil, fmt.Errorf("%s: [privacy] mode must be one of automatic_cloud, local_only, manual_cloud, found %q", path, file.Privacy.Mode)
		}

		var lanes []LaneConfig
		var list []string
		if err := meta.PrimitiveDecode(file.Check.Lanes, &list); err == nil {
			for _, name := range list {
				lanes = append(lanes, LaneConfig{Name: name})
			}
		} else {
			var table map[string]string
			if err := meta.PrimitiveDecode(file.Check.Lanes, &table); err == nil {
				for name, cmd := range table {
					lanes = append(lanes, LaneConfig{Name: name, Command: cmd})
				}
				sort.Slice(lanes, func(i, j int) bool {
					return lanes[i].Name < lanes[j].Name
				})
			}
		}

		return &Manifest{
			Path:            path,
			Scope:           file.Area.Scope,
			DeclaredTypes:   file.Wiki.Types,
			UntouchedDays:   file.Wiki.UntouchedDays,
			LayoutWiki:      file.Layout.Wiki,
			LayoutHub:       file.Layout.Hub,
			LayoutReview:    file.Layout.Review,
			Lanes:           lanes,
			PrivacyMode:     file.Privacy.Mode,
			NeverGlobs:      file.Privacy.Never,
			IndexInclude:    file.Index.Include,
			IndexExclude:    file.Index.Exclude,
			IndexUnsearched: file.Index.Unsearched,
			OnMerge:         file.Maintenance.OnMerge,
			MergeBranch:     mergeBranch(file.Maintenance.Branch),
		}, nil
	}
	return nil, fmt.Errorf("%s: %w (%s)", dir, ErrNoManifest,
		strings.Join(names, ", "))
}

// KnowsType reports whether this area accepts t as a page type.
//
// Both comparisons are literal, which is the pair `rank_of` makes:
// `page_type in CORE_TYPES` and then `page_type in declared`, neither of
// them normalised on either side.
func (m *Manifest) KnowsType(t string) bool {
	if knownTypes[t] {
		return true
	}
	for _, declared := range m.DeclaredTypes {
		if declared == t {
			return true
		}
	}
	return false
}

// HubLayout is where the vault keeps its hub pages, checked, or the empty
// string when the manifest says nothing. It is `hub_layout` of
// `src/brain/manifest.py:96-130`, and it is a method rather than a field
// because Python checks the value where it is used and not where the file is
// read: `read_manifest` copies `[layout]` through untouched, and only the one
// caller that builds a hub pointer asks for it (`src/brain/cli.py:1547`). A
// reader that refused the whole declaration here would stop an area from
// being linted at all over a key no other rule reads.
//
// An unusable value is refused rather than used, and the reason is the one
// `hub_layout`'s own docstring gives (`:99-108`): the pointer is joined onto
// the signpost's root, a value that escapes builds a page no catalog can
// name, and `unlisted-area` then reports every area whose wiki lies outside
// the vault -- "So der Lauf geht rot an den Bereichen, die nichts falsch
// gemacht haben, statt an der einen Zeile, die es tat."
//
// The three tests are Python's, in Python's order, and the order is what
// makes the complaint useful. The backslash comes first because a posix
// split hides both `..` and the root behind one, and because neither flavour
// of path calls `\srv\hub` absolute -- without this test such a value would
// pass all the others (`:113-117`).
//
// Absolute is asked in both flavours for the reason stated at `:120-122`:
// `PurePosixPath("C:/x").is_absolute()` is False, so a posix test alone
// would pass a drive letter, and `PureWindowsPath("/srv/x").is_absolute()`
// is False, so a windows test alone would pass a rooted path. With the
// backslash already refused, the windows half reduces to a drive followed
// by a separator.
//
// There is no root check, unlike `[layout] wiki`, and `:127-129` says why: a
// vault that keeps its hub pages directly at its own root is an unusual
// layout, not a broken one.
//
// Measured value by value against `hub_layout` rather than derived from its
// source; the fourteen values and their answers stand in
// `manifest_test.go`.
func (m *Manifest) HubLayout() (string, error) {
	value := m.LayoutHub
	// No arm for the unsaid key: an empty value carries no backslash, is
	// not absolute and splits into one part that is not `..`, so it falls
	// through to the same answer an arm would have returned. The arm was
	// written and struck out again -- its mutant survived, because there
	// is nothing for it to decide.
	if strings.Contains(value, `\`) {
		return "", fmt.Errorf(
			"[layout] hub must use forward slashes, found %q", value)
	}
	if hubEscapes(value) {
		return "", fmt.Errorf(
			"[layout] hub must stay inside the vault, found %q", value)
	}
	return value, nil
}

// hubEscapes says whether a hub value leaves the vault: absolute in either
// flavour of path, or carrying a `..` component. Backslashes are the
// caller's to refuse first -- this function reads posix, and a value spelt
// with backslashes would hide both answers from it.
//
// A drive is one character before `:/`, and pathlib does not ask that the
// character be a letter. Measured, not assumed: `hub_layout` refuses
// `1:/x`, `#:/x` and a value beginning with an umlaut exactly as it refuses
// `C:/x`, while `:/x` (no character) and `ab:/c` (two) come back
// untouched. A letter test stood here first and was the narrower answer --
// it let a rooted value through, and the mutation round is what asked the
// question that found it.
//
// The first *rune*, not the first byte: a one-character drive spelt outside
// ASCII is two bytes here and one character to pathlib, and indexing bytes
// would answer differently for the one form nobody writes. `DecodeRune`
// answers `RuneError` with size 1 on invalid UTF-8, which then fails the
// length test below on any value short enough to matter and is judged by
// the `..` arm like any other text.
//
// Only the `..` component counts, never `..` as text: `PurePosixPath` hands
// out parts, so `a..b` and `...` are ordinary names. The `.` component is
// not refused either -- `a/./b` comes back untouched -- because it joins
// onto a path without moving it.
func hubEscapes(value string) bool {
	if strings.HasPrefix(value, "/") {
		return true
	}
	if _, size := utf8.DecodeRuneInString(value); size > 0 &&
		len(value) >= size+2 && value[size] == ':' &&
		value[size+1] == '/' {
		return true
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

// WikiLayout is where the bundle sits inside its repository, checked, or
// the empty string when the manifest says nothing. It is `wiki_layout` of
// `src/brain/manifest.py:60-93`, and it is a method for the reason
// `HubLayout` gives for being one: Python checks the value where it is
// used, and a reader that refused the whole declaration here would stop
// an area from being read at all over a key no other rule touches.
//
// Repo-relative and nothing else, and the docstring at `:61-66` says why:
// "The registration holds one absolute path per area, and a linked
// worktree has another one; an absolute value here would block writing on
// the branch, which is the reason the wiki moved into the repo at all."
// That is also what makes this the value `CheckFile` finds a bundle root
// with: it is the one statement of the root that a worktree does not
// invalidate.
//
// Three of the four refusals are `HubLayout`'s, in the same order and for
// the same reasons, so `hubEscapes` answers two of them here as well --
// `wiki_layout` and `hub_layout` spell those three tests identically
// (`:76-92` against `:113-126`) and only the complaint differs.
//
// The fourth has no counterpart next door: a value whose parts are empty
// names the repository root, and `:91-92` refuses it for a wiki where
// `hub_layout` lets it stand -- `:127-129` is that function's comment
// saying it has no such check. `.`, `./` and `./.` are that case measured;
// the empty value never reaches the test, because an unsaid key is
// answered above it.
//
// `[layout] review` is read by neither method. Nothing in this check
// catalog reads it, and a validator for a key with no reader would be a
// rule nobody could see fail.
//
// Measured value by value against `wiki_layout` running rather than
// derived from its source; the twenty-two values and their answers stand
// in `manifest_test.go`.
func (m *Manifest) WikiLayout() (string, error) {
	value := m.LayoutWiki
	// The unsaid key, and unlike in `HubLayout` this arm decides
	// something: without it the empty value falls through to the root
	// test below, which is exactly what it is -- the repository root --
	// and an area that declares no layout would be refused for it.
	if value == "" {
		return "", nil
	}
	if strings.Contains(value, `\`) {
		return "", fmt.Errorf(
			"[layout] wiki must use forward slashes, found %q", value)
	}
	if hubEscapes(value) {
		return "", fmt.Errorf(
			"[layout] wiki must stay inside the repository, found %q",
			value)
	}
	if !namesAPart(value) {
		return "", fmt.Errorf(
			"[layout] wiki must not be the repository root, found %q",
			value)
	}
	return value, nil
}

// namesAPart says whether a value has any path component at all, which is
// `PurePosixPath(value).parts` being non-empty. pathlib drops the empty
// component a doubled or trailing slash leaves and the `.` component that
// joins onto a path without moving it, and keeps everything else -- a
// blank is a name like any other, which is why nothing is trimmed here.
func namesAPart(value string) bool {
	for _, part := range strings.Split(value, "/") {
		if part != "" && part != "." {
			return true
		}
	}
	return false
}

// unsafeInScope is `_UNSAFE` of `src/brain/paths.py:12`. Its comment says
// what it is for: "A scope is a user-supplied string with slashes in it.
// Anything that is not a plain name becomes a dash, so a scope can never
// climb out of the state directory or collide with a path separator on
// either platform."
//
// A negated class matches by rune in Go as in Python, so a scope spelt
// outside ASCII collapses as one character and not as its bytes --
// measured, `Ä/b` flattens to `b` on both sides.
var unsafeInScope = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// flat is `_flat` (`src/brain/paths.py:38-39`): every run of unsafe
// characters becomes one dash, then the dashes at both ends come off.
func flat(scope string) string {
	return strings.Trim(unsafeInScope.ReplaceAllString(scope, "-"), "-")
}

// ManifestDir is the directory an area's manifest is read from, and it is
// not always the area itself: `manifest_path`
// (`src/brain/registry.py:128-134`) asks `area_artifact_dir` (`:120-126`),
// which answers the state directory for a read-only area because "writing
// a catalog into it would be the first forbidden write (spec 5.5,
// decision 36)" -- and the manifest "follows the artefacts, for the same
// reason they moved".
//
// The effect on this check catalog is nil today and the reader deserves
// the number rather than the reassurance: of the nine areas in the
// registration on this machine three are read-only, all three have a
// `.brain.toml` under `%LOCALAPPDATA%\brain\areas\` (counted by listing
// that directory), and not one of the three declares `[wiki] types` or
// `untouched_days` -- so both values it supplies are today the ones an
// absent manifest would have given anyway. It is built all the same,
// because the day one of them declares a type, a reader that looked in
// the area would report `house/unknown-type` on every page carrying it,
// and nothing would say why.
//
// This is only where the manifest is looked for. Which name is preferred
// inside that directory stays `ReadManifest`'s question, which is why
// this returns a directory and not a file -- `manifest_path` answers a
// file and repeats the search order to do so.
func ManifestDir(area Area, stateDir string) string {
	if area.ReadOnly {
		return filepath.Join(stateDir, "areas", flat(area.Scope))
	}
	return area.Path
}
