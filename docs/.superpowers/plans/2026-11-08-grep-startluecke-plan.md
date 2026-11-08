# Startlücke von `graph grep`: Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (empfohlen) oder superpowers:executing-plans, um diesen Plan Aufgabe für Aufgabe umzusetzen. Schritte sind mit `- [ ]` markiert.

**Ziel:** `loomux graph grep` wird als Einmalaufruf schnell genug, um tgrep und `dsh-tgrep` abzulösen: korrekt auf jedem Baumzustand, und mit `--trust-index` so schnell wie tgrep; `loomux serve` hält den Graphen per Wächter frisch.

**Architektur:** Ein paralleler Walk pro Aufruf speist beide Driftprüfungen. Kleine Inhaltsdrift beantwortet ein Overlay statt eines Neubaus. Index und eine neue Spans-Beiakte werden per mmap gelesen, der Scan läuft parallel mit Literal-Vorfilter. `--trust-index` überspringt Walk und Record. Ein Wächter in `serve` prüft nach einer Ruhepause und baut bei Drift im Hintergrund neu.

**Tech Stack:** Go 1.26 (`toolchain go1.27.3`), `golang.org/x/sys` v0.49.0, neu `github.com/fsnotify/fsnotify` v1.10.1 (nur außerhalb Windows), Node (Plugin `dsh-loomux`, `node --test`).

**Spec:** `docs/.superpowers/specs/2026-11-08-grep-startluecke-design.md`. Messungen und Spike: `docs/.superpowers/research/2026-11-07-grep-startluecke.md`, Spike-Diff `docs/.superpowers/research/2026-11-07-grep-startluecke/spike-s6-s16.diff` (Wegwerfcode, nur als Referenz; Schalter `LMX_SPIKE_*` kommen nicht in den Bau).

## Global Constraints

- Coverage 100 % je Funktion; eine Ausnahme nur mit `//coverage:exempt <Grund>` direkt über `func` (`AGENTS.md`). Das Gate misst unter Windows; reine POSIX-Funktionen tragen `//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests`.
- Jede neue Testzeile läuft vor dem Produktivcode rot.
- Kein `init()`. Plattformcode in `foo_windows.go` / `foo_other.go` (`//go:build !windows`).
- `CGO_ENABLED=0`; kein `-race` im Gate. Nebenläufiger Code bekommt deshalb deterministische Tests (Seams, Kanäle), keine Schlaf-Tests.
- Abhängigkeiten exakt gepinnt; neu nur `github.com/fsnotify/fsnotify v1.10.1`.
- Commits: Conventional Commits, englisch, Scope ist ein Code-Bereich (`grep`, `corpus`, `trigram`, `serve`, `cli`, `dsh`, `docs`). Kein Plan-, Task- oder Spec-Name. **Kein `Co-Authored-By` und keine Modellnennung** (globale CLAUDE.md schlägt die Vorgabe des Werkzeugs). Hooks nie überspringen.
- Doku deutsch und englisch parallel; Code, Kommentare, Meldungen englisch.
- Tests: `go test ./<paket>/ -run <Name> -count=1`. Für `go vet` und `loomux check gocover` auf diesem Rechner `GOCACHE`, `GOTMPDIR`, `TEMP`, `TMP` unter `<repo>/.loomux/state/go` setzen, **nicht** für `go test`. Gate vor jedem Commit: `sh ci/gate.sh`. Bekannte Altfehler: `internal/child` (flaky), `internal/setup/hostfile`.
- Konstanten aus der Spec: Overlay-Grenze `200`; Walk-Leser `16`; Ersetzungsbudget `2 * time.Second`; Ruhepause des Wächters `2 * time.Second`; Register neu lesen `time.Minute`.
- `skeleton`, `audit`, `blast` und die übrigen Graph-Befehle behalten `loadGraph`/`EnsureFresh` unverändert.

## Review Focus

1. **Commit nach einer Änderung:** `.git/index` bewegt, verfolgte Menge gleich → `overlay`, kein Neubau. Wer committet und dann sucht, darf keine 4 s warten. Test in Aufgabe 10.
2. **`(?i)` im Muster ohne `-i`, Datei mit dem Kelvin-Zeichen U+212A:** Der Treffer muss bleiben. Test in Aufgabe 8.
3. **`\A`/`\z` und ein leer treffendes Muster auf einer Datei mit abschließendem `\n`:** dieselben Zeilen wie heute. Test in Aufgabe 8.
4. **Index wird ersetzt, während ein anderer Leser ihn abgebildet hat (Windows):** Der Build wartet bis zum Budget und gelingt, sobald der Leser schließt. Test in Aufgabe 6.
5. **`go build` schreibt in ein ignoriertes `bin/`:** Der Wächter prüft, findet keine Drift und baut nicht. Test in Aufgabe 12.

## Entscheidungen des Bauenden

Aus der Kartierung des Codes, in der Spec nachgetragen:

- Ein Walk mit Matcher statt rohem Walk plus Ableitung; die Graph-Dateimenge entsteht per `sourceset.FromListing`.
- `grep` geht nicht mehr über `loadGraph`: erst Record, Walk und beide Driften, dann Overlay, Neubau oder Treffer.
- Upgrade: Der Record bekommt Version 3 (gespeicherter Digest). Alte Records gelten als fehlend und lösen den schon vorhandenen Neubau aus. Eine fehlende Spans-Beiakte allein führt zu `wiring.json`, ohne Neubau.
- `grepspans.idx` wird vor `store.Write` gelöscht und nach `writeCorpus` mit dessen Digest geschrieben.
- Der mmap-Helfer lebt in einem eigenen kleinen Paket `internal/code/mmap`, weil `trigram` und die Spans-Beiakte ihn teilen.

## Voraussetzung (Mensch)

- [ ] Die unveröffentlichte Arbeit in `dsh-loomux/`, `docs/en|de/cli-reference.md`, `AGENTS.md`, `ci/dsh-loomux.sh` ist committet. Aufgaben 13 und 14 bauen darauf auf und dürfen sie nicht zurückdrehen.
- [ ] Neuer Zweig `feat/grep-startup` vom aktuellen Stand, in einem eigenen Worktree (`superpowers:using-git-worktrees`). Der Spike-Worktree `spike/grep-startup` wird nicht gemergt.

---

### Aufgabe 1: Atomares Ersetzen mit Budget

**Files:**
- Modify: `internal/lock/replace.go:9-75`
- Modify: `internal/code/freshness/fingerprint.go:84-98`
- Test: `internal/lock/replace_test.go`, `internal/code/freshness/fingerprint_test.go`

**Interfaces:**
- Produces: `func lock.ReplaceEventually(source, target string, budget time.Duration) error`; `var lock.SidecarBudget = 2 * time.Second` (Seam: Tests setzen es kurz); `var lock.OnRetry func()` (nil in Produktion; wird nach jedem gescheiterten Rename aufgerufen, Seam für Tests anderer Pakete); `var lock.replaceSleep = time.Sleep` (Seam). Bestehende Aufrufer behalten ihr Verhalten mit `budget = replaceTries * replacePause`.

