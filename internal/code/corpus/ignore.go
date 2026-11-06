package corpus

import (
	"crypto/sha256"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/gitenv"
)

// Source is one ignore file the build read, with what a freshness probe and
// the corpus digest need: its size, its modification time and the SHA-256 of
// its bytes.
//
// Rel is repository-relative for a file inside the repository. A global
// excludes file outside it keeps its absolute path, because there is no
// repository-relative name for it; the record says so and the digest covers
// the bytes, not a machine-independent path.
type Source struct {
	Rel   string
	Kind  string
	Size  int64
	MTime int64
	Hash  [32]byte
	// Body is the file's text. A probe rebuilds the matcher from the record
	// without reading an ignore file, so the patterns travel with it.
	Body []byte
}

// pattern is one parsed line of an ignore file.
type pattern struct {
	// base is the repository-relative directory of the ignore file; "" is the
	// repository root. A pattern only applies below its base.
	base     string
	negate   bool
	dirOnly  bool
	anchored bool
	segs     []string
	ci       bool
}

// Rules is the ignore evaluation of one repository: the patterns of every
// source, the tracked paths, and the settings that decide how a pattern
// matches. It implements Matcher.
type Rules struct {
	top        string
	prefix     string
	ignoreCase bool

	tracked     map[string]bool
	trackedDirs map[string]bool

	global  []pattern
	exclude []pattern
	dirs    map[string][]pattern
	loaded  map[string]bool

	sources    []Source
	sourceSeen map[string]bool

	indexPath   string
	indexSize   int64
	indexMTime  int64
	configPath  string
	configSize  int64
	configMTime int64
	excludePath string
	globalPath  string
}

// Load reads the ignore state of the repository that contains root.
//
// It returns a nil *Rules and a nil error when root is not inside a git
// repository (or git cannot answer): the caller then walks without ignore
// evaluation. Patterns are parsed here; a .gitignore is read when the first
// path below its directory is evaluated, the way git reads it.
//
// The returned sources are the files read so far; the .gitignore files under
// root join the list while the walk runs, so a caller that wants all of them
// records them after the walk.
func Load(root string) (*Rules, []Source, error) {
	top, err := gitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, nil, nil
	}
	r := &Rules{
		top:         filepath.Clean(strings.TrimSpace(top)),
		tracked:     map[string]bool{},
		trackedDirs: map[string]bool{},
		dirs:        map[string][]pattern{},
		loaded:      map[string]bool{},
		sourceSeen:  map[string]bool{},
	}
	if prefix, err := gitOutput(root, "rev-parse", "--show-prefix"); err == nil {
		r.prefix = strings.TrimSuffix(filepath.ToSlash(strings.TrimSpace(prefix)), "/")
	}
	if v, err := gitOutput(root, "config", "--bool", "--get", "core.ignorecase"); err == nil {
		r.ignoreCase = strings.TrimSpace(v) == "true"
	}

	r.indexPath = r.gitPath(root, "index")
	if info, err := os.Stat(r.indexPath); err == nil {
		r.indexSize, r.indexMTime = info.Size(), info.ModTime().UnixNano()
	}
	r.configPath = r.gitPath(root, "config")
	if info, err := os.Stat(r.configPath); err == nil {
		r.configSize, r.configMTime = info.Size(), info.ModTime().UnixNano()
	}
	r.excludePath = r.gitPath(root, "info/exclude")
	if data, ok := r.loadSource(r.excludePath, r.relOf(r.excludePath), "exclude"); ok {
		r.exclude = r.parse(data, "")
	}
	if r.globalPath = r.globalExcludes(root); r.globalPath != "" {
		if data, ok := r.loadSource(r.globalPath, r.relOf(r.globalPath), "global"); ok {
			r.global = r.parse(data, "")
		}
	}

	tracked, err := gitOutput(root, "ls-files", "-z", "--cached")
	if err != nil {
		return nil, nil, err
	}
	for _, rel := range strings.Split(tracked, "\x00") {
		if rel == "" {
			continue
		}
		r.tracked[rel] = true
		for dir := path.Dir(rel); dir != "." && dir != "/"; dir = path.Dir(dir) {
			r.trackedDirs[dir] = true
		}
	}
	return r, r.Sources(), nil
}

