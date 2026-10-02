# `init --brain=none` registriert das Projekt als Workspace — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux init` mit abgeschaltetem Brain-Modul trägt das Projekt als Bereich ohne `wiki` mit `workspace = true` in die Registry ein, damit der Schreibwächter den Projektbaum öffnet.

**Architecture:** Ein neuer Teil `workspace` im Modul Hooks (`internal/setup/parts.go`) mit denselben Vorgaben wie `area`; `setup.Build` plant die Handlung `workspace-add` nur, wenn das Brain-Modul aus ist, und nennt sonst den Grund als Notiz. Der Executor von `init` (`internal/cli/init.go`, `initRun.action`) schreibt den Eintrag direkt über `config.AddArea` unter der Registry-Sperre, nicht über `area add`, das immer ein Wiki setzt. Wächter, Index und `area add` bleiben unverändert.

**Tech Stack:** Go 1.26, Standardbibliothek; Tests mit `go test`, Mutationen per `go test -overlay`.

**Spec:** `docs/.superpowers/specs/2026-10-02-loomux-brain-none-workspace-design.md`

## Global Constraints

- Ein neuer Teil `workspace` im Modul **Hooks**, nicht Brain (Spec, Entscheidung 1).
- Er läuft nur, wenn das Brain-Modul aus ist; mit Brain an registriert `area` wie bisher (Entscheidung 2).
- Er schreibt nur die Registry: `scope`, `path`, `workspace = true`, **ohne `wiki`**, über `config.AddArea`. Nichts in `.loomux/config.toml`, kein Wiki-Gerüst, keine Routing-Regel, kein Merge-Hook (Entscheidung 3).
- Dieselben Bedingungen wie `area`: frisches Projekt (kein loomux-Checkout), kein `[area]` in `.loomux/config.toml`, kein Registry-Bereich an dieser Wurzel. Bereichsname `Choice.Scope`, sonst `project/<Verzeichnisname>` (Entscheidung 4).
- Keine neue Form von `area add`; der Wächter ändert sich nicht (Entscheidungen 5 und 6).
- Coverage 100 % je Funktion (`AGENTS.md`); Ausnahme nur mit `//coverage:exempt <reason>` direkt über `func`.
- Code, Bezeichner, Kommentare, Meldungen und Commit-Texte englisch; Planprosa und Spec-Nachtrag deutsch; die Doku je Sprache unter `docs/en` und `docs/de`.
- Commits nach Conventional Commits, ohne Arbeitspapier im Text, mit dem Nutzer als Autor und **ohne** `Co-Authored-By`-Zeile (globale `CLAUDE.md`). Nachrichten per Write in eine Datei und `git commit -F <datei> > <scratch>/commit.log 2>&1`, nie nach `/dev/null`.
- Kein Agent pusht. Kein Agent fährt `loomux init` gegen ein echtes Projekt (nur `--dry-run` in einer Testwelt), kein `config set|apply`, kein `area add` gegen ein echtes Projekt.
- `<scratch>` ist das Scratchpad-Verzeichnis der ausführenden Sitzung; Overlay-JSON nennt Pfade in Windows-Form `C:/…`, sonst ignoriert `go test -overlay` sie still.

## Review Focus

1. **Ein Projekt, das schon einen Eintrag ohne `wiki` hat** (die drei `iam_*`-Projekte, von Hand am 2026-10-02 eingetragen, unter anderem Bereichsnamen als `init` wählen würde): `Facts.Registered` ist wahr, der Teil ist aus, die Registry bleibt byte-gleich, kein `workspace-add`. Test: `TestInitWithoutTheBrainKeepsAnEntryAtItsRoot` (Task 2); erzwungen über eine frühere Antwort: `TestARegisteredRootGetsNoWorkspaceAdd` (Task 1).
2. **Ein Bereichsname, den schon ein anderer Pfad trägt** (`project/demo` an einem anderen Ordner): `Registered` ist falsch, `workspace-add` läuft, `config.AddArea` lehnt mit `scope "project/demo" is already registered` ab; `setup.Apply` bricht ab, `init` endet mit Exit 1, die Registry bleibt byte-gleich und `.loomux/state/installed.toml` ungeschrieben. `workspace-add` läuft wie `area-add` vor allen Dateien, sodass auch keine Host-Einträge, keine `.loomux/config.toml` und keine `.gitignore`-Änderung entstehen; sonst schärften die Host-Einträge eine Schranke, die keinen Baum öffnet. Test: `TestInitWithoutTheBrainStopsAtATakenScope` (Task 2).
3. **`--brain=none` in einem loomux-Checkout**: `fresh` ist falsch, der Teil ist aus, wie `area` auch. Test: die Erwartungstabelle in `TestACheckoutGetsOnlyWhatItHasCheckedIn` bekommt `"workspace": false` (Task 1).
4. **`--dry-run`**: der Plan nennt `workspace-add: register project/demo as a workspace without wiki in the registry`, die Registry entsteht nicht. Test: `TestInitDryRunWithoutTheBrainNamesTheWorkspace` (Task 1).
5. **`init`, von einem Agenten gestartet**: unverändert verweigert der Wächter `loomux init` ohne `--dry-run` für einen Agenten; der neue Teil ändert an der Befehlsregel nichts, und der Executor ruft keinen Unterbefehl, den der Wächter sehen könnte. Kein neuer Test; der Task-Reviewer prüft, dass `internal/brain/guard` und `internal/hooks` im Diff nicht vorkommen.

Zusätzlich, ohne eigenen Test in diesem Plan, aber vom Reviewer zu lesen:

- **Ein Workspace-Projekt, das später `[area]` in `.loomux/config.toml` bekommt**: Dann liest der Reindex die Erklärung, und weil der Eintrag kein `wiki` hat (`OwnWikiPrefix` gibt nil), wird jedes `**/*.md` unter der Projektwurzel eine Sammlung. Gemessen per Mutant in Task 2 (die Sammlung `project-ws` mit `path: …/ws`, `pattern: '**/*.md'`). Der Plan ändert das nicht; die Doku nennt es.
- **Die Umstellung (`internal/switchover`)** ruft `init --yes --brain=none` für Projekte ohne eigenes Wiki (`apply_test.go:723`). Ein frisches Ziel bekommt künftig einen Workspace-Eintrag; die drei schon eingetragenen Projekte treffen die Notiz `workspace: skipped; the registry has an area at this root already`. Bei Hooks an und Brain aus nennt `Build` sie an einer registrierten Wurzel, ob der Teil gewählt ist oder nicht (abgewählt ist er dort von Haus aus).

