# ulflow M1, Welle 2 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Tasks 1 bis 4 laufen mit superpowers:dispatching-parallel-agents: vier Subagenten in einer Nachricht, jeder in seinem eigenen Worktree. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `ulflow` lädt einen Flow aus TOML und prüft ihn in sieben Stufen, die Bausteine `gate`, `exit` und `agent` sind gebaut, und der Runner geht einen Graphen, geht ein Journal nach, hält an Toren an und hält Deckel ein — alles zusammengeführt in `feature/agent-harness`. Was fehlt, ist danach nur noch `cmd/flow` (Welle 3).

**Architecture:** Task 0 zieht in `.worktrees/agent-harness` die Verträge nach, die zwei oder mehr Lanes gemeinsam bräuchten: das Rohfeld am Knoten für den Definitions-Hash, `Coerce` als einzige Typtabelle, `ReadText`/`Render` als einzige Textstelle, ein konkretes Register und `json.Number` im Journalleser. Danach laufen vier Lanes gleichzeitig, jede in `.worktrees/ulflow-<lane>` auf `ulflow/<lane>`: `loader` besitzt das neue Paket `internal/flowload`, `gate-exit` und `agent` je eigene Dateien in `internal/blocks`, `runner` das Paket `internal/runner`. Keine Lane fasst eine Datei einer anderen an. Zusammengeführt wird mit `--no-ff` in `feature/agent-harness`.

**Tech Stack:** Go mit dem Sprachstand `go 1.22` aus `go.mod` (installiert ist go1.27.0, maßgeblich ist `go.mod`), `github.com/BurntSushi/toml` v1.6.0, `encoding/json`, `crypto/sha256`, git auf dem PATH.

**Spec:** `docs/.superpowers/specs/2026-09-11-ulflow-laufzeit-design.md`, Abschnitte „Verhaltensvertrag", „Das Flow-Format", „Bausteine", „Prüfung beim Laden", „Auffindung", „Modellwahl und Modell-Port", „Journal" und „Wellen" (Welle 2).

**Vorgänger:** `docs/.superpowers/plans/2026-09-11-ulflow-welle-0-und-1.md` und dessen Befunde in
`docs/.superpowers/plans/2026-09-11-ulflow-welle-0-und-1-befunde.md`. Die fünfzehn Punkte unter „Für den Plan von Welle 2" sind unten einzeln entschieden.

## Global Constraints

- Kommentare, Bezeichner, Fehlermeldungen und Commit-Nachrichten englisch.
- Sprachstand `go 1.22`: kein range-over-func, kein `maps.Keys`, kein `slices.Collect`, keine API ab Go 1.23.
- TDD: jeder Code-Schritt beginnt mit einem Test, der vorher rot ist.
- 100 % Coverage in jedem Paket, das der Task anlegt oder ändert; `go test -cover` zeigt `coverage: 100.0% of statements`.
- Vor jedem Commit `gofmt -l` auf die Dateien des Tasks: leere Ausgabe. Sonst `gofmt -w` und neu prüfen.
- Kein `Co-Authored-By` und keine Nennung eines Modells im Commit. Mehrzeilige Nachrichten über eine Datei und `git commit -F`.
- Vor jedem Commit `git rev-parse --abbrev-ref HEAD`, `git log --oneline -1` und `git diff --cached --stat` lesen; nur die Dateien stagen, die der Task nennt.
- Kein Push. Kein `--no-verify`.
- Keine Änderung an `go.mod` oder `go.sum`.
- Eine Lane fasst nur die Dateien an, die ihr Task unter **Files** nennt.
- Kein Paket ruft `time.Now`. Der Runner bekommt seine Uhr übergeben.
- Ein frischer Worktree bekommt vor seinem ersten Commit `env -u VIRTUAL_ENV uv sync`. Sonst scheitert der Pre-Commit-Hook an `.venv/Scripts/ultraloom.exe` („Zugriff verweigert"): Das `#` im Pfad lässt uv das Projekt bei jedem `uv run` neu installieren.
- Nach dem letzten Commit in einem Lane-Worktree die `dmypy`-Daemons aus dessen `.venv` beenden. Jeder Commit hinterlässt einen, und `ulguard worktree-remove` scheitert sonst mit „Invalid argument".
- Die Paketdokumentation neuer Pakete steht in `doc.go`; keine zweite `// Package …`-Zeile in anderen Dateien.
- Dateien mit Backslashes im Inhalt mit Write oder Edit schreiben, nie über ein Bash-Heredoc: dort wurde in Welle 1 einmal `\\` zu `\`.
- Ein Shell-Befehl je Schritt. Jeder Befehl nennt sein Verzeichnis mit `cd "<Worktree>" && …`.
- Meldet ein Hook einen Befund über ein anderes Verzeichnis als den eigenen Worktree, wird er berichtet und nicht behoben.

## Abweichung vom Format der Welle-0-und-1-Pläne

Dort stand zu jedem Schritt der vollständige Implementierungscode. Für `internal/flowload` und `internal/runner` steht er hier nicht: beide Pakete sind zusammen größer als alle sieben Blattpakete der Welle 1, und ein zweimal geschriebener Lader hilft niemandem. Was stattdessen dasteht:

- Für `internal/blocks` (Tasks 2 und 3) **Tests und Implementierung vollständig**. Beide sind kurz.
- Für `internal/flowload` (Task 1) **die Tests vollständig**, dazu eine Tabelle mit allen elf Testflows, was in jedem falsch ist und welche Meldung er wörtlich erwarten soll. Der Rumpf ist Signaturen und Ablauf je Funktion.
- Für `internal/runner` (Task 4) die Tests des Gehens vollständig; Tore und Nachgehen als Tabelle, ein Testname je Zeile des Verhaltensvertrags mit dem, was er prüft. Der Gang steht als kommentiertes Gerüst mit jedem frühen Rücksprung darin.

Die Tests sind der Vertrag; der Rumpf ist Handwerk. Wo dieser Plan eine Tabelle statt Code gibt, ist jede Zeile ein Testfall mit der Meldung, die er erwartet — keine Zeile davon ist „und so weiter".

## Entscheidungen, die dieser Plan trifft

Die fünfzehn Punkte aus den Befunden der Welle 1, in deren Reihenfolge. Vier davon hat der Nutzer am 2026-09-11 entschieden; sie sind als solche gekennzeichnet.

1. **`journal.Entries` decodiert mit `UseNumber`** (Nutzerentscheidung). Zahlen im `delta` kommen als `json.Number` statt als `float64`, und `flow.Coerce` wandelt sie mit `strconv.ParseInt` in den deklarierten Typ. Damit überlebt eine ganze Zahl über 2^53 das Nachgehen, und die Eingabe-Hashes späterer Knoten bleiben dieselben. `journal.Canonical` liest ohnehin schon mit `UseNumber`, und `json.Number` marshalt sich als das Literal, das dastand — der Schreiber bleibt unberührt.
2. **`flow.Node` bekommt `Raw map[string]any`** (Nutzerentscheidung), den Knoteneintrag, wie TOML ihn decodiert hat. Der Runner gibt `node.Raw` an `journal.DefinitionHash`, nicht `flow.Node`: eine Struktur ohne JSON-Tags trüge Go-Feldnamen in den Hash, und `max_visits` steckte schon als geparste `Cap` darin. Nicht-endliche Zahlen (`nan`, `inf`) in einem Knotenschlüssel lehnt der Lader in Stufe 1 ab, damit `Canonical` nicht erst zur Laufzeit scheitert.
3. **Ein Name in `[params]` und `[state]` zugleich ist ein Ladefehler**, gemeldet in Stufe 1 zusammen mit den übrigen Deklarationsbefunden. Sonst entschiede die Reihenfolge, in der eine Bedingung die beiden Tabellen befragt, was ein Name bedeutet.
4. **`true`, `false` und `END` sind reserviert** — für Knoten-, Feld- und Parameternamen, Stufe 1. `expr` liest `true` und `false` als Literal, bevor es Parameter nachschlägt, und `END` ist der Pseudoknoten. Beide Namen wären sonst nicht bloß hässlich, sondern unerreichbar.
5. **Der Runner baut `State.Fields` beim Start aus allen Deklarationen** und entfernt nie einen Schlüssel. `comparison.Holds` prüft Typen per Assertion; ein fehlendes Feld wäre dort keine falsche Antwort, sondern eine Panik. Ein Delta mit einem Namen, den der Flow nicht deklariert, ist ein Knotenfehler.
6. **Der Eingabe-Hash deckt Felder und Besuchszähler** (Nutzerentscheidung). Python hasht nur die Daten; ein Tor auf einem Zyklus pausiert dort nur dann erneut, wenn sich zwischen zwei Durchgängen ein Feld bewegt hat, und gibt sonst still die alte Antwort zurück. Mit den Zählern im Hash bekommt jeder Durchgang einen eigenen Schlüssel. Die Spec erlaubt die Abweichung ausdrücklich: keine Seite setzt Läufe der anderen fort.
7. **Der Baustein `agent` setzt den Anbieter vor den Modellnamen.** `model.Reply.Model` ist ein nackter Name; `flow.Result.Model` braucht `<anbieter>:<modell>`. Meldet der Adapter nichts, gilt `flowcfg.Resolved.Label()`.
8. **`exit` lehnt die Codes 0, 1 und 3 beim Laden ab** (Nutzerentscheidung), in `Check` und damit in Stufe 2. Die Laufzeit belegt sie mit fertig, Fehler und pausiert; ein Lauf, der mit 3 endet, ohne an einem Tor zu warten, wäre für jeden Aufrufer eine Falle.
9. **Der Runner prüft alle Deckel einmal beim Start** gegen die aufgelösten Parameter und bricht vor dem ersten Journaleintrag ab, wenn einer unter 1 liegt. Der Lader kann nur Literale prüfen; `--option max_rounds=-1` fällt erst hier auf. Nicht erst beim Erreichen des Knotens: ein Tippfehler in einer Option soll keinen bezahlten Modellaufruf kosten.
10. **Für eine Kante ohne `when` ruft der Lader `ParseCondition` nicht.** Eine leere Bedingung ist ein Fehler, und `when = ""` wird als solcher gemeldet.
11. **Feld-, Parameter- und Knotennamen sind `[A-Za-z_][A-Za-z0-9_]*`**, dieselbe Regel wie für Flow-Namen. Ein Name außerhalb davon ließe sich in keiner Bedingung nennen.
12. **`flow.Coerce(Type, any) (Value, error)` ist die einzige Typtabelle** und steht in Task 0. Der Lader wandelt damit die Vorgaben aus TOML (`int64`), der Runner die Deltas aus dem Journal (`json.Number`, `[]any`) und die Antworten des Modells. Zwei Tabellen driften; eine ist testbar.
13. **Prüfstufe 7 löst die drei Stufen der Modellkette einzeln auf:** `Resolve(node, "")`, `Resolve("", flow)` und `Resolve("", "")`. `Resolve` prüft nur den ersten gesetzten Namen und kennt die Stufe nicht, die ihn gesetzt hat; einzeln aufgelöst nennt die Meldung sie.
14. **Ein Deckel unter 1 ist ein Fehler, auch `Cap{}`.** Das ist Punkt 9 aus der anderen Richtung: ein handgebauter Testknoten ohne `MaxVisits` scheitert damit laut statt still. Der Brief der Runner-Lane sagt es noch einmal ausdrücklich.
15. **Das Lader-Paket heißt `internal/flowload`** und steht in Task 0 in der Spec-Tabelle „Pakete". `internal/flow/doc.go` sagt heute schon „lives in a package of its own"; mit Task 0 ist der Satz wahr.

Dazu vier Entscheidungen, die beim Schreiben dieses Plans dazukamen:

16. **`flow.ReadText` und `flow.Render` stehen in Task 0.** Tor und Agent rendern beide einen Text relativ zu `Graph.Dir`, und die beiden Lanes dürfen keine Datei teilen. `internal/flow` darf `internal/flow/tmpl` importieren — `tmpl` importiert nur die Standardbibliothek, es entsteht kein Zyklus.
17. **Das konkrete Register heißt `flow.Catalog`** und steht in Task 0. Lader (Stufen 2 und 4) und Runner brauchen dieselbe Implementierung von `flow.Registry`; in `internal/flow` gibt es dafür keinen Zyklus. `NewCatalog` nimmt entgegen, was es hält, und lehnt eine doppelte Knotenart ab. Es gibt keine Methode, die nachträglich eintragen kann.
18. **`Result.Exit` beendet den Lauf sofort; keine Fehlerkante wird angeboten.** Pythons `FlowExit` war eine Ausnahme und lief deshalb durch den Fehlerzweig, wo eine Fehlerkante sie fangen und den Code verschlucken konnte. Ein `exit`-Knoten ist kein Fehlschlag, sondern ein Ende mit Ansage. Der Journaleintrag bleibt `outcome = "error"` mit der Meldung als `detail`, damit das Vokabular des Journals das von Python bleibt und `ulguard` nichts Neues lernen muss; der Code reist im Ergebnis, nicht im Journal — wie in Python.
19. **M1 bettet keinen Flow ein.** `//go:embed` auf ein leeres Verzeichnis ist ein Übersetzungsfehler, und die Spec sagt, der erste mitgelieferte Flow ist der MVP. `flowload.List` meldet deshalb nur Projekt-Flows; die Verdeckungsregel ist als Funktion da und mit einer Testliste geprüft, nicht mit einem eingebetteten Verzeichnis.

## Ablauf

1. **Task 0** in `.worktrees/agent-harness` durch einen Subagenten. Orchestrator prüft: `go build ./...`, `go vet ./...`, `go test ./...`, `git log --oneline -1`.
2. **Lane-Worktrees anlegen**, vier Befehle, je einer (Orchestrator):
   - `git -C "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" worktree add "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-loader" -b ulflow/loader feature/agent-harness`
   - dasselbe für `gate-exit`, `agent`, `runner`
3. **In jedem Lane-Worktree** `cd "…/.worktrees/ulflow-<lane>" && env -u VIRTUAL_ENV uv sync` (Orchestrator, vor dem Dispatch).
4. **Tasks 1–4 parallel**: eine Nachricht, vier Agent-Aufrufe, jeder mit `model: "opus"`, dem vollständigen Text seines Tasks, den Global Constraints und seinem Worktree-Pfad. Einen `effort` nimmt der Agent-Aufruf nicht an; der Subagent läuft mit dem Effort der Sitzung. Nicht `isolation: "worktree"` benutzen; das legt unter `.claude/worktrees/` an.
5. **Task 5**: Prüfen und Zusammenführen durch den Orchestrator.

**Warum der Orchestrator den Diff jeder Lane liest:** `ulguard` prüft Pfadregeln relativ zu `--root` der Sitzung. Ein Pfad in einem Lane-Worktree liegt außerhalb davon, bleibt absolut und trifft keine Regel. In Lane-Worktrees schützen nur die Befehlsregeln. Deshalb ist `git diff --stat` gegen die Dateiliste der Lane Teil von Task 5.

**Warum `gate-exit` und `agent` trotzdem parallel laufen:** Sie teilen das Paket `internal/blocks`, aber keine Datei. Der Preis dafür steht in beiden Briefen: **keine gemeinsame `helpers_test.go`**. Jede Lane baut ihre Testhelfer in ihrer eigenen Testdatei auf, auch wenn sie sich ähneln. Zwei Lanes, die dieselbe Hilfsdatei anlegen, erzeugen beim Zusammenführen einen Konflikt in einer Datei, die keine von beiden besitzt.

---

### Task 0: Verträge nachziehen

**Worktree:** `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness`, Zweig `feature/agent-harness`

**Files:**
- Modify: `internal/flow/graph.go` (Feld `Raw` an `Node`, Methode `Cap.Limit`)
- Modify: `internal/flow/doc.go` (ein Satz)
- Create: `internal/flow/coerce.go`, `internal/flow/catalog.go`, `internal/flow/text.go`
- Test: `internal/flow/coerce_test.go`, `internal/flow/catalog_test.go`, `internal/flow/text_test.go`, `internal/flow/graph_test.go`
- Modify: `.gitattributes` (zwei Zeilen für die neuen Testdaten)
- Modify: `internal/journal/journal.go` (Decoder mit `UseNumber`)
- Modify: `internal/journal/journal_test.go`, `internal/journal/write_test.go` (zwei Vergleiche auf `float64`)
- Modify: `docs/.superpowers/specs/2026-09-11-ulflow-laufzeit-design.md` (Tabelle „Pakete")

**Interfaces:**
- Consumes: `internal/flow/tmpl` (`Parse`, `Template.Render`) — neu als Import von `internal/flow`; `tmpl` importiert nur die Standardbibliothek, ein Zyklus entsteht nicht.
- Produces:
  - `flow.Node.Raw map[string]any`
  - `func (n flow.Node) StringKey(name string) string`
  - `func (n flow.Node) UnknownKeys(known ...string) []string`
  - `func (c flow.Cap) Limit(params flow.Params) (int, error)`
  - `func flow.Coerce(t flow.Type, raw any) (flow.Value, error)`
  - `func flow.NewCatalog(blocks []flow.Block, predicates map[string]flow.Predicate) (*flow.Catalog, error)`
  - `func (c *flow.Catalog) Block(kind string) (flow.Block, bool)`
  - `func (c *flow.Catalog) Predicate(name string) (flow.Predicate, bool)`
  - `func (c *flow.Catalog) Kinds() []string`
  - `func (c *flow.Catalog) Predicates() map[string]flow.Predicate`
  - `func flow.ReadText(dir string, t flow.Text) ([]byte, error)`
  - `func flow.Render(raw []byte, state flow.State, params flow.Params) (string, error)`
  - `journal.Entries` liefert Zahlen im `delta` als `json.Number`

- [ ] **Step 1: Write the failing tests**

`internal/flow/coerce_test.go`:

```go
package flow_test

import (
	"encoding/json"
	"testing"

	"github.com/xidus90/ultra-loom/internal/flow"
)

func TestCoerceTakesEveryShapeAValueArrivesIn(t *testing.T) {
	cases := []struct {
		name string
		kind flow.Type
		raw  any
		want flow.Value
	}{
		{"a string stays one", flow.String, "x", "x"},
		{"an int stays one", flow.Int, 1, 1},
		{"TOML hands out int64", flow.Int, int64(2), 2},
		{"JSON without UseNumber hands out float64", flow.Int, float64(3), 3},
		{"the journal hands out json.Number", flow.Int, json.Number("4"), 4},
		{"a bool stays one", flow.Bool, true, true},
		{"a list of strings stays one", flow.StringList, []string{"a"}, []string{"a"}},
		{"JSON hands out []any", flow.StringList, []any{"a", "b"}, []string{"a", "b"}},
		{"an empty []any is an empty list", flow.StringList, []any{}, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := flow.Coerce(c.kind, c.raw)
			if err != nil {
				t.Fatal(err)
			}
			if list, ok := c.want.([]string); ok {
				gotList, ok := got.([]string)
				if !ok {
					t.Fatalf("got %T, want []string", got)
				}
				if len(gotList) != len(list) {
					t.Fatalf("got %v, want %v", gotList, list)
				}
				for i := range list {
					if gotList[i] != list[i] {
						t.Fatalf("got %v, want %v", gotList, list)
					}
				}
				return
			}
			if got != c.want {
				t.Fatalf("got %#v, want %#v", got, c.want)
			}
		})
	}
}

// The whole reason Entries reads with UseNumber: 2^53+1 has no float64.
func TestCoerceKeepsAnIntTooLargeForAFloat(t *testing.T) {
	got, err := flow.Coerce(flow.Int, json.Number("9007199254740993"))
	if err != nil {
		t.Fatal(err)
	}
	if got != 9007199254740993 {
		t.Fatalf("got %v, want 9007199254740993", got)
	}
}

func TestCoerceRefusesWhatIsNotTheDeclaredType(t *testing.T) {
	cases := []struct {
		name string
		kind flow.Type
		raw  any
		want string
	}{
		{"an int is no string", flow.String, 1, `cannot read 1 as string`},
		{"a string is no int", flow.Int, "1", `cannot read 1 as int`},
		{"a fraction is no int", flow.Int, float64(3.5), `cannot read 3.5 as int; it is not a whole number`},
		{"a fraction is no int in json.Number either", flow.Int, json.Number("3.5"), `cannot read 3.5 as int; it is not a whole number`},
		{"a number too large for an int", flow.Int, json.Number("99999999999999999999"), `cannot read 99999999999999999999 as int`},
		{"an int is no bool", flow.Bool, 1, `cannot read 1 as bool`},
		{"a string is no list", flow.StringList, "a", `cannot read a as list[string]`},
		{"a list of ints is no list of strings", flow.StringList, []any{1}, `cannot read 1 as string`},
		{"an unknown type", flow.Type("date"), "x", `unknown type "date"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := flow.Coerce(c.kind, c.raw)
			if err == nil {
				t.Fatal("want an error")
			}
			if err.Error() != c.want {
				t.Fatalf("got %q, want %q", err, c.want)
			}
		})
	}
}
```

`internal/flow/catalog_test.go`:

```go
package flow_test

