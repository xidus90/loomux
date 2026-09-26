# Stufe 4c-2: `dev bench search` und ein Berichtsschema — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux dev bench search` misst die Trefferqualität (Rang der
erwarteten Quelle, Treffer bei Rang ≤ 3) und die Latenz der Suchkette über
einen Fragensatz oder den eingecheckten Korpus `v1`. Die drei Messbefehle
stehen danach unter einer Gruppe, `dev bench hooks|repos|search`, und
schreiben eine gemeinsame Hülle je Lauf.

**Architecture:** Das neue Paket `internal/dev/benchreport` hält die Hülle
eines Laufs (`Report`, `Timing`, `Environment`), rechnet als einzige Stelle
Median, Minimum und Maximum und schreibt `.md` und `.json` beide oder keine.
`benchhooks` und `benchcorpus` ziehen darauf um; `docs/benchmarks.json`
bleibt ein Bestand je Repo mit `timings[]` in Millisekunden. Das neue Paket
`internal/dev/benchsearch` ist die Übertragung von `brain bench`: Fragensatz,
Korpusprüfung, Qualität, Latenz, Bericht und der Lauf samt Korpusmodus. Der
Korpusmodus sucht über `search.QmdPort` mit `--index loomux-bench-<zufall>`
in einem Wegwerf-Zustand, der Alltagsmodus über den Dienst
(`search.QmdMcpPort`).

**Tech Stack:** Go des Moduls, `gopkg.in/yaml.v3` (schon im `go.mod`),
`encoding/json`, `crypto/sha256`, `internal/lock`, `internal/config`,
`internal/brain/{search,index,identity,privacy,answer}`,
`internal/dev/fakeqmd`.

**Spec:** `docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md`,
Abschnitte „4c im Einzelnen“, „Abweichungen beim Planen von 4c“ (die Teile zu
4c-2) und **„Abweichungen beim Planen von 4c-2“** (gilt, wo er den beiden
anderen widerspricht), dazu „Parität“ und „Selbstnutzung“.

**Referenz:** `ultra-brain`, Tag `loomux-3-source`. Gelesen:
`src/brain/bench/{questions,corpus,quality,latency,report}.py` (vollständig),
`src/brain/cli.py:1813-2368` (`_bench` und alles, was es ruft). Der Korpus
liegt dort unter `bench/corpus/v1/` (106 Dateien, 330 246 Bytes).

## Befunde, gegen den Code gelesen am 2026-09-26

**B1. Zwei Bench-Befehle, zwei Mediane.** `devCommands`
(`internal/cli/dev.go:49-59`) hat `bench` → `devBench` (`:455`) und
`bench-hooks` → `devBenchHooks` (`:155`). `benchhooks.median`
(`benchhooks.go:162`) sortiert selbst, `benchcorpus.calculateMedian`
(`benchcorpus.go:347`) erwartet sortierte Eingabe; Minimum und Maximum liest
`benchcorpus` als `sorted[0]`/`sorted[len-1]` an sechs Stellen
(`benchcorpus.go:159-161`, `report.go:51-54`, `template.go:148-151`,
`matrix.go:188-199`). `formatDuration` (`report.go:119`) und `ms`
(`benchhooks.go:183`) sind dieselbe Funktion.

**B2. `benchhooks` hat kein JSON.** Es schreibt nur die Markdown-Tabelle auf
stdout (`benchhooks.go:48-68`); die Exit-Codes sind die des letzten Laufs.

**B3. `benchcorpus` schreibt Nanosekunden.** Alle Dauern sind
`time.Duration` (`types.go:47-85`). `docs/benchmarks.json` ist ein Array von
`RepoAudit` (243 Zeilen, 650 KB), geschrieben nur von `SaveReport`
(`storage.go:32-117`), das zusätzlich `docs/benchmarks-skipped.json`, die
Seiten `docs/{en,de}/benchmarks/<sprache>/<slug>.md` und
`docs/{en,de}/benchmarks/matrix.md` erzeugt.

**B4. `--timeout` ist tot.** `devBench` setzt `Options.Timeout`
(`dev.go:466`), gelesen wird es nirgends; `BenchmarkRepo` reicht nur
`opts.ComponentTimeout` an den Runner (`benchcorpus.go:92`). Der Fehler
liegt auf master.

**B5. Die Suchkette nimmt ihre Verzeichnisse als Parameter.**
`search.ExecuteSearch(query, scope, profile, n, channel, port, registryDir,
fallbackDir, now)` (`search/search.go:52`) und `answer.RunWith(ports, req,
registryDir, fallbackDir, notice)` (`answer/answer.go:109`) lesen den Zustand
nur über diese Parameter. Ein Wegwerf-Zustand braucht kein `os.Setenv`.
Ohne Abgleichsstempel meldet `StaleReconcile` nichts (`stamp.go:65`).

**B6. Ein read-only Bereich lebt im Zustand.** `config.ManifestDir` legt
Deklaration und Register eines `readonly`-Bereichs unter
`<zustand>/areas/<flat(scope)>/` (`config/manifest.go:481`); die Deklaration
heißt dort `.loomux/config.toml` (`manifest.go:45`). Das Register schreibt
`identity.RenderIdentities` (`identity.go:199`), neue IDs gibt
`identity.NewDocID` (`:56`), den Hash `identity.ContentHash` (`:71`).

**B7. `QmdPort` kennt keinen Indexnamen.** Er baut `qmd <sub> …` für
`search|vsearch|query`, `ls`, `update`, `status`, `embed`
(`search/qmd.go:72-243`). `parseQmdJSON` nimmt nach `qmd://<coll>/` den Rest
als Pfad (`:124-126`); mit `--index` hängt qmd 2.8.3 `?index=<name>` an
(gemessen 2026-09-26), bei `ls` und `status` nicht.

**B8. `QmdConfigPath` kennt nur `index.yml`** (`index/qmdconfig.go:141-151`).
`SyncCollections(configPath, wanted, owned)` (`:228`) nimmt den Pfad als
Parameter und legt beim ersten Schreiben `configPath + ".brain-backup"` an
(`:289-296`). `index.Models(configPath)` (`:412`) liest den Block `models:`.

**B9. `fakeqmd` übergeht die Anfrage.** `hitsIn` (`fakeqmd.go:221`) liefert
die Treffer der Fixture in ihrer Reihenfolge, nur nach Sammlung gefiltert;
`RunCLI` (`:75`) kennt kein `--index`, `searchFlags` (`:188`) verweigert jedes
unbekannte Argument. Die Fixture wird strikt dekodiert (`:62`).

**B10. Ein Katalog braucht `index.md`.** `catalog.ReadAreaCatalog` liest
`<bereichszustand>/index.md` und scheitert, wenn es fehlt
(`catalog/area.go`). Im Wegwerf-Zustand des Korpusmodus gibt es keines.

**B11. Sperren.** `lock.TryAcquire(path) (*Handle, bool, error)` und
`lock.Acquire` (`lock/lock.go:30-35`); `config.StateDir()` ist der echte
Zustand (`config/registry.go:205`).

**B12. Version.** `cli.Version` wird per `-ldflags -X` gesetzt
(`release/build.go:41`).

## Entscheidungen

Aus der Spec übernommen, hier nur, was der Plan zusätzlich festlegt (zur
Durchsicht durch den Nutzer):

- **E1. `--out DIR` für alle drei Befehle.** Jeder schreibt
  `bench-<stempel>-<befehl>[-<profil>].md` und `.json` in ein vorhandenes
  Verzeichnis, beide oder keine; liegt eine davon schon da, bricht der Lauf vor
  der ersten Messung ab. `dev bench repos` verliert `--json-out`, und `--out`
  nimmt dort ein Verzeichnis statt einer Datei. Das ist Teil des ohnehin
  brechenden Umbaus (`release:major`). `dev bench search` behält die
  Dateinamen der Referenz, `bench-<stempel>-<profil>.{md,json}`, weil der
  echte Fragensatz in `98 Messung` schon Dateien dieses Namens trägt.
- **E2. Der Bericht ist englisch.** Überschriften, Hinweise und Fehlermeldungen
  von `dev bench search` sind englisch wie jede Ausgabe von loomux; die
  Sortennamen `exakt`, `umschreibung`, `gemischt`, `sprachuebergreifend` und
  die Feldnamen des Fragensatzes bleiben, weil sie das Dateiformat sind. Die
  deutschen Seiten von `benchcorpus` (`template.go`, `matrix.go`) bleiben
  zweisprachig wie heute.
- **E3. `--corpus` mit `--latency` wird verweigert.** Im Wegwerf-Zustand gibt
  es kein `index.md` (B10), die Operation `catalog` scheiterte also, und die
  Latenz über die CLI ist laut Spec nicht zu berichten. Meldung:
  `--latency measures the everyday chain; drop it with --corpus`.
- **E4. Die Hülle trägt die Nutzlast als `json.RawMessage`.** `payload` ist je
  Befehl eine eigene Struktur (`search`: Fragen, Befunde, Latenzziel;
  `repos`: die `RepoAudit`-Zeilen; `hooks`: keine). So bleibt `Report` ohne
  `any`.
- **E5. Sperre und Sammlung.** Die Sperre je Indexname liegt unter
  `<echter zustand>/bench/<name>.lock`. Aufgeräumt wird in einem `defer`; ein
  Fehler dort ist eine Warnung auf stderr, kein Exit 1 (wie `_corpus_clear`
  der Referenz).
- **E6. Median.** `benchreport.Median` mittelt bei gerader Anzahl die beiden
  mittleren Werte in `float64` Millisekunden. Die alte Ganzzahl-Mittelung von
  `benchhooks` (`(a+b)/2` in Nanosekunden) fällt weg; der Unterschied ist
  höchstens eine halbe Nanosekunde.

## Global Constraints

- Coverage 100 % je Funktion; eine Ausnahme nur mit `//coverage:exempt <grund>`
  direkt über `func` (AGENTS.md).
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst.
- Code, Bezeichner, Kommentare, Meldungen und Commits englisch; dieser Plan und
  die Akte `parity/stufe-4c-2.md` deutsch.
- Commits nach Conventional Commits, ohne Arbeitspapier im Text, ohne
  Modell als Mitautor. Scope nennt ein Codegebiet (`dev`, `search`, `index`).
- Mehrzeilige Commit-Nachrichten per Datei im Scratchpad und
  `git commit -F <datei>`. **Kein `Co-Authored-By`** und keine Werbezeile,
  auch wenn die Harness es vorschlägt — die Regel des Nutzers geht vor. Nie
  `--no-verify`. Die Ausgabe des Tors erst ganz in eine Datei im Scratchpad,
  dann filtern.
- Der Korpus steht unter CC BY-SA 4.0 und behält Lizenz und
  Quellenangabe in seinem Verzeichnis; `.gitattributes` setzt dort `-text`.
- Nie die echte `index.yml` oder `index.sqlite` in einem Test berühren:
  Tests laufen gegen `fakeqmd` oder `search.FakePort`, mit `t.TempDir()` als
  Zustand und `XDG_CONFIG_HOME` auf ein `t.TempDir()`.
- Exit 1 bei jedem Fehler von Korpus, Fragensatz, Suche, Konfiguration und
  Dateisystem, `error: <problem>` je Zeile auf stderr; Exit 2 bei falscher
  Benutzung der Flags. Keine Schwelle für Qualität oder Latenz.
- Release `release:major` (Umbenennung der Befehle).

## Review Focus

1. **Zwei Läufe gleichzeitig im Korpusmodus.** Der zweite darf den Index des
   ersten nicht wegräumen. Test in Task 9: eine gehaltene Sperre auf
   `loomux-bench-a` → `sweepStale` lässt `loomux-bench-a.yml` stehen.
2. **Absturz mitten in der Vorbereitung.** Ein Fehler nach dem Anlegen von
   `<name>.yml`, aber vor dem ersten Suchlauf, hinterlässt weder Wegwerf-Zustand
   noch `<name>.*`. Test in Task 9 mit einem Port, dessen `Refresh` scheitert.
3. **Windows-Pfade im Fragensatz.** `expect` in anderer Groß-/Kleinschreibung
   oder mit `..` trifft dieselbe Datei. Test in Task 6 (`TestRankMatchesAcrossSpelling`).
4. **Ein halber Bericht.** Scheitert das Schreiben des JSON, ist das `.md`
   wieder weg, und der Lauf endet mit Exit 1. Test in Task 2
   (`TestWriteBothRemovesTheMarkdownWhenTheJSONFails`).
5. **Ein Korpus mit CRLF-Zeilenenden.** Ein Checkout mit `autocrlf` würde die
   Prüfsummen brechen. Test in Task 5: eine Notiz mit `\r\n` → Befund
   `checksum … does not match the manifest`; `.gitattributes` setzt `-text`.

---

### Task 1: `--timeout` von `dev bench` wirkt je Repository

Eigenständiger `fix`, denn der Fehler liegt auf master (B4).

**Files:**
- Modify: `internal/dev/benchcorpus/benchcorpus.go:37-176`
- Test: `internal/dev/benchcorpus/benchcorpus_test.go`

**Interfaces:**
- Produces: `BenchmarkRepo` gibt `ErrRepoTimeout` (umhüllt) zurück, sobald
  `opts.Timeout > 0` abgelaufen ist; jeder Runner-Aufruf bekommt
  `min(opts.ComponentTimeout, verbleibend)`.

- [ ] **Step 1: Failing test**

```go
func TestBenchmarkRepoStopsAtTheRepositoryTimeout(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	var budgets []time.Duration
	runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
		budgets = append(budgets, timeout)
		now = now.Add(40 * time.Second)
		return "", 0, false, nil
	}
	opts := Options{WarmRuns: 3, Timeout: 100 * time.Second, ComponentTimeout: 60 * time.Second}
	_, err := BenchmarkRepo(".", opts, runner, clock, goRepoFS, noLookPath)
	if !errors.Is(err, ErrRepoTimeout) {
		t.Fatalf("err = %v, want ErrRepoTimeout", err)
	}
	want := []time.Duration{60 * time.Second, 60 * time.Second, 20 * time.Second}
	if !slices.Equal(budgets, want) {
		t.Fatalf("budgets = %v, want %v", budgets, want)
	}
}

func TestBenchmarkRepoWithoutTimeoutKeepsTheComponentBudget(t *testing.T) {
	// Timeout 0 means no repository deadline, as before.
	...same runner, opts.Timeout = 0 → err == nil, every budget 60s
}
```

