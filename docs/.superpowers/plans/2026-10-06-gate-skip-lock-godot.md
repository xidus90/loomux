# Tor-Infrastruktur: Überspringen, Sperre, `{godot}` — Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eine Lane kann bei Commits und Turn-Enden aussetzen, die nur von ihr benannte Pfade ändern; eine Lane kann ihren Stack und ihre Area je Checkout gegen einen zweiten Lauf sperren; eine `gdscript`-Lane kann `{godot}` schreiben und bekommt ein Binary, das zu `project.godot` passt.

**Architecture:** Alles sitzt in `internal/verify` (Schema, Plan, Lauf) mit zwei Zubringern: `internal/gitwork` liefert die geänderten Pfade, `internal/pathkey` den Glob-Matcher der Policy. Die Aufrufer `internal/cli/check.go`, `internal/hooks/stop.go` und `internal/hooks/post_edit.go` reichen geänderte Pfade, den Aufrufernamen der Sperre und den Godot-Finder in `PlanEnv` und `RunOptions` hinein. Vorab ein eigener Fix: ein `budget` erreicht am Turn-Ende einen Nachfolger, der Dateien des Vorgängers liest.

**Tech Stack:** Go 1.26 (Toolchain 1.27), git ≥ 2.54, `internal/lock` (Betriebssystemsperre), `internal/child`.

**Spec:** `docs/.superpowers/specs/2026-10-06-gate-skip-lock-godot-design.md`

## Global Constraints

- Ohne die neuen Schlüssel verhält sich jedes Projekt wie heute; kein Preset setzt `skip_when_only`, `lock` oder `{godot}`.
- Coverage 100 % je Funktion; Ausnahme nur mit `//coverage:exempt <reason>` direkt über `func`.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst.
- Code, Kommentare, Meldungen, Commits englisch; Doku unter `docs/en` und `docs/de` mit gleichen Dateinamen.
- Commit-Nachrichten nach Conventional Commits, ohne Arbeitspapier (kein Plan, keine Spec, kein Task), ohne Modell als Mitautor.
- Mehrzeilige Commit-Nachrichten per Write in eine Datei im Scratchpad und `git commit -F <datei>`; die Ausgabe des Commits nie verwerfen, sondern nach `<scratchpad>/commit-<task>.log` lenken.
- Kein `python -` und kein Heredoc an einen Interpreter; kein Beenden von Prozessen nach Namen; kein Hintergrundprozess, der nach dem Bericht weiterläuft.
- Vor jedem Commit als Bedingung prüfen: `[ "$(git rev-parse --show-toplevel)" = "C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/gate-skip-lock-godot" ] && [ "$(git branch --show-current)" = feat/gate-skip-lock-godot ]`.
- Jede neue Testzeile läuft vor dem Produktionscode rot, an einer Assertion (`--- FAIL` mit Meldung), nicht am Compile-Fehler: neue Felder und Funktionen werden dafür zuerst als wirkungslose Stubs angelegt. Der Bericht zeigt Befehl und vollständige Ausgabe.
- Je Task eine Mutationsrunde über die neue Logik mit `go test -overlay` (Overlay-Pfade in der Form `C:/…`, ohne `-cover`), zuerst ein unveränderter Kontrollmutant, der überleben muss; je Mutant „gebaut: ja“ und die tötende Testzeile. Ausgabe je Mutant in `<scratchpad>/t<N>-mut-<n>.txt`.

## Review Focus

1. **Umbenennung aus dem Code nach `docs/`.** Mit gits Umbenennungserkennung erscheint nur der neue Pfad, die Lane setzte aus, obwohl Code verschwand. Erwartung: beide Pfade zählen (`--no-renames`). Test in Task 3.
2. **Pfad mit Umlaut.** Ohne `-z` quotet git ihn (`"docs/\303\244.md"`), er passt auf kein Muster, die Ersparnis fällt still weg. Erwartung: der Pfad kommt roh an und passt. Test in Task 3.
3. **`GIT_INDEX_FILE` eines anderen Repos.** loomux' eigener pre-commit fährt `go test`, die Testwelten erben die Variable. Erwartung: ein Index außerhalb des Git-Verzeichnisses des Roots ist keine Antwort, alles läuft. Test in Task 3.
4. **`--version` mit vorangestellter Zeile.** Ein Build, der vor der Version eine Warnzeile druckt. Erwartung: die erste Zeile, die als Version lesbar ist, gilt. Test in Task 6.
5. **Frischer Klon ohne `.loomux/state`.** Erwartung: die Sperre legt ihren Ordner selbst an. Test in Task 5.

---

### Task 1: Ein `budget` erreicht einen Nachfolger, der Dateien des Vorgängers liest

Eigener `fix:`-Commit: der Fehler besteht auf master.

> **Korrekturen aus der Vorabprüfung (gehen dem Text unten vor).**
> - K1.1: `Consumes` statt „`Reads` nicht leer“ ist Spec-Nachtrag 1, so gewollt.
> - K1.2: Zusätzlich ein Test im Stop-Gate (`internal/hooks/stop_test.go`), nach dem Muster von `TestStopReportsABudgetThatRanOut` (Welt `stopWorld`, `stuckClock`): eine Kette `test → coverage`, in der `test` als `budget` endet und `coverage` den Report des Test-Laufs liest. Erwartet: `ExitInternal` mit „not everything was verified“, `Base` unverändert und kein grüner Baum geschrieben (`stateOf(t, root).Green` leer bzw. unverändert). Vor dem Fix muss er rot sein (`coverage` wird `failed`, der Turn antwortet `ExitDenied`). Enthält das Stop-Profil der Welt `coverage` nicht, setzt der Test es in `.loomux/config.toml` (`[verify.profiles] stop = [...]`); fährt `test` dort nicht seine Messform, wählt der Test eine Welt oder Konfiguration, in der `coverage` per `Reads` am Test hängt. Der RED-Nachweis zeigt genau diese Assertion.
> - K1.3: Explizite Pfade beim `git add`; nie `internal/verify` als Ordner (dort liegt eine fremde, nicht zu committende `internal/verify/.who`).

**Files:**
- Modify: `internal/verify/plan.go` (Feld `Job.Consumes`, `settle`)
- Modify: `internal/verify/run.go` (`inherit`)
- Test: `internal/verify/run_test.go`, `internal/verify/plan_test.go`
- Modify: `docs/en/configuration.md`, `docs/de/configuration.md` (Abschnitt „States and verdict“, Punkt „What a lane inherits“)

**Interfaces:**
- Produces: `Job.Consumes bool` — die Lane liest, was ihr Vorgänger schreibt (Coverage-Datei per Name oder, bei Python, über `COVERAGE_FILE`). Task 4 setzt es auch für einen übersprungenen Vorgänger.

- [ ] **Step 1: Stub anlegen.** In `Job` (`internal/verify/plan.go`) hinter `Reads`:

```go
	// Consumes says the job reads what its predecessor writes: a coverage
	// file by name, or Python's data file through COVERAGE_FILE.
	Consumes bool
```

- [ ] **Step 2: Failing tests schreiben.** In `internal/verify/run_test.go` hinter `TestRunLetsACheckRunPastABudgetPredecessor`:

```go
// A predecessor stopped by the budget wrote nothing, or half: a lane that
// reads its files takes the budget over instead of failing on them.
func TestRunHandsABudgetOnToALaneThatReadsItsPredecessor(t *testing.T) {
	pre := job("test/go", -1)
	pre.Pre = StateBudget
	next := job("coverage/go", 0, "c")
	next.Consumes = true
	next.Reads = []string{filepath.Join(t.TempDir(), "missing.out")}
	f := &fakeStart{answer: ok}
	out := Run([]Job{pre, next}, opts(f))
	if out[1].State != StateBudget || out[1].BlockedBy != "" || len(f.started) != 0 {
		t.Fatalf("%+v, started %v", out[1], f.started)
	}
}
```

In `internal/verify/plan_test.go` hinter `TestPlanLinksPythonCoverageToTheMeasuringTestLane`:

```go
func TestSettleMarksALaneThatReadsItsPredecessor(t *testing.T) {
	req := Request{Kinds: []string{"test", "coverage"}}
	jobs, _ := Plan(effFor(t, "", goOnly), req, env(t.TempDir()))
	if jobs[0].Consumes || !jobs[1].Consumes || jobs[1].After != 0 {
		t.Fatalf("go: %+v", jobs)
	}
	// Python's report reads the data file through the environment.
	jobs, _ = Plan(effFor(t, "", pythonOnly), req, env(t.TempDir()))
	if !jobs[1].Consumes || jobs[1].Reads != nil || jobs[1].After != 0 {
		t.Fatalf("python: %+v", jobs[1])
	}
	// after only orders: a lane that reads nothing waits and consumes nothing.
	src := "[verify.go.coverage]\ncommands = [\"echo\"]\nafter = \"test\"\n"
	jobs, _ = Plan(effFor(t, src, goOnly), req, env(t.TempDir()))
	if jobs[1].Consumes || jobs[1].After != 0 {
		t.Fatalf("ordering only: %+v", jobs[1])
	}
}
```

- [ ] **Step 3: Rot laufen lassen.**

Run: `go test ./internal/verify -run "TestRunHandsABudgetOnToALaneThatReadsItsPredecessor|TestSettleMarksALaneThatReadsItsPredecessor" -count=1 -v`
Expected: beide `--- FAIL`; der erste mit `State:failed` und `missing.out is missing`, der zweite mit `go: ` (Consumes false).

- [ ] **Step 4: Implementieren.** In `settle` (`internal/verify/plan.go`) die Zeile im Verknüpfungszweig:

```go
			if p.Pre != "" || writesAny(p, links[j], needs) {
				job.After, job.Reads, job.Consumes = j, l.reads, len(needs) > 0
				return nil
			}
```

In `inherit` (`internal/verify/run.go`):

```go
	case StateBudget, StateMissingTool, StateUnready:
		// In a check only a spent budget gets here, the other two are red. A
		// lane that reads its predecessor's files has nothing to read then;
		// one that only waits for it runs as it would have.
		if r.opt.Scope == ScopeEdit || job.Consumes {
			return Outcome{Job: job, State: pred.State, Probation: r.probation(job)}, true
		}
```

- [ ] **Step 5: Grün und Nachbarn.**

Run: `go test ./internal/verify ./internal/hooks -count=1`
Expected: PASS, auch `TestRunLetsACheckRunPastABudgetPredecessor` und `TestRunInheritsWhatAnEditCannotJudge`.

- [ ] **Step 6: Doku.** In `docs/en/configuration.md`, Punkt „What a lane inherits“, den Satz ersetzen durch:

```markdown
- **What a lane inherits.** When the lane it waits for could not run, a lane
  takes over that state (`unavailable`, `not-applicable`, and in the edit scope
  also `budget`, `missing-tool`, `unready`) instead of turning `blocked`. In a
  check it takes over `budget` too when it reads the files that lane writes
  (a coverage file by name, or Python's data file): a lane stopped by the
  budget wrote nothing to read. A lane that only waits runs.
```

Den deutschen Zwilling in `docs/de/configuration.md` (Zeile mit „Edit-Scope auch `budget`, `missing-tool`, `unready`“) gleich ergänzen:

```markdown
  Im Check übernimmt sie auch `budget`, wenn sie die Dateien dieser Lane liest
  (eine Coverage-Datei per Name oder Pythons Datendatei): eine Lane, die das
  Budget gestoppt hat, hat nichts zu lesen hinterlassen. Eine Lane, die nur
  wartet, läuft.
```

- [ ] **Step 7: Mutationsrunde.** Mutanten: (a) Kontrolle unverändert; (b) `|| job.Consumes` → `|| false`; (c) `len(needs) > 0` → `true`; (d) `len(needs) > 0` → `len(l.reads) > 0`. (b)–(d) müssen sterben; (d) an der Python-Zeile.

- [ ] **Step 8: Commit.**

Nachricht (Datei `<scratchpad>/msg-t1.txt`):

```
fix(verify): hand a spent budget on to a lane that reads its predecessor

At a turn end a coverage lane behind a test lane the budget stopped
started anyway, found no profile and turned red, so the turn was held
over a budget. It now takes the budget over when it reads that lane's
files; a lane that only waits for it runs as before.
```

```bash
git add internal/verify/plan.go internal/verify/run.go internal/verify/run_test.go internal/verify/plan_test.go docs/en/configuration.md docs/de/configuration.md
git commit -F "$S/msg-t1.txt" > "$S/commit-t1.log" 2>&1
```

---

### Task 2: Der Glob-Matcher der Policy zieht nach `internal/pathkey`

