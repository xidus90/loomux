# loomux Stufe 4e Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die Umstellung der vier Wirte vorbereiten und danach die Lese-Rückfälle auf `ultra-brain` entfernen: erst ein nur lesendes `loomux area check`, dann die Checkliste des Menschen, dann der Aufräum-PR.

**Architecture:** `area check` zerlegt ein altes Manifest in Schlüssel und legt sie gegen `config.DeclarationKeys()` und `schema.Keys()`; der Leser `ReadDeclaration` bleibt unverändert. Der Aufräum-PR entfernt `LegacyBrainDirUntilStage3` und `ReadAreaManifestUntilStage4`; ein erschlagener `ReplaceDir`-Tausch wird beim Lesen aufgelöst (`ResolvedAreaDir` liefert das Aside), nicht zurückgeschoben.

**Tech Stack:** Go, `github.com/BurntSushi/toml` (das Modul aus `third_party/toml`), die vorhandenen Pakete `internal/config`, `internal/config/schema`, `internal/lock`, `internal/cli`.

**Spec:** `docs/.superpowers/specs/2026-09-28-loomux-stufe-4e-design.md`; Fusions-Spec Nachtrag #25.

## Global Constraints

- Nichts wird auf `master` gearbeitet; Zweig `docs/stage-4e-spec` (Spec, Plan und A landen auf einem Zweig, kein Plan-PR allein). C ist ein eigener PR, erst nach der Checkliste.
- Code, Bezeichner, Kommentare, Meldungen und Commit-Texte englisch; Arbeitspapiere unter `docs/.superpowers/` deutsch.
- Coverage 100 % je Funktion; ein Ausschluss nur mit `//coverage:exempt <reason>` direkt über `func`.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst.
- Ein Commit je Änderung, Conventional Commits, kein Arbeitspapier im Text (kein „4e“, „Task“, „Plan“), kein Modell als Mitautor. Der Mensch pusht; der Agent nennt den Befehl.
- Migrationsplan (`docs/en/migration.md`, `docs/de/migration.md`), Roadmap und beide READMEs ziehen im selben PR mit; `internal/plancheck` hält sie gegen die Spec.
- Messungen kommen chronologisch in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`.
- Schreibende Shell-Zeilen mit Backslashes nie im Bash-Heredoc: Quelltext per Write/Edit.
- Der Agent legt keine Konfiguration in `.loomux/config.toml` ab und ändert kein Manifest eines Bereichs; er liest nur.

## Review Focus

- Ein Manifest ohne `[area]` unter `.loomux/config.toml` ist Policy, kein Fehler; unter `.brain.toml` ist es ein abgelehntes Manifest.
- Ein Schlüssel auf Tiefe drei (`llm.local.enabled`) wird als ein Eintrag `llm.local` gemeldet, nicht dreimal.
- Ein Manifest mit zwei Namen (`.loomux/config.toml` und `.brain.toml`) meldet beide und nennt, welcher gewinnt; der andere ist verdeckt.
- Eine Datei, die kein TOML ist, ist ein Befund (Exit 1), kein Absturz.
- `area check` legt nichts an und ändert nichts (Verzeichnis vorher gegen nachher).
- `ResolvedAreaDir` mit Aside und Ziel liefert das Ziel; mit Aside ohne Ziel das Aside; ein schreibbarer Bereich kennt kein Aside.
- Nach dem Entfernen der Rückfälle liest kein Test mehr das Altverzeichnis.

---

## Dateien

| Datei | Verantwortung |
|---|---|
| `internal/cli/areacheck.go` (neu) | `areaCheck`, `classifyKeys`, `flattenKeys`, Ausgabe |
| `internal/cli/areacheck_test.go` (neu) | Tests dazu |
| `internal/cli/area.go` | Dispatcher: Unterbefehl `check`, `areaUsage` |
| `docs/.superpowers/parity/stufe-4e.md` (neu) | Messungen und die Checkliste des Menschen |
| `docs/en/cli-reference.md`, `docs/de/cli-reference.md` | Befehl `area check` |
| C: `internal/config/artifacts.go`, `legacy.go` und die Aufrufer | Rückfälle, Aside |
| C: `internal/brain/search/stamp.go`, `internal/brain/status/status.go`, `internal/brain/graph/read.go` | Ratschläge |

---

## Stück A: `loomux area check`

### Task 1: Messungen, die den Rest bestimmen (kein Produktcode)

Ergebnisse gehen in `docs/.superpowers/parity/stufe-4e.md`, Abschnitt „Messungen“, mit Datum. Die Datei wird in diesem Task angelegt.

**Files:**
- Create: `docs/.superpowers/parity/stufe-4e.md`

- [ ] **Step 1: `init --dry-run` gegen ein `[area]` in `.loomux/config.toml`**

Auf einer Wegwerfkopie (`git worktree add <scratchpad>/probe HEAD`), nicht im Arbeitsbaum. Ein `[area] scope = "probe/x"` von Hand an `.loomux/config.toml` anhängen, dann:

Ein Aufruf, eine schlichte Zeile, ohne Pipe, Umleitung und `cd &&` (der Wächter lässt die Nur-Lese-Form nur so durch); die Ausgabe kommt aus dem Tool-Ergebnis:

```sh
go run ./cmd/loomux init --dry-run --yes
```

Festhalten: bleibt der `[area]`-Block in der Ausgabe erhalten oder wird die Datei ersetzt? Ergebnis in die Akte; es bestimmt die Reihenfolge der Blöcke 4 und 5 der Checkliste (Task 4).

- [ ] **Step 2: die vier Manifeste, die die Probe vom 2026-09-28 nicht deckte**

Die zehn Bereiche stehen in der alten Registry `%LOCALAPPDATA%\brain\registry.toml` (die neue unter `%LOCALAPPDATA%\loomux` ist eine Vorlage mit einem Eintrag; `loomux area list` gibt es nicht). Gelesen sind `brain-knowledge`, `ecoflow`, `iam_wiki`, `space`, `ultra-brain`, `ultraloom`. Offen sind vier; ihre Pfade aus der Registry lesen, nur lesen, nichts anfassen. (Die Klassen je Schlüssel misst danach Task 3 selbst; hier steht nur, welche Dateien es gibt.)

- [ ] **Step 3: die Leser des Bestands, die an `ResolvedAreaDir` vorbeigehen**

```sh
git grep -n "config.ManifestDir(" -- 'internal/**/*.go' ':!*_test.go'
git grep -n "ResolvedAreaDir(" -- 'internal/**/*.go' ':!*_test.go'
```

Je Fundstelle in die Akte: öffnet sie den Bereichsordner für **Lesen** (braucht das Aside) oder für **Schreiben/Tausch** (braucht es nicht)? Erwartet laut Spec: `apply/resolve.go`, `apply/stock.go`, `guard/registry.go`, `index/staging.go`, `config/registry.go`, `dev/benchsearch/corpusrun.go`.

- [ ] **Step 4: `Manifest.Lanes` und `moveStock`**

```sh
git grep -n "\.Lanes\b" -- 'internal/**/*.go' ':!*_test.go'
```

Festhalten, wer `Manifest.Lanes` außer `ReadManifest` liest (am 2026-09-28 gelesen: niemand; die `verify`-Treffer sind andere Typen), und was `internal/brain/apply/stock.go` außer der Kopie aus dem Altverzeichnis tut (Aufruf von `lock.Recover`, `registerWrite`, `beforeSwap`).

- [ ] **Step 4a: Deckt `DeclarationKeys ∪ schema.Keys` jeden Leser? (Kernbehauptung von `area check`)**

```sh
git grep -n 'toml:"\|toml.Unmarshal\|toml.Decode\|PrimitiveDecode' -- 'internal/**/*.go' ':!*_test.go'
```

Je Fundstelle, die einen Bereichs- oder Projekt-Manifest liest (`config/manifest.go`, `declaration.go`, `policy.go`, `agent.go`, `modules.go`, `flowsettings.go`, `modelsettings.go`, `searchsettings.go`): welche Abschnitte und Schlüssel? Differenz gegen `DeclarationKeys() ∪ schema.Keys()` in die Akte. Am 2026-09-28 gemessen: `ReadManifest` liest `[check] lanes` in `Manifest.Lanes`, das kein Verbraucher nutzt; `[layout] sources` schreibt `area add` (`internal/cli/area.go:283`), aber kein Leser liest es. Beide sind für den Menschen wirkungslos und bekommen in Task 2 einen Hinweis in `legacyHints`. Jede weitere Differenz wird ein Eintrag dort (wenn der Schlüssel wirkungslos ist) oder eine dritte Tabelle in `classifyKeys` (wenn ein Leser ihn wirklich liest).

- [ ] **Step 5: die sechs offenen Entscheidungen aus `parity/artefakte-nach-lebensdauer.md`**

Lesen. Je Entscheidung: berührt sie Zustandsdateien unter `%LOCALAPPDATA%\loomux` oder das Manifest? Wenn ja, steht sie vor der Checkliste (Block 3 oder 4); sonst nicht.

- [ ] **Step 6: Akte schreiben und committen**

Die Akte hat die Abschnitte „Messungen“ (Steps 1–5, mit Datum und `loomux --version`) und „Checkliste“ (Task 4 füllt sie). Committen:

```sh
git add docs/.superpowers/parity/stufe-4e.md
git commit -F <scratchpad>/msg.txt   # docs(migration): record what the host switch-over rests on
```

### Task 2: `classifyKeys` — Schlüssel gegen die zwei Tabellen

**Files:**
- Create: `internal/cli/areacheck.go`
- Test: `internal/cli/areacheck_test.go`

**Interfaces:**
- Produces:
  - `type keyClass string` mit den Werten `classRead`, `classElsewhere`, `classIgnored`.
  - `type keyReport struct { ID string; Class keyClass; Hint string }`.
  - `func flattenKeys(document map[string]any) []string` — sortierte IDs bis Tiefe zwei.
  - `func classifyKeys(document map[string]any) []keyReport`.

- [ ] **Step 1: den Test schreiben, der rot läuft**

`internal/cli/areacheck_test.go`:

```go
package cli

