package benchcases

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const oldSettings = `{
  "hooks": {
    "SessionStart": [{"hooks": [{"type": "command", "command": "uv run --project \"${CLAUDE_PROJECT_DIR}/.ultraloom/vendor/ultraloom\" ultraloom hook session-start --root \"${CLAUDE_PROJECT_DIR}\""}]}],
    "PreToolUse": [
      {"matcher": "Write|Edit|Bash", "hooks": [{"type": "command", "command": "ulguard --root \"${CLAUDE_PROJECT_DIR}\""}]},
      {"matcher": "NotebookEdit", "hooks": [{"type": "command", "command": "never-matches"}]},
      {"matcher": "", "hooks": [{"type": "command", "command": "brain guard"}]}
    ],
    "Stop": [{"matcher": "wiki", "hooks": [{"type": "command", "command": "brain wiki-gate --root \"${CLAUDE_PROJECT_DIR}\""}]}]
  }
}`

func TestBuildMakesOneCasePerEventInFixedOrder(t *testing.T) {
	root := "C:/Users/me/Documents/#GIT/my project"
	cases, payloads, err := Build([]byte(oldSettings), root, root+"/README.md", "C:/out")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, c := range cases {
		names = append(names, c.Name)
	}
	want := []string{"SessionStart", "PreToolUse (Edit on README.md)", "Stop"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	pre := cases[1]
	// Two groups match Edit (the plain one and the empty matcher), the
	// NotebookEdit one does not: two steps, started together.
	if pre.Mode != "par" || len(pre.Steps) != 2 {
		t.Fatalf("PreToolUse = mode %q with %d steps, want par with 2", pre.Mode, len(pre.Steps))
	}
	if got := pre.Steps[0].Argv; !reflect.DeepEqual(got, []string{"ulguard", "--root", root}) {
		t.Errorf("argv = %q, want the root, with its space and #, as one argument", got)
	}
	if cases[0].Mode != "single" || cases[0].Dir != root || cases[0].Stdin != "C:/out/payload-SessionStart.json" {
		t.Errorf("SessionStart case = %+v", cases[0])
	}
	if len(payloads) != 3 || payloads[0].Name != "payload-SessionStart.json" {
		t.Fatalf("payloads = %d, first %q", len(payloads), payloads[0].Name)
	}
	var edit struct {
		Tool  string `json:"tool_name"`
		Input struct {
			Path string `json:"file_path"`
		} `json:"tool_input"`
		Cwd string `json:"cwd"`
	}
	if err := json.Unmarshal(payloads[1].Data, &edit); err != nil {
		t.Fatal(err)
	}
	if edit.Tool != "Edit" || edit.Input.Path != root+"/README.md" || edit.Cwd != root {
		t.Errorf("edit payload = %+v", edit)
	}
}

func TestBuildIgnoresTheMatcherOfAnEventWithoutTools(t *testing.T) {
	// The Stop group carries the matcher "wiki", which matches no tool.
	cases, _, err := Build([]byte(oldSettings), "/r", "/r/a.md", "/o")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 3 {
		t.Fatalf("cases = %+v, want three", cases)
	}
	if last := cases[len(cases)-1]; last.Name != "Stop" || len(last.Steps) != 1 {
		t.Errorf("last case = %+v, want Stop with its one step", last)
	}
}

func TestMatchesToolAnchorsTheWholeNameAndKnowsTheTwoWildcards(t *testing.T) {
	cases := []struct {
		matcher string
		want    bool
	}{
		{"", true}, {"*", true}, {"Edit", true}, {"Write|Edit", true},
		{"NotebookEdit", false}, {"Edi", false}, {"Edit|", true},
		{"MultiEdit", false}, {"Edit.*", true}, {"Edits", false},
		{"Bash|Write", false}, {"Bash|Edit", true}, {".*Edit", true},
	}
	for _, c := range cases {
		got, err := matchesTool(c.matcher, "Edit")
		if err != nil || got != c.want {
			t.Errorf("matchesTool(%q) = %v, %v; want %v", c.matcher, got, err, c.want)
		}
	}
	// The anchors bind the whole alternation, not only its first and last branch.
	if got, _ := matchesTool("Write|Edit", "Writer"); got {
		t.Errorf("Write|Edit must not match Writer")
	}
	if got, _ := matchesTool("Write|Edit", "NotebookEdit"); got {
		t.Errorf("Write|Edit must not match NotebookEdit")
	}
	if got, err := matchesTool("(", "Edit"); err == nil || got {
		t.Errorf("a broken expression must be an error and no match, got %v, %v", got, err)
	}
}

func TestBuildCountsAStarMatcherAsEveryTool(t *testing.T) {
	settings := `{"hooks": {"PreToolUse": [{"matcher": "*", "hooks": [{"command": "a"}]}, {"hooks": [{"command": "b"}]}]}}`
	cases, _, err := Build([]byte(settings), "/r", "/r/a.md", "/o")
	if err != nil || len(cases) != 1 || len(cases[0].Steps) != 2 {
		t.Fatalf("cases = %+v, err %v; want one case with both steps", cases, err)
	}
}

func TestPayloadCarriesWhatEachEventNeeds(t *testing.T) {
	decode := func(event string) map[string]any {
		var doc map[string]any
		if err := json.Unmarshal(payload(event, "/r", "/r/a.md"), &doc); err != nil {
			t.Fatal(err)
		}
		return doc
	}
	if d := decode("SessionStart"); d["source"] != "startup" || d["cwd"] != "/r" || d["hook_event_name"] != "SessionStart" || d["tool_name"] != nil || d["stop_hook_active"] != nil {
		t.Errorf("SessionStart = %v", d)
	}
	if d := decode("Stop"); d["stop_hook_active"] != false || d["source"] != nil || d["tool_name"] != nil {
		t.Errorf("Stop = %v", d)
	}
	post := decode("PostToolUse")
	if resp, ok := post["tool_response"].(map[string]any); !ok || resp["success"] != true || post["tool_name"] != "Edit" || post["source"] != nil {
		t.Errorf("PostToolUse = %v", post)
	}
	pre := decode("PreToolUse")
	if pre["tool_response"] != nil || pre["tool_name"] != "Edit" || pre["hook_event_name"] != "PreToolUse" || pre["source"] != nil {
		t.Errorf("PreToolUse = %v", pre)
	}
	for name, d := range map[string]map[string]any{"PreToolUse": pre, "PostToolUse": post} {
		in, ok := d["tool_input"].(map[string]any)
		if !ok || in["file_path"] != "/r/a.md" || in["old_string"] != "a" || in["new_string"] != "b" {
			t.Errorf("%s tool_input = %v", name, d["tool_input"])
		}
	}
	for _, event := range []string{"SubagentStart", "SubagentStop"} {
		d := decode(event)
		if d["tool_name"] != nil || d["source"] != nil || d["stop_hook_active"] != nil || d["tool_input"] != nil || d["hook_event_name"] != event || d["cwd"] != "/r" {
			t.Errorf("%s = %v", event, d)
		}
	}
}

func TestPayloadCarriesTheSessionAndTheSubagentAHostNames(t *testing.T) {
	for _, event := range events {
		var d map[string]any
		if err := json.Unmarshal(payload(event, "/r", "/r/a.md"), &d); err != nil {
			t.Fatal(err)
		}
		if d["session_id"] != "loomux-bench" {
			t.Errorf("%s session_id = %v", event, d["session_id"])
		}
		if d["transcript_path"] != "/r/.loomux-bench-no-transcript.jsonl" {
			t.Errorf("%s transcript_path = %v", event, d["transcript_path"])
		}
		sub := event == "SubagentStart" || event == "SubagentStop"
		if (d["agent_id"] == "loomux-bench-agent") != sub || (d["agent_type"] == "general-purpose") != sub {
			t.Errorf("%s agent_id = %v, agent_type = %v; want them only on the subagent events", event, d["agent_id"], d["agent_type"])
		}
		if !sub && (d["agent_id"] != nil || d["agent_type"] != nil) {
			t.Errorf("%s carries agent fields: %v", event, d)
		}
	}
}

func TestBuildRefusesWhatItCannotMeasure(t *testing.T) {
	cases := map[string]struct{ settings, want string }{
		"no hooks":         {`{}`, "no hook"},
		"empty hooks":      {`{"hooks": {}}`, "no hook"},
		"only other tool":  {`{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"command": "x"}]}]}}`, "no hook"},
		"bad matcher":      {`{"hooks": {"PreToolUse": [{"matcher": "(", "hooks": [{"command": "x"}]}]}}`, "PreToolUse"},
		"bad post matcher": {`{"hooks": {"PostToolUse": [{"matcher": "(", "hooks": [{"command": "x"}]}]}}`, "PostToolUse"},
		"empty command":    {`{"hooks": {"Stop": [{"hooks": [{"command": ""}]}]}}`, "Stop"},
		"blank command":    {`{"hooks": {"SubagentStop": [{"hooks": [{"command": " \t "}]}]}}`, "SubagentStop"},
		"open quote":       {`{"hooks": {"Stop": [{"hooks": [{"command": "x \"y"}]}]}}`, "Stop"},
		"not json":         {`{`, "not valid JSON"},
	}
	for name, c := range cases {
		_, _, err := Build([]byte(c.settings), "/r", "/r/a.md", "/o")
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want one naming %q", name, err, c.want)
		}
	}
}

