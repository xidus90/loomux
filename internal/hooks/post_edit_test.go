package hooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestRunPostEdit(t *testing.T) {
	tests := []struct {
		name         string
		payload      string
		stacks       []string
		expectedCmds []string
		expectedExit int
	}{
		{
			name:         "Python file triggers only python tools",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "src/main.py"}}`,
			stacks:       []string{"python", "uv", "cpp", "gdscript"},
			expectedCmds: []string{"ruff check", "dmypy run"},
			expectedExit: 0,
		},
		{
			name:         "Python plain without uv triggers mypy",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "src/main.py"}}`,
			stacks:       []string{"python"},
			expectedCmds: []string{"ruff check", "mypy"},
			expectedExit: 0,
		},
		{
			name:         "C++ file triggers only cpp tools",
			payload:      `{"tool_name": "Write", "tool_input": {"file_path": "core/engine.cpp"}}`,
			stacks:       []string{"python", "cpp", "cmake"},
			expectedCmds: []string{"clang-format -i", "cmake --build"},
			expectedExit: 0,
		},
		{
			name:         "GDScript file triggers only gdlint",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "player.gd"}}`,
			stacks:       []string{"gdscript", "cpp"},
			expectedCmds: []string{"gdlint"},
			expectedExit: 0,
		},
		{
			name:         "TypeScript file triggers eslint and tsc",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "app.ts"}}`,
			stacks:       []string{"typescript"},
			expectedCmds: []string{"npx eslint --cache app.ts", "npx tsc --noEmit"},
			expectedExit: 0,
		},
		{
			name:         "TypeScript file in nested workspace triggers prefix npm commands",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "frontend/src/app.tsx"}}`,
			stacks:       []string{"typescript"},
			expectedCmds: []string{"npx --prefix frontend eslint --config frontend/eslint.config.js --cache frontend/src/app.tsx", "npm --prefix frontend run typecheck"},
			expectedExit: 0,
		},
		{
			name:         "CSS file triggers stylelint",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "src/styles.css"}}`,
			stacks:       []string{"css"},
			expectedCmds: []string{"npx stylelint src/styles.css"},
			expectedExit: 0,
		},
		{
			name:         "HTML file triggers htmlhint",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "public/index.html"}}`,
			stacks:       []string{"html"},
			expectedCmds: []string{"npx htmlhint public/index.html"},
			expectedExit: 0,
		},
		{
			name:         "Shell file triggers shellcheck",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "scripts/deploy.sh"}}`,
			stacks:       []string{"shell"},
			expectedCmds: []string{"shellcheck scripts/deploy.sh"},
			expectedExit: 0,
		},
		{
			name:         "SQL file triggers sqlfluff",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "queries/users.sql"}}`,
			stacks:       []string{"sql"},
			expectedCmds: []string{"sqlfluff lint queries/users.sql"},
			expectedExit: 0,
		},
		{
			name:         "Vue file triggers vue-tsc",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "src/Component.vue"}}`,
			stacks:       []string{"vue"},
			expectedCmds: []string{"npx vue-tsc --noEmit"},
			expectedExit: 0,
		},
		{
			name:         "Svelte file triggers svelte-check",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "src/App.svelte"}}`,
			stacks:       []string{"svelte"},
			expectedCmds: []string{"npx svelte-check"},
			expectedExit: 0,
		},
		{
			name:         "Rust file triggers clippy and fmt",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "src/lib.rs"}}`,
			stacks:       []string{"rust"},
			expectedCmds: []string{"cargo clippy", "cargo fmt"},
			expectedExit: 0,
		},
		{
			name:         "Markdown file triggers no commands when wiki stack is inactive",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "docs/README.md"}}`,
			stacks:       []string{"python"},
			expectedCmds: []string{},
			expectedExit: 0,
		},
		{
			name:         "Markdown file outside wiki directory is skipped even when wiki stack is active",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "README.md"}}`,
			stacks:       []string{"python", "wiki"},
			expectedCmds: []string{},
			expectedExit: 0,
		},
		{
			name:         "Go file triggers go vet",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "cmd/main.go"}}`,
			stacks:       []string{"go"},
			expectedCmds: []string{"go vet ./..."},
			expectedExit: 0,
		},
		{
			name:         "Markdown file exits immediately with 0",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "docs/README.md"}}`,
			stacks:       []string{"python", "cpp", "gdscript"},
			expectedCmds: nil,
			expectedExit: 0,
		},
		{
			name:         "Unknown extension triggers fallback to all stacks",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "custom.xyz"}}`,
			stacks:       []string{"python", "uv", "cpp"},
			expectedCmds: []string{"ruff check", "dmypy run", "clang-format -i", "cmake --build"},
			expectedExit: 0,
		},
		{
			name:         "NotebookEdit with notebook_path",
			payload:      `{"tool_name": "NotebookEdit", "tool_input": {"notebook_path": "analysis.py"}}`,
			stacks:       []string{"python"},
			expectedCmds: []string{"ruff check"},
			expectedExit: 0,
		},
		{
			name:         "Invalid json exits 0",
			payload:      `invalid json`,
			stacks:       []string{"python"},
			expectedCmds: nil,
			expectedExit: 0,
		},
		{
			name:         "Empty path exits 0",
			payload:      `{"tool_name": "Edit", "tool_input": {}}`,
			stacks:       []string{"python"},
			expectedCmds: nil,
			expectedExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ranCmds []string
			var mu sync.Mutex
			mockRunner := func(dir string, cmd string) (string, error) {
				mu.Lock()
				defer mu.Unlock()
				ranCmds = append(ranCmds, cmd)
				return "", nil
			}

			var stderr bytes.Buffer
			exitCode := runPostEditWithStacks(strings.NewReader(tt.payload), &stderr, configuredCMakeRoot(t), tt.stacks, mockRunner)
			if exitCode != tt.expectedExit {
				t.Fatalf("expected exit %d, got %d", tt.expectedExit, exitCode)
			}
			if len(tt.expectedCmds) == 0 && len(ranCmds) != 0 {
				t.Fatalf("expected no commands, ran %v", ranCmds)
			}
			for _, exp := range tt.expectedCmds {
				found := false
				for _, ran := range ranCmds {
					if strings.Contains(ran, exp) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected command containing %q in %v", exp, ranCmds)
				}
			}
		})
	}
}

func TestRunPostEditFailure(t *testing.T) {
	payload := `{"tool_name": "Edit", "tool_input": {"file_path": "src/main.py"}}`
	mockFailRunner := func(dir string, cmd string) (string, error) {
		return "syntax error in file\n", errors.New("exit 1")
	}

	var stderr bytes.Buffer
	exitCode := runPostEditWithStacks(strings.NewReader(payload), &stderr, ".", []string{"python"}, mockFailRunner)
	if exitCode != ExitDenied {
		t.Fatalf("expected ExitDenied (2), got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "syntax error") {
		t.Fatalf("expected stderr output, got %q", stderr.String())
	}

	// Failure with empty out string
	mockEmptyFail := func(dir string, cmd string) (string, error) {
		return "", errors.New("generic error")
	}
	var stderr2 bytes.Buffer
	exitCode2 := runPostEditWithStacks(strings.NewReader(payload), &stderr2, ".", []string{"python"}, mockEmptyFail)
	if exitCode2 != ExitDenied {
		t.Fatalf("expected ExitDenied (2), got %d", exitCode2)
	}
}

func TestRunPostEditPayloadEdgeCases(t *testing.T) {
	mockRunner := func(dir string, cmd string) (string, error) {
		return "", nil
	}

	// 1. Invalid JSON
	var stderr bytes.Buffer
	if code := runPostEditWithStacks(strings.NewReader("invalid json"), &stderr, ".", []string{"python"}, mockRunner); code != ExitOK {
		t.Fatalf("expected ExitOK on invalid json, got %d", code)
	}

	// 2. Empty payload
	if code := runPostEditWithStacks(strings.NewReader(`{}`), &stderr, ".", []string{"python"}, mockRunner); code != ExitOK {
		t.Fatalf("expected ExitOK on empty payload, got %d", code)
	}

	// 3. notebook_path support
	notebookPayload := `{"tool_name": "NotebookEdit", "tool_input": {"notebook_path": "src/notebook.ipynb"}}`
	if code := runPostEditWithStacks(strings.NewReader(notebookPayload), &stderr, ".", []string{"python"}, mockRunner); code != ExitOK {
		t.Fatalf("expected ExitOK for notebook_path, got %d", code)
	}

	// 4. Explicit ignored extensions
	for _, ext := range []string{".txt", ".json", ".yaml", ".yml", ".toml", ".svg", ".png", ".jpg", ".jpeg", ".import", ".lock"} {
		payload := `{"tool_name": "Edit", "tool_input": {"file_path": "file` + ext + `"}}`
		if code := runPostEditWithStacks(strings.NewReader(payload), &stderr, ".", []string{"python"}, mockRunner); code != ExitOK {
			t.Fatalf("expected ExitOK for ignored ext %s, got %d", ext, code)
		}
	}
}

func TestDefaultCommandRunnerMissingCommand(t *testing.T) {
	// A tool the PATH does not answer costs the lane, not the run: no output
	// and no error.
	if out, err := commandRunner(exec.LookPath, io.Discard)(".", "command_that_definitely_does_not_exist_xyz123"); out != "" || err != nil {
		t.Fatalf("out %q, err %v: a missing tool skips its lane", out, err)
	}

	// Normal failure returning error. `cmd` is the Windows shell, and the lane
	// runner starts `sh` everywhere else, so this half of the test is about
	// the host it is measured on.
	if runtime.GOOS != "windows" {
		t.Skip("cmd is the Windows shell")
	}
	_, errFail := commandRunner(exec.LookPath, io.Discard)(".", "cmd /c exit 42")
	if errFail == nil {
		t.Fatal("expected error on exit 42")
	}
}

func TestIsWikiPath(t *testing.T) {
	tests := []struct {
		rawPath  string
		root     string
		wikiDir  string
		expected bool
	}{
		{"wiki/index.md", "", "wiki/", true},
		{"docs/wiki/page.md", "", "docs/wiki", true},
		{"mybundle/page.md", "", "mybundle", true},
		{"sub/mybundle/page.md", "", "mybundle", true},
		{"README.md", "", "wiki/", false},
		{"docs/README.md", "", "wiki/", false},
		{"wiki", "", "", true},
		{"/root/docs/wiki/page.md", "/root", "docs/wiki", true},
		{"/root/other/page.md", "/root", "docs/wiki", false},
	}

	for _, tt := range tests {
		got := isWikiPath(tt.rawPath, tt.root, tt.wikiDir)
		if got != tt.expected {
			t.Errorf("isWikiPath(%q, %q, %q) = %v, want %v", tt.rawPath, tt.root, tt.wikiDir, got, tt.expected)
		}
	}
}

func TestGetCommandsForStacksVariants(t *testing.T) {
	// 1. Rust
	rustCmds := getCommandsForStacks([]string{"rust"}, "rust", true, "src/main.rs", "", ".", "wiki")
	if len(rustCmds) != 2 || !strings.Contains(rustCmds[0].text, "cargo clippy") {
		t.Fatalf("expected cargo clippy, got %v", rustCmds)
	}

	// 2. Go
	goCmds := getCommandsForStacks([]string{"go"}, "go", true, "main.go", "", ".", "wiki")
	if len(goCmds) != 1 || !strings.Contains(goCmds[0].text, "go vet") {
		t.Fatalf("expected go vet, got %v", goCmds)
	}

	// 3. GDScript without target
	gdCmds := getCommandsForStacks([]string{"gdscript"}, "gdscript", false, "", "", ".", "wiki")
	if len(gdCmds) != 1 || !strings.Contains(gdCmds[0].text, "gdlint .") {
		t.Fatalf("expected gdlint ., got %v", gdCmds)
	}

	// 4. CPP without target
	cppCmds := getCommandsForStacks([]string{"cpp"}, "cpp", false, "", "", ".", "wiki")
	if len(cppCmds) != 2 || !strings.Contains(cppCmds[0].text, "clang-format -i") {
		t.Fatalf("expected clang-format -i, got %v", cppCmds)
	}

	// 5. TypeScript with target in nested dir
	tsNested := getCommandsForStacks([]string{"typescript"}, "typescript", true, "frontend/src/app.ts", "", ".", "wiki")
	if len(tsNested) != 2 || !strings.Contains(tsNested[0].text, "npx --prefix frontend eslint") {
		t.Fatalf("expected npx --prefix frontend eslint, got %v", tsNested)
	}

	// 6. TypeScript root without target
	tsRootNoTarget := getCommandsForStacks([]string{"typescript"}, "typescript", false, "", "", ".", "wiki")
	if len(tsRootNoTarget) != 2 || !strings.Contains(tsRootNoTarget[0].text, "npx eslint --cache .") {
		t.Fatalf("expected npx eslint --cache ., got %v", tsRootNoTarget)
	}

	// 7. Vue without target in root
	vueRoot := getCommandsForStacks([]string{"vue"}, "vue", false, "", "", ".", "wiki")
	if len(vueRoot) != 1 || !strings.Contains(vueRoot[0].text, "npx vue-tsc --noEmit") {
		t.Fatalf("expected npx vue-tsc --noEmit, got %v", vueRoot)
	}

	// 8. Svelte without target in root
	svelteRoot := getCommandsForStacks([]string{"svelte"}, "svelte", false, "", "", ".", "wiki")
	if len(svelteRoot) != 1 || !strings.Contains(svelteRoot[0].text, "npx svelte-check") {
		t.Fatalf("expected npx svelte-check, got %v", svelteRoot)
	}

	// 9. CSS variants
	cssNested := getCommandsForStacks([]string{"css"}, "css", true, "frontend/src/styles.css", "", ".", "wiki")
	if len(cssNested) != 1 || !strings.Contains(cssNested[0].text, "npx --prefix frontend stylelint") {
		t.Fatalf("expected npx --prefix frontend stylelint, got %v", cssNested)
	}
	cssNoTarget := getCommandsForStacks([]string{"css"}, "css", false, "", "", ".", "wiki")
	if len(cssNoTarget) != 1 || !strings.Contains(cssNoTarget[0].text, "npx stylelint \"**/*.{css,scss}\"") {
		t.Fatalf("expected glob stylelint, got %v", cssNoTarget)
	}

	// 10. HTML variant without target
	htmlNoTarget := getCommandsForStacks([]string{"html"}, "html", false, "", "", ".", "wiki")
	if len(htmlNoTarget) != 1 || !strings.Contains(htmlNoTarget[0].text, "npx htmlhint \"**/*.html\"") {
		t.Fatalf("expected glob htmlhint, got %v", htmlNoTarget)
	}

	// 11. Shell variant without target
	shNoTarget := getCommandsForStacks([]string{"shell"}, "shell", false, "", "", ".", "wiki")
	if len(shNoTarget) != 1 || !strings.Contains(shNoTarget[0].text, "shellcheck **/*.sh") {
		t.Fatalf("expected glob shellcheck, got %v", shNoTarget)
	}

	// 12. SQL variant without target
	sqlNoTarget := getCommandsForStacks([]string{"sql"}, "sql", false, "", "", ".", "wiki")
	if len(sqlNoTarget) != 1 || !strings.Contains(sqlNoTarget[0].text, "sqlfluff lint .") {
		t.Fatalf("expected sqlfluff lint ., got %v", sqlNoTarget)
	}

	// 13. The wiki lane. It carries a target because it has no other shape,
	// and its text names loomux itself: the page is linted in this process, so
	// the text is a label for the report rather than a command line.
	wikiLane := getCommandsForStacks([]string{"wiki"}, "wiki", true, "wiki/concept.md", "", ".", "wiki")
	if len(wikiLane) != 1 || wikiLane[0].text != "loomux lint wiki/concept.md" {
		t.Fatalf("expected loomux lint wiki/concept.md, got %v", wikiLane)
	}

	// 14. Python with pyright (plain without uv)
	pyrightPlain := getCommandsForStacks([]string{"python", "pyright"}, "python", true, "src/main.py", "", ".", "wiki")

	if len(pyrightPlain) != 2 || !strings.Contains(pyrightPlain[1].text, "pyright") || strings.Contains(pyrightPlain[1].text, "uv run") {
		t.Fatalf("expected plain pyright, got %v", pyrightPlain)
	}
	pyrightUV := getCommandsForStacks([]string{"python", "pyright", "uv"}, "python", true, "src/main.py", "", ".", "wiki")

	if len(pyrightUV) != 2 || !strings.Contains(pyrightUV[1].text, "uv run pyright") {
		t.Fatalf("expected uv run pyright, got %v", pyrightUV)
	}

	// 15. Vue nested
	vueNested := getCommandsForStacks([]string{"vue"}, "vue", true, "frontend/src/Component.vue", "", ".", "wiki")
	if len(vueNested) != 1 || !strings.Contains(vueNested[0].text, "npm --prefix frontend run typecheck") {
		t.Fatalf("expected npm --prefix frontend run typecheck, got %v", vueNested)
	}

	// 16. Svelte nested
	svelteNested := getCommandsForStacks([]string{"svelte"}, "svelte", true, "frontend/src/App.svelte", "", ".", "wiki")
	if len(svelteNested) != 1 || !strings.Contains(svelteNested[0].text, "npm --prefix frontend run check") {
		t.Fatalf("expected npm --prefix frontend run check, got %v", svelteNested)
	}
}

// A file whose extension names no stack draws the full chain -- every lane the
// project has, because a gate that skips a check unnoticed is worse than none.
// The wiki lane is the one exception, and this is why: it lints one page and
// has no argument-less form. Where `sqlfluff lint .` and `shellcheck **/*.sh`
// still say something without a target, a wiki lint without a page has
// nothing to read -- so the lane that cannot ask its question stays out of the
// chain instead of poisoning it.
func TestWikiLaneStaysOutWithoutATarget(t *testing.T) {
	cmds := getCommandsForStacks([]string{"wiki"}, "", false, ".gitignore", "", ".", "wiki")

	for _, cmd := range cmds {
		if strings.HasPrefix(cmd.text, "loomux lint") {
			t.Fatalf("expected no argument-less wiki lint, got %v", cmds)
		}
	}
}

// gdlint reads .gdlintrc from the working directory upwards, so where the
// check starts decides which rules it applies. A project that keeps its Godot
// tree under godot/ and is checked from the repository root gets no
// configuration at all: the exclusion list stays unread and the whole of
// addons/ is linted on default limits, which is how a single edit turns into
// thousands of findings.
func TestGdscriptRunsWhereTheGodotTreeStands(t *testing.T) {
	cmds := getCommandsForStacks([]string{"gdscript"}, "gdscript", false, "", "godot", ".", "wiki")

	if len(cmds) != 1 {
		t.Fatalf("expected one command, got %v", cmds)
	}
	if cmds[0].dir != "godot" {
		t.Fatalf("dir = %q, want %q", cmds[0].dir, "godot")
	}
	if cmds[0].text != "gdlint ." {
		t.Fatalf("text = %q, want %q", cmds[0].text, "gdlint .")
	}
}

// The target arrives named from the repository root, and the check runs one
// directory down -- so the path has to lose that first segment or gdlint looks
// for godot/godot/ui/system/system_view.gd.
func TestGdscriptTargetIsRelativeToTheGodotTree(t *testing.T) {
	cmds := getCommandsForStacks([]string{"gdscript"}, "gdscript", true, "godot/ui/system/system_view.gd", "godot", ".", "wiki")

	if len(cmds) != 1 || cmds[0].dir != "godot" {
		t.Fatalf("expected one command in godot/, got %v", cmds)
	}
	if cmds[0].text != "gdlint ui/system/system_view.gd" {
		t.Fatalf("text = %q, want %q", cmds[0].text, "gdlint ui/system/system_view.gd")
	}
}

// Every other lane keeps running at the root: they either read their
// configuration from a file they are told about or carry their limits in the
// command line, so moving them would change what they check for no gain.
func TestOtherLanesStayAtTheRoot(t *testing.T) {
	cmds := getCommandsForStacks([]string{"go"}, "go", true, "main.go", "godot", ".", "wiki")

	if len(cmds) != 1 || cmds[0].dir != "" {
		t.Fatalf("expected go vet at the root, got %v", cmds)
	}
}

// The lane-availability tests. `defaultCommandRunner` used to answer this
// question by matching the console's own words -- "is not recognized as an
// internal or external command" -- and a German Windows prints "ist entweder
// falsch geschrieben oder konnte nicht gefunden werden" instead. So the
// escape hatch never opened here, and every edit to a file with an unmapped
// extension blocked with exit 2 on a machine without shellcheck. The
// replacement asks the PATH, which speaks no language.
func TestLaneTool(t *testing.T) {
	tests := []struct {
		command string
		want    string
	}{
		{"shellcheck **/*.sh", "shellcheck"},
		{"npx --prefix frontend eslint app.ts", "npx"},
		{"uv run brain lint file.md", "uv"},
		{"go vet ./...", "go"},
		{"   spaced   out  ", "spaced"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := laneTool(tt.command); got != tt.want {
			t.Errorf("laneTool(%q) = %q, want %q", tt.command, got, tt.want)
		}
	}
}