`goRepoFS` und `noLookPath` sind die vorhandenen Hilfen in
`benchcorpus_test.go` (eine `fstest.MapFS` mit `go.mod` und `main.go`, ein
`lookPath`, das alles findet); fehlen sie unter diesem Namen, die dort
benutzten Hilfen nehmen.

- [ ] **Step 2: Run** `go test ./internal/dev/benchcorpus -run RepositoryTimeout` → FAIL (`ErrRepoTimeout` undefiniert).

- [ ] **Step 3: Implementation**

```go
// ErrRepoTimeout ends the measurement of one repository once --timeout is
// spent; BenchmarkCorpus records it as a skip like any other failure.
var ErrRepoTimeout = errors.New("repository timeout exceeded")
```

In `BenchmarkRepo` vor `measure`:

```go
	var deadline time.Time
	if opts.Timeout > 0 {
		deadline = clock().Add(opts.Timeout)
	}
	budget := func() (time.Duration, error) {
		if deadline.IsZero() {
			return opts.ComponentTimeout, nil
		}
		left := deadline.Sub(clock())
		if left <= 0 {
			return 0, fmt.Errorf("%w after %s", ErrRepoTimeout, opts.Timeout)
		}
		return min(opts.ComponentTimeout, left), nil
	}
```

und in `measure` statt `opts.ComponentTimeout`:

```go
			limit, err := budget()
			if err != nil {
				return nil, 0, err
			}
			t0 := clock()
			_, code, timedOut, err := runner(absDir, step.argv, step.stdin, limit)
```

Ein `ComponentTimeout` von 0 heißt heute „kein Limit“; `min(0, left)` wäre 0
und damit ebenfalls „kein Limit“, der Frist zum Trotz. Deshalb:
`if opts.ComponentTimeout <= 0 { return left, nil }` vor dem `min`. Test dafür:
`TestBenchmarkRepoWithoutComponentTimeoutUsesTheRepositoryBudget`.

Die Baseline läuft durch dasselbe `measure`; ein Timeout dort setzt wie jeder
Fehler `baselineErr` (`benchcorpus.go:115-123`) — das ist gewollt: die
Baseline ist Vergleich, nicht Messung.

- [ ] **Step 4: Run** `go test ./internal/dev/benchcorpus` → PASS; `go run ./cmd/loomux check coverage` für das Paket grün.

- [ ] **Step 5: Commit**

```
fix(dev): honour --timeout per repository in dev bench

The flag was parsed and never read, so a hanging repository held a
corpus run for as long as its components allowed. Each command now gets
the smaller of its own deadline and what is left of the repository's.
```

---

### Task 2: `internal/dev/benchreport`, die Hülle eines Laufs

**Files:**
- Create: `internal/dev/benchreport/report.go`, `stats.go`, `write.go`, `environment.go`
- Test: `internal/dev/benchreport/report_test.go`, `stats_test.go`, `write_test.go`, `environment_test.go`

**Interfaces:**
- Produces:

```go
package benchreport

const Schema = 1

type Timing struct {
	Name       string    `json:"name"`
	ColdMS     float64   `json:"cold_ms"`
	WarmMS     []float64 `json:"warm_ms"`
	MedianMS   float64   `json:"median_ms"`
	MinMS      float64   `json:"min_ms"`
	MaxMS      float64   `json:"max_ms"`
	ExitCodes  []int     `json:"exit_codes,omitempty"`
	Applicable *bool     `json:"applicable,omitempty"` // nil: applicable
	TimedOut   int       `json:"timed_out,omitempty"`  // runs that hit their deadline
}

type Environment struct {
	OS      string            `json:"os"`
	Arch    string            `json:"arch"`
	CPU     string            `json:"cpu"`
	Go      string            `json:"go"`
	Loomux  string            `json:"loomux"`
	Qmd     string            `json:"qmd,omitempty"`
	Models  map[string]string `json:"models,omitempty"`
	Profile string            `json:"profile,omitempty"`
	Port    string            `json:"port,omitempty"` // "daemon" | "cli"
}

type Report struct {
	Schema      int             `json:"schema"`
	Command     string          `json:"command"` // "hooks" | "repos" | "search"
	Stamp       string          `json:"stamp"`
	Environment Environment     `json:"environment"`
	Timings     []Timing        `json:"timings"`
	Payload     json.RawMessage `json:"payload,omitempty"`
}

func MS(d time.Duration) float64
func Median(ms []float64) float64          // 0 for none
func Summarize(name string, coldMS float64, warmMS []float64) Timing
func Stamp(now time.Time) string           // UTC "2006-01-02-1504"
func Current(loomux string) Environment    // runtime.GOOS, GOARCH, runtime.Version(), cpuName()
func (r Report) JSON() ([]byte, error)     // indent 2, trailing newline
func (t Timing) Row() string               // "| name | 1.2 ms | 3.4 ms | 1.0 ms | 5.0 ms |"
func FormatMS(ms float64) string           // "%.1f ms"
func Targets(dir, base string) (md, js string, err error)  // both absent, dir exists
func WriteBoth(md, js string, text, payload []byte) error   // both or neither
```

- [ ] **Step 1: Failing tests** (`stats_test.go`)

```go
func TestMedianOfAnOddCountIsTheMiddle(t *testing.T) {
	if got := Median([]float64{5, 1, 3}); got != 3 {
		t.Fatalf("Median = %v, want 3", got)
	}
}

func TestMedianOfAnEvenCountAveragesTheMiddleTwo(t *testing.T) {
	if got := Median([]float64{4, 1, 2, 3}); got != 2.5 {
		t.Fatalf("Median = %v, want 2.5", got)
	}
}

func TestMedianOfNothingIsZeroAndLeavesTheInputUnsorted(t *testing.T) {
	in := []float64{3, 1, 2}
	Median(in)
	if !slices.Equal(in, []float64{3, 1, 2}) {
		t.Fatalf("input reordered to %v", in)
	}
	if Median(nil) != 0 {
		t.Fatal("Median(nil) != 0")
	}
}

func TestMSIsTheDivisionTheOldRendererDid(t *testing.T) {
	d := 89138123 * time.Nanosecond
	if MS(d) != float64(d)/float64(time.Millisecond) {
		t.Fatal("MS differs from float64(d)/1e6")
	}
}

func TestSummarizeKeepsColdApartFromTheWarmRuns(t *testing.T) {
	got := Summarize("keyword", 900, []float64{12, 10, 11})
	want := Timing{Name: "keyword", ColdMS: 900, WarmMS: []float64{12, 10, 11},
		MedianMS: 11, MinMS: 10, MaxMS: 12}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 2: Run** `go test ./internal/dev/benchreport` → FAIL (Paket fehlt).

- [ ] **Step 3: `stats.go`**

```go
// Package benchreport is the one shape every benchmark of loomux leaves
// behind: a head naming where it ran, the timings of one run, and a payload
// of the command's own.
package benchreport

// MS is a duration in milliseconds, the unit every report carries. It is
// the very division the renderers did before, so a stored value renders to
// the same text.
func MS(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

// Median is the middle of the warm runs; an even count averages the two
// middle values, so a run of four does not report the upper one.
func Median(ms []float64) float64 {
	if len(ms) == 0 {
		return 0
	}
	s := slices.Clone(ms)
	slices.Sort(s)
	half := len(s) / 2
	if len(s)%2 == 1 {
		return s[half]
	}
	return (s[half-1] + s[half]) / 2
}

// Summarize keeps the cold run apart: folded into the median it would hide
// both itself and the warm spread.
func Summarize(name string, coldMS float64, warmMS []float64) Timing {
	t := Timing{Name: name, ColdMS: coldMS, WarmMS: warmMS, MedianMS: Median(warmMS)}
	if len(warmMS) > 0 {
		t.MinMS, t.MaxMS = slices.Min(warmMS), slices.Max(warmMS)
	}
	return t
}

// FormatMS is the one spelling of a time in a table.
func FormatMS(ms float64) string { return fmt.Sprintf("%.1f ms", ms) }

// Row is a timing as a table row: cold, warm median, minimum, maximum.
func (t Timing) Row() string {
	return fmt.Sprintf("| %s | %s | %s | %s | %s |", t.Name,
		FormatMS(t.ColdMS), FormatMS(t.MedianMS), FormatMS(t.MinMS), FormatMS(t.MaxMS))
}
```

- [ ] **Step 4: Failing tests** (`report_test.go`, `environment_test.go`)

```go
func TestStampIsUTCToTheMinute(t *testing.T) {
	at := time.Date(2026, 9, 26, 23, 59, 30, 0, time.FixedZone("CEST", 2*3600))
	if got := Stamp(at); got != "2026-09-26-2159" {
		t.Fatalf("Stamp = %q", got)
	}
}

func TestReportJSONIsIndentedWithANewlineAndOmitsAnEmptyPayload(t *testing.T) {
	r := Report{Schema: Schema, Command: "hooks", Stamp: "s",
		Environment: Environment{OS: "windows", Arch: "amd64", CPU: "c", Go: "go1.27.0", Loomux: "dev"},
		Timings: []Timing{Summarize("a", 1, []float64{2})}}
	got, err := r.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(got, []byte("}\n")) || bytes.Contains(got, []byte(`"payload"`)) {
		t.Fatalf("unexpected JSON:\n%s", got)
	}
	if !bytes.Contains(got, []byte("\n  \"schema\": 1,")) {
		t.Fatalf("not indented by two:\n%s", got)
	}
}

func TestCurrentNamesTheRuntime(t *testing.T) {
	env := Current("3.2.0")
	if env.OS != runtime.GOOS || env.Arch != runtime.GOARCH || env.Go != runtime.Version() || env.Loomux != "3.2.0" {
		t.Fatalf("env = %+v", env)
	}
}
```

- [ ] **Step 5: `report.go` und `environment.go`**

```go
// Stamp is the minute a run started, in UTC, as file names and heads carry it.
func Stamp(now time.Time) string { return now.UTC().Format("2006-01-02-1504") }

// JSON is the report as written to disk: indented by two, one newline at the end.
func (r Report) JSON() ([]byte, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
```

```go
// cpuName is a seam: the processor's name comes from a different place on
// every system, and a test must not depend on the machine it runs on.
var cpuName = processorName

// Current is the environment this process runs in; the caller adds what
// only it knows (qmd, models, profile, port).
func Current(loomux string) Environment {
	return Environment{OS: runtime.GOOS, Arch: runtime.GOARCH, CPU: cpuName(),
		Go: runtime.Version(), Loomux: loomux}
}

// processorName reads the processor's name where the system hands it out
// without a subprocess, and the logical CPU count otherwise.
//
//coverage:exempt reads the host's processor name from the environment or /proc, which differs per machine
func processorName() string {
	if name := os.Getenv("PROCESSOR_IDENTIFIER"); name != "" {
		return name
	}
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if key, value, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(key) == "model name" {
				return strings.TrimSpace(value)
			}
		}
	}
	return fmt.Sprintf("%d logical CPUs", runtime.NumCPU())
}
```

Die Paketvariable `cpuName` parst nichts und ist damit kein Verstoß gegen
„keine Paketvariable, die eingebettete Daten parst“.

- [ ] **Step 6: Failing tests** (`write_test.go`)

```go
func TestTargetsRefusesAnExistingFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bench-s-hooks.json"), nil, 0o644)
	if _, _, err := Targets(dir, "bench-s-hooks"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err = %v", err)
	}
}

func TestTargetsRefusesAMissingDirectory(t *testing.T) {
	_, _, err := Targets(filepath.Join(t.TempDir(), "nope"), "b")
	if err == nil || !strings.Contains(err.Error(), "no directory at") {
		t.Fatalf("err = %v", err)
	}
}

func TestWriteBothWritesLFAsGiven(t *testing.T) {
	dir := t.TempDir()
	md, js, _ := Targets(dir, "b")
	if err := WriteBoth(md, js, []byte("# a\n"), []byte("{}\n")); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(md)
	if string(got) != "# a\n" {
		t.Fatalf("md = %q", got)
	}
}

func TestWriteBothRemovesTheMarkdownWhenTheJSONFails(t *testing.T) {
	dir := t.TempDir()
	md := filepath.Join(dir, "b.md")
	js := filepath.Join(dir, "missing", "b.json") // parent absent: the write fails
	if err := WriteBoth(md, js, []byte("x"), []byte("y")); err == nil {
		t.Fatal("no error")
	}
	if _, err := os.Stat(md); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("markdown left behind: %v", err)
	}
}
```

- [ ] **Step 7: `write.go`**

```go
// Targets names the two files of a run and refuses before the first
// measurement if either exists: a run that measures for minutes and then
// cannot save has wasted them.
func Targets(dir, base string) (string, string, error) {
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return "", "", fmt.Errorf("no directory at %s; name one with --out", dir)
	}
	md, js := filepath.Join(dir, base+".md"), filepath.Join(dir, base+".json")
	for _, target := range []string{md, js} {
		if _, err := os.Stat(target); err == nil {
			return "", "", fmt.Errorf("%s already exists", target)
		}
	}
	return md, js, nil
}

