package cli

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/flows"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/blocks"
	"github.com/xidus90/loomux/internal/flow/load"
	"github.com/xidus90/loomux/internal/flow/model"
	"github.com/xidus90/loomux/internal/flow/runs"
)

// flowHarness is the outside a test hands to flowCLI: both streams in buffers, a
// clock that moves half a second per call, a model that answers from a queue,
// the real blocks and the real catalog.
type flowHarness struct {
	deps   flowDeps
	fake   *model.Fake
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func newFlowHarness(answers ...model.Answer) *flowHarness {
	h := &flowHarness{fake: model.NewFake(answers...), stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	now := time.Unix(0, 0)
	h.deps = flowDeps{
		Stdout: h.stdout,
		Stderr: h.stderr,
		Clock: func() time.Time {
			now = now.Add(500 * time.Millisecond)
			return now
		},
		Models:  func(string) (model.Model, error) { return h.fake, nil },
		Blocks:  []flow.Block{blocks.Agent{}, blocks.Gate{}, blocks.Exit{}},
		Bundled: flows.FS(),
	}
	return h
}

// run calls flowCLI with --root appended, after emptying both buffers so that
// each call's output stands alone.
func (h *flowHarness) run(root string, args ...string) int {
	return h.runHere(append(args, "--root", root)...)
}

// runHere calls flowCLI without --root: the root is found from the working
// directory.
func (h *flowHarness) runHere(args ...string) int {
	h.stdout.Reset()
	h.stderr.Reset()
	return flowCLI(args, h.deps)
}

// flowDone is the one answer the example flow's draft needs to reach its gate.
func flowDone() model.Answer {
	return model.Answer{Reply: model.Reply{Fields: map[string]any{"verdict": "done", "count": 2}, Tokens: 120}}
}

// flowPausedExample is an empty project whose first run of the bundled example
// waits at its gate.
func flowPausedExample(t *testing.T, h *flowHarness) string {
	t.Helper()
	root := t.TempDir()
	if code := h.run(root, "run", "example"); code != flowExitPaused {
		t.Fatalf("the run did not pause: exit %d, stderr %q", code, h.stderr.String())
	}
	return root
}

// flowProjectExample copies the bundled example into the project's own flow
// folder, so that a test can change what the example is made of.
func flowProjectExample(t *testing.T, root string) {
	t.Helper()
	bundled, err := fs.Sub(flows.FS(), "example")
	if err != nil {
		t.Fatal(err)
	}
	err = fs.WalkDir(bundled, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		body, err := fs.ReadFile(bundled, path)
		if err != nil {
			return err
		}
		flowWrite(t, root, load.Dir+"/example/"+path, string(body))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// flowAsk is askFlow as the project's own flow of that name, with its question.
func flowAsk(t *testing.T, root, name string) {
	t.Helper()
	flowWrite(t, root, load.Dir+"/"+name+"/flow.toml", askFlow)
	flowWrite(t, root, load.Dir+"/"+name+"/questions/confirm.md", "Ship it?\n")
}

// flowConfig writes the project's .loomux/config.toml, where both readers of a
// flow command look.
func flowConfig(t *testing.T, root, body string) {
	t.Helper()
	path := config.ManifestPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	flowOverwrite(t, path, body)
}

// flowWrite writes one file below root, its directories included.
func flowWrite(t *testing.T, root, relative, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func flowOverwrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func flowRemove(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
}

// flowLines are a file's lines, without the empty one after the last newline.
func flowLines(t *testing.T, path string) []string {
	t.Helper()
	return strings.Split(strings.TrimSuffix(string(flowReadBytes(t, path)), "\n"), "\n")
}

func flowReadBytes(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// flowRunFiles are the names under a project's runs directory; none when it is
// not there.
func flowRunFiles(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(runs.Dir)))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// flowReadMarker is run id's marker, which the test needs to be there.
func flowReadMarker(t *testing.T, root, id string) *runs.Marker {
	t.Helper()
	marker, err := runs.ReadMarker(runs.MarkerPath(root, id))
	if err != nil || marker == nil {
		t.Fatalf("marker = %+v, err = %v", marker, err)
	}
	return marker
}

// measureBlock is a node kind that needs the run's baseline and does nothing
// else, so that the refusal for a missing one has a flow to refuse.
type measureBlock struct{}

func (measureBlock) Kind() string                          { return "measure" }
func (measureBlock) Check(flow.Node) []string              { return nil }
func (measureBlock) Texts(flow.Node) []flow.Text           { return nil }
func (measureBlock) Writes(flow.Node) map[string]flow.Type { return nil }
func (measureBlock) NeedsBaseline() bool                   { return true }

func (measureBlock) Define(*flow.Graph, flow.Node, flow.Env) (flow.Definition, error) {
	return flow.Definition{}, nil
}

func (measureBlock) Run(context.Context, *flow.Graph, flow.Node, flow.State, flow.Env) (flow.Result, error) {
	return flow.Result{}, nil
}

// flakyBlock cannot be defined for a node named b. A run reaches b after a has
// taken its step, so the call fails with a journal already on disk.
type flakyBlock struct{ measureBlock }

func (flakyBlock) Kind() string        { return "flaky" }
func (flakyBlock) NeedsBaseline() bool { return false }

func (flakyBlock) Define(_ *flow.Graph, node flow.Node, _ flow.Env) (flow.Definition, error) {
	if node.Name == "b" {
		return flow.Definition{}, errors.New("the definition of b is gone")
	}
	return flow.Definition{}, nil
}

const measureFlow = `schema_version = 1

[flow]
start = "look"

[[node]]
name = "look"
kind = "measure"

[[edge]]
from = "look"
to   = "END"
`

const flakyFlow = `schema_version = 1

[flow]
start = "a"

[[node]]
name = "a"
kind = "flaky"

[[node]]
name = "b"
kind = "flaky"

[[edge]]
from = "a"
to   = "b"

[[edge]]
from = "b"
to   = "END"
`

// askFlow is a gate and nothing else: a flow the real outside runs, since it
// asks no model.
const askFlow = `schema_version = 1

[flow]
start = "confirm"

[state]
answer      = { type = "string", default = "" }
answer_text = { type = "string", default = "" }

[[node]]
name     = "confirm"
kind     = "gate"
question = "questions/confirm.md"
choices  = ["yes", "no"]
answer   = "answer"

[[edge]]
from = "confirm"
to   = "END"
`

// pairFlow asks two providers, which a run with one model cannot serve.
const pairFlow = `schema_version = 1

[flow]
start = "one"

[state]
verdict = { type = "string", default = "" }

[[node]]
name        = "one"
kind        = "agent"
role        = "writer"
instruction = "instructions/one.md"
reply       = { verdict = "string" }

[[node]]
name        = "two"
kind        = "agent"
role        = "critic"
instruction = "instructions/one.md"
reply       = { verdict = "string" }

[[edge]]
from = "one"
to   = "two"

[[edge]]
from = "two"
to   = "END"
`

const pairConfig = `[agent.models.w]
provider = "claude"

[agent.models.c]
provider = "gemini"

[agent.roles]
writer = "w"
critic = "c"
`