- [ ] **Schritt 1: Failing tests**
  - `TestReplaceEventuallyGivesUpAfterTheBudget`: Ziel ist ein Verzeichnis, `replaceSleep` zählt Aufrufe und addiert Zeit auf einer Fake-Uhr; Budget 100 ms, Pause 20 ms → Fehler, höchstens 6 Versuche.
  - `TestReplaceEventuallySucceedsWhenTheTargetFrees`: erster `os.Rename` scheitert über einen Rename-Seam, zweiter gelingt → `nil`, Zielinhalt = Quelle.
  - `TestWriteIsAtomic` (freshness): Nach `freshness.Write` liegt keine `*.tmp`-Datei im Cache-Verzeichnis; `TestWriteReportsAWriteFailure` bleibt grün.
- [ ] **Schritt 2:** `go test ./internal/lock/ ./internal/code/freshness/ -count=1` → FAIL (Symbol fehlt).
- [ ] **Schritt 3:** `ReplaceEventually` aus `replaceEventually` ableiten (Schleife bis Budget statt fester Versuchszahl), `replaceEventually` ruft sie mit dem alten Budget. `freshness.Write` schreibt `Path(root)+".<pid>.tmp"` und ersetzt per `lock.ReplaceEventually(tmp, Path(root), lock.SidecarBudget)`.
- [ ] **Schritt 4:** Tests grün; `go test ./internal/lock/... ./internal/code/... -count=1` grün.
- [ ] **Schritt 5:** Commit `feat(lock): replace files with a time budget and write the fingerprint atomically`.

### Aufgabe 2: Paralleler Walk

**Files:**
- Modify: `internal/code/corpus/walk.go:120-185`
- Test: `internal/code/corpus/walk_test.go`, neu `internal/code/corpus/walk_parallel_test.go`

**Interfaces:**
- Produces: `corpus.Walk(root string, m Matcher) (Listing, error)` unverändert in Signatur und Ergebnis; intern parallel mit `const walkReaders = 16`. Neuer Seam `var readDir = os.ReadDir` in `corpus`.

- [ ] **Schritt 1: Failing tests**
  - In `walk_parallel_test.go` ein serielles Orakel `walkSerial` (Kopie der heutigen `WalkDir`-Fassung, nur im Test) und `TestWalkEqualsSerialWalk`: Baum mit 3 Ebenen, `.gitignore` in Unterverzeichnissen, ignoriertes `node_modules` ohne und mit verfolgter Datei, Symlink, gesperrtes Verzeichnis (`testlock.LockDir`) → `reflect.DeepEqual(Walk, walkSerial)` mit und ohne Matcher, inklusive `Sources()` des Matchers.
  - `TestWalkReadsDirectoriesConcurrently`: der `readDir`-Seam blockiert beim ersten Unterverzeichnis, bis ein zweites gelesen wird → beide Lesungen laufen gleichzeitig, kein Deadlock in 5 s (Kanal, kein Sleep). Der Matcher ist dabei nicht beteiligt, weil er hinter dem Mutex liegt.
  - `fakeMatcher.asked` bekommt einen Mutex (walk_test.go:21-27).
- [ ] **Schritt 2:** `go test ./internal/code/corpus/ -run 'TestWalk' -count=1` → `TestWalkReadsDirectoriesConcurrently` FAIL.
- [ ] **Schritt 3:** Walk mit Semaphore aus 16 Lesern über `os.ReadDir`, Matcher-Aufrufe hinter einem `sync.Mutex`, Einträge sammeln, am Ende nach `Rel` sortieren. Teilergebnisse eines fehlerhaften `ReadDir` werden behalten wie bei `WalkDir`; eine Wurzel, die Symlink ist, wird nicht betreten. Fehlerarme ohne Testweg bekommen `//coverage:exempt` mit Grund.
- [ ] **Schritt 4:** `go test ./internal/code/... -count=1` grün (u. a. `TestWalkMatchesGitFileList`, `TestIgnoredMatchesGitCheckIgnore`, `sourceset` `TestGraphListUnchanged`).
- [ ] **Schritt 5:** Commit `perf(corpus): read directories in parallel during the walk`.

### Aufgabe 3: Ein Walk für beide Driften

**Files:**
- Modify: `internal/code/sourceset/sourceset.go:78-100`
- Modify: `internal/code/freshness/fingerprint.go:105-150`
- Test: `internal/code/sourceset/sourceset_test.go`, `internal/code/freshness/fingerprint_test.go`

**Interfaces:**
- Produces: `func sourceset.FromListing(l corpus.Listing) []SourceFile` (Filter aus `Stat`); `Stat(root)` = `Walk(root, nil)` + `FromListing`. `func freshness.ProbeFiles(root, extractor string, files []sourceset.SourceFile) (*Drift, error)`; `Probe` = `sourceset.Stat` + `ProbeFiles`.

- [ ] **Schritt 1: Failing tests**
  - `TestFromListingEqualsStat`: Baum mit `.go` in ignoriertem Verzeichnis, `.x.go`, Datei > 1 MB, Datei unter `vendor/` → `FromListing(Walk(root, nil))` = `Stat(root)`.
  - `TestFromListingOfAMatcherWalkEqualsStat`: im git-Repo mit `.gitignore` → `FromListing(Walk(root, RulesFromRecord(rec)))` = `Stat(root)` (ignorierte `.go`-Dateien bleiben enthalten).
  - `TestProbeFilesEqualsProbe`: nach einer Änderung an einer `.go`-Datei liefern beide dieselbe `Drift`.
- [ ] **Schritt 2:** Tests FAIL (Symbole fehlen).
- [ ] **Schritt 3:** Filter und Probe-Rumpf verschieben; keine Verhaltensänderung.
- [ ] **Schritt 4:** `go test ./internal/code/... -count=1` grün.
- [ ] **Schritt 5:** Commit `refactor(corpus): probe the graph from a listing the caller walked`.

### Aufgabe 4: Record v3 mit gespeichertem Digest, verfolgte Menge

**Files:**
- Modify: `internal/code/corpus/record.go:19,58-74,89-120,179-195`, `internal/code/corpus/ignore.go:123-135`
- Modify: `internal/code/query/corpus.go:16-23,80-90`
- Test: `internal/code/corpus/record_test.go`, `internal/code/corpus/ignore_test.go`, `internal/code/query/corpus_test.go`

**Interfaces:**
- Produces: `Record.Digest string` (`json:"digest"`, hex); `corpusRecordVersion = 3`; `NewRecord` füllt `Digest`; `Read` lehnt leeren Digest ab. `func corpus.TrackedSet(root string) ([]string, error)` (ein `git ls-files -z --cached`, sortiert, außerhalb eines Repos `nil, nil`). `CorpusStats.Digest [32]byte`.

