# ulflow M1, Welle 3 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Die Tasks laufen nacheinander, ein Subagent je Task, alle im Harness-Worktree. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `ulflow` ist ein Binary mit `run`, `resume`, `replay`, `show`, `list` und `--option`, das den Beispiel-Flow der Spec gegen das Fake-Modell zu einem byte-genauen Golden-Journal führt. `ulguard hook session-start` meldet einen wartenden Go-Lauf mit `ulflow resume`. Die Install-Skripte bauen `ulflow` mit, und alle drei Binaries sind in beiden Ständen neu gebaut.

**Architecture:** Task 0 zieht drei kleine Verträge nach, die `cmd/flow` braucht: Laufnummern werden über die Marke exklusiv beansprucht, `NewCatalog` lehnt ein nil-Prädikat ab, und `Runner.Resume` lehnt unter Replay ein offenes Tor ab. Task 1 lässt den Session-Start-Hook die Marke lesen. Task 2 baut `flowload.Params` und den Beispiel-Flow als Testdatei. Task 3 baut `cmd/flow` als dünne Kante über `internal/`: `cli(args, Deps)` bekommt Uhr, Modellfabrik und Bausteine hineingereicht, `main` reicht die echten. Task 4 schreibt das Golden-Journal. Task 5 erledigt Installation, Neubau, Messung und Befunde.

**Tech Stack:** Go mit dem Sprachstand `go 1.22` aus `go.mod` (installiert ist go1.27.0, maßgeblich ist `go.mod`), `flag`, `encoding/json`, `github.com/BurntSushi/toml` v1.6.0 (schon im Modul), git auf dem PATH, PowerShell 7 für `scripts/install.ps1`.

**Spec:** `docs/.superpowers/specs/2026-09-11-ulflow-laufzeit-design.md`, Abschnitte „Verhaltensvertrag", „Das Flow-Format", „Modellwahl und Modell-Port", „Journal", „Kommandozeile", „Installation", „Wellen" (Welle 3) und „Nachweis".

**Vorgänger:** `docs/.superpowers/plans/2026-09-11-ulflow-welle-2.md` und dessen Befunde in `docs/.superpowers/plans/2026-09-11-ulflow-welle-2-befunde.md`, Abschnitt „Für den Plan von Welle 3".

## Global Constraints

- Kommentare, Bezeichner, Fehlermeldungen und Commit-Nachrichten englisch.
- Sprachstand `go 1.22`: kein range-over-func, kein `maps.Keys`, kein `slices.Collect`, keine API ab Go 1.23.
- TDD: jeder Code-Schritt beginnt mit einem Test, der vorher rot ist, und der Grund des Rots wird gelesen.
- 100 % Coverage in jedem Paket, das der Task anlegt oder ändert; `go test -cover` zeigt `coverage: 100.0% of statements`. Einzige Ausnahme ist `main()` in `cmd/flow`, mit Kommentar begründet wie in `cmd/init/main.go`.
- Vor jedem Commit `gofmt -l` auf die Dateien des Tasks: leere Ausgabe.
- Kein `Co-Authored-By` und keine Nennung eines Modells im Commit. Mehrzeilige Nachrichten über eine Datei und `git commit -F`; die Datei danach löschen.
- Vor jedem Commit `git rev-parse --abbrev-ref HEAD`, `git log --oneline -1` und `git diff --cached --stat` lesen; nur die Dateien stagen, die der Task nennt.
- Kein Push. Kein `--no-verify`. Kein `--amend`.
- Keine Änderung an `go.mod` oder `go.sum`.
- Kein Paket unter `internal/` ruft `time.Now`. In `cmd/flow` steht es genau einmal, in `production`.
- Dateien mit Backslashes im Inhalt mit Write oder Edit schreiben, nie über ein Bash-Heredoc.
- Ein Shell-Befehl je Schritt. Jeder Befehl nennt sein Verzeichnis mit `cd "<Worktree>" && …` oder `git -C "<Worktree>"`.
- Worktree für alle Tasks: `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness`, Zweig `feature/agent-harness`.
- Meldet ein Hook einen Befund über ein anderes Verzeichnis als diesen Worktree, wird er berichtet und nicht behoben.

## Entscheidungen, die dieser Plan trifft

Die ersten zehn hat der Nutzer am 2026-09-14 freigegeben.

1. **Nacheinander, ein Subagent je Task, im Harness-Worktree.** Keine Lane-Worktrees: Die Spec nennt Welle 3 nicht teilbar, und die drei Vorarbeiten sind zu klein, als dass sich `uv sync` und dmypy-Abbau in eigenen Worktrees lohnen.
2. **Der Hook liest die Marke über `runs.ReadMarker`.** Mit `runtime="go"` heißt die Zeile `ulflow resume <id> --answer …`, sonst bleibt sie `ultraloom resume`. Gemessen am 2026-09-14 gegen `9c0516d`: `internal/runs` in `ulguard` gelinkt kostet 3.584 Byte (5.947.904 → 5.951.488) und im Median von 25 warmen `ulguard status` 0,9 ms (49,2 → 50,1 ms, Streuung 46,9–58,7 ms). Der Probe-Import rief keine Funktion auf; Task 5 misst nach Task 1 noch einmal mit echter Nutzung.
3. **Eine Laufnummer wird über die Marke beansprucht.** `WriteMarker` legt die Datei mit `O_EXCL` an, `runs.Claim` rechnet bei einer schon vorhandenen Marke `NextID` neu und versucht es höchstens zehnmal. Das Journal entsteht erst danach.
4. **`--option name=wert`:** `int` über `strconv.Atoi`, `bool` genau `true` oder `false`, `string` roh, `list[string]` als JSON-Liste (`--option tags=["a","b"]`). Ein unbekannter Name oder ein Wert, der nicht zum Typ passt, ist eine Ablehnung vor dem Start mit Exit 1. Das leistet `flowload.Params(graph, options)`: Vorgaben plus Optionen.
5. **`cli(args, Deps)`** bekommt Ausgabeströme, Uhr, Modellfabrik `func(provider string) (model.Model, error)` und Bausteine. Die echte Fabrik antwortet `no adapter for provider <anbieter> yet`, und zwar bevor ein Lauf eine Datei hat. Tests reichen `model.Fake` hinein.
6. **Golden-Journal ist der Beispiel-Flow der Spec** (agent → gate → exit): `run` pausiert am Tor, `resume --answer "no: too thin"` endet mit Exit 4. Derselbe Flow ist der zweite Verbindungstest des Laders und schließt den Befund, dass `exit` nie gegen den echten Katalog geladen wurde.
7. **Ausgabe wie `cli.py`:** `run <id>: <status>`, danach Frage oder Detail. `show` mit den Spalten von `cli.py:455–456`. Warnungen als `warning: …` auf stderr. Exit 0 fertig, 1 Fehler oder Ablehnung, 2 Aufruffehler, 3 pausiert, sonst der Code des Bausteins.
8. **Ablehnungen bei `resume` und `replay` nach `cli.py:311–415`**, mit `ulflow` statt `ultraloom`. `Runner.Resume` lehnt unter Replay ein offenes Tor ab wie `Run`.
9. **Installation:** beide Skripte bauen `ulflow`, `install.ps1` legt den Bash-Shim an, `.gitignore` kennt `ulflow`. Neubau von `ulflow`, `ulguard`, `ulinit` im Worktree und in `~/go/bin`. `ulflow --version` sagt `0.1.0`.
10. **Messungen** kommen nach `docs/benchmarks.md` und `docs/benchmarks.de.md`.

Beim Schreiben dieses Plans dazugekommen:

11. **`NextID` zählt Marken mit.** Ein Lauf beansprucht seine Nummer mit der Marke, bevor sein Journal existiert; eine Zählung nur über Journale gäbe dieselbe Nummer erneut aus und liefe in `Claim` zehnmal gegen dieselbe Marke. Python zählt nur Journale (`cli.py:143–148`); das bleibt so, weil Python unverändert bleibt.
12. **Ein Lauf, den der Runner vor seinem ersten Schritt ablehnt, hinterlässt keine Marke.** `startChecks` liegen im Runner und laufen erst nach `Claim`. Findet `cmd/flow` nach einem Go-Fehler kein Journal, löscht es die Marke wieder. Ein Lauf mit Journal behält sie.
13. **M1 führt einen Anbieter je Flow.** `flow.Env` trägt ein Modell. Ein Flow, dessen Agentenknoten zwei Anbieter auflösen, wird mit Namen beider abgelehnt; mehrere Adapter sind M2.
14. **Keine Laufzeit setzt Läufe der anderen fort.** Eine Marke ohne `runtime="go"` wird von `resume` und `replay` abgelehnt: Die Eingabe-Hashes beider Seiten unterscheiden sich (Spec, „Bewusst geändert"), ein Nachgehen fände nichts und arbeitete jeden Knoten neu.
15. **`replay` fragt keine Modellfabrik.** Ein Replay führt keinen Knoten aus.
16. **`list`:** eine Zeile je Flow, `%-24s %-8s %s` mit Name, Herkunft und `ok` oder den Befunden; weitere Befundzeilen um vier Leerzeichen eingerückt.
17. **Der Beispiel-Flow weicht in zwei Punkten vom Text der Spec ab:** ohne das nie geschriebene Feld `notes` und ohne `max_visits` am Tor, das auf keinem Zyklus liegt. Die Anweisung nennt `{{max_rounds}}`, die Frage `{{count}}`, damit Platzhalter aus Parametern und Zustand beide geprüft werden.
18. **Eine Frage aus einer Datei endet mit ihrem Zeilenumbruch.** `cmd/flow` schneidet ihn bei der Ausgabe ab, damit keine Leerzeile folgt; das Journal behält die Bytes.
19. **Eine Marke, die der Hook nicht lesen kann,** wird wie ein beschädigtes Journal auf stderr benannt, und der Lauf wird trotzdem gemeldet, mit `ultraloom resume`.

## Außerhalb dieses Plans

- Die Lader-Politur aus den Befunden der Welle 2: Meldung `known names:`, unbekannte Tabellen auf oberster Ebene, der Kommentar zu Graphregel 5, Tokens auf Pausen-, Exit- und Agenten-Fehlerpfad. Ein eigener kleiner Plan danach.
- Echte Adapter für `claude -p` und `agy -p` (M2).
- Jede Änderung an Python.

## Ablauf

1. **Task 0** durch einen Subagenten. Orchestrator prüft: `go test -cover ./internal/runs/ ./internal/flow/ ./internal/runner/`, `git log --oneline -1`, Diff gegen die Dateiliste.
2. **Vor Task 1** baut der Orchestrator den heutigen Hook zum Vergleich: `cd "<Worktree>" && go build -o "<Scratchpad>/ulguard-before.exe" ./cmd/guard`.
3. **Tasks 1 bis 4** nacheinander, je ein Subagent mit `model: "opus"`, dem vollständigen Text seines Tasks, den Global Constraints und dem Worktree-Pfad. Einen `effort` nimmt der Agent-Aufruf nicht an. Nach jedem Task prüft der Orchestrator Diff, Tests und Coverage selbst, bevor der nächste startet, und lässt ein Review laufen.
4. **Task 5** durch den Orchestrator.
5. Nach jedem Subagenten: `git -C "<Worktree>" ls-remote origin feature/agent-harness` lesen. Erwartet: leer.

---

### Task 0: Verträge nachziehen

**Files:**
- Modify: `internal/runs/marker.go` (`WriteMarker` mit `O_EXCL`, Helfer `writing`)
- Modify: `internal/runs/id.go` (`NextID` zählt Marken mit)
- Create: `internal/runs/claim.go`
- Test: `internal/runs/marker_test.go`, `internal/runs/id_test.go`, `internal/runs/claim_internal_test.go`
- Modify: `internal/flow/catalog.go` (nil-Prädikat)
- Test: `internal/flow/catalog_test.go`
- Modify: `internal/runner/runner.go` (`Resume` unter Replay)
- Test: `internal/runner/retrace_test.go`

**Interfaces:**
- Produces: `func runs.Claim(root string, marker runs.Marker) (string, error)` — schreibt die Marke unter der nächsten freien Nummer und gibt die Nummer zurück.
- Produces: `runs.WriteMarker` gibt für eine schon vorhandene Datei einen Fehler zurück, für den `errors.Is(err, fs.ErrExist)` gilt.
- Produces: `flow.NewCatalog` gibt `predicate "<name>" is nil` zurück.
- Produces: `(*runner.Runner).Resume(ctx, nil)` mit `Replay: true` auf einem Journal mit offenem Tor gibt `this run waits at a gate; answer it with resume` zurück.

- [ ] **Step 1: Write the failing tests for the exclusive marker**

In `internal/runs/marker_test.go` die Importe um `"errors"` und `"io/fs"` ergänzen und ans Ende anhängen:

```go
// A marker is claimed, never overwritten: two runs that computed the same
// number must not share one journal, and the second finds out here.
func TestWriteMarkerRefusesAMarkerThatExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0001.flow")
	if err := runs.WriteMarker(path, marker()); err != nil {
		t.Fatal(err)
	}
	err := runs.WriteMarker(path, marker())
	if !errors.Is(err, fs.ErrExist) || !strings.Contains(err.Error(), "writing") {
		t.Fatalf("err = %v", err)
	}
}

// A run outside a repository has no baseline, and its marker carries neither
// baseline line. Wave 1 left this unwritten.
func TestAMarkerWithoutABaselineHasNoBaselineLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0001.flow")
	without := marker()
	without.Baseline = nil
	if err := runs.WriteMarker(path, without); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "spec_to_board\nruntime=\"go\"\ntopic=\"a b\"\nulflow_version=\"0.1.0\"\nzeta=\"x\\ny\"\n"
	if string(got) != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	back, err := runs.ReadMarker(path)
	if err != nil || back.Baseline != nil {
		t.Fatalf("baseline = %+v, err = %v", back.Baseline, err)
	}
}

func TestClaimNumbersRunsInOrder(t *testing.T) {
	root := t.TempDir()
	first, err := runs.Claim(root, marker())
	if err != nil || first != "0001" {
		t.Fatalf("first = %q, err = %v", first, err)
	}
	second, err := runs.Claim(root, marker())
	if err != nil || second != "0002" {
		t.Fatalf("second = %q, err = %v", second, err)
	}
	if _, err := os.Stat(runs.MarkerPath(root, "0002")); err != nil {
		t.Fatal(err)
	}
}

// A refusal that is not a taken number is not tried again.
func TestClaimPassesOnWhatWriteMarkerRefuses(t *testing.T) {
	nameless := marker()
	nameless.Flow = ""
	_, err := runs.Claim(t.TempDir(), nameless)
	if err == nil || !strings.Contains(err.Error(), "a marker needs the name of its flow") {
		t.Fatalf("err = %v", err)
	}
}
```

Create `internal/runs/claim_internal_test.go`:

```go
package runs

import (
	"strings"
	"testing"
)

// A number another run took in the meantime is computed again, not shared.
func TestClaimComputesAgainWhenTheNumberIsTaken(t *testing.T) {
	root := t.TempDir()
	if err := WriteMarker(MarkerPath(root, "0001"), Marker{Flow: "other"}); err != nil {
		t.Fatal(err)
	}
	offered := []string{"0001", "0002"}
	next := func(string) string {
		id := offered[0]
		offered = offered[1:]
		return id
	}
	id, err := claim(root, Marker{Flow: "mine"}, next)
	if err != nil || id != "0002" {
		t.Fatalf("id = %q, err = %v", id, err)
	}
	taken, err := ReadMarker(MarkerPath(root, "0001"))
	if err != nil || taken.Flow != "other" {
		t.Fatalf("the taken marker was overwritten: %+v, %v", taken, err)
	}
}

func TestClaimGivesUpAfterTenTakenNumbers(t *testing.T) {
	root := t.TempDir()
	if err := WriteMarker(MarkerPath(root, "0001"), Marker{Flow: "other"}); err != nil {
		t.Fatal(err)
	}
	asked := 0
	_, err := claim(root, Marker{Flow: "mine"}, func(string) string {
		asked++
		return "0001"
	})
	if err == nil || !strings.Contains(err.Error(), "no free run number") || asked != 10 {
		t.Fatalf("asked %d times, err = %v", asked, err)
	}
}
```

In `internal/runs/id_test.go` den Test `TestNextIDCountsPastTheHighestNumericJournal` ersetzen:

```go
// A counter over journals and markers, and never a clock. A marker counts
// because a run claims its number with its marker before its journal exists.
// Names that are not numbers, and a number no int holds, do not count.
func TestNextIDCountsPastTheHighestNumberARunFileCarries(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"0001.jsonl", "0009.jsonl", "0012.flow", "notes.jsonl", "12a.jsonl", ".jsonl", "99999999999999999999.jsonl"} {
		touch(t, filepath.Join(root, runs.Dir, name))
	}
	if got := runs.NextID(root); got != "0013" {
		t.Fatalf("got %s, want 0013", got)
	}
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test ./internal/runs/`
Expected: Build-Fehler `undefined: runs.Claim` und `undefined: claim`. Das ist das erste Rot. Welche Tests danach aus dem richtigen Grund rot sind, zeigt Step 4, sobald `claim.go` baut.

- [ ] **Step 3: Write `claim.go`**

```go
package runs

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

// claimAttempts bounds how often a number another run took in the meantime is
// computed again. Ten runs starting within one marker write of each other is
// not a project this has to serve.
const claimAttempts = 10

// Claim gives a new run its number by writing its marker, and returns the
// number.
//
// The marker is the claim: it is created exclusively, so of two runs that
// computed the same number only one gets it, and the other computes again. The
// journal comes after, and so never belongs to two runs.
func Claim(root string, marker Marker) (string, error) {
	return claim(root, marker, NextID)
}

func claim(root string, marker Marker, next func(string) string) (string, error) {
	for attempt := 0; attempt < claimAttempts; attempt++ {
		id := next(root)
		err := WriteMarker(MarkerPath(root, id), marker)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		return id, nil
	}
	return "", fmt.Errorf("no free run number under %s after %d attempts", filepath.Join(root, Dir), claimAttempts)
}
```

- [ ] **Step 4: Run again and read which tests are still red**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test ./internal/runs/`
Expected: FAIL in genau diesen:
- `TestWriteMarkerRefusesAMarkerThatExists` (die zweite Marke wird überschrieben, `err = <nil>`),
- `TestClaimNumbersRunsInOrder` (`second = "0001"`, weil `NextID` Marken nicht zählt),
- `TestNextIDCountsPastTheHighestNumberARunFileCarries` (`got 0010`),
- `TestClaimComputesAgainWhenTheNumberIsTaken` (`id = "0001", err = <nil>`: der alte `WriteMarker` überschreibt die fremde Marke ohne Fehler, und der Test fällt schon an seinem ersten `Fatalf`),
- `TestClaimGivesUpAfterTenTakenNumbers` (`asked 1 times`).

`TestAMarkerWithoutABaselineHasNoBaselineLines` und `TestClaimPassesOnWhatWriteMarkerRefuses` sind grün: Sie legen bestehendes Verhalten fest, das bisher ungeprüft war.

- [ ] **Step 5: Make `WriteMarker` exclusive and `NextID` count markers**

In `internal/runs/marker.go` das Ende von `WriteMarker` ab `dir := filepath.Dir(path)` ersetzen:

```go
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	// O_EXCL: a marker is claimed, never overwritten. Two runs that computed
	// the same number would otherwise share one journal without a word.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return writing(path, err)
	}
	_, err = file.WriteString(strings.Join(lines, "\n") + "\n")
	return writing(path, errors.Join(err, file.Close()))
}

// writing names the marker a write failed on, and is nil when nothing failed.
// One exit for the claim and the write alike, because a write into a file that
// was just created cannot be made to fail from a test.
func writing(path string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("writing %s: %w", path, err)
}
```

In `internal/runs/id.go` `NextID` samt Kommentar ersetzen:

```go
// NextID is one more than the highest number a journal or a marker carries,
// four digits. A counter and never a clock.
//
// cli.py's next_run_id counts journals alone. Markers count here because Claim
// takes a number with the marker before the journal exists, and a count over
// journals would hand the same number out again.
//
// A directory that cannot be read counts as one without runs, as Python's glob
// has it. On Windows even a file in its place reads as absent (measured on
// 2026-09-11), and a write into it fails loudly in WriteMarker.
func NextID(root string) string {
	entries, _ := os.ReadDir(filepath.Join(root, Dir))
	highest := 0
	for _, entry := range entries {
		stem, isRunFile := strings.CutSuffix(entry.Name(), ".jsonl")
		if !isRunFile {
			stem, isRunFile = strings.CutSuffix(entry.Name(), ".flow")
		}
		if !isRunFile || !digits(stem) {
			continue
		}
		number, err := strconv.Atoi(stem)
		if err != nil {
			continue
		}
		highest = max(highest, number)
	}
	return fmt.Sprintf("%04d", highest+1)
}
```

- [ ] **Step 6: Run the runs tests with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -cover ./internal/runs/`
Expected: `ok … coverage: 100.0% of statements`. `TestWriteMarkerRefuses` bleibt grün: der Fall „path is a directory" meldet jetzt über `writing` ebenfalls `writing`.

- [ ] **Step 7: Write the failing catalog test**

In `internal/flow/catalog_test.go` nach `catalog(...)` einfügen:

```go
// truthy is a predicate that holds for every state, for tests that need one
// registered and do not care what it says.
func truthy(flow.State, flow.Params) bool { return true }
```

In `TestCatalogListsWhatItHoldsSorted` `map[string]flow.Predicate{"b": nil, "a": nil}` durch `map[string]flow.Predicate{"b": truthy, "a": truthy}` ersetzen, in `TestPredicatesIsACopy` `map[string]flow.Predicate{"a": nil}` durch `map[string]flow.Predicate{"a": truthy}` und `made.Predicates()["b"] = nil` durch `made.Predicates()["b"] = truthy`. Ans Ende anhängen:

```go
// Found like any other, a nil predicate would panic inside the first condition
// that names it, far from whoever registered it.
func TestNewCatalogRefusesANilPredicate(t *testing.T) {
	_, err := flow.NewCatalog(nil, map[string]flow.Predicate{"green": truthy, "red": nil})
	if err == nil || err.Error() != `predicate "red" is nil` {
		t.Fatalf("err = %v", err)
	}
}
```

- [ ] **Step 8: Run it to see it fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -run TestNewCatalogRefusesANilPredicate ./internal/flow/`
Expected: FAIL mit `err = <nil>`.

- [ ] **Step 9: Refuse a nil predicate**

In `internal/flow/catalog.go` in `NewCatalog` die Schleife `for name, predicate := range predicates { … }` ersetzen:

```go
	// In name order, so that of two nil predicates the same one is named every
	// time.
	names := make([]string, 0, len(predicates))
	for name := range predicates {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if predicates[name] == nil {
			return nil, fmt.Errorf("predicate %q is nil", name)
		}
		made.predicates[name] = predicates[name]
	}
```

Den Kommentar über `NewCatalog` um einen Satz ergänzen: `A nil predicate is refused too: it would be found like any other and panic inside the first condition that names it.`

- [ ] **Step 10: Run the flow tests with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -cover ./internal/flow/`
Expected: `ok … coverage: 100.0% of statements`.

- [ ] **Step 11: Write the failing runner test**

In `internal/runner/retrace_test.go` nach `TestReplayRefusesARunThatWaitsAtAGate` einfügen:

```go
// Resume without an answer refuses a replay of a waiting run the way Run does,
// rather than reporting that the gate is not in the journal.
func TestReplayResumeRefusesARunThatWaitsAtAGate(t *testing.T) {
	log := &memJournal{}
	first := runner.New(runner.Options{
		Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(),
	})
	if _, err := first.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	again := runner.New(runner.Options{
		Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(), Replay: true,
	})
	_, err := again.Resume(context.Background(), nil)
	if err == nil || err.Error() != "this run waits at a gate; answer it with resume" {
		t.Fatalf("err = %v", err)
	}
}
```

- [ ] **Step 12: Run it to see it fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -run TestReplayResumeRefusesARunThatWaitsAtAGate ./internal/runner/`
Expected: FAIL mit `err = <nil>` (die Resume endet heute mit einem Result `node "ask" is not in the journal`).

- [ ] **Step 13: Refuse it in `Resume`**

In `internal/runner/runner.go` über `Run` eine Variable anlegen:

```go
// errWaitsAtAGate refuses a replay of a run that is still open. A replay
// reproduces a run that ended; one waiting at a gate has no ending to
// reproduce, and retracing it would stop at the pause and report it as the
// replay's own result.
var errWaitsAtAGate = errors.New("this run waits at a gate; answer it with resume")
```

In `Run` den Block `if gate != nil { … }` samt Kommentar durch `if gate != nil { return Result{}, errWaitsAtAGate }` ersetzen. In `Resume` ab `r.entries = entries` bis zum Ende der Funktion ersetzen:

```go
	r.entries = entries
	r.retracing = true
	r.appended = false
	gate, err := r.journal.Pending()
	if err != nil {
		return Result{}, err
	}
	if r.replay && gate != nil {
		return Result{}, errWaitsAtAGate
	}
	if answer == nil {
		return r.walk(ctx, r.initialState(), nil)
	}
	if gate == nil {
		// An answer with nothing to answer is a mistake worth reporting: a
		// silent run from the start would discard the answer and charge for
		// every node again, which is the worst reading of a user's "yes".
		return Result{}, errors.New("no gate is waiting for an answer")
	}
	return r.walk(ctx, r.initialState(), &pendingAnswer{key: gate.InputHash, text: *answer})
```

- [ ] **Step 14: Run the runner tests with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -cover ./internal/runner/`
Expected: `ok … coverage: 100.0% of statements`. Wird ein Fall aus `TestAJournalThatCannotBeReadEndsTheCall` jetzt anders erreicht, bleibt er grün: jede Resume liest `Pending` nun vor dem Gang.

- [ ] **Step 15: gofmt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && gofmt -l internal/runs internal/flow internal/runner`
Expected: keine Ausgabe.

- [ ] **Step 16: Commit**

Stagen: die zehn Dateien unter **Files**. Nachrichtendatei `.git-commit-msg.txt` im Worktree:

```
Claim a run number with its marker, and refuse two open doors

A marker is now created exclusively, and Claim computes the number again
when another run took it in the meantime. NextID counts markers as well
as journals, because the marker comes first. Wave 1 found that two runs
could otherwise be handed one number and append to one journal.

