package hooks

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

const stopGateWant = "the stop gate's own controls are not written by the party it gates"

// shellReasons is what the guard says about line, which Bash and
// PowerShell must say alike.
func shellReasons(t *testing.T, root, line string, policy config.Policy) []string {
	t.Helper()
	bash := checkTool(root, "Bash", map[string]any{"command": line}, policy)
	if power := checkTool(root, "PowerShell", map[string]any{"command": line}, policy); !slices.Equal(bash, power) {
		t.Errorf("%q: Bash %q, PowerShell %q", line, bash, power)
	}
	return bash
}

// A project's path rules hold for the shell as for a writing tool.
func TestAProjectPathRuleHoldsForTheShell(t *testing.T) {
	root := t.TempDir()
	policy := config.Policy{Paths: []config.PathRule{
		{Match: []string{"notes.txt"}, Reason: "notes are a human's"},
		{Match: []string{"bin/*"}, Reason: "build output"},
	}}
	for line, want := range map[string][]string{
		"echo x > notes.txt":            {"notes are a human's"},
		"rm -f notes.txt":               {"notes are a human's"},
		"mv bin/new.exe bin/loomux.exe": {"build output"},
		"rm -rf bin":                    {"build output"},
	} {
		if got := shellReasons(t, root, line, policy); !slices.Equal(got, want) {
			t.Errorf("%q: reasons %q, want %q", line, got, want)
		}
	}
	for _, line := range []string{"cat notes.txt", "go build -o bin/loomux.exe ./cmd/loomux"} {
		if got := shellReasons(t, root, line, policy); len(got) != 0 {
			t.Errorf("%q: reasons %q, want none", line, got)
		}
	}
}

// The command rules and the path rules answer one call together, each
// reason once, the command's first.
func TestACallNamesEachReasonOnce(t *testing.T) {
	root := t.TempDir()
	got := shellReasons(t, root, "rm .loomux/config.toml; tee .loomux/config.toml; git push", config.Policy{})
	want := []string{"Whether commits reach the remote is a human's decision.", manifestReason}
	if !slices.Equal(got, want) {
		t.Fatalf("reasons %q, want %q", got, want)
	}
	got = checkTool(root, "Write", map[string]any{"file_path": ".env", "notebook_path": ".env"}, config.Policy{})
	if !slices.Equal(got, []string{"secrets are not written by an agent"}) {
		t.Fatalf("a writing tool with the same target twice: %q", got)
	}
}

// manifestShellWrites write or remove the manifest, each in its own way.
func manifestShellWrites() []string {
	return []string{
		"cat >> .loomux/config.toml <<'EOF'\n[policy]\nEOF",
		"echo x >.loomux/config.toml",
		`echo x > "./.loomux/config.toml"`,
		`echo x > .loomux\config.toml`,
		"echo x >| /repo/.loomux/config.toml",
		"echo x 2>&1 >> .LOOMUX/Config.toml",
		"sed -i 's/a/b/' .loomux/config.toml",
		"sed -i.bak 's/a/b/' .loomux/config.toml",
		"sed -Ei 's/a/b/' .loomux/config.toml",
		"sed --in-place -e 's/a/b/' .loomux/config.toml",
		"sed 's/a/b/' .loomux/config.toml -i",
		"/usr/bin/sed -i x .loomux/config.toml",
		"perl -pi -e 's/a/b/' .loomux/config.toml",
		"printf x | tee -a .loomux/config.toml",
		"git status; tee .loomux/config.toml < x",
		"sudo tee .loomux/config.toml",
		"Set-Content -Path .loomux/config.toml -Value x",
		`"x" | Out-File .loomux\config.toml`,
		`Out-File -FilePath:.loomux\config.toml`,
		`Add-Content .loomux/config.toml "x"`,
		"Clear-Content .loomux/config.toml",
		"'x' | Tee-Object -FilePath .loomux/config.toml",
		"[IO.File]::WriteAllText('.loomux/config.toml', 'x')",
		"cp other.toml .loomux/config.toml",
		"cp -f other.toml '.loomux/config.toml'",
		"mv tmp .loomux/config.toml && ls",
		"mv .loomux/config.toml elsewhere.toml",
		"rm .loomux/config.toml",
		"Remove-Item .loomux\\config.toml",
		"Copy-Item -Destination .loomux\\config.toml -Path x.toml",
		"Copy-Item x.toml .loomux\\config.toml",
		"git checkout -- .loomux/config.toml",
		"dd if=x of=.loomux/config.toml",
		"x=$(sed -i s/a/b/ .loomux/config.toml)",
	}
}

