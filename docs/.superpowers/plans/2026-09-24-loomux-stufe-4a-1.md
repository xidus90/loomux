# Stufe 4a-1: Schema und `loomux config` — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** loomux zeigt und ändert jede Einstellung von `.loomux/config.toml`
über ein Schema und einen kommentarerhaltenden Zeileneditor, als
`loomux config` (interaktiv auf `x/term` und als `list|get|set`), und
`[modules]` schaltet Hooks, Brain und Graph zur Laufzeit, ohne den Wächter.

**Architecture:** Ein schmaler Leser `config.ReadModules` liegt auf dem
Hook-Pfad; alles Schwere — `internal/config/schema` (Schlüssel, Vorgaben,
Herkunft, Prüfung über die heutigen Lader), `internal/config/edit`
(Textchirurgie) und `internal/tui` (Oberfläche) — erreicht nur die
Befehlszeile, ein Tor-Test hält das fest. Geprüft wird ein neuer Wert, indem
der geänderte Text in ein Wegwerf-Projekt geschrieben und von denselben
Ladern gelesen wird, die im Betrieb gelten.

**Tech Stack:** Go 1.x des Moduls, `github.com/BurntSushi/toml`
(`third_party/toml`), `golang.org/x/term` (neu), `golang.org/x/sys` (schon
direkt), `internal/lock.ReplaceText`, `internal/shellwords`.

**Spec:** `docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md`,
Abschnitte „Entscheidungen“, „4a-1 im Einzelnen“, „Selbstnutzung“, „Messen“.

## Global Constraints

- Code, Bezeichner, Kommentare, Fehlermeldungen und Commits englisch; dieser
  Plan und die Akte deutsch.
- Coverage 100 % je Funktion; eine Ausnahme nur mit `//coverage:exempt <Grund>`
  direkt über `func`.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst; alles beim
  ersten Gebrauch.
- `internal/hooks` importiert weder `internal/config/schema` noch
  `internal/config/edit` noch `internal/tui` (Tor-Test, Task 12).
- Fehlt `[modules]` oder ein Schlüssel darin, ist das Modul **an**.
- `[modules].hooks = false` schaltet post-edit, `stop`, `session-start` und
  `subagent-*` ab, **nie** `pre-tool-use`.
- Vorgaben werden nie in die Datei geschrieben.
- Geschrieben wird nur über `lock.ReplaceText` und erst, wenn alle Lader den
  neuen Text annehmen.
- Exit-Codes von `config`: 0 Erfolg oder abgelehnte Bestätigung, 1 Lader oder
  Editor lehnen ab, 2 falscher Aufruf.
- Commit-Nachrichten nach Conventional Commits, ohne Plan-, Stufen- oder
  Aufgabennamen; Autor ist der Nutzer, kein Modell als Mitautor.
- Vor jedem Commit Zweig und HEAD lesen; mehrzeilige Nachrichten über eine
  Datei und `git commit -F`.
- Neueste Version einer neuen Abhängigkeit wählen.
- **Kein `regexp.MustCompile` und keine andere allokierende Initialisierung
  auf Paketebene** in den neuen Paketen: `cmd/loomux` linkt `cli`, `cli`
  linkt `schema`, `edit` und `tui`, und deren Paket-Inits laufen darum bei
  **jedem** `loomux hook pre-tool-use` mit. `cmd/loomux/start_test.go` lässt
  jedes Paket-Init über 500 Allokationen scheitern. Ausdrücke kommen in ein
  `sync.OnceValue`, wie `builtinCommands` in `internal/hooks/guard.go`.

## Ausführungsreihenfolge und Menschenschritte

Drei Schritte führt **der Mensch** aus; ein Subagent versucht sie nie, der
Orchestrator hält dort an und fragt:

- **Task 1** (ändert kurz die Nutzerkonfiguration von Claude Code),
- **Task 13, Step 11** (drei Terminals von Hand prüfen),
- **Task 16, Step 4**, der Teil, in dem ein Mensch `config set` ruft.

Task 5 hängt am Ergebnis von Task 1. Darum läuft die Reihenfolge
**1 (Mensch, parallel) · 2 · 3 · 4 · 6 · 7 · 8 · 9 · 10 · 11 · 12 · 13 · 14 ·
15 · 5 · 16**: Der Mensch misst, während die Subagenten 2–4 und 6–15 bauen.

## Dateien

| Datei | Verantwortung |
|---|---|
| `internal/config/modules.go` (neu) | `Modules`, `ReadModules`, `ParseModules` — der schmale Leser |
| `internal/config/declaration.go` | `DeclarationKeys()` legt die gelesenen Schlüssel offen |
| `internal/config/policy.go` | `PolicyKeys()` |
| `internal/verify/schema.go` | `TopKeys()`; `ReadConfig` schaltet die Wiki-Lane bei `brain = false` ab |
| `internal/verify/commit/policy.go` | `KnownKeys()` |
| `internal/worktree/mirror/mirrorcfg.go` | unverändert, wird von `schema.Validate` gerufen |
| `internal/cli/hook.go` | `[modules].hooks` vor jedem Ereignis außer `pre-tool-use` |
| `internal/mcptools/tools.go` | `For(m config.Modules)` filtert die Werkzeugliste |
| `internal/bridge/bridge.go` | `Options.Tools` (nil = alle); registriert nur die übergebenen Werkzeuge |
| `internal/cli/mcp.go` | `--root`, Suche aufwärts, Hinweis auf stderr |
| `internal/config/schema/schema.go` (neu) | `Key`, `Keys()`, `Lookup`, `Kind` |
| `internal/config/schema/validate.go` (neu) | `Validate(text)` über die Lader |
| `internal/config/schema/current.go` (neu) | `Current(root)` mit Wert und Herkunft |
| `internal/config/edit/edit.go` (neu) | `Set`, `AppendBlock`, `ErrAmbiguous` |
| `internal/config/edit/value.go` (neu) | `Render` eines Werts als TOML-Literal |
| `internal/config/edit/diff.go` (neu) | `Diff` zweier Texte zeilenweise |
| `internal/tui/terminal.go` (neu) | `Terminal`-Schnittstelle, `Key`, Tastendekodierung |
| `internal/tui/raw.go`, `raw_windows.go`, `raw_other.go` (neu) | Hülle um `x/term` und VT-Modus, `//coverage:exempt` |
| `internal/tui/list.go`, `input.go`, `confirm.go` (neu) | die Bausteine |
| `internal/cli/config.go` (neu) | `loomux config` in allen Formen |
| `internal/cli/commands.go` | Eintrag `"config"` |
| `internal/hooks/guard.go` | dritte eingebaute Befehlsregel |
| `internal/cli/imports_test.go` | Tor-Test gegen `schema`, `edit`, `tui` im Hook-Pfad |
| `docs/en/cli-reference.md`, `docs/de/cli-reference.md`, `README.md`, `README.de.md` | Befehl dokumentiert |
| `docs/en/benchmarks.md`, `docs/de/benchmarks.md` | Messungen |
| `docs/.superpowers/parity/stufe-4a-1.md` (neu) | Akte |
| `docs/en/migration.md`, `docs/de/migration.md` | 4a-1 ✅ am Ende |

---

### Task 1 (Mensch): Messen, wo ein MCP-Server startet

Die Spec macht den Modulfilter der Brücke davon abhängig, dass Claude Code
einen stdio-Server aus dem Nutzerbereich im Projektverzeichnis startet. Das
wird vor Task 5 gemessen. **Diese Aufgabe führt der Mensch aus**, weil sie
die Nutzerkonfiguration von Claude Code kurz ändert. Gemessen wird zweimal:
mit der CLI (`claude -p`) und in der **Claude-Desktop-App** (Code-Tab), in der
der Nutzer arbeitet — die CLI allein beweist nicht, dass die App stdio-Server
aus demselben Verzeichnis startet.

**Files:**
- Create: `docs/.superpowers/parity/stufe-4a-1.md`

- [ ] **Step 1: Sonde bauen**

`%TEMP%\mcp-cwd-probe.cmd` mit diesem Inhalt anlegen:

```bat
@echo off
cd > "%TEMP%\mcp-cwd.txt"
```

- [ ] **Step 2: Mensch registriert die Sonde im Nutzerbereich**

```bash
claude mcp add --scope user cwd-probe -- cmd /c "%TEMP%\mcp-cwd-probe.cmd"
```

- [ ] **Step 3: Mensch startet Claude Code in einem Unterverzeichnis eines Projekts**

```bash
cd C:/Users/micro/Documents/#GIT/loomux/internal && claude -p "list your mcp servers"
```

Danach `%TEMP%\mcp-cwd.txt` lesen. Erwartet, wenn die Annahme trägt: der
Pfad, in dem `claude` gestartet wurde (`…\loomux\internal`). Die Brücke sucht
von dort aufwärts und findet `…\loomux\.loomux\config.toml`.

Dann `%TEMP%\mcp-cwd.txt` löschen, in der Desktop-App eine neue Sitzung auf
einem Worktree von loomux öffnen und nach dem Start der Sitzung die Datei
erneut lesen. Erwartet: der Pfad des Worktrees.

- [ ] **Step 4: Mensch entfernt die Sonde**

```bash
claude mcp remove --scope user cwd-probe
```

- [ ] **Step 5: Ergebnis in die Akte**

`docs/.superpowers/parity/stufe-4a-1.md` anlegen:

```markdown
# Paritätsakte Stufe 4a-1

4a-1 hat keine Referenz: `loomux config` und `[modules]` sind neu. Die Akte
hält Messungen, Entscheidungen beim Bau und die Mutationsrunde fest.

## Messungen vor dem Bau

### Arbeitsverzeichnis eines stdio-MCP-Servers (2026-09-24)

Sonde im Nutzerbereich, `claude -p` gestartet in `<pfad>`; die Sonde schrieb
`<ergebnis>`. Desktop-App, Sitzung auf `<worktree>`; die Sonde schrieb
`<ergebnis>`. Folge für Task 5: <Filter wie geplant | Rückfall der Spec: kein
Filter, Hinweis auf stderr>.
```

Die Platzhalter in spitzen Klammern ersetzt der Ausführende durch das
Gemessene; der Satz „Folge für Task 5“ entscheidet, welcher Zweig von Task 5
gebaut wird.

- [ ] **Step 6: Commit**

```bash
git add docs/.superpowers/parity/stufe-4a-1.md
git commit -m "docs: record where a user-scope MCP server starts"
```

---

### Task 2: `[modules]` lesen

**Files:**
- Create: `internal/config/modules.go`
- Test: `internal/config/modules_test.go`

**Interfaces:**
- Produces: `type Modules struct { Hooks, Brain, Graph bool }`;
  `func AllModules() Modules`; `func ReadModules(root string) (Modules, error)`;
  `func ParseModules(path string, doc map[string]any) (Modules, error)`;
  `func ModuleKeys() []string` (`["brain", "graph", "hooks"]`).

- [ ] **Step 1: Write the failing test**

```go
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifest(t *testing.T, text string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ManifestPath(root), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestReadModulesWithoutAFileTurnsEverythingOn(t *testing.T) {
	got, err := ReadModules(t.TempDir())
	if err != nil || got != AllModules() {
		t.Fatalf("got %+v, %v; want all on", got, err)
	}
}

func TestReadModulesWithoutTheTableTurnsEverythingOn(t *testing.T) {
	got, err := ReadModules(writeManifest(t, "[area]\nscope = \"x\"\n"))
	if err != nil || got != AllModules() {
		t.Fatalf("got %+v, %v; want all on", got, err)
	}
}

func TestReadModulesKeepsAMissingKeyOn(t *testing.T) {
	got, err := ReadModules(writeManifest(t, "[modules]\ngraph = false\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := (Modules{Hooks: true, Brain: true, Graph: false}); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestReadModulesRefusesWhatItCannotUse(t *testing.T) {
	for name, text := range map[string]string{
		"unknown key":  "[modules]\nos = true\n",
		"not a bool":   "[modules]\nbrain = \"no\"\n",
		"not a table":  "modules = 1\n",
		"broken toml":  "[modules\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ReadModules(writeManifest(t, text))
			if err == nil || !strings.Contains(err.Error(), "config.toml") {
				t.Fatalf("got %v, want an error naming the file", err)
			}
		})
	}
}

func TestReadModulesReportsAnUnreadableFile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(ManifestPath(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadModules(root); err == nil {
		t.Fatal("a directory where the file should be must be an error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run Modules -count=1`
Expected: FAIL, `undefined: ReadModules`.

- [ ] **Step 3: Write minimal implementation**

```go
package config

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"

	"github.com/BurntSushi/toml"
)

// Modules is the [modules] table: which parts of loomux run in a project.
// The guard is not among them. The write barrier is global -- it protects
// the read-only areas of every repository -- so no project may switch it off.
type Modules struct {
	Hooks bool
	Brain bool
	Graph bool
}

// AllModules is what a project that says nothing gets: everything on, so a
// repository without [modules] behaves as it did before the table existed.
func AllModules() Modules { return Modules{Hooks: true, Brain: true, Graph: true} }

// ModuleKeys are the keys [modules] knows, sorted.
func ModuleKeys() []string { return []string{"brain", "graph", "hooks"} }

// ReadModules reads [modules] of the project at root. It is the one reader
// on the per-edit path, so it decodes nothing but the document and judges
// nothing but this table.
func ReadModules(root string) (Modules, error) {
	path := ManifestPath(root)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return AllModules(), nil
	}
	if err != nil {
		return Modules{}, fmt.Errorf("%s: %w", path, err)
	}
	doc := map[string]any{}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return Modules{}, fmt.Errorf("%s: not valid TOML: %w", path, err)
	}
	return ParseModules(path, doc)
}

// ParseModules reads [modules] from a decoded document. An unknown key is
// refused: `graf = false` that silently left the graph on would look
// configured and not be.
func ParseModules(path string, doc map[string]any) (Modules, error) {
	modules := AllModules()
	raw, present := doc["modules"]
	if !present {
		return modules, nil
	}
	table, ok := raw.(map[string]any)
	if !ok {
		return Modules{}, fmt.Errorf("%s: [modules] must be a table, found %s", path, tomlType(raw))
	}
	keys := make([]string, 0, len(table))
	for key := range table {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !slices.Contains(ModuleKeys(), key) {
			return Modules{}, fmt.Errorf("%s: [modules] does not know %q; known: %v", path, key, ModuleKeys())
		}
		on, ok := table[key].(bool)
		if !ok {
			return Modules{}, fmt.Errorf("%s: [modules] %s must be true or false, found %s", path, key, tomlType(table[key]))
		}
		switch key {
		case "hooks":
			modules.Hooks = on
		case "brain":
			modules.Brain = on
		default:
			modules.Graph = on
		}
	}
	return modules, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config -run Modules -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/modules.go internal/config/modules_test.go
git commit -m "feat(config): read which modules a project runs"
```

---

### Task 3: Hooks richten sich nach `[modules].hooks`

**Files:**
- Modify: `internal/cli/hook.go` (nach der Auflösung von `resolved`, vor dem `switch event`)
- Test: `internal/cli/hook_test.go`

**Interfaces:**
- Consumes: `config.ReadModules(root string) (config.Modules, error)` aus Task 2.

- [ ] **Step 1: Write the failing test**

