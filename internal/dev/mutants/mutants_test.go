package mutants

import (
	"path/filepath"
	"testing"
)

func TestMutantPrintsTheScriptsLine(t *testing.T) {
	m := Mutant{Family: "a3", Path: filepath.Join(t.TempDir(), "fixture.go.txt"), Line: 5, Was: "\tif a > 0 {", Now: "\tif a >= 0 {"}
	if got := m.String(); got != "(a3) fixture.go.txt:5  if a > 0 {  ->  if a >= 0 {" {
		t.Fatalf("%q", got)
	}
}

func TestClassifyReadsWhatGoTestPrinted(t *testing.T) {
	// Shortened from go test's own output (go1.27.0 windows/amd64, 2026-09-15).
	for _, c := range []struct {
		name     string
		printed  string
		passed   bool
		timedOut bool
		want     Outcome
	}{
		{"green", "ok  \texample.com/probe/p\t0.131s\n", true, false, Passed},
		{"red with a markdown heading in the message", "--- FAIL: TestSign (0.00s)\n    p_test.go:14: # brain\n        \n        Sign(1) = 0\nFAIL\nFAIL\texample.com/probe/p\t0.152s\nFAIL\n", false, false, Failed},
		{"compile error", "# example.com/probe/p [example.com/probe/p.test]\n.\\ov\\compile.go:4:2: declared and not used: x\nFAIL\texample.com/probe/p [build failed]\nFAIL\n", false, false, BuildFailed},
		{"vet error", "# example.com/probe/p\n# [example.com/probe/p]\n.\\ov\\vet.go:9:19: fmt.Sprintf format %d has arg \"x\" of wrong type string\nFAIL\texample.com/probe/p [build failed]\nFAIL\n", false, false, BuildFailed},
		{"package header behind other output", "FAIL\n# example.com/probe/p\n", false, false, BuildFailed},
		// The test binary's own -timeout ends the run with exit 1 like a
		// failure, but it is no answer from the suite: a timeout.
		{"test binary timeout", "panic: test timed out after 2s\n\trunning tests:\n\t\tTestSign (2s)\n\ngoroutine 8 [running]:\nFAIL\texample.com/probe/p\t2.150s\nFAIL\n", false, false, TimedOut},
		{"patience ran out", "", false, true, TimedOut},
	} {
		if got := classify(c.printed, c.passed, c.timedOut); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}
