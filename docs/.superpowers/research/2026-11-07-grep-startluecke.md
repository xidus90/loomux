# Startlücke von `graph grep`: Spikes und Messungen

Stand 2026-11-07. Wegwerf-Spikes auf dem Branch `spike/grep-startup`
(Worktree `.claude/worktrees/spike-grep-startup`, nicht für den Merge). S1–S5
stehen im lokalen Commit `35da79d3`, S7–S12 nur im Index des Worktrees (Diff in `spike-s6-s12.diff`),
weil der Pre-Commit-Hook den Spike-Stand ablehnt. Skripte und Rohdaten liegen
in [`2026-11-07-grep-startluecke/`](2026-11-07-grep-startluecke/).

**Kurz:** Ohne Daemon kommt `graph grep` unter Linux auf das Niveau von
`grep -rn` mit Ausschlüssen und unter Windows auf das Doppelte bis Dreifache
von tgrep, bei sonst gleichen Treffern. Der Abschnitt „Zweite Runde“ am Ende
löst einige Befunde der ersten Runde ab.

## Ziel

`graph grep` soll tgrep und `dsh-tgrep` ersetzen können. Es muss tgrep nicht
schlagen, aber ungefähr auf sein Niveau kommen und in jedem Fall schneller
sein als die Werkzeuge, die ein Agent sonst nimmt: `git grep` und `findstr`
unter Windows, `grep -rn` unter Linux. Das gilt im Agentenalltag, also auch
für den ersten Aufruf nach einer Änderung am Arbeitsbaum, nicht nur direkt
nach `graph build`.

## Ausgangslage: wohin die Zeit ging

Der Benchmark-Eintrag vom 2026-11-06 (`docs/en/benchmarks.md`, „The Gate
Against tgrep“) schrieb die Lücke dem Laden von `wiring.json` und dem
Trigramm-Index zu. Die Profile zeigen ein anderes Bild. Gemessen wurde der
Aufruf genau so, wie `dsh-loomux` ihn absetzt
(`graph grep -- <muster> --json --channel cloud --max-hits 300`, ohne
`--no-refresh`):

1. **Zwei vollständige Walks pro Aufruf.** `loadGraph` → `ask.EnsureFresh` →
   `freshness.Probe` → `sourceset.Stat` → `corpus.Walk(root, nil)` für die
   Frische des Graphen, danach `corpus.Walk(root, matcher)` für die Drift des
   Suchkorpus. Jeder Walk kostet ~150–185 ms, und das sind fast nur
   Windows-Syscalls: Verzeichnisse öffnen, lesen und schließen, 3266
   Verzeichnisse nacheinander. Die Ignore-Regeln kosten praktisch nichts
   (150 ms mit Matcher, 146 ms ohne).
2. **`wiring.json` (16 MB) dekodieren: ~60–70 ms**, nur damit jeder Treffer
   sein umschließendes Symbol und dessen `in_degree` bekommt.
3. **Ein geänderter Baum löst einen vollen Neubau aus.** Schon eine geänderte
   Datei führt ohne `--no-refresh` zu `graph build` (3,5–4,5 s), mit
   `--no-refresh` zu einem Vollscan ohne Index (600–900 ms). Agenten arbeiten
   fast immer auf einem geänderten Baum, und der Benchmark vom 2026-11-06 hat
   nur den sauberen gemessen.
4. **Der Scan ist seriell.** `grep.Search` liest Datei für Datei und matcht
   Zeile für Zeile. Das spielt bei Mustern mit vielen Kandidaten
   (Alternation) und ohne Literal (Vollscan) die Hauptrolle.

Trigramm-Index lesen (~10 ms), Record lesen und vergleichen (~10–20 ms) sowie
`blast.New` mit `FileSpans` (~10 ms) fallen daneben kaum ins Gewicht.

## Die Spikes

Jeder Spike hängt an einem Umgebungsschalter `LMX_SPIKE_<NAME>=1`, damit ein
einziges Binary alle Varianten misst.

