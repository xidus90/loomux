# loomux Stufe 1a — Implementierungsplan

**Stand 2026-09-17: abgeschlossen.** Die fünf
Fertig-Kriterien der Fusions-Spec sind belegt: die 19 Fälle unter
`testdata/cases/1a` laufen in `TestRecordedCasesOfStage1a`, alle Zeilen von
`docs/.superpowers/parity/stufe-1a.md` sind freigegeben (die meisten am
2026-09-15 mit `f2d1fbf`, die nachgetragene `wiki-drift`-Zeile am 2026-09-17 mit
`299249c`), das Tor hält 100 % je Funktion, die Messung steht im Abschnitt
`2026-09-15 01:20` von `docs/{en,de}/benchmarks.md` (24,5 ms gegen 72 ms), und
der Pilot läuft (`4c5eda8`, `eeda71e`, Rauchtest `5f9c0b9` — das Binary direkt
gerufen, nicht aus einer neuen Claude-Sitzung). Die Mutationsrunde lief erst in
Stufe 1b-1; `internal/brain/guard` und `internal/hooks` sind dort geparkt und
nicht bewertet (`parity/stufe-1b-1-geparkte-mutanten.md`). Task 16 Step 7 lief
erst am 2026-09-17, weil 1b-1 und der Wiki-Umzug `loomux-src/ul` und
`loomux-src/ub` noch brauchten; beide Worktrees sind entfernt, die Tags
`loomux-1a-source` bleiben. **Die Kästchen sind nie gepflegt worden
und sagen nichts über den Fortschritt.** Der Messeintrag steht nach R15b in
`docs/{en,de}/`, nicht in `docs/benchmarks.md`.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ein Go-Binary `loomux` im neuen Repo, das den Hook-Pfad beider Altrepos (Policy + Schreibschranke in einem Wächter, post-edit, session-start), `lint`, `wiki-gate` und die Worktree-Befehle trägt, sich selbst als Pilot prüft und seine Parität über aufgezeichnete Fälle belegt.

**Architecture:** Vorhandener Go-Code aus `ultraloom` und `ultra-brain` wird mit seinen Tests umgezogen und auf 100 % Coverage gehoben; nur die Nahtstellen sind neu: ein Einstieg `cli.Run`, der vereinte Wächter, die In-Prozess-Wiki-Lane, der Fallrunner im Prozess und drei `dev`-Werkzeuge. Pakete, die 1a nicht benutzt, ziehen später um.

**Tech Stack:** Go ≥ 1.25 (`go 1.25.0` in `go.mod`, gebaut mit Go 1.27), `github.com/BurntSushi/toml v1.6.0`, `gopkg.in/yaml.v3 v3.0.1`, `golang.org/x/sys v0.18.0`. Kein weiteres Fremdpaket.

**Spec:** `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md` (liegt im ultraloom-Worktree `claude/ultra-loom-brain-fusion-a5bb17`; Task 1 kopiert sie nach loomux).

## Global Constraints

- **Wo die ausführende Sitzung läuft:** in `C:\Users\micro\Documents\#GIT\loomux`, nicht in ultraloom. Aus ultraloom heraus feuerten dessen Projekt-Hooks bei jedem `.go`-Edit im falschen Repo (`go vet ./...`, `ulinit check gofmt cmd internal`), jeder Stop führe ultraloom's Prüfkette, und `brain guard` verweigerte Writes unter `#GIT/loomux`. `~/.claude/settings.json` trägt am 2026-09-14 nur die globalen Hooks `ulguard worktree-link`/`worktree-unlink`, keinen `brain guard`; im loomux-Verzeichnis ohne `.ultraloom/config.toml` enden beide still. Bis Task 16 hat die Sitzung dort keine Schreibschranke — das ist der Preis des Piloten.
- **Pfade.** `LOOMUX=/c/Users/micro/Documents/#GIT/loomux` (neues Repo). Quellen nur aus den Tag-Worktrees `SRC_UL=/c/Users/micro/Documents/#GIT/loomux-src/ul` (ultraloom, Tag `loomux-1a-source`) und `SRC_UB=/c/Users/micro/Documents/#GIT/loomux-src/ub` (ultra-brain, Tag `loomux-1a-source`). Nie aus einem laufenden Checkout kopieren.
- **Modulpfad** `github.com/xidus90/loomux`; alle Pakete unter `internal/`, Einstieg `cmd/loomux`.
- **Lizenz** PolyForm Noncommercial 1.0, Datei `LICENSE.md` wortgleich aus `SRC_UB/LICENSE.md`.
- **Sprache.** Code, Bezeichner, Kommentare, Fehlermeldungen, Commit-Nachrichten, `AGENTS.md`, `CLAUDE.md`, `.claude/**` englisch. README und Benchmarks zweisprachig (`X.md` englisch, `X.de.md` deutsch). Pläne, Specs und `docs/.superpowers/parity/` deutsch.
- **Coverage 100 % je Funktion.** Das Tor ist `loomux dev covergate` (Task 2) über `go test ./... -covermode=set -coverpkg=./...`. Eine Funktion darf darunter liegen, wenn **die Zeile direkt über `func`** `//coverage:exempt <reason>` trägt, mit nicht leerem Grund.
- **Konfiguration.** Projekt: `.loomux/config.toml` — für Agenten nie beschreibbar (Spec, Fehlerverhalten); jeder Schritt, der diese Datei in einem echten Repo anlegt oder ändert, ist ein **Mensch-Schritt**. Global: `LOOMUX_STATE_DIR`, sonst `%LOCALAPPDATA%\loomux`, sonst `$XDG_STATE_HOME/loomux`, sonst `~/.local/state/loomux`.
- **Exit-Codes Wächter:** 0 erlauben, 2 verweigern, **nie 1**. session-start: 0 oder 1, nie 2.
- **Kein Push.** Kein Subagent pusht, kein `gh repo create` ohne ausdrückliches Ja des Nutzers im Chat. Nach jedem Subagenten `git -C "$LOOMUX" log -1 --format='%an <%ae>'` lesen.
- **Commits** unter der Nutzeridentität, ohne `Co-Authored-By`. Mehrzeilige Nachrichten über eine Datei und `git commit -F`. Vor jedem Commit `git branch --show-current`, `git rev-parse --short HEAD` und `git diff --cached --stat` lesen.
- **Subagenten** mit `model: "opus"`; ein Subagent je Task.
- **Umzugsregel.** Ein umgezogenes Paket behält Code und Tests wortgleich bis auf: Paketname, Importpfade, die im Task genannten Pfad-Literale, und neue Tests für 100 %. Jede weitere Änderung ist ein Befund im Task-Bericht.
- **Ein Shell-Befehl je Aufruf**, keine langen `&&`-Ketten.

## Dateistruktur nach Stufe 1a

| Pfad | Verantwortung | Herkunft |
|---|---|---|
| `cmd/loomux/main.go` | `os.Exit(cli.Run(...))` | neu |
| `internal/cli/` | Dispatch, Unterbefehle, Fall-Suite | neu, Teile aus `SRC_UB/cmd/brain`, `SRC_UL/cmd/guard`, `SRC_UL/cmd/init/check.go` |
| `internal/dev/covergate/` | Coverage-Tor | neu |
| `internal/dev/swap/` | Pilot-Binary tauschen | neu |
| `internal/dev/benchhooks/` | Hook-Messwerkzeug | neu (Harness vom 2026-09-14) |
| `internal/dev/recordcase/`, `internal/dev/importcases/` | Fälle aufzeichnen und übersetzen | neu |
| `internal/cases/` | Fallformat, Welt, Vergleich, Runner im Prozess | `SRC_UB/pkg/cases` |
| `internal/config/` | Manifest, Registry, StateDir, Policy | `SRC_UB/pkg/config` + neu `policy.go` |
| `internal/testlock/` | Test-Helfer: unlesbare Datei | `SRC_UB/internal/testlock` |
| `internal/gitenv/` | Git-Umgebung säubern | `SRC_UL/internal/gitenv` + `SRC_UB/pkg/gitenv` |
| `internal/gitwork/` | HEAD, Commit-Zeit, check-ignore | `SRC_UL/internal/gitwork` |
| `internal/hosts/` | Host-Naht | `SRC_UL/internal/hostio` |
| `internal/sessions/` | Sitzungszustand | `SRC_UL/internal/sessions` |
| `internal/detect/` | Projektfakten | `SRC_UL/internal/detect` |
| `internal/verify/`, `internal/verify/commit/` | gofmt-Prüfung, Commit-Nachricht | `SRC_UL/internal/verify` (nur `gofmt.go`), `SRC_UL/internal/commit` |
| `internal/worktree/topo/`, `…/junction/`, `…/mirror/` | Worktree-Spiegel | `SRC_UL/internal/{worktreetopo,junction,mirrorcfg}` |
| `internal/brain/guard/` | Schreibschranke | `SRC_UB/pkg/guard` |
| `internal/brain/check/`, `internal/brain/wiki/` | Befundmodell, Wiki-Lint, Wiki-Gate | `SRC_UB/pkg/{check,wiki}` |
| `internal/hooks/` | pre-tool-use, post-tool-use, session-start, status, worktree | `SRC_UL/cmd/guard` + neu |
| `testdata/cases/1a-source/`, `testdata/cases/1a/` | Aufzeichnungen alt, übersetzt | Task 14 |
| `docs/.superpowers/parity/stufe-1a.md` | Abweichungsliste | Task 14 |
| `.githooks/pre-commit`, `.githooks/commit-msg` | Tore | Task 2, 4 |

**Nicht in 1a** (ziehen später um): `SRC_UL` `answers`, `interview`, `render`, `tooling`, `write`, `settings`, `coverage`, `journal`, `cmd/init` außer `check.go`; `SRC_UB` `catalog`, `graph`, `index`, `layout`, `maintenance`, `mcp`, `privacy`, `reader`, `search`, `web`, `check/{code,house,okf,run}`, `cmd/brain` außer `lint`/`wiki-gate`. **Entfällt:** `vendoring`, `brainpath`, `internal/verify/types*` (dmypy, Stufe 2).

---

### Task 0: Vorbedingungen (Mensch, oder mit ausdrücklichem Ja im Chat)

**Files:** keine im Repo.

- [ ] **Step 1: Quelltags setzen**

```bash
git -C "/c/Users/micro/Documents/#GIT/ultraloom" tag loomux-1a-source 9d01a60
```
```bash
git -C "/c/Users/micro/Documents/#GIT/ultra-brain" tag loomux-1a-source 3cc72d2
```

- [ ] **Step 2: Tag-Worktrees als Lesequelle anlegen**

```bash
git -C "/c/Users/micro/Documents/#GIT/ultraloom" worktree add --detach "/c/Users/micro/Documents/#GIT/loomux-src/ul" loomux-1a-source
```
```bash
git -C "/c/Users/micro/Documents/#GIT/ultra-brain" worktree add --detach "/c/Users/micro/Documents/#GIT/loomux-src/ub" loomux-1a-source
```

- [ ] **Step 3 (nur falls doch aus ultraloom gearbeitet wird): loomux in der alten brain-Registry beschreibbar machen**

Entfällt, wenn die Sitzung wie vorgesehen in `#GIT/loomux` läuft. Sonst verweigert `brain guard` jeden Write unter `#GIT/loomux`. An `%LOCALAPPDATA%\brain\registry.toml` anhängen (vorher Sicherung `registry.toml.bak-loomux`):

```toml
[[area]]
scope     = "project/loomux"
path      = "C:/Users/micro/Documents/#GIT/loomux"
wiki      = "C:/Users/micro/Documents/#GIT/loomux/docs/wiki"
workspace = true
```

- [ ] **Step 4: Repo-Verzeichnis anlegen**

```bash
git init "/c/Users/micro/Documents/#GIT/loomux"
```

- [ ] **Step 5 (optional, nur Mensch):** GitHub-Repo `xidus90/loomux` privat anlegen und als `origin` eintragen. Kein Push in dieser Stufe.

- [ ] **Step 6: Prüfen**

```bash
git -C "/c/Users/micro/Documents/#GIT/loomux-src/ul" rev-parse --short HEAD
```
Expected: `9d01a60`. Dasselbe für `ub`: `3cc72d2`.

### Task 1: Gerüst und Einstieg `cli.Run`

**Files:**
- Create: `go.mod`, `.gitignore`, `LICENSE.md`, `AGENTS.md`, `CLAUDE.md`, `README.md`, `README.de.md`
- Create: `cmd/loomux/main.go`, `internal/cli/cli.go`, `internal/cli/commands.go`
- Test: `internal/cli/cli_test.go`
- Create: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`, `docs/.superpowers/plans/2026-09-14-loomux-stufe-1a.md` (Kopien aus dem ultraloom-Worktree)

**Interfaces:**
- Produces: `cli.Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int`; `cli.Version string`; `type command func(args []string, stdin io.Reader, stdout, stderr io.Writer) int`; `var commands map[string]command` in `commands.go` — jede spätere Task trägt dort genau eine Zeile ein.

- [ ] **Step 1: Modul anlegen**

```bash
cd "/c/Users/micro/Documents/#GIT/loomux"
```
```bash
go mod init github.com/xidus90/loomux
```
Dann in `go.mod` die Zeile `go …` auf `go 1.25.0` setzen.

- [ ] **Step 2: Rahmendateien schreiben**

`.gitignore`:
```
/bin/
/coverage.out
*.test
```

`LICENSE.md`: wortgleiche Kopie.
```bash
cp "/c/Users/micro/Documents/#GIT/loomux-src/ub/LICENSE.md" "/c/Users/micro/Documents/#GIT/loomux/LICENSE.md"
```

`AGENTS.md`:
```markdown
# loomux

One Go binary for the hook path, the check chain and the knowledge system
that `ultraloom` and `ultra-brain` provided separately. Design:
`docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`.

## Where things live

- `cmd/loomux` is the entry point and nothing else; every command lives under `internal/`.
- Specs, plans and parity lists live under `docs/.superpowers/`.
- Recorded behaviour of the old tools lives under `testdata/cases/`.

## Languages

Everything that instructs an LLM, and everything that is not prose, is
English: this file, `CLAUDE.md`, `.claude/**`, code, comments, error
messages, commit messages. Documentation is bilingual: `X.md` is English and
the standard, `X.de.md` sits beside it. Working papers under
`docs/.superpowers/` are German and never translated.

## Rules

- `.loomux/config.toml` is never written by an agent. It declares the areas a
  write barrier trusts and the policy that guards edits; propose changes, a
  human writes them.
- Coverage is 100% per function. A function may stay below only with
  `//coverage:exempt <reason>` on the line directly above `func`.
- No `init()` and no package-level variable parses embedded data; load on first use.
- Commits carry the user as author and committer and credit no model or agent.
- Nobody but a human pushes.
- Performance measurements go chronologically into `docs/benchmarks.md` and
  `docs/benchmarks.de.md`: date and time, what was measured, baseline against
  change, cold and warm.

## Commands

- Gate: `.githooks/pre-commit` (gofmt, go vet, tests with coverage, pilot binary).
- `go run ./cmd/loomux dev covergate --profile coverage.out`
```

`CLAUDE.md`:
```markdown
@AGENTS.md

# For Claude Code only

- Subagents never push; after a subagent run read `git log -1 --format='%an <%ae>'`.
- The hooks in `.claude/settings.json` call `bin/loomux.exe`, which the
  pre-commit gate rebuilds. If session-start warns that the binary predates
  HEAD, rebuild before trusting a refusal.
```

`README.md`:
```markdown
# loomux

[Deutsch](README.de.md)

One Go binary for hooks, checks and an LLM wiki. Stage 1a of the merge of
`ultraloom` and `ultra-brain`; not ready for other projects yet.

Licence: PolyForm Noncommercial 1.0 (`LICENSE.md`).
```

`README.de.md`:
```markdown
# loomux

[English](README.md)

Ein Go-Binary für Hooks, Prüfungen und ein LLM-Wiki. Stufe 1a der Fusion von
`ultraloom` und `ultra-brain`; für andere Projekte noch nicht bereit.

Lizenz: PolyForm Noncommercial 1.0 (`LICENSE.md`).
```

Spec und Plan kopieren:
```bash
mkdir -p "/c/Users/micro/Documents/#GIT/loomux/docs/.superpowers/specs" "/c/Users/micro/Documents/#GIT/loomux/docs/.superpowers/plans"
```
```bash
cp "/c/Users/micro/Documents/#GIT/ultraloom/.claude/worktrees/ultra-loom-brain-fusion-a5bb17/docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md" "/c/Users/micro/Documents/#GIT/loomux/docs/.superpowers/specs/"
```
```bash
cp "/c/Users/micro/Documents/#GIT/ultraloom/.claude/worktrees/ultra-loom-brain-fusion-a5bb17/docs/.superpowers/plans/2026-09-14-loomux-stufe-1a.md" "/c/Users/micro/Documents/#GIT/loomux/docs/.superpowers/plans/"
```

- [ ] **Step 3: Failing test schreiben** — `internal/cli/cli_test.go`

```go
package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func run(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Run(args, strings.NewReader(""), &out, &errb)
	return code, out.String(), errb.String()
}

func TestVersionPrintsTheVersion(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-v"} {
		code, out, _ := run(arg)
		if code != 0 || out != "loomux "+Version+"\n" {
			t.Fatalf("%s: code %d, out %q", arg, code, out)
		}
	}
}

