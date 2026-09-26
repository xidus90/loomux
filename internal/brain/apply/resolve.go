package apply

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// resolved is where a case lies and what it writes into: the part of `_Place`
// (apply.py:280-295) that `_resolve` settles. Every path is kept as spelt,
// not as resolved -- the resolution is the check, as in Python.
type resolved struct {
	vault     string // the nearest ancestor carrying the marker
	review    string // the vault's review centre, `[layout] review`
	wiki      string // the wiki of the case's own area, from the registry
	directory string // the case directory
	area      config.Area
}

// resolve is `_resolve` (apply.py:361-409) short of `_preflight`, which
// needs the place the caller builds from this.
//
// Two anchors from two sources. The vault is the tree the case lies in, so
// it is found by walking up; its `[layout] review` says whether the case
// really lies in the review centre. The wiki belongs to `c.Area`, which the
// vault's manifest cannot answer for: the review centre is vault-wide, so
// the vault of a case is routinely some other area's. Only the registry
// maps an area to a wiki.
func resolve(casePath string, c maintenance.Case, areas []config.Area) (resolved, error) {
	directory := filepath.Dir(casePath)
	vault, manifest, err := nearestVault(directory)
	if err != nil {
		return resolved{}, refuse("%s", err)
	}
	if manifest == nil {
		return resolved{}, refuse("%s: no area declaration above this case; there is no vault to write", casePath)
	}
	review, err := layoutEntry(manifest.LayoutReview, manifest.Path, "review")
	if err != nil {
		return resolved{}, err
	}
	review = filepath.Join(vault, filepath.FromSlash(review))
	if !guard.IsRelativeTo(directory, review) {
		return resolved{}, refuse("%s: not inside the review centre %s", casePath, review)
	}
	area, err := wikiArea(c, casePath, areas)
	if err != nil {
		return resolved{}, err
	}
	wiki := filepath.FromSlash(area.WikiPath)
	resolvedWiki, err := resolvePath(wiki)
	if err != nil {
		return resolved{}, err
	}
	resolvedVault, err := resolvePath(vault)
	if err != nil {
		return resolved{}, err
	}
	// A wiki outside the vault is legitimate when the registration names a
	// real bundle, a project area's own repository; every other write then
	// anchors on the wiki (`anchorFor`). One that is no bundle is refused
	// here, by name, rather than at the first scaffold write.
	if !guard.IsRelativeTo(resolvedWiki, resolvedVault) && !isBundle(resolvedWiki) {
		return resolved{}, refuse("%s: the wiki of area %s is %s, outside the vault %s and not a scaffolded bundle",
			casePath, pytext.Repr(c.Area), wiki, vault)
	}
	return resolved{vault: vault, review: review, wiki: wiki, directory: directory, area: area}, nil
}

// nearestVault walks `directory.parents` for `_MANIFEST` (apply.py:193,
// :373-376): the case directory itself is not asked, its ancestors are,
// nearest first. Python's marker is `.brain.toml`; until stage 4 a vault
// declares itself under any name config.ReadAreaManifestUntilStage4 reads,
// loomux's `.loomux/config.toml` first. A `.loomux/config.toml` without
// [area] is policy only and marks no vault, so the walk goes on past it; a
// declaration that is there and does not read is an error, as
// `read_manifest` raises one. No manifest and no error: no vault above.
func nearestVault(directory string) (string, *config.Manifest, error) {
	for dir := directory; filepath.Dir(dir) != dir; {
		dir = filepath.Dir(dir)
		manifest, err := config.ReadAreaManifestUntilStage4(dir)
		if errors.Is(err, config.ErrNoManifest) {
			continue
		}
		return dir, manifest, err
	}
	return "", nil, nil
}

// isFile is `Path.is_file`: a regular file, links followed.
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// layoutEntry is `_layout` (apply.py:656-676): one `[layout]` entry,
// checked before anything is anchored to it. The manifest is synchronised
// vault content, so an absolute value or a `..` would move the very anchor
// the containment checks compare against. A value with a volume name is
// refused too: pathlib would put `C:x` on the drive's own current
// directory, and filepath.Join would glue it onto the vault instead.
func layoutEntry(value, manifestPath, key string) (string, error) {
	if value == "" {
		return "", refuse("%s: declares no [layout] %s; there is nowhere to go", manifestPath, key)
	}
	if !plainRelative(value) {
		return "", refuse("%s: [layout] %s must be relative to the area, found %s",
			manifestPath, key, pytext.Repr(value))
	}
	return value, nil
}