```go
func TestHookEndsAtOnceWhenTheHooksModuleIsOff(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[modules]\nhooks = false\n")
	called := false
	restore := postToolUse
	postToolUse = func(io.Reader, io.Writer, io.Writer, string, time.Duration) int { called = true; return 2 }
	t.Cleanup(func() { postToolUse = restore })
	var out, errOut bytes.Buffer
	code := Run([]string{"hook", "post-tool-use", "--host", "claude", "--root", root}, strings.NewReader("{}"), &out, &errOut)
	if code != 0 || called {
		t.Fatalf("code %d, called %v; want 0 and no lanes", code, called)
	}
}

func TestTheGuardRunsEvenWhenTheHooksModuleIsOff(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[modules]\nhooks = false\n")
	t.Setenv("LOOMUX_STATE_DIR", t.TempDir())
	payload := `{"tool_name":"Write","tool_input":{"file_path":".env","content":"x"}}`
	var out, errOut bytes.Buffer
	code := Run([]string{"hook", "pre-tool-use", "--host", "claude", "--root", root}, strings.NewReader(payload), &out, &errOut)
	if code != hooks.ExitDenied {
		t.Fatalf("code %d; the guard must still refuse a write to .env", code)
	}
}

func TestHookRefusesABrokenModulesTable(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[modules]\nhooks = 1\n")
	var out, errOut bytes.Buffer
	code := Run([]string{"hook", "stop", "--host", "claude", "--root", root}, strings.NewReader("{}"), &out, &errOut)
	if code != hooks.ExitInternal || !strings.Contains(errOut.String(), "[modules]") {
		t.Fatalf("code %d, stderr %q", code, errOut.String())
	}
}
```

`writeFile` gibt es in den Tests von `internal/cli` noch nicht, wenn der
Compiler das meldet; dann in `hook_test.go` anlegen:

```go
func writeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run 'HooksModule|BrokenModules' -count=1`
Expected: FAIL (`called` ist true bzw. Code 2).

- [ ] **Step 3: Write minimal implementation**

In `hookCommand`, direkt vor `switch event {`:

```go
	// The guard is not a module: the write barrier is global and protects
	// other repositories' read-only areas, so no project switches it off.
	if event != "pre-tool-use" {
		modules, err := config.ReadModules(resolved)
		if err != nil {
			fmt.Fprintf(stderr, "loomux hook %s: %v\n", event, err)
			return failure
		}
		if !modules.Hooks {
			return 0
		}
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run 'Hook' -count=1`
Expected: PASS, auch die bestehenden Hook-Tests.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/hook.go internal/cli/hook_test.go
git commit -m "feat(hooks): let [modules] switch the session hooks off, never the guard"
```

---

### Task 4: `brain = false` schaltet die Wiki-Lane ab

**Files:**
- Modify: `internal/verify/schema.go:93-122` (`ReadConfig`, `ParseConfig`)
- Test: `internal/verify/schema_test.go`

**Interfaces:**
- Consumes: `config.ParseModules(path, doc)` aus Task 2.
- Produces: unverändert `ReadConfig(root) (Config, error)`; bei
  `brain = false` ist `cfg.Stacks["wiki"]["lint"].Lane.Off == true`.

- [ ] **Step 1: Read how an off lane is represented**

Run: `grep -n "Off" internal/verify/schema.go internal/verify/effective.go`
Die Lane `wiki/lint` ist aus, wenn `Stacks["wiki"]["lint"]` ein `Override`
mit `Lane.Off = true` ist; so setzt `parseStack` sie für `[verify.wiki]
lint = false` (`schema.go:224`). Die Implementierung in Step 3 baut genau
diesen Wert.

- [ ] **Step 2: Write the failing test**

```go
func TestTheBrainModuleOffTurnsTheWikiLaneOff(t *testing.T) {
	doc := map[string]any{"modules": map[string]any{"brain": false}}
	cfg, err := ParseConfig("config.toml", doc)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Stacks["wiki"]["lint"].Lane.Off {
		t.Fatal("brain = false must turn lint/wiki off")
	}
}

