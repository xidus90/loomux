# Wächterregel gegen Interpreter auf stdin — Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Der PreToolUse-Wächter verweigert jede Shell-Zeile, in der ein Interpreter sein Programm von stdin liest, ohne dass dort echter Inhalt ankommt.

**Architecture:** Neue Datei `internal/hooks/guardstdin.go` mit `readsProgramFromStdin(line)`, gerufen in `checkTool` neben `armsOrDisarms`. Sie entfernt Heredoc-Rümpfe nur für sich, liest über `anyRunLine`, `lineVariants`, den anführungszeichenbewussten Schnitt, `readings` und `dropPrefixes` und prüft dann Tabellen je Interpreter. Die versionierte Differenzbatterie bekommt die Interpreter-Zeilen dazu; ihr Rekorder ergänzt künftig nur fehlende Zeilen.

**Tech Stack:** Go 1.x (Modul `github.com/xidus90/loomux`), `go test`, `go test -overlay`, `uv run --script` für das Mutationsskript.

**Spec:** `docs/.superpowers/specs/2026-10-06-guard-stdin-interpreter-design.md`

## Global Constraints

- 100 % Coverage je Funktion; eine Ausnahme nur mit `//coverage:exempt <grund>` direkt über `func` (keine geplant).
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst (Map- und Slice-Literale sind erlaubt).
- Code, Bezeichner, Kommentare, Fehlermeldungen und Commits englisch; Kommentare auf einer anderen Abstraktionsebene als die Zeile darunter; keine Historie im Code (kein Task, keine Spec).
- Conventional Commits; eine Commit-Nachricht nennt kein Arbeitspapier.
- Autor ist der Mensch: kein `Co-Authored-By`, keine Modellnennung in Commit oder PR.
- Niemand außer dem Menschen pusht.
- Subagenten: keine Prozesse nach Namen beenden (nur eigene PIDs); keinen Befehl starten, der auf stdin wartet (`python -`, ein Heredoc an einen Interpreter); Backslashes nie durch die Shell schreiben (Heredoc, `printf`, `sed`, `python -c`), sondern per Write/Edit; kein Hintergrundprozess bleibt laufen.
- Overlay-Pfade immer als `C:/…`; `-overlay` nie zusammen mit `-cover`.
- Die Ausgabe jedes langen Laufs zuerst ganz in eine Datei im Scratchpad, dann filtern.
- Scratchpad: `C:/Users/micro/AppData/Local/Temp/claude/C--Users-micro-Documents--GIT-loomux--claude-worktrees-hungry-golick-7871dc/97d5c277-02bb-4919-88dc-82cd69acfaec/scratchpad` (unten `$S`).
- Worktree: `C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/hungry-golick-7871dc`, Zweig `feat/guard-stdin-interpreter`. Vor jedem git-Schreibbefehl `git rev-parse --show-toplevel` und `git branch --show-current` lesen.

## Review Focus

1. Ein Heredoc mit einer Zeile, die einem Befehl gleicht, aber kein Interpreter-Aufruf ist (`cat <<'EOF' > notes.md` mit `python -` im Rumpf), muss durchgehen — und die Schreibregeln müssen solche Rümpfe weiter als Befehle lesen (Differenzbatterie: nichts, was vorher verweigert war, geht durch).
2. Versions- und Hilfeaufrufe (`python -V`, `node -v`, `perl -v`, `ruby -v`, `python --help`) dürfen nicht verweigert werden, obwohl sie kein Programm nennen.
3. Echte Pipes (`echo … | python -`, `Get-Content x.py | python -`) und `<`-Dateien bleiben erlaubt; `||` ist keine Pipe.
4. Text in Anführungszeichen ist Daten: `git commit -m "fix; python -"` geht durch.
5. Pfadformen unter Windows (`C:\Python314\python.exe -`, `python.exe`, `python3.12`) werden erkannt, `pythonic` nicht.

Jede dieser Zeilen hat ihren Fall in `stdinLines()` (Task 1).

## Gemessene Grundlage

Der Code unten ist vor dem Plan als Prototyp per Overlay am Paket gelaufen (2026-10-06):

- `go test ./internal/...` mit Regel: grün, kein bestehender Test bricht.
- RED gegen Stubs (Regel liefert immer `false`, `withoutHereBodies` gibt die Zeile zurück): `--- FAIL: TestTheGuardRefusesAnInterpreterThatReadsStdin` mit 88 Zeilen `refused false, want true` (44 Zeilen × 2 Modi) plus `PowerShell: python - passes` und `run_command: python - passes`; `--- FAIL: TestWithoutHereBodiesKeepsTheCommands` mit 11 Abweichungen.
- Batterie: der Merge-Rekorder fügt die neuen Zeilen an, alle `pass`, und ändert keine bestehende (gemessen mit 88 Zeilen vor den letzten zwei Testfällen; mit ihnen sind es 90). An den Stubs meldet `TestTheGuardOpensNothingItRefusedBefore` je verweigerte Zeile ein `still passes`, mit der Regel ist er grün. Ein vollständiges Neuaufnehmen hätte dagegen 282 `pass`-Urteile der Arm-/Gate-Änderung auf `refused` gesetzt und deren Nachweis entwertet; darum der Merge-Rekorder.
- Mutationsrunde: 52 Mutanten (jede Teilbedingung, jede Tabellenzeile und jeder Wertlisten-Eintrag, den ein Test trennen kann), alle getötet; ein unveränderter Kontrollmutant überlebt. Zwei Mutanten waren äquivalent und haben den Code vereinfacht: die `<<`-Prüfung in `fromFile` (deckt `inline` ab) und ein eigener `<<<`-Fall in `hereDocs`.
- Proben: `echo 'print(1)' | uv run --no-project -` druckt `1` (uv 0.12.16); nacktes `uv run` endet mit Exit 2; `echo 'print 1' | perl -c` meldet `- syntax OK`; `perl -I /tmp` und `perl -M strict` nehmen den Wert getrennt; `python -V`, `node -v`, `perl -v` enden ohne Skript. Ein leerer Heredoc in `python -` hing in Git Bash auf dieser Maschine, bis er gestoppt wurde.

