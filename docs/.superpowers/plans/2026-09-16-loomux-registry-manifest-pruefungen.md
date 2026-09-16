# Registry- und Manifestprüfungen an einer Stelle — Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux brain` und die Schreibschranke verweigern kaputte Registries und Manifeste über dieselben Funktionen in `internal/config`, mit loomux-eigenen Meldungen und strengen Typen.

**Architecture:** `config.ReadRegistry` dekodiert in `map[string]any` und prüft G1–G14. Die neue Funktion `config.ReadDeclaration` prüft ein Manifest nach M1–M17 und baut daraus `*config.Manifest`. `(*Manifest).InboxLayout` prüft `[layout] inbox`. `privacy.VisibleAreas` und `internal/brain/guard` rufen diese Funktionen. Die eigenen Leser der Schranke und ihre Python-Nachbauten (`truthy`, `pyRepr`) entfallen. `config.ReadManifest` für Hook und Lint bleibt unverändert.

**Tech Stack:** Go (Modul `github.com/xidus90/loomux`), `github.com/BurntSushi/toml` v1.6.0, Windows 11.

**Spec:** `docs/.superpowers/specs/2026-09-16-loomux-registry-manifest-pruefungen-design.md` — Regeln G1–G14 und M1–M17 samt Wortlauten stehen dort. Dieser Plan wiederholt sie als Code und Tests.

## Global Constraints

- Basis: `master` ab `e1b4343`; Zweig `claude/recursing-bartik-b2d7a1`, Worktree `C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/recursing-bartik-b2d7a1`.
- Coverage 100 % pro Funktion; Ausnahme nur mit `//coverage:exempt <reason>` direkt über `func`.
- Kein `init()`; keine Paketvariable, die eingebettete Daten parst.
- Code, Bezeichner, Kommentare, Commit-Nachrichten englisch; Paritätsliste deutsch; Benchmarks in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`.
- Commits: Autor und Committer ist der Nutzer (`git config user.name`/`user.email` des Repos); kein `Co-Authored-By`, kein Modell im Text. Mehrzeilige Nachrichten über eine Datei und `git commit -F`.
- Niemand pusht.
- Subagenten je Task mit `model: "opus"` und `effort: "low"`, beides ausdrücklich gesetzt. Nach jedem Subagenten `git log -1 --format='%an <%ae>'` lesen.
- Vor jedem Commit: `git branch --show-current` und `git log -1 --oneline` lesen.
- Das Commit-Gate ist `.githooks/pre-commit` (gofmt, `go vet`, `go test ./... -count=1 -coverpkg=./...`, `loomux dev covergate`, Neubau von `bin/loomux.exe`). Es läuft bei jedem Commit; kein `--no-verify`.
- `.loomux/config.toml` schreibt kein Agent.
- Kein Test startet Python.
- Meldungsformen (Spec): Zeichenketten mit `%q`, Typen als `string`, `integer`, `float`, `boolean`, `datetime`, `array`, `table`, Positionen `#N` ab 1.

## Dateien

| Datei | Verantwortung |
|---|---|
| `internal/config/tomlvalue.go` (neu) | TOML-Typnamen und die drei Feldleser (`requiredString`, `optionalString`, `optionalBool`) samt Meldungsform |
| `internal/config/registry.go` | `ReadRegistry` streng (G1–G14) |
| `internal/config/declaration.go` (neu) | `ErrNoArea`, `ReadDeclaration` (M1–M17), `InboxLayout` |
| `internal/config/manifest.go` | `Manifest` bekommt `Path` und `LayoutInbox`; `readManifestAmong` verliert `requireScope` |
| `internal/config/legacy.go` | `ReadAreaManifestUntilStage4` über `ReadDeclaration` |
| `internal/brain/privacy/areas.go` | `VisibleAreas` prüft `inbox` jedes Bereichs |
| `internal/brain/guard/registry.go`, `manifest.go`, `guard.go`, `python.go` | rufen `config`, eigene Leser entfallen |
| `testdata/bench/registry-checks.json` (neu) | Messfälle vorher/nachher |
| `docs/.superpowers/parity/registry-manifest-pruefungen.md` (neu), `docs/.superpowers/parity/stufe-1b-1.md` | Parität |
| `docs/en/benchmarks.md`, `docs/de/benchmarks.md`, `README.md`, `README.de.md` | Messung, Stand |

---

### Task 1: Ausgangsmessung festhalten

Vor jeder Codeänderung, damit „vorher“ wirklich der alte Code ist. Kein Commit.

**Files:**
- Create (außerhalb des Repos): `C:/Users/micro/AppData/Local/Temp/loomux-registry-bench/`

**Interfaces:**
- Consumes: nichts
- Produces: `…/loomux-registry-bench/loomux-before.exe`, `…/state` (Kopie von `%LOCALAPPDATA%\loomux`), `…/before-go-bench.txt`

- [ ] **Step 1: Stand prüfen**

Run: `git log -1 --oneline` und `git status --short`
Expected: HEAD ist der Plan-Commit auf `e1b4343`, Arbeitsbaum sauber, noch keine Änderung unter `internal/`.

- [ ] **Step 2: Vorher-Binary bauen und Zustand kopieren**

```bash
B="$TEMP/loomux-registry-bench"; mkdir -p "$B"
go build -o "$B/loomux-before.exe" ./cmd/loomux
cp -r "$LOCALAPPDATA/loomux" "$B/state"
```

- [ ] **Step 3: Go-Benchmarks vorher**

```bash
B="$TEMP/loomux-registry-bench"
LOOMUX_BENCH_REGISTRY="$B/state" LOOMUX_BENCH_TARGET="C:/Users/micro/Documents/#GIT/loomux/README.md" go test ./internal/brain/guard/ -run '^$' -bench DecideAgainstTheRealRegistry -benchtime 50x -benchmem -count 5 > "$B/before-go-bench.txt"
LOOMUX_BENCH_REGISTRY="$B/state" LOOMUX_BENCH_LEGACY="$LOCALAPPDATA/brain" go test ./internal/brain/search/ -run '^$' -bench VisibleAreasOfTheRealRegistry -benchtime 50x -benchmem -count 5 >> "$B/before-go-bench.txt"
date '+%Y-%m-%d %H:%M' >> "$B/before-go-bench.txt"
```

Expected: beide Benchmarks laufen (kein `SKIP`, kein `Fatal`). Steht dort `the registry was not read`, zeigt `LOOMUX_BENCH_TARGET` nicht in einen registrierten Bereich; dann den `path` eines schreibbaren Bereichs aus `$B/state/registry.toml` nehmen und den gewählten Pfad in der Datei notieren.

---

### Task 2: `config.ReadRegistry` streng

**Files:**
- Create: `internal/config/tomlvalue.go`
- Modify: `internal/config/registry.go` (Typen `registryEntry`, `registryFile` und Funktion `ReadRegistry` ersetzen)
- Test: `internal/config/registry_test.go`

**Interfaces:**
- Consumes: `ManifestDir(area Area, stateDir string) string` (besteht in `manifest.go`)
- Produces:
  - `func ReadRegistry(stateDir string) ([]Area, error)` (Signatur unverändert, Verhalten streng)
  - `func tomlType(value any) string`
  - `func found(value any) string`
  - `func requiredString(table map[string]any, key, owner, sep string) (string, error)`
  - `func optionalString(table map[string]any, key, owner, sep string) (string, error)`
  - `func optionalBool(table map[string]any, key, owner, sep string) (bool, error)`

- [ ] **Step 1: Die zwei Tests des Überspringens durch einen Verweigerungstest ersetzen**

In `internal/config/registry_test.go` die Funktionen `TestABrokenAreaDoesNotKillTheRest` und `TestAnEntryWithoutAScopeIsDroppedToo` löschen und dafür einfügen:

```go
func TestARegistryRefusesWhatItCannotUse(t *testing.T) {
	// Every rule of the registry, and every TOML type a refusal can name.
	for name, row := range map[string]struct{ body, want string }{
		"area as a table": {"[area]\nscope = \"x\"\npath = \"/a\"\n",
			"area must be an array of [[area]] tables, found table"},
		"entry not a table": {"area = [1]\n",
			"[[area]] #1 must be a table, found integer"},
		"missing scope": {"[[area]]\npath = \"/a\"\n",
			`[[area]] #1 is missing "scope"`},
		"scope an integer": {"[[area]]\nscope = 3\npath = \"/a\"\n",
			"[[area]] #1: scope must be a non-empty string, found integer"},
		"scope a boolean": {"[[area]]\nscope = true\npath = \"/a\"\n",
			"[[area]] #1: scope must be a non-empty string, found boolean"},
		"empty scope": {"[[area]]\nscope = \"\"\npath = \"/a\"\n",
			`[[area]] #1: scope must be a non-empty string, found ""`},
		"duplicate scope": {"[[area]]\nscope = \"x\"\npath = \"/a\"\n\n[[area]]\nscope = \"x\"\npath = \"/b\"\n",
			`[[area]] #2: duplicate scope "x" (first at #1)`},
		"unusable scope": {"[[area]]\nscope = \"///\"\npath = \"/a\"\n",
			`[[area]] "///": scope has no letter, digit, "_", "." or "-" and cannot name a state directory`},
		"shared state directory": {"[[area]]\nscope = \"a/b\"\npath = \"/a\"\n\n[[area]]\nscope = \"a-b\"\npath = \"/b\"\n",
			`scopes "a/b" and "a-b" share the state directory "a-b"`},
		"missing path": {"[[area]]\nscope = \"x\"\n",
			`[[area]] "x" is missing "path"`},
		"path a datetime": {"[[area]]\nscope = \"x\"\npath = 1979-05-27\n",
			`[[area]] "x": path must be a non-empty string, found datetime`},
		"empty path": {"[[area]]\nscope = \"x\"\npath = \"\"\n",
			`[[area]] "x": path must be a non-empty string, found ""`},
		"wiki an array": {"[[area]]\nscope = \"x\"\npath = \"/a\"\nwiki = [\"/a\"]\n",
			`[[area]] "x": wiki must be a non-empty string, found array`},
		"empty wiki": {"[[area]]\nscope = \"x\"\npath = \"/a\"\nwiki = \"\"\n",
			`[[area]] "x": wiki must be a non-empty string, found ""`},
		"readonly a string": {"[[area]]\nscope = \"x\"\npath = \"/a\"\nreadonly = \"yes\"\n",
			`[[area]] "x": readonly must be a boolean, found string`},
		"signpost an integer": {"[[area]]\nscope = \"x\"\npath = \"/a\"\nsignpost = 1\n",
			`[[area]] "x": signpost must be a boolean, found integer`},
		"shared a float": {"[[area]]\nscope = \"x\"\npath = \"/a\"\nshared = 1.5\n",
			`[[area]] "x": shared must be a boolean, found float`},
		"workspace an integer": {"[[area]]\nscope = \"x\"\npath = \"/a\"\nworkspace = 0\n",
			`[[area]] "x": workspace must be a boolean, found integer`},
		"two signposts": {"[[area]]\nscope = \"x\"\npath = \"/a\"\nsignpost = true\n\n[[area]]\nscope = \"y\"\npath = \"/b\"\nsignpost = true\n",
			`scopes "x" and "y" both declare signpost; only one area may`},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeRegistry(t, dir, row.body)
			areas, err := ReadRegistry(dir)
			want := filepath.Join(dir, "registry.toml") + ": " + row.want
			if err == nil || err.Error() != want {
				t.Fatalf("ReadRegistry = %+v, %v; want %q", areas, err, want)
			}
		})
	}
}