// WriteBoth writes both files or neither: a markdown without its JSON is a
// half report, and it would take the name a rerun in the same minute needs.
func WriteBoth(md, js string, text, payload []byte) error {
	if err := os.WriteFile(md, text, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(js, payload, 0o644); err != nil {
		_ = os.Remove(md)
		return err
	}
	return nil
}
```

- [ ] **Step 8: Run** `go test ./internal/dev/benchreport -cover` → PASS, 100 % außer `processorName`.

- [ ] **Step 9: Commit**

```
feat(dev): add one report shape for every benchmark

A run's head (system, versions, models, profile, search path), its
timings in milliseconds with median, minimum and maximum computed in one
place, and a payload of the command's own. Markdown and JSON are written
both or neither, and an existing file is refused before measuring.
```

---

### Task 3: `benchhooks` auf die Hülle

**Files:**
- Modify: `internal/dev/benchhooks/benchhooks.go:42-185`, `internal/cli/dev.go:155-190` (Step 4)
- Test: `internal/dev/benchhooks/benchhooks_test.go`

**Interfaces:**
- Consumes: `benchreport.Summarize`, `benchreport.MS`, `Timing.Row`.
- Produces: `func Measure(cases []Case, n int, run func(Case, Step) (int, error), now func() time.Time) ([]benchreport.Timing, error)` und `func Table(timings []benchreport.Timing) string`. `Run` entfällt; der Befehl (Task 4) ruft `Measure`, schreibt `Table` auf stdout und mit `--out` beide Dateien.

- [ ] **Step 1: Tests umstellen.** Die 12 Tests in `benchhooks_test.go` rufen heute `Run(cases, n, &buf, run, now)` und vergleichen die Tabelle. Sie rufen künftig `Measure` und vergleichen `Table(timings)` mit **derselben** erwarteten Tabelle — bis auf die Spalte `exit codes`, die jetzt aus `Timing.ExitCodes` kommt (die Codes des letzten Laufs, wie heute). Dazu neu:

```go
func TestMeasureReportsEveryCaseAsATiming(t *testing.T) {
	clock := fakeClock(10*time.Millisecond, 2*time.Millisecond, 4*time.Millisecond, 3*time.Millisecond)
	got, err := Measure([]Case{{Name: "guard", Steps: []Step{{Argv: []string{"x"}}}}}, 3, exitZero, clock)
	if err != nil {
		t.Fatal(err)
	}
	want := []benchreport.Timing{{Name: "guard", ColdMS: 10, WarmMS: []float64{2, 4, 3},
		MedianMS: 3, MinMS: 2, MaxMS: 4, ExitCodes: []int{0}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}
```

`fakeClock` und `exitZero` sind die Hilfen, die die Datei heute schon für
`Run` hat; heißen sie anders, deren Namen nehmen.

- [ ] **Step 2: Run** → FAIL (`Measure` undefiniert).

- [ ] **Step 3: Implementation.** `Run` wird zu `Measure` (Validierung wie
  heute zuerst), die Schleife sammelt `benchreport.MS(d)` statt Dauern und
  hängt `benchreport.Summarize(c.Name, cold, warm)` mit `ExitCodes: codes`
  an. `median`, `low`, `high`, `ms` fallen weg.

```go
// Table is the markdown the command prints: one row per case, the exit
// codes of the last run in the last column.
func Table(timings []benchreport.Timing) string {
	var b strings.Builder
	b.WriteString("| case | cold (1st run) | warm median | warm min | warm max | exit codes |\n")
	b.WriteString("|---|---:|---:|---:|---:|---|\n")
	for _, t := range timings {
		fmt.Fprintf(&b, "%s %v |\n", t.Row(), t.ExitCodes)
	}
	return b.String()
}
```

- [ ] **Step 4: Run** `go test ./internal/dev/benchhooks -cover` → PASS, 100 %. `go build ./...` scheitert noch an `internal/cli/dev.go` (`benchhooks.Run`); das zieht Task 4 nach — bis dahin in `devBenchHooks` die Zeile `err = benchhooks.Run(...)` ersetzen durch:

```go
	var timings []benchreport.Timing
	if err == nil {
		timings, err = benchhooks.Measure(cases, *n, benchExec, time.Now)
	}
	if err == nil {
		fmt.Fprint(stdout, benchhooks.Table(timings))
	}
```

Die vorhandenen Tests in `internal/cli/dev_test.go:212-291` bleiben grün.

- [ ] **Step 5: Commit** — wird beim Gruppieren in den Commit von Task 4 gefaltet (derselbe Umbau); jetzt als Zwischenstand:

```
refactor(dev): measure hook cases into the shared report timings
```

---

### Task 4: `benchcorpus` auf die Hülle und `docs/benchmarks.json` in Millisekunden

**Files:**
- Modify: `internal/dev/benchcorpus/types.go:46-108`, `benchcorpus.go`, `report.go`, `template.go`, `matrix.go`, `status.go`, `storage.go`
- Modify: `internal/cli/dev.go:455-548` und `internal/cli/dev_test.go:530-760` nur so weit, wie die geänderten Felder von `RepoAudit` es verlangen (dort gebaute `RepoAudit`-Werte bekommen `Timings`); `BenchmarkReport` und die Aufrufe bleiben
- Modify: `docs/benchmarks.json` (umgerechnet, siehe Step 6)
- Test: alle `*_test.go` des Pakets

Vor Step 1 `grep -rn "WarmMedian\|WarmMin\|WarmMax\|TimingRun\|ComponentTiming\|\.Cold\b\|\.Warm\b" internal cmd --include=*.go` außerhalb des Pakets laufen lassen; jeder Treffer gehört in diese Task. Am 2026-09-26 waren es nur `internal/cli/dev.go` und `dev_test.go`.

**Interfaces:**
- Consumes: `benchreport.Timing`, `Summarize`, `MS`, `FormatMS`, `Median`.
- Produces:

```go
type RepoAudit struct {
	RepoURL        string               `json:"repo_url"`
	Dir            string               `json:"dir"`
	Language       string               `json:"language"`
	Framework      string               `json:"framework,omitempty"`
	Tier           string               `json:"tier"`
	CommitSHA      string               `json:"commit_sha,omitempty"`
	SampleFile     string               `json:"sample_file,omitempty"`
	DetectedStacks []string             `json:"detected_stacks"`
	ExecutedLanes  []string             `json:"executed_lanes"`
	Audit          []CheckAudit         `json:"audit"`
	MissingGaps    []string             `json:"missing_gaps"`
	CoverageRate   float64              `json:"coverage_rate"`
	Timings        []benchreport.Timing `json:"timings"`
	HookWarmMedian float64              `json:"hook_warm_median_ms,omitempty"`
	ClaudeWarmMed  float64              `json:"claude_warm_median_ms,omitempty"`
	Speedup        float64              `json:"speedup,omitempty"`
	BaselineError  string               `json:"baseline_error,omitempty"`
}

// Named timings of a row, in this order: TotalTiming first, then one per
// component, then one per baseline component as "baseline:<name>".
const TotalTiming = "total"
func BaselineTiming(component string) string   // "baseline:" + component
func (a *RepoAudit) Timing(name string) (benchreport.Timing, bool)
func (a *RepoAudit) Components() []benchreport.Timing  // neither total nor baseline
```

`ComponentTiming` und `TimingRun` entfallen. **`BenchmarkReport` bleibt** als
Behälter eines Laufs (`Timestamp`, `Mode`, `WarmRuns`, `Repos`, `Skipped`),
ebenso diese Signaturen, damit `internal/cli` in dieser Task nur die Felder
von `RepoAudit` nachzieht:

```go
func BenchmarkRepo(dir string, opts Options, runner ProcessRunner, clock func() time.Time, openFS func(string) (fs.FS, error), lookPath func(string) (string, error)) (*RepoAudit, error)
func BenchmarkCorpus(matrixData []byte, opts Options, cloner Cloner, benchRepo func(string, Options) (*RepoAudit, error)) (*BenchmarkReport, error)
func FormatMarkdown(report *BenchmarkReport, w io.Writer) error
func FormatJSON(report *BenchmarkReport, w io.Writer) error          // bis Task 5; dann ersetzt durch die Hülle
func FormatMatrixMarkdown(report *BenchmarkReport, lang string, w io.Writer) error // Zeitstempel aus report.Timestamp
func FormatDetailMarkdown(audit *RepoAudit, lang string, w io.Writer) error
func SaveReport(report *BenchmarkReport, docsDir string, ops StorageOps) error
```

Die JSON-Ausgabe eines Laufs als Hülle (`Command: "repos"`, `Timings`: die
`total`-Einträge je Repo unter dem Namen des Repos, `Payload`:
`{"repos": [...], "skipped": [...]}`) baut erst Task 5, denn sie braucht
`cli.Version` für den Kopf; `FormatJSON` fällt dort weg.

- [ ] **Step 1: Tests zuerst umstellen.** Jeder Test, der heute `Cold`,
  `Warm`, `WarmMedian`, `WarmMin`, `WarmMax`, `ComponentTiming` oder
  `TimingRun` baut, baut stattdessen `Timings` über eine Testhilfe:

```go
func timings(total benchreport.Timing, parts ...benchreport.Timing) []benchreport.Timing {
	total.Name = TotalTiming
	return append([]benchreport.Timing{total}, parts...)
}
```

Die erwarteten Markdown-Texte (`report_test.go`, `template_test.go`,
`matrix_test.go`) ändern sich **nicht**: gleiche Millisekunden ergeben gleiche
Zeilen. Ein Test, dessen erwarteter Text sich ändern müsste, zeigt einen
Fehler im Umbau.

Neuer Test in `benchcorpus_test.go`:

```go
func TestBenchmarkRepoNamesTotalComponentsAndBaseline(t *testing.T) {
	// one Claude hook in .claude/settings.json → "baseline:claude PreToolUse"
	audit := runFakeRepoWithBaseline(t)
	var names []string
	for _, tm := range audit.Timings {
		names = append(names, tm.Name)
	}
	want := []string{"total", "pre-tool-use", "post-tool-use", "graph build", "baseline:claude PreToolUse"}
	if !slices.Equal(names, want) {
		t.Fatalf("names = %v", names)
	}
	if g, _ := audit.Timing("graph build"); g.Applicable == nil || *g.Applicable {
		t.Fatalf("graph build without Go must be applicable=false: %+v", g)
	}
}
```

`runFakeRepoWithBaseline` entsteht aus der vorhandenen Fixture des Tests,
der heute die Baseline prüft (Suche nach `ClaudeHooks` in
`benchcorpus_test.go`).

- [ ] **Step 2: Run** → FAIL (Kompilierfehler).

- [ ] **Step 3: `BenchmarkRepo`.** `measure` liefert je Schritt `(ms float64,
  code int, timedOut bool, applicable bool)`. Nach der Schleife entsteht je
  Name eine `benchreport.Summarize(name, coldMS, warmMS)`, dazu
  `ExitCodes` (einer je Lauf: kalt, dann warm), `TimedOut` (Zahl der Läufe)
  und `Applicable: ptr(false)`, wenn der Schritt nicht anwendbar war.
  `HookWarmMedian = benchreport.Median(hookWarmMS)`,
  `ClaudeWarmMed = benchreport.Median(claudeWarmMS)`,
  `Speedup = ClaudeWarmMed / HookWarmMedian`. `calculateMedian` entfällt.

- [ ] **Step 4: Renderer.** `template.go` und `matrix.go` lesen die Werte aus
  `audit.Timing(name)` bzw. `audit.Components()` statt sie aus `Warm[]` neu
  zu rechnen (`template.go:131-163`, `matrix.go:175-199`);
  `formatDuration(x)` wird `benchreport.FormatMS(x)`. `exitStatus`,
  `componentRuns` und `allRuns` lesen `Timing.ExitCodes` und
  `Timing.TimedOut`.

- [ ] **Step 5: Run** `go test ./internal/dev/benchcorpus -cover` → PASS, 100 %.

- [ ] **Step 6: Umrechnung von `docs/benchmarks.json` (Wegwerf, nicht
  eingecheckt).** Als Wegwerf-Test `internal/dev/benchcorpus/zz_convert_test.go`
  im Paket — ein eigenes Modul im Scratchpad dürfte
  `github.com/xidus90/loomux/internal/…` nicht importieren. Die **alten** Typen
  (aus `git show master:internal/dev/benchcorpus/types.go`) werden dort unter
  eigenen Namen lokal deklariert (`oldAudit`, `oldRun`, `oldComponent`), die
  neuen sind die echten des Pakets. Der Test liest den Bestand, schreibt ihn
  neu und wird nach dem Lauf gelöscht (`git status` zeigt ihn danach nicht
  mehr). Ablauf:

```go
for each old row:
	var ts []Timing
	ts = append(ts, sum("total", old.Cold.Total, totals(old.Warm)))        // median_ms := MS(old.WarmMedian), min/max aus WarmMin/WarmMax
	for i, c := range old.Cold.Components:
		ts = append(ts, component(c, warmAt(old.Warm, i)))                  // median wie der alte Renderer: calculateMedian(sorted)
	for i, c := range old.Cold.Baseline:
		ts = append(ts, component(c, baselineAt(old.Warm, i)), name "baseline:"+c.Name)
	new.HookWarmMedian = MS(old.HookWarmMedian); new.ClaudeWarmMed = MS(old.ClaudeWarmMed)
```

**Wichtig:** Der Median einer Zeile wird aus dem **alten** Ganzzahl-Median
(`calculateMedian` in Nanosekunden) umgerechnet, nicht neu in `float64`
gemittelt — nur so rendern die Seiten bytegleich (E6 gilt für neue Läufe).
`ExitCodes` je Lauf aus `ExitCode` der Komponenten, `TimedOut` als Zahl.
JSON mit `json.MarshalIndent(rows, "", "  ")` und abschließendem `\n`, wie
`SaveReport` heute schreibt.

Prüfung: Mit dem neuen Binary die Seiten aus dem umgerechneten JSON neu
erzeugen, ohne zu messen. Dafür bekommt `SaveReport` keinen neuen Weg;
stattdessen ein Test im Paket, der **einmal lokal** läuft:

```go
// TestRegeneratedPagesMatchTheCommittedOnes renders matrix.md and every
// per-repo page from docs/benchmarks.json and compares them byte for byte
// with the committed files. It guards the one-off conversion of the store
// to milliseconds.
func TestRegeneratedPagesMatchTheCommittedOnes(t *testing.T) { ... }
```

Er liest `../../../docs/benchmarks.json` und `benchmarks-skipped.json`,
rendert mit `FormatMatrixMarkdown` und `FormatDetailMarkdown` für `en` und
`de` und vergleicht mit `docs/{en,de}/benchmarks/matrix.md` und den Seiten je
Repo. Den Zeitstempel der Matrix steht nicht im Bestand; der Test liest ihn
aus der Zeile `- **Last Updated:** …` der committeten `matrix.md` und reicht
ihn dem Renderer. Auf master nachgeprüft (2026-09-26, Wegwerftest): der
heutige Renderer erzeugt aus dem heutigen Bestand alle 486 Seiten je Repo
bytegleich, die Matrix bis auf eben diese Zeile. Das Kriterium ist also
erreichbar. `exitStatus` (`status.go:11`) fragt nur, ob **irgendein** Lauf in
die Frist lief, und sonst nach der Menge der Exit-Codes; `TimedOut` als Zahl
und `ExitCodes` je Lauf tragen das, die Zeile „Gesamt“ vereinigt beides über
die anwendbaren Komponenten. Er bleibt im
Paket als Wächter über Bestand und Renderer. Rot heißt: Umrechnung oder
Renderer falsch — erst die Differenz lesen, nie die Seiten neu committen,
um ihn grün zu machen.

- [ ] **Step 7: Run** `go test ./internal/dev/benchcorpus -run Regenerated` → PASS.

- [ ] **Step 8: Commit** (Zwischenstand, wird mit Task 3 und 4 gruppiert, siehe Task 12)

```
refactor(dev): store benchmark timings in milliseconds in the shared shape
```

---

### Task 5: `dev bench hooks|repos|search` als Gruppe

**Files:**
- Modify: `internal/cli/dev.go:49-59,155-190,455-548`
- Test: `internal/cli/dev_test.go:212-291,530-…`

**Interfaces:**
- Consumes: `benchhooks.Measure/Table`, `benchcorpus.*`, `benchreport.*`.
- Produces: `devCommands["bench"] = devBenchGroup`; `benchCommands = map[string]command{"hooks": devBenchHooks, "repos": devBenchRepos}`. `search` trägt erst Task 10 ein, samt seiner Zeile in `benchUsage` und dem `search` in der ersten Zeile der Hilfe; bis dahin nennt die Hilfe nur `<hooks|repos>`, damit kein Commit einen Befehl verspricht, der mit „unknown subcommand“ antwortet.

- [ ] **Step 1: Failing tests**

```go
func TestDevBenchWithoutSubcommandPrintsTheGroupsHelp(t *testing.T) {
	code, out, errOut := runDev(t, "bench")
	if code != 2 || out != "" {
		t.Fatalf("code=%d out=%q", code, out)
	}
	for _, want := range []string{"usage: loomux dev bench <hooks|repos>", "hooks", "repos"} {
		if !strings.Contains(errOut, want) {
			t.Fatalf("help lacks %q:\n%s", want, errOut)
		}
	}
}

func TestDevBenchHooksKeepsTheOldBehaviour(t *testing.T) { /* was TestDevBenchHooks…: argv "bench", "hooks", <case>, "-n", "2" */ }

func TestDevBenchHooksWritesBothFilesWithOut(t *testing.T) {
	dir := t.TempDir()
	code, _, _ := runDev(t, "bench", "hooks", caseFile(t), "-n", "1", "--out", dir)
	// files bench-<stamp>-hooks.md and .json exist; the JSON's "command" is "hooks"
}

func TestTheOldBenchNamesAreGone(t *testing.T) {
	for _, old := range [][]string{{"bench-hooks", "x"}, {"bench", "--dir", "."}} {
		code, _, errOut := runDev(t, old...)
		if code != 2 {
			t.Fatalf("%v: code %d, stderr %q", old, code, errOut)
		}
	}
}
```

`runDev` ist die vorhandene Hilfe in `dev_test.go`, die `devCommand` mit
Puffern ruft; heißt sie anders, die vorhandene nehmen. `benchClock` (dev.go)
ist die Naht für den Stempel.

`dev bench --dir .` scheitert, weil `--dir` kein Unterbefehl ist: Meldung
`loomux dev bench: unknown subcommand "--dir"` plus Hilfe, Exit 2.

- [ ] **Step 2: Run** `go test ./internal/cli -run DevBench` → FAIL.

- [ ] **Step 3: Implementation**

```go
var benchCommands = map[string]command{
	"hooks": devBenchHooks,
	"repos": devBenchRepos,
}

const benchUsage = `usage: loomux dev bench <hooks|repos> [flags]
  hooks   time the hook commands of a case file
  repos   time the hooks on repositories and audit their lanes
`

func devBenchGroup(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, benchUsage)
		return 2
	}
	sub, ok := benchCommands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "loomux dev bench: unknown subcommand %q\n%s", args[0], benchUsage)
		return 2
	}
	return sub(args[1:], stdin, stdout, stderr)
}
```

`devBench` heißt `devBenchRepos`, alle Meldungen `loomux dev bench repos: …`,
`--json-out` fällt weg, `benchcorpus.FormatJSON` ebenso — das JSON eines
Laufs ist jetzt die Hülle aus Task 4 („Die JSON-Ausgabe eines Laufs als
Hülle“), gebaut hier mit `benchreport.Current(Version)`, `--out DIR` schreibt über `benchreport.Targets` und
`WriteBoth` die Dateien `bench-<stempel>-repos.{md,json}`; ohne `--out` geht
das Markdown auf stdout wie heute. `devBenchHooks`: FlagSet-Name
`dev bench hooks`, Meldungen `loomux dev bench hooks: …`, neues `--out DIR`
(Dateien `bench-<stempel>-hooks.{md,json}`, Markdown = Kopf + `Table`), das
Markdown bleibt auf stdout. `Targets` läuft **vor** der ersten Messung.

Der Kopf des Markdowns für `hooks` und `repos`:

```go
func benchHead(command string, env benchreport.Environment, stamp string) string {
	return fmt.Sprintf("# loomux dev bench %s — %s\n\n- system: %s/%s, %s\n- go: %s\n- loomux: %s\n\n",
		command, stamp, env.OS, env.Arch, env.CPU, env.Go, env.Loomux)
}
```

`Environment` kommt aus `benchreport.Current(Version)`.

- [ ] **Step 4: Run** `go test ./internal/cli -cover` → PASS, 100 % für `dev.go`.

- [ ] **Step 5: Commit**

```
feat(dev)!: group the benchmarks under dev bench hooks|repos|search

