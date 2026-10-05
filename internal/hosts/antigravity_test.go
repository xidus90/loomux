package hosts_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
)

// TestReadAntigravityFromTheMeasuredPayloads verifies that all four measured
// Antigravity hook payloads decode cleanly and yield the expected SessionID.
func TestReadAntigravityFromTheMeasuredPayloads(t *testing.T) {
	cases := []string{
		"agy-stop.json",
		"agy-inv.json",
		"agy-pre.json",
		"agy-post.json",
	}

	for _, filename := range cases {
		t.Run(filename, func(t *testing.T) {
			path := filepath.Join("testdata", filename)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading payload file %s: %v", filename, err)
			}

			got, err := hosts.Read(hosts.HostAntigravity, bytes.NewReader(data))
			if err != nil {
				t.Fatalf("unexpected error reading %s: %v", filename, err)
			}

			if got.SessionID != "s1" {
				t.Errorf("expected session_id %q, got %q", "s1", got.SessionID)
			}
			if got.AgentID != "" {
				t.Errorf("expected empty agent_id for Antigravity, got %q", got.AgentID)
			}
			if got.AgentType != "" {
				t.Errorf("expected empty agent_type for Antigravity, got %q", got.AgentType)
			}
		})
	}
}

// TestReadAntigravitySnakeCase verifies that conversation_id is accepted when
// conversationId is absent or empty.
func TestReadAntigravitySnakeCase(t *testing.T) {
	for _, body := range []string{
		`{"conversation_id": "s2"}`,
		`{"conversationId": "", "conversation_id": "s3"}`,
	} {
		got, err := hosts.Read(hosts.HostAntigravity, strings.NewReader(body))
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", body, err)
		}
		if got.SessionID == "" {
			t.Errorf("expected non-empty SessionID for %s, got empty", body)
		}
	}
}

// TestReadAntigravityRefusesWhatIsNotAPayload ensures non-JSON and non-object
// payloads are refused with appropriate errors.
func TestReadAntigravityRefusesWhatIsNotAPayload(t *testing.T) {
	for _, testCase := range []struct {
		body   string
		reason string
	}{
		{"", "stdin is not JSON"},
		{"not json", "stdin is not JSON"},
		{"[1, 2]", "a hook payload is an object"},
		{`"a string"`, "a hook payload is an object"},
		{"42", "a hook payload is an object"},
		{"null", "a hook payload is an object"},
	} {
		_, err := hosts.Read(hosts.HostAntigravity, strings.NewReader(testCase.body))
		if err == nil {
			t.Errorf("expected a refusal for %q", testCase.body)
			continue
		}
		if !strings.Contains(err.Error(), testCase.reason) {
			t.Errorf("refusal for %q should mention %q, got %v", testCase.body, testCase.reason, err)
		}
	}
}

// TestReadAntigravityReportsAFailingStdin verifies that I/O read errors are propagated.
func TestReadAntigravityReportsAFailingStdin(t *testing.T) {
	boom := errors.New("broken stdin")
	_, err := hosts.Read(hosts.HostAntigravity, brokenReader{err: boom})
	if err == nil {
		t.Fatal("expected an error on failing stdin")
	}
	if !errors.Is(err, boom) {
		t.Errorf("expected wrapped error %v, got %v", boom, err)
	}
	if !strings.Contains(err.Error(), "reading stdin") {
		t.Errorf("expected 'reading stdin' in message, got %v", err)
	}
}

// TestReadAntigravityAcceptsAMistypedOrMissingSessionID verifies that missing
// or non-string session IDs decode as empty string without error.
func TestReadAntigravityAcceptsAMistypedOrMissingSessionID(t *testing.T) {
	for _, body := range []string{
		`{"conversationId": 42}`,
		`{"conversationId": null}`,
		`{}`,
	} {
		got, err := hosts.Read(hosts.HostAntigravity, strings.NewReader(body))
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", body, err)
		}
		if got.SessionID != "" {
			t.Errorf("expected empty SessionID for %s, got %q", body, got.SessionID)
		}
	}
}