NewCatalog refuses a nil predicate, which would have been found like any
other and panicked in the first condition naming it. Resume under replay
refuses a run that waits at a gate, as Run already did, instead of
reporting that the gate is not in the journal.
```

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && git commit -F .git-commit-msg.txt`

---

### Task 1: Der Session-Start-Hook meldet Go-Läufe mit `ulflow`

**Files:**
- Modify: `cmd/guard/hook_session_start.go`
- Test: `cmd/guard/hook_session_start_test.go`

**Interfaces:**
- Consumes: `runs.ReadMarker(path string) (*runs.Marker, error)`, `runs.MarkerPath(root, id string) string`, `runs.Dir` aus Task 0 und Welle 1.
- Produces: die Zeile `answer it with: ulflow resume <id> --answer "your answer"` für Läufe mit `runtime="go"`.

- [ ] **Step 1: Write the failing tests**

In `cmd/guard/hook_session_start_test.go` den Import `"github.com/xidus90/ultra-loom/internal/runs"` ergänzen und nach `TestHookSessionStartReportsAPausedRun` einfügen:

```go
// A run ulflow started is carried on by ulflow. The marker's runtime line says
// which runtime started it; a marker without one -- every marker Python wrote --
// keeps the ultraloom command.
func TestHookSessionStartNamesTheRuntimeThatResumesARun(t *testing.T) {
	root := project(t)
	writeRun(t, root, "0001", waitingRun)
	writeRun(t, root, "0002", waitingRun)
	if err := runs.WriteMarker(runs.MarkerPath(root, "0001"), runs.Marker{Flow: "example", Runtime: "go", Version: "0.1.0"}); err != nil {
		t.Fatal(err)
	}
	if err := runs.WriteMarker(runs.MarkerPath(root, "0002"), runs.Marker{Flow: "verify_until_green"}); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runHookSessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude")

	if code != ExitOK || stderr.Len() != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr.String())
	}
	context := additionalContext(t, stdout.Bytes())
	if !strings.Contains(context, `answer it with: ulflow resume 0001 --answer "your answer"`) {
		t.Fatalf("a Go run is resumed by ulflow: %q", context)
	}
	if !strings.Contains(context, `answer it with: ultraloom resume 0002 --answer "your answer"`) {
		t.Fatalf("a Python run keeps ultraloom: %q", context)
	}
}

// A marker that cannot be read is named the way a damaged journal is, and the
// run is still announced -- with the command that has always owned runs.
func TestHookSessionStartNamesAMarkerItCannotReadAndCarriesOn(t *testing.T) {
	root := project(t)
	writeRun(t, root, "0001", waitingRun)
	marker := filepath.Join(root, ".ultraloom", "runs", "0001.flow")
	if err := os.WriteFile(marker, []byte("example\nno equals sign\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runHookSessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude")

	if code != ExitOK {
		t.Fatalf("a damaged marker does not end a session, got %d", code)
	}
	if !strings.Contains(stderr.String(), "0001.flow") {
		t.Fatalf("the damaged marker is named: %q", stderr.String())
	}
	if !strings.Contains(additionalContext(t, stdout.Bytes()), "ultraloom resume 0001") {
		t.Fatalf("the run is still announced: %q", stdout.String())
	}
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -run "TestHookSessionStartNames" ./cmd/guard/`
Expected: `TestHookSessionStartNamesTheRuntimeThatResumesARun` FAIL mit `a Go run is resumed by ulflow`. `TestHookSessionStartNamesAMarkerItCannotReadAndCarriesOn` FAIL mit `the damaged marker is named: ""`.

- [ ] **Step 3: Read the marker in the hook**

In `cmd/guard/hook_session_start.go` den Import `"github.com/xidus90/ultra-loom/internal/runs"` ergänzen, die Konstante `runDir` samt ihrem Kommentar löschen und in `waiting` `filepath.FromSlash(runDir)` durch `filepath.FromSlash(runs.Dir)` ersetzen. Die Zeile, die `lines` anhängt, ersetzen:

```go
		lines = append(lines, fmt.Sprintf(
			"run %s is waiting at %s: %s\n  answer it with: %s resume %s --answer \"your answer\"",
			runID, gate.Node, gate.Question, resumer(root, runID, stderr), runID))
```

Ans Ende der Datei:

```go
// resumer is the program that carries a run on: ulflow for the runs it started,
// which their marker says, and ultraloom for every other. A Python marker has
// no runtime line, and neither runtime resumes the other's runs.
//
// A marker that cannot be read is named the way a damaged journal is, and the
// run is still announced with the command that has always owned runs: hiding
// the question behind a damaged marker would leave a waiting run unannounced.
func resumer(root, runID string, stderr io.Writer) string {
	marker, err := runs.ReadMarker(runs.MarkerPath(root, runID))
	if err != nil {
		fmt.Fprintf(stderr, "ulguard hook session-start: %v\n", err)
		return "ultraloom"
	}
	if marker != nil && marker.Runtime == "go" {
		return "ulflow"
	}
	return "ultraloom"
}
```

- [ ] **Step 4: Run the guard tests with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -cover ./cmd/guard/`
Expected: `ok`, alle Tests grün. Die Coverage von `cmd/guard` war vor dem Task 99,1 %; sie darf nicht sinken, und `resumer` ist vollständig abgedeckt. Die fehlenden 0,9 % sind älter als dieser Plan und nicht sein Auftrag.

- [ ] **Step 5: gofmt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && gofmt -l cmd/guard`
Expected: keine Ausgabe.

- [ ] **Step 6: Commit**

```
Tell a waiting ulflow run to resume with ulflow

session-start told every waiting run to resume with ultraloom, a Go run
included, and wave 3's green criterion would have passed with the wrong
command. The hook now reads the run's marker: runtime="go" names ulflow,
and a marker without the line -- every one Python wrote -- keeps
ultraloom. A marker that cannot be read is named on stderr and the run is
still announced.

Linking internal/runs into ulguard was measured before: 3.6 KB and about
a millisecond at the median of 25 warm runs, inside the spread.
```

---

### Task 2: `flowload.Params` und der Beispiel-Flow

**Files:**
- Create: `internal/flowload/params.go`
- Test: `internal/flowload/params_test.go`
- Create: `internal/flowload/testdata/example/example.toml`, `internal/flowload/testdata/example/instructions/draft.md`, `internal/flowload/testdata/example/instructions/approve-question.md`
- Modify: `internal/flowload/blocks_test.go`

**Interfaces:**
- Consumes: `flow.Graph.Params map[string]flow.Field` mit Vorgaben im deklarierten Go-Typ (Welle 2).
- Produces: `func flowload.Params(graph *flow.Graph, options map[string]string) (flow.Params, error)`. Der Fehler ist `flowload.Findings`, eine Zeile je Option, nach Optionsname sortiert.
- Produces: `internal/flowload/testdata/example/` — der Flow, den `cmd/flow` in Task 3 und 4 in ein Testprojekt kopiert.

- [ ] **Step 1: Write the failing params tests**

Create `internal/flowload/params_test.go`:

```go
package flowload_test

import (
	"reflect"
	"testing"

	"github.com/xidus90/ultra-loom/internal/flow"
	"github.com/xidus90/ultra-loom/internal/flowload"
)

func declared() *flow.Graph {
	return &flow.Graph{Name: "example", Params: map[string]flow.Field{
		"rounds": {Type: flow.Int, Default: 5},
		"strict": {Type: flow.Bool, Default: false},
		"topic":  {Type: flow.String, Default: "tests"},
		"tags":   {Type: flow.StringList, Default: []string{}},
	}}
}

func TestParamsAreTheDefaultsWithoutOptions(t *testing.T) {
	got, err := flowload.Params(declared(), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := flow.Params{"rounds": 5, "strict": false, "topic": "tests", "tags": []string{}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// An option arrives as text, from the command line or from a marker, and
// leaves in the Go type its parameter declares.
func TestAnOptionTakesTheDeclaredType(t *testing.T) {
	got, err := flowload.Params(declared(), map[string]string{
		"rounds": "7", "strict": "true", "topic": "a b", "tags": `["x","y"]`,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := flow.Params{"rounds": 7, "strict": true, "topic": "a b", "tags": []string{"x", "y"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// Every option it cannot use, one line each and in name order, the way a load
// stage reports every finding it has.
func TestParamsRefuseEveryOptionTheyCannotRead(t *testing.T) {
	_, err := flowload.Params(declared(), map[string]string{
		"round": "7", "rounds": "seven", "strict": "yes", "tags": "x,y",
	})
	want := `option round is no parameter of flow "example"; known parameters: rounds, strict, tags, topic
option rounds: cannot read "seven" as int
option strict: cannot read "yes" as bool
option tags: cannot read "x,y" as list[string]`
	if err == nil || err.Error() != want {
		t.Fatalf("err =\n%v\nwant\n%s", err, want)
	}
}

func TestParamsOfAFlowWithoutParameters(t *testing.T) {
	_, err := flowload.Params(&flow.Graph{Name: "example"}, map[string]string{"x": "1"})
	want := `option x is no parameter of flow "example"; known parameters: none`
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v", err)
	}
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -run Params ./internal/flowload/`
Expected: Build-Fehler `undefined: flowload.Params`.

- [ ] **Step 3: Write `params.go`**

```go
package flowload

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"

	"github.com/xidus90/ultra-loom/internal/flow"
)

// Params are a run's parameters: every declared default, with the options a run
// was started with laid over them.
//
// An option is text wherever it comes from -- the command line, or a run's
// marker on resume -- so it is read here into the type its parameter declares.
// An int is a decimal number, a bool is true or false and nothing else, a
// string is the text itself, and a list is a JSON list of strings, the encoding
// a marker already writes its values in. Every option that is no parameter or
// does not read as its type is a finding, and all of them come back at once.
func Params(graph *flow.Graph, options map[string]string) (flow.Params, error) {
	params := make(flow.Params, len(graph.Params))
	for name, field := range graph.Params {
		params[name] = field.Default
	}
	names := make([]string, 0, len(options))
	for name := range options {
		names = append(names, name)
	}
	slices.Sort(names)
	var found Findings
	for _, name := range names {
		field, declared := graph.Params[name]
		if !declared {
			found = append(found, fmt.Sprintf("option %s is no parameter of flow %q; known parameters: %s",
				name, graph.Name, list(graph.Params)))
			continue
		}
		value, ok := readOption(field.Type, options[name])
		if !ok {
			found = append(found, fmt.Sprintf("option %s: cannot read %q as %s", name, options[name], field.Type))
			continue
		}
		params[name] = value
	}
	if len(found) > 0 {
		return nil, found
	}
	return params, nil
}

// readOption reads one option's text as the type its parameter declares.
func readOption(t flow.Type, text string) (flow.Value, bool) {
	switch t {
	case flow.Int:
		number, err := strconv.Atoi(text)
		return number, err == nil
	case flow.Bool:
		return text == "true", text == "true" || text == "false"
	case flow.StringList:
		var items []string
		err := json.Unmarshal([]byte(text), &items)
		return items, err == nil && items != nil
	default:
		return text, true
	}
}
```

`list` ist die vorhandene Funktion `func list[V any](held map[string]V) string` in `decl.go:348`: die Namen sortiert und mit `, ` verbunden, oder `none`.

`items != nil` lehnt `null` ab: `json.Unmarshal` nimmt `null` für eine Scheibe ohne Fehler an und ließe eine nil-Liste in einen Zustand, den `flow.Value` als `[]string` verspricht.

- [ ] **Step 4: Run the params tests**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -run Params ./internal/flowload/`
Expected: PASS.

- [ ] **Step 5: Pin `null` for a list**

`readOption` hat einen Zweig, den kein Test erreicht: eine Liste, die als `null` ankommt. In `TestParamsRefuseEveryOptionTheyCannotRead` wäre er ein fünftes Rot; als eigener Test lesbarer:

```go
// null is valid JSON and no list: read as one, it would put a nil into a
// state that promises a []string.
func TestParamsRefuseNullForAList(t *testing.T) {
	_, err := flowload.Params(declared(), map[string]string{"tags": "null"})
	want := `option tags: cannot read "null" as list[string]`
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v", err)
	}
}
```

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -run TestParamsRefuseNullForAList ./internal/flowload/`
Expected: PASS (die Bedingung `items != nil` steht schon; der Test legt sie fest). Zur Gegenprobe `&& items != nil` vorübergehend entfernen, den Test rot sehen (`err = <nil>`), zurücksetzen.

- [ ] **Step 6: Write the failing joining test**

`internal/flowload/blocks_test.go` vollständig ersetzen:

```go
package flowload_test

import (
	"testing"

	"github.com/xidus90/ultra-loom/internal/blocks"
	"github.com/xidus90/ultra-loom/internal/flow"
	"github.com/xidus90/ultra-loom/internal/flowcfg"
	"github.com/xidus90/ultra-loom/internal/flowload"
)

// realCatalog holds the blocks a real run uses, not the doubles the loader's
// own tests register.
func realCatalog(t *testing.T) *flow.Catalog {
	t.Helper()
	catalog, err := flow.NewCatalog([]flow.Block{blocks.Agent{}, blocks.Gate{}, blocks.Exit{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

// The lanes of wave 2 were built apart on purpose: the loader tested against
// blocks of its own, and the blocks were written without a loader. This is
// where they meet. The planning flow passes all seven stages against the blocks
// a real run uses.
func TestThePlanningFlowLoadsAgainstTheRealBlocks(t *testing.T) {
	graph, err := flowload.Load("testdata/planning/planning.toml", realCatalog(t), flowcfg.Config{})
	if err != nil {
		t.Fatalf("the planning flow does not load:\n%v", err)
	}
	if len(graph.Nodes) != 12 {
		t.Fatalf("%d nodes", len(graph.Nodes))
	}
}

// The planning flow has no exit node, and every other fixture's exit omits the
// code the real Exit requires. The example flow of the spec holds all three
// kinds, so exit meets the loader here. cmd/flow runs the same file for its
// golden journal.
func TestTheExampleFlowLoadsAgainstTheRealBlocks(t *testing.T) {
	graph, err := flowload.Load("testdata/example/example.toml", realCatalog(t), flowcfg.Config{})
	if err != nil {
		t.Fatalf("the example flow does not load:\n%v", err)
	}
	kinds := map[string]bool{}
	for _, node := range graph.Nodes {
		kinds[node.Kind] = true
	}
	if !kinds["agent"] || !kinds["gate"] || !kinds["exit"] {
		t.Fatalf("kinds = %v", kinds)
	}
}
```

- [ ] **Step 7: Run it to see it fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -run TestTheExampleFlowLoadsAgainstTheRealBlocks ./internal/flowload/`
Expected: FAIL mit `example.toml: open testdata/example/example.toml: …` (die Datei gibt es nicht).

- [ ] **Step 8: Write the example flow**

Create `internal/flowload/testdata/example/example.toml`:

```toml
schema_version = 1

[flow]
name  = "example"
start = "draft"

[params]
max_rounds = { type = "int", default = 5 }

[state]
verdict     = { type = "string", default = "" }
count       = { type = "int", default = 0 }
answer      = { type = "string", default = "" }
answer_text = { type = "string", default = "" }

[[node]]
name        = "draft"
kind        = "agent"
instruction = "instructions/draft.md"
effort      = "high"
tools       = "edit"
max_visits  = "max_rounds + 1"
reply       = { verdict = "string", count = "int" }

[[node]]
name     = "approve"
kind     = "gate"
question = "instructions/approve-question.md"
choices  = ["yes", "no"]
answer   = "answer"

[[node]]
name    = "stop"
kind    = "exit"
code    = 4
message = "rejected after {{count}} rounds"

[[edge]]
from = "draft"
to   = "approve"
when = "verdict == \"done\""

[[edge]]
from = "draft"
to   = "draft"

[[edge]]
from = "approve"
to   = "END"
when = "answer == \"yes\""

[[edge]]
from = "approve"
to   = "stop"