func TestHelpGoesToStdoutAndSucceeds(t *testing.T) {
	code, out, _ := run("help")
	if code != 0 || !strings.Contains(out, "Usage: loomux <command>") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestNoArgumentsIsAUsageError(t *testing.T) {
	code, _, errOut := run()
	if code != 2 || !strings.Contains(errOut, "Usage: loomux <command>") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestUnknownCommandIsAUsageError(t *testing.T) {
	code, _, errOut := run("frobnicate")
	if code != 2 || !strings.Contains(errOut, `unknown command "frobnicate"`) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestKnownCommandReceivesTheRest(t *testing.T) {
	var got []string
	commands["probe"] = func(args []string, _ io.Reader, _, _ io.Writer) int {
		got = args
		return 7
	}
	defer delete(commands, "probe")
	code, _, _ := run("probe", "a", "b")
	if code != 7 || strings.Join(got, ",") != "a,b" {
		t.Fatalf("code %d, args %v", code, got)
	}
}
```

- [ ] **Step 4: Test laufen lassen, er scheitert**

Run: `go test ./internal/cli/`
Expected: FAIL, `undefined: Run`.

- [ ] **Step 5: Implementieren**

`internal/cli/cli.go`:
```go
// Package cli is the one entry point of loomux: it maps the first argument to
// a command and returns the exit code instead of exiting, so tests and the
// case suite can drive every command in-process.
package cli

import (
	"fmt"
	"io"
	"sort"
)

// Version is overwritten at build time with -ldflags "-X".
var Version = "0.0.0-dev"

type command func(args []string, stdin io.Reader, stdout, stderr io.Writer) int

// Run dispatches one invocation and returns its exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "loomux %s\n", Version)
		return 0
	case "help", "--help", "-h":
		usage(stdout)
		return 0
	}
	cmd, ok := commands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "loomux: unknown command %q\n", args[0])
		usage(stderr)
		return 2
	}
	return cmd(args[1:], stdin, stdout, stderr)
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage: loomux <command> [arguments]")
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(w, "  %s\n", name)
	}
}
```

`internal/cli/commands.go`:
```go
package cli

// commands is the whole command table. Each stage-1a task adds its line here.
var commands = map[string]command{}
```

`cmd/loomux/main.go`:
```go
package main

import (
	"os"

	"github.com/xidus90/loomux/internal/cli"
)

//coverage:exempt process entry; every decision lives in cli.Run, which is tested
func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
```

- [ ] **Step 6: Tests laufen lassen, sie bestehen**

Run: `go test ./...`
Expected: `ok github.com/xidus90/loomux/internal/cli`.

- [ ] **Step 7: Commit**

```bash
git add .
```
Nachricht in `$TEMP/loomux-msg.txt`: `Start loomux with one entry point that returns exit codes`, dann:
```bash
git commit -F "$TEMP/loomux-msg.txt"
```

---

### Task 2: Coverage-Tor, Binärtausch, Pre-Commit

**Files:**
- Create: `internal/dev/covergate/covergate.go`, Test `covergate_test.go`
- Create: `internal/dev/swap/swap.go`, Test `swap_test.go`
- Create: `internal/cli/dev.go`, Test `dev_test.go`
- Modify: `internal/cli/commands.go` (Zeile `"dev": devCommand,`)
- Create: `.githooks/pre-commit`

**Interfaces:**
- Produces: `covergate.Line{File string; Line int; Func string; Percent float64}`; `covergate.Parse(r io.Reader) ([]Line, error)`; `covergate.Gate(lines []Line, module string, read func(string) ([]byte, error), w io.Writer) int`; `swap.Swap(dir string) error`; `devCommand` mit Unterbefehlen `covergate --profile P` und `swap-binary --dir D`. Spätere Tasks hängen `bench-hooks`, `record-case`, `import-cases` an `devCommand`.

- [ ] **Step 1: Failing tests** — `internal/dev/covergate/covergate_test.go`

```go
package covergate

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const sample = "github.com/xidus90/loomux/internal/a/a.go:10:\tFull\t\t100.0%\n" +
	"github.com/xidus90/loomux/internal/a/a.go:20:\tPartial\t\t85.7%\n" +
	"github.com/xidus90/loomux/cmd/loomux/main.go:9:\tmain\t\t0.0%\n" +
	"total:\t\t\t\t(statements)\t98.0%\n"

func TestParseReadsEveryFunctionLineAndSkipsTotal(t *testing.T) {
	lines, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 || lines[1] != (Line{File: "github.com/xidus90/loomux/internal/a/a.go", Line: 20, Func: "Partial", Percent: 85.7}) {
		t.Fatalf("%+v", lines)
	}
}

func TestParseRefusesAMalformedLine(t *testing.T) {
	if _, err := Parse(strings.NewReader("garbage\n")); err == nil {
		t.Fatal("want error")
	}
	if _, err := Parse(strings.NewReader("x.go:abc:\tF\t1.0%\n")); err == nil {
		t.Fatal("want error for a line number that is not a number")
	}
	if _, err := Parse(strings.NewReader("x.go:1:\tF\tabc%\n")); err == nil {
		t.Fatal("want error for a percentage that is not a number")
	}
}

func files(m map[string]string) func(string) ([]byte, error) {
	return func(p string) ([]byte, error) {
		if s, ok := m[p]; ok {
			return []byte(s), nil
		}
		return nil, errors.New("no such file")
	}
}

func TestGatePassesWhenEveryShortfallIsExempt(t *testing.T) {
	lines, _ := Parse(strings.NewReader(sample))
	read := files(map[string]string{
		"internal/a/a.go":    strings.Repeat("\n", 18) + "//coverage:exempt the error arm needs a full disk\nfunc Partial() {}\n",
		"cmd/loomux/main.go": strings.Repeat("\n", 7) + "//coverage:exempt process entry\nfunc main() {}\n",
	})
	var out bytes.Buffer
	if code := Gate(lines, "github.com/xidus90/loomux", read, &out); code != 0 {
		t.Fatalf("code %d: %s", code, out.String())
	}
}

func TestGateFailsAndNamesEveryUnexemptShortfall(t *testing.T) {
	lines, _ := Parse(strings.NewReader(sample))
	read := files(map[string]string{
		"internal/a/a.go":    strings.Repeat("\n", 18) + "//coverage:exempt \nfunc Partial() {}\n",
		"cmd/loomux/main.go": "func main() {}\n",
	})
	var out bytes.Buffer
	code := Gate(lines, "github.com/xidus90/loomux", read, &out)
	if code != 1 || !strings.Contains(out.String(), "internal/a/a.go:20 Partial 85.7%") ||
		!strings.Contains(out.String(), "cmd/loomux/main.go:9 main 0.0%") {
		t.Fatalf("code %d: %s", code, out.String())
	}
}

func TestGateFailsWhenTheSourceCannotBeRead(t *testing.T) {
	lines, _ := Parse(strings.NewReader(sample))
	var out bytes.Buffer
	if code := Gate(lines, "github.com/xidus90/loomux", files(nil), &out); code != 1 {
		t.Fatalf("code %d", code)
	}
}
```

- [ ] **Step 2: Scheitern sehen**

Run: `go test ./internal/dev/covergate/`
Expected: FAIL, `undefined: Parse`.

- [ ] **Step 3: Implementieren** — `internal/dev/covergate/covergate.go`

```go
// Package covergate turns `go tool cover -func` output into a gate: every
// function at 100%, or an exemption with a reason written above it.
package covergate

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const marker = "//coverage:exempt "

// Line is one function row of `go tool cover -func`.
type Line struct {
	File    string
	Line    int
	Func    string
	Percent float64
}

// Parse reads the rows and skips the closing total.
func Parse(r io.Reader) ([]Line, error) {
	var lines []Line
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "total:") || strings.TrimSpace(text) == "" {
			continue
		}
		fields := strings.Fields(text)
		if len(fields) != 3 {
			return nil, fmt.Errorf("unexpected cover line %q", text)
		}
		location := strings.Split(strings.TrimSuffix(fields[0], ":"), ":")
		if len(location) < 2 {
			return nil, fmt.Errorf("unexpected location in %q", text)
		}
		number, err := strconv.Atoi(location[len(location)-1])
		if err != nil {
			return nil, fmt.Errorf("line number in %q: %w", text, err)
		}
		percent, err := strconv.ParseFloat(strings.TrimSuffix(fields[2], "%"), 64)
		if err != nil {
			return nil, fmt.Errorf("percentage in %q: %w", text, err)
		}
		lines = append(lines, Line{
			File:    strings.Join(location[:len(location)-1], ":"),
			Line:    number,
			Func:    fields[1],
			Percent: percent,
		})
	}
	return lines, scanner.Err()
}

// Gate reports every function below 100% whose declaration is not preceded
// by an exemption with a reason, and returns 1 if there is any.
func Gate(lines []Line, module string, read func(string) ([]byte, error), w io.Writer) int {
	code := 0
	for _, l := range lines {
		if l.Percent >= 100 {
			continue
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(l.File, module), "/")
		if exempt(read, rel, l.Line) {
			continue
		}
		fmt.Fprintf(w, "not covered: %s:%d %s %.1f%%\n", rel, l.Line, l.Func, l.Percent)
		code = 1
	}
	return code
}

func exempt(read func(string) ([]byte, error), path string, line int) bool {
	data, err := read(path)
	if err != nil {
		return false
	}
	rows := strings.Split(string(data), "\n")
	if line < 2 || line-2 >= len(rows) {
		return false
	}
	above := strings.TrimSpace(rows[line-2])
	return strings.HasPrefix(above+" ", marker) && strings.TrimSpace(strings.TrimPrefix(above, strings.TrimSpace(marker))) != ""
}
```

- [ ] **Step 4: Bestehen sehen** — `go test ./internal/dev/covergate/` → PASS. Fehlt eine Zeile an 100 %, einen Test für genau diesen Arm ergänzen (etwa `line < 2`).

- [ ] **Step 5: Failing tests** — `internal/dev/swap/swap_test.go`

```go
package swap

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSwapPutsTheNewBinaryInPlaceAndKeepsTheOld(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "loomux.exe"), "old")
	write(t, filepath.Join(dir, "loomux.new.exe"), "new")
	if err := Swap(dir); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(dir, "loomux.exe")) != "new" || read(t, filepath.Join(dir, "loomux.old.exe")) != "old" {
		t.Fatal("binaries not swapped")
	}
	if _, err := os.Stat(filepath.Join(dir, "loomux.new.exe")); !os.IsNotExist(err) {
		t.Fatal("new binary still there")
	}
}

func TestSwapWorksWithoutAPreviousBinary(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "loomux.new.exe"), "new")
	if err := Swap(dir); err != nil || read(t, filepath.Join(dir, "loomux.exe")) != "new" {
		t.Fatalf("err %v", err)
	}
}

func TestSwapRefusesWithoutANewBinary(t *testing.T) {
	if err := Swap(t.TempDir()); err == nil {
		t.Fatal("want error")
	}
}
```

- [ ] **Step 6: Scheitern sehen** — `go test ./internal/dev/swap/` → FAIL `undefined: Swap`.

- [ ] **Step 7: Implementieren** — `internal/dev/swap/swap.go`

```go
// Package swap replaces the pilot binary while hooks may be running it.
// Windows lets a running executable be renamed but not overwritten, so the
// current one is moved aside first.
package swap

import (
	"fmt"
	"os"
	"path/filepath"
)

// Swap moves dir/loomux.new.exe to dir/loomux.exe, keeping the previous one
// as dir/loomux.old.exe.
func Swap(dir string) error {
	current := filepath.Join(dir, "loomux.exe")
	next := filepath.Join(dir, "loomux.new.exe")
	old := filepath.Join(dir, "loomux.old.exe")
	if _, err := os.Stat(next); err != nil {
		return fmt.Errorf("no new binary at %s: %w", next, err)
	}
	// Best effort: an older binary may still be running and cannot be removed.
	_ = os.Remove(old)
	if _, err := os.Stat(current); err == nil {
		if err := os.Rename(current, old); err != nil {
			return fmt.Errorf("moving %s aside: %w", current, err)
		}
	}
	if err := os.Rename(next, current); err != nil {
		return fmt.Errorf("putting %s in place: %w", next, err)
	}
	return nil
}
```
Die beiden `Rename`-Fehlerarme sind ohne Dateisystemfehler nicht erreichbar: entweder über ein schreibgeschütztes Verzeichnis testen (`requireWindows` + `icacls`, Muster aus `SRC_UL/cmd/guard/worktree_test.go:17`) oder `Swap` mit `//coverage:exempt <reason>` markieren und den Grund im Bericht nennen.

- [ ] **Step 8: Failing tests** — `internal/cli/dev_test.go`

```go
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevNeedsASubcommand(t *testing.T) {
	code, _, errOut := run("dev")
	if code != 2 || !strings.Contains(errOut, "loomux dev: subcommand required") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevSwapBinary(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "loomux.new.exe"), []byte("x"), 0o755)
	if code, _, errOut := run("dev", "swap-binary", "--dir", dir); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if code, _, _ := run("dev", "swap-binary", "--dir", dir); code != 1 {
		t.Fatalf("second swap without a new binary must fail, got %d", code)
	}
}

func TestDevCovergateReadsTheCoverTool(t *testing.T) {
	coverFunc = func(profile string) ([]byte, error) {
		return []byte("github.com/xidus90/loomux/internal/cli/cli.go:1:\tRun\t100.0%\n"), nil
	}
	defer func() { coverFunc = runCoverFunc }()
	if code, _, errOut := run("dev", "covergate", "--profile", "c.out"); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
}
```
Weitere Arme (unbekannter Unterbefehl → 2, fehlerhafte Flags → 2, `coverFunc` liefert Fehler → 1, `Parse` scheitert → 1) je mit einem Test derselben Form.

- [ ] **Step 9: Implementieren** — `internal/cli/dev.go`

```go
package cli

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/xidus90/loomux/internal/dev/covergate"
	"github.com/xidus90/loomux/internal/dev/swap"
)

const module = "github.com/xidus90/loomux"

var coverFunc = runCoverFunc

//coverage:exempt starts the go toolchain; the gate's logic is tested through coverFunc
func runCoverFunc(profile string) ([]byte, error) {
	return exec.Command("go", "tool", "cover", "-func="+profile).Output()
}

var devCommands = map[string]command{
	"covergate":   devCovergate,
	"swap-binary": devSwapBinary,
}

func devCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "loomux dev: subcommand required")
		return 2
	}
	sub, ok := devCommands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "loomux dev: unknown subcommand %q\n", args[0])
		return 2
	}
	return sub(args[1:], stdin, stdout, stderr)
}

func devCovergate(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev covergate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	profile := fs.String("profile", "coverage.out", "coverage profile written by go test")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	out, err := coverFunc(*profile)
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev covergate: %v\n", err)
		return 1
	}
	lines, err := covergate.Parse(bytes.NewReader(out))
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev covergate: %v\n", err)
		return 1
	}
	return covergate.Gate(lines, module, os.ReadFile, stdout)
}

func devSwapBinary(args []string, _ io.Reader, _, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev swap-binary", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", "bin", "directory holding loomux.new.exe")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := swap.Swap(*dir); err != nil {
		fmt.Fprintf(stderr, "loomux dev swap-binary: %v\n", err)
		return 1
	}
	return 0
}
```
In `commands.go` die Zeile `"dev": devCommand,` eintragen.

- [ ] **Step 10: Pre-Commit-Tor** — `.githooks/pre-commit`

```sh
#!/bin/sh
# Pre-commit gate of loomux itself (stage 1a): format, vet, tests with
# per-function coverage, then rebuild the pilot binary the hooks call.
set -eu
cd "$(git rev-parse --show-toplevel)"
unformatted=$(gofmt -l cmd internal)
if [ -n "$unformatted" ]; then
	echo "gofmt: these files are not formatted:" >&2
	echo "$unformatted" >&2
	exit 1
fi
go vet ./...
go test ./... -covermode=set -coverpkg=./... -coverprofile=coverage.out
go run ./cmd/loomux dev covergate --profile coverage.out
mkdir -p bin
go build -o bin/loomux.new.exe ./cmd/loomux
go run ./cmd/loomux dev swap-binary --dir bin
```
```bash
git config core.hooksPath .githooks
```

- [ ] **Step 11: Tor einmal von Hand fahren**

Run: `sh .githooks/pre-commit`
Expected: Exit 0, `bin/loomux.exe` existiert.

- [ ] **Step 12: Commit** (das Tor läuft dabei selbst) — Nachricht: `Gate every commit on per-function coverage and rebuild the pilot binary`.

---

## Umzugsverfahren (gilt für die Tasks 3–10)

Jeder Umzug folgt denselben Schritten; die Tasks nennen nur Quelle, Ziel, Paketname und die geänderten Literale.

1. Dateien kopieren, Tests eingeschlossen, `testdata/` und `templates/` eingeschlossen:
   ```bash
   mkdir -p "$LOOMUX/<ziel>"
   ```
   ```bash
   cp -r "$SRC/<quelle>/." "$LOOMUX/<ziel>/"
   ```
2. Paketnamen setzen (nur wenn er sich ändert), in Nicht-Test- und Testdateien:
   ```bash
   sed -i 's/^package <alt>$/package <neu>/; s/^package <alt>_test$/package <neu>_test/' "$LOOMUX/<ziel>"/*.go
   ```