func TestBuildWalksEveryEventInOrderAndNamesThem(t *testing.T) {
	settings := `{"hooks": {
		"Stop": [{"hooks": [{"command": "s"}]}],
		"SubagentStop": [{"hooks": [{"command": "ss"}]}],
		"SubagentStart": [{"hooks": [{"command": "sa"}]}],
		"PostToolUse": [{"matcher": "Edit", "hooks": [{"command": "p"}]}],
		"PreToolUse": [{"hooks": [{"command": "q"}]}],
		"SessionStart": [{"hooks": [{"command": "a"}]}],
		"Unknown": [{"hooks": [{"command": "u"}]}]
	}}`
	cases, payloads, err := Build([]byte(settings), "/r", "/r/docs/a.md", "/o")
	if err != nil || len(cases) != 6 || len(payloads) != 6 {
		t.Fatalf("cases = %d, payloads = %d, err %v; want six each", len(cases), len(payloads), err)
	}
	var names, files, stdins []string
	for _, c := range cases {
		names = append(names, c.Name)
		stdins = append(stdins, c.Stdin)
		if c.Dir != "/r" || c.Mode != "single" || len(c.Steps) != 1 {
			t.Errorf("case %+v", c)
		}
	}
	for _, p := range payloads {
		files = append(files, p.Name)
	}
	wantNames := []string{"SessionStart", "PreToolUse (Edit on a.md)", "PostToolUse (Edit on a.md)", "SubagentStart", "SubagentStop", "Stop"}
	wantFiles := []string{"payload-SessionStart.json", "payload-PreToolUse.json", "payload-PostToolUse.json", "payload-SubagentStart.json", "payload-SubagentStop.json", "payload-Stop.json"}
	if !reflect.DeepEqual(names, wantNames) || !reflect.DeepEqual(files, wantFiles) {
		t.Fatalf("names = %q, payloads = %q", names, files)
	}
	for i, p := range payloads {
		if stdins[i] != "/o/"+p.Name {
			t.Errorf("stdin %q does not name payload %q", stdins[i], p.Name)
		}
		var doc map[string]any
		if err := json.Unmarshal(p.Data, &doc); err != nil || doc["hook_event_name"] == nil || !strings.HasPrefix(p.Name, "payload-"+doc["hook_event_name"].(string)) {
			t.Errorf("payload %s = %s, %v", p.Name, p.Data, err)
		}
	}
}

