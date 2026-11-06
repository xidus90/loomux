# Fusion tgrep und Code-Graph: Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (empfohlen) oder superpowers:executing-plans, um diesen Plan Aufgabe für Aufgabe umzusetzen. Schritte sind mit `- [ ]` markiert.

**Ziel:** `loomux graph grep` bekommt einen Trigramm-Index, der zusammen mit dem Graphen gebaut wird. Das DSH-Plugin wird eine dünne Fassade über `loomux graph grep --json --channel cloud`. tgrep fliegt ganz raus, erst nach einem Benchmark-Gate (Teil C). Kein Rückfall.

**Architektur:** Teil A baut zuerst einen Walk (`internal/code/corpus`) mit Git-Ignore-Auswertung, Größen- und Binärregel und einen Digest, der Einstellungen und Ignore-Eingaben einschließt (Tasks A4a–A4d). Dann baut Teil A in loomux ein neues Paket `internal/code/trigram` ohne Extraktor-Import. Es wird in `graph build` geschrieben und in `graph grep` für Kandidaten genutzt; die Verifikation bleibt `grep.Search`. Teil B ist ein Node-Paket `dsh-loomux` im DSH-Profil, das das Tool-Schema übersetzt, routet und Fehler mit Hinweis abdeckt. Teil C misst gegen tgrep und entfernt tgrep erst nach dem Gate.

**Tech Stack:** Go (go.mod: `go 1.26.0`), `regexp`/`regexp/syntax` (stdlib), Node.js >= 22.19 für das Plugin, JSON.

**Spec:** `docs/.superpowers/specs/2026-11-03-tgrep-loomux-fusion-design.md`

## Global Constraints

- Coverage 100 % je Funktion. Eine Ausnahme nur mit `//coverage:exempt <Grund>` direkt über `func`.
- Kein `init()`. Eingebettete Daten werden beim ersten Gebrauch geladen (`sync.OnceValue`).
- Ein Binary, CGo-frei. Keine neue Abhängigkeit im Go-Teil.
- tgrep ist nur Entwicklungswerkzeug für das Gate (Teil C), keine Laufzeit-Abhängigkeit und kein Rückfall.
- Commit-Nachrichten: Conventional Commits, englisch, ohne Verweis auf Arbeitspapiere (kein Plan-, Task- oder Stage-Name), ohne Modellnennung und ohne `Co-Authored-By`. Ein Thema je Commit.
- **Kein Agent committet, pusht oder öffnet einen Pull Request.** Die Schritte „Commit-Vorschlag“ sind Header für den Menschen, der committet.
- Jede neue Testzeile läuft vor dem Produktivcode rot.
- Vor jedem Commit: `sh ci/gate.sh` grün.
- Keine Abhängigkeit für den Ignore-Parser: reines Go, geprüft gegen `git check-ignore` als Orakel in den Tests. `git` aufrufen ist erlaubt; das ist schon Muster in `internal/brain/vcs/vcs.go:310` und `internal/cli/area.go:260`.
- Der Plan liest `iam_backend` nicht. Der Zugriff ist erst in C2 zu bestätigen (Spec, Frage 15).
- Schlüssel `[grep] max_filesize` wird erst geschrieben, wenn Spec-Frage 18 entschieden ist. Vorher gilt die Vorgabe 67108864 als Konstante, und die CLI-Flags wirken.
- Die Änderung an `.loomux/config.toml` ist nicht Teil dieses Plans.
- `_identities.tsv`, `scratch-edit-unused`, `.tgrep/` und die untracked Arbeitspapiere gehören nicht in einen Commit.
- Dokumentation auf Deutsch in `docs/.superpowers/`; Code, Kommentare, Fehlermeldungen und Flags englisch.

## Review Focus

1. **Äquivalenz:** Index-Pfad und Scan-Pfad liefern gleiche `groups`, `total_hits`, `truncated_hits`. Test in A6.
2. **Kandidaten sind eine Obermenge:** keine Trefferdatei fehlt in der Kandidatenliste. Fuzz-Test in A4.
3. **Nicht-ASCII und `-i`:** kein Index bei Faltungsgefahr. Test in A1.
4. **`never` auf dem CLI-Weg:** `--channel cloud` filtert, ungelesenes Manifest liefert Exit 1. Test in A7.
5. **Deterministische Bytes:** zweimal gebaut, gleiche Bytes. Test in A3.
6. **Routing im Plugin:** Ein Pfad außerhalb der Repo-Wurzel führt nie zu einem stillen Leer, sondern zu einem Fehler mit Hinweis. Endungen sind kein Routingkriterium mehr (vierter Durchgang). Ignorierte, binäre und zu große Dateien sind Zähler im Ergebnis, kein Fehler. Test in B3.
10. **Walk und Ignore:** `Ignored` je besuchtem Pfad gleich `git check-ignore --stdin`; Suchkorpus gleich `git ls-files --cached --others --exclude-standard`, gefiltert mit denselben Regeln; Graphliste gleich dem Differentialtest gegen die alte Logik. Tests in A4a, A4b und A10. Der Vergleich „ignorierte Menge gleich git“ ist falsch: der Walk betritt geprunte Verzeichnisse nicht.
11. **Digest:** Jede Änderung von Einstellung, Ignore-Quelle oder Tracking macht den Index `stale`. Test in A4d.
12. **Gate-Gleichheit:** Dateisätze vor der Messung geprüft (Zahl, Stichprobe, Mengendifferenz) und im Eintrag. Task C2.
7. **Gegenprüfung der Plugin-Seite:** erledigt (Schritt B0). Befunde: Spec, Abschnitt „Stand der Plugin-Seite (belegt)“.
8. **Keine stillen Verluste im Plugin:** Exit 1 ist ein Fehler, nie ein leeres Ergebnis; der stdout-Deckel ist ein Fehler, kein Abschneiden. Tests in B4.
9. **Benchmark-Gate:** gleiche Dateiliste und gleiche Treffer vor dem Vergleich; das Abbruchkriterium steht im Eintrag. Task C3.

## Entscheidungen des Bauenden

Die Spec ließ diese Punkte offen oder der Brief stimmte nicht. Sie gelten für diesen Plan:

- **Index unter `cache/`:** `.loomux/state/graph/cache/trigram.idx`, nicht unter einem eigenen Unterordner des Graphen (Abweichung vom Brief, begründet in der Spec).
- **Digest:** SHA-256 über die kanonischen Einstellungen, die Hashes der Ignore-Quellen und die sortierten Paare `(rel, hash)` des Suchkorpus (Task A4d).
- **Postings auf Dateiebene**, ohne Zeilenpositionen in v1.
- **Index immer gefaltet** (ASCII-Kleinbuchstaben). Nicht-ASCII-Muster mit `-i` fallen auf den Scan zurück.
- **Keine eigene Symboltabelle** im Index; die Knoten des Graphen genügen.
- **`files_searched`** bleibt die Zahl der geprüften Dateien; auf dem Index-Pfad ist das die Kandidatenzahl. Das Plugin benennt das im `render`-Text.
- **`--channel`** ist neu und Vorgabe `local` (heutiges Verhalten). Das Plugin setzt immer `cloud`.
- **Feld `error` im JSON-Modus:** eingeführt (beschlossen, Spec-Frage 9); das Plugin klassifiziert am Code, nicht am Wortlaut.
- **Plugin-Paket:** `dsh-loomux` (entschieden durch den Menschen, Frage 2 der Spec).
- **Walk in `internal/code/corpus`:** ein Walk liefert alle Dateien mit Ignore-Flag und Tracking-Flag. `sourceset.Stat` wird zum Graph-Filter darüber; `goModPaths` bleibt eigener Walk (Spec, A1).
- **Abstieg:** Ein Punkt-Verzeichnis oder ein `skipDirs`-Verzeichnis, das ignoriert ist und keinen getrackten Pfad enthält, wird nicht betreten. Alles andere wird betreten (Spec, „Suchkorpus und Walk“).
- **Ignore-Matcher in reinem Go:** Git-Regeln, Tests mit `git check-ignore` als Orakel. Quellen: nested `.gitignore` über und unter der Wurzel, `.git/info/exclude`, `core.excludesFile` (per `git config --path`). Standardquelle und `core.ignorecase`-Mechanik stehen unter „Offen“ und werden im Test belegt.
- **Tracking-Stand:** `git ls-files -z` beim Build. Der Probe-Pfad vergleicht nur Größe und Änderungszeit von `.git/index`; geänderter Stand löst Neulesen aus.
- **Record `cache/corpus.json`:** zweiter Record neben `fingerprint.json`. Der Graph-Record bleibt unverändert.
- **Binärregel:** NUL-Byte in der ganzen Datei (Spec, „Binärdateien“). UTF-16 gilt als binär.
- **Größe:** Vorgabe 67108864 Byte; `0` und `--no-max-filesize` heißen unbegrenzt. Graphkorpus bleibt bei 1 000 000 Byte.
- **Vergleich gegen den Record:** Ohne Flag ist eine Abweichung Drift; der Rebuild nimmt die Einstellung des Aufrufs. Mit Flag gibt es keinen Rebuild, sondern Scan mit Notiz und `stale`. Messläufe: `--no-refresh` mit denselben Flags wie der Build (Spec, „Vergleich gegen den Record“).
- **Keine `--no-ignore`, keine `--hidden` in v1** (Spec, „CLI“).
- **Fehler bei Lesen:** Suchdateien werden übersprungen und gezählt. Graphdateien brechen den Build wie bisher ab.
- **Kein Rückfall, kein tgrep im Plugin:** entschieden (Spec, Fragen 3 und 10). Task B5 entfällt; die Fehlerfälle stehen in B4.
- **Kein Feld `engine`:** Es gibt keinen zweiten Suchweg. B2 ist entsprechend ohne `engine`.
- **Daemon:** Das Ziel-Plugin startet keinen `tgrep serve` und kennt keine Daemon-Optionen (`autoStartDaemon`, `daemonArgs`, `daemonIdleTimeoutMs`, `daemonReadyTimeoutMs`, `stopDaemonOnExit`). Grund: Z3 und der Befund `index.js:283–367`.
- **Konfiguration:** Das Ziel-Plugin behält `enabled` und `maxLines` (Vorgabe 300, Klammer 20–5000, Z5). `extraArgs` und `maxFileSize` sind tgrep-spezifisch und entfallen.
- **Tool-Beschreibung:** Die Beschreibung nennt heute tgrep (`index.js:755`). Der Ziel-Text ändert den Modell-Prompt; das wird im Ergebnis vermerkt.

