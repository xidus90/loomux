package convert

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

const sentence = "Der Bericht beschreibt die Abnahme der zweiten Scheibe."

type fakeModel struct {
	mu       sync.Mutex
	scope    string
	sentence string
	describe []string
	place    []string
}

func (m *fakeModel) prompts() (describe, place []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.describe), slices.Clone(m.place)
}

// serveModel switches the model on for the machine and answers from a
// loopback server; scope is what place answers.
func serveModel(t *testing.T, w *world, scope string) *fakeModel {
	t.Helper()
	m := &fakeModel{scope: scope, sentence: sentence}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		prompt, _ := body["prompt"].(string)
		_, format := body["format"]
		m.mu.Lock()
		answer := ""
		switch {
		case strings.HasPrefix(prompt, "<!-- version: ablage-v1\n") && format:
			m.place = append(m.place, prompt)
			answer = fmt.Sprintf(`{"scope": %q, "grund": "Es passt."}`, m.scope)
		case strings.HasPrefix(prompt, "<!-- version: beschreibung-v1 -->\n") && !format:
			m.describe = append(m.describe, prompt)
			answer = m.sentence
		default:
			t.Errorf("a request of no known role (format %v): %.40q", format, prompt)
		}
		m.mu.Unlock()
		_ = json.NewEncoder(rw).Encode(map[string]string{"response": answer})
	}))
	t.Cleanup(server.Close)
	put(t, w.state, "config.toml", fmt.Sprintf("[model]\nenabled = true\nendpoint = %q\n", server.URL))
	return m
}

func roles(describe, place bool) string {
	return fmt.Sprintf("[model]\nroles = { propose = true, place = %t, describe = %t }\n", place, describe)
}

func TestTheDescriptionLandsInTheHeadAskedWithTheBody(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", roles(true, false))
	m := serveModel(t, w, "knowledge")
	put(t, inbox, "video.txt", "[00:00] Hallo zusammen.\n[00:02] Und dann weiter.\n")
	if out, err := w.run(); err != nil || len(out.Skipped) != 0 {
		t.Fatalf("%+v %v", out, err)
	}
	if !strings.Contains(read(t, filepath.Join(inbox, "video.txt.md")), "description: "+sentence+"\n") {
		t.Fatal("no description")
	}
	describe, place := m.prompts()
	if len(describe) != 1 || !strings.HasSuffix(describe[0], "[00:00] Hallo zusammen. Und dann weiter.\n") || len(place) != 0 {
		t.Fatalf("%q %q", describe, place)
	}
}

func TestARefusedDescriptionLeavesTheHeadAndIsAskedAgain(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", roles(true, false))
	m := serveModel(t, w, "knowledge")
	m.sentence = "The report describes the second slice."
	put(t, inbox, "video.txt", "[00:00] Hallo zusammen.\n")
	w.run()
	if strings.Contains(read(t, filepath.Join(inbox, "video.txt.md")), "description") {
		t.Fatal("a refused sentence landed")
	}
	m.mu.Lock()
	m.sentence = sentence
	m.mu.Unlock()
	w.run()
	if describe, _ := m.prompts(); len(describe) != 2 || !strings.Contains(read(t, filepath.Join(inbox, "video.txt.md")), "description: "+sentence) {
		t.Fatalf("asked %d times", len(describe))
	}
}

func TestAStandingSentenceIsNeverAskedFor(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", roles(true, true))
	m := serveModel(t, w, "knowledge")
	put(t, inbox, "video.txt", "[00:00] Hallo zusammen.\n")
	w.run()
	first := read(t, filepath.Join(inbox, "video.txt.md"))
	out, _ := w.run()
	describe, place := m.prompts()
	if len(out.Written)+len(out.Suggested) != 0 || len(describe) != 1 || len(place) != 1 || read(t, filepath.Join(inbox, "video.txt.md")) != first {
		t.Fatalf("%+v %d %d", out, len(describe), len(place))
	}
}

