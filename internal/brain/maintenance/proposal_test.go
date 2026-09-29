package maintenance_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/brain/search"
)

const rejectedNote = "manual review: the local proposer returned no usable proposal (slice-6 spec §3)"
const localOnlyNote = "manual review: this area is local_only, so no skill path is offered (spec 5)"

// ollama answers every request with answer and counts the questions. The
// warm-up a client sends before its first question, the one request that
// caps its answer with num_predict, is answered but not counted.
func ollama(t *testing.T, answer string) (string, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Options map[string]any `json:"options"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, warmUp := body.Options["num_predict"]; !warmUp {
			calls.Add(1)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"response": answer})
	}))
	t.Cleanup(server.Close)
	return server.URL, &calls
}

func modelOn(t *testing.T, stateDir, endpoint string) {
	t.Helper()
	writeUnder(t, stateDir, "config.toml", "[model]\nenabled = true\nendpoint = \""+endpoint+"\"\n")
}

// changedClosedSource is the case world of reconcile_test.go in a closed area.
// Its package's first diff segment D1 carries the added line below.
func changedClosedSource(t *testing.T) (*world, string) {
	area := sourceArea("project/a")
	area.PrivacyMode = "local_only"
	return changedSource(t, area), "+func B() int { return 1 }"
}

func proposalQuoting(line string) string {
	return "## B1 - Die Quelle hat eine Funktion B bekommen.\n\nevidence: D1\n\n```\n" + line + "\n```\n"
}

// cancellingModel is a model that ends the pass while it computes: it cancels
// the pass's context and answers only once the request is gone.
//
// The body is read first: net/http watches for the client going away only
// once the request body is consumed, and a handler that skipped it would wait
// for a disconnect it never hears of.
func cancellingModel(t *testing.T, cancel context.CancelFunc) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		cancel()
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func TestALocalOnlyCaseWithAProposalIsNotManual(t *testing.T) {
	w, added := changedClosedSource(t)
	endpoint, calls := ollama(t, proposalQuoting(added))
	modelOn(t, w.StateDir, endpoint)
	raised := mustReconcile(t, w, w.Now()).Cases[0]
	if raised.Manual || raised.Note != "" || !raised.LocalOnly || raised.PromptVersion != "vorschlag-v4" || calls.Load() != 1 {
		t.Fatalf("%+v, calls %d", raised, calls.Load())
	}
	data, err := os.ReadFile(filepath.Join(caseDirOf(w, raised), "proposal.md"))
	if err != nil || string(data) != proposalQuoting(added) {
		t.Fatalf("%q %v", data, err)
	}
	written, _ := maintenance.ReadCase(filepath.Join(caseDirOf(w, raised), "case.toml"))
	if written.PromptVersion != "vorschlag-v4" || written.Manual {
		t.Fatalf("%+v", written)
	}
}

func TestARefusedProposalMakesAManualCaseWithTheSecondNote(t *testing.T) {
	w, _ := changedClosedSource(t)
	endpoint, calls := ollama(t, proposalQuoting("+erfunden"))
	modelOn(t, w.StateDir, endpoint)
	raised := mustReconcile(t, w, w.Now()).Cases[0]
	if !raised.Manual || raised.Note != rejectedNote || raised.PromptVersion != "" || calls.Load() != 1 {
		t.Fatalf("%+v", raised)
	}
	if _, err := os.Stat(filepath.Join(caseDirOf(w, raised), "proposal.md")); err == nil {
		t.Fatal("a refused proposal was written")
	}
}

// secondPassAsksNothing runs a pass a day later over the unchanged sources and
// holds that the model was not asked again and the case's files stayed as the
// first pass wrote them.
func secondPassAsksNothing(t *testing.T, w *world, first maintenance.Case, calls *atomic.Int32) {
	t.Helper()
	dir := caseDirOf(w, first)
	before := map[string][]byte{}
	for _, name := range []string{"case.toml", "proposal.md"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			before[name] = data
		}
	}
	asked := calls.Load()
	second := mustReconcile(t, w, w.Now().Add(24*time.Hour))
	if len(second.Cases) != 1 || second.Cases[0].ID != first.ID || calls.Load() != asked {
		t.Fatalf("second pass: %+v, calls %d after %d", second.Cases, calls.Load(), asked)
	}
	for _, name := range []string{"case.toml", "proposal.md"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if want, had := before[name]; had != (err == nil) || string(data) != string(want) {
			t.Fatalf("%s changed: %q, was %q", name, data, want)
		}
	}
}

// A standing case is not asked again: the answer it carries stays its answer.
func TestAStandingCaseIsNotAskedAgain(t *testing.T) {
	w, added := changedClosedSource(t)
	endpoint, calls := ollama(t, proposalQuoting(added))
	modelOn(t, w.StateDir, endpoint)
	first := mustReconcile(t, w, w.Now()).Cases[0]
	if first.Manual || calls.Load() != 1 {
		t.Fatalf("%+v, calls %d", first, calls.Load())
	}
	secondPassAsksNothing(t, w, first, calls)
}

// Nor is a case whose first answer was refused: the manual case stands.
func TestAStandingRefusedCaseIsNotAskedAgain(t *testing.T) {
	w, _ := changedClosedSource(t)
	endpoint, calls := ollama(t, proposalQuoting("+erfunden"))
	modelOn(t, w.StateDir, endpoint)
	first := mustReconcile(t, w, w.Now()).Cases[0]
	if !first.Manual || first.Note != rejectedNote || calls.Load() != 1 {
		t.Fatalf("%+v, calls %d", first, calls.Load())
	}
	secondPassAsksNothing(t, w, first, calls)
}

func TestAnUnreachableModelIsARefusedProposalToo(t *testing.T) {
	w, _ := changedClosedSource(t)
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	modelOn(t, w.StateDir, server.URL)
	if raised := mustReconcile(t, w, w.Now()).Cases[0]; !raised.Manual || raised.Note != rejectedNote {
		t.Fatalf("%+v", raised)
	}
}

func TestWithTheModelOffTheOldNoteStays(t *testing.T) {
	w, _ := changedClosedSource(t)
	if raised := mustReconcile(t, w, w.Now()).Cases[0]; !raised.Manual || raised.Note != localOnlyNote {
		t.Fatalf("%+v", raised)
	}
}

// An area's own `enabled = false` beats the global switch.
func TestAnAreaThatSwitchesTheModelOffIsNotAsked(t *testing.T) {
	area := sourceArea("project/a")
	area.PrivacyMode = "local_only"
	area.Declaration = "[model]\nenabled = false\n"
	w := changedSource(t, area)
	endpoint, calls := ollama(t, "x")
	modelOn(t, w.StateDir, endpoint)
	if raised := mustReconcile(t, w, w.Now()).Cases[0]; raised.Note != localOnlyNote || calls.Load() != 0 {
		t.Fatalf("%+v, calls %d", raised, calls.Load())
	}
}

func TestAnOpenAreaIsNeverAsked(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	endpoint, calls := ollama(t, "x")
	modelOn(t, w.StateDir, endpoint)
	if raised := mustReconcile(t, w, w.Now()).Cases[0]; raised.Manual || raised.Note != "" || calls.Load() != 0 {
		t.Fatalf("%+v, calls %d", raised, calls.Load())
	}
}

// A broken [model] must not stop a run that would not have used it.
func TestAnOpenVaultNeverReadsTheModelBlock(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	writeUnder(t, w.StateDir, "config.toml", "model = 5\n")
	mustReconcile(t, w, w.Now())
}

func TestAMisconfiguredModelStopsTheRunBeforeAnyCase(t *testing.T) {
	for name, text := range map[string]string{
		"broken block":     "model = 5\n",
		"off the loopback": "[model]\nenabled = true\nendpoint = \"http://192.0.2.1:11434\"\n",
	} {
		w, _ := changedClosedSource(t)
		writeUnder(t, w.StateDir, "config.toml", text)
		if _, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now()); err == nil || !strings.Contains(err.Error(), "[model]") {
			t.Errorf("%s: %v", name, err)
		}
		root, _ := maintenance.ReviewRoot(w.Areas, w.Lookup())
		if entries, _ := os.ReadDir(filepath.Join(root, search.CollectionName("project/a"))); len(entries) != 0 {
			t.Errorf("%s: a case was written: %v", name, entries)
		}
	}
}

// A pass that ends while the model computes writes nothing.
func TestACancelledPassWritesNoCaseWhileTheModelComputes(t *testing.T) {
	w, _ := changedClosedSource(t)
	ctx, cancel := context.WithCancel(context.Background())
	modelOn(t, w.StateDir, cancellingModel(t, cancel))
	_, err := maintenance.ReconcileContext(ctx, w.Areas, w.Lookup(), w.Now())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
	root, _ := maintenance.ReviewRoot(w.Areas, w.Lookup())
	if entries, _ := os.ReadDir(filepath.Join(root, search.CollectionName("project/a"))); len(entries) != 0 {
		t.Fatalf("a case was written: %v", entries)
	}
}

// The same cancellation over a standing case on an older source state: it
// must stay as it was, proposal and all.
func TestACancelledPassKeepsTheStandingCase(t *testing.T) {
	w, _ := changedClosedSource(t)
	standing := mustReconcile(t, w, w.Now()).Cases[0]
	dir := caseDirOf(w, standing)
	if err := os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("kept\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w.Change(t, "project/a", "src/a.go", "package a\n\nfunc C() int { return 2 }\n")
	ctx, cancel := context.WithCancel(context.Background())
	modelOn(t, w.StateDir, cancellingModel(t, cancel))
	if _, err := maintenance.ReconcileContext(ctx, w.Areas, w.Lookup(), w.Now().Add(24*time.Hour)); !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "proposal.md")); err != nil || string(data) != "kept\n" {
		t.Fatalf("the standing case was touched: %q %v", data, err)
	}
}
