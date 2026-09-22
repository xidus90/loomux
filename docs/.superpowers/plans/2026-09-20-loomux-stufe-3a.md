# loomux Stufe 3a — Implementierungsplan: Index, Abgleich, Bereichs-Onboarding

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** loomux erkennt selbst, dass sich Wissen geändert hat: es indiziert
(`reindex`, `embed`), gleicht vorher ab (`reconcile`), eröffnet Fälle im
Prüfzentrum und nimmt neue Bereiche auf (`area add`) — alles mit eigenem
Zustand unter `LOOMUX_STATE_DIR`.

**Architecture:** Vier Schichten, von unten nach oben. `internal/lock` und
`internal/config` bekommen die Schreibseite (atomares Ersetzen, Registry).
`internal/config/legacy.go` stellt von „alles alt" auf „neu zuerst, alt als
Rückfall" um. `internal/brain/index` zieht aus `ultra-brain/pkg/index` um und
bringt `reindex`/`embed`. `internal/brain/maintenance` ist neu geschrieben
nach `src/brain/maintenance/` und bringt Fallakte, Ereignisprotokoll und
`reconcile`. Die CLI hängt vier Befehle daran.

**Tech Stack:** Go ≥ 1.25, `github.com/BurntSushi/toml`, `go test -cover`,
`loomux check precommit` als Tor. Keine neuen Abhängigkeiten.

**Spec:** `docs/.superpowers/specs/2026-09-19-loomux-stufe-3-design.md`

## Global Constraints

- **Sprache — die `AGENTS.md` dieses Repos gilt, nicht die allgemeine Regel.**
  `AGENTS.md:28`: englisch sind diese Datei, `.claude/**`, **Code, Kommentare,
  Fehlermeldungen und Commit-Nachrichten**. Deutsch sind nur die Dokumente
  unter `docs/.superpowers/` und die deutschen Handbuchseiten. Die erste
  Fassung dieses Plans verlangte deutsche Kommentare — das war falsch und ist
  am 2026-09-20 korrigiert (Ruling im Ledger).
  **Eine Ausnahme bleibt:** Nutzermeldungen, die aus der Referenz übernommen
  werden, sind dort deutsch (`cli.py:1019-1038`, `init.py:22-30`) und bleiben
  es — sie sind Teil des Verhaltens, nicht der Sprache des Codes.
- **TDD, 100 % Coverage**, jeder Ausschluss mit Begründung im Code.
- **Statische Typen**, kein `interface{}` ohne Grund.
- **Referenz ist Python** (`ultra-brain` Tag `loomux-3-source` auf `3cc72d2`).
  Wo eine Go-Form in `ultra-brain/pkg/` existiert, zieht sie als Code um, ist
  aber nicht die Referenz.
- **Zustandsort:** geschrieben wird nach `LOOMUX_STATE_DIR`, nie ins
  Altverzeichnis. Gelesen wird neu zuerst, alt als Rückfall.
- **Startzeit-Regel:** kein `init()` und keine Paketvariable parst eingebettete
  Daten. Geladen wird beim ersten Gebrauch.
- **`hooks` importiert nichts aus `brain/maintenance` oder `brain/index`.**
  Der Pfad an jedem Edit bleibt frei von der Pflegeschicht.
- **Commits:** englisch, konventionell, mehrzeilige Nachrichten über eine
  Datei und `git commit -F`, nie über ein Heredoc. **Kein `Co-Authored-By:`**
  auf ein Modell, keine Werbezeile.
- **Ein Shell-Befehl je Aufruf**, keine langen `&&`-Ketten.
- **Subagenten:** `model: "opus"`, `effort: "low"` — **beides explizit
  setzen**. Erben fällt sonst still auf den Sitzungswert zurück.
- **Kein `.git/COMMIT_BODY`.** Dies ist ein Worktree; `.git` ist dort eine
  Datei, kein Verzeichnis. Mehrzeilige Commit-Nachrichten kommen in eine Datei
  im Scratchpad-Verzeichnis der Sitzung und werden mit `git commit -F <pfad>`
  übergeben.
- **Vorgeklärt am 2026-09-20, damit kein Subagent es zweimal erhebt:**
  - `index.Identity` und `internal/brain/identity.Identity` sind **feldgleich**
    (`DocID`, `Relative`, `ContentHash`, `Revision`). Der Umzug in Task 4
    importiert das loomux-Paket, ohne den Typ anzupassen.
  - `internal/brain/catalog` kennt **keinen** `Document`-Typ — es liefert den
    Katalog als Text. `index.Document` zieht darum frisch um, ohne Konflikt.
  - Es gibt **keinen** importierbaren Repo-Testhelfer:
    `internal/gitwork/gitwork_test.go:114 func repo(t *testing.T) string` ist
    paketprivat. Task 6 schreibt ihn, und die Tasks 10 und 11 benutzen den von
    dort. **Nicht drei Mal bauen.**
- **Vor jedem Commit Zweig und HEAD lesen** (`git status -sb`), weil eine
  fremde Sitzung im selben Checkout den Index leeren kann.

## File Structure

| Datei | Verantwortung | Task |
|---|---|---|
| `internal/lock/replace.go` | `ReplaceText`: daneben schreiben, atomar tauschen, unter Windows wiederholen | 1 |
| `internal/config/artifacts.go` | `ArtifactLookup`: ein Pfad neu, sonst alt; ersetzt den festen Griff ins Altverzeichnis | 2 |
| `internal/config/legacy.go` | umgestellt: `LegacyBrainDirUntilStage3` wird der **Rückfall**, nicht der Ort | 2 |
| `internal/config/registrywrite.go` | `AddArea`, `WriteRegistry`: unter Sperre lesen, ändern, atomar zurückschreiben | 3 |
| `internal/brain/index/{walk,document,reindex,qmdconfig,catalogwrite}.go` | Umzug aus `ultra-brain/pkg/index`; `identity` und die Katalog-Leseseite sind seit 1b-1 da | 4 |
| `internal/cli/index.go` | `loomux reindex`, `loomux embed` | 5 |
| `internal/brain/vcs/vcs.go` | Leseseite von `vcs.py`: `ShowBlob`, `ChangedPaths`, `CommitSubjects`, `RepositoryRoot`, `CommonDirectory` | 6 |
| `internal/brain/maintenance/events.go` | Ereignisprotokoll: Zeilenformat, lesen, ablegen | 7 |
| `internal/brain/maintenance/case.go` | Fallakte: `CaseID`, `CaseDir`, `WriteCase`, `ReadCase` | 8 |
| `internal/brain/maintenance/derive.go` | `Dependents`: wer auf eine Wiki-Seite zeigt | 9 |
| `internal/brain/maintenance/scan.go` | `_scan`, `_baseline`, `_decode`, die Statistikdatei | 10 |
| `internal/brain/maintenance/reconcile.go` | Prüfzentrum, Quellfälle, Merge-Fälle, Entdoppelung, Bericht | 11, 12 |
| `internal/cli/maintenance.go` | `loomux reconcile` | 13 |
| `internal/brain/index/catchup.go` | der Auffangdurchgang vor `reindex` | 14 |
| `internal/cli/area.go` | `loomux area add` | 15 |
| `internal/cli/cases_3a_test.go`, `testdata/cases/3a/**` | die Fallsuite | 16 |

---

### Task 0: Arbeitsort und Vorbedingungen

**Files:** keine. Dieser Task schreibt keinen Code; er stellt drei Dinge fest,
deren Antworten spätere Tasks ändern.

- [ ] **Step 1: Den Arbeitsort prüfen**

Run: `git status -sb`
Expected: ein eigener Worktree, abgezweigt von `master`, nicht `master`
selbst und nicht der Zweig einer anderen Sitzung. Die Zweige `sdd-2c` und
`claude/2b-planung-4d6dcc` laufen parallel — nichts von 3a gehört dorthin.

- [ ] **Step 2: Den Merge-Stand von 2c feststellen**

Run: `ls internal/cases/gitworld.go`
Expected: entweder die Datei (2c ist drin) oder „No such file".

Ist sie **nicht** da, gilt für Task 16: die Fälle mit echtem Git bleiben
geparkt und stehen mit Grund in `parity/stufe-3a.md`. **Baue keine zweite
Git-Welt daneben** — das gäbe einen sicheren Merge-Konflikt in
`internal/cases`.

- [ ] **Step 3: Klären, ob der Rekorder ein `.git` durchreicht**

`tools/cases.py` stellt `world/` als Dateibaum her und vergleicht
`world_after/` genauso (`tools/cases.py:36,121-159`). Ob ein `.git` darin
unverändert durchläuft, ist **nicht geprüft**. Probiere es an **einem** Fall
aus, bevor Task 16 dreißig aufzeichnet:

```bash
cd ~/Documents/#GIT/ultra-brain
```

```bash
uv run python tools/cases.py --help
```

Dann einen Fall mit einem Repo in `world/` aufnehmen und zurückspielen.
Reicht es nicht, ergänze den Rekorder **in `ultra-brain`**, wo er lebt — und
notiere das als Voraussetzung von Task 16.

- [ ] **Step 4: Den Befund festhalten**

Schreib die drei Antworten in `docs/.superpowers/parity/stufe-3a.md` unter
eine Überschrift „Vorbedingungen, festgestellt am <Datum>". Die späteren Tasks
lesen sie dort, statt sie noch einmal zu erheben.

---

### Task 1: `internal/lock/replace.go` — atomar ersetzen

**Files:**
- Create: `internal/lock/replace.go`
- Test: `internal/lock/replace_test.go`

**Interfaces:**
- Consumes: nichts.
- Produces: `func ReplaceText(path, text string) error` — schreibt `text`
  UTF-8 ohne BOM und **ohne CRLF-Übersetzung** in eine temporäre Datei im
  Verzeichnis von `path`, dann `os.Rename`. Unter Windows wird der Tausch bis
  zu fünf Mal mit 20 ms Pause wiederholt. Scheitert er endgültig, wird die
  temporäre Datei entfernt und der Fehler zurückgegeben.

**Warum so:** `locking.replace_text` (`src/brain/locking.py:219-252`).
`newline=""` dort ist der Grund für „ohne CRLF-Übersetzung": die Registry wird
byteweise mit dem Go-Leser verglichen, eine Übersetzung wäre ein stiller
Unterschied. Der Name der temporären Datei enthält keinen PID, sondern kommt
von `os.CreateTemp` — zwei Schreiber eines Prozesses träfen sich sonst.

- [ ] **Step 1: Write the failing test**

```go
package lock_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/lock"
)

func TestReplaceTextWritesContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.toml")
	if err := lock.ReplaceText(path, "scope = \"a\"\n"); err != nil {
		t.Fatalf("ReplaceText: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "scope = \"a\"\n" {
		t.Fatalf("content = %q", got)
	}
}

// Der Lauf darf keine .tmp-Datei zurücklassen: ein Leser, der das
// Verzeichnis listet, sähe sonst zwei Registries.
func TestReplaceTextLeavesNoTemporary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.toml")
	if err := lock.ReplaceText(path, "x\n"); err != nil {
		t.Fatalf("ReplaceText: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
}

// CRLF bleibt CRLF und LF bleibt LF: die Registry wird byteweise verglichen.
func TestReplaceTextDoesNotTranslateNewlines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mixed.txt")
	if err := lock.ReplaceText(path, "a\r\nb\n"); err != nil {
		t.Fatalf("ReplaceText: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "a\r\nb\n" {
		t.Fatalf("content = %q", got)
	}
}

// Ein Zielverzeichnis, das es nicht gibt, ist ein Fehler und kein stilles
// Anlegen: wer hierher schreibt, hat den Ort schon hergestellt.
func TestReplaceTextRefusesMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope", "registry.toml")
	if err := lock.ReplaceText(path, "x"); err == nil {
		t.Fatal("ReplaceText: want error for a missing directory")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/lock/ -run TestReplaceText -v`
Expected: FAIL, `undefined: lock.ReplaceText`

- [ ] **Step 3: Write minimal implementation**

```go
package lock

import (
	"os"
	"path/filepath"
	"time"
)

// replaceTries und replacePause sind `_replace_eventually`s Werte
// (src/brain/locking.py:238-252). Windows verweigert den Tausch, solange ein
// anderer Prozess das Ziel offen hält, und ein Leser trifft dieses Fenster
// innerhalb von Millisekunden.
const (
	replaceTries = 5
	replacePause = 20 * time.Millisecond
)

// ReplaceText schreibt text daneben und tauscht ihn dann ein.
//
// Ohne Übersetzung der Zeilenenden: die Registry wird byteweise mit dem
// Python-Leser verglichen, und eine Umstellung auf CRLF wäre ein stiller
// Unterschied (locking.replace_text, `newline=""`).
func ReplaceText(path, text string) error {
	directory := filepath.Dir(path)
	// Ein eigener Name je Schreiber, nicht die PID: zwei Schreiber eines
	// Prozesses träfen sich sonst auf derselben temporären Datei.
	temporary, err := os.CreateTemp(directory, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	name := temporary.Name()
	if _, err := temporary.WriteString(text); err != nil {
		temporary.Close()
		os.Remove(name)
		return err
	}
	if err := temporary.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := replaceEventually(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// replaceEventually tauscht die Datei ein und gibt Windows Zeit, einen
// offenen Lesegriff fallen zu lassen.
func replaceEventually(source, target string) error {
	var err error
	for attempt := 0; attempt < replaceTries; attempt++ {
		if err = os.Rename(source, target); err == nil {
			return nil
		}
		time.Sleep(replacePause)
	}
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/lock/ -cover`
Expected: PASS, coverage 100.0 % für das Paket

