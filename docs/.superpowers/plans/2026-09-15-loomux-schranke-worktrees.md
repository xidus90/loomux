# Schreibschranke: verknüpfte Worktrees — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ein verknüpfter Git-Worktree eines als `workspace = true` registrierten Repos ist ohne eigenen Registry-Eintrag beschreibbar.

**Architecture:** Eine neue Datei `internal/brain/guard/worktree.go` liest Gits Worktree-Verwaltungsdateien (`.git`-Datei, `commondir`, `gitdir`) direkt, ohne `git`-Prozess. `Decide` fragt sie nur für Ziele, die unter keiner Wurzel liegen, und nimmt gefundene Worktree-Wurzeln in die Allow-List auf. Alles andere an der Schranke bleibt.

**Tech Stack:** Go (Werkzeugkette wie installiert, heute 1.27.0), Git 2.54 für den Integrationstest, `loomux dev bench-hooks` für die Messung.

**Spec:** `docs/.superpowers/specs/2026-09-15-loomux-schranke-worktrees-design.md`

## Global Constraints

- Arbeitsort: Worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1`, Branch `barrier-worktrees`. Jeder Befehl der Tasks 1–2 läuft dort (`$LOOMUX=/c/Users/micro/Documents/#GIT/loomux-sdd-1b1`).
- Coverage 100 % je Funktion. Eine Funktion darf nur mit `//coverage:exempt <reason>` direkt über `func` darunter bleiben. Dieser Plan braucht keinen Ausschluss.
- Kein `init()`; keine Paketvariable parst eingebettete Daten. Eine Funktionsvariable als Testnaht wie `askGit` ist erlaubt.
- Code, Bezeichner, Kommentare, Fehlermeldungen und Commit-Nachrichten englisch. Arbeitspapiere unter `docs/.superpowers/` deutsch. Doku unter `docs/en/` und `docs/de/` mit gleichen Dateinamen; `README.md` und `README.de.md` gemeinsam.
- Kommentare begründen auf einer anderen Ebene als die Zeile darunter; jede Behauptung in einem Kommentar ist gegen den Code oder eine Messung geprüft.
- Keine neue Ablehnungsmeldung. Ein Ziel, das kein Worktree öffnet, bekommt die bestehende `lies outside every writable tree`.
- Commits: Autor und Committer sind der Nutzer (`git config user.name` / `user.email` des Repos); keine `Co-Authored-By`-Zeile, kein Modell genannt. Nachricht aus einer Datei mit `git commit -F <datei>`, nie per Heredoc, kein `--amend`. Vor jedem Commit `git branch --show-current` und `git rev-parse --short HEAD` lesen.
- Niemand außer dem Menschen pusht.
- Kein Agent schreibt `%LOCALAPPDATA%\loomux\registry.toml` oder eine `.loomux/config.toml`.
- Messungen chronologisch in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`: Datum und Uhrzeit, was gemessen wurde, Basis gegen Änderung, kalt und warm.
- Commit-Nachrichtendateien liegen im SDD-Workspace `C:/Users/micro/Documents/#GIT/loomux/.superpowers/sdd/2026-09-15-loomux-schranke-worktrees/` (git-ignoriert).

## Dateistruktur

| Datei | Verantwortung |
|---|---|
| `internal/brain/guard/worktree.go` (neu) | Erkennung: `linkedWorktreeRoots`, `linkedWorktreeRoot`, `repositoryCommon`, `linkedCommon`, `pointer`, `resolvedOrEmpty`, Testnaht `readGitFile` |
| `internal/brain/guard/worktree_test.go` (neu) | Fixtures von Hand, Decide-Tests je Zweig, Integration mit echtem Git |
| `internal/brain/guard/guard.go` (ändern) | Einbau in `Decide` zwischen der Rückkehr bei leerem `outside` und `reviewCentre` |
| `testdata/bench/barrier-worktrees.json`, `testdata/bench/write-in-worktree.json`, `testdata/bench/edit-readme-main.json` (neu) | Messfälle |
| `docs/en/benchmarks.md`, `docs/de/benchmarks.md` (ändern) | Messeintrag |
| `docs/en/getting-started.md`, `docs/de/getting-started.md` (ändern) | Hinweis zur Registry und zu Worktrees |
| `README.md`, `README.de.md` (ändern) | ein Satz in der Zeile des Pre-Tool-Wächters |
| `docs/.superpowers/parity/schranke-worktrees.md` (neu) | eine Paritätszeile |

---

### Task 0: Arbeitsort (Controller und Mensch)

**Files:** keine im Repo. Außerhalb: Branch-Umbenennung im Worktree, ein Registry-Block (nur Mensch), ein Basis-Binary unter `C:/Users/micro/AppData/Local/Temp/loomux-barrier/`.

**Interfaces:** keine.

Die Claude-Sitzung bleibt im Hauptcheckout. Die Hooks rufen dann `C:/Users/micro/Documents/#GIT/loomux/bin/loomux.exe`, das Binary von `master` ohne diese Änderung; es erlaubt Writes im Worktree nur mit dem Registry-Block aus Step 3. Ein Sitzungswechsel ist für diese Stufe nicht nötig.

- [ ] **Step 1: Stand prüfen**

```bash
git -C "/c/Users/micro/Documents/#GIT/loomux" status --short --branch
```
Expected: erste Zeile `## master...origin/master [ahead N]`, sonst keine Zeile.
```bash
git -C "/c/Users/micro/Documents/#GIT/loomux-sdd-1b1" status --short --branch
```
Expected: `## sdd-1b-1`, sonst keine Zeile.