func TestARegistryWithoutAreasAnswersNone(t *testing.T) {
	for _, body := range []string{"", "area = []\n"} {
		dir := t.TempDir()
		writeRegistry(t, dir, body)
		areas, err := ReadRegistry(dir)
		if err != nil || len(areas) != 0 {
			t.Errorf("%q: ReadRegistry = %+v, %v; want no areas", body, areas, err)
		}
	}
}

func TestFalseFlagsAndOneSignpostAreFine(t *testing.T) {
	dir := t.TempDir()
	writeRegistry(t, dir, "[[area]]\nscope = \"x\"\npath = \"/a\"\nreadonly = false\nsignpost = true\n\n"+
		"[[area]]\nscope = \"y\"\npath = \"/b\"\nsignpost = false\n")
	areas, err := ReadRegistry(dir)
	if err != nil || len(areas) != 2 || areas[0].ReadOnly || !areas[0].Signpost || areas[1].Signpost {
		t.Fatalf("ReadRegistry = %+v, %v", areas, err)
	}
}
```

In `TestAMissingRegistryIsAnError` am Ende ergänzen (G1: kein vorangestellter Pfad):

```go
	if n := strings.Count(err.Error(), "registry.toml"); n != 1 {
		t.Errorf("error %q names the file %d times; the read error already names it once", err, n)
	}
```

In `TestABrokenRegistryFileIsAnError` den Kommentarsatz ab „Measured, the same branch catches `[area]` …“ bis zum Ende des Kommentars streichen; `[area]` als Tabelle hat jetzt eine eigene Regel (G3).

- [ ] **Step 2: Test laufen lassen, er muss scheitern**

Run: `go test ./internal/config/ -run 'TestARegistryRefusesWhatItCannotUse|TestARegistryWithoutAreasAnswersNone|TestFalseFlagsAndOneSignpostAreFine|TestAMissingRegistryIsAnError' -count=1`
Expected: FAIL. `TestARegistryRefusesWhatItCannotUse` scheitert in fast allen Unterfällen (heute TOML-Fehler oder kein Fehler), `TestAMissingRegistryIsAnError` zählt den Dateinamen zweimal.

- [ ] **Step 3: `tomlvalue.go` anlegen**

```go
package config

import (
	"fmt"
	"strconv"
	"time"
)

// tomlType names a decoded value by its TOML type, the vocabulary a person
// who wrote the file can find in it. BurntSushi hands every date and time
// back as time.Time, and both []any and []map[string]any are arrays --
// nothing else comes out of a document decoded into a map.
func tomlType(value any) string {
	switch value.(type) {
	case string:
		return "string"
	case int64:
		return "integer"
	case float64:
		return "float"
	case bool:
		return "boolean"
	case time.Time:
		return "datetime"
	case map[string]any:
		return "table"
	}
	return "array"
}

// found names a value that should have been a non-empty string: a string
// is quoted -- the only one that gets here is the empty one or a
// value outside an allowed set -- and anything else is named by its type.
func found(value any) string {
	if text, ok := value.(string); ok {
		return strconv.Quote(text)
	}
	return tomlType(value)
}

// requiredString is optionalString for a key that has to be there. owner
// names the table in the refusal and sep joins it to the key, because the
// registry writes `[[area]] #1: scope` and a manifest `[area] scope`.
func requiredString(table map[string]any, key, owner, sep string) (string, error) {
	if _, present := table[key]; !present {
		return "", fmt.Errorf("%s is missing %q", owner, key)
	}
	return optionalString(table, key, owner, sep)
}

// optionalString answers "" for an absent key and refuses a present one
// that is no string or the empty string.
func optionalString(table map[string]any, key, owner, sep string) (string, error) {
	value, present := table[key]
	if !present {
		return "", nil
	}
	if text, ok := value.(string); ok && text != "" {
		return text, nil
	}
	return "", fmt.Errorf("%s%s%s must be a non-empty string, found %s", owner, sep, key, found(value))
}

// optionalBool answers false for an absent key and refuses a present one
// that is not true or false: a flag spelt "yes" is a typo, not a yes.
func optionalBool(table map[string]any, key, owner, sep string) (bool, error) {
	value, present := table[key]
	if !present {
		return false, nil
	}
	flag, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%s%s%s must be a boolean, found %s", owner, sep, key, tomlType(value))
	}
	return flag, nil
}
```

- [ ] **Step 4: `ReadRegistry` in `registry.go` ersetzen**

Die Typen `registryEntry` und `registryFile` sowie die Funktion `ReadRegistry` samt ihrem Doc-Kommentar löschen und ersetzen durch:

```go
// ReadRegistry reads the areas registered in stateDir and refuses a registry
// it cannot use whole. The write barrier reads the same file through this
// function, so a registry either works for both or for neither.
//
// A broken entry refuses the call instead of being skipped: two entries of
// one scope or one state directory leave nothing that could safely be
// dropped, and a skipped entry would show brain a different registry than
// the barrier sees.
func ReadRegistry(stateDir string) ([]Area, error) {
	path := filepath.Join(stateDir, registryName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	document := map[string]any{}
	if err := toml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("%s: not valid TOML: %w", path, err)
	}
	entries, err := areaEntries(document)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	reading := registryReading{
		stateDir:    stateDir,
		positions:   map[string]int{},
		directories: map[string]string{},
	}
	areas := make([]Area, 0, len(entries))
	for i, entry := range entries {
		area, err := reading.area(i+1, entry)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		areas = append(areas, area)
	}
	return areas, nil
}

// areaEntries is the `area` array. The decoder answers []map[string]any
// when every element is a table and []any otherwise.
func areaEntries(document map[string]any) ([]any, error) {
	switch value := document["area"].(type) {
	case nil:
		return nil, nil
	case []map[string]any:
		entries := make([]any, len(value))
		for i, entry := range value {
			entries[i] = entry
		}
		return entries, nil
	case []any:
		return value, nil
	default:
		return nil, fmt.Errorf("area must be an array of [[area]] tables, found %s", tomlType(value))
	}
}

// registryReading carries what one entry's rules need to know about the
// entries before it.
type registryReading struct {
	stateDir    string
	positions   map[string]int
	directories map[string]string
	signposted  string
}

// area checks one entry in the order of the spec's rules G4 to G14.
func (r *registryReading) area(position int, raw any) (Area, error) {
	entry, ok := raw.(map[string]any)
	if !ok {
		return Area{}, fmt.Errorf("[[area]] #%d must be a table, found %s", position, tomlType(raw))
	}
	numbered := fmt.Sprintf("[[area]] #%d", position)
	scope, err := requiredString(entry, "scope", numbered, ": ")
	if err != nil {
		return Area{}, err
	}
	if first, taken := r.positions[scope]; taken {
		return Area{}, fmt.Errorf("%s: duplicate scope %q (first at #%d)", numbered, scope, first)
	}
	r.positions[scope] = position
	named := fmt.Sprintf("[[area]] %q", scope)
	directory := stateDirOf(r.stateDir, scope)
	if directory == stateDirOf(r.stateDir, "") {
		return Area{}, fmt.Errorf(`%s: scope has no letter, digit, "_", "." or "-" and cannot name a state directory`, named)
	}
	if other, taken := r.directories[directory]; taken {
		return Area{}, fmt.Errorf("scopes %q and %q share the state directory %q", other, scope, filepath.Base(directory))
	}
	r.directories[directory] = scope
	path, err := requiredString(entry, "path", named, ": ")
	if err != nil {
		return Area{}, err
	}
	wiki, err := optionalString(entry, "wiki", named, ": ")
	if err != nil {
		return Area{}, err
	}
	flags := map[string]bool{}
	for _, key := range []string{"readonly", "signpost", "shared", "workspace"} {
		flag, err := optionalBool(entry, key, named, ": ")
		if err != nil {
			return Area{}, err
		}
		flags[key] = flag
	}
	if flags["signpost"] {
		if r.signposted != "" {
			return Area{}, fmt.Errorf("scopes %q and %q both declare signpost; only one area may", r.signposted, scope)
		}
		r.signposted = scope
	}
	return Area{
		Scope:    scope,
		Path:     path,
		WikiPath: wiki,
		ReadOnly: flags["readonly"],
		Signpost: flags["signpost"],
		Shared:   flags["shared"],

		Workspace: flags["workspace"],
	}, nil
}

// stateDirOf is the directory a read-only area of this scope keeps its
// artefacts in; ManifestDir holds the rule that names it.
func stateDirOf(stateDir, scope string) string {
	return ManifestDir(Area{Scope: scope, ReadOnly: true}, stateDir)
}
```

Im Doc-Kommentar von `Area` den Absatz, der mit `// WikiPath is the one optional value.` beginnt, bis zu seiner Leerzeile ersetzen durch:

```go
// WikiPath is the one optional value: an entry without `wiki` registers an
// area with WikiPath "", and an entry with an empty `wiki` is refused, so ""
// always means "names no wiki". A pointer would push that distinction into
// every caller for the one field that has it.
```

- [ ] **Step 5: Tests laufen lassen**

Run: `gofmt -w internal/config && go test ./internal/config/ -count=1`
Expected: PASS.

Run: `go test ./internal/brain/... ./internal/cli/ ./internal/dev/importcases/ -count=1`
Expected: PASS. Die Schranke liest die Registry noch selbst. Die Fallwelten tragen nur Wahrheitswerte und nichtleere Zeichenketten, gezählt am 2026-09-16.

- [ ] **Step 6: Commit**

Nachricht in `$TEMP/loomux-registry-bench/msg-task2.txt`:

```
Refuse a registry entry loomux cannot use

ReadRegistry skipped an entry without scope or path and checked nothing
else, so a duplicate scope or a second signpost worked silently. It now
decodes into a map, checks every entry in order and refuses the whole
file with the entry and the reason, the way the write barrier already
refuses writes on the same defects.
```

```bash
git branch --show-current
git log -1 --oneline
git add internal/config/tomlvalue.go internal/config/registry.go internal/config/registry_test.go
git commit -F "$TEMP/loomux-registry-bench/msg-task2.txt"
```

---

### Task 3: `config.ReadDeclaration` und der Bereichsleser

**Files:**
- Create: `internal/config/declaration.go`, `internal/config/declaration_test.go`
- Modify: `internal/config/manifest.go` (`Manifest`, `readManifestAmong`, `ReadManifest`)
- Modify: `internal/config/legacy.go` (`ReadAreaManifestUntilStage4`)
- Test: `internal/config/legacy_test.go`

**Interfaces:**
- Consumes: `tomlType`, `found`, `requiredString`, `optionalString`, `optionalBool` (Task 2); `DefaultUntouchedDays`, `ErrNoManifest`, `manifestNames` (bestehen)
- Produces:
  - `var ErrNoArea = errors.New("the configuration declares no [area]")`
  - `func ReadDeclaration(path string) (*Manifest, error)`
  - `func (m *Manifest) InboxLayout() (string, error)`
  - `Manifest.Path string`, `Manifest.LayoutInbox string`
  - `func ReadAreaManifestUntilStage4(dir string) (*Manifest, error)` (Signatur unverändert)

- [ ] **Step 1: Tests für `ReadDeclaration` und `InboxLayout` schreiben**

`internal/config/declaration_test.go`:

```go
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// declarationFile writes body to a config.toml in a fresh directory and
// answers its path.
func declarationFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const areaX = "[area]\nscope = \"x\"\n"

func TestADeclarationRefusesWhatItCannotUse(t *testing.T) {
	for name, row := range map[string]struct{ body, want string }{
		"area a string":      {"area = \"x\"\n", "[area] must be a table, found string"},
		"privacy an integer": {"privacy = 5\n" + areaX, "[privacy] must be a table, found integer"},
		"wiki a datetime":    {"wiki = 1979-05-27\n" + areaX, "[wiki] must be a table, found datetime"},
		"maintenance a string": {"maintenance = \"x\"\n" + areaX,
			"[maintenance] must be a table, found string"},
		"model an array":  {"model = [1]\n" + areaX, "[model] must be a table, found array"},
		"layout a boolean": {"layout = true\n" + areaX, "[layout] must be a table, found boolean"},
		"index a float":   {"index = 1.5\n" + areaX, "[index] must be a table, found float"},
		"missing scope":   {"[area]\nname = \"x\"\n", `[area] is missing "scope"`},
		"scope an integer": {"[area]\nscope = 3\n",
			"[area] scope must be a non-empty string, found integer"},
		"empty scope": {"[area]\nscope = \"\"\n", `[area] scope must be a non-empty string, found ""`},
		"scope before mode": {"[area]\nscope = \"\"\n\n[privacy]\nmode = \"cloud\"\n",
			`[area] scope must be a non-empty string, found ""`},
		"unknown mode": {areaX + "\n[privacy]\nmode = \"cloud\"\n",
			`[privacy] mode must be one of automatic_cloud, local_only, manual_cloud, found "cloud"`},
		"mode an integer": {areaX + "\n[privacy]\nmode = 1\n",
			"[privacy] mode must be one of automatic_cloud, local_only, manual_cloud, found integer"},
		"types a string": {areaX + "\n[wiki]\ntypes = \"Topic\"\n",
			"[wiki] types must be an array of strings, found string"},
		"types with an integer": {areaX + "\n[wiki]\ntypes = [\"Topic\", 1]\n",
			"[wiki] types #2 must be a string, found integer"},
		"on_merge a string": {areaX + "\n[maintenance]\non_merge = \"yes\"\n",
			"[maintenance] on_merge must be a boolean, found string"},
		"empty branch": {areaX + "\n[maintenance]\nbranch = \"\"\n",
			`[maintenance] branch must be a non-empty string, found ""`},
		"enabled a string": {areaX + "\n[model]\nenabled = \"yes\"\n",
			"[model] enabled must be a boolean, found string"},
		"roles an integer": {areaX + "\n[model]\nroles = 5\n",
			"[model] roles must be a table, found integer"},
		"unknown roles": {areaX + "\n[model.roles]\nzap = false\nplace = true\nguess = true\n",
			`[model] roles has unknown "guess", "zap"; known are describe, place, propose`},
		"role a string": {areaX + "\n[model.roles]\nplace = \"yes\"\n",
			"[model] roles.place must be a boolean, found string"},
		"hub an integer": {areaX + "\n[layout]\nhub = 1\n",
			"[layout] hub must be a string, found integer"},
		"include a string": {areaX + "\n[index]\ninclude = \"*.md\"\n",
			"[index] include must be an array of strings, found string"},
		"exclude with an integer": {areaX + "\n[index]\nexclude = [\"a\", 3]\n",
			"[index] exclude #2 must be a string, found integer"},
		"never a string": {areaX + "\n[privacy]\nnever = \"x\"\n",
			"[privacy] never must be an array of strings, found string"},
		"untouched_days zero": {areaX + "\n[wiki]\nuntouched_days = 0\n",
			"[wiki] untouched_days must be an integer >= 1, found 0"},
		"untouched_days true": {areaX + "\n[wiki]\nuntouched_days = true\n",
			"[wiki] untouched_days must be an integer >= 1, found boolean"},
		"untouched_days a string": {areaX + "\n[wiki]\nuntouched_days = \"5\"\n",
			"[wiki] untouched_days must be an integer >= 1, found string"},
	} {
		t.Run(name, func(t *testing.T) {
			path := declarationFile(t, row.body)
			manifest, err := ReadDeclaration(path)
			want := path + ": " + row.want
			if err == nil || err.Error() != want {
				t.Fatalf("ReadDeclaration = %+v, %v; want %q", manifest, err, want)
			}
		})
	}
}

func TestADeclarationCarriesWhatItDeclares(t *testing.T) {
	path := declarationFile(t, "[area]\nscope = \"project/demo\"\n\n"+
		"[privacy]\nmode = \"local_only\"\nnever = [\"secret/**\"]\n\n"+
		"[wiki]\ntypes = [\"Runbook\"]\nuntouched_days = 30\n\n"+
		"[maintenance]\non_merge = true\nbranch = \"main\"\n\n"+
		"[model]\nenabled = false\n\n[model.roles]\nplace = true\n\n"+
		"[layout]\nwiki = \"docs/wiki\"\nhub = \"hub\"\nreview = \"95\"\ninbox = \"00 In\"\n\n"+
		"[index]\ninclude = [\"**/*.md\"]\nexclude = [\"drafts/**\"]\nunsearched = [\"archive/**\"]\n")
	m, err := ReadDeclaration(path)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join([]string{m.Path, m.Scope, m.PrivacyMode, strings.Join(m.NeverGlobs, ","),
		strings.Join(m.DeclaredTypes, ","), m.LayoutWiki, m.LayoutHub, m.LayoutReview, m.LayoutInbox,
		strings.Join(m.IndexInclude, ","), strings.Join(m.IndexExclude, ","), strings.Join(m.IndexUnsearched, ",")}, "|")
	want := path + "|project/demo|local_only|secret/**|Runbook|docs/wiki|hub|95|00 In|**/*.md|drafts/**|archive/**"
	if got != want || m.UntouchedDays != 30 {
		t.Errorf("got %q and %d days, want %q and 30", got, m.UntouchedDays, want)
	}
}

func TestADeclarationThatSaysNothingElseGetsTheDefaults(t *testing.T) {
	m, err := ReadDeclaration(declarationFile(t, areaX))
	if err != nil {
		t.Fatal(err)
	}
	if m.PrivacyMode != "manual_cloud" || m.UntouchedDays != DefaultUntouchedDays ||
		m.NeverGlobs != nil || m.DeclaredTypes != nil || m.LayoutInbox != "" {
		t.Errorf("ReadDeclaration = %+v", m)
	}
}

func TestAConfigurationWithoutAnAreaIsErrNoArea(t *testing.T) {
	path := declarationFile(t, "[layout]\nwiki = 1\n")
	_, err := ReadDeclaration(path)
	if !errors.Is(err, ErrNoArea) || err.Error() != path+": the configuration declares no [area]" {
		t.Fatalf("err = %v, want ErrNoArea naming the file", err)
	}
}

func TestADeclarationThatCannotBeReadIsNoAbsence(t *testing.T) {
	// The file that vanishes between a stat and the read: an error, and
	// neither ErrNoArea nor anything a caller could take for "no manifest".
	path := filepath.Join(t.TempDir(), "gone.toml")
	_, err := ReadDeclaration(path)
	if errors.Is(err, ErrNoArea) || !errors.Is(err, fs.ErrNotExist) ||
		!strings.HasPrefix(err.Error(), path+": cannot be read: ") {
		t.Fatalf("err = %v", err)
	}
}

func TestADeclarationThatIsNoTomlNamesTheFile(t *testing.T) {
	path := declarationFile(t, "[area\n")
	_, err := ReadDeclaration(path)
	if err == nil || !strings.HasPrefix(err.Error(), path+": not valid TOML: ") {
		t.Fatalf("err = %v", err)
	}
}

func TestAnInboxMustStayInsideTheArea(t *testing.T) {
	outside := t.TempDir()
	m := &Manifest{Path: "p/config.toml", LayoutInbox: outside}
	if _, err := m.InboxLayout(); err == nil ||
		err.Error() != "p/config.toml: [layout] inbox must be relative to the area, found "+strconv.Quote(outside) {
		t.Errorf("InboxLayout = %v", err)
	}
	for _, value := range []string{"", "00 In"} {
		m := &Manifest{LayoutInbox: value}
		if got, err := m.InboxLayout(); err != nil || got != value {
			t.Errorf("InboxLayout(%q) = %q, %v", value, got, err)
		}
	}
}
```

