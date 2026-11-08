# Startlücke von `graph grep` schließen: Design

Stand 2026-11-08. Grundlage: `docs/.superpowers/research/2026-11-07-grep-startluecke.md`
(Spikes S1–S16, Messungen, Auswertung des tgrep-Quellcodes). Der Spike liegt im
Worktree `.claude/worktrees/spike-grep-startup` (Zweig `spike/grep-startup`) und
ist Wegwerfcode; dieses Design beschreibt den echten Bau.

## Anlass und Ziel

tgrep und `dsh-tgrep` sollen abgelöst werden. Dafür muss `loomux graph grep` als
Einmalaufruf schnell genug sein. Vorgabe des Menschen:

- loomux muss tgrep nicht schlagen, aber etwa auf dessen Niveau liegen und auf
  jeden Fall besser sein als ein normales grep. Vergleich: `git grep` und
  `findstr` unter Windows, `grep -rn` mit Ausschlüssen unter Linux.
- Das gilt auch auf einem geänderten Arbeitsbaum, nicht nur direkt nach einem
  Build.
- Die Frischeprüfung lässt sich abschalten. Abgeschaltet muss loomux so schnell
  sein wie tgrep.

Heute kostet ein Aufruf unter Windows 440–1140 ms auf dem sauberen Baum und
~4 s nach einer Änderung, weil zwei serielle Walks laufen und jede Drift einen
Neubau auslöst.

## Ziele

1. **Mit Prüfung (Standard):** korrekte Treffer auf jedem Baumzustand, bei
   den im Spike gemessenen Zeiten (Windows, Baum mit ~8400 Dateien): Literal
   ≤ ~90 ms, Alternation ≤ ~110 ms, `-i` ≤ ~95 ms, ohne Literal ≤ ~270 ms;
   nach einer Änderung an bis zu 200 Dateien kein Neubau, ≤ ~350 ms.
2. **Ohne Prüfung (`--trust-index`):** Literal, Alternation und `-i` höchstens
   so langsam wie tgrep im Einmalaufruf (gemessen: 23/50/31 ms gegen
   23/66/94 ms).
3. **Gleiche Treffer:** Das JSON gleicht dem heutigen in Gruppen, Reihenfolge,
   `path`, `symbol.id`, `in_degree` und Trefferzeilen, auf sauberem Baum für
   alle vier Benchmark-Muster. Mit `--trust-index` gilt das auf einem Baum,
   der zum Index passt.
4. **Linux:** mit Prüfung auf dem Niveau von `grep -rn --exclude-dir=.git
   --exclude-dir=.loomux` (gemessen 42/68/50/112 gegen 48/57/60/163 ms).

## Nicht-Ziele

- `graph_grep` als Tool im Dienst mit Schranke vor jeder Antwort (Kandidat
  C1). Er ist der einzige Weg zu tgreps Niveau *mit* Prüfung und kommt als
  eigenes Design mit eigenem Spike. Der Wächter aus Abschnitt 7 ist dafür die
  Grundlage.
- Exakte Trigramme zusätzlich zu den gefalteten (C3). Die Gegenprüfung fand
  einen Fehler im Vorschlag (Fold-Flags je Literal) und nur 3–7 ms Gewinn.
- Schlankerer Prozessstart (C6) und gestreamte JSON-Ausgabe (C8): im
  Messrauschen.
- NTFS-USN-Journal, Verzeichnis-mtimes, git-Index-Stempel als Ersatz für den
  Walk: gemessen oder belegt nutzlos (siehe Research-Dokument).
- GC abschalten (S10): im Rauschen.
- Änderungen an anderen Graph-Befehlen außer dem gemeinsamen Walk.

## Anti-Ziele

- **Stumm falsche Treffer im Standardmodus.** Was tgrep im Einmalaufruf tut,
  darf loomux nur mit ausdrücklichem `--trust-index`.
- **Ein kaputter Index, der „keine Treffer“ meldet.** Jede Prüfung, die beim
  Lesen von Index oder Beiakte scheitert, fällt auf den Scan bzw. den
  bisherigen Weg zurück, nie auf ein leeres Ergebnis.
- **`--no-refresh` umdeuten.** Es behält seine Bedeutung („nicht neu bauen,
  aber den aktuellen Baum lesen“).