func TestABrokenModulesTableFailsTheVerifyConfig(t *testing.T) {
	doc := map[string]any{"modules": map[string]any{"brain": "no"}}
	if _, err := ParseConfig("config.toml", doc); err == nil {
		t.Fatal("want the modules error")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/verify -run 'BrainModule|BrokenModules' -count=1`
Expected: FAIL.

- [ ] **Step 4: Write minimal implementation**

`ParseConfig` wird zu:

```go
func ParseConfig(path string, doc map[string]any) (Config, error) {
	cfg := defaults()
	if raw, ok := doc["verify"]; ok {
		if err := parseVerify(&cfg, raw); err != nil {
			return Config{}, fmt.Errorf("%s: %w", path, err)
		}
	}
	modules, err := config.ParseModules(path, doc)
	if err != nil {
		return Config{}, err
	}
	if !modules.Brain {
		// The wiki lane is the brain module's lane; with the module off it
		// is off the way `[verify.wiki] lint = false` turns it off.
		if cfg.Stacks["wiki"] == nil {
			cfg.Stacks["wiki"] = map[string]Override{}
		}
		lint := cfg.Stacks["wiki"]["lint"]
		lint.Lane.Off = true
		cfg.Stacks["wiki"]["lint"] = lint
	}
	return cfg, nil
}
```

Stimmt der Feldname nach Step 1 nicht (`Lane.Off`), den tatsächlichen aus
`parseStack` übernehmen — der Test prüft das Feld, das `post_edit.go:170`
liest.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/verify/... ./internal/hooks/... -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/verify/schema.go internal/verify/schema_test.go
git commit -m "feat(verify): drop the wiki lane when the brain module is off"
```

---

### Task 5: Die Brücke filtert nach `[modules]`

Welcher Zweig gebaut wird, sagt die Akte aus Task 1. **Zweig A** (die Sonde
zeigte das Startverzeichnis): wie unten. **Zweig B** (sie zeigte etwas
anderes): Step 3 bis 5 entfallen bis auf `--root`; `mcpCommand` schreibt
ohne `--root` einmal `loomux mcp: [modules] is not applied without --root:
the host does not start this server in the project` auf stderr und
registriert alle Werkzeuge.

**Files:**
- Modify: `internal/mcptools/tools.go` (neue Funktion `For`)
- Modify: `internal/bridge/bridge.go:24-70` (`Options.Tools`, nil heißt alle)
- Modify: `internal/cli/mcp.go:71-96` (`--root`)
- Test: `internal/mcptools/tools_test.go`, `internal/cli/mcp_test.go`

**Interfaces:**
- Consumes: `config.ReadModules`, `config.AllModules` (Task 2),
  `hosts.FindRoot(start string) (string, error)`.
- Produces: `func mcptools.For(m config.Modules) []*mcp.Tool`;
  `bridge.Options.Tools []*mcp.Tool` — `nil` heißt `mcptools.Tools()`, damit
  jede bestehende Konstruktionsstelle (`cases_1b2_test.go:269`, die Tests in
  `internal/bridge`) unverändert alle Werkzeuge anbietet;
  `func mcpArgs(args []string) (privacy.Channel, string, *brainUsageError)` —
  der zweite Wert ist `--root`.

- [ ] **Step 1: Write the failing test for the filter**

```go
func TestForDropsTheToolsOfAModuleThatIsOff(t *testing.T) {
	all := len(Tools())
	noBrain := For(config.Modules{Hooks: true, Brain: false, Graph: true})
	if len(noBrain) != all-len(Brain()) {
		t.Fatalf("got %d tools, want %d", len(noBrain), all-len(Brain()))
	}
	for _, tool := range noBrain {
		if strings.HasPrefix(tool.Name, "brain_") {
			t.Fatalf("%s must be gone", tool.Name)
		}
	}
	noGraph := For(config.Modules{Hooks: true, Brain: true, Graph: false})
	for _, tool := range noGraph {
		if strings.HasPrefix(tool.Name, "graph_") {
			t.Fatalf("%s must be gone", tool.Name)
		}
	}
	if len(For(config.AllModules())) != all {
		t.Fatal("all modules on must keep every tool")
	}
}
```

- [ ] **Step 2: Run it to see it fail**

Run: `go test ./internal/mcptools -run For -count=1`
Expected: FAIL, `undefined: For`.

- [ ] **Step 3: Implement `For`**

```go
// For is the tool list a project with these modules offers: a tool of a
// module that is off is not listed, so the host never sees it and a call to
// it is an unknown tool.
func For(m config.Modules) []*mcp.Tool {
	// Empty, never nil: bridge.Options reads nil as "every tool".
	tools := make([]*mcp.Tool, 0, len(Tools()))
	if m.Brain {
		tools = append(tools, Brain()...)
	}
	if m.Graph {
		tools = append(tools, Graph()...)
	}
	return tools
}
```

Prüfen, dass `Tools()` genau `Brain()` gefolgt von `Graph()` ist
(`tools.go:60`); ist die Reihenfolge anders, `For` so ordnen, dass
`For(AllModules())` dieselbe Reihenfolge wie `Tools()` hat, und das im Test
mit `reflect.DeepEqual(For(config.AllModules()), Tools())` festhalten.

- [ ] **Step 4: Bridge uses it**

In `bridge.Options`:

```go
	// Tools are what this bridge offers; nil offers every tool. The command
	// line passes the list filtered by the project's [modules]. Nil rather
	// than an empty list as the default, so a caller that forgets the field
	// gets the old behaviour instead of a server with no tools.
	Tools []*mcp.Tool
```

In `Run` die Schleife ersetzen:

```go
	tools := opts.Tools
	if tools == nil {
		tools = mcptools.Tools()
	}
	for _, tool := range tools {
```

Ein Test in `internal/bridge` hält fest, dass `Options{}` alle Werkzeuge
anbietet und `Options{Tools: mcptools.Brain()}` nur die fünf `brain_*`.
Achtung: `mcptools.For` mit allen Modulen aus muss eine **leere, nicht nil**
Liste liefern (`make([]*mcp.Tool, 0)`), sonst bietet eine Brücke mit
`brain = false, graph = false` wieder alles an; ein Test prüft das.

- [ ] **Step 5: Write the failing CLI test**

```go
func offers(o bridge.Options, prefix string) bool {
	for _, tool := range o.Tools {
		if strings.HasPrefix(tool.Name, prefix) {
			return true
		}
	}
	return false
}

func TestMcpReadsTheModulesOfTheProjectAbove(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[modules]\ngraph = false\n")
	sub := filepath.Join(root, "internal")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	var got bridge.Options
	restore := bridgeRun
	bridgeRun = func(_ context.Context, o bridge.Options) error { got = o; return nil }
	t.Cleanup(func() { bridgeRun = restore })
	var out, errOut bytes.Buffer
	if code := Run([]string{"mcp"}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("code %d: %s", code, errOut.String())
	}
	if offers(got, "graph_") || !offers(got, "brain_") {
		t.Fatalf("tools %v; want graph off, brain on", got.Tools)
	}
}

func TestMcpWithoutAProjectOffersEverything(t *testing.T) {
	t.Chdir(t.TempDir())
	var got bridge.Options
	restore := bridgeRun
	bridgeRun = func(_ context.Context, o bridge.Options) error { got = o; return nil }
	t.Cleanup(func() { bridgeRun = restore })
	var out, errOut bytes.Buffer
	Run([]string{"mcp"}, strings.NewReader(""), &out, &errOut)
	if len(got.Tools) != len(mcptools.Tools()) {
		t.Fatalf("%d tools; want all", len(got.Tools))
	}
}

func TestMcpTakesAnExplicitRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[modules]\nbrain = false\n")
	var got bridge.Options
	restore := bridgeRun
	bridgeRun = func(_ context.Context, o bridge.Options) error { got = o; return nil }
	t.Cleanup(func() { bridgeRun = restore })
	var out, errOut bytes.Buffer
	Run([]string{"mcp", "--root", root}, strings.NewReader(""), &out, &errOut)
	if offers(got, "brain_") {
		t.Fatal("brain must be off")
	}
}

func TestMcpRefusesABrokenModulesTable(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[modules]\nbrain = 3\n")
	var out, errOut bytes.Buffer
	if code := Run([]string{"mcp", "--root", root}, strings.NewReader(""), &out, &errOut); code != 1 {
		t.Fatalf("code %d; a broken table refuses the bridge", code)
	}
	if out.Len() != 0 {
		t.Fatal("nothing may reach stdout, the host's pipe")
	}
}
```

Wenn `t.Chdir` in `internal/cli` auf einen `getwd`-Stub trifft
(`stubGetwd` in `area_test.go`), den Stub in diesen Tests ebenso setzen.

- [ ] **Step 6: Run to see them fail**

Run: `go test ./internal/cli -run Mcp -count=1`
Expected: FAIL.

- [ ] **Step 7: Implement `--root` and the lookup**

`mcpArgs` gibt zusätzlich `root string` zurück; im Schleifenkopf:

```go
		flag, value, glued := strings.Cut(args[i], "=")
		if flag != "--channel" && flag != "--root" {
			extra = append(extra, args[i])
			continue
		}
		if !glued {
			if i+1 == len(args) {
				return "", "", &brainUsageError{message: "argument " + flag + ": expected one argument"}
			}
			i++
			value = args[i]
		}
		if flag == "--root" {
			root = value
			continue
		}
```

In `mcpCommand` nach dem Parsen:

```go
	modules, err := projectModules(root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux mcp: %v\n", err)
		return 1
	}
```

und `bridge.Options{…, Tools: mcptools.For(modules)}`. Die Werkzeugliste
cacht der Host (`setCacheable`): Eine Änderung an `[modules]` wirkt erst,
wenn der Host die Brücke neu startet — das steht in der Befehlsreferenz
(Task 16). Neu in `mcp.go`:

```go
// projectModules is the [modules] of the project this bridge serves: the one
// named by --root, else the first .loomux/config.toml above the working
// directory the host started it in. No project is not an error -- a host may
// start the bridge anywhere -- and offers every tool.
func projectModules(root string) (config.Modules, error) {
	if root == "" {
		found, err := hosts.FindRoot(".")
		if err != nil {
			return config.AllModules(), nil
		}
		root = found
	}
	return config.ReadModules(root)
}
```

`mcpUsage()` wird `usage: loomux mcp [--channel {local,cloud}] [--root DIR]`.
Prüfen, ob ein aufgezeichneter Fall aus 1b-2 den Wortlaut von `mcpUsage`
vergleicht (`grep -rn "usage: loomux mcp" testdata internal`); wenn ja, den
Fall als freigegebene Abweichung in die Akte und die erwartete Ausgabe
nachziehen.

- [ ] **Step 8: Run all touched tests**

Run: `go test ./internal/mcptools ./internal/bridge ./internal/cli -count=1`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/mcptools internal/bridge internal/cli/mcp.go internal/cli/mcp_test.go
git commit -m "feat(mcp): offer only the tools of the project's modules"
```

---

### Task 6: Die Lader legen ihre Schlüssel offen

Das Schema aus Task 7 muss jeden Schlüssel kennen, den ein Lader liest. Die
Lader bekommen dafür je eine Funktion, die ihre Liste zurückgibt, und lesen
selbst aus derselben Liste — so kann sie nicht veralten.

**Files:**
- Modify: `internal/config/declaration.go` (die Literale in `declaration` aus `DeclarationKeys()` lesen)
- Modify: `internal/config/policy.go`, `internal/verify/schema.go`, `internal/verify/commit/policy.go`
- Test: je `_test.go` daneben

**Interfaces:**
- Produces:
  - `func config.DeclarationKeys() map[string][]string` —
    `area: [scope]`, `privacy: [mode, never]`, `wiki: [types, untouched_days]`,
    `maintenance: [on_merge, branch]`, `model: [enabled, roles]`,
    `layout: [wiki, hub, review, inbox]`, `index: [include, exclude, unsearched]`.
  - `func config.PolicyKeys() map[string][]string` —
    `policy.paths.rules: [match, reason]`, `policy.commands.rules: [regex, reason]`.
  - `func verify.TopKeys() []string` — `[max_parallel, timeout, profiles]`.
  - `func commit.KnownKeys() []string` — `[allow, conventional, language, threshold]`.

- [ ] **Step 1: Write the failing tests**

`internal/config/declaration_test.go`:

```go
func TestDeclarationKeysNameEverySectionTheReaderChecks(t *testing.T) {
	keys := DeclarationKeys()
	for _, section := range declarationSections() {
		if _, ok := keys[section]; !ok {
			t.Errorf("DeclarationKeys lacks [%s]", section)
		}
	}
	if !slices.Equal(keys["layout"], []string{"wiki", "hub", "review", "inbox"}) {
		t.Errorf("layout keys %v", keys["layout"])
	}
}
```

`internal/verify/commit/policy_test.go`:

```go
func TestKnownKeysAreTheOnesTheReaderAccepts(t *testing.T) {
	if !slices.Equal(KnownKeys(), []string{"allow", "conventional", "language", "threshold"}) {
		t.Fatal(KnownKeys())
	}
}
```

`internal/verify/schema_test.go`:

```go
func TestTopKeysAreTheScalarsOfVerify(t *testing.T) {
	if !slices.Equal(TopKeys(), []string{"max_parallel", "timeout", "profiles"}) {
		t.Fatal(TopKeys())
	}
}
```

`internal/config/policy_test.go`:

```go
func TestPolicyKeysNameBothRuleLists(t *testing.T) {
	keys := PolicyKeys()
	if !slices.Equal(keys["policy.paths.rules"], []string{"match", "reason"}) ||
		!slices.Equal(keys["policy.commands.rules"], []string{"regex", "reason"}) {
		t.Fatal(keys)
	}
}
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/config ./internal/verify/... -run 'Keys' -count=1`
Expected: FAIL, undefined.

- [ ] **Step 3: Implement**

`declaration.go`:

```go
// DeclarationKeys are the keys ReadDeclaration reads, per section. The
// reader takes its lists from here, so the schema of `loomux config` and the
// reader cannot drift apart.
func DeclarationKeys() map[string][]string {
	return map[string][]string{
		"area":        {"scope"},
		"privacy":     {"mode", "never"},
		"wiki":        {"types", "untouched_days"},
		"maintenance": {"on_merge", "branch"},
		"model":       {"enabled", "roles"},
		"layout":      {"wiki", "hub", "review", "inbox"},
		"index":       {"include", "exclude", "unsearched"},
	}
}
```

In `declaration` die beiden Schleifen auf
`DeclarationKeys()["layout"]` und `DeclarationKeys()["index"]` umstellen.

`policy.go`:

```go
// PolicyKeys are the keys of the two rule lists ReadPolicy reads.
func PolicyKeys() map[string][]string {
	return map[string][]string{
		"policy.paths.rules":    {"match", "reason"},
		"policy.commands.rules": {"regex", "reason"},
	}
}
```

`verify/schema.go` — `parseVerify` prüft heute `max_parallel`, `timeout`,
`profiles` in einem `switch`; die Funktion daneben:

```go
// TopKeys are the scalar keys of [verify]; every other key names a stack.
func TopKeys() []string { return []string{"max_parallel", "timeout", "profiles"} }
```

`verify/commit/policy.go`:

```go
// KnownKeys are the keys [commit] accepts.
func KnownKeys() []string { return slices.Clone(knownCommitKeys) }
```

- [ ] **Step 4: Run all config and verify tests**

Run: `go test ./internal/config/... ./internal/verify/... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config internal/verify
git commit -m "refactor(config): let each reader name the keys it reads"
```

---

### Task 7: Das Schema

**Files:**
- Create: `internal/config/schema/schema.go`
- Test: `internal/config/schema/schema_test.go`

**Interfaces:**
- Consumes: Task 2 und Task 6.
- Produces:

```go
type Kind int
const (
	String Kind = iota
	Int
	Bool
	Enum
	StringList
	Duration   // "600s"
	Table      // shown, not edited
	TableList  // [[…]] blocks: shown, appended, never rewritten
)
type Module string
const (
	Base  Module = "base"
	Hooks Module = "hooks"
	Brain Module = "brain"
	Graph Module = "graph"
)
type Key struct {
	Section string   // "commit", "policy.paths.rules", "verify"
	Name    string   // "language"; "" for a Table/TableList entry itself
	Kind    Kind
	Default string   // TOML literal of the default; "" when there is none
	Choices []string // Enum only
	Module  Module
	Doc     string   // one English sentence
}
func (k Key) ID() string                  // "commit.language"; the section for Table/TableList
func Keys() []Key                         // sorted by Module, Section, Name
func Lookup(id string) (Key, bool)
func GlobalKeys() []Key                   // empty until 4c adds [model]
```

- [ ] **Step 1: Write the failing tests**

```go
package schema

import (
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/verify"
	"github.com/xidus90/loomux/internal/verify/commit"
)

func ids(section string) []string {
	var out []string
	for _, k := range Keys() {
		if k.Section == section && k.Name != "" {
			out = append(out, k.Name)
		}
	}
	slices.Sort(out)
	return out
}

func sorted(s []string) []string { c := slices.Clone(s); slices.Sort(c); return c }

func TestTheSchemaKnowsEveryKeyTheReadersRead(t *testing.T) {
	for section, keys := range config.DeclarationKeys() {
		if got := ids(section); !slices.Equal(got, sorted(keys)) {
			t.Errorf("[%s]: schema %v, reader %v", section, got, sorted(keys))
		}
	}
	if got := ids("modules"); !slices.Equal(got, sorted(config.ModuleKeys())) {
		t.Errorf("[modules]: %v", got)
	}
	if got := ids("commit"); !slices.Equal(got, sorted(slices.DeleteFunc(commit.KnownKeys(), func(k string) bool { return k == "allow" }))) {
		t.Errorf("[commit]: %v", got)
	}
	if got := ids("verify"); !slices.Equal(got, sorted(verify.TopKeys())) {
		t.Errorf("[verify]: %v", got)
	}
	if got := ids("worktree"); !slices.Equal(got, []string{"mirror"}) {
		t.Errorf("[worktree]: %v", got)
	}
	for _, list := range []string{"commit.allow", "policy.paths.rules", "policy.commands.rules"} {
		k, ok := Lookup(list)
		if !ok || k.Kind != TableList {
			t.Errorf("%s must be a TableList", list)
		}
	}
}

func TestEveryKeyHasAModuleAndADoc(t *testing.T) {
	for _, k := range Keys() {
		if k.Module == "" || k.Doc == "" {
			t.Errorf("%s lacks module or doc", k.ID())
		}
		if k.Kind == Enum && len(k.Choices) == 0 {
			t.Errorf("%s is an enum without choices", k.ID())
		}
	}
}

func TestLookupFindsByID(t *testing.T) {
	k, ok := Lookup("commit.language")
	if !ok || k.Kind != Enum || !slices.Equal(k.Choices, []string{"en", "de"}) || k.Default != `"en"` {
		t.Fatalf("%+v %v", k, ok)
	}
	if _, ok := Lookup("commit.nope"); ok {
		t.Fatal("unknown id found")
	}
}

func TestGlobalKeysAreEmptyForNow(t *testing.T) {
	if len(GlobalKeys()) != 0 {
		t.Fatal("the global file has no keys before [model] arrives")
	}
}
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/config/schema -count=1`
Expected: FAIL (Paket fehlt).

- [ ] **Step 3: Implement**

`schema.go` mit `Kind`, `Module`, `Key`, `ID`, `Lookup`, `GlobalKeys` wie
oben und `Keys()` als Literal. Die Vorgaben sind die, die die Lader heute
setzen; wo ein Lader keine hat, bleibt `Default` leer:

```go
func Keys() []Key {
	keys := []Key{
		{Section: "modules", Name: "hooks", Kind: Bool, Default: "true", Module: Base, Doc: "Run the session hooks: post-edit, stop, session-start and subagent. The guard always runs."},
		{Section: "modules", Name: "brain", Kind: Bool, Default: "true", Module: Base, Doc: "Run the brain module: its MCP tools, the wiki lane, convert and fetch."},
		{Section: "modules", Name: "graph", Kind: Bool, Default: "true", Module: Base, Doc: "Offer the code graph's MCP tools."},
		{Section: "commit", Name: "language", Kind: Enum, Choices: []string{"en", "de"}, Default: `"en"`, Module: Base, Doc: "The language commit messages are written in."},
		{Section: "commit", Name: "conventional", Kind: Bool, Default: "true", Module: Base, Doc: "Check the header against Conventional Commits."},
		{Section: "commit", Name: "threshold", Kind: Int, Default: "2", Module: Base, Doc: "How many words of the other language a line may carry."},
		{Section: "commit.allow", Kind: TableList, Module: Base, Doc: "Lines the language check lets through, each with a regex and a reason."},
		{Section: "policy.paths.rules", Kind: TableList, Module: Base, Doc: "Paths no agent may write, each with a match and a reason."},
		{Section: "policy.commands.rules", Kind: TableList, Module: Base, Doc: "Shell commands no agent may run, each with a regex and a reason."},
		{Section: "worktree", Name: "mirror", Kind: StringList, Default: "[]", Module: Base, Doc: "Directories a worktree links to the main checkout instead of owning."},
		{Section: "verify", Name: "max_parallel", Kind: Int, Module: Hooks, Doc: "How many lanes run at once; the number of CPUs when unset."},
		{Section: "verify", Name: "timeout", Kind: Duration, Default: `"600s"`, Module: Hooks, Doc: "How long one command may run."},
		{Section: "verify", Name: "profiles", Kind: Table, Module: Hooks, Doc: "Which kinds each profile runs: edit, precommit, stop."},
		{Section: "area", Name: "scope", Kind: String, Module: Brain, Doc: "The scope this project is registered under, e.g. project/loomux."},
		{Section: "layout", Name: "wiki", Kind: String, Module: Brain, Doc: "Where the wiki bundle lives, relative to the root."},
		{Section: "layout", Name: "hub", Kind: String, Module: Brain, Doc: "Where the hub pages live."},
		{Section: "layout", Name: "review", Kind: String, Module: Brain, Doc: "Where review cases are filed."},
		{Section: "layout", Name: "inbox", Kind: String, Module: Brain, Doc: "Where files wait to be converted."},
		{Section: "wiki", Name: "types", Kind: StringList, Module: Brain, Doc: "Page types this area declares beyond the known ones."},
		{Section: "wiki", Name: "untouched_days", Kind: Int, Default: "180", Module: Brain, Doc: "After how many days a page counts as untouched."},
		{Section: "index", Name: "include", Kind: StringList, Module: Brain, Doc: "Globs of the files the index reads."},
		{Section: "index", Name: "exclude", Kind: StringList, Module: Brain, Doc: "Globs the index skips."},
		{Section: "index", Name: "unsearched", Kind: StringList, Module: Brain, Doc: "Globs that are registered but never given to qmd."},
		{Section: "privacy", Name: "mode", Kind: Enum, Choices: []string{"cloud", "manual_cloud", "local_only"}, Default: `"manual_cloud"`, Module: Brain, Doc: "What may leave the machine."},
		{Section: "privacy", Name: "never", Kind: StringList, Default: "[]", Module: Brain, Doc: "Globs that never leave the machine."},
		{Section: "maintenance", Name: "on_merge", Kind: Bool, Module: Brain, Doc: "Record merges for reconciliation."},
		{Section: "maintenance", Name: "branch", Kind: String, Module: Brain, Doc: "The branch whose merges count."},
		{Section: "model", Name: "enabled", Kind: Bool, Module: Brain, Doc: "Let the local model write proposals for this area."},
		{Section: "model", Name: "roles", Kind: Table, Module: Brain, Doc: "Which roles the local model takes."},
	}
	slices.SortStableFunc(keys, func(a, b Key) int {
		return cmp.Or(cmp.Compare(moduleOrder(a.Module), moduleOrder(b.Module)),
			cmp.Compare(a.Section, b.Section), cmp.Compare(a.Name, b.Name))
	})
	return keys
}
```

Die Werte von `privacy.mode` (`Choices`, `Default`) und `maintenance`,
`model`, `untouched_days` **vor dem Schreiben** gegen `privacyMode`,
`checkRoles`, `untouchedDays` in `declaration.go:135-222` und
`DefaultUntouchedDays` lesen und übernehmen, was dort steht; der Test aus
Task 8 (`Validate` nimmt jede Vorgabe an) fängt einen falschen Wert.

`moduleOrder`: `Base` 0, `Hooks` 1, `Brain` 2, `Graph` 3. `ID()`:
`Section + "." + Name`, bei leerem `Name` nur `Section`.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/config/schema -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/schema
git commit -m "feat(config): describe every key of the project configuration"
```

---

### Task 8: Prüfen über die Lader

**Files:**
- Create: `internal/config/schema/validate.go`
- Test: `internal/config/schema/validate_test.go`

**Interfaces:**
- Produces: `func Validate(text string) error` — nil, wenn jeder Lader den
  Text als `.loomux/config.toml` annimmt; sonst der erste Fehler, mit dem
  Pfadpräfix des Wegwerf-Projekts ersetzt durch `.loomux/config.toml`.

- [ ] **Step 1: Write the failing tests**

```go
func TestValidateAcceptsTheRealConfigurationOfLoomux(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", ".loomux", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(string(data)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAcceptsEveryDefault(t *testing.T) {
	var b strings.Builder
	section := ""
	for _, k := range Keys() {
		if k.Name == "" || k.Default == "" {
			continue
		}
		if k.Section != section {
			fmt.Fprintf(&b, "[%s]\n", k.Section)
			section = k.Section
		}
		fmt.Fprintf(&b, "%s = %s\n", k.Name, k.Default)
	}
	if !strings.Contains(b.String(), "[area]") {
		b.WriteString("[area]\nscope = \"project/x\"\n")
	}
	if err := Validate(b.String()); err != nil {
		t.Fatalf("%v\n%s", err, b.String())
	}
}

func TestValidateRefusesWhatAReaderRefuses(t *testing.T) {
	for name, text := range map[string]string{
		"commit language": "[commit]\nlanguage = \"fr\"\n",
		"modules":         "[modules]\nbrain = 1\n",
		"policy regex":    "[[policy.commands.rules]]\nregex = '(?!x)'\nreason = \"r\"\n",
		"verify":          "[verify]\nnope = 1\n",
		"privacy":         "[area]\nscope = \"s\"\n[privacy]\nmode = \"open\"\n",
		"worktree":        "[worktree]\nmirror = \"x\"\n",
		"toml":            "[commit\n",
	} {
		t.Run(name, func(t *testing.T) {
			err := Validate(text)
			if err == nil {
				t.Fatal("want an error")
			}
			if !strings.Contains(err.Error(), ".loomux/config.toml") || strings.Contains(err.Error(), os.TempDir()) {
				t.Fatalf("error must name the project file, not the scratch copy: %v", err)
			}
		})
	}
}
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/config/schema -run Validate -count=1`
Expected: FAIL.

- [ ] **Step 3: Implement**

```go
package schema

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/verify"
	"github.com/xidus90/loomux/internal/verify/commit"
	"github.com/xidus90/loomux/internal/worktree/mirror"
)

// Validate asks the readers that run in operation whether they accept text
// as a project's .loomux/config.toml. They read files, so text goes into a
// scratch project first; a second checker beside them would be a second
// yardstick, and the two would drift.
func Validate(text string) error {
	root, err := os.MkdirTemp("", "loomux-config-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	path := config.ManifestPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return err
	}
	return named(path, readAll(root, path))
}

func readAll(root, path string) error {
	if _, err := config.ReadDeclaration(path); err != nil && !errors.Is(err, config.ErrNoArea) {
		return err
	}
	if _, err := config.ReadModules(root); err != nil {
		return err
	}
	if _, err := config.ReadPolicy(root); err != nil {
		return err
	}
	if _, err := verify.ReadConfig(root); err != nil {
		return err
	}
	if _, err := commit.ReadPolicy(root); err != nil {
		return err
	}
	_, err := mirror.Mirror(root)
	return err
}

// named puts the project's own name where the scratch path stood: the human
// reading the refusal fixes .loomux/config.toml, not a temporary directory.
func named(scratch string, err error) error {
	if err == nil {
		return nil
	}
	return errors.New(strings.ReplaceAll(err.Error(), scratch, filepath.ToSlash(filepath.Join(".loomux", "config.toml"))))
}
```

Die Fehlerverkettung geht durch `errors.New` verloren; kein Aufrufer fragt
sie ab (`config` gibt nur den Text aus). Prüfen, ob `mirror.Mirror` bei einer
Datei ohne `[worktree]` `nil, nil` gibt (`mirrorcfg.go:28-38`); sonst dessen
„nichts konfiguriert“-Fehler wie `ErrNoArea` behandeln.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/config/schema -count=1`
Expected: PASS. Scheitert `TestValidateAcceptsEveryDefault`, ist eine
Vorgabe in `Keys()` falsch — den Wert aus dem Lader übernehmen, nicht den Test
ändern.

- [ ] **Step 5: Commit**

```bash
git add internal/config/schema
git commit -m "feat(config): validate a configuration through the readers that run"
```

---

### Task 9: Der Zeileneditor

**Files:**
- Create: `internal/config/edit/edit.go`, `internal/config/edit/value.go`, `internal/config/edit/diff.go`
- Test: `internal/config/edit/edit_test.go`, `value_test.go`, `diff_test.go`

**Interfaces:**
- Produces:
  - `var ErrAmbiguous = errors.New(...)`
  - `func Set(text, section, key, literal string) (string, error)` — setzt
    `key = literal` in `[section]`; ergänzt Schlüssel oder Sektion, wenn sie
    fehlen.
  - `func Remove(text, section, key string) (string, error)` — entfernt die
    Zeile (für „zurück auf die Vorgabe“); eine danach leere Sektion samt Kopf
    ebenso.
  - `func AppendBlock(text, section string, pairs [][2]string) string` —
    hängt `[[section]]` mit `name = literal`-Zeilen an.
  - `func Render(kind schema.Kind, input string) (string, error)` — macht
    aus einer Eingabe ein TOML-Literal: `String`/`Enum` → gequotet,
    `Int` → Ziffern geprüft, `Bool` → `true|false`, `Duration` → gequotet
    nach `time.ParseDuration`, `StringList` → Komma-getrennt zu `["a", "b"]`.
  - `func Diff(before, after string) string` — Zeilen mit `-`/`+`, dazu je
    eine Zeile Kontext.

`edit` importiert `schema` nur für `schema.Kind` in `Render`; `schema`
importiert `edit` nicht.

- [ ] **Step 1: Write the failing tests for `Set`**

```go
package edit

import (
	"errors"
	"strings"
	"testing"
)

func TestSetReplacesAValueAndKeepsItsComment(t *testing.T) {
	in := "# head\n[commit]\nlanguage = \"en\"  # prose is German\nthreshold = 2\n"
	got, err := Set(in, "commit", "language", `"de"`)
	if err != nil {
		t.Fatal(err)
	}
	want := "# head\n[commit]\nlanguage = \"de\"  # prose is German\nthreshold = 2\n"
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestSetReplacesAMultiLineList(t *testing.T) {
	in := "[index]\ninclude = [\n  \"a\",  # first\n  \"b]\",\n]\nexclude = []\n"
	got, err := Set(in, "index", "include", `["c"]`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "[index]\ninclude = [\"c\"]\nexclude = []\n" {
		t.Fatalf("got\n%s", got)
	}
}

func TestSetAddsAKeyAfterTheLastKeyOfItsSection(t *testing.T) {
	in := "[commit]\nlanguage = \"en\"\n\n# next\n[layout]\nwiki = \"docs\"\n"
	got, _ := Set(in, "commit", "threshold", "3")
	want := "[commit]\nlanguage = \"en\"\nthreshold = 3\n\n# next\n[layout]\nwiki = \"docs\"\n"
	if got != want {
		t.Fatalf("got\n%s", got)
	}
}

func TestSetAddsAMissingSectionAtTheEnd(t *testing.T) {
	got, _ := Set("[area]\nscope = \"s\"\n", "modules", "graph", "false")
	if got != "[area]\nscope = \"s\"\n\n[modules]\ngraph = false\n" {
		t.Fatalf("got\n%s", got)
	}
}

func TestSetTreatsASubTableAsItsOwnSection(t *testing.T) {
	in := "[verify]\ntimeout = \"600s\"\n[verify.go.lint]\ncommands = []\n"
	got, _ := Set(in, "verify", "max_parallel", "4")
	want := "[verify]\ntimeout = \"600s\"\nmax_parallel = 4\n[verify.go.lint]\ncommands = []\n"
	if got != want {
		t.Fatalf("got\n%s", got)
	}
}

func TestSetRefusesWhatItCannotPlace(t *testing.T) {
	for name, in := range map[string]string{
		"dotted key":     "commit.language = \"en\"\n",
		"inline table":   "commit = { language = \"en\" }\n",
		"twice":          "[commit]\nthreshold = 1\n[commit]\nlanguage = \"en\"\n",
		"key twice":      "[commit]\nlanguage = \"en\"\nlanguage = \"de\"\n",
		"quoted key":     "[commit]\n\"language\" = \"en\"\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Set(in, "commit", "language", `"de"`); !errors.Is(err, ErrAmbiguous) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestSetKeepsCRLF(t *testing.T) {
	got, _ := Set("[commit]\r\nlanguage = \"en\"\r\n", "commit", "language", `"de"`)
	if got != "[commit]\r\nlanguage = \"de\"\r\n" {
		t.Fatalf("%q", got)
	}
}

func TestRemoveDropsTheLineAndAnEmptySection(t *testing.T) {
	got, _ := Remove("[area]\nscope = \"s\"\n\n[modules]\ngraph = false\n", "modules", "graph")
	if got != "[area]\nscope = \"s\"\n" {
		t.Fatalf("%q", got)
	}
	got, _ = Remove("[commit]\nlanguage = \"de\"\nthreshold = 3\n", "commit", "language")
	if got != "[commit]\nthreshold = 3\n" {
		t.Fatalf("%q", got)
	}
}

func TestAppendBlockAddsATableListEntry(t *testing.T) {
	got := AppendBlock("[area]\nscope = \"s\"\n", "policy.commands.rules",
		[][2]string{{"regex", `'pip\s+install'`}, {"reason", `"uv, never pip."`}})
	want := "[area]\nscope = \"s\"\n\n[[policy.commands.rules]]\nregex = 'pip\\s+install'\nreason = \"uv, never pip.\"\n"
	if got != want {
		t.Fatalf("got\n%s", got)
	}
	if !strings.HasSuffix(AppendBlock("", "x", nil), "[[x]]\n") {
		t.Fatal("an empty file gets the block alone")
	}
}
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/config/edit -count=1`
Expected: FAIL (Paket fehlt).

- [ ] **Step 3: Implement `edit.go`**

```go
// Package edit changes .loomux/config.toml as text, so every comment and
// every line it does not touch stays as the human wrote it. What it cannot
// place without guessing, it refuses; the human edits that by hand.
package edit

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var ErrAmbiguous = errors.New("cannot place this change without guessing; edit the file by hand")

type lineForms struct{ header, listHeader, bareKey, otherKey *regexp.Regexp }

// forms compiles on first use: this package is linked into the binary every
// hook runs, and a package-level compile would run on each of those starts.
var forms = sync.OnceValue(func() lineForms {
	return lineForms{
		header:     regexp.MustCompile(`^\s*\[([A-Za-z0-9_.-]+)\]\s*(#.*)?$`),
		listHeader: regexp.MustCompile(`^\s*\[\[([A-Za-z0-9_.-]+)\]\]\s*(#.*)?$`),
		bareKey:    regexp.MustCompile(`^\s*([A-Za-z0-9_-]+)\s*=`),
		otherKey:   regexp.MustCompile(`^\s*("[^"]*"|'[^']*'|[A-Za-z0-9_-]+\.[A-Za-z0-9_.-]+)\s*=`),
	}
})

// line is one physical line and what it opens; a multi-line value spans
// several and is kept together as one entry.
type entry struct {
	first, last int    // line indexes, inclusive
	section     string // the [table] it sits in; "" before the first header
	key         string // "" for headers, comments and blank lines
	isHeader    bool
	isList      bool
}

func split(text string) (lines []string, eol string) {
	eol = "\n"
	if strings.Contains(text, "\r\n") {
		eol = "\r\n"
	}
	body := strings.TrimSuffix(text, eol)
	if body == "" {
		return nil, eol
	}
	return strings.Split(body, eol), eol
}

func join(lines []string, eol string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, eol) + eol
}

// scan walks the lines once. It refuses a document it cannot map: a dotted or
// quoted key could belong to the section being changed, and an inline table
// hides keys inside one line.
func scan(lines []string, section string) ([]entry, error) {
	var out []entry
	current := ""
	f := forms()
	header, listHeader, bareKey, otherKey := f.header, f.listHeader, f.bareKey, f.otherKey
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		switch {
		case listHeader.MatchString(l):
			current = listHeader.FindStringSubmatch(l)[1]
			out = append(out, entry{first: i, last: i, section: current, isHeader: true, isList: true})
		case header.MatchString(l):
			current = header.FindStringSubmatch(l)[1]
			out = append(out, entry{first: i, last: i, section: current, isHeader: true})
		case otherKey.MatchString(l):
			if current == section || strings.HasPrefix(otherKey.FindStringSubmatch(l)[1], section+".") || current == "" {
				return nil, fmt.Errorf("line %d: %w", i+1, ErrAmbiguous)
			}
			out = append(out, entry{first: i, last: i, section: current})
		case bareKey.MatchString(l):
			key := bareKey.FindStringSubmatch(l)[1]
			value := l[strings.Index(l, "=")+1:]
			if current == "" && key == strings.SplitN(section, ".", 2)[0] && strings.Contains(value, "{") {
				return nil, fmt.Errorf("line %d: %w", i+1, ErrAmbiguous)
			}
			last := i
			depth := brackets(value)
			for depth > 0 && last+1 < len(lines) {
				last++
				depth += brackets(lines[last])
			}
			out = append(out, entry{first: i, last: last, section: current, key: key})
			i = last
		default:
			out = append(out, entry{first: i, last: i, section: current})
		}
	}
	return out, nil
}

// brackets is the change in [ ] depth over s, outside strings and comments.
func brackets(s string) int {
	depth := 0
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == '\\' && quote == '"' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			return depth
		case c == '[':
			depth++
		case c == ']':
			depth--
		}
	}
	return depth
}

// comment is the trailing comment of a one-line value, with the spaces before it.
func comment(l string) string {
	var quote byte
	for i := strings.Index(l, "=") + 1; i < len(l); i++ {
		c := l[i]
		switch {
		case quote != 0:
			if c == '\\' && quote == '"' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			j := i
			for j > 0 && (l[j-1] == ' ' || l[j-1] == '\t') {
				j--
			}
			return l[j:]
		}
	}
	return ""
}

func locate(entries []entry, section, key string) (headers []entry, hits []entry, lastInSection int) {
	lastInSection = -1
	for _, e := range entries {
		if e.isHeader && e.section == section && !e.isList {
			headers = append(headers, e)
		}
		if !e.isHeader && e.section == section && e.key != "" {
			lastInSection = e.last
			if e.key == key {
				hits = append(hits, e)
			}
		}
		if e.isHeader && e.section == section && !e.isList && lastInSection < e.last {
			lastInSection = e.last
		}
	}
	return headers, hits, lastInSection
}

// Set writes key = literal into [section].
func Set(text, section, key, literal string) (string, error) {
	lines, eol := split(text)
	entries, err := scan(lines, section)
	if err != nil {
		return "", err
	}
	headers, hits, last := locate(entries, section, key)
	if len(headers) > 1 || len(hits) > 1 {
		return "", ErrAmbiguous
	}
	line := key + " = " + literal
	switch {
	case len(hits) == 1:
		hit := hits[0]
		if hit.first == hit.last {
			line += comment(lines[hit.first])
		}
		lines = append(lines[:hit.first], append([]string{line}, lines[hit.last+1:]...)...)
	case len(headers) == 1:
		lines = append(lines[:last+1], append([]string{line}, lines[last+1:]...)...)
	default:
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "["+section+"]", line)
	}
	return join(lines, eol), nil
}

