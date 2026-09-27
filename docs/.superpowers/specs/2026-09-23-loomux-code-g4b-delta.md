# loomux G4b — Nachtrag zum G4-Delta

**Datum:** 2026-09-23  
**Stand:** entworfen und vom Nutzer freigegeben am 2026-09-23 (Brainstorming, Entscheidungen in
§8); nach einem Review gegen den Code am selben Tag berichtigt (Refresh-Status, ein Job je Stack,
Grenzen der Lane, Dateiknoten, E2 erst nach Messung); nach der Umsetzung am selben Tag an vier
Stellen dem Code nachgezogen (Belege über Überlappung in §3, Edit-Scope und Profil `stop` in §4,
Basis der Messung E2′).  
**Ergänzt:** [`2026-09-22-loomux-code-g4-delta.md`](2026-09-22-loomux-code-g4-delta.md). Diese Datei
ersetzt das G4-Delta nicht; sie berichtigt es dort, wo es am Code von `master` `e3cab29c` nicht
hält, und legt den Umfang der Phase G4b fest. Wo beide sich widersprechen, gilt diese Datei.  
**Arbeitsort:** Branch `feat/code-g4b`, abgezweigt von `master` `e3cab29c`. Der Arbeitsort im Kopf
des G4-Deltas (`.worktrees/code-g4`, `27af73f`) und die Branches `code-g4*` sind Reste von G4a vor
dem Squash und werden nicht weiterverwendet.

## 1. Warum es diesen Nachtrag gibt

Das G4-Delta wurde vor G4a geschrieben. Beim Nachrechnen gegen den heutigen Code halten fünf
Stellen nicht:

- **Die Lanes passen nicht ins `[verify]`-Schema.** `Kinds()` kennt nur `lint`, `types`, `test`,
  `coverage` (`internal/verify/schema.go`). Lanes namens `blast-audit` und `graph-freshness` gibt
  es dort nicht, und `ParseConfig` verweigert den Schlüssel
  `[verify.project.blast_audit] threshold` aus E2 als unbekannt.
- **„na“ und „übersprungen“ haben für eine Befehls-Lane keinen Weg.** Ein Befehl wird nur über
  seinen Exit-Code zu `ok` oder `failed`; `StateNotApplicable` entsteht nur als `Pre`, im Plan.
- **`unavailable` wäre rot.** `CheckVerdict` meldet eine Art, deren Lanes alle `unavailable` sind,
  als „nothing to check“ mit Exit 1. Ein fehlender Graph in CI machte jeden Lauf rot.
- **Die Frische-Lane aus §3.8 wäre fast immer rot.** Den Graphen baut außer einer Abfrage
  (`ask.EnsureFresh`) niemand neu. Wer ändert und committet, hat Drift.
- **Der Stop-Hook aus E4 läuft so nicht.** `precommit` und `stop` laufen beide im `ScopeCheck`;
  eine Lane weiß nicht, in welchem Profil sie läuft, und `--cached` sieht am Zugende fast immer
  einen leeren Index.

Dazu kleinere Punkte:

- `verify.WriteEdit` schreibt die übersprungenen Lanes sofort und privat (`writeSkipped`), der
  Blast-Hinweis braucht aber dasselbe JSON-Objekt (§3.5.4 des G4-Deltas).
- `SymbolsInFile`, das die Budgettabelle in §3.5 nennt, gibt es als `model.SymbolsInFile(g, path)`
  (ein linearer Lauf über die Knoten), nicht in `blast`. Der Monitor braucht keinen neuen Accessor.
- **`ask.EnsureFresh` gibt nichts zurück** (`internal/code/ask/refresh.go`). Gehaltene Sperre,
  fehlgeschlagener Neubau und fehlgeschlagene Probe sind nur `notice`-Text. Ein fremder Extraktor
  ist für `freshness.read` ein fehlender Record und löst einen Neubau aus; ein veraltetes Schema
  sieht `Probe` nicht, weil es `wiring.json` nicht liest.
