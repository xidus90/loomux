package verify

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/child"
)

// placeholderGodot is the Godot binary a gdscript lane names.
const placeholderGodot = "{godot}"

// godotSources are the outside answers {godot} needs: the variable, the
// configured path, PATH, which files exist, and the binary's own --version.
type godotSources struct {
	root, configured, goos string
	getenv                 func(string) string
	abs                    func(string) string
	look                   func(string) (string, error)
	exists                 func(string) bool
	readDir                func(string) ([]string, error)
	version                func(string) (string, error)
}

func runtimeGOOS() string { return runtime.GOOS }

var lookPath = exec.LookPath

// probeTimeout is how long a binary has to answer --version, and errBudget
// says the run's budget ran out before it did.
const probeTimeout = 30 * time.Second

var errBudget = errors.New("the budget is spent")

// GodotFor is the finder of a real run, asking each binary for its version
// once. project.godot is the source of the version a project wants (no
// guessed fallback); see the configuration docs for the search order. left
// is what remains of the run's budget, nil for none: the probe takes no longer
// than that, and a lane whose binary it could not ask in time is `budget`.
func GodotFor(root, configured string, start func(child.Spec) child.Result, left func() time.Duration) func(string) (string, State, string) {
	var mu sync.Mutex
	seen := map[string][2]string{}
	s := godotSources{
		root: root, configured: configured, goos: runtimeGOOS(),
		getenv: os.Getenv,
		abs: func(p string) string {
			a, _ := filepath.Abs(p)
			return cmp.Or(a, p)
		},
		look:   lookPath,
		exists: func(p string) bool { fi, err := os.Stat(p); return err == nil && !fi.IsDir() },
		readDir: func(dir string) ([]string, error) {
			entries, err := os.ReadDir(dir)
			if err != nil {
				return nil, err
			}
			var names []string
			for _, e := range entries {
				if !e.IsDir() {
					names = append(names, e.Name())
				}
			}
			return names, nil
		},
		version: func(bin string) (string, error) {
			mu.Lock()
			defer mu.Unlock()
			if v, ok := seen[bin]; ok {
				return v[0], errorOf(v[1])
			}
			timeout, capped := probeTimeout, false
			if left != nil {
				rest := left()
				if rest <= 0 {
					return "", errBudget
				}
				if rest < timeout {
					timeout, capped = rest, true
				}
			}
			res := start(child.Spec{Argv: []string{bin, "--version"}, Timeout: timeout})
			out, msg := res.Stdout, ""
			switch {
			case res.Err != nil:
				msg = res.Err.Error()
			case res.TimedOut && capped:
				// Not remembered: it says nothing of the binary.
				return "", errBudget
			case res.Code != 0:
				msg = "exit " + strconv.Itoa(res.Code)
			}
			seen[bin] = [2]string{out, msg}
			return out, errorOf(msg)
		},
	}
	return s.resolve
}

func errorOf(msg string) error {
	if msg == "" {
		return nil
	}
	return errors.New(msg)
}

// named is the binary a human named for this run or this project, which of
// the two named it, and what was looked at when neither did.
//
// These two sources keep their authority: a named binary of another line is
// refused rather than passed over, because naming one is a deliberate act and
// a wrong answer to it belongs in the open. Only the searched sources --
// GODOT_BIN_DIR and PATH, which nobody named for this project in particular
// -- are offered as candidates.
func (s godotSources) named() (bin, from string, looked []string) {
	if v := s.getenv("GODOT_BIN"); v != "" {
		// The lane runs the binary from its area, so a path relative to
		// loomux's own directory has to be pinned before anything uses it.
		v = s.abs(v)
		if s.exists(v) {
			return s.console(v), "GODOT_BIN", nil
		}
		looked = append(looked, "GODOT_BIN names "+v+", which is not there")
	} else {
		looked = append(looked, "GODOT_BIN is unset")
	}
	if s.configured != "" {
		p := filepath.FromSlash(s.configured)
		if !filepath.IsAbs(p) {
			p = filepath.Join(s.root, p)
		}
		if s.exists(p) {
			return s.console(p), "[verify.gdscript] godot", nil
		}
		looked = append(looked, "[verify.gdscript] godot names "+p+", which is not there")
	} else {
		looked = append(looked, "[verify.gdscript] godot is unset")
	}
	return "", "", looked
}

