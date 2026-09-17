package main

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// initLine is one line of GODEBUG=inittrace=1: the package and the number
// of allocations its package variables and init functions made.
var initLine = regexp.MustCompile(`^init (\S+) @.* (\d+) allocs$`)

// maxInitAllocs is the start-time rule in a number. Resolving the local
// time zone in a package variable cost 1,673 allocations and ~19 ms per
// process on Windows (docs/en/benchmarks.md, 2026-09-17 14:16); the largest
// init of loomux itself makes 131. A package over the line does real work at start, on
// every hook, and should do it on first use instead.
const maxInitAllocs = 500

func TestStartDoesNoWorkInPackageInit(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	binary := filepath.Join(t.TempDir(), "loomux.exe")
	build := exec.Command("go", "build", "-o", binary, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	run := exec.Command(binary, "version")
	run.Env = append(os.Environ(), "GODEBUG=inittrace=1")
	var trace bytes.Buffer
	run.Stderr = &trace
	if err := run.Run(); err != nil {
		t.Fatalf("loomux version: %v\n%s", err, trace.String())
	}
	seen := 0
	lines := bufio.NewScanner(&trace)
	for lines.Scan() {
		m := initLine.FindStringSubmatch(lines.Text())
		if m == nil {
			continue
		}
		seen++
		allocs, _ := strconv.Atoi(m[2])
		if allocs > maxInitAllocs {
			t.Errorf("init of %s makes %d allocations at start, more than %d", m[1], allocs, maxInitAllocs)
		}
	}
	if seen == 0 {
		t.Fatalf("no init lines in the trace:\n%s", trace.String())
	}
}
