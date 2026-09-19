// Package detect answers what a project is, and nothing else.
//
// It takes an fs.FS rather than a path so the tests need no directories on
// disk, and it writes nothing: every decision that follows from these facts
// is made elsewhere. `--dry-run` is free because of that.
package detect

import (
	"io/fs"
	"path"
	"sort"
	"strings"
)

// okfMarker is what separates a brain bundle from a docs folder that happens
// to be called wiki. It stands in the front matter of the bundle's index.
const okfMarker = "okf_version"

// Facts is what a tree says about itself.
type Facts struct {
	Stacks    []string
	HasGit    bool
	HooksPath string
	WikiMode  string
	WikiPath  string
	// GodotDir is the area the Godot tree stands in, "" for the root.
	// gdlint reads .gdlintrc from the working directory upwards, so a
	// check started anywhere above this directory silently runs on
	// defaults -- which is a different set of rules than the project's.
	GodotDir string
	// Areas names, per stack, every area a signal of that stack matched in:
	// "." for the root, else the top-level directory. A check runs one lane
	// per area, so two web apps under one root are linted where each lives.
	Areas map[string][]string
	// Ambiguous carries findings that must not be decided here -- a Django
	// project whose migrations may be generated or wanted, a package.json
	// that may be tooling only. The interview resolves them.
	Ambiguous []string
}

// Detect reads the root and one level below it, for workspaces.
//
// Two facts are missing from what it can return: HooksPath lives in git's
// config and the neighbour wiki lies outside the root. Both have their own
// function in this package, and cmd/init joins them to these facts.
func Detect(root fs.FS) Facts {
	areas := searchAreas(root)
	found := map[string]bool{}
	areasOf := map[string]map[string]bool{}
	for _, sig := range signals {
		for _, area := range areas {
			if !matches(root, area, sig) {
				continue
			}
			for _, stack := range sig.stacks {
				found[stack] = true
				if areasOf[stack] == nil {
					areasOf[stack] = map[string]bool{}
				}
				areasOf[stack][area] = true
			}
		}
	}
	facts := Facts{Stacks: sorted(found), Areas: map[string][]string{}, Ambiguous: doubts(root, areas), GodotDir: godotArea(root, areas)}
	for stack, set := range areasOf {
		facts.Areas[stack] = sorted(set)
	}
	if _, err := fs.Stat(root, ".git"); err == nil {
		facts.HasGit = true
	}
	facts.readWiki(root)
	if facts.WikiMode != "" && facts.WikiMode != "none" {
		found["wiki"] = true
		facts.Stacks = sorted(found)
	}
	return facts
}

// readWiki decides between the two things a wiki/ directory can be.
//
// The marker rather than the mere directory: a repository that keeps plain
// documentation under wiki/ would otherwise be told it has a brain bundle,
// and everything downstream would act on it.
func (facts *Facts) readWiki(root fs.FS) {
	if manifest, ok := firstManifest(root); ok && declaresWiki(manifest) {
		facts.WikiMode = "brain"
		facts.WikiPath = "wiki/"
		for _, cand := range []string{"docs/wiki", "wiki"} {
			if _, err := fs.Stat(root, cand); err == nil {
				facts.WikiPath = cand + "/"
				break
			}
		}
		return
	}

	for _, cand := range []string{"wiki", "docs/wiki"} {
		entries, err := fs.ReadDir(root, cand)
		if err != nil || len(entries) == 0 {
			continue
		}
		index, err := fs.ReadFile(root, path.Join(cand, "index.md"))
		if err == nil && strings.Contains(string(index), okfMarker) {
			facts.WikiMode, facts.WikiPath = "brain", cand+"/"
			return
		}
		bundle, err := fs.ReadFile(root, path.Join(cand, "bundle.toml"))
		if err == nil && strings.Contains(string(bundle), okfMarker) {
			facts.WikiMode, facts.WikiPath = "brain", cand+"/"
			return
		}
		facts.Ambiguous = append(facts.Ambiguous, wikiNote)
		return
	}
}