[[edge]]
from = "stop"
to   = "END"
```

Create `internal/flowload/testdata/example/instructions/draft.md` (eine Zeile, mit Zeilenumbruch am Ende):

```
Write a draft plan. You have {{max_rounds}} rounds.
```

Create `internal/flowload/testdata/example/instructions/approve-question.md`:

```
Approve the plan after {{count}} rounds?
```

`.gitattributes` pinnt `internal/flowload/testdata/**` schon auf LF.

- [ ] **Step 9: Run the flowload tests with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -cover ./internal/flowload/`
Expected: `ok … coverage: 100.0% of statements`. Scheitert der Verbindungstest mit Befunden, ist die Ursache ein Schlüssel oder eine Regel, die der Beispiel-Flow verletzt: den Befund lesen und die Testdatei korrigieren, nicht den Lader.

- [ ] **Step 10: gofmt and commit**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && gofmt -l internal/flowload`
Expected: keine Ausgabe.

```
Read a run's options into its parameters, and load the example flow

Params lays the options a run was started with over the declared
defaults: an int is a decimal number, a bool true or false, a string
itself, and a list a JSON list of strings -- the encoding the run marker
writes already. Every option that is no parameter or does not read as
its type comes back at once.

The example flow of the spec joins the loader's real-blocks test. The
planning flow has no exit node, so exit had never been loaded against
the real catalog; this flow holds all three kinds, and cmd/flow will run
it for the golden journal.
```

---

### Task 3: `cmd/flow`

**Files:**
- Create: `cmd/flow/main.go` — Paketdoku, `version`, Exit-Codes, `Deps`, `main`, `production`, `cli`, `arguments`, `refuse`
- Create: `cmd/flow/session.go` — `session`, `project`, `openFlow`, `needsBaseline`, `modelFor`, `walker`, `finish`
- Create: `cmd/flow/run.go` — `optionFlag`, `runCommand`, `takeBaseline`, `forgetUnstarted`
- Create: `cmd/flow/resume.go` — `resumeCommand`, `replayCommand`, `recorded`, `noRun`
- Create: `cmd/flow/show.go` — `showCommand`, `listCommand`
- Create: `cmd/flow/journal.go` — `fileJournal`
- Test: `cmd/flow/helpers_test.go`, `cmd/flow/main_test.go`, `cmd/flow/run_test.go`, `cmd/flow/resume_test.go`, `cmd/flow/show_test.go`
- Modify: `.gitignore` (`/ulflow`, `/ulflow.exe`, `/flow.exe`, `/ulflow_*.exe`)

**Interfaces:**
- Consumes: `runs.Claim`, `runs.ReadMarker`, `runs.JournalPath`, `runs.MarkerPath`, `runs.Dir` (Task 0); `flowload.Params`, `internal/flowload/testdata/example/` (Task 2); `flowload.Find`, `flowload.Load`, `flowload.List`, `flowcfg.Load`, `flowcfg.Config.Resolve`, `flow.NewCatalog`, `runner.New`, `journal.Entries`, `journal.Append`, `journal.Pending`, `gitwork.HeadCommit`, `gitwork.ChangedFiles` (Wellen 1 und 2).
- Produces: `func cli(args []string, deps Deps) int`, `type Deps struct { Stdout, Stderr io.Writer; Clock runner.Clock; Models func(provider string) (model.Model, error); Blocks []flow.Block }`, die Testhelfer `newHarness`, `done`, `pausedExample`, `exampleProject` — Task 4 benutzt sie.

- [ ] **Step 1: Write the test helpers**

Create `cmd/flow/helpers_test.go`:

```go
package main

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/ultra-loom/internal/blocks"
	"github.com/xidus90/ultra-loom/internal/flow"
	"github.com/xidus90/ultra-loom/internal/gitenv"
	"github.com/xidus90/ultra-loom/internal/model"
)

// harness is the outside a test hands to cli: both streams in buffers, a clock
// that moves half a second per call, a model that answers from a queue, and the
// real blocks.
type harness struct {
	deps   Deps
	fake   *model.Fake
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func newHarness(answers ...model.Answer) *harness {
	h := &harness{fake: model.NewFake(answers...), stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	now := time.Unix(0, 0)
	h.deps = Deps{
		Stdout: h.stdout,
		Stderr: h.stderr,
		Clock: func() time.Time {
			now = now.Add(500 * time.Millisecond)
			return now
		},
		Models: func(string) (model.Model, error) { return h.fake, nil },
		Blocks: []flow.Block{blocks.Agent{}, blocks.Gate{}, blocks.Exit{}},
	}
	return h
}

// run calls cli with --root appended, after emptying both buffers so that each
// call's output stands alone.
func (h *harness) run(root string, args ...string) int {
	h.stdout.Reset()
	h.stderr.Reset()
	return cli(append(args, "--root", root), h.deps)
}

// done is the one answer the example flow's draft needs to reach its gate.
func done() model.Answer {
	return model.Answer{Reply: model.Reply{Fields: map[string]any{"verdict": "done", "count": 2}, Tokens: 120}}
}

// exampleProject is a project holding the example flow of the spec, copied from
// the fixture internal/flowload loads against the real blocks.
func exampleProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join("..", "..", "internal", "flowload", "testdata", "example")
	target := filepath.Join(root, ".ultraloom", "flows")
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(filepath.Join(target, relative), 0o755)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(target, relative), body, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// pausedExample is an example project whose first run waits at its gate.
func pausedExample(t *testing.T, h *harness) string {
	t.Helper()
	root := exampleProject(t)
	if code := h.run(root, "run", "example"); code != exitPaused {
		t.Fatalf("the run did not pause: exit %d, stderr %q", code, h.stderr.String())
	}
	return root
}

// writeFile writes one file below root, its directories included.
func writeFile(t *testing.T, root, relative, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func overwrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func remove(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
}

// lines are a file's lines, without the empty one after the last newline.
func lines(t *testing.T, path string) []string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
}

// runFiles are the names under a project's runs directory; none when it is
// not there.
func runFiles(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, ".ultraloom", "runs"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// gitInit makes root a repository with one commit, so that a run there has a
// baseline. gitenv keeps a hook's GIT_DIR from pointing the commands elsewhere.
func gitInit(t *testing.T, root string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "t@example.invalid"},
		{"config", "user.name", "Test"},
		{"commit", "--allow-empty", "-m", "first"},
	} {
		command := exec.Command("git", args...)
		command.Dir = root
		command.Env = gitenv.Environ()
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// measureBlock is a node kind that needs the run's baseline and does nothing
// else, so that the refusal for a missing one has a flow to refuse.
type measureBlock struct{}

func (measureBlock) Kind() string                          { return "measure" }
func (measureBlock) Check(flow.Node) []string              { return nil }
func (measureBlock) Texts(flow.Node) []flow.Text           { return nil }
func (measureBlock) Writes(flow.Node) map[string]flow.Type { return nil }
func (measureBlock) NeedsBaseline() bool                   { return true }

func (measureBlock) Define(*flow.Graph, flow.Node, flow.Env) (flow.Definition, error) {
	return flow.Definition{}, nil
}

func (measureBlock) Run(context.Context, *flow.Graph, flow.Node, flow.State, flow.Env) (flow.Result, error) {
	return flow.Result{}, nil
}

// flakyBlock cannot be defined for a node named b. A run reaches b after a has
// taken its step, so the call fails with a journal already on disk.
type flakyBlock struct{ measureBlock }

func (flakyBlock) Kind() string        { return "flaky" }
func (flakyBlock) NeedsBaseline() bool { return false }

func (flakyBlock) Define(_ *flow.Graph, node flow.Node, _ flow.Env) (flow.Definition, error) {
	if node.Name == "b" {
		return flow.Definition{}, errors.New("the definition of b is gone")
	}
	return flow.Definition{}, nil
}

const measureFlow = `schema_version = 1

[flow]
name  = "measure"
start = "look"

[[node]]
name = "look"
kind = "measure"

[[edge]]
from = "look"
to   = "END"
`

const flakyFlow = `schema_version = 1

[flow]
name  = "flaky"
start = "a"

[[node]]
name = "a"
kind = "flaky"

[[node]]
name = "b"
kind = "flaky"

[[edge]]
from = "a"
to   = "b"

[[edge]]
from = "b"
to   = "END"
`

// askFlow is a gate and nothing else: a flow the real outside runs, since it
// asks no model.
const askFlow = `schema_version = 1

[flow]
name  = "ask"
start = "confirm"

[state]
answer      = { type = "string", default = "" }
answer_text = { type = "string", default = "" }

[[node]]
name     = "confirm"
kind     = "gate"
question = "questions/confirm.md"
choices  = ["yes", "no"]
answer   = "answer"

[[edge]]
from = "confirm"
to   = "END"
`

// pairFlow asks two providers, which M1 refuses.
const pairFlow = `schema_version = 1

[flow]
name  = "pair"
start = "one"

[state]
verdict = { type = "string", default = "" }

[[node]]
name        = "one"
kind        = "agent"
model       = "writer"
instruction = "instructions/one.md"
reply       = { verdict = "string" }

[[node]]
name        = "two"
kind        = "agent"
model       = "critic"
instruction = "instructions/one.md"
reply       = { verdict = "string" }

[[edge]]
from = "one"
to   = "two"

[[edge]]
from = "two"
to   = "END"
`

const pairConfig = `[agent.models.writer]
provider = "claude"

[agent.models.critic]
provider = "gemini"
`
```

- [ ] **Step 2: Write the failing command-line tests**

Create `cmd/flow/main_test.go`:

```go
package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	h := newHarness()
	if code := cli([]string{"--version"}, h.deps); code != exitOK || h.stdout.String() != "0.1.0\n" {
		t.Fatalf("exit %d, stdout %q", code, h.stdout.String())
	}
}

func TestUsageErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		code int
		want string
	}{
		{"no command", nil, exitUsage, "usage:"},
		{"an unknown command", []string{"fly"}, exitUsage, `unknown command "fly"`},
		{"run without a flow", []string{"run"}, exitUsage, "ulflow run: the name is missing"},
		{"a flag where the run belongs", []string{"resume", "--answer", "yes"}, exitUsage, "ulflow resume: the name is missing"},
		{"an argument too many", []string{"show", "0001", "0002"}, exitUsage, `ulflow show: unexpected argument "0002"`},
		{"an option without a value", []string{"run", "example", "--option", "max_rounds"}, exitUsage, `want name=value, got "max_rounds"`},
		{"an option given twice", []string{"run", "example", "--option", "a=1", "--option", "a=2"}, exitUsage, "option a is given twice"},
		{"an unknown flag", []string{"replay", "0001", "--nope"}, exitUsage, "flag provided but not defined: -nope"},
		{"asking for help", []string{"list", "-h"}, exitOK, "-root"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness()
			code := cli(c.args, h.deps)
			if code != c.code || !strings.Contains(h.stderr.String(), c.want) {
				t.Fatalf("exit %d, stderr %q, want %d and %q", code, h.stderr.String(), c.code, c.want)
			}
		})
	}
}

// The real outside refuses every provider until M2 brings the adapters, keeps
// time with the wall clock, and knows the three blocks M1 builds.
func TestProductionHasNoAdapterYet(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := production(&stdout, &stderr)
	if _, err := deps.Models("claude"); err == nil || err.Error() != "no adapter for provider claude yet" {
		t.Fatalf("err = %v", err)
	}
	if deps.Clock == nil || deps.Stdout != &stdout || deps.Stderr != &stderr {
		t.Fatalf("deps = %+v", deps)
	}
	kinds := make([]string, 0, len(deps.Blocks))
	for _, block := range deps.Blocks {
		kinds = append(kinds, block.Kind())
	}
	if !slices.Equal(kinds, []string{"agent", "gate", "exit"}) {
		t.Fatalf("kinds = %v", kinds)
	}
}
```

Create `cmd/flow/run_test.go`:

```go
package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/ultra-loom/internal/blocks"
	"github.com/xidus90/ultra-loom/internal/runs"
)

func TestRunPausesAtTheGateAndRecordsTheRun(t *testing.T) {
	root := exampleProject(t)
	h := newHarness(done())
	if code := h.run(root, "run", "example"); code != exitPaused {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got, want := h.stdout.String(), "run 0001: paused\nApprove the plan after 2 rounds?\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := lines(t, runs.JournalPath(root, "0001")); len(got) != 2 {
		t.Fatalf("journal = %q", got)
	}
	marker, err := runs.ReadMarker(runs.MarkerPath(root, "0001"))
	if err != nil {
		t.Fatal(err)
	}
	if marker.Flow != "example" || marker.Runtime != "go" || marker.Version != version || len(marker.Options) != 0 {
		t.Fatalf("marker = %+v", marker)
	}
	if marker.Baseline != nil {
		t.Fatalf("a temporary directory is no repository, and a run there has no baseline: %+v", marker.Baseline)
	}
}

// An option reaches the run in its declared type and the marker as text.
func TestRunTakesAnOptionIntoTheRunAndTheMarker(t *testing.T) {
	root := exampleProject(t)
	h := newHarness(done())
	if code := h.run(root, "run", "example", "--option", "max_rounds=3"); code != exitPaused {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	seen := h.fake.Seen()
	if len(seen) != 1 || !strings.Contains(seen[0].Prompt, "You have 3 rounds.") {
		t.Fatalf("requests = %+v", seen)
	}
	marker, err := runs.ReadMarker(runs.MarkerPath(root, "0001"))
	if err != nil || marker.Options["max_rounds"] != "3" {
		t.Fatalf("marker = %+v, err = %v", marker, err)
	}
}

// Inside a repository a run takes the commit it starts from, and the marker
// carries it.
func TestRunInARepositoryRecordsItsBaseline(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	writeFile(t, root, ".ultraloom/flows/measure.toml", measureFlow)
	h := newHarness()
	h.deps.Blocks = append(h.deps.Blocks, measureBlock{})
	if code := h.run(root, "run", "measure"); code != exitOK {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	marker, err := runs.ReadMarker(runs.MarkerPath(root, "0001"))
	if err != nil || marker.Baseline == nil || len(marker.Baseline.Commit) != 40 {
		t.Fatalf("marker = %+v, err = %v", marker, err)
	}
}

// A flow without an agent node needs no model, so the real outside -- which has
// no adapter -- runs it.
func TestRunOfAFlowWithoutAgentsAsksForNoModel(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".ultraloom/flows/ask.toml", askFlow)
	writeFile(t, root, ".ultraloom/flows/questions/confirm.md", "Ship it?\n")
	h := newHarness()
	h.deps.Models = production(h.stdout, h.stderr).Models
	if code := h.run(root, "run", "ask"); code != exitPaused {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got, want := h.stdout.String(), "run 0001: paused\nShip it?\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

// Every refusal before a run begins leaves nothing under the runs directory:
// no journal, and no marker holding a number for a run that never took a step.
func TestRunRefusesBeforeTheRunExists(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		arrange func(t *testing.T, root string, h *harness)
		want    string
	}{
		{name: "a flow that is not there", args: []string{"run", "nope"}, want: `no flow named "nope"`},
		{name: "a flow that does not load", args: []string{"run", "broken"},
			arrange: func(t *testing.T, root string, _ *harness) {
				writeFile(t, root, ".ultraloom/flows/broken.toml", "schema_version = 7\n")
			},
			want: "schema_version 7 is unknown"},
		{name: "an option that is no parameter", args: []string{"run", "example", "--option", "rounds=3"},
			want: `option rounds is no parameter of flow "example"`},
		{name: "an option of the wrong type", args: []string{"run", "example", "--option", "max_rounds=five"},
			want: `option max_rounds: cannot read "five" as int`},
		{name: "a provider without an adapter", args: []string{"run", "example"},
			arrange: func(_ *testing.T, _ string, h *harness) { h.deps.Models = production(h.stdout, h.stderr).Models },
			want:    "no adapter for provider claude yet"},
		{name: "two providers in one flow", args: []string{"run", "pair"},
			arrange: func(t *testing.T, root string, _ *harness) {
				writeFile(t, root, ".ultraloom/flows/pair.toml", pairFlow)
				writeFile(t, root, ".ultraloom/flows/instructions/one.md", "Judge the plan.\n")
				writeFile(t, root, ".ultraloom/config.toml", pairConfig)
			},
			want: `flow "pair" asks claude and gemini; M1 runs one provider per flow`},
		{name: "a damaged config", args: []string{"run", "example"},
			arrange: func(t *testing.T, root string, _ *harness) {
				writeFile(t, root, ".ultraloom/config.toml", "[agent]\ndefault = 3\n")
			},
			want: "[agent].default must be a non-empty string"},
		{name: "two blocks of one kind", args: []string{"run", "example"},
			arrange: func(_ *testing.T, _ string, h *harness) { h.deps.Blocks = append(h.deps.Blocks, blocks.Gate{}) },
			want:    `two blocks claim kind "gate"`},
		{name: "a flow that needs a baseline outside a repository", args: []string{"run", "measure"},
			arrange: func(t *testing.T, root string, h *harness) {
				writeFile(t, root, ".ultraloom/flows/measure.toml", measureFlow)
				h.deps.Blocks = append(h.deps.Blocks, measureBlock{})
			},
			want: "measure measures against the commit it starts from"},
		{name: "a ceiling an option makes zero", args: []string{"run", "example", "--option", "max_rounds=-1"},
			want: `node "draft" allows 0 visits`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := exampleProject(t)
			h := newHarness(done())
			if c.arrange != nil {
				c.arrange(t, root, h)
			}
			code := h.run(root, c.args...)
			if code != exitFailed || !strings.Contains(h.stderr.String(), c.want) {
				t.Fatalf("exit %d, stderr %q, want %q", code, h.stderr.String(), c.want)
			}
			if files := runFiles(t, root); len(files) != 0 {
				t.Fatalf("a refused run left %v behind", files)
			}
		})
	}
}

// A run that took a step before its call failed keeps its marker: the journal
// is a record of what happened, and the marker says which flow it belongs to.
func TestARunThatTookAStepKeepsItsMarker(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".ultraloom/flows/flaky.toml", flakyFlow)
	h := newHarness()
	h.deps.Blocks = append(h.deps.Blocks, flakyBlock{})
	code := h.run(root, "run", "flaky")
	if code != exitFailed || !strings.Contains(h.stderr.String(), "the definition of b is gone") {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if files := runFiles(t, root); !slices.Equal(files, []string{"0001.flow", "0001.jsonl"}) {
		t.Fatalf("files = %v", files)
	}
}

// A file where the runs directory belongs: the claim fails, and says where.
func TestRunRefusesWhenItCannotClaimANumber(t *testing.T) {
	root := exampleProject(t)
	writeFile(t, root, ".ultraloom/runs", "a file where the runs directory belongs")
	h := newHarness(done())
	if code := h.run(root, "run", "example"); code != exitFailed || !strings.Contains(h.stderr.String(), "creating") {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
}
```

Create `cmd/flow/resume_test.go`:

```go
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/ultra-loom/internal/runs"
)

func TestResumeWithARejectionEndsAtTheExit(t *testing.T) {
	h := newHarness(done())
	root := pausedExample(t, h)
	if code := h.run(root, "resume", "0001", "--answer", "no: too thin"); code != 4 {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got, want := h.stdout.String(), "run 0001: error\nrejected after 2 rounds\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := lines(t, runs.JournalPath(root, "0001")); len(got) != 4 {
		t.Fatalf("journal = %q", got)
	}
	if seen := h.fake.Seen(); len(seen) != 1 {
		t.Fatalf("the resume asked the model again: %d requests", len(seen))
	}
}

func TestResumeWithApprovalIsDone(t *testing.T) {
	h := newHarness(done())
	root := pausedExample(t, h)
	if code := h.run(root, "resume", "0001", "--answer", "yes"); code != exitOK {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got := h.stdout.String(); got != "run 0001: done\n" {
		t.Fatalf("stdout = %q", got)
	}
}

// Without --answer the gate asks again and nothing is written: a run somebody
// looks at is not a run that moved.
func TestResumeWithoutAnAnswerAsksAgain(t *testing.T) {
	h := newHarness(done())
	root := pausedExample(t, h)
	if code := h.run(root, "resume", "0001"); code != exitPaused {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got, want := h.stdout.String(), "run 0001: paused\nApprove the plan after 2 rounds?\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := lines(t, runs.JournalPath(root, "0001")); len(got) != 2 {
		t.Fatalf("journal = %q", got)
	}
}

// An answer no choice matches is refused, names the choices, and leaves the
// gate open for the next one.
func TestResumeRefusesAnAnswerNoChoiceMatches(t *testing.T) {
	h := newHarness(done())
	root := pausedExample(t, h)
	if code := h.run(root, "resume", "0001", "--answer", "maybe"); code != exitFailed || !strings.Contains(h.stderr.String(), "the choices are yes, no") {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if code := h.run(root, "resume", "0001", "--answer", "yes"); code != exitOK {
		t.Fatalf("the gate did not stay open: exit %d, stderr %q", code, h.stderr.String())
	}
}

func TestResumeAndReplayRefuse(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		arrange func(t *testing.T, root string, h *harness)
		want    string
	}{
		{name: "a run that is not there", args: []string{"resume", "0009"}, want: `no run "0009" under`},
		{name: "a replay of a run that is not there", args: []string{"replay", "0009"}, want: `no run "0009" under`},
		{name: "a run without its marker", args: []string{"resume", "0001"},
			arrange: func(t *testing.T, root string, _ *harness) { remove(t, runs.MarkerPath(root, "0001")) },
			want:    `run "0001" does not say which flow it belongs to`},
		{name: "a damaged marker", args: []string{"resume", "0001"},
			arrange: func(t *testing.T, root string, _ *harness) {
				overwrite(t, runs.MarkerPath(root, "0001"), "example\nno equals sign\n")
			},
			want: "option line without '='"},
		{name: "a run of the Python runtime", args: []string{"resume", "0001"},
			arrange: func(t *testing.T, root string, _ *harness) { overwrite(t, runs.MarkerPath(root, "0001"), "example\n") },
			want:    "run 0001 was started by the Python runtime"},
		{name: "a damaged journal", args: []string{"resume", "0001"},
			arrange: func(t *testing.T, root string, _ *harness) { overwrite(t, runs.JournalPath(root, "0001"), "{not json\n") },
			want:    "line 1 is not a journal entry"},
		{name: "a flow that is gone", args: []string{"resume", "0001"},
			arrange: func(t *testing.T, root string, _ *harness) {
				remove(t, filepath.Join(root, ".ultraloom", "flows", "example.toml"))
			},
			want: `no flow named "example"`},
		{name: "a provider without an adapter", args: []string{"resume", "0001", "--answer", "yes"},
			arrange: func(_ *testing.T, _ string, h *harness) { h.deps.Models = production(h.stdout, h.stderr).Models },
			want:    "no adapter for provider claude yet"},
		{name: "a run that waits at no gate", args: []string{"resume", "0001"},
			arrange: func(t *testing.T, root string, h *harness) {
				if code := h.run(root, "resume", "0001", "--answer", "yes"); code != exitOK {
					t.Fatalf("exit %d", code)
				}
			},
			want: "run 0001 is not waiting at a gate; there is nothing to answer"},
		{name: "a replay of a run that waits at a gate", args: []string{"replay", "0001"},
			want: `run 0001 never finished: it is waiting at gate "approve"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness(done())
			root := pausedExample(t, h)
			if c.arrange != nil {
				c.arrange(t, root, h)
			}
			code := h.run(root, c.args...)
			if code != exitFailed || !strings.Contains(h.stderr.String(), c.want) {
				t.Fatalf("exit %d, stderr %q, want %q", code, h.stderr.String(), c.want)
			}
		})
	}
}

// A run whose flow measures against a commit, started where there was none,
// cannot be carried on: taking a baseline now would measure against the tree
// the run has meanwhile edited.
func TestARunWithoutTheBaselineItsFlowNeedsIsNotCarriedOn(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".ultraloom/flows/measure.toml", measureFlow)
	if err := runs.WriteMarker(runs.MarkerPath(root, "0001"), runs.Marker{Flow: "measure", Runtime: "go", Version: version}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ".ultraloom/runs/0001.jsonl", "")
	h := newHarness()
	h.deps.Blocks = append(h.deps.Blocks, measureBlock{})
	if code := h.run(root, "replay", "0001"); code != exitFailed || !strings.Contains(h.stderr.String(), "run 0001 was started outside a repository") {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
}

// A replay reproduces how the run ended, runs no node and writes nothing. The
// journal carries no exit code, so a replayed exit is a failure without one.
func TestReplayReproducesTheEndingAndWritesNothing(t *testing.T) {
	h := newHarness(done())
	root := pausedExample(t, h)
	if code := h.run(root, "resume", "0001", "--answer", "no: too thin"); code != 4 {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	before, err := os.ReadFile(runs.JournalPath(root, "0001"))
	if err != nil {
		t.Fatal(err)
	}
	if code := h.run(root, "replay", "0001"); code != exitFailed {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got, want := h.stdout.String(), "run 0001: error\nrejected after 2 rounds\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	after, err := os.ReadFile(runs.JournalPath(root, "0001"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("the replay wrote to the journal: %v", err)
	}
	if seen := h.fake.Seen(); len(seen) != 1 {
		t.Fatalf("the replay asked the model: %d requests", len(seen))
	}
}

// An instruction improved since the run is reported, never refused.
func TestReplayWarnsAboutAChangedInstruction(t *testing.T) {
	h := newHarness(done())
	root := pausedExample(t, h)
	if code := h.run(root, "resume", "0001", "--answer", "yes"); code != exitOK {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	overwrite(t, filepath.Join(root, ".ultraloom", "flows", "instructions", "draft.md"),
		"Write a better draft plan. You have {{max_rounds}} rounds.\n")
	code := h.run(root, "replay", "0001")
	if code != exitOK || !strings.Contains(h.stderr.String(), `warning: node "draft" ran on a definition that has changed since`) {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
}
```

Create `cmd/flow/show_test.go`:

```go
package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/ultra-loom/internal/runs"
)

func TestShowPrintsOneLinePerEntry(t *testing.T) {
	h := newHarness(done())
	root := pausedExample(t, h)
	if code := h.run(root, "resume", "0001", "--answer", "no: too thin"); code != 4 {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if code := h.run(root, "show", "0001"); code != exitOK {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	got := strings.Split(strings.TrimSuffix(h.stdout.String(), "\n"), "\n")
	want := [][]string{
		{"draft", "agent", "ok", "120", "tok", "0.50s", "edit"},
		{"approve", "gate", "paused", "0", "tok", "0.50s", "-"},
		{"approve", "gate", "ok", "0", "tok", "0.00s", "-"},
		{"stop", "exit", "error", "0", "tok", "0.50s", "-"},
	}
	if len(got) != len(want) {
		t.Fatalf("lines = %q", got)
	}
	for i := range want {
		if fields := strings.Fields(got[i]); !slices.Equal(fields, want[i]) {
			t.Fatalf("line %d = %q, want %v", i+1, got[i], want[i])
		}
	}
	// The columns are fixed, as cli.py printed them: node in 24, kind in 6.
	if strings.Index(got[0], "agent") != 25 || strings.Index(got[0], "ok") != 32 {
		t.Fatalf("columns moved: %q", got[0])
	}
}

func TestShowRefuses(t *testing.T) {
	h := newHarness(done())
	root := pausedExample(t, h)
	if code := h.run(root, "show", "0009"); code != exitFailed || !strings.Contains(h.stderr.String(), `no run "0009" under`) {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	overwrite(t, runs.JournalPath(root, "0001"), "{not json\n")
	if code := h.run(root, "show", "0001"); code != exitFailed || !strings.Contains(h.stderr.String(), "line 1 is not a journal entry") {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
}

func TestListNamesEveryFlowAndWhyOneWillNotLoad(t *testing.T) {
	root := exampleProject(t)
	writeFile(t, root, ".ultraloom/flows/broken.toml", "schema_version = 7\n")
	h := newHarness()
	if code := h.run(root, "list"); code != exitOK {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	got := strings.Split(strings.TrimSuffix(h.stdout.String(), "\n"), "\n")
	if len(got) < 3 {
		t.Fatalf("lines = %q", got)
	}
	if first := strings.Fields(got[0]); len(first) < 3 || !slices.Equal(first[:3], []string{"broken", "project", "broken.toml:"}) ||
		!strings.Contains(got[0], "schema_version 7 is unknown") {
		t.Fatalf("first line = %q", got[0])
	}
	if !strings.HasPrefix(got[1], "    broken.toml: ") {
		t.Fatalf("a further finding is indented: %q", got[1])
	}
	if last := strings.Fields(got[len(got)-1]); !slices.Equal(last, []string{"example", "project", "ok"}) {
		t.Fatalf("last line = %q", got[len(got)-1])
	}
}

func TestListRefusesAProjectItCannotRead(t *testing.T) {
	root := exampleProject(t)
	writeFile(t, root, ".ultraloom/config.toml", "[agent]\ndefault = 3\n")
	h := newHarness()
	if code := h.run(root, "list"); code != exitFailed || !strings.Contains(h.stderr.String(), "[agent].default must be a non-empty string") {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
}
```

- [ ] **Step 3: Run the tests to see them fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test ./cmd/flow/`
Expected: Build-Fehler (`undefined: cli`, `undefined: Deps`, `undefined: exitPaused`, …). Das Paket gibt es noch nicht.

- [ ] **Step 4: Write `main.go`**

```go
// Command ulflow runs flows: it loads a graph, walks it, journals every step,
// stops at gates, and carries a paused run on or replays a finished one.
//
// Everything a run decides lives under internal/; this package is the edge:
// arguments in, text and an exit code out.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/xidus90/ultra-loom/internal/blocks"
	"github.com/xidus90/ultra-loom/internal/flow"
	"github.com/xidus90/ultra-loom/internal/model"
	"github.com/xidus90/ultra-loom/internal/runner"
)

// version is what a run's marker records as ulflow_version.
const version = "0.1.0"

// The exit codes of the behaviour contract. 2 is a usage error, as the flag
// package and argparse both answer one; a paused run is 3 so that a script can
// tell "waiting at a gate" from "typed it wrong".
const (
	exitOK     = 0
	exitFailed = 1
	exitUsage  = 2
	exitPaused = 3
)

const usage = `usage:
  ulflow run <flow> [--option name=value]... [--root dir]
  ulflow resume <run> [--answer text] [--root dir]
  ulflow replay <run> [--root dir]
  ulflow show <run> [--root dir]
  ulflow list [--root dir]
  ulflow --version`

// Deps are what a run takes from outside its arguments. Tests hand in a clock
// that repeats itself, a model that answers from a queue and blocks of their
// own; main hands in the real ones.
type Deps struct {
	Stdout io.Writer
	Stderr io.Writer
	Clock  runner.Clock
	// Models finds the model for a provider, before a run writes anything.
	Models func(provider string) (model.Model, error)
	Blocks []flow.Block
}

// Uncovered on purpose, and the one statement that may be: os.Exit ends the
// test binary along with everything else. production and cli are the same
// program without it, and both are covered.
func main() {
	os.Exit(cli(os.Args[1:], production(os.Stdout, os.Stderr)))
}

// production is the real outside: the wall clock, the three blocks M1 builds,
// and no adapter for any provider. Those arrive with M2; until then a flow that
// asks a model is refused before its run exists.
func production(stdout, stderr io.Writer) Deps {
	return Deps{
		Stdout: stdout,
		Stderr: stderr,
		Clock:  time.Now,
		Models: func(provider string) (model.Model, error) {
			return nil, fmt.Errorf("no adapter for provider %s yet", provider)
		},
		Blocks: []flow.Block{blocks.Agent{}, blocks.Gate{}, blocks.Exit{}},
	}
}

// cli is main without the process, so the tests can drive it.
func cli(args []string, deps Deps) int {
	if len(args) == 0 {
		fmt.Fprintln(deps.Stderr, usage)
		return exitUsage
	}
	switch args[0] {
	case "--version":
		fmt.Fprintln(deps.Stdout, version)
		return exitOK
	case "run":
		return runCommand(args[1:], deps)
	case "resume":
		return resumeCommand(args[1:], deps)
	case "replay":
		return replayCommand(args[1:], deps)
	case "show":
		return showCommand(args[1:], deps)
	case "list":
		return listCommand(args[1:], deps)
	default:
		fmt.Fprintf(deps.Stderr, "ulflow: unknown command %q\n%s\n", args[0], usage)
		return exitUsage
	}
}

// arguments reads a command's name and its flags. flag stops at the first
// argument that is no flag, so the name comes first and the flags after it;
// anything left over is a mistake and gets the usage.
func arguments(command string, args []string, takesName bool, flags *flag.FlagSet, stderr io.Writer) (string, int, bool) {
	flags.SetOutput(stderr)
	name := ""
	if takesName {
		if len(args) == 0 || strings.HasPrefix(args[0], "-") {
			fmt.Fprintf(stderr, "ulflow %s: the name is missing\n%s\n", command, usage)
			return "", exitUsage, false
		}
		name, args = args[0], args[1:]
	}
	if err := flags.Parse(args); err != nil {
		// Asking for help is not a failure: an exit code of 2 for -h makes
		// every wrapper script think the tool broke.
		if errors.Is(err, flag.ErrHelp) {
			return "", exitOK, false
		}
		return "", exitUsage, false
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "ulflow %s: unexpected argument %q\n%s\n", command, flags.Arg(0), usage)
		return "", exitUsage, false
	}
	return name, exitOK, true
}

// refuse says why on stderr. A refusal is exit 1, like a failed run: the
// caller's next step is the same, reading the reason.
func refuse(deps Deps, err error) int {
	fmt.Fprintln(deps.Stderr, err)
	return exitFailed
}
```

- [ ] **Step 5: Write `session.go`**

```go
package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/xidus90/ultra-loom/internal/flow"
	"github.com/xidus90/ultra-loom/internal/flowcfg"
	"github.com/xidus90/ultra-loom/internal/flowload"
	"github.com/xidus90/ultra-loom/internal/model"
	"github.com/xidus90/ultra-loom/internal/runner"
	"github.com/xidus90/ultra-loom/internal/runs"
)

// session is one run as the runner needs it: where it lives, the graph it walks
// and what it was started with.
type session struct {
	root     string
	id       string
	graph    *flow.Graph
	catalog  *flow.Catalog
	agent    flowcfg.Config
	params   flow.Params
	baseline *flow.Baseline
	model    model.Model
}

// project is what every command reads before a flow: the [agent] table, and the
// blocks this build knows.
func project(root string, deps Deps) (flowcfg.Config, *flow.Catalog, error) {
	agent, err := flowcfg.Load(root)
	if err != nil {
		return flowcfg.Config{}, nil, err
	}
	catalog, err := flow.NewCatalog(deps.Blocks, nil)
	if err != nil {
		return flowcfg.Config{}, nil, err
	}
	return agent, catalog, nil
}

// openFlow finds a flow by name, loads it through all seven stages, and lays
// the options a run was started with over its parameters.
func openFlow(root, name string, options map[string]string, deps Deps) (session, error) {
	agent, catalog, err := project(root, deps)
	if err != nil {
		return session{}, err
	}
	path, err := flowload.Find(name, root)
	if err != nil {
		return session{}, err
	}
	graph, err := flowload.Load(path, catalog, agent)
	if err != nil {
		return session{}, err
	}
	params, err := flowload.Params(graph, options)
	if err != nil {
		return session{}, err
	}
	return session{root: root, graph: graph, catalog: catalog, agent: agent, params: params}, nil
}

// needsBaseline says whether a node of the flow measures against the commit a
// run starts from.
func (s session) needsBaseline() bool {
	for _, node := range s.graph.Nodes {
		// Stage 2 of the load has found a block for every kind.
		block, _ := s.catalog.Block(node.Kind)
		if block.NeedsBaseline() {
			return true
		}
	}
	return false
}

// modelFor finds the model the flow's agent nodes ask, before the run writes
// anything: a provider without an adapter is a refusal now rather than a failed
// node after a journal exists. A flow without agent nodes needs none. M1 runs
// one provider per flow, because a run's Env holds one model.
func (s session) modelFor(deps Deps) (model.Model, error) {
	var providers []string
	for _, node := range s.graph.Nodes {
		if node.Kind != "agent" {
			continue
		}
		// Stage 7 of the load has resolved every chain already.
		resolved, _ := s.agent.Resolve(node.Model, s.graph.Model)
		if !slices.Contains(providers, resolved.Provider) {
			providers = append(providers, resolved.Provider)
		}
	}
	switch len(providers) {
	case 0:
		return nil, nil
	case 1:
		return deps.Models(providers[0])
	default:
		slices.Sort(providers)
		return nil, fmt.Errorf("flow %q asks %s; M1 runs one provider per flow",
			s.graph.Name, strings.Join(providers, " and "))
	}
}

// walker is the runner over this session's journal.
func (s session) walker(deps Deps, replay bool) *runner.Runner {
	return runner.New(runner.Options{
		Graph:   s.graph,
		Catalog: s.catalog,
		Journal: fileJournal{path: runs.JournalPath(s.root, s.id)},
		Env: flow.Env{
			Root:     s.root,
			Params:   s.params,
			Baseline: s.baseline,
			Model:    s.model,
			Agent:    s.agent,
		},
		Clock:  deps.Clock,
		Warn:   func(text string) { fmt.Fprintf(deps.Stderr, "warning: %s\n", text) },
		Replay: replay,
	})
}

// finish prints how a run ended and turns it into the exit code, as cli.py did:
// 0 done, 3 paused, a block's own code, and 1 for every other failure.
func finish(deps Deps, id string, result runner.Result, err error) int {
	if err != nil {
		return refuse(deps, err)
	}
	fmt.Fprintf(deps.Stdout, "run %s: %s\n", id, result.Status)
	for _, text := range []string{result.Question, result.Detail} {
		if text != "" {
			// A question read from a file ends in a newline of its own.
			fmt.Fprintln(deps.Stdout, strings.TrimRight(text, "\n"))
		}
	}
	switch {
	case result.Status == "paused":
		return exitPaused
	case result.Status == "done":
		return exitOK
	case result.ExitCode != nil:
		return *result.ExitCode
	default:
		return exitFailed
	}
}
```

- [ ] **Step 6: Write `run.go`**

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/xidus90/ultra-loom/internal/flow"
	"github.com/xidus90/ultra-loom/internal/gitwork"
	"github.com/xidus90/ultra-loom/internal/runs"
)

// optionFlag collects --option name=value, as often as it is given. A name
// given twice is refused: which of two values a run meant is not a guess to
// make.
type optionFlag map[string]string

func (o optionFlag) String() string { return "" }

func (o optionFlag) Set(text string) error {
	name, value, found := strings.Cut(text, "=")
	if !found || name == "" {
		return fmt.Errorf("want name=value, got %q", text)
	}
	if _, taken := o[name]; taken {
		return fmt.Errorf("option %s is given twice", name)
	}
	o[name] = value
	return nil
}

func runCommand(args []string, deps Deps) int {
	flags := flag.NewFlagSet("ulflow run", flag.ContinueOnError)
	root := flags.String("root", ".", "the project directory")
	options := optionFlag{}
	flags.Var(options, "option", "a parameter of the flow as name=value; may be repeated")
	name, code, ok := arguments("run", args, true, flags, deps.Stderr)
	if !ok {
		return code
	}
	s, err := openFlow(*root, name, options, deps)
	if err != nil {
		return refuse(deps, err)
	}
	if s.model, err = s.modelFor(deps); err != nil {
		return refuse(deps, err)
	}
	// Taken once, here, and carried in the marker from now on: asked again on a
	// resume, git would answer with the tree the run has meanwhile edited.
	s.baseline = takeBaseline(*root)
	if s.needsBaseline() && s.baseline == nil {
		return refuse(deps, fmt.Errorf(
			"%s measures against the commit it starts from, and git gives %s none; start it inside a repository", name, *root))
	}
	marker := runs.Marker{Flow: name, Options: options, Baseline: s.baseline, Runtime: "go", Version: version}
	if s.id, err = runs.Claim(*root, marker); err != nil {
		return refuse(deps, err)
	}
	result, err := s.walker(deps, false).Run(context.Background())
	if err != nil {
		forgetUnstarted(*root, s.id)
	}
	return finish(deps, s.id, result, err)
}

// takeBaseline is what a run starts from, or nil where git cannot say. Both
// questions are asked either way, so that one branch answers them: outside a
// repository both fail, and a commit without its changed files beside it would
// read like a whole baseline.
func takeBaseline(root string) *flow.Baseline {
	commit, headErr := gitwork.HeadCommit(root)
	dirty, changedErr := gitwork.ChangedFiles(root)
	if headErr != nil || changedErr != nil {
		return nil
	}
	return &flow.Baseline{Commit: commit, Dirty: dirty}
}

// forgetUnstarted takes back the marker of a run the runner refused before its
// first step. Such a run has not happened, and a marker with no journal beside
// it would keep its number taken for nothing. A run with a journal keeps its
// marker: the journal records what happened, and the marker says which flow.
func forgetUnstarted(root, id string) {
	if _, err := os.Stat(runs.JournalPath(root, id)); err == nil {
		return
	}
	_ = os.Remove(runs.MarkerPath(root, id))
}
```

- [ ] **Step 7: Write `resume.go`**

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xidus90/ultra-loom/internal/journal"
	"github.com/xidus90/ultra-loom/internal/runs"
)

func resumeCommand(args []string, deps Deps) int {
	flags := flag.NewFlagSet("ulflow resume", flag.ContinueOnError)
	root := flags.String("root", ".", "the project directory")
	text := flags.String("answer", "", "the answer to the run's open gate")
	id, code, ok := arguments("resume", args, true, flags, deps.Stderr)
	if !ok {
		return code
	}
	// No --answer at all is a look at the gate, which asks again. An empty
	// --answer is an answer, and one no choice matches.
	var answer *string
	flags.Visit(func(set *flag.Flag) {
		if set.Name == "answer" {
			answer = text
		}
	})
	s, gate, err := recorded(*root, id, deps)
	if err != nil {
		return refuse(deps, err)
	}
	if gate == nil {
		// A resume over a complete journal would execute nothing and report
		// done -- exit 0 for a run nobody carried onward.
		return refuse(deps, fmt.Errorf(
			"run %s is not waiting at a gate; there is nothing to answer. Use `ulflow replay` to re-derive it, or `ulflow run` to start a new one", id))
	}
	if s.model, err = s.modelFor(deps); err != nil {
		return refuse(deps, err)
	}
	result, err := s.walker(deps, false).Resume(context.Background(), answer)
	return finish(deps, id, result, err)
}

func replayCommand(args []string, deps Deps) int {
	flags := flag.NewFlagSet("ulflow replay", flag.ContinueOnError)
	root := flags.String("root", ".", "the project directory")
	id, code, ok := arguments("replay", args, true, flags, deps.Stderr)
	if !ok {
		return code
	}
	s, gate, err := recorded(*root, id, deps)
	if err != nil {
		return refuse(deps, err)
	}
	if gate != nil {
		return refuse(deps, fmt.Errorf(
			"run %s never finished: it is waiting at gate %q; answer it with `ulflow resume` before replaying", id, gate.Node))
	}
	// A replay executes no node, so it asks for no model.
	result, err := s.walker(deps, true).Run(context.Background())
	return finish(deps, id, result, err)
}

// recorded reads back what a run was started with -- its flow, its options, its
// baseline -- and the gate it waits at, if any.
func recorded(root, id string, deps Deps) (session, *journal.PendingGate, error) {
	path := runs.JournalPath(root, id)
	if _, err := os.Stat(path); err != nil {
		return session{}, nil, noRun(root, id)
	}
	marker, err := runs.ReadMarker(runs.MarkerPath(root, id))
	if err != nil {
		return session{}, nil, err
	}
	if marker == nil {
		return session{}, nil, fmt.Errorf("run %q does not say which flow it belongs to", id)
	}
	// Neither runtime carries on the other's runs: their input hashes differ,
	// so a retrace would find nothing and do every node's work again.
	if marker.Runtime != "go" {
		return session{}, nil, fmt.Errorf("run %s was started by the Python runtime; carry it on with ultraloom", id)
	}
	gate, err := journal.Pending(path)
	if err != nil {
		return session{}, nil, err
	}
	s, err := openFlow(root, marker.Flow, marker.Options, deps)
	if err != nil {
		return session{}, nil, err
	}
	s.id, s.baseline = id, marker.Baseline
	if s.needsBaseline() && s.baseline == nil {
		// Taking one now would measure against the tree the run has meanwhile
		// edited, and everything it changed would count as untouched.
		return session{}, nil, fmt.Errorf(
			"run %s was started outside a repository, and flow %s measures against a commit; start a new run with `ulflow run`", id, marker.Flow)
	}
	return s, gate, nil
}

// noRun refuses a run number nothing was journalled under.
func noRun(root, id string) error {
	return fmt.Errorf("no run %q under %s", id, filepath.Join(root, runs.Dir))
}
```

- [ ] **Step 8: Write `show.go` and `journal.go`**

`cmd/flow/show.go`:

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/xidus90/ultra-loom/internal/flowload"
	"github.com/xidus90/ultra-loom/internal/journal"
	"github.com/xidus90/ultra-loom/internal/runs"
)

// showCommand prints one line per journal entry, in cli.py's columns: node,
// kind, outcome, tokens, seconds and tool profile.
func showCommand(args []string, deps Deps) int {
	flags := flag.NewFlagSet("ulflow show", flag.ContinueOnError)
	root := flags.String("root", ".", "the project directory")
	id, code, ok := arguments("show", args, true, flags, deps.Stderr)
	if !ok {
		return code
	}
	path := runs.JournalPath(*root, id)
	if _, err := os.Stat(path); err != nil {
		return refuse(deps, noRun(*root, id))
	}
	entries, err := journal.Entries(path)
	if err != nil {
		return refuse(deps, err)
	}
	for _, entry := range entries {
		tools := "-"
		if entry.Tools != nil && *entry.Tools != "" {
			tools = *entry.Tools
		}
		fmt.Fprintf(deps.Stdout, "%-24s %-6s %-7s %7d tok %7.2fs %s\n",
			entry.Node, entry.Kind, entry.Outcome, entry.Tokens, entry.Seconds, tools)
	}
	return exitOK
}

// listCommand prints every flow a project has, with its origin and ok, or the
// findings that keep it from loading: a flow missing from the list reads as a
// flow nobody wrote.
func listCommand(args []string, deps Deps) int {
	flags := flag.NewFlagSet("ulflow list", flag.ContinueOnError)
	root := flags.String("root", ".", "the project directory")
	if _, code, ok := arguments("list", args, false, flags, deps.Stderr); !ok {
		return code
	}
	agent, catalog, err := project(*root, deps)
	if err != nil {
		return refuse(deps, err)
	}
	for _, entry := range flowload.List(*root, catalog, agent) {
		state := "ok"
		if entry.Problem != "" {
			state = strings.ReplaceAll(entry.Problem, "\n", "\n    ")
		}
		fmt.Fprintf(deps.Stdout, "%-24s %-8s %s\n", entry.Name, entry.Origin, state)
	}
	return exitOK
}
```

`cmd/flow/journal.go`:

```go
package main

import "github.com/xidus90/ultra-loom/internal/journal"

// fileJournal is a run's journal file as the runner reads and writes it. The
// runner takes an interface, not a path, so that its own tests need no files;
// this is where the path comes back.
type fileJournal struct{ path string }

func (j fileJournal) Entries() ([]journal.Entry, error) { return journal.Entries(j.path) }

func (j fileJournal) Append(entry journal.Entry) error { return journal.Append(j.path, entry) }

func (j fileJournal) Pending() (*journal.PendingGate, error) { return journal.Pending(j.path) }
```

- [ ] **Step 9: Run the cmd/flow tests with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -cover ./cmd/flow/`
Expected: `ok`, alle Tests grün, Coverage knapp unter 100 %: nur `main()` ist ungedeckt.

Die ungedeckten Zeilen nachlesen:

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -coverprofile=cmd-flow.out ./cmd/flow/`

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go tool cover -func=cmd-flow.out`
Expected: jede Funktion außer `main` bei `100.0%`. Danach `cmd-flow.out` löschen. Steht eine andere Funktion unter 100 %, fehlt ein Test: ihn schreiben, rot sehen, grün machen — nicht die Zeile entfernen, außer sie ist nachweislich unerreichbar, und dann mit Kommentar, warum.

Scheitert ein Test mit einer Meldung, die der Plan anders erwartet, als der Code aus Wellen 1 und 2 sie formuliert (etwa ein Befund des Laders), gilt der Code: den Befund im Bericht nennen und die Erwartung im Test angleichen, nicht die Meldung im Paket.

- [ ] **Step 10: Ignore the binary**

In `.gitignore` nach `/ulinit_*.exe` einfügen:

```
/ulflow
/ulflow.exe
/flow.exe
/ulflow_*.exe
```

`/flow.exe` fängt, was `go install ./cmd/flow` fälschlich ablegen würde; die globale CLAUDE.md erklärt, warum `go install` hier der falsche Weg ist.

- [ ] **Step 11: vet, gofmt and commit**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go vet ./cmd/flow/`
Expected: keine Ausgabe.

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && gofmt -l cmd/flow`
Expected: keine Ausgabe.

```
Add ulflow: run, resume, replay, show and list

cmd/flow is the edge of the Go runtime: it finds and loads a flow, reads
the run's options into its parameters, asks for the one model its agent
nodes need, takes the baseline once, claims a run number with the marker
and hands the rest to the runner. resume and replay read the marker back
and refuse what cli.py refused, with ulflow in the hints; show prints
cli.py's columns, and list names every flow with why one will not load.

Clock, model factory and blocks come in through Deps. The real factory
has no adapter yet, so a flow with an agent node is refused before its
run exists, and tests hand in the fake model instead. A run the runner
refuses before its first step takes its marker back.
```

---

### Task 4: Golden-Journal

**Files:**
- Create: `cmd/flow/golden_test.go`
- Create: `cmd/flow/testdata/golden/example.jsonl` (erzeugt, gelesen, dann eingecheckt)
- Modify: `.gitattributes` (eine Zeile)

**Interfaces:**
- Consumes: `newHarness`, `done`, `pausedExample`, `exitPaused`, `cli` aus Task 3.

- [ ] **Step 1: Pin the golden file to LF**

In `.gitattributes` nach der Zeile `internal/flowload/testdata/**     text eol=lf` einfügen:

```
cmd/flow/testdata/**              text eol=lf
```

- [ ] **Step 2: Write the golden test**

Create `cmd/flow/golden_test.go`:

```go
package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/ultra-loom/internal/runs"
)

var update = flag.Bool("update", false, "write the golden journal instead of comparing against it")

// The golden journal: the example flow against the fake model and a clock that
// repeats itself, run to its gate, answered, and ended at its exit. Byte for
// byte, because a journal is what every later resume reads, and a change in one
// byte of it is a change in what a run of this build finds there.
func TestTheExampleFlowWritesTheGoldenJournal(t *testing.T) {
	h := newHarness(done())
	root := pausedExample(t, h)
	if code := h.run(root, "resume", "0001", "--answer", "no: too thin"); code != 4 {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	got, err := os.ReadFile(runs.JournalPath(root, "0001"))
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "golden", "example.jsonl")
	if *update {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("the journal is not the golden one\n got:\n%s\nwant:\n%s", got, want)
	}
}
```

Der Zweig `if *update` liegt in einer Testdatei und zählt nicht zur Coverage.

- [ ] **Step 3: Run it to see it fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -run TestTheExampleFlowWritesTheGoldenJournal ./cmd/flow/`
Expected: FAIL mit `open testdata\golden\example.jsonl: The system cannot find the path specified.` (oder der POSIX-Entsprechung).

- [ ] **Step 4: Write the golden journal**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -run TestTheExampleFlowWritesTheGoldenJournal ./cmd/flow/ -update`
Expected: `ok`.

- [ ] **Step 5: Read the golden journal line by line**

Die Datei `cmd/flow/testdata/golden/example.jsonl` mit Read öffnen und gegen diese Erwartung prüfen. Hashes werden nicht nachgerechnet; alles andere schon:

| Zeile | `node` | `kind` | `outcome` | `delta` | `tools` | `effort` | `tokens` | `seconds` | `detail` | `model` |
|---|---|---|---|---|---|---|---|---|---|---|
| 1 | `draft` | `agent` | `ok` | `{"count":2,"verdict":"done"}` | `"edit"` | `"high"` | 120 | 0.5 | `null` | `"claude:cli-default"` |
| 2 | `approve` | `gate` | `paused` | `{}` | `null` | `null` | 0 | 0.5 | `"Approve the plan after 2 rounds?\n"` | `null` |
| 3 | `approve` | `gate` | `ok` | `{"answer":"no","answer_text":"too thin"}` | `null` | `null` | 0 | 0 | `"answered: no: too thin"` | `null` |
| 4 | `stop` | `exit` | `error` | `{}` | `null` | `null` | 0 | 0.5 | `"rejected after 2 rounds"` | `null` |

Dazu:
- Jede Zeile trägt `input_hash` und `definition_hash`, beide nicht leer.
- `input_hash` von Zeile 2 und 3 sind gleich: die Antwort gehört dem Besuch, an dem das Tor pausierte.
- Die Datei endet mit genau einem `\n`, und keine Zeile enthält `\r`.

Weicht eine Zeile ab, ist das ein Befund über den Runner oder einen Baustein: nicht die Datei von Hand ändern, sondern die Abweichung berichten und anhalten.

- [ ] **Step 6: Run the whole module and check coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -cover ./...`
Expected: alles grün; `internal/runs`, `internal/flow`, `internal/runner`, `internal/flowload` bei 100,0 %; `cmd/flow` nur mit `main` ungedeckt.

- [ ] **Step 7: gofmt and commit**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && gofmt -l cmd/flow`
Expected: keine Ausgabe.

```
Pin the journal of the example flow byte for byte

The example flow runs against the fake model and a clock that repeats
itself: it pauses at its gate, is answered "no: too thin", and ends at
its exit with code 4. The four lines it writes are the golden journal,
compared byte for byte. It is the first test that meets the runner with
the real journal, the real blocks and the real loader at once.
```

---

### Task 5: Installation, Neubau, Messung, Befunde (Orchestrator)

**Files:**
- Modify: `scripts/install.ps1`, `scripts/install.sh`
- Modify: `docs/benchmarks.md`, `docs/benchmarks.de.md`
- Create: `docs/.superpowers/plans/2026-09-14-ulflow-welle-3-befunde.md`

- [ ] **Step 1: Build ulflow in the install scripts**

`scripts/install.ps1`: nach dem Block `Building ulinit.exe...` einfügen:

```powershell
Write-Host "Building ulflow.exe..." -ForegroundColor Gray
go build -o (Join-Path $goBin "ulflow.exe") (Join-Path $root "cmd\flow")
```

`$tools = @("ulguard", "ulinit")` wird `$tools = @("ulguard", "ulinit", "ulflow")`, und in der OK-Meldung `(ulguard, ulinit)` wird `(ulguard, ulinit, ulflow)`.

`scripts/install.sh`: nach dem Block `Building ulinit...` einfügen:

```bash
echo "Building ulflow..."
go build -o "${GOBIN_DIR}/ulflow${EXE}" "${ROOT_DIR}/cmd/flow"
```

`chmod +x` bekommt `"${GOBIN_DIR}/ulflow${EXE}"` als drittes Argument, und die OK-Meldung nennt `(ulguard, ulinit, ulflow)`.

- [ ] **Step 2: Build the three binaries in the worktree**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go build -o ulflow.exe ./cmd/flow`

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go build -o ulguard.exe ./cmd/guard`

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go build -o ulinit.exe ./cmd/init`

Expected: je keine Ausgabe; `git status --short` zeigt keines der drei Binaries.

- [ ] **Step 3: Install into `~/go/bin`**

Dieser Schritt tauscht das `ulguard` aus, das die Hooks aller laufenden Sitzungen rufen. Der Nutzer hat den Neubau am 2026-09-14 freigegeben.

Run: `pwsh -NoProfile -File "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness/scripts/install.ps1"`
Expected: `Building ulguard.exe...`, `Building ulinit.exe...`, `Building ulflow.exe...`, drei `Created Bash shim`, `[OK] UltraLoom binaries (ulguard, ulinit, ulflow) …`.

Run: `ulflow --version`
Expected: `0.1.0`. Kommt `command not found`, liegt `~/go/bin` nicht auf dem PATH dieser Shell; mit `"$HOME/go/bin/ulflow.exe" --version` gegenprüfen und berichten.

- [ ] **Step 4: End to end with the installed binaries**

Ein Projekt im Scratchpad mit dem Tor-Flow aus `cmd/flow/helpers_test.go` (`askFlow`). Mit Write anlegen:
- `<Scratchpad>/ulflow-e2e/.ultraloom/config.toml` mit dem Inhalt `[verify]` und Zeilenumbruch,
- `<Scratchpad>/ulflow-e2e/.ultraloom/flows/ask.toml` mit dem Inhalt von `askFlow`,
- `<Scratchpad>/ulflow-e2e/.ultraloom/flows/questions/confirm.md` mit `Ship it?` und Zeilenumbruch.

Run: `ulflow run ask --root "<Scratchpad>/ulflow-e2e"`
Expected: `run 0001: paused` und `Ship it?`, Exit 3.

Run: `printf '{"session_id":"e2e"}' | ulguard hook session-start --host claude --root "<Scratchpad>/ulflow-e2e"`
Expected: ein JSON-Umschlag, dessen `additionalContext` `run 0001 is waiting at confirm: Ship it?` und `answer it with: ulflow resume 0001 --answer "your answer"` enthält. Das ist das Grün-Kriterium der Spec für Welle 3.

Run: `ulflow resume 0001 --answer yes --root "<Scratchpad>/ulflow-e2e"`
Expected: `run 0001: done`, Exit 0.

Run: `ulflow show 0001 --root "<Scratchpad>/ulflow-e2e"`
Expected: zwei Zeilen, `confirm gate paused …` und `confirm gate ok …`.

Run: `ulflow list --root "<Scratchpad>/ulflow-e2e"`
Expected: `ask                      project  ok`.

- [ ] **Step 5: Measure the hook with the marker actually read**

Vergleich zwischen `<Scratchpad>/ulguard-before.exe` (vor Task 1 gebaut, siehe „Ablauf") und dem neuen `ulguard.exe` im Worktree, beide gegen das E2E-Projekt aus Step 4 nach einem neuen, wartenden Lauf (`ulflow run ask --root …` noch einmal ausführen, er bekommt `0002`). Gemessen wird der Pfad, der die Marke liest: `hook session-start`.

Run (PowerShell):

```powershell
$e2e = '<Scratchpad>\ulflow-e2e'; foreach ($exe in '<Scratchpad>\ulguard-before.exe', 'C:\Users\micro\Documents\#GIT\ultraloom\.worktrees\agent-harness\ulguard.exe') { '{"session_id":"bench"}' | & $exe hook session-start --host claude --root $e2e *> $null; $times = foreach ($i in 1..25) { (Measure-Command { '{"session_id":"bench"}' | & $exe hook session-start --host claude --root $e2e *> $null }).TotalMilliseconds }; $sorted = $times | Sort-Object; "{0}: cold-ish first run discarded; median {1:N1} ms, min {2:N1} ms, max {3:N1} ms, size {4} bytes" -f (Split-Path $exe -Leaf), $sorted[12], $sorted[0], $sorted[24], (Get-Item $exe).Length }
```

Expected: zwei Zeilen mit Median, Minimum, Maximum und Größe. Die Zahlen gehen unverändert in Step 6.

- [ ] **Step 6: Record the measurements**

In `docs/benchmarks.md` unter `## Chronological Benchmark Log` als ersten Eintrag (neueste zuerst):

```markdown
### 2026-09-14 — ulguard session-start: reading the run marker

* **Repository:** `ultraloom` on `feature/agent-harness`, Windows 11, Go 1.27. Probe at `9c0516d`; hook measurement before and after the commit of wave 3's Task 1.
* **Objective:** `ulguard hook session-start` names `ulflow resume` for a Go run, which means reading the run's marker through `internal/runs`. That pulls `internal/flow`, `internal/flowcfg`, `internal/model` and `internal/flow/tmpl` into the hook binary; TOML was linked already.
* **Method:** Probe: `ulguard status --root <worktree>`, 25 warm runs after one discarded, with and without a blank import of `internal/runs` (the import calls nothing, so the linker keeps little of it). Real use: `ulguard hook session-start --host claude` against a project with one waiting Go run, 25 warm runs after one discarded, the binary before Task 1 against the one after. PowerShell `Measure-Command`, median of 25.
* **Findings:**

| Scenario | Before | After | Difference |
| :--- | :---: | :---: | :---: |
| Probe, `ulguard status`, binary size | 5,947,904 bytes | 5,951,488 bytes | +3,584 bytes |
| Probe, `ulguard status`, warm | median 49.2 ms (46.9–58.7) | median 50.1 ms (48.7–52.9) | +0.9 ms, inside the spread |
| `hook session-start`, one waiting Go run, warm | <Step 5, before> | <Step 5, after> | <difference> |
| `hook session-start`, binary size | <Step 5, before> | <Step 5, after> | <difference> |
```

Die vier Zellen der letzten beiden Zeilen tragen die Zahlen aus Step 5, in derselben Schreibweise wie die Probe-Zeilen. In `docs/benchmarks.de.md` unter `## Chronologisches Benchmark-Protokoll` derselbe Eintrag auf Deutsch, Überschrift `### 14.09.2026 — ulguard session-start: die Laufmarke lesen`, Zahlen mit Dezimalkomma wie in den übrigen deutschen Einträgen.

- [ ] **Step 7: Commit install scripts and benchmarks**

Stagen: `scripts/install.ps1`, `scripts/install.sh`, `docs/benchmarks.md`, `docs/benchmarks.de.md`.

```
Install ulflow beside ulguard and ulinit, and record the hook's cost

Both install scripts build ulflow as the third binary with go build -o,
and the PowerShell script gives it a Bash shim like the other two.

Reading the run marker in ulguard's session-start is measured twice: a
probe that only linked internal/runs, and the hook itself with a waiting
Go run before and after the change.
```

- [ ] **Step 8: Read the remote**

Run: `git -C "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" ls-remote origin feature/agent-harness`
Expected: leer.

- [ ] **Step 9: Write the findings of wave 3**

`docs/.superpowers/plans/2026-09-14-ulflow-welle-3-befunde.md`, nach dem Muster der Befunde von Welle 2: Stand mit Commit, Entscheidungen während der Ausführung (Tabelle mit Grund und Kosten), Messungen, Offenes aus den Reviews, was für die Lader-Politur und für M2 bleibt, Betrieb. Die SDD-Arbeitsdateien unter `.superpowers/sdd/` werden danach gelöscht, nicht vorher.

Commit:

```
Record what wave 3 of ulflow decided and left open
```