## Entscheidungen des Menschen

1. Prüfung abschaltbar über einen **eigenen Schalter `--trust-index`**;
   `--no-refresh` bleibt, wie es ist.
2. **Prüfung standardmäßig an**, in der CLI wie im Plugin. Das Plugin bekommt
   eine Option `trustIndex` (Standard `false`).
3. Übernommen werden die Spikes S1–S9, S11, S12, S14–S16 und der gespeicherte
   Digest (C5). S10 entfällt.
4. `loomux serve` bekommt einen **Wächter, der unabhängig von Aufrufen neu
   baut** (Abschnitt 7). Damit gibt es keine Zeitgrenze für das Overlay;
   ohne Dienst baut der Aufruf ab 200 geänderten Dateien neu.

## Architektur

Alle Änderungen liegen unter `internal/code/` und `internal/cli/graph.go`, dazu
das Plugin `dsh-loomux`.

### 1. Walk (`corpus`)

- **Paralleler Walk (S1):** Verzeichnisse werden parallel gelesen, begrenzt
  auf 16 gleichzeitige Leser (gemessen gesättigt; der Spike nahm 4×CPU). Die
  Ausgabe bleibt nach Pfad sortiert. `*corpus.Rules` ist nicht
  nebenläufigkeitsfest (`Ignored` lädt Ignore-Dateien nach); der Matcher
  läuft daher hinter einem Mutex. Die Ignore-Auswertung lädt schon heute die
  ganze Verzeichniskette vor dem Abgleich, die Reihenfolge des Walks ändert
  am Ergebnis nichts (die frühere Annahme dazu war falsch).
- **Ein Walk pro Aufruf (S2):** *Meine Entscheidung, einfacher als der
  Spike:* Es gibt nur den Walk mit Matcher. Die Dateimenge des Graphen ergibt
  sich daraus über einen Filter (`sourceset.FromListing`), denn der Walk mit
  Matcher markiert ignorierte Dateien nur und überspringt lediglich
  Punkt-Verzeichnisse und ignorierte Prune-Verzeichnisse ohne verfolgte
  Datei, die `sourceset` ohnehin verwirft. Die Frischeprobe des Graphen nimmt
  diese Liste entgegen (`freshness.ProbeFiles`). Ohne Record läuft der Walk
  wie heute ohne Matcher.

### 2. Beiakten des Builds (`query`, `corpus`, `trigram`)

Alle Beiakten werden von `graph build` geschrieben, atomar (temp + rename),
mit Formatversion und gebunden an den Graphen über den Corpus-Digest. Passt
eine Beiakte nicht, nimmt der Aufruf den heutigen Weg. *Meine Entscheidung
(ersetzt die frühere Regel „fehlende Beiakte baut einmal neu“):* Der Record
bekommt eine neue Version (gespeicherter Digest), alte Records gelten als
fehlend, und ein fehlender Record baut schon heute neu. Damit baut jeder
Baum nach dem Upgrade genau einmal neu, ohne neue Regel. Fehlt danach nur
die Spans-Beiakte, liest `grep` `wiring.json` (langsam, korrekt), ohne
Neubau. `ErrNoIndex` bleibt `none`, `ErrCorrupt` bleibt `stale`.

- **Bindung der Spans-Beiakte:** Die Spans hängen am Graphen, der Digest am
  Suchkorpus. `graph build` löscht `grepspans.idx` deshalb vor
  `store.Write` und schreibt sie als Letztes, nach `writeCorpus`, mit dessen
  Digest. Scheitert ein Schritt dazwischen, fehlt die Beiakte, und `grep`
  nimmt `wiring.json`.
- **Stabile Span-Reihenfolge:** `model.FileSpans` sortiert stabil mit der
  Knoten-ID als letztem Kriterium, sonst wäre „JSON mit Beiakte = JSON ohne“
  bei gleichen Spans nicht verlässlich.

