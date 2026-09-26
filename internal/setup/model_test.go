package setup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/model"
	"github.com/xidus90/loomux/internal/config"
)

// fakeOllama answers Has and Pull without a network and counts the pulls.
type fakeOllama struct {
	has     bool
	hasErr  error
	steps   [][3]any // status, completed, total, handed to progress in order
	pullErr error
	pulls   int
}

func (o *fakeOllama) Has(context.Context, string) (bool, error) { return o.has, o.hasErr }

func (o *fakeOllama) Pull(_ context.Context, _ string, progress func(string, int64, int64)) error {
	o.pulls++
	for _, s := range o.steps {
		progress(s[0].(string), s[1].(int64), s[2].(int64))
	}
	return o.pullErr
}

// useOllama makes every client of the test o; a non-nil err stands for a
// guard that refuses the endpoint.
func useOllama(t *testing.T, o *fakeOllama, err error) {
	t.Helper()
	old := newOllama
	newOllama = func(config.ModelSettings) (ollama, error) {
		if err != nil {
			return nil, err
		}
		return o, nil
	}
	t.Cleanup(func() { newOllama = old })
}

// No test of this package may reach an Ollama of the machine: every client
// is one that holds the model, unless a test says otherwise.
func TestMain(m *testing.M) {
	newOllama = func(config.ModelSettings) (ollama, error) { return &fakeOllama{has: true}, nil }
	os.Exit(m.Run())
}

