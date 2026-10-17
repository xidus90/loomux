# Schalter `llm_config`: Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ein Feld `llm_config` je Area in `registry.toml`, das nur ein
Mensch umlegt. Ist es an, dürfen Agenten die `.loomux/config.toml` genau
dieses Projekts mit Datei-Werkzeugen und mit `loomux config set|unset … --yes`
bearbeiten.

**Architecture:** `config` liest, schreibt und ändert das Feld. `brain/guard`
bekommt die eine Auskunft „welche Area liegt an dieser Wurzel, und ist ihr
Schalter an“ (`AreaAt`) und nimmt das eigene Manifest dieser Area aus der
Manifest-Sperre in `Decide`. `hooks` fragt dieselbe Auskunft lazy ab, nur wenn
eine Manifest-Regel oder eine `config set`-Zeile anschlägt. Die CLI bekommt
`config llm [on|off]`, `init` stellt am Ende eine Frage.

**Tech Stack:** Go (Stdlib), `internal/config`, `internal/brain/guard`,
`internal/hooks`, `internal/cli`, `internal/tui`.

**Spec:** `docs/.superpowers/specs/2026-10-16-llm-config-switch-design.md`.
Die Nummern E1 … E9 verweisen auf deren Entscheidungen.

## Global Constraints

- Code, Kommentare, Meldungen, Commits englisch; Plan, Spec und Wiki deutsch.
- 100 % Coverage je Funktion; Ausnahme nur mit `//coverage:exempt <reason>` direkt über `func`.
- TDD: Jede neue Testzeile läuft vor dem Code rot.
- Commits nach Conventional Commits, Scope ist ein Code-Bereich (`config`, `guard`, `hooks`, `cli`, `docs`); kein `Co-Authored-By`, Autor ist der Mensch.
- Gate vor jedem Commit: `.githooks/pre-commit` läuft von selbst; von Hand `sh ci/gate.sh`.
- Feldname in der Registry: `llm_config`, Typ bool; fehlt er, gilt `false`; geschrieben wird er nur, wenn er `true` ist (E1, E5).
- Befehl: `loomux config llm` (liest), `loomux config llm on|off` (schreibt, nur Mensch) (E2).
- Die Wächter-Differenzbatterie (`LOOMUX_RECORD_GUARD_BATTERY`, `internal/hooks/guardholes_test.go`) zeigt keine Zeile, die vorher verweigert war und mit Schalter **aus** jetzt durchgeht.
- Hook-Latenz: Ein Aufruf, der weder das Manifest noch `loomux config set|unset` trifft, liest die Registry nicht zusätzlich (Messung wie in `docs/de/benchmarks.md`).
- Fehler schließen: unlesbare Registry, keine Area an der Wurzel → Schalter gilt als aus (E8).
- Agenten pushen nie.

## Review Focus

1. **Fremdes Manifest bei offenem Schalter:** `../other/.loomux/config.toml`, das Manifest eines Geschwister-Worktrees eines anderen Projekts oder ein absoluter Pfad dorthin bleibt verweigert. Tests in Task 2 (`Decide`) und Task 3 (Pfadregel).
2. **Schreibweise unter Windows:** `.LOOMUX\Config.toml` und `.loomux/config.toml` sind dieselbe Datei; die Ausnahme gilt für beide, die Sperre für ein fremdes Projekt auch für beide. Tests in Task 2 und Task 3.
3. **Verkettete Zeilen:** `loomux config set a b --yes && loomux config llm on` und `loomux config set a b --yes; loomux area add …` bleiben verweigert, auch bei offenem Schalter. Test in Task 3.
4. **Löschen statt Bearbeiten:** `rm -rf .loomux` und `Remove-Item .loomux -Recurse` bleiben bei offenem Schalter verweigert (Run-Dateien, Hook-Zustand, `armed.toml`). Test in Task 3.
5. **Schalter für eine Wurzel ohne Area:** In einem Projekt ohne Registry-Eintrag bricht `config llm on` mit Hinweis auf `loomux init` ab, und der Wächter bleibt zu. Tests in Task 2 und Task 4.

---

## Dateien

