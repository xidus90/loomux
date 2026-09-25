package hostfile

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
)

var claude = hosts.HostClaude

// commands decodes a merged file and returns every command under one event
// of the container key, in file order.
func commands(t *testing.T, merged []byte, container, event string) []string {
	t.Helper()
	var root map[string]any
	if err := json.Unmarshal(merged, &root); err != nil {
		t.Fatalf("result is not JSON: %v\n%s", err, merged)
	}
	list, _ := root[container].(map[string]any)[event].([]any)
	var out []string
	for _, block := range list {
		out = append(out, firstCommand(block.(map[string]any)))
	}
	return out
}

func TestMergeLeavesLoomuxsOwnSettingsByteForByte(t *testing.T) {
	existing, err := os.ReadFile("testdata/loomux-settings.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Merge(claude, existing, Entries(claude, Canonical))
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(got.Added) != 0 || len(got.Kept) != 6 || len(got.Foreign) != 0 {
		t.Fatalf("added %v, kept %v, foreign %v; want six kept", got.Added, got.Kept, got.Foreign)
	}
	if !bytes.Equal(got.Merged, existing) {
		t.Fatalf("merged differs from the input:\n%s", got.Merged)
	}
}

func TestMergeAddsBesideOldEntriesAndNamesThem(t *testing.T) {
	existing := []byte(`{"hooks":{"PreToolUse":[
		{"matcher":"Write|Edit|NotebookEdit|Bash|PowerShell",
		 "hooks":[{"type":"command","command":"ulguard --root \"${CLAUDE_PROJECT_DIR}\"","timeout":10}]},
		{"matcher":"Write|Edit|MultiEdit|NotebookEdit",
		 "hooks":[{"type":"command","command":"brain guard","timeout":15}]}
	]}}`)
	got, err := Merge(claude, existing, Entries(claude, Canonical))
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	wantAdded := []string{
		"SessionStart/",
		"PreToolUse/Write|Edit|MultiEdit|NotebookEdit|Bash|PowerShell",
		"PostToolUse/Write|Edit|MultiEdit|NotebookEdit",
		"Stop/",
		"SubagentStart/",
		"SubagentStop/",
	}
	if !reflect.DeepEqual(got.Added, wantAdded) || len(got.Kept) != 0 || len(got.Foreign) != 0 {
		t.Fatalf("added %v, kept %v, foreign %v", got.Added, got.Kept, got.Foreign)
	}
	pre := commands(t, got.Merged, "hooks", "PreToolUse")
	wantPre := []string{
		`ulguard --root "${CLAUDE_PROJECT_DIR}"`,
		"brain guard",
		Canonical + ` hook pre-tool-use --host claude --root "${CLAUDE_PROJECT_DIR}"`,
	}
	if !reflect.DeepEqual(pre, wantPre) {
		t.Fatalf("PreToolUse = %q, want %q", pre, wantPre)
	}
	again, err := Merge(claude, got.Merged, Entries(claude, Canonical))
	if err != nil {
		t.Fatalf("second Merge: %v", err)
	}
	if len(again.Added) != 0 || len(again.Kept) != 6 || !bytes.Equal(again.Merged, got.Merged) {
		t.Fatalf("second merge added %v, kept %v", again.Added, again.Kept)
	}
}

// A hook of the project on our event and matcher is reported and left where
// it is; ours goes beside it, and on the next run it is kept and the
// project's hook is reported again.
func TestMergeReportsAForeignHookOnTheSameSlotAndAddsBesideIt(t *testing.T) {
	existing := []byte(`{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"theirs"}]}]}}`)
	wanted := []Entry{{Event: "Stop", Command: "loomux hook stop"}}
	got, err := Merge(claude, existing, wanted)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if !reflect.DeepEqual(got.Added, []string{"Stop/"}) || !reflect.DeepEqual(got.Foreign, []string{"Stop/"}) {
		t.Fatalf("added %v, foreign %v", got.Added, got.Foreign)
	}
	if stop := commands(t, got.Merged, "hooks", "Stop"); !reflect.DeepEqual(stop, []string{"theirs", "loomux hook stop"}) {
		t.Fatalf("Stop = %q", stop)
	}
	again, err := Merge(claude, got.Merged, wanted)
	if err != nil {
		t.Fatalf("second Merge: %v", err)
	}
	if !reflect.DeepEqual(again.Kept, []string{"Stop/"}) || !reflect.DeepEqual(again.Foreign, []string{"Stop/"}) {
		t.Fatalf("kept %v, foreign %v", again.Kept, again.Foreign)
	}
}

// An own block with another command is still ours: nothing is rewritten.
func TestMergeKeepsAnOwnBlockWithAnotherCommand(t *testing.T) {
	existing := []byte(`{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"loomux hook stop --old"}]}]}}`)
	got, err := Merge(claude, existing, []Entry{{Event: "Stop", Command: "loomux hook stop"}})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if !reflect.DeepEqual(got.Kept, []string{"Stop/"}) || !bytes.Equal(got.Merged, existing) {
		t.Fatalf("kept %v, merged %s", got.Kept, got.Merged)
	}
}

func TestMergeRefusesWhatItCannotRead(t *testing.T) {
	for _, input := range []string{`not json`, `[]`, `{"hooks": []}`, `{"hooks": {"Stop": {}}}`} {
		_, err := Merge(claude, []byte(input), Entries(claude, Canonical))
		if err == nil {
			t.Errorf("Merge(%s): want an error, not a repair", input)
			continue
		}
		if !strings.Contains(err.Error(), ".claude/settings.json") {
			t.Errorf("Merge(%s): err = %v, want the file named", input, err)
		}
	}
}

func TestMergeRefusesAHostWithoutAHookFile(t *testing.T) {
	if _, err := Merge(hosts.HostCodex, nil, nil); err == nil {
		t.Fatal("want an error for a host without a hook file")
	}
}

func TestMergeKeepsForeignTopLevelKeysInTheirOrder(t *testing.T) {
	existing := []byte(`{"permissions":{"allow":["Bash"]},"env":{"A":"1"},"hooks":{},"model":"opus"}`)
	got, err := Merge(claude, existing, []Entry{{Event: "Stop", Command: "loomux hook stop"}})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	merged := string(got.Merged)
	order := []string{`"permissions"`, `"env"`, `"hooks"`, `"model"`}
	last := -1
	for _, key := range order {
		at := strings.Index(merged, key)
		if at <= last {
			t.Fatalf("key %s out of order:\n%s", key, merged)
		}
		last = at
	}
}

func TestMergeWritesTheAntigravityGroup(t *testing.T) {
	existing := []byte(`{"wiki-guard":{"PreToolUse":[{"matcher":"write_to_file","hooks":[{"type":"command","command":"brain guard"}]}]}}`)
	wanted := entries(hosts.HostAntigravity, Canonical, true)
	got, err := Merge(hosts.HostAntigravity, existing, wanted)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(got.Added) != 2 {
		t.Fatalf("added %v", got.Added)
	}
	if pre := commands(t, got.Merged, "loomux", "PreToolUse"); len(pre) != 1 || !Owned(pre[0]) {
		t.Fatalf("loomux PreToolUse = %q", pre)
	}
	if guard := commands(t, got.Merged, "wiki-guard", "PreToolUse"); !reflect.DeepEqual(guard, []string{"brain guard"}) {
		t.Fatalf("wiki-guard PreToolUse = %q", guard)
	}
	if _, err := Merge(hosts.HostAntigravity, []byte(`{"loomux":[]}`), wanted); err == nil ||
		!strings.Contains(err.Error(), ".agents/hooks.json") {
		t.Fatalf("err = %v, want the file named", err)
	}
}

func TestBinaryOfFollowsTheCheckout(t *testing.T) {
	own, err := os.ReadFile("testdata/loomux-settings.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := BinaryOf(claude, own); got != Checkout {
		t.Errorf("BinaryOf(loomux) = %s, want Checkout", got)
	}
	canonical, err := Merge(claude, nil, Entries(claude, Canonical))
	if err != nil {
		t.Fatal(err)
	}
	foreignCheckout := []byte(`{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"\"${CLAUDE_PROJECT_DIR}/bin/other.exe\""}]}]}}`)
	for name, input := range map[string][]byte{
		"empty":            nil,
		"canonical":        canonical.Merged,
		"broken":           []byte(`not json`),
		"foreign checkout": foreignCheckout,
		"no container":     []byte(`{"hooks":[]}`),
	} {
		if got := BinaryOf(claude, input); got != Canonical {
			t.Errorf("BinaryOf(%s) = %s, want Canonical", name, got)
		}
	}
}

func TestBinaryOfReadsTheAntigravityGroup(t *testing.T) {
	existing := []byte(`{"loomux":{"PreToolUse":[{"hooks":[{"type":"command","command":"\"${CLAUDE_PROJECT_DIR}/bin/loomux.exe\" hook pre-tool-use"}]}]}}`)
	if got := BinaryOf(hosts.HostAntigravity, existing); got != Checkout {
		t.Fatalf("BinaryOf(antigravity) = %s, want Checkout", got)
	}
}

// Carried over from ulinit's settings merge, where it guarded the same
// round trip.

func TestAnEmptyFileBecomesOneWithJustOurHook(t *testing.T) {
	got, err := Merge(claude, nil, []Entry{{Event: "Stop", Command: "ours", Timeout: 60}})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if !strings.Contains(string(got.Merged), `"timeout": 60`) {
		t.Fatalf("merged = %s, want the timeout carried", got.Merged)
	}
}

// TestATimeoutOfZeroIsLeftOut: absent means "the default", and writing a zero
// would mean "no time at all".
func TestATimeoutOfZeroIsLeftOut(t *testing.T) {
	got, err := Merge(claude, nil, []Entry{{Event: "Stop", Command: "ours"}})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if strings.Contains(string(got.Merged), "timeout") {
		t.Fatalf("merged = %s, want no timeout key", got.Merged)
	}
}

func TestANullFileIsTreatedAsAnEmptyOne(t *testing.T) {
	got, err := Merge(claude, []byte("null"), []Entry{{Event: "Stop", Command: "ours"}})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if !strings.Contains(string(got.Merged), "ours") {
		t.Fatalf("merged = %s, want the entry added", got.Merged)
	}
}

// TestAMatcherlessEntryDoesNotClaimEveryEntry: Stop and SessionStart carry no
// matcher, and an entry that matches everything would report the first
// unrelated hook in the list as holding its slot.
func TestAMatcherlessEntryDoesNotClaimEveryEntry(t *testing.T) {
	before := `{"hooks":{"Stop":[{"matcher":"Write","hooks":[{"type":"command","command":"theirs"}]}]}}`
	got, err := Merge(claude, []byte(before), []Entry{{Event: "Stop", Command: "ours"}})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(got.Foreign) != 0 || len(got.Added) != 1 {
		t.Fatalf("added %v, foreign %v", got.Added, got.Foreign)
	}
}

// An item of the list that is not an object is someone else's mistake, not
// ours to repair -- but it must not be read as a slot either.
func TestAnEntryThatIsNotAnObjectIsIgnoredForTheLookup(t *testing.T) {
	before := `{"hooks":{"Stop":["nonsense"]}}`
	got, err := Merge(claude, []byte(before), []Entry{{Event: "Stop", Command: "ours"}})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if !strings.Contains(string(got.Merged), "nonsense") || !strings.Contains(string(got.Merged), "ours") {
		t.Fatalf("merged = %s, want both", got.Merged)
	}
}

func TestMergePreservesTheLifecycleOrder(t *testing.T) {
	entries := []Entry{
		{Event: "Stop", Command: "stop-cmd"},
		{Event: "SessionStart", Command: "session-cmd"},
		{Event: "SubagentStop", Command: "sub-stop-cmd"},
		{Event: "PreToolUse", Command: "pre-cmd"},
		{Event: "SubagentStart", Command: "sub-start-cmd"},
		{Event: "PostToolUse", Command: "post-cmd"},
		{Event: "Custom", Command: "custom-cmd"},
	}
	res, err := Merge(claude, nil, entries)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	merged := string(res.Merged)
	last := -1
	for _, event := range []string{"SessionStart", "PreToolUse", "PostToolUse", "SubagentStart", "SubagentStop", "Stop", "Custom"} {
		at := strings.Index(merged, `"`+event+`"`)
		if at <= last {
			t.Fatalf("event %s out of order:\n%s", event, merged)
		}
		last = at
	}
}

func TestHookBlockAndCommandKeyOrder(t *testing.T) {
	res, err := Merge(claude, nil, []Entry{{Event: "PostToolUse", Matcher: "Write|Edit", Command: "run.sh", Timeout: 180}})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	merged := string(res.Merged)
	last := -1
	for _, key := range []string{`"matcher"`, `"hooks": [`, `"type"`, `"command"`, `"timeout"`} {
		at := strings.LastIndex(merged, key)
		if at <= last {
			t.Fatalf("key %s out of order:\n%s", key, merged)
		}
		last = at
	}
}

func TestFormatBlockAndCommandFallbacks(t *testing.T) {
	rawJSON := `{"hooks":{"CustomEvent":["not-a-map", {"hooks": "not-a-list"}, {"hooks": ["not-a-cmd-map", {"command": "second"}]}]}}`
	res, err := Merge(claude, []byte(rawJSON), []Entry{{Event: "Stop", Command: "ours"}})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	for _, want := range []string{"not-a-map", "not-a-list", "not-a-cmd-map"} {
		if !strings.Contains(string(res.Merged), want) {
			t.Fatalf("fallback lost %s:\n%s", want, res.Merged)
		}
	}
}

func TestExtractTopEntriesAndFormatRootEdgeCases(t *testing.T) {
	if entries := extractTopEntries(nil); entries != nil {
		t.Fatalf("want nil for nil input, got %v", entries)
	}
	if entries := extractTopEntries([]byte("[1,2,3]")); entries != nil {
		t.Fatalf("want nil for array input, got %v", entries)
	}
	if entries := extractTopEntries([]byte("{broken")); len(entries) != 0 {
		t.Fatalf("want 0 entries for broken json, got %v", entries)
	}
	if out := formatRoot(nil, nil, "hooks"); string(out) != "{}\n" {
		t.Fatalf("formatRoot empty: out=%q", out)
	}
	entries := []topEntry{{key: "raw", raw: json.RawMessage(`"simple"`)}}
	out := formatRoot(entries, map[string]any{"other": 123}, "hooks")
	if !strings.Contains(string(out), `"raw": "simple"`) || !strings.Contains(string(out), `"other": 123`) {
		t.Fatalf("formatRoot raw text failed: out=%q", out)
	}
}

func TestOrderHooksAndFormatBlockEdgeCases(t *testing.T) {
	hooks := map[string]any{
		"CustomEvent": "not-a-list",
		"SessionStart": []any{
			"string-block-item",
			map[string]any{
				"matcher":   "Write",
				"hooks":     []any{map[string]any{"type": "command", "command": "echo 1", "timeout": 5, "extra": "custom"}},
				"extraProp": true,
			},
		},
	}
	str := string(orderHooks(hooks))
	if !strings.Contains(str, `"CustomEvent": "not-a-list"`) || !strings.Contains(str, `"extra": "custom"`) || !strings.Contains(str, `"extraProp": true`) {
		t.Fatalf("orderHooks output missing extra keys: %s", str)
	}
	if got := string(orderHooks(nil)); got != "{}" {
		t.Fatalf("expected {}, got %s", got)
	}
	formatted := formatBlock(map[string]any{"matcher": "Write", "hooks": "not-a-slice"}, "  ")
	if !strings.Contains(formatted, `"not-a-slice"`) {
		t.Fatalf("expected non-slice hooks to be formatted, got %s", formatted)
	}
}

func TestAMatcherlessEntryWritesNoMatcher(t *testing.T) {
	if _, ok := blockFor(Entry{Event: "Stop", Command: "loomux hook stop"})["matcher"]; ok {
		t.Fatal("a block without a matcher got a matcher key")
	}
}

func TestMergeWritesEveryTopLevelKeyOnce(t *testing.T) {
	for _, existing := range []string{`{"hooks":{},"model":"opus"}`, `{"model":"opus"}`} {
		got, err := Merge(claude, []byte(existing), []Entry{{Event: "Stop", Command: "loomux hook stop"}})
		if err != nil {
			t.Fatalf("Merge(%s): %v", existing, err)
		}
		// Top-level keys stand at two spaces; the blocks inside carry their
		// own "hooks" deeper in.
		for _, key := range []string{"\n  \"hooks\":", "\n  \"model\":"} {
			if n := strings.Count(string(got.Merged), key); n != 1 {
				t.Errorf("Merge(%s): %s %d times:\n%s", existing, key, n, got.Merged)
			}
		}
	}
}

func TestOrderHooksWritesOnlyTheEventsThereAreOnce(t *testing.T) {
	got := string(orderHooks(map[string]any{"Stop": []any{}}))
	want := "{\n    \"Stop\": [\n    ]\n  }"
	if got != want {
		t.Fatalf("orderHooks = %q, want %q", got, want)
	}
}

func TestFormatBlockIndentsHooksThatAreNoList(t *testing.T) {
	got := formatBlock(map[string]any{"hooks": map[string]any{"a": 1}}, "")
	want := "{\n  \"hooks\": {\n    \"a\": 1\n  }\n}"
	if got != want {
		t.Fatalf("formatBlock = %q, want %q", got, want)
	}
}

func TestMergeKeepsForeignCommandsUnescaped(t *testing.T) {
	existing := []byte(`{"statusLine": {"command": "a > b"}, "env": {"X": "<y>"},
		"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "a && b"}, "x & y"]}],
		"Odd": {"k": "c < d"},
		"PreToolUse": [{"matcher": "*", "hooks": {"q": "e & f"}}]}}`)
	got, err := Merge(claude, existing, Entries(claude, Canonical))
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	for _, want := range []string{`"a > b"`, `"<y>"`, `"a && b"`, `"x & y"`, `"c < d"`, `"e & f"`} {
		if !bytes.Contains(got.Merged, []byte(want)) {
			t.Errorf("no %s in\n%s", want, got.Merged)
		}
	}
	if bytes.Contains(got.Merged, []byte(`\u00`)) {
		t.Errorf("escaped:\n%s", got.Merged)
	}
}

func TestMergeRecognisesLoomuxEntriesWhereverTheyStand(t *testing.T) {
	b := Canonical
	all := func(skip string) string {
		var blocks []string
		for _, e := range Entries(claude, b) {
			if e.Event == "PreToolUse" {
				continue
			}
			blocks = append(blocks, `"`+e.Event+`": [{"matcher": "`+e.Matcher+`", "hooks": [{"type": "command", "command": `+strconvQuote(e.Command)+`}]}]`)
		}
		return strings.Join(blocks, ", ") + ", " + skip
	}
	pre := `"${LOCALAPPDATA}/loomux/bin/loomux.exe" hook pre-tool-use --host claude --root x`
	for name, block := range map[string]string{
		// ulinit's matcher, from before MultiEdit joined it.
		"old matcher": `"PreToolUse": [{"matcher": "Write|Edit|NotebookEdit|Bash|PowerShell", "hooks": [{"type": "command", "command": ` + strconvQuote(pre) + `}]}]`,
		// Ours second in a block of the project on our matcher.
		"second": `"PreToolUse": [{"matcher": "Write|Edit|MultiEdit|NotebookEdit|Bash|PowerShell", "hooks": [{"type": "command", "command": "echo x"}, {"type": "command", "command": ` + strconvQuote(pre) + `}]}]`,
	} {
		existing := []byte(`{"hooks": {` + all(block) + `}}`)
		got, err := Merge(claude, existing, Entries(claude, b))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(got.Added) != 0 || !bytes.Equal(got.Merged, existing) {
			t.Errorf("%s: added %v\n%s", name, got.Added, got.Merged)
		}
		if note := "PreToolUse: kept an own entry under matcher Write|Edit|NotebookEdit|Bash|PowerShell"; name == "old matcher" &&
			(len(got.Notes) != 1 || got.Notes[0] != note) {
			t.Errorf("%s: notes %v", name, got.Notes)
		}
	}
	// Another hook of ours under another matcher is no stand-in.
	other := `{"hooks": {"PreToolUse": [{"matcher": "Read", "hooks": [{"type": "command", "command": "loomux hook post-tool-use"}, "x"]}]}}`
	if got, _ := Merge(claude, []byte(other), Entries(claude, b)); !slices.Contains(got.Added, "PreToolUse/Write|Edit|MultiEdit|NotebookEdit|Bash|PowerShell") {
		t.Errorf("added %v", got.Added)
	}
	if hookEventOf("loomux hook") != "" || hookEventOf(`"unclosed`) != "" {
		t.Error("hookEventOf reads a hook that is not there")
	}
}

func strconvQuote(s string) string { q, _ := json.Marshal(s); return string(q) }

// firstCommand is the command of a block's first hook, "" where there is none.
func firstCommand(item map[string]any) string {
	hooks, ok := item["hooks"].([]any)
	if !ok || len(hooks) == 0 {
		return ""
	}
	first, ok := hooks[0].(map[string]any)
	if !ok {
		return ""
	}
	cmd, _ := first["command"].(string)
	return cmd
}

func TestBinaryOfReadsEveryCommandOfABlock(t *testing.T) {
	existing := []byte(`{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "echo x"},` +
		`{"type": "command", "command": "\"${CLAUDE_PROJECT_DIR}/bin/loomux.exe\" hook stop"}]}]}}`)
	if got := BinaryOf(claude, existing); got != Checkout {
		t.Fatalf("BinaryOf = %s, want Checkout", got)
	}
}
