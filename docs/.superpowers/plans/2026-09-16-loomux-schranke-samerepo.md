# Schreibschranke: `sameRepository` ohne `git rev-parse` — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `sameRepository` beantwortet „derselbe Repository-Verbund?" aus Gits Zeigerdateien statt über zwei `git rev-parse`-Prozesse, und die Messung zeigt, was das im Worktree spart.

**Architecture:** Eine neue Funktion `registeredCommon` in `internal/brain/guard/worktree.go` steigt vom registrierten Pfad bis zum ersten `.git`-Eintrag und liest dort `repositoryCommon`. `sameRepository` in `path.go` vergleicht `repositoryCommon(candidate)` damit per `pathsEqual`; `askGit`, `gitCommonDir` und ihr Zeitlimit entfallen. Reihenfolge in `Decide` und `declaredWikiRoot` bleibt.

**Tech Stack:** Go 1.27.0 (windows/amd64), Git 2.54 für die Integrationstests, `loomux dev bench-hooks` für die Messung.

**Spec:** `docs/.superpowers/specs/2026-09-16-loomux-schranke-samerepo-design.md`

## Global Constraints

- Arbeitsort: Worktree `C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/cranky-kilby-f5a456`, Branch `claude/cranky-kilby-f5a456`. Jeder Befehl läuft dort, außer wo ein Schritt ausdrücklich etwas anderes sagt.
- Coverage 100 % je Funktion; ein Ausschluss nur mit `//coverage:exempt <reason>` direkt über `func`. Dieser Plan braucht keinen.
- Kein `init()`; keine Paketvariable parst eingebettete Daten. Funktionsvariablen als Testnaht (`readGitFile`) sind erlaubt.
- Code, Bezeichner, Kommentare, Fehlermeldungen, Commit-Nachrichten englisch. Arbeitspapiere unter `docs/.superpowers/` deutsch. `docs/en/` und `docs/de/` gemeinsam pflegen.
- Kommentare begründen auf einer anderen Ebene als die Zeile darunter; jede Behauptung im Kommentar gegen Code oder Messung geprüft.
- Keine neue Ablehnungsmeldung.
- Reihenfolge in `Decide` und in `declaredWikiRoot` nicht anfassen: ein kaputtes Manifest über dem Ziel verweigert weiter auch im Workspace.
- Commits: Autor und Committer ist der Nutzer (Repo-`user.name`/`user.email`); keine `Co-Authored-By`-Zeile, kein Modell genannt. Nachricht aus einer Datei mit `git commit -F <datei>`, nie per Heredoc, kein `--amend`. Vor jedem Commit `git branch --show-current` und `git rev-parse --short HEAD` lesen.
- Commit-Nachrichtendateien liegen in `C:/Users/micro/Documents/#GIT/loomux/.superpowers/sdd/2026-09-16-loomux-schranke-samerepo/` (git-ignoriert).
- Niemand außer dem Menschen pusht.
- Kein Agent schreibt `%LOCALAPPDATA%\loomux\registry.toml` oder eine `.loomux/config.toml`.
- Messungen chronologisch in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`: Datum und Uhrzeit, Gegenstand, Basis gegen Änderung, kalt und warm.
- Der Worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1` wird nicht verändert; er ist Messobjekt.

## Dateistruktur

| Datei | Verantwortung |
|---|---|
| `internal/brain/guard/worktree.go` | neu: `registeredCommon` |
| `internal/brain/guard/worktree_test.go` | neu: Tests für `registeredCommon`, `sameRepository` mit Datei-Fixtures, Zähltest mit Manifest |
| `internal/brain/guard/path.go` | `sameRepository` umgestellt; `askGit`, `gitCommonDir`, `gitCommonDirTimeout` und die Importe `context`, `os/exec`, `time`, `gitenv` entfernt |
| `internal/brain/guard/internal_test.go` | git-Stub-Tests portiert oder entfernt |
| `internal/brain/guard/mutation_test.go` | ein git-Stub-Test portiert |
| `docs/en/benchmarks.md`, `docs/de/benchmarks.md` | neuer Eintrag, Verweis im Eintrag vom 2026-09-15 |
| `README.md`, `README.de.md` | nur falls die Messung die Zeile „Unified Pre-Tool Guard" ändert |