| Datei | Verantwortung |
|---|---|
| `internal/config/registry.go`, `registrywrite.go` | Feld lesen, rendern, `SetLLMConfig` |
| `internal/brain/guard/registry.go`, `areaat.go` (neu), `guard.go` | `AreaAt`, Ausnahme in `Decide` |
| `internal/brain/guard/run.go` | Hinweis an jeder Ablehnung |
| `internal/hooks/guard.go`, `guardjudge.go`, `pretool.go` | Ausnahme in Pfadregel und für `config set`, Hinweis |
| `internal/cli/config.go`, `config_llm.go` (neu) | `config llm`, Zeile in `config list` |
| `internal/cli/init.go`, `init_ui.go` | Frage am Ende |
| `docs/de|en/configuration.md`, `docs/de|en/hooks.md`, `docs/wiki/…` | Doku, Wiki |

---

### Task 1: Feld `llm_config` in der Registry

**Files:**
- Modify: `internal/config/registry.go` (`Area`, Flag-Schleife in `area`)
- Modify: `internal/config/registrywrite.go` (`RenderRegistry`, neue `SetLLMConfig`)
- Test: `internal/config/registry_test.go`, `internal/config/registrywrite_test.go`

**Interfaces:**
- Produces: `Area.LLMConfig bool`; `func SetLLMConfig(stateDir, scope string, on bool) error`

- [ ] **Step 1: Failing tests**
  - `TestReadRegistryReadsLLMConfig`: `llm_config = true` → `Area.LLMConfig == true`; Feld fehlt → `false`; `llm_config = "yes"` → Fehler, der `llm_config` nennt.
  - `TestRenderRegistryWritesLLMConfigOnlyWhenTrue`: `true` rendert `llm_config = true` nach `workspace`; `false` rendert die Zeile nicht, die Bytes einer Area ohne Feld bleiben wie vorher.
  - `TestSetLLMConfig`: setzt und löscht das Feld einer registrierten Scope, lässt die anderen Areas byte-gleich; unbekannte Scope → Fehler `scope "x" is not registered`, Datei unverändert; fehlende Registry → Fehler.
- [ ] **Step 2: Run** `go test ./internal/config/ -run 'LLMConfig'` → FAIL.
- [ ] **Step 3: Implement.** `"llm_config"` in die Flag-Liste von `area` und von `RenderRegistry`. `SetLLMConfig` läuft wie `AddArea` in `underRegistryLock`, liest mit `ReadRegistry` (nicht `readRegistryIfPresent`: ohne Registry gibt es nichts umzuschalten) und schreibt mit `writeRegistryLocked`.
- [ ] **Step 4: Run** `go test ./internal/config/` → PASS, Coverage 100 %.
- [ ] **Step 5: Commit** `feat(config): carry llm_config on a registered area`

### Task 2: `AreaAt` und das offene Manifest in `Decide`

**Files:**
- Modify: `internal/brain/guard/registry.go` (`area.llmConfig`)
- Create: `internal/brain/guard/areaat.go`
- Modify: `internal/brain/guard/guard.go` (Schritt 1 in `Decide`)
- Test: `internal/brain/guard/areaat_test.go`, `internal/brain/guard/guard_test.go`

**Interfaces:**
- Consumes: `config.Area.LLMConfig` (Task 1)
- Produces: `func AreaAt(root, stateDir string) (scope string, llmConfig bool, err error)` — `scope == ""` heißt: keine Area an dieser Wurzel.

- [ ] **Step 1: Failing tests** (`areaat_test.go`)
  - `TestAreaAtFindsTheAreaWhosePathIsRoot`: Area mit `path` = Wurzel → ihre Scope und ihr Schalter; Wurzel in einem Unterordner einer Area → `""` (nur die Projektwurzel trägt das Manifest).
  - `TestAreaAtFollowsALinkedWorktree`: Wurzel ist ein verknüpfter Worktree einer `workspace`-Area (Aufbau wie in `worktree_test.go`) → Scope und Schalter der Area.
  - `TestAreaAtComparesPathsAsTheFileSystemDoes`: unter Windows trifft `c:\…\LOOMUX` die Area `C:/…/loomux`.
  - `TestAreaAtRefusesAnUnreadableRegistry`: kaputte Registry → `err != nil`.