func globalModel(t *testing.T, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(os.Getenv("LOOMUX_STATE_DIR"), "config.toml"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func modelDefault(f Facts) bool {
	for _, p := range Parts(f) {
		if p.ID == "model" {
			return p.Default
		}
	}
	panic("no part model")
}

func TestThePartModelIsOnForALocalOnlyProjectOrAnEnabledModel(t *testing.T) {
	for name, tc := range map[string]struct {
		project, global string
		want            bool
	}{
		"neither":          {"", "", false},
		"manual_cloud":     {"[privacy]\nmode = \"manual_cloud\"\n", "", false},
		"local_only":       {"[privacy]\nmode = \"local_only\"\n", "", true},
		"enabled":          {"", "[model]\nenabled = true\n", true},
		"disabled":         {"", "[model]\nenabled = false\n", false},
		"broken project":   {"[privacy\n", "", false},
		"privacy no table": {"privacy = 1\n", "", false},
	} {
		t.Run(name, func(t *testing.T) {
			root := world(t, map[string]string{configPath: tc.project})
			globalModel(t, tc.global)
			f := gather(t, root, "")
			if got := modelDefault(f); got != tc.want {
				t.Errorf("default = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestThePartModelNamesTheModel(t *testing.T) {
	root := world(t, map[string]string{})
	globalModel(t, "[model]\nname = \"qwen:7b\"\n")
	for _, p := range Parts(gather(t, root, "")) {
		if p.ID == "model" && p.Label != "pull the local model qwen:7b into Ollama when it is missing" {
			t.Errorf("label = %q", p.Label)
		}
	}
}

// buildModel plans the part model alone for the facts of a fresh world.
func buildModel(t *testing.T, global string) Plan {
	t.Helper()
	root := world(t, map[string]string{})
	globalModel(t, global)
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	for id := range c.Parts {
		c.Parts[id] = id == "model"
	}
	p, err := Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAMissingModelIsPulledAfterConfirmation(t *testing.T) {
	o := &fakeOllama{}
	useOllama(t, o, nil)
	p := buildModel(t, "[model]\nname = \"qwen:7b\"\n")
	want := []Action{{Part: "model", ID: "model-pull", Describe: "ollama pull qwen:7b"}}
	if !slices.Equal(p.Actions, want) || len(p.Notes) != 0 {
		t.Errorf("actions = %v, notes = %v", p.Actions, p.Notes)
	}
	// The plan asks and never pulls; a dry run shows it and ends there.
	if o.pulls != 0 {
		t.Errorf("the plan pulled %d times", o.pulls)
	}
}

func TestAModelThereIsLeftAlone(t *testing.T) {
	useOllama(t, &fakeOllama{has: true}, nil)
	if p := buildModel(t, ""); len(p.Actions)+len(p.Notes) != 0 {
		t.Errorf("actions = %v, notes = %v", p.Actions, p.Notes)
	}
}

func TestAnOllamaNotRunningIsANote(t *testing.T) {
	useOllama(t, &fakeOllama{hasErr: fmt.Errorf("%w: connection refused", model.ErrUnreachable)}, nil)
	p := buildModel(t, "[model]\nendpoint = \"http://localhost:4242\"\nname = \"qwen:7b\"\n")
	want := []string{"ollama is not running at http://localhost:4242; start it and run init again, or run: ollama pull qwen:7b"}
	if len(p.Actions) != 0 || !slices.Equal(p.Notes, want) {
		t.Errorf("actions = %v, notes = %v", p.Actions, p.Notes)
	}
}

// An Ollama that answers, but not with its list of models, is named as it
// answered, not as one that is not running.
func TestAnOllamaAnsweringOddlyIsNamed(t *testing.T) {
	useOllama(t, &fakeOllama{hasErr: errors.New("GET /api/tags answered 500 Internal Server Error")}, nil)
	p := buildModel(t, "[model]\nendpoint = \"http://localhost:4242\"\nname = \"qwen:7b\"\n")
	want := []string{"model: skipped; ollama at http://localhost:4242 could not be asked for qwen:7b: " +
		"GET /api/tags answered 500 Internal Server Error"}
	if len(p.Actions) != 0 || !slices.Equal(p.Notes, want) {
		t.Errorf("actions = %v, notes = %v", p.Actions, p.Notes)
	}
}

func TestARefusedEndpointIsANote(t *testing.T) {
	useOllama(t, nil, errors.New("[model] endpoint must stay on the loopback"))
	p := buildModel(t, "")
	if len(p.Actions) != 0 || !slices.Equal(p.Notes, []string{"model: skipped; [model] endpoint must stay on the loopback"}) {
		t.Errorf("actions = %v, notes = %v", p.Actions, p.Notes)
	}
}

// A global file that does not read stops no init; the part names it.
func TestUnreadableModelSettingsAreANote(t *testing.T) {
	p := buildModel(t, "[model]\nenabled = \"yes\"\n")
	if len(p.Actions) != 0 || len(p.Notes) != 1 || !strings.HasPrefix(p.Notes[0], "model: skipped; ") ||
		!strings.Contains(p.Notes[0], "config.toml") {
		t.Errorf("actions = %v, notes = %v", p.Actions, p.Notes)
	}
}

// The real client and its guard stand behind the seam.
func TestTheSeamIsTheLoopbackClient(t *testing.T) {
	old := newOllama
	newOllama = realOllama
	t.Cleanup(func() { newOllama = old })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer server.Close()
	p := buildModel(t, "[model]\nendpoint = \""+server.URL+"\"\nname = \"qwen:7b\"\n")
	if !slices.Equal(actions(p), []string{"model-pull"}) {
		t.Errorf("actions = %v, notes = %v", p.Actions, p.Notes)
	}
	p = buildModel(t, "[model]\nendpoint = \"http://192.0.2.1:11434\"\n")
	if len(p.Notes) != 1 || !strings.Contains(p.Notes[0], "must stay on the loopback") {
		t.Errorf("notes = %v", p.Notes)
	}
}

func TestPullModelReportsProgressThrottled(t *testing.T) {
	o := &fakeOllama{steps: [][3]any{
		{"pulling manifest", int64(0), int64(0)},
		{"pulling abc", int64(0), int64(1000)},
		{"pulling abc", int64(50), int64(1000)},
		{"pulling abc", int64(120), int64(1000)},
		{"pulling abc", int64(150), int64(1000)},
		{"pulling abc", int64(999), int64(1000)},
		{"pulling abc", int64(1000), int64(1000)},
		{"verifying sha256 digest", int64(0), int64(0)},
		{"verifying sha256 digest", int64(0), int64(0)},
		{"success", int64(0), int64(0)},
	}}
	useOllama(t, o, nil)
	var out bytes.Buffer
	if err := PullModel(context.Background(), config.ModelSettings{Name: "qwen:7b"}, &out); err != nil {
		t.Fatal(err)
	}
	want := "model-pull: pulling qwen:7b\n" +
		"model-pull: pulling manifest\n" +
		"model-pull: pulling abc 0%\n" +
		"model-pull: pulling abc 10%\n" +
		"model-pull: pulling abc 90%\n" +
		"model-pull: pulling abc 100%\n" +
		"model-pull: verifying sha256 digest\n" +
		"model-pull: success\n"
	if out.String() != want {
		t.Errorf("progress =\n%s\nwant\n%s", out.String(), want)
	}
}

func TestPullModelPassesFailuresOn(t *testing.T) {
	useOllama(t, &fakeOllama{pullErr: errors.New("file does not exist")}, nil)
	if err := PullModel(context.Background(), config.ModelSettings{Name: "x"}, &bytes.Buffer{}); err == nil ||
		err.Error() != "file does not exist" {
		t.Errorf("err = %v", err)
	}
	useOllama(t, nil, errors.New("refused"))
	if err := PullModel(context.Background(), config.ModelSettings{Name: "x"}, &bytes.Buffer{}); err == nil ||
		err.Error() != "refused" {
		t.Errorf("err = %v", err)
	}
}

// A failed pull is a note: init goes on, and the record is written.
func TestApplyTurnsAFailedPullIntoANote(t *testing.T) {
	root := world(t, map[string]string{})
	p := Plan{Actions: []Action{
		{Part: "model", ID: "model-pull", Describe: "ollama pull qwen:7b"},
		{Part: "graph-build", ID: "graph-build", Describe: "loomux graph build"},
	}}
	var ran []string
	run := func(a Action) error {
		ran = append(ran, a.ID)
		if a.ID == "model-pull" {
			return fmt.Errorf("the pull ended without success")
		}
		return nil
	}
	r, err := Apply(root, p, Choice{}, all, run, there, "1", applyTime)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ran, []string{"model-pull", "graph-build"}) || len(r.Failed) != 0 ||
		!slices.Equal(r.Written, []string{"graph-build"}) ||
		!slices.Equal(r.Notes, []string{"model-pull: the pull ended without success; run it by hand: ollama pull qwen:7b"}) {
		t.Errorf("ran = %v, report = %+v", ran, r)
	}
	if !exists(root, installedPath) {
		t.Error("a failed pull kept installed.toml from being written")
	}
}

func TestApplyRecordsASuccessfulPull(t *testing.T) {
	root := world(t, map[string]string{})
	p := Plan{Actions: []Action{{Part: "model", ID: "model-pull", Describe: "ollama pull x"}}}
	r, err := Apply(root, p, Choice{}, all, func(Action) error { return nil }, there, "1", applyTime)
	if err != nil || !slices.Equal(r.Written, []string{"model-pull"}) || len(r.Notes) != 0 {
		t.Errorf("report = %+v, err = %v", r, err)
	}
}
