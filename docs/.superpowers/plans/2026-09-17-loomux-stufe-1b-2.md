# loomux Stufe 1b-2 — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux serve` als langlebiger MCP-Dienst über Streamable HTTP mit zwei
Kanal-Listenern, dazu die stdio-Brücke `loomux mcp`, die ein MCP-Wirt startet.

**Architecture:** `serve` hält zwei HTTP-Listener auf `127.0.0.1`, je einen für
`local` und `cloud`, jeder mit eigenem Token; der Kanal ist damit die Adresse und
kein Argument. Die fünf Werkzeuge `brain_*` rufen dieselbe Antwortfunktion wie
die CLI, die dafür aus `internal/cli` nach `internal/brain/answer` zieht. Die
Brücke ist ein reiner Umleiter: sie beantwortet `tools/list` selbst aus einer mit
`serve` geteilten Liste und reicht `tools/call` weiter.

**Tech Stack:**
- Go, `go.mod` steigt von `go 1.25.0` auf `go 1.26.0` (Task 1 begründet das).
- Neu: `github.com/modelcontextprotocol/go-sdk v1.8.0`.
- Angehoben: `golang.org/x/sys` v0.18.0 → v0.48.0, `golang.org/x/text` v0.38.0 → v0.42.0.
- Unverändert: `github.com/BurntSushi/toml v1.6.0` (gepatchte Kopie in `third_party/toml`), `gopkg.in/yaml.v3 v3.0.1`.

**Spec:** `docs/.superpowers/specs/2026-09-17-loomux-stufe-1b-2-design.md`

**Vorgänger:** `docs/.superpowers/plans/2026-09-15-loomux-stufe-1b-1.md` (abgeschlossen).
Die dort entstandenen Pakete `internal/brain/*` sind die Grundlage; dieser Plan
schreibt keines davon neu.

---

## Global Constraints

Diese Abschnitte gelten für **jede** Task. Sie werden in den Tasks nicht
wiederholt.

**Arbeitsort**
- Empfohlen ist ein eigener Worktree auf Branch `sdd-1b-2`; Task 0 legt ihn an.
  Den Pfad trägt **der Mensch** vorher in die loomux-Registry ein, sonst
  verweigert die Schreibschranke des Piloten jeden Write.
- Der Hauptcheckout auf `master` kommt nur mit ausdrücklichem Ja des Nutzers in
  Frage. **Achtung:** am 2026-09-17 arbeitete eine zweite Sitzung im
  Hauptcheckout; vor jedem Commit `git branch --show-current` und
  `git log -1 --format=%H` lesen.
- Temporäre Dateien schreibt Go-Code (`t.TempDir()`), nie die Edit/Write-Werkzeuge.

**Sprache**
- Code, Bezeichner, Kommentare, Fehlermeldungen, Commit-Nachrichten: **englisch**.
- Pläne, Specs, `docs/.superpowers/parity/`: **deutsch**.
- `docs/en/*.md` und `docs/de/*.md` sagen dasselbe.

**Commits**
- Der Mensch ist Autor und Committer. **Kein `Co-Authored-By`** auf ein Modell,
  keine Werbezeile.
- Mehrzeilige Nachrichten über eine Datei und `git commit -F <datei>`, nie über
  ein Heredoc.
- **Nie `--no-verify`.** Schlägt das Tor fehl, wird die Ursache behoben.
- Niemand außer dem Menschen pusht.

**Coverage**
- 100 % je Funktion, Tor ist `loomux dev covergate` im Pre-Commit.
- Ausnahme nur mit `//coverage:exempt <reason>` in der Zeile **direkt über**
  `func`; der Grund nennt, was ohne Eingriff ins Betriebssystem unerreichbar ist.

**Startzeit**
- Kein `init()` und keine Paketvariable parst Daten oder baut Schemata. Alles auf
  ersten Gebrauch.
- `cmd/loomux/start_test.go` lässt keine Paketinitialisierung über 500
  Allokationen zu. Gemessen mit gelinktem SDK: Höchstwert `encoding/gob` 367.

**Zwei ungeprüfte Annahmen — nicht wegtesten, sondern so behandeln**
- **Job-Object:** ob ein MCP-Wirt uns in ein Job-Object ohne `BREAKAWAY_OK`
  steckt, ist **nicht** gemessen; die Probe am 2026-09-17 lief in gar keinem
  Job. Der Entwurf fängt den Fehlschlag im Verhalten auf (Task 9), er schließt
  ihn nicht aus. Ein Test, der Breakaway als „geht immer" behauptet, ist falsch.
- **`encoding/gob` mit 367 Allokationen** ist der Wert **mit** gelinktem SDK;
  ob der SDK ihn hereinzieht, ist nicht gegengemessen. Steigt der Wert in Task 1
  über 500, ist das ein Befund für den Menschen, keine Aufgabe des Workers.

**Konfiguration**
- `.loomux/config.toml` schreibt kein Agent. Änderungsvorschläge gehen an den
  Menschen.

**Zustandsverzeichnis**
- Alle neuen Dateien (`serve.json`, `serve.lock`, `qmd.lock`, `logs/serve.log`)
  liegen unter `config.StateDir()`, **nie** unter einem fest verdrahteten Pfad.
  Nur so isoliert ein Test sein `serve` vom echten.

---

## Dateien und Zuständigkeiten

Vor den Tasks der Schnitt, damit die Grenzen feststehen:

| Datei | Zuständigkeit |
|---|---|
| `internal/lock/lock.go` | prozessübergreifende Sperre: `Acquire`, `TryAcquire`, `Release` |
| `internal/lock/lock_windows.go` / `lock_other.go` | `LockFileEx` gegen `flock` |
| `internal/brain/answer/answer.go` | `Run` und die sechs Helfer, aus `internal/cli` umgezogen |
| `internal/mcptools/tools.go` | die fünf `*mcp.Tool` in fester Reihenfolge, auf ersten Gebrauch gebaut |
| `internal/serve/state.go` | `serve.json` lesen, atomar schreiben, Bauidentität |
| `internal/serve/serve.go` | Lebenszyklus: Sperre nehmen, Listener bauen, warten, herunterfahren |
| `internal/serve/auth.go` | Token-Middleware |
| `internal/serve/spawn.go` | entkoppelter Start, Breakaway, Logdatei |
| `internal/serve/spawn_windows.go` / `spawn_other.go` | die Prozessattribute je Plattform |
| `internal/serve/control.go` | `status` und `stop`, inklusive `--force` |
| `internal/serve/brain/tools.go` | die fünf Handler, ruft `answer.Run` |
| `internal/bridge/bridge.go` | stdio-Server, `tools/list` lokal, `tools/call` weiterleiten |
| `internal/bridge/connect.go` | `serve` finden, Bauidentität vergleichen, neu starten, eine Wiederholung |
| `internal/cli/serve.go`, `internal/cli/mcp.go` | Argumente lesen, sonst nichts |
| `internal/cli/imports_test.go` | `hooks` importiert nie `serve` oder `bridge` |
| `internal/cases/` | neuer Vergleichsmodus für MCP-Fälle |
| `testdata/cases/1b-2/`, `testdata/cases/1b-2-map.toml` | der Fallkorpus |

---

## Task 0: Worktree und Ausgangsstand

**Files:**
- Keine Codeänderung.

**Interfaces:**
- Consumes: nichts.
- Produces: einen Arbeitsort, auf den alle folgenden Tasks sich beziehen.

- [ ] **Step 1: Arbeitsort klären**

Der Mensch entscheidet und trägt ggf. den Worktree in die Registry ein:

```powershell
git worktree add ../loomux-sdd-1b-2 -b sdd-1b-2
```

Ohne Registry-Eintrag verweigert die Schreibschranke jeden Write im neuen Baum.

- [ ] **Step 2: Ausgangsstand festhalten**

```powershell
git log -1 --format='%H %an %ad'
go build -o bin/loomux.exe ./cmd/loomux
sh .githooks/pre-commit
```

Erwartet: Exit 0. Ein rotes Tor **vor** der ersten Änderung ist ein Befund für
den Menschen, keine Aufgabe dieses Plans.

- [ ] **Step 3: Grundlinie der Startzeit messen**

```powershell
Measure-Command { 1..12 | ForEach-Object { ./bin/loomux.exe --version | Out-Null } }
(Get-Item bin/loomux.exe).Length
```

Zahlen notieren; Task 14 vergleicht gegen sie.

---

## Task 1: Abhängigkeiten anheben

**Files:**
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: nichts.
- Produces: `github.com/modelcontextprotocol/go-sdk v1.8.0` ist importierbar.

**Warum die `go`-Direktive steigt:** der SDK begnügt sich mit `go 1.25.0`, aber
`x/sys` v0.48.0 und `x/text` v0.42.0 verlangen `go 1.26.0` (gemessen am
2026-09-17; `x/sys` v0.47.0 wäre noch 1.25.0). Die Projektregel „immer die
neueste Fassung" gewinnt gegen das Offenhalten älterer Toolchains.

- [ ] **Step 1: Abhängigkeiten ziehen**

```powershell
go get github.com/modelcontextprotocol/go-sdk@v1.8.0
go get golang.org/x/sys@v0.48.0
go get golang.org/x/text@v0.42.0
go mod tidy
```

`go.mod` muss danach `go 1.26.0` tragen. Steht dort mehr als 1.26.0, ist etwas
anderes eingezogen — nachsehen, nicht hinnehmen.

- [ ] **Step 2: Die Version prüfen**

```powershell
go list -m -f '{{.Version}}' github.com/modelcontextprotocol/go-sdk
```

Erwartet: `v1.8.0`. Diese Prüfung gehört in den Schritt, **nicht** in einen Test:
ein Go-Test, der `go.mod` liest, prüft ein Bauartefakt und bricht, sobald das
Paket umzieht.

- [ ] **Step 3: Ein Paketgerüst, damit der SDK wirklich gelinkt wird**

`internal/mcptools/tools.go` — vorerst nur so viel, dass der Import steht:

```go
// Package mcptools holds the tool list that serve and the bridge share.
//
// One list in one place: the bridge answers tools/list on its own so that a
// cold qmd start cannot land inside a host's handshake, and a list that drifted
// from serve's would be the worst bug this layer could have.
package mcptools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// Names are the five tools in their canonical order. The order is fixed because
// the specification asks for a deterministic tools/list.
func Names() []string {
	return []string{"brain_search", "brain_catalog", "brain_read", "brain_neighbors", "brain_status"}
}

// A function reference, not a version constant: no loomux code names a
// protocol revision -- the SDK negotiates one upward with the host and speaks
// the one it negotiates downward to serve.
var _ = mcp.NewServer
```

- [ ] **Step 4: Tests und Tor**

```powershell
go test ./internal/mcptools/ -v
go test ./cmd/loomux/ -run TestStartDoesNoWorkInPackageInit -v
sh .githooks/pre-commit
```

Erwartet: alles PASS. Der Init-Test ist hier der wichtige: er beweist, dass der
SDK keine Paketinitialisierung über 500 Allokationen mitbringt.

- [ ] **Step 5: Messen und eintragen**

