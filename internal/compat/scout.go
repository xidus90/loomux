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
)

// Upstream answers the stable release versions a source publishes, without
// a leading "v" or other decoration.
type Upstream interface {
	Releases(source string) ([]string, error)
}

// Finding is a version newer than everything its line holds. Line is
// <fixture>/<name>, Version the new major.minor (the major for a pin of a
// major), Probe the entry that would try it and TOML that entry as an
// [[entry]] block, ready to paste into compat.toml.
type Finding struct {
	Title   string `json:"title"`
	Line    string `json:"line"`
	Version string `json:"version"`
	Probe   Entry  `json:"probe"`
	TOML    string `json:"toml"`
}

// IssueTitle is the title the finding's issue carries.
func IssueTitle(f Finding) string { return "compat: " + f.Line + " " + f.Version + " is out" }

const (
	godotSource    = "https://api.github.com/repos/godotengine/godot/releases?per_page=100"
	pythonSource   = "https://www.python.org/api/v2/downloads/release/?is_published=true"
	goSource       = "https://go.dev/dl/?mode=json"
	githubReleases = "https://api.github.com/repos/%s/releases?per_page=100"
	pypiSource     = "https://pypi.org/pypi/%s/json"
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
	cache := map[string][]string{}
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
			if !slices.ContainsFunc(rels, func(r string) bool { _, ok := parseVer(r); return ok }) {
				return nil, fmt.Errorf("%s: no release versions", src)
			}
			cache[src] = rels
		}
		if f, ok := scoutGroup(g, rels, top[g.kind+"/"+g.name]); ok {
			out = append(out, f)
		}
	}
	return out, nil
}

// scoutGroup finds the newest release that is a new minor of the line's
// major or, for the line holding the highest major, a newer major.
func scoutGroup(g *group, rels []string, topMajor int) (Finding, bool) {
	var best []int
	var raw string
	for _, r := range rels {
		v, ok := parseVer(r)
		if !ok {
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
	f := Finding{Line: g.fixture + "/" + g.name, Version: version, Probe: probe, TOML: entryTOML(probe)}
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

// Resolved reports whether the list holds, for the line of an issue title,
// a version at least the one the title announces.
func Resolved(entries []Entry, title string) bool {
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

// Releases fetches source and reads its stable versions.
func (h HTTPUpstream) Releases(source string) ([]string, error) {
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	vs, err := parseReleases(source, body)
	if err != nil {
		return nil, fmt.Errorf("unreadable answer: %w", err)
	}
	return vs, nil
}

// parseReleases reads the stable versions out of the answer of source, each
// in its own format.
func parseReleases(source string, body []byte) ([]string, error) {
	var out []string
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
			if v, ok := strings.CutPrefix(r.Name, "Python "); ok && !r.Pre {
				out = append(out, v)
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
			if r.Stable {
				out = append(out, strings.TrimPrefix(r.Version, "go"))
			}
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
				out = append(out, v)
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
			v, stable := strings.CutSuffix(r.Tag, "-stable")
			if r.Draft || r.Pre || (source == godotSource && !stable) {
				continue
			}
			out = append(out, strings.TrimPrefix(v, "v"))
		}
	}
	return out, nil
}

// pypiFile is the part of a PyPI release file that decides whether it counts.
type pypiFile struct {
	Yanked bool `json:"yanked"`
}
