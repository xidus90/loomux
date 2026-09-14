package guard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// bundleDir and manifestName name the one manifest loomux reads. ultra-brain
// had a second, legacy spelling; loomux cut it on 2026-09-14.
const (
	bundleDir    = ".loomux"
	manifestName = "config.toml"
)

// privacyModes is `_PRIVACY_MODES` (src/brain/manifest.py:11). A misspelt
// mode must fail loudly: silently falling back would turn the strictest
// setting in the system into the most permissive one.
var privacyModes = []string{"automatic_cloud", "local_only", "manual_cloud"}

// manifest is the part of an area's declaration this barrier reads. Only
// two fields survive the reading, and the rest of `read_manifest` is here
// for its *refusals*: a manifest the Python side rejects has to refuse
// this call too, or the two barriers answer different registries.
type manifest struct {
	scope  string
	layout map[string]any
}

// table is `data.get(key, {})` followed by a method call on the result.
// Python raises AttributeError where the value is not a mapping, and
// `decide`'s catch turns that into a refusal -- so this reports the same
// failure rather than treating a stray string as an empty table.
func table(document map[string]any, key string) (map[string]any, bool) {
	value, present := document[key]
	if !present {
		return map[string]any{}, true
	}
	nested, ok := value.(map[string]any)
	return nested, ok
}

// errNoArea says that a `.loomux/config.toml` carries no `[area]` table.
// That file is the one project configuration, with a section per module,
// and a project that uses only the policy declares no brain area: every
// caller reads this as a missing manifest. It is no failure to read -- a
// broken file, or an `[area]` without a scope, still answers an ordinary
// error and still refuses.
var errNoArea = errors.New("the configuration declares no [area]")

// readManifest is `read_manifest` (src/brain/manifest.py:18-56), checks
// and order included. The order is what decides which complaint a broken
// file produces, and Python's is the constructor's argument order: scope,
// privacy mode, wiki types, on_merge, branch, then layout, the three
// index globs, the privacy globs and untouched_days.
func readManifest(path string) (manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, err
	}
	document := map[string]any{}
	if err := toml.Unmarshal(data, &document); err != nil {
		return manifest{}, fmt.Errorf("%s: not valid TOML: %w", path, err)
	}
	// Asked before `table`, which answers an absent key as an empty table
	// and would turn a file without an area into one without a scope.
	if _, present := document["area"]; !present {
		return manifest{}, errNoArea
	}
	area, ok := table(document, "area")
	if !ok {
		return manifest{}, fmt.Errorf("%s: [area] must be a table", path)
	}
	scope, _ := area["scope"].(string)
	if scope == "" {
		return manifest{}, fmt.Errorf(
			"%s: [area] scope is required and must be a non-empty string",
			path)
	}
	index, indexIsTable := table(document, "index")
	privacy, ok := table(document, "privacy")
	if !ok {
		return manifest{}, fmt.Errorf("%s: [privacy] must be a table", path)
	}
	mode := any("manual_cloud")
	if declared, present := privacy["mode"]; present {
		mode = declared
	}
	if text, ok := mode.(string); !ok || !contains(privacyModes, text) {
		return manifest{}, fmt.Errorf(
			"%s: [privacy] mode must be one of %s, found %s",
			path, strings.Join(sorted(privacyModes), ", "), pyRepr(mode))
	}
	wiki, ok := table(document, "wiki")
	if !ok {
		return manifest{}, fmt.Errorf("%s: [wiki] must be a table", path)
	}
	if !isStringList(wiki["types"]) {
		return manifest{}, fmt.Errorf(
			"%s: [wiki] types must be a list of strings", path)
	}
	maintenance, ok := table(document, "maintenance")
	if !ok {
		return manifest{}, fmt.Errorf(
			"%s: [maintenance] must be a table", path)
	}
	if merge, present := maintenance["on_merge"]; present {
		if _, ok := merge.(bool); !ok {
			return manifest{}, fmt.Errorf(
				"%s: [maintenance] on_merge must be a boolean, found %s",
				path, pyRepr(merge))
		}
	}
	if branch, present := maintenance["branch"]; present {
		if text, ok := branch.(string); !ok || text == "" {
			return manifest{}, fmt.Errorf(
				"%s: [maintenance] branch must be a non-empty string", path)
		}
	}
	layout, ok := table(document, "layout")
	if !ok {
		return manifest{}, fmt.Errorf("%s: [layout] must be a table", path)
	}
	if !indexIsTable {
		return manifest{}, fmt.Errorf("%s: [index] must be a table", path)
	}
	for _, key := range []string{"include", "exclude", "unsearched"} {
		if err := globs(path, "index", key, index[key]); err != nil {
			return manifest{}, err
		}
	}
	if err := globs(path, "privacy", "never", privacy["never"]); err != nil {
		return manifest{}, err
	}
	if days, present := wiki["untouched_days"]; present {
		// `bool` passes as an `int` in Python, which is excluded
		// explicitly there: `untouched_days = true` would otherwise
		// become a threshold of one day (src/brain/manifest.py:138-140).
		number, ok := days.(int64)
		if _, isBool := days.(bool); !ok || isBool || number < 1 {
			return manifest{}, fmt.Errorf(
				"%s: [wiki] untouched_days must be an integer >= 1", path)
		}
	}
	return manifest{scope: scope, layout: layout}, nil
}

