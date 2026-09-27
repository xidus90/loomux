# loomux G4 — Delta zur Säule-3-Spec

**Datum:** 2026-09-22  
**Stand:** entworfen und vom Nutzer freigegeben am 2026-09-22. Die vier Entscheidungen sind
getroffen (§9); der Plan folgt in zwei Phasen: G4a (Navigation & MCP) und G4b (Blast, Hooks & Verify). **G4a und G4b umgesetzt** (G4b 2026-09-23,
nach dem G4b-Nachtrag). Offen: G4c (Stop-Hook mit Blast-Logik, siehe Roadmap).  
**Berichtigt und für G4b ergänzt durch** [`2026-09-23-loomux-code-g4b-delta.md`](2026-09-23-loomux-code-g4b-delta.md).  
**Ergänzt:** [`2026-09-14-loomux-code-graph-design.md`](2026-09-14-loomux-code-graph-design.md),
[`2026-09-16-loomux-code-g1-delta.md`](2026-09-16-loomux-code-g1-delta.md),
[`2026-09-17-loomux-code-g2-design.md`](2026-09-17-loomux-code-g2-design.md) und
[`2026-09-18-loomux-code-g3-delta.md`](2026-09-18-loomux-code-g3-delta.md). Diese Datei ersetzt die
Säule-3-Spec nicht, sie berichtigt und verengt sie für die Stufe G4.  
**Referenz:** `trailhq/Graft`, Commit `1e352a3`, MIT. Gelesen: `src/graph/traverse.ts`,
`src/graph/traverse-cli.ts`, `src/blast/blast.ts`, `src/blast/diff.ts`, `src/blast/evidence.ts`,
`src/search/grep.ts`, `src/graph/map.ts`, `src/mcp/tools.ts`.  
**Arbeitsort:** Worktree `C:\Users\micro\Documents\#GIT\loomux\.worktrees\code-g4`, Branch `code-g4`,
abgezweigt von `master` `27af73f`.  

## 1. Warum es dieses Delta gibt

§10 der Säule-3-Spec gibt G4 die CLI-Palette, den Blast-Monitor im Post-Edit-Hook und die
Verknüpfung von Code-Symbolen mit Second-Brain-Seiten; das G3-Delta (§8) schiebt die vier übrigen
MCP-Werkzeuge mit ihren CLI-Geschwistern dazu. Das Architektur-Review vom 2026-09-22 hat gezeigt,
dass mehrere Annahmen des ursprünglichen Entwurfs am echten Graphen nicht halten:

- **Der Walk existiert schon.** `internal/code/blast` (G1) ist die Portierung von `traverse.ts`;
  `Index.Reach` ist `impactOfMany`. Ein eigenes Paket `traverse` wäre ein zweiter Walk.
- **Der Go-Graph hat keine Typkanten.** Gemessen am Graphen des Repos (2026-09-22, 2 829 Knoten,
  9 093 Kanten, 3 MB `wiring.json`): `calls` 5 160, `contains` 2 574, `imports` 1 359, sonst
  nichts. Kein `references`, kein `implements`, kein `extends`. Die 175 Knoten der Arten `struct`,
  `type` und `interface` haben über Walk-Relationen keinen einzigen Eingang.
- **Der Blast-Monitor war so beschrieben, dass er immer schweigt** (§3.5).
- **Die Lane `blast-audit` prüfte den falschen Commit und hatte keine rote Bedingung** (§3.7).
- **Der Umfang weicht ab.** Die Zeile G4 in `docs/*/migration.md` nennt Palette und Blast-Monitor,
  §10 der Säule-3-Spec zusätzlich die Brain-Verknüpfung; §8 dort nennt einen Stop-Hook, den keine
  der beiden Stellen G4 zuordnet (§9, E4).

## 2. Zuschnitt

