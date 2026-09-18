# loomux dev bench — Benchmark-Harness, Stack-Installation und Lücken-Audit

**Datum:** 2026-09-18  
**Status:** Überarbeitet nach Review (2026-09-18)  
**Ort:** `internal/dev/benchcorpus/`, `internal/cli/dev.go`, `internal/hooks/`  
**Bezug:** `docs/open_source_matrix.json`, `docs/en/benchmarks.md`, `docs/de/benchmarks.md`, `internal/dev/benchhooks`

---

## 1. Ziel & Motivation

Loomux vereint die Schreibschranke, die Prüfkette und das Wissenssystem in einem einzigen Go-Binary. Um die Leistungsfähigkeit und Robustheit von Loomux objektiv zu beweisen, Optimierungspotenziale aufzudecken und blinde Flecken im Ökosystem systematisch zu identifizieren, führt dieser Entwurf das Werkzeug **`loomux dev bench`** ein.

Das Werkzeug erfüllt vier Kernaufgaben:
1. **Sicheres Staging & Inspektion:** Rein lesende und risikofreie Analyse beliebiger Ziel-Repositories über `fs.FS` ohne unkontrollierte Fremdcode-Ausführung.
2. **Deterministische Latenz-Messung:** 1 Kaltstartlauf gefolgt von $N$ warmen Läufen (Default: 3, warmer Median, Minimum, Maximum nach `benchhooks`-Standard).
3. **Vorher-/Nachher-Vergleich:** Latenz- und Durchsatzvergleich gegen vorhandene Vorläufer (z. B. `.claude/`-Hooks, Shell-Hooks, Python-Tools) bei Vorhandensein.
4. **Test- & Lücken-Audit (Gap Analysis):** Normalisierter Abgleich nativer Projekt-Checks (z. B. `pytest`, `npm test`, `ruff`, `mvn test`) mit den von Loomux erkannten Stacks und verfügbaren Lanes, um fehlende Sprach-, Test- und Linter-Unterstützungen präzise auszuweisen.
5. **Duales Einsatzprofil:**
   - **Einzel-Repository:** Jeder Entwickler kann `loomux dev bench` im eigenen Projekt ausführen und erhält ein detailliertes Audit für genau dieses Repository.
   - **Korpus-Modus:** Batch-Ausführung über die Open-Source-Matrix (`docs/open_source_matrix.json`), konfigurierbar auf die Top-$N$ Sprachen (Default: 5).

---

## 2. CLI-Schnittstelle

Der Befehl wird als Subcommand von `loomux dev` bereitgestellt:

```sh
# Modus 1: Einzelnes Repository (Default: aktuelles Arbeitsverzeichnis)
loomux dev bench [--dir <pfad>] [weitere Flags]

# Modus 2: Korpus-Benchmark über Open-Source-Matrix
loomux dev bench --corpus docs/open_source_matrix.json [--languages 5] [weitere Flags]
```

### Parameter & Flags

| Flag | Typ | Standard | Validierung | Beschreibung |
|---|---|---|---|---|
| `--dir` | String | `.` | Existierendes Verzeichnis | Pfad zum Ziel-Repository für den Einzel-Benchmark. |
| `--corpus` | String | `""` | Existierende JSON-Datei | Pfad zur `open_source_matrix.json` für den Batch-Lauf. |
| `--languages` | Int | `5` | $\ge 1$ | Anzahl der Top-Sprachen im Korpus-Modus (z. B. 5, 10, 15, 25). |
| `--tier` | String | `"Sehr viel"` | Bekannte Tier-Bezeichnung | Sterne-Kategorie für Korpus-Sampling (`Sehr viel` / Top-Repo je Framework, oder `all`). |
| `--warm` | Int | `3` | $\ge 1$ | Anzahl der warmen Messläufe nach dem Kaltstart (Medianberechnung). |
| `--cache-dir` | String | `".loomux/cache/corpus"` | - | Lokales Cache-Verzeichnis für Korpus-Klone. |
| `--timeout` | Duration | `30s` | $> 0$ | Timeout pro gemessenem Schritt/Befehl. |
| `--out` | String | `""` | - | Zieldatei für den Markdown-Bericht (Default: stdout). |
| `--json-out` | String | `""` | - | Zieldatei für den maschinenlesbaren JSON-Bericht. |

---

## 3. Die 4 Phasen des Benchmarks

Jedes evaluierte Repository durchläuft vier deterministische Phasen:

### Phase 1: Bestandsaufnahme & Baseline (Vorher)
- **Sichere Inspektion (`fs.FS`):** Das Projekt wird rein lesend durchsucht:
  - Vorhandene Claude-Hooks (`.claude/settings.json`) oder Git-Hooks (`.githooks/`, `.git/hooks/`).
  - Native Test-/Linter-Konfigurationen:
    - *Python:* `pyproject.toml`, `pytest.ini`, `setup.cfg`, `requirements.txt`, `manage.py`, `ruff.toml`.
    - *JavaScript/TypeScript:* `package.json` (Scripts: `test`, `lint`, `typecheck`), `eslint.config.*`, `jest.config.*`, `vitest.config.*`.
    - *Go:* `go.mod`, `*_test.go`, `.golangci.yml`.
    - *Rust:* `Cargo.toml`, `tests/`.
    - *C++:* `CMakeLists.txt`, `Makefile`, `.clang-tidy`.
    - *Java:* `pom.xml`, `build.gradle`, `build.gradle.kts`.
    - *C#:* `*.sln`, `*.csproj`.
    - *PHP:* `composer.json`, `phpunit.xml`.
