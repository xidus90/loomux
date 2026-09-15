package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/brain/wiki"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/detect"
)

type CommandRunner func(dir string, cmd string) (string, error)

// A command is a check plus the directory it has to run in.
//
// The directory is not decoration: gdlint reads .gdlintrc from the working
// directory upwards, so a lane started above the project's own tree runs on
// defaults and applies limits nobody in that project wrote down. An empty dir
// means the root, which is where every other lane belongs.
//
// A lane with a run function is the exception to all of that: it asks its
// question inside this process and never reaches a shell, so neither the
// directory nor the PATH concerns it. The wiki lane is the one such lane.
type command struct {
	dir  string
	text string
	run  func() (string, error)
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
	// wikiDirFor and not detection first: the manifest's [layout] wiki outranks
	// what detection guesses, and wikiDirFor already falls back to detection and
	// then to wiki/. Asking detection first inverted that, so a manifest with
	// both a [wiki] table and a [layout] wiki lost the lane: detection answered
	// wiki/, a directory nobody created, and the edited page was judged to lie
	// outside the bundle.
	wikiDir := wikiDirFor(root)
	return runPostEditWithContext(stdin, stdout, stderr, root, stacksWithWiki(facts.Stacks, root), wikiDir, facts.GodotDir, defaultRunnerFor)
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
	commands := getCommandsForStacks(stacks, targetStack, hasTarget, rawPath, godotDir, root, wikiDir)

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
			var out string
			var err error
			if c.run != nil {
				out, err = c.run()
			} else {
				out, err = runner(filepath.Join(root, c.dir), c.text)
			}
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
// read.
//
// Plain stdout and stderr at exit 0 could not be traced to a reader from
// inside this repository, so nothing is claimed about them. A JSON document
// with `hookSpecificOutput.additionalContext` is the one field this repository
// can point at: it is what `writeClaudeContext` in internal/hosts/claude.go
// writes for the model, from the harness table of the superpowers port
// document. What stood here was `systemMessage`, a field no envelope in this
// repository defines and no adapter reads.
//
// Nothing is written when nothing was skipped, because stdout that is not
// valid JSON turns a passed hook into a hook-error notice.
//
//coverage:exempt json.Marshal cannot fail on a map of strings
func reportSkipped(stdout io.Writer, notices string) {
	if notices == "" {
		return
	}
	document := map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":     "PostToolUse",
			"additionalContext": strings.TrimRight(notices, "\n"),
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
func getCommandsForStacks(stacks []string, targetStack string, hasTarget bool, targetPath string, godotDir string, projectRoot string, wikiDir string) []command {
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
	// **/*.sh` still name something without a target; a wiki lint reads a
	// single page and has nothing to read without one. A lane that cannot ask
	// its question is silent rather than wrong.
	//
	// It is also the one lane that runs inside this process: the lint lives in
	// this binary (internal/brain/wiki), so the text below is a label for the
	// report rather than a command line. What it replaces is `uv run brain
	// lint <page>` -- a second binary, a Python environment and a process per
	// edit, for a check this program already carries.
	if shouldRun("wiki") && hasTarget && targetPath != "" {
		target := targetPath
		cmds = append(cmds, command{
			text: "loomux lint " + target,
			run: func() (string, error) {
				var report strings.Builder
				// Claude names the edited file absolutely; a relative target
				// comes from a caller that speaks from the root.
				page := target
				if !filepath.IsAbs(page) {
					page = filepath.Join(projectRoot, page)
				}
				wikiRoot := filepath.Join(projectRoot, filepath.FromSlash(wikiDir))
				if code := wiki.LintReport(page, wikiRoot, &report); code != 0 {
					return report.String(), fmt.Errorf("wiki lint found errors in %s", target)
				}
				return "", nil
			},
		})
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

// wikiDirFor answers where this project's wiki bundle is, seen from its root.
//
// The answer comes from the resolver the lint itself uses, wiki.Root: the
// manifest's [layout] wiki, then docs/wiki, then wiki, then a neighbour
// bundle. What stood here read `bundle` out of ultraloom's
// .ultraloom/answers.toml line by line -- a file loomux does not write, and a
// second opinion about the wiki beside the manifest's own.
//
// Detection still answers where wiki.Root found nothing: a project that
// declares a wiki it has not created yet keeps the place detection named for
// it, and "wiki/" is the last word.
func wikiDirFor(projectRoot string) string {
	if dir := wiki.Root(projectRoot); dir != "" {
		return relativeToRoot(projectRoot, dir)
	}
	if detected := detect.Detect(os.DirFS(projectRoot)).WikiPath; detected != "" {
		return filepath.ToSlash(detected)
	}
	return "wiki/"
}

//coverage:exempt filepath.Rel fails only across volumes, and wiki.Root builds every answer from projectRoot itself
func relativeToRoot(projectRoot, dir string) string {
	rel, err := filepath.Rel(projectRoot, dir)
	if err != nil {
		return filepath.ToSlash(dir)
	}
	return filepath.ToSlash(rel)
}

// declaresWikiLayout answers whether this project says it has a wiki through
// the one key loomux has for that, [layout] wiki.
//
// detect.Detect does not read that key -- it knows [wiki], `wiki = true` and
// an okf_version in the bundle -- so without this the wiki lane never started
// in a loomux project, the pilot's own repository first among them. The
// manifest's word alone is not enough: wiki.Root has to arrive at the declared
// place, or the lane would lint against a directory nobody created.
func declaresWikiLayout(projectRoot string) bool {
	manifest, err := config.ReadManifest(projectRoot)
	if err != nil {
		return false
	}
	layout, err := manifest.WikiLayout()
	if err != nil || layout == "" {
		return false
	}
	return wiki.Root(projectRoot) == filepath.Join(projectRoot, filepath.FromSlash(layout))
}

// stacksWithWiki adds the wiki lane to what detection found, wherever the
// manifest declares a wiki that detection cannot see.
func stacksWithWiki(stacks []string, projectRoot string) []string {
	if slices.Contains(stacks, "wiki") || !declaresWikiLayout(projectRoot) {
		return stacks
	}
	return append(slices.Clone(stacks), "wiki")
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