---

## Was der Code heute tut (gelesen am 2026-10-02 auf `acbfeb6f`)

- `setup.Parts` (`internal/setup/parts.go:28-52`) listet die Teile; `area` ist Brain, Vorgabe `fresh && !hasArea(f.Config) && !f.Registered` (Zeile 41).
- `setup.Build` (`internal/setup/plan.go:76`) bildet `on(id) = c.Parts[id] && c.moduleOn(modules[id])`; der `area`-Block (Zeilen 169-183) notiert `area: skipped; …` für `[area]` in der Konfiguration und für `f.Registered` und plant sonst `area-add`. `b.action(part, id, describe)` und `b.note(n)` stehen in Zeilen 267-271.
- `setup.Apply` (`internal/setup/apply.go:62`) zieht nur `binary-*` und `area-add` vor die Dateien; jede andere Handlung läuft in Planreihenfolge danach, ein Fehler dort bricht den Lauf ab und lässt `installed.toml` ungeschrieben.
- `initRun.action` (`internal/cli/init.go:476-501`) führt `area-add` als `r.sub("area", "add", …)` aus; `r.root` ist absolut (`filepath.Abs`, Zeile 200). `approved["area-add"]` (Zeile 264) betrifft nur den Merge-Hook und bleibt.
- `config.AddArea` (`internal/config/registrywrite.go:120`) liest und schreibt unter `underRegistryLock` und lehnt einen schon registrierten Bereichsnamen ab. `RenderRegistry` lässt `wiki` bei leerem `WikiPath` weg und schreibt `workspace = true` nach `path`.
- `writableRoots` (`internal/brain/guard/guard.go:150-169`) öffnet `path` eines Bereichs mit `workspace`, auch ohne `wiki`.
- Die Interview-Frage nach dem Bereichsnamen (`internal/cli/init_ui.go:77`) kommt nur bei `area` mit Brain an. **Entscheidung dieses Plans:** Sie bleibt so; `workspace` nimmt `Choice.Scope` ohne Rückfrage (die Spec verlangt denselben Namen, keine Frage). Eine neue Frage verschöbe die Tastenfolgen von `TestInitAsksPerModule`, das Brain abschaltet und danach `y` schickt.
- `init --help` druckt nur die Flags (`initUsage`, `internal/cli/init.go:33`), keine Teile-Liste. Die Teile-Liste der Spec ist die Interview-Zeile `module hooks (host-entries, git-hooks, verify-skill, workspace)`, die `setup.Parts` speist, und die Tabelle in `cli-reference.md`.
- **Aufgezeichnete Fälle:** Kein Fall unter `testdata/cases` fährt `loomux init` (die `3a`-Fälle mit `init` sind ultra-brains `brain-mcp init`, übersetzt auf `area add`), keiner `--brain=none`. Es ändert sich keine Aufzeichnung. `internal/switchover/apply_test.go:723` prüft nur die Befehlszeile an einer Naht und bleibt grün.

## Dateien

- Modify: `internal/setup/parts.go` — der Teil `workspace` und die ID-Liste im Kommentar von `Part`.
- Modify: `internal/setup/plan.go` — der `workspace`-Block in `Build`, die ID-Liste im Kommentar von `Action`.
- Modify: `internal/setup/apply.go` — nur der Reihenfolge-Kommentar von `Apply`.
- Modify: `internal/setup/parts_test.go` — Erwartungstabelle des Checkouts, zwei neue Tests.
- Modify: `internal/cli/init.go` — `case "workspace-add"` in `initRun.action`.
- Create: `internal/cli/init_workspace_test.go` — Testwelt-Tests für `init`.
- Create: `internal/brain/index/workspace_test.go` — Reindex legt keine Sammlung an.
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md`, `docs/en/configuration.md`, `docs/de/configuration.md`, `README.md`, `README.de.md`.
- Modify: `docs/.superpowers/specs/2026-09-29-loomux-stufe-4e-umstellung-vorbereitet-design.md` — Nachtrag.
- Modify: `docs/.superpowers/specs/2026-10-02-loomux-brain-none-workspace-design.md` — Korrektur des Abschnitts zum Index.

---

### Task 1: Der Teil `workspace` und seine Planung

**Files:**
- Modify: `internal/setup/parts.go:14-16, 40-41`
- Modify: `internal/setup/plan.go:41-43, 165`
- Modify: `internal/setup/apply.go:49`
- Modify: `internal/setup/parts_test.go:18-20` und am Dateiende
- Create: `internal/cli/init_workspace_test.go`

**Interfaces:**
- Consumes: `setup.Facts{Config, Registered, Checkout}`, `hasArea(string) bool`, `Choice.moduleOn(schema.Module) bool`, `builder.action(part, id, describe string)`, `builder.note(string)`.
- Produces: Teil-ID `"workspace"` (Modul `schema.Hooks`); Handlung `setup.Action{Part: "workspace", ID: "workspace-add", Describe: "register <scope> as a workspace without wiki in the registry"}`; Notizen `workspace: skipped; .loomux/config.toml declares [area] already` und `workspace: skipped; the registry has an area at this root already`. Test-Hilfe `registryText(t *testing.T) string` in Paket `cli`.

- [ ] **Step 1: Die Erwartungstabelle des Checkouts erweitern**

In `internal/setup/parts_test.go`, `TestACheckoutGetsOnlyWhatItHasCheckedIn`, die Zeilen

```go
		"tools": true, "host-entries": true, "git-hooks": true, "verify-skill": false,
		"area": false, "merge-hook": false, "brain-skills": false, "model": false, "graph-build": false,
