# Versionierung, Releases und CI — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Jeder gemergte PR nach `master` erzeugt über GitHub Actions eine
SemVer-Version samt `CHANGELOG.md`-Eintrag, Tag, Binaries und Pre-release; jeder
PR und Push durchläuft Gate und CLI-Smoke-Test auf Windows und Linux.

**Architecture:** Die Regeln (nächste Version, PR-Rumpf prüfen, Changelog
einfügen, Cross-Build) liegen plattformneutral in `internal/release` und werden
über `loomux dev release <sub>` gerufen. `ci/gate.sh` und `ci/smoke.sh` sind
plattformneutrale Shell. `.github/` enthält nur Auslöser, Rechte, Token und die
GitHub-API-Aufrufe.

**Tech Stack:** Go (Standardbibliothek: `regexp`, `strconv`, `crypto/sha256`,
`encoding/json`, `os/exec`), POSIX-sh, Bash, GitHub Actions, `gh`, `jq`.
**Keine neue Go-Abhängigkeit.**

**Spec:** `docs/.superpowers/specs/2026-09-18-github-releases-design.md`. Bei
Widerspruch gilt die Spec.

## Global Constraints

- Coverage 100 % pro Funktion; Ausnahme nur mit `//coverage:exempt <grund>`
  direkt über `func`.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst.
- Code, Kommentare, Fehlermeldungen, Commit-Nachrichten englisch; Prosa in
  `docs/.superpowers/` deutsch.
- Exit-Codes der `dev`-Befehle: 0 ok, 1 Befund, 2 Aufruffehler.
- Commits mit dem Menschen als Autor, ohne Nennung eines Modells; mehrzeilige
  Nachrichten über Datei und `git commit -F`. Vor jedem Commit Zweig und HEAD
  lesen. Kein Agent pusht.
- Tags `vX.Y.Z`; erste Version `1.0.0`; `RELEASE_CHANNEL` `beta` | `stable`.
- Label-Namen exakt: `release:major`, `release:minor`, `release:patch`,
  `release:none`.
- Erlaubte Changelog-Überschriften exakt: `Added`, `Changed`, `Deprecated`,
  `Removed`, `Fixed`, `Security`.
- Build-Ziele exakt: `windows/amd64`, `linux/amd64`, `linux/arm64`,
  `darwin/amd64`, `darwin/arm64`; Name `loomux_<v>_<os>_<arch>[.exe]`.
- Actions: vor dem Eintragen die jeweils neueste Hauptversion auf GitHub
  nachsehen (`gh release view -R actions/checkout` usw.) und diese pinnen; die
  Versionen unten sind der Stand beim Schreiben.
- Das Gate `.githooks/pre-commit` läuft bei jedem Commit; es muss grün sein.

## Dateien

| Datei | Verantwortung |
|---|---|
| `internal/cli/cli.go` | `Channel`, Versionsausgabe |
| `internal/release/version.go` | `NextVersion` |
| `internal/release/body.go` | `ParseBody` |
| `internal/release/changelog.go` | `InsertChangelog` |
| `internal/release/build.go` | `Build`, `Targets` |
| `internal/cli/release.go` | `loomux dev release …` |
| `ci/gate.sh`, `ci/smoke.sh` | Gate und Smoke-Test |
| `.githooks/pre-commit` | ruft `ci/gate.sh` |
| `.github/workflows/{ci,pr-label,release}.yml` | GitHub-Auslöser |
| `.github/scripts/release.sh` | Release-Ablauf gegen die GitHub-API |
| `AGENTS.md`, `README.md`, `README.de.md` | Regeln und Einrichtung |

---

### Task 1: Kanal in der Versionsausgabe

**Files:**
- Modify: `internal/cli/cli.go:12-30`
- Test: `internal/cli/cli_test.go`

**Interfaces:**
- Produces: `var Channel string` in `internal/cli`, gesetzt per
  `-X github.com/xidus90/loomux/internal/cli.Channel=<c>`;
  `func versionLine() string`.

- [ ] **Step 1: Failing test** — in `cli_test.go` den Test
  `TestVersionPrintsTheVersion` ersetzen:

```go
func TestVersionPrintsTheVersion(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-v"} {
		code, out, _ := run(arg)
		if code != 0 || out != "loomux 0.0.0-dev\n" {
			t.Fatalf("%s: code %d, out %q", arg, code, out)
		}
	}
}

func TestVersionNamesAChannelOtherThanStable(t *testing.T) {
	defer func(v, c string) { Version, Channel = v, c }(Version, Channel)
	for channel, want := range map[string]string{
		"":       "loomux 1.2.3\n",
		"stable": "loomux 1.2.3\n",
		"beta":   "loomux 1.2.3 (beta)\n",
	} {
		Version, Channel = "1.2.3", channel
		if _, out, _ := run("version"); out != want {
			t.Fatalf("channel %q: out %q, want %q", channel, out, want)
		}
	}
}
```

- [ ] **Step 2:** `go test ./internal/cli -run TestVersion` → FAIL, `undefined: Channel`.

- [ ] **Step 3: Implementierung** in `cli.go`: Kommentar und Variablen ersetzen,
  den `version`-Zweig auf `versionLine()` umstellen.

```go
// Version and Channel are what `loomux version` answers. A plain `go build`
// leaves them at 0.0.0-dev and empty; the release build sets both with
// -ldflags "-X github.com/xidus90/loomux/internal/cli.Version=… -X …Channel=…".
var (
	Version = "0.0.0-dev"
	Channel = ""
)

// versionLine names the channel only while it is not the stable one, so a
// stable release reads like any plain version string.
func versionLine() string {
	if Channel == "" || Channel == "stable" {
		return "loomux " + Version
	}
	return "loomux " + Version + " (" + Channel + ")"
}
```

```go
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, versionLine())
		return 0
```

- [ ] **Step 4:** `go test ./internal/cli -run TestVersion` → PASS.
- [ ] **Step 5: Commit** `Name the release channel in loomux version`.

---

### Task 2: `release.NextVersion`

**Files:**
- Create: `internal/release/version.go`, `internal/release/version_test.go`

**Interfaces:**
- Produces: `func NextVersion(tags []string, bump string) (string, error)` —
  `bump` ∈ `major|minor|patch`, sonst `ErrBump`; Rückgabe ohne `v`.
  `var ErrBump = errors.New("bump must be major, minor or patch")`.

- [ ] **Step 1: Failing test**