- [ ] **Step 2: Branch umbenennen und auf `master` bringen**

```bash
git -C "/c/Users/micro/Documents/#GIT/loomux-sdd-1b1" branch -m sdd-1b-1 barrier-worktrees
```
Expected: keine Ausgabe.
```bash
git -C "/c/Users/micro/Documents/#GIT/loomux-sdd-1b1" merge --ff-only master
```
Expected: `Fast-forward` oder `Already up to date.`; danach:
```bash
test "$(git -C "/c/Users/micro/Documents/#GIT/loomux-sdd-1b1" rev-parse HEAD)" = "$(git -C "/c/Users/micro/Documents/#GIT/loomux" rev-parse master)"; echo "exit=$?"
```
Expected: `exit=0`.

- [ ] **Step 3: Registry-Eintrag (nur Mensch)**

Der Mensch hängt an `C:\Users\micro\AppData\Local\loomux\registry.toml` an:
```toml
[[area]]
scope     = "project/loomux-sdd-1b1"
path      = "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1"
workspace = true
```
Probe (liest nur):
```bash
printf '%s' '{"session_id":"probe","hook_event_name":"PreToolUse","cwd":"C:/Users/micro/Documents/#GIT/loomux-sdd-1b1","tool_name":"Write","tool_input":{"file_path":"C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/internal/probe.go","content":"x"}}' | "/c/Users/micro/Documents/#GIT/loomux/bin/loomux.exe" hook pre-tool-use --host claude --root "C:/Users/micro/Documents/#GIT/loomux"; echo "exit=$?"
```
Expected: nur `exit=0`. Kommt `lies outside every writable tree`, steht der Block nicht in der Datei.

- [ ] **Step 4: Basis-Binary und Pilot-Binary bauen**

Ab hier mit `$LOOMUX` als Arbeitsverzeichnis.
```bash
mkdir -p "/c/Users/micro/AppData/Local/Temp/loomux-barrier/state"
```
```bash
go build -o "/c/Users/micro/AppData/Local/Temp/loomux-barrier/before.exe" ./cmd/loomux
```
Expected: keine Ausgabe. Das ist der Stand vor jeder Änderung, die Basis der Messung in Task 2.
```bash
go build -o bin/loomux.exe ./cmd/loomux
```
Expected: keine Ausgabe.

- [ ] **Step 5: Tor grün**

```bash
git branch --show-current
```
Expected: `barrier-worktrees`.
```bash
sh .githooks/pre-commit
```
Expected: Exit 0, kein `FAIL`, kein `not covered:`.

- [ ] **Step 6: SDD-Workspace**

Der Controller führt `scripts/sdd-workspace docs/.superpowers/plans/2026-09-15-loomux-schranke-worktrees.md` aus dem Hauptcheckout aus und legt das Ledger an.

Paritätszeilen: keine.

---

### Task 1: Erkennung und Einbau in `Decide`

**Files:**
- Create: `internal/brain/guard/worktree.go`
- Create: `internal/brain/guard/worktree_test.go`
- Modify: `internal/brain/guard/guard.go` (in `Decide`, direkt nach `if len(outside) == 0 { return "", false }`, vor `centre := reviewCentre(areas, stateDir)`)

**Interfaces:**
- Consumes (bestehend, Paket `guard`): `type area struct { scope, path, wikiPath string; workspace, readOnly bool; … }`, `parents(path string) []string`, `resolvePath(target string) (string, error)`, `pathsEqual(left, right string) bool`, `inside(resolved string, roots []string) bool`; Testhelfer `mkdir`, `write`, `posix`, `writeCall`, `allow`, `deny`, `mustResolve`, `registryOf`, `cycleOfTwo`, `run` aus `decide_test.go` und `internal_test.go`.
- Produces: `linkedWorktreeRoots(targets []string, areas []area) []string` (von `Decide` gerufen); `var readGitFile = os.ReadFile`.

- [ ] **Step 1: Fixtures und der erste fehlschlagende Test**

`internal/brain/guard/worktree_test.go`:

```go
package guard

import (
	"path/filepath"
	"testing"
)

// fakeRepository lays out the one part of a main checkout this reading
// looks at: a `.git` directory.
func fakeRepository(t *testing.T, base, name string) string {
	t.Helper()
	main := filepath.Join(base, name)
	mkdir(t, filepath.Join(main, ".git"))
	return main
}

// fakeLinked lays out by hand what `git worktree add` leaves on disk for a
// linked worktree `name` beside `main`, as Git 2.54 wrote it on 2026-09-15:
// a `.git` file naming the administration directory, and there `commondir`
// and a `gitdir` pointing back. `relative` spells all three pointers the
// way `worktree.useRelativePaths` does.
func fakeLinked(t *testing.T, main, name string, relative bool) string {
	t.Helper()
	linked := filepath.Join(filepath.Dir(main), name)
	admin := filepath.Join(main, ".git", "worktrees", name)
	gitdir := admin
	commondir := filepath.Join(main, ".git")
	back := filepath.Join(linked, ".git")
	if relative {
		gitdir = filepath.Join("..", filepath.Base(main), ".git", "worktrees", name)
		commondir = filepath.Join("..", "..")
		back = filepath.Join("..", "..", "..", "..", name, ".git")
	}
	write(t, filepath.Join(linked, ".git"), "gitdir: "+posix(gitdir)+"\n")
	write(t, filepath.Join(admin, "commondir"), posix(commondir)+"\n")
	write(t, filepath.Join(admin, "gitdir"), posix(back)+"\n")
	return linked
}

// adminOf is the administration directory fakeLinked wrote for `name`.
func adminOf(main, name string) string {
	return filepath.Join(main, ".git", "worktrees", name)
}

// workspaceRegistry registers `path` as the workspace `project/demo`, with
// whatever further `[[area]]` blocks the case adds.
func workspaceRegistry(t *testing.T, base, path string, more ...string) string {
	t.Helper()
	state := filepath.Join(base, "state")
	body := "[[area]]\nscope = \"project/demo\"\npath = \"" + posix(path) +
		"\"\nworkspace = true\n"
	for _, area := range more {
		body += "\n" + area
	}
	write(t, filepath.Join(state, "registry.toml"), body)
	return state
}

func TestALinkedWorktreeOfAWorkspaceMayBeWritten(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	state := workspaceRegistry(t, base, main)
	allow(t, writeCall(filepath.Join(linked, "src", "a.go")), state)
}
```