import (
	"reflect"
	"testing"
)

func TestFlattenKeysStopsAtDepthTwo(t *testing.T) {
	document := map[string]any{
		"top": "x",
		"area": map[string]any{"scope": "s"},
		"llm": map[string]any{"local": map[string]any{"enabled": false}},
		"relevance": []map[string]any{{"a": "b"}},
		"empty": map[string]any{},
	}
	got := flattenKeys(document)
	want := []string{"area.scope", "empty", "llm.local", "relevance", "top"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("flattenKeys = %v, want %v", got, want)
	}
}

func TestClassifyKeysSeparatesReadElsewhereAndIgnored(t *testing.T) {
	document := map[string]any{
		"area":    map[string]any{"scope": "knowledge"},
		"layout":  map[string]any{"inbox": "00", "sources": "10"},
		"commit":  map[string]any{"language": "de"},
		"llm":     map[string]any{"local": map[string]any{"enabled": false}},
		"nonsense": map[string]any{"k": int64(1)},
	}
	got := map[string]keyReport{}
	for _, r := range classifyKeys(document) {
		got[r.ID] = r
	}
	cases := map[string]keyClass{
		"area.scope":      classRead,
		"layout.inbox":    classRead,
		"layout.sources":  classIgnored, // the reader reads no `sources`
		"commit.language": classElsewhere,
		"llm.local":       classIgnored,
		"nonsense.k":      classIgnored,
	}
	for id, class := range cases {
		if got[id].Class != class {
			t.Errorf("%s: class %q, want %q", id, got[id].Class, class)
		}
	}
	if got["llm.local"].Hint == "" {
		t.Errorf("llm.local carries no hint")
	}
	if got["nonsense.k"].Hint != "" {
		t.Errorf("nonsense.k carries hint %q", got["nonsense.k"].Hint)
	}
	if got["layout.sources"].Hint == "" {
		t.Errorf("layout.sources carries no hint")
	}
}

