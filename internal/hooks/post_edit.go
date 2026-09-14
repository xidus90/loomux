package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/detect"
)

type CommandRunner func(dir string, cmd string) (string, error)

// A command is a check plus the directory it has to run in.
//
// The directory is not decoration: gdlint reads .gdlintrc from the working
// directory upwards, so a lane started above the project's own tree runs on
// defaults and applies limits nobody in that project wrote down. An empty dir
// means the root, which is where every other lane belongs.
type command struct {
	dir  string
	text string
}

func at(dir, text string) command { return command{dir: dir, text: text} }

func root(text string) command { return command{text: text} }

// The one child in this program that keeps the inherited environment: no
// gitenv.Environ here, and on purpose. A PostToolUse hook is not a git hook,
// so nothing git exported is in this environment to begin with, and the
// commands built above are linters and compilers -- ruff, gdlint, go vet,
// npx, cmake -- none of which asks git about a repository. The strip belongs
// where a git question is asked out of an inherited environment: worktree.go
// for this program's own git calls, and `without_location` in
// `src/ultraloom/process.py` for the children the Python check lane spawns.
// commandRunner builds the runner from its two outside edges: what answers
// the PATH, and where a dropped lane is named.
//
// Both are parameters because the interesting case has no output of its own.
// A lane whose tool is missing is skipped -- exit 0 stays, and a missing
// optional tool blocks no edit -- but it is skipped out loud: one line naming
// the lane and the tool. Silence here was the older answer, with `ulguard
// status` as the only place that said so; a check that never started then
// looked exactly like one that passed, and on 2026-09-12 a PATH that had lost
// `~/go/bin` took two lanes down that way without a word.
func commandRunner(look func(string) (string, error), notice io.Writer) CommandRunner {
	return func(dir, command string) (string, error) {
		// Before the shell, so a missing tool costs the lane and not the exit
		// code.
		if !laneAvailable(command, look) {
			fmt.Fprintf(notice, "loomux hook post-tool-use: lane skipped, %q is not on PATH: %s\n", laneTool(command), command)
			return "", nil
		}
		return runLane(dir, command)
	}
}

