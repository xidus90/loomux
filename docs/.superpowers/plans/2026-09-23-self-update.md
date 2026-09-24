# Self-Update des maschinenweiten Binarys — Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `serve` hält `%LOCALAPPDATA%\loomux\bin\loomux.exe` aus den
GitHub-Releases seines Kanals aktuell, `loomux self-update` tut dasselbe von
Hand, und der Sitzungsstart warnt, wenn `serve` woanders läuft oder das Update
scheitert.

**Architecture:** Ein neues Paket `internal/selfupdate` kapselt Versionswahl,
`gh`-Aufrufe, Prüfsumme, Tausch und die Statusdatei `update.json`. `serve`
bekommt eine Update-Schleife, die ein von der Kommandozeile gereichtes
`Update`-Callback ruft; `serve` selbst kennt keine Releases. Der
Sitzungsstart-Hook liest nur `update.json`. Der vorhandene Tausch
`internal/dev/swap` zieht nach `internal/swap`.

**Tech Stack:** Go (Modul `github.com/xidus90/loomux`), `gh`-CLI zur Laufzeit,
`internal/lock` für Sperre und atomares Schreiben.

**Spec:** `docs/.superpowers/specs/2026-09-23-self-update-design.md`

## Global Constraints

- Coverage 100 % pro Funktion; eine Ausnahme nur mit
  `//coverage:exempt <reason>` direkt über `func`.
- Kein `init()`, keine Paketvariable parst eingebettete Daten.
- Code, Bezeichner, Kommentare, Fehlermeldungen, Commits: Englisch. Doku unter
  `docs/de/` und `README.de.md`: Deutsch.
- Commits nach Conventional Commits, ohne Verweis auf Plan, Stufe oder Task,
  ohne Co-Author-Zeile auf ein Modell. Mehrzeilige Nachrichten über
  `git commit -F <datei>`.
- Niemand pusht außer dem Menschen. Niemand arbeitet auf `master`; Zweig ist
  `feat/self-update`.
- `.loomux/config.toml` schreibt kein Agent.
- `selfupdate` importiert nie `serve`, `hooks`, `cli` oder `bridge`.
- `hooks` importiert nie `serve` oder `bridge` (`internal/cli/imports_test.go`).
- Kanonischer Ort: `filepath.Join(stateDir, "bin", "loomux.exe")`.
- Repo der Releases: `xidus90/loomux`. Asset:
  `loomux_<ver>_<goos>_<goarch>` plus `.exe` unter Windows; Summen in
  `SHA256SUMS`.
- Jeder Aufruf eines externen Programms hat 2 Minuten Frist.
- Subagenten: `model: "opus"`, `effort: "low"`, beides explizit.
- Gate vor jedem Commit: `.githooks/pre-commit` läuft von selbst
  (`sh ci/gate.sh`). Ein Task ist erst fertig, wenn es grün ist.

## Dateien

| Datei | Verantwortung |
|---|---|
| `internal/swap/` (verschoben aus `internal/dev/swap/`) | Laufendes Binary beiseite, neues an seinen Platz |
| `internal/selfupdate/version.go` | semver lesen, `Newer`, Release je Kanal wählen |
| `internal/selfupdate/status.go` | `Outcome`, `Status`, `update.json`, kanonischer Ort |
| `internal/selfupdate/gh.go` | `Runner`, `ExecRunner`, Frist, `latest` |
| `internal/selfupdate/fetch.go` | Download, Prüfsumme, `--version`, Stempel, Ablage als `loomux.new.exe` |
| `internal/selfupdate/update.go` | `Options`, `Result`, `Run` |
| `internal/selfupdate/fake_test.go` | Test-Double für `gh` und das geladene Binary |
| `internal/serve/update.go` | `UpdateLoop` |
| `internal/serve/serve.go` | `Options.Update`, `Options.UpdateAfter`, Start der Schleife |
| `internal/cli/selfupdate.go` | Befehl `loomux self-update`, `selfUpdateOptions` |
| `internal/cli/serve.go` | `serveForeground` reicht `Update` an `serve` |
| `internal/hooks/hook_session_start.go` | `updateWarnings` |
| Doku | `README.md`, `README.de.md`, `docs/{en,de}/cli-reference.md`, `docs/{en,de}/migration.md` |

---

### Task 1: `swap` aus den Dev-Werkzeugen lösen

**Files:**
- Move: `internal/dev/swap/` → `internal/swap/` (alle drei Dateien)
- Modify: `internal/cli/dev.go:26` (Import)

**Interfaces:**
- Produces: `swap.Swap(dir string) error` unter
  `github.com/xidus90/loomux/internal/swap`, Verhalten unverändert.

- [ ] **Step 1: Verschieben**

```bash
git mv internal/dev/swap internal/swap
```

- [ ] **Step 2: Import anpassen**

In `internal/cli/dev.go` die Zeile

```go
	"github.com/xidus90/loomux/internal/dev/swap"
```

ersetzen durch

```go
	"github.com/xidus90/loomux/internal/swap"
```

- [ ] **Step 3: Paketkommentar erweitern**

In `internal/swap/swap.go` die erste Zeile

```go
// Package swap replaces the pilot binary while hooks may be running it.
```

ersetzen durch

```go
// Package swap replaces a loomux binary while processes may be running it:
// the pilot binary of a checkout, and the machine-wide one the self-update
// installs.
```

- [ ] **Step 4: Tests laufen lassen**

Run: `go test ./internal/swap/ ./internal/cli/ -run 'Swap|Dev'`
Expected: PASS. `go build ./...` ohne Fehler; `grep -rn "dev/swap" --include=*.go .` findet nichts mehr.

- [ ] **Step 5: Commit**

```bash
git add -A internal/swap internal/dev internal/cli/dev.go
git commit -m "refactor(swap): move the binary swap out of the dev tools"
```

---

### Task 2: Versionen lesen und das Release des Kanals wählen

**Files:**
- Create: `internal/selfupdate/version.go`
- Test: `internal/selfupdate/version_test.go`

**Interfaces:**
- Produces:
  - `func Newer(tag, running string) bool`
  - `type Release struct { Tag string \`json:"tagName"\`; Prerelease bool \`json:"isPrerelease"\` }`
  - `func pick(releases []Release, channel string) (Release, bool)`
  - `func parseVersion(s string) (version, bool)`

- [ ] **Step 1: Failing test schreiben**

`internal/selfupdate/version_test.go`:

```go
package selfupdate

import "testing"

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		tag, running string
		want         bool
	}{
		{"v2.8.0", "2.7.0", true},
		{"2.8.0", "2.7.0", true},
		{"v10.0.0", "9.9.9", true},
		{"v2.7.0", "2.7.0", false},
		{"v2.6.9", "2.7.0", false},
		{"v2.8.0-rc1", "2.7.0", false},
		{"v2.8", "2.7.0", false},
		{"v2.08.0", "2.7.0", false},
		{"v-1.0.0", "0.0.0", false},
		{"v2.8.0", "0.0.0-dev", false},
	} {
		if got := Newer(c.tag, c.running); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.tag, c.running, got, c.want)
		}
	}
}

// Every release so far is a pre-release: release.sh marks each one so while
// RELEASE_CHANNEL is unset. A beta binary must still find them.
func TestPickFollowsTheChannel(t *testing.T) {
	all := []Release{
		{Tag: "v2.6.0", Prerelease: true},
		{Tag: "v2.10.0", Prerelease: true},
		{Tag: "v2.7.0", Prerelease: false},
		{Tag: "nightly", Prerelease: true},
	}
	onlyPre := []Release{{Tag: "v2.7.0", Prerelease: true}, {Tag: "v2.6.0", Prerelease: true}}
	for _, c := range []struct {
		name     string
		releases []Release
		channel  string
		want     string
		found    bool
	}{
		{"beta takes the highest of all", all, "beta", "v2.10.0", true},
		{"stable skips pre-releases", all, "stable", "v2.7.0", true},
		{"no channel reads as stable", all, "", "v2.7.0", true},
		{"beta over today's releases", onlyPre, "beta", "v2.7.0", true},
		{"stable over today's releases", onlyPre, "stable", "", false},
		{"nothing listed", nil, "beta", "", false},
	} {
		got, found := pick(c.releases, c.channel)
		if found != c.found || got.Tag != c.want {
			t.Errorf("%s: pick = %q, %v; want %q, %v", c.name, got.Tag, found, c.want, c.found)
		}
	}
}
```

- [ ] **Step 2: Test laufen lassen, er muss fehlschlagen**

Run: `go test ./internal/selfupdate/`
Expected: FAIL, `undefined: Newer`.

- [ ] **Step 3: Implementieren**

`internal/selfupdate/version.go`:

```go
// Package selfupdate keeps the machine-wide loomux binary at the newest
// release of its channel: it asks gh for the releases, downloads one, checks
// it against SHA256SUMS and its own --version, and swaps it in. What happened
// is written to update.json, which session start reads.
package selfupdate

import (
	"strconv"
	"strings"
)

// version is a release number without a suffix; the tags this repository
// cuts carry none.
type version [3]int

// parseVersion reads "v1.2.3" or "1.2.3". Anything else, a suffix or a
// leading zero included, is not a version the updater acts on.
func parseVersion(s string) (version, bool) {
	parts := strings.Split(strings.TrimPrefix(s, "v"), ".")
	if len(parts) != 3 {
		return version{}, false
	}
	var v version
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || part != strconv.Itoa(n) {
			return version{}, false
		}
		v[i] = n
	}
	return v, true
}

func (v version) less(w version) bool {
	for i := range v {
		if v[i] != w[i] {
			return v[i] < w[i]
		}
	}
	return false
}

// Newer reports whether tag names a later release than running. A side that
// does not parse is never newer, so a guess never replaces a binary and a
// development build never asks for one.
func Newer(tag, running string) bool {
	t, ok := parseVersion(tag)
	if !ok {
		return false
	}
	r, ok := parseVersion(running)
	if !ok {
		return false
	}
	return r.less(t)
}

// Release is one entry of `gh release list --json tagName,isPrerelease`.
type Release struct {
	Tag        string `json:"tagName"`
	Prerelease bool   `json:"isPrerelease"`
}

// pick is the highest release the channel takes, by version rather than by
// publication. The stable channel, and a build that names none, takes no
// pre-release; every other channel takes both.
func pick(releases []Release, channel string) (Release, bool) {
	stable := channel == "" || channel == "stable"
	var best Release
	var bestVersion version
	found := false
	for _, r := range releases {
		if stable && r.Prerelease {
			continue
		}
		v, ok := parseVersion(r.Tag)
		if !ok {
			continue
		}
		if !found || bestVersion.less(v) {
			best, bestVersion, found = r, v, true
		}
	}
	return best, found
}
```

- [ ] **Step 4: Tests laufen lassen**

Run: `go test ./internal/selfupdate/ -cover`
Expected: PASS, `coverage: 100.0% of statements`.

- [ ] **Step 5: Commit**

```bash
git add internal/selfupdate
git commit -m "feat(selfupdate): pick the highest release of the running channel"
```

---

### Task 3: Statusdatei und kanonischer Ort

**Files:**
- Create: `internal/selfupdate/status.go`
- Test: `internal/selfupdate/status_test.go`

**Interfaces:**
- Consumes: `lock.ReplaceText(path, text string) error` aus `internal/lock`.
- Produces:
  - `type Outcome string` mit `Current`, `Updated`, `Skipped`, `Failed`, `Busy`
  - `type Status struct { CheckedAt time.Time; Executable, Running string; Result Outcome; Version, Error string }` (JSON-Namen siehe Code)
  - `func StatusPath(stateDir string) string`
  - `func Canonical(stateDir string) string`
  - `func IsCanonical(path, stateDir string) bool`
  - `func WriteStatus(stateDir string, s Status) error`
  - `func ReadStatus(stateDir string) (*Status, error)` — `nil, nil` wenn die Datei fehlt

- [ ] **Step 1: Failing tests schreiben**

`internal/selfupdate/status_test.go`:

```go
package selfupdate

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestReadStatusIsNilWithoutAFile(t *testing.T) {
	st, err := ReadStatus(t.TempDir())
	if st != nil || err != nil {
		t.Fatalf("ReadStatus = %v, %v; want nil, nil", st, err)
	}
}

func TestStatusRoundTrips(t *testing.T) {
	dir := t.TempDir()
	want := Status{
		CheckedAt:  time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC),
		Executable: `C:\loomux.exe`,
		Running:    "2.7.0",
		Result:     Updated,
		Version:    "2.8.0",
	}
	if err := WriteStatus(dir, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadStatus(dir)
	if err != nil || got == nil || *got != want {
		t.Fatalf("ReadStatus = %+v, %v; want %+v", got, err, want)
	}
	data, _ := os.ReadFile(StatusPath(dir))
	if !strings.Contains(string(data), `"result": "updated"`) {
		t.Fatalf("update.json does not carry the spec's field names:\n%s", data)
	}
}

func TestReadStatusRefusesBrokenJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(StatusPath(dir), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadStatus(dir); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("err = %v, want a parse error", err)
	}
}

func TestReadStatusReportsAnUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(StatusPath(dir), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadStatus(dir); err == nil || !strings.Contains(err.Error(), "read") {
		t.Fatalf("err = %v, want a read error", err)
	}
}

func TestWriteStatusFailsWhenTheStateDirectoryIsAFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "state")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteStatus(file, Status{}); err == nil {
		t.Fatal("WriteStatus into a file succeeded")
	}
}

func TestIsCanonical(t *testing.T) {
	dir := t.TempDir()
	canonical := Canonical(dir)
	if IsCanonical(canonical, dir) {
		t.Fatal("a canonical path that does not exist counts as canonical")
	}
	if err := os.MkdirAll(filepath.Dir(canonical), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(canonical, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(dir, "other.exe")
	if err := os.WriteFile(other, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsCanonical(canonical, dir) {
		t.Fatal("the canonical binary is not canonical")
	}
	if IsCanonical(other, dir) {
		t.Fatal("another file counts as canonical")
	}
	if IsCanonical(filepath.Join(dir, "missing.exe"), dir) {
		t.Fatal("a missing file counts as canonical")
	}
	if runtime.GOOS == "windows" && !IsCanonical(strings.ToUpper(canonical), dir) {
		t.Fatal("the same file in other letter case is not canonical")
	}
}
```

- [ ] **Step 2: Tests laufen lassen, sie müssen fehlschlagen**

Run: `go test ./internal/selfupdate/`
Expected: FAIL, `undefined: ReadStatus`.

- [ ] **Step 3: Implementieren**

`internal/selfupdate/status.go`:

```go
package selfupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/xidus90/loomux/internal/lock"
)

// Outcome is what one update pass came to.
type Outcome string

const (
	Current Outcome = "current"
	Updated Outcome = "updated"
	Skipped Outcome = "skipped"
	Failed  Outcome = "failed"
	// Busy means another pass held the lock. It is never written: the pass
	// that holds the lock writes its own result.
	Busy Outcome = "busy"
)

// Status is update.json: what the last pass found, for session start to read
// without asking the network or the service.
type Status struct {
	CheckedAt  time.Time `json:"checked_at"`
	Executable string    `json:"executable"`
	Running    string    `json:"running"`
	Result     Outcome   `json:"result"`
	Version    string    `json:"version"`
	Error      string    `json:"error"`
}

// StatusPath is update.json in the state directory.
func StatusPath(stateDir string) string { return filepath.Join(stateDir, "update.json") }

// Canonical is where the machine-wide binary lives. Nothing else is ever
// updated.
func Canonical(stateDir string) string { return filepath.Join(stateDir, "bin", "loomux.exe") }

// IsCanonical asks the file system rather than comparing strings: on Windows
// os.Executable may spell the same file in other letters or another form.
func IsCanonical(path, stateDir string) bool {
	have, err := os.Stat(path)
	if err != nil {
		return false
	}
	want, err := os.Stat(Canonical(stateDir))
	if err != nil {
		return false
	}
	return os.SameFile(have, want)
}

// WriteStatus replaces update.json whole, so a reader never sees half of it.
func WriteStatus(stateDir string, s Status) error {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", stateDir, err)
	}
	// Marshalling a struct of strings and a time cannot fail.
	data, _ := json.MarshalIndent(s, "", "  ")
	return lock.ReplaceText(StatusPath(stateDir), string(data)+"\n")
}

// ReadStatus is update.json, or nil when no pass has written one yet.
func ReadStatus(stateDir string) (*Status, error) {
	data, err := os.ReadFile(StatusPath(stateDir))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", StatusPath(stateDir), err)
	}
	var s Status
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", StatusPath(stateDir), err)
	}
	return &s, nil
}
```

- [ ] **Step 4: Tests laufen lassen**

Run: `go test ./internal/selfupdate/ -cover`
Expected: PASS, 100 %. Liefert `ReadStatus` auf einem Verzeichnis unter
Linux keinen Fehler (Go meldet dort `is a directory` beim Lesen, sollte also
gehen), den Test nicht aufweichen, sondern melden.

- [ ] **Step 5: Commit**

```bash
git add internal/selfupdate
git commit -m "feat(selfupdate): record each update pass in update.json"
```

---

### Task 4: `gh` aufrufen, Release finden, herunterladen und prüfen

**Files:**
- Create: `internal/selfupdate/gh.go`, `internal/selfupdate/fetch.go`
- Create: `internal/selfupdate/fake_test.go`
- Test: `internal/selfupdate/gh_test.go`, `internal/selfupdate/fetch_test.go`

**Interfaces:**
- Consumes: `pick`, `Release` (Task 2).
- Produces:
  - `const Repo = "xidus90/loomux"`
  - `type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)`
  - `func ExecRunner(ctx context.Context, name string, args ...string) ([]byte, error)`
  - `func AssetName(ver, goos, goarch string) string`
  - `func latest(ctx context.Context, run Runner, channel string) (Release, error)`
  - `func fetch(ctx context.Context, o Options, tag, dir string) error` — legt `dir/loomux.new.exe` ab
  - `var callTimeout = 2 * time.Minute`
- `Options` wird in Task 5 vollständig definiert; für diesen Task genügen die
  Felder `GOOS`, `GOARCH`, `Run`, `Now`. Lege `Options` deshalb schon hier in
  `update.go` an, mit genau diesen vier Feldern; Task 5 ergänzt die übrigen.

- [ ] **Step 1: Test-Double schreiben**

`internal/selfupdate/fake_test.go`:

```go
package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// fakeGH answers the three calls an update makes: the release list, the
// download into --dir, and the downloaded binary's --version.
type fakeGH struct {
	list    string
	files   map[string]string
	version string
	fail    map[string]error
	calls   []string
}

func (f *fakeGH) run(_ context.Context, name string, args ...string) ([]byte, error) {
	key := "--version"
	if name == "gh" {
		key = args[1]
	}
	f.calls = append(f.calls, key)
	if err := f.fail[key]; err != nil {
		return nil, err
	}
	switch key {
	case "list":
		return []byte(f.list), nil
	case "download":
		dir := args[len(args)-1]
		for n, body := range f.files {
			if err := os.WriteFile(filepath.Join(dir, n), []byte(body), 0o755); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	return []byte(f.version), nil
}

// release is a fake whose only release is a well-formed v<ver> beta.
func release(ver string) *fakeGH {
	asset := AssetName(ver, "windows", "amd64")
	body := "binary " + ver
	sum := sha256.Sum256([]byte(body))
	return &fakeGH{
		list:    fmt.Sprintf(`[{"tagName":"v%s","isPrerelease":true}]`, ver),
		files:   map[string]string{asset: body, "SHA256SUMS": hex.EncodeToString(sum[:]) + "  " + asset + "\n"},
		version: "loomux " + ver + " (beta)\n",
		fail:    map[string]error{},
	}
}
```

- [ ] **Step 2: Failing tests für `gh.go` schreiben**

`internal/selfupdate/gh_test.go`:

```go
package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestLatestTakesTheHighestReleaseOfTheChannel(t *testing.T) {
	f := &fakeGH{list: `[{"tagName":"v2.7.0","isPrerelease":true},{"tagName":"v2.10.0","isPrerelease":true}]`}
	var args []string
	run := func(ctx context.Context, name string, a ...string) ([]byte, error) {
		args = a
		return f.run(ctx, name, a...)
	}
	rel, err := latest(context.Background(), run, "beta")
	if err != nil || rel.Tag != "v2.10.0" {
		t.Fatalf("latest = %v, %v; want v2.10.0", rel, err)
	}
	for _, want := range []string{"list", "--repo", Repo, "--exclude-drafts", "tagName,isPrerelease"} {
		if !slices.Contains(args, want) {
			t.Errorf("gh release list misses %q: %v", want, args)
		}
	}
}

// A repository that has releases and gives none is a fault, not a state:
// before the channel was read, exactly this made every pass say "current".
func TestLatestFailsWhenTheChannelHasNoRelease(t *testing.T) {
	f := &fakeGH{list: `[{"tagName":"v2.7.0","isPrerelease":true}]`}
	_, err := latest(context.Background(), f.run, "")
	if err == nil || err.Error() != "no release in channel stable" {
		t.Fatalf("err = %v", err)
	}
}

func TestLatestRefusesAnswersItCannotRead(t *testing.T) {
	f := &fakeGH{list: `not json`}
	if _, err := latest(context.Background(), f.run, "beta"); err == nil || !strings.Contains(err.Error(), "parse gh release list") {
		t.Fatalf("err = %v", err)
	}
}

func TestLatestNamesAMissingGh(t *testing.T) {
	f := &fakeGH{fail: map[string]error{"list": fmt.Errorf("gh: %w", exec.ErrNotFound)}}
	_, err := latest(context.Background(), f.run, "beta")
	if err == nil || err.Error() != "gh not found; install GitHub CLI and run gh auth login" {
		t.Fatalf("err = %v", err)
	}
}

func TestLatestPassesOtherFailuresOn(t *testing.T) {
	f := &fakeGH{fail: map[string]error{"list": errors.New("gh: HTTP 404: Not Found")}}
	if _, err := latest(context.Background(), f.run, "beta"); err == nil || err.Error() != "gh: HTTP 404: Not Found" {
		t.Fatalf("err = %v", err)
	}
}

func TestACallThatOutlivesItsDeadlineSaysSo(t *testing.T) {
	callTimeout = 10 * time.Millisecond
	t.Cleanup(func() { callTimeout = 2 * time.Minute })
	hang := func(ctx context.Context, _ string, _ ...string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if _, err := latest(context.Background(), hang, "beta"); err == nil || !strings.Contains(err.Error(), "gh timed out") {
		t.Fatalf("err = %v", err)
	}
}

func TestFirstLine(t *testing.T) {
	for in, want := range map[string]string{
		"":                           "",
		"one":                        "one",
		"  first \r\nsecond\n":       "first",
		"\nHTTP 401: Bad credentials": "HTTP 401: Bad credentials",
	} {
		if got := firstLine(in); got != want {
			t.Errorf("firstLine(%q) = %q, want %q", in, got, want)
		}
	}
}
```

- [ ] **Step 3: Tests laufen lassen, sie müssen fehlschlagen**

Run: `go test ./internal/selfupdate/`
Expected: FAIL, `undefined: latest`.

- [ ] **Step 4: `gh.go` und das Gerüst von `Options` schreiben**

`internal/selfupdate/gh.go`:

```go
package selfupdate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Repo is where the releases come from. It is private, so every call goes
// through gh and the login gh already holds; loomux never handles a token.
const Repo = "xidus90/loomux"

// Runner runs one program and answers its standard output. A failure carries
// the first line of standard error, which is where gh says what went wrong.
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// callTimeout bounds every external call. A variable so that a test can reach
// the deadline without waiting two minutes for it.
var callTimeout = 2 * time.Minute

// ExecRunner is the real Runner.
//
//coverage:exempt starts a real process; every caller is tested through a fake Runner, and firstLine, the only logic here, is tested on its own
func ExecRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}
	if line := firstLine(stderr.String()); line != "" {
		return out, fmt.Errorf("%s: %s: %w", filepath.Base(name), line, err)
	}
	return out, fmt.Errorf("%s: %w", filepath.Base(name), err)
}

// firstLine is the first non-empty line of s, trimmed.
func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(line)
}

// call runs one program under the deadline and names the two failures a user
// acts on differently from the rest: no gh at all, and a call that hung.
func call(ctx context.Context, run Runner, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	out, err := run(ctx, name, args...)
	switch {
	case err == nil:
		return out, nil
	case name == "gh" && errors.Is(err, exec.ErrNotFound):
		return nil, errors.New("gh not found; install GitHub CLI and run gh auth login")
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return nil, fmt.Errorf("%s timed out after %s", filepath.Base(name), callTimeout)
	}
	return nil, err
}

// latest is the release this binary's channel should run. Not `gh release
// view` without a tag: that knows only the release marked latest, and a
// repository of pre-releases has none (measured 2026-09-23: "release not
// found").
func latest(ctx context.Context, run Runner, channel string) (Release, error) {
	out, err := call(ctx, run, "gh", "release", "list", "--repo", Repo,
		"--exclude-drafts", "--limit", "30", "--json", "tagName,isPrerelease")
	if err != nil {
		return Release{}, err
	}
	var releases []Release
	if err := json.Unmarshal(out, &releases); err != nil {
		return Release{}, fmt.Errorf("parse gh release list: %w", err)
	}
	rel, ok := pick(releases, channel)
	if !ok {
		return Release{}, fmt.Errorf("no release in channel %s", channelName(channel))
	}
	return rel, nil
}

// channelName is how a build without a channel names its own: stable, as
// cli.versionLine reads it.
func channelName(channel string) string {
	if channel == "" {
		return "stable"
	}
	return channel
}
```

`internal/selfupdate/update.go` (Gerüst, Task 5 ergänzt):

```go
package selfupdate

import "time"

// Options describe the running binary to an update pass. Everything the pass
// would otherwise ask the process for is a field, so a test can be any
// platform and any version.
type Options struct {
	GOOS   string
	GOARCH string
	Run    Runner
	Now    func() time.Time
}
```

Der Timeout-Test: `callTimeout` meldet sich in der Meldung als
`gh timed out after 10ms`; der Test prüft nur `gh timed out`.

- [ ] **Step 5: Failing tests für `fetch.go` schreiben**

`internal/selfupdate/fetch_test.go`:

```go
package selfupdate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var stamp = time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)

func fetchOptions(f *fakeGH) Options {
	return Options{GOOS: "windows", GOARCH: "amd64", Run: f.run, Now: func() time.Time { return stamp }}
}

func TestFetchStagesAVerifiedBinary(t *testing.T) {
	dir := t.TempDir()
	if err := fetch(context.Background(), fetchOptions(release("2.8.0")), "v2.8.0", dir); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, "loomux.new.exe")
	data, err := os.ReadFile(staged)
	if err != nil || string(data) != "binary 2.8.0" {
		t.Fatalf("staged = %q, %v", data, err)
	}
	// The activation is the modification time: OlderThan compares nothing
	// else, so the pass sets it rather than trusting the download's.
	info, _ := os.Stat(staged)
	if !info.ModTime().Equal(stamp) {
		t.Fatalf("mtime = %v, want %v", info.ModTime(), stamp)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("the download directory was left behind: %v", entries)
	}
}

func TestFetchRefuses(t *testing.T) {
	asset := AssetName("2.8.0", "windows", "amd64")
	for _, c := range []struct {
		name  string
		spoil func(f *fakeGH)
		want  string
	}{
		{"no SHA256SUMS", func(f *fakeGH) { delete(f.files, "SHA256SUMS") }, "release v2.8.0 has no SHA256SUMS"},
		{"no line for the asset", func(f *fakeGH) { f.files["SHA256SUMS"] = "abc  other.exe\n" }, "SHA256SUMS of v2.8.0 has no line for " + asset},
		{"no asset", func(f *fakeGH) { delete(f.files, asset) }, "release v2.8.0 has no asset " + asset},
		{"a changed asset", func(f *fakeGH) { f.files[asset] = "tampered" }, "checksum mismatch for " + asset},
		{"another version inside", func(f *fakeGH) { f.version = "loomux 2.7.0 (beta)\n" }, `downloaded binary reports "loomux 2.7.0 (beta)", not loomux 2.8.0`},
		{"a binary that does not run", func(f *fakeGH) { f.fail["--version"] = errors.New("exec format error") }, "run downloaded binary: exec format error"},
		{"a failed download", func(f *fakeGH) { f.fail["download"] = errors.New("gh: no assets match") }, "gh: no assets match"},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			f := release("2.8.0")
			c.spoil(f)
			err := fetch(context.Background(), fetchOptions(f), "v2.8.0", dir)
			if err == nil || err.Error() != c.want {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
			if _, err := os.Stat(filepath.Join(dir, "loomux.new.exe")); !os.IsNotExist(err) {
				t.Fatal("a refused download was staged")
			}
		})
	}
}