- [ ] **Step 5: Commit**

```bash
git add internal/lock/replace.go internal/lock/replace_test.go
```

```bash
git commit -m "feat(lock): replace a file's text atomically"
```

---

### Task 2: `internal/config` — neu zuerst, alt als Rückfall

**Files:**
- Create: `internal/config/artifacts.go`
- Test: `internal/config/artifacts_test.go`
- Modify: `internal/config/legacy.go` (Kommentar von „Ort" auf „Rückfall")

**Interfaces:**
- Consumes: nichts.
- Produces:
  ```go
  // ArtifactLookup findet eine Zustandsdatei: neu zuerst, alt als Rückfall.
  type ArtifactLookup struct {
      Primary  string // LOOMUX_STATE_DIR
      Fallback string // LegacyBrainDirUntilStage3()
  }
  func NewArtifactLookup() ArtifactLookup
  func (l ArtifactLookup) Resolve(relative string) string
  func (l ArtifactLookup) WritePath(relative string) string
  ```
  `Resolve` gibt den Pfad unter `Primary`, wenn dort eine Datei **oder ein
  Verzeichnis** liegt, sonst den unter `Fallback`; ist `Fallback` leer, immer
  `Primary`. `WritePath` gibt immer `Primary` — geschrieben wird nie ins
  Altverzeichnis.

**Warum das eine eigene Aufgabe ist:** `config/legacy.go:33` schickt heute die
Leser aus 1b-1 — `search`, `status`, `catalog`, `read`, `neighbors` — fest ins
Altverzeichnis, begründet damit, dass loomux keinen eigenen Schreiber hat. Ab
Task 4 hat es einen. Ohne diese Umstellung schriebe `reindex` neue Register
und `search` läse die alten.

- [ ] **Step 1: Write the failing test**

```go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

func TestArtifactLookupPrefersPrimary(t *testing.T) {
	primary, fallback := t.TempDir(), t.TempDir()
	relative := filepath.Join("areas", "project-loomux", "_identities.tsv")
	mustWrite(t, filepath.Join(primary, relative), "new")
	mustWrite(t, filepath.Join(fallback, relative), "old")

	lookup := config.ArtifactLookup{Primary: primary, Fallback: fallback}
	if got, want := lookup.Resolve(relative), filepath.Join(primary, relative); got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

func TestArtifactLookupFallsBackWhenPrimaryHasNothing(t *testing.T) {
	primary, fallback := t.TempDir(), t.TempDir()
	relative := filepath.Join("areas", "project-space", "index.md")
	mustWrite(t, filepath.Join(fallback, relative), "old")

	lookup := config.ArtifactLookup{Primary: primary, Fallback: fallback}
	if got, want := lookup.Resolve(relative), filepath.Join(fallback, relative); got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

// Fehlt die Datei auf beiden Seiten, gilt der neue Ort: der Fehler, den der
// Aufrufer dann sieht, nennt den Pfad, an dem sie liegen soll.
func TestArtifactLookupNamesPrimaryWhenNeitherHasIt(t *testing.T) {
	primary, fallback := t.TempDir(), t.TempDir()
	lookup := config.ArtifactLookup{Primary: primary, Fallback: fallback}
	if got, want := lookup.Resolve("missing.tsv"), filepath.Join(primary, "missing.tsv"); got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

// Ein Verzeichnis zählt wie eine Datei: `areas/<scope>/` ist der Ort, den ein
// Bereich belegt, und ein leeres Verzeichnis dort ist eine Aussage.
func TestArtifactLookupAcceptsADirectory(t *testing.T) {
	primary, fallback := t.TempDir(), t.TempDir()
	relative := filepath.Join("areas", "project-loomux")
	if err := os.MkdirAll(filepath.Join(primary, relative), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	lookup := config.ArtifactLookup{Primary: primary, Fallback: fallback}
	if got, want := lookup.Resolve(relative), filepath.Join(primary, relative); got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

func TestArtifactLookupWritesOnlyToPrimary(t *testing.T) {
	primary, fallback := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(fallback, "stamp.txt"), "old")
	lookup := config.ArtifactLookup{Primary: primary, Fallback: fallback}
	if got, want := lookup.WritePath("stamp.txt"), filepath.Join(primary, "stamp.txt"); got != want {
		t.Fatalf("WritePath = %q, want %q", got, want)
	}
}

// Ohne Altverzeichnis gibt es nichts zurückzufallen.
func TestArtifactLookupWithoutFallback(t *testing.T) {
	primary := t.TempDir()
	lookup := config.ArtifactLookup{Primary: primary}
	if got, want := lookup.Resolve("x"), filepath.Join(primary, "x"); got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestArtifactLookup -v`
Expected: FAIL, `undefined: config.ArtifactLookup`

- [ ] **Step 3: Write minimal implementation**

```go
package config

import (
	"os"
	"path/filepath"
)

// ArtifactLookup findet eine Zustandsdatei der Pflegeschicht: neu zuerst,
// alt als Rückfall.
//
// Bis Stufe 3 lagen Artefakte und Stempel allein im Verzeichnis von
// ultra-brain, weil loomux keinen Schreiber hatte. Seit `reindex` und
// `reconcile` hat es einen, und von da an gilt: geschrieben wird nur neu,
// gelesen wird neu und, solange dort nichts liegt, alt. `loomux migrate`
// (Stufe 4) zieht den Rest um; danach ist Fallback leer und diese Struktur
// eine Hülle um einen Pfad.
type ArtifactLookup struct {
	Primary  string
	Fallback string
}

// NewArtifactLookup nimmt die beiden Orte, die diese Maschine erklärt.
func NewArtifactLookup() ArtifactLookup {
	return ArtifactLookup{Primary: StateDir(), Fallback: LegacyBrainDirUntilStage3()}
}

// Resolve gibt den Ort, an dem relative zu lesen ist.
func (l ArtifactLookup) Resolve(relative string) string {
	primary := filepath.Join(l.Primary, relative)
	if l.Fallback == "" {
		return primary
	}
	if _, err := os.Stat(primary); err == nil {
		return primary
	}
	fallback := filepath.Join(l.Fallback, relative)
	if _, err := os.Stat(fallback); err == nil {
		return fallback
	}
	// Weder hier noch da: der neue Ort ist der, den die Fehlermeldung des
	// Aufrufers nennen soll.
	return primary
}

// WritePath gibt den Ort, an den relative zu schreiben ist — immer der neue.
func (l ArtifactLookup) WritePath(relative string) string {
	return filepath.Join(l.Primary, relative)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -cover`
Expected: PASS

- [ ] **Step 5: Kommentar in `legacy.go` nachziehen**

`LegacyBrainDirUntilStage3`s Doku sagt heute, loomux habe keinen eigenen
Schreiber. Ersetze den Absatz durch:

```go
// LegacyBrainDirUntilStage3 ist das Zustandsverzeichnis von ultra-brain. Seit
// Stufe 3a ist es der **Rückfall**, nicht der Ort: ArtifactLookup liest neu
// zuerst und fällt hierher zurück, solange `loomux migrate` (Stufe 4) den
// Bestand nicht umgezogen hat. Geschrieben wird hierher nie.
// Die Registry kommt nicht von hier; sie ist StateDirs.
```

- [ ] **Step 6: Die 1b-1-Leser auf den Lookup umstellen**

Die beiden Aufrufer sind `internal/cli/brain.go:55` und
`internal/cli/serve.go:109`. Sie reichen heute `legacyDir` durch bis
`braincatalog.ReadAreaCatalog`, `reader.ReadDocument`,
`privacy.VisibleAreas` und `search.ReadLastRun`. Ersetze den durchgereichten
`legacyDir string` **nicht** durch eine neue Signatur in jedem Paket — gib
stattdessen `config.NewArtifactLookup().Resolve(...)` dort hinein, wo der Pfad
heute aus `legacyDir` zusammengesetzt wird, und lasse die Signaturen stehen,
indem `legacyDir` zum Feld `Fallback` wird.

Konkret, eine Stelle je Paket:
- `internal/brain/catalog/area.go:20` (`AreaArtifactDir`)
- `internal/brain/reader` (`ReadDocument`)
- `internal/brain/privacy` (`VisibleAreas`)
- `internal/brain/search/stamp.go:27` (`ReadLastRun`)

- [ ] **Step 7: Der Test, der die Umstellung überhaupt beweist**

Die 1b-1-Fallsuite reicht dafür **nicht**: setzt jede ihrer Welten
`LOOMUX_LEGACY_BRAIN_DIR` und liegt unter `LOOMUX_STATE_DIR` nichts, dann
antwortet der Rückfall dasselbe wie der alte feste Griff — der Test wäre vor
und nach der Umstellung grün und zeigte nichts.

Schreib darum einen Test, der **beide** Seiten belegt und sie unterscheidbar
macht:

```go
// Liegt unter beiden Orten ein Register, gewinnt das neue. Ohne diesen Test
// wäre die Umstellung unbeweisbar: mit leerem Primary antwortet der Rückfall
// dasselbe wie der feste Griff ins Altverzeichnis.
func TestReadersPreferTheNewStateDir(t *testing.T) {
	primary, legacy := t.TempDir(), t.TempDir()
	relative := filepath.Join("areas", "project-a", "index.md")
	mustWrite(t, filepath.Join(primary, relative), "# new\n")
	mustWrite(t, filepath.Join(legacy, relative), "# old\n")
	t.Setenv("LOOMUX_STATE_DIR", primary)
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", legacy)

	out, _, code := runCLI(t, "catalog", "--scope", "project/a")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out, "new") {
		t.Fatalf("the reader answered from the legacy directory:\n%s", out)
	}
}
```

**Hinweis:** Der Test braucht eine Registry unter `primary`, die
`project/a` kennt, und ein Manifest im Bereich. Bau die Welt mit dem
vorhandenen Helfer des `cli`-Pakets; ist die Verdrahtung von `catalog` dafür
zu umständlich, leg den Test stattdessen in `internal/brain/catalog` auf
`ReadAreaCatalog` direkt — die Aussage bleibt dieselbe.

- [ ] **Step 8: Die Fallsuite 1b-1 muss grün bleiben**

Run: `go test ./internal/cli/ -run TestCases1b1 -v`
Expected: PASS. Das ist der Nachweis, dass die Umstellung die Leser nicht
bricht — nicht der Nachweis, dass sie wirkt; der steht in Step 7.

- [ ] **Step 9: Commit**

```bash
git add internal/config/ internal/brain/
```

```bash
git commit -m "feat(config): read state new-first, legacy as fallback"
```

---

### Task 3: `internal/config` — die Registry schreiben

**Files:**
- Create: `internal/config/registrywrite.go`
- Test: `internal/config/registrywrite_test.go`

**Interfaces:**
- Consumes: `config.Area`, `config.ReadRegistry` (`internal/config/registry.go:57`),
  `lock.Acquire`, `lock.ReplaceText` (Task 1).
- Produces:
  ```go
  func RenderRegistry(areas []Area) string
  func WriteRegistry(stateDir string, areas []Area) error
  func AddArea(stateDir string, area Area) error
  ```
  `RenderRegistry` schreibt `[[area]]`-Tabellen in stabiler Reihenfolge (nach
  `Scope` sortiert) und lässt jede optionale Flagge weg, die `false` ist.
  `WriteRegistry` nimmt die Sperre `<stateDir>/registry.lock`, prüft das
  Ergebnis mit dem Leser und schreibt es über `ReplaceText`. `AddArea` liest,
  hängt an, schreibt — und gibt einen Fehler, wenn der `Scope` schon da ist.

**Warum die Prüfung vor dem Schreiben:** `ReadRegistry` verweigert eine
Registry, die es nicht ganz benutzen kann (doppelter Scope, kollidierendes
Zustandsverzeichnis, zwei `signpost`). Eine Schreibseite, die diese Regeln
nicht kennt, könnte eine Datei hinterlassen, die niemand mehr lesen kann.
Darum wird das gerenderte Ergebnis vor dem Tausch durch denselben Leser
geschickt.

- [ ] **Step 1: Write the failing test**