In `internal/config/legacy_test.go` die erwarteten Meldungen anpassen:

`TestALegacyManifestWithoutAScopeIsRefused` ersetzt ihre Schleife durch:

```go
	for name, row := range map[string]struct{ body, want string }{
		"no area table":  {"[privacy]\nmode = \"local_only\"\n", `[area] is missing "scope"`},
		"no scope key":   {"[area]\nname = \"x\"\n", `[area] is missing "scope"`},
		"empty scope":    {"[area]\nscope = \"\"\n", `[area] scope must be a non-empty string, found ""`},
		"and a bad mode": {"[privacy]\nmode = \"bogus\"\n", `[area] is missing "scope"`},
	} {
		dir := t.TempDir()
		legacyWrite(t, dir, ".brain.toml", row.body)
		_, err := ReadAreaManifestUntilStage4(dir)
		want := filepath.Join(dir, ".brain.toml") + ": " + row.want
		if err == nil || err.Error() != want {
			t.Errorf("%s: err = %v, want %q", name, err, want)
		}
	}
```

Ihr Kommentar lautet dann: `// An old name has no policy-only form, so a file without a scope is refused, and the scope is asked before the mode.`

`TestAnAreaTableWithoutAScopeInTheLoomuxManifestIsRefused` ersetzt ihre Schleife durch:

```go
	for name, row := range map[string]struct{ body, want string }{
		"empty table": {"[area]\n", `[area] is missing "scope"`},
		"empty scope": {"[area]\nscope = \"\"\n", `[area] scope must be a non-empty string, found ""`},
	} {
		dir := t.TempDir()
		legacyWrite(t, dir, filepath.Join(".loomux", "config.toml"), row.body)
		legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \"brain\"\n")
		_, err := ReadAreaManifestUntilStage4(dir)
		want := filepath.Join(dir, ".loomux", "config.toml") + ": " + row.want
		if err == nil || err.Error() != want {
			t.Errorf("%s: err = %v, want %q", name, err, want)
		}
	}
```

Und neu:

```go
func TestALegacyManifestIsCheckedWhole(t *testing.T) {
	dir := t.TempDir()
	legacyWrite(t, dir, ".brain.toml", "[area]\nscope = \"k\"\n\n[model]\nenabled = \"yes\"\n")
	_, err := ReadAreaManifestUntilStage4(dir)
	want := filepath.Join(dir, ".brain.toml") + ": [model] enabled must be a boolean, found string"
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
}
```

- [ ] **Step 2: Tests laufen lassen, sie müssen scheitern**

Run: `go test ./internal/config/ -count=1`
Expected: FAIL beim Kompilieren (`ReadDeclaration`, `ErrNoArea`, `InboxLayout`, `Manifest.Path`, `Manifest.LayoutInbox` undefiniert).

- [ ] **Step 3: `Manifest` erweitern und `readManifestAmong` vereinfachen**

In `internal/config/manifest.go`:

1. In `type Manifest struct` als erstes Feld `Path string` einfügen und nach `LayoutReview` das Feld `LayoutInbox string` einfügen. Über `Path` kommt der Kommentar `// Path is the file the declaration was read from; refusals name it.`
2. `ReadManifest` ruft `readManifestAmong(repoRoot, manifestNames)`.
3. `readManifestAmong` verliert den Parameter `requireScope` und den ganzen Block `if requireScope { … }`. Im zurückgegebenen `&Manifest{…}` kommt `Path: path,` hinzu. Aus ihrem Doc-Kommentar die Sätze über `ReadAreaManifestUntilStage4`, `requireScope` und `read_manifest` entfernen. Der Kommentar lautet dann: `// readManifestAmong reads the first of names below dir that is a regular file, for ReadManifest. It decodes into the typed wire shape and checks only the privacy mode: the lint and the post-edit hook read a declaration as "none" on any error, and a stricter reader would switch their wiki lane off silently. The write barrier refuses a broken area declaration visibly.`
4. Prüfen: `grep -n "readManifestAmong" internal/config/*.go` zeigt nur noch die Definition und den Aufruf in `ReadManifest`.
5. Den Paketkommentar am Kopf von `manifest.go` (`// Package config reads the manifest in which an area declares itself.` bis `package config`) ersetzen durch:

```go
// Package config reads the registry and the manifest in which an area
// declares itself.
//
// There are two manifest readers, and the difference is deliberate.
// ReadDeclaration checks a declaration whole and refuses it; brain and the
// write barrier read through it. ReadManifest supplies the few values the
// lint and the post-edit hook need and checks almost nothing, because both
// read any error as "no manifest" and a stricter reader would switch their
// wiki lane off without a word.
```

- [ ] **Step 4: `declaration.go` anlegen**

```go
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// ErrNoArea says that a configuration carries no [area] table. A project that
// uses only loomux's policy declares no brain area, and every caller reads
// this as "no declaration here" (stage 1a, R7a).
var ErrNoArea = errors.New("the configuration declares no [area]")

// ReadDeclaration reads one area declaration and refuses it whole where a
// value loomux reads, or one the Python reference refused, has the wrong type
// or an unusable value. Unknown keys are not judged: real manifests carry
// keys of tools that are not loomux. The rules and their order are M1 to M17
// of docs/.superpowers/specs/2026-09-16-loomux-registry-manifest-pruefungen-design.md.
func ReadDeclaration(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: cannot be read: %w", path, err)
	}
	document := map[string]any{}
	if err := toml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("%s: not valid TOML: %w", path, err)
	}
	manifest, err := declaration(document)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	manifest.Path = path
	return manifest, nil
}

// declarationSections are the tables a declaration reads, in the order their
// shape is checked.
func declarationSections() []string {
	return []string{"area", "privacy", "wiki", "maintenance", "model", "layout", "index"}
}

func declaration(document map[string]any) (*Manifest, error) {
	if _, present := document["area"]; !present {
		return nil, ErrNoArea
	}
	sections := map[string]map[string]any{}
	for _, name := range declarationSections() {
		value, present := document[name]
		if !present {
			sections[name] = map[string]any{}
			continue
		}
		section, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("[%s] must be a table, found %s", name, tomlType(value))
		}
		sections[name] = section
	}
	scope, err := requiredString(sections["area"], "scope", "[area]", " ")
	if err != nil {
		return nil, err
	}
	mode, err := privacyMode(sections["privacy"])
	if err != nil {
		return nil, err
	}
	types, err := stringList(sections["wiki"], "types", "[wiki]")
	if err != nil {
		return nil, err
	}
	if _, err := optionalBool(sections["maintenance"], "on_merge", "[maintenance]", " "); err != nil {
		return nil, err
	}
	if _, err := optionalString(sections["maintenance"], "branch", "[maintenance]", " "); err != nil {
		return nil, err
	}
	if _, err := optionalBool(sections["model"], "enabled", "[model]", " "); err != nil {
		return nil, err
	}
	if err := checkRoles(sections["model"]); err != nil {
		return nil, err
	}
	layout := map[string]string{}
	for _, key := range []string{"wiki", "hub", "review", "inbox"} {
		value, present := sections["layout"][key]
		if !present {
			continue
		}
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("[layout] %s must be a string, found %s", key, tomlType(value))
		}
		layout[key] = text
	}
	globs := map[string][]string{}
	for _, key := range []string{"include", "exclude", "unsearched"} {
		if globs[key], err = stringList(sections["index"], key, "[index]"); err != nil {
			return nil, err
		}
	}
	never, err := stringList(sections["privacy"], "never", "[privacy]")
	if err != nil {
		return nil, err
	}
	days, err := untouchedDays(sections["wiki"])
	if err != nil {
		return nil, err
	}
	return &Manifest{
		Scope:           scope,
		DeclaredTypes:   types,
		UntouchedDays:   days,
		LayoutWiki:      layout["wiki"],
		LayoutHub:       layout["hub"],
		LayoutReview:    layout["review"],
		LayoutInbox:     layout["inbox"],
		PrivacyMode:     mode,
		NeverGlobs:      never,
		IndexInclude:    globs["include"],
		IndexExclude:    globs["exclude"],
		IndexUnsearched: globs["unsearched"],
	}, nil
}

// privacyMode is the declared mode, manual_cloud where none is declared. A
// misspelt mode must fail loudly: falling back would turn the strictest
// setting into the most permissive one.
func privacyMode(privacy map[string]any) (string, error) {
	value, present := privacy["mode"]
	if !present {
		return "manual_cloud", nil
	}
	switch mode, _ := value.(string); mode {
	case "automatic_cloud", "local_only", "manual_cloud":
		return mode, nil
	}
	return "", fmt.Errorf("[privacy] mode must be one of automatic_cloud, local_only, manual_cloud, found %s", found(value))
}

// stringList is an optional array of strings. Written as a bare string, a
// glob list would otherwise match nothing and a run would succeed on zero
// files.
func stringList(section map[string]any, key, owner string) ([]string, error) {
	value, present := section[key]
	if !present {
		return nil, nil
	}
	list, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%s %s must be an array of strings, found %s", owner, key, tomlType(value))
	}
	texts := make([]string, 0, len(list))
	for i, element := range list {
		text, ok := element.(string)
		if !ok {
			return nil, fmt.Errorf("%s %s #%d must be a string, found %s", owner, key, i+1, tomlType(element))
		}
		texts = append(texts, text)
	}
	return texts, nil
}

// checkRoles refuses an unknown role rather than skipping it: a typo would
// leave a role switched off that the person believes they switched on.
func checkRoles(model map[string]any) error {
	value, present := model["roles"]
	if !present {
		return nil
	}
	roles, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("[model] roles must be a table, found %s", tomlType(value))
	}
	known := []string{"describe", "place", "propose"}
	var unknown []string
	for name := range roles {
		if !slices.Contains(known, name) {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		quoted := make([]string, len(unknown))
		for i, name := range unknown {
			quoted[i] = strconv.Quote(name)
		}
		return fmt.Errorf("[model] roles has unknown %s; known are %s", strings.Join(quoted, ", "), strings.Join(known, ", "))
	}
	for _, name := range known {
		if _, err := optionalBool(roles, name, "[model]", " roles."); err != nil {
			return err
		}
	}
	return nil
}

// untouchedDays is the lint threshold; a boolean is no number of days.
func untouchedDays(wiki map[string]any) (int, error) {
	value, present := wiki["untouched_days"]
	if !present {
		return DefaultUntouchedDays, nil
	}
	days, ok := value.(int64)
	if !ok {
		return 0, fmt.Errorf("[wiki] untouched_days must be an integer >= 1, found %s", tomlType(value))
	}
	if days < 1 {
		return 0, fmt.Errorf("[wiki] untouched_days must be an integer >= 1, found %d", days)
	}
	return int(days), nil
}

// InboxLayout is where the area's inbox sits, checked, or "" when the
// declaration names none. An absolute value would leave the area tree when
// joined onto it. Absolute is filepath.IsAbs on the running platform, so on
// Windows a rooted `/in` without a drive passes, as it did in the barrier.
func (m *Manifest) InboxLayout() (string, error) {
	if filepath.IsAbs(m.LayoutInbox) {
		return "", fmt.Errorf("%s: [layout] inbox must be relative to the area, found %q", m.Path, m.LayoutInbox)
	}
	return m.LayoutInbox, nil
}
```