// manifestShellReads read the manifest, or write beside it.
func manifestShellReads() []string {
	return []string{
		"cat .loomux/config.toml",
		"grep policy .loomux/config.toml",
		"Get-Content .loomux/config.toml",
		"sed -n '1,5p' .loomux/config.toml",
		"sed -n '/min/p' .loomux/config.toml",
		"cp .loomux/config.toml backup.toml",
		"Copy-Item .loomux\\config.toml -Destination backup.toml",
		"cat .loomux/config.toml > out.txt",
		"echo x > other.toml; cat .loomux/config.toml",
		"echo x > .loomux/config.toml.bak",
		"echo x > my.loomux/config.toml",
		"grep tee .loomux/config.toml",
		"git show HEAD:.loomux/config.toml",
		"dd if=.loomux/config.toml of=copy.toml",
		"sed -i s/a/b/ other.toml; cat .loomux/config.toml",
	}
}

// runFileShellWrites write or remove a run file, with the reasons each gets.
// The flips against the old command rule are marked.
func runFileShellWrites() map[string][]string {
	runs := []string{runFilesWant}
	// Flip: .loomux/state holds the stop gate's hooks as well as the runs.
	state := []string{stopGateWant, runFilesWant}
	return map[string][]string{
		"echo x >> .loomux/state/runs/0001.jsonl":                  runs,
		"rm .loomux/state/runs/0001.flow":                          runs,
		`Remove-Item .loomux\state\runs\0001.jsonl`:                runs,
		"sed -i s/a/b/ .loomux/state/runs/0001.jsonl":              runs,
		"rm -r .loomux/state/runs":                                 runs,
		"git rm .loomux/state/runs/0001.jsonl":                     runs,
		"cp other.jsonl ./.loomux/state/runs/0001.jsonl":           runs,
		"Set-Content -Path .loomux/state/runs/0001.jsonl -Value x": runs,
		`echo x > ".LOOMUX/State/Runs/0001.jsonl"`:                 runs,
		`rd -Recurse .loomux\state\runs`:                           runs,
		`rmdir /s /q .loomux\state\runs`:                           runs,
		"[IO.Directory]::Delete('.loomux/state/runs', $true)":      runs,
		"git clean -fdx .loomux/state/runs":                        runs,
		"Remove-Item -Recurse .loomux/state":                       state,
		"rm -r .loomux/state/":                                     state,
		// The world holds .loomux/state/runs alone, so the glob reaches it alone.
		"rm -r .loomux/state/*":      runs,
		`del /s /q .loomux\state\r*`: runs,
		"git rm -r .loomux/state":    state,
		// Flip: removing the hooks folder was allowed.
		"rm -r .loomux/state/hooks": {stopGateWant},
		// Flip: removing .loomux was a hole. The armed lanes go with the
		// folder, so their reason stands beside the manifest's.
		"rm -rf .loomux": {stopGateWant, manifestReason, armedReason, runFilesWant},
	}
}

// runFileShellReads read the run files, or copy into the folder above them.
func runFileShellReads() []string {
	return []string{
		"cat .loomux/state/runs/0001.jsonl",
		"ls .loomux/state/runs",
		"cp .loomux/state/runs/0001.jsonl backup.jsonl",
		"echo x > .loomux/state/runsx",
		"grep answered .loomux/state/runs/0001.jsonl > out.txt",
		"cp -r x .loomux/state/",
		"ls .loomux/state",
	}
}

// flowShellWrites write or remove a protected flow's folder, with the flows
// each reaches; the old command rule named both flows in one reason.
func flowShellWrites() map[string][]string {
	example, review := []string{bundledWant("example")}, []string{bundledWant("review")}
	both := []string{bundledWant("example"), bundledWant("review")}
	return map[string][]string{
		"rm -r .loomux/flows/example":                            example,
		"git rm .loomux/flows/review/questions/q.md":             review,
		"echo x > .loomux/flows/Example/flow.toml":               example,
		`Remove-Item -Recurse .loomux\flows\example`:             example,
		"cp x.toml ./.loomux/flows/example/flow.toml":            example,
		`sed -i s/a/b/ ".LOOMUX/FLOWS/REVIEW/instructions/r.md"`: review,
		`rmdir /s /q .loomux\flows\review`:                       review,
		`rd -Recurse .loomux\flows\example`:                      example,
		"[IO.Directory]::Delete('.loomux/flows/review', $true)":  review,
		"git clean -fd .loomux/flows/example":                    example,
		"rm -r .loomux/flows":                                    both,
		"rm -r .loomux/flows/":                                   both,
		"rm -r .loomux/flows/*":                                  both,
		`Remove-Item -Recurse .loomux\flows\e*`:                  example,
		"git rm -r .loomux/flows":                                both,
		`del /s /q .loomux\flows\*\flow.toml`:                    both,
		"cp evil.md .loomux/flows/ex*/instructions/draft.md":     example,
		"cp evil.md .loomux/flows/exampl?/instructions/draft.md": example,
	}
}