func TestTheModelIsLeftAloneWhileTheMachineSwitchedItOff(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	m := serveModel(t, w, "knowledge")
	settings := filepath.Join(w.state, "config.toml")
	put(t, w.state, "config.toml", strings.Replace(read(t, settings), "enabled = true", "enabled = false", 1))
	put(t, inbox, "video.txt", "[00:00] Hallo.\n")
	out, err := w.run()
	if err != nil || !slices.Equal(out.Written, []string{filepath.Join(inbox, "video.txt.md")}) {
		t.Fatalf("%+v %v", out, err)
	}
	if describe, place := m.prompts(); len(describe)+len(place) != 0 {
		t.Fatal("asked while off")
	}
}

// Off beats on, from below as well.
func TestAnAreaThatSwitchedTheModelOffIsNeverAsked(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "[model]\nenabled = false\n")
	m := serveModel(t, w, "knowledge")
	put(t, inbox, "video.txt", "[00:00] Hallo.\n")
	out, err := w.run()
	if err != nil || !slices.Equal(out.Written, []string{filepath.Join(inbox, "video.txt.md")}) {
		t.Fatalf("%+v %v", out, err)
	}
	if describe, place := m.prompts(); len(describe)+len(place) != 0 {
		t.Fatal("asked although the area said no")
	}
}

func TestAMisconfiguredModelStopsTheRunBeforeAnythingIsWritten(t *testing.T) {
	for block, want := range map[string]string{
		"[model]\nenabled = 5\n": "enabled",
		"[model]\nenabled = true\nendpoint = \"http://192.168.0.10:11434\"\n": "must stay on the loopback",
	} {
		w := newWorld(t)
		inbox := w.inbox("knowledge", "")
		put(t, w.state, "config.toml", block)
		put(t, inbox, "video.txt", "[00:00] Hallo.\n")
		if _, err := w.run(); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v", block, err)
		}
		if entries, _ := os.ReadDir(inbox); len(entries) != 1 {
			t.Errorf("%q: something was written", block)
		}
	}
}

// The proposers are built before the first file: an earlier area that never
// asks the model is not converted either, when a later one would ask off the
// loopback.
func TestAMisconfiguredLaterAreaStopsTheRunBeforeAnEarlierOneIsWritten(t *testing.T) {
	w := newWorld(t)
	first := w.inbox("knowledge", "[model]\nenabled = false\n")
	second := w.inbox("project/x", "")
	put(t, w.state, "config.toml", "[model]\nenabled = true\nendpoint = \"http://192.168.0.10:11434\"\n")
	put(t, first, "video.txt", "[00:00] Hallo.\n")
	put(t, second, "video.txt", "[00:00] Welt.\n")
	if _, err := w.run(); err == nil || !strings.Contains(err.Error(), "must stay on the loopback") {
		t.Fatal(err)
	}
	for _, inbox := range []string{first, second} {
		if entries, _ := os.ReadDir(inbox); len(entries) != 1 {
			t.Errorf("%s: something was written", inbox)
		}
	}
}

