package model

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

const placeAnswer = `{"scope": "project/ultra-brain", "grund": "Es geht um den Indexer."}`

func place(t *testing.T, answer string, scopes ...string) (string, bool) {
	t.Helper()
	return allRoles(t, answer, nil).Place(context.Background(), "text", scopes)
}

func TestThePlacePromptIsTheReferencesByteForByte(t *testing.T) {
	sum := sha256.Sum256([]byte(placePrompt))
	if hex.EncodeToString(sum[:]) != "efb2ed013e30e1d38ae3a682d53288c4c48a6b3a236937a9d9d7f83f13ec5148" {
		t.Fatal("ablage-v1.md is not the reference's")
	}
	if strings.Count(placePrompt, "{") != 2 || !strings.Contains(placePrompt, "{scopes}") || !strings.Contains(placePrompt, "{text}") {
		t.Fatal("the prompt carries braces str.format would read")
	}
}

func TestAScopeFromTheRegisterComesBack(t *testing.T) {
	if got, ok := place(t, placeAnswer, "project/ultra-brain", "space"); !ok || got != "project/ultra-brain" {
		t.Fatalf("%q %v", got, ok)
	}
}

func TestPlaceDropsWhatIsNoKnownScope(t *testing.T) {
	for _, answer := range []string{
		`{"scope": "project/erfunden", "grund": "..."}`,
		"project/ultra-brain",
		`{"grund": "weiss nicht"}`,
		`["project/ultra-brain"]`,
		`{"scope": 5, "grund": "x"}`,
	} {
		if got, ok := place(t, answer, "project/ultra-brain"); ok {
			t.Errorf("%q passed as %q", answer, got)
		}
	}
}

// Python's json.loads reads NaN, Infinity and a number past float64, and
// nests as deep as the stack allows; the reference places such an answer,
// encoding/json refuses it, and loomux places nothing. Only an endpoint that
// ignores the format schema sends one; a row of the parity list. The object
// and 9 999 lists inside it are encoding/json's 10 000 levels, and still read.
func TestPlaceRefusesWhatOnlyPythonsJSONReads(t *testing.T) {
	nested := func(n int) string { return strings.Repeat("[", n) + strings.Repeat("]", n) }
	answer := func(grund string) string { return `{"scope": "project/ultra-brain", "grund": ` + grund + `}` }
	if got, ok := place(t, answer(nested(9999)), "project/ultra-brain"); !ok || got != "project/ultra-brain" {
		t.Fatalf("10 000 levels: %q %v", got, ok)
	}
	for _, grund := range []string{"NaN", "Infinity", "-Infinity", "1e400", nested(10000)} {
		if got, ok := place(t, answer(grund), "project/ultra-brain"); ok {
			t.Errorf("%.12s passed as %q", grund, got)
		}
	}
}

func TestTheSchemaGoesOutAsTheFormat(t *testing.T) {
	var bodies []map[string]any
	allRoles(t, placeAnswer, &bodies).Place(context.Background(), "text", []string{"project/ultra-brain"})
	want := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scope": map[string]any{"type": "string"},
			"grund": map[string]any{"type": "string"},
		},
		"required": []any{"scope", "grund"},
	}
	if !reflect.DeepEqual(bodies[0]["format"], want) {
		t.Fatalf("format %v", bodies[0]["format"])
	}
}

func TestTheScopesAndTheHeadReachThePlacePrompt(t *testing.T) {
	var bodies []map[string]any
	allRoles(t, placeAnswer, &bodies).Place(context.Background(), strings.Repeat("A", 1799)+"B"+strings.Repeat("C", 500), []string{"project/ultra-brain", "space"})
	prompt := bodies[0]["prompt"].(string)
	if !strings.Contains(prompt, "\n- project/ultra-brain\n- space\n") {
		t.Fatal("the scopes are not in the rig's list form")
	}
	if !strings.Contains(prompt, strings.Repeat("A", 1799)+"B") || strings.Contains(prompt, "C") {
		t.Fatal("the cut is not the first 1800 characters")
	}
}

// A text that names a placeholder is not read as one: str.format fills each
// field once, and so does the replacer.
func TestATextNamingAPlaceholderStaysText(t *testing.T) {
	var bodies []map[string]any
	allRoles(t, placeAnswer, &bodies).Place(context.Background(), "über {scopes} und {text}", []string{"a"})
	if !strings.Contains(bodies[0]["prompt"].(string), "über {scopes} und {text}") {
		t.Fatal("the text was filled in again")
	}
}

func TestPlaceWithoutAnAnswerIsNone(t *testing.T) {
	settings := config.ModelSettings{Enabled: true, Endpoint: "http://127.0.0.1:1", Name: "m", Roles: map[string]bool{"place": true}}
	p, _ := ProposerFor(settings, &config.Manifest{}, "place")
	if got, ok := p.Place(context.Background(), "t", []string{"a"}); ok {
		t.Fatalf("%q", got)
	}
}