```powershell
(Get-Item bin/loomux.exe).Length
Measure-Command { 1..12 | ForEach-Object { ./bin/loomux.exe --version | Out-Null } }
```

Ergebnis gegen Task 0 in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`
eintragen, mit Datum, Uhrzeit, kalt und warm.

- [ ] **Step 6: Commit**

```powershell
git add go.mod go.sum internal/mcptools/ docs/en/benchmarks.md docs/de/benchmarks.md
git commit -F ../msg.txt
```

Nachricht: `Pull in the MCP SDK and raise the dependency floor`.

---

## Task 2: `internal/lock` — die prozessübergreifende Sperre

**Files:**
- Create: `internal/lock/lock.go`, `internal/lock/lock_windows.go`, `internal/lock/lock_other.go`
- Test: `internal/lock/lock_test.go`

**Interfaces:**
- Consumes: nichts.
- Produces:
  - `func Acquire(path string) (*Handle, error)` — blockiert, bis die Sperre frei ist.
  - `func TryAcquire(path string) (*Handle, bool, error)` — nimmt sie oder gibt `false` zurück.
  - `func (h *Handle) Release() error`
  - `func (h *Handle) PID() int` — die PID, die in der Datei steht; nur zur Anzeige.

**Warum PID nur zur Anzeige:** die Ein-Instanz-Garantie ist die des
Betriebssystems. Stirbt der Prozess, gibt das OS die Sperre frei. Eine
PID-Prüfung wäre ein zweiter, schwächerer Mechanismus daneben.

- [ ] **Step 1: Die fehlschlagenden Tests schreiben**

`internal/lock/lock_test.go`:

```go
package lock_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/lock"
)

func TestTryAcquireTakesAFreeLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.lock")
	h, ok, err := lock.TryAcquire(path)
	if err != nil {
		t.Fatalf("TryAcquire: %v", err)
	}
	if !ok {
		t.Fatal("expected to take a free lock")
	}
	defer h.Release()
	if h.PID() != os.Getpid() {
		t.Errorf("PID() = %d, want %d", h.PID(), os.Getpid())
	}
}

func TestTryAcquireRefusesAHeldLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.lock")
	first, ok, err := lock.TryAcquire(path)
	if err != nil || !ok {
		t.Fatalf("first TryAcquire: %v, ok=%v", err, ok)
	}
	defer first.Release()

	// Same process, second handle: the file lock is per handle, not per process,
	// so this is the same refusal a second process would see.
	second, ok, err := lock.TryAcquire(path)
	if err != nil {
		t.Fatalf("second TryAcquire: %v", err)
	}
	if ok {
		second.Release()
		t.Fatal("expected the second TryAcquire to be refused")
	}
}

func TestReleaseLetsTheNextTakerIn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.lock")
	first, _, err := lock.TryAcquire(path)
	if err != nil {
		t.Fatalf("TryAcquire: %v", err)
	}
	if err := first.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	second, ok, err := lock.TryAcquire(path)
	if err != nil || !ok {
		t.Fatalf("after Release: %v, ok=%v", err, ok)
	}
	second.Release()
}

func TestAcquireRefusesAnUnwritablePath(t *testing.T) {
	// A directory where a file is expected: the open fails, and the failure must
	// come back as an error rather than as a taken lock.
	dir := t.TempDir()
	if _, _, err := lock.TryAcquire(dir); err == nil {
		t.Fatal("expected an error for a directory path")
	}
}
```

- [ ] **Step 2: Tests laufen lassen**

```powershell
go test ./internal/lock/ -v
```

Erwartet: FAIL, das Paket gibt es nicht.

- [ ] **Step 3: Die plattformunabhängige Hälfte schreiben**

`internal/lock/lock.go`:

```go
// Package lock is the cross-process lock, ported from the reference's
// locking.py.
//
// The guarantee is the operating system's: a handle holds the lock for as long
// as it is open, and a process that dies releases it without anyone cleaning
// up. The PID in the file is for `serve status` to print, never for deciding
// who owns the lock.
package lock

import (
	"fmt"
	"os"
	"strconv"
)

// Handle is a held lock. Release it exactly once.
type Handle struct {
	file *os.File
	pid  int
}

// TryAcquire takes the lock at path, or reports false when someone else holds
// it.
func TryAcquire(path string) (*Handle, bool, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false, fmt.Errorf("open lock %s: %w", path, err)
	}
	held, err := tryLock(file)
	if err != nil {
		file.Close()
		return nil, false, fmt.Errorf("lock %s: %w", path, err)
	}
	if !held {
		file.Close()
		return nil, false, nil
	}
	pid := os.Getpid()
	if err := writePID(file, pid); err != nil {
		unlock(file)
		file.Close()
		return nil, false, err
	}
	return &Handle{file: file, pid: pid}, true, nil
}

// Acquire blocks until the lock at path is free.
func Acquire(path string) (*Handle, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock %s: %w", path, err)
	}
	if err := waitLock(file); err != nil {
		file.Close()
		return nil, fmt.Errorf("lock %s: %w", path, err)
	}
	pid := os.Getpid()
	if err := writePID(file, pid); err != nil {
		unlock(file)
		file.Close()
		return nil, err
	}
	return &Handle{file: file, pid: pid}, nil
}

// PID is the process that holds this handle.
func (h *Handle) PID() int { return h.pid }

// Release gives the lock up.
func (h *Handle) Release() error {
	if err := unlock(h.file); err != nil {
		h.file.Close()
		return err
	}
	return h.file.Close()
}

