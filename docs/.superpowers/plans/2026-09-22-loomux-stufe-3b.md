# loomux Stufe 3b — Implementierungsplan: Prüfzentrum entscheiden

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ein Mensch entscheidet im Prüfzentrum: `loomux cases` listet die
wartenden Fälle, `loomux case <id>` zeigt Paket und Vorschlag, `loomux
approve` wendet einen belegten Vorschlag an, lehnt ab oder stellt zurück und
committet das Ergebnis auf den aktuellen Ref des Tresors.

**Architecture:** Hybrid (Spec, „Bauweise“). Vier Schichten, von unten nach
oben. `internal/brain/vcs` bekommt die Schreibseite (`CommitPaths`),
neu aus `vcs.py`. `internal/brain/evidence` zieht aus
`ultra-brain/pkg/maintenance/evidence.go` um. `internal/brain/apply` bekommt
`patch` und `frontmatter` als Umzug und die Pipeline von `approve` neu aus
`apply.py`. Die CLI hängt `cases`, `case` und `approve` daran; `lookup`,
`privacy` und `format` ziehen dorthin nach, wo sie gebraucht werden.
Davor rüstet Task 1 und 2 den Fall-Harness für Commits auf.

**Tech Stack:** Go ≥ 1.25, `github.com/BurntSushi/toml`, `gopkg.in/yaml.v3`
(schon in `go.mod`), `git` als Kindprozess, `loomux check precommit` als Tor.
Keine neuen Abhängigkeiten.

**Spec:** `docs/.superpowers/specs/2026-09-19-loomux-stufe-3-design.md`,
Abschnitt „3b im Einzelnen“ mit „Befunde 3b“, „Bauweise“ und „Parität 3b“.

## Global Constraints

- **Sprache — `AGENTS.md` gilt.** Englisch sind Code, Kommentare,
  Fehlermeldungen und Commit-Nachrichten. Deutsch sind nur die Dokumente unter
  `docs/.superpowers/` und die deutschen Handbuchseiten. **Ausnahme:**
  Nutzermeldungen, die aus der Referenz übernommen werden, sind dort deutsch
  (`Fall …`, `Bereich:`, `verworfene Behauptung`, `Hinweis: …`, die Vermerke
  und Auditzeilen aus `apply.py`) und bleiben es — sie sind Verhalten.
- **Commit-Nachrichten nennen kein Arbeitspapier** (`AGENTS.md`, Regel vom
  2026-09-22): kein `(3b)` als Scope, kein „Task 7“, kein „laut Plan“. Der
  Scope nennt ein Codegebiet: `vcs`, `evidence`, `apply`, `cli`, `cases`.
- **TDD, 100 % Coverage je Funktion**, jeder Ausschluss mit
  `//coverage:exempt <Grund>` direkt über `func`.
- **Statische Typen**, kein `any`/`interface{}` ohne Grund.
- **Referenz ist Python** — `ultra-brain` Tag `loomux-3-source` (`3cc72d2`).
  Eine umziehende Go-Datei ist Code, nicht Referenz: wo sie von Python
  abweicht, gilt Python, und die Abweichung wird geheilt, nicht übernommen.
- **Zustandsort:** geschrieben wird nach `LOOMUX_STATE_DIR`, gelesen neu
  zuerst, alt als Rückfall (`config.NewArtifactLookup`).
- **Startzeit-Regel:** kein `init()`, keine Paketvariable parst eingebettete
  Daten.
- **`hooks` importiert nichts aus `brain/apply`, `brain/evidence`,
  `brain/maintenance`, `brain/vcs`.** Der Tor-Test über den Importgraphen
  (`internal/hooks`) muss grün bleiben.
- **Schreiben nur atomar:** jede Datei über `lock.ReplaceText`, Vergleich
  vorher gegen `os.ReadFile` (nicht `pytext.ReadText`, das CRLF faltet).
- **Commits:** mehrzeilige Nachrichten über eine Datei im Scratchpad und
  `git commit -F <pfad>`, nie über ein Heredoc, nie `.git/COMMIT_BODY` (in
  einem Worktree ist `.git` eine Datei). **Kein `Co-Authored-By:`** auf ein
  Modell, keine Werbezeile.
- **Ein Shell-Befehl je Aufruf.**
- **Vor jedem Commit Zweig und HEAD lesen** (`git status -sb`).
- **Subagenten:** `model: "opus"`, `effort: "low"` — beides explizit setzen.
- **Vorgeklärt am 2026-09-22, damit kein Subagent es zweimal erhebt:**
  - `ultra-brain/pkg/maintenance/case.go` zieht **nicht** um;
    `internal/brain/maintenance/case.go` ist die Fallakte (`ReadCase`,
    `WriteCase`, `CaseID`, `CaseDir`, `Case`, `SourceState`).
  - `ReviewRoot` gibt es schon: `internal/brain/maintenance/reconcile.go:192`,
    `ReviewRoot(areas []config.Area, lookup config.ArtifactLookup) (string, error)`.
    `lookup.go` bringt nur `FindCase` mit.
  - Die Zeilenform von `cases` gibt es schon:
    `internal/cli/maintenance.go` — `listCases` (`:88`), `caseAddresses`
    (`:146`), `caseLine` (`:172`).
  - `gitenv.Environ()` entfernt `GIT_AUTHOR_*`/`GIT_COMMITTER_*`. Die
    Identität eines Commits kommt aus der Git-Konfiguration des Repos.
  - `index.Identity` aus ultra-brain ist in loomux
    `internal/brain/identity.Identity` (feldgleich).

## File Structure

| Datei | Verantwortung | Task |
|---|---|---|
| `internal/cases/gitworld.go` | Identität als Repo-Konfiguration; `WriteGitAfter` | 1 |
| `internal/cases/runner.go` | `git.after` nach dem Lauf, wenn der Fall es erwartet | 1 |
| `internal/dev/recordcase/recordcase.go` | `git.after` beim Aufzeichnen | 1 |
| `internal/cases/state.go` | neue Normalisierungen (`{{USER}}`, `{{NOW}}`, `{{TODAY}}`, `{{SHA}}`) | 2 |
| `internal/dev/importcases/importcases.go` | `[[stdout]]`-Regel | 2 |
| `internal/brain/vcs/commit.go` | `CommitPaths`, `Commit`, `ErrRefMoved`, Weigerungen | 3 |
| `internal/brain/evidence/*.go` | Umzug `evidence.go`: Vorschlag, Paket, Belegprüfung | 4 |
| `internal/brain/apply/patch.go` | Umzug `patch.go`: Hunks lesen und exakt anwenden | 5 |
| `internal/brain/apply/frontmatter.go` | Umzug `frontmatter.go`, gegen PyYAML gehoben | 6 |
| `internal/brain/apply/protocol.go` | `safe`, Vermerke, `audit.md`, `log.md` | 7 |
| `internal/brain/apply/place.go` | Schreibschranke `gate`/`preflight`, `touched` | 8 |
| `internal/brain/apply/resolve.go` | Tresor, Prüfzentrum, Wiki, Quellen über alle Bereiche | 9 |
| `internal/brain/apply/commit.go` | `commit`/`report` mit einer Wiederholung | 10 |
| `internal/brain/apply/reject.go` | Ablehnen | 10 |
| `internal/brain/apply/approve.go` | `Approve`, `Result`, Fehlerarten, Anwenden | 11 |
| `internal/brain/maintenance/lookup.go` | `FindCase` (Umzug) | 12 |
| `internal/cli/cases.go` | `loomux cases`, `loomux case` | 12 |
| `internal/cli/approve.go` | `loomux approve`, `--defer`, Bericht, Nachlauf | 13 |
| `testdata/cases/3b*/` | Aufnahmen, Übersetzung, Welten | 14 |
| `internal/cli/cases_3b_test.go` | Fallsuite 3b | 14 |
| `docs/.superpowers/parity/stufe-3b.md` | Akte | 0, laufend |

---

### Task 0: Vorbedingungen — kein Code

**Files:**
- Create: `docs/.superpowers/parity/stufe-3b.md`

- [ ] **Step 1: Arbeitsort und Stand lesen**

Run: `git status -sb`
Expected: ein Zweig, nicht `master`, sauber.

- [ ] **Step 2: Referenz-Tag prüfen**

Run: `git -C ../ultra-brain rev-parse --short loomux-3-source`
Expected: `3cc72d2`

Run: `git -C ../ultra-brain log --oneline loomux-3-source..master -- src/brain/maintenance src/brain/cli.py`
Expected: leer. Ist es nicht leer, anhalten und den Menschen fragen, ob der
Tag verschoben wird.

- [ ] **Step 3: Akte anlegen**

