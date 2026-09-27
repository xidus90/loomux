package journal_test

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/flow/journal"
)

// encoding/json sorts map keys but writes struct fields in declaration order;
// a hash that turned on that order would redo finished work on resume.
func TestCanonicalSortsKeysAtEveryDepthAndKeepsHTML(t *testing.T) {
	value := struct {
		Zeta  int            `json:"zeta"`
		Alpha map[string]any `json:"alpha"`
	}{Zeta: 2, Alpha: map[string]any{"y": []any{1, "<&>"}, "x": nil}}

	got, err := journal.Canonical(value)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"alpha":{"x":null,"y":[1,"<&>"]},"zeta":2}`; string(got) != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestCanonicalKeepsLargeNumbersExact(t *testing.T) {
	got, err := journal.Canonical(map[string]any{"big": uint64(18446744073709551615), "small": 0.1})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"big":18446744073709551615,"small":0.1}`; string(got) != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestCanonicalRefusesWhatJSONCannotSay(t *testing.T) {
	_, err := journal.Canonical(math.Inf(1))
	if err == nil || !strings.Contains(err.Error(), "cannot serialize") {
		t.Fatalf("got %v", err)
	}
}

func TestInputHashIsSha256OfTheCanonicalPayload(t *testing.T) {
	got, err := journal.InputHash("draft", map[string]any{"b": 2, "a": 1})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(`{"data":{"a":1,"b":2},"node":"draft"}`))
	if want := hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	other, err := journal.InputHash("review", map[string]any{"a": 1, "b": 2})
	if err != nil {
		t.Fatal(err)
	}
	if other == got {
		t.Fatal("the node name is part of the hash")
	}
}

func TestInputHashRefusesUnserializableData(t *testing.T) {
	if _, err := journal.InputHash("n", map[string]any{"f": func() {}}); err == nil {
		t.Fatal("a func has no JSON form, so it has no hash")
	}
}

func TestDefinitionHashMovesWithEveryPart(t *testing.T) {
	node := map[string]any{"name": "draft"}
	text := []byte("Write it.")
	tools := []string{"Read"}
	base, err := journal.DefinitionHash(node, text, "claude:cli-default", "high", tools)
	if err != nil {
		t.Fatal(err)
	}
	again, err := journal.DefinitionHash(node, text, "claude:cli-default", "high", tools)
	if err != nil {
		t.Fatal(err)
	}
	if again != base {
		t.Fatal("the same definition must hash the same")
	}
	variants := map[string]func() (string, error){
		"node": func() (string, error) {
			return journal.DefinitionHash(map[string]any{"name": "review"}, text, "claude:cli-default", "high", tools)
		},
		"text": func() (string, error) {
			return journal.DefinitionHash(node, []byte("Write it well."), "claude:cli-default", "high", tools)
		},
		"model": func() (string, error) {
			return journal.DefinitionHash(node, text, "claude:claude-opus-5", "high", tools)
		},
		"effort": func() (string, error) {
			return journal.DefinitionHash(node, text, "claude:cli-default", "low", tools)
		},
		"tools": func() (string, error) {
			return journal.DefinitionHash(node, text, "claude:cli-default", "high", []string{"Edit", "Read"})
		},
	}
	for part, hash := range variants {
		got, err := hash()
		if err != nil {
			t.Fatal(err)
		}
		if got == base {
			t.Errorf("changing the %s left the hash unchanged", part)
		}
	}
}

func TestDefinitionHashRefusesAnUnserializableNode(t *testing.T) {
	if _, err := journal.DefinitionHash(map[string]any{"x": math.NaN()}, nil, "", "", nil); err == nil {
		t.Fatal("NaN has no JSON form, so it has no hash")
	}
}