//coverage:exempt the sh arm runs only where runtime.GOOS is not windows, and this suite is measured on Windows
func runLane(dir, command string) (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// laneTool is the executable a lane command starts with, or "" for a text
// that names none.
func laneTool(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// laneAvailable answers whether a lane can run at all, by asking the PATH.
//
// This used to be decided after the fact, by matching the console's own words
// against "is not recognized as an internal or external command". A German
// Windows prints "ist entweder falsch geschrieben oder konnte nicht gefunden
// werden", so the escape hatch never opened on this machine: every edit to a
// file with an unmapped extension ran the whole chain -- which is the
// deliberate fallback of the selective-dispatch design -- reached the shell
// lane without shellcheck installed, and blocked the edit with exit 2. The
// PATH speaks no language.
//
// A text with no tool is available, not refused. There is nothing to look up,
// and answering "unavailable" would drop a lane over a bug elsewhere and hide
// it.
func laneAvailable(command string, look func(string) (string, error)) bool {
	tool := laneTool(command)
	if tool == "" {
		return true
	}
	_, err := look(tool)
	return err == nil
}

var explicitIgnoredExtensions = map[string]bool{
	".txt":    true,
	".json":   true,
	".yaml":   true,
	".yml":    true,
	".toml":   true,
	".svg":    true,
	".png":    true,
	".jpg":    true,
	".jpeg":   true,
	".import": true,
	".lock":   true,
}

var extensionStackMap = map[string]string{
	".py":     "python",
	".gd":     "gdscript",
	".cpp":    "cpp",
	".hpp":    "cpp",
	".cc":     "cpp",
	".cxx":    "cpp",
	".c":      "cpp",
	".h":      "cpp",
	".ts":     "typescript",
	".tsx":    "typescript",
	".js":     "typescript",
	".jsx":    "typescript",
	".vue":    "vue",
	".svelte": "svelte",
	".css":    "css",
	".scss":   "css",
	".sass":   "css",
	".less":   "css",
	".html":   "html",
	".htm":    "html",
	".sh":     "shell",
	".bash":   "shell",
	".zsh":    "shell",
	".sql":    "sql",
	".go":     "go",
	".rs":     "rust",
	".md":     "wiki",
}

func PostToolUse(stdin io.Reader, stdout io.Writer, stderr io.Writer, root string) int {
	facts := detect.Detect(os.DirFS(root))
	wikiDir := facts.WikiPath
	if wikiDir == "" {
		wikiDir = resolveWikiDir(root)
	}
	return runPostEditWithContext(stdin, stdout, stderr, root, facts.Stacks, wikiDir, facts.GodotDir, defaultRunnerFor)
}

// defaultRunnerFor is the production factory: the runner has to be built
// around the notice writer this run owns, so it is built here rather than
// handed in.
func defaultRunnerFor(notice io.Writer) CommandRunner {
	return commandRunner(exec.LookPath, notice)
}

func runPostEditWithStacks(stdin io.Reader, stderr io.Writer, root string, stacks []string, runner CommandRunner) int {
	return runPostEditWithContext(stdin, io.Discard, stderr, root, stacks, "wiki/", "", func(io.Writer) CommandRunner { return runner })
}

func runPostEditWithContext(stdin io.Reader, stdout io.Writer, stderr io.Writer, root string, stacks []string, wikiDir string, godotDir string, runnerFor func(io.Writer) CommandRunner) int {
	var payload HookPayload
	if err := json.NewDecoder(stdin).Decode(&payload); err != nil {
		return ExitOK
	}

	rawPath, ok := payload.ToolInput["file_path"].(string)
	if !ok || rawPath == "" {
		rawPath, _ = payload.ToolInput["notebook_path"].(string)
	}
	if rawPath == "" {
		return ExitOK
	}

	ext := strings.ToLower(filepath.Ext(rawPath))
	if explicitIgnoredExtensions[ext] {
		return ExitOK
	}

	targetStack, hasTarget := extensionStackMap[ext]
	if targetStack == "wiki" && !isWikiPath(rawPath, root, wikiDir) {
		return ExitOK
	}
	commands := getCommandsForStacks(stacks, targetStack, hasTarget, rawPath, godotDir)

	var wg sync.WaitGroup
	var mu sync.Mutex
	hasFailure := false

	// Every lane writes its notice here rather than straight out: the
	// goroutines run side by side, and one document at the end is the only
	// shape a reader can parse.
	var notices lockedBuilder
	runner := runnerFor(&notices)

	for _, cmd := range commands {
		wg.Add(1)
		go func(c command) {
			defer wg.Done()
			out, err := runner(filepath.Join(root, c.dir), c.text)
			if err != nil {
				mu.Lock()
				defer mu.Unlock()
				hasFailure = true
				io.WriteString(stderr, out)
				if out == "" {
					io.WriteString(stderr, err.Error()+"\n")
				}
			}
		}(cmd)
	}
	wg.Wait()

	if hasFailure {
		return ExitDenied
	}
	reportSkipped(stdout, notices.String())
	return ExitOK
}

// lockedBuilder is a strings.Builder every lane may write to at once.
type lockedBuilder struct {
	mu      sync.Mutex
	builder strings.Builder
}

func (b *lockedBuilder) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.builder.Write(p)
}

func (b *lockedBuilder) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.builder.String()
}

// reportSkipped puts the dropped lanes where a PostToolUse hook exiting 0 is
// actually read.
//
// Neither stdout nor stderr is shown to anybody at exit 0 -- both end in the
// debug log. A JSON document on stdout is the exception: its `systemMessage`
// reaches the model. Nothing is written when nothing was skipped, because
// stdout that is not valid JSON turns a passed hook into a hook-error notice.
//
//coverage:exempt json.Marshal cannot fail on a map of strings
func reportSkipped(stdout io.Writer, notices string) {
	if notices == "" {
		return
	}
	document := map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName": "PostToolUse",
			"systemMessage": strings.TrimRight(notices, "\n"),
		},
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		// Unreachable for a map of strings, and silent on purpose: a hook that
		// cannot phrase its aside must not fail the edit over it.
		return
	}
	fmt.Fprintf(stdout, "%s\n", encoded)
}