`docs/.superpowers/parity/stufe-3b.md`, deutsch, mit den Abschnitten
„Vorbedingungen, festgestellt am <Datum>“ (Ergebnis von Step 2),
„Abweichungen der Go-Form, gelesen“ (die Liste aus „Befunde 3b“ der Spec,
je Punkt mit Zeilenangabe in `approve.go` und `apply.py`, Entscheidung:
**Python gilt**), „Geerbt“ (der `--reject`-Fehler, `OFFENE_AUFGABEN.md:213`)
und einer leeren Tabelle „Abweichungsliste“ (Fall · Alt · Neu · Begründung).

- [ ] **Step 4: Commit**

```bash
git add docs/.superpowers/parity/stufe-3b.md
```

```bash
git commit -m "docs(parity): open the record for deciding review cases"
```

---

### Task 1: Git-Welten für Commits — Identität und `git.after`

**Files:**
- Modify: `internal/cases/gitworld.go`
- Modify: `internal/cases/runner.go`
- Modify: `internal/dev/recordcase/recordcase.go`
- Test: `internal/cases/gitworld_test.go`, `internal/cases/runner_test.go`,
  `internal/dev/recordcase/recordcase_test.go`

**Interfaces:**
- Produces:
  ```go
  // GitAfterName is the file a case compares a commit through.
  const GitAfterName = "git.after"
  // WriteGitAfter writes subject and tree of HEAD and the tracked paths of
  // the repository at repo into repo/git.after.
  func WriteGitAfter(repo string) error
  ```

**Die Regeln:**

1. **Die Identität wird Konfiguration des Repos.** `BuildGitWorld` setzt
   heute Autor und Committer nur als Umgebung seiner eigenen Aufrufe
   (`gitworld.go:195-197`). Zusätzlich schreibt es nach `git init`
   `git config user.name "loomux cases"` und
   `git config user.email cases@loomux.invalid` ins Repo. Sonst fände ein
   `commit-tree` aus `approve` beim Abspielen keine Identität, weil
   `gitenv.Environ()` die Variablen entfernt und `HOME` leer ist.
2. **`git.after` hat genau diese Form**, LF-getrennt, mit abschließendem LF:
   ```
   <Betreff des HEAD-Commits>
   <Tree-SHA des HEAD-Commits>
   <jeder Pfad aus git ls-tree -r --name-only HEAD, eine Zeile je Pfad>
   ```
   Der Commit-SHA steht **nicht** darin: er hängt an Zeit und Identität. Der
   Tree hängt nur am Inhalt und ist darum ohne Normalisierung vergleichbar.
3. **Opt-in.** Der Runner schreibt `git.after` nur, wenn der erwartete Baum
   (`world_after`, sonst `world`) im Repo-Verzeichnis eine Datei `git.after`
   trägt. So bleiben die Git-Fälle aus 2c und 3a unberührt.
4. **`git.after` ist keine `InfraPath`.** Es wird verglichen wie jede Datei.
5. **Der Rekorder** schreibt `git.after` mit derselben Funktion, wenn
   `Spec.GitAfter` gesetzt ist (neues Feld, Flag `--git-after` an
   `loomux dev record-case`).

- [ ] **Step 1: Write the failing test** — `gitworld_test.go`:

```go
func TestBuildGitWorldWritesTheIdentityIntoTheRepository(t *testing.T) {
	dir := t.TempDir()
	writeGitToml(t, dir, "dir = \"repo\"\n[[commit]]\nmessage = \"one\"\npaths = [\"a.md\"]\n[commit.files]\n\"a.md\" = \"a\\n\"\n")
	if err := cases.BuildGitWorld(dir); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"user.name": "loomux cases", "user.email": "cases@loomux.invalid"} {
		got := gitOutput(t, filepath.Join(dir, "repo"), "config", "--local", key)
		if got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestWriteGitAfterNamesSubjectTreeAndPaths(t *testing.T) {
	dir := t.TempDir()
	writeGitToml(t, dir, "dir = \"repo\"\n[[commit]]\nmessage = \"one\"\npaths = [\"b.md\", \"a.md\"]\n[commit.files]\n\"a.md\" = \"a\\n\"\n\"b.md\" = \"b\\n\"\n")
	if err := cases.BuildGitWorld(dir); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(dir, "repo")
	if err := cases.WriteGitAfter(repo); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(repo, cases.GitAfterName))
	if err != nil {
		t.Fatal(err)
	}
	tree := gitOutput(t, repo, "rev-parse", "HEAD^{tree}")
	want := "one\n" + tree + "\na.md\nb.md\n"
	if string(got) != want {
		t.Fatalf("git.after = %q, want %q", got, want)
	}
}
```

`writeGitToml` und `gitOutput` sind Helfer im Testpaket; `gitOutput` führt
git mit `gitenv.Environ()` aus und schneidet das abschließende LF ab. Gibt es
in `gitworld_test.go` schon einen Helfer dieser Art, wird er benutzt, nicht
ein zweiter gebaut.

Im `runner_test.go` zwei Tests: ein Fall, dessen `world_after/repo/git.after`
den Commit nennt, wird grün; ein Fall **ohne** `git.after` im erwarteten Baum
bekommt nach dem Lauf keine solche Datei (kein `unexpected extra file`).
Im `recordcase_test.go`: `Spec{GitAfter: true}` legt `git.after` in die
Aufnahme.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cases/ ./internal/dev/recordcase/ -run "Identity|GitAfter" -v`
Expected: FAIL — `WriteGitAfter` und `GitAfterName` sind nicht definiert.

- [ ] **Step 3: Implement**

`gitworld.go`: nach dem `init` des Repos die beiden `config`-Aufrufe über
dieselbe `worldGit`-Ausführung wie die übrigen. `WriteGitAfter`: drei
git-Aufrufe (`log -1 --format=%s`, `rev-parse HEAD^{tree}`,
`ls-tree -r --name-only HEAD`) mit `gitenv.Environ()`, Ergebnis über
`os.WriteFile`. `runner.go`: nach `run`, vor dem Vergleich, für jedes
Repo-Verzeichnis aus `git.toml` prüfen, ob der erwartete Baum dort
`git.after` trägt, und nur dann `WriteGitAfter` rufen. `recordcase.go`:
Feld `GitAfter bool`, Flag `--git-after`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cases/ ./internal/dev/recordcase/ ./internal/cli/ -cover`
Expected: PASS, 100 % in den geänderten Funktionen; die Fallsuiten 2c und 3a
bleiben grün.

- [ ] **Step 5: Commit**

```bash
git add internal/cases internal/dev/recordcase internal/cli/dev.go
```

```bash
git commit -m "test(cases): compare a commit a case makes through git.after"
```

---

### Task 2: Normalisierung und Übersetzung für Entscheidungen

**Files:**
- Modify: `internal/cases/state.go`
- Modify: `internal/dev/importcases/importcases.go`
- Test: `internal/cases/state_test.go`, `internal/dev/importcases/importcases_test.go`

**Interfaces:**
- Consumes: `cases.NormalizeState` (3a, `state.go:73`).
- Produces: dieselbe Signatur, erweitert; `importcases.Mapping.Stdout []Rule`
  (TOML `[[stdout]]`, `from`/`to`).

**Die Regeln (Spec, „Parität 3b“), alle auf beiden Seiten gleich:**

1. `human:<nicht leer, ohne Leerraum>` in `audit.md` und in der Frontmatter
   (`verified[].by`) wird `human:{{USER}}`. Nur diese Form; ein Prüfer, der
   nicht mit `human:` beginnt, bleibt stehen.
2. **Der Stempel dieses Laufs.** `approve` schreibt keinen `last-run.txt`
   vor sich; der Stempel ist der, der in der Auditüberschrift
   `## <iso> — <target> (Fall `<id>`)` steht. Er wird `{{NOW}}` — dort, in
   `generated.at` und in `verified[].at` der geänderten Seite. Getauscht wird
   nur die `isoformat()`-Form aus 3a (UTC, sechs oder keine
   Nachkommastellen, `+00:00`) und nur, wenn der Wert **nicht** in der
   Ausgangswelt steht (ältere Auditblöcke bleiben byte-gleich).
3. Der Tag dieses Stempels wird `{{TODAY}}` in der neuen Zeile von `log.md`
   (`- <YYYY-MM-DD> — …`), und nur dort.
4. Auf stdout wird `committet als <40 hex>` zu `committet als {{SHA}}`.
5. `[[stdout]]` im Import ersetzt im aufgezeichneten stdout jedes Vorkommen
   von `from` durch `to`. Für 3b: `brain case --package ` →
   `loomux case --package `. Jede Regel ist eine Abweichung, die die Akte
   nennt.