- [ ] **Schritt 1: Failing tests**
  - `TestWriteReadRoundTrip` erweitert: `got.Digest == hex(Digest(rec))`.
  - `TestReadRefusesWhatItCannotUse`: neuer Fall leerer Digest; Version 2 wird abgelehnt.
  - `TestTrackedSetMatchesLoad`: `TrackedSet(root)` = `rules.TrackedPaths()` aus `Load`, `slices.Equal`.
  - `TestTrackedSetFailsOnABrokenIndex` nach Vorlage `TestLoadFailsOnABrokenIndex`.
  - `TestBuildIndexDigestMatchesRecord` (query): `stats.Corpus.Digest == ix.Digest()` und `rec.Digest == hex(ix.Digest())`.
- [ ] **Schritt 2:** FAIL.
- [ ] **Schritt 3:** Feld, Versionssprung, `TrackedSet` aus dem `ls-files`-Block von `Load` herauslösen (`Load` ruft ihn), `writeCorpus` berechnet den Digest einmal für Record, `trigram.Write` und `CorpusStats`.
- [ ] **Schritt 4:** `go test ./internal/code/... -count=1` grün.
- [ ] **Schritt 5:** Commit `feat(corpus): store the corpus digest in the record`.

### Aufgabe 5: Binäre Fassung von Corpus-Record und Fingerprint

**Files:**
- Create: `internal/code/corpus/binary_record.go`
- Modify: `internal/code/corpus/record.go` (`Write`, `Read`), `internal/code/freshness/fingerprint.go` (`Write`, `read`)
- Test: `internal/code/corpus/binary_record_test.go`, `internal/code/freshness/fingerprint_test.go`, `internal/code/corpus/probe_test.go`

**Interfaces:**
- Produces: `func corpus.WriteMirror(jsonPath string, v any) error` und `func corpus.ReadMirror(jsonPath string, v any) bool`. Die Fassung liegt in `<json>.bin` (gob), mit Kopf `{Version int; Size int64; MTimeNS int64}` der JSON-Datei; `ReadMirror` liefert `false`, wenn die Datei fehlt, die Version fremd ist oder Größe/mtime der JSON-Datei nicht passen. Gelesen über den Seam `readFile`.

- [ ] **Schritt 1: Failing tests**
  - `TestReadMirrorRoundTrip`; `TestReadMirrorRefusesAChangedJSON` (JSON danach neu geschrieben → `false`); `TestReadMirrorRefusesAnotherVersion`; `TestReadMirrorRefusesGarbage`.
  - `TestReadPrefersTheMirror` (corpus): JSON-Datei per Seam unlesbar gemacht, Mirror gültig → `Read` gelingt; `TestReadIgnoresAStaleMirror`: Mirror passt nicht → JSON wird gelesen.
  - `TestProbeCleanWithoutReads` (probe_test.go:39) erlaubt den Mirror-Pfad als zweite Lesung.
  - Fingerprint: `TestProbeTreatsABrokenRecordAsUnknown` (:238) bleibt grün, obwohl ein gültiger Mirror *eines anderen JSON* daneben liegt.
- [ ] **Schritt 2:** FAIL.
- [ ] **Schritt 3:** Beide `Write` schreiben JSON atomar, danach den Mirror atomar (`lock.ReplaceEventually`, `lock.SidecarBudget`); ein Mirror-Fehler ist eine Notiz, kein Fehler. Beide Leser versuchen zuerst `ReadMirror`.
- [ ] **Schritt 4:** `go test ./internal/code/... -count=1` grün; `TestBuildAnnouncesOnlyARecordItCouldNotWrite` (build_test.go:17) zählt weiter 0 bzw. 1 Notiz.
- [ ] **Schritt 5:** Commit `perf(corpus): read the records from a binary mirror bound to their JSON`.

### Aufgabe 6: Indexformat 2, lazy und per mmap

**Files:**
- Create: `internal/code/mmap/mmap.go`, `mmap_windows.go`, `mmap_other.go`, `mmap_test.go`
- Modify: `internal/code/trigram/format.go`, `candidates.go`
- Modify: `internal/code/query/corpus.go:85` (einziger Aufrufer von `trigram.Write`)
- Test: `internal/code/trigram/format_test.go`, `candidates_test.go`, `fuzz_test.go`, `testdata/golden.idx` (von Hand neu erzeugen), `internal/code/query/corpus_test.go`

**Interfaces:**
- Produces:
  - `func mmap.Open(path string) (*mmap.File, error)`; `(*File).Bytes() []byte`; `(*File).Close() error`. Leere Datei → leeres Slice ohne Abbildung. Windows: `CreateFileMapping`/`MapViewOfFile`/`UnmapViewOfFile` über `golang.org/x/sys/windows`; sonst `unix.Mmap`/`Munmap`.
  - `type trigram.Counters struct{ Ignored, SkippedSize, SkippedBinary int; IgnoreCase bool }`.
  - `trigram.Build(files []File, extractor string, settings []byte, digest [32]byte, c Counters) ([]byte, error)`; `trigram.Write(root string, digest [32]byte, extractor string, settings []byte, c Counters, files []File) error`.
  - `formatVersion = 2`; Kopf hinter `bufferBytes`: `u32 ignored, u32 skippedSize, u32 skippedBinary, u8 ignoreCase`.
  - `(*Index).Counters() Counters`; `(*Index).Close() error`; `(*Index).Postings(gram uint32) (ids []uint32, ok bool)` (`ok=false` = ungültig; fehlendes Trigramm = `nil, true`).
  - `Candidates` liefert bei `ok=false` `ix.Files(), false`.
  - `Write` ersetzt per `lock.ReplaceEventually(tmp, final, lock.SidecarBudget)`.

- [ ] **Schritt 1: Failing tests**
  - mmap: `TestOpenMapsTheFile`, `TestOpenEmptyFile`, `TestOpenMissingFile` (`errors.Is(err, fs.ErrNotExist)`), `TestCloseTwiceIsSafe`.
  - `headerSize()` und alle Tests mit handgerechneten Offsets (`TestParseCountMismatch` :413, `TestParsePostingNamesNoFile` :433) um die Zählerbytes verschieben; `TestParseVersionMismatch` (:151) schreibt Version 1.
  - `TestRoundTrip`, `TestHeaderCarriesSettings`: `ix.Counters() == Counters{3, 4, 5, true}`.
  - Validierung pro Trigramm: `TestParseOffsetOutOfRange`, `TestParseLengthOutOfRange`, `TestParsePostingsNotAscending`, `TestParseGramNotSorted`, `TestParseGramOutOfRange`, `TestParsePostingNamesNoFile` erwarten jetzt `Read` ohne Fehler und `Candidates(...) → used == false`, `files == ix.Files()`.
  - `TestCandidatesFallBackOnACorruptPosting` (neu, candidates_test.go).
  - Review Focus 4, `TestWriteWaitsForAMappedReader` (nur Windows, `format_windows_test.go`): Index öffnen, `Write` in Goroutine starten, `lock.OnRetry` meldet den ersten gescheiterten Rename über einen Kanal, dann `Close` → `Write` liefert `nil`, neuer Inhalt lesbar.
  - `TestWriteErrors/rename error` setzt `lock.SidecarBudget` auf 50 ms.
  - Query: die fünf `trigram.Read`-Aufrufe in corpus_test.go (:29, :43, :63, :100, :173) schließen den Index; `TestBuildIndexFailureIsNote` (:120) setzt das Budget kurz.