- [ ] **Step 2: Failing tests** (`guard_test.go`)
  - `TestDecideOpensTheOwnManifestWhenTheSwitchIsOn`: Write auf `<area>/.loomux/config.toml`, Area `workspace = true`, `llm_config = true` → nicht verweigert; dasselbe mit `.LOOMUX\Config.toml` unter Windows.
  - `TestDecideKeepsEveryOtherManifestShut`: Schalter an in Area A; Write auf das Manifest von Area B (Schalter aus), auf `<A>/sub/.loomux/config.toml` und auf ein Manifest außerhalb aller Wurzeln → verweigert mit dem bisherigen Grund.
  - `TestDecideKeepsTheManifestShutWhenTheRegistryBreaks`: Schalter an, Registry danach kaputt → verweigert.
  - `TestDecideOpensTheManifestOfALinkedWorktree`: Manifest im verknüpften Worktree einer Area mit Schalter an → nicht verweigert.
- [ ] **Step 3: Run** `go test ./internal/brain/guard/ -run 'AreaAt|Manifest'` → FAIL.
- [ ] **Step 4: Implement.** `AreaAt` löst `root` mit `ResolvePath` auf, liest mit `readRegistry`, vergleicht mit `pathsEqual` gegen jeden aufgelösten `area.path`, sonst über `linkedWorktreeRoots([]string{root}, areas)` gegen die Wurzel. In `Decide` bleibt Schritt 1 vor dem Registry-Lesen; trifft `pick(resolved, isManifest)`, wird nur dann durchgelassen, wenn **jedes** Manifest-Ziel `filepath.Join(<Wurzel einer Area mit llmConfig>, ".loomux", "config.toml")` ist (Vergleich mit `pathsEqual`); jeder Fehler beim Lesen lässt die bisherige Ablehnung stehen. Danach laufen die übrigen Schritte unverändert (Allow-List, Zonen).
- [ ] **Step 5: Run** `go test ./internal/brain/guard/` → PASS, Coverage 100 %.
- [ ] **Step 6: Commit** `feat(guard): open an area's own manifest when its llm_config is on`

### Task 3: Ausnahmen im Pre-Tool-Wächter und Hinweis

**Files:**
- Modify: `internal/hooks/guardjudge.go` (`judge.configOpen`, `judgePath`)
- Modify: `internal/hooks/guard.go` (`checkTool`, neue `editsOwnConfig`)
- Modify: `internal/hooks/pretool.go` (Zustandsverzeichnis an den Judge, Hinweis)
- Modify: `internal/brain/guard/run.go` (Hinweis anhängen)
- Test: `internal/hooks/guard_test.go`, `internal/hooks/guardshell_test.go`, `internal/hooks/pretool_test.go`

**Interfaces:**
- Consumes: `guard.AreaAt` (Task 2)
- Produces: `func checkToolOpen(root, tool string, input map[string]any, policy config.Policy, open func() bool) []string` (`checkTool` ruft es mit `func() bool { return false }`); `func editsOwnConfig(line string) bool`; `guard.Run(stdin, stdout, stderr, stateDir, note string)`.

- [ ] **Step 1: Failing tests**
  - `TestTheOwnManifestIsOpenWhenTheSwitchIsOn`: `Write`/`Edit` auf `.loomux/config.toml` und `.LOOMUX/Config.toml` mit `open = true` → kein `manifestReason`; mit `open = false` → `manifestReason` wie heute.
  - `TestAForeignManifestStaysShut`: `open = true`, Ziel `../other/.loomux/config.toml`, `sub/.loomux/config.toml` → `manifestReason`.
  - `TestRemovingLoomuxStaysRefused`: `open = true`, `rm -rf .loomux`, `Remove-Item .loomux -Recurse` → verweigert.
  - `TestEditsOwnConfig`: wahr für `loomux config set model.enabled true --yes` und `loomux config unset flow.default --yes`; falsch ohne `--yes`, mit `--root x`, `--root=x`, `--global`, `--propose`, mit `&&`, `;`, `|`, Heredoc, für `loomux config llm on`, `loomux config apply --all --yes`, `loomux area add …`.
  - `TestConfigSetPassesOnlyWhenTheSwitchIsOn`: die wahren Zeilen aus `TestEditsOwnConfig` → mit `open = true` ohne Grund, mit `open = false` der bisherige `writesConfiguration`-Grund; `loomux config set a b --yes && loomux config llm on` → mit `open = true` verweigert.
  - `TestTheSwitchIsNotReadForAnOrdinaryCall`: `open` zählt Aufrufe; `Write` auf `src/a.go` und `git status` → 0 Aufrufe.
  - `TestARefusalSaysTheSwitchIsOn` (`pretool_test.go`): Schalter an, ein verweigerter Aufruf → Ablehnung endet mit `llm_config is on for this project: agents may edit .loomux/config.toml (turn it off with loomux config llm off)`; Schalter aus → dieser Satz fehlt.
