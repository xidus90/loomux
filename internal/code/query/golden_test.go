package query_test

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/query"
	"github.com/xidus90/loomux/internal/gitenv"
)

var update = flag.Bool("update", false, "update golden files")

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copyDir: %v", err)
	}
}

func checkGolden(t *testing.T, goldenPath, actual string) {
	t.Helper()
	// Normalize CRLF to LF for deterministic comparison across platforms
	actual = strings.ReplaceAll(actual, "\r\n", "\n")
	if *update {
		if err := os.WriteFile(goldenPath, []byte(actual), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	expectedBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v (run with -update to generate)", err)
	}
	expected := strings.ReplaceAll(string(expectedBytes), "\r\n", "\n")
	if actual != expected {
		t.Errorf("golden mismatch for %s:\n--- got ---\n%s\n--- want ---\n%s", filepath.Base(goldenPath), actual, expected)
	}
}

func TestGolden(t *testing.T) {
	casesDir := filepath.Join("..", "..", "..", "testdata", "cases", "graph")
	repoSrc := filepath.Join(casesDir, "repo")

	root := t.TempDir()
	copyDir(t, repoSrc, root)

	// Build graph
	if _, _, err := query.Build(root, func(string) {}); err != nil {
		t.Fatalf("build: %v", err)
	}

	// 1. Callers
	callersAns, _, err := query.Callers(root, "Add", query.CallersOptions{
		Direction: blast.In,
		Depth:     1,
		NoRefresh: true,
	})
	if err != nil {
		t.Fatalf("callers: %v", err)
	}
	checkGolden(t, filepath.Join(casesDir, "callers.golden"), query.CallersReport(callersAns))

	// 2. Skeleton
	skelAns, _, err := query.Skeleton(root, "calc/calc.go", query.SkeletonOptions{
		NoRefresh: true,
	})
	if err != nil {
		t.Fatalf("skeleton: %v", err)
	}
	checkGolden(t, filepath.Join(casesDir, "skeleton.golden"), query.SkeletonReport(skelAns))

	// 3. Grep
	grepAns, _, err := query.Grep(root, "Add", query.GrepOptions{
		NoRefresh: true,
	})
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	checkGolden(t, filepath.Join(casesDir, "grep.golden"), query.GrepReport(grepAns))

	// 4. Map
	mapAns, _, err := query.Map(root, query.MapOptions{
		MaxDirs:   16,
		NoRefresh: true,
	})
	if err != nil {
		t.Fatalf("map: %v", err)
	}
	checkGolden(t, filepath.Join(casesDir, "map.golden"), query.MapReport(mapAns))

	// 5. Stats
	statsAns, err := query.GraphStats(root)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	// Normalise Wiring size as it depends on file system/newline encoding
	statsReport := query.StatsReport(statsAns)
	lines := strings.Split(strings.ReplaceAll(statsReport, "\r\n", "\n"), "\n")
	var normLines []string
	for _, l := range lines {
		if strings.HasPrefix(l, "  Wiring size:") {
			normLines = append(normLines, "  Wiring size:  <normalized>")
		} else {
			normLines = append(normLines, l)
		}
	}
	checkGolden(t, filepath.Join(casesDir, "stats.golden"), strings.Join(normLines, "\n"))
}

func TestGoldenBlast(t *testing.T) {
	casesDir := filepath.Join("..", "..", "..", "testdata", "cases", "graph")
	root := t.TempDir()
	copyDir(t, filepath.Join(casesDir, "repo"), root)
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"},
		{"config", "core.autocrlf", "false"}, {"config", "commit.gpgsign", "false"},
		{"add", "."}, {"commit", "-qm", "init"},
	} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = gitenv.Environ()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, _, err := query.Build(root, func(string) {}); err != nil {
		t.Fatalf("build: %v", err)
	}

	// Line 5 of calc/calc.go is Add's body.
	calc := filepath.Join(root, "calc", "calc.go")
	data, err := os.ReadFile(calc)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(data), "\treturn a + b\n", "\treturn b + a\n", 1)
	if edited == string(data) {
		t.Fatal("calc.go no longer holds Add's body")
	}
	if err := os.WriteFile(calc, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	ans, _, err := query.Blast(root, query.BlastOptions{NoRefresh: true})
	if err != nil {
		t.Fatalf("blast: %v", err)
	}
	checkGolden(t, filepath.Join(casesDir, "blast.golden"), query.BlastReport(ans))

	// Staged, the same change is an audit finding at threshold 2.
	add := exec.Command("git", "-C", root, "add", ".")
	add.Env = gitenv.Environ()
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	audit, _, err := query.Audit(root, query.AuditOptions{Cached: true, Threshold: 2})
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	checkGolden(t, filepath.Join(casesDir, "audit.golden"), query.AuditReport(audit, 2))
}