import (
	"context"
	"slices"
	"testing"

	"github.com/xidus90/ultra-loom/internal/flow"
)

type namedBlock struct{ kind string }

func (b namedBlock) Kind() string                          { return b.kind }
func (namedBlock) Check(flow.Node) []string                { return nil }
func (namedBlock) Texts(flow.Node) []flow.Text             { return nil }
func (namedBlock) Writes(flow.Node) map[string]flow.Type   { return nil }
func (namedBlock) NeedsBaseline() bool                     { return false }

func (namedBlock) Define(*flow.Graph, flow.Node, flow.Env) (flow.Definition, error) {
	return flow.Definition{}, nil
}

func (namedBlock) Run(context.Context, *flow.Graph, flow.Node, flow.State, flow.Env) (flow.Result, error) {
	return flow.Result{}, nil
}

func catalog(t *testing.T, blocks []flow.Block, predicates map[string]flow.Predicate) *flow.Catalog {
	t.Helper()
	made, err := flow.NewCatalog(blocks, predicates)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func TestCatalogFindsWhatItHolds(t *testing.T) {
	always := func(flow.State, flow.Params) bool { return true }
	made := catalog(t, []flow.Block{namedBlock{"gate"}, namedBlock{"exit"}},
		map[string]flow.Predicate{"is_clean": always})

	block, ok := made.Block("gate")
	if !ok || block.Kind() != "gate" {
		t.Fatalf("Block(gate) = %v, %v", block, ok)
	}
	if _, ok := made.Block("command"); ok {
		t.Fatal("command is not registered")
	}
	if _, ok := made.Predicate("is_clean"); !ok {
		t.Fatal("is_clean is registered")
	}
	if _, ok := made.Predicate("nothing"); ok {
		t.Fatal("nothing is not registered")
	}
}

// Sorted, because both lists end up in a load message and a map's order does not.
func TestCatalogListsWhatItHoldsSorted(t *testing.T) {
	made := catalog(t, []flow.Block{namedBlock{"gate"}, namedBlock{"agent"}},
		map[string]flow.Predicate{"b": nil, "a": nil})
	if kinds := made.Kinds(); !slices.Equal(kinds, []string{"agent", "gate"}) {
		t.Fatalf("Kinds() = %v", kinds)
	}
	names := make([]string, 0, 2)
	for name := range made.Predicates() {
		names = append(names, name)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"a", "b"}) {
		t.Fatalf("Predicates() = %v", names)
	}
}

// A copy, so that a loader handing the map to expr.ParseCondition cannot
// widen what every later flow may name.
func TestPredicatesIsACopy(t *testing.T) {
	made := catalog(t, nil, map[string]flow.Predicate{"a": nil})
	made.Predicates()["b"] = nil
	if _, ok := made.Predicate("b"); ok {
		t.Fatal("the returned map is the catalog's own")
	}
}

func TestNewCatalogRefusesTwoBlocksOfOneKind(t *testing.T) {
	_, err := flow.NewCatalog([]flow.Block{namedBlock{"gate"}, namedBlock{"gate"}}, nil)
	if err == nil || err.Error() != `two blocks claim kind "gate"` {
		t.Fatalf("err = %v", err)
	}
}
```

`internal/flow/text_test.go`:

```go
package flow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/ultra-loom/internal/flow"
)

func TestReadTextTakesAFileNextToTheFlow(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "instructions"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "instructions", "draft.md")
	if err := os.WriteFile(path, []byte("Draft {{topic}}.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, err := flow.ReadText(dir, flow.Text{Key: "instruction", Path: "instructions/draft.md"})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "Draft {{topic}}.\n" {
		t.Fatalf("got %q", raw)
	}
}

func TestReadTextTakesAnInlineText(t *testing.T) {
	raw, err := flow.ReadText(t.TempDir(), flow.Text{Key: "message", Inline: "done in {{count}}"})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "done in {{count}}" {
		t.Fatalf("got %q", raw)
	}
}

func TestReadTextNamesTheKeyOfAMissingFile(t *testing.T) {
	_, err := flow.ReadText(t.TempDir(), flow.Text{Key: "question", Path: "ask.md"})
	if err == nil || !strings.HasPrefix(err.Error(), "reading question ask.md: ") {
		t.Fatalf("err = %v", err)
	}
}

