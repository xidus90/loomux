package cli

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/switchover"
)

func TestDevSwitchoverWithoutSubcommandPrintsTheGroupsHelp(t *testing.T) {
	code, out, errOut := run("dev", "switchover")
	if code != 2 || out != "" {
		t.Fatalf("code=%d out=%q", code, out)
	}
	for _, want := range []string{"usage: loomux dev switchover <render|prune-hooks> [flags]",
		"  render       write a project's apply.sh from a JSON file of its parameters",
		"  prune-hooks  remove the hook groups of a settings.json that loomux replaces"} {
		if !strings.Contains(errOut, want) {
			t.Fatalf("help lacks %q:\n%s", want, errOut)
		}
	}
	code, _, errOut = run("dev", "switchover", "nope")
	if code != 2 || !strings.Contains(errOut, `loomux dev switchover: unknown subcommand "nope"`) ||
		!strings.Contains(errOut, "usage: loomux dev switchover") {
		t.Fatalf("unknown: code %d, stderr %q", code, errOut)
	}
}

// switchoverParams writes a parameter file that renders.
func switchoverParams(t *testing.T, dir string) (string, switchover.Params) {
	t.Helper()
	p := switchover.Params{Name: "x", Project: "/p/x", Loomux: "loomux", ConfigNew: "/prep/config.toml",
		OldFiles: []string{".ultraloom"}}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "params.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path, p
}

func TestDevSwitchoverRenderWritesTheScript(t *testing.T) {
	dir := t.TempDir()
	params, p := switchoverParams(t, dir)
	out := filepath.Join(dir, "apply.sh")
	code, stdout, errOut := run("dev", "switchover", "render", "--params", params, "--out", out)
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	want, err := switchover.Render(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want || !strings.HasPrefix(want, "#!/bin/sh\n") {
		t.Fatalf("the file is not the rendered script:\n%.300s", got)
	}
	if stdout != out+"\n" {
		t.Fatalf("stdout %q", stdout)
	}
}

// A script cut short by a failed write would look like the one a human is to
// read, and would keep the next render from writing the whole one.
func TestDevSwitchoverRenderLeavesNoHalfWrittenScript(t *testing.T) {
	dir := t.TempDir()
	params, _ := switchoverParams(t, dir)
	out := filepath.Join(dir, "apply.sh")
	orig := writeAndCloseFile
	writeAndCloseFile = func(w io.WriteCloser, data []byte) error {
		return errors.Join(orig(w, data[:len(data)/2]), errors.New("disk full"))
	}
	code, _, errOut := run("dev", "switchover", "render", "--params", params, "--out", out)
	writeAndCloseFile = orig
	if code != 1 || !strings.Contains(errOut, "loomux dev switchover render: ") || !strings.Contains(errOut, "disk full") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("a truncated script is left (%v)", err)
	}
	if code, _, errOut := run("dev", "switchover", "render", "--params", params, "--out", out); code != 0 {
		t.Fatalf("the next render: code %d, err %q", code, errOut)
	}
}

func TestDevSwitchoverRenderReadsTheJSONNames(t *testing.T) {
	dir := t.TempDir()
	params := filepath.Join(dir, "params.json")
	text := `{"name": "x", "project": "/p/x", "loomux": "loomux", "config_new": "/c",
		"old_files": [".ultraloom", ".brain.toml"], "old_hooks": ["ulguard", "brain guard"],
		"wiki_srcs": ["/v/91 Projekte/x"], "wiki_dst": "/p/x/docs/wiki",
		"vault": "/v", "vault_old": "91 Projekte/x", "state_area": "/s",
		"registry": "/r", "registry_new": "/r.new",
		"registry_sum": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}`
	if err := os.WriteFile(params, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "apply.sh")
	if code, _, errOut := run("dev", "switchover", "render", "--params", params, "--out", out); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	got, _ := os.ReadFile(out)
	for _, line := range []string{"CONFIG_NEW='/c'", "OLD_FILES='.ultraloom .brain.toml'", "OLD_HOOKS='ulguard|brain guard'",
		"WIKI_SRCS='/v/91 Projekte/x'", "WIKI_DST='/p/x/docs/wiki'", "VAULT='/v'", "VAULT_OLD='91 Projekte/x'",
		"STATE_AREA='/s'", "REGISTRY='/r'", "REGISTRY_NEW='/r.new'"} {
		if !strings.Contains(string(got), "\n"+line+"\n") {
			t.Errorf("the script lacks %s", line)
		}
	}
}

