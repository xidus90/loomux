package godotsuite

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/verify/linecov"
)

// copyTree copies the directory src to dst.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestGlueAgainstARealEngine runs the embedded runner and hook once against
// the fixture project with a real engine. It needs GODOT_BIN, the gdUnit4
// addon directory in LOOMUX_GDUNIT4_DIR and the Nano Coverage addon directory
// in LOOMUX_NANO_DIR; without them it skips, as CI has no engine.
func TestGlueAgainstARealEngine(t *testing.T) {
	bin := os.Getenv("GODOT_BIN")
	gdunit := os.Getenv("LOOMUX_GDUNIT4_DIR")
	nano := os.Getenv("LOOMUX_NANO_DIR")
	if bin == "" || gdunit == "" || nano == "" {
		t.Skip("needs GODOT_BIN, LOOMUX_GDUNIT4_DIR and LOOMUX_NANO_DIR")
	}
	root := t.TempDir()
	copyTree(t, filepath.Join("..", "..", "..", "testdata", "godot", "project"), root)
	copyTree(t, gdunit, filepath.Join(root, "addons", "gdUnit4"))
	copyTree(t, nano, filepath.Join(root, "addons", "nano_coverage_godot"))

	shard := t.TempDir()
	reports := filepath.Join(t.TempDir(), "reports")
	engine := func(args ...string) (string, error) {
		cmd := exec.Command(bin, args...)
		cmd.Dir = root
		var out bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &out
		cmd.Env = append(os.Environ(), "APPDATA="+t.TempDir(), "LOOMUX_COVERAGE_SHARD_DIR="+shard)
		return out.String(), cmd.Run()
	}
	if out, err := engine("--path", root, "--headless", "--import"); err != nil {
		t.Fatalf("import: %v\n%s", err, out)
	}
	if err := WriteGlue(root); err != nil {
		t.Fatal(err)
	}

	// Relative, as runShard passes it.
	rd, err := filepath.Rel(root, reports)
	if err != nil {
		t.Fatal(err)
	}
	out, err := engine("--path", root, "-s", "-d", "--remote-debug", "tcp://127.0.0.1:0",
		"res://"+GlueDir+"/run_GdUnitCmdTool.gd", "-a", "res://test", "-c",
		"--headless", "--ignoreHeadlessMode", "-rd", filepath.ToSlash(rd))
	if err == nil {
		t.Fatalf("a failing test must give a non-zero exit\n%s", out)
	}
	xml, rerr := os.ReadFile(filepath.Join(reports, "report_1", "results.xml"))
	if rerr != nil {
		t.Fatalf("no results.xml: %v\n%s", rerr, out)
	}
	if !strings.Contains(string(xml), "<failure") || !strings.Contains(string(xml), "red_test") {
		t.Errorf("results.xml lacks the red test's failure:\n%s", xml)
	}
	if _, err := os.Stat(filepath.Join(shard, "report", "lcov.info")); err != nil {
		t.Errorf("the hook wrote no LCOV report: %v\n%s", err, out)
	}
}

