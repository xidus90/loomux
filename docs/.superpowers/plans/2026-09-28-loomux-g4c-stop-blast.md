# G4c Stop-Hook mit Blast-Logik — Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die Art `graph` läuft am Rundenende im Profil `stop` über eine Indexkopie im Git-Verzeichnis, die unversionierte Dateien trägt.

**Architecture:** `gitwork` bekommt eine Form von `ContentTree`, die ihre Indexkopie im eigenen Git-Verzeichnis behält. `query.GraphReady` wird in einen geteilten Vorbau (Graph, HEAD, laufende Operation) und die Index-Frage zerlegt. `internal/hooks` baut daraus einen `StopIndex` (Kopie, Content-Tree, Baum von HEAD, `Ready`, `Env`, `Close`), den `RunStop` und `loomux check stop` gleich benutzen.

**Tech Stack:** Go, git über `gitwork`/`query.runGit`.

**Spec:** `docs/.superpowers/specs/2026-09-25-loomux-code-g4c-delta.md`

## Global Constraints

- Coverage 100 % je Funktion; Ausnahme nur mit `//coverage:exempt <Grund>` direkt über `func`.
- Name der Kopie: `loomux-stop-index-<pid>` in `git rev-parse --absolute-git-dir`.
- Profilvorgabe `stop` = `lint, types, test, coverage, graph`.
- Notizen wörtlich: „no graph at .loomux/state/graph/wiring.json“, „no HEAD to compare with“ (+ `: <err>` wie heute), die `inProgress`-Notizen, „nothing changed against HEAD“.
- Die Form ohne Behalten (`ContentTree(root, os.TempDir())`) bleibt der Weg, wenn `graph` nicht angefragt ist oder kein `wiring.json` existiert.
- Commits: Conventional Commits, kein Arbeitspapier im Text, keine Modell-Mitautorschaft.
- Release: `release:minor`.

## Review Focus

1. Kopie bleibt liegen bei frühem Ausstieg (Planfehler, `PrepareCover`-Fehler, grüner Baum) — Test in Task 3 prüft nach jedem Ausstieg, dass kein `loomux-stop-index-*` im Git-Verzeichnis liegt.
2. Verknüpfter Worktree: Kopie muss unter `…/.git/worktrees/<name>` liegen, sonst verwirft `indexFileFor` sie still und `blast-audit --cached` liest den echten (leeren) Index → grün ohne Prüfung. Test in Task 1 und Task 3.
3. Kaputte Konfiguration bei schon grünem Baum: nach der Umstellung Exit 1 statt 0 — so gewollt (Spec §3); Test in Task 3 hält es fest.
4. Projekt ohne Go-Stack: `graph` ohne Befehl ⇒ `not-applicable`, Rundenende bleibt Exit 0 und Basis rückt vor — Test in Task 3.
5. Eigenes `stop`-Profil ohne `graph`: keine Kopie im Git-Verzeichnis — Test in Task 3.

---

### Task 1: `gitwork.KeptContentTree`

**Files:**
- Modify: `internal/gitwork/gitwork.go:150-207`
- Test: `internal/gitwork/gitwork_test.go`

**Interfaces:**
- Produces: `func KeptContentTree(root string) (tree, index string, err error)` — Kopie unter `<absolute-git-dir>/loomux-stop-index-<pid>`, nicht gelöscht; bei Fehler ist `index` leer und nichts bleibt liegen.

- [ ] **Step 1: Tests schreiben**
  - `TestKeptContentTreeLiesInTheGitDir`: Repo mit Commit, dazu unversionierte `new.go` und `.loomux/state/x`. `tree, index, err := KeptContentTree(root)`; `filepath.Dir(index)` ist per `os.SameFile` gleich `git rev-parse --absolute-git-dir`; `git ls-files` mit `GIT_INDEX_FILE=index` enthält `new.go`, nicht `.loomux/state/x`; `tree` == `ContentTree(root, t.TempDir())`.
  - `TestKeptContentTreeInALinkedWorktree`: `git worktree add ../wt`; Kopie liegt unter `<main>/.git/worktrees/wt/`.
  - `TestKeptContentTreeLeavesNothingOnFailure`: Verzeichnis ohne Repository ⇒ Fehler, `index == ""`.