---

### Task 0: Basis-Binary und Mess-Registry

Keine Codeänderung, kein Commit. Muss vor Task 1 laufen, weil `before.exe` den Stand vor jeder Änderung braucht.

**Files:** keine im Repo; `C:/Users/micro/AppData/Local/Temp/loomux-barrier/` entsteht.

**Interfaces:**
- Consumes: nichts.
- Produces: `C:/Users/micro/AppData/Local/Temp/loomux-barrier/before.exe`, `C:/Users/micro/AppData/Local/Temp/loomux-barrier/state/registry.toml`; den notierten Basis-Commit.

- [ ] **Step 1: Codestand gegen die Basis prüfen**

```bash
git diff e4e0dc2 --stat -- cmd internal go.mod go.sum
```
Expected: keine Ausgabe. Kommt etwas, anhalten und melden — dann ist der Arbeitsbaum nicht die Basis.

- [ ] **Step 2: Basis-Commit notieren**

```bash
git rev-parse --short HEAD
```
Expected: ein Hash; für den Benchmark-Eintrag notieren (Basis ist Code-gleich mit `e4e0dc2`).

- [ ] **Step 3: Verzeichnis anlegen**

```bash
mkdir -p "/c/Users/micro/AppData/Local/Temp/loomux-barrier/state"
```
Expected: keine Ausgabe.

- [ ] **Step 4: `before.exe` bauen**

```bash
go build -o "/c/Users/micro/AppData/Local/Temp/loomux-barrier/before.exe" ./cmd/loomux
```
Expected: keine Ausgabe.

- [ ] **Step 5: Mess-Registry schreiben**

Eine Kopie für die Messung, nicht die echte Registry. Sie führt nur den Hauptcheckout als Workspace.
```bash
printf '[[area]]\nscope = "project/loomux"\npath = "C:/Users/micro/Documents/#GIT/loomux"\nworkspace = true\n' > "/c/Users/micro/AppData/Local/Temp/loomux-barrier/state/registry.toml"
```
Expected: keine Ausgabe.

- [ ] **Step 6: Messobjekt prüfen**

```bash
git -C "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1" rev-parse --short HEAD
```
Expected: `e4e0dc2`. Weicht es ab, weitermachen, aber den Stand für den Benchmark-Eintrag notieren.

---

### Task 1: `registeredCommon`

**Files:**
- Modify: `internal/brain/guard/worktree.go` (neue Funktion hinter `repositoryCommon`)
- Test: `internal/brain/guard/worktree_test.go` (neue Tests ans Ende)

**Interfaces:**
- Consumes (bestehend, Paket `guard`): `repositoryCommon(root string) string`, `parents(path string) []string`, `mustResolve(t, path) string`; Testhelfer `fakeRepository(t, base, name) string`, `fakeLinked(t, main, name string, relative bool) string`, `mkdir`, `write`, `posix`.
- Produces: `func registeredCommon(registered string) string` — das aufgelöste gemeinsame Git-Verzeichnis des Repositorys, in dem `registered` liegt; `""`, wenn `registered` nicht existiert, das nächste `.git` darüber nicht verstanden wird, oder keins da ist.

- [ ] **Step 1: Failing tests schreiben**

An `internal/brain/guard/worktree_test.go` anhängen:

```go
func TestARegisteredRootIsItsOwnRepository(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	want := mustResolve(t, filepath.Join(main, ".git"))
	if got := registeredCommon(main); got != want {
		t.Fatalf("registeredCommon(root) = %q, want %q", got, want)
	}
}

func TestARegisteredSubdirectoryClimbsToItsRepository(t *testing.T) {
	// `git rev-parse --git-common-dir` answers from any directory inside a
	// checkout, and a registered area is not always a checkout root.
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	area := filepath.Join(main, "vault", "demo")
	mkdir(t, area)
	want := mustResolve(t, filepath.Join(main, ".git"))
	if got := registeredCommon(area); got != want {
		t.Fatalf("registeredCommon(subdirectory) = %q, want %q", got, want)
	}
}

func TestAMissingRegisteredPathHasNoRepository(t *testing.T) {
	// git cannot start in a directory that is not there; a climb from one
	// would borrow the repository of whatever ancestor still stands.
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	if got := registeredCommon(filepath.Join(main, "gone")); got != "" {
		t.Fatalf("registeredCommon(missing) = %q, want the empty answer", got)
	}
}

func TestARegisteredPathOutsideEveryRepositoryHasNone(t *testing.T) {
	// Assumes no directory above the test's temporary directory carries a
	// `.git`; on 2026-09-16 none from C:\ to %TEMP% did.
	base := t.TempDir()
	plain := filepath.Join(base, "plain")
	mkdir(t, plain)
	if got := registeredCommon(plain); got != "" {
		t.Fatalf("registeredCommon(plain) = %q, want the empty answer", got)
	}
}

func TestARegisteredLinkedWorktreeNamesItsCommonDirectory(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	want := mustResolve(t, filepath.Join(main, ".git"))
	if got := registeredCommon(linked); got != want {
		t.Fatalf("registeredCommon(linked) = %q, want %q", got, want)
	}
}

func TestAnAreaInASubmoduleDoesNotClimbIntoTheSuperproject(t *testing.T) {
	// A submodule's `.git` file points at `<super>/.git/modules/<name>`,
	// which holds neither `gitdir` nor `commondir`. The climb has to stop
	// there: git names the submodule's own directory, and borrowing the
	// superproject's would make its worktrees one repository with an area
	// git keeps apart.
	base := t.TempDir()
	super := fakeRepository(t, base, "super")
	module := filepath.Join(super, "mod")
	mkdir(t, filepath.Join(super, ".git", "modules", "mod"))
	write(t, filepath.Join(module, ".git"), "gitdir: ../.git/modules/mod\n")
	area := filepath.Join(module, "vault")
	mkdir(t, area)
	if got := registeredCommon(area); got != "" {
		t.Fatalf("registeredCommon(in submodule) = %q, want the empty answer", got)
	}
}
```

- [ ] **Step 2: Rot prüfen**

```bash
go test ./internal/brain/guard -run "TestARegistered|TestAMissingRegistered|TestAnAreaInASubmodule" -count=1
```
Expected: FAIL beim Übersetzen mit `undefined: registeredCommon`.

- [ ] **Step 3: Implementieren**

In `internal/brain/guard/worktree.go` direkt hinter `repositoryCommon` einfügen:

```go
// registeredCommon is the common git directory of the repository a
// registered area lies in, or "" where there is none this reading
// understands.
//
// It climbs where the candidate side does not, because the question it
// replaces -- `git rev-parse --git-common-dir` -- answers from any directory
// inside a checkout, and a registered area need not be a checkout root. Two
// limits keep the climb from answering more than git did. It does not start
// from a path that is not there, which git refuses and a climb would pass
// over to whatever ancestor still stands. And it stops at the first `.git`
// of any kind, understood or not: past a submodule's `.git` file lies the
// superproject, whose worktrees git keeps apart from the submodule.
func registeredCommon(registered string) string {
	if _, err := os.Stat(registered); err != nil {
		return ""
	}
	for _, directory := range append([]string{registered},
		parents(registered)...) {
		if _, err := os.Lstat(filepath.Join(directory, ".git")); err == nil {
			return repositoryCommon(directory)
		}
	}
	return ""
}
```

- [ ] **Step 4: Grün prüfen**

```bash
go test ./internal/brain/guard -run "TestARegistered|TestAMissingRegistered|TestAnAreaInASubmodule" -count=1
```
Expected: `ok`.

- [ ] **Step 5: Coverage je Funktion prüfen**

```bash
go test ./internal/brain/guard -count=1 -coverprofile=coverage.out
```
```bash
go tool cover -func=coverage.out | grep registeredCommon
```
Expected: `100.0%`.

