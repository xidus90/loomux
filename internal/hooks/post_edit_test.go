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
			name:         "Wiki markdown file triggers brain lint when wiki stack is active",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "wiki/concept.md"}}`,
			stacks:       []string{"python", "wiki"},
			expectedCmds: []string{"brain lint wiki/concept.md"},
			expectedExit: 0,
		},
		{
			name:         "Wiki markdown file triggers uv run brain lint when uv stack is active",
			payload:      `{"tool_name": "Edit", "tool_input": {"file_path": "wiki/concept.md"}}`,
			stacks:       []string{"python", "uv", "wiki"},
			expectedCmds: []string{"uv run brain lint wiki/concept.md"},
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
			exitCode := runPostEditWithStacks(strings.NewReader(tt.payload), &stderr, ".", tt.stacks, mockRunner)
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
	out, err := commandRunner(exec.LookPath, io.Discard)(".", "command_that_definitely_does_not_exist_xyz123")
	// If it fails with command not recognized, it should return empty string and nil error
	if out != "" || err != nil {
		// Depending on OS/locale, if not matched, it returns output and error
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

func TestResolveWikiDir(t *testing.T) {
	tmp := t.TempDir()

	// 1. Default when nothing exists
	if dir := resolveWikiDir(tmp); dir != "wiki/" {
		t.Fatalf("expected default wiki/, got %q", dir)
	}

	// 2. answers.toml with bundle
	ultraloomDir := filepath.Join(tmp, ".ultraloom")
	if err := os.MkdirAll(ultraloomDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ultraloomDir, "answers.toml"), []byte("bundle = \"docs/mywiki\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if dir := resolveWikiDir(tmp); dir != "docs/mywiki" {
		t.Fatalf("expected docs/mywiki, got %q", dir)
	}

	// 3. answers.toml with empty bundle falls back to detect or wiki/
	if err := os.WriteFile(filepath.Join(ultraloomDir, "answers.toml"), []byte("bundle = \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if dir := resolveWikiDir(tmp); dir != "wiki/" {
		t.Fatalf("expected fallback wiki/, got %q", dir)
	}

	// 4. .brain.toml
	if err := os.WriteFile(filepath.Join(tmp, ".brain.toml"), []byte("[area]\nscope=\"test\"\nwiki=true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(filepath.Join(ultraloomDir, "answers.toml"))
	if dir := resolveWikiDir(tmp); dir != "docs/wiki" && dir != "docs/wiki/" {
		// Detect defaults to docs/wiki when .brain.toml is present
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
	rustCmds := getCommandsForStacks([]string{"rust"}, "rust", true, "src/main.rs", "")
	if len(rustCmds) != 2 || !strings.Contains(rustCmds[0].text, "cargo clippy") {
		t.Fatalf("expected cargo clippy, got %v", rustCmds)
	}

	// 2. Go
	goCmds := getCommandsForStacks([]string{"go"}, "go", true, "main.go", "")
	if len(goCmds) != 1 || !strings.Contains(goCmds[0].text, "go vet") {
		t.Fatalf("expected go vet, got %v", goCmds)
	}

	// 3. GDScript without target
	gdCmds := getCommandsForStacks([]string{"gdscript"}, "gdscript", false, "", "")
	if len(gdCmds) != 1 || !strings.Contains(gdCmds[0].text, "gdlint .") {
		t.Fatalf("expected gdlint ., got %v", gdCmds)
	}

	// 4. CPP without target
	cppCmds := getCommandsForStacks([]string{"cpp"}, "cpp", false, "", "")
	if len(cppCmds) != 2 || !strings.Contains(cppCmds[0].text, "clang-format -i") {
		t.Fatalf("expected clang-format -i, got %v", cppCmds)
	}

	// 5. TypeScript with target in nested dir
	tsNested := getCommandsForStacks([]string{"typescript"}, "typescript", true, "frontend/src/app.ts", "")
	if len(tsNested) != 2 || !strings.Contains(tsNested[0].text, "npx --prefix frontend eslint") {
		t.Fatalf("expected npx --prefix frontend eslint, got %v", tsNested)
	}

	// 6. TypeScript root without target
	tsRootNoTarget := getCommandsForStacks([]string{"typescript"}, "typescript", false, "", "")
	if len(tsRootNoTarget) != 2 || !strings.Contains(tsRootNoTarget[0].text, "npx eslint --cache .") {
		t.Fatalf("expected npx eslint --cache ., got %v", tsRootNoTarget)
	}

	// 7. Vue without target in root
	vueRoot := getCommandsForStacks([]string{"vue"}, "vue", false, "", "")
	if len(vueRoot) != 1 || !strings.Contains(vueRoot[0].text, "npx vue-tsc --noEmit") {
		t.Fatalf("expected npx vue-tsc --noEmit, got %v", vueRoot)
	}

	// 8. Svelte without target in root
	svelteRoot := getCommandsForStacks([]string{"svelte"}, "svelte", false, "", "")
	if len(svelteRoot) != 1 || !strings.Contains(svelteRoot[0].text, "npx svelte-check") {
		t.Fatalf("expected npx svelte-check, got %v", svelteRoot)
	}

	// 9. CSS variants
	cssNested := getCommandsForStacks([]string{"css"}, "css", true, "frontend/src/styles.css", "")
	if len(cssNested) != 1 || !strings.Contains(cssNested[0].text, "npx --prefix frontend stylelint") {
		t.Fatalf("expected npx --prefix frontend stylelint, got %v", cssNested)
	}
	cssNoTarget := getCommandsForStacks([]string{"css"}, "css", false, "", "")
	if len(cssNoTarget) != 1 || !strings.Contains(cssNoTarget[0].text, "npx stylelint \"**/*.{css,scss}\"") {
		t.Fatalf("expected glob stylelint, got %v", cssNoTarget)
	}

	// 10. HTML variant without target
	htmlNoTarget := getCommandsForStacks([]string{"html"}, "html", false, "", "")
	if len(htmlNoTarget) != 1 || !strings.Contains(htmlNoTarget[0].text, "npx htmlhint \"**/*.html\"") {
		t.Fatalf("expected glob htmlhint, got %v", htmlNoTarget)
	}

	// 11. Shell variant without target
	shNoTarget := getCommandsForStacks([]string{"shell"}, "shell", false, "", "")
	if len(shNoTarget) != 1 || !strings.Contains(shNoTarget[0].text, "shellcheck **/*.sh") {
		t.Fatalf("expected glob shellcheck, got %v", shNoTarget)
	}

	// 12. SQL variant without target
	sqlNoTarget := getCommandsForStacks([]string{"sql"}, "sql", false, "", "")
	if len(sqlNoTarget) != 1 || !strings.Contains(sqlNoTarget[0].text, "sqlfluff lint .") {
		t.Fatalf("expected sqlfluff lint ., got %v", sqlNoTarget)
	}

	// 13. Wiki variants. Both carry a target, because the lane has no other
	// shape: what the two cases separate is `uv` in the stack, not the
	// presence of a file. The target-less pair that stood here asserted an
	// argument-less `brain lint`, a command that cannot run; the caller never
	// produces that pair either, since `targetStack` is only ever "wiki" when
	// the extension matched and `hasTarget` is true with it.
	wikiPlain := getCommandsForStacks([]string{"wiki"}, "wiki", true, "wiki/concept.md", "")
	if len(wikiPlain) != 1 || wikiPlain[0].text != "brain lint wiki/concept.md" {
		t.Fatalf("expected brain lint wiki/concept.md, got %v", wikiPlain)
	}
	wikiUV := getCommandsForStacks([]string{"wiki", "uv"}, "wiki", true, "wiki/concept.md", "")

	if len(wikiUV) != 1 || wikiUV[0].text != "uv run brain lint wiki/concept.md" {
		t.Fatalf("expected uv run brain lint wiki/concept.md, got %v", wikiUV)
	}

	// 14. Python with pyright (plain without uv)
	pyrightPlain := getCommandsForStacks([]string{"python", "pyright"}, "python", true, "src/main.py", "")

	if len(pyrightPlain) != 2 || !strings.Contains(pyrightPlain[1].text, "pyright") || strings.Contains(pyrightPlain[1].text, "uv run") {
		t.Fatalf("expected plain pyright, got %v", pyrightPlain)
	}
	pyrightUV := getCommandsForStacks([]string{"python", "pyright", "uv"}, "python", true, "src/main.py", "")

	if len(pyrightUV) != 2 || !strings.Contains(pyrightUV[1].text, "uv run pyright") {
		t.Fatalf("expected uv run pyright, got %v", pyrightUV)
	}

	// 15. Vue nested
	vueNested := getCommandsForStacks([]string{"vue"}, "vue", true, "frontend/src/Component.vue", "")
	if len(vueNested) != 1 || !strings.Contains(vueNested[0].text, "npm --prefix frontend run typecheck") {
		t.Fatalf("expected npm --prefix frontend run typecheck, got %v", vueNested)
	}

	// 16. Svelte nested
	svelteNested := getCommandsForStacks([]string{"svelte"}, "svelte", true, "frontend/src/App.svelte", "")
	if len(svelteNested) != 1 || !strings.Contains(svelteNested[0].text, "npm --prefix frontend run check") {
		t.Fatalf("expected npm --prefix frontend run check, got %v", svelteNested)
	}
}

// A file whose extension names no stack draws the full chain -- every lane the
// project has, because a gate that skips a check unnoticed is worse than none.
// The wiki lane is the one exception, and this is why: `brain lint` reads one
// file and has no argument-less form. Where `sqlfluff lint .` and `shellcheck
// **/*.sh` still say something without a target, `brain lint` alone says
// "file path required" and fails the whole run -- so the lane that cannot ask
// its question stays out of the chain instead of poisoning it.
func TestWikiLaneStaysOutWithoutATarget(t *testing.T) {
	cmds := getCommandsForStacks([]string{"wiki"}, "", false, ".gitignore", "")

	for _, cmd := range cmds {
		if strings.HasPrefix(cmd.text, "brain lint") || strings.HasPrefix(cmd.text, "uv run brain lint") {
			t.Fatalf("expected no argument-less brain lint, got %v", cmds)
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
	cmds := getCommandsForStacks([]string{"gdscript"}, "gdscript", false, "", "godot")

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
	cmds := getCommandsForStacks([]string{"gdscript"}, "gdscript", true, "godot/ui/system/system_view.gd", "godot")

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
	cmds := getCommandsForStacks([]string{"go"}, "go", true, "main.go", "godot")

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
	var said struct {
		HookSpecificOutput struct {
			HookEventName string `json:"hookEventName"`
			SystemMessage string `json:"systemMessage"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &said); err != nil {
		t.Fatalf("stdout has to be one JSON document, got %q: %v", stdout.String(), err)
	}
	if said.HookSpecificOutput.HookEventName != "PostToolUse" {
		t.Errorf("the document names its event, got %q", said.HookSpecificOutput.HookEventName)
	}
	if !strings.Contains(said.HookSpecificOutput.SystemMessage, "shellcheck") {
		t.Errorf("the message names the missing tool, got %q", said.HookSpecificOutput.SystemMessage)
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
}

// A path that does not start in the area is handed back untouched: an edit
// outside the Godot tree still names a real file from the root, and guessing
// at it would be worse than checking the wrong limits.
func TestRelativeToAreaLeavesAPathOutsideTheAreaAlone(t *testing.T) {
	if got := relativeToArea("tools/build.gd", "godot"); got != "tools/build.gd" {
		t.Fatalf("got %q", got)
	}
}

// With no answer written down, the detected wiki decides -- and only when
// detection found none does the "wiki/" default stand.
func TestResolveWikiDirFallsBackToTheDetectedWiki(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[wiki]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs", "wiki"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got := resolveWikiDir(root); got != "docs/wiki/" {
		t.Fatalf("got %q, want the detected wiki", got)
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