## Offene Entscheidungen

- **Rückfall-Engine im Plugin:** entschieden (Spec, Frage 10): kein Rückfall, Fehler mit Hinweis. Verworfen: `rg`, das Stock-Tool, Kandidat A (nicht belegt), tgrep (entfernt).
- **`never` ab Werk auf dem CLI-Weg:** entschieden (Spec, Frage 5): Opt-in über `--channel cloud`, das Plugin setzt es immer.
- **Textscan über nicht abgedeckte Endungen zusammenführen:** entfällt (Spec, Frage 14: keine Endungslücke mehr im Suchkorpus).
- **Konfigurationsabschnitt `[grep]` (Spec, Frage 18):** entschieden. `max_filesize` steht in `[grep]`; A4c schreibt den Schlüssel, A10 ergänzt die Doku und präzisiert den Abschnitt „`[graph]` — there is none“ mit der Begründung (Hooks und MCP-Aufrufe ohne Flag).
- **`--hidden` / `--no-ignore` (Spec, Frage 19):** entschieden: nicht in v1; Punkt-Verzeichnisse wie `.github/` bleiben vorerst nicht durchsuchbar.
- **UTF-16 und Symlinks (Spec, Fragen 16 und 20):** entschieden: UTF-16 binär überspringen, Symlinks im Suchkorpus überspringen, keine Namensmuster-Ausschlüsse.
- **Git-Standardquelle für globale Excludes und `core.ignorecase`-Mechanik (Spec, Frage 21):** in A4b per Test gegen `git` belegen, vor der Übernahme nicht festlegen.
- **Benchmark-Ziele:** Richtwerte aus der Spec, festgelegt erst nach der Baseline in A0. Das Gate (C3) nutzt das Abbruchkriterium, nicht diese Richtwerte.
- **Konflikt Rollout-Reihenfolge (markiert):** Umgesetzt ist die Entscheidung des Menschen (`dsh-loomux` registrieren und prüfen, dann `dsh-tgrep` abschalten). Die frühere Fassung der Spec verlangte das Abschalten vor Phase 2. Vorbedingung für C4: Spec, Frage 12 (Host-Semantik) geklärt.
- **Zweiter Korpus für das Gate:** `iam_backend` (`C:\Users\micro\Documents\#GIT\iam_backend`, Python), beschlossen (Spec, Frage 15). In dieser Sitzung nicht gelesen; C2 bestätigt den Zugriff vor der Messung.
- **Abdeckungslücke ohne Rückfall:** beschlossen (Spec, Frage 14). Der Suchkorpus deckt alle Textdateien ab; was fehlt (ignoriert, binär, zu groß), zählt das Ergebnis. Der Weg des Agenten für diese Dateien ist nicht Teil des Plans.
- **Paketordner `dsh-tgrep`:** offen, ob er nach dem Abschalten in `node_modules` bleibt.

## Global Ordering

Teil A läuft auf dem Zweig `docs/tgrep-loomux-fusion` oder einem Folgezweig,
den der Mensch anlegt. Teil B beginnt, wenn B0 erfüllt ist. Die Teile
überschneiden sich nur an der CLI-Schnittstelle (A8 → B1).

Abhängigkeiten:
- A0 → alles (Baseline vor Änderung).
- A1 → A2 → A3 → A4 (Paket).
- A4a → A4b → A4c → A4d (Walk, Ignore, Größe/Binär/Konfiguration, Digest und Record). A4a braucht keinen Trigram-Code.
- A3, A4d → A5 (Build; schreibt Index und Record aus denselben Bytes) → A6 (Grep über den Suchkorpus) → A9 (Gültigkeit).
- A7 (Privacy) → A8 (CLI) → A10 (Doku, Gate).
- B0 (Lesen von `index.js`) → B1–B4, B6–B8. B8 Schritt 2 (Veröffentlichung) nur nach Freigabe des Menschen.
- B1 und B2 können gegen ein festes JSON-Fixture laufen, bevor A8 steht.
- A4d, A5, A6, A8 → C3 (das Gate misst den Index-Pfad; C1 und C2 vorher).
- B8 Schritt 3 (`dsh-loomux` registriert), B7 Schritt 2 und Spec-Frage 12 → C4.
- C3 bestanden → C4 (Entfernung von tgrep).

---

## Teil A: loomux (Go)

### Task A0: Baseline messen

