# loomux Stufe 2c — Implementierungsplan: das Tor am Rundenende und die Subagenten-Hooks

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux hook stop` fährt an jedem Rundenende das Profil `stop` und hält
die Runde mit Exit 2 an, solange etwas rot ist. `loomux hook subagent-start` und
`subagent-stop` halten fest, was ein Subagent an `origin`, an lokalen Branches
und an `HEAD` bewegt hat. `stop` stellt diesen Befund dem Hauptagenten zu. Der
Antigravity-Adapter wird nach einer eigenen Messung gebaut.

**Architecture:** `internal/gitwork` bekommt den Inhaltsbaum (temporärer Index,
`git write-tree`), `ls-remote`, die lokalen Branches und `log --oneline`.
`internal/sessions` hält je Sitzung `base`, `blocks`, `green` und je Subagent
eine eigene, atomar geschriebene Datei. `internal/hooks/stop.go` und
`subagent.go` sind die Abläufe; die Prüfkette ist `verify.Plan`/`Run` im
Check-Scope wie bei `check`, dazu die Lane `lint/wiki` im Prozess. Die Parität
belegen Git-Welten, die eine `git.toml` beim Staging aufbaut, und zwei neue
Vergleichsklassen `state` und `finding`.

**Tech Stack:** Go (Toolchain 1.27.0, `go.mod` bleibt bei `go 1.25.0`),
Standardbibliothek, `github.com/BurntSushi/toml` über `third_party/toml`,
`git` ≥ 2.28 (für `init -b`). **Keine neue Abhängigkeit.**

**Spec:** `docs/.superpowers/specs/2026-09-19-loomux-stufe-2c-design.md`. Bei
Widerspruch gilt die Spec, außer an den Stellen, die der Abschnitt „Nachträge an
die Spec“ unten nennt; Task 20 trägt sie in die Spec ein.

## Global Constraints

- **Arbeitsort:** Worktree
  `C:\Users\micro\Documents\#GIT\loomux\.claude\worktrees\github-versioning-releases-cli-ca4f58`,
  Branch `sdd-2c`, abgezweigt von `origin/master` `99e474d`, ohne Upstream.
- **Modulpfad** `github.com/xidus90/loomux`.
- **Sprache:** Code, Bezeichner, Kommentare, Fehlermeldungen und
  Commit-Nachrichten englisch. Plan, Spec und `docs/.superpowers/parity/`
  deutsch. `docs/en/**` englisch, `docs/de/**` deutsch, gleiche Dateinamen.
- **Kein `init()`, keine Paketvariable, die Daten parst.**
- **Coverage 100 % je Funktion.** Eine Ausnahme nur mit
  `//coverage:exempt <reason>` als letzte Kommentarzeile direkt über `func`.
- **Das Tor weist ungestagte Eingaben ab** (`*.go`, `*.toml`, `go.mod`,
  `go.sum`, `testdata`, `.githooks`, `ci`, `.loomux`). Vor jedem Commit alles
  stagen, was die Aufgabe angefasst hat.
- **`.loomux/config.toml` und `.claude/settings*.json` schreibt kein Agent.**
  Tasks 1, 14 und 16 sind Halte für den Menschen.
- **Commits:** der Mensch ist Autor und Committer; kein Modell, kein Agent,
  kein `Co-Authored-By`. Conventional Commits; kein Commit trägt `!`.
  Mehrzeilige Nachrichten über eine Datei und `git commit -F`, nie ein Heredoc.
  **Vor jedem Commit `git branch --show-current` und `git log -1 --format=%h`
  lesen.** Niemand außer einem Menschen pusht; vor dem ersten Push läuft
  `release-pr`.
- **Determinismus:** keine Map-Iteration, deren Reihenfolge in ein Ergebnis
  einfließt; sortiert wird in Byte-Ordnung (`slices.Sorted(maps.Keys(m))`).
- **Plattform:** Windows zuerst; alles hier ist plattformneutral.
- **Git-Aufrufe** gehen durch `gitenv` (`gitwork.git`, `child.Run`), mit einer
  Ausnahme: der Aufbau der Git-Welten in `internal/cases`, der die
  Identitätsvariablen setzen muss, die `gitenv` streicht.
- **Subagenten**, falls eingesetzt: `model: "opus"` und `effort: "low"`, beides
  ausdrücklich gesetzt. Nach jedem Subagenten
  `git log -1 --format='%an <%ae>'` lesen.

## Nachträge an die Spec

Beim Planen gegen den Code gerechnet; Task 20 trägt sie in die Spec ein.

1. **Kein `GIT_SSH_COMMAND`.** Die Umgebungsvariable schlägt `core.sshCommand`
   und die SSH-Einstellung des Nutzers (etwa einen Agenten über ein eigenes
   `ssh`). `ls-remote` bekommt nur `GIT_TERMINAL_PROMPT=0`; ein SSH ohne TTY
   fragt nicht, sondern scheitert, und die Frist von 10 s fängt den Rest.
   `gitenv` streicht `GIT_TERMINAL_PROMPT` nicht (`gitenv.go:43`).
2. **Remote-`HEAD` bleibt in `refs`.** `ls-remote` meldet `<sha>\tHEAD`;
   Python hat diese Zeile mitverglichen (`subagent_stop.py` `_refs` und
   `_split` trennen nur die lokale Markierung `HEAD\t<sha>` ab). Go tut das
   auch, also meldet ein bewegter Standard-Branch zwei Zeilen: `origin HEAD
   moved …` und `origin refs/heads/master moved …`.
3. **`heads` fehlt in übersetzten Snapshots.** Python kannte keine lokalen
   Branches. Ein Snapshot mit `"heads": null` vergleicht keine Branches; einer
   mit `{}` schon. Go schreibt nie `null`.
4. **Ein ignoriertes Wurzelverzeichnis** (`gitwork.ErrIgnoredRoot`) zählt für
   `stop` wie „kein Repo“: kein Fingerabdruck, die Kette läuft jedes Mal.
5. **`lint/wiki` steht am Ende.** `cli/check.go` und `stop` hängen die
   Wiki-Lane hinter die geplanten Jobs; `After` sind Indizes, und ein
   Einfügen verschöbe sie. Die Reihenfolgeregel der 2a-Spec („Arten in
   Anfrage-Reihenfolge“) gilt damit für `lint/wiki` nicht.
6. **„Nichts geprüft“ heißt `CheckVerdict` mit Notizen.** Eine angefragte Art
   ohne Lane, die lief, ist bei `stop` kein Grün: Exit 1 mit den Notizen, keine
   Basis. Das ist Pythons Regel „alle roten sind `UNAVAILABLE`“.
7. **Befunde ohne Präfix in der Datei.** Die Agent-Datei trägt die Zeilen ohne
   `subagent <id>: `; `stop` setzt das Präfix beim Zustellen, die
   Vergleichsklasse `finding` beim Vergleich.
8. **Eine Basis, die es nicht mehr gibt, zählt wie keine.** Nach `--amend`
   oder Rebase und einem `gc` löst `TreeOf(base)` nicht mehr auf. Das ist
   kein Git-Fehler, der hält und zählt, sondern: Warnung, HEAD als Basis, ein
   grüner Lauf setzt die Basis neu. Sonst hielte `stop` drei Runden je
   Nutzerrunde mit einem Hinweis auf Lanes, die nichts damit zu tun haben.
9. **Mit Befunden wird auch Exit 1 zu Exit 2.** Die Befunddateien sind nach
   dem Zustellen gelöscht; nur ein Halt sorgt dafür, dass der Hauptagent sie
   liest. Ein Ladefehler der Konfiguration, ein Planfehler oder ein
   aufgebrauchtes Budget halten die Runde deshalb an, wenn im selben Aufruf
   Befunde zugestellt wurden, und lassen sie sonst enden. Das zählt nicht als
   Blockade.
10. **Die Welten sind überall dieselben Bytes.** `.gitattributes` setzt
    `* text=auto eol=lf`; jeder Checkout hat LF, gleich wie `core.autocrlf`
    steht. Die SHAs der Git-Welten sind damit auf jeder Maschine gleich; der
    Linux-Lauf in `ci.yml` fährt die Fallsuiten heute nicht, nur `child`.

---

## File Structure

| Datei | Verantwortung |
|---|---|
| `internal/cases/gitworld.go` | `GitWorld`, `BuildGitWorld`, `InfraPath` — Git-Welten aus `git.toml` |
| `internal/cases/runner.go`, `case.go` (ändern) | Git-Welt nach dem Staging; Klassen `state` und `finding`; `.git` aus dem Baumvergleich |
| `internal/cases/hookstate.go` | `compareState`, `compareFindings` |
| `internal/dev/recordcase/recordcase.go` (ändern) | Git-Welt beim Aufzeichnen, stderr in eine Datei |
| `internal/dev/importcases/hookstate.go` | `.ultraloom/hooks/*.json` und `.claude/.no-verify` falten |
| `internal/sessions/state.go` (ändern) | `SessionState{Blocks, Base, Green}` |
| `internal/sessions/agents.go` | `Snapshot`, `AgentFile`, `ReadAgent`, `WriteAgent`, `RemoveAgent`, `Findings` |
| `internal/sessions/sessions.go` (ändern) | `Forget` nimmt das Sitzungsverzeichnis mit |
| `internal/gitwork/gitwork.go` (ändern) | `Head`, `TreeOf`, `ContentTree`, `LocalHeads`, `LogOneline`, `EmptyTree`, `ErrNotRepository` |
| `internal/gitwork/remote.go` | `LsRemote`, `RemoteTimeout` |
| `internal/hosts/hostio.go`, `claude.go` (ändern) | `AgentID`, `AgentType`; Ereignis in `WriteContext` |
| `internal/hosts/antigravity.go` | Adapter nach der Messung (aus `codex.go` gezogen) |
| `internal/verify/schema.go` (ändern) | eingebautes Profil `stop` |
| `internal/hooks/wikigate.go` | `WikiGateJobs` — `lint/wiki` im Check-Scope |
| `internal/hooks/stop.go` | `Stop`, `RunStop`, `StopEnv`, `DefaultStopBudget`, `MaxBlocks`, `NoVerifyMarker` |
| `internal/hooks/subagent.go` | `SubagentStart`, `SubagentStop`, Snapshot und Vergleich |
| `internal/hooks/guard.go`, `status.go` (ändern) | Marker-Regel, Stop-Abschnitt und Audit |
| `internal/cli/hook.go`, `check.go` (ändern) | drei Ereignisse, `--budget` für `stop`; Wiki-Lane in `check` |
| `internal/cli/cases_2c_test.go` | Fallsuite 2c |
| `testdata/cases/2c-worlds/`, `2c-payloads/`, `2c-source/`, `2c-map.toml`, `2c/` | Welten, Nutzlasten, Aufzeichnungen, Übersetzung, übersetzte Fälle |
| `docs/.superpowers/parity/stufe-2c.md` | Abweichungsliste, Mutationsrunde |
| `docs/{en,de}/*.md`, READMEs, Fusions-Spec, 2c-Spec (ändern) | Doku |

---

### Task 0: Arbeitsort prüfen

**Files:** keine. **Interfaces:** keine.

- [ ] **Schritt 1: Ort und Basis lesen**

```sh
git branch --show-current
git log -1 --format='%h %an <%ae>'
git status --porcelain
```

Erwartet: `sdd-2c`, HEAD auf dem letzten Spec-Commit, Autor der Mensch,
Arbeitsbaum sauber (bis auf diesen Plan, falls noch nicht committet).

- [ ] **Schritt 2: Tor einmal leer fahren**

```sh
sh ci/gate.sh
```

Erwartet: grün (gemessen am 2026-09-19: 21–22 s). Ist es das nicht, liegt der
Fehler auf `master` — melden, nicht reparieren.

---

### Task 1: **Halt** — Messung auf der Claude-Seite

Die Spec verlangt den Nachweis, dass `SubagentStart` und `SubagentStop` die
`session_id` des Hauptagenten und dieselbe `agent_id` tragen. Ohne ihn baut
Task 11 auf eine Annahme.

**Files:**
- Create: `testdata/cases/2c-payloads/claude-subagent-start.json`,
  `claude-subagent-stop.json`, `claude-stop.json`, `README.md`

- [ ] **Schritt 1: Dem Menschen den Rekorder nennen**

Wörtlich vorlegen, zum Eintragen in `.claude/settings.local.json` (nicht
eingecheckt):

```json
{
  "hooks": {
    "SubagentStart": [{"hooks": [{"type": "command", "timeout": 20,
      "command": "uv run --no-project python -c \"import os,sys,uuid; d=os.path.join(os.environ['CLAUDE_PROJECT_DIR'],'.loomux','probe'); os.makedirs(d,exist_ok=True); open(os.path.join(d,'start-'+uuid.uuid4().hex+'.json'),'w',encoding='utf-8').write(sys.stdin.read())\""}]}],
    "SubagentStop": [{"hooks": [{"type": "command", "timeout": 20,
      "command": "uv run --no-project python -c \"import os,sys,uuid; d=os.path.join(os.environ['CLAUDE_PROJECT_DIR'],'.loomux','probe'); os.makedirs(d,exist_ok=True); open(os.path.join(d,'stop-'+uuid.uuid4().hex+'.json'),'w',encoding='utf-8').write(sys.stdin.read())\""}]}],
    "Stop": [{"hooks": [{"type": "command", "timeout": 20,
      "command": "uv run --no-project python -c \"import os,sys,uuid; d=os.path.join(os.environ['CLAUDE_PROJECT_DIR'],'.loomux','probe'); os.makedirs(d,exist_ok=True); open(os.path.join(d,'turn-'+uuid.uuid4().hex+'.json'),'w',encoding='utf-8').write(sys.stdin.read())\""}]}]
  }
}
```

Python über `uv` und kein PowerShell: Der Befehl enthält kein `$`, das die
Shell des Hooks (unter Windows ungemessen, ob `cmd` oder Git Bash) vorher
ersetzen könnte.

Dann in einer **neuen** Claude-Code-Sitzung in diesem Worktree einen Subagenten
starten lassen (ein Explore-Agent mit „list the files in internal/sessions“)
und die Runde enden lassen. Warten, bis der Mensch sagt, dass es gelaufen ist.

- [ ] **Schritt 2: Auswerten**

```sh
ls .loomux/probe/
```

Für jede Datei `session_id`, `agent_id`, `agent_type`, `hook_event_name`
lesen. Zu belegen:

1. `start-*` und `stop-*` tragen dieselbe `agent_id`.
2. Beide tragen die `session_id` von `turn-*` (dem Hauptagenten).
3. Beide tragen `agent_type`.

Trifft 1 oder 2 nicht zu: **Halt**, dem Menschen die Felder zeigen. Der
Schlüssel der Agent-Datei (Task 4) ändert sich dann, und die Spec bekommt einen
Nachtrag, bevor es weitergeht.

- [ ] **Schritt 3: Nutzlasten ablegen**

Je eine Datei nach `testdata/cases/2c-payloads/` kopieren, mit gekürzten
Pfaden: `transcript_path`, `agent_transcript_path` und `cwd` werden
`{{WORLD}}/…`, `session_id` wird `s1`, `agent_id` wird `a1`. Alle übrigen
Felder bleiben wörtlich. `README.md` (deutsch): Datum, Claude-Code-Version
(`claude --version`), was belegt ist.

Danach `.loomux/probe/` löschen und den Menschen bitten, den Rekorder wieder aus
`.claude/settings.local.json` zu nehmen.

- [ ] **Schritt 4: Commit**

```sh
git add testdata/cases/2c-payloads
git commit -m "test(hooks): record the Claude Code subagent and stop payloads"
```

---

### Task 2: `internal/cases` — Git-Welten aus `git.toml`

Stop und die Subagenten-Hooks brauchen echte Repos mit Commits und Remote. Ein
`world/.git/` lässt sich im loomux-Repo nicht committen; die Welt beschreibt ihr
Repo deshalb in `git.toml`, und das Staging baut es auf.

**Files:**
- Create: `internal/cases/gitworld.go`, `internal/cases/gitworld_test.go`
- Modify: `internal/cases/runner.go` (`RunCase` baut die Welt; `collectFiles`
  überspringt `InfraPath`)

**Interfaces:**
- Produces:
  - `const GitWorldFile = "git.toml"`
  - `type GitWorld struct { Commits []GitCommit; Remote *GitRemote; Branches map[string]int; Worktree map[string]string }`
  - `type GitCommit struct { Message string; Paths []string; Files map[string]string }`
  - `type GitRemote struct { Push map[string]int }`
  - `func BuildGitWorld(dir string) error` — nichts ohne `git.toml`
  - `func InfraPath(rel string) bool` — `rel` mit Schrägstrichen; wahr für
    `.git`, `.origin.git` und alles darunter

- [ ] **Schritt 1: Tests schreiben**

```go
package cases

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

func writeWorld(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const twoCommits = `
[[commit]]
message = "base"
paths = ["a.txt"]

[[commit]]
message = "second"
[commit.files]
"a.txt" = "two\n"

[branches]
feature = 1

[remote.push]
master = 2

