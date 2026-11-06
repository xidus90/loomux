# Ledger: tgrep-loomux-Fusion

Stand: 2026-11-06, Sitzungen „Umsetzung Teil A", „Teil B" und „Gate Teil C".
Dieses Papier hält den Fortschritt und die Entscheidungen fest, die beim
Ausführen des Plans
(`docs/.superpowers/plans/2026-11-03-tgrep-loomux-fusion-plan.md`) gefallen
sind. Es ergänzt den Plan, ersetzt ihn nicht.

## Erledigt

| Task | Ergebnis | Beleg |
|---|---|---|
| A0 | Baseline: Cold-Build 1,278 s, warme Latenzen, Graphkennzahlen; beide Benchmark-Dateien ergänzt | `docs/en/benchmarks.md`, `docs/de/benchmarks.md` (2026-11-03 23:15) |
| A1 | `trigram.Literals`: Muster → Disjunktion von Konjunktionen; `--fixed`, `-i` (ASCII-Faltung), Fehler wie der Scan | `internal/code/trigram/literals.go`, 100 % Abdeckung |
| A2 | `trigram.Pack`/`Grams`: 3 Byte, little-endian-Wert, ASCII-gefaltet, sortiert/dedupliziert, Nicht-ASCII-Fenster übersprungen | `internal/code/trigram/trigram.go`, 100 % |
| A3 | Indexformat: Kopf/Dateitabelle/Trigrammtabelle/Postings, `Build`/`Write`/`Read`/`decode`, `ErrNoIndex`/`ErrStale`/`ErrCorrupt`, Golden-Datei | `internal/code/trigram/format.go`, `testdata/golden.idx`, 100 % |
| A4 | `Candidates`: Schnitt je Konjunktion, Vereinigung je Alternative, `used`-Flag; Fuzz 60 s / 712 699 Ausführungen ohne Fund | `internal/code/trigram/candidates.go`, 100 % |
| A4a | `corpus.Walk` (ein Walk, `Entry`/`Listing`/`Matcher`), `sourceset.Stat` als Filter darauf, Differentialtest gegen die alte Logik auf Testbaum und Repo | `internal/code/corpus/walk.go`, `internal/code/sourceset/legacy_test.go`, 100 % (Walk mit `coverage:exempt` für drei unerreichbare Arme) |
| A4b | `corpus.Load`/`Rules`: Git-Erkennung, `.gitignore` (nested, lazy), `.git/info/exclude`, globale Excludes (konfiguriert und Git-Standardquelle), Git-Muster (Negation, Anker, `**`, Verzeichnismuster, Escapes, Klassen), Tracked-Ausnahme, `core.ignorecase`, Index-Stempel; Orakel gegen `git check-ignore` und `git ls-files --cached --others --exclude-standard` | `internal/code/corpus/ignore.go`, `ignore_test.go`, `ignore_unit_test.go`; Abdeckung 98 %, die drei Funktionen unter 100 % tragen `coverage:exempt` |
| A4c | Suchkorpus-Auswahl: `DefaultMaxFilesize` (64 MiB), `ResolveMaxFilesize` (Flag > Konfiguration > Standard, 0 = unbegrenzt, negativ = Fehler), `WithinSizeLimit` (inklusive Grenze), `SearchCorpus` (ignoriert, Symlinks, Dot-Namen, zu groß), `IsBinary` (NUL-Regel), Schema-Schlüssel `grep.max_filesize` (Modul `Graph`) | `internal/code/corpus/options.go`, `binary.go`, `internal/config/schema/schema.go`; 100 % |
| A4d | Suchkorpus-Record `cache/corpus.json`: Version, kanonische Einstellungen, Ignore-Quellen mit Text, Index-/Config-Stempel, Tracked-Menge, Git-Pfade, `Print` je Datei; `Digest` (Einstellungen + Quell-Hashes + `(rel, hash)`), atomares `Write`, `Read`; `Probe` ohne Git (Matcher aus dem Record, Stat zuerst, Hash nur bei Abweichung, neue/gelöschte Ignore-Quellen und neue `.gitignore` als Drift) | `internal/code/corpus/record.go`, `probe.go`; Abdeckung 97,6 %, die fünf Funktionen unter 100 % tragen `coverage:exempt` mit Begründung |
| A5 | `graph build` schreibt Suchkorpus-Record und Trigramm-Index; Körper aus dem Extrakt wiederverwendet (`Stats.bodies`), Suchkorpus liest nur, was der Walk nicht schon lieferte; Index-/Record-Fehler sind Notizen und brechen den Build nicht; `BuildOptions.MaxFilesize` | `internal/code/query/corpus.go`, `build.go`, `corpus_test.go`; `query` 98,4 % paketlokal, `go vet` sauber |
| A6 | `graph grep` sucht über den Suchkorpus statt nur über Graphdateien; Kandidaten aus `trigram.Literals`/`Candidates`, sonst Scan; `index` = `used`/`scan`/`stale`/`none`; Zähler `files_in_scope`, `candidates`, `ignored`, `skipped_binary`, `skipped_size`; Binärdateien werden in beiden Wegen übersprungen; Äquivalenztest Index gegen Scan je Muster | `internal/code/query/grep.go`, `grep_equivalence_test.go`, `internal/code/grep/grep.go`; beide Pakete grün |
| A7 | Kanal `cloud` filtert über die Projektdeklaration (`privacy.IsReadable` + `Conceals`), fehlendes Manifest heißt „nichts verborgen“, unlesbares Manifest bricht ab; `--glob` filtert Pfade vor dem Lesen; Präfix unter `never` antwortet leer mit Notiz statt `prefix not indexed` | `internal/code/query/grep_privacy.go`, `grep_privacy_test.go` |
| A8 | CLI-Flags `graph grep --channel/--glob/--max-filesize/--no-max-filesize` und `graph build --max-filesize/--no-max-filesize`; ungültiger Kanal, negativer Wert und beide Größenflags zusammen → Exit 2; JSON trägt die neuen Felder; Regex-Fehler bleibt Exit 1 | `internal/cli/graph.go`, `graph_test.go` |
| A9 | Gültigkeit beim Refresh: Record, Matcher aus dem Record und ein Walk entscheiden ohne git über den Rebuild; ein Flag-Wert schreibt den Index nie um (Notiz „settings differ from index“), `--no-refresh` baut nichts und meldet den Index als `stale`, wenn der Baum sich bewegt hat; Sperre, Ignore-Änderung, `git add -f` und ein fehlender Record (Rebuild, Scan, Sperre) sind getestet | `internal/code/query/grep.go` (`Grep`, `rebuild`), `grep_fresh_test.go` |
| A10 | Doku und Messung: `cli-reference` en/de §6 (Suchkorpus, Trigramm-Index, `--channel`, `--glob`, Größenflags, JSON-Felder), `configuration` en/de §4 (drei Beiakten) und `[grep]` als reserviert, Benchmarks en/de mit kaltem Build, warmen Medianen, Probe und Orakel, READMEs (Capability-Zeile, Roadmap-Zeile „Fusion von tgrep und Code-Graph“), Wiki-Seiten zu Graph und Datenschutz samt Protokoll und `wiki repin`; Performance-Ziele des Entwurfs verfehlt und als Abweichung dokumentiert; PR-Körper mit `parse-body` geprüft | `docs/en|de/{cli-reference,configuration,benchmarks}.md`, `README.md`, `README.de.md`, `docs/wiki/topics/{code-graph,datenschutz-und-kanaele}.md`; Messung 2026-11-04 20:05 |
| B0 | Ist-Zustand des Plugins belegt: `dsh-tgrep` 0.1.13, Config-Schema, `apply`, `presentationMeta`, `presentCall`/`presentResult`, `execute`, `runTgrep`, Patch-Form (`- insert:`) | `C:\Users\micro\.dsh\profiles\desktop\node_modules\dsh-tgrep\index.js`; Notizen im Plan |
| B1–B4 | Neues Paket `dsh-loomux`: Argumente, Ergebnisform, Routing, Fehlerklassen; je eigene Testdatei | `dsh-loomux/src/{args,result,route,errors,run}.js`, `test/*.test.js` |
| B5 | entfällt (kein Rückfall, keine zweite Suche) | Plan, Entscheidung 3/10 der Spec |
| B6 | Privacy-Vertrag als Test: `--channel cloud` immer, keine Eingabe erreicht `--channel` | `dsh-loomux/test/args.test.js` |
| B7 | 59 Tests grün, Coverage 100 % Zeilen/Funktionen (Branches 87,79 %); Lauf gegen die im Profil installierte Kopie mit `LOOMUX_BIN` auf den Repo-Build | `dsh-loomux/test/*`, `.cache/plugin-integration.mjs`, `.cache/plugin-profile-run.mjs` |
| B8 | `package.json` 0.1.0 ohne Laufzeit-Abhängigkeiten; Profil registriert (`installed: true`, `enabled: true`, `rows: [{rowId: "loomux"}]`); `npm publish` **nicht** ausgeführt | `dsh-loomux/package.json`, `dsh-loomux/cordis.patch.yml`, Profil `desktop`; `plugin_manager list_bundles` |
| C1 | tgrep 1.2.3 als Messwerkzeug (WinGet-Paket, nur absoluter Pfad) | Benchmark-Eintrag 2026-11-06 |
| C2 | Methode vor der Messung festgelegt; Dateisatz- und Treffermengen-Gleichheit geprüft | Benchmark-Eintrag 2026-11-06; Plan, Task C2 |
| C3 | Gate **nicht bestanden**: Treffermengen in beiden Korpora identisch, Latenz 3–13× über tgrep bei Literal/Alternation/`-i`; nur das Muster ohne Literal gewinnt loomux | `docs/en/benchmarks.md`, `docs/de/benchmarks.md` (2026-11-06 11:35) |
Kein Commit, kein Push – die Commits macht der Mensch; die Vorschläge stehen
im Plan.