- [ ] **Step 2: Test laufen lassen, er muss scheitern**

```bash
go test ./internal/brain/guard -run 'TestALinkedWorktreeOfAWorkspaceMayBeWritten' -count=1
```
Expected: `FAIL` mit `expected allowed, refused with "… lies outside every writable tree; …"`.

- [ ] **Step 3: Minimale Umsetzung**

`internal/brain/guard/worktree.go`:

```go
package guard

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// readGitFile reads one of git's pointer files. A variable so that a test
// can hold that a write the registry already opens reads none of them;
// nothing outside a test writes it.
var readGitFile = os.ReadFile

// linkedWorktreeRoots is the root of every linked git worktree that holds
// one of `targets` and belongs to the repository of a workspace area.
//
// Not in the Python barrier, where a worktree needed a registry entry of its
// own (spec 2026-09-15-loomux-schranke-worktrees). A linked worktree is the
// registered repository checked out a second time, so what `workspace` opens
// in one it opens in the other. Git is read from its files instead of being
// asked: `git rev-parse` took a 42 ms median on this machine, against a whole
// hook of 24.5 ms.
func linkedWorktreeRoots(targets []string, areas []area) []string {
	commons := []string{}
	for _, registered := range areas {
		if !registered.workspace {
			continue
		}
		if common := repositoryCommon(registered.path); common != "" {
			commons = append(commons, common)
		}
	}
	found := []string{}
	for _, target := range targets {
		if root := linkedWorktreeRoot(target, commons); root != "" {
			found = append(found, root)
		}
	}
	return found
}

// linkedWorktreeRoot is the nearest directory above `target` that is a
// linked worktree sharing one of `commons`, or "".
//
// It climbs past a directory that does not match instead of stopping there:
// a repository nested in a worktree lies below the worktree's root, just as
// the same path in the main checkout lies below the registered tree.
func linkedWorktreeRoot(target string, commons []string) string {
	for _, directory := range parents(target) {
		common := linkedCommon(directory)
		if common == "" {
			continue
		}
		if slices.ContainsFunc(commons, func(registered string) bool {
			return pathsEqual(registered, common)
		}) {
			return directory
		}
	}
	return ""
}

// repositoryCommon is the common git directory of the repository checked out
// at `root`: its `.git` directory, or the one its `.git` file leads to where
// the registration names a linked worktree itself. "" where `root` has
// neither.
func repositoryCommon(root string) string {
	dotGit := filepath.Join(root, ".git")
	if info, err := os.Stat(dotGit); err == nil && info.IsDir() {
		return resolvedOrEmpty(dotGit)
	}
	return linkedCommon(root)
}

// linkedCommon is the common git directory of the linked worktree rooted at
// `directory`, or "" where it is none.
//
// The back pointer is what makes a planted `.git` file worthless: a copy of a
// real worktree's pointer names an administration directory whose `gitdir`
// leads to the real worktree, not to the copy. A submodule and a
// `--separate-git-dir` checkout carry a `.git` file as well, but no
// `commondir`, and fall out on that.
func linkedCommon(directory string) string {
	dotGit := filepath.Join(directory, ".git")
	admin := pointer(dotGit, "gitdir: ", directory)
	if admin == "" {
		return ""
	}
	back := pointer(filepath.Join(admin, "gitdir"), "", admin)
	if back == "" || !pathsEqual(back, resolvedOrEmpty(dotGit)) {
		return ""
	}
	return pointer(filepath.Join(admin, "commondir"), "", admin)
}

// pointer is the resolved path a git pointer file names, counted from `base`
// where it is relative. "" where the file cannot be read, lacks `prefix`, or
// names a path that does not resolve: each one means "not a worktree", and
// that answer keeps the tree shut.
func pointer(file, prefix, base string) string {
	data, err := readGitFile(file)
	if err != nil {
		return ""
	}
	named, found := strings.CutPrefix(
		strings.TrimRight(string(data), " \t\r\n"), prefix)
	if !found {
		return ""
	}
	if !filepath.IsAbs(named) {
		named = filepath.Join(base, named)
	}
	return resolvedOrEmpty(named)
}

// resolvedOrEmpty is `resolvePath` where an error and no answer mean the
// same thing: not this repository.
func resolvedOrEmpty(path string) string {
	resolved, err := resolvePath(path)
	if err != nil {
		return ""
	}
	return resolved
}
```