// With describe off, ProposerFor never judges the endpoint for it; place is
// where the run stops.
func TestAnEndpointOffTheLoopbackStopsTheRunWhenOnlyPlaceIsOn(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", roles(false, true))
	put(t, w.state, "config.toml", "[model]\nenabled = true\nendpoint = \"http://192.168.0.10:11434\"\n")
	put(t, inbox, "video.txt", "[00:00] Hallo.\n")
	if _, err := w.run(); err == nil || !strings.Contains(err.Error(), "must stay on the loopback") {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(inbox); len(entries) != 1 {
		t.Fatal("something was written")
	}
}

// With place off, describe is where the run stops: no later question about
// the same endpoint stops it instead.
func TestAnEndpointOffTheLoopbackStopsTheRunWhenOnlyDescribeIsOn(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", roles(true, false))
	put(t, w.state, "config.toml", "[model]\nenabled = true\nendpoint = \"http://192.168.0.10:11434\"\n")
	put(t, inbox, "video.txt", "[00:00] Hallo.\n")
	if _, err := w.run(); err == nil || !strings.Contains(err.Error(), "must stay on the loopback") {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(inbox); len(entries) != 1 {
		t.Fatal("something was written")
	}
}

func TestTheSuggestionNamesTheScopeAndMovesNothing(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", roles(false, true))
	elsewhere := w.area("project/ultra-brain", "", false)
	m := serveModel(t, w, "project/ultra-brain")
	put(t, inbox, "video.txt", "[00:00] Hallo zusammen.\n")
	out, _ := w.run()
	if !slices.Equal(out.Suggested, []string{"video.txt.md: belongs in project/ultra-brain, left in the inbox"}) {
		t.Fatalf("%q", out.Suggested)
	}
	if entries, _ := os.ReadDir(elsewhere); len(entries) != 1 { // only .loomux
		t.Fatal("something moved")
	}
	_, place := m.prompts()
	if len(place) != 1 || !strings.Contains(place[0], "\n- knowledge\n- project/ultra-brain\n") || !strings.Contains(place[0], "[00:00] Hallo zusammen.") {
		t.Fatalf("%q", place)
	}
}

func TestAScopeTheRegisterDoesNotKnowIsNoSuggestion(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", roles(false, true))
	serveModel(t, w, "project/erfunden")
	put(t, inbox, "video.txt", "[00:00] Hallo zusammen.\n")
	if out, _ := w.run(); len(out.Suggested) != 0 || len(out.Written) != 1 {
		t.Fatalf("%+v", out)
	}
}

func TestTargetsAreNeverMoreOpenThanTheInbox(t *testing.T) {
	w := newWorld(t)
	open := w.inbox("knowledge", roles(false, true))
	closed := w.inbox("project/x", "[privacy]\nmode = \"local_only\"\n\n"+roles(false, true))
	w.area("project/bare", "", false)
	w.area("project/auto", "[area]\nscope = \"project/auto\"\n\n[privacy]\nmode = \"automatic_cloud\"\n", false)
	w.area("corpus", "", true)
	m := serveModel(t, w, "knowledge")
	put(t, open, "video.txt", "[00:00] Offen aus dem Wissensbereich.\n")
	put(t, closed, "video.txt", "[00:00] Vertraulich aus dem Projekt.\n")
	w.run()
	_, place := m.prompts()
	for _, prompt := range place {
		switch {
		case strings.Contains(prompt, "Vertraulich"):
			if !strings.Contains(prompt, "\n- project/x\n") || strings.Contains(prompt, "- knowledge\n") || strings.Contains(prompt, "- project/bare\n") {
				t.Errorf("the closed inbox was offered %q", prompt)
			}
		default:
			if !strings.Contains(prompt, "- knowledge\n- project/x\n- project/bare\n") || strings.Contains(prompt, "project/auto") || strings.Contains(prompt, "corpus") {
				t.Errorf("the open inbox was offered %q", prompt)
			}
		}
	}
	if len(place) != 2 {
		t.Fatalf("%d placements", len(place))
	}
}

// A file is asked about under the switches of its own area.
func TestEachFileIsAskedUnderItsOwnAreasSwitches(t *testing.T) {
	for _, c := range []struct {
		block    string
		withheld []string
	}{
		{"[model]\nenabled = false\n", []string{"describe", "place"}},
		{roles(false, true), []string{"describe"}},
		{roles(true, false), []string{"place"}},
	} {
		w := newWorld(t)
		open := w.inbox("knowledge", "")
		closed := w.inbox("project/x", c.block)
		m := serveModel(t, w, "knowledge")
		put(t, open, "video.txt", "[00:00] Offen aus dem Wissensbereich.\n")
		put(t, closed, "video.txt", "[00:00] Vertraulich aus dem Projekt.\n")
		w.run()
		describe, place := m.prompts()
		for role, prompts := range map[string][]string{"describe": describe, "place": place} {
			count := func(body string) int {
				return len(slices.DeleteFunc(slices.Clone(prompts), func(p string) bool { return !strings.Contains(p, body) }))
			}
			want := 1
			if slices.Contains(c.withheld, role) {
				want = 0
			}
			if count("Offen") != 1 || count("Vertraulich") != want {
				t.Errorf("%q %s: open %d, closed %d", c.block, role, count("Offen"), count("Vertraulich"))
			}
		}
	}
}