- [ ] **Schritt 2:** `go test ./internal/code/mmap/ ./internal/code/trigram/ -count=1` → FAIL.
- [ ] **Schritt 3:** mmap-Paket; `decode` prüft nur Kopf, Dateitabelle und Längen, behält das Slice und die Tabellenoffsets; `Postings` sucht binär in der 16-Byte-Tabelle (Referenz: Spike-Diff Zeilen 787–877) und prüft dort Bereich, Teilbarkeit, Aufsteigen und `id < fileCount`. `golden.idx` neu erzeugen und den Befehl dafür im Test-Kommentar nennen. `writeCorpus` übergibt `trigram.Counters{st.Ignored, st.SkippedSize, st.SkippedBinary, rec.IgnoreCase}`.
- [ ] **Schritt 4:** `FuzzCandidatesSuperset` 30 s: `go test ./internal/code/trigram/ -run '^$' -fuzz FuzzCandidatesSuperset -fuzztime 30s`; danach `go test ./internal/code/... -count=1` grün.
- [ ] **Schritt 5:** Commit `perf(trigram): map the index and look up postings lazily`.

### Aufgabe 7: Spans-Beiakte

**Files:**
- Modify: `internal/code/model/spans.go:28-35`
- Create: `internal/code/spanidx/spanidx.go`, `spanidx_test.go`
- Modify: `internal/code/query/build.go` (`BuildWith`), `internal/code/query/corpus.go`
- Test: `internal/code/model/spans_test.go`, `internal/code/query/corpus_test.go`

**Interfaces:**
- Produces:
  - `model.FileSpans`: `sort.SliceStable`, Reihenfolge `From` aufsteigend, `To` absteigend, dann `Node.ID` aufsteigend.
  - `const spanidx.Name = "grepspans.idx"`.
  - `func spanidx.Write(root string, digest [32]byte, g *model.Graph) error`: Kopf `magic "LGSP"`, `u32 version=1`, 32 B Digest, `u32 n`, `n × u32` Offsets; Einträge `u16 len, path, u32 len, JSON` mit `[]{node, from, to, in_degree}` je Datei in `FileSpans`-Reihenfolge; `in_degree` aus `blast.New(g).InDegree`. Atomar über `lock.ReplaceEventually`.
  - `func spanidx.Open(root string, digest [32]byte) (*Reader, error)`: `ErrNoSpans` bei fehlender Datei, `ErrMismatch` bei fremdem Digest oder Version, `ErrCorrupt` bei Bereichsfehlern (alle Grenzen geprüft).
  - `(*Reader).Spans(path string) []model.SymbolSpan` (dekodiert lazy, merkt sich das Ergebnis; ein kaputter Block → `nil` und `(*Reader).Broken() bool` wird wahr); `(*Reader).InDegree(id model.NodeID) int`; `(*Reader).Close() error`.
  - `BuildWith`: `os.Remove(store.CachePath(root, spanidx.Name))` vor `store.Write`; `spanidx.Write(root, stats.Corpus.Digest, g)` nach `writeCorpus`, nur wenn der Korpus geschrieben wurde; ein Fehler ist eine Notiz.

- [ ] **Schritt 1: Failing tests**
  - `TestFileSpansBreaksTiesByID` (zwei Knoten mit gleichem Span).
  - `TestWriteOpenRoundTrip`: für jede Datei von `testdata/cases/graph/repo` sind `Spans(path)` und `InDegree` gleich `FileSpans(g)` und `blast.New(g).InDegree`.
  - `TestOpenRefusesAnotherDigest`, `TestOpenRefusesTruncatedFile` (jede Kürzung von 0 bis len-1 → Fehler, kein Panic), `TestOpenMissing`.
  - `TestBuildWritesSpansLast` (query): Seam lässt `writeCorpus` scheitern → keine `grepspans.idx`; normaler Build → `spanidx.Open(root, stats.Corpus.Digest)` gelingt.
  - `TestBuildAnnouncesOnlyARecordItCouldNotWrite` bleibt bei 0/1 Notiz.
- [ ] **Schritt 2:** FAIL.
- [ ] **Schritt 3:** Umsetzen (Layout-Referenz: Spike-Diff Zeilen 878–948, dort ohne Kopf und Grenzprüfung).
- [ ] **Schritt 4:** `go test ./internal/code/... -count=1` grün.
- [ ] **Schritt 5:** Commit `perf(grep): write the symbol spans per file into a mapped sidecar`.

### Aufgabe 8: Scan parallel, Vorfilter, ein Regex-Lauf

**Files:**
- Modify: `internal/code/grep/grep.go:33-53,87-260`
- Create: `internal/code/grep/scan.go`, `scan_test.go`
- Test: `internal/code/grep/grep_test.go`

**Interfaces:**
- Consumes: `trigram.Literals` (Aufgabe 6 unverändert).
- Produces:
  - `grep.Options` neu: `Spans func(path string) []model.SymbolSpan`; `InDegree func(model.NodeID) int`. Sind sie gesetzt, nutzt `Search` sie statt `spans`/`x`; `g` darf dann `nil` sein, wenn `Files` gesetzt ist.
  - `func Search(g *model.Graph, x *blast.Index, spans map[string][]model.SymbolSpan, pattern string, opts Options, read FileReader) (Result, error)` behält die Signatur.
  - `Result.IndexBuilt string` (`json:"index_built,omitempty"`). Die Kommentare zu `Options.Index`/`Result.Index` nennen `used, scan, stale, none, overlay, trusted`.