func TestClassifyKeysKnowsTheVerifyTablesTheSchemaDoesNotName(t *testing.T) {
	// internal/verify reads [verify.<stack>.<kind>] and
	// [verify.gdscript] import_check; the schema names neither.
	document := map[string]any{"verify": map[string]any{"go": map[string]any{"lint": "x"}, "gdscript": map[string]any{"import_check": "y"}}}
	for _, r := range classifyKeys(document) {
		if r.Class == classIgnored {
			t.Errorf("%s reported ignored", r.ID)
		}
	}
}

func TestClassifyKeysCallsALanesTableIgnoredWithAHint(t *testing.T) {
	// ReadManifest decodes [check] lanes into Manifest.Lanes, which nothing
	// uses: a move by hand loses nothing, but no reader acts on it either.
	reports := classifyKeys(map[string]any{"check": map[string]any{"lanes": []any{}}})
	if len(reports) != 1 || reports[0].Class != classIgnored || reports[0].Hint == "" {
		t.Fatalf("got %+v", reports)
	}
}

func TestClassifyKeysKnowsATableTheSchemaOnlyNamesBelow(t *testing.T) {
	// `policy.paths` is not a key of its own; the schema knows
	// `policy.paths.rules`.
	document := map[string]any{"policy": map[string]any{"paths": map[string]any{}}}
	reports := classifyKeys(document)
	if len(reports) != 1 || reports[0].Class != classElsewhere {
		t.Fatalf("got %+v, want one elsewhere entry", reports)
	}
}
```

- [ ] **Step 2: rot laufen lassen**

Run: `go test ./internal/cli -run 'TestFlattenKeys|TestClassifyKeys' -count=1`
Expected: FAIL (`undefined: flattenKeys`, `classifyKeys`; Build-Fehler ist hier der erwartete erste Rot-Lauf).

- [ ] **Step 3: minimal implementieren**

`internal/cli/areacheck.go`:

```go
package cli

import (
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/schema"
)

// keyClass says what the readers do with one key of an area declaration.
type keyClass string

const (
	// classRead: config.ReadDeclaration reads the key.
	classRead keyClass = "read"
	// classElsewhere: another reader of the same file reads it (`[commit]`,
	// `[verify]`), so moving the file loses nothing.
	classElsewhere keyClass = "elsewhere"
	// classIgnored: no reader reads it, and a move by hand would drop it
	// without a word.
	classIgnored keyClass = "ignored"
)

// keyReport is one key of a manifest and what becomes of it.
type keyReport struct {
	ID    string
	Class keyClass
	Hint  string
}

// legacyHints name where an old key belongs in today's schema. A hint only:
// nothing here translates a value.
var legacyHints = map[string]string{
	"llm.local":      "the local model is [model] enabled and roles",
	"layout.sources": "written by `area add`, read by no reader",
	"check.lanes":    "read into a field nothing uses; lanes live under [verify]",
	"area.wiki":      "written by `area add`, read by no reader; the wiki place is [layout] wiki",
	"maintenance.merge_branch": "written by ultra-brain, read by no reader; loomux reads [maintenance] branch",
}

// flattenKeys lists the keys of a decoded manifest as dotted IDs, sorted:
// `section.key` for a table's entries, the table itself for anything deeper
// or a list of tables, the bare name for a top-level value.
func flattenKeys(document map[string]any) []string {
	var ids []string
	for name, value := range document {
		table, isTable := value.(map[string]any)
		if !isTable {
			ids = append(ids, name)
			continue
		}
		if len(table) == 0 {
			ids = append(ids, name)
			continue
		}
		for key := range table {
			ids = append(ids, name+"."+key)
		}
	}
	slices.Sort(ids)
	return ids
}

// classifyKeys sorts every key of a manifest into read, elsewhere or ignored,
// with the tables the readers already name: config.DeclarationKeys for the
// declaration and the schema for the rest of `.loomux/config.toml`.
func classifyKeys(document map[string]any) []keyReport {
	read := map[string]bool{}
	for section, keys := range config.DeclarationKeys() {
		for _, key := range keys {
			read[section+"."+key] = true
		}
	}
	var reports []keyReport
	for _, id := range flattenKeys(document) {
		switch {
		case read[id]:
			reports = append(reports, keyReport{ID: id, Class: classRead})
		case schemaKnows(id):
			reports = append(reports, keyReport{ID: id, Class: classElsewhere})
		default:
			reports = append(reports, keyReport{ID: id, Class: classIgnored, Hint: legacyHints[id]})
		}
	}
	return reports
}