## Entscheidungen beim Ausführen

1. **Nicht-ASCII-Literale fallen immer auf den Scan zurück**, nicht nur bei
   `-i`. Der Index faltet nur ASCII; ein exaktes Nicht-ASCII-Literal wäre als
   Trigramm nicht abfragbar. Die konservative Antwort kostet einen Scan, nie
   einen Treffer. (Spec nennt nur den `-i`-Fall; die Ausweitung ist harmlos.)
2. **`Matcher` hat drei Methoden**: `Ignored`, `Tracked`, `HasTrackedUnder`.
   Der Plan nannte `Tracked` nur am `Entry`, nicht im Interface; ohne die
   Methode kann der Walk das Feld nicht füllen.
3. **`corpus` besitzt `pruneDirs`**, `sourceset.SkipDir` delegiert dorthin.
   Sonst gäbe es zwei Kopien derselben Liste, die auseinanderlaufen können.
4. **Differentialtest liegt in `sourceset`** (nicht `corpus/legacy_test.go`):
   geprüft wird der Vertrag von `sourceset.Stat`; ein Test in `corpus` müsste
   `sourceset` importieren und wäre kein unabhängiger Zeuge mehr.
5. **Golden-Test ohne `testdata/corpus`**: Der Format-Test baut seinen Korpus
   im Speicher, damit die Golden-Bytes deterministisch bleiben und keine
   Fixture-Dateien gepflegt werden müssen. `Write`/`Read` laufen über
   `t.TempDir()`.