**Files:**
- Create: `internal/pathkey/glob.go`, `internal/pathkey/glob_test.go`
- Modify: `internal/pathkey/pathkey.go` (Paketkommentar)
- Modify: `internal/hooks/guard.go` (`matchGlob` entfernen, Aufruf Zeile ~350), `internal/hooks/guardjudge.go` (Zeilen ~458–459)
- Modify: `internal/hooks/guard_test.go` (Test `TestMatchGlobReadsALeadingDoubleStarAsAnyDirectory` zieht um)

**Interfaces:**
- Produces: `pathkey.Glob(pattern, name string) (bool, error)` — genau das Verhalten von `matchGlob`.

> **Korrekturen aus der Vorabprüfung (gehen dem Text unten vor).**
> - K2.1: `matchGlob` ist der einzige Nutzer von `path` in `internal/hooks/guard.go`; der Import `path` fällt dort mit weg. Der Aufruf steht seit v1.1.0 bei Zeile ~353.

- [ ] **Step 1: Verschieben.** `matchGlob` mit seinem Doku-Kommentar unverändert nach `internal/pathkey/glob.go` als `Glob` (rekursiver Aufruf ebenfalls `Glob`), Import `path` und `strings`. In `guard.go` und `guardjudge.go` jeden `matchGlob(` durch `pathkey.Glob(` ersetzen, Import `github.com/xidus90/loomux/internal/pathkey` ergänzen, `matchGlob` löschen.

- [ ] **Step 2: Test umziehen.** `TestMatchGlobReadsALeadingDoubleStarAsAnyDirectory` aus `internal/hooks/guard_test.go` nach `internal/pathkey/glob_test.go` (Paket `pathkey`), Aufrufe `Glob(`, Name `TestGlobReadsALeadingDoubleStarAsAnyDirectory`. Dazu, weil `Glob` jetzt auch Commit-Pfade liest:

```go
// A pattern with a slash ending in /** takes the directory and all below it;
// one without a slash looks at the file name in any directory.
func TestGlobOfDirectoriesAndNames(t *testing.T) {
	for _, row := range []struct {
		pattern, name string
		want          bool
	}{
		{"docs/**", "docs", true},
		{"docs/**", "docs/de/a.md", true},
		{"docs/**", "docsx/a.md", false},
		{"*.md", "docs/de/ä.md", true},
		{"*.md", "a.mdx", false},
		{"README.md", "sub/README.md", true},
	} {
		if got, err := Glob(row.pattern, row.name); err != nil || got != row.want {
			t.Errorf("Glob(%q, %q) = %v, %v; want %v", row.pattern, row.name, got, err, row.want)
		}
	}
}
```

- [ ] **Step 3: Paketkommentar.** `internal/pathkey/pathkey.go`:

```go
// Package pathkey compares paths the way the file system of a platform
// tells them apart: in its slash direction, and without regard to case
// where the file system ignores it. Glob is the one exception to the case
// rule: it matches slash-separated paths against the globs of a policy rule
// or a lane, byte for byte.
```

- [ ] **Step 4: Tests.**

Run: `go test ./internal/pathkey ./internal/hooks -count=1`
Expected: PASS. Ein RED gibt es hier nicht (reines Verschieben); der Nachweis ist, dass `go vet ./...` sauber ist und `grep -rn matchGlob internal` nichts mehr findet.

- [ ] **Step 5: Commit.**

```
refactor: move the policy glob matcher into pathkey

The verify lanes match changed paths against the same globs as the
policy rules, and verify cannot import hooks.
```

```bash
git add internal/pathkey internal/hooks/guard.go internal/hooks/guardjudge.go internal/hooks/guard_test.go
git commit -F "$S/msg-t2.txt" > "$S/commit-t2.log" 2>&1
```

---

### Task 3: Geänderte Pfade aus git

**Files:**
- Create: `internal/gitwork/paths.go`, `internal/gitwork/paths_test.go`

**Interfaces:**
- Produces:
  - `gitwork.StagedPaths(root, inherited string) ([]string, error)` — die Pfade, die der Index eines Commit-Hooks gegen HEAD ändert; `inherited` ist das `GIT_INDEX_FILE` des Hooks.
  - `gitwork.ChangedBetween(root, from, to string) ([]string, error)` — die Pfade, in denen sich zwei Bäume unterscheiden.

> **Korrekturen aus der Vorabprüfung (gehen dem Text unten vor).**
> - K3.1: Task 3 und Task 4 gehen an einen Implementierer und enden in einem Commit (Step 7 unten entfällt als Haltepunkt); das Review sieht den gemeinsamen Diff.
> - K3.2: Der Test mit relativem Index läuft aus einem Unterverzeichnis, sonst überlebt der Mutant „`absIndex` gibt `inherited` unverändert zurück“ (git löst ihn gegen `root` ebenso auf): Unterordner `sub` anlegen, `t.Chdir(filepath.Join(root, "sub"))`, `StagedPaths(root, "../.git/index")`. Der Mutant heißt richtig: `filepath.Join(wd, inherited)` → `inherited`.
> - K3.3: Coverage ohne Ausnahme: `own, _ := os.Stat(…)` (ein `nil`-`FileInfo` lässt `os.SameFile` `false` antworten) und in `absIndex` `wd, _ := os.Getwd()` wie `CommitIndex`; die `//coverage:exempt`-Zeile an `absIndex` entfällt.
> - K3.4: Weitere Tests (Spec-Nachtrag 3): (a) die Index-Kopien eines Commits: `b.txt` stagen, die Bytes von `.git/index` lesen, `git reset -q`, dann je eine Kopie als `<gitdir>/index.lock` und `<gitdir>/next-index-123.lock` schreiben; `StagedPaths(root, <kopie>)` ergibt `["b.txt"]`, die Kopie danach löschen. (b) Abschluss eines Merges: Zweig `side` mit neuer Datei `c.go`, zurück auf den Ausgangszweig, dort ein eigener Commit (damit kein Fast-Forward), `git merge --no-commit --no-ff -q side`; `StagedPaths(root, <gitdir>/index)` ergibt `["c.go"]`. (c) In `TestStagedPathsRefusesAnotherRepositorysIndex` zusätzlich: `<gitdir>/index.lock`, das nicht existiert, ist ein Fehler; `<gitdir>/next-index-1.lock` mit Inhalt `garbage` ist ein Fehler (`diff --cached` scheitert). Commit-Identität so setzen, wie `repoWithCommit` es tut.

- [ ] **Step 1: Stubs.** `internal/gitwork/paths.go`:

```go
package gitwork

// StagedPaths names the paths a commit hook's index changes against HEAD.
func StagedPaths(root, inherited string) ([]string, error) { return nil, nil }

// ChangedBetween names the paths in which two trees differ.
func ChangedBetween(root, from, to string) ([]string, error) { return nil, nil }
```

- [ ] **Step 2: Failing tests.** `internal/gitwork/paths_test.go` mit den Hilfen `repoWithCommit`, `mustGit`, `mustGitWith` aus `gitwork_test.go`:

```go
package gitwork

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func indexOf(t *testing.T, root string) string {
	t.Helper()
	return filepath.Join(mustGit(t, root, "rev-parse", "--absolute-git-dir"), "index")
}

func TestStagedPathsNamesWhatTheCommitChanges(t *testing.T) {
	root := repoWithCommit(t)
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"docs/ä.md": "x", "b.go": "package b"} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustGit(t, root, "add", "docs", "b.go")
	got, err := StagedPaths(root, indexOf(t, root))
	slices.Sort(got)
	if err != nil || !slices.Equal(got, []string{"b.go", "docs/ä.md"}) {
		t.Fatalf("%q %v", got, err)
	}
}

// A rename lists both ends: code that left must count.
func TestStagedPathsListsBothEndsOfARename(t *testing.T) {
	root := repoWithCommit(t)
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustGit(t, root, "mv", "a.txt", "docs/a.txt")
	got, err := StagedPaths(root, indexOf(t, root))
	slices.Sort(got)
	if err != nil || !slices.Equal(got, []string{"a.txt", "docs/a.txt"}) {
		t.Fatalf("%q %v", got, err)
	}
}

// git names the index relative to the directory it starts the hook in.
func TestStagedPathsReadsARelativeIndexFromTheHooksDirectory(t *testing.T) {
	root := repoWithCommit(t)
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, root, "add", "b.txt")
	t.Chdir(root)
	got, err := StagedPaths(root, ".git/index")
	if err != nil || !slices.Equal(got, []string{"b.txt"}) {
		t.Fatalf("%q %v", got, err)
	}
}

// An index of another repository is no answer: the gate runs every lane.
func TestStagedPathsRefusesAnotherRepositorysIndex(t *testing.T) {
	root, other := repoWithCommit(t), repoWithCommit(t)
	if _, err := StagedPaths(root, indexOf(t, other)); err == nil {
		t.Fatal("a foreign index must be an error")
	}
	if _, err := StagedPaths(root, filepath.Join(t.TempDir(), "index")); err == nil {
		t.Fatal("an index that is not there must be an error")
	}
	if _, err := StagedPaths(t.TempDir(), indexOf(t, root)); err == nil {
		t.Fatal("a root outside git must be an error")
	}
}

func TestChangedBetweenNamesBothEndsOfARename(t *testing.T) {
	root := repoWithCommit(t)
	from := mustGit(t, root, "rev-parse", "HEAD^{tree}")
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustGit(t, root, "mv", "a.txt", "docs/ä.txt")
	to := mustGit(t, root, "write-tree")
	got, err := ChangedBetween(root, from, to)
	slices.Sort(got)
	if err != nil || !slices.Equal(got, []string{"a.txt", "docs/ä.txt"}) {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := ChangedBetween(root, from, "0000000000000000000000000000000000000000"); err == nil {
		t.Fatal("a tree git does not know must be an error")
	}
}
```

- [ ] **Step 3: Rot.**

Run: `go test ./internal/gitwork -run "StagedPaths|ChangedBetween" -count=1 -v`
Expected: jeder Test `--- FAIL` an seiner Assertion (`[] <nil>` bzw. „must be an error“).

- [ ] **Step 4: Implementieren.** `internal/gitwork/paths.go`:

```go
package gitwork

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StagedPaths names the paths a commit hook's index changes against HEAD:
// the paths of the commit under way. inherited is the hook's GIT_INDEX_FILE,
// relative to the directory git starts the hook in. Only an index in root's
// own git directory answers -- index, index.lock or a commit of paths'
// next-index-<pid>.lock (see CommitIndex) --, so a variable a test process
// inherited from another repository's hook is an error, not a list.
//
// --no-renames lists both ends of a rename, -z hands paths over unquoted.
func StagedPaths(root, inherited string) ([]string, error) {
	index, err := absIndex(inherited)
	if err != nil {
		return nil, err
	}
	out, err := git(root, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return nil, err
	}
	own, err := os.Stat(strings.TrimSpace(out))
	if err != nil {
		return nil, err
	}
	if given, err := os.Stat(filepath.Dir(index)); err != nil || !os.SameFile(own, given) {
		return nil, fmt.Errorf("%s is no index of %s", inherited, root)
	}
	if _, err := os.Stat(index); err != nil {
		return nil, err
	}
	out, err = gitWith(root, []string{"GIT_INDEX_FILE=" + index}, "diff", "--cached", "--name-only", "--no-renames", "-z")
	if err != nil {
		return nil, err
	}
	return splitNUL(out), nil
}

// ChangedBetween names the paths in which two trees differ, both ends of a
// rename among them.
func ChangedBetween(root, from, to string) ([]string, error) {
	out, err := git(root, "diff-tree", "-r", "--name-only", "--no-renames", "-z", from, to)
	if err != nil {
		return nil, err
	}
	return splitNUL(out), nil
}

func splitNUL(out string) []string {
	return strings.FieldsFunc(out, func(r rune) bool { return r == 0 })
}

// absIndex resolves a hook's GIT_INDEX_FILE against the directory git
// started the hook in, which is this process's working directory.
//
//coverage:exempt os.Getwd fails only when the working directory was removed under the process
func absIndex(inherited string) (string, error) {
	if filepath.IsAbs(inherited) {
		return inherited, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, inherited), nil
}
```

- [ ] **Step 5: Grün.**

Run: `go test ./internal/gitwork -count=1`
Expected: PASS.

- [ ] **Step 6: Mutationsrunde.** Mutanten: Kontrolle; `--no-renames` streichen (stirbt an beiden Rename-Tests); `-z` streichen und `splitNUL` durch `strings.Fields` ersetzen (stirbt am Umlaut); `!os.SameFile(own, given)` → `false` (stirbt am fremden Index); relativer Zweig: `filepath.Join(wd, index)` → `index` (stirbt am relativen Test).

- [ ] **Step 7: Kein eigener Commit.** Task 3 und 4 bilden zusammen den Feature-Commit; Task 3 wird nur gestaged, wenn der Implementierer anhält: `git add internal/gitwork/paths.go internal/gitwork/paths_test.go`.

---

