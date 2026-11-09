package compat

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// Release is one version a source publishes, without a leading "v" or other
// decoration; Pre marks an alpha, beta, release candidate or dev build.
type Release struct {
	Version string
	Pre     bool
}

// Upstream answers the releases a source publishes, and for a runtime which
// of its major.minor versions are past end of support.
type Upstream interface {
	Releases(source string) ([]Release, error)
	EOL(runtime string) (map[string]bool, error)
}

// Finding is what the scout reports for a line, of one Kind:
//   - "new": a version newer than everything the line holds. Line is
//     <fixture>/<name>, Version the new major.minor (the major for a pin of a
//     major), Probe the entry that would try it and TOML that entry as an
//     [[entry]] block, ready to paste into compat.toml.
//   - "pre": a pre-release of the runtime's next version. Version is the
//     pre-release itself, the probe carries it as Pre, and TOML stays empty:
//     an entry proven only on a pre-release never goes into the list.
//   - "eol": a listed runtime version past end of support. Line is the
//     runtime, there is no probe and no TOML: the entries are to go.
type Finding struct {
	Title   string `json:"title"`
	Kind    string `json:"kind"`
	Line    string `json:"line"`
	Version string `json:"version"`
	Probe   *Entry `json:"probe,omitempty"`
	TOML    string `json:"toml"`
}

// IssueTitle is the title the finding's issue carries.
func IssueTitle(f Finding) string {
	if f.Kind == "eol" {
		return "compat: " + f.Line + " " + f.Version + " is past end of support"
	}
	return "compat: " + f.Line + " " + f.Version + " is out"
}