// manifestNames holds one name, because loomux has one manifest. It stays a
// list because the search below is written as one: the first file that can be
// read decides.
//
// It is a hand copy of internal/config's list of the same name, which
// `config.ManifestPath` reads as `manifestNames[0]`. Detection takes an fs.FS
// and config takes a root path, so neither can be the other's caller here; a
// second manifest name would have to be added in both places.
var manifestNames = []string{".loomux/config.toml"}

// firstManifest returns the text of the first manifest loomux would read.
func firstManifest(root fs.FS) (string, bool) {
	for _, name := range manifestNames {
		if data, err := fs.ReadFile(root, name); err == nil {
			return string(data), true
		}
	}
	return "", false
}

// declaresWiki reads the one thing detection needs from a manifest: whether
// it says this area has a wiki, as `wiki = true` or as a `[wiki]` table.
// A `wiki = "docs/wiki"` under [layout] names a place and says nothing about
// whether there is a wiki, so only the literal true counts.
func declaresWiki(manifest string) bool {
	for _, line := range strings.Split(manifest, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[wiki]") {
			return true
		}
		if strings.HasPrefix(trimmed, "wiki") && strings.Contains(trimmed, "=") {
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "wiki"))
			value = strings.TrimSpace(strings.TrimPrefix(value, "="))
			if value == "true" {
				return true
			}
		}
	}
	return false
}

// doubts collects the questions the tree raises without answering.
func doubts(root fs.FS, areas []string) []string {
	var notes []string
	for _, doubt := range ambiguities {
		for _, area := range areas {
			if !exists(root, path.Join(area, doubt.path)) {
				continue
			}
			if doubt.unless != "" && exists(root, path.Join(area, doubt.unless)) {
				continue
			}
			notes = append(notes, doubt.note)
			break
		}
	}
	return notes
}

// searchAreas names the root and the directories a workspace member could be.
//
// One level and no further: below that, a directory belongs to a member's own
// layout, and a rule fired from there says something about the wrong project.
// Dot directories are skipped for the same reason -- `.godot/` and `.venv/`
// carry caches, not projects, and a false alarm costs more than a missing rule.
func searchAreas(root fs.FS) []string {
	areas := []string{"."}
	entries, err := fs.ReadDir(root, ".")
	if err != nil {
		return areas
	}
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			areas = append(areas, entry.Name())
		}
	}
	return areas
}

// godotArea names the directory the Godot tree stands in.
//
// project.godot before .gdlintrc: the engine file says a project is here,
// while a lone .gdlintrc could as well configure a linter for scripts that
// belong to a project elsewhere. Both are looked for because a tree may carry
// only the second one, and the answer this serves is where gdlint has to
// start.
func godotArea(root fs.FS, areas []string) string {
	for _, marker := range []string{"project.godot", ".gdlintrc"} {
		for _, area := range areas {
			if exists(root, path.Join(area, marker)) {
				if area == "." {
					return ""
				}
				return area
			}
		}
	}
	return ""
}

func matches(root fs.FS, area string, sig signal) bool {
	if sig.besides != "" && !exists(root, path.Join(area, sig.besides)) {
		return false
	}
	for _, name := range candidates(root, area, sig) {
		if sig.contains == "" {
			return true
		}
		body, err := fs.ReadFile(root, name)
		if err == nil && strings.Contains(string(body), sig.contains) {
			return true
		}
	}
	return false
}

func candidates(root fs.FS, area string, sig signal) []string {
	if sig.path != "" {
		full := path.Join(area, sig.path)
		if exists(root, full) {
			return []string{full}
		}
		return nil
	}
	entries, err := fs.ReadDir(root, area)
	if err != nil {
		return nil
	}
	var hits []string
	for _, entry := range entries {
		if ok, _ := path.Match(sig.glob, entry.Name()); ok {
			hits = append(hits, path.Join(area, entry.Name()))
		}
	}
	return hits
}

func exists(root fs.FS, name string) bool {
	_, err := fs.Stat(root, name)
	return err == nil
}

func sorted(set map[string]bool) []string {
	var all []string
	for key := range set {
		all = append(all, key)
	}
	sort.Strings(all)
	return all
}