- **Spans-Beiakte (S3, S9, S16):** eine Datei `cache/grepspans.idx`, Layout
  wie im Spike: Kopf (Magic, Version, Digest), `u32 n`, `n × u32` Offsets der
  nach Pfad sortierten Einträge, Einträge aus Pfad und Block. Ein Block hält
  die Spans einer Datei samt Knoten ohne `body_text` und `in_degree`, schon
  sortiert. *Meine Entscheidung:* Blockformat JSON (die Gegenprüfung maß für
  ein Binärformat < 1 ms Gewinn). Gelesen per mmap; dekodiert werden nur
  Dateien mit gezählten Treffern. `grep` liest `wiring.json` nur noch, wenn
  die Beiakte fehlt oder nicht passt.
- **Records binär (S8):** `corpus.json` und `fingerprint.json` bleiben die
  kanonischen Dateien. `graph build` schreibt daneben eine binäre Fassung
  (gob mit Version), statt wie im Spike beim ersten Lesen. Die binäre Fassung
  trägt Größe und mtime (ns) des JSON; der Leser nimmt sie nur, wenn beide
  zur JSON-Datei passen, sonst das JSON. `freshness.Write` wird dabei atomar.
- **Ein Helfer für atomares Ersetzen:** `lock.replaceEventually`
  (`internal/lock/replace.go`) wird exportiert, mit einstellbarem Budget, und
  von allen Schreibern dieser Beiakten benutzt.
- **Gespeicherter Digest (C5):** Der Corpus-Digest wird beim Build in den
  Record geschrieben und beim Lesen verglichen, nicht neu berechnet (~3 ms).
- **Zähler im Indexkopf:** `ignored`, `skipped_size`, `skipped_binary` und
  das `ignorecase`-Flag kommen zusätzlich in den Kopf von `trigram.idx`,
  damit `--trust-index` ohne Record dieselben Zähler und Einstellungen kennt.
  Das hebt die Formatversion des Index.

### 3. Index lesen (`trigram`)

- **Lazy (S7):** Kopf und Dateitabelle werden gelesen, ein Trigramm wird binär
  in der sortierten Tabelle gesucht, nur seine Postings werden gelesen und
  geprüft.
- **mmap (S15):** Der Index wird nur lesend eingeblendet (Windows
  `CreateFileMapping`/`MapViewOfFile`, sonst `syscall.Mmap`) und nach dem
  Aufruf wieder freigegeben: `Index` bekommt `Close()`, und `Grep` schließt
  ihn vor der Rückkehr. Das gilt auch für `graph_find_all`, das `query.Grep`
  im Dienst aufruft.
- **Kaputte Postings:** Die Suche eines Trigramms meldet getrennt „nicht
  vorhanden“ und „ungültig“. Bei „ungültig“ gibt `Candidates` `used=false`
  zurück, und der Aufruf scannt.
- **Ersetzen unter Windows:** Solange irgendein Prozess die Datei abgebildet
  hat, scheitert das `os.Rename` des Schreibers (Zugriff verweigert). Das
  betrifft `trigram.idx` und `grepspans.idx`. *Meine Entscheidung:* Der
  Schreiber wiederholt das Rename mit kurzem Backoff bis etwa 2 s (ein
  Einmalaufruf hält die Abbildung ~30 ms); `serve` gibt die Abbildung nach
  jeder Anfrage frei. Scheitert es trotzdem, schlägt der Build mit klarer
  Meldung fehl, und der alte Index bleibt gültig. Kein prozessübergreifendes
  Lese-Lock.
- **Rückfall:** Scheitert eine Prüfung (Länge, Offset, Sortierung), meldet
  `Candidates` „nicht benutzt“, und der Aufruf scannt. Der Spike lieferte hier
  stumm `nil`; `TestParsePostingNamesNoFile` deckt das ab.

### 4. Drift-Overlay nur für `grep` (S4)

- Ist die Drift reiner Dateiinhalt (geändert, neu, gelöscht) und betrifft
  höchstens 200 Dateien, baut `grep` weder Graph noch Korpus neu. Der Index
  liefert die Kandidaten der unveränderten Dateien; geänderte und neue Dateien
  werden immer gelesen, gelöschte fehlen in der aktuellen Liste. `index`
  meldet `overlay`, auch bei einem Muster ohne Literal.