- [ ] **Step 6: Commit**

Nachricht in `C:/Users/micro/Documents/#GIT/loomux/.superpowers/sdd/2026-09-16-loomux-schranke-samerepo/task1.txt`:
```
Find the repository a registered area lies in from git's files

registeredCommon climbs from a registered path to the nearest .git and
reads the common directory there, stopping at the first .git of any kind
so that an area inside a submodule never borrows the superproject.
```
```bash
git branch --show-current
```
```bash
git rev-parse --short HEAD
```
```bash
git add internal/brain/guard/worktree.go internal/brain/guard/worktree_test.go
```
```bash
git commit -F "C:/Users/micro/Documents/#GIT/loomux/.superpowers/sdd/2026-09-16-loomux-schranke-samerepo/task1.txt"
```
Expected: Pre-commit-Gate grün, ein Commit mit dem Nutzer als Autor. `coverage.out` nicht committen.

---

### Task 2: `sameRepository` auf den Dateibefund umstellen

**Files:**
- Modify: `internal/brain/guard/path.go` (`sameRepository`; `askGit`, `gitCommonDir`, `gitCommonDirTimeout` löschen; Importe)
- Modify: `internal/brain/guard/internal_test.go`
- Modify: `internal/brain/guard/mutation_test.go`
- Test: `internal/brain/guard/worktree_test.go`

**Interfaces:**
- Consumes: `registeredCommon(registered string) string` aus Task 1; bestehend `repositoryCommon`, `pathsEqual`, `resolvePath`, `readGitFile`, `workspaceRegistry(t, base, path string, more ...string) string`, `fakeRepository`, `fakeLinked`, `adminOf`, `cycleOfTwo`, `run`, `allow`, `deny`, `writeCall`.
- Produces: `func sameRepository(candidate, registered string) bool` mit unveränderter Signatur.

- [ ] **Step 1: Failing tests schreiben**

An `internal/brain/guard/worktree_test.go` anhängen:

```go
func TestAWorktreeIsTheSameRepositoryAsItsRegisteredTree(t *testing.T) {
	// Laid out by hand, so git itself would not recognise either side:
	// only a reading of the files can say yes here.
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	area := filepath.Join(main, "vault", "demo")
	mkdir(t, area)
	if !sameRepository(linked, main) {
		t.Error("a linked worktree is not the same repository as its main checkout")
	}
	if !sameRepository(linked, area) {
		t.Error("a linked worktree is not the same repository as an area inside its main checkout")
	}
}

func TestAnotherCheckoutIsNotTheSameRepository(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	other := fakeRepository(t, base, "other")
	if sameRepository(other, main) {
		t.Fatal("two unrelated checkouts compared as one repository")
	}
}

func TestAWorktreeIsNoRepositoryOfAnAreaOutsideEveryRepository(t *testing.T) {
	// The empty answer of registeredCommon against a real common directory.
	// `gitCommonDir` caught the empty answer explicitly because two empty
	// answers compared equal; here only one side can be empty, and
	// `pathsEqual` has to keep the two apart.
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	plain := filepath.Join(base, "plain")
	mkdir(t, plain)
	if sameRepository(linked, plain) {
		t.Fatal("a worktree matched an area that lies in no repository")
	}
}

func TestAWriteInTheMainCheckoutUnderAManifestReadsNoGitFile(t *testing.T) {
	// The manifest makes `declaredWikiRoot` ask `sameRepository` about the
	// registered directory itself, which has to answer before any file is
	// read.
	old := readGitFile
	t.Cleanup(func() { readGitFile = old })
	readGitFile = func(name string) ([]byte, error) {
		t.Errorf("read %s for a write in the registered checkout", name)
		return old(name)
	}
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	state := workspaceRegistry(t, base, main)
	write(t, filepath.Join(main, ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n")
	allow(t, writeCall(filepath.Join(main, "src", "a.go")), state)
}

func TestACommondirThatLeadsInACircleIsNoCommonDirectory(t *testing.T) {
	tmp := t.TempDir()
	circle := filepath.Join(tmp, "circle")
	cycleOfTwo(t, circle, filepath.Join(tmp, "other"))
	main := fakeRepository(t, tmp, "main")
	linked := fakeLinked(t, main, "linked", false)
	write(t, filepath.Join(adminOf(main, "linked"), "commondir"),
		posix(circle)+"\n")
	// "" is what every other failed reading says, and the caller reads it
	// as "not the same repository".
	if got := repositoryCommon(linked); got != "" {
		t.Errorf("repositoryCommon = %q, want the empty answer", got)
	}
	if sameRepository(linked, main) {
		t.Error("a worktree whose commondir leads in a circle matched")
	}
}
```