| Spike | Schalter | Was er tut | Beleg für gleiches Ergebnis |
|---|---|---|---|
| S1 paralleler Walk | `PARWALK` | `corpus.Walk` liest Verzeichnisse parallel (Semaphore 4×CPU), die `.gitignore` eines Verzeichnisses wird vor dessen Einträgen geladen, der Matcher hinter einem Mutex | Liste mit und ohne Matcher per `reflect.DeepEqual` gleich der seriellen (7706 Einträge); 150–165 ms → 27–35 ms |
| S2 ein Walk pro Prozess | `ONEWALK` | Der Walk ohne Matcher (Frischeprobe des Graphen) wird pro Prozess gemerkt, die Korpusliste mit Matcher daraus abgeleitet: Ignore-Dateien flach zuerst laden, Prune-Regel je Prune-Verzeichnis, dann die Flags | Abgeleitete Liste per `DeepEqual` gleich dem eigenen Walk; Ableitung 7,7 ms statt eines zweiten Walks |
| S3 Spans-Beiakte | `SPANS` | `graph build` schreibt `cache/grepspans.gob`: je Datei die Symbol-Spans samt Knoten (ohne `body_text`) und `in_degree`. `grep` liest diese statt `wiring.json` | JSON von Basis und S1–S5 auf dem sauberen Baum für alle vier Muster gleich, mit `--max-hits 20000`: Gruppen in Reihenfolge, `path`, `symbol.id`, `in_degree` und Trefferzeilen (9, 364, 12 und 3560 Gruppen) |
| S4 Drift-Overlay | `OVERLAY` | Ist die Drift nur Dateiinhalt (geändert, neu, gelöscht) und höchstens 200 Dateien, wird weder der Graph noch der Korpus neu gebaut. Der Index liefert die Kandidaten der unveränderten Dateien, geänderte und neue Dateien werden immer gelesen, gelöschte fehlen in der aktuellen Liste. `index` meldet `overlay` | Treffer auf dem geänderten Baum gleich `rg` und gleich der Basis nach ihrem Neubau |
| S5 paralleler Scan | `SCANPAR` | `grep.Search` liest und matcht die Kandidaten auf allen Kernen. Vor dem Zerlegen in Zeilen läuft ein Vorfilter `(?m)<muster>` über die ganze Datei; Dateien mit `\r` überspringen den Vorfilter, weil `$` sonst vor `\r\n` nicht träfe | Treffer in allen Läufen gleich; `go test ./internal/code/...` grün mit S1, S2, S3 und S5 |

Mit `OVERLAY` scheitern fünf bestehende Tests (`TestAuditNeverRebuildsTheGraph`,
`TestRebuildRefreshesIndex`, `TestLockedRebuildAnswersFromDisk`,
`TestGrepReportsAWalkFailureDuringRebuild`, `TestSkeletonRefreshesOnDrift`).
Das ist erwartet, denn sie verlangen den Neubau. Es zeigt aber auch einen
Fehler des Spikes: Der Schalter sitzt in `loadGraph` und wirkt damit auch auf
`skeleton` und `audit`. Im echten Bau gehört er nur zu `grep`.

## Methode

- **Maschine:** Windows 11 Pro, AMD Ryzen 7 9800X3D (16 Threads), Go 1.27.3.
  Linux: WSL2 Ubuntu (Kernel 6.6.114.1), GNU grep 3.12, ext4.
- **Korpus:** der Spike-Worktree, 8450 versionierte Dateien mit 17,2 MB; im
  Suchkorpus von loomux ~7700 Dateien. Graph: `graph build --no-reuse`
  3,5 s, `tgrep index` 0,5 s.
- **Aufrufe:**
  - loomux wie das Plugin: `graph grep -- P --json --channel cloud [-i] --max-hits 300`. Gezählt wird `total_hits + truncated_hits`.
  - tgrep 1.2.3: `tgrep search --no-unicode --index-path .cache/tgrep-index [-i] P .`
  - `rg -n [-i] P` (ripgrep 15.2.0)
  - `git grep -n -E [-i] P`
  - `findstr /s /n … *`: kennt keine Alternation mit `|` und kein `{3}`/`\w`; deshalb `/r "Search Grep"` (Leerzeichen heißt ODER) und `[A-Z][A-Z][A-Z][a-zA-Z0-9_]`
  - `grep -rnE [-i] P .` aus Git für Windows (MSYS)
- **Sauberer Baum:** ein Aufwärmlauf, dann der Median aus sieben Läufen.
  Vor jeder loomux-Variante wird der Graph-Zustand aus einer Kopie
  zurückgespielt, die direkt nach dem Bau entstanden ist.
- **Geänderter Baum:** An 20 Go-Dateien unter `internal/brain/` wird je eine
  Zeile `// spike drift: func Search marker` angehängt. Gemessen wird der
  **erste Aufruf nach der Änderung**: Vor jedem der fünf Läufe wird der
  Zustand von vor der Änderung zurückgespielt, gemeldet wird der Median. Die
  zustandslosen Werkzeuge laufen wie beim sauberen Baum.
- **Linux:** derselbe Baum als `git ls-files`-Tar in `~/bench-grep-tree`
  (ext4), dort per `git init` und Commit als Repository angelegt. loomux als
  `GOOS=linux`-Build mit allen fünf Schaltern. Ein Aufwärmlauf, Median aus
  sieben. `grep -rn` durchsucht dort auch `.loomux/state` (16 MB
  `wiring.json`, 13 MB Index), wie es ein Agent ohne Ausschlüsse täte.