---

### Task 1: Die Regel, ihre Tests und die Differenzbatterie

**Files:**
- Create: `internal/hooks/guardstdin.go`
- Create: `internal/hooks/guardstdin_test.go`
- Modify: `internal/hooks/guard.go` (in `checkTool`, direkt nach dem `armsOrDisarms`-Block, ca. Zeile 325–327)
- Modify: `internal/hooks/guardgate_test.go` (`battery`, `TestRecordTheGuardBattery`, `TestTheGuardOpensNothingItRefusedBefore`)
- Modify: `internal/hooks/testdata/guard-battery-before.tsv` (nur per Rekorder)

**Interfaces:**
- Consumes (bestehend, Paket `hooks`): `anyRunLine(line string, judge func(line string, nested bool) bool) bool`, `lineVariants(line string) []string`, `splitSegments(line string, quoteAware bool) []string`, `readings(segment string) [][]string`, `dropPrefixes(words []string) []string`, `verbOf(program string) string`, `redirection(w string) (isRedirect, bare bool)`, `checkTool(root, tool string, input map[string]any, policy config.Policy) []string`, `readBattery(t *testing.T) map[string]bool`, `refusedBy(root, line string) bool`.
- Produces: `const stdinReason string`, `func readsProgramFromStdin(line string) bool`, `func withoutHereBodies(line string) string`, `func stdinLines() map[string]bool` (Test).

- [ ] **Step 1: Stub anlegen und in `checkTool` verdrahten**

`internal/hooks/guardstdin.go` (Stub; Step 5 ersetzt ihn ganz):

```go
package hooks

// stdinReason refuses an interpreter that would wait on stdin for its
// program, or get it as inline text a shell may have rewritten.
const stdinReason = "loomux refuses an interpreter that reads its program from stdin (`python -`, a bare `python`, `uv run -`, " +
	"a heredoc or here-string into one): with nothing piped in it waits until it is stopped, and a shell may rewrite " +
	"the backslashes of inline program text. Write the script to a file in the scratchpad with the host's file tool " +
	"(Write, write_to_file) and run that file."

func readsProgramFromStdin(line string) bool { return false }

func withoutHereBodies(line string) string { return line }
```

In `internal/hooks/guard.go`, `checkTool`, nach

```go
				if armsOrDisarms(line, policy.Strict) {
					reasons = append(reasons, gateReason)
				}
```

einfügen:

```go
				if readsProgramFromStdin(line) {
					reasons = append(reasons, stdinReason)
				}
```

- [ ] **Step 2: Tests schreiben**

`internal/hooks/guardstdin_test.go` per Write (die Datei enthält Backslashes; nie per Shell schreiben):