- [ ] **Step 2: Run** `go test ./internal/hooks/ ./internal/brain/guard/ -run 'Manifest|EditsOwnConfig|ConfigSet|Switch|Removing'` → FAIL.
- [ ] **Step 3: Implement.**
  - `judge` bekommt `open func() bool` (in `PreToolUse` als `sync.OnceValue` über `guard.AreaAt(root, stateDir)`, Fehler → `false`). `judgePath` lässt `manifestReason` nur dann weg, wenn `strings.EqualFold(rel, ".loomux/config.toml")` und `j.open()`; alle anderen Gründe derselben Datei bleiben.
  - `editsOwnConfig` nimmt nur eine `plainLine` mit genau einem Segment; die Wörter sind `loomux config set|unset <key> [<value>]` mit `flagOn(args, "--yes", "-yes")`, ohne `--root`, `--global`, `--propose` in jeder Schreibweise. In `checkTool` entfällt der `writesConfiguration`-Grund nur bei `editsOwnConfig(line) && j.open()`.
  - Der Hinweis-Satz aus dem Test steht als Konstante in `hooks` und geht an beide Ablehnungen in `PreToolUse` (Policy und `guard.Run`), nur wenn `open()` wahr ist; `guard.Run` hängt einen nicht leeren `note` mit Leerzeile an den Grund.
- [ ] **Step 4: Run** `go test ./internal/hooks/ ./internal/brain/guard/` → PASS; `LOOMUX_RECORD_GUARD_BATTERY=1 go test ./internal/hooks/ -run Battery` zeigt keine neu durchgelassene Zeile.
- [ ] **Step 5: Run** die Hook-Messung aus `docs/de/benchmarks.md`; Werte für „Write außerhalb des Manifests“ bleiben im Rauschen, neue Zeile für „Write auf das Manifest, Schalter an“ in die Benchmark-Seite.
- [ ] **Step 6: Commit** `feat(hooks): let an agent edit its project's config when llm_config is on`

### Task 4: `loomux config llm [on|off]` und `config list`

**Files:**
- Create: `internal/cli/config_llm.go`
- Modify: `internal/cli/config.go` (Dispatch, Usage, `configList`)
- Modify: `internal/hooks/guard.go` (`programWrites`, Fall `config`)
- Test: `internal/cli/config_llm_test.go`, `internal/cli/config_test.go`, `internal/hooks/guardshell_test.go`

**Interfaces:**
- Consumes: `guard.AreaAt`, `config.SetLLMConfig`, `config.StateDir`
- Produces: `func configLLM(root string, args []string, stdout, stderr io.Writer) int`

- [ ] **Step 1: Failing tests**
  - `TestConfigLLMShowsTheSwitch`: ohne Argument → `llm_config: off (project/x)` bzw. `on`, Exit 0.
  - `TestConfigLLMTurnsTheSwitch`: `on` → Registry trägt `llm_config = true`, Ausgabe `llm_config: on (project/x)`; `off` → Zeile weg.
  - `TestConfigLLMOutsideAnArea`: keine Area an der Wurzel → Exit 1, stderr nennt `loomux init`; unbekanntes Argument, `--global`, `--yes` → Usage, Exit 2.
  - `TestConfigListNamesTheOpenSwitch`: Schalter an → `config list` beginnt mit der Zeile `llm_config: on — agents may edit this file`; `--json` bleibt unverändert.
  - `TestTheGuardRefusesConfigLLMOnAndOff` (`guardshell_test.go`): `loomux config llm on`, `loomux config llm off` → verweigert, auch mit offenem Schalter; `loomux config llm` → durchgelassen.