3. Importpfade im **ganzen** loomux-Baum umschreiben, dann Paketbezeichner:
   ```bash
   grep -rl '<alter importpfad>' "$LOOMUX" --include=*.go | xargs -r sed -i 's#<alter importpfad>#<neuer importpfad>#g'
   ```
   ```bash
   grep -rl '<neuer importpfad>' "$LOOMUX" --include=*.go | xargs -r sed -i 's/\b<alt>\./<neu>./g'
   ```
4. `go mod tidy` (holt die Fremdpakete, die das umgezogene Paket importiert), `gofmt -w` auf das Ziel, dann `go vet ./<ziel>/...` und `go test ./<ziel>/...` — grün, bevor irgendetwas anderes geändert wird. Nach `go mod tidy` die Versionen in `go.mod` auf die im Tech Stack genannten setzen (`go get github.com/BurntSushi/toml@v1.6.0` usw.). ultra-brain war gegen toml **v1.4.0** grün: ein Test, der auf toml-Fehlerwortlaut prüft und unter v1.6.0 scheitert, ist ein Befund für den Bericht, kein Anlass, die Version zu senken.
5. Die im Task genannten Literale ändern; Tests, die das alte Literal prüfen, im selben Schritt auf das neue umstellen.
6. Coverage heben:
   ```bash
   go test ./<ziel>/... -covermode=set -coverpkg=./... -coverprofile="$TEMP/pkg.out"
   ```
   ```bash
   go tool cover -func="$TEMP/pkg.out"
   ```
   Für jede Zeile unter 100 %: `go tool cover -html="$TEMP/pkg.out" -o "$TEMP/pkg.html"` zeigt die roten Blöcke. Je Block ein Test, der genau ihn ausführt (Test zuerst, rot sehen, dann grün). Ist ein Block ohne Eingriff ins Betriebssystem nicht erreichbar, bekommt die Funktion `//coverage:exempt <reason>`; der Grund nennt, was fehlen müsste, damit der Block läuft.
7. `sh .githooks/pre-commit`, dann Commit.

---

### Task 3: `gitenv` aus beiden Repos zusammenführen

**Files:**
- Create: `internal/gitenv/gitenv.go`, `internal/gitenv/gitenv_test.go` (aus `SRC_UL/internal/gitenv`)
- Create: `internal/gitenv/scan_test.go` (aus `SRC_UB/pkg/gitenv/scan_test.go`)
- Nicht übernommen: `SRC_UB/pkg/gitenv/gitenv_test.go` — es liest `src/brain/maintenance/vcs.py`; Eintrag in die Paritätsliste (Task 14).

**Interfaces:**
- Produces: `gitenv.Location []string` (Namen), `gitenv.Clean(parent []string) []string`, `gitenv.Environ() []string`.

- [ ] **Step 1:** `SRC_UL/internal/gitenv` nach `internal/gitenv` umziehen (Verfahren, Schritte 1–4; Importpfad `github.com/xidus90/ultra-loom/internal/gitenv` → `github.com/xidus90/loomux/internal/gitenv`).
- [ ] **Step 2: Failing test** — in `internal/gitenv/gitenv_test.go` ergänzen:

```go
func TestCleanDropsEveryInheritedGitVariable(t *testing.T) {
	parent := []string{
		"PATH=/bin",
		"GIT_DIR=/x", "GIT_CONFIG_PARAMETERS='a=b'", "GIT_AUTHOR_NAME=Brain",
		"GIT_CONFIG_KEY_0=user.name", "GIT_CONFIG_VALUE_0=Brain",
		"GIT_TRACE=1",
	}
	got := Clean(parent)
	want := []string{"PATH=/bin", "GIT_TRACE=1"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %v, want %v", got, want)
	}
}
```
`GIT_TRACE` bleibt, weil keine der beiden Listen es nennt; `GIT_AUTHOR_NAME` fällt, weil ultra-brains Liste es nennt (Grund: Memory „ultra-brain: lokale Git-Identität“ — Tests schrieben „Brain“ ins Repo).

- [ ] **Step 3:** `go test ./internal/gitenv/` → FAIL (`GIT_CONFIG_PARAMETERS` u. a. bleiben stehen).
- [ ] **Step 4: Implementieren** — in `gitenv.go` `Location` durch die vollständige Liste aus `SRC_UB/pkg/gitenv/gitenv.go` ersetzen (alle 28 Namen, in dieser Reihenfolge) und die Präfixe ergänzen:

```go
// numbered names are GIT_CONFIG_KEY_<n> and GIT_CONFIG_VALUE_<n>, whose count is open.
var numbered = []string{"GIT_CONFIG_KEY_", "GIT_CONFIG_VALUE_"}

func Clean(parent []string) []string {
	cleaned := make([]string, 0, len(parent))
	for _, entry := range parent {
		name, _, found := strings.Cut(entry, "=")
		if found && (contains(Location, name) || hasAnyPrefix(name, numbered)) {
			continue
		}
		cleaned = append(cleaned, entry)
	}
	return cleaned
}

func hasAnyPrefix(name string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
```
Den Paketkommentar um einen Satz ergänzen: die Liste ist ultra-brains, weil sie auch Identität und Konfiguration abschneidet.

- [ ] **Step 5:** `go test ./internal/gitenv/` → PASS.
- [ ] **Step 6: Scan-Test übernehmen.** `SRC_UB/pkg/gitenv/scan_test.go` kopieren, Paket `gitenv`. Er läuft vom Paketverzeichnis `../..` zur Modulwurzel — dieselbe Tiefe wie in ultra-brain, also unverändert. In der Funktion, die `Clean` erkennt (`scan_test.go:64`), `selector.Sel.Name == "Clean"` zu `selector.Sel.Name == "Clean" || selector.Sel.Name == "Environ"` erweitern.
- [ ] **Step 7:** `go test ./internal/gitenv/` → PASS. Coverage 100 % (Verfahren Schritt 6).
- [ ] **Step 8: Commit** — `Merge both gitenv packages under the wider list of inherited variables`.

---

### Task 4: Hosts, Sitzungszustand, Git-Fragen, `check gofmt` und `check commit-msg`

**Files:**
- Move: `SRC_UL/internal/hostio` → `internal/hosts` (Paket `hosts`)
- Move: `SRC_UL/internal/sessions` → `internal/sessions`
- Move: `SRC_UL/internal/gitwork` → `internal/gitwork`
- Move: `SRC_UL/internal/verify/gofmt.go`, `gofmt_test.go` → `internal/verify/` (nur diese zwei Dateien)
- Move: `SRC_UL/internal/commit` → `internal/verify/commit` (Paket `commit`)
- Create: `internal/cli/check.go`, Test `internal/cli/check_test.go`
- Modify: `internal/cli/commands.go` (`"check": checkCommand,`)
- Create: `.githooks/commit-msg`

**Interfaces:**
- Consumes: `gitenv.Environ()` (Task 3).
- Produces: `hosts.Host`, `hosts.ParseHost(string) (Host, error)`, `hosts.Payload{Event, SessionID string}`, `hosts.Read(Host, io.Reader) (Payload, error)`, `hosts.WriteContext(Host, io.Writer, []string) error`, `hosts.FindRoot(start string) (string, error)`, `hosts.ErrNoRoot`; `sessions.StateDir`, `sessions.ReadState(root, id string) SessionState`, `sessions.WriteState(root, id string, s SessionState) error`, `sessions.Others`, `sessions.Forget`; `gitwork.HeadCommit(root string) (string, error)`; `verify.CheckGoFormat(paths []string) ([]string, error)`; `commit.ValidateCommitMessage(string) error`; `checkCommand` mit `gofmt [paths…]` und `commit-msg <file>`.

- [ ] **Step 1:** Die fünf Umzüge nach dem Verfahren, Schritte 1–4.
- [ ] **Step 2: Literale.**
  - `internal/hosts/hostio.go`: `ErrNoRoot`-Text und `findRoot` suchen `.loomux/config.toml` statt `.ultraloom/config.toml` (`filepath.Join(current, ".loomux", "config.toml")`); Paketkommentar „ulinit writes“ → „loomux init writes“.
  - `internal/sessions/sessions.go:24`: `const StateDir = ".loomux/state/hooks"`.
  - Tests beider Pakete auf die neuen Literale umstellen.
- [ ] **Step 3:** `go test ./internal/hosts/ ./internal/sessions/ ./internal/gitwork/ ./internal/verify/...` → PASS.
- [ ] **Step 4: Failing tests** — `internal/cli/check_test.go`

```go
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckNeedsASubcommand(t *testing.T) {
	if code, _, errOut := run("check"); code != 2 || !strings.Contains(errOut, "loomux check: subcommand required") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckCommitMsgAcceptsEnglish(t *testing.T) {
	file := filepath.Join(t.TempDir(), "msg")
	os.WriteFile(file, []byte("Add the first loomux command\n"), 0o644)
	if code, _, errOut := run("check", "commit-msg", file); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestCheckCommitMsgRefusesGerman(t *testing.T) {
	file := filepath.Join(t.TempDir(), "msg")
	os.WriteFile(file, []byte("Füge den ersten Befehl hinzu und prüfe die Änderung\n"), 0o644)
	if code, _, _ := run("check", "commit-msg", file); code != 1 {
		t.Fatalf("code %d", code)
	}
}

func TestCheckGofmtNamesUnformattedFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.go"), []byte("package x\nfunc  F(){}\n"), 0o644)
	code, out, _ := run("check", "gofmt", dir)
	if code != 1 || !strings.Contains(out, "x.go") {
		t.Fatalf("code %d, out %q", code, out)
	}
}
```
Dazu je ein Test für: `commit-msg` ohne Datei → 2, unlesbare Datei → 1, `gofmt` mit sauberem Verzeichnis → 0, unbekannter Unterbefehl → 2.

- [ ] **Step 5:** Scheitern sehen: `go test ./internal/cli/` → FAIL `undefined: checkCommand`.
- [ ] **Step 6: Implementieren** — `internal/cli/check.go` übernimmt die Arme `commit-msg` und `gofmt` aus `SRC_UL/cmd/init/check.go:49-90` wörtlich in dieser Form (Meldungspräfix `loomux check`, Exit-Codes wie dort):

```go
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/xidus90/loomux/internal/verify"
	"github.com/xidus90/loomux/internal/verify/commit"
)

func checkCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "loomux check: subcommand required: commit-msg, gofmt")
		return 2
	}
	switch args[0] {
	case "commit-msg":
		if len(args) != 2 {
			fmt.Fprintln(stderr, "loomux check commit-msg: exactly one message file required")
			return 2
		}
		content, err := os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintf(stderr, "loomux check commit-msg: %v\n", err)
			return 1
		}
		if err := commit.ValidateCommitMessage(string(content)); err != nil {
			fmt.Fprintf(stderr, "loomux check commit-msg: %v\n", err)
			return 1
		}
		return 0
	case "gofmt":
		unformatted, err := verify.CheckGoFormat(args[1:])
		if err != nil {
			fmt.Fprintf(stderr, "loomux check gofmt: %v\n", err)
			return 1
		}
		for _, file := range unformatted {
			fmt.Fprintln(stdout, file)
		}
		if len(unformatted) > 0 {
			return 1
		}
		return 0
	}
	fmt.Fprintf(stderr, "loomux check: unknown subcommand %q\n", args[0])
	return 2
}
```
Weicht `SRC_UL/cmd/init/check.go` in einem Arm von diesem Code ab (Exit-Code, Textausgabe, leere Pfadliste), gilt die Quelle; die Abweichung im Bericht nennen. `commands.go`: `"check": checkCommand,`.

- [ ] **Step 7:** `go test ./internal/cli/` → PASS; Coverage 100 % für alle fünf Pakete und `check.go`.
- [ ] **Step 8: commit-msg-Hook** — `.githooks/commit-msg`

```sh
#!/bin/sh
# Commit messages of loomux are English; the pilot binary checks them.
set -eu
cd "$(git rev-parse --show-toplevel)"
if [ -x bin/loomux.exe ]; then
	exec bin/loomux.exe check commit-msg "$1"
fi
exec go run ./cmd/loomux check commit-msg "$1"
```

- [ ] **Step 9: Commit** — `Move the host seam, session state and the two message checks`.

---

### Task 5: Worktree-Spiegel und Projektfakten

**Files:**
- Move: `SRC_UL/internal/worktreetopo` → `internal/worktree/topo` (Paket `topo`)
- Move: `SRC_UL/internal/junction` → `internal/worktree/junction` (Paket `junction`, Build-Tags bleiben)
- Move: `SRC_UL/internal/mirrorcfg` → `internal/worktree/mirror` (Paket `mirror`)
- Move: `SRC_UL/internal/detect` → `internal/detect`

**Interfaces:**
- Consumes: `gitenv.Environ()`.
- Produces: `mirror.Mirror(root string) ([]string, error)`; `detect.Detect(fsys fs.FS) Facts` mit `Facts.Stacks []string`, `Facts.WikiPath string`, `Facts.GodotDir string`, `Facts.WikiMode string`; die exportierten Namen von `topo` und `junction` unverändert.

- [ ] **Step 1:** Vier Umzüge, Verfahren Schritte 1–4. Nach dem Umbenennen `worktreetopo.` → `topo.` und `mirrorcfg.` → `mirror.` in allen Aufrufern.
- [ ] **Step 2: Literale.**
  - `internal/worktree/mirror/mirrorcfg.go`: gelesen wird `.loomux/config.toml`, Tabelle `[worktree]`, Schlüssel `mirror` bleibt. Den Paketsatz „every other table belongs to the Python side“ streichen; neu: „every other table belongs to internal/config“.
  - `internal/detect/detect.go:105-108`: `manifestNames = []string{".loomux/config.toml"}`; der Kommentar verweist auf `internal/config` statt auf `ultra-brain/pkg/config/manifest.go`.
  - Tests beider Pakete umstellen.
- [ ] **Step 3:** `go test ./internal/worktree/... ./internal/detect/` → PASS (auf Windows; die Windows-Tests laufen).
- [ ] **Step 4: Coverage heben.** Bekannt unter 100 % (Messung vom 2026-09-14 in ultraloom): `detect.go:200 matches` 88,9 %; `junction_windows.go:30 Create` 92,9 %, `:55 setMountPoint` 93,3 %, `:137 reparseTarget` 81,0 %. Die Junction-Arme, die nur ein fehlschlagender `DeviceIoControl` erreicht, bekommen `//coverage:exempt` mit dem Namen des Systemaufrufs als Grund; `matches` bekommt Tests.
- [ ] **Step 5:** `sh .githooks/pre-commit` → 0.
- [ ] **Step 6: Commit** — `Move the worktree mirror and project detection under the new config path`.

---

### Task 6: `config` — Manifest, Registry, StateDir, Policy

**Files:**
- Move: `SRC_UB/pkg/config` → `internal/config` (Paket bleibt `config`)
- Move: `SRC_UB/internal/testlock` → `internal/testlock`
- Create: `internal/config/policy.go`, Test `internal/config/policy_test.go`

**Interfaces:**
- Produces: `config.StateDir() string`; `config.ReadRegistry(stateDir string) ([]config.Area, error)`; `config.ReadManifest(repoRoot string) (*config.Manifest, error)` (Felder wie `SRC_UB/pkg/config/manifest.go:97-110`); `config.ErrNoManifest`; neu `config.ManifestPath(root string) string`, `config.Policy{Paths []config.PathRule; Commands []config.CommandRule}`, `config.PathRule{Match []string; Reason string}`, `config.CommandRule{Regex *regexp.Regexp; Source, Reason string}`, `config.ReadPolicy(root string) (config.Policy, error)`.

- [ ] **Step 1:** Beide Umzüge, Verfahren Schritte 1–4 (`github.com/xidus90/ultra-brain/pkg/config` → `github.com/xidus90/loomux/internal/config`, `github.com/xidus90/ultra-brain/internal/testlock` → `github.com/xidus90/loomux/internal/testlock`).
- [ ] **Step 2: Literale.**
  - `registry.go:128`: `const stateDirEnv = "LOOMUX_STATE_DIR"`.
  - `registry.go:167,169,172,174`: `"brain"` → `"loomux"`.
  - `manifest.go:45-48`: `var manifestNames = []string{filepath.Join(".loomux", "config.toml")}`; Kommentar darüber: „The one manifest name. loomux cut the two spellings of ultra-brain (`.ultra-brain/config.toml`, `.brain.toml`) on 2026-09-14; `guard.declarationIn` repeats this name for the write barrier.“
  - Tests, die `BRAIN_STATE_DIR`, `brain` als Verzeichnisnamen oder die zwei alten Manifestnamen prüfen, auf die neuen Werte umstellen; Tests, die **die Reihenfolge der zwei Namen** prüfen, entfallen — im Bericht aufzählen.
- [ ] **Step 3:** `go test ./internal/config/ ./internal/testlock/` → PASS.
- [ ] **Step 4: Failing tests** — `internal/config/policy_test.go`

```go
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".loomux"), 0o755)
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestReadPolicyWithoutAConfigIsEmpty(t *testing.T) {
	policy, err := ReadPolicy(t.TempDir())
	if err != nil || len(policy.Paths) != 0 || len(policy.Commands) != 0 {
		t.Fatalf("%+v, %v", policy, err)
	}
}

func TestReadPolicyAcceptsAStringOrAListOfGlobs(t *testing.T) {
	root := writeConfig(t, `
[[policy.paths.rules]]
match  = "generated/*"
reason = "generated"

[[policy.paths.rules]]
match  = ["coverage.out", "bin/*"]
reason = "build output"