func writePID(file *os.File, pid int) error {
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("truncate lock: %w", err)
	}
	if _, err := file.WriteAt([]byte(strconv.Itoa(pid)), 0); err != nil {
		return fmt.Errorf("write pid: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Die beiden Plattformhälften schreiben**

`internal/lock/lock_windows.go`:

```go
//go:build windows

package lock

import (
	"os"

	"golang.org/x/sys/windows"
)

func tryLock(file *os.File) (bool, error) {
	err := lockFile(file, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY)
	if err == windows.ERROR_LOCK_VIOLATION {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func waitLock(file *os.File) error {
	return lockFile(file, windows.LOCKFILE_EXCLUSIVE_LOCK)
}

func lockFile(file *os.File, flags uint32) error {
	overlapped := new(windows.Overlapped)
	return windows.LockFileEx(windows.Handle(file.Fd()), flags, 0, 1, 0, overlapped)
}

func unlock(file *os.File) error {
	overlapped := new(windows.Overlapped)
	return windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, overlapped)
}
```

`internal/lock/lock_other.go`:

```go
//go:build !windows

package lock

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func tryLock(file *os.File) (bool, error) {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func waitLock(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_EX)
}

func unlock(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
```

- [ ] **Step 5: Tests laufen lassen**

```powershell
go test ./internal/lock/ -v -cover
```

Erwartet: PASS. Die POSIX-Datei ist auf dieser Maschine unerreichbar; sie bekommt
**keinen** `coverage:exempt`-Kommentar, sondern das Tor sieht sie wegen der
Build-Tags gar nicht. Wenn `covergate` doch klagt, ist das ein Befund für den
Menschen.

- [ ] **Step 6: Commit**

Nachricht: `Add the cross-process lock ported from locking.py`.

---

## Task 3: `brainRun` zieht nach `internal/brain/answer`

**Files:**
- Create: `internal/brain/answer/answer.go`
- Modify: `internal/cli/brain.go` (heute Zeilen 79–201)
- Test: `internal/brain/answer/answer_test.go`; die bestehenden `internal/cli/brain_test.go` und `internal/cli/cases_1b1_test.go` müssen grün bleiben.

**Interfaces:**
- Consumes: `internal/brain/*` aus 1b-1.
- Produces:

```go
// In internal/brain/answer:
// Request is one command and its arguments. Query carries the single
// positional: the search query for search, the relative path for read and
// neighbors. They never occur together, so there is no second field for it --
// do not add a Relative.
type Request struct {
	Command   string // "search", "catalog", "read", "neighbors", "status"
	Query     string // search: the query; read/neighbors: the relative path
	Scope     string
	Profile   string
	Count     int
	Section   string
	Channel   privacy.Channel
}

func Run(req Request, registryDir, legacyDir string, notice func(string)) (text string, notes []string, err error)
```

**Warum der Umzug:** `internal/cli` wird `internal/serve` für den Einstiegspunkt
importieren; importierte `internal/serve` seinerseits `internal/cli` für die
Antwort, gäbe es einen Importzyklus.

**Warum die Signatur sich ändert — zwei Gründe, beide im Code nachgerechnet:**
1. `brainArgs` ist der Ergebnistyp des CLI-Parsers und bleibt in `internal/cli`.
   `Run` nimmt die Werte einzeln.
2. `brainSearch` nimmt heute ein `stderr io.Writer` (`internal/cli/brain.go:103`)
   und benutzt es **nur**, um den Warm-Hinweis hineinzuschreiben
   (`internal/cli/brain.go:104-106`). Daraus wird `notice func(string)`. Die CLI
   reicht eine Funktion herein, die `note: %s\n` auf stderr schreibt; `serve`
   reicht eine herein, die eine Fortschrittsmeldung schickt.

**Wichtig:** `notes` und der Warm-Hinweis sind **zwei verschiedene Dinge**.
`notes` sind `answer.Findings` — die Befunde der Suche, die es auch ohne
Kaltstart gibt. Der Warm-Hinweis kommt aus dem Rückruf, den
`search.DefaultConnectWith` genau dann auslöst, wenn er den Daemon selbst
starten musste. Wer beides zusammenlegt, verliert einen von beiden.

- [ ] **Step 1: Den fehlschlagenden Test schreiben**

`internal/brain/answer/answer_test.go`:

```go
package answer_test

import (
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/privacy"
)

func TestRunStatusAnswersWithoutANotice(t *testing.T) {
	dir := t.TempDir()
	var heard []string
	text, notes, err := answer.Run(answer.Request{
		Command: "status",
		Channel: privacy.ChannelLocal,
	}, dir, dir, func(m string) { heard = append(heard, m) })
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if text == "" {
		t.Error("status answered with empty text")
	}
	if len(notes) != 0 {
		t.Errorf("status produced notes: %v", notes)
	}
	if len(heard) != 0 {
		t.Errorf("status produced a notice: %v", heard)
	}
}

func TestRunRefusesAnUnknownCommand(t *testing.T) {
	dir := t.TempDir()
	_, _, err := answer.Run(answer.Request{Command: "nonesuch"}, dir, dir, func(string) {})
	if err == nil {
		t.Fatal("expected an error for an unknown command")
	}
	if !strings.Contains(err.Error(), "nonesuch") {
		t.Errorf("error does not name the command: %v", err)
	}
}

func TestRunToleratesANilNotice(t *testing.T) {
	// serve and the corpus both pass a notice; a caller that does not must not
	// crash the answer.
	dir := t.TempDir()
	if _, _, err := answer.Run(answer.Request{
		Command: "status",
		Channel: privacy.ChannelLocal,
	}, dir, dir, nil); err != nil {
		t.Fatalf("Run with nil notice: %v", err)
	}
}
```

- [ ] **Step 2: Test laufen lassen**

```powershell
go test ./internal/brain/answer/ -v
```

Erwartet: FAIL, das Paket gibt es nicht.

- [ ] **Step 3: Den Umzug ausführen**

`internal/cli/brain.go:79-201` — `brainRun`, `brainSearch`, `brainCatalog`,
`brainArea`, `brainRead`, `brainNeighbors`, `brainStatus` und `brainSearchPort`
wandern nach `internal/brain/answer/answer.go`. Dort:

```go
// Run answers one command: the text for the reader, the search findings as
// notes, or the error.
//
// notice hears the warming hint when the search had to start the engine. It is
// not a note: notes come back with the answer, the hint happens while
// connecting, and a caller that has no stderr -- serve -- turns it into a
// progress notification instead.
func Run(req Request, registryDir, legacyDir string, notice func(string)) (string, []string, error) {
	if notice == nil {
		notice = func(string) {}
	}
	switch req.Command {
	case "search":
		return search(req, registryDir, legacyDir, notice)
	case "catalog":
		text, err := catalog(req.Scope, req.Channel, registryDir, legacyDir)
		return text, nil, err
	case "read":
		text, err := read(req.Query, req.Scope, req.Section, req.Channel, registryDir, legacyDir)
		return text, nil, err
	case "neighbors":
		text, err := neighbors(req.Query, req.Scope, req.Channel, registryDir, legacyDir)
		return text, nil, err
	case "status":
		text, err := status(req.Channel, registryDir, legacyDir)
		return text, nil, err
	}
	return "", nil, fmt.Errorf("unknown command: %s", req.Command)
}
```

`search` ist das umgezogene `brainSearch`, mit `notice` statt `stderr`:

```go
func search(req Request, registryDir, legacyDir string, notice func(string)) (string, []string, error) {
	port := searchPort(notice)
	if closer, ok := port.(io.Closer); ok {
		// Let go of the session whatever the answer was; the answer does not
		// depend on how letting go went.
		defer closer.Close()
	}
	found, err := brainsearch.ExecuteSearch(req.Query, req.Scope, brainsearch.Profile(req.Profile),
		req.Count, req.Channel, port, registryDir, legacyDir, now())
	if err != nil {
		return "", nil, err
	}
	return brainsearch.FormatSearch(found), found.Findings, nil
}
```

- [ ] **Step 4: `internal/cli/brain.go` auf den Rest eindampfen**

Dort bleiben `brainCommand`, `brainRefuse` und die Übersetzung von `brainArgs`
nach `answer.Request`:

```go
func brainCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	// ... parser as before, unchanged ...
	req := answer.Request{
		Command: parser.name,
		Query:   parsed.positional,
		Scope:   parsed.values["--scope"],
		Profile: parsed.values["--profile"],
		Count:   parsed.n,
		Section: parsed.values["--section"],
		Channel: privacy.Channel(parsed.values["--channel"]),
	}
	registryDir, legacyDir := config.StateDir(), config.LegacyBrainDirUntilStage3()
	out, notes, err := answer.Run(req, registryDir, legacyDir, func(message string) {
		fmt.Fprintf(stderr, "note: %s\n", message)
	})
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	io.WriteString(stdout, out)
	for _, note := range notes {
		fmt.Fprintf(stderr, "note: %s\n", note)
	}
	return 0
}
```

- [ ] **Step 5: Die 1b-1-Fälle als Nachweis laufen lassen**

```powershell
go test ./internal/brain/answer/ ./internal/cli/ -v
```

Erwartet: PASS, **einschließlich** `cases_1b1_test.go`. Die aufgezeichneten Fälle
sind der Beweis, dass der Umzug am Verhalten nichts geändert hat. Ändert sich
auch nur ein Byte im erwarteten stdout, ist der Umzug falsch — nicht der Fall.

- [ ] **Step 6: Tor und Commit**

```powershell
sh .githooks/pre-commit
```

Nachricht: `Move the brain answer out of the CLI so serve can call it`.

---

## Task 4: Die geteilte Werkzeugliste

**Files:**
- Modify: `internal/mcptools/tools.go` (aus Task 1)
- Test: `internal/mcptools/tools_test.go`

**Interfaces:**
- Consumes: `github.com/modelcontextprotocol/go-sdk/mcp`.
- Produces:
  - `func Tools() []*mcp.Tool` — die fünf Werkzeuge in fester Reihenfolge, auf ersten Gebrauch gebaut.
  - `const CacheTTL = 5 * time.Minute`, `const CacheScope = "public"`.

**Die Argumente, gegen die Referenz nachgerechnet** — hier sind zwei Fallen:

- Die Profile heißen `fast`, `full`, `keyword` (`ultra-brain/src/brain/search/port.py:19-41`
  und `internal/cli/brainargs.go:111`). Das `fast|balanced|deep` aus dem
  Go-Prototyp `ultra-brain/pkg/mcp/tools.go` ist **veraltet** und wird nicht
  übernommen.
- `n` ist über MCP **10** (`ultra-brain/src/brain/daemon/tools.py`), auf der
  Kommandozeile **5** (`cli.py:483`, in 1b-1 als `brainResultCount`). Die
  MCP-Front behält die 10. Das ist eine bewusste Abweichung und steht so in der
  Paritätsliste.

- [ ] **Step 1: Die fehlschlagenden Tests schreiben**

`internal/mcptools/tools_test.go`:

```go
package mcptools_test

import (
	"encoding/json"
	"testing"

	"github.com/xidus90/loomux/internal/mcptools"
)

func TestToolsAreTheFiveInCanonicalOrder(t *testing.T) {
	got := mcptools.Tools()
	want := []string{"brain_search", "brain_catalog", "brain_read", "brain_neighbors", "brain_status"}
	if len(got) != len(want) {
		t.Fatalf("got %d tools, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Name != want[i] {
			t.Errorf("tool %d is %q, want %q", i, got[i].Name, want[i])
		}
	}
}

func TestToolsAreByteIdenticalAcrossCalls(t *testing.T) {
	// The bridge answers tools/list from this list and serve serves the same
	// one. The specification asks for a deterministic order; a list that
	// differed between the two would be this layer's worst bug.
	first, err := json.Marshal(mcptools.Tools())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	second, err := json.Marshal(mcptools.Tools())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(first) != string(second) {
		t.Error("two calls to Tools() marshal differently")
	}
}

func TestSearchProfilesAreTheReferenceNames(t *testing.T) {
	schema := schemaOf(t, "brain_search")
	props := schema["properties"].(map[string]any)
	profile := props["profile"].(map[string]any)
	got := profile["enum"].([]any)
	want := []string{"fast", "full", "keyword"}
	if len(got) != len(want) {
		t.Fatalf("profile enum has %d values, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].(string) != want[i] {
			t.Errorf("profile %d is %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSearchCountDefaultsToTenOverMCP(t *testing.T) {
	// The command line defaults to 5 (cli.py:483), the MCP front to 10
	// (daemon/tools.py). Parity here is with the MCP front.
	schema := schemaOf(t, "brain_search")
	props := schema["properties"].(map[string]any)
	n := props["n"].(map[string]any)
	if n["default"].(float64) != 10 {
		t.Errorf("n default is %v, want 10", n["default"])
	}
}

func TestReadRequiresScopeAndRelative(t *testing.T) {
	schema := schemaOf(t, "brain_read")
	required := schema["required"].([]any)
	if len(required) != 2 {
		t.Fatalf("read requires %d fields, want 2", len(required))
	}
	if required[0].(string) != "scope" || required[1].(string) != "relative" {
		t.Errorf("read requires %v, want [scope relative]", required)
	}
}

func TestStatusTakesNoArguments(t *testing.T) {
	schema := schemaOf(t, "brain_status")
	props, ok := schema["properties"].(map[string]any)
	if ok && len(props) != 0 {
		t.Errorf("status takes arguments: %v", props)
	}
}

func schemaOf(t *testing.T, name string) map[string]any {
	t.Helper()
	for _, tool := range mcptools.Tools() {
		if tool.Name != name {
			continue
		}
		raw, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatalf("marshal schema of %s: %v", name, err)
		}
		var schema map[string]any
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatalf("unmarshal schema of %s: %v", name, err)
		}
		return schema
	}
	t.Fatalf("no tool named %s", name)
	return nil
}
```

- [ ] **Step 2: Tests laufen lassen**

```powershell
go test ./internal/mcptools/ -v
```

Erwartet: FAIL, `Tools` gibt es nicht.

- [ ] **Step 3: Die Liste schreiben**

`internal/mcptools/tools.go`:

```go
package mcptools

import (
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CacheTTL and CacheScope let a host cache tools/list. Five static tools make
// that free.
const (
	CacheTTL   = 5 * time.Minute
	CacheScope = "public"
)

var (
	once  sync.Once
	tools []*mcp.Tool
)

// Tools are the five tools in their canonical order.
//
// Built on first use rather than in a package variable: the start floor of
// every loomux invocation, the per-edit hook included, is measured, and a
// schema built at init would be paid by callers that never speak MCP.
func Tools() []*mcp.Tool {
	once.Do(build)
	return tools
}

func build() {
	scope := map[string]any{
		"type":        "string",
		"description": "area to ask, or 'all'",
	}
	relative := map[string]any{"type": "string"}
	tools = []*mcp.Tool{
		{
			Name:        "brain_search",
			Description: "Search the visible areas.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
					"scope": scope,
					// fast is purely vectorial, full the hybrid chain, keyword
					// BM25 (search/port.py). The prototype's fast/balanced/deep
					// never existed in the reference.
					"profile": map[string]any{"type": "string", "enum": []string{"fast", "full", "keyword"}},
					// Ten, not the command line's five: parity here is with the
					// MCP front (daemon/tools.py), not with cli.py.
					"n": map[string]any{"type": "integer", "default": 10},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "brain_catalog",
			Description: "The root catalog, or one area's.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"scope": scope},
			},
		},
		{
			Name:        "brain_read",
			Description: "Exactly one file, optionally one section of it.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope":    scope,
					"relative": relative,
					"section":  map[string]any{"type": "string"},
				},
				"required": []string{"scope", "relative"},
			},
		},
		{
			Name:        "brain_neighbors",
			Description: "Incoming and outgoing links of one page.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope":    scope,
					"relative": relative,
				},
				"required": []string{"scope", "relative"},
			},
		},
		{
			Name:        "brain_status",
			Description: "What to know before trusting an answer.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}
```

- [ ] **Step 4: Tests und Init-Tor**

```powershell
go test ./internal/mcptools/ -v -cover
go test ./cmd/loomux/ -run TestStartDoesNoWorkInPackageInit -v
```

Erwartet: PASS. Der Init-Test beweist, dass die Schemata nicht beim Start
entstehen.

- [ ] **Step 5: Commit**

Nachricht: `Add the five brain tools as one shared list`.

---

## Task 5: `serve.json` — Zustand, atomar geschrieben

**Files:**
- Create: `internal/serve/state.go`
- Test: `internal/serve/state_test.go`

**Interfaces:**
- Consumes: `internal/config` (`StateDir`).
- Produces:

```go
type Endpoint struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

type State struct {
	Local      Endpoint `json:"local"`
	Cloud      Endpoint `json:"cloud"`
	PID        int      `json:"pid"`
	Executable string   `json:"executable"`
	Size       int64    `json:"size"`
	ModTime    time.Time `json:"mod_time"`
	BrokeAway  bool     `json:"broke_away"`
}

func StatePath(stateDir string) string
func LockPath(stateDir string) string
func QmdLockPath(stateDir string) string
func LogPath(stateDir string) string
func ReadState(stateDir string) (*State, error)
func WriteState(stateDir string, s *State) error
func BuildIdentity() (path string, size int64, modTime time.Time, err error)
func (s *State) OlderThan(modTime time.Time) bool
```

**`OlderThan` ist die Regel „neuer gewinnt".** `serve` ist maschinenweit,
`bin/loomux.exe` liegt pro Checkout, und es gibt mehrere Klone nebeneinander.
Eine Brücke startet `serve` nur neu, wenn ihr **eigenes** Programm neuer ist;
eine ältere tötet nie ein neueres.

- [ ] **Step 1: Die fehlschlagenden Tests schreiben**

`internal/serve/state_test.go`:

```go
package serve_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/serve"
)

func newState() *serve.State {
	return &serve.State{
		Local:      serve.Endpoint{URL: "http://127.0.0.1:1/mcp", Token: "l"},
		Cloud:      serve.Endpoint{URL: "http://127.0.0.1:2/mcp", Token: "c"},
		PID:        4711,
		Executable: `C:\loomux\bin\loomux.exe`,
		Size:       17,
		ModTime:    time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC),
	}
}

func TestWriteThenReadRoundTrips(t *testing.T) {
	dir := t.TempDir()
	want := newState()
	if err := serve.WriteState(dir, want); err != nil {
		t.Fatalf("WriteState: %v", err)
	}
	got, err := serve.ReadState(dir)
	if err != nil {
		t.Fatalf("ReadState: %v", err)
	}
	if got.Local != want.Local || got.Cloud != want.Cloud || got.PID != want.PID {
		t.Errorf("round trip lost data: %+v", got)
	}
	if !got.ModTime.Equal(want.ModTime) {
		t.Errorf("ModTime = %v, want %v", got.ModTime, want.ModTime)
	}
}

func TestWriteLeavesNoTemporaryFileBehind(t *testing.T) {
	dir := t.TempDir()
	if err := serve.WriteState(dir, newState()); err != nil {
		t.Fatalf("WriteState: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(serve.StatePath(dir)) {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("state directory holds %v, want only the state file", names)
	}
}

func TestReadMissingStateSaysSo(t *testing.T) {
	if _, err := serve.ReadState(t.TempDir()); err == nil {
		t.Fatal("expected an error for a missing state file")
	}
}

func TestReadBrokenStateSaysSo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(serve.StatePath(dir), []byte("{not json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := serve.ReadState(dir); err == nil {
		t.Fatal("expected an error for a broken state file")
	}
}

func TestOlderThanComparesTheBuild(t *testing.T) {
	s := newState()
	newer := s.ModTime.Add(time.Minute)
	older := s.ModTime.Add(-time.Minute)

	if !s.OlderThan(newer) {
		t.Error("a newer build should win")
	}
	if s.OlderThan(older) {
		t.Error("an older build must never win")
	}
	if s.OlderThan(s.ModTime) {
		t.Error("the same build is not newer")
	}
}

func TestPathsAllSitUnderTheStateDir(t *testing.T) {
	dir := t.TempDir()
	for name, got := range map[string]string{
		"state":    serve.StatePath(dir),
		"lock":     serve.LockPath(dir),
		"qmd lock": serve.QmdLockPath(dir),
		"log":      serve.LogPath(dir),
	} {
		if rel, err := filepath.Rel(dir, got); err != nil || strings.HasPrefix(rel, "..") {
			t.Errorf("%s path %q is not under the state dir", name, got)
		}
	}
}
```

- [ ] **Step 2: Tests laufen lassen**

```powershell
go test ./internal/serve/ -v
```

Erwartet: FAIL, das Paket gibt es nicht.

- [ ] **Step 3: `state.go` schreiben**

```go
// Package serve is the long-lived MCP service: two listeners, one per channel,
// and the lifecycle around them.
package serve

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Endpoint is one channel's address and its token.
type Endpoint struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// State is what serve.json holds. It is a hint, never the truth: the truth is
// whether the listener answers and whether serve.lock is held.
type State struct {
	Local      Endpoint  `json:"local"`
	Cloud      Endpoint  `json:"cloud"`
	PID        int       `json:"pid"`
	Executable string    `json:"executable"`
	Size       int64     `json:"size"`
	ModTime    time.Time `json:"mod_time"`
	BrokeAway  bool      `json:"broke_away"`
}

// StatePath is serve.json under the state directory.
func StatePath(stateDir string) string { return filepath.Join(stateDir, "serve.json") }

// LockPath is the lock serve holds for its whole life.
func LockPath(stateDir string) string { return filepath.Join(stateDir, "serve.lock") }

// QmdLockPath guards probing and starting qmd. brain search takes the same one.
func QmdLockPath(stateDir string) string { return filepath.Join(stateDir, "qmd.lock") }

// LogPath is where a detached serve writes, since nobody reads its stderr.
func LogPath(stateDir string) string { return filepath.Join(stateDir, "logs", "serve.log") }

// ReadState reads serve.json.
func ReadState(stateDir string) (*State, error) {
	data, err := os.ReadFile(StatePath(stateDir))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", StatePath(stateDir), err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", StatePath(stateDir), err)
	}
	return &s, nil
}

// WriteState writes serve.json atomically: a temporary file next to it, then a
// rename. A reader never sees half a file.
func WriteState(stateDir string, s *State) error {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", stateDir, err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	temp, err := os.CreateTemp(stateDir, "serve-*.json")
	if err != nil {
		return fmt.Errorf("create temporary state file: %w", err)
	}
	name := temp.Name()
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		os.Remove(name)
		return fmt.Errorf("write temporary state file: %w", err)
	}
	if err := temp.Close(); err != nil {
		os.Remove(name)
		return fmt.Errorf("close temporary state file: %w", err)
	}
	if err := os.Chmod(name, 0o600); err != nil {
		os.Remove(name)
		return fmt.Errorf("chmod temporary state file: %w", err)
	}
	if err := os.Rename(name, StatePath(stateDir)); err != nil {
		os.Remove(name)
		return fmt.Errorf("rename state file: %w", err)
	}
	return nil
}

// BuildIdentity describes the running program, taken from os.Executable rather
// than from a fixed bin/ path: serve is machine-wide while bin/loomux.exe sits
// in one checkout of several.
func BuildIdentity() (string, int64, time.Time, error) {
	path, err := os.Executable()
	if err != nil {
		return "", 0, time.Time{}, fmt.Errorf("find the running program: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", 0, time.Time{}, fmt.Errorf("stat %s: %w", path, err)
	}
	return path, info.Size(), info.ModTime(), nil
}

// OlderThan reports whether the recorded build is older than the one described.
// Newer wins, and only newer: an older bridge never restarts a newer serve, or
// two hosts out of two checkouts would kill each other on every call.
func (s *State) OlderThan(modTime time.Time) bool {
	return s.ModTime.Before(modTime)
}
```

Nur die Zeit entscheidet. `Size` steht trotzdem in `State`, weil `serve status`
den laufenden Bau anzeigen soll; für „neuer gewinnt" wäre eine Größenprüfung
eine zweite Regel neben der ersten, die bei gleichem Zeitstempel widersprechen
könnte.

- [ ] **Step 4: Tests laufen lassen**

```powershell
go test ./internal/serve/ -v -cover
```

Erwartet: PASS, 100 % je Funktion.

- [ ] **Step 5: Commit**

Nachricht: `Record the service endpoints and the build that serves them`.

---

## Task 6: Die Token-Middleware und der Cross-Origin-Schutz

**Files:**
- Create: `internal/serve/auth.go`
- Test: `internal/serve/auth_test.go`

**Interfaces:**
- Consumes: nichts aus diesem Plan.
- Produces:
  - `func NewToken() (string, error)` — 32 Bytes aus `crypto/rand`, base64url.
  - `func RequireToken(token string, next http.Handler) http.Handler`

**Warum vor dem SDK-Handler:** ein falscher Token endet mit 401, bevor irgendetwas
MCP-Ähnliches passiert. Und warum zusätzlich `http.CrossOriginProtection`: ohne
sie könnte eine beliebige Webseite im Browser des Nutzers gegen `127.0.0.1`
schießen; der Token allein hilft nicht, sobald er je in eine URL gerät.

- [ ] **Step 1: Die fehlschlagenden Tests schreiben**

`internal/serve/auth_test.go`:

```go
package serve_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xidus90/loomux/internal/serve"
)

func TestNewTokenIsLongAndFresh(t *testing.T) {
	first, err := serve.NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if len(first) < 40 {
		t.Errorf("token is %d characters, want at least 40", len(first))
	}
	second, err := serve.NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if first == second {
		t.Error("two tokens are the same")
	}
}

func TestRequireTokenLetsTheRightTokenThrough(t *testing.T) {
	handler := serve.RequireToken("secret", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTeapot {
		t.Errorf("status %d, want %d", rec.Code, http.StatusTeapot)
	}
}

func TestRequireTokenRefusesAWrongToken(t *testing.T) {
	var reached bool
	handler := serve.RequireToken("secret", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached = true
	}))
	for _, header := range []string{"", "Bearer", "Bearer wrong", "secret", "Basic secret"} {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("header %q gave status %d, want 401", header, rec.Code)
		}
	}
	if reached {
		t.Error("a refused request reached the handler")
	}
}
```

- [ ] **Step 2: Tests laufen lassen**

Erwartet: FAIL.

- [ ] **Step 3: `auth.go` schreiben**

```go
package serve

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// NewToken is one listener's secret: 32 bytes of randomness, base64url.
func NewToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("draw a token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

// RequireToken refuses every request that does not carry the listener's token,
// before anything MCP-shaped happens.
//
// The comparison is constant-time. The secret never leaves this machine, but a
// timing side channel is cheap to avoid and expensive to argue about.
func RequireToken(token string, next http.Handler) http.Handler {
	want := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		got := []byte(strings.TrimSpace(req.Header.Get("Authorization")))
		if subtle.ConstantTimeCompare(got, want) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, req)
	})
}
```

- [ ] **Step 4: Tests laufen lassen**

```powershell
go test ./internal/serve/ -v -cover
```

Erwartet: PASS.

- [ ] **Step 5: Commit**

Nachricht: `Guard each listener with its own bearer token`.

---

## Task 7: Die fünf Werkzeuge in `serve`

**Files:**
- Create: `internal/serve/brain/tools.go`
- Test: `internal/serve/brain/tools_test.go`

**Interfaces:**
- Consumes: `answer.Run` (Task 3), `mcptools.Tools` (Task 4).
- Produces:

```go
// Deps lets a test replace the answer without a registry on disk.
type Deps struct {
	Answer      func(answer.Request, string, string, func(string)) (string, []string, error)
	RegistryDir string
	LegacyDir   string
}

func Register(server *mcp.Server, channel privacy.Channel, deps Deps)
```

**Die Fehlertrennung ist der Kern dieser Task** (Referenz, Spec 4.5): was der
Kern verweigert, ist Inhalt fürs Modell — `CallToolResult` mit `IsError: true`
und der Begründung. Was der Transport verliert, ist eine Störung — ein Fehler
aus dem Handler, den der SDK als MCP-Fehler nach oben gibt. **Nie eine leere
Trefferliste, wo eine Leitung fehlt**, sonst lernt das Modell, einen toten
Dienst als „nichts gefunden" zu lesen.

`answer.Run` gibt heute für beides ein `error` zurück. Die Trennung fällt nach
der Herkunft: alles, was `answer.Run` zurückgibt, ist eine Aussage des Kerns
über die Frage — unbekannter Bereich, fehlende Datei, kaputtes Manifest — und
damit `IsError: true`. Eine Störung entsteht erst eine Ebene tiefer, im
qmd-Sprung aus Task 10, und kommt von dort als `error` heraus.

- [ ] **Step 1: Die fehlschlagenden Tests schreiben**

`internal/serve/brain/tools_test.go`:

```go
package brain_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/privacy"
	servebrain "github.com/xidus90/loomux/internal/serve/brain"
)

// connect wires a server with the five tools to an in-memory client. No socket,
// no port, no waiting.
func connect(t *testing.T, channel privacy.Channel, deps servebrain.Deps) *mcp.ClientSession {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "loomux", Version: "test"}, nil)
	servebrain.Register(server, channel, deps)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func TestTheFiveToolsAreListed(t *testing.T) {
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(answer.Request, string, string, func(string)) (string, []string, error) {
			return "", nil, nil
		},
	})
	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	want := []string{"brain_search", "brain_catalog", "brain_read", "brain_neighbors", "brain_status"}
	if len(res.Tools) != len(want) {
		t.Fatalf("got %d tools, want %d", len(res.Tools), len(want))
	}
	for i := range want {
		if res.Tools[i].Name != want[i] {
			t.Errorf("tool %d is %q, want %q", i, res.Tools[i].Name, want[i])
		}
	}
}

func TestTheListenersChannelReachesTheAnswer(t *testing.T) {
	var seen privacy.Channel
	session := connect(t, privacy.ChannelCloud, servebrain.Deps{
		Answer: func(req answer.Request, _, _ string, _ func(string)) (string, []string, error) {
			seen = req.Channel
			return "ok", nil, nil
		},
	})
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if seen != privacy.ChannelCloud {
		t.Errorf("the answer saw channel %q, want cloud", seen)
	}
}

func TestArgumentsReachTheAnswer(t *testing.T) {
	var seen answer.Request
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(req answer.Request, _, _ string, _ func(string)) (string, []string, error) {
			seen = req
			return "ok", nil, nil
		},
	})
	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "brain_search",
		Arguments: map[string]any{
			"query":   "zettel",
			"scope":   "wiki",
			"profile": "full",
			"n":       3,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if seen.Command != "search" || seen.Query != "zettel" || seen.Scope != "wiki" ||
		seen.Profile != "full" || seen.Count != 3 {
		t.Errorf("the answer saw %+v", seen)
	}
}

func TestACoreRefusalIsContentForTheModel(t *testing.T) {
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(answer.Request, string, string, func(string)) (string, []string, error) {
			return "", nil, errors.New("unknown scope: nonesuch")
		},
	})
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_catalog"})
	if err != nil {
		t.Fatalf("a core refusal must not be an MCP error: %v", err)
	}
	if !res.IsError {
		t.Error("a core refusal must set IsError")
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "unknown scope") {
		t.Errorf("the refusal does not carry its reason: %q", text)
	}
}

func TestTheAnswerTextComesBackAsTextContent(t *testing.T) {
	session := connect(t, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(answer.Request, string, string, func(string)) (string, []string, error) {
			return "the catalog", nil, nil
		},
	})
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_catalog"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatal("a good answer must not set IsError")
	}
	if got := res.Content[0].(*mcp.TextContent).Text; got != "the catalog" {
		t.Errorf("text is %q", got)
	}
}
```

- [ ] **Step 2: Tests laufen lassen**

Erwartet: FAIL, das Paket gibt es nicht.

- [ ] **Step 3: `tools.go` schreiben**

```go
// Package brain registers the five knowledge tools on an MCP server.
//
// It is the thinnest possible layer: it maps one tool call to one answer.Run
// and turns the three return values into the three things MCP has for them.
package brain

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/mcptools"
)

// Deps are what the tools need. A test replaces Answer and needs no registry.
type Deps struct {
	Answer      func(answer.Request, string, string, func(string)) (string, []string, error)
	RegistryDir string
	LegacyDir   string
}

// Register adds the five tools to server. The channel is the listener's, never
// an argument: a channel a caller can name is a claim.
func Register(server *mcp.Server, channel privacy.Channel, deps Deps) {
	for _, tool := range mcptools.Tools() {
		server.AddTool(tool, handler(tool.Name, channel, deps))
	}
}

func handler(name string, channel privacy.Channel, deps Deps) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := arguments(req)
		request := answer.Request{
			Command: command(name),
			Query:   text(args, "query", "relative"),
			Scope:   str(args, "scope"),
			Profile: str(args, "profile"),
			Count:   count(args),
			Section: str(args, "section"),
			Channel: channel,
		}
		notice := func(message string) { report(ctx, req, message) }
		out, notes, err := deps.Answer(request, deps.RegistryDir, deps.LegacyDir, notice)
		if err != nil {
			// What the core refuses is content for the model, not an outage:
			// the model can act on "unknown scope", and reporting it as a
			// protocol error would take that chance away.
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			}, nil
		}
		for _, note := range notes {
			report(ctx, req, note)
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: out}}}, nil
	}
}

// report sends one line upwards as progress, never to stderr: no host reads our
// stderr. Without a progress token from the host the line lapses, and that is
// right -- a message with no recipient is not one.
func report(ctx context.Context, req *mcp.CallToolRequest, message string) {
	token := req.Params.GetProgressToken()
	if token == nil {
		return
	}
	// A failed notification must not fail the answer: the answer stands either
	// way, and the host asked for the answer, not for the commentary.
	_ = req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
		ProgressToken: token,
		Message:       message,
	})
}
```

Die Helfer, ausgeschrieben — `req.Params.Arguments` ist `json.RawMessage`
(`protocol.go:250`), weil die Werkzeuge über `(*Server).AddTool` registriert
werden und nicht über das generische `mcp.AddTool[In, Out]`; das Auspacken ist
also unsere Sache:

```go
// arguments unpacks the raw call arguments. A call without arguments and a call
// with broken ones both end as an empty map: the schema already told the host
// what is required, and a parse error here would be an outage, which this is
// not.
func arguments(req *mcp.CallToolRequest) map[string]any {
	if len(req.Params.Arguments) == 0 {
		return map[string]any{}
	}
	var args map[string]any
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return map[string]any{}
	}
	return args
}

// command turns brain_search into search. The family prefix is for the host's
// flat tool list; the answer knows the bare names.
func command(name string) string {
	return strings.TrimPrefix(name, "brain_")
}

func str(args map[string]any, key string) string {
	value, ok := args[key].(string)
	if !ok {
		return ""
	}
	return value
}

// text is the one positional the answer takes: the query for search, the
// relative path for read and neighbors. They never occur together.
func text(args map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := str(args, key); value != "" {
			return value
		}
	}
	return ""
}

// count reads n. JSON numbers arrive as float64. Zero means "not given", and
// answer.Run puts its own default in its place.
func count(args map[string]any) int {
	value, ok := args["n"].(float64)
	if !ok {
		return 0
	}
	return int(value)
}
```

- [ ] **Step 4: Tests laufen lassen**

```powershell
go test ./internal/serve/brain/ -v -cover
```

Erwartet: PASS, 100 % je Funktion. Für `report` ohne Token und mit Token je ein
Test.

- [ ] **Step 5: Commit**

Nachricht: `Serve the five brain tools over MCP`.

---

## Task 8: `serve` — Listener, Lebenszyklus, Sperre

**Files:**
- Create: `internal/serve/serve.go`
- Test: `internal/serve/serve_test.go`

**Interfaces:**
- Consumes: `internal/lock` (Task 2), `state.go` (Task 5), `auth.go` (Task 6), `internal/serve/brain` (Task 7).
- Produces:

```go
type Options struct {
	StateDir   string
	RegistryDir string
	LegacyDir  string
	Foreground bool
	BrokeAway  bool
}

// Run takes serve.lock, binds both listeners, writes serve.json and blocks
// until ctx ends or a stop request arrives. It returns nil on an orderly stop
// and ErrAlreadyRunning when another serve holds the lock.
func Run(ctx context.Context, opts Options) error

var ErrAlreadyRunning = errors.New("another loomux serve already holds the lock")
```

**Zustandslos** (`StreamableHTTPOptions.Stateless: true`): kein
`Mcp-Session-Id`, GET und DELETE antworten mit 405, je Anfrage eine temporäre
Sitzung. Das ist die Richtung der Revision 2026-07-28 (SEP-2567) und nimmt dem
Neustart seinen komplizierten Fall — es gibt keine „unbekannte Sitzung" mehr,
nur eine abgelehnte Verbindung auf einem Port, den es nicht mehr gibt.
Fortschrittsmeldungen bleiben möglich, solange sie im Kontext einer laufenden
Anfrage entstehen, und genau so entstehen unsere (Task 7).

- [ ] **Step 1: Die fehlschlagenden Tests schreiben**

`internal/serve/serve_test.go`:

```go
package serve_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/serve"
)

// start runs serve in the background and waits for serve.json to appear.
func start(t *testing.T, dir string) (*serve.State, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve.Run(ctx, serve.Options{StateDir: dir, RegistryDir: dir, LegacyDir: dir}) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("serve did not stop within ten seconds")
		}
	})
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if state, err := serve.ReadState(dir); err == nil && state.Local.URL != "" {
			return state, cancel
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("serve did not write its state within ten seconds")
	return nil, cancel
}

func TestBothListenersAnswerOnLoopbackWithDifferentTokens(t *testing.T) {
	dir := t.TempDir()
	state, _ := start(t, dir)

	for name, endpoint := range map[string]serve.Endpoint{"local": state.Local, "cloud": state.Cloud} {
		if !strings.HasPrefix(endpoint.URL, "http://127.0.0.1:") {
			t.Errorf("%s listens on %q, not on loopback", name, endpoint.URL)
		}
	}
	if state.Local.URL == state.Cloud.URL {
		t.Error("both channels share one address")
	}
	if state.Local.Token == state.Cloud.Token {
		t.Error("both channels share one token")
	}
}

func TestTheWrongTokenIsRefused(t *testing.T) {
	dir := t.TempDir()
	state, _ := start(t, dir)

	// The cloud token on the local listener: a caller who has one channel's
	// secret must not reach the other's door.
	req, err := http.NewRequest(http.MethodPost, state.Local.URL, strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+state.Cloud.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status %d, want 401", res.StatusCode)
	}
}

func TestASecondServeRefusesToStart(t *testing.T) {
	dir := t.TempDir()
	start(t, dir)

	// Run must return at once: TryAcquire refuses and Run gives up. The timeout
	// is a safety net for a broken implementation, never a wait -- an
	// implementation that blocks here and passes the test is wrong.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := serve.Run(ctx, serve.Options{StateDir: dir, RegistryDir: dir, LegacyDir: dir})
	if !errors.Is(err, serve.ErrAlreadyRunning) {
		t.Errorf("second Run returned %v, want ErrAlreadyRunning", err)
	}
}

func TestStateCarriesTheRunningBuild(t *testing.T) {
	dir := t.TempDir()
	state, _ := start(t, dir)
	if state.Executable == "" || state.Size == 0 || state.ModTime.IsZero() {
		t.Errorf("state does not describe the build: %+v", state)
	}
}

func TestGetIsRefusedInStatelessMode(t *testing.T) {
	dir := t.TempDir()
	state, _ := start(t, dir)

	req, err := http.NewRequest(http.MethodGet, state.Local.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+state.Local.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET gave status %d, want 405", res.StatusCode)
	}
}
```

- [ ] **Step 2: Tests laufen lassen**

Erwartet: FAIL.

- [ ] **Step 3: `serve.go` schreiben**

Der Kern, in dieser Reihenfolge:

```go
// Run holds the service for its whole life.
//
// The order matters: the lock first, so a second serve never binds a port; then
// the listeners, so serve.json can name real addresses; then the state file,
// which is the last thing a bridge waits for.
func Run(ctx context.Context, opts Options) error {
	handle, held, err := lock.TryAcquire(LockPath(opts.StateDir))
	if err != nil {
		return err
	}
	if !held {
		return ErrAlreadyRunning
	}
	defer handle.Release()

	local, err := listen(privacy.ChannelLocal, opts)
	if err != nil {
		return err
	}
	defer local.Close()
	cloud, err := listen(privacy.ChannelCloud, opts)
	if err != nil {
		return err
	}
	defer cloud.Close()
	// ... write State, serve both, wait for ctx or the stop endpoint ...
}
```

Je Kanal ein eigener `*mcp.Server` mit den Werkzeugen aus Task 7, dahinter:

```go
handler := mcp.NewStreamableHTTPHandler(
	func(*http.Request) *mcp.Server { return server },
	&mcp.StreamableHTTPOptions{
		// Stateless follows the 2026-07-28 direction (SEP-2567). It also kills
		// the hardest restart case: there is no session to go unknown, only a
		// port that is gone.
		Stateless: true,
		// Without this, any web page in the user's browser could shoot at
		// 127.0.0.1. The token does not help once it has been in a URL.
		CrossOriginProtection: http.NewCrossOriginProtection(),
	})
mux := http.NewServeMux()
mux.Handle("/mcp", RequireToken(token, handler))
```

Der Port ist 0, das Betriebssystem wählt; die tatsächliche Adresse kommt aus
`listener.Addr()` und geht nach `serve.json`.

Die `ServerOptions` jedes Kanals setzen das Zwischenspeichern der Werkzeugliste
aus `mcptools.CacheTTL` und `mcptools.CacheScope` — fünf statische Werkzeuge,
das kostet nichts. Ein Test prüft, dass `tools/list` die Werte trägt.

- [ ] **Step 4: Der Fortschritts-Test durch den echten Handler**

`InMemoryTransports` aus Task 7 beweist nicht, dass eine Fortschrittsmeldung
**über HTTP** ankommt. Im zustandslosen Betrieb geht das nur im Kontext einer
laufenden Anfrage, und genau darauf baut der Warm-Hinweis. Ein Test, der einen
echten Client gegen den echten Listener fährt, einen Progress-Token mitgibt und
die Meldung empfängt:

```go
func TestAProgressNotificationReachesTheClientOverHTTP(t *testing.T) {
	dir := t.TempDir()
	state, _ := start(t, dir)

	heard := make(chan string, 4)
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, &mcp.ClientOptions{
		ProgressNotificationHandler: func(_ context.Context, req *mcp.ProgressNotificationClientRequest) {
			heard <- req.Params.Message
		},
	})
	transport := &mcp.StreamableClientTransport{
		Endpoint: state.Local.URL,
		// A stateless server answers GET with 405; a client that opens a
		// standalone SSE stream would see an error that is none.
		DisableStandaloneSSE: true,
		HTTPClient:           tokenClient(state.Local.Token),
	}
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	params := &mcp.CallToolParams{Name: "brain_status"}
	params.SetProgressToken("p1")
	if _, err := session.CallTool(context.Background(), params); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	select {
	case <-heard:
	case <-time.After(5 * time.Second):
		t.Error("no progress notification arrived over HTTP")
	}
}
```

`tokenClient` ist ein `*http.Client` mit einem `RoundTripper`, der den
`Authorization`-Kopf setzt. Damit der Test überhaupt eine Meldung auslöst, gibt
der `status`-Handler in diesem Test eine feste Notiz zurück — über die
`Deps.Answer` aus Task 7.

- [ ] **Step 5: Tests laufen lassen**

```powershell
go test ./internal/serve/ -v -cover
```

Erwartet: PASS.

- [ ] **Step 6: Commit**

Nachricht: `Run two channel listeners under one lock`.

---

## Task 9: Start, Stopp, Status

**Files:**
- Create: `internal/serve/spawn.go`, `internal/serve/spawn_windows.go`, `internal/serve/spawn_other.go`, `internal/serve/control.go`
- Test: `internal/serve/spawn_test.go`, `internal/serve/control_test.go`

**Interfaces:**
- Produces:
  - `func Spawn(stateDir string, spawner func(cmd *exec.Cmd) error) (brokeAway bool, err error)`
    — `Spawn` baut den `exec.Cmd` samt Prozessattributen und Logdatei, der
    Spawner startet ihn. So kann ein Test `cmd.Stdout` und
    `cmd.SysProcAttr` prüfen, ohne einen Prozess zu erzeugen.
  - `func Status(stateDir string) (string, error)` — fragt die Listener, statt nur die Sperrdatei zu lesen.
  - `func Stop(stateDir string, force bool) error`

**Drei Regeln, jede mit ihrem Grund:**

1. **`serve` erbt nie die stdio des Aufrufers.** Der stdout einer Brücke ist die
   MCP-Leitung ihres Wirts; ein geerbter Deskriptor zerstört das Framing und
   hält die Leitung offen, wenn die Brücke endet. `cmd.Stdin/Stdout/Stderr`
   bleiben `nil` (das ergibt `os.DevNull`), das Log geht nach
   `LogPath(stateDir)`. `search/daemon.go:57` macht es für qmd bereits so.
2. **`DETACHED_PROCESS|CREATE_BREAKAWAY_FROM_JOB`** unter Windows, `Setsid`
   sonst. Scheitert das Breakaway-Flag — ein Wirt kann uns in ein Job-Object
   ohne `BREAKAWAY_OK` gesteckt haben —, wird ohne es gestartet, `BrokeAway`
   bleibt `false`, und `Status` sagt: dieser Dienst stirbt mit seinem Wirt.
3. **`Stop` ist ein Endpunkt, kein Signal.** POST auf den local-Listener mit dem
   local-Token; die Listener fahren geordnet herunter, die Sperre wird frei.
   Antwortet der Endpunkt nicht, bricht `Stop(dir, true)` den Prozess über die
   PID aus `serve.json` ab — nur auf dieses Flag hin, nie von allein.

- [ ] **Step 1: Die fehlschlagenden Tests schreiben**

Kernpunkte, je ein Test:

```go
func TestSpawnNeverInheritsStdio(t *testing.T) {
	var got *exec.Cmd
	_, err := serve.Spawn(t.TempDir(), func(cmd *exec.Cmd) error {
		got = cmd
		return nil
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	// nil means os.DevNull. The caller's stdout may be a host's MCP pipe; an
	// inherited descriptor would wreck its framing and hold it open.
	if got.Stdin != nil || got.Stdout != nil {
		t.Error("the child inherited stdin or stdout")
	}
	if len(got.Args) < 2 || got.Args[1] != "serve" {
		t.Errorf("args are %v, want the serve subcommand", got.Args)
	}
	if got.SysProcAttr == nil {
		t.Error("the child was built without detach attributes")
	}
}

func TestSpawnFallsBackWhenBreakawayIsRefused(t *testing.T) {
	calls := 0
	brokeAway, err := serve.Spawn(t.TempDir(), func(*exec.Cmd) error {
		calls++
		if calls == 1 {
			return errors.New("access is denied")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if calls != 2 {
		t.Errorf("the spawner was called %d times, want two: with and without breakaway", calls)
	}
	if brokeAway {
		t.Error("brokeAway must be false after the fallback")
	}
}

func TestStatusNamesTheServiceThatDiesWithItsHost(t *testing.T) {
	// A state written with BrokeAway=false must make Status say so, because the
	// user otherwise has no way to learn it.
}

func TestStopWithoutForceDoesNotKill(t *testing.T) {
	// No listener, no PID kill: Stop returns an error naming `serve status`.
}
```

- [ ] **Step 2: Tests laufen lassen** — Erwartet: FAIL.

- [ ] **Step 3: Implementieren** wie in den drei Regeln beschrieben.

- [ ] **Step 4: Tests und Tor** — Erwartet: PASS, 100 %.

- [ ] **Step 5: Commit**

Nachricht: `Start, stop and inspect the service`.

---

## Task 10: qmd unter geteilter Sperre

**Files:**
- Modify: `internal/brain/search/daemon.go`, `internal/brain/answer/answer.go`
- Create: `internal/brain/search/shared.go`
- Test: `internal/brain/search/shared_test.go`

**Interfaces:**
- Produces: `func EnsureDaemon(qmdLockPath string, env map[string]string, port int) error` — Probe und Start im kurzen kritischen Abschnitt.

**Was sich ändert:** `search.DefaultConnectWith` (`internal/brain/search/mcp.go:84`)
probt heute mit `session.Reachable()` und startet bei Bedarf mit
`StartDaemonWith`. Genau dieses Paar kommt in den kritischen Abschnitt — nicht
ein zweiter Startweg daneben.

**Warum geteilt:** `brain search` startet qmd, und `serve` tut es beim
Hochfahren auch. Ohne gemeinsame Sperre erzeugen zwei Starter zwei Daemons auf
8765. Die Sperre ist **nicht** `serve.lock`: die hält `serve` sein Leben lang,
ein zweiter Nehmer käme nie hinein.

**Und eine Ebene tiefer dieselbe Regel wie oben:** stirbt qmd unter einem
langlebigen `serve` weg, wird unter `qmd.lock` neu geprobt, bei Bedarf neu
gestartet und der Aufruf **genau einmal** wiederholt — sonst ein Fehler, nie
eine leere Trefferliste.

- [ ] **Step 1..5** analog: Test, der zwei nebenläufige `EnsureDaemon`-Aufrufe
  nur einen Start auslösen lässt; Test, der nach einem toten Daemon genau eine
  Wiederholung sieht; Implementierung; Tor; Commit
  (`Share one lock between the two qmd starters`).

---

## Task 11: Die Brücke

**Files:**
- Create: `internal/bridge/bridge.go`, `internal/bridge/connect.go`
- Test: `internal/bridge/bridge_test.go`, `internal/bridge/connect_test.go`

**Interfaces:**
- Consumes: `mcptools.Tools` (Task 4), `serve.ReadState`/`Spawn`/`Stop` (Tasks 5, 9).
- Produces:

```go
type Options struct {
	StateDir string
	Channel  privacy.Channel
	Spawn    func(stateDir string) (bool, error) // replaced in tests
}

func Run(ctx context.Context, opts Options) error
```

**Die Brücke ist ein Umleiter, kein zweiter Kern.** Name auf Name, Argumente auf
Argumente, Ergebnis zurück. Tut diese Schicht je mehr, ist etwas falsch.

Fünf Regeln, jede mit ihrem Grund:

1. **`tools/list` beantwortet sie selbst**, aus `mcptools`. Die Beschreibungen
   sind statisch, und ein Abruf legte einen qmd-Kaltstart mitten in den
   Handschlag.
2. **Der Start von `serve` läuft im Hintergrund**, nebenläufig zum `initialize`
   des Wirts. Der Wirt sieht sofort einen antwortenden Server. Ein gescheiterter
   Start reißt die Brücke **nicht** mit: der Wirt erfährt von der Störung durch
   den Aufruf, der `serve` braucht, nicht durch einen Server, der verschwindet.
3. **„Neuer gewinnt":** neu gestartet wird nur, wenn das eigene Programm neuer
   ist als das in `serve.json`. Sonst schössen zwei Wirte aus zwei Klonen
   einander bei jedem Aufruf ab.
4. **Der Neustart ist eine Abfolge:** Stop-Endpunkt mit dem local-Token, warten
   bis `serve.lock` frei ist (250-ms-Takt, 60 s), dann starten. Ein neues
   `serve` kann die Sperre nicht nehmen, solange das alte sie hält. Dafür liest
   die Brücke **beide** Token aus `serve.json`, unabhängig von ihrem `--channel`
   — sonst könnte eine cloud-Brücke nie neu starten. Das Tor bleibt die Adresse,
   die sie *anspricht*.
5. **Eine Wiederholung.** „Verbindung abgelehnt" auf einem weitergeleiteten
   Aufruf ⇒ `serve.json` neu lesen, neu aushandeln, **genau einmal**
   wiederholen; erst dann ein MCP-Fehler, der `loomux serve status` nennt.

**Der Client der Brücke setzt `DisableStandaloneSSE: true`** — ein zustandsloser
Server beantwortet GET mit 405, und ein Client, der dort einen SSE-Strom
aufmachen will, bekommt bei jedem Start einen Fehler zu sehen, der keiner ist.

**Der Progress-Token reist nach unten.** Ohne ihn stirbt der Warm-Hinweis in der
Brücke.

- [ ] **Step 1: Die fehlschlagenden Tests für `connect.go` schreiben**

`internal/bridge/connect_test.go`:

```go
package bridge_test

import (
	"errors"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/bridge"
	"github.com/xidus90/loomux/internal/serve"
)

func stateWithBuild(modTime time.Time) *serve.State {
	return &serve.State{
		Local:      serve.Endpoint{URL: "http://127.0.0.1:1/mcp", Token: "l"},
		Cloud:      serve.Endpoint{URL: "http://127.0.0.1:2/mcp", Token: "c"},
		PID:        4711,
		Executable: `C:\other-checkout\bin\loomux.exe`,
		Size:       17,
		ModTime:    modTime,
	}
}

func TestAnOlderBridgeNeverRestartsANewerService(t *testing.T) {
	// serve is machine-wide, bin/loomux.exe sits in one checkout of several.
	// "Anything different restarts it" would let two hosts out of two clones
	// kill each other on every call.
	now := time.Now()
	decision := bridge.Decide(stateWithBuild(now), now.Add(-time.Hour))
	if decision != bridge.Keep {
		t.Errorf("decision is %v, want Keep", decision)
	}
}

func TestANewerBridgeRestartsTheService(t *testing.T) {
	now := time.Now()
	decision := bridge.Decide(stateWithBuild(now), now.Add(time.Hour))
	if decision != bridge.Restart {
		t.Errorf("decision is %v, want Restart", decision)
	}
}

func TestTheSameBuildIsKept(t *testing.T) {
	now := time.Now()
	if decision := bridge.Decide(stateWithBuild(now), now); decision != bridge.Keep {
		t.Errorf("decision is %v, want Keep", decision)
	}
}

func TestNoStateMeansStart(t *testing.T) {
	if decision := bridge.Decide(nil, time.Now()); decision != bridge.Start {
		t.Errorf("decision is %v, want Start", decision)
	}
}

func TestRestartStopsTheOldServiceBeforeStarting(t *testing.T) {
	// A newly spawned serve cannot take serve.lock while the old one holds it.
	// Stop, wait for the lock, then spawn -- in that order, or the new one dies
	// on startup and the bridge waits sixty seconds for nothing.
	var order []string
	deps := bridge.Deps{
		Stop:     func(string) error { order = append(order, "stop"); return nil },
		WaitFree: func(string, time.Duration) error { order = append(order, "wait"); return nil },
		Spawn:    func(string) (bool, error) { order = append(order, "spawn"); return true, nil },
	}
	if err := bridge.Restart(t.TempDir(), deps); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	want := []string{"stop", "wait", "spawn"}
	if len(order) != len(want) {
		t.Fatalf("order is %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order is %v, want %v", order, want)
		}
	}
}

func TestRestartStillSpawnsWhenTheOldServiceIsAlreadyGone(t *testing.T) {
	// A stop against a dead listener is not a failure: the goal is a free lock,
	// and a service that is already gone has reached it.
	deps := bridge.Deps{
		Stop:     func(string) error { return errors.New("connection refused") },
		WaitFree: func(string, time.Duration) error { return nil },
		Spawn:    func(string) (bool, error) { return true, nil },
	}
	if err := bridge.Restart(t.TempDir(), deps); err != nil {
		t.Fatalf("Restart: %v", err)
	}
}
```

- [ ] **Step 2: Tests laufen lassen**

```powershell
go test ./internal/bridge/ -v
```

Erwartet: FAIL, das Paket gibt es nicht.

- [ ] **Step 3: `connect.go` schreiben**

```go
// Package bridge is the stdio front a host starts: one redirector per host.
//
// Name to name, arguments to arguments, result back. If this layer ever does
// more than that, something is wrong.
package bridge

import (
	"fmt"
	"time"

	"github.com/xidus90/loomux/internal/serve"
)

// Decision is what to do with the service described by serve.json.
type Decision int

const (
	// Keep uses the running service as it is.
	Keep Decision = iota
	// Start spawns one because none is recorded.
	Start
	// Restart replaces an older one with this build.
	Restart
)

// Decide reads the recorded build against ours. Newer wins, and only newer.
func Decide(state *serve.State, ours time.Time) Decision {
	if state == nil {
		return Start
	}
	if state.OlderThan(ours) {
		return Restart
	}
	return Keep
}

// Deps are the three moves a restart makes. A test replaces all three.
type Deps struct {
	Stop     func(stateDir string) error
	WaitFree func(lockPath string, timeout time.Duration) error
	Spawn    func(stateDir string) (brokeAway bool, err error)
}

// StartTimeout and StartTick are the qmd handshake's figures from stage 1b-1,
// reused deliberately: one waiting rhythm in this project, not two.
const (
	StartTimeout = 60 * time.Second
	StartTick    = 250 * time.Millisecond
)

// Restart replaces the running service with this build.
//
// The order is forced: a spawned serve cannot take serve.lock while the old one
// holds it. Stopping first is not politeness, it is the only order that works.
func Restart(stateDir string, deps Deps) error {
	// A stop that fails is not an error: a service that is already gone has
	// reached the goal this call has, which is a free lock.
	_ = deps.Stop(stateDir)
	if err := deps.WaitFree(serve.LockPath(stateDir), StartTimeout); err != nil {
		return fmt.Errorf("the old service did not let go of its lock: %w", err)
	}
	if _, err := deps.Spawn(stateDir); err != nil {
		return fmt.Errorf("start the service: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Die Tests für `bridge.go` schreiben**

`internal/bridge/bridge_test.go` — hier sind die drei Verhaltensweisen, an denen
die Brücke gemessen wird:

```go
func TestToolsListIsAnsweredWithoutAService(t *testing.T) {
	// The descriptions are static, and fetching them would put a cold qmd start
	// inside the host's handshake. No service runs in this test at all.
	session := connectBridge(t, bridge.Options{StateDir: t.TempDir()})
	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(res.Tools) != 5 {
		t.Fatalf("got %d tools without a service, want 5", len(res.Tools))
	}
	if res.Tools[0].Name != "brain_search" {
		t.Errorf("first tool is %q", res.Tools[0].Name)
	}
}

func TestTheListIsByteIdenticalToTheServices(t *testing.T) {
	bridged, err := json.Marshal(toolsOf(t, connectBridge(t, bridge.Options{StateDir: t.TempDir()})))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	served, err := json.Marshal(mcptools.Tools())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(bridged) != string(served) {
		t.Error("the bridge's list and the service's list differ")
	}
}

func TestACallIsForwardedUnchanged(t *testing.T) {
	// Name to name, arguments to arguments. The fake service records what
	// arrived.
	var seen *mcp.CallToolParamsRaw
	service := fakeService(t, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		seen = req.Params
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
	})
	session := connectBridge(t, bridge.Options{StateDir: service.StateDir})
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "brain_read",
		Arguments: map[string]any{"scope": "wiki", "relative": "index.md"},
	}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if seen.Name != "brain_read" {
		t.Errorf("the service saw %q", seen.Name)
	}
	if !strings.Contains(string(seen.Arguments), `"relative":"index.md"`) {
		t.Errorf("the service saw arguments %s", seen.Arguments)
	}
}