- [ ] **Step 1: Write the failing test** — je Regel ein Test in
  `state_test.go`, der beide Seiten mit verschiedenen Stempeln, Prüfern und
  SHAs normalisiert und Gleichheit verlangt, und je Regel ein Gegentest, der
  eine abweichende Form stehen lässt: ein Stempel mit `Z`, ein Prüfer
  `model:x`, eine SHA mit 39 Zeichen, ein Auditblock aus der Ausgangswelt.

```go
func TestNormalizeStateFoldsTheStampOfAnApproval(t *testing.T) {
	const older = "## 2026-01-01T00:00:00+00:00 — a.md (Fall `x`)\n"
	world := map[string][]byte{"vault/audit.md": []byte(older)}
	left := cases.NormalizeState(world, approvalTree(older, "2026-09-22T08:16:27.936837+00:00", "human:alice"))
	right := cases.NormalizeState(world, approvalTree(older, "2026-09-23T10:00:00+00:00", "human:bob"))
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("trees differ after normalizing:\n%s\n%s", left["vault/audit.md"], right["vault/audit.md"])
	}
	if !strings.HasPrefix(string(left["vault/audit.md"]), older) {
		t.Fatal("an older audit block was folded")
	}
}
```

`NormalizeState(world, tree map[string][]byte) map[string][]byte` ist die
Signatur aus 3a (`state.go:73`). `approvalTree(older, stamp, reviewer)`
liefert den Baum, den ein `approve` hinterlässt: `vault/audit.md` mit dem
alten und einem neuen Block unter `stamp`, `vault/log.md` mit einer neuen
Zeile vom Tag des Stempels, `vault/wiki/a.md` mit `generated.at: <stamp>` und
`verified: [{by: <reviewer>, at: <stamp>}]`.

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/cases/ ./internal/dev/importcases/ -v`
Expected: FAIL in den neuen Tests.

- [ ] **Step 3: Implement** — in `state.go` je Regel eine Funktion neben
  den drei aus 3a; in `importcases.go` das Feld und die Ersetzung im stdout
  beim Übersetzen eines Falls.

- [ ] **Step 4: Run**

Run: `go test ./internal/cases/ ./internal/dev/importcases/ ./internal/cli/ -cover`
Expected: PASS, 100 %; Fallsuite 3a grün.

- [ ] **Step 5: Spec nachführen** — die fünf Regeln stehen schon in
  „Parität 3b“; weicht die Umsetzung ab, wird die Spec im selben Commit
  korrigiert.

- [ ] **Step 6: Commit**

```bash
git add internal/cases internal/dev/importcases
```

```bash
git commit -m "test(cases): fold the reviewer, stamp and commit of an approval"
```

---

### Task 3: `internal/brain/vcs` — die Schreibseite

**Files:**
- Create: `internal/brain/vcs/commit.go`
- Test: `internal/brain/vcs/commit_test.go`
- Referenz: `src/brain/maintenance/vcs.py:69-345` (Tests:
  `tests/maintenance/test_vcs.py`, 41)

**Interfaces:**
- Consumes: `run`, `said`, `gitenv.Environ()` aus `vcs.go`.
- Produces:
  ```go
  type Commit struct {
      Head    string // the ref's commit after the call
      Created bool   // false: the tree was unchanged, nothing was committed
  }
  var ErrRefMoved = errors.New("the branch moved while committing")
  var ErrNotARepository = errors.New("not a directory")
  // CommitPaths commits exactly add (written) and remove (deleted) onto the
  // current ref of repo through a scratch index. It returns nil, nil when
  // repo is no git repository.
  func CommitPaths(repo, message string, add, remove []string, scratch string) (*Commit, error)
  ```

**Die Regeln, alle aus `vcs.py`:**

1. **Nur die genannten Pfade.** Über eine eigene Index-Datei
   (`GIT_INDEX_FILE=<scratch>/index`), nie über den Index des Nutzers
   (`:193-197`; unbrauchbare Scratch-Datei ⇒ Fehler).
2. **Weigerungen vor jedem Schreiben:** ein absoluter Pfad oder einer mit
   `..` (`:278-282`); ein Pfad in `add` und `remove` zugleich (`:177-179`);
   im Git-Verzeichnis existiert `rebase-merge`, `rebase-apply` oder
   `MERGE_HEAD` ⇒ `"<pfad> exists: finish the rebase or merge first"`
   (`:69`, `:265-275`). **Cherry-Pick wird nicht geprüft** — wie die
   Referenz; die Akte vermerkt es.
3. **Ablauf** (`:108-228`): `rev-parse --absolute-git-dir` (scheitert ⇒ `nil,
   nil`); `symbolic-ref --quiet HEAD`, sonst `HEAD` (abgelöst);
   `read-tree <old>`; je `add` `update-index --add -- <p>`; je `remove`
   `ls-files -z -- :(literal)<p>` und `update-index --force-remove`; ein
   nicht verfolgter `remove`-Pfad ist ein Fehler (`:231-251`); `write-tree`;
   gleicher Tree wie `old` ⇒ `&Commit{Head: old, Created: false}`;
   `commit-tree <tree> -p <old>` mit der Nachricht auf stdin;
   `update-ref <ref> <new> <old>`.
4. **`ErrRefMoved`**, wenn der symbolische Ref von HEAD sich während des
   Aufrufs geändert hat (`:220-221`) oder `update-ref` scheitert und der Ref
   nicht mehr `old` ist (`:223-227`). Ein anderer Fehler von `update-ref`
   ist ein gewöhnlicher Fehler mit git's stderr (`said`).
5. **Jeder Exitcode wird geprüft.** Anders als `approve.go:294-394`.
6. **Keine Identität setzen.** Sie kommt aus der Konfiguration des Repos
   (`gitenv` entfernt die Variablen, `:311`).

- [ ] **Step 1: Write the failing tests** — je Regel mindestens ein Test,
  benannt nach dem Python-Test, dem er entspricht (Kommentar mit Namen und
  Zeile aus `test_vcs.py`). Pflicht sind:
  `test_a_rebase_in_progress_is_refused` (`:414`),
  `test_an_unfinished_merge_is_refused` (`:427`), nur genannte Pfade werden
  committet, der Index des Nutzers bleibt unberührt, `ErrRefMoved` bei
  bewegtem Ref (Ref zwischen `read-tree` und `update-ref` über eine
  Testnaht versetzen), Weigerung bei Zweigwechsel, unveränderter Baum ⇒
  `Created == false`, Pfad mit führendem Leerzeichen, Pfad mit `*` (literal),
  nicht verfolgter `remove`-Pfad, kein Repo ⇒ `nil, nil`, geerbtes
  `GIT_DIR` wird ignoriert.

```go
func TestCommitPathsRefusesARebaseInProgress(t *testing.T) {
	// test_vcs.py:414 test_a_rebase_in_progress_is_refused
	repo := newRepo(t)
	writeFile(t, repo, "a.md", "changed\n")
	if err := os.Mkdir(filepath.Join(gitDir(t, repo), "rebase-merge"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := vcs.CommitPaths(repo, "m", []string{"a.md"}, nil, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "finish the rebase or merge first") {
		t.Fatalf("err = %v", err)
	}
	if head(t, repo) != initialHead(t, repo) {
		t.Fatal("a commit landed during a rebase")
	}
}
```

`newRepo` ist der Repo-Helfer, den `vcs_test.go` aus 3a schon hat; er wird
benutzt, nicht ein zweiter gebaut.

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/brain/vcs/ -run CommitPaths -v`
Expected: FAIL — `CommitPaths` nicht definiert.

- [ ] **Step 3: Implement** nach den sechs Regeln. Die Testnaht für den
  bewegten Ref ist eine paketprivate Funktionsvariable `beforeUpdateRef
  func()`, im Normalfall `nil`.

- [ ] **Step 4: Run**

Run: `go test ./internal/brain/vcs/ -cover`
Expected: PASS, 100 %.

- [ ] **Step 5: Commit**

```bash
git add internal/brain/vcs
```

```bash
git commit -m "feat(vcs): commit named paths onto the current ref"
```

---

### Task 4: `internal/brain/evidence` — Umzug der Belegbindung

**Files:**
- Create: `internal/brain/evidence/evidence.go` (aus
  `ultra-brain/pkg/maintenance/evidence.go`, 449 Zeilen)
- Test: `internal/brain/evidence/evidence_test.go` (aus `evidence_test.go`,
  419 Zeilen) und die fehlenden Fälle aus `tests/maintenance/test_evidence.py`
  (46)
- Referenz: `src/brain/maintenance/evidence.py` (677 Zeilen)

**Interfaces:**
- Produces:
  ```go
  type Segment struct{ Number, Kind, Label, Body string } // evidence.go:11
  type Claim struct{ Heading, Segment, Quote, Body string }
  type PackageError struct{ Msg string }
  func (e *PackageError) Error() string
  func Normalised(text string) string
  func ReadProposal(text string) []Claim
  func CheckEvidence(claims []Claim, segments []Segment) (passed []Claim, complaints []string)
  func ReadPackage(text string) ([]Segment, error)
  func FencedBlocks(text string) [][2]string // [info, body]
  ```
  Die Feldnamen folgen der Go-Form; weichen sie dort ab, gelten die der
  Go-Form, und dieser Block wird im Commit korrigiert.

**Die Regeln:**

1. **Umzug, nicht Neuschrift.** Datei und Tests kopieren, Paketname
   `evidence`, Importe auf loomux umstellen. Kein Verhalten ändern, bevor die
   Python-Fälle laufen.
2. **Die Python-Tests sind das Orakel.** Jeder Test aus `test_evidence.py`,
   der in `evidence_test.go` kein Gegenstück hat, wird übertragen, mit Name
   und Zeile im Kommentar. Pflicht: CRLF und einzelnes CR, die Zaunrunden
   C1/C2/I1–I4, Setext, Tab, HTML-Kommentar, Bidi-Steuerzeichen, Anzahl,
   Reihenfolge und Lücken der Segmente.
3. **Scheitert ein übertragener Test,** gilt Python: die Go-Form wird
   geheilt, und die Heilung steht in der Akte (Abweichungsliste, Spalte
   „Alt“ = Go-Form).
4. **Das Paket muss zum Schreiber aus 3a passen.** Ein Test rendert mit
   `maintenance.RenderPackage` (3a) ein Paket mit je einem D-, W- und
   Q-Segment, darunter ein Rumpf mit drei Backticks, und liest es mit
   `ReadPackage` zurück: Anzahl, Arten und Rümpfe gleich. Dieser Test
   importiert `maintenance` nur im Testpaket `evidence_test`.

- [ ] **Step 1:** Dateien kopieren, Paket umbenennen, Importe umstellen.
- [ ] **Step 2:** Run: `go test ./internal/brain/evidence/ -cover` —
  Expected: PASS für die umgezogenen Tests.
- [ ] **Step 3:** Die fehlenden Python-Tests übertragen (Regel 2) und den
  Rundlauftest (Regel 4) schreiben.
- [ ] **Step 4:** Run: `go test ./internal/brain/evidence/ -v` — Expected:
  jeder Fehlschlag ist eine Abweichung der Go-Form; heilen nach Regel 3.
- [ ] **Step 5:** Run: `go test ./internal/brain/evidence/ -cover` —
  Expected: PASS, 100 %.
- [ ] **Step 6: Commit** — Nachricht über eine Datei, weil sie die Herkunft
  nennt:

```
feat(evidence): bind every claim to a verbatim quote

Moved from ultra-brain pkg/maintenance/evidence.go with its tests; the
checks the Python reference has and the Go form lacked are carried over.
```

```bash
git add internal/brain/evidence docs/.superpowers/parity/stufe-3b.md
```

```bash
git commit -F <scratchpad>/commit-evidence.txt
```

---

### Task 5: `internal/brain/apply/patch.go` — Hunks exakt anwenden

**Files:**
- Create: `internal/brain/apply/patch.go` (aus `pkg/maintenance/patch.go`, 109)
- Test: `internal/brain/apply/patch_test.go` (aus `patch_test.go`, 210)
- Referenz: `apply.py:_collect` (`:1065`), `_hunks` (`:1077`), `_patch`
  (`:1115`), Kopf-Regex `:225`

**Interfaces:**
- Produces:
  ```go
  type Hunk struct {
      At       int
      Old, New []string
  }
  type RefusedError struct{ Msg string } // a proposal that cannot be applied
  func (e *RefusedError) Error() string
  func CollectDiff(body string) (string, error)          // _collect
  func ParseUnifiedDiff(diff string) ([]Hunk, error)      // _hunks
  func ApplyHunks(text string, hunks []Hunk) (string, error) // _patch
  ```

**Die Regeln (Python gilt):**

1. Der Diff ist der Rumpf des ersten Zauns aus `evidence.FencedBlocks`, dessen
   Info-Zeichenkette mit `diff` beginnt. Kein solcher Zaun ⇒ `*RefusedError`.
2. Kopf `^@@ -(\d+)(?:,(\d+))? \+…@@` (`:225`). Eine reine Einfügung
   (Anzahl 0) ankert bei `start`, nicht `start-1`. Zeilen, die mit `\`
   beginnen, werden übersprungen; eine leere Zeile ist Kontext.
3. **Exakt, ohne Unschärfe**, auf der LF-gefalteten Seite. Kein Kopf,
   überlappende Hunks, ein Hunk hinter dem Seitenende, ein unpassender
   Kontext ⇒ jeweils `*RefusedError`.
4. **`*RefusedError` aus diesem Weg schreibt nichts** — kein Vermerk, keine
   Flagge, kein Auditblock (Spec, „Zwei Wege zu `ProposalRefused`“). Das
   durchzusetzen ist Sache von Task 11; hier nur: jeder dieser Fehler ist
   vom Typ `*RefusedError`, nicht ein nackter `error` (die Go-Form gab
   nackte Fehler zurück).

- [ ] **Step 1:** Umzug wie Task 4, dann für jeden Fall aus
  `test_apply.py`, der Hunks prüft, einen Test, der den Fehlertyp mit
  `errors.As` verlangt.
- [ ] **Step 2:** Run: `go test ./internal/brain/apply/ -run "Hunk|Diff|Patch" -v` —
  Expected: FAIL für die Fehlertypen.
- [ ] **Step 3:** Fehler auf `*RefusedError` umstellen, `CollectDiff`
  ergänzen.
- [ ] **Step 4:** Run: `go test ./internal/brain/apply/ -cover` — PASS, 100 %.
- [ ] **Step 5: Commit**

```bash
git add internal/brain/apply
```

```bash
git commit -m "feat(apply): apply a proposed diff exactly or refuse it"
```

---

### Task 6: `internal/brain/apply/frontmatter.go` — gegen PyYAML gehoben

**Files:**
- Create: `internal/brain/apply/frontmatter.go` (aus `frontmatter.go`, 168)
- Test: `internal/brain/apply/frontmatter_test.go`, dazu
  `internal/brain/apply/testdata/frontmatter/*.{in,out}.md`
- Referenz: `apply.py:_advance` (`:1143-1178`), `_advance_register`
  (`:1181-1214`)

**Interfaces:**
- Consumes: `identity.Identity`, `identity.ReadIdentities`,
  `identity.RenderIdentities` (bzw. die loomux-Namen in
  `internal/brain/identity`).
- Produces:
  ```go
  type SourceUpdate struct {
      DocID       string
      ContentHash string
  }
  func AdvanceFrontmatter(page string, updates []SourceUpdate, reviewer string, now time.Time) (string, error)
  func AdvanceRegister(tsvPath, relative string, id identity.Identity) error
  func IsoFormat(t time.Time) string // Python datetime.isoformat() of a UTC time
  ```

**Die Regeln:**

1. Frontmatter ist Pflicht: `\A---\n(.*?)\n---[ \t]*\n` auf der LF-gefalteten
   Seite. Fehlt sie ⇒ Fehler.
2. Je passendem `sources[]`-Eintrag: `content_hash` neu, `revision` + 1.
   `generated.at` = `IsoFormat(now)`. An `verified` wird `{by: reviewer, at:
   IsoFormat(now)}` angehängt.
3. **`IsoFormat`** ist `datetime.isoformat()`: `2026-09-22T08:16:27.936837+00:00`,
   ohne Nachkommastellen, wenn die Mikrosekunden 0 sind. Nie `Z`, nie
   Millisekunden.
4. **Ausgabe wie `yaml.safe_dump(sort_keys=False, allow_unicode=True,
   default_flow_style=False)`.** Das ist das Risiko dieses Tasks (Spec,
   Befunde). Darum zuerst Goldens: Step 1 erzeugt sie **mit Python aus dem
   Tag**, dann wird yaml.v3 an sie gehalten. Unterschiede, die sich mit
   yaml.v3 nicht beheben lassen (Einrückung von Folgen, Anführungszeichen,
   Datumswerte), bekommt ein eigener kleiner Emitter für die Formen, die in
   Wiki-Frontmatter vorkommen — nicht eine Abweichung in der Akte.

- [ ] **Step 1: Goldens mit der Referenz erzeugen** — ein PEP-723-Skript im
  Scratchpad, ausgeführt mit dem Python von ultra-brain:

Run: `uv run --project ../ultra-brain python <scratchpad>/frontmatter_goldens.py`

Das Skript ruft für jede Eingabe in `internal/brain/apply/testdata/frontmatter/*.in.md`
`brain.maintenance.apply._advance` mit festem `now =
2026-09-22T08:16:27.936837+00:00`, Prüfer `human:tester` und einem Update je
`sources[]`-Eintrag (`content_hash = "sha256:" + "0"*64`) und schreibt
`*.out.md`. Eingaben: mindestens die Frontmatter von fünf echten Seiten aus
`docs/wiki/` (Quellen, Synthese, Thema, Entität, eine mit Umlauten und
Anführungszeichen), eine mit leerem `verified`, eine mit Kommentar (PyYAML
verliert ihn — die Golden zeigt es).

- [ ] **Step 2: Write the failing test** — ein Tabellen-Test über alle
  Paare, `AdvanceFrontmatter` mit denselben Werten, Ausgabe byte-gleich
  `*.out.md`.
- [ ] **Step 3:** Run: `go test ./internal/brain/apply/ -run Frontmatter -v` —
  Expected: FAIL (die Go-Form rendert anders oder existiert noch nicht).
- [ ] **Step 4:** Umzug, dann heben nach Regel 4, bis alle Goldens gleich sind.
- [ ] **Step 5:** `AdvanceRegister`: Revision + 1 und neuer Hash für eine
  `doc_id`, Rest des Registers byte-gleich; Test mit einem Register aus drei
  Zeilen.
- [ ] **Step 6:** Run: `go test ./internal/brain/apply/ -cover` — PASS, 100 %.
- [ ] **Step 7: Commit**

```bash
git add internal/brain/apply
```

```bash
git commit -m "feat(apply): advance frontmatter and register as the reference renders them"
```

---

### Task 7: `internal/brain/apply/protocol.go` — Texte des Protokolls

**Files:**
- Create: `internal/brain/apply/protocol.go`
- Test: `internal/brain/apply/protocol_test.go`
- Referenz: `apply.py:201-217` (Vermerke, `_ALLOWED`, `_REPLACEMENT`),
  `_safe` (`:1224-1264`), `_append_log` (`:1267`), `_append_audit`
  (`:1276-1299`), `_append` (`:1302`), `_unrecorded` (`:984`)

**Interfaces:**
- Produces:
  ```go
  const MovedNote, RefusedNote, AmendNote string // wörtlich aus apply.py:201-206
  func Safe(value string) string
  func LogLine(now time.Time, target string, applied int, caseID string) string
  type AuditEntry struct {
      Now                time.Time
      Target, CaseID     string
      Claims, Complaints []string
      Decided, Changed   string
  }
  func RenderAudit(e AuditEntry) string
  func Append(existing, block string) string
  func Unrecorded(audit, block string) bool
  ```

**Die Regeln:**

1. **Texte wörtlich aus `apply.py` kopieren**, mit Zeilenangabe im
   Kommentar. `MovedNote` ist mehrzeilig in `:201-204` zusammengesetzt; der
   Wert ist der zusammengesetzte String.
2. **`Safe`**: behält Zeichen aus `" .,:;/-_()…·"` und jedes Zeichen der
   Unicode-Kategorien `L*` und `N*` (ganz `N`, nicht nur `Nd` —
   `unicode.IsLetter` und `unicode.IsNumber`), ersetzt alles andere durch
   `·`, schneidet nach **200** Zeichen (Runen, nicht Bytes).
3. `LogLine`: `- <YYYY-MM-DD> — `<Safe(target)>`: <n> Behauptung(en) eingearbeitet (Fall `<id>`)`.
4. `RenderAudit`: `## <IsoFormat(now)> — <target> (Fall `<id>`)`, Leerzeile,
   `- vorgeschlagen: <n> Behauptung(en), <m> ohne Beleg`,
   `- entschieden: <Safe(decided)>`, `- tatsächlich geändert: <Safe(changed)>`,
   je Beschwerde `- verworfen: <Safe(c)>`.
5. `Append`: schneidet die abschließenden Zeilenumbrüche der vorhandenen
   Datei ab und verbindet mit genau einer Leerzeile.
6. `Unrecorded`: wahr, wenn derselbe Ausgang noch nicht protokolliert ist
   (`:984` genau lesen — verglichen wird der Block ohne Kopfzeile).

- [ ] **Step 1: Write the failing tests** — je Regel; für `Safe` eine
  Tabelle mit Steuerzeichen, Emoji, `²` (Kategorie `No`, bleibt), Bidi,
  201 Zeichen; Erwartungswerte **aus der Referenz erzeugt** (dasselbe
  Skript-Muster wie Task 6, Funktion `_safe`), nicht von Hand.
- [ ] **Step 2:** Run: `go test ./internal/brain/apply/ -run "Safe|Log|Audit|Append|Unrecorded" -v` — FAIL.
- [ ] **Step 3:** Implementieren.
- [ ] **Step 4:** Run: `go test ./internal/brain/apply/ -cover` — PASS, 100 %.
- [ ] **Step 5: Commit**

```bash
git add internal/brain/apply
```

```bash
git commit -m "feat(apply): render the audit and log lines of a decision"
```

---

### Task 8: `internal/brain/apply/place.go` — die Schreibschranke

**Files:**
- Create: `internal/brain/apply/place.go`
- Test: `internal/brain/apply/place_test.go`
- Referenz: `apply.py:_gate` (`:556-632`), `_normalised` (`:635`),
  `_preflight`, `_write`, `_record_case`, `_remove`

**Interfaces:**
- Produces:
  ```go
  // place writes only below its anchors and records every file it touched,
  // so an abort can name them.
  type place struct {
      anchor  string   // the vault
      wiki    string
      touched []string // relative to anchor, in write order
  }
  func (p *place) write(path, text string) error
  func (p *place) recordCase(path string, c maintenance.Case) error
  func (p *place) remove(dir string) error
  func (p *place) preflight(paths ...string) error
  ```

**Die Regeln:**

1. **Jeder Schreibzugriff geht durch `gate`.** `gate` verweigert: eine
   Pfadkomponente, die ein Link oder eine Junction ist; eine Komponente, die
   außerhalb des Ankers auflöst; einen Gerüstnamen (`_schema.md`, `index.md`,
   `log.md`, `audit.md`, `_identities.tsv`) als **Ziel**, auch in
   normalisierter Form — abschließende Punkte und Leerzeichen, ein
   `:`-Datenstrom, ein 8.3-Kurzname (`_normalised`). `log.md`, `audit.md`
   und `_identities.tsv` schreibt `place` über eigene, benannte Wege.
2. **`touched`** wird **vor** dem Schreiben ergänzt, nicht danach: auch ein
   halb gescheiterter Schreibvorgang steht im Abbruchhinweis.
3. Schreiben über `lock.ReplaceText` nur bei geänderten Bytes; Löschen eines
   Fallverzeichnisses über `os.RemoveAll` **mit** Fehlerprüfung (anders als
   `approve.go`, das `_ =` schreibt).
4. **`preflight`** prüft `log.md`, `audit.md`, `_identities.tsv` und das
   Fallverzeichnis, bevor irgendetwas geschrieben wird.
5. 8.3-Kurznamen gibt es nur unter Windows; ihre Auflösung liegt in einer
   Datei mit `//go:build windows` und einem Gegenstück für die übrigen
   Systeme. Die Tests für 8.3 laufen nur unter Windows (`t.Skip` sonst,
   mit Grund).

- [ ] **Step 1: Write the failing tests** — je Regel; die
  Einschluss-, Link- und Gerüstnamen-Tests aus `test_apply.py` übertragen,
  mit Name und Zeile. Junction-Tests mit `mklink /J` über `cmd /c` unter
  Windows, Symlink sonst.
- [ ] **Step 2:** Run: `go test ./internal/brain/apply/ -run "Gate|Place|Preflight" -v` — FAIL.
- [ ] **Step 3:** Implementieren.
- [ ] **Step 4:** Run: `go test ./internal/brain/apply/ -cover` — PASS, 100 %
  (plattformabhängige Zweige mit `//coverage:exempt` und Grund, wenn sie auf
  der Maschine des Tors nicht laufen).
- [ ] **Step 5: Commit**

```bash
git add internal/brain/apply
```

```bash
git commit -m "feat(apply): write only inside the vault and name every touched file"
```

---

### Task 9: `internal/brain/apply/resolve.go` — Tresor, Prüfzentrum, Wiki, Quellen

**Files:**
- Create: `internal/brain/apply/resolve.go`
- Test: `internal/brain/apply/resolve_test.go`
- Referenz: `apply.py:_resolve` (`:361-409`), `_wiki` (`:413-434`),
  `_layout` (`:656-676`), `_is_bundle`, `_check_target` (`:814-847`),
  `_target` (`:850-900`), `_resolve_sources` (`:155-186`)

**Interfaces:**
- Consumes: `config.ReadRegistry`, `config.Area`, der Manifestleser aus
  `internal/config` (`.loomux/config.toml`), `identity.ReadIdentities`.
- Produces:
  ```go
  type resolved struct {
      vault, review, wiki string
      area               config.Area
  }
  func resolve(casePath string, c maintenance.Case, areas []config.Area) (resolved, error)
  func checkTarget(target string) error
  func targetPath(r resolved, target string) (string, error)
  type sourceFile struct{ docID, path, register string }
  func resolveSources(areas []config.Area, docIDs []string) ([]sourceFile, error)
  ```

**Die Regeln:**

1. **Der Tresor** ist der nächste Vorfahr des Falls mit einer
   `.loomux/config.toml` (Referenz: `.brain.toml`; die Übersetzung steht in
   der Akte).
2. `[layout].review` muss relativ sein und darf kein `..` enthalten; der Fall
   muss im Prüfzentrum liegen.
3. **Das Wiki kommt aus der Registry**, aus dem Eintrag, dessen Scope
   `case.Area` ist. Liegt es außerhalb des Tresors, muss es `_schema.md`
   tragen (`_is_bundle`).
4. `checkTarget` verweigert: leer, Leerraum am Rand, Zeichen der Kategorien
   `Cc`, `Cf`, `Zl`, `Zp` (die Go-Form prüfte nur `Cc`).
5. `targetPath` verweigert: absolut, **jede** `..`-Komponente (die Go-Form
   suchte nur `"../"`, sodass `a/..` durchging), Gerüstnamen, außerhalb des
   Wikis, verlinkte Komponenten. Eine fehlende Seite ist ein Fehler.
6. **`resolveSources` sucht jede `doc_id` in den `_identities.tsv` aller
   registrierten Bereiche** und liefert `area.Path/relative`. Unbekannte
   `doc_id`s werden übersprungen (Referenz). Die Go-Form las nur das Register
   des Tresors.

- [ ] **Step 1: Write the failing tests** — je Regel, übertragen aus
  `test_apply.py` mit Namen; Pflicht: `a/..` als Ziel, `U+200E` (Cf) im
  Ziel, eine Quelle in einem zweiten Bereich außerhalb des Tresors.
- [ ] **Step 2:** Run: `go test ./internal/brain/apply/ -run "Resolve|Target|Sources" -v` — FAIL.
- [ ] **Step 3:** Implementieren.
- [ ] **Step 4:** Run: `go test ./internal/brain/apply/ -cover` — PASS, 100 %.
- [ ] **Step 5: Commit**

```bash
git add internal/brain/apply
```

```bash
git commit -m "feat(apply): resolve vault, review centre, wiki and sources of a case"
```

---

### Task 10: Ablehnen und Committen — `commit.go`, `reject.go`

**Files:**
- Create: `internal/brain/apply/commit.go`, `internal/brain/apply/reject.go`
- Test: `internal/brain/apply/commit_test.go`, `internal/brain/apply/reject_test.go`
- Referenz: `apply.py:_reject` (`:698-727`), `_commit`/`_report`
  (`:1318-1378`)

**Interfaces:**
- Consumes: `vcs.CommitPaths`, `vcs.ErrRefMoved` (Task 3); `place` (Task 8);
  `RenderAudit`, `Append`, `Safe` (Task 7); `resolve` (Task 9).
- Produces:
  ```go
  // commit lands the named paths and never fails the decision: a git
  // problem comes back as a warning.
  func commit(vault, caseID, said, message string, add, remove []string, scratch string) (sha, warning string)
  func reject(r resolved, p *place, c maintenance.Case, casePath, reviewer string, now time.Time, scratch string) (Result, error)
  ```

**Die Regeln:**

1. **`commit` wiederholt genau einmal** nach `vcs.ErrRefMoved`. Jeder andere
   Git-Fehler wird zur Warnung, nie zum Fehler. Die Warnungen wörtlich
   (`apply.py:1364-1377`):
   `<id>: <said>, but not committed (<err>)`,
   `<id>: no git repository in the vault; <said> but not committed`,
   `<id>: nothing to commit <on the first attempt|after the retry>; the case belongs back in the queue`.
   `said` ist `written` oder `decision recorded`.
2. **Ablehnen** braucht keinen Vorschlag und prüft keine Hashes: Auditblock
   anhängen, Fallverzeichnis löschen, Commit
   `Reject the proposed change to <Safe(target)>` mit **nur** `audit.md` in
   `add` und dem Fallverzeichnis in `remove`. `Result{Written: false,
   Dropped: nil}`.
3. **Geerbt:** Ablehnen rückt Revision und Hash der Seite nicht vor
   (Spec, Befunde). Ein Test hält das Verhalten fest und nennt die
   Akte im Kommentar.

- [ ] **Step 1: Write the failing tests** — Wiederholung nach bewegtem Ref
  (Naht aus Task 3), zweites `ErrRefMoved` ⇒ Warnung, kein Repo ⇒ Warnung,
  unveränderter Baum ⇒ Warnung „nothing to commit“, Ablehnen schreibt Audit
  und löscht den Fall, Ablehnen committet nur `audit.md`.
- [ ] **Step 2:** Run: `go test ./internal/brain/apply/ -run "Commit|Reject" -v` — FAIL.
- [ ] **Step 3:** Implementieren.
- [ ] **Step 4:** Run: `go test ./internal/brain/apply/ -cover` — PASS, 100 %.
- [ ] **Step 5: Commit**

```bash
git add internal/brain/apply
```

```bash
git commit -m "feat(apply): reject a proposal and commit a decision with one retry"
```

---

### Task 11: `internal/brain/apply/approve.go` — Anwenden

**Files:**
- Create: `internal/brain/apply/approve.go`
- Test: `internal/brain/apply/approve_test.go`
- Referenz: `apply.py:approve` (`:307-358`), `_apply` (`:730-811`),
  `_guard` (`:913-941`), `_guard_sources` (`:944-981`), `_refuse`
  (`:1032-1062`), `_same_file` (`:1013`); Tests `test_apply.py` (126)

**Interfaces:**
- Consumes: alles aus Task 4–10; `maintenance.ReadCase`, `maintenance.WriteCase`.
- Produces:
  ```go
  var Decisions = []string{"approve", "reject"}
  type Result struct {
      Case     maintenance.Case
      Decision string
      Written  bool
      Commit   string // "" when nothing was committed
      Warning  string
      Dropped  []string
  }
  // ApplyError, not Error: embedded as a field named Error, it would shadow
  // the promoted Error method, and TargetMoved would not be an error.
  type ApplyError struct {
      Msg   string
      Dirty []string
  }
  func (e *ApplyError) Error() string
  type TargetMoved struct{ ApplyError }
  type SourceMoved struct{ ApplyError }
  type ProposalRefused struct{ ApplyError }
  type Options struct {
      Amend    string // "" for the proposal in the case
      Decision string
      Reviewer string
      Now      time.Time
      Scratch  string
  }
  func Approve(casePath string, areas []config.Area, o Options) (Result, error)
  ```

**Die Regeln:**

1. **Eingang:** `Decision` ∈ `Decisions` (`defer` ist **kein** Wert hier —
   anders als die Go-Form; die CLI behandelt es). `Reviewer` =
   `human:<nicht leer>`. Ein unlesbarer Fall wird `*ApplyError`.
2. **`Dirty`** jedes `*ApplyError` und jedes `os`-Fehlers ist `place.touched`
   (`:343-358`).
3. **Zielwache** (`_guard`): Hash der Seite ≠ `target_hash` ⇒ `note =
   MovedNote` in `case.toml`, Auditblock nur, wenn `Unrecorded`, dann
   `*TargetMoved`. **Quellwache** (`_guard_sources`) entsprechend mit
   `*SourceMoved`, über `resolveSources` (alle Bereiche). Beide Blöcke
   tragen die Zahl der Behauptungen aus `proposal.md` (die Go-Form gab
   `nil`).
4. Eine Nachbesserung (`Amend`), deren `content_hash` gleich dem des
   Vorschlags ist, wird verweigert. Fehlt die Quelldatei ⇒
   `<pfad>: no proposal to approve`.
5. **Zwei Wege zu `*ProposalRefused`:**
   - **Belegprüfung scheitert** (nichts bestanden) ⇒ `_refuse`: `note =
     RefusedNote` bzw. `AmendNote` bei einer Nachbesserung; `manual = true`
     **nur** beim eigenen Vorschlag; Auditblock mit `- entschieden:
     verworfen (Evidenzbindung)` nur, wenn `Unrecorded`.
   - **`*RefusedError` aus `patch`** (Task 5) ⇒ `*ProposalRefused` mit
     derselben Meldung, und es wird **nichts** geschrieben.
   Ein Test je Weg prüft die Dateiwelt danach.
6. **Anwenden:** Hunks auf die Seite, `AdvanceFrontmatter`, je betroffenem
   Register `AdvanceRegister` (nur Register **im** Tresor werden gestagt),
   `LogLine`, Auditblock, Fallverzeichnis löschen, Commit
   `Land the reviewed change to <Safe(target)>` mit Seite, `log.md`,
   `audit.md` und Registern in `add`, Fallverzeichnis in `remove`.
   `Result{Written: true, Dropped: <Beschwerden der verworfenen Behauptungen>}`.
7. Die Reihenfolge der Prüfungen ist die der Referenz; ein Fall mit mehreren
   Mängeln meldet denselben ersten wie Python.

- [ ] **Step 1: Write the failing tests** — die Tests aus `test_apply.py`
  übertragen, gruppiert nach Regel, je mit Name und Zeile im Kommentar.
  Pflicht: Erfolg (Seite, Log, Audit, Register, Löschung, Commit), bewegtes
  Ziel, bewegte Quelle in einem zweiten Bereich, doppelter Auditblock
  bleibt aus, Belegprüfung scheitert (Vermerk, Flagge, Audit), Hunk passt
  nicht (nichts geschrieben), Nachbesserung gleich Vorschlag, `Dirty` nach
  einem Schreibfehler mitten im Anwenden.
- [ ] **Step 2:** Run: `go test ./internal/brain/apply/ -run Approve -v` — FAIL.
- [ ] **Step 3:** Implementieren, Regel für Regel.
- [ ] **Step 4:** Run: `go test ./internal/brain/apply/ -cover` — PASS, 100 %.
- [ ] **Step 5: Commit**

```bash
git add internal/brain/apply
```

```bash
git commit -m "feat(apply): approve a proven proposal and land it in one commit"
```

---

### Task 12: `loomux cases` und `loomux case`

**Files:**
- Create: `internal/brain/maintenance/lookup.go` (nur `FindCase`, aus
  `pkg/maintenance/lookup.go:66`)
- Create: `internal/cli/cases.go`
- Modify: `internal/cli/commands.go`, `internal/cli/maintenance.go`
- Test: `internal/brain/maintenance/lookup_test.go`, `internal/cli/cases_test.go`
- Referenz: `cli.py:_cases` (`:1136-1193`), `_find_case` (`:1196-1225`),
  `_case` (`:1228-1281`), `_withheld` (`:1284-1305`), `_show` (`:1308-1320`),
  Zeilen `:921-939`; Go-Form `format.go`, `privacy.go`, `cases.go:28`

**Interfaces:**
- Consumes: `maintenance.ReviewRoot`, `listCases`, `caseAddresses`,
  `caseLine` (3a).
- Produces:
  ```go
  func FindCase(reviewRoot, identifier string) (string, error) // case directory
  ```
  CLI: `"cases": casesCommand`, `"case": caseCommand` in `commands.go`.

**Die Regeln:**

1. **`FindCase`** vergleicht Verzeichnisnamen exakt, nie als Glob; die drei
   Meldungen wörtlich aus `:1196-1225` (kein Prüfzentrum / nicht gefunden /
   mehrdeutig mit beiden Pfaden).
2. **`cases`** benutzt die 3a-Helfer. Neu: die stderr-Warnung, wenn `id` vom
   Verzeichnisnamen abweicht (`:1166-1179`, wörtlich), und die Reihenfolge
   wie Pythons `sorted(rglob)` unter Windows — ohne Beachtung der
   Groß-/Kleinschreibung (`cases.go:28 lessCasePath` zieht um). Ist
   `caseAddresses` schon so sortiert, bleibt es dabei, und ein Test hält
   `alpha-case` vor `Zeta-case` fest.
3. **`case <id> [--package]`**: die Zeilen in der Reihenfolge der Referenz
   (`Fall …`, `Bereich:`, `Ziel:`, je Quelle `Quelle:`, dann die Hinweise,
   `Vermerk:`). **Zurückgehalten** wird bei `local_only` im Fall, fehlendem
   Manifest oder `privacy_mode == "local_only"` — dann ohne `--package` der
   Block `===== zurückgehalten =====` mit `Bewusst ausgeben: loomux case
   --package <dir>` (Abweichung zur Referenz, übersetzt durch `[[stdout]]`
   aus Task 2). Sonst `package.md`, `proposal.md` und, wenn gesetzt,
   `superseded-proposal.md`, je mit Leerzeile und `===== <name> =====`, oder
   `(nicht vorhanden: <pfad>)`.
4. `--package` darf **vor und nach** der ID stehen, wie bei argparse
   (`cli.py:601-611`). Go's `flag` hört an der ersten Nicht-Flagge auf; die
   Flaggen werden darum vor dem Parsen einsortiert, wie `approve` es braucht
   (Task 13, Regel 1) — ein gemeinsamer Helfer, nicht zwei. Die Go-Form
   verweigerte `--package` nach der ID mit Exit 1; das gehört in Task 0 in
   die Liste der Abweichungen der Go-Form.
5. **Die Reihenfolge ohne Groß-/Kleinschreibung** (Regel 2) ist Pythons
   Verhalten unter Windows. loomux trägt sie auf jedes System; die Akte hält
   das als bewusste Abweichung fest.
6. Exit: 0; 1 bei fehlendem Prüfzentrum, unbekanntem Fall, unlesbarer
   Fallakte; 2 bei Aufruffehlern.

- [ ] **Step 1: Write the failing tests** — `lookup_test.go` je Meldung;
  `cases_test.go` mit einer Welt aus `newReconcileWorld` (3a,
  `maintenance_test.go:48`): Liste, leere Liste (`keine offenen Fälle`),
  umbenannte ID, Reihenfolge, `case` offen, zurückgehalten, mit
  `--package`, fehlende Datei, unbekannt, mehrdeutig.
- [ ] **Step 2:** Run: `go test ./internal/cli/ ./internal/brain/maintenance/ -run "Cases|CaseCommand|FindCase" -v` — FAIL.
- [ ] **Step 3:** Implementieren; `format.go` und `privacy.go` aus ultra-brain
  dienen als Vorlage, gilt aber die Referenz.
- [ ] **Step 4:** Run: `go test ./internal/cli/ ./internal/brain/maintenance/ -cover` — PASS, 100 %.
- [ ] **Step 5: Commit**

```bash
git add internal/cli internal/brain/maintenance
```

```bash
git commit -m "feat(cli): list waiting review cases and show one"
```

---

### Task 13: `loomux approve`

**Files:**
- Create: `internal/cli/approve.go`
- Modify: `internal/cli/commands.go`
- Test: `internal/cli/approve_test.go`
- Referenz: `cli.py:613-623`, `_approve` (`:1334-1387`), `_reviewer`
  (`:1323-1331`), `_report_decision` (`:1408-1429`), der Abbruchhinweis
  (`:1390-1405`), `_technical_update` (`:1432-1516`)

**Interfaces:**
- Consumes: `apply.Approve`, `maintenance.FindCase`, `maintenance.ReviewRoot`,
  `reindexCommand` (3a), der Flaggen-Helfer aus Task 12.
- Produces: `"approve": approveCommand`.

**Die Regeln:**

1. `approve <id> [--amend PATH | --reject | --defer]`; die drei schließen
   sich aus, Aufruffehler ⇒ Exit 2. Flaggen dürfen vor und nach der ID
   stehen (die Referenz nimmt beides über argparse).
2. **`--defer` ruft `apply` nie:** Fall lesen und prüfen, bei abweichender
   ID warnen, `Fall <dir> zurückgestellt; er bleibt unverändert in der
   Warteschlange.`, Exit 0.
3. **Prüfer** `human:<Benutzername>` (`os/user`, Domänenteil abgeschnitten,
   wie `getpass.getuser()` ihn liefert). **Scratch**
   `<state>/maintenance/index`.
4. **Fehler** (`*apply.ApplyError` und Unterarten, `os`-Fehler): stderr
   `error: <msg>`, bei `Dirty` der Hinweis wörtlich aus `:1390-1405` und je
   Datei eine Zeile mit zwei Leerzeichen Einzug; Exit 1.
5. **Bericht:** stdout `Fall <id>: <approve|reject>`, je verworfener
   Behauptung `  verworfene Behauptung: <c>`, bei Commit
   `committet als <sha>`; sonst stderr
   `<geschrieben|entschieden>, aber nicht committet: <warnung>`. **Exit 0,
   auch wenn der Commit scheiterte.**
6. **Nachlauf, nur wenn `Written`:** `reindexCommand(nil, nil, io.Discard,
   stderr)` — **einmal, ohne eigenen `catchUp` davor**. `reindex` holt den
   Abgleich selbst nach (`catchUpBeforeIndexing`, `internal/cli/index.go:104`):
   scheitert er, meldet es auf stderr und indiziert **nicht**; eröffnet er
   Fälle, listet es sie auf stderr und indiziert weiter. Das ist genau die
   Form von `_technical_update`. Ein zweiter `reconcile` davor liefe doppelt.
   Der Exit von `reindex` ändert den Exit von `approve` nicht. Das lokale Modell
   gibt es nicht; ein `local_only`-Fall bekommt `manual = true` (Regel aus
   3a). Den Dienst anzuhalten (`_ask_daemon_to_reload`) entfällt: `serve`
   liest den Index je Anfrage — die Akte hält es fest, nachdem Step 1 es
   am Code von `internal/serve` geprüft hat.

- [ ] **Step 1: Write the failing tests** — je Regel; Nachlauf mit einer
  Welt, in der `reconcile` scheitert (kein Prüfzentrum erklärbar), und einer,
  in der er einen neuen Fall eröffnet.
- [ ] **Step 2:** Run: `go test ./internal/cli/ -run Approve -v` — FAIL.
- [ ] **Step 3:** Implementieren.
- [ ] **Step 4:** Run: `go test ./internal/cli/ -cover` — PASS, 100 %.
- [ ] **Step 5: Commit**

```bash
git add internal/cli
```

```bash
git commit -m "feat(cli): decide a review case with approve, reject or defer"
```

---

### Task 14: Fallsuite 3b — aufzeichnen, übersetzen, abspielen

**Files:**
- Create: `testdata/cases/3b-worlds/*`, `testdata/cases/3b-source/*`,
  `testdata/cases/3b/*`, `testdata/cases/3b-map.toml`
- Create: `internal/cli/cases_3b_test.go`
- Modify: `testdata/cases/README.md`, `docs/.superpowers/parity/stufe-3b.md`

**Die Fälle** (Spec, „Parität 3b“; je Welt ein Tresor mit Git, Prüfzentrum,
Wiki, Registry):

| Befehl | Fälle |
|---|---|
| `cases` | leer · drei Fälle · Groß-/Kleinschreibung · umbenannte ID · unlesbare Akte · kein Prüfzentrum |
| `case` | offen · zurückgehalten · `--package` · fehlender Vorschlag · unbekannt · mehrdeutig |
| `approve` | Erfolg mit Commit · Nachbesserung · Ablehnen · Zurückstellen · bewegtes Ziel · bewegte Quelle · Belegprüfung scheitert · Hunk passt nicht · Rebase läuft · kein Repo · leere Argumente · unbekannter Fall |

- [ ] **Step 1: Welten bauen** unter `testdata/cases/3b-worlds/`, je mit
  `git.toml`; jede `approve`-Welt mit einem Paket, das `maintenance.RenderPackage`
  erzeugt hat, damit die Segmentgrenzen die aus 3a sind.
- [ ] **Step 2: Aufzeichnen** mit der Referenz, je Fall ein Aufruf:

Run: `go run ./cmd/loomux dev record-case --exe ../ultra-brain/.venv/Scripts/brain-mcp.exe --world testdata/cases/3b-worlds/<welt> --out testdata/cases/3b-source/<verb>/<name> --git-after --cmd "brain-mcp approve <id>"`

Die genauen Flaggen von `record-case` stehen in `internal/cli/dev.go`;
weicht die Zeile ab, gilt `dev.go`. `--git-after` nur für `approve`.

- [ ] **Step 3: Übersetzen** — `3b-map.toml` mit `[[command]]`
  (`brain-mcp cases` → `loomux cases`, `brain-mcp case` → `loomux case`,
  `brain-mcp approve` → `loomux approve`), `manifests = "verbatim"`,
  `[[stdout]]` aus Task 2.

Run: `go run ./cmd/loomux dev import-cases --from testdata/cases/3b-source --to testdata/cases/3b --map testdata/cases/3b-map.toml`

- [ ] **Step 4: Suite schreiben** nach dem Muster von `cases_3a_test.go`:
  `wantCases3b` fest, `expected3b` für jede freigegebene Abweichung mit
  Grund, derselbe `RunFunc` (Zustand, Konfiguration, `cases.GitEnv`,
  aufgezeichnete Suchmaschine).

- [ ] **Step 5: Abspielen**

Run: `go test ./internal/cli/ -run TestCases3b -v`
Expected: PASS, oder jeder Unterschied steht mit Grund in der
Abweichungsliste der Akte und in `expected3b`. Ein Unterschied in einer
geschriebenen Datei oder in `git.after` wird **geheilt**, nicht eingetragen,
es sei denn, er ist eine der Übersetzungen der Akte.

- [ ] **Step 6: Commit**

```bash
git add testdata/cases internal/cli/cases_3b_test.go docs/.superpowers/parity/stufe-3b.md
```

```bash
git commit -m "test(cases): hold cases, case and approve to the recorded reference"
```

---

### Task 15: **Halt** — Selbstnutzung

**Menschlicher Schritt zuerst.** Das Prüfzentrum dieser Maschine ist
`brain-knowledge/95 Prüfzentrum/`, unversioniert, ohne Remote (Spec,
„Selbstnutzung“). `approve` committet in dieses Repo.

- [ ] **Step 1:** `bin/loomux.exe cases` und, falls ein Fall wartet,
  `bin/loomux.exe case <id>` gegen die echte Registry laufen lassen; die
  Ausgabe mit `uv run --project ../ultra-brain brain cases` vergleichen und
  das Ergebnis in die Akte schreiben.
- [ ] **Step 2: Den Menschen fragen und warten:** ob ein echter Fall mit
  `loomux approve` entschieden werden soll, und welcher. Ohne Fall oder ohne
  Freigabe hält die Akte fest, dass `approve` nur über die Fallsuite belegt
  ist, und warum.

---

### Task 16: Mutationsrunde, Messungen, Startzeit

- [ ] **Step 1: Mutationen**

Run: `./bin/loomux.exe dev mutants ./internal/brain/apply`

Run: `./bin/loomux.exe dev mutants ./internal/brain/vcs`

Run: `./bin/loomux.exe dev mutants ./internal/brain/evidence`

Jeder Überlebende bekommt einen Test oder einen Eintrag in der Akte mit Grund.

- [ ] **Step 2: Messen** — Median aus 10 warmen Läufen, je gegen die
  Referenz: `loomux cases` gegen `uv run brain cases`, `loomux case <id>`
  gegen `uv run brain case <id>`, `loomux approve --defer <id>` gegen
  `uv run brain approve --defer <id>`, auf einer Wegwerfwelt. Eintrag in
  `docs/en/benchmarks.md` und `docs/de/benchmarks.md` (Datum, Uhrzeit, was,
  Vergleich, kalt und warm).

- [ ] **Step 3: Startzeit**

Run: `$env:GODEBUG="inittrace=1"; ./bin/loomux.exe --version`
Expected: kein neues Paket-Init mit mehr als 500 Allokationen; der Tor-Test
aus „Zonenfreier Startpfad“ bleibt grün.

- [ ] **Step 4:** Run: `go run ./cmd/loomux check precommit` — Expected: grün,
  Coverage 100 %.

- [ ] **Step 5: Commit**

```bash
git add docs internal
```

```bash
git commit -m "test(apply): cover the mutants that survived and record the timings"
```

---

### Task 17: Doku und Migrationsplan

**Files:**
- Modify: `docs/en/migration.md`, `docs/de/migration.md` (3b ✅, Priorität
  „—“; 3c und 4 nennen `3b ✅`; die Funktionen „Abgleich“ und neu
  „Prüfzentrum entscheiden“)
- Modify: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`
  (Teilstufentabelle 3b ✅; Reihenfolgetabelle)
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md` (Abschnitte
  `cases`, `case`, `approve`)
- Modify: `README.md`, `README.de.md` (aktive Befehle, Stand der Migration)

- [ ] **Step 1:** Alle Dateien nachführen, beide Sprachen gleich.
- [ ] **Step 2:** Run: `go run ./cmd/loomux check precommit` — grün
  (`lint/wiki` eingeschlossen).
- [ ] **Step 3: Commit**

```bash
git add docs README.md README.de.md
```

```bash
git commit -m "docs: document deciding review cases and mark it done"
```

## Fertig, wenn

1. alle übersetzten Fälle von 3b grün oder freigegeben in `parity/stufe-3b.md`,
2. Coverage 100 %, jeder Ausschluss begründet,
3. die Mutationsrunde über `apply`, `vcs` und `evidence` gelaufen,
   Überlebende dokumentiert,
4. die drei Messungen eingetragen,
5. das loomux-Repo fährt `cases` und `case` gegen die echte Registry, und
   `approve` ist entweder an einem echten Fall gelaufen oder die Akte sagt,
   warum nicht (Task 15).