[[policy.commands.rules]]
regex  = '(^|[\n;&|(`+"`"+`])\s*pip\s+install([^\w-]|$)'
reason = "uv, never pip"
`)
	policy, err := ReadPolicy(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(policy.Paths) != 2 || strings.Join(policy.Paths[1].Match, ",") != "coverage.out,bin/*" || policy.Paths[0].Match[0] != "generated/*" {
		t.Fatalf("%+v", policy.Paths)
	}
	if len(policy.Commands) != 1 || !policy.Commands[0].Regex.MatchString("uv run x; pip install y") {
		t.Fatalf("%+v", policy.Commands)
	}
}

func TestReadPolicyRefusesARegexGoCannotCompile(t *testing.T) {
	root := writeConfig(t, `
[[policy.commands.rules]]
regex  = 'git\s+push(?![\w-])'
reason = "lookahead"
`)
	_, err := ReadPolicy(root)
	if err == nil || !strings.Contains(err.Error(), "config.toml") || !strings.Contains(err.Error(), "git\\s+push(?![\\w-])") {
		t.Fatalf("err %v", err)
	}
}

func TestReadPolicyRefusesUnreadableTOML(t *testing.T) {
	root := writeConfig(t, "[[policy.paths.rules]\n")
	if _, err := ReadPolicy(root); err == nil || !strings.Contains(err.Error(), "config.toml") {
		t.Fatalf("err %v", err)
	}
}

func TestReadPolicyRefusesAMatchOfTheWrongType(t *testing.T) {
	root := writeConfig(t, "[[policy.paths.rules]]\nmatch = 3\nreason = \"x\"\n")
	if _, err := ReadPolicy(root); err == nil {
		t.Fatal("want error")
	}
}
```
Dazu Tests für: leere `reason` → Fehler; Listenelement kein String → Fehler; Datei vorhanden, aber unlesbar (`testlock.Lock`) → Fehler.

- [ ] **Step 5:** `go test ./internal/config/` → FAIL `undefined: ReadPolicy`.
- [ ] **Step 6: Implementieren** — `internal/config/policy.go`

```go
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/BurntSushi/toml"
)

// PathRule refuses writes to paths matching any of its globs.
type PathRule struct {
	Match  []string
	Reason string
}

// CommandRule refuses shell commands its expression matches.
type CommandRule struct {
	Regex  *regexp.Regexp
	Source string
	Reason string
}

// Policy is the [policy] table of .loomux/config.toml.
type Policy struct {
	Paths    []PathRule
	Commands []CommandRule
}

// ManifestPath is where a project's one configuration file lives.
func ManifestPath(root string) string {
	return filepath.Join(root, ".loomux", "config.toml")
}

type policyFile struct {
	Policy struct {
		Paths struct {
			Rules []struct {
				Match  any    `toml:"match"`
				Reason string `toml:"reason"`
			} `toml:"rules"`
		} `toml:"paths"`
		Commands struct {
			Rules []struct {
				Regex  string `toml:"regex"`
				Reason string `toml:"reason"`
			} `toml:"rules"`
		} `toml:"commands"`
	} `toml:"policy"`
}

// ReadPolicy reads the policy of the project at root. A missing file is an
// empty policy; every other failure is an error naming the file, because the
// guard refuses on it and the human who fixes the file needs to find it.
//
// Every expression is compiled here. ulguard dropped the compile error and a
// rule with a lookahead never matched, without a word (spec, 2026-09-14).
func ReadPolicy(root string) (Policy, error) {
	path := ManifestPath(root)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Policy{}, nil
	}
	if err != nil {
		return Policy{}, fmt.Errorf("%s: %w", path, err)
	}
	var file policyFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return Policy{}, fmt.Errorf("%s: %w", path, err)
	}
	var policy Policy
	for i, rule := range file.Policy.Paths.Rules {
		globs, err := globList(rule.Match)
		if err != nil {
			return Policy{}, fmt.Errorf("%s: [[policy.paths.rules]] #%d match: %w", path, i+1, err)
		}
		if rule.Reason == "" {
			return Policy{}, fmt.Errorf("%s: [[policy.paths.rules]] #%d needs a reason", path, i+1)
		}
		policy.Paths = append(policy.Paths, PathRule{Match: globs, Reason: rule.Reason})
	}
	for i, rule := range file.Policy.Commands.Rules {
		compiled, err := regexp.Compile(rule.Regex)
		if err != nil {
			return Policy{}, fmt.Errorf("%s: [[policy.commands.rules]] #%d regex %s does not compile: %w", path, i+1, rule.Regex, err)
		}
		if rule.Reason == "" {
			return Policy{}, fmt.Errorf("%s: [[policy.commands.rules]] #%d needs a reason", path, i+1)
		}
		policy.Commands = append(policy.Commands, CommandRule{Regex: compiled, Source: rule.Regex, Reason: rule.Reason})
	}
	return policy, nil
}

func globList(value any) ([]string, error) {
	switch v := value.(type) {
	case string:
		return []string{v}, nil
	case []any:
		globs := make([]string, 0, len(v))
		for _, element := range v {
			text, ok := element.(string)
			if !ok {
				return nil, fmt.Errorf("list element %v is not a string", element)
			}
			globs = append(globs, text)
		}
		return globs, nil
	}
	return nil, fmt.Errorf("must be a string or a list of strings, found %T", value)
}
```

- [ ] **Step 7:** `go test ./internal/config/` → PASS, Coverage 100 %. `testlock` erreicht 100 % über die Tests der nutzenden Pakete (`-coverpkg=./...`); was nur unter POSIX läuft (`lock_other.go`), bekommt `//coverage:exempt built only off Windows; the gate runs on Windows`.
- [ ] **Step 8: Commit** — `Move the brain config under one manifest name and compile policy rules on load`.

---

### Task 7: Schreibschranke `brain/guard`

**Files:**
- Move: `SRC_UB/pkg/guard` → `internal/brain/guard`
- Create: `internal/brain/guard/scratchpad.go`, Test `scratchpad_test.go`
- Create: `internal/brain/guard/export_test.go` nur falls für neue Tests nötig

**Interfaces:**
- Consumes: nichts aus loomux außer `config` (wie in ultra-brain).
- Produces: `guard.Run(stdin io.Reader, stdout, stderr io.Writer, stateDir string) int` (unverändert), `guard.Decide(payload map[string]any, stateDir string) (string, bool)` (unverändert), neu `guard.Refuse(stdout, stderr io.Writer, reason string) int`, neu `guard.Call(payload map[string]any) (name string, args map[string]any)` (exportiertes `extractCall`), neu `guard.WriteTargets(args map[string]any) []string` (die Werte von `file_path`, `notebook_path`, `TargetFile`, `target_file`, wie `Decide` sie heute sammelt), neu `guard.IsWritingTool(name string) bool`.

- [ ] **Step 1:** Umzug, Verfahren Schritte 1–4.
- [ ] **Step 2: Literale** — `manifest.go:19-23`:

```go
// bundleDir and manifestName name the one manifest loomux reads. ultra-brain
// had a second, legacy spelling; loomux cut it on 2026-09-14.
const (
	bundleDir    = ".loomux"
	manifestName = "config.toml"
)
```
`declarationIn` (`manifest.go:294-304`) verliert den `legacy`-Arm. `guard.go:247-252` (`isManifest`): `config.toml` zählt nur in `.loomux`; `.brain.toml` zählt nicht mehr. Tests auf `.loomux/config.toml` umstellen; Tests, die nur den Altnamen prüften, entfallen (im Bericht aufzählen).

- [ ] **Step 3:** `go test ./internal/brain/guard/` → PASS.
- [ ] **Step 4: Exporte, test-first.** In `run_test.go` ergänzen:

```go
func TestRefuseWritesBothChannelsAndBlocks(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Refuse(&out, &errb, "no"); code != 2 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(out.String(), `"permissionDecision": "deny"`) || errb.String() != "no\n" {
		t.Fatalf("out %q err %q", out.String(), errb.String())
	}
}
```
In `guard_test.go` (oder dem Test, der `extractCall` heute prüft) die Aufrufe auf `Call` umstellen und ergänzen:

```go
func TestWriteTargetsCollectsEveryKnownKey(t *testing.T) {
	got := WriteTargets(map[string]any{"file_path": "a", "notebook_path": "b", "TargetFile": "c", "target_file": "d", "other": "e", "file_path_empty": ""})
	if strings.Join(got, ",") != "a,b,c,d" {
		t.Fatalf("%v", got)
	}
}

func TestIsWritingTool(t *testing.T) {
	if !IsWritingTool("Edit") || !IsWritingTool("write_to_file") || IsWritingTool("Bash") {
		t.Fatal("wrong classification")
	}
}
```
Rot sehen, dann: `extractCall` → `Call` umbenennen (alle Aufrufer); `Run` ruft `Refuse` statt der zwei Zeilen `stdout.Write`/`lastWord`; `WriteTargets` und `IsWritingTool` aus dem Code von `Decide` herauslösen (Schritte 1–2 der Entscheidung), sodass `Decide` sie selbst benutzt. Reihenfolge der Schlüssel in `WriteTargets` = die Reihenfolge, in der `Decide` sie heute liest; weicht sie von `file_path, notebook_path, TargetFile, target_file` ab, gilt der Code und der Test wird angepasst.

- [ ] **Step 5: Scratchpad-Ausnahme, test-first** — `scratchpad_test.go`

```go
package guard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScratchpadOfAClaudeSessionIsOpen(t *testing.T) {
	temp := t.TempDir()
	tempDir = func() string { return temp }
	defer func() { tempDir = os.TempDir }()
	file := filepath.Join(temp, "claude", "C--project", "0b1c-session", "scratchpad", "notes.txt")
	os.MkdirAll(filepath.Dir(file), 0o755)
	base := scratchpadBase()
	resolved, err := resolvePath(file)
	if err != nil {
		t.Fatal(err)
	}
	if !isScratchpad(resolved, base) {
		t.Fatalf("%s not recognised below %s", resolved, base)
	}
}

func TestOnlyTheScratchpadDirectoryIsOpen(t *testing.T) {
	temp := t.TempDir()
	tempDir = func() string { return temp }
	defer func() { tempDir = os.TempDir }()
	base := scratchpadBase()
	for _, rel := range []string{
		filepath.Join("claude", "C--project", "0b1c-session", "tasks", "x.output"),
		filepath.Join("claude", "C--project", "scratchpad", "x"),
		filepath.Join("claude", "C--project", "0b1c-session", "scratchpad"),
		filepath.Join("other", "C--project", "0b1c-session", "scratchpad", "x"),
	} {
		path := filepath.Join(temp, rel)
		os.MkdirAll(filepath.Dir(path), 0o755)
		resolved, _ := resolvePath(path)
		if isScratchpad(resolved, base) {
			t.Fatalf("%s must stay closed", rel)
		}
	}
}

func TestScratchpadBaseIsEmptyWhenTempIsNotAbsolute(t *testing.T) {
	tempDir = func() string { return "relative" }
	defer func() { tempDir = os.TempDir }()
	if scratchpadBase() != "" {
		t.Fatal("a relative temp directory must open nothing")
	}
}
```
Und ein Ende-zu-Ende-Test über `Decide`: Registry mit einem Bereich, Ziel im Scratchpad → erlaubt; Ziel daneben unter `tasks/` → verweigert.

- [ ] **Step 6: Implementieren** — `scratchpad.go`

```go
package guard

import (
	"os"
	"path/filepath"
)

// The scratchpad of a Claude Code session is the third place outside the
// registered trees the barrier leaves open (loomux spec, parity list of stage
// 1a). Claude Code hands every session a directory
// <temp>/claude/<project>/<session>/scratchpad for its throwaway files; the
// barrier refused it on 2026-09-14 and the session had nowhere else to put them.

var tempDir = os.TempDir

// scratchpadBase resolves <temp>/claude, or "" where temp is not absolute or
// does not resolve. Like the memory bases it may only ever open.
func scratchpadBase() string {
	temp := tempDir()
	if !filepath.IsAbs(temp) {
		return ""
	}
	resolved, err := resolvePath(filepath.Join(temp, "claude"))
	if err != nil {
		return ""
	}
	return resolved
}

// isScratchpad reports whether a resolved target lies below
// <base>/<project>/<session>/scratchpad/, counted in components.
func isScratchpad(resolved, base string) bool {
	if base == "" {
		return false
	}
	rest, ok := below(spelled(resolved), base)
	return ok && len(rest) >= 4 && rest[2] == "scratchpad"
}
```
In `Decide` an beiden Stellen, an denen heute `isMemory(...)` entscheidet (`guard.go:428` und die Sammlung der Ziele außerhalb aller Wurzeln), `|| isScratchpad(path, scratch)` ergänzen, mit `scratch := scratchpadBase()` einmal neben `memoryBases()`. Die Ablehnungsmeldung nennt den Scratchpad-Baum neben den Gedächtnisbäumen (`filepath.Join(base, "*", "*", "scratchpad")`).

- [ ] **Step 7:** `go test ./internal/brain/guard/` → PASS; Coverage 100 % (Paket stand in ultra-brain bei 100 %).
- [ ] **Step 8: Commit** — `Move the write barrier, cut the legacy manifest name and open the session scratchpad`.

---

### Task 8: Wiki-Lint und Wiki-Gate als Befehle

**Files:**
- Move: `SRC_UB/pkg/check/check.go` samt Tests → `internal/brain/check` (nur das Basispaket, nicht `code`, `house`, `okf`, `run`)
- Move: `SRC_UB/pkg/wiki` → `internal/brain/wiki`
- Create: `internal/brain/wiki/root.go`, `internal/brain/wiki/report.go`, Tests `root_test.go`, `report_test.go`
- Create: `internal/cli/wiki.go`, Test `internal/cli/wiki_test.go`
- Modify: `internal/cli/commands.go` (`"lint": lintCommand,` und `"wiki-gate": wikiGateCommand,`)

**Interfaces:**
- Consumes: `config.ReadManifest(root)`, `config.ErrNoManifest` (Task 6), `gitenv.Environ()` (Task 3).
- Produces: `wiki.Root(projectRoot string) string` (ersetzt `FindWikiPath`); `wiki.LintReport(target, wikiRoot string, w io.Writer) int`; `wiki.GateReport(projectRoot string, stdout, stderr io.Writer) int`; `wiki.CheckWikiGate(projectRoot string) []wiki.GateViolation` (unverändert bis auf `Root`); `lintCommand`, `wikiGateCommand`.

- [ ] **Step 1:** Beide Umzüge, Verfahren Schritte 1–4 — mit einer Ausnahme in Schritt 1 für `check`: nur die Dateien des Basispakets, nicht seine Unterverzeichnisse (`code`, `house`, `okf`, `run` importieren `cases` und `config` und ziehen später um):
  ```bash
  mkdir -p "$LOOMUX/internal/brain/check"
  ```
  ```bash
  cp "$SRC_UB/pkg/check/"*.go "$LOOMUX/internal/brain/check/"
  ```
- [ ] **Step 2: Failing tests** — `root_test.go`

```go
package wiki

import (
	"os"
	"path/filepath"
	"testing"
)

func mkdir(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(parts...)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRootPrefersTheManifestLayout(t *testing.T) {
	root := t.TempDir()
	want := mkdir(t, root, "knowledge", "pages")
	mkdir(t, root, "docs", "wiki")
	mkdir(t, root, ".loomux")
	os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[area]\nscope = \"project/x\"\n[layout]\nwiki = \"knowledge/pages\"\n"), 0o644)
	if got := Root(root); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRootFallsBackToDocsWikiThenWiki(t *testing.T) {
	root := t.TempDir()
	want := mkdir(t, root, "wiki")
	if got := Root(root); got != want {
		t.Fatalf("got %q", got)
	}
	want = mkdir(t, root, "docs", "wiki")
	if got := Root(root); got != want {
		t.Fatalf("got %q", got)
	}
}

func TestRootIsEmptyWithoutAnyWiki(t *testing.T) {
	if got := Root(t.TempDir()); got != "" {
		t.Fatalf("got %q", got)
	}
}
```
Dazu: Manifest nennt ein nicht existierendes Layout → Rückfall auf `docs/wiki`; Nachbar-Wiki (`<parent>/iam_wiki` für `iam_backend`) wie im bisherigen `FindWikiPath`-Test.

- [ ] **Step 3:** FAIL sehen, dann `root.go`:

```go
package wiki

import (
	"os"
	"path/filepath"

	"github.com/xidus90/loomux/internal/config"
)

// Root answers where a project's wiki bundle is: the manifest's [layout] wiki
// when it names an existing directory, then docs/wiki, then wiki, then a
// neighbour wiki of the same family. "" means the project has none.
//
// ultra-brain's FindWikiPath read [wiki] path and [area] wiki from .brain.toml;
// loomux has one key for this, [layout] wiki (parity list, stage 1a).
func Root(projectRoot string) string {
	if manifest, err := config.ReadManifest(projectRoot); err == nil && manifest.LayoutWiki != "" {
		if dir := filepath.Join(projectRoot, filepath.FromSlash(manifest.LayoutWiki)); isDir(dir) {
			return dir
		}
	}
	for _, candidate := range []string{filepath.Join(projectRoot, "docs", "wiki"), filepath.Join(projectRoot, "wiki")} {
		if isDir(candidate) {
			return candidate
		}
	}
	return neighbour(projectRoot)
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
```
`neighbour(projectRoot string) string` ist Schritt 3 des bisherigen `FindWikiPath` (`gate.go` ab „Neighbour wiki“) wörtlich als eigene Funktion. `FindWikiPath`, `brainToml` und die TOML-Importe in `gate.go` entfallen; `CheckWikiGate` ruft `Root`. Die alten `FindWikiPath`-Tests werden zu `Root`-Tests oder entfallen, wenn sie `[wiki] path`/`[area] wiki` prüfen (Bericht).

