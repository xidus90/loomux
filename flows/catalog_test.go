package flows_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/flows"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/blocks"
	"github.com/xidus90/loomux/internal/flow/journal"
	"github.com/xidus90/loomux/internal/flow/load"
	"github.com/xidus90/loomux/internal/flow/model"
	"github.com/xidus90/loomux/internal/flow/runner"
)

var update = flag.Bool("update", false, "write the golden journals instead of comparing")

// script is a catalog flow's _test/script.toml: the run's options, then one
// step per visit that needs an answer from outside, in the order the run
// asks. An agent step carries the reply fields, or an error the fake model
// raises instead; a gate step carries the answer and names its node, which
// the run must be paused at. An agent step names its node too, and the
// journal must show the model's answers going to those nodes in that order.
type script struct {
	Options map[string]string `toml:"options"`
	Steps   []step            `toml:"step"`
}

type step struct {
	Node   string         `toml:"node"`
	Reply  map[string]any `toml:"reply"`
	Tokens int            `toml:"tokens"`
	Error  string         `toml:"error"`
	Answer *string        `toml:"answer"`
}

// fileJournal is the journal on disk, as a real run keeps it: the golden file
// holds the bytes journal.Append writes, so the test writes them the same way.
type fileJournal struct{ path string }

func (j fileJournal) Entries() ([]journal.Entry, error)      { return journal.Entries(j.path) }
func (j fileJournal) Append(entry journal.Entry) error       { return journal.Append(j.path, entry) }
func (j fileJournal) Pending() (*journal.PendingGate, error) { return journal.Pending(j.path) }

// halfSecondClock moves half a second per call from the epoch, so a duration
// in a golden journal is the same number on every machine.
func halfSecondClock() runner.Clock {
	now := time.Unix(0, 0)
	return func() time.Time {
		now = now.Add(500 * time.Millisecond)
		return now
	}
}

// A step is a gate's answer, a model's error or a model's reply, and a field
// of another kind beside it would be dropped without a word.
func TestAScriptStepWithAFieldOfAnotherKindIsRefused(t *testing.T) {
	for source, want := range map[string]string{
		"[[step]]\nnode = \"a\"\nanswer = \"yes\"\nreply = { ok = true }\n":                                                  `step 1 (a): an answer takes no reply, tokens or error`,
		"[[step]]\nnode = \"a\"\nanswer = \"yes\"\nerror = \"boom\"\n":                                                       `step 1 (a): an answer takes no reply, tokens or error`,
		"[[step]]\nnode = \"a\"\nanswer = \"yes\"\ntokens = 3\n":                                                             `step 1 (a): an answer takes no reply, tokens or error`,
		"[[step]]\nnode = \"a\"\nreply = { ok = true }\n\n[[step]]\nnode = \"b\"\nerror = \"boom\"\nreply = { ok = true }\n": `step 2 (b): an error takes no reply or tokens`,
		"[[step]]\nnode = \"a\"\nerror = \"boom\"\ntokens = 3\n":                                                             `step 1 (a): an error takes no reply or tokens`,
		"[[step]]\nnode = \"a\"\nanswer = \"yes\"\ncolour = 3\n":                                                             `unknown keys [step.colour]`,
	} {
		if _, err := parseScript([]byte(source)); err == nil || err.Error() != want {
			t.Errorf("%q: err = %v, want %q", source, err, want)
		}
	}
	plan, err := parseScript([]byte("[[step]]\nnode = \"a\"\nreply = { ok = true }\ntokens = 3\n\n[[step]]\nnode = \"g\"\nanswer = \"\"\n"))
	if err != nil || len(plan.Steps) != 2 {
		t.Fatalf("plan %+v, err %v", plan, err)
	}
}

// readScript reads a script from disk. The script is not in the binary -- the
// embed directive leaves _test/ out.
func readScript(t *testing.T, path string) script {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("a catalog flow needs _test/script.toml: %v", err)
	}
	plan, err := parseScript(data)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return plan
}

