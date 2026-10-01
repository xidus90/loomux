package hooks

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

var strictPolicy = config.Policy{Strict: true}

// command is a shell call's input.
func command(line string) map[string]any { return map[string]any{"command": line} }

// Strict mode knows loomux by what it is told: a copied or renamed binary
// answering a gate or writing the configuration is refused there, and passes
// the default mode, which knows loomux by its name.
func TestStrictModeKnowsLoomuxByItsArguments(t *testing.T) {
	root := project(t)
	has := func(reasons []string, part string) bool {
		return slices.ContainsFunc(reasons, func(r string) bool { return strings.Contains(r, part) })
	}
	for line, part := range map[string]string{
		"doc.exe flow resume 0001 --answer yes":     "a flow's gate asks a human",
		`.\x\doc.exe flow resume 0001 --answer=yes`: "a flow's gate asks a human",
		"./git.exe flow resume 0001 --answer yes":   "a flow's gate asks a human",
		"doc.exe config set a b":                    "a human runs them",
		"doc.exe init":                              "a human runs them",
		"doc.exe area add":                          "a human runs them",
		"doc.exe merge-hook install":                "a human runs them",
		"doc.exe dev switchover prune-hooks":        "a human runs them",
		// A file named like a word of the shell is a program like any other.
		"./if config set a b":                 "a human runs them",
		"./do init":                           "a human runs them",
		"/x/exec init":                        "a human runs them",
		"./command config set a b":            "a human runs them",
		"./function flow resume 1 --answer y": "a flow's gate asks a human",
	} {
		if got := checkTool(root, "Bash", command(line), strictPolicy); !has(got, part) {
			t.Errorf("strict %q: reasons %q, want %q", line, got, part)
		}
		if got := checkTool(root, "Bash", command(line), config.Policy{}); has(got, part) {
			t.Errorf("default %q: reasons %q", line, got)
		}
	}
	for _, line := range []string{
		"git config set user.name x", "gh config set editor vim", "npm init -y", "terraform init",
		"cat config", "echo config set a b", "ls init", "go mod init x",
	} {
		if got := checkTool(root, "Bash", command(line), strictPolicy); len(got) != 0 {
			t.Errorf("strict %q: reasons %q, want none", line, got)
		}
	}
}

// A program the guard does not know is refused on a protected path in strict
// mode; the read list and loomux's reading commands stay open.
func TestStrictModeRefusesAnUnknownProgramOnAProtectedPath(t *testing.T) {
	root := project(t)
	isStrict := func(r string) bool { return strings.HasPrefix(r, "in strict mode loomux refuses `") }
	for _, line := range []string{
		"frob .loomux/config.toml",
		"frob --in=.loomux/config.toml",
		"frob -o.loomux/config.toml",
		`sh -c "frob .loomux/config.toml"`,
		`bash -c 'cd .loomux && frob config.toml'`,
		// The words after the string are the script's $0, $1, …
		`bash -c 'frob "$1"' _ .loomux/config.toml`,
		`sh -c 'frob "$0"' .loomux/config.toml`,
		"git stash push .loomux/config.toml",
		"python edit.py .loomux/state/runs/0001.jsonl",
		// A file named like a word of the shell is a program like any other.
		"./function rm .loomux/config.toml",
		"./do .loomux/config.toml",
	} {
		if got := checkTool(root, "Bash", command(line), strictPolicy); !slices.ContainsFunc(got, isStrict) {
			t.Errorf("strict %q: reasons %q", line, got)
		}
		if got := checkTool(root, "Bash", command(line), config.Policy{}); len(got) != 0 {
			t.Errorf("default %q: reasons %q, want none", line, got)
		}
	}
	for _, line := range []string{
		"cat .loomux/config.toml", `type .loomux\config.toml`, "Get-Content .loomux/config.toml",
		"less .loomux/config.toml", "more .loomux/config.toml", "head -n 3 .loomux/config.toml",
		"tail .loomux/config.toml", "wc -l .loomux/config.toml", "stat .loomux/config.toml",
		"file .loomux/config.toml", "jq . .loomux/state/runs/x.json", "ls .loomux", "dir .loomux",
		"Get-ChildItem .loomux", "grep x .loomux/config.toml", "rg x .loomux",
		"Select-String x .loomux/config.toml", "diff .loomux/config.toml x",
		"git diff .loomux/config.toml", "git log -- .loomux/config.toml", "git show HEAD:.loomux/config.toml",
		"git status .loomux", "git blame .loomux/config.toml", "git add .loomux/config.toml",
		"git commit -m x .loomux/config.toml", "loomux flow list", "loomux flow show 0001",
		"loomux config get a", "loomux config list", "loomux config proposals", "loomux check precommit",
		"frob src/x", "go vet .", "echo .loomux/config.toml",
		// The resolved . of ./... and the : a stream cut leaves are no removal of the root.
		"go test ./...", "go vet ./...", "cut -d: -f1 f", "awk -F: '{print $1}' f",
		// A shell that runs a string is known; the line inside is judged.
		`bash -c "cat .loomux/config.toml"`, `powershell -Command "Get-Content .loomux/config.toml"`,
		`sh -c "sed -e x .loomux/config.toml > y"`,
	} {
		if got := checkTool(root, "Bash", command(line), strictPolicy); len(got) != 0 {
			t.Errorf("strict %q: reasons %q, want none", line, got)
		}
	}
}