func TestARefusedConnectionIsRetriedExactlyOnce(t *testing.T) {
	// A commit rebuilds bin/loomux.exe, a newer bridge restarts serve, and every
	// other bridge's connection dies with it. One retry hides that from a host
	// that did nothing wrong. Two retries would hide a real outage.
	attempts := 0
	service := fakeService(t, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		attempts++
		if attempts == 1 {
			return nil, errors.New("connection refused")
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
	})
	session := connectBridge(t, bridge.Options{StateDir: service.StateDir})
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if attempts != 2 {
		t.Errorf("the service was called %d times, want 2", attempts)
	}
	if res.IsError {
		t.Error("the retried call came back as an error")
	}
}

func TestAnOutageIsAnErrorAndNeverAnEmptyResult(t *testing.T) {
	// Reporting an outage as an empty hit list would teach the model to read a
	// dead service as "nothing found".
	service := fakeService(t, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, errors.New("connection refused")
	})
	session := connectBridge(t, bridge.Options{StateDir: service.StateDir})
	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_search",
		Arguments: map[string]any{"query": "anything"}})
	if err == nil {
		t.Fatal("an outage came back as a result")
	}
	if !strings.Contains(err.Error(), "loomux serve status") {
		t.Errorf("the error does not say what to do: %v", err)
	}
}