- [ ] **Schritt 1: Failing tests** (alle gegen die serielle Zeilenlogik von heute als Orakel `scanLines` im Test):
  - `TestSearchUsesSpanFunctions`: `g == nil`, `Files` gesetzt, `Spans`/`InDegree` aus einer Map → gleiche Gruppen wie mit Graph.
  - `TestScanEqualsLineScan` (Tabelle): Muster `func Search`, `Search|Grep`, `-i func search`, `[A-Z]{3}\w+`, `^$`, `\Afoo`, `foo\z`, `(?s)a.b`, `[^x]`; Dateien mit `\n`, `\r\n`, ohne Endzeilenumbruch, mit abschließendem `\n`, Binär-NUL → identische `lines`/`texts`.
  - Review Focus 2, `TestPrefilterKeepsInlineFoldWithKelvin`: Muster `(?i)kelvin`, `IgnoreCase=false`, Datei `"Kelvin\n"` → 1 Treffer.
  - Review Focus 3, `TestWholeFileSkipsTextAnchors`: `\Afoo` auf `"bar\nfoo\n"` → Treffer in Zeile 2 wie heute (heutige Logik: `\A` gilt je Zeile).
  - `TestSearchOrderIsStableUnderParallelScan`: 200 Dateien, `MaxHits` 7 → gleiche Gruppen und `TruncatedHits` wie seriell, 20 Läufe.
  - `TestSearchEdgeCases` „Nil graph“ (:303) bleibt: `g == nil` ohne `Files` → leer, kein Fehler.
- [ ] **Schritt 2:** `go test ./internal/code/grep/ -count=1` → FAIL.
- [ ] **Schritt 3:** `scan.go`: Lesen und Matchen je Datei auf `runtime.NumCPU()` Workern in ein Ergebnis-Slice, danach serielle Zusammenführung in Dateireihenfolge (maxHits und Kürzung wie heute). Vorfilter: Literal-Mengen aus `trigram.Literals`; Fold-Erkennung über `syntax.Parse` (irgendein Knoten mit `FoldCase`) statt `opts.IgnoreCase`; bei Fold und Nicht-ASCII-Byte kein Vorfilter. Ein-Lauf-Weg nur, wenn die geparste Regex keinen Zeilenumbruch treffen kann und kein `OpBeginText`/`OpEndText` enthält, und nur für Dateien ohne `\r`. Kein getrennter Ganzdatei-Vorfilter.
- [ ] **Schritt 4:** `go test ./internal/code/... -count=1` grün; Golden `query/golden_test.go` unverändert.
- [ ] **Schritt 5:** Commit `perf(grep): scan candidates in parallel behind a literal prefilter`.

### Aufgabe 9: Eigener Frischeweg für `grep`

**Files:**
- Create: `internal/code/query/grep_fresh.go`
- Modify: `internal/code/query/grep.go:44-260`
- Test: `internal/code/query/grep_paths_test.go`, `grep_fresh_test.go`, `grep_equivalence_test.go`

**Interfaces:**
- Consumes: `corpus.Walk`, `corpus.Compare`, `sourceset.FromListing`, `freshness.ProbeFiles` (Aufgabe 3), `Record.Digest` (Aufgabe 4), `trigram.Read`/`Close` (Aufgabe 6), `spanidx.Open` (Aufgabe 7), `grep.Options.Spans/InDegree` (Aufgabe 8).
- Produces:
  - `type grepState struct{ rec corpus.Record; hasRecord bool; rules *corpus.Rules; listing corpus.Listing; corpusDrift *corpus.Drift; graphDrift *freshness.Drift }`.
  - `func probeGrep(root string, maxFilesize int64) (grepState, error)`: liest den Record, walkt **einmal** (`walkCorpus`, mit Matcher wenn Record da), berechnet beide Driften.
  - `func needsRebuild(s grepState, root string) (reason string)`: leer = nein. Gründe wie heute in `ask.Refresh` (kein Fingerprint, anderer Extraktor, `!lexicon.Usable`, Graph-Drift) plus Corpus-Drift. Dieser Schritt ändert das Verhalten noch nicht (Overlay folgt in Aufgabe 10).
  - `Grep` ruft `loadGraph` nicht mehr; nach einem Neubau liest es den Zustand einmal neu, ohne zweiten Walk, wenn `BuildWith` die Liste weitergibt (`BuildOptions.Listing *corpus.Listing`, optional; nil = heute).
  - Symbole: `spanidx.Open(root, d)` mit `d` = hex-dekodierter `rec.Digest`; bei Fehler `store.Read` + `FileSpans`/`blast.New` wie heute. Index und Reader werden vor der Rückkehr geschlossen.
  - Vergleich mit dem gespeicherten Digest statt `corpus.Digest(rec)` (grep.go:212).
- [ ] **Schritt 1: Failing tests**
  - `TestGrepWalksOncePerCall` (grep_paths_test.go): Zähler am `walkCorpus`-Seam und an einem neuen Seam `statSources` → genau 1 Walk auf sauberem Baum.
  - `TestGrepUsesTheSpansSidecar`: `store.Read`-Seam schlägt fehl → `Grep` gelingt mit Symbolen (Beiakte); `TestGrepFallsBackToTheGraphWithoutSidecar`: Beiakte gelöscht → gleiche JSON-Gruppen.
  - `TestGrepClosesTheIndex` (Windows): nach `Grep` lässt sich `trigram.idx` löschen.
  - `TestGrepWalkFailureIsReported` ersetzt die Zählschwellen in `TestGrepReportsAWalkFailureDuringRebuild` (:77) und `TestGrepRebuildWithoutRecordReportsAWalkFailure` (:90) für einen Walk pro Aufruf.
  - Equivalence: `TestEquivalenceWithScan` und das Golden bleiben grün.
- [ ] **Schritt 2:** FAIL.
- [ ] **Schritt 3:** Umbauen; die Doku-Zeilen von `Grep` (grep.go:44-49) an die Funktion statt an `readRecord` hängen.
- [ ] **Schritt 4:** `go test ./internal/code/... ./internal/cli/... ./internal/serve/... -count=1` grün.
- [ ] **Schritt 5:** Commit `perf(grep): check freshness with one walk and read symbols from the sidecar`.

### Aufgabe 10: Drift-Overlay

**Files:**
- Modify: `internal/code/query/grep_fresh.go`, `grep.go`
- Test: `internal/code/query/grep_fresh_test.go`, `grep_overlay_test.go` (neu)

