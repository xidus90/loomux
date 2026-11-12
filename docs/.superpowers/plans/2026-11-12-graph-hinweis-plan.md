# Graph-Hinweis im Wächter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Der PreToolUse-Hook lehnt das erste grep über Code einer Session einmal ab und nennt `loomux graph callers <Name>`.

**Architecture:** Ein Detektor in `internal/hooks` liest Grep-Werkzeug und Shell-Zeilen mit der Zerlegung des Wächters; `PreToolUse` fragt ihn, nachdem die Policy nichts verweigert hat und bevor die Schreibschranke läuft. Eine Markierungsdatei im Session-Ordner (`internal/sessions`) sorgt für „einmal“; ein Schalter `[guard] graph_hint` in der Policy schaltet ab.

**Tech Stack:** Go 1.27, bestehende Pakete `internal/hooks`, `internal/config`, `internal/sessions`, `internal/code/store`.

**Spec:** `docs/.superpowers/specs/2026-11-12-graph-hinweis-design.md`

## Global Constraints

- PreToolUse-Budget 35 ms; der Hinweis liest außer der Markierung keine Datei und prüft den Graph nur per `os.Stat`.
- Jeder Fehler auf dem Weg des Hinweises lässt den Aufruf durch (kein `guard.Refuse` aus einem Fehler).
- Markierung vor der Ablehnung schreiben; scheitert das Schreiben, keine Ablehnung.
- Wortlaut eigen, nie „loomux policy refused“.
- Code-Endungen: `.go`, `.py`, `.gd`. Doku-/Log-/Konfig-Bereiche: `.md`, `.log`, `.json`, `.toml`, `.yaml`, `.yml`, Pfade unter `docs/`.
- TDD, 100 % Coverage, Kommentare englisch, Doku de und en.

## Review Focus

1. Ein grep in einer Pipe hinter einem anderen Befehl (`git log | grep Foo`) liest Standardeingabe, keinen Code: kein Hinweis. (Task 3)
2. Ein grep mit Code-Endung im Muster, aber Doku als Bereich (`grep -rn "func X" docs/`): kein Hinweis. (Task 3)
3. Payload ohne `session_id`: kein Hinweis, Aufruf geht durch. (Task 4)
4. `cwd` in einem Unterordner des Bereichs mit Graph: der Graph wird über die Elternordner gefunden. (Task 4)
5. Zweiter identischer Aufruf in derselben Session und demselben Agenten geht durch; ein anderer Subagent (`agent_id`) bekommt seinen eigenen Hinweis. (Task 2, Task 4)

---

### Task 1: Schalter `[guard] graph_hint`

**Files:**
- Modify: `internal/config/guardsettings.go`, `internal/config/policy.go`
- Test: `internal/config/guardsettings_test.go` (bestehend erweitern)
- Docs: `docs/de/configuration.md`, `docs/en/configuration.md` (`[guard]`-Abschnitt und kommentiertes Muster in Abschnitt 3)

**Interfaces:**
- Produces: `config.Policy.GraphHintOff bool` (Nullwert = Hinweis an); `GuardKeys()` enthält `"graph_hint"`.

- [ ] **Step 1: Failing tests** — `TestGuardReadsGraphHint`: `[guard]\ngraph_hint = false` → `GraphHintOff == true`; ohne Schlüssel → `false`; `graph_hint = "no"` → Fehler mit `[guard] graph_hint must be a boolean, found string`.
- [ ] **Step 2:** `go test ./internal/config/ -run GraphHint` → FAIL.
- [ ] **Step 3:** `parseGuard` gibt `guardSettings{strict, graphHintOff bool}` zurück; `ReadPolicy` setzt beide Felder.
- [ ] **Step 4:** Test grün, `go test ./internal/config/` grün.
- [ ] **Step 5:** Doku de/en: Schlüssel mit Vorgabe `true`, was er tut, Verweis auf `hooks.md`.

### Task 2: Markierung einmal pro Session und Agent

