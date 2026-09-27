package flow_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/flow"
)

func TestCapLimitAddsToItsParameter(t *testing.T) {
	limit, err := flow.Cap{Param: "max_rounds", Add: 1}.Limit(flow.Params{"max_rounds": 5})
	if err != nil {
		t.Fatal(err)
	}
	if limit != 6 {
		t.Fatalf("limit = %d, want 6", limit)
	}
}

func TestCapLimitWithoutAParameterIsItsNumber(t *testing.T) {
	limit, err := flow.Cap{Add: 3}.Limit(nil)
	if err != nil {
		t.Fatal(err)
	}
	if limit != 3 {
		t.Fatalf("limit = %d, want 3", limit)
	}
}

func TestStringKeyReadsOnlyAString(t *testing.T) {
	node := flow.Node{Keys: map[string]any{"instruction": "draft.md", "code": int64(4)}}
	if node.StringKey("instruction") != "draft.md" {
		t.Fatalf("got %q", node.StringKey("instruction"))
	}
	if node.StringKey("code") != "" || node.StringKey("nothing") != "" {
		t.Fatal("a key that is not a string reads as empty")
	}
}

func TestUnknownKeysAreSorted(t *testing.T) {
	node := flow.Node{Keys: map[string]any{"question": "q.md", "zzz": 1, "answer": "a", "aaa": 1}}
	got := node.UnknownKeys("question", "answer")
	if len(got) != 2 || got[0] != "aaa" || got[1] != "zzz" {
		t.Fatalf("got %v", got)
	}
	if node.UnknownKeys("question", "answer", "aaa", "zzz") != nil {
		t.Fatal("want nil when every key is known")
	}
}

// A finding for an unknown key shows every key a node of the kind may hold,
// the shared ones and the block's, so a typo meets the word it should have been.
func TestKeyFindingsNameTheKnownKeys(t *testing.T) {
	node := flow.Node{Keys: map[string]any{"question": "q.md", "zzz": 1, "aaa": 1}}
	got := node.KeyFindings("question", "answer")
	const known = "; known keys: answer, kind, max_visits, name, question, role"
	if len(got) != 2 || got[0] != `unknown key "aaa"`+known || got[1] != `unknown key "zzz"`+known {
		t.Fatalf("got %q", got)
	}
	if node.KeyFindings("question", "aaa", "zzz") != nil {
		t.Fatal("want nil when every key is known")
	}
}

func TestSharedKeysAreACopy(t *testing.T) {
	keys := flow.SharedKeys()
	keys[0] = "changed"
	if flow.SharedKeys()[0] != "kind" {
		t.Fatal("a caller changed the shared keys")
	}
}

// The loader checks both before a run; a hand-built graph does not, and the
// runner is the last place where an unusable ceiling can still be named.
func TestCapLimitRefusesAParameterItCannotUse(t *testing.T) {
	if _, err := (flow.Cap{Param: "rounds"}).Limit(nil); err == nil ||
		err.Error() != `max_visits names parameter "rounds", which the run does not have` {
		t.Fatalf("err = %v", err)
	}
	if _, err := (flow.Cap{Param: "rounds"}).Limit(flow.Params{"rounds": "five"}); err == nil ||
		err.Error() != `max_visits names parameter "rounds", which is not an int` {
		t.Fatalf("err = %v", err)
	}
}