// agy sends invocationNum as a JSON number, 0 on the first model call;
// protojson writes a 64-bit counter as a decimal string, so that spelling
// counts too. Anything else, and a string that is no whole number or does not
// fit 64 bits, reads as the first invocation.
func TestReadAntigravityReadsTheInvocationAsANumberOrADecimalString(t *testing.T) {
	for body, repeat := range map[string]bool{
		`{"conversationId":"s1","invocationNum":1}`:                      true,
		`{"conversationId":"s1","invocationNum":"1"}`:                    true,
		`{"conversationId":"s1","invocationNum":2}`:                      true,
		`{"conversationId":"s1","invocationNum":"2"}`:                    true,
		`{"conversationId":"s1","invocationNum":"12"}`:                   true,
		`{"conversationId":"s1","invocationNum":0}`:                      false,
		`{"conversationId":"s1","invocationNum":"0"}`:                    false,
		`{"conversationId":"s1","invocationNum":"x"}`:                    false,
		`{"conversationId":"s1","invocationNum":"2.0"}`:                  false,
		`{"conversationId":"s1","invocationNum":""}`:                     false,
		`{"conversationId":"s1","invocationNum":"99999999999999999999"}`: false,
		`{"conversationId":"s1","invocationNum":true}`:                   false,
		`{"conversationId":"s1","invocationNum":null}`:                   false,
		`{"conversationId":"s1","invocationNum":[2]}`:                    false,
		`{"conversationId":"s1"}`:                                        false,
	} {
		got, err := hosts.Read(hosts.HostAntigravity, strings.NewReader(body))
		if err != nil {
			t.Fatalf("%s: %v", body, err)
		}
		if got.Repeat != repeat {
			t.Errorf("%s: Repeat = %v, want %v", body, got.Repeat, repeat)
		}
	}
}

// TestWriteAntigravityContext verifies that writeAntigravityContext encodes lines
// as protojson injectSteps with ephemeralMessage.
func TestWriteAntigravityContext(t *testing.T) {
	var buf bytes.Buffer
	lines := []string{"hello", "world"}
	if err := hosts.WriteContext(hosts.HostAntigravity, "PreInvocation", &buf, lines); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload struct {
		InjectSteps []struct {
			EphemeralMessage string `json:"ephemeralMessage"`
		} `json:"injectSteps"`
	}
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("stdout is not valid JSON: %v, raw: %s", err, buf.String())
	}
	if len(payload.InjectSteps) != 1 {
		t.Fatalf("expected 1 injectStep, got %d", len(payload.InjectSteps))
	}
	if got := payload.InjectSteps[0].EphemeralMessage; got != "hello\nworld" {
		t.Errorf("expected %q, got %q", "hello\nworld", got)
	}
}

// TestWriteAntigravityContextEmpty verifies that an empty or nil slice of lines
// writes nothing and returns nil.
func TestWriteAntigravityContextEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := hosts.WriteContext(hosts.HostAntigravity, "PreInvocation", &buf, nil); err != nil {
		t.Fatalf("unexpected error on nil: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty buffer, got %q", buf.String())
	}

	if err := hosts.WriteContext(hosts.HostAntigravity, "PreInvocation", &buf, []string{}); err != nil {
		t.Fatalf("unexpected error on empty: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty buffer, got %q", buf.String())
	}
}

// TestWriteAntigravityContextFailingWriter verifies that errors from a broken
// writer are returned wrapped.
func TestWriteAntigravityContextFailingWriter(t *testing.T) {
	boom := errors.New("write failed")
	err := hosts.WriteContext(hosts.HostAntigravity, "PreInvocation", brokenWriter{err: boom}, []string{"line"})
	if err == nil || !errors.Is(err, boom) {
		t.Fatalf("expected wrapped error %v, got %v", boom, err)
	}
}
