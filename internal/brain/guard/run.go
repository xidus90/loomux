package guard

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// blockingExit is the only code that stops a tool call. The host blocks on
// 2 whatever stdout carries, reads 1 as a non-blocking error and runs the
// tool anyway, and on 0 falls back to whatever decision it can parse off
// stdout. Which is why no path here reaches 1, and why a refusal reaches 2
// rather than trusting the envelope to be read.
const blockingExit = 2

// jsonNumber is the decoded shape of a JSON number kept as its own text,
// so that `pythonTypeName` can tell an int from a float the way Python
// does. JSON has one number type; Python has two.
type jsonNumber = json.Number

const unreadable = "the wiki guard cannot read the hook payload, so it refuses"

// Run reads the hook payload, writes the answer, and leaves with 0 or 2
// -- never 1.
//
// A refusal leaves with 2 and says why on both channels. It used to leave
// with 0 and rely on the host reading the envelope off stdout; measured on
// this host on 2026-09-06, it does not, and a probe file landed under
// `10 Rohquellen` while the refusal sat unread. Exit 2 blocks whatever
// stdout says, so that is where the decision lives now.
//
// Both channels all the same, because they are read under different
// conditions: the host prefers `permissionDecisionReason` out of the
// envelope and falls back to stderr when the JSON does not parse or does
// not arrive. Neither can be checked from in here, so both are written and
// neither is required to succeed -- the exit code carries the refusal even
// when both are gone.
//
// So nothing may escape here, the edges included. The decision is reached
// inside one recover, and *writing* it sits inside a second: the answer
// goes down a pipe that can be gone by the time it is written.
//
// The envelope is rendered once and written in a single call rather than
// streamed: a channel that tears mid-render leaves the host half an
// object to parse, and half a deny is not a deny. A short write counts as
// a failure for the same reason -- an unchecked Go `Write` would report
// success having delivered half.
func Run(stdin io.Reader, stdout, stderr io.Writer, stateDir string) int {
	reason, refuse := answer(stdin, stateDir)
	if !refuse {
		return 0
	}
	return Refuse(stdout, stderr, reason)
}

// Refuse writes one refusal on both channels and answers the blocking
// exit. Exported so that a hook deciding more than this barrier refuses in
// the same shape.
func Refuse(stdout, stderr io.Writer, reason string) int {
	envelope := denyEnvelope(reason)
	// The result is deliberately unchecked: a short or failing write used
	// to be the only thing that reached the blocking exit, and now every
	// refusal does. There is nothing left to fall back to.
	_, _ = stdout.Write([]byte(envelope))
	lastWord(stderr, reason)
	return blockingExit
}

// answer reaches the decision with every way out of it caught. A panic
// anywhere below would otherwise end the process with a code the host
// reads as "carry on", which is the one outcome a write barrier may never
// produce by accident.
func answer(stdin io.Reader, stateDir string) (reason string, refuse bool) {
	defer func() {
		if broke := recover(); broke != nil {
			reason = fmt.Sprintf(
				"the wiki guard broke down, so it refuses: %v", broke)
			refuse = true
		}
	}()
	data, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Sprintf(
			"the wiki guard broke down, so it refuses: %v", err), true
	}
	payload, err := decodeOnly(data)
	if err != nil {
		return fmt.Sprintf("%s: %v", unreadable, err), true
	}
	object, ok := payload.(map[string]any)
	if !ok {
		return fmt.Sprintf("%s: expected an object, found %s", unreadable,
			pythonTypeName(payload)), true
	}
	return Decide(object, stateDir)
}

// decodeOnly reads exactly one JSON value and refuses anything after it.
// `json.load` consumes the whole stream and calls trailing bytes an
// error; a decoder that stopped at the first value would judge half a
// payload and never mention the rest.
//
// `UseNumber` keeps a number as its text, which is what lets the type
// name below tell `3` from `3.0` the way Python does.
func decodeOnly(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var payload any
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	// Asked for a second value rather than for the bytes that are left:
	// the decoder skips the whitespace a payload may legitimately end
	// with, and reading the buffer instead would have an error arm that
	// a reader over a byte slice can never take.
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, errors.New("Extra data")
	}
	return payload, nil
}

// denyEnvelope renders the answer byte for byte as `json.dumps` did: keys
// in this order, `", "` and `": "` for separators, every character outside
// ASCII escaped, and no trailing newline. The shape is kept because a host
// that does read stdout should find a well-formed decision there; the
// blocking itself no longer depends on it.
//
// The top-level `decision` and `reason` are the older hook spelling and
// are kept beside `hookSpecificOutput` rather than dropped: nothing here
// has measured which of the two this host parses, and the exit code makes
// the question harmless either way.
func denyEnvelope(reason string) string {
	quoted := pythonJSONString(reason)
	return `{"decision": "deny", "reason": ` + quoted +
		`, "hookSpecificOutput": {"hookEventName": "PreToolUse", ` +
		`"permissionDecision": "deny", "permissionDecisionReason": ` +
		quoted + `}}`
}

// lastWord puts the reason where the host reads a blocking error from.
// Best effort by construction -- stderr may be as gone as stdout was, and
// the exit code carries the refusal either way. This only decides whether
// the person and the model are told why.
func lastWord(stderr io.Writer, reason string) {
	_, _ = fmt.Fprintln(stderr, reason)
}