[worktree]
"a.txt" = "three\n"
`

func TestBuildGitWorldMakesTheDeclaredRepository(t *testing.T) {
	dir := writeWorld(t, map[string]string{
		"git.toml":            twoCommits,
		"a.txt":               "one\n",
		".ultraloom/state.json": `{"base": "{{COMMIT:1}}", "head": "{{COMMIT:2}}"}`,
	})
	if err := BuildGitWorld(dir); err != nil {
		t.Fatal(err)
	}
	first := gitOut(t, dir, "rev-parse", "HEAD~1")
	second := gitOut(t, dir, "rev-parse", "HEAD")
	if got := gitOut(t, dir, "rev-parse", "feature"); got != first {
		t.Fatalf("feature at %s, want %s", got, first)
	}
	if got := gitOut(t, dir, "ls-remote", "origin", "refs/heads/master"); !strings.HasPrefix(got, second) {
		t.Fatalf("origin master: %q, want %s", got, second)
	}
	if got := gitOut(t, dir, "show", "HEAD~1:a.txt"); got != "one" {
		t.Fatalf("first commit holds %q", got)
	}
	body, _ := os.ReadFile(filepath.Join(dir, "a.txt"))
	if string(body) != "three\n" {
		t.Fatalf("worktree a.txt = %q", body)
	}
	state, _ := os.ReadFile(filepath.Join(dir, ".ultraloom", "state.json"))
	if want := `{"base": "` + first + `", "head": "` + second + `"}`; string(state) != want {
		t.Fatalf("tokens: %s", state)
	}
	// The fixture files of the world are no change a hook should see.
	if got := gitOut(t, dir, "status", "--porcelain"); got != "M a.txt" {
		t.Fatalf("status %q", got)
	}
}

// Fixed identity and dates: the same declaration is the same SHA on every
// machine, which recordings and replays depend on.
func TestBuildGitWorldIsDeterministic(t *testing.T) {
	var heads []string
	for range 2 {
		dir := writeWorld(t, map[string]string{"git.toml": twoCommits, "a.txt": "one\n"})
		if err := BuildGitWorld(dir); err != nil {
			t.Fatal(err)
		}
		heads = append(heads, gitOut(t, dir, "rev-parse", "HEAD"))
	}
	if heads[0] != heads[1] {
		t.Fatalf("two builds, two heads: %v", heads)
	}
}

func TestBuildGitWorldWithoutDeclarationDoesNothing(t *testing.T) {
	dir := writeWorld(t, map[string]string{"a.txt": "x"})
	if err := BuildGitWorld(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Fatalf(".git: %v", err)
	}
}

func TestBuildGitWorldRefuses(t *testing.T) {
	cases := map[string]string{
		"bad toml":        "[[commit]",
		"commit out of range": "[[commit]]\nmessage = \"a\"\n[branches]\nx = 2\n",
		"push out of range":   "[[commit]]\nmessage = \"a\"\n[remote.push]\nmaster = 0\n",
		"missing path":    "[[commit]]\nmessage = \"a\"\npaths = [\"nope.txt\"]\n",
		"token out of range": "[[commit]]\nmessage = \"a\"\n",
	}
	for name, decl := range cases {
		t.Run(name, func(t *testing.T) {
			files := map[string]string{"git.toml": decl}
			if name == "token out of range" {
				files["x.json"] = "{{COMMIT:9}}"
			}
			if err := BuildGitWorld(writeWorld(t, files)); err == nil {
				t.Fatal("want an error")
			}
		})
	}
}

func TestInfraPath(t *testing.T) {
	for rel, want := range map[string]bool{
		".git": true, ".git/index": true, ".origin.git/HEAD": true,
		"a.txt": false, "sub/.git": false, ".github/x": false,
	} {
		if InfraPath(rel) != want {
			t.Errorf("InfraPath(%q) = %v", rel, !want)
		}
	}
}
```

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/cases/ -run 'GitWorld|InfraPath'`
Erwartet: FAIL, `BuildGitWorld` undefiniert.

- [ ] **Schritt 3: Implementieren**

`internal/cases/gitworld.go`:

```go
package cases

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/gitenv"
)

// GitWorldFile declares the repository a staged world becomes.
const GitWorldFile = "git.toml"

// GitWorld is git.toml: commits in order, local branches and what the remote
// holds by commit number (from 1), and files changed after the last commit.
type GitWorld struct {
	Commits  []GitCommit       `toml:"commit"`
	Remote   *GitRemote        `toml:"remote"`
	Branches map[string]int    `toml:"branches"`
	Worktree map[string]string `toml:"worktree"`
}

// GitCommit commits the staged files it names and the files it spells out.
type GitCommit struct {
	Message string            `toml:"message"`
	Paths   []string          `toml:"paths"`
	Files   map[string]string `toml:"files"`
}

// GitRemote is a bare repository at .origin.git, and the commit each of its
// branches is pushed to.
type GitRemote struct {
	Push map[string]int `toml:"push"`
}

// excluded are the files of the test bench itself: a hook that measures the
// working tree must not see them as somebody's change.
var excluded = []string{"/git.toml", "/faketool.json", "/.origin.git/", "/.ultraloom/", "/.loomux/", "/.claude/"}

// InfraPath reports whether a slash-separated path inside a world belongs to
// the repositories BuildGitWorld made rather than to the world.
func InfraPath(rel string) bool {
	top, _, _ := strings.Cut(rel, "/")
	return top == ".git" || top == ".origin.git"
}

// BuildGitWorld turns dir into the repository its git.toml declares, then
// puts each commit's SHA in place of {{COMMIT:<n>}} in every other file.
//
// Git runs here without gitenv's strip and with an identity of its own: the
// variables gitenv takes out are exactly the ones that make the same
// declaration the same SHA on every machine. The user's global and system
// configuration are kept out for the same reason.
func BuildGitWorld(dir string) error {
	raw, err := os.ReadFile(filepath.Join(dir, GitWorldFile))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var w GitWorld
	if _, err := toml.Decode(string(raw), &w); err != nil {
		return fmt.Errorf("%s: %w", GitWorldFile, err)
	}
	empty, err := os.CreateTemp("", "gitworld-config-*")
	if err != nil {
		return err
	}
	empty.Close()
	defer os.Remove(empty.Name())
	g := &worldGit{dir: dir, config: empty.Name()}
	g.run("init", "-q", "-b", "master")
	if g.err == nil {
		// Written, not appended: with the user's configuration kept out, no
		// template put anything there worth keeping.
		g.fail(writeFile(dir, ".git/info/exclude", strings.Join(excluded, "\n")+"\n"))
	}
	var shas []string
	for i, c := range w.Commits {
		shas = append(shas, g.commit(i+1, c))
	}
	commit := func(n int) (string, error) {
		if n < 1 || n > len(shas) {
			return "", fmt.Errorf("%s names commit %d of %d", GitWorldFile, n, len(shas))
		}
		return shas[n-1], nil
	}
	for _, name := range slices.Sorted(maps.Keys(w.Branches)) {
		sha, err := commit(w.Branches[name])
		g.fail(err)
		g.run("branch", name, sha)
	}
	if w.Remote != nil {
		g.run("init", "-q", "--bare", "-b", "master", ".origin.git")
		g.run("remote", "add", "origin", "./.origin.git")
		for _, branch := range slices.Sorted(maps.Keys(w.Remote.Push)) {
			sha, err := commit(w.Remote.Push[branch])
			g.fail(err)
			g.run("push", "-q", "origin", sha+":refs/heads/"+branch)
		}
	}
	for _, path := range slices.Sorted(maps.Keys(w.Worktree)) {
		g.fail(writeFile(dir, path, w.Worktree[path]))
	}
	if g.err != nil {
		return g.err
	}
	return replaceCommitTokens(dir, shas)
}

// worldGit runs git in one world and keeps the first failure.
type worldGit struct {
	dir, config string
	err         error
}

func (g *worldGit) fail(err error) {
	if g.err == nil {
		g.err = err
	}
}

func (g *worldGit) run(args ...string) string {
	return g.runAt(0, args...)
}

// runAt runs git with commit n's date; a date per commit keeps the SHAs
// apart when two commits carry the same tree and message.
func (g *worldGit) runAt(n int, args ...string) string {
	if g.err != nil {
		return ""
	}
	date := fmt.Sprintf("@%d +0000", 1767225600+60*n)
	cmd := exec.Command("git", append([]string{"-c", "core.autocrlf=false", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=" + os.DevNull}, args...)...)
	cmd.Dir = g.dir
	cmd.Env = append(gitenv.Environ(),
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+g.config,
		"GIT_AUTHOR_NAME=loomux cases", "GIT_AUTHOR_EMAIL=cases@loomux.invalid", "GIT_AUTHOR_DATE="+date,
		"GIT_COMMITTER_NAME=loomux cases", "GIT_COMMITTER_EMAIL=cases@loomux.invalid", "GIT_COMMITTER_DATE="+date)
	out, err := cmd.CombinedOutput()
	if err != nil {
		g.err = fmt.Errorf("git %s in %s: %v: %s", strings.Join(args, " "), g.dir, err, strings.TrimSpace(string(out)))
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (g *worldGit) commit(n int, c GitCommit) string {
	paths := slices.Clone(c.Paths)
	for _, path := range slices.Sorted(maps.Keys(c.Files)) {
		g.fail(writeFile(g.dir, path, c.Files[path]))
		paths = append(paths, path)
	}
	if len(paths) > 0 {
		g.run(append([]string{"add", "--"}, paths...)...)
	}
	g.runAt(n, "commit", "-q", "--allow-empty", "-m", c.Message)
	return g.run("rev-parse", "HEAD")
}

func writeFile(dir, rel, body string) error {
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

// replaceCommitTokens puts the SHAs into every file of the world outside the
// repositories and the declaration.
func replaceCommitTokens(dir string, shas []string) error {
	token := regexp.MustCompile(`\{\{COMMIT:(\d+)\}\}`)
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if InfraPath(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == GitWorldFile {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var bad error
		out := token.ReplaceAllFunc(data, func(m []byte) []byte {
			n, _ := strconv.Atoi(string(token.FindSubmatch(m)[1]))
			if n < 1 || n > len(shas) {
				bad = fmt.Errorf("%s names commit %d of %d", rel, n, len(shas))
				return m
			}
			return []byte(shas[n-1])
		})
		if bad != nil {
			return bad
		}
		return os.WriteFile(path, out, 0o644)
	})
}
```

Für die Arme, die kein Test erreicht (ein `os.CreateTemp`, das scheitert; ein
`WalkDir`-Fehler während der Ersetzung), je ein `//coverage:exempt` mit Grund,
oder die Temp-Erzeugung als Paketvariable-Naht (`var createTemp = os.CreateTemp`)
wie `mkdirTemp` in `runner.go` und im Test ersetzen. Die Naht ist vorzuziehen.

`internal/cases/runner.go`, in `RunCase` direkt nach `StageWorld`:

```go
	// A world may declare a repository; it is built after staging, so the
	// SHAs are the same every time and {{COMMIT:<n>}} can stand for them.
	if err := BuildGitWorld(tmpDir); err != nil {
		return nil, err
	}
```

In `collectFiles`, vor `d.IsDir()`:

```go
		// The repositories a git world is built into are the bench, not the
		// world: git rewrites its index on a read, which is no change anybody
		// made.
		if rel, _ := filepath.Rel(dir, path); InfraPath(filepath.ToSlash(rel)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
```

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/cases/ -cover`
Erwartet: PASS, 100.0 %. Die Fallsuiten 1a, 1b-1, 1b-2 und 2a
(`go test ./internal/cli/ -run TestCases -count=1`) bleiben grün: Keine ihrer
Welten hat eine `git.toml`.

- [ ] **Schritt 5: Commit**

```sh
git add internal/cases
git commit -m "test(cases): build git worlds from git.toml when a world is staged"
```

---

### Task 3: Der Rekorder baut Git-Welten und nimmt stderr auf

**Files:**
- Modify: `internal/dev/recordcase/recordcase.go`, `recordcase_test.go`

**Interfaces:**
- Consumes: `cases.BuildGitWorld`, `cases.InfraPath` (Task 2)
- Produces: ein Fall kann eine Datei `stderr` tragen; `LoadCase` liest sie
  nicht.

- [ ] **Schritt 1: Tests schreiben**

In `recordcase_test.go`, neben den vorhandenen Helfern (der Testprozess spielt
das alte Binary über `helperArgv`, wie die MCP-Tests):

```go
func TestRecordKeepsStderrAsEvidence(t *testing.T) {
	out := filepath.Join(t.TempDir(), "case")
	err := Record(Spec{
		Argv:  helperArgv(t),
		Cmd:   "old say-on-stderr",
		World: t.TempDir(),
		Out:   out,
		Env:   []string{"HELPER_SCRIPT=stderr:gone wrong"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(out, "stderr"))
	if err != nil || string(got) != "gone wrong\n" {
		t.Fatalf("stderr = %q, %v", got, err)
	}
}

func TestRecordWritesNoStderrFileForSilence(t *testing.T) {
	out := filepath.Join(t.TempDir(), "case")
	if err := Record(Spec{Argv: helperArgv(t), Cmd: "old quiet", World: t.TempDir(), Out: out}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "stderr")); !os.IsNotExist(err) {
		t.Fatalf("stderr file: %v", err)
	}
}

func TestRecordBuildsTheGitWorldAndLeavesItOutOfWorldAfter(t *testing.T) {
	world := t.TempDir()
	os.WriteFile(filepath.Join(world, "git.toml"), []byte("[[commit]]\nmessage = \"base\"\n"), 0o644)
	os.WriteFile(filepath.Join(world, "head.txt"), []byte("{{COMMIT:1}}"), 0o644)
	out := filepath.Join(t.TempDir(), "case")
	if err := Record(Spec{Argv: helperArgv(t), Cmd: "old quiet", World: world, Out: out}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "world_after", ".git")); !os.IsNotExist(err) {
		t.Fatalf("world_after/.git: %v", err)
	}
	head, _ := os.ReadFile(filepath.Join(out, "world_after", "head.txt"))
	if len(head) != 40 {
		t.Fatalf("token not replaced: %q", head)
	}
}
```

Den Helfer des Pakets so erweitern, dass `HELPER_SCRIPT=stderr:<text>` den
Text mit Zeilenende auf stderr schreibt und mit 0 endet. Liest der vorhandene
Helfer sein Skript anders, dieselbe Fähigkeit in dessen Form ergänzen — vorher
`helperMCP` und `helperFront` in `mcp_test.go` lesen.

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/dev/recordcase/ -run 'Stderr|GitWorld'`
Erwartet: FAIL.

- [ ] **Schritt 3: Implementieren**

In `Record`, nach `cases.StageWorld`:

```go
	if err := cases.BuildGitWorld(tmp); err != nil {
		return err
	}
```

Vor `cmd.Run()`:

```go
	// Kept as evidence and never compared: the old hooks said everything on
	// stderr, and a deviation list has to quote what they said.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
```

Nach dem Aufbau von `files`:

```go
	if said := bytes.ReplaceAll(stderr.Bytes(), []byte("\r\n"), []byte("\n")); len(said) > 0 {
		files["stderr"] = cases.Normalize(said, tmp)
	}
```

In `copyTree`, im `WalkDir`-Callback direkt nach dem Ermitteln von `rel`:

```go
		// The repositories BuildGitWorld made are rebuilt at every replay;
		// copying them would put a .git into the corpus.
		if cases.InfraPath(filepath.ToSlash(rel)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
```

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/dev/recordcase/ -cover`
Erwartet: PASS, 100.0 %.

- [ ] **Schritt 5: Commit**

```sh
git add internal/dev/recordcase
git commit -m "test(cases): record git worlds and keep stderr as evidence"
```

---

### Task 4: `internal/sessions` — `green` und eine Datei je Subagent

**Files:**
- Modify: `internal/sessions/state.go`, `state_test.go`, `sessions.go`, `sessions_test.go`
- Create: `internal/sessions/agents.go`, `agents_test.go`

**Interfaces:**
- Produces:
  - `type SessionState struct { Blocks int; Base, Green string }` — `Snapshots` entfällt
  - `type Snapshot struct { Head string; Heads, Refs map[string]string; Remote string }` mit JSON `head`, `heads`, `refs`, `remote`
  - `const RemoteOK = "ok"`, `const RemoteUnavailable = "unavailable"`
  - `type AgentFile struct { Snapshot *Snapshot; Finding []string }` mit JSON `snapshot,omitempty`, `finding,omitempty`
  - `func ReadAgent(root, sessionID, agentID string) (AgentFile, bool)`
  - `func WriteAgent(root, sessionID, agentID string, f AgentFile) error` — atomar
  - `func RemoveAgent(root, sessionID, agentID string) error` — fehlende Datei ist kein Fehler
  - `type Finding struct { AgentID string; Lines []string }`
  - `func Findings(root, sessionID string) []Finding` — nur Dateien mit `finding`, nach Dateiname sortiert
  - `Forget` entfernt zusätzlich `<StateDir>/<safe id>/`

- [ ] **Schritt 1: Tests schreiben**

`state_test.go`: Die Tests, die `Snapshots` setzen oder lesen, verlieren diese
Felder. Neu:

```go
func TestStateKeepsGreen(t *testing.T) {
	root := t.TempDir()
	want := SessionState{Blocks: 2, Base: "b", Green: "g"}
	if err := WriteState(root, "s1", want); err != nil {
		t.Fatal(err)
	}
	if got := ReadState(root, "s1"); got != want {
		t.Fatalf("got %+v", got)
	}
}

// A file from before stage 2c carries snapshots and no green; it still holds
// a counter and a base worth keeping.
func TestReadStateOfAnOlderFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, filepath.FromSlash(StateDir), "s1.json")
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(`{"base": "b", "blocks": 1, "snapshots": {"a": "x"}}`), 0o644)
	if got := ReadState(root, "s1"); got != (SessionState{Blocks: 1, Base: "b"}) {
		t.Fatalf("got %+v", got)
	}
}

func TestWriteStateLeavesGreenOutWhenEmpty(t *testing.T) {
	root := t.TempDir()
	WriteState(root, "s1", SessionState{Blocks: 0})
	raw, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(StateDir), "s1.json"))
	if string(raw) != `{"base":null,"blocks":0}` {
		t.Fatalf("%s", raw)
	}
}
```

`TestReadState…` für einen fehlenden `blocks` bleibt (leerer Zustand); der für
fehlende `snapshots` kehrt sich um: ohne `snapshots`, mit `blocks`, ist die
Datei gültig.

`agents_test.go`:

```go
package sessions

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAgentRoundTrip(t *testing.T) {
	root := t.TempDir()
	snap := &Snapshot{Head: "h", Heads: map[string]string{}, Refs: map[string]string{"HEAD": "r"}, Remote: RemoteOK}
	if err := WriteAgent(root, "s1", "a/1", AgentFile{Snapshot: snap}); err != nil {
		t.Fatal(err)
	}
	got, ok := ReadAgent(root, "s1", "a/1")
	if !ok || !reflect.DeepEqual(got.Snapshot, snap) {
		t.Fatalf("got %+v, %v", got.Snapshot, ok)
	}
	// The id is cleaned like a session id: it may not decide where the file lands.
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(StateDir), "s1", "agents", "a1.json")); err != nil {
		t.Fatal(err)
	}
}

// null and {} are two answers: a snapshot translated from Python never took
// the local branches, one taken by loomux found none.
func TestSnapshotKeepsNilHeadsApartFromEmpty(t *testing.T) {
	root := t.TempDir()
	WriteAgent(root, "s1", "a", AgentFile{Snapshot: &Snapshot{Remote: RemoteOK}})
	got, _ := ReadAgent(root, "s1", "a")
	if got.Snapshot.Heads != nil {
		t.Fatalf("heads = %#v", got.Snapshot.Heads)
	}
}

func TestReadAgentOfNothing(t *testing.T) {
	if _, ok := ReadAgent(t.TempDir(), "s1", "a"); ok {
		t.Fatal("want absent")
	}
}

func TestReadAgentOfADamagedFile(t *testing.T) {
	root := t.TempDir()
	WriteAgent(root, "s1", "a", AgentFile{Finding: []string{"x"}})
	os.WriteFile(filepath.Join(root, filepath.FromSlash(StateDir), "s1", "agents", "a.json"), []byte("{"), 0o644)
	if _, ok := ReadAgent(root, "s1", "a"); ok {
		t.Fatal("want absent")
	}
}