// schemaKnows says whether some key of the schema is id or lies below it, or
// whether a reader outside the schema reads it: internal/verify decodes every
// `[verify.<stack>.<kind>]` table and `[verify.gdscript] import_check`, which
// the schema does not list by name.
func schemaKnows(id string) bool {
	if id == "verify" || strings.HasPrefix(id, "verify.") {
		return true
	}
	if _, ok := schema.Lookup(id); ok {
		return true
	}
	for _, key := range schema.Keys() {
		if strings.HasPrefix(key.ID(), id+".") {
			return true
		}
	}
	return false
}
```

Hinweis für den Implementierer: `flattenKeys` legt einen leeren Tisch (`[area]` ohne Schlüssel) als einen Eintrag ab; der Test `TestClassifyKeysKnowsATableTheSchemaOnlyNamesBelow` deckt genau diesen Pfad (`policy.paths` ist ein leerer Tisch). Die zwei `append`-Zweige für Nicht-Tisch und leeren Tisch dürfen zusammengefasst werden, wenn Coverage und Tests es zulassen.

- [ ] **Step 4: grün laufen lassen**

Run: `go test ./internal/cli -run 'TestFlattenKeys|TestClassifyKeys' -count=1`
Expected: PASS

- [ ] **Step 5: eine Regel entfernen und prüfen, dass genau ein Test rot wird**

In `classifyKeys` den Zweig `case schemaKnows(id):` auskommentieren (per `go test -overlay`, ohne den Baum anzufassen): `commit.language` und `policy.paths` müssen rot werden. Dann die Zeile `Hint: legacyHints[id]` entfernen: `llm.local` muss rot werden.

- [ ] **Step 6: Commit**

```sh
git add internal/cli/areacheck.go internal/cli/areacheck_test.go
git commit -F <scratchpad>/msg.txt   # feat(cli): sort the keys of an area manifest by what the readers do with them
```

### Task 3: der Befehl `loomux area check <pfad>`

**Files:**
- Modify: `internal/cli/area.go` (Dispatcher und `areaUsage`)
- Modify: `internal/cli/areacheck.go`
- Test: `internal/cli/areacheck_test.go`

**Interfaces:**
- Consumes: `classifyKeys`, `keyReport` (Task 2); `config.ReadDeclaration`, `config.ErrNoArea`, `config.ReadAreaManifestUntilStage4`.
- Produces: `func areaCheck(args []string, stdout, stderr io.Writer) int`; Ausgabezeile `<name>: <id>  <class>  <hint>`, Exit 0/1/2.

Zu prüfende Namen: `.loomux/config.toml`, `.ultra-brain/config.toml`, `.brain.toml` (in dieser Reihenfolge, wie der Leser sie probiert).

Regeln:

- Kein Argument, mehr als eines oder ein Pfad, der kein Verzeichnis ist: Meldung `usage: loomux area check <path>` auf stderr, Exit 2.
- Je vorhandene reguläre Datei: die Schlüsselzeilen; ein Manifest, das der Leser ablehnt, eine Zeile `<name>: refused: <Meldung>`. `ErrNoArea` unter `.loomux/config.toml` ist kein Befund (`<name>: no [area], policy only`); unter den zwei alten Namen ist es abgelehnt.
- Eine Datei, die kein TOML ist: `<name>: refused: not valid TOML: …`, Exit 1.
- Eine Zeile `chosen: <name>`, welche Datei `ReadAreaManifestUntilStage4` heute wählt (oder `chosen: none`); jede andere Datei, die ein `[area]` trägt, bekommt eine Zeile `<name>: shadowed`. Eine Datei ohne `[area]` (Policy) ist nie verdeckt.
- Exit 1, wenn ein Schlüssel ignoriert oder ein Manifest abgelehnt ist oder keine Datei da ist; sonst 0.
- Der Befehl schreibt nichts.

- [ ] **Step 1: die Tests schreiben, die rot laufen**

Ergänze `areacheck_test.go` um Tests mit einem `t.TempDir()`, in das die Dateien geschrieben werden (Write-Werkzeug bzw. `os.WriteFile` im Test, nicht per Shell):

```go
func runAreaCheck(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := areaCheck(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func writeManifest(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAreaCheckAcceptsAFullyReadManifest(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".loomux/config.toml", "[area]\nscope = \"x\"\n")
	code, out, _ := runAreaCheck(t, root)
	if code != 0 || !strings.Contains(out, "area.scope  read") || !strings.Contains(out, "chosen: "+filepath.Join(".loomux", "config.toml")) {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckFailsOnAnIgnoredKeyAndNamesIt(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".brain.toml", "[area]\nscope = \"x\"\n\n[llm.local]\nenabled = false\n")
	code, out, _ := runAreaCheck(t, root)
	if code != 1 || !strings.Contains(out, "llm.local  ignored") || !strings.Contains(out, "[model]") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckRefusesAnOldNameWithoutScope(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".brain.toml", "[layout]\ninbox = \"00\"\n")
	code, out, _ := runAreaCheck(t, root)
	if code != 1 || !strings.Contains(out, ".brain.toml: refused:") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckTreatsPolicyOnlyLoomuxConfigAsFine(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".loomux/config.toml", "[commit]\nlanguage = \"en\"\n")
	writeManifest(t, root, ".brain.toml", "[area]\nscope = \"x\"\n")
	code, out, _ := runAreaCheck(t, root)
	if code != 0 || !strings.Contains(out, "no [area], policy only") || !strings.Contains(out, "chosen: .brain.toml") || strings.Contains(out, "config.toml: shadowed") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckSaysWhichNameShadowsTheOther(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".loomux/config.toml", "[area]\nscope = \"x\"\n")
	writeManifest(t, root, ".brain.toml", "[area]\nscope = \"x\"\n")
	_, out, _ := runAreaCheck(t, root)
	if !strings.Contains(out, ".brain.toml: shadowed") {
		t.Fatalf("out %q", out)
	}
}

func TestAreaCheckReportsAFileThatIsNotTOML(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".brain.toml", "[area\n")
	code, out, _ := runAreaCheck(t, root)
	if code != 1 || !strings.Contains(out, "not valid TOML") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckFailsWhenNoManifestIsThere(t *testing.T) {
	code, out, _ := runAreaCheck(t, t.TempDir())
	if code != 1 || !strings.Contains(out, "chosen: none") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckRefusesWrongArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"a", "b"}, {filepath.Join(t.TempDir(), "missing")}} {
		code, _, errOut := runAreaCheck(t, args...)
		if code != 2 || !strings.Contains(errOut, "usage: loomux area check <path>") {
			t.Fatalf("args %v: code %d, stderr %q", args, code, errOut)
		}
	}
}

func TestAreaCheckWritesNothing(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".brain.toml", "[area]\nscope = \"x\"\n")
	before := snapshotTree(t, root)
	runAreaCheck(t, root)
	if after := snapshotTree(t, root); !reflect.DeepEqual(before, after) {
		t.Fatalf("the tree changed: %v -> %v", before, after)
	}
}