```go
package release

import (
	"errors"
	"testing"
)

func TestNextVersion(t *testing.T) {
	tags := []string{"v1.2.3", "v1.10.0", "v1.10.0", "v1.9.9", "v2.0.0-rc.1", "latest", "v01.0.0", "g2a-before-rebase"}
	for bump, want := range map[string]string{"major": "2.0.0", "minor": "1.11.0", "patch": "1.10.1"} {
		got, err := NextVersion(tags, bump)
		if err != nil || got != want {
			t.Fatalf("%s: got %q, %v; want %q", bump, got, err, want)
		}
	}
}

func TestNextVersionStartsAtOneWithoutATag(t *testing.T) {
	for _, bump := range []string{"major", "minor", "patch"} {
		if got, err := NextVersion([]string{"nightly", ""}, bump); err != nil || got != "1.0.0" {
			t.Fatalf("%s: got %q, %v", bump, got, err)
		}
	}
}

func TestNextVersionRefusesAnUnknownBump(t *testing.T) {
	if _, err := NextVersion(nil, "none"); !errors.Is(err, ErrBump) {
		t.Fatalf("err %v", err)
	}
}
```

- [ ] **Step 2:** `go test ./internal/release` → FAIL (Paket fehlt).

- [ ] **Step 3: Implementierung**

```go
// Package release holds the platform-neutral rules of a loomux release: the
// next version, the checks on a pull request body, the changelog entry and
// the cross build. CI glue for a particular forge calls it through
// `loomux dev release`.
package release

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

// ErrBump names a bump that is not one of the three SemVer positions.
var ErrBump = errors.New("bump must be major, minor or patch")

// tagPattern accepts only plain release tags; leading zeros and pre-release
// suffixes are no releases of this scheme and must not win the maximum.
var tagPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// NextVersion returns the version after the highest release tag. Without any
// release tag the first version is 1.0.0, whatever the bump says.
func NextVersion(tags []string, bump string) (string, error) {
	if bump != "major" && bump != "minor" && bump != "patch" {
		return "", fmt.Errorf("%w, got %q", ErrBump, bump)
	}
	var best [3]int
	found := false
	for _, tag := range tags {
		m := tagPattern.FindStringSubmatch(tag)
		if m == nil {
			continue
		}
		var v [3]int
		for i := range v {
			v[i], _ = strconv.Atoi(m[i+1])
		}
		if !found || less(best, v) {
			best, found = v, true
		}
	}
	if !found {
		return "1.0.0", nil
	}
	switch bump {
	case "major":
		best = [3]int{best[0] + 1, 0, 0}
	case "minor":
		best = [3]int{best[0], best[1] + 1, 0}
	default:
		best[2]++
	}
	return fmt.Sprintf("%d.%d.%d", best[0], best[1], best[2]), nil
}

func less(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}
```

  `strconv.Atoi` kann nach dem Regex nur bei Überlauf scheitern (mehr als 19
  Ziffern); der Fehler wird bewusst verworfen. Der doppelte Tag `v1.10.0` im
  Test deckt den Gleichstand in `less` ab.

- [ ] **Step 4:** `go test ./internal/release -cover` → PASS, 100 %.
- [ ] **Step 5: Commit** `Compute the next release version from tags`.

---

### Task 3: `release.ParseBody`

**Files:**
- Create: `internal/release/body.go`, `internal/release/body_test.go`

**Interfaces:**
- Produces:
  `type Parsed struct { Bump string \`json:"bump"\`; Changelog string \`json:"changelog"\` }`
  und `func ParseBody(labels []string, body string) (Parsed, []string)` — die
  zweite Rückgabe sind Befunde, leer heißt gültig. `Bump` ist `major`,
  `minor`, `patch` oder `none`. `Changelog` beginnt mit der ersten
  `###`-Zeile und endet mit genau einem `\n`; bei `none` leer.

- [ ] **Step 1: Failing test**

```go
package release

import (
	"reflect"
	"testing"
)

const goodBody = "Release: minor — adds export\r\n\r\n## Changelog\r\n### Added\r\n- `loomux graph export`\r\n\r\n### Fixed\r\n- a crash\r\n\r\n## Notes\r\nnot part of it\r\n"

func TestParseBodyTakesTheChangelogBlock(t *testing.T) {
	got, problems := ParseBody([]string{"bug", "release:minor"}, goodBody)
	want := Parsed{Bump: "minor", Changelog: "### Added\n- `loomux graph export`\n\n### Fixed\n- a crash\n"}
	if len(problems) != 0 || got != want {
		t.Fatalf("got %#v, %v", got, problems)
	}
}

func TestParseBodyNoneNeedsNoChangelog(t *testing.T) {
	got, problems := ParseBody([]string{"release:none"}, "")
	if len(problems) != 0 || got != (Parsed{Bump: "none"}) {
		t.Fatalf("got %#v, %v", got, problems)
	}
}

func TestParseBodyFindings(t *testing.T) {
	for name, tc := range map[string]struct {
		labels []string
		body   string
		want   []string
	}{
		"no label":      {nil, goodBody, []string{"exactly one release:* label is required, found 0"}},
		"two labels":    {[]string{"release:minor", "release:patch"}, goodBody, []string{"exactly one release:* label is required, found 2"}},
		"unknown label": {[]string{"release:huge"}, goodBody, []string{`unknown release label "release:huge"`}},
		"no block":      {[]string{"release:patch"}, "text", []string{"the body has no \"## Changelog\" section"}},
		"empty block":   {[]string{"release:patch"}, "## Changelog\n### Fixed\n", []string{"the changelog has no entry"}},
		"bad heading":   {[]string{"release:patch"}, "## Changelog\n### Misc\n- x\n", []string{`changelog heading "### Misc" is not one of Added, Changed, Deprecated, Removed, Fixed, Security`}},
		"orphan entry":  {[]string{"release:patch"}, "## Changelog\n- x\n", []string{"changelog entry \"- x\" stands before any ### heading", "the changelog has no entry"}},
	} {
		if _, got := ParseBody(tc.labels, tc.body); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}
```

- [ ] **Step 2:** `go test ./internal/release -run ParseBody` → FAIL.

- [ ] **Step 3: Implementierung**