// Remove takes key out of [section]; a section left with no key goes too,
// together with the blank line before it.
func Remove(text, section, key string) (string, error) {
	lines, eol := split(text)
	entries, err := scan(lines, section)
	if err != nil {
		return "", err
	}
	headers, hits, _ := locate(entries, section, key)
	if len(headers) > 1 || len(hits) > 1 {
		return "", ErrAmbiguous
	}
	if len(hits) == 0 {
		return text, nil
	}
	hit := hits[0]
	lines = append(lines[:hit.first], lines[hit.last+1:]...)
	rest, _ := scan(lines, section)
	keys := 0
	for _, e := range rest {
		if !e.isHeader && e.section == section && e.key != "" {
			keys++
		}
	}
	if keys == 0 && len(headers) == 1 {
		h := headers[0].first
		start := h
		if start > 0 && strings.TrimSpace(lines[start-1]) == "" {
			start--
		}
		lines = append(lines[:start], lines[h+1:]...)
	}
	return join(lines, eol), nil
}

// AppendBlock adds one [[section]] entry at the end.
func AppendBlock(text, section string, pairs [][2]string) string {
	lines, eol := split(text)
	if len(lines) > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, "[["+section+"]]")
	for _, p := range pairs {
		lines = append(lines, p[0]+" = "+p[1])
	}
	return join(lines, eol)
}
```

`scan` darf für die Prüfung „gehört der gepunktete Schlüssel zu dieser
Sektion“ vereinfacht werden, solange `TestSetRefusesWhatItCannotPlace` grün
bleibt. `header` passt nicht auf `[[…]]`, weil `[A-Za-z0-9_.-]+` keine
Klammer zulässt; darum prüft `scan` `listHeader` zuerst.

- [ ] **Step 4: Tests for `Render` and `Diff`**

```go
func TestRenderMakesALiteralPerKind(t *testing.T) {
	cases := []struct {
		kind  schema.Kind
		in    string
		want  string
		fails bool
	}{
		{schema.String, `docs/wiki`, `"docs/wiki"`, false},
		{schema.Enum, `de`, `"de"`, false},
		{schema.Int, `3`, `3`, false},
		{schema.Int, `three`, ``, true},
		{schema.Bool, `false`, `false`, false},
		{schema.Bool, `no`, ``, true},
		{schema.Duration, `90s`, `"90s"`, false},
		{schema.Duration, `soon`, ``, true},
		{schema.StringList, `a, b`, `["a", "b"]`, false},
		{schema.StringList, ``, `[]`, false},
		{schema.Table, `x`, ``, true},
	}
	for _, c := range cases {
		got, err := Render(c.kind, c.in)
		if (err != nil) != c.fails || got != c.want {
			t.Errorf("Render(%v, %q) = %q, %v", c.kind, c.in, got, err)
		}
	}
}

// TestRenderRoundTripsThroughTheDecoder holds Render and the input form of
// schema.Current together: what Render writes decodes back to what was typed.
func TestRenderRoundTripsThroughTheDecoder(t *testing.T) {
	for _, c := range []struct {
		kind schema.Kind
		in   string
		want any
	}{
		{schema.String, `docs\wiki "x"`, `docs\wiki "x"`},
		{schema.StringList, "a, b", []any{"a", "b"}},
	} {
		literal, err := Render(c.kind, c.in)
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if _, err := toml.Decode("v = "+literal, &doc); err != nil {
			t.Fatalf("%s does not decode: %v", literal, err)
		}
		if !reflect.DeepEqual(doc["v"], c.want) {
			t.Errorf("%q → %s → %#v", c.in, literal, doc["v"])
		}
	}
}

func TestDiffShowsChangedLinesWithContext(t *testing.T) {
	got := Diff("a\nb\nc\nd\n", "a\nB\nc\nd\n")
	want := "  a\n- b\n+ B\n  c\n"
	if got != want {
		t.Fatalf("got\n%s", got)
	}
	if Diff("x\n", "x\n") != "" {
		t.Fatal("no change, no diff")
	}
}
```

- [ ] **Step 5: Implement `value.go` and `diff.go`**

```go
package edit

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/schema"
)

