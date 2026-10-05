# Release-Brücke Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die nächste Beta der alten Zählung (heute 7.2.0) bringt den neuen Versionsvergleich, den Beta-Kanal als Markierung, `upgrade --beta|--stable|--version` und `dev release next-beta`, damit jede Installation später v1.0.0 selbst holt.

**Architecture:** `internal/selfupdate` vergleicht künftig *gemeldete* Versionen (`7.2.0 (beta)`, `1.0.0`, `1.1.0-beta.2`) und *Releases* (Tag plus `isPrerelease`) über einen gemeinsamen Typ `version` mit Zählung (alt/neu), Nummer und Beta-Nummer. Der Kanal einer Maschine liegt in `<Zustandsverzeichnis>/channel`; ein Lauf hat einen Modus (Kanal, `beta`, `stable`, feste Version). `release.sh` und `release.yml` bleiben unverändert, damit die Brücke selbst noch als Beta-Pre-Release erscheint.

**Tech Stack:** Go, `gh` als Runner (gefälscht über `fakeGH` in `internal/selfupdate/fake_test.go`).

**Spec:** `docs/.superpowers/specs/2026-10-05-loomux-release-neustart-design.md` (Schritt 1 „Brücke“, Abschnitte „Versionsschema“, „Vergleich und Kanal“, „`loomux upgrade`“).

## Global Constraints

- Beta-Form: `X.Y.Z-beta.N`, N ≥ 1, ohne führende Null; jeder andere Suffix (auch `0.0.0-dev`, `-rc1`) ist keine Version.
- Alte Zählung: eine Version ohne Suffix, deren Binary `(beta)` hinter der Nummer meldet, oder ein Release ohne Suffix mit `isPrerelease=true`. Sie rangiert unter jeder Version der neuen Zählung.
- Rangfolge innerhalb einer Zählung: SemVer, `1.1.0-beta.2` < `1.1.0-beta.10` < `1.1.0`.
- Markierung: Datei `<Zustandsverzeichnis>/channel`, Inhalt `beta` (ein abschließender Zeilenumbruch erlaubt). Fehlt sie: stabil. Fremder Inhalt oder unlesbar: stabil, und der Fehler steht in `update.json`.
- Ein Binary der alten Zählung ohne Datei nimmt die Releases der alten Zählung und die stabilen der neuen, aber keine neue Beta; mit Datei nimmt es alles (nachgetragen nach dem Abschluss-Review).
- Stabiler Kanal nimmt kein Release mit `isPrerelease=true`; der Beta-Kanal nimmt alle.
- `--beta`, `--stable`, `--version` schließen sich gegenseitig aus (Exit 2).
- Keine Änderung an `.github/scripts/release.sh`, `.github/workflows/release.yml`, `cli.Channel`, `dev release build --channel`.
- `RELEASE_CHANNEL` muss bis zum Neustart-Schritt 3b `beta` sein. Wird vorher ein 7.x ohne `--prerelease` veröffentlicht, liest `releaseVersion` es als neue Zählung. Das installierte Binary druckt dann kein `(beta)`, und ein Binary im stabilen Kanal käme nie zu v1.0.0. Darum prüft der Mensch vor dem Release der Brücke mit `gh variable get RELEASE_CHANNEL`, dass die Variable `beta` ist.
- 100 % Coverage je Funktion; jeder neue Test läuft vor seinem Code rot (Befehl und `--- FAIL` im Bericht); je neuer Funktion eine Mutationsrunde per `go test -overlay` mit Windows-Pfaden (`C:/…`), jede Teilbedingung einzeln gestrichen.
- Kommentare englisch, ohne Plan-, Stufen- oder Spec-Nummern; Commits nach Conventional Commits ohne Arbeitspapier-Namen.

## Review Focus

1. **Brücke gegen v1.0.0:** Ein laufendes `7.2.0` mit Kanal `beta`, Liste `[v7.2.0 pre, v1.0.0 stabil]` → nimmt v1.0.0. Test in Task 3.
2. **Neues Binary gegen altes Pre-Release:** Ein laufendes `1.0.0` ohne Markierung, Liste `[v7.1.0 pre, v1.0.0]` → bleibt aktuell; mit Markierung ebenfalls (alte Zählung rangiert darunter). Test in Task 3.
3. **`AtLeast` beim `init`:** installiert `7.2.0 (beta)`, `init` ist `1.0.0` → nicht neu genug; installiert `1.0.0`, `init` ist `7.2.0 (beta)` → neu genug. Test in Task 1.
4. **Beta-Markierung überlebt ein stabiles Release:** markiert, läuft `1.1.0-beta.2`, Liste `[v1.1.0, v1.2.0-beta.1]` → nimmt `1.2.0-beta.1`; ohne die Beta in der Liste → nimmt `1.1.0`, und die Markierung bleibt. Test in Task 3.
5. **`--version` auf eine fehlende Version:** `gh release view` scheitert → Exit 1 mit „no release x.y.z“, keine Markierung geändert. Test in Task 3/4.

---

### Task 1: Versionen beider Zählungen vergleichen