BREAKING CHANGE: dev bench-hooks is now dev bench hooks and dev bench is
now dev bench repos; the old names are gone. dev bench alone prints the
group's help. Both commands take --out DIR and write a markdown and a
JSON report there, both or neither; repos drops --json-out, and its JSON
and docs/benchmarks.json carry milliseconds instead of nanoseconds.
```

---

### Task 6: qmd mit Indexnamen — `QmdPort`, `QmdConfigPathFor`, `fakeqmd`

**Files:**
- Modify: `internal/brain/search/qmd.go`, `internal/brain/index/qmdconfig.go:139-151`, `internal/dev/fakeqmd/fakeqmd.go`
- Test: `internal/brain/search/qmd_test.go`, `internal/brain/index/qmdconfig_test.go`, `internal/dev/fakeqmd/fakeqmd_test.go`

**Interfaces:**
- Produces:
  - `search.QmdPort.Index string` — nicht leer: jeder Aufruf trägt `--index <Index>` direkt hinter dem Programmnamen.
  - `index.QmdConfigPathFor(name string) string` — `<XDG_CONFIG_HOME or ~/.config>/qmd/<name>.yml`; `QmdConfigPath()` = `QmdConfigPathFor("index")`.
  - `index.QmdCacheDir() string` — `<XDG_CACHE_HOME or ~/.cache>/qmd`.
  - `fakeqmd.Fixture.Queries map[string][]Hit` (`json:"queries"`) — gibt es einen Eintrag für den Text der Anfrage, ersetzt er `Hits`.
  - `fakeqmd` nimmt `--index NAME` vor dem Unterbefehl an und hängt in der JSON-Ausgabe der CLI-Suche `?index=NAME` an `file`.

- [ ] **Step 1: Failing tests**

```go
// qmd_test.go
func TestQmdPortPutsTheIndexBehindTheProgram(t *testing.T) {
	var seen [][]string
	port := &QmdPort{Executable: "qmd", Index: "loomux-bench-x", Runner: func(argv []string) ([]byte, []byte, int, error) {
		seen = append(seen, argv)
		return []byte("[]"), nil, 0, nil
	}}
	port.Search("q", []string{"c"}, ProfileKeyword, 5)
	port.Indexed("c")
	port.Refresh(nil)
	port.Embed(nil)
	port.NotYetSearchable()
	for _, argv := range seen {
		if len(argv) < 3 || argv[1] != "--index" || argv[2] != "loomux-bench-x" {
			t.Fatalf("argv = %v", argv)
		}
	}
}

func TestQmdPortCutsTheIndexFromTheFileURI(t *testing.T) {
	out := `[{"file":"qmd://colla/baustatik-04.md?index=loomux-bench-x","docid":"#1"}]`
	port := &QmdPort{Index: "loomux-bench-x", Runner: stdoutOf(out)}
	hits, err := port.Search("q", []string{"colla"}, ProfileKeyword, 5)
	if err != nil || hits[0].Relative != "baustatik-04.md" {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
}

func TestQmdPortWithoutIndexKeepsAQuestionMarkInTheName(t *testing.T) {
	// No index configured: the path is taken as written, as before.
	out := `[{"file":"qmd://c/a?b.md","docid":"#1"}]`
	hits, _ := (&QmdPort{Runner: stdoutOf(out)}).Search("q", nil, ProfileKeyword, 5)
	if hits[0].Relative != "a?b.md" {
		t.Fatalf("relative = %q", hits[0].Relative)
	}
}
```

```go
// qmdconfig_test.go
func TestQmdConfigPathForNamesTheIndexFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/x")
	if got := QmdConfigPathFor("loomux-bench-a"); got != filepath.Join("/x", "qmd", "loomux-bench-a.yml") {
		t.Fatalf("got %q", got)
	}
	if QmdConfigPath() != QmdConfigPathFor("index") {
		t.Fatal("QmdConfigPath is not the index named index")
	}
}

func TestQmdCacheDirHonoursXDG(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/c")
	if got := QmdCacheDir(); got != filepath.Join("/c", "qmd") {
		t.Fatalf("got %q", got)
	}
}
```

```go
// fakeqmd_test.go
func TestFakeQmdAnswersAQueryFromItsOwnList(t *testing.T) {
	f := &Fixture{Hits: []Hit{{Collection: "c", Relative: "any.md", DocID: "#0"}},
		Queries: map[string][]Hit{"Eurocode": {{Collection: "c", Relative: "b.md", DocID: "#1"}}}}
	var out, errOut bytes.Buffer
	if code := f.RunCLI([]string{"search", "Eurocode", "--json", "-n", "5"}, &out, &errOut); code != 0 {
		t.Fatal(errOut.String())
	}
	if !strings.Contains(out.String(), "qmd://c/b.md") || strings.Contains(out.String(), "any.md") {
		t.Fatalf("out = %s", out.String())
	}
}