// snapshotTree lists every path under root with its size and modification time.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	seen := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		seen[path] = fmt.Sprintf("%d %d", info.Size(), info.ModTime().UnixNano())
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return seen
}

func TestAreaCheckIsReachableAsAnAreaSubcommand(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".loomux/config.toml", "[area]\nscope = \"x\"\n")
	var stdout, stderr bytes.Buffer
	if code := areaCommand([]string{"check", root}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("code %d, stderr %q", code, stderr.String())
	}
}
```

(`bytes`, `fmt`, `io/fs`, `os`, `path/filepath`, `strings` importieren.)

- [ ] **Step 2: rot laufen lassen**

Run: `go test ./internal/cli -run 'TestAreaCheck' -count=1`
Expected: FAIL (`undefined: areaCheck`, dann `usage` statt `check`).

- [ ] **Step 3: implementieren**

In `areacheck.go`:

```go
// readManifest and the manifest names are variables so that a test can make
// the read fail.
var readManifest = os.ReadFile

var manifestNames = []string{
	filepath.Join(".loomux", "config.toml"),
	filepath.Join(".ultra-brain", "config.toml"),
	".brain.toml",
}

const areaCheckUsage = "usage: loomux area check <path>"

// areaCheck is `loomux area check`: for the manifests of one area root, what
// the readers do with each key. It reads and writes nothing else.
func areaCheck(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, areaCheckUsage)
		return 2
	}
	root := args[0]
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		fmt.Fprintln(stderr, areaCheckUsage)
		return 2
	}
	chosen := ""
	if manifest, err := config.ReadAreaManifestUntilStage4(root); err == nil {
		chosen, _ = filepath.Rel(root, manifest.Path)
	}
	needsHand := false
	found := false
	for _, name := range manifestNames {
		path := filepath.Join(root, name)
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			continue
		}
		found = true
		if checkOneManifest(stdout, root, name, name == manifestNames[0]) {
			needsHand = true
		}
		if chosen != "" && name != chosen && declaresArea(path) {
			fmt.Fprintf(stdout, "%s: shadowed\n", name)
		}
	}
	if chosen == "" {
		fmt.Fprintln(stdout, "chosen: none")
	} else {
		fmt.Fprintf(stdout, "chosen: %s\n", chosen)
	}
	if !found || needsHand {
		return 1
	}
	return 0
}

// declaresArea says whether the file carries an [area] table.
func declaresArea(path string) bool {
	_, err := config.ReadDeclaration(path)
	return !errors.Is(err, config.ErrNoArea)
}