- [ ] **Step 4: Failing tests** — `report_test.go`

```go
package wiki

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintReportReturnsOneForAnErrorFinding(t *testing.T) {
	wikiRoot := t.TempDir()
	page := filepath.Join(wikiRoot, "page.md")
	os.WriteFile(page, []byte("no frontmatter at all\n"), 0o644)
	var out bytes.Buffer
	code := LintReport(page, wikiRoot, &out)
	if code != 1 || !strings.Contains(out.String(), "[error:") {
		t.Fatalf("code %d, out %q", code, out.String())
	}
}

func TestLintReportNamesAFileItCannotRead(t *testing.T) {
	var out bytes.Buffer
	if code := LintReport(filepath.Join(t.TempDir(), "missing.md"), t.TempDir(), &out); code != 1 || !strings.Contains(out.String(), "lint error:") {
		t.Fatalf("code %d, out %q", code, out.String())
	}
}

func TestGateReportPassesWithoutAWiki(t *testing.T) {
	var out, errb bytes.Buffer
	if code := GateReport(t.TempDir(), &out, &errb); code != 0 || !strings.HasPrefix(out.String(), "OK: Wiki Gate passed.") {
		t.Fatalf("code %d, out %q", code, out.String())
	}
}
```
Vorher prüfen, welcher Text `check.Error` als Severity druckt (`internal/brain/check/check.go`); `"[error:"` im Test an die tatsächliche Schreibweise anpassen. Dazu ein Test für `LintReport` mit einer gültigen Seite → 0 (eine gültige Seite aus den bestehenden `lint_test.go`-Fixtures übernehmen) und für `GateReport` mit einer Verletzung → 1 und dem Text `Wiki-Gate Violation(s) Detected`.

- [ ] **Step 5:** FAIL sehen, dann `report.go` — die Ausgabe ist die von `SRC_UB/cmd/brain/main.go:567-622`, ohne `os.Exit`:

```go
package wiki

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/xidus90/loomux/internal/brain/check"
)

// LintReport lints one page, writes one line per finding and returns 1 when
// any finding is an error or the page cannot be linted.
func LintReport(target, wikiRoot string, w io.Writer) int {
	findings, err := LintSingleFile(target, wikiRoot)
	if err != nil {
		fmt.Fprintf(w, "lint error: %v\n", err)
		return 1
	}
	code := 0
	for _, f := range findings {
		fmt.Fprintf(w, "  • [%s:%s] %s: %s\n", f.Severity, f.Rule, f.Relative, f.Message)
		if f.Severity == check.Error {
			code = 1
		}
	}
	return code
}

// GateReport runs the wiki gate for a project and reports as `brain wiki-gate` did.
func GateReport(projectRoot string, stdout, stderr io.Writer) int {
	violations := CheckWikiGate(projectRoot)
	if len(violations) == 0 {
		fmt.Fprintln(stdout, "OK: Wiki Gate passed. All bundle links valid and no drift detected.")
		return 0
	}
	fmt.Fprintf(stderr, "\n❌ Wiki-Gate Violation(s) Detected in %s:\n", filepath.Base(projectRoot))
	for _, v := range violations {
		fmt.Fprintf(stderr, "  • [%s] %s\n", v.Name, v.Message)
	}
	fmt.Fprintf(stderr, "\nFound %d violation(s).\n", len(violations))
	return 1
}
```

- [ ] **Step 6: Befehle, test-first** — `internal/cli/wiki_test.go`: `lint` ohne Argument → 2 mit `file path required`; `lint <page>` mit `--root <dir>` nutzt `wiki.Root(dir)`, sonst das Arbeitsverzeichnis, und fällt auf das Verzeichnis der Seite zurück, wenn `Root` "" liefert; Befunde gehen nach stderr. `wiki-gate --root <dir>` → Code von `GateReport`; ohne `--root` das Arbeitsverzeichnis; ein unbekanntes Flag → 2.

```go
package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xidus90/loomux/internal/brain/wiki"
)

var getwd = os.Getwd

func lintCommand(args []string, _ io.Reader, _, stderr io.Writer) int {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "loomux lint: file path required: loomux lint [--root DIR] <file>")
		return 2
	}
	target := fs.Arg(0)
	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux lint: %v\n", err)
		return 1
	}
	wikiRoot := wiki.Root(project)
	if wikiRoot == "" {
		wikiRoot = filepath.Dir(target)
	}
	return wiki.LintReport(target, wikiRoot, stderr)
}

func wikiGateCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("wiki-gate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux wiki-gate: %v\n", err)
		return 1
	}
	return wiki.GateReport(project, stdout, stderr)
}

func projectRoot(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	return getwd()
}
```
Das Flag `--root` für `lint` ist neu (ultra-brain nahm das Arbeitsverzeichnis); Eintrag in die Paritätsliste als Erweiterung, kein Verhaltenswechsel ohne Flag.

- [ ] **Step 7: Coverage heben.** Bekannt unter 100 % in ultra-brain: `gate.go:116 cleanEnv` 93,8 %, `:141 getGitChangedFiles` 46,2 %, `:161 CheckWikiGate` 0 %; `lint.go:35 LintSingleFile` 72,7 %, `:100 LintBundle` 91,7 %; `parse.go:48 ReadPage` 96,2 %. `getGitChangedFiles` und `CheckWikiGate` brauchen ein echtes Git-Repo im `t.TempDir()` (Muster: `SRC_UB/pkg/wiki/gate_test.go:15`). `cleanEnv` ist eine zweite Kopie der Git-Säuberung: durch `gitenv.Environ()` ersetzen, die Funktion entfällt, und der Scan-Test aus Task 3 muss danach grün bleiben.
- [ ] **Step 8:** `sh .githooks/pre-commit` → 0. **Commit** — `Serve wiki lint and the wiki gate from loomux without exiting mid-function`.

---

### Task 9: `cmd/guard` wird `internal/hooks`, mit Befehlen `hook`, `status`, `worktree`

**Files:**
- Move: `SRC_UL/cmd/guard/{guard,post_edit,status,worktree,hook_session_start}.go` und ihre `_test.go` → `internal/hooks/` (Paket `hooks`). **Nicht** übernommen: `main.go`, `main_test.go` (ersetzt durch `internal/cli/hook.go`).
- Create: `internal/cli/hook.go`, Test `internal/cli/hook_test.go`
- Modify: `internal/cli/commands.go` (`"hook": hookCommand,`, `"status": statusCommand,`, `"explain": statusCommand,`, `"doctor": statusCommand,`, `"worktree": worktreeCommand,`)

**Interfaces:**
- Consumes: `hosts.*` (Task 4), `sessions.*`, `gitwork.HeadCommit`, `detect.Detect`, `topo`, `junction`, `mirror.Mirror` (Task 5).
- Produces (umbenannt, sonst unverändert): `hooks.ExitOK=0`, `hooks.ExitInternal=1`, `hooks.ExitDenied=2`; `hooks.PostToolUse(stdin io.Reader, stdout, stderr io.Writer, root string) int` (war `runPostEdit`); `hooks.SessionStart(stdin io.Reader, stdout, stderr io.Writer, root, host string) int` (war `runHookSessionStart`); `hooks.Status(stdout, stderr io.Writer, root string) int` (war `runStatus`); `hooks.WorktreeLink(stdout, stderr io.Writer, root string) int`, `hooks.WorktreeUnlink(stdout, stderr io.Writer, stdin io.Reader, root string) int`, `hooks.WorktreeRemove(stdout, stderr io.Writer, target string) int`. `runGuard` bleibt in diesem Task unverändert und unexportiert; Task 10 ersetzt es.

- [ ] **Step 1:** Umzug nach dem Verfahren, Schritte 1–4: `package main` → `package hooks`; die fünf Funktionen umbenennen (in Code und Tests); Tests, die `cli(...)` aus `main.go` rufen, auf die umbenannten Funktionen umstellen oder — wenn sie nur die Flag-Auswertung prüfen — nach `internal/cli/hook_test.go` verschieben (Step 3).
- [ ] **Step 2: Literale, außer denen der Tasks 10–12.** In Nicht-Kommentar-Code von `internal/hooks`:
  - Meldungspräfixe `ulguard post-edit:` → `loomux hook post-tool-use:`, `ulguard hook session-start:` → `loomux hook session-start:`, `ultraloom-guard` → `loomux`, `usage: ulguard worktree-remove` → `usage: loomux worktree remove`.
  - `status.go`: Banner `UltraLoom Guard & Hook Inspection` → `loomux Hook Inspection`; Zeilen `-> ulguard (…)` → `-> loomux hook pre-tool-use (policy and write barrier)`, `-> ulguard post-edit (…)` → `-> loomux hook post-tool-use (concurrent lanes per file type)`.
  - `status.go:61-65` (`auditSettings`): `hasPreGuard` ⇔ Kommando enthält `loomux` und `pre-tool-use`; `hasPostEdit` ⇔ enthält `loomux` und `post-tool-use`. `knownLegacyHooks` bekommt zwei Einträge vorn: `{"ulguard", "superseded by 'loomux hook pre-tool-use' and 'loomux hook post-tool-use'"}`, `{"brain guard", "merged into 'loomux hook pre-tool-use'"}`.
  - Übrige Treffer von `grep -n '\.ultraloom\|ulguard\|ulinit' internal/hooks/*.go` außerhalb von Kommentaren im Bericht auflisten; Kommentare, die Python-Dateien als Herkunft zitieren, bleiben.
  - `hook_session_start.go`: `runDir`, `waiting` und der Import von `journal` entfallen **hier**, samt ihrer Tests — `journal` zieht in 1a nicht um, ohne diesen Schritt kompiliert das Paket nicht. `SessionStart` schreibt bis Task 12 `hosts.WriteContext(host, stdout, nil)`. Wartende Flow-Läufe meldet session-start erst mit der Flow-Migration (Spec, Sitzungshooks).
- [ ] **Step 3: Failing tests** — `internal/cli/hook_test.go`

```go
package cli

import (
	"strings"
	"testing"
)

func TestHookNeedsAnEvent(t *testing.T) {
	code, _, errOut := run("hook")
	if code != 2 || !strings.Contains(errOut, "usage: loomux hook <event>") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestHookRefusesAnUnknownEvent(t *testing.T) {
	code, _, errOut := run("hook", "stop", "--host", "claude")
	if code != 2 || !strings.Contains(errOut, `unknown event "stop"`) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestHookSessionStartNeedsAHost(t *testing.T) {
	code, _, errOut := run("hook", "session-start", "--root", t.TempDir())
	if code != 1 || !strings.Contains(errOut, "--host is required") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestHookPreToolUseRefusesWithoutAHost(t *testing.T) {
	code, _, _ := run("hook", "pre-tool-use", "--root", t.TempDir())
	if code != 2 {
		t.Fatalf("a write barrier refuses a malformed call, got %d", code)
	}
}

func TestWorktreeRemoveNeedsExactlyOnePath(t *testing.T) {
	for _, args := range [][]string{{"worktree", "remove"}, {"worktree", "remove", ""}, {"worktree", "remove", "a", "b"}} {
		if code, _, _ := run(args...); code != 1 {
			t.Fatalf("%v: code %d", args, code)
		}
	}
}
```
Dazu je ein Test: `status --root <tmp>` → 0 und Banner auf stdout; `worktree` ohne Unterbefehl → 2; `worktree link --bad` → 1; `hook post-tool-use --root <tmp>` mit `{"tool_name":"Edit","tool_input":{"file_path":"x.json"}}` → 0; `hook session-start --host nobody --root <tmp>` → 1; `hook session-start --host claude` ohne `--root` in einem Verzeichnis ohne `.loomux/config.toml` (über `t.Chdir`) → 1 mit `no .loomux/config.toml`.

- [ ] **Step 4:** FAIL sehen, dann `internal/cli/hook.go`:

```go
package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/hooks"
	"github.com/xidus90/loomux/internal/hosts"
)

// Exit codes when a hook call is malformed: a write barrier refuses (2), an
// announcement never blocks (1).
var malformed = map[string]int{
	"pre-tool-use":  hooks.ExitDenied,
	"post-tool-use": hooks.ExitInternal,
	"session-start": hooks.ExitInternal,
}

func hookCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: loomux hook <event> --host <host> [--root <dir>]")
		return 2
	}
	event := args[0]
	failure, known := malformed[event]
	if !known {
		fmt.Fprintf(stderr, "loomux hook: unknown event %q\n", event)
		return 2
	}
	flags := flag.NewFlagSet("loomux hook "+event, flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "path to the project root; found upwards when empty")
	host := flags.String("host", "", "the harness calling: claude, antigravity or codex")
	if err := flags.Parse(args[1:]); err != nil {
		return failure
	}
	if *host == "" {
		fmt.Fprintf(stderr, "loomux hook %s: --host is required: expected claude, antigravity or codex\n", event)
		return failure
	}
	if _, err := hosts.ParseHost(*host); err != nil {
		fmt.Fprintf(stderr, "loomux hook %s: %v\n", event, err)
		return failure
	}
	resolved := *root
	if resolved == "" {
		found, err := hosts.FindRoot(".")
		if err != nil && event != "pre-tool-use" {
			fmt.Fprintf(stderr, "loomux hook %s: %v\n", event, err)
			return failure
		}
		// The barrier is global and needs no project; without a root it
		// judges against the registry alone.
		resolved = found
	}
	switch event {
	case "pre-tool-use":
		return preToolUse(stdin, stdout, stderr, resolved, config.StateDir())
	case "post-tool-use":
		return hooks.PostToolUse(stdin, stdout, stderr, resolved)
	default:
		return hooks.SessionStart(stdin, stdout, stderr, resolved, *host)
	}
}

// preToolUse is replaced by hooks.PreToolUse in Task 10.
var preToolUse = func(stdin io.Reader, stdout, stderr io.Writer, root, stateDir string) int {
	return hooks.ExitDenied
}

func statusCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loomux status", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "path to the project root")
	if err := flags.Parse(args); err != nil {
		return hooks.ExitInternal
	}
	return hooks.Status(stdout, stderr, *root)
}

func worktreeCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: loomux worktree link|unlink [--root <dir>] | remove <worktree path>")
		return 2
	}
	if args[0] == "remove" {
		if len(args) != 2 || args[1] == "" {
			fmt.Fprintln(stderr, "usage: loomux worktree remove <worktree path>")
			return hooks.ExitInternal
		}
		return hooks.WorktreeRemove(stdout, stderr, args[1])
	}
	flags := flag.NewFlagSet("loomux worktree "+args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "path to the project root")
	if err := flags.Parse(args[1:]); err != nil {
		return hooks.ExitInternal
	}
	switch args[0] {
	case "link":
		return hooks.WorktreeLink(stdout, stderr, *root)
	case "unlink":
		return hooks.WorktreeUnlink(stdout, stderr, stdin, *root)
	}
	fmt.Fprintf(stderr, "loomux worktree: unknown subcommand %q\n", args[0])
	return 2
}
```
`TestHookPreToolUseRefusesWithoutAHost` besteht mit dem Platzhalter nicht über dessen Rückgabe, sondern über den `--host`-Arm; der Platzhalter selbst bekommt keinen eigenen Test und fällt in Task 10.

- [ ] **Step 5:** `go test ./internal/hooks/ ./internal/cli/` → PASS.
- [ ] **Step 6: Coverage heben** — bekannt aus ultraloom: `post_edit.go:64 runLane` 85,7 %, `:165 defaultRunnerFor` 0 %, `:258 reportSkipped` 90 %, `:301 getCommandsForStacks` 98,7 %, `:445 relativeToArea` 85,7 %, `:478 isWikiPath` 93,3 %; `status.go:83 runStatus` 99,3 %; `worktree.go:280 unlink` 91,7 %, `:365 sweep` 94,4 %; `sessions/state.go:74 WriteState` 93,8 % (falls nicht schon in Task 4). `defaultRunnerFor` wird über `PostToolUse` mit einer echten Lane erreicht, deren Werkzeug sicher existiert (`go version` auf einer `.go`-Datei im Stack `go`). `reportSkipped`s `json.Marshal`-Fehlerarm ist für eine `map[string]any` aus Strings unerreichbar: `//coverage:exempt json.Marshal cannot fail on a map of strings`. Der Test `post_edit_test.go:287` (`cmd /c exit 42`) läuft nur unter Windows — mit `if runtime.GOOS != "windows" { t.Skip(...) }` absichern.
- [ ] **Step 7:** `sh .githooks/pre-commit` → 0. **Commit** — `Move the hook commands into internal/hooks behind loomux hook, status and worktree`.

---

### Task 10: Ein Wächter — `loomux hook pre-tool-use`

**Files:**
- Create: `internal/hooks/pretool.go`, Test `internal/hooks/pretool_test.go`
- Modify: `internal/hooks/guard.go` (Policy-Teil), `internal/hooks/guard_test.go`
- Modify: `internal/cli/hook.go` (Platzhalter `preToolUse` → `hooks.PreToolUse`)

