package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
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

// treeSitterModule is the one exception to that rule, held to a ceiling of
// its own instead. Its core and its grammar runtime build their tables in
// init, and that work does run on every hook. It was accepted instead of a
// patched copy that loads on first use: the cost is deterministic, and every
// update of a fast-moving library would redo the patch.
const treeSitterModule = "github.com/odvcencio/gotreesitter"

// maxTreeSitterAllocs is what every package of treeSitterModule may allocate
// at start, all of them together. Measured on 2026-09-26 at v0.55.0: 11,291
// (the core 1,108, grammars/runtime 10,180, grammars/python 3); the binary's
// init sum rose by about 1.6 ms with them (median of five runs, 2.94 to 4.56
// ms, docs/en/benchmarks.md, 2026-09-26 17:25). The headroom is for the
// grammars later languages link; a new version or a new grammar that crosses
// the line is measured again before the line moves.
const maxTreeSitterAllocs = 20_000

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
	seen, findings := initFindings(trace.String())
	if seen == 0 {
		t.Fatalf("no init lines in the trace:\n%s", trace.String())
	}
	for _, f := range findings {
		t.Error(f)
	}
}

// initFindings reads an inittrace: how many init lines it holds, and one
// message for every package over maxInitAllocs and for treeSitterModule's
// packages when together they are over maxTreeSitterAllocs.
func initFindings(trace string) (int, []string) {
	seen, treeSitter := 0, 0
	var findings []string
	lines := bufio.NewScanner(strings.NewReader(trace))
	for lines.Scan() {
		m := initLine.FindStringSubmatch(lines.Text())
		if m == nil {
			continue
		}
		seen++
		allocs, _ := strconv.Atoi(m[2])
		switch {
		case m[1] == treeSitterModule || strings.HasPrefix(m[1], treeSitterModule+"/"):
			treeSitter += allocs
		case allocs > maxInitAllocs:
			findings = append(findings, fmt.Sprintf("init of %s makes %d allocations at start, more than %d", m[1], allocs, maxInitAllocs))
		}
	}
	if treeSitter > maxTreeSitterAllocs {
		findings = append(findings, fmt.Sprintf("init of %s makes %d allocations at start across its packages, more than %d", treeSitterModule, treeSitter, maxTreeSitterAllocs))
	}
	return seen, findings
}

func TestInitFindingsHoldTreeSitterToItsCeiling(t *testing.T) {
	line := func(pkg string, allocs int) string {
		return fmt.Sprintf("init %s @4.6 ms, 0.52 ms clock, 165648 bytes, %d allocs\n", pkg, allocs)
	}
	// Today's measurement: each tree-sitter package may pass the per-package
	// line, since only their sum counts, and a package of its own at the line
	// passes too.
	today := line(treeSitterModule, 1108) + line(treeSitterModule+"/grammars/runtime", 10180) +
		line(treeSitterModule+"/grammars/python", 3) + line("github.com/xidus90/loomux/internal/cli", maxInitAllocs) +
		"not an init line\n"
	if seen, findings := initFindings(today); seen != 4 || len(findings) != 0 {
		t.Errorf("initFindings(today) = %d, %q; want 4 lines and no finding", seen, findings)
	}

	// A grammar that pushes the sum over the ceiling fails with the sum and
	// the ceiling; a package that only shares the module's name as a prefix
	// is held to the per-package line.
	over := today + line(treeSitterModule+"/grammars/typescript", 8710) + line(treeSitterModule+"x", 501)
	_, findings := initFindings(over)
	want := []string{
		"init of " + treeSitterModule + "x makes 501 allocations at start, more than 500",
		"init of " + treeSitterModule + " makes 20001 allocations at start across its packages, more than 20000",
	}
	if !reflect.DeepEqual(findings, want) {
		t.Errorf("initFindings(over) = %q, want %q", findings, want)
	}
}