func TestDevSwitchoverRenderRefusesAWrongCall(t *testing.T) {
	dir := t.TempDir()
	params, _ := switchoverParams(t, dir)
	out := filepath.Join(dir, "apply.sh")
	for _, args := range [][]string{
		{"--out", out},
		{"--params", params},
		{"--params", params, "--out", out, "extra"},
		{"--bogus"},
	} {
		code, _, errOut := run(append([]string{"dev", "switchover", "render"}, args...)...)
		if code != 2 {
			t.Errorf("%q: code %d, stderr %q", args, code, errOut)
		}
		if exists(out) {
			t.Fatalf("%q wrote the script", args)
		}
	}
	_, _, errOut := run("dev", "switchover", "render", "--out", out)
	if !strings.Contains(errOut, "loomux dev switchover render: --params and --out are required") {
		t.Errorf("stderr %q", errOut)
	}
	_, _, errOut = run("dev", "switchover", "render", "--params", params, "--out", out, "extra")
	if !strings.Contains(errOut, `loomux dev switchover render: unexpected argument "extra"`) {
		t.Errorf("stderr %q", errOut)
	}
}

func TestDevSwitchoverRenderRefusesParametersItCannotUse(t *testing.T) {
	for name, text := range map[string]string{
		"invalid JSON":   `{"name": "x",`,
		"unknown field":  `{"name": "x", "project": "/p", "loomux": "l", "config_new": "/c", "old_file": [".x"]}`,
		"trailing data":  `{"name": "x", "project": "/p", "loomux": "l", "config_new": "/c"} {}`,
		"trailing brace": `{"name": "x", "project": "/p", "loomux": "l", "config_new": "/c"} }`,
		"missing field":  `{"name": "x", "project": "/p", "loomux": "l"}`,
	} {
		dir := t.TempDir()
		params := filepath.Join(dir, "params.json")
		if err := os.WriteFile(params, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(dir, "apply.sh")
		code, _, errOut := run("dev", "switchover", "render", "--params", params, "--out", out)
		if code != 1 || !strings.Contains(errOut, "loomux dev switchover render: ") {
			t.Errorf("%s: code %d, stderr %q", name, code, errOut)
		}
		if exists(out) {
			t.Errorf("%s: the script was written", name)
		}
	}
	dir := t.TempDir()
	gone := filepath.Join(dir, "gone.json")
	code, _, errOut := run("dev", "switchover", "render", "--params", gone, "--out", filepath.Join(dir, "a.sh"))
	if code != 1 || !strings.Contains(errOut, "loomux dev switchover render: "+readError(t, gone)) {
		t.Errorf("missing file: code %d, stderr %q", code, errOut)
	}
	params := filepath.Join(dir, "params.json")
	os.WriteFile(params, []byte(`{"name": "x",`), 0o644)
	if _, _, errOut = run("dev", "switchover", "render", "--params", params, "--out", filepath.Join(dir, "a.sh")); !strings.Contains(errOut, "render: "+params+": ") {
		t.Errorf("the broken file is not named: %q", errOut)
	}
	os.WriteFile(params, []byte(`{"name": "x", "project": "/p", "loomux": "l"}`), 0o644)
	_, _, errOut = run("dev", "switchover", "render", "--params", params, "--out", filepath.Join(dir, "a.sh"))
	if !strings.Contains(errOut, "config_new is required") {
		t.Errorf("the reason is not named: %q", errOut)
	}
}

func TestDevSwitchoverRenderRefusesAnExistingScript(t *testing.T) {
	dir := t.TempDir()
	params, _ := switchoverParams(t, dir)
	out := filepath.Join(dir, "apply.sh")
	if err := os.WriteFile(out, []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := run("dev", "switchover", "render", "--params", params, "--out", out)
	if code != 1 || !strings.Contains(errOut, "loomux dev switchover render: ") {
		t.Fatalf("code %d, stderr %q", code, errOut)
	}
	if got, _ := os.ReadFile(out); string(got) != "mine\n" {
		t.Fatalf("the file became %q", got)
	}
}

const switchoverSettings = `{
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "uv run ultraloom hook session-start"}]}
    ],
    "PreToolUse": [
      {"matcher": "Edit", "hooks": [{"type": "command", "command": "ulguard --root x"}]},
      {"matcher": "Edit", "hooks": [{"type": "command", "command": "loomux hook pre-tool-use --host claude"}]}
    ],
    "Stop": [
      {"hooks": [{"type": "command", "command": "brain wiki-gate"}, {"type": "command", "command": "my-own-check"}]}
    ]
  }
}
`

// switchoverSettingsFile writes the settings and dates the file back, so that
// a write shows in its modification time.
func switchoverSettingsFile(t *testing.T, text string) (string, time.Time) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	return path, old
}

func TestDevSwitchoverPruneHooksRemovesTheReplacedGroups(t *testing.T) {
	path, _ := switchoverSettingsFile(t, switchoverSettings)
	code, out, errOut := run("dev", "switchover", "prune-hooks", "--file", path,
		"--match", "ulguard", "--match", "ultraloom hook", "--match", "brain wiki-gate")
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	want := "removed: SessionStart: uv run ultraloom hook session-start\n" +
		"removed: PreToolUse: ulguard --root x\n" +
		"kept: Stop: brain wiki-gate\n"
	if out != want {
		t.Fatalf("stdout\n%s\nwant\n%s", out, want)
	}
	pruned, _, _, err := switchover.PruneHooks([]byte(switchoverSettings), []string{"ulguard", "ultraloom hook", "brain wiki-gate"})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != string(pruned) {
		t.Fatalf("file\n%s\nwant\n%s", got, pruned)
	}
}

func TestDevSwitchoverPruneHooksLeavesAFileWithoutAMatchAlone(t *testing.T) {
	path, old := switchoverSettingsFile(t, switchoverSettings)
	orig := switchoverWriteFile
	switchoverWriteFile = func(string, []byte, os.FileMode) error { return errors.New("written") }
	t.Cleanup(func() { switchoverWriteFile = orig })
	code, out, errOut := run("dev", "switchover", "prune-hooks", "--file", path, "--match", "nothing-like-it")
	if code != 0 || out != "removed nothing\n" {
		t.Fatalf("code %d, stdout %q, stderr %q", code, out, errOut)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(old) {
		t.Fatalf("the file was written: %v", info.ModTime())
	}
	if got, _ := os.ReadFile(path); string(got) != switchoverSettings {
		t.Fatal("the file changed")
	}
	// A group that mixes a match with another command stays and is named.
	code, out, _ = run("dev", "switchover", "prune-hooks", "--file", path, "--match", "brain wiki-gate")
	if code != 0 || out != "removed nothing\nkept: Stop: brain wiki-gate\n" {
		t.Fatalf("code %d, stdout %q", code, out)
	}
}

func TestDevSwitchoverPruneHooksRefusesAWrongCall(t *testing.T) {
	path, _ := switchoverSettingsFile(t, switchoverSettings)
	for _, args := range [][]string{
		{"--file", path},
		{"--match", "ulguard"},
		{"--file", path, "--match", "ulguard", "extra"},
		{"--bogus"},
	} {
		code, _, errOut := run(append([]string{"dev", "switchover", "prune-hooks"}, args...)...)
		if code != 2 {
			t.Errorf("%q: code %d, stderr %q", args, code, errOut)
		}
	}
	_, _, errOut := run("dev", "switchover", "prune-hooks", "--file", path)
	if !strings.Contains(errOut, "loomux dev switchover prune-hooks: --file and at least one --match are required") {
		t.Errorf("stderr %q", errOut)
	}
	_, _, errOut = run("dev", "switchover", "prune-hooks", "--file", path, "--match", "x", "extra")
	if !strings.Contains(errOut, `loomux dev switchover prune-hooks: unexpected argument "extra"`) {
		t.Errorf("stderr %q", errOut)
	}
	if got, _ := os.ReadFile(path); string(got) != switchoverSettings {
		t.Fatal("a wrong call changed the file")
	}
}

func TestDevSwitchoverPruneHooksReportsWhatFails(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.json")
	os.WriteFile(broken, []byte(`{"hooks": `), 0o644)
	for name, file := range map[string]string{"missing": filepath.Join(dir, "gone.json"), "a directory": dir, "broken": broken} {
		code, _, errOut := run("dev", "switchover", "prune-hooks", "--file", file, "--match", "ulguard")
		if code != 1 || !strings.Contains(errOut, "loomux dev switchover prune-hooks: ") {
			t.Errorf("%s: code %d, stderr %q", name, code, errOut)
		}
	}
	// A file that cannot be read is reported with the reason of the read.
	gone := filepath.Join(dir, "gone.json")
	if _, _, errOut := run("dev", "switchover", "prune-hooks", "--file", gone, "--match", "ulguard"); !strings.Contains(errOut, "prune-hooks: "+readError(t, gone)) {
		t.Errorf("missing: stderr %q", errOut)
	}
	path, _ := switchoverSettingsFile(t, switchoverSettings)
	orig := switchoverWriteFile
	switchoverWriteFile = func(string, []byte, os.FileMode) error { return errors.New("disk full") }
	t.Cleanup(func() { switchoverWriteFile = orig })
	code, out, errOut := run("dev", "switchover", "prune-hooks", "--file", path, "--match", "ulguard")
	if code != 1 || !strings.Contains(errOut, "loomux dev switchover prune-hooks: disk full") || out != "" {
		t.Fatalf("code %d, stdout %q, stderr %q", code, out, errOut)
	}
}