**Files:**
- Create: `internal/sessions/once.go`
- Test: `internal/sessions/once_test.go`

**Interfaces:**
- Produces: `sessions.MarkOnce(root, sessionID, agentID, name string) (first bool, err error)` — legt `<root>/.loomux/state/hooks/<session>/<name>[.<agent>]` mit `O_CREATE|O_EXCL` an; `first` ist true nur, wenn diese Datei neu angelegt wurde. Leere `sessionID` → `(false, nil)`. Der Ordner liegt dort, wo `Forget` ihn entfernt.

- [ ] **Step 1: Failing tests** — `TestMarkOnceIsTrueOnlyTheFirstTime` (zweiter Aufruf false), `TestMarkOnceKeepsAgentsApart` (agent "a" und "b" je einmal true, "" eigene Markierung), `TestMarkOnceWithoutASessionMarksNothing`, `TestForgetRemovesTheMark`, `TestMarkOnceReportsAnUnwritableState` (Zustandsordner ist eine Datei → err != nil, first false).
- [ ] **Step 2:** FAIL. **Step 3:** implementieren, Namen über das bestehende `safeName`. **Step 4:** grün, Coverage 100 %.

### Task 3: Detektor

**Files:**
- Create: `internal/hooks/graphhint.go`
- Test: `internal/hooks/graphhint_test.go`

**Interfaces:**
- Consumes: `commandTools`, `commandLines`, `typedLines` aus `internal/hooks/guard.go`.
- Produces: `graphHintSymbol(tool string, input map[string]any) (symbol string, ok bool)`.

Regeln (aus dem Spike `nudge.py`, ins Go übertragen):
- Grep-Werkzeug: Muster `pattern`, Bereich aus `path`, `glob`, `type`.
- Shell: je Zeile die Befehle (`&&`, `||`, `;`, `|`, Zeilenumbruch); ein Befehl zählt, wenn sein Programm `grep`, `rg`, `git grep` oder `Select-String` ist **und** er nicht hinter einem `|` steht. Muster ist das erste Argument, das nicht mit `-` beginnt (oder der Wert von `-e`/`-Pattern`); der Rest ist der Bereich.
- Symbol: zuerst `func [Empfänger] X`, `type X`, dann `X\(`/`.X\(`, sonst ein bloßer Bezeichner in CamelCase/mixedCase. Nur Bezeichner mit ≥ 4 Zeichen, nicht in `{func,type,return,nil,true,false,err,string,error,struct,interface,main,test}`.
- Bereich: Code, wenn er eine Code-Endung, `--include=*.go|py|gd`, `-t go|py`, `--type=…` oder einen Pfad ohne Doku-Endung nennt; nie, wenn er nur Doku/Log/Konfig nennt; ohne Bereichsangabe bei Shell-grep nicht Code (Standardeingabe), beim Grep-Werkzeug Code.

- [ ] **Step 1: Table test `TestGraphHintSymbol`** mit mindestens diesen Zeilen (tool, input → symbol, ok):
  - Grep `{"pattern":"relInRoot\\("}` → `relInRoot`, true
  - Grep `{"pattern":"func Plan","path":"internal/verify"}` → `Plan`, true
  - Grep `{"pattern":"Schreibschranke","path":"docs/wiki"}` → false
  - Bash `grep -rn "blastAside(" --include=*.go internal` → `blastAside`, true
  - Bash `git log --oneline | grep Radius` → false
  - Bash `grep -n "func sample" -A25 internal/code/query/x_test.go` → `sample`, true
  - Bash `grep -E "timed-out|red|failed" /tmp/run.log` → false
  - Bash `sed -i 's/a/b/' internal/x.go; grep -n "preRe" internal/compat/scout.go` → `preRe`, true
  - Bash `grep -rn "Mitigation|JobObject" docs/` → false
  - PowerShell `Select-String -Path internal\*.go -Pattern "NewIndex"` → `NewIndex`, true
  - Bash `ls internal` → false; Write-Werkzeug → false