```go
package release

import (
	"fmt"
	"strings"
)

// Parsed is what a pull request says about its release.
type Parsed struct {
	Bump      string `json:"bump"`
	Changelog string `json:"changelog"`
}

var bumps = map[string]string{
	"release:major": "major",
	"release:minor": "minor",
	"release:patch": "patch",
	"release:none":  "none",
}

var headings = map[string]bool{
	"Added": true, "Changed": true, "Deprecated": true,
	"Removed": true, "Fixed": true, "Security": true,
}

// ParseBody checks labels and body of a pull request and returns the bump
// and the changelog block. The findings are complete rather than first-only,
// so an author fixes the body in one round.
func ParseBody(labels []string, body string) (Parsed, []string) {
	var found []string
	for _, l := range labels {
		if strings.HasPrefix(l, "release:") {
			found = append(found, l)
		}
	}
	if len(found) != 1 {
		return Parsed{}, []string{fmt.Sprintf("exactly one release:* label is required, found %d", len(found))}
	}
	bump, ok := bumps[found[0]]
	if !ok {
		return Parsed{}, []string{fmt.Sprintf("unknown release label %q", found[0])}
	}
	if bump == "none" {
		return Parsed{Bump: bump}, nil
	}
	block, ok := changelogBlock(body)
	if !ok {
		return Parsed{}, []string{`the body has no "## Changelog" section`}
	}
	var problems []string
	entries, inSection := 0, false
	for _, line := range block {
		switch {
		case strings.HasPrefix(line, "### "):
			if !headings[strings.TrimSpace(line[4:])] {
				problems = append(problems, fmt.Sprintf("changelog heading %q is not one of Added, Changed, Deprecated, Removed, Fixed, Security", line))
			}
			inSection = true
		case strings.HasPrefix(line, "- "):
			if !inSection {
				problems = append(problems, fmt.Sprintf("changelog entry %q stands before any ### heading", line))
				continue
			}
			entries++
		}
	}
	if entries == 0 {
		problems = append(problems, "the changelog has no entry")
	}
	if len(problems) > 0 {
		return Parsed{}, problems
	}
	return Parsed{Bump: bump, Changelog: strings.TrimSpace(strings.Join(block, "\n")) + "\n"}, nil
}

// changelogBlock returns the lines after "## Changelog" up to the next
// second-level heading. GitHub stores bodies with CRLF when they were typed
// in the browser, so line ends are normalised first.
func changelogBlock(body string) ([]string, bool) {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "## Changelog" {
			continue
		}
		rest := lines[i+1:]
		for j, l := range rest {
			if strings.HasPrefix(l, "## ") {
				return rest[:j], true
			}
		}
		return rest, true
	}
	return nil, false
}
```

- [ ] **Step 4:** `go test ./internal/release -cover` → PASS, 100 %.
- [ ] **Step 5: Commit** `Check the release label and changelog of a pull request`.

---

### Task 4: `release.InsertChangelog`

**Files:**
- Create: `internal/release/changelog.go`, `internal/release/changelog_test.go`

**Interfaces:**
- Produces:
  `func InsertChangelog(existing []byte, version, date, link, notes string) ([]byte, error)`
  — `existing == nil` heißt: Datei fehlt. `var ErrDuplicate` für eine schon
  eingetragene Version. `const ChangelogHeader`.

- [ ] **Step 1: Failing test**

```go
package release

import (
	"errors"
	"testing"
)

func TestInsertChangelogCreatesTheFile(t *testing.T) {
	got, err := InsertChangelog(nil, "1.0.0", "2026-09-18", "https://x/pull/1", "### Added\n- a\n")
	want := ChangelogHeader + "\n## [1.0.0] - 2026-09-18\n\n<https://x/pull/1>\n\n### Added\n- a\n"
	if err != nil || string(got) != want {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestInsertChangelogPutsTheNewestFirst(t *testing.T) {
	old := ChangelogHeader + "\n## [1.0.0] - 2026-09-18\n\n<l1>\n\n### Added\n- a\n"
	got, err := InsertChangelog([]byte(old), "1.0.1", "2026-09-19", "l2", "### Fixed\n- b\n")
	want := ChangelogHeader + "\n## [1.0.1] - 2026-09-19\n\n<l2>\n\n### Fixed\n- b\n" +
		"\n## [1.0.0] - 2026-09-18\n\n<l1>\n\n### Added\n- a\n"
	if err != nil || string(got) != want {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestInsertChangelogAppendsBelowAHeaderWithoutEntries(t *testing.T) {
	got, err := InsertChangelog([]byte("# Changelog\n"), "1.0.0", "d", "l", "### Added\n- a\n")
	if err != nil || string(got) != "# Changelog\n\n## [1.0.0] - d\n\n<l>\n\n### Added\n- a\n" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestInsertChangelogRefusesADuplicate(t *testing.T) {
	old := ChangelogHeader + "\n## [1.0.0] - 2026-09-18\n"
	if _, err := InsertChangelog([]byte(old), "1.0.0", "d", "l", "n"); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("err %v", err)
	}
}
```

- [ ] **Step 2:** `go test ./internal/release -run InsertChangelog` → FAIL.

- [ ] **Step 3: Implementierung**

```go
package release

import (
	"errors"
	"fmt"
	"strings"
)

// ChangelogHeader opens a CHANGELOG.md that the first release creates.
const ChangelogHeader = "# Changelog\n\nAll notable changes to loomux are listed here, newest first. The format\nfollows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions\nfollow [Semantic Versioning](https://semver.org/).\n"

// ErrDuplicate refuses a version the file already names, so a repeated
// release run cannot write an entry twice.
var ErrDuplicate = errors.New("version already in changelog")

// InsertChangelog puts the entry of one release above all older entries,
// directly below the header. nil stands for a missing file.
func InsertChangelog(existing []byte, version, date, link, notes string) ([]byte, error) {
	text := string(existing)
	if existing == nil {
		text = ChangelogHeader
	}
	if strings.Contains(text, "\n## ["+version+"]") {
		return nil, fmt.Errorf("%w: %s", ErrDuplicate, version)
	}
	entry := fmt.Sprintf("\n## [%s] - %s\n\n<%s>\n\n%s", version, date, link, strings.TrimRight(notes, "\n")+"\n")
	if i := strings.Index(text, "\n## ["); i >= 0 {
		return []byte(text[:i+1] + entry[1:] + "\n" + text[i+1:]), nil
	}
	return []byte(strings.TrimRight(text, "\n") + "\n" + entry), nil
}
```

- [ ] **Step 4:** `go test ./internal/release -cover` → PASS, 100 %.
- [ ] **Step 5: Commit** `Insert a release entry into CHANGELOG.md`.

---

### Task 5: `release.Build`

**Files:**
- Create: `internal/release/build.go`, `internal/release/build_test.go`

**Interfaces:**
- Produces:
  `type Target struct{ OS, Arch string }`, `var Targets = []Target{…}` (fünf
  Ziele in der Reihenfolge der Constraints),
  `type GoBuild func(env []string, args ...string) error`,
  `func Build(version, channel, out string, run GoBuild) ([]string, error)` —
  liefert die geschriebenen Dateinamen (ohne Pfad) inkl. `SHA256SUMS`,
  `func ExecGoBuild(env []string, args ...string) error` (echter Aufruf).