// Render turns what a human typed into the TOML literal for kind.
func Render(kind schema.Kind, input string) (string, error) {
	input = strings.TrimSpace(input)
	switch kind {
	case schema.String, schema.Enum:
		return config.QuoteTOML(input), nil
	case schema.Int:
		if _, err := strconv.Atoi(input); err != nil {
			return "", fmt.Errorf("%q is not a whole number", input)
		}
		return input, nil
	case schema.Bool:
		if input != "true" && input != "false" {
			return "", fmt.Errorf("%q is not true or false", input)
		}
		return input, nil
	case schema.Duration:
		if _, err := time.ParseDuration(input); err != nil {
			return "", fmt.Errorf("%q is not a duration like 90s", input)
		}
		return config.QuoteTOML(input), nil
	case schema.StringList:
		if input == "" {
			return "[]", nil
		}
		parts := strings.Split(input, ",")
		for i, p := range parts {
			parts[i] = config.QuoteTOML(strings.TrimSpace(p))
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	default:
		return "", fmt.Errorf("a table is edited by hand")
	}
}
```

```go
package edit

import "strings"

// Diff is a small line diff: the changed lines with one line of context
// around each run. Configuration files are short; an LCS is not needed to
// show a one-line change, and every change here is one key.
func Diff(before, after string) string {
	a := strings.Split(strings.TrimSuffix(before, "\n"), "\n")
	b := strings.Split(strings.TrimSuffix(after, "\n"), "\n")
	start := 0
	for start < len(a) && start < len(b) && a[start] == b[start] {
		start++
	}
	endA, endB := len(a), len(b)
	for endA > start && endB > start && a[endA-1] == b[endB-1] {
		endA--
		endB--
	}
	if start == endA && start == endB {
		return ""
	}
	var out strings.Builder
	if start > 0 {
		out.WriteString("  " + a[start-1] + "\n")
	}
	for _, l := range a[start:endA] {
		out.WriteString("- " + l + "\n")
	}
	for _, l := range b[start:endB] {
		out.WriteString("+ " + l + "\n")
	}
	if endA < len(a) {
		out.WriteString("  " + a[endA] + "\n")
	}
	return out.String()
}
```

`config.QuoteTOML` gibt es (`registrywrite.go:51`); prüfen, dass es
Backslashes und Anführungszeichen maskiert — dann ist `Render` für Pfade
unter Windows richtig.

- [ ] **Step 6: Run all edit tests**

Run: `go test ./internal/config/edit -count=1`
Expected: PASS.

- [ ] **Step 7: Coverage**

Run: `go test ./internal/config/edit -count=1 -coverprofile=cover.out && go run ./cmd/loomux check gocover cover.out`
Expected: 100 % je Funktion. Fehlende Zweige (Escape in `brackets`,
Kommentar im mehrzeiligen Wert) mit eigenen Tests schließen.

- [ ] **Step 8: Commit**

```bash
git add internal/config/edit
git commit -m "feat(config): edit the configuration as text and keep its comments"
```

---

### Task 10: Wert und Herkunft

**Files:**
- Create: `internal/config/schema/current.go`
- Test: `internal/config/schema/current_test.go`

**Interfaces:**
- Produces:

```go
type Origin string
const (
	Set     Origin = "set"
	Default Origin = "default"
	Preset  Origin = "preset"
	Unset   Origin = "unset"
)
type Entry struct {
	Key    Key
	Value  string // TOML literal as written, or the default; "" when unset
	Input  string // the value in the form edit.Render accepts: unquoted, lists as "a, b"
	Origin Origin
	Count  int    // TableList: number of [[…]] blocks
}
func Current(text string) ([]Entry, error) // in Keys() order
```

- [ ] **Step 1: Write the failing test**

```go
func TestCurrentTellsWhereEachValueComesFrom(t *testing.T) {
	text := "[commit]\nlanguage = \"de\"\n[[policy.paths.rules]]\nmatch = [\"a\"]\nreason = \"r\"\n[[policy.paths.rules]]\nmatch = [\"b\"]\nreason = \"r\"\n[verify.go.lint]\ncommands = []\n"
	entries, err := Current(text)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Entry{}
	for _, e := range entries {
		byID[e.Key.ID()] = e
	}
	if e := byID["commit.language"]; e.Origin != Set || e.Value != `"de"` {
		t.Errorf("language %+v", e)
	}
	if e := byID["commit.threshold"]; e.Origin != Default || e.Value != "2" {
		t.Errorf("threshold %+v", e)
	}
	if e := byID["policy.paths.rules"]; e.Origin != Set || e.Count != 2 {
		t.Errorf("paths %+v", e)
	}
	if e := byID["layout.hub"]; e.Origin != Unset {
		t.Errorf("hub %+v", e)
	}
	if e := byID["verify.profiles"]; e.Origin != Preset {
		t.Errorf("profiles %+v", e)
	}
	if len(entries) != len(Keys()) {
		t.Fatalf("%d entries for %d keys", len(entries), len(Keys()))
	}
}

func TestInputFormIsWhatAHumanTypes(t *testing.T) {
	entries, err := Current("[layout]\nwiki = 'docs\\wiki \"x\"'\n[index]\ninclude = [\"a\", \"b\"]\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		switch e.Key.ID() {
		case "layout.wiki":
			if e.Input != `docs\wiki "x"` {
				t.Errorf("wiki input %q", e.Input)
			}
		case "index.include":
			if e.Input != "a, b" {
				t.Errorf("include input %q", e.Input)
			}
		case "commit.threshold":
			if e.Input != "2" {
				t.Errorf("default input %q", e.Input)
			}
		}
	}
}

func TestCurrentRefusesBrokenTOML(t *testing.T) {
	if _, err := Current("[x"); err == nil {
		t.Fatal("want an error")
	}
}
```

- [ ] **Step 2: Run to see it fail**

Run: `go test ./internal/config/schema -run Current -count=1`
Expected: FAIL.

- [ ] **Step 3: Implement**

```go
package schema

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

// Current pairs every key with its value in text and says where that value
// comes from. The value shown for a set key is re-encoded from the decoded
// document, so a list written across lines is shown on one.
func Current(text string) ([]Entry, error) {
	doc := map[string]any{}
	if err := toml.Unmarshal([]byte(text), &doc); err != nil {
		return nil, fmt.Errorf(".loomux/config.toml: not valid TOML: %w", err)
	}
	var out []Entry
	for _, k := range Keys() {
		value, found := lookup(doc, k)
		e := Entry{Key: k}
		switch {
		case k.Kind == TableList:
			list, _ := value.([]map[string]any)
			e.Count = len(list)
			e.Origin = Unset
			if found {
				e.Origin = Set
			}
		case found:
			e.Origin, e.Value, e.Input = Set, literal(value), inputForm(value)
		case k.Section == "verify" && k.Kind == Table:
			e.Origin = Preset
		case k.Default != "":
			e.Origin, e.Value = Default, k.Default
			var decoded map[string]any
			if _, err := toml.Decode("v = "+k.Default, &decoded); err == nil {
				e.Input = inputForm(decoded["v"])
			}
		default:
			e.Origin = Unset
		}
		out = append(out, e)
	}
	return out, nil
}

func lookup(doc map[string]any, k Key) (any, bool) {
	var node any = doc
	path := strings.Split(k.Section, ".")
	if k.Name != "" {
		path = append(path, k.Name)
	}
	for _, part := range path {
		table, ok := node.(map[string]any)
		if !ok {
			return nil, false
		}
		if node, ok = table[part]; !ok {
			return nil, false
		}
	}
	return node, true
}

// inputForm is value the way a human types it and edit.Render reads it:
// a string without quotes or escapes, a list as "a, b", anything else as Go
// prints it. Taking the quotes off the TOML literal instead would leave its
// escapes in, and Render would escape them a second time.
func inputForm(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = fmt.Sprint(item)
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprint(v)
	}
}

// literal is value as one line of TOML.
func literal(value any) string {
	var b strings.Builder
	_ = toml.NewEncoder(&b).Encode(map[string]any{"v": value})
	return strings.TrimSpace(strings.TrimPrefix(b.String(), "v = "))
}
```

Für eine Tabelle (`profiles`, `roles`) gibt `literal` mehrere Zeilen aus;
`Current` zeigt sie so, die Oberfläche kürzt sie auf die erste Zeile plus
„…“. Wenn `toml.Encoder` für eine Tabelle einen `[v]`-Kopf statt `v = `
schreibt, für `Kind == Table` stattdessen `fmt.Sprint(value)` zeigen und das
im Test festhalten.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/config/schema -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/schema
git commit -m "feat(config): show each setting with its value and origin"
```

---

### Task 11: `loomux config list|get|set` und `--global`

**Files:**
- Create: `internal/cli/config.go`
- Modify: `internal/cli/commands.go` (`"config": configCommand`)
- Test: `internal/cli/config_test.go`

**Interfaces:**
- Consumes: `schema.Keys/Lookup/GlobalKeys/Current/Validate`,
  `edit.Set/Remove/Render/Diff`, `lock.ReplaceText`, `config.ManifestPath`,
  `config.StateDir()`.
- Produces: `func configCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int`;
  die Seam `var configInteractive = runConfigUI` (Task 13 füllt sie; bis dahin
  meldet sie „needs a terminal“ und gibt 2).

Formen:
- `loomux config list [--json] [--root DIR | --global]`
- `loomux config get <id> [--root DIR | --global]` — gibt den Wert aus (Set
  oder Default), Exit 1 bei unbekanntem Schlüssel, bei `Unset` leere Zeile
  und 0.
- `loomux config set <id> <value> [--yes] [--root DIR | --global]` — `value`
  ist die Eingabe für `edit.Render`; der Wert `default` entfernt die Zeile
  (`edit.Remove`). Ist der neue Wert gleich der Vorgabe, entfernt `set` die
  Zeile ebenso (Vorgaben werden nie geschrieben). Zeigt `edit.Diff`, fragt
  `write these changes? [y/N]` auf stderr und liest eine Zeile von stdin;
  `--yes` überspringt die Frage.
- `loomux config` ohne Unterbefehl → `configInteractive`.
- Globale Datei: `<StateDir>/config.toml`; `--global` nutzt `GlobalKeys()`
  statt `Keys()`, und `Validate` entfällt, solange `GlobalKeys()` leer ist —
  jeder `set` auf `--global` scheitert dann an „unknown key“.

- [ ] **Step 1: Write the failing tests**

```go
func configRoot(t *testing.T, text string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), text)
	return root
}

func runConfig(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(append([]string{"config"}, args...), strings.NewReader(stdin), &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestConfigListShowsValueAndOrigin(t *testing.T) {
	root := configRoot(t, "[commit]\nlanguage = \"de\"\n")
	code, out, _ := runConfig(t, "", "list", "--root", root)
	if code != 0 {
		t.Fatal(code)
	}
	for _, want := range []string{"commit.language", `"de"`, "set", "commit.threshold", "default"} {
		if !strings.Contains(out, want) {
			t.Errorf("list lacks %q:\n%s", want, out)
		}
	}
}

func TestConfigListJSON(t *testing.T) {
	root := configRoot(t, "")
	code, out, _ := runConfig(t, "", "list", "--json", "--root", root)
	var rows []map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &rows) != nil || len(rows) == 0 {
		t.Fatalf("code %d, out %s", code, out)
	}
}

func TestConfigGet(t *testing.T) {
	root := configRoot(t, "[commit]\nlanguage = \"de\"\n")
	if code, out, _ := runConfig(t, "", "get", "commit.language", "--root", root); code != 0 || out != "\"de\"\n" {
		t.Fatalf("%d %q", code, out)
	}
	if code, out, _ := runConfig(t, "", "get", "commit.threshold", "--root", root); code != 0 || out != "2\n" {
		t.Fatalf("%d %q", code, out)
	}
	if code, _, errOut := runConfig(t, "", "get", "commit.nope", "--root", root); code != 1 || !strings.Contains(errOut, "unknown") {
		t.Fatalf("%d %q", code, errOut)
	}
}

func TestConfigSetAsksAndWrites(t *testing.T) {
	root := configRoot(t, "# mine\n[commit]\nlanguage = \"en\"  # keep\n")
	code, _, errOut := runConfig(t, "y\n", "set", "commit.language", "de", "--root", root)
	if code != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
	if !strings.Contains(errOut, "+ language = \"de\"  # keep") {
		t.Errorf("no diff shown:\n%s", errOut)
	}
	data, _ := os.ReadFile(filepath.Join(root, ".loomux", "config.toml"))
	if string(data) != "# mine\n[commit]\nlanguage = \"de\"  # keep\n" {
		t.Fatalf("file:\n%s", data)
	}
}

func TestConfigSetWritesNothingWhenDeclined(t *testing.T) {
	root := configRoot(t, "[commit]\nlanguage = \"en\"\n")
	code, _, errOut := runConfig(t, "n\n", "set", "commit.language", "de", "--root", root)
	data, _ := os.ReadFile(filepath.Join(root, ".loomux", "config.toml"))
	if code != 0 || string(data) != "[commit]\nlanguage = \"en\"\n" || !strings.Contains(errOut, "nothing written") {
		t.Fatalf("%d %q %s", code, errOut, data)
	}
}

func TestConfigSetToTheDefaultRemovesTheLine(t *testing.T) {
	root := configRoot(t, "[area]\nscope = \"s\"\n\n[commit]\nlanguage = \"de\"\n")
	runConfig(t, "", "set", "commit.language", "en", "--yes", "--root", root)
	data, _ := os.ReadFile(filepath.Join(root, ".loomux", "config.toml"))
	if string(data) != "[area]\nscope = \"s\"\n" {
		t.Fatalf("file:\n%s", data)
	}
}

func TestConfigSetRefusesWhatAReaderRefuses(t *testing.T) {
	root := configRoot(t, "")
	code, _, errOut := runConfig(t, "", "set", "commit.language", "fr", "--yes", "--root", root)
	if code != 1 || !strings.Contains(errOut, "language") {
		t.Fatalf("%d %s", code, errOut)
	}
}

func TestConfigSetRefusesATable(t *testing.T) {
	root := configRoot(t, "")
	if code, _, _ := runConfig(t, "", "set", "verify.profiles", "x", "--yes", "--root", root); code != 1 {
		t.Fatal(code)
	}
}

func TestConfigSetRefusesAnAmbiguousFile(t *testing.T) {
	root := configRoot(t, "commit.language = \"en\"\n")
	code, _, errOut := runConfig(t, "", "set", "commit.language", "de", "--yes", "--root", root)
	if code != 1 || !strings.Contains(errOut, "by hand") {
		t.Fatalf("%d %s", code, errOut)
	}
}

func TestConfigGlobalKnowsNoKeyYet(t *testing.T) {
	t.Setenv("LOOMUX_STATE_DIR", t.TempDir())
	if code, out, _ := runConfig(t, "", "list", "--global"); code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("%d %q", code, out)
	}
	if code, _, _ := runConfig(t, "", "set", "model.enabled", "true", "--yes", "--global"); code != 1 {
		t.Fatal(code)
	}
}

func TestConfigUsage(t *testing.T) {
	for _, args := range [][]string{{"nope"}, {"get"}, {"set", "x"}, {"list", "--global", "--root", "x"}} {
		if code, _, _ := runConfig(t, "", args...); code != 2 {
			t.Errorf("%v: code %d, want 2", args, code)
		}
	}
}

func TestConfigWithoutATerminalNamesTheOtherForms(t *testing.T) {
	// Chdir into a project so the target resolves; the interactive seam is
	// the transitional one of this task, which never opens a console.
	t.Chdir(configRoot(t, ""))
	code, _, errOut := runConfig(t, "")
	if code != 2 || !strings.Contains(errOut, "config list") {
		t.Fatalf("%d %s", code, errOut)
	}
}
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/cli -run Config -count=1`
Expected: FAIL.

- [ ] **Step 3: Implement `config.go`**