func TestBuildStartsSeveralHooksOfOneGroupTogether(t *testing.T) {
	settings := `{"hooks": {"Stop": [{"hooks": [{"command": "a"}, {"command": "b"}]}], "SessionStart": [{"hooks": [{"command": "c"}]}]}}`
	cases, _, err := Build([]byte(settings), "/r", "/r/a.md", "/o")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 2 || cases[0].Mode != "single" || cases[1].Mode != "par" || len(cases[1].Steps) != 2 {
		t.Errorf("cases = %+v", cases)
	}
}

func TestBuildFillsTheProjectDirectoryEverywhere(t *testing.T) {
	settings := `{"hooks": {"Stop": [{"hooks": [{"command": "x ${CLAUDE_PROJECT_DIR}/a ${CLAUDE_PROJECT_DIR}/b"}]}]}}`
	cases, _, err := Build([]byte(settings), "/r", "/r/a.md", "/o")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 {
		t.Fatalf("cases = %+v", cases)
	}
	if got := cases[0].Steps[0].Argv; !reflect.DeepEqual(got, []string{"x", "/r/a", "/r/b"}) {
		t.Errorf("argv = %q", got)
	}
}

func TestBuildSkipsATooleventWhoseGroupsAllMissTheEdit(t *testing.T) {
	settings := `{"hooks": {"PostToolUse": [{"matcher": "Bash", "hooks": [{"command": "x"}]}], "Stop": [{"hooks": [{"command": "y"}]}]}}`
	cases, payloads, err := Build([]byte(settings), "/r", "/r/a.md", "/o")
	if err != nil || len(cases) != 1 || cases[0].Name != "Stop" || len(payloads) != 1 {
		t.Errorf("cases = %+v, payloads = %d, err %v", cases, len(payloads), err)
	}
}

func TestSplitKeepsQuotedWordsWhole(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`a b  c`, []string{"a", "b", "c"}},
		{`a "b c" d`, []string{"a", "b c", "d"}},
		{`a 'b "c"' d`, []string{"a", `b "c"`, "d"}},
		{`a "b \"c\" d"`, []string{"a", `b "c" d`}},
		{`x"y z"w`, []string{"xy zw"}},
		{`""`, []string{""}},
		{"a\tb", []string{"a", "b"}},
		{"\t a \t", []string{"a"}},
		{`a "" b`, []string{"a", "", "b"}},
		{`a ''`, []string{"a", ""}},
		{`'a\b'`, []string{`a\b`}},
		{`'a\'`, []string{`a\`}},
		{`'a\\b'`, []string{`a\\b`}},
		{`'a\"b'`, []string{`a\"b`}},
		{`"a\\b"`, []string{`a\b`}},
		{`"a\nb"`, []string{`a\nb`}},
		{`"a\\"`, []string{`a\`}},
		{`"it's"`, []string{"it's"}},
		{`'say "hi"'`, []string{`say "hi"`}},
		{`"a\'b"`, []string{`a\'b`}},
		{``, nil},
		{`   `, nil},
	}
	for _, c := range cases {
		got, err := split(c.in)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("split(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	for in, quote := range map[string]string{`a 'b`: "'", `a "b`: `"`, `"a\`: `"`, `"a\"`: `"`} {
		if _, err := split(in); err == nil || !strings.Contains(err.Error(), "unterminated "+quote+" quote") {
			t.Errorf("split(%q): an open quote must be an error naming %s, got %v", in, quote, err)
		}
	}
}