### Task 4: `skip_when_only` — Schema, Plan, Aufrufer, Doku

**Files:**
- Modify: `internal/verify/schema.go` (`Lane.SkipWhenOnly`, `parseLaneTable`, neue `laneGlobs`)
- Modify: `internal/verify/effective.go` (`merge`)
- Modify: `internal/verify/show.go` (`writeLane`)
- Modify: `internal/verify/plan.go` (`PlanEnv.Changed`, `Job.Skipped`, `planJob`, `settle`, neue `skippedBy`)
- Modify: `internal/cli/check.go` (Seam `checkStaged`, `env.Changed`)
- Modify: `internal/hooks/stop.go` (`stopTrees` liefert `from`, `RunStop` setzt `Changed`, Seam `stopChanged`)
- Test: `internal/verify/schema_test.go`, `internal/verify/show_test.go`, `internal/verify/plan_test.go`, `internal/cli/check_test.go`, `internal/hooks/stop_test.go`
- Modify: `docs/en/configuration.md`, `docs/de/configuration.md`

**Interfaces:**
- Consumes: `pathkey.Glob` (Task 2), `gitwork.StagedPaths`, `gitwork.ChangedBetween` (Task 3), `Job.Consumes` (Task 1).
- Produces: `Lane.SkipWhenOnly []string`, `PlanEnv.Changed []string` (nil oder leer: keine Liste, alles läuft), `Job.Skipped bool`.

> **Korrekturen aus der Vorabprüfung (gehen dem Text unten vor).**
> - K4.1: Reihenfolge von RED und Code: `effFor` bricht per `t.Fatal` ab, solange `parseLaneTable` `skip_when_only` nicht kennt; Plan-, Show- und Merge-Tests würden also im Helfer scheitern, nicht an ihrer Assertion. Darum: Schema-Tests schreiben → RED (`unknown key "skip_when_only"`) → Step 5 (Schema, Merge, Show) → Show- und Merge-Test grün, Plan-Tests schreiben → RED (`TestPlanSkipsALaneWhenEveryChangedPathIsNamed` an `test: `, der neue Test aus K4.3 an seiner Assertion) → Step 6.
> - K4.2: In `skippedBy` `ok, _ := pathkey.Glob(glob, p); return ok` (`Glob` liefert nie `true` mit Fehler; der Mutant `ok && err == nil` → `ok` entfällt als äquivalent). Kommentar entsprechend: ein Muster, das beim Abgleich scheitert, passt nicht, und die Lane läuft.
> - K4.3 (Spec-Nachtrag 2): Der `p.Skipped`-Zweig in `settle` verknüpft nur einen Nachfolger, der liest (`len(needs) > 0`); ein Nachfolger, der nur ordnet, fällt durch auf das bestehende `continue` für `not-applicable` und läuft. Zusätzlicher Test in `plan_test.go`: `skipDocs` plus `[verify.go.coverage]` mit `commands = ["echo"]` und `after = "test"`, Kinds `test`, `coverage`, geändert nur `docs/a.md`: `jobs[0].Skipped`, aber `jobs[1].Pre == ""`, `!jobs[1].Consumes`, `jobs[1].After == -1` (bzw. der Wert, den `job()`/`planJob` für „nicht verknüpft“ setzen — vorher im Code nachsehen). Mutant: die Bedingung `len(needs) > 0` im Skipped-Zweig gestrichen, stirbt hier.
> - K4.4: `put` (`internal/cli/check_arm_hook_test.go`) legt keine Ordner an: vor `put(t, root, ".loomux/config.toml", …)` und `put(t, root, "docs/a.md", …)` je `os.MkdirAll` auf `.loomux` und `docs`.
> - K4.5: Der Doku-Kommentar von `stopTrees` nennt den neuen Rückgabewert `from`.
> - K4.6: Doku zusätzlich: `docs/en/hooks.md` (Abschnitt um „**Chain.** Every lane runs within `--budget`“, ~Zeile 934) und der Abschnitt zu `loomux check` in `docs/en/cli-reference.md` sagen in einem Satz, dass eine Lane mit `skip_when_only` im pre-commit (an `GIT_INDEX_FILE` erkannt) und am Turn-Ende (gegen den zuletzt grünen Baum) aussetzen kann; deutsche Zwillinge gleich.
> - K4.7: Files zusätzlich: `internal/verify/effective_test.go`, `internal/verify/run_test.go`, `docs/{en,de}/hooks.md`, `docs/{en,de}/cli-reference.md`. `git add` mit expliziten Dateipfaden, nie `internal/verify` als Ordner (dort liegt eine fremde `internal/verify/.who`, die nicht committet wird).

- [ ] **Step 1: Stubs.** `Lane` bekommt `SkipWhenOnly []string` (Kommentar: „Globs, relative to the repository root, of paths the lane does not care about; a commit or a turn end that changes only such paths plans the lane as not-applicable.“), `PlanEnv` bekommt `Changed []string` („The paths a commit or a turn end changes; nil outside both, and then every lane runs.“), `Job` bekommt `Skipped bool` („Skipped says skip_when_only took the lane out; lanes after it still link to it and inherit not-applicable.“). `parseLaneTable` kennt den Schlüssel noch nicht.

- [ ] **Step 2: Failing tests Schema und Show.** In `internal/verify/schema_test.go`:

```go
func TestParseSkipWhenOnly(t *testing.T) {
	cfg, err := parse(t, "[verify.go.test]\nskip_when_only = [\"**/*.md\", \"docs/**\"]\n")
	if err != nil || !slices.Equal(cfg.Stacks["go"]["test"].Lane.SkipWhenOnly, []string{"**/*.md", "docs/**"}) || !cfg.Stacks["go"]["test"].Set["skip_when_only"] {
		t.Fatalf("%v %+v", err, cfg.Stacks["go"]["test"])
	}
	for src, want := range map[string]string{
		"[verify.go.test]\nskip_when_only = []\n":          "[verify.go.test].skip_when_only is empty",
		"[verify.go.test]\nskip_when_only = \"*.md\"\n":    "[verify.go.test].skip_when_only must be a list of globs",
		"[verify.go.test]\nskip_when_only = [1]\n":         "[verify.go.test].skip_when_only #1 must be a string",
		"[verify.go.test]\nskip_when_only = [\"a/[x\"]\n":  `[verify.go.test].skip_when_only #1 "a/[x" is no glob`,
	} {
		if _, err := parse(t, src); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", src, err, want)
		}
	}
}
```

(Falls `slices` oder `strings` in `schema_test.go` noch nicht importiert sind, ergänzen.) In `internal/verify/effective_test.go` ein Fall, dass die Tabellenform den Schlüssel über das Preset legt und die übrigen Preset-Schlüssel bleiben:

```go
func TestMergeLaysSkipWhenOnlyOverThePreset(t *testing.T) {
	eff := effFor(t, "[verify.go.test]\nskip_when_only = [\"docs/**\"]\n", goOnly)
	lane := eff.Stacks["go"]["test"].Lane
	if !slices.Equal(lane.SkipWhenOnly, []string{"docs/**"}) || lane.Measuring == "" {
		t.Fatalf("%+v", lane)
	}
}
```

In `internal/verify/show_test.go`:

```go
func TestWriteShowPrintsSkipWhenOnly(t *testing.T) {
	var out strings.Builder
	eff := effFor(t, "[verify.go.test]\nskip_when_only = [\"docs/**\"]\n", goOnly)
	WriteShow(&out, eff, []string{"test"}, env(`C:\repo`))
	if !strings.Contains(out.String(), "skip_when_only = [\"docs/**\"]  # config\n") {
		t.Fatalf("%s", out.String())
	}
}
```

- [ ] **Step 3: Failing tests Plan.** In `internal/verify/plan_test.go`:

```go
func skipEnv(root string, changed ...string) PlanEnv {
	e := env(root)
	e.Changed = changed
	return e
}

const skipDocs = "[verify.go.test]\nskip_when_only = [\"docs/**\", \"**/*.md\"]\n"

func TestPlanSkipsALaneWhenEveryChangedPathIsNamed(t *testing.T) {
	req := Request{Kinds: []string{"lint", "test", "coverage"}}
	jobs, err := Plan(effFor(t, skipDocs, goOnly), req, skipEnv(t.TempDir(), "docs/a.go", "README.md", "x/y.md"))
	if err != nil || names(jobs) != "lint/go test/go coverage/go" {
		t.Fatalf("%v %v", err, names(jobs))
	}
	if jobs[0].Pre != "" {
		t.Fatalf("lint names no glob: %+v", jobs[0])
	}
	if jobs[1].Pre != StateNotApplicable || !jobs[1].Skipped || jobs[1].Note != "only skip_when_only paths changed (docs/a.go, +2)" {
		t.Fatalf("test: %+v", jobs[1])
	}
	// coverage keeps its link and inherits; it must not measure the suite itself.
	if cov := jobs[2]; cov.After != 1 || cov.Measure != nil || cov.Pre != "" || !cov.Consumes {
		t.Fatalf("coverage: %+v", cov)
	}
}

func TestPlanRunsALaneWhenOneChangedPathIsNotNamed(t *testing.T) {
	req := Request{Kinds: []string{"test"}}
	for _, changed := range [][]string{{"docs/a.md", "a.go"}, {}, nil} {
		jobs, _ := Plan(effFor(t, skipDocs, goOnly), req, skipEnv(t.TempDir(), changed...))
		if jobs[0].Pre != "" || jobs[0].Skipped {
			t.Errorf("%q: %+v", changed, jobs[0])
		}
	}
}

func TestPlanNeverSkipsInAnEdit(t *testing.T) {
	src := "[verify.go.lint]\nskip_when_only = [\"*.go\"]\n"
	e := skipEnv(t.TempDir(), "a.go")
	jobs, _ := Plan(effFor(t, src, goOnly), Request{Kinds: []string{"lint"}, Scope: ScopeEdit, File: "a.go"}, e)
	if len(jobs) != 1 || jobs[0].Pre != "" {
		t.Fatalf("%+v", jobs)
	}
}

// Off with false is not skipped: coverage measures itself, as before.
func TestPlanKeepsASwitchedOffTestApartFromASkippedOne(t *testing.T) {
	src := "[verify.go]\ntest = false\n"
	jobs, _ := Plan(effFor(t, src, goOnly), Request{Kinds: []string{"test", "coverage"}}, skipEnv(t.TempDir(), "docs/a.md"))
	if jobs[0].Skipped || jobs[1].After != -1 || jobs[1].Measure == nil {
		t.Fatalf("%+v", jobs)
	}
}
```

Und in `internal/verify/run_test.go`, damit die Kette im Lauf hält:

```go
func TestRunHandsASkipOnToTheLaneAfterIt(t *testing.T) {
	pre := job("test/go", -1)
	pre.Pre, pre.Skipped = StateNotApplicable, true
	next := job("coverage/go", 0, "c")
	next.Consumes = true
	f := &fakeStart{answer: ok}
	if out := Run([]Job{pre, next}, opts(f)); out[1].State != StateNotApplicable || len(f.started) != 0 {
		t.Fatalf("%+v %v", out[1], f.started)
	}
}
```

(Dieser Lauftest ist schon vor dem Code grün, weil `inherit` `not-applicable` heute weitergibt; er hält den Vertrag fest und zählt nicht als RED.)

- [ ] **Step 4: Rot.**

Run: `go test ./internal/verify -run "SkipWhenOnly|Skip|NeverSkips|SwitchedOffTestApart" -count=1 -v`
Expected: Schema-Tests `--- FAIL` mit `unknown key "skip_when_only"`; `TestPlanSkipsALaneWhenEveryChangedPathIsNamed` `--- FAIL` an `test: `; die übrigen Plan-Tests grün (sie halten fest, was sich nicht ändern darf).

- [ ] **Step 5: Implementieren Schema, Merge, Show.** In `parseLaneTable` (`internal/verify/schema.go`) neben `needs`:

```go
		case "skip_when_only":
			o.Lane.SkipWhenOnly, err = laneGlobs(table, value)
```

Neue Funktion hinter `laneNeeds`:

```go
// laneGlobs reads skip_when_only: globs relative to the repository root, in
// the syntax of the policy's path rules, each one path.Match can read.
func laneGlobs(table string, value any) ([]string, error) {
	list, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%s.skip_when_only must be a list of globs", table)
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("%s.skip_when_only is empty", table)
	}
	out := make([]string, 0, len(list))
	for i, item := range list {
		s, ok := item.(string)
		if !ok || s == "" {
			return nil, fmt.Errorf("%s.skip_when_only #%d must be a string", table, i+1)
		}
		if _, err := path.Match(s, ""); err != nil {
			return nil, fmt.Errorf("%s.skip_when_only #%d %q is no glob: %v", table, i+1, s, err)
		}
		out = append(out, s)
	}
	return out, nil
}
```

(Import `path` ergänzen.) In `merge` (`internal/verify/effective.go`) vor `return base`:

```go
	if o.Set["skip_when_only"] {
		base.SkipWhenOnly = o.Lane.SkipWhenOnly
	}