- [ ] **Step 2: Rot prüfen**

```bash
go test ./internal/brain/guard -run "TestAWorktreeIsTheSameRepositoryAsItsRegisteredTree" -count=1
```
Expected: FAIL mit `a linked worktree is not the same repository as its main checkout` — git erkennt die handgelegten Fixtures nicht, und `gitCommonDir` antwortet `""`.

- [ ] **Step 3: `sameRepository` umstellen**

In `internal/brain/guard/path.go` den Doc-Kommentar und Rumpf von `sameRepository` ersetzen durch:

```go
// sameRepository is `_same_repository`:
// whether `candidate` is the registered tree itself or a worktree of it.
//
// Two demands, and the second is the one that is easy to miss. Sharing
// the common directory only says "somewhere in this repository", which
// every subdirectory does too -- so a manifest planted in one of them
// would open its own subtree. A worktree root is the one directory that
// carries a `.git` of its own, and `repositoryCommon` answers only there.
//
// Read from git's files instead of asking git, unlike Python (spec
// 2026-09-16-loomux-schranke-samerepo): the two `git rev-parse` calls
// this replaced ran on every write in a linked worktree below a manifest.
// Where git would find a repository these files do not describe -- a
// `--separate-git-dir` checkout, `core.worktree` -- the answer is false,
// which closes the tree rather than opening it.
func sameRepository(candidate, registered string) bool {
	here, hereErr := resolvePath(candidate)
	there, thereErr := resolvePath(registered)
	// Unresolvable is not the same repository. The caller reads a false
	// here as "this manifest declares nothing", which closes the tree
	// rather than opening it.
	if hereErr != nil || thereErr != nil {
		return false
	}
	if pathsEqual(here, there) {
		return true
	}
	common := repositoryCommon(candidate)
	return common != "" && pathsEqual(common, registeredCommon(registered))
}
```

Hinweis zur letzten Zeile: Der Wächter `common != ""` ist nötig. Antworten beide Seiten `""`, ist `pathsEqual("", "")` wahr (`components("")` ist auf beiden Seiten leer), und ein Kandidat mit unverständlichem `.git` gälte als Worktree eines Bereichs außerhalb jedes Repositorys. Eine leere Antwort *nur* von `registeredCommon` braucht dagegen keinen Zweig. Step 6 prüft den Wächter per Mutation.

- [ ] **Step 4: git-Aufruf löschen**

In `internal/brain/guard/path.go` löschen: die Konstante `gitCommonDirTimeout` mit Kommentar, die Variable `askGit` mit Kommentar, die Funktion `gitCommonDir` mit Kommentar. Aus dem Importblock entfernen: `"context"`, `"os/exec"`, `"time"`, `"github.com/xidus90/loomux/internal/gitenv"`.

```bash
go vet ./internal/brain/guard
```
Expected: Fehler nur in den Testdateien (`undefined: askGit`, `undefined: gitCommonDir`); `path.go` selbst übersetzt. Meldet `go vet` einen unbenutzten oder fehlenden Import in `path.go`, dort korrigieren.

- [ ] **Step 5: git-Stub-Tests portieren**

In `internal/brain/guard/internal_test.go`:

1. `TestGitCommonDirRefusesRatherThanGuesses` ganz löschen.
2. `TestGitCommonDirJoinsARelativeAnswerAndKeepsAnAbsoluteOne` ganz löschen. `mustResolve` direkt darunter bleibt.
3. `TestTheRealGitIsAskedWithTheEnvironmentCleaned` ersetzen durch:

```go
func TestAnInheritedGitDirMakesNoTwoTreesOneRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine")
	}
	base := t.TempDir()
	main := filepath.Join(base, "main")
	mkdir(t, main)
	run(t, main, "init")
	run(t, main, "-c", "user.email=t@t", "-c", "user.name=t",
		"commit", "--allow-empty", "-m", "first")
	linked := filepath.Join(base, "linked")
	run(t, main, "worktree", "add", "-b", "side", linked)
	unrelated := filepath.Join(base, "unrelated")
	mkdir(t, unrelated)
	run(t, unrelated, "init")
	// Set after the fixture, and at the registered repository: inherited by
	// a git child, GIT_DIR outranks the directory it is pointed at, so every
	// tree would answer with this common directory and the unrelated one
	// would pass for a worktree. Whoever brings a git call back without
	// `gitenv` fails here.
	t.Setenv("GIT_DIR", filepath.Join(main, ".git"))
	if !sameRepository(linked, main) {
		t.Error("a real linked worktree is not the same repository")
	}
	if sameRepository(unrelated, main) {
		t.Error("an unrelated repository passed for a worktree")
	}
}
```

4. In `TestABrokenPayloadCannotSlipThroughAPanic` die drei Zeilen

```go
	old := askGit
	t.Cleanup(func() { askGit = old })
	askGit = func(string) (string, error) { panic("git blew up") }
```
ersetzen durch
```go
	old := readGitFile
	t.Cleanup(func() { readGitFile = old })
	readGitFile = func(string) ([]byte, error) { panic("git file blew up") }
```

5. `TestGitAnsweringWithACircleIsNoCommonDirectory` ganz löschen (ersetzt durch `TestACommondirThatLeadsInACircleIsNoCommonDirectory` aus Step 1).

In `internal/brain/guard/mutation_test.go` `TestAGitThatCannotAnswerMakesNoTwoTreesOneRepository` ersetzen durch:

```go
func TestAnUnreadableGitFileMakesNoTwoTreesOneRepository(t *testing.T) {
	// Neither side is a repository this reading understands: the planted
	// `.git` names an administration directory that is not there, and the
	// registered tree has no `.git` above it. Both answers are "", and
	// without the emptiness test in `sameRepository` two empty answers
	// would compare equal -- so any directory carrying a `.git` would pass
	// for a worktree of a tree outside every repository.
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	repo := filepath.Join(tmp, "repo")
	mkdir(t, repo)
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"project/demo\"\npath = \""+posix(repo)+"\"\n")
	planted := filepath.Join(tmp, "planted")
	write(t, filepath.Join(planted, ".git"), "gitdir: elsewhere\n")
	write(t, filepath.Join(planted, ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"w\"\n")
	deny(t, writeCall(filepath.Join(planted, "w", "x.md")), state,
		"the registry declares no writable wiki path and no workspace")
}
```

- [ ] **Step 6: Wächter gegen Mutation prüfen**

Den Wächter `common != "" &&` in `sameRepository` vorübergehend entfernen.
```bash
go test ./internal/brain/guard -run "TestAnUnreadableGitFileMakesNoTwoTreesOneRepository" -count=1
```
Expected: FAIL. Bleibt der Test grün, trägt der Wächter nichts, und das ist dem Controller zu melden statt den Wächter still zu streichen. Danach den Wächter wiederherstellen.

- [ ] **Step 7: Ganzes Paket grün**

```bash
go test ./internal/brain/guard -count=1
```
Expected: `ok`. Insbesondere grün: `TestASubdirectoryOfTheRegisteredRepositoryIsNoWorktreeOfIt`, `TestAWorktreeOfTheRegisteredRepositoryDeclaresItsWiki`, `TestAPlantedGitBesideAPlantedManifestBuysNothing`, alle Tests in `worktree_test.go`.

- [ ] **Step 8: Keine Reste**

```bash
git grep -n "askGit\|gitCommonDir" -- internal
```
Expected: keine Ausgabe.