```
internal/code/model/         FileSpans: Symbole einer Datei, nach Span-Start sortiert, Spans einmal geparst
internal/code/blast/         Resolve (resolveSymbol), EdgeWalk mit Grafts Tiefe-1-Semantik, Seeds, Radius
internal/code/diff/          reiner Parser für git --name-status -z und --unified=0, liest io.Reader
internal/code/skeleton/      Signaturen und Typen einer Datei aus den Knoten; kein I/O
internal/code/grep/          Regex über indizierte Dateien; liest über eine injizierte Lesefunktion
internal/code/repomap/       Repo-Karte mit Verzeichnis-Clustern, Hubs, Hotspots und Token-Budget
internal/code/query/         Koordinator: Laden, Frische, git-Aufruf, Lesefunktion, Textberichte
internal/serve/graph/        vier Handler mehr, jeder mit keep(path) des Kanals
internal/mcptools/           vier Werkzeugdefinitionen mehr, elf insgesamt
internal/hooks/post_edit.go  Blast-Monitor als Teil von RunPostEdit, kein eigenes Paket
internal/verify/             Lanes graph-freshness und blast-audit (§3.7, §3.8)
internal/cli/graph.go        callers, blast, grep, skeleton, map, stats
```

**Verworfen gegenüber dem Rohentwurf:**

- **Kein Paket `traverse`.** `Resolve` und `EdgeWalk` kommen nach `blast`, neben `Reach`. Ein
  zweites Paket mit eigenem Walk wären zwei Antworten auf dieselbe Frage.
- **Kein Diff-Parsing in `blast`.** Das Parsen der git-Ausgabe ist ein eigener reiner Parser in
  `internal/code/diff`; `git` ruft nur `query` auf. `blast` bekommt fertige `[]diff.File`.
- **Keine Stats-Kennzahl ohne Paket.** `stats` ist eine Funktion in `query` (Knoten, Kanten je
  Relation, Dateien, Sprachen, Größe von `wiring.json`); ein Paket dafür lohnt nicht.

**Rechner und Koordinator.** Rechner bekommen einen geladenen `*model.Graph` und, wo sie Quelltext
brauchen, eine Funktion `read func(path string) ([]byte, error)`. Sie öffnen selbst keine Datei und
starten keinen Prozess. Das hält sie ohne Dateisystem testbar und gibt dem Privacy-Filter einen
einzigen Ort (§3.10). `query` entscheidet Frische, ruft git und reicht `os.ReadFile` unter der
Wurzel hinein.

## 3. Berichtigungen

### 3.1 Die Tiefe 1 ist nicht `Reach(…, 1)`

`edgeWalk` in `traverse.ts` nimmt für `depth <= 1` den Einzelschritt `callersOf`/`calleesOf`, erst
darüber den BFS. Die beiden unterscheiden sich in drei Punkten, und alle drei sind sichtbar:

| | Tiefe 1 (`callersOf`) | Tiefe > 1 (`impactOfMany`) = `Reach` |
|---|---|---|
| doppelte Kante zum selben Nachbarn | zwei Treffer | ein Treffer |
| Selbstschleife (Rekursion) | die Funktion ist ihr eigener Aufrufer | Start nie eigener Treffer |
| Dateiknoten als Start | nur die Kanten des Dateiknotens | Dateiknoten und alle seine Symbole |

**Es gilt:** `EdgeWalk` bildet das nach. Tiefe 1 ist ein Kantenscan über das Index, nicht `Reach`.

### 3.2 `Resolve` folgt dem Code, nicht dem Kommentar

Portiert wird `resolveSymbol` in drei Stufen: Name oder Id-Suffix `#q`/`.q` ohne Beachtung der
Groß- und Kleinschreibung, mit `stripOrdinals` an **jeder** Segmentgrenze (`C~2.m`); dann das letzte
Punkt-Segment als bloßer Name; dann Dateiknoten nach Name, Pfad oder Pfadsuffix. Mehrdeutigkeit ist
kein Fehler, alle Treffer in Graphreihenfolge.

- **`--in` ist ein segmentweiser Präfix**, obwohl Grafts Kommentar „Substring" sagt, und ein
  Präfix, unter dem kein Knoten liegt, ist ein Fehler (`assertPrefixIndexed`). Die Regel gibt es
  schon: `lexicon.UnderPrefix`.