func TestFakeQmdTakesAnIndexAndMarksItsURIs(t *testing.T) {
	f := &Fixture{Hits: []Hit{{Collection: "c", Relative: "a.md", DocID: "#1"}}}
	var out, errOut bytes.Buffer
	code := f.RunCLI([]string{"--index", "n", "search", "q", "--json", "-n", "5"}, &out, &errOut)
	if code != 0 || !strings.Contains(out.String(), `"qmd://c/a.md?index=n"`) {
		t.Fatalf("code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
}

func TestFakeQmdListsWithoutTheIndexMark(t *testing.T) {
	// qmd ls does not append ?index= (measured 2026-09-26).
	f := &Fixture{Collections: map[string][]string{"c": {"a.md"}}}
	var out, errOut bytes.Buffer
	f.RunCLI([]string{"--index", "n", "ls", "c"}, &out, &errOut)
	if strings.Contains(out.String(), "?index=") {
		t.Fatalf("out = %s", out.String())
	}
}
```

- [ ] **Step 2: Run** `go test ./internal/brain/search ./internal/brain/index ./internal/dev/fakeqmd` → FAIL.

- [ ] **Step 3: Implementation.** In `qmd.go` eine Hilfe, die alle fünf Aufrufe nutzen:

```go
// command is qmd's argv for one subcommand. A named index goes right behind
// the program: qmd reads it as a global option, and it separates both the
// collection list (<name>.yml) and the index (<name>.sqlite) while the
// models stay shared.
func (q *QmdPort) command(args ...string) []string {
	exe := q.Executable
	if exe == "" {
		exe = "qmd"
	}
	argv := []string{exe}
	if q.Index != "" {
		argv = append(argv, "--index", q.Index)
	}
	return append(argv, args...)
}
```

`parseQmdJSON(stdout []byte, index string)`: nach dem `Cut` bei gesetztem
`index` `rel = strings.TrimSuffix(rel, "?index="+index)`. Nur dieses Suffix,
damit ein `?` im Dateinamen ohne Index unangetastet bleibt.

In `qmdconfig.go` `QmdConfigPathFor` und `QmdCacheDir` nach dem Muster von
`QmdConfigPath` (`XDG_CACHE_HOME`, sonst `~/.cache`). qmd 2.8.3 beachtet
`XDG_CACHE_HOME` auch unter Windows (gemessen 2026-09-26).

In `fakeqmd.go`: `RunCLI` schneidet ein führendes `--index NAME` ab und merkt
es sich im Aufruf (Parameter an `search`); `search` nimmt
`f.Queries[args[0]]`, wenn vorhanden, sonst `f.Hits`, filtert wie `hitsIn`
und hängt bei gesetztem Index `"?index=" + name` an `File`. `MCPHandler`
bekommt dieselbe Auswahl über `Queries` (Text der Anfrage aus `searches[0].query`
bzw. `query`), aber kein `?index=`.

- [ ] **Step 4: Run** → PASS, Coverage 100 % der geänderten Funktionen.

- [ ] **Step 5: Commit**

```
feat(search): let the qmd port and config address a named index

qmd --index <name> keeps a collection list and an index of its own while
sharing the models. The CLI port passes it on every call and cuts the
?index= suffix qmd appends to a search hit's file; the fake qmd accepts
the option and can answer each query text with hits of its own.
```

---

### Task 7: Fragensatz und Korpusprüfung, dazu der Korpus `v1`

**Files:**
- Create: `internal/dev/benchsearch/questions.go`, `corpus.go`
- Create: `testdata/bench/search/v1/` (106 Dateien aus der Referenz), `testdata/bench/search/v1/LICENSE.md`
- Modify: `.gitattributes`
- Test: `internal/dev/benchsearch/questions_test.go`, `corpus_test.go`

**Interfaces:**
- Produces:

```go
package benchsearch

type Kind string
const (
	Exact        Kind = "exakt"
	Paraphrase   Kind = "umschreibung"
	Mixed        Kind = "gemischt"
	CrossLingual Kind = "sprachuebergreifend"
)
var kinds = [...]Kind{Exact, Paraphrase, Mixed, CrossLingual} // order of every table

type Shape map[Kind]int
func DefaultShape() Shape // 13/13/10/14

type Question struct {
	ID, Query, Evidence, Note string
	Kind   Kind
	Expect string // absolute, cleaned
}

type Problems []string          // error listing every problem, one per line
func (p Problems) Error() string

func LoadQuestions(path string, shape Shape) ([]Question, error) // error is Problems or I/O
func CheckCorpus(stand string) error                               // error is Problems
```

- [ ] **Step 1: Daten holen und prüfen** (vor jedem Code, damit die Tests echte Daten lesen können):

```bash
git -C "C:/Users/micro/Documents/#GIT/ultra-brain" -c core.autocrlf=false archive loomux-3-source bench/corpus/v1 | tar -x -C "<scratchpad>/corpus"
```

`-c core.autocrlf=false` ist Pflicht: in beiden Repos steht
`core.autocrlf=true`, und `git archive` wendet die Umwandlung des
Arbeitsbaums an. Dann nach `testdata/bench/search/v1/` kopieren (ohne das
Präfix `bench/corpus/v1`). Zählen: 106 Dateien, 330 246 Bytes (`find … -type f | wc -l`,
`du -b`). Jede Prüfsumme gegen `manifest.json` rechnen (`sha256sum notes/*.md`
gegen `jq -r 'to_entries[] | "\(.value.sha256)  notes/\(.key)"'`), 100 von 100
gleich. `.gitattributes` bekommt `testdata/bench/search/** -text` **vor** dem
`git add`, dann `git add --renormalize` unnötig; nach dem Commit
`git ls-files --eol testdata/bench/search/v1/notes | grep -v "i/lf"` leer.

`LICENSE.md` (englisch, kurz): Die Texte unter `notes/` sind wörtliche
Abschnitte aus der deutschsprachigen Wikipedia, abgerufen am 2026-08-21,
lizenziert unter CC BY-SA 4.0 (`https://creativecommons.org/licenses/by-sa/4.0/`);
die Quelle jeder Notiz steht in `HERKUNFT.md` und `manifest.json`; die Lizenz
von loomux gilt für dieses Verzeichnis nicht.

- [ ] **Step 2: Failing tests** (`questions_test.go`)

```go
func TestLoadQuestionsReadsTheCorpusSet(t *testing.T) {
	qs, err := LoadQuestions(filepath.Join(corpusDir(t), "questions.yaml"), DefaultShape())
	if err != nil {
		t.Fatal(err)
	}
	if len(qs) != 50 || qs[0].ID != "c01" || qs[0].Kind != Exact {
		t.Fatalf("got %d, first %+v", len(qs), qs[0])
	}
	if !filepath.IsAbs(qs[0].Expect) || filepath.Base(qs[0].Expect) != "baustatik-04.md" {
		t.Fatalf("expect = %q", qs[0].Expect)
	}
}

func TestLoadQuestionsCollectsEveryProblem(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("Es gibt\ninsgesamt 58 Teilnormen."), 0o644)
	writeYAML(t, dir, `
- {id: q1, sort: exakt, query: a, expect: a.md, beleg: "Es gibt insgesamt 58 Teilnormen."}
- {id: q1, sort: exakt, query: b, expect: a.md, beleg: "fehlt"}
- {id: q3, sort: raten, query: c, expect: a.md, beleg: "Es gibt"}
- {id: q4, query: d, expect: nope.md, beleg: x}
- {id: q5, sort: exakt, query: e, expect: nope.md, beleg: x}
`)
	_, err := LoadQuestions(filepath.Join(dir, "questions.yaml"), Shape{Exact: 1})
	var p Problems
	if !errors.As(err, &p) {
		t.Fatalf("err = %v", err)
	}
	want := []string{
		"q1: duplicate id \"q1\"",
		"q1: evidence not found verbatim in " + filepath.Join(dir, "a.md"),
		"q3: unknown sort \"raten\"",
		"q4: missing field \"sort\"",
		"q5: expect does not exist: " + filepath.Join(dir, "nope.md"),
		"exakt: 3 questions, expected 1",
	}
	if !slices.Equal(p, want) {
		t.Fatalf("problems:\n%s", strings.Join(p, "\n"))
	}
}

func TestLoadQuestionsNamesBrokenYAML(t *testing.T) { /* "- [" → ["questions.yaml is not valid YAML: …"] */ }
func TestLoadQuestionsWantsAList(t *testing.T)     { /* "a: 1" → "the question set must be a list of entries, found map" */ }
func TestLoadQuestionsWantsMappings(t *testing.T)  { /* "- 3" → "entry 1: must be a mapping, found int" */ }
func TestHinweisIsOptional(t *testing.T)           { /* ohne hinweis: Note == "" und kein Problem */ }
```

Die Reihenfolge der Befunde ist die der Referenz: je Eintrag in Dateifolge
(fehlende Felder, doppelte ID, Ziel und Beleg, dann Sorte), zuletzt die Form je
Sorte in der Folge von `kinds`. Die Typnamen in Meldungen folgen Go
(`map`, `int`, `string`, `[]interface {}` wird als `list` gemeldet — die
Hilfe `typeName(v any) string` bildet `map[string]any → "map"`,
`[]any → "list"`, sonst `fmt.Sprintf("%T")`). Hier ist `any` unvermeidlich:
`yaml.v3` dekodiert in `any`.

- [ ] **Step 3: Run** → FAIL.

- [ ] **Step 4: `questions.go`** (Übertragung von `questions.py`)

```go
// collapsed joins the words with single spaces. strings.Fields splits at
// every Unicode space, the no-break space of a Wikipedia excerpt included,
// which is what Python's \s does in a str pattern and Go's regexp \s does not.
func collapsed(s string) string { return strings.Join(strings.Fields(s), " ") }

func LoadQuestions(path string, shape Shape) ([]Question, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, Problems{fmt.Sprintf("%s is not valid YAML: %v", filepath.Base(path), err)}
	}
	entries, ok := raw.([]any)
	if !ok {
		return nil, Problems{fmt.Sprintf("the question set must be a list of entries, found %s", typeName(raw))}
	}
	var problems Problems
	var questions []Question
	seen := map[string]bool{}
	base := filepath.Dir(path)
	for i, entry := range entries {
		if q, ok := readQuestion(entry, i+1, seen, &problems, base); ok {
			questions = append(questions, q)
		}
	}
	problems = append(problems, shapeProblems(questions, shape)...)
	if len(problems) > 0 {
		return nil, problems
	}
	return questions, nil
}
```

**Übertragungsfalle `\s`:** Python-`\s` ist in `str`-Mustern Unicode, das von
Go ASCII. Wikipedia-Auszüge tragen geschützte Leerzeichen (U+00A0) etwa
zwischen Zahl und Einheit; mit `regexp` `\s+` fände Go den Beleg nicht.
`collapsed` ist deshalb `strings.Join(strings.Fields(s), " ")`
(`unicode.IsSpace` kennt U+00A0), nicht die Regex oben. Test:
`TestEvidenceMatchesAcrossANoBreakSpace` — Ziel `2 kN/m³`, Beleg
`2 kN/m³` → kein Problem.

`readQuestion` folgt `_entry` Zeile für Zeile: Pflichtfelder `id, sort,
query, expect, beleg` in dieser Reihenfolge, Name `entry N` ohne `id`;
doppelte ID; `expect := filepath.Clean(filepath.Join(base, fmt.Sprint(fields["expect"])))`;
Zielprüfung (`expect does not exist: …` oder
`evidence not found verbatim in …` nach `collapsed`); dann die Sorte.
Werte werden mit `fmt.Sprint` zu Text wie `str(...)` in der Referenz.

`shapeProblems` zählt je Sorte und meldet in der Folge von `kinds`
`<sorte>: <n> questions, expected <m>` für jede Sorte der Form mit
abweichender Zahl.

- [ ] **Step 5: Failing tests** (`corpus_test.go`)

```go
func TestTheCheckedInCorpusPasses(t *testing.T) {
	if err := CheckCorpus(corpusDir(t)); err != nil {
		t.Fatal(err)
	}
}