```go
package config_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

func TestRenderRegistrySortsByScope(t *testing.T) {
	text := config.RenderRegistry([]config.Area{
		{Scope: "project/zeta", Path: "C:/z"},
		{Scope: "project/alpha", Path: "C:/a"},
	})
	if strings.Index(text, "project/alpha") > strings.Index(text, "project/zeta") {
		t.Fatalf("areas are not sorted:\n%s", text)
	}
}

// Eine Flagge, die false ist, steht nicht da: die Datei eines gewöhnlichen
// Bereichs bleibt die, die sie vor der Flagge war.
func TestRenderRegistryOmitsFalseFlags(t *testing.T) {
	text := config.RenderRegistry([]config.Area{{Scope: "project/a", Path: "C:/a"}})
	for _, key := range []string{"readonly", "signpost", "shared", "workspace", "wiki"} {
		if strings.Contains(text, key) {
			t.Fatalf("rendered %q for a plain area:\n%s", key, text)
		}
	}
}

func TestWriteRegistryRoundTrips(t *testing.T) {
	stateDir := t.TempDir()
	want := []config.Area{
		{Scope: "project/loomux", Path: "C:/Users/x/loomux", WikiPath: "docs/wiki"},
		{Scope: "project/space", Path: "C:/Users/x/space", ReadOnly: true, Signpost: true},
	}
	if err := config.WriteRegistry(stateDir, want); err != nil {
		t.Fatalf("WriteRegistry: %v", err)
	}
	got, err := config.ReadRegistry(stateDir)
	if err != nil {
		t.Fatalf("ReadRegistry: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("read %d areas, wrote %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("area %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// Eine Registry, die der Leser ablehnen würde, wird gar nicht erst
// geschrieben: die alte Datei bleibt stehen.
func TestWriteRegistryRefusesWhatTheReaderWouldReject(t *testing.T) {
	stateDir := t.TempDir()
	first := []config.Area{{Scope: "project/a", Path: "C:/a"}}
	if err := config.WriteRegistry(stateDir, first); err != nil {
		t.Fatalf("WriteRegistry: %v", err)
	}
	broken := []config.Area{
		{Scope: "project/a", Path: "C:/a", Signpost: true},
		{Scope: "project/b", Path: "C:/b", Signpost: true},
	}
	if err := config.WriteRegistry(stateDir, broken); err == nil {
		t.Fatal("WriteRegistry: want error for two signposts")
	}
	got, err := config.ReadRegistry(stateDir)
	if err != nil || len(got) != 1 {
		t.Fatalf("the old registry did not survive: %v, %d areas", err, len(got))
	}
}

func TestAddAreaAppends(t *testing.T) {
	stateDir := t.TempDir()
	if err := config.WriteRegistry(stateDir, []config.Area{{Scope: "project/a", Path: "C:/a"}}); err != nil {
		t.Fatalf("WriteRegistry: %v", err)
	}
	if err := config.AddArea(stateDir, config.Area{Scope: "project/b", Path: "C:/b"}); err != nil {
		t.Fatalf("AddArea: %v", err)
	}
	got, _ := config.ReadRegistry(stateDir)
	if len(got) != 2 {
		t.Fatalf("read %d areas, want 2", len(got))
	}
}

// Ein zweiter Bereich desselben Scope ist ein Fehler, kein Überschreiben.
func TestAddAreaRefusesADuplicateScope(t *testing.T) {
	stateDir := t.TempDir()
	area := config.Area{Scope: "project/a", Path: "C:/a"}
	if err := config.AddArea(stateDir, area); err != nil {
		t.Fatalf("AddArea: %v", err)
	}
	err := config.AddArea(stateDir, config.Area{Scope: "project/a", Path: "C:/other"})
	if err == nil || !strings.Contains(err.Error(), "project/a") {
		t.Fatalf("AddArea: want an error naming the scope, got %v", err)
	}
}

// Auf eine leere Maschine schreibt AddArea die erste Registry.
func TestAddAreaCreatesTheRegistry(t *testing.T) {
	stateDir := t.TempDir()
	if err := config.AddArea(stateDir, config.Area{Scope: "project/a", Path: "C:/a"}); err != nil {
		t.Fatalf("AddArea: %v", err)
	}
	if _, err := config.ReadRegistry(stateDir); err != nil {
		t.Fatalf("ReadRegistry: %v", err)
	}
	_ = filepath.Join(stateDir, "registry.toml")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run 'TestRenderRegistry|TestWriteRegistry|TestAddArea' -v`
Expected: FAIL, `undefined: config.RenderRegistry`

- [ ] **Step 3: Write minimal implementation**

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/lock"
)

// registryLockName ist die Sperrdatei neben der Registry. Ein eigener Name
// statt der Registry selbst: die Sperre hält einen Bytebereich, und ein
// gesperrter Bereich wäre für Leser unlesbar, solange er etwas bedeutet.
const registryLockName = "registry.lock"

// RenderRegistry schreibt die Bereiche als [[area]]-Tabellen.
//
// Stabil nach Scope sortiert, damit ein Paritätsfall die Datei byteweise
// vergleichen kann; eine Flagge, die false ist, steht nicht da, damit die
// Datei eines gewöhnlichen Bereichs die bleibt, die sie vor der Flagge war.
func RenderRegistry(areas []Area) string {
	sorted := append([]Area(nil), areas...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Scope < sorted[j].Scope })
	var out strings.Builder
	for i, area := range sorted {
		if i > 0 {
			out.WriteString("\n")
		}
		out.WriteString("[[area]]\n")
		fmt.Fprintf(&out, "scope = %s\n", quoteTOML(area.Scope))
		fmt.Fprintf(&out, "path = %s\n", quoteTOML(area.Path))
		if area.WikiPath != "" {
			fmt.Fprintf(&out, "wiki = %s\n", quoteTOML(area.WikiPath))
		}
		for _, flag := range []struct {
			name string
			set  bool
		}{
			{"readonly", area.ReadOnly},
			{"signpost", area.Signpost},
			{"shared", area.Shared},
			{"workspace", area.Workspace},
		} {
			if flag.set {
				fmt.Fprintf(&out, "%s = true\n", flag.name)
			}
		}
	}
	return out.String()
}

// WriteRegistry ersetzt die Registry unter der Sperre und nur, wenn der Leser
// das Ergebnis annimmt.
//
// Die Gegenprobe vor dem Tausch ist der Punkt: ReadRegistry verweigert eine
// Registry, die es nicht ganz benutzen kann, und eine Schreibseite ohne diese
// Regeln könnte eine Datei hinterlassen, die niemand mehr liest.
func WriteRegistry(stateDir string, areas []Area) error {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return err
	}
	handle, err := lock.Acquire(filepath.Join(stateDir, registryLockName))
	if err != nil {
		return err
	}
	defer handle.Release()
	return writeRegistryLocked(stateDir, areas)
}

// writeRegistryLocked ist der Rumpf ohne Sperre, damit AddArea lesen und
// schreiben kann, ohne sie zwischendurch abzugeben.
func writeRegistryLocked(stateDir string, areas []Area) error {
	text := RenderRegistry(areas)
	if err := checkRegistryText(stateDir, text); err != nil {
		return err
	}
	return lock.ReplaceText(filepath.Join(stateDir, registryName), text)
}

// checkRegistryText schickt den gerenderten Text durch den Leser, in einem
// eigenen Verzeichnis, damit die stehende Registry unberührt bleibt.
func checkRegistryText(stateDir, text string) error {
	scratch, err := os.MkdirTemp(stateDir, ".registry-check-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(scratch)
	if err := os.WriteFile(filepath.Join(scratch, registryName), []byte(text), 0o644); err != nil {
		return err
	}
	if _, err := ReadRegistry(scratch); err != nil {
		return fmt.Errorf("refusing to write a registry the reader rejects: %w", err)
	}
	return nil
}

// AddArea nimmt einen Bereich in die Registry auf.
func AddArea(stateDir string, area Area) error {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return err
	}
	handle, err := lock.Acquire(filepath.Join(stateDir, registryLockName))
	if err != nil {
		return err
	}
	defer handle.Release()
	areas, err := readRegistryIfPresent(stateDir)
	if err != nil {
		return err
	}
	for _, standing := range areas {
		if standing.Scope == area.Scope {
			return fmt.Errorf("%s: scope %q is already registered", stateDir, area.Scope)
		}
	}
	return writeRegistryLocked(stateDir, append(areas, area))
}

// readRegistryIfPresent gibt eine leere Liste für die Maschine, auf der noch
// nie ein Bereich angemeldet wurde, und reicht jeden anderen Fehler weiter.
func readRegistryIfPresent(stateDir string) ([]Area, error) {
	areas, err := ReadRegistry(stateDir)
	if err != nil && os.IsNotExist(err) {
		return nil, nil
	}
	return areas, err
}
```

**Hinweis für den Umsetzer:** `quoteTOML` gibt es womöglich schon in
`internal/config/tomlvalue.go`. Prüfe das zuerst (`grep -n "func quote" internal/config/`);
wenn ja, benutze die vorhandene Form, statt eine zweite zu schreiben. Prüfe
auch, ob `ReadRegistry` bei fehlender Datei wirklich einen `os.IsNotExist`
tragenden Fehler gibt — wenn es ihn einpackt, braucht
`readRegistryIfPresent` ein `errors.Is`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -cover`
Expected: PASS, 100 % für die neuen Zeilen

- [ ] **Step 5: Commit**

```bash
git add internal/config/registrywrite.go internal/config/registrywrite_test.go
```

```bash
git commit -m "feat(config): write the registry under a lock"
```

---

### Task 4: `internal/brain/index` — der Umzug aus `pkg/index`

**Files:**
- Create: `internal/brain/index/walk.go`, `document.go`, `reindex.go`,
  `qmdconfig.go`, `catalogwrite.go`
- Test: die mitgezogenen `*_test.go` derselben Namen
- Quelle: `~/Documents/#GIT/ultra-brain/pkg/index/` (Tag `loomux-3-source`)

**Interfaces:**
- Consumes: `config.Area`, `config.ArtifactLookup` (Task 2),
  `internal/brain/identity` (seit 1b-1: `ReadIdentities`, `RenderIdentities`,
  `ContentHash`, `MatchRenames`, `NewDocID`), `internal/brain/catalog`
  (Leseseite), `internal/brain/privacy`, `internal/brain/search`,
  `internal/brain/graph`.
- Produces:
  ```go
  func FindFiles(area config.Area, manifest config.Manifest) ([]string, error)
  func RenderCatalog(...) string
  func WriteCatalogs(area config.Area, documents []Document, targetDir string) error
  func ReadIntro(path string) (string, error)
  func Reindex(registryPath, stateDir string) error
  ```

**Was genau umzieht und was nicht.** `pkg/index/` ohne Tests sind 1.496
Zeilen. **`identity.go` (196) zieht nicht um** — es ist seit 1b-1 als
`internal/brain/identity/identity.go` da, mit denselben Funktionen plus
`parseRevision`. Von **`catalog.go` (194) zieht nur die Schreibseite um**:
`RenderCatalog`, `WriteCatalogs`, `ReadIntro`, `formatDestination`,
`escapeLinkText`, `writeIfChanged`, `parentOf`; die Leseseite
(`ReadAreaCatalog`, `RenderRootCatalog`) ist als `internal/brain/catalog` da.
Voll um ziehen: `walk.go` (225), `document.go` (139), `reindex.go` (310),
`qmd_config.go` (432).

`pkg/index` importiert nur `pkg/{config,privacy,search,graph}` — alle vier
gibt es in loomux. Es gibt keine weitere Abhängigkeit aufzulösen.

- [ ] **Step 1: Die Quelldateien kopieren**

```bash
cp ~/Documents/#GIT/ultra-brain/pkg/index/walk.go internal/brain/index/walk.go
```

Ebenso `document.go`, `reindex.go`, `qmd_config.go` → `qmdconfig.go` und
`catalog.go` → `catalogwrite.go`. Die Tests derselben Namen ebenfalls.

- [ ] **Step 2: Die Importe und den Paketnamen umschreiben**

`package index` bleibt. Ersetze in allen kopierten Dateien:
- `github.com/xidus90/ultra-brain/pkg/config` → `github.com/xidus90/loomux/internal/config`
- `github.com/xidus90/ultra-brain/pkg/privacy` → `github.com/xidus90/loomux/internal/brain/privacy`
- `github.com/xidus90/ultra-brain/pkg/search` → `github.com/xidus90/loomux/internal/brain/search`
- `github.com/xidus90/ultra-brain/pkg/graph` → `github.com/xidus90/loomux/internal/brain/graph`

Aus `catalogwrite.go` die Leseseite **entfernen** und stattdessen
`internal/brain/catalog` importieren, wo sie gebraucht wird. Aus allen Dateien
die `Identity`-Formen entfernen und `internal/brain/identity` importieren.

- [ ] **Step 3: Übersetzen lassen und die Lücken schließen**