func TestFindingsAreSortedAndSkipSnapshots(t *testing.T) {
	root := t.TempDir()
	WriteAgent(root, "s1", "b", AgentFile{Finding: []string{"two"}})
	WriteAgent(root, "s1", "a", AgentFile{Finding: []string{"one"}})
	WriteAgent(root, "s1", "c", AgentFile{Snapshot: &Snapshot{Remote: RemoteOK}})
	dir := filepath.Join(root, filepath.FromSlash(StateDir), "s1", "agents")
	os.WriteFile(filepath.Join(dir, "d.json"), []byte("{"), 0o644)
	os.WriteFile(filepath.Join(dir, ".agent-x.tmp"), []byte(`{"finding":["no"]}`), 0o644)
	os.Mkdir(filepath.Join(dir, "e.json"), 0o755)
	want := []Finding{{AgentID: "a", Lines: []string{"one"}}, {AgentID: "b", Lines: []string{"two"}}}
	if got := Findings(root, "s1"); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
	if Findings(root, "nobody") != nil {
		t.Fatal("want nil for a session without agents")
	}
}

func TestRemoveAgent(t *testing.T) {
	root := t.TempDir()
	WriteAgent(root, "s1", "a", AgentFile{Finding: []string{"x"}})
	if err := RemoveAgent(root, "s1", "a"); err != nil {
		t.Fatal(err)
	}
	if err := RemoveAgent(root, "s1", "a"); err != nil {
		t.Fatalf("second remove: %v", err)
	}
}

func TestWriteAgentReportsADirectoryItCannotMake(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".loomux", "state"), 0o755)
	os.WriteFile(filepath.Join(root, filepath.FromSlash(StateDir)), []byte("a file"), 0o644)
	if err := WriteAgent(root, "s1", "a", AgentFile{}); err == nil {
		t.Fatal("want an error")
	}
}
```

`sessions_test.go`, neu:

```go
func TestForgetTakesTheAgentsWith(t *testing.T) {
	root := t.TempDir()
	WriteState(root, "s1", SessionState{})
	WriteAgent(root, "s1", "a", AgentFile{Finding: []string{"x"}})
	if err := Forget(root, "s1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(StateDir), "s1")); !os.IsNotExist(err) {
		t.Fatalf("agents dir: %v", err)
	}
}
```

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/sessions/`
Erwartet: FAIL (Kompilierfehler: `Green`, `WriteAgent` …).

- [ ] **Schritt 3: Implementieren**

`state.go`:

```go
// SessionState is what one session carries between two calls of the stop
// gate: how many turn ends in a row it held, the commit it measures from,
// and the tree it last found green.
//
// Base is a string and not a pointer: the empty string is never a commit,
// so both absences are one, and an absent base is written as JSON null.
// Green is the tree SHA of the last green run, or "" before the first.
type SessionState struct {
	Blocks int
	Base   string
	Green  string
}

// stateFile is the shape on disk. Pointers, because a missing key and a
// null must both arrive as absent. Alphabetical, so one state is one key
// order whoever wrote it.
//
// The snapshots of the subagents lived here until stage 2c and now have a
// file each (agents.go): two subagents that start in one message run their
// hooks in parallel, and one file for both lost one of them. An older file
// still reads -- its snapshots key is ignored.
type stateFile struct {
	Base   *string `json:"base"`
	Blocks *int    `json:"blocks"`
	Green  string  `json:"green,omitempty"`
}

// ReadState is what this session left behind, or an empty state. Every
// failure answers empty and none is reported: raising would end a turn over
// a counter whose worst case is a few extra rounds.
func ReadState(root, sessionID string) SessionState {
	raw, err := os.ReadFile(statePath(root, sessionID))
	if err != nil {
		return SessionState{}
	}
	var file stateFile
	if err := json.Unmarshal(raw, &file); err != nil || file.Blocks == nil {
		return SessionState{}
	}
	state := SessionState{Blocks: *file.Blocks, Green: file.Green}
	if file.Base != nil {
		state.Base = *file.Base
	}
	return state
}

// WriteState keeps this state for the next call.
//
//coverage:exempt the json.Marshal err arm needs a stateFile field that does not encode; an int and strings always do
func WriteState(root, sessionID string, state SessionState) error {
	path := statePath(root, sessionID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating the directory for %s: %w", path, err)
	}
	file := stateFile{Blocks: &state.Blocks, Green: state.Green}
	if state.Base != "" {
		base := state.Base
		file.Base = &base
	}
	body, err := json.Marshal(file)
	if err != nil {
		return fmt.Errorf("encoding the state of session %s: %w", sessionID, err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
```

`agents.go`:

```go
package sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Remote says whether the remote answered when a snapshot was taken.
const (
	RemoteOK          = "ok"
	RemoteUnavailable = "unavailable"
)

// Snapshot is where origin, the local branches and HEAD stood when a
// subagent started. Heads is nil in a snapshot that never looked at the
// local branches -- one translated from the Python hooks -- and empty in one
// that found none; the comparison only reads branches from the second.
type Snapshot struct {
	Head   string            `json:"head"`
	Heads  map[string]string `json:"heads"`
	Refs   map[string]string `json:"refs"`
	Remote string            `json:"remote"`
}

// AgentFile is one subagent's file: its snapshot while it runs, what it
// changed once it stopped. The lines carry no prefix; whoever shows them
// names the agent.
type AgentFile struct {
	Snapshot *Snapshot `json:"snapshot,omitempty"`
	Finding  []string  `json:"finding,omitempty"`
}

// Finding is what one stopped subagent left for the main agent.
type Finding struct {
	AgentID string
	Lines   []string
}

func agentDir(root, sessionID string) string {
	return filepath.Join(root, filepath.FromSlash(StateDir), safeName(sessionID), "agents")
}

func agentPath(root, sessionID, agentID string) string {
	return filepath.Join(agentDir(root, sessionID), safeName(agentID)+".json")
}

// ReadAgent is one subagent's file, or false when there is none it can read.
func ReadAgent(root, sessionID, agentID string) (AgentFile, bool) {
	return readAgentFile(agentPath(root, sessionID, agentID))
}

func readAgentFile(path string) (AgentFile, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return AgentFile{}, false
	}
	var f AgentFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return AgentFile{}, false
	}
	return f, true
}

// WriteAgent replaces one subagent's file in one step: a temp file in the
// same directory, then a rename. A background subagent can stop while the
// main agent's stop gate reads the directory, and a half-written file would
// read as none.
//
//coverage:exempt the json.Marshal err arm needs a field that does not encode; strings, maps and slices of strings always do
func WriteAgent(root, sessionID, agentID string, f AgentFile) error {
	dir := agentDir(root, sessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	body, err := json.Marshal(f)
	if err != nil {
		return fmt.Errorf("encoding the file of agent %s: %w", agentID, err)
	}
	tmp, err := createTemp(dir, ".agent-*.tmp")
	if err != nil {
		return fmt.Errorf("writing into %s: %w", dir, err)
	}
	_, werr := tmp.Write(body)
	cerr := tmp.Close()
	if werr != nil || cerr != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("writing %s: %v", tmp.Name(), cmpErr(werr, cerr))
	}
	path := agentPath(root, sessionID, agentID)
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("replacing %s: %w", path, err)
	}
	return nil
}

// createTemp is a seam: a write into a file just created fails only when the
// disk does.
var createTemp = os.CreateTemp

func cmpErr(a, b error) error {
	if a != nil {
		return a
	}
	return b
}

// RemoveAgent deletes one subagent's file; one that is not there is gone.
func RemoveAgent(root, sessionID, agentID string) error {
	path := agentPath(root, sessionID, agentID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing %s: %w", path, err)
	}
	return nil
}

// Findings are the files of stopped subagents, sorted by name. A file that
// still holds a snapshot belongs to a subagent that runs; one that cannot be
// read is skipped, like a damaged session file.
func Findings(root, sessionID string) []Finding {
	entries, err := os.ReadDir(agentDir(root, sessionID))
	if err != nil {
		return nil
	}
	var out []Finding
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		f, ok := readAgentFile(filepath.Join(agentDir(root, sessionID), name))
		if !ok || len(f.Finding) == 0 {
			continue
		}
		out = append(out, Finding{AgentID: strings.TrimSuffix(name, ".json"), Lines: f.Finding})
	}
	slices.SortFunc(out, func(a, b Finding) int { return strings.Compare(a.AgentID, b.AgentID) })
	return out
}
```

`os.ReadDir` sortiert schon nach Namen; das `SortFunc` bleibt, weil die Ordnung
die Ausgabe von `stop` bestimmt und nicht an einer Eigenschaft von `ReadDir`
hängen soll.

Der Test für den Schreibfehler setzt `createTemp` auf eine Funktion, die eine
Datei zurückgibt, deren `Write` scheitert (etwa eine schon geschlossene
`*os.File`). Ein zweiter Test setzt `createTemp` auf einen Fehler.

`sessions.go`, `Forget`:

```go
// Forget removes this session's own file and its subagents' files.
//
// Nobody did this before, which is why `Others` needs a staleness rule at all.
// What is not there is not an error: a session that never wrote state still
// ends.
func Forget(root, sessionID string) error {
	path := statePath(root, sessionID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing %s: %w", path, err)
	}
	dir := filepath.Join(root, filepath.FromSlash(StateDir), safeName(sessionID))
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing %s: %w", dir, err)
	}
	return nil
}
```

Den Paketkommentar in `sessions.go` berichtigen: Die Dateien schreibt loomux
(`session-start`, `stop`, `subagent-start`, `subagent-stop`), nicht mehr die
Python-Seite; ultraloom schreibt nach `.ultraloom/hooks/`, einem anderen
Verzeichnis.

- [ ] **Schritt 4: Aufrufer nachziehen**

```sh
grep -rn "Snapshots" internal --include=*.go
```

Jede Fundstelle außerhalb von `sessions` entfernt das Feld (heute nur Tests in
`internal/hooks`). Dann:

Run: `go test ./internal/sessions/ ./internal/hooks/ -cover`
Erwartet: PASS, `sessions` 100.0 %.

- [ ] **Schritt 5: Commit**

```sh
git add internal/sessions internal/hooks
git commit -m "feat(sessions): keep the green tree and a file per subagent"
```

---

### Task 5: Import und Vergleichsklassen `state` und `finding`

**Files:**
- Create: `internal/dev/importcases/hookstate.go`, `hookstate_test.go`
- Modify: `internal/dev/importcases/importcases.go` (`TranslateWorld` ruft
  `foldHookState`)
- Create: `internal/cases/hookstate.go`, `hookstate_test.go`
- Modify: `internal/cases/case.go` (Klassen zulassen), `runner.go` (Klassen
  anwenden)

**Interfaces:**
- Consumes: `sessions.WriteState`, `WriteAgent`, `ReadState`, `ReadAgent`,
  `Snapshot` (Task 4)
- Produces:
  - `func ParseOldSnapshot(stored string) *sessions.Snapshot` (importcases)
  - Klassen `state` und `finding` in `LoadCase` und `RunCase`

- [ ] **Schritt 1: Tests schreiben — Import**

```go
package importcases

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/sessions"
)

func TestParseOldSnapshot(t *testing.T) {
	got := ParseOldSnapshot("c1\tHEAD\nc1\trefs/heads/master\nHEAD\tc0\n")
	want := &sessions.Snapshot{Head: "c0", Refs: map[string]string{"HEAD": "c1", "refs/heads/master": "c1"}, Remote: sessions.RemoteOK}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
	// Python stored "" for a remote that did not answer.
	if got := ParseOldSnapshot("HEAD\tc0\n"); got.Remote != sessions.RemoteUnavailable || got.Refs != nil {
		t.Fatalf("got %+v", got)
	}
}

func TestFoldHookStateMovesTheOldFiles(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, ".ultraloom", "hooks")
	os.MkdirAll(old, 0o755)
	os.WriteFile(filepath.Join(old, "s1.json"), []byte(`{"base": "b", "blocks": 2, "snapshots": {"a1": "c1\trefs/heads/master\nHEAD\tc1\n"}}`), 0o644)
	os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	os.WriteFile(filepath.Join(dir, ".claude", ".no-verify"), nil, 0o644)
	if err := foldHookState(dir); err != nil {
		t.Fatal(err)
	}
	if got := sessions.ReadState(dir, "s1"); got != (sessions.SessionState{Blocks: 2, Base: "b"}) {
		t.Fatalf("state %+v", got)
	}
	agent, ok := sessions.ReadAgent(dir, "s1", "a1")
	if !ok || agent.Snapshot.Head != "c1" || agent.Snapshot.Heads != nil {
		t.Fatalf("agent %+v %v", agent.Snapshot, ok)
	}
	for _, gone := range []string{".ultraloom/hooks", ".claude/.no-verify"} {
		if _, err := os.Stat(filepath.Join(dir, gone)); !os.IsNotExist(err) {
			t.Fatalf("%s: %v", gone, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".loomux", "no-verify")); err != nil {
		t.Fatal(err)
	}
}

func TestFoldHookStateWithoutOldFiles(t *testing.T) {
	if err := foldHookState(t.TempDir()); err != nil {
		t.Fatal(err)
	}
}

func TestFoldHookStateRefusesADamagedFile(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, ".ultraloom", "hooks")
	os.MkdirAll(old, 0o755)
	os.WriteFile(filepath.Join(old, "s1.json"), []byte("{"), 0o644)
	if err := foldHookState(dir); err == nil {
		t.Fatal("want an error: a recording whose state cannot be read is no evidence")
	}
}
```

- [ ] **Schritt 2: Tests schreiben — Vergleichsklassen**

`internal/cases/hookstate_test.go`:

```go
package cases

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/sessions"
)

func TestCompareStateReadsBaseAndBlocks(t *testing.T) {
	want, got := t.TempDir(), t.TempDir()
	sessions.WriteState(want, "s1", sessions.SessionState{Blocks: 1, Base: "b"})
	sessions.WriteState(got, "s1", sessions.SessionState{Blocks: 1, Base: "b", Green: "ignored"})
	if m := compareState(want, got); len(m) != 0 {
		t.Fatalf("mismatches %v", m)
	}
	sessions.WriteState(got, "s1", sessions.SessionState{Blocks: 2, Base: "c"})
	if m := compareState(want, got); len(m) != 2 {
		t.Fatalf("mismatches %v", m)
	}
}

func TestCompareStateOfAMissingSession(t *testing.T) {
	want := t.TempDir()
	sessions.WriteState(want, "s1", sessions.SessionState{Blocks: 1})
	if m := compareState(want, t.TempDir()); len(m) != 1 {
		t.Fatalf("mismatches %v", m)
	}
}

func TestCompareFindings(t *testing.T) {
	got := t.TempDir()
	sessions.WriteAgent(got, "s1", "a1", sessions.AgentFile{Finding: []string{"origin x is new at c"}})
	sessions.WriteAgent(got, "s1", "a2", sessions.AgentFile{Snapshot: &sessions.Snapshot{}})
	if m := compareFindings([]byte("subagent a1: origin x is new at c\n"), got); len(m) != 0 {
		t.Fatalf("mismatches %v", m)
	}
	if m := compareFindings([]byte("subagent a1: something else\n"), got); len(m) != 1 {
		t.Fatalf("mismatches %v", m)
	}
	if m := compareFindings(nil, t.TempDir()); len(m) != 0 {
		t.Fatalf("mismatches %v", m)
	}
	os.MkdirAll(filepath.Join(got, ".loomux", "state", "hooks", "s1", "agents", "x.json"), 0o755)
	if m := compareFindings([]byte("subagent a1: origin x is new at c\n"), got); len(m) != 0 {
		t.Fatalf("a directory is no agent file: %v", m)
	}
}
```

Dazu in `case_test.go` je ein Fall, dass `LoadCase` `compare` = `state` und
`finding` annimmt, und in `runner_test.go` je ein `RunCase` mit einer kleinen
Welt je Klasse (der `RunFunc` schreibt den Zustand bzw. die Agent-Datei in
`dir`), einmal passend, einmal nicht.

- [ ] **Schritt 3: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/cases/ ./internal/dev/importcases/`
Erwartet: FAIL.

- [ ] **Schritt 4: Implementieren — Import**

`internal/dev/importcases/hookstate.go`:

```go
package importcases

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/sessions"
)

// oldState is .ultraloom/hooks/<id>.json as state.py wrote it.
type oldState struct {
	Base      *string           `json:"base"`
	Blocks    int               `json:"blocks"`
	Snapshots map[string]string `json:"snapshots"`
}

// foldHookState moves the old hooks' files into loomux's layout: the session
// file under .loomux/state/hooks, each snapshot into its agent's file, and
// the marker to .loomux/no-verify. Written through internal/sessions, so the
// bytes are the ones loomux writes and reads.
func foldHookState(dir string) error {
	old := filepath.Join(dir, ".ultraloom", "hooks")
	entries, err := os.ReadDir(old)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(old, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var s oldState
		if err := json.Unmarshal(raw, &s); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		state := sessions.SessionState{Blocks: s.Blocks}
		if s.Base != nil {
			state.Base = *s.Base
		}
		if err := sessions.WriteState(dir, id, state); err != nil {
			return err
		}
		for _, agent := range slices.Sorted(maps.Keys(s.Snapshots)) {
			if err := sessions.WriteAgent(dir, id, agent, sessions.AgentFile{Snapshot: ParseOldSnapshot(s.Snapshots[agent])}); err != nil {
				return err
			}
		}
	}
	if err := os.RemoveAll(old); err != nil {
		return err
	}
	return foldMarker(dir)
}

// foldMarker moves .claude/.no-verify to where loomux looks for it.
func foldMarker(dir string) error {
	from := filepath.Join(dir, ".claude", ".no-verify")
	if _, err := os.Stat(from); os.IsNotExist(err) {
		return nil
	}
	to := filepath.Join(dir, ".loomux", "no-verify")
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	return os.Rename(from, to)
}

// ParseOldSnapshot reads what subagent_start.py stored: the lines of
// `git ls-remote origin`, and a line `HEAD\t<sha>` for the local HEAD. The
// remote lines are `<sha>\t<ref>`, the marker the other way round, which is
// how subagent_stop.py's _split tells them apart. A snapshot without one
// remote line is a remote that did not answer: Python stored "" for it.
func ParseOldSnapshot(stored string) *sessions.Snapshot {
	s := &sessions.Snapshot{Remote: sessions.RemoteUnavailable}
	for _, line := range strings.Split(stored, "\n") {
		if head, ok := strings.CutPrefix(line, "HEAD\t"); ok {
			s.Head = head
			continue
		}
		sha, ref, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		if s.Refs == nil {
			s.Refs = map[string]string{}
			s.Remote = sessions.RemoteOK
		}
		s.Refs[ref] = sha
	}
	return s
}
```

In `TranslateWorld`, als erste Zeile:

```go
	if err := foldHookState(dir); err != nil {
		return err
	}
