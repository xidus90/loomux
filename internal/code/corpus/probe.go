package corpus

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Settings are the inputs of a probe that come from configuration rather than
// from the tree.
type Settings struct {
	// MaxFilesize is the resolved size limit; 0 means unlimited.
	MaxFilesize int64
}

// Drift is what moved since the build. Clean means the record still stands.
type Drift struct {
	Changed  []string
	Added    []string
	Removed  []string
	Sources  []string
	Settings bool
	Index    bool
	Config   bool
}

// Clean reports whether the record still describes the tree.
func (d Drift) Clean() bool { return d.Count() == 0 }

// Count is how many things moved, in any category.
func (d Drift) Count() int {
	n := len(d.Changed) + len(d.Added) + len(d.Removed) + len(d.Sources)
	for _, moved := range []bool{d.Settings, d.Index, d.Config} {
		if moved {
			n++
		}
	}
	return n
}

// readFile is the read the probe path goes through, so a test can count that
// the clean path reads nothing but the record.
var readFile = os.ReadFile

// Probe compares the working tree against the record of the last build.
//
// A nil Drift with a nil error means UNKNOWN: never built, or a record this
// code cannot read. The caller treats that as "rebuild", never as clean.
//
// The probe never calls git. It rebuilds the ignore matcher from the record's
// sources and tracked paths and compares .git/index and .git/config by stat;
// when either stamp moved it reports drift, and the rebuild that follows is
// the one that runs git.
//
//coverage:exempt the Walk error arm needs the root to vanish between reading the record and walking it; the record lives inside root, so a missing root also means a missing record and the probe returns unknown before the walk
func Probe(root string, s Settings) (*Drift, error) {
	rec, ok := Read(root)
	if !ok {
		return nil, nil
	}
	rules := rulesFromRecord(rec)
	listing, err := Walk(root, matcherForRules(rules))
	if err != nil {
		return nil, err
	}
	return Compare(rec, listing, rules, s), nil
}

// matcherForRules keeps a nil *Rules out of the Matcher interface.
func matcherForRules(rules *Rules) Matcher {
	if rules == nil {
		return nil
	}
	return rules
}

// RulesFromRecord rebuilds the matcher the record was written with, without
// reading an ignore file. A caller that walks with it can compare the listing
// through Compare and skip the git calls Load would make.
func RulesFromRecord(rec Record) *Rules { return rulesFromRecord(rec) }

// Compare is the probe over a listing the caller already walked. The rules
// must be the ones the listing was walked with; a caller that has the record
// builds them with RulesFromRecord and keeps the walk.
func Compare(rec Record, listing Listing, rules *Rules, s Settings) *Drift {
	d := &Drift{}
	if !bytes.Equal(rec.Settings, CanonicalSettings(s.MaxFilesize, rec.IgnoreCase)) {
		d.Settings = true
	}
	if abs := gitAbs(rec, rec.GitPaths.Index); abs != "" && !stampMatches(abs, rec.Stamp) {
		d.Index = true
	}
	if abs := gitAbs(rec, rec.GitPaths.Config); abs != "" && !stampMatches(abs, rec.ConfigStamp) {
		d.Config = true
	}

	// The file set is compared at the limit the record was built with: a
	// caller asking for another limit changes the answer, not the tree, and
	// the settings flag already reports that.
	limit := s.MaxFilesize
	if rec.MaxFilesize != s.MaxFilesize {
		limit = rec.MaxFilesize
	}
	files, _, _ := SearchCorpus(listing, limit)
	keep := make(map[string]bool, len(files))
	for _, f := range files {
		keep[f.Rel] = true
		p, known := rec.Files[f.Rel]
		if !known {
			d.Added = append(d.Added, f.Rel)
			continue
		}
		if trustedPrint(p, f) {
			continue
		}
		sum, err := hashFile(f.Abs)
		if err != nil {
			// Unreadable right now: leave it to the next probe rather than
			// reporting drift a rebuild could not repair either.
			continue
		}
		if sum != p.Hash {
			d.Changed = append(d.Changed, f.Rel)
		}
	}
	for rel := range rec.Files {
		if !keep[rel] {
			d.Removed = append(d.Removed, rel)
		}
	}
	d.Sources = append(d.Sources, changedSources(rec)...)
	d.Sources = append(d.Sources, addedIgnoreFiles(rec, rules, listing)...)
	d.Sources = dedupeSorted(d.Sources)
	sort.Strings(d.Changed)
	sort.Strings(d.Added)
	sort.Strings(d.Removed)
	return d
}