Run: `go build ./internal/brain/index/`
Expected: zuerst Fehler über fehlende Typen. Jeder Fehler hat genau eine
richtige Antwort: entweder der Typ liegt in einem loomux-Paket (importieren)
oder er gehört zur Schreibseite (mitziehen). **Nichts neu erfinden.**

- [ ] **Step 4: Den Artefaktort auf den Lookup umstellen**

Überall, wo der umgezogene Code `stateDir` zu `areas/<scope>/…`
zusammensetzt, ist der Schreibpfad `lookup.WritePath(...)` und der Lesepfad
`lookup.Resolve(...)`. Das ist die Stelle, an der Task 2 wirksam wird.

**Auflage aus Task 2 — ein Bereich wird als Ganzes sichtbar oder gar nicht.**
Die Auflösung aus Task 2 greift auf **Verzeichnisebene**: sobald unter
`<state>/areas/<scope>/` irgendetwas liegt, kommen **alle** Lesevorgänge
dieses Bereichs von dort. Und `privacy.VisibleAreas` gibt beim ersten
fehlenden Manifest für **alle** Bereiche auf, nicht nur für den einen — ein
halb geschriebenes `areas/<scope>/` macht damit den ganzen Tresor
unbeantwortbar, solange es liegt.

`Reindex` schreibt einen Bereich darum **erst vollständig in ein
Staging-Verzeichnis** neben dem Ziel und benennt es dann hinein. `internal/lock`
kann das heute auf Dateiebene (`ReplaceText`); was hier gebraucht wird, ist
dieselbe Bewegung auf Verzeichnisebene. Schreib einen Test, der einen
Abbruch mitten im Schreiben nachstellt und prüft, dass danach entweder der
alte Stand vollständig dasteht oder gar keiner — nie die Hälfte.

- [ ] **Step 5: Tests laufen lassen**

Run: `go test ./internal/brain/index/ -cover`
Expected: PASS. Coverage wird unter 100 % liegen — das ist erwartet, siehe
Step 6.

- [ ] **Step 6: Auf 100 % heben**

Run: `go test ./internal/brain/index/ -coverprofile=cover.out`
dann: `go tool cover -func=cover.out`

Für jede Zeile unter 100 %: entweder einen Test nachreichen oder einen
Ausschluss **mit Begründung im Code** setzen. Die Regel der Spec: umgezogen
heißt nicht fertig.

- [ ] **Step 7: Commit**

```bash
git add internal/brain/index/
```

```bash
git commit -F "$TEMP/loomux-commit-body.txt"
```

mit dieser Nachricht in der Datei (**nicht** `.git/COMMIT_BODY` — dies ist ein
Worktree, dort ist `.git` eine Datei):

```
feat(index): port the index run from ultra-brain

walk, document, reindex, qmd config and the catalog write side move across
with their tests and are raised to full coverage. identity and the catalog
read side already arrived with stage 1b-1 and are imported, not copied.
```

---

### Task 5: `loomux reindex` und `loomux embed`

**Files:**
- Create: `internal/cli/index.go`
- Test: `internal/cli/index_test.go`
- Modify: `internal/cli/commands.go` (die beiden Befehle eintragen)

**Interfaces:**
- Consumes: `index.Reindex` (Task 4), `search.QmdMcpPort.Embed`
  (`internal/brain/search/mcp.go:239`), `config.ReadRegistry`.
- Produces: `func runReindex(args []string, stdout, stderr io.Writer) int`
  und `func runEmbed(...) int`, beide über die Einstiegsform der Spec
  (`Run(args, stdin, stdout, stderr) int`), damit die Fälle im Prozess laufen
  und `go test -cover` sie zählt.

**Noch ohne Auffangdurchgang.** Der kommt in Task 14, wenn `reconcile` steht.
Diesen Task hier nicht damit vermischen: ein Reviewer soll den Indexlauf
annehmen können, ohne über das Tor zu entscheiden.

- [ ] **Step 1: Write the failing test**

```go
package cli_test

import (
	"strings"
	"testing"
)

// Ein leerer Zustand ist kein Fehler: eine Maschine ohne angemeldeten Bereich
// hat nichts zu indizieren, und das ist eine Aussage, kein Zusammenbruch.
func TestReindexOnAnEmptyState(t *testing.T) {
	state := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", state)
	out, errOut, code := runCLI(t, "reindex")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if strings.TrimSpace(out) == "" && strings.TrimSpace(errOut) == "" {
		t.Fatal("reindex said nothing at all")
	}
}

// Fehlt qmd, nennt embed den Namen und den Installationsbefehl und ist rot --
// nie eine leere Erfolgsmeldung.
func TestEmbedWithoutQmdIsRed(t *testing.T) {
	state := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", state)
	t.Setenv("PATH", t.TempDir())
	_, errOut, code := runCLI(t, "embed")
	if code == 0 {
		t.Fatalf("exit = 0 without qmd, stderr = %s", errOut)
	}
	if !strings.Contains(errOut, "qmd") {
		t.Fatalf("stderr does not name qmd: %s", errOut)
	}
}
```

**Hinweis:** `runCLI` ist der vorhandene Testhelfer des Pakets. Suche ihn
(`grep -n "func runCLI" internal/cli/`) und benutze ihn, statt einen zweiten
zu schreiben; heißt er anders, passe die Tests an seinen Namen an.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run 'TestReindex|TestEmbed' -v`
Expected: FAIL — unbekannter Unterbefehl

- [ ] **Step 3: Die Befehle eintragen und den Lauf anhängen**

`internal/cli/index.go` hält die beiden Einstiege; die Argumentprüfung folgt
dem Muster der Nachbarn in `internal/cli/`. `reindex` nimmt `--registry` und
`--state-dir` wie die Referenz (`cli.py:463-466`), `embed` dieselben.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run 'TestReindex|TestEmbed' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/index.go internal/cli/index_test.go internal/cli/commands.go
```

```bash
git commit -m "feat(cli): add reindex and embed"
```

---

### Task 6: `internal/brain/vcs` — die Leseseite

**Files:**
- Create: `internal/brain/vcs/vcs.go`
- Test: `internal/brain/vcs/vcs_test.go`
- Referenz: `src/brain/maintenance/vcs.py:285-500`

**Interfaces:**
- Consumes: `internal/gitenv` (die saubere Umgebung für einen Git-Aufruf).
- Produces:
  ```go
  func ShowBlob(directory, relative string) ([]byte, error)   // nil, nil wenn HEAD den Pfad nicht kennt
  func ChangedPaths(directory, first, last string) ([]string, error)
  func CommitSubjects(directory, first, last string) ([]string, error)
  func RepositoryRoot(directory string) (string, error)       // "" wenn kein Repo
  func CommonDirectory(directory string) (string, error)
  func HooksDirectory(repo string) (string, error)
  ```
  Kein `CommitPaths` — die Schreibseite ist 3b.

**Warum eine saubere Umgebung:** `vcs._clean_env` (`vcs.py:311-319`) streicht
die Git-Variablen, die eine laufende Sitzung setzt; ein Aufruf, der sie erbt,
läse das Repo einer fremden Operation. `internal/gitenv` tut dasselbe in Go —
prüfe seine Form (`grep -n "^func" internal/gitenv/*.go`) und benutze sie.

- [ ] **Step 1: Write the failing test**

```go
package vcs_test

import (
	"os/exec"
	"testing"

	"github.com/xidus90/loomux/internal/brain/vcs"
)

// Der einfachste echte Fall: ein Repo mit einem Commit gibt den Inhalt der
// committeten Datei zurück.
func TestShowBlobReturnsTheCommittedBytes(t *testing.T) {
	repo := newRepo(t, map[string]string{"a.md": "first\n"})
	got, err := vcs.ShowBlob(repo, "a.md")
	if err != nil {
		t.Fatalf("ShowBlob: %v", err)
	}
	if string(got) != "first\n" {
		t.Fatalf("blob = %q", got)
	}
}

// Ein Pfad, den HEAD nicht kennt, ist kein Fehler: es gibt keine Grundlinie,
// und der Aufrufer sagt das im Paket laut.
func TestShowBlobReturnsNilForAnUnknownPath(t *testing.T) {
	repo := newRepo(t, map[string]string{"a.md": "first\n"})
	got, err := vcs.ShowBlob(repo, "never.md")
	if err != nil {
		t.Fatalf("ShowBlob: %v", err)
	}
	if got != nil {
		t.Fatalf("blob = %q, want nil", got)
	}
}

// Ein Verzeichnis ohne Repo ist kein Fehler, sondern "kein Repo".
func TestRepositoryRootOutsideARepo(t *testing.T) {
	root, err := vcs.RepositoryRoot(t.TempDir())
	if err != nil {
		t.Fatalf("RepositoryRoot: %v", err)
	}
	if root != "" {
		t.Fatalf("root = %q, want \"\"", root)
	}
}

// newRepo legt ein Repo mit einem Commit an. Ein echtes git, kein Stub: die
// Fälle dieser Stufe laufen ebenfalls gegen echtes git, und ein Stub würde
// genau die Unterschiede verstecken, die der Paritätsnachweis sucht.
func newRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	// Anlegen, Dateien schreiben, `git init`, `git add`, `git commit` mit
	// fester Identität und -c commit.gpgsign=false.
	return ""
}
```

**Hinweis für den Umsetzer:** `newRepo` ist hier absichtlich als Rumpf
gezeigt. Schau zuerst, ob `internal/gitwork` oder `internal/cases` bereits
einen Repo-Helfer für Tests hat (`grep -rn "git init" internal/ --include=*_test.go`);
wenn ja, benutze den. Wenn nicht, schreib ihn hier vollständig aus, mit
`-c user.name`, `-c user.email` und `-c commit.gpgsign=false`, damit er auf
jeder Maschine läuft.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/brain/vcs/ -v`
Expected: FAIL, das Paket gibt es nicht

- [ ] **Step 3: Write the implementation**

Nach `vcs.py`, Funktion für Funktion. Die drei Regeln, die dort im Kommentar
stehen und die mit umziehen:
- Ein Pfad außerhalb des Repos wird abgelehnt (`_reject_path_outside`).
- Eine laufende Git-Operation (Rebase, Merge, Cherry-Pick) wird abgelehnt
  (`_reject_operation_in_progress`) — auch auf der Leseseite, weil ein Repo
  mitten in einer Operation kein verlässliches HEAD hat.
- `ShowBlob` fragt `git show HEAD:<pfad>` und gibt `nil, nil`, wenn git den
  Pfad nicht kennt; jeder andere Exit-Code ist ein Fehler.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/brain/vcs/ -cover`
Expected: PASS, 100 %

- [ ] **Step 5: Commit**

```bash
git add internal/brain/vcs/
```

```bash
git commit -m "feat(vcs): read git for the maintenance layer"
```

---

### Task 7: `internal/brain/maintenance/events.go` — das Ereignisprotokoll

**Files:**
- Create: `internal/brain/maintenance/events.go`
- Test: `internal/brain/maintenance/events_test.go`
- Referenz: `src/brain/maintenance/merge_events.py:118-136, 187-270, 535-604`

**Interfaces:**
- Consumes: `config.ArtifactLookup` (Task 2), `lock.ReplaceText` (Task 1).
- Produces:
  ```go
  type MergeEvent struct {
      Repo   string    // ein String, kein Pfadtyp: er kommt aus der Textdatei,
                       // die der Hook schrieb, und wird nicht normalisiert
      First  string    // der Commit vor dem Merge
      Last   string    // der Commit danach
      Branch string
      At     time.Time // geparst aus dem ISO-8601-Stempel der Zeile
  }
  // Key ist, was zwei Aufzeichnungen zu demselben Merge macht: der Bereich,
  // in einem Repository (merge_events.MergeEvent.key).
  func (e MergeEvent) Key() [3]string { return [3]string{e.Repo, e.First, e.Last} }
  func EventsPath(lookup config.ArtifactLookup) string
  func ReadEvents(lookup config.ArtifactLookup) ([]MergeEvent, error)
  func DropEvent(lookup config.ArtifactLookup, event MergeEvent) error
  ```
  **Kein `RecordEvent`.** Sein einziger Schreiber ist der Shell-Hook aus
  Stufe 4, und die Fälle dieser Stufe bringen die Protokolldatei fertig in
  ihrer `world/` mit. Eine Schreibfunktion, die nur ihr eigener Test benutzt,
  ist kein Code, den diese Stufe trägt.

**Kein `Scope` und kein `Area` im Ereignis.** Die Zuordnung zu einem Bereich
macht Task 12 über das Repository (`_resolve_event`, `reconcile.py:574-579`,
gegen `by_repository`). Ein Feld dafür zu erfinden ginge an der Referenz
vorbei.

**Das Format ist eine Zeile mit fünf durch Tabulator getrennten Feldern:**
`repo \t first \t last \t branch \t stamp \n` (`merge_events.py:542`). Eine
Zeile, die nicht fünf Felder hat, wird übersprungen — der Hook schreibt
nebenläufig, und eine halbe Zeile ist ein Zustand, den es gibt.