In `internal/brain/guard/guard.go`, in `Decide`, zwischen diesen beiden bestehenden Stellen:

```go
	if len(outside) == 0 {
		return "", false
	}
	// Only now, for the same reason: finding the review centre reads a
```

wird eingefügt (nach der schließenden Klammer, vor dem Kommentar `// Only now, …`):

```go
	// A linked worktree of a workspace is asked about only here, for the
	// targets nothing else opened: a write inside a registered tree has
	// already returned above and reads no git file. The zones were decided
	// before, so a worktree root cannot reopen one, and "no root at all"
	// cannot change -- a worktree root needs a workspace area, which is a
	// root itself.
	linked := linkedWorktreeRoots(outside, areas)
	roots = append(roots, linked...)
	outside = slices.DeleteFunc(outside, func(path string) bool {
		return inside(path, linked)
	})
	if len(outside) == 0 {
		return "", false
	}
```

- [ ] **Step 4: Test laufen lassen, er muss bestehen**

```bash
go test ./internal/brain/guard -run 'TestALinkedWorktreeOfAWorkspaceMayBeWritten' -count=1
```
Expected: `ok`.

- [ ] **Step 5: Die Zweige der Erkennung als Tests**

Den Importblock von `internal/brain/guard/worktree_test.go` auf diese Pakete erweitern — die Tests unten brauchen sie, und Go lehnt ungenutzte Importe ab, darum stehen sie nicht schon in Step 1:

```go
import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)
```

Dann an die Datei anhängen:

```go
func TestRelativePointersAreFollowedAsGitWritesThem(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", true)
	state := workspaceRegistry(t, base, main)
	allow(t, writeCall(filepath.Join(linked, "src", "a.go")), state)
}

func TestAWorktreeOfAnAreaWithoutWorkspaceStaysShut(t *testing.T) {
	tmp := t.TempDir()
	main := fakeRepository(t, tmp, "repo")
	linked := fakeLinked(t, main, "linked", false)
	// registryOf names <tmp>/repo with a wiki and no `workspace`.
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	deny(t, writeCall(filepath.Join(linked, "src", "a.go")), state,
		"lies outside every writable tree")
}

func TestAWorktreeOfAnUnregisteredRepositoryStaysShut(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	other := fakeRepository(t, base, "other")
	foreign := fakeLinked(t, other, "foreign", false)
	state := workspaceRegistry(t, base, main)
	deny(t, writeCall(filepath.Join(foreign, "a.go")), state,
		"lies outside every writable tree")
}

func TestAWorkspaceWithoutGitOpensNoWorktree(t *testing.T) {
	base := t.TempDir()
	plain := filepath.Join(base, "plain")
	mkdir(t, plain)
	other := fakeRepository(t, base, "other")
	foreign := fakeLinked(t, other, "foreign", false)
	state := workspaceRegistry(t, base, plain)
	deny(t, writeCall(filepath.Join(foreign, "a.go")), state,
		"lies outside every writable tree")
}

func TestABorrowedGitFileOpensNothing(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	borrowed, err := os.ReadFile(filepath.Join(linked, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	planted := filepath.Join(base, "planted")
	write(t, filepath.Join(planted, ".git"), string(borrowed))
	state := workspaceRegistry(t, base, main)
	// The pointer is genuine and so is the administration directory; only
	// its `gitdir` leads to the real worktree and not to the copy.
	deny(t, writeCall(filepath.Join(planted, "a.go")), state,
		"lies outside every writable tree")
	allow(t, writeCall(filepath.Join(linked, "a.go")), state)
}

func TestAMissingBackPointerOpensNothing(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	if err := os.Remove(filepath.Join(adminOf(main, "linked"), "gitdir")); err != nil {
		t.Fatal(err)
	}
	state := workspaceRegistry(t, base, main)
	deny(t, writeCall(filepath.Join(linked, "a.go")), state,
		"lies outside every writable tree")
}

func TestWithoutCommondirTheGitFileIsNoLinkedWorktree(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	// The shape a submodule has: a `.git` file and an administration
	// directory, but no `commondir` in it.
	if err := os.Remove(filepath.Join(adminOf(main, "linked"), "commondir")); err != nil {
		t.Fatal(err)
	}
	state := workspaceRegistry(t, base, main)
	deny(t, writeCall(filepath.Join(linked, "a.go")), state,
		"lies outside every writable tree")
}

func TestAGitFileWithoutThePrefixIsNoPointer(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	write(t, filepath.Join(linked, ".git"), posix(adminOf(main, "linked"))+"\n")
	state := workspaceRegistry(t, base, main)
	deny(t, writeCall(filepath.Join(linked, "a.go")), state,
		"lies outside every writable tree")
}

func TestARegisteredLinkedWorktreeOpensItsSiblings(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	first := fakeLinked(t, main, "first", false)
	second := fakeLinked(t, main, "second", false)
	state := workspaceRegistry(t, base, first)
	allow(t, writeCall(filepath.Join(second, "a.go")), state)
}

func TestTheSearchClimbsPastANestedRepository(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	nested := filepath.Join(linked, "vendor", "nested")
	mkdir(t, filepath.Join(nested, ".git"))
	state := workspaceRegistry(t, base, main)
	allow(t, writeCall(filepath.Join(nested, "x.go")), state)
}

func TestAReadonlyZoneInsideAWorktreeOutranksIt(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	zoneRepo := filepath.Join(base, "zone")
	mkdir(t, zoneRepo)
	wiki := filepath.Join(linked, "docs", "wiki")
	state := workspaceRegistry(t, base, main,
		"[[area]]\nscope = \"project/zone\"\npath = \""+posix(zoneRepo)+
			"\"\nwiki = \""+posix(wiki)+"\"\nreadonly = true\n")
	deny(t, writeCall(filepath.Join(wiki, "x.md")), state,
		"the registration calls this area read-only")
}

func TestAMixedCallNamesTheWorktreeAmongWhatIsAllowed(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	state := workspaceRegistry(t, base, main)
	outsidePath := filepath.Join(base, "outside", "n.ipynb")
	payload := map[string]any{
		"tool_name": "Write",
		"tool_input": map[string]any{
			"file_path":     filepath.Join(linked, "a.go"),
			"notebook_path": outsidePath,
		},
	}
	reason := deny(t, payload, state, "lies outside every writable tree")
	if !strings.HasPrefix(reason, mustResolve(t, outsidePath)+" lies outside") {
		t.Fatalf("the refusal names more than the outside target: %q", reason)
	}
	if !strings.Contains(reason, mustResolve(t, linked)) {
		t.Fatalf("the worktree root is not among the allowed trees: %q", reason)
	}
}

func TestAWriteInsideARegisteredTreeReadsNoGitFile(t *testing.T) {
	old := readGitFile
	t.Cleanup(func() { readGitFile = old })
	readGitFile = func(name string) ([]byte, error) {
		t.Errorf("read %s for a write the registry already opens", name)
		return old(name)
	}
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	state := workspaceRegistry(t, base, main)
	allow(t, writeCall(filepath.Join(main, "src", "a.go")), state)
}

func TestAPointerThatCannotBeResolvedIsNoPointer(t *testing.T) {
	tmp := t.TempDir()
	first := filepath.Join(tmp, "a")
	cycleOfTwo(t, first, filepath.Join(tmp, "b"))
	if got := resolvedOrEmpty(filepath.Join(first, "x")); got != "" {
		t.Fatalf("a path through a cycle resolved to %q", got)
	}
}

func TestARealLinkedWorktreeOfAWorkspaceMayBeWritten(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine")
	}
	base := t.TempDir()
	main := filepath.Join(base, "main")
	mkdir(t, main)
	run(t, main, "init")
	run(t, main, "commit", "--allow-empty", "-m", "first")
	absolute := filepath.Join(base, "absolute")
	run(t, main, "worktree", "add", "-b", "absolute", absolute)
	relative := filepath.Join(base, "relative")
	run(t, main, "-c", "worktree.useRelativePaths=true",
		"worktree", "add", "-b", "relative", relative)
	other := filepath.Join(base, "other")
	mkdir(t, other)
	run(t, other, "init")

	state := workspaceRegistry(t, base, main)
	allow(t, writeCall(filepath.Join(absolute, "internal", "probe.go")), state)
	allow(t, writeCall(filepath.Join(relative, "internal", "probe.go")), state)
	deny(t, writeCall(filepath.Join(other, "a.go")), state,
		"lies outside every writable tree")
}
```

