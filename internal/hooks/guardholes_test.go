package hooks

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/xidus90/loomux/internal/config"
)

const holesBatteryFile = "testdata/guard-battery-holes.tsv"

// encodedCommand is s as pwsh -EncodedCommand takes it: UTF-16LE in base64.
func encodedCommand(s string) string {
	units := utf16.Encode([]rune(s))
	b := make([]byte, 2*len(units))
	for i, u := range units {
		binary.LittleEndian.PutUint16(b[2*i:], u)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// guardedInside are the loomux commands and the write the string shells
// carry in the battery.
func guardedInside() []string {
	return []string{
		"loomux config apply",
		"loomux init",
		"loomux flow resume 1 --answer y",
		"loomux dev switchover prune-hooks --file x --match y",
		"echo x > .loomux/config.toml",
	}
}

// stringShellLines run a line from a string, each around every guarded
// line, and the lines around them that only read.
func stringShellLines() []string {
	return append(guardedStringShellLines(),
		"sh -c \"loomux config list\"",
		"iex 'loomux config get a'",
		"sh -c \"cd x && cat y\"",
		"bash -c 'frob \"$1\"' _ x",
		"pwsh -c pwsh -c pwsh -c rm x",
		"pwsh -enc "+encodedCommand("Get-ChildItem"),
		"bash -o pipefail -c 'go test ./...'",
		"env -S 'go vet ./...'",
		"echo hi | sh -c 'cat'",
		"cat notes.txt | sh script.sh",
		"ls | iex",
		"echo 'go test ./...' | sh",
	)
}

// guardedStringShellLines are the lines of stringShellLines that run a
// guarded line, and a string nested deeper than the guard reads.
func guardedStringShellLines() []string {
	lines := []string{"pwsh -c pwsh -c pwsh -c pwsh -c loomux init"}
	for _, inner := range guardedInside() {
		lines = append(lines,
			"sh -c \""+inner+"\"",
			"bash -c '"+inner+"'",
			"pwsh -c '"+inner+"'",
			"pwsh -Command "+inner,
			"powershell -NoProfile -Command \""+inner+"\"",
			"iex '"+inner+"'",
			"Invoke-Expression \""+inner+"\"",
			"eval '"+inner+"'",
			"pwsh -enc "+encodedCommand(inner),
			"powershell -EncodedCommand "+encodedCommand(inner),
			"bash -o pipefail -c '"+inner+"'",
			"cmd /c\""+inner+"\"",
			"env -S '"+inner+"'",
			"sh -c \"cd x && "+inner+"\"",
			"echo \""+inner+"\" | sh",
			"printf '%s\\n' '"+inner+"' | bash -s",
			"'"+inner+"' | iex",
			"Write-Output '"+inner+"' | pwsh -Command -",
			"echo '"+inner+"' | Invoke-Expression",
			"bash <<< '"+inner+"'",
		)
	}
	return lines
}

// folderCopyLines copy or move into a folder, and copies beside one.
func folderCopyLines() []string {
	return []string{
		"cp /tmp/config.toml .loomux/",
		"cp /tmp/config.toml .loomux",
		"cp a/config.toml b/config.toml .loomux",
		"cp -t .loomux /tmp/config.toml",
		"mv /tmp/config.toml .loomux/",
		"Copy-Item C:/tmp/config.toml .loomux/",
		"Copy-Item -Path x/config.toml -Destination .loomux",
		"Move-Item x/config.toml .loomux",
		"cp -r /tmp/x/.loomux .",
		"cp -r /tmp/tpl/. .",
		"rsync -a /tmp/tpl/ .",
		`xcopy C:\tmp\config.toml .loomux\`,
		`robocopy C:\tmp .loomux config.toml`,
		"cp a b",
		"cp -r src dst",
		"cp x.txt docs/",
		"rsync -t a b",
		"install -m 644 a b",
		"ln -s a b",
		"cp /tmp/notes.md .loomux/",
		"cp -r /tmp/flows .loomux/",
	}
}

// patchLines apply a patch, by name or from a redirection.
func patchLines() []string {
	return []string{
		"patch .loomux/config.toml < p.diff",
		"patch -o .loomux/config.toml a < p.diff",
		"patch -p1 < p.diff",
		"patch -p1 -i p.diff",
		"patch < p.diff",
		"git apply p.diff",
		"git apply -R p.diff",
		"git apply --directory=.loomux q.diff",
		"git am p.patch",
		"git am < p.patch",
		"git apply missing.diff",
		"git apply ok.diff",
		"patch -p1 < ok.diff",
		"git apply --check ok.diff",
		"git am ok.diff",
	}
}

// assignmentLines set a variable or alias and use it on the same line.
func assignmentLines() []string {
	return []string{
		"D=.loomux; echo x > $D/config.toml",
		"D=.loomux; echo x > ${D}/config.toml",
		"export D=.loomux; rm $D/config.toml",
		"F=.loomux/config.toml; echo x > $F",
		"$D='.loomux'; Set-Content \"$D/config.toml\" x",
		"$D = '.loomux'; Set-Content $d/config.toml x",
		"$env:D='.loomux'; Remove-Item $env:D/config.toml",
		"Set-Variable D .loomux; rm $D/config.toml",
		"Set-Variable -Name D -Value .loomux; rm $D/config.toml",
		`set D=.loomux& echo x > %D%\config.toml`,
		"L=loomux; $L config apply",
		"$L='loomux'; & $L config apply",
		"alias l=loomux; l init",
		"Set-Alias l loomux; l init",
		"M=loomux; $M init",
		"D=docs; echo x > $D/a.md",
		"X=1; loomux config list",
		"alias ll='ls -l'; ll .loomux",
	}
}

// spellingLines name the manifest by a short name or a trailing dot.
func spellingLines() []string {
	return []string{
		"echo x > LOOMUX~1/config.toml",
		"echo x > .loomux/CONFIG~1.TOM",
		"echo x > .loomux/config.toml.",
		`echo x > ".loomux/config.toml "`,
		"echo x > .loomux./config.toml",
		"rm -rf LOOMUX~1",
		"echo x > PROGRA~1/x",
		"echo x > docs/a.",
		"echo x > .loomux/CONFIG~1.TXT",
	}
}

// codeStringLines hand an unknown program code that names a path.
func codeStringLines() []string {
	return []string{
		`python -c "open('.loomux/config.toml','w')"`,
		`node -e "require('fs').writeFileSync('.loomux/config.toml','')"`,
		`python -c "print(1)"`,
	}
}

// tailLines write behind a variable that is set nowhere on the line.
func tailLines() []string {
	return []string{
		"echo x > $D/config.toml",
		"echo x > ${D}fig.toml",
		"echo x > $D/state/hooks/x",
		"echo x > $D/notes.md",
		"echo x > $D",
	}
}

// holesBattery is every line the differential probe asks about, each once,
// in a fixed order.
func holesBattery() []string {
	lines := slices.Concat(stringShellLines(), folderCopyLines(), patchLines(), assignmentLines(),
		spellingLines(), codeStringLines(), tailLines(), manifestShellWrites(), manifestShellReads())
	slices.Sort(lines)
	return slices.Compact(lines)
}

// refusedAfter are the battery's lines the guard refuses after the fix in
// both modes; a line among them it refused before already is no flip, and
// the differential test does not ask it to be one.
func refusedAfter() []string {
	return guardedStringShellLines()
}

// holesFlips are the lines the fix turns from a pass into a refusal in the
// default mode.
func holesFlips() map[string]bool {
	flips := map[string]bool{}
	for _, line := range refusedAfter() {
		flips[line] = true
	}
	return flips
}

// holesFlipsStrict are the lines the fix turns from a pass into a refusal
// in strict mode.
func holesFlipsStrict() map[string]bool {
	return holesFlips()
}

// A loomux command or a write inside a string a shell runs is refused like
// the line itself, by Bash and PowerShell alike, and with the reason the
// line itself gets.
func TestALineInsideAStringIsJudgedLikeTheLine(t *testing.T) {
	root := holesWorld(t)
	for _, inner := range guardedInside() {
		want := shellReasons(t, root, inner, config.Policy{})
		if len(want) == 0 {
			t.Fatalf("%q: the line itself is not refused", inner)
		}
		for _, line := range guardedStringShellLines()[1:] {
			if !strings.Contains(line, inner) && !strings.Contains(line, encodedCommand(inner)) {
				continue
			}
			got := shellReasons(t, root, line, config.Policy{})
			for _, w := range want {
				if !slices.Contains(got, w) {
					t.Errorf("%q: reasons %q, want %q among them", line, got, w)
				}
			}
		}
	}
}

// A command inside a string is no direct call, so its flag exempts nothing:
// the guard does not read the string the way the shell does.
func TestAFlagInsideAStringExemptsNothing(t *testing.T) {
	for _, line := range []string{
		"sh -c 'loomux init --dry-run'",
		`pwsh -c "loomux config set a b --propose"`,
		"echo 'loomux init --dry-run' | sh",
	} {
		if !writesConfiguration(line, false) {
			t.Errorf("must refuse %q", line)
		}
	}
	for _, line := range []string{"sh -c 'loomux config list'", "iex 'loomux config proposals'"} {
		if writesConfiguration(line, false) {
			t.Errorf("must allow %q", line)
		}
	}
}

// A string nested deeper than the guard reads is refused with a reason of
// its own; one level less is read through.
func TestAStringNestedTooDeepRefuses(t *testing.T) {
	root := t.TempDir()
	got := checkTool(root, "Bash", command("pwsh -c pwsh -c pwsh -c pwsh -c echo hi"), config.Policy{})
	if !slices.ContainsFunc(got, func(r string) bool { return strings.Contains(r, "shells deep") }) {
		t.Fatalf("reasons %q, want the depth reason", got)
	}
	if got := checkTool(root, "Bash", command("pwsh -c pwsh -c pwsh -c echo hi"), config.Policy{}); len(got) != 0 {
		t.Fatalf("three shells deep: reasons %q, want none", got)
	}
}

// innerLine finds the string every shell runs.
func TestInnerLineReadsEveryStringShell(t *testing.T) {
	for args, want := range map[string]string{
		"iex|a b":                            "a b",
		"Invoke-Expression|-Command|a b":     "a b",
		"x=1|iex|a|b":                        "a b",
		"pwsh|-enc|" + encodedCommand("a b"): "a b",
		"powershell|-EncodedCommand|" + encodedCommand("a"): "a",
		"pwsh|-e|" + encodedCommand("c d"):                  "c d",
		"bash|-o|pipefail|-c|a":                             "a",
		"bash|+o|posix|-c|a":                                "a",
		"bash|-eo|pipefail|-c|a":                            "a",
		"cmd|/c\"a b\"":                                     "\"a b\"",
		"cmd|/ca|b":                                         "a b",
		"env|-S|a b":                                        "a b",
		"env|-Sa b":                                         "a b",
		"env|--split-string=a b":                            "a b",
		"env|--split-string|a b":                            "a b",
		"env|-i|-S|a":                                       "a",
		"bash|--norc|-c|a":                                  "a",
		"cmd|/k|a":                                          "a",
		"env|-S|a|b":                                        "a b",
	} {
		line, _, ok := innerLine(strings.Split(args, "|"))
		if !ok || line != want {
			t.Errorf("%q: line %q, %v; want %q", args, line, ok, want)
		}
	}
	for _, args := range []string{
		"pwsh|-enc|!!", "pwsh|-enc|" + base64.StdEncoding.EncodeToString([]byte{1}), "pwsh|-enc",
		"env|a", "env|-S", "env|-i", "bash|+c|a", "bash|-o", "bash|-o|pipefail", "cmd|/q", "iex",
	} {
		if line, _, ok := innerLine(strings.Split(args, "|")); ok {
			t.Errorf("%q: line %q, want none", args, line)
		}
	}
}

// pipedLine is the line a shell reads from the segment before its pipe.
func TestPipedLineReadsWhatTheSegmentBeforePrints(t *testing.T) {
	for _, row := range []struct{ prev, args, want string }{
		{`echo "a b"`, "sh", "a b"},
		{"echo a b", "bash|-s", "a b"},
		{`printf '%s\n' 'a b'`, "bash|-s|x", "a b"},
		{`printf 'a b'`, "zsh", "a b"},
		{"'a b'", "iex", "a b"},
		{`"a b"`, "Invoke-Expression", "a b"},
		{"Write-Output 'a b'", "pwsh|-Command|-", "a b"},
		{"echo a", "pwsh", "a"},
		{"echo a", "powershell|-c|-", "a"},
		{"echo a", "cmd", "a"},
		{"echo a", "cmd|/q", "a"},
		{"  echo a", "dash", "a"},
		{"echo -e a b", "sh", "a b"},
		{"echo a", "bash|--norc", "a"},
		{"echo a", "bash|-o|pipefail", "a"},
		{"echo a", "pwsh|-File|-", "a"},
		{"echo a", "pwsh|-NoProfile", "a"},
	} {
		line, ok := pipedLine(row.prev, strings.Split(row.args, "|"))
		if !ok || line != row.want {
			t.Errorf("%q | %q: line %q, %v; want %q", row.prev, row.args, line, ok, row.want)
		}
	}
	for _, row := range []struct{ prev, args string }{
		{"echo a", "sh|-c|cat"},
		{"cat f", "sh"},
		{"echo a", "sh|script.sh"},
		{"ls", "iex"},
		{"echo a", "iex|b"},
		{"echo a", "pwsh|-File|x.ps1"},
		{"echo a", "cmd|/c|x"},
		{"echo a", "cat"},
		{"", "sh"},
		{"echo a", ""},
		{"echo -n", "sh"},
		{"echo a", "pwsh|-Command|ls"},
		{"echo a", "pwsh|-Command"},
		{"printf", "sh"},
	} {
		args := strings.Split(row.args, "|")
		if row.args == "" {
			args = nil
		}
		if line, ok := pipedLine(row.prev, args); ok {
			t.Errorf("%q | %q: line %q, want none", row.prev, row.args, line)
		}
	}
}

// Only a pipe feeds a shell what the segment before prints: after ; or &&
// the shell reads the terminal, and the echo only prints.
func TestOnlyAPipeFeedsAShell(t *testing.T) {
	if got := fedLines("echo 'a b'", []string{"sh"}, false); len(got) != 0 {
		t.Fatalf("no pipe: %q", got)
	}
	root := t.TempDir()
	for _, line := range []string{
		"echo 'loomux config apply'; sh",
		"echo 'echo x > .loomux/config.toml' && sh",
		"echo 'loomux init'; iex",
	} {
		if got := shellReasons(t, root, line, config.Policy{}); len(got) != 0 {
			t.Errorf("%q: reasons %q, want none", line, got)
		}
	}
}

// A shell named by a quoted path that holds a parenthesis is fed as well:
// only the cut that honours quotes keeps such a path whole.
func TestAQuotedShellPathIsFedToo(t *testing.T) {
	for _, line := range []string{
		`echo "loomux init" | "C:\Program Files (x86)\Git\bin\bash.exe"`,
		`echo 'loomux config apply' | "C:\Program Files (x86)\PowerShell\pwsh.exe" -Command -`,
	} {
		if !writesConfiguration(line, false) {
			t.Errorf("must refuse %q", line)
		}
	}
}

// hereString is the line a shell reads from <<<.
func TestHereStringIsTheLineAShellReads(t *testing.T) {
	for args, want := range map[string]string{
		"bash|<<<|a b": "a b",
		"bash|<<<a b":  "a b",
		"sh|-s|<<<|a":  "a",
	} {
		if line, ok := hereString(strings.Split(args, "|")); !ok || line != want {
			t.Errorf("%q: %q, %v; want %q", args, line, ok, want)
		}
	}
	for _, args := range []string{"cat|<<<|a", "bash|<<<", "bash|-c|x|<<<|a", "bash"} {
		if line, ok := hereString(strings.Split(args, "|")); ok {
			t.Errorf("%q: %q, want none", args, line)
		}
	}
}

// holesWorld is the project the battery is judged in: the manifest, and
// patches that change it, one that changes nothing protected, and one
// without the a/ and b/ prefixes.
func holesWorld(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	toManifest := "diff --git a/.loomux/config.toml b/.loomux/config.toml\n--- a/.loomux/config.toml\n+++ b/.loomux/config.toml\n@@ -1 +1 @@\n-x\n+y\n"
	files := map[string]string{
		".loomux/config.toml": "x\n",
		"p.diff":              toManifest,
		"p.patch":             "From 1 Mon Sep 17 00:00:00 2001\nSubject: [PATCH] x\n\n---\n" + toManifest,
		"q.diff":              "--- config.toml\n+++ config.toml\n@@ -1 +1 @@\n-x\n+y\n",
		"ok.diff":             "diff --git a/src/a.go b/src/a.go\n--- a/src/a.go\n+++ b/src/a.go\n@@ -1 +1 @@\n-x\n+y\n",
	}
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// holesVerdict says whether the guard refuses line from Bash.
func holesVerdict(root, line string, strict bool) bool {
	return len(checkTool(root, "Bash", map[string]any{"command": line}, config.Policy{Strict: strict})) > 0
}

// verdictWord is how the record spells a verdict.
func verdictWord(refused bool) string {
	if refused {
		return "refused"
	}
	return "pass"
}

// TestRecordTheHolesBattery writes what the guard says about every line of
// the battery, in the default and in strict mode. It runs only when asked,
// once, at the state before a change to the guard; the file it writes is the
// "before" the differential test reads.
func TestRecordTheHolesBattery(t *testing.T) {
	if os.Getenv("LOOMUX_RECORD_GUARD_BATTERY") != "1" {
		t.Skip("set LOOMUX_RECORD_GUARD_BATTERY=1 to record the guard's verdicts")
	}
	root := holesWorld(t)
	var b strings.Builder
	for _, line := range holesBattery() {
		fmt.Fprintf(&b, "%s\t%s\t%s\n", verdictWord(holesVerdict(root, line, false)),
			verdictWord(holesVerdict(root, line, true)), strconv.Quote(line))
	}
	if err := os.MkdirAll(filepath.Dir(holesBatteryFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(holesBatteryFile, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The differential probe: no line the guard refused before passes now, in
// either mode, and the lines that flipped to a refusal are exactly the
// expected ones. The record is taken on Windows, and both columns hang on
// it: strict mode resolves through the file system, and whether C:/… or
// /tmp/… is absolute decides how the default mode spells a path. Elsewhere
// the test does not run; the Windows gate holds it.
func TestTheHolesFixOpensNothing(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the record is a Windows one")
	}
	data, err := os.ReadFile(holesBatteryFile)
	if err != nil {
		t.Fatal(err)
	}
	type verdicts struct{ def, strict bool }
	before := map[string]verdicts{}
	for row := range strings.SplitSeq(strings.TrimSuffix(string(data), "\n"), "\n") {
		fields := strings.SplitN(row, "\t", 3)
		if len(fields) != 3 {
			t.Fatalf("%q: want three fields", row)
		}
		line, err := strconv.Unquote(fields[2])
		if err != nil {
			t.Fatalf("%q: %v", row, err)
		}
		before[line] = verdicts{fields[0] == "refused", fields[1] == "refused"}
	}
	lines := holesBattery()
	if len(before) != len(lines) {
		t.Fatalf("the battery has %d lines, the record %d: record it again at the state before the change", len(lines), len(before))
	}
	root := holesWorld(t)
	modes := []struct {
		strict bool
		flips  map[string]bool
		was    func(verdicts) bool
	}{
		{false, holesFlips(), func(v verdicts) bool { return v.def }},
		{true, holesFlipsStrict(), func(v verdicts) bool { return v.strict }},
	}
	for _, mode := range modes {
		for _, line := range lines {
			was, recorded := before[line]
			if !recorded {
				t.Errorf("%q is not in the record: record it again at the state before the change", line)
				continue
			}
			now := holesVerdict(root, line, mode.strict)
			switch {
			case mode.was(was) && !now:
				t.Errorf("strict=%v: %q was refused and now passes", mode.strict, line)
			case !mode.was(was) && now && !mode.flips[line]:
				t.Errorf("strict=%v: %q passed and is now refused, but is no expected flip", mode.strict, line)
			case mode.flips[line] && !now:
				t.Errorf("strict=%v: %q is an expected flip and still passes", mode.strict, line)
			}
		}
		for line := range mode.flips {
			if !slices.Contains(lines, line) {
				t.Errorf("strict=%v: the expected flip %q is no line of the battery", mode.strict, line)
			}
		}
	}
}