- Alle Zeiten sind Wanduhrzeit einschließlich Prozessstart, in ms. Der
  OS-Dateicache wurde nicht geleert.

## Ergebnisse Windows

Median in ms. Basis ist `HEAD` (`bab2f74e`), S1 bis S5 kumulativ.

### Sauberer Baum

| Muster | Basis | S1 | S1+S2 | S1–S3 | S1–S4 | S1–S5 | tgrep | rg | git grep | findstr /s | grep -rn (MSYS) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Literal `func Search` | 442 | 213 | 166 | 114 | 115 | **113** | 25 | 180 | 174 | 1150 | 4807 |
| Alternation `Search\|Grep` | 596 | 374 | 338 | 279 | 277 | **173** | 69 | 219 | 170 | 1345 | 5049 |
| `-i func search` | 503 | 235 | 198 | 138 | 144 | **111** | 96 | 174 | 180 | 1245 | 5004 |
| ohne Literal `[A-Z]{3}\w+` | 1142 | 874 | 838 | 717 | 776 | **247** | 5351 | 185 | 180 | 1766 | 4956 |

### Erster Aufruf nach einer Änderung an 20 Dateien

| Muster | Basis | S1 | S1+S2 | S1–S3 | S1–S4 | S1–S5 | tgrep | rg | git grep | findstr /s | grep -rn (MSYS) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Literal `func Search` | 4168 | 3461 | 2892 | 2914 | 135 | **129** | 23 | 179 | 167 | 1312 | 4888 |
| Alternation `Search\|Grep` | 4102 | 3975 | 3689 | 3605 | 317 | **232** | 76 | 217 | 175 | 1353 | 4841 |
| `-i func search` | 4461 | 3695 | 3064 | 2831 | 155 | **129** | 97 | 188 | 184 | 1142 | 4844 |
| ohne Literal `[A-Z]{3}\w+` | 4242 | 3942 | 3785 | 3734 | 899 | **371** | 5415 | 177 | 177 | 1559 | 4855 |

### Treffer

| Muster | loomux sauber | loomux geändert | tgrep geändert | rg geändert | git grep geändert |
|---|---:|---:|---:|---:|---:|
| Literal | 31 | 51 | **31** | 51 | 53 |
| Alternation | 824 | 844 | **825** | 844 | 898 |
| `-i` | 40 | 60 | **42** | 60 | 63 |
| ohne Literal | 8969 | 8969 | 8969 | 8979 | 10275 |

Alle loomux-Varianten liefern in jedem Fall dieselbe Trefferzahl wie die
Basis. Auf dem sauberen Baum stimmen loomux, tgrep und `rg` bei den drei
Mustern mit Literal überein. `git grep` sucht nur versionierte Dateien.
`findstr /s` und `grep -rn` durchsuchen alles, auch `.git`, `.loomux/state`
(29 MB), `bin/base.exe` und `bin/spike.exe`. MSYS-`grep` mit
`--exclude-dir` für `.git`, `.loomux`, `bin` und `.cache` braucht auf dem
sauberen Baum trotzdem 1217 / 1325 / 1326 / 1278 ms (Literal, Alternation,
`-i`, ohne Literal). Unter Windows liegt die Last also im Prozess- und
Dateizugriff von MSYS, nicht in den Artefakten. `findstr` kennt keinen
Ausschluss.

## Ergebnisse Linux (WSL2, ext4)

Zweiter Lauf, Median aus sieben, in ms. „grep, Ausschlüsse“ ist
`grep -rnE --exclude-dir=.git --exclude-dir=.loomux`.

| Muster | `grep -rnE .` | grep, Ausschlüsse | `git grep -nE` | loomux S1–S5 | loomux Treffer, `index` |
|---|---:|---:|---:|---:|---|
| Literal | 217.0 | **53.2** | 328.1 | 137.8 | 31, `used` |
| Alternation | 238.6 | **60.2** | 323.3 | 186.8 | 824, `used` |
| `-i` | 220.6 | **56.8** | 333.3 | 150.1 | 40, `used` |
| ohne Literal | 452.7 | **148.6** | 336.4 | 209.8 | 8969, `scan` |

`graph build --no-reuse` dort: 2,8 s. Die Trefferzahlen von loomux sind
`total_hits + truncated_hits` und gleich denen unter Windows.

## Befunde