**Abgelegte Ereignisse** stehen in einer zweiten Datei
(`merge_events._dropped_path`). `DropEvent` hängt dort an, statt aus dem
Protokoll zu löschen: der Hook hält es offen, und ein Schreiber, der es
kürzt, verlöre eine Zeile, die zwischen Lesen und Schreiben ankam.

- [ ] **Step 1: Write the failing test**

```go
package maintenance_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/brain/maintenance"
)

func TestReadEventsParsesFiveFields(t *testing.T) {
	lookup := writeEvents(t, "C:/repo\tabc123\tdef456\tmain\t2026-09-19T10:00:00+02:00\n")
	got, err := maintenance.ReadEvents(lookup)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("read %d events, want 1", len(got))
	}
	want := maintenance.MergeEvent{
		Repo: "C:/repo", First: "abc123", Last: "def456",
		Branch: "main", Stamp: "2026-09-19T10:00:00+02:00",
	}
	if got[0] != want {
		t.Fatalf("event = %+v, want %+v", got[0], want)
	}
}

// Eine halbe Zeile ist ein Zustand, den es gibt: der Hook schreibt
// nebenläufig. Sie wird übersprungen, nicht zum Fehler.
func TestReadEventsSkipsAMalformedLine(t *testing.T) {
	lookup := writeEvents(t, "C:/repo\tabc\n"+
		"C:/repo\tabc123\tdef456\tmain\t2026-09-19T10:00:00+02:00\n")
	got, err := maintenance.ReadEvents(lookup)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("read %d events, want 1", len(got))
	}
}

// Kein Protokoll ist kein Fehler: auf einer Maschine ohne Hook gibt es keins.
func TestReadEventsWithoutAFile(t *testing.T) {
	lookup := config.ArtifactLookup{Primary: t.TempDir()}
	got, err := maintenance.ReadEvents(lookup)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("read %d events from nothing", len(got))
	}
}

// Ein abgelegtes Ereignis kommt beim nächsten Lesen nicht zurück.
func TestDropEventHidesIt(t *testing.T) {
	lookup := writeEvents(t, "C:/repo\tabc123\tdef456\tmain\t2026-09-19T10:00:00+02:00\n")
	events, _ := maintenance.ReadEvents(lookup)
	if err := maintenance.DropEvent(lookup, events[0]); err != nil {
		t.Fatalf("DropEvent: %v", err)
	}
	got, err := maintenance.ReadEvents(lookup)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("a dropped event came back: %+v", got)
	}
}

// Das Protokoll selbst wird nicht gekürzt: der Hook hält es offen, und eine
// Zeile, die zwischen Lesen und Schreiben ankommt, ginge sonst verloren.
func TestDropEventLeavesTheLogAlone(t *testing.T) {
	line := "C:/repo\tabc123\tdef456\tmain\t2026-09-19T10:00:00+02:00\n"
	lookup := writeEvents(t, line)
	events, _ := maintenance.ReadEvents(lookup)
	_ = maintenance.DropEvent(lookup, events[0])
	raw, err := os.ReadFile(maintenance.EventsPath(lookup))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(raw) != line {
		t.Fatalf("the log was rewritten: %q", raw)
	}
}

func writeEvents(t *testing.T, content string) config.ArtifactLookup {
	t.Helper()
	primary := t.TempDir()
	lookup := config.ArtifactLookup{Primary: primary}
	path := maintenance.EventsPath(lookup)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return lookup
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/brain/maintenance/ -run 'TestReadEvents|TestDropEvent' -v`
Expected: FAIL, das Paket gibt es nicht

- [ ] **Step 3: Write the implementation**

Nach `merge_events.py`. Die Dateinamen kommen von dort (`events_path`,
`_dropped_path`) und liegen unter `maintenance/` im Zustandsverzeichnis. Ein
abgelegtes Ereignis wird über das Tripel `(repo, first, last)` erkannt
(`merge_events._dropped`, Zeile 557-565) — **nicht** über alle fünf Felder:
der Zeitstempel eines Hooks, der zweimal lief, unterscheidet sich.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/brain/maintenance/ -cover`
Expected: PASS, 100 %

- [ ] **Step 5: Commit**

```bash
git add internal/brain/maintenance/events.go internal/brain/maintenance/events_test.go
```

```bash
git commit -m "feat(maintenance): read and drop merge events"
```

---

### Task 8: `internal/brain/maintenance/case.go` — die Fallakte

**Files:**
- Create: `internal/brain/maintenance/case.go`
- Test: `internal/brain/maintenance/case_test.go`
- Referenz: `src/brain/maintenance/case.py` (253 Zeilen, vollständig)

**Interfaces:**
- Consumes: `lock.ReplaceText` (Task 1).
- Produces:
  ```go
  type SourceState struct {
      DocID       string
      Revision    int
      ContentHash string
  }
  type Case struct {
      ID, Area, Target, TargetHash, State, Trigger, Weight string
      Created  time.Time
      Sources  []SourceState
      Note     string // "" heißt: kein note-Schlüssel
      SupersededProposal string
      Manual, LocalOnly  bool
      PromptVersion      string
  }
  func CaseID(area, target string, now time.Time) string
  func CaseDir(reviewRoot, scope, id string) string
  func WriteCase(path string, c Case) (changed bool, err error)
  func ReadCase(path string) (Case, error)
  ```

**Die drei Regeln, die mit umziehen:**

1. **`CaseID` ist deterministisch**, damit ein zweiter Lauf nicht denselben
   Fall zweimal eröffnet: `<letztes Segment des Scope>-<YYYY-MM-DD>-<4 Hex>`,
   wobei die vier Hex-Zeichen die ersten vier des SHA-256 über
   `area + "\n" + target` sind. Über **beide**, nicht nur `target`: zwei
   Bereiche mit gleich benannter Datei kollidierten am selben Tag sonst.
2. **Die Feldreihenfolge in `case.toml` liegt fest** (`case.py:141-171`), und
   eine Flagge, die `false` ist, steht nicht da — so bleibt die Datei eines
   gewöhnlichen Falls die, die sie vor der Flagge war. Reihenfolge: `id`,
   `area`, `target`, `target_hash`, `state`, `trigger`, `weight`, `created`,
   dann optional `note`, `superseded_proposal`, `manual`, `local_only`,
   `prompt_version`, dann je Quelle ein `[[sources]]`-Block mit `doc_id`,
   `revision`, `content_hash`.
3. **Ein `created` ohne Zone wird abgelehnt.** Es liefe durch TOMLs
   local-datetime und käme in der Zone des Lesers zurück — ein stiller
   Unterschied, teurer zu finden als hier abzulehnen.

`WriteCase` gibt zurück, **ob sich etwas geändert hat** (`write_if_changed`):
ein zweiter identischer Schreibvorgang darf nicht als Änderung gemeldet
werden, sonst schickt der Bericht den Leser nach einer Änderung suchen, die
git nicht zeigt.

- [ ] **Step 1: Write the failing test**

```go
package maintenance_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
)

func TestCaseIDIsDeterministic(t *testing.T) {
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	first := maintenance.CaseID("project/loomux", "docs/wiki/a.md", now)
	second := maintenance.CaseID("project/loomux", "docs/wiki/a.md", now)
	if first != second {
		t.Fatalf("%q != %q", first, second)
	}
	if !strings.HasPrefix(first, "loomux-2026-09-19-") {
		t.Fatalf("id = %q", first)
	}
}

// Über area und target zusammen: zwei Bereiche mit gleich benannter Datei
// kollidierten am selben Tag sonst.
//
// Die beiden Bereiche müssen dasselbe letzte Segment tragen. Die erste
// Fassung dieses Tests nahm `project/one` und `project/two` — da unterscheiden
// sich schon die Präfixe, der Digest wird nie erreicht, und ein Digest allein
// über `target` überlebte den Test (gemessen am 2026-09-20, Task 8).
func TestCaseIDSeparatesTwoAreasWithTheSameLastSegment(t *testing.T) {
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	a := maintenance.CaseID("vault/loomux", "a.md", now)
	b := maintenance.CaseID("project/loomux", "a.md", now)
	if a == b {
		t.Fatalf("two areas produced one id: %q", a)
	}
}

func TestWriteCaseRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "case.toml")
	want := maintenance.Case{
		ID: "loomux-2026-09-19-abcd", Area: "project/loomux",
		Target: "docs/wiki/a.md", TargetHash: "sha256:aa",
		State: "source_changed", Trigger: "source_change", Weight: "change",
		Created: time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC),
		Sources: []maintenance.SourceState{{DocID: "d1", Revision: 3, ContentHash: "sha256:bb"}},
	}
	if _, err := maintenance.WriteCase(path, want); err != nil {
		t.Fatalf("WriteCase: %v", err)
	}
	got, err := maintenance.ReadCase(path)
	if err != nil {
		t.Fatalf("ReadCase: %v", err)
	}
	if got.ID != want.ID || got.Target != want.Target || len(got.Sources) != 1 {
		t.Fatalf("case = %+v", got)
	}
	if !got.Created.Equal(want.Created) {
		t.Fatalf("created = %v, want %v", got.Created, want.Created)
	}
}

// Eine Flagge, die false ist, steht nicht in der Datei.
func TestWriteCaseOmitsFalseFlags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "case.toml")
	c := maintenance.Case{
		ID: "a", Area: "project/a", Target: "a.md", TargetHash: "sha256:aa",
		State: "source_changed", Trigger: "source_change", Weight: "change",
		Created: time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC),
	}
	if _, err := maintenance.WriteCase(path, c); err != nil {
		t.Fatalf("WriteCase: %v", err)
	}
	raw, _ := os.ReadFile(path)
	for _, key := range []string{"manual", "local_only", "note", "prompt_version"} {
		if strings.Contains(string(raw), key) {
			t.Fatalf("rendered %q for a plain case:\n%s", key, raw)
		}
	}
}

// Ein zweiter identischer Schreibvorgang ist keine Änderung.
func TestWriteCaseReportsNoChangeTwice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "case.toml")
	c := maintenance.Case{
		ID: "a", Area: "project/a", Target: "a.md", TargetHash: "sha256:aa",
		State: "source_changed", Trigger: "source_change", Weight: "change",
		Created: time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC),
	}
	if changed, _ := maintenance.WriteCase(path, c); !changed {
		t.Fatal("the first write reported no change")
	}
	if changed, _ := maintenance.WriteCase(path, c); changed {
		t.Fatal("the second identical write reported a change")
	}
}

// Ein created ohne Zone liefe durch TOMLs local-datetime und käme in der Zone
// des Lesers zurück.
func TestWriteCaseRefusesANaiveTimestamp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "case.toml")
	c := maintenance.Case{
		ID: "a", Area: "project/a", Target: "a.md", TargetHash: "sha256:aa",
		State: "source_changed", Trigger: "source_change", Weight: "change",
	}
	if _, err := maintenance.WriteCase(path, c); err == nil {
		t.Fatal("WriteCase: want an error for a zero-value created")
	}
}
```

**Hinweis zur Zone in Go:** Python unterscheidet naiv und zonenbehaftet;
Go nicht — jede `time.Time` trägt eine Location. Die entsprechende Ablehnung
ist die **Nullzeit** (`c.Created.IsZero()`): ein Fall, dem niemand einen
Zeitpunkt gegeben hat. Schreib das als Kommentar in den Code, damit der
nächste Leser nicht nach dem naiven Fall sucht.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/brain/maintenance/ -run 'TestCaseID|TestWriteCase|TestReadCase' -v`
Expected: FAIL, `undefined: maintenance.CaseID`

- [ ] **Step 3: Write the implementation**

Nach `case.py`, Feld für Feld. `WriteCase` rendert selbst, statt einen
TOML-Kodierer zu benutzen: die Reihenfolge liegt fest und ein Kodierer
sortiert.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/brain/maintenance/ -cover`
Expected: PASS, 100 %

- [ ] **Step 5: Commit**

```bash
git add internal/brain/maintenance/case.go internal/brain/maintenance/case_test.go
```

```bash
git commit -m "feat(maintenance): read and write a case file"
```

---

### Task 9: `package.go` und `derive.go` — das Paket und die Abhängigen

**Files:**
- Create: `internal/brain/maintenance/package.go`, `internal/brain/maintenance/derive.go`
- Test: `internal/brain/maintenance/package_test.go`, `derive_test.go`
- Referenz: `src/brain/maintenance/package.py` (163), `derive.py` (33)

**Interfaces:**
- Consumes: `maintenance.Case` (Task 8).
- Produces:
  ```go
  type Segment struct {
      Kind  string // "S" (Quelle) oder "D" (Diff/Beleg)
      Name  string
      Body  string
  }
  func RenderPackage(c Case, segments []Segment) string
  func Dependents(wikiPath string) (map[string][]string, error)
  ```

**Warum `package.go` in 3a liegt und nicht in 3b:** `_land_case` schreibt
neben jeder Fallakte ein `package.md` (`reconcile.py:783-800`). Ohne
`RenderPackage` gäbe es keinen vollständigen Fall — und die Belegprüfung aus
3b liest genau dieses Paket. Die **Prüfung** ist 3b; das **Paket** ist 3a.

`Dependents` liest die Wiki-Seiten und gibt je Seite die, die auf sie zeigen.
33 Zeilen in Python; der Go-Code ist nicht viel länger.

- [ ] **Step 1: Write the failing test**

```go
package maintenance_test