func TestFetchAcceptsTheBinaryModeMarker(t *testing.T) {
	f := release("2.8.0")
	asset := AssetName("2.8.0", "windows", "amd64")
	f.files["SHA256SUMS"] = strings.Replace(f.files["SHA256SUMS"], "  "+asset, " *"+asset, 1)
	if err := fetch(context.Background(), fetchOptions(f), "v2.8.0", t.TempDir()); err != nil {
		t.Fatal(err)
	}
}

func TestFetchFailsWithoutItsDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	err := fetch(context.Background(), fetchOptions(release("2.8.0")), "v2.8.0", missing)
	if err == nil || !strings.Contains(err.Error(), "create download directory") {
		t.Fatalf("err = %v", err)
	}
}

func TestFetchFailsWhenItCannotStage(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "loomux.new.exe")
	if err := os.MkdirAll(filepath.Join(blocker, "inside"), 0o700); err != nil {
		t.Fatal(err)
	}
	err := fetch(context.Background(), fetchOptions(release("2.8.0")), "v2.8.0", dir)
	if err == nil || !strings.Contains(err.Error(), "stage") {
		t.Fatalf("err = %v", err)
	}
}

func TestReportsVersion(t *testing.T) {
	for out, want := range map[string]bool{
		"loomux 2.8.0\n":        true,
		"loomux 2.8.0 (beta)\n": true,
		"loomux 2.8.01\n":       false,
		"loomux 2.8\n":          false,
		"":                      false,
	} {
		if got := reportsVersion(out, "2.8.0"); got != want {
			t.Errorf("reportsVersion(%q) = %v, want %v", out, got, want)
		}
	}
}

func TestAssetName(t *testing.T) {
	if got := AssetName("2.7.0", "windows", "amd64"); got != "loomux_2.7.0_windows_amd64.exe" {
		t.Errorf("windows: %s", got)
	}
	if got := AssetName("2.7.0", "linux", "arm64"); got != "loomux_2.7.0_linux_arm64" {
		t.Errorf("linux: %s", got)
	}
}
```

- [ ] **Step 6: Tests laufen lassen, sie müssen fehlschlagen**

Run: `go test ./internal/selfupdate/ -run 'Fetch|Reports|Asset'`
Expected: FAIL, `undefined: fetch`.

- [ ] **Step 7: `fetch.go` schreiben**

`internal/selfupdate/fetch.go`:

```go
package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// sumsName is the checksum asset every release carries (internal/release).
const sumsName = "SHA256SUMS"

// AssetName is the release asset for one platform, named the way
// internal/release names it.
func AssetName(ver, goos, goarch string) string {
	name := fmt.Sprintf("loomux_%s_%s_%s", ver, goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// fetch downloads tag's binary for this platform into a directory of its own
// under dir, holds it against SHA256SUMS and against its own --version, and
// stages it as dir/loomux.new.exe for swap. Nothing is staged unless every
// check passed.
func fetch(ctx context.Context, o Options, tag, dir string) error {
	ver := strings.TrimPrefix(tag, "v")
	asset := AssetName(ver, o.GOOS, o.GOARCH)
	tmp, err := os.MkdirTemp(dir, "update-*")
	if err != nil {
		return fmt.Errorf("create download directory: %w", err)
	}
	defer os.RemoveAll(tmp)
	if _, err := call(ctx, o.Run, "gh", "release", "download", tag, "--repo", Repo,
		"--pattern", asset, "--pattern", sumsName, "--dir", tmp); err != nil {
		return err
	}
	if err := verifySum(tmp, asset, tag); err != nil {
		return err
	}
	path := filepath.Join(tmp, asset)
	out, err := call(ctx, o.Run, path, "--version")
	if err != nil {
		return fmt.Errorf("run downloaded binary: %w", err)
	}
	if !reportsVersion(string(out), ver) {
		return fmt.Errorf("downloaded binary reports %q, not loomux %s", firstLine(string(out)), ver)
	}
	return stage(path, filepath.Join(dir, "loomux.new.exe"), o.Now())
}

// verifySum holds the asset against its line in SHA256SUMS. The line may
// carry sha256sum's binary-mode marker before the name.
func verifySum(dir, asset, tag string) error {
	sums, err := os.ReadFile(filepath.Join(dir, sumsName))
	if err != nil {
		return fmt.Errorf("release %s has no %s", tag, sumsName)
	}
	want := ""
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == asset {
			want = strings.ToLower(fields[0])
		}
	}
	if want == "" {
		return fmt.Errorf("%s of %s has no line for %s", sumsName, tag, asset)
	}
	data, err := os.ReadFile(filepath.Join(dir, asset))
	if err != nil {
		return fmt.Errorf("release %s has no asset %s", tag, asset)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != want {
		return fmt.Errorf("checksum mismatch for %s", asset)
	}
	return nil
}

// reportsVersion accepts "loomux <ver>" alone or followed by the channel.
func reportsVersion(out, ver string) bool {
	line := firstLine(out)
	return line == "loomux "+ver || strings.HasPrefix(line, "loomux "+ver+" ")
}

// stage stamps the binary with now and moves it to target. The stamp is the
// activation: a bridge replaces serve only for a newer modification time
// (serve.State.OlderThan), and a download must not depend on what time gh
// happens to leave on the file.
//
//coverage:exempt the Chtimes arm needs the OS to refuse new times on a file this process has just written and run; the Rename arm is covered
func stage(path, target string, now time.Time) error {
	if err := os.Chtimes(path, now, now); err != nil {
		return fmt.Errorf("stamp %s: %w", path, err)
	}
	if err := os.Rename(path, target); err != nil {
		return fmt.Errorf("stage %s: %w", target, err)
	}
	return nil
}
```

- [ ] **Step 8: Tests laufen lassen**

Run: `go test ./internal/selfupdate/ -cover`
Expected: PASS. Coverage ohne `ExecRunner` und `stage` 100 %; die Gate-Prüfung
`go run ./cmd/loomux check coverage` ist grün.

- [ ] **Step 9: Commit**

```bash
git add internal/selfupdate
git commit -m "feat(selfupdate): download a release through gh and verify it"
```

---

### Task 5: Ein Update-Durchlauf — `selfupdate.Run`

**Files:**
- Modify: `internal/selfupdate/update.go`
- Test: `internal/selfupdate/update_test.go`

**Interfaces:**
- Consumes: `latest`, `fetch`, `Newer` (Tasks 2, 4), `WriteStatus`,
  `IsCanonical`, `Canonical` (Task 3), `swap.Swap` (Task 1),
  `lock.TryAcquire(path string) (*lock.Handle, bool, error)`.
- Produces:
  - `type Options struct { StateDir, Executable, Version, Channel, GOOS, GOARCH string; Run Runner; Now func() time.Time }`
  - `type Result struct { Outcome Outcome; Version string; Err error; StatusErr error }`
  - `const DevVersion = "0.0.0-dev"`
  - `func Run(ctx context.Context, o Options) Result`

- [ ] **Step 1: Failing tests schreiben**

`internal/selfupdate/update_test.go`:

```go
package selfupdate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/lock"
)

// installed is a state directory whose canonical binary is "old" at v2.7.0
// beta, with the options of a process running it.
func installed(t *testing.T, f *fakeGH) Options {
	t.Helper()
	dir := t.TempDir()
	exe := Canonical(dir)
	if err := os.MkdirAll(filepath.Dir(exe), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	return Options{
		StateDir: dir, Executable: exe, Version: "2.7.0", Channel: "beta",
		GOOS: "windows", GOARCH: "amd64", Run: f.run,
		Now: func() time.Time { return stamp },
	}
}

func canonicalBody(t *testing.T, o Options) string {
	t.Helper()
	data, err := os.ReadFile(Canonical(o.StateDir))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func recorded(t *testing.T, o Options) Status {
	t.Helper()
	st, err := ReadStatus(o.StateDir)
	if err != nil || st == nil {
		t.Fatalf("update.json: %v, %v", st, err)
	}
	return *st
}

func TestRunInstallsANewerRelease(t *testing.T) {
	o := installed(t, release("2.8.0"))
	res := Run(context.Background(), o)
	if res.Outcome != Updated || res.Version != "2.8.0" || res.Err != nil {
		t.Fatalf("Run = %+v", res)
	}
	if got := canonicalBody(t, o); got != "binary 2.8.0" {
		t.Fatalf("canonical binary = %q", got)
	}
	old, _ := os.ReadFile(filepath.Join(filepath.Dir(Canonical(o.StateDir)), "loomux.old.exe"))
	if string(old) != "old" {
		t.Fatalf("the previous binary was not kept aside: %q", old)
	}
	st := recorded(t, o)
	if st.Result != Updated || st.Version != "2.8.0" || st.Running != "2.7.0" || st.Executable != o.Executable || !st.CheckedAt.Equal(stamp) {
		t.Fatalf("update.json = %+v", st)
	}
}

func TestRunLeavesACurrentBinaryAlone(t *testing.T) {
	f := release("2.7.0")
	o := installed(t, f)
	res := Run(context.Background(), o)
	if res.Outcome != Current || res.Version != "2.7.0" {
		t.Fatalf("Run = %+v", res)
	}
	if strings.Join(f.calls, ",") != "list" {
		t.Fatalf("a current binary downloaded something: %v", f.calls)
	}
	if recorded(t, o).Result != Current {
		t.Fatal("current was not recorded")
	}
}

func TestRunSkips(t *testing.T) {
	for _, c := range []struct {
		name  string
		bend  func(o *Options)
		want  string
	}{
		{"off Windows", func(o *Options) { o.GOOS = "linux" }, "self-update runs on Windows only"},
		{"a development build", func(o *Options) { o.Version = DevVersion }, "development build 0.0.0-dev is never replaced"},
		{"a binary elsewhere", func(o *Options) { o.Executable = filepath.Join(o.StateDir, "elsewhere.exe") }, "running from "},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := release("2.8.0")
			o := installed(t, f)
			c.bend(&o)
			res := Run(context.Background(), o)
			if res.Outcome != Skipped || res.Err == nil || !strings.Contains(res.Err.Error(), c.want) {
				t.Fatalf("Run = %+v", res)
			}
			if len(f.calls) != 0 {
				t.Fatalf("a skipped pass called out: %v", f.calls)
			}
			if st := recorded(t, o); st.Result != Skipped || !strings.Contains(st.Error, c.want) {
				t.Fatalf("update.json = %+v", st)
			}
		})
	}
}

func TestRunFailsAndKeepsTheBinary(t *testing.T) {
	for _, c := range []struct {
		name  string
		spoil func(f *fakeGH)
		want  string
	}{
		{"no gh", func(f *fakeGH) { f.fail["list"] = fmt.Errorf("gh: %w", os.ErrNotExist) }, "gh: "},
		{"a changed asset", func(f *fakeGH) { f.files[AssetName("2.8.0", "windows", "amd64")] = "tampered" }, "checksum mismatch"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := release("2.8.0")
			c.spoil(f)
			o := installed(t, f)
			res := Run(context.Background(), o)
			if res.Outcome != Failed || res.Err == nil || !strings.Contains(res.Err.Error(), c.want) {
				t.Fatalf("Run = %+v", res)
			}
			if got := canonicalBody(t, o); got != "old" {
				t.Fatalf("a failed pass touched the binary: %q", got)
			}
			if st := recorded(t, o); st.Result != Failed || st.Error != res.Err.Error() {
				t.Fatalf("update.json = %+v", st)
			}
		})
	}
}