6. **`Literals` gibt `ok=false` ohne Fehler zurück**, wenn keine Bedingung
   ableitbar ist; nur Syntaxfehler und RE2-fremde Formen (Lookaround,
   Backreference) sind Fehler. Die Fehlertexte entsprechen `grep.unsupported`,
   dessen Helfer unexportiert bleibt (Kopie statt API-Erweiterung).
7. **Git-Standardquelle der globalen Excludes ist `$XDG_CONFIG_HOME/git/ignore`,
   sonst `$HOME/.config/git/ignore`** – am 2026-11-04 mit `git check-ignore`
   gemessen, nicht angenommen (Frage 21). Eine `~/.gitignore_global` ohne
   gesetztes `core.excludesFile` greift nicht.
8. **`core.ignorecase=true` faltet Muster und Namen**, ebenfalls gegen git
   belegt; `false` bleibt exakt. Ein Verzeichnis, unter dem eine getrackte
   Datei liegt, meldet git als nicht ignoriert; der Walk steigt dort wegen
   `HasTrackedUnder` ohnehin ab.
9. **`git ls-files` ist der einzige Git-Aufruf, dessen Fehler `Load`
   zurückgibt.** Ein kaputter Index würde sonst still jede Datei als
   ungetrackt behandeln und getrackte Dateien fälschlich als ignoriert
   ausblenden. Alle anderen Git-Abfragen sind best effort: fehlt git oder
   antwortet es nicht, gilt keine Ignore-Auswertung (bzw. die Standardquelle).
10. **`Source` trägt `Kind` und `Body`.** Die Probe baut den Matcher aus dem
    Record neu, ohne eine Ignore-Datei zu lesen; ohne den Text ginge das nur
    mit einem Lesevorgang je Abfrage. `Kind` unterscheidet `.gitignore`,
    `.git/info/exclude` und globale Excludes, weil der Pfad einer globalen
    Quelle außerhalb des Repos liegen kann.