// checkOneManifest prints the keys of one manifest and says whether it needs
// a hand: a key no reader reads, or a manifest a reader refuses.
func checkOneManifest(stdout io.Writer, root, name string, policyOnlyIsFine bool) bool {
	path := filepath.Join(root, name)
	data, err := readManifest(path)
	if err != nil {
		fmt.Fprintf(stdout, "%s: refused: %v\n", name, err)
		return true
	}
	document := map[string]any{}
	if err := toml.Unmarshal(data, &document); err != nil {
		fmt.Fprintf(stdout, "%s: refused: not valid TOML: %v\n", name, err)
		return true
	}
	needsHand := false
	for _, report := range classifyKeys(document) {
		fmt.Fprintf(stdout, "%s: %s  %s  %s\n", name, report.ID, report.Class, report.Hint)
		if report.Class == classIgnored {
			needsHand = true
		}
	}
	if _, err := config.ReadDeclaration(path); errors.Is(err, config.ErrNoArea) && policyOnlyIsFine {
		fmt.Fprintf(stdout, "%s: no [area], policy only\n", name)
	} else if err != nil {
		fmt.Fprintf(stdout, "%s: refused: %v\n", name, err)
		needsHand = true
	}
	return needsHand
}
```

Die Ausgabe schreibt `area.scope  read  ` mit zwei Leerzeichen und einem leeren Hinweis; die Tests suchen `area.scope  read` als Präfix. Imports: `errors`, `fmt`, `io`, `os`, `path/filepath`, `github.com/BurntSushi/toml`.

In `area.go`:

```go
func areaCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "check" {
		return areaCheck(args[1:], stdout, stderr)
	}
	if len(args) == 0 || args[0] != "add" {
		fmt.Fprintln(stderr, areaUsage)
		return 2
	}
	return areaAdd(args[1:], stdin, stdout, stderr)
}
```

und `areaUsage` bekommt `"loomux area check <path>"` als zweite Zeile.

- [ ] **Step 4: grün, dann die Nachbartests**

Run: `go test ./internal/cli -count=1`
Expected: PASS (auch die bestehenden Tests, die `areaUsage` prüfen; falls einer den alten Wortlaut erwartet, passt der Test an, nicht der Befehl).

- [ ] **Step 5: eine Regel entfernen, je Test rot**

Per `-overlay`: (a) `policyOnlyIsFine` immer `false` setzen — `TestAreaCheckTreatsPolicyOnlyLoomuxConfigAsFine` rot; (b) die `shadowed`-Zeile entfernen — `TestAreaCheckSaysWhichNameShadowsTheOther` rot; (c) die Prüfung `IsRegular` weglassen — dafür gehört dieser Test in Step 1 dazu und läuft dagegen rot:

```go
func TestAreaCheckSkipsADirectoryNamedLikeAManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".brain.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runAreaCheck(t, root)
	if code != 1 || strings.Contains(out, ".brain.toml:") || !strings.Contains(out, "chosen: none") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckReportsAManifestThatCannotBeRead(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".brain.toml", "[area]\nscope = \"x\"\n")
	original := readManifest
	t.Cleanup(func() { readManifest = original })
	readManifest = func(string) ([]byte, error) { return nil, errors.New("denied") }
	code, out, _ := runAreaCheck(t, root)
	if code != 1 || !strings.Contains(out, ".brain.toml: refused: denied") {
		t.Fatalf("code %d, out %q", code, out)
	}
}
```

(`errors` importieren.)

- [ ] **Step 6: Selbstnutzung an den echten Bereichen**

Je Bereich ein eigener Aufruf, eine schlichte Zeile ohne Schleife, Umleitung und `cd &&`; die Ausgabe kommt aus dem Tool-Ergebnis:

```sh
go build -o bin/loomux.exe ./cmd/loomux
```
```sh
./bin/loomux.exe area check "C:/Users/micro/Documents/#GIT/brain-knowledge"
```

(und ebenso für die übrigen Bereiche). Nur lesend. Verweigert der loomux-Wächter `area check` für Agenten (er kennt `area add` als Schreiber), ist das ein Befund: der Implementierer nennt dem Controller die Zeile, der Nutzer führt sie mit `!` aus, und die Wächterregel bekommt einen eigenen kleinen Fix-Commit (`area check` ist nur lesend). Ausgabe in die Akte (Abschnitt „Messungen“), samt den Klassen je Schlüssel. Die vier weiteren Bereiche aus Task 1 Step 2 folgen. Hat ein Manifest Schlüssel, die die Spec-Tabelle `legacyHints` nicht kennt, aber ein Gegenstück haben (etwa `maintenance.watch`), ergänzt der Implementierer die Tabelle und einen Testfall.

- [ ] **Step 7: Doku und Commit**

`docs/en/cli-reference.md` und `docs/de/cli-reference.md`: der Befehl, seine Ausgabe, die drei Exit-Codes, der Hinweis „nur lesend; entfällt mit dem Rückfall“. `README.md` und `README.de.md`, falls sie die `area`-Befehle aufzählen.

```sh
git add internal/cli docs
git commit -F <scratchpad>/msg.txt   # feat(cli): add `area check`, a read-only report on an old area manifest
```

### Task 4: die Checkliste und der Stand der Stufe

**Files:**
- Modify: `docs/.superpowers/parity/stufe-4e.md` (Abschnitt „Checkliste“)
- Modify: `docs/en/migration.md`, `docs/de/migration.md`, `README.md`, `README.de.md`

- [ ] **Step 1: die Checkliste schreiben**

Die Blöcke aus der Spec (Abschnitt „B“), in dieser Reihenfolge, mit `- [ ]` je Schritt und dem genauen Bash-Befehl, wo ein Befehl nötig ist. Die Reihenfolge der Blöcke 4 und 5 folgt dem Ergebnis von Task 1 Step 1 (erhält `init` ein von Hand eingetragenes `[area]`: erst 4, dann 5; sonst erst 5, dann 4 mit dem Hinweis, dass `init` die Datei ersetzt). Vorwärtsschrägstriche, `/c/…`-Pfade; eine nötige Antwort per Pipe.

- [ ] **Step 2: Migrationsplan, README, Roadmap**

Die 4e-Zeile in `migration.md` (en/de) auf den Stand „A gebaut, Checkliste offen, Aufräum-PR wartet“ bringen; Priorität bleibt 3, der Status bleibt „offen“ (nicht ✅). README und Roadmap nachziehen. Dann `go run ./cmd/loomux check precommit` (das Gate hält `internal/plancheck` gegen die Spec).

- [ ] **Step 3: Commit**

```sh
git add docs README.md README.de.md
git commit -F <scratchpad>/msg.txt   # docs(migration): add the checklist for switching the hosts over
```

### Task 5: Mutationsrunde für A und das Ende des PR

- [ ] **Step 1:** `loomux dev mutants` über `internal/cli/areacheck.go` laufen lassen (Aufruf und Ausgabe wie in den vorigen Akten, `parity/stufe-4a-2.md`). Überlebende Mutanten: erst Tests nachlegen, was bleibt, mit Begründung in den Abschnitt „Überlebende Mutanten“ der Akte.
- [ ] **Step 2:** Die Commits gruppieren und den PR mit dem Skill `release-pr` öffnen (Label `release:minor`: ein neuer Befehl; der Changelog-Block nennt `area check`). Der Mensch pusht.

---

## Stück B: die Checkliste (Mensch)

Der Mensch hakt `parity/stufe-4e.md` ab. Vor Stück C müssen die Blöcke „Maschinenzustand“ und „Deklarationen“ erledigt sein; „je Wirt“ und „Rauchtest“ dürfen parallel zu C laufen, müssen aber vor ✅ stehen. Der Agent hakt nichts ab und legt in keinem Bereich eine Deklaration an.

---

## Stück C: der Aufräum-PR (nach der Checkliste, eigener Zweig von `master`)

Jeder Task beginnt mit `git fetch` und einem Rebase auf `origin/master`.

### Task 6: `ResolvedAreaDir` löst ein Aside auf

**Files:**
- Modify: `internal/config/artifacts.go` (`ResolvedAreaDir`)
- Test: `internal/config/artifacts_test.go`

**Interfaces:**
- Consumes: `lock.AsideSuffix` (`internal/lock`, hat keine Importe aus dem Projekt, also kein Zyklus).
- Produces: `ResolvedAreaDir` bleibt mit unveränderter Signatur; ein read-only-Bereich, dessen Zielordner fehlt und dessen `<ziel>.loomux-aside` da ist, liefert den Aside-Pfad.

Vorher prüfen: Task 1 Step 3 hat gesagt, welche Leser diesen Weg gar nicht gehen (Task 7).

- [ ] **Step 1: die drei diskriminierenden Tests schreiben**

```go
func TestResolvedAreaDirPrefersTheTargetOverItsAside(t *testing.T) {
	state := t.TempDir()
	area := Area{Scope: "space", ReadOnly: true}
	target := ManifestDir(area, state)
	makeDirs(t, target, target+lock.AsideSuffix)
	if got := ResolvedAreaDir(area, state, ""); got != target {
		t.Fatalf("got %q, want the target %q", got, target)
	}
}