import (
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
)

func TestRenderPackageCarriesTheCaseHead(t *testing.T) {
	c := maintenance.Case{
		ID: "loomux-2026-09-19-abcd", Area: "project/loomux",
		Target: "docs/wiki/a.md", TargetHash: "sha256:aa",
		State: "source_changed", Trigger: "source_change", Weight: "change",
		Created: time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC),
	}
	text := maintenance.RenderPackage(c, []maintenance.Segment{
		{Kind: "S", Name: "src/a.go", Body: "package a\n"},
	})
	if !strings.Contains(text, c.ID) {
		t.Fatalf("package does not name the case:\n%s", text)
	}
	if !strings.Contains(text, "src/a.go") {
		t.Fatalf("package does not name the segment:\n%s", text)
	}
}

// Zwei Läufe mit derselben Eingabe geben dasselbe Paket: der Fall wird
// byteweise verglichen.
func TestRenderPackageIsStable(t *testing.T) {
	c := maintenance.Case{
		ID: "a", Area: "project/a", Target: "a.md", TargetHash: "sha256:aa",
		State: "source_changed", Trigger: "source_change", Weight: "change",
		Created: time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC),
	}
	segments := []maintenance.Segment{{Kind: "S", Name: "a", Body: "x"}}
	if maintenance.RenderPackage(c, segments) != maintenance.RenderPackage(c, segments) {
		t.Fatal("two renders differ")
	}
}
```

Für `Dependents` einen Test mit zwei Seiten, von denen eine auf die andere
zeigt, und die Erwartung, dass die Zielseite ihre Quelle nennt.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/brain/maintenance/ -run 'TestRenderPackage|TestDependents' -v`
Expected: FAIL

- [ ] **Step 3: Write the implementation**

Nach `package.py` und `derive.py`. **Das genaue Format des Pakets steht
dort** — lies die Datei und übernimm es Zeichen für Zeichen; die Belegprüfung
aus 3b hängt an den Segmentgrenzen, und eine verschobene Zeile bricht sie.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/brain/maintenance/ -cover`
Expected: PASS, 100 %

- [ ] **Step 5: Commit**

```bash
git add internal/brain/maintenance/package.go internal/brain/maintenance/derive.go internal/brain/maintenance/package_test.go internal/brain/maintenance/derive_test.go
```

```bash
git commit -m "feat(maintenance): render a case package and derive dependents"
```

---

### Task 10: `scan.go` — was sich geändert hat

**Files:**
- Create: `internal/brain/maintenance/scan.go`
- Test: `internal/brain/maintenance/scan_test.go`
- Referenz: `src/brain/maintenance/reconcile.py:369-450, 991-1030`

**Interfaces:**
- Consumes: `identity.ReadIdentities`, `identity.ContentHash`,
  `index.FindFiles` (Task 4), `vcs.ShowBlob` (Task 6),
  `config.ArtifactLookup` (Task 2), `lock.ReplaceText` (Task 1).
- Produces:
  ```go
  type Changed struct {
      DocID       string
      Relative    string
      Revision    int
      ContentHash string
      Text        string
      Baseline    string // "" heißt: keine verlässliche Grundlinie
  }
  func Scan(area config.Area, manifest config.Manifest, lookup config.ArtifactLookup) (checked, hashed int, changed map[string]Changed, err error)
  ```

**Die vier Regeln, die hier entscheiden:**

1. **Der Stempelcache spart das Hashen.** Je Quelle werden `mtime` und Größe
   gemerkt (`_read_stats`/`_write_stats`); stimmt beides, wird nicht gehasht.
   Nur die Frischen kommen in die neue Statistikdatei — eine veränderte Quelle
   fällt heraus und wird beim nächsten Lauf wieder geprüft.
2. **Eine verschwundene Quelle ist keine Änderung.** Sie wird
   übersprungen; über sie entscheidet die Umbenennungserkennung des
   Indexlaufs.
3. **`Decode` faltet nur CRLF nach LF**, nicht ein einzelnes CR. Genau so
   faltet `identity.ContentHash`. Beide Seiten eines Diffs müssen nach
   derselben Regel gefaltet werden, sonst behauptet das Paket eine Änderung an
   einer Zeile, die niemand angefasst hat — und die Belegprüfung aus 3b ließe
   sie durch, weil sie zitierbar ist.
4. **Die Grundlinie ist nur dann eine**, wenn der committete Inhalt nach
   derselben Faltung denselben Hash hat wie das Register. Alles andere ist
   `""`. `HEAD` blind zu glauben wäre schlechter als nichts: ein
   Handcommit zwischen zwei Freigaben beschriebe eine Änderung, die nie
   stattfand.

- [ ] **Step 1: Write the failing test**

Der erste Test vollständig, als Muster für die übrigen fünf:

```go
package maintenance_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/config"
)

// Eine Quelle, deren Hash zum Register passt, ist nicht verändert: sie wird
// gezählt, aber sie steht in nichts.
func TestScanFindsNothingWhenNothingMoved(t *testing.T) {
	world := newArea(t, map[string]string{"src/a.go": "package a\n"})
	checked, _, changed, err := maintenance.Scan(world.Area, world.Manifest, world.Lookup)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if checked != 1 {
		t.Fatalf("checked = %d, want 1", checked)
	}
	if len(changed) != 0 {
		t.Fatalf("changed = %+v, want none", changed)
	}
}