1. **Unter Windows ist das Ziel ohne Daemon erreichbar.** S1–S5 zusammen
   sind bei allen Mustern und beiden Baumzuständen schneller als `findstr` und
   MSYS-`grep`. Gegen `git grep` und `rg` zugleich gewinnen sie Literal und
   `-i` auf beiden Bäumen (4 von 8 Fällen, 111–129 ms gegen 167–188 ms). Die
   Alternation auf dem sauberen Baum liegt gleichauf mit `git grep` (173
   gegen 170 ms, `rg` 219 ms). Langsamer sind die Alternation auf dem
   geänderten Baum (232 gegen 175 ms) und das Muster ohne Literal (247 bzw.
   371 gegen ~180 ms).
   Gegenüber tgrep liegt `-i` gleichauf (111 gegen 96 ms), Literal und
   Alternation sind 2,5- bis 4,5-mal langsamer, und ohne Literal ist loomux
   21-mal schneller.
2. **Unter Linux ist das Ziel nicht erreicht.** loomux schlägt `git grep`
   und ein nacktes `grep -rn .`, das `.git` und `.loomux/state` mitliest.
   Ein `grep -rn` mit `--exclude-dir=.git --exclude-dir=.loomux` ist auf
   ext4 aber 1,4- bis 3,1-mal schneller (53–149 ms gegen 138–210 ms). Dort
   kosten Walk und Prozessstart kaum etwas, und die Fixkosten von loomux
   (Record, Ignore-Regeln, Index, Beiakte, Frischevergleich) fallen ins
   Gewicht.
3. **tgrep ist auf einem geänderten Baum falsch, und zwar stumm.** Es findet
   die 20 neuen Zeilen nicht (31 statt 51 Treffer beim Literal). Gesucht
   nach dem Text der neuen Zeile, endet es mit Exit 1, ohne Ausgabe auf
   stdout oder stderr. Seine ~25 ms bekommt es dadurch, dass es nicht auf
   Frische prüft. loomux prüft bei jedem Aufruf, und das kostet den Walk.
4. **Den größten Schritt bringt das Overlay** (S4): Der erste Aufruf nach
   einer Änderung fällt von 3,5–4,5 s auf 129–371 ms. Ohne S4 ist jede
   Optimierung im Agentenalltag fast wirkungslos, weil fast jeder Aufruf
   nach einer Änderung kommt.
5. **Der Rest beim Literal (~113 ms) ist nur grob zerlegt.** Der
   Prozessstart ist klein: `loomux version` misst mit `perf_counter` 13 ms
   (Basis wie Spike), `git --version` 16,5 ms. Ein Profil im Prozess ergab
   ~43 ms pro Aufruf ohne Walk. Die Zahl ist zu niedrig, weil der Merkzettel
   von S2 über die Benchmark-Iterationen hinweg lebt und den Walk nur einmal
   zählt. Darin stecken: Trigramm-Index dekodieren ~11 ms, Spans-Beiakte
   (gob) ~9 ms, Record lesen ~6 ms, Ignore-Regeln aus dem Record ~5 ms, GC
   ~7 ms und der parallele Scan. Hinzu kommen der Walk mit Ableitung
   (~35 ms) und der Prozessstart. Gut 20 ms sind nicht zugeordnet.
6. **MSYS-`grep` mit `\w` ist unzuverlässig:** 2 Treffer statt ~9000 beim
   Muster ohne Literal (mit Ausschlüssen 0). Für den Vergleich zählt nur die
   Zeit.

## Was der Spike nicht klärt

- **Symbolzuordnung im Overlay:** Für geänderte Dateien stammen die Spans aus
  dem alten Graphen. Treffer in einer geänderten Datei können deshalb unter
  dem falschen Symbol landen oder ohne Symbol bleiben. Möglich wären: die
  geänderten Dateien beim Aufruf neu extrahieren (wenige Dateien, schnell),
  die Treffer dieser Dateien ohne Symbol melden, oder die Zuordnung als
  unsicher kennzeichnen.
- **Wann das Overlay neu baut:** Die Grenze von 200 Dateien ist gegriffen.
  Offen ist, ob ein Neubau im Hintergrund, bei `graph build` oder ab einer
  Schwelle kommt, und wie `--no-refresh` sich dazu verhält.
- **Gemerkter Walk nach Neubau:** S2 merkt sich den Walk für den ganzen
  Prozess. Ein Neubau im selben Prozess ändert nur `.loomux/state`, und den
  sieht der Walk nicht. Für langlebige Prozesse (`serve`) muss der
  Merkzettel trotzdem pro Aufruf gelten.
- **Format der Beiakte:** gob ist ein Spike-Format. Der echte Bau braucht eine
  versionierte, atomar geschriebene Datei mit Bindung an den Graphen, etwa
  über den Digest, wie beim Trigramm-Index.
- **Reihenfolge der `.gitignore`:** S1 lädt die `.gitignore` eines
  Verzeichnisses vor allen seinen Einträgen. Der serielle Walk lädt sie in
  lexikalischer Reihenfolge, also nach Einträgen, deren Name vor `.gitignore`
  sortiert (etwa `#…` oder `-…`). Auf diesem Baum sind beide Listen gleich;
  S1 entspricht eher git.
