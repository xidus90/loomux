package hosts_test

import (
	"bytes"
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
			path := filepath.Join("..", "..", "testdata", "cases", "2c-payloads", filename)
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

// TestWriteAntigravityContextFailsClosed verifies that writeAntigravityContext
// always returns ErrNoAdapter until context emission for Antigravity is implemented.
func TestWriteAntigravityContextFailsClosed(t *testing.T) {
	var out bytes.Buffer
	writeErr := hosts.WriteContext(hosts.HostAntigravity, "PreInvocation", &out, []string{"line"})
	if !errors.Is(writeErr, hosts.ErrNoAdapter) {
		t.Fatalf("expected ErrNoAdapter, got %v", writeErr)
	}
	if out.Len() != 0 {
		t.Fatalf("expected nothing written, got %q", out.String())
	}

	if err := hosts.WriteContext(hosts.HostAntigravity, "PreInvocation", &out, nil); !errors.Is(err, hosts.ErrNoAdapter) {
		t.Fatalf("expected ErrNoAdapter for empty lines, got %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected nothing written, got %q", out.String())
	}
}