func TestLaneAvailable(t *testing.T) {
	found := func(string) (string, error) { return "/usr/bin/shellcheck", nil }
	missing := func(string) (string, error) { return "", errors.New("executable file not found in %PATH%") }

	if !laneAvailable("shellcheck **/*.sh", found) {
		t.Error("a lane whose tool is on PATH is available")
	}
	if laneAvailable("shellcheck **/*.sh", missing) {
		t.Error("a lane whose tool is not on PATH is unavailable")
	}
	// An empty text names no tool, so there is nothing to look up and nothing
	// to refuse. Answering "unavailable" would drop a lane over a bug
	// somewhere else and hide it.
	if !laneAvailable("", missing) {
		t.Error("a lane with no tool is not refused for a missing tool")
	}
}

// The whole point, measured against the real PATH rather than a stub: a tool
// that is not installed costs the lane and not the exit code.
func TestDefaultCommandRunnerSkipsMissingTool(t *testing.T) {
	out, err := commandRunner(exec.LookPath, io.Discard)("", "ulguard-no-such-tool-8f3a --version")
	if err != nil {
		t.Fatalf("a missing tool must not fail the hook, got %v", err)
	}
	if out != "" {
		t.Fatalf("a skipped lane says nothing here, got %q", out)
	}
}

// A lane that never started must not pass in silence. Exit 0 stays -- a
// missing optional tool blocks nothing -- but the line names which lane was
// dropped and which tool was missing, so the gap is visible at the moment it
// opens rather than only in `ulguard status`.
func TestCommandRunnerNamesTheSkippedLane(t *testing.T) {
	missing := func(string) (string, error) { return "", errors.New("executable file not found in %PATH%") }
	var notice strings.Builder

	out, err := commandRunner(missing, &notice)("", "shellcheck **/*.sh")

	if err != nil {
		t.Fatalf("a missing tool must not fail the hook, got %v", err)
	}
	if out != "" {
		t.Fatalf("a skipped lane contributes no output, got %q", out)
	}
	said := notice.String()
	if !strings.Contains(said, "shellcheck") {
		t.Errorf("the notice names the missing tool, got %q", said)
	}
	if !strings.Contains(said, "shellcheck **/*.sh") {
		t.Errorf("the notice names the lane that was dropped, got %q", said)
	}
}