- [ ] **Step 2:** FAIL. **Step 3:** implementieren. **Step 4:** grün, 100 %.

### Task 4: Hinweis im PreToolUse

**Files:**
- Modify: `internal/hooks/pretool.go`
- Test: `internal/hooks/graphhint_test.go`

**Interfaces:**
- Consumes: `graphHintSymbol` (Task 3), `sessions.MarkOnce` (Task 2), `config.Policy.GraphHintOff` (Task 1), `store.WiringPath`.
- Produces: `graphAt(cwd string) bool` (sucht `store.WiringPath(dir)` von `cwd` aufwärts bis zum Laufwerkswurzel), `graphHintReason(symbol string) string`.

Wortlaut (exakt):
```
loomux: this repository has a code graph that answers who calls what in one step.
Instead of grepping for `<symbol>`, run `loomux graph callers <symbol>` (callers; add `--direction out` for callees) or `loomux graph blast` (what a change reaches).
This hint comes once per session: running the same command again goes through.
```

Ablauf in `PreToolUse`: nach `policyReasons` ohne Gründe und vor `guard.Run`; die Policy wird dafür aus `policyReasons` mitgegeben (kein zweites Lesen). Ablehnen nur, wenn alle gelten: `!policy.GraphHintOff`, Symbol gefunden, `graphAt(cwd)`, `MarkOnce(root, session_id, agent_id, "graph-hint")` gibt `true, nil`. Ablehnung über `guard.Refuse(stdout, stderr, graphHintReason(symbol))`.

- [ ] **Step 1: Failing tests** — `TestPreToolUseHintsOnceAtTheFirstCodeGrep` (erster Aufruf Exit 2 mit Wortlaut, zweiter Exit 0), `TestPreToolUseHintsNotWithoutAGraph`, `TestPreToolUseHintsNotWhenSwitchedOff`, `TestPreToolUseHintsNotWithoutASession`, `TestPreToolUseFindsTheGraphAboveCwd`, `TestPreToolUseHintsEachSubagentOnce`, `TestPreToolUseLetsThroughWhenTheMarkCannotBeWritten`, `TestPreToolUseHintNeverSaysPolicyRefused`.
- [ ] **Step 2:** FAIL. **Step 3:** implementieren. **Step 4:** `go test ./internal/hooks/ -run 'PreToolUse|GraphHint'` grün, Coverage der neuen Funktionen 100 %.
- [ ] **Step 5: Commit** (Tasks 1–4 zusammen): `feat(guard): point the first code grep of a session at the code graph`.

### Task 5: Matcher, Messung, Doku

**Files:**
- Modify: `internal/setup/hostfile/table.go:47` (`Grep` in den PreToolUse-Matcher), `.claude/settings.json`, die Tests, die der Kommentar bei `table.go:60` nennt
- Create: `testdata/bench/grep-code.json`, `testdata/bench/bash-grep-code.json` (Hook-Nutzlasten)
- Docs: `docs/de|en/hooks.md` (Entscheidungsweg von pre-tool-use), `docs/de|en/benchmarks.md`, `docs/wiki/syntheses/graph-hinweis-im-waechter.md`, `docs/wiki/log.md`

- [ ] **Step 1:** Matcher-Tests auf den neuen Matcher anpassen, rot sehen, Tabelle und Settings ändern, grün.
- [ ] **Step 2:** `go run ./cmd/loomux dev bench hooks` mit den neuen Fällen, kalt und warm; Ergebnis gegen 35 ms in `benchmarks.md` (de/en).
- [ ] **Step 3:** Abnahme: Runde 4 (acht Aufgaben, v0 ohne Hinweis gegen das neue Binary als Hook, je zwei Läufe) mit dem Spike-Läufer, Ergebnis in `benchmarks.md` und der Synthese; Wiki-Log; `wiki repin`.
- [ ] **Step 4: Commits:** `feat(setup): hand the Grep tool to the pre-tool hook` und `docs: …`.