func TestCheckCorpusFindsAChangedByte(t *testing.T) {
	stand := copyCorpus(t) // t.TempDir() copy of testdata/bench/search/v1
	note := filepath.Join(stand, "notes", "baustatik-01.md")
	data, _ := os.ReadFile(note)
	os.WriteFile(note, bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n")), 0o644)
	var p Problems
	if !errors.As(CheckCorpus(stand), &p) || !strings.Contains(p[0], "baustatik-01.md: checksum ") {
		t.Fatalf("problems = %v", p)
	}
}

func TestCheckCorpusStopsOnlyForFilesItReads(t *testing.T) {
	stand := copyCorpus(t)
	os.Remove(filepath.Join(stand, "HERKUNFT.md"))
	var p Problems
	errors.As(CheckCorpus(stand), &p)
	if !slices.Contains(p, "HERKUNFT.md is missing") || len(p) != 1 {
		t.Fatalf("problems = %v", p)
	}
	os.Remove(filepath.Join(stand, "manifest.json"))
	errors.As(CheckCorpus(stand), &p)
	if !slices.Equal(p, Problems{"manifest.json is missing"}) {
		t.Fatalf("problems = %v", p)
	}
}

func TestCheckCorpusHoldsHerkunftBothWays(t *testing.T)   { /* Name gestrichen → "x.md: not named in HERKUNFT.md"; Name "geist.md" ergänzt → "geist.md: named in HERKUNFT.md but not among the notes" */ }
func TestCheckCorpusHoldsThePartition(t *testing.T)       { /* Notiz in zwei Themen, Thema mit 9, Nachbar = eigenes Thema, Nachbar unbekannt */ }
func TestCheckCorpusWantsFiveReverseQuestions(t *testing.T) { /* vier englische sprachuebergreifend-Fragen ins Deutsche umschreiben → "4 questions in the reverse direction, expected at least 5" */ }
func TestIsReverseCountsFunctionWords(t *testing.T) {
	cases := map[string]bool{
		"What does the Eurocode say about snow?": true,
		"Was sagt der Eurocode zum Schnee?":      false,
		"in is":                                  false, // both languages: counted for neither
	}
	for query, want := range cases {
		if got := isReverse(Question{Kind: CrossLingual, Query: query}); got != want {
			t.Errorf("%q: %v", query, got)
		}
	}
	if isReverse(Question{Kind: Exact, Query: "What does the"}) {
		t.Error("only cross-lingual questions can be reverse")
	}
}
```

- [ ] **Step 6: `corpus.go`** (Übertragung von `corpus.py`)

Konstanten `notesWanted = 100`, `themesWanted = 10`, `reverseMinimum = 5`;
die Wortlisten wörtlich aus `corpus.py:43-81` als `map[string]bool` in einer
Funktion (`englishWords()`, `germanWords()`), nicht als Paketvariable mit
Literal — ein Literal ist erlaubt, die Regel gilt nur für eingebettete
Daten; trotzdem als Funktion, damit kein Test die Karte verändern kann.
`named := regexp.MustCompile(`[\p{L}\p{N}_-]+\.md`)` — Pythons `\w` ist
Unicode, Gos nicht —, `word := regexp.MustCompile(`[a-zA-Zäöüß]+`)` (in beiden
Sprachen wörtlich, bleibt).

Ablauf wie `check`: erst die vier gelesenen Dateien (`notes`,
`themes.yaml`, `questions.yaml`, `manifest.json`) — fehlt eine, sofort
`Problems` nur mit diesen; dann Zahl der Notizen, `HERKUNFT.md`, Manifest
(Pflichtfelder `sha256, quelle, abgerufen, lizenz`, leere Felder,
Prüfsumme über die Rohbytes, Manifestschlüssel ohne Notiz), Themen
(`name, nachbar, notizen`; Zahl 10 je Thema; eigener Nachbar; unbekannter
Nachbar; unbekannte Notiz; doppelt und gar nicht beansprucht), Fragen
(`LoadQuestions(…, DefaultShape())`, dann `isReverse` ≥ 5). `earlier` der
Referenz entfällt — die CLI reicht es nie weiter (Spec-Nachtrag).

- [ ] **Step 7: Run** `go test ./internal/dev/benchsearch -cover` → PASS, 100 %.

- [ ] **Step 8: Commit**

```
feat(dev): check a search question set and the corpus v1

The question set and the rules of a corpus stand, ported from brain
bench: every problem is collected before anything runs. The corpus v1 is
a hundred verbatim German Wikipedia excerpts under CC BY-SA 4.0 with
their provenance, checksums and fifty questions; git keeps its bytes as
they are, since the checksums hold the raw files.
```

---

### Task 8: Qualität, Latenz und der Bericht von `search`

**Files:**
- Create: `internal/dev/benchsearch/quality.go`, `latency.go`, `report.go`
- Test: `quality_test.go`, `latency_test.go`, `report_test.go`

**Interfaces:**
- Consumes: `Question`, `Kind`, `kinds`; `benchreport.Summarize`, `Timing`, `Report`, `Environment`.
- Produces:

```go
const top = 3

type Ask func(query string) ([]string, error) // ranked absolute paths
type Outcome struct {
	Question  Question
	Rank      int // 0: not found
	Hit       bool
	ElapsedMS float64
}
func RunQuality(qs []Question, ask Ask, clock func() time.Time) ([]Outcome, error)

type Operation struct {
	Name string
	Call func() error
}
func MeasureLatency(ops []Operation, repeat int, clock func() time.Time) ([]benchreport.Timing, error)

type Latency struct {
	Document, Query string
	Timings         []benchreport.Timing
}
type Run struct {
	Stamp       string
	Profile     string
	Environment benchreport.Environment
	Documents   int
	QuestionSet string
	Corpus      string // "" outside corpus mode
	Outcomes    []Outcome
	Findings    []string
	Latency     *Latency
}
func Markdown(r Run) string
func Envelope(r Run) (benchreport.Report, error)
```

- [ ] **Step 1: Failing tests** (`quality_test.go`)

```go
func TestRankIsThePositionOfTheExpectedFile(t *testing.T) {
	dir := t.TempDir()
	a, b := touch(t, dir, "a.md"), touch(t, dir, "b.md")
	qs := []Question{{ID: "1", Kind: Exact, Expect: b}, {ID: "2", Kind: Exact, Expect: a}}
	answers := map[string][]string{"": {a, b}}
	outcomes, err := RunQuality(qs, func(string) ([]string, error) { return answers[""], nil }, tick(5*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if outcomes[0].Rank != 2 || !outcomes[0].Hit || outcomes[1].Rank != 1 || outcomes[0].ElapsedMS != 5 {
		t.Fatalf("%+v", outcomes)
	}
}

func TestAMissBeyondTheTopThreeKeepsItsRank(t *testing.T) { /* expected at 4 → Rank 4, Hit false */ }
func TestNotFoundIsRankZero(t *testing.T)                 { /* → Rank 0, Hit false */ }

func TestRankMatchesAcrossSpelling(t *testing.T) {
	dir := t.TempDir()
	a := touch(t, dir, "Note.md")
	spelled := filepath.Join(dir, "sub", "..", "Note.md")
	if runtime.GOOS == "windows" {
		spelled = strings.ToLower(spelled)
	}
	out, _ := RunQuality([]Question{{Kind: Exact, Expect: spelled}},
		func(string) ([]string, error) { return []string{a}, nil }, tick(0))
	if out[0].Rank != 1 {
		t.Fatalf("rank = %d", out[0].Rank)
	}
}

// Through a junction the two spellings share no prefix; only the file
// system can tell they are one file (8.3 names on the runner likewise).
func TestRankMatchesThroughAJunction(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("junctions are a Windows form")
	}
	real := t.TempDir()
	a := touch(t, real, "a.md")
	link := filepath.Join(t.TempDir(), "j")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, real).CombinedOutput(); err != nil {
		t.Fatalf("mklink: %v %s", err, out)
	}
	out, _ := RunQuality([]Question{{Kind: Exact, Expect: filepath.Join(link, "a.md")}},
		func(string) ([]string, error) { return []string{a}, nil }, tick(0))
	if out[0].Rank != 1 {
		t.Fatalf("rank = %d", out[0].Rank)
	}
}

func TestASearchErrorEndsTheRun(t *testing.T) { /* ask returns error → RunQuality returns it, no outcomes */ }
```

`sameFile(a, b string) bool`: erst der billige Vergleich —
`filepath.Clean` beider, unter Windows mit `strings.EqualFold` —, nur bei
Ungleichheit `os.Stat` auf beide und `os.SameFile` (ein fehlender Treffer
ist dann einfach ungleich). Das ist `normcase(resolve())` der Referenz plus
das, was Kurznamen und Junctions verlangen (Lernliste des Nutzers, Eintrag
„Zwei Pfade nach Schreibweise verglichen“). Der Test über die Junction ist
lokal rot, wenn `SameFile` fehlt.

- [ ] **Step 2: Failing tests** (`latency_test.go`)

```go
func TestMeasureLatencyRunsColdThenRepeatWarm(t *testing.T) {
	calls := 0
	ops := []Operation{{Name: "keyword", Call: func() error { calls++; return nil }}}
	got, err := MeasureLatency(ops, 3, tick(2*time.Millisecond))
	if err != nil || calls != 4 || got[0].ColdMS != 2 || len(got[0].WarmMS) != 3 {
		t.Fatalf("got %+v calls=%d err=%v", got, calls, err)
	}
}

func TestMeasureLatencyRefusesRepeatBelowOne(t *testing.T) {
	if _, err := MeasureLatency(nil, 0, tick(0)); err == nil || err.Error() != "repeat must be at least 1, got 0" {
		t.Fatalf("err = %v", err)
	}
}

func TestMeasureLatencyStopsAtAFailingOperation(t *testing.T) { /* Call returns error → returned */ }
```

- [ ] **Step 3: Failing tests** (`report_test.go`) — goldene Texte

```go
func TestMarkdownOfAnEverydayRun(t *testing.T) {
	got := Markdown(sampleRun(false))
	want := readGolden(t, "testdata/report-everyday.md")
	if got != want {
		t.Fatalf("diff:\n%s", diffLines(want, got))
	}
}

func TestMarkdownOfACorpusRunCarriesTheCaveat(t *testing.T) { /* golden report-corpus.md */ }

func TestEnvelopeCarriesTheQuestionsAndTheLatencyTarget(t *testing.T) {
	env, err := Envelope(sampleRun(true))
	if err != nil {
		t.Fatal(err)
	}
	if env.Command != "search" || env.Schema != 1 || len(env.Timings) != 5 {
		t.Fatalf("%+v", env)
	}
	var payload struct {
		Questions []struct {
			ID   string `json:"id"`
			Rank *int   `json:"rank"`
		} `json:"questions"`
		Latency *struct{ Document, Query string } `json:"latency"`
	}
	json.Unmarshal(env.Payload, &payload)
	if payload.Questions[1].Rank != nil { // the miss
		t.Fatalf("a miss must carry rank null")
	}
}
```

`sampleRun` baut einen `Run` mit zwei Fragen (ein Treffer auf Rang 1, ein
Fehlschlag), einem Befund und optional Latenz. Die Goldens schreibt der
Implementierer einmal von Hand nach dem Format unten, nicht aus der Ausgabe
kopiert, und liest sie gegen.

- [ ] **Step 4: Run** → FAIL.

- [ ] **Step 5: Implementation**

`quality.go`:

```go
// RunQuality asks every question in the set's order and measures one rank
// each, never a score: a high score is no proof of truth. A search error
// ends the run, so a partial run cannot pass for a whole one.
func RunQuality(qs []Question, ask Ask, clock func() time.Time) ([]Outcome, error) {
	outcomes := make([]Outcome, 0, len(qs))
	for _, q := range qs {
		started := clock()
		ranked, err := ask(q.Query)
		if err != nil {
			return nil, err
		}
		elapsed := benchreport.MS(clock().Sub(started))
		rank := rankOf(q.Expect, ranked)
		outcomes = append(outcomes, Outcome{Question: q, Rank: rank, Hit: rank > 0 && rank <= top, ElapsedMS: elapsed})
	}
	return outcomes, nil
}
```

`latency.go`:

```go
// MeasureLatency times every operation once cold and repeat times warm.
// The cold run stands apart: folded into the median it would hide both.
func MeasureLatency(ops []Operation, repeat int, clock func() time.Time) ([]benchreport.Timing, error) {
	if repeat < 1 {
		return nil, fmt.Errorf("repeat must be at least 1, got %d", repeat)
	}
	timings := make([]benchreport.Timing, 0, len(ops))
	for _, op := range ops {
		cold, err := once(op, clock)
		if err != nil {
			return nil, err
		}
		warm := make([]float64, 0, repeat)
		for range repeat {
			ms, err := once(op, clock)
			if err != nil {
				return nil, err
			}
			warm = append(warm, ms)
		}
		timings = append(timings, benchreport.Summarize(op.Name, cold, warm))
	}
	return timings, nil
}
```

`report.go` — das Markdown (englisch, E2):

```
# Search bench <stamp> — profile `<profile>`

- qmd: <env.Qmd>
- models: embedding=…, query_expansion=…, rerank=…
- system: <os>/<arch>, <cpu>
- loomux: <version>
- search path: <daemon|cli>
- indexed documents: <n>
- question set: <path>
- corpus: <stand>                         (nur im Korpusmodus)

<corpus caveat>                           (nur im Korpusmodus)
<fast caveat>                             (nur bei profile fast)

## Hit quality

| Sort | Hits |
|---|---|
| exakt | 13/13 |
| umschreibung | 11/13 |
| gemischt | 8/10 |
| sprachuebergreifend | 11/14 |
| total | 43/50 |

Median hits: 12 ms
Median misses: —

## Misses

- c07 (umschreibung): rank 5 — <query>
- c19 (gemischt): not found — <query>
(oder: none)

## Findings

Messages of the search chain during the run. They belong beside the
numbers: a miss with a finding may be no measurement at all but a silent
failure of the engine.

- <finding>
(oder: none)

## Latency                                (nur mit --latency)

"Warm" here means the chain has already searched in this run: the latency
runs after the quality pass.

- document read: `<scope>/<relative>`
- query: `<query>`

| Operation | cold | warm median | min | max |
|---|---|---|---|---|
| catalog | 1 ms | … |
```

Die drei Hinweise sinngemäß aus `report.py` übersetzt:
- Korpus: „These numbers measure **regression** against an artificial stock. They say whether the chain got worse than at the last stand — and they can carry **no decision**, neither about the architecture nor about a model. That is what the real question set is for.“
- `fast`: „This run is **purely vectorial** (`qmd vsearch`) and therefore measures **both** the embedding model's share of the language bridge and whether `fast` carries across languages.“
- Die Referenz nennt §16.3 und die Scheibe 2c; das entfällt, beides sind Papiere der Referenz.

Zeiten in der Tabelle der Latenz als ganze Millisekunden (`%.0f ms`) wie
die Referenz; `Median hits/misses` ebenso, „—“ für eine leere Gruppe.

`Envelope` baut `benchreport.Report{Schema: 1, Command: "search", Stamp,
Environment, Timings: latency timings oder leer}` und die Nutzlast:

```go
type payload struct {
	QuestionSet string           `json:"question_set"`
	Corpus      *string          `json:"corpus"`   // null outside corpus mode
	Documents   int              `json:"documents"`
	Questions   []questionRecord `json:"questions"`
	Findings    []string         `json:"findings"` // [] never null
	Latency     *latencyTarget   `json:"latency"`  // null without --latency
}
type questionRecord struct {
	ID        string  `json:"id"`
	Sort      Kind    `json:"sort"`
	Rank      *int    `json:"rank"` // null: not found
	Hit       bool    `json:"hit"`
	ElapsedMS float64 `json:"elapsed_ms"`
}
type latencyTarget struct {
	Document string `json:"document"`
	Query    string `json:"query"`
}
```

- [ ] **Step 6: Run** `go test ./internal/dev/benchsearch -cover` → PASS, 100 %.

- [ ] **Step 7: Commit**

```
feat(dev): measure hit rank and latency of the search chain

Ported from brain bench: the rank of the expected source among the first
ten, a hit at rank three or better, a cold run and repeated warm runs per
operation, and a report in markdown and in the shared shape.
```

---

### Task 9: Der Korpusmodus — Wegwerf-Zustand, benannter Index, Sperre

**Files:**
- Create: `internal/dev/benchsearch/corpusrun.go`
- Test: `internal/dev/benchsearch/corpusrun_test.go`

**Interfaces:**
- Consumes: `index.QmdConfigPathFor`, `index.QmdCacheDir`, `index.SyncCollections`, `index.DropCollections`, `index.Models`, `index.QmdConfigPath`, `index.AlwaysExcludes`, `identity.*`, `lock.TryAcquire`, `lock.Acquire`, `search.CollectionName`.
- Produces:

```go
const CorpusScope = "loomux-bench-corpus"
const indexPrefix = "loomux-bench-"

type Prepared struct {
	Stand    string // absolute
	StateDir string // throwaway registry dir
	Index    string // qmd index name
	Scope    string // CorpusScope
}

// PrepareCorpus takes the lock, writes the throwaway state and the named
// qmd config. The returned cleanup is always safe to call, also after an
// error, and reports what it could not remove through warn.
func PrepareCorpus(stand, lockDir string, random func() string, warn func(string)) (*Prepared, func(), error)

// SweepStale removes a loomux-bench-* index nobody holds a lock on.
func SweepStale(lockDir string, warn func(string))
```

- [ ] **Step 1: Failing tests**

```go
func isolate(t *testing.T) (lockDir string) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	os.MkdirAll(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "qmd"), 0o755)
	os.MkdirAll(index.QmdCacheDir(), 0o755)
	os.WriteFile(index.QmdConfigPath(), []byte("models:\n  embed: E\n  generate: G\n  rerank: R\n"), 0o644)
	return t.TempDir()
}