func TestRenderSeesFieldsAndParams(t *testing.T) {
	state := flow.State{Fields: map[string]flow.Value{"notes": []string{"one", "two"}}}
	params := flow.Params{"max_rounds": 5}
	got, err := flow.Render([]byte("{{max_rounds}} rounds:\n{{notes}}"), state, params)
	if err != nil {
		t.Fatal(err)
	}
	if got != "5 rounds:\n- one\n- two" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderPassesOnWhatIsNoPlaceholder(t *testing.T) {
	_, err := flow.Render([]byte("{{ name }}"), flow.State{}, nil)
	if err == nil {
		t.Fatal("want an error")
	}
}
```

`internal/flow/graph_test.go`:

```go
package flow_test

import (
	"testing"

	"github.com/xidus90/ultra-loom/internal/flow"
)

func TestCapLimitAddsToItsParameter(t *testing.T) {
	limit, err := flow.Cap{Param: "max_rounds", Add: 1}.Limit(flow.Params{"max_rounds": 5})
	if err != nil {
		t.Fatal(err)
	}
	if limit != 6 {
		t.Fatalf("limit = %d, want 6", limit)
	}
}

func TestCapLimitWithoutAParameterIsItsNumber(t *testing.T) {
	limit, err := flow.Cap{Add: 3}.Limit(nil)
	if err != nil {
		t.Fatal(err)
	}
	if limit != 3 {
		t.Fatalf("limit = %d, want 3", limit)
	}
}

func TestStringKeyReadsOnlyAString(t *testing.T) {
	node := flow.Node{Keys: map[string]any{"instruction": "draft.md", "code": int64(4)}}
	if node.StringKey("instruction") != "draft.md" {
		t.Fatalf("got %q", node.StringKey("instruction"))
	}
	if node.StringKey("code") != "" || node.StringKey("nothing") != "" {
		t.Fatal("a key that is not a string reads as empty")
	}
}

func TestUnknownKeysAreSorted(t *testing.T) {
	node := flow.Node{Keys: map[string]any{"question": "q.md", "zzz": 1, "answer": "a", "aaa": 1}}
	got := node.UnknownKeys("question", "answer")
	if len(got) != 2 || got[0] != "aaa" || got[1] != "zzz" {
		t.Fatalf("got %v", got)
	}
	if node.UnknownKeys("question", "answer", "aaa", "zzz") != nil {
		t.Fatal("want nil when every key is known")
	}
}

// The loader checks both before a run; a hand-built graph does not, and the
// runner is the last place where an unusable ceiling can still be named.
func TestCapLimitRefusesAParameterItCannotUse(t *testing.T) {
	if _, err := (flow.Cap{Param: "rounds"}).Limit(nil); err == nil ||
		err.Error() != `max_visits names parameter "rounds", which the run does not have` {
		t.Fatalf("err = %v", err)
	}
	if _, err := (flow.Cap{Param: "rounds"}).Limit(flow.Params{"rounds": "five"}); err == nil ||
		err.Error() != `max_visits names parameter "rounds", which is not an int` {
		t.Fatalf("err = %v", err)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test ./internal/flow/`
Expected: FAIL — `undefined: flow.Coerce`, `undefined: flow.NewCatalog`, `undefined: flow.ReadText`, `c.Limit undefined`.

- [ ] **Step 3: Implement**

`internal/flow/coerce.go`:

```go
package flow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// Coerce turns a value from outside into the Go type its declared Type stands
// for. It is the only table of its kind: TOML hands out int64, a journal read
// with UseNumber hands out json.Number, and JSON hands out []any for a list.
// Written twice, the two copies would drift, and a flow would mean one thing
// when loaded and another when resumed.
func Coerce(t Type, raw any) (Value, error) {
	switch t {
	case String:
		text, ok := raw.(string)
		if !ok {
			return nil, notA(raw, t)
		}
		return text, nil
	case Int:
		return toInt(raw)
	case Bool:
		flag, ok := raw.(bool)
		if !ok {
			return nil, notA(raw, t)
		}
		return flag, nil
	case StringList:
		return toList(raw)
	default:
		return nil, fmt.Errorf("unknown type %q", string(t))
	}
}

func toInt(raw any) (Value, error) {
	switch value := raw.(type) {
	case int:
		return value, nil
	case int64:
		return int(value), nil
	case float64:
		if value != float64(int64(value)) {
			return nil, notWhole(raw)
		}
		return int(value), nil
	case json.Number:
		number, err := strconv.ParseInt(value.String(), 10, 64)
		if err == nil {
			return int(number), nil
		}
		// Out of an int64's range is not "not a whole number": 10^20 is one,
		// and calling it a fraction would send the reader looking for a dot.
		if errors.Is(err, strconv.ErrRange) {
			return nil, notA(raw, Int)
		}
		if _, floatErr := value.Float64(); floatErr == nil {
			return nil, notWhole(raw)
		}
		return nil, notA(raw, Int)
	default:
		return nil, notA(raw, Int)
	}
}

func toList(raw any) (Value, error) {
	switch value := raw.(type) {
	case []string:
		return value, nil
	case []any:
		list := make([]string, 0, len(value))
		for _, item := range value {
			text, err := Coerce(String, item)
			if err != nil {
				return nil, err
			}
			list = append(list, text.(string))
		}
		return list, nil
	default:
		return nil, notA(raw, StringList)
	}
}

func notA(raw any, t Type) error {
	return fmt.Errorf("cannot read %v as %s", raw, string(t))
}

func notWhole(raw any) error {
	return fmt.Errorf("cannot read %v as int; it is not a whole number", raw)
}
```

Note: Die `ErrRange`-Abzweigung ist der Grund, warum `errors` importiert wird. Ohne sie fiele `json.Number("99999999999999999999")` in die Float64-Prüfung, käme als „not a whole number" heraus, und der Test in Step 1 schlüge fehl.

`internal/flow/catalog.go`:

```go
package flow

import (
	"fmt"
	"slices"
)

// Catalog is the Registry every loader and every runner shares: the blocks and
// predicates this build knows. It takes what it holds at construction and has
// no way to add later, so nothing can widen a running flow's vocabulary.
type Catalog struct {
	blocks     map[string]Block
	predicates map[string]Predicate
}

// NewCatalog gathers blocks by their own Kind and refuses a kind twice: a
// second block under a known name would shadow the first silently, and which
// one wins would depend on the order somebody wrote a slice literal in.
func NewCatalog(blocks []Block, predicates map[string]Predicate) (*Catalog, error) {
	made := &Catalog{blocks: make(map[string]Block, len(blocks)), predicates: make(map[string]Predicate, len(predicates))}
	for _, block := range blocks {
		if _, taken := made.blocks[block.Kind()]; taken {
			return nil, fmt.Errorf("two blocks claim kind %q", block.Kind())
		}
		made.blocks[block.Kind()] = block
	}
	for name, predicate := range predicates {
		made.predicates[name] = predicate
	}
	return made, nil
}

// Block finds the block for a node kind.
func (c *Catalog) Block(kind string) (Block, bool) {
	block, ok := c.blocks[kind]
	return block, ok
}

// Predicate finds a predicate by name.
func (c *Catalog) Predicate(name string) (Predicate, bool) {
	predicate, ok := c.predicates[name]
	return predicate, ok
}

// Kinds are the registered node kinds, sorted, as a load message lists them.
func (c *Catalog) Kinds() []string {
	kinds := make([]string, 0, len(c.blocks))
	for kind := range c.blocks {
		kinds = append(kinds, kind)
	}
	slices.Sort(kinds)
	return kinds
}

// Predicates is what expr.ParseCondition wants: a copy, so that a caller
// writing into it cannot widen what every later flow may name.
func (c *Catalog) Predicates() map[string]Predicate {
	copied := make(map[string]Predicate, len(c.predicates))
	for name, predicate := range c.predicates {
		copied[name] = predicate
	}
	return copied
}
```

`internal/flow/text.go`:

```go
package flow

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xidus90/ultra-loom/internal/flow/tmpl"
)

// ReadText returns a text's raw bytes: the file next to the flow, or the text
// itself when it is inline. The bytes, not the rendered result, are what the
// definition hash covers -- a replay answers which wording produced a result.
func ReadText(dir string, t Text) ([]byte, error) {
	if t.Path == "" {
		return []byte(t.Inline), nil
	}
	raw, err := os.ReadFile(filepath.Join(dir, t.Path))
	if err != nil {
		return nil, fmt.Errorf("reading %s %s: %w", t.Key, t.Path, err)
	}
	return raw, nil
}

// Render fills a text's placeholders from the state and the run's parameters.
//
// One map for both: a name is declared under [params] or under [state] and
// never under both, which the loader enforces. Blocks render through here and
// nowhere else, so a gate's question and an agent's instruction cannot drift
// apart in what a placeholder means.
func Render(raw []byte, state State, params Params) (string, error) {
	template, err := tmpl.Parse(string(raw))
	if err != nil {
		return "", err
	}
	values := make(map[string]any, len(params)+len(state.Fields))
	for name, value := range params {
		values[name] = value
	}
	for name, value := range state.Fields {
		values[name] = value
	}
	return template.Render(values)
}
```

In `internal/flow/graph.go`, `Node` gains one field and `Cap` one method:

```go
// Node is one [[node]] entry of a flow file.
type Node struct {
	Name      string
	Kind      string
	Model     string // a name under [agent.models]; empty when the node names none
	MaxVisits Cap
	// Keys are the kind's own keys as TOML decoded them. The block for Kind
	// checks and reads them; nothing else does.
	Keys map[string]any
	// Raw is the whole entry as TOML decoded it, the shared keys included. The
	// definition hash is taken over this and not over the struct: the struct
	// has no JSON tags, so its Go field names would land in the hash, and
	// MaxVisits would arrive parsed rather than as the file wrote it.
	Raw map[string]any
}

// StringKey is the node's own key `name` when it holds a string, and "" when
// it holds anything else or nothing. Three blocks and the loader read keys this
// way; written once here, none of them has to guess what TOML handed over.
func (n Node) StringKey(name string) string {
	text, _ := n.Keys[name].(string)
	return text
}

// UnknownKeys are the node's own keys that are not in known, sorted. A block
// calls it in Check, so `instructon` is a load finding and not an instruction
// nobody reads.
func (n Node) UnknownKeys(known ...string) []string {
	var unknown []string
	for key := range n.Keys {
		if !slices.Contains(known, key) {
			unknown = append(unknown, key)
		}
	}
	slices.Sort(unknown)
	return unknown
}

// Limit is the ceiling this Cap stands for, given a run's parameters. The
// loader has checked the parameter exists and is an int; a hand-built graph
// has not, and this is the last place where an unusable ceiling can be named
// rather than panicked over.
func (c Cap) Limit(params Params) (int, error) {
	if c.Param == "" {
		return c.Add, nil
	}
	raw, ok := params[c.Param]
	if !ok {
		return 0, fmt.Errorf("max_visits names parameter %q, which the run does not have", c.Param)
	}
	number, ok := raw.(int)
	if !ok {
		return 0, fmt.Errorf("max_visits names parameter %q, which is not an int", c.Param)
	}
	return number + c.Add, nil
}
```

In `internal/flow/doc.go`, the last sentence loses its promise and names the package:

```
// package. The loader that fills a Graph from TOML lives in internal/flowload,
// because it parses conditions with internal/flow/expr, which imports this
// package.
```

In `internal/journal/journal.go`, the second pass of `decode` reads with `UseNumber`:

```go
	decoder := json.NewDecoder(strings.NewReader(line))
	// A delta's int field is an int, not a float64 that happens to look like
	// one. Read the ordinary way, 2^53+1 comes back as 2^53, and a resume
	// would hash a payload the run never had. Canonical already reads this
	// way, and json.Number marshals back as the literal that stood there, so
	// the writer is unchanged.
	decoder.UseNumber()
	var entry Entry
	if err := decoder.Decode(&entry); err != nil {
		return Entry{}, err
	}
	return entry, nil
```

- [ ] **Step 4: Pin the new test data to LF**

`git config core.autocrlf` steht auf diesem Rechner so, dass git beim Auschecken CRLF schreibt — der Commit der Welle-1-Pläne meldete „LF will be replaced by CRLF". Die Testdaten der Wellen 2 und 3 werden byteweise verglichen: eine gerenderte Frage, eine Anweisung mit ihrem Zeilenumbruch am Ende, später das Golden-Journal. `.gitattributes` pinnt heute `*.go` und die Render-Testdaten; zwei Zeilen kommen dazu:

```
internal/blocks/testdata/**       text eol=lf
internal/flowload/testdata/**     text eol=lf
```

- [ ] **Step 5: Fix the two journal tests the change touches**

`internal/journal/journal_test.go:48` and `internal/journal/write_test.go:62` compare a delta value against `float64(1)` and `float64(2)`. Both become `json.Number`:

```go
	if got[0].Delta["n"] != json.Number("1") {
```

```go
	if *got[1].Model != "claude:claude-opus-5" || got[1].Delta["n"] != json.Number("2") {
```

Add a test that says why, in `internal/journal/journal_test.go`:

```go
// The reason Entries reads with UseNumber: this number has no float64, and a
// resume that hashed the rounded value would look for a payload no run had.
func TestEntriesKeepsAnIntTooLargeForAFloat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0001.jsonl")
	line := `{"node":"n","kind":"agent","input_hash":"h","delta":{"big":9007199254740993},` +
		`"outcome":"ok","tools":null,"effort":null,"tokens":0,"seconds":0,"detail":null}`
	if err := os.WriteFile(path, []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := journal.Entries(path)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Delta["big"] != json.Number("9007199254740993") {
		t.Fatalf("delta = %#v", got[0].Delta)
	}
}
```

- [ ] **Step 6: Run the tests with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -cover ./internal/flow/ ./internal/journal/`
Expected: beide `ok … coverage: 100.0% of statements`.

- [ ] **Step 7: Run the whole suite**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test ./...`
Expected: alles grün, `cmd/guard` eingeschlossen — es liest dasselbe Journal.

- [ ] **Step 8: Update the spec's package table**

In `docs/.superpowers/specs/2026-09-11-ulflow-laufzeit-design.md`, Abschnitt „Pakete", direkt unter der Zeile für `internal/flow`, eine Zeile einfügen und die Zeile für `internal/flow` kürzen:

```
| `internal/flow` | Verträge (Typen, Zustand, Graph, `Condition`, `Block`, `Registry` und `Catalog`), `Coerce`, `ReadText`/`Render` |
| `internal/flowload` | Format lesen, Prüfung beim Laden, Auffindung |
```

Im selben Abschnitt den Satz „Platzhalter, Bausteine und Laufdateien haben eigene Pakete…" unverändert lassen. In der Tabelle der Welle 2 unter „Wellen" `internal/flow (außer den Verträgen)` durch `internal/flowload` ersetzen.

- [ ] **Step 9: gofmt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && gofmt -l internal/flow internal/journal`
Expected: keine Ausgabe.

- [ ] **Step 10: Commit**

Nachricht in eine Datei schreiben, dann:

```
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && git add internal/flow internal/journal .gitattributes docs/.superpowers/specs/2026-09-11-ulflow-laufzeit-design.md && git commit -F <message-file>
```

Betreff: `Give wave 2 the contracts its four lanes would have to share`

---

### Task 1: Lane `loader` — `internal/flowload`

**Worktree:** `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-loader`, Zweig `ulflow/loader`

**Files:**
- Create: `internal/flowload/doc.go`, `internal/flowload/load.go`, `internal/flowload/decl.go`, `internal/flowload/check.go`, `internal/flowload/discover.go`
- Test: `internal/flowload/load_test.go`, `internal/flowload/check_test.go`, `internal/flowload/discover_test.go`, `internal/flowload/discover_internal_test.go`
- Create: `internal/flowload/testdata/…` (die Flows unten, je in einem eigenen Verzeichnis)

**Interfaces:**
- Consumes: `flow.Graph`, `flow.Node`, `flow.Edge`, `flow.Cap`, `flow.Field`, `flow.Type`, `flow.Text`, `flow.Block`, `flow.Coerce`, `flow.ReadText`, `flow.Catalog` (Task 0); `expr.ParseCondition`, `expr.ParseCap`, `expr.Names`; `tmpl.Parse`, `Template.Names`; `flowcfg.Config.Resolve`; `github.com/BurntSushi/toml`. **Nicht** `internal/blocks`.
- Produces:
  - `const flowload.SchemaVersion = 1`
  - `const flowload.Dir = ".ultraloom/flows"`
  - `type flowload.Findings []string` mit `func (f Findings) Error() string` — eine Zeile je Befund
  - `func flowload.Load(path string, catalog *flow.Catalog, agent flowcfg.Config) (*flow.Graph, error)`
  - `type flowload.Entry struct{ Name, Origin, Problem string }`
  - `func flowload.List(root string, catalog *flow.Catalog, agent flowcfg.Config) []Entry`
  - `func flowload.Find(name, root string) (string, error)`

**Die Lane besitzt kein anderes Paket, und sie importiert `internal/blocks` nicht** — das Paket entsteht gerade in zwei anderen Lanes. Die Tests dieser Lane registrieren **eigene Bausteine** im `Catalog`, die sich verhalten wie die echten: `agent` liest `instruction` und schreibt die Felder aus `reply`, `gate` liest `question` und schreibt `answer` und `<answer>_text`, `exit` hat ein inline `message` und ein `Check`, das einen Befund liefert. Das ist Absicht: Diese Lane prüft, dass der Lader jeden Baustein gleich behandelt, nicht was ein bestimmter Baustein von seinen Schlüsseln hält. Lader und echte Bausteine treffen sich in Task 5 und in Welle 3.

#### Was `Load` tut

Sieben Stufen in dieser Reihenfolge. **Jede Stufe meldet alle ihre Befunde, und nach der ersten Stufe mit Befunden hört `Load` auf** — eine Stufe baut auf dem auf, was die vorige für heil erklärt hat. Der Rückgabewert ist dann `Findings`, sonst der gefüllte `*flow.Graph` und `nil`.

Jeder Befund beginnt mit dem Dateinamen, wie die Spec ihn zeigt:

```
example.toml: edge draft→approve reads "verdit"; known fields: answer, count, notes, verdict
```

| Stufe | prüft | Beispielbefund (wörtlich) |
|---|---|---|
| 1 | TOML lesbar, `schema_version`, Pflichtschlüssel, Namen, reservierte Namen, Doppeldeklarationen, Typen und Vorgaben, Kantenschlüssel, endliche Zahlen | `x.toml: schema_version 2 is unknown; ulflow knows version 1` |
| 2 | Knotenart registriert, `command` mit eigener Meldung, danach `Block.Check` je Knoten | `x.toml: node "run" has kind "command"; command nodes are planned but not built` |
| 3 | jede Datei aus `Block.Texts` existiert | `x.toml: node "draft": reading instruction instructions/draft.md: open …: The system cannot find the file specified.` |
| 4 | Platzhalter, Bedingungen, Deckel | `x.toml: edge draft→approve: reads "verdit"; known fields: verdict` |
| 5 | geschriebene Felder gegen `[state]` und gegeneinander | `x.toml: node "draft" writes "verdict" as int, but [state] declares it as string` |
| 6 | die fünf Graphregeln | `x.toml: node "draft" sits on a cycle but allows one visit; raise its max_visits` |
| 7 | Modellkette, drei getrennte Auflösungen | `x.toml: node "draft": model "wrter" is not under [agent.models]; known models: writer` |

**Stufe 1 im Einzelnen.** Ein `toml.Decode` in `map[string]any`, nicht in ein Struct: `github.com/BurntSushi/toml` verschluckt `agent = 3` in einem Struct-Feld ohne Fehler (in Welle 1 gemessen). Geprüft wird, in dieser Reihenfolge, aber alles gesammelt:

- `schema_version` fehlt oder ist nicht 1.
- `[flow]` fehlt, `name` fehlt oder ist kein Name, `start` fehlt.
- Jeder `[[node]]` hat `name` und `kind`; kein Name zweimal.
- Kein Knoten-, Feld- oder Parametername ist `true`, `false` oder `END`, und jeder passt auf `[A-Za-z_][A-Za-z0-9_]*` (Entscheidungen 4 und 11).
- Kein Name steht in `[params]` und `[state]` zugleich (Entscheidung 3).
- Jedes Feld und jeder Parameter hat einen bekannten `type`, und `default` geht durch `flow.Coerce`. Fehlt `default`, ist das ein Befund: die Spec verlangt für jedes Feld eine Vorgabe.
- Jeder `[[edge]]` hat `from` und `to`. `when` zusammen mit `on_error = true` ist ein Befund; `when = ""` ebenso (Entscheidung 10).
- Kein Wert in einem Knoteneintrag ist `nan` oder `inf`. TOML kennt beide, `journal.Canonical` kann sie nicht schreiben, und der Fehler fiele sonst erst nach dem ersten Modellaufruf an (Entscheidung 2).

Was Stufe 1 überlebt, füllt `flow.Node`: `Name`, `Kind`, `Model`, `Keys` (alles außer `name`, `kind`, `model`, `max_visits`) und `Raw` (der ganze Eintrag, wie TOML ihn decodiert hat). `MaxVisits` füllt erst Stufe 4, weil `ParseCap` die Parametertypen braucht; bis dahin steht `Cap{Add: 1}` darin.

**Stufe 4 im Einzelnen.** Platzhalter werden nur auf Existenz geprüft: alle vier Typen rendert `tmpl` (Listen als `- `-Zeilen), ein Typfehler ist dort nicht möglich. Bedingungen und Deckel prüfen ihre Typen selbst, über `expr.Names{Fields: …, Params: …, Predicates: catalog.Predicates()}`.

**Stufe 6 hält auch in sich an.** Die fünf Graphregeln bauen aufeinander auf, wie in `graph.py:166–191`: Wenn Regel 1 oder 2 etwas findet, laufen 3 bis 5 nicht mehr. Sonst meldete ein `start = "nowhere"` auch noch jeden Knoten als unerreichbar, und der Leser suchte drei Fehler statt einem. Regeln 3, 4 und 5 laufen zusammen, weil sie einander nicht widersprechen — daher hat `stage6_island` zwei Befunde und `stage6_graph` einen.

**Stufe 6, Regel 5 mit einem Deckel über einen Parameter.** `max_visits = "max_rounds + 1"` ist beim Laden nicht auszurechnen. Ein `Cap` mit gesetztem `Param` gilt hier als „mehr als ein Besuch erlaubt"; ob der Parameter zur Laufzeit 0 oder weniger ergibt, prüft der Runner beim Start (Entscheidung 9). Ein `Cap` ohne `Param` und mit `Add <= 1` auf einem Zyklus ist der Befund.

**Auffindung.** `Find` prüft den Namen gegen dieselbe Namensregel und sucht `.ultraloom/flows/<name>.toml`. `List` liest das Verzeichnis, versucht jede Datei zu laden und trägt den Grund in `Problem` ein, statt sie wegzulassen. Mitgelieferte Flows gibt es in M1 keine (Entscheidung 19); die Verdeckungsregel steht als unexportiertes `merge(project, bundled []Entry) []Entry` da und wird direkt getestet.

- [ ] **Step 1: Write the failing tests for stage 1 and the happy path**

`internal/flowload/load_test.go`:

```go
package flowload_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/ultra-loom/internal/flow"
	"github.com/xidus90/ultra-loom/internal/flowcfg"
	"github.com/xidus90/ultra-loom/internal/flowload"
)

// The blocks this lane tests against are its own. internal/blocks is being
// written in two other lanes right now, and a loader that could only be tested
// with the real ones would be a loader that waits for them. These behave like
// the real ones in the three things the loader asks of a block: which of its
// own keys it objects to, which texts a node renders, and which fields it
// writes.
type testBlock struct{ kind string }

func (b testBlock) Kind() string { return b.kind }

func (b testBlock) Check(node flow.Node) []string {
	if b.kind != "exit" {
		return nil
	}
	code, ok := node.Keys["code"].(int64)
	if ok && (code == 0 || code == 1 || code == 3) {
		return []string{fmt.Sprintf("code %d is the runtime's own; a block may not choose 0, 1 or 3", code)}
	}
	return nil
}

func (b testBlock) Texts(node flow.Node) []flow.Text {
	switch b.kind {
	case "agent":
		return []flow.Text{{Key: "instruction", Path: node.StringKey("instruction")}}
	case "gate":
		return []flow.Text{{Key: "question", Path: node.StringKey("question")}}
	default:
		return []flow.Text{{Key: "message", Inline: node.StringKey("message")}}
	}
}

func (b testBlock) Writes(node flow.Node) map[string]flow.Type {
	switch b.kind {
	case "agent":
		reply, _ := node.Keys["reply"].(map[string]any)
		written := make(map[string]flow.Type, len(reply))
		for name, kind := range reply {
			text, _ := kind.(string)
			written[name] = flow.Type(text)
		}
		return written
	case "gate":
		answer := node.StringKey("answer")
		return map[string]flow.Type{answer: flow.String, answer + "_text": flow.String}
	default:
		return nil
	}
}

func (testBlock) NeedsBaseline() bool { return false }

func (testBlock) Define(*flow.Graph, flow.Node, flow.Env) (flow.Definition, error) {
	return flow.Definition{}, nil
}

func (testBlock) Run(context.Context, *flow.Graph, flow.Node, flow.State, flow.Env) (flow.Result, error) {
	return flow.Result{}, nil
}

func catalog(t *testing.T) *flow.Catalog {
	t.Helper()
	made, err := flow.NewCatalog(
		[]flow.Block{testBlock{"agent"}, testBlock{"gate"}, testBlock{"exit"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func load(t *testing.T, dir, file string, agent flowcfg.Config) (*flow.Graph, error) {
	t.Helper()
	return flowload.Load(filepath.Join("testdata", dir, file), catalog(t), agent)
}

func TestTheLoadedGraphSaysWhatTheFileSaid(t *testing.T) {
	graph, err := load(t, "planning", "planning.toml", flowcfg.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if graph.Name != "planning" || graph.Start != "intake" {
		t.Fatalf("name = %q, start = %q", graph.Name, graph.Start)
	}
	if len(graph.Nodes) != 12 {
		t.Fatalf("%d nodes", len(graph.Nodes))
	}
	// File order, because the first edge whose condition holds is the one a run
	// takes, and a map would make that order the runtime's business.
	if graph.Nodes[0].Name != "intake" || graph.Edges[0].From != "intake" {
		t.Fatalf("order lost: %q, %q", graph.Nodes[0].Name, graph.Edges[0].From)
	}
	if graph.Dir != filepath.Join("testdata", "planning") {
		t.Fatalf("dir = %q", graph.Dir)
	}
}

// Raw is what the definition hash is taken over, so it holds the entry as the
// file wrote it -- max_visits as its text, not as the parsed Cap.
func TestANodeKeepsItsRawEntry(t *testing.T) {
	graph, err := load(t, "planning", "planning.toml", flowcfg.Config{})
	if err != nil {
		t.Fatal(err)
	}
	node := graph.Nodes[1] // clarify
	if node.Raw["max_visits"] != "max_rounds + 1" {
		t.Fatalf("raw max_visits = %#v", node.Raw["max_visits"])
	}
	if _, ok := node.Keys["name"]; ok {
		t.Fatal("Keys holds a shared key")
	}
	if node.MaxVisits != (flow.Cap{Param: "max_rounds", Add: 1}) {
		t.Fatalf("cap = %#v", node.MaxVisits)
	}
}

func TestDefaultsArriveInTheDeclaredGoType(t *testing.T) {
	graph, err := load(t, "planning", "planning.toml", flowcfg.Config{})
	if err != nil {
		t.Fatal(err)
	}
	// TOML hands out int64; a state field declared int holds an int.
	if graph.Params["max_rounds"].Default != 3 {
		t.Fatalf("max_rounds = %#v", graph.Params["max_rounds"].Default)
	}
	if list, ok := graph.State["notes"].Default.([]string); !ok || len(list) != 0 {
		t.Fatalf("notes = %#v", graph.State["notes"].Default)
	}
}
```

- [ ] **Step 2: Write the stage fixtures and their test**

Jede Stufe bekommt ein Verzeichnis unter `internal/flowload/testdata/` mit genau den Dateien, die sie braucht. Die Tabelle ist vollständig; die Dateien sind so klein, wie die Stufe es erlaubt.

| Verzeichnis | Datei | was darin falsch ist | erwartete Befunde (wörtlich, ohne das führende `<datei>: `) |
|---|---|---|---|
| `stage1_version` | `f.toml` | `schema_version = 2` | `schema_version 2 is unknown; ulflow knows version 1` |
| `stage1_names` | `f.toml` | `[[node]] name = "END"`, ein Feld `max-rounds`, ein Name in `[params]` und `[state]`, ein `default`, das nicht zum `type` passt — **und** an einem Knoten `max_visits = "rounds + 1"` ohne Parameter `rounds`, damit `TestLoadStopsAfterTheFirstStageWithFindings` etwas zu beweisen hat | `"END" is reserved and cannot be a node name`<br>`field "max-rounds" is not a name; a name is [A-Za-z_][A-Za-z0-9_]*`<br>`"count" is declared under [params] and under [state]`<br>`field "count" default: cannot read x as int` |
| `stage1_edges` | `f.toml` | eine Fehlerkante mit `when`, eine Kante mit `when = ""`, eine Kante ohne `to` | `the error edge draft→fix cannot carry a when; an error edge is unconditional`<br>`edge draft→approve has an empty when`<br>`edge 3 has no to` |
| `stage2_kinds` | `f.toml` | ein Knoten `kind = "command"`, einer `kind = "widget"`, ein `exit` mit `code = 3` (der Befund kommt aus dem `Check` des Testbausteins — die Stufe soll zeigen, dass sie ihn weiterreicht, nicht was der echte `exit` von Codes hält) | `node "run" has kind "command"; command nodes are planned but not built`<br>`node "w" has kind "widget"; known kinds: agent, exit, gate`<br>`node "stop": code 3 is the runtime's own; a block may not choose 0, 1 or 3` |
| `stage3_texts` | `f.toml` | ein `agent`-Knoten, dessen `instruction` es nicht gibt | beginnt mit `node "draft": reading instruction instructions/draft.md: ` |
| `stage4_names` | `f.toml`, `instructions/draft.md` | Platzhalter `{{verdit}}` in der Anweisung, `when = "verdit == \"done\""`, `max_visits = "rounds + 1"` ohne Parameter `rounds`. Der Fehlertext für die Bedingung ist der von `comparisonTerm` (`condition.go:219`), nicht der von `predicateTerm`: `verdit == "done"` sind drei Token und damit ein Vergleich. `known fields` sind nur die Zustandsfelder; Parameter stehen in einer eigenen Tabelle | `node "draft" instruction names "verdit"; known fields: verdict`<br>`edge draft→approve: reads "verdit"; known fields: verdict`<br>`node "draft": max_visits "rounds + 1" names no parameter; known parameters: max_rounds` |
| `stage5_writes` | `f.toml`, `instructions/draft.md`, `questions/ask.md` | ein `agent` mit `reply = { verdict = "int" }`, während `[state] verdict` ein `string` ist; ein Tor mit `answer = "choice"`, wobei `[state]` **`choice_text`** deklariert und `choice` nicht — so bleibt es bei einem Befund statt zweien, denn ein Tor schreibt beide Felder | `node "draft" writes "verdict" as int, but [state] declares it as string`<br>`node "approve" writes "choice", which [state] does not declare` |
| `stage6_graph` | `f.toml` | `start = "nowhere"` | `start node "nowhere" does not exist` |
| `stage6_cycle` | `f.toml` | zwei `exit`-Knoten in einem Zyklus, beide ohne `max_visits` | `node "a" sits on a cycle but allows one visit; raise its max_visits`<br>`node "b" sits on a cycle but allows one visit; raise its max_visits` |
| `stage6_island` | `f.toml` | ein Knoten ohne Weg vom Start, ein Knoten ohne ausgehende Kante | `unreachable node(s): island`<br>`node "stop" has no outgoing edge` |
| `stage7_model` | `f.toml`, `instructions/draft.md` | `model = "wrter"` am Knoten | `node "draft": model "wrter" is not under [agent.models]; known models: writer` |

Der Test dazu, in `internal/flowload/check_test.go`:

```go
package flowload_test

import (
	"strings"
	"testing"

	"github.com/xidus90/ultra-loom/internal/flowcfg"
)

func TestEachStageHasAFlowThatFailsInIt(t *testing.T) {
	writer := flowcfg.Config{Models: map[string]flowcfg.ModelSpec{
		"writer": {Provider: "claude", Model: "claude-opus-5"},
	}}
	cases := []struct {
		dir   string
		agent flowcfg.Config
		want  []string
	}{
		{dir: "stage1_version", want: []string{
			"schema_version 2 is unknown; ulflow knows version 1",
		}},
		// Je eine solche Zeile für stage1_names, stage1_edges, stage2_kinds,
		// stage3_texts, stage4_names, stage5_writes, stage6_graph,
		// stage6_cycle und stage6_island, mit genau den Befunden, die die
		// Tabelle oben für sie nennt, in deren Reihenfolge.
		{dir: "stage7_model", agent: writer, want: []string{
			`node "draft": model "wrter" is not under [agent.models]; known models: writer`,
		}},
	}
	for _, c := range cases {
		t.Run(c.dir, func(t *testing.T) {
			_, err := load(t, c.dir, "f.toml", c.agent)
			if err == nil {
				t.Fatal("want findings")
			}
			lines := strings.Split(err.Error(), "\n")
			if len(lines) != len(c.want) {
				t.Fatalf("got %d findings, want %d:\n%s", len(lines), len(c.want), err)
			}
			for i, want := range c.want {
				if !strings.HasSuffix(lines[i], want) || !strings.HasPrefix(lines[i], "f.toml: ") {
					t.Fatalf("finding %d is %q, want %q behind the file name", i, lines[i], want)
				}
			}
		})
	}
}

// The point of collecting: a file with three mistakes in one stage is read once.
func TestAStageReportsEveryFindingItHas(t *testing.T) {
	_, err := load(t, "stage1_edges", "f.toml", flowcfg.Config{})
	if err == nil || len(strings.Split(err.Error(), "\n")) != 3 {
		t.Fatalf("err = %v", err)
	}
}

// And the point of stopping: stage 4 reads names stage 1 has not cleared yet,
// so a file that fails in stage 1 is not also reported against later stages.
func TestLoadStopsAfterTheFirstStageWithFindings(t *testing.T) {
	_, err := load(t, "stage1_names", "f.toml", flowcfg.Config{})
	if err == nil {
		t.Fatal("want findings")
	}
	if strings.Contains(err.Error(), "max_visits") {
		t.Fatalf("a later stage ran anyway:\n%s", err)
	}
}
```

- [ ] **Step 3: Write the discovery tests**

`internal/flowload/discover_test.go`:

```go
package flowload_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/ultra-loom/internal/flowcfg"
	"github.com/xidus90/ultra-loom/internal/flowload"
)

func project(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, filepath.FromSlash(flowload.Dir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestListNamesEveryFlowAndWhyOneWillNotLoad(t *testing.T) {
	root := project(t, map[string]string{
		"good.toml": mustRead(t, filepath.Join("testdata", "minimal", "f.toml")),
		"bad.toml":  "schema_version = 2\n",
		"1st.toml":  "",
	})
	got := flowload.List(root, catalog(t), flowcfg.Config{})
	if len(got) != 3 {
		t.Fatalf("%d entries: %+v", len(got), got)
	}
	// Sorted by name, so a list is the same list twice running.
	if got[0].Name != "1st" || got[1].Name != "bad" || got[2].Name != "good" {
		t.Fatalf("order: %+v", got)
	}
	if got[0].Problem == "" || got[1].Problem == "" {
		t.Fatalf("a flow that cannot be loaded needs its reason: %+v", got)
	}
	if got[2].Problem != "" || got[2].Origin != "project" {
		t.Fatalf("good: %+v", got[2])
	}
}

func TestListOfADirectoryThatIsNotThereIsEmpty(t *testing.T) {
	if got := flowload.List(t.TempDir(), catalog(t), flowcfg.Config{}); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestFindNamesThePathOfAFlow(t *testing.T) {
	root := project(t, map[string]string{"good.toml": "schema_version = 1\n"})
	path, err := flowload.Find("good", root)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(root, filepath.FromSlash(flowload.Dir), "good.toml") {
		t.Fatalf("path = %q", path)
	}
}

// The name reaches Find from the command line and from a run's marker, and it
// is interpolated into a path. "../../evil" is exactly what a name is not.
func TestFindRefusesWhatIsNoName(t *testing.T) {
	for _, name := range []string{"../../evil", "with space", "1st", ""} {
		if _, err := flowload.Find(name, t.TempDir()); err == nil {
			t.Fatalf("%q was accepted", name)
		}
	}
}

func TestFindSaysWhichFlowsThereAreWhenItFindsNone(t *testing.T) {
	root := project(t, map[string]string{"good.toml": "schema_version = 1\n"})
	_, err := flowload.Find("other", root)
	if err == nil || err.Error() != `no flow named "other"; known flows: good` {
		t.Fatalf("err = %v", err)
	}
}
```

`internal/flowload/discover_internal_test.go`:

```go
package flowload

import "testing"

// M1 bundles no flow, so the rule that a project flow hides a bundled one of
// the same name has nothing to act on yet. It is tested here rather than left
// until the MVP, because the MVP is when it first matters and the worst time
// to discover it was never written.
func TestAProjectFlowHidesABundledOneOfTheSameName(t *testing.T) {
	merged := merge(
		[]Entry{{Name: "review", Origin: "project"}},
		[]Entry{{Name: "review", Origin: "bundled"}, {Name: "ship", Origin: "bundled"}},
	)
	if len(merged) != 2 {
		t.Fatalf("%+v", merged)
	}
	if merged[0].Name != "review" || merged[0].Origin != "project" {
		t.Fatalf("the project flow lost: %+v", merged[0])
	}
	if merged[1].Name != "ship" || merged[1].Origin != "bundled" {
		t.Fatalf("%+v", merged[1])
	}
}

func TestBundledIsEmptyInM1(t *testing.T) {
	if got := bundled(); len(got) != 0 {
		t.Fatalf("M1 ships no flow, got %+v", got)
	}
}
```

- [ ] **Step 4: Run the tests to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-loader" && go test ./internal/flowload/...`
Expected: FAIL — `undefined: flowload.Load`.

- [ ] **Step 5: Write the planning flow**

`internal/flowload/testdata/planning/planning.toml` — die Topologie des Abschnitts „Planen" aus `docs/.superpowers/specs/2026-09-10-agent-harness-graph.md`, mit den drei Knotenarten, die M1 hat: die Codeknoten `intake` und `board` und der Prüferfächer werden `agent`-Knoten, die drei MENSCH-Tore werden `gate`-Knoten.

```toml
schema_version = 1

[flow]
name  = "planning"
start = "intake"

[params]
max_rounds = { type = "int", default = 3 }

[state]
verdict     = { type = "string",       default = "" }
answer      = { type = "string",       default = "" }
answer_text = { type = "string",       default = "" }
notes       = { type = "list[string]", default = [] }

[[node]]
name        = "intake"
kind        = "agent"
instruction = "instructions/intake.md"
tools       = "edit"
reply       = { verdict = "string" }

[[node]]
name        = "clarify"
kind        = "agent"
instruction = "instructions/clarify.md"
tools       = "read_only"
max_visits  = "max_rounds + 1"
reply       = { verdict = "string" }

[[node]]
name       = "gate_clarify"
kind       = "gate"
question   = "questions/clarify.md"
choices    = ["yes", "no"]
answer     = "answer"
max_visits = "max_rounds + 1"

[[node]]
name        = "spec"
kind        = "agent"
instruction = "instructions/spec.md"
tools       = "edit"
max_visits  = "max_rounds + 1"
reply       = { verdict = "string" }

[[node]]
name        = "check_spec"
kind        = "agent"
instruction = "instructions/check-spec.md"
tools       = "read_only"
max_visits  = "max_rounds + 1"
reply       = { verdict = "string" }

[[node]]
name       = "gate_spec"
kind       = "gate"
question   = "questions/spec.md"
choices    = ["yes", "no"]
answer     = "answer"
max_visits = "max_rounds + 1"

[[node]]
name        = "tagger"
kind        = "agent"
instruction = "instructions/tagger.md"
tools       = "read_only"
reply       = { verdict = "string" }

[[node]]
name        = "plan"
kind        = "agent"
instruction = "instructions/plan.md"
tools       = "edit"
max_visits  = "max_rounds + 1"
reply       = { verdict = "string" }

[[node]]
name        = "check_plan"
kind        = "agent"
instruction = "instructions/check-plan.md"
tools       = "read_only"
max_visits  = "max_rounds + 1"
reply       = { verdict = "string" }

[[node]]
name       = "gate_plan"
kind       = "gate"
question   = "questions/plan.md"
choices    = ["yes", "no"]
answer     = "answer"
max_visits = "max_rounds + 1"

[[node]]
name        = "clarify_plan"
kind        = "agent"
instruction = "instructions/clarify-plan.md"
tools       = "read_only"
max_visits  = "max_rounds + 1"
reply       = { verdict = "string" }

[[node]]
name        = "board"
kind        = "agent"
instruction = "instructions/board.md"
tools       = "edit"
reply       = { verdict = "string" }

[[edge]]
from = "intake"
to   = "clarify"

[[edge]]
from = "clarify"
to   = "gate_clarify"

[[edge]]
from = "gate_clarify"
to   = "spec"
when = "answer == \"yes\""

[[edge]]
from = "gate_clarify"
to   = "clarify"

[[edge]]
from = "spec"
to   = "check_spec"

[[edge]]
from = "check_spec"
to   = "clarify"
when = "verdict == \"open\""

[[edge]]
from = "check_spec"
to   = "gate_spec"
when = "verdict == \"clean\""

[[edge]]
from = "check_spec"
to   = "spec"

[[edge]]
from = "gate_spec"
to   = "tagger"
when = "answer == \"yes\""

[[edge]]
from = "gate_spec"
to   = "clarify"

[[edge]]
from = "tagger"
to   = "plan"

[[edge]]
from = "plan"
to   = "check_plan"

[[edge]]
from = "check_plan"
to   = "clarify_plan"
when = "verdict == \"open\""

[[edge]]
from = "check_plan"
to   = "gate_plan"
when = "verdict == \"clean\""

[[edge]]
from = "check_plan"
to   = "plan"

[[edge]]
from = "gate_plan"
to   = "board"
when = "answer == \"yes\""

[[edge]]
from = "gate_plan"
to   = "clarify_plan"

[[edge]]
from = "clarify_plan"
to   = "plan"

[[edge]]
from = "board"
to   = "END"
```

Dazu neun einzeilige Anweisungen unter `instructions/` (`intake.md`, `clarify.md`, `spec.md`, `check-spec.md`, `tagger.md`, `plan.md`, `check-plan.md`, `clarify-plan.md`, `board.md`) und drei Fragen unter `questions/` (`clarify.md`, `spec.md`, `plan.md`). Je eine Zeile, englisch, und mindestens eine davon mit einem Platzhalter, damit Stufe 4 auf dem guten Flow etwas zu tun hat — zum Beispiel `instructions/clarify.md`:

```
Clarify the request. You have {{max_rounds}} rounds and these notes:
{{notes}}
```

Dazu `internal/flowload/testdata/minimal/f.toml`, den `discover_test.go` kopiert: der kleinste Flow, der lädt — ein `exit`-Knoten mit einer Kante nach `END`.

- [ ] **Step 6: Implement**

Vier Dateien, in dieser Reihenfolge zu schreiben, weil jede die vorige braucht:

`decl.go` — Stufe 1. `func declarations(file string, doc map[string]any) (*flow.Graph, Findings)`. Baut den Graphen so weit, wie Stufe 1 ihn verantworten kann: `Name`, `Start`, `Model`, `Params`, `State`, `Nodes` mit `Raw` und `Keys`, `Edges` mit `From`, `To`, `OnError` und dem rohen `when`-Text. Die Kantenbedingung bleibt bis Stufe 4 `nil`.

`check.go` — Stufen 2 bis 7, je eine Funktion mit der Signatur `func stageN(file string, graph *flow.Graph, catalog *flow.Catalog, agent flowcfg.Config) Findings`. Stufe 4 füllt nebenbei `Edge.When` und `Node.MaxVisits`; das ist der einzige Ort, an dem eine Prüfung schreibt, und der Kommentar sagt es.

`load.go` — `Load` liest die Datei, decodiert nach `map[string]any`, ruft die Stufen der Reihe nach und hört nach der ersten mit Befunden auf. `Findings.Error` verbindet mit `"\n"`.

`discover.go` — `Dir`, `Find`, `List`, `merge`, `bundled`.

`doc.go` — ein Absatz: was das Paket tut, und der Satz, warum es nicht in `internal/flow` liegt (es parst Bedingungen mit `internal/flow/expr`, und das importiert `internal/flow`).

- [ ] **Step 7: Run the tests with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-loader" && go test -cover ./internal/flowload/...`
Expected: `ok … coverage: 100.0% of statements`.

- [ ] **Step 8: gofmt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-loader" && gofmt -l internal/flowload`
Expected: keine Ausgabe.

- [ ] **Step 9: Commit**

```
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-loader" && git add internal/flowload && git commit -F <message-file>
```

Betreff: `Load a flow from TOML and refuse it in seven stages`

---

### Task 2: Lane `gate-exit` — die Bausteine `gate` und `exit`

**Worktree:** `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-gate-exit`, Zweig `ulflow/gate-exit`

**Files:**
- Create: `internal/blocks/gate.go`, `internal/blocks/exit.go`
- Test: `internal/blocks/gate_test.go`, `internal/blocks/exit_test.go`
- Create: `internal/blocks/testdata/questions/approve.md`, `internal/blocks/testdata/broken/question.md`

**Interfaces:**
- Consumes: `flow.Block`, `flow.Node`, `flow.Text`, `flow.Definition`, `flow.Result`, `flow.Exit`, `flow.Env`, `flow.State`, `flow.Delta`, `flow.ErrInvalidAnswer`, `flow.ReadText`, `flow.Render`, `flow.Node.StringKey`, `flow.Node.UnknownKeys` (Task 0).
- Produces: `blocks.Gate` und `blocks.Exit`, beide `flow.Block`.

**Diese Lane legt keine `helpers_test.go` an.** Die Lane `agent` schreibt im selben Paket; eine Datei, die beide anlegen, ist beim Zusammenführen ein Konflikt in einer Datei, die keiner von beiden gehört. Testhelfer stehen in `gate_test.go` und `exit_test.go`, auch wenn sie einander ähneln, und tragen einen Namen mit `gate`/`exit` darin.

**`internal/blocks/doc.go` existiert schon** (aus Welle 0) und wird nicht angefasst.

- [ ] **Step 1: Write the failing tests**

`internal/blocks/gate_test.go`:

```go
package blocks_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/ultra-loom/internal/blocks"
	"github.com/xidus90/ultra-loom/internal/flow"
)

func gateNode(keys map[string]any) flow.Node {
	return flow.Node{Name: "approve", Kind: "gate", MaxVisits: flow.Cap{Add: 1}, Keys: keys}
}

func gateGraph() *flow.Graph {
	return &flow.Graph{Name: "f", File: "f.toml", Dir: "testdata"}
}

var gateKeys = map[string]any{
	"question": "questions/approve.md",
	"choices":  []any{"yes", "no"},
	"answer":   "answer",
}

func TestGateAsksItsQuestionAndPauses(t *testing.T) {
	state := flow.State{Fields: map[string]flow.Value{"count": 2}}
	got, err := blocks.Gate{}.Run(context.Background(), gateGraph(), gateNode(gateKeys), state, flow.Env{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Question == nil {
		t.Fatal("a gate without an answer pauses")
	}
	if *got.Question != "Approve after 2 rounds?" {
		t.Fatalf("question = %q", *got.Question)
	}
	if got.Delta != nil || got.Exit != nil {
		t.Fatalf("a pause changes nothing: %+v", got)
	}
}

func TestGateSplitsTheAnswerIntoChoiceAndRest(t *testing.T) {
	cases := []struct{ answer, choice, rest string }{
		{"yes", "yes", ""},
		{"no", "no", ""},
		{"no: the spec is thin", "no", "the spec is thin"},
		{"no the spec is thin", "no", "the spec is thin"},
		{"  yes  ", "yes", ""},
		{"no:\tstill open\n", "no", "still open"},
	}
	for _, c := range cases {
		t.Run(c.answer, func(t *testing.T) {
			answer := c.answer
			env := flow.Env{Answer: &answer}
			got, err := blocks.Gate{}.Run(context.Background(), gateGraph(), gateNode(gateKeys), flow.State{}, env)
			if err != nil {
				t.Fatal(err)
			}
			if got.Question != nil {
				t.Fatal("an answered gate does not pause again")
			}
			if got.Delta["answer"] != c.choice || got.Delta["answer_text"] != c.rest {
				t.Fatalf("delta = %#v", got.Delta)
			}
		})
	}
}

// Case matters, so that "no" and "No" cannot quietly mean the same thing: a
// gate is the place where a person's word is taken literally or not at all.
func TestGateRefusesAnAnswerThatIsNoChoice(t *testing.T) {
	for _, answer := range []string{"No", "yesterday", "maybe", "", "ye"} {
		t.Run(answer, func(t *testing.T) {
			text := answer
			env := flow.Env{Answer: &text}
			_, err := blocks.Gate{}.Run(context.Background(), gateGraph(), gateNode(gateKeys), flow.State{}, env)
			if !errors.Is(err, flow.ErrInvalidAnswer) {
				t.Fatalf("err = %v, want ErrInvalidAnswer", err)
			}
			if !strings.Contains(err.Error(), "yes, no") {
				t.Fatalf("the message names the choices: %v", err)
			}
		})
	}
}

func TestGateWritesTheAnswerFieldAndItsText(t *testing.T) {
	written := blocks.Gate{}.Writes(gateNode(gateKeys))
	if written["answer"] != flow.String || written["answer_text"] != flow.String {
		t.Fatalf("writes = %v", written)
	}
	if len(written) != 2 {
		t.Fatalf("writes = %v", written)
	}
}

func TestGateNamesItsQuestionAsAText(t *testing.T) {
	texts := blocks.Gate{}.Texts(gateNode(gateKeys))
	if len(texts) != 1 || texts[0].Key != "question" || texts[0].Path != "questions/approve.md" {
		t.Fatalf("texts = %+v", texts)
	}
}

func TestGateDefineCarriesTheQuestionBytesAndNoModel(t *testing.T) {
	got, err := blocks.Gate{}.Define(gateGraph(), gateNode(gateKeys), flow.Env{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(got.Text), "Approve after {{count}}") {
		t.Fatalf("text = %q", got.Text)
	}
	if got.Model != "" || got.Effort != "" || got.Profile != "" || got.Tools != nil {
		t.Fatalf("a gate has no model and no tools: %+v", got)
	}
}

func TestGateCheckFindsEveryProblemAtOnce(t *testing.T) {
	cases := []struct {
		name string
		keys map[string]any
		want []string
	}{
		{"nothing at all", map[string]any{}, []string{
			"question must name a file next to the flow",
			"answer must name a state field",
			"choices must be a list of at least two texts",
		}},
		{"an unknown key", map[string]any{
			"question": "q.md", "answer": "a", "choices": []any{"yes", "no"}, "choicez": "x",
		}, []string{`unknown key "choicez"`}},
		{"a choice that is not one word", map[string]any{
			"question": "q.md", "answer": "a", "choices": []any{"yes please", "no"},
		}, []string{`choice "yes please" must not hold whitespace or ":"`}},
		{"a choice twice", map[string]any{
			"question": "q.md", "answer": "a", "choices": []any{"yes", "yes"},
		}, []string{`choice "yes" is listed twice`}},
		{"a choice that is not a text", map[string]any{
			"question": "q.md", "answer": "a", "choices": []any{"yes", 2},
		}, []string{"choices must be a list of at least two texts"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := blocks.Gate{}.Check(gateNode(c.keys))
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Fatalf("finding %d is %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestGateOfAWellFormedNodeHasNothingToSay(t *testing.T) {
	// The parentheses are not decoration: `if blocks.Gate{}.Kind()` parses the
	// brace as the start of the if's body.
	if got := (blocks.Gate{}).Check(gateNode(gateKeys)); got != nil {
		t.Fatalf("got %v", got)
	}
	if (blocks.Gate{}).Kind() != "gate" {
		t.Fatal("kind")
	}
	if (blocks.Gate{}).NeedsBaseline() {
		t.Fatal("a gate asks a person, not the working tree")
	}
}

func TestGateReportsAQuestionItCannotRead(t *testing.T) {
	node := gateNode(map[string]any{"question": "questions/nope.md", "choices": []any{"yes", "no"}, "answer": "answer"})
	if _, err := (blocks.Gate{}).Run(context.Background(), gateGraph(), node, flow.State{}, flow.Env{}); err == nil {
		t.Fatal("want an error")
	}
	if _, err := (blocks.Gate{}).Define(gateGraph(), node, flow.Env{}); err == nil {
		t.Fatal("want an error")
	}
}

func TestGateReportsAQuestionItCannotRender(t *testing.T) {
	graph := gateGraph()
	graph.Dir = filepath.Join("testdata", "broken")
	node := gateNode(map[string]any{"question": "question.md", "choices": []any{"yes", "no"}, "answer": "answer"})
	if _, err := (blocks.Gate{}).Run(context.Background(), graph, node, flow.State{}, flow.Env{}); err == nil {
		t.Fatal("want an error")
	}
}
```

`internal/blocks/testdata/questions/approve.md`:

```
Approve after {{count}} rounds?
```

`internal/blocks/testdata/broken/question.md` — eine Datei mit `{{ name }}`, also einem Platzhalter mit Leerraum, den `tmpl.Parse` ablehnt.

`internal/blocks/exit_test.go`:

```go
package blocks_test

import (
	"context"
	"testing"

	"github.com/xidus90/ultra-loom/internal/blocks"
	"github.com/xidus90/ultra-loom/internal/flow"
)

func exitNode(keys map[string]any) flow.Node {
	return flow.Node{Name: "stop", Kind: "exit", MaxVisits: flow.Cap{Add: 1}, Keys: keys}
}

func exitGraph() *flow.Graph {
	return &flow.Graph{Name: "f", File: "f.toml", Dir: "testdata"}
}

func TestExitEndsTheRunWithItsCodeAndRenderedMessage(t *testing.T) {
	node := exitNode(map[string]any{"code": int64(4), "message": "rejected after {{count}} rounds"})
	state := flow.State{Fields: map[string]flow.Value{"count": 3}}
	got, err := blocks.Exit{}.Run(context.Background(), exitGraph(), node, state, flow.Env{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Exit == nil {
		t.Fatal("an exit node ends the run")
	}
	if got.Exit.Code != 4 || got.Exit.Message != "rejected after 3 rounds" {
		t.Fatalf("exit = %+v", got.Exit)
	}
	if got.Delta != nil || got.Question != nil || got.Tokens != 0 {
		t.Fatalf("an exit changes no field: %+v", got)
	}
}

func TestExitWritesNothingAndNeedsNoBaseline(t *testing.T) {
	node := exitNode(map[string]any{"code": int64(4), "message": "done"})
	// The parentheses are not decoration: `if blocks.Exit{}.Kind()` parses the
	// brace as the start of the if's body.
	if (blocks.Exit{}).Writes(node) != nil {
		t.Fatal("an exit writes no field")
	}
	if (blocks.Exit{}).NeedsBaseline() {
		t.Fatal("an exit reads no working tree")
	}
	if (blocks.Exit{}).Kind() != "exit" {
		t.Fatal("kind")
	}
}

// Inline, not a path: a message is a sentence, and a file per sentence would
// be a file per sentence.
func TestExitNamesItsMessageAsAnInlineText(t *testing.T) {
	texts := blocks.Exit{}.Texts(exitNode(map[string]any{"message": "done"}))
	if len(texts) != 1 || texts[0].Key != "message" || texts[0].Path != "" || texts[0].Inline != "done" {
		t.Fatalf("texts = %+v", texts)
	}
}

func TestExitDefineCarriesTheMessageBytes(t *testing.T) {
	got, err := blocks.Exit{}.Define(exitGraph(), exitNode(map[string]any{"message": "done"}), flow.Env{})
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Text) != "done" || got.Model != "" {
		t.Fatalf("definition = %+v", got)
	}
}

// The runtime spends 0 for done, 1 for failed and 3 for paused. A block that
// could choose one of them would make an exit code unreadable: a 3 would mean
// either "waiting at a gate" or "this flow said so".
func TestExitRefusesTheRuntimesOwnCodes(t *testing.T) {
	for _, code := range []int64{0, 1, 3} {
		node := exitNode(map[string]any{"code": code, "message": "m"})
		got := blocks.Exit{}.Check(node)
		if len(got) != 1 {
			t.Fatalf("code %d: %v", code, got)
		}
	}
}

func TestExitCheckFindsEveryProblemAtOnce(t *testing.T) {
	cases := []struct {
		name string
		keys map[string]any
		want []string
	}{
		{"nothing at all", map[string]any{}, []string{
			"code must be an integer other than 0, 1 and 3",
			"message must be a text",
		}},
		{"a code out of range", map[string]any{"code": int64(300), "message": "m"},
			[]string{"code 300 is outside 0..255"}},
		{"a code that is no integer", map[string]any{"code": "four", "message": "m"},
			[]string{"code must be an integer other than 0, 1 and 3"}},
		{"an unknown key", map[string]any{"code": int64(4), "message": "m", "mesage": "m"},
			[]string{`unknown key "mesage"`}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := blocks.Exit{}.Check(exitNode(c.keys))
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Fatalf("finding %d is %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestExitReportsAMessageItCannotRender(t *testing.T) {
	node := exitNode(map[string]any{"code": int64(4), "message": "{{ name }}"})
	if _, err := (blocks.Exit{}).Run(context.Background(), exitGraph(), node, flow.State{}, flow.Env{}); err == nil {
		t.Fatal("want an error")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-gate-exit" && go test ./internal/blocks/...`
Expected: FAIL — `undefined: blocks.Gate`.

- [ ] **Step 3: Implement the gate**

`internal/blocks/gate.go`:

```go
package blocks

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/xidus90/ultra-loom/internal/flow"
)

// Gate is the node kind that stops a run and asks a person. It replaces the
// Python gate's `apply`, which ran a flow's own code on the answer: a decision
// point that can run arbitrary code is a decision point nobody can read.
type Gate struct{}

// Kind is "gate".
func (Gate) Kind() string { return "gate" }

// Check reads the gate's own keys.
func (Gate) Check(node flow.Node) []string {
	var findings []string
	if node.StringKey("question") == "" {
		findings = append(findings, "question must name a file next to the flow")
	}
	if node.StringKey("answer") == "" {
		findings = append(findings, "answer must name a state field")
	}
	choices, ok := gateChoices(node)
	if !ok {
		findings = append(findings, "choices must be a list of at least two texts")
	}
	for index, choice := range choices {
		// A choice is matched by equality or as a prefix before ":" or space.
		// One that holds either of those separators could not be told from a
		// choice plus its reason.
		if strings.ContainsAny(choice, " \t\n:") {
			findings = append(findings, fmt.Sprintf("choice %q must not hold whitespace or \":\"", choice))
		}
		if slices.Index(choices, choice) != index {
			findings = append(findings, fmt.Sprintf("choice %q is listed twice", choice))
		}
	}
	for _, key := range node.UnknownKeys("question", "choices", "answer") {
		findings = append(findings, fmt.Sprintf("unknown key %q", key))
	}
	return findings
}

// Texts names the question file.
func (Gate) Texts(node flow.Node) []flow.Text {
	return []flow.Text{{Key: "question", Path: node.StringKey("question")}}
}

// Writes names the answer field and the field that carries the rest of the
// answer, both strings.
func (Gate) Writes(node flow.Node) map[string]flow.Type {
	answer := node.StringKey("answer")
	return map[string]flow.Type{answer: flow.String, answer + "_text": flow.String}
}

// NeedsBaseline is false: a gate asks a person, not the working tree.
func (Gate) NeedsBaseline() bool { return false }

// Define is the question's bytes and nothing else. A gate calls no model, so
// there is no model, effort or tool list to fingerprint.
func (Gate) Define(graph *flow.Graph, node flow.Node, _ flow.Env) (flow.Definition, error) {
	raw, err := flow.ReadText(graph.Dir, Gate{}.Texts(node)[0])
	if err != nil {
		return flow.Definition{}, err
	}
	return flow.Definition{Text: raw}, nil
}

// Run asks, or takes the answer a resume brought.
func (Gate) Run(_ context.Context, graph *flow.Graph, node flow.Node, state flow.State, env flow.Env) (flow.Result, error) {
	if env.Answer == nil {
		raw, err := flow.ReadText(graph.Dir, Gate{}.Texts(node)[0])
		if err != nil {
			return flow.Result{}, err
		}
		question, err := flow.Render(raw, state, env.Params)
		if err != nil {
			return flow.Result{}, err
		}
		return flow.Result{Question: &question}, nil
	}

	answer := strings.TrimSpace(*env.Answer)
	choices, _ := gateChoices(node)
	field := node.StringKey("answer")
	for _, choice := range choices {
		rest, ok := gateMatch(answer, choice)
		if !ok {
			continue
		}
		return flow.Result{Delta: flow.Delta{field: choice, field + "_text": rest}}, nil
	}
	return flow.Result{}, fmt.Errorf("%w; the choices are %s", flow.ErrInvalidAnswer, strings.Join(choices, ", "))
}

// gateMatch reports whether the answer is this choice, and what it said beyond
// it. Upper and lower case count: "no" and "No" are not quietly the same word.
func gateMatch(answer, choice string) (string, bool) {
	if answer == choice {
		return "", true
	}
	if !strings.HasPrefix(answer, choice) {
		return "", false
	}
	rest := answer[len(choice):]
	if !strings.ContainsAny(rest[:1], " \t\n:") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(rest, ":")), true
}

// gateChoices reads the choices as TOML decoded them. It reports false for
// anything that is not a list of at least two texts, so that Check has one
// finding to make and Run has a list it can trust.
func gateChoices(node flow.Node) ([]string, bool) {
	raw, ok := node.Keys["choices"].([]any)
	if !ok || len(raw) < 2 {
		return nil, false
	}
	choices := make([]string, 0, len(raw))
	for _, item := range raw {
		text, ok := item.(string)
		if !ok {
			return nil, false
		}
		choices = append(choices, text)
	}
	return choices, true
}
```

- [ ] **Step 4: Implement the exit**

`internal/blocks/exit.go`:

```go
package blocks

import (
	"context"
	"fmt"

	"github.com/xidus90/ultra-loom/internal/flow"
)

// Exit is the node kind that ends a run with a code of its own. A flow with
// more than one way of failing needs more than one way of saying so: a hook
// cannot tell "the checks stayed red" from "the repairer touched a test file"
// if both arrive as 1.
type Exit struct{}

// Kind is "exit".
func (Exit) Kind() string { return "exit" }

// Check reads the exit's own keys.
//
// 0, 1 and 3 are refused here rather than at the command line: the runtime
// spends them on done, failed and paused, and a 3 that could also mean "this
// flow said so" is a code no caller can act on.
func (Exit) Check(node flow.Node) []string {
	var findings []string
	code, ok := node.Keys["code"].(int64)
	switch {
	case !ok || code == 0 || code == 1 || code == 3:
		findings = append(findings, "code must be an integer other than 0, 1 and 3")
	case code < 0 || code > 255:
		findings = append(findings, fmt.Sprintf("code %d is outside 0..255", code))
	}
	if _, ok := node.Keys["message"].(string); !ok {
		findings = append(findings, "message must be a text")
	}
	for _, key := range node.UnknownKeys("code", "message") {
		findings = append(findings, fmt.Sprintf("unknown key %q", key))
	}
	return findings
}

// Texts names the message, which is inline: a message is a sentence, and a
// file per sentence would be a file per sentence.
func (Exit) Texts(node flow.Node) []flow.Text {
	return []flow.Text{{Key: "message", Inline: node.StringKey("message")}}
}

// Writes nothing: an exit ends the run, it does not change it.
func (Exit) Writes(flow.Node) map[string]flow.Type { return nil }

// NeedsBaseline is false.
func (Exit) NeedsBaseline() bool { return false }

// Define is the message's bytes.
func (Exit) Define(graph *flow.Graph, node flow.Node, _ flow.Env) (flow.Definition, error) {
	raw, err := flow.ReadText(graph.Dir, Exit{}.Texts(node)[0])
	if err != nil {
		return flow.Definition{}, err
	}
	return flow.Definition{Text: raw}, nil
}

// Run renders the message and hands the runner the code to end with.
func (Exit) Run(_ context.Context, graph *flow.Graph, node flow.Node, state flow.State, env flow.Env) (flow.Result, error) {
	raw, err := flow.ReadText(graph.Dir, Exit{}.Texts(node)[0])
	if err != nil {
		return flow.Result{}, err
	}
	message, err := flow.Render(raw, state, env.Params)
	if err != nil {
		return flow.Result{}, err
	}
	code, _ := node.Keys["code"].(int64)
	return flow.Result{Exit: &flow.Exit{Code: int(code), Message: message}}, nil
}
```

- [ ] **Step 5: Run the tests with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-gate-exit" && go test -cover ./internal/blocks/...`
Expected: `ok … coverage: 100.0% of statements`.

- [ ] **Step 6: gofmt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-gate-exit" && gofmt -l internal/blocks`
Expected: keine Ausgabe.

- [ ] **Step 7: Commit**

```
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-gate-exit" && git add internal/blocks && git commit -F <message-file>
```

Betreff: `Ask at a gate and end at an exit`

---

### Task 3: Lane `agent` — der Baustein `agent`

**Worktree:** `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-agent`, Zweig `ulflow/agent`

**Files:**
- Create: `internal/blocks/agent.go`
- Test: `internal/blocks/agent_test.go`
- Create: `internal/blocks/testdata/instructions/draft.md`

**Interfaces:**
- Consumes: `flow.Block`, `flow.Node`, `flow.Text`, `flow.Definition`, `flow.Result`, `flow.Env`, `flow.State`, `flow.Delta`, `flow.Type`, `flow.Coerce`, `flow.ReadText`, `flow.Render`, `flow.Node.StringKey`, `flow.Node.UnknownKeys` (Task 0); `model.Model`, `model.Request`, `model.Reply`, `model.ReplyType`, `model.NewFake`, `model.Answer`; `flowcfg.Config.Resolve`, `flowcfg.Resolved.Label`, `flowcfg.Tools`.
- Produces: `blocks.Agent`, ein `flow.Block`.

**Diese Lane legt keine `helpers_test.go` an**, aus demselben Grund wie die Lane `gate-exit`: beide schreiben in `internal/blocks`. Testhelfer stehen in `agent_test.go` und tragen `agent` im Namen. **`internal/blocks/testdata/questions/` und `testdata/broken/` gehören der anderen Lane** — diese legt nur `testdata/instructions/` an.

**Schlüssel des Knotens:** `instruction` (Pfad, Pflicht), `reply` (Tabelle Feldname → `"string"`/`"int"`/`"bool"`, Pflicht, nicht leer), `tools` (Profilname, Vorgabe `read_only`), `effort` (frei, wird durchgereicht). `model` ist ein geteilter Schlüssel und steht in `flow.Node.Model`, nicht in `Keys`.

- [ ] **Step 1: Write the failing tests**

`internal/blocks/agent_test.go`:

```go
package blocks_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/ultra-loom/internal/blocks"
	"github.com/xidus90/ultra-loom/internal/flow"
	"github.com/xidus90/ultra-loom/internal/flowcfg"
	"github.com/xidus90/ultra-loom/internal/model"
)

func agentNode(keys map[string]any) flow.Node {
	node := flow.Node{Name: "draft", Kind: "agent", MaxVisits: flow.Cap{Add: 1}, Keys: keys}
	return node
}

func agentKeys() map[string]any {
	return map[string]any{
		"instruction": "instructions/draft.md",
		"tools":       "edit",
		"effort":      "high",
		"reply":       map[string]any{"verdict": "string", "count": "int"},
	}
}

func agentGraph() *flow.Graph {
	return &flow.Graph{Name: "f", File: "f.toml", Dir: "testdata"}
}

func agentConfig() flowcfg.Config {
	return flowcfg.Config{
		Default: "writer",
		Models:  map[string]flowcfg.ModelSpec{"writer": {Provider: "claude", Model: "claude-opus-5"}},
	}
}

func TestAgentAsksTheModelWhatTheInstructionSays(t *testing.T) {
	fake := model.NewFake(model.Answer{Reply: model.Reply{
		Fields: map[string]any{"verdict": "done", "count": float64(2)},
		Tokens: 41,
		Model:  "claude-opus-5-20260501",
	}})
	state := flow.State{Fields: map[string]flow.Value{"topic": "the spec"}}
	env := flow.Env{Model: fake, Agent: agentConfig(), Params: flow.Params{"max_rounds": 5}}

	got, err := (blocks.Agent{}).Run(context.Background(), agentGraph(), agentNode(agentKeys()), state, env)
	if err != nil {
		t.Fatal(err)
	}
	if got.Delta["verdict"] != "done" || got.Delta["count"] != 2 {
		t.Fatalf("delta = %#v", got.Delta)
	}
	if got.Tokens != 41 {
		t.Fatalf("tokens = %d", got.Tokens)
	}
	// The provider in front of the bare name the adapter reported: a journal
	// line that said "claude-opus-5-20260501" alone would not say who answered.
	if got.Model != "claude:claude-opus-5-20260501" {
		t.Fatalf("model = %q", got.Model)
	}

	seen := fake.Seen()
	if len(seen) != 1 {
		t.Fatalf("%d requests", len(seen))
	}
	if seen[0].Prompt != "Draft the spec in 5 rounds.\n" {
		t.Fatalf("prompt = %q", seen[0].Prompt)
	}
	if !slices.Equal(seen[0].Tools, []string{"Edit", "Glob", "Grep", "Read", "Write"}) {
		t.Fatalf("tools = %v", seen[0].Tools)
	}
	if seen[0].Effort != "high" || seen[0].Provider != "claude" || seen[0].Model != "claude-opus-5" {
		t.Fatalf("request = %+v", seen[0])
	}
	if seen[0].Reply["verdict"] != model.ReplyString || seen[0].Reply["count"] != model.ReplyInt {
		t.Fatalf("reply schema = %v", seen[0].Reply)
	}
}

// The adapter may not know which model answered. Then the chain's own answer
// stands, which is what the journal would otherwise have nothing to record.
func TestAgentFallsBackToTheResolvedLabel(t *testing.T) {
	fake := model.NewFake(model.Answer{Reply: model.Reply{
		Fields: map[string]any{"verdict": "done", "count": 1},
	}})
	env := flow.Env{Model: fake, Agent: agentConfig()}
	got, err := (blocks.Agent{}).Run(context.Background(), agentGraph(), agentNode(agentKeys()), flow.State{}, env)
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != "claude:claude-opus-5" {
		t.Fatalf("model = %q", got.Model)
	}
}

// No stage of the chain names a model: the adapter passes none and the CLI
// takes its own default. The journal still says which provider was asked.
func TestAgentWithoutAnyModelNamesTheCliDefault(t *testing.T) {
	fake := model.NewFake(model.Answer{Reply: model.Reply{
		Fields: map[string]any{"verdict": "done", "count": 1},
	}})
	env := flow.Env{Model: fake, Agent: flowcfg.Config{}}
	got, err := (blocks.Agent{}).Run(context.Background(), agentGraph(), agentNode(agentKeys()), flow.State{}, env)
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != "claude:cli-default" {
		t.Fatalf("model = %q", got.Model)
	}
	if fake.Seen()[0].Model != "" {
		t.Fatalf("the adapter was handed a model: %q", fake.Seen()[0].Model)
	}
}

func TestAgentTakesTheNodesModelOverTheFlows(t *testing.T) {
	config := flowcfg.Config{Models: map[string]flowcfg.ModelSpec{
		"writer": {Provider: "claude", Model: "claude-opus-5"},
		"cheap":  {Provider: "agy", Model: "gemini"},
	}}
	graph := agentGraph()
	graph.Model = "writer"
	node := agentNode(agentKeys())
	node.Model = "cheap"
	fake := model.NewFake(model.Answer{Reply: model.Reply{Fields: map[string]any{"verdict": "d", "count": 1}}})

	got, err := (blocks.Agent{}).Run(context.Background(), graph, node, flow.State{}, flow.Env{Model: fake, Agent: config})
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != "agy:gemini" {
		t.Fatalf("model = %q", got.Model)
	}
}

// Every field, and no other. A reply short of one field is a model that did
// not do what it was told, and a reply with one too many is a model that
// answered a different question.
func TestAgentRefusesAReplyThatIsNotExactlyWhatWasAsked(t *testing.T) {
	cases := []struct {
		name   string
		fields map[string]any
		want   string
	}{
		{"a field missing", map[string]any{"verdict": "done"}, `no value for "count"`},
		{"a field too many", map[string]any{"verdict": "d", "count": 1, "extra": true}, `unexpected field "extra"`},
		{"a field of the wrong type", map[string]any{"verdict": "d", "count": "two"}, `field "count": cannot read two as int`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := model.NewFake(model.Answer{Reply: model.Reply{Fields: c.fields}})
			env := flow.Env{Model: fake, Agent: agentConfig()}
			_, err := (blocks.Agent{}).Run(context.Background(), agentGraph(), agentNode(agentKeys()), flow.State{}, env)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want %q in it", err, c.want)
			}
		})
	}
}

func TestAgentPassesOnWhatTheModelRefused(t *testing.T) {
	boom := errors.New("the CLI is not installed")
	fake := model.NewFake(model.Answer{Err: boom})
	env := flow.Env{Model: fake, Agent: agentConfig()}
	_, err := (blocks.Agent{}).Run(context.Background(), agentGraph(), agentNode(agentKeys()), flow.State{}, env)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}

// M1 registers no adapter, so this is the message a user meets before the
// runtime spends anything. The runner turns it into an error entry.
func TestAgentWithoutAModelSaysSo(t *testing.T) {
	env := flow.Env{Agent: agentConfig()}
	_, err := (blocks.Agent{}).Run(context.Background(), agentGraph(), agentNode(agentKeys()), flow.State{}, env)
	if err == nil || !strings.Contains(err.Error(), `node "draft" needs a model`) {
		t.Fatalf("err = %v", err)
	}
}

func TestAgentDefineFingerprintsEverythingTheNodeWasTold(t *testing.T) {
	got, err := (blocks.Agent{}).Define(agentGraph(), agentNode(agentKeys()), flow.Env{Agent: agentConfig()})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(got.Text), "Draft {{topic}}") {
		t.Fatalf("text = %q", got.Text)
	}
	if got.Model != "claude:claude-opus-5" || got.Effort != "high" || got.Profile != "edit" {
		t.Fatalf("definition = %+v", got)
	}
	if !slices.Equal(got.Tools, []string{"Edit", "Glob", "Grep", "Read", "Write"}) {
		t.Fatalf("tools = %v", got.Tools)
	}
}

func TestAgentDefineReportsAModelTheChainCannotResolve(t *testing.T) {
	node := agentNode(agentKeys())
	node.Model = "wrter"
	if _, err := (blocks.Agent{}).Define(agentGraph(), node, flow.Env{Agent: agentConfig()}); err == nil {
		t.Fatal("want an error")
	}
}

func TestAgentWritesExactlyItsReplyFields(t *testing.T) {
	written := (blocks.Agent{}).Writes(agentNode(agentKeys()))
	if len(written) != 2 || written["verdict"] != flow.String || written["count"] != flow.Int {
		t.Fatalf("writes = %v", written)
	}
}

func TestAgentNamesItsInstructionAsAText(t *testing.T) {
	texts := (blocks.Agent{}).Texts(agentNode(agentKeys()))
	if len(texts) != 1 || texts[0].Key != "instruction" || texts[0].Path != "instructions/draft.md" {
		t.Fatalf("texts = %+v", texts)
	}
	if (blocks.Agent{}).Kind() != "agent" {
		t.Fatal("kind")
	}
	if (blocks.Agent{}).NeedsBaseline() {
		t.Fatal("an agent node reads what its instruction says, not the working tree")
	}
}

func TestAgentCheckFindsEveryProblemAtOnce(t *testing.T) {
	cases := []struct {
		name string
		keys map[string]any
		want []string
	}{
		{"nothing at all", map[string]any{}, []string{
			"instruction must name a file next to the flow",
			"reply must name at least one field with its type",
		}},
		{"a reply type that is no scalar", map[string]any{
			"instruction": "i.md", "reply": map[string]any{"notes": "list[string]"},
		}, []string{`reply field "notes" is list[string]; a reply holds string, int or bool`}},
		{"an unknown tool profile", map[string]any{
			"instruction": "i.md", "reply": map[string]any{"v": "string"}, "tools": "root",
		}, []string{`unknown tool profile "root"; known profiles: edit, mcp, read_only, shell`}},
		{"an unknown key", map[string]any{
			"instruction": "i.md", "reply": map[string]any{"v": "string"}, "instructon": "i.md",
		}, []string{`unknown key "instructon"`}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := (blocks.Agent{}).Check(agentNode(c.keys))
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Fatalf("finding %d is %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

// The default, so that a node that says nothing about tools cannot write.
func TestAgentWithoutToolsIsReadOnly(t *testing.T) {
	keys := agentKeys()
	delete(keys, "tools")
	got, err := (blocks.Agent{}).Define(agentGraph(), agentNode(keys), flow.Env{Agent: agentConfig()})
	if err != nil {
		t.Fatal(err)
	}
	if got.Profile != "read_only" || !slices.Equal(got.Tools, []string{"Glob", "Grep", "Read"}) {
		t.Fatalf("definition = %+v", got)
	}
}

func TestAgentPassesTheConfiguredMcpServers(t *testing.T) {
	keys := agentKeys()
	keys["tools"] = "mcp"
	config := agentConfig()
	config.MCPServers = []string{"brain"}
	got, err := (blocks.Agent{}).Define(agentGraph(), agentNode(keys), flow.Env{Agent: config})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Tools, []string{"Glob", "Grep", "Read", "mcp__brain"}) {
		t.Fatalf("tools = %v", got.Tools)
	}
}
```

`internal/blocks/testdata/instructions/draft.md`:

```
Draft {{topic}} in {{max_rounds}} rounds.
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-agent" && go test ./internal/blocks/...`
Expected: FAIL — `undefined: blocks.Agent`.

- [ ] **Step 3: Implement**

`internal/blocks/agent.go`:

```go
package blocks

import (
	"context"
	"fmt"
	"slices"

	"github.com/xidus90/ultra-loom/internal/flow"
	"github.com/xidus90/ultra-loom/internal/flowcfg"
	"github.com/xidus90/ultra-loom/internal/model"
)

// defaultProfile is what a node that says nothing about tools gets. Read-only,
// because the profiles are the only ceiling between a node and Write, and a
// node that forgot to say so has not asked for one.
const defaultProfile = "read_only"

// replyTypes maps a flow's types onto what a reply may hold. A reply is flat
// on purpose: a model that has to fill a list is a model that has to be told
// what a list looks like in its answer format, and every adapter would tell it
// differently.
var replyTypes = map[string]model.ReplyType{
	string(flow.String): model.ReplyString,
	string(flow.Int):    model.ReplyInt,
	string(flow.Bool):   model.ReplyBool,
}

// Agent is the node kind that renders an instruction, asks a model and checks
// the answer against the fields the node declared.
type Agent struct{}

// Kind is "agent".
func (Agent) Kind() string { return "agent" }

// Check reads the agent's own keys.
func (Agent) Check(node flow.Node) []string {
	var findings []string
	if node.StringKey("instruction") == "" {
		findings = append(findings, "instruction must name a file next to the flow")
	}
	reply, _ := node.Keys["reply"].(map[string]any)
	if len(reply) == 0 {
		findings = append(findings, "reply must name at least one field with its type")
	}
	for _, name := range sortedNames(reply) {
		declared, _ := reply[name].(string)
		if _, ok := replyTypes[declared]; !ok {
			findings = append(findings,
				fmt.Sprintf("reply field %q is %s; a reply holds string, int or bool", name, declared))
		}
	}
	if profile := agentProfile(node); profile != "" {
		if _, err := flowcfg.Tools(profile, nil); err != nil {
			findings = append(findings, err.Error())
		}
	}
	for _, key := range node.UnknownKeys("instruction", "reply", "tools", "effort") {
		findings = append(findings, fmt.Sprintf("unknown key %q", key))
	}
	return findings
}

// Texts names the instruction file.
func (Agent) Texts(node flow.Node) []flow.Text {
	return []flow.Text{{Key: "instruction", Path: node.StringKey("instruction")}}
}

// Writes are the reply's fields, with the types the node declared for them.
func (Agent) Writes(node flow.Node) map[string]flow.Type {
	reply, _ := node.Keys["reply"].(map[string]any)
	written := make(map[string]flow.Type, len(reply))
	for name, declared := range reply {
		text, _ := declared.(string)
		written[name] = flow.Type(text)
	}
	return written
}

// NeedsBaseline is false: an agent node reads what its instruction says.
func (Agent) NeedsBaseline() bool { return false }

// Define is everything the node's result depends on besides its input: the
// instruction's bytes, the resolved model, the effort and the tool list.
func (Agent) Define(graph *flow.Graph, node flow.Node, env flow.Env) (flow.Definition, error) {
	raw, err := flow.ReadText(graph.Dir, Agent{}.Texts(node)[0])
	if err != nil {
		return flow.Definition{}, err
	}
	resolved, err := env.Agent.Resolve(node.Model, graph.Model)
	if err != nil {
		return flow.Definition{}, fmt.Errorf("node %q: %w", node.Name, err)
	}
	profile := agentProfile(node)
	tools, err := flowcfg.Tools(profile, env.Agent.MCPServers)
	if err != nil {
		return flow.Definition{}, fmt.Errorf("node %q: %w", node.Name, err)
	}
	return flow.Definition{
		Text:    raw,
		Model:   resolved.Label(),
		Effort:  node.StringKey("effort"),
		Profile: profile,
		Tools:   tools,
	}, nil
}

// Run renders the instruction, asks the model and turns the answer into a delta.
func (Agent) Run(ctx context.Context, graph *flow.Graph, node flow.Node, state flow.State, env flow.Env) (flow.Result, error) {
	if env.Model == nil {
		// Named here and not only at the command line: a run that got this far
		// without an adapter would otherwise fail inside a nil call.
		return flow.Result{}, fmt.Errorf("node %q needs a model; M1 registers no adapter yet", node.Name)
	}
	definition, err := Agent{}.Define(graph, node, env)
	if err != nil {
		return flow.Result{}, err
	}
	prompt, err := flow.Render(definition.Text, state, env.Params)
	if err != nil {
		return flow.Result{}, err
	}
	resolved, err := env.Agent.Resolve(node.Model, graph.Model)
	if err != nil {
		return flow.Result{}, fmt.Errorf("node %q: %w", node.Name, err)
	}

	written := Agent{}.Writes(node)
	schema := make(map[string]model.ReplyType, len(written))
	for name, declared := range written {
		schema[name] = replyTypes[string(declared)]
	}

	reply, err := env.Model.Ask(ctx, model.Request{
		Prompt:   prompt,
		Tools:    definition.Tools,
		Effort:   definition.Effort,
		Provider: resolved.Provider,
		Model:    resolved.Model,
		Reply:    schema,
	})
	if err != nil {
		return flow.Result{}, err
	}

	delta, err := agentDelta(reply.Fields, written)
	if err != nil {
		return flow.Result{}, fmt.Errorf("node %q: %w", node.Name, err)
	}
	label := definition.Model
	if reply.Model != "" {
		// The adapter reports a bare name; the journal records who answered.
		label = resolved.Provider + ":" + reply.Model
	}
	return flow.Result{Delta: delta, Tokens: reply.Tokens, Model: label}, nil
}

// agentDelta checks the answer against the declared fields and converts every
// value into the Go type its type stands for. Every field, and no other.
func agentDelta(fields map[string]any, written map[string]flow.Type) (flow.Delta, error) {
	delta := make(flow.Delta, len(written))
	for _, name := range sortedTypes(written) {
		raw, ok := fields[name]
		if !ok {
			return nil, fmt.Errorf("no value for %q", name)
		}
		value, err := flow.Coerce(written[name], raw)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", name, err)
		}
		delta[name] = value
	}
	for _, name := range sortedNames(fields) {
		if _, ok := written[name]; !ok {
			return nil, fmt.Errorf("unexpected field %q", name)
		}
	}
	return delta, nil
}

// sortedNames and sortedTypes exist so that a message about a reply with two
// mistakes in it names the same one twice running.
func sortedNames(table map[string]any) []string {
	names := make([]string, 0, len(table))
	for name := range table {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func sortedTypes(table map[string]flow.Type) []string {
	names := make([]string, 0, len(table))
	for name := range table {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// agentProfile is the node's tool profile, or the default when it names none.
func agentProfile(node flow.Node) string {
	if profile := node.StringKey("tools"); profile != "" {
		return profile
	}
	return defaultProfile
}
```

Note zu `TestAgentAsksTheModelWhatTheInstructionSays`: Der Platzhalter `{{topic}}` kommt aus dem Zustand, `{{max_rounds}}` aus den Parametern. Die erwartete Zeile im Test ist `"Draft the spec in 5 rounds.\n"` — die Datei endet mit einem Zeilenumbruch, und `Render` schneidet nichts ab.

- [ ] **Step 4: Run the tests with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-agent" && go test -cover ./internal/blocks/...`
Expected: `ok … coverage: 100.0% of statements`.

Die Lane sieht `gate.go` und `exit.go` nicht — sie entstehen gerade woanders. 100 % bezieht sich auf die Anweisungen, die diese Lane anlegt; nach dem Zusammenführen misst Task 5 das Paket als Ganzes.

- [ ] **Step 5: gofmt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-agent" && gofmt -l internal/blocks`
Expected: keine Ausgabe.

- [ ] **Step 6: Commit**

```
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-agent" && git add internal/blocks && git commit -F <message-file>
```

Betreff: `Ask a model what an instruction says and check its answer`

---

### Task 4: Lane `runner` — `internal/runner`

**Worktree:** `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-runner`, Zweig `ulflow/runner`

**Files:**
- Create: `internal/runner/runner.go`, `internal/runner/walk.go`
- Modify: `internal/runner/doc.go` (aus Welle 0, bisher nur die Paketzeile)
- Test: `internal/runner/runner_test.go`, `internal/runner/gate_test.go`, `internal/runner/retrace_test.go`

**Interfaces:**
- Consumes: `flow.Graph`, `flow.Node`, `flow.Edge`, `flow.Cap.Limit`, `flow.State`, `flow.Delta`, `flow.Params`, `flow.Value`, `flow.Coerce`, `flow.Catalog`, `flow.Block`, `flow.Result`, `flow.Exit`, `flow.Env`, `flow.ErrInvalidAnswer`, `flow.End` (Task 0); `journal.Entry`, `journal.PendingGate`, `journal.InputHash`, `journal.DefinitionHash`, `journal.Lookup`.
- Produces:
  - `type runner.Journal interface { Entries() ([]journal.Entry, error); Append(journal.Entry) error; Pending() (*journal.PendingGate, error) }`
  - `type runner.Clock func() time.Time`
  - `type runner.Options struct { Graph *flow.Graph; Catalog *flow.Catalog; Journal Journal; Env flow.Env; Clock Clock; Warn func(string); Replay bool }`
  - `type runner.Result struct { Status, Node, Question, Detail string; State flow.State; ExitCode *int }`
  - `func runner.New(opts Options) *Runner`
  - `func (r *Runner) Run(ctx context.Context) (Result, error)`
  - `func (r *Runner) Resume(ctx context.Context, answer *string) (Result, error)`

**Diese Lane importiert `internal/blocks` und `internal/flowload` nicht.** Sie baut ihre Graphen von Hand und registriert eigene Testbausteine im `Catalog` — so verlangt es der Verhaltensvertrag, der über Gehen und Nachgehen spricht und nicht über TOML. Lader, echte Bausteine und Runner treffen sich in Welle 3.

#### Entscheidungen dieser Lane

- **Der Runner prüft den Graphen nicht.** Pythons `Runner.run` rief `graph.validate()`; in Go tut das der Lader, Stufe 6, bevor ein `*flow.Graph` überhaupt entsteht. Ein zweites Mal prüfen hieße, die fünf Regeln zweimal zu schreiben. Was der Runner stattdessen beim Start prüft, steht unten.
- **`Journal` ist eine Schnittstelle mit drei Methoden**, nicht ein Pfad. Der Runner schreibt und liest dasselbe Journal, und ein Test, der dafür eine Datei anlegen müsste, prüfte das Dateisystem mit. `Pending` gehört dazu, weil `journal.Pending` einen Pfad nimmt und diese Lane `internal/journal` nicht ändern darf.
- **Der Eingabe-Hash geht über Felder und Besuche** (Entscheidung 6): `journal.InputHash(name, map[string]any{"fields": state.Fields, "visits": state.Visits})`, mit dem Besuch dieses Knotens schon gezählt.
- **Die Uhr liefert `time.Time`**, und die Dauer ist `end.Sub(start).Seconds()`. Ein Test übergibt eine Uhr, die bei jedem Aufruf um eine feste Spanne weiterrückt.
- **Warnungen gehen an `Warn`**, nicht an `os.Stderr`. Wer nach stderr schreibt, ist `cmd/flow` in Welle 3.

#### Was der Runner beim Start prüft, vor dem ersten Journaleintrag

1. Jeder Deckel: `node.MaxVisits.Limit(env.Params)`. Ein Fehler oder ein Ergebnis unter 1 beendet den Aufruf mit einem Go-Fehler (Entscheidungen 9 und 14).
2. Braucht ein Knoten die Basis (`Block.NeedsBaseline`) und ist `env.Baseline` nil, ebenso.
3. Für jede Knotenart muss der `Catalog` einen Baustein haben; sonst ebenso.

Diese drei geben `(Result{}, error)` zurück, kein `Result` mit `Status: "error"`: Ein Lauf, der so endet, hat nicht stattgefunden, und die Spec verlangt, dass eine Ablehnung kein Journal anlegt.

- [ ] **Step 1: Write the failing tests — walking, caps, error edges**

`internal/runner/runner_test.go`:

```go
package runner_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xidus90/ultra-loom/internal/flow"
	"github.com/xidus90/ultra-loom/internal/journal"
	"github.com/xidus90/ultra-loom/internal/runner"
)

// memJournal is the journal a test run writes into. A file would test the file
// system; what is under test is what the runner writes, and when.
type memJournal struct{ lines []journal.Entry }

func (j *memJournal) Entries() ([]journal.Entry, error) { return j.lines, nil }

func (j *memJournal) Append(entry journal.Entry) error {
	j.lines = append(j.lines, entry)
	return nil
}

func (j *memJournal) Pending() (*journal.PendingGate, error) {
	var open *journal.PendingGate
	for _, entry := range j.lines {
		switch entry.Outcome {
		case "paused":
			detail := ""
			if entry.Detail != nil {
				detail = *entry.Detail
			}
			open = &journal.PendingGate{Node: entry.Node, Question: detail, InputHash: entry.InputHash}
		case "ok":
			if open != nil && open.Node == entry.Node && open.InputHash == entry.InputHash {
				open = nil
			}
		}
	}
	return open, nil
}

// stepBlock is a node that does one thing: apply a fixed delta, or fail, or
// end the run. Everything the runner is asked about is visible in what it
// journals afterwards.
type stepBlock struct {
	delta flow.Delta
	err   error
	exit  *flow.Exit
	runs  *int
}

func (stepBlock) Kind() string                        { return "step" }
func (stepBlock) Check(flow.Node) []string            { return nil }
func (stepBlock) Texts(flow.Node) []flow.Text         { return nil }
func (stepBlock) NeedsBaseline() bool                 { return false }

func (b stepBlock) Writes(flow.Node) map[string]flow.Type {
	written := make(map[string]flow.Type, len(b.delta))
	for name := range b.delta {
		written[name] = flow.String
	}
	return written
}

func (stepBlock) Define(*flow.Graph, flow.Node, flow.Env) (flow.Definition, error) {
	return flow.Definition{Text: []byte("v1")}, nil
}

func (b stepBlock) Run(context.Context, *flow.Graph, flow.Node, flow.State, flow.Env) (flow.Result, error) {
	if b.runs != nil {
		*b.runs++
	}
	if b.err != nil {
		return flow.Result{}, b.err
	}
	if b.exit != nil {
		return flow.Result{Exit: b.exit}, nil
	}
	return flow.Result{Delta: b.delta, Tokens: 7}, nil
}

// ticks hands out a clock that moves half a second per call, so a duration in
// the journal is a number a test can name.
func ticks() runner.Clock {
	now := time.Unix(0, 0)
	return func() time.Time {
		now = now.Add(500 * time.Millisecond)
		return now
	}
}

func node(name string, visits int) flow.Node {
	// Cap{} means no visit at all, so every hand-built node says what it allows.
	return flow.Node{Name: name, Kind: "step", MaxVisits: flow.Cap{Add: visits}}
}

func graphOf(nodes []flow.Node, edges []flow.Edge, state map[string]flow.Field) *flow.Graph {
	return &flow.Graph{Name: "f", File: "f.toml", Start: nodes[0].Name, Nodes: nodes, Edges: edges, State: state}
}

func catalogOf(t *testing.T, blocks ...flow.Block) *flow.Catalog {
	t.Helper()
	made, err := flow.NewCatalog(blocks, nil)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func TestARunWalksToEndAndJournalsEveryStep(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 1), node("b", 1)},
		[]flow.Edge{{From: "a", To: "b"}, {From: "b", To: flow.End}},
		map[string]flow.Field{"note": {Type: flow.String, Default: ""}},
	)
	run := runner.New(runner.Options{
		Graph:   graph,
		Catalog: catalogOf(t, stepBlock{delta: flow.Delta{"note": "done"}}),
		Journal: log,
		Clock:   ticks(),
	})

	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "done" {
		t.Fatalf("status = %q, detail = %q", got.Status, got.Detail)
	}
	if len(log.lines) != 2 {
		t.Fatalf("%d entries", len(log.lines))
	}
	if log.lines[0].Node != "a" || log.lines[0].Outcome != "ok" || log.lines[0].Tokens != 7 {
		t.Fatalf("entry = %+v", log.lines[0])
	}
	if log.lines[0].Seconds != 0.5 {
		t.Fatalf("seconds = %v", log.lines[0].Seconds)
	}
	if log.lines[0].DefinitionHash == nil || *log.lines[0].DefinitionHash == "" {
		t.Fatal("every ulflow entry carries a definition hash")
	}
	if log.lines[0].Model != nil {
		t.Fatal("a block without a model records none")
	}
	if got.State.Fields["note"] != "done" {
		t.Fatalf("state = %#v", got.State.Fields)
	}
}

// File order, and a condition that holds decides. An edge without a condition
// always holds, which is why it is written last.
func TestTheFirstEdgeWhoseConditionHoldsIsTaken(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 1), node("yes", 1), node("no", 1)},
		[]flow.Edge{
			{From: "a", To: "yes", When: fieldIs("note", "done")},
			{From: "a", To: "no"},
			{From: "yes", To: flow.End},
			{From: "no", To: flow.End},
		},
		map[string]flow.Field{"note": {Type: flow.String, Default: ""}},
	)
	run := runner.New(runner.Options{
		Graph:   graph,
		Catalog: catalogOf(t, stepBlock{delta: flow.Delta{"note": "done"}}),
		Journal: log,
		Clock:   ticks(),
	})
	if _, err := run.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if log.lines[1].Node != "yes" {
		t.Fatalf("went to %q", log.lines[1].Node)
	}
}

func TestARunWithNoEdgeThatAppliesEndsWithAnError(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 1), node("b", 1)},
		[]flow.Edge{{From: "a", To: "b", When: fieldIs("note", "never")}, {From: "b", To: flow.End}},
		map[string]flow.Field{"note": {Type: flow.String, Default: ""}},
	)
	run := runner.New(runner.Options{Graph: graph, Catalog: catalogOf(t, stepBlock{}), Journal: log, Clock: ticks()})
	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.Detail != `no edge out of "a" applies to the current state` {
		t.Fatalf("result = %+v", got)
	}
}

// The runaway-loop guard. 0 tokens and 0 seconds, because nothing ran.
func TestANodeOverItsCapEndsTheRun(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 2)},
		[]flow.Edge{{From: "a", To: "a"}},
		map[string]flow.Field{"n": {Type: flow.Int, Default: 0}},
	)
	run := runner.New(runner.Options{Graph: graph, Catalog: catalogOf(t, stepBlock{}), Journal: log, Clock: ticks()})
	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.Detail != `node "a" exceeded max_visits=2` {
		t.Fatalf("result = %+v", got)
	}
	last := log.lines[len(log.lines)-1]
	if last.Outcome != "error" || last.Tokens != 0 || last.Seconds != 0 {
		t.Fatalf("entry = %+v", last)
	}
}

func TestANodeThatFailsTakesItsErrorEdge(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 1), node("fix", 1)},
		[]flow.Edge{{From: "a", To: flow.End}, {From: "a", To: "fix", OnError: true}, {From: "fix", To: flow.End}},
		nil,
	)
	catalog := catalogOf(t, stepBlock{err: errors.New("the tool is gone")})
	run := runner.New(runner.Options{Graph: graph, Catalog: catalog, Journal: log, Clock: ticks()})
	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Both nodes are the same block here, so "fix" fails too and there is no
	// second error edge: the run ends where the fallback ran out.
	if got.Status != "error" || got.Node != "fix" {
		t.Fatalf("result = %+v", got)
	}
	if len(log.lines) != 2 || log.lines[0].Outcome != "error" {
		t.Fatalf("entries = %+v", log.lines)
	}
	if log.lines[0].Detail == nil || *log.lines[0].Detail != "the tool is gone" {
		t.Fatalf("detail = %v", log.lines[0].Detail)
	}
}

// An exit is an ending with a reason, not a failure: no error edge is offered,
// because a fallback that swallowed the code would leave the caller with 1.
func TestAnExitEndsTheRunWithItsCodeAndSkipsTheErrorEdge(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 1), node("fix", 1)},
		[]flow.Edge{{From: "a", To: flow.End}, {From: "a", To: "fix", OnError: true}, {From: "fix", To: flow.End}},
		nil,
	)
	catalog := catalogOf(t, stepBlock{exit: &flow.Exit{Code: 4, Message: "rejected"}})
	run := runner.New(runner.Options{Graph: graph, Catalog: catalog, Journal: log, Clock: ticks()})
	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.ExitCode == nil || *got.ExitCode != 4 || got.Detail != "rejected" {
		t.Fatalf("result = %+v", got)
	}
	if len(log.lines) != 1 || log.lines[0].Outcome != "error" {
		t.Fatalf("entries = %+v", log.lines)
	}
}

func TestAStartIsRefusedBeforeAnyEntryIsWritten(t *testing.T) {
	cases := []struct {
		name  string
		graph *flow.Graph
		env   flow.Env
	}{
		{
			name:  "a cap that is zero",
			graph: graphOf([]flow.Node{node("a", 0)}, []flow.Edge{{From: "a", To: flow.End}}, nil),
		},
		{
			name: "a cap whose parameter the run does not have",
			graph: graphOf(
				[]flow.Node{{Name: "a", Kind: "step", MaxVisits: flow.Cap{Param: "rounds", Add: 1}}},
				[]flow.Edge{{From: "a", To: flow.End}}, nil),
		},
		{
			name: "a cap that a parameter makes zero",
			graph: graphOf(
				[]flow.Node{{Name: "a", Kind: "step", MaxVisits: flow.Cap{Param: "rounds"}}},
				[]flow.Edge{{From: "a", To: flow.End}}, nil),
			env: flow.Env{Params: flow.Params{"rounds": 0}},
		},
		{
			name:  "a kind for which the catalog has no block",
			graph: graphOf([]flow.Node{{Name: "a", Kind: "widget", MaxVisits: flow.Cap{Add: 1}}}, []flow.Edge{{From: "a", To: flow.End}}, nil),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log := &memJournal{}
			run := runner.New(runner.Options{
				Graph: c.graph, Catalog: catalogOf(t, stepBlock{}), Journal: log, Env: c.env, Clock: ticks(),
			})
			if _, err := run.Run(context.Background()); err == nil {
				t.Fatal("want a refusal")
			}
			if len(log.lines) != 0 {
				t.Fatalf("a refused run wrote %d entries", len(log.lines))
			}
		})
	}
}

// A flow whose node needs the baseline and a run that has none: refused, so
// that the node does not first spend a model call and then find out.
func TestARunIsRefusedWhenANodeNeedsABaselineAndThereIsNone(t *testing.T) {
	log := &memJournal{}
	graph := graphOf([]flow.Node{node("a", 1)}, []flow.Edge{{From: "a", To: flow.End}}, nil)
	run := runner.New(runner.Options{
		Graph: graph, Catalog: catalogOf(t, baselineBlock{}), Journal: log, Clock: ticks(),
	})
	if _, err := run.Run(context.Background()); err == nil {
		t.Fatal("want a refusal")
	}
	if len(log.lines) != 0 {
		t.Fatalf("a refused run wrote %d entries", len(log.lines))
	}
}

func TestADeltaTheFlowDoesNotDeclareIsANodeError(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 1)},
		[]flow.Edge{{From: "a", To: flow.End}},
		map[string]flow.Field{"note": {Type: flow.String, Default: ""}},
	)
	catalog := catalogOf(t, stepBlock{delta: flow.Delta{"nope": "x"}})
	run := runner.New(runner.Options{Graph: graph, Catalog: catalog, Journal: log, Clock: ticks()})
	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" {
		t.Fatalf("result = %+v", got)
	}
}
```

Dazu zwei Helfer, die dieselbe Datei trägt: `fieldIs(name, want string) flow.Condition` — eine winzige `flow.Condition`-Implementierung, die `state.Fields[name] == want` prüft, und `baselineBlock`, ein `stepBlock` mit `NeedsBaseline() bool { return true }`. Beide gehören in `runner_test.go`, nicht in eine eigene Hilfsdatei.

- [ ] **Step 2: Write the failing tests — gates**

`internal/runner/gate_test.go`, mit einem `gateBlock`, der beim ersten Lauf `Result{Question: …}` liefert und mit gesetztem `env.Answer` ein Delta:

| Test | prüft |
|---|---|
| `TestAGateWritesOnePausedEntryAndEndsPaused` | Eintrag `paused`, `detail` ist die Frage, 0 Tokens, `Result.Status == "paused"`, `Result.Question` gesetzt |
| `TestASecondPassAtTheSameVisitWritesNoSecondPausedEntry` | zweimal `Resume(nil)`: das Journal hat danach genau einen `paused`-Eintrag |
| `TestAnAnswerBelongsToTheVisitNotToTheNode` | Tor auf einem Zyklus: erster Durchgang beantwortet, zweiter pausiert erneut; die zweite Antwort landet auf dem zweiten Eintrag, nicht auf dem ersten |
| `TestTheAnswerEntryIsOkWithZeroSecondsAndItsDetail` | `outcome == "ok"`, `seconds == 0`, `detail == "answered: yes"` |
| `TestResumeWithAnAnswerAndNoWaitingGateIsAnError` | Meldung `no gate is waiting for an answer` |
| `TestAnAnswerTheGateRefusesLeavesItOpen` | der Baustein liefert `flow.ErrInvalidAnswer`: `Resume` gibt einen Go-Fehler zurück, das Journal bekommt **keinen** Eintrag, und ein zweites `Resume(nil)` findet das Tor noch offen |

Der letzte ist der Grund, warum `flow.ErrInvalidAnswer` überhaupt existiert: eine ungültige Antwort ist kein Knotenfehler, nimmt keine Fehlerkante und verbraucht den Besuch nicht.

- [ ] **Step 3: Write the failing tests — retracing, resume and replay**

`internal/runner/retrace_test.go`:

| Test | prüft |
|---|---|
| `TestAResumeDoesNotRunWhatTheJournalCovers` | Zähler im Baustein: nach `Resume` ist der erste Knoten nicht erneut gelaufen |
| `TestRetracingTakesTheLatestOkEntryForNameAndInput` | zwei `ok`-Einträge zum selben Schlüssel, der jüngere gilt |
| `TestRetracingStopsWhereTheJournalStops` | ab dem ersten Knoten ohne Eintrag wird wieder gearbeitet, und der Baustein läuft |
| `TestARetracedDeltaArrivesInTheDeclaredType` | ein Journal mit `{"n": 2}` als `json.Number`: der Zustand hält danach `int(2)`, nicht `json.Number` |
| `TestReplayRunsNoNode` | Zähler bleibt 0, und das Journal wächst nicht |
| `TestReplayOfANodeWithoutAnEntryIsAnError` | Meldung `node "b" is not in the journal`, **keine** Fehlerkante, obwohl der Knoten eine hat |
| `TestReplayRefusesAnAnswer` | Meldung `a replay cannot take an answer; resume the run instead` |
| `TestReplayRefusesARunThatWaitsAtAGate` | abgelehnt |
| `TestAChangedDefinitionWarnsAndCarriesOn` | der Eintrag trägt einen anderen `definition_hash`: `Warn` wird mit dem Knotennamen gerufen, `Status` bleibt `done` |
| `TestAnEntryWithoutADefinitionHashWarnsAboutNothing` | ein Journal aus der Python-Laufzeit: `Warn` wird nicht gerufen |

Der vorletzte ist die Spec-Zeile „Abweichung wird gemeldet, nie abgelehnt": Eine Anweisung am Tor zu verbessern soll erlaubt bleiben.

- [ ] **Step 4: Run the tests to verify they fail**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-runner" && go test ./internal/runner/...`
Expected: FAIL — `undefined: runner.New`.

- [ ] **Step 5: Implement**

`runner.go` trägt die Typen, `New`, `Run`, `Resume` und die drei Startprüfungen. `walk.go` trägt den Gang. Der Ablauf, Schritt für Schritt:

```go
// walk is the whole behaviour contract in one loop. It is one function on
// purpose: every early return is a way a run can end, and they read best next
// to each other.
func (r *Runner) walk(ctx context.Context, state flow.State, answer *pendingAnswer) (Result, error) {
	name := r.graph.Start
	for name != flow.End {
		node := r.node(name)
		state = withVisit(state, name)

		// The ceiling is checked before the block runs, and the entry it
		// writes carries 0 tokens and 0 seconds: nothing ran.
		if over, limit := r.overCap(node, state); over {
			detail := fmt.Sprintf("node %q exceeded max_visits=%d", name, limit)
			// write error entry, return Result{Status: "error", Detail: detail}
		}

		hash := inputHash(name, state)

		// Matched on the pause's own key, not on the node's name: a gate on a
		// cycle pauses once per pass, and an earlier pass is already answered.
		// Keying on the name would spend the answer there, where retracing
		// short-circuits before it is even read.
		var given *string
		if answer != nil && answer.key == hash {
			given = &answer.text
			answer = nil
		}

		if r.retracing {
			// The most recent *successful* entry, not the most recent one: a
			// visit limit and a gate's pause both write a non-ok entry under
			// the key of an entry that succeeded.
			if cached, ok := journal.Lookup(r.entries, name, hash, "ok"); ok {
				r.warnOnChangedDefinition(node, cached)
				next, err := r.merge(state, cached.Delta)  // through flow.Coerce
				…
				continue
			}
			if r.replay {
				// Not an error outcome: an error outcome would be offered the
				// node's error edge, and taking a fallback the original run
				// never took would make a broken replay look like a run that
				// handled a failure.
				return Result{Status: "error", Node: name,
					Detail: fmt.Sprintf("node %q is not in the journal", name)}, nil
			}
			// A resume has caught up. Everything from here is new work.
			r.retracing = false
		}

		started := r.clock()
		result, err := r.block(node).Run(ctx, r.graph, node, state, r.env(given))
		seconds := r.clock().Sub(started).Seconds()

		switch {
		case errors.Is(err, flow.ErrInvalidAnswer):
			// Not a node failure: no entry, no error edge, no visit spent. The
			// gate stays open and the caller is told what the choices are.
			return Result{}, err
		case err != nil:
			// write error entry with seconds and the message
			// take the error edge if there is one, otherwise end
		case result.Exit != nil:
			// write error entry with the message; end with the code, and offer
			// no error edge (an exit is an ending, not a failure)
		case result.Question != nil:
			// only when this very visit is not already recorded as paused: a
			// run someone checked on ten times would otherwise read as ten
			// pauses
		default:
			// write ok entry; merge the delta; take the next edge
		}
	}
	return Result{Status: "done", State: state}, nil
}
```

Was das Schreiben angeht: **eine** Stelle baut den `journal.Entry`, und sie kehrt bei `r.replay` sofort zurück, ohne zu schreiben — so entkommt ihr kein Pfad, weder Deckel noch Pause noch Fehler. Die Felder:

| Feld | Wert |
|---|---|
| `Node`, `Kind` | `node.Name`, `node.Kind` |
| `InputHash` | der Hash über Felder und Besuche |
| `Delta` | das Delta des Bausteins, leer bei Pause, Fehler und Deckel |
| `Outcome` | `"ok"`, `"paused"` oder `"error"` |
| `Tools`, `Effort` | `Definition.Profile` und `Definition.Effort`, jeweils nil, wenn leer |
| `Tokens`, `Seconds` | vom Baustein und von der Uhr; 0 und 0 beim Deckel, 0 Sekunden beim Antworteintrag |
| `Detail` | Frage, Fehlermeldung, `"answered: <antwort>"` oder nil |
| `Model` | `Result.Model`, nil wenn leer |
| `DefinitionHash` | immer gesetzt, aus `journal.DefinitionHash(node.Raw, def.Text, def.Model, def.Effort, def.Tools)` |

`Resume(ctx, answer)`:

```
if r.replay && answer != nil      → Result{}, error "a replay cannot take an answer; resume the run instead"
startChecks()                     → error
r.entries = journal.Entries()
r.retracing = true
if answer == nil                  → walk(initialState, nil)
gate := journal.Pending()
if gate == nil                    → Result{}, error "no gate is waiting for an answer"
if r.replay                       → Result{}, error (a replay refuses a run that waits at a gate)
walk(initialState, &pendingAnswer{key: gate.InputHash, text: *answer})
```

`Run(ctx)`:

```
startChecks()                                → error
r.retracing = r.replay
if r.replay:
    r.entries = journal.Entries()
    if journal.Pending() != nil              → Result{}, error
        // "this run waits at a gate; answer it with resume"
        // A replay reproduces a run that ended. One that is still open has no
        // ending to reproduce, and retracing it would stop at the pause and
        // report it as the replay's own result.
walk(initialState, nil)
```

`doc.go` bekommt einen Absatz: was ein Lauf, ein Fortsetzen und ein Nachgehen sind, und der Satz, warum der Runner den Graphen nicht prüft.

- [ ] **Step 6: Run the tests with coverage**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-runner" && go test -cover ./internal/runner/...`
Expected: `ok … coverage: 100.0% of statements`.

- [ ] **Step 7: gofmt**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-runner" && gofmt -l internal/runner`
Expected: keine Ausgabe.

- [ ] **Step 8: Commit**

```
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-runner" && git add internal/runner && git commit -F <message-file>
```

Betreff: `Walk a graph, retrace a journal and stop at a gate`

---

### Task 5: Prüfen und Zusammenführen (Orchestrator)

**Worktree:** `C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness`

- [ ] **Step 1: Read each lane's diff against its file list**

Für jede der vier Lanes:

Run: `git -C "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" diff --stat feature/agent-harness..ulflow/<lane>`
Expected: nur die Dateien, die der Task unter **Files** nennt. Etwas anderes ist ein Befund und wird berichtet, nicht stillschweigend übernommen.

- [ ] **Step 2: Merge the four lanes, one command each**

Run: `git -C "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" merge --no-ff ulflow/loader`
Expected: sauber. Dann `gate-exit`, `agent`, `runner` in dieser Reihenfolge. `gate-exit` und `agent` fassen dasselbe Paket an; ein Konflikt hier heißt, dass eine der beiden eine Datei angelegt hat, die ihr nicht gehört.

- [ ] **Step 3: Run everything**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && go test -cover ./...`
Expected: alles grün. `internal/blocks` steht jetzt zum ersten Mal als Ganzes da und muss 100 % zeigen.

- [ ] **Step 4: Prove the three lanes fit together**

Der Lader hat gegen Testbausteine geprüft, der Runner gegen handgebaute Graphen. Ob der echte Lader den echten Bausteinen genügt, hat bis hierher niemand gemessen. Ein Test in `internal/blocks` wäre ein Importzyklus rückwärts; er gehört zu `internal/flowload` und wird hier nachgetragen:

`internal/flowload/blocks_test.go`:

```go
package flowload_test

import (
	"testing"

	"github.com/xidus90/ultra-loom/internal/blocks"
	"github.com/xidus90/ultra-loom/internal/flow"
	"github.com/xidus90/ultra-loom/internal/flowcfg"
	"github.com/xidus90/ultra-loom/internal/flowload"
)

// The lanes were built apart on purpose; this is where they meet. The planning
// flow passes all seven stages against the blocks a real run uses, not against
// the doubles the loader lane tested with.
func TestThePlanningFlowLoadsAgainstTheRealBlocks(t *testing.T) {
	real, err := flow.NewCatalog([]flow.Block{blocks.Agent{}, blocks.Gate{}, blocks.Exit{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := flowload.Load("testdata/planning/planning.toml", real, flowcfg.Config{})
	if err != nil {
		t.Fatalf("the planning flow does not load:\n%v", err)
	}
	if len(graph.Nodes) != 12 {
		t.Fatalf("%d nodes", len(graph.Nodes))
	}
}
```

Scheitert er, ist die Ursache fast sicher ein Schlüssel, den der echte Baustein anders liest als der Testbaustein — der Befund gehört in die Befunde der Welle, und die Reparatur in den Commit dieses Schritts.

- [ ] **Step 5: gofmt over everything that moved**

Run: `cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && gofmt -l internal cmd`
Expected: keine Ausgabe.

- [ ] **Step 6: Commit the joining test**

```
cd "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" && git add internal/flowload/blocks_test.go && git commit -F <message-file>
```

Betreff: `Load the planning flow against the blocks a run uses`

- [ ] **Step 7: Read the remote instead of believing a report**

Run: `git -C "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/agent-harness" ls-remote origin feature/agent-harness`
Expected: der Stand vor dieser Welle. Kein Subagent pusht; wenn hier etwas Neues steht, ist das der Befund.

- [ ] **Step 8: Tear the lane worktrees down**

Vorher in jedem Lane-Worktree die `dmypy`-Daemons aus dessen `.venv` beenden, sonst meldet git beim Löschen „Invalid argument" und das Verzeichnis bleibt liegen.

Run: `ulguard worktree-remove "C:/Users/micro/Documents/#GIT/ultraloom/.worktrees/ulflow-loader"`
Expected: entfernt. Dann die drei anderen, je ein Befehl. Der Harness-Worktree bleibt.

- [ ] **Step 9: Write the findings**

`docs/.superpowers/plans/2026-09-11-ulflow-welle-2-befunde.md`, nach dem Muster der Welle 1: Entscheidungen während der Ausführung, Messungen, was für Welle 3 offen bleibt, kleinere Befunde, Betrieb der Lane-Worktrees. Der Plan für Welle 3 geht davon aus.

---

## Außerhalb dieses Plans

- `cmd/flow` mit `run`, `resume`, `replay`, `show`, `list` und `--option`, die Exit-Codes, das Golden-Journal von Ende zu Ende, die Install-Skripte und der Neubau von `ulflow` und `ulguard` in beiden Ständen — das ist Welle 3.
- Die Ablehnung „no adapter for provider claude yet" vor dem Start. Sie gehört an die Kommandozeile, die entscheidet, welchen `model.Model` sie in die `Env` legt; in M1 legt sie keinen hinein.
- Die vier Punkte, die die Befunde der Welle 1 für Welle 3 notiert haben, vor allem `ulguard hook session-start`, das für einen Go-Lauf `ultraloom resume` ausgibt.
- Die „kleineren Befunde, bewusst offen" aus Welle 1. Sie liegen alle in Paketen, die keine Lane dieser Welle besitzt.
- `worktree.py` und jede andere Änderung an Python. M1 ändert kein Python.
- Adapter für `claude -p` und `agy -p` (M2), die Knotenart `command` (nach dem MVP), der parallele Prüferfächer.