- **Go-Paketfilter (Entscheidung E3):** Enthält `query` einen Punkt (`<pkg>.<symbol>`), prüft
  `Resolve` zuerst, ob `<pkg>` dem deklarierten Paketnamen oder dem letzten Verzeichnissegment eines
  Knotens namens `<symbol>` entspricht, und engt die Treffer darauf ein. Erst wenn das nichts
  liefert, greift Stufe 2 (letztes Punkt-Segment über alle Pakete). Das verhindert, dass
  `hooks.Write` alle 15 `Write`-Funktionen im Repo trifft.
- **Dateinamen-Schutz:** Eine Datei mit bekannter Dateiendung (`.go`, `.ts`, `.py` etc.) wird nicht
  durch ein Symbol namens `go` gekapert; Stufe 3 greift bei Dateiendungen vor Stufe 2.
- **Tiefe über MCP:** `"all"` und `"full"` sind die Hülle, eine Zahl wird abgerundet, alles unter 1
  und alles Nicht-Zahlige ist 1 (`tools.ts`).

### 3.3 Belege kommen aus dem Quelltext, nicht aus der Kante

`model.Edge` trägt keine Position. Graft findet den Beleg für „A ruft B" heuristisch: die erste
Zeile im Span von A, die B als Wort erwähnt (`evidence.ts`: `referenceLine(path, span, [wordRe(name)], read)`).
Belege brauchen deshalb die Lesefunktion und einen frischen Graphen; ist der Graph für die Datei
nicht frisch, entfällt der Beleg, der Treffer bleibt.

### 3.4 Eine inDegree-Definition

Graft hat zwei: `grep.ts` zählt eingehende **Walk**-Kanten, `changedAreas` in `blast.ts` zählt
**alle** Kanten einschließlich `contains`. Nach der zweiten hat jedes Symbol mindestens 1.

**Es gilt:** inDegree zählt eingehende Walk-Kanten, überall. Die Abweichung für das Hub-Ranking in
`changedAreas` geht mit Begründung in die Paritätsliste. `contains` verschiebt dort fast jedes
Symbol um genau 1 und ändert die Reihenfolge kaum.

**Testdateien:** `_test.go` ist indiziert, 1 763 der 2 829 Knoten (62 %). Das trägt das
Testsignal (§3.6), bläht aber das inDegree-Ranking in `grep` und `repomap` durch Testhelfer auf.
Graft tut dasselbe; paritätstreu, als Befund in der Paritätsliste.

### 3.5 Der Blast-Monitor im Post-Edit-Hook

Der Entwurf ließ den Monitor „bei unfrischem Index oder inDegree 0" schweigen. Beides trägt nicht:

- **Unfrisch ist nach einem Edit immer.** Die gerade bearbeitete Datei weicht per Definition vom
  Fingerprint ab. Ein voller `freshness.Probe` widerspricht außerdem §5.3 der Säule-3-Spec (nur der
  Pfad aus stdin) und kostet allein 24 ms.
- **Zeilen treffen alte Spans.** Der Hook liest heute nur `file_path`, kein `tool_response`; einen
  Diff hat er nicht. Selbst mit Diff lägen die neuen Zeilennummern gegen die Spans des alten
  Graphen.
- **inDegree 0 heißt im Go-Graphen „Typ".** Jede Änderung an einem struct, type oder interface
  bliebe stumm, also gerade die, die am häufigsten Aufrufer bricht (Entscheidung E1).

**Es gilt:**

1. Der Monitor läuft nur für eine Datei, die der Graph kennt und die der Extraktor versteht.
   Fehlt der Graph (`store.Read` mit `os.ErrNotExist`), ist sein Schema veraltet
   (`model.ErrSchemaVersion`) oder stammt er von einem anderen Extraktor (`Meta.Extractor` ≠
   `golang.Version`), schweigt er. `query.Check` läuft im Hook nicht; es extrahiert das ganze Repo.
2. Seeds kommen ohne Zeilen-Mapping zustande: `golang.File` extrahiert die bearbeitete Datei neu
   (0,5 ms), und je Symbol wird der `BodyHash` gegen die Knoten desselben Pfads im Graphen
   verglichen. Seeds sind die **geänderten und die entfernten** Ids; die entfernten zuerst, ihre
   Aufrufer brechen sicher. Neue Symbole haben im alten Graphen keine Aufrufer und seeden nichts.
   Hat sich kein Symbol geändert, aber die Datei (Importe, Konstanten auf Paketebene), ist der
   Dateiknoten der Seed, wie in `blast.ts`.
