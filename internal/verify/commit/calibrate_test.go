package commit_test

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/verify/commit"
)

var sampleMessages = []string{
	"Let the gate run one profile",
	"Rename the page to der-alte-fall.md",
	`The page says "der Bericht und das Ergebnis"`,
	"Das Ergebnis und der Bericht fehlen vollstaendig",
}

func TestCalibrateThresholds(t *testing.T) {
	result := commit.Calibrate(sampleMessages, "en", []int{1, 2, 3}, nil)

	// Threshold 2 refuses only message 3
	refused2 := result[2]
	if len(refused2) != 1 || refused2[0] != 3 {
		t.Errorf("expected only message 3 refused at threshold 2, got: %v", refused2)
	}

	// Higher thresholds refuse a subset
	if len(result[3]) > len(result[2]) || len(result[2]) > len(result[1]) {
		t.Errorf("expected monotone decrease in refusals: %+v", result)
	}
}

func TestCalibrateWithAllow(t *testing.T) {
	messages := []string{"Das Ergebnis und der Bericht fehlen", "Ein Bericht fehlt und das Problem"}
	allow := []*regexp.Regexp{regexp.MustCompile(`^Das Ergebnis`)}

	resWithout := commit.Calibrate(messages, "en", []int{1, 2}, nil)
	if len(resWithout[2]) != 2 {
		t.Errorf("expected 2 refused without allow, got %v", resWithout[2])
	}

	resWith := commit.Calibrate(messages, "en", []int{1, 2}, allow)
	if len(resWith[2]) != 1 || resWith[2][0] != 1 {
		t.Errorf("expected message 0 exempted by allow, got %v", resWith[2])
	}
}

func TestCalibrateRender(t *testing.T) {
	messages := append(sampleMessages, "Ein Bericht fehlt und der Fehler")
	var buf bytes.Buffer
	commit.Render(messages, "en", []int{1, 2}, &buf, nil)
	out := buf.String()

	if !strings.Contains(out, "5 messages, checked as en") {
		t.Errorf("expected header in output: %s", out)
	}
	if !strings.Contains(out, "  threshold 1: 2 refused") {
		t.Errorf("expected threshold 1 line, got: %s", out)
	}
	if !strings.Contains(out, "  threshold 2: 2 refused") {
		t.Errorf("expected threshold 2 line, got: %s", out)
	}
	if !strings.Contains(out, "    #4  Das Ergebnis und der Bericht fehlen vollstaendig") {
		t.Errorf("expected entry #4, got: %s", out)
	}
	if !strings.Contains(out, "    #5  Ein Bericht fehlt und der Fehler") {
		t.Errorf("expected entry #5, got: %s", out)
	}
}

func TestSubject(t *testing.T) {
	if s := commit.Subject(""); s != "" {
		t.Errorf("expected empty for empty string, got %q", s)
	}
	if s := commit.Subject("\n\n  \nSubject line\nBody"); s != "Subject line" {
		t.Errorf("expected skipped blank lines, got %q", s)
	}
}

func TestReadMessages(t *testing.T) {
	origRunner := commit.GitRunner
	defer func() { commit.GitRunner = origRunner }()

	// 1. Success case with NUL separator
	commit.GitRunner = func(dir string, timeout time.Duration, argv ...string) (child.Result, error) {
		return child.Result{
			Code:   0,
			Stdout: "Second commit\n\nWith body\x00First commit\n\x00\x00",
		}, nil
	}
	msgs, err := commit.ReadMessages(".", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 || msgs[0] != "Second commit\n\nWith body" || msgs[1] != "First commit\n" {
		t.Errorf("unexpected messages: %v", msgs)
	}

	// 2. Timeout case
	commit.GitRunner = func(dir string, timeout time.Duration, argv ...string) (child.Result, error) {
		return child.Result{TimedOut: true}, nil
	}
	_, err = commit.ReadMessages(".", 2)
	if err == nil || !strings.Contains(err.Error(), "took longer than") {
		t.Errorf("expected timeout error, got: %v", err)
	}

	// 3. Process error with stderr
	commit.GitRunner = func(dir string, timeout time.Duration, argv ...string) (child.Result, error) {
		return child.Result{Code: 128, Stderr: "fatal: not a git repository"}, nil
	}
	_, err = commit.ReadMessages(".", 2)
	if err == nil || !strings.Contains(err.Error(), "fatal: not a git repository") {
		t.Errorf("expected stderr in error, got: %v", err)
	}

	// 4. Process error with empty stderr
	commit.GitRunner = func(dir string, timeout time.Duration, argv ...string) (child.Result, error) {
		return child.Result{Code: 1}, nil
	}
	_, err = commit.ReadMessages(".", 2)
	if err == nil || !strings.Contains(err.Error(), "git exited 1") {
		t.Errorf("expected exit code in error, got: %v", err)
	}

	// 5. Runner error
	commit.GitRunner = func(dir string, timeout time.Duration, argv ...string) (child.Result, error) {
		return child.Result{}, errors.New("exec failed")
	}
	_, err = commit.ReadMessages(".", 2)
	if err == nil || !strings.Contains(err.Error(), "exec failed") {
		t.Errorf("expected runner error, got: %v", err)
	}
}

func TestDefaultGitRunnerReportsAMissingGit(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := commit.ReadMessages(t.TempDir(), 1)
	if err == nil || strings.Contains(err.Error(), "git exited") {
		t.Errorf("expected the start failure itself, got: %v", err)
	}
}

func TestDefaultGitRunner(t *testing.T) {
	// Call defaultGitRunner in an isolated temp dir to exercise the real seam
	dir := t.TempDir()
	msgs, err := commit.ReadMessages(dir, 1)
	if err == nil {
		t.Errorf("expected error in non-repo dir, got msgs: %v", msgs)
	}
}