func TestTheProgressTokenTravelsDownwards(t *testing.T) {
	// Without it the warming hint dies in the bridge.
	var token any
	service := fakeService(t, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token = req.Params.GetProgressToken()
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
	})
	session := connectBridge(t, bridge.Options{StateDir: service.StateDir})
	params := &mcp.CallToolParams{Name: "brain_status"}
	params.SetProgressToken("p1")
	if _, err := session.CallTool(context.Background(), params); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if token == nil {
		t.Error("the service saw no progress token")
	}
}
```

`fakeService` startet einen echten `serve`-ähnlichen HTTP-Server mit **einem**
Werkzeughandler und schreibt eine `serve.json` in ein eigenes
Zustandsverzeichnis. `connectBridge` startet `bridge.Run` über
`mcp.NewInMemoryTransports` statt über echtes stdio.

- [ ] **Step 5: Tests laufen lassen** — Erwartet: FAIL.

- [ ] **Step 6: `bridge.go` schreiben**

Der Aufbau, mit den Gründen an den Stellen, an denen sie gelten:

```go
// Run serves one host until it goes away.
func Run(ctx context.Context, opts Options) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "loomux", Version: version.Version}, nil)
	for _, tool := range mcptools.Tools() {
		// Answered here rather than fetched: the descriptions are static, and
		// asking would put a cold start inside the handshake. The same list
		// object as the service's, so the two cannot drift.
		server.AddTool(tool, forward(opts))
	}

	// Nudged in the background: making sure a service exists blocks for the ten
	// seconds of a cold start, and the host must see a ready server long before
	// that. A failed start stays a failed start -- it must not take the
	// answering front down with it, or the host learns of the outage from a
	// server that vanishes instead of from the call that needed it.
	go func() { _ = ensure(opts) }()

	return server.Run(ctx, &mcp.StdioTransport{})
}
```

`forward` verbindet sich beim ersten Aufruf gegen die Adresse des eigenen
Kanals, reicht Name, Argumente und den Progress-Token durch und wiederholt genau
einmal:

```go
res, err := session.CallTool(ctx, params)
if err == nil {
	return res, nil
}
// serve.json neu lesen, neu verbinden, genau einmal wiederholen.
session, connectErr := reconnect(opts)
if connectErr != nil {
	return nil, outage(connectErr)
}
res, err = session.CallTool(ctx, params)
if err != nil {
	return nil, outage(err)
}
return res, nil
```

```go
// outage says what a human can do about it. It is never an empty result: that
// would read as "nothing found" to the model.
func outage(err error) error {
	return fmt.Errorf("the loomux service did not answer (%w); run `loomux serve status`", err)
}
```

Der Client setzt `DisableStandaloneSSE: true`: ein zustandsloser Server
beantwortet GET mit 405, und ein Client, der dort einen SSE-Strom aufmachen
will, bekommt bei jedem Start einen Fehler zu sehen, der keiner ist.

`ensure` liest `serve.json`, ruft `Decide` und je nach Entscheidung nichts,
`Spawn` oder `Restart`. Es liest **beide** Token aus `serve.json`, unabhängig
vom `--channel` — sonst könnte eine cloud-Brücke nie neu starten. Das Tor bleibt
die Adresse, die sie anspricht.

- [ ] **Step 7: Tests und Tor**

```powershell
go test ./internal/bridge/ -v -cover
sh .githooks/pre-commit
```

Erwartet: PASS, 100 % je Funktion.

- [ ] **Step 8: Commit**

Nachricht: `Bridge one stdio host to the service`.

---

## Task 12: Einstiegspunkte und die Importgrenze

**Files:**
- Create: `internal/cli/serve.go`, `internal/cli/mcp.go`, `internal/cli/imports_test.go`
- Modify: `internal/cli/commands.go`

**Interfaces:**
- Produces: `loomux serve [--foreground]`, `loomux serve status`, `loomux serve stop [--force]`, `loomux mcp [--channel local|cloud]`.

`--channel` fällt ohne Angabe auf `local` zurück, wie `loomux brain`
(`internal/cli/brainargs.go:103-107`).

- [ ] **Step 1: Den Test der Importgrenze schreiben**

```go
// TestHooksNeverImportServeOrBridge keeps the per-edit path away from the MCP
// SDK. The rule is structural, and a sentence in a spec does not hold it.
func TestHooksNeverImportServeOrBridge(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "github.com/xidus90/loomux/internal/hooks").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, forbidden := range []string{
		"github.com/xidus90/loomux/internal/serve",
		"github.com/xidus90/loomux/internal/bridge",
		"github.com/modelcontextprotocol/go-sdk/mcp",
	} {
		if strings.Contains(string(out), forbidden+"\n") {
			t.Errorf("internal/hooks depends on %s", forbidden)
		}
	}
}
```

- [ ] **Step 2..5:** Test laufen lassen (FAIL bzw. PASS je nach Stand), die zwei
  Einstiegspunkte schreiben, Argumentfehler im Stil von `brainRefuse`, Tor,
  Commit (`Wire serve and the bridge into the command line`).

---

## Task 13: Der Fallkorpus

**Files:**
- Modify: `internal/cases/`, `internal/dev/recordcase/`
- Create: `testdata/cases/1b-2/`, `testdata/cases/1b-2-map.toml`
- Test: `internal/cli/cases_1b2_test.go`

**Verglichen wird der Textinhalt der `CallToolResult` und `isError`**, nicht der
Umschlag. Grund: die Python-Front spricht über das Python-MCP-SDK, die Brücke
über das Go-SDK; `initialize` liefert schon deshalb andere Bytes, und die
Werkzeuge heißen ohnehin anders. Ein byteweiser Vergleich erzeugte eine
Paritätsliste, die überwiegend aus „Unterschied der Bibliothek" bestünde und bei
jeder SDK-Aktualisierung neu wüchse.

Drei Dinge, die das Gerüst braucht:

1. **Isolation über `LOOMUX_STATE_DIR`.** Ein Fall bekommt sein eigenes `serve`;
   die Pfade aus Task 5 hängen deshalb alle am Zustandsverzeichnis.
2. **Jeder Fall räumt sein `serve` weg** — am Fallende `serve stop` auf dem
   Zustandsverzeichnis des Falls, sonst lässt ein `go test`-Lauf entkoppelte
   Prozesse liegen.
3. **Ein neuer Vergleichsmodus**, kein Schalter am vorhandenen Byte-Vergleich.

`1b-2-map.toml` trägt die Umbenennung:

```toml
# Stage 1b-2 renames the five MCP tools into the brain family. The protocol has
# no nested tools, and the code graph will put graph_* into the same server, so
# the prefix is the only family separation available.