- Nicht gemessen: ein großes Repository, ein kalter Dateicache, und
  `iam_backend` als zweiter Korpus.

## Weitere Hebel, nicht gespikt

- **Fixkosten pro Aufruf** messen und senken: die nicht zugeordneten ~20 ms,
  das Lesen von Record und Ignore-Regeln, die Unterschiede unter Linux.
  Darauf zielt die Lücke zu `grep -rn` mit Ausschlüssen.
- **GC für den Einmalaufruf** abschalten oder `GOGC` anheben (~7 ms).
- **Trigramm-Index und Beiakte lazy lesen**, etwa per mmap, nur die
  benötigten Posting-Listen und nur die Spans der Trefferdateien
  (zusammen ~20 ms).
- **Muster ohne Literal:** Der parallele Scan liegt bei 247 ms gegen 180 ms
  von `rg`. Der Rest ist Go-RE2 gegen den Rust-Regex-Motor von `rg`.
- **Daemon:** Ein residenter Prozess mit Watcher (`loomux serve`) würde Walk
  und Fixkosten ganz sparen und käme in tgreps Bereich. Unter Windows ist er
  für das Ziel nicht nötig. Unter Linux wäre er der sichere Weg an `grep -rn`
  mit Ausschlüssen vorbei.

## Widerspruch zu bestehenden Seiten

Diese Messungen widersprechen der Begründung in Befund 2 des Eintrags
„2026-11-06 11:35 — The Gate Against tgrep (Part C)“ in
`docs/en/benchmarks.md` (und seinem deutschen Zwilling) sowie der
Roadmap-Zeile „Fusion of tgrep and the code graph“ in `README.md` und
`README.de.md`. Dort heißt es, die Zeit gehe in das Laden von `wiring.json`
und Index bei jedem Aufruf. Tatsächlich gehen auf dem sauberen Baum die
größten Anteile in zwei serielle Walks, auf dem geänderten in den Neubau.
Die Seiten sind nicht geändert. Wie sie angepasst werden, entscheidet der
Mensch.

## Zweite Runde: Fixkosten (S7–S12)

Auftrag nach der ersten Runde: so weit wie möglich optimieren, auch gegen
`grep -rn` mit Ausschlüssen unter Linux.

### Wohin die Zeit nach S1–S5 ging

Ein Phasen-Trace im Binary (`LMX_SPIKE_TRACE=1`, Windows, Literal, Werte
pro Aufruf) zeigt, dass die eigentliche Suche nur ~2 ms kostet. Der Rest ist
Lesen und Dekodieren von Zustand:

| Phase | vor S7–S10 | nach S7–S10 |
|---|---:|---:|
| `loadGraph`: Walk, Frische-Record lesen und vergleichen | 39–43 | 32–34 |
| Spans-Beiakte | 8 | 4,6 |
| Korpus-Record (`corpus.json`, 2,3 MB JSON, Hashes als Zahlenlisten) | 6–10 | 3–4 |
| Ignore-Regeln aus dem Record, Vergleich | 6–12 | 7–10 |
| Trigramm-Index | 13–18 | 6–8 |
| Suche | 2 | 2 |

Der Walk ist nicht kürzer zu bekommen: Alle ~3300 besuchten Verzeichnisse
enthalten Korpusdateien, davon 2887 unter `testdata`. Mehr Parallelität hilft
ab 16 gleichzeitigen Lesern nicht mehr (4: 49–62 ms, 16: 37–45 ms, 256:
36–45 ms für `loadGraph`).

### Die Spikes