// flowShellReads leave the protected flows alone.
func flowShellReads() []string {
	return []string{
		"rm -r .loomux/flows/mine",
		"rm -r .loomux/flows/mine/*",
		"cat .loomux/flows/example/flow.toml",
		"echo x > .loomux/flows/example2/flow.toml",
		"cp -r .loomux/flows/example .loomux/flows/mine",
		"cp x .loomux/flows/mine/instructions/a.md",
		"cp -r mine .loomux/flows/",
		"ls .loomux/flows",
		"git status",
	}
}

// flowWorld is a project whose protected flows are example (the catalog)
// and review (an override), both on disk beside a flow of the agent's own.
func flowWorld(t *testing.T) string {
	t.Helper()
	catalog(t, "example")
	root := project(t)
	manifest(t, root, "[flow]\noverrides = [\"review\"]\n")
	mkfile(t, root, ".loomux/flows/example/flow.toml")
	mkfile(t, root, ".loomux/flows/example/instructions/draft.md")
	mkfile(t, root, ".loomux/flows/review/flow.toml")
	mkfile(t, root, ".loomux/flows/mine/flow.txt")
	return root
}

func TestTheShellPathCheckKeepsTheManifestBaseline(t *testing.T) {
	root := t.TempDir()
	for _, line := range manifestShellWrites() {
		if got := shellReasons(t, root, line, config.Policy{}); !slices.Equal(got, []string{manifestReason}) {
			t.Errorf("%q: reasons %q, want the manifest's", line, got)
		}
	}
	for _, line := range manifestShellReads() {
		if got := shellReasons(t, root, line, config.Policy{}); len(got) != 0 {
			t.Errorf("%q: reasons %q, want none", line, got)
		}
	}
}

func TestTheShellPathCheckKeepsTheRunFileBaseline(t *testing.T) {
	catalog(t)
	root := t.TempDir()
	mkfile(t, root, ".loomux/state/runs/0001.jsonl")
	for line, want := range runFileShellWrites() {
		if got := shellReasons(t, root, line, config.Policy{}); !slices.Equal(got, want) {
			t.Errorf("%q: reasons %q, want %q", line, got, want)
		}
	}
	for _, line := range runFileShellReads() {
		if got := shellReasons(t, root, line, config.Policy{}); len(got) != 0 {
			t.Errorf("%q: reasons %q, want none", line, got)
		}
	}
}

func TestTheShellPathCheckKeepsTheFlowBaseline(t *testing.T) {
	root := flowWorld(t)
	for line, want := range flowShellWrites() {
		if got := shellReasons(t, root, line, config.Policy{}); !slices.Equal(got, want) {
			t.Errorf("%q: reasons %q, want %q", line, got, want)
		}
	}
	for _, line := range flowShellReads() {
		if got := shellReasons(t, root, line, config.Policy{}); len(got) != 0 {
			t.Errorf("%q: reasons %q, want none", line, got)
		}
	}
}

