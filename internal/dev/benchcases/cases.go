// Package benchcases turns the hook configuration of a project into the
// case file `loomux dev bench hooks` measures, so the inventory of what runs
// on an edit comes from the project and not from a list written by hand. It
// reads the old configuration before a project is switched over and the new
// one after, and both sides name a case after its event, which is what lets
// `loomux dev bench compare` pair them.
package benchcases

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xidus90/loomux/internal/dev/benchhooks"
)

// Payload is one stdin file a case reads: its name within the output
// directory and its content.
type Payload struct {
	Name string
	Data []byte
}

// events are the hook events of a Claude settings file, in the order a
// session meets them.
var events = []string{"SessionStart", "PreToolUse", "PostToolUse", "SubagentStart", "SubagentStop", "Stop"}

// toolEvent reports whether the matcher of an event names a tool.
func toolEvent(event string) bool { return event == "PreToolUse" || event == "PostToolUse" }

type settingsFile struct {
	Hooks map[string][]struct {
		Matcher string `json:"matcher"`
		Hooks   []struct {
			Command string `json:"command"`
		} `json:"hooks"`
	} `json:"hooks"`
}

// Build makes one case per event that has a command for the sample edit.
// Commands of one event run together (mode par), as a host starts them.
func Build(settings []byte, root, file, dir string) ([]benchhooks.Case, []Payload, error) {
	var parsed settingsFile
	if err := json.Unmarshal(settings, &parsed); err != nil {
		return nil, nil, fmt.Errorf("settings: not valid JSON: %w", err)
	}
	var cases []benchhooks.Case
	var payloads []Payload
	for _, event := range events {
		var steps []benchhooks.Step
		for _, group := range parsed.Hooks[event] {
			if toolEvent(event) {
				ok, err := matchesTool(group.Matcher, "Edit")
				if err != nil {
					return nil, nil, fmt.Errorf("%s: matcher %q: %w", event, group.Matcher, err)
				}
				if !ok {
					continue
				}
			}
			for _, h := range group.Hooks {
				argv, err := split(strings.ReplaceAll(h.Command, "${CLAUDE_PROJECT_DIR}", root))
				if err != nil {
					return nil, nil, fmt.Errorf("%s: %w", event, err)
				}
				if len(argv) == 0 {
					return nil, nil, fmt.Errorf("%s: a hook names no command", event)
				}
				steps = append(steps, benchhooks.Step{Argv: argv})
			}
		}
		if len(steps) == 0 {
			continue
		}
		name := event
		if toolEvent(event) {
			name = fmt.Sprintf("%s (Edit on %s)", event, filepath.Base(file))
		}
		mode := "single"
		if len(steps) > 1 {
			mode = "par"
		}
		payloadName := "payload-" + event + ".json"
		cases = append(cases, benchhooks.Case{Name: name, Dir: root, Stdin: dir + "/" + payloadName, Mode: mode, Steps: steps})
		payloads = append(payloads, Payload{Name: payloadName, Data: payload(event, root, file)})
	}
	if len(cases) == 0 {
		return nil, nil, fmt.Errorf("settings: no hook of any event applies to an edit")
	}
	return cases, payloads, nil
}

// matchesTool applies a hook matcher to a tool name: an empty matcher and "*"
// match every tool, anything else is a regular expression that must match the
// whole name.
func matchesTool(matcher, tool string) (bool, error) {
	if matcher == "" || matcher == "*" {
		return true, nil
	}
	re, err := regexp.Compile("^(?:" + matcher + ")$")
	if err != nil {
		return false, err
	}
	return re.MatchString(tool), nil
}

// payload is the stdin a host hands the hook of an event: the event, the
// working directory and, for the tool events, an edit of file. It has no
// error to return: the document holds only strings, booleans and maps of
// them, which encoding/json always marshals.
//
// Every payload names a session, as a host's does: the hooks that keep
// state per session (stop, subagent-start, subagent-stop of loomux and of
// ultraloom) reject a payload without session_id before doing any work, and
// the subagent hooks also need agent_id. The ids are fixed and plainly
// synthetic, so the state they leave is recognisable. transcript_path
// points at a file that does not exist: no hook reads it today, and one
// that starts to must not read a real transcript.
func payload(event, root, file string) []byte {
	doc := map[string]any{
		"hook_event_name": event, "cwd": root,
		"session_id": "loomux-bench", "transcript_path": root + "/.loomux-bench-no-transcript.jsonl",
	}
	edit := map[string]any{"file_path": file, "old_string": "a", "new_string": "b"}
	switch event {
	case "PreToolUse":
		doc["tool_name"], doc["tool_input"] = "Edit", edit
	case "PostToolUse":
		doc["tool_name"], doc["tool_input"], doc["tool_response"] = "Edit", edit, map[string]any{"success": true}
	case "SessionStart":
		doc["source"] = "startup"
	case "Stop":
		doc["stop_hook_active"] = false
	case "SubagentStart", "SubagentStop":
		doc["agent_id"], doc["agent_type"] = "loomux-bench-agent", "general-purpose"
	}
	data, _ := json.Marshal(doc)
	return data
}

// split cuts a command line into arguments. It is no shell: it knows the
// forms hook configurations use -- words separated by blanks, single and
// double quotes, and \" or \\ inside double quotes -- and nothing else. An
// operator (;, |, &&, a redirection) is a word like any other, and nothing
// is expanded or globbed.
func split(s string) ([]string, error) {
	var args []string
	var cur strings.Builder
	inWord := false
	var quote rune
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case quote != 0:
			switch {
			case r == quote:
				quote = 0
			case r == '\\' && quote == '"' && i+1 < len(rs) && (rs[i+1] == '"' || rs[i+1] == '\\'):
				i++
				cur.WriteRune(rs[i])
			default:
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote, inWord = r, true
		case r == ' ' || r == '\t':
			if inWord {
				args = append(args, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated %c quote in %q", quote, s)
	}
	if inWord {
		args = append(args, cur.String())
	}
	return args, nil
}