// TestRunAgainstARealEngine drives Run end to end with the real child: two
// shards, the red test as the only finding, and with Coverage the disk
// bracket on the autoload.
func TestRunAgainstARealEngine(t *testing.T) {
	bin := os.Getenv("GODOT_BIN")
	gdunit := os.Getenv("LOOMUX_GDUNIT4_DIR")
	nano := os.Getenv("LOOMUX_NANO_DIR")
	if bin == "" || gdunit == "" || nano == "" {
		t.Skip("needs GODOT_BIN, LOOMUX_GDUNIT4_DIR and LOOMUX_NANO_DIR")
	}
	root := filepath.Join(t.TempDir(), "proj")
	copyTree(t, filepath.Join("..", "..", "..", "testdata", "godot", "project"), root)
	copyTree(t, gdunit, filepath.Join(root, "addons", "gdUnit4"))
	copyTree(t, nano, filepath.Join(root, "addons", "nano_coverage_godot"))

	// The import rewrites project.godot and registers the coverage hook in it;
	// put the original back as loomux godot rescan does.
	cfg, err := os.ReadFile(filepath.Join(root, "project.godot"))
	if err != nil {
		t.Fatal(err)
	}
	imp := exec.Command(bin, "--path", root, "--headless", "--import")
	imp.Dir = root
	if out, err := imp.CombinedOutput(); err != nil {
		t.Fatalf("import: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(root, "project.godot"), cfg, 0o644); err != nil {
		t.Fatal(err)
	}
	sources := map[string][]byte{}
	for _, rel := range []string{"src/a.gd", "autoload/clock.gd"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		sources[rel] = data
	}

	checkout := t.TempDir()
	opts := Options{Root: root, CheckoutRoot: checkout, Area: ".", Godot: bin, RunID: "it", CPUs: 2}
	onlyRed := func(res Result) {
		t.Helper()
		if len(res.Findings) != 1 || !strings.HasPrefix(res.Findings[0], "res://test/red_test.gd test_fails") {
			t.Fatalf("want only the red test, got %q", res.Findings)
		}
	}

	res, err := Run(opts)
	if err != nil {
		t.Fatal(err)
	}
	onlyRed(res)
	// The red run keeps its shard directories: one per planned shard.
	for _, i := range []string{"0", "1"} {
		if !exists(filepath.Join(checkout, ".loomux", "state", "godot-suite", "it-root", i)) {
			t.Errorf("shard %s did not run", i)
		}
	}

	opts.Coverage = filepath.Join(checkout, "cov")
	opts.RunID = "it-cov"
	if res, err = Run(opts); err != nil {
		t.Fatal(err)
	}
	onlyRed(res)
	infos, _ := filepath.Glob(filepath.Join(opts.Coverage, "*.info"))
	if len(infos) == 0 {
		t.Fatal("no coverage report was copied")
	}
	hit := false
	for _, p := range infos {
		data, _ := os.ReadFile(p)
		_, block, found := strings.Cut(string(data), "SF:autoload/clock.gd\n")
		block, _, _ = strings.Cut(block, "end_of_record")
		for _, l := range strings.Split(block, "\n") {
			if n, ok := strings.CutPrefix(l, "DA:"); found && ok && !strings.HasSuffix(n, ",0") {
				hit = true
			}
		}
	}
	if !hit {
		t.Error("the autoload shows no hit: the disk instrumentation did not reach it")
	}
	for rel, was := range sources {
		if now, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))); !bytes.Equal(now, was) {
			t.Errorf("%s changed:\n%s", rel, now)
		}
	}
	// loomux's own glue under .godot is no source of the project's.
	lc, err := linecov.Check(linecov.Options{Root: root, Reports: []string{opts.Coverage}})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range lc.Findings {
		if strings.Contains(f.Path, ".godot/") {
			t.Errorf("linecov looks into .godot: %v", f)
		}
	}
	if exists(filepath.Join(root, "coverage-data")) {
		t.Error("coverage-data was left behind")
	}
}

// TestRunnerQuitsWhenTheNanoExtensionIsNotLoaded runs a coverage shard with
// the Nano Coverage addon but without its libraries, as on a platform whose
// library the addon names under another file name than it ships. The engine
// must quit with a message instead of idling after a script error until the
// lane's timeout.
func TestRunnerQuitsWhenTheNanoExtensionIsNotLoaded(t *testing.T) {
	bin := os.Getenv("GODOT_BIN")
	gdunit := os.Getenv("LOOMUX_GDUNIT4_DIR")
	nano := os.Getenv("LOOMUX_NANO_DIR")
	if bin == "" || gdunit == "" || nano == "" {
		t.Skip("needs GODOT_BIN, LOOMUX_GDUNIT4_DIR and LOOMUX_NANO_DIR")
	}
	root := t.TempDir()
	copyTree(t, filepath.Join("..", "..", "..", "testdata", "godot", "project"), root)
	copyTree(t, gdunit, filepath.Join(root, "addons", "gdUnit4"))
	addon := filepath.Join(root, "addons", "nano_coverage_godot")
	copyTree(t, nano, addon)
	if err := os.RemoveAll(filepath.Join(addon, "bin")); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	engine := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, bin, args...)
		cmd.Dir = root
		var out bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &out
		cmd.Env = append(os.Environ(), "APPDATA="+t.TempDir(), "LOOMUX_COVERAGE_SHARD_DIR="+t.TempDir())
		err := cmd.Run()
		return out.String(), err
	}
	// Only the class cache the import leaves matters, not its exit.
	engine("--path", root, "--headless", "--import")
	if err := WriteGlue(root); err != nil {
		t.Fatal(err)
	}

	out, err := engine("--path", root, "-s", "-d", "--remote-debug", "tcp://127.0.0.1:0",
		"res://"+GlueDir+"/run_GdUnitCmdTool.gd", "-a", "res://test", "-c",
		"--headless", "--ignoreHeadlessMode", "-rd", filepath.Join(t.TempDir(), "reports"))
	if ctx.Err() != nil {
		t.Fatalf("the engine did not quit on its own\n%s", out)
	}
	if err == nil {
		t.Fatalf("a run without the extension must give a non-zero exit\n%s", out)
	}
	if !strings.Contains(out, "loomux: the Nano Coverage extension is not loaded") {
		t.Errorf("the output does not name the missing extension:\n%s", out)
	}
}
