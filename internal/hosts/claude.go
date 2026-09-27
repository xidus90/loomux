package hosts

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// errNotAnObject is payload.py's second refusal, word for word. Named because
// two paths reach it: the decode that fails on an array, a string or a number,
// and the one that succeeds on JSON null.
var errNotAnObject = errors.New("a hook payload is an object")

// claudeAnswer is the envelope Claude Code reads a hook's added context from,
// for whichever event hookEventName names.
type claudeAnswer struct {
	HookSpecificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	} `json:"hookSpecificOutput"`
}

// readClaude is payload.py's `read`, narrowed to the fields in use.
//
// It refuses exactly what that module refuses and nothing more: stdin that is
// not JSON, and JSON that is not an object. Both refusals are worded the way
// payload.py words them, so the two hooks say the same thing for as long as
// both exist.
//
// The fields are read by type assertion rather than decoded into a struct, so
// a value of the wrong type reads as absent instead of as damage. That is not
// laxity, it is where the decision belongs: payload.py's whole job is "is this
// an object", and what counts as a usable session id is decided one layer up,
// by `recordBase` in internal/hooks, which files nothing when there is no usable id
// -- as `_record_base` in session_start.py (fa3dd38):52 did, returning without
// a word when the id was not a string. A struct decode would refuse
// `{"session_id": 5}` and exit 1 where the Python hook exited 0.
func readClaude(r io.Reader) (Payload, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return Payload{}, fmt.Errorf("reading stdin: %w", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		// A decode into a map tells the two cases apart by error type: bad
		// syntax is a *json.SyntaxError, while an array, a string or a number
		// is a *json.UnmarshalTypeError. Only the first is "not JSON".
		var wrongType *json.UnmarshalTypeError
		if errors.As(err, &wrongType) {
			return Payload{}, errNotAnObject
		}
		return Payload{}, fmt.Errorf("stdin is not JSON: %w", err)
	}
	if payload == nil {
		// JSON null decodes into a nil map and reports no error, and null is
		// no more an object than an array is.
		return Payload{}, errNotAnObject
	}
	event, _ := payload["hook_event_name"].(string)
	sessionID, _ := payload["session_id"].(string)
	agentID, _ := payload["agent_id"].(string)
	agentType, _ := payload["agent_type"].(string)
	return Payload{Event: event, SessionID: sessionID, AgentID: agentID, AgentType: agentType}, nil
}

// writeClaudeContext puts lines where the model will read them, as the
// answer to event: it serves any event the caller names, today SessionStart
// and PostToolUse.
//
// `hookSpecificOutput.additionalContext` is the field Claude Code documents
// for a hook's added context, and the field the superpowers port table
// names for this harness (superpowers/6.3.0/docs/porting-to-a-new-harness.md,
// harness table at :788). What plain stdout from such a hook does instead
// could not be measured from inside this repository, so nothing is claimed
// about it here.
//
// The encoder writes straight into w and its escaping is turned off: a context
// line is prose, and a `>` in it belongs in the transcript as itself. The only
// failure it can report is w's, because the document holds nothing but
// strings.
func writeClaudeContext(w io.Writer, event string, lines []string) error {
	var answer claudeAnswer
	answer.HookSpecificOutput.HookEventName = event
	answer.HookSpecificOutput.AdditionalContext = strings.Join(lines, "\n")

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(answer); err != nil {
		return fmt.Errorf("writing the hook answer: %w", err)
	}
	return nil
}