- [ ] **Step 2:** `go test ./internal/gitwork -run KeptContentTree` ⇒ FAIL (undefined).
- [ ] **Step 3: Umsetzen** — den Rumpf ab dem Kopieren in `writeContentTree(root, real, index string) (string, error)` ziehen; `ContentTree` bleibt mit `defer os.Remove(index)`. Neu:

```go
// KeptContentTree is ContentTree with its copy of the index kept inside the
// git directory, the only place a graph lane accepts an inherited index
// from. The caller removes index.
func KeptContentTree(root string) (tree, index string, err error) {
	dir, err := git(root, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", "", err
	}
	// Same spelling as ContentTree: --path-format=absolute would need git 2.31.
	out, err := git(root, "rev-parse", "--git-path", "index")
	if err != nil {
		return "", "", err
	}
	real := strings.TrimSpace(out)
	if !filepath.IsAbs(real) {
		real = filepath.Join(root, real)
	}
	index = filepath.Join(strings.TrimSpace(dir), fmt.Sprintf("loomux-stop-index-%d", os.Getpid()))
	tree, err = writeContentTree(root, real, index)
	if err != nil {
		os.Remove(index)
		return "", "", err
	}
	return tree, index, nil
}
```
- [ ] **Step 4:** Tests grün, `go test ./internal/gitwork` grün, Coverage 100 %.
- [ ] **Step 5:** Commit `feat(gitwork): keep the content tree's index copy in the git directory`.

### Task 2: `query.GraphPrereq`

**Files:**
- Modify: `internal/code/query/refresh.go:52-89`
- Test: `internal/code/query/refresh_test.go`

**Interfaces:**
- Produces: `func GraphPrereq(root string) (bool, string)` — Schritte 1–3 (Graph, HEAD, laufende Operation), Notizen unverändert. `GraphReady` = `GraphPrereq` + `diff --cached --quiet`.

- [ ] **Step 1:** Test `TestGraphPrereqIgnoresTheIndex`: gebauter Graph, nichts gestaged ⇒ `GraphPrereq` `true, ""`, `GraphReady` weiter `false, "nothing staged"`. Bestehende Tests für Marker/HEAD bleiben an `GraphReady`.
- [ ] **Step 2:** FAIL (undefined).
- [ ] **Step 3:** Rumpf bis zur Marker-Schleife nach `GraphPrereq` ziehen; `GraphReady` ruft ihn und fragt danach den Index.
- [ ] **Step 4:** `go test ./internal/code/query` grün.
- [ ] **Step 5:** Commit `refactor(query): split the graph probe from its question about the index`.

### Task 3: Der Stop-Hook bekommt die Lane

**Files:**
- Create: `internal/hooks/stopgraph.go`, `internal/hooks/stopgraph_test.go`
- Modify: `internal/hooks/stop.go:155-214`, `internal/verify/schema.go:80-84`, Tests, die die Profilvorgabe aufzählen (`grep -rn '"coverage"}' internal/`)
- Test: `internal/hooks/stop_test.go`

**Interfaces:**
- Consumes: `gitwork.KeptContentTree`, `gitwork.TreeOf`, `gitwork.Head`, `query.GraphPrereq`, `store.WiringPath`.
- Produces (`internal/hooks`):

```go
// StopIndex is the turn end's view for the graph lane: everything git does
// not ignore, against HEAD, through a copy of the index the caller closes.
type StopIndex struct {
	Index, Tree, HeadTree string
}

// OpenStopIndex builds the copy; nil without error where there is no
// repository or no HEAD, which Ready then reports.
func OpenStopIndex(root string) (*StopIndex, error)
func (s *StopIndex) Ready(root string) (bool, string) // GraphPrereq, then Tree == HeadTree ⇒ false, "nothing changed against HEAD"
func (s *StopIndex) Env(root string) []string       // []string{"GIT_INDEX_FILE=" + s.Index}; nil for a nil receiver
func (s *StopIndex) Close()                         // removes Index; safe on nil
// WantsStopIndex: "graph" in kinds and wiring.json exists.
func WantsStopIndex(root string, kinds []string) bool
```