- **`Reach` expandiert keinen Dateiknoten**, nur `EdgeWalk` tut das. Tabellenzeile 3 in §3.1 des
  G4-Deltas („`Reach`: Dateiknoten und alle seine Symbole") beschreibt `EdgeWalk` ab Tiefe 2, nicht
  `Reach`. Importkanten zeigen außerdem auf eine Repräsentanten-Datei je Paket
  (`resolve.importTarget`), nicht auf die importierte Datei.
- **Mehrere Go-Bereiche** (`detect.searchAreas`: je Verzeichnis erster Ebene mit `go.mod`) ergeben
  je Art einen Job je Bereich, und `verify.Run` startet sie parallel.

## 2. Umfang von G4b

| # | Baustein | Inhalt |
|---|---|---|
| 0 | Messung vorher | Post-Edit-Hook kalt und warm auf diesem Repo, bevor `post_edit.go` sich ändert |
| 1 | `internal/code/diff` | `ParseNameStatus` und `ApplyHunks` wie §4 des G4-Deltas; Kappung 24 Zeilen je Hunk, 200 je Datei |
| 2 | `blast.Radius` | wie §3.6 des G4-Deltas, Testsignal nach §3 hier; dazu `(x *Index) InDegreeWhere(id, keep func(*model.Node) bool) int` für `--skip-test-callers` (E2′) |
| 3 | `query`, CLI `graph blast`, MCP `graph_blast` | §3 hier |
| 3a | Messung E2 | `blast-audit` auf den letzten 50 Commits dieses Repos nachspielen, Rot-Quote je Schwelle; vor Baustein 4 (§4, E2′) |
| 4 | `ask.Refresh`, Art `graph` mit einer Lane im Go-Preset | §4 hier |
| 5 | Blast-Monitor im Post-Edit-Hook | §5 hier |
| 6 | Messung nachher, Mutanten, Doku | §7 des G4-Deltas, ergänzt um §6 hier |

**Nicht in G4b:** der Stop-Hook mit Blast-Logik (§8, E4′), Typkanten im Extraktor, ein Sidecar für
die Rückwärts-Adjazenz, Concepts, Modell-Labels, Owners und Reviewers.

## 3. `graph blast` und `graph_blast`

**Bereich des Diffs** (§3.7 des G4-Deltas, unverändert): ohne `--base` Arbeitsbaum gegen HEAD, bei
sauberem Baum `HEAD~1...HEAD`; mit `--base X` der Bereich `X...HEAD`. Neu ist `--cached`: Index
gegen HEAD, für die Lane. `--base` und `--cached` schließen sich aus. `git` ruft nur `query`, immer
mit `-c core.quotePath=false` und `--relative`, gestartet mit `Dir` = der loomux-Wurzel: `git diff`
nennt Pfade sonst relativ zur Top-Level-Wurzel des Repos, der Graph relativ zur loomux-Wurzel, und
bei einem Bereich in einem Unterverzeichnis passten beide nicht.

**Dateiknoten in `Radius`.** Es gilt §3.6 des G4-Deltas (Dateiknoten nur, wenn kein Symbol getroffen
wurde), gewalkt mit `Reach` **ohne** Expansion auf die Symbole der Datei. Ein Dateiknoten erreicht
damit nur die Importeure, und das nur in der Repräsentanten-Datei eines Pakets. Diese Grenze steht
in der Paritätsliste.

**Testsignal, genau.** Je geänderter Datei (Bereich), in dieser Reihenfolge:

1. `na`, wenn die Datei selbst eine Testdatei ist (Go: `_test.go`) oder kein Seed `function` oder
   `method` ist;
2. sonst die Testdateien $T$ = die Pfade aller Treffer von `Reach(behaviourale Seeds, In, All)`, die
   Testdateien sind — **immer die Hülle**, unabhängig von `--depth`, das nur die gemeldeten Treffer
   begrenzt; ein Test, der über einen Helfer aufruft, erreicht den Bereich auch;
3. `none`, wenn $T$ leer ist; `changed`, wenn eine Datei aus $T$ im selben Diff geändert ist; sonst
   `stale`.

Die Regel „eine Testdatei ist `_test.go`" ist Go-only, solange es nur den Go-Extraktor gibt
(Paritätsliste).

**CLI:** `loomux graph blast [--base X | --cached] [--depth N|all] [--json]`. Die Textausgabe nennt
je Bereich die Seeds, die Treffer mit Tiefe, das Testsignal und bis zu vier Belege (`MAX_EVIDENCE`,
`MAX_LINES` aus §3.9 des G4-Deltas); gelöschte Dateien stehen am Ende. Ein Beleg sind die Hunks, die
den Span des Seeds **überlappen**, nicht nur die ganz darin liegenden wie `hunksIn` in Graft: ein
Hunk über den Rand eines Symbols ändert es trotzdem, und die Seeds kommen über `Innermost` ebenso
aus der Überlappung. Das ist Absicht und steht in der Paritätsliste. `--json` ist die Struktur
von `blast.Report`. Golden-Files unter `testdata/cases/graph/` auf dem Mini-Repo der G4a-Fälle.

**MCP `graph_blast`** (zwölftes Werkzeug). Die Web-OS-Spec setzt es für die Stufe W4 voraus
(`2026-09-14-loomux-web-os-design.md`), spezifiziert war es bisher nirgends.

- Parameter: `scope` (Pflicht), `base` (optional; fehlt es, gilt der Bereich ohne `--base`),
  `depth` (`"all"`/`"full"` ist die Hülle, eine Zahl wird abgerundet, und was danach unter 1 liegt
  oder nicht zahlig ist, ist 1). Kein `cached`; das braucht nur die Lane. `parseDepth` in
  `internal/serve/graph` macht heute `0.5` zu `0` (leerer Walk) und verstößt damit gegen §3.2 des
  G4-Deltas. Der Fehler besteht schon auf `master` und wird auf diesem Branch in einem eigenen
  `fix`-Commit behoben; `graph_blast` nutzt danach dieselbe Funktion wie `graph_trace_calls`.
- Antwort: derselbe Textbericht wie `graph blast` ohne `--json`, wie bei den übrigen Graph-Werkzeugen
  (`CallersReport` & Co.).
- `git` läuft in der Wurzel des Bereichs, den `scope` benennt. Ist sie kein git-Repo oder fehlt
  HEAD, antwortet das Werkzeug mit einem Fehler und eigener Meldung, nicht mit einem leeren Bericht.
- **Privacy, fail-closed über `keep(path)`:** Belege sind rohe Diff-Zeilen, dasselbe Risiko wie
  `graph_find_all`. Eine geänderte Datei in einem abgelehnten Pfad wird gezählt, weder gewalkt
  noch zitiert; Treffer in abgelehnten Pfaden fallen weg und zählen in `Hidden`.

## 4. Die Art `graph`

**Schema.** `Kinds()` wird `lint, types, test, coverage, graph`. `Reserved` kennt zusätzlich
`graph-fresh` und `blast-audit`. Die Profilvorgabe `precommit` bekommt `graph`; `edit` und `stop`
bleiben, wie sie sind. Ein Projekt, das `precommit` in `[verify.profiles]` selbst definiert, nimmt
die Art erst auf, wenn es sie einträgt; `.loomux/config.toml` dieses Repos definiert kein Profil.
Mitzuziehen sind der Usage-Text in `internal/cli/check.go`, der Kommentar „same four kinds" an der
Profilvorgabe `stop` in `schema.go` und die Tests, die die Arten aufzählen.

**Go-Preset:**

```toml
[stack.go.graph]
commands = ["{loomux} check graph-fresh", "{loomux} check blast-audit --cached --threshold 5"]
```

Den Schwellenwert ändert ein Mensch, indem er `commands` in `[verify.go.graph]` ersetzt, **beide**
Einträge: wer nur `blast-audit` schreibt, verliert die Frische. **Das ersetzt den Schlüssel aus E2**;
das Schema bekommt keinen neuen Schlüssel. Der Eintrag ins Preset folgt erst der Messung aus E2′.

**Reihenfolge.** Die Befehle einer Lane laufen nacheinander (die Lane ist nicht `threaded`), aber
`verify.Run` bricht nach einem roten Befehl nicht ab. `blast-audit` läuft also auch nach einem roten
`graph-fresh`, gegen den Graphen auf der Platte; die Lane ist dann ohnehin rot. Das bleibt so.

**Ein Job je Stack, an der Wurzel.** Für die Art `graph` legt `Plan` je Stack genau einen Job an, im
Bereich `.`, auch wenn der Stack mehrere Bereiche hat. Der Graph gehört der loomux-Wurzel; ein Job je
Bereich hieße N parallele Neubauten um dieselbe Sperre und N gleiche Berichte.

**Nur im Check-Scope.** Im `ScopeEdit` legt `Plan` für die Art `graph` gar keinen Job an, wie für
eine Lane ohne Befehl; der Bericht nennt sie dort nicht. Ein Profil `edit` mit `graph` ist erlaubt,
baut aber nie bei einem Edit neu. Im Profil `stop` (auch `ScopeCheck`) ist sie immer
`not-applicable` mit der Notiz „graph lanes need a graph probe“, wenn ein Projekt sie dort einträgt:
`hooks/stop.go` setzt kein `GraphReady`. Die Vorgabe trägt sie nicht ein (E4′).

**Prüfung im Plan.** `PlanEnv` bekommt `GraphReady func(root string) (bool, string)`, nach dem
Vorbild von `ImportReady`, aufgerufen mit `env.Root`. `nil` heißt `not-applicable`, damit die
Aufrufer in `hooks/post_edit.go` und `hooks/stop.go` nichts setzen müssen. Die Prüfung antwortet in
dieser Reihenfolge `false` mit Notiz; die erste kostet keinen git-Aufruf, der Rest zwei:

1. kein `wiring.json` unter `.loomux/state/graph/`,
2. HEAD fehlt oder die Wurzel ist kein git-Repo (`git rev-parse --git-path MERGE_HEAD HEAD` mit
   Exit ≠ 0),
3. die Datei, die dieselbe Ausgabe als `MERGE_HEAD` nennt, existiert (bei einem Merge zeigt
   `--cached` fremde Änderungen als eigene). `--git-path` statt `.git/MERGE_HEAD`, weil `.git` in
   einem Worktree eine Datei ist,
4. der Index ist leer (`git diff --cached --quiet` mit Exit 0; Exit 1 heißt Änderungen, jeder andere
   Exit heißt Fehler und ergibt `false` mit dem Text von git).

Die Lane ist dann `Pre = StateNotApplicable` — **nicht** `StateUnavailable`, das `CheckVerdict` rot
werten würde (§1). Die Prüfung gilt der ganzen Lane.

**Wo die Lane nicht läuft.** Sie ist opt-in und lokal: sie läuft nur, wo jemand den Graphen gebaut
hat. `not-applicable` ist sie in CI und in jedem frischen Klon (`.loomux/state/` ist git-ignoriert),
bei einem manuellen `loomux check precommit` ohne gestagte Änderungen, bei `commit --amend` ohne neue
Änderungen, bei einem Merge und beim Root-Commit.

**`ask.Refresh`.** `EnsureFresh` gibt nichts zurück (§1). Neu ist

```go
type Status int // Clean, Rebuilt
func Refresh(root, extractor string, rebuild Rebuild, wait time.Duration) (Status, error)
```

mit derselben Probe, derselben Sperre und derselben Stale-Regel. `EnsureFresh` wird zur Hülle um
`Refresh(…, 0)` und schreibt die bisherigen `notice`-Texte aus Status und Fehler; ihre Aufrufer in
`query/ask.go` und `query/load.go` bleiben unverändert. Fehler, die ein Aufrufer unterscheiden kann:
`ErrLocked` (Sperre nach `wait` noch gehalten, mit Pfad und Alter der Sperre), ein Fehler der Probe,
ein Fehler des Neubaus.

**`loomux check graph-fresh`.** Ruft `Refresh(…, 30 s)`. Drift, ein fehlender Fingerprint, ein
fremder Extraktor und ein veraltetes Schema von `wiring.json` lösen einen Neubau aus und sind grün
(E7). Rot (Exit 1) ist nur

- ein fehlgeschlagener Neubau,
- ein Fehler der Probe,
- eine Sperre, die nach 30 s noch gehalten wird. Die Meldung nennt Pfad und Alter; eine Sperre älter
  als `lockStale` (eine Stunde) übernimmt `Refresh` wie heute.

Das berichtigt §3.8 des G4-Deltas (Drift ⇒ rot) und die erste Fassung dieses Nachtrags (fremder
Extraktor ⇒ rot). Grenze: der Graph bildet den Arbeitsbaum ab, der Diff der Lane den Index. Bei
teilweise gestagten Dateien können einzelne Seeds daneben liegen; das steht in der Paritätsliste.

**`loomux check blast-audit --cached --threshold N`.** `git diff --cached` gegen HEAD, dann
`blast.Radius`. Rot (Exit 1), wenn ein Bereich das Testsignal `none` oder `stale` hat **und** einer
seiner Seeds `InDegree >= N` hat (E2). Die Ausgabe nennt je rotem Bereich die Seeds mit inDegree und
das Signal. Ohne `--cached` oder mit `--base` verhält sich der Befehl wie `graph blast`; nur die Lane
nutzt `--cached`. `blast-audit` baut den Graphen nie selbst neu (das ist Sache von `graph-fresh`,
und zwei Neubauten liefen um dieselbe Sperre). `--skip-test-callers` zählt für die rote Bedingung
nur eingehende Walk-Kanten aus Nicht-Testdateien (`Index.InDegreeWhere`); der Schalter existiert,
damit E2′ beide Zählweisen am echten Befehl messen kann, und bleibt als Wahl für das Preset.

**E2′ — erst messen.** `InDegree` zählt auch Aufrufer aus Testdateien (§3.4 des G4-Deltas: 62 % der
Knoten sind Testcode). Wer eine gut getestete Funktion ändert, ohne ihren Test anzufassen, bekommt
`stale` und damit Rot. Bevor die Lane ins Go-Preset kommt, spielt Baustein 3a `blast-audit` auf den
letzten 50 Commits dieses Repos nach (je Commit `<c>~1...<c>`, gegen den Graphen von `<c>`, den die
Lane nach `graph-fresh` sieht; die Messung zeigte, dass der Graph des Elternstands neue Dateien nicht
kennt und die Tests desselben Commits nicht enthält) und meldet die
Rot-Quote für N = 3, 5 und 10, mit und ohne Testaufrufer im inDegree. Mit diesen Zahlen entscheidet
der Nutzer Schwelle und Zählweise; ~~bis dahin gilt E2 als vorläufig~~ (entschieden, siehe den nächsten Satz). Die Messung geht in die
Benchmarks. Entschieden am 2026-09-23 nach der Messung in docs/*/benchmarks.md: N = 5, alle Aufrufer (--skip-test-callers änderte nichts).

**Kosten.** Ein Neubau je Commit mit Drift, dazu der Blast und höchstens zwei git-Aufrufe im Plan
(je rund 40 ms). Alles wird gemessen (§6).

## 5. Der Blast-Monitor im Post-Edit-Hook

Es gilt §3.5 des G4-Deltas, mit diesen Festlegungen:

1. **Nur Go-Dateien, die der Graph kennt.** `.go` unter der Wurzel. `store.Read` mit
   `os.ErrNotExist`, `model.ErrSchemaVersion` oder `Meta.Extractor != golang.Version` heißt
   Schweigen. Weder `freshness.Probe` noch `query.Check` läuft im Hook.
2. **Seeds über `BodyHash`.** Die bearbeitete Datei wird gelesen und mit `golang.File` neu
   extrahiert; je Symbol-Id wird ihr `BodyHash` gegen die Knoten desselben Pfads verglichen
   (die Knoten des Graphen mit diesem `Path`). Seeds sind die entfernten Ids, dann die geänderten. Neue
   Symbole seeden nichts. **Der Dateiknoten seedet im Monitor nie**, anders als §3.5.2 des
   G4-Deltas: `Reach` expandiert ihn nicht, und seine eingehenden Kanten sind Importe, die nur in der
   Repräsentanten-Datei eines Pakets ankommen (§1) — der Hinweis hinge davon ab, welche Datei eines
   Pakets bearbeitet wurde. Kein geändertes oder entferntes Symbol — auch ein schon frischer Graph —
   ist Schweigen.
3. **Meldung.** `Reach(seeds, In, 1)` ohne Treffer in derselben Datei; je Treffer
   `Name (Pfad)`, höchstens zehn Zeilen und ein Restzähler. Ist ein Seed `struct`, `interface` oder
   `type`, folgt der Hinweis aus E1, auch ohne Treffer.
4. **Nie ein Befund.** Lesefehler, Parsefehler (eine halb bearbeitete Datei) und ein unpassender
   Graph sind Schweigen. Der Monitor gibt nie Exit 1 und blockiert nie.
5. **Wiederholung.** Der Graph bleibt bis zum nächsten Neubau alt; jeder weitere Edit an derselben
   Datei meldet dieselben Seeds erneut. Das bleibt so und steht in `docs/*/hooks.md`; ein Gedächtnis
   je Datei ist nicht G4b.

**Schnittstelle in `verify`.** `WriteEdit(stdout, stderr io.Writer, outs []Outcome, aside string) int`.
`aside` steht unter den übersprungenen Lanes im **selben** `additionalContext`, und nur, wenn keine
Lane rot ist; `writeSkipped` schreibt weiter nichts, wenn beides leer ist. `RunPostEdit` rechnet den
Hinweis nach `verify.Run` und vor `WriteEdit`, und nur, wenn keine Lane rot ist. `WriteEdit` hat einen
Produktionsaufrufer (`post_edit.go`).

**Ort.** `internal/hooks/blast_monitor.go`: `blastAside(root, rel string, read func(string) ([]byte, error)) string`,
ohne Prozess testbar. `store.Read` bekommt die Wurzel, `read` die Datei.

## 6. Nachweise, Messung, Doku

**Nachweise zusätzlich zu §6 des G4-Deltas:**

- Monitor: Parsefehler ⇒ Schweigen; fremder Extraktor ⇒ Schweigen; frischer Graph ⇒ Schweigen;
  nur Paketebene geändert ⇒ Schweigen.
- `GraphReady`: kein Graph, `MERGE_HEAD`, kein HEAD, leerer Index ⇒ `not-applicable`, und
  `CheckVerdict` bleibt dabei 0; git-Fehler (Exit 128) ⇒ `not-applicable` mit Text; `nil` ⇒
  `not-applicable`.
- `Plan`: Go-Stack mit zwei Bereichen ⇒ ein `graph`-Job im Bereich `.`; `ScopeEdit` ⇒ kein Lauf.
- `Refresh`: sauber ⇒ `Clean`; Drift, fremder Extraktor, altes Schema ⇒ `Rebuilt`; gehaltene Sperre ⇒
  `ErrLocked` nach `wait`; `EnsureFresh` schreibt dieselben Texte wie vorher.
- `graph-fresh`: Neubau fehlgeschlagen ⇒ rot; Sperre ⇒ rot mit Pfad und Alter.
- `blast-audit`: `--base` und `--cached` zusammen ⇒ Fehler.
- `graph blast` in einem Unterverzeichnis-Bereich: Pfade relativ zur loomux-Wurzel.
- `graph_blast`: abgelehnter Pfad wird gezählt, nicht zitiert; Bereich ohne git ⇒ Fehler;
  `depth` 0,5 ⇒ 1.
- `ParseConfig`: ein Profil namens `graph` kollidiert mit der Art.

**Messung** (zusätzlich zu §7 des G4-Deltas): `check graph-fresh` mit und ohne Drift,
`check blast-audit --cached` und die beiden git-Aufrufe von `GraphReady` auf diesem Repo; die
Rot-Quote aus E2′. Die Größe von `wiring.json`, ab der der Monitor die 100 ms reißt, wird an einem
synthetisch vervielfachten Graphen gemessen.

**Doku** (zusätzlich zu §7 des G4-Deltas):

- `docs/*/cli-reference.md`: `graph blast`, `check graph-fresh`, `check blast-audit`, das Werkzeug
  `graph_blast`.
- `docs/*/configuration.md`: die Art `graph`, die Profilvorgabe, das Ersetzen von `commands`.
- `docs/*/hooks.md`: der Blast-Monitor, seine Grenzen (Typen, Wiederholung) und das Zusammenspiel
  mit roten Lanes.
- `docs/*/migration.md`: G4b ✅; die Zeile G4b nennt die Art `graph` statt der Lanes
  `blast-audit`/`graph-freshness`; „elf Werkzeuge" wird zwölf; die Fähigkeiten „Post-Tool Blast
  Monitor" und „Blast-Radius-Engine" folgen ihrer Stufe; eine neue offene Zeile für den Stop-Hook mit
  Blast-Logik.
- Die Fusion-Spec bekommt die neue Stufe „Stop-Hook mit Blast-Logik" **zuerst** (AGENTS.md: eine
  neue Stufe geht erst in die Fusion-Spec, der Plan folgt).
- `parity/code-g4.md` bekommt einen Abschnitt G4b.

## 7. Berichtigungen am G4-Delta, zusammengefasst

| Stelle im G4-Delta | Berichtigt durch |
|---|---|
| Kopf, Arbeitsort | Kopf hier |
| §2, „vier Werkzeugdefinitionen mehr, elf insgesamt" | §3: G4b fügt `graph_blast` hinzu, zwölf |
| §3.1, Tabellenzeile Dateiknoten für `Reach` | §1: das ist `EdgeWalk`, `Reach` expandiert nicht |
| §3.5, Budgettabelle `SymbolsInFile` | §1: liegt in `model`, nicht in `blast` |
| §3.5.2, Dateiknoten als Seed im Monitor | §5.2, kein Dateiknoten im Monitor |
| §3.5.4, dasselbe `additionalContext` | §5, `WriteEdit(…, aside)` |
| §3.7, Lane `blast-audit`, `na` | §4, Art `graph`, `GraphReady` |
| §3.8, Drift ⇒ rot, fehlender Graph ⇒ übersprungen | §4, Neubau über `ask.Refresh`; fehlend ⇒ `not-applicable` |
| E2, `[verify.project.blast_audit] threshold` | §4, `--threshold` im Befehl; E2′ |
| E4, Stop-Hook im Profil `stop` | §8, E4′ |

## 8. Getroffene Entscheidungen

**E5 — Ort der Lanes: eine fünfte Art `graph`.** Verworfen: unter `go.coverage` anhängen (ein roter
Blast wäre ein Coverage-Befund) und eingebaute Befehle ohne Profil (die Kette stünde nicht mehr
ganz in `[verify]`). Unter `go.test` ginge es nicht: `plan.go` ersetzt die Befehle beim Messen.

**E6 — „na": Prüfung im Plan.** Verworfen: Exit 0 mit Meldung (der Bericht zeigte `ok`, wo `na`
gemeint ist) und ein eigener Exit-Code (kollidiert, sobald ein fremdes Werkzeug den Befehl ersetzt).

**E4′ — Stop-Hook: nicht in G4b.** `graph` kommt nicht ins Profil `stop`. Der Stop-Hook mit
Blast-Logik wird eine eigene Stufe (Fusion-Spec zuerst, dann Migrationsplan) und braucht eine Form
Arbeitsbaum-gegen-HEAD, die weiß, dass sie am Zugende läuft.

**E7 — Drift in der Lane: neu bauen, dann prüfen.** Verworfen: rot bei Drift (fast jeder Commit rot)
und nur warnen (Blast gegen einen alten Graphen seedet falsch). Ein fremder Extraktor und ein altes
Schema sind Drift im Sinn dieser Regel.

**E8 — MCP `graph_blast` in G4b.** W4 setzt es voraus; der Handler folgt dem Muster der vier
G4a-Handler.

**E2′ — Die rote Bedingung von `blast-audit` erst nach Messung.** Verworfen: Testaufrufer ohne
Messung ausschließen (weicht von der einen inDegree-Definition ab, ohne Beleg) und E2 unverändert
übernehmen (Risiko „fast jeder Commit rot"). Schwelle und Zählweise entscheidet der Nutzer nach
Baustein 3a. Entschieden am 2026-09-23 nach der Messung in docs/*/benchmarks.md: N = 5, alle Aufrufer (--skip-test-callers änderte nichts).