func TestResolvedAreaDirFallsBackToTheAsideWhenTheTargetIsGone(t *testing.T) {
	state := t.TempDir()
	area := Area{Scope: "space", ReadOnly: true}
	target := ManifestDir(area, state)
	makeDirs(t, target+lock.AsideSuffix)
	if got := ResolvedAreaDir(area, state, ""); got != target+lock.AsideSuffix {
		t.Fatalf("got %q, want the aside", got)
	}
}

func TestResolvedAreaDirWithNeitherAnswersTheTarget(t *testing.T) {
	state := t.TempDir()
	area := Area{Scope: "space", ReadOnly: true}
	if got := ResolvedAreaDir(area, state, ""); got != ManifestDir(area, state) {
		t.Fatalf("got %q", got)
	}
}

func TestResolvedAreaDirOfAWritableAreaKnowsNoAside(t *testing.T) {
	root := t.TempDir()
	makeDirs(t, root+lock.AsideSuffix)
	area := Area{Scope: "x", Path: root}
	if got := ResolvedAreaDir(area, t.TempDir(), ""); got != root {
		t.Fatalf("got %q", got)
	}
}
```

(`makeDirs` ist ein kleiner Helfer mit `os.MkdirAll`; `lock` importieren.)

- [ ] **Step 2: rot laufen lassen**

Run: `go test ./internal/config -run TestResolvedAreaDir -count=1`
Expected: FAIL im zweiten Test (`got` ist das Ziel, nicht das Aside).

- [ ] **Step 3: implementieren**

`ResolvedAreaDir` bekommt nach dem `lookup.Resolve` den Aside-Zweig; hier noch mit dem Rückfall, den Task 8 entfernt:

```go
	resolved := lookup.Resolve(filepath.Join("areas", flat(area.Scope)))
	if _, err := os.Stat(resolved); err == nil {
		return resolved
	}
	aside := resolved + lock.AsideSuffix
	if _, err := os.Stat(aside); err == nil {
		return aside
	}
	return resolved