// candidate is one binary the search offers and the source that offered it.
type candidate struct{ Path, From string }

// candidates are the binaries to try in order where nobody named one: the
// engines of GODOT_BIN_DIR, then godot and godot4 from the PATH.
func (s godotSources) candidates() (cands []candidate, looked []string) {
	dir := s.getenv("GODOT_BIN_DIR")
	engines, err := dirCandidates(dir, s.readDir, s.console)
	switch {
	case dir == "":
		looked = append(looked, "GODOT_BIN_DIR is unset")
	case errors.Is(err, fs.ErrNotExist):
		looked = append(looked, "GODOT_BIN_DIR names "+dir+", which is not there")
	case err != nil:
		looked = append(looked, "GODOT_BIN_DIR names "+dir+", which cannot be read: "+err.Error())
	case len(engines) == 0:
		looked = append(looked, "GODOT_BIN_DIR names "+dir+", which holds no engine")
	default:
		for _, p := range engines {
			cands = append(cands, candidate{Path: p, From: "GODOT_BIN_DIR"})
		}
	}
	for _, name := range []string{"godot", "godot4"} {
		if p, err := s.look(name); err == nil {
			cands = append(cands, candidate{Path: s.console(p), From: "PATH"})
			break
		}
	}
	if !slices.ContainsFunc(cands, func(c candidate) bool { return c.From == "PATH" }) {
		looked = append(looked, "neither godot nor godot4 is on PATH")
	}
	return cands, looked
}

// console is the _console.exe beside a Windows build when there is one: the
// window build writes nothing to stdout.
func (s godotSources) console(p string) string {
	ext := filepath.Ext(p)
	stem := strings.TrimSuffix(p, ext)
	if s.goos != "windows" || !strings.EqualFold(ext, ".exe") || isConsoleBuild(p) {
		return p
	}
	if c := stem + "_console" + ext; s.exists(c) {
		return c
	}
	return p
}

// wayOut closes every note about a binary that does not fit: the three
// sources a human can set, in the order they are read.
const wayOut = "Set GODOT_BIN or [verify.gdscript] godot, or keep the line's engine in GODOT_BIN_DIR."

// isConsoleBuild says whether p is named like Godot's console build.
func isConsoleBuild(p string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSuffix(p, filepath.Ext(p))), "_console")
}

// resolve is PlanEnv.Godot: the binary for the project in dir, or the state
// the lane takes and why.
//
// The project is read first: without the line it wants there is nothing to
// choose a binary by. A binary a human named is then held to that line, and
// only where nobody named one are the searched candidates tried in turn, the
// first that fits winning. Each distinct binary answers --version once per
// run, so a machine of three lines pays three questions and no more.
func (s godotSources) resolve(dir string) (string, State, string) {
	want, mono, err := projectGodot(filepath.Join(dir, "project.godot"))
	if err != nil {
		return "", StateUnready, err.Error()
	}
	if bin, from, looked := s.named(); bin != "" {
		got, state, note, _ := s.check(bin, from, want, mono)
		if state != "" {
			note += " " + wayOut
		}
		return got, state, note
	} else if cands, searched := s.candidates(); len(cands) > 0 {
		var first State
		var firstNote string
		var named bool
		for _, c := range cands {
			bin, state, note, reported := s.check(c.Path, c.From, want, mono)
			switch {
			case state == "":
				return bin, "", ""
			case state == StateBudget:
				// The run ended, not the search: nothing was learnt of the
				// remaining candidates, and saying "wrong version" would
				// name a finding nobody made.
				return "", state, note
			case first == "" || (reported && !named):
				// A candidate that answered a version tells the reader more
				// than one that answered nothing, whatever their order: a zip
				// archive sorted among the engines must not hide the line
				// that was actually found.
				first, firstNote, named = state, note, reported
			}
		}
		return "", first, firstNote + " " + wayOut
	} else {
		return "", StateMissingTool, "no Godot binary: " + strings.Join(append(looked, searched...), "; ")
	}
}