- [ ] **Step 5: `ReadAreaManifestUntilStage4` in `legacy.go` ersetzen**

Imports von `legacy.go`: `errors`, `fmt`, `os`, `path/filepath`, `runtime`, `strings`.

```go
// ReadAreaManifestUntilStage4 reads an area's declaration under the name loomux writes, or
// under one of ultra-brain's names until stage 4 moves the hosts:
// .loomux/config.toml, else .ultra-brain/config.toml, else .brain.toml. Only brain/* calls it.
//
// The first name that is a regular file decides, and it is checked whole by
// ReadDeclaration. A .loomux/config.toml without [area] is policy only and
// the next name is asked; the old names have no such form, so there the
// missing table is a missing scope.
func ReadAreaManifestUntilStage4(dir string) (*Manifest, error) {
	for _, name := range manifestNamesUntilStage4 {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			continue
		}
		manifest, err := ReadDeclaration(path)
		if errors.Is(err, ErrNoArea) {
			if name == manifestNamesUntilStage4[0] {
				continue
			}
			return nil, fmt.Errorf("%s: [area] is missing %q", path, "scope")
		}
		return manifest, err
	}
	return nil, fmt.Errorf("%s: %w (%s)", dir, ErrNoManifest, strings.Join(manifestNamesUntilStage4, ", "))
}
```

Der Kommentar über den Ablaufzeitpunkt an `manifestNamesUntilStage4` bleibt stehen.

- [ ] **Step 6: Tests laufen lassen**

Run: `go test ./internal/config/ -count=1`
Expected: PASS. Falls `TestALockedFirstNameIsAnErrorEvenBesideAnOpenLegacyName` scheitert, prüfen, ob `ReadDeclaration` `cannot be read: ` vor den Fehler setzt.

Run: `gofmt -w internal/config && go vet ./internal/config/`
Expected: keine Ausgabe. `gofmt -w` richtet nur die Spalten der Testtabellen aus; die Codeblöcke dieses Plans sind am 2026-09-16 probeweise übernommen, kompiliert und mit 100 % Abdeckung in `internal/config` getestet worden.

Run: `go test ./internal/brain/... ./internal/cli/ ./internal/hooks/ -count=1`
Expected: PASS.

- [ ] **Step 7: Coverage der neuen Funktionen prüfen**

```bash
go test ./internal/config/ -count=1 -covermode=set -coverprofile="$TEMP/loomux-registry-bench/cfg.out"
go tool cover -func="$TEMP/loomux-registry-bench/cfg.out" | grep -v "100.0%"
```

Expected: nur die Zeile `total:`. Steht eine Funktion aus `declaration.go`, `tomlvalue.go`, `registry.go` oder `legacy.go` darunter, fehlt ein Testfall. Den Fall ergänzen, keine Ausnahme setzen.

- [ ] **Step 8: Commit**

Nachricht in `$TEMP/loomux-registry-bench/msg-task3.txt`:

```
Check an area declaration whole before brain reads it

The brain reader checked only the scope and the privacy mode; a broken
glob list failed as a TOML type error and a bad untouched_days, model
or inbox went through. ReadDeclaration checks every value loomux reads
or the reference refused, with loomux's own wording, and the stage 4
reader uses it for all three names.
```

```bash
git branch --show-current
git log -1 --oneline
git add internal/config/declaration.go internal/config/declaration_test.go internal/config/manifest.go internal/config/legacy.go internal/config/legacy_test.go
git commit -F "$TEMP/loomux-registry-bench/msg-task3.txt"
```

---

### Task 4: `privacy.VisibleAreas` prüft `inbox`

**Files:**
- Modify: `internal/brain/privacy/areas.go` (`VisibleAreas`, Kommentar von `Single`)
- Test: `internal/brain/privacy/areas_test.go`

**Interfaces:**
- Consumes: `config.ReadRegistry`, `config.ReadAreaManifestUntilStage4` (über `VisibleManifest`), `(*config.Manifest).InboxLayout`
- Produces: `VisibleAreas` (Signatur unverändert)

- [ ] **Step 1: Tests schreiben**

In `internal/brain/privacy/areas_test.go` anhängen:

```go
func TestVisibleAreasRefusesADuplicateScope(t *testing.T) {
	registryDir, legacyDir := buildWorld(t,
		registered{scope: "a", mode: "manual_cloud", manifestName: ".brain.toml"},
		registered{scope: "a", mode: "local_only", manifestName: ".brain.toml"},
	)
	got, err := privacy.VisibleAreas(registryDir, legacyDir, "a", privacy.ChannelLocal)
	if err == nil || got != nil || !strings.HasSuffix(err.Error(), `[[area]] #2: duplicate scope "a" (first at #1)`) {
		t.Fatalf("got %v, %v; want the duplicate refused", got, err)
	}
}

func TestVisibleAreasRefusesAnAbsoluteInboxEvenOfAHiddenArea(t *testing.T) {
	// The reference checks every area's inbox while reading the registry,
	// before visibility is asked, so a local_only area cannot hide a broken
	// declaration from a cloud caller.
	root := t.TempDir()
	area := filepath.Join(root, "closed")
	inbox := filepath.ToSlash(filepath.Join(root, "in"))
	writeFile(t, filepath.Join(root, "state", "registry.toml"),
		fmt.Sprintf("[[area]]\nscope = \"closed\"\npath = %q\n", filepath.ToSlash(area)))
	writeFile(t, filepath.Join(area, ".brain.toml"),
		fmt.Sprintf("[area]\nscope = \"closed\"\n\n[privacy]\nmode = \"local_only\"\n\n[layout]\ninbox = %q\n", inbox))
	got, err := privacy.VisibleAreas(filepath.Join(root, "state"), filepath.Join(root, "legacy"), "all", privacy.ChannelCloud)
	if err == nil || got != nil || !strings.Contains(err.Error(), "[layout] inbox must be relative to the area") {
		t.Fatalf("got %v, %v; want the inbox refused", got, err)
	}
}
```

- [ ] **Step 2: Tests laufen lassen**

Run: `go test ./internal/brain/privacy/ -run 'Duplicate|AbsoluteInbox' -count=1`
Expected: `TestVisibleAreasRefusesADuplicateScope` PASS, denn die Registry ist seit Task 2 streng. `TestVisibleAreasRefusesAnAbsoluteInboxEvenOfAHiddenArea` FAIL (`got [], <nil>`).

- [ ] **Step 3: `VisibleAreas` ändern**

In `internal/brain/privacy/areas.go` die Schleife ersetzen:

```go
	for _, area := range areas {
		manifest, seen, err := VisibleManifest(config.ManifestDir(area, legacyDir), ch)
		if err != nil {
			return nil, err
		}
		if _, err := manifest.InboxLayout(); err != nil {
			return nil, err
		}
		if seen {
			visible = append(visible, VisibleArea{Area: area, Manifest: manifest})
		}
	}
```

Im Doc-Kommentar von `VisibleAreas` den Satz „Python reads the manifest of every registered area before it looks at scope, so the first registry or manifest error ends the call, whichever area it belongs to.“ ersetzen durch: „Every registered area's declaration is read and its inbox checked before scope and visibility are asked, so the first registry or declaration error ends the call, whichever area it belongs to -- a hidden area included.“

Im Doc-Kommentar von `Single` anhängen: „The registry refuses two entries of one scope, so there is at most one.“

- [ ] **Step 4: Tests laufen lassen**

Run: `go test ./internal/brain/... ./internal/cli/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

Nachricht in `$TEMP/loomux-registry-bench/msg-task4.txt`:

```
Refuse an absolute inbox before brain answers any area

The reference checks every area's inbox while reading the registry. brain
now does the same for every registered area, hidden ones included, before
it asks scope or channel.
```

```bash
git branch --show-current
git log -1 --oneline
git add internal/brain/privacy/areas.go internal/brain/privacy/areas_test.go
git commit -F "$TEMP/loomux-registry-bench/msg-task4.txt"
```

---

### Task 5: Die Schranke liest über `config`

**Files:**
- Modify: `internal/brain/guard/registry.go`, `internal/brain/guard/manifest.go`, `internal/brain/guard/guard.go`, `internal/brain/guard/python.go`
- Test: `internal/brain/guard/decide_test.go`, `internal/brain/guard/internal_test.go`, `internal/brain/guard/mutation_test.go`, `internal/config/manifest_test.go`