11. **Der Digest enthält die Stempel nicht.** Er bindet Einstellungen,
    Quell-Hashes und die `(rel, hash)`-Paare; `.git/index` und `.git/config`
    vergleicht die Probe selbst per Stat. Plan-Test A4d hieß
    „DigestChangesWithTrackingStamp“; er heißt jetzt
    „DigestChangesWithTrackedFile“ und belegt, dass `git add -f` (Datei kommt
    hinzu) und `git rm --cached` (Datei fällt weg) den Digest ändern – das ist
    die Wirkung, die zählt, und der Digest bleibt zeitunabhängig (Spec
    „Digest und Gültigkeit“).
12. **Ein fehlender `.git/index` ist ein gültiger Stempel.** Ein Repository
    ohne `git add` hat keine Indexdatei; der Record schreibt Größe und Zeit 0,
    und die Probe behandelt „Datei fehlt, Stempel 0“ als unverändert. Legt ein
    späteres `git add` den Index an, weicht der Stat ab und die Probe meldet
    Drift.
13. **`maxExtractor` steigt von 64 auf 256 Byte.** Der Index schreibt die
    Extraktorversion in den Kopf; `all.Version()` ist länger als 64 Byte, und
    `trigram.Write` hätte den Build sonst immer mit „extractor longer than
    64 bytes“ abgebrochen. Die Grenze bleibt eine feste Zahl (kein
    unbegrenztes Feld), damit das Format lesbar bleibt; die Tests prüfen
    weiterhin die Grenze selbst (`maxExtractor+1`).
14. **Der Äquivalenztest läuft auf einer Fixture, nicht auf dem loomux-Repo.**
    Der Plan nennt Testkorpus und loomux-Repo; ein Test, der im Repo baut,
    schreibt `.loomux/state` und kollidiert mit gleichzeitigen Läufen und
    fremder Arbeit. Die Fixture (`sample()` + README, ignorierte Logdatei,
    Binärblob, Übergröße, Manifest) deckt dieselben Fälle; die Repo-Skala
    misst Teil C.
15. **Der Corpus-Probe entscheidet auch ohne Rebuild über `stale`.** Ein
    `--no-refresh` baut nichts, prüft aber den Record und meldet den Index als
    `stale`, wenn der Baum sich bewegt hat; sonst wäre eine geänderte
    `.gitignore` bei `--no-refresh` unsichtbar. Das kostet einen Walk, keine
    Hashes (Stat zuerst).
16. **Ein Flag-Wert ist eine Messung, kein Drift.** Weicht `--max-filesize`
    vom Record ab, wird nicht gebaut; die Antwort scannt und notiert
    „settings differ from index“. Ohne Flag baut der Refresh auf den
    aufgelösten Wert (heute die Vorgabe, weil Frage 18 offen ist und kein
    `[grep]`-Abschnitt gelesen wird).
17. **A9 `TestConfigChangeRebuilds` entfällt vorerst.** Der Test setzt einen
    gelesenen Konfigurationswert voraus; Frage 18 ist nicht entschieden, und
    der Plan sagt selbst „nur nach Frage 18“. Sobald der Mensch `[grep]`
    beschließt, kommt der Reader plus dieser Test.
18. **Der saubere Pfad braucht kein git.** `Grep` liest den Record, baut den
    Matcher mit `corpus.RulesFromRecord` und geht einmal den Baum entlang;
    `corpus.Compare` entscheidet über Drift. Erst Drift oder ein fehlender
    Record zahlen `corpus.Load` (git) und den Rebuild. Vorher kostete jede
    Abfrage zwei Walks und einen git-Aufruf; warm fiel sie von rund 1015 ms
    auf 288 ms, ohne die Semantik zu ändern.
19. **Der Record ist Version 2 und trägt die Zähler.** `ignored`,
    `skipped_size`, `skipped_binary` und `max_filesize` stehen im Record, damit
    der saubere Pfad sie ohne erneute Auswahl melden kann; ein Record der
    alten Version ist unbekannt und wird neu gebaut. `Compare` vergleicht die
    Dateimenge am Limit des Records, nicht am Flag-Wert: Ein anderes Limit
    ändert die Antwort, nicht den Baum, und bleibt ein Settings-Drift.