// Sources returns the ignore files read so far, sorted by path.
func (r *Rules) Sources() []Source {
	out := append([]Source(nil), r.sources...)
	sort.Slice(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
	return out
}

// RepoTop is the repository root Load found.
func (r *Rules) RepoTop() string { return r.top }

// Prefix is root's path relative to the repository root, "" at the top.
func (r *Rules) Prefix() string { return r.prefix }

// IgnoreCase reports core.ignorecase.
func (r *Rules) IgnoreCase() bool { return r.ignoreCase }

// TrackedPaths returns the tracked paths, sorted.
func (r *Rules) TrackedPaths() []string {
	out := make([]string, 0, len(r.tracked))
	for rel := range r.tracked {
		out = append(out, rel)
	}
	sort.Strings(out)
	return out
}

// IndexStamp is the path, size and modification time of .git/index, the stamp
// a freshness probe compares to notice `git add -f` and `git rm --cached`.
func (r *Rules) IndexStamp() (string, int64, int64) {
	return r.indexPath, r.indexSize, r.indexMTime
}

// ConfigStamp is the path, size and modification time of .git/config, the
// stamp that notices a changed core.ignorecase or core.excludesFile.
func (r *Rules) ConfigStamp() (string, int64, int64) {
	return r.configPath, r.configSize, r.configMTime
}

// GitPaths returns the resolved index and exclude paths and the global
// excludes file, "" when there is none.
func (r *Rules) GitPaths() (index, exclude, global string) {
	return r.indexPath, r.excludePath, r.globalPath
}

// Ignored reports whether rel is excluded, the way git decides it: the last
// matching pattern wins, a tracked file is never ignored, and a path below an
// ignored directory stays ignored even when a later pattern names it.
func (r *Rules) Ignored(rel string, isDir bool) bool {
	full := r.full(rel)
	if full == "" || full == "." {
		return false
	}
	if !isDir && r.tracked[full] {
		return false
	}
	parts := strings.Split(full, "/")
	for i := 1; i < len(parts); i++ {
		if r.eval(strings.Join(parts[:i], "/"), true) {
			return true
		}
	}
	return r.eval(full, isDir)
}

// Tracked reports whether the file is in the repository index.
func (r *Rules) Tracked(rel string) bool {
	return r.tracked[r.full(rel)]
}

// HasTrackedUnder reports whether a tracked file lives below dir.
func (r *Rules) HasTrackedUnder(rel string) bool {
	full := r.full(rel)
	if full == "" || full == "." {
		return len(r.tracked) > 0
	}
	return r.trackedDirs[full]
}

// eval runs the patterns of every source against one repository-relative
// path, lowest precedence first: the global excludes file, .git/info/exclude,
// then the .gitignore chain from the repository root down to the path's
// directory. A deeper .gitignore therefore wins over a shallower one, and a
// negation only undoes a match of equal or lower precedence.
func (r *Rules) eval(full string, isDir bool) bool {
	dir := path.Dir(full)
	if dir == "." {
		dir = ""
	}
	chain := dirChain(dir)
	for _, d := range chain {
		r.loadDir(d)
	}
	ignored := false
	apply := func(patterns []pattern) {
		for _, p := range patterns {
			if p.matches(full, isDir) {
				ignored = !p.negate
			}
		}
	}
	apply(r.global)
	apply(r.exclude)
	for _, d := range chain {
		apply(r.dirs[d])
	}
	return ignored
}

// LoadIgnore reads the .gitignore beside rel, so a walk records every visited
// ignore source even where a tracked file short-circuits the ignore query.
// rel is repository-root relative.
func (r *Rules) LoadIgnore(rel string) {
	dir := path.Dir(r.full(rel))
	if dir == "." {
		dir = ""
	}
	r.loadDir(dir)
}

// loadDir reads the .gitignore of one directory, once.
func (r *Rules) loadDir(relDir string) {
	if r.loaded[relDir] {
		return
	}
	r.loaded[relDir] = true
	abs := filepath.Join(r.top, filepath.FromSlash(relDir), ".gitignore")
	rel := path.Join(relDir, ".gitignore")
	if relDir == "" {
		rel = ".gitignore"
	}
	data, ok := r.loadSource(abs, rel, "gitignore")
	if !ok {
		return
	}
	r.dirs[relDir] = r.parse(data, relDir)
}

// loadSource records one ignore file's bytes and metadata, once.
//
//coverage:exempt the os.Stat arm needs the file to vanish between read and stat
func (r *Rules) loadSource(abs, rel, kind string) ([]byte, bool) {
	if abs == "" || r.sourceSeen[abs] {
		return nil, false
	}
	r.sourceSeen[abs] = true
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, false
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, false
	}
	r.sources = append(r.sources, Source{
		Rel:   rel,
		Kind:  kind,
		Size:  info.Size(),
		MTime: info.ModTime().UnixNano(),
		Hash:  sha256.Sum256(data),
		Body:  data,
	})
	return data, true
}