// The counterpart: a lane whose tool is there says nothing extra.
func TestCommandRunnerIsSilentWhenTheToolIsThere(t *testing.T) {
	var notice strings.Builder

	_, _ = commandRunner(exec.LookPath, &notice)("", "go version")

	if said := notice.String(); said != "" {
		t.Errorf("a lane that ran writes no notice, got %q", said)
	}
}

// Where the notice has to land to be seen at all.
//
// A PostToolUse hook that exits 0 has neither stdout nor stderr read by
// anybody: both go to the debug log. The one channel left open at exit 0 is a
// JSON document on stdout, whose `systemMessage` reaches the model. So the
// skipped lane is named there -- exit 0 keeps the edit unblocked, and the gap
// still arrives somewhere.
func TestPostEditNamesASkippedLaneWhereItIsRead(t *testing.T) {
	missing := func(string) (string, error) { return "", errors.New("executable file not found in %PATH%") }
	payload := `{"tool_input":{"file_path":"x.sh"}}`
	var stdout, stderr bytes.Buffer

	code := runPostEditWithContext(
		strings.NewReader(payload), &stdout, &stderr, t.TempDir(),
		[]string{"shell"}, "wiki/", "",
		func(notice io.Writer) CommandRunner { return commandRunner(missing, notice) },
	)

	if code != ExitOK {
		t.Fatalf("a missing optional tool blocks no edit, got exit %d", code)
	}
	// Decoded into a generic map and walked by key, not into the struct the
	// code wrote: a decode into that struct passes whatever field the code
	// chose and says nothing about where the harness looks.
	var said map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &said); err != nil {
		t.Fatalf("stdout has to be one JSON document, got %q: %v", stdout.String(), err)
	}
	specific, ok := said["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatalf("the document carries no hookSpecificOutput: %q", stdout.String())
	}
	if specific["hookEventName"] != "PostToolUse" {
		t.Errorf("the document names its event, got %v", specific["hookEventName"])
	}
	// The field this repository's own Claude adapter writes for the model.
	context, ok := specific["additionalContext"].(string)
	if !ok {
		t.Fatalf("the notice is not at hookSpecificOutput.additionalContext: %q", stdout.String())
	}
	if !strings.Contains(context, "shellcheck") {
		t.Errorf("the message names the missing tool, got %q", context)
	}
	if _, stray := specific["systemMessage"]; stray {
		t.Errorf("systemMessage is no field of the PostToolUse envelope: %q", stdout.String())
	}
}