20. **Der Walk lädt jede besuchte `.gitignore`.** Ein getracktes
    `.gitignore` kurzschließt `Ignored` und wurde nie als Quelle vermerkt; die
    Probe meldete es dann bei jeder Abfrage als neu. `Walk` ruft deshalb
    `Rules.LoadIgnore` für jede gesehene `.gitignore`-Datei. Zusätzlich löst
    `gitAbs` relative Git-Pfade gegen die gespeicherte Wurzel auf; ohne das
    verglich die Probe auf Windows `.git\info\exclude` relativ zum
    Arbeitsverzeichnis und meldete dauerhaft Drift.
21. **Die Entwurfsziele sind verfehlt und stehen als Abweichung in den
    Benchmarks**, nicht als Gate: warm 288,6 ms (Ziel < 50 ms), Index 83,9 %
    der Korpus-Bytes (Ziel < 10 %), kalter Build 3,494 s gegen 1,278 s der
    Basis (+173 %, Ziel +20 %). Das Gate ist der Vergleich gegen tgrep in
    Teil C; ein Scheitern dort lässt tgrep stehen, statt die Ziele zu
    verbiegen.
22. **Das Orakel auf dem Repo ist grün.** 7678 besuchte Dateien, 0
    Abweichungen gegen `git check-ignore --stdin`; Korpus 7648 gegen
    `git ls-files` 8426, Unterschied vollständig erklärt (778 Punktverzeichnisse,
    6 C-quotierte Nicht-ASCII-Pfade, dieselben 6 ungequotet auf unserer
    Seite), keine unerklärte Zeile.
23. **Ein Glob ohne Schrägstrich gilt auf jeder Tiefe.** ripgrep liest `*.md`
    gegen den Dateinamen, der Privacy-Übersetzer hätte es an der Wurzel
    verankert; `globFilter` setzt deshalb `**/` davor. Und die Präfixprüfung
    läuft auf dem Korpus **vor** dem Glob (`grep.Options.All`): sonst meldet
    `--glob '*.go' --in internal/code/grep` ein „prefix not indexed“, obwohl
    der Prefix existiert. Beide Fehler standen erst im realen Lauf.
24. **Das Profil überschreibt das Binary.** Das `loomux` im PATH ist 1.17.0
    und kennt `--glob`/`--channel` nicht; der Profil-Patch setzt `binary` auf
    `C:\Users\micro\Documents\#GIT\loomux\bin\loomux.exe`, bis ein Release die
    Flags mitbringt. Das Paket kopiert nach `node_modules/dsh-loomux` und
    `dsh-loomux/` im Profil; `package.json` trägt `file:dsh-loomux`.
25. **Das Gate ist gescheitert, C4 bleibt gesperrt.** Die Treffermengen sind in
    beiden Korpora für alle vier Muster exakt gleich; loomux verliert aber bei
    Literal (279 gegen 21 ms), Alternation (428 gegen 51 ms) und `-i` (307
    gegen 82 ms) und gewinnt nur beim Muster ohne Literal (817 gegen 4622 ms).
    Ursache ist der Prozessstart (16 MB Wiring plus 13 MB Index je Aufruf),
    nicht die Suche. tgrep bleibt installiert, `dsh-tgrep` bleibt aktiv; die
    Entscheidung des Menschen, tgrep erst nach bestandenem Gate zu entfernen,
    wird nicht gebeugt.
26. **`iam_backend` läuft als Kopie.** Schreiben außerhalb des Arbeitsbereichs
    ist gesperrt (`graph build --root …` scheitert mit `Zugriff verweigert`),
    und sein `.loomux` hat keinen Trigramm-Index. Die 517 Dateien aus tgrep's
    Liste lagen als Scratch-Baum im Repo; loomux zählte 515 (zwei `.mo` sind
    binär). Der Baum ist wieder gelöscht. Der Kontrolllauf außerhalb des Repos
    entfällt aus demselben Grund.
27. **Frage 12 ist noch nicht belegt.** Der laufende DSH-Prozess lädt
    `dsh-tgrep`, sein `grep` ist das von tgrep; welches `grep` nach einem
    Neustart mit beiden Paketen antwortet, entscheidet die Ladereihenfolge und
    ist nur dort beobachtbar. Ein Ersatzserver wurde nicht gestartet.
## Offen

- Teil C4 (tgrep entfernen, `dsh-tgrep` abschalten): **gesperrt**, das Gate aus
  C3 ist nicht bestanden. Nötig ist eine Latenzverbesserung (verzögertes Laden
  des Index, kleinere Wiring-Datei oder ein residenter Prozess) oder eine
  andere Entscheidung des Menschen.