- [ ] **Step 9: Coverage je Funktion**

```bash
go test ./internal/brain/guard -count=1 -coverprofile=coverage.out
```
```bash
go run ./cmd/loomux dev covergate --profile coverage.out
```
Expected: kein Befund. `coverage.out` nicht committen.

- [ ] **Step 10: Pilot-Binary neu bauen**

```bash
go build -o bin/loomux.exe ./cmd/loomux
```
Expected: keine Ausgabe. Die Hooks dieses Checkouts rufen `bin/loomux.exe`.

- [ ] **Step 11: Commit**

Nachricht in `C:/Users/micro/Documents/#GIT/loomux/.superpowers/sdd/2026-09-16-loomux-schranke-samerepo/task2.txt`:
```
Decide sameRepository from git's files instead of git rev-parse

A write in a linked worktree below a manifest naming a registered scope
started two git processes. The candidate now has to be a checkout root by
repositoryCommon, the registered side climbs to its nearest .git, and
askGit and gitCommonDir are gone. Layouts only git's configuration
describes answer false, which closes the tree.
```
```bash
git branch --show-current
```
```bash
git rev-parse --short HEAD
```
```bash
git add internal/brain/guard/path.go internal/brain/guard/internal_test.go internal/brain/guard/mutation_test.go internal/brain/guard/worktree_test.go
```
```bash
git commit -F "C:/Users/micro/Documents/#GIT/loomux/.superpowers/sdd/2026-09-16-loomux-schranke-samerepo/task2.txt"
```
Expected: Pre-commit-Gate grün.

---

### Task 3: Messen und dokumentieren

**Files:**
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`
- Modify (bedingt): `README.md`, `README.de.md`

**Interfaces:**
- Consumes: Commit aus Task 2; `before.exe`, Mess-Registry und Basis-Commit aus Task 0; `testdata/bench/barrier-worktrees.json` unverändert.
- Produces: keine Code-Schnittstelle.

- [ ] **Step 1: `after.exe` bauen**

```bash
go build -o "/c/Users/micro/AppData/Local/Temp/loomux-barrier/after.exe" ./cmd/loomux
```
Expected: keine Ausgabe.

- [ ] **Step 2: Zeitpunkt und Commit notieren**

```bash
date '+%Y-%m-%d %H:%M'
```
```bash
git rev-parse --short HEAD
```
Beides für die Überschrift und den Methodenabsatz notieren.

- [ ] **Step 3: Messen**

```bash
LOOMUX_STATE_DIR="C:/Users/micro/AppData/Local/Temp/loomux-barrier/state" go run ./cmd/loomux dev bench-hooks testdata/bench/barrier-worktrees.json -n 20
```
Expected: vier Zeilen, **alle Exit-Codes `[0]`** — anders als am 2026-09-15 erlaubt schon `before.exe` den Worktree-Write. Weicht ein Exit-Code ab, anhalten und melden. Die ganze Ausgabe in den Bericht.

- [ ] **Step 4: Englischen Eintrag anhängen**

Ans Ende von `docs/en/benchmarks.md`, Zahlen genau aus Step 3; die Lesart schreibt, was die Zahlen zeigen, auch wenn es der Erwartung widerspricht:

```markdown
## <YYYY-MM-DD HH:MM aus Step 2> — sameRepository Without git rev-parse

