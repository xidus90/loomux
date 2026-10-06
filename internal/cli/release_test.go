package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/release"
)

func runIn(stdin string, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Run(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func TestDevReleaseNeedsAKnownSubcommand(t *testing.T) {
	if code, _, e := run("dev", "release"); code != 2 || !strings.Contains(e, "subcommand required") {
		t.Fatalf("code %d, err %q", code, e)
	}
	if code, _, e := run("dev", "release", "nope"); code != 2 || !strings.Contains(e, `unknown subcommand "nope"`) {
		t.Fatalf("code %d, err %q", code, e)
	}
}

func TestDevReleaseNextVersion(t *testing.T) {
	if code, out, _ := runIn("v1.0.0\nv1.1.0\n", "dev", "release", "next-version", "--bump", "patch"); code != 0 || out != "1.1.1\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
	file := filepath.Join(t.TempDir(), "tags")
	os.WriteFile(file, []byte("v3.0.0\n"), 0o644)
	if code, out, _ := run("dev", "release", "next-version", "--bump", "major", "--tags", file); code != 0 || out != "4.0.0\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
	if code, _, _ := run("dev", "release", "next-version", "--bump", "none"); code != 2 {
		t.Fatalf("bad bump: code %d", code)
	}
	if code, _, _ := run("dev", "release", "next-version", "--bump", "patch", "--tags", filepath.Join(t.TempDir(), "missing")); code != 2 {
		t.Fatalf("missing file: code %d", code)
	}
	if code, _, _ := run("dev", "release", "next-version", "--bogus"); code != 2 {
		t.Fatalf("bad flag: code %d", code)
	}
}

func TestDevReleaseNextBeta(t *testing.T) {
	if code, out, _ := runIn("v1.0.0\nv1.1.0-beta.1\n", "dev", "release", "next-beta", "--bump", "minor"); code != 0 || out != "1.1.0-beta.2\n" {
		t.Fatalf("next-beta = %d %q", code, out)
	}
	if code, _, errs := run("dev", "release", "next-beta", "--bump", "none"); code != 2 || !strings.Contains(errs, "loomux dev release next-beta: bump must be major, minor or patch") {
		t.Fatalf("bad bump = %d %q", code, errs)
	}
	if code, _, _ := run("dev", "release", "next-beta", "--bump", "patch", "--tags", filepath.Join(t.TempDir(), "missing")); code != 2 {
		t.Fatalf("missing tag file = %d", code)
	}
	if code, _, errs := run("dev", "release", "next-beta", "--bogus"); code != 2 || !strings.Contains(errs, "dev release next-beta") {
		t.Fatalf("unknown flag = %d %q", code, errs)
	}
}

func TestDevReleaseParseBody(t *testing.T) {
	body := "## Changelog\n### Fixed\n- x\n"
	code, out, _ := runIn(body, "dev", "release", "parse-body", "--labels", "release:patch")
	if code != 0 || out != `{"bump":"patch","changelog":"### Fixed\n- x\n"}`+"\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
	code, _, e := runIn("", "dev", "release", "parse-body", "--labels", "")
	if code != 1 || !strings.Contains(e, "found 0") {
		t.Fatalf("code %d, err %q", code, e)
	}
	if code, _, _ := run("dev", "release", "parse-body", "--body", filepath.Join(t.TempDir(), "missing")); code != 2 {
		t.Fatalf("missing file: code %d", code)
	}
	if code, _, _ := run("dev", "release", "parse-body", "--bogus"); code != 2 {
		t.Fatalf("bad flag: code %d", code)
	}
}

func TestDevReleaseParseBodyCommits(t *testing.T) {
	dir := t.TempDir()
	body := "## Changelog\n### Fixed\n- x\n"
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte(content), 0o644)
		return p
	}
	ok := write("ok.json", `["fix: a", "chore: b"]`)
	if code, _, e := runIn(body, "dev", "release", "parse-body", "--labels", "release:patch", "--commits", ok); code != 0 {
		t.Fatalf("ok: code %d, err %q", code, e)
	}
	high := write("high.json", `["feat: x\n\nbody"]`)
	code, _, e := runIn(body, "dev", "release", "parse-body", "--labels", "release:patch", "--commits", high)
	if code != 1 || !strings.Contains(e, `commit "feat: x" needs at least release:minor, the label is release:patch`) {
		t.Fatalf("high: code %d, err %q", code, e)
	}
	bad := write("bad.json", `{"not": "an array"}`)
	if code, _, _ := runIn(body, "dev", "release", "parse-body", "--labels", "release:patch", "--commits", bad); code != 2 {
		t.Fatalf("bad json: code %d", code)
	}
	if code, _, _ := runIn(body, "dev", "release", "parse-body", "--labels", "release:patch", "--commits", filepath.Join(dir, "missing")); code != 2 {
		t.Fatalf("missing: code %d", code)
	}
}