- Spec-Frage 12: welches `grep` nach einem Neustart mit `dsh-tgrep` und
  `dsh-loomux` antwortet.
- `npm publish` von `dsh-loomux` (ausdrückliche Freigabe des Menschen nötig).
- A9 `TestConfigChangeRebuilds`, sobald Frage 18 entschieden ist
- Wiki-Seiten nach `wiki repin`: Der Mensch prüft die Pins im Pull-Request-Diff
- Performance: warm 288,6 ms gegen Entwurfsziel < 50 ms, Index 83,9 % gegen
  < 10 %, Build 3,494 s gegen +20 %; das Gate aus Teil C ist entschieden
  (gescheitert), nicht diese Zahlen.

## Umgebung

- `go build ./...` und `go vet ./...` grün (2026-11-06); `go test ./...`
  grün bis auf zwei vorbestehende, nicht von diesem Branch verursachte
  Fehlschläge: `internal/child` (`TestRunSeparatesStreamsAndNormalisesNewlines`,
  bekannt flaky) und `internal/setup/hostfile`
  (`TestTheRepositorysOwnSettingsNeedNoChange`, die fremde
  `.claude/settings.json`-Änderung im Baum). Die berührten Pakete
  (`internal/code/{grep,query,corpus,trigram}`, `internal/cli`) laufen einzeln
  grün.
- `check precommit` ist in dieser Umgebung nicht vollständig grün zu bekommen:
  ohne umgeleitete Go-Caches scheitert `go vet ./...` im Gate an
  `mkdir %LOCALAPPDATA%\Temp\go-build…: Zugriff verweigert`, mit
  `GOCACHE`/`GOTMPDIR` unter `.loomux\state\go` brechen vier Tests, die einen
  Temp-Pfad außerhalb des Repos erwarten (`internal/verify/rules`,
  `internal/worktree/{junction,topo}`). `lint` ist mit umgeleitetem
  `GOCACHE`/`GOTMPDIR` grün. `check gocover` scheitert am `go list` darunter
  (Modul-Cache außerhalb des Repos nicht schreibbar); die Abdeckung wurde
  deshalb mit `go tool cover -func` aus demselben Profil gelesen:
  `grep.Search`, `query.Grep` und die neuen Glob-/Präfix-Funktionen liegen bei
  100 %, die Funktionen darunter sind vorbestehend und tragen
  `coverage:exempt`.
- Zwei Aufrufregeln für `go test` in dieser Umgebung: `-coverpkg` und
  `-coverprofile` **quotieren**. Unquoted `-coverpkg=github.com/xidus90/loomux/...`
  verliert den Anfangsteil, Go meldet „matched no packages“ und schreibt kein
  Profil. Und der Profilpfad braucht Schrägstriche
  (`.cache/cover-all.out`); mit Backslash entsteht keine Datei.
- `gitenv`-Wächter grün: `gitOutput` und die beiden Testhelfer setzen
  `gitenv.Environ()`.
- `lint/wiki-sources` grün nach `wiki repin` (Register-Revisionen in
  `_identities.tsv`).
- `check gocover` braucht umgeleitete Go-Caches
  (`GOCACHE`/`GOTMPDIR`/`TEMP`/`TMP` unter `<repo>\.loomux\state\go`), weil
  `%LOCALAPPDATA%\Temp\go-build*` hier „Zugriff verweigert“ liefert. Für
  `go test` dürfen diese Variablen **nicht** gesetzt sein: `t.TempDir()` landet
  sonst im Repo, und die Tests „outside repository“ scheitern.
- Zwei vorbestehende, nicht von diesem Branch verursachte Gate-Findings:
  `internal/setup/hostfile` scheitert, weil die fremde
  `.claude/settings.json`-Änderung (GOCACHE-Präfix im Stop-Hook) im Baum liegt;
  `lint/go` im Gate scheitert an Zugriffsrechten des Go-Build-Cache
  (`go vet ./...` direkt ist grün).
- DSH-Sandbox: Kindprozesse dürfen außerhalb des Arbeitsbereichs nichts
  anlegen (siehe Entscheidung 26); `node --test` und `bin\loomux.exe` laufen
  innerhalb normal.
- Fremde, uncommittete Arbeit im Baum bleibt unangetastet: `.tgrep/`,
  `.loomux/armed.toml`, `.githooks/post-merge`, `_identities.tsv`,
  Wiki-Dateien, `.claude/settings.json`, `scratch-edit-unused`.