| Spike | Schalter | Was er tut | Ergebnis |
|---|---|---|---|
| S7 Index lazy | `LAZYIDX` | Der Index wird nicht mehr ganz dekodiert (100k Trigramme in eine Map, 3 Mio. Postings kopiert und geprüft). Gelesen werden Kopf und Dateitabelle; ein Trigramm wird binär in der sortierten Tabelle gesucht, nur seine Postings werden gelesen und geprüft | 13–18 → 6–8 ms. Fällt eine Prüfung durch, liefert der Spike `nil`, also „keine Kandidaten“, und das ist stumm falsch (`TestParsePostingNamesNoFile` rot). Im echten Bau muss sie auf einen Scan zurückfallen |
| S8 Records binär | `RECGOB` | `corpus.json` und `fingerprint.json` bekommen einen gob-Spiegel, der neu entsteht, wenn das JSON neuer ist | Korpus-Record 6–10 → 3–4 ms |
| S9 Spans lazy | `LAZYSPANS` | Die Beiakte hält je Datei einen eigenen Block; dekodiert werden nur Dateien mit gezählten Treffern | 8 → 4,6 ms |
| S10 GC aus | `GCOFF` | `debug.SetGCPercent(-1)` für den Einmalaufruf | im Rauschen |
| S11 Literal-Vorfilter | `LITPRE` | Die Pflicht-Literale aus `trigram.Literals` filtern vor der Regex, erst die Datei, dann jede Zeile (`bytes.Contains` auf einer ASCII-kleingeschriebenen Kopie). Unter `-i` überspringt eine Datei mit Nicht-ASCII-Bytes den Filter, weil `(?i)k` auch das Kelvin-Zeichen trifft | Suche bei der Alternation 46–57 → 9–12 ms. Vorher lief Gos NFA ohne gemeinsames Präfix über jede Zeile |
| S12 Ein Regex-Lauf pro Datei | `WHOLE` | Kann die Regex keinen Zeilenumbruch treffen (kein `.` mit `s`, keine Klasse mit `\n`, kein `\A`/`\z`), läuft sie einmal über die Datei, und die Treffer werden auf Zeilen abgebildet. Dateien mit `\r` nehmen den Weg Zeile für Zeile | Muster ohne Literal unter Linux 120 → 112 ms; unter Windows im Rauschen |

Gleichheit: Das JSON von Basis und S1–S12 ist bei allen vier Mustern
identisch (Gruppen in Reihenfolge, `path`, `symbol.id`, `in_degree`,
Trefferzeilen; `--max-hits 20000`). `go test ./internal/code/grep/
./internal/code/query/` ist mit S5, S11 und S12 grün.

Fehler im Spike, beim Messen gefunden: Beim ersten `graph build` in einem
frischen Baum gibt es `cache/` noch nicht, wenn die Spans geschrieben werden.
Der Spike verschluckte den Fehler, und die erste Linux-Messung lief ohne
Beiakte (Spans-Phase 60 ms). Die Zahlen unten stammen aus der behobenen
Fassung.

### Ergebnisse Windows

Windows streute in dieser Runde stark: Dieselbe Variante lag beim Literal
mal bei 71, mal bei 105 ms, und auch `findstr` und die Basis schwankten um
40–80 %. Die Tabellen zeigen die letzte vollständige Reihe; die Spanne aus
allen Reihen steht darunter.

Geänderter Baum, erster Aufruf nach der Änderung (Median aus fünf, ms):

| Muster | Basis | S1–S5 | S1–S10 | S1–S12 | tgrep | rg | git grep |
|---|---:|---:|---:|---:|---:|---:|---:|
| Literal | 3665 | 120 | 108 | 113 | 21 | 177 | 164 |
| Alternation | 3830 | 191 | 166 | 139 | 69 | 173 | 162 |
| `-i` | 3899 | 142 | 136 | 128 | 93 | 186 | 176 |
| ohne Literal | 4506 | 332 | 334 | 311 | 5508 | 185 | 171 |

Sauberer Baum (Median aus sieben, ms):

| Muster | Basis | S1–S10 | S1–S12 | tgrep | rg | git grep |
|---|---:|---:|---:|---:|---:|---:|
| Literal | 496 | 91 | 105 | 27 | 189 | 185 |
| Alternation | 735 | 164 | 134 | 75 | 184 | 182 |
| `-i` | 553 | 90 | 80 | 120 | 186 | 169 |
| ohne Literal | 1033 | 229 | 240 | 5197 | 183 | 176 |

Spanne von S1–S10/S1–S12 über alle Reihen auf sauberem Baum: Literal
71–105 ms, Alternation 133–164 ms, `-i` 80–106 ms, ohne Literal 229–288 ms.

### Ergebnisse Linux (WSL2, ext4)

Median aus sieben, ms. „grep, Ausschlüsse“ wie oben.

| Muster | `grep -rnE .` | grep, Ausschlüsse | `git grep -nE` | loomux S1–S5 | loomux S1–S12 | Treffer, `index` |
|---|---:|---:|---:|---:|---:|---|
| Literal | 216 | 48 | 338 | 138 | **42** | 31, `used` |
| Alternation | 236 | **57** | 352 | 187 | 68 | 825, `used` |
| `-i` | 241 | 60 | 350 | 150 | **50** | 40, `used` |
| ohne Literal | 516 | 163 | 354 | 210 | **112** | 8996, `scan` |

Phasen unter Linux mit S1–S10, Literal: `loadGraph` 8–10 ms, Spans 4–5,
Record 3, Regeln 3–4, Vergleich 3–5, Index 6–9, Suche 1–2; gesamt 39–43 ms.
`loomux version` 5 ms, `/bin/true` 2–3 ms.

### Was ein residenter Prozess brächte

