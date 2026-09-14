package hosts_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
)

// The seam, and the whole of it. Codex is not installed on the development
// machine and its hook contract could not be computed from here, so there is
// no adapter -- only the promise that an unanswerable host is refused rather
// than answered in a shape somebody guessed.
//
// Reading refuses, and so does writing: a hook that read nothing must not go
// on to report something. ErrNoAdapter and not just any error, because the
// caller has to tell "this host has no adapter" apart from "this host does not
// exist".
func TestCodexFailsClosed(t *testing.T) {
	if _, err := hosts.Read(hosts.HostCodex, strings.NewReader(`{"session_id": "x"}`)); !errors.Is(err, hosts.ErrNoAdapter) {
		t.Errorf("the codex seam refuses to read with ErrNoAdapter, got %v", err)
	}
	var out bytes.Buffer
	if err := hosts.WriteContext(hosts.HostCodex, &out, []string{"a"}); !errors.Is(err, hosts.ErrNoAdapter) {
		t.Errorf("the codex seam refuses to write with ErrNoAdapter, got %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("a refusal writes nothing, got %q", out.String())
	}

	// And an empty call refuses too: the refusal does not wait for content.
	// "Nothing to say writes nothing" is the Claude arm's rule, so a seam that
	// answered nil to no lines would report success for a host it cannot
	// write to at all.
	if err := hosts.WriteContext(hosts.HostCodex, &out, nil); !errors.Is(err, hosts.ErrNoAdapter) {
		t.Errorf("the codex seam refuses an empty write with ErrNoAdapter, got %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("a refusal writes nothing, got %q", out.String())
	}
}

// Antigravity is a seam on both sides, and for two different reasons.
//
// Reading: no payload was ever recorded. The measurement of 2026-09-10 (agy
// 1.1.24) found that in print mode neither Stop nor PreInvocation fires at
// all, across three configurations, so nothing was captured to read a shape
// from. Forwarding to the Claude adapter would be a guess that looks like an
// implementation.
//
// Writing: whether PreInvocation can put context in front of the model is
// unanswered for the same reason -- the event did not fire.
func TestAntigravityFailsClosed(t *testing.T) {
	_, readErr := hosts.Read(hosts.HostAntigravity, strings.NewReader(`{"session_id": "x"}`))
	if !errors.Is(readErr, hosts.ErrNoAdapter) {
		t.Fatalf("the antigravity read arm refuses with ErrNoAdapter, got %v", readErr)
	}
	if !strings.Contains(readErr.Error(), "unmeasured") {
		t.Fatalf("the refusal says the payload shape is unmeasured, got %v", readErr)
	}

	var out bytes.Buffer
	writeErr := hosts.WriteContext(hosts.HostAntigravity, &out, []string{"a"})
	if !errors.Is(writeErr, hosts.ErrNoAdapter) {
		t.Fatalf("the antigravity context path is not built yet and must say so, got %v", writeErr)
	}
	if !strings.Contains(writeErr.Error(), "PreInvocation") {
		t.Fatalf("the refusal names what is missing, got %v", writeErr)
	}
	if out.Len() != 0 {
		t.Fatalf("a refusal writes nothing, got %q", out.String())
	}

	// The empty call refuses as well. Silence for no lines is the Claude arm's
	// rule and not a property of the answer, so this arm owes the caller
	// ErrNoAdapter whether there is anything to write or not.
	if err := hosts.WriteContext(hosts.HostAntigravity, &out, nil); !errors.Is(err, hosts.ErrNoAdapter) {
		t.Fatalf("the antigravity arm refuses an empty write with ErrNoAdapter, got %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("a refusal writes nothing, got %q", out.String())
	}
}