[[tool]]
from = "search"
to   = "brain_search"

[[tool]]
from = "catalog"
to   = "brain_catalog"

[[tool]]
from = "read"
to   = "brain_read"

[[tool]]
from = "neighbors"
to   = "brain_neighbors"

[[tool]]
from = "status"
to   = "brain_status"
```

- [ ] **Step 1..6:** Vergleichsmodus test-first bauen; Fälle gegen die
  Python-Front aufzeichnen (`uv run --no-sync`, `PYTHONDONTWRITEBYTECODE=1`,
  `PYTHONUTF8=1` wie in 1b-1); Fälle grün fahren; jede Abweichung in die
  Paritätsliste; Tor; Commit (`Record the MCP front against the reference`).

---

## Task 14: Abnahme der Stufe

**Files:**
- Create: `docs/.superpowers/parity/stufe-1b-2.md`
- Modify: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`, `docs/en/benchmarks.md`, `docs/de/benchmarks.md`, `docs/en/cli-reference.md`, `docs/de/cli-reference.md`, `README.md`, `README.de.md`

- [ ] **Step 1: Die Fusions-Spec entschärfen**

Zwei Zeilen dort sind überholt und bekommen einen Verweis auf die 1b-2-Spec,
damit der nächste Leser nicht dem älteren Dokument glaubt:
- „Werkzeuge wie heute: `search`, `catalog`, `read`, `neighbors`, `status`"
- „Den Kanal wählt die Brücke per Flag; er ist Teil der Anfrage, keine eigene Bindung"

