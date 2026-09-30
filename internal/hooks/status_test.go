package hooks

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/verify"
)

func TestRunStatus(t *testing.T) {
	tmp := t.TempDir()

	// 1. Create a dummy python + wiki project
	if err := os.WriteFile(filepath.Join(tmp, "pyproject.toml"), []byte("[project]\nname=\"test\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "uv.lock"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".loomux", "config.toml"), []byte("[area]\nscope=\"test\"\nwiki=true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	settingsJSON := `{
		"hooks": {
			"PreToolUse": [{"hooks": [{"command": "loomux hook pre-tool-use"}]}],
			"PostToolUse": [{"hooks": [{"command": "loomux hook post-tool-use"}, {"command": "python format_on_edit.py"}]}]
		}
	}`
	if err := os.WriteFile(filepath.Join(tmp, ".claude", "settings.json"), []byte(settingsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Status(&stdout, &stderr, tmp)
	if code != ExitOK {
		t.Fatalf("expected ExitOK (0), got %d", code)
	}

	out := stdout.String()
	if !strings.Contains(out, "loomux Hook Inspection") {
		t.Fatalf("expected header, got %s", out)
	}
	// The lanes are the edit profile as the presets lay it out, not a list of
	// their own.
	for _, want := range []string{
		"     * python (*.py) lint: uvx ruff check . --output-format=concise [preset]\n",
		"     * python (*.py) types: uv run --with mypy mypy --no-error-summary --no-pretty --exclude-gitignore . [preset]\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output, got %s", want, out)
		}
	}
	// The lane is named by the command this binary carries. `brain lint` was a
	// second binary and a Python environment, and this very report calls the
	// old guard obsolete two sections down -- naming it as what loomux runs
	// sent the reader to a tool that is gone.
	if !strings.Contains(out, "loomux lint") {
		t.Fatalf("expected the wiki lane to name 'loomux lint', got %s", out)
	}
	if strings.Contains(out, "brain lint") || strings.Contains(out, "brain wiki-gate") {
		t.Fatalf("no line may name a retired binary as what loomux runs, got %s", out)
	}
	if !strings.Contains(out, "format_on_edit.py") {
		t.Fatalf("expected legacy finding for format_on_edit.py, got %s", out)
	}
	// The reason a legacy hook is obsolete names the command that replaced it,
	// and that is a loomux subcommand. `ulguard` is itself listed as superseded
	// two lines above, so naming it as a successor would send the reader to a
	// binary this very report calls gone. The fixture configures no `ulguard`
	// command, so the string can only reach stdout through a reason.
	if !strings.Contains(out, "Reason: superseded by 'loomux hook post-tool-use'") {
		t.Fatalf("expected the successor to be named in the reason, got %s", out)
	}
	if strings.Contains(out, "ulguard") {
		t.Fatalf("no reason may name ulguard as a successor, got %s", out)
	}
}

func TestRunStatusAllStacksAndNoLegacy(t *testing.T) {
	tmp := t.TempDir()

	// Add files for all stacks
	files := map[string]string{
		"project.godot":      "",
		"CMakeLists.txt":     "cmake_minimum_required(VERSION 3.20)",
		"tsconfig.json":      "{}",
		"package.json":       "{}",
		"App.vue":            "<template></template>",
		"App.svelte":         "<script></script>",
		"style.css":          "body {}",
		"index.html":         "<html></html>",
		"script.sh":          "#!/bin/sh",
		".sqlfluff":          "[sqlfluff]\ndialect = ansi\n",
		"query.sql":          "SELECT 1;",
		"Cargo.toml":         "[package]\nname=\"sample\"\nversion=\"0.1.0\"\n",
		"go.mod":             "module sample\ngo 1.24\n",
		".golangci.yml":      "",
		"pyrightconfig.json": "{}",
		"requirements.txt":   "pytest",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Clean settings without legacy hooks and with no Pre/Post
	if err := os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".claude", "settings.json"), []byte(`{"hooks":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Status(&stdout, &stderr, tmp)
	if code != ExitOK {
		t.Fatalf("expected ExitOK, got %d", code)
	}

	out := stdout.String()
	// Each lane in the form post-edit runs it: the on-file command where the
	// preset has one.
	for _, expected := range []string{
		"gdscript (*.gd) lint: uvx --from gdtoolkit gdlint {file} [preset]",
		"cpp (*.c, *.cc, *.cpp, *.cxx, *.h, *.hpp) lint: clang-format --dry-run --Werror {file} [preset]",
		"cpp (*.c, *.cc, *.cpp, *.cxx, *.h, *.hpp) types: cmake --build build --parallel [preset]",
		"typescript (*.js, *.jsx, *.ts, *.tsx) lint: npx eslint --cache {file} [preset]",
		"vue (*.vue) types: npx vue-tsc --noEmit [preset]",
		"svelte (*.svelte) types: npx svelte-check [preset]",
		"css (*.css, *.less, *.sass, *.scss) lint: npx stylelint {file} [preset]",
		"html (*.htm, *.html) lint: npx htmlhint {file} [preset]",
		"shell (*.bash, *.sh, *.zsh) lint: shellcheck {file} [preset]",
		"sql (*.sql) lint: sqlfluff lint {file} [preset]",
		"rust (*.rs) lint: cargo clippy -- -D warnings ; cargo fmt --check [preset]",
		"go (*.go) lint: go vet ./... ; {loomux} check gofmt {file} [preset, parallel]",
		"python (*.py) types: uv run pyright [",
		"No obsolete or redundant legacy hooks found",
		"UltraBrain Wiki: Inactive / Disabled",
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("expected output to contain %q, but got:\n%s", expected, out)
		}
	}
	// No preset runs these; the old list printed them anyway. golangci-lint
	// itself stays in the detected signals above.
	for _, gone := range []string{"golangci-lint run", "clang-format -i", "dmypy"} {
		if strings.Contains(out, gone) {
			t.Fatalf("no lane runs %q, but the report names it:\n%s", gone, out)
		}
	}
}

// The wiki line and the lane list answer from the same place.
//
// A loomux project declares its bundle as `[layout] wiki`, which detection
// does not read: the report listed the stack `wiki`, printed the wiki lane --
// and called the wiki disabled three lines above. It also read the bundle
// directory from detection, so the lane line named a directory nobody created.
func TestStatusReportsTheDeclaredWikiAsActive(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".loomux", "config.toml"),
		[]byte("[area]\nscope = \"project/x\"\n[layout]\nwiki = \"notes\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := Status(&stdout, &stderr, tmp); code != ExitOK {
		t.Fatalf("code %d", code)
	}
	out := stdout.String()
	if strings.Contains(out, "UltraBrain Wiki: Inactive") {
		t.Fatalf("the report lists the wiki stack and calls the wiki disabled:\n%s", out)
	}
	for _, want := range []string{"UltraBrain Wiki: Active", "'notes'", "*.md (in notes)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in:\n%s", want, out)
		}
	}
}