**Interfaces:**
- Consumes: `config.ReadPolicy(root) (config.Policy, error)` (Task 6); `guard.Run`, `guard.Refuse`, `guard.Call`, `guard.WriteTargets`, `guard.IsWritingTool` (Task 7).
- Produces: `hooks.PreToolUse(stdin io.Reader, stdout, stderr io.Writer, root, stateDir string) int` — nur 0 oder 2.

- [ ] **Step 1: Failing tests** — `internal/hooks/pretool_test.go`

```go
package hooks

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// world builds a project with a registry that makes the whole project writable.
func world(t *testing.T, config string) (root, state string) {
	t.Helper()
	root = t.TempDir()
	state = t.TempDir()
	if config != "" {
		os.MkdirAll(filepath.Join(root, ".loomux"), 0o755)
		os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte(config), 0o644)
	}
	registry := "[[area]]\nscope = \"project/x\"\npath = " + quote(root) + "\nwiki = " + quote(filepath.Join(root, "docs", "wiki")) + "\nworkspace = true\n"
	os.WriteFile(filepath.Join(state, "registry.toml"), []byte(registry), 0o644)
	return root, state
}

func quote(path string) string { return `"` + filepath.ToSlash(path) + `"` }

func payload(tool, key, value string) string {
	return `{"tool_name":"` + tool + `","tool_input":{"` + key + `":` + quote(value) + `}}`
}

func call(t *testing.T, root, state, input string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := PreToolUse(strings.NewReader(input), &out, &errb, root, state)
	return code, out.String(), errb.String()
}

func TestAWriteInsideTheProjectIsAllowed(t *testing.T) {
	root, state := world(t, "")
	if code, _, errOut := call(t, root, state, payload("Write", "file_path", filepath.Join(root, "main.go"))); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestABuiltinPolicyRuleRefusesFirst(t *testing.T) {
	root, state := world(t, "")
	code, out, errOut := call(t, root, state, payload("Write", "file_path", filepath.Join(root, ".env")))
	if code != 2 || !strings.Contains(errOut, "secrets are not written by an agent") || !strings.Contains(out, `"permissionDecision": "deny"`) {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestAConfiguredCommandRuleRefusesABashCall(t *testing.T) {
	root, state := world(t, "[[policy.commands.rules]]\nregex = '(^|\\s)pip\\s+install([^\\w-]|$)'\nreason = \"uv, never pip\"\n")
	code, _, errOut := call(t, root, state, `{"tool_name":"Bash","tool_input":{"command":"pip install requests"}}`)
	if code != 2 || !strings.Contains(errOut, "uv, never pip") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestAnUnreadablePolicyRefusesAndNamesTheFile(t *testing.T) {
	root, state := world(t, "[[policy.paths.rules]\n")
	code, _, errOut := call(t, root, state, payload("Write", "file_path", filepath.Join(root, "main.go")))
	if code != 2 || !strings.Contains(errOut, "config.toml") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestTheConfigItselfIsNeverWritable(t *testing.T) {
	root, state := world(t, "[area]\nscope = \"project/x\"\n")
	code, _, _ := call(t, root, state, payload("Edit", "file_path", filepath.Join(root, ".loomux", "config.toml")))
	if code != 2 {
		t.Fatalf("code %d", code)
	}
}

func TestAWriteOutsideEveryAreaIsRefusedByTheBarrier(t *testing.T) {
	root, state := world(t, "")
	code, _, _ := call(t, root, state, payload("Write", "file_path", filepath.Join(t.TempDir(), "elsewhere.md")))
	if code != 2 {
		t.Fatalf("code %d", code)
	}
}

func TestAnUnreadablePayloadIsRefusedNotPassed(t *testing.T) {
	root, state := world(t, "")
	if code, _, _ := call(t, root, state, "not json"); code != 2 {
		t.Fatalf("ulguard answered 1 here; loomux refuses, got %d", code)
	}
}

func TestAnAntigravityWriteIsJudgedByPolicyToo(t *testing.T) {
	root, state := world(t, "")
	input := `{"toolCall":{"name":"write_to_file","args":{"TargetFile":` + quote(filepath.Join(root, ".env")) + `}}}`
	code, _, errOut := call(t, root, state, input)
	if code != 2 || !strings.Contains(errOut, "secrets") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestAPanicInThePolicyStepRefuses(t *testing.T) {
	root, state := world(t, "")
	readPolicy = func(string) (config.Policy, error) { panic("boom") }
	defer func() { readPolicy = config.ReadPolicy }()
	code, _, errOut := call(t, root, state, payload("Write", "file_path", filepath.Join(root, "main.go")))
	if code != 2 || !strings.Contains(errOut, "boom") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}
```
Import `github.com/xidus90/loomux/internal/config` im Testkopf ergänzen. Liest die Registry der Schranke `path` mit Vorwärtsschrägstrichen anders als erwartet, gilt `SRC_UB/pkg/guard`s eigener Test als Vorlage für die Registry-Zeilen.

- [ ] **Step 2:** `go test ./internal/hooks/ -run 'Pre|Policy|Config|Barrier|Payload|Antigravity|Panic|Builtin|Configured|Unreadable'` → FAIL `undefined: PreToolUse`.
- [ ] **Step 3: Policy-Teil umbauen** — in `internal/hooks/guard.go`: `PolicyFile`, `PathRule`, `CommandRule`, `loadPolicy` und `runGuard` entfallen samt ihrer Tests (`HookPayload` bleibt: `post_edit.go` dekodiert die Claude-Nutzlast hinein); die eingebauten Regeln werden `config.PathRule`/`config.CommandRule`-Werte (Regexe über `regexp.MustCompile` beim ersten Gebrauch in einer `sync.OnceValue`, nicht in einer Paketvariable, wegen der Startzeit-Regel); `checkTool` bekommt diese Form:

```go
var builtinCommands = sync.OnceValue(func() []config.CommandRule {
	return []config.CommandRule{{
		Regex:  regexp.MustCompile(`(^|\s)git\s+push(\s|$)`),
		Source: `(^|\s)git\s+push(\s|$)`,
		Reason: "Whether commits reach the remote is a human's decision.",
	}}
})

// commandTools are the tools whose "command" argument is a shell line.
var commandTools = map[string]bool{"Bash": true, "PowerShell": true}

func checkTool(root, tool string, input map[string]any, policy config.Policy) []string {
	var reasons []string
	if guard.IsWritingTool(tool) {
		for _, target := range guard.WriteTargets(input) {
			rel := relativePath(target, root)
			for _, rule := range append(builtinPathRules, policy.Paths...) {
				for _, glob := range rule.Match {
					if matchGlob(glob, rel) {
						reasons = append(reasons, rule.Reason)
						break
					}
				}
			}
		}
	}
	if commandTools[tool] {
		if line, ok := input["command"].(string); ok {
			for _, rule := range append(builtinCommands(), policy.Commands...) {
				if rule.Regex.MatchString(line) {
					reasons = append(reasons, rule.Reason)
				}
			}
		}
	}
	return reasons
}
```
`builtinPathRules` wird `[]config.PathRule` mit `Match: []string{…}` je Eintrag (Inhalte wie `SRC_UL/cmd/guard/guard.go:49-69`, `.ultraloom/hooks/**` → `.loomux/state/hooks/**`). Die bestehenden `checkTool`-Tests auf die neue Signatur umstellen.

- [ ] **Step 4: Implementieren** — `internal/hooks/pretool.go`

```go
package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/config"
)

var readPolicy = config.ReadPolicy

// PreToolUse answers one PreToolUse hook with 0 or 2, never 1: the project's
// policy first, then the global write barrier. Everything that goes wrong on
// the way refuses, because a host reads 1 as "carry on".
func PreToolUse(stdin io.Reader, stdout, stderr io.Writer, root, stateDir string) (code int) {
	defer func() {
		if broke := recover(); broke != nil {
			code = guard.Refuse(stdout, stderr, fmt.Sprintf("the loomux guard broke down, so it refuses: %v", broke))
		}
	}()
	data, err := io.ReadAll(stdin)
	if err != nil {
		return guard.Refuse(stdout, stderr, fmt.Sprintf("loomux cannot read the hook payload, so it refuses: %v", err))
	}
	reasons, err := policyReasons(data, root)
	if err != nil {
		return guard.Refuse(stdout, stderr, fmt.Sprintf("loomux cannot read its policy, so it refuses: %v", err))
	}
	if len(reasons) > 0 {
		return guard.Refuse(stdout, stderr, "loomux policy refused this tool call:\n  - "+strings.Join(reasons, "\n  - "))
	}
	return guard.Run(bytes.NewReader(data), stdout, stderr, stateDir)
}

// policyReasons judges the call against the policy. A payload it cannot
// decode yields no reasons: refusing it is the barrier's job, with the
// barrier's wording, one step later.
func policyReasons(data []byte, root string) ([]string, error) {
	var object map[string]any
	if json.Unmarshal(data, &object) != nil {
		return nil, nil
	}
	tool, input := guard.Call(object)
	if tool == "" {
		return nil, nil
	}
	policy, err := readPolicy(root)
	if err != nil {
		return nil, err
	}
	return checkTool(root, tool, input, policy), nil
}
```
In `internal/cli/hook.go` den Platzhalter löschen und `case "pre-tool-use": return hooks.PreToolUse(stdin, stdout, stderr, resolved, config.StateDir())` schreiben.

- [ ] **Step 5:** `go test ./internal/hooks/ ./internal/cli/` → PASS; Coverage 100 % (`io.ReadAll`-Fehlerarm über einen `iotest.ErrReader`).
- [ ] **Step 6:** `sh .githooks/pre-commit` → 0. **Commit** — `Answer PreToolUse with one guard: policy first, then the global write barrier`.

---

### Task 11: `loomux hook post-tool-use` — Wiki-Lane im Prozess, Wiki aus der Konfiguration

**Files:**
- Modify: `internal/hooks/post_edit.go`, `internal/hooks/post_edit_test.go`, `internal/hooks/status.go`

**Interfaces:**
- Consumes: `wiki.Root(projectRoot) string`, `wiki.LintReport(target, wikiRoot string, w io.Writer) int` (Task 8).
- Produces: `hooks.PostToolUse` unverändert in der Signatur.

- [ ] **Step 1: Failing tests** — in `post_edit_test.go`:

```go
func TestTheWikiLaneRunsInProcess(t *testing.T) {
	root := t.TempDir()
	wikiDir := filepath.Join(root, "docs", "wiki")
	os.MkdirAll(wikiDir, 0o755)
	page := filepath.Join(wikiDir, "page.md")
	os.WriteFile(page, []byte("no frontmatter at all\n"), 0o644)
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

func TestTheWikiDirectoryComesFromTheManifest(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".loomux"), 0o755)
	os.MkdirAll(filepath.Join(root, "notes"), 0o755)
	os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[area]\nscope = \"project/x\"\n[layout]\nwiki = \"notes\"\n"), 0o644)
	if got := wikiDirFor(root); got != "notes" {
		t.Fatalf("got %q", got)
	}
}
```
Die zwei `resolveWikiDir`-Tests (answers.toml) entfallen; im Bericht nennen.

- [ ] **Step 2:** FAIL sehen.
- [ ] **Step 3: Implementieren.**
  - `command` bekommt ein Feld `run func() (string, error)`; ist es gesetzt, ruft die Goroutine in `runPostEditWithContext` `c.run()` statt `runner(...)`:

```go
go func(c command) {
	defer wg.Done()
	var out string
	var err error
	if c.run != nil {
		out, err = c.run()
	} else {
		out, err = runner(filepath.Join(root, c.dir), c.text)
	}
	// … unchanged failure handling …
}(cmd)
```
  - `getCommandsForStacks` bekommt die Parameter `root, wikiDir string` (alle Aufrufer anpassen, auch `status.go:242 unavailableLanes`). Der Wiki-Arm (`post_edit.go:427-432` in ultraloom) wird:

```go
if shouldRun("wiki") && hasTarget && targetPath != "" {
	target := targetPath
	cmds = append(cmds, command{
		text: "loomux lint " + target,
		run: func() (string, error) {
			var report strings.Builder
			wikiRoot := filepath.Join(root, filepath.FromSlash(wikiDir))
			if code := wiki.LintReport(filepath.Join(root, target), wikiRoot, &report); code != 0 {
				return report.String(), fmt.Errorf("wiki lint found errors in %s", target)
			}
			return "", nil
		},
	})
}
```
  - `laneAvailable` wird für eine Lane mit `run != nil` nicht gefragt (kein Werkzeug auf dem PATH nötig); `unavailableLanes` in `status.go` überspringt solche Lanes.
  - `resolveWikiDir` wird `wikiDirFor(root string) string`: `wiki.Root(root)` relativ zu `root` mit Vorwärtsschrägstrichen; ist das Ergebnis leer, `detect.Detect(os.DirFS(root)).WikiPath`; sonst `"wiki/"`. `PostToolUse` und `Status` rufen `wikiDirFor`.
- [ ] **Step 4:** `go test ./internal/hooks/` → PASS, Coverage 100 %.
- [ ] **Step 5:** `sh .githooks/pre-commit` → 0. **Commit** — `Lint wiki pages inside the post-edit hook instead of through uv and a second binary`.

---

### Task 12: `loomux hook session-start` — Basis festhalten, veraltetes Pilot-Binary melden

**Files:**
- Modify: `internal/hooks/hook_session_start.go`, `internal/hooks/hook_session_start_test.go`

**Interfaces:**
- Produces: `hooks.SessionStart` unverändert in der Signatur; intern `staleBinary(root string) []string`, `newestSource(root string) (time.Time, string, error)`.

Die Vergleichsgröße ist **nicht** HEAD: das Pre-Commit-Tor baut `bin/loomux.exe`, und Git schreibt den Commit erst danach — die Commit-Zeit läge immer nach dem Binary, und die Warnung stünde nach jedem Commit. Verglichen wird mit der jüngsten Änderungszeit der Quellen, die das Binary bestimmen: `go.mod`, `go.sum`, jede `.go`-Datei unter `cmd/` und `internal/`.

- [ ] **Step 1: Failing tests** — `hook_session_start_test.go`:

```go
func pilot(t *testing.T, binaryAge, sourceAge time.Duration) (root, binary string) {
	t.Helper()
	root = t.TempDir()
	binary = filepath.Join(root, "bin", "loomux.exe")
	source := filepath.Join(root, "internal", "cli", "cli.go")
	for path, age := range map[string]time.Duration{binary: binaryAge, source: sourceAge} {
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, []byte("x"), 0o644)
		stamp := time.Now().Add(-age)
		os.Chtimes(path, stamp, stamp)
	}
	return root, binary
}

func TestSessionStartWarnsWhenASourceIsNewerThanThePilotBinary(t *testing.T) {
	root, binary := pilot(t, 2*time.Hour, time.Minute)
	executable = func() (string, error) { return binary, nil }
	defer func() { executable = os.Executable }()
	lines := staleBinary(root)
	if len(lines) != 1 || !strings.Contains(lines[0], "internal/cli/cli.go") {
		t.Fatalf("%v", lines)
	}
}

func TestSessionStartIsQuietWhenTheBinaryIsNewest(t *testing.T) {
	root, binary := pilot(t, time.Minute, 2*time.Hour)
	executable = func() (string, error) { return binary, nil }
	defer func() { executable = os.Executable }()
	if lines := staleBinary(root); len(lines) != 0 {
		t.Fatalf("%v", lines)
	}
}

func TestSessionStartSaysNothingAboutABinaryOutsideTheProject(t *testing.T) {
	root, _ := pilot(t, 2*time.Hour, time.Minute)
	executable = func() (string, error) { return filepath.Join(t.TempDir(), "loomux.exe"), nil }
	defer func() { executable = os.Executable }()
	if lines := staleBinary(root); len(lines) != 0 {
		t.Fatalf("%v", lines)
	}
}
```
Plus: `executable` liefert Fehler → keine Zeile; `absPath` liefert Fehler → keine Zeile; `os.Stat` auf das Binary scheitert → keine Zeile; ein Verzeichnis unter `internal/`, das sich nicht lesen lässt (`testlock` oder eine injizierte `walk`-Funktion) → keine Zeile; `.go`-Dateien außerhalb von `cmd/` und `internal/` (etwa `testdata/x.go`) zählen nicht.

- [ ] **Step 2:** FAIL sehen, dann in `hook_session_start.go` (`runDir`/`waiting` sind seit Task 9 fort; `SessionStart` schreibt jetzt `lines := staleBinary(root)`):