Repository `loomux`, branch `claude/cranky-kilby-f5a456`, commit `<Hash aus Step 2>`;
measured worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1` at `<Stand aus Task 0 Step 6>`.

**Goal.** Test the attribution the entry of 2026-09-15 15:39 left open: that the
warm gap of about 34 ms between a write in a linked worktree and one in the main
checkout is the two `git rev-parse --git-common-dir` calls `sameRepository` made.
This change reads git's pointer files instead.

**Method.** `loomux dev bench-hooks testdata/bench/barrier-worktrees.json -n 20`:
one cold run per case, then 20 warm ones. `LOOMUX_STATE_DIR` points at a copy of the
registry that registers only the main checkout (`workspace = true`). Binaries:
`before.exe` built from `<Basis-Commit aus Task 0 Step 2>` (code identical to
`e4e0dc2`), `after.exe` built from the commit above, both with Go `<go version>`.
Both binaries allow the worktree write; the case names are those of 2026-09-15.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| <vier Zeilen aus Step 3> |

### Reading

1. **Worktree:** <after gegen before warm und kalt, in ms, mit Spannweiten; ob die
   Lücke zum Hauptcheckout geschlossen ist und wie viel übrig bleibt>.
2. **Main checkout:** <after gegen before; überlappen die Spannweiten, ist es
   Rauschen — `sameRepository` kehrt dort vor jedem Dateizugriff zurück>.
3. **The attribution:** <bestätigt, wenn der Worktree-Write um etwa die frühere
   Lücke fällt; sonst ausdrücklich: die Lücke hat eine andere Ursache, und welche
   Teile übrig bleiben>. Against the target of 72 ms: <Abstand warm>.
```

- [ ] **Step 5: Verweis im älteren englischen Eintrag**

In `docs/en/benchmarks.md`, Eintrag „The Write Barrier in a Linked Worktree", Punkt 3, hinter `This pass did not measure that attribution.` anfügen:
```
   The entry of <YYYY-MM-DD HH:MM> below measures it.
```
Sonst nichts an dem Eintrag ändern.

- [ ] **Step 6: Deutschen Eintrag anhängen**

Ans Ende von `docs/de/benchmarks.md` dieselbe Struktur und dieselben Zahlen auf Deutsch (Dezimalkomma), Überschrift `## <YYYY-MM-DD HH:MM> — sameRepository ohne git rev-parse`, Abschnitte **Ziel.**, **Methode.**, Tabelle mit den Spaltenköpfen des Eintrags vom 2026-09-15 in `docs/de/benchmarks.md`, `### Lesart`.

- [ ] **Step 7: Verweis im älteren deutschen Eintrag**

In `docs/de/benchmarks.md`, Eintrag vom 2026-09-15 15:39, Punkt 3, hinter `Diese Zuordnung hat dieser Durchgang nicht gemessen.` anfügen:
```
   Der Eintrag vom <YYYY-MM-DD HH:MM> unten misst sie.
```

- [ ] **Step 8: README prüfen**

```bash
grep -n "Unified Pre-Tool Guard\|Pre-Tool-Wächter\|Pre-Tool-Guard" README.md README.de.md
```
Die Zeile nennt `<35ms budget; 32–34ms measured on predecessor`. Liegt der Worktree-Write warm nach Step 3 unter 35 ms, in beiden READMEs in derselben Zeile ergänzen: `a write in a linked worktree measured <X> ms warm (<datum>)` bzw. `ein Write in einem verknüpften Worktree gemessen <X> ms warm (<datum>)`. Liegt er darüber, die Zeile unverändert lassen und das im Bericht nennen.

- [ ] **Step 9: Commit**

Nachricht in `C:/Users/micro/Documents/#GIT/loomux/.superpowers/sdd/2026-09-16-loomux-schranke-samerepo/task3.txt`:
```
Measure the write barrier in a worktree without git rev-parse
```
```bash
git branch --show-current
```
```bash
git rev-parse --short HEAD
```
```bash
git add docs/en/benchmarks.md docs/de/benchmarks.md
```
Nur wenn Step 8 sie geändert hat:
```bash
git add README.md README.de.md
```
```bash
git commit -F "C:/Users/micro/Documents/#GIT/loomux/.superpowers/sdd/2026-09-16-loomux-schranke-samerepo/task3.txt"
```
Expected: Pre-commit-Gate grün.

- [ ] **Step 10: Spec-Stand**

In `docs/.superpowers/specs/2026-09-16-loomux-schranke-samerepo-design.md` die Zeile `**Stand:** entworfen, nicht umgesetzt` ersetzen durch `**Stand:** umgesetzt, gemessen <YYYY-MM-DD>`. Commit mit Nachrichtendatei `task3-spec.txt`:
```
Mark the sameRepository spec as implemented
```
