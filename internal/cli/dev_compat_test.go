package cli

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/compat"
)

// matrixJobs reads a per-stack matrix, which names every stack, and returns
// its jobs in stack order.
func matrixJobs(t *testing.T, out string) []compat.Job {
	t.Helper()
	var doc map[string]struct{ Include []compat.Job }
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("%q: %v", out, err)
	}
	if len(doc) != len(compat.ByStack(nil)) {
		t.Fatalf("%q: %d stacks, want every one", out, len(doc))
	}
	var jobs []compat.Job
	for _, stack := range slices.Sorted(maps.Keys(doc)) {
		jobs = append(jobs, doc[stack].Include...)
	}
	return jobs
}

func TestDevCompatNeedsMatrix(t *testing.T) {
	for _, args := range [][]string{{"dev", "compat"}, {"dev", "compat", "nope"}} {
		if code, _, errOut := run(args...); code != 2 || !strings.Contains(errOut, "subcommand required: matrix, run, scout") {
			t.Errorf("%v: code %d, err %q", args, code, errOut)
		}
	}
}

func TestDevCompatMatrix(t *testing.T) {
	list, _ := compat.Load()
	code, out, _ := run("dev", "compat", "matrix")
	if jobs := matrixJobs(t, out); code != 0 || len(jobs) != 0 || !strings.Contains(out, `"go":{"include":[]}`) {
		t.Fatalf("no selection: code %d, out %q", code, out)
	}
	code, out, _ = run("dev", "compat", "matrix", "--full")
	if jobs := matrixJobs(t, out); code != 0 || len(jobs) != len(compat.Matrix(list, compat.Selection{Full: true})) {
		t.Fatalf("full: code %d, %d jobs", code, len(jobs))
	}
	k := list[1].Key()
	code, out, _ = run("dev", "compat", "matrix", "--entry", k)
	want := `"` + list[1].Stack + `":{"include":[{"entry":"` + k + `","os":"linux"},{"entry":"` + k + `","os":"windows"}]}`
	if code != 0 || len(matrixJobs(t, out)) != 2 || !strings.Contains(out, want) {
		t.Fatalf("entry: code %d, out %q", code, out)
	}
}

func TestDevCompatMatrixRefuses(t *testing.T) {
	if code, _, errOut := run("dev", "compat", "matrix", "--entry", "nope"); code != 2 || !strings.Contains(errOut, `unknown entry "nope"`) {
		t.Errorf("unknown entry: code %d, err %q", code, errOut)
	}
	if code, _, _ := run("dev", "compat", "matrix", "--bogus"); code != 2 {
		t.Errorf("unknown flag: code %d", code)
	}
	if code, _, errOut := run("dev", "compat", "matrix", "--changed-since", "no-such-ref"); code != 1 || errOut == "" {
		t.Errorf("unknown ref: code %d, err %q", code, errOut)
	}
}

func TestDevCompatMatrixChangedSince(t *testing.T) {
	code, out, errOut := run("dev", "compat", "matrix", "--changed-since", "HEAD")
	if code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	matrixJobs(t, out)
}