// A write whose path holds an expansion is refused in strict mode where the
// fixed part before it may be, or lie above, a protected path.
func TestStrictModeRefusesAnExpansionThatMayLandOnAProtectedPath(t *testing.T) {
	catalog(t, "example")
	root := project(t)
	for _, line := range []string{
		"echo x > .loomux/$X",
		"echo x > .loomux/con${X}",
		"echo x > .e$X",
		"cp x .loomux/sta$X",
		"echo x > .loomux/flows/ex%X%/flow.toml",
		"echo x > ../sib/.loomux/$X",
		// Braces unfold and a glob cuts the fixed part as an expansion does.
		"echo x > .loomux/{con,x}$X",
		"echo x > .loo{mux,x}/con$X",
		"echo x > .loomux/c?n$X",
		// An absolute fixed part is weighed relative to the root as well.
		"echo x > " + filepath.ToSlash(root) + "/.aw$X",
	} {
		got := checkTool(root, "Bash", command(line), strictPolicy)
		if !slices.ContainsFunc(got, func(r string) bool { return strings.Contains(r, "the expansion may land on a protected path") }) {
			t.Errorf("strict %q: reasons %q", line, got)
		}
		if got := checkTool(root, "Bash", command(line), config.Policy{}); len(got) != 0 {
			t.Errorf("default %q: reasons %q, want none", line, got)
		}
	}
	// Past 64 variants the brace group is weighed as written, cut at its brace.
	overflow := checkTool(root, "Bash", command("echo x > .loomux/c{1..65}$X"), strictPolicy)
	if !slices.ContainsFunc(overflow, func(r string) bool { return strings.Contains(r, "the expansion may land on a protected path") }) {
		t.Errorf("strict past 64 variants: reasons %q", overflow)
	}
	for _, line := range []string{
		"echo x > $X", "echo x > $env:TMP/x", "echo x > build/$X.txt", "echo x > src/$X",
		"echo x > $null", "echo x 2>nul", "echo x > /dev/null", "echo x 2>&1", "echo x >&2",
	} {
		if got := checkTool(root, "Bash", command(line), strictPolicy); len(got) != 0 {
			t.Errorf("strict %q: reasons %q, want none", line, got)
		}
	}
}

// The globs an expansion may reach are the project's too, as it spelled
// them, and every flow folder while [flow] does not read.
func TestStrictModeWeighsAnExpansionAgainstEveryProtectedGlob(t *testing.T) {
	root := project(t)
	manifest(t, root, "[flow]\noverrides = 5\n")
	lands := func(reasons []string) bool {
		return slices.ContainsFunc(reasons, func(r string) bool { return strings.Contains(r, "the expansion may land on a protected path") })
	}
	policy := config.Policy{Strict: true, Paths: []config.PathRule{{Match: []string{"secret/**"}, Reason: "kept"}}}
	for line, want := range map[string]bool{
		"echo x > secret/$X":          true,
		"echo x > Secret/$X":          false,
		"echo x > .loomux/flows/m$X":  true,
		"echo x > build/flows/m$X":    false,
		"echo x > nested/secret/a$X":  false,
		"echo x > src/.loomux/flo$X/": true,
	} {
		if got := checkTool(root, "Bash", command(line), policy); lands(got) != want {
			t.Errorf("%q: reasons %q, want the expansion refused: %v", line, got, want)
		}
	}
}

func TestIsDeviceNamesTheWindowsDevices(t *testing.T) {
	for rel, want := range map[string]bool{"nul": true, "x/NUL.txt": true, "com1": true, "lpt9": true, "conout$": true,
		"com0": false, "comx": false, "nullx": false, "notes.txt": false} {
		if got := isDevice(rel); got != want {
			t.Errorf("isDevice(%q) = %v", rel, got)
		}
	}
}

// Strict mode adds refusals, never removes one: every default refusal of the
// baseline holds there too.
func TestStrictModeKeepsEveryDefaultRefusal(t *testing.T) {
	root := t.TempDir()
	for _, line := range manifestShellWrites() {
		if got := checkTool(root, "Bash", command(line), strictPolicy); !slices.Contains(got, manifestReason) {
			t.Errorf("%q: reasons %q", line, got)
		}
	}
}

// An unknown program's relative paths start where an earlier cd moved.
func TestStrictModeJudgesAnUnknownProgramBehindACd(t *testing.T) {
	root := project(t)
	policy := config.Policy{Strict: true, Paths: []config.PathRule{{Match: []string{"bin/*"}, Reason: "build output"}}}
	for line, want := range map[string]bool{
		"cd src && frob --out ../bin/app.exe": true,
		"cd src && frob ../.loomux/no-verify": true,
		"cd src && frob bin/app.exe":          false,
	} {
		got := checkTool(root, "Bash", command(line), policy)
		if refused := slices.ContainsFunc(got, func(r string) bool { return strings.HasPrefix(r, "in strict mode loomux refuses `frob`") }); refused != want {
			t.Errorf("%q: reasons %q, want refused: %v", line, got, want)
		}
	}
}
