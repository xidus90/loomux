package faketool

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), FixtureName)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func envFor(p string) func(string) string {
	return func(k string) string {
		if k == FixtureEnv {
			return p
		}
		return ""
	}
}

func TestMatchTakesTheLongestPrefixAndStripsExe(t *testing.T) {
	f, err := Load(fixture(t, `{"answers":[
		{"prefix":"uv run","exit":3},
		{"prefix":"uv run pytest","exit":0,"stdout":"1 passed\n"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	a, ok := f.Match([]string{`C:\bin\uv.exe`, "run", "pytest", "-q"})
	if !ok || a.Exit != 0 || a.Stdout != "1 passed\n" {
		t.Fatalf("got %+v %v", a, ok)
	}
	if _, ok := f.Match([]string{"ruff"}); ok {
		t.Fatal("ruff has no answer")
	}
}

func TestMatchWantsAWordBoundary(t *testing.T) {
	f := &Fixture{Answers: []Answer{{Prefix: "uv run", Exit: 3}}}
	if a, ok := f.Match([]string{"uv", "run"}); !ok || a.Exit != 3 {
		t.Fatalf("exact line: %+v %v", a, ok)
	}
	if _, ok := f.Match([]string{"uv", "runner"}); ok {
		t.Fatal("uv runner is not uv run")
	}
}

func TestLoadOfAMissingFileIsEmpty(t *testing.T) {
	f, err := Load(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil || len(f.Answers) != 0 {
		t.Fatalf("%+v %v", f, err)
	}
}

func TestLoadRefusesBrokenJSON(t *testing.T) {
	if _, err := Load(fixture(t, `{`)); err == nil {
		t.Fatal("want error")
	}
}

func TestLoadRefusesAnUnreadablePath(t *testing.T) {
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("a directory is no fixture")
	}
}

func TestApplyWritesFilesWithTheDirToken(t *testing.T) {
	dir := t.TempDir()
	f := &Fixture{}
	a := Answer{Writes: []Write{{Path: "out/cover.out", Content: "mode: set\n{{DIR}}/a.go:1.1,2.2 1 1\n"}}}
	if err := f.Apply(a, dir); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "out", "cover.out"))
	if !bytes.Contains(got, []byte(filepath.ToSlash(dir)+"/a.go")) {
		t.Fatalf("%s", got)
	}
}

func TestApplyFailsWhenTheTargetIsADirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := Answer{Writes: []Write{{Path: "out", Content: "x"}}}
	if err := (&Fixture{}).Apply(a, dir); err == nil {
		t.Fatal("want error")
	}
}

func TestApplyFailsWhenTheParentIsAFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "out"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	a := Answer{Writes: []Write{{Path: "out/cover.out", Content: "x"}}}
	if err := (&Fixture{}).Apply(a, dir); err == nil {
		t.Fatal("want error")
	}
}

func TestMainAnswersAndFailsLoudly(t *testing.T) {
	p := fixture(t, `{"answers":[{"prefix":"ruff check","exit":1,"stdout":"E1 x\n"}]}`)
	getenv := envFor(p)
	var out, errb bytes.Buffer
	if code := Main([]string{"ruff.exe", "check", "."}, getenv, t.TempDir(), &out, &errb); code != 1 || out.String() != "E1 x\n" {
		t.Fatalf("code %d out %q", code, out.String())
	}
	out.Reset()
	if code := Main([]string{"mypy"}, getenv, t.TempDir(), &out, &errb); code != 127 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(errb.String(), `faketool: no answer for "mypy"`) {
		t.Fatalf("stderr %q", errb.String())
	}
}

func TestMainNamesABrokenFixture(t *testing.T) {
	p := fixture(t, `{`)
	var out, errb bytes.Buffer
	if code := Main([]string{"ruff"}, envFor(p), t.TempDir(), &out, &errb); code != 127 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(errb.String(), p) {
		t.Fatalf("stderr %q does not name %s", errb.String(), p)
	}
}

func TestMainSleepsAsTheAnswerSays(t *testing.T) {
	var slept []time.Duration
	orig := sleep
	sleep = func(d time.Duration) { slept = append(slept, d) }
	t.Cleanup(func() { sleep = orig })
	p := fixture(t, `{"answers":[{"prefix":"ctest","sleep_ms":250}]}`)
	var out, errb bytes.Buffer
	if code := Main([]string{"ctest"}, envFor(p), t.TempDir(), &out, &errb); code != 0 {
		t.Fatalf("code %d stderr %q", code, errb.String())
	}
	if len(slept) != 1 || slept[0] != 250*time.Millisecond {
		t.Fatalf("slept %v", slept)
	}
}

func TestMainFailsWhenAWriteFails(t *testing.T) {
	cwd := t.TempDir()
	if err := os.Mkdir(filepath.Join(cwd, "cover.out"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := fixture(t, `{"answers":[{"prefix":"gcovr","stdout":"ok\n","writes":[{"path":"cover.out","content":"x"}]}]}`)
	var out, errb bytes.Buffer
	if code := Main([]string{"gcovr"}, envFor(p), cwd, &out, &errb); code != 127 || out.Len() != 0 {
		t.Fatalf("code %d out %q", code, out.String())
	}
	if !strings.HasPrefix(errb.String(), "faketool: ") {
		t.Fatalf("stderr %q", errb.String())
	}
}