var standardSrcDirs = map[string]bool{
	"src":    true,
	"lib":    true,
	"pkg":    true,
	"cmd":    true,
	"tests":  true,
	"test":   true,
	"dist":   true,
	"build":  true,
	"public": true,
}

func getWorkspaceDir(targetPath string, hasTarget bool) string {
	if !hasTarget || targetPath == "" {
		return ""
	}
	norm := strings.TrimPrefix(filepath.ToSlash(targetPath), "./")
	parts := strings.Split(norm, "/")
	if len(parts) > 1 && parts[0] != "." && parts[0] != "" && !standardSrcDirs[parts[0]] {
		return parts[0]
	}
	return ""
}

//coverage:exempt the workspace arms without a target path cannot run: getWorkspaceDir answers "" unless there is one, so targetDir and an absent target exclude each other
func getCommandsForStacks(stacks []string, targetStack string, hasTarget bool, targetPath string, godotDir string) []command {
	var cmds []command
	has := func(stack string) bool {
		for _, s := range stacks {
			if s == stack {
				return true
			}
		}
		return false
	}
	shouldRun := func(stack string) bool {
		if !has(stack) {
			return false
		}
		if hasTarget && targetStack != "" {
			return targetStack == stack
		}
		return true
	}

	targetDir := getWorkspaceDir(targetPath, hasTarget)

	if shouldRun("python") {
		typeChecker := "mypy --no-error-summary --no-pretty"
		if has("pyright") {
			if has("uv") {
				typeChecker = "uv run pyright"
			} else {
				typeChecker = "pyright"
			}
		} else if has("uv") {
			typeChecker = "dmypy run -- --no-error-summary --no-pretty"
		}
		cmds = append(cmds, root("ruff check --output-format=concise ."), root(typeChecker))
	}
	if shouldRun("gdscript") {
		if hasTarget && targetPath != "" {
			cmds = append(cmds, at(godotDir, fmt.Sprintf("gdlint %s", relativeToArea(targetPath, godotDir))))
		} else {
			cmds = append(cmds, at(godotDir, "gdlint ."))
		}
	}
	if shouldRun("cpp") {
		if hasTarget && targetPath != "" {
			cmds = append(cmds, root(fmt.Sprintf("clang-format -i %s", targetPath)), root("cmake --build build --parallel"))
		} else {
			cmds = append(cmds, root("clang-format -i"), root("cmake --build build --parallel"))
		}
	}
	if shouldRun("typescript") {
		if targetDir != "" {
			if hasTarget && targetPath != "" {
				cmds = append(cmds,
					root(fmt.Sprintf("npx --prefix %s eslint --config %s/eslint.config.js --cache %s", targetDir, targetDir, targetPath)),
					root(fmt.Sprintf("npm --prefix %s run typecheck", targetDir)),
				)
			} else {
				cmds = append(cmds,
					root(fmt.Sprintf("npm --prefix %s run lint", targetDir)),
					root(fmt.Sprintf("npm --prefix %s run typecheck", targetDir)),
				)
			}
		} else {
			if hasTarget && targetPath != "" {
				cmds = append(cmds, root(fmt.Sprintf("npx eslint --cache %s", targetPath)), root("npx tsc --noEmit"))
			} else {
				cmds = append(cmds, root("npx eslint --cache ."), root("npx tsc --noEmit"))
			}
		}
	}
	if shouldRun("vue") {
		if targetDir != "" {
			cmds = append(cmds, root(fmt.Sprintf("npm --prefix %s run typecheck", targetDir)))
		} else {
			cmds = append(cmds, root("npx vue-tsc --noEmit"))
		}
	}
	if shouldRun("svelte") {
		if targetDir != "" {
			cmds = append(cmds, root(fmt.Sprintf("npm --prefix %s run check", targetDir)))
		} else {
			cmds = append(cmds, root("npx svelte-check"))
		}
	}
	if shouldRun("css") {
		if targetDir != "" && hasTarget && targetPath != "" {
			cmds = append(cmds, root(fmt.Sprintf("npx --prefix %s stylelint %s", targetDir, targetPath)))
		} else if hasTarget && targetPath != "" {
			cmds = append(cmds, root(fmt.Sprintf("npx stylelint %s", targetPath)))
		} else {
			cmds = append(cmds, root("npx stylelint \"**/*.{css,scss}\""))
		}
	}
	if shouldRun("html") {
		if hasTarget && targetPath != "" {
			cmds = append(cmds, root(fmt.Sprintf("npx htmlhint %s", targetPath)))
		} else {
			cmds = append(cmds, root("npx htmlhint \"**/*.html\""))
		}
	}
	if shouldRun("shell") {
		if hasTarget && targetPath != "" {
			cmds = append(cmds, root(fmt.Sprintf("shellcheck %s", targetPath)))
		} else {
			cmds = append(cmds, root("shellcheck **/*.sh"))
		}
	}
	if shouldRun("sql") {
		if hasTarget && targetPath != "" {
			cmds = append(cmds, root(fmt.Sprintf("sqlfluff lint %s", targetPath)))
		} else {
			cmds = append(cmds, root("sqlfluff lint ."))
		}
	}
	if shouldRun("rust") {
		cmds = append(cmds, root("cargo clippy -- -D warnings"), root("cargo fmt --check"))
	}
	if shouldRun("go") {
		cmds = append(cmds, root("go vet ./..."))
	}
	// The one lane with no argument-less form, and therefore the one that
	// stays out when the chain runs wide. `sqlfluff lint .` and `shellcheck
	// **/*.sh` still name something without a target; `brain lint` reads a
	// single file, answers "file path required" to anything else, and fails
	// the run it was appended to. A lane that cannot ask its question is
	// silent rather than wrong.
	if shouldRun("wiki") && hasTarget && targetPath != "" {
		prefix := "brain lint"
		if has("uv") {
			prefix = "uv run brain lint"
		}
		cmds = append(cmds, root(fmt.Sprintf("%s %s", prefix, targetPath)))
	}

	return cmds
}