func TestRunFailsWhenEverySlotIsHeld(t *testing.T) {
	o := installed(t, release("2.8.0"))
	bin := filepath.Dir(Canonical(o.StateDir))
	for i := range 16 {
		name := "loomux.old.exe"
		if i > 0 {
			name = fmt.Sprintf("loomux.old.%d.exe", i)
		}
		if err := os.MkdirAll(filepath.Join(bin, name, "held"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	res := Run(context.Background(), o)
	if res.Outcome != Failed || !strings.Contains(res.Err.Error(), "slots") {
		t.Fatalf("Run = %+v", res)
	}
	if got := canonicalBody(t, o); got != "old" {
		t.Fatalf("canonical binary = %q", got)
	}
}

func TestRunStepsAsideForAPassInProgress(t *testing.T) {
	f := release("2.8.0")
	o := installed(t, f)
	handle, held, err := lock.TryAcquire(filepath.Join(o.StateDir, "update.lock"))
	if err != nil || !held {
		t.Fatalf("take the lock: %v, %v", held, err)
	}
	defer handle.Release()
	res := Run(context.Background(), o)
	if res.Outcome != Busy || res.Err == nil || res.Err.Error() != "update in progress" {
		t.Fatalf("Run = %+v", res)
	}
	if st, _ := ReadStatus(o.StateDir); st != nil {
		t.Fatalf("a busy pass wrote update.json: %+v", st)
	}
	if len(f.calls) != 0 {
		t.Fatalf("a busy pass called out: %v", f.calls)
	}
}

func TestRunFailsWhenTheLockCannotBeOpened(t *testing.T) {
	o := installed(t, release("2.8.0"))
	if err := os.MkdirAll(filepath.Join(o.StateDir, "update.lock", "x"), 0o700); err != nil {
		t.Fatal(err)
	}
	if res := Run(context.Background(), o); res.Outcome != Failed || res.Err == nil {
		t.Fatalf("Run = %+v", res)
	}
}

func TestRunReportsAStatusItCouldNotWrite(t *testing.T) {
	o := installed(t, release("2.7.0"))
	if err := os.MkdirAll(filepath.Join(StatusPath(o.StateDir), "x"), 0o700); err != nil {
		t.Fatal(err)
	}
	res := Run(context.Background(), o)
	// The pass itself came to its outcome; only its record is missing.
	if res.Outcome != Current || res.Err != nil || res.StatusErr == nil {
		t.Fatalf("Run = %+v", res)
	}
}
```

- [ ] **Step 2: Tests laufen lassen, sie müssen fehlschlagen**

Run: `go test ./internal/selfupdate/ -run Run`
Expected: FAIL, `unknown field StateDir in struct literal`.

- [ ] **Step 3: `update.go` vollständig schreiben**

`internal/selfupdate/update.go` ersetzen durch:

```go
package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/lock"
	"github.com/xidus90/loomux/internal/swap"
)

// DevVersion is what a plain go build reports (cli.Version's default). Such a
// binary is someone's work in progress and is never replaced.
const DevVersion = "0.0.0-dev"

// Options describe the running binary to an update pass. Everything the pass
// would otherwise ask the process for is a field, so a test can be any
// platform and any version; the caller passes cli.Version and cli.Channel,
// because this package may not import cli.
type Options struct {
	StateDir   string
	Executable string
	Version    string
	Channel    string
	GOOS       string
	GOARCH     string
	Run        Runner
	Now        func() time.Time
}

// Result is what a pass came to. Version is the release installed, or the
// running one when it is current. StatusErr is a failure to write
// update.json, apart from the pass's own outcome: a binary swapped in stays
// swapped in even when the record of it could not be written.
type Result struct {
	Outcome   Outcome
	Version   string
	Err       error
	StatusErr error
}

// Run is one update pass, recorded in update.json unless another pass held
// the lock.
func Run(ctx context.Context, o Options) Result {
	res := run(ctx, o)
	if res.Outcome == Busy {
		return res
	}
	st := Status{
		CheckedAt:  o.Now().UTC(),
		Executable: o.Executable,
		Running:    o.Version,
		Result:     res.Outcome,
		Version:    res.Version,
	}
	if res.Err != nil {
		st.Error = res.Err.Error()
	}
	res.StatusErr = WriteStatus(o.StateDir, st)
	return res
}

// run decides and acts. A failure anywhere leaves loomux.exe where it was.
func run(ctx context.Context, o Options) Result {
	if o.GOOS != "windows" {
		return Result{Outcome: Skipped, Err: errors.New("self-update runs on Windows only")}
	}
	if o.Version == DevVersion {
		return Result{Outcome: Skipped, Err: fmt.Errorf("development build %s is never replaced", DevVersion)}
	}
	canonical := Canonical(o.StateDir)
	if !IsCanonical(o.Executable, o.StateDir) {
		return Result{Outcome: Skipped, Err: fmt.Errorf("running from %s, not from %s", o.Executable, canonical)}
	}
	handle, held, err := lock.TryAcquire(filepath.Join(o.StateDir, "update.lock"))
	if err != nil {
		return Result{Outcome: Failed, Err: err}
	}
	if !held {
		return Result{Outcome: Busy, Err: errors.New("update in progress")}
	}
	defer handle.Release()

	rel, err := latest(ctx, o.Run, o.Channel)
	if err != nil {
		return Result{Outcome: Failed, Err: err}
	}
	if !Newer(rel.Tag, o.Version) {
		return Result{Outcome: Current, Version: o.Version}
	}
	ver := strings.TrimPrefix(rel.Tag, "v")
	dir := filepath.Dir(canonical)
	if err := fetch(ctx, o, rel.Tag, dir); err != nil {
		return Result{Outcome: Failed, Version: ver, Err: err}
	}
	if err := swap.Swap(dir); err != nil {
		return Result{Outcome: Failed, Version: ver, Err: err}
	}
	return Result{Outcome: Updated, Version: ver}
}
```

- [ ] **Step 4: Tests laufen lassen**

Run: `go test ./internal/selfupdate/ -cover`
Expected: PASS. Schlägt `TestRunFailsWhenTheLockCannotBeOpened` unter Linux
fehl, weil `TryAcquire` ein Verzeichnis öffnen kann, das Verhalten von
`lock.TryAcquire` nachlesen und den Test an einen Fall anpassen, den es
wirklich verweigert — nicht den Zweig per `coverage:exempt` ausnehmen.

- [ ] **Step 5: Commit**

```bash
git add internal/selfupdate
git commit -m "feat(selfupdate): run one update pass under a lock"
```

---

### Task 6: Die Update-Schleife in `serve`

**Files:**
- Create: `internal/serve/update.go`
- Modify: `internal/serve/serve.go` (`Options`, `Run` nach `WriteState`)
- Test: `internal/serve/update_test.go`, `internal/serve/spawn_internal_test.go`

**Interfaces:**
- Produces:
  - `func UpdateLoop(ctx context.Context, update func(context.Context), first, interval time.Duration, after func(time.Duration) <-chan time.Time)`
  - `serve.Options.Update func(context.Context)` — nil: keine Schleife
  - `serve.Options.UpdateAfter func(time.Duration) <-chan time.Time` — nil: `time.After`
  - `const UpdateFirst = time.Minute`, `const UpdateInterval = 24 * time.Hour`

- [ ] **Step 1: Failing tests schreiben**

`internal/serve/update_test.go`:

```go
package serve_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/serve"
)

func TestUpdateLoopWaitsFirstThenEveryInterval(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var waits []time.Duration
	after := func(d time.Duration) <-chan time.Time {
		waits = append(waits, d)
		c := make(chan time.Time, 1)
		c <- time.Now()
		return c
	}
	runs := 0
	update := func(context.Context) {
		runs++
		if runs == 3 {
			cancel()
		}
	}
	serve.UpdateLoop(ctx, update, time.Minute, 24*time.Hour, after)
	if runs != 3 {
		t.Fatalf("runs = %d, want 3", runs)
	}
	want := []time.Duration{time.Minute, 24 * time.Hour, 24 * time.Hour}
	for i, w := range want {
		if waits[i] != w {
			t.Fatalf("waits = %v, want %v first", waits, want)
		}
	}
}

func TestUpdateLoopEndsBeforeTheFirstPassWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	never := func(time.Duration) <-chan time.Time { return nil }
	serve.UpdateLoop(ctx, func(context.Context) { t.Fatal("a cancelled loop ran a pass") }, time.Minute, time.Hour, never)
}