```

ersetzen durch

```go
		"tools": true, "host-entries": true, "git-hooks": true, "verify-skill": false,
		"workspace": false, "area": false, "merge-hook": false, "brain-skills": false, "model": false, "graph-build": false,
```

- [ ] **Step 2: Die Planungstests anfügen**

Ans Ende von `internal/setup/parts_test.go` (die Importe `slices`, `testing` und `schema` sind schon da):

```go

func TestTheWorkspaceRegistersTheProjectOnlyWithoutTheBrain(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	if !c.Parts["workspace"] {
		t.Fatal("workspace is not chosen for a fresh project")
	}
	for _, tc := range []struct {
		name         string
		brain, hooks bool
		want         []string // the registering actions planned
	}{
		{"brain on", true, true, []string{"area-add"}},
		{"brain off", false, true, []string{"workspace-add"}},
		{"brain and hooks off", false, false, nil},
	} {
		c.Modules[schema.Brain], c.Modules[schema.Hooks] = tc.brain, tc.hooks
		p, err := Build(f, c, reader(root))
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, id := range actions(p) {
			if id == "area-add" || id == "workspace-add" {
				got = append(got, id)
			}
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: registering actions %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestARegisteredRootGetsNoWorkspaceAdd(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod})
	f := gather(t, root, "")
	f.Registered = true
	c := DefaultChoice(f, Answers{})
	if c.Parts["workspace"] {
		t.Errorf("workspace is chosen for a registered root")
	}
	// Chosen anyway, through an earlier answer: Build still holds back.
	c.Parts["workspace"] = true
	c.Modules[schema.Brain] = false
	p, _ := Build(f, c, reader(root))
	if slices.Contains(actions(p), "workspace-add") || !hasNote(p, "workspace: skipped; the registry has an area") {
		t.Errorf("actions = %v, notes = %v", actions(p), p.Notes)
	}
	f.Registered = false
	f.Config = "[area]\nscope = \"project/demo\"\n"
	if DefaultChoice(f, Answers{}).Parts["workspace"] {
		t.Errorf("workspace is chosen over a declared area")
	}
	p, _ = Build(f, c, reader(root))
	if slices.Contains(actions(p), "workspace-add") || !hasNote(p, "workspace: skipped; .loomux/config.toml declares [area]") {
		t.Errorf("actions = %v, notes = %v", actions(p), p.Notes)
	}
}
```

- [ ] **Step 3: Den Dry-Run-Test anlegen**

Create `internal/cli/init_workspace_test.go`:

```go
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// registryText is the machine registry of the test world, "" when there is
// none.
func registryText(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(os.Getenv("LOOMUX_STATE_DIR"), "registry.toml"))
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestInitDryRunWithoutTheBrainNamesTheWorkspace(t *testing.T) {
	root, _ := initWorld(t)
	code, out, errOut := run("init", "--root", root, "--dry-run", "--brain=none")
	if code != 0 {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	if !strings.Contains(out, "  workspace-add: register project/demo as a workspace without wiki in the registry\n") {
		t.Errorf("stdout:\n%s", out)
	}
	if registryText(t) != "" {
		t.Errorf("dry run wrote the registry:\n%s", registryText(t))
	}
}
```

`initWorld` (`internal/cli/init_test.go:63`) setzt `LOCALAPPDATA` und `LOOMUX_STATE_DIR` in ein Wegwerfverzeichnis; die Registry der Testwelt liegt dort. `XDG_CONFIG_HOME` braucht dieser Task nicht, weil `init` keinen Reindex fährt (die Testwelt hängt `--no-reindex` an `area add`, und `workspace-add` ruft keinen).

- [ ] **Step 4: Rot laufen lassen**

Run: `go test ./internal/setup/ -run 'Workspace|Checkout' -count=1 > <scratch>/t1-red.log 2>&1; go test ./internal/cli/ -run InitDryRunWithoutTheBrain -count=1 >> <scratch>/t1-red.log 2>&1; grep -E -- '--- FAIL|_test.go' <scratch>/t1-red.log`

Expected (gemessen):

```
--- FAIL: TestACheckoutGetsOnlyWhatItHasCheckedIn
    parts_test.go:23: 14 parts, want 15
--- FAIL: TestTheWorkspaceRegistersTheProjectOnlyWithoutTheBrain
    parts_test.go:…: workspace is not chosen for a fresh project
--- FAIL: TestARegisteredRootGetsNoWorkspaceAdd
    parts_test.go:…: actions = [binary-install graph-build], notes = []
--- FAIL: TestInitDryRunWithoutTheBrainNamesTheWorkspace
    init_workspace_test.go:…: stdout:
```

- [ ] **Step 5: Den Teil anlegen**

In `internal/setup/parts.go` den Kommentar von `Part.ID`

```go
	// ID is one of "binary", "config", "gitignore", "agents-md", "mcp-json",
	// "tools", "host-entries", "git-hooks", "verify-skill", "area",
	// "merge-hook", "brain-skills", "model", "graph-build".
```

ersetzen durch

```go
	// ID is one of "binary", "config", "gitignore", "agents-md", "mcp-json",
	// "tools", "host-entries", "git-hooks", "verify-skill", "workspace",
	// "area", "merge-hook", "brain-skills", "model", "graph-build".
```

und in `Parts` direkt nach der Zeile `{schema.Hooks, "verify-skill", "the skill verify-until-green", fresh},` einfügen:

```go
		// The write barrier opens only trees the registry names; without
		// the brain module no area is registered, so this part registers
		// the project as a workspace without a wiki instead.
		{schema.Hooks, "workspace", "register the project as a workspace without wiki, when the brain module is off",
			fresh && !hasArea(f.Config) && !f.Registered},
```

- [ ] **Step 6: Die Handlung planen**

In `internal/setup/plan.go` den Kommentar von `Action`

```go
// Action is one step init runs rather than writes; ID is one of
// "binary-install", "binary-build", "hooks-path", "area-add", "merge-hook",
// "model-pull", "graph-build".
```

ersetzen durch

```go
// Action is one step init runs rather than writes; ID is one of
// "binary-install", "binary-build", "hooks-path", "workspace-add", "area-add",
// "merge-hook", "model-pull", "graph-build".
```

und in `Build` direkt vor `// declares says whether area add will write the declaration, and with` einfügen:

```go
	// With the brain module on, area registers the project and brings the
	// wiki; workspace would register it a second time.
	if on("workspace") && !c.moduleOn(schema.Brain) {
		switch {
		case hasArea(f.Config):
			b.note("workspace: skipped; .loomux/config.toml declares [area] already")
		case f.Registered:
			b.note("workspace: skipped; the registry has an area at this root already")
		default:
			b.action("workspace", "workspace-add", "register "+c.Scope+" as a workspace without wiki in the registry")
		}
	}
```

In `internal/setup/apply.go` im Kommentar von `Apply` die Zeile

```go
// binary, area-add, files, hooks-path, merge-hook, model-pull, graph-build,
```

ersetzen durch

```go
// binary, area-add, files, hooks-path, workspace-add, merge-hook, model-pull, graph-build,
```

`Apply` selbst bleibt: `workspace-add` fällt in den `default`-Zweig und läuft nach den Dateien in Planreihenfolge.

- [ ] **Step 7: Grün laufen lassen**

Run: `go test ./internal/setup/ -count=1 > <scratch>/t1-green.log 2>&1; go test ./internal/cli/ -run 'InitDryRunWithoutTheBrain|InitAsksPerModule|InitNamesAModuleSwitchedOff' -count=1 >> <scratch>/t1-green.log 2>&1; tail -3 <scratch>/t1-green.log`
Expected: `ok  github.com/xidus90/loomux/internal/setup` und `ok  github.com/xidus90/loomux/internal/cli`.

- [ ] **Step 8: Mutationsrunde**

Write `<scratch>/mutants.py` (Task 2 benutzt dieselbe Datei):

```python
import json, pathlib, subprocess, sys

W = pathlib.Path(sys.argv[1]).resolve()
WANT = set(sys.argv[2:])
OUT = pathlib.Path(__file__).resolve().parent / "mutants"
OUT.mkdir(exist_ok=True)
MUTANTS = [
    ("m1", "internal/setup/plan.go", 'on("workspace") && !c.moduleOn(schema.Brain)', 'on("workspace") && true', "./internal/setup/", "Workspace"),
    ("m2", "internal/setup/plan.go", 'case f.Registered:\n\t\t\tb.note("workspace:', 'case false:\n\t\t\tb.note("workspace:', "./internal/setup/", "Workspace"),
    ("m3", "internal/setup/plan.go", 'case hasArea(f.Config):\n\t\t\tb.note("workspace:', 'case false:\n\t\t\tb.note("workspace:', "./internal/setup/", "Workspace"),
    ("m4", "internal/setup/parts.go", 'module is off",\n\t\t\tfresh && !hasArea(f.Config) && !f.Registered', 'module is off",\n\t\t\tfresh && !hasArea(f.Config)', "./internal/setup/", "Workspace"),
    ("m5", "internal/setup/parts.go", 'module is off",\n\t\t\tfresh && !hasArea(f.Config) && !f.Registered', 'module is off",\n\t\t\tfresh && !f.Registered', "./internal/setup/", "Workspace"),
    ("m6", "internal/cli/init.go", "Workspace: true})", "Workspace: false})", "./internal/cli/", "InitWithoutTheBrain"),
    ("m7", "internal/cli/init.go", "Path: filepath.ToSlash(r.root), Workspace", "Path: r.root, Workspace", "./internal/cli/", "InitWithoutTheBrain"),
    ("m8", "internal/brain/index/reindex.go", "manifest, err := config.ReadAreaManifestUntilStage4(source)\n\tif err != nil {", "manifest, err := config.ReadAreaManifestUntilStage4(source)\n\tif err != nil {\n\t\tmanifest, err = &config.Manifest{}, nil\n\t}\n\tif err != nil {", "./internal/brain/index/", "WorkspaceWithoutDeclaration"),
]
for name, rel, old, new, pkg, run in MUTANTS:
    if name not in WANT:
        continue
    src = (W / rel).read_text(encoding="utf-8")
    assert src.count(old) == 1, (name, old)
    mut = OUT / (name + ".go")
    mut.write_text(src.replace(old, new), encoding="utf-8", newline="\n")
    ov = OUT / (name + ".json")
    ov.write_text(json.dumps({"Replace": {(W / rel).as_posix(): mut.as_posix()}}), encoding="utf-8")
    r = subprocess.run(["go", "test", "-overlay", ov.as_posix(), "-count=1", "-run", run, pkg],
                       cwd=W, capture_output=True, text=True)
    out = r.stdout + r.stderr
    if "build failed" in out or "[setup failed]" in out:
        verdict = "BADMUTANT"
    elif r.returncode != 0 and "--- FAIL" in out:
        verdict = "killed"
    else:
        verdict = "SURVIVED"
    print(name, verdict, [l.strip() for l in out.splitlines() if l.startswith("--- FAIL")])
```

`W.resolve()` liefert unter Windows `C:\…`, `as_posix()` daraus `C:/…`: die Form, die `-overlay` annimmt. `python -B`, damit kein `__pycache__` einen alten Stand trägt.

Run: `python -B <scratch>/mutants.py "$(pwd)" m1 m2 m3 m4 m5 > <scratch>/mut1.log 2>&1; cat <scratch>/mut1.log`

Expected (gemessen):

```
m1 killed ['--- FAIL: TestTheWorkspaceRegistersTheProjectOnlyWithoutTheBrain (…)']
m2 killed ['--- FAIL: TestARegisteredRootGetsNoWorkspaceAdd (…)']
m3 killed ['--- FAIL: TestARegisteredRootGetsNoWorkspaceAdd (…)']
m4 killed ['--- FAIL: TestARegisteredRootGetsNoWorkspaceAdd (…)']
m5 killed ['--- FAIL: TestARegisteredRootGetsNoWorkspaceAdd (…)']
```

Jede Zeile `SURVIVED` oder `BADMUTANT` ist ein Befund: Testeingabe schärfen, nicht den Mutanten.

- [ ] **Step 9: Commit**

Write `<scratch>/msg1.txt`:

```
feat(setup): plan a workspace entry when the brain module is off

Without the brain module init registered no area, and the write barrier
opens only trees the registry names, so every write in such a project was
refused. The new hooks part workspace plans workspace-add under the same
conditions as area: a fresh project without [area] and without a registry
entry at its root.
```

Run: `git add internal/setup/parts.go internal/setup/plan.go internal/setup/apply.go internal/setup/parts_test.go internal/cli/init_workspace_test.go && git commit -F <scratch>/msg1.txt > <scratch>/commit1.log 2>&1; tail -5 <scratch>/commit1.log`
Expected: das pre-commit-Tor grün, ein Commit. Danach `git log -1 --format='%an <%ae>'` zeigt den Nutzer.

---

### Task 2: `init` schreibt den Workspace-Eintrag

**Files:**
- Modify: `internal/cli/init.go:490-491`
- Modify: `internal/cli/init_workspace_test.go`
- Create: `internal/brain/index/workspace_test.go`

**Interfaces:**
- Consumes: Handlung `workspace-add` aus Task 1; `config.AddArea(stateDir string, area config.Area) error`; `config.StateDir() string`; `config.Area{Scope, Path string; Workspace bool}`; aus Task 1 `registryText(t)`; aus `internal/cli/init_test.go` `initWorld`, `registerArea`, `writeAt`, `there`, `run`; aus `internal/cli/hook_test.go` `runWith`; `hooks.ExitDenied` (`internal/hooks/guard.go:26`, Wert 2).
- Produces: der Registry-Eintrag `[[area]]\nscope = "<scope>"\npath = "<root mit />"\nworkspace = true\n`.

- [ ] **Step 1: Die Testwelt-Tests anfügen**

In `internal/cli/init_workspace_test.go` den Importblock ersetzen durch

```go
import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/hooks"
)
```

und nach `registryText` einfügen:

```go
// guardWrite sends a Write of notes.md under root through the write barrier
// and answers its exit code.
func guardWrite(t *testing.T, root string) int {
	t.Helper()
	target := filepath.ToSlash(filepath.Join(root, "notes.md"))
	payload := `{"tool_name":"Write","tool_input":{"file_path":"` + target + `","content":"x"}}`
	code, _, _ := runWith(payload, "hook", "pre-tool-use", "--host", "claude", "--root", root)
	return code
}

func TestInitWithoutTheBrainRegistersAWorkspace(t *testing.T) {
	root, s := initWorld(t)
	if code := guardWrite(t, root); code != hooks.ExitDenied {
		t.Fatalf("before init: the barrier answered %d, want %d", code, hooks.ExitDenied)
	}
	code, out, errOut := run("init", "--root", root, "--yes", "--brain=none", "--graph=none")
	if code != 0 {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	want := "[[area]]\nscope = \"project/demo\"\npath = \"" + filepath.ToSlash(root) + "\"\nworkspace = true\n"
	if got := registryText(t); got != want {
		t.Errorf("registry:\n%s\nwant:\n%s", got, want)
	}
	if len(s.actions) != 0 {
		t.Errorf("subcommands run: %v", s.actions)
	}
	if !strings.Contains(out, "written: workspace-add") {
		t.Errorf("report:\n%s", out)
	}
	if code := guardWrite(t, root); code != 0 {
		t.Errorf("after init: the barrier answered %d, want 0", code)
	}
	registry := registryText(t)
	code, out, errOut = run("init", "--root", root, "--yes", "--brain=none", "--graph=none")
	if code != 0 || !strings.HasPrefix(out, "nothing to change\n") {
		t.Errorf("second run: code %d: %s\n%s", code, errOut, out)
	}
	if registryText(t) != registry {
		t.Errorf("second run changed the registry:\n%s", registryText(t))
	}
}

func TestInitWithoutTheBrainKeepsAnEntryAtItsRoot(t *testing.T) {
	root, _ := initWorld(t)
	// The hand-made entry of a project set up before init could: no wiki,
	// workspace = true, a scope init would not pick.
	text := "[[area]]\nscope = \"project/own\"\npath = \"" + filepath.ToSlash(root) + "\"\nworkspace = true\n"
	writeAt(t, filepath.Join(os.Getenv("LOOMUX_STATE_DIR"), "registry.toml"), text)
	code, out, errOut := run("init", "--root", root, "--yes", "--brain=none", "--graph=none")
	if code != 0 {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	if registryText(t) != text || strings.Contains(out, "workspace-add") {
		t.Errorf("registry:\n%s\nstdout:\n%s", registryText(t), out)
	}
}

func TestInitWithoutTheBrainStopsAtATakenScope(t *testing.T) {
	root, _ := initWorld(t)
	other := t.TempDir()
	registerArea(t, [2]string{"project/demo", other})
	registry := registryText(t)
	code, out, errOut := run("init", "--root", root, "--yes", "--brain=none", "--graph=none")
	if code != 1 || !strings.Contains(errOut, `scope "project/demo" is already registered`) {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	if registryText(t) != registry || there(root, ".loomux/state/installed.toml") {
		t.Errorf("registry:\n%s", registryText(t))
	}
}
```

Der Wächter läuft in derselben Testwelt: `initWorld` hat `LOOMUX_STATE_DIR` gesetzt, `hook pre-tool-use` liest die Registry dort (so auch `TestHookPreToolUseIsReachedWithoutARoot`, `internal/cli/hook_test.go:444`). `notes.md` ist ein neutraler Name: kein Geheimnis, nichts unter `.loomux/`.

- [ ] **Step 2: Den Reindex-Test anlegen**

Create `internal/brain/index/workspace_test.go`:

```go
package index

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
)

// The entry `init --brain=none` writes: a workspace without wiki whose
// project declares no [area]. Reindex skips it for the missing declaration
// and gives it no collection, while an area beside it is still indexed.
func TestReindexGivesAWorkspaceWithoutDeclarationNoCollection(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	t.Setenv("XDG_CONFIG_HOME", tmp)

	workspace := filepath.Join(tmp, "ws")
	if err := os.MkdirAll(filepath.Join(workspace, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".loomux", "config.toml"), []byte("[modules]\nbrain = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "notes.md"), []byte("# Notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	validAreaDir := filepath.Join(tmp, "valid_area")
	setupTestArea(t, validAreaDir, "[area]\nscope = \"valid\"\n")
	if err := os.WriteFile(filepath.Join(validAreaDir, "valid.md"), []byte("# Valid\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	registryContent := "[[area]]\nscope = \"project/ws\"\npath = \"" + filepath.ToSlash(workspace) + "\"\nworkspace = true\n\n" +
		"[[area]]\nscope = \"valid\"\npath = \"" + filepath.ToSlash(validAreaDir) + "\"\n"
	regPath := writeTestRegistry(t, stateDir, registryContent)

	port := search.NewFakePort()
	var stderr bytes.Buffer
	code, err := ReindexWithOutput(regPath, stateDir, "", port, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code %d, err %v: %s", code, err, stderr.String())
	}
	out := stderr.String()
	if !strings.Contains(out, "skipping project/ws: ") || !strings.Contains(out, "updated qmd collections: valid\n") {
		t.Errorf("stderr:\n%s", out)
	}
	qmd, err := os.ReadFile(QmdConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if name := search.CollectionName("project/ws"); strings.Contains(string(qmd), name) {
		t.Errorf("a collection %s for the workspace:\n%s", name, qmd)
	}
	if len(port.Refreshed) != 1 || len(port.Refreshed[0]) != 1 || port.Refreshed[0][0] != "valid" {
		t.Errorf("refreshed %#v", port.Refreshed)
	}
}
```

qmd ist hier gestubbt: `search.NewFakePort()` statt eines echten Diensts, `XDG_CONFIG_HOME` auf das Testverzeichnis, damit `QmdConfigPath()` nicht die echte `~/.config/qmd/index.yml` trifft (wie `TestReindexSkippingMissingAreaAndManifest`, `reindex_test.go:220`). Dieser Test hält heutiges Verhalten fest und ist **vom ersten Lauf an grün**: Er ändert keinen Produktionscode, seine Rot-Probe ist der Mutant m8 in Step 6.

- [ ] **Step 3: Rot laufen lassen**

Run: `go test ./internal/cli/ -run InitWithoutTheBrain -count=1 > <scratch>/t2-red.log 2>&1; go test ./internal/brain/index/ -run WorkspaceWithoutDeclaration -count=1 >> <scratch>/t2-red.log 2>&1; grep -E -- '--- (FAIL|PASS)|_test.go|^ok' <scratch>/t2-red.log`

Expected (gemessen auf dem Stand nach Task 1):

```
--- FAIL: TestInitWithoutTheBrainRegistersAWorkspace
    init_workspace_test.go:…: code 1: loomux init: workspace-add: no action workspace-add
--- FAIL: TestInitWithoutTheBrainStopsAtATakenScope
    init_workspace_test.go:…: code 1: loomux init: workspace-add: no action workspace-add
ok  github.com/xidus90/loomux/internal/brain/index
```

`TestInitWithoutTheBrainKeepsAnEntryAtItsRoot` ist schon grün: Ein registrierter Wurzelpfad schaltet den Teil seit Task 1 ab, `workspace-add` wird gar nicht geplant. Er hält fest, dass der Executor daran nichts ändert; seine Regel töten m2 und m4 in Task 1.

- [ ] **Step 4: Den Executor schreiben**

In `internal/cli/init.go`, `initRun.action`, direkt nach

```go
	case "area-add":
		return r.sub("area", "add", "--path", r.root, "--scope", r.scope, "--yes")
```

einfügen:

```go
	case "workspace-add":
		// Not through area add, which always gives the area a wiki.
		return config.AddArea(config.StateDir(), config.Area{Scope: r.scope, Path: filepath.ToSlash(r.root), Workspace: true})
```

`config` und `path/filepath` sind in `init.go` schon importiert. `filepath.ToSlash`, weil `area add` den Pfad ebenso schreibt (`planArea`, `internal/cli/area.go:207`).

- [ ] **Step 5: Grün laufen lassen**

Run: `go vet ./... > <scratch>/vet.log 2>&1; echo vet=$?; go test ./internal/setup/ ./internal/cli/ ./internal/brain/index/ ./internal/switchover/ -count=1 -coverprofile=<scratch>/cover.out > <scratch>/t2-green.log 2>&1; echo test=$?; tail -4 <scratch>/t2-green.log; go tool cover -func=<scratch>/cover.out | grep -E 'parts.go:.*Parts|plan.go:.*Build|init.go:.*action'`

Expected: `vet=0`, `test=0`, vier Pakete `ok`, und

```
…/internal/cli/init.go:…:	action	100.0%
…/internal/setup/parts.go:…:	Parts	100.0%
…/internal/setup/plan.go:…:	Build	100.0%
```

`internal/cli` braucht rund 80 s, `internal/switchover` rund 70 s.

- [ ] **Step 6: Mutationsrunde**

Run: `python -B <scratch>/mutants.py "$(pwd)" m6 m7 m8 > <scratch>/mut2.log 2>&1; cat <scratch>/mut2.log`

Expected (gemessen):

```
m6 killed ['--- FAIL: TestInitWithoutTheBrainRegistersAWorkspace (…)']
m7 killed ['--- FAIL: TestInitWithoutTheBrainRegistersAWorkspace (…)']
m8 killed ['--- FAIL: TestReindexGivesAWorkspaceWithoutDeclarationNoCollection (…)']
```

m8 lässt den Reindex eine fehlende Erklärung wie eine leere lesen; der Test sieht dann die Sammlung `project-ws` mit `path: …/ws` und `pattern: '**/*.md'` (das ist der Fall aus der Review Focus: ein Workspace, der später `[area]` bekommt).

- [ ] **Step 7: Commit**

Write `<scratch>/msg2.txt`:

```
feat(cli): register a workspace without wiki when init runs without the brain

init --brain=none now writes the registry entry the write barrier needs:
the project's scope and path with workspace = true and no wiki, through
config.AddArea under the registry lock. area add is not used, because it
always gives the area a wiki. A scope that another path holds stops the
run with AddArea's refusal.
```

Run: `git add internal/cli/init.go internal/cli/init_workspace_test.go internal/brain/index/workspace_test.go && git commit -F <scratch>/msg2.txt > <scratch>/commit2.log 2>&1; tail -5 <scratch>/commit2.log`
Expected: Tor grün, ein Commit.

---

### Task 3: Doku, README und Spec-Nachträge

**Files:**
- Modify: `docs/en/cli-reference.md:1125, 1430-1432, 1461`
- Modify: `docs/de/cli-reference.md:1156, 1485-1487, 1517`
- Modify: `docs/en/configuration.md:991-992`
- Modify: `docs/de/configuration.md:1022-1023`
- Modify: `README.md:244`, `README.de.md:245`
- Modify: `docs/.superpowers/specs/2026-09-29-loomux-stufe-4e-umstellung-vorbereitet-design.md` (Ende)
- Modify: `docs/.superpowers/specs/2026-10-02-loomux-brain-none-workspace-design.md` (Abschnitt „Was ein Bereich ohne Wiki sonst bewirkt“)

**Interfaces:**
- Consumes: Teil `workspace`, Handlung `workspace-add` und ihr Beschreibungstext aus Task 1 und 2.
- Produces: nichts für Code.

- [ ] **Step 1: `cli-reference.md` englisch**

Zeile 1125, im Satz zu `init_args`, `(`--brain=none` for a project whose wiki lives in another area)` ersetzen durch:

```
(`--brain=none` for a project whose wiki lives in another area; `init` then registers it as a workspace without wiki)
```

Den Punkt `--hooks`, `--brain`, `--graph` (Zeilen 1430-1432) ersetzen durch:

```
- **`--hooks`, `--brain`, `--graph`**: `all` turns on every part of the
  module, `none` turns the module off, `each` asks part by part. A flag beats
  the answers of an earlier run. `--brain=none` still registers the project,
  as a workspace without wiki (the part `workspace`), so that the write
  barrier opens its tree.
```

Nach der Tabellenzeile `| hooks | `verify-skill` | … |` (Zeile 1461) einfügen:

```
| hooks | `workspace` | only while the brain module is off: `workspace-add` writes a registry entry with the scope, `path` and `workspace = true` and no `wiki`, under the registry lock as `area add` does; nothing in `.loomux/config.toml`, no wiki, no routing rule, no merge hook. Without an entry the write barrier refuses every write in the project. A scope another path already holds stops the run with exit 1 | on, off in a checkout or an area already declared or registered |
```

- [ ] **Step 2: `cli-reference.md` deutsch**

Zeile 1156: `(`--brain=none` für ein Projekt, dessen Wiki in einem anderen Bereich liegt)` ersetzen durch:

```
(`--brain=none` für ein Projekt, dessen Wiki in einem anderen Bereich liegt; `init` registriert es dann als Workspace ohne Wiki)
```

Den Punkt `--hooks`, `--brain`, `--graph` (Zeilen 1485-1487) ersetzen durch:

```
- **`--hooks`, `--brain`, `--graph`**: `all` schaltet jeden Teil des Moduls
  an, `none` das Modul aus, `each` fragt Teil für Teil. Ein Flag schlägt die
  Antworten eines früheren Laufs. `--brain=none` registriert das Projekt
  trotzdem, als Workspace ohne Wiki (der Teil `workspace`), damit die
  Schreibschranke seinen Baum öffnet.
```

Nach der Tabellenzeile `| hooks | `verify-skill` | … |` (Zeile 1517) einfügen:

```
| hooks | `workspace` | nur, solange das Brain-Modul aus ist: `workspace-add` schreibt einen Registry-Eintrag mit dem Bereichsnamen, `path` und `workspace = true` und ohne `wiki`, unter der Registry-Sperre wie `area add`; nichts in `.loomux/config.toml`, kein Wiki, keine Routing-Regel, kein Merge-Hook. Ohne Eintrag verweigert die Schreibschranke jeden Write im Projekt. Ein Bereichsname, den schon ein anderer Pfad trägt, beendet den Lauf mit Exit 1 | an, aus in einem Checkout oder bei einem schon erklärten oder registrierten Bereich |
```

- [ ] **Step 3: `configuration.md` englisch und deutsch**

`docs/en/configuration.md`, Zeilen 991-992:

```
The write barrier (`loomux hook pre-tool-use`) lets tools write where the
registry declares an area. A linked git worktree of an area registered with
```

ersetzen durch

```
The write barrier (`loomux hook pre-tool-use`) lets tools write where the
registry declares an area: in its wiki unless it is read-only, and in its
`path` when it has `workspace = true`. An area without `wiki` opens only its
`path`; `loomux init --brain=none` registers a project so. The index skips
such an area as long as the project declares no `[area]`; with one, every
`**/*.md` under `path` becomes its collection. A linked git worktree of an area registered with
```

`docs/de/configuration.md`, Zeilen 1022-1023:

```
Die Schreibschranke (`loomux hook pre-tool-use`) lässt Werkzeuge schreiben, wo
die Registry einen Bereich erklärt. Ein verknüpfter Git-Worktree eines Bereichs
```

ersetzen durch

```
Die Schreibschranke (`loomux hook pre-tool-use`) lässt Werkzeuge schreiben, wo
die Registry einen Bereich erklärt: in seinem Wiki, wenn er nicht
schreibgeschützt ist, und in seinem `path`, wenn er `workspace = true` hat. Ein
Bereich ohne `wiki` öffnet nur `path`; `loomux init --brain=none` registriert
ein Projekt so. Der Index überspringt einen solchen Bereich, solange das
Projekt kein `[area]` erklärt; mit einer Erklärung wird jedes `**/*.md` unter
`path` seine Sammlung. Ein verknüpfter Git-Worktree eines Bereichs
```

Danach die Absätze mit `go run ./cmd/loomux dev …` nicht neu umbrechen; nur die ersetzten Zeilen ändern sich.

- [ ] **Step 4: README**

`README.md:244`: in der Zeile `loomux init  # set a project up in modules (hooks, brain, graph): binary, config, host entries, git hooks, merge hook, skills; …` nach `merge hook, skills` einfügen: `, and the registry entry the write barrier needs (with --brain=none a workspace without wiki)`.

`README.de.md:245`: in der Zeile `loomux init  # Richtet ein Projekt in Modulen ein (hooks, brain, graph): Binary, Konfiguration, Host-Einträge, Git-Hooks, Merge-Hook, Skills; …` nach `Merge-Hook, Skills` einfügen: `, dazu den Registry-Eintrag, den die Schreibschranke braucht (mit --brain=none ein Workspace ohne Wiki)`.

Beide Zeilen per Edit, nicht per `sed` (Backticks und lange Zeile).

- [ ] **Step 5: Nachtrag zur 4e-Umstellungsspec**

Ans Ende von `docs/.superpowers/specs/2026-09-29-loomux-stufe-4e-umstellung-vorbereitet-design.md` anfügen:

```markdown

## Nachtrag 2026-10-02: „kein eigener Bereich“ heißt „kein Brain-Bereich“

Der Abschnitt „Entschieden am 2026-09-29“ sagt für `iam_backend`, `iam_frontend` und
`iam_workers` „keinen eigenen Bereich und kein Wiki“ und zugleich „Hooks, Guards und Graph
wie bei allen anderen“. Beides zugleich ging mit `init` bis 6.1.0 nicht: Ohne
Registry-Eintrag verweigert der Schreibwächter jeden Write im Projekt (gemessen am
2026-10-02). Gemeint ist **kein Brain-Bereich**: kein Wiki, keine Erklärung `[area]`, kein
Merge-Hook, aber ein Registry-Eintrag mit `path` und `workspace = true` ohne `wiki`.

Die drei Projekte haben diesen Eintrag seit dem 2026-10-02, von Hand eingetragen. Künftig
schreibt ihn `init --yes --brain=none` selbst (Teil `workspace`, Spec
`2026-10-02-loomux-brain-none-workspace-design.md`); an einer Wurzel mit Eintrag lässt `init`
ihn unberührt und nennt den Grund als Notiz. Der Reindex überspringt die drei Bereiche,
weil sie kein `[area]` erklären, und meldet bei jedem Lauf `skipping project/<name>: …`.
```

- [ ] **Step 6: Korrektur in der eigenen Spec**

In `docs/.superpowers/specs/2026-10-02-loomux-brain-none-workspace-design.md` den Abschnitt `## Was ein Bereich ohne Wiki sonst bewirkt` bis vor `## Tests` ersetzen durch:

```markdown
## Was ein Bereich ohne Wiki sonst bewirkt

Gelesen im Code von `origin/master` am 2026-10-02 und beim Planen per Test geprüft: Ein
Bereich ohne `wiki` wird vom Index **nicht** übersprungen, weil das Wiki fehlt.
`OwnWikiPrefix` gibt nil und weitet den Lauf damit auf den ganzen `path`
(`TestReindexSkippingMissingAreaAndManifest` indiziert den Bereich `valid` ohne `wiki`).
Der Workspace, den `init` schreibt, bekommt keine Sammlung, weil das Projekt kein `[area]`
erklärt: `ReadAreaManifestUntilStage4` scheitert, `indexArea` meldet
`skipping <scope>: …` und überspringt ihn, bei jedem Reindex. Bekommt das Projekt später
eine Erklärung, wird jedes `**/*.md` unter `path` seine Sammlung. Der Plan hält das
heutige Verhalten mit einem Test fest (Reindex legt für den Bereich keine Sammlung an).
Wartung und Prüfung (`check/run/run.go`, `check/run/targets.go`,
`check/house/federation.go`, `apply/resolve.go`) sind nur gelesen, nicht geprobt.
```

- [ ] **Step 7: Prüfen, dass keine Doku das alte Verhalten weiter behauptet**

Run: `git grep -n -e 'brain=none' -e 'brain none' -- '*.md' ':!docs/.superpowers/plans' > <scratch>/grep-docs.log; cat <scratch>/grep-docs.log`
Expected: jede Fundstelle außerhalb der Arbeitspapiere nennt den Workspace-Eintrag oder spricht nur von der Befehlszeile der Umstellung (`init_args`). Eine Stelle, die `--brain=none` als „ohne Registrierung“ beschreibt, wird im selben Commit angepasst.

Run: `go test ./internal/plancheck/ -count=1 > <scratch>/plancheck.log 2>&1; tail -2 <scratch>/plancheck.log`
Expected: `ok` (Migrationsplan und Spec-Tabellen bleiben unberührt; dieser Plan startet und beendet keine Stufe).

- [ ] **Step 8: Commit**

Write `<scratch>/msg3.txt`:

```
docs(setup): describe the workspace entry of init without the brain

The init reference names the new hooks part workspace and what
--brain=none now registers; the configuration reference says that an area
without wiki opens only its path and when the index takes it in.
```

Run: `git add docs/en/cli-reference.md docs/de/cli-reference.md docs/en/configuration.md docs/de/configuration.md README.md README.de.md docs/.superpowers/specs/2026-09-29-loomux-stufe-4e-umstellung-vorbereitet-design.md docs/.superpowers/specs/2026-10-02-loomux-brain-none-workspace-design.md && git commit -F <scratch>/msg3.txt > <scratch>/commit3.log 2>&1; tail -5 <scratch>/commit3.log`
Expected: Tor grün, ein Commit.

---

## Pull Request

Label `release:minor` (ein `feat`, neuer Teil und neue Handlung, kompatibel). Der Rumpf trägt `Release: minor — init --brain=none registers the project as a workspace so the write barrier opens it.` und

```markdown
## Changelog

### Added
- `loomux init` has a new hooks part `workspace`: with the brain module off (`--brain=none`) it registers the project as a workspace without wiki, so the write barrier lets tools write in it.
```

Vor dem Push gruppiert der Skill `release-pr` die Commits (Task 1 und 2 sind ein Thema und dürfen zu einem `feat`-Commit gefaltet werden); den Push nennt der Agent dem Menschen.