- [ ] **Step 1: Tests `stopgraph_test.go`** (Welten mit `cases.BuildGitWorld` wie `gitWorld`, Graph per `query.Build`):
  - kein `wiring.json` ⇒ Ready `false`, Notiz „no graph …“;
  - leeres Repo ohne Commit ⇒ `OpenStopIndex` nil, `(*StopIndex)(nil).Ready` ⇒ „no HEAD to compare with…“;
  - je Marker (MERGE_HEAD, rebase-merge, rebase-apply, CHERRY_PICK_HEAD, REVERT_HEAD) die Notiz aus `inProgress`;
  - sauberer Baum ⇒ „nothing changed against HEAD“;
  - Änderung ⇒ `true, ""`; `Env` trägt die Kopie; `Close` entfernt sie; `WantsStopIndex` false ohne `graph` oder ohne Graph.
- [ ] **Step 2: Tests `stop_test.go`** — `RunStop` mit Fake-Start, der für `{loomux} check blast-audit` die `Env` des Specs liest:
  - Graph gebaut, Hub geändert, neuer unversionierter `_test.go`: Fake prüft per `git ls-files` mit der gereichten `GIT_INDEX_FILE`, dass der Test drin ist, antwortet 0 ⇒ Exit 0, Basis rückt vor;
  - Fake antwortet 1 mit Befund ⇒ Exit 2, Befund auf stderr, `Blocks == 1`;
  - kein Graph ⇒ `graph/go` `not-applicable`, Exit 0, Basis rückt vor;
  - Projekt ohne Go (nur `project`-Stack) ⇒ Exit 0 (Review Focus 4);
  - nach Exit 0, Exit 2, grünem Baum, Planfehler (`stopPlan` ersetzt) und `PrepareCover`-Fehler liegt kein `loomux-stop-index-*` im Git-Verzeichnis (Review Focus 1);
  - `[verify.profiles] stop = ["lint"]` ⇒ keine Kopie entsteht (Glob während des Fakes leer) (Review Focus 5);
  - verknüpfter Worktree: gereichte Kopie liegt unter `.git/worktrees/<name>` (Review Focus 2);
  - kaputte `config.toml` bei grünem Baum ⇒ Exit 1 (Review Focus 3).
- [ ] **Step 3:** FAIL.
- [ ] **Step 4: Umsetzen**
  - `schema.go`: `"stop": {"lint", "types", "test", "coverage", "graph"}`, Kommentar: „The kinds of precommit: graph judges the working tree against HEAD through a copy of the index (hooks.StopIndex). A project whose suite is too slow for every turn end narrows it here and keeps a gate that moves.“
  - `stop.go`: `facts`, `editLoad`, `kinds` vor `stopTrees`; `stopTrees` bekommt `keep bool`. Bei `keep` ruft es **`OpenStopIndex(root)`** — genau der Konstruktor, den `check stop` nutzt, kein zweiter Weg — und nimmt `idx.Tree` als Content-Tree; es liefert zusätzlich den `*StopIndex`. Ohne `keep` weiter `stopTree(root, os.TempDir())`. `OpenStopIndex` baut über den Seam `stopKeptTree` und setzt `HeadTree` via `headTree(root, head)`.
  - Festgestellt (Probe 2026-09-28): `verify.Run` reicht `job.Env` als `child.Spec.Env` weiter (`run.go:198`); `graph-fresh`/`query.Build` rufen kein git, nur `blast` und `GraphReady` lesen den Index. Der Fake darf also `s.Env` lesen, und die Kopie wirkt nur auf `blast-audit`. Direkt danach `defer idx.Close()`. `PlanEnv` bekommt `GraphReady: idx.Ready, GraphEnv: idx.Env`. Der Kommentar über `RunStop` nennt die neue Reihenfolge: „payload, findings, counter, marker, config, tree, chain“.
  - Seam `stopKeptTree = gitwork.KeptContentTree` neben `stopTree` für den Git-Fehlerpfad.
