package model

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

const goodSentence = "Der Bericht beschreibt die Abnahme der zweiten Scheibe."

// allRoles is answering with every role on, and every request body kept.
func allRoles(t *testing.T, answer string, bodies *[]map[string]any) *Proposer {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		if bodies != nil {
			*bodies = append(*bodies, got)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"response": answer})
	}))
	t.Cleanup(server.Close)
	settings := config.ModelSettings{Enabled: true, Endpoint: server.URL, Name: "m",
		Roles: map[string]bool{"propose": true, "place": true, "describe": true}}
	p, err := ProposerFor(settings, &config.Manifest{}, "describe")
	if err != nil || p == nil {
		t.Fatal(p, err)
	}
	return p
}

func describe(t *testing.T, answer, text string) (string, bool) {
	t.Helper()
	return allRoles(t, answer, nil).Describe(context.Background(), text)
}

func TestTheDescribePromptIsTheReferencesByteForByte(t *testing.T) {
	sum := sha256.Sum256([]byte(describePrompt))
	if hex.EncodeToString(sum[:]) != "a2f11bf4b41592a0ee798a3ebb0335740aa90b5d3c0a1d7642f3e298ca919125" {
		t.Fatal("beschreibung-v1.md is not the reference's")
	}
	if strings.Count(describePrompt, "{") != 1 || !strings.Contains(describePrompt, "{text}") {
		t.Fatal("the prompt carries braces str.format would read")
	}
}

func TestAGoodSentenceComesBackTightened(t *testing.T) {
	for _, answer := range []string{goodSentence, "  " + goodSentence + "\n\n"} {
		if got, ok := describe(t, answer, "langer text"); !ok || got != goodSentence {
			t.Errorf("%q: %q %v", answer, got, ok)
		}
	}
}

func TestDescribeRefusesWhatBreaksARule(t *testing.T) {
	for _, answer := range []string{
		"Ein Satz. Und noch einer dazu.",
		"Der Bericht " + strings.Repeat("und der Anhang ", 10) + "sind da.",
		"The report describes the second slice.",
		"Das Pro-jekt ist in acht Scheiben zerlegt.",
		"Der Bericht: die Abnahme der zweiten Scheibe.",
		"Der Bericht beschreibt\ndie Abnahme der Scheibe.",
		"Der Bericht ist in #1 der Reihe.",
		"[Der Bericht] beschreibt die Abnahme der zweiten Scheibe.",
	} {
		if got, ok := describe(t, answer, "t"); ok {
			t.Errorf("%q passed as %q", answer, got)
		}
	}
}

// Twenty-two words is the limit the rig judged against, and it still passes.
func TestTwentyTwoWordsPassAndTwentyThreeDoNot(t *testing.T) {
	limit := "Der Bericht " + strings.Repeat("und der Anhang ", 6) + "sind da."
	if got, ok := describe(t, limit, "t"); !ok || got != limit {
		t.Fatalf("%d words: %q %v", WordCount(limit), got, ok)
	}
	over := "Der Bericht " + strings.Repeat("und der Anhang ", 6) + "sind alle da."
	if got, ok := describe(t, over, "t"); ok {
		t.Fatalf("%d words passed as %q", WordCount(over), got)
	}
}

func TestAQuoteInsideTheSentenceStands(t *testing.T) {
	answer := `Der Bericht nennt die Regel "Aus schlägt An" und ihre Grenzen.`
	if got, ok := describe(t, answer, "t"); !ok || got != answer {
		t.Fatalf("%q %v", got, ok)
	}
}

func TestOnlyTheMeasuredHeadReachesTheDescribePrompt(t *testing.T) {
	var bodies []map[string]any
	allRoles(t, goodSentence, &bodies).Describe(context.Background(), strings.Repeat("Ä", 1799)+"B"+strings.Repeat("C", 500))
	prompt := bodies[0]["prompt"].(string)
	if !strings.Contains(prompt, strings.Repeat("Ä", 1799)+"B") || strings.Contains(prompt, "C") {
		t.Fatal("the cut is not the first 1800 characters")
	}
	if _, present := bodies[0]["format"]; present {
		t.Fatal("describe sent a schema")
	}
}

func TestDescribeWithoutAnAnswerIsNone(t *testing.T) {
	settings := config.ModelSettings{Enabled: true, Endpoint: "http://127.0.0.1:1", Name: "m", Roles: map[string]bool{"describe": true}}
	p, _ := ProposerFor(settings, &config.Manifest{}, "describe")
	if got, ok := p.Describe(context.Background(), "t"); ok {
		t.Fatalf("%q", got)
	}
}
