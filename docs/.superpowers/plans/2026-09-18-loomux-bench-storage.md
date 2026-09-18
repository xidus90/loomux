# Implementierungsplan: Benchmark-Persistierung & Templates (Detailseiten & Matrix pro Sprache)

> **Für Agenten:** ERFORDERLICHER SUB-SKILL: Nutze superpowers:subagent-driven-development (empfohlen) oder superpowers:executing-plans, um diesen Plan taskweise abzuarbeiten. Schritte nutzen Checkbox-Syntax (`- [ ]`).

**Ziel:** Automatisiertes Speichern von Benchmarks in `docs/` mit einer strukturierten Hierarchie: pro Sprache ein Unterordner (`docs/{en,de}/benchmarks/<sprache>/<slug>.md`) für Detailberichte und eine zentrale Gesamt-Matrix (`docs/{en,de}/benchmarks/matrix.md`), steuerbar über `loomux dev bench --save`.

**Architektur:**
- Erweiterung des Pakets `internal/dev/benchcorpus/`:
  - `template.go`: Zweisprachige Formatierer (`en`/`de`) für Detailseiten und Matrix sowie Slug-Helfer.
  - `storage.go`: Dateisystem-Orchestrierung (`SaveReport`) mit injizierten Datei- und Verzeichnisoperationen (`StorageOps`).
- Integration in `internal/cli/dev.go` (`--save`, `--report-dir`).

**Spezifikation:** `docs/.superpowers/specs/2026-09-18-loomux-bench-storage-design.md`

## Globale Regeln
- 100 % Statement-Coverage pro Funktion (`covergate`).
- Reine Go-Standardbibliothek.
- Strikte Dependency-Injection für Dateisystemoperationen (`StorageOps`).
- Dokumentationsparität zwischen `docs/en/` und `docs/de/`.

---

### Task 1: Slug-Helfer & Detailseiten-Formatierer (`template.go`)

**Dateien:**
- Erstellen: `internal/dev/benchcorpus/template.go`
- Test: `internal/dev/benchcorpus/template_test.go`

**Schnittstellen:**
- `func RepoSlug(audit *RepoAudit) string`
- `func LanguageSlug(lang string, stacks []string) string`
- `func FormatDetailMarkdown(audit *RepoAudit, lang string, w io.Writer) error`

- [ ] **Schritt 1: Fehlschlagende Tests schreiben**
  In `template_test.go` Tests für `RepoSlug`, `LanguageSlug` und `FormatDetailMarkdown` (sowohl für `"de"` als auch `"en"` mit allen Tabellen und Lücken) anlegen.

- [ ] **Schritt 2: Tests ausführen und Fehlschlag prüfen**
  `go test ./internal/dev/benchcorpus -run TestDetail`

- [ ] **Schritt 3: `template.go` implementieren**
  Slug-Erzeugung, Sprachordner-Normalisierung und zweisprachige Detailseiten formatieren.

- [ ] **Schritt 4: 100 % Coverage verifizieren**
  `go test -count=1 "-coverprofile=coverage.out" ./internal/dev/benchcorpus`
  `go tool cover -func coverage.out | Select-String "template.go"`

- [ ] **Schritt 5: Commit**
  ```bash
  git add internal/dev/benchcorpus/template.go internal/dev/benchcorpus/template_test.go
  git commit -m "feat(benchcorpus): add slug helpers and bilingual detail page formatter"
  ```

---

### Task 2: Gesamt-Matrix-Formatierer (`matrix.go`)

**Dateien:**
- Erstellen: `internal/dev/benchcorpus/matrix.go`
- Test: `internal/dev/benchcorpus/matrix_test.go`

**Schnittstellen:**
- `func FormatMatrixMarkdown(report *BenchmarkReport, lang string, w io.Writer) error`
- `func MergeAudits(existing []*RepoAudit, incoming *RepoAudit) []*RepoAudit`

- [ ] **Schritt 1: Fehlschlagende Tests schreiben**
  In `matrix_test.go` Tests für Matrix-Tabelle (deutsche und englische Kopfzeilen, Links auf `<sprache>/<slug>.md`), Aggregat-Zusammenfassungen und das Zusammenführen/Aktualisieren bestehender Einträge anlegen.

- [ ] **Schritt 2: Tests ausführen und Fehlschlag prüfen**
  `go test ./internal/dev/benchcorpus -run TestMatrix`

- [ ] **Schritt 3: `matrix.go` implementieren**
  Matrix-Formatierung mit deterministischer Sortierung nach Sprache und Repository sowie Merge-Logik.