- *Meine Entscheidungen zur genauen Bedingung:* Gezählt werden geänderte,
  neue und gelöschte Dateien, und zwar beide Driften: die des Suchkorpus
  (`corpus.Compare`) und die des Graphen (`freshness.ProbeFiles`), jede
  höchstens 200. Fehlt der Fingerprint des Graphen, hat er einen anderen
  Extraktor oder ist die Ask-Beiakte unbrauchbar (`lexicon.Usable`), wird neu
  gebaut wie heute in `ask.Refresh`. Die Reihenfolge in `Grep` ändert sich
  dafür: erst Record, Walk und beide Driften, dann Overlay oder Neubau; `grep`
  ruft `loadGraph`/`EnsureFresh` nicht mehr auf.
- Andere Drift (Einstellungen, Konfiguration, Ignore-Quellen) und mehr als
  200 Dateien lösen wie heute einen Neubau aus.
- **`.git/index` bewegt:** Die Datei ändert sich bei jedem `git add`, Commit,
  Checkout und oft bei `git status`. Löste das einen Neubau aus, griffe das
  Overlay im Alltag (ändern, committen, suchen) fast nie. *Meine
  Entscheidung:* Ist der git-Index die einzige Nicht-Inhalts-Drift, wird die
  Menge der verfolgten Dateien neu bestimmt (wie beim Build) und mit
  `rec.Tracked` verglichen. Ist sie gleich, gilt das Overlay; sonst Neubau.
  Bestimmt wird sie mit einem einzigen `git ls-files -z --cached` (neuer
  Helfer `corpus.TrackedSet`), nicht mit dem vollen `corpus.Load` (~8
  git-Aufrufe), gemessen ~20 ms. Ist die Menge gleich, schreibt `grep` den
  neuen Stempel von `.git/index` in den Record (Digest unverändert), auch mit
  `--no-refresh`, damit nur der erste Aufruf nach dem Commit den git-Aufruf
  zahlt (Entscheidung des Bauenden während der Umsetzung, Vorschlag an den
  Menschen ohne Antwort). Die Kosten gehören in die
  Benchmark-Abnahme (Szenario „nach einem Commit“). Ein bekannter Fehler bei
  einer Wurzel in einem Unterverzeichnis des Repos (`ls-files` liefert Pfade
  relativ zur Wurzel, die Regeln erwarten sie mit Präfix) wird nicht hier
  behoben, sondern als eigener Punkt notiert.
- **`--no-refresh`:** *Meine Entscheidung:* Das Overlay gilt auch dort, denn
  es baut nicht neu. `index` meldet dann `overlay` statt `stale`.
- Der Schalter sitzt nur im `grep`-Pfad, nicht in `loadGraph`. `skeleton`,
  `audit`, `blast` usw. verhalten sich wie heute (im Spike war das falsch).
- **Symbole in geänderten Dateien:** Die Spans stammen aus dem alten Graphen.
  *Meine Entscheidung:* Treffer in geänderten und neuen Dateien kommen ohne
  Symbol und mit `in_degree` 0, wie eine Datei ohne Knoten. Lieber kein
  Symbol als ein falsches. Neu extrahieren ist ein späterer Schritt.
- Die Grenze von 200 ist eine Konstante, keine Einstellung.

### 5. Scan (`grep`)

- **Parallel (S5)** auf `runtime.NumCPU()` Workern; die Reihenfolge des
  Ergebnisses bleibt die heutige.
- **Literal-Vorfilter (S11):** Die Pflicht-Literale aus `trigram.Literals`
  (Disjunktion von Konjunktionen, ASCII-kleingeschrieben) filtern erst die
  Datei, dann jede Zeile, per `bytes.Contains` auf einer ASCII-kleinen Kopie.
  Ist die Regex irgendwo case-insensitiv (`-i` oder ein `(?i)` im Muster),
  überspringt eine Datei mit Nicht-ASCII-Bytes den Filter (Kelvin-Zeichen).
  Der Spike prüfte nur `-i` und verlor so Treffer bei `(?i)` im Muster.
- **Ein Regex-Lauf pro Datei (S12):** Kann die Regex keinen Zeilenumbruch
  treffen und enthält sie kein `\A`/`\z`, läuft sie im `(?m)`-Modus einmal
  über die Datei, und die Treffer werden auf Zeilen abgebildet. Dateien mit
  `\r` nehmen den Weg Zeile für Zeile. Den getrennten Ganzdatei-Vorfilter aus
  S5 gibt es nicht: Er verlor Treffer bei `\A`/`\z` (Fehler im Spike), und
  der Ein-Lauf-Weg leistet dasselbe.
