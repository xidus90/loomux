package hosts_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
)

func answer(host hosts.Host, event string, code int, out, reason string) (int, string) {
	var w strings.Builder
	got := hosts.Answer(host, event, &w, code, []byte(out), reason)
	return got, w.String()
}

// Claude Code and Codex get exactly what the hook wrote, with its code.
func TestAnswerPassesClaudeAndCodexThrough(t *testing.T) {
	for _, host := range []hosts.Host{hosts.HostClaude, hosts.HostCodex} {
		for _, code := range []int{0, 1, 2} {
			got, out := answer(host, "stop", code, "{}\n", "red")
			if got != code || out != "{}\n" {
				t.Fatalf("[%s %d] code %d, out %q", host, code, got, out)
			}
		}
	}
}

// pre-tool-use keeps its exit 2 on Antigravity: that is what refuses the call.
func TestAnswerKeepsTheRefusalOfAntigravitysPreToolUse(t *testing.T) {
	got, out := answer(hosts.HostAntigravity, "pre-tool-use", 2, `{"decision":"deny"}`, "no")
	if got != 2 || out != `{"decision":"deny"}` {
		t.Fatalf("code %d, out %q", got, out)
	}
}

// A held stop becomes a continue decision carrying the reason, with exit 0.
func TestAnswerTurnsAntigravitysHeldStopIntoContinue(t *testing.T) {
	got, out := answer(hosts.HostAntigravity, "stop", 2, "", "loomux hook stop: go vet: red <b>\n")
	if got != 0 || out != `{"decision":"continue","reason":"loomux hook stop: go vet: red <b>"}`+"\n" {
		t.Fatalf("code %d, out %q", got, out)
	}
}

// A red post-edit lane keeps its exit 2, which agy hands the model as a
// warning, and the Claude-shaped stdout beside it is dropped.
func TestAnswerKeepsAntigravitysRedEdit(t *testing.T) {
	got, out := answer(hosts.HostAntigravity, "post-tool-use", 2, `{"hookSpecificOutput":{}}`, "gofmt: red\n")
	if got != 2 || out != "" {
		t.Fatalf("code %d, out %q", got, out)
	}
}

// A pass and a hook that could not judge both end with 0 and keep stdout:
// session-start's context is on it.
func TestAnswerEndsAntigravitysOtherCodesWithZero(t *testing.T) {
	for _, code := range []int{0, 1} {
		got, out := answer(hosts.HostAntigravity, "session-start", code, `{"injectSteps":[]}`, "boom")
		if got != 0 || out != `{"injectSteps":[]}` {
			t.Fatalf("[%d] code %d, out %q", code, got, out)
		}
	}
}

// A passed or unjudged edit on Antigravity ends with 0 and drops the
// Claude-shaped notices.
func TestAnswerDropsAntigravitysPostEditNotices(t *testing.T) {
	for _, code := range []int{0, 1} {
		got, out := answer(hosts.HostAntigravity, "post-tool-use", code, `{"hookSpecificOutput":{}}`, "")
		if got != 0 || out != "" {
			t.Fatalf("[%d] code %d, out %q", code, got, out)
		}
	}
}

// A decision that cannot be written ends as a failed command, the nearest
// thing to a hold; a pass-through that cannot be written keeps its code.
func TestAnswerOnAStdoutThatFails(t *testing.T) {
	broken := brokenWriter{err: errors.New("closed")}
	if got := hosts.Answer(hosts.HostAntigravity, "stop", broken, 2, nil, "red"); got != 2 {
		t.Fatalf("held stop: code %d", got)
	}
	if got := hosts.Answer(hosts.HostAntigravity, "stop", broken, 0, []byte("x"), ""); got != 0 {
		t.Fatalf("pass: code %d", got)
	}
	if got := hosts.Answer(hosts.HostClaude, "stop", broken, 2, []byte("x"), ""); got != 2 {
		t.Fatalf("claude: code %d", got)
	}
}