**Files:**
- Modify: `internal/selfupdate/version.go` (ganze Datei außer `Release`)
- Modify: `internal/selfupdate/update.go:112,118,133-145` (Aufrufe von `Newer`, `InstalledVersion`)
- Modify: `internal/cli/init.go:206` (`Running{Version: bareVersion(), …}`)
- Modify: `internal/setup/facts.go:82-83,98-99` (Kommentar: „what the running init reports, cli's bareVersion“)
- Test: `internal/selfupdate/version_test.go`, `internal/setup/plan_test.go` (ein Fall)

**Interfaces:**
- Produces:
  - `type version struct { old bool; num [3]int; beta int }` und `func (v version) less(w version) bool`
  - `func parseVersion(s string) (version, bool)` — `"1.2.3"`, `"v1.2.3"`, `"1.2.3-beta.4"`; nie `old`
  - `func parseReported(s string) (version, bool)` — `"7.2.0 (beta)"` ist alt, `"1.0.0"`, `"1.1.0-beta.1"`, `"1.1.0-beta.1 (beta)"` neu; jeder andere Kanaltext oder Rest: false
  - `func releaseVersion(r Release) (version, bool)`
  - `func Reported(ver, channel string) string` — `ver` bei leerem oder `stable` Kanal, sonst `ver + " (" + channel + ")"`
  - `func Newer(r Release, running string) bool` (running ist gemeldet)
  - `func AtLeast(have, want string) bool` (beide gemeldet)
  - `func IsVersion(s string) bool` (gemeldet)
  - `func InstalledVersion(ctx, run, path) (string, bool)` gibt die ganze gemeldete Form zurück, z. B. `"7.2.0 (beta)"`

- [ ] **Step 1: Tests schreiben** (ersetzen `TestNewer`, `TestAtLeast` in `version_test.go`)

```go
func TestParseVersion(t *testing.T) {
	for _, c := range []struct {
		in   string
		want version
		ok   bool
	}{
		{"1.2.3", version{num: [3]int{1, 2, 3}}, true},
		{"v1.2.3", version{num: [3]int{1, 2, 3}}, true},
		{"1.2.3-beta.4", version{num: [3]int{1, 2, 3}, beta: 4}, true},
		{"1.2.3-beta.10", version{num: [3]int{1, 2, 3}, beta: 10}, true},
		{"1.2.3-beta.0", version{}, false},
		{"1.2.3-beta.04", version{}, false},
		{"1.2.3-beta", version{}, false},
		{"1.2.3-rc1", version{}, false},
		{"0.0.0-dev", version{}, false},
		{"1.02.3", version{}, false},
		{"1.2", version{}, false},
		{"", version{}, false},
	} {
		got, ok := parseVersion(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parseVersion(%q) = %+v, %v; want %+v, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestParseReportedKnowsTheOldCount(t *testing.T) {
	for _, c := range []struct {
		in   string
		want version
		ok   bool
	}{
		{"7.2.0 (beta)", version{old: true, num: [3]int{7, 2, 0}}, true},
		{"7.2.0", version{num: [3]int{7, 2, 0}}, true},
		{"1.1.0-beta.1", version{num: [3]int{1, 1, 0}, beta: 1}, true},
		{"1.1.0-beta.1 (beta)", version{num: [3]int{1, 1, 0}, beta: 1}, true},
		{"7.2.0 (nightly)", version{}, false},
		{"7.2.0 (beta) x", version{}, false},
		{"0.0.0-dev", version{}, false},
	} {
		got, ok := parseReported(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parseReported(%q) = %+v, %v; want %+v, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestReleaseVersionReadsTheCountFromThePrereleaseFlag(t *testing.T) {
	for _, c := range []struct {
		r    Release
		want version
	}{
		{Release{Tag: "v7.1.0", Prerelease: true}, version{old: true, num: [3]int{7, 1, 0}}},
		{Release{Tag: "v1.0.0"}, version{num: [3]int{1, 0, 0}}},
		{Release{Tag: "v1.1.0-beta.2", Prerelease: true}, version{num: [3]int{1, 1, 0}, beta: 2}},
	} {
		if got, ok := releaseVersion(c.r); !ok || got != c.want {
			t.Errorf("releaseVersion(%+v) = %+v, %v; want %+v", c.r, got, ok, c.want)
		}
	}
	if _, ok := releaseVersion(Release{Tag: "nightly"}); ok {
		t.Error("nightly read as a version")
	}
}

func TestLessOrdersBothCounts(t *testing.T) {
	ordered := []string{"7.1.0 (beta)", "7.2.0 (beta)", "1.0.0", "1.1.0-beta.2", "1.1.0-beta.10", "1.1.0", "1.1.1", "2.0.0"}
	for i := range ordered {
		for j := range ordered {
			a, _ := parseReported(ordered[i])
			b, _ := parseReported(ordered[j])
			if got := a.less(b); got != (i < j) {
				t.Errorf("%s < %s = %v, want %v", ordered[i], ordered[j], got, i < j)
			}
		}
	}
}

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		r       Release
		running string
		want    bool
	}{
		{Release{Tag: "v2.8.0", Prerelease: true}, "2.7.0 (beta)", true},
		{Release{Tag: "v2.7.0", Prerelease: true}, "2.7.0 (beta)", false},
		{Release{Tag: "v1.0.0"}, "7.2.0 (beta)", true},
		{Release{Tag: "v7.1.0", Prerelease: true}, "1.0.0", false},
		{Release{Tag: "v1.1.0"}, "1.1.0-beta.3", true},
		{Release{Tag: "v1.1.0-beta.3", Prerelease: true}, "1.1.0", false},
		{Release{Tag: "v2.8.0-rc1"}, "2.7.0", false},
		{Release{Tag: "v2.8.0"}, "0.0.0-dev", false},
	} {
		if got := Newer(c.r, c.running); got != c.want {
			t.Errorf("Newer(%+v, %q) = %v, want %v", c.r, c.running, got, c.want)
		}
	}
}

func TestAtLeast(t *testing.T) {
	for _, c := range []struct {
		have, want string
		ok         bool
	}{
		{"2.13.0 (beta)", "2.13.0 (beta)", true},
		{"2.11.1 (beta)", "2.13.0 (beta)", false},
		{"7.2.0 (beta)", "1.0.0", false},
		{"1.0.0", "7.2.0 (beta)", true},
		{"1.1.0-beta.1", "1.1.0", false},
		{"", "2.13.0", false},
		{"2.13.0", "0.0.0-dev", false},
	} {
		if got := AtLeast(c.have, c.want); got != c.ok {
			t.Errorf("AtLeast(%q, %q) = %v, want %v", c.have, c.want, got, c.ok)
		}
	}
	for s, want := range map[string]bool{"2.13.0": true, "7.2.0 (beta)": true, "1.1.0-beta.1": true, DevVersion: false, "": false} {
		if got := IsVersion(s); got != want {
			t.Errorf("IsVersion(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestReported(t *testing.T) {
	for _, c := range [][3]string{{"7.2.0", "beta", "7.2.0 (beta)"}, {"1.0.0", "", "1.0.0"}, {"1.0.0", "stable", "1.0.0"}} {
		if got := Reported(c[0], c[1]); got != c[2] {
			t.Errorf("Reported(%q, %q) = %q, want %q", c[0], c[1], got, c[2])
		}
	}
}
```

In `internal/setup/plan_test.go`, Tabelle von `TestAntigravityEntriesWaitForAnInstalledBinaryAsNewAsInit` (Zeile 502ff.), zwei Fälle dazu:

```go
		{"old count before the restart", "1.0.0", "7.2.0 (beta)", true,
			"antigravity: no entries; the installed loomux 7.2.0 (beta) is older than this init 1.0.0; run loomux upgrade"},
		{"restart after the old count", "7.2.0 (beta)", "1.0.0", true, ""},
```

- [ ] **Step 2: Rot laufen lassen**

Run: `go test ./internal/selfupdate/ ./internal/setup/ -run 'TestParse|TestRelease|TestLess|TestNewer|TestAtLeast|TestReported|Antigravity' -count=1`
Expected: Build-Fehler `undefined: parseReported` usw.; danach mit Stubs (`parseReported` = `parseVersion`, `Newer` ohne Zählung) `--- FAIL: TestNewer` beim Fall `v1.0.0` gegen `7.2.0 (beta)`.

- [ ] **Step 3: `version.go` umsetzen**

```go
// version is a release number of either count: old marks one from before the
// restart at 1.0.0, which ranks below every number after it; beta is N of an
// -beta.N suffix, 0 for a release without one.
type version struct {
	old  bool
	num  [3]int
	beta int
}

// parseVersion reads "1.2.3", "v1.2.3" or "1.2.3-beta.4". Any other suffix,
// and a leading zero anywhere, is no version the updater acts on.
func parseVersion(s string) (version, bool) {
	core, suffix, hasSuffix := strings.Cut(strings.TrimPrefix(s, "v"), "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return version{}, false
	}
	var v version
	for i, part := range parts {
		n, ok := plainNumber(part)
		if !ok {
			return version{}, false
		}
		v.num[i] = n
	}
	if !hasSuffix {
		return v, true
	}
	n, ok := plainNumber(strings.TrimPrefix(suffix, "beta."))
	if !ok || n == 0 || !strings.HasPrefix(suffix, "beta.") {
		return version{}, false
	}
	v.beta = n
	return v, true
}

// plainNumber is a non-negative decimal without a leading zero.
func plainNumber(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	return n, err == nil && n >= 0 && s == strconv.Itoa(n)
}

// parseReported reads what a binary says it is after "loomux ": the version,
// optionally followed by " (beta)". A plain number with that channel is the
// old count; a beta of the new count carries its suffix and may say so too.
func parseReported(s string) (version, bool) {
	ver, channel, hasChannel := strings.Cut(s, " ")
	v, ok := parseVersion(ver)
	if !ok || (hasChannel && channel != "(beta)") {
		return version{}, false
	}
	v.old = hasChannel && v.beta == 0
	return v, true
}

// releaseVersion places a release in its count. A tag alone cannot: v8.0.0
// of the old count and v1.0.0 of the new look alike, but every release of
// the old count was published as a pre-release and every new pre-release
// carries -beta.N.
func releaseVersion(r Release) (version, bool) {
	v, ok := parseVersion(r.Tag)
	v.old = ok && r.Prerelease && v.beta == 0
	return v, ok
}

func (v version) less(w version) bool {
	if v.old != w.old {
		return v.old
	}
	if v.num != w.num {
		for i := range v.num {
			if v.num[i] != w.num[i] {
				return v.num[i] < w.num[i]
			}
		}
	}
	switch {
	case v.beta == w.beta:
		return false
	case v.beta == 0:
		return false
	case w.beta == 0:
		return true
	}
	return v.beta < w.beta
}

// Reported is what a binary built with ver and channel says after "loomux ",
// the form Newer, AtLeast and IsVersion read.
func Reported(ver, channel string) string {
	if channel == "" || channel == "stable" {
		return ver
	}
	return ver + " (" + channel + ")"
}

// Newer reports whether r is a later release than the running binary, given
// as Reported. A side that does not parse is never newer, so a guess never
// replaces a binary and a development build never asks for one.
func Newer(r Release, running string) bool {
	t, ok := releaseVersion(r)
	if !ok {
		return false
	}
	have, ok := parseReported(running)
	return ok && have.less(t)
}

// AtLeast reports whether have is no older than want, both as Reported.
// Unlike !Newer it is false when either side does not parse: a binary that
// cannot say what it is does not pass for one new enough.
func AtLeast(have, want string) bool {
	h, ok := parseReported(have)
	if !ok {
		return false
	}
	w, ok := parseReported(want)
	return ok && !h.less(w)
}

// IsVersion reports whether s, as Reported, is a release version, as a
// development build's DevVersion is not.
func IsVersion(s string) bool {
	_, ok := parseReported(s)
	return ok
}
```

`update.go`: `Newer(rel, Reported(o.Version, o.Channel))` und `Newer(rel, have)`; `InstalledVersion`:

```go
// InstalledVersion is what the binary at path says it is after "loomux ",
// in the form Reported gives. An answer that is not a release version is no
// answer, so that the pass falls back to the running version rather than
// holding back an update on a guess.
func InstalledVersion(ctx context.Context, run Runner, path string) (string, bool) {
	out, err := call(ctx, run, path, "--version")
	if err != nil {
		return "", false
	}
	rest, found := strings.CutPrefix(firstLine(string(out)), "loomux ")
	return rest, found && IsVersion(rest)
}
```

`internal/cli/init.go:206`: `setup.Running{Version: bareVersion(), VersionOf: binaryVersion}`.

- [ ] **Step 4: Grün und ganze Pakete**

Run: `go test ./internal/selfupdate/ ./internal/setup/ ./internal/cli/ -count=1`
Expected: PASS. Bestehende Tests, die `InstalledVersion` mit `"loomux 2.7.0 (beta)"` erwarten, bekommen `"2.7.0 (beta)"` statt `"2.7.0"`; das ist die neue Form, nicht ein Fehler.

- [ ] **Step 5: Mutationsrunde** über `parseVersion`, `parseReported`, `releaseVersion`, `less`, `AtLeast`: jede Teilbedingung einzeln streichen (`v.old = …` auf `false`, `n == 0` weg, `channel != "(beta)"` weg, Zweige von `less` vertauscht), je Mutant die tötende Testzeile im Bericht.

- [ ] **Step 6: Commit**

```bash
git add internal/selfupdate internal/setup internal/cli/init.go
git commit -F <datei>   # feat(selfupdate): rank betas and the count before 1.0.0 below every new release
```

### Task 2: Kanal-Markierung

**Files:**
- Create: `internal/selfupdate/channel.go`
- Test: `internal/selfupdate/channel_test.go`

**Interfaces:**
- Produces: `func ReadChannel(stateDir string) (beta bool, err error)`, `func WriteChannel(stateDir string, beta bool) error`, `func ChannelPath(stateDir string) string`

- [ ] **Step 1: Tests**

```go
func TestReadChannel(t *testing.T) {
	for _, c := range []struct {
		name, body string
		write      bool
		beta       bool
		err        string
	}{
		{"missing", "", false, false, ""},
		{"beta", "beta", true, true, ""},
		{"beta with newline", "beta\n", true, true, ""},
		{"beta with CRLF", "beta\r\n", true, true, ""},
		{"foreign", "nightly\n", true, false, `channel file holds "nightly", not beta`},
	} {
		dir := t.TempDir()
		if c.write {
			if err := os.WriteFile(ChannelPath(dir), []byte(c.body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		beta, err := ReadChannel(dir)
		if beta != c.beta || (err == nil) != (c.err == "") || (err != nil && !strings.Contains(err.Error(), c.err)) {
			t.Errorf("%s: ReadChannel = %v, %v; want %v, %q", c.name, beta, err, c.beta, c.err)
		}
	}
}

func TestReadChannelReportsAnUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(ChannelPath(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	if beta, err := ReadChannel(dir); beta || err == nil {
		t.Fatalf("ReadChannel on a directory = %v, %v", beta, err)
	}
}

func TestWriteChannelSetsAndClears(t *testing.T) {
	dir := t.TempDir()
	if err := WriteChannel(dir, false); err != nil {
		t.Fatalf("clearing a missing marker: %v", err)
	}
	if err := WriteChannel(dir, true); err != nil {
		t.Fatal(err)
	}
	if beta, err := ReadChannel(dir); !beta || err != nil {
		t.Fatalf("after set: %v, %v", beta, err)
	}
	if err := WriteChannel(dir, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ChannelPath(dir)); !os.IsNotExist(err) {
		t.Fatalf("marker still there: %v", err)
	}
}

func TestWriteChannelFailsWhereItCannotWrite(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	if err := WriteChannel(dir, true); err == nil {
		t.Fatal("wrote into a missing directory")
	}
	full := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ChannelPath(full), "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteChannel(full, false); err == nil {
		t.Fatal("removed a non-empty directory")
	}
}
```

- [ ] **Step 2: Rot** — `go test ./internal/selfupdate/ -run Channel -count=1` → `undefined: ChannelPath`.

- [ ] **Step 3: Umsetzen**

```go
package selfupdate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ChannelPath is the marker that puts a machine on the beta channel: its
// presence, holding "beta", lets every pass take betas as well as stable
// releases, until upgrade --stable or --version <stable> removes it.
func ChannelPath(stateDir string) string { return filepath.Join(stateDir, "channel") }

// ReadChannel reports whether the machine is on the beta channel. A missing
// marker is the stable channel; one that cannot be read or holds anything
// else counts as stable too, and the error says why.
func ReadChannel(stateDir string) (bool, error) {
	data, err := os.ReadFile(ChannelPath(stateDir))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read channel file: %w", err)
	}
	if got := strings.TrimSpace(string(data)); got != "beta" {
		return false, fmt.Errorf("channel file holds %q, not beta", got)
	}
	return true, nil
}

// WriteChannel sets the marker for beta and removes it otherwise.
func WriteChannel(stateDir string, beta bool) error {
	path := ChannelPath(stateDir)
	if beta {
		return os.WriteFile(path, []byte("beta\n"), 0o644)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
```

- [ ] **Step 4: Grün** — `go test ./internal/selfupdate/ -run Channel -count=1` → PASS; Mutationsrunde (`TrimSpace` weg, `!errors.Is` → `err != nil`, `beta` invertiert).

- [ ] **Step 5: Commit** — `feat(selfupdate): keep the beta channel as a marker in the state directory`

### Task 3: Auswahl und Modus eines Laufs

**Files:**
- Modify: `internal/selfupdate/version.go` (`pick`), `internal/selfupdate/gh.go` (`latest`, neu `view`, `channelName` entfällt), `internal/selfupdate/update.go` (`Options`, `run`, `installLocked`, `Run`)
- Modify: `internal/selfupdate/fake_test.go` (Schlüssel `view`, Feld `view string`)
- Modify: `internal/hooks/hook_session_start.go:184-206` (`updateWarnings`), Test: `internal/hooks/hook_session_start_test.go:629ff.`
- Test: `internal/selfupdate/version_test.go` (`TestPickFollowsTheChannel` neu), `gh_test.go`, `update_test.go`

**Interfaces:**
- Consumes: `releaseVersion`, `Reported`, `Newer`, `ReadChannel`, `WriteChannel` (Tasks 1, 2)
- Produces:
  - `const ( ModeChannel = ""; ModeBeta = "beta"; ModeStable = "stable" )`
  - `Options.Mode string`, `Options.Pin string` (Version ohne `v`, z. B. `"1.1.0-beta.2"`); `Pin` und `Mode` schließen sich aus, das prüft der Aufrufer
  - `func pick(releases []Release, beta bool) (Release, bool)`
  - `func latest(ctx context.Context, run Runner, beta bool) (Release, error)` — Fehlertext `no release in channel beta|stable`
  - `func view(ctx context.Context, run Runner, ver string) (Release, error)` — `gh release view v<ver> --repo Repo --json tagName,isPrerelease`; Fehlertext `no release <ver>: <gh-fehler>`

- [ ] **Step 1: Tests**

`version_test.go`, ersetzt `TestPickFollowsTheChannel`:

```go
func TestPickFollowsTheChannel(t *testing.T) {
	old := []Release{{Tag: "v7.1.0", Prerelease: true}, {Tag: "v7.2.0", Prerelease: true}}
	restarted := []Release{{Tag: "v7.2.0", Prerelease: true}, {Tag: "v1.0.0"}}
	later := []Release{{Tag: "v1.2.0-beta.1", Prerelease: true}, {Tag: "v1.1.0"}, {Tag: "v1.1.0-beta.10", Prerelease: true}, {Tag: "v1.1.0-beta.2", Prerelease: true}}
	for _, c := range []struct {
		name     string
		releases []Release
		beta     bool
		want     string
		found    bool
	}{
		{"beta over the old count", old, true, "v7.2.0", true},
		{"stable over the old count", old, false, "", false},
		{"beta: the restart outranks the bridge", restarted, true, "v1.0.0", true},
		{"stable: the restart", restarted, false, "v1.0.0", true},
		{"beta takes the newest beta", later, true, "v1.2.0-beta.1", true},
		{"stable skips betas", later, false, "v1.1.0", true},
		{"nothing listed", nil, true, "", false},
		{"no tag is a version", []Release{{Tag: "nightly", Prerelease: true}, {Tag: "v2.8"}}, true, "", false},
	} {
		got, found := pick(c.releases, c.beta)
		if found != c.found || got.Tag != c.want {
			t.Errorf("%s: pick = %q, %v; want %q, %v", c.name, got.Tag, found, c.want, c.found)
		}
	}
}
```

`gh_test.go`: `latest(…, true)` statt `"beta"`, `latest(…, false)` statt `""`; Fehlertexte bleiben `no release in channel beta|stable`. Neu:

```go
func TestViewAsksForOneTag(t *testing.T) {
	f := &fakeGH{view: `{"tagName":"v1.1.0-beta.2","isPrerelease":true}`}
	var args []string
	run := func(ctx context.Context, name string, a ...string) ([]byte, error) {
		args = a
		return f.run(ctx, name, a...)
	}
	rel, err := view(context.Background(), run, "1.1.0-beta.2")
	if err != nil || rel != (Release{Tag: "v1.1.0-beta.2", Prerelease: true}) {
		t.Fatalf("view = %+v, %v", rel, err)
	}
	if !slices.Equal(args[:3], []string{"release", "view", "v1.1.0-beta.2"}) || !slices.Contains(args, Repo) {
		t.Errorf("gh args = %v", args)
	}
}

func TestViewNamesAVersionThatIsNotThere(t *testing.T) {
	f := &fakeGH{fail: map[string]error{"view": errors.New("release not found")}}
	_, err := view(context.Background(), f.run, "9.9.9")
	if err == nil || err.Error() != "no release 9.9.9: release not found" {
		t.Fatalf("err = %v", err)
	}
}

func TestViewRefusesWhatIsNoVersion(t *testing.T) {
	f := &fakeGH{view: `{"tagName":"nightly"}`}
	if _, err := view(context.Background(), f.run, "nightly"); err == nil {
		t.Fatal("took nightly")
	}
	f = &fakeGH{view: `not json`}
	if _, err := view(context.Background(), f.run, "1.0.0"); err == nil {
		t.Fatal("took a broken answer")
	}
}
```

`update_test.go` (Hilfe `releases(list string, ver string)` wie `release(ver)`, aber mit eigener Liste; `installed(t, f)` wie bisher mit `Version: "2.7.0", Channel: "beta"`):

```go
// releasesOf is release(ver) with a list of its own: ver is the one asset
// the download serves.
func releasesOf(ver, list string) *fakeGH {
	f := release(ver)
	f.list = list
	return f
}

func TestRunTheBridgeTakesTheRestart(t *testing.T) {
	f := releasesOf("1.0.0", `[{"tagName":"v7.2.0","isPrerelease":true},{"tagName":"v1.0.0","isPrerelease":false}]`)
	f.version = "loomux 1.0.0\n"
	o := installed(t, f)
	o.Version, f.installed = "7.2.0", "loomux 7.2.0 (beta)\n"
	if res := Run(context.Background(), o); res.Outcome != Updated || res.Version != "1.0.0" {
		t.Fatalf("res = %+v", res)
	}
}

func TestRunANewBinaryLeavesTheOldCountAlone(t *testing.T) {
	for _, marked := range []bool{false, true} {
		f := releasesOf("7.1.0", `[{"tagName":"v7.1.0","isPrerelease":true},{"tagName":"v1.0.0","isPrerelease":false}]`)
		o := installed(t, f)
		o.Version, o.Channel, f.installed = "1.0.0", "", "loomux 1.0.0\n"
		if marked {
			if err := WriteChannel(o.StateDir, true); err != nil {
				t.Fatal(err)
			}
		}
		if res := Run(context.Background(), o); res.Outcome != Current || res.Version != "1.0.0" {
			t.Fatalf("marked=%v: res = %+v", marked, res)
		}
	}
}

func TestRunAMarkedBetaStaysOnBetasAfterAStableRelease(t *testing.T) {
	f := releasesOf("1.2.0-beta.1", `[{"tagName":"v1.1.0","isPrerelease":false},{"tagName":"v1.2.0-beta.1","isPrerelease":true}]`)
	f.version = "loomux 1.2.0-beta.1\n"
	o := installed(t, f)
	o.Version, o.Channel, f.installed = "1.1.0-beta.2", "", "loomux 1.1.0-beta.2\n"
	if err := WriteChannel(o.StateDir, true); err != nil {
		t.Fatal(err)
	}
	if res := Run(context.Background(), o); res.Outcome != Updated || res.Version != "1.2.0-beta.1" {
		t.Fatalf("res = %+v", res)
	}
	if beta, _ := ReadChannel(o.StateDir); !beta {
		t.Fatal("the marker went")
	}
}

func TestRunAMarkedBetaTakesTheStableReleaseThatOvertakesIt(t *testing.T) {
	f := releasesOf("1.1.0", `[{"tagName":"v1.1.0","isPrerelease":false},{"tagName":"v1.1.0-beta.2","isPrerelease":true}]`)
	f.version = "loomux 1.1.0\n"
	o := installed(t, f)
	o.Version, o.Channel, f.installed = "1.1.0-beta.2", "", "loomux 1.1.0-beta.2\n"
	if err := WriteChannel(o.StateDir, true); err != nil {
		t.Fatal(err)
	}
	if res := Run(context.Background(), o); res.Outcome != Updated || res.Version != "1.1.0" {
		t.Fatalf("res = %+v", res)
	}
	if beta, _ := ReadChannel(o.StateDir); !beta {
		t.Fatal("the marker went")
	}
}

func TestRunUnmarkedNewBinaryTakesNoBeta(t *testing.T) {
	f := releasesOf("1.2.0-beta.1", `[{"tagName":"v1.1.0","isPrerelease":false},{"tagName":"v1.2.0-beta.1","isPrerelease":true}]`)
	o := installed(t, f)
	o.Version, o.Channel, f.installed = "1.1.0", "", "loomux 1.1.0\n"
	if res := Run(context.Background(), o); res.Outcome != Current {
		t.Fatalf("res = %+v", res)
	}
}

func TestRunRecordsAForeignMarkerAndTakesStable(t *testing.T) {
	f := releasesOf("1.2.0-beta.1", `[{"tagName":"v1.1.0","isPrerelease":false},{"tagName":"v1.2.0-beta.1","isPrerelease":true}]`)
	o := installed(t, f)
	o.Version, o.Channel, f.installed = "1.1.0", "", "loomux 1.1.0\n"
	if err := os.WriteFile(ChannelPath(o.StateDir), []byte("nightly"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := Run(context.Background(), o)
	if res.Outcome != Current || !strings.Contains(recorded(t, o).Error, `channel file holds "nightly"`) {
		t.Fatalf("res = %+v, status = %+v", res, recorded(t, o))
	}
}

func TestRunModes(t *testing.T) {
	list := `[{"tagName":"v1.1.0","isPrerelease":false},{"tagName":"v1.2.0-beta.1","isPrerelease":true},{"tagName":"v1.0.0","isPrerelease":false}]`
	for _, c := range []struct {
		name, mode, pin, running, asset string
		marked                          bool
		outcome                         Outcome
		version                         string
		markerAfter                     bool
	}{
		{"beta sets the marker and takes the newest", ModeBeta, "", "1.1.0", "1.2.0-beta.1", false, Updated, "1.2.0-beta.1", true},
		{"beta when current still sets the marker", ModeBeta, "", "1.2.0-beta.1", "1.2.0-beta.1", false, Current, "1.2.0-beta.1", true},
		{"stable goes back from a beta", ModeStable, "", "1.2.0-beta.1", "1.1.0", true, Updated, "1.1.0", false},
		{"pin a beta sets the marker", "", "1.2.0-beta.1", "1.1.0", "1.2.0-beta.1", false, Updated, "1.2.0-beta.1", true},
		{"pin a stable downgrades and clears", "", "1.0.0", "1.2.0-beta.1", "1.0.0", true, Updated, "1.0.0", false},
		{"pin what runs is current", "", "1.1.0", "1.1.0", "1.1.0", true, Current, "1.1.0", false},
	} {
		f := releasesOf(c.asset, list)
		f.view = fmt.Sprintf(`{"tagName":"v%s","isPrerelease":%v}`, c.pin, strings.Contains(c.pin, "-"))
		f.version = "loomux " + c.asset + "\n"
		o := installed(t, f)
		o.Version, o.Channel, f.installed = c.running, "", "loomux "+c.running+"\n"
		o.Mode, o.Pin = c.mode, c.pin
		if c.marked {
			if err := WriteChannel(o.StateDir, true); err != nil {
				t.Fatal(err)
			}
		}
		res := Run(context.Background(), o)
		if res.Outcome != c.outcome || res.Version != c.version {
			t.Errorf("%s: res = %+v", c.name, res)
		}
		if beta, _ := ReadChannel(o.StateDir); beta != c.markerAfter {
			t.Errorf("%s: marker = %v, want %v", c.name, beta, c.markerAfter)
		}
	}
}

func TestRunPinOfAMissingVersionFailsAndKeepsTheMarker(t *testing.T) {
	f := release("1.1.0")
	f.fail["view"] = errors.New("release not found")
	o := installed(t, f)
	o.Pin = "9.9.9"
	if err := WriteChannel(o.StateDir, true); err != nil {
		t.Fatal(err)
	}
	res := Run(context.Background(), o)
	if res.Outcome != Failed || res.Err.Error() != "no release 9.9.9: release not found" {
		t.Fatalf("res = %+v", res)
	}
	if beta, _ := ReadChannel(o.StateDir); !beta {
		t.Fatal("a failed pass changed the marker")
	}
}

func TestRunReportsAMarkerItCouldNotWrite(t *testing.T) {
	f := release("2.8.0")
	o := installed(t, f)
	o.Mode = ModeBeta
	if err := os.MkdirAll(filepath.Join(ChannelPath(o.StateDir), "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := Run(context.Background(), o)
	if res.Outcome != Updated || res.Err == nil || !strings.Contains(res.Err.Error(), "channel") {
		t.Fatalf("res = %+v", res)
	}
}
```

Hinweis zu `TestRunReportsAMarkerItCouldNotWrite`: Ein Verzeichnis an `ChannelPath` lässt `ReadChannel` scheitern (wird „stabil“) und `WriteChannel(true)` ebenso. Der Lauf installiert trotzdem (Modus `beta` nimmt alles), und `res.Err` nennt die Markierung.

- [ ] **Step 2: Rot** — `go test ./internal/selfupdate/ -count=1` → Build-Fehler (`ModeBeta`, `view`, `Options.Pin`), dann mit Stubs `--- FAIL: TestRunModes`.

- [ ] **Step 3: Umsetzen**

`version.go`:

```go
// pick is the highest release the channel takes, by version rather than by
// publication. The stable channel takes no pre-release; the beta channel
// takes every release, and the count before 1.0.0 ranks below all others.
func pick(releases []Release, beta bool) (Release, bool) {
	var best Release
	var bestVersion version
	found := false
	for _, r := range releases {
		if !beta && r.Prerelease {
			continue
		}
		v, ok := releaseVersion(r)
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

`gh.go`: `latest(ctx, run, beta bool)` mit `pick(releases, beta)` und `fmt.Errorf("no release in channel %s", map[bool]string{true: "beta", false: "stable"}[beta])` (als kleine Funktion `channelName(beta bool) string`), dazu:

```go
// view is the one release a pinned pass asks for.
func view(ctx context.Context, run Runner, ver string) (Release, error) {
	out, err := call(ctx, run, "gh", "release", "view", "v"+ver, "--repo", Repo, "--json", "tagName,isPrerelease")
	if err != nil {
		return Release{}, fmt.Errorf("no release %s: %w", ver, err)
	}
	var r Release
	if err := json.Unmarshal(out, &r); err != nil {
		return Release{}, fmt.Errorf("parse gh release view: %w", err)
	}
	if _, ok := releaseVersion(r); !ok {
		return Release{}, fmt.Errorf("no release %s: %s is no version", ver, r.Tag)
	}
	return r, nil
}
```

`call` (`gh.go:57`) reicht den Fehler des Runners unverändert durch, außer bei fehlendem `gh` und Zeitüberschreitung; darum lautet der Text im Test `no release 9.9.9: release not found`.

`fake_test.go`: Feld `view string`, im `switch key` ein `case "view": return []byte(f.view), nil`.

`update.go`:

```go
// The modes of a pass by hand. A pass in the machine's channel is the
// default and serve's only mode.
const (
	ModeChannel = ""
	ModeBeta    = "beta"
	ModeStable  = "stable"
)
```

`Options` bekommt `Mode string` und `Pin string` (Kommentar: „Mode or Pin, set by upgrade; Pin is a version without the v“). `installLocked` wird zu:

```go
func installLocked(ctx context.Context, o Options, canonical string, byRunning bool) Result {
	handle, held, err := lock.TryAcquire(filepath.Join(o.StateDir, "update.lock"))
	if err != nil {
		return Result{Outcome: Failed, Err: err}
	}
	if !held {
		return Result{Outcome: Busy, Err: errors.New("update in progress")}
	}
	defer handle.Release()

	running := Reported(o.Version, o.Channel)
	rel, forced, markerErr, err := choose(ctx, o, running)
	if err != nil {
		return Result{Outcome: Failed, Err: err}
	}
	res := place(ctx, o, canonical, byRunning, running, rel, forced)
	if res.Outcome == Current || res.Outcome == Updated {
		markerErr = errors.Join(markerErr, settle(o, rel))
	}
	if markerErr != nil {
		res.Err = errors.Join(res.Err, markerErr)
	}
	return res
}

// choose is the release a pass aims at. A pin or --stable aims at one
// release even when it is older than what runs (forced); --beta and the
// machine's channel aim at the newest one they take. A marker that cannot
// be read leaves the machine on stable, and its error travels on.
func choose(ctx context.Context, o Options, running string) (rel Release, forced bool, markerErr, err error) {
	switch {
	case o.Pin != "":
		rel, err = view(ctx, o.Run, o.Pin)
		return rel, true, nil, err
	case o.Mode == ModeStable:
		rel, err = latest(ctx, o.Run, false)
		return rel, true, nil, err
	case o.Mode == ModeBeta:
		rel, err = latest(ctx, o.Run, true)
		return rel, false, nil, err
	}
	beta, markerErr := ReadChannel(o.StateDir)
	if v, ok := parseReported(running); ok && v.old {
		beta = true
	}
	rel, err = latest(ctx, o.Run, beta)
	return rel, false, markerErr, err
}

// settle leaves the marker as the pass asked: --beta sets it, --stable
// clears it, a pin follows the kind of release it pinned, and a pass in the
// channel leaves it alone.
func settle(o Options, rel Release) error {
	switch {
	case o.Pin != "":
		v, _ := releaseVersion(rel)
		return WriteChannel(o.StateDir, v.beta > 0)
	case o.Mode == ModeBeta:
		return WriteChannel(o.StateDir, true)
	case o.Mode == ModeStable:
		return WriteChannel(o.StateDir, false)
	}
	return nil
}

// place puts rel at canonical unless what runs or what is there already
// is it (forced) or is at least as new (otherwise).
func place(ctx context.Context, o Options, canonical string, byRunning bool, running string, rel Release, forced bool) Result {
	ver := strings.TrimPrefix(rel.Tag, "v")
	done := func(have string) bool {
		if forced {
			h, ok := parseReported(have)
			r, _ := releaseVersion(rel)
			return ok && h.num == r.num && h.beta == r.beta
		}
		return !Newer(rel, have)
	}
	if byRunning && done(running) {
		return Result{Outcome: Current, Version: o.Version}
	}
	// A serve that installed the release keeps running the version before it
	// until a bridge replaces it; asked only the running version, every pass
	// until then would install the same release again.
	if have, ok := InstalledVersion(ctx, o.Run, canonical); ok && done(have) {
		return Result{Outcome: Current, Version: strings.TrimSuffix(have, " (beta)")}
	}
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

`Result.Version` bleibt ohne Kanal (`2.7.0`), wie `upgrade` es mit `v` davor druckt; deshalb `TrimSuffix(have, " (beta)")` beim installierten Binary. Der Fall `forced` vergleicht nur Nummer und Beta, nicht die Zählung: ein `--version 7.1.0` gibt es nach dem Neustart nicht mehr, und vorher meint dieselbe Nummer dasselbe Release.

- [ ] **Step 3b: Sitzungsstart warnt bei einem Problem mit der Markierung.** `updateWarnings` (`internal/hooks/hook_session_start.go:184`) warnt heute nur bei `Result == Failed`. Ein Lauf mit Ergebnis `current` oder `updated`, der trotzdem einen `error` trägt (fremde oder nicht schreibbare Markierung), bliebe sonst unsichtbar. Die Maschine stünde still auf stabil. Zwei Zeilen in die Tabelle von `TestUpdateWarnings` (`internal/hooks/hook_session_start_test.go:639ff.`):

```go
		{"a pass that kept the binary but not the channel", "windows", false, func(dir string) *selfupdate.Status {
			return &selfupdate.Status{Source: selfupdate.SourceServe, Executable: selfupdate.Canonical(dir), Result: selfupdate.Current, CheckedAt: at, Error: `channel file holds "nightly", not beta`}
		}, []string{`loomux update channel at 2026-09-24T08:00:00Z: channel file holds "nightly", not beta; this machine takes stable releases until upgrade --beta or --stable sets it`}},
		{"a skipped pass says why without a warning", "windows", false, func(dir string) *selfupdate.Status {
			return &selfupdate.Status{Source: selfupdate.SourceCLI, Executable: `C:\repo\bin\loomux.exe`, Result: selfupdate.Skipped, Error: "running from C:\\repo"}
		}, nil},
```

Umsetzung direkt unter dem `Failed`-Zweig:

```go
	if (st.Result == selfupdate.Current || st.Result == selfupdate.Updated) && st.Error != "" {
		lines = append(lines, fmt.Sprintf("loomux update channel at %s: %s; this machine takes stable releases until upgrade --beta or --stable sets it",
			st.CheckedAt.UTC().Format(time.RFC3339), st.Error))
	}
```

Die Spec-Zeile zur unlesbaren Markierung ergänzt der Implementierer nicht, das ist schon geschehen (siehe Spec „Kanal einer Maschine“, letzter Punkt, nachgetragen mit diesem Plan).

- [ ] **Step 4: Grün** — `go test ./internal/selfupdate/ ./internal/cli/ ./internal/hooks/ -count=1` → PASS; Coverage `go test -coverprofile` je Funktion 100 %.

- [ ] **Step 5: Mutationsrunde** über `pick`, `choose`, `settle`, `place.done`: `!beta && r.Prerelease` → `r.Prerelease`; `v.old` → `false`; `forced`-Zweige vertauscht; `settle`-Fälle einzeln auf `nil`.

- [ ] **Step 6: Commit** — `feat(selfupdate): take betas on a marked machine and pass a mode or a pinned version`

### Task 4: `loomux upgrade --beta|--stable|--version` und `init`

**Files:**
- Modify: `internal/cli/upgrade.go`
- Test: `internal/cli/upgrade_test.go`
- Modify: `docs/en/cli-reference.md` und `docs/de/cli-reference.md` (Abschnitt `loomux upgrade`), `README.md`/`README.de.md` nur wenn sie `upgrade` beschreiben (`grep -n "loomux upgrade" README*.md`)

**Interfaces:**
- Consumes: `selfupdate.ModeBeta`, `ModeStable`, `Options.Mode`, `Options.Pin` (Task 3)

- [ ] **Step 1: Tests** (neben `TestUpgradeCommand`)

```go
func TestUpgradePassesTheModeAndThePin(t *testing.T) {
	for _, c := range []struct {
		args      []string
		mode, pin string
	}{
		{nil, selfupdate.ModeChannel, ""},
		{[]string{"--beta"}, selfupdate.ModeBeta, ""},
		{[]string{"--stable"}, selfupdate.ModeStable, ""},
		{[]string{"--version", "1.1.0-beta.2"}, selfupdate.ModeChannel, "1.1.0-beta.2"},
		{[]string{"--version", "v1.0.0"}, selfupdate.ModeChannel, "1.0.0"},
	} {
		var got selfupdate.Options
		selfUpdateRun = func(_ context.Context, o selfupdate.Options) selfupdate.Result {
			got = o
			return selfupdate.Result{Outcome: selfupdate.Current, Version: "1.0.0"}
		}
		t.Cleanup(func() { selfUpdateRun = selfupdate.Run })
		var out, errs bytes.Buffer
		if code := upgradeCommand(c.args, nil, &out, &errs); code != 0 {
			t.Fatalf("%v: exit %d, %s", c.args, code, errs.String())
		}
		if got.Mode != c.mode || got.Pin != c.pin {
			t.Errorf("%v: mode %q pin %q, want %q %q", c.args, got.Mode, got.Pin, c.mode, c.pin)
		}
	}
}

func TestUpgradeRefusesTwoChoices(t *testing.T) {
	calls := fakeSelfUpdate(t, selfupdate.Result{Outcome: selfupdate.Current})
	for _, args := range [][]string{
		{"--beta", "--stable"},
		{"--beta", "--version", "1.0.0"},
		{"--stable", "--version", "1.0.0"},
		{"--version", "nightly"},
		{"--version", "1.0.0 (beta)"},
		{"--version"},
		{"extra"},
	} {
		var out, errs bytes.Buffer
		if code := upgradeCommand(args, nil, &out, &errs); code != 2 || !strings.Contains(errs.String(), "usage: loomux upgrade") {
			t.Errorf("%v: exit %d, %q", args, code, errs.String())
		}
	}
	if *calls != 0 {
		t.Fatalf("a refused call ran a pass")
	}
}

func TestUpgradeShowsAMarkerProblemWithoutFailing(t *testing.T) {
	fakeSelfUpdate(t, selfupdate.Result{Outcome: selfupdate.Updated, Version: "1.2.0-beta.1", Err: errors.New("channel: access denied")})
	var out, errs bytes.Buffer
	if code := upgradeCommand([]string{"--beta"}, nil, &out, &errs); code != 0 || !strings.Contains(errs.String(), "channel: access denied") {
		t.Fatalf("exit %d, %q", code, errs.String())
	}
}
```

- [ ] **Step 2: Rot** — `go test ./internal/cli/ -run TestUpgrade -count=1` → `--- FAIL` (heute ist jedes Argument ein Usage-Fehler).

- [ ] **Step 3: Umsetzen**

```go
const upgradeUsage = "usage: loomux upgrade [--beta | --stable | --version <x.y.z>]"

// upgradeCommand is `loomux upgrade`: one pass by hand, the same one serve
// runs daily, or one aimed at the newest beta, the newest stable release or
// a single version.
func upgradeCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	beta := fs.Bool("beta", false, "")
	stable := fs.Bool("stable", false, "")
	pin := fs.String("version", "", "")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || choices(*beta, *stable, *pin != "") > 1 ||
		(*pin != "" && (strings.Contains(*pin, " ") || !selfupdate.IsVersion(strings.TrimPrefix(*pin, "v")))) {
		fmt.Fprintln(stderr, upgradeUsage)
		return 2
	}
	o := selfUpdateOptions(selfupdate.SourceCLI)
	o.Pin = strings.TrimPrefix(*pin, "v")
	switch {
	case *beta:
		o.Mode = selfupdate.ModeBeta
	case *stable:
		o.Mode = selfupdate.ModeStable
	}
	res := selfUpdateRun(context.Background(), o)
	if res.StatusErr != nil {
		fmt.Fprintf(stderr, "loomux upgrade: record update.json: %v\n", res.StatusErr)
	}
	switch res.Outcome {
	case selfupdate.Current:
		reportMarker(stderr, res)
		fmt.Fprintf(stdout, "already current (v%s)\n", res.Version)
		return 0
	case selfupdate.Updated:
		reportMarker(stderr, res)
		fmt.Fprintf(stdout, "updated to v%s; serve switches on the next bridge\n", res.Version)
		return 0
	case selfupdate.Skipped:
		fmt.Fprintf(stderr, "loomux upgrade: skipped: %v\n", res.Err)
		return 2
	}
	fmt.Fprintf(stderr, "loomux upgrade: %v\n", res.Err)
	return 1
}

// choices counts the flags that each pick the release.
func choices(set ...bool) int {
	n := 0
	for _, s := range set {
		if s {
			n++
		}
	}
	return n
}

// reportMarker names a problem with the channel marker of a pass that
// otherwise succeeded; the binary is in place, only the channel is not.
func reportMarker(stderr io.Writer, res selfupdate.Result) {
	if res.Err != nil {
		fmt.Fprintf(stderr, "loomux upgrade: %v\n", res.Err)
	}
}
```

`--version` als Flag von `upgrade` kollidiert nicht mit dem globalen `loomux --version`: `cli.Run` fängt es nur als erstes Argument ab (`cli.go:45`), `upgradeCommand` bekommt die Argumente nach `upgrade`.

Doku `cli-reference` (beide Sprachen, gleicher Inhalt; englisch ab `docs/en/cli-reference.md:885`, dort wo `<state dir>/update.json` beschrieben ist, die Datei `<state dir>/channel` daneben nennen): Aufrufzeile mit den drei Flags; Schritt 1 „im Kanal der Maschine: mit Markierung `<Zustandsverzeichnis>/channel` Betas und stabile Releases, sonst nur stabile; ein Binary aus der Zählung vor 1.0.0 gilt als markiert“; je Flag ein Satz wie in der Spec-Tabelle; „ein Downgrade per `--version` hält nicht, `serve` hebt binnen 24 Stunden wieder an“; Exit 2 auch für zwei Flags zugleich und eine Version, die keine ist.

- [ ] **Step 4: Grün** — `go test ./internal/cli/ -count=1` → PASS.

- [ ] **Step 5: Commit** — `feat(upgrade): choose betas, stable releases or one version by hand`

### Task 5: `loomux dev release next-beta`

**Files:**
- Modify: `internal/release/version.go` (neu `NextBeta`), `internal/cli/release.go` (Unterbefehl)
- Test: `internal/release/version_test.go`, `internal/cli/release_test.go`
- Modify: `docs/en/cli-reference.md:1148-1151` und die deutsche Entsprechung (Überschrift `dev release <next-version|next-beta|…>`, ein Punkt `next-beta --bump … [--tags <file>]`: druckt `X.Y.Z-beta.N`, N eins über der höchsten Beta derselben Version)

**Interfaces:**
- Consumes: `NextVersion(tags []string, bump string) (string, error)`
- Produces: `func NextBeta(tags []string, bump string) (string, error)`

- [ ] **Step 1: Tests**

```go
func TestNextBeta(t *testing.T) {
	for _, c := range []struct {
		tags []string
		bump string
		want string
	}{
		{nil, "minor", "1.0.0-beta.1"},
		{[]string{"v1.0.0"}, "minor", "1.1.0-beta.1"},
		{[]string{"v1.0.0", "v1.1.0-beta.1", "v1.1.0-beta.2"}, "minor", "1.1.0-beta.3"},
		{[]string{"v1.0.0", "v1.1.0-beta.9", "v1.1.0-beta.10"}, "minor", "1.1.0-beta.11"},
		{[]string{"v1.0.0", "v1.0.1-beta.4"}, "minor", "1.1.0-beta.1"},
		{[]string{"v1.0.0", "v1.1.0-beta.02", "v1.1.0-rc.5", "archive/parity-recordings"}, "minor", "1.1.0-beta.1"},
	} {
		got, err := NextBeta(c.tags, c.bump)
		if err != nil || got != c.want {
			t.Errorf("NextBeta(%v, %s) = %q, %v; want %q", c.tags, c.bump, got, err, c.want)
		}
	}
	if _, err := NextBeta(nil, "huge"); !errors.Is(err, ErrBump) {
		t.Fatalf("err = %v", err)
	}
}
```

`internal/cli/release_test.go`, im Stil des `next-version`-Tests daneben (`runIn`, `run`):

```go
func TestDevReleaseNextBeta(t *testing.T) {
	if code, out, _ := runIn("v1.0.0\nv1.1.0-beta.1\n", "dev", "release", "next-beta", "--bump", "minor"); code != 0 || out != "1.1.0-beta.2\n" {
		t.Fatalf("next-beta = %d %q", code, out)
	}
	if code, _, errs := run("dev", "release", "next-beta", "--bump", "none"); code != 2 || !strings.Contains(errs, "loomux dev release next-beta: bump must be major, minor or patch") {
		t.Fatalf("bad bump = %d %q", code, errs)
	}
	if code, _, _ := run("dev", "release", "next-beta", "--bump", "patch", "--tags", filepath.Join(t.TempDir(), "missing")); code != 2 {
		t.Fatalf("missing tag file = %d", code)
	}
	if code, _, _ := run("dev", "release", "next-beta", "--bogus"); code != 2 {
		t.Fatalf("unknown flag = %d", code)
	}
}
```

- [ ] **Step 2: Rot** — `go test ./internal/release/ ./internal/cli/ -run 'NextBeta|next-beta' -count=1` → `undefined: NextBeta`.

- [ ] **Step 3: Umsetzen**

```go
// betaPattern matches a beta tag of the new count: vX.Y.Z-beta.N, N without
// a leading zero. Compiled on first use like tagPattern.
var betaPattern = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^v((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))-beta\.([1-9][0-9]*)$`)
})