func TestAuditSettingsEdgeCases(t *testing.T) {
	tmp := t.TempDir()

	// 1. Missing settings file
	findings, installed := auditSettings(tmp)
	if len(findings) != 0 || len(installed) != 0 {
		t.Fatalf("expected empty for missing settings, got findings=%v installed=%v", findings, installed)
	}

	// 2. Invalid JSON in settings file
	claudeDir := filepath.Join(tmp, ".claude")
	_ = os.MkdirAll(claudeDir, 0o755)
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("invalid json"), 0o644)
	findings, installed = auditSettings(tmp)
	if len(findings) != 0 || len(installed) != 0 {
		t.Fatalf("expected empty for invalid json, got findings=%v installed=%v", findings, installed)
	}

	// 3. All legacy hooks in settings
	allLegacyJSON := `{
		"hooks": {
			"PreToolUse": [
				{"hooks": [{"command": "python guard_paths.py"}]}
			],
			"PostToolUse": [
				{"hooks": [{"command": "python post_edit.py"}, {"command": "python generate_index.py"}, {"command": "python lint.py"}]}
			],
			"Stop": [
				{"hooks": [{"command": "python wiki_gate.py"}]}
			]
		}
	}`
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(allLegacyJSON), 0o644)
	findings, installed = auditSettings(tmp)
	if len(findings) != 5 {
		t.Fatalf("expected 5 legacy findings, got %d", len(findings))
	}
	if len(installed) != 0 {
		t.Fatalf("expected no loomux hook installed, got %v", installed)
	}

	// 4. Five of the events this binary serves, each named once: every one is
	// reported, an event nobody wired is not, and no legacy entry matches a
	// loomux command. What this fixture cannot show is which entry `stop` came
	// from -- fixture 6 pins that.
	allLoomuxJSON := `{
		"hooks": {
			"PreToolUse": [{"hooks": [{"command": "loomux hook pre-tool-use --host claude"}]}],
			"PostToolUse": [{"hooks": [{"command": "loomux hook post-tool-use --host claude"}]}],
			"SessionStart": [{"hooks": [{"command": "loomux hook session-start --host claude"}]}],
			"Stop": [{"hooks": [{"command": "loomux hook stop --host claude"}]}],
			"SubagentStop": [{"hooks": [{"command": "loomux hook subagent-stop --host claude"}]}]
		}
	}`
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(allLoomuxJSON), 0o644)
	findings, installed = auditSettings(tmp)
	if len(findings) != 0 {
		t.Fatalf("a loomux command is no legacy hook, got %v", findings)
	}
	for _, event := range []string{"pre-tool-use", "post-tool-use", "session-start", "stop", "subagent-stop"} {
		if !installed[event] {
			t.Fatalf("%s not seen in %v", event, installed)
		}
	}
	if installed["subagent-start"] {
		t.Fatalf("subagent-start is not wired here, got %v", installed)
	}

	// 5. The old Python session hooks are named as superseded, and each is a
	// finding of its own.
	legacySessionJSON := `{
		"hooks": {
			"Stop": [{"hooks": [{"command": "ultraloom hook stop"}]}],
			"SubagentStart": [{"hooks": [{"command": "ultraloom hook subagent-start"}]}],
			"SubagentStop": [{"hooks": [{"command": "ultraloom hook subagent-stop"}]}]
		}
	}`
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(legacySessionJSON), 0o644)
	findings, _ = auditSettings(tmp)
	if len(findings) != 3 {
		t.Fatalf("expected 3 legacy session findings, got %v", findings)
	}
	for _, f := range findings {
		if !strings.Contains(f.Reason, "superseded by 'loomux hook ") {
			t.Fatalf("reason %q", f.Reason)
		}
	}

	// 6. The word after `hook` is read whole. Only the subagent hook is wired
	// here, so a matcher that looked for the bare event name -- `stop`
	// anywhere in the command, or behind a word boundary that `-` satisfies
	// -- would report a stop gate this project does not have, and the report
	// would call the turn end guarded when nothing guards it. The literal
	// `hook stop` as a substring is not what this catches: `hook subagent-stop`
	// does not contain it.
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(
		`{"hooks":{"SubagentStop":[{"hooks":[{"command":"loomux hook subagent-stop --host claude"}]}]}}`), 0o644)
	_, installed = auditSettings(tmp)
	if !installed["subagent-stop"] {
		t.Fatalf("subagent-stop not seen in %v", installed)
	}
	if installed["stop"] {
		t.Fatalf("`hook stop` was found inside `hook subagent-stop`: %v", installed)
	}
}