- [ ] **Step 5:** `go test ./internal/hooks ./internal/verify` grün; Profil-Aufzählungen nachgezogen; Coverage 100 %.
- [ ] **Step 6:** Commit `feat(hooks): run the graph lane at the stop gate against HEAD`.

### Task 4: `loomux check stop` stellt den Hook nach

**Files:**
- Modify: `internal/cli/check.go:185-215`
- Test: `internal/cli/check_test.go`

**Interfaces:**
- Consumes: `hooks.WantsStopIndex`, `hooks.OpenStopIndex`, `(*StopIndex).Ready/Env/Close`.

- [ ] **Step 1: Tests**
  - `TestCheckStopJudgesTheWorkingTreeLikeTheHook`: `builtRepo` committed (Hilfe wie `stagedRun`, aber **ohne** `git add`), `moveFile`, unversionierter `lib/new_test.go` ruft `Run`. Stub führt `blast-audit` im Prozess aus: `t.Setenv` jeder `GIT_INDEX_FILE` aus `s.Env`, dann `blastAudit(s.Argv[3:]...)` mit `--root root`. `run("check","stop","--root",root)` ⇒ `graph/go: ok`.
  - Dieselbe Welt ohne den neuen Test ⇒ Exit 1, `lib/lib.go [stale]` in der Ausgabe.
  - `check precommit` in derselben Welt ⇒ `graph/go: not-applicable … nothing staged` (echter Index bleibt).
  - Nach jedem Lauf keine Kopie im Git-Verzeichnis.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3:** In `checkRun` nach `checkLoad`, vor `--show`:

```go
if request == "stop" && hooks.WantsStopIndex(root, kinds) {
	idx, err := hooks.OpenStopIndex(root)
	if err != nil {
		return fail(err)
	}
	defer idx.Close()
	env.GraphReady, env.GraphEnv = idx.Ready, idx.Env
}
```
- [ ] **Step 4:** `go test ./internal/cli` grün, Coverage 100 %; dazu `go test -race -run TestCheckStop -count=20 ./internal/cli` grün (der Stub setzt `GIT_INDEX_FILE` aus einer Lane-Goroutine).
- [ ] **Step 5:** Commit `feat(cli): judge the graph at check stop as the stop gate does`.

### Task 5: Messung, Selbstnutzung, Doku

**Files:**
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`, `docs/en/migration.md`, `docs/de/migration.md`, `docs/en/configuration.md`, `docs/de/configuration.md`, `README.md`, `README.de.md`, `docs/wiki/topics/code-graph.md`, `docs/wiki/topics/scheiben-und-abnahme.md`, `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`, `docs/.superpowers/parity/code-g4.md`

- [ ] **Step 1:** `go build -o bin/loomux.exe ./cmd/loomux`. Stop-Hook kalt und warm messen, Profil mit vier Arten (`[verify.profiles] stop` temporär im Scratchpad-Klon) gegen fünf, je fünf Läufe nach einem Edit an `internal/gitenv/gitenv.go`; Datum, Uhrzeit, Binary-Commit notieren; in beide `benchmarks.md`.
- [ ] **Step 2: Selbstnutzung:** in einem Scratch-Worktree einen Hub (`gitenv.Environ`) ohne Test ändern, `bin/loomux.exe hook stop` mit Payload `{"session_id":"g4c","hook_event_name":"Stop"}` < Datei ⇒ Exit 2 mit Befund; mit neuem unversionierten Test ⇒ Exit 0. Ausgabe in `parity/code-g4.md`.
- [ ] **Step 3:** Doku nach Spec §7: G4c ✅ in Fusion-Spec (Stufentabelle, Reihenfolge), Migrationsplan (Stufe, Fähigkeit „Stop-Hook Blast Audit“, Mermaid), `configuration.md` (Absatz zur Lane `graph`: im Profil `stop` über Indexkopie gegen HEAD), READMEs, Wiki. Vorher `grep -rn "not-applicable" docs README*` und `grep -rn "without graph\|ohne graph" docs internal` nach Sätzen, die das alte Verhalten beschreiben (alle Sprachen).
- [ ] **Step 4:** `sh ci/gate.sh` grün.
- [ ] **Step 5:** Commit `docs: describe the graph lane at the stop gate`.