// Run starts the loop when it is given a pass, and ends it with the service.
func TestRunStartsTheUpdateLoop(t *testing.T) {
	dir := t.TempDir()
	ran := make(chan struct{}, 1)
	var calls atomic.Int32
	after := func(time.Duration) <-chan time.Time {
		if calls.Add(1) > 1 {
			return nil
		}
		c := make(chan time.Time, 1)
		c <- time.Now()
		return c
	}
	startWith(t, dir, serve.Options{
		StateDir: dir, RegistryDir: dir, LegacyDir: dir,
		Update:      func(context.Context) { ran <- struct{}{} },
		UpdateAfter: after,
	})
	select {
	case <-ran:
	case <-time.After(5 * time.Second):
		t.Fatal("serve never ran an update pass")
	}
}
```

In `internal/serve/spawn_internal_test.go` ergänzen (Import `slices`):

```go
// gh reads its login from the user's configuration, and serve runs it. The
// child keeps every variable it does not replace, these included.
func TestChildEnvKeepsWhatGhNeeds(t *testing.T) {
	base := []string{`PATH=C:\bin`, `APPDATA=C:\a`, `USERPROFILE=C:\u`, `GH_CONFIG_DIR=C:\gh`}
	got := childEnv(base, "/state", false)
	for _, want := range base {
		if !slices.Contains(got, want) {
			t.Errorf("childEnv dropped %s: %v", want, got)
		}
	}
}
```

- [ ] **Step 2: Tests laufen lassen, sie müssen fehlschlagen**

Run: `go test ./internal/serve/ -run 'UpdateLoop|StartsTheUpdateLoop|ChildEnvKeeps'`
Expected: FAIL, `undefined: serve.UpdateLoop`. `TestChildEnvKeepsWhatGhNeeds`
besteht schon jetzt — er hält das Verhalten fest, er treibt keinen Code.

- [ ] **Step 3: Implementieren**

`internal/serve/update.go`:

```go
package serve

import (
	"context"
	"time"
)

// The pace of the self-update: a minute after start, so the pass does not
// compete with the first requests, and daily after that.
const (
	UpdateFirst    = time.Minute
	UpdateInterval = 24 * time.Hour
)

// UpdateLoop runs update once first has passed and then every interval, until
// ctx ends. It is not part of the upkeep: the upkeep holds the first answer
// until its pass is done, and an update must hold none.
func UpdateLoop(ctx context.Context, update func(context.Context), first, interval time.Duration, after func(time.Duration) <-chan time.Time) {
	wait := first
	for {
		select {
		case <-ctx.Done():
			return
		case <-after(wait):
		}
		update(ctx)
		wait = interval
	}
}
```

In `internal/serve/serve.go`, `Options`, nach dem Feld `Answer`:

```go
	// Update is one self-update pass; nil runs none. The command line fills
	// it, so serve knows nothing of releases.
	Update func(context.Context)
	// UpdateAfter is the clock of the update loop, time.After when nil.
	UpdateAfter func(time.Duration) <-chan time.Time
```

In `Run`, direkt nach dem erfolgreichen `WriteState`-Block und vor dem
abschließenden `select`:

```go
	if opts.Update != nil {
		after := opts.UpdateAfter
		if after == nil {
			after = time.After
		}
		loop, cancel := context.WithCancel(ctx)
		defer cancel()
		go UpdateLoop(loop, opts.Update, UpdateFirst, UpdateInterval, after)
	}
```

`time` ist in `serve.go` womöglich noch nicht importiert; ergänzen.

- [ ] **Step 4: Tests laufen lassen**

Run: `go test ./internal/serve/ -cover`
Expected: PASS. Der Zweig `after = time.After` wird von keinem Test genommen,
wenn jeder Test `UpdateAfter` setzt; dann einen Test ergänzen, der `Update`
ohne `UpdateAfter` setzt und nur prüft, dass `serve` startet und stoppt (die
erste Minute läuft dabei nicht ab).

- [ ] **Step 5: Commit**

```bash
git add internal/serve
git commit -m "feat(serve): run a self-update pass daily"
```

---

### Task 7: `loomux self-update` und die Verdrahtung in `serve`

**Files:**
- Create: `internal/cli/selfupdate.go`
- Modify: `internal/cli/commands.go` (Zeile `"self-update"`),
  `internal/cli/serve.go` (`serveForeground`)
- Test: `internal/cli/selfupdate_test.go`, `internal/cli/imports_test.go`

**Interfaces:**
- Consumes: `selfupdate.Run`, `selfupdate.Options`, `selfupdate.Result`,
  `selfupdate.ExecRunner`, `serve.Options.Update` (Task 6).
- Produces: `var selfUpdateRun = selfupdate.Run`,
  `func selfUpdateOptions() selfupdate.Options`,
  `func selfUpdateCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int`.

- [ ] **Step 1: Failing tests schreiben**

`internal/cli/selfupdate_test.go` (package `cli`, damit die Seams erreichbar
sind; prüfe mit `head -1 internal/cli/dev_test.go`, welches Paket die
Nachbartests nutzen, und nimm dasselbe — nutzen sie `cli_test`, lege die Seams
über eine `export_test.go` offen, wie es das Paket schon tut):

```go
package cli

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/selfupdate"
)

func fakeSelfUpdate(t *testing.T, res selfupdate.Result) *int {
	t.Helper()
	calls := 0
	selfUpdateRun = func(context.Context, selfupdate.Options) selfupdate.Result {
		calls++
		return res
	}
	t.Cleanup(func() { selfUpdateRun = selfupdate.Run })
	return &calls
}

func TestSelfUpdateCommand(t *testing.T) {
	for _, c := range []struct {
		name       string
		res        selfupdate.Result
		code       int
		out, errs  string
	}{
		{"current", selfupdate.Result{Outcome: selfupdate.Current, Version: "2.7.0"}, 0, "already current (v2.7.0)\n", ""},
		{"updated", selfupdate.Result{Outcome: selfupdate.Updated, Version: "2.8.0"}, 0, "updated to v2.8.0; serve switches on the next bridge\n", ""},
		{"skipped", selfupdate.Result{Outcome: selfupdate.Skipped, Err: errors.New("development build 0.0.0-dev is never replaced")}, 2, "", "loomux self-update: skipped: development build 0.0.0-dev is never replaced\n"},
		{"failed", selfupdate.Result{Outcome: selfupdate.Failed, Err: errors.New("checksum mismatch for x")}, 1, "", "loomux self-update: checksum mismatch for x\n"},
		{"busy", selfupdate.Result{Outcome: selfupdate.Busy, Err: errors.New("update in progress")}, 1, "", "loomux self-update: update in progress\n"},
		{"status unwritten", selfupdate.Result{Outcome: selfupdate.Current, Version: "2.7.0", StatusErr: errors.New("disk full")}, 0, "already current (v2.7.0)\n", "loomux self-update: record update.json: disk full\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			fakeSelfUpdate(t, c.res)
			var out, errs bytes.Buffer
			code := selfUpdateCommand(nil, nil, &out, &errs)
			if code != c.code || out.String() != c.out || errs.String() != c.errs {
				t.Fatalf("code %d, out %q, err %q", code, out.String(), errs.String())
			}
		})
	}
}

func TestSelfUpdateTakesNoArguments(t *testing.T) {
	calls := fakeSelfUpdate(t, selfupdate.Result{})
	var errs bytes.Buffer
	if code := selfUpdateCommand([]string{"--to", "x"}, nil, &bytes.Buffer{}, &errs); code != 2 {
		t.Fatalf("code = %d", code)
	}
	if *calls != 0 || !strings.Contains(errs.String(), "usage: loomux self-update") {
		t.Fatalf("calls %d, err %q", *calls, errs.String())
	}
}