```go
var (
	executable = os.Executable
	absPath    = filepath.Abs
)

// staleBinary names the running binary when it lives inside the project and
// a source that decides its behaviour changed after it was built: a pilot
// hook then judges with yesterday's rules.
func staleBinary(root string) []string {
	path, err := executable()
	if err != nil {
		return nil
	}
	absRoot, err := absPath(root)
	if err != nil {
		return nil
	}
	rel, err := filepath.Rel(absRoot, path)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	newest, name, err := newestSource(absRoot)
	if err != nil || !info.ModTime().Before(newest) {
		return nil
	}
	return []string{fmt.Sprintf(
		"loomux binary %s is older than %s; rebuild with: go build -o bin/loomux.new.exe ./cmd/loomux, then go run ./cmd/loomux dev swap-binary --dir bin",
		filepath.ToSlash(rel), name)}
}

// newestSource is the latest modification among go.mod, go.sum and the .go
// files under cmd/ and internal/, with that file's slash-separated path.
func newestSource(root string) (time.Time, string, error) {
	var newest time.Time
	var name string
	consider := func(path string, info fs.FileInfo) {
		if info.ModTime().After(newest) {
			newest = info.ModTime()
			rel, _ := filepath.Rel(root, path)
			name = filepath.ToSlash(rel)
		}
	}
	for _, file := range []string{"go.mod", "go.sum"} {
		if info, err := os.Stat(filepath.Join(root, file)); err == nil {
			consider(filepath.Join(root, file), info)
		}
	}
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, err error) error {
			if errors.Is(err, fs.ErrNotExist) && path == filepath.Join(root, dir) {
				return filepath.SkipDir
			}
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			consider(path, info)
			return nil
		})
		if err != nil {
			return time.Time{}, "", err
		}
	}
	return newest, name, nil
}
```
`entry.Info()`-Fehler ist ohne gleichzeitiges Löschen nicht auslösbar: `//coverage:exempt` ist dafür **nicht** zulässig, weil `newestSource` sonst getestet ist; stattdessen den Arm über eine Datei testen, die zwischen `WalkDir` und `Info()` verschwindet, oder `WalkDir` als Variable `walkDir` injizieren.

- [ ] **Step 3:** `go test ./internal/hooks/` → PASS, Coverage 100 %.
- [ ] **Step 4:** `sh .githooks/pre-commit` → 0. **Commit** — `Warn at session start when a source is newer than the pilot binary`.

---

### Task 13: Fallkorpus im Prozess — Runner, Rekorder, Übersetzer

**Files:**
- Move: `SRC_UB/pkg/cases` → `internal/cases` (ohne `suite_test.go`; die Suite ersetzt Task 14)
- Modify: `internal/cases/case.go`, `internal/cases/runner.go` und Tests
- Create: `internal/dev/recordcase/recordcase.go`, Test
- Create: `internal/dev/importcases/importcases.go`, Test
- Modify: `internal/cli/dev.go` (`"record-case"`, `"import-cases"` in `devCommands`)

**Interfaces:**
- Produces:
  - `cases.WorldToken = "{{WORLD}}"`
  - `cases.Case` wie bisher plus `Compare string` (`"data"` oder `"message"`, aus der Datei `compare`, Vorgabe `"data"`)
  - `type cases.RunFunc func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int`
  - `cases.RunCase(c *Case, run RunFunc) (*RunOutcome, error)` — ersetzt `RunCase(c, exePath)`
  - `cases.StageWorld(src, dst string) error` — ersetzt `{{WORLD}}` in Dateiinhalten durch `filepath.ToSlash(dst)`
  - `cases.Normalize(data []byte, dir string) []byte` — ersetzt `filepath.ToSlash(dir)`, `dir` und JSON-escaptes `dir` (`\` → `\\`) durch `{{WORLD}}`
  - `cases.CompareTrees(actualDir, expectedDir string) ([]string, error)` — normalisiert die Inhalte von `actualDir` vor dem Vergleich
  - `recordcase.Spec{Exe, Cmd, World, Stdin, Out, Notes, Compare string}`; `recordcase.Record(s Spec) error`
  - `importcases.Mapping{Commands []importcases.Rule}`, `importcases.Rule{From, To string}`; `importcases.Import(from, to string, m Mapping) error`; `importcases.TranslateWorld(dir string) error`

- [ ] **Step 1:** Umzug, Verfahren Schritte 1–4; `suite_test.go` nicht kopieren.
- [ ] **Step 2: Failing tests** — `runner_test.go` ergänzen:

```go
func TestRunCaseSubstitutesTheWorldAndComparesInProcess(t *testing.T) {
	dir := t.TempDir()
	caseDir := filepath.Join(dir, "verb", "one")
	os.MkdirAll(filepath.Join(caseDir, "world"), 0o755)
	os.WriteFile(filepath.Join(caseDir, "world", "a.txt"), []byte("at {{WORLD}}\n"), 0o644)
	os.WriteFile(filepath.Join(caseDir, "cmd"), []byte("loomux echo {{WORLD}}/a.txt\n"), 0o644)
	os.WriteFile(filepath.Join(caseDir, "exit"), []byte("0\n"), 0o644)
	os.WriteFile(filepath.Join(caseDir, "stdout"), []byte("{{WORLD}}/a.txt\n"), 0o644)
	c, err := LoadCase(caseDir)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := RunCase(c, func(args []string, world string, _ io.Reader, stdout, _ io.Writer) int {
		data, _ := os.ReadFile(filepath.Join(world, "a.txt"))
		if string(data) != "at "+filepath.ToSlash(world)+"\n" {
			t.Errorf("world not substituted: %q", data)
		}
		fmt.Fprintln(stdout, args[1])
		return 0
	})
	if err != nil || !outcome.Passed {
		t.Fatalf("%v %+v", err, outcome)
	}
}

func TestAMessageCaseIgnoresStdout(t *testing.T) {
	c := &Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "loomux x", ExitCode: 2, Stdout: []byte("old words"), Compare: "message"}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)
	outcome, err := RunCase(c, func([]string, string, io.Reader, io.Writer, io.Writer) int { return 2 })
	if err != nil || !outcome.Passed {
		t.Fatalf("%v %+v", err, outcome)
	}
}

func TestACommandThatIsNotLoomuxIsRefused(t *testing.T) {
	c := &Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "brain guard"}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)
	if _, err := RunCase(c, nil); err == nil {
		t.Fatal("want error")
	}
}
```
Dazu: `LoadCase` liest `compare` (fehlend → `data`; unbekannter Wert → Fehler); `Normalize` für alle drei Schreibweisen; `CompareTrees` meldet eine Datei, die nach Normalisierung abweicht.

- [ ] **Step 3:** FAIL sehen, dann `runner.go` umbauen: `RunCase(c, run)` stagt die Welt in `os.MkdirTemp`, ersetzt `{{WORLD}}` in den Befehls-Tokens und in `c.Stdin`, verlangt `tokens[0] == "loomux"`, ruft `run(tokens[1:], tmp, stdin, &stdout, io.Discard)`, normalisiert stdout mit `Normalize` und vergleicht: Exit immer; stdout nur bei `Compare == "data"`; `world_after` immer, wenn vorhanden. `exec`, `BRAIN_STATE_DIR` und `PYTHONUNBUFFERED` entfallen aus dem Runner.
- [ ] **Step 4: Rekorder, test-first** — `recordcase_test.go` baut mit `go build` nichts, sondern nimmt als `Exe` das Test-Binary selbst (Muster `os.Args[0]` mit `-test.run=TestHelperProcess` und einer Umgebungsvariable), das `{{WORLD}}`-haltige Argumente zurückschreibt, und prüft: `cmd` enthält `{{WORLD}}` statt des Temp-Pfads; `stdout` normalisiert; `world/` ist die Eingabewelt, `world_after/` der normalisierte Zustand danach; `exit` stimmt; `notes.md` trägt `Notes`; `compare` nur, wenn gesetzt.

```go
package recordcase

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/cases"
)

// Spec is one recording: which old binary, which command, which world.
type Spec struct {
	Exe     string // absolute path of the old binary; replaces the command's first token
	Cmd     string // command line with {{WORLD}}, first token the old name ("ulguard", "brain", "ulinit")
	World   string // directory to stage
	Stdin   string // file with the payload, may contain {{WORLD}}; "" for none
	Out     string // case directory to create
	Notes   string // text for notes.md: tag, binary, what the case shows
	Compare string // "" (data) or "message"
}

// Record runs the old binary in a staged copy of the world and writes the case.
func Record(s Spec) error {
	tmp, err := os.MkdirTemp("", "record-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := cases.StageWorld(s.World, tmp); err != nil {
		return err
	}
	world := filepath.ToSlash(tmp)
	tokens, err := cases.SplitCommand(s.Cmd)
	if err != nil {
		return err
	}
	for i := range tokens {
		tokens[i] = strings.ReplaceAll(tokens[i], cases.WorldToken, world)
	}
	var stdin []byte
	if s.Stdin != "" {
		if stdin, err = os.ReadFile(s.Stdin); err != nil {
			return err
		}
	}
	cmd := exec.Command(s.Exe, tokens[1:]...)
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "BRAIN_STATE_DIR="+tmp)
	cmd.Stdin = bytes.NewReader(bytes.ReplaceAll(stdin, []byte(cases.WorldToken), []byte(world)))
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	exit := 0
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return fmt.Errorf("running %s: %w", s.Exe, err)
		}
		exit = exitErr.ExitCode()
	}
	files := map[string][]byte{
		"cmd":      []byte(s.Cmd + "\n"),
		"exit":     []byte(fmt.Sprintf("%d\n", exit)),
		"stdout":   cases.Normalize(stdout.Bytes(), tmp),
		"notes.md": []byte(s.Notes + "\n"),
	}
	if stdin != nil {
		files["stdin"] = stdin
	}
	if s.Compare != "" {
		files["compare"] = []byte(s.Compare + "\n")
	}
	if err := os.MkdirAll(s.Out, 0o755); err != nil {
		return err
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(s.Out, name), data, 0o644); err != nil {
			return err
		}
	}
	if err := copyTree(s.World, filepath.Join(s.Out, "world"), nil); err != nil {
		return err
	}
	return copyTree(tmp, filepath.Join(s.Out, "world_after"), func(data []byte) []byte { return cases.Normalize(data, tmp) })
}
```
`copyTree(src, dst string, transform func([]byte) []byte) error` im selben Paket: rekursiv kopieren, Inhalte optional transformieren. `SplitCommand` ist in `internal/cases` bereits exportiert.

- [ ] **Step 5: Übersetzer, test-first** — `importcases_test.go` prüft an einer Welt mit `.brain.toml` (`[area]`, `[layout]`, `[check.lanes]`), `.ultraloom/policy.toml`, `.ultraloom/config.toml` (`[verify]`, `[worktree]`) und `.ultraloom/answers.toml` (`bundle = "docs/wiki/"`):
  - danach existiert nur `.loomux/config.toml`; sie enthält `area`, `layout`, `policy`, `worktree`; **nicht** `check`, `verify`;
  - `layout.wiki` stammt aus dem Manifest; nur ohne Manifest-Layout aus `bundle`, ohne Schrägstrich am Ende;
  - dieselbe Übersetzung unter `areas/<name>/`;
  - `cmd` und `stdin` eines Falls: `ulguard --root {{WORLD}}` → laut Regel; Pfad-Literale `.brain.toml`, `.ultra-brain/config.toml`, `.ultraloom/policy.toml` → `.loomux/config.toml`;
  - ein Fall ohne passende Regel → Fehler mit dem Fallnamen.

```go
package importcases