**Interfaces:**
- Consumes: `config.ReadRegistry`, `config.ReadDeclaration`, `config.ErrNoArea`, `(*config.Manifest).InboxLayout`, `(*config.Manifest).WikiLayout`, `config.Manifest.LayoutReview`, `config.ManifestDir`
- Produces: `readRegistry(stateDir string) ([]area, error)` (Signatur unverändert); `area` unverändert

- [ ] **Step 1: Erwartungen der Schrankentests umstellen (rot)**

In `internal/brain/guard/decide_test.go` die erwarteten Teilstrings so ändern (Funktion → neuer dritter Argumentwert von `deny`):

| Test | neuer Teilstring |
|---|---|
| `TestAreasDeclaredAsATableRefuse` | `"area must be an array of [[area]] tables, found table"` |
| `TestAnEntryWithoutAPathRefuses` | `` `[[area]] "project/demo" is missing "path"` `` |
| `TestAnEntryWithoutAScopeRefuses` | `` `[[area]] #1 is missing "scope"` `` |
| `TestAScopeThatIsNotAStringRefuses` | `"[[area]] #1: scope must be a non-empty string, found integer"` |
| `TestADuplicateScopeRefuses` | `` `[[area]] #2: duplicate scope "a" (first at #1)` `` |
| `TestTwoScopesSharingAStateDirectoryRefuse` | `` `scopes "a/b" and "a-b" share the state directory "a-b"` `` |
| `TestAScopeWithoutUsableCharactersRefuses` | `` `[[area]] "///": scope has no letter, digit` `` |
| `TestTwoSignpostsRefuse` | `` `scopes "a" and "b" both declare signpost; only one area may` `` |
| `TestAnEntryThatIsNotATableRefuses` | `"[[area]] #1 must be a table, found integer"` |
| `TestAWikiThatIsNotAStringRefuses` | `` `[[area]] "a": wiki must be a non-empty string, found integer` `` |
| `TestAManifestWithoutAScopeRefusesEverything` | `` `[area] is missing "scope"` `` |
| `TestAnInboxThatIsNotAStringRefusesEverything` | `"[layout] inbox must be a string, found integer"` |
| `TestABadPrivacyModeRefusesEverything` | `` `[privacy] mode must be one of automatic_cloud, local_only, manual_cloud, found "x"` `` |
| `TestABadTypesListRefusesEverything` | `"[wiki] types must be an array of strings, found string"` |
| `TestABadOnMergeRefusesEverything` | `"[maintenance] on_merge must be a boolean, found integer"` |
| `TestABadGlobListRefusesEverything` | `"[index] include #1 must be a string, found integer"` |
| `TestAnAreaTableWithoutAScopeStillRefuses` | `` `[area] scope must be a non-empty string, found ""` `` |
| `TestAWikiLayoutReachingOutIsRefused` (Zeile mit `found '../out'`) | `` `[layout] wiki must stay inside the repository, found "../out"` `` |
| `TestAnAbsoluteWikiLayoutIsRefused` (Zeile mit `found '/srv/w'`) | `` `[layout] wiki must stay inside the repository, found "/srv/w"` `` |

Kommentare, die sich auf Python-Wortlaut oder `repr` berufen, in diesen Tests streichen. Die Kommentare, die das *Verhalten* begründen, bleiben.

Fünf Tests ändern ihr Urteil und ihren Namen:

```go
func TestAReadonlyStringRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"),
		"readonly = \"yes\"")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		`[[area]] "project/demo": readonly must be a boolean, found string`)
}

func TestAZeroWorkspaceRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"),
		"workspace = 0")
	deny(t, writeCall(filepath.Join(tmp, "repo", "a.py")), state,
		`[[area]] "project/demo": workspace must be a boolean, found integer`)
}

func TestAnEmptyWikiStringRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"a\"\npath = \"/x\"\nwiki = \"\"\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		`[[area]] "a": wiki must be a non-empty string, found ""`)
}

func TestAFalseWikiLayoutRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = false\n")
	deny(t, writeCall(filepath.Join(tmp, "repo", "w", "x.md")), state,
		"[layout] wiki must be a string, found boolean")
}

func TestAReviewThatIsNotAStringRefusesEverything(t *testing.T) {
	// Checked while the registry is read, like every other declaration
	// value: a broken review closes all writes, not only the exemption.
	tmp := t.TempDir()
	state := withReview(t, tmp, "1")
	deny(t, writeCall(caseFile(tmp, "proposal.md")), state,
		"[layout] review must be a string, found integer")
}
```

Sie ersetzen in dieser Reihenfolge `TestAReadonlyStringIsReadAsTrue`, `TestAZeroWorkspaceIsFalse`, `TestAnEmptyWikiStringIsNoWiki`, `TestAFalseWikiLayoutDeclaresNothing` und `TestAReviewThatIsNotAStringClosesTheExemption`. `withReview` schreibt das Manifest nach `<tmp>/repo/.loomux/config.toml`, und dort liest `manifestPath` für den registrierten Bereich; der zweite Registry-Durchlauf trifft den Wert also.

`TestAnEmptyScopeOrPathRefuses` bekommt die Tabelle:

```go
	for body, want := range map[string]string{
		"[[area]]\nscope = \"\"\npath = \"/x\"\n": `[[area]] #1: scope must be a non-empty string, found ""`,
		"[[area]]\nscope = \"a\"\npath = \"\"\n":  `[[area]] "a": path must be a non-empty string, found ""`,
	} {
		write(t, filepath.Join(state, "registry.toml"), body)
		deny(t, writeCall(filepath.Join(tmp, "x.md")), state, want)
	}
```

Neu in `decide_test.go` nach `TestABadGlobListRefusesEverything`:

```go
func TestABrokenModelSectionRefusesEverything(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	target := writeCall(filepath.Join(tmp, "vault", "demo", "x.md"))
	area := "[area]\nscope = \"project/demo\"\n\n"
	for body, want := range map[string]string{
		"model = 5\n\n" + area:                       "[model] must be a table, found integer",
		area + "[model]\nenabled = \"yes\"\n":        "[model] enabled must be a boolean, found string",
		area + "[model]\nroles = 5\n":                "[model] roles must be a table, found integer",
		area + "[model.roles]\nguess = true\n":       `[model] roles has unknown "guess"; known are describe, place, propose`,
		area + "[model.roles]\nplace = \"yes\"\n":    "[model] roles.place must be a boolean, found string",
	} {
		write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"), body)
		deny(t, target, state, want)
	}
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		area+"[model]\nenabled = true\n\n[model.roles]\nplace = true\n")
	allow(t, target, state)
}
```

In `internal/brain/guard/internal_test.go`:
- löschen: `TestPyReprRendersWhatReprRenders`, `TestPyReprFloatKeepsTheDecimalPointPythonKeeps`, `TestTruthyIsPythonsBool`, `TestAColonThatNamesNoDriveIsAnOrdinaryName`, `TestTheGoWikiLayoutAndTheBarriersAnswerTheSame`, `TestReadManifestAnswersTheErrorOfAFileItCannotRead` (ersetzt durch `TestADeclarationThatCannotBeReadIsNoAbsence` aus Task 3)
- `TestATypesListOfNonStringsIsRefused`: Teilstring `"[wiki] types #1 must be a string, found integer"`

In `internal/brain/guard/mutation_test.go`:
- löschen: `TestADriveIsOnlyADriveWithTheSeparatorBehindIt`, `TestAValueOfNothingButDotsAndSlashesNamesTheRoot` (die Werte ziehen unten nach `internal/config`)
- `TestABrokenManifestSaysWhichDefectItFound`: in der Map `"[area] must be a table"` → `"[area] must be a table, found string"`; die Prüfung `strings.Contains(reason, "scope is required")` → `strings.Contains(reason, "is missing \"scope\"")`; den Block ab `// And the read error` bis vor die schließende Klammer löschen; im Kommentar `errNoArea` → `config.ErrNoArea`, `readManifest` → `config.ReadDeclaration`
- `TestAMissingKeyIsBlamedOnTheEntryThatCanBeFound` ersetzen durch:

```go
func TestAMissingKeyIsBlamedOnTheEntryThatCanBeFound(t *testing.T) {
	// Where the scope is known the entry is named by it; where it is the
	// missing key, by its position.
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"project/demo\"\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		`[[area]] "project/demo" is missing "path"`)
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"a\"\npath = \"/a\"\n\n[[area]]\npath = \"/x\"\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		`[[area]] #2 is missing "scope"`)
}
```

In `internal/config/manifest_test.go` die Werte der gelöschten Schrankentests an `WikiLayout` festhalten:

```go
func TestTheWikiValuesTheBarrierHeldAreStillJudgedSo(t *testing.T) {
	// Moved from the barrier when it began to read through this package:
	// a drive is one character followed by ":/", and a value of nothing but
	// dots and slashes names the repository root.
	for value, kept := range map[string]bool{
		"C:x": true, "ab:/c": true, ":/x": true, "1:/w": false, "\u00c4:/w": false,
		"C:/": false, "C:/w": false, "/srv/w": false, "//": false,
		".": false, "./": false, ".//": false, "./.": false, "a/../..": false,
	} {
		got, err := (&Manifest{LayoutWiki: value}).WikiLayout()
		if kept && (err != nil || got != value) {
			t.Errorf("WikiLayout(%q) = %q, %v; wanted it kept", value, got, err)
		}
		if !kept && err == nil {
			t.Errorf("WikiLayout(%q) = %q; wanted a refusal", value, got)
		}
	}
}
```

- [ ] **Step 2: Tests laufen lassen, sie müssen scheitern**

Run: `go test ./internal/brain/guard/ -count=1`
Expected: FAIL. Die umgestellten Teilstrings fehlen in den alten Meldungen, `TestABrokenModelSectionRefusesEverything` wird erlaubt. Kompilierfehler wegen nicht mehr benutzter Imports in den Testdateien (`errors`, `fs`) entfernen, bis nur noch Testfehler bleiben.

Run: `go test ./internal/config/ -run TestTheWikiValuesTheBarrierHeldAreStillJudgedSo -count=1`
Expected: PASS (reiner Nachweis, dass `config` diese Werte schon so beurteilt). Scheitert ein Wert, nicht `config` ändern, sondern zurückmelden: Dann antworteten Schranke und `config` verschieden.

- [ ] **Step 3: `guard/registry.go` ersetzen**

Die ganze Datei wird zu:

```go
package guard