// The ordinary run writes nothing to stdout: invalid JSON there would turn a
// green hook into a hook-error notice, so an empty channel is the only safe
// silence.
func TestPostEditWritesNoDocumentWhenEveryLaneRan(t *testing.T) {
	ran := func(_ string, _ string) (string, error) { return "", nil }
	payload := `{"tool_input":{"file_path":"x.sh"}}`
	var stdout, stderr bytes.Buffer

	runPostEditWithContext(
		strings.NewReader(payload), &stdout, &stderr, t.TempDir(),
		[]string{"shell"}, "wiki/", "",
		func(io.Writer) CommandRunner { return ran },
	)

	if stdout.Len() != 0 {
		t.Errorf("a run with nothing to report says nothing, got %q", stdout.String())
	}
}

// The production factory, exercised through the one door that builds it: a
// real lane, on a project whose tool is certain to be there. The go lane runs
// `go vet ./...`, so the root needs a module and a file that compiles.
func TestPostToolUseRunsTheGoLaneThroughItsOwnRunner(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module sample\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("package sample\n\nfunc Sample() int { return 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	payload := `{"tool_name":"Edit","tool_input":{"file_path":"sample.go"}}`
	var stdout, stderr bytes.Buffer
	if code := PostToolUse(strings.NewReader(payload), &stdout, &stderr, root); code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	// A lane whose tool is missing is skipped, and a skipped lane says so on
	// stdout -- so silence is what tells a lane that ran and passed from one
	// that never started.
	if stdout.Len() != 0 {
		t.Fatalf("the lane did not run: %s", stdout.String())
	}
}

// A path that does not start in the area is handed back untouched: an edit
// outside the Godot tree still names a real file from the root, and guessing
// at it would be worse than checking the wrong limits.
func TestRelativeToAreaLeavesAPathOutsideTheAreaAlone(t *testing.T) {
	if got := relativeToArea("tools/build.gd", "godot"); got != "tools/build.gd" {
		t.Fatalf("got %q", got)
	}
}

// The wiki directory itself, named absolutely: the path carries no separator
// after the directory's name, so only the form relative to the root answers.
func TestIsWikiPathMatchesTheWikiDirectoryItself(t *testing.T) {
	root := t.TempDir()
	if !isWikiPath(filepath.Join(root, "notes"), root, "notes") {
		t.Fatal("the wiki directory itself is a wiki path")
	}
}

// A wiki page is linted in this process: no shell lane starts for it, and a
// page the lint refuses still blocks the edit.
func TestTheWikiLaneRunsInProcess(t *testing.T) {
	root := t.TempDir()
	writeWikiPage(t, root, "page.md", "no frontmatter at all\n")

	var stderr bytes.Buffer
	shell := func(io.Writer) CommandRunner {
		return func(dir, command string) (string, error) {
			t.Fatalf("no shell lane may run for a wiki page, got %q", command)
			return "", nil
		}
	}
	input := `{"tool_name":"Edit","tool_input":{"file_path":"docs/wiki/page.md"}}`
	code := runPostEditWithContext(strings.NewReader(input), io.Discard, &stderr, root, []string{"wiki"}, "docs/wiki", "", shell)
	if code != ExitDenied || !strings.Contains(stderr.String(), "page.md") {
		t.Fatalf("code %d, err %q", code, stderr.String())
	}
}

// Claude names the edited file absolutely, so the lane joins the root only
// under a relative path -- a join on an absolute one would name the root twice.
func TestTheWikiLaneTakesAnAbsoluteTargetAsItIs(t *testing.T) {
	root := t.TempDir()
	page := writeWikiPage(t, root, "page.md", "no frontmatter at all\n")

	var stderr bytes.Buffer
	input := `{"tool_name":"Edit","tool_input":{"file_path":` + asJSON(t, page) + `}}`
	code := runPostEditWithContext(strings.NewReader(input), io.Discard, &stderr, root, []string{"wiki"}, "docs/wiki", "", noShell)
	if code != ExitDenied || !strings.Contains(stderr.String(), "page.md") {
		t.Fatalf("code %d, err %q", code, stderr.String())
	}
}

// A page the lint passes leaves the edit alone and says nothing.
func TestTheWikiLaneLetsACleanPageThrough(t *testing.T) {
	root := t.TempDir()
	writeWikiPage(t, root, "sample.md", "---\ntitle: Sample Concept\ntype: concept\ndescription: A valid OKF test document\n---\n\n# Sample Concept\nThis is a test concept.\n")

	var stderr bytes.Buffer
	input := `{"tool_name":"Edit","tool_input":{"file_path":"docs/wiki/sample.md"}}`
	code := runPostEditWithContext(strings.NewReader(input), io.Discard, &stderr, root, []string{"wiki"}, "docs/wiki", "", noShell)
	if code != ExitOK || stderr.String() != "" {
		t.Fatalf("code %d, err %q", code, stderr.String())
	}
}

// noShell is the runner factory for a run that must not reach a shell.
func noShell(io.Writer) CommandRunner {
	return func(dir, command string) (string, error) { return "", nil }
}

func writeWikiPage(t *testing.T, root, name, body string) string {
	t.Helper()
	dir := filepath.Join(root, "docs", "wiki")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(dir, name)
	if err := os.WriteFile(page, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return page
}

// asJSON quotes a path the way a hook payload carries it, backslashes and all.
func asJSON(t *testing.T, value string) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func writeManifest(t *testing.T, root, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The manifest is the one place a loomux project names its wiki.
func TestTheWikiDirectoryComesFromTheManifest(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"notes\"\n")
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got := wikiDirFor(root); got != "notes" {
		t.Fatalf("got %q", got)
	}
}

// Nothing declared and nothing on disk: the default stands.
func TestTheWikiDirectoryFallsBackToTheDefault(t *testing.T) {
	if got := wikiDirFor(t.TempDir()); got != "wiki/" {
		t.Fatalf("got %q", got)
	}
}

// A declared wiki with no directory behind it: wiki.Root answers nothing, and
// detection has the say.
func TestTheWikiDirectoryFallsBackToTheDetectedWiki(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[wiki]\n")

	if got := wikiDirFor(root); got != "wiki/" {
		t.Fatalf("got %q, want the detected wiki", got)
	}
}

// A neighbour wiki lies beside the project, so the answer leaves the root --
// which is the place wiki.Root found and the one the lane has to read.
func TestTheWikiDirectoryCanNameANeighbour(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "iam_backend")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(parent, "iam_wiki"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got := wikiDirFor(root); got != "../iam_wiki" {
		t.Fatalf("got %q", got)
	}
}

// A loomux project declares its wiki as `[layout] wiki`, and detection does
// not read that key: it knows `[wiki]`, `wiki = true` and `okf_version` only.
// Without this the lane never fired in the pilot's own repository.
func TestTheLayoutWikiAddsTheWikiStack(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"notes\"\n")
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(root, "notes", "page.md")
	if err := os.WriteFile(page, []byte("no frontmatter at all\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	input := `{"tool_name":"Edit","tool_input":{"file_path":` + asJSON(t, page) + `}}`
	if code := PostToolUse(strings.NewReader(input), &stdout, &stderr, root); code != ExitDenied {
		t.Fatalf("code %d, err %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "page.md") {
		t.Fatalf("err %q", stderr.String())
	}
}

// The precedence wikiDirFor documents has to be the one the caller uses.
//
// A manifest carrying both a `[wiki]` table and a `[layout] wiki` elsewhere
// made detection answer first -- `wiki/`, which nobody created -- and the lane
// then judged the edited page against a directory the manifest does not mean.
// The page lies in the declared bundle and is broken, so the lane has to
// refuse.
func TestTheDeclaredWikiOutranksTheDetectedOne(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n[wiki]\n[layout]\nwiki = \"notes\"\n")
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(root, "notes", "page.md")
	if err := os.WriteFile(page, []byte("no frontmatter at all\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	input := `{"tool_name":"Edit","tool_input":{"file_path":` + asJSON(t, page) + `}}`
	if code := PostToolUse(strings.NewReader(input), &stdout, &stderr, root); code != ExitDenied {
		t.Fatalf("code %d, err %q: the lane read the detected wiki, not the declared one", code, stderr.String())
	}
}

// A manifest without that key, and a directory without a manifest, declare
// nothing.
func TestAProjectWithoutALayoutWikiDeclaresNothing(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n")

	if declaresWikiLayout(root) {
		t.Fatal("a manifest without [layout] wiki declares no wiki")
	}
	if declaresWikiLayout(filepath.Join(root, "nowhere")) {
		t.Fatal("a directory without a manifest declares no wiki")
	}
}

// A declared layout that names no directory is no declaration either: wiki.Root
// passes it by, and the lane would read a place the manifest does not mean.
func TestALayoutWikiWithoutADirectoryDeclaresNothing(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"notes\"\n")

	if declaresWikiLayout(root) {
		t.Fatal("a layout nobody created declares no wiki")
	}
}

// A layout that leaves the repository is refused by WikiLayout, and the
// refusal arrives here as "nothing declared".
func TestAnInvalidLayoutWikiDeclaresNothing(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"../elsewhere\"\n")

	if declaresWikiLayout(root) {
		t.Fatal("a layout outside the repository declares no wiki")
	}
}

func TestTargetCommandsForStacks(t *testing.T) {
	// Broad (no target)
	broadCmds := TargetCommandsForStacks([]string{"python", "go"}, "", "", "", "")
	if len(broadCmds) != 3 { // ruff, mypy, go vet
		t.Fatalf("expected 3 commands for broad python+go, got %v", broadCmds)
	}

	// Targeted python
	pyCmds := TargetCommandsForStacks([]string{"python", "go"}, "foo.py", "", "", "")
	if len(pyCmds) != 2 {
		t.Fatalf("expected 2 commands for foo.py, got %v", pyCmds)
	}

	// Targeted wiki outside wiki
	wikiOutside := TargetCommandsForStacks([]string{"wiki"}, "other/foo.md", "", "/repo", "wiki")
	if len(wikiOutside) != 0 {
		t.Fatalf("expected 0 commands for wiki outside wiki, got %v", wikiOutside)
	}

	// Targeted wiki inside wiki
	wikiDir := t.TempDir()
	wikiInside := TargetCommandsForStacks([]string{"wiki"}, filepath.Join(wikiDir, "page.md"), "", t.TempDir(), wikiDir)
	if len(wikiInside) != 1 {
		t.Fatalf("expected 1 command for wiki inside wiki, got %v", wikiInside)
	}
}

// A C++ checkout that was never configured has no build tree, and cmake then
// fails on the missing directory for every edit. That is a precondition the
// edit cannot fix, so the lane is skipped out loud, like a missing tool.
func TestPostEditSkipsTheCMakeLaneWithoutAConfiguredBuild(t *testing.T) {
	for _, configured := range []bool{false, true} {
		root := t.TempDir()
		if configured {
			if err := os.MkdirAll(filepath.Join(root, "build"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "build", "CMakeCache.txt"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		var mu sync.Mutex
		var ran []string
		record := func(_ string, command string) (string, error) {
			mu.Lock()
			defer mu.Unlock()
			ran = append(ran, command)
			return "", nil
		}
		var stdout, stderr bytes.Buffer

		code := runPostEditWithContext(
			strings.NewReader(`{"tool_input":{"file_path":"x.cpp"}}`), &stdout, &stderr, root,
			[]string{"cpp"}, "wiki/", "",
			func(io.Writer) CommandRunner { return record },
		)

		if code != ExitOK {
			t.Fatalf("configured=%v: exit %d, stderr %q", configured, code, stderr.String())
		}
		cmake := false
		for _, c := range ran {
			if strings.HasPrefix(c, "cmake --build build") {
				cmake = true
			}
		}
		if cmake != configured {
			t.Errorf("configured=%v: cmake lane ran=%v, commands %v", configured, cmake, ran)
		}
		named := strings.Contains(stdout.String(), "build/CMakeCache.txt")
		if named == configured {
			t.Errorf("configured=%v: the skip is named only when it happens, stdout %q", configured, stdout.String())
		}
	}
}

// configuredCMakeRoot is a project root whose C++ build tree has been
// configured, so the cmake lane has what it needs to run.
func configuredCMakeRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "build"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "build", "CMakeCache.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}