3. Gemeldet wird `Reach(seeds, In, 1)`, ohne Treffer in derselben Datei. Leer heißt Schweigen.
   Ist ein Seed ein `struct`, `interface` oder `type`, wird ein expliziter Hinweis ausgegeben (§9, E1).
4. Die Ausgabe geht in **dasselbe** `hookSpecificOutput.additionalContext`, das `writeSkipped`
   schon schreibt. Zwei JSON-Objekte auf stdout machen die Ausgabe ungültig, und Claude Code meldet
   einen Hook-Fehler. Ist eine Lane rot (Exit 2, Ausgabe auf stderr), entfällt der Blast-Hinweis:
   der rote Befund ist wichtiger, und stderr bleibt dem Befund.
5. Der Monitor blockiert nie und liefert nie Exit 1; ein Fehler beim Lesen des Graphen ist
   Schweigen, nicht Befund.

**Budget.** Gemessen am 2026-09-22, warm, im Prozess, Median aus 15 Läufen, Windows (grobe Uhr):

| Schritt | Zeit |
|---|---|
| `store.Read` (3 MB JSON) | 12 ms |
| `blast.New` | 2 ms |
| `golang.File` (eine Datei) | 0,5 ms |
| `SymbolsInFile`, `Reach` | < 1 ms |
| zum Vergleich: `freshness.Probe` | 24 ms |
| zum Vergleich: `query.Check` | 192 ms |

Die 100 ms sind Eigenzeit **zusätzlich** zu den Lanes, die `RunPostEdit` ohnehin fährt (Budget
50 s). Ohne `Probe` liegt der Monitor auf diesem Repo bei rund 15 ms plus Startboden. Die Kosten
wachsen linear mit `wiring.json`; ab etwa dem Fünffachen dieses Repos reicht das Budget nicht mehr.
Dann braucht es die Pro-Datei-Karten aus §5.3 oder einen Sidecar mit der Rückwärts-Adjazenz. Das ist
nicht G4, aber G4 misst die Größe, ab der es nötig wird, und trägt sie in die Benchmarks ein.

### 3.6 `blast` folgt dem Code von `blast.ts`

- **Der Dateiknoten wird nur geseedet, wenn kein Symbol getroffen wurde.** Grafts Kopfkommentar
  sagt „the file node is seeded too"; der Code sagt das Gegenteil und begründet es. Es gilt der Code.
- **Innermost** heißt hier: ein Symbol, das kein anderes getroffenes Symbol enthält. `grep.ts`
  nimmt dagegen den größten Span-Start. Zwei Regeln, beide paritätstreu, beide in `model.FileSpans`
  als getrennte Funktionen.
- **Ein Walk je geänderter Datei**, zusammengeführt bei der flachsten Tiefe, mit der Menge der
  Dateien, die einen Treffer erreicht haben.
- **Reine Löschung** (`+N,0`) ist die eine Zeile vor der Lücke. **Gelöschte Dateien** werden
  gelistet, nicht gewalkt. Treffer ohne Knoten (unaufgelöster Import) fallen weg.
- **Testsignal** `changed | stale | none | na` wie in Graft. `behavioural` ist in Graft
  `function | method | class`; Go kennt kein `class`. **Es gilt:** `function` und `method`;
  `struct`, `type` und `interface` zählen nicht, aus demselben Grund, aus dem Graft Interfaces
  ausnimmt.
- **Nicht übernommen:** Concepts aus `--deep`, das Benennen per Modell (`--name`), Owners und
  Reviewers aus git. Die Labels fallen auf die deterministische Stufe `symbol` zurück.

### 3.7 Die Lane `blast-audit`

§7.1 der Säule-3-Spec: `loomux graph blast --base HEAD~1`, „erkennt Änderungen an hochgradig
vernetzten Symbolen, für die keine Tests ausgeführt wurden". Nichts davon stimmt:

- Graft macht aus `--base X` den Bereich `X...HEAD`: nur Committetes, gegen die Merge-Base. Im
  Pre-Commit ist `HEAD~1...HEAD` der **vorige** Commit, nicht der, der entsteht.
- Die Zeilenbereiche beziehen sich auf HEAD, der Graph auf den Arbeitsbaum.
- `HEAD~1` fehlt beim ersten Commit und in flachen CI-Klonen.
- Das Testsignal misst „eine Testdatei, die diesen Bereich erreicht, ist im Diff geändert", nicht
  „Tests wurden ausgeführt".
- Graft hat keine Schwelle. Eine Lane ohne rote Bedingung meldet immer grün.

**Es gilt:**

- `graph blast` ohne `--base` vergleicht den Arbeitsbaum mit HEAD und fällt bei sauberem Baum auf
  `HEAD~1...HEAD` zurück, wie Graft. Mit `--base X` gilt `X...HEAD`.
- Die Lane im Pre-Commit vergleicht den **Index** mit HEAD (`git diff --cached`), also genau das, was
  committet wird. Gibt es kein HEAD (Root-Commit) oder ist der Index leer, ist die Lane `na`, nicht rot.
- **Rote Bedingung (Entscheidung E2):** Die Lane wird rot (Exit 1), wenn mindestens ein geänderter
  Seed $\ge N$ eingehende Walk-Kanten hat (`inDegree >= N`) **und** das Testsignal `none` oder `stale`
  ist. $N = 3$ Standard im Go-Preset, überschreibbar in `[verify]`.

### 3.8 Die Lane `graph-freshness`

