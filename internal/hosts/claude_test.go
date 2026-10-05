package hosts_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
)

// brokenReader and brokenWriter stand for a stdin that dies mid-read and a
// stdout that is closed: both are real for a hook whose host goes away, and
// neither can be produced by a string or a buffer.
type brokenReader struct{ err error }

func (b brokenReader) Read([]byte) (int, error) { return 0, b.err }

type brokenWriter struct{ err error }

func (b brokenWriter) Write([]byte) (int, error) { return 0, b.err }

func TestReadClaude(t *testing.T) {
	body := `{"hook_event_name": "SessionStart", "session_id": "abc-123"}`

	got, err := hosts.Read(hosts.HostClaude, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	if got.Event != "SessionStart" || got.SessionID != "abc-123" {
		t.Fatalf("unexpected payload: %+v", got)
	}
}

// The reference's rule: stdin that is not a JSON object is not a hook payload,
// and the refusal names why. Checked against the reference --
// it raises "stdin is not JSON: ..." and "a hook payload is an object", and
// those two are told apart here as well.
func TestReadClaudeRefusesWhatIsNotAPayload(t *testing.T) {
	for _, testCase := range []struct {
		body   string
		reason string
	}{
		{"", "stdin is not JSON"},
		{"not json", "stdin is not JSON"},
		{"[1, 2]", "a hook payload is an object"},
		{`"a string"`, "a hook payload is an object"},
		{"42", "a hook payload is an object"},
		// JSON null is not an object either. It needs its own case because it
		// decodes into a nil map without an error, where every other non-object
		// above fails the decode outright.
		{"null", "a hook payload is an object"},
	} {
		_, err := hosts.Read(hosts.HostClaude, strings.NewReader(testCase.body))
		if err == nil {
			t.Errorf("expected a refusal for %q", testCase.body)
			continue
		}
		if !strings.Contains(err.Error(), testCase.reason) {
			t.Errorf("refusal for %q was %v, want it to name %q", testCase.body, err, testCase.reason)
		}
	}
}

// A stdin that fails mid-read is not a malformed payload, and the refusal says
// which of the two happened.
func TestReadClaudeReportsAFailingStdin(t *testing.T) {
	boom := errors.New("pipe closed")

	_, err := hosts.Read(hosts.HostClaude, brokenReader{err: boom})

	if !errors.Is(err, boom) {
		t.Fatalf("expected the reader's error, got %v", err)
	}
	if !strings.Contains(err.Error(), "reading stdin") {
		t.Fatalf("the refusal names what failed, got %v", err)
	}
}

// A session id of the wrong type reads as absent, not as damage. payload.py
// insists only that the payload is an object; what counts as a usable id is
// decided one layer up, by the `not isinstance(session_id, str)` line in
// `_record_base` (session_start.py (fa3dd38):52), which returned without a
// word. Refusing here would exit 1 where the Python hook exited 0.
func TestReadClaudeAcceptsAMistypedSessionID(t *testing.T) {
	got, err := hosts.Read(hosts.HostClaude, strings.NewReader(`{"hook_event_name": "SessionStart", "session_id": 5}`))
	if err != nil {
		t.Fatalf("a mistyped session id is the hook's business, not the adapter's: %v", err)
	}
	if got.SessionID != "" {
		t.Fatalf("expected an empty session id, got %q", got.SessionID)
	}
	if got.Event != "SessionStart" {
		t.Fatalf("the rest of the payload still reads: %+v", got)
	}
}

// The same for the event name: a non-string there is not the adapter's
// business either, and no hook has to be taught to distrust the type.
func TestReadClaudeAcceptsAMistypedEventName(t *testing.T) {
	got, err := hosts.Read(hosts.HostClaude, strings.NewReader(`{"hook_event_name": [1], "session_id": "abc"}`))
	if err != nil {
		t.Fatalf("a mistyped event name is not damage: %v", err)
	}
	if got.Event != "" || got.SessionID != "abc" {
		t.Fatalf("unexpected payload: %+v", got)
	}
}

// A payload without a session id still reads: `recordBase` in internal/hooks files
// no base then and says nothing, because there is nowhere to file it. That
// decision lives in the hook, so the adapter must not refuse here. Checked
// against _record_base in session_start.py (fa3dd38):43-59.
func TestReadClaudeAcceptsAMissingSessionID(t *testing.T) {
	got, err := hosts.Read(hosts.HostClaude, strings.NewReader(`{"hook_event_name": "SessionStart"}`))
	if err != nil {
		t.Fatalf("a missing session id is the hook's business, not the adapter's: %v", err)
	}
	if got.SessionID != "" {
		t.Fatalf("expected an empty session id, got %q", got.SessionID)
	}
}

// hookSpecificOutput.additionalContext is the field Claude Code documents for
// SessionStart context, so that is the field this writes. What plain stdout
// does instead could not be measured from here and is not claimed.
func TestWriteClaudeContext(t *testing.T) {
	var out bytes.Buffer

	if err := hosts.WriteContext(hosts.HostClaude, "SessionStart", &out, []string{"line one", "line two"}); err != nil {
		t.Fatal(err)
	}

	body := out.String()
	if !strings.Contains(body, `"hookEventName":"SessionStart"`) {
		t.Fatalf("the envelope names its event: %s", body)
	}
	if !strings.Contains(body, "line one\\nline two") {
		t.Fatalf("the lines arrive joined by a newline: %s", body)
	}
}

// A line is prose and not HTML: a greater-than sign in a context line has to
// arrive as itself and not as the unicode escape the encoder writes for it
// unless it is told otherwise.
func TestWriteClaudeContextLeavesProseAlone(t *testing.T) {
	var out bytes.Buffer

	if err := hosts.WriteContext(hosts.HostClaude, "SessionStart", &out, []string{"run 7 > gate open"}); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), "run 7 > gate open") {
		t.Fatalf("expected the line unescaped: %s", out.String())
	}
}