- **Baseline-Messung:** Falls Vorläufer-Hooks oder Claude-Hooks im lokalen Repo existieren, wird deren Laufzeit (1x kalt, $N$x warm) als Vergleichsbasis gemessen. In Fremd-Repos werden keine Fremdskripte ohne explizite Freigabe ausgeführt.

### Phase 2: Stack-Bereitstellung & Isolation
- Die Loomux-Komponenten werden passiv/in-memory bereitgestellt, ohne das Ziel-Repository destruktiv zu mutieren:
  - Exportierte Lane-Ermittlung aus `internal/hooks`: Bereitstellung der aktiven Lanes für die erkannten Stacks.
  - CodeGraph-Prüfung: Erkennung, ob der CodeGraph für die Sprache anwendbar ist (`golang`).
  - Brain/Wiki: Erkennung vorhandener Dokumentations-Bundles (`detect.Detect` -> `WikiMode`).

### Phase 3: Performance-Messung (Kalt & Warm)
Gemessen werden die Kernkomponenten des Loomux-Laufzeitpfads:
1. **Schreibschranke (`hook pre-tool-use`):** Validierung eines simulierten Edits gegen Registry/Policy.
2. **Prüfkette (`hook post-tool-use`):** Ausführung der ermittelten Loomux-Lanes.
3. **CodeGraph (`graph build`):** Zeit für Extraktion, Auflösung und Schreiben von `wiring.json` (**nur wenn Sprache anwendbar, sonst als `n/a` ausgewiesen**).
4. **Brain:** Wissensabfrage (`search`/`catalog`).

- **Kaltstart:** Exakt der 1. Durchlauf vor Caching.
- **Warmstart:** $N$ Durchläufe (Standard: 3), Berechnung von Median, Minimum und Maximum (gemäß `docs/en/benchmarks.md`):
  $$\text{Warm}_{\text{median}} = \text{median}(t_1, \dots, t_N)$$
- **Delta/Speedup:** Falls eine Vorläufer-Baseline vorlag:
  $$\Delta = t_{\text{Loomux}} - t_{\text{Vorher}}, \quad \text{Speedup} = \frac{t_{\text{Vorher}}}{t_{\text{Loomux}}}$$

### Phase 4: Normalisiertes Test- & Lücken-Audit (Gap Analysis)
- **Normalisierter Werkzeug-Abgleich:**
  Statt reinem Textvergleich werden normalisierte Werkzeug-IDs und Kategorien abgeglichen:
  - Werkzeug: `ruff`, `pytest`, `eslint`, `tsc`, `vitest`, `jest`, `cargo-test`, `go-test`, `dotnet-test`, `mvn-test`
  - Kategorie: `lint`, `typecheck`, `test`, `format`
- **Berechnung der Abdeckungsquote:**
  $$\text{Coverage} = \frac{|\text{Native Checks mit aktiver Loomux-Lane}|}{|\text{Native Checks gesamt}|}$$