// NextBeta is the next beta towards the release NextVersion would cut: its
// version with -beta.N, N one past the highest beta of that version.
func NextBeta(tags []string, bump string) (string, error) {
	base, err := NextVersion(tags, bump)
	if err != nil {
		return "", err
	}
	n := 0
	for _, tag := range tags {
		m := betaPattern().FindStringSubmatch(tag)
		if m == nil || m[1] != base {
			continue
		}
		if k, err := strconv.Atoi(m[2]); err == nil && k > n {
			n = k
		}
	}
	return fmt.Sprintf("%s-beta.%d", base, n+1), nil
}
```

`internal/cli/release.go`: `"next-beta": devReleaseNextBeta`, eine Kopie von `devReleaseNextVersion` mit `release.NextBeta` und dem Namen `dev release next-beta` in FlagSet und Meldung. Doppelt ist hier billiger als eine Hülle über zwei Zeilen Unterschied; wer sie zusammenlegen will, macht `devReleaseNext(name string, next func([]string, string) (string, error)) command`.

- [ ] **Step 4: Grün** — `go test ./internal/release/ ./internal/cli/ -count=1` → PASS; Mutationsrunde (`m[1] != base` weg, `k > n` → `k >= 0`, `n+1` → `n`).

- [ ] **Step 5: Commit** — `feat(release): name the next beta of a version`

### Task 6: Plan, README, PR

**Files:**
- Modify: `docs/en/migration.md`, `docs/de/migration.md` (Zeile 4f, Spalte „was bleibt“: die Brücke ist gebaut; Status bleibt „offen“, Priorität wie in der Spec)
- Modify: `README.md`, `README.de.md` (Abschnitt Releases: der Kanal einer Maschine und die drei `upgrade`-Flags; `RELEASE_CHANNEL` bleibt bis zum Neustart beschrieben)

- [ ] **Step 1:** Zeile 4f in beiden Plänen ergänzen: „Die Brücke für den Neustart ist gebaut (`upgrade --beta|--stable|--version`, Kanal-Markierung, Rangfolge beider Zählungen); der Neustart selbst folgt nach PR C (Nachtrag #31).“ Danach `go test ./internal/plancheck/ -count=1` → PASS.
- [ ] **Step 2:** README beider Sprachen nachziehen, `grep -n "upgrade" README*.md` als Anker.
- [ ] **Step 3:** Commit `docs: describe the beta channel marker and the upgrade flags`.
- [ ] **Step 4:** Dem Menschen `gh variable get RELEASE_CHANNEL` nennen; erwartet `beta`, sonst vor dem Merge auf `beta` setzen lassen. Dann `release-pr`-Skill: Commits gruppieren, Label `release:minor`, Changelog-Block (Added: `loomux upgrade --beta`, `--stable`, `--version`; Changed: a machine is on the beta channel while `<state>/channel` holds `beta`, and a release of the count before 1.0.0 ranks below every later one), Push-Befehl für den Menschen nennen.