// relativeToArea renames a path the hook gave from the repository root into
// one the check can use from inside its own area.
//
// Without this the lane would look for godot/godot/ui/system/system_view.gd.
// A path that does not start in the area is handed back untouched: an edit
// outside the Godot tree still names a real file from the root, and guessing
// at it would be worse than checking the wrong limits.
func relativeToArea(targetPath, area string) string {
	if area == "" {
		return targetPath
	}
	norm := strings.TrimPrefix(filepath.ToSlash(targetPath), "./")
	prefix := filepath.ToSlash(area) + "/"
	if strings.HasPrefix(norm, prefix) {
		return strings.TrimPrefix(norm, prefix)
	}
	return targetPath
}

func resolveWikiDir(root string) string {
	if data, err := os.ReadFile(filepath.Join(root, ".ultraloom", "answers.toml")); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "bundle") && strings.Contains(trimmed, "=") {
				parts := strings.SplitN(trimmed, "=", 2)
				val := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
				if val != "" {
					return filepath.ToSlash(val)
				}
			}
		}
	}
	facts := detect.Detect(os.DirFS(root))
	if facts.WikiPath != "" {
		return filepath.ToSlash(facts.WikiPath)
	}
	return "wiki/"
}

func isWikiPath(rawPath, root, configuredWikiDir string) bool {
	wikiDir := strings.TrimSuffix(filepath.ToSlash(configuredWikiDir), "/")
	if wikiDir == "" {
		wikiDir = "wiki"
	}

	norm := strings.TrimPrefix(filepath.ToSlash(rawPath), "./")

	if strings.HasPrefix(norm, wikiDir+"/") || norm == wikiDir ||
		strings.Contains(norm, "/"+wikiDir+"/") ||
		strings.HasPrefix(norm, "docs/wiki/") || strings.HasPrefix(norm, "wiki/") {
		return true
	}

	if root != "" {
		rel, err := filepath.Rel(root, rawPath)
		if err == nil {
			relNorm := strings.TrimPrefix(filepath.ToSlash(rel), "./")
			if strings.HasPrefix(relNorm, wikiDir+"/") || relNorm == wikiDir ||
				strings.Contains(relNorm, "/"+wikiDir+"/") ||
				strings.HasPrefix(relNorm, "docs/wiki/") || strings.HasPrefix(relNorm, "wiki/") {
				return true
			}
		}
	}
	return false
}