import (
	"errors"
	"path/filepath"

	"github.com/xidus90/loomux/internal/config"
)

// area is one registered area, reduced to the fields this barrier decides
// on. The paths stay as the file spells them; whoever compares them
// resolves them first.
type area struct {
	scope     string
	path      string
	wikiPath  string
	readOnly  bool
	workspace bool
}

// readRegistry reads the registry through config, which brain reads it
// through too, and then every registered area's declaration: a broken
// declaration in any area -- a read-only one included, whose manifest lives
// under the state directory -- refuses every write.
func readRegistry(stateDir string) ([]area, error) {
	registered, err := config.ReadRegistry(stateDir)
	if err != nil {
		return nil, err
	}
	areas := make([]area, 0, len(registered))
	for _, entry := range registered {
		areas = append(areas, area{
			scope:     entry.Scope,
			path:      entry.Path,
			wikiPath:  entry.WikiPath,
			readOnly:  entry.ReadOnly,
			workspace: entry.Workspace,
		})
	}
	for _, registered := range areas {
		if err := checkDeclaration(registered, stateDir); err != nil {
			return nil, err
		}
	}
	return areas, nil
}

// areaStateDir is where a read-only area keeps its artefacts; the rule that
// names it lives in config.ManifestDir.
func areaStateDir(stateDir, scope string) string {
	return config.ManifestDir(
		config.Area{Scope: scope, ReadOnly: true}, stateDir)
}

// manifestPath is the one declaration this barrier reads for a registered
// area: the manifest follows the artefacts, so a read-only area's
// declaration is read from the state directory rather than from the tree
// the barrier does not own.
func manifestPath(registered area, stateDir string) string {
	base := registered.path
	if registered.readOnly {
		base = areaStateDir(stateDir, registered.scope)
	}
	return filepath.Join(base, bundleDir, manifestName)
}

// checkDeclaration refuses a registered area whose declaration exists and is
// broken. Registering an area before it declares itself is normal, so a
// missing file and a configuration without [area] are no defect.
func checkDeclaration(registered area, stateDir string) error {
	declaration := manifestPath(registered, stateDir)
	if !isRegularFile(declaration) {
		return nil
	}
	manifest, err := config.ReadDeclaration(declaration)
	if errors.Is(err, config.ErrNoArea) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = manifest.InboxLayout()
	return err
}
```

- [ ] **Step 4: `guard/manifest.go` ersetzen**

Die ganze Datei wird zu:

```go
package guard

import (
	"os"
	"path/filepath"
)

// bundleDir and manifestName name the one manifest the barrier reads.
// ultra-brain had a second, legacy spelling; loomux cut it on 2026-09-14.
const (
	bundleDir    = ".loomux"
	manifestName = "config.toml"
)

// declarationIn is the manifest this one directory carries, or "" if it
// carries none. manifestPath answers for a registered area; this walk climbs
// a path no area is known for yet.
func declarationIn(directory string) string {
	bundled := filepath.Join(directory, bundleDir, manifestName)
	if isRegularFile(bundled) {
		return bundled
	}
	return ""
}

// isRegularFile follows links and answers no rather than failing where the
// path cannot be looked at.
func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
```

- [ ] **Step 5: `guard/guard.go` anpassen**

In `reviewCentre` diesen Block

```go
		read, err := readManifest(declaration)
		if errors.Is(err, errNoArea) {
			continue
		}
		if err != nil {
			return ""
		}
		declared := read.layout["review"]
		if !truthy(declared) {
			continue
		}
		value, ok := declared.(string)
		if !ok {
			return ""
		}
		root := filepath.Join(registered.path, value)
```

ersetzen durch

```go
		read, err := config.ReadDeclaration(declaration)
		if errors.Is(err, config.ErrNoArea) {
			continue
		}
		if err != nil {
			return ""
		}
		if read.LayoutReview == "" {
			continue
		}
		root := filepath.Join(registered.path, read.LayoutReview)
```

In `declaredWikiRoot` `readManifest(declaration)` → `config.ReadDeclaration(declaration)`, `errNoArea` → `config.ErrNoArea`, `read.scope` → `read.Scope`, `wikiLayout(read.layout)` → `read.WikiLayout()`.

`"github.com/xidus90/loomux/internal/config"` in die Imports von `guard.go` aufnehmen, falls es dort fehlt. Kommentare in `guard.go`, die `readManifest`, `errNoArea` oder `truthy` nennen, auf die neuen Namen umschreiben.

- [ ] **Step 6: `guard/python.go` kürzen**

`truthy`, `pyRepr`, `pyReprString` und `pyReprFloat` samt Kommentaren löschen und den Import `strconv` entfernen. `pythonJSONString`, `utf16Pair` und `pythonTypeName` bleiben.

Run: `grep -rn "truthy\|pyRepr\|readManifest\|errNoArea\|wikiLayout(\|checkInbox" internal/brain/guard/`
Expected: keine Treffer außer Kommentaren, die ausdrücklich Geschichte beschreiben. Solche Kommentare umschreiben, bis keine Treffer bleiben.

- [ ] **Step 7: Tests laufen lassen**

Run: `gofmt -w internal && go vet ./...`
Expected: keine Ausgabe. Nach den Löschungen sind `errors` und `io/fs` in `internal_test.go` und `mutation_test.go` unbenutzt (so am 2026-09-16 probeweise gemessen); `go vet` nennt sie, und sie werden entfernt. Gegen den Probestand scheiterten 24 alte Tests, und jeder davon steht in Step 1; Step 1 schärft darüber hinaus einige, die mit einem allgemeinen Teilstring weiter bestanden.

Run: `go test ./internal/brain/guard/ ./internal/config/ ./internal/hooks/ ./internal/cli/ ./internal/dev/... -count=1`
Expected: PASS. Am 2026-09-16 probeweise gemessen: Mit dem Nicht-Test-Code dieses Tasks bleiben `internal/cli` (einschließlich der 1a- und 1b-1-Fälle), `internal/hooks` und `internal/dev/...` grün. Kein aufgezeichneter Fall hält einen Registry- oder Manifestwortlaut fest. Scheitert hier ein Fall, ist das keine Testreparatur: anhalten und berichten.

- [ ] **Step 8: Coverage der Schranke prüfen**

```bash
go test ./... -count=1 -covermode=set -coverpkg=./... -coverprofile=coverage.out
go run ./cmd/loomux dev covergate --profile coverage.out
```

Expected: covergate grün. Nennt es `reviewCentre` oder `checkDeclaration`, fehlt ein Zweig. `TestAReviewCentreIsDroppedWhenItsManifestBreaksUnderIt` deckt den Fehlerzweig von `reviewCentre`. `TestAConfigWithoutAnAreaDoesNotCloseTheReviewCentre` deckt `ErrNoArea`.

- [ ] **Step 9: Commit**

Nachricht in `$TEMP/loomux-registry-bench/msg-task5.txt`:

```
Read the registry and declarations of the barrier through config