```

- [ ] **Schritt 5: Implementieren — Vergleichsklassen**

`internal/cases/hookstate.go`:

```go
package cases

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/sessions"
)

// compareState pins what the stop gate decided: for every session file the
// recording left, the base it measures from and the blocks it counted. The
// green tree and every other file are loomux's own and not compared.
func compareState(expectedRoot, actualRoot string) []string {
	var out []string
	for _, id := range sessionIDs(expectedRoot) {
		want, got := sessions.ReadState(expectedRoot, id), sessions.ReadState(actualRoot, id)
		if want.Base != got.Base {
			out = append(out, fmt.Sprintf("session %s: base expected %q, got %q", id, want.Base, got.Base))
		}
		if want.Blocks != got.Blocks {
			out = append(out, fmt.Sprintf("session %s: blocks expected %d, got %d", id, want.Blocks, got.Blocks))
		}
	}
	return out
}

func sessionIDs(root string) []string {
	entries, _ := os.ReadDir(filepath.Join(root, filepath.FromSlash(sessions.StateDir)))
	var ids []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			ids = append(ids, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	return ids
}

// compareFindings pins what subagent-stop found: the lines the Python hook
// printed against the finding lines of every agent file, each with the
// prefix Python put in front, both sides sorted.
func compareFindings(expected []byte, actualRoot string) []string {
	want := lines(string(expected))
	var got []string
	paths, _ := filepath.Glob(filepath.Join(actualRoot, filepath.FromSlash(sessions.StateDir), "*", "agents", "*.json"))
	for _, path := range paths {
		session := filepath.Base(filepath.Dir(filepath.Dir(path)))
		agent := strings.TrimSuffix(filepath.Base(path), ".json")
		f, ok := sessions.ReadAgent(actualRoot, session, agent)
		if !ok {
			continue
		}
		for _, line := range f.Finding {
			got = append(got, "subagent "+agent+": "+line)
		}
	}
	slices.Sort(want)
	slices.Sort(got)
	if slices.Equal(want, got) {
		return nil
	}
	return []string{fmt.Sprintf("findings expected:\n%s\ngot:\n%s", strings.Join(want, "\n"), strings.Join(got, "\n"))}
}

func lines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
```

`case.go`, die Prüfung der Klasse:

```go
		if !slices.Contains([]string{"data", "message", "lanes", "state", "finding"}, compare) {
			return nil, fmt.Errorf("unknown compare %q in %s", compare, cleanDir)
		}
```

und den Kommentar am Feld `Compare` um die zwei Klassen ergänzen.

`runner.go`, in `RunCase` den Vergleich ersetzen:

```go
	// A message case pins the exit code alone: its wording is loomux's own. A
	// lanes case pins the verdict per kind. A state case pins what a stop
	// gate decided, a finding case what a subagent hook found; both read
	// loomux's state, and the rest of the tree is theirs to differ in.
	switch c.Compare {
	case "lanes":
		mismatches = append(mismatches, compareLanes(c.Stdout, actualStdout)...)
	case "state":
		// Always against world_after: a git world has its commit tokens
		// replaced at every staging, so the recorder always wrote one, and
		// the world itself holds tokens where the SHAs belong.
		if !c.HasWorldAfter {
			mismatches = append(mismatches, "a state case needs a world_after")
			break
		}
		mismatches = append(mismatches, compareState(filepath.Join(c.Path, "world_after"), tmpDir)...)
	case "finding":
		mismatches = append(mismatches, compareFindings(c.Stdout, tmpDir)...)
	case "message":
	default:
		if !bytes.Equal(actualStdout, c.Stdout) {
			mismatches = append(mismatches, fmt.Sprintf("stdout mismatch: expected %d bytes, got %d bytes", len(c.Stdout), len(actualStdout)))
		}
	}
	if c.HasWorldAfter && c.Compare != "state" && c.Compare != "finding" {
```

(der Rest des `if` bleibt).

- [ ] **Schritt 6: Tests laufen lassen**

Run: `go test ./internal/cases/ ./internal/dev/importcases/ -cover`
Erwartet: PASS, beide 100.0 %. Dann alle Fallsuiten:
`go test ./internal/cli/ -run TestCases -count=1` — grün.

- [ ] **Schritt 7: Commit**

```sh
git add internal/cases internal/dev/importcases
git commit -m "test(cases): fold the old hook state and compare stop and subagent cases"
```

---

### Task 6: Welten, Aufzeichnung und Import

**Files:**
- Create: `testdata/cases/2c-worlds/<welt>/…`, `testdata/cases/2c-source/…`
  (vom Rekorder geschrieben), `testdata/cases/2c-map.toml`,
  `testdata/cases/2c/…` (vom Import geschrieben)

**Interfaces:**
- Consumes: Tasks 2, 3, 5; die Nutzlasten aus Task 1.

- [ ] **Schritt 1: Die Referenz prüfen**

```sh
git -C "C:/Users/micro/Documents/#GIT/ultraloom" rev-parse HEAD
uvreal="$(command -v uv)"
"$uvreal" run --no-sync --project "C:/Users/micro/Documents/#GIT/ultraloom" ultraloom hook stop --help
```

Erwartet: `9d01a60…` und eine Hilfe, die `--root` nennt. Steht `--root` dort
vor `hook` statt dahinter, die `--cmd`-Zeilen unten entsprechend umstellen.
Anderes HEAD: Halt, den Menschen fragen.

- [ ] **Schritt 2: Werkzeug-Attrappen bauen**

Wie in 2a, Task 2, Schritt 5: `faketool` als `uv.exe` und `uvx.exe` in
`$TMP/faketools`. `git` bleibt echt.

- [ ] **Schritt 3: Gemeinsamer Grundstock**

Jede `stop`-Welt enthält `pyproject.toml` und `tests/test_a.py` wörtlich aus
`testdata/cases/2a-worlds/python-green/`, dazu `a.py` (`x = 1\n`) und eine
`faketool.json` wie `python-green` (grün) bzw. wie `python-red-lint` (rot).
Die Sitzungsdatei `.ultraloom/hooks/s1.json` ist Python-Form:

```json
{"base": "{{COMMIT:1}}", "blocks": 0, "snapshots": {}}
```

Nutzlasten (`stdin`) liegen **außerhalb** der Welten, unter
`testdata/cases/2c-payloads/`; in einer Welt wären sie eine untracked Datei
und damit für `stop` eine Änderung:

- `stop.json`: `{"session_id": "s1", "hook_event_name": "Stop", "stop_hook_active": false}`
- `subagent-start.json`: `{"session_id": "s1", "agent_id": "a1", "agent_type": "general-purpose", "hook_event_name": "SubagentStart"}`
- `subagent-stop.json`: wie `subagent-start.json` mit `"hook_event_name": "SubagentStop"`
- `subagent-no-agent.json`: `{"session_id": "s1", "hook_event_name": "SubagentStart"}`
- `not-json.txt`: `not json`

Wo Task 1 andere Feldnamen belegt hat, gelten die aus Task 1.

- [ ] **Schritt 4: Welten**

`git.toml` der `stop`-Welten, außer wo die Tabelle Abweichendes nennt:

```toml
[[commit]]
message = "base"
paths = ["pyproject.toml", "tests/test_a.py", "a.py"]

[[commit]]
message = "second"
[commit.files]
"a.py" = "x = 2\n"

[worktree]
"a.py" = "x = 3\n"
```

| Welt | Abweichung vom Grundstock | zeigt |
|---|---|---|
| `stop-green` | — | grün: Exit 0, `base` = Commit 2 |
| `stop-red` | Fixture rot (`uvx ruff check` Exit 1) | rot: Exit 2, `blocks` 1 |
| `stop-no-base` | Sitzungsdatei `{"base": null, "blocks": 0, "snapshots": {}}` | ohne Basis gegen HEAD; grün |
| `stop-marker` | Fixture rot, dazu `.claude/.no-verify` | Exit 0, nichts läuft |
| `stop-bad-payload` | `--stdin` `not-json.txt` | Exit 1 |
| `stop-unchanged` | kein `[worktree]`, Sitzungsdatei mit `"base": "{{COMMIT:2}}"` | Exit 0, nichts läuft |
| `stop-gave-up` | Fixture rot, `"blocks": 3` | Python: Exit 0, `blocks` bleibt 3 |
| `stop-untracked-only` | Fixture rot, kein `[worktree]`, `base` Commit 2, dazu eine neue Datei `b.py` | Python: Exit 0 ohne Lauf |

Subagenten-Welten, je mit Remote:

```toml
[[commit]]
message = "base"
paths = ["a.txt"]

[[commit]]
message = "second"
[commit.files]
"a.txt" = "two\n"

[remote.push]
master = 2
```

und `a.txt` (`one\n`). Die Sitzungsdatei hält den Snapshot von `a1` in
Python-Form, JSON-kodiert (`\t`, `\n`):

| Welt | Snapshot `a1` / Änderung | Python schreibt |
|---|---|---|
| `subagent-start-no-agent` | `--stdin` `subagent-no-agent.json`, kein Snapshot | Exit 1 |
| `subagent-start-records` | kein Snapshot | Exit 0, Snapshot in `world_after` |
| `subagent-stop-no-snapshot` | kein Snapshot | `subagent a1: no snapshot for this subagent; nothing to compare` |
| `subagent-stop-ref-moved` | `{{COMMIT:1}}\tHEAD\n{{COMMIT:1}}\trefs/heads/master\nHEAD\t{{COMMIT:1}}\n` | `origin HEAD moved …`, `origin refs/heads/master moved …`, `new commit …` |
| `subagent-stop-ref-new` | wie `ref-moved`, dazu `[remote.push] feature = 1` in `git.toml` | zusätzlich `origin refs/heads/feature is new at …` |
| `subagent-stop-ref-gone` | wie `ref-moved` mit einer Zeile `{{COMMIT:1}}\trefs/heads/old\n` mehr | zusätzlich `origin refs/heads/old is gone; it was …` |
| `subagent-stop-new-commit` | `{{COMMIT:2}}\tHEAD\n{{COMMIT:2}}\trefs/heads/master\nHEAD\t{{COMMIT:1}}\n` | nur `new commit …` |

- [ ] **Schritt 5: Aufzeichnen**

Je Welt:

```sh
ul="C:/Users/micro/Documents/#GIT/ultraloom"
go run ./cmd/loomux dev record-case \
  --argv "$uvreal run --no-sync --project $ul ultraloom" \
  --path-prepend "$TMP/faketools" \
  --env "LOOMUX_FAKE_TOOL_FIXTURE={{WORLD}}/faketool.json" \
  --cmd "ultraloom hook stop --root {{WORLD}}" \
  --stdin testdata/cases/2c-payloads/stop.json \
  --world testdata/cases/2c-worlds/stop-green \
  --out testdata/cases/2c-source/hook-stop/green \
  --notes "ultraloom at tag loomux-1a-source (9d01a60); tools answered by faketool; git world from git.toml" \
  --compare state
```

Die Verben heißen `hook-stop`, `hook-subagent-start`, `hook-subagent-stop`;
der Fallname ist der Weltname ohne Präfix. `--compare`: `state` für `stop`
(außer `stop-bad-payload`: `message`), `finding` für `subagent-stop`,
`message` für `subagent-start-no-agent` und **`state`** für
`subagent-start-records`: Dort schreiben beide Seiten einen Snapshot, Python
mit `heads: null` nach dem Falten, Go mit den echten Branches, und der
Baumvergleich der Klasse `message` wäre immer rot. `state` pinnt Exit-Code,
`base` und `blocks` und lässt den Snapshot aus; was im Snapshot steht, prüfen
die Unit-Tests von Task 11.

Nach jedem Fall `exit` und `stderr` lesen und gegen die Tabelle halten. Weicht
Python von der Spalte „zeigt“ ab, gilt die Aufzeichnung; die Tabelle hier wird
berichtigt, und der Unterschied kommt in die Abweichungsliste (Task 17), wenn
Go ihn nicht teilt.

- [ ] **Schritt 6: Übersetzen und importieren**

`testdata/cases/2c-map.toml`:

```toml
# Stage 2c: the old session hooks run under the same verbs in loomux; the
# host is a flag there, and loomux init writes it.

[[command]]
from = "ultraloom hook stop"
to   = "loomux hook stop --host claude"

[[command]]
from = "ultraloom hook subagent-start"
to   = "loomux hook subagent-start --host claude"

[[command]]
from = "ultraloom hook subagent-stop"
to   = "loomux hook subagent-stop --host claude"
```

```sh
go run ./cmd/loomux dev import-cases --map testdata/cases/2c-map.toml --from testdata/cases/2c-source --to testdata/cases/2c
```

Stichprobe: `testdata/cases/2c/hook-stop/green/world/.loomux/state/hooks/s1.json`
ist Go-Form, `…/world/.ultraloom/hooks/` fehlt.

- [ ] **Schritt 7: Commit**

```sh
git add testdata/cases/2c-worlds testdata/cases/2c-payloads testdata/cases/2c-source testdata/cases/2c-map.toml testdata/cases/2c
git commit -m "test(cases): record the stage 2c session hooks from ultraloom"
```

---

### Task 7: `internal/gitwork` — Inhaltsbaum, Branches, Remote, Log

**Files:**
- Modify: `internal/gitwork/gitwork.go`, `gitwork_test.go`
- Create: `internal/gitwork/remote.go`, `remote_test.go`

**Interfaces:**
- Produces:
  - `const EmptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"`
  - `var ErrNotRepository = errors.New("not a git working tree")`
  - `func Head(root string) (string, error)` — `""`, `nil` bei unborn HEAD;
    `ErrNotRepository` bzw. `ErrIgnoredRoot` gewrappt
  - `func TreeOf(root, commit string) (string, error)`
  - `func ContentTree(root, scratch string) (string, error)`
  - `func LocalHeads(root string) (map[string]string, error)` — Schlüssel `refs/heads/<name>`
  - `func LogOneline(root, from, to string) ([]string, error)`
  - `func LsRemote(root, remote string) (map[string]string, error)`
  - `const RemoteTimeout = 10 * time.Second`

- [ ] **Schritt 1: Tests schreiben**

Die vorhandenen Tests legen Repos mit einem Helfer an; ihn lesen und
weiterverwenden (`gitwork_test.go`). Neu:

```go
func TestHeadOfAnUnbornRepository(t *testing.T) {
	root := t.TempDir()
	mustGit(t, root, "init", "-q")
	if head, err := Head(root); err != nil || head != "" {
		t.Fatalf("head %q, %v", head, err)
	}
}

func TestHeadOutsideARepository(t *testing.T) {
	if _, err := Head(t.TempDir()); !errors.Is(err, ErrNotRepository) {
		t.Fatalf("err %v", err)
	}
}

func TestContentTreeMatchesHeadWhenClean(t *testing.T) {
	root := repoWithCommit(t)
	tree, err := ContentTree(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	head, _ := TreeOf(root, "HEAD")
	if tree != head {
		t.Fatalf("tree %s, head tree %s", tree, head)
	}
}

func TestContentTreeSeesChangesAndNewFilesButNotState(t *testing.T) {
	root := repoWithCommit(t)
	clean, _ := ContentTree(root, t.TempDir())
	os.MkdirAll(filepath.Join(root, ".loomux", "state"), 0o755)
	os.WriteFile(filepath.Join(root, ".loomux", "state", "x"), []byte("x"), 0o644)
	if tree, _ := ContentTree(root, t.TempDir()); tree != clean {
		t.Fatal("machine state moved the tree")
	}
	os.WriteFile(filepath.Join(root, "new.txt"), []byte("n"), 0o644)
	if tree, _ := ContentTree(root, t.TempDir()); tree == clean {
		t.Fatal("a new file did not move the tree")
	}
	// The real index is untouched.
	if out := mustGit(t, root, "diff", "--cached", "--name-only"); out != "" {
		t.Fatalf("staged: %q", out)
	}
}

func TestContentTreeOfAnUnbornRepository(t *testing.T) {
	root := t.TempDir()
	mustGit(t, root, "init", "-q")
	if tree, err := ContentTree(root, t.TempDir()); err != nil || tree != EmptyTree {
		t.Fatalf("tree %s, %v", tree, err)
	}
}

func TestLocalHeadsAndLog(t *testing.T) {
	root := repoWithCommit(t)
	first := mustGit(t, root, "rev-parse", "HEAD")
	mustGit(t, root, "branch", "feature")
	os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0o644)
	mustGit(t, root, "add", "b.txt")
	mustGit(t, root, "commit", "-q", "-m", "second")
	heads, err := LocalHeads(root)
	if err != nil || heads["refs/heads/feature"] != first || len(heads) != 2 {
		t.Fatalf("heads %v, %v", heads, err)
	}
	log, err := LogOneline(root, first, "HEAD")
	if err != nil || len(log) != 1 || !strings.HasSuffix(log[0], " second") {
		t.Fatalf("log %v, %v", log, err)
	}
}
```

`remote_test.go`:

```go
func TestLsRemoteReadsTheRemote(t *testing.T) {
	root := repoWithCommit(t)
	bare := t.TempDir()
	mustGit(t, bare, "init", "-q", "--bare")
	mustGit(t, root, "remote", "add", "origin", bare)
	mustGit(t, root, "push", "-q", "origin", "HEAD:refs/heads/master")
	refs, err := LsRemote(root, "origin")
	head := mustGit(t, root, "rev-parse", "HEAD")
	if err != nil || refs["refs/heads/master"] != head {
		t.Fatalf("refs %v, %v", refs, err)
	}
}

func TestLsRemoteWithoutRemote(t *testing.T) {
	if _, err := LsRemote(repoWithCommit(t), "origin"); err == nil {
		t.Fatal("want an error")
	}
}

func TestLsRemoteGivesUpAtItsDeadline(t *testing.T) {
	old := remoteStart
	t.Cleanup(func() { remoteStart = old })
	var seen child.Spec
	remoteStart = func(s child.Spec) child.Result { seen = s; return child.Result{Code: -1, TimedOut: true} }
	if _, err := LsRemote(t.TempDir(), "origin"); err == nil {
		t.Fatal("want an error")
	}
	if seen.Timeout != RemoteTimeout || !slices.Contains(seen.Env, "GIT_TERMINAL_PROMPT=0") {
		t.Fatalf("spec %+v", seen)
	}
}
```

Die Helfer `mustGit` und `repoWithCommit` in `gitwork_test.go` anlegen, falls
es sie unter anderem Namen noch nicht gibt: `repoWithCommit` legt ein Repo mit
Identität per `-c user.name=t -c user.email=t@t` und einer Datei `a.txt` an.

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/gitwork/`
Erwartet: FAIL.

- [ ] **Schritt 3: Implementieren**

In `gitwork.go`:

```go
// EmptyTree is the tree of a commit that holds nothing; git knows it by this
// name in every repository, with or without a commit.
const EmptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// ErrNotRepository marks a root no working tree of git covers.
var ErrNotRepository = errors.New("not a git working tree")

// Head is the commit HEAD names, and "" without an error when the
// repository has no commit yet: that is a state to measure from, the empty
// tree, and not a failure.
func Head(root string) (string, error) {
	if ignored(root) {
		return "", fmt.Errorf("%s: %w", root, ErrIgnoredRoot)
	}
	if _, err := git(root, "rev-parse", "--is-inside-work-tree"); err != nil {
		return "", fmt.Errorf("%s: %w", root, ErrNotRepository)
	}
	// --verify -q exits 1 without a word for a HEAD that names no commit.
	out, err := git(root, "rev-parse", "-q", "--verify", "HEAD^{commit}")
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

// TreeOf is the tree a commit holds.
func TreeOf(root, commit string) (string, error) {
	out, err := git(root, "rev-parse", commit+"^{tree}")
	return strings.TrimSpace(out), err
}

// ContentTree is the tree the working tree would be if everything git does
// not ignore were committed now, loomux's own state left out.
//
// Written through a copy of the index, so the real one never changes: a
// copy and not an empty index, because git only re-hashes a file whose stat
// changed against the index it is given. Measured on 2026-09-19 on 7,322
// files: 224-245 ms. The same content is the same SHA whatever HEAD and the
// base are, which is what lets the stop gate remember a green tree.
func ContentTree(root, scratch string) (string, error) {
	out, err := git(root, "rev-parse", "--git-path", "index")
	if err != nil {
		return "", err
	}
	real := strings.TrimSpace(out)
	if !filepath.IsAbs(real) {
		real = filepath.Join(root, real)
	}
	if err := os.MkdirAll(scratch, 0o755); err != nil {
		return "", err
	}
	index := filepath.Join(scratch, fmt.Sprintf("index-%d", os.Getpid()))
	defer os.Remove(index)
	data, err := os.ReadFile(real)
	switch {
	case err == nil:
		if err := os.WriteFile(index, data, 0o644); err != nil {
			return "", err
		}
	case errors.Is(err, fs.ErrNotExist):
		// A repository without a commit may have no index yet; git makes one
		// where GIT_INDEX_FILE points.
	default:
		return "", err
	}
	env := []string{"GIT_INDEX_FILE=" + index}
	if _, err := gitWith(root, env, "add", "-A"); err != nil {
		return "", err
	}
	if _, err := gitWith(root, env, "rm", "-r", "-q", "--cached", "--ignore-unmatch", "--", ".loomux/state"); err != nil {
		return "", err
	}
	tree, err := gitWith(root, env, "write-tree")
	return strings.TrimSpace(tree), err
}

// LocalHeads maps every local branch, by its full ref, to its commit.
func LocalHeads(root string) (map[string]string, error) {
	out, err := git(root, "for-each-ref", "--format=%(objectname)%09%(refname)", "refs/heads")
	if err != nil {
		return nil, err
	}
	return parseRefs(out), nil
}

// LogOneline is `git log --oneline from..to`, one line per commit.
func LogOneline(root, from, to string) ([]string, error) {
	out, err := git(root, "log", "--oneline", from+".."+to)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimRight(line, "\r"); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

// parseRefs reads `<sha>\t<ref>` lines, the form of ls-remote and of the
// for-each-ref format above.
func parseRefs(out string) map[string]string {
	refs := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		sha, ref, ok := strings.Cut(strings.TrimRight(line, "\r"), "\t")
		if ok {
			refs[ref] = sha
		}
	}
	return refs
}
```

`git` wird zu einem Aufruf von `gitWith(root, nil, arguments...)`; `gitWith`
hängt `extra` hinter `gitenv.Environ()` an (ein späterer `KEY=VALUE` gewinnt
in `exec`). `GIT_INDEX_FILE` steht in `gitenv.Location` und wird deshalb erst
nach dem Strip gesetzt — das ist der Grund, warum es ein Parameter ist und
keine Umgebung des Aufrufers.

`remote.go`:

```go
package gitwork

import (
	"fmt"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/child"
)

// RemoteTimeout is how long a remote may take to answer. Ten seconds is long
// for ls-remote against a reachable host and short enough that a dead one
// costs nothing anybody notices -- subagent_stop.py's number.
const RemoteTimeout = 10 * time.Second

// remoteStart is the seam a test uses to stand in for a remote that hangs.
var remoteStart = child.Run

// LsRemote maps every ref the remote names to its commit, HEAD included.
//
// Through child, so a hung connection costs its deadline and no more. No
// credential prompt: a hook has no terminal to ask on, and a prompt would
// hold the call until the deadline. SSH is left as the user set it up -- an
// ssh without a terminal refuses rather than asks.
func LsRemote(root, remote string) (map[string]string, error) {
	res := remoteStart(child.Spec{
		Argv:    []string{"git", "ls-remote", remote},
		Dir:     root,
		Env:     []string{"GIT_TERMINAL_PROMPT=0"},
		Timeout: RemoteTimeout,
	})
	switch {
	case res.Err != nil:
		return nil, fmt.Errorf("git ls-remote %s: %w", remote, res.Err)
	case res.TimedOut:
		return nil, fmt.Errorf("git ls-remote %s: no answer within %s", remote, RemoteTimeout)
	case res.Code != 0:
		return nil, fmt.Errorf("git ls-remote %s: exit %d: %s", remote, res.Code, strings.TrimSpace(res.Stderr))
	}
	return parseRefs(res.Stdout), nil
}
```

Den Paketkommentar von `gitwork` um die neuen Fragen ergänzen (stop gate:
Inhaltsbaum; Subagenten-Hooks: Branches, Remote, Log).

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/gitwork/ -cover`
Erwartet: PASS, 100.0 %. Ein Arm, den kein Repo erreicht (etwa `MkdirAll` auf
`scratch` oder ein `ReadFile`-Fehler, der kein `NotExist` ist), bekommt einen
Test mit einer Datei an der Stelle des Verzeichnisses bzw. des Index; nur wo
auch das nicht geht, ein `//coverage:exempt` mit Grund.

- [ ] **Schritt 5: Commit**

```sh
git add internal/gitwork
git commit -m "feat(gitwork): measure the content tree, local branches and the remote"
```

---

### Task 8: `internal/hosts` — Agent-Felder und das Ereignis im Kontext

**Files:**
- Modify: `internal/hosts/hostio.go`, `claude.go`, `claude_test.go`, `hostio_test.go`
- Modify: `internal/hooks/hook_session_start.go` (Aufruf von `WriteContext`)

**Interfaces:**
- Produces:
  - `Payload.AgentID`, `Payload.AgentType` (string; fehlend oder falsch
    getypt: `""`)
  - `func WriteContext(host Host, event string, w io.Writer, lines []string) error`

- [ ] **Schritt 1: Tests schreiben**

```go
func TestReadClaudeTakesTheAgent(t *testing.T) {
	p, err := Read(HostClaude, strings.NewReader(`{"hook_event_name":"SubagentStop","session_id":"s","agent_id":"a","agent_type":"Explore"}`))
	if err != nil || p.AgentID != "a" || p.AgentType != "Explore" {
		t.Fatalf("%+v, %v", p, err)
	}
	p, _ = Read(HostClaude, strings.NewReader(`{"agent_id": 5}`))
	if p.AgentID != "" {
		t.Fatalf("a mistyped id reads as absent: %+v", p)
	}
}

func TestWriteContextNamesTheEvent(t *testing.T) {
	var out strings.Builder
	if err := WriteContext(HostClaude, "SessionStart", &out, []string{"x"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"hookEventName":"SessionStart"`) {
		t.Fatalf("%s", out.String())
	}
}
```

Die vorhandenen Aufrufe von `WriteContext` in den Tests bekommen das Ereignis.
Liegt ein Test für Task 1 vor (`testdata/cases/2c-payloads/claude-*.json`),
liest ein weiterer Test jede dieser Dateien über `Read(HostClaude, …)` und
prüft `SessionID == "s1"` und `AgentID == "a1"` (außer `claude-stop.json`).

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/hosts/`
Erwartet: FAIL.

- [ ] **Schritt 3: Implementieren**

`hostio.go`, in `Payload` nach `SessionID`:

```go
	// AgentID and AgentType name the subagent a SubagentStart or
	// SubagentStop fired for, "" on every other event. Claude Code sends both
	// with the main agent's session_id (measured in stage 2c, Task 1), which
	// is what files a subagent's snapshot under its session.
	AgentID   string
	AgentType string
```

Den Kommentar über `Payload` („Two fields, and no more“) auf „the fields the
hooks read“ berichtigen.

`WriteContext(host Host, event string, w io.Writer, lines []string)`; der
Claude-Arm ruft `writeClaudeContext(w, event, lines)`; `writeClaudeContext`
setzt `answer.HookSpecificOutput.HookEventName = event`. Die Arme für
Antigravity und Codex nehmen `event` an und ignorieren es, bis Task 15.

`claude.go`, in `readClaude`:

```go
	agentID, _ := payload["agent_id"].(string)
	agentType, _ := payload["agent_type"].(string)
	return Payload{Event: event, SessionID: sessionID, AgentID: agentID, AgentType: agentType}, nil
```

`hook_session_start.go`: `hosts.WriteContext(host, "SessionStart", stdout, lines)`.

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/hosts/ ./internal/hooks/ -cover`
Erwartet: PASS, `hosts` 100.0 %.

- [ ] **Schritt 5: Commit**

```sh
git add internal/hosts internal/hooks/hook_session_start.go
git commit -m "feat(hosts): read the subagent of a payload and name the event of an answer"
```

---

### Task 9: Profil `stop` und `lint/wiki` im Check-Scope

**Files:**
- Modify: `internal/verify/schema.go`, `schema_test.go`
- Create: `internal/hooks/wikigate.go`, `wikigate_test.go`
- Modify: `internal/cli/check.go`, `check_test.go`

**Interfaces:**
- Produces:
  - Profil `stop` in `defaults()` mit `{"lint", "types", "test", "coverage"}`
  - `func WikiGateJobs(eff verify.Effective, facts detect.Facts, root string, kinds []string) []verify.Job`
    — `facts` reicht der Aufrufer durch, der sie schon hat; ein zweites
    `detect.Detect` wäre ein zweiter Baumlauf

- [ ] **Schritt 1: Tests schreiben**

`schema_test.go`:

```go
func TestDefaultsHoldAStopProfile(t *testing.T) {
	cfg, err := ParseConfig("", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cfg.Profiles["stop"], []string{"lint", "types", "test", "coverage"}) {
		t.Fatalf("stop = %v", cfg.Profiles["stop"])
	}
}

// stop is a profile like edit and precommit: a project with a slow suite
// narrows it rather than living without the gate.
func TestAStopProfileCanBeNarrowed(t *testing.T) {
	cfg, err := ParseConfig("", map[string]any{"verify": map[string]any{"profiles": map[string]any{"stop": []any{"lint"}}}})
	if err != nil || !slices.Equal(cfg.Profiles["stop"], []string{"lint"}) {
		t.Fatalf("%v, %v", cfg.Profiles["stop"], err)
	}
}
```

`wikigate_test.go` (Welten wie in `post_edit_test.go` für die Wiki-Lane
anlegen; dort den Helfer für ein Wiki-Bündel lesen und wiederverwenden):

```go
func TestWikiGateJobsRunTheGateInProcess(t *testing.T) {
	root := wikiProject(t) // docs/wiki mit gültigem Bündel und [layout] wiki
	eff := resolvedFor(t, root)
	jobs := WikiGateJobs(eff, factsOf(root), root, []string{"lint", "test"})
	if len(jobs) != 1 || jobs[0].Name != "lint/wiki" || jobs[0].Fn == nil {
		t.Fatalf("jobs %+v", jobs)
	}
	if out, err := jobs[0].Fn(); err != nil {
		t.Fatalf("a valid bundle failed: %s %v", out, err)
	}
}

func TestWikiGateJobsReportAViolation(t *testing.T) {
	root := wikiProject(t)
	// Eine Seite mit einem toten Link, wie in den Tests von wiki.CheckWikiGate.
	breakBundle(t, root)
	out, err := WikiGateJobs(resolvedFor(t, root), factsOf(root), root, []string{"lint"})[0].Fn()
	if err == nil || !strings.Contains(out, "Violation") {
		t.Fatalf("out %q, err %v", out, err)
	}
}

func TestWikiGateJobsOnlyForLintAndAWiki(t *testing.T) {
	root := wikiProject(t)
	if jobs := WikiGateJobs(resolvedFor(t, root), factsOf(root), root, []string{"test"}); jobs != nil {
		t.Fatalf("jobs without lint: %+v", jobs)
	}
	plain := t.TempDir()
	if jobs := WikiGateJobs(resolvedFor(t, plain), factsOf(plain), plain, []string{"lint"}); jobs != nil {
		t.Fatalf("jobs without a wiki: %+v", jobs)
	}
	off := wikiProject(t)
	appendConfig(t, off, "[verify.wiki]\nlint = false\n")
	if jobs := WikiGateJobs(resolvedFor(t, off), factsOf(off), off, []string{"lint"}); jobs != nil {
		t.Fatalf("jobs with the lane off: %+v", jobs)
	}
}
```

`wikiProject`, `resolvedFor`, `factsOf`, `breakBundle` und `appendConfig`
sind Testhelfer in `wikigate_test.go`: `factsOf` ist
`detect.Detect(os.DirFS(root))`, `resolvedFor` ruft `editLoad(root,
factsOf(root))`. Wo `post_edit_test.go` schon gleichwertige
Helfer hat, diese benutzen statt neue zu schreiben.

`check_test.go`: Ein Fall mit einem Wiki-Projekt, in dem `loomux check lint`
eine Zeile `lint/wiki: ok [in-process]` druckt, und einer, in dem ein
kaputtes Bündel Exit 1 ergibt.

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/verify/ ./internal/hooks/ ./internal/cli/ -run 'Stop|WikiGate|Check'`
Erwartet: FAIL.

- [ ] **Schritt 3: Implementieren**

`schema.go`, in `defaults()`:

```go
		Profiles: map[string][]string{
			"edit":      {"lint", "types"},
			"precommit": {"lint", "types", "test", "coverage"},
			// What the stop gate runs at every turn end. The same four kinds
			// as precommit by default; a project whose suite is too slow for
			// every turn end narrows it here and keeps a gate that moves.
			"stop": {"lint", "types", "test", "coverage"},
		},
```

`internal/hooks/wikigate.go`:

```go
package hooks

import (
	"errors"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/brain/wiki"
	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/verify"
)

// WikiGateJobs is the wiki's lane for a whole project: the lint over the
// bundle and the drift check `loomux wiki-gate` runs, in this process. It
// runs where lint is asked for, the project has a wiki, and [verify.wiki]
// lint = false did not switch it off -- the same three conditions the edit
// lane has, with the whole bundle in place of the edited page.
//
// In `loomux check` and the stop gate alike: ultraloom ran the gate as a
// Stop entry of its own, beside the chain, with a count of its own; one lane
// in one chain is one verdict and one counter.
func WikiGateJobs(eff verify.Effective, facts detect.Facts, root string, kinds []string) []verify.Job {
	if !slices.Contains(kinds, "lint") || eff.Config.Stacks["wiki"]["lint"].Lane.Off {
		return nil
	}
	if !slices.Contains(stacksWithWiki(facts.Stacks, root), "wiki") {
		return nil
	}
	return []verify.Job{{
		Name: "lint/wiki", Kind: "lint", Stack: "wiki", Area: ".", Origin: "in-process", Dir: root, After: -1,
		Fn: func() (string, error) {
			var report strings.Builder
			if wiki.GateReport(root, &report, &report) != 0 {
				return report.String(), errors.New("the wiki gate found violations")
			}
			return "", nil
		},
	}}
}
```

`cli/check.go`: `checkLoad` gibt die `detect.Facts`, die es für `Resolve`
ohnehin erhebt, als vierten Wert zurück; in `checkRun` nach `checkPlan`:

```go
	// The wiki's lane is built by the hooks, like its edit lane; appended
	// behind the plan, because a job's After is an index into it.
	jobs = append(jobs, hooks.WikiGateJobs(eff, facts, root, kinds)...)
```

(`internal/hooks` importieren.) `--show` bleibt unverändert: Die Wiki-Lane hat
keine Befehle, die sich als TOML drucken ließen.

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/verify/ ./internal/hooks/ ./internal/cli/ -cover`
Erwartet: PASS, 100.0 %. Die 2a-Fallsuite bleibt grün (keine Welt hat ein
Wiki). `go run ./cmd/loomux check lint` in diesem Repo zeigt `lint/wiki: ok`.

- [ ] **Schritt 5: Commit**

```sh
git add internal/verify internal/hooks internal/cli
git commit -m "feat(check): add a stop profile and the wiki gate as a lint lane"
```

---

### Task 10: `internal/hooks/stop.go` — das Tor am Rundenende

**Files:**
- Create: `internal/hooks/stop.go`, `stop_test.go`

**Interfaces:**
- Consumes: `hosts.Read`, `Payload.SessionID` (Task 8); `sessions.*`
  (Task 4); `gitwork.Head`, `TreeOf`, `ContentTree`, `EmptyTree`,
  `ErrNotRepository`, `ErrIgnoredRoot` (Task 7); `WikiGateJobs`, Profil
  `stop` (Task 9); `editLoad` (`post_edit.go`)
- Produces:
  - `const DefaultStopBudget = 270 * time.Second`
  - `const MaxBlocks = 3`
  - `const NoVerifyMarker = ".loomux/no-verify"`
  - `type StopEnv struct { Start func(child.Spec) child.Result; Look func(string) (string, error); Loomux string; Budget time.Duration; Now func() time.Time; ImportReady func(dir string) bool }`
  - `func Stop(stdin io.Reader, stderr io.Writer, root, hostName string, budget time.Duration) int`
  - `func RunStop(stdin io.Reader, stderr io.Writer, root, hostName string, env StopEnv) int`

- [ ] **Schritt 1: Tests schreiben**

Ein Helfer baut Welten mit `cases.BuildGitWorld` (Task 2), damit die Tests
dieselben Repos sehen wie die Fallsuite; ein zweiter liefert einen `StopEnv`,
dessen `Start` aus einer Tabelle antwortet (Präfix der Kommandozeile → Exit),
und dessen `Look` jedes Werkzeug findet.

```go
package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/sessions"
)