```

In `writeLane` (`internal/verify/show.go`) hinter `needs`:

```go
	if len(l.SkipWhenOnly) > 0 {
		key("skip_when_only", quoteList(l.SkipWhenOnly))
	}
```

- [ ] **Step 6: Implementieren Plan.** In `planJob` (`internal/verify/plan.go`) direkt hinter dem Block `if !hasCommand(...) { … }`:

```go
	if note := skippedBy(req, r.Lane, env.Changed); note != "" {
		job.Pre, job.Note, job.Skipped = StateNotApplicable, note, true
		return job, link{}, true, nil
	}
```

Neue Funktion:

```go
// skippedBy is the note of a lane that sits out this run, or "": only a
// commit or a turn end hands changed paths in, and every one of them has to
// match a glob of the lane. A glob that fails to match for an error keeps the
// lane running.
func skippedBy(req Request, lane Lane, changed []string) string {
	if req.Scope != ScopeCheck || len(lane.SkipWhenOnly) == 0 || len(changed) == 0 {
		return ""
	}
	for _, p := range changed {
		if !slices.ContainsFunc(lane.SkipWhenOnly, func(glob string) bool {
			ok, err := pathkey.Glob(glob, p)
			return ok && err == nil
		}) {
			return ""
		}
	}
	note := "only skip_when_only paths changed (" + changed[0]
	if len(changed) > 1 {
		note += fmt.Sprintf(", +%d", len(changed)-1)
	}
	return note + ")"
}
```

(Import `github.com/xidus90/loomux/internal/pathkey`.) In `settle` den Schleifenkopf so ändern, dass ein übersprungener Vorgänger vor dem `continue` für `not-applicable` greift:

```go
		for j, p := range jobs {
			if p.Kind != l.after || p.Stack != job.Stack || p.Area != job.Area {
				continue
			}
			// A lane that sat this run out is still the one this lane reads:
			// it hands its not-applicable on instead of letting a measure
			// step run the suite it skipped.
			if p.Skipped {
				job.After, job.Reads, job.Consumes = j, l.reads, len(needs) > 0
				return nil
			}
			if p.Pre == StateNotApplicable {
				continue
			}
```

- [ ] **Step 7: Grün verify.**

Run: `go test ./internal/verify -count=1`
Expected: PASS.

- [ ] **Step 8: Failing test Commit-Weg.** In `internal/cli/check_test.go`:

```go
// Inside a commit hook a lane sits out a commit of paths it names; by hand,
// without GIT_INDEX_FILE, it runs.
func TestCheckSkipsALaneForACommitOfNamedPaths(t *testing.T) {
	root := goWorld(t)
	put(t, root, ".loomux/config.toml", "[verify.go.test]\nskip_when_only = [\"docs/**\"]\n[verify.go]\ncoverage = false\n")
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"},
		{"config", "commit.gpgsign", "false"}, {"add", "."}, {"commit", "-qm", "init"},
	} {
		gitIn(t, root, args...)
	}
	put(t, root, "docs/a.md", "x")
	gitIn(t, root, "add", "docs")
	seen := stubCheck(t, green)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(root, ".git", "index"))
	code, out, errOut := run("check", "test", "--root", root)
	if code != 0 {
		t.Fatalf("code %d\n%s%s", code, out, errOut)
	}
	if slices.ContainsFunc(*seen, func(s string) bool { return strings.HasPrefix(s, "go test") }) || !strings.Contains(out, "only skip_when_only paths changed (docs/a.md)") {
		t.Fatalf("started %v\n%s", *seen, out)
	}
	t.Setenv("GIT_INDEX_FILE", "")
	*seen = (*seen)[:0]
	run("check", "test", "--root", root)
	if !slices.ContainsFunc(*seen, func(s string) bool { return strings.HasPrefix(s, "go test") }) {
		t.Fatalf("by hand every lane runs: %v", *seen)
	}
}
```

`run(args ...string) (int, string, string)` ist die vorhandene Hilfe in `check_test.go`, `put` steht in `check_arm_hook_test.go`, `gitIn` in `area_test.go`, alle im Paket `cli`.

- [ ] **Step 9: Failing test Turn-Ende.** In `internal/hooks/stop_test.go`:

```go
// After a green turn, a turn that changes only named paths leaves the test
// lane out; the lint still runs.
func TestStopSkipsALaneForATurnOfNamedPaths(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeWorldFile(t, root, ".loomux/config.toml", "[verify.go]\ncoverage = false\n[verify.go.test]\nskip_when_only = [\"docs/**\"]\n")
	if code, errOut := runStop(t, root, s1, greenTools()); code != ExitOK {
		t.Fatalf("first turn: %d %s", code, errOut)
	}
	writeWorldFile(t, root, "docs/a.md", "x")
	var mu sync.Mutex // lanes start their tools from several goroutines
	var started []string
	env := greenTools()
	inner := env.Start
	env.Start = func(s child.Spec) child.Result {
		mu.Lock()
		started = append(started, strings.Join(s.Argv, " "))
		mu.Unlock()
		return inner(s)
	}
	if code, errOut := runStop(t, root, s1, env); code != ExitOK {
		t.Fatalf("second turn: %d %s", code, errOut)
	}
	if slices.ContainsFunc(started, func(s string) bool { return strings.HasPrefix(s, "go test") }) || !slices.ContainsFunc(started, func(s string) bool { return strings.HasPrefix(s, "go vet") }) {
		t.Fatalf("started %v", started)
	}
}
```

Ein zweiter Fall hält fest, dass ohne grünen Baum gegen die Basis verglichen wird: dort ist auch `a.go` geändert, also startet `go test`.

```go
func TestStopRunsALaneWithoutAGreenTreeToCompareTo(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeWorldFile(t, root, ".loomux/config.toml", "[verify.go]\ncoverage = false\n[verify.go.test]\nskip_when_only = [\"docs/**\"]\n")
	writeWorldFile(t, root, "docs/a.md", "x")
	var mu sync.Mutex
	var started []string
	env := greenTools()
	inner := env.Start
	env.Start = func(s child.Spec) child.Result {
		mu.Lock()
		started = append(started, strings.Join(s.Argv, " "))
		mu.Unlock()
		return inner(s)
	}
	if code, errOut := runStop(t, root, s1, env); code != ExitOK {
		t.Fatalf("%d %s", code, errOut)
	}
	if !slices.ContainsFunc(started, func(s string) bool { return strings.HasPrefix(s, "go test") }) {
		t.Fatalf("against the base a.go changed too: %v", started)
	}
}
```

- [ ] **Step 10: Rot.**

Run: `go test ./internal/cli -run TestCheckSkipsALaneForACommitOfNamedPaths -count=1 -v` und `go test ./internal/hooks -run TestStopSkipsALaneForATurnOfNamedPaths -count=1 -v`
Expected: `--- FAIL` mit `started [... go test ...]`.

- [ ] **Step 11: Implementieren Aufrufer.** In `internal/cli/check.go` die Seam-Liste um `checkStaged = gitwork.StagedPaths` ergänzen und hinter dem `if *show { … }`-Block:

```go
	// Inside a commit hook git hands its index in: a lane may sit out a
	// commit of paths it names. By hand there is no index, and every lane
	// runs. An index that does not answer leaves the list empty, too.
	if index := os.Getenv("GIT_INDEX_FILE"); index != "" {
		if changed, err := checkStaged(root, index); err == nil {
			env.Changed = changed
		}
	}
```

In `internal/hooks/stop.go` die Seams um `stopChanged = gitwork.ChangedBetween` ergänzen. `stopTrees` liefert zusätzlich `from`: `(head, tree, from string, idx *StopIndex, err error)`; vor dem `if tree == baseTree` steht

```go
	// What the turn changed is judged against the last green tree, not the
	// base: after the first turn of code the base would always still differ
	// in that code.
	from = state.Green
	if from == "" {
		from = baseTree
	}
```

und jedes `return` reicht `from` mit (beim Fehler `""`). In `RunStop` den Aufruf auf `head, tree, from, idx, err := stopTrees(...)` umstellen und vor `stopPlan`:

```go
	// A list only where git measured both trees; without one every lane runs.
	var changed []string
	if tree != "" && from != "" {
		changed, _ = stopChanged(root, from, tree)
	}
```

und in der `verify.PlanEnv{…}` des Stop-Gates `Changed: changed` setzen.

- [ ] **Step 12: Grün.**

Run: `go test ./internal/verify ./internal/cli ./internal/hooks ./internal/gitwork ./internal/pathkey -count=1`
Expected: PASS.

- [ ] **Step 13: Doku.** In `docs/en/configuration.md`, Tabelle „The keys of the table form“, hinter `needs`:

```markdown
| `skip_when_only` | list | Globs, relative to the repository root and in the syntax of `[policy.paths]`, of paths the lane does not care about. At a commit (inside the pre-commit hook, where git hands in `GIT_INDEX_FILE`) and at a turn end (against the last green tree), a lane whose every changed path matches one of them is `not-applicable` with the note `only skip_when_only paths changed (…)`. A lane that waits for it (`after`) inherits that, so `[verify.go.test] skip_when_only` keeps `coverage` from measuring the suite itself; a run that asks for `coverage` alone needs the key on `coverage`. With no changed path, by hand (`loomux check` without the hook) and in an edit every lane runs. Matching is case-sensitive; a path that differs only in case runs the lane. |
```

In der Tabelle „States and verdict“ in der Zeile `not-applicable` die Spalte „When“ um „, or every changed path matched `skip_when_only`“ ergänzen. Den deutschen Zwilling in `docs/de/configuration.md` gleich ergänzen (Zeile und Satz übersetzt).

- [ ] **Step 14: Mutationsrunde.** Mutanten: Kontrolle; `req.Scope != ScopeCheck` → `false` (stirbt an `TestPlanNeverSkipsInAnEdit`); `len(changed) == 0` → `false` (stirbt an der leeren Liste); `!slices.ContainsFunc` → `slices.ContainsFunc` (Logik auf „irgendein Pfad“, stirbt an `a.go`-Fall); `ok && err == nil` → `ok`; der `p.Skipped`-Zweig in `settle` gestrichen (stirbt an `cov.Measure != nil`); `from = state.Green` → `from = baseTree` (stirbt am Turn-Test); `env.Changed = changed` gestrichen (stirbt am Commit-Test).

- [ ] **Step 15: Commit (mit Task 3).**

```
feat(verify): let a lane sit out a commit or turn end of paths it names