Geschätzt im Prozess (Windows, wiederholte `Grep`-Aufrufe in einem Go-Benchmark,
Zustandsdateien weiter pro Aufruf gelesen):

| | Literal | Alternation |
|---|---:|---:|
| Walk pro Aufruf (Daemon ohne Watcher) | 62 ms | 96 ms |
| Kein Walk (Daemon mit Watcher) | 25 ms | 62 ms |

Ein Daemon hielte die Zustandsdateien zusätzlich im Speicher, der Client
zahlt aber seinen eigenen Start (~13 ms Windows, ~5 ms Linux) plus IPC. Ohne
Watcher brächte ein Daemon also wenig gegenüber One-Shot. Mit Watcher
(`ReadDirectoryChangesW` bzw. inotify) käme er beim Literal in tgreps Bereich
und bliebe dabei frisch. Der Preis: eine neue Abhängigkeit oder eigener
Watcher-Code, ein Lebenszyklus (Start, Ende, mehrere Wurzeln, abgestürzter
Daemon) und ein lokaler Kanal mit eigener Absicherung. Das widerspricht dem
bisherigen Entwurf, der loomux als One-Shot-Binary hält.

### Neue Befunde

1. **Linux: Ziel erreicht.** S1–S12 sind bei Literal, `-i` und ohne Literal
   schneller als `grep -rn` mit Ausschlüssen. Die Alternation liegt knapp
   dahinter (68 gegen 57 ms). Befund 2 der ersten Runde ist damit abgelöst.
2. **Windows: das Doppelte bis Dreifache von tgrep.** Literal ~70–105 gegen
   ~25 ms, Alternation ~135–165 gegen ~70 ms. Bei `-i` gleichauf oder
   schneller, ohne Literal ~20-mal schneller. Gegen `git grep` und `rg`
   gewinnen Literal, Alternation und `-i` auf beiden Bäumen. Das Muster ohne
   Literal bleibt langsamer (230–330 gegen ~180 ms); dort begrenzt das Lesen
   aller ~7700 Dateien.
3. **Die feste Untergrenze unter Windows** sind der Walk (~27 ms, an 16
   Lesern gesättigt) und der Prozessstart (~13 ms). Darunter kommt nur ein
   Prozess, der nicht walkt.

## Dritte Runde: Prüfung abschaltbar (S13–S16)