// parseScript reads a script's text. A key it does not know is a typo, and a
// field a step's kind does not read -- a reply beside an answer, tokens
// beside an error -- would be dropped; either would lose a step's answer
// without a word, so both are refused.
func parseScript(data []byte) (script, error) {
	var plan script
	meta, err := toml.Decode(string(data), &plan)
	if err != nil {
		return script{}, err
	}
	if unknown := meta.Undecoded(); len(unknown) > 0 {
		return script{}, fmt.Errorf("unknown keys %v", unknown)
	}
	for i, s := range plan.Steps {
		replies := s.Reply != nil || s.Tokens != 0
		switch {
		case s.Answer != nil && (replies || s.Error != ""):
			return script{}, fmt.Errorf("step %d (%s): an answer takes no reply, tokens or error", i+1, s.Node)
		case s.Error != "" && replies:
			return script{}, fmt.Errorf("step %d (%s): an error takes no reply or tokens", i+1, s.Node)
		}
	}
	return plan, nil
}

// The contribution test: every catalog flow, loaded as the binary ships it,
// runs against the fake model along its script, and its journal is the
// golden one byte for byte. A journal is what every later resume reads, so a
// change in one byte of it is a change in what a run of this build finds.
func TestEveryCatalogFlowRunsToItsGoldenJournal(t *testing.T) {
	catalog, err := flow.NewCatalog([]flow.Block{blocks.Agent{}, blocks.Gate{}, blocks.Exit{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range flows.Names() {
		t.Run(name, func(t *testing.T) {
			folder := filepath.Join("catalog", name)
			if _, err := os.Stat(filepath.Join(folder, "README.md")); err != nil {
				t.Fatalf("a catalog flow needs a README.md: %v", err)
			}
			plan := readScript(t, filepath.Join(folder, "_test", "script.toml"))
			found, err := load.Find(t.TempDir(), flows.FS(), config.FlowSettings{}, name)
			if err != nil {
				t.Fatal(err)
			}
			graph, err := load.Load(found, catalog)
			if err != nil {
				t.Fatal(err)
			}
			params, err := load.Params(graph, plan.Options)
			if err != nil {
				t.Fatal(err)
			}

			var answers []model.Answer
			var agentNodes []string
			var gates []step
			for _, s := range plan.Steps {
				switch {
				case s.Answer != nil:
					gates = append(gates, s)
					continue
				case s.Error != "":
					answers = append(answers, model.Answer{Err: errors.New(s.Error)})
				default:
					answers = append(answers, model.Answer{Reply: model.Reply{Fields: s.Reply, Tokens: s.Tokens}})
				}
				agentNodes = append(agentNodes, s.Node)
			}
			fake := model.NewFake(answers...)
			path := filepath.Join(t.TempDir(), "journal.jsonl")
			run := runner.New(runner.Options{
				Graph: graph, Catalog: catalog, Journal: fileJournal{path},
				Env: flow.Env{Params: params, Model: fake}, Clock: halfSecondClock(),
			})

			ctx := context.Background()
			result, err := run.Run(ctx)
			for err == nil && result.Status == "paused" {
				if len(gates) == 0 {
					t.Fatalf("the run paused at %s, and the script has no answer left", result.Node)
				}
				next := gates[0]
				gates = gates[1:]
				if next.Node != result.Node {
					t.Fatalf("the run paused at %s, and the script's next answer is for %s", result.Node, next.Node)
				}
				result, err = run.Resume(ctx, next.Answer)
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(gates) > 0 {
				t.Errorf("the run ended %s with %d gate steps left, the first for %s", result.Status, len(gates), gates[0].Node)
			}
			if asked := len(fake.Seen()); asked != len(answers) {
				t.Errorf("the model was asked %d times, and the script has %d agent steps", asked, len(answers))
			}
			entries, err := journal.Entries(path)
			if err != nil {
				t.Fatal(err)
			}
			var answered []string
			for _, entry := range entries {
				if entry.Kind == "agent" {
					answered = append(answered, entry.Node)
				}
			}
			if !slices.Equal(answered, agentNodes) {
				t.Errorf("the journal has agent visits %v, and the script's agent steps name %v", answered, agentNodes)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			golden := filepath.Join(folder, "_test", "journal.jsonl")
			// A run the checks above failed writes no golden journal: the
			// comparison below still runs and says how far it strayed.
			if *update && !t.Failed() {
				if err := os.WriteFile(golden, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("%v -- run go test ./flows -update to write it", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("the journal is not the golden one\n got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}