- [ ] **Step 6: Paket laufen lassen**

```bash
go test ./internal/brain/guard -count=1
```
Expected: `ok`. Schlägt ein neuer Test fehl, ist die Umsetzung aus Step 3 oder die Fixture falsch, nicht die Erwartung — Erwartungen folgen der Spec. Die bestehenden Tests `TestAWorktreeOfTheRegisteredRepositoryDeclaresItsWiki`, `TestAGitThatCannotAnswerMakesNoTwoTreesOneRepository` und `TestASubdirectoryOfTheRegisteredRepositoryIsNoWorktreeOfIt` registrieren kein `workspace` und müssen unverändert grün bleiben.

- [ ] **Step 7: Coverage je Funktion prüfen**

```bash
go test ./... -count=1 -covermode=set -coverpkg=./... -coverprofile=coverage.out
```
Expected: kein `FAIL`.
```bash
go run ./cmd/loomux dev covergate --profile coverage.out
```
Expected: Exit 0, keine Zeile `not covered:`. Nennt `covergate` eine der neuen Funktionen, fehlt der Test für ihren Zweig — Test ergänzen, kein `//coverage:exempt`.

- [ ] **Step 8: gofmt und vet**

```bash
gofmt -l cmd internal
```
Expected: keine Ausgabe.
```bash
go vet ./...
```
Expected: keine Ausgabe.

- [ ] **Step 9: Commit**

Nachricht nach `C:/Users/micro/Documents/#GIT/loomux/.superpowers/sdd/2026-09-15-loomux-schranke-worktrees/commit-task-1.txt`:

```text
Open linked worktrees of a workspace in the write barrier

A linked git worktree of a repository registered with workspace = true
is writable without a registry entry of its own. The barrier reads the
worktree's .git file, commondir and the back pointer in gitdir directly
and spawns no git; it asks only for targets no registered tree holds.
```