// check holds one binary to the line the project wants. reported says whether
// the binary named a version at all, which is what makes its note worth
// showing over another candidate's silence.
func (s godotSources) check(bin, from string, want Line, mono bool) (_ string, _ State, note string, reported bool) {
	out, err := s.version(bin)
	if errors.Is(err, errBudget) {
		return "", StateBudget, "not started: the budget is spent before " + bin + " said its version", false
	}
	got, hasMono, ok := parseGodotVersion(out)
	if err != nil || !ok {
		cause := "its --version output names none"
		if err != nil {
			cause = err.Error()
		}
		note = fmt.Sprintf("%s (%s) gave no version: %s", bin, from, cause)
		if s.goos == "windows" && !isConsoleBuild(bin) {
			note += "; the window build writes nothing to stdout, so name the _console.exe"
		}
		return "", StateUnready, note, false
	}
	if !want.Matches(got) || (mono && !hasMono) {
		need := want.String()
		if mono {
			need += " with .NET (mono)"
		}
		return "", StateUnready, fmt.Sprintf("Godot %s found (%s: %s), project.godot wants %s.", got, from, bin, need), true
	}
	return bin, "", "", true
}

// parseGodotVersion reads `<major>.<minor>[.<patch>].<status>[.mono]…` from
// the first line of out that has that shape.
func parseGodotVersion(out string) (version string, mono, ok bool) {
	for _, line := range strings.Split(out, "\n") {
		tokens := strings.Split(strings.TrimSpace(line), ".")
		if len(tokens) < 3 || !digits(tokens[0]) || !digits(tokens[1]) {
			continue
		}
		return tokens[0] + "." + tokens[1], slices.Contains(tokens, "mono"), true
	}
	return "", false, false
}

func digits(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

var featureVersion = regexp.MustCompile(`"(\d+\.\d+)"`)

// configVersionLine is the major line a project file's own format version
// implies. 4 is Godot 3.1 and later, 5 is Godot 4, but only the 3 line gets a
// reading from it: 5 spans 4.0 to 4.8 and so measures nothing a comparison
// could use, while 4 is the whole of a line whose projects never wrote
// config/features. Godot 3.0 wrote 3 and is left out with every other value:
// it is a line nobody maintains, and guessing at it would serve no project.
var configVersionLine = map[string]Line{"4": {Major: "3"}}

// projectGodot reads the line a project wants and whether its [dotnet]
// section asks for the mono build.
//
// config/features is what Godot 4 writes and the first thing read; a Godot 3
// project has no such setting, and there config_version carries the line.
func projectGodot(path string) (want Line, mono bool, err error) {
	unread := func(err error) error {
		return fmt.Errorf("%s cannot be read, so there is no Godot version to check against: %w", path, err)
	}
	// Whole, not by line: a packed array can run a line past any buffer.
	data, err := os.ReadFile(path)
	if err != nil {
		return Line{}, false, unread(err)
	}
	var fromFormat Line
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "[dotnet]" {
			mono = true
		}
		if rest, found := strings.CutPrefix(line, "config/features="); found && want == (Line{}) {
			if m := featureVersion.FindStringSubmatch(rest); m != nil {
				major, minor, _ := strings.Cut(m[1], ".")
				want = Line{Major: major, Minor: minor}
			}
		}
		if rest, found := strings.CutPrefix(line, "config_version="); found && fromFormat == (Line{}) {
			fromFormat = configVersionLine[strings.TrimSpace(rest)]
		}
	}
	if want == (Line{}) {
		want = fromFormat
	}
	if want == (Line{}) {
		return Line{}, false, fmt.Errorf("%s names no version in config/features", path)
	}
	return want, mono, nil
}