// preRe reads a pre-release the way the runtimes spell it: Go "1.28rc1" and
// "1.28beta1", Python "3.16.0a1", Godot "4.8-beta2". Compiled on first use:
// package start has an allocation budget (cmd/loomux start_test.go).
var preRe = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^(\d+)\.(\d+)(?:\.\d+)?-?(dev|a|alpha|b|beta|rc)(\d+)$`)
})

// parsePre reads the major.minor of a pre-release; anything else does not
// parse.
func parsePre(s string) ([]int, bool) {
	m := preRe().FindStringSubmatch(s)
	if m == nil {
		return nil, false
	}
	major, _ := strconv.Atoi(m[1]) // digits only, see parseVer
	minor, _ := strconv.Atoi(m[2])
	return []int{major, minor}, true
}

// preOrder ranks pre-releases of one major.minor: dev before alpha before
// beta before rc, then by number. s must parse with parsePre.
func preOrder(s string) []int {
	m := preRe().FindStringSubmatch(s)
	stage := map[string]int{"dev": 0, "a": 1, "alpha": 1, "b": 2, "beta": 2, "rc": 3}[m[3]]
	n, _ := strconv.Atoi(m[4])
	return []int{stage, n}
}

const (
	// godot-builds holds the pre-releases too; godotengine/godot only the
	// stable ones.
	godotSource  = "https://api.github.com/repos/godotengine/godot-builds/releases?per_page=100"
	pythonSource = "https://www.python.org/api/v2/downloads/release/?is_published=true"
	// Without include=all go.dev lists only the maintained stable releases.
	goSource       = "https://go.dev/dl/?mode=json&include=all"
	githubReleases = "https://api.github.com/repos/%s/releases?per_page=100"
	// eolSource covers go, python and godot under those names; python.org and
	// Godot publish no machine-readable end of support of their own.
	eolSource  = "https://endoflife.date/api/v1/products/%s"
	pypiSource = "https://pypi.org/pypi/%s/json"
)

var runtimeSources = map[string]string{"godot": godotSource, "python": pythonSource, "go": goSource}

var addonRepos = map[string]string{
	"GUT":          "bitwes/Gut",
	"gdUnit4":      "godot-gdunit-labs/gdUnit4",
	"NanoCoverage": "IgorBayerl/nano-coverage-godot",
}

// sourceOf is where the releases of a runtime, addon or tool come from.
func sourceOf(kind, name string) (string, error) {
	switch kind {
	case "runtime":
		if s, ok := runtimeSources[name]; ok {
			return s, nil
		}
	case "addon":
		if r, ok := addonRepos[name]; ok {
			return fmt.Sprintf(githubReleases, r), nil
		}
	default:
		return fmt.Sprintf(pypiSource, name), nil
	}
	return "", fmt.Errorf("no source known for %s %s", kind, name)
}

var plainVersionRe = regexp.MustCompile(`^\d+(\.\d+)*$`)

// parseVer reads a plain numeric version; a pre-release or anything else
// does not parse.
func parseVer(s string) ([]int, bool) {
	s = strings.TrimPrefix(s, "v")
	if !plainVersionRe.MatchString(s) {
		return nil, false
	}
	var v []int
	for _, p := range strings.Split(s, ".") {
		n, _ := strconv.Atoi(p) // digits only; an absurdly long one reads as 0
		v = append(v, n)
	}
	return v, true
}

func part(v []int, i int) int {
	if i < len(v) {
		return v[i]
	}
	return 0
}

// cmpVer compares numerically per component; a missing one is 0.
func cmpVer(a, b []int) int {
	for i := range max(len(a), len(b)) {
		if c := cmp.Compare(part(a, i), part(b, i)); c != 0 {
			return c
		}
	}
	return 0
}

// subject is one thing an entry pins: its runtime, an addon or a tool.
type subject struct{ kind, name, version string }

func subjectsOf(e Entry) []subject {
	out := []subject{{"runtime", e.Runtime, e.Version}}
	for _, a := range e.Addons {
		out = append(out, subject{"addon", a.Name, a.Version})
	}
	for _, name := range slices.Sorted(maps.Keys(e.Tools)) {
		// A pin is a glob: "0.17.*" holds 0.17, "3.*" holds 3.
		out = append(out, subject{"tool", name, strings.TrimSuffix(strings.TrimSuffix(e.Tools[name], "*"), ".")})
	}
	return out
}

// group is a line at one major: the entries of a fixture pinning one subject.
type group struct {
	fixture string
	subject
	major  int
	parts  int   // version parts that count: 1 for a pinned major, else 2
	max    []int // the highest version of the line
	newest Entry // the entry with the highest runtime version
}

// Scout answers what up knows that the list does not: per line, the newest
// minor within its major, or a newer major when the line holds the highest
// major of its subject. Patch releases and pre-releases never count.
func Scout(entries []Entry, up Upstream) ([]Finding, error) {
	var groups []*group
	top := map[string]int{} // kind/name -> highest major in the list
	for _, e := range entries {
		ev, _ := parseVer(e.Version) // the list is validated MAJOR.MINOR
		for _, s := range subjectsOf(e) {
			v, ok := parseVer(s.version)
			if !ok {
				return nil, fmt.Errorf("%s: %s %q is no version", e.Key(), s.name, s.version)
			}
			top[s.kind+"/"+s.name] = max(top[s.kind+"/"+s.name], v[0])
			i := slices.IndexFunc(groups, func(g *group) bool {
				return g.fixture == e.Fixture && g.subject.kind == s.kind && g.name == s.name && g.major == v[0]
			})
			if i < 0 {
				groups = append(groups, &group{fixture: e.Fixture, subject: s, major: v[0], parts: min(len(v), 2), max: v, newest: e})
				continue
			}
			g := groups[i]
			if cmpVer(v, g.max) > 0 {
				g.max = v
			}
			if gv, _ := parseVer(g.newest.Version); cmpVer(ev, gv) >= 0 {
				g.newest = e
			}
		}
	}
	cache := map[string][]Release{}
	var out []Finding
	for _, g := range groups {
		src, err := sourceOf(g.kind, g.name)
		if err != nil {
			return nil, err
		}
		rels, ok := cache[src]
		if !ok {
			if rels, err = up.Releases(src); err != nil {
				return nil, fmt.Errorf("%s: %w", src, err)
			}
			if !slices.ContainsFunc(rels, func(r Release) bool { _, ok := parseVer(r.Version); return ok && !r.Pre }) {
				return nil, fmt.Errorf("%s: no release versions", src)
			}
			cache[src] = rels
		}
		if f, ok := scoutGroup(g, rels, top[g.kind+"/"+g.name]); ok {
			out = append(out, f)
		}
	}
	eol, err := scoutEOL(entries, up)
	if err != nil {
		return nil, err
	}
	return append(out, eol...), nil
}

// scoutEOL reports each listed runtime version past end of support, once per
// runtime and version however many fixtures run it. A version the source
// does not know is an error: silence would read as "still supported".
func scoutEOL(entries []Entry, up Upstream) ([]Finding, error) {
	versions := map[string][]string{}
	for _, e := range entries {
		if !slices.Contains(versions[e.Runtime], e.Version) {
			versions[e.Runtime] = append(versions[e.Runtime], e.Version)
		}
	}
	var out []Finding
	for _, runtime := range slices.Sorted(maps.Keys(versions)) {
		eol, err := up.EOL(runtime)
		if err != nil {
			return nil, fmt.Errorf("end of support of %s: %w", runtime, err)
		}
		for _, v := range slices.Sorted(slices.Values(versions[runtime])) {
			past, ok := eol[v]
			if !ok {
				return nil, fmt.Errorf("end of support: the source knows no %s %s", runtime, v)
			}
			if past {
				f := Finding{Kind: "eol", Line: runtime, Version: v}
				f.Title = IssueTitle(f)
				out = append(out, f)
			}
		}
	}
	return out, nil
}

// scoutGroup finds the newest release that is a new minor of the line's
// major or, for the line holding the highest major, a newer major.
func scoutGroup(g *group, rels []Release, topMajor int) (Finding, bool) {
	var best []int
	var raw string
	for _, rel := range rels {
		r := rel.Version
		v, ok := parseVer(r)
		if !ok || rel.Pre {
			continue
		}
		// gdtoolkit follows Godot's major, which the runtime line decides.
		newMajor := v[0] > g.major && g.major == topMajor && g.name != "gdtoolkit"
		newMinor := g.parts == 2 && v[0] == g.major && part(v, 1) > part(g.max, 1)
		if (newMajor || newMinor) && (best == nil || cmpVer(v, best) > 0) {
			best, raw = v, strings.TrimPrefix(r, "v")
		}
	}
	if best == nil {
		if g.kind == "runtime" {
			return scoutPre(g, rels, topMajor)
		}
		return Finding{}, false
	}
	version := strconv.Itoa(best[0])
	if g.parts == 2 {
		version += "." + strconv.Itoa(part(best, 1))
	}
	probe := g.newest
	probe.Addons = slices.Clone(probe.Addons)
	probe.Tools = maps.Clone(probe.Tools)
	switch g.kind {
	case "runtime":
		probe.Version = version
	case "addon":
		for i := range probe.Addons {
			if probe.Addons[i].Name == g.name {
				probe.Addons[i].Version = raw
			}
		}
	default:
		probe.Tools[g.name] = version + ".*"
	}
	f := Finding{Kind: "new", Line: g.fixture + "/" + g.name, Version: version, Probe: &probe, TOML: entryTOML(probe)}
	f.Title = IssueTitle(f)
	return f, true
}

// scoutPre finds the newest pre-release of a runtime line's next version, by
// the same rule a stable one would have to meet. It runs only when no stable
// release is new, so a released version always wins over its candidates.
func scoutPre(g *group, rels []Release, topMajor int) (Finding, bool) {
	var best []int
	var raw string
	for _, rel := range rels {
		v, ok := parsePre(rel.Version)
		if !ok || !rel.Pre {
			continue
		}
		newMajor := v[0] > g.major && g.major == topMajor
		newMinor := v[0] == g.major && v[1] > part(g.max, 1)
		if !newMajor && !newMinor {
			continue
		}
		// major, minor, stage, number: the newest candidate sorts last.
		if key := append(v, preOrder(rel.Version)...); best == nil || cmpVer(key, best) > 0 {
			best, raw = key, rel.Version
		}
	}
	if best == nil {
		return Finding{}, false
	}
	probe := g.newest
	probe.Addons = slices.Clone(probe.Addons)
	probe.Tools = maps.Clone(probe.Tools)
	probe.Version = fmt.Sprintf("%d.%d", best[0], best[1])
	probe.Pre = raw
	f := Finding{Kind: "pre", Line: g.fixture + "/" + g.name, Version: raw, Probe: &probe}
	f.Title = IssueTitle(f)
	return f, true
}

// entryTOML writes e as an [[entry]] block in the form compat.toml uses.
func entryTOML(e Entry) string {
	q := strconv.Quote
	var b strings.Builder
	fmt.Fprintf(&b, "[[entry]]\nstack = %s\nruntime = %s\nversion = %s\nfixture = %s\n", q(e.Stack), q(e.Runtime), q(e.Version), q(e.Fixture))
	kinds := make([]string, len(e.Kinds))
	for i, k := range e.Kinds {
		kinds[i] = q(k)
	}
	fmt.Fprintf(&b, "kinds = [%s]\n", strings.Join(kinds, ", "))
	if len(e.Addons) > 0 {
		var as []string
		for _, a := range e.Addons {
			as = append(as, fmt.Sprintf("{ name = %s, version = %s }", q(a.Name), q(a.Version)))
		}
		fmt.Fprintf(&b, "addons = [%s]\n", strings.Join(as, ", "))
	}
	if len(e.Tools) > 0 {
		var ts []string
		for _, n := range slices.Sorted(maps.Keys(e.Tools)) {
			ts = append(ts, n+" = "+q(e.Tools[n]))
		}
		fmt.Fprintf(&b, "tools = { %s }\n", strings.Join(ts, ", "))
	}
	return b.String()
}

var titleRe = regexp.MustCompile(`^compat: (\S+)/(\S+) (\S+) is out$`)

// eolTitleRe is compiled on first use, like preRe.
var eolTitleRe = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^compat: (\S+) (\S+) is past end of support$`)
})

