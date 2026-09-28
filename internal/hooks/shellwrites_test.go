package hooks

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// spelled is how a test reads targets: w: for a write, rm: for a removal, the
// path with forward slashes, sorted and each once.
func spelled(targets []shellTarget) []string {
	var out []string
	for _, t := range targets {
		kind := "w:"
		if t.removes {
			kind = "rm:"
		}
		spelling := kind + strings.ReplaceAll(t.path, `\`, "/")
		for _, f := range t.filters {
			spelling += " where " + f.glob
		}
		out = append(out, spelling)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// Every verb of the table names the target it writes or removes. A row holds
// what the line must yield among its targets; the readings may add others
// (a cmd switch, a script word), which only ever refuse more.
func TestShellWritesReadsTheTargetsOfEveryVerb(t *testing.T) {
	root := t.TempDir()
	mkfile(t, root, "notes.txt")
	for line, want := range map[string][]string{
		"echo x > .loomux/config.toml":                                      {"w:.loomux/config.toml"},
		"echo x>.loomux/config.toml":                                        {"w:.loomux/config.toml"},
		"echo x &> .loomux/config.toml":                                     {"w:.loomux/config.toml"},
		"echo x 2>>.loomux/config.toml":                                     {"w:.loomux/config.toml"},
		"echo x >| .loomux/config.toml":                                     {"w:.loomux/config.toml"},
		"'x' *> .loomux/config.toml":                                        {"w:.loomux/config.toml"},
		"tee -a .loomux/config.toml":                                        {"w:.loomux/config.toml"},
		"Set-Content -Path .loomux/config.toml -Value x":                    {"w:.loomux/config.toml"},
		`Out-File -FilePath:.loomux\config.toml`:                            {"w:.loomux/config.toml"},
		"touch .loomux/state/runs/0001.flow":                                {"w:.loomux/state/runs/0001.flow"},
		"rm -rf .loomux":                                                    {"rm:.loomux"},
		"rm -- -x .loomux/config.toml":                                      {"rm:-x", "rm:.loomux/config.toml"},
		`rmdir /s /q .loomux\state\runs`:                                    {"rm:.loomux/state/runs"},
		"mv .loomux/flows elsewhere":                                        {"rm:.loomux/flows", "w:elsewhere"},
		"Move-Item x -Destination .loomux/config.toml":                      {"rm:x", "w:.loomux/config.toml"},
		`Rename-Item .loomux\x config.toml`:                                 {"rm:.loomux/x", "w:.loomux/config.toml"},
		"cp -r mine .loomux/flows/":                                         {"w:.loomux/flows/"},
		"cp -t .loomux/flows/example x":                                     {"w:.loomux/flows/example"},
		`Copy-Item x.toml -Dest .loomux\config.toml`:                        {"w:.loomux/config.toml"},
		"install -m 644 x .loomux/config.toml":                              {"w:.loomux/config.toml"},
		"rsync -a src/ .loomux/flows/example/":                              {"w:.loomux/flows/example/"},
		"ln -s evil .loomux/config.toml":                                    {"w:.loomux/config.toml"},
		"dd if=x of=.loomux/config.toml":                                    {"w:.loomux/config.toml"},
		"tar -xf evil.tar -C .loomux/flows/example":                         {"w:.loomux/flows/example"},
		"tar --extract --file evil.tar --directory=.loomux/flows/example":   {"w:.loomux/flows/example"},
		"tar -czf .loomux/config.toml src":                                  {"w:.loomux/config.toml"},
		"tar cf .loomux/config.toml src":                                    {"w:.loomux/config.toml"},
		"tar --create --file=.loomux/config.toml src":                       {"w:.loomux/config.toml"},
		"unzip evil.zip -d .loomux/flows/example":                           {"w:.loomux/flows/example"},
		`Expand-Archive evil.zip -DestinationPath .loomux\flows\example`:    {"w:.loomux/flows/example"},
		`robocopy evil .loomux\flows\example /MIR`:                          {"w:.loomux/flows/example"},
		`xcopy evil .loomux\flows\example /E /I`:                            {"w:.loomux/flows/example"},
		"New-Item -Path .loomux/config.toml -Force":                         {"w:.loomux/config.toml"},
		"New-Item -Path .loomux -Name config.toml":                          {"w:.loomux/config.toml"},
		"ni .loomux/state/runs/0002.flow":                                   {"w:.loomux/state/runs/0002.flow"},
		"curl -o .loomux/config.toml https://example.invalid/x":             {"w:.loomux/config.toml"},
		"curl -sSLo .loomux/config.toml https://example.invalid/x":          {"w:.loomux/config.toml"},
		"curl --output=.loomux/config.toml https://example.invalid/x":       {"w:.loomux/config.toml"},
		"wget -O .loomux/config.toml https://example.invalid/x":             {"w:.loomux/config.toml"},
		"Invoke-WebRequest https://example.invalid/x -OutFile .loomux/c.t":  {"w:.loomux/c.t"},
		"find .loomux/state -name '*.jsonl' -delete":                        {"rm:.loomux/state where *.jsonl"},
		"find . -iname '*.JSONL' -exec rm {} +":                             {"rm:. where *.JSONL"},
		"find . -path './.loomux/*/0001.*' -name x -delete":                 {"rm:. where ./.loomux/*/0001.* where x"},
		"curl --output-dir .loomux --remote-name-all https://x/config.toml": {"w:.loomux/config.toml"},
		"eval rm .loomux/config.toml":                                       {"rm:.loomux/config.toml"},
		"x=1 eval 'rm .loomux/config.toml'":                                 {"rm:.loomux/config.toml"},
		"echo x > $(pwd)/.loomux/config.toml":                               {"w:$_/.loomux/config.toml"},
		"Set-Content -Path (Join-Path $PWD '.loomux/config.toml')":          {"w:./.loomux/config.toml"},
		"find -delete": {"rm:."},
		"find .loomux/flows/example -exec rm {} +":                   {"rm:.loomux/flows/example"},
		"sed -i s/a/b/ .loomux/config.toml":                          {"w:.loomux/config.toml"},
		"sed 's/a/b/' .loomux/config.toml -i":                        {"w:.loomux/config.toml"},
		"sed -i.bak -e s/a/b/ .loomux/config.toml":                   {"w:.loomux/config.toml"},
		"sed --in-place --expression=s/a/b/ .loomux/config.toml":     {"w:.loomux/config.toml"},
		"sed --in-place --file script.sed .loomux/config.toml":       {"w:.loomux/config.toml"},
		"perl -pi -e 's/a/b/' .loomux/config.toml":                   {"w:.loomux/config.toml"},
		"sed -i -- s/a/b/ .loomux/config.toml":                       {"w:.loomux/config.toml"},
		"git mv .loomux/flows/example x":                             {"rm:.loomux/flows/example", "w:x"},
		"git -C . rm -r .loomux/state":                               {"rm:.loomux/state"},
		"git checkout -- .loomux/config.toml":                        {"w:.loomux/config.toml"},
		"git checkout HEAD notes.txt":                                {"w:notes.txt"},
		"git restore --staged --worktree notes.txt":                  {"w:notes.txt"},
		"git clean -fdx":                                             {"rm:."},
		"git clean -fd -e keep .loomux/flows/example":                {"rm:.loomux/flows/example"},
		"git clean -f -- .loomux/flows/example":                      {"rm:.loomux/flows/example"},
		"[IO.File]::WriteAllText('.loomux/config.toml', 'x')":        {"w:.loomux/config.toml"},
		`[IO.Directory]::Delete('.loomux\state\runs', $true)`:        {"rm:.loomux/state/runs"},
		"[System.IO.File]::Copy('x', \".loomux/config.toml\")":       {"w:.loomux/config.toml"},
		"[IO.Directory]::CreateDirectory('.loomux/flows/example/x')": {"w:.loomux/flows/example/x"},
		"sudo -u root tee .loomux/config.toml":                       {"w:.loomux/config.toml"},
		"xargs -n 1 rm .loomux/config.toml":                          {"rm:.loomux/config.toml"},
		"timeout -s KILL 60 rm .loomux/config.toml":                  {"rm:.loomux/config.toml"},
		"env -u HOME rm .loomux/config.toml":                         {"rm:.loomux/config.toml"},
		"cd .loomux && rm config.toml":                               {"rm:.loomux/config.toml"},
		"cd .loomux; cd flows; rm -r example":                        {"rm:.loomux/flows/example"},
		`Set-Location .loomux\flows; Remove-Item example`:            {"rm:.loomux/flows/example"},
		"pushd /repo/.loomux && rm config.toml":                      {"rm:/repo/.loomux/config.toml"},
		"cd .loomux && popd && rm config.toml":                       {"rm:config.toml"},
		"Move-Item x -Destination:.loomux/config.toml":               {"rm:x", "w:.loomux/config.toml"},
		"New-Item -Name .loomux/config.toml":                         {"w:.loomux/config.toml"},
		"[IO.File]::Copy((Get-Item x), '.loomux/c.t')":               {"w:.loomux/c.t"},
		"[IO.File]::Delete(.loomux/config.toml)":                     {"rm:.loomux/config.toml"},
		"[IO.File]::Delete('.loomux/state/runs/1.flow'":              {"rm:.loomux/state/runs/1.flow"},
		"echo 'x .loomux/config.toml":                                {"w:.loomux/config.toml"},
		"echo x >.loomux/config.toml":                                {"w:.loomux/config.toml"},
		"echo 'a' > f":                                               {"w:f"},
		`echo \"a > .loomux/config.toml`:                             {"w:.loomux/config.toml"},
		"echo `\"a > .loomux/config.toml":                            {"w:.loomux/config.toml"},
		"cd .loomux && curl -O https://x/config.toml":                {"w:.loomux/config.toml"},
		"curl -sLO https://x/a/config.toml?v=1#top":                  {"w:config.toml"},
		"curl --output-dir .loomux -O https://x/config.toml":         {"w:.loomux/config.toml"},
		"curl --output-dir=.loomux --remote-name https://x/c.t":      {"w:.loomux/c.t"},
		"wget -P .loomux/flows https://x/flow.toml":                  {"w:.loomux/flows/flow.toml"},
		"wget --directory-prefix=.loomux https://x/config.toml":      {"w:.loomux/config.toml"},
		"cd .loomux && wget https://x/config.toml":                   {"w:.loomux/config.toml"},
		"git clean -fdxe keep":                                       {"rm:."},
		"find .loomux/state -name '*.jsonl' | xargs rm":              {"rm:.loomux/state where *.jsonl"},
		"gci .loomux -Recurse -Filter *.jsonl | ri":                  {"rm:.loomux where *.jsonl"},
		"find .loomux/state | xargs rm":                              {"rm:.loomux/state"},
		"ls .loomux/state/runs/* | xargs rm -f":                      {"rm:.loomux/state/runs/*"},
		"Get-ChildItem .loomux/state -Recurse | Remove-Item":         {"rm:.loomux/state"},
		`gci .loomux\state\runs | ri -Force`:                         {"rm:.loomux/state/runs"},
		"mv -t .loomux/flows src":                                    {"rm:src", "w:.loomux/flows"},
		"sudo -R /srv rm .loomux/config.toml":                        {"rm:.loomux/config.toml"},
		`sh -c "echo x > .loomux/config.toml"`:                       {"w:.loomux/config.toml"},
		"bash -c 'echo x > .loomux/config.toml'":                     {"w:.loomux/config.toml"},
		`cmd /c "echo x > .loomux\config.toml"`:                      {"w:.loomux/config.toml"},
		`pwsh -c "'x' > .loomux/config.toml"`:                        {"w:.loomux/config.toml"},
		`powershell -Command "Remove-Item .loomux"`:                  {"rm:.loomux"},
		"cd .loomux && bash -lc 'rm config.toml'":                    {"rm:.loomux/config.toml"},
		"pwsh -c pwsh -c pwsh -c rm x":                               {"rm:x"},
	} {
		targets, _ := shellWrites(root, line)
		got := spelled(targets)
		for _, w := range want {
			if !slices.Contains(got, w) {
				t.Errorf("%q: targets %q, want %q among them", line, got, w)
			}
		}
	}
}

// A name filter is matched as find matches it, and a listing past its limit
// is taken for the removal of its start paths.
func TestAFilteredListingKeepsWhatItsPatternMatches(t *testing.T) {
	for _, row := range []struct {
		glob, s string
		want    bool
	}{
		{"*.orig", "a.orig", true}, {"*.orig", "a.origx", false}, {"a/*/c", "a/b/x/c", true},
		{"?.go", "a.go", true}, {"?.go", ".go", false}, {"[ab].go", "b.go", true},
		{"[!ab].go", "b.go", false}, {"[^ab].go", "c.go", true}, {"[a-c]x", "bx", true},
		{"[a-c]x", "dx", false}, {"[x", "[x", true}, {"[]x", "x", false}, {"ab", "a", false},
	} {
		if got := fnmatch(row.glob, row.s); got != row.want {
			t.Errorf("fnmatch(%q, %q) = %v", row.glob, row.s, got)
		}
	}
	for line, want := range map[string]string{
		"a $(b $(c)) d": "a $_ d", "a $(b": "a $(b", "a `b` c `d": "a $_ c `d",
	} {
		if got := foldSubstitutions(line); got != want {
			t.Errorf("foldSubstitutions(%q) = %q", line, got)
		}
	}
	root := t.TempDir()
	mkfile(t, root, "a/x.orig")
	mkfile(t, root, "a/y.orig")
	j := newJudge(root, config.Policy{})
	target := shellTarget{filters: []nameFilter{{glob: "*.orig"}}}
	for _, rel := range []string{"a", filepath.Join(root, "a")} {
		if got, ok := j.listing(rel, target, 10); !ok || len(got) != 2 {
			t.Errorf("%q within the limit: %q %v", rel, got, ok)
		}
	}
	if got, ok := j.listing("missing", target, 10); !ok || len(got) != 0 {
		t.Errorf("a missing start: %q %v", got, ok)
	}
	if _, ok := newJudge(root, config.Policy{}).listing("a", target, 2); ok {
		t.Error("past the limit: listed")
	}
	if got := newJudge(root, config.Policy{}).filteredReasons(".", target, 2); !slices.Contains(got, manifestReason) {
		t.Errorf("past the limit the removal of the start: %q", got)
	}
	for args, want := range map[string]string{
		"-Path src -Include *.tmp,*.bak":  "[src] 2",
		"-LiteralPath src -Filter *.orig": "[src] 1",
		"src -Filter *.orig":              "[src] 1",
		"-Recurse -Include *.tmp":         "[.] 1",
		"-Include *,*.tmp":                "[] 0",
		"-Recurse":                        "[.] 0",
	} {
		starts, filters := childItemFilters(strings.Fields(args))
		if got := fmt.Sprint(starts, " ", len(filters)); got != want {
			t.Errorf("childItemFilters(%q) = %s, want %s", args, got, want)
		}
	}
}

// What only reads, and what writes nowhere, yields no target at all.
func TestShellWritesFindsNoTargetInAReadingLine(t *testing.T) {
	root := t.TempDir()
	for _, line := range []string{
		"cat .loomux/config.toml",
		"ls .loomux/flows",
		"sed -n '1,5p' .loomux/config.toml",
		"find . -name x",
		"git status",
		"git checkout feat/x",
		"git checkout -b x",
		"git checkout --orphan x",
		"git restore --staged x",
		"git restore -S x",
		"git clean -n",
		"git clean --dry-run -d",
		"git clean -nd",
		"echo x 2>&1",
		"echo hi >&2",
		"cat < in.txt",
		"cat <<'EOF'",
		`git commit -m "fix(guard): keep .loomux/state/runs"`,
		"robocopy onlyone",
		"loomux flow list",
		"tar -xf evil.tar",
		"echo x > ''",
		"[IO.File]::ReadAllText('.loomux/config.toml')",
		"curl -o",
		"cd",
		"Expand-Archive x -DestinationPath",
		"curl -- -o x",
		"git --version",
		"grep '>' notes.txt",
		"grep -n '=>' .loomux/config.toml",
		"awk '$1 > 5' f",
		`git commit -m "a -> .loomux/config.toml"`,
		"curl -O https://x",
		"curl -O https://x/",
		"wget https://x",
		"ls x || rm",
		`bash -c 'grep ">" f'`,
		`git commit -m "clean .loomux/state | xargs rm"`,
		`echo "see .loomux/state | rm"`,
		"sh -c",
		"cmd /c",
		// A command nested deeper than three shells is not read: a named limit.
		"pwsh -c pwsh -c pwsh -c pwsh -c rm x",
	} {
		if targets, _ := shellWrites(root, line); len(targets) != 0 {
			t.Errorf("%q: targets %q, want none", line, spelled(targets))
		}
	}
	// A copy into the folder above a kept one writes that folder, it removes nothing.
	targets, _ := shellWrites(root, "cp -r mine .loomux/flows/")
	if got := spelled(targets); !slices.Equal(got, []string{"w:.loomux/flows/"}) {
		t.Fatalf("cp into the folder above: %q", got)
	}
}

// Where a line names a path in two roles, each is read in the right one and
// nothing else is added.
func TestShellWritesNamesExactlyTheTargetsOfAVerb(t *testing.T) {
	root := t.TempDir()
	for line, want := range map[string][]string{
		"git clean -fdxe keep":                        {"rm:."},
		"mv -t .loomux/flows src":                     {"rm:src", "w:.loomux/flows"},
		"cd .loomux && curl -O https://x/config.toml": {"w:.loomux/config.toml"},
		"grep '>' notes.txt > out":                    {"w:out"},
		"ls .loomux | rm x":                           {"rm:x"},
		"curl -o.loomux/config.toml https://x/y":      {"w:.loomux/config.toml"},
	} {
		targets, _ := shellWrites(root, line)
		if got := spelled(targets); !slices.Equal(got, want) {
			t.Errorf("%q: targets %q, want %q", line, got, want)
		}
	}
}

// unknown names each program of a trusted segment the table does not know,
// with its arguments; a quote-blind fragment of quoted text is never one.
func TestShellWritesNamesTheProgramsItDoesNotKnow(t *testing.T) {
	root := t.TempDir()
	_, unknown := shellWrites(root, "frob .loomux/config.toml")
	if !slices.ContainsFunc(unknown, func(args []string) bool {
		return slices.Equal(args, []string{"frob", ".loomux/config.toml"})
	}) {
		t.Fatalf("frob: unknown %q", unknown)
	}
	for _, line := range []string{"git stash push x", "loomux flow run x", `sh -c "rm x"`, "go vet .", "loomux init"} {
		if _, unknown := shellWrites(root, line); len(unknown) == 0 {
			t.Errorf("%q: nothing unknown", line)
		}
	}
	for _, line := range []string{
		"cat .loomux/config.toml", "git status", "git diff x", "loomux flow list", "loomux flow show 1",
		"go run ./cmd/loomux config get a", "loomux config list", "loomux config proposals", "loomux check precommit",
		"rm x", "echo hi", "X=1", "cd x", "find . -name y",
		`git commit -m "fix(guard): keep .loomux/state/runs"`,
		`git commit -m "a; frob .loomux/config.toml"`,
	} {
		if _, unknown := shellWrites(root, line); len(unknown) != 0 {
			t.Errorf("%q: unknown %q, want none", line, unknown)
		}
	}
}