`graph check` kostet auf diesem Repo 192 ms, im Pre-Commit einmal je Commit; das ist tragbar. In CI
ist der Graph git-ignoriert und fehlt. **Es gilt:** fehlender Graph ⇒ Lane übersprungen mit Meldung,
nicht rot (§5.3 der Säule-3-Spec: „nicht initialisiert"). Fremder oder veralteter Graph und Drift ⇒
rot. Die Lane läuft vor `blast-audit`, denn ein Blast gegen einen driftenden Graphen seedet falsche
Symbole.

### 3.9 `repomap` und Belege gegen Graft verifiziert

`src/graph/map.ts` und `src/blast/evidence.ts` wurden vollständig gegen Graft `1e352a3` gelesen:

- **Repo-Map-Regeln (`map.ts`):**
  - Standard-Grenzwerte: `DEFAULT_MAX_DIRS = 16`, `DEFAULT_HUBS_PER_DIR = 3`, `DEFAULT_HOTSPOTS = 12`.
  - **Monolith-Verfeinerung:** `SPLIT_THRESHOLD = 0.6`. Nimmt ein einzelnes Verzeichnissegment mehr
    als 60 % aller Dateiknoten ein (z. B. ein flaches `internal/`), wird dieses Verzeichnis ein
    Segment tiefer aufgespalten (`internal` $\to$ `internal/code`, `internal/cli`, …).
  - Hubs und Hotspots sortieren nach `inDegree` absteigend, Ties nach Name aufsteigend, dann Pfad.
    Knoten mit `inDegree == 0` werden verworfen.
- **Diff-Belege (`evidence.ts`):**
  - `MAX_EVIDENCE = 4`: Maximal 4 geänderte Symbole erhalten ihr eigenes Diff-Snippet, der Rest
    wird in der Übersicht gezählt.
  - `MAX_LINES = 6`: Hunks werden nach 6 Zeilen mit Restzeilen-Marker (`+N lines`) gefaltet.
  - `hunksIn(file, span)` filtert exakt die Hunks des Diffs, die innerhalb der Zeilenspanne liegen.

### 3.10 Privacy für die vier neuen Werkzeuge

Das G3-Delta (§3.4) filtert vor dem Ranking. Für G4 gilt dasselbe, fail-closed über `keep(path)`
des Kanals:

- `graph_find_all` liefert rohe Quelltextzeilen und ist das größte Risiko. Die Lesefunktion, die
  `query` hineinreicht, verweigert einen Pfad, den `keep` ablehnt; die Datei wird gezählt, nicht
  gelesen.
- `graph_trace_calls` blendet Treffer in abgelehnten Pfaden aus und zählt sie (`Hidden`), wie
  `Drift.Only`.
- `graph_file_api` auf einen abgelehnten Pfad antwortet wie auf einen unbekannten.
- `graph_repo_map` lässt abgelehnte Verzeichnisse und ihre Hubs weg.

## 4. Schnittstellen

```go
// internal/code/model
type SymbolSpan struct{ Node *Node; From, To int }
func FileSpans(g *Graph) map[string][]SymbolSpan // einmal je Graph, nach From sortiert
func Enclosing(spans []SymbolSpan, line int) *Node       // grep.ts: größter Start
func Innermost(spans []SymbolSpan, from, to int) []*Node // blast.ts: enthält kein anderes

// internal/code/blast
func Resolve(g *model.Graph, query, in string) ([]*model.Node, error)
func (x *Index) EdgeWalk(start *model.Node, dir Direction, depth Depth) []Hit
func (x *Index) InDegree(id model.NodeID) int // nur Walk-Relationen
func Radius(g *model.Graph, x *Index, changed []diff.File, depth Depth) Report

// internal/code/diff
func ParseNameStatus(r io.Reader) ([]File, error) // -z, R/C mit drei Feldern
func ApplyHunks(files []File, r io.Reader) error  // --unified=0, Post-Image-Zeilen
```

Die genauen Formen von `skeleton`, `grep` und `repomap` legt der Plan fest; verbindlich ist hier nur
die Regel aus §2: Graph und Lesefunktion rein, Ergebnisstruktur raus, kein eigenes I/O.

## 5. Paritätsfallen für `parity/code-g4.md`

Zusätzlich zu §3:

- **RE2 statt JavaScript-RegExp.** Keine Lookarounds, keine Rückverweise. Ein solches Muster ist ein
  Fehler mit eigener Meldung, kein leeres Ergebnis. `fixed` ist `regexp.QuoteMeta`, `ignore_case`
  ist `(?i)`.
- **Kappung auf 160 Zeichen:** JavaScript zählt UTF-16-Einheiten; Go schneidet an Rune-Grenzen und
  zählt Runen.
- **Sortierung** der Grep-Gruppen stabil (`sort.SliceStable`): inDegree absteigend, dann Pfad.
- **Treffer über `maxHits`** werden weitergezählt (`truncated.hits`), nicht abgebrochen; nicht
  lesbare Dateien zählen in `truncated.files`.
- **CRLF:** Graft teilt an `\n` und trimmt; `$` trifft vor einem `\r` nicht. Go verhält sich gleich;
  festhalten, weil es unter Windows überrascht.
- **`diff`-Kappungen:** 24 Zeilen je Hunk, 200 je Datei; die Bereiche bleiben vollständig.
- **`-c core.quotePath=false`** beim git-Aufruf, sonst kommen Pfade mit Umlauten oktal maskiert.

## 6. Nachweise

- `EdgeWalk`: je ein Fall für doppelte Kante, Selbstschleife und Dateiknoten auf Tiefe 1 und auf
  Tiefe 2 (§3.1).
- `Resolve`: `Cache.get` gegen `#Cache.get`, Ordinal auf innerem Segment, Stufe 2 mit Paket-Präfix,
  Dateistufe, `--in` auf nicht indiziertem Präfix ⇒ Fehler.
- Blast-Monitor: Edit an einem Funktionsrumpf mit Aufrufer in anderer Datei ⇒ Hinweis; Edit an
  einer entfernten Funktion ⇒ Hinweis mit ihren Aufrufern; Edit an struct/type/interface ⇒ Hinweis
  über unverknüpfte Typkanten; Edit ohne Aufrufer ⇒ nichts auf stdout; fehlender Graph ⇒ nichts;
  rote Lane ⇒ kein Hinweis und stdout bleibt gültiges JSON oder leer.
- `diff`: Rename mit drei Feldern, `+N,0`, Pfad mit Leerzeichen und Umlaut, Datei ohne Hunks.
- `blast-audit` ohne HEAD ⇒ `na`; rot bei fehlenden Tests und $\ge N$ Kanten; `graph-freshness` ohne
  Graph ⇒ übersprungen.
- Golden-Files unter `testdata/cases/graph/` für `callers`, `blast`, `grep` und `map` auf einem
  Mini-Repo.

## 7. Tor, Mutationen, Messung, Doku

**Tor:** unverändert. **Mutationen:** `loomux dev mutants internal/code/blast internal/code/diff
internal/code/grep`; Überlebende in `parity/code-g4.md`.

**Messung:** ein datierter Eintrag in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`: der
Post-Edit-Hook kalt und warm vor und nach dem Monitor auf diesem Repo, und die Größe von
`wiring.json`, ab der der Monitor die 100 ms reißt (§3.5).

**Doku:** `README.md`, `README.de.md`, `docs/*/cli-reference.md` (sechs Befehle), `docs/*/migration.md`
(Zeile G4 und die Fähigkeiten „Post-Tool Blast Monitor" und „Symbol-gekoppelter Grep" nach
Abschluss), der Kopf der Säule-3-Spec (Stand und Verweis auf dieses Delta), und §7.1 der Säule-3-Spec
verweist auf §3.7 hier.

## 8. Was G4 offen lässt

- Pro-Datei-Karten oder Rückwärts-Adjazenz als Sidecar für große Repos (§3.5).
- Typkanten im Go-Extraktor (`extract/golang`), ausgelagert auf Folge-Stufe (§9, E1).
- Concepts, Modell-Labels, Owners und Reviewers aus `blast.ts` (§3.6).

## 9. Getroffene Entscheidungen

**E1 — Typänderungen im Blast: Option (b).**
Der Go-Graph hat keine Kanten auf Typen. Ein Edit an einem struct, interface oder type hat in v1 keine
Aufrufer. Der Blast-Monitor gibt bei einem geänderten Typ-Seed ausdrücklich den Hinweis:
`[graph] Modified struct/interface/type: type coupling not wired in graph v1 (check references via grep)`
aus, anstatt stumm zu bleiben. Eine vollständige Typverknüpfung im Extraktor wird nicht in G4
erzwungen, um bestehende `wiring.json`-Instanzen stabil zu halten.

**E2 — Wann `blast-audit` rot wird: Schwellenwert $N \ge 3$ mit `stale`/`none`.**
Die Lane wird rot (Exit 1), wenn ein geänderter Bereich das Testsignal `none` oder `stale` hat
**und** mindestens ein Seed dieses Bereichs mindestens $N$ eingehende Walk-Kanten besitzt (`inDegree >= N`).
Standardwert ist $N = 3$ im Go-Preset, konfigurierbar in `.loomux/config.toml` unter
`[verify.project.blast_audit] threshold = N`. Fehlt HEAD oder ist der Index leer, bleibt die Lane `na`.
**Überholt durch E2′ im G4b-Nachtrag:** kein Konfigurationsschlüssel; das Go-Preset ruft
`check blast-audit --cached --threshold 5` (entschieden 2026-09-23 nach der Messung).

**E3 — Paketname in `Resolve`: Go-spezifischer Paketfilter aktiviert.**
Enthält `query` einen Punkt (`<pkg>.<symbol>`), prüft `Resolve` zuerst, ob `<pkg>` dem Paketnamen oder
dem letzten Verzeichnissegment eines Knotens namens `<symbol>` gleicht. Nur wenn kein Knoten passt,
fällt die Suche auf Grafts Standard-Stufe 2 (letztes Punkt-Segment als Bare Name über das gesamte Repo)
zurück. Dateiendungen (`x.go`) sind gegen Namenskaperung geschützt.

**E4 — Umfang von G4: Trennung von Code-Graph und Second Brain.**
G4 beschränkt sich strikt auf den Code-Graphen (Palette, vier MCP-Werkzeuge, Blast-Monitor, die zwei
Verify-Lanes). Die Verknüpfung von Code-Symbolen mit Second-Brain-Seiten (`internal/brain/wiki`) wird
eine eigene spätere Stufe nach Stufe 3. Der Stop-Hook aus §8 teilt sich die Logik mit `blast-audit`
und läuft im Profil `stop` der Prüfkette.