// Eine Quelle, deren Inhalt sich bewegt hat, steht mit ihrem neuen Hash und
// ihrem Text in changed.
func TestScanReportsAChangedSource(t *testing.T) {
	world := newArea(t, map[string]string{"src/a.go": "package a\n"})
	path := filepath.Join(world.Area.Path, "src", "a.go")
	if err := os.WriteFile(path, []byte("package a\n\nfunc B() {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, hashed, changed, err := maintenance.Scan(world.Area, world.Manifest, world.Lookup)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if hashed != 1 {
		t.Fatalf("hashed = %d, want 1", hashed)
	}
	if len(changed) != 1 {
		t.Fatalf("changed = %d entries, want 1", len(changed))
	}
	for _, item := range changed {
		if item.Relative != "src/a.go" {
			t.Fatalf("relative = %q", item.Relative)
		}
		if item.Text != "package a\n\nfunc B() {}\n" {
			t.Fatalf("text = %q", item.Text)
		}
	}
}

// newArea baut einen Bereich mit einem Register, dessen Hashes zu den Dateien
// passen: der Ausgangszustand, von dem aus eine Änderung eine ist.
//
// Schreib diesen Helfer hier vollständig aus. Er legt an:
//   - ein Verzeichnis je Datei aus files, mit ihrem Inhalt,
//   - ein .loomux/config.toml mit [area] und [layout],
//   - unter lookup.WritePath("areas/<scope>/_identities.tsv") ein Register
//     über identity.RenderIdentities, mit identity.ContentHash je Datei,
//   - eine config.ArtifactLookup{Primary: t.TempDir()} ohne Fallback.
func newArea(t *testing.T, files map[string]string) areaWorld { t.Helper(); return areaWorld{} }

type areaWorld struct {
	Area     config.Area
	Manifest config.Manifest
	Lookup   config.ArtifactLookup
}

var _ = identity.ContentHash
```

Die übrigen vier folgen demselben Muster, mit dieser Welt und dieser
Erwartung:

| Test | Welt | Erwartung |
|---|---|---|
| `TestScanUsesTheStampCache` | Zweimal `Scan` ohne Änderung dazwischen | `hashed == 1` beim ersten, `hashed == 0` beim zweiten Lauf |
| `TestScanSkipsAVanishedSource` | Register kennt `src/a.go`, die Datei ist gelöscht | `checked == 0`, `changed` leer, kein Fehler |
| `TestDecodeFoldsOnlyCRLF` | Zwei Dateien, eine mit `a\r\nb`, eine mit `a\nb`, und eine dritte mit einem einzelnen `a\rb` | Die ersten beiden geben denselben Text, die dritte behält ihr `\r` |
| `TestBaselineIsEmptyWhenHeadDisagrees` | Repo-Helfer aus Task 6, HEAD trägt einen anderen Inhalt als das Register | `Baseline == ""` |

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/brain/maintenance/ -run 'TestScan|TestDecode|TestBaseline' -v`
Expected: FAIL

- [ ] **Step 3: Write the implementation**

Nach `reconcile.py:369-450`. Die Statistikdatei liegt unter
`lookup.WritePath(filepath.Join("maintenance", "stats-<scope>.tsv"))` — den
genauen Namen gibt `_stat_path` (`reconcile.py:991-993`); übernimm ihn
wörtlich, sonst findet ein übersetzter Fall die Datei nicht.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/brain/maintenance/ -cover`
Expected: PASS, 100 %

- [ ] **Step 5: Commit**

```bash
git add internal/brain/maintenance/scan.go internal/brain/maintenance/scan_test.go
```

```bash
git commit -m "feat(maintenance): scan an area for changed sources"
```

---

### Task 11: `reconcile.go` — Prüfzentrum und Quellfälle

**Files:**
- Create: `internal/brain/maintenance/reconcile.go`
- Test: `internal/brain/maintenance/reconcile_test.go`
- Referenz: `src/brain/maintenance/reconcile.py:151-268, 313-367, 452-530, 692-870, 941-1030`

**Interfaces:**
- Consumes: alles aus den Tasks 7–10.
- Produces:
  ```go
  type Report struct {
      Checked, Hashed int
      Cases           []Case
      LastRun         time.Time
      Unreadable      []string
  }
  var ErrNoReviewCentre = errors.New("no area declares a review centre")
  func ReviewRoot(areas []config.Area, lookup config.ArtifactLookup) (string, error)
  func Reconcile(areas []config.Area, lookup config.ArtifactLookup, now time.Time) (Report, error)
  func ReadLastRun(lookup config.ArtifactLookup) (time.Time, error)
  ```
  In diesem Task erzeugt `Reconcile` **nur Quellfälle**; die Merge-Fälle
  kommen in Task 12.

**Die fünf Entscheidungen dieses Tasks:**

1. **Das Prüfzentrum ist eins.** `_review_root` findet den Bereich, dessen
   Manifest `[layout].review` erklärt. Erklären zwei verschiedene Orte, ist
   das ein Fehler. Erklärt keiner einen, ist es `ErrNoReviewCentre` — den
   der Aufrufer unterscheiden können muss, weil `reindex` ihn anders behandelt
   als jeden anderen Fehler (Task 14).
2. **Ein fehlendes Manifest wird übersprungen**, ein unlesbares bricht ab
   (`_manifests`, `reconcile.py:253-268`).
3. **Ein stehender Fall bleibt stehen**, wenn sich seine Quellen nicht bewegt
   haben: er behält `created`, seine `id` und jeden Vorschlag, der schon
   neben ihm liegt (`_land_case`, `reconcile.py:748-752`).
4. **`local_only` folgt dem heutigen Manifest**, auch bei einem stehenden Fall
   (`_with_current_mode`): es ist der Schalter, der einen Quelldiff vom
   Cloud-Modell fernhält, und der darf nicht an einem alten Stand hängen.
5. **Kein Vorschlag in 3a.** `_proposers` baut in der Referenz einen
   Ollama-Proposer je `local_only`-Bereich; das lokale Modell ist Stufe 4.
   3a fragt niemanden: ein `local_only`-Bereich bekommt
   `Manual: true` und ein `Note`, das sagt, warum kein Vorschlag daneben
   liegt. Die Feldsemantik gibt das her (`case.py:52-58`). **Der Eintrag
   gehört in `docs/.superpowers/parity/stufe-3a.md`** — er steht dort schon,
   prüfe nur, dass er zum gebauten Verhalten passt.

- [ ] **Step 1: Write the failing tests**

Die beiden mit der meisten Aussage vollständig:

```go
// Erklärt kein Bereich ein Prüfzentrum, ist das ErrNoReviewCentre -- und der
// Aufrufer muss es mit errors.Is erkennen können, weil reindex es anders
// behandelt als jeden anderen Fehler (Task 14).
func TestReconcileWithoutAReviewCentre(t *testing.T) {
	world := newVault(t, vaultSpec{Areas: []areaSpec{{
		Scope: "project/a",
		Files: map[string]string{"docs/wiki/a.md": "# A\n"},
		// Kein Review: das ist der Zustand unter Prüfung.
	}}})
	_, err := maintenance.Reconcile(world.Areas, world.Lookup, world.Now)
	if !errors.Is(err, maintenance.ErrNoReviewCentre) {
		t.Fatalf("err = %v, want ErrNoReviewCentre", err)
	}
}

// Ein zweiter Lauf ohne Änderung lässt den Fall stehen: gleiche id, gleiches
// created, und jeder Vorschlag, der schon daneben liegt, bleibt liegen.
func TestReconcileLeavesAStandingCaseAlone(t *testing.T) {
	world := newVault(t, vaultSpec{Areas: []areaSpec{{
		Scope:  "project/a",
		Review: "95 Prüfzentrum",
		Files:  map[string]string{"src/a.go": "package a\n", "docs/wiki/a.md": "# A\n"},
	}}})
	world.Change(t, "project/a", "src/a.go", "package a\n\nfunc B() {}\n")

	first, err := maintenance.Reconcile(world.Areas, world.Lookup, world.Now)
	if err != nil {
		t.Fatalf("first Reconcile: %v", err)
	}
	if len(first.Cases) != 1 {
		t.Fatalf("first run raised %d cases, want 1", len(first.Cases))
	}

	later := world.Now.Add(24 * time.Hour)
	second, err := maintenance.Reconcile(world.Areas, world.Lookup, later)
	if err != nil {
		t.Fatalf("second Reconcile: %v", err)
	}
	if len(second.Cases) != 1 {
		t.Fatalf("second run raised %d cases, want 1", len(second.Cases))
	}
	if second.Cases[0].ID != first.Cases[0].ID {
		t.Fatalf("id moved: %q -> %q", first.Cases[0].ID, second.Cases[0].ID)
	}
	if !second.Cases[0].Created.Equal(first.Cases[0].Created) {
		t.Fatalf("created moved: %v -> %v", first.Cases[0].Created, second.Cases[0].Created)
	}
}
```

`newVault` ist der Welt-Helfer dieses Pakets: mehrere Bereiche, je mit
Dateien, optionalem `[layout].review` und optionalem `privacy_mode`, dazu
Registry, Register und `ArtifactLookup`. Schreib ihn beim ersten Test
vollständig aus; `world.Change` schreibt eine Datei neu, damit der nächste
Scan sie als verändert sieht.

Die übrigen sechs folgen demselben Muster:

| Test | Welt | Erwartung |
|---|---|---|
| `TestReconcileOnAnEmptyVault` | Keine Bereiche | Leerer Bericht, kein Fehler |
| `TestReconcileWithTwoReviewCentres` | Zwei Bereiche mit verschiedenem `[layout].review` | Fehler, der beide Orte nennt; **nicht** `ErrNoReviewCentre` |
| `TestReconcileRaisesACaseForAChangedSource` | Ein Bereich, eine geänderte Quelle | Ein Fall, und neben ihm liegen `case.toml` **und** `package.md` |
| `TestReconcileSkipsAnAreaWithoutAManifest` | Zwei Bereiche, einer ohne `.loomux/config.toml` | Kein Fehler; der andere wird normal geprüft |
| `TestReconcileFailsOnABrokenManifest` | Ein Bereich, dessen `config.toml` kein gültiges TOML ist | Fehler, der die Datei nennt |
| `TestReconcileMarksALocalOnlyCaseManual` | Ein Bereich mit `privacy_mode = "local_only"` und einer geänderten Quelle | Der Fall trägt `Manual: true` und ein `Note`; **kein** Vorschlag daneben, und **kein** Versuch, ein Modell zu erreichen |

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/brain/maintenance/ -run TestReconcile -v`
Expected: FAIL

- [ ] **Step 3: Write the implementation**

Der Ablauf, gegen `reconcile.py:194-228`: Manifeste laden → Prüfzentrum
bestimmen → je Bereich `Scan` → Quellfälle → entdoppeln → `last-run.txt`
schreiben → Bericht. Jede Fallakte über `lock.ReplaceText`, unter der Sperre
des Prüfzentrums.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/brain/maintenance/ -cover`
Expected: PASS, 100 %

- [ ] **Step 5: Commit**

```bash
git add internal/brain/maintenance/reconcile.go internal/brain/maintenance/reconcile_test.go
```

```bash
git commit -m "feat(maintenance): reconcile sources against the register"
```

---

### Task 12: `reconcile.go` — Merge-Fälle, Ablösung, Entdoppelung

**Files:**
- Modify: `internal/brain/maintenance/reconcile.go`
- Test: `internal/brain/maintenance/reconcile_merge_test.go`
- Referenz: `src/brain/maintenance/reconcile.py:488-690, 941-990`

**Interfaces:**
- Consumes: `ReadEvents`, `DropEvent` (Task 7), `vcs.ChangedPaths`,
  `vcs.CommitSubjects` (Task 6), alles aus Task 11.
- Produces: keine neue öffentliche Form — `Reconcile` tut jetzt mehr.

**Die vier Regeln, die hier entscheiden und die leicht falsch gebaut werden:**

1. **Merge-Fälle kommen nach den Quellfällen.** Ein Merge-Fall tritt hinter
   einen stehenden Quellfall derselben Seite zurück, und er kann nur einen
   sehen, der schon auf der Platte liegt (`reconcile.py:213-215`). Die
   Reihenfolge ist keine Geschmacksfrage.
2. **Ein stehender Merge-Fall wird von einem Quellfall nicht überschrieben.**
   Er trägt Belege aus einem verbrauchten Merge-Ereignis; überschreiben hieße,
   sie endgültig zu verlieren. Die Quelländerung bleibt im Register
   unvorgerückt und geht in einem späteren Lauf auf
   (`_land_case`, `reconcile.py:753-759`).
3. **`weight` ist für beide Erzeuger `change`.** Ein Merge ist ein Ereignis
   und kein Fälligwerden, und das Vokabular ist auf `change | time`
   geschlossen. Einen dritten Wert zu erfinden änderte das Format vor der
   Messung, die ihn rechtfertigen müsste.
4. **Entdoppelt wird nach `id`**, und die erste Fassung gewinnt.

- [ ] **Step 1: Write the failing tests**

Alle fünf mit `newVault` aus Task 11, erweitert um ein Ereignisprotokoll in
der Welt (`world.Events(t, ...)` schreibt die Zeilen, die der Hook geschrieben
hätte):

| Test | Welt | Erwartung |
|---|---|---|
| `TestReconcileRaisesAMergeCase` | Ein Bereich mit Repo, ein Ereignis, dessen `ChangedPaths` eine Wiki-Seite berühren | Ein Fall mit `Trigger == "merge"`, `Weight == "change"`, **keine** `Sources` |
| `TestMergeCaseDefersToAStandingSourceCase` | Für dieselbe Seite liegt schon ein Quellfall; dann kommt das Ereignis | Ein Fall, nicht zwei; der stehende Quellfall behält seine `ID` |
| `TestSourceChangeDoesNotOverwriteAStandingMergeCase` | Ein stehender Merge-Fall, danach eine Quelländerung derselben Seite | Der Merge-Fall steht unverändert; die Quelländerung ist im Register **nicht** vorgerückt und geht in einem späteren Lauf auf |
| `TestConsumedEventIsDropped` | Ein Ereignis, zwei Läufe | Der zweite Lauf eröffnet keinen zweiten Fall; das Ereignis steht in der Ablage |
| `TestReconcileDeduplicatesByID` | Zwei Wege auf dieselbe `id` (gleicher Bereich, gleiches Ziel, gleicher Tag) | Genau ein Fall im Bericht |

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/brain/maintenance/ -run 'TestMerge|TestConsumed|TestReconcileDedup|TestReconcileRaisesAMerge' -v`
Expected: FAIL

- [ ] **Step 3: Write the implementation**

Nach `_merge_cases`, `_by_repository`, `_resolve_event`, `_merge_evidence`,
`_land_merge`, `_absorbable`, `_standing_case`, `_supersede`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/brain/maintenance/ -cover`
Expected: PASS, 100 %

- [ ] **Step 5: Commit**

```bash
git add internal/brain/maintenance/
```

```bash
git commit -m "feat(maintenance): raise merge cases behind the source ones"
```

---

### Task 13: `loomux reconcile`

**Files:**
- Create: `internal/cli/maintenance.go`
- Test: `internal/cli/maintenance_test.go`
- Modify: `internal/cli/commands.go`

**Interfaces:**
- Consumes: `maintenance.Reconcile` (Tasks 11, 12), `config.ReadRegistry`.
- Produces: `loomux reconcile`, **ohne `--state-dir`**. Ruling vom
  2026-09-20: kein Befehl dieser Stufe nimmt die Flagge an, wie schon die
  fünf brain-Befehle aus 1b-1 sie nicht annehmen — ein Zustandsmodell, und
  das ist die Umgebung. Die Übersetzung der Fälle entfernt sie. Die Zeile
  steht in `parity/stufe-3a.md`.

**Die Ausgabe** folgt `cli._reconcile` (`cli.py:1055-1090`): der Bericht auf
stdout (geprüft, gehasht, Fälle mit ihrem Ort), unlesbare Quellen auf stderr.
Exit 0, auch wenn Fälle aufgegangen sind — ein eröffneter Fall ist das
Ergebnis des Befehls, kein Fehler.

- [ ] **Step 1: Write the failing test**

Über den vorhandenen CLI-Testhelfer (`grep -n "func runCLI" internal/cli/`),
mit `LOOMUX_STATE_DIR` auf eine gebaute Welt:

| Test | Welt | Erwartung |
|---|---|---|
| `TestReconcileExitsZeroWithCases` | Ein Bereich mit Prüfzentrum und einer geänderten Quelle | Exit **0**; stdout nennt Zahl und Ort der Fälle. Ein eröffneter Fall ist das Ergebnis des Befehls, kein Fehler |
| `TestReconcileWithoutAReviewCentreIsRed` | Ein Bereich ohne `[layout].review` | Exit ≠ 0; stderr nennt `[layout]` und `review` |
| `TestReconcileReportsUnreadableSources` | Eine Quelle, die nicht lesbar ist (unter Windows: eine Datei, an deren Stelle ein Verzeichnis steht) | Exit 0; stderr nennt den Pfad; der Lauf bricht **nicht** ab |

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestReconcile -v`
Expected: FAIL, unbekannter Unterbefehl

- [ ] **Step 3: Write the implementation**

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestReconcile -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/maintenance.go internal/cli/maintenance_test.go internal/cli/commands.go
```

```bash
git commit -m "feat(cli): add reconcile"
```

---

### Task 14: Der Auffangdurchgang vor `reindex`

**Files:**
- Create: `internal/brain/index/catchup.go`
- Modify: `internal/cli/index.go`
- Test: `internal/cli/index_catchup_test.go`
- Referenz: `src/brain/cli.py:964-1042`

**Interfaces:**
- Consumes: `maintenance.Reconcile`, `maintenance.ErrNoReviewCentre`,
  `index.Reindex`.
- Produces: keine neue öffentliche Form — `loomux reindex` tut jetzt mehr.

**Drei Zustände, und `reindex` bricht in zweien davon nicht ab.** Das ist der
Punkt, an dem die Referenz anders ist, als man es beim ersten Lesen erwartet:

| Zustand | Verhalten |
|---|---|
| Durchgang grün, Fälle eröffnet | Fälle auf **stderr** aufzählen (stdout trägt das Ergebnis des Befehls, und die Fälle sind eine Nebenwirkung), dann indizieren |
| `ErrNoReviewCentre` | Lange Warnung auf stderr, dann indizieren. Die Begründung steht in `cli.py:1005-1023`: ein Korpus ohne Prüfzentrum hat kein Tor, um das man laufen könnte, und das Indizieren hier zu verweigern machte den ganzen Befehl von einer Erklärung abhängig, die der Indexlauf sonst nirgends liest |
| Jeder andere Fehler des Durchgangs | **Exit 1, nicht indiziert.** Die Meldung nennt Ursache und Weg: beheben, `loomux reconcile`, dann `loomux reindex` |

**Die Warnung und die Fehlermeldung sind deutsch** — sie stehen so in
`cli.py:1019-1038` und sind Teil des Verhaltens. Übernimm sie wörtlich, mit
`brain` durch `loomux` ersetzt.

- [ ] **Step 1: Write the failing tests**

| Test | Welt | Erwartung |
|---|---|---|
| `TestReindexIndexesDespiteOpenCases` | Prüfzentrum da, eine geänderte Quelle | Exit **0**; stderr zählt die Fälle auf; die Artefakte unter `<state>/areas/` sind **neu geschrieben** (der Nachweis, dass indiziert wurde) |
| `TestReindexWarnsWithoutAReviewCentreAndStillIndexes` | Kein Bereich erklärt `review` | Exit **0**; stderr trägt die lange deutsche Warnung; die Artefakte sind neu geschrieben |
| `TestReindexRefusesWhenTheCatchUpFails` | Ein kaputtes Manifest (derselbe Zustand wie `TestReconcileFailsOnABrokenManifest`) | Exit **1**; stderr nennt Ursache und den Weg; die Artefakte sind **unverändert** |

Der dritte Test ist der wichtigste: er ist der einzige, der zeigt, dass es
überhaupt einen Zustand gibt, in dem nicht indiziert wird. Prüfe die
Unverändertheit über den Änderungszeitpunkt **und** den Inhalt einer
Artefaktdatei, nicht nur über den Exit-Code.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestReindex -v`
Expected: FAIL

- [ ] **Step 3: Write the implementation**

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestReindex -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/brain/index/catchup.go internal/cli/index.go internal/cli/index_catchup_test.go
```

```bash
git commit -F "$TEMP/loomux-commit-body.txt"
```

mit dieser Nachricht in der Datei:

```
feat(index): reconcile before every index run

The register advances a source's hash whenever the content moved, so an
index run without a preceding reconcile would carry a knowledge change past
the review gate. The catch-up is not a gate of its own: open cases are
listed on stderr and the run continues, matching the reference. Only a
failing catch-up stops the command.
```

---

### Task 15: `loomux area add`

**Files:**
- Create: `internal/cli/area.go`
- Test: `internal/cli/area_test.go`
- Modify: `internal/cli/commands.go`
- Referenz: `src/brain/init.py` (452), `src/brain/cli.py:639-657`

**Interfaces:**
- Consumes: `config.AddArea` (Task 3), `index.Reindex` (Task 4),
  `lock.ReplaceText` (Task 1).
- Produces: `loomux area add [--path P] [--scope S] [--wiki W] [--sources S]
  [--merge-branch B] [--privacy M] [--no-reindex] [-y|--yes]`.

**Was es tut**, in dieser Reihenfolge:
1. Bereich in die Registry (`config.AddArea`).
2. `[area]` und `[layout]` in die `.loomux/config.toml` des Repos.
3. Wiki-Gerüst anlegen, wenn es keins gibt.
4. Die Weichenregel in die Projektanweisung schreiben — **auf Deutsch**, weil
   sie in ein fremdes Repo kopiert und dort von Menschen und Modellen gelesen
   wird (`init.py:18-30`, der Kommentar dort begründet genau das).
5. `reindex`, außer bei `--no-reindex`.

**Nicht übernommen:** `--agents` und `--no-hook`. Die Hook- und
Agenten-Installation gehört `loomux init` in Stufe 4 (Nachtrag #5).

**Vorgabe für `--scope`** ist `project/<name des Verzeichnisses>`.

- [ ] **Step 1: Write the failing tests**

| Test | Welt | Erwartung |
|---|---|---|
| `TestAreaAddRegistersANewArea` | Ein leeres Repo unter `t.TempDir()` | Exit 0; `ReadRegistry` findet den Bereich; `.loomux/config.toml` trägt `[area]` und `[layout]`; das Wiki-Verzeichnis existiert |
| `TestAreaAddRefusesADuplicateScope` | Derselbe Scope schon registriert | Exit ≠ 0; stderr nennt den Scope; die Registry hat **einen** Eintrag |
| `TestAreaAddSkipsReindexOnRequest` | Neues Repo, `--no-reindex` | Exit 0; unter `<state>/areas/<scope>/` liegt **kein** Register |
| `TestAreaAddRefusesToPromptWithoutATerminal` | Neues Repo, **ohne** `--yes`, stdin geschlossen | Exit ≠ 0 mit einer Meldung, die `--yes` nennt. Ein Befehl, der auf eine Antwort wartet, die nie kommt, hängt einen Hook auf |
| `TestAreaAddWritesTheRoutingRuleInGerman` | Neues Repo mit `AGENTS.md` | Die Datei enthält die Überschrift `## Wohin welches Wissen gehört` und den deutschen Absatz aus `init.py:22-30` |

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestAreaAdd -v`
Expected: FAIL

- [ ] **Step 3: Write the implementation**

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestAreaAdd -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/area.go internal/cli/area_test.go internal/cli/commands.go
```

```bash
git commit -m "feat(cli): onboard an area with area add"
```

---

### Task 16: **Halt** — der Tag, dann die Fallsuite 3a

**Files:**
- Create: `testdata/cases/3a/**`, `internal/cli/cases_3a_test.go`
- Modify: `internal/dev/importcases/importcases.go` (die Übersetzungstabelle
  von 3a)

**Menschlicher Schritt zuerst.** Im Repo `ultra-brain`, vom Menschen, nicht
von einem Agenten:

```bash
git -C ~/Documents/#GIT/ultra-brain tag loomux-3-source 3cc72d2
```

Derselbe Commit wie `loomux-1a-source`; das Repo hat sich seit Stufe 1a nicht
bewegt. Der Tag steht danach in `notes.md` jedes Falls.

**Vorbedingung, die zuerst zu klären ist** (das war Task 0 der Spec-Liste):
Stellt `tools/cases.py` eine Welt mit einem `.git` unverändert her? `reconcile`,
`vcs` und das Ereignisprotokoll brauchen echte Repos. Probiere es an einem
Fall aus, bevor du dreißig aufzeichnest. Reicht es nicht, ergänze den Rekorder
— und zwar **in `ultra-brain`**, wo er lebt.

**Zweite Vorbedingung:** Ist `sdd-2c` gemergt? `internal/cases/gitworld.go`
bringt die Git-Welten auf der loomux-Seite. Prüfe mit
`git log --oneline master | grep -i gitworld` oder
`ls internal/cases/gitworld.go`. Ist es nicht da, bleiben die Git-Fälle
**geparkt** und stehen mit Grund in `parity/stufe-3a.md`. **Baue keine zweite
Git-Welt daneben.**

- [ ] **Step 1: Die Fälle aufzeichnen**

In `ultra-brain`, über `tools/cases.py`. Der Satz steht in der Spec unter
„Parität → Aufzeichnen": zehn Fälle für `reconcile`, fünf für `reindex`, drei
für `embed`, vier für `area add`.

- [ ] **Step 2: Übersetzen**

`brain reconcile` → `loomux reconcile`, `brain reindex` → `loomux reindex`,
`brain embed` → `loomux embed`, `brain init -y` → `loomux area add -y`,
`.brain.toml`/`.ultra-brain/config.toml` → `.loomux/config.toml`,
`%LOCALAPPDATA%\brain` → `…\loomux`, `BRAIN_STATE_DIR` → `LOOMUX_STATE_DIR`.
Der Originalfall bleibt als Beleg daneben.

- [ ] **Step 3: Die Normalisierung von `<state>/areas/` festlegen**

Identitätsregister und Kataloge tragen Zeitstempel. Die 1b-1-Akte hat dafür
keine Normalisierung festgelegt, weil dort nichts schrieb. Lege sie hier fest,
so eng wie möglich, und **trage sie in die Spec nach** (Abschnitt „Parität →
Vergleichsklassen").

- [ ] **Step 4: Abspielen**

Run: `go test ./internal/cli/ -run TestCases3a -v`
Expected: PASS oder ein Eintrag in der Abweichungsliste mit Begründung.

- [ ] **Step 5: Commit**

```bash
git add testdata/cases/3a/ internal/cli/cases_3a_test.go internal/dev/importcases/ docs/.superpowers/parity/stufe-3a.md
```

```bash
git commit -m "test(3a): record and replay the parity cases"
```

---

### Task 17: **Halt** — die Bereichskonfiguration des loomux-Repos

Ein Mensch trägt `[area]` und `[layout]` in `.loomux/config.toml` ein.

**Ohne `[layout].review`.** Das Prüfzentrum des Tresors ist
`95 Prüfzentrum/` in `brain-knowledge`; ein zweiter erklärter Ort wäre ein
Fehler und kein Vorrang. Die Fälle von loomux landen dort.

**Das ist zugleich der Punkt, an dem das Risiko sichtbar wird**, das die
Fusions-Spec unter „Befunde" führt: `brain-knowledge` hat keinen Remote, und
genau dieses Verzeichnis ist unversioniert. Kein Bauhindernis für 3a, aber
sag es dem Menschen an dieser Stelle noch einmal.

- [ ] **Step 1: Den Menschen fragen und warten**

---

### Task 18: Selbstnutzung

**Files:**
- Modify: `.loomux/config.toml` (vom Menschen in Task 17), ggf. `ci/`

- [ ] **Step 1: `loomux reconcile` im eigenen Repo fahren**

Run: `./bin/loomux.exe reconcile`
Expected: ein Bericht, und entweder keine Fälle oder Fälle im Prüfzentrum von
`brain-knowledge`.

- [ ] **Step 2: `loomux reindex` im eigenen Repo fahren**

Run: `./bin/loomux.exe reindex`
Expected: der Auffangdurchgang läuft, dann der Indexlauf.

- [ ] **Step 3: Den alten Aufruf ersetzen**

Wo eine Projektanweisung oder ein Hook noch `uv run brain reindex` ruft, wird
daraus `loomux reindex`. Such mit
`grep -rn "brain reindex\|brain reconcile" --include=*.md --include=*.json --include=*.toml .`

- [ ] **Step 4: Commit**

```bash
git add -A
```

```bash
git commit -m "chore: run reconcile and reindex on loomux itself"
```

---

### Task 19: Mutationsrunde und Messungen

**Files:**
- Modify: `docs/.superpowers/parity/stufe-3a.md`, `docs/de/benchmarks.md`,
  `docs/en/benchmarks.md`

- [ ] **Step 1: Die Mutationsrunde über die Entscheidungspakete**

Run: `./bin/loomux.exe dev mutants internal/brain/maintenance`
dann: `./bin/loomux.exe dev mutants internal/config`

Jeder überlebende Mutant bekommt entweder einen nachgereichten Test oder
einen Eintrag in `parity/stufe-3a.md` mit Begründung.

- [ ] **Step 2: Die drei Messungen**

| Fall | Vergleich |
|---|---|
| `reconcile` über den loomux-Bereich, warm | gegen `uv run brain reconcile` |
| `reindex` über den loomux-Bereich, kalt und warm | gegen `uv run brain reindex` |
| `area add` auf einem leeren Repo | gegen `uv run brain init -y` |

Median aus zehn warmen Läufen, wie die Messungen der früheren Stufen.

- [ ] **Step 3: Die Startzeit nachlesen**

Run: `GODEBUG=inittrace=1 ./bin/loomux.exe --version`
Expected: kein `init()` der neuen Pakete parst eingebettete Daten. Den Befund
in `docs/de/benchmarks.md` eintragen.

- [ ] **Step 4: Das Tor fahren**

Run: `./bin/loomux.exe check precommit`
Expected: grün, Coverage 100 %.

- [ ] **Step 5: Commit**

```bash
git add docs/
```

```bash
git commit -m "docs(3a): record the mutation round and the measurements"
```

---

### Task 20: `migration.md` nachziehen

**Files:**
- Modify: `docs/en/migration.md`, `docs/de/migration.md`

**Warum dieser Task existiert:** `AGENTS.md:89-96` verlangt, dass **jeder**
Pull Request, der Migrationsarbeit beginnt, abschließt, hinzufügt oder
streicht, beide Migrationspläne im selben Pull Request nachzieht. Die erste
Fassung dieses Plans hatte das übersehen (Ruling im Ledger, 2026-09-20).

- [ ] **Step 1: Stufe 3 aufteilen**

Heute steht Stufe 3 dort als eine Zeile auf `open`. Sie wird zu 3a, 3b, 3c
mit den Inhalten aus der Spec; 3a trägt den Stand, den dieser Zweig erreicht
hat.

- [ ] **Step 2: Die neuen Fähigkeiten eintragen**

`reindex`, `embed`, `reconcile`, `area add` — je mit Herkunft
(`ultra-brain/pkg/index`, `src/brain/maintenance/`, `src/brain/init.py`) und
Stand.

- [ ] **Step 3: Nachtrag #17 als Lücke eintragen**

`reindex`/`embed` waren keiner Stufe zugeordnet; die Zeile hält fest, dass
sie es seit dem 2026-09-19 sind.

- [ ] **Step 4: Die Abhängigen neu lesen**

Stufe 4 hängt an 2b, 2c und 3. Solange 3 nur zu einem Drittel fertig ist,
bleibt sie es.

- [ ] **Step 5: Commit**

```bash
git add docs/en/migration.md docs/de/migration.md
```

```bash
git commit -m "docs(migration): split stage 3 and record its first part"
```

---

## Fertig, wenn

1. alle übersetzten Fälle von 3a grün oder freigegeben in `parity/stufe-3a.md`,
2. Coverage 100 %, jeder Ausschluss begründet,
3. die Mutationsrunde über `brain/maintenance` und die Registry-Schreibseite
   gelaufen, Überlebende dokumentiert,
4. die drei Messungen in beiden Benchmark-Dateien,
5. das loomux-Repo fährt `loomux reconcile` und `loomux reindex` auf sich selbst.