// full turns a walk-relative path into a repository-relative one.
func (r *Rules) full(rel string) string {
	rel = strings.TrimPrefix(strings.TrimPrefix(rel, "./"), "/")
	if rel == "." {
		rel = ""
	}
	if r.prefix == "" {
		return rel
	}
	if rel == "" {
		return r.prefix
	}
	return r.prefix + "/" + rel
}

// relOf is the repository-relative name of an absolute path, or the path
// itself when it lies outside the repository.
func (r *Rules) relOf(abs string) string {
	if abs == "" {
		return ""
	}
	rel, err := filepath.Rel(r.top, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(rel)
}

// gitPath resolves one path git keeps, absolute.
//
//coverage:exempt the empty-output arm never fires for rev-parse --git-path, and the absolute arm needs git to answer with an absolute path, which it does not for a relative invocation
func (r *Rules) gitPath(root, name string) string {
	out, err := gitOutput(root, "rev-parse", "--git-path", name)
	if err != nil {
		return ""
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return ""
	}
	if filepath.IsAbs(out) {
		return filepath.Clean(out)
	}
	return filepath.Join(root, filepath.FromSlash(out))
}

// globalExcludes is the configured core.excludesFile, or git's default when
// the key is unset: $XDG_CONFIG_HOME/git/ignore, else ~/.config/git/ignore.
// Measured against git check-ignore, see the package's tests.
func (r *Rules) globalExcludes(root string) string {
	if out, err := gitOutput(root, "config", "--path", "--get", "core.excludesFile"); err == nil {
		if v := strings.TrimSpace(out); v != "" {
			return v
		}
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" && filepath.IsAbs(xdg) {
		return filepath.Join(xdg, "git", "ignore")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "git", "ignore")
}

// gitOutput runs one git command and returns its standard output.
func gitOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = gitenv.Environ()
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// dirChain is "" and every ancestor directory of dir, root first.
func dirChain(dir string) []string {
	if dir == "" {
		return []string{""}
	}
	parts := strings.Split(dir, "/")
	chain := make([]string, 0, len(parts)+1)
	chain = append(chain, "")
	for i := range parts {
		chain = append(chain, strings.Join(parts[:i+1], "/"))
	}
	return chain
}

// parse parses one ignore file's lines into patterns and stamps the case
// sensitivity the repository's core.ignorecase asks for.
func (r *Rules) parse(body []byte, base string) []pattern {
	patterns := parseIgnore(body, base)
	for i := range patterns {
		patterns[i].ci = r.ignoreCase
	}
	return patterns
}

// parseIgnore parses one ignore file's lines into patterns.
func parseIgnore(body []byte, base string) []pattern {
	var out []pattern
	for _, raw := range strings.Split(string(body), "\n") {
		line := strings.TrimSuffix(raw, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = trimTrailingSpaces(line)
		if line == "" {
			continue
		}
		negate := strings.HasPrefix(line, "!")
		if negate {
			line = line[1:]
		}
		dirOnly := strings.HasSuffix(line, "/")
		line = strings.TrimSuffix(line, "/")
		if line == "" {
			continue
		}
		anchored := strings.Contains(line, "/")
		line = strings.TrimPrefix(line, "/")
		if line == "" {
			continue
		}
		out = append(out, pattern{
			base:     base,
			negate:   negate,
			dirOnly:  dirOnly,
			anchored: anchored,
			segs:     strings.Split(line, "/"),
		})
	}
	return out
}

// trimTrailingSpaces drops trailing spaces git ignores, unless a backslash
// escaped them.
func trimTrailingSpaces(line string) string {
	for len(line) > 0 && line[len(line)-1] == ' ' {
		backslashes := 0
		for i := len(line) - 2; i >= 0 && line[i] == '\\'; i-- {
			backslashes++
		}
		if backslashes%2 == 1 {
			return line
		}
		line = line[:len(line)-1]
	}
	return line
}

// matches reports whether the pattern names the path.
func (p pattern) matches(full string, isDir bool) bool {
	if p.dirOnly && !isDir {
		return false
	}
	rel := full
	if p.base != "" {
		prefix := p.base + "/"
		if !strings.HasPrefix(full, prefix) {
			return false
		}
		rel = full[len(prefix):]
	}
	if rel == "" {
		return false
	}
	if !p.anchored {
		return globSegment(p.segs[0], path.Base(rel), p.ci)
	}
	return matchSegments(p.segs, strings.Split(rel, "/"), p.ci)
}

// matchSegments matches the pattern's segments against the path's segments,
// with "**" standing for any number of whole segments.
func matchSegments(pattern, segments []string, ignoreCase bool) bool {
	if len(pattern) == 0 {
		return len(segments) == 0
	}
	if pattern[0] == "**" {
		// "**" stands for any number of segments, including none: git reports
		// the directory of a trailing abc/** as ignored too (measured against
		// git check-ignore).
		for i := 0; i <= len(segments); i++ {
			if matchSegments(pattern[1:], segments[i:], ignoreCase) {
				return true
			}
		}
		return false
	}
	if len(segments) == 0 || !globSegment(pattern[0], segments[0], ignoreCase) {
		return false
	}
	return matchSegments(pattern[1:], segments[1:], ignoreCase)
}

// globSegment matches one glob segment against one name, following fnmatch
// with FNM_PATHNAME: "*" and "?" do not cross the segment boundary.
func globSegment(pattern, name string, ignoreCase bool) bool {
	if ignoreCase {
		pattern, name = strings.ToLower(pattern), strings.ToLower(name)
	}
	return globMatch(pattern, name)
}

// globMatch is globSegment's matcher.
func globMatch(pattern, name string) bool {
	i := 0
	for i < len(pattern) {
		switch pattern[i] {
		case '*':
			for i < len(pattern) && pattern[i] == '*' {
				i++
			}
			if i == len(pattern) {
				return true
			}
			for k := 0; k <= len(name); k++ {
				if globMatch(pattern[i:], name[k:]) {
					return true
				}
			}
			return false
		case '?':
			if len(name) == 0 {
				return false
			}
			i++
			name = name[1:]
		case '[':
			ok, consumed, matched := classMatch(pattern[i:], name)
			if !ok {
				return false
			}
			i += consumed
			name = name[matched:]
		case '\\':
			if i+1 >= len(pattern) {
				if len(name) == 0 || name[0] != '\\' {
					return false
				}
				i++
				name = name[1:]
				continue
			}
			if len(name) == 0 || name[0] != pattern[i+1] {
				return false
			}
			i += 2
			name = name[1:]
		default:
			if len(name) == 0 || name[0] != pattern[i] {
				return false
			}
			i++
			name = name[1:]
		}
	}
	return len(name) == 0
}

// classMatch matches one character against a bracket expression at the start
// of pattern. It returns whether the class matched the character, how many
// bytes of the pattern it consumed and how many bytes of the name.
func classMatch(pattern, name string) (matched bool, consumed, used int) {
	if len(name) == 0 {
		return false, 0, 0
	}
	i := 1
	negate := false
	if i < len(pattern) && (pattern[i] == '!' || pattern[i] == '^') {
		negate = true
		i++
	}
	hit := false
	first := true
	for i < len(pattern) {
		if pattern[i] == ']' && !first {
			break
		}
		first = false
		lo := pattern[i]
		if lo == '\\' && i+1 < len(pattern) {
			i++
			lo = pattern[i]
		}
		i++
		hi := lo
		if i+1 < len(pattern) && pattern[i] == '-' && pattern[i+1] != ']' {
			i++
			hi = pattern[i]
			if hi == '\\' && i+1 < len(pattern) {
				i++
				hi = pattern[i]
			}
			i++
		}
		if name[0] >= lo && name[0] <= hi {
			hit = true
		}
	}
	if i >= len(pattern) {
		// An unterminated class is a literal "[".
		return name[0] == '[', 1, 1
	}
	i++
	return hit != negate, i, 1
}