```

Der Kommentar an der Funktion sagt: ein Leser schreibt nichts (ein `lock.Recover` im Lesepfad würde einem laufenden `ReplaceDir` das Aside wegnehmen), und nennt das Restfenster.

- [ ] **Step 4: grün, dann mit `-overlay` den Aside-Zweig entfernen: der zweite Test muss rot werden.**

- [ ] **Step 5: `VisibleAreas` antwortet nach einem erschlagenen Tausch**

In `internal/brain/privacy/areas_test.go` einen Fall ergänzen: eine Registry mit einem read-only-Bereich, dessen Ordner nur als `<ziel>.loomux-aside` mit einem gültigen `.loomux/config.toml` (oder dem Manifest, das der Leser dort findet) vorliegt; `VisibleAreas(..., "all", …)` antwortet ohne Fehler. Ohne Task 6 ist er rot.

- [ ] **Step 6: Kommentar an `ReplaceDir` und Absatz in `parity/stufe-3a.md`**

Der Satz „That fallback ends with stage 4, and the note in … says what has to be decided“ wird durch die getroffene Entscheidung ersetzt. In `stufe-3a.md` wird der Abschnitt „Stufe 4: der abwesende Bereichsordner nach einem erschlagenen Tausch“ gestrichen und die Entscheidung in die Abweichungstabelle der Stufe eingetragen, die sie trifft (Vorschrift dort, letzter Absatz des Abschnitts).

- [ ] **Step 7: Commit**

```sh
git commit -F <scratchpad>/msg.txt   # fix(config): read an area whose directory swap was killed from its aside
```

### Task 7: die Leser, die an `ResolvedAreaDir` vorbeigehen

Nur die Fundstellen aus Task 1 Step 3, die den Bereichsordner zum **Lesen** öffnen. Je Fundstelle ein Test, der erst rot läuft, dann die Änderung auf `config.ResolvedAreaDir(area, state, "")` (oder auf eine gemeinsame Hilfe daneben, wenn drei Stellen dasselbe brauchen). Schreibende Stellen (`index/staging.go`, `apply/stock.go`, wo sie tauschen) bleiben bei `ManifestDir`: sie sind Schreiber und rufen `lock.Recover` selbst.

Beispiel (`internal/brain/guard/registry.go:51`): der Test legt einen read-only-Bereich nur als Aside an und verlangt, dass die Schranke seine Deklaration findet; rot ohne die Änderung.

- [ ] Steps: je Fundstelle Test rot, Änderung, grün, `-overlay`-Mutation, ein Commit `fix(<scope>): …` je Fundstelle, die einen eigenen Fehler war; gleichartige Stellen in einem Commit.

### Task 8: die Rückfälle entfernen

**Files:** die Aufrufer laut `git grep`, `internal/config/legacy.go`, `internal/config/artifacts.go`, `internal/brain/apply/stock.go`.

- [ ] **Step 1: Bestand aufnehmen**

```sh
git grep -n "UntilStage3\|UntilStage4\|LOOMUX_LEGACY_BRAIN_DIR\|manifestNamesUntilStage4" > <scratchpad>/legacy-before.txt
git grep -n "loomux migrate" -- 'internal/**/*.go' > <scratchpad>/migrate-mentions.txt
```

- [ ] **Step 2: `ReadAreaManifestUntilStage4` durch `ReadDeclaration(filepath.Join(dir, ".loomux", "config.toml"))` ersetzen**

Die zwölf Aufrufer laut Spec. Der Fall „`.loomux/config.toml` ohne `[area]` ist Policy und fragt den nächsten Namen“ entfällt mit den alten Namen: `ErrNoArea` bedeutet dann „keine Deklaration“. Die Tests, die einen alten Namen erwarteten, werden entfernt oder auf `.loomux/config.toml` umgestellt; kein Test behält einen alten Manifestnamen außer `area check`.

- [ ] **Step 3: `LegacyBrainDirUntilStage3`, `LOOMUX_LEGACY_BRAIN_DIR`, `ArtifactLookup.Fallback` und `legacy.go` entfernen**

`ArtifactLookup` wird zur Hülle um einen Pfad oder fällt ganz weg (der Compiler zeigt die vier Aufrufer: `cli/brain.go`, `cli/dev.go`, `cli/serve.go`, `config/artifacts.go`). `serve.Options.LegacyDir` entfällt.

- [ ] **Step 4: `Manifest.Lanes` und `apply.moveStock`**

Nur entfernen, was Task 1 Step 4 als reinen Rückfall belegt hat. `Manifest.Lanes` hat außer `ReadManifest` keinen Leser (nur `manifest_test.go:465`): das Feld entfällt mit seinem Test. Von `moveStock` bleibt bei leerem Fallback allein `lock.Recover(target)`. **Dieses `lock.Recover` im Schreibpfad von `approve` (`internal/brain/apply/approve.go`, Register schreiben) bleibt**, oder Task 6 steht davor: fällt es weg, schriebe `approve` das Register in ein fehlendes Ziel, neben dem ein Aside einer erschlagenen `ReplaceDir` liegt, und ein späteres `Recover` löschte das Aside, also den alten Bestand. Ein Test stellt das nach (Aside, kein Ziel, `approve`) und darf nur grün sein, wenn der Aufruf bleibt.

- [ ] **Step 5: `area check` behält seinen Zweck**

`area check` liest die drei Namen selbst (`manifestNames`) und ruft `ReadAreaManifestUntilStage4` für die Zeile `chosen:`; diese Zeile bekommt eine eigene kleine Auswahl in `areacheck.go`, damit der Befehl den Rückfall überlebt, bis 4f ihn entfernt.

- [ ] **Step 6: Gate und Kommentare**

`go build ./... && go vet ./...`, dann `git grep -n "migrate"` gegen `migrate-mentions.txt`: jeder Kommentar, der `loomux migrate` als Ende nennt, ist nachgezogen (`internal/config/legacy.go` entfällt, `artifacts.go`, `internal/cli/index.go`, `internal/brain/graph/read.go`, `internal/brain/index/reindex.go`).

- [ ] **Step 7: Commit**

```sh
git commit -F <scratchpad>/msg.txt   # refactor(config): drop the fallbacks onto ultra-brain's state directory and manifest names
```

### Task 9: die Ratschläge

**Files:** `internal/brain/search/stamp.go` (`ReconcileAdvice`), `internal/brain/status/status.go`, `internal/brain/graph/read.go`, die Fälle unter `testdata/cases/` von 1b-1 und 1b-2.

- [ ] **Step 1: die Erwartung ändern, rot laufen lassen**

Die Fälle, die `brain reconcile`/`brain reindex` im Wortlaut halten, bekommen `loomux reconcile` und `loomux reindex`. `go test ./internal/brain/... ./internal/cli/...` läuft rot.

- [ ] **Step 2: die Meldungen ändern** (`ReconcileAdvice`, `status.go`, zwölf Meldungen in `graph/read.go`); der Kommentar an `ReconcileAdvice` entfällt.

- [ ] **Step 3: Aufzeichnungen unter `testdata/cases/*-source` nicht anfassen** (Belege des alten Verhaltens); nur die Erwartungsseite der Fälle ändert sich, und die Abweichungsliste der Stufe (`parity/stufe-3a.md`) nennt die Änderung.

- [ ] **Step 4: Commit**

```sh
git commit -F <scratchpad>/msg.txt   # fix(brain): name loomux's own commands in the advice it gives
```

### Task 10: Messen, Mutation, Doku, PR

- [ ] **Step 1: Messen** — `loomux brain search` und `privacy.VisibleAreas` warm, vor und nach Task 6, mit Ziel und mit Aside; in `docs/en/benchmarks.md` und `docs/de/benchmarks.md` eintragen (Datum und Uhrzeit, was gemessen, Basis gegen Änderung, kalt und warm).
- [ ] **Step 2: Mutationsrunde** über `internal/config/artifacts.go` (Aside-Zweig) und die geänderten Leser; Überlebende dokumentieren (Abschnitt „Überlebende Mutanten“ in `parity/stufe-4e.md`; `internal/plancheck` verlangt ihn für ✅).
- [ ] **Step 3: Doku** — `docs/en/migration.md` und `docs/de/migration.md` (4e ✅ erst, wenn Checkliste, Rauchtests und beide PRs stehen; 4f-Abhängigkeit neu lesen), Roadmap, beide READMEs, `docs/*/cli-reference.md`; die Fusions-Spec-Zeile der Stufe.
- [ ] **Step 4: PR** — mit `release-pr`; Label `release:patch` (Rückfälle entfallen ohne neue Fähigkeit; ist ein Befehl oder Ausgabewortlaut betroffen, `release:major` prüfen: die Ratschläge in Meldungen sind kein Protokoll, aber der Wegfall von `LOOMUX_LEGACY_BRAIN_DIR` ist ein Umgebungsschalter). Der Nutzer entscheidet das Label nach Sicht des Diffs. Der Mensch pusht.

---

## Self-Review

**Spec-Abdeckung:** A (Tasks 1–5), B (Task 4 und „Stück B“), C mit 1′ (Tasks 6–7), Rückfälle (Task 8), Ratschläge (Task 9), Messen und Bedingungen 3–5 (Task 10). Die offenen Punkte der Spec sind Task 1 (`init --dry-run`, `Lanes`, `moveStock`, Entscheidungen, vier Manifeste). Der Nachtrag #25 steht in der Fusions-Spec.

**Bekannte Lücken, die der Mensch kennen muss:** Task 7 und die Reihenfolge in Task 4 hängen von Task 1 ab; ihre Fundstellen stehen erst danach fest. Wo Task 1 keine Fundstelle ergibt, entfällt der Task.