func TestDevReleaseChangelogInsert(t *testing.T) {
	file := filepath.Join(t.TempDir(), "CHANGELOG.md")
	args := []string{"dev", "release", "changelog-insert", "--version", "1.0.0", "--date", "2026-09-18", "--link", "l", "--file", file}
	if code, _, e := runIn("### Added\n- a\n", args...); code != 0 {
		t.Fatalf("code %d: %s", code, e)
	}
	got, _ := os.ReadFile(file)
	if !strings.Contains(string(got), "## [1.0.0] - 2026-09-18") {
		t.Fatalf("file %q", got)
	}
	if code, _, _ := runIn("### Added\n- a\n", args...); code != 1 {
		t.Fatalf("duplicate: code %d", code)
	}
	if code, _, _ := run("dev", "release", "changelog-insert", "--version", "1.0.0"); code != 2 {
		t.Fatalf("missing flags: code %d", code)
	}
	if code, _, _ := run("dev", "release", "changelog-insert", "--bogus"); code != 2 {
		t.Fatalf("bad flag: code %d", code)
	}
	dir := t.TempDir()
	if code, _, _ := runIn("n", "dev", "release", "changelog-insert", "--version", "1", "--date", "d", "--link", "l", "--file", dir); code != 2 {
		t.Fatalf("directory as file: code %d", code)
	}
	if code, _, _ := run("dev", "release", "changelog-insert", "--version", "1", "--date", "d", "--link", "l", "--file", file, "--notes", filepath.Join(dir, "missing")); code != 2 {
		t.Fatalf("missing notes: code %d", code)
	}
}

func TestDevReleaseBuild(t *testing.T) {
	defer func(g release.GoBuild) { releaseGo = g }(releaseGo)
	releaseGo = func(env []string, args ...string) error {
		for i, a := range args {
			if a == "-o" {
				return os.WriteFile(args[i+1], []byte("bin"), 0o755)
			}
		}
		return nil
	}
	out := t.TempDir()
	code, stdout, e := run("dev", "release", "build", "--version", "1.0.0", "--out", out)
	if code != 0 || !strings.HasSuffix(stdout, "SHA256SUMS\n") {
		t.Fatalf("code %d, out %q, err %q", code, stdout, e)
	}
	if code, _, _ := run("dev", "release", "build", "--version", "1.0.0", "--channel", "beta", "--out", out); code != 2 {
		t.Fatalf("--channel is gone: code %d", code)
	}
	if code, _, _ := run("dev", "release", "build", "--out", out); code != 2 {
		t.Fatalf("missing version: code %d", code)
	}
	if code, _, _ := run("dev", "release", "build", "--bogus"); code != 2 {
		t.Fatalf("bad flag: code %d", code)
	}
	releaseGo = func([]string, ...string) error { return os.ErrPermission }
	if code, _, _ := run("dev", "release", "build", "--version", "1.0.0", "--out", out); code != 1 {
		t.Fatalf("failed build: code %d", code)
	}
}