// Resolved reports whether the list holds, for the line of an issue title,
// a version at least the one the title announces; for an end-of-support
// title, whether the list no longer holds that runtime version.
func Resolved(entries []Entry, title string) bool {
	if m := eolTitleRe().FindStringSubmatch(title); m != nil {
		return !slices.ContainsFunc(entries, func(e Entry) bool { return e.Runtime == m[1] && e.Version == m[2] })
	}
	m := titleRe.FindStringSubmatch(title)
	if m == nil {
		return false
	}
	want, ok := parseVer(m[3])
	if !ok {
		return false
	}
	for _, e := range entries {
		if e.Fixture != m[1] {
			continue
		}
		for _, s := range subjectsOf(e) {
			if v, ok := parseVer(s.version); ok && s.name == m[2] && cmpVer(v[:min(len(v), len(want))], want) >= 0 {
				return true
			}
		}
	}
	return false
}

// HTTPUpstream reads the real sources. Client may be nil; Token, when set, is
// sent to the GitHub API.
type HTTPUpstream struct {
	Client *http.Client
	Token  string
}

// Releases fetches source and reads its releases.
func (h HTTPUpstream) Releases(source string) ([]Release, error) {
	body, err := h.get(source)
	if err != nil {
		return nil, err
	}
	vs, err := parseReleases(source, body)
	if err != nil {
		return nil, fmt.Errorf("unreadable answer: %w", err)
	}
	return vs, nil
}