- **Katalogisierung der Lücken:**
  - `MissingLanes`: Werkzeug im Projekt konfiguriert, aber Loomux bietet keine Lane dafür (z. B. `pytest` oder `dotnet test`).
  - `MissingStacks`: Gesamte Programmiersprache wird von Loomux noch nicht erkannt (z. B. C#, Java, PHP).
  - `MissingToolOnPath`: Lane existiert, aber Binärdatei fehlt auf dem System.

---

## 4. Paketarchitektur & API-Erweiterung

### Erweiterung von `internal/hooks`
`internal/hooks` erhält eine exportierte Funktion zur Ermittlung konfigurierter Lanes:
```go
package hooks

// TargetCommandsForStacks liefert die Kommandos für eine Menge von Stacks
func TargetCommandsForStacks(stacks []string, targetPath string, godotDir string, projectRoot string, wikiDir string) []string
```

### Paket `internal/dev/benchcorpus`
```
internal/dev/benchcorpus/
├── benchcorpus.go      // Haupt-Orchestrierung: BenchmarkRepo, BenchmarkCorpus, Optionen
├── inspect.go          // Projekt-Inspektion (Erkennung nativer Linter/Tests im Ziel-Repo via fs.FS)
├── gap.go              // Lücken-Audit: Normalisierter Abgleich nativ vs. Loomux-Lanes
├── report.go           // Formatierung der deterministischen Markdown- und JSON-Berichte
└── benchcorpus_test.go // 100% Test-Coverage mit injizierten Runnern, Clock und Mock-FS
```

### Datenstrukturen

```go
type Options struct {
    TargetDir   string        `json:"target_dir"`
    CorpusFile  string        `json:"corpus_file"`
    Languages   int           `json:"languages"`    // Default: 5
    Tier        string        `json:"tier"`         // Default: "Sehr viel"
    WarmRuns    int           `json:"warm_runs"`    // Default: 3
    CacheDir    string        `json:"cache_dir"`
    Timeout     time.Duration `json:"timeout"`
    OutFile     string        `json:"out_file"`
    JSONOutFile string        `json:"json_out_file"`
}

type CheckAudit struct {
    Tool     string `json:"tool"`     // z. B. "ruff", "pytest", "eslint"
    Category string `json:"category"` // lint | typecheck | test | format
    Native   string `json:"native"`   // z. B. "package.json: scripts.test"
    Lane     string `json:"lane"`     // Loomux-Kommando oder ""
    OnPath   bool   `json:"on_path"`  // Werkzeug im PATH vorhanden
}

type ComponentTiming struct {
    Component    string    `json:"component"`
    Applicable   bool      `json:"applicable"`   // z. B. false für CodeGraph bei non-Go
    ColdMs       float64   `json:"cold_ms"`
    WarmMedianMs float64   `json:"warm_median_ms"`
    WarmMinMs    float64   `json:"warm_min_ms"`
    WarmMaxMs    float64   `json:"warm_max_ms"`
    WarmRuns     []float64 `json:"warm_runs"`
    ExitCodes    []int     `json:"exit_codes"`
    BeforeMs     float64   `json:"before_ms,omitempty"`
    Speedup      float64   `json:"speedup,omitempty"`
}

type RepoAudit struct {
    RepoName       string            `json:"repo_name"`
    Language       string            `json:"language,omitempty"`
    Framework      string            `json:"framework,omitempty"`
    CommitSHA      string            `json:"commit_sha,omitempty"`
    DetectedStacks []string          `json:"detected_stacks"`
    ExecutedLanes  []string          `json:"executed_lanes"`
    Checks         []CheckAudit      `json:"checks"`
    MissingGaps    []string          `json:"missing_gaps"`
    CoverageRate   float64           `json:"coverage_rate"`
    Timings        []ComponentTiming `json:"timings"`
}

type BenchmarkReport struct {
    Timestamp  string       `json:"timestamp"`
    Version    string       `json:"version"`
    GitCommit  string       `json:"git_commit"`
    Method     string       `json:"method"` // z. B. "1x kalt, 3x warmer Median"
    WarmCount  int          `json:"warm_count"`
    TotalRepos int          `json:"total_repos"`
    Audits     []RepoAudit  `json:"audits"`
}
```

---

## 5. Berichtsformat (Markdown)

```markdown
# Loomux Benchmark & Lücken-Audit

- **Datum:** 2026-09-18T15:35:00Z
- **Version:** loomux v0.1.0 (commit 65c1568)
- **Methode:** 1x kalt, 3x warmer Median (gemäß benchmarks.md-Standard)
- **Modus:** Einzel-Repository (`.`) | Korpus (Top 5 Sprachen, 50 Repositories)

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Vorher (Claude/Alt) | Delta / Speedup | Status |
|---|---:|---:|---:|---:|---:|---:|---|
| **Schreibschranke** (`pre-tool-use`) | 28,5 ms | 23,8 ms | 23,1 ms | 24,9 ms | 72,0 ms | -48,2 ms (3,0x) | [0] |
| **Prüfkette** (`post-tool-use`) | 142,0 ms | 88,4 ms | 84,2 ms | 95,1 ms | 650,0 ms | -561,6 ms (7,4x) | [0] |
| **CodeGraph** (`graph build`) | n/a | n/a | n/a | n/a | - | n/a (non-Go) | - |
| **Brain** (`brain search`) | 34,1 ms | 28,0 ms | 27,2 ms | 29,8 ms | 890,0 ms | -862,0 ms (31x) | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `ruff` | lint | `pyproject.toml: [tool.ruff]` | `ruff check --output-format=concise .` | Ja | ✅ Aktiv |
| `mypy` | typecheck | `pyproject.toml: [tool.mypy]` | `mypy --no-error-summary --no-pretty` | Ja | ✅ Aktiv |
| `pytest` | test | `pytest.ini` | *keine* | Ja | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **66,7 %** (2 von 3 nativen Checks gegriffen)
- **Identifizierte Lücken (Gaps):**
  - ⚠️ `pytest` deklariert, aber keine Test-Lane in Loomux vorhanden.
```

---

## 6. Test- und Qualitätsstrategie (100% Coverage)

1. **Injektions-Design:**
   - Dateizugriffe erfolgen strikt über `io/fs.FS`.
   - Externe Prozessausführungen laufen über `type ProcessRunner func(dir string, argv []string) (string, int, error)`.
   - Zeitmessung läuft über `type Clock func() time.Time`.
   - Klonen erfolgt über `type Cloner func(repoURL, targetDir string) (string, error)`.
2. **Vollständige Unittests:**
   - 100 % Coverage für jedes File in `internal/dev/benchcorpus/` ohne echte Netzwerk- oder Prozessaufrufe.
   - Separate Integrationstests prüfen den CLI-Aufruf und Flag-Validierungen in `internal/cli/dev_test.go`.