func TestSelfUpdateOptionsDescribeThisProcess(t *testing.T) {
	t.Setenv(config.StateDirEnv, t.TempDir())
	o := selfUpdateOptions()
	if o.StateDir != config.StateDir() || o.Version != Version || o.Channel != Channel ||
		o.GOOS != runtime.GOOS || o.GOARCH != runtime.GOARCH || o.Executable == "" || o.Run == nil || o.Now == nil {
		t.Fatalf("options = %+v", o)
	}
}
```

Für `serveForeground`: In `internal/cli/serve_test.go` gibt es Tests, die
`serveRun` ersetzen (`grep -n "serveRun =" internal/cli/*_test.go`). Nach deren
Muster einen Test ergänzen:

```go
func TestServeForegroundHandsTheServiceAnUpdatePass(t *testing.T) {
	var got serve.Options
	serveRun = func(_ context.Context, opts serve.Options) error { got = opts; return nil }
	t.Cleanup(func() { serveRun = serve.Run })
	calls := fakeSelfUpdate(t, selfupdate.Result{Outcome: selfupdate.Current})
	if code := serveForeground(&bytes.Buffer{}); code != 0 {
		t.Fatalf("code = %d", code)
	}
	if got.Update == nil {
		t.Fatal("serve got no update pass")
	}
	got.Update(context.Background())
	if *calls != 1 {
		t.Fatalf("the pass ran selfupdate %d times", *calls)
	}
}
```

(Ersetzen die vorhandenen Tests auch `serveNotify`, dasselbe hier tun.)

In `internal/cli/imports_test.go` am Ende ergänzen:

```go
// selfupdate sits below its three callers; an import back into any of them
// would be a cycle or would pull the MCP stack onto the session-start path.
func TestSelfupdateStaysBelowItsCallers(t *testing.T) {
	deps, err := dependencies("github.com/xidus90/loomux/internal/selfupdate")
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range []string{
		"github.com/xidus90/loomux/internal/serve",
		"github.com/xidus90/loomux/internal/hooks",
		"github.com/xidus90/loomux/internal/cli",
		"github.com/xidus90/loomux/internal/bridge",
	} {
		if deps[pkg] {
			t.Errorf("internal/selfupdate depends on %s", pkg)
		}
	}
}
```

- [ ] **Step 2: Tests laufen lassen, sie müssen fehlschlagen**

Run: `go test ./internal/cli/ -run 'SelfUpdate|Selfupdate|UpdatePass'`
Expected: FAIL, `undefined: selfUpdateCommand`.

- [ ] **Step 3: Implementieren**

`internal/cli/selfupdate.go`:

```go
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/selfupdate"
)

// selfUpdateRun is the seam the command and serve's pass are tested through:
// the real one reaches GitHub.
var selfUpdateRun = selfupdate.Run

// selfUpdateCommand is `loomux self-update`: one pass by hand, the same one
// serve runs daily.
func selfUpdateCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: loomux self-update")
		return 2
	}
	res := selfUpdateRun(context.Background(), selfUpdateOptions())
	if res.StatusErr != nil {
		fmt.Fprintf(stderr, "loomux self-update: record update.json: %v\n", res.StatusErr)
	}
	switch res.Outcome {
	case selfupdate.Current:
		fmt.Fprintf(stdout, "already current (v%s)\n", res.Version)
		return 0
	case selfupdate.Updated:
		fmt.Fprintf(stdout, "updated to v%s; serve switches on the next bridge\n", res.Version)
		return 0
	case selfupdate.Skipped:
		fmt.Fprintf(stderr, "loomux self-update: skipped: %v\n", res.Err)
		return 2
	}
	fmt.Fprintf(stderr, "loomux self-update: %v\n", res.Err)
	return 1
}

// selfUpdateOptions describe this process to the updater. An executable the
// OS cannot name stays empty, and the pass then skips as not canonical.
func selfUpdateOptions() selfupdate.Options {
	exe, _ := os.Executable()
	return selfupdate.Options{
		StateDir:   config.StateDir(),
		Executable: exe,
		Version:    Version,
		Channel:    Channel,
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		Run:        selfupdate.ExecRunner,
		Now:        time.Now,
	}
}
```

`internal/cli/commands.go`, alphabetisch nach `"serve"`:

```go
	"self-update": selfUpdateCommand,
```

`internal/cli/serve.go`, in `serveForeground` im `serve.Options`-Literal nach
`BrokeAway`:

```go
		Update:      func(ctx context.Context) { selfUpdateRun(ctx, selfUpdateOptions()) },
```

- [ ] **Step 4: Tests laufen lassen**

Run: `go test ./internal/cli/ -cover` und `gofmt -l internal/cli`
Expected: PASS, keine Ausgabe von `gofmt`. `go run ./cmd/loomux self-update`
aus dem Checkout meldet `skipped: development build 0.0.0-dev is never
replaced` mit Exit 2.

- [ ] **Step 5: Commit**

```bash
git add internal/cli
git commit -m "feat(cli): update the machine-wide binary with loomux self-update"
```

---

### Task 8: Warnungen im Sitzungsstart

**Files:**
- Modify: `internal/hooks/hook_session_start.go`
- Test: `internal/hooks/hook_session_start_test.go`

**Interfaces:**
- Consumes: `selfupdate.ReadStatus`, `selfupdate.IsCanonical`,
  `selfupdate.Canonical`, `selfupdate.WriteStatus`, `selfupdate.Failed`,
  `config.StateDir()`, `config.StateDirEnv`.
- Produces: `func updateWarnings(stateDir string) []string`.

- [ ] **Step 1: Failing tests schreiben**

In `internal/hooks/hook_session_start_test.go` ergänzen (Imports
`github.com/xidus90/loomux/internal/selfupdate` und, falls nicht vorhanden,
`os`, `path/filepath`, `strings`, `time`):

```go
func canonicalIn(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	exe := selfupdate.Canonical(dir)
	if err := os.MkdirAll(filepath.Dir(exe), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestUpdateWarnings(t *testing.T) {
	at := time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		name   string
		status func(dir string) *selfupdate.Status
		want   []string
	}{
		{"no pass yet", func(string) *selfupdate.Status { return nil }, nil},
		{"all well", func(dir string) *selfupdate.Status {
			return &selfupdate.Status{Executable: selfupdate.Canonical(dir), Result: selfupdate.Current}
		}, nil},
		{"serve from a checkout", func(string) *selfupdate.Status {
			return &selfupdate.Status{Executable: `C:\repo\bin\loomux.exe`, Result: selfupdate.Skipped}
		}, []string{`loomux serve runs from C:\repo\bin\loomux.exe, not from `}},
		{"a failed pass", func(dir string) *selfupdate.Status {
			return &selfupdate.Status{Executable: selfupdate.Canonical(dir), Result: selfupdate.Failed, CheckedAt: at, Error: "gh not found"}
		}, []string{"loomux self-update failed at 2026-09-24T08:00:00Z: gh not found"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := canonicalIn(t)
			if st := c.status(dir); st != nil {
				if err := selfupdate.WriteStatus(dir, *st); err != nil {
					t.Fatal(err)
				}
			}
			got := updateWarnings(dir)
			if len(got) != len(c.want) {
				t.Fatalf("updateWarnings = %v, want %v", got, c.want)
			}
			for i := range c.want {
				if !strings.HasPrefix(got[i], c.want[i]) {
					t.Fatalf("line %d = %q, want prefix %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestUpdateWarningsNamesAnUnreadableStatus(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(selfupdate.StatusPath(dir), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := updateWarnings(dir)
	if len(got) != 1 || !strings.HasPrefix(got[0], "loomux cannot read the self-update status: ") {
		t.Fatalf("updateWarnings = %v", got)
	}
}
```

- [ ] **Step 2: Tests laufen lassen, sie müssen fehlschlagen**

Run: `go test ./internal/hooks/ -run UpdateWarnings`
Expected: FAIL, `undefined: updateWarnings`.

- [ ] **Step 3: Implementieren**

In `internal/hooks/hook_session_start.go` die Imports um
`github.com/xidus90/loomux/internal/config` (falls fehlend) und
`github.com/xidus90/loomux/internal/selfupdate` ergänzen, in `SessionStart`

```go
	lines := staleBinary(root)
```

ersetzen durch

```go
	lines := append(staleBinary(root), updateWarnings(config.StateDir())...)
```

und nach `staleBinary` einfügen:

```go
// updateWarnings reads what serve's last self-update pass left in
// update.json. Only the file: session start asks neither the network nor the
// service, and hooks may not import serve. Without the file it says nothing,
// because a machine without a running service is not at fault.
//
// There is deliberately no warning by age. serve checks a minute after it
// starts, so after two days off the first session would read one while
// nothing is wrong.
func updateWarnings(stateDir string) []string {
	st, err := selfupdate.ReadStatus(stateDir)
	if err != nil {
		return []string{fmt.Sprintf("loomux cannot read the self-update status: %v", err)}
	}
	if st == nil {
		return nil
	}
	var lines []string
	if canonical := selfupdate.Canonical(stateDir); !selfupdate.IsCanonical(st.Executable, stateDir) {
		lines = append(lines, fmt.Sprintf(
			"loomux serve runs from %s, not from %s; point the MCP entry at %s",
			st.Executable, canonical, canonical))
	}
	if st.Result == selfupdate.Failed {
		lines = append(lines, fmt.Sprintf("loomux self-update failed at %s: %s",
			st.CheckedAt.UTC().Format(time.RFC3339), st.Error))
	}
	return lines
}
```

- [ ] **Step 4: Tests laufen lassen**

Run: `go test ./internal/hooks/ ./internal/cli/ -cover`
Expected: PASS, auch `TestHooks…`-Importtests in `internal/cli/imports_test.go`.
Die übrigen SessionStart-Tests laufen mit dem echten `config.StateDir()`;
setzt keiner `LOOMUX_STATE_DIR`, liest der Test die `update.json` dieser
Maschine. Dann in der gemeinsamen Hilfsfunktion der SessionStart-Tests
`t.Setenv(config.StateDirEnv, t.TempDir())` ergänzen.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks
git commit -m "feat(hooks): warn at session start when serve runs elsewhere or its update failed"
```

---

### Task 9: Doku nachziehen und das Tor laufen lassen

**Files:**
- Modify: `README.md`, `README.de.md`, `docs/en/cli-reference.md`,
  `docs/de/cli-reference.md`, `docs/en/migration.md`, `docs/de/migration.md`

- [ ] **Step 1: README.md**

Im Block „Active Commands“ nach der Zeile `loomux serve stop …`:

```
loomux self-update                  # replace the machine-wide binary with the newest release of its channel; serve does this daily
```

Die Zeile `loomux hook session-start` ändern zu:

```
loomux hook session-start           # record the session's base commit; warn about a stale binary, a serve outside the install location and a failed self-update
```

Im Abschnitt „Releases“ nach dem `sha256sum`-Block einfügen:

````markdown
### Installing the machine-wide binary

The MCP bridge and `loomux serve` run from one binary per machine,
`%LOCALAPPDATA%\loomux\bin\loomux.exe`, taken from a release and never from a
checkout. Install it once with the [GitHub CLI](https://cli.github.com/),
logged in with `gh auth login`:

```powershell
$bin = "$env:LOCALAPPDATA\loomux\bin"
New-Item -ItemType Directory -Force $bin | Out-Null
gh release download <tag> --repo xidus90/loomux --pattern 'loomux_*_windows_amd64.exe' --pattern SHA256SUMS --dir $bin
```

Check the file against `SHA256SUMS`, rename it to `loomux.exe`, delete
`SHA256SUMS`, and point the MCP entry at it:

```powershell
claude mcp add loomux -s user -- "$env:LOCALAPPDATA\loomux\bin\loomux.exe" mcp --channel local
```

From then on `serve` keeps it current: a minute after it starts and daily
after that it takes the highest release of the binary's own channel through
`gh`, checks it against `SHA256SUMS` and its `--version`, and swaps the file.
The next bridge replaces the running service. `loomux self-update` does the
same by hand. What the last pass found is in `update.json` beside the binary's
directory; session start warns when `serve` runs from anywhere else or the
pass failed. Windows only for now.
````

- [ ] **Step 2: README.de.md**

Dieselben drei Stellen auf Deutsch. Befehlszeile:

```
loomux self-update                  # ersetzt das maschinenweite Binary durch das neueste Release seines Kanals; serve tut das täglich
```

`session-start`-Zeile:

```
loomux hook session-start           # merkt sich den Basis-Commit der Sitzung; warnt bei veraltetem Binary, bei einem serve außerhalb des Installationsorts und bei gescheitertem Self-Update
```

Abschnitt (unter „Releases“ bzw. der deutschen Entsprechung, nach dem
`sha256sum`-Block):

````markdown
### Das maschinenweite Binary installieren

MCP-Brücke und `loomux serve` laufen aus einem Binary je Rechner,
`%LOCALAPPDATA%\loomux\bin\loomux.exe`, das aus einem Release stammt und nie
aus einem Checkout. Einmal installieren mit der
[GitHub CLI](https://cli.github.com/), angemeldet per `gh auth login`:

```powershell
$bin = "$env:LOCALAPPDATA\loomux\bin"
New-Item -ItemType Directory -Force $bin | Out-Null
gh release download <tag> --repo xidus90/loomux --pattern 'loomux_*_windows_amd64.exe' --pattern SHA256SUMS --dir $bin
```

Die Datei gegen `SHA256SUMS` prüfen, in `loomux.exe` umbenennen,
`SHA256SUMS` löschen und den MCP-Eintrag darauf zeigen lassen:

```powershell
claude mcp add loomux -s user -- "$env:LOCALAPPDATA\loomux\bin\loomux.exe" mcp --channel local
```

Danach hält `serve` es aktuell: eine Minute nach dem Start und danach täglich
holt es über `gh` das höchste Release des eigenen Kanals, prüft es gegen
`SHA256SUMS` und seine `--version` und tauscht die Datei. Die nächste Brücke
ersetzt den laufenden Dienst. `loomux self-update` tut dasselbe von Hand. Was
der letzte Durchlauf fand, steht in `update.json` im Zustandsverzeichnis; der
Sitzungsstart warnt, wenn `serve` woanders läuft oder der Durchlauf
gescheitert ist. Vorerst nur unter Windows.
````

- [ ] **Step 3: cli-reference (en)**

In `docs/en/cli-reference.md`, Abschnitt 8, nach `### \`loomux serve stop [--force]\`` und seinem Text:

````markdown
### `loomux self-update`

One self-update pass by hand; `serve` runs the same pass a minute after it
starts and every 24 hours after that. It acts only on the machine-wide
binary, `<state dir>/bin/loomux.exe`, when that is the running binary and
carries a release version.

1. Lists the releases through `gh release list` and takes the highest version
   of the running binary's channel (`beta` takes pre-releases, `stable` does
   not).
2. Downloads the Windows asset and `SHA256SUMS` through `gh release
   download`, checks the checksum and the new binary's `--version`.
3. Stamps the file with the current time and swaps it in; the old one goes to
   `loomux.old.exe` or the first free numbered slot. The next bridge replaces
   the running `serve`.

Writes `<state dir>/update.json` (`checked_at`, `executable`, `running`,
`result` = `current` | `updated` | `skipped` | `failed`, `version`, `error`).
A pass that finds `update.lock` held steps aside and writes nothing.

| Exit | Meaning |
|---|---|
| 0 | `already current (vX)` or `updated to vX` |
| 1 | the pass failed, or another pass is running |
| 2 | skipped: not on Windows, a development build, or not the machine-wide binary |
````

Beim Abschnitt `### \`loomux hook session-start\`` einen Satz anhängen:

```markdown
It also reads `update.json` and warns when `serve` runs from another binary
than `<state dir>/bin/loomux.exe`, or when the last self-update pass failed.
```

- [ ] **Step 4: cli-reference (de)**

Dieselben Stellen in `docs/de/cli-reference.md` (Überschriften mit
`grep -n "serve stop\|session-start" docs/de/cli-reference.md` finden), auf
Deutsch:

````markdown
### `loomux self-update`

Ein Self-Update-Durchlauf von Hand; `serve` fährt denselben eine Minute nach
dem Start und danach alle 24 Stunden. Er wirkt nur auf das maschinenweite
Binary, `<Zustandsverzeichnis>/bin/loomux.exe`, wenn das das laufende Binary
ist und eine Release-Version trägt.

1. Listet die Releases über `gh release list` und nimmt die höchste Version
   im Kanal des laufenden Binarys (`beta` nimmt Prereleases, `stable` nicht).
2. Lädt das Windows-Asset und `SHA256SUMS` über `gh release download`, prüft
   die Prüfsumme und die `--version` des neuen Binarys.
3. Stempelt die Datei mit der aktuellen Zeit und tauscht sie ein; das alte
   Binary kommt nach `loomux.old.exe` oder in den ersten freien nummerierten
   Platz. Die nächste Brücke ersetzt den laufenden `serve`.

Schreibt `<Zustandsverzeichnis>/update.json` (`checked_at`, `executable`,
`running`, `result` = `current` | `updated` | `skipped` | `failed`, `version`,
`error`). Ein Durchlauf, der `update.lock` belegt findet, tritt zurück und
schreibt nichts.

| Exit | Bedeutung |
|---|---|
| 0 | `already current (vX)` oder `updated to vX` |
| 1 | der Durchlauf ist gescheitert, oder ein anderer läuft |
| 2 | ausgelassen: nicht Windows, ein Entwicklungs-Build oder nicht das maschinenweite Binary |
````

Satz bei `session-start`:

```markdown
Außerdem liest er `update.json` und warnt, wenn `serve` aus einem anderen
Binary als `<Zustandsverzeichnis>/bin/loomux.exe` läuft oder der letzte
Self-Update-Durchlauf gescheitert ist.
```

- [ ] **Step 5: Migrationsplan**

In `docs/en/migration.md`, Zeile „Self-Update“: Status
`📋 **Specified** (\`docs/.superpowers/specs/2026-09-23-self-update-design.md\`)`
ersetzen durch `✅ **Implemented**`. In `docs/de/migration.md` entsprechend
`📋 **Spezifiziert** (…)` durch `✅ **Implementiert**`.

- [ ] **Step 6: Tor laufen lassen**

Run: `sh ci/gate.sh`
Expected: alle Lanes `ok` (lint/go, test/go, coverage/go, lint/wiki).

- [ ] **Step 7: Commit**

```bash
git add README.md README.de.md docs/en docs/de
git commit -m "docs: document installing the machine-wide binary and loomux self-update"
```

- [ ] **Step 8: Übergabe**

Den Skill `release-pr` benutzen: Commits nach Thema gruppieren (die
`fixup!`-Commits des Spec-Commits hineinfalten), Label `release:minor`,
Body mit `Release: minor — …` und `## Changelog` (`### Added`: `loomux
self-update`; `serve` keeps the machine-wide binary current; session start
warns when serve runs elsewhere or the update failed). Den Push-Befehl dem
Menschen nennen, nicht selbst pushen.

---

## Nach dem Merge (Handarbeit, kein Task)

Sobald das Release mit diesem Code erschienen ist, einmal von Hand
`%LOCALAPPDATA%\loomux\bin\loomux.exe self-update` ausführen: Es muss
`updated to v…` melden, und `update.json` muss `updated` tragen. Erst danach
aktualisiert sich das installierte Binary selbst — v2.7.0 kennt die Schleife
noch nicht.
