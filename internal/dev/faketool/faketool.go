// Package faketool answers for a checking tool from a fixture, so that a
// recording or a replay of `check` exercises the chain and not the tools.
// One executable lies on PATH under every tool name the chain calls; the
// fixture says, per command line, what the tool prints, writes and exits with.
package faketool

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FixtureName is the fixture's file name in the root of a world.
const FixtureName = "faketool.json"

// FixtureEnv names the fixture the faketool executable answers from.
const FixtureEnv = "LOOMUX_FAKE_TOOL_FIXTURE"

// Write is one file a tool leaves behind, relative to its working directory.
// `{{DIR}}` in Content stands for that directory in forward slashes.
type Write struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Answer is what the tool does for every command line that starts with Prefix.
type Answer struct {
	Prefix  string  `json:"prefix"`
	Exit    int     `json:"exit"`
	Stdout  string  `json:"stdout"`
	SleepMS int     `json:"sleep_ms,omitempty"`
	Writes  []Write `json:"writes,omitempty"`
}

// Fixture holds the answers of every tool in one world.
type Fixture struct {
	Answers []Answer `json:"answers"`
}

// sleep is the seam a test replaces to see a delay without waiting for it.
var sleep = time.Sleep

// Load reads a fixture. A world without one has tools that know no answer.
func Load(path string) (*Fixture, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Fixture{}, nil
	}
	if err != nil {
		return nil, err
	}
	var f Fixture
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &f, nil
}

// The tool is named by its base name without .exe: the recorder finds it on
// PATH as uv.exe, the replay seam sees whatever argv the plan built.
func line(argv []string) string {
	head := strings.TrimSuffix(filepath.Base(argv[0]), ".exe")
	return strings.Join(append([]string{head}, argv[1:]...), " ")
}

// Match finds the answer with the longest prefix that ends on a word boundary
// of the command line; on a tie the later answer wins.
func (f *Fixture) Match(argv []string) (Answer, bool) {
	l := line(argv)
	best, found := Answer{}, false
	for _, a := range f.Answers {
		if (l == a.Prefix || strings.HasPrefix(l, a.Prefix+" ")) && len(a.Prefix) >= len(best.Prefix) {
			best, found = a, true
		}
	}
	return best, found
}

// Apply leaves the answer's files in dir.
func (f *Fixture) Apply(a Answer, dir string) error {
	for _, w := range a.Writes {
		target := filepath.Join(dir, filepath.FromSlash(w.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		body := strings.ReplaceAll(w.Content, "{{DIR}}", filepath.ToSlash(dir))
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Main is the executable. args is the whole os.Args, program name included,
// because the program name is the tool. Anything it cannot answer exits 127,
// as a shell does for a command it cannot find, so a gap in a fixture is loud.
func Main(args []string, getenv func(string) string, cwd string, stdout, stderr io.Writer) int {
	f, err := Load(getenv(FixtureEnv))
	if err != nil {
		fmt.Fprintf(stderr, "faketool: %v\n", err)
		return 127
	}
	a, ok := f.Match(args)
	if !ok {
		fmt.Fprintf(stderr, "faketool: no answer for %q\n", line(args))
		return 127
	}
	sleep(time.Duration(a.SleepMS) * time.Millisecond)
	if err := f.Apply(a, cwd); err != nil {
		fmt.Fprintf(stderr, "faketool: %v\n", err)
		return 127
	}
	io.WriteString(stdout, a.Stdout)
	return a.Exit
}