- [ ] **Schritt 4: 100 % Coverage verifizieren**
  `go test -count=1 "-coverprofile=coverage.out" ./internal/dev/benchcorpus`
  `go tool cover -func coverage.out | Select-String "matrix.go"`

- [ ] **Schritt 5: Commit**
  ```bash
  git add internal/dev/benchcorpus/matrix.go internal/dev/benchcorpus/matrix_test.go
  git commit -m "feat(benchcorpus): add bilingual matrix formatter and audit merger"
  ```

---

### Task 3: Storage-Engine mit Sprachunterordnern (`storage.go`)

**Dateien:**
- Erstellen: `internal/dev/benchcorpus/storage.go`
- Test: `internal/dev/benchcorpus/storage_test.go`

**Schnittstellen:**
- `type StorageOps struct`
- `func SaveReport(report *BenchmarkReport, docsDir string, ops StorageOps) error`

- [ ] **Schritt 1: Fehlschlagende Tests schreiben**
  In `storage_test.go` testen:
  - Anlegen der Verzeichnisse `docs/en/benchmarks/<sprache>/` und `docs/de/benchmarks/<sprache>/`.
  - Schreiben der Detailseiten in beiden Sprachen.
  - Schreiben/Aktualisieren der `matrix.md` in beiden Sprachen.
  - Fehlerbehandlung bei Schreib- oder Verzeichnisfehlern.

- [ ] **Schritt 2: Tests ausführen und Fehlschlag prüfen**
  `go test ./internal/dev/benchcorpus -run TestSaveReport`

- [ ] **Schritt 3: `storage.go` implementieren**
  Dateisystem-Persistierung über `StorageOps` orchestrieren.

- [ ] **Schritt 4: 100 % Coverage verifizieren**
  `go test -count=1 "-coverprofile=coverage.out" ./internal/dev/benchcorpus`
  `go tool cover -func coverage.out | Select-String "storage.go"`

- [ ] **Schritt 5: Commit**
  ```bash
  git add internal/dev/benchcorpus/storage.go internal/dev/benchcorpus/storage_test.go
  git commit -m "feat(benchcorpus): implement storage engine for language-partitioned benchmarks"
  ```

---

### Task 4: CLI-Integration (`loomux dev bench --save`)

**Dateien:**
- Modifizieren: `internal/cli/dev.go`
- Modifizieren: `internal/cli/dev_test.go`

- [ ] **Schritt 1: Tests in `dev_test.go` erweitern**
  Tests für `--save` und `--report-dir` hinzufügen (sowohl Erfolgsfall als auch Fehlerpfad bei Speicherfehlern).

- [ ] **Schritt 2: Tests ausführen und Fehlschlag prüfen**
  `go test ./internal/cli -run TestDevBenchSave`

- [ ] **Schritt 3: `dev.go` anpassen**
  Flags `--save` und `--report-dir` in `devBench` registrieren und `SaveReport` aufrufen.

- [ ] **Schritt 4: 100 % Coverage verifizieren**
  `go test -count=1 "-coverprofile=coverage.out" ./internal/cli`
  `go run ./cmd/loomux dev covergate --profile coverage.out`

- [ ] **Schritt 5: Commit**
  ```bash
  git add internal/cli/dev.go internal/cli/dev_test.go
  git commit -m "feat(cli): add --save and --report-dir flags to loomux dev bench"
  ```

---

### Task 5: Pilotlauf, Dokumentation & Persistierung im Projekt

- [ ] **Schritt 1: `loomux.exe` kompilieren**
  `go build -o bin/loomux.exe ./cmd/loomux`

- [ ] **Schritt 2: Pilotlauf mit `--save` ausführen**
  `.\bin\loomux.exe dev bench --dir . --warm 3 --save`
  Prüfen, ob:
  - `docs/en/benchmarks/go/loomux.md`
  - `docs/de/benchmarks/go/loomux.md`
  - `docs/en/benchmarks/matrix.md`
  - `docs/de/benchmarks/matrix.md`
  erzeugt wurden.

- [ ] **Schritt 3: CLI-Referenz und READMEs aktualisieren**
  Dokumentation von `--save` und `--report-dir` in:
  - `docs/en/cli-reference.md` & `docs/de/cli-reference.md`
  - `README.md` & `README.de.md`

- [ ] **Schritt 4: Gesamttests und Covergate**
  Pre-Commit Gate prüfen.

- [ ] **Schritt 5: Commit**
  ```bash
  git add docs/ README.md README.de.md
  git commit -m "docs: document --save and generate initial benchmark matrix and detail pages"
  ```