The write barrier kept its own readers that rebuilt Python's repr and
truthiness. It now reads through the functions brain uses, so both refuse
the same files with the same words. The barrier refuses what it let
through before: a broken [model] section, a flag that is not a boolean,
an empty wiki path, and a [layout] value that is not a string.
```

```bash
git branch --show-current
git log -1 --oneline
git add internal/brain/guard internal/config/manifest_test.go
git commit -F "$TEMP/loomux-registry-bench/msg-task5.txt"
```

---

### Task 6: Nachweis auf echten Daten, Messung, Parität, READMEs

**Files:**
- Create: `testdata/bench/registry-checks.json`, `docs/.superpowers/parity/registry-manifest-pruefungen.md`
- Modify: `docs/.superpowers/parity/stufe-1b-1.md`, `docs/en/benchmarks.md`, `docs/de/benchmarks.md`, `README.md`, `README.de.md`, `docs/.superpowers/specs/2026-09-16-loomux-registry-manifest-pruefungen-design.md` (Kopfzeile `**Stand:**`)

**Interfaces:**
- Consumes: `…/loomux-registry-bench/loomux-before.exe`, `…/before-go-bench.txt` (Task 1); `bin/loomux.exe` (vom Gate in Task 5 gebaut)
- Produces: nichts für spätere Tasks

- [ ] **Step 1: Echte Daten**

```bash
bin/loomux.exe brain catalog | grep -c '^\* '
```

Expected: `11`. Andernfalls anhalten und die Fehlermeldung berichten. Eine Verweigerung auf der echten Registry ist ein Befund, keine Testanpassung.

```bash
printf '{"tool_name":"Edit","tool_input":{"file_path":"C:/Users/micro/Documents/#GIT/loomux/README.md","old_string":"a","new_string":"b"}}' | bin/loomux.exe hook pre-tool-use --host claude --root "C:/Users/micro/Documents/#GIT/loomux"; echo "exit $?"
```

Expected: keine Ausgabe, `exit 0`.

- [ ] **Step 2: Messfälle anlegen**

`testdata/bench/registry-checks.json`:

```json
[
  {
    "name": "before: loomux hook pre-tool-use (Edit on README.md, real registry)",
    "dir": "C:/Users/micro/Documents/#GIT/loomux",
    "stdin": "C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/recursing-bartik-b2d7a1/testdata/bench/edit-readme-main.json",
    "mode": "single",
    "steps": [{"argv": ["C:/Users/micro/AppData/Local/Temp/loomux-registry-bench/loomux-before.exe", "hook", "pre-tool-use", "--host", "claude", "--root", "C:/Users/micro/Documents/#GIT/loomux"]}]
  },
  {
    "name": "after: loomux hook pre-tool-use (Edit on README.md, real registry)",
    "dir": "C:/Users/micro/Documents/#GIT/loomux",
    "stdin": "C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/recursing-bartik-b2d7a1/testdata/bench/edit-readme-main.json",
    "mode": "single",
    "steps": [{"argv": ["C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/recursing-bartik-b2d7a1/bin/loomux.exe", "hook", "pre-tool-use", "--host", "claude", "--root", "C:/Users/micro/Documents/#GIT/loomux"]}]
  },
  {
    "name": "before: loomux brain catalog (real registry)",
    "dir": "C:/Users/micro/Documents/#GIT/loomux",
    "mode": "single",
    "steps": [{"argv": ["C:/Users/micro/AppData/Local/Temp/loomux-registry-bench/loomux-before.exe", "brain", "catalog"]}]
  },
  {
    "name": "after: loomux brain catalog (real registry)",
    "dir": "C:/Users/micro/Documents/#GIT/loomux",
    "mode": "single",
    "steps": [{"argv": ["C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/recursing-bartik-b2d7a1/bin/loomux.exe", "brain", "catalog"]}]
  }
]
```

Vor dem Lauf prüfen, ob `dev bench-hooks` für Fälle ohne `stdin` eine leere Eingabe liefert. `internal/cli/dev.go` und `internal/dev/benchhooks/benchhooks.go` lesen. `1b-1-brain.json` hat Fälle ohne `stdin`, dort funktioniert es also.

- [ ] **Step 3: Messen**

```bash
B="$TEMP/loomux-registry-bench"
date '+%Y-%m-%d %H:%M' > "$B/after.txt"
bin/loomux.exe dev bench-hooks testdata/bench/registry-checks.json -n 20 >> "$B/after.txt"
LOOMUX_BENCH_REGISTRY="$B/state" LOOMUX_BENCH_TARGET="C:/Users/micro/Documents/#GIT/loomux/README.md" go test ./internal/brain/guard/ -run '^$' -bench DecideAgainstTheRealRegistry -benchtime 50x -benchmem -count 5 >> "$B/after.txt"
LOOMUX_BENCH_REGISTRY="$B/state" LOOMUX_BENCH_LEGACY="$LOCALAPPDATA/brain" go test ./internal/brain/search/ -run '^$' -bench VisibleAreasOfTheRealRegistry -benchtime 50x -benchmem -count 5 >> "$B/after.txt"
cat "$B/before-go-bench.txt" "$B/after.txt"
```

Expected: alle Exitcodes `[0]`. Nimmt `…/after`-Median der Schranke mehr als 3 ms oder der Go-Benchmark mehr als 20 % gegenüber `before` zu, anhalten und mit `-cpuprofile` untersuchen, bevor irgendetwas eingetragen wird. Befund und Profil berichten.

- [ ] **Step 4: Benchmarks eintragen**

Am Ende von `docs/en/benchmarks.md` einen Abschnitt im Format der bestehenden Einträge anhängen:
- Überschrift `## <Datum Uhrzeit aus after.txt> — Registry and Declaration Checks in One Place`
- Kopfabsatz: Repository, Worktree, Zweig, Commit von Task 5
- **Goal.** Was die strengen Leser an der Schranke und an `brain catalog` kosten
- **Method.** Die Befehle aus Step 3 und Task 1, Binaries `loomux-before.exe` (Commit aus Task 1) und `bin/loomux.exe`, Zustandskopie, Maschine
- Tabelle `bench-hooks` (cold, warm median, warm min, warm max, exit codes) aus `after.txt`
- Tabelle Go-Benchmarks (ns/op, B/op, allocs/op), vorher gegen nachher, jeweils Median der fünf Läufe
- `### Reading`: die gemessenen Unterschiede in Zahlen, kalt und warm getrennt, ohne Deutung über die Zahlen hinaus

Denselben Abschnitt deutsch in `docs/de/benchmarks.md` anhängen, gleiche Zahlen, Überschrift `— Registry- und Deklarationsprüfungen an einer Stelle`.

- [ ] **Step 5: Paritätsliste anlegen**

`docs/.superpowers/parity/registry-manifest-pruefungen.md`:

```markdown
# Paritätsliste Registry- und Manifestprüfungen

**Quelle:** ultra-brain `loomux-1a-source` (`3cc72d2`), Python-Referenz `src/brain/registry.py` und `src/brain/manifest.py`, gemessen am 2026-09-16 (Anhang der Spec `docs/.superpowers/specs/2026-09-16-loomux-registry-manifest-pruefungen-design.md`).
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers. „Alt“ ist die Python-Referenz, „Neu“ ist `loomux brain` und die Schreibschranke.
**Stand:** Alle Zeilen offen.

| Bereich | Alt | Neu | Begründung | Freigabe |
|---|---|---|---|---|
| Wortlaut aller Registry- und Manifestmeldungen | `repr`-Form, z. B. `duplicate scope 'x'`, `[[area]] entry {'path': '…'} is missing the 'scope' key`, `[wiki] types must be a list of strings` | loomux-Form nach G1–G14 und M1–M17 der Spec, z. B. `[[area]] #2: duplicate scope "x" (first at #1)`, `[[area]] #1 is missing "scope"`, `[wiki] types must be an array of strings, found string`; gilt für `brain` und Schranke | Entscheidung vom 2026-09-16: Die Referenz bestimmt, was verweigert wird, nicht den Wortlaut; die Meldungen landen bei einem Menschen, der die Datei repariert | offen |
| Strenge Typen | `readonly`/`signpost`/`shared`/`workspace` über `bool()`, `wiki = ""` als kein Wiki, `[layout]`-Werte falsy als nicht gesagt | verweigert, wenn kein Wahrheitswert, leer oder keine Zeichenkette | ein Tippfehler in einer Flagge wirkt sonst still; echte Registries und Manifeste tragen nur gültige Typen (gemessen 2026-09-16) | offen |
| Traceback-Fälle | `wiki = 1` in der Registry: `TypeError`; `privacy = 5` oder `wiki = 5` im Manifest: `AttributeError` | Meldung nach G12 bzw. M2, Exit 1 | eine Meldung statt eines Abbruchs | offen |
| Reihenfolge bei zwei Defekten | `inbox` eines Bereichs wird geprüft, bevor der nächste Registry-Eintrag gelesen wird | erst alle Registry-Einträge, dann je Bereich das Manifest | gleiches Urteil, der genannte Grund kann abweichen | offen |
| Registry nicht lesbar | `[Errno 2] No such file or directory: '<pfad>'` | `open <pfad>: <Systemtext>` (bisher `<pfad>: open <pfad>: …`) | der Go-Fehler nennt den Pfad schon; ersetzt den Registry-Teil der Zeile „Betriebssystemfehler im Wortlaut“ der Liste 1b-1 | offen |
| Schranke verweigert mehr | — (die Referenz prüfte `[model]` beim Lesen) | die Schranke verweigert jeden Write bei kaputtem `[model]` und bei `[layout] wiki`/`hub`/`review`/`inbox`, die keine Zeichenkette sind, in irgendeinem registrierten Bereich; bisher ließ sie `[model]` durch, und ein nicht-textuelles `review` hob nur die Vorschlagsausnahme auf | eine Wahrheit für `brain` und Schranke; gemessen trifft es heute keinen Bereich | offen |
```

- [ ] **Step 6: Liste 1b-1 ergänzen**

In `docs/.superpowers/parity/stufe-1b-1.md`:
- In den Zeilen „Registry-Prüfungen“, „`[area] scope` als Nicht-Zeichenkette“ und „Weitere Prüfungen von `read_manifest`“ das Freigabefeld `freigegeben 2026-09-16 (Nachtrag: Registry- und Manifestprüfungen nachrüsten)` ersetzen durch `freigegeben 2026-09-16 (Nachtrag: Registry- und Manifestprüfungen nachrüsten; nachgerüstet, siehe registry-manifest-pruefungen.md)`.
- In der Zeile „Betriebssystemfehler im Wortlaut“ im Feld „Neu“ die Angabe `bei der Registry \`<pfad>: open <pfad>: …\`` ersetzen durch `bei der Registry \`open <pfad>: …\` (seit registry-manifest-pruefungen.md)`.

Run: `grep -n "nachgerüstet, siehe registry-manifest-pruefungen.md\|seit registry-manifest-pruefungen.md" docs/.superpowers/parity/stufe-1b-1.md`
Expected: vier Treffer.

- [ ] **Step 7: READMEs und Spec-Stand**

In `README.md`, Tabellenzeile „Unified Pre-Tool Guard“: in der Beschreibungsspalte den Satz `Registry and area declarations are read through the same checks as the brain commands; a broken entry refuses every write.` anhängen. Zeile „Brain Data Commands“: `A registry or area declaration loomux cannot use refuses the call and names the file, entry and reason.` anhängen.

In `README.de.md` dieselben Zeilen: „Registry und Bereichsdeklarationen laufen durch dieselben Prüfungen wie die Brain-Befehle; ein kaputter Eintrag verweigert jeden Write.“ und „Eine Registry oder Bereichsdeklaration, die loomux nicht verwenden kann, verweigert den Aufruf und nennt Datei, Eintrag und Grund.“

In der Spec `**Stand:** entworfen, nicht umgesetzt` → `**Stand:** umgesetzt (Plan 2026-09-16-loomux-registry-manifest-pruefungen.md), Paritätszeilen offen`.

- [ ] **Step 8: Gesamtes Gate und Commit**

Nachricht in `$TEMP/loomux-registry-bench/msg-task6.txt`:

```
Measure and record the registry and declaration checks

Adds the before and after measurement of the barrier and brain catalog on
the real registry, the parity list of the deviations the checks bring,
and the notes in both READMEs.
```

```bash
git branch --show-current
git log -1 --oneline
git add testdata/bench/registry-checks.json docs/.superpowers/parity/registry-manifest-pruefungen.md docs/.superpowers/parity/stufe-1b-1.md docs/en/benchmarks.md docs/de/benchmarks.md README.md README.de.md docs/.superpowers/specs/2026-09-16-loomux-registry-manifest-pruefungen-design.md
git commit -F "$TEMP/loomux-registry-bench/msg-task6.txt"
git log -8 --format='%h %an <%ae> | %s'
```

Expected: Gate grün. Nach Spec und Plan fünf Commits (Tasks 2–6), alle mit dem Nutzer als Autor, keiner mit Modell im Text. Nicht pushen.