const stopWorld = `
[[commit]]
message = "base"
paths = ["go.mod", "a.go", "a_test.go"]

[[commit]]
message = "second"
[commit.files]
"a.go" = "package a\n\nfunc A() int { return 2 }\n"

[worktree]
"a.go" = "package a\n\nfunc A() int { return 3 }\n"
`

// gitWorld stages a Go project with two commits and a change on top, and a
// session s1 whose base is the first commit.
func gitWorld(t *testing.T, decl string, state string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"git.toml":  decl,
		"go.mod":    "module a\n\ngo 1.25\n",
		"a.go":      "package a\n\nfunc A() int { return 1 }\n",
		"a_test.go": "package a\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) { A() }\n",
		".loomux/state/hooks/s1.json": state,
		// The fake go test writes no profile, and the coverage lane checks
		// that one is there; this is a test of the gate, not of gocover.
		".loomux/config.toml": "[verify.go]\ncoverage = false\n",
	}
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, []byte(body), 0o644)
	}
	if err := cases.BuildGitWorld(root); err != nil {
		t.Fatal(err)
	}
	return root
}

// fakeTools answers every process from its first two words; a line it does
// not know exits 0. {loomux} check gofmt and gocover answer 0 as well: this
// is a test of the gate, not of the tools.
func fakeTools(exits map[string]int) StopEnv {
	return StopEnv{
		Start: func(s child.Spec) child.Result {
			key := strings.Join(s.Argv[:min(2, len(s.Argv))], " ")
			return child.Result{Code: exits[key], Stdout: key + "\n"}
		},
		Look:   func(name string) (string, error) { return name, nil },
		Loomux: "loomux",
		Budget: DefaultStopBudget,
		Now:    time.Now,
	}
}

