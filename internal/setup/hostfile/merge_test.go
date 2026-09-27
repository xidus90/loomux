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
	wanted := Entries(hosts.HostAntigravity, Canonical)
	got, err := Merge(hosts.HostAntigravity, existing, wanted)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(got.Added) != 4 {
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

// agy refuses the whole file when PreInvocation or Stop hold a block: they
// get their handler directly, the tool events a block with matcher and hooks.
// Merged again, the flat handlers count as ours and nothing is added.
func TestMergeWritesAntigravitysFlatEventsFlat(t *testing.T) {
	wanted := Entries(hosts.HostAntigravity, Canonical)
	got, err := Merge(hosts.HostAntigravity, nil, wanted)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	var root map[string]map[string][]map[string]any
	if err := json.Unmarshal(got.Merged, &root); err != nil {
		t.Fatalf("result: %v\n%s", err, got.Merged)
	}
	for _, event := range []string{"PreInvocation", "Stop"} {
		handler := root["loomux"][event][0]
		if _, isBlock := handler["hooks"]; isBlock || !Owned(handler["command"].(string)) || handler["type"] != "command" {
			t.Fatalf("%s = %v, want a flat handler", event, handler)
		}
	}
	for _, event := range []string{"PreToolUse", "PostToolUse"} {
		if block := root["loomux"][event][0]; block["matcher"] == nil || block["hooks"] == nil {
			t.Fatalf("%s = %v, want a block", event, block)
		}
	}
	again, err := Merge(hosts.HostAntigravity, got.Merged, wanted)
	if err != nil || len(again.Added) != 0 || !bytes.Equal(again.Merged, got.Merged) {
		t.Fatalf("again: added %v, err %v", again.Added, err)
	}
}

// Another group is carried over token for token: the same keys in the same
// order, the same escapes and numbers; only the indentation is the file's
// two spaces. A group already written that way comes back byte for byte.
func TestMergeCarriesEveryOtherAntigravityGroupOver(t *testing.T) {
	indented := "{\n    \"PreToolUse\": [\n      {\n        \"matcher\": \"run_command\",\n" +
		"        \"hooks\": [\n          {\n            \"type\": \"command\",\n" +
		"            \"command\": \"brain guard\"\n          }\n        ]\n      }\n    ]\n  }"
	compact := `{"z":1.0,"a":"x \u0026 y","m":[true,null]}`
	existing := []byte("{\n  \"indented\": " + indented + ",\n  \"compact\": " + compact + "\n}\n")
	got, err := Merge(hosts.HostAntigravity, existing, Entries(hosts.HostAntigravity, Canonical))
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	merged := string(got.Merged)
	if !strings.Contains(merged, "\"indented\": "+indented+",\n") {
		t.Errorf("the indented group changed:\n%s", merged)
	}
	want := "\"compact\": {\n    \"z\": 1.0,\n    \"a\": \"x \\u0026 y\",\n    \"m\": [\n      true,\n      null\n    ]\n  }"
	if !strings.Contains(merged, want) {
		t.Errorf("the compact group lost a token:\n%s", merged)
	}
	if !strings.Contains(merged, "\"loomux\": {") {
		t.Errorf("no group loomux:\n%s", merged)
	}
}

// A group of the project that already runs one of our hooks is named and
// left alone, whether the merge adds anything or not: the hook now fires
// twice, and the group is not ours to repair. Any form of the call counts --
// a bare loomux, another --root, the quoted ${LOCALAPPDATA} path -- as long
// as a loomux binary runs the same hook; another loomux command, or another
// program running a "hook", does not.
func TestMergeNamesAnotherGroupThatRunsOurCommand(t *testing.T) {
	wanted := Entries(hosts.HostAntigravity, Canonical)
	pre := wanted[1].Command
	group := func(command string) string {
		return `{"PreToolUse":[{"hooks":[{"type":"command","command":` + strconvQuote(command) + `}]}]}`
	}
	existing := []byte(`{"zeta":{"PreToolUse":[{"matcher":"x","hooks":[{"type":"command","command":` + strconvQuote(pre) + `}]}]},` +
		`"alpha":{"deep":{"list":[{"command":` + strconvQuote(pre) + `}]}},` +
		`"other":{"PreToolUse":[{"hooks":[{"command":"loomux hook pre-tool-use"},{"command":7}]}]},` +
		`"rooted":` + group(AntigravityBinary+" hook pre-tool-use --host antigravity --root .") + `,` +
		`"quoted":` + group(`"${LOCALAPPDATA}/loomux/bin/loomux.exe" hook pre-tool-use --host antigravity --root ..`) + `,` +
		`"stopper":` + group(`C:\Users\x\AppData\Local\loomux\bin\LOOMUX.EXE hook stop --host antigravity`) + `,` +
		`"check":` + group("loomux check precommit") + `,` +
		`"foreign":` + group("mytool hook pre-tool-use") + `,` +
		`"plain":"x"}`)
	got, err := Merge(hosts.HostAntigravity, existing, wanted)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	want := []string{
		"the group alpha already runs loomux hook pre-tool-use; it now fires twice",
		"the group other already runs loomux hook pre-tool-use; it now fires twice",
		"the group quoted already runs loomux hook pre-tool-use; it now fires twice",
		"the group rooted already runs loomux hook pre-tool-use; it now fires twice",
		"the group stopper already runs loomux hook stop; it now fires twice",
		"the group zeta already runs loomux hook pre-tool-use; it now fires twice",
	}
	if !reflect.DeepEqual(got.Notes, want) {
		t.Fatalf("notes = %q, want %q", got.Notes, want)
	}
	if !bytes.Contains(got.Merged, []byte(`"zeta": {`)) || len(got.Added) != 4 {
		t.Fatalf("added %v:\n%s", got.Added, got.Merged)
	}
	again, err := Merge(hosts.HostAntigravity, got.Merged, wanted)
	if err != nil {
		t.Fatalf("second Merge: %v", err)
	}
	if len(again.Added) != 0 || !bytes.Equal(again.Merged, got.Merged) || !reflect.DeepEqual(again.Notes, want) {
		t.Fatalf("second merge added %v, notes %q", again.Added, again.Notes)
	}
	// Claude's file has no groups; its hooks object is read as before.
	claudeFile := []byte(`{"alpha":{"command":` + strconvQuote(pre) + `}}`)
	if got, _ := Merge(claude, claudeFile, nil); len(got.Notes) != 0 {
		t.Fatalf("claude notes = %q", got.Notes)
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

// A root that is null is a root that is not an object, in either host's
// file: refused with the file named, never replaced by one.
func TestANullRootIsRefused(t *testing.T) {
	for _, host := range []hosts.Host{claude, hosts.HostAntigravity} {
		_, err := Merge(host, []byte("null"), []Entry{{Event: "Stop", Command: "ours"}})
		if err == nil || !strings.Contains(err.Error(), Path(host)+" is not a JSON object: its root is null") {
			t.Errorf("Merge(%s, null): err = %v", host, err)
		}
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
	// Ours second in a block of the project on our matcher.
	second := `"PreToolUse": [{"matcher": "Write|Edit|MultiEdit|NotebookEdit|Bash|PowerShell", "hooks": [{"type": "command", "command": "echo x"}, {"type": "command", "command": ` + strconvQuote(pre) + `}]}]`
	existing := []byte(`{"hooks": {` + all(second) + `}}`)
	got, err := Merge(claude, existing, Entries(claude, b))
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if len(got.Added) != 0 || !bytes.Equal(got.Merged, existing) {
		t.Errorf("second: added %v\n%s", got.Added, got.Merged)
	}
	// A command of ours under ulinit's matcher, from before MultiEdit joined
	// it, stays and gets a block for MultiEdit beside it.
	ulinit := `"PreToolUse": [{"matcher": "Write|Edit|NotebookEdit|Bash|PowerShell", "hooks": [{"type": "command", "command": ` + strconvQuote(pre) + `}]}]`
	got, err = Merge(claude, []byte(`{"hooks": {`+all(ulinit)+`}}`), Entries(claude, b))
	if err != nil {
		t.Fatalf("old matcher: %v", err)
	}
	note := "PreToolUse: kept an own entry under matcher Write|Edit|NotebookEdit|Bash|PowerShell; added one for MultiEdit"
	if !reflect.DeepEqual(got.Added, []string{"PreToolUse/MultiEdit"}) || !reflect.DeepEqual(got.Notes, []string{note}) {
		t.Errorf("old matcher: added %v, notes %v", got.Added, got.Notes)
	}
	wantPre := []string{pre, Entries(claude, b)[1].Command}
	if prePre := commands(t, got.Merged, "hooks", "PreToolUse"); !reflect.DeepEqual(prePre, wantPre) {
		t.Errorf("old matcher: PreToolUse = %q, want %q", prePre, wantPre)
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

// A root of null parses into a nil map without an error; it is a root that
// is no object, and the file is refused rather than written over.
func TestMergeRefusesANullRoot(t *testing.T) {
	for _, existing := range []string{"null", " null\n"} {
		_, err := Merge(claude, []byte(existing), Entries(claude, Canonical))
		if err == nil || !strings.Contains(err.Error(), ".claude/settings.json") {
			t.Errorf("Merge(%q): err = %v, want one naming the file", existing, err)
		}
	}
}

// An own block on the slot that runs something else than the entry -- an
// old binary, an old subcommand -- is kept, never rewritten, but named, so
// "kept" does not pass for "current".
func TestMergeNamesAStaleOwnEntry(t *testing.T) {
	existing := []byte(`{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"C:/old/loomux.exe hook stop --host claude"}]}]}}`)
	want := []Entry{{Event: "Stop", Command: Canonical + " hook stop --host claude"}}
	got, err := Merge(claude, existing, want)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if !slices.Equal(got.Kept, []string{"Stop/"}) || !bytes.Equal(got.Merged, existing) {
		t.Fatalf("kept %v, merged %s; want the block kept as it is", got.Kept, got.Merged)
	}
	if len(got.Notes) != 1 || !strings.Contains(got.Notes[0], "C:/old/loomux.exe hook stop") ||
		!strings.Contains(got.Notes[0], "Stop") {
		t.Fatalf("notes %v, want one naming the stale command", got.Notes)
	}
}

// A block that holds a command of its own beside a hooks list runs both; our
// entry in the list is found, and none is added beside it.
func TestMergeFindsAnEntryInTheListOfABlockWithACommand(t *testing.T) {
	entry := Entry{Event: "Stop", Command: Canonical + " hook stop --host claude"}
	existing := []byte(`{"hooks":{"Stop":[{"command":"echo x","hooks":[{"type":"command","command":` +
		string(EncodeJSON(entry.Command, "", "")) + `}]}]}}`)
	got, err := Merge(claude, existing, []Entry{entry})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(got.Added) != 0 || !slices.Equal(got.Kept, []string{"Stop/"}) {
		t.Fatalf("added %v, kept %v; want the entry found", got.Added, got.Kept)
	}
}

func TestMergeNamesNothingForACurrentOwnEntry(t *testing.T) {
	existing, err := os.ReadFile("testdata/loomux-settings.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Merge(claude, existing, Entries(claude, BinaryOf(claude, existing)))
	if err != nil || len(got.Notes) != 0 {
		t.Fatalf("notes %v, err %v; want none for loomux's own settings", got.Notes, err)
	}
}

// The fixture above is a copy, and a copy drifts: it once lacked the
// worktree hooks the real file had. This holds the repository's own hook
// file to what init would write -- no entry of ours missing, nothing
// rewritten -- while hooks a developer added beside ours are left to them.
func TestTheRepositorysOwnSettingsNeedNoChange(t *testing.T) {
	own, err := os.ReadFile("../../../.claude/settings.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Merge(claude, own, Entries(claude, BinaryOf(claude, own)))
	if err != nil || len(got.Added) != 0 || len(got.Notes) != 0 || !bytes.Equal(got.Merged, own) {
		t.Fatalf("added %v, notes %v, err %v; init would change .claude/settings.json", got.Added, got.Notes, err)
	}
}

// antigravityWithMatcher is the file init writes for Antigravity, its
// PreToolUse block under matcher instead of the one Entries wants.
func antigravityWithMatcher(t *testing.T, matcher string) []byte {
	t.Helper()
	fresh, err := Merge(hosts.HostAntigravity, nil, Entries(hosts.HostAntigravity, Canonical))
	if err != nil {
		t.Fatal(err)
	}
	want := `"matcher": ` + strconvQuote(Entries(hosts.HostAntigravity, Canonical)[1].Matcher)
	if !bytes.Contains(fresh.Merged, []byte(want)) {
		t.Fatalf("no PreToolUse matcher to replace in\n%s", fresh.Merged)
	}
	return bytes.Replace(fresh.Merged, []byte(want), []byte(`"matcher": `+strconvQuote(matcher)), 1)
}

// An own block from before a tool joined the matcher gets a block for that
// tool beside it, with the same command; the old block stays as it is.
func TestMergeAddsABlockForTheToolsAnOldMatcherLacks(t *testing.T) {
	old := "write_to_file|replace_file_content|multi_replace_file_content|run_command|send_command_input"
	existing := antigravityWithMatcher(t, old)
	wanted := Entries(hosts.HostAntigravity, Canonical)
	got, err := Merge(hosts.HostAntigravity, existing, wanted)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if !reflect.DeepEqual(got.Added, []string{"PreToolUse/manage_task"}) {
		t.Fatalf("added %v, want the block for manage_task", got.Added)
	}
	note := "PreToolUse: kept an own entry under matcher " + old + "; added one for manage_task"
	if !reflect.DeepEqual(got.Notes, []string{note}) {
		t.Fatalf("notes %v, want %q", got.Notes, note)
	}
	var root map[string]map[string][]map[string]any
	if err := json.Unmarshal(got.Merged, &root); err != nil {
		t.Fatalf("result: %v\n%s", err, got.Merged)
	}
	pre := root["loomux"]["PreToolUse"]
	if len(pre) != 2 || pre[0]["matcher"] != old || pre[1]["matcher"] != "manage_task" ||
		firstCommand(pre[0]) != firstCommand(pre[1]) || firstCommand(pre[1]) != wanted[1].Command {
		t.Fatalf("PreToolUse = %v, want the old block and one for manage_task with the same command", pre)
	}
}

// The appended block and the old one together cover the matcher, so the
// next run finds both, adds nothing and names both.
func TestMergeAddsNothingTwiceForASplitMatcher(t *testing.T) {
	old := "write_to_file|replace_file_content|multi_replace_file_content|run_command|send_command_input"
	wanted := Entries(hosts.HostAntigravity, Canonical)
	first, err := Merge(hosts.HostAntigravity, antigravityWithMatcher(t, old), wanted)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	again, err := Merge(hosts.HostAntigravity, first.Merged, wanted)
	if err != nil {
		t.Fatalf("second Merge: %v", err)
	}
	if len(again.Added) != 0 || !bytes.Equal(again.Merged, first.Merged) {
		t.Fatalf("second merge added %v\n%s", again.Added, again.Merged)
	}
	note := "PreToolUse: kept an own entry under matcher " + old + ", manage_task"
	if !reflect.DeepEqual(again.Notes, []string{note}) {
		t.Fatalf("notes %v, want %q", again.Notes, note)
	}
}

// Two old blocks of ours whose matchers together cover the wanted one leave
// nothing to add.
func TestMergeAddsNothingWhereTwoOldBlocksCoverTheMatcher(t *testing.T) {
	wanted := []Entry{{Event: "PreToolUse", Matcher: "Write|Edit|Bash", Command: "loomux hook pre-tool-use"}}
	existing := []byte(`{"hooks":{"PreToolUse":[` +
		`{"matcher":"Write|Edit","hooks":[{"type":"command","command":"loomux hook pre-tool-use"}]},` +
		`{"matcher":"Bash","hooks":[{"type":"command","command":"loomux hook pre-tool-use --old"}]}]}}`)
	got, err := Merge(claude, existing, wanted)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(got.Added) != 0 || !bytes.Equal(got.Merged, existing) ||
		!reflect.DeepEqual(got.Notes, []string{"PreToolUse: kept an own entry under matcher Write|Edit, Bash"}) {
		t.Fatalf("added %v, notes %v", got.Added, got.Notes)
	}
}

// Only a flat list of names can be counted. An own matcher that is a regular
// expression, or a wanted one that is, gets the note alone, and so does one
// that already names every wanted tool and more.
func TestMergeOnlyNotesAnOldMatcherItCannotCountOrThatLacksNothing(t *testing.T) {
	for _, tc := range []struct{ have, want string }{
		{".*", "Write|Bash"},
		{"Write|Ba.*", "Write|Bash"},
		{"Write||Bash", "Write|Bash|Read"},
		{"Write", "Write|Ba.*"},
		{"Write|Bash|Read", "Write|Bash"},
	} {
		wanted := []Entry{{Event: "PreToolUse", Matcher: tc.want, Command: "loomux hook pre-tool-use"}}
		existing := []byte(`{"hooks":{"PreToolUse":[{"matcher":` + strconvQuote(tc.have) +
			`,"hooks":[{"type":"command","command":"loomux hook pre-tool-use"}]}]}}`)
		got, err := Merge(claude, existing, wanted)
		if err != nil {
			t.Fatalf("%s over %s: %v", tc.want, tc.have, err)
		}
		note := "PreToolUse: kept an own entry under matcher " + tc.have
		if len(got.Added) != 0 || !bytes.Equal(got.Merged, existing) || !reflect.DeepEqual(got.Notes, []string{note}) {
			t.Errorf("%s over %s: added %v, notes %v", tc.want, tc.have, got.Added, got.Notes)
		}
	}
}

// An entry without a matcher is no flat list either: a Stop of ours under a
// matcher it does not want is kept with the note. So is a PreToolUse block
// of ours without a matcher key, which Claude Code runs for every tool; the
// note names its matcher (none).
func TestMergeOnlyNotesAMatcherlessEntryKeptUnderAMatcher(t *testing.T) {
	for _, tc := range []struct {
		existing string
		wanted   Entry
		note     string
	}{
		{
			`{"hooks":{"Stop":[{"matcher":"x","hooks":[{"type":"command","command":"loomux hook stop"}]}]}}`,
			Entry{Event: "Stop", Command: "loomux hook stop"},
			"Stop: kept an own entry under matcher x",
		},
		{
			`{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"loomux hook pre-tool-use"}]}]}}`,
			Entry{Event: "PreToolUse", Matcher: "Write|Edit|Bash", Command: "loomux hook pre-tool-use"},
			"PreToolUse: kept an own entry under matcher (none)",
		},
	} {
		got, err := Merge(claude, []byte(tc.existing), []Entry{tc.wanted})
		if err != nil {
			t.Fatalf("%s: %v", tc.wanted.Event, err)
		}
		if len(got.Added) != 0 || !bytes.Equal(got.Merged, []byte(tc.existing)) || !reflect.DeepEqual(got.Notes, []string{tc.note}) {
			t.Fatalf("%s: added %v, notes %v", tc.wanted.Event, got.Added, got.Notes)
		}
	}
}