```bash
git branch --show-current
```
Expected: `barrier-worktrees`.
```bash
git add internal/brain/guard/worktree.go internal/brain/guard/worktree_test.go internal/brain/guard/guard.go
```
```bash
git commit -F "/c/Users/micro/Documents/#GIT/loomux/.superpowers/sdd/2026-09-15-loomux-schranke-worktrees/commit-task-1.txt"
```
Expected: das pre-commit-Tor läuft grün durch, ein Commit entsteht.
```bash
git log -1 --format='%an <%ae> | %cn <%ce>%n%B'
```
Expected: Autor und Committer sind der Nutzer; keine `Co-Authored-By`-Zeile.

Paritätszeilen: eine, geschrieben in Task 2.

---

### Task 2: Messung, Doku und Paritätszeile

**Files:**
- Create: `testdata/bench/barrier-worktrees.json`, `testdata/bench/write-in-worktree.json`, `testdata/bench/edit-readme-main.json`
- Create: `docs/.superpowers/parity/schranke-worktrees.md`
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md` (Eintrag ans Ende)
- Modify: `docs/en/getting-started.md`, `docs/de/getting-started.md`
- Modify: `README.md`, `README.de.md`

**Interfaces:**
- Consumes: den Commit aus Task 1; `C:/Users/micro/AppData/Local/Temp/loomux-barrier/before.exe` aus Task 0 Step 4.
- Produces: keine Code-Schnittstelle.

- [ ] **Step 1: Das geänderte Binary bauen**

```bash
go build -o "/c/Users/micro/AppData/Local/Temp/loomux-barrier/after.exe" ./cmd/loomux
```
Expected: keine Ausgabe.

- [ ] **Step 2: Mess-Registry anlegen**

Eine Kopie für die Messung, nicht die echte Registry. Sie kennt nur den Hauptcheckout als Workspace — der Worktree ist absichtlich nicht eingetragen.
```bash
printf '[[area]]\nscope = "project/loomux"\npath = "C:/Users/micro/Documents/#GIT/loomux"\nworkspace = true\n' > "/c/Users/micro/AppData/Local/Temp/loomux-barrier/state/registry.toml"
```
Expected: keine Ausgabe.

- [ ] **Step 3: Messfälle schreiben**

`testdata/bench/write-in-worktree.json` (eine Zeile):
```json
{"hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/internal/probe.go","content":"x"},"cwd":"C:/Users/micro/Documents/#GIT/loomux-sdd-1b1"}
```

`testdata/bench/edit-readme-main.json` (eine Zeile):
```json
{"hook_event_name":"PreToolUse","tool_name":"Edit","tool_input":{"file_path":"C:/Users/micro/Documents/#GIT/loomux/README.md","old_string":"a","new_string":"b"},"cwd":"C:/Users/micro/Documents/#GIT/loomux"}
```

`testdata/bench/barrier-worktrees.json`:
```json
[
  {
    "name": "before: Write in linked worktree, no registry entry",
    "dir": "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1",
    "stdin": "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/testdata/bench/write-in-worktree.json",
    "mode": "single",
    "steps": [
      {
        "argv": [
          "C:/Users/micro/AppData/Local/Temp/loomux-barrier/before.exe",
          "hook", "pre-tool-use", "--host", "claude",
          "--root", "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1"
        ]
      }
    ]
  },
  {
    "name": "after: Write in linked worktree, no registry entry",
    "dir": "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1",
    "stdin": "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/testdata/bench/write-in-worktree.json",
    "mode": "single",
    "steps": [
      {
        "argv": [
          "C:/Users/micro/AppData/Local/Temp/loomux-barrier/after.exe",
          "hook", "pre-tool-use", "--host", "claude",
          "--root", "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1"
        ]
      }
    ]
  },
  {
    "name": "before: Edit on README.md in main checkout",
    "dir": "C:/Users/micro/Documents/#GIT/loomux",
    "stdin": "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/testdata/bench/edit-readme-main.json",
    "mode": "single",
    "steps": [
      {
        "argv": [
          "C:/Users/micro/AppData/Local/Temp/loomux-barrier/before.exe",
          "hook", "pre-tool-use", "--host", "claude",
          "--root", "C:/Users/micro/Documents/#GIT/loomux"
        ]
      }
    ]
  },
  {
    "name": "after: Edit on README.md in main checkout",
    "dir": "C:/Users/micro/Documents/#GIT/loomux",
    "stdin": "C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/testdata/bench/edit-readme-main.json",
    "mode": "single",
    "steps": [
      {
        "argv": [
          "C:/Users/micro/AppData/Local/Temp/loomux-barrier/after.exe",
          "hook", "pre-tool-use", "--host", "claude",
          "--root", "C:/Users/micro/Documents/#GIT/loomux"
        ]
      }
    ]
  }
]
```

- [ ] **Step 4: Messen**

```bash
date '+%Y-%m-%d %H:%M'
```
Den Zeitpunkt für die Überschrift notieren.
```bash
LOOMUX_STATE_DIR="C:/Users/micro/AppData/Local/Temp/loomux-barrier/state" go run ./cmd/loomux dev bench-hooks testdata/bench/barrier-worktrees.json -n 20
```
Expected: eine Tabelle mit vier Zeilen. Exit-Codes: `before: Write in linked worktree` `[2]`, `after: Write in linked worktree` `[0]`, beide Hauptcheckout-Zeilen `[0]`. Weicht ein Exit-Code ab, anhalten und im Bericht melden — dann misst der Fall nicht, was er soll. Die ganze Ausgabe in den Bericht.

- [ ] **Step 5: Benchmark-Eintrag schreiben**

An `docs/en/benchmarks.md` anhängen, Zahlen und Zeitpunkt aus Step 4 übernehmen, Zeilen der Tabelle genau aus der Ausgabe:

```markdown
## <YYYY-MM-DD HH:MM from Step 4> — The Write Barrier in a Linked Worktree