func runStop(t *testing.T, root, payload string, env StopEnv) (int, string) {
	t.Helper()
	var errOut strings.Builder
	code := RunStop(strings.NewReader(payload), &errOut, root, "claude", env)
	return code, errOut.String()
}

const s1 = `{"session_id":"s1","hook_event_name":"Stop"}`
```

Die Fälle, jeder ein eigener Test mit Namen nach dem Schritt der Spec:

| Test | Welt / Zustand | Erwartet |
|---|---|---|
| `TestStopRefusesAPayloadThatIsNoJSON` | beliebig, Nutzlast `nope` | Exit 1 |
| `TestStopRefusesAPayloadWithoutSession` | `{}` | Exit 1, stderr nennt `session_id` |
| `TestStopPassesAndMovesTheBase` | `stopWorld`, `{"base":"{{COMMIT:1}}","blocks":0}` | Exit 0; `base` = HEAD; `green` = `gitwork.ContentTree`; `blocks` 0 |
| `TestStopHoldsARedChainAndCounts` | wie oben, `go vet` Exit 1 | Exit 2; stderr enthält `lint/go: failed`; `blocks` 1; `base` unverändert |
| `TestStopShowsOnlyRedLanes` | wie oben | stderr enthält keine Zeile mit `: ok [` |
| `TestStopGivesUpAfterThreeInARow` | `blocks` 3, `go vet` Exit 1 | Exit 0; stderr `gave up after 3 consecutive blocks`; `blocks` 0; kein Werkzeug gestartet (Zähler in `Start`) |
| `TestStopResetsTheCounterWhenGreen` | `blocks` 2, alles grün | Exit 0; `blocks` 0 |
| `TestStopSkipsAGreenTree` | erst grüner Lauf, dann ohne Änderung ein zweiter mit `go vet` Exit 1 | zweiter Lauf Exit 0, kein Werkzeug gestartet |
| `TestStopSkipsATreeEqualToTheBase` | kein `[worktree]`, `base` = Commit 2 | Exit 0, kein Werkzeug |
| `TestStopSeesAnUntrackedFile` | kein `[worktree]`, `base` = Commit 2, neue Datei `b.go`, `go vet` Exit 1 | Exit 2 |
| `TestStopWithoutBaseMeasuresFromHead` | `{"base":null,"blocks":0}` | Exit 0; stderr warnt `no base commit`; `base` = HEAD |
| `TestStopOutsideARepositoryRunsTheChain` | Verzeichnis ohne `git.toml` | Kette läuft; Exit 0; `base` und `green` leer |
| `TestStopInAnUnbornRepository` | `git.toml` ohne `[[commit]]`, Dateien untracked | Kette läuft (Baum ≠ leerer Baum); Exit 0; `base` leer |
| `TestStopHoldsAGitFailure` | `stopTree` gibt einen Fehler | Exit 2; `blocks` 1 |
| `TestStopMeasuresFromHeadWhenTheBaseIsGone` | `base` = eine SHA, die es nicht gibt (`"0000…0001"`) | stderr `base … is gone`; Kette läuft; Exit 0; `base` = HEAD |
| `TestStopHoldsALoadErrorWhenItDeliveredFindings` | Befund plus kaputte Konfiguration | Exit 2; `blocks` 0 (Nachtrag 9) |
| `TestStopMarkerLetsTheTurnEnd` | `.loomux/no-verify` gesetzt, `go vet` Exit 1 | Exit 0, kein Werkzeug |
| `TestStopDeliversFindingsAndHolds` | Agent-Datei `a1` mit `finding: ["origin x is new at c"]`, alles grün | Exit 2; stderr beginnt mit `subagent a1: origin x is new at c`; Agent-Datei gelöscht; `blocks` 0 |
| `TestStopDeliversFindingsEvenWithTheMarker` | wie oben plus Marker | Exit 2, Befund auf stderr |
| `TestStopLeavesRunningSubagentsAlone` | Agent-Datei mit `snapshot` | Datei bleibt; Exit 0 |
| `TestStopReportsABadConfig` | `.loomux/config.toml` mit `[verify] nope = 1` | Exit 1 |
| `TestStopReportsABudgetThatRanOut` | `Budget: time.Second`; `Now` gibt beim ersten Aufruf `t0` und danach immer `t0 + 1h` zurück, so dass `verify.Run` die Frist vor dem ersten Start überschritten sieht | Exit 1; stderr `not everything was verified`; `base` unverändert |
| `TestStopReportsNothingVerified` | Welt ohne Go-Tests (`a_test.go` fehlt) | Exit 1; stderr `nothing to check for` `test`; `base` unverändert |
| `TestStopWithAnUnknownHost` | `hostName` `nope` | Exit 1 |
| `TestStopReportsAStateItCannotWrite` | `.loomux/state/hooks` ist eine Datei, alles grün | Exit 0; stderr nennt den Schreibfehler |

Für `TestStopHoldsAGitFailure` wird die Naht `stopTree` ersetzt; für den Plan-
Fehlerarm `stopPlan`. Beide wie `editPlan` in `post_edit.go`.

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/hooks/ -run Stop`
Erwartet: FAIL.

- [ ] **Schritt 3: Implementieren**

```go
package hooks

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/sessions"
	"github.com/xidus90/loomux/internal/verify"
)

// DefaultStopBudget is how long the gate's chain may take in all: under the
// 300 s its settings entry grants, with room for git and the report. The
// budget belongs to the scope, as it does for post-edit.
const DefaultStopBudget = 270 * time.Second

// MaxBlocks is how many turn ends in a row the gate holds before it lets one
// go. It guards against a loop, and a loop is blocks in a row: a green run in
// between ends the row.
const MaxBlocks = 3

// NoVerifyMarker is the file a human sets to let turns end unchecked. The
// write barrier keeps it out of an agent's reach (guard.go).
const NoVerifyMarker = ".loomux/no-verify"

// StopEnv is what a stop run needs from outside: what starts a tool and
// finds it, which binary {loomux} names, the budget, the clock, and whether
// Godot has imported a project. A nil ImportReady asks the disk.
type StopEnv struct {
	Start       func(child.Spec) child.Result
	Look        func(string) (string, error)
	Loomux      string
	Budget      time.Duration
	Now         func() time.Time
	ImportReady func(dir string) bool
}

// The seams no world can provoke: a plan that fails, and git failing on a
// repository it just answered about.
var (
	stopPlan = verify.Plan
	stopTree = gitwork.ContentTree
)

// errNoRepository says the root is outside every tree git measures; the gate
// then runs the chain every time.
var errNoRepository = errors.New("no repository")

// Stop checks that everything since the last green pass is green before a
// turn ends, with the real tools.
func Stop(stdin io.Reader, stderr io.Writer, root, hostName string, budget time.Duration) int {
	loomux, err := editExecutable()
	if err != nil {
		loomux = "loomux"
	}
	return RunStop(stdin, stderr, root, hostName, StopEnv{
		Start: child.Run, Look: exec.LookPath, Loomux: loomux, Budget: budget, Now: time.Now, ImportReady: verify.ImportReady,
	})
}

// RunStop is the stop gate: 0 lets the turn end, 2 holds it with the reason
// on stderr, 1 is a gate that could not judge and holds nothing. The order is
// the spec's: payload, findings of the subagents, marker, counter, tree,
// config, chain.
func RunStop(stdin io.Reader, stderr io.Writer, root, hostName string, env StopEnv) int {
	say := func(format string, a ...any) { fmt.Fprintf(stderr, "loomux hook stop: "+format+"\n", a...) }
	host, err := hosts.ParseHost(hostName)
	if err != nil {
		say("%v", err)
		return ExitInternal
	}
	payload, err := hosts.Read(host, stdin)
	if err != nil {
		say("%v", err)
		return ExitInternal
	}
	id := payload.SessionID
	if id == "" {
		// No shared fallback file: two sessions would count each other's
		// blocks (stop.py:102-109).
		say("payload carries no session_id")
		return ExitInternal
	}
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}

	// Before the marker: what a subagent did to the remote and the branches is
	// for the main agent to see whether or not anything is verified.
	delivered := deliverFindings(stderr, root, id)
	end := func(code int) int {
		if delivered && code != ExitDenied {
			return ExitDenied
		}
		return code
	}
	if _, err := os.Stat(filepath.Join(root, NoVerifyMarker)); err == nil {
		return end(ExitOK)
	}

	state := sessions.ReadState(root, id)
	save := func() {
		if err := sessions.WriteState(root, id, state); err != nil {
			say("%v", err)
		}
	}
	if state.Blocks >= MaxBlocks {
		say("gave up after %d consecutive blocks; base stays at %s. Fix the lanes or set %s.", MaxBlocks, shortSHA(state.Base), NoVerifyMarker)
		state.Blocks = 0
		save()
		return end(ExitOK)
	}

	head, tree, err := stopTrees(stderr, root, state)
	switch {
	case errors.Is(err, errNoRepository):
	case err != nil:
		say("%v", err)
		state.Blocks++
		save()
		return ExitDenied
	case tree == state.Green:
		return end(ExitOK)
	}

	facts := detect.Detect(os.DirFS(root))
	eff, err := editLoad(root, facts)
	if err != nil {
		say("%v", err)
		return end(ExitInternal)
	}
	// Built in: a config can replace the stop profile but not remove it.
	kinds, _ := verify.ExpandProfile(eff.Config, "stop")
	runID := verify.NewRunID(env.Now(), os.Getpid())
	ready := env.ImportReady
	if ready == nil {
		ready = verify.ImportReady
	}
	jobs, err := stopPlan(eff, verify.Request{Kinds: kinds, Scope: verify.ScopeCheck}, verify.PlanEnv{
		Root: root, Loomux: env.Loomux, RunID: runID, HasTests: verify.HasTests, ImportReady: ready,
	})
	if err != nil {
		say("%v", err)
		return end(ExitInternal)
	}
	jobs = append(jobs, WikiGateJobs(eff, facts, root, kinds)...)
	if err := verify.PrepareCover(root); err != nil {
		say("%v", err)
		return end(ExitInternal)
	}
	outs := verify.Run(jobs, verify.RunOptions{
		Scope: verify.ScopeCheck, MaxParallel: eff.Config.MaxParallel, Timeout: eff.Config.Timeout,
		Budget: env.Budget, Start: env.Start, Look: env.Look, Now: env.Now,
	})
	code := stopVerdict(stderr, kinds, outs)
	if err := verify.CleanCover(root, runID, code == ExitOK); err != nil {
		say("cleaning coverage files: %v", err)
	}
	switch code {
	case ExitDenied:
		state.Blocks++
		save()
		return ExitDenied
	case ExitOK:
		state.Base, state.Green, state.Blocks = head, tree, 0
		save()
	}
	return end(code)
}

// stopTrees answers where HEAD stands and what tree the work is, or
// errNoRepository. A tree equal to the base's is nothing new; it is returned
// as the green tree's twin so the caller has one comparison to make.
func stopTrees(stderr io.Writer, root string, state sessions.SessionState) (head, tree string, err error) {
	head, err = gitwork.Head(root)
	if errors.Is(err, gitwork.ErrNotRepository) || errors.Is(err, gitwork.ErrIgnoredRoot) {
		return "", "", errNoRepository
	}
	if err != nil {
		return "", "", err
	}
	base := state.Base
	if base == "" && head != "" {
		fmt.Fprintf(stderr, "loomux hook stop: no base commit for this session; measuring from HEAD, so what this session committed stays unseen\n")
		base = head
	}
	baseTree := gitwork.EmptyTree
	if base != "" {
		if baseTree, err = gitwork.TreeOf(root, base); err != nil {
			// A base that amend, rebase and gc took away is no git failure
			// to hold the turn over: measure from HEAD, and the next green
			// run sets a new base (plan, Nachtrag 8).
			fmt.Fprintf(stderr, "loomux hook stop: base %s is gone; measuring from HEAD\n", shortSHA(base))
			baseTree = gitwork.EmptyTree
			if head != "" {
				if baseTree, err = gitwork.TreeOf(root, head); err != nil {
					return "", "", err
				}
			}
		}
	}
	// The copied index goes to the system's temp directory, not to the state:
	// a state directory that cannot be written must not become a git failure
	// that holds every turn.
	tree, err = stopTree(root, os.TempDir())
	if err != nil {
		return "", "", err
	}
	if tree == baseTree {
		// Nothing since the base: the same answer as a green tree.
		return head, state.Green, nil
	}
	return head, tree, nil
}
```

Achtung beim letzten Arm: `tree == baseTree` gibt `state.Green` als `tree`
zurück, damit der Aufrufer mit `tree == state.Green` endet. Ist `state.Green`
leer und der Baum gleich der Basis, wäre `tree == ""`; der Aufrufer vergleicht
`"" == ""` und endet ebenfalls — gewollt. Einen eigenen Test dafür gibt es
(`TestStopSkipsATreeEqualToTheBase`).

```go
// stopVerdict writes the red lanes and says what they mean for the turn:
// red holds it, a budget that ran out or a kind with nothing to check leaves
// it unjudged, anything else passes.
func stopVerdict(stderr io.Writer, kinds []string, outs []verify.Outcome) int {
	var red []verify.Outcome
	for _, o := range outs {
		if verify.Red(o.State, verify.ScopeCheck) {
			red = append(red, o)
		}
	}
	if len(red) > 0 {
		// Only the red lanes: what reaches the agent's context is what it
		// has to fix.
		verify.WriteCheck(stderr, red, false)
		return ExitDenied
	}
	if slices.ContainsFunc(outs, func(o verify.Outcome) bool { return o.State == verify.StateBudget }) {
		fmt.Fprintln(stderr, "loomux hook stop: not everything was verified; raise --budget or shrink the stop profile")
		return ExitInternal
	}
	if _, notes := verify.CheckVerdict(kinds, outs); len(notes) > 0 {
		for _, note := range notes {
			fmt.Fprintf(stderr, "loomux hook stop: %s\n", note)
		}
		fmt.Fprintln(stderr, "loomux hook stop: nothing was verified for these kinds; the base stays")
		return ExitInternal
	}
	return ExitOK
}

// deliverFindings writes what stopped subagents left and removes their files
// once written. It reports whether there was anything, which holds the turn.
func deliverFindings(stderr io.Writer, root, sessionID string) bool {
	found := sessions.Findings(root, sessionID)
	for _, f := range found {
		for _, line := range f.Lines {
			fmt.Fprintf(stderr, "subagent %s: %s\n", f.AgentID, line)
		}
	}
	for _, f := range found {
		if err := sessions.RemoveAgent(root, sessionID, f.AgentID); err != nil {
			fmt.Fprintf(stderr, "loomux hook stop: %v\n", err)
		}
	}
	return len(found) > 0
}

func shortSHA(sha string) string {
	if sha == "" {
		return "no commit"
	}
	return sha[:min(12, len(sha))]
}
```

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/hooks/ -cover`
Erwartet: PASS, 100.0 %.

- [ ] **Schritt 5: Commit**

```sh
git add internal/hooks
git commit -m "feat(hooks): gate the turn end with the stop profile"
```

---

### Task 11: `internal/hooks/subagent.go` — Snapshot und Befund

**Files:**
- Create: `internal/hooks/subagent.go`, `subagent_test.go`

**Interfaces:**
- Consumes: `Payload.AgentID` (Task 8); `sessions.Snapshot`, `AgentFile`,
  `ReadAgent`, `WriteAgent`, `RemoveAgent` (Task 4); `gitwork.LsRemote`,
  `LocalHeads`, `Head`, `LogOneline` (Task 7)
- Produces:
  - `func SubagentStart(stdin io.Reader, stderr io.Writer, root, hostName string) int`
  - `func SubagentStop(stdin io.Reader, stderr io.Writer, root, hostName string) int`

- [ ] **Schritt 1: Tests schreiben**

Welten über `cases.BuildGitWorld` mit `[remote.push]`; die Änderungen zwischen
Start und Stop macht der Test selbst mit `git` (commit, push, branch, push
`:refs/heads/x` zum Löschen).

| Test | zwischen Start und Stop | Befund (Zeilen ohne Präfix) |
|---|---|---|
| `TestSubagentStartRefusesWithoutAgent` | — (Nutzlast ohne `agent_id`) | Exit 1, keine Datei |
| `TestSubagentStartRecordsTheSnapshot` | — | Exit 0; `snapshot.refs` enthält `HEAD` und `refs/heads/master`; `heads` ≠ nil; `remote` `ok` |
| `TestSubagentStopWithoutSnapshotIsSilent` | — | Exit 0; stderr leer; keine Datei |
| `TestSubagentStopWithNothingChangedRemovesTheFile` | nichts | Exit 0; Datei weg |
| `TestSubagentStopSeesAPush` | Commit, `push origin HEAD:master` | `origin HEAD moved <a> -> <b>`, `origin refs/heads/master moved <a> -> <b>`, `branch master moved <a> -> <b>`, `new commit <kurz> third` |
| `TestSubagentStopSeesANewAndAGoneRef` | `push origin HEAD:refs/heads/feature`, `push origin :refs/heads/old` | `origin refs/heads/feature is new at <h>`, `origin refs/heads/old is gone; it was <h>` |
| `TestSubagentStopSeesALocalBranch` | `branch work`, Commit auf `work` über `git commit` im selben Repo nach `switch work`, zurück `switch master` | `branch work is new at <h>` |
| `TestSubagentStopCountsACommitOnce` | Commit auf master (HEAD und Branch bewegen sich gleich) | genau eine `new commit`-Zeile |
| `TestSubagentStopWithARemoteGoneAtStop` | `remote remove origin` | `remote could not be read at stop`, keine `origin`-Zeile |
| `TestSubagentStopWithARemoteMissingAtStart` | Snapshot mit `remote: unavailable` von Hand | `remote could not be read at start` |
| `TestSubagentStopWithATranslatedSnapshot` | Snapshot mit `heads: null` | keine `branch`-Zeile |
| `TestSubagentHooksWithAnUnknownHost` | `hostName` `nope` | Exit 1 |
| `TestSubagentStopReportsAFileItCannotWrite` | `agents/` ist nach dem Start eine Datei | Exit 1 |

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/hooks/ -run Subagent`
Erwartet: FAIL.

- [ ] **Schritt 3: Implementieren**

```go
package hooks

import (
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/sessions"
)

// remoteRefs is the seam for a remote that answers at one end and not the
// other.
var remoteRefs = gitwork.LsRemote

// SubagentStart writes down where origin, the local branches and HEAD stand
// before a subagent runs. It never blocks: 0, or 1 for a payload it cannot
// file.
func SubagentStart(stdin io.Reader, stderr io.Writer, root, hostName string) int {
	payload, root, ok := subagentPayload(stdin, stderr, root, hostName, "subagent-start")
	if !ok {
		return ExitInternal
	}
	snap := takeSnapshot(root)
	if err := sessions.WriteAgent(root, payload.SessionID, payload.AgentID, sessions.AgentFile{Snapshot: &snap}); err != nil {
		fmt.Fprintf(stderr, "loomux hook subagent-start: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

// SubagentStop compares against the snapshot and leaves what moved for the
// main agent's stop gate. Its own stdout and exit 2 would reach the subagent,
// not the main agent -- the only one who should hear about it.
func SubagentStop(stdin io.Reader, stderr io.Writer, root, hostName string) int {
	payload, root, ok := subagentPayload(stdin, stderr, root, hostName, "subagent-stop")
	if !ok {
		return ExitInternal
	}
	before, found := sessions.ReadAgent(root, payload.SessionID, payload.AgentID)
	if !found || before.Snapshot == nil {
		// Silent: this hook may have been switched on midway, and nobody
		// would have read the line Python printed here.
		return ExitOK
	}
	lines := compareSnapshots(root, *before.Snapshot, takeSnapshot(root))
	var err error
	if len(lines) == 0 {
		err = sessions.RemoveAgent(root, payload.SessionID, payload.AgentID)
	} else {
		err = sessions.WriteAgent(root, payload.SessionID, payload.AgentID, sessions.AgentFile{Finding: lines})
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux hook subagent-stop: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

func subagentPayload(stdin io.Reader, stderr io.Writer, root, hostName, event string) (hosts.Payload, string, bool) {
	host, err := hosts.ParseHost(hostName)
	if err == nil {
		var p hosts.Payload
		if p, err = hosts.Read(host, stdin); err == nil {
			if p.SessionID == "" || p.AgentID == "" {
				fmt.Fprintf(stderr, "loomux hook %s: payload carries no agent_id\n", event)
				return hosts.Payload{}, "", false
			}
			if abs, err := filepath.Abs(root); err == nil {
				root = abs
			}
			return p, root, true
		}
	}
	fmt.Fprintf(stderr, "loomux hook %s: %v\n", event, err)
	return hosts.Payload{}, "", false
}

// takeSnapshot never fails: an unreachable remote is a fact about the
// machine, not a finding about the subagent, and it is written down as such.
func takeSnapshot(root string) sessions.Snapshot {
	snap := sessions.Snapshot{Remote: sessions.RemoteOK, Heads: map[string]string{}}
	if refs, err := remoteRefs(root, "origin"); err == nil {
		snap.Refs = refs
	} else {
		snap.Remote = sessions.RemoteUnavailable
	}
	if heads, err := gitwork.LocalHeads(root); err == nil {
		snap.Heads = heads
	}
	snap.Head, _ = gitwork.Head(root)
	return snap
}

// compareSnapshots names every ref of origin and every local branch that
// moved, appeared or vanished, then the commits HEAD and the moved branches
// gained, each once.
func compareSnapshots(root string, before, after sessions.Snapshot) []string {
	var lines []string
	switch {
	case before.Remote != sessions.RemoteOK:
		lines = append(lines, "remote could not be read at start")
	case after.Remote != sessions.RemoteOK:
		lines = append(lines, "remote could not be read at stop")
	default:
		lines = append(lines, refLines("origin ", before.Refs, after.Refs, nil)...)
	}
	short := func(ref string) string { return strings.TrimPrefix(ref, "refs/heads/") }
	if before.Heads != nil {
		lines = append(lines, refLines("branch ", before.Heads, after.Heads, short)...)
	}
	var ranges [][2]string
	if before.Head != "" && after.Head != "" && before.Head != after.Head {
		ranges = append(ranges, [2]string{before.Head, after.Head})
	}
	if before.Heads != nil {
		for _, ref := range slices.Sorted(maps.Keys(before.Heads)) {
			was, now := before.Heads[ref], after.Heads[ref]
			if now != "" && was != now {
				ranges = append(ranges, [2]string{was, now})
			}
		}
	}
	seen := map[string]bool{}
	for _, r := range ranges {
		commits, _ := gitwork.LogOneline(root, r[0], r[1])
		for _, c := range commits {
			if !seen[c] {
				seen[c] = true
				lines = append(lines, "new commit "+c)
			}
		}
	}
	return lines
}

// refLines compares two ref tables in sorted order, both directions: a ref
// deleted is as much a push as one created.
func refLines(label string, old, now map[string]string, name func(string) string) []string {
	if name == nil {
		name = func(ref string) string { return ref }
	}
	all := map[string]bool{}
	for ref := range old {
		all[ref] = true
	}
	for ref := range now {
		all[ref] = true
	}
	var lines []string
	for _, ref := range slices.Sorted(maps.Keys(all)) {
		was, is := old[ref], now[ref]
		switch {
		case was == is:
		case was == "":
			lines = append(lines, fmt.Sprintf("%s%s is new at %s", label, name(ref), is))
		case is == "":
			lines = append(lines, fmt.Sprintf("%s%s is gone; it was %s", label, name(ref), was))
		default:
			lines = append(lines, fmt.Sprintf("%s%s moved %s -> %s", label, name(ref), was, is))
		}
	}
	return lines
}
```

Das Wort `origin ` ist hier das Label und kein Name des Remotes: loomux fragt
nur `origin` (Spec, Entscheidungen), wie Python.

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/hooks/ -cover`
Erwartet: PASS, 100.0 %.

- [ ] **Schritt 5: Commit**

```sh
git add internal/hooks
git commit -m "feat(hooks): snapshot origin and the branches around a subagent"
```

---

### Task 12: Verteiler, Marker-Regel, `status`

**Files:**
- Modify: `internal/cli/hook.go`, `hook_test.go`
- Modify: `internal/hooks/guard.go`, `guard_test.go`
- Modify: `internal/hooks/status.go`, `status_test.go`

**Interfaces:**
- Consumes: `hooks.Stop`, `SubagentStart`, `SubagentStop`, `DefaultStopBudget`,
  `NoVerifyMarker`

- [ ] **Schritt 1: Tests schreiben**

`hook_test.go`: `TestHookRefusesAnUnknownEvent` fragt `hook nope` statt
`hook stop`. Neu:

```go
func TestHookRoutesTheStopGate(t *testing.T) {
	old := stopHook
	t.Cleanup(func() { stopHook = old })
	var budget time.Duration
	stopHook = func(_ io.Reader, _ io.Writer, _, _ string, b time.Duration) int { budget = b; return 2 }
	code, _, _ := run("hook", "stop", "--host", "claude", "--root", t.TempDir(), "--budget", "5s")
	if code != 2 || budget != 5*time.Second {
		t.Fatalf("code %d, budget %s", code, budget)
	}
}

func TestHookStopDefaultsItsBudget(t *testing.T) {
	old := stopHook
	t.Cleanup(func() { stopHook = old })
	var budget time.Duration
	stopHook = func(_ io.Reader, _ io.Writer, _, _ string, b time.Duration) int { budget = b; return 0 }
	run("hook", "stop", "--host", "claude", "--root", t.TempDir())
	if budget != hooks.DefaultStopBudget {
		t.Fatalf("budget %s", budget)
	}
}

// A malformed call ends the turn: 1, never the 2 that would hold it.
func TestHookStopWithoutHostEndsTheTurn(t *testing.T) {
	if code, _, _ := run("hook", "stop", "--root", t.TempDir()); code != 1 {
		t.Fatalf("code %d", code)
	}
}

func TestHookRoutesTheSubagentHooks(t *testing.T) {
	for _, event := range []string{"subagent-start", "subagent-stop"} {
		code, _, errOut := run("hook", event, "--host", "claude", "--root", t.TempDir())
		// Empty stdin is no JSON: the hook itself answered, with its own name.
		if code != 1 || !strings.Contains(errOut, "loomux hook "+event) {
			t.Fatalf("%s: code %d, err %q", event, code, errOut)
		}
	}
}
```

`guard_test.go`, Zeile 38: `".loomux/no-verify"` statt
`".claude/.no-verify"`.

`status_test.go`: Die Fixture mit `Stop`-Eintrag (`status_test.go:226`) prüft
jetzt `[OK] Stop: 'loomux hook stop' installed`, eine ohne ihn `[INFO] Stop:`;
dasselbe für `SubagentStart` und `SubagentStop`. Der Abschnitt `[Stop]` nennt
`loomux hook stop` auch ohne Wiki.

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/cli/ ./internal/hooks/ -run 'Hook|Guard|Status'`
Erwartet: FAIL.

- [ ] **Schritt 3: Implementieren**

`cli/hook.go`:

```go
// Exit codes when a hook call is malformed: a write barrier refuses (2);
// everything else ends with 1, which blocks nothing -- a stop gate that
// cannot read its call must not hold the turn over it.
var malformed = map[string]int{
	"pre-tool-use":   hooks.ExitDenied,
	"post-tool-use":  hooks.ExitInternal,
	"session-start":  hooks.ExitInternal,
	"stop":           hooks.ExitInternal,
	"subagent-start": hooks.ExitInternal,
	"subagent-stop":  hooks.ExitInternal,
}

// The seams a test uses to see what the flags handed on.
var (
	postToolUse = hooks.PostToolUse
	stopHook    = hooks.Stop
)
```

Das Budget:

```go
	// post-edit and the stop gate run lanes, so only they have a budget.
	budget := new(time.Duration)
	switch event {
	case "post-tool-use":
		budget = flags.Duration("budget", hooks.DefaultBudget, "how long the post-edit lanes may take in all")
	case "stop":
		budget = flags.Duration("budget", hooks.DefaultStopBudget, "how long the stop gate's lanes may take in all")
	}
```

Der Verteiler:

```go
	switch event {
	case "pre-tool-use":
		return hooks.PreToolUse(stdin, stdout, stderr, resolved, config.StateDir())
	case "post-tool-use":
		return postToolUse(stdin, stdout, stderr, resolved, *budget)
	case "stop":
		return stopHook(stdin, stderr, resolved, *host, *budget)
	case "subagent-start":
		return hooks.SubagentStart(stdin, stderr, resolved, *host)
	case "subagent-stop":
		return hooks.SubagentStop(stdin, stderr, resolved, *host)
	default:
		return hooks.SessionStart(stdin, stdout, stderr, resolved, *host)
	}
```

`guard.go:41`:

```go
	{Match: []string{".loomux/no-verify"}, Reason: "the stop gate's own controls are not written by the party it gates"},
```

`status.go`: `auditSettings` gibt statt `hasPre, hasPost` eine Menge zurück:

```go
// installedHooks are the loomux hook events a settings file wires, by the
// subcommand the command line names.
func auditSettings(root string) ([]LegacyFinding, map[string]bool)
```

Innen: für jeden Befehl, der `loomux` enthält, das Wort nach `hook` lesen und
eintragen:

```go
// hookEvent is the subcommand after `hook`; a word boundary on both sides, so
// `hook stop` is not found inside `hook subagent-stop`.
var hookEvent = regexp.MustCompile(`\bhook\s+([a-z-]+)`)
```

(`regexp.MustCompile` in einer Funktion oder als `sync.OnceValue`, nicht als
Paketvariable, die beim Start kompiliert: Startzeit-Regel.) Die Ausgabe druckt
je Ereignis eine Zeile in fester Reihenfolge:

```go
	for _, h := range []struct{ event, name string }{
		{"PreToolUse", "pre-tool-use"}, {"PostToolUse", "post-tool-use"}, {"SessionStart", "session-start"},
		{"Stop", "stop"}, {"SubagentStart", "subagent-start"}, {"SubagentStop", "subagent-stop"},
	} {
		if installed[h.name] {
			fmt.Fprintf(stdout, " [OK] %s: 'loomux hook %s' installed\n", h.event, h.name)
		} else {
			fmt.Fprintf(stdout, " [INFO] %s: 'loomux hook %s' not found in .claude/settings.json\n", h.event, h.name)
		}
	}
```

Der Abschnitt `[Stop]`:

```go
	fmt.Fprintln(stdout, "\n[Stop] (Session End Gate)")
	fmt.Fprintln(stdout, "  -> loomux hook stop --host claude --root \"${CLAUDE_PROJECT_DIR}\" (profile `stop`, the wiki gate as lint/wiki)")
```

`wiki_gate.py` in `knownLegacyHooks` (`status.go:28`) begründet jetzt „superseded
by the lint/wiki lane of 'loomux hook stop'“. `knownLegacyHooks` bekommt drei
Einträge für die alten Sitzungshooks — `ultraloom hook stop`,
`ultraloom hook subagent-start`, `ultraloom hook subagent-stop` —, jeweils mit
„superseded by 'loomux hook <event>'“; vorher lesen, wie die Tabelle ihre
Einträge erkennt (`scriptName` als Teilstring), und den Teilstring so wählen,
dass `loomux hook stop` nicht getroffen wird.

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/cli/ ./internal/hooks/ -cover`
Erwartet: PASS, 100.0 %.

- [ ] **Schritt 5: Commit**

```sh
git add internal/cli internal/hooks
git commit -m "feat(hooks): route stop and the subagent hooks and move the no-verify marker"
```

---

### Task 13: Fallsuite 2c

**Files:**
- Create: `internal/cli/cases_2c_test.go`

**Interfaces:**
- Consumes: der Korpus aus Task 6; `useFakeTools` aus `cases_2a_test.go`
  (dessen Muster); `stopHook`-Naht aus Task 12.

- [ ] **Schritt 1: Suite schreiben**

```go
package cli

import (
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/hooks"
)

// wantCases2c is pinned, not merely non-zero: a partial import must not pass
// as parity. Raise it with the corpus when a case is added.
const wantCases2c = 15

// approved2c names every case whose replay differs from its recording, with
// the number of the deviation in docs/.superpowers/parity/stufe-2c.md that
// explains it. A case missing here must pass; a case listed here must fail.
var approved2c = map[string]string{
	"hook-stop/gave-up":            "2: the counter counts blocks in a row and giving up resets it to 0; Python kept 3",
	"hook-stop/untracked-only":     "16: a new file is a change; Python's git diff did not see it and let the turn end unchecked",
	"hook-subagent-stop/no-snapshot": "9: silent without a snapshot; Python printed a line nobody read",
}

// TestCases2c replays the recordings of ultraloom's session hooks against
// loomux hook. The tools of the stop gate answer from the world's fixture,
// at the seam the gate starts its processes through.
func TestCases2c(t *testing.T) {
	all, err := cases.DiscoverCases(filepath.Join("..", "..", "testdata", "cases", "2c"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != wantCases2c {
		t.Fatalf("found %d cases, want %d", len(all), wantCases2c)
	}
	for _, c := range all {
		t.Run(c.Verb+"/"+c.Name, func(t *testing.T) {
			outcome, err := cases.RunCase(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Chdir(dir)
				t.Setenv("LOOMUX_STATE_DIR", dir)
				useFakeStopTools(t, dir)
				return Run(args, stdin, stdout, stderr)
			})
			if err != nil {
				t.Fatal(err)
			}
			reason, approved := approved2c[c.Verb+"/"+c.Name]
			switch {
			case approved && outcome.Passed:
				t.Fatalf("listed as a deviation (%s) but passes; remove it from approved2c and parity/stufe-2c.md", reason)
			case !approved && !outcome.Passed:
				t.Fatalf("%s\nstderr:\n%s", strings.Join(outcome.Mismatches, "\n"), outcome.ActualStderr)
			}
		})
	}
}

// useFakeStopTools points the stop gate at the world's fixture, the way
// useFakeTools points check at it, for as long as the case runs.
func useFakeStopTools(t *testing.T, dir string) {
	t.Helper()
	useFakeTools(t, dir) // sets checkStart, checkLook, checkExecutable
	old := stopHook
	t.Cleanup(func() { stopHook = old })
	stopHook = func(stdin io.Reader, stderr io.Writer, root, host string, budget time.Duration) int {
		return hooks.RunStop(stdin, stderr, root, host, hooks.StopEnv{
			Start: checkStart, Look: checkLook, Loomux: fakeSelf, Budget: budget, Now: time.Now,
		})
	}
}
```

Eine Welt ohne `faketool.json` (die Subagenten-Welten) lässt `useFakeTools`
scheitern; `useFakeStopTools` ruft `useFakeTools` deshalb nur, wenn
`faketool.json` in `dir` liegt.

- [ ] **Schritt 2: Laufen lassen**

Run: `go test ./internal/cli/ -run TestCases2c -count=1 -v`

Jeder Fall, der scheitert und nicht in `approved2c` steht, ist ein Fehler in
Tasks 10–11 **oder** eine Abweichung, die die Spec nicht kennt. Den Unterschied
an der Aufzeichnung (`stderr` im Quellfall) festmachen. Eine unbekannte
Abweichung: Halt, dem Menschen vorlegen; nach seiner Entscheidung entweder
Code ändern oder in `approved2c` und in die Abweichungsliste (Task 17).

- [ ] **Schritt 3: Commit**

```sh
git add internal/cli/cases_2c_test.go
git commit -m "test(cli): replay the stage 2c recordings"
```

---

### Task 14: **Halt** — Messung bei Antigravity

Der Mensch sitzt an einer interaktiven `agy`-Sitzung; der Agent bereitet vor
und wertet aus.

**Files:**
- Create: `testdata/cases/2c-payloads/agy-*.json`, Nachtrag in
  `docs/.superpowers/specs-ul/2026-09-10-antigravity-hook-messung.md`

- [ ] **Schritt 1: Rekorder vorlegen**

Wörtlich dem Menschen geben, zum Eintragen in `.agents/hooks.json` eines
vertrauten Testordners (nicht in loomux):

```json
{
  "PreToolUse": [{"matcher": "invoke_subagent", "hooks": [{"type": "command",
    "command": "uv run --no-project python -c \"import sys,uuid; open('probe-pre-'+uuid.uuid4().hex+'.json','w',encoding='utf-8').write(sys.stdin.read())\""}]}],
  "PostToolUse": [{"matcher": "invoke_subagent", "hooks": [{"type": "command",
    "command": "uv run --no-project python -c \"import sys,uuid; open('probe-post-'+uuid.uuid4().hex+'.json','w',encoding='utf-8').write(sys.stdin.read())\""}]}],
  "PreInvocation": [{"type": "command",
    "command": "uv run --no-project python -c \"import sys,uuid; open('probe-inv-'+uuid.uuid4().hex+'.json','w',encoding='utf-8').write(sys.stdin.read()); print('{\\\"hookSpecificOutput\\\":{\\\"additionalContext\\\":\\\"PROBE-CONTEXT-7\\\"}}')\""}],
  "Stop": [{"type": "command",
    "command": "uv run --no-project python -c \"import os,sys,uuid; open('probe-stop-'+uuid.uuid4().hex+'.json','w',encoding='utf-8').write(sys.stdin.read()); held=os.path.exists('probe-held'); open('probe-held','a').close(); held or (sys.stderr.write('PROBE-HOLD: say the word PROBE-SEEN\\n'), sys.exit(2))\""}]
}
```

Die Dateien landen in `.agents/` (Arbeitsverzeichnis der Hooks, Befund 2 der
Messung vom 2026-09-10). In der Sitzung: eine Aufgabe geben, die einen
Subagenten startet, und die Runde enden lassen.

- [ ] **Schritt 2: Die vier Fragen beantworten**

1. **Hält Exit 2 bei `Stop` die Runde an?** Ja, wenn die Runde weiterging und
   das Modell `PROBE-SEEN` sagte. Dann ist stderr der Kanal.
2. **Kann `PreInvocation` Kontext einspeisen?** Ja, wenn das Modell auf Frage
   „was steht in deinem Kontext über PROBE?“ `PROBE-CONTEXT-7` nennt.
3. **Nutzlast und ID bei `invoke_subagent`:** aus `probe-pre-*` und
   `probe-post-*` die Felder lesen; gibt es eine ID, die in beiden gleich ist?
4. **Eigene Frist bei `Stop`?** Nur messbar mit einem Hook, der schläft: den
   `Stop`-Befehl einmal durch `Start-Sleep 400` ersetzen und die Zeit bis zum
   Abbruch nehmen, falls der Mensch das will; sonst „nicht gemessen“.

- [ ] **Schritt 3: Ablegen**

Die Nutzlasten nach `testdata/cases/2c-payloads/agy-{pre,post,inv,stop}.json`,
Pfade und IDs gekürzt wie in Task 1. Den Messbericht der Spec vom 2026-09-10
um einen datierten Abschnitt „Nachmessung 2c“ ergänzen: agy-Version, die vier
Antworten, was offen bleibt.

- [ ] **Schritt 4: Commit**

```sh
git add testdata/cases/2c-payloads docs/.superpowers/specs-ul/2026-09-10-antigravity-hook-messung.md
git commit -m "docs(spec): measure the Antigravity stop and subagent hooks"
```

---

### Task 15: Antigravity-Adapter

**Files:**
- Create: `internal/hosts/antigravity.go`, `antigravity_test.go`
- Modify: `internal/hosts/codex.go` (die Antigravity-Stubs ziehen aus)

**Interfaces:**
- Consumes: die Nutzlasten aus Task 14
- Produces: `readAntigravity` füllt `Payload` aus einer gemessenen Nutzlast;
  `writeAntigravityContext` schreibt, wenn Frage 2 ja ergab, sonst bleibt
  `ErrNoAdapter`.

- [ ] **Schritt 1: Tests aus den Messungen**

```go
func TestReadAntigravityFromTheMeasuredPayloads(t *testing.T) {
	for _, name := range []string{"agy-stop.json", "agy-pre.json", "agy-post.json", "agy-inv.json"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "cases", "2c-payloads", name))
		if err != nil {
			t.Skipf("%s was not measured", name)
		}
		p, err := Read(HostAntigravity, bytes.NewReader(raw))
		if err != nil || p.SessionID != "s1" {
			t.Fatalf("%s: %+v, %v", name, p, err)
		}
	}
}
```

Für `agy-pre.json` und `agy-post.json` zusätzlich `AgentID == "a1"`, wenn
Frage 3 eine gemeinsame ID ergab. Ein `t.Skipf` ist nur für eine Datei
zulässig, die Task 14 ausdrücklich als nicht gemessen vermerkt hat.

- [ ] **Schritt 2: Implementieren**

`readAntigravity` in `antigravity.go` liest die Nutzlast wie `readClaude` als
Objekt (dieselben zwei Verweigerungen, `errNotAnObject` wiederverwenden) und
nimmt die Felder unter den Namen, die Task 14 gefunden hat. Die Namen stehen in
einer Tabelle am Kopf der Datei, mit Datum und agy-Version der Messung:

```go
// antigravityFields names where agy puts what Payload holds, as measured in
// stage 2c (testdata/cases/2c-payloads/agy-*.json, agy <version>, <date>).
var antigravityFields = struct{ event, session, agent, agentType string }{
	event:     "<Feldname aus der Messung>",
	session:   "<Feldname aus der Messung>",
	agent:     "<Feldname aus der Messung, leer wenn Frage 3 keine ID ergab>",
	agentType: "<Feldname aus der Messung, leer wenn keiner>",
}
```

Die spitzen Klammern ersetzt die Ausführung durch die gemessenen Namen; ein
Commit mit spitzen Klammern ist ein Fehler, den `grep -n '<Feldname' internal/hosts`
vor dem Commit findet. Liegt `agent_id` in der Nutzlast verschachtelt (etwa
unter `toolCall.args`), liest der Adapter den Pfad; der Test hält ihn fest.

Ergab Frage 3 keine gemeinsame ID: `AgentID` bleibt `""`, und
`subagent-start`/`-stop` verweigern mit Exit 1 („payload carries no
agent_id“). `loomux init` (Stufe 4) schreibt diese beiden Einträge für
Antigravity dann nicht; das geht in die Abweichungsliste.

`writeAntigravityContext`: Frage 2 ja → dieselbe Hülle wie Claude mit dem
Ereignis; nein → `ErrNoAdapter` bleibt, der Kommentar nennt die Messung.

- [ ] **Schritt 3: Tests laufen lassen**

Run: `go test ./internal/hosts/ -cover`
Erwartet: PASS, 100.0 %.

- [ ] **Schritt 4: Commit**

```sh
grep -n '<Feldname' internal/hosts || true
git add internal/hosts
git commit -m "feat(hosts): read Antigravity's measured hook payloads"
```

Der `grep` muss leer sein.

---

### Task 16: **Halt** — der Mensch trägt die Hooks ein

`.claude/settings.json` schreibt kein Agent.

- [ ] **Schritt 1: Dem Menschen die Einträge nennen**

Wörtlich, zum Ergänzen unter `"hooks"` in `.claude/settings.json`, in der Form
der vorhandenen Einträge:

```json
"Stop": [{"hooks": [{"type": "command", "timeout": 300,
  "command": "\"${CLAUDE_PROJECT_DIR}/bin/loomux.exe\" hook stop --host claude --root \"${CLAUDE_PROJECT_DIR}\" --budget 270s"}]}],