- [ ] **Step 1: Failing test**

```go
package release

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGo writes the -o target with content naming its GOOS/GOARCH, so the
// checksums differ per file.
func fakeGo(calls *[][]string) GoBuild {
	return func(env []string, args ...string) error {
		*calls = append(*calls, append(append([]string{}, env...), args...))
		for i, a := range args {
			if a == "-o" {
				return os.WriteFile(args[i+1], []byte(strings.Join(env, " ")), 0o755)
			}
		}
		return errors.New("no -o")
	}
}

func TestBuildWritesAllTargetsAndSums(t *testing.T) {
	out := t.TempDir()
	var calls [][]string
	names, err := Build("1.2.3", "beta", out, fakeGo(&calls))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"loomux_1.2.3_windows_amd64.exe", "loomux_1.2.3_linux_amd64", "loomux_1.2.3_linux_arm64",
		"loomux_1.2.3_darwin_amd64", "loomux_1.2.3_darwin_arm64", "SHA256SUMS",
	}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("names %v", names)
	}
	first := strings.Join(calls[0], " ")
	for _, part := range []string{"CGO_ENABLED=0", "GOOS=windows", "GOARCH=amd64", "-trimpath",
		"-X github.com/xidus90/loomux/internal/cli.Version=1.2.3 -X github.com/xidus90/loomux/internal/cli.Channel=beta",
		"./cmd/loomux"} {
		if !strings.Contains(first, part) {
			t.Fatalf("call %q lacks %q", first, part)
		}
	}
	sums, _ := os.ReadFile(filepath.Join(out, "SHA256SUMS"))
	data, _ := os.ReadFile(filepath.Join(out, want[0]))
	sum := sha256.Sum256(data)
	if !strings.HasPrefix(string(sums), hex.EncodeToString(sum[:])+"  "+want[0]+"\n") {
		t.Fatalf("sums %q", sums)
	}
}

func TestBuildStopsAtTheFirstFailure(t *testing.T) {
	fail := func([]string, ...string) error { return errors.New("boom") }
	if _, err := Build("1.0.0", "", t.TempDir(), fail); err == nil || !strings.Contains(err.Error(), "windows/amd64: boom") {
		t.Fatalf("err %v", err)
	}
}

func TestBuildReportsAnUnwritableOutput(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	os.WriteFile(file, nil, 0o644)
	var calls [][]string
	if _, err := Build("1.0.0", "", file, fakeGo(&calls)); err == nil {
		t.Fatal("want error for an output that is a file")
	}
}

func TestBuildReportsAMissingBinary(t *testing.T) {
	noop := func([]string, ...string) error { return nil }
	if _, err := Build("1.0.0", "", t.TempDir(), noop); err == nil {
		t.Fatal("want error when go build wrote nothing")
	}
}

func TestBuildReportsUnwritableSums(t *testing.T) {
	out := t.TempDir()
	var calls [][]string
	inner := fakeGo(&calls)
	blockSums := func(env []string, args ...string) error {
		if len(calls) == len(Targets)-1 {
			os.Mkdir(filepath.Join(out, "SHA256SUMS"), 0o755)
		}
		return inner(env, args...)
	}
	if _, err := Build("1.0.0", "", out, blockSums); err == nil {
		t.Fatal("want error when SHA256SUMS cannot be written")
	}
}

func TestExecGoBuildRunsGo(t *testing.T) {
	if err := ExecGoBuild(nil, "version"); err != nil {
		t.Fatal(err)
	}
	if err := ExecGoBuild(nil, "no-such-subcommand"); err == nil {
		t.Fatal("want error")
	}
}
```

- [ ] **Step 2:** `go test ./internal/release -run Build` → FAIL.

- [ ] **Step 3: Implementierung**

```go
package release

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Target is one platform a release ships a binary for.
type Target struct{ OS, Arch string }

// Targets are the platforms of every release, in the order of its assets.
var Targets = []Target{
	{"windows", "amd64"}, {"linux", "amd64"}, {"linux", "arm64"},
	{"darwin", "amd64"}, {"darwin", "arm64"},
}

// GoBuild runs `go` with extra environment; tests replace it.
type GoBuild func(env []string, args ...string) error

// ExecGoBuild runs the real go tool and keeps its output in the error.
func ExecGoBuild(env []string, args ...string) error {
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Build cross-compiles every target into out and writes SHA256SUMS next to
// them. It returns the file names in asset order.
func Build(version, channel, out string, run GoBuild) ([]string, error) {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return nil, err
	}
	ldflags := fmt.Sprintf("-X github.com/xidus90/loomux/internal/cli.Version=%s -X github.com/xidus90/loomux/internal/cli.Channel=%s", version, channel)
	var names []string
	var sums strings.Builder
	for _, t := range Targets {
		name := fmt.Sprintf("loomux_%s_%s_%s", version, t.OS, t.Arch)
		if t.OS == "windows" {
			name += ".exe"
		}
		path := filepath.Join(out, name)
		env := []string{"CGO_ENABLED=0", "GOOS=" + t.OS, "GOARCH=" + t.Arch}
		if err := run(env, "build", "-trimpath", "-ldflags", ldflags, "-o", path, "./cmd/loomux"); err != nil {
			return nil, fmt.Errorf("build %s/%s: %w", t.OS, t.Arch, err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		fmt.Fprintf(&sums, "%s  %s\n", hex.EncodeToString(sum[:]), name)
		names = append(names, name)
	}
	if err := os.WriteFile(filepath.Join(out, "SHA256SUMS"), []byte(sums.String()), 0o644); err != nil {
		return nil, err
	}
	return append(names, "SHA256SUMS"), nil
}
```

- [ ] **Step 4:** `go test ./internal/release -cover` → PASS, 100 %.
- [ ] **Step 5: Commit** `Cross-build release binaries with checksums`.

---

### Task 6: `loomux dev release`

**Files:**
- Create: `internal/cli/release.go`, `internal/cli/release_test.go`
- Modify: `internal/cli/dev.go` (Eintrag `"release": devRelease` in `devCommands`)

**Interfaces:**
- Consumes: `release.NextVersion`, `release.ParseBody`, `release.Parsed`,
  `release.InsertChangelog`, `release.ErrDuplicate`, `release.Build`,
  `release.ExecGoBuild`.