**Interfaces:**
- Produces: `const overlayLimit = 200`; `func overlayFits(s grepState, root string) bool`. Bedingung: Corpus-Drift ohne `Settings`, `Config`, `Sources`; `Index` nur, wenn `corpus.TrackedSet(root)` gleich `rec.Tracked` (`slices.Equal`); ist sie gleich, schreibt `grep` den neuen Stempel von `.git/index` in den Record (`rec.Stamp`, samt binärer Fassung, atomar; Digest unverändert), auch mit `NoRefresh`, damit der nächste Aufruf den git-Aufruf (~20 ms) nicht wieder zahlt; ein Schreibfehler ist eine Notiz; `len(Changed)+len(Added)+len(Removed) <= 200` für beide Driften; Graph-Drift nicht nil; `lexicon.Usable`. Gilt auch mit `NoRefresh`. Kandidaten = Index ∪ Changed ∪ Added (Corpus); `Spans`/`InDegree` liefern für Changed ∪ Added (Corpus und Graph) `nil`/0. `index = "overlay"`, auch ohne Literal.
- [ ] **Schritt 1: Failing tests**
  - `TestOverlayFindsNewLines`: README und eine `.go`-Datei geändert, eine neue Datei → Treffer gleich `rg`-artigem Orakel (`scanLines` über alle Dateien), `index == "overlay"`, kein Neubau (Seam `BuildWith` zählt 0).
  - `TestOverlayHitsHaveNoSymbolInChangedFiles`.
  - `TestOverlayIgnoresDeletedFiles`.
  - Review Focus 1, `TestOverlayAfterACommit`: Änderung committen → `.git/index` bewegt, Menge gleich → `overlay`; danach steht der neue Stempel im Record, und ein zweiter Aufruf ruft `TrackedSet` nicht mehr (Seam zählt 0).
  - `TestGitAddForceMakesStale` (:163) bleibt: Menge anders → Neubau.
  - `TestOverlayLimit`: 201 geänderte Dateien → Neubau.
  - `TestOverlayUnderNoRefresh`.
  - `TestOverlayWithGlobInAndCloud`: `--glob`, `--in` und Kanal `cloud` filtern geänderte Dateien wie unveränderte.
  - Angepasst: `TestRebuildRefreshesIndex` (:25) → `overlay`; `TestLockedRebuildAnswersFromDisk` (:43) nutzt eine `.gitignore`-Änderung, damit weiter neu gebaut wird; `TestNoFlagNoRefreshKeepsScan` (:114) prüfen und anpassen.
- [ ] **Schritt 2:** FAIL.
- [ ] **Schritt 3:** Umsetzen in `grep_fresh.go`.
- [ ] **Schritt 4:** `go test ./internal/code/... ./internal/cli/... -count=1` grün.
- [ ] **Schritt 5:** Commit `feat(grep): answer small content drift from an overlay instead of a rebuild`.

### Aufgabe 11: `--trust-index`

**Files:**
- Modify: `internal/code/query/grep.go` (`GrepOptions`), `grep_fresh.go`
- Modify: `internal/cli/graph.go:396-470`
- Test: `internal/code/query/grep_trust_test.go` (neu), `internal/cli/graph_test.go`

**Interfaces:**
- Produces: `GrepOptions.TrustIndex bool`; CLI-Flag `--trust-index` („answer from the index without checking the tree“). Ablauf: `store.WiringPath` muss existieren (sonst `ErrNoGraph`), `trigram.Read`, Extraktor = `all.Version()`, mit `--max-filesize` Einstellungen = `CanonicalSettings(flag, ix.Counters().IgnoreCase)`, `spanidx.Open(root, ix.Digest())`. Dateiliste aus `ix.Files()`, Zähler aus `ix.Counters()`, `FilesInScope = len(files) + SkippedBinary`. `index = "trusted"`, `IndexBuilt` = mtime von `trigram.idx` in RFC 3339. Jeder Fehler → Notiz `index not trusted: <reason>` und der normale Weg.
- [ ] **Schritt 1: Failing tests**
  - `TestTrustIndexMatchesCheckedJSON`: sauberer Baum, alle vier Benchmark-Muster → JSON gleich bis auf `index`, `index_built`.
  - `TestTrustIndexDoesNotWalk`: `walkCorpus`- und `readRecord`-Seam zählen 0.
  - `TestTrustIndexAnswersFromTheIndexOnAChangedTree`: neue Datei mit Treffer → nicht gefunden; geänderte Kandidatendatei → aktuelle Zeilen (hält das Verhalten fest).
  - `TestTrustIndexFallsBack`: Index fehlt / Extraktor fremd / Beiakte fremd / `--max-filesize` passt nicht → Notiz mit Präfix `index not trusted:` und Ergebnis des Prüfmodus.
  - `TestTrustIndexWithoutGraph` → `ErrNoGraph`.
  - CLI: `TestGraphGrepTrustIndex` (`index == "trusted"`, `index_built` parsebar als RFC 3339); `TestGraphGrepJSONHasIndexField` (:1348) bleibt ohne `index_built`.
- [ ] **Schritt 2:** FAIL.
- [ ] **Schritt 3:** Umsetzen.
- [ ] **Schritt 4:** `go test ./internal/code/... ./internal/cli/... -count=1` grün.
- [ ] **Schritt 5:** Commit `feat(grep): add --trust-index to answer from the index without a walk`.

### Aufgabe 12: Wächter, Kern

**Files:**
- Create: `internal/serve/watch/watch.go`, `watch_test.go`
- Modify: `internal/code/query/grep_fresh.go` (Export für den Wächter)
- Test: `internal/code/query/refresh_test.go`

**Interfaces:**
- Produces:
  - `func query.RefreshIfDrifted(root string, notice func(string)) (rebuilt bool, err error)`: `probeGrep` mit `rec.MaxFilesize` (ohne Record die Vorgabe), bei `needsRebuild` oder Overlay-Drift `ask.Refresh` mit `Force` und `BuildWith(root, BuildOptions{MaxFilesize: &max})`; `*ask.LockedError` wird durchgereicht.
  - `type watch.Event struct{ Root string; Overflow bool }`.
  - `type watch.Source interface{ Add(root string, extra []string) error; Remove(root string); Events() <-chan Event; Close() error }` (`extra` = `ExtraPaths` des Bereichs: einzelne git-Dateien außerhalb der Wurzel, deren Ereignisse mit `Root` = Wurzel gemeldet werden).
  - `type watch.Watcher struct` mit Feldern `Source Source; Areas func() ([]string, error); Refresh func(root string) (bool, error); After func(time.Duration) <-chan time.Time; Log io.Writer; Relevant func(root, path string) bool`.
  - `func (w *Watcher) Run(ctx context.Context)`: Bereiche beim Start und alle `time.Minute` aus `Areas` abgleichen (Add/Remove); je Wurzel Ruhepause `2 * time.Second` nach dem letzten relevanten Ereignis; danach `Refresh`; Ereignisse während eines Refresh → genau ein weiterer Lauf; `Overflow` → Refresh ohne Relevanzprüfung; `*ask.LockedError` → Ruhepause von vorn; andere Fehler → eine Zeile in `Log`; bei `ctx.Done()` wartet `Run` auf einen laufenden Refresh und schließt die Quelle.
  - `func watch.Relevant(rec corpus.Record, root, path string) bool`: falsch bei einem Verzeichnissegment mit führendem `.` relativ zur Wurzel; wahr für die absoluten Pfade aus `rec.GitPaths` (Index, Config, Exclude, aufgelöst über einen neuen Export `corpus.GitAbs(rec, rel string) string`) und für Punkt-Dateien.
  - `func watch.ExtraPaths(rec corpus.Record) []string`: die git-Dateien, die außerhalb der Wurzel liegen (Worktree, Wurzel im Unterverzeichnis). `Source.Add` beobachtet zusätzlich deren Verzeichnisse und meldet nur Ereignisse für genau diese Dateien, mit der Wurzel des Bereichs.
  - Versionsschutz: Hat der Fingerprint auf der Platte einen anderen Extraktor als `all.Version()` dieses Dienstes, baut der Wächter nicht neu und schreibt einmal je Bereich eine Zeile ins Log (sonst bauten Dienst und CLI verschiedener Versionen sich gegenseitig neu).
  - `func watch.Areas(registryDir string) func() ([]string, error)`: `config.ReadRegistry`, nur Bereiche mit `store.WiringPath`; fehlendes Register → leere Liste (Muster `reconcileRegistered`).