```go
package hooks

import (
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// stdinLines are command lines about interpreters and what the stdin rule
// says of each: true for a refusal.
func stdinLines() map[string]bool {
	return map[string]bool{
		"python -":  true,
		"python3 -": true,
		"python":    true,
		"cd x && python3 - <<'EOF'\nprint(1)\nEOF": true,
		"python <<EOF\nprint(1)\nEOF":              true,
		"PYTHONIOENCODING=utf8 python -":           true,
		"C:/Python314/python.exe -":                true,
		`C:\Python314\python.exe -`:                true,
		"node":                                     true,
		"uv run python -":                          true,
		"python -X utf8 -":                         true,
		"python - > out.txt 2>&1":                  true,
		"uv run -":                                 true,
		"uv run --with rich -":                     true,
		"uvx --from x python -":                    true,
		"perl -c":                                  true,
		"python -E -":                              true,
		"py -3.12 -":                               true,
		"python3.12 -":                             true,
		"ruby -E utf-8 -":                          true,
		"node --input-type=module -":               true,
		"node -r fs -":                             true,
		"python -W ignore -":                       true,
		"python -- -":                              true,
		"python -u":                                true,
		"cat <<'EOF' | python -\nprint(1)\nEOF":    true,
		"@'\nprint(1)\n'@ | python -":              true,
		"$code = @\"\nprint(1)\n\"@ | python -":    true,
		`sh -c "python -"`:                         true,
		"a || python -":                            true,
		"python <<<'print(1)'":                     true,
		"echo x | python - <<'EOF'\nprint(1)\nEOF": true,
		"cat <<-EOF > a.txt\n\tx\n\tEOF\npython -": true,
		"cat <<A <<B\nx\nA\ny\nB\nnode -":          true,
		"nohup python -":                           true,
		"env -u X python -":                        true,
		"time python -":                            true,
		"exec python -":                            true,
		"echo '@'\npython -":                       true,
		"x ;python -":                              true,
		"python -S \"C:/Users/micro/.claude/scripts/x.py\"": false,
		"python -c 'print(1)'":                              false,
		"python -m pytest -q":                               false,
		"python script.py arg":                              false,
		"echo 'print(1)' | python -":                        false,
		"python - < script.py":                              false,
		"python - 0<script.py":                              false,
		"go test ./... && python3 tools/x.py":               false,
		"node -e 'console.log(1)'":                          false,
		"cat <<'EOF' > file.txt\npython -\nEOF":             false,
		"git log --format=%B":                               false,
		"echo python":                                       false,
		"uv run python script.py":                           false,
		"uv run --script tool.py":                           false,
		"uv run":                                            false,
		"uv run pytest -q":                                  false,
		"python --version":                                  false,
		"python -V":                                         false,
		"node -v":                                           false,
		"perl -v":                                           false,
		"ruby -v":                                           false,
		"perl -ne 'print' f.txt":                            false,
		"perl -e'print 1'":                                  false,
		"python -uc 'print(1)'":                             false,
		"python -W ignore script.py":                        false,
		"python -Wignore script.py":                         false,
		"python script.py -":                                false,
		"node --eval=1":                                     false,
		"node --print 1":                                    false,
		"py -V:3.12 x.py":                                   false,
		"git commit -m \"fix; python -\"":                   false,
		"Get-Content x.py | python -":                       false,
		"cat <<'EOF' > a.md\nuv run -\nEOF":                 false,
		"cat <<\"END\" > a.md\nnode\nEND":                   false,
		"cat <<\\END > a.md\nperl\nEND":                     false,
		"$t = @'\npython -\n'@":                             false,
		"pythonic -":                                        false,
		"cat x | grep y | python script.py":                 false,
		"bash <<< 'echo hi'":                                false,
		"ruby -e 'p 1'":                                     false,
		"uv build -":                                        false,
		"python -- -u":                                      false,
		"sh -c \"cat <<EOF > f\npython -\nEOF\"":            false,
		"cat <<'EOF' > f\nx\nEOF\npython -":                 true,
		"perl -E'say 1'":                                    false,
		"node --input-type=module x.js":                     false,
		"python > out.txt":                                  true,
		"node --input-type module -":                        true,
		"python -c'print(1)'":                               false,
		"perl -I lib -":                                     true,
	}
}

// The rule refuses exactly the lines it means, in both modes and from every
// tool that carries a shell line.
func TestTheGuardRefusesAnInterpreterThatReadsStdin(t *testing.T) {
	root := t.TempDir()
	for line, want := range stdinLines() {
		for _, strict := range []bool{false, true} {
			got := slices.Contains(checkTool(root, "Bash", map[string]any{"command": line}, config.Policy{Strict: strict}), stdinReason)
			if got != want {
				t.Errorf("strict %v, %q: refused %v, want %v", strict, line, got, want)
			}
		}
	}
	for tool, input := range map[string]map[string]any{
		"PowerShell":  {"command": "python -"},
		"run_command": {"CommandLine": "python -"},
		"manage_task": {"Action": "send_input", "Input": "python -\n"},
	} {
		if !slices.Contains(checkTool(root, tool, input, config.Policy{}), stdinReason) {
			t.Errorf("%s: python - passes", tool)
		}
	}
}

// A heredoc's body is cut up to its end word, and a here-string becomes <<@.
func TestWithoutHereBodiesKeepsTheCommands(t *testing.T) {
	for line, want := range map[string]string{
		"cat <<EOF > f\nx\nEOF\npython -": "cat <<EOF > f\npython -",
		"cat <<-EOF\n\tx\n\tEOF\nls":      "cat <<-EOF\nls",
		"cat << 'E O'\nx":                 "cat << 'E O'",
		"cat <<EOF\r\nx\r\nEOF\r\nls":     "cat <<EOF\r\nls",
		"echo '<<EOF'\nls":                "echo '<<EOF'\nls",
		"bash <<< x\nls":                  "bash <<< x\nls",
		"@'\nx\n'@ | python -":            "<<@ | python -",
		"$a = @\"\nx\n\"@":                "$a = <<@",
		"(@'\nx":                          "(<<@",
		"echo '@'\nls":                    "echo '@'\nls",
		"cat <<A <<B\n1\nA\n2\nB\nls":     "cat <<A <<B\nls",
		"cat <<\nls":                      "cat <<\nls",
		"cat <<'E'\nx\nE\nls":             "cat <<'E'\nls",
		"cat <<\"E\"\nx\nE\nls":           "cat <<\"E\"\nls",
		"cat <<\\E\nx\nE\nls":             "cat <<\\E\nls",
		"bash <<<x\nls":                   "bash <<<x\nls",
	} {
		if got := withoutHereBodies(line); got != want {
			t.Errorf("%q: %q, want %q", line, got, want)
		}
	}
}
```

Danach `gofmt -l internal/hooks` muss leer sein (sonst `gofmt -w internal/hooks/guardstdin_test.go`).

In `internal/hooks/guardgate_test.go`:

`battery()` — nach der Schleife über `gateLines()`:

```go
	for line := range stdinLines() {
		lines = append(lines, line)
	}
```

`TestRecordTheGuardBattery` — Kommentar und Rumpf ersetzen, damit nur fehlende Zeilen aufgenommen werden und die Urteile früherer Änderungen stehen bleiben:

```go
// TestRecordTheGuardBattery writes what the guard says about every line of
// the battery the record lacks. It runs only when asked, once, at the state
// before a change to the guard that adds lines; the file it writes is the
// "before" the differential test reads. A line already recorded keeps its
// verdict, so each line's "before" stays the state before the change that
// brought it.
func TestRecordTheGuardBattery(t *testing.T) {
	if os.Getenv("LOOMUX_RECORD_GUARD_BATTERY") != "1" {
		t.Skip("set LOOMUX_RECORD_GUARD_BATTERY=1 to record the guard's verdicts")
	}
	before := map[string]bool{}
	if _, err := os.Stat(batteryFile); err == nil {
		before = readBattery(t)
	}
	root := t.TempDir()
	var b strings.Builder
	for _, line := range battery() {
		refused, recorded := before[line]
		if !recorded {
			refused = refusedBy(root, line)
		}
		verdict := "pass"
		if refused {
			verdict = "refused"
		}
		b.WriteString(verdict + "\t" + strconv.Quote(line) + "\n")
	}
```

(der Rest der Funktion — `MkdirAll`, `WriteFile` — bleibt.)

`TestTheGuardOpensNothingItRefusedBefore` — Kommentar ersetzen:

```go
// The differential probe: no line the guard refused before passes now, and
// the lines that flipped to a refusal are exactly the writes of the armed
// lanes, the calls that arm or disarm and the interpreters that read their
// program from stdin. The record predates the stdin rule; the armed lanes'
// lines were refused already when it was taken.
```

und nach der Schleife über `gateLines()` einfügen:

```go
	for line, refused := range stdinLines() {
		if refused {
			flips[line] = true
		}
	}
```

- [ ] **Step 3: Batterie am Stub-Stand aufnehmen**

Der Stub verweigert nichts, der Stand ist also der Basisstand für die Regel.

```bash
LOOMUX_RECORD_GUARD_BATTERY=1 go test ./internal/hooks -run TestRecordTheGuardBattery -count=1 > "$S/t1-record.txt" 2>&1
git diff --stat internal/hooks/testdata/guard-battery-before.tsv
git diff internal/hooks/testdata/guard-battery-before.tsv | grep -c '^-[pr]'
git diff internal/hooks/testdata/guard-battery-before.tsv | grep -c '^+refused'
```

Expected: `1 file changed, 90 insertions(+)`, dann `0` und `0` — keine bestehende Zeile geändert, keine neue Zeile vorher verweigert.

- [ ] **Step 4: RED zeigen**

```bash
go test ./internal/hooks -run 'Stdin|HereBodies|TestTheGuardOpensNothing|TestTheNamedGaps' -count=1 -v > "$S/t1-red.txt" 2>&1
grep -E '^--- FAIL' "$S/t1-red.txt"
grep -c 'refused false, want true' "$S/t1-red.txt"
grep -c 'still passes' "$S/t1-red.txt"
```

Expected:
```
--- FAIL: TestTheGuardOpensNothingItRefusedBefore
--- FAIL: TestTheGuardRefusesAnInterpreterThatReadsStdin
--- FAIL: TestWithoutHereBodiesKeepsTheCommands
88
44
```
Dazu je eine Zeile `PowerShell: python - passes`, `run_command: python - passes` und `manage_task: python - passes`.
Im Bericht: Befehl und die vollständige `--- FAIL`-Ausgabe (die Datei `$S/t1-red.txt` reicht als Beleg, der Controller liest sie).

- [ ] **Step 5: Die Regel schreiben**

`internal/hooks/guardstdin.go` ganz ersetzen, per Write:

```go
package hooks

import (
	"slices"
	"strings"
)

// stdinReason refuses an interpreter that would wait on stdin for its
// program, or get it as inline text a shell may have rewritten.
const stdinReason = "loomux refuses an interpreter that reads its program from stdin (`python -`, a bare `python`, `uv run -`, " +
	"a heredoc or here-string into one): with nothing piped in it waits until it is stopped, and a shell may rewrite " +
	"the backslashes of inline program text. Write the script to a file in the scratchpad with the host's file tool " +
	"(Write, write_to_file) and run that file."

// interpreter is how one interpreter reads its arguments, from its manual
// and measured with Python 3.13, Node 24 and Perl 5.42.
type interpreter struct {
	program []string // flags that carry the program on the line
	values  []string // flags whose value is the next word, or the rest of a bundle
	info    []string // whole words that print and exit without reading a program
}

var python = interpreter{
	program: []string{"-c", "-m"},
	values:  []string{"-W", "-X", "--check-hash-based-pycs"},
	info:    []string{"-V", "-VV", "--version", "-h", "-?", "--help", "--help-env", "--help-xoptions", "--help-all"},
}

// interpreters are keyed by verbOf of the program; python3.12 and its kin
// are looked up as python (interpreterOf).
var interpreters = map[string]interpreter{
	"python": python,
	"py":     python,
	"node": {
		program: []string{"-e", "--eval", "-p", "--print"},
		values:  []string{"-r", "--require", "--import", "--loader", "--experimental-loader", "-C", "--conditions", "--input-type"},
		info:    []string{"-v", "--version", "-h", "--help", "--v8-options"},
	},
	// -c checks the syntax of a program it reads like any other: from a
	// file, or from stdin.
	"perl": {program: []string{"-e", "-E"}, values: []string{"-I", "-M", "-m"}, info: []string{"-v", "-V", "-h"}},
	// -E takes an encoding, not a program.
	"ruby": {program: []string{"-e"}, values: []string{"-r", "-I", "-C", "-E"}, info: []string{"-v", "--version", "-h", "--help"}},
}

// uvValues are the flags of uv run and uvx whose value is the next word.
var uvValues = []string{"--with", "--with-editable", "--with-requirements", "--python", "-p", "--project",
	"--directory", "--package", "--extra", "--group", "--env-file", "--index", "--from"}

// readsProgramFromStdin says whether line, or a line it runs from a string,
// starts an interpreter whose program comes from stdin with nothing real on
// it: no file and no pipe from a command that prints.
func readsProgramFromStdin(line string) bool {
	return anyRunLine(withoutHereBodies(line), func(l string, nested bool) bool {
		// The outer line has lost its bodies already; a second pass would
		// take the lines after a heredoc's head for a body.
		if nested {
			l = withoutHereBodies(l)
		}
		return lineReadsStdin(l)
	})
}

// lineReadsStdin is readsProgramFromStdin for one line without the lines it
// runs from strings. It cuts only outside quotes: a quoted ; is data, and
// this rule keeps a task from hanging rather than an agent from a path.
func lineReadsStdin(line string) bool {
	for _, variant := range lineVariants(line) {
		prev, at := "", 0
		for _, segment := range splitSegments(variant, true) {
			// || leaves an empty segment between its bars; text inline on
			// the line is no stdin, piped or not.
			inline := strings.Contains(segment, "<<")
			piped := at > 0 && variant[at-1] == '|' && strings.TrimSpace(prev) != "" && !strings.Contains(prev, "<<")
			at += len(segment) + 1
			prev = segment
			for _, words := range readings(segment) {
				if programFromStdin(words) && (inline || !piped && !fromFile(words)) {
					return true
				}
			}
		}
	}
	return false
}

// fromFile says whether words redirect stdin with < or 0<. A segment with a
// heredoc or here-string never gets here as fed (lineReadsStdin).
func fromFile(words []string) bool {
	return slices.ContainsFunc(words, func(w string) bool {
		return strings.HasPrefix(strings.TrimPrefix(w, "0"), "<")
	})
}

// programFromStdin says whether words, a reading of one segment, run an
// interpreter that takes its program from stdin: directly, behind uv run or
// uvx, or as uv run - itself.
func programFromStdin(words []string) bool {
	program := dropPrefixes(words)
	if len(program) == 0 {
		return false
	}
	if args, ok := uvRunArgs(program); ok {
		if len(args) == 0 {
			// uv run without a command fails at once.
			return false
		}
		if args[0] == "-" {
			return true
		}
		program = args
	}
	in, ok := interpreterOf(program[0])
	return ok && in.fromStdin(program[1:])
}

// uvRunArgs are the words after uv run or uvx and their flags: the command
// they run, or - for a script on stdin.
func uvRunArgs(words []string) ([]string, bool) {
	var rest []string
	switch {
	case verbOf(words[0]) == "uvx":
		rest = words[1:]
	case verbOf(words[0]) == "uv" && len(words) > 1 && words[1] == "run":
		rest = words[2:]
	default:
		return nil, false
	}
	for len(rest) > 0 && len(rest[0]) > 1 && rest[0][0] == '-' {
		n := 1
		if slices.Contains(uvValues, rest[0]) {
			n = 2
		}
		rest = rest[min(n, len(rest)):]
	}
	return rest, true
}

// interpreterOf is the interpreter word names by its base name without
// .exe; python followed by a version (python3, python3.12) is python.
func interpreterOf(word string) (interpreter, bool) {
	name := verbOf(word)
	if version, ok := strings.CutPrefix(name, "python"); ok && strings.Trim(version, "0123456789.") == "" {
		name = "python"
	}
	in, ok := interpreters[name]
	return in, ok
}

// fromStdin says whether the interpreter, given args, reads its program from
// stdin: a lone - as its first positional, or no positional at all. A
// program flag or an info word means it does not; a redirection is no
// argument.
func (in interpreter) fromStdin(args []string) bool {
	flags := true
	for i := 0; i < len(args); i++ {
		a := args[i]
		if isRedirect, bare := redirection(a); isRedirect {
			if bare {
				i++
			}
			continue
		}
		switch {
		case flags && a == "--":
			flags = false
		case flags && slices.Contains(in.info, a):
			return false
		case flags && strings.HasPrefix(a, "--"):
			name, _, glued := strings.Cut(a, "=")
			if slices.Contains(in.program, name) {
				return false
			}
			if !glued && slices.Contains(in.values, name) {
				i++
			}
		case flags && len(a) > 1 && a[0] == '-':
			program, next := in.bundle(a)
			if program {
				return false
			}
			if next {
				i++
			}
		default:
			return a == "-"
		}
	}
	return true
}

// bundle reads a short flag word letter by letter (-uc, -Wignore, -lne):
// program says a letter carries the program, next that the last letter
// takes the next word as its value. A value letter before the last takes
// the rest of the word.
func (in interpreter) bundle(word string) (program, next bool) {
	for j := 1; j < len(word); j++ {
		letter := "-" + word[j:j+1]
		if slices.Contains(in.program, letter) {
			return true, false
		}
		if slices.Contains(in.values, letter) {
			return false, j == len(word)-1
		}
	}
	return false, false
}

// hereDoc is the end word of one heredoc a line opens, and whether it was
// opened with <<-, which strips leading tabs from the lines.
type hereDoc struct {
	word string
	dash bool
}

// withoutHereBodies is line without the bodies of its heredocs and with
// each PowerShell here-string replaced by the word <<@: what they hold is
// data, not commands. A heredoc's head (<<EOF) stays on its line, so that a
// segment still shows it takes inline text.
func withoutHereBodies(line string) string {
	lines := strings.Split(line, "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if head, closing, ok := hereStringHead(l); ok {
			j := i + 1
			for j < len(lines) && !strings.HasPrefix(lines[j], closing) {
				j++
			}
			if j < len(lines) {
				head += lines[j][len(closing):]
			}
			out = append(out, head)
			i = j
			continue
		}
		out = append(out, l)
		for _, doc := range hereDocs(l) {
			for i+1 < len(lines) {
				i++
				body := strings.TrimRight(lines[i], "\r")
				if doc.dash {
					body = strings.TrimLeft(body, "\t")
				}
				if body == doc.word {
					break
				}
			}
		}
	}
	return strings.Join(out, "\n")
}

// hereStringHead reads a line that opens a PowerShell here-string: it ends
// in @' or @" after a blank, = or ( or alone. head is the line with the
// opener replaced by <<@, closing the text a line must start with to end it.
func hereStringHead(l string) (head, closing string, ok bool) {
	t := strings.TrimRight(l, " \t\r")
	if !strings.HasSuffix(t, "@'") && !strings.HasSuffix(t, `@"`) {
		return "", "", false
	}
	if len(t) > 2 && !strings.ContainsRune(" \t=(", rune(t[len(t)-3])) {
		return "", "", false
	}
	return t[:len(t)-2] + "<<@", t[len(t)-1:] + "@", true
}