- Produces: CLI
  - `loomux dev release next-version --bump <b> [--tags <datei>|-]` → stdout `X.Y.Z\n`
  - `loomux dev release parse-body --labels <csv> [--body <datei>|-]` → stdout JSON + `\n`; Befunde auf stderr, Exit 1
  - `loomux dev release changelog-insert --version <v> --date <d> --link <l> [--file CHANGELOG.md] [--notes <datei>|-]` → Exit 1 bei `ErrDuplicate`
  - `loomux dev release build --version <v> [--channel <c>] --out <verz>` → stdout ein Dateiname pro Zeile
  - Paketvariable `var releaseGo release.GoBuild = release.ExecGoBuild` für Tests.

- [ ] **Step 1: Failing tests** in `release_test.go`:

```go
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/release"
)

func runIn(stdin string, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Run(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func TestDevReleaseNeedsAKnownSubcommand(t *testing.T) {
	if code, _, e := run("dev", "release"); code != 2 || !strings.Contains(e, "subcommand required") {
		t.Fatalf("code %d, err %q", code, e)
	}
	if code, _, e := run("dev", "release", "nope"); code != 2 || !strings.Contains(e, `unknown subcommand "nope"`) {
		t.Fatalf("code %d, err %q", code, e)
	}
}

func TestDevReleaseNextVersion(t *testing.T) {
	if code, out, _ := runIn("v1.0.0\nv1.1.0\n", "dev", "release", "next-version", "--bump", "patch"); code != 0 || out != "1.1.1\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
	file := filepath.Join(t.TempDir(), "tags")
	os.WriteFile(file, []byte("v3.0.0\n"), 0o644)
	if code, out, _ := run("dev", "release", "next-version", "--bump", "major", "--tags", file); code != 0 || out != "4.0.0\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
	if code, _, _ := run("dev", "release", "next-version", "--bump", "none"); code != 2 {
		t.Fatalf("bad bump: code %d", code)
	}
	if code, _, _ := run("dev", "release", "next-version", "--bump", "patch", "--tags", filepath.Join(t.TempDir(), "missing")); code != 2 {
		t.Fatalf("missing file: code %d", code)
	}
	if code, _, _ := run("dev", "release", "next-version", "--bogus"); code != 2 {
		t.Fatalf("bad flag: code %d", code)
	}
}

func TestDevReleaseParseBody(t *testing.T) {
	body := "## Changelog\n### Fixed\n- x\n"
	code, out, _ := runIn(body, "dev", "release", "parse-body", "--labels", "release:patch")
	if code != 0 || out != `{"bump":"patch","changelog":"### Fixed\n- x\n"}`+"\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
	code, _, e := runIn("", "dev", "release", "parse-body", "--labels", "")
	if code != 1 || !strings.Contains(e, "found 0") {
		t.Fatalf("code %d, err %q", code, e)
	}
	if code, _, _ := run("dev", "release", "parse-body", "--body", filepath.Join(t.TempDir(), "missing")); code != 2 {
		t.Fatalf("missing file: code %d", code)
	}
	if code, _, _ := run("dev", "release", "parse-body", "--bogus"); code != 2 {
		t.Fatalf("bad flag: code %d", code)
	}
}

func TestDevReleaseChangelogInsert(t *testing.T) {
	file := filepath.Join(t.TempDir(), "CHANGELOG.md")
	args := []string{"dev", "release", "changelog-insert", "--version", "1.0.0", "--date", "2026-09-18", "--link", "l", "--file", file}
	if code, _, e := runIn("### Added\n- a\n", args...); code != 0 {
		t.Fatalf("code %d: %s", code, e)
	}
	got, _ := os.ReadFile(file)
	if !strings.Contains(string(got), "## [1.0.0] - 2026-09-18") {
		t.Fatalf("file %q", got)
	}
	if code, _, _ := runIn("### Added\n- a\n", args...); code != 1 {
		t.Fatalf("duplicate: code %d", code)
	}
	if code, _, _ := run("dev", "release", "changelog-insert", "--version", "1.0.0"); code != 2 {
		t.Fatalf("missing flags: code %d", code)
	}
	if code, _, _ := run("dev", "release", "changelog-insert", "--bogus"); code != 2 {
		t.Fatalf("bad flag: code %d", code)
	}
	dir := t.TempDir()
	if code, _, _ := runIn("n", "dev", "release", "changelog-insert", "--version", "1", "--date", "d", "--link", "l", "--file", dir); code != 2 {
		t.Fatalf("directory as file: code %d", code)
	}
	if code, _, _ := run("dev", "release", "changelog-insert", "--version", "1", "--date", "d", "--link", "l", "--file", file, "--notes", filepath.Join(dir, "missing")); code != 2 {
		t.Fatalf("missing notes: code %d", code)
	}
}

func TestDevReleaseBuild(t *testing.T) {
	defer func(g release.GoBuild) { releaseGo = g }(releaseGo)
	releaseGo = func(env []string, args ...string) error {
		for i, a := range args {
			if a == "-o" {
				return os.WriteFile(args[i+1], []byte("bin"), 0o755)
			}
		}
		return nil
	}
	out := t.TempDir()
	code, stdout, e := run("dev", "release", "build", "--version", "1.0.0", "--channel", "beta", "--out", out)
	if code != 0 || !strings.HasSuffix(stdout, "SHA256SUMS\n") {
		t.Fatalf("code %d, out %q, err %q", code, stdout, e)
	}
	if code, _, _ := run("dev", "release", "build", "--out", out); code != 2 {
		t.Fatalf("missing version: code %d", code)
	}
	if code, _, _ := run("dev", "release", "build", "--bogus"); code != 2 {
		t.Fatalf("bad flag: code %d", code)
	}
	releaseGo = func([]string, ...string) error { return os.ErrPermission }
	if code, _, _ := run("dev", "release", "build", "--version", "1.0.0", "--out", out); code != 1 {
		t.Fatalf("failed build: code %d", code)
	}
}
```

- [ ] **Step 2:** `go test ./internal/cli -run DevRelease` → FAIL.

- [ ] **Step 3: Implementierung** `internal/cli/release.go`:

```go
package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xidus90/loomux/internal/release"
)

var releaseGo release.GoBuild = release.ExecGoBuild

var releaseCommands = map[string]command{
	"build":            devReleaseBuild,
	"changelog-insert": devReleaseChangelog,
	"next-version":     devReleaseNextVersion,
	"parse-body":       devReleaseParseBody,
}

func devRelease(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "loomux dev release: subcommand required")
		return 2
	}
	sub, ok := releaseCommands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "loomux dev release: unknown subcommand %q\n", args[0])
		return 2
	}
	return sub(args[1:], stdin, stdout, stderr)
}

// readInput reads a named file, or stdin for "-".
func readInput(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(path)
}

func devReleaseNextVersion(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev release next-version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	bump := fs.String("bump", "", "major, minor or patch")
	tags := fs.String("tags", "-", "file with one tag per line, - for stdin")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	data, err := readInput(*tags, stdin)
	if err == nil {
		var v string
		if v, err = release.NextVersion(strings.Fields(string(data)), *bump); err == nil {
			fmt.Fprintln(stdout, v)
			return 0
		}
	}
	fmt.Fprintf(stderr, "loomux dev release next-version: %v\n", err)
	return 2
}

func devReleaseParseBody(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev release parse-body", flag.ContinueOnError)
	fs.SetOutput(stderr)
	labels := fs.String("labels", "", "comma-separated labels of the pull request")
	body := fs.String("body", "-", "file with the pull request body, - for stdin")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	data, err := readInput(*body, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev release parse-body: %v\n", err)
		return 2
	}
	parsed, problems := release.ParseBody(strings.FieldsFunc(*labels, func(r rune) bool { return r == ',' }), string(data))
	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintf(stderr, "loomux dev release parse-body: %s\n", p)
		}
		return 1
	}
	out, _ := json.Marshal(parsed)
	fmt.Fprintf(stdout, "%s\n", out)
	return 0
}

func devReleaseChangelog(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev release changelog-insert", flag.ContinueOnError)
	fs.SetOutput(stderr)
	version := fs.String("version", "", "release version without v")
	date := fs.String("date", "", "release date, YYYY-MM-DD")
	link := fs.String("link", "", "URL of the pull request")
	file := fs.String("file", "CHANGELOG.md", "changelog to update")
	notes := fs.String("notes", "-", "file with the changelog block, - for stdin")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *version == "" || *date == "" || *link == "" {
		fmt.Fprintln(stderr, "loomux dev release changelog-insert: --version, --date and --link are required")
		return 2
	}
	block, err := readInput(*notes, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev release changelog-insert: %v\n", err)
		return 2
	}
	existing, err := os.ReadFile(*file)
	if errors.Is(err, os.ErrNotExist) {
		existing, err = nil, nil
	}
	var updated []byte
	if err == nil {
		updated, err = release.InsertChangelog(existing, *version, *date, *link, string(block))
	}
	if err == nil {
		err = os.WriteFile(*file, updated, 0o644)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev release changelog-insert: %v\n", err)
		if errors.Is(err, release.ErrDuplicate) {
			return 1
		}
		return 2
	}
	return 0
}

func devReleaseBuild(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev release build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	version := fs.String("version", "", "release version without v")
	channel := fs.String("channel", "", "release channel, e.g. beta")
	out := fs.String("out", "dist", "output directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *version == "" {
		fmt.Fprintln(stderr, "loomux dev release build: --version is required")
		return 2
	}
	names, err := release.Build(*version, *channel, *out, releaseGo)
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev release build: %v\n", err)
		return 1
	}
	for _, n := range names {
		fmt.Fprintln(stdout, n)
	}
	return 0
}
```

  In `dev.go` in `devCommands` ergänzen: `"release": devRelease,`
  (alphabetisch zwischen `"record-case"` und `"swap-binary"`).

- [ ] **Step 4:** `go test ./internal/cli ./internal/release -cover` → PASS;
  danach das volle Gate per Commit.
- [ ] **Step 5: Commit** `Expose the release rules as loomux dev release`.

---

### Task 7: `ci/gate.sh`, `ci/smoke.sh`, Pre-commit-Hook

**Files:**
- Create: `ci/gate.sh`, `ci/smoke.sh`
- Modify: `.githooks/pre-commit`

**Interfaces:**
- Produces: `sh ci/gate.sh` (Wurzel des Repos als cwd, Exit ≠ 0 bei Befund);
  `sh ci/smoke.sh <binary>`.

- [ ] **Step 1: `ci/smoke.sh` anlegen** (zuerst, er ist der Test für den Build):

```sh
#!/bin/sh
# Smoke test of a built loomux binary: the three answers every build must
# give, whatever platform or forge runs it.
set -u
bin=${1:?usage: ci/smoke.sh <binary>}
fail=0
out=$("$bin" version) || { echo "smoke: version exited $?" >&2; fail=1; }
case $out in
"loomux "*) ;;
*) echo "smoke: version answered '$out'" >&2; fail=1 ;;
esac
"$bin" help >/dev/null || { echo "smoke: help exited $?" >&2; fail=1; }
"$bin" gibtesnicht >/dev/null 2>&1
rc=$?
[ "$rc" -eq 2 ] || { echo "smoke: unknown command exited $rc, want 2" >&2; fail=1; }
exit $fail
```

- [ ] **Step 2: Rot nachweisen.** `sh ci/smoke.sh /bin/false` → Exit 1 mit
  drei Meldungen. `sh ci/smoke.sh bin/loomux.exe` → Exit 0.

- [ ] **Step 3: `ci/gate.sh` anlegen** — die vier Prüfschritte aus
  `.githooks/pre-commit` wortgleich samt ihrer Kommentare übernehmen:

```sh
#!/bin/sh
# The gate of loomux: format, vet, tests with per-function coverage. The
# pre-commit hook and every CI forge run exactly this script.
set -eu
cd "$(git rev-parse --show-toplevel)"
unformatted=$(gofmt -l cmd internal)
if [ -n "$unformatted" ]; then
	echo "gofmt: these files are not formatted:" >&2
	echo "$unformatted" >&2
	exit 1
fi
go vet ./...
# -count=1: a package served from the test cache adds no counts to the
# -coverpkg profile, and covergate then reports its unchanged functions at 0%.
# The pattern names the module, not ./...: a directory pattern also takes in
# third_party/toml, which go.mod replaces the parser with, and that copy is
# not held to our coverage rule.
go test ./... -count=1 -covermode=set -coverpkg=github.com/xidus90/loomux/... -coverprofile=coverage.out
go run ./cmd/loomux dev covergate --profile coverage.out
```

- [ ] **Step 4: `.githooks/pre-commit` umbauen.** Die Index-Prüfung bleibt;
  in ihren beiden `git`-Aufrufen `ci` zur Pfadliste hinzufügen
  (`-- '*.go' go.mod go.sum testdata .githooks ci`). Die Zeilen von
  `unformatted=…` bis `go run ./cmd/loomux dev covergate …` ersetzen durch
  `sh ci/gate.sh`. `mkdir -p bin` und die zwei Build-/Swap-Zeilen bleiben.
  Kopfkommentar anpassen: „…then the shared gate in ci/gate.sh, then rebuild
  the pilot binary the hooks call.“

- [ ] **Step 5: Prüfen.** Eine Datei in `internal/release` testweise
  unformatiert stagen → Commit wird mit `gofmt:` abgelehnt; zurücksetzen.
  Dann regulär committen: Gate läuft grün.