skip_when_only lists globs a lane does not care about. Inside the
pre-commit hook the paths come from the index git hands in, at a turn
end from the tree against the last green one; when every one matches,
the lane is not-applicable and the lanes after it inherit that. By hand
and in an edit every lane runs, as without the key.
```

```bash
git add internal/gitwork/paths.go internal/gitwork/paths_test.go internal/verify internal/cli/check.go internal/cli/check_test.go internal/hooks/stop.go internal/hooks/stop_test.go docs/en/configuration.md docs/de/configuration.md
git commit -F "$S/msg-t4.txt" > "$S/commit-t4.log" 2>&1
```

---

### Task 5: `lock` — eine Lane sperrt Stack und Area je Checkout

**Files:**
- Create: `internal/verify/lanelock.go`, `internal/verify/lanelock_test.go`
- Modify: `internal/verify/schema.go` (`Lane.Lock`, `parseLaneTable`)
- Modify: `internal/verify/effective.go` (`merge`), `internal/verify/show.go` (`writeLane`)
- Modify: `internal/verify/plan.go` (`Job.Lock`, `planJob`)
- Modify: `internal/verify/run.go` (`RunOptions.Caller`, `.Waiting`, `.Sleep`; `commands`)
- Modify: `internal/cli/check.go`, `internal/hooks/stop.go`, `internal/hooks/post_edit.go` (Caller, Waiting)
- Modify: `docs/en/configuration.md`, `docs/de/configuration.md`

**Interfaces:**
- Produces: `Lane.Lock bool`; `Job.Lock string` (Pfad der Sperrdatei, `""` ohne Sperre); `verify.LockPath(root, stack, area string) string`; `RunOptions.Caller string`, `RunOptions.Waiting func(Job, string)`, `RunOptions.Sleep func(time.Duration)` (nil heißt `time.Sleep`).

> **Korrekturen aus der Vorabprüfung (gehen dem Text unten vor).**
> - K5.1: Step 1 legt auch `func WaitingTo(io.Writer) func(Job, string) { return func(Job, string) {} }` als Stub an, sonst baut das Testpaket nicht.
> - K5.2: `lockedJob` baut seinen Pfad selbst: `j.Lock = filepath.Join(root, ".loomux", "state", "locks", "gdscript-root.lock")`, nicht über `LockPath` (gegen den Stub `""` scheitern sonst vier Tests in `holdLock` statt an ihrer Assertion, und `TestRunTakesAFreeLockBesideAStaleHolderFile` schreibt eine `.who` ins Paketverzeichnis). Ebenso `b.Lock` in `TestRunLocksTwoAreasApart`: `filepath.Join(root, ".loomux", "state", "locks", "gdscript-b.lock")`. `LockPath` behält seinen eigenen RED-Test.
> - K5.3: In `TestRunHoldsNoSlotWhileWaitingForTheLock` wartet `Look` begrenzt: `select { case <-waiting: case <-time.After(5 * time.Second): }`, damit Stub-Phase und Mutanten ohne `Waiting` rot werden statt zu hängen.
> - K5.4: RED-Reihenfolge wie K4.1: Schema-Test → RED → Schema-, Merge-, Show-Code → Plan-, Show-, Lauf- und Lock-Tests → RED an den Assertionen → übriger Code. Der cli-Fall `TestCheckSaysWhoHoldsTheLock` läuft vor Step 5 einmal rot.
> - K5.5: Coverage: (a) Test, dass eine Sperre, die nicht zu öffnen ist, `failed` ergibt: an `job.Lock` liegt ein Ordner (`os.MkdirAll(j.Lock, …)`), sodass `TryAcquire` scheitert (der vorhandene `TestRunFailsALockItCannotOpen` erreicht nur den `MkdirAll`-Fehler). (b) Statt der Methode `runner.sleep` setzt `Run` `opt.Sleep = time.Sleep`, wenn es `nil` ist; `takeLock` ruft `r.opt.Sleep(lockTick)`.
> - K5.6: `TestCheckSaysWhoHoldsTheLock`: vor `put` `os.MkdirAll` auf `.loomux`; `check_test.go` importiert `internal/lock` und `internal/verify`, falls noch nicht.
> - K5.7: Doku zusätzlich: `docs/{en,de}/hooks.md` am Abschnitt „**Chain.**“: die Wartezeit auf eine Sperre zählt am Turn-Ende gegen `--budget`.
> - K5.8: Files zusätzlich: `plan_test.go`, `schema_test.go`, `show_test.go`, `effective_test.go`, `internal/cli/check_test.go`, `docs/{en,de}/hooks.md`. `git add` mit expliziten Pfaden (nicht `internal/verify` als Ordner, nicht `stop_test.go`/`post_edit_test.go`, die der Task nicht anfasst). Vor dem Commit `git status --short` lesen: `internal/verify/.who` darf nicht gestaged sein.

- [ ] **Step 1: Stubs.** `Lane.Lock bool` („Lock takes the lane's stack and area for itself in this checkout while its processes run; a second run waits.“), `Job.Lock string`, die drei `RunOptions`-Felder, und in `lanelock.go`:

```go
package verify

// LockPath is the lock file of a stack's area in the checkout at root.
func LockPath(root, stack, area string) string { return "" }
```

- [ ] **Step 2: Failing tests.** `internal/verify/lanelock_test.go`:

```go
package verify

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/lock"
)

func TestLockPathPerStackAndArea(t *testing.T) {
	root := filepath.Join("C:", "repo")
	if got := LockPath(root, "gdscript", "."); got != filepath.Join(root, ".loomux", "state", "locks", "gdscript-root.lock") {
		t.Fatal(got)
	}
	if got := LockPath(root, "go", "a/b"); got != filepath.Join(root, ".loomux", "state", "locks", "go-a_b.lock") {
		t.Fatal(got)
	}
}