- [ ] **Step 2: Die vier Messungen**

Chronologisch in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`, je kalt
und warm, Grundlinie gegen Änderung:

1. Binärgröße und Startboden vor und nach dem SDK (Grundlinie aus Task 0).
2. Der Hook-Pfad, unverändert nachgewiesen — dass `hooks` das SDK nicht anfasst,
   muss messbar bleiben, nicht nur strukturell stimmen.
3. Handschlag der Brücke: Start bis Antwort auf `initialize`, mit laufendem und
   mit kaltem `serve`.
4. Ein `brain_search` über die Brücke gegen dasselbe `loomux brain search`
   direkt — der Aufpreis der zwei Sprünge.

- [ ] **Step 3: Mutationsrunde**

```powershell
go run ./cmd/loomux dev mutants --packages ./internal/serve/...,./internal/bridge/...,./internal/lock/...,./internal/mcptools/...
```

Überlebende dokumentieren wie in `parity/stufe-1b-1-geparkte-mutanten.md`.

- [ ] **Step 4: Paritätsliste**

`docs/.superpowers/parity/stufe-1b-2.md` im Format von `stufe-1b-1.md`: Fall,
Alt, Neu, Begründung, Freigabe. Ausdrücklich eigene Zeilen für die
Werkzeugumbenennung, für `n = 10` gegen `n = 5`, und für den Vergleichsmodus.

- [ ] **Step 5: Dokumentation**

`docs/*/cli-reference.md` um `serve` und `mcp` erweitern, beide READMEs auf den
tatsächlichen Stand bringen.

- [ ] **Step 6: Mensch-Schritte**

- Die `.mcp.json`-Einträge der Wirte auf `loomux mcp --channel local` zeigen
  lassen; `loomux init` übernimmt das erst in Stufe 4.
- Freigabe der Paritätsliste.
- **Rauchtest:** ein `brain_search` aus Claude Code über die Brücke, ein zweiter
  Wirt parallel, ein Commit dazwischen — er zeigt, ob „neuer gewinnt" und die
  Ein-Wiederholung tragen.

- [ ] **Step 7: Commit**

Nachricht: `Close stage 1b-2 with the parity list and the measurements`.

---

## Fertig ist die Stufe, wenn

1. alle Fälle der Stufe grün sind oder freigegeben in der Paritätsliste stehen,
2. die Coverage 100 % je Funktion ist, jeder Ausschluss begründet,
3. die Mutationsrunde gelaufen ist und ihre Überlebenden dokumentiert sind,
4. die vier Messungen in `docs/{en,de}/benchmarks.md` stehen,
5. das loomux-Repo den Dienst selbst benutzt — ein Wirt dieses Projekts spricht
   über die Brücke mit `serve`. **Das ist ein Mensch-Schritt:** den
   `.mcp.json`-Eintrag schreibt der Mensch, kein Subagent.