- [ ] **Schritt 1: Failing tests** (Fake-`Source` über Kanäle, Fake-`After` wie `TestKeepUpRunsOncePerIntervalUntilTheEnd`, upkeep_test.go:131):
  - `TestOneRefreshAfterTheQuietPeriod`: drei Ereignisse, ein Tick → genau ein `Refresh`.
  - `TestEventsDuringARefreshCauseOneMore`.
  - `TestOverflowRefreshes`.
  - Review Focus 5, `TestIgnoredBuildOutputDoesNotRebuild`: echtes Repo mit `bin/` in `.gitignore`, Datei in `bin/` schreiben, `RefreshIfDrifted` → `false`, kein Build (Seam zählt 0).
  - `TestRelevantSkipsDotDirectories`: `.git/objects/x`, `.loomux/state/graph/wiring.json`, `.claude/worktrees/a/b.go` → falsch; `.gitignore`, `sub/.gitignore`, `src/a.go` → wahr; `rec.GitPaths.Index` außerhalb der Wurzel → wahr.
  - `TestLockedRefreshRearms`; `TestRefreshErrorIsLogged`.
  - `TestForeignExtractorIsNotRebuilt`: Fingerprint mit Extraktor `other/1` → kein Build, eine Log-Zeile, beim zweiten Ereignis keine weitere.
  - `TestExtraPathsOfAWorktree`: Worktree-Wurzel → `ExtraPaths` nennt Index und Config im Haupt-Repo.
  - `TestAreasFollowTheRegistry`: zweiter Abgleich ohne einen Bereich → `Remove`.
  - `TestRunWaitsForTheRefreshBeforeItReturns`.
  - `TestRefreshIfDriftedKeepsTheMaxFilesize` (query): Record mit `MaxFilesize` 1024 → Build-Seam sieht 1024.
- [ ] **Schritt 2:** FAIL.
- [ ] **Schritt 3:** Umsetzen; nur Kanäle und Seams, kein `time.Sleep`.
- [ ] **Schritt 4:** `go test ./internal/serve/... ./internal/code/... -count=1` grün.
- [ ] **Schritt 5:** Commit `feat(serve): rebuild drifted graphs in the background after a quiet period`.

### Aufgabe 13: Wächter, Plattformquellen und Einbau in `serve`

**Files:**
- Create: `internal/serve/watch/source_windows.go`, `source_other.go`, `source_windows_test.go`
- Modify: `go.mod`, `go.sum` (`github.com/fsnotify/fsnotify v1.10.1`)
- Modify: `internal/serve/serve.go:46-70,117-210`, `internal/cli/serve.go:101-115`
- Test: `internal/serve/serve_internal_test.go`, `internal/cli/serve_test.go`

**Interfaces:**
- Consumes: `watch.Watcher`, `watch.Areas`, `watch.Relevant`, `query.RefreshIfDrifted` (Aufgabe 12).
- Produces:
  - `func watch.NewSource() (Source, error)`. Windows: je Wurzel ein Verzeichnis-Handle mit `FILE_FLAG_BACKUP_SEMANTICS|FILE_FLAG_OVERLAPPED` und `ReadDirectoryChangesW(..., watchSubtree=true, FILE_NOTIFY_CHANGE_FILE_NAME|DIR_NAME|SIZE|LAST_WRITE)`; null gelieferte Bytes oder `ERROR_NOTIFY_ENUM_DIR` → `Event{Overflow: true}`. Sonst: fsnotify mit einem Watch je Verzeichnis (Punkt-Verzeichnisse ausgenommen), neue Verzeichnisse werden nachgetragen; `fsnotify.ErrEventOverflow` → Overflow; ein `Add`-Fehler (etwa inotify-Limit) wird zurückgegeben, der Wächter loggt und überspringt den Bereich. `source_other.go` trägt die POSIX-Ausnahme aus den Global Constraints.
  - `serve.Options.Watch func(ctx context.Context)` (nil = kein Wächter) und `serve.Options.Log io.Writer` (nil = `os.Stderr`). `Run` startet `Watch` wie Upkeep und wartet vor `handle.Release()` auf sein Ende.
  - `serveForeground` füllt `Watch` mit einem `watch.Watcher` über `config.StateDir()` und `Log` mit `stderr`.
- [ ] **Schritt 1: Failing tests**
  - `TestSourceReportsAWrite` (Windows): temporäre Wurzel, Datei schreiben → Ereignis mit dieser Wurzel innerhalb von 5 s (Kanal mit Timeout).
  - `TestSourceCloseReleasesTheDirectory` (Windows): nach `Close` lässt sich die Wurzel löschen.
  - `TestRunWaitsForTheWatcherBeforeItReturns` (serve_internal_test.go, nach `runUntilItReturns` :53): Fake-`Watch` blockiert bis zum Kontextende; `expectTheLockIsFree` erst danach.
  - `TestServeForegroundHandsTheServiceAWatcher` (cli/serve_test.go nach :379): `stubServeRun` sieht `Watch != nil`, `Log == stderr`.
  - Bestehende Dienst-Tests laufen unverändert ohne Wächter.
- [ ] **Schritt 2:** FAIL.
- [ ] **Schritt 3:** Umsetzen; `go get github.com/fsnotify/fsnotify@v1.10.1`, vorher Pflegestand erneut prüfen (Release-Datum, nicht archiviert) und im Commit-Text nicht erwähnen.
- [ ] **Schritt 4:** `go test ./internal/serve/... ./internal/cli/... -count=1` grün; `GOOS=linux go build ./...` und `GOOS=darwin go build ./...` bauen.
- [ ] **Schritt 5:** Commit `feat(serve): watch registered areas and refresh them in the background`.

### Aufgabe 14: Plugin

**Files:**
- Modify: `dsh-loomux/index.js:79-90,167-189`, `dsh-loomux/src/args.js:1-7,50-95`, `dsh-loomux/src/result.js:46-71`, `dsh-loomux/README.md`
- Test: `dsh-loomux/test/args.test.js`, `index.test.js`, `result.test.js`