// holdLock takes path in this process, as another run would; LockFileEx and
// flock lock per handle, so a second TryAcquire here is refused.
func holdLock(t *testing.T, path, who string) *lock.Handle {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	h, ok, err := lock.TryAcquire(path)
	if err != nil || !ok {
		t.Fatalf("%v %v", ok, err)
	}
	if who != "" {
		if err := os.WriteFile(path+".who", []byte(who), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { h.Release() })
	return h
}

func lockedJob(root string) Job {
	j := job("test/gdscript", -1, "t")
	j.Lock = LockPath(root, "gdscript", ".")
	return j
}

func TestRunTakesAndReleasesTheLaneLock(t *testing.T) {
	root := t.TempDir() // no .loomux/state yet: a fresh clone
	j := lockedJob(root)
	var whoDuring string
	f := &fakeStart{answer: func(child.Spec) child.Result {
		data, _ := os.ReadFile(j.Lock + ".who")
		whoDuring = string(data)
		return child.Result{}
	}}
	o := opts(f)
	o.Caller = "check precommit"
	if out := Run([]Job{j}, o); out[0].State != StateOK {
		t.Fatalf("%+v", out[0])
	}
	if !strings.Contains(whoDuring, "caller=check precommit") || !strings.Contains(whoDuring, "pid=") {
		t.Fatalf("holder file while running: %q", whoDuring)
	}
	if _, err := os.Stat(j.Lock + ".who"); !os.IsNotExist(err) {
		t.Fatalf("holder file left behind: %v", err)
	}
	h, ok, err := lock.TryAcquire(j.Lock)
	if err != nil || !ok {
		t.Fatalf("lock not released: %v %v", ok, err)
	}
	h.Release()
}

func TestRunWaitsForTheLockUntilTheBudgetIsSpent(t *testing.T) {
	root := t.TempDir()
	j := lockedJob(root)
	holdLock(t, j.Lock, "pid=4711\ncaller=check precommit\nsince=2026-10-06T20:00:00Z\n")
	f := &fakeStart{answer: ok}
	c := &clock{t: time.Date(2026, 10, 6, 20, 0, 0, 0, time.UTC)}
	var told []string
	o := opts(f)
	o.Scope, o.Budget, o.Now, o.Sleep = ScopeCheck, time.Minute, c.now, func(time.Duration) {}
	o.Waiting = func(j Job, who string) { told = append(told, j.Name+": "+who) }
	out := Run([]Job{j}, o)
	if out[0].State != StateBudget || len(f.started) != 0 || !strings.Contains(out[0].Output, "held by loomux pid 4711, check precommit") {
		t.Fatalf("%+v", out[0])
	}
	if len(told) != 1 || !strings.HasPrefix(told[0], "test/gdscript: held by loomux pid 4711, check precommit, since ") {
		t.Fatalf("told %q", told)
	}
}

func TestRunWaitsForTheLockUntilTheTimeoutWithoutABudget(t *testing.T) {
	root := t.TempDir()
	j := lockedJob(root)
	holdLock(t, j.Lock, "")
	c := &clock{t: time.Now()}
	o := opts(&fakeStart{answer: ok})
	o.Timeout, o.Now, o.Sleep = time.Minute, c.now, func(time.Duration) {}
	if out := Run([]Job{j}, o); out[0].State != StateTimedOut || !strings.Contains(out[0].Output, "held by another run") {
		t.Fatalf("%+v", out[0])
	}
}

// While one lane waits for its lock it holds no slot: with one slot, the free
// lane is held back until the other is waiting, must then start, and only
// then is the lock let go. A waiting lane that kept its slot would leave the
// free lane unstarted, and the lock would go only after 400 sleeps.
func TestRunHoldsNoSlotWhileWaitingForTheLock(t *testing.T) {
	root := t.TempDir()
	locked := lockedJob(root)
	h := holdLock(t, locked.Lock, "")
	f := &fakeStart{answer: ok}
	o := opts(f)
	o.MaxParallel = 1
	waiting := make(chan struct{})
	o.Waiting = func(Job, string) { close(waiting) }
	o.Look = func(s string) (string, error) {
		if s == "l" {
			<-waiting
		}
		return s, nil
	}
	var once sync.Once
	sleeps := 0
	o.Sleep = func(time.Duration) {
		sleeps++
		f.mu.Lock()
		freeRan := len(f.started) > 0
		f.mu.Unlock()
		if freeRan || sleeps > 400 {
			once.Do(func() { h.Release() })
		}
		time.Sleep(5 * time.Millisecond)
	}
	out := Run([]Job{locked, job("lint/gdscript", -1, "l")}, o)
	if sleeps > 400 || out[0].State != StateOK || out[1].State != StateOK || !slices.Equal(f.started, []string{"l", "t"}) {
		t.Fatalf("sleeps %d, started %v, %+v", sleeps, f.started, out)
	}
}

// A holder file a dead run left beside a free lock is not read.
func TestRunTakesAFreeLockBesideAStaleHolderFile(t *testing.T) {
	root := t.TempDir()
	j := lockedJob(root)
	if err := os.MkdirAll(filepath.Dir(j.Lock), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(j.Lock+".who", []byte("pid=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var told int
	o := opts(&fakeStart{answer: ok})
	o.Waiting = func(Job, string) { told++ }
	if out := Run([]Job{j}, o); out[0].State != StateOK || told != 0 {
		t.Fatalf("%+v told %d", out[0], told)
	}
}

func TestRunLocksTwoAreasApart(t *testing.T) {
	root := t.TempDir()
	a := lockedJob(root)
	b := job("test/gdscript@b", -1, "t")
	b.Lock = LockPath(root, "gdscript", "b")
	holdLock(t, a.Lock, "")
	o := opts(&fakeStart{answer: ok})
	if out := Run([]Job{b}, o); out[0].State != StateOK {
		t.Fatalf("%+v", out[0])
	}
}

func TestRunFailsALockItCannotOpen(t *testing.T) {
	root := t.TempDir()
	// The locks directory is a file: nothing can be made below it.
	put := filepath.Join(root, ".loomux", "state")
	if err := os.MkdirAll(put, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(put, "locks"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if out := Run([]Job{lockedJob(root)}, opts(&fakeStart{answer: ok})); out[0].State != StateFailed {
		t.Fatalf("%+v", out[0])
	}
}
```

Plan-, Schema- und Show-Tests:

```go
// plan_test.go
func TestPlanHandsALockedLaneItsLockFile(t *testing.T) {
	root := t.TempDir()
	src := "[verify.go.test]\nlock = true\n"
	jobs, _ := Plan(effFor(t, src, goOnly), Request{Kinds: []string{"lint", "test"}}, env(root))
	if jobs[0].Lock != "" || jobs[1].Lock != LockPath(root, "go", ".") {
		t.Fatalf("%q %q", jobs[0].Lock, jobs[1].Lock)
	}
}

// schema_test.go
func TestParseLock(t *testing.T) {
	cfg, err := parse(t, "[verify.go.test]\nlock = true\n")
	if err != nil || !cfg.Stacks["go"]["test"].Lane.Lock || !cfg.Stacks["go"]["test"].Set["lock"] {
		t.Fatalf("%v %+v", err, cfg.Stacks["go"]["test"])
	}
	if _, err := parse(t, "[verify.go.test]\nlock = \"yes\"\n"); err == nil || !strings.Contains(err.Error(), "[verify.go.test].lock must be a boolean") {
		t.Fatal(err)
	}
}

// show_test.go
func TestWriteShowPrintsLock(t *testing.T) {
	var out strings.Builder
	WriteShow(&out, effFor(t, "[verify.go.test]\nlock = true\n", goOnly), []string{"test"}, env(`C:\repo`))
	if !strings.Contains(out.String(), "lock = true  # config\n") {
		t.Fatalf("%s", out.String())
	}
}
```

`merge` bekommt einen Fall in `effective_test.go`: `lock = false` aus der Konfiguration schaltet ein `lock = true` des Presets ab, das Teil 3 setzen wird.

```go
func TestMergeSwitchesAPresetLockOff(t *testing.T) {
	off := Override{Lane: Lane{Lock: false}, Set: map[string]bool{"lock": true}}
	if merge(Lane{Lock: true, Commands: []string{"t"}}, off).Lock {
		t.Fatal("lock = false must win over the preset")
	}
	if !merge(Lane{Lock: true}, Override{Set: map[string]bool{}}).Lock {
		t.Fatal("a table without lock keeps the preset's")
	}
}
```

Und `verify.WaitingTo` (Step 4) bekommt seinen Test in `lanelock_test.go`:

```go
func TestWaitingToWritesOneLine(t *testing.T) {
	var out strings.Builder
	WaitingTo(&out)(Job{Name: "test/go"}, "held by another run")
	if out.String() != "test/go: waiting for the lock (held by another run)\n" {
		t.Fatalf("%q", out.String())
	}
}
```

Ein Ende-zu-Ende-Fall für `check` in `internal/cli/check_test.go`:

```go
func TestCheckSaysWhoHoldsTheLock(t *testing.T) {
	root := goWorld(t)
	put(t, root, ".loomux/config.toml", "[verify]\ntimeout = 1\n[verify.go]\ncoverage = false\n[verify.go.test]\nlock = true\n")
	path := verify.LockPath(root, "go", ".")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	h, ok, err := lock.TryAcquire(path)
	if err != nil || !ok {
		t.Fatalf("%v %v", ok, err)
	}
	defer h.Release()
	stubCheck(t, green)
	t.Setenv("GIT_INDEX_FILE", "")
	code, out, errOut := run("check", "test", "--root", root)
	if code != 1 || !strings.Contains(errOut, "test/go: waiting for the lock (held by another run)") || !strings.Contains(out, "test/go: timed-out") {
		t.Fatalf("%d\n%s\n%s", code, out, errOut)
	}
}
```

- [ ] **Step 3: Rot.**

Run: `go test ./internal/verify -run "Lock" -count=1 -v`
Expected: `--- FAIL` an den Assertionen (`LockPath` liefert `""`, `unknown key "lock"`, `Lock` bleibt leer, Lauf ohne Sperre).

- [ ] **Step 4: Implementieren.** `internal/verify/lanelock.go`:

```go
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/lock"
)

// lockTick is how often a lane asks for its lock again, as lock.WaitFree.
const lockTick = 250 * time.Millisecond

// LockPath is the lock file of a stack's area in the checkout at root, named
// as CoverPaths names its files. A linked worktree has its own state
// directory and with it its own lock.
func LockPath(root, stack, area string) string {
	if area == "." {
		area = "root"
	}
	return filepath.Join(root, ".loomux", "state", "locks", stack+"-"+strings.ReplaceAll(area, "/", "_")+".lock")
}

// takeLock takes job's lock and answers how to let it go, or the state the
// lane ends in. It waits outside the process slots, as a lane waits for its
// predecessor, and no longer than the budget or, without one, the timeout.
// The lock file stays empty -- a Windows lock denies reads of its range --,
// so who holds it is written beside it; the lock is the truth, the holder
// file a courtesy for whoever waits.
func (r *runner) takeLock(job Job) (release func(), state State, msg string) {
	if err := os.MkdirAll(filepath.Dir(job.Lock), 0o755); err != nil {
		return nil, StateFailed, err.Error()
	}
	limit, over := r.deadline, StateBudget
	if limit.IsZero() && r.opt.Timeout > 0 {
		limit, over = r.opt.Now().Add(r.opt.Timeout), StateTimedOut
	}
	told := false
	for {
		h, held, err := lock.TryAcquire(job.Lock)
		if err != nil {
			return nil, StateFailed, err.Error()
		}
		if held {
			holder := fmt.Sprintf("pid=%d\ncaller=%s\nsince=%s\n", os.Getpid(), r.opt.Caller, r.opt.Now().UTC().Format(time.RFC3339))
			_ = lock.ReplaceText(job.Lock+".who", holder) // a courtesy; the lock holds without it
			return func() {
				os.Remove(job.Lock + ".who")
				h.Release()
			}, "", ""
		}
		who := holderOf(job.Lock, r.opt.Now())
		if !told && r.opt.Waiting != nil {
			r.opt.Waiting(job, who)
		}
		told = true
		if !limit.IsZero() && !r.opt.Now().Before(limit) {
			return nil, over, "not started: waited for the lock, " + who
		}
		r.sleep(lockTick)
	}
}

// holderOf reads the holder file beside a lock someone holds.
func holderOf(path string, now time.Time) string {
	data, err := os.ReadFile(path + ".who")
	if err != nil {
		return "held by another run"
	}
	fields := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), "="); ok {
			fields[k] = v
		}
	}
	who := "held by loomux pid " + fields["pid"]
	if fields["caller"] != "" {
		who += ", " + fields["caller"]
	}
	if since, err := time.Parse(time.RFC3339, fields["since"]); err == nil {
		who += fmt.Sprintf(", since %ds", int(now.Sub(since).Seconds()))
	}
	return who
}

func (r *runner) sleep(d time.Duration) {
	if r.opt.Sleep != nil {
		r.opt.Sleep(d)
		return
	}
	time.Sleep(d)
}
```

In `commands` (`internal/verify/run.go`) hinter der `Look`-Schleife:

```go
	if job.Lock != "" {
		release, state, msg := r.takeLock(job)
		if release == nil {
			return state, msg
		}
		defer release()
	}
```

Schema: in `parseLaneTable`

```go
		case "lock":
			b, ok := value.(bool)
			if !ok {
				return Override{}, fmt.Errorf("%s.lock must be a boolean", table)
			}
			o.Lane.Lock = b
```

`merge`: `if o.Set["lock"] { base.Lock = o.Lane.Lock }`. `writeLane`: `if l.Lock { key("lock", "true") }`. `planJob` direkt vor `return job, l, true, nil` am Ende:

```go
	if r.Lane.Lock {
		job.Lock = LockPath(env.Root, stack, area)
	}
```

In `lanelock.go` zusätzlich die Hilfe, die alle drei Aufrufer teilen:

```go
// WaitingTo is RunOptions.Waiting for a run that reports on w.
func WaitingTo(w io.Writer) func(Job, string) {
	return func(j Job, who string) { fmt.Fprintf(w, "%s: waiting for the lock (%s)\n", j.Name, who) }
}
```

(Import `io`.)

- [ ] **Step 5: Aufrufer.** `internal/cli/check.go` in `verify.RunOptions{…}`: `Caller: "check " + request, Waiting: verify.WaitingTo(stderr),`. `internal/hooks/stop.go`: `Caller: "hook stop", Waiting: verify.WaitingTo(stderr),`. `internal/hooks/post_edit.go`: `Caller: "hook post-tool-use", Waiting: verify.WaitingTo(stderr),`. Der Ende-zu-Ende-Fall steht für `check` (Step 2); Stop und Post-Edit reichen dieselbe Hilfe mit einem Literal durch.

- [ ] **Step 6: Grün.**

Run: `go test ./internal/verify ./internal/cli ./internal/hooks -count=1`
Expected: PASS.

- [ ] **Step 7: Doku.** `docs/en/configuration.md`, Tabelle der Tabellenschlüssel:

```markdown
| `lock` | boolean | Take the lane's stack and area for itself in this checkout while its processes run, `measure` included: a second run of a lane with the same stack and area waits, holding no `max_parallel` slot, and prints `<lane>: waiting for the lock (held by loomux pid …, <caller>, since …s)`. At a turn end and in an edit the wait counts against the budget and ends as `budget`; in `loomux check` and the pre-commit gate it ends after `[verify].timeout` as `timed-out`. The lock is the operating system's, under `.loomux/state/locks/`, and a dead holder lets it go. For tools that rewrite sources while they measure. |
```

Den deutschen Zwilling in `docs/de/configuration.md` gleich ergänzen.

- [ ] **Step 8: Mutationsrunde.** Mutanten: Kontrolle; `limit.IsZero() && r.opt.Timeout > 0` → `false` (stirbt am Timeout-Test, der dann hängt — darum mit `go test -timeout 60s` fahren und TIMEOUT als getötet zählen); `over` vertauscht; `!told &&` gestrichen (stirbt an `len(told) != 1`); `os.Remove(job.Lock + ".who")` gestrichen; `defer release()` gestrichen (stirbt an „lock not released“); `if r.Lane.Lock` → `if true`.

- [ ] **Step 9: Commit.**

```
feat(verify): lock a lane's stack and area against a second run

lock = true takes an operating-system lock per stack and area under
.loomux/state/locks while the lane's processes run. A second run waits
outside the process slots, says who holds the lock, and gives up at the
budget or, without one, at the timeout. For tools such as Nano Coverage
that rewrite the sources they measure.
```

```bash
git add internal/verify internal/cli/check.go internal/cli/check_test.go internal/hooks/stop.go internal/hooks/stop_test.go internal/hooks/post_edit.go internal/hooks/post_edit_test.go docs/en/configuration.md docs/de/configuration.md
git commit -F "$S/msg-t5.txt" > "$S/commit-t5.log" 2>&1
```

---

### Task 6: `{godot}` — finden und gegen `project.godot` prüfen

**Files:**
- Create: `internal/verify/godot.go`, `internal/verify/godot_test.go`
- Modify: `internal/verify/schema.go` (`Config.Godot`, `parseStack`, Platzhalterprüfung)
- Modify: `internal/verify/presets.go` (`parsePresetStack`: Platzhalterprüfung)
- Modify: `internal/verify/plan.go` (`PlanEnv.Godot`, `planJob`)
- Modify: `internal/cli/check.go`, `internal/hooks/stop.go`, `internal/hooks/post_edit.go` (`PlanEnv.Godot`)
- Modify: `docs/en/configuration.md`, `docs/de/configuration.md`

**Interfaces:**
- Produces:
  - `Config.Godot string` aus `[verify.gdscript] godot`.
  - `PlanEnv.Godot func(dir string) (bin string, pre State, note string)`; nil heißt: eine Lane mit `{godot}` ist `missing-tool`.
  - `verify.GodotFor(root, configured string, start func(child.Spec) child.Result) func(dir string) (string, State, string)` — der Finder eines echten Laufs, ein `--version` je Binary.

> **Korrekturen aus der Vorabprüfung (gehen dem Text unten vor).**
> - K6.1: Step 1 legt auch `type godotSources struct{…}` mit den Feldern aus Step 4 und `func (s godotSources) resolve(dir string) (string, State, string) { return "", "", "" }` als Stub an (so sagt es schon Step 3).
> - K6.2: `TestParseGodotVersion` bekommt die Zeile `"4.7\n": {"", false, false}`, sonst überlebt der Mutant `len(tokens) < 3` → `< 2`.
> - K6.3: Coverage: (a) in `TestGodotPrefersTheConsoleExeOnWindows` ein Fall unter `windows` mit einer gefundenen `.exe` ohne `_console`-Geschwister: das gefundene Binary selbst gilt; (b) in `internal/verify/presets_test.go` ein Fall, dass ein Preset mit `{godot}` in einer `go`-Lane beim Laden abgelehnt wird.
> - K6.4: Mehrere Zeilen `config/features` in `project.godot`: ein Test, dass die erste gilt, gehört dazu (der Mutant „`version == ""`-Wächter gestrichen“ muss sterben).
> - K6.5: Spec-Nachtrag 4: eingesetzte `godotSources` statt eines Fake-Binarys über die Testwelten, so gewollt.
> - K6.6: Doku zusätzlich: in `docs/en/configuration.md` die Aufzählung der Stack-Tabellenschlüssel neben `import_check` (~Zeile 197) nennt `godot`; deutscher Zwilling gleich.
> - K6.7: Files zusätzlich: `schema_test.go`, `plan_test.go`, `presets_test.go`. `git add` mit expliziten Pfaden (nicht `internal/verify` als Ordner).

- [ ] **Step 1: Stubs.** `Config.Godot string`, `PlanEnv.Godot`, und in `godot.go`:

```go
package verify

import "github.com/xidus90/loomux/internal/child"

// GodotFor is the finder of a real run.
func GodotFor(root, configured string, start func(child.Spec) child.Result) func(string) (string, State, string) {
	return func(string) (string, State, string) { return "", "", "" }
}

func parseGodotVersion(out string) (version string, mono, ok bool) { return "", false, false }

func projectGodot(path string) (version string, mono bool, err error) { return "", false, nil }
```

- [ ] **Step 2: Failing tests.** `internal/verify/godot_test.go`:

```go
package verify

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGodotVersion(t *testing.T) {
	for out, want := range map[string]struct {
		version string
		mono, ok bool
	}{
		"4.7.1.stable.mono.official.a13da4feb\n":                 {"4.7", true, true},
		"4.7.1.stable.official.a13da4feb\r\n":                    {"4.7", false, true},
		"4.8.beta1.mono.official.0123abcd\n":                     {"4.8", true, true},
		"WARNING: no audio driver\n4.7.stable.official.abc\n":    {"4.7", false, true},
		"":                                                       {"", false, false},
		"Godot Engine\n":                                         {"", false, false},
	} {
		v, mono, ok := parseGodotVersion(out)
		if v != want.version || mono != want.mono || ok != want.ok {
			t.Errorf("%q: %q %v %v", out, v, mono, ok)
		}
	}
}

func TestProjectGodotReadsFeaturesAndDotnet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "project.godot")
	body := "[application]\nconfig/features=PackedStringArray(\"4.7\", \"Mobile\")\n\n[dotnet]\nproject/assembly_name=\"Kontari\"\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, mono, err := projectGodot(path); v != "4.7" || !mono || err != nil {
		t.Fatalf("%q %v %v", v, mono, err)
	}
	if err := os.WriteFile(path, []byte("config/features=PackedStringArray(\"Forward Plus\")\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := projectGodot(path); err == nil {
		t.Fatal("no version in config/features must be an error")
	}
	if _, _, err := projectGodot(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("a missing project.godot must be an error")
	}
}