"SubagentStart": [{"hooks": [{"type": "command", "timeout": 30,
  "command": "\"${CLAUDE_PROJECT_DIR}/bin/loomux.exe\" hook subagent-start --host claude --root \"${CLAUDE_PROJECT_DIR}\""}]}],
"SubagentStop": [{"hooks": [{"type": "command", "timeout": 30,
  "command": "\"${CLAUDE_PROJECT_DIR}/bin/loomux.exe\" hook subagent-stop --host claude --root \"${CLAUDE_PROJECT_DIR}\""}]}]
```

Dazu der Hinweis: `bin/loomux.exe` vorher neu bauen
(`go build -o bin/loomux.new.exe ./cmd/loomux`, dann
`go run ./cmd/loomux dev swap-binary --dir bin`), sonst kennt das Binary die
Ereignisse nicht. Warten, bis der Mensch sagt, dass es eingetragen ist.

- [ ] **Schritt 2: Prüfen, dass es wirkt**

```sh
go run ./cmd/loomux status
```

Erwartet: `[OK] Stop`, `[OK] SubagentStart`, `[OK] SubagentStop`.

Ab hier läuft das Tor an jedem Rundenende dieser Sitzung: Tasks 17–21 laufen
unter `stop`, eine Runde mit Änderungen kostet rund 22 s mehr. Das ist die
Selbstnutzung und gewollt.

---

### Task 17: Selbstnutzung und Abweichungsliste

**Files:**
- Create: `docs/.superpowers/parity/stufe-2c.md`

- [ ] **Schritt 1: Selbstnutzung belegen**

In dieser Sitzung eine Änderung an einer Go-Datei, die `go vet` bemängelt
(etwa ein `fmt.Printf("%d", "x")` in einer Testdatei), und die Runde enden
lassen. Erwartet: Die Runde wird mit `vet/…`-Befund gehalten; Befund beheben;
die nächste Runde endet grün, und `.loomux/state/hooks/<id>.json` trägt einen
neuen `base` und `green`. Einen Subagenten starten, der einen lokalen Branch
anlegt; das Rundenende zeigt `subagent <id>: branch … is new at …`. Beides mit
Datum im Kopf der Akte festhalten; den Branch danach löschen.

- [ ] **Schritt 2: Akte schreiben**

Aufbau wie `parity/stufe-2a.md`: Kopf (Stand, Quelle `loomux-1a-source`
`9d01a60`, Zahl der Fälle 15, Belege aus Schritt 1), dann die Liste 1–17 der
Spec plus die Nachträge dieses Plans, die Verhalten ändern (1: kein
`GIT_SSH_COMMAND`; 4: ignoriertes Wurzelverzeichnis; 6: „nichts geprüft“;
8: verschwundene Basis; 9: Exit 1 wird mit Befunden Exit 2) und
was Task 14/15 ergab. Je Eintrag: Fall (Name aus `testdata/cases/2c/`, oder
„kein Fall — Unit-Test `<name>`“), altes Verhalten mit Zitat aus der
aufgenommenen `stderr`-Datei des Quellfalls, neues Verhalten, Begründung,
Freigabe (leer; der Mensch trägt sie ein).

Dazu ein Abschnitt „Absichten der Python-Tests“: je Test aus
`tests/hooks/test_stop.py` (26), `test_subagent_start.py` (4),
`test_subagent_stop.py` (16) am Tag `loomux-1a-source` eine Zeile: Name, der
Go-Test, der dieselbe Absicht prüft, oder die Nummer der Abweichung.

- [ ] **Schritt 3: Abgleich prüfen**

```sh
grep -o '"hook-[^"]*"' internal/cli/cases_2c_test.go | sort
grep -c '^def test_' "C:/Users/micro/Documents/#GIT/ultraloom/tests/hooks/test_stop.py"
```

Jeder Name aus `approved2c` steht in `stufe-2c.md`, und die Zahl der Zeilen im
Abschnitt „Absichten“ ist 46.

- [ ] **Schritt 4: Commit**

```sh
git add docs/.superpowers/parity/stufe-2c.md
git commit -m "docs(parity): list the stage 2c deviations"
```

---

### Task 18: Mutationsrunde

- [ ] **Schritt 1: Laufzeit schätzen**

```sh
go test ./internal/hooks/ -count=1
go test ./internal/sessions/ ./internal/gitwork/ -count=1
```

Die Zeiten notieren. Jeder Mutant in `hooks` startet echtes Git (Welten über
`BuildGitWorld`). Liegt die Schätzung für `hooks` mit `--workers` = Kernzahl
über 30 Minuten, die Runde auf `stop.go` und `subagent.go` begrenzen
(`--files`, falls `dev mutants` es kennt; sonst `--family`) und das in der Akte
begründen.

- [ ] **Schritt 2: Runde fahren**

```sh
go run ./cmd/loomux dev mutants ./internal/sessions ./internal/gitwork
go run ./cmd/loomux dev mutants ./internal/hooks
```

- [ ] **Schritt 3: Überlebende verfügen**

Je Überlebendem: Test nachreichen (neuer Lauf nur dieses Mutanten mit
`--only`), oder als gleichwertig begründen. Beides in `parity/stufe-2c.md`,
Abschnitt „Mutationsrunde“, mit Befehl, Commit und Worker-Zahl.

- [ ] **Schritt 4: Commit**

```sh
git add internal docs/.superpowers/parity/stufe-2c.md
git commit -m "test(hooks): kill the survivors of the stage 2c mutation round"
```

---

### Task 19: Messen

**Files:**
- Create: `testdata/bench/2c-hooks.json`
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`