// globs is `_globs` (src/brain/manifest.py:164-183). The section is
// passed in rather than assumed: `never` lives under `[privacy]`, and a
// message that names `[index]` sends the reader looking for a key that is
// not there.
func globs(path, section, key string, value any) error {
	if value == nil {
		return nil
	}
	list, ok := value.([]any)
	if !ok {
		return fmt.Errorf("%s: [%s] %s must be an array of strings",
			path, section, key)
	}
	for _, element := range list {
		if _, ok := element.(string); !ok {
			return fmt.Errorf("%s: [%s] %s contains a non-string: %s",
				path, section, key, pyRepr(element))
		}
	}
	return nil
}

func isStringList(value any) bool {
	if value == nil {
		return true
	}
	list, ok := value.([]any)
	if !ok {
		return false
	}
	for _, element := range list {
		if _, ok := element.(string); !ok {
			return false
		}
	}
	return true
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func sorted(values []string) []string {
	copied := append([]string(nil), values...)
	sort.Strings(copied)
	return copied
}

// wikiLayout is `wiki_layout` (src/brain/manifest.py:59-93): where the
// bundle sits inside its repository, checked, or "" where the key is
// unsaid. Repo-relative and nothing else -- the registration holds one
// absolute path per area and a linked worktree has another one, so an
// absolute value here would block writing on the branch, which is the
// reason the wiki moved into the repo at all.
//
// `pkg/config`'s `(*Manifest).WikiLayout` is the same four tests over the
// same value and is deliberately not called: it reads a typed string
// field, so it cannot see a `wiki = 1` at all, and it quotes with `%q`
// where the barrier's reason has to quote with `repr`. A test holds the
// two together on the values both can be asked.
func wikiLayout(layout map[string]any) (string, error) {
	declared := layout["wiki"]
	if !truthy(declared) {
		return "", nil
	}
	value, ok := declared.(string)
	if !ok {
		// `"\\" in value` raises TypeError on a number, and `decide`'s
		// catch turns that into a refusal.
		return "", fmt.Errorf(
			"[layout] wiki must be a string, found %s", pyRepr(declared))
	}
	// The backslash comes first because a posix split hides both `..` and
	// the root behind one, and because neither flavour of path calls
	// `\srv\wiki` absolute -- without this test such a value would pass
	// all the others.
	if strings.Contains(value, `\`) {
		return "", fmt.Errorf(
			"[layout] wiki must use forward slashes, found %s",
			pyRepr(value))
	}
	if escapes(value) {
		return "", fmt.Errorf(
			"[layout] wiki must stay inside the repository, found %s",
			pyRepr(value))
	}
	if !namesAPart(value) {
		return "", fmt.Errorf(
			"[layout] wiki must not be the repository root, found %s",
			pyRepr(value))
	}
	return value, nil
}

// escapes says whether a value leaves the repository: absolute in either
// flavour of path, or carrying a `..` component. Neither flavour alone
// covers both spellings -- `PurePosixPath("C:/x").is_absolute()` is False
// and `PureWindowsPath("/srv/x").is_absolute()` is False -- so both are
// asked. With the backslash already refused, the windows half reduces to
// a drive followed by a separator, and pathlib does not ask that the
// drive be a letter.
func escapes(value string) bool {
	if strings.HasPrefix(value, "/") {
		return true
	}
	for offset, r := range value {
		// The first *rune*: a one-character drive spelt outside ASCII is
		// two bytes here and one character to pathlib.
		size := len(string(r))
		if offset == 0 && len(value) >= size+2 && value[size] == ':' &&
			value[size+1] == '/' {
			return true
		}
		break
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

// namesAPart is `PurePosixPath(value).parts` being non-empty. pathlib
// drops the empty component a doubled or trailing slash leaves and the
// `.` component that joins onto a path without moving it, and keeps
// everything else -- a blank is a name like any other, which is why
// nothing is trimmed here.
func namesAPart(value string) bool {
	for _, part := range strings.Split(value, "/") {
		if part != "" && part != "." {
			return true
		}
	}
	return false
}

// declarationIn is `_declaration_in`: the
// manifest this one directory carries, or "" if it carries none.
//
// The place is `registry.manifest_path`'s, and it is repeated rather than
// called: that function answers about a *registered* area, from its scope
// and the state directory, while this walk climbs a path no area is known
// for yet -- asking it would mean already knowing the answer.
func declarationIn(directory string) string {
	bundled := filepath.Join(directory, bundleDir, manifestName)
	if isRegularFile(bundled) {
		return bundled
	}
	return ""
}

// isRegularFile is `Path.is_file()`: follows links, and answers no rather
// than raising where the path cannot be looked at.
func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