// The findings are listed in the order of the event names and not in the
// order a map hands them out.
func TestAuditSettingsOrdersItsFindingsByEvent(t *testing.T) {
	tmp := t.TempDir()
	claudeDir := filepath.Join(tmp, ".claude")
	_ = os.MkdirAll(claudeDir, 0o755)
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{
		"hooks": {
			"SubagentStop": [{"hooks": [{"command": "ultraloom hook subagent-stop"}]}],
			"PreToolUse": [{"hooks": [{"command": "python guard_paths.py"}]}],
			"Stop": [{"hooks": [{"command": "ultraloom hook stop"}]}]
		}
	}`), 0o644)
	for i := 0; i < 5; i++ {
		findings, _ := auditSettings(tmp)
		got := []string{findings[0].Event, findings[1].Event, findings[2].Event}
		if !slices.Equal(got, []string{"PreToolUse", "Stop", "SubagentStop"}) {
			t.Fatalf("order %v", got)
		}
	}
}

func TestRunStatusPlainPythonAndGo(t *testing.T) {
	tmp := t.TempDir()

	// Only requirements.txt (plain python) and go.mod (plain go)
	_ = os.WriteFile(filepath.Join(tmp, "requirements.txt"), []byte("requests"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module sample\ngo 1.24\n"), 0o644)

	// Settings with only PreToolUse
	claudeDir := filepath.Join(tmp, ".claude")
	_ = os.MkdirAll(claudeDir, 0o755)
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{"hooks":{"PreToolUse":[{"hooks":[{"command":"loomux hook pre-tool-use"}]}]}}`), 0o644)

	var stdout, stderr bytes.Buffer
	code := Status(&stdout, &stderr, tmp)
	if code != ExitOK {
		t.Fatalf("expected ExitOK, got %d", code)
	}

	out := stdout.String()
	for _, want := range []string{
		"python (*.py) types: uv run --with mypy mypy --no-error-summary --no-pretty --exclude-gitignore . [preset]",
		"go (*.go) lint: go vet ./... ; {loomux} check gofmt {file} [preset, parallel]",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q, got:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "[OK] PreToolUse: 'loomux hook pre-tool-use' installed") {
		t.Fatalf("expected pre installed, got:\n%s", out)
	}
	// Every event this binary serves gets a line, installed or not.
	for _, want := range []string{
		"[INFO] PostToolUse: 'loomux hook post-tool-use' not found in .claude/settings.json",
		"[INFO] SessionStart: 'loomux hook session-start' not found in .claude/settings.json",
		"[INFO] Stop: 'loomux hook stop' not found in .claude/settings.json",
		"[INFO] SubagentStart: 'loomux hook subagent-start' not found in .claude/settings.json",
		"[INFO] SubagentStop: 'loomux hook subagent-stop' not found in .claude/settings.json",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q, got:\n%s", want, out)
		}
	}
	// The gate is the Stop hook now, wiki or no wiki: this project has none.
	if !strings.Contains(out, "  -> loomux hook stop --host claude --root \"${CLAUDE_PROJECT_DIR}\" (profile `stop`, the wiki gate as lint/wiki)") {
		t.Fatalf("expected the stop gate named, got:\n%s", out)
	}
}