Repository `loomux`, worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1`, branch
`barrier-worktrees`, commit `<git rev-parse --short HEAD>`.

**Goal.** Show what opening linked worktrees costs: a write in a worktree with no
registry entry of its own, before the change (refused) and after it (allowed), and
a write in the main checkout before and after, which the change must leave untouched.

**Method.** `loomux dev bench-hooks testdata/bench/barrier-worktrees.json -n 20`:
one cold run per case, then 20 warm ones. `LOOMUX_STATE_DIR` points at a copy of the
registry that registers only the main checkout (`workspace = true`). Binaries:
`before.exe` built from `<Basis-Commit aus Task 0>`, `after.exe` built from the commit
above, both with Go `<go version>`.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| <vier Zeilen aus der Ausgabe von Step 4> |

### Reading

1. <Worktree: after gegen before, in ms; die Ablehnung vorher läuft den ganzen Weg bis zur Meldung, das Erlauben nachher liest drei kleine Dateien je Elternverzeichnis bis zum Treffer.>
2. <Hauptcheckout: after gegen before, in ms, mit überlappenden oder getrennten Warm-Spannen; die Änderung darf hier nichts kosten, weil `Decide` vor der Worktree-Suche zurückkehrt.>
3. <Beide Werte gegen den Zielwert von 72 ms.>
```

Die spitzen Klammern sind Anweisungen an den Schreibenden, keine Textteile: jede wird durch den gemessenen Wert oder den ausformulierten Satz ersetzt, und im fertigen Eintrag steht keine spitze Klammer mehr. Die Lesart nennt nur, was die Zahlen zeigen.

Denselben Eintrag deutsch an `docs/de/benchmarks.md` anhängen: Überschrift `## <Zeitpunkt> — Die Schreibschranke im verknüpften Worktree`, Abschnitte **Ziel.**, **Methode.**, Tabelle mit denselben Zahlen (Spaltenköpfe wie in den bestehenden deutschen Einträgen der Datei), `### Lesart`.

- [ ] **Step 6: getting-started (en, de)**

In `docs/en/getting-started.md` direkt nach dem Codeblock, der mit
```text
Write barrier:    Enforced (global registry + project policy)
Overall status:   READY (Green)
```
und der schließenden Zeile ```` ``` ```` endet, eine Leerzeile und dann einfügen:

```markdown
> [!NOTE]
> The write barrier opens only the trees listed in the global registry
> (`%LOCALAPPDATA%\loomux\registry.toml`). A linked git worktree of a repository
> registered with `workspace = true` counts as part of that repository and needs no
> entry of its own; the barrier reads git's worktree files and starts no `git`
> process. A worktree moved without `git worktree repair` stays shut.
```

In `docs/de/getting-started.md` an der entsprechenden Stelle (nach dem Codeblock mit `Schreibschranke:  Aktiv (Globale Registry + Projekt-Policy)` und `Gesamtstatus:     BEREIT (Grün)`):

```markdown
> [!NOTE]
> Die Schreibschranke öffnet nur die Bäume, die in der globalen Registry stehen
> (`%LOCALAPPDATA%\loomux\registry.toml`). Ein verknüpfter Git-Worktree eines Repos,
> das mit `workspace = true` registriert ist, zählt zu diesem Repo und braucht keinen
> eigenen Eintrag; die Schranke liest dafür Gits Worktree-Dateien und startet keinen
> `git`-Prozess. Ein ohne `git worktree repair` verschobener Worktree bleibt gesperrt.
```

- [ ] **Step 7: READMEs**

`README.md`, Zeile des Pre-Tool-Wächters. Alt:
```text
| Unified Pre-Tool Guard | Single-pass validation of write barriers, path protections, and forbidden commands (<35ms budget; 32–34ms measured on predecessor). | ✅ **Implemented** (Stage 1a) |
```
Neu:
```text
| Unified Pre-Tool Guard | Single-pass validation of write barriers, path protections, and forbidden commands (<35ms budget; 32–34ms measured on predecessor). Linked git worktrees of a registered workspace are writable without a registry entry of their own. | ✅ **Implemented** (Stage 1a) |
```

`README.de.md`. Alt:
```text
| Einheitlicher Pre-Tool Wächter | Prüfung von Schreibschranken, Pfadregeln und verbotenen Befehlen (< 35 ms Zielbudget; 32–34 ms gemessen am Vorgänger). | ✅ **Implementiert** (Stufe 1a) |
```
Neu:
```text
| Einheitlicher Pre-Tool Wächter | Prüfung von Schreibschranken, Pfadregeln und verbotenen Befehlen (< 35 ms Zielbudget; 32–34 ms gemessen am Vorgänger). Verknüpfte Git-Worktrees eines registrierten Workspace sind ohne eigenen Registry-Eintrag beschreibbar. | ✅ **Implementiert** (Stufe 1a) |
```

- [ ] **Step 8: Paritätsliste**

`docs/.superpowers/parity/schranke-worktrees.md`:

```markdown
# Paritätsliste Schreibschranke: verknüpfte Worktrees

