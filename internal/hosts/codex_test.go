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
	if err := hosts.WriteContext(hosts.HostCodex, "SessionStart", &out, []string{"a"}); !errors.Is(err, hosts.ErrNoAdapter) {
		t.Errorf("the codex seam refuses to write with ErrNoAdapter, got %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("a refusal writes nothing, got %q", out.String())
	}

	// And an empty call refuses too: the refusal does not wait for content.
	// "Nothing to say writes nothing" is the Claude arm's rule, so a seam that
	// answered nil to no lines would report success for a host it cannot
	// write to at all.
	if err := hosts.WriteContext(hosts.HostCodex, "SessionStart", &out, nil); !errors.Is(err, hosts.ErrNoAdapter) {
		t.Errorf("the codex seam refuses an empty write with ErrNoAdapter, got %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("a refusal writes nothing, got %q", out.String())
	}
}
