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

	"github.com/xidus90/loomux/internal/brain/evidence"
	"github.com/xidus90/loomux/internal/config"
)

const goodProposal = "## B1 - Die Quelle traegt jetzt einen neuen Stand.\n\nevidence: D1\n\n```\n+neu\n```\n"
const inventedProposal = "## B1 - Die Quelle traegt jetzt einen erfundenen Stand.\n\nevidence: D1\n\n```\n+erfunden\n```\n"

var segments = []evidence.Segment{{Number: "D1", Kind: "D", Label: "diff", Body: "@@ -1 +1 @@\n-alt\n+neu\n"}}

func TestThePromptIsTheReferencesByteForByte(t *testing.T) {
	sum := sha256.Sum256([]byte(proposePrompt))
	if hex.EncodeToString(sum[:]) != "05e2ebbbf448a0b5ce8562719b9cd564dfd1f62dac1b8f199fed02aa0c19b1ab" {
		t.Fatal("vorschlag-v4.md is not the reference's")
	}
	// Replacing {paket} is str.format only while no other brace is there.
	if strings.Count(proposePrompt, "{") != 1 || strings.Count(proposePrompt, "}") != 1 || !strings.Contains(proposePrompt, "{paket}") {
		t.Fatal("the prompt carries braces str.format would read")
	}
}

func answering(t *testing.T, answer string, prompts *[]string) config.ModelSettings {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		if prompts != nil {
			*prompts = append(*prompts, got["prompt"].(string))
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"response": answer})
	}))
	t.Cleanup(server.Close)
	return config.ModelSettings{Enabled: true, Endpoint: server.URL, Name: "m", Roles: map[string]bool{"propose": true}}
}

func TestProposeKeepsAProposalTheEvidencePasses(t *testing.T) {
	var prompts []string
	p, err := ProposerFor(answering(t, goodProposal, &prompts), &config.Manifest{}, "propose")
	if err != nil || p == nil {
		t.Fatal(p, err)
	}
	text, ok := p.Propose(context.Background(), "PAKET", segments)
	if !ok || text != goodProposal {
		t.Fatalf("%q %v", text, ok)
	}
	if len(prompts) != 1 || prompts[0] != strings.ReplaceAll(proposePrompt, "{paket}", "PAKET") {
		t.Fatal("the prompt is not the file with the package in it")
	}
}

func TestProposeDropsWhatTheEvidenceRefuses(t *testing.T) {
	for name, answer := range map[string]string{
		"invented":   inventedProposal,
		"one of two": goodProposal + "\n" + strings.Replace(inventedProposal, "B1", "B2", 1),
		"no claims":  "Ich weiss es nicht.",
		"empty":      "",
	} {
		p, _ := ProposerFor(answering(t, answer, nil), &config.Manifest{}, "propose")
		if text, ok := p.Propose(context.Background(), "P", segments); ok || text != "" {
			t.Errorf("%s: %q %v", name, text, ok)
		}
	}
}

func TestProposeWithoutAnAnswerIsNone(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	s := config.ModelSettings{Enabled: true, Endpoint: server.URL, Roles: map[string]bool{"propose": true}}
	server.Close()
	p, _ := ProposerFor(s, &config.Manifest{}, "propose")
	if _, ok := p.Propose(context.Background(), "P", segments); ok {
		t.Fatal("a closed port proposed")
	}
}

func TestTheGateBuildsNoClientForARoleThatIsOff(t *testing.T) {
	off := false
	foreign := config.ModelSettings{Enabled: true, Endpoint: "http://192.0.2.1", Roles: map[string]bool{"propose": true}}
	for name, c := range map[string]struct {
		s config.ModelSettings
		m *config.Manifest
	}{
		"global off": {config.ModelSettings{Endpoint: "http://192.0.2.1", Roles: foreign.Roles}, &config.Manifest{}},
		"area off":   {foreign, &config.Manifest{ModelEnabled: &off}},
		"role off":   {foreign, &config.Manifest{ModelRoles: map[string]bool{"place": true}}},
	} {
		// A foreign endpoint that passes unnoticed proves no client was built.
		if p, err := ProposerFor(c.s, c.m, "propose"); p != nil || err != nil {
			t.Errorf("%s: %v %v", name, p, err)
		}
	}
	if _, err := ProposerFor(foreign, &config.Manifest{}, "propose"); err == nil {
		t.Fatal("a foreign endpoint was accepted once the role was on")
	}
}