- [ ] **Step 2: Run** `go test ./internal/cli/ ./internal/hooks/ -run 'ConfigLLM|OpenSwitch'` → FAIL.
- [ ] **Step 3: Implement.** `configCommand` reicht `sub == "llm"` vor dem Flag-Parsen an `configLLM` weiter (nur `--root` erlaubt). Wurzel wie bei `resolveConfigTarget`. In `programWrites` unter `case "config"` kommt `"llm"` in die lesenden Fälle nur bei `len(args) == 2`; `on`/`off` fallen in den verweigernden Rest.
- [ ] **Step 4: Run** `go test ./internal/cli/ ./internal/hooks/` → PASS, Coverage 100 %.
- [ ] **Step 5: Commit** `feat(cli): add loomux config llm to show and turn the switch`

### Task 5: Frage am Ende von `loomux init`

**Files:**
- Modify: `internal/cli/init.go` (nach `printReport`, vor `return 0`)
- Modify: `internal/cli/init_ui.go` (neue `askLLMConfig`)
- Test: `internal/cli/init_ui_test.go`, `internal/cli/init_test.go`

**Interfaces:**
- Consumes: `guard.AreaAt`, `config.SetLLMConfig`, `tui.Confirm`
- Produces: `func askLLMConfig(t tui.Terminal, scope string) (bool, error)`

- [ ] **Step 1: Failing tests**
  - `TestAskLLMConfigAsksOnce`: Frage-Text `let the LLM edit .loomux/config.toml of this project? (you can change this later with loomux config llm on|off)`; Enter ohne Auswahl → `false`.
  - `TestInitSetsTheSwitchOnYes`: interaktiver Lauf (Test-Terminal wie in `init_ui_test.go`), Antwort Ja → Registry trägt `llm_config = true`; Nein → kein Feld.
  - `TestInitAsksNothingWithYesOrWithoutArea`: `--yes`, `--dry-run`, abgelehnte `area-add` oder fehlgeschlagener Lauf → keine Frage, kein Feld.
  - `TestInitSetsTheSwitchOnARegisteredArea`: zweiter `init` in einem schon registrierten Projekt, Ja → Feld gesetzt (über `SetLLMConfig`, nicht `AddArea`).
- [ ] **Step 2: Run** `go test ./internal/cli/ -run 'LLMConfig'` → FAIL.
- [ ] **Step 3: Implement.** Nur wenn `!o.yes && !o.dryRun`, der Lauf ohne Fehler endete und `guard.AreaAt(root, config.StateDir())` eine Scope liefert: Terminal mit `openTerminal` öffnen, `askLLMConfig`, bei Ja `config.SetLLMConfig`; Fehler beim Setzen → stderr und Exit 1.
- [ ] **Step 4: Run** `go test ./internal/cli/` → PASS, Coverage 100 %.
- [ ] **Step 5: Commit** `feat(cli): ask at the end of init whether the LLM may edit the config`

### Task 6: Doku, Grundsatz, Wiki

**Files:**
- Modify: `docs/de/configuration.md`, `docs/en/configuration.md` (Feld `llm_config`, `config llm`, Init-Frage)
- Modify: `docs/de/hooks.md`, `docs/en/hooks.md` (Ausnahmen von Manifest-Sperre und Mensch-Befehl, Hinweis-Satz)
- Modify: `docs/wiki/topics/schreibschranke.md` (Abschnitt „Der Schalter `llm_config`“), `docs/wiki/topics/architektur-grundsaetze.md` (Satz zur Manifest-Regel nach E9), `docs/wiki/index.md`
- Weitere Seitentypen nach `AGENTS.md`

- [ ] **Step 1:** Doku de/en mit gleicher Gliederung (Link-Parität nach `AGENTS.md`).
- [ ] **Step 2:** Grundsatz in `architektur-grundsaetze.md` ergänzen: Standard bleibt „nie beschreibbar“; ein Mensch kann es pro Projekt mit `loomux config llm on` aufheben und übernimmt damit, dass das LLM auch die Grenzen dieser Config ändern kann.
- [ ] **Step 3:** `schreibschranke.md`: Was der Schalter öffnet (genau das eigene Manifest, auch im verknüpften Worktree), was zu bleibt, Restrisiko; `implemented_in` nachziehen.
- [ ] **Step 4: Run** `sh ci/gate.sh` → grün (Doku-Lanes, Wiki-Lint).
- [ ] **Step 5: Commit** `docs: describe the llm_config switch`