func TestPrepareCorpusWritesStateAndANamedConfig(t *testing.T) {
	lockDir := isolate(t)
	p, cleanup, err := PrepareCorpus(corpusDir(t), lockDir, fixed("a1"), t.Log)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if p.Index != "loomux-bench-a1" || p.Scope != CorpusScope {
		t.Fatalf("%+v", p)
	}
	areas, _ := config.ReadRegistry(p.StateDir)
	if len(areas) != 1 || !areas[0].ReadOnly || areas[0].Path != filepath.ToSlash(filepath.Join(p.Stand, "notes")) {
		t.Fatalf("areas = %+v", areas)
	}
	reg, _ := identity.ReadIdentities(filepath.Join(config.ManifestDir(areas[0], p.StateDir), "_identities.tsv"))
	if len(reg) != 100 {
		t.Fatalf("register holds %d", len(reg))
	}
	if m := index.Models(index.QmdConfigPathFor(p.Index)); m["embedding"] != "E" || m["rerank"] != "R" {
		t.Fatalf("models = %v", m)
	}
	real, _ := os.ReadFile(index.QmdConfigPath())
	if strings.Contains(string(real), CorpusScope) {
		t.Fatal("the real index.yml was touched")
	}
}

func TestCleanupRemovesEverythingOfTheRun(t *testing.T) {
	lockDir := isolate(t)
	p, cleanup, _ := PrepareCorpus(corpusDir(t), lockDir, fixed("a1"), t.Log)
	for _, suffix := range []string{".sqlite", ".sqlite-wal", ".sqlite-shm"} {
		os.WriteFile(filepath.Join(index.QmdCacheDir(), p.Index+suffix), nil, 0o644)
	}
	cleanup()
	for _, gone := range []string{p.StateDir, index.QmdConfigPathFor(p.Index), index.QmdConfigPathFor(p.Index) + ".brain-backup",
		filepath.Join(index.QmdCacheDir(), p.Index+".sqlite"), filepath.Join(lockDir, p.Index+".lock")} {
		if _, err := os.Stat(gone); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s left behind", gone)
		}
	}
}

func TestSweepStaleLeavesAHeldIndexAlone(t *testing.T) {
	lockDir := isolate(t)
	held := filepath.Join(lockDir, "loomux-bench-a.lock")
	h, _ := lock.Acquire(held)
	defer h.Release()
	os.WriteFile(index.QmdConfigPathFor("loomux-bench-a"), nil, 0o644)
	os.WriteFile(index.QmdConfigPathFor("loomux-bench-b"), nil, 0o644)
	os.WriteFile(filepath.Join(index.QmdCacheDir(), "loomux-bench-b.sqlite"), nil, 0o644)
	SweepStale(lockDir, t.Log)
	if _, err := os.Stat(index.QmdConfigPathFor("loomux-bench-a")); err != nil {
		t.Fatal("a held index was swept")
	}
	if _, err := os.Stat(index.QmdConfigPathFor("loomux-bench-b")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("a stale index stayed")
	}
}

func TestSweepStaleNeverTouchesTheUsersOwnIndexes(t *testing.T) {
	lockDir := isolate(t)
	SweepStale(lockDir, t.Log)
	if _, err := os.Stat(index.QmdConfigPath()); err != nil {
		t.Fatal("index.yml removed")
	}
}

func TestAFailedPreparationCleansUpToo(t *testing.T) {
	lockDir := isolate(t)
	// a stand whose notes directory is a file: writing the register fails
	stand := t.TempDir()
	os.WriteFile(filepath.Join(stand, "notes"), nil, 0o644)
	_, cleanup, err := PrepareCorpus(stand, lockDir, fixed("a1"), t.Log)
	if err == nil {
		t.Fatal("no error")
	}
	cleanup()
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(index.QmdConfigPath()), "loomux-bench-*"))
	if len(matches) != 0 {
		t.Fatalf("left behind: %v", matches)
	}
}
```

- [ ] **Step 2: Run** → FAIL.

- [ ] **Step 3: Implementation.** `PrepareCorpus`:

1. `name := indexPrefix + random()`; Sperre `lock.Acquire(filepath.Join(lockDir, name+".lock"))` (vorher `os.MkdirAll(lockDir)`).
2. `state := filepath.Join(lockDir, name)`, angelegt mit `os.MkdirAll` — nicht in TEMP: Go führt bei Ctrl+C kein `defer` aus, und so räumt der nächste `SweepStale` den Zustand zusammen mit dem Index weg (er löscht zu jedem freien Namen auch `<lockDir>/<name>/`).
3. `cleanup` sofort bauen (alles folgende darf scheitern): `os.RemoveAll(state)`, `removeIndex(name)` (die Dateien `<cfg>/qmd/<name>.yml`, `<name>.yml.brain-backup`, `<cache>/qmd/<name>.sqlite`, `-wal`, `-shm`; `fs.ErrNotExist` ist kein Fehler, jeder andere eine Warnung), Sperre freigeben, Sperrdatei löschen.
4. `registry.toml` im Zustand, LF, `readonly = true`, `path` mit Vorwärtsstrichen (wie die Referenz, B6):

```go
reg := fmt.Sprintf("[[area]]\nscope = %q\npath = %q\nreadonly = true\n", CorpusScope, filepath.ToSlash(notes))
```

5. Bereichszustand `config.ManifestDir(config.Area{Scope: CorpusScope, ReadOnly: true}, state)`; darin `.loomux/config.toml`:

```toml
[area]
scope = "loomux-bench-corpus"

[index]
include = ["**/*.md"]
```

Vor dem Schreiben gegen den echten Leser prüfen: der Test oben liest den
Bereich über `privacy.VisibleAreas(state, "", CorpusScope, privacy.ChannelLocal)`
— schlägt das fehl, fehlt ein Pflichtschlüssel des Schemas, und die
Deklaration wird ergänzt, nicht der Leser.

6. Register: je Notiz (sortiert) `identity.Identity{DocID: identity.NewDocID(), Relative: name, ContentHash: hash, Revision: 1}`, geschrieben mit `identity.RenderIdentities`.
7. Die Modelle: `index.Models(index.QmdConfigPath())` → `<name>.yml` mit
   `models:` (`embed`, `generate`, `rerank`; ein `unknown` lässt den
   Schlüssel weg, dann nimmt qmd seine Vorgabe und `environment.models` sagt
   es).
8. `index.SyncCollections(index.QmdConfigPathFor(name), map[string]index.CollectionSpec{search.CollectionName(CorpusScope): {Path: notes, Pattern: "**/*.md", Ignore: index.AlwaysExcludes}}, index.OwnershipRecord{Read: owned, Write: owned})` mit `owned := filepath.Join(state, "qmd-owned.txt")`.

**qmds Embed-Sperre** (gelesen 2026-09-26 in
`@tobilu/qmd/dist/cli/embed-lock.js`, qmd 2.8.3): `.qmd-embed.lock` liegt neben
der Datenbank, mit `--index` also im **geteilten** `~/.cache/qmd`, und hält
die PID des Halters; eine verwaiste Sperre räumt qmd selbst per PID-Prüfung
weg. Folgen: ein Korpuslauf und ein `qmd embed` des Nutzers schließen sich
aus — das `Embed` des Korpuslaufs scheitert dann mit qmds Meldung, und der
Lauf endet mit Exit 1, was so bleibt. Weder `cleanup` noch `SweepStale`
fassen `.qmd-embed.lock` je an.

`SweepStale`: `filepath.Glob(<cfg>/qmd/loomux-bench-*.yml)`, je Name
`lock.TryAcquire(<lockDir>/<name>.lock)`; frei → `removeIndex(name)`, freigeben,
Sperrdatei löschen; gehalten → überspringen. Ebenso verwaiste
`<cache>/qmd/loomux-bench-*.sqlite` ohne `.yml`.

- [ ] **Step 4: Run** `go test ./internal/dev/benchsearch -cover` → PASS, 100 %.

- [ ] **Step 5: Commit**

```
feat(dev): prepare the corpus in a throwaway state and a named qmd index

The corpus is registered read-only in a temporary state with a fresh
register and searched in a qmd index of its own, which copies the user's
model choice. A lock per index name lets a later run sweep what a crashed
one left, and never what a running one holds.
```

---

### Task 10: Der Lauf und `loomux dev bench search`

**Files:**
- Create: `internal/dev/benchsearch/run.go`
- Modify: `internal/cli/dev.go` (`benchCommands["search"]`, `devBenchSearch`)
- Test: `internal/dev/benchsearch/run_test.go`, `internal/cli/dev_test.go`

**Interfaces:**
- Consumes: alles aus Task 6–9; `search.ExecuteSearch`, `answer.RunWith`, `privacy.Channel`.
- Produces:

```go
type Options struct {
	Scope        string // "" = default "knowledge"
	ScopeSet     bool   // --scope given
	Profile      search.Profile
	Channel      privacy.Channel
	Out          string // "" = default
	Questions    string // "" = default
	Corpus       string // "" or a stand directory
	Latency      bool
	LatencyQuery string
	Repeat       int
}

type Deps struct {
	StateDir, FallbackDir string            // the real state
	Daemon  func() search.SearchPort         // everyday: the service
	CLI     func(index string) search.SearchPort // corpus: QmdPort with --index
	QmdVersion func() string
	Models  func(configPath string) map[string]string // index.Models; a seam, so no test reads the machine's index.yml
	Loomux  string
	Now     func() time.Time
	Clock   func() time.Time                 // for timings
	Random  func() string
	Warn    func(string)
}

// Bench runs one measurement and returns the markdown printed on success.
// Every refusal is a Problems or a plain error; the caller prints each line
// as "error: <line>" and exits 1.
func Bench(o Options, d Deps) (string, error)
```

- [ ] **Step 1: Failing tests** (`run_test.go`, gegen `search.FakePort` bzw. `fakeqmd` über `QmdPort{Runner: …}`)

```go
func TestCorpusRefusesScopeQuestionsAndLatency(t *testing.T) {
	for name, o := range map[string]Options{
		"scope":     {Corpus: "x", ScopeSet: true, Out: "o"},
		"questions": {Corpus: "x", Questions: "q", Out: "o"},
		"latency":   {Corpus: "x", Latency: true, Out: "o"},
		"no out":    {Corpus: "x"},
	} {
		if _, err := Bench(o, Deps{}); err == nil {
			t.Errorf("%s: no refusal", name)
		}
	}
}

func TestAnUnsoundCorpusWritesNothing(t *testing.T) {
	lockDir := isolate(t)
	stand := copyCorpus(t)
	os.Remove(filepath.Join(stand, "notes", "baustatik-01.md"))
	_, err := Bench(Options{Corpus: stand, Out: t.TempDir()}, depsWith(lockDir))
	var p Problems
	if !errors.As(err, &p) {
		t.Fatalf("err = %v", err)
	}
	if m, _ := filepath.Glob(filepath.Join(filepath.Dir(index.QmdConfigPath()), "loomux-bench-*")); len(m) != 0 {
		t.Fatalf("left behind: %v", m)
	}
}

func TestCorpusRunRanksThroughTheNamedIndex(t *testing.T) {
	lockDir := isolate(t)
	out := t.TempDir()
	fixture := corpusFixture(t) // fakeqmd.Fixture: Collections[loomux-bench-corpus] = the 100 notes, Queries[c.Query] = expected note first for c01..c43, elsewhere for the rest
	d := depsWith(lockDir)
	d.CLI = func(idx string) search.SearchPort { return &search.QmdPort{Index: idx, Runner: fixture.Runner(t)} }
	md, err := Bench(Options{Corpus: corpusDir(t), Out: out, Profile: search.ProfileKeyword}, d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "| total | 43/50 |") {
		t.Fatalf("markdown:\n%s", md)
	}
	files, _ := filepath.Glob(filepath.Join(out, "bench-*-keyword.*"))
	if len(files) != 2 {
		t.Fatalf("files = %v", files)
	}
	for _, argv := range fixture.Calls() {
		if argv[1] != "--index" {
			t.Fatalf("a qmd call without the index: %v", argv)
		}
	}
}

func TestEverydayRunUsesTheDaemonAndTheDefaults(t *testing.T) {
	// registry with area "knowledge" at dir/k, dir/k/98 Messung/questions.yaml with a shape-conformant set
	// Daemon: search.FakePort scripted; Profile default fast; files bench-<stamp>-fast.{md,json} in "98 Messung"
	// environment.port == "daemon"
}

func TestAnEmptyIndexIsNoMeasurement(t *testing.T) {
	// FakePort.Indexed → nothing for every area → error "the engine holds no document for knowledge, so there is nothing to measure against; index the area before measuring"
}

func TestLatencyProbesTheQueryFirst(t *testing.T) {
	// keyword probe answers empty → error naming --latency-query; no timings run
}

func TestLatencyTimesFiveOperationsOnTheFirstDocument(t *testing.T) {
	// Indexed returns b.md, a.md → document "knowledge/a.md"; timings named catalog, read, keyword, fast, full; repeat 2 → 3 calls each
}

func TestFindingsAreNamedByTheirQuery(t *testing.T) {
	// FakePort answers empty twice for query "x" → finding "x: the search engine answered empty twice…"
}

func TestSeveralAreasNeedOut(t *testing.T) {
	// --scope all over two areas without --out → "a run over 2 areas has no measurement folder of its own — name one with --out"
}

func TestTheReportIsWrittenBeforeCleanupAndPrintedAfter(t *testing.T) {
	// Corpus run: after Bench returns, the throwaway state is gone and both files exist.
}
```

`depsWith(lockDir)` setzt `StateDir` auf ein `t.TempDir()` mit `bench`-Unterordner
für die Sperren — die Sperren liegen unter `<StateDir>/bench` (E5), also
`lockDir == filepath.Join(StateDir, "bench")` — und `Models` auf eine
Funktion, die feste Namen liefert. Jeder Test, der einen Korpuslauf
vorbereitet, ruft zusätzlich `isolate(t)` (Task 9), weil `PrepareCorpus` die
Modelle aus `index.QmdConfigPath()` kopiert.

`corpusFixture`, `fixture.Runner(t)` und `fixture.Calls()` sind **lokale
Hilfen in `run_test.go`**, nicht Teil von `fakeqmd`: ein Typ mit
`search.RunnerFunc` in `fakeqmd` riskierte einen Importzyklus mit den
internen Tests von `search`. Die Hilfe umhüllt `(*fakeqmd.Fixture).RunCLI`:

```go
type recordingQmd struct {
	fixture *fakeqmd.Fixture
	calls   [][]string
}