- [ ] **Step 6: Commit** `Share one gate script between the hook and CI`.

---

### Task 8: GitHub-Workflows und Release-Skript

**Files:**
- Create: `.github/workflows/ci.yml`, `.github/workflows/pr-label.yml`,
  `.github/workflows/release.yml`, `.github/scripts/release.sh`

**Interfaces:**
- Consumes: `ci/gate.sh`, `ci/smoke.sh`, alle `loomux dev release`-Befehle.
- Produces: Workflow-Namen `ci`, `pr-label`, `release`; Secrets
  `RELEASE_APP_ID`, `RELEASE_APP_PRIVATE_KEY`; Variable `RELEASE_CHANNEL`.

- [ ] **Step 1: `.github/workflows/ci.yml`**

```yaml
name: ci
on:
  pull_request:
    branches: [master]
  push:
    branches: [master]
    # The release commit touches only the changelog; running ci on it would
    # start a release run that can cancel a pending one of a real merge.
    paths-ignore: [CHANGELOG.md]
permissions:
  contents: read
jobs:
  gate:
    strategy:
      fail-fast: false
      matrix:
        os: [windows-latest, ubuntu-latest]
    runs-on: ${{ matrix.os }}
    defaults:
      run:
        shell: bash
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
      - run: sh ci/gate.sh
      - run: go build -o bin/loomux.exe ./cmd/loomux
      - run: sh ci/smoke.sh bin/loomux.exe
```

  (`loomux.exe` auch auf Linux: der Name ist nur ein Dateiname, und die
  Schritte bleiben so plattformgleich.)

- [ ] **Step 2: `.github/workflows/pr-label.yml`**

```yaml
name: pr-label
on:
  pull_request:
    branches: [master]
    types: [opened, edited, labeled, unlabeled, synchronize, reopened]
permissions:
  contents: read
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      # Built from master, not from the pull request: a fork could otherwise
      # change the rules so that its own body always passes.
      - uses: actions/checkout@v5
        with:
          ref: master
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
      - run: go build -o "$RUNNER_TEMP/loomux" ./cmd/loomux
      - name: Check label and changelog
        env:
          BODY: ${{ github.event.pull_request.body }}
          LABELS: ${{ join(github.event.pull_request.labels.*.name, ',') }}
        run: |
          printf '%s' "$BODY" > "$RUNNER_TEMP/body.md"
          "$RUNNER_TEMP/loomux" dev release parse-body --labels "$LABELS" --body "$RUNNER_TEMP/body.md"
```

- [ ] **Step 3: `.github/scripts/release.sh`**

```bash
#!/usr/bin/env bash
# Release of one merged pull request against the GitHub API. Every decision
# comes from `loomux dev release`; this script only fetches and publishes.
set -euo pipefail
: "${GH_TOKEN:?}" "${REPO:?}" "${LOOMUX:?}"
channel=${CHANNEL:-beta}
work=$RUNNER_TEMP/release
mkdir -p "$work"

if [ -n "${DISPATCH_PR:-}" ]; then
	pr=$DISPATCH_PR
else
	pr=$(gh api "repos/$REPO/commits/$SHA/pulls" \
		--jq '[.[] | select(.merged_at != null and .base.ref == "master")][0].number // empty')
	if [ -z "$pr" ]; then
		echo "no merged pull request for $SHA; nothing to release"
		exit 0
	fi
fi
gh api "repos/$REPO/pulls/$pr" >"$work/pr.json"
jq -r '.body // ""' "$work/pr.json" >"$work/body.md"
labels=$(jq -r '[.labels[].name] | join(",")' "$work/pr.json")
link=$(jq -r .html_url "$work/pr.json")
"$LOOMUX" dev release parse-body --labels "$labels" --body "$work/body.md" >"$work/parsed.json"
bump=$(jq -r .bump "$work/parsed.json")
if [ "$bump" = none ]; then
	echo "pull request #$pr is release:none"
	exit 0
fi
jq -r .changelog "$work/parsed.json" >"$work/notes.md"

# The contents API commits on top of the current master and refuses only when
# CHANGELOG.md itself changed since we read it: then read again, once.
commit=
for attempt in 1 2; do
	git fetch --quiet --force --tags origin master
	git checkout --quiet --force --detach origin/master
	version=$(git tag -l 'v*' | "$LOOMUX" dev release next-version --bump "$bump")
	"$LOOMUX" dev release changelog-insert --version "$version" --date "$(date -u +%F)" \
		--link "$link" --notes "$work/notes.md"
	sha_arg=()
	if old=$(git rev-parse -q --verify origin/master:CHANGELOG.md); then
		sha_arg=(-f "sha=$old")
	fi
	if commit=$(gh api -X PUT "repos/$REPO/contents/CHANGELOG.md" \
		-f message="Release v$version" -f branch=master \
		-f content="$(base64 -w0 CHANGELOG.md)" "${sha_arg[@]}" --jq .commit.sha); then
		break
	fi
	commit=
	echo "attempt $attempt: CHANGELOG.md moved on master" >&2
done
if [ -z "$commit" ]; then
	echo "release of #$pr failed; rerun release.yml by hand with pr=$pr" >&2
	exit 1
fi

git fetch --quiet origin "$commit"
git checkout --quiet --force --detach "$commit"
"$LOOMUX" dev release build --version "$version" --channel "$channel" --out "$work/dist"
pre=()
[ "$channel" = stable ] || pre=(--prerelease)
gh release create "v$version" --target "$commit" --title "v$version" \
	--notes-file "$work/notes.md" "${pre[@]}" "$work"/dist/*
echo "released v$version for #$pr"
```

  `chmod +x` ist unter Windows wirkungslos; stattdessen
  `git update-index --chmod=+x .github/scripts/release.sh ci/gate.sh ci/smoke.sh`
  nach dem `git add`.

- [ ] **Step 4: `.github/workflows/release.yml`**