// plainRelative says that value, read with either separator, is a path
// below wherever it is joined: not rooted, no volume, no `..` component.
func plainRelative(value string) bool {
	slashed := strings.ReplaceAll(value, `\`, "/")
	if strings.HasPrefix(slashed, "/") || filepath.IsAbs(value) || filepath.VolumeName(value) != "" {
		return false
	}
	return !slices.Contains(strings.Split(slashed, "/"), "..")
}

// wikiArea is `_wiki` (apply.py:413-434): the registry entry of the case's
// own area, refused by name where the registry does not know it or knows it
// without a wiki -- a decision would otherwise land on another area's page.
func wikiArea(c maintenance.Case, casePath string, areas []config.Area) (config.Area, error) {
	for _, area := range areas {
		if area.Scope != c.Area {
			continue
		}
		if area.WikiPath == "" {
			return config.Area{}, refuse("%s: area %s is registered with no wiki; there is nowhere to go",
				casePath, pytext.Repr(c.Area))
		}
		return area, nil
	}
	return config.Area{}, refuse("%s: case belongs to area %s, which the registry does not register",
		casePath, pytext.Repr(c.Area))
}

// isBundle is `_is_bundle` (apply.py:437-446): `_schema.md` is written by
// the scaffold alone, so it tells a real wiki from a tree the registry
// merely names.
func isBundle(path string) bool {
	return isFile(filepath.Join(path, "_schema.md"))
}

// checkTarget is `_check_target` (apply.py:814-847): the target's form,
// judged for every decision before any path is built from it. The target
// is interpolated into `log.md`, `audit.md` and the commit message, all
// line-structured, so it carries no control, format or line-separator
// character. Surrounding whitespace is refused because Windows trims a
// trailing blank when it opens a file: two case keys, one page.
func checkTarget(target string) error {
	if pytext.Strip(target) == "" {
		return refuse("case target is empty")
	}
	if target != pytext.Strip(target) {
		return refuse("%s: target has surrounding whitespace", pytext.Repr(target))
	}
	for _, char := range target {
		if unicode.In(char, unicode.Cc, unicode.Cf, unicode.Zl, unicode.Zp) {
			return refuse("%s: target contains a control character", pytext.Repr(target))
		}
	}
	return nil
}

// targetPath is `_target` (apply.py:850-900) followed by the one check its
// only caller, `_apply`, makes next (apply.py:740): the page the case points
// at, once it is shown to be inside the wiki, a page and there.
//
// The spelling is checked first, since an absolute target or a `..` would
// replace the wiki prefix and the hash guard cannot catch it; then the name,
// since `audit.md` or the register lies inside the wiki and is no page; then
// the resolved path, since a link inside the wiki carries no `..`; and then
// every component on the way, since a link that stays inside the wiki still
// makes the protocols name a page that was never changed.
func targetPath(r resolved, target string) (string, error) {
	page, err := targetPlace(r, target)
	if err != nil {
		return "", err
	}
	if !isFile(page) {
		return "", refuse("%s: the case's target page is gone", page)
	}
	return page, nil
}

// targetPlace is every check of targetPath but the last: where the page the
// case points at lies, whether or not it is still there. A rejection asks
// this, since a page gone since the case was formed must not keep it open.
func targetPlace(r resolved, target string) (string, error) {
	parts, ok := targetParts(target)
	if !ok {
		return "", refuse("%s: target is not inside the wiki", target)
	}
	for _, part := range parts {
		if isScaffoldName(part) {
			return "", refuse("%s: a scaffold file, not a page a case may change", target)
		}
	}
	page := filepath.Join(append([]string{r.wiki}, parts...)...)
	resolvedPage, err := resolvePath(page)
	if err != nil {
		return "", err
	}
	resolvedWiki, err := resolvePath(r.wiki)
	if err != nil {
		return "", err
	}
	if !guard.IsRelativeTo(resolvedPage, resolvedWiki) {
		return "", refuse("%s: target resolves to a path not inside the wiki", target)
	}
	walked := r.wiki
	for _, part := range parts {
		walked = filepath.Join(walked, part)
		if isLink(walked) {
			return "", refuse("%s: %s is a link, not a page", target, part)
		}
	}
	return page, nil
}

// targetParts is `PurePosixPath(target.replace("\\", "/")).parts` for a
// target that is not absolute and has no `..`, and false for one that is:
// empty and `.` components drop out as pathlib drops them. A volume name
// counts as absolute here, which pathlib does not quite say -- see
// layoutEntry.
func targetParts(target string) ([]string, bool) {
	if !plainRelative(target) {
		return nil, false
	}
	var parts []string
	for _, part := range strings.Split(strings.ReplaceAll(target, `\`, "/"), "/") {
		if part != "" && part != "." {
			parts = append(parts, part)
		}
	}
	return parts, true
}

// sourceFile is `_SourceLocation` (apply.py:145-152) as its readers use it:
// the file to hash, the key of its row and the register the row lives in --
// twice, because until stage 4 a read-only area may still be read from
// ultra-brain's state directory while it is written only to loomux's.
type sourceFile struct {
	docID    string
	relative string // the row's key, slashed, as the register spells it
	path     string // area.Path joined with relative
	readFrom string // the `_identities.tsv` the row was found in
	register string // the `_identities.tsv` an advanced row is written to
}

// registerRead is `area_artifact_dir(area, state_dir) / _REGISTER` as it is
// read: a read-only area's register lies in the state directory, the new
// one first and, as long as that holds nothing of the area, the old one.
func registerRead(area config.Area, lookup config.ArtifactLookup) string {
	return filepath.Join(config.ResolvedAreaDir(area, lookup.Primary, lookup.Fallback), registerName)
}

// registerWrite is the same register as it is written: always under the new
// state directory, where `index` publishes a read-only area's stock too.
// moveStock has to run before the first write into a read-only area that
// is still read from the old place.
func registerWrite(area config.Area, lookup config.ArtifactLookup) string {
	return filepath.Join(config.ManifestDir(area, lookup.Primary), registerName)
}

// registersOf is every registered area's register as it is written,
// present or not, for `place.registers` -- the list
// `_is_external_register` builds, and the paths the barrier lets a
// scaffold write reach outside the vault.
func registersOf(areas []config.Area, lookup config.ArtifactLookup) []string {
	registers := make([]string, 0, len(areas))
	for _, area := range areas {
		registers = append(registers, registerWrite(area, lookup))
	}
	return registers
}

// resolveSources is `_resolve_sources` (apply.py:155-186): every doc_id
// looked up in the registers of all registered areas, in registry order, so
// that a case citing a source of another area -- a project area seen from a
// central vault -- is found where that area keeps it. A doc_id no register
// knows is left out rather than refused, and once every doc_id is found no
// further register is read.
//
// Within one register, a doc_id on two rows goes to the relative that sorts
// first; Python takes the first in file order, which the register reader's
// map does not keep.
func resolveSources(areas []config.Area, lookup config.ArtifactLookup, docIDs []string) (map[string]sourceFile, error) {
	needed := map[string]bool{}
	for _, docID := range docIDs {
		needed[docID] = true
	}
	found := map[string]sourceFile{}
	for _, area := range areas {
		if len(needed) == 0 {
			break
		}
		readFrom := registerRead(area, lookup)
		if !isFile(readFrom) {
			continue
		}
		identities, err := identity.ReadIdentities(readFrom)
		if err != nil {
			return nil, err
		}
		relatives := make([]string, 0, len(identities))
		for relative := range identities {
			relatives = append(relatives, relative)
		}
		slices.Sort(relatives)
		for _, relative := range relatives {
			docID := identities[relative].DocID
			if !needed[docID] {
				continue
			}
			found[docID] = sourceFile{
				docID:    docID,
				relative: relative,
				path:     filepath.Join(area.Path, filepath.FromSlash(relative)),
				readFrom: readFrom,
				register: registerWrite(area, lookup),
			}
			delete(needed, docID)
		}
	}
	return found, nil
}