// Every hook the settings wire is reported as installed, each by the
// subcommand its command line names.
func TestStatusReportsEveryInstalledHook(t *testing.T) {
	tmp := t.TempDir()
	claudeDir := filepath.Join(tmp, ".claude")
	_ = os.MkdirAll(claudeDir, 0o755)
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{
		"hooks": {
			"PreToolUse": [{"hooks": [{"command": "loomux hook pre-tool-use --host claude"}]}],
			"PostToolUse": [{"hooks": [{"command": "loomux hook post-tool-use --host claude"}]}],
			"SessionStart": [{"hooks": [{"command": "loomux hook session-start --host claude"}]}],
			"Stop": [{"hooks": [{"command": "loomux hook stop --host claude"}]}],
			"SubagentStart": [{"hooks": [{"command": "loomux hook subagent-start --host claude"}]}],
			"SubagentStop": [{"hooks": [{"command": "loomux hook subagent-stop --host claude"}]}]
		}
	}`), 0o644)

	var stdout, stderr bytes.Buffer
	if code := Status(&stdout, &stderr, tmp); code != ExitOK {
		t.Fatalf("code %d", code)
	}
	out := stdout.String()
	for _, h := range [][2]string{
		{"PreToolUse", "pre-tool-use"}, {"PostToolUse", "post-tool-use"}, {"SessionStart", "session-start"},
		{"Stop", "stop"}, {"SubagentStart", "subagent-start"}, {"SubagentStop", "subagent-stop"},
	} {
		want := " [OK] " + h[0] + ": 'loomux hook " + h[1] + "' installed"
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q, got:\n%s", want, out)
		}
	}
}

func TestRunStatusPyrightWithUV(t *testing.T) {
	tmp := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmp, "pyproject.toml"), []byte("[project]\nname=\"p\"\n[tool.pyright]\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, "uv.lock"), []byte(""), 0o644)

	var stdout, stderr bytes.Buffer
	code := Status(&stdout, &stderr, tmp)
	if code != ExitOK {
		t.Fatalf("expected ExitOK, got %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "python (*.py) types: uv run pyright [") {
		t.Fatalf("expected uv run pyright in status output, got:\n%s", out)
	}
}

// The report reads [verify] as the hook does: an override shows with its
// origin, a lane switched off says so, and a config the lanes cannot be read
// from is named instead of a list.
func TestStatusShowsTheLanesAsVerifyShapesThem(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "go.mod"), []byte("module sample\n"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "requirements.txt"), []byte("requests"), 0o644)
	_ = os.MkdirAll(filepath.Join(root, "notes"), 0o755)
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"notes\"\n[verify.go]\nlint = \"golangci-lint run\"\n[verify.python]\ntypes = false\n[verify.wiki]\nlint = false\n")
	var stdout, stderr bytes.Buffer
	Status(&stdout, &stderr, root)
	out := stdout.String()
	for _, want := range []string{
		"  -> loomux hook post-tool-use (profile `edit`: lint, types):\n",
		"     * go (*.go) lint: golangci-lint run [config]\n",
		"     * python (*.py) types: off [config]\n",
		"     * *.md (in notes): off [config]\n",
		"     * ignored (.txt, .json, .yaml, .yml, .toml, .svg, .png, .jpg, .jpeg, .import, .lock): [SKIPPED] no lane runs\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in:\n%s", want, out)
		}
	}

	writeManifest(t, root, "[verify\n")
	stdout.Reset()
	Status(&stdout, &stderr, root)
	if !strings.Contains(stdout.String(), "     [ERROR] the lanes cannot be read: ") {
		t.Fatalf("a broken config is named, got:\n%s", stdout.String())
	}
}

// The missing-tool report is computed from the lanes [verify] and the presets
// lay out, which the hook runs, and not from the lane list printed above it.
// Two lists drift, and this one has to be about the lanes that actually run.
func TestUnavailableLanes(t *testing.T) {
	nothing := func(string) (string, error) { return "", errors.New("not found") }
	everything := func(string) (string, error) { return "/usr/bin/x", nil }
	lanes := func(stacks ...string) verify.Effective {
		eff, err := editLoad(t.TempDir(), detect.Facts{Stacks: stacks})
		if err != nil {
			t.Fatal(err)
		}
		return eff
	}

	// {loomux} is this binary and names nothing to install.
	missing := unavailableLanes(lanes("shell", "go"), nothing)
	if len(missing) != 2 || missing[0] != "go" || missing[1] != "shellcheck" {
		t.Fatalf("expected [go shellcheck] sorted, got %v", missing)
	}

	if got := unavailableLanes(lanes("shell", "go"), everything); len(got) != 0 {
		t.Fatalf("expected nothing missing, got %v", got)
	}

	// A tool named by two lanes is reported once: the answer is about tools to
	// install, not about lanes that mention them.
	onlyGo := func(name string) (string, error) {
		if name == "go" {
			return "/usr/bin/go", nil
		}
		return "", errors.New("not found")
	}
	if got := unavailableLanes(lanes("typescript", "vue"), onlyGo); len(got) != 1 || got[0] != "npx" {
		t.Fatalf("expected [npx] once, got %v", got)
	}

	// No lanes, nothing to install: what Status hands on when the config
	// cannot be read.
	if got := unavailableLanes(verify.Effective{}, nothing); len(got) != 0 {
		t.Fatalf("expected nothing without lanes, got %v", got)
	}
}

func TestRenderLaneTools(t *testing.T) {
	var green bytes.Buffer
	renderLaneTools(&green, nil)
	if !strings.Contains(green.String(), "[OK]") {
		t.Errorf("a machine with every tool says so: %q", green.String())
	}

	var red bytes.Buffer
	renderLaneTools(&red, []string{"shellcheck"})
	if !strings.Contains(red.String(), "shellcheck") || !strings.Contains(red.String(), "[WARN]") {
		t.Errorf("a missing tool is named and marked: %q", red.String())
	}
}