// TranslateWorld rewrites the configuration files of the old tools in dir
// (and in every dir/areas/<name>) into one .loomux/config.toml.
func TranslateWorld(dir string) error
```
Umsetzung: je Verzeichnis die Dateien mit `toml.DecodeFile` in `map[string]any` lesen; Manifest = erstes vorhandenes von `.ultra-brain/config.toml`, `.brain.toml`; daraus `check` löschen; `policy` aus `.ultraloom/policy.toml` übernehmen; `worktree` aus `.ultraloom/config.toml`; `bundle` aus `answers.toml` nur als `layout.wiki`, wenn das Manifest keines hat; ist die Ergebnis-Map leer, keine Datei schreiben; sonst mit `toml.NewEncoder` nach `.loomux/config.toml` schreiben und die alten Dateien löschen (`.ultraloom/` und `.ultra-brain/` entfallen, wenn leer). `Import(from, to, m)`: für jeden Fall aus `cases.DiscoverCases(from, "")` den Fallordner nach `to/<verb>/<name>` kopieren, erste passende Regel (`strings.HasPrefix(cmd, rule.From)`) anwenden, Pfad-Literale in `cmd` und `stdin` ersetzen, `TranslateWorld` auf `world` und `world_after` anwenden.

- [ ] **Step 6: dev-Befehle** — `record-case` mit Flags `--exe --cmd --world --stdin --out --notes --compare`; `import-cases` mit `--map <toml> --from <dir> --to <dir>`, die Map-Datei in dieser Form:

```toml
[[command]]
from = "ulguard --root {{WORLD}}"
to   = "loomux hook pre-tool-use --host claude --root {{WORLD}}"
```
Tests je Befehl: fehlende Pflichtflags → 2; Fehler aus `Record`/`Import` → 1; Erfolg → 0.
- [ ] **Step 7:** `go test ./internal/cases/ ./internal/dev/... ./internal/cli/` → PASS, Coverage 100 %.
- [ ] **Step 8:** `sh .githooks/pre-commit` → 0. **Commit** — `Run recorded cases in-process and record them from the old binaries`.

---

### Task 14: Fälle der Stufe 1a aufzeichnen, übersetzen, grün machen

**Files:**
- Create: `testdata/cases/1a-source/**` (Aufzeichnungen), `testdata/cases/1a/**` (übersetzt), `testdata/cases/1a-map.toml`, `testdata/cases/1a-worlds/**` (Eingabewelten), `testdata/cases/1a-payloads/**`
- Create: `internal/cli/cases_test.go`
- Create: `docs/.superpowers/parity/stufe-1a.md`

- [ ] **Step 1: Alte Binaries aus den Tags bauen** (in `$TEMP/loomux-old`, nicht im Repo):

```bash
go -C "/c/Users/micro/Documents/#GIT/loomux-src/ul" build -o "$TEMP/loomux-old/ulguard.exe" ./cmd/guard
```
```bash
go -C "/c/Users/micro/Documents/#GIT/loomux-src/ul" build -o "$TEMP/loomux-old/ulinit.exe" ./cmd/init
```
```bash
go -C "/c/Users/micro/Documents/#GIT/loomux-src/ub" build -o "$TEMP/loomux-old/brain.exe" ./cmd/brain
```

- [ ] **Step 2: Welten und Nutzlasten anlegen.** Jede Welt ist ein Verzeichnis unter `testdata/cases/1a-worlds/<name>/`; Pfade in Dateien schreiben `{{WORLD}}`.
  - `project-writable/`: `registry.toml` = ein `[[area]]` mit `scope = "project/w"`, `path = "{{WORLD}}"`, `wiki = "{{WORLD}}/docs/wiki"`, `workspace = true`; `.brain.toml` = `[area] scope = "project/w"`, `[layout] wiki = "docs/wiki"`; `.ultraloom/policy.toml` = eine Pfadregel `match = "generated/*"`, `reason = "generated files are rebuilt, not edited"`; `docs/wiki/index.md` = eine gültige Seite aus `SRC_UB/pkg/wiki/lint_test.go`; `docs/wiki/broken.md` = `no frontmatter at all`.
  - `guard-allow-writable/`, `guard-deny-readonly/`: wörtliche Kopien der `world/` aus `SRC_UB/bench/cases/guard/<name>/`, plus deren `stdin` nach `1a-payloads/`.
  - `plain/`: leeres Verzeichnis mit einer Datei `msg-en.txt` (`Add the first loomux command`) und `msg-de.txt` (`Füge den ersten Befehl hinzu und prüfe die Änderung`).
  - Nutzlasten (`1a-payloads/*.json`): `write-env.json` (`Write`, `{{WORLD}}/.env`), `write-generated.json` (`Write`, `{{WORLD}}/generated/x.go`), `write-main.json` (`Write`, `{{WORLD}}/main.go`), `write-manifest.json` (`Edit`, `{{WORLD}}/.brain.toml`), `bash-push.json` (`Bash`, `git push origin master`), `bash-test.json` (`Bash`, `go test ./...`), `garbage.txt` (`not json`), `agy-outside.json` (`{"toolCall":{"name":"write_to_file","args":{"TargetFile":"{{WORLD}}/../outside.md"}}}`), `edit-json.json` (`Edit`, `{{WORLD}}/data.json`), `edit-readme.json` (`Edit`, `{{WORLD}}/README.md`), `edit-broken-page.json` (`Edit`, `{{WORLD}}/docs/wiki/broken.md`), `session.json` (`{"hook_event_name":"SessionStart"}`).

- [ ] **Step 3: Aufzeichnen.** Je Fall ein Aufruf, `--notes` nennt Tag `loomux-1a-source`, Binary und was der Fall zeigt. Befehlsvorlage:

```bash
go run ./cmd/loomux dev record-case --exe "$TEMP/loomux-old/ulguard.exe" --cmd "ulguard --root {{WORLD}}" --world testdata/cases/1a-worlds/project-writable --stdin testdata/cases/1a-payloads/write-env.json --out testdata/cases/1a-source/hook-pre-tool-use/policy-refuses-env --compare message --notes "loomux-1a-source, ulguard: a builtin path rule refuses .env"
```

| Verb / Fall | Exe | Cmd | Welt | Stdin | Compare |
|---|---|---|---|---|---|
| hook-pre-tool-use/policy-refuses-env | ulguard | `ulguard --root {{WORLD}}` | project-writable | write-env | message |
| hook-pre-tool-use/policy-refuses-configured-path | ulguard | wie oben | project-writable | write-generated | message |
| hook-pre-tool-use/policy-refuses-git-push | ulguard | wie oben | project-writable | bash-push | message |
| hook-pre-tool-use/policy-allows-bash | ulguard | wie oben | project-writable | bash-test | message |
| hook-pre-tool-use/unreadable-payload | ulguard | wie oben | project-writable | garbage | message |
| hook-pre-tool-use/barrier-allows-writable | brain | `brain guard` | guard-allow-writable | deren stdin | message |
| hook-pre-tool-use/barrier-refuses-readonly | brain | `brain guard` | guard-deny-readonly | deren stdin | message |
| hook-pre-tool-use/barrier-refuses-manifest | brain | `brain guard` | project-writable | write-manifest | message |
| hook-pre-tool-use/barrier-refuses-antigravity-outside | brain | `brain guard` | project-writable | agy-outside | message |
| hook-pre-tool-use/barrier-allows-main | brain | `brain guard` | project-writable | write-main | message |
| hook-post-tool-use/ignored-extension | ulguard | `ulguard post-edit --root {{WORLD}}` | project-writable | edit-json | message |
| hook-post-tool-use/markdown-outside-wiki | ulguard | wie oben | project-writable | edit-readme | message |
| hook-post-tool-use/broken-wiki-page | ulguard | wie oben | project-writable | edit-broken-page | message |
| lint/valid-page | brain | `brain lint {{WORLD}}/docs/wiki/index.md` | project-writable | — | data |
| lint/page-without-frontmatter | brain | `brain lint {{WORLD}}/docs/wiki/broken.md` | project-writable | — | data |
| wiki-gate/outside-git | brain | `brain wiki-gate --root {{WORLD}}` | project-writable | — | message |
| hook-session-start/no-session-id | ulguard | `ulguard hook session-start --host claude --root {{WORLD}}` | plain | session | data |
| check-commit-msg/english | ulinit | `ulinit check commit-msg {{WORLD}}/msg-en.txt` | plain | — | data |
| check-commit-msg/german | ulinit | `ulinit check commit-msg {{WORLD}}/msg-de.txt` | plain | — | data |

`brain guard` braucht `BRAIN_STATE_DIR` = Welt; der Rekorder setzt ihn. Die Worktree-Befehle und `status` bekommen keine Fälle — sie brauchen echte Git-Worktrees und Junctions; ihre umgezogenen Tests belegen sie (Paritätsliste).

- [ ] **Step 4: Übersetzungstabelle** — `testdata/cases/1a-map.toml`:

```toml
[[command]]
from = "ulguard post-edit --root {{WORLD}}"
to   = "loomux hook post-tool-use --host claude --root {{WORLD}}"

[[command]]
from = "ulguard hook session-start"
to   = "loomux hook session-start"

[[command]]
from = "ulguard --root {{WORLD}}"
to   = "loomux hook pre-tool-use --host claude --root {{WORLD}}"

[[command]]
from = "brain guard"
to   = "loomux hook pre-tool-use --host claude --root {{WORLD}}"

[[command]]
from = "brain lint"
to   = "loomux lint --root {{WORLD}}"

[[command]]
from = "brain wiki-gate"
to   = "loomux wiki-gate"

[[command]]
from = "ulinit check"
to   = "loomux check"
```
Die Reihenfolge zählt: die längere Vorsilbe steht vor der kürzeren.

```bash
go run ./cmd/loomux dev import-cases --map testdata/cases/1a-map.toml --from testdata/cases/1a-source --to testdata/cases/1a
```

- [ ] **Step 5: Suite** — `internal/cli/cases_test.go`

```go
package cli

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/cases"
)

func TestRecordedCasesOfStage1a(t *testing.T) {
	all, err := cases.DiscoverCases(filepath.Join("..", "..", "testdata", "cases", "1a"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 {
		t.Fatal("no cases found")
	}
	for _, c := range all {
		t.Run(c.Verb+"/"+c.Name, func(t *testing.T) {
			outcome, err := cases.RunCase(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Chdir(dir)
				t.Setenv("LOOMUX_STATE_DIR", dir)
				return Run(args, stdin, stdout, stderr)
			})
			if err != nil {
				t.Fatal(err)
			}
			if !outcome.Passed {
				t.Fatalf("%s\n%s", strings.Join(outcome.Mismatches, "\n"), c.Notes)
			}
		})
	}
}
```

- [ ] **Step 6: Rot lesen, nicht wegdrücken.** `go test ./internal/cli/ -run TestRecordedCasesOfStage1a`. Jeder rote Fall ist entweder ein Fehler in loomux (reparieren, test-first im betroffenen Paket) oder eine gewollte Abweichung. Nur im zweiten Fall wird die Erwartung in `testdata/cases/1a/<fall>/` angepasst — und **im selben Commit** in der Paritätsliste eingetragen. `1a-source` wird nie angepasst.
- [ ] **Step 7: Paritätsliste** — `docs/.superpowers/parity/stufe-1a.md`:

```markdown
# Paritätsliste Stufe 1a

**Quelle:** ultraloom `loomux-1a-source` (`9d01a60`), ultra-brain `loomux-1a-source` (`3cc72d2`).
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers, bevor Stufe 1a als fertig gilt.

| Fall / Bereich | Alt | Neu | Begründung | Freigabe |
|---|---|---|---|---|
| hook-pre-tool-use/unreadable-payload | ulguard: Exit 1, Host lässt den Aufruf durch | Exit 2 | Wächter scheitert geschlossen (Spec, Fehlerverhalten) | |
| Policy-Ablehnung, alle Fälle | ulguard: nur stderr | zusätzlich deny-Hülle auf stdout | ein Wächter, eine Antwortform (`guard.Refuse`) | |
| Befehlsregeln mit Lookahead | ulguard: Kompilierfehler verworfen, Regel greift nie | Konfiguration wird verweigert, Datei und Regel genannt | Befund 2026-09-14 | |
| Scratchpad der Claude-Sitzung | brain guard: verweigert | erlaubt unter `<temp>/claude/*/*/scratchpad/` | Spec, Abweichungsliste | |
| Manifestnamen | `.ultra-brain/config.toml`, `.brain.toml` | nur `.loomux/config.toml` | harter Schnitt | |
| Wiki-Ort | `[wiki] path`, `[area] wiki` in `.brain.toml`; `bundle` in `answers.toml` | nur `[layout] wiki` | eine Stelle für eine Frage | |
| `[check.lanes]` | brain liest Lanes aus dem Manifest | nicht gelesen | `[verify]` kommt in Stufe 2 | |
| hook-post-tool-use/broken-wiki-page | `uv run brain lint` als Kindprozess | Lint im Prozess | Spec, Hook-Pfad | |
| hook-session-start | meldet wartende Läufe aus `.ultraloom/runs` | meldet sie nicht; warnt vor veraltetem Pilot-Binary | Flow-Migration ist Folgeprojekt | |
| `lint --root` | nicht vorhanden | neues Flag, ohne Flag wie bisher | Fallsuite braucht einen festen Projektort | |
| Antigravity `run_command` | brain guard: kein Schreibwerkzeug | Policy prüft es nicht | Argumentschlüssel ungemessen; Stufe 2 | |
| `SRC_UB/pkg/gitenv/gitenv_test.go` | liest `vcs.py` | entfällt | kein Python im Produkt | |
| status, worktree link/unlink/remove | — | keine Fälle, umgezogene Tests | brauchen echte Worktrees und Junctions | |
| Mutationsrunde | — | in Stufe 1b | `dev mutants` entsteht dort | |
```
Jede in Schritt 6 neu gefundene Abweichung als weitere Zeile.

- [ ] **Step 8:** `sh .githooks/pre-commit` → 0. **Commit** — `Record the old hooks, translate them and hold loomux to them`.

---

### Task 15: Messen — `dev bench-hooks`, Startzeit, die 40 ms der Schranke

**Files:**
- Create: `internal/dev/benchhooks/benchhooks.go`, Test
- Modify: `internal/cli/dev.go` (`"bench-hooks"`)
- Create: `testdata/bench/1a-hooks.json`
- Create: `docs/benchmarks.md`, `docs/benchmarks.de.md`

**Interfaces:**
- Produces: `benchhooks.Case{Name, Dir, Stdin, Mode string; Steps []benchhooks.Step}`, `benchhooks.Step{Argv []string}`; `benchhooks.Run(cases []Case, n int, w io.Writer, exec func(Case, Step) (int, error), now func() time.Time) error`.

- [ ] **Step 1: Failing test** — mit einem `exec`, das nichts startet, und einer Uhr, die je Aufruf um feste Schritte vorrückt:

```go
func TestRunReportsColdAndTheWarmMedian(t *testing.T) {
	ticks := []time.Duration{0, 10, 10, 13, 13, 15, 15, 17} // start/stop pairs: cold 10ms, warm 3, 2, 2
	i := 0
	start := time.Unix(0, 0)
	now := func() time.Time { d := ticks[i] * time.Millisecond; i++; return start.Add(d) }
	var out bytes.Buffer
	c := Case{Name: "probe", Mode: "single", Steps: []Step{{Argv: []string{"x"}}}}
	err := Run([]Case{c}, 3, &out, func(Case, Step) (int, error) { return 0, nil }, now)
	if err != nil || !strings.Contains(out.String(), "| probe | 10.0 ms | 2.0 ms | 2.0 ms | 3.0 ms | [0] |") {
		t.Fatalf("%v\n%s", err, out.String())
	}
}
```
Dazu: `Mode "seq"` addiert zwei Schritte; `Mode "par"` startet beide nebenläufig (über ein `exec`, das auf einem Kanal wartet, bis beide laufen); ein `exec`-Fehler bricht mit Fehler ab; eine gerade Anzahl warmer Läufe mittelt die zwei mittleren Werte.

- [ ] **Step 2:** FAIL sehen, dann implementieren: der Harness vom 2026-09-14 (`docs/benchmarks.md` in ultraloom, Eintrag 16:05, Methode) mit injiziertem `exec` und `now`; die Tabelle hat die Spalten `case | cold (1st run) | warm median | warm min | warm max | exit codes`. Der echte `exec` (`os/exec`, Stdin aus `Case.Stdin`, `Dir`) bekommt `//coverage:exempt starts processes; the timing logic is tested through exec`. `dev bench-hooks <cases.json> [-n 20]` liest JSON in `[]Case`.
- [ ] **Step 3: Messfälle** — `testdata/bench/1a-hooks.json` mit denselben Fällen wie am 2026-09-14 16:05 (Edit auf `README.md` des loomux-Repos): `bin/loomux.exe hook pre-tool-use --host claude --root <repo>`; zum Vergleich im selben Lauf `ulguard --root <repo>` und `brain guard`, einzeln und parallel; `bin/loomux.exe hook session-start --host claude --root <repo>`; `bin/loomux.exe version` als Startboden. Absolute Pfade des Messrechners, weil es eine Messung und kein Test ist.
- [ ] **Step 4: Messen**

```bash
go run ./cmd/loomux dev bench-hooks testdata/bench/1a-hooks.json -n 20
```
Zielwert: `hook pre-tool-use` warm ≤ 72 ms (Spec, Hook-Pfad).

- [ ] **Step 5: Startzeit nachweisen**

```bash
GODEBUG=inittrace=1 bin/loomux.exe version
```
Jede `init`-Zeile über 1 ms wird mit Paketnamen notiert; stammt sie aus loomux-Code, ist das ein Verstoß gegen die Startzeit-Regel und wird vor dem Commit behoben (Wert beim ersten Gebrauch laden).

- [ ] **Step 6: Die 40 ms zerlegen.** Ein Benchmark in `internal/brain/guard` (`BenchmarkDecideAgainstTheRealRegistry`), der `Decide` gegen eine Kopie von `%LOCALAPPDATA%\brain\registry.toml` misst — die Kopie liegt in `$TEMP`, nicht im Repo, der Benchmark überspringt sich ohne `LOOMUX_BENCH_REGISTRY`. Mit `-cpuprofile` die drei größten Posten benennen; Verdacht, vor der Messung nicht als Befund zu schreiben: der `git rev-parse --git-common-dir`-Aufruf je Ziel (`askGit`).

```bash
go test ./internal/brain/guard/ -run '^$' -bench BenchmarkDecideAgainstTheRealRegistry -benchtime 50x -cpuprofile "$TEMP/guard.prof"
```
```bash
go tool pprof -top "$TEMP/guard.prof"
```

- [ ] **Step 7: Eintrag** in `docs/benchmarks.md` und `docs/benchmarks.de.md`, Form wie der ultraloom-Eintrag vom 2026-09-14 16:05: Datum und Uhrzeit, Repo und Commit, Ziel, Methode, Tabelle kalt/warm, Lesart mit dem Vergleich zu 33 / 72 / 73 / 114 ms, dem Ergebnis von `inittrace` und den drei Posten der Schranke. Kein Zielwert für Stufe 2, bevor die Posten benannt sind.
- [ ] **Step 8:** `sh .githooks/pre-commit` → 0. **Commit** — `Measure the merged guard against the two it replaces`.

---

### Task 16: Pilot — loomux prüft sich selbst

**Files:**
- Create (**Mensch**): `.loomux/config.toml`, `%LOCALAPPDATA%\loomux\registry.toml`
- Create: `.claude/settings.json`
- Modify: `docs/.superpowers/parity/stufe-1a.md`

- [ ] **Step 1 (Mensch): Projektkonfiguration** — `.loomux/config.toml`:

```toml
# loomux checks itself with its own binary (stage 1a pilot).
# Agents never write this file; propose a change, a human writes it.
[area]
scope = "project/loomux"

[layout]
wiki = "docs/wiki"

[[policy.paths.rules]]
match  = ["coverage.out", "bin/*"]
reason = "Build output belongs to the pre-commit gate, not to an edit."

[[policy.paths.rules]]
match  = ["testdata/cases/1a-source/**"]
reason = "Recordings of the old tools are evidence; re-record them, never edit them."

[[policy.commands.rules]]
regex  = '(^|[\n;&|(`])\s*git\s+push([^\w-]|$)'
reason = "Whether commits reach the remote is a human's decision."
```

- [ ] **Step 2 (Mensch): globale Registry** — `%LOCALAPPDATA%\loomux\registry.toml`:

```toml
[[area]]
scope     = "project/loomux"
path      = "C:/Users/micro/Documents/#GIT/loomux"
wiki      = "C:/Users/micro/Documents/#GIT/loomux/docs/wiki"
workspace = true
```
Solange nur dieser Bereich registriert ist, verweigert der Pilot jeder Sitzung **im loomux-Repo** Schreibzugriffe in andere Repos. Das ist gewollt; eine Sitzung, die gleichzeitig in ultraloom arbeitet, startet dort.

- [ ] **Step 3: Hooks eintragen** — `.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "\"${CLAUDE_PROJECT_DIR}/bin/loomux.exe\" hook session-start --host claude --root \"${CLAUDE_PROJECT_DIR}\"",
            "timeout": 20
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Write|Edit|MultiEdit|NotebookEdit|Bash|PowerShell",
        "hooks": [
          {
            "type": "command",
            "command": "\"${CLAUDE_PROJECT_DIR}/bin/loomux.exe\" hook pre-tool-use --host claude --root \"${CLAUDE_PROJECT_DIR}\"",
            "timeout": 15
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Write|Edit|MultiEdit|NotebookEdit",
        "hooks": [
          {
            "type": "command",
            "command": "\"${CLAUDE_PROJECT_DIR}/bin/loomux.exe\" hook post-tool-use --host claude --root \"${CLAUDE_PROJECT_DIR}\"",
            "timeout": 60
          }
        ]
      }
    ]
  }
}
```

- [ ] **Step 4: Rauchtest in einer neuen Claude-Sitzung im loomux-Repo** (Mensch startet sie; Ergebnisse in die Paritätsliste unter „Pilot“):
  1. Session-Start: kein Fehler. Nach einer Änderung an einer `.go`-Datei ohne Neubau (`touch internal/cli/cli.go`) erscheint die Warnung beim nächsten Session-Start; nach `sh .githooks/pre-commit` nicht mehr — auch nicht nach dem Commit, der auf das Tor folgt.
  2. Edit an `internal/cli/cli.go` (Kommentarzeile) → erlaubt; post-edit läuft ohne Meldung.
  3. Write an `.env` → verweigert, Grund `secrets`.
  4. Edit an `.loomux/config.toml` → verweigert.
  5. Write an `C:/Users/micro/Documents/#GIT/ultraloom/x.md` → verweigert.
  6. Write in das Scratchpad der Sitzung → erlaubt.
  7. Bash `git push` → verweigert.
  8. Commit mit englischer Nachricht → Tore grün; mit deutscher → commit-msg verweigert.

- [ ] **Step 5: Abschluss prüfen** gegen die Spec, „Eine Stufe ist fertig, wenn“:
  1. `go test ./internal/cli/ -run TestRecordedCasesOfStage1a` grün, jede Abweichung in der Liste — **Freigabespalte vom Nutzer ausgefüllt**;
  2. `sh .githooks/pre-commit` grün (Coverage-Tor);
  3. Mutationsrunde: nach 1b verschoben, in der Liste vermerkt;
  4. Eintrag in `docs/benchmarks.md`/`.de.md` aus Task 15;
  5. Rauchtest aus Step 4 vollständig.
- [ ] **Step 6: Commit** — `Let loomux guard its own repository`.
- [ ] **Step 7: Aufräumen (Mensch oder mit Ja):** die beiden Tag-Worktrees entfernen.

```bash
git -C "/c/Users/micro/Documents/#GIT/ultraloom" worktree remove "/c/Users/micro/Documents/#GIT/loomux-src/ul"
```
```bash
git -C "/c/Users/micro/Documents/#GIT/ultra-brain" worktree remove "/c/Users/micro/Documents/#GIT/loomux-src/ub"
```
Die Tags bleiben: sie sind die Quelle der Aufzeichnungen in `testdata/cases/1a-source`.

