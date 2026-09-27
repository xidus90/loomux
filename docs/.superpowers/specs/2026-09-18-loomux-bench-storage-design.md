# Spezifikation: Benchmark-Persistierung & Templates (Detailseiten & Matrix)

- **Datum:** 2026-09-18
- **Status:** Umgesetzt; seit `6369f93b` als `loomux dev bench repos --save` (§1, §4.3 und §5 nennen noch `loomux dev bench --save`).
- **Arbeitszweig:** `open-source-matrix`

---

## 1. Ziel & Motivation

Das Benchmark- und Lücken-Audit-Werkzeug `loomux dev bench` liefert reproduzierbare Messergebnisse für Einzel-Repositories und das Open-Source-Matrix-Korpus.
Diese Ergebnisse sollen direkt im Projekt versioniert und als mehrsprachige Dokumentation bereitgestellt werden.
Die Ablage erfolgt hierarchisch:
- **Pro Sprache ein eigener Unterordner** für die Detailseiten der Repositories.
- **Eine zentrale Gesamt-Matrix** (`matrix.md`), die alle getesteten Projekte mit ihren Kennzahlen zusammenfasst und auf die jeweiligen Detailseiten verlinkt.
- **Automatisierte Ablage** via `loomux dev bench --save`.

---

## 2. Verzeichnisstruktur (Bilingual)

Gemäß der Paritätsregel in `AGENTS.md` existieren identische Dateistrukturen in `docs/en/` und `docs/de/`:

```
docs/
├── en/
│   └── benchmarks/
│       ├── matrix.md
│       ├── python/
│       │   ├── django_django.md
│       │   └── pallets_flask.md
│       ├── go/
│       │   ├── gin-gonic_gin.md
│       │   └── loomux.md
│       └── rust/
│           └── tokio-rs_tokio.md
└── de/
    └── benchmarks/
        ├── matrix.md
        ├── python/
        │   ├── django_django.md
        │   └── pallets_flask.md
        ├── go/
        │   ├── gin-gonic_gin.md
        │   └── loomux.md
        └── rust/
            └── tokio-rs_tokio.md
```

---

## 3. Template-Spezifikationen

### 3.1 Detailseite: `docs/{en,de}/benchmarks/<sprache>/<slug>.md`

- **Dateiname:** `<slug>.md` (z. B. `django_django.md`, `loomux.md`).
- **Slug-Bildung:** Aus `RepoURL` oder `Dir` (z. B. `https://github.com/django/django` → `django_django`; für lokales Repo `.` → `loomux`).
- **Sprach-Ordner:** Kleingeschriebener Sprachbezeichner (`python`, `go`, `javascript`, `typescript`, `rust`, etc.).
- **Inhalt:**
  - Zurück-Link zur Gesamt-Matrix: `[← Zurück zur Matrix](../matrix.md)`.
  - Metadaten: Sprache, Framework, Sterne-Kategorie, Commit-SHA, Messdatum, Methode.
  - **Abschnitt 1: Performance & Latenzen** (Tabelle mit Kalt, Warmer Median, Min, Max, Status je Komponente + Gesamt + Baseline-Speedup).
  - **Abschnitt 2: Test- & Lücken-Audit** (Tabelle aller nativen Werkzeuge, Loomux-Lane, PATH-Status, Aktiv/Fehlt + Abdeckungsquote + Gaps).
  - **Abschnitt 3: Erkannte Stacks & Lanes** (Auflistung erkannter Stacks und exakter Loomux-Kommandos).

### 3.2 Gesamt-Matrix: `docs/{en,de}/benchmarks/matrix.md`

- **Inhalt:**
  - Titel & Einleitung mit Aggregat-Kennzahlen: Gesamtzahl getesteter Repositories, durchschnittliche Abdeckungsquote, durchschnittlicher Hook-Speedup.
  - Große Matrix-Tabelle:
    - Spalten: Sprache, Framework, Repository, Sterne-Kategorie, `pre-tool-use` (warm), `post-tool-use` (warm), `graph build` (warm), Baseline-Speedup, Abdeckungsquote, Detailbericht.
    - Sortierung: Deterministisch geordnet nach Sprache, Framework und Repository-Name.
    - Link zur Detailseite: `[Details](<sprache>/<slug>.md)`.

---

## 4. Paketarchitektur & APIs

In `internal/dev/benchcorpus/`:

### 4.1 Neue Funktionen in `template.go`
```go
// FormatDetailMarkdown erzeugt das Markdown für eine einzelne Detailseite.
// lang ist "de" oder "en".
func FormatDetailMarkdown(audit *RepoAudit, lang string, w io.Writer) error

// FormatMatrixMarkdown erzeugt das Markdown für die Gesamt-Matrix.
// lang ist "de" oder "en".
func FormatMatrixMarkdown(report *BenchmarkReport, lang string, w io.Writer) error

// Slug-Helfer
func RepoSlug(audit *RepoAudit) string
func LanguageSlug(lang string, stacks []string) string
```

### 4.2 Persistierung in `storage.go`
```go
type StorageOps struct {
    WriteFile func(path string, data []byte, perm os.FileMode) error
    MkdirAll  func(path string, perm os.FileMode) error
    ReadFile  func(path string) ([]byte, error)
}

// SaveReport speichert alle Detailseiten unter docs/{en,de}/benchmarks/<sprache>/<slug>.md
// und aktualisiert/erzeugt docs/{en,de}/benchmarks/matrix.md.
func SaveReport(report *BenchmarkReport, docsDir string, ops StorageOps) error
```

### 4.3 Matrix-Zusammenführung bei Einzel-Repo-Läufen
Wird `loomux dev bench --dir . --save` aufgerufen:
1. Existiert `matrix.md` bereits, wird sie gelesen und die Einträge analysiert.
2. Das aktuelle Repository wird aktualisiert oder neu eingefügt.
3. Die Matrix wird deterministisch formatiert und neu geschrieben.
4. Die Detailseite im entsprechenden Sprachordner (z. B. `go/loomux.md`) wird erzeugt/aktualisiert.

---

## 5. CLI-Integration

In `internal/cli/dev.go`:
- Flag `--save`: Schreibt die generierten Berichte direkt in die Dokumentation.
- Flag `--report-dir`: Standardwert `"docs"`. Ermöglicht alternative Ausgabeordner für Tests.

---

## 6. Test- und Qualitätsstrategie (100% Coverage)

1. Strikte Dependency-Injection über `StorageOps` (keine echten Plattenzugriffe in Unittests notwendig).
2. Vollständige Unittests für deutsche und englische Formatierung.
3. Tests für Slug-Generierung, Verzeichnishierarchie pro Sprache, Neu- und Update-Pfade in der Matrix.
4. CLI-Tests in `internal/cli/dev_test.go` für `--save` und `--report-dir`.