- **Schnittstelle:** `grep.Search` bekommt Spans und In-Degree über zwei
  Funktionen in `Options` (`Spans`, `InDegree`) statt über Graph und
  Blast-Index, damit die Beiakte sie liefern kann. Der Weg über den Graphen
  bleibt als Adapter.

### 6. `--trust-index` (S14)

- Neuer Schalter für `graph grep`. Er schließt keinen anderen aus; zusammen
  mit `--max-filesize` gilt er nur, wenn der Wert zum Index passt.
- Ablauf: Prüfen, dass `wiring.json` existiert (Stat), dann den Index per mmap
  öffnen. Die Dateiliste kommt aus der Dateitabelle des Index, die Zähler aus
  seinem Kopf, die Symbole aus der Spans-Beiakte. Kein Walk, kein Record, keine
  Ignore-Regeln, kein Neubau.
- **Rückfall auf den Prüfmodus mit Notiz** (`index not trusted: <Grund>`):
  Index fehlt, ist veraltet (Format, Extraktorversion, Einstellungen) oder
  kaputt, oder die Beiakte passt nicht zum Index.
- Ausgabe: `index` meldet `trusted`. Neues Feld `index_built` mit der mtime
  von `trigram.idx` (RFC 3339), damit ein Agent sieht, wie alt die Antwort
  sein kann. *Meine Entscheidung:* mtime statt eines Zeitstempels im Kopf,
  weil sie nichts kostet; das Feld ist `omitempty` und fehlt im Prüfmodus,
  damit das JSON dort unverändert bleibt.
- Auf einem Baum, der zum Index passt, gleicht das JSON dem des Prüfmodus bis
  auf `index` und `index_built`. Die Dateitabelle des Index enthält keine
  Binärdateien; `files_in_scope` wird deshalb aus Tabelle plus
  `skipped_binary` gebildet. Auf einem geänderten Baum liest der Aufruf die
  Kandidaten des alten Index mit ihrem aktuellen Inhalt; ein Test hält genau
  dieses Verhalten fest.
- Kanal und Privacy bleiben unverändert (`--channel cloud` filtert wie
  heute).

### 7. Wächter und Neubau im Hintergrund (`serve`)

`loomux serve` läuft schon als Hintergrunddienst (gestartet von der
stdio-Bridge, mit Lebenszyklus, Token und Kanälen). Er bekommt einen Wächter,
der den Graphen unabhängig von Aufrufen frisch hält. Der Einmalaufruf bleibt
vollständig und korrekt auch ohne Dienst; der Dienst sorgt nur dafür, dass
er fast immer einen frischen Index vorfindet.

- **Was beobachtet wird:** jeder Bereich aus `config.ReadRegistry` (nicht
  `VisibleAreas`, das ist ein Kanal-Filter), der einen gebauten Graphen hat
  (`wiring.json` vorhanden). Das Register wird jede Minute neu gelesen;
  kommt ein Bereich hinzu oder verschwindet einer, wird die Beobachtung
  angepasst.
- **Auslöser:** *Meine Entscheidung:* jede Änderung unter der Wurzel, außer
  in Pfaden mit einem Verzeichnissegment, das mit `.` beginnt (wie der Walk;
  das schließt `.git/`, `.loomux/`, `.claude/` ein), und außer den
  git-Dateien `index`, `config`, `info/exclude` aus `rec.GitPaths`, die
  zählen, wo immer sie liegen (in einem Worktree liegen sie im Haupt-Repo).
  Punkt-*Dateien* wie `.gitignore` zählen. Liegen die git-Dateien außerhalb
  der Wurzel, beobachtet der Wächter deren Verzeichnisse zusätzlich.
- **Versionsschutz:** Hat der Fingerprint einen anderen Extraktor als der
  laufende Dienst, baut der Wächter nicht und schreibt eine Log-Zeile. Sonst
  bauten Dienst und CLI verschiedener Versionen sich gegenseitig neu.