// Every gap of the review is refused in the default mode, and the lines a
// careless reader would refuse stay open.
func TestTheShellPathCheckClosesTheGaps(t *testing.T) {
	root := flowWorld(t)
	mkfile(t, root, ".loomux/state/runs/0001.jsonl")
	mkfile(t, root, "build/a.txt")
	mkfile(t, root, "src/main.go")
	example := bundledWant("example")
	for line, want := range map[string]string{
		"ln -s evil .loomux/config.toml":                                           manifestReason,
		"tar -xf evil.tar -C .loomux/flows/example":                                example,
		"unzip evil.zip -d .loomux/flows/example":                                  example,
		`Expand-Archive evil.zip -DestinationPath .loomux\flows\example`:           example,
		"New-Item -Path .loomux/config.toml -Force":                                manifestReason,
		"ni .loomux/state/runs/0002.flow":                                          runFilesWant,
		"touch .loomux/state/runs/0002.flow":                                       runFilesWant,
		`robocopy evil .loomux\flows\example /MIR`:                                 example,
		`xcopy evil .loomux\flows\example /E /I`:                                   example,
		"rsync -a evil/ .loomux/flows/example/":                                    example,
		"find .loomux/state/runs -delete":                                          runFilesWant,
		"find .loomux/flows/example -exec rm {} +":                                 example,
		"curl -sSLo .loomux/config.toml https://example.invalid/x":                 manifestReason,
		"wget -O .loomux/config.toml https://example.invalid/x":                    manifestReason,
		"Invoke-WebRequest https://example.invalid/x -OutFile .loomux/config.toml": manifestReason,
		"echo x > .loomux/sta*/runs/0001.jsonl":                                    runFilesWant,
		"rm -rf .loomux/sta*/runs":                                                 runFilesWant,
		"cp evil .loomux/flows/{example,zz}/flow.toml":                             example,
		"sudo -u root tee .loomux/config.toml":                                     manifestReason,
		"xargs -n 1 rm .loomux/config.toml":                                        manifestReason,
		"timeout -s KILL 60 rm .loomux/config.toml":                                manifestReason,
		"rm -rf .loomux":                              manifestReason,
		"mv .loomux/flows elsewhere":                  example,
		"rm -r .loomux/state":                         runFilesWant,
		"echo x > ../sibling/.loomux/config.toml":     manifestReason,
		"echo x > /repo/.loomux/config.toml":          manifestReason,
		`echo x > C:\repo\.loomux\config.toml`:        manifestReason,
		"cd .loomux && rm config.toml":                manifestReason,
		`git commit -m "a; rm .loomux/config.toml b"`: manifestReason,
		// A named limit: the line is cut at its breaks, a heredoc body too.
		"cat > notes.md <<'EOF'\nrm -rf .loomux\nEOF": manifestReason,
		// Without a name filter find removes its start paths; git clean
		// without a path the root.
		"find . -delete": manifestReason,
		"git clean -fdx": manifestReason,
		// With one, what lies on disk and matches it.
		"find . -name config.toml -delete":         manifestReason,
		"find . -name '*.jsonl' -delete":           runFilesWant,
		"gci . -Recurse -Include *.toml | ri":      manifestReason,
		"find . ! -name '*.orig' -delete":          manifestReason,
		"find . -regex '.*orig' -delete":           manifestReason,
		"find . -name '*' -delete":                 manifestReason,
		"find . -name '*.orig' -o -type f -delete": manifestReason,
		// A name test after the action narrows nothing: find acts first.
		"find . -delete -name '*.orig'":           manifestReason,
		"find . -exec rm -rf {} + -name '*.orig'": manifestReason,
		// find reads \x as x.
		`find . -name 'config.tom\l' -delete`:  manifestReason,
		`find . -name 'config\.toml' -delete`:  manifestReason,
		`find .loomux -name '*\.toml' -delete`: manifestReason,
		`find . -name 'x\' -delete`:            manifestReason,
		// A -Filter with a three-letter extension matches 8.3 short names too.
		"gci . -Recurse -Filter *.tom | ri": manifestReason,
		// A substitution in front of a fixed tail keeps the tail.
		"echo x > $(pwd)/.loomux/config.toml":                                  manifestReason,
		"rm -rf $(pwd)/.loomux/state/runs":                                     runFilesWant,
		"rm -rf $(git rev-parse --show-toplevel)/.loomux":                      manifestReason,
		"echo x > $(dirname $(pwd))/x/.loomux/config.toml":                     manifestReason,
		"echo x > `pwd`/.loomux/config.toml":                                   manifestReason,
		`eval "rm .loomux/config.toml"`:                                        manifestReason,
		"eval rm .loomux/config.toml":                                          manifestReason,
		"Set-Content -Path (Join-Path $PWD '.loomux/config.toml')":             manifestReason,
		`Set-Content (Join-Path -Path . -ChildPath ".loomux" "config.toml") x`: manifestReason,
	} {
		if got := shellReasons(t, root, line, config.Policy{}); !slices.Contains(got, want) {
			t.Errorf("%q: reasons %q, want %q among them", line, got, want)
		}
	}
	for _, line := range []string{
		"cp -r mine .loomux/flows/",
		"mv mine .loomux/flows/",
		"cat .loomux/config.toml",
		"ls .loomux/state/runs",
		"git status",
		"git checkout feat/x",
		"git checkout -b x",
		"git restore --staged x",
		"git clean -n",
		"git clean --dry-run -d",
		"rm -rf build/*",
		"rm -rf *",
		"rm -rf build",
		`git commit -m "fix(guard): keep .loomux/state/runs"`,
		"echo x 2>&1",
		"echo x >&2",
		"echo x > /dev/null",
		"echo x 2>nul",
		"echo x > $null",
		"find . -name x",
		"echo $(date) > notes.txt",
		"find . -name '*.orig' -delete",
		"find . -name '*.tmp' -delete",
		"find . -type d -name __pycache__ -exec rm -rf {} +",
		"find . -name '*.orig' | xargs rm",
		"gci . -Recurse -Include *.tmp | ri",
		"Set-Content -Path (Join-Path $PWD 'notes.txt')",
	} {
		if got := shellReasons(t, root, line, config.Policy{}); len(got) != 0 {
			t.Errorf("%q: reasons %q, want none", line, got)
		}
	}
}