func (r *recordingQmd) Runner(t *testing.T) search.RunnerFunc {
	return func(argv []string) ([]byte, []byte, int, error) {
		r.calls = append(r.calls, argv)
		var out, errOut bytes.Buffer
		code := r.fixture.RunCLI(argv[1:], &out, &errOut)
		return out.Bytes(), errOut.Bytes(), code, nil
	}
}

func (r *recordingQmd) Calls() [][]string { return r.calls }
```

`RunCLI` kennt `update` und `embed` schon (B9, `qmd-calls.log` nur im
Binary); `--index` lernt es in Task 6.

- [ ] **Step 2: Run** → FAIL.

- [ ] **Step 3: `run.go`**, in der Reihenfolge von `_bench`:

1. Stempel einmal: `stamp := benchreport.Stamp(d.Now())`.
2. Korpus-Widersprüche (`--corpus` mit `--scope`, `--questions`, `--latency`; ohne `--out`).
3. Korpus: `CheckCorpus(abs(stand))` — **vor** jedem Schreiben; dann `SweepStale(lockDir, d.Warn)`, `PrepareCorpus(…)`, `defer cleanup()`.
4. `stateDir`, `scope` (Korpus: Wegwerf-Zustand und `CorpusScope`; sonst `d.StateDir` und `o.Scope` bzw. `knowledge`).
5. `areas`: `config.ReadRegistry(stateDir)`; `all` → alle; sonst genau der eine, sonst `no area named %q in the registry`.
6. `out`: `o.Out` oder, bei genau einem Bereich, `<bereich>/98 Messung`; `Targets(out, "bench-"+stamp+"-"+profile)`.
7. Fragensatz: Korpus `<stand>/questions.yaml`, sonst `o.Questions` oder `<out>/questions.yaml`; `LoadQuestions(…, DefaultShape())`.
8. Port: Korpus `d.CLI(prepared.Index)`, danach `port.Refresh` und `port.Embed` für die Sammlung; sonst `d.Daemon()`. `environment.port`: `cli` bzw. `daemon`.
9. Leerer Index → Fehler (Wortlaut in Task 10 Step 1). **`Indexed` nimmt einen Sammlungsnamen, keinen Scope:** Leerer-Index-Prüfung, Leseziel und `documents` rufen `port.Indexed(search.CollectionName(scope))`, wie `status` es tut (`status/status.go:145`); das Leseziel trägt trotzdem den Scope (`<scope>/<relative>`). Test `TestAScopeWithASlashIsListedByItsCollection`: Bereich `project/x`, `FakePort.Indexed` kennt nur `project-x` → der Lauf misst, statt „holds no document“ zu melden.
10. `ask`: `search.ExecuteSearch(query, scope, profile, 10, channel, port, stateDir, fallback, d.Now())`, Befunde als `query + ": " + f`, Treffer zu `filepath.Join(areaPath[hit.Scope], filepath.FromSlash(hit.Relative))`. `fallback` im Korpusmodus `""`.
11. `RunQuality`, dann bei `--latency` die Probe (`keyword`, n=5, leer → Fehler) und `MeasureLatency` über `catalog`/`read` (über `answer.RunWith` mit `Request{Command: "catalog"|"read"}` und `Ports{Search: …port…, Now: d.Now}`), `keyword`, `fast`, `full` (je `ExecuteSearch` mit n=5); Dokument = kleinstes `(scope, relative)` über `port.Indexed`.
12. `Environment`: `benchreport.Current(d.Loomux)` plus `Qmd: d.QmdVersion()`, `Models: d.Models(<Korpus: QmdConfigPathFor(name), sonst QmdConfigPath()>)`, `Profile`, `Port`.
13. `Markdown`, `Envelope` → `JSON`, `WriteBoth`. Rückgabe des Markdowns; der Aufrufer druckt es — nach dem `cleanup`, weil das `defer` in `Bench` schon gelaufen ist.

`QmdVersion`: `search.Launcher("qmd")` + `--version`, leere Ausgabe oder Fehler → `unknown` (nie Abbruch). Das ist eine Naht mit `//coverage:exempt`, weil sie einen Prozess startet.

- [ ] **Step 4: `devBenchSearch` in `dev.go`**

Flags: `--scope`, `--profile keyword|fast|full` (Vorgabe `fast`), `--channel local|cloud` (Vorgabe `local`), `--out`, `--questions`, `--corpus` (Wert `v1` heißt `<wurzel>/testdata/bench/search/v1`, die Wurzel aus `git rev-parse --show-toplevel` über eine Naht `benchRepoRoot`; liegt dort kein `testdata/bench/search/v1`, bricht der Befehl vor jeder Prüfung ab mit `--corpus v1 needs a loomux checkout; name the stand's directory instead`; jeder andere Wert ist ein Pfad), `--latency`, `--latency-query` (Vorgabe `latenz`), `--repeat` (Vorgabe 10, < 1 → Exit 2). `ScopeSet` über `fs.Visit`. Fehler: `Problems` zeilenweise, sonst eine Zeile, jeweils `error: …` auf stderr, Exit 1. Erfolg: Markdown auf stdout, Exit 0. `benchCommands["search"] = devBenchSearch`; `benchUsage` bekommt `<hooks|repos|search>` in der ersten Zeile und die Zeile `  search  measure the rank of search hits and the chain's latency`, und `TestDevBenchWithoutSubcommandPrintsTheGroupsHelp` erwartet sie.

Tests in `dev_test.go`: Flag-Fehler (Exit 2), `--repeat 0` (Exit 2), ein Korpusfehler (Exit 1 mit `error: ` je Zeile), ein Erfolg über eine Naht `benchSearchRun = benchsearch.Bench`.

- [ ] **Step 5: Run** `go test ./internal/dev/benchsearch ./internal/cli -cover` → PASS, 100 %.

- [ ] **Step 6: Commit**

```
feat(dev): add dev bench search over a question set or the corpus v1

The everyday run searches the registered areas through the search
service; the corpus run searches the checked-in corpus through the qmd
command line in a named index. Both write the report into --out, and
--latency times catalog, read and the three search profiles.
```

---

### Task 11: Parität, Selbstnutzung und Doku

**Files:**
- Create: `docs/.superpowers/parity/stufe-4c-2.md`
- Modify: `docs/en/migration.md`, `docs/de/migration.md`, `README.md`, `README.de.md`, `docs/en/cli-reference.md`, `docs/de/cli-reference.md`, `docs/en/benchmarks.md`, `docs/de/benchmarks.md`, `CHANGELOG.md` nur über den PR-Rumpf (nicht von Hand)

- [ ] **Step 1: Parität (von Hand, Mensch oder Controller).** Die Referenz
  schreibt in den geteilten Index; der steht im WAL-Modus und wird vom
  qmd-Dienst offen gehalten. Reihenfolge:

  1. Ein ruhiges Fenster: keine andere Sitzung ruft `brain_*`, sonst startet
     der Dienst neu.
  2. `loomux serve stop`, dann den qmd-Dienst selbst über seine PID beenden
     (`~/.cache/qmd/mcp.pid`, gezielt diese eine PID, nie per Namensfilter)
     und prüfen, dass kein `qmd … mcp` mehr läuft.
  3. Sichern: `index.yml`, `index.sqlite` und, falls vorhanden,
     `index.sqlite-wal` und `index.sqlite-shm`, dazu `sha256sum` jeder Datei.
  4. Mit `update` und `embed` über alle 14 echten Sammlungen ist ein langer
     Lauf der Referenz zu erwarten; im Hintergrund starten, Ausgabe in eine
     Datei.

Dann in `ultra-brain` (Tag `loomux-3-source`, eigene Arbeitskopie per
`git worktree add`): `uv run brain bench --corpus bench/corpus/v1 --profile keyword --out <scratch>/ref`,
danach `loomux dev bench search --corpus v1 --profile keyword --out <scratch>/lx`.
Ränge je Frage aus beiden JSON vergleichen (`questions[].rank`). Eine
Abweichung wird mit den Scores beider Läufe (`qmd search … --json` einmal
mit `-c` im geteilten, einmal im benannten Index) in der Akte begründet,
bevor sie als Fehler gilt. Danach prüfen, dass kein qmd-Dienst läuft,
denselben Satz Dateien zurücklegen (eine `-wal`/`-shm`, die es vorher nicht
gab, löschen) und jede mit `sha256sum` gegen die Sicherung vergleichen.

- [ ] **Step 2: Selbstnutzung.**
  - `loomux dev bench search --corpus v1 --profile fast --out <scratch>` gegen
    die Baseline 43/50 (13/13, 11/13, 8/10, 11/14).
  - Ein Alltagslauf mit `--latency` über den Dienst.
  - `dev bench hooks` über einen vorhandenen Fallsatz mit `--out`.
  - `dev bench repos --dir . --out <scratch>`.

  Qualität und Alltagslatenz in beide `benchmarks.md` nach dem Muster der
  Chronik (Datum und Uhrzeit, was gemessen, kalt und warm).

- [ ] **Step 3: Akte `parity/stufe-4c-2.md`** (deutsch): Paritätslauf mit
  qmd-Version, Rängen je Frage, Scores bei Abweichungen, die Sicherung und
  ihr Vergleich; die Selbstnutzung; die freigegebenen Abweichungen
  (englischer Bericht E2, `--index` statt der echten `index.yml`,
  `--corpus --latency` verweigert E3, Hülle statt des JSON der Referenz).

- [ ] **Step 4: Doku.**
  - `migration.md` en/de: Zeile 4c-2 auf „🚧 built 2026-09-2x; self-use open“ bzw. ✅, wenn Step 2 erledigt ist; die Fähigkeit „Local Model“ nennt `dev bench search` als gebaut.
  - READMEs: die zwei alten Zeilen (`README.md:228-229`, `README.de.md:230-231`) durch drei ersetzen: `dev bench hooks`, `dev bench repos`, `dev bench search`.
  - `cli-reference.md` en/de: `dev bench` als Gruppe mit den drei Unterbefehlen und ihren Flags; `dev bench hooks` fehlte bisher ganz.
  - `benchmarks.md` en/de: die Chronik bleibt, wie sie ist (Spec); oben ein Satz, dass ältere Einträge die alten Befehlsnamen tragen.
  - Per grep nach `bench-hooks` und `dev bench [` in `docs/wiki/topics/scheiben-und-abnahme.md:108` und allen übrigen Treffern außerhalb von `docs/.superpowers/` suchen und nachziehen.

- [ ] **Step 5: Commit**

```
docs: document dev bench hooks|repos|search and the search bench's results
```

---

### Task 12: Gruppieren, Tor, Pull Request

- [ ] **Step 1:** `release-pr`-Skill: Commits nach Thema gruppieren — Task 3, 4
  und 5 werden **ein** Commit `feat(dev)!: group the benchmarks under dev bench
  hooks|repos|search` (der Umbau auf die Hülle gehört zur Umbenennung, beide
  ändern dieselben Befehle); der Spec-Commit bleibt `docs: settle …`; Task 1
  bleibt eigener `fix`.
- [ ] **Step 2:** `sh ci/gate.sh` grün, Ausgabe erst ganz in eine Datei im Scratchpad.
- [ ] **Step 3:** Label `release:major`, Rumpf mit `Release: major — dev bench-hooks and dev bench are renamed` und `## Changelog`:

```
### Added
- `loomux dev bench search` measures the rank of search hits and the search chain's latency over a question set or the checked-in corpus `v1`.
- `dev bench hooks` and `dev bench repos` write a markdown and a JSON report with `--out DIR`.
### Changed
- **Breaking:** `loomux dev bench-hooks` is now `loomux dev bench hooks`, `loomux dev bench` is now `loomux dev bench repos`; `dev bench` alone prints the group's help.
- **Breaking:** `dev bench repos` takes `--out DIR` instead of `--out FILE` and `--json-out FILE`; its JSON and `docs/benchmarks.json` carry milliseconds.
### Fixed
- `dev bench repos --timeout` now limits the time per repository; it was ignored.
```

- [ ] **Step 4:** `parse-body` prüfen, Push-Befehl für den Menschen nennen.

## Selbstprüfung (2026-09-26)

- **Spec-Abdeckung:** Hülle (T2), Umstellung `benchhooks`/`benchcorpus` und
  Bestand in ms (T3, T4), Gruppe und Hilfe (T5), `--index` an jedem Aufruf,
  Parser, `QmdConfigPathFor`, `fakeqmd` mit Treffern je Anfrage (T6),
  Fragensatz, Korpusprüfung mit den Listen aus `corpus.py`, Daten, Lizenz,
  `.gitattributes` (T7), Qualität, Latenz mit Probe, Bericht (T8),
  Wegwerf-Zustand, Modelle aus `index.yml`, Sperre, Aufräumen samt
  `-wal`/`-shm`/Sicherung (T9), Lauf mit allen Vorgaben der Referenz,
  Suchweg je Modus, `environment.port` (T10), `--timeout` (T1),
  Paritätsprobe mit Sicherung beider Dateien, Selbstnutzung, Doku (T11),
  `release:major` (T12). Zustand über Parameter statt `os.Setenv` (T10
  Schritt 3.10).
- **Platzhalter:** keine „TBD“; Testkörper, die nur skizziert sind, nennen
  Eingabe und erwartete Meldung wörtlich.
- **Typen:** `Timing`, `Report`, `Environment` (T2) werden in T3, T4, T8,
  T10 unter denselben Namen benutzt; `Problems` (T7) in T9 und T10;
  `CorpusScope`, `PrepareCorpus`, `SweepStale` (T9) in T10.
- **Review Focus:** alle fünf Zeilen haben einen Test in ihrer Task.