Vorgabe (2026-11-08): Die Frischeprüfung soll sich ein- und ausschalten
lassen; ausgeschaltet muss loomux so schnell sein wie tgrep. tgrep prüft im
Einzelaufruf nicht (siehe oben) und liest seinen Index per mmap
([microsoft/tgrep](https://github.com/microsoft/tgrep)). Als Schalter dient
im Spike das vorhandene `--no-refresh`, das bisher nur den Neubau
unterdrückt, aber weiter walkt.

| Spike | Schalter | Was er tut |
|---|---|---|
| S13 | `TRUST` | Mit `--no-refresh` kommt die Dateiliste aus dem Record statt aus dem Walk |
| S14 | `TRUSTIX` | Mit `--no-refresh` nennt der Trigramm-Index allein den Korpus; Record, Regeln und Walk entfallen |
| S15 | `MMAP` | Index per mmap statt `os.ReadFile` (nur mit `LAZYIDX`) |
| S16 | `SPANIDX` | Spans je Datei in einer Datei mit sortierter Offsettabelle, per mmap; Öffnen kostet kein Dekodieren |

Phasen eines Literal-Aufrufs ohne Prüfung (Trace, Windows): mit S13
~31 ms im Prozess (Spans 5, Record 4, Regeln 4, Liste 2,5, Index 5–10,
Suche 3), mit S14–S16 ~5 ms (davon Suche ~3).

Windows, Median aus 11 verschränkten Läufen, ms:

| Muster | S1–S12 (Prüfung an) | + S15/S16 (an) | S13–S16 (aus) | tgrep |
|---|---:|---:|---:|---:|
| Literal | 98 | 86 | **23** | 23 |
| Alternation | 107 | 102 | **50** | 66 |
| `-i` | 92 | 81 | **31** | 94 |
| ohne Literal | 266 | 265 | **206** | 5459 |

Treffer: Das JSON mit und ohne Prüfung ist bei allen vier Mustern in den
Treffern gleich. Abweichend sind nur die Zähler `ignored`,
`skipped_binary` und `files_in_scope`, weil sie im Record stehen und S14 ihn
nicht liest; im Design gehören sie in den Indexkopf. Die Trefferzahl von
tgrep weicht um eins ab, weil sein Index älter ist als der Baum: genau der
Frischefehler, den der Schalter in Kauf nimmt.

Befunde:

1. **Ohne Prüfung ist loomux auf tgreps Niveau oder schneller**, bei
   `-i` dreimal, ohne Literal 26-mal so schnell.
2. **Mit Prüfung bleibt die Untergrenze der Walk.** S15/S16 sparen dort nur
   ~5–10 ms.
3. **Ohne Prüfung braucht der Aufruf weder Record noch Regeln**, solange
   der Index die Zähler mitführt.

## Was tgrep anders macht (Quellcode und Messung, 2026-11-08)

Ausgewertet wurden der Quellcode von tgrep 1.2.3, eigene Messungen und eine
Phasenkarte von loomux; jede Übernahmeidee wurde danach gegengeprüft.

Die Lücke bei einem Literal (~68 ms) zerlegt:

| Phase | loomux S1–S12 | tgrep | Differenz |
|---|---:|---:|---:|
| Prozessstart | 15 | 9,5 | +5,5 (40-MB-Image, 163 Paket-Inits) |
| Frischeprüfung (Walk + Fingerprint) | 42 | 0 | +42 |
| Record, Regeln, Liste, zweiter Vergleich, Spans | 18 | 0 | +18 |
| Index öffnen | 7 | ~8 | ±0 |
| Prüfen der Kandidaten | 2 | 2 | ±0 |
| Ausgabe | 4–5 | <1 | +4 |

Weitere Befunde:

- **Die Alternation liest bei loomux viermal so viele Dateien** (488 statt
  109–120), weil der loomux-Index nur kleingeschriebene Trigramme kennt.
  tgrep führt die exakten Bytes und zusätzlich die gefaltete Form.
- **tgrep prüft Kandidaten seriell** und öffnet jede Datei pfadweise mit
  mehreren Handle-Abfragen. Daher braucht es ohne Literal 5–6 s; loomux ist
  dort 20-mal so schnell.
- **Der tgrep-Server hat keine Lese-Schranke.** Eine Änderung ist erst nach
  ~40–65 ms sichtbar, die erste Suche danach verfehlt sie.

Übernahmekandidaten nach der Gegenprüfung (Windows):

| Kandidat | Ersparnis | Bestanden | Anmerkung |
|---|---:|---|---|
| C1 Daemon mit Watcher und Cookie-Schranke | 50–65 ms | ja | einziger Weg zu tgreps Niveau *mit* Prüfung; Ignore-Quellen außerhalb des Baums (`.git/index`, globale Excludes) müssen mitbeobachtet werden |
| C5 Index per mmap, Digest gespeichert statt berechnet | 5–9 ms | ja | mmap-Teil = S15; neu: ~3 ms für den Digest |
| C7 Spans mit Offsettabelle | 3–5 ms | ja | weitgehend = S16 |
| C4 eine Stempeltabelle, ein Vergleich, Ignore-Zustand gespeichert | 4–7 ms | knapp, im Rauschen | |
| C3 exakte Trigramme zusätzlich | 3–7 ms (Alternation) | nein, so nicht | Vorfilter muss Fold-Flags je Literal kennen (`(?i)` im Muster) |
| C6 schlankerer Prozessstart | 1–3 ms | nein, im Rauschen | |
| C8 kompakte, gestreamte JSON-Ausgabe | <1 ms | nein, im Rauschen | |
| C2 NTFS-USN-Journal statt Walk | 0 oder negativ | nein | unprivilegiert ohne Dateinamen, Lesen 25–357 ms |

Nicht übertragbar: Verzeichnis-mtimes (ändern sich nicht bei Inhaltsänderung),
git-Index-Stempel (nur verfolgte Dateien), serielles Prüfen, mmap jeder
Kandidatendatei. Das Vertrauen in den Index (S13/S14) verletzt die
Frischeanforderung und ist nur als ausdrücklich abgeschaltete Prüfung
zulässig, wie in der dritten Runde vorgegeben.

Rohdaten: `workflow-tgrep-vergleich.json` im Artefaktordner.

## Empfehlung

Als nächster Schritt ein Design über S1–S9, S11 und S12 ohne Daemon:
paralleler Walk, ein Walk pro Aufruf, versionierte und an den Graphen
gebundene Beiakten (Spans je Datei, binäre Records), Index lazy mit
Scan-Rückfall bei Fehlern, Drift-Overlay nur für `grep` mit einer Regel für
die Symbolzuordnung geänderter Dateien, paralleler Scan mit
Literal-Vorfilter und Ein-Lauf-Regex. S10 (GC aus) bringt nichts Messbares
und entfällt. Dazu S14–S16 als abschaltbare Prüfung: ohne Prüfung tgreps
Niveau, mit Prüfung korrekt auf geändertem Baum. Der Daemon mit Watcher ist
eine eigene Entscheidung: Er ist der einzige Weg zu tgreps Niveau *mit*
Prüfung unter Windows.