// world is a Godot project dir whose project.godot wants version, with
// .NET when mono is set.
func godotWorld(t *testing.T, version string, mono bool) string {
	t.Helper()
	dir := t.TempDir()
	body := "config/features=PackedStringArray(\"" + version + "\")\n"
	if mono {
		body += "[dotnet]\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "project.godot"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// sources answers every outside question from maps: variables, PATH, files
// that exist, and what each binary says to --version.
func sources(root, configured, goos string, vars map[string]string, path map[string]string, files []string, versions map[string]string) godotSources {
	return godotSources{
		root: root, configured: configured, goos: goos,
		getenv: func(k string) string { return vars[k] },
		look: func(name string) (string, error) {
			if p, ok := path[name]; ok {
				return p, nil
			}
			return "", errors.New("not on PATH")
		},
		exists: func(p string) bool {
			for _, f := range files {
				if filepath.Clean(f) == filepath.Clean(p) {
					return true
				}
			}
			return false
		},
		version: func(bin string) (string, error) {
			if v, ok := versions[bin]; ok {
				return v, nil
			}
			return "", errors.New("exit 1")
		},
	}
}

// All three sources set: the variable wins, then the configured path, then
// PATH; a source naming a file that is not there is passed over.
func TestGodotSearchOrder(t *testing.T) {
	dir := godotWorld(t, "4.7", false)
	root := filepath.Join("C:", "repo")
	conf := filepath.Join(root, ".tools", "godot.exe")
	v := "4.7.1.stable.official.x\n"
	all := map[string]string{"/env/godot": v, conf: v, "/path/godot": v}
	vars := map[string]string{"GODOT_BIN": "/env/godot"}
	onPath := map[string]string{"godot": "/path/godot"}
	files := []string{"/env/godot", conf, "/path/godot"}
	for name, row := range map[string]struct {
		files []string
		want  string
	}{
		"variable":   {files, "/env/godot"},
		"configured": {files[1:], conf},
		"path":       {files[2:], "/path/godot"},
	} {
		s := sources(root, ".tools/godot.exe", "linux", vars, onPath, row.files, all)
		if bin, pre, note := s.resolve(dir); bin != row.want || pre != "" {
			t.Errorf("%s: %q %q %q", name, bin, pre, note)
		}
	}
}

func TestGodotPrefersTheConsoleExeOnWindows(t *testing.T) {
	dir := godotWorld(t, "4.7", true)
	gui := `C:\tools\Godot_v4.7.1-stable_mono_win64.exe`
	console := `C:\tools\Godot_v4.7.1-stable_mono_win64_console.exe`
	v := map[string]string{console: "4.7.1.stable.mono.official.x\n"}
	// Through the variable, not the configured path: a configured `C:\…` is
	// no absolute path on the Linux runner and would be joined to the root.
	s := sources(`C:\repo`, "", "windows", map[string]string{"GODOT_BIN": gui}, nil, []string{gui, console}, v)
	if bin, pre, _ := s.resolve(dir); bin != console || pre != "" {
		t.Fatalf("%q %q", bin, pre)
	}
	s.goos = "linux"
	if bin, pre, _ := s.resolve(dir); bin != "" || pre != StateUnready {
		t.Fatalf("not windows: %q %q", bin, pre)
	}
}

func TestGodotJudgesTheBinaryAgainstTheProject(t *testing.T) {
	bin := "/path/godot"
	for name, row := range map[string]struct {
		want    string
		mono    bool
		version string
		pre     State
		note    string
	}{
		"same minor, other patch": {"4.7", false, "4.7.3.stable.official.x\n", "", ""},
		"other minor":             {"4.7", false, "4.8.stable.official.x\n", StateUnready, "Godot 4.8 found (PATH: /path/godot), project.godot wants 4.7"},
		"mono missing":            {"4.7", true, "4.7.1.stable.official.x\n", StateUnready, "wants 4.7 with .NET (mono)"},
		"mono where none needed":  {"4.7", false, "4.7.1.stable.mono.official.x\n", "", ""},
		"no version":              {"4.7", false, "", StateUnready, "gave no version; on Windows name the _console.exe"},
	} {
		dir := godotWorld(t, row.want, row.mono)
		versions := map[string]string{}
		if row.version != "" {
			versions[bin] = row.version
		}
		s := sources("/repo", "", "linux", nil, map[string]string{"godot": bin}, []string{bin}, versions)
		got, pre, note := s.resolve(dir)
		if pre != row.pre || !strings.Contains(note, row.note) || (pre == "" && got != bin) {
			t.Errorf("%s: %q %q %q", name, got, pre, note)
		}
	}
}

func TestGodotWithoutABinaryOrAProject(t *testing.T) {
	dir := godotWorld(t, "4.7", false)
	s := sources("/repo", "", "linux", map[string]string{"GODOT_BIN": "/gone"}, nil, nil, nil)
	if _, pre, note := s.resolve(dir); pre != StateMissingTool || !strings.Contains(note, "GODOT_BIN names /gone, which is not there") {
		t.Fatalf("%q %q", pre, note)
	}
	s = sources("/repo", "", "linux", nil, map[string]string{"godot": "/p/godot"}, []string{"/p/godot"}, map[string]string{"/p/godot": "4.7.stable.official.x"})
	if _, pre, note := s.resolve(t.TempDir()); pre != StateUnready || !strings.Contains(note, "project.godot") {
		t.Fatalf("%q %q", pre, note)
	}
}
```

Schema- und Plan-Tests:

```go
// schema_test.go
func TestParseGodotPathAndPlaceholder(t *testing.T) {
	cfg, err := parse(t, "[verify.gdscript]\ngodot = \".tools/godot.exe\"\n")
	if err != nil || cfg.Godot != ".tools/godot.exe" {
		t.Fatalf("%v %q", err, cfg.Godot)
	}
	for src, want := range map[string]string{
		"[verify.gdscript]\ngodot = \"\"\n":             "[verify.gdscript].godot must be a path",
		"[verify.go]\ngodot = \"x\"\n":                  `[verify.go] has unknown key "godot"`,
		"[verify.go]\ntest = \"{godot} --headless\"\n": "[verify.go].test uses {godot}, which only gdscript lanes know",
	} {
		if _, err := parse(t, src); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v", src, err)
		}
	}
}

// plan_test.go
func TestPlanPutsTheGodotBinaryIn(t *testing.T) {
	facts := detect.Facts{Stacks: []string{"gdscript"}, Areas: map[string][]string{"gdscript": {"game"}}}
	root := t.TempDir()
	src := "[verify.gdscript.test]\ncommands = [\"{godot} --headless -s run.gd\"]\n"
	e := env(root)
	var asked string
	e.Godot = func(dir string) (string, State, string) { asked = dir; return `C:\g\godot_console.exe`, "", "" }
	jobs, err := Plan(effFor(t, src, facts), Request{Kinds: []string{"lint", "test"}}, e)
	if err != nil || jobs[1].Argvs[0][0] != `C:\g\godot_console.exe` || asked != filepath.Join(root, "game") {
		t.Fatalf("%v %+v asked %q", err, jobs, asked)
	}
	e.Godot = func(string) (string, State, string) { return "", StateUnready, "Godot 4.8 found" }
	jobs, _ = Plan(effFor(t, src, facts), Request{Kinds: []string{"test"}}, e)
	if jobs[0].Pre != StateUnready || jobs[0].Note != "Godot 4.8 found" {
		t.Fatalf("%+v", jobs[0])
	}
	e.Godot = nil
	jobs, _ = Plan(effFor(t, src, facts), Request{Kinds: []string{"test"}}, e)
	if jobs[0].Pre != StateMissingTool {
		t.Fatalf("%+v", jobs[0])
	}
	// A lane without {godot} never asks, nor one whose commands name it while
	// this run drives its measuring form, which does not.
	e.Godot = func(string) (string, State, string) { t.Fatal("asked"); return "", "", "" }
	Plan(effFor(t, "", facts), Request{Kinds: []string{"lint"}}, e)
	measured := src + "measuring = \"sh measure.sh {coverprofile}\"\n[verify.gdscript.coverage]\ncommands = [\"report {coverprofile}\"]\nafter = \"test\"\n"
	Plan(effFor(t, measured, facts), Request{Kinds: []string{"test", "coverage"}}, e)
}
```

- [ ] **Step 3: Rot.**

Run: `go test ./internal/verify -run "Godot" -count=1 -v`
Expected: `--- FAIL` an den Assertionen (`godotSources` fehlt zunächst; damit der Lauf an Assertionen scheitert, legt Step 1 auch `type godotSources struct{…}` mit den Feldern oben und `func (s godotSources) resolve(dir string) (string, State, string) { return "", "", "" }` als Stub an).

- [ ] **Step 4: Implementieren.** `internal/verify/godot.go`:

```go
package verify

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/child"
)

// placeholderGodot is the Godot binary a gdscript lane names.
const placeholderGodot = "{godot}"

// godotSources are the outside answers {godot} needs: the variable, the
// configured path, PATH, which files exist, and the binary's own --version.
type godotSources struct {
	root, configured, goos string
	getenv                 func(string) string
	look                   func(string) (string, error)
	exists                 func(string) bool
	version                func(string) (string, error)
}

// GodotFor is the finder of a real run, asking each binary for its version
// once. project.godot is the source of the version a project wants (no
// guessed fallback); see the configuration docs for the search order.
func GodotFor(root, configured string, start func(child.Spec) child.Result) func(string) (string, State, string) {
	var mu sync.Mutex
	seen := map[string][2]string{}
	s := godotSources{
		root: root, configured: configured, goos: runtimeGOOS(),
		getenv: os.Getenv,
		look:   lookPath,
		exists: func(p string) bool { fi, err := os.Stat(p); return err == nil && !fi.IsDir() },
		version: func(bin string) (string, error) {
			mu.Lock()
			defer mu.Unlock()
			if v, ok := seen[bin]; ok {
				return v[0], errorOf(v[1])
			}
			res := start(child.Spec{Argv: []string{bin, "--version"}, Timeout: 30 * time.Second})
			out, msg := res.Stdout, ""
			switch {
			case res.Err != nil:
				msg = res.Err.Error()
			case res.Code != 0:
				msg = "exit " + strconv.Itoa(res.Code)
			}
			seen[bin] = [2]string{out, msg}
			return out, errorOf(msg)
		},
	}
	return s.resolve
}

func errorOf(msg string) error {
	if msg == "" {
		return nil
	}
	return errors.New(msg)
}

// find is the first binary a source names that exists, which source named
// it, and what was looked at when none did.
func (s godotSources) find() (bin, from string, looked []string) {
	if v := s.getenv("GODOT_BIN"); v != "" {
		if s.exists(v) {
			return s.console(v), "GODOT_BIN", nil
		}
		looked = append(looked, "GODOT_BIN names "+v+", which is not there")
	} else {
		looked = append(looked, "GODOT_BIN is unset")
	}
	if s.configured != "" {
		p := filepath.FromSlash(s.configured)
		if !filepath.IsAbs(p) {
			p = filepath.Join(s.root, p)
		}
		if s.exists(p) {
			return s.console(p), "[verify.gdscript] godot", nil
		}
		looked = append(looked, "[verify.gdscript] godot names "+p+", which is not there")
	} else {
		looked = append(looked, "[verify.gdscript] godot is unset")
	}
	for _, name := range []string{"godot", "godot4"} {
		if p, err := s.look(name); err == nil {
			return s.console(p), "PATH", nil
		}
	}
	return "", "", append(looked, "neither godot nor godot4 is on PATH")
}

// console is the _console.exe beside a Windows build when there is one: the
// window build writes nothing to stdout.
func (s godotSources) console(p string) string {
	ext := filepath.Ext(p)
	stem := strings.TrimSuffix(p, ext)
	if s.goos != "windows" || !strings.EqualFold(ext, ".exe") || strings.HasSuffix(strings.ToLower(stem), "_console") {
		return p
	}
	if c := stem + "_console" + ext; s.exists(c) {
		return c
	}
	return p
}

// resolve is PlanEnv.Godot: the binary for the project in dir, or the state
// the lane takes and why.
func (s godotSources) resolve(dir string) (string, State, string) {
	bin, from, looked := s.find()
	if bin == "" {
		return "", StateMissingTool, "no Godot binary: " + strings.Join(looked, "; ")
	}
	want, mono, err := projectGodot(filepath.Join(dir, "project.godot"))
	if err != nil {
		return "", StateUnready, err.Error()
	}
	out, err := s.version(bin)
	got, hasMono, ok := parseGodotVersion(out)
	if err != nil || !ok {
		return "", StateUnready, fmt.Sprintf("%s (%s) gave no version; on Windows name the _console.exe", bin, from)
	}
	if got != want || (mono && !hasMono) {
		need := want
		if mono {
			need += " with .NET (mono)"
		}
		return "", StateUnready, fmt.Sprintf("Godot %s found (%s: %s), project.godot wants %s. Set GODOT_BIN or [verify.gdscript] godot.", got, from, bin, need)
	}
	return bin, "", ""
}

