package verify

import (
	"cmp"
	"errors"
	"fmt"
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

// find is the first binary a source names that exists, which source named
// it, and what was looked at when none did.
func (s godotSources) find() (bin, from string, looked []string) {
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
	for _, name := range []string{"godot", "godot4"} {
		if p, err := s.look(name); err == nil {
			return s.console(p), "PATH", nil
		}
	}
	return "", "", append(looked, "neither godot nor godot4 is on PATH")
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

// isConsoleBuild says whether p is named like Godot's console build.
func isConsoleBuild(p string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSuffix(p, filepath.Ext(p))), "_console")
}

// resolve is PlanEnv.Godot: the binary for the project in dir, or the state
// the lane takes and why.
func (s godotSources) resolve(dir string) (string, State, string) {
	bin, from, looked := s.find()
	if bin == "" {
		return "", StateMissingTool, "no Godot binary: " + strings.Join(looked, "; ")
	}
	want, mono, err := projectGodot(filepath.Join(dir, "project.godot"))
	if err != nil {
		return "", StateUnready, err.Error()
	}
	out, err := s.version(bin)
	if errors.Is(err, errBudget) {
		return "", StateBudget, "not started: the budget is spent before " + bin + " said its version"
	}
	got, hasMono, ok := parseGodotVersion(out)
	if err != nil || !ok {
		cause := "its --version output names none"
		if err != nil {
			cause = err.Error()
		}
		note := fmt.Sprintf("%s (%s) gave no version: %s", bin, from, cause)
		if s.goos == "windows" && !isConsoleBuild(bin) {
			note += "; the window build writes nothing to stdout, so name the _console.exe"
		}
		return "", StateUnready, note
	}
	if got != want || (mono && !hasMono) {
		need := want
		if mono {
			need += " with .NET (mono)"
		}
		return "", StateUnready, fmt.Sprintf("Godot %s found (%s: %s), project.godot wants %s. Set GODOT_BIN or [verify.gdscript] godot.", got, from, bin, need)
	}
	return bin, "", ""
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

// projectGodot reads the version a project wants from config/features and
// whether its [dotnet] section asks for the mono build.
func projectGodot(path string) (version string, mono bool, err error) {
	unread := func(err error) error {
		return fmt.Errorf("%s cannot be read, so there is no Godot version to check against: %w", path, err)
	}
	// Whole, not by line: a packed array can run a line past any buffer.
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, unread(err)
	}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "[dotnet]" {
			mono = true
		}
		if rest, found := strings.CutPrefix(line, "config/features="); found && version == "" {
			if m := featureVersion.FindStringSubmatch(rest); m != nil {
				version = m[1]
			}
		}
	}
	if version == "" {
		return "", false, fmt.Errorf("%s names no version in config/features", path)
	}
	return version, mono, nil
}