**Files:**
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md` (neuer Eintrag, chronologisch)

**Interfaces:** keine Codeänderung.

- [ ] **Step 1: Vorbedingung prüfen.** `go build -o bin/loomux.exe ./cmd/loomux`. Ergebnis: Binary vorhanden.
- [ ] **Step 2: Messmuster festlegen.** Vier Muster auf dem loomux-Repo: ein Literal (z. B. `func Search`), eine Alternation (`Search|Grep`), ein Muster ohne Literal (`[A-Z]{3}\w+`), ein Muster mit `-i`.
- [ ] **Step 3: Kalt und warm messen.** Kalt: nach `loomux graph build --no-reuse`. Warm: 5 Läufe, Median. Befehl: `loomux graph grep <muster> --no-refresh --json > NUL`, Zeit mit `Measure-Command` oder `time`.
- [ ] **Step 4: Eintrag schreiben.** Datum, Uhrzeit, Basis (`1816634d`), gemessene Werte, Befehle. Ausgabe der Rohdaten als Anhang oder Verweis.
- [ ] **Step 5: Kein tgrep-Vergleich in A0.** A0 misst nur den Scan. Der Vergleich mit tgrep ist das Gate in Task C3.
- [ ] **Step 6: Ausgangskorpus und Probe-Aufwand messen (vierter Durchgang).** Der Scan von A0 liest nur Graph-Knoten; das ist der alte Korpus. Zusätzlich: (a) Zeit einer Abfrage mit Refresh bei unverändertem Baum, 5 Läufe, Median (Probe-Aufwand heute); (b) Zahl der Graphdateien und Bytes; (c) Zahl und Bytes aller Dateien im loomux-Repo, die `git ls-files --cached --others --exclude-standard` liefert, zur Einordnung des neuen Korpus. Zählmethode im Eintrag beschreiben. `iam_backend` wird in A0 nicht gemessen (Spec, Frage 15).

**Akzeptanz:** Beide `benchmarks.md` haben einen Eintrag mit Baseline-Werten. Die Zielwerte der Spec werden erst danach bestätigt oder angepasst.

**Commit-Vorschlag (Mensch):** `docs: record the graph grep baseline`

---

### Task A1: Literale aus dem Muster

**Files:**
- Create: `internal/code/trigram/literals.go`
- Create: `internal/code/trigram/literals_test.go`

**Interfaces:**
- Produces: `func Literals(pattern string, fixed, ignoreCase bool) (sets [][]string, ok bool, err error)`. `sets` ist eine Disjunktion von Konjunktionen von Literalen (Bytes), `ok == false` heißt „keine Bedingung“.

- [ ] **Step 1: Tests rot.** Tabelle:
  - `abcdef` → ein Literal `abcdef`, ok.
  - `ab|cd` → keine Bedingung (Alternation auf Wurzel, Länge < 3 je Zweig). Oder Disjunktion, wenn beide Zweige ≥ 3 Byte haben (`abc|def` → zwei Mengen).
  - `[a-z]+` → keine Bedingung.
  - `a.*bcd` → Literal `bcd` (Konjunktion mit Wildcard dazwischen).
  - `--fixed` mit `a.b` → Literal `a.b`, wörtlich.
  - `-i` mit `ABC` → Literal `abc` (ASCII gefaltet).
  - `-i` mit `Ä` → `ok == false`, kein Fehler, Begründung im Ergebnis (`reason: "non-ascii fold"`).
  - Ungültiges Regex `(` → Fehler.
  - Lookaround `(?=x)` → Fehler `unsupported`, passend zu `grep.unsupported()`.
  - Lauf → FAIL.
- [ ] **Step 2: Implementieren.** `regexp/syntax.Parse` mit `Perl`-Flags, dann Traversierung der `Regexp`-Knoten: `OpLiteral` mit ≥ 3 Byte liefert ein Literal; `OpConcat` verbindet; `OpAlternate` bildet Disjunktion; alles andere liefert keine Bedingung.
- [ ] **Step 3: Grün.** `go test ./internal/code/trigram/ -run Literal -v`.

**Akzeptanz:** Jeder Fall der Tabelle liefert das erwartete Ergebnis. Nicht-ASCII mit `-i` führt nie zu einer Bedingung.

**Commit-Vorschlag:** `feat(graph): derive literal conditions from a grep pattern`

---

### Task A2: Trigramme aus Bytes

**Files:**
- Create: `internal/code/trigram/trigram.go`
- Create: `internal/code/trigram/trigram_test.go`

**Interfaces:**
- Produces: `func Grams(data []byte) []uint32` (sortiert, dedupliziert); `func Pack(s string) (uint32, bool)`. ASCII-Faltung: `A`–`Z` → `a`–`z`.

- [ ] **Step 1: Tests rot.** `Grams("abcd")` → `{abc, bcd}`; `Grams("ABC")` == `Grams("abc")`; `Grams("ab")` → leer; Zeilenende zwischen Zeichen zählt; `Pack("abc")` ok, `Pack("äb")` → `false`.
- [ ] **Step 2: Implementieren.** Gleitendes Fenster, Faltung, Sortierung, Dedup.
- [ ] **Step 3: Grün.**

**Akzeptanz:** Trigramm-Menge ist sortiert, ohne Duplikate, unabhängig von der Reihenfolge der Eingabedateien.

**Commit-Vorschlag:** `feat(graph): extract case-folded trigrams from file bytes`

---

### Task A3: Indexformat schreiben und lesen

**Files:**
- Create: `internal/code/trigram/format.go`
- Create: `internal/code/trigram/format_test.go`
- Create: `internal/code/trigram/testdata/corpus/` (kleiner Korpus, 4–6 Dateien, mit `.go`, `.py` und einer CRLF-Datei)
- Create: `internal/code/trigram/testdata/golden.idx` (erzeugt beim ersten grünen Lauf, Byte-Vergleich danach)

**Interfaces:**
- Produces:
  - `type File struct { Rel string; Data []byte }`
  - `func Build(files []File, extractor string, settings []byte, digest [32]byte) ([]byte, error)`. `settings` sind die kanonischen Einstellungen aus A4d; sie stehen im Kopf mit Längenfeld (Spec, Bytes).
  - `func Write(root string, digest [32]byte, extractor string, settings []byte, files []File) error` (atomar, PID im Temp-Namen wie `store.Write`)
  - `func Read(root string) (*Index, error)`
  - `type Index struct { … }` mit `Files() []string`, `Postings(gram uint32) []uint32`
  - Fehler: `ErrNoIndex`, `ErrStale`, `ErrCorrupt`

- [ ] **Step 1: Tests rot.**
  - `TestRoundTrip`: Build → Read → Postings stimmen mit Eingabe.
  - `TestDeterministic`: zwei Builds derselben Eingabe ergeben identische Bytes; Eingabe in anderer Reihenfolge ebenso.
  - `TestGolden`: Bytes stimmen mit `testdata/golden.idx` überein. Die Golden-Datei entsteht einmalig in Step 3, vor dem grünen Lauf dieses Tests.
  - `TestBadMagic`, `TestVersionMismatch` (→ `ErrStale`), `TestTruncated` (→ `ErrCorrupt`), `TestOffsetOutOfRange`, `TestLengthOutOfRange`, `TestPathTooLong` (> 64 Byte Extraktor).
  - `TestWriteAtomic`: kein halbfertiges `trigram.idx` nach einem Fehler.
  - `TestFilesSorted`: Dateien nach Byte-Reihenfolge des Pfades.
  - `TestHeaderCarriesSettings`: Einstellungsbytes stehen im Kopf mit Längenfeld; Round-Trip liefert sie zurück; eine Änderung der Einstellungen ändert die Bytes.
  - Lauf → FAIL.
- [ ] **Step 2: Implementieren.** Kopf wie in der Spec-Tabelle, Dateitabelle, Trigrammtabelle (16 Byte je Eintrag), Postings. Lesen mit Prüfung jeder Länge und jedes Offsets vor dem Zugriff.
- [ ] **Step 3: Golden erzeugen.** Die Golden-Datei entsteht einmal über ein temporäres Hilfsprogramm außerhalb des Produktivcodes (kein Update-Flag im Paket), wird per Hand gegen die Erwartung geprüft und dann eingecheckt.
- [ ] **Step 4: Grün.** `go test ./internal/code/trigram/ -v`.

**Akzeptanz:** Zwei Builds sind byteidentisch. Jede verletzte Struktur liefert `ErrCorrupt` oder `ErrStale`, nie einen Panic.

**Commit-Vorschlag:** `feat(graph): write and read the trigram index`

---

### Task A4: Kandidatenauswahl

**Files:**
- Create: `internal/code/trigram/candidates.go`
- Create: `internal/code/trigram/candidates_test.go`
- Create: `internal/code/trigram/fuzz_test.go`

**Interfaces:**
- Produces: `func (ix *Index) Candidates(literals [][]string, ...) (files []string, used bool)`. `used == false` heißt: keine Bedingung, alle Dateien.

- [ ] **Step 1: Tests rot.**
  - `TestCandidatesIntersect`: Literal mit Trigrammen `abc` und `bcd` → Schnitt der Postings.
  - `TestDisjunctionUnion`: Alternation → Vereinigung.
  - `TestNoConditionUsesAll`: keine Bedingung → alle Dateien, `used == false`.
  - `TestMissingGramEmpty`: Trigramm in keiner Datei → leere Liste, `used == true` (korrekt: kein Treffer möglich).
  - `FuzzCandidatesSuperset` (Fuzz): Zufallstexte und aus Zeichen zusammengesetzte Muster; für jede Datei, auf der `regexp.Match` trifft, muss die Datei in den Kandidaten sein. Seed-Korpus aus dem Testkorpus.
  - Lauf → FAIL.
- [ ] **Step 2: Implementieren.** Postings als sortierte Listen, Schnitt per Zwei-Zeiger, Vereinigung per Mischen.
- [ ] **Step 3: Fuzzen.** `go test ./internal/code/trigram/ -run=^$ -fuzz=FuzzCandidatesSuperset -fuzztime=60s`. Ein Fehlschlag ist ein Fund: Literalableitung korrigieren, nie den Test abschwächen.
- [ ] **Step 4: Grün.**

**Akzeptanz:** Der Fuzz-Test findet keine Trefferdatei außerhalb der Kandidaten.

**Commit-Vorschlag:** `feat(graph): select grep candidates from trigram postings`

---

### Task A4a: Ein Walk für Graph und Suche

**Files:**
- Create: `internal/code/corpus/walk.go`, `internal/code/corpus/walk_test.go`
- Create: `internal/code/corpus/legacy_test.go` (Kopie der alten `sourceset.Stat`-Logik aus Stand 1816634d, nur als Referenz im Test)
- Modify: `internal/code/sourceset/sourceset.go` (`Stat` wird Filter über `corpus.Walk`), `internal/code/sourceset/sourceset_test.go`

**Interfaces:**
- Produces: `type Entry struct { Rel, Abs string; Size, MTime int64; Ignored, Tracked, Link bool }`; `type Listing struct { Entries []Entry; Repo bool }`; `func Walk(root string, m Matcher) (Listing, error)`. (Der Typ heißt `Listing`, weil `Walk` die Funktion ist.) `Matcher` ist ein Interface mit `Ignored(rel string, isDir bool) bool` und `HasTrackedUnder(dir string) bool`. `m == nil` heißt: keine Ignore-Auswertung (außerhalb eines Repos).
- Consumes: nichts Neues; die Ignore-Implementierung kommt in A4b.

- [ ] **Step 1: Alte Logik als Referenz.** Die bisherige `sourceset.Stat` wird vor der Änderung als `legacyGraphList` in `legacy_test.go` kopiert. Der Differentialtest gegen den neuen Graphfilter läuft auf einem Testbaum mit Grenzfällen und auf dem loomux-Repo. Ein festes Golden gilt nicht, weil das Repo wächst.
- [ ] **Step 2: Tests rot.**
  - `TestGraphListUnchanged`: neuer Graphfilter gleich `legacyGraphList` auf dem Testbaum (Grenzfälle: Punkt-Datei `.x.go` bleibt drin; `testdata/` und `node_modules/` draußen; Datei mit 1 000 000 Byte drin, 1 000 001 Byte draußen; Symlink wie bisher; verschachteltes `dist/`) und auf dem loomux-Repo.
  - `TestPrunesDotDirs`: Datei unter `.loomux/state/` und `.github/` erscheint in keinem Eintrag.
  - `TestPrunesIgnoredSkipDirWithoutTracked`: ignoriertes `node_modules/` ohne getrackte Datei wird nicht betreten (Zähler der besuchten Verzeichnisse im Test).
  - `TestDescendsIgnoredSkipDirWithTracked`: ignoriertes `dist/` mit getrackter Datei wird betreten; die Datei hat `Tracked` und nicht `Ignored`.
  - `TestDescendsIgnoredNonSkipDir`: ignoriertes `docs/gen/` wird betreten; Dateien haben `Ignored`, Graphfilter nimmt sie wie bisher.
  - `TestOutsideRepoNoIgnore`: `m == nil`; `node_modules/x.md` ist Eintrag mit `Ignored == false`.
  - `TestSymlinkFlag`: Symlink-Eintrag hat `Link == true`; Graphfilter verhält sich wie heute (Spec, Frage 16).
  - `TestUnreadableDirSkipped`: Lesefehler eines Verzeichnisses bricht den Walk nicht ab (wie `sourceset.go:121–123`).
  - Lauf → FAIL.
- [ ] **Step 3: Implementieren.** `filepath.WalkDir`; Prune-Regel: `dot(name) || (skipDirs[name] && m != nil && m.Ignored(rel, true) && !m.HasTrackedUnder(rel))`. Einträge nach Byte-Reihenfolge des Pfades sortieren. `sourceset.Stat` filtert `Entries` nach dem bestehenden Graphprädikat (Endung, Größe `maxFileBytes`, `SkipDir` nur auf Verzeichnissegmente, nie auf den Dateinamen).
- [ ] **Step 4: Grün.** `go test ./internal/code/corpus/ ./internal/code/sourceset/ -v`.

**Akzeptanz:** Der Differentialtest gegen `legacyGraphList` ist gleich, auf dem Testbaum und auf dem loomux-Repo. Kein Eintrag unter `.git` oder einem Punkt-Verzeichnis.

**Commit-Vorschlag:** `refactor(graph): derive the graph file list from one corpus walk`

---

### Task A4b: Ignore-Auswertung nach Git-Regeln

**Files:**
- Create: `internal/code/corpus/ignore.go`, `internal/code/corpus/ignore_test.go`
- Create: `internal/code/corpus/testdata/ignore/` (Tabellenfälle mit Erwartung je Fall)

**Interfaces:**
- Produces: `func Load(root string) (Matcher, []Source, error)`. Gibt `nil` zurück, wenn `git rev-parse --show-toplevel` fehlschlägt (kein Repo). `type Source struct { Rel string; Size, MTime int64; Hash [32]byte }` für jede gelesene Ignore-Quelle (`.gitignore`, `.git/info/exclude`, `core.excludesFile`). Git wird nur beim Build aufgerufen, nicht pro Abfrage (A4d). Dieselbe Funktion liefert den Tracking-Stand: `git ls-files -z` beim Build und den Stand von `.git/index` (`git rev-parse --git-path index`).
- Consumes: `git` über `os/exec` (Muster in `internal/brain/vcs/vcs.go:310`).

- [ ] **Step 1: Tests rot.** Tabelle, je Fall mit Erwartung. Ein Fall je Zeile der Spec-Liste: `*.log`; `!keep.log` nach `*.log`; `/top.txt` (verankert); `build/` (nur Verzeichnis); `**/gen/*.go`; `sub/.gitignore` überschreibt die Wurzel; Negation kann eine Datei in einem ausgeschlossenen Verzeichnis nicht wieder einschließen; `\#` als Escape; Leerzeichen am Ende; `.git/info/exclude`; `core.excludesFile` gesetzt; `core.ignorecase=true` (Erwartung erst nach Test gegen git gesetzt, siehe Schritt 2); getrackte Datei unter einem Ignore-Muster (`git add -f`) ist nicht ignoriert; außerhalb eines Repos `Load` liefert `nil`.
- [ ] **Step 2: Belegen, was offen ist.** Im Test werden die Fälle `core.excludesFile` ungesetzt und `core.ignorecase=true` gegen `git check-ignore -v` in einem temporären Repo geprüft. Das Ergebnis steht im Eintrag und in der Spec (Frage 21). Vorher wird keine Mechanik übernommen.
- [ ] **Step 3: Implementieren.** Zeilenweiser Parser, Musterabgleich in reinem Go (keine neue Abhängigkeit). Späteres Muster gewinnt. Verzeichnis-Ausschluss wirkt auf alle Kinder.
- [ ] **Step 4: Grün.** `go test ./internal/code/corpus/ -run Ignore -v`.
- [ ] **Step 5: Orakel-Test im Gate (A10).** Auf dem loomux-Repo: für jeden besuchten Pfad `Ignored` gleich dem Ergebnis von `git check-ignore --stdin`. Auf der Dateiliste: Suchkorpus gleich `git ls-files --cached --others --exclude-standard`, gefiltert mit denselben Binär-, Größen- und Punkt-Regeln. Abweichungen werden im Eintrag benannt.

**Akzeptanz:** Alle Tabellenfälle stimmen mit `git check-ignore` überein. Außerhalb eines Repos keine Auswertung.

**Commit-Vorschlag:** `feat(graph): evaluate git ignore rules for the search corpus`

---

### Task A4c: Größe, Binärregel und Konfiguration

**Files:**
- Create: `internal/code/corpus/options.go`, `internal/code/corpus/options_test.go`, `internal/code/corpus/binary.go`, `internal/code/corpus/binary_test.go`
- Modify (nur nach Entscheidung Spec-Frage 18): `internal/config/schema/schema.go` (Schlüssel `grep.max_filesize`, `Kind: Int`, `Module: Graph`, Vorgabe `67108864`), `internal/config/schema/schema_test.go`
- Modify: `internal/cli/graph.go` (Flags werden in A8 verdrahtet; hier nur die Auflösung)

**Interfaces:**
- Produces: `func ResolveMaxFilesize(flag *int64, configured *int64) (int64, error)` mit der Rangfolge Flag, Konfiguration, Vorgabe 67108864. `0` ist unbegrenzt. Negative Werte sind ein Fehler. `func IsBinary(data []byte) bool` (enthält ein NUL-Byte).

- [ ] **Step 1: Tests rot.**
  - `TestIsBinary`: NUL am Anfang, in der Mitte, am letzten Byte → binär; leere Datei → Text; UTF-8 ohne NUL → Text; UTF-16 mit BOM → binär (Grenzfall, Spec Frage 20).
  - `TestSizeLimitBoundary`: Größe gleich Grenze → im Korpus; Grenze + 1 → `skipped_size`.
  - `TestResolveMaxFilesize`: Flag gewinnt über Konfiguration; Konfiguration gewinnt über Vorgabe; `0` unbegrenzt; `--no-max-filesize` entspricht `0`; negativ → Fehler.
  - `TestSchemaKeyMaxFilesize` (nur wenn Frage 18 entschieden): Schlüssel vorhanden, Typ `Int`, Modul `Graph`, Vorgabe `67108864`.
  - Lauf → FAIL.
- [ ] **Step 2: Implementieren.** `bytes.IndexByte(data, 0) >= 0`. Die Grenzprüfung im Filter nach dem Walk, nicht im Walk (Spec, Go-Pakete).
- [ ] **Step 3: Grün.**

**Akzeptanz:** Die Grenze ist inklusiv. Ohne Entscheidung zu Frage 18 wird kein Konfigurationsschlüssel geschrieben; die Vorgabe gilt als Konstante.

**Commit-Vorschlag:** `feat(graph): cap search file size and detect binary files`

---

### Task A4d: Digest und Record für den Suchkorpus

**Files:**
- Create: `internal/code/corpus/record.go`, `internal/code/corpus/record_test.go`
- Create: `internal/code/corpus/probe.go`, `internal/code/corpus/probe_test.go`

**Interfaces:**
- Produces:
  - `type Record struct { Version int; Settings []byte; Sources []Source; Stamp Stamp; ConfigStamp Stamp; Tracked []string; GitPaths GitPaths; Files map[string]Print }` mit `Print{Size, MTime int64; Hash string; Binary bool}`. `Stamp` ist Größe und Änderungszeit von `.git/index`, `ConfigStamp` die von `.git/config`. `Tracked` ist die sortierte Liste aus `git ls-files -z`. `GitPaths` hält die von git gemeldeten Pfade (Repo-Wurzel, Index, Config, `core.excludesFile`), damit die Probe ohne `git` auskommt.
  - `func Digest(rec Record) [32]byte`: SHA-256 über kanonische Einstellungen, Hashes der Ignore-Quellen (Pfad-sortiert) und die Paare `(rel, hash)` des Suchkorpus.
  - `func Write(root string, rec Record) error` nach `store.CachePath(root, "corpus.json")`, atomar wie `store.Write`.
  - `func Probe(root string, s Settings) (*Drift, error)`: Walk, Stat-Vergleich, Hash nur bei Abweichung (Regel `trusted` aus `internal/code/freshness/fingerprint.go:178–180`), Ignore-Quellen und `.git/index` per Stat, Konfiguration per Stat. `nil, nil` heißt unbekannt. Der Matcher wird aus den gespeicherten Quellen und `Tracked` neu gebaut; `git` läuft nur, wenn `Stamp` oder `ConfigStamp` abweicht, und dann ist die Probe ohnehin Drift.
- Consumes: `corpus.Walk` (A4a), `Load` (A4b), `ResolveMaxFilesize` (A4c).

- [ ] **Step 1: Tests rot.**
  - `TestDigestDeterministic`: gleiche Eingabe, gleiche Bytes; Eingabe in anderer Reihenfolge ebenso.
  - `TestDigestChangesWithSettings`: andere Größe → anderer Digest.
  - `TestDigestChangesWithIgnoreSource`: geänderte `.gitignore` → anderer Digest.
  - `TestDigestChangesWithTrackingStamp`: anderer `.git/index`-Stand ohne Dateiänderung → anderer Digest (Simulation von `git add -f`).
  - `TestProbeCleanWithoutReads`: unveränderter Baum, Zähler der `ReadFile`-Aufrufe ist null (außer Ignore-Quellen bei Stat-Abweichung).
  - `TestProbeDriftOnContent`: Datei mit geänderter Größe und Hash → `Changed`.
  - `TestProbeEmptyHashNeverTrusted`: leerer Hash → Verdacht, wird gehasht.
  - `TestProbeUnknownWithoutRecord`: kein Record → `nil, nil`.
  - `TestProbeConfigChangeIsDrift`: andere Größe in der Konfiguration → Drift.
  - `TestProbeNoGitWithoutStampChange`: Zähler der `git`-Aufrufe (Test-Seam im Aufrufer) ist null, wenn `Stamp` und `ConfigStamp` gleich sind.
  - `TestProbeConfigStampChangeIsDrift`: geänderte `.git/config` (etwa `core.ignorecase`) → Drift und Neuaufbau mit `git`.
  - Lauf → FAIL.
- [ ] **Step 2: Implementieren.** Kanonische Einstellungsbytes: fester Text `max_filesize=<n>;binary=nul-v1;ignore=git-v1;ignorecase=<bool>`.
- [ ] **Step 3: Grün.**

**Akzeptanz:** Jede Eingabe, die Indexbytes ändern kann, geht in den Digest ein. Der unveränderte Probe-Pfad liest keine Suchdatei.

**Commit-Vorschlag:** `feat(graph): record the search corpus and its settings as a digest`

---

### Task A5: Index im Build

**Files:**
- Modify: `internal/code/query/build.go` (nach dem Schreiben von `wiring.json`, dem Freshness-Record und dem Suchkorpus-Record aus A4d)
- Modify: `internal/code/query/build_test.go`
- Modify: `internal/cli/graph.go` (`graphBuild`: Hinweis auf `stderr` bei Fehler, kein Abbruch)

**Interfaces:**
- Consumes: `trigram.Write` (mit Einstellungsbytes), `corpus.Walk` und `corpus.Write` (A4a, A4d), `freshness.Write`'s `hashes`-Karte (Graphkorpus), `sourceset.Stat` (Graphfilter).

- [ ] **Step 1: Tests rot.**
  - `TestBuildWritesIndex`: nach `Build` existiert `cache/trigram.idx`, `Read` liefert die Dateiliste der Quelle.
  - `TestBuildIndexDigestMatchesRecord`: Digest im Index gleich dem Digest im Suchkorpus-Record (A4d); der Graph-Record bleibt unberührt.
  - `TestBuildSkipsUnreadableSearchFile`: unlesbare Suchdatei bricht den Build nicht ab; Zähler `unreadable` steigt; Record führt sie mit leerem Hash.
  - `TestBuildGraphFileUnreadableAborts`: unlesbare Graphdatei bricht den Build ab wie bisher (`build.go:133–136`).
  - `TestBuildRecordsSettings`: Einstellungen aus Flag oder Konfiguration stehen im Index und im Record.
  - `TestBuildIndexFailureIsNote`: nicht schreibbares `cache/` → Build endet mit 0 (wie Freshness), Notiz auf `stderr`.
  - `TestBuildIndexSameBytesAsSource`: Der Build liest jede Datei des Suchkorpus und des Graphkorpus nur einmal (Zähler im Test-Reader).
  - Lauf → FAIL.
- [ ] **Step 2: Implementieren.** Die Bytes, die `Build` hasht, werden auch dem Index übergeben. Kein zweites Lesen.
- [ ] **Step 3: Grün.** Dazu `go test ./internal/code/query/ -run Build`.

**Akzeptanz:** Ein Build schreibt Graph, Freshness-Record, Suchkorpus-Record und Index aus denselben Bytes. Ein Index- oder Record-Fehler bricht den Build nicht ab (wie der Freshness-Record). Ein Lesefehler einer Graphdatei bricht ihn weiter ab.

**Commit-Vorschlag:** `feat(graph): build the trigram index with the graph`

---

### Task A6: Index in `graph grep`

**Files:**
- Modify: `internal/code/query/grep.go` (`Grep`: Kandidaten holen, Reader auf Kandidaten einschränken, `Result`-Felder setzen)
- Modify: `internal/code/grep/grep.go` (`Result` um `Index`, `FilesInScope`, `Candidates`; `Search` bekommt eine optionale Kandidatenmenge; ohne Menge unverändert)
- Modify: `internal/code/query/grep_test.go`, `internal/code/grep/grep_test.go`
- Create: `internal/code/query/grep_equivalence_test.go`

**Interfaces:**
- `grep.Options` bekommt `Candidates map[string]bool` (nil = alle).
- `Search` bekommt die Dateiliste des Suchkorpus (`corpus.Walk` gefiltert) statt der Pfade der Graph-Knoten. Der Scan und der Index-Pfad nutzen dieselbe Liste, sonst ist A6 nicht vergleichbar (Anti-Ziel A1).
- Die Gruppe ohne Symbol (`Symbol == nil`, `omitempty`) ist der Dateitreffer. Die Textform kennt ihn schon (`GrepReport`, „file level“).
- `Result` bekommt `Index string` (`used`, `scan`, `stale`, `none`), `FilesInScope int`, `Candidates int`, `SkippedBinary int`, `SkippedSize int`, `Ignored int` (JSON-Namen `index`, `files_in_scope`, `candidates`, `skipped_binary`, `skipped_size`, `ignored`).
- Der Präfix-Check `prefix not indexed` (`internal/code/grep/grep.go:98–110`) prüft gegen den Suchkorpus, nicht gegen die Graph-Knoten.

- [ ] **Step 1: Tests rot.**
  - `TestEquivalenceWithScan`: auf dem Suchkorpus des Testkorpus und des loomux-Repos, für eine Liste von Mustern (literal, `-i`, Alternation, ohne Literal, `--fixed`, `--in`): `groups`, `total_hits`, `truncated_hits` aus Index-Pfad und Scan-Pfad sind gleich.
  - `TestScanCoversNonGraphFiles`: eine `.md`-Datei mit Treffer erscheint im Scan und im Index-Pfad, ohne Symbol.
  - `TestIgnoredFileNotSearched`: eine ignorierte Datei mit Treffer erscheint nicht; `ignored` zählt sie.
  - `TestBinaryAndSizeCounted`: binäre und zu große Dateien erscheinen nicht, ihre Zähler steigen.
  - `TestIndexUsedWhenLiteral`, `TestIndexScanWhenNoLiteral`.
  - `TestStaleIndexScans`: Digest passt nicht → `index: stale`, Ergebnis gleich Scan.
  - `TestNoIndexScans`: `index: none`.
  - `TestMaxHitsSameOnBothPaths`.
  - Lauf → FAIL.
- [ ] **Step 2: Implementieren.** Gültigkeitsprüfung: Extraktorversion, Formatversion, Digest gegen Freshness-Record. Nur bei Gültigkeit Kandidaten.
- [ ] **Step 3: Grün.** Gesamter Paketlauf: `go test ./internal/code/...`.

**Akzeptanz:** Auf dem Testkorpus sind Index- und Scan-Ergebnis in allen Feldern außer `files_searched`, `index`, `files_in_scope`, `candidates` identisch, einschließlich der neuen Zähler.

**Commit-Vorschlag:** `feat(graph): answer grep from the trigram index when it is current`

---

### Task A7: Privacy-Kanal für `graph grep`

**Files:**
- Modify: `internal/code/query/grep.go` (`GrepOptions.Channel`, `GrepOptions.Glob`; `Keep` aus Channel)
- Create: `internal/code/query/grep_privacy.go` (Erzeugung von `Keep` aus Manifest; Fail-closed)
- Create: `internal/code/query/grep_privacy_test.go`

**Interfaces:**
- Consumes: `privacy.IsReadable(manifest, path)`, `VisibleArea.Conceals`. Das Manifest wird wie in `internal/serve/graph` geladen (zu prüfen: Funktion, die dort die Area liefert; die Spec nennt nur `readable(area)`).

- [ ] **Step 1: Vorarbeit lesen.** Die Funktion in `internal/serve` finden, die die Area für einen Pfad auflöst. Ergebnis im Commit-Text des Schritts festhalten (nur im Arbeitsprotokoll, nicht im Commit).
- [ ] **Step 2: Tests rot.**
  - `TestCloudHidesNeverPath`: Pfad unter `[privacy] never` fehlt in den Treffern auf `cloud`.
  - `TestLocalUnchanged`: `local` liefert die heutige Antwort.
  - `TestCloudUnreadableManifestExitsOne`: nicht lesbares Manifest → Fehler, kein Ergebnis.
  - `TestCloudPrefixUnderNeverEmpty`: `--in` auf `never`-Pfad → leeres Ergebnis mit Notiz, kein `prefix not indexed`.
  - `TestGlobFilters`: `--glob "**/*.go"` schränkt Kandidaten ein.
  - Lauf → FAIL.
- [ ] **Step 3: Implementieren.** `Keep` kommt aus dem Prädikat; `Glob` wirkt auf Pfad vor dem Lesen.
- [ ] **Step 4: Grün.**

**Akzeptanz:** `cloud` gibt keinen Treffer aus einer `never`-Datei zurück, auch nicht über Index oder Glob. Das gilt für alle Dateien des Suchkorpus, nicht nur für Graphdateien (Test `TestCloudHidesNeverMarkdown` ergänzen). Ohne lesbares Manifest gibt es keine Antwort.

**Commit-Vorschlag:** `feat(graph): filter grep by privacy on the cloud channel`

---

### Task A8: CLI-Flags

**Files:**
- Modify: `internal/cli/graph.go` (`graphGrep`: `--channel`, `--glob`; Usage)
- Modify: `internal/cli/graph_test.go`

**Interfaces:**
- Flags: `--channel local|cloud` (Vorgabe `local`), `--glob <pattern>`. Ungültiger Channel → Exit 2.
- Flags neu (vierter Durchgang): `--max-filesize <bytes>` und `--no-max-filesize` bei `graph grep` und `graph build`. Negativer Wert → Exit 2. `--max-filesize` und `--no-max-filesize` zusammen → Exit 2.
- Die Auflösung der Rangfolge steht in A4c (`ResolveMaxFilesize`); hier nur die Verdrahtung.

- [ ] **Step 1: Tests rot.**
  - `TestGrepChannelFlag`: `local` ist Vorgabe; `cloud` setzt `Keep`.
  - `TestGrepInvalidChannelUsage`: Exit 2.
  - `TestGrepGlobFlag`.
  - `TestGrepJSONHasIndexField`: `index`, `files_in_scope`, `candidates` im Ausgabe-JSON.
  - `TestGrepInvalidRegexExitOne`: unverändert Exit 1, Meldung `invalid regex` (Wortlaut bleibt, Frage 9).
  - Lauf → FAIL.
- [ ] **Step 2: Implementieren.**
- [ ] **Step 3: Grün.**

**Akzeptanz:** Alle bisherigen Flags und Exit-Codes funktionieren unverändert. Die neuen Felder sind additiv.

**Commit-Vorschlag:** `feat(cli): add channel and glob to graph grep`

---

### Task A9: Gültigkeit beim Rebuild

**Files:**
- Modify: `internal/code/query/grep.go` (Notiz bei `stale`)
- Modify: `internal/code/query/grep_test.go`

- [ ] **Step 1: Tests rot.**
  - `TestRebuildRefreshesIndex`: Datei ändern, `graph grep` (mit Refresh) → Index neu, `index: used`.
  - `TestLockedRebuildAnswersFromDisk`: Sperre gehalten → Notiz, Antwort aus Platte, `index` gemäß Gültigkeit.
  - `TestFlagSettingsNoRebuild`: `graph grep --max-filesize X` gegen einen Index mit anderem Wert → kein Rebuild, Scan mit X, Notiz „settings differ from index“, `index: stale`.
  - `TestConfigChangeRebuilds`: Konfigurationsänderung → Drift → Rebuild mit dem neuen Wert (nur nach Frage 18).
  - `TestNoFlagRebuildsToDefault`: Index mit Flag X, Abfrage ohne Flag mit Refresh → Drift, Rebuild auf die Vorgabe. Mit `--no-refresh` bleibt es beim Scan mit Notiz.
  - `TestIgnoreEditMakesStale`: geänderte `.gitignore` → Drift → Rebuild; mit `--no-refresh` → `stale`.
  - `TestGitAddForceMakesStale`: `git add -f` einer ignorierten Datei ohne Dateiänderung → Drift über `.git/index`-Stand → Rebuild, die Datei ist danach im Suchkorpus (Integrationstest mit temporärem Repo).
  - Lauf → FAIL.
- [ ] **Step 2: Implementieren** (vermutlich nur Notiztext; Logik sitzt in A6).
- [ ] **Step 3: Grün.**

**Akzeptanz:** Der Index ist nach jedem Refresh gültig oder wird als `stale` gemeldet.

**Commit-Vorschlag:** `test(graph): cover index freshness across a rebuild`

---

### Task A10: Doku, Benchmarks, Gate

**Files:**
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md` (§6: `graph grep` mit `--channel`, `--glob`, Felder `index`, `files_in_scope`, `candidates`; §6-Anmerkung zu `never` präzisieren)
- Modify: `docs/en/configuration.md`, `docs/de/configuration.md` (Hinweis zum Cloud-Kanal des Graphen; `grep.max_filesize` und der Abschnitt `[grep]` nur nach Spec-Frage 18; `cache/corpus.json` in §4)
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md` (Messung nach der Änderung, kalt und warm, gegen A0)
- Modify: `README.md`, `README.de.md` (Fähigkeit, Roadmap: Zeile „Fusion tgrep und Code-Graph“ mit Abhängigkeiten und Priorität; Priorität setzt der Mensch)
- Modify: Wiki-Seiten unter `docs/wiki/` nach Ermittlung über `docs/wiki/index.md` (nicht raten)

- [x] **Step 1: Doku schreiben.** Erledigt: `cli-reference` §6, `configuration` §4 und `[grep]`, Benchmarks, READMEs, Wiki-Seiten `topics/code-graph.md` und `topics/datenschutz-und-kanaele.md` samt Protokollzeile; `docs/de/index.md` ist die einzige, schon vorher vorhandene Abweichung der Dateilisten.
- [x] **Step 2: Messung nach der Änderung.** Erledigt, Eintrag 2026-11-04 20:05 in beiden Sprachen: kalter Build 3,494 s (Bericht) / 3,397 s (Uhr), warme Mediane 288,6 / 459,8 / 796,9 / 371,1 ms, Probe 417,8 ms, Korpus 7648 Dateien und 15,66 MB gegen 1136 Graphdateien und 16,96 MB der Basis; beide Korpora im Eintrag.
- [x] **Step 2a: Orakel-Test auf dem Repo.** Erledigt: 7678 besuchte Dateien, 0 Abweichungen gegen `git check-ignore --stdin`; 7648 Korpusdateien gegen 8426 von `git ls-files`, Unterschied vollständig erklärt (778 Punktverzeichnisse, 6 C-quotierte Pfade, dieselben 6 ungequotet); im Eintrag und als Entscheidung 22 im Ledger.
- [x] **Step 3: Zielwerte prüfen.** Erledigt: alle drei Entwurfsziele verfehlt (warm 288,6 gegen < 50 ms, Index 83,9 % gegen < 10 %, Build +173 % gegen +20 %); als Erkenntnis 4 im Eintrag und als Entscheidung 21 im Ledger, das Gate bleibt Teil C.
- [x] **Step 4: Gate.** `check precommit` gelaufen: alle Testpakete grün außer zwei vorbestehenden Umgebungsfällen (`gitenv` behoben, `hostfile` scheitert an der fremden `.claude/settings.json`-Änderung im Baum); `lint/wiki-sources` grün nach `wiki repin`; `go vet ./...` direkt grün, im Gate scheitert `lint/go` an Zugriffsrechten des Go-Build-Cache. Coverage: siehe `check coverage`-Lauf.
- [x] **Step 5: Release-Angaben vorbereiten.** Erledigt: Entwurf in `.cache/pr-body.md` (nicht versioniert), `parse-body -labels release:minor` liefert `bump: minor` und den Changelog-Block; der Mensch öffnet den Pull Request über `release-pr`.

**Akzeptanz:** Gate grün, Coverage 100 %, beide Doku-Sprachen identisch in der Dateiliste, Benchmarks in beiden Sprachen.

**Commit-Vorschlag:** `docs: describe the grep index and its channel`

---

## Teil B: DSH-Plugin

### Task B0: Plugin lesen (Vorbedingung)

**Files:** keine Änderung.

- [x] **Step 1: Leseberechtigung.** Erteilt für `index.js`. `package.json` und `cordis.patch.yml` aus demselben Ordner gelesen. `README.md` nicht gelesen; für die Befunde nicht nötig.
- [x] **Step 2: Abgleich.** Erledigt. Ergebnis: Spec, Abschnitt „Stand der Plugin-Seite (belegt)“. Korrigiert: `matches`-Form, Name der Karte, `max_results`-Klammer, Timeout-Zuständigkeit, Exit-1-Semantik, stilles Abschneiden am stdout-Deckel, fehlender Rückfall.
- [x] **Step 3: Rückfall festlegen.** `@deepseek-ai/dsh-tool-fs-search` liegt nicht in `node_modules\` (dort nur `dsh-tgrep`). `tgrep` ist nicht auf PATH. Das Plugin hat heute keinen Rückfall. Festlegung (Mensch, Fragen 3 und 10): kein Rückfall.

**Akzeptanz:** Erfüllt. Die Spec enthält keine unbelegte Plugin-Angabe mehr; offen bleibt Spec-Frage 12 (Host-Semantik).

---

### Task B1: Schema zu Argumenten

**Files:**
- Create: `dsh-loomux/src/args.js`
- Create: `dsh-loomux/test/args.test.js`

**Interfaces:**
- Produces: `buildArgs(input) → string[]`. Abbildung: `pattern` → Positionsargument; `path` → `--in`; `include` → `--glob` (Task A8; ohne Endungsprüfung, weil der Suchkorpus alle Textdateien umfasst); `case_insensitive` → `-i`; `max_results` → `--max-hits` (Klammer 20–5000, Vorgabe 300 bei 0 oder NaN wie `index.js:858`); immer `--json --channel cloud`.

- [x] **Step 1: Tests rot.** `node --test`.
  - `pattern` fehlt → Fehler.
  - `max_results` 0 → 300; 5 → 20; 6000 → 5000; `abc` → 300.
  - `include` mit Leerzeichen bleibt ein Argument (kein Shell-Split).
  - `include` mit führendem `!` oder Komma auf Top-Ebene → Fehler; `{a,b}` bleibt erlaubt.
  - `include: "*.md"` wird unverändert übergeben; es gibt keine Endungsprüfung mehr (Spec, Abdeckung).
  - `path` nur aus Leerzeichen → Fehler.
  - `--channel cloud` immer enthalten, auch wenn Eingabe es verlangt (kein Schalter für den Agenten).
  - Lauf → FAIL.
- [x] **Step 2: Implementieren.** `spawn` mit Argumentliste, nie mit Shell.
- [x] **Step 3: Grün.**

**Akzeptanz:** Kein Argument kann `--channel local` erreichen. Keine Shell-Interpretation.

**Ergebnis:** `dsh-loomux/src/args.js` mit `buildArgs`, `clampMaxResults` und
`validateInclude`; 11 Tests in `test/args.test.js` grün. Die Argumentliste
beginnt mit `graph grep` (die Subkommandofolge fehlte zuerst und ließ jeden
Aufruf in die Nutzungshilfe laufen), dann Muster, `--json --channel cloud`,
`-i`/`--glob`/`--in`, `--max-hits`.

**Commit-Vorschlag (Mensch):** `feat: map the grep tool schema to loomux graph grep`

---

### Task B2: Ergebnis zu `matches`

**Files:**
- Create: `dsh-loomux/src/result.js`
- Create: `dsh-loomux/test/result.test.js`
- Create: `dsh-loomux/test/fixtures/grep-*.json` (aus echten `loomux graph grep --json`-Ausgaben, nach A8)

**Interfaces:**
- Produces: `toMatches(grepResult) → { matches: [{path, lineNumber, line, symbol?, kind?, signature?, inDegree?, edges?}] }` (Ist-Form: `lineNumber`, kein `text`). `presentationMeta` bleibt `{ shape: 'matches', files, truncated, total }`. `index`, `skipped_binary`, `skipped_size`, `ignored`, `unreadable` gehen in den `render`-Text (kein `engine`, siehe Entscheidungen).

- [x] **Step 1: Tests rot.** Gruppen werden zu flachen Treffern; Symbolfelder nur bei vorhandenem Knoten; Dateitreffer ohne Symbol bleibt gültig; `index` und die Zähler (`ignored`, `skipped_binary`, `skipped_size`, `unreadable`) stehen im `render`-Text, nicht in `presentationMeta`; Replay-Test: `presentResult` liefert aus der gespeicherten Karte dieselben Dateien und Zeilen (Symbolfelder fallen sonst weg, `narrowSearchFiles`, `index.js:676–693`).
- [x] **Step 2: Implementieren** gegen die Ist-Form aus B0 (`presentationMeta`, `index.js:724–736`).
- [x] **Step 3: Grün.**

**Akzeptanz:** Ein Treffer aus einer Gruppe mit Symbol und einer ohne Symbol sind beide korrekt abgebildet.

**Ergebnis:** `dsh-loomux/src/result.js` mit `toMatches`, `describeIndex`,
`renderText`, `presentationMeta` und `capSearchMeta`; 10 Tests in
`test/result.test.js` grün. Fixtures aus echten Ausgaben:
`grep-file-group.json`, `grep-symbol-group.json` (`GrepReport`, Zeile 284,
`in_degree` 6) und `grep-empty.json` (`groups: null`, 7656 Dateien im
Suchbereich).

**Commit-Vorschlag (Mensch):** `feat: shape loomux grep answers as search matches`

---

### Task B3: Routing

**Files:**
- Create: `dsh-loomux/src/route.js`
- Create: `dsh-loomux/test/route.test.js`

**Interfaces:**
- Produces: `route(input, repoRoot) → 'loomux' | 'error'` plus Hinweistext. Endungen sind kein Kriterium mehr (vierter Durchgang, Spec „Abdeckung“).

- [x] **Step 1: Tests rot.**
  - kein `path` → `loomux`; die Zähler der Antwort stehen im `render`-Text.
  - `path` innerhalb der Repo-Wurzel → `loomux`, auch wenn dort ignorierte Dateien liegen (Zähler, kein Fehler).
  - `path` außerhalb der Repo-Wurzel → `error` mit Hinweis.
  - `include: "*.md"` → `loomux` ohne Fehler (Spec, Abdeckung; Frage 14 beschlossen).
  - `include: "*.go"` → `loomux`.
  - `path: internal/code` trifft nicht `internal/codex/…` (Präfixgrenze an der Pfadsegmentgrenze; die Semantik von `lexicon.UnderPrefix` vorher lesen).
  - Lauf → FAIL.
- [x] **Step 2: Implementieren.** Prüfung nur gegen die Repo-Wurzel (`path`) und die Form der Antwort aus A8. Keine Endungsliste im Plugin.
- [x] **Step 3: Grün.**

**Akzeptanz:** Es gibt keine Konstellation, in der `loomux` mit einem leeren Treffer antwortet, ohne dass die Zähler (`ignored`, `skipped_binary`, `skipped_size`, `unreadable`) im Ergebnis stehen. Ein `path` außerhalb der Repo-Wurzel ist immer ein Fehler.

**Ergebnis:** `dsh-loomux/src/route.js` mit `underPrefix` an der
Pfadsegmentgrenze und `route`; 10 Tests in `test/route.test.js` grün. Ein Pfad
außerhalb der Wurzel ergibt einen Fehler mit Öffnen-oder-Shell-Hinweis; ein
fehlendes `path` routet immer zu loomux.

**Commit-Vorschlag (Mensch):** `feat: route grep to the graph only where it covers the search`

---

### Task B4: Fehlerpfade

**Files:**
- Create: `dsh-loomux/src/errors.js`
- Create: `dsh-loomux/test/errors.test.js`

**Interfaces:**
- Produces: `classify(exit, stderr, { aborted, stdoutBytes, spawnError }) → 'no-binary' | 'no-graph' | 'invalid-regex' | 'unsupported' | 'prefix-not-indexed' | 'usage' | 'timeout' | 'stdout-cap' | 'unknown'`. Exit 1 ist nie `ok`.

- [x] **Step 1: Tests rot.** Wortlaute aus dem Go-Code: `no graph. Run` → Fehler mit Hinweis `loomux graph build`; `invalid regex:` → Fehler; `not supported in RE2 regex:` → Fehler; `prefix not indexed:` → Fehler; Exit 1 ohne bekannte Meldung → Fehler, nie leer; Exit 2 → Fehler (Aufrufsfehler des Plugins); fehlendes `loomux` (ENOENT) → Fehler mit Hinweis (Frage 10); Abbruch → Fehler mit Hinweis; stdout über dem Deckel → Fehler, kein Abschneiden; stderr-Notiz bei Exit 0 → Notiz im Ergebnis.
- [x] **Step 2: Implementieren.** Der Wortlaut-Abgleich ist fragil; die Tests pinnen die Wortlaute aus `internal/code/grep/grep.go:80,89,108` und `internal/code/query/ask.go:16`. Die Ist-Klassifikation (`index.js:1012`, nur Exit ≥ 2 als Fehler) wird nicht übernommen.
- [x] **Step 3: Grün.**

**Akzeptanz:** Eine leere Liste wird nie als Antwort auf einen Fehler zurückgegeben (Anti-Ziel A3 der Spec).

**Ergebnis:** `dsh-loomux/src/errors.js` mit neun Klassen, Hinweisen und
`errorFor`; 9 Tests in `test/errors.test.js` grün. `run.js` kappt stdout bei
20 MiB und meldet das als Fehler, nie als Abschneiden; ein fehlendes Binary
(ENOENT) und ein Abbruch sind eigene Klassen.

**Commit-Vorschlag (Mensch):** `feat: classify loomux graph grep failures`

---

### Task B5: Rückfall (entfällt)

**Entscheidung:** Spec, Fragen 3 und 10. Kein Rückfall, kein tgrep, keine optionale Binary, keine Textsuche im Plugin. Dieser Task wird nicht umgesetzt; die Nummer bleibt, damit die Referenzen stabil bleiben. Die Fehlerfälle mit Hinweis stehen in B4.

**Files:** keine.

**Akzeptanz:** Das Plugin enthält kein Rückfall-Modul. Fehlende Hilfsmittel ergeben einen Fehler mit Hinweis (B4), nie eine leere Trefferliste.

---

### Task B6: Privacy-Vertrag des Plugins

**Files:**
- Modify: `dsh-loomux/test/args.test.js`

- [x] **Step 1: Test.** Ein Test prüft, dass die vom Plugin erzeugte Argumentliste für alle Eingaben `--channel cloud` enthält. Einen Rückfall gibt es nicht (Spec, Frage 10), also keinen zweiten Suchweg im Plugin. Der Shell-Weg des Agenten liegt außerhalb des Plugins und bleibt offen (Spec, Privacy, Frage 6).
- [x] **Step 2: Grün.**

**Akzeptanz:** Der Vertrag steht als Test, nicht nur im Text.

**Ergebnis:** In `test/args.test.js` prüfen zwei Fälle, dass `--channel cloud`
in jeder erzeugten Liste steht und keine Eingabe `--channel` erreicht; die
Rendering-Tests halten die Zähler im `render`-Text fest.

---

### Task B7: Gesamtlauf und Coverage

**Files:** keine neuen.

- [x] **Step 1:** `node --test --experimental-test-coverage` (Node >= 22.19; `engines` des Ist-Pakets, `npm test` = `node --test`). Ziel 100 % je Funktion (globale Projektanweisung: 100 % Coverage). Abweichungen mit Begründung im Plan vermerken.
- [x] **Step 2: Lauf gegen das reale Profil.** `loomux graph grep` aus dem Plugin gegen das loomux-Repo, eine Suche je Routing-Fall. Dabei wird geprüft, welches `grep` antwortet, solange `dsh-tgrep` noch aktiv ist (Spec, Frage 12).

**Akzeptanz:** Coverage 100 % oder begründete Ausnahme; reale Läufe dokumentiert.

**Ergebnis:** `node --test` in `dsh-loomux`: 59 Tests grün.
`node --test --experimental-test-coverage`: Zeilen 100,00 %, Funktionen
100,00 %, Branches 87,79 % (`index.js`, `args.js`, `route.js` je 100/100;
`run.js` 75 %, `errors.js` 90 %, `result.js` 92,73 % Branches). Die
Branch-Lücken sind `??`-Rückfälle und Fehlerzweige, die nur mit einem
defekten Host erreichbar wären. Reale Läufe: `.cache/plugin-integration.mjs`
gegen `bin\loomux.exe` (Treffer, leere Antwort, Glob, ungültige Regex, Pfad
außerhalb, fehlendes Binary, Meta) und `.cache/plugin-profile-run.mjs` gegen
die im Profil installierte Kopie
(`node_modules/dsh-loomux/index.js`) mit `LOOMUX_BIN` auf den Repo-Build;
Routing, Glob und Karte grün. **Frage 12 bleibt offen:** der laufende
DSH-Prozess lädt noch `dsh-tgrep`, sein `grep`-Tool ist das von tgrep; welches
`grep` nach einem Neustart mit beiden Paketen antwortet, lässt sich nur dort
beobachten (kein Ersatzserver gestartet).

---

### Task B8: Veröffentlichung

**Files:**
- Create: `dsh-loomux/package.json` (Name, Version fixiert, Node-Engine `>=22.19`)
- Modify: `cordis.patch.yml` im Profil (`insert`-Eintrag, wie `dsh-tgrep`)

- [x] **Step 1: Paketname und Version.** Neueste Version der Abhängigkeiten recherchieren (Registry), exakt pinnen. Pflegestand prüfen (letzte Releases, offene Issues). Vorlage: `package.json` von `dsh-tgrep` 0.1.13 (MIT, `type: module`, `exports`, `files`, `engines`, `dsh.bundle.patch`, `publishConfig`).
- [ ] **Step 2: Veröffentlichung.** `npm publish` oder `PublishPlugin` nur nach ausdrücklicher Freigabe des Menschen. Ein Agent veröffentlicht nicht eigenständig.
- [x] **Step 3: Profil eintragen.** `dsh-loomux` als `insert`-Eintrag mit neuer `id` im Profil-Patch registrieren, wie `dsh-tgrep`. `dsh-tgrep` bleibt in dieser Phase aktiv; das Abschalten und Entfernen ist Task C4 und erst nach dem Gate zulässig (Spec, Fragen 11 und 12).

**Akzeptanz:** Vor jeder Veröffentlichung liegt eine ausdrückliche Freigabe des Menschen vor. Das Abschalten von `dsh-tgrep` steht nicht in diesem Task (C4).

**Ergebnis:** `dsh-loomux/package.json` 0.1.0, `type: module`, `exports`,
`files`, `engines.node >=22.19.0`, `dsh.bundle.patch`, keine
Laufzeit-Abhängigkeiten. Die Profilregistrierung ist erfolgt: `package.json`
des Profils `desktop` führt `file:dsh-loomux` und den Bundle-Eintrag;
`plugin_manager list_bundles` zeigt `dsh-loomux` als Profil-Bundle mit
`rows: [{rowId: "loomux", moduleName: "dsh-loomux"}]`, `enabled: true`,
`installed: true`. Der Profil-Patch setzt `binary` auf den Repo-Build, weil das
`loomux` im PATH 1.17.0 ist und `--glob`/`--channel` nicht kennt.
`npm publish` ist **nicht** ausgeführt und wartet auf die Freigabe des
Menschen.

---

## Teil C: Gate und Entfernung von tgrep

Verbindliche Benchmark-Aufgabe (Entscheidung des Menschen). Sie steht vor dem Entfernen von tgrep und vor dem Abschalten von `dsh-tgrep`. tgrep ist hier nur Messwerkzeug, nie Laufzeit-Abhängigkeit des Plugins.

**Abhängigkeiten (explizit):**
- C3 braucht A4d, A5, A6 und A8 (Index-Pfad, `graph grep --json`, Suchkorpus-Record) sowie C1 und C2.
- C4 braucht: C3 bestanden, B8 Schritt 3 (`dsh-loomux` im Profil registriert), B7 Schritt 2 (Lauf gegen das reale Profil) und Spec-Frage 12 (belegt, welches `grep` antwortet).

### Task C1: tgrep als Entwicklungswerkzeug installieren

**Files:** keine im Repo.

- [x] **Step 1: Version ermitteln.** Neueste Release-Version von `https://github.com/microsoft/tgrep/releases` oder per `gh api repos/microsoft/tgrep/releases/latest`. Die Version steht im Benchmark-Eintrag; vorab wird keine festgelegt.
- [x] **Step 2: Installieren.** Windows-x64-Zip aus dem Release nach `.cargo\bin` entpacken, oder `cargo install --path tgrep-cli --locked` aus dem Quellcode.
- [x] **Step 3: Prüfen.** `tgrep --version`; Ausgabe in den Eintrag übernehmen.
- [x] **Step 4: Nur per absolutem Pfad aufrufen.** Ist `.cargo\bin` auf dem PATH des DSH-Hosts, kann `dsh-tgrep` (`autoStartDaemon: true`) Daemons starten. Ob das so ist, ist nicht belegt; der Plan vermeidet es.

**Akzeptanz:** `tgrep --version` läuft; Version und Installationsweg stehen im Benchmark-Eintrag.

**Ergebnis:** tgrep **1.2.3** liegt bereits als WinGet-Paket unter
`C:\Users\micro\AppData\Local\Microsoft\WinGet\Packages\Microsoft.tgrep_Microsoft.Winget.Source_8wekyb3d8bbwe\tgrep.exe`;
alle Läufe rufen diesen absoluten Pfad auf. `--version` und `--help` geprüft
(`--files`, `--index-path`, `--max-filesize`, `--no-ignore`, `--no-unicode`,
`--vimgrep`).

---

### Task C2: Vergleichsmethode festlegen

**Files:** keine im Repo; die Methode steht im Benchmark-Eintrag.

- [x] **Step 1: Gleiche Dateiliste auf dem echten Repository.** Der Vergleich läuft auf dem loomux-Repo selbst (Gate-Methode, Entscheidung 5). Beide Seiten bekommen dieselben Ignore-Regeln und dieselbe Größenpolitik: loomux mit A4b und `--max-filesize`; tgrep mit den Flags, die C1 in `tgrep --help` für Ignore-Regeln und Größe findet. Die tgrep-Flags sind nur Benchmark-Notiz (Spec, Frage 13). Vor der Messung wird die Gleichheit der Dateisätze geprüft: (a) Anzahl; (b) Stichprobe von 50 Pfaden mit festem Seed; (c) Mengendifferenz. Quelle der loomux-Liste: die Dateitabelle des Suchindex (`trigram.Read(...).Files()`). Orakel: `git ls-files --cached --others --exclude-standard`, gefiltert mit denselben Binär-, Größen- und Punkt-Regeln; je besuchtem Pfad `git check-ignore --stdin`. Kann tgrep seine Dateiliste nicht ausgeben, bleibt die Trefferprüfung, und das wird im Eintrag benannt.
- [ ] **Step 1b: Kontrolllauf (zulässig).** Die kopierte Liste aus Step 1 in ein leeres Verzeichnis außerhalb des Repos, ohne `.git`. Beide Werkzeuge laufen dort ungefiltert. Das stützt die Methode, ersetzt aber nicht den Lauf auf dem Repo.
- [x] **Step 2: Gleiche Treffer.** `loomux graph grep` mit `--max-hits` über der Trefferzahl; `truncated_hits == 0` prüfen. tgrep ohne Kappung. Verglichen wird die Menge der `(path, line)`-Paare, Pfade relativ mit `/`. Ungleiche Mengen = Gate nicht bestanden; Ursache im Eintrag.
- [x] **Step 3: Kalt und warm, für beide gleich definiert.** Kalt: erster Lauf nach Neuaufbau des Index (loomux: `graph build --no-reuse`; tgrep: Index in einem neuen `--index-path`, Erzeugungszeit getrennt ausgewiesen). Warm: Median von 5 Läufen nach einem Vorlauf. Der Betriebssystem-Dateicache wird nicht geleert; das steht im Eintrag. Einstellungsregel für Messläufe (Spec, „Vergleich gegen den Record“): loomux baut mit `graph build --max-filesize X --no-refresh` und misst mit `--no-refresh --max-filesize X`; zwischen Build und Messung kein Lauf ohne Flag.
- [x] **Step 4: Muster.** Literal (`func Search`), Alternation (`Search|Grep`), Muster ohne Literal (`[A-Z]{3}\w+`), `-i` mit ASCII-Muster. Nicht-ASCII mit `-i` fällt bei loomux auf den Scan zurück und ist deshalb nicht Teil des Gates. Muster müssen in beiden Werkzeugen gleich gelesen werden; Abweichungen werden vermerkt.
- [x] **Step 5: Zweiter Korpus `iam_backend`.** Pfad `C:\Users\micro\Documents\#GIT\iam_backend` (Python, Spec, Frage 15, beschlossen). **Nicht gelesen.** Vor dem Lesen den Zugriff bestätigen lassen; in einer früheren Sitzung fehlte er. Ohne Zugriff: Eintrag „nicht gemessen“, das Gate gilt dann nur für das loomux-Repo, und das steht im Eintrag. Mit Zugriff: dieselbe Methode wie Step 1.

**Akzeptanz:** Die Methode steht vor der Messung im Eintrag, mit Begründung für jeden Punkt.

**Ergebnis:** Die Methode steht im Benchmark-Eintrag vom 2026-11-06. Abweichungen,
alle dort benannt: (1) tgrep schreibt mit `--vimgrep` (sein `--json` erzeugt für
`-i search` über 512 MB, weil jeder Submatch einzeln serialisiert wird), loomux
mit `--json`; (2) beide lesen ASCII-Klassen, tgrep mit `--no-unicode`; (3) das
`-i`-Muster ist `func search` statt `search` (dasselbe Muster in beiden
Werkzeugen, anders wäre der Trefferberg nicht vergleichbar); (4) der
Kontrolllauf Step 1b entfällt: ein Prozess dieser Sitzung darf außerhalb des
Arbeitsbereichs keinen Zustand anlegen, `graph build` scheitert dort mit
`Zugriff verweigert`; (5) `iam_backend` wurde nicht an Ort und Stelle gemessen,
sondern als Kopie aus tgrep's 517-Dateien-Liste im Arbeitsbereich; das echte
`.loomux` hat keinen Trigramm-Index, und Schreiben dorthin ist gesperrt. Die
Dateisatz-Gleichheit auf dem Repo: tgrep 7668, loomux 7662, Differenz genau die
sechs Binärdateien; die Treffermengen stimmen für alle vier Muster exakt.

---

### Task C3: Messung und Gate-Entscheid

**Files:**
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md` (neuer Eintrag, chronologisch)

**Interfaces:** keine Codeänderung.

- [x] **Step 1: Messen.** Nach C2 auf dem loomux-Repo und dem zweiten Korpus, je Muster, kalt und warm, für loomux (Index-Pfad) und tgrep. Vorher die Dateisatz-Gleichheit aus C2 Step 1 prüfen; Abweichungen stehen im Eintrag.
- [x] **Step 2: Lesart anwenden.** „Mindestens gleich schnell“ heißt: Median loomux ≤ Median tgrep, je Muster, je Korpus, warm und kalt. Diese Lesart steht im Eintrag; eine andere braucht die Zustimmung des Menschen.
- [x] **Step 3: Eintrag schreiben.** Datum, Uhrzeit, Befehl, Baseline gegen Änderung (A0-Werte und Index-Werte), tgrep-Version, Methode aus C2. In beiden Sprachen.
- [x] **Step 4: Gate-Entscheid.** Bestanden: C4 ist zulässig. Nicht bestanden: Ursache im Eintrag; C4 wird verschoben, bis die Lücke geschlossen ist oder der Mensch anders entscheidet.

**Akzeptanz:** Beide `benchmarks.md` haben den Eintrag mit Gate-Entscheid; keine Abweichung ist verschwiegen.

**Ergebnis: Gate nicht bestanden.** Repo warm: Literal 279,3 ms gegen 21,2 ms,
Alternation 427,5 gegen 51,3, ohne Literal 817,4 gegen 4.621,7 (loomux
gewinnt), `-i` 306,6 gegen 82,3. `iam_backend`-Kopie warm: 124,7 gegen 31,6,
227,2 gegen 40,8, 577,9 gegen 434,0 (loomux gewinnt), 121,6 gegen 24,1. Die
Treffermengen sind in beiden Korpora für alle vier Muster identisch; die
Latenz kommt aus dem Prozessstart (16 MB Wiring plus 13 MB Index je Aufruf).
Eintrag: `docs/en/benchmarks.md` und `docs/de/benchmarks.md`, 2026-11-06 11:35
mit Build-Zeiten (loomux 3829 ms, tgrep 503 ms) und beiden Tabellen. C4 bleibt
damit gesperrt; tgrep und `dsh-tgrep` bleiben aktiv.

**Commit-Vorschlag (Mensch):** `docs: record the grep benchmark against tgrep`

---

### Task C4: tgrep-Erbe entfernen (nur nach Gate)

**Files:**
- Modify: Profil-Patch (`cordis.patch.yml` im Profil): Eintrag `id: tgrep` (`dsh-tgrep`) entfernen. Der Ort im Profil ist nicht belegt; vor dem Ändern ermitteln.
- Delete: `.tgrep/` im Workspace `C:\Users\micro\Documents\#GIT\loomux` (nicht committen).

**Vorbedingungen (alle):**
- C3: Gate bestanden. — **nicht erfüllt** (C3 gescheitert).
- B8 Schritt 3: `dsh-loomux` im Profil registriert. — erfüllt.
- B7 Schritt 2: Lauf gegen das reale Profil grün. — erfüllt, Frage 12 offen.
- Spec, Frage 12: belegt, welches `grep` antwortet. — offen.

**Nicht ausgeführt.** Das Gate ist gescheitert; nach der verbindlichen
Entscheidung bleibt tgrep installiert und `dsh-tgrep` aktiv, bis die
Latenzlücke geschlossen ist oder der Mensch anders entscheidet. Kein Schritt
dieses Tasks wurde begonnen; `.tgrep/` und der Profil-Eintrag sind unverändert.

- [ ] **Step 1: Abschalten.** `dsh-tgrep` im Profil-Patch deaktivieren (Konfiguration `enabled: false`; Beleg Spec, `index.js:467–470`) und prüfen, dass `grep` von `dsh-loomux` beantwortet wird (`render`-Text).
- [ ] **Step 2: Prozesse beenden.** Pid aus `<workspace>/.tgrep/serve.json` lesen, sofern vorhanden, und nur diesen Prozess beenden. Fehlt die Datei (Stand 2026-11-03: fehlt), Prozessliste auf `tgrep` prüfen und nichts blind beenden.
- [ ] **Step 3: Eintrag entfernen.** `id: tgrep` aus dem Profil-Patch löschen, nachdem Step 1 geprüft ist.
- [ ] **Step 4: Workspace aufräumen.** `.tgrep/` ansehen (Stand 2026-11-03: nur ein leeres `serve.log`), dann entfernen. Nicht committen.
- [ ] **Step 5: PATH-Abhängigkeit prüfen.** Den Profil-Ordner nach Aufrufen von `tgrep` durchsuchen; außer dem Paketordner selbst darf es keinen Treffer geben.
- [ ] **Step 6: Paketordner.** `dsh-tgrep` in `node_modules` bleibt, bis der Mensch entscheidet (offen, Spec, Frage 11).

**Akzeptanz:** `grep` wird von `dsh-loomux` beantwortet; kein `tgrep serve` läuft; `.tgrep/` fehlt im Workspace; kein Profil-Eintrag ruft `tgrep` auf.

**Commit-Vorschlag:** keiner. Im Repo ändert sich nur das Entfernen von `.tgrep/`, und das wird nicht committet.

---

## Self-Review

- **Spec-Abdeckung:** Ziele Z1–Z6 → A1–A9, B1–B4. Privacy → A7, B6. Schnittstellen → A8, B1, B2. Fehlerfälle → A6, A9, B4 (B5 entfällt). Teststrategie → je Task. Rollout → A0, A10, B7, B8, C1–C4. Offene Fragen 1–15 → Abschnitt Offene Entscheidungen und Spec.
- **Typen:** `trigram.File`, `trigram.Index`, `grep.Result` (um `Index`, `FilesInScope`, `Candidates`), `query.GrepOptions` (um `Channel`, `Glob`) sind in den Tasks einheitlich benannt.
- **Platzhalter:** Der Schritt A3 Step 3 enthält eine Umformulierung zur Golden-Datei; die Vorgehensweise ist festgelegt (Erzeugung per Hilfsprogramm im Test, Prüfung per Hand). Der Schritt A7 Step 1 ist eine Vorarbeit mit offener Funktionsnamenswahl, die beim Lesen von `internal/serve` festgelegt wird.
- **Lücken:** Die Plugin-Seite ist belegt (B0). Entschieden sind jetzt Frage 14 (Suchkorpus) und 15 (`iam_backend`, nicht gelesen). Offen bleiben die Host-Semantik (Frage 12), Namensmuster und Symlinks (16), Graph-Größenlimit (17), der Konfigurationsabschnitt `[grep]` (18, blockiert `max_filesize`), `--hidden` (19), UTF-16 (20), die Git-Standardquelle und `core.ignorecase` (21, im Test belegen) und der Paketordner `dsh-tgrep`. Die tgrep-Flags (Frage 13) sind nur Benchmark-Notiz.
- **Gate-Reihenfolge:** C3 nach A5, A6, A8, C1, C2; C4 nach C3, B8 Schritt 3, B7 Schritt 2 und Spec-Frage 12.
- **Commits:** Kein Schritt führt einen Commit aus. Die Commit-Vorschläge sind Header für den Menschen.
