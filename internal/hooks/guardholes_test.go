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
	var lines []string
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
	return append(lines,
		"pwsh -c pwsh -c pwsh -c pwsh -c loomux init",
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

// holesFlips are the lines the fix turns from a pass into a refusal in the
// default mode.
func holesFlips() map[string]bool {
	return map[string]bool{}
}

// holesFlipsStrict are the lines the fix turns from a pass into a refusal
// in strict mode.
func holesFlipsStrict() map[string]bool {
	return map[string]bool{}
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