// hereDocs are the heredocs l opens, in order: << or <<- outside quotes,
// with the end word unquoted. The third < of a here-string's <<< ends the
// word before it starts, so <<< opens none.
func hereDocs(l string) []hereDoc {
	var out []hereDoc
	var quote byte
	for i := 0; i < len(l); i++ {
		c := l[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case strings.HasPrefix(l[i:], "<<"):
			rest, dash := strings.CutPrefix(l[i+2:], "-")
			rest = strings.TrimLeft(rest, " \t")
			end := strings.IndexAny(rest, " \t\r;|&<>()")
			if end < 0 {
				end = len(rest)
			}
			if word := strings.NewReplacer(`'`, "", `"`, "", `\`, "").Replace(rest[:end]); word != "" {
				out = append(out, hereDoc{word: word, dash: dash})
			}
			i++
		}
	}
	return out
}
```

- [ ] **Step 6: GREEN**

```bash
gofmt -l internal/hooks
go vet ./internal/hooks
go test ./internal/hooks -count=1 > "$S/t1-green.txt" 2>&1; echo "exit $?"; tail -3 "$S/t1-green.txt"
```

Expected: `gofmt` gibt nichts aus, `go vet` gibt nichts aus, `exit 0` und `ok  github.com/xidus90/loomux/internal/hooks`.

- [ ] **Step 7: Mutationsrunde**

Das Skript liegt im Scratchpad: `$S/mutate_real.py`. Es liest `internal/hooks/guardstdin.go` aus dem Baum, wendet je Mutant eine Ersetzung an, legt sie per Overlay (Pfade `C:/…`) über die Datei und fährt `go test ./internal/hooks -run 'Stdin|HereBodies'`. Ein Build-Fehler zählt als `BADMUTANT(build)`, nicht als getötet. Es enthält einen unveränderten Kontrollmutanten (`control-unchanged`), der überleben muss.

```bash
uv run --script "$S/mutate_real.py" > "$S/t1-mutants.txt" 2>&1; echo "exit $?"; grep -v '^killed' "$S/t1-mutants.txt"
```

Expected: genau eine Zeile `SURVIVED         control-unchanged`. Jeder andere Überlebende ist eine Testlücke: einen Fall in `stdinLines()` oder `TestWithoutHereBodiesKeepsTheCommands` ergänzen, der den Mutanten tötet, und die Runde wiederholen. Ein `BADMUTANT` heißt, der Suchtext passt nicht mehr zum Code: den Mutanten so umschreiben, dass er baut (`&& false`, `|| true`), nicht streichen. Ein neu hinzugekommener Zweig bekommt einen eigenen Mutanten in der Liste. Nach der Runde per `git status --short` prüfen, dass im Baum nur die Dateien dieses Tasks geändert sind.

- [ ] **Step 8: Coverage im echten Baum**

```bash
go test ./internal/hooks -count=1 -covermode=set -coverprofile="$S/t1-cover.out" > "$S/t1-cover.txt" 2>&1; echo "exit $?"
go tool cover -func="$S/t1-cover.out" | grep guardstdin.go | grep -v '100.0%'
```

Expected: `exit 0`, und die zweite Zeile gibt nichts aus (jede Funktion in `guardstdin.go` bei 100 %). Liegt eine darunter, ist die Abhilfe ein Fall in `stdinLines()` oder in der Tabelle von `TestWithoutHereBodiesKeepsTheCommands`, nie `//coverage:exempt`.

- [ ] **Step 9: Durch das echte Hook-Binary proben**

```bash
go build -o bin/loomux.new.exe ./cmd/loomux
go run ./cmd/loomux dev swap-binary --dir bin
```

Dann je eine Nutzlast per Write nach `$S/payload-refuse.json` und `$S/payload-pass.json`:

```json
{"tool_name": "Bash", "tool_input": {"command": "python3 - <<'EOF'\nprint(1)\nEOF"}}
```

```json
{"tool_name": "Bash", "tool_input": {"command": "cat <<'EOF' > notes.md\npython -\nEOF"}}
```

```bash
bin/loomux.exe hook pre-tool-use --host claude --root . < "$S/payload-refuse.json" > "$S/t1-hook-refuse.txt" 2>&1; echo "exit $?"
bin/loomux.exe hook pre-tool-use --host claude --root . < "$S/payload-pass.json" > "$S/t1-hook-pass.txt" 2>&1; echo "exit $?"
grep -c 'reads its program from stdin' "$S/t1-hook-refuse.txt"
```

Expected: `exit 2`, `exit 0`, dann eine Zahl ≥ 1.

- [ ] **Step 10: Commit**

```bash
git rev-parse --show-toplevel
git branch --show-current
git add internal/hooks/guardstdin.go internal/hooks/guardstdin_test.go internal/hooks/guard.go internal/hooks/guardgate_test.go internal/hooks/testdata/guard-battery-before.tsv
git commit -F "$S/t1-msg.txt" > "$S/t1-commit.txt" 2>&1; echo "exit $?"; tail -15 "$S/t1-commit.txt"
git log -1 --format='%an <%ae> %s'
```

`$S/t1-msg.txt` per Write:

```
feat(hooks): refuse an interpreter that reads its program from stdin

An agent that starts python -, a bare node or uv run - with nothing on
stdin waits until it is stopped, and a program written into a heredoc
reaches the interpreter after the shell may have rewritten its
backslashes. The guard now refuses such a call from every shell tool of
both hosts, in both modes: python, py, node, perl and ruby, also behind
uv run or uvx, read by each one's own flags. A pipe from a command or a
< file is real stdin; a heredoc or here-string is not, piped or not.

Heredoc bodies are data to this rule alone; the write rules still read
them as commands. The battery's recorder now adds only the lines its
record lacks, so each line keeps the verdict from before the change that
brought it.
```

Expected: `exit 0`, Autor `Christoph Wübbels <christoph.wuebbels@gmail.com>`. Das pre-commit-Tor läuft mit; bei Rot zuerst `grep -- '--- FAIL' "$S/t1-commit.txt"`.

---

### Task 2: Doku in beiden Sprachen

**Files:**
- Modify: `docs/en/hooks.md` (Tabelle der eingebauten Regeln, Zeile nach `loomux gate`, ca. 462; neuer Absatz nach „The armed lanes have named gaps“, ca. 464–469)
- Modify: `docs/de/hooks.md` (dieselben Stellen, ca. 486 und 488–493)
- Modify: `docs/en/configuration.md:66`, `docs/de/configuration.md:66`
- Modify: `README.md`, `README.de.md` (Abschnitt 1, nach dem Absatz mit dem Link auf `hooks.md`)

**Interfaces:**
- Consumes: `stdinReason` aus Task 1, wörtlich.
- Produces: nichts für spätere Tasks.

- [ ] **Step 1: `docs/en/hooks.md`**

Neue Tabellenzeile direkt nach der `loomux gate`-Zeile (kein `|` im Zelltext außer den Spaltentrennern):

```markdown
| Command | an interpreter (`python`, `python3`, `python3.N`, `py`, `node`, `perl`, `ruby`, also behind `uv run` or `uvx`, and `uv run -` itself) that takes its program from stdin with nothing real on it: a lone `-` or no program, with neither a pipe from a command nor a `<` file; a heredoc or here-string into it, or piped into it, never counts | loomux refuses an interpreter that reads its program from stdin (`python -`, a bare `python`, `uv run -`, a heredoc or here-string into one): with nothing piped in it waits until it is stopped, and a shell may rewrite the backslashes of inline program text. Write the script to a file in the scratchpad with the host's file tool (Write, write_to_file) and run that file. |
```

Neuer Absatz nach dem Absatz „**The armed lanes have named gaps** …“:

```markdown
**Interpreters that read stdin** (`internal/hooks/guardstdin.go`). An agent
that starts `python -` or a bare `node` with nothing on stdin waits until it
is stopped, and a program written inline in a heredoc reaches the
interpreter after the shell may have rewritten its backslashes. The rule
reads each interpreter's own flags: a flag that carries the program
(`python -c`/`-m`, `node -e`/`-p`, `perl -e`/`-E`, `ruby -e`) or a version
or help word (`python -V`, `node -v`, `perl -v`) passes, a flag that takes a
value skips it (`python -X utf8`, `perl -I lib`, `ruby -E utf-8`), short
flags are read as a bundle (`-uc`, `-lne`), and the first other word is the
script; `-` there, or no word at all, is stdin. `perl -c` and `python -E`
carry no program. stdin is real behind a `<` redirection or a pipe from a
segment without a heredoc (`echo 'print(1)' | python -`,
`Get-Content x.py | python -`, `python - < x.py`); a heredoc or here-string
into the interpreter, or piped into it (`cat <<EOF | python -`,
`@'…'@ | python -`), is not. Unlike the write rules it cuts the line only
outside quotes, so `git commit -m "fix; python -"` passes; a string a shell
runs (`sh -c "python -"`) is read as for the other command rules. A
heredoc's body and a here-string's text are data to this rule alone:
`cat <<'EOF' > notes.md` with `python -` in its body passes it, while the
write rules still read such a body as commands. It does not see
`python -i`, an interpreter in a variable set elsewhere than on the line,
interpreters outside the list (`pypy`, `deno`, `bun`, `php`), a value flag
of `node` or `uv` it does not know, whose value it takes for the script,
and an interpreter in a heredoc a shell runs (`cat <<'EOF' | sh` with
`python -` in the body). It refuses more than a shell would: a pipe
continued on the next line, `|&`, `echo 'python -' | sh`, a `<<` in quotes
before the pipe (`echo "<<" | python -`), and a heredoc an agent types into
an Antigravity task line by line.
```

- [ ] **Step 2: `docs/de/hooks.md`**

Tabellenzeile nach der `loomux gate`-Zeile (Grund englisch wie in den übrigen Zeilen):

```markdown
| Befehl | ein Interpreter (`python`, `python3`, `python3.N`, `py`, `node`, `perl`, `ruby`, auch hinter `uv run` oder `uvx`, und `uv run -` selbst), der sein Programm von stdin nimmt, ohne dass dort echter Inhalt ankommt: ein einzelnes `-` oder kein Programm, ohne Pipe aus einem Befehl und ohne `<`-Datei; ein Heredoc oder Here-String hinein, auch über eine Pipe, zählt nie | loomux refuses an interpreter that reads its program from stdin (`python -`, a bare `python`, `uv run -`, a heredoc or here-string into one): with nothing piped in it waits until it is stopped, and a shell may rewrite the backslashes of inline program text. Write the script to a file in the scratchpad with the host's file tool (Write, write_to_file) and run that file. |
```

Absatz nach „**Die scharfen Lanes haben benannte Lücken** …“:

```markdown
**Interpreter, die von stdin lesen** (`internal/hooks/guardstdin.go`). Ein
Agent, der `python -` oder ein nacktes `node` ohne Inhalt auf stdin startet,
wartet, bis er gestoppt wird, und ein Programm, das in einem Heredoc in der
Zeile steht, erreicht den Interpreter, nachdem die Shell seine Backslashes
umgeschrieben haben kann. Die Regel liest die Flaggen jedes Interpreters:
eine Flagge, die das Programm trägt (`python -c`/`-m`, `node -e`/`-p`,
`perl -e`/`-E`, `ruby -e`), oder ein Versions- oder Hilfewort
(`python -V`, `node -v`, `perl -v`) lässt durch, eine Flagge mit Wert
überspringt ihn (`python -X utf8`, `perl -I lib`, `ruby -E utf-8`), kurze
Flaggen werden als Bündel gelesen (`-uc`, `-lne`), und das erste andere
Wort ist das Skript; `-` an dieser Stelle oder gar kein Wort heißt stdin.
`perl -c` und `python -E` tragen kein Programm. Echtes stdin gibt es hinter
einer `<`-Umleitung oder einer Pipe aus einem Segment ohne Heredoc
(`echo 'print(1)' | python -`, `Get-Content x.py | python -`,
`python - < x.py`); ein Heredoc oder Here-String in den Interpreter, auch
über eine Pipe (`cat <<EOF | python -`, `@'…'@ | python -`), zählt nicht.
Anders als die Schreibregeln schneidet sie die Zeile nur außerhalb von
Anführungszeichen, `git commit -m "fix; python -"` geht also durch; einen
String, den eine Shell ausführt (`sh -c "python -"`), liest sie wie die
übrigen Befehlsregeln. Der Rumpf eines Heredocs und der Text eines
Here-Strings sind nur für diese Regel Daten: `cat <<'EOF' > notes.md` mit
`python -` im Rumpf lässt sie durch, die Schreibregeln lesen so einen Rumpf
weiter als Befehle. Sie sieht nicht `python -i`, einen Interpreter in einer
Variablen, die anderswo als in der Zeile gesetzt ist, Interpreter außerhalb
der Liste (`pypy`, `deno`, `bun`, `php`), eine Wertflagge von `node` oder
`uv`, die sie nicht kennt und deren Wert sie für das Skript hält, und einen
Interpreter in einem Heredoc, den eine Shell ausführt (`cat <<'EOF' | sh`
mit `python -` im Rumpf). Sie verweigert mehr, als eine Shell täte: eine
Pipe, die erst in der nächsten Zeile weitergeht, `|&`, `echo 'python -' |
sh`, ein `<<` in Anführungszeichen vor der Pipe (`echo "<<" | python -`) und
einen Heredoc, den ein Agent Zeile für Zeile in einen Antigravity-Task
tippt.
```

- [ ] **Step 3: `configuration.md` in beiden Sprachen**

`docs/en/configuration.md:66` — den ersten Satz

```markdown
One command rule is built in and needs no entry here: `git push`.
```

ersetzen durch

```markdown
The command rules for `git push`, for loomux's own commands that are a human's (`init`, `config`, `gate arm`, a gate's `--answer`) and for an interpreter that reads its program from stdin are built in and need no entry here.
```

`docs/de/configuration.md:66` — den ersten Satz

```markdown
Eine Befehlsregel ist eingebaut und braucht hier keinen Eintrag: `git push`.
```

ersetzen durch

```markdown
Eingebaut sind die Befehlsregeln für `git push`, für loomux' eigene Befehle, die einem Menschen gehören (`init`, `config`, `gate arm`, das `--answer` eines Gates), und für einen Interpreter, der sein Programm von stdin liest; sie brauchen hier keinen Eintrag.
```

Der Rest beider Zeilen bleibt.

- [ ] **Step 4: READMEs**

`README.md`, Abschnitt „1. The Autonomous Guard & Verification Loop“, nach der Zeile `Every phase with its payloads, exit codes and budgets: [hook lifecycle](docs/en/hooks.md).` eine Leerzeile und:

```markdown
Besides paths and loomux's own commands, the guard refuses an interpreter that would read its program from stdin with nothing on it (`python -`, a bare `node`, `uv run -`, a heredoc into one): such a call hangs until it is stopped. An agent writes the script to a file and runs that ([decision path](docs/en/hooks.md#7-the-decision-path-of-pre-tool-use)).
```

`README.de.md`, nach `Jede Phase mit Nutzlast, Exitcodes und Budgets: [Hook-Lebenszyklus](docs/de/hooks.md).`:

```markdown
Neben Pfaden und loomux' eigenen Befehlen verweigert der Wächter einen Interpreter, der sein Programm von stdin lesen würde, ohne dass dort etwas ankommt (`python -`, ein nacktes `node`, `uv run -`, ein Heredoc hinein): so ein Aufruf hängt, bis er gestoppt wird. Ein Agent schreibt das Skript in eine Datei und führt die aus ([Entscheidungsweg](docs/de/hooks.md#7-der-entscheidungsweg-von-pre-tool-use)).
```

- [ ] **Step 5: Prüfen**

```bash
grep -rn "One command rule is built in\|Eine Befehlsregel ist eingebaut" docs README*
grep -c "guardstdin.go" docs/en/hooks.md docs/de/hooks.md
grep -rn 'git.push' docs/en docs/de README.md README.de.md | grep -v 'push --force\|pushes\|pushed'
git diff --stat
```

Expected: der erste `grep` findet nichts, der zweite meldet je `1`. Der dritte zeigt nur die Payload-Beispiele in `hooks.md` (Zeilen 84 und 110 je Sprache), die Regeltabelle und die neuen Sätze in `configuration.md` — findet er eine weitere Aufzählung eingebauter Befehlsregeln, kommt die Interpreter-Regel dort dazu. `git diff --stat` nennt genau die sechs Dateien dieses Tasks. Dazu die Tabellen beider `hooks.md` lesen: die neue Zeile hat drei Zellen wie ihre Nachbarn.

- [ ] **Step 6: Commit als Fixup des Feature-Commits**

```bash
git rev-parse --show-toplevel
git branch --show-current
git add docs/en/hooks.md docs/de/hooks.md docs/en/configuration.md docs/de/configuration.md README.md README.de.md
git commit --fixup=HEAD > "$S/t2-commit.txt" 2>&1; echo "exit $?"; tail -10 "$S/t2-commit.txt"
```

Expected: `exit 0`. Der `release-pr`-Skill faltet den Fixup vor dem Push in den `feat(hooks)`-Commit.

---

## Nach den Tasks (Controller)

- Abschlussreview über den ganzen Zweig.
- `release-pr`-Skill: Fixups falten, `parse-body` prüfen, Label `release:minor`, `## Changelog` mit `### Added`; den Push-Befehl dem Menschen nennen und warten.
- Nach dem Push: `gh pr create … --assignee @me --label release:minor`, dann Auto-fix für den PR einschalten.