- [ ] **Schritt 1: Bench-Fälle anlegen**

Vorher `testdata/bench/2a-check.json` lesen und dieselbe Form nehmen. Die
Welten baut `dev bench-hooks` nicht; deshalb eine Welt einmal mit
`BuildGitWorld` in den Scratchpad legen (ein kleines `go run` über
`internal/cases`, oder `loomux dev record-case` mit einem Leerlauf) und `dir`
darauf zeigen lassen:

1. `stop-unchanged` — `bin/loomux.exe hook stop …` in einer Welt, deren Baum
   gleich `green` ist (kein Werkzeug startet: Eigenzeit plus Fingerabdruck);
2. `stop-fake-chain` — mit faketool auf dem PATH, geänderter Baum;
3. `subagent-start` und `subagent-stop` gegen ein lokales Bare-Remote;
4. `content-tree` — ein Benchmark `BenchmarkContentTree` in
   `gitwork_test.go` über dieses Repo (`go test ./internal/gitwork/ -bench ContentTree -run ^$`).

- [ ] **Schritt 2: Messen**

```sh
go build -o bin/loomux.exe ./cmd/loomux
go run ./cmd/loomux dev bench-hooks -n 20 testdata/bench/2c-hooks.json
```

Zusätzlich `stop` ohne neuen Inhalt auf **diesem** Repo (7.322 Dateien): Ziel
etwa 300 ms (Spec, Messen).

- [ ] **Schritt 3: Eintragen**

In beide `benchmarks.md` chronologisch: Datum und Uhrzeit, was gemessen wurde,
kalt und warm, `ls-remote` gegen ein lokales Remote und gegen GitHub
getrennt ausgewiesen.

- [ ] **Schritt 4: Commit**

```sh
git add testdata/bench/2c-hooks.json docs/en/benchmarks.md docs/de/benchmarks.md internal/gitwork/gitwork_test.go
git commit -m "docs(bench): measure the stop gate and the subagent hooks"
```

---

### Task 20: Doku und Specs

**Files:**
- Modify: `docs/en/hooks.md`, `docs/de/hooks.md`, `docs/en/cli-reference.md`,
  `docs/de/cli-reference.md`, `docs/en/configuration.md`,
  `docs/de/configuration.md`, `docs/en/migration.md`, `docs/de/migration.md`,
  `README.md`, `README.de.md`,
  `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`,
  `docs/.superpowers/specs/2026-09-19-loomux-stufe-2c-design.md`

Vorher `ls docs/en docs/de` lesen und die tatsächlichen Dateinamen nehmen.

- [ ] **Schritt 1: `hooks.md`** — Phase 3 des Diagramms ohne „not built
yet“; §8 mit allen fünf Hooks als „runs“, je ein Unterabschnitt wie
`session-start today` für `stop` (Ablauf, Exit-Codes, Zähler, Fingerabdruck,
Marker, Budget) und für `subagent-start`/`-stop` (Snapshot, Befund, Zustellung
über `stop`); der Satz über unbekannte Ereignisse (`:359`) nennt die sechs
bekannten. Englisch und deutsch, gleiche Struktur.

- [ ] **Schritt 2: `cli-reference.md`** — `hook stop --budget`,
`hook subagent-start`, `hook subagent-stop`; die Zeile `:179` („Makes no
worktree junctions … Stage 2c“) nachziehen.

- [ ] **Schritt 3: `configuration.md`** — Profil `stop`; `lint/wiki` im
Check-Scope; `.loomux/no-verify`.

- [ ] **Schritt 4: `migration.md`** (Pflicht nach `AGENTS.md`) — Stufe 2c
auf fertig, die übernommenen Fähigkeiten (`stop`, `subagent-*`,
Antigravity-Adapter nach Messung) mit Status.

- [ ] **Schritt 5: READMEs** — Stand und Fahrplan: 2c fertig.

- [ ] **Schritt 6: Fusions-Spec** — Zeile 2c auf ✅ mit Datum, Spec und Plan;
die Prioritätstabelle: „sofort baubar“ bei 2c um „Antigravity nach eigener
Messung“ berichtigen; Kopfzeile „Stand“.

- [ ] **Schritt 7: 2c-Spec** — „Stand“ auf umgesetzt; die zehn Nachträge
dieses Plans und was Tasks 1, 13, 14, 15 ergeben haben, als Abschnitt
„Nachträge“ am Ende, wie in der 2a-Spec.

- [ ] **Schritt 8: Commit**

```sh
git add README.md README.de.md docs
git commit -m "docs: document the stop gate, the subagent hooks and stage 2c"
```

---

### Task 21: Abschluss

- [ ] **Schritt 1: Tor und Fallsuiten**

```sh
sh ci/gate.sh
go test ./internal/cli/ -run 'TestCases' -count=1
```

Erwartet: grün.

- [ ] **Schritt 2: Fertig-Kriterien der Fusions-Spec abhaken**

1. Fälle grün oder freigegeben in `stufe-2c.md` (Freigaben trägt der Mensch).
2. 100 % Coverage — Tor grün.
3. Mutationsrunde dokumentiert — Task 18.
4. Zielwerte gemessen — Task 19.
5. loomux nutzt `stop` und `subagent-*` selbst — Tasks 16 und 17.

- [ ] **Schritt 3: Übergabe**

Commits nach Thema gruppieren und den Pull Request vorbereiten: das übernimmt
der Skill `release-pr` (Label `release:minor`, `feat(hooks)`), vor dem ersten
Push. Gepusht wird vom Menschen.