**Quelle:** ultra-brain `loomux-1a-source` (`3cc72d2`), `_writable_roots`.
**Spec:** [2026-09-15-loomux-schranke-worktrees-design.md](../specs/2026-09-15-loomux-schranke-worktrees-design.md)
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers, bevor die Stufe als fertig gilt.

| Fall / Bereich | Alt | Neu | Begründung | Freigabe |
|---|---|---|---|---|
| Write in einem verknüpften Worktree eines `workspace`-Bereichs ohne eigenen Registry-Eintrag | brain guard: verweigert (`lies outside every writable tree`) | erlaubt; Erkennung über `.git`-Datei, `commondir` und Rückverweis `gitdir`, ohne `git`-Prozess | Ein Worktree ist dasselbe Repository; `git rev-parse` kostet 42 ms gegen 24,5 ms Hook (Spec) | offen |
```

- [ ] **Step 9: Tor und Commit**

Nachricht nach `C:/Users/micro/Documents/#GIT/loomux/.superpowers/sdd/2026-09-15-loomux-schranke-worktrees/commit-task-2.txt`:

```text
Measure and document linked worktrees in the write barrier

Benchmark a write in a linked worktree without a registry entry before
and after the change, and a write in the main checkout, which the change
leaves untouched. Note the behaviour in getting-started and the READMEs,
and record the deviation from the Python barrier in a parity list.
```

```bash
git branch --show-current
```
Expected: `barrier-worktrees`.
```bash
git add testdata/bench/barrier-worktrees.json testdata/bench/write-in-worktree.json testdata/bench/edit-readme-main.json docs/.superpowers/parity/schranke-worktrees.md docs/en/benchmarks.md docs/de/benchmarks.md docs/en/getting-started.md docs/de/getting-started.md README.md README.de.md
```
```bash
git commit -F "/c/Users/micro/Documents/#GIT/loomux/.superpowers/sdd/2026-09-15-loomux-schranke-worktrees/commit-task-2.txt"
```
Expected: Tor grün, ein Commit.
```bash
git log -1 --format='%an <%ae> | %cn <%ce>%n%B'
```
Expected: Autor und Committer sind der Nutzer; keine `Co-Authored-By`-Zeile.

Paritätszeilen: eine, `offen` bis zur Freigabe in Task 3.

---

### Task 3: Merge und Abnahme (Controller und Mensch)

**Files:** keine neuen. Außerhalb: `master`, `bin/loomux.exe` im Hauptcheckout, die Registry (nur Mensch).

**Interfaces:** keine.

- [ ] **Step 1: Freigabe der Paritätszeile (Mensch)**

Der Controller legt die Zeile aus `docs/.superpowers/parity/schranke-worktrees.md` vor. Nach der Freigabe setzt er `offen` auf `freigegeben <Datum>` und committet das auf `barrier-worktrees` (Nachricht aus Datei: `Approve the worktree deviation of the write barrier`).

- [ ] **Step 2: Merge nach `master` (nur nach ausdrücklichem Ja)**

```bash
git -C "/c/Users/micro/Documents/#GIT/loomux" status --short --branch
```
Expected: `## master…`, sonst keine Zeile.
```bash
git -C "/c/Users/micro/Documents/#GIT/loomux" merge --ff-only barrier-worktrees
```
Expected: `Fast-forward`.

- [ ] **Step 3: Binary des Hauptcheckouts neu bauen**

Ein Fast-Forward-Merge löst kein pre-commit-Tor aus; das Binary, das die Hooks rufen, ist sonst der alte Stand.
```bash
go -C "/c/Users/micro/Documents/#GIT/loomux" build -o bin/loomux.exe ./cmd/loomux
```
Expected: keine Ausgabe.

- [ ] **Step 4: Registry-Eintrag entfernen (nur Mensch)**

Der Mensch entfernt den Block `project/loomux-sdd-1b1` aus `C:\Users\micro\AppData\Local\loomux\registry.toml`. Der Controller schlägt dazu den Ersatz der Vorlagen-Kommentarzeilen vor:

```toml
# Ein verknüpfter Git-Worktree eines Repos mit workspace = true braucht keinen
# eigenen Eintrag: die Schranke erkennt ihn an seiner .git-Datei. Nur ein
# Worktree eines Repos, das hier nicht als workspace steht, bleibt gesperrt.
```

- [ ] **Step 5: Abnahme**

Dieselbe Probe wie in Task 0 Step 3, jetzt **ohne** Registry-Eintrag:
```bash
printf '%s' '{"session_id":"probe","hook_event_name":"PreToolUse","cwd":"C:/Users/micro/Documents/#GIT/loomux-sdd-1b1","tool_name":"Write","tool_input":{"file_path":"C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/internal/probe.go","content":"x"}}' | "/c/Users/micro/Documents/#GIT/loomux/bin/loomux.exe" hook pre-tool-use --host claude --root "C:/Users/micro/Documents/#GIT/loomux"; echo "exit=$?"
```
Expected: nur `exit=0`.

- [ ] **Step 6: Übergang zu 1b-1**

```bash
git -C "/c/Users/micro/Documents/#GIT/loomux-sdd-1b1" switch -c sdd-1b-1 master
```
Expected: `Switched to a new branch 'sdd-1b-1'`. Im 1b-1-Ledger wird als erste Ruling festgehalten: Task 0 Step 4 des 1b-1-Plans entfällt als Registry-Eintrag und bleibt als Probe ohne Eintrag; die Erwartung von Step 2 und Zeile 175 sind veraltet.

Paritätszeilen: keine neuen.