// Nothing to say is silence and not an empty envelope. What Claude Code does
// with an additionalContext of "" could not be measured from inside this
// repository, so nothing is claimed about it: writing no envelope needs no
// host-side justification.
func TestWriteClaudeContextOfNothingWritesNothing(t *testing.T) {
	var out bytes.Buffer

	if err := hosts.WriteContext(hosts.HostClaude, "SessionStart", &out, nil); err != nil {
		t.Fatal(err)
	}

	if out.Len() != 0 {
		t.Fatalf("expected no output, got %q", out.String())
	}
}

func TestWriteClaudeContextReportsAFailingStdout(t *testing.T) {
	boom := errors.New("stdout closed")

	err := hosts.WriteContext(hosts.HostClaude, "SessionStart", brokenWriter{err: boom}, []string{"a"})

	if !errors.Is(err, boom) {
		t.Fatalf("expected the writer's error, got %v", err)
	}
}

// The switch refuses a Host nobody parsed rather than falling through to one
// of the arms. Reachable only from inside the package's own type, which is
// exactly why it is tested: an unreachable branch would be an exclusion
// without a reason.
func TestAnUnparsedHostIsRefused(t *testing.T) {
	if _, err := hosts.Read(hosts.Host("nonsense"), strings.NewReader("{}")); err == nil {
		t.Error("Read refuses a host it does not know")
	}
	var out bytes.Buffer
	if err := hosts.WriteContext(hosts.Host("nonsense"), "SessionStart", &out, []string{"a"}); err == nil {
		t.Error("WriteContext refuses a host it does not know")
	}
	if out.Len() != 0 {
		t.Errorf("a refusal writes nothing, got %q", out.String())
	}
}

// A SubagentStart or SubagentStop names the subagent it fired for, and a
// mistyped id reads as absent like every other field of this adapter.
func TestReadClaudeTakesTheAgent(t *testing.T) {
	p, err := hosts.Read(hosts.HostClaude, strings.NewReader(`{"hook_event_name":"SubagentStop","session_id":"s","agent_id":"a","agent_type":"Explore"}`))
	if err != nil || p.AgentID != "a" || p.AgentType != "Explore" {
		t.Fatalf("%+v, %v", p, err)
	}
	p, _ = hosts.Read(hosts.HostClaude, strings.NewReader(`{"agent_id": 5}`))
	if p.AgentID != "" {
		t.Fatalf("a mistyped id reads as absent: %+v", p)
	}
}

// The envelope names the event the caller answers, not a fixed SessionStart:
// the same writer answers the other events from stage 2c on.
func TestWriteContextNamesTheEvent(t *testing.T) {
	// An event that is not SessionStart: with the old hard-coded string in
	// place this is the assertion that fails, which is what makes the test
	// hold the new behaviour rather than the old one.
	var out strings.Builder
	if err := hosts.WriteContext(hosts.HostClaude, "SubagentStop", &out, []string{"x"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"hookEventName":"SubagentStop"`) {
		t.Fatalf("%s", out.String())
	}
}