- **Ruhepause:** Geprüft wird ~2 s nach der letzten Änderung, nicht pro
  Ereignis. Kommen während eines Neubaus Änderungen, folgt danach genau ein
  weiterer Lauf. Die Pause ist eine Konstante.
- **Erst prüfen, dann bauen:** *Meine Entscheidung:* Nach der Ruhepause
  läuft dieselbe Driftprüfung wie im Einmalaufruf (Walk, beide Driften, ~50–
  80 ms). Nur bei Drift wird neu gebaut. Sonst löste jedes `go build` in ein
  ignoriertes `bin/` einen Neubau von ~4 s aus.
- **Neubau:** derselbe Weg wie `graph build` (`BuildWith`, mit der
  `MaxFilesize` aus dem Record, sonst der Vorgabe), unter der vorhandenen
  Neubau-Sperre (`ask.Refresh` mit `Force`). Ist die Sperre von einem
  Einmalaufruf belegt, beginnt die Ruhepause von vorn. Ein Einmalaufruf, der
  während des Neubaus kommt, nimmt das Overlay und wartet nicht.
- **Ende des Dienstes:** `Run` beendet den Wächter und wartet auf einen
  laufenden Neubau, bevor es die Dienstsperre freigibt (Muster von Upkeep).
  Alle Beobachtungs-Handles sind dann geschlossen.
- **Log:** `serve` bekommt einen Log-Writer in `Options` (Vorgabe stderr,
  das im Hintergrundbetrieb in `serve.log` landet). Der Wächter ist ein
  Feld in `Options`, das nur `serveForeground` füllt; bestehende
  Dienst-Tests laufen ohne ihn.
- Verschachtelte Bereiche: Ein Ereignis kann Eltern- und Kindbereich
  auslösen. Das ist in Ordnung, weil die Driftprüfung beim unveränderten
  Bereich nichts baut.
- **Überlauf und Fehler:** Meldet der Wächter einen Überlauf (Puffer voll,
  Ereignisse verloren), wird neu gebaut. Scheitert die Beobachtung eines
  Bereichs (etwa am inotify-Limit unter Linux), schreibt der Dienst eine
  Warnung in sein Log und beobachtet diesen Bereich nicht; der Einmalaufruf
  deckt ihn wie ohne Dienst ab.
- **Mechanik:** *Meine Entscheidung:* unter Windows ein rekursives
  `ReadDirectoryChangesW` je Wurzel über `golang.org/x/sys/windows` (schon
  Abhängigkeit, v0.49.0). Sonst `github.com/fsnotify/fsnotify` v1.10.1 mit
  einem Watch je Verzeichnis (geprüft 2026-11-08: Release 2026-05-04, nicht
  archiviert, aktiv gepflegt; kann nicht rekursiv beobachten, daher nicht
  für Windows). Neu angelegte Verzeichnisse werden nachgetragen.
- **Ohne laufenden Dienst** (CI, frischer Rechner) gilt alles wie ohne
  Wächter: Overlay bis 200 Dateien, darüber Neubau im Aufruf.
- Kein neues Tool in diesem Schritt; `graph_grep` im Dienst ist ein
  Nicht-Ziel (siehe dort).

### 8. Plugin (`dsh-loomux`)

- Config-Option `trustIndex` (Standard `false`) neben `maxResults` und
  `binary`. Ist sie `true`, hängt `buildArgs` `--trust-index` an. Das Feld ist
  keine Eingabe des Tools: Der Agent kann es nicht pro Aufruf setzen.
- `result.js`: `describeIndex` bekommt eigene Sätze für `trusted` (mit
  `index_built`) und `overlay`; heute fielen beide unter „kein Index“.
- Die Plugin-Dateien tragen unveröffentlichte Arbeit des Menschen. Der Plan
  setzt voraus, dass sie vorher committet ist.
- README des Plugins: wann die Option sinnvoll ist (CI, frischer Checkout,
  Baum wird nach dem Build nicht verändert).

## Tests und Abnahme

- TDD, 100 % Coverage wie im Projekt üblich.
- Gleichheitstests aus dem Spike werden echte Tests: paralleler Walk = serieller
  Walk (`reflect.DeepEqual`), `sourceset.FromListing` = `sourceset.Stat`, JSON mit
  Beiakte = JSON ohne, Overlay-Treffer = Treffer nach Neubau.