```yaml
name: release
on:
  workflow_run:
    workflows: [ci]
    types: [completed]
  workflow_dispatch:
    inputs:
      pr:
        description: Number of the merged pull request to release
        required: true
        type: number
permissions:
  contents: read
concurrency:
  group: release
  cancel-in-progress: false
jobs:
  release:
    # event == push: a pull request from a fork branch named master must not
    # pass as a merge.
    if: >-
      github.event_name == 'workflow_dispatch' ||
      (github.event.workflow_run.conclusion == 'success' &&
       github.event.workflow_run.event == 'push' &&
       github.event.workflow_run.head_branch == 'master')
    runs-on: ubuntu-latest
    steps:
      - id: app
        uses: actions/create-github-app-token@v2
        with:
          app-id: ${{ secrets.RELEASE_APP_ID }}
          private-key: ${{ secrets.RELEASE_APP_PRIVATE_KEY }}
      - uses: actions/checkout@v5
        with:
          ref: master
          fetch-depth: 0
          fetch-tags: true
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
      - run: go build -o "$RUNNER_TEMP/loomux" ./cmd/loomux
      - run: bash .github/scripts/release.sh
        env:
          GH_TOKEN: ${{ steps.app.outputs.token }}
          REPO: ${{ github.repository }}
          SHA: ${{ github.event.workflow_run.head_sha }}
          DISPATCH_PR: ${{ inputs.pr }}
          CHANNEL: ${{ vars.RELEASE_CHANNEL }}
          LOOMUX: ${{ runner.temp }}/loomux
```

- [ ] **Step 5: Prüfen.** `actionlint` über die drei Workflows (per
  `go run github.com/rhysd/actionlint/cmd/actionlint@latest`) → keine Befunde.
  `bash -n .github/scripts/release.sh` → kein Syntaxfehler. `shellcheck`, falls
  vorhanden, über alle drei Skripte.
- [ ] **Step 6: Commit** `Run the gate and releases on GitHub Actions`.

---

### Task 9: Regeln und Dokumentation

**Files:**
- Modify: `AGENTS.md`, `README.md`, `README.de.md`
- Modify: `.gitattributes` (falls nötig: `*.sh text eol=lf`)

- [ ] **Step 1: `.gitattributes`** prüfen; fehlt eine LF-Regel für Shell,
  `*.sh text eol=lf` ergänzen — ein CRLF-Skript bricht auf Linux mit
  `$'\r': command not found`.

- [ ] **Step 2: `AGENTS.md`** — unter „Where things live“:

```markdown
- `internal/release` holds the release rules (next version, pull request
  body, changelog, cross build) behind `loomux dev release`. `ci/` holds the
  gate and the smoke test every forge runs. `.github/` holds only what is
  GitHub's: triggers, permissions, tokens and API calls. A second forge gets
  its own directory and calls the same scripts and subcommands.
```

  Unter „Rules“:

```markdown
- Every pull request to `master` carries exactly one label `release:major`
  (breaking change to a command, flag, hook protocol, config format or exit
  code), `release:minor` (new feature, compatible), `release:patch` (bug fix
  or dependency update, compatible) or `release:none` (docs, CI or tests
  only). Whoever opens the pull request reads the diff, picks the label (the
  higher one when in doubt), sets it with `gh pr create --label`, and writes
  into the body a line `Release: <level> — <one-sentence reason>` and, unless
  `release:none`, a `## Changelog` block in Keep a Changelog form, English,
  from the user's point of view, headings only `Added`, `Changed`,
  `Deprecated`, `Removed`, `Fixed`, `Security`. When the pull request
  changes, label and block follow. `loomux dev release parse-body` is the
  check.
- The one exception to the rules on commit authors and pushes: `Release v*`
  commits and `v*` tags made by `.github/workflows/release.yml` through the
  `loomux-release` GitHub App. No agent uses that app.
```

  Die bestehende Regel „Commits carry the user…“ und „Nobody but a human
  pushes.“ bekommen je den Zusatz „(see the release exception above)“. Unter
  „Commands“ den Gate-Punkt ersetzen durch
  `- Gate: \`sh ci/gate.sh\`; \`.githooks/pre-commit\` runs it, then rebuilds the pilot binary.`

- [ ] **Step 3: `README.md`** — Abschnitt „Releases“ vor dem Abschnitt zur
  Lizenz: Download von der Releases-Seite, Prüfung mit `SHA256SUMS`,
  Versionsschema (Tabelle der Labels), Hinweis „Every release is a beta
  pre-release until `RELEASE_CHANNEL` is set to `stable`“, Link auf
  `CHANGELOG.md`. Danach Unterabschnitt „Setting up releases (maintainers)“
  mit den Befehlen aus Task 10, Schritt 1–4, als Liste.

- [ ] **Step 4: `README.de.md`** — derselbe Abschnitt auf Deutsch
  („Releases“, „Releases einrichten (Maintainer)“), inhaltsgleich.

- [ ] **Step 5: Commit** `Document the release rules and setup`.

---

### Task 10: Einrichtung auf GitHub (Mensch) und erster Durchlauf

Kein Agent führt diese Schritte ohne ausdrückliche Zustimmung aus; jeder
ändert das Repo oder das Konto. Der Agent legt sie dem Menschen vor.

- [ ] **Step 1: Labels**

```bash
gh label create release:major --color B60205 --description "Breaking change"
gh label create release:minor --color 0E8A16 --description "New feature, compatible"
gh label create release:patch --color 1D76DB --description "Bug fix or dependency update"
gh label create release:none --color CCCCCC --description "No release"
```

- [ ] **Step 2: Variable** — `gh variable set RELEASE_CHANNEL --body beta`

- [ ] **Step 3: GitHub App** (Weboberfläche, Settings → Developer settings →
  GitHub Apps → New): Name `loomux-release`, Webhook aus, Repository
  permissions `Contents: Read and write`, `Pull requests: Read-only`,
  `Metadata: Read-only`; „Only on this account“. Private Key erzeugen, App nur
  in `xidus90/loomux` installieren. Dann:

```bash
gh secret set RELEASE_APP_ID --body <app-id>
gh secret set RELEASE_APP_PRIVATE_KEY < loomux-release.private-key.pem
```

- [ ] **Step 4: Rulesets** (erst nach dem Umschalten auf öffentlich):
  Ruleset für `master` und eines für Tags `v*`, jeweils mit der App
  `loomux-release` als einzigem Bypass-Akteur.

- [ ] **Step 5: Erster Durchlauf.** Der PR dieses Plans selbst trägt
  `release:none` → `ci` und `pr-label` grün, `release` endet ohne Release.
  Danach ein kleiner PR mit `release:patch` und Changelog-Block → nach dem
  Merge existieren `CHANGELOG.md` mit `## [1.0.0]`, Tag `v1.0.0`, ein
  Pre-release mit fünf Binaries und `SHA256SUMS`; `loomux_1.0.0_windows_amd64.exe version`
  antwortet `loomux 1.0.0 (beta)`.

- [ ] **Step 6:** Ist `ci` auf `ubuntu-latest` rot, weil Tests oder
  Coverage Windows voraussetzen: anhalten und dem Menschen den Befund
  vorlegen. Nicht durch Überspringen oder `coverage:exempt` grün machen.