// parseGodotVersion reads `<major>.<minor>[.<patch>].<status>[.mono]…` from
// the first line of out that has that shape.
func parseGodotVersion(out string) (version string, mono, ok bool) {
	for _, line := range strings.Split(out, "\n") {
		tokens := strings.Split(strings.TrimSpace(line), ".")
		if len(tokens) < 3 || !digits(tokens[0]) || !digits(tokens[1]) {
			continue
		}
		return tokens[0] + "." + tokens[1], slices.Contains(tokens, "mono"), true
	}
	return "", false, false
}

func digits(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil && s != ""
}

var featureVersion = regexp.MustCompile(`"(\d+\.\d+)"`)

// projectGodot reads the version a project wants from config/features and
// whether its [dotnet] section asks for the mono build.
func projectGodot(path string) (version string, mono bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", false, fmt.Errorf("%s cannot be read, so there is no Godot version to check against: %w", path, err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[dotnet]" {
			mono = true
		}
		if rest, found := strings.CutPrefix(line, "config/features="); found && version == "" {
			if m := featureVersion.FindStringSubmatch(rest); m != nil {
				version = m[1]
			}
		}
	}
	if version == "" {
		return "", false, fmt.Errorf("%s names no version in config/features", path)
	}
	return version, mono, nil
}
```

`regexp.MustCompile` als Paketvariable ist erlaubt (kein eingebettetes Datum); `runtimeGOOS` und `lookPath` sind zwei Zeilen in derselben Datei: `func runtimeGOOS() string { return runtime.GOOS }` und `var lookPath = exec.LookPath` (Imports `runtime`, `os/exec`). `GodotFor` selbst bekommt in `godot_test.go`:

```go
// One --version per binary and run, whatever the number of areas; a binary
// that fails says so every time without a second start.
func TestGodotForAsksEachBinaryOnce(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "godot")
	if err := os.WriteFile(bin, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GODOT_BIN", bin)
	for _, row := range []struct {
		res  child.Result
		pre  State
	}{
		{child.Result{Stdout: "4.7.1.stable.official.x\n"}, ""},
		{child.Result{Code: 1}, StateUnready},
		{child.Result{Err: errors.New("cannot start")}, StateUnready},
	} {
		starts := 0
		find := GodotFor(t.TempDir(), "", func(s child.Spec) child.Result {
			starts++
			if !slices.Equal(s.Argv, []string{bin, "--version"}) {
				t.Errorf("argv %v", s.Argv)
			}
			return row.res
		})
		dir := godotWorld(t, "4.7", false)
		for range 2 {
			if _, pre, note := find(dir); pre != row.pre {
				t.Errorf("%+v: %q %q", row.res, pre, note)
			}
		}
		if starts != 1 {
			t.Errorf("%+v: %d starts", row.res, starts)
		}
	}
}
```

(Imports `slices` und `github.com/xidus90/loomux/internal/child` in `godot_test.go`.)

Schema (`internal/verify/schema.go`), in `parseStack` neben `import_check`:

```go
		case key == "godot" && stack == "gdscript":
			s, ok := v.(string)
			if !ok || s == "" {
				return errors.New("[verify.gdscript].godot must be a path")
			}
			cfg.Godot = s
```

und hinter der Schleife, vor `checkCycle`:

```go
	if err := checkGodotPlaceholder("verify."+stack, stack, func(kind string) Lane { return lanes[kind].Lane }); err != nil {
		return err
	}
```

mit

```go
// checkGodotPlaceholder refuses {godot} outside gdscript: no other stack
// has a project.godot to check the binary against.
func checkGodotPlaceholder(owner, stack string, lane func(string) Lane) error {
	if stack == "gdscript" {
		return nil
	}
	for _, kind := range Kinds() {
		l := lane(kind)
		if slices.ContainsFunc(slices.Concat(l.Commands, l.OnFile, []string{l.Measure, l.Measuring}), func(c string) bool {
			return strings.Contains(c, placeholderGodot)
		}) {
			return fmt.Errorf("[%s].%s uses %s, which only gdscript lanes know", owner, kind, placeholderGodot)
		}
	}
	return nil
}
```

In `parsePresetStack` (`internal/verify/presets.go`) vor `checkLanes`:

```go
	if err := checkGodotPlaceholder(owner, name, func(kind string) Lane { return stack.Lanes[kind] }); err != nil {
		return PresetStack{}, err
	}
```

Plan (`internal/verify/plan.go`), in `planJob` **hinter** der Zuweisung `if measuring { cmds = []string{r.Lane.Measuring} }` und vor `CoverPaths`: so fragt nur eine Lane, deren tatsächlich gefahrene Befehle oder deren `measure` `{godot}` nennen. `settle` expandiert `measure` mit demselben Replacer, `{godot}` ist darin dann schon gesetzt.

```go
	godot := ""
	if slices.ContainsFunc(slices.Concat(cmds, []string{r.Lane.Measure}), func(c string) bool {
		return strings.Contains(c, placeholderGodot)
	}) {
		if env.Godot == nil {
			job.Pre, job.Note = StateMissingTool, "{godot} has no finder in this run"
			return job, link{}, true, nil
		}
		bin, pre, note := env.Godot(dir)
		if pre != "" {
			job.Pre, job.Note = pre, note
			return job, link{}, true, nil
		}
		godot = bin
	}
```

und in `pairs` `"{godot}", godot` ergänzen.

- [ ] **Step 5: Aufrufer.** `internal/cli/check.go`: `env.Godot = verify.GodotFor(root, eff.Config.Godot, checkStart)` beim Bauen von `env`. `internal/hooks/stop.go`: im `verify.PlanEnv{…}` `Godot: verify.GodotFor(root, eff.Config.Godot, env.Start)`. `internal/hooks/post_edit.go`: in `editJobs` `Godot: verify.GodotFor(root, eff.Config.Godot, env.Start)`. `GateLanes` (`internal/hooks/lanestates.go`) bleibt ohne Finder: eine Lane mit `{godot}` ist dort `missing-tool` und wird damit gelistet, ohne dass `--version` läuft.

- [ ] **Step 6: Grün.**

Run: `go test ./internal/verify ./internal/cli ./internal/hooks -count=1`
Expected: PASS.

- [ ] **Step 7: Doku.** `docs/en/configuration.md`: in der `[verify]`-Schlüsseltabelle hinter `gdscript.import_check`:

```markdown
| `gdscript.godot` | string | The Godot binary `{godot}` names, relative to the root or absolute. Checked after `GODOT_BIN` and before `PATH`. |
```

In der Platzhaltertabelle:

```markdown
| `{godot}` | `gdscript` only, anywhere else a load error. The first of `GODOT_BIN`, `[verify.gdscript] godot` and `godot`/`godot4` on the `PATH` that exists; on Windows the `_console.exe` beside a found `.exe`. Its `--version` must match the major and minor version of `config/features` in the area's `project.godot`, and be a mono build when the project has a `[dotnet]` section; otherwise the lane is `unready` and says which binary it found and what the project wants. Nothing found is `missing-tool`. |
```

Im Preset-Absatz zu GDScript das Beispiel auf `{godot}` umstellen:

```toml
[verify.gdscript.test]
commands = ["{godot} --headless --script tests/run_tests.gd"]
```

und den Satz zum Schreiben des Pfads („Write the path with forward slashes …“) als Hinweis für den Fall stehen lassen, dass ein Projekt den Pfad direkt in den Befehl schreibt. Den deutschen Zwilling gleich ändern.

- [ ] **Step 8: Mutationsrunde.** Mutanten: Kontrolle; Reihenfolge Variable/Konfiguration vertauscht (stirbt an `TestGodotSearchOrder`); `s.goos != "windows"` → `false` (stirbt am Linux-Zweig); `got != want` → `false`; `mono && !hasMono` → `false`; `len(tokens) < 3` → `< 2`; `continue` → `break` (stirbt an der Warnzeile); `version == ""`-Wächter in `projectGodot` gestrichen (die erste Fundstelle gewinnt nicht mehr — ein zweiter Test mit zwei `config/features`-Zeilen ist nötig, falls er überlebt); Cache in `GodotFor` umgangen (stirbt am Zähltest).

- [ ] **Step 9: Commit.**

```
feat(verify): resolve {godot} for gdscript lanes and check it against project.godot

{godot} names the first Godot binary GODOT_BIN, [verify.gdscript] godot
or PATH offers, the console build on Windows. Its --version has to match
the major and minor version of config/features and be a mono build where
the project has a [dotnet] section; otherwise the lane is unready and
says what it found against what the project wants.
```

```bash
git add internal/verify internal/cli/check.go internal/hooks/stop.go internal/hooks/post_edit.go docs/en/configuration.md docs/de/configuration.md
git commit -F "$S/msg-t6.txt" > "$S/commit-t6.log" 2>&1
```

---

### Task 7: Messungen, Roadmap und Endprüfung

**Files:**
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`
- Modify: `README.md`, `README.de.md` (Abschnitt Roadmap)

- [ ] **Step 1: Benchmarks.** Ans Ende von `docs/en/benchmarks.md` (und übersetzt in `docs/de/benchmarks.md`) einen Eintrag im Format der vorhandenen:

```markdown
## 2026-10-06 — What a Lane Could Save by Sitting Out a Commit

**Goal.** Whether letting lanes sit out commits that do not touch them is
worth it, and whether loomux can tell those commits by the stack alone.

**Method.** One warm run of loomux's own gate (`go run ./cmd/loomux check
precommit`, `757d600b`, AMD Ryzen 7 9800X3D, Windows 11 Pro): 3 min 50 s,
`test/go` 226.6 s of it. Listing the changed paths: `git diff --cached
--name-only` and `git diff HEAD --name-only` in `space`, median of 10 runs
with PowerShell `Measure-Command`: 89 ms and 83 ms.
`Godot_v4.7.1-stable_mono_win64_console.exe --version`, median of 5: 57 ms.
The last 400 non-merge commits of 12 repositories, replayed against two
rules: *by stack* skips a lane when no changed path has its stack's
extension; *hybrid* skips only when every path has another stack's
extension and runs on a path no stack claims. *Missed* counts skipped
commits that changed a file a test of that stack demonstrably reads.

| Repository, lane | By stack | Hybrid | Missed |
| --- | ---: | ---: | ---: |
| loomux, go | 39 % | 35 % | 0 |
| space, python | 97 % | 77 % | 83 |
| odysseus, python | 31 % | 29 % | 65 |
| iam_backend, python | 33 % | 25 % | 16 |
| open-design, typescript | 24 % | 15 % | 13 |
| Strata, python | 66 % | 34 % | 1 |

**Result.** A rule by stack would switch off tests that were affected in
five of six repositories checked, so `skip_when_only` names the paths per
lane and nothing is skipped without it. The gate run is one warm run, not a
median.
```

- [ ] **Step 2: Roadmap.** In `README.md` unter „Coming“ vier Zeilen, Priorität `—` (die setzt der Nutzer im PR), und dasselbe übersetzt in `README.de.md`:

```markdown
| **LCOV coverage gate** | `loomux check lcov`: reads and merges LCOV reports, holds each line to a threshold, takes exemption markers with a reason and refutes one on a line that ran, and is red on a report that is missing, empty or older than the run; for any tool that writes LCOV (Nano Coverage, vitest, gcovr) | — | — | — |
| **Godot test suite** | `loomux check godot-suite`: gdUnit4 headless in shards, Nano Coverage instrumented and restored, a stale class cache named with the command that refreshes it, and `test`/`coverage` presets for `gdscript` that use `{godot}` and `lock` | — | LCOV coverage gate | — |
| **Project rules as checks** | Built-in checks for a test module per source, leftover instrumentation, sources outside every checked directory and a core that must not reach the UI, configured per project | — | — | — |
| **space on loomux alone** | `space` drops `godot_quality.py` and its helpers; its pre-commit runs `loomux check precommit` only | — | Godot test suite, Project rules as checks | — |
```

- [ ] **Step 3: Endprüfung.**

Run: `sh ci/gate.sh > "$S/gate-final.log" 2>&1; echo $?` und danach `grep -n -- "--- FAIL\|failed" "$S/gate-final.log"`.
Expected: Exit 0, keine Treffer. Außerdem `grep -rn "matchGlob" internal` leer, und `grep -rln "skip_when_only\|{godot}" docs/en docs/de` nennt `configuration.md` in beiden Sprachen.

- [ ] **Step 4: Commit.**

```
docs: record what sitting out a commit saves and add the Godot follow-ups to the roadmap
```

```bash
git add docs/en/benchmarks.md docs/de/benchmarks.md README.md README.de.md
git commit -F "$S/msg-t7.txt" > "$S/commit-t7.log" 2>&1
```