- Bestehende Tests, die bei kleiner Drift einen Neubau durch `grep` verlangen
  (im Spike: `TestRebuildRefreshesIndex`,
  `TestGrepReportsAWalkFailureDuringRebuild`), werden an das Overlay
  angepasst. Tests für `skeleton` und `audit` bleiben unverändert grün.
- Rückfälle je Beiakte und Index: fehlt, falsche Version, falscher Digest,
  abgeschnitten → bisheriger Weg, nie leeres Ergebnis.
- `--trust-index` auf geändertem Baum: liefert den Stand des Index (Test
  dokumentiert das bewusst) und meldet `trusted`.
- Wächter: Änderung → genau ein Neubau nach der Ruhepause; Änderungsschub
  während des Neubaus → genau ein weiterer; Überlauf → Neubau; Ereignisse
  in `.loomux/state/` und `.git/` (außer den drei Dateien) → kein Neubau;
  gescheiterte Beobachtung → Warnung, Dienst läuft weiter. Die Zeitquelle
  ist austauschbar, damit die Tests nicht schlafen.
- **Benchmark-Abnahme:** *Meine Entscheidung:* ein manuelles Gate vor dem
  Merge, kein CI-Gate. Die Skripte aus dem Research-Ordner werden ohne
  Spike-Schalter neu geschrieben: altes gegen neues Binary, tgrep-Pfad als
  Argument, Szenarien sauber, nach Änderung an 20 Dateien, nach einem Commit,
  mit und ohne `--trust-index`. Toleranz 20 % auf die Zahlen aus „Ziele“
  (Windows streut stark). Bei Linux gilt „auf Niveau“ als höchstens 20 %
  langsamer als `grep -rn` mit Ausschlüssen. Ein Verfehlen blockiert den
  Merge und kommt mit Zahlen zum Menschen. Ergebnis als datierter Eintrag in
  `docs/en/benchmarks.md` und `docs/de/benchmarks.md`.

## Doku

- `docs/en|de/cli-reference.md`: `--trust-index`, Feld `index_built`, Wert
  `trusted` und `overlay` für `index`, der Text zu `--no-refresh` (ein
  geänderter Baum heißt dann nicht mehr pauschal `stale`), die Beiakten bei
  `graph build`, der Wächter bei `serve`, `graph_find_all`. Im deutschen
  grep-Abschnitt steht die Exit-Code-Zeile doppelt; die veraltete fällt weg.
- `docs/en|de/configuration.md`: Dateitabelle um `grepspans.idx` und die
  binären Records ergänzen.
- `docs/en|de/benchmarks.md`: neuer datierter Eintrag.
- Wikiseiten zu `graph grep` nach `AGENTS.md` im selben Task.
- **Korrektur (Entscheidung des Menschen, 2026-11-08):** Befund 2 des
  Benchmark-Eintrags „2026-11-06 11:35 — The Gate Against tgrep (Part C)“ in
  `docs/en/benchmarks.md` und `docs/de/benchmarks.md` sowie die Roadmap-Zeile
  „Fusion of tgrep and the code graph“ in `README.md` und `README.de.md`
  werden korrigiert. Die Ursache war nicht das Laden von `wiring.json` und
  Index, sondern zwei serielle Walks (sauberer Baum) und der Neubau (nach
  einer Änderung). Die Korrektur verweist auf das Research-Dokument und
  ersetzt den Text nicht stumm: Im Benchmark-Eintrag bleibt die alte Aussage
  durchgestrichen oder als „korrigiert am …“ markiert stehen.
  Die Roadmap-Zeile bekommt nach dem Bau den neuen Stand.

## Dem Bauenden überlassen

- Paketgrenzen und Namen, solange die Spike-Schalter (`LMX_SPIKE_*`) ganz
  verschwinden.
- Wie der Walk-Merkzettel durchgereicht wird.
- Ob die binäre Record-Fassung gob oder ein eigenes Format ist.
- Reihenfolge der Umsetzung; Vorschlag: Walk → Beiakten/Index → Scan →
  Overlay → `--trust-index` → Wächter → Plugin.