// rulesFromRecord rebuilds the matcher the record was written with, without
// reading an ignore file.
func rulesFromRecord(rec Record) *Rules {
	if rec.GitPaths.Root == "" {
		return nil
	}
	r := &Rules{
		top:         rec.GitPaths.Root,
		prefix:      rec.GitPaths.Prefix,
		ignoreCase:  rec.IgnoreCase,
		tracked:     make(map[string]bool, len(rec.Tracked)),
		trackedDirs: map[string]bool{},
		dirs:        map[string][]pattern{},
		loaded:      map[string]bool{},
		sourceSeen:  map[string]bool{},
	}
	for _, rel := range rec.Tracked {
		r.tracked[rel] = true
		for dir := path.Dir(rel); dir != "." && dir != "/"; dir = path.Dir(dir) {
			r.trackedDirs[dir] = true
		}
	}
	for _, s := range rec.Sources {
		dir := ""
		if s.Kind == "gitignore" {
			dir = strings.TrimSuffix(path.Dir(s.Rel), "/")
			if dir == "." {
				dir = ""
			}
			r.loaded[dir] = true
		}
		switch s.Kind {
		case "exclude":
			r.exclude = r.parse(s.Body, "")
		case "global":
			r.global = r.parse(s.Body, "")
		default:
			r.dirs[dir] = r.parse(s.Body, dir)
		}
	}
	return r
}

// changedSources reports the ignore sources whose stamp or bytes moved since
// the build, and the sources that disappeared.
func changedSources(rec Record) []string {
	var out []string
	for _, src := range rec.Sources {
		abs := sourceAbs(rec, src.Rel)
		info, err := os.Stat(abs)
		if err != nil {
			out = append(out, src.Rel)
			continue
		}
		if info.Size() == src.Size && info.ModTime().UnixNano() == src.MTime {
			continue
		}
		data, err := readFile(abs)
		if err != nil {
			out = append(out, src.Rel)
			continue
		}
		if sha256.Sum256(data) != src.Hash {
			out = append(out, src.Rel)
		}
	}
	known := make(map[string]bool, len(rec.Sources))
	for _, src := range rec.Sources {
		known[sourceAbs(rec, src.Rel)] = true
	}
	for _, p := range []string{rec.GitPaths.Exclude, rec.GitPaths.Global} {
		abs := gitAbs(rec, p)
		if abs == "" || known[abs] {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			out = append(out, relForAbs(rec, abs))
		}
	}
	return out
}

// addedIgnoreFiles reports .gitignore files the walk saw that the record does
// not know: creating one changes which files the corpus holds, and it is not
// itself in the corpus because its name starts with a dot.
func addedIgnoreFiles(rec Record, rules *Rules, listing Listing) []string {
	known := make(map[string]bool, len(rec.Sources))
	for _, src := range rec.Sources {
		if src.Kind == "gitignore" {
			known[src.Rel] = true
		}
	}
	var out []string
	for _, e := range listing.Entries {
		if path.Base(e.Rel) != ".gitignore" || e.Link {
			continue
		}
		rel := e.Rel
		if rules != nil {
			rel = rules.full(e.Rel)
		}
		if !known[rel] {
			out = append(out, rel)
		}
	}
	return out
}

// gitAbs resolves a git path of the record. The record may store the index and
// exclude paths repository-relative with the platform's separator; resolving
// them against the stored root makes the comparison independent of the
// caller's working directory and of slash style.
func gitAbs(rec Record, p string) string {
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(rec.GitPaths.Root, p))
}

// sourceAbs is where an ignore source lives now: an absolute Rel stands for
// itself, everything else sits under the repository root the record stored.
func sourceAbs(rec Record, rel string) string {
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel)
	}
	return filepath.Join(rec.GitPaths.Root, filepath.FromSlash(rel))
}

// relForAbs is the repository-relative form of a path outside the sources,
// used for the source name a probe reports.
func relForAbs(rec Record, abs string) string {
	if rec.GitPaths.Root == "" {
		return filepath.ToSlash(abs)
	}
	rel, err := filepath.Rel(rec.GitPaths.Root, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(rel)
}

// stampMatches reports whether the file at abs still has the recorded stamp.
// A zero stamp for a missing file is a match: a repository without an index
// (nothing added yet) records zero, and creating the index later changes the
// answer.
func stampMatches(abs string, want Stamp) bool {
	info, err := os.Stat(abs)
	if err != nil {
		return want == Stamp{} && os.IsNotExist(err)
	}
	return info.Size() == want.Size && info.ModTime().UnixNano() == want.MTime
}

// trustedPrint reports whether a recorded print may stand for the file on
// disk without reading it. An empty hash means the last build never got the
// bytes, so it is never trusted.
func trustedPrint(p Print, f SearchFile) bool {
	return p.Hash != "" && p.Size == f.Size && p.MTime == f.MTime
}

// hashFile is sha256 over a file's bytes, as full hex.
func hashFile(abs string) (string, error) {
	b, err := readFile(abs)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// dedupeSorted sorts names and drops duplicates, keeping the first of each.
func dedupeSorted(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	sort.Strings(names)
	out := names[:1]
	for _, name := range names[1:] {
		if name != out[len(out)-1] {
			out = append(out, name)
		}
	}
	return out
}