**Interfaces:**
- Produces: `createGrepTool(config)` mit `config.trustIndex?: boolean` (Standard `false`); `buildArgs(input, { trustIndex })` hängt `--trust-index` vor `--max-hits` an. `describeIndex` mit eigenen Sätzen für `overlay` und `trusted` (letzterer nennt `index_built`).
- [ ] **Schritt 1: Failing tests**
  - `config.trustIndex adds --trust-index` (index.test.js, Vorlage :27); ohne Option kein Flag.
  - `no input can reach --trust-index` (args.test.js, Vorlage :73): `{ trust_index: true, trustIndex: true }` als Tool-Argument → kein Flag.
  - `describeIndex covers every index state` (result.test.js:53) um `overlay` und `trusted` erweitert; `trusted` enthält den Wert von `index_built`.
  - Die `deepEqual`-Asserts der Argumentreihenfolge (index.test.js :33, :57-83) bleiben gültig.
- [ ] **Schritt 2:** `sh ci/dsh-loomux.sh` → FAIL.
- [ ] **Schritt 3:** Umsetzen; Kopfkommentar von `args.js` nennt, dass keine Eingabe `--trust-index` erreicht. README: Abschnitt „Configuration“ um `trustIndex: false` und den Einsatzfall (CI, frischer Checkout, Baum nach dem Build unverändert).
- [ ] **Schritt 4:** `sh ci/dsh-loomux.sh` grün (100 % Zeilen und Funktionen).
- [ ] **Schritt 5:** Commit `feat(dsh): let the plugin trust the index when configured`.

### Aufgabe 15: Benchmark-Abnahme

**Files:**
- Create: `docs/.superpowers/research/2026-11-07-grep-startluecke/acceptance.py`
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`

**Interfaces:**
- Produces: `uv run --script acceptance.py <repo> <old.exe> <new.exe> <tgrep.exe> <out.json>` (PEP 723). Szenarien: sauber; nach Änderung an 20 Dateien; nach Commit (Änderung committet); `--trust-index`. Muster: `func Search`, `Search|Grep`, `-i func search`, `[A-Z]{3}\w+`. Je 1 Aufwärmlauf, Median aus 11, verschränkt. Prüft Trefferinhalt alt = neu (Prüfmodus) und neu = neu mit `--trust-index` auf sauberem Baum. Linux-Teil: `wslbench.sh` aus dem Research-Ordner ohne `LMX_SPIKE_*`, Vergleich mit `grep -rnE --exclude-dir=.git --exclude-dir=.loomux`.

- [ ] **Schritt 1:** Skript schreiben (ohne Spike-Schalter; tgrep-Pfad als Argument).
- [ ] **Schritt 2:** Lauf unter Windows und WSL; Rohdaten neben das Skript.
- [ ] **Schritt 3: Abnahme** (Median, Toleranz 20 % schon eingerechnet): Prüfmodus Literal ≤ 108 ms, Alternation ≤ 132 ms, `-i` ≤ 114 ms, ohne Literal ≤ 324 ms, nach Änderung ≤ 420 ms, nach Commit ≤ 108 ms (Literal); `--trust-index` Literal/Alternation/`-i` ≤ 1,2 × tgrep-Median desselben Laufs; Linux Prüfmodus ≤ 1,2 × `grep -rn` mit Ausschlüssen je Muster. **Ein Verfehlen stoppt hier: Zahlen an den Menschen, kein Merge.**
- [ ] **Schritt 4:** Datierter Eintrag in `docs/en/benchmarks.md` und `docs/de/benchmarks.md` (Format wie die bestehenden Einträge). Im Eintrag „2026-11-06 11:35 — The Gate Against tgrep (Part C)“ bleibt Befund 2 stehen und wird als korrigiert markiert (EN :4091-4094, DE :4139-4142), ebenso die Zeile „Offen“ danach, mit Verweis auf den neuen Eintrag und das Research-Dokument.
- [ ] **Schritt 5:** Commit `docs: record the grep start-up benchmark and correct the earlier finding`.

### Aufgabe 16: Doku und Wiki

**Files:**
- Modify: `docs/en/cli-reference.md` (:847, :891, :895, :906-907, §8 `serve` :1201, `graph_find_all` :1363), `docs/de/cli-reference.md` (:773, :817, :821, :831-833, `serve` :1129)
- Modify: `docs/en/configuration.md:1193-1194`, `docs/de/configuration.md:1103-1104`
- Modify: `README.md:128`, `README.de.md:128`
- Modify: `docs/wiki/topics/code-graph.md:107-115`, `docs/wiki/log.md`

- [ ] **Schritt 1:** cli-reference beider Sprachen: `--trust-index`, `index` mit `overlay`/`trusted`, `index_built`, Overlay-Grenze 200 und Commit-Fall, neuer Text zu `--no-refresh`, Beiakten bei `graph build`, Wächter bei `serve`, `graph_find_all` mit Overlay. Bei `--trust-index` die Abweichung aus Spec 6 nennen: unter `--in`/`--glob` zählt `files_in_scope` alle Binärdateien mit, ein Präfix nur mit Binärdatei gibt `prefix not indexed`. Die doppelte deutsche Exit-Code-Zeile (:833) fällt weg. `go test ./internal/cli/ -run TestTheDocsNameTheGraphCommandsAsTheExceptionOfNever -count=1` grün.
- [ ] **Schritt 2:** configuration.md beider Sprachen: Zeilen für `grepspans.idx`, `corpus.json.bin`, `fingerprint.json.bin`.
- [ ] **Schritt 3:** Roadmap-Zeile in beiden READMEs: Ursache korrigiert (zwei Walks und Neubau, nicht das Laden), neuer Stand mit den Zahlen aus Aufgabe 15; ob tgrep jetzt abgelöst ist, entscheidet der Mensch nach der Abnahme, bis dahin „Offen: tgrep ablösen“.
- [ ] **Schritt 4:** Wiki `code-graph.md` (Overlay, `--trust-index`, Wächter) und ein Eintrag in `log.md` (Format :8-15); danach `loomux wiki repin` für die Seiten, die `docs/de/cli-reference.md` zitieren, nachdem sie neu gelesen und angepasst sind. `lint/wiki-sources` und `lint/wiki` grün.
- [ ] **Schritt 5:** `sh ci/gate.sh` grün; Commit `docs: describe the grep overlay, --trust-index and the serve watcher`.

### Abschluss

- [ ] Ganze Zweig-Prüfung (Review auf dem stärksten Modell), dann `release-pr` (Label `release:minor`). Push nur durch den Menschen.
- [ ] Aufräumen nach dem Merge, mit Rückfrage: Spike-Worktree `spike/grep-startup` samt `--no-verify`-Commit `35da79d3`, WSL `~/bench-grep-tree` und `~/spike-linux`, `.cache/tgrep-index` im Spike-Worktree.
- [ ] Eigener Punkt (nicht in diesem Zweig): `TrackedSet`/`Load` bei einer Wurzel in einem Unterverzeichnis des Repos (Pfade ohne Präfix).