// A glob is matched against the disk, not against a rule: rm -rf build/*
// passes beside *.pem until a key lies in build.
func TestAGlobIsJudgedByWhatItMatches(t *testing.T) {
	root := t.TempDir()
	mkfile(t, root, "build/a.txt")
	if got := shellReasons(t, root, "rm -rf build/*", config.Policy{}); len(got) != 0 {
		t.Fatalf("no key in build: %q", got)
	}
	mkfile(t, root, "build/server.pem")
	if got := shellReasons(t, root, "rm -rf build/*", config.Policy{}); !slices.Equal(got, []string{"secrets are not written by an agent"}) {
		t.Fatalf("a key in build: %q", got)
	}
}

// A [flow] that will not read may name any folder under .loomux/flows, so
// removing that folder is refused for it, and a removal elsewhere is not.
func TestRemovingTheFlowsFolderUnderAnUnreadableFlowTableIsRefused(t *testing.T) {
	root := project(t)
	manifest(t, root, "[flow]\noverrides = 5\n")
	const prefix = "loomux cannot read [flow] of .loomux/config.toml, so it refuses writes under .loomux/flows: "
	got := shellReasons(t, root, "rm -r .loomux/flows", config.Policy{})
	if len(got) != 1 || !strings.HasPrefix(got[0], prefix) {
		t.Errorf("the flows folder: reasons %q", got)
	}
	if got := shellReasons(t, root, "rm -r build", config.Policy{}); len(got) != 0 {
		t.Errorf("elsewhere: reasons %q", got)
	}
}

func TestTooManyBraceVariantsRefuse(t *testing.T) {
	got := shellReasons(t, t.TempDir(), "touch x{1..65}", config.Policy{})
	if len(got) != 1 || !strings.Contains(got[0], "more than 64 brace variants") {
		t.Fatalf("reasons %q", got)
	}
}

func TestBelowFindsTheFolderAboveAKeptPath(t *testing.T) {
	for _, row := range []struct {
		rel, glob  string
		fold, want bool
	}{
		{".", ".aws/**", true, true},
		{".", "*.pem", true, false},
		{".loomux", "**/.loomux/config.toml", true, true},
		{"x/.loomux", "**/.loomux/config.toml", true, true},
		// The kept file itself, under any directory, is no folder above it.
		{"x/.loomux/config.toml", "**/.loomux/config.toml", true, false},
		{"../sib/.loomux/state", "**/.loomux/state/runs/**", true, true},
		// The folder whose content the rule keeps is above that content.
		{".loomux/state/runs", "**/.loomux/state/runs/**", true, true},
		{"build", "**/.loomux/config.toml", true, false},
		{"build", "*.pem", true, false},
		{"docs", "docs/*.md", false, true},
		{"x/docs", "docs/*.md", false, false},
		{"x", "*/y.md", false, false},
		{".LOOMUX", "**/.loomux/config.toml", true, true},
		{".LOOMUX", ".loomux/state/**", false, false},
		// A folder of the name a glob under any directory keeps the content of.
		{"x/secrets", "**/secrets/*", false, true},
	} {
		if got := below(row.rel, row.glob, row.fold); got != row.want {
			t.Errorf("below(%q, %q, %v) = %v, want %v", row.rel, row.glob, row.fold, got, row.want)
		}
	}
	if literalPrefix("a/b*/c") != "a" || literalPrefix("*.pem") != "" || literalPrefix("a/b") != "a/b" {
		t.Fatal("literalPrefix")
	}
}