```go
package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/edit"
	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/lock"
)

// configInteractive is the full-screen form; the seam lets tests of the
// plain forms run without a terminal.
var configInteractive = func(target configTarget, stdin io.Reader, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "loomux config: the interactive form needs a terminal; use `loomux config list`, `get` or `set`")
	return 2
}

// configTarget is the one file a run reads and writes, and the keys it knows.
type configTarget struct {
	path   string
	keys   []schema.Key
	global bool
}

func configUsage(stderr io.Writer) int {
	fmt.Fprintln(stderr, "usage: loomux config [list [--json] | get <key> | set <key> <value> [--yes]] [--root DIR | --global]")
	return 2
}

func configCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	sub := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub, args = args[0], args[1:]
	}
	flags := flag.NewFlagSet("loomux config", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "project root; found upwards when empty")
	global := flags.Bool("global", false, "the machine-wide file in the state directory")
	yes := flags.Bool("yes", false, "write without asking")
	asJSON := flags.Bool("json", false, "list as JSON")
	positional, err := parseInterspersed(flags, args)
	// The call is judged before the file is looked for: a wrong call is a 2
	// wherever it is typed, also outside any project.
	arity := map[string]int{"": 0, "list": 0, "get": 1, "set": 2}
	want, known := arity[sub]
	if err != nil || (*global && *root != "") || !known || len(positional) != want {
		return configUsage(stderr)
	}
	target, err := resolveConfigTarget(*root, *global)
	if err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	switch {
	case sub == "" && len(positional) == 0:
		return configInteractive(target, stdin, stdout, stderr)
	case sub == "list" && len(positional) == 0:
		return configList(target, *asJSON, stdout, stderr)
	case sub == "get" && len(positional) == 1:
		return configGet(target, positional[0], stdout, stderr)
	default:
		return configSet(target, positional[0], positional[1], *yes, stdin, stderr)
	}
}

func resolveConfigTarget(root string, global bool) (configTarget, error) {
	if global {
		return configTarget{path: filepath.Join(config.StateDir(), "config.toml"), keys: schema.GlobalKeys(), global: true}, nil
	}
	if root == "" {
		found, err := hosts.FindRoot(".")
		if err != nil {
			return configTarget{}, err
		}
		root = found
	}
	return configTarget{path: config.ManifestPath(root), keys: schema.Keys()}, nil
}

func (t configTarget) read() (string, error) {
	data, err := os.ReadFile(t.path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return string(data), err
}

func (t configTarget) entries(text string) ([]schema.Entry, error) {
	all, err := schema.Current(text)
	if err != nil {
		return nil, err
	}
	var out []schema.Entry
	for _, e := range all {
		for _, k := range t.keys {
			if k.ID() == e.Key.ID() {
				out = append(out, e)
			}
		}
	}
	return out, nil
}

func (t configTarget) lookup(id string) (schema.Key, bool) {
	for _, k := range t.keys {
		if k.ID() == id {
			return k, true
		}
	}
	return schema.Key{}, false
}

func configList(t configTarget, asJSON bool, stdout, stderr io.Writer) int {
	text, err := t.read()
	if err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	entries, err := t.entries(text)
	if err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	if asJSON {
		rows := []map[string]any{}
		for _, e := range entries {
			rows = append(rows, map[string]any{"key": e.Key.ID(), "module": e.Key.Module, "value": e.Value, "origin": e.Origin, "count": e.Count})
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rows)
		return 0
	}
	w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	for _, e := range entries {
		value := e.Value
		if e.Key.Kind == schema.TableList {
			value = fmt.Sprintf("%d entries", e.Count)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Key.Module, e.Key.ID(), firstLine(value), e.Origin)
	}
	_ = w.Flush()
	return 0
}

func firstLine(s string) string {
	if first, _, cut := strings.Cut(s, "\n"); cut {
		return first + " …"
	}
	return s
}

func configGet(t configTarget, id string, stdout, stderr io.Writer) int {
	if _, ok := t.lookup(id); !ok {
		fmt.Fprintf(stderr, "loomux config: unknown key %q\n", id)
		return 1
	}
	text, err := t.read()
	if err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	entries, err := t.entries(text)
	if err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	for _, e := range entries {
		if e.Key.ID() == id {
			fmt.Fprintln(stdout, e.Value)
		}
	}
	return 0
}

// proposeChange computes the new text for one key; callers show and confirm
// it. It is shared with the interactive form.
func proposeChange(t configTarget, text, id, input string) (string, error) {
	key, ok := t.lookup(id)
	if !ok {
		return "", fmt.Errorf("unknown key %q", id)
	}
	if key.Kind == schema.Table || key.Kind == schema.TableList {
		return "", fmt.Errorf("%s is a table; edit it by hand", id)
	}
	var next string
	var err error
	literal := ""
	if input != "default" {
		if literal, err = edit.Render(key.Kind, input); err != nil {
			return "", err
		}
	}
	if input == "default" || literal == key.Default {
		next, err = edit.Remove(text, key.Section, key.Name)
	} else {
		next, err = edit.Set(text, key.Section, key.Name, literal)
	}
	if err != nil {
		return "", err
	}
	if !t.global {
		if err := schema.Validate(next); err != nil {
			return "", err
		}
	}
	return next, nil
}

func configSet(t configTarget, id, input string, yes bool, stdin io.Reader, stderr io.Writer) int {
	text, err := t.read()
	if err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	next, err := proposeChange(t, text, id, input)
	if err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	diff := edit.Diff(text, next)
	if diff == "" {
		fmt.Fprintln(stderr, "loomux config: already so; nothing written")
		return 0
	}
	fmt.Fprint(stderr, diff)
	if !yes {
		fmt.Fprint(stderr, "write these changes? [y/N] ")
		answer, _ := bufio.NewReader(stdin).ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			fmt.Fprintln(stderr, "loomux config: declined; nothing written")
			return 0
		}
	}
	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	if err := lock.ReplaceText(t.path, next); err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	fmt.Fprintf(stderr, "loomux config: wrote %s\n", t.path)
	return 0
}
```

`parseInterspersed` gibt es in `internal/cli/interspersed.go`; vor dem
Benutzen seine Signatur lesen und den Aufruf anpassen. Die Meldungen „nothing
written“ in beiden Fällen enthalten die Wörter, die die Tests suchen.

In `commands.go` die Zeile `"config":    configCommand,` alphabetisch
einfügen und den Kommentar über der Tabelle unverändert lassen.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/cli -run Config -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/config.go internal/cli/config_test.go internal/cli/commands.go
git commit -m "feat(cli): list, get and set the configuration with loomux config"
```

---

### Task 12: Der Hook-Pfad bleibt schlank

**Files:**
- Modify: `internal/cli/imports_test.go`

- [ ] **Step 1: Write the test**

Unter `forbiddenMaintenanceForHooks` eine eigene Liste und ein eigener Test:

```go
// forbiddenConfigUIForHooks is the configuration command's weight: the
// schema pulls every reader, the editor its text surgery, the interface the
// terminal. [modules] on the per-edit path is read by config.ReadModules
// alone.
func forbiddenConfigUIForHooks() []string {
	return []string{
		"github.com/xidus90/loomux/internal/config/schema",
		"github.com/xidus90/loomux/internal/config/edit",
		"github.com/xidus90/loomux/internal/tui",
		"golang.org/x/term",
	}
}

func TestHooksNeverImportTheConfigurationCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("asks the go tool for the import graph")
	}
	hooks, err := dependencies(hooksPackage)
	if err != nil {
		t.Fatal(err)
	}
	cli, err := dependencies("github.com/xidus90/loomux/internal/cli")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range forbiddenConfigUIForHooks() {
		if hooks[forbidden] {
			t.Errorf("%s depends on %s", hooksPackage, forbidden)
		}
		if !cli[forbidden] && forbidden != "golang.org/x/term" && forbidden != "github.com/xidus90/loomux/internal/tui" {
			t.Errorf("the command line no longer reaches %s; the list is stale", forbidden)
		}
	}
}
```

Die beiden Ausnahmen im zweiten `if` entfallen in Task 14, sobald `tui` und
`x/term` vom Befehl erreicht werden; dort wird dieser Test angepasst.

- [ ] **Step 2: Run it**

Run: `go test ./internal/cli -run ConfigurationCommand -count=1`
Expected: PASS (die Hooks importieren nichts davon; `cli` erreicht `schema`
und `edit` seit Task 11).

- [ ] **Step 3: Commit**

```bash
git add internal/cli/imports_test.go
git commit -m "test(cli): keep the configuration command off the per-edit path"
```

---

### Task 13: Die Oberfläche

**Files:**
- Create: `internal/tui/terminal.go`, `internal/tui/keys.go`, `internal/tui/list.go`, `internal/tui/input.go`, `internal/tui/confirm.go`, `internal/tui/fake.go` (nur für Tests? nein — `fake.go` exportiert `Script`, damit `internal/cli` es ebenfalls nutzt)
- Create: `internal/tui/raw.go`, `internal/tui/raw_windows.go`, `internal/tui/raw_other.go`
- Test: `internal/tui/*_test.go`
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Produces:

```go
type Key struct {
	Rune rune   // printable input; 0 otherwise
	Name string // "up", "down", "enter", "esc", "backspace", "tab", "ctrl-c", "/"-less
}
type Terminal interface {
	ReadKey() (Key, error)
	Write(p []byte) (int, error)
	Size() (width, height int)
}
func Open(in, out *os.File) (Terminal, func() error, error) // raw mode; restore
func IsTerminal(f *os.File) bool
func Script(width, height int, keys ...Key) *Scripted       // Terminal for tests
func (s *Scripted) Output() string                          // everything written, ANSI stripped

type Row struct {
	Group string // module; rows of one group are contiguous
	Label string // "commit.language"
	Value string
	Note  string // origin
}
// List shows rows, lets the human move and filter with '/', and returns the
// index chosen with enter, or -1 on esc/q.
func List(t Terminal, title string, rows []Row) (int, error)
// Input edits one line; choices, when given, cycle with tab and are the only
// accepted answers; check runs on enter and shows its error in place.
func Input(t Terminal, prompt, initial string, choices []string, check func(string) error) (string, bool, error)
// Confirm shows text (a diff) and asks y/n.
func Confirm(t Terminal, text, question string) (bool, error)
```

- [ ] **Step 1: Abhängigkeit holen**

Run: `go get golang.org/x/term@latest && go mod tidy`
Danach in `go.mod` prüfen, dass `golang.org/x/sys` nicht zurückgestuft wurde
(Erinnerung „Neueste Abhängigkeitsversionen“: die übrigen Pins mitprüfen).

- [ ] **Step 2: Tastendekodierung — failing test**

```go
func TestDecodeReadsKeysAndEscapeSequences(t *testing.T) {
	in := bytes.NewReader([]byte("a\x1b[A\x1b[B\r\x7f\t\x03\x1b"))
	d := newDecoder(in)
	want := []Key{{Rune: 'a'}, {Name: "up"}, {Name: "down"}, {Name: "enter"}, {Name: "backspace"}, {Name: "tab"}, {Name: "ctrl-c"}, {Name: "esc"}}
	for i, w := range want {
		got, err := d.next()
		if err != nil || got != w {
			t.Fatalf("key %d: got %+v, %v; want %+v", i, got, err, w)
		}
	}
	if _, err := d.next(); err != io.EOF {
		t.Fatalf("want EOF, got %v", err)
	}
}

func TestDecodeReadsUTF8(t *testing.T) {
	d := newDecoder(strings.NewReader("ü"))
	if k, _ := d.next(); k.Rune != 'ü' {
		t.Fatalf("%+v", k)
	}
}
```

- [ ] **Step 3: Implement `keys.go`**

```go
package tui

import (
	"bufio"
	"io"
	"unicode/utf8"
)

type decoder struct{ r *bufio.Reader }

func newDecoder(r io.Reader) *decoder { return &decoder{r: bufio.NewReader(r)} }

// next reads one key. A lone ESC is esc; ESC [ A..D are the arrows. Windows
// consoles in VT input mode send the same sequences, which is why raw_windows
// switches that mode on.
func (d *decoder) next() (Key, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return Key{}, err
	}
	switch b {
	case '\r', '\n':
		return Key{Name: "enter"}, nil
	case 0x7f, 0x08:
		return Key{Name: "backspace"}, nil
	case '\t':
		return Key{Name: "tab"}, nil
	case 0x03:
		return Key{Name: "ctrl-c"}, nil
	case 0x1b:
		if d.r.Buffered() == 0 {
			return Key{Name: "esc"}, nil
		}
		if next, _ := d.r.ReadByte(); next != '[' {
			return Key{Name: "esc"}, nil
		}
		code, err := d.r.ReadByte()
		if err != nil {
			return Key{Name: "esc"}, nil
		}
		switch code {
		case 'A':
			return Key{Name: "up"}, nil
		case 'B':
			return Key{Name: "down"}, nil
		case 'C':
			return Key{Name: "right"}, nil
		case 'D':
			return Key{Name: "left"}, nil
		}
		return Key{Name: "esc"}, nil
	}
	if b < utf8.RuneSelf {
		return Key{Rune: rune(b)}, nil
	}
	_ = d.r.UnreadByte()
	r, _, err := d.r.ReadRune()
	return Key{Rune: r}, err
}
```

`d.r.Buffered() == 0` unterscheidet ein allein gedrücktes ESC von einer
Sequenz nur, solange die Konsole die Sequenz in einem Rutsch liefert; das ist
unter Windows Terminal und conhost im VT-Modus so (Messpunkt der Spec,
„Offen und vor dem Bau zu messen“ — in Step 11 von Hand prüfen).

- [ ] **Step 4: `Scripted` und `terminal.go`**

```go
package tui

import (
	"io"
	"regexp"
	"strings"
)

type Key struct {
	Rune rune
	Name string
}

type Terminal interface {
	ReadKey() (Key, error)
	Write(p []byte) (int, error)
	Size() (width, height int)
}

// Scripted is a terminal that plays keys and records what was drawn.
type Scripted struct {
	keys          []Key
	out           strings.Builder
	width, height int
}

func Script(width, height int, keys ...Key) *Scripted {
	return &Scripted{keys: keys, width: width, height: height}
}

func (s *Scripted) ReadKey() (Key, error) {
	if len(s.keys) == 0 {
		return Key{}, io.EOF
	}
	k := s.keys[0]
	s.keys = s.keys[1:]
	return k, nil
}

func (s *Scripted) Write(p []byte) (int, error) { return s.out.Write(p) }
func (s *Scripted) Size() (int, int)            { return s.width, s.height }

// ansi compiles on first use; see the package-init rule in the plan's
// global constraints.
var ansi = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`) })

// Output is what was written with the escape sequences taken out.
func (s *Scripted) Output() string {
	return strings.ReplaceAll(ansi().ReplaceAllString(s.out.String(), ""), "\r\n", "\n")
}

// Keys builds a key list from a short spelling: "down", "enter", or text.
func Keys(parts ...string) []Key {
	var out []Key
	for _, p := range parts {
		switch p {
		case "up", "down", "left", "right", "enter", "esc", "backspace", "tab", "ctrl-c":
			out = append(out, Key{Name: p})
		default:
			for _, r := range p {
				out = append(out, Key{Rune: r})
			}
		}
	}
	return out
}
```

Die Imports von `terminal.go` sind `io`, `regexp`, `strings`, `sync`.

**Zeilenenden:** Im Raw-Modus springt `\n` unter POSIX nur eine Zeile tiefer,
nicht an den Zeilenanfang. Die Bausteine schreiben darum `\r\n`
(`const eol = "\r\n"` in `list.go`, überall statt `"\n"` in Ausgaben an das
Terminal); `Scripted.Output` normalisiert `\r\n` zu `\n`, damit die Tests
unverändert bleiben.

- [ ] **Step 5: `List` — failing tests**

```go
func rows() []Row {
	return []Row{
		{Group: "base", Label: "commit.language", Value: `"de"`, Note: "set"},
		{Group: "base", Label: "commit.threshold", Value: "2", Note: "default"},
		{Group: "brain", Label: "layout.wiki", Value: `"docs/wiki"`, Note: "set"},
	}
}

func TestListMovesAndChooses(t *testing.T) {
	term := Script(80, 20, Keys("down", "down", "enter")...)
	got, err := List(term, "loomux config", rows())
	if err != nil || got != 2 {
		t.Fatalf("got %d, %v", got, err)
	}
	out := term.Output()
	for _, want := range []string{"base", "brain", "commit.language", `"docs/wiki"`, "default"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q", want)
		}
	}
}

func TestListStopsAtTheEnds(t *testing.T) {
	term := Script(80, 20, Keys("up", "enter")...)
	if got, _ := List(term, "t", rows()); got != 0 {
		t.Fatal(got)
	}
	term = Script(80, 20, Keys("down", "down", "down", "enter")...)
	if got, _ := List(term, "t", rows()); got != 2 {
		t.Fatal(got)
	}
}

func TestListFilters(t *testing.T) {
	term := Script(80, 20, Keys("/", "wiki", "enter", "enter")...)
	if got, _ := List(term, "t", rows()); got != 2 {
		t.Fatalf("filtered choice must map back to the full list, got %d", got)
	}
}

func TestListFilterBackspaceAndEsc(t *testing.T) {
	term := Script(80, 20, Keys("/", "wx", "backspace", "esc", "enter")...)
	if got, _ := List(term, "t", rows()); got != 0 {
		t.Fatalf("esc in the filter clears it, got %d", got)
	}
}

func TestListQuits(t *testing.T) {
	for _, k := range [][]Key{Keys("esc"), Keys("q"), Keys("ctrl-c")} {
		if got, err := List(Script(80, 20, k...), "t", rows()); got != -1 || err != nil {
			t.Fatalf("%v: %d %v", k, got, err)
		}
	}
}

func TestListReportsTheEndOfInput(t *testing.T) {
	if _, err := List(Script(80, 20), "t", rows()); err != io.EOF {
		t.Fatal(err)
	}
}

func TestListScrollsInASmallWindow(t *testing.T) {
	many := make([]Row, 30)
	for i := range many {
		many[i] = Row{Group: "g", Label: fmt.Sprintf("k%02d", i)}
	}
	keys := make([]Key, 0, 31)
	for range 25 {
		keys = append(keys, Key{Name: "down"})
	}
	term := Script(40, 8, append(keys, Key{Name: "enter"})...)
	if got, _ := List(term, "t", many); got != 25 {
		t.Fatal(got)
	}
}
```

- [ ] **Step 6: Implement `list.go`**

```go
package tui

import (
	"fmt"
	"strings"
)

const (
	clearScreen = "\x1b[H\x1b[2J"
	reverse     = "\x1b[7m"
	reset       = "\x1b[0m"
)

func List(t Terminal, title string, rows []Row) (int, error) {
	cursor, filter, filtering := 0, "", false
	for {
		visible := matching(rows, filter)
		if cursor >= len(visible) {
			cursor = max(len(visible)-1, 0)
		}
		draw(t, title, rows, visible, cursor, filter, filtering)
		k, err := t.ReadKey()
		if err != nil {
			return -1, err
		}
		if filtering {
			switch {
			case k.Name == "enter":
				filtering = false
			case k.Name == "esc":
				filtering, filter = false, ""
			case k.Name == "backspace" && filter != "":
				filter = filter[:len(filter)-len(string([]rune(filter)[len([]rune(filter))-1]))]
			case k.Rune != 0:
				filter += string(k.Rune)
			}
			continue
		}
		switch {
		case k.Name == "up" && cursor > 0:
			cursor--
		case k.Name == "down" && cursor < len(visible)-1:
			cursor++
		case k.Name == "enter" && len(visible) > 0:
			return visible[cursor], nil
		case k.Name == "esc" || k.Name == "ctrl-c" || k.Rune == 'q':
			return -1, nil
		case k.Rune == '/':
			filtering = true
		}
	}
}

func matching(rows []Row, filter string) []int {
	var out []int
	for i, r := range rows {
		if strings.Contains(strings.ToLower(r.Label+" "+r.Value), strings.ToLower(filter)) {
			out = append(out, i)
		}
	}
	return out
}

func draw(t Terminal, title string, rows []Row, visible []int, cursor int, filter string, filtering bool) {
	_, height := t.Size()
	var b strings.Builder
	b.WriteString(clearScreen + title + "\n")
	if filtering || filter != "" {
		fmt.Fprintf(&b, "/%s\n", filter)
	} else {
		b.WriteString("↑↓ move · enter change · / filter · q quit\n")
	}
	room := max(height-3, 1)
	top := max(cursor-room+1, 0)
	group := ""
	for n, i := range visible[top:min(top+room, len(visible))] {
		r := rows[i]
		if r.Group != group {
			fmt.Fprintf(&b, "[%s]\n", r.Group)
			group = r.Group
		}
		line := fmt.Sprintf("  %-28s %-24s %s", r.Label, r.Value, r.Note)
		if top+n == cursor {
			line = reverse + line + reset
		}
		b.WriteString(line + "\n")
	}
	_, _ = t.Write([]byte(b.String()))
}
```

Die Backspace-Zeile im Filter auf `[]rune` umstellen, wenn sie beim Lesen
unklar ist: `rs := []rune(filter); filter = string(rs[:len(rs)-1])`.

- [ ] **Step 7: `Input` und `Confirm` — failing tests**

```go
func TestInputEditsAndChecks(t *testing.T) {
	check := func(s string) error {
		if s == "bad" {
			return errors.New("no good")
		}
		return nil
	}
	term := Script(80, 10, Keys("backspace", "backspace", "backspace", "bad", "enter", "backspace", "backspace", "backspace", "fine", "enter")...)
	got, ok, err := Input(term, "value", "old", nil, check)
	if err != nil || !ok || got != "fine" {
		t.Fatalf("%q %v %v", got, ok, err)
	}
	if !strings.Contains(term.Output(), "no good") {
		t.Fatal("the check's error must be shown")
	}
}

func TestInputCyclesChoices(t *testing.T) {
	term := Script(80, 10, Keys("tab", "enter")...)
	got, ok, _ := Input(term, "language", "en", []string{"en", "de"}, nil)
	if !ok || got != "de" {
		t.Fatalf("%q %v", got, ok)
	}
	term = Script(80, 10, Keys("tab", "tab", "enter")...)
	if got, _, _ := Input(term, "language", "en", []string{"en", "de"}, nil); got != "en" {
		t.Fatalf("tab wraps, got %q", got)
	}
}

func TestInputCancels(t *testing.T) {
	if _, ok, err := Input(Script(80, 10, Keys("esc")...), "v", "x", nil, nil); ok || err != nil {
		t.Fatal(ok, err)
	}
	if _, _, err := Input(Script(80, 10), "v", "x", nil, nil); err != io.EOF {
		t.Fatal(err)
	}
}

func TestConfirm(t *testing.T) {
	for keys, want := range map[string]bool{"y": true, "n": false} {
		term := Script(80, 10, Keys(keys)...)
		got, err := Confirm(term, "- a\n+ b\n", "write?")
		if err != nil || got != want || !strings.Contains(term.Output(), "+ b") {
			t.Fatalf("%s: %v %v", keys, got, err)
		}
	}
	term := Script(80, 10, Keys("x", "esc")...)
	if got, _ := Confirm(term, "", "write?"); got {
		t.Fatal("esc is no")
	}
	if _, err := Confirm(Script(80, 10), "", "write?"); err != io.EOF {
		t.Fatal(err)
	}
}
```

- [ ] **Step 8: Implement `input.go` and `confirm.go`**

```go
package tui

import (
	"fmt"
	"slices"
)

func Input(t Terminal, prompt, initial string, choices []string, check func(string) error) (string, bool, error) {
	value, problem := []rune(initial), ""
	for {
		msg := ""
		if problem != "" {
			msg = "\n  " + problem
		}
		hint := ""
		if len(choices) > 0 {
			hint = fmt.Sprintf("  (tab: %v)", choices)
		}
		_, _ = fmt.Fprintf(t, "%s%s: %s%s%s", clearScreen, prompt, string(value), hint, msg)
		k, err := t.ReadKey()
		if err != nil {
			return "", false, err
		}
		switch {
		case k.Name == "esc" || k.Name == "ctrl-c":
			return "", false, nil
		case k.Name == "enter":
			if check != nil {
				if err := check(string(value)); err != nil {
					problem = err.Error()
					continue
				}
			}
			return string(value), true, nil
		case k.Name == "tab" && len(choices) > 0:
			i := slices.Index(choices, string(value))
			value = []rune(choices[(i+1)%len(choices)])
		case k.Name == "backspace" && len(value) > 0:
			value = value[:len(value)-1]
		case k.Rune != 0 && len(choices) == 0:
			value = append(value, k.Rune)
		}
		problem = ""
	}
}
```

```go
package tui

import "fmt"

func Confirm(t Terminal, text, question string) (bool, error) {
	_, _ = fmt.Fprintf(t, "%s%s\n%s [y/n] ", clearScreen, text, question)
	for {
		k, err := t.ReadKey()
		if err != nil {
			return false, err
		}
		switch {
		case k.Rune == 'y' || k.Rune == 'Y':
			return true, nil
		case k.Rune == 'n' || k.Rune == 'N' || k.Name == "esc" || k.Name == "ctrl-c":
			return false, nil
		}
	}
}
```

`fmt.Fprintf(t, …)` verlangt, dass `Terminal` `io.Writer` erfüllt — das tut
es über `Write`.

- [ ] **Step 9: Raw-Hülle**

`raw.go`:

```go
package tui

import (
	"os"

	"golang.org/x/term"
)

// IsTerminal says whether f is a console the interactive forms can use.
func IsTerminal(f *os.File) bool { return term.IsTerminal(int(f.Fd())) }

type rawTerminal struct {
	in  *decoder
	out *os.File
}

func (r rawTerminal) ReadKey() (Key, error)       { return r.in.next() }
func (r rawTerminal) Write(p []byte) (int, error) { return r.out.Write(p) }

//coverage:exempt asks the real console for its size; a test process has none
func (r rawTerminal) Size() (int, int) {
	w, h, err := term.GetSize(int(r.out.Fd()))
	if err != nil {
		return 80, 24
	}
	return w, h
}

// Open puts the console into raw mode and returns a restore function that
// also clears the screen.
//
//coverage:exempt raw mode needs a real console; the widgets are tested through Scripted
func Open(in, out *os.File) (Terminal, func() error, error) {
	state, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return nil, nil, err
	}
	restoreVT := enableVT(out)
	restore := func() error {
		_, _ = out.WriteString("\x1b[H\x1b[2J")
		restoreVT()
		return term.Restore(int(in.Fd()), state)
	}
	return rawTerminal{in: newDecoder(in), out: out}, restore, nil
}
```

`raw_windows.go`:

```go
//go:build windows

package tui

import (
	"os"

	"golang.org/x/sys/windows"
)

// enableVT lets the console interpret the escape sequences the widgets
// write; conhost does not by default.
//
//coverage:exempt changes the mode of a real console handle
func enableVT(out *os.File) func() {
	handle := windows.Handle(out.Fd())
	var mode uint32
	if windows.GetConsoleMode(handle, &mode) != nil {
		return func() {}
	}
	_ = windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	return func() { _ = windows.SetConsoleMode(handle, mode) }
}
```

`raw_other.go`:

```go
//go:build !windows

package tui

import "os"

//coverage:exempt nothing to switch on outside Windows
func enableVT(*os.File) func() { return func() {} }
```

Die `//coverage:exempt`-Zeile steht nach AGENTS.md **direkt über `func`**;
der Doc-Kommentar kommt davor. `IsTerminal` und `ReadKey`/`Write` von
`rawTerminal` bekommen Tests: `IsTerminal` auf einer Datei aus `os.CreateTemp`
(false), `rawTerminal{in: newDecoder(strings.NewReader("a")), out: tmpfile}`
für die beiden anderen.

- [ ] **Step 10: Run tests and coverage**

Run: `go test ./internal/tui -count=1 -coverprofile=cover.out && go run ./cmd/loomux check gocover cover.out`
Expected: PASS, 100 % je Funktion außer den begründeten Ausnahmen.

- [ ] **Step 11: Von Hand prüfen (Messpunkt der Spec)**

In Windows Terminal, in conhost (`conhost.exe cmd`) und im Terminal der
Claude-App je einmal ein Testprogramm starten, das `Open` und `List` mit den
Zeilen aus `rows()` ruft (ein `go run` einer Datei unter dem Scratchpad, nicht
im Repo). Pfeiltasten, `/`, ESC allein, Enter prüfen. Ergebnis in die Akte
unter „Messungen vor dem Bau“, je Terminal eine Zeile.

- [ ] **Step 12: Commit**

```bash
git add go.mod go.sum internal/tui
git commit -m "feat(tui): add a full-screen list, input and confirmation on x/term"
```

---

### Task 14: `loomux config` interaktiv

**Files:**
- Modify: `internal/cli/config.go` (`configInteractive` wird `runConfigUI`)
- Create: `internal/cli/config_ui.go`
- Test: `internal/cli/config_ui_test.go`
- Modify: `internal/cli/imports_test.go` (die Ausnahmen aus Task 12 fallen)

**Interfaces:**
- Consumes: `tui.List/Input/Confirm/Open/IsTerminal/Script/Keys`,
  `proposeChange`, `configTarget` aus Task 11.
- Produces: `var openTerminal = func() (tui.Terminal, func() error, error)`
  als Naht; `func configUI(t tui.Terminal, target configTarget) error`.

- [ ] **Step 1: Write the failing test**

```go
func TestConfigUIChangesOneValue(t *testing.T) {
	root := configRoot(t, "[commit]\nlanguage = \"en\"\n")
	target, _ := resolveConfigTarget(root, false)
	entries, _ := target.entries("[commit]\nlanguage = \"en\"\n")
	index := -1
	for i, e := range entries {
		if e.Key.ID() == "commit.language" {
			index = i
		}
	}
	keys := []tui.Key{}
	for range index {
		keys = append(keys, tui.Key{Name: "down"})
	}
	keys = append(keys, tui.Keys("enter", "tab", "enter", "y", "q")...)
	term := tui.Script(100, 60, keys...)
	if err := configUI(term, target); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, ".loomux", "config.toml"))
	if string(data) != "[commit]\nlanguage = \"de\"\n" {
		t.Fatalf("file:\n%s", data)
	}
}

func TestConfigUIShowsARefusalAndWritesNothing(t *testing.T) {
	root := configRoot(t, "")
	target, _ := resolveConfigTarget(root, false)
	entries, _ := target.entries("")
	index := -1
	for i, e := range entries {
		if e.Key.ID() == "verify.timeout" {
			index = i
		}
	}
	keys := []tui.Key{}
	for range index {
		keys = append(keys, tui.Key{Name: "down"})
	}
	keys = append(keys, tui.Keys("enter", "backspace", "backspace", "backspace", "backspace", "backspace", "soon", "enter", "esc", "q")...)
	term := tui.Script(100, 60, keys...)
	if err := configUI(term, target); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(term.Output(), "not a duration") {
		t.Fatal("the refusal must be shown in place")
	}
	if _, err := os.Stat(filepath.Join(root, ".loomux", "config.toml")); err != nil {
		t.Fatal(err)
	}
}

func TestConfigUITablesAreShownNotEdited(t *testing.T) {
	root := configRoot(t, "")
	target, _ := resolveConfigTarget(root, false)
	entries, _ := target.entries("")
	index := -1
	for i, e := range entries {
		if e.Key.ID() == "verify.profiles" {
			index = i
		}
	}
	keys := []tui.Key{}
	for range index {
		keys = append(keys, tui.Key{Name: "down"})
	}
	keys = append(keys, tui.Keys("enter", "n", "q")...)
	term := tui.Script(100, 60, keys...)
	if err := configUI(term, target); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(term.Output(), "edit it by hand") {
		t.Fatal(term.Output())
	}
}

func TestConfigInteractiveOpensTheTerminal(t *testing.T) {
	root := configRoot(t, "")
	restore := openTerminal
	openTerminal = func() (tui.Terminal, func() error, error) {
		return tui.Script(80, 20, tui.Keys("q")...), func() error { return nil }, nil
	}
	t.Cleanup(func() { openTerminal = restore })
	if code, _, errOut := runConfig(t, "", "--root", root); code != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
	openTerminal = func() (tui.Terminal, func() error, error) { return nil, nil, errors.New("no console") }
	if code, _, errOut := runConfig(t, "", "--root", root); code != 2 || !strings.Contains(errOut, "config list") {
		t.Fatalf("%d %s", code, errOut)
	}
}
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/cli -run ConfigUI -count=1`
Expected: FAIL.

- [ ] **Step 3: Implement `config_ui.go`**

```go
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/config/edit"
	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/lock"
	"github.com/xidus90/loomux/internal/tui"
)

// openTerminal is the seam between the command and a real console.
var openTerminal = func() (tui.Terminal, func() error, error) {
	if !tui.IsTerminal(os.Stdin) || !tui.IsTerminal(os.Stdout) {
		return nil, nil, errors.New("not a terminal")
	}
	return tui.Open(os.Stdin, os.Stdout)
}

func runConfigUI(target configTarget, _ io.Reader, _, stderr io.Writer) int {
	term, restore, err := openTerminal()
	if err != nil {
		fmt.Fprintln(stderr, "loomux config: the interactive form needs a terminal; use `loomux config list`, `get` or `set`")
		return 2
	}
	err = configUI(term, target)
	_ = restore()
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	return 0
}

// configUI is the list loop: choose a row, change it, see the diff, confirm.
// Every change is written on its own, the way `config set` writes it.
func configUI(term tui.Terminal, target configTarget) error {
	for {
		text, err := target.read()
		if err != nil {
			return err
		}
		entries, err := target.entries(text)
		if err != nil {
			return err
		}
		rows := make([]tui.Row, len(entries))
		for i, e := range entries {
			value := e.Value
			if e.Key.Kind == schema.TableList {
				value = fmt.Sprintf("%d entries", e.Count)
			}
			rows[i] = tui.Row{Group: string(e.Key.Module), Label: e.Key.ID(), Value: firstLine(value), Note: string(e.Origin)}
		}
		chosen, err := tui.List(term, "loomux config — "+target.path, rows)
		if err != nil || chosen < 0 {
			return err
		}
		if err := changeOne(term, target, text, entries[chosen]); err != nil {
			return err
		}
	}
}

func changeOne(term tui.Terminal, target configTarget, text string, e schema.Entry) error {
	if e.Key.Kind == schema.Table || e.Key.Kind == schema.TableList {
		_, err := tui.Confirm(term, e.Key.Doc+"\n\n"+e.Value, e.Key.ID()+" is a table; edit it by hand. Back?")
		return err
	}
	initial := e.Input
	var next string
	_, ok, err := tui.Input(term, e.Key.ID()+" — "+e.Key.Doc, initial, e.Key.Choices, func(s string) error {
		var err error
		next, err = proposeChange(target, text, e.Key.ID(), s)
		return err
	})
	if err != nil || !ok {
		return err
	}
	diff := edit.Diff(text, next)
	if diff == "" {
		return nil
	}
	yes, err := tui.Confirm(term, diff, "write these changes?")
	if err != nil || !yes {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target.path), 0o755); err != nil {
		return err
	}
	return lock.ReplaceText(target.path, next)
}
```

In `config.go` die Naht auf `var configInteractive = runConfigUI` setzen und
die Übergangsfassung aus Task 11 löschen. `TestConfigWithoutATerminalNamesTheOtherForms`
aus Task 11 bekommt jetzt eine Naht, sonst setzte ein Mensch, der `go test`
in einer Konsole startet, diese Konsole in den Raw-Modus:

```go
	restore := openTerminal
	openTerminal = func() (tui.Terminal, func() error, error) { return nil, nil, errors.New("not a terminal") }
	t.Cleanup(func() { openTerminal = restore })
```

**Bool-Schlüssel** bekommen in der Oberfläche die Auswahl `true`/`false`:
in `changeOne` `choices := e.Key.Choices; if e.Key.Kind == schema.Bool {
choices = []string{"true", "false"} }` und `choices` an `tui.Input` geben.

**`[verify.<stack>.<art>]`-Tabellen** eines Projekts zeigt `config list`
nicht als eigene Schlüssel; die Selbstnutzung in Task 16 erwartet sie darum
nicht. Das steht in der Befehlsreferenz: Diese Tabellen ändert man von Hand,
`loomux check precommit --show` zeigt ihre Wirkung.

- [ ] **Step 4: Import test tightened**

In `TestHooksNeverImportTheConfigurationCommand` die beiden Ausnahmen im
zweiten `if` streichen:

```go
		if !cli[forbidden] {
			t.Errorf("the command line no longer reaches %s; the list is stale", forbidden)
		}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/cli ./internal/tui -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli
git commit -m "feat(cli): browse and change the configuration interactively"
```

---

### Task 15: Die Wächterregel

**Files:**
- Modify: `internal/hooks/guard.go` (`builtinCommands`, neue Funktion `configWriteSource`)
- Test: `internal/hooks/guard_test.go`

**Interfaces:**
- Produces: `func writesConfiguration(line string) bool`; in `checkTool` der
  Grund `"loomux init, config and area add write the configuration the guard reads; a human runs them"`.
  Eine Funktion statt einer Regex-Regel: „`init` ohne `--dry-run`“ braucht
  ein Lookahead, und RE2 kennt keins.

- [ ] **Step 1: Write the failing test**

```go
func TestTheGuardRefusesCommandsThatWriteTheConfiguration(t *testing.T) {
	refused := []string{
		"loomux init",
		"loomux init --yes",
		"bin/loomux.exe init --hooks=all",
		`"${LOCALAPPDATA}/loomux/bin/loomux.exe" config`,
		"loomux config set commit.language de --yes",
		"loomux config --global",
		"loomux config set model.enabled true --global",
		"loomux area add --path .",
		"go run ./cmd/loomux config set commit.language de",
		"cd x && loomux config",
		"LOOMUX_STATE_DIR=x loomux area add",
	}
	allowed := []string{
		"loomux init --dry-run",
		"loomux init --detect-only",
		"loomux config list",
		"loomux config get commit.language",
		"loomux config list --global",
		"loomux check precommit",
		"echo loomux config set",
		"grep 'loomux init' docs",
	}
	for _, line := range refused {
		if !writesConfiguration(line) {
			t.Errorf("must refuse %q", line)
		}
	}
	for _, line := range allowed {
		if writesConfiguration(line) {
			t.Errorf("must allow %q", line)
		}
	}
}

func TestCheckToolNamesTheConfigurationReason(t *testing.T) {
	got := checkTool(t.TempDir(), "Bash", map[string]any{"command": "loomux config set x y"}, config.Policy{})
	if !slices.ContainsFunc(got, func(r string) bool { return strings.Contains(r, "a human runs them") }) {
		t.Fatalf("reasons %v", got)
	}
	if got := checkTool(t.TempDir(), "Bash", map[string]any{"command": "loomux config list"}, config.Policy{}); len(got) != 0 {
		t.Fatalf("reasons %v", got)
	}
}
```

- [ ] **Step 2: Run to see it fail**

Run: `go test ./internal/hooks -run WriteTheConfiguration -count=1`
Expected: FAIL.

- [ ] **Step 3: Implement**

Die Regel ist eine Funktion und wird in `checkTool` als eigener Schritt
geprüft; `config.CommandRule` bekommt keinen neuen Typ:

```go
// writesConfiguration says whether a shell line runs a loomux command that
// writes .loomux/config.toml or the global config in-process, past every path
// rule. It reads words, not a file system, and has the manifest rule's
// limits: an alias or a variable holding the program passes.
func writesConfiguration(line string) bool {
	for _, segment := range segments(line) {
		words, err := shellwords.Split(segment)
		if err != nil || len(words) == 0 {
			continue
		}
		words = dropAssignments(words)
		args, ok := loomuxArgs(words)
		if !ok || len(args) == 0 {
			continue
		}
		switch args[0] {
		case "init":
			if !slices.Contains(args, "--dry-run") && !slices.Contains(args, "--detect-only") {
				return true
			}
		case "config":
			if len(args) == 1 || strings.HasPrefix(args[1], "-") || args[1] == "set" {
				return true
			}
		case "area":
			if len(args) > 1 && args[1] == "add" {
				return true
			}
		}
	}
	return false
}

// segments splits a line where a new command starts: ; | & && || and line breaks.
var segmentBreaks = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`&&|\|\||[;|&\n]`) })

func segments(line string) []string {
	return segmentBreaks().Split(line, -1)
}

func dropAssignments(words []string) []string {
	for len(words) > 0 && strings.Contains(words[0], "=") && !strings.HasPrefix(words[0], "-") {
		words = words[1:]
	}
	return words
}

// loomuxArgs returns the arguments after the program when the program is
// loomux: by name, by a path ending in loomux or loomux.exe, or go run of
// ./cmd/loomux.
func loomuxArgs(words []string) ([]string, bool) {
	base := strings.ToLower(filepath.Base(filepath.FromSlash(words[0])))
	if base == "loomux" || base == "loomux.exe" {
		return words[1:], true
	}
	if base == "go" && len(words) > 2 && words[1] == "run" && strings.TrimSuffix(filepath.ToSlash(words[2]), "/") == "./cmd/loomux" {
		return words[3:], true
	}
	return nil, false
}
```

In `checkTool`, im Block `if commandTools[tool]`, nach der Regelschleife:

```go
			if writesConfiguration(line) {
				reasons = append(reasons, "loomux init, config and area add write the configuration the guard reads; a human runs them")
			}
```

Die Signatur von `shellwords.Split` in `internal/shellwords` vorher lesen;
wenn sie anders heißt, den dortigen Namen nehmen. `"${LOCALAPPDATA}/…"`:
prüfen, ob `shellwords` Variablen stehen lässt — dann endet das Wort auf
`loomux.exe` und passt.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/hooks -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks
git commit -m "feat(guard): refuse agents the commands that write the configuration"
```

---

### Task 16: Dokumentation, Messen, Selbstnutzung, Abschluss

**Files:**
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md`
- Modify: `README.md`, `README.de.md`
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`
- Modify: `docs/.superpowers/parity/stufe-4a-1.md`
- Modify: `docs/en/migration.md`, `docs/de/migration.md`

- [ ] **Step 1: Befehlsreferenz**

Je Sprache ein Abschnitt `loomux config` mit den vier Formen, `--root`,
`--global`, `--yes`, `--json`, den Exit-Codes aus „Global Constraints“, der
Regel „Vorgaben werden nie geschrieben“, und dem Hinweis, dass der Wächter
den Befehl einem Agenten verweigert. Ein Abschnitt `[modules]` mit den drei
Schlüsseln und dem Satz, dass der Wächter immer prüft. `loomux mcp` bekommt
`--root`.

- [ ] **Step 2: READMEs**

Unter den Befehlen `loomux config` in einer Zeile; unter der Konfiguration
`[modules]`.

- [ ] **Step 3: Messen**

Binär bauen und die Pilot-Kopie tauschen, dann messen:

```bash
go build -o bin/loomux.new.exe ./cmd/loomux
```

```bash
go run ./cmd/loomux dev swap-binary --dir bin
```

```bash
go run ./cmd/loomux dev bench-hooks
```

`dev bench-hooks` misst `hook pre-tool-use` und `post-tool-use`; die Zahlen
neben den letzten Eintrag in `docs/*/benchmarks.md` stellen (Baseline 7,5 ms
warm für `pre-tool-use`). `pre-tool-use` liest `[modules]` nicht — der Wert
darf sich nicht bewegen; `post-tool-use` liest es und darf höchstens um das
Rauschen steigen. Dazu `loomux config list` kalt und 10× warm messen
(`Measure-Command` in PowerShell oder `hyperfine`, falls vorhanden) und mit
Datum, Uhrzeit, Befehl und Rohwerten eintragen, deutsch und englisch.

- [ ] **Step 4: Selbstnutzung**

```bash
bin/loomux.exe config list
```

Die Ausgabe zeigt die echte Konfiguration von loomux: `area.scope` =
`"project/loomux"` (set), `layout.wiki` (set), `index.include` (set),
`policy.paths.rules` und `policy.commands.rules` mit ihrer Anzahl,
`commit.language` (default). Die `[verify.go.*]`-Tabellen erscheinen nicht
als Schlüssel (siehe Task 14). Die Ausgabe in die Akte unter
„Selbstnutzung“.

Dann ein Mensch — nicht der Agent, der Wächter verweigert es — führt einmal
aus und bestätigt mit `n`:

```bash
bin/loomux.exe config set commit.threshold 3
```

Erwartet: Diff mit `+ threshold = 3` in einer neuen Sektion `[commit]`,
Frage, „declined; nothing written“. Ergebnis in die Akte.

Ein Agent prüft die Wächterregel, indem er `bin/loomux.exe config set
commit.threshold 3 --yes` über Bash versucht: Der Wächter verweigert mit dem
Grund aus Task 15. In die Akte.

- [ ] **Step 5: Mutationsrunde**

```bash
go run ./cmd/loomux dev mutants ./internal/config/edit ./internal/config/schema ./internal/tui ./internal/cli
```

Überlebende mit Begründung in die Akte unter „Mutationsrunde“; wo ein
Überlebender einen fehlenden Test zeigt, den Test ergänzen und die Runde für
das Paket wiederholen. Die Flags von `dev mutants` vorher mit
`go run ./cmd/loomux dev mutants --help` lesen.

- [ ] **Step 6: Migrationsplan**

In beiden `migration.md` die Zeile **4a-1** auf ✅ setzen, „Was sie gebracht
hat“ auf das Gebaute umschreiben, die Funktion „Konfiguration und
Einrichtung“ auf 🚧 (4a-1 gebaut, 4a-2 offen). 4a-2 und 4c neu lesen: beide
hängen an 4a-1 — ihre Spalte „Hängt ab von“ bekommt `4a-1 ✅`.

- [ ] **Step 7: Gate**

```bash
sh ci/gate.sh
```

Expected: grün.

- [ ] **Step 8: Commit**

```bash
git add docs README.md README.de.md
git commit -m "docs: document loomux config and [modules]"
```

---

## Selbstprüfung des Plans

- **Spec-Abdeckung (4a-1):** Schema → Task 6, 7; Prüfung über die Lader →
  Task 8; Zeileneditor samt Regel „Vorgaben nie schreiben“ → Task 9, 11
  (`set` auf die Vorgabe entfernt die Zeile); Herkunft → Task 10;
  `config list|get|set` und `--global` → Task 11; Oberfläche → Task 13;
  interaktives `config` → Task 14; `[modules]` zur Laufzeit → Task 2
  (Leser), 3 (Hooks, Wächter bleibt), 4 (Wiki-Lane), 5 (Brücke);
  `convert`/`fetch` verweigern kommt mit 4d; Messpunkt MCP-Arbeitsverzeichnis
  → Task 1 samt Rückfall in Task 5; schlanker Hook-Pfad → Task 12, 14;
  Wächterregel inklusive `area add` → Task 15; Selbstnutzung, Messen,
  Mutationsrunde → Task 16; VT-Messpunkt → Task 13, Step 11.
- **Abweichung von der Spec, die der Plan festhält:** Das Schema kennt
  `[maintenance]` (`on_merge`, `branch`), `[model]` (`enabled`, `roles`) und
  `[layout].inbox`, die die Spec in ihrer Tabelle nicht nennt, die
  `ReadDeclaration` aber heute schon liest. Der Test aus Task 7 verlangt sie;
  die Spec-Tabelle ist beim Planen nachgezogen.
- **Typen:** `config.Modules`/`ReadModules`/`ParseModules`/`AllModules`/
  `ModuleKeys` (Task 2) in Task 3, 4, 5, 7, 8; `schema.Key`/`Kind`/`Keys`/
  `Lookup`/`GlobalKeys` (Task 7) in 9, 10, 11, 14; `schema.Entry`/`Origin`/
  `Current` (Task 10) in 11, 14; `edit.Set`/`Remove`/`Render`/`Diff`/
  `AppendBlock` (Task 9) in 11, 14; `configTarget`/`proposeChange` (Task 11)
  in 14; `tui.*` (Task 13) in 14.