// EOL fetches the runtime's releases from endoflife.date and reads which
// major.minor versions are past end of support.
func (h HTTPUpstream) EOL(runtime string) (map[string]bool, error) {
	body, err := h.get(fmt.Sprintf(eolSource, runtime))
	if err != nil {
		return nil, err
	}
	var doc struct {
		Result struct {
			Releases []struct {
				Name  string `json:"name"`
				IsEOL bool   `json:"isEol"`
			} `json:"releases"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("unreadable answer: %w", err)
	}
	out := map[string]bool{}
	for _, r := range doc.Result.Releases {
		out[r.Name] = r.IsEOL
	}
	return out, nil
}

// get fetches source; the GitHub API gets the token when there is one.
func (h HTTPUpstream) get(source string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, source, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if h.Token != "" && strings.HasPrefix(source, "https://api.github.com/") {
		req.Header.Set("Authorization", "Bearer "+h.Token)
	}
	resp, err := cmp.Or(h.Client, http.DefaultClient).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// parseReleases reads the releases out of the answer of source, each in its
// own format. Only the runtimes' sources report pre-releases; for addons and
// tools a flagged pre-release is left out, as the list never pins one.
func parseReleases(source string, body []byte) ([]Release, error) {
	var out []Release
	switch {
	case source == pythonSource:
		var rs []struct {
			Name string `json:"name"`
			Pre  bool   `json:"pre_release"`
		}
		if err := json.Unmarshal(body, &rs); err != nil {
			return nil, err
		}
		for _, r := range rs {
			// python.org also lists "Python install manager 26.4 beta 3".
			if v, ok := strings.CutPrefix(r.Name, "Python "); ok && v != "" && v[0] >= '0' && v[0] <= '9' {
				out = append(out, Release{v, r.Pre})
			}
		}
	case source == goSource:
		var rs []struct {
			Version string `json:"version"`
			Stable  bool   `json:"stable"`
		}
		if err := json.Unmarshal(body, &rs); err != nil {
			return nil, err
		}
		for _, r := range rs {
			out = append(out, Release{strings.TrimPrefix(r.Version, "go"), !r.Stable})
		}
	case strings.HasPrefix(source, "https://pypi.org/"):
		var doc struct {
			Releases map[string][]pypiFile `json:"releases"`
		}
		if err := json.Unmarshal(body, &doc); err != nil {
			return nil, err
		}
		// A release without files, or with every file yanked, is not one.
		for _, v := range slices.Sorted(maps.Keys(doc.Releases)) {
			if slices.ContainsFunc(doc.Releases[v], func(f pypiFile) bool { return !f.Yanked }) {
				out = append(out, Release{Version: v})
			}
		}
	default: // GitHub releases
		var rs []struct {
			Tag   string `json:"tag_name"`
			Draft bool   `json:"draft"`
			Pre   bool   `json:"prerelease"`
		}
		if err := json.Unmarshal(body, &rs); err != nil {
			return nil, err
		}
		for _, r := range rs {
			if r.Draft {
				continue
			}
			if source != godotSource {
				if !r.Pre {
					out = append(out, Release{Version: strings.TrimPrefix(r.Tag, "v")})
				}
				continue
			}
			// Godot tags a stable release "<version>-stable" and flags a
			// pre-release such as "4.8-beta2"; an unflagged tag without
			// -stable is neither.
			if v, stable := strings.CutSuffix(r.Tag, "-stable"); stable && !r.Pre {
				out = append(out, Release{Version: v})
			} else if r.Pre {
				out = append(out, Release{r.Tag, true})
			}
		}
	}
	return out, nil
}

// pypiFile is the part of a PyPI release file that decides whether it counts.
type pypiFile struct {
	Yanked bool `json:"yanked"`
}
