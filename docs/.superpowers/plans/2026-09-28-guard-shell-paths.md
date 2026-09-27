# Wächter: Shell-Schreibziele als Pfade prüfen — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Der Wächter liest die Ziele eines Shell-Schreib- oder Löschbefehls
mit einer Verbtabelle, faltet Braces und Globs wie eine Shell auf und prüft sie
mit derselben Pfadschleife wie Write/Edit; ein Modus `[guard] mode = "strict"`
hält zusätzlich gegen gezieltes Umgehen. Dazu werden zwei flaky Tests
behoben.

**Architecture:** Drei neue Dateien in `internal/hooks`: `pathspell.go`
(Braces, Globs gegen die Platte, Stream-Anhang), `shellwrites.go` (Leser:
Zeile → Ziele `shellTarget{path, removes, shell}` über die bestehende Kette
`lineVariants` → `splitSegments` → `readings` → `dropPrefixes`) und
`guardjudge.go` (ein `judge` je Werkzeugaufruf: eingebaute Regeln, `[policy]
paths`, Flow-Ordner, Vorfahren, im Strikt-Modus `guard.ResolvePath`,
unbekannte Verben und Expansionen). `checkTool` sammelt die Ziele beider Wege
und fragt den `judge` einmal; `writeSource`, `globName` und
`flowFolderCommand` entfallen. Der Modus kommt über `config.Policy.Strict`, das
`config.ReadPolicy` aus `[guard]` liest — alle bestehenden Tests mit
`config.Policy{}` bleiben im Standardmodus.

**Tech Stack:** Go des Moduls (`path`, `path/filepath`, `regexp`, `sync`,
`slices`, `maps`), `internal/shellwords`, `internal/brain/guard`
(`ResolvePath`, `WriteTargets`, `IsWritingTool`), `internal/config`,
`internal/config/schema`, `golang.org/x/sys/windows` (nur im Windows-Test,
schon im `go.mod`). Keine neue Abhängigkeit.

**Spec:** `docs/.superpowers/specs/2026-09-28-guard-shell-paths-design.md`
(Stand nach der Durchsicht durch Fable). Gelesen gegen den Code auf
`feat/flow-runtime` (Worktree `.worktrees/flow-a`, Kopf `4fa9aa9a`):
`internal/hooks/{guard,guardflow,pretool,flowruns}.go` samt Tests,
`internal/brain/guard/path.go`, `internal/config/{policy,flowsettings,modules}.go`,
`internal/config/schema/{schema,validate}.go` samt Tests,
`internal/cli/cases_4c1_test.go`, `internal/cases/runner.go`,
`internal/code/query/{git,git_test,blast_test}.go`,
`testdata/bench/flow-hooks.json`, `docs/en/{hooks,configuration,flows,cli-reference,benchmarks}.md`.

## Vor Task 1: Zweig auf den Stand von #50 bringen

Der Code, den dieser Plan ändert, liegt auf `feat/flow-runtime` (PR #50). Der
Zweig `fix/guard-shell-paths` trägt heute nur Spec und Plan auf `master`.

- [ ] **Schritt 0.1: Stand prüfen.** Im Worktree `.worktrees/guard-shell`:
  `git rev-parse --show-toplevel --abbrev-ref HEAD` muss den Worktree-Pfad und
  `fix/guard-shell-paths` nennen. Dann `git fetch origin` und
  `gh pr view 50 --json state,mergedAt`.
- [ ] **Schritt 0.2: Rebase, genau einer der beiden Befehle.**
  - #50 ist gemergt: `git rebase origin/master`
  - #50 ist offen (gestapelt): `git rebase origin/feat/flow-runtime`

  Danach muss `internal/hooks/guardflow.go` existieren und `writeSource` in
  `internal/hooks/guard.go` stehen. Gestapelt heißt: der spätere PR zielt auf
  `feat/flow-runtime` oder wird nach dem Merge von #50 noch einmal mit
  `git rebase origin/master` umgesetzt; `git diff origin/master...HEAD` (drei
  Punkte) zeigt dann nur diesen Zweig.
- [ ] **Schritt 0.3: Basis festhalten.** `git rev-parse HEAD` nach dem Rebase
  notieren (Akte, nicht Commit): die Messung in Task 9 baut `before.exe` aus
  genau diesem Stand.
- [ ] **Schritt 0.4: Pilot-Binary bauen**, damit die Hooks dieser Sitzung den
  Code von #50 fahren: `go build -o bin/loomux.exe ./cmd/loomux`.

## Global Constraints

- Abdeckung 100 % je Funktion; eine Ausnahme nur mit `//coverage:exempt <Grund>` direkt über `func`.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst; Regex werden per `sync.OnceValue` beim ersten Gebrauch kompiliert. Reine String-Listen als Paketvariable sind erlaubt.
- Code, Bezeichner, Kommentare, Fehlermeldungen und Commits englisch; dieser Plan und Akten deutsch.
- Commits: Conventional Commits, Scope ist ein Code-Bereich (`guard`, `config`, `cli`, `query`), kein Arbeitspapier im Text; Autor ist der Mensch, kein `Co-Authored-By`, keine Werbezeile. Nachricht per Write in eine Datei im Scratchpad, dann `git commit -F <datei>`.
- Vor jedem Commit `git rev-parse --show-toplevel --abbrev-ref HEAD` gegen `.worktrees/guard-shell` / `fix/guard-shell-paths` lesen. Der Pre-commit-Hook fährt das Tor (`sh ci/gate.sh`) und baut danach `bin/loomux.exe` neu; das ist erwartet.
- Niemand außer dem Menschen pusht. Der Worktree `.worktrees/flow-a` wird nicht verändert.
- `.loomux/config.toml` schreibt kein Agent. Ein Agent schlägt vor (`loomux config set … --propose`), ein Mensch wendet an.
- Code und Tests mit Backslashes nur per Write/Edit, nie per Heredoc; danach mit `grep` nachlesen.
- Tests arbeiten in `t.TempDir()`; `project(t)` (setzt auch `LOOMUX_STATE_DIR`) und `catalog(t, …)` aus den bestehenden Tests benutzen.
- Manifest-Grund wörtlich: `.loomux/config.toml: the manifest is where the barrier reads its own limits, so no agent may write it`.
- Braces: höchstens 64 Varianten je Wort, darüber Ablehnung.
- Modus: Schlüssel `[guard] mode`, Werte `default` und `strict`, Modul Base, fehlend = `default`, jeder andere Wert ist ein Konfigurationsfehler (Ablehnung wie bei kaputter Konfiguration).
- Budget: `pre-tool-use` höchstens 35 ms warm (Median), heute 9–13 ms.
- **Nach Task 4b** hat der Pre-commit-Hook `bin/loomux.exe` mit den neuen Regeln gebaut. Ab dann verweigert der Wächter dem Agenten dieser Sitzung selbst `rm coverage.out`, `mv bin/…`, `rm -rf .loomux/state/…`, `rm -rf bin`, `git clean -fdx` (Projektregeln `coverage.out`, `bin/*`, `.loomux/state/**`, `testdata/cases/*-source/**`). Das ist gewollt; solche Aufräumarbeit nennt der Agent dem Menschen, statt einen Umweg zu suchen.

## Review Focus

Fünf Eingaben, die die Spec nahelegt, aber nicht ausdrücklich prüft; jede hat
ihren Test in der genannten Task.

1. **Zitierter Text, der nach `(` oder `;` einen geschützten Pfad nennt** (Commit-Nachrichten dieses Repos): `git commit -m "fix(guard): keep .loomux/state/runs"` bleibt erlaubt, in beiden Modi. `git commit -m "a; rm .env b"` bleibt verweigert — dieselbe bewusste Überverweigerung wie bisher (`echo "x; loomux init"`). Tests: Task 3 (`unknown` leer), Task 4a (erlaubt/verweigert).
2. **Ein `cd` vor einem relativen Pfad:** `cd .loomux && rm config.toml` wird verweigert. Die Spec schweigt; der Leser führt dafür je Schnitt ein Basisverzeichnis mit. Tests: Task 3 und Task 4a.
3. **Geräte und Abflüsse:** `> /dev/null`, `2>nul`, `> $null`, `2>&1`, `>&2` werden in keinem Modus verweigert. Tests: Task 4a (Standard), Task 6 (strikt).
4. **Heredoc-Körper, der einen Befehl nennt:** `cat > notes.md <<'EOF'` + Zeile `rm -rf .loomux` wird verweigert, weil die Zeile am Umbruch geschnitten wird wie bisher. Benannte Grenze, im Test festgenagelt, damit eine Änderung auffällt. Test: Task 4a.
5. **Löschen an der Wurzel:** `find . -name '*.tmp' -delete` und `git clean -fdx` ohne Pfad werden verweigert, weil die Startpfade bzw. die Wurzel Vorfahr jedes geschützten Pfads sind (die Spec verlangt es für `find` und `git clean`). Test: Task 4a mit den erwarteten Gründen; die Doku nennt es (Task 9).

## Entscheidungen, die die Spec offen ließ oder die ihr widersprachen

- **E1 `--`:** Die Spec sagt, „alles nach `--`“ sei kein Ziel. Das hieße, `rm -- .loomux/config.toml` ginge durch. Entschieden: `--` selbst ist kein Ziel, jedes Wort danach ist eines, auch eines mit `-`.
- **E2 `mv` in einen Vorfahren:** Quellen einer Verschiebung sind Löschungen (Vorfahren zählen), das Ziel ist eine Schreibung (Vorfahren zählen nicht). `mv mine .loomux/flows/` bleibt erlaubt wie `cp -r mine .loomux/flows/`.
- **E3 Vorfahren:** Eine Regel ohne Schrägstrich (`*.pem`, `coverage.out`) hat keinen Vorfahren; `rm -rf build` wird nicht danach beurteilt, was es enthalten könnte. Für eine Regel mit Schrägstrich ist der feste Teil der Pfad vor dem ersten Glob-Element; `rel` ist Vorfahr, wenn der feste Teil unter `rel` liegt, oder `rel` die Wurzel `.` ist. Eine Regel `**/…` zählt außerdem, wenn `rel` auf einen führenden Teil ihres festen Teils endet (`x/.loomux`, `../sib/.loomux/state`) — sonst wäre jeder Ordner Vorfahr. Keine Plattensuche. Vorfahren gelten für eingebaute Regeln, `[policy] paths` und Flow-Ordner.
- **E4 unzerlegbare Segmente:** Die geratenen pfadartigen Wörter gelten als Schreibung, nicht als Löschung — sonst machte ein einsamer `.` in einem kaputten Segment jeden Befehl zur Löschung der Wurzel. Geraten wird nur im anführungszeichenbewussten Schnitt; der blinde Schnitt zerlegt Anführungszeichen-Text (Commit-Nachrichten) und würde ihn verweigern. Dasselbe gilt für die „unbekannten Programme“ des Strikt-Modus.
- **E5 Leseliste:** Über die Spec hinaus zählen `echo`, `printf`, `Write-Output`, `cd`, `Set-Location`, `sl`, `pushd`, `popd`, `Test-Path`, `gc`, `gci`, `sls` als lesend; sie nennen Pfade, ohne sie zu schreiben.
- **E6 Strikt, Programmname egal:** Wörtlich genommen machte das `git config set`, `gh config set`, `npm init`, `cat config` zu Konfigurationsschreibungen. Entschieden: im Strikt-Modus gilt jedes Programm als loomux, außer einem nackten Namen (ohne Pfad) aus `knownTools` (git, gh, go, npm, npx, pnpm, yarn, cargo, uv, uvx, poetry, pip, pip3, docker, dotnet, terraform, kubectl, helm, make, cmake), der Verbtabelle oder der Leseliste. Ein Pfad auf eine Datei dieses Namens (`./git.exe`) zählt nicht als bekannt.
- **E7 Strikt, unbekanntes Programm:** Ein Wort, dessen Pfad die Wurzel `.` ist, zählt nicht (`go vet .`, `make .`); andere Wörter werden mit Vorfahren geprüft. Für ein Wort mit Leerzeichen (`sh -c "…"`) werden auch dessen Felder geprüft. Folge: im Strikt-Modus verweigert der Wächter in diesem Repo `go build -o bin/loomux.exe ./cmd/loomux` (Projektregel `bin/*`), den Befehl, den `AGENTS.md` zum Neubauen nennt. Darum bleibt dieses Repo im Standardmodus; kein Task schreibt `[guard]` in `.loomux/config.toml` (siehe Task 5, letzter Schritt).
- **E8 Strikt, Expansion:** Ein leerer fester Teil (`> $X`, `> $env:TMP/x`) wird nicht beurteilt: die Expansion kann jeder absolute Pfad sein, und dort entscheidet die Schreibschranke.
- **E9 Geräte im Strikt-Modus:** `guard.ResolvePath` auf `nul`, `con` usw. darf nicht zur Ablehnung führen; ein Fehler bei einem Gerätenamen wird übergangen.
- **E10 .NET:** Über die Spec hinaus zählt `[IO.File]::Copy*` als Schreibung; `Directory.Move` löscht Quelle und Ziel (Überverweigerung, benannt). Argumente werden mit `/` statt `\` gelesen.
- **E11 Reihenfolge der Gründe:** Befehlsregeln, Konfigurations- und Torantwort-Gründe zuerst, dann die Pfadgründe aller Ziele, jeder Grund einmal. Der Flow-Grund nennt je Flow einen Namen (bisher: ein Grund mit allen Namen).
- **E13 Brace-Bruchstücke:** Die Spec sagt, die Bruchstücke der Variante mit abgesetzten Braces ergäben kein Ziel. Für ein löschendes Verb stimmt das nicht ganz: aus `rm -rf .loomux/flows/{mine,zz}` wird in dieser Variante auch `.loomux/flows/` zum Löschziel, und die Vorfahrenregel nennt dann die geschützten Flows. Das ist eine Überverweigerung, keine Lücke (die unzerlegte Variante trägt die richtige Expansion); sie bleibt und wird in `docs/{en,de}/hooks.md` unter den Überverweigerungen genannt.
- **E12 Aufteilung:** Task 4 ist zweigeteilt (4a: Pfadprüfung neben `writeSource`, Grundlinie gegen den neuen Weg; 4b: Umschalten und Entfernen), so wie die Spec „bevor `writeSource` fällt“ verlangt. Task 5 (Konfiguration) und 6 (Strikt-Verhalten) sind getrennt, weil ein Reviewer den Schlüssel ohne das Verhalten annehmen kann.

## Dateien

- Create `internal/hooks/pathspell.go`, `internal/hooks/pathspell_test.go` — Task 2
- Create `internal/hooks/shellwrites.go`, `internal/hooks/shellwrites_test.go` — Task 3
- Create `internal/hooks/guardjudge.go`, `internal/hooks/guardshell_test.go` — Task 4a
- Create `internal/hooks/guardstrict_test.go`, `internal/hooks/guardstrict_windows_test.go`, `internal/hooks/guardstrict_other_test.go` — Task 6
- Create `internal/config/guardsettings.go`, Tests in `internal/config/policy_test.go` — Task 5
- Create `testdata/bench/guard-shell-hooks.json` — Task 9
- Modify `internal/hooks/guard.go`, `internal/hooks/guardflow.go`, `internal/hooks/guard_test.go`, `internal/hooks/guardflow_test.go`, `internal/hooks/pretool_test.go`
- Modify `internal/config/policy.go`, `internal/config/schema/schema.go`, `internal/config/schema/schema_test.go`
- Modify `internal/cli/cases_4c1_test.go` — Task 7
- Modify `internal/code/query/…` je nach Befund — Task 8
- Modify `docs/{en,de}/{hooks,configuration,cli-reference,flows,benchmarks}.md`, `README.md`, `README.de.md`

---

### Task 1: `**/` im Matcher, loomux' eigene Regeln unter jedem Verzeichnis, Manifest als Pfadregel

**Files:**
- Modify: `internal/hooks/guard.go` (`builtinPathRules`, `matchGlob`)
- Modify: `internal/hooks/guardflow.go` (`flowFolderReasons`, neu `flowFolderName`)
- Test: `internal/hooks/guard_test.go`, `internal/hooks/guardflow_test.go`

**Interfaces:**
- Consumes: nichts Neues.
- Produces: `const manifestReason string`; `matchGlob(pattern, name string) (bool, error)` versteht ein führendes `**/`; `flowFolderName(rel string) (name string, under bool)`. Die Regeln `**/.loomux/config.toml`, `**/.loomux/state/hooks/**`, `**/.loomux/state/runs/**` stehen in `builtinPathRules`.

Kippungen dieser Task (Write/Edit): ein Write/Edit auf `.loomux/config.toml`
wird jetzt schon von der Policy verweigert (Umschlag „loomux policy refused
this tool call“ mit dem Manifest-Grund) statt erst von der Schreibschranke;
`docs/.loomux/flows/example/flow.toml` (bisher erlaubt in
`TestAWritingToolMayNotTouchABundledFlowFolder`) wird verweigert. Der Fall
`testdata/cases/1a/hook-pre-tool-use/barrier-refuses-manifest` vergleicht nur
den Exit-Code (`compare` = `message`) und bleibt grün.

- [ ] **Schritt 1: Failing Tests schreiben.** In `guard_test.go` neu:

```go
// A leading **/ stands for any directory, the root included, and is anchored
// at an element: my.loomux is no .loomux.
func TestMatchGlobReadsALeadingDoubleStarAsAnyDirectory(t *testing.T) {
	for _, row := range []struct {
		pattern, name string
		want          bool
	}{
		{"**/.loomux/config.toml", ".loomux/config.toml", true},
		{"**/.loomux/config.toml", "../sibling/.loomux/config.toml", true},
		{"**/.loomux/config.toml", "/repo/.loomux/config.toml", true},
		{"**/.loomux/config.toml", "C:/repo/.loomux/config.toml", true},
		{"**/.loomux/config.toml", "my.loomux/config.toml", false},
		{"**/.loomux/config.toml", ".loomux/config.toml.bak", false},
		{"**/.loomux/state/runs/**", "x/.loomux/state/runs", true},
		{"**/.loomux/state/runs/**", "x/.loomux/state/runs/0001.jsonl", true},
		{"**/.loomux/state/runs/**", ".loomux/state/runsx/0001.jsonl", false},
	} {
		got, err := matchGlob(row.pattern, row.name)
		if err != nil || got != row.want {
			t.Errorf("matchGlob(%q, %q) = %v, %v; want %v", row.pattern, row.name, got, err, row.want)
		}
	}
	if _, err := matchGlob("**/foo/*[x", "a/foo/b"); err == nil {
		t.Fatal("a bad class behind **/ must still be an error")
	}
}

// loomux's own files are kept under any directory: a sibling worktree's
// .loomux, or one named by its absolute path, like this project's.
func TestLoomuxsOwnRulesHoldUnderAnyDirectory(t *testing.T) {
	root := t.TempDir()
	for path, want := range map[string]string{
		"/repo/.loomux/config.toml":                          manifestReason,
		"../sibling/.loomux/config.toml":                     manifestReason,
		filepath.Join(t.TempDir(), ".loomux", "config.toml"): manifestReason,
		"../sibling/.loomux/state/runs/0001.jsonl":           runFilesWant,
		"../sibling/.loomux/state/hooks/stop.py":             "the stop gate's own controls are not written by the party it gates",
	} {
		if got := checkTool(root, "Write", map[string]any{"file_path": path}, config.Policy{}); len(got) != 1 || got[0] != want {
			t.Errorf("%s: reasons %q, want %q", path, got, want)
		}
	}
	for _, path := range []string{"my.loomux/config.toml", ".loomux/config.toml.bak", "docs/loomux/config.toml"} {
		if got := checkTool(root, "Write", map[string]any{"file_path": path}, config.Policy{}); len(got) != 0 {
			t.Errorf("%s: reasons %q, want none", path, got)
		}
	}
}
```

In `TestProtectedBuiltinPathsCarryTheirReason` zwei Zeilen ergänzen:

```go
		{"manifest", ".loomux/config.toml", manifestReason},
		{"manifest in capitals", ".LOOMUX/Config.toml", manifestReason},
```

In `guardflow_test.go`, `TestAWritingToolMayNotTouchABundledFlowFolder`: die
Zeile `"docs/.loomux/flows/example/flow.toml",` aus der erlaubten Liste
streichen und in die verweigerte Tabelle aufnehmen, dazu eine zweite:

```go
		{"docs/.loomux/flows/example/flow.toml", "example"},
		{"../sibling/.loomux/flows/review/flow.toml", "review"},
```

- [ ] **Schritt 2: Laufen lassen, Fehlschlag sehen.**
  Run: `go test ./internal/hooks/ -run 'TestMatchGlobReadsALeadingDoubleStar|TestLoomuxsOwnRules|TestProtectedBuiltinPaths|TestAWritingToolMayNotTouchABundledFlowFolder' -count=1`
  Expected: FAIL (`manifestReason` undefiniert).

- [ ] **Schritt 3: Implementieren.** In `guard.go` über `builtinPathRules`:

```go
// manifestReason refuses an agent the manifest, to a writing tool and to a
// shell line alike.
const manifestReason = ".loomux/config.toml: the manifest is where the barrier reads its own limits, so no agent may write it"
```

In `builtinPathRules` die drei `.loomux`-Zeilen ersetzen (die Kommentare zu
den zweiten Kopien bleiben stehen) und das Manifest aufnehmen:

```go
	{Match: []string{NoVerifyMarker}, Reason: "the stop gate's own controls are not written by the party it gates"},
	// loomux's own files under any directory: a write into a sibling
	// worktree's .loomux, or one named by its absolute path, is the same
	// write as into this project's.
	{Match: []string{"**/.loomux/config.toml"}, Reason: manifestReason},
	// The literal below is the second copy of sessions.StateDir; a rule is a
	// verbatim glob here, so the two are kept in step by hand.
	{Match: []string{"**/.loomux/state/hooks/**"}, Reason: "the stop gate's own controls are not written by the party it gates"},
	// The second copy of runs.Dir, kept in step by hand like the one above.
	{Match: []string{"**/.loomux/state/runs/**"}, Reason: runFilesReason},
```

`matchGlob` bekommt vor seinem bisherigen Körper:

```go
	// A leading **/ is any directory, the root included: the rest is tried
	// against the name and against every tail of it that starts after a
	// slash, so an element is matched whole.
	if rest, anywhere := strings.CutPrefix(pattern, "**/"); anywhere {
		for {
			if matched, err := matchGlob(rest, name); err != nil || matched {
				return matched, err
			}
			slash := strings.IndexByte(name, '/')
			if slash < 0 {
				return false, nil
			}
			name = name[slash+1:]
		}
	}
```

Den Doc-Kommentar von `matchGlob` um den Satz ergänzen: „A leading `**/`
stands for any directory, the root included.“ Zwei Kommentare, die das neue
Verhalten falsch machen, nachziehen: über `builtinPathRules` „Built-in rules
that protect secrets, stop gate controls, and lock files.“ → „Built-in rules
that protect secrets, the manifest, the stop gate's controls, the run files
and lock files.“; im Doc-Kommentar von `relativePath` den Satz „…; only the
rules without a slash, which are matched against the base name, still reach
it.“ ergänzen um „— and the rules under `**/`, which match loomux's own files
under any directory.“ Danach
`grep -rn "only slash-free\|only the rules without a slash\|nur Muster ohne" internal docs`
und jeden Treffer in `internal/` hier, jeden in `docs/` in Task 9 Schritt 6
bearbeiten.

In `guardflow.go` `flowFolderReasons` auf `flowFolderName` umstellen:

```go
// flowFolderReasons judges a write to rel against the protected flows: rel
// under a .loomux/flows folder, the project's or one under any directory, the
// way the built-in rules keep .loomux. The folder name is compared in lower
// case: Windows and macOS keep Example and example as one folder. The config
// is read only for such a path.
func flowFolderReasons(root, rel string) []string {
	name, under := flowFolderName(rel)
	if !under {
		return nil
	}
	protected, err := protectedFlows(root)
	if err != nil {
		return []string{unreadableFlowsReason(err)}
	}
	if at := slices.IndexFunc(protected, func(p string) bool { return strings.ToLower(p) == name }); at >= 0 {
		return []string{bundledFlowReason(protected[at])}
	}
	return nil
}

// flowFolderName is the element right below the innermost .loomux/flows
// folder in rel, in lower case, and whether rel lies below one at all.
func flowFolderName(rel string) (string, bool) {
	lower := "/" + strings.ToLower(rel)
	at := strings.LastIndex(lower, "/"+flowsDir+"/")
	if at < 0 {
		return "", false
	}
	name, _, _ := strings.Cut(lower[at+len(flowsDir)+2:], "/")
	return name, true
}
```

- [ ] **Schritt 4: Laufen lassen, grün sehen.**
  Run: `go test ./internal/hooks/ ./internal/cli/ -count=1`
  Expected: PASS. Hält ein Test in `internal/cli` den Wortlaut der Schranke
  („no writing tool may touch it“) für ein Edit des Manifests über
  `pre-tool-use` fest, ist das eine Kippung dieser Task: auf den
  Policy-Umschlag mit `manifestReason` umstellen und in der Akte nennen
  (beim Lesen am 2026-09-28 war keiner zu finden).

- [ ] **Schritt 5: Commit.**

```text
fix(guard): keep loomux's own files from writing tools under any directory

The built-in rules for the run files and the stop gate's state now match
under any directory with a leading **/, so a write into a sibling
worktree's .loomux is refused like one into this project's; the flow
folder rule reads a .loomux/flows folder the same way. The manifest gets
a built-in path rule of its own, so the policy refuses a writing tool on
it before the barrier is asked.
```

`git add internal/hooks/guard.go internal/hooks/guardflow.go internal/hooks/guard_test.go internal/hooks/guardflow_test.go`, dann `git commit -F <datei>`.

---

### Task 2: Pfadschreibweise — Braces, Globs, Stream-Anhang

**Files:**
- Create: `internal/hooks/pathspell.go`
- Test: `internal/hooks/pathspell_test.go`

**Interfaces:**
- Consumes: `relativePath(raw, root string) string` (guard.go).
- Produces: `const maxBraceVariants = 64`; `spellings(root, word string) ([]string, error)` — die relativen Pfade, die eine Shell aus einem Zielwort macht; `unfoldBraces(word string) ([]string, bool)`; `expandGlob(root, word string) []string`; `withoutStream(p string) string`; `isLetter(c byte) bool`; `letters(s string) bool`.

- [ ] **Schritt 1: Failing Tests schreiben** (`pathspell_test.go`):

```go
package hooks

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// mkfile makes a file under root, and the folders above it.
func mkfile(t *testing.T, root, rel string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestUnfoldBracesExpandsLikeBash(t *testing.T) {
	for word, want := range map[string][]string{
		"a/{b,c}/d":       {"a/b/d", "a/c/d"},
		"{a,b}{1,2}":      {"a1", "a2", "b1", "b2"},
		"x/{a,{b,c}}":     {"x/a", "x/b", "x/c"},
		"r{1..3}":         {"r1", "r2", "r3"},
		"r{3..1}":         {"r3", "r2", "r1"},
		"r{01..03}":       {"r01", "r02", "r03"},
		"r{1..9..4}":      {"r1", "r5", "r9"},
		"{a..c}":          {"a", "b", "c"},
		"${X}/{a,b}":      {"${X}/a", "${X}/b"},
		"{x}":             {"{x}"},
		"{}":              {"{}"},
		"open{a,b":        {"open{a,b"},
		"{a{b,c}":         {"{ab", "{ac"},
		"plain":           {"plain"},
		".loomux/{a,b}/x": {".loomux/a/x", ".loomux/b/x"},
	} {
		got, ok := unfoldBraces(word)
		if !ok || !slices.Equal(got, want) {
			t.Errorf("unfoldBraces(%q) = %q, %v; want %q", word, got, ok, want)
		}
	}
	if got, ok := unfoldBraces("{1..64}"); !ok || len(got) != 64 {
		t.Fatalf("64 variants: %d, %v", len(got), ok)
	}
	for _, word := range []string{"{1..65}", "{a,b}{a,b}{a,b}{a,b}{a,b}{a,b}{a,b}"} {
		if _, ok := unfoldBraces(word); ok {
			t.Errorf("%q unfolds past the limit", word)
		}
	}
	for _, word := range []string{"{1..x}", "{a..bb}", "{1..3..0}", "{1..3..x}", "{1..2..3..4}"} {
		if got, ok := unfoldBraces(word); !ok || !slices.Equal(got, []string{word}) {
			t.Errorf("%q is no range and stays: %q, %v", word, got, ok)
		}
	}
}

func TestWithoutStreamCutsAnNTFSStream(t *testing.T) {
	for p, want := range map[string]string{
		".loomux/config.toml:backup": ".loomux/config.toml",
		"C:/x/config.toml:s:$DATA":   "C:/x/config.toml",
		"C:/x/config.toml":           "C:/x/config.toml",
		"plain":                      "plain",
	} {
		if got := withoutStream(p); got != want {
			t.Errorf("withoutStream(%q) = %q, want %q", p, got, want)
		}
	}
}

// A glob is matched against the disk as bash matches it: a leading * or ?
// skips names that begin with a dot, and a glob that matches nothing stays.
func TestExpandGlobMatchesTheDiskLikeBash(t *testing.T) {
	root := t.TempDir()
	mkfile(t, root, ".loomux/state/runs/0001.jsonl")
	mkfile(t, root, "build/a.txt")
	mkfile(t, root, ".hidden/x")
	rel := func(ps []string) []string {
		var out []string
		for _, p := range ps {
			out = append(out, relativePath(p, root))
		}
		slices.Sort(out)
		return out
	}
	if got := rel(expandGlob(root, "*")); !slices.Equal(got, []string{"build"}) {
		t.Errorf("* = %q, want only build", got)
	}
	if got := rel(expandGlob(root, ".loomux/sta*/runs")); !slices.Equal(got, []string{".loomux/state/runs"}) {
		t.Errorf("sta* = %q", got)
	}
	if got := rel(expandGlob(root, ".*")); !slices.Contains(got, ".loomux") || !slices.Contains(got, ".hidden") {
		t.Errorf(".* = %q, want the dot folders", got)
	}
	if got := expandGlob(root, "nothing/*"); !slices.Equal(got, []string{"nothing/*"}) {
		t.Errorf("a glob without a match = %q, want it as written", got)
	}
	if got := expandGlob(root, "bad/["); !slices.Equal(got, []string{"bad/["}) {
		t.Errorf("a glob filepath.Match cannot read = %q, want it as written", got)
	}
	if got := expandGlob(root, filepath.Join(root, "build", "*")); len(got) != 1 {
		t.Errorf("an absolute glob = %q", got)
	}
	if got := expandGlob(root, "plain/x"); !slices.Equal(got, []string{"plain/x"}) {
		t.Errorf("no glob = %q", got)
	}
}

func TestSpellingsAreRelativeToTheRoot(t *testing.T) {
	root := t.TempDir()
	got, err := spellings(root, "./.loomux/flows/{example,zz}/x")
	if err != nil || !slices.Equal(got, []string{".loomux/flows/example/x", ".loomux/flows/zz/x"}) {
		t.Fatalf("braces: %q, %v", got, err)
	}
	got, err = spellings(root, filepath.Join(root, ".loomux", "config.toml")+":backup")
	if err != nil || !slices.Equal(got, []string{".loomux/config.toml"}) {
		t.Fatalf("stream: %q, %v", got, err)
	}
	_, err = spellings(root, "x{1..65}")
	if err == nil || !strings.Contains(err.Error(), strconv.Itoa(maxBraceVariants)) {
		t.Fatalf("past the limit: %v", err)
	}
}
```

- [ ] **Schritt 2: Laufen lassen.** Run: `go test ./internal/hooks/ -run 'TestUnfoldBraces|TestWithoutStream|TestExpandGlob|TestSpellings' -count=1` — Expected: FAIL (undefiniert).

- [ ] **Schritt 3: Implementieren** (`pathspell.go`):

```go
package hooks

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// maxBraceVariants is how many words one brace expansion may unfold to
// before the guard refuses rather than judge them one by one.
const maxBraceVariants = 64

// spellings are the paths a shell makes of one target word, each relative to
// root the way a rule spells a path: braces unfolded, a stream name cut off,
// globs matched against the disk. PowerShell unfolds no braces, and bash none
// inside quotes; reading them anyway only ever refuses more. An error is a
// refusal the caller names.
func spellings(root, word string) ([]string, error) {
	words, ok := unfoldBraces(word)
	if !ok {
		return nil, fmt.Errorf("loomux does not unfold more than %d brace variants of %q, so it refuses", maxBraceVariants, word)
	}
	var out []string
	for _, w := range words {
		for _, p := range expandGlob(root, withoutStream(w)) {
			out = append(out, relativePath(p, root))
		}
	}
	return out, nil
}

// unfoldBraces is bash's brace expansion of word: {a,b}, {1..3}, {01..03},
// {1..9..2} and {a..c}, nested and in sequence, never a ${…}. A brace group
// without a comma or a range, or one left open, stays as written. ok is false
// past maxBraceVariants.
func unfoldBraces(word string) ([]string, bool) {
	out := []string{word}
	for i := 0; i < len(out); {
		next, unfolded := unfoldFirst(out[i])
		if !unfolded {
			i++
			continue
		}
		out = append(out[:i], append(next, out[i+1:]...)...)
		if len(out) > maxBraceVariants {
			return nil, false
		}
	}
	return out, true
}

// unfoldFirst unfolds the first brace group of word that expands.
func unfoldFirst(word string) ([]string, bool) {
	for open := 0; open < len(word); open++ {
		if word[open] != '{' || open > 0 && word[open-1] == '$' {
			continue
		}
		end, commas := braceGroup(word, open)
		if end < 0 {
			continue
		}
		var alternatives []string
		if len(commas) > 0 {
			last := open + 1
			for _, c := range commas {
				alternatives = append(alternatives, word[last:c])
				last = c + 1
			}
			alternatives = append(alternatives, word[last:end])
		} else if sequence, ok := braceRange(word[open+1 : end]); ok {
			alternatives = sequence
		} else {
			continue
		}
		out := make([]string, len(alternatives))
		for i, alternative := range alternatives {
			out[i] = word[:open] + alternative + word[end+1:]
		}
		return out, true
	}
	return nil, false
}

// braceGroup is the index of the } that closes the { at open, or -1, and the
// commas at the group's own depth.
func braceGroup(word string, open int) (int, []int) {
	depth := 0
	var commas []int
	for i := open; i < len(word); i++ {
		switch word[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, commas
			}
		case ',':
			if depth == 1 {
				commas = append(commas, i)
			}
		}
	}
	return -1, nil
}

// braceRange is the sequence a range stands for: numbers, zero-padded when
// either end is, or single letters, with an optional step. A range longer
// than maxBraceVariants stops one past it, which the caller refuses.
func braceRange(inner string) ([]string, bool) {
	parts := strings.Split(inner, "..")
	if len(parts) < 2 || len(parts) > 3 {
		return nil, false
	}
	step := 1
	if len(parts) == 3 {
		s, err := strconv.Atoi(parts[2])
		if err != nil || s == 0 {
			return nil, false
		}
		if s < 0 {
			s = -s
		}
		step = s
	}
	from, errFrom := strconv.Atoi(parts[0])
	to, errTo := strconv.Atoi(parts[1])
	isLetters := errFrom != nil || errTo != nil
	width := 0
	if isLetters {
		if len(parts[0]) != 1 || len(parts[1]) != 1 || !isLetter(parts[0][0]) || !isLetter(parts[1][0]) {
			return nil, false
		}
		from, to = int(parts[0][0]), int(parts[1][0])
	} else if padded(parts[0]) || padded(parts[1]) {
		width = max(len(parts[0]), len(parts[1]))
	}
	direction := 1
	if to < from {
		direction = -1
	}
	var out []string
	for v := from; (v-to)*direction <= 0 && len(out) <= maxBraceVariants; v += direction * step {
		if isLetters {
			out = append(out, string(rune(v)))
		} else {
			out = append(out, fmt.Sprintf("%0*d", width, v))
		}
	}
	return out, true
}

// padded says whether a range end is written with leading zeros.
func padded(end string) bool {
	return len(end) > 1 && end[0] == '0'
}

// withoutStream cuts an NTFS stream name off a path -- x.toml:backup writes
// x.toml -- after the colon of a drive letter.
func withoutStream(p string) string {
	start := 0
	if len(p) >= 2 && p[1] == ':' && isLetter(p[0]) {
		start = 2
	}
	if colon := strings.IndexByte(p[start:], ':'); colon >= 0 {
		return p[:start+colon]
	}
	return p
}

// expandGlob matches word against the disk under root as bash does: a *, ?
// or [ in a name, and a name that begins with a dot only for a pattern
// element that does too. A pattern that matches nothing, or one filepath.Glob
// cannot read, stays as written, as bash leaves it.
func expandGlob(root, word string) []string {
	if !strings.ContainsAny(word, "*?[") {
		return []string{word}
	}
	pattern := word
	if !filepath.IsAbs(pattern) {
		pattern = filepath.Join(root, pattern)
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return []string{word}
	}
	var out []string
	for _, m := range matches {
		if !hidesDotNames(pattern, m) {
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		return []string{word}
	}
	return out
}

// hidesDotNames says whether bash would skip match for pattern: one of its
// names begins with a dot where the pattern element does not.
func hidesDotNames(pattern, match string) bool {
	want := strings.Split(filepath.ToSlash(filepath.Clean(pattern)), "/")
	got := strings.Split(filepath.ToSlash(match), "/")
	if len(want) != len(got) {
		return false
	}
	for i, name := range got {
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(want[i], ".") {
			return true
		}
	}
	return false
}

// isLetter says whether c is an ASCII letter.
func isLetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// letters says whether s is one or more ASCII letters.
func letters(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isLetter(s[i]) {
			return false
		}
	}
	return s != ""
}
```

  Deckt der Test `hidesDotNames` mit ungleicher Länge nicht ab (Glob liefert
  immer gleich viele Elemente), die Zeile `if len(want) != len(got)` streichen
  statt sie auszunehmen. `letters` wird erst in Task 3 gerufen; bis dahin im
  Test `letters("")` und `letters("a1")` einmal aufrufen, damit die Abdeckung
  hält:

```go
func TestLettersAreASCIILettersOnly(t *testing.T) {
	if !letters("sSLo") || letters("") || letters("a1") {
		t.Fatal("letters")
	}
}
```

- [ ] **Schritt 4: Laufen lassen.** Run: `go test ./internal/hooks/ -run 'TestUnfoldBraces|TestWithoutStream|TestExpandGlob|TestSpellings|TestLetters' -count=1` — Expected: PASS. Dann `go run ./cmd/loomux check coverage` und prüfen, dass keine Funktion aus `pathspell.go` unter 100 % liegt.

- [ ] **Schritt 5: Commit.**

```text
feat(guard): spell a shell target the way the shell expands it

spellings unfolds brace expansion (lists, ranges, steps, zero padding,
at most 64 variants), cuts an NTFS stream name and matches globs against
the disk with bash's rule for leading dots, then names each path relative
to the root. Nothing calls it yet.
```

---

### Task 3: Leser für Shell-Schreibziele und Wrapper-Schalter mit Wert

**Files:**
- Create: `internal/hooks/shellwrites.go`
- Modify: `internal/hooks/guard.go` (`dropPrefixes`, neu `wrapperFlags`, `wrapperTakesValue`; Doc-Kommentare von `readings` und `dropPrefixes`)
- Modify: `docs/en/cli-reference.md` (ca. Z. 298–301), `docs/de/cli-reference.md` (ca. Z. 314–317)
- Test: `internal/hooks/shellwrites_test.go`, `internal/hooks/guard_test.go`

**Interfaces:**
- Consumes: `lineVariants`, `splitSegments`, `readings`, `tolerantWords`, `dropPrefixes`, `baseName`, `loomuxArgs` (guard.go); `letters`, `isLetter` (Task 2).
- Produces:
  - `type shellTarget struct { path string; removes bool; shell bool }`
  - `shellWrites(root, line string) (targets []shellTarget, unknown [][]string)` — `unknown` hält je Lesung eines vertrauenswürdigen Segments mit unbekanntem Programm dessen Programm und Argumente.
  - Listen `everyFileWrites`, `everyFileRemoves`, `moveVerbs`, `renameVerbs`, `copyVerbs`, `readVerbs`, `changeDirectory` (`[]string`).
  - `verbOf(program string) string`, `positional(args []string) []string`, `psValues(args []string, name string) []string`, `posixValues(args []string, short, long string) []string`.

Kippungen dieser Task: die drei Lücken „Wrapper flags that take a value“ in
`TestTheGuardRefusesCommandsThatWriteTheConfiguration` (`sudo -u root loomux
init`, `xargs -n 1 loomux init`, `timeout -s KILL 60 loomux init`) werden
verweigert und wandern in `loomuxSpellings()`, damit auch die Torantwort sie
prüft.

- [ ] **Schritt 1: Failing Tests schreiben** (`shellwrites_test.go`):

```go
package hooks

import (
	"slices"
	"strings"
	"testing"
)

// spelled is how a test reads targets: w: for a write, rm: for a removal, the
// path with forward slashes, sorted and each once.
func spelled(targets []shellTarget) []string {
	var out []string
	for _, t := range targets {
		kind := "w:"
		if t.removes {
			kind = "rm:"
		}
		out = append(out, kind+strings.ReplaceAll(t.path, `\`, "/"))
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// Every verb of the table names the target it writes or removes. A row holds
// what the line must yield among its targets; the readings may add others
// (a cmd switch, a script word), which only ever refuse more.
func TestShellWritesReadsTheTargetsOfEveryVerb(t *testing.T) {
	root := t.TempDir()
	mkfile(t, root, "notes.txt")
	for line, want := range map[string][]string{
		"echo x > .loomux/config.toml":                                      {"w:.loomux/config.toml"},
		"echo x>.loomux/config.toml":                                        {"w:.loomux/config.toml"},
		"echo x &> .loomux/config.toml":                                     {"w:.loomux/config.toml"},
		"echo x 2>>.loomux/config.toml":                                     {"w:.loomux/config.toml"},
		"'x' *> .loomux/config.toml":                                        {"w:.loomux/config.toml"},
		"tee -a .loomux/config.toml":                                        {"w:.loomux/config.toml"},
		"Set-Content -Path .loomux/config.toml -Value x":                    {"w:.loomux/config.toml"},
		`Out-File -FilePath:.loomux\config.toml`:                            {"w:.loomux/config.toml"},
		"touch .loomux/state/runs/0001.flow":                                {"w:.loomux/state/runs/0001.flow"},
		"rm -rf .loomux":                                                    {"rm:.loomux"},
		"rm -- -x .loomux/config.toml":                                      {"rm:-x", "rm:.loomux/config.toml"},
		`rmdir /s /q .loomux\state\runs`:                                    {"rm:.loomux/state/runs"},
		"mv .loomux/flows elsewhere":                                        {"rm:.loomux/flows", "w:elsewhere"},
		"Move-Item x -Destination .loomux/config.toml":                      {"rm:x", "w:.loomux/config.toml"},
		`Rename-Item .loomux\x config.toml`:                                 {"rm:.loomux/x", "w:.loomux/config.toml"},
		"cp -r mine .loomux/flows/":                                         {"w:.loomux/flows/"},
		"cp -t .loomux/flows/example x":                                     {"w:.loomux/flows/example"},
		`Copy-Item x.toml -Dest .loomux\config.toml`:                        {"w:.loomux/config.toml"},
		"install -m 644 x .loomux/config.toml":                              {"w:.loomux/config.toml"},
		"rsync -a src/ .loomux/flows/example/":                              {"w:.loomux/flows/example/"},
		"ln -s evil .loomux/config.toml":                                    {"w:.loomux/config.toml"},
		"dd if=x of=.loomux/config.toml":                                    {"w:.loomux/config.toml"},
		"tar -xf evil.tar -C .loomux/flows/example":                         {"w:.loomux/flows/example"},
		"tar --extract --file evil.tar --directory=.loomux/flows/example":   {"w:.loomux/flows/example"},
		"tar -czf .loomux/config.toml src":                                  {"w:.loomux/config.toml"},
		"tar cf .loomux/config.toml src":                                    {"w:.loomux/config.toml"},
		"tar --create --file=.loomux/config.toml src":                       {"w:.loomux/config.toml"},
		"unzip evil.zip -d .loomux/flows/example":                           {"w:.loomux/flows/example"},
		`Expand-Archive evil.zip -DestinationPath .loomux\flows\example`:    {"w:.loomux/flows/example"},
		`robocopy evil .loomux\flows\example /MIR`:                          {"w:.loomux/flows/example"},
		`xcopy evil .loomux\flows\example /E /I`:                            {"w:.loomux/flows/example"},
		"New-Item -Path .loomux/config.toml -Force":                         {"w:.loomux/config.toml"},
		"New-Item -Path .loomux -Name config.toml":                          {"w:.loomux/config.toml"},
		"ni .loomux/state/runs/0002.flow":                                   {"w:.loomux/state/runs/0002.flow"},
		"curl -o .loomux/config.toml https://example.invalid/x":             {"w:.loomux/config.toml"},
		"curl -sSLo .loomux/config.toml https://example.invalid/x":          {"w:.loomux/config.toml"},
		"curl --output=.loomux/config.toml https://example.invalid/x":       {"w:.loomux/config.toml"},
		"wget -O .loomux/config.toml https://example.invalid/x":             {"w:.loomux/config.toml"},
		"Invoke-WebRequest https://example.invalid/x -OutFile .loomux/c.t":  {"w:.loomux/c.t"},
		"find .loomux/state -name '*.jsonl' -delete":                        {"rm:.loomux/state"},
		"find -delete":                                                      {"rm:."},
		"find .loomux/flows/example -exec rm {} +":                          {"rm:.loomux/flows/example"},
		"sed -i s/a/b/ .loomux/config.toml":                                 {"w:.loomux/config.toml"},
		"sed 's/a/b/' .loomux/config.toml -i":                               {"w:.loomux/config.toml"},
		"sed -i.bak -e s/a/b/ .loomux/config.toml":                          {"w:.loomux/config.toml"},
		"sed --in-place --expression=s/a/b/ .loomux/config.toml":            {"w:.loomux/config.toml"},
		"sed --in-place --file script.sed .loomux/config.toml":              {"w:.loomux/config.toml"},
		"perl -pi -e 's/a/b/' .loomux/config.toml":                          {"w:.loomux/config.toml"},
		"sed -i -- s/a/b/ .loomux/config.toml":                              {"w:.loomux/config.toml"},
		"git mv .loomux/flows/example x":                                    {"rm:.loomux/flows/example", "w:x"},
		"git -C . rm -r .loomux/state":                                      {"rm:.loomux/state"},
		"git checkout -- .loomux/config.toml":                               {"w:.loomux/config.toml"},
		"git checkout HEAD notes.txt":                                       {"w:notes.txt"},
		"git restore --staged --worktree notes.txt":                         {"w:notes.txt"},
		"git clean -fdx":                                                    {"rm:."},
		"git clean -fd -e keep .loomux/flows/example":                       {"rm:.loomux/flows/example"},
		"git clean -f -- .loomux/flows/example":                             {"rm:.loomux/flows/example"},
		"[IO.File]::WriteAllText('.loomux/config.toml', 'x')":               {"w:.loomux/config.toml"},
		`[IO.Directory]::Delete('.loomux\state\runs', $true)`:               {"rm:.loomux/state/runs"},
		"[System.IO.File]::Copy('x', \".loomux/config.toml\")":              {"w:.loomux/config.toml"},
		"[IO.Directory]::CreateDirectory('.loomux/flows/example/x')":        {"w:.loomux/flows/example/x"},
		"sudo -u root tee .loomux/config.toml":                              {"w:.loomux/config.toml"},
		"xargs -n 1 rm .loomux/config.toml":                                 {"rm:.loomux/config.toml"},
		"timeout -s KILL 60 rm .loomux/config.toml":                         {"rm:.loomux/config.toml"},
		"env -u HOME rm .loomux/config.toml":                                {"rm:.loomux/config.toml"},
		"cd .loomux && rm config.toml":                                      {"rm:.loomux/config.toml"},
		"cd .loomux; cd flows; rm -r example":                               {"rm:.loomux/flows/example"},
		`Set-Location .loomux\flows; Remove-Item example`:                   {"rm:.loomux/flows/example"},
		"pushd /repo/.loomux && rm config.toml":                             {"rm:/repo/.loomux/config.toml"},
		"cd .loomux && popd && rm config.toml":                              {"rm:config.toml"},
		"echo 'x .loomux/config.toml":                                       {"w:.loomux/config.toml"},
	} {
		targets, _ := shellWrites(root, line)
		got := spelled(targets)
		for _, w := range want {
			if !slices.Contains(got, w) {
				t.Errorf("%q: targets %q, want %q among them", line, got, w)
			}
		}
	}
}

// What only reads, and what writes nowhere, yields no target at all.
func TestShellWritesFindsNoTargetInAReadingLine(t *testing.T) {
	root := t.TempDir()
	for _, line := range []string{
		"cat .loomux/config.toml",
		"ls .loomux/flows",
		"sed -n '1,5p' .loomux/config.toml",
		"find . -name x",
		"git status",
		"git checkout feat/x",
		"git checkout -b x",
		"git checkout --orphan x",
		"git restore --staged x",
		"git restore -S x",
		"git clean -n",
		"git clean --dry-run -d",
		"git clean -nd",
		"echo x 2>&1",
		"echo hi >&2",
		"cat < in.txt",
		"cat <<'EOF'",
		`git commit -m "fix(guard): keep .loomux/state/runs"`,
		"robocopy onlyone",
		"loomux flow list",
		"tar -xf evil.tar",
		"echo x > ''",
	} {
		if targets, _ := shellWrites(root, line); len(targets) != 0 {
			t.Errorf("%q: targets %q, want none", line, spelled(targets))
		}
	}
	// A copy into the folder above a kept one writes that folder, it removes nothing.
	targets, _ := shellWrites(root, "cp -r mine .loomux/flows/")
	if got := spelled(targets); !slices.Equal(got, []string{"w:.loomux/flows/"}) {
		t.Fatalf("cp into the folder above: %q", got)
	}
}

// unknown names each program of a trusted segment the table does not know,
// with its arguments; a quote-blind fragment of quoted text is never one.
func TestShellWritesNamesTheProgramsItDoesNotKnow(t *testing.T) {
	root := t.TempDir()
	_, unknown := shellWrites(root, "frob .loomux/config.toml")
	if !slices.ContainsFunc(unknown, func(args []string) bool {
		return slices.Equal(args, []string{"frob", ".loomux/config.toml"})
	}) {
		t.Fatalf("frob: unknown %q", unknown)
	}
	for _, line := range []string{"git stash push x", "loomux flow run x", `sh -c "rm x"`, "go vet ."} {
		if _, unknown := shellWrites(root, line); len(unknown) == 0 {
			t.Errorf("%q: nothing unknown", line)
		}
	}
	for _, line := range []string{
		"cat .loomux/config.toml", "git status", "git diff x", "loomux flow list", "loomux flow show 1",
		"go run ./cmd/loomux config get a", "loomux config list", "loomux config proposals", "loomux check precommit",
		"rm x", "echo hi", "X=1", "cd x", "find . -name y",
		`git commit -m "fix(guard): keep .loomux/state/runs"`,
		`git commit -m "a; frob .loomux/config.toml"`,
	} {
		if _, unknown := shellWrites(root, line); len(unknown) != 0 {
			t.Errorf("%q: unknown %q, want none", line, unknown)
		}
	}
}
```

In `guard_test.go`, `loomuxSpellings()`: nach dem Block „Wrappers without flags, …“ ergänzen

```go
		// Wrapper flags that take a value.
		"sudo -u root loomux init",
		"xargs -n 1 loomux init",
		"timeout -s KILL 60 loomux init",
		"env -u HOME loomux init",
```

und in `TestTheGuardRefusesCommandsThatWriteTheConfiguration` die drei Zeilen
samt Kommentar „// Wrapper flags that take a value.“ aus `holes` streichen.

- [ ] **Schritt 2: Laufen lassen.** Run: `go test ./internal/hooks/ -run 'TestShellWrites|TestTheGuardRefusesCommandsThatWriteTheConfiguration|TestAnAgentMayNotAnswerAGateInAnySpelling' -count=1` — Expected: FAIL (undefiniert; die Wrapper-Zeilen gehen durch).

- [ ] **Schritt 3a: `dropPrefixes` lernt Schalter mit Wert.** In `guard.go` die
  Fälle `sudo … xargs` und `timeout` ersetzen:

```go
		case base == "sudo" || base == "env" || base == "xargs":
			n += wrapperFlags(base, words[1:])
		case base == "command" || base == "exec" || base == "nohup" || base == "time":
			n += flagCount(words[1:])
```

```go
		case base == "timeout":
			// The duration comes before the program.
			n += wrapperFlags(base, words[1:]) + 1
```

und darunter:

```go
// wrapperFlags is how many words at the head of words are the wrapper's
// flags, a flag's separate value included; -- ends them and counts.
func wrapperFlags(wrapper string, words []string) int {
	n := 0
	for n < len(words) && len(words[n]) > 1 && words[n][0] == '-' {
		if words[n] == "--" {
			return n + 1
		}
		if wrapperTakesValue(wrapper, words[n]) {
			n++
		}
		n++
	}
	return min(n, len(words))
}

// wrapperTakesValue names the flags of sudo, env, xargs and timeout whose
// value is the next word.
func wrapperTakesValue(wrapper, flag string) bool {
	switch wrapper {
	case "sudo":
		return slices.Contains([]string{"-u", "-g", "-C", "-D", "-h", "-p", "-r", "-t", "-U", "-T",
			"--user", "--group", "--close-from", "--chdir", "--host", "--prompt", "--role", "--type",
			"--other-user", "--command-timeout"}, flag)
	case "env":
		return slices.Contains([]string{"-u", "-C", "-S", "--unset", "--chdir", "--split-string"}, flag)
	case "xargs":
		return slices.Contains([]string{"-n", "-L", "-P", "-s", "-I", "-d", "-E", "-a",
			"--max-args", "--max-lines", "--max-procs", "--max-chars", "--delimiter", "--arg-file"}, flag)
	}
	return slices.Contains([]string{"-s", "-k", "--signal", "--kill-after"}, flag)
}
```

  Doc-Kommentar von `dropPrefixes`: „with their flags, as long as a flag
  carries no separate value“ → „with their flags, and the separate value of
  the flags wrapperTakesValue names“. In `readings`: „a wrapper flag with a
  separate value (sudo -u root),“ aus „Not guaranteed“ streichen. In
  `docs/en/cli-reference.md` „a wrapper flag with a separate value (`sudo -u
  root loomux init`, `xargs -n 1 …`, `timeout -s KILL 60 …`);“ streichen, in
  `docs/de/cli-reference.md` „ein Wrapper-Flag mit eigenem Wert (`sudo -u root
  loomux init`, `xargs -n 1 …`, `timeout -s KILL 60 …`);“ streichen.

- [ ] **Schritt 3b: Leser schreiben** (`shellwrites.go`):

```go
package hooks

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/shellwords"
)

// shellTarget is one path a call writes or removes, as the call spells it.
type shellTarget struct {
	path    string
	removes bool // a removal, or the source of a move: what lies below it counts too
	shell   bool // spelled by a shell: braces and globs are still to unfold
}

// The verb table, by base name in lower case without .exe (verbOf).
var (
	// everyFileWrites write every file they name.
	everyFileWrites = []string{"tee", "tee-object", "set-content", "add-content", "ac", "out-file",
		"clear-content", "clc", "truncate", "touch"}
	// everyFileRemoves remove every file or folder they name, a folder with all it holds.
	everyFileRemoves = []string{"rm", "del", "erase", "remove-item", "ri", "rd", "rmdir", "unlink", "shred"}
	// moveVerbs remove their sources and write their destination.
	moveVerbs = []string{"mv", "move", "move-item", "mi"}
	// renameVerbs remove the item and write its new name in the same folder.
	renameVerbs = []string{"rename-item", "ren", "rni", "rename"}
	// copyVerbs write only their destination.
	copyVerbs = []string{"cp", "copy", "copy-item", "cpi", "install", "rsync", "ln"}
	// readVerbs read what they name, or only name it.
	readVerbs = []string{"cat", "type", "get-content", "gc", "less", "more", "head", "tail", "wc",
		"stat", "file", "jq", "ls", "dir", "get-childitem", "gci", "grep", "rg", "select-string",
		"sls", "diff", "echo", "printf", "write-output", "cd", "set-location", "sl", "pushd",
		"popd", "test-path"}
	// changeDirectory move the place relative paths after them start from.
	changeDirectory = []string{"cd", "set-location", "sl", "pushd"}
)

// shellWrites reads a shell line for the paths it writes or removes, by a
// table of verbs rather than an expression per path. unknown holds, for every
// reading of a trusted segment whose program is in neither the table nor
// readVerbs, that program and its arguments; strict mode judges those.
//
// Every variant of the line (lineVariants) and both cuts of each
// (splitSegments) are read, and every reading adds targets, never removes
// one: a misread quote costs a false refusal, not a pass. The quote-blind cut
// also breaks inside quoted text, so the guesses for a segment no strict split
// reads and the unknown programs come from the quote-aware cut alone: a
// commit message that names .loomux/state/runs behind a parenthesis would
// otherwise be refused. Within a cut a cd, Set-Location or pushd moves the
// place later relative targets start from; popd goes back to the root.
func shellWrites(root, line string) (targets []shellTarget, unknown [][]string) {
	for _, variant := range lineVariants(line) {
		targets = append(targets, dotNetTargets(variant)...)
		cuts := [][]string{splitSegments(variant, false)}
		if strings.ContainsAny(variant, `"'`) {
			cuts = append(cuts, splitSegments(variant, true))
		}
		for n, cut := range cuts {
			// A line without quotes cuts the same both ways.
			trusted := n == 1 || len(cuts) == 1
			base := ""
			for _, segment := range cut {
				if _, err := shellwords.Split(segment); err != nil && trusted {
					targets = append(targets, under(base, guessedTargets(segment))...)
				}
				all := readings(segment)
				var place []string
				for i, words := range all {
					found, args, known := segmentWrites(joinPlace(root, base), words)
					targets = append(targets, under(base, found)...)
					if !known && trusted {
						unknown = append(unknown, args)
					}
					// The tolerant reading keeps a PowerShell path whole.
					if i == len(all)-2 {
						place = args
					}
				}
				base = changedDirectory(base, place)
			}
		}
	}
	return targets, unknown
}

// segmentWrites is what one reading of one segment writes: its redirections
// wherever they stand, then what its program does to its arguments. args are
// the program and its arguments without prefixes and redirections; known says
// whether the program is in the verb table or readVerbs. dir is where
// relative paths start, for git checkout's question whether a path exists.
func segmentWrites(dir string, words []string) (targets []shellTarget, args []string, known bool) {
	for i := 0; i < len(words); i++ {
		w := words[i]
		redirect, writes, bare, target := redirectTarget(w)
		if !redirect {
			op := strings.IndexByte(w, '>')
			if op < 0 {
				args = append(args, w)
				continue
			}
			// echo x>f: the redirection is glued to the word before it.
			args = append(args, w[:op])
			_, writes, bare, target = redirectTarget(w[op:])
		}
		if bare && i+1 < len(words) {
			i++
			if writes {
				target = words[i]
			}
		}
		if target = strings.TrimSpace(target); target != "" {
			targets = append(targets, shellTarget{path: target, shell: true})
		}
	}
	args = dropPrefixes(args)
	if len(args) == 0 {
		return targets, nil, true
	}
	found, known := verbWrites(dir, args)
	return append(targets, found...), args, known
}

// redirectTarget reads w as a redirection: whether it is one, whether it
// writes, whether its target is the next word, and the target glued to it.
// > >> >| 2> &> and PowerShell's *> write; < and << read; a target that
// begins with & is a descriptor (2>&1, >&2).
func redirectTarget(w string) (redirect, writes, bare bool, target string) {
	i := 0
	for i < len(w) && (w[i] >= '0' && w[i] <= '9' || w[i] == '&' || w[i] == '*') {
		i++
	}
	if i == len(w) || w[i] != '<' && w[i] != '>' {
		return false, false, false, ""
	}
	writes = w[i] == '>'
	for i < len(w) && strings.IndexByte("<>|!", w[i]) >= 0 {
		i++
	}
	rest := w[i:]
	if strings.HasPrefix(rest, "&") {
		return true, false, false, ""
	}
	if !writes {
		return true, false, rest == "", ""
	}
	return true, true, rest == "", rest
}

// verbWrites is what a program does to its arguments, by the verb table.
func verbWrites(dir string, args []string) ([]shellTarget, bool) {
	verb := verbOf(args[0])
	rest := args[1:]
	switch {
	case slices.Contains(everyFileWrites, verb):
		return targetsOf(positional(rest), false), true
	case slices.Contains(everyFileRemoves, verb):
		return targetsOf(positional(rest), true), true
	case slices.Contains(moveVerbs, verb):
		return moved(rest), true
	case slices.Contains(renameVerbs, verb):
		return renamed(rest), true
	case slices.Contains(copyVerbs, verb):
		// rsync's -t keeps times; cp, install and ln take a target folder with it.
		return destination(rest, verb != "rsync"), true
	}
	switch verb {
	case "dd":
		var out []string
		for _, a := range rest {
			if value, ok := strings.CutPrefix(a, "of="); ok {
				out = append(out, value)
			}
		}
		return targetsOf(out, false), true
	case "tar":
		return tarWrites(rest), true
	case "unzip":
		return targetsOf(posixValues(rest, "-d", ""), false), true
	case "expand-archive":
		return targetsOf(psValues(rest, "destinationpath"), false), true
	case "robocopy", "xcopy":
		if words := positional(rest); len(words) > 1 {
			return targetsOf(words[1:2], false), true
		}
		return nil, true
	case "new-item", "ni":
		return newItem(rest), true
	case "curl", "wget", "invoke-webrequest", "iwr":
		return downloads(rest), true
	case "find":
		return findRemoves(rest), true
	case "sed":
		return inPlace(rest, "ef"), true
	case "perl":
		return inPlace(rest, "eE"), true
	case "git":
		return gitWrites(dir, rest)
	}
	return nil, reads(args)
}

// verbOf is a program's name as the verb table spells it: its base name in
// lower case, without .exe.
func verbOf(program string) string {
	return strings.TrimSuffix(baseName(program), ".exe")
}

// targetsOf makes shell targets of paths, skipping empty words.
func targetsOf(paths []string, removes bool) []shellTarget {
	var out []shellTarget
	for _, p := range paths {
		if p != "" {
			out = append(out, shellTarget{path: p, removes: removes, shell: true})
		}
	}
	return out
}

// positional are the arguments that are neither a flag (-x, --x) nor a cmd
// switch (/s, /E:ON), with the value of a PowerShell parameter glued after a
// colon (-FilePath:x) among them; after -- every word counts, one that begins
// with - too. A flag's separate value is positional as well: the guard does
// not know which flags take one, and a wrong path only ever refuses.
func positional(args []string) []string {
	var out []string
	for i, a := range args {
		switch {
		case a == "--":
			return append(out, args[i+1:]...)
		case strings.HasPrefix(a, "-"):
			if _, value, glued := strings.Cut(a, ":"); glued && value != "" {
				out = append(out, value)
			}
		case cmdSwitch(a):
		default:
			out = append(out, a)
		}
	}
	return out
}

// cmdSwitch says whether a is a switch of a cmd program (/s, /Q, /E:ON): a
// slash and one element, which no path the rules keep ever is.
func cmdSwitch(a string) bool {
	return len(a) > 1 && a[0] == '/' && !strings.ContainsAny(a[1:], `/\`)
}

// psValues are the values of the PowerShell parameter name, in any case and
// abbreviated to no fewer than three letters, glued after a colon or as the
// next word.
func psValues(args []string, name string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) < 4 || a[0] != '-' || a[1] == '-' {
			continue
		}
		spelled, value, glued := strings.Cut(a[1:], ":")
		if len(spelled) < 3 || !strings.HasPrefix(name, strings.ToLower(spelled)) {
			continue
		}
		if glued {
			out = append(out, value)
		} else if i+1 < len(args) {
			i++
			out = append(out, args[i])
		}
	}
	return out
}

// posixValues are the values of a POSIX flag: -o f, -of, a bundle ending in
// the letter (-sSLo f), --output f and --output=f. Either spelling may be
// empty; -- ends the flags.
func posixValues(args []string, short, long string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		separate := false
		switch {
		case a == "--":
			return out
		case short != "" && a == short || long != "" && a == long:
			separate = true
		case long != "" && strings.HasPrefix(a, long+"="):
			out = append(out, a[len(long)+1:])
		case short == "" || strings.HasPrefix(a, "--") || len(a) < 3 || a[0] != '-':
		case a[1] == short[1]:
			out = append(out, a[2:])
		case letters(a[1:]) && a[len(a)-1] == short[1]:
			separate = true
		}
		if separate && i+1 < len(args) {
			i++
			out = append(out, args[i])
		}
	}
	return out
}

// moved is a move: the destination -- -Destination, or the last positional
// argument -- is written, and every other positional argument, a source, is
// removed.
func moved(args []string) []shellTarget {
	words := positional(args)
	dest := psValues(args, "destination")
	if len(dest) == 0 && len(words) > 1 {
		dest = words[len(words)-1:]
	}
	var sources []string
	for _, w := range words {
		if !slices.Contains(dest, w) {
			sources = append(sources, w)
		}
	}
	return append(targetsOf(sources, true), targetsOf(dest, false)...)
}

// renamed is Rename-Item and ren: the item is removed, and its new name,
// which lies in the item's folder, written.
func renamed(args []string) []shellTarget {
	items := psValues(args, "path")
	names := psValues(args, "newname")
	for _, w := range positional(args) {
		switch {
		case slices.Contains(items, w) || slices.Contains(names, w):
		case len(items) == 0:
			items = append(items, w)
		case len(names) == 0:
			names = append(names, w)
		}
	}
	out := targetsOf(items, true)
	for _, item := range items {
		for _, name := range names {
			out = append(out, shellTarget{path: path.Join(path.Dir(filepath.ToSlash(item)), name), shell: true})
		}
	}
	return out
}

// destination is where a copy lands: -Destination (PowerShell), -t or
// --target-directory where the verb takes one, else the last positional
// argument.
func destination(args []string, targetFlag bool) []shellTarget {
	dest := psValues(args, "destination")
	if targetFlag {
		dest = append(dest, posixValues(args, "-t", "--target-directory")...)
	}
	if words := positional(args); len(dest) == 0 && len(words) > 0 {
		dest = words[len(words)-1:]
	}
	return targetsOf(dest, false)
}

// tarWrites are where tar writes: the folder it extracts into (-C,
// --directory), and when it creates -- a c in a flag word or in its first,
// dashless word, or --create -- the archive (-f, --file, an f ending a
// bundle). What an archive holds is unknown, so an extraction into the
// working folder without -C writes nothing the guard can name.
func tarWrites(args []string) []shellTarget {
	out := posixValues(args, "-C", "--directory")
	creates := slices.Contains(args, "--create")
	for i, a := range args {
		bundle, dashed := strings.CutPrefix(a, "-")
		if !strings.HasPrefix(a, "--") && letters(bundle) && (dashed || i == 0) && strings.ContainsRune(bundle, 'c') {
			creates = true
		}
	}
	if creates {
		out = append(out, posixValues(args, "-f", "--file")...)
		if len(args) > 1 && !strings.HasPrefix(args[0], "-") && letters(args[0]) && strings.HasSuffix(args[0], "f") {
			out = append(out, args[1])
		}
	}
	return targetsOf(out, false)
}

// newItem is where New-Item creates: -Path, or every positional argument,
// each alone and joined with -Name when one is given.
func newItem(args []string) []shellTarget {
	paths := psValues(args, "path")
	names := psValues(args, "name")
	if len(paths) == 0 {
		for _, w := range positional(args) {
			if !slices.Contains(names, w) {
				paths = append(paths, w)
			}
		}
	}
	out := slices.Clone(paths)
	for _, p := range paths {
		for _, n := range names {
			out = append(out, p+"/"+n)
		}
	}
	if len(paths) == 0 {
		out = names
	}
	return targetsOf(out, false)
}

// downloads are the files curl, wget and Invoke-WebRequest write: -o and
// --output, -O and --output-document, -OutFile.
func downloads(args []string) []shellTarget {
	out := posixValues(args, "-o", "--output")
	out = append(out, posixValues(args, "-O", "--output-document")...)
	out = append(out, psValues(args, "outfile")...)
	return targetsOf(out, false)
}

// findRemoves are find's start paths when it deletes what it finds: with
// -delete, or -exec, -execdir, -ok or -okdir running a removing verb. The
// tests in between are not judged, so every start path counts as removed;
// without one find starts at the working folder.
func findRemoves(args []string) []shellTarget {
	var starts []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") || a == "(" || a == "!" {
			break
		}
		starts = append(starts, a)
	}
	if len(starts) == 0 {
		starts = []string{"."}
	}
	for i, a := range args {
		switch a {
		case "-delete":
			return targetsOf(starts, true)
		case "-exec", "-execdir", "-ok", "-okdir":
			if i+1 < len(args) && slices.Contains(everyFileRemoves, verbOf(args[i+1])) {
				return targetsOf(starts, true)
			}
		}
	}
	return nil
}

// inPlace are the files sed -i or perl -i edits in place: every positional
// argument but the script, which is the first one unless a flag brings it --
// one of scriptFlags alone or ending a bundle, or --expression, --file. -i is
// case-sensitive, because perl -I is an include path; it counts alone, with
// a suffix (-i.bak), in a bundle (-Ei, -pi) and as --in-place.
func inPlace(args []string, scriptFlags string) []shellTarget {
	edits, scripted := false, false
	var words []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			words = append(words, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(a, "--"):
			name, _, glued := strings.Cut(a[2:], "=")
			switch {
			case strings.HasPrefix(name, "in-place"):
				edits = true
			case name == "expression" || name == "file":
				scripted = true
				if !glued {
					i++
				}
			}
		case len(a) > 1 && a[0] == '-':
			flags, _, _ := strings.Cut(a[1:], ".")
			edits = edits || strings.ContainsRune(flags, 'i')
			if flags != "" && strings.ContainsRune(scriptFlags, rune(flags[len(flags)-1])) {
				scripted = true
				i++
			}
		default:
			words = append(words, a)
		}
	}
	if !edits {
		return nil
	}
	if !scripted && len(words) > 0 {
		words = words[1:]
	}
	return targetsOf(words, false)
}

// gitWrites is what a git subcommand writes: mv and rm their paths, checkout
// and restore the paths checkedOut names, clean what cleaned names. diff,
// log, show, status, blame, add and commit read; every other subcommand is
// unknown.
func gitWrites(dir string, args []string) ([]shellTarget, bool) {
	sub, rest := gitSubcommand(args)
	switch sub {
	case "mv":
		return moved(rest), true
	case "rm":
		return targetsOf(positional(rest), true), true
	case "checkout", "restore":
		return checkedOut(dir, sub, rest), true
	case "clean":
		return cleaned(rest), true
	case "diff", "log", "show", "status", "blame", "add", "commit":
		return nil, true
	}
	return nil, false
}

// gitSubcommand skips git's global options -- with the next word as the
// value of -C, -c, --git-dir, --work-tree and --namespace -- and answers the
// subcommand and its arguments. Paths after -C count from the working
// folder all the same: a named limit.
func gitSubcommand(args []string) (string, []string) {
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "-C" || a == "-c" || a == "--git-dir" || a == "--work-tree" || a == "--namespace":
			i++
		case strings.HasPrefix(a, "-"):
		default:
			return a, args[i+1:]
		}
	}
	return "", nil
}

// checkedOut are the paths git checkout or restore overwrites: every word
// after --, or without --, each argument that exists as a path. Without one
// it switches a branch, and restore --staged without --worktree touches only
// the index: neither writes a file.
func checkedOut(dir, sub string, args []string) []shellTarget {
	if sub == "restore" && (slices.Contains(args, "--staged") || slices.Contains(args, "-S")) &&
		!slices.Contains(args, "--worktree") && !slices.Contains(args, "-W") {
		return nil
	}
	if at := slices.Index(args, "--"); at >= 0 {
		return targetsOf(args[at+1:], false)
	}
	var paths []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-b" || a == "-B" || a == "--orphan" || a == "-s" || a == "--source":
			i++
		case strings.HasPrefix(a, "-"):
		default:
			place := a
			if !filepath.IsAbs(place) {
				place = filepath.Join(dir, place)
			}
			if _, err := os.Lstat(place); err == nil {
				paths = append(paths, a)
			}
		}
	}
	return targetsOf(paths, false)
}

// cleaned is what git clean removes: the paths it names, or the root without
// one. -n in a bundle and --dry-run only list; -e and --exclude bring a
// pattern, not a path.
func cleaned(args []string) []shellTarget {
	var paths []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			paths = append(paths, args[i+1:]...)
			i = len(args)
		case a == "--dry-run" || len(a) > 1 && a[0] == '-' && a[1] != '-' && a[1] != 'e' && letters(a[1:]) && strings.ContainsRune(a, 'n'):
			return nil
		case a == "-e" || a == "--exclude":
			i++
		case strings.HasPrefix(a, "-"):
		default:
			paths = append(paths, a)
		}
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}
	return targetsOf(paths, true)
}

// reads says whether a program only reads: one of readVerbs, or a loomux
// command that reads -- flow list and show, config get, list and proposals,
// and every check.
func reads(args []string) bool {
	if slices.Contains(readVerbs, verbOf(args[0])) {
		return true
	}
	sub, ok := loomuxArgs(args)
	if !ok || len(sub) == 0 {
		return false
	}
	switch sub[0] {
	case "check":
		return true
	case "flow":
		return len(sub) > 1 && (sub[1] == "list" || sub[1] == "show")
	case "config":
		return len(sub) > 1 && (sub[1] == "get" || sub[1] == "list" || sub[1] == "proposals")
	}
	return false
}

// dotNetCall finds a .NET file call in PowerShell: the class, the method and
// where the text after its opening parenthesis starts.
var dotNetCall = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)\[(?:system\.)?io\.(file|directory)\]::(\w+)\s*\(`)
})

// dotNetTargets are the paths the .NET calls of a line write or remove:
// File's Write*, Append*, Create*, Copy* and Replace*, Delete and Move, and
// Directory's CreateDirectory, Delete and Move. splitSegments cuts at the
// parenthesis, so the arguments are read from the line up to the one that
// closes it, with slashes for backslashes.
func dotNetTargets(line string) []shellTarget {
	var out []shellTarget
	for _, m := range dotNetCall().FindAllStringSubmatchIndex(line, -1) {
		class, method := strings.ToLower(line[m[2]:m[3]]), strings.ToLower(line[m[4]:m[5]])
		removes, writes := dotNetMethod(class, method)
		if !writes {
			continue
		}
		for _, arg := range callArguments(line[m[1]:]) {
			out = append(out, shellTarget{path: strings.ReplaceAll(arg, `\`, "/"), removes: removes, shell: true})
		}
	}
	return out
}

// dotNetMethod says whether a .NET file method removes and whether it
// writes at all.
func dotNetMethod(class, method string) (removes, writes bool) {
	switch {
	case method == "delete" || method == "move":
		return true, true
	case class == "directory":
		return false, method == "createdirectory"
	}
	for _, prefix := range []string{"write", "append", "create", "copy", "replace"} {
		if strings.HasPrefix(method, prefix) {
			return false, true
		}
	}
	return false, false
}

// callArguments splits the text after an opening parenthesis at its commas,
// up to the parenthesis that closes it, and answers each quoted argument
// without its quotes and each unquoted one that looks like a path.
func callArguments(text string) []string {
	var out []string
	take := func(arg string) {
		arg = strings.TrimSpace(arg)
		switch {
		case len(arg) >= 2 && (arg[0] == '\'' || arg[0] == '"') && arg[len(arg)-1] == arg[0]:
			out = append(out, arg[1:len(arg)-1])
		case pathLike(arg):
			out = append(out, arg)
		}
	}
	depth, start := 0, 0
	var quote byte
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '(':
			depth++
		case c == ')' && depth == 0:
			take(text[start:i])
			return out
		case c == ')':
			depth--
		case c == ',' && depth == 0:
			take(text[start:i])
			start = i + 1
		}
	}
	take(text[start:])
	return out
}

// guessedTargets are the words of a segment no strict split reads that look
// like a path, each taken for a write: what the segment does to them is
// unknown, and a removal would make a lone . the whole project.
func guessedTargets(segment string) []shellTarget {
	var out []shellTarget
	for _, w := range strings.Fields(segment) {
		if w = strings.Trim(w, `"'`); pathLike(w) {
			out = append(out, shellTarget{path: w, shell: true})
		}
	}
	return out
}

// pathLike says whether a word looks like a path: it holds a slash or a
// backslash, or begins with a dot.
func pathLike(w string) bool {
	return strings.ContainsAny(w, `/\`) || strings.HasPrefix(w, ".")
}

// rooted says whether p names a place without the working folder: an
// absolute path on either platform, or one under a drive letter.
func rooted(p string) bool {
	return filepath.IsAbs(p) || strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\`) ||
		len(p) >= 2 && p[1] == ':' && isLetter(p[0])
}

// under puts relative targets below base, the place an earlier cd moved to.
func under(base string, targets []shellTarget) []shellTarget {
	if base == "" {
		return targets
	}
	for i, t := range targets {
		if !rooted(t.path) {
			targets[i].path = base + "/" + t.path
		}
	}
	return targets
}

// joinPlace is where a relative path under base lies on the disk.
func joinPlace(root, base string) string {
	if rooted(base) {
		return base
	}
	return filepath.Join(root, filepath.FromSlash(base))
}

// changedDirectory is base after one segment: a cd, Set-Location or pushd
// moves it to its last positional argument (home without one), popd back to
// the root, anything else leaves it.
func changedDirectory(base string, args []string) string {
	if len(args) == 0 {
		return base
	}
	verb := verbOf(args[0])
	if verb == "popd" {
		return ""
	}
	if !slices.Contains(changeDirectory, verb) {
		return base
	}
	words := positional(args[1:])
	if len(words) == 0 {
		return "~"
	}
	next := filepath.ToSlash(words[len(words)-1])
	if rooted(next) || base == "" {
		return next
	}
	return base + "/" + next
}
```

  `callArguments` ruft `take` als Closure; das ist kein Paketzustand. Für
  `unknown` ist ein Segment mit leeren `args` bekannt (`segmentWrites` gibt
  dann `known = true`), darum braucht `shellWrites` keinen Längen-Test.

- [ ] **Schritt 4: Laufen lassen.** Run: `go test ./internal/hooks/ -count=1`, dann `go run ./cmd/loomux check coverage`. Expected: PASS, keine Funktion aus `shellwrites.go` unter 100 %. Fehlt einer Verzweigung ein Test (etwa `psValues` mit einem Wert als nächstem Wort am Zeilenende, `posixValues` mit `--`, `dotNetMethod` für `[IO.File]::ReadAllText`, `changedDirectory` ohne Argument), eine Zeile in die Tabelle von `TestShellWritesReadsTheTargetsOfEveryVerb` bzw. `TestShellWritesFindsNoTargetInAReadingLine` nachtragen, zum Beispiel:

```go
		"[IO.File]::ReadAllText('.loomux/config.toml')",
		"curl -o",
		"cd",
		"Copy-Item x -Destination",
```

  (in die „keine Ziele“-Liste; `cd` allein ergibt Basis `~`, aber kein Ziel).

- [ ] **Schritt 5: Commit.**

```text
feat(guard): read the paths a shell line writes from a table of verbs

shellWrites reads every variant and cut of a line and answers each path a
verb, a redirection or a .NET file call writes or removes: tee, Set-Content,
rm and Remove-Item, mv and Rename-Item, cp and Copy-Item, rsync, ln, dd,
tar, unzip, Expand-Archive, robocopy, xcopy, New-Item, touch, curl, wget,
Invoke-WebRequest, find -delete, sed -i, perl -i and git mv, rm, checkout,
restore and clean. A cd moves the place later relative paths start from.
It also names the programs it does not know. Nothing judges the targets
yet.

dropPrefixes now skips the separate value of sudo -u, env -u, xargs -n and
timeout -s, so a loomux call behind them is judged like any other.
```

---

### Task 4a: Eine Pfadprüfung für alle Ziele eines Aufrufs, Grundlinie gegen den neuen Weg

**Files:**
- Create: `internal/hooks/guardjudge.go`
- Modify: `internal/hooks/guard.go` (`checkTool`: Write/Edit über den `judge`, Gründe einmal)
- Modify: `internal/hooks/guardflow.go` (`flowFolderReasons` nimmt die Flows als Funktion)
- Test: `internal/hooks/guardshell_test.go` (neu)

**Interfaces:**
- Consumes: `shellTarget`, `shellWrites` (Task 3); `spellings` (Task 2); `pathReasons`, `relativePath`, `builtinPathRules`, `manifestReason` (guard.go); `protectedFlows`, `bundledFlowReason`, `unreadableFlowsReason`, `flowFolderName`, `flowsDir`.
- Produces:
  - `type judge struct { root string; policy config.Policy; flows func() ([]string, error) }`
  - `newJudge(root string, policy config.Policy) judge`
  - `(j judge) reasons(targets []shellTarget) []string` — nicht dedupliziert
  - `(j judge) pathReasons(rel string, removes bool) []string`
  - `(j judge) ancestorReasons(rel string) []string`
  - `below(rel, glob string, fold bool) bool`, `literalPrefix(glob string) string`
  - `uniqueReasons(reasons []string) []string`
  - `flowFolderReasons(rel string, protected func() ([]string, error)) []string` (neue Signatur)
  - Testhilfe `shellReasons(t *testing.T, root, line string, policy config.Policy) []string` und die Tabellen `manifestShellWrites()`, `manifestShellReads()`, `runFileShellWrites()`, `runFileShellReads()`, `flowShellWrites()`, `flowShellReads()`.

Die bestehenden Tests der Shell-Regeln bleiben in dieser Task unverändert und
grün; `writeSource` arbeitet noch. Die neuen Tests prüfen dieselben Zeilen
gegen den neuen Weg. **Kippungen gegenüber der Grundlinie** (jede im Test
kommentiert):

| Zeile | bisher | neu |
|---|---|---|
| alle 34 Manifest-Zeilen | „…so no shell command may write it“ | `manifestReason` |
| `Remove-Item -Recurse .loomux/state`, `rm -r .loomux/state/`, `git rm -r .loomux/state` | nur Laufdateien | Stopp-Tor **und** Laufdateien (Vorfahr von `state/hooks` und `state/runs`) |
| `rm -r .loomux/state/hooks` | erlaubt | Stopp-Tor |
| `rm -rf .loomux` | erlaubt (Lücke) | Stopp-Tor, Manifest, Laufdateien |
| jede Flow-Zeile | ein Grund mit ``example`, `review`` | je Flow ein Grund: `bundledWant("example")` bzw. `bundledWant("review")`, beide für `.loomux/flows` selbst |
| `rm -r .loomux/state/*`, `del … .loomux\state\r*`, `Remove-Item … flows\e*`, `del … flows\*\flow.toml`, `cp … flows/ex*/…`, `cp … flows/exampl?/…` | verweigert ohne Platte | verweigert nur, weil der Test die Ordner anlegt; ein Glob, der nichts trifft, bleibt wörtlich (wie bash) |

- [ ] **Schritt 1: Failing Tests schreiben** (`guardshell_test.go`):

```go
package hooks

import (
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

const stopGateWant = "the stop gate's own controls are not written by the party it gates"

// shellReasons is what the path check says about the targets of line.
func shellReasons(t *testing.T, root, line string, policy config.Policy) []string {
	t.Helper()
	targets, _ := shellWrites(root, line)
	return uniqueReasons(newJudge(root, policy).reasons(targets))
}

// manifestShellWrites write or remove the manifest, each in its own way.
func manifestShellWrites() []string {
	return []string{
		"cat >> .loomux/config.toml <<'EOF'\n[policy]\nEOF",
		"echo x >.loomux/config.toml",
		`echo x > "./.loomux/config.toml"`,
		`echo x > .loomux\config.toml`,
		"echo x >| /repo/.loomux/config.toml",
		"echo x 2>&1 >> .LOOMUX/Config.toml",
		"sed -i 's/a/b/' .loomux/config.toml",
		"sed -i.bak 's/a/b/' .loomux/config.toml",
		"sed -Ei 's/a/b/' .loomux/config.toml",
		"sed --in-place -e 's/a/b/' .loomux/config.toml",
		"sed 's/a/b/' .loomux/config.toml -i",
		"/usr/bin/sed -i x .loomux/config.toml",
		"perl -pi -e 's/a/b/' .loomux/config.toml",
		"printf x | tee -a .loomux/config.toml",
		"git status; tee .loomux/config.toml < x",
		"sudo tee .loomux/config.toml",
		"Set-Content -Path .loomux/config.toml -Value x",
		`"x" | Out-File .loomux\config.toml`,
		`Out-File -FilePath:.loomux\config.toml`,
		`Add-Content .loomux/config.toml "x"`,
		"Clear-Content .loomux/config.toml",
		"'x' | Tee-Object -FilePath .loomux/config.toml",
		"[IO.File]::WriteAllText('.loomux/config.toml', 'x')",
		"cp other.toml .loomux/config.toml",
		"cp -f other.toml '.loomux/config.toml'",
		"mv tmp .loomux/config.toml && ls",
		"mv .loomux/config.toml elsewhere.toml",
		"rm .loomux/config.toml",
		"Remove-Item .loomux\\config.toml",
		"Copy-Item -Destination .loomux\\config.toml -Path x.toml",
		"Copy-Item x.toml .loomux\\config.toml",
		"git checkout -- .loomux/config.toml",
		"dd if=x of=.loomux/config.toml",
		"x=$(sed -i s/a/b/ .loomux/config.toml)",
	}
}

// manifestShellReads read the manifest, or write beside it.
func manifestShellReads() []string {
	return []string{
		"cat .loomux/config.toml",
		"grep policy .loomux/config.toml",
		"Get-Content .loomux/config.toml",
		"sed -n '1,5p' .loomux/config.toml",
		"sed -n '/min/p' .loomux/config.toml",
		"cp .loomux/config.toml backup.toml",
		"Copy-Item .loomux\\config.toml -Destination backup.toml",
		"cat .loomux/config.toml > out.txt",
		"echo x > other.toml; cat .loomux/config.toml",
		"echo x > .loomux/config.toml.bak",
		"echo x > my.loomux/config.toml",
		"grep tee .loomux/config.toml",
		"git show HEAD:.loomux/config.toml",
		"dd if=.loomux/config.toml of=copy.toml",
		"sed -i s/a/b/ other.toml; cat .loomux/config.toml",
	}
}

// runFileShellWrites write or remove a run file, with the reasons each gets.
// The flips against the old command rule are marked.
func runFileShellWrites() map[string][]string {
	runs := []string{runFilesWant}
	// Flip: .loomux/state holds the stop gate's hooks as well as the runs.
	state := []string{stopGateWant, runFilesWant}
	return map[string][]string{
		"echo x >> .loomux/state/runs/0001.jsonl":       runs,
		"rm .loomux/state/runs/0001.flow":               runs,
		`Remove-Item .loomux\state\runs\0001.jsonl`:     runs,
		"sed -i s/a/b/ .loomux/state/runs/0001.jsonl":   runs,
		"rm -r .loomux/state/runs":                      runs,
		"git rm .loomux/state/runs/0001.jsonl":          runs,
		"cp other.jsonl ./.loomux/state/runs/0001.jsonl": runs,
		"Set-Content -Path .loomux/state/runs/0001.jsonl -Value x": runs,
		`echo x > ".LOOMUX/State/Runs/0001.jsonl"`:                 runs,
		`rd -Recurse .loomux\state\runs`:                           runs,
		`rmdir /s /q .loomux\state\runs`:                           runs,
		"[IO.Directory]::Delete('.loomux/state/runs', $true)":      runs,
		"git clean -fdx .loomux/state/runs":                        runs,
		"Remove-Item -Recurse .loomux/state":                       state,
		"rm -r .loomux/state/":                                     state,
		// The world holds .loomux/state/runs alone, so the glob reaches it alone.
		"rm -r .loomux/state/*":         runs,
		`del /s /q .loomux\state\r*`:    runs,
		"git rm -r .loomux/state":       state,
		// Flip: removing the hooks folder was allowed.
		"rm -r .loomux/state/hooks":     {stopGateWant},
		// Flip: removing .loomux was a hole.
		"rm -rf .loomux":                {stopGateWant, manifestReason, runFilesWant},
	}
}

// runFileShellReads read the run files, or copy into the folder above them.
func runFileShellReads() []string {
	return []string{
		"cat .loomux/state/runs/0001.jsonl",
		"ls .loomux/state/runs",
		"cp .loomux/state/runs/0001.jsonl backup.jsonl",
		"echo x > .loomux/state/runsx",
		"grep answered .loomux/state/runs/0001.jsonl > out.txt",
		"cp -r x .loomux/state/",
		"ls .loomux/state",
	}
}

// flowShellWrites write or remove a protected flow's folder, with the flows
// each reaches; the old command rule named both flows in one reason.
func flowShellWrites() map[string][]string {
	example, review := []string{bundledWant("example")}, []string{bundledWant("review")}
	both := []string{bundledWant("example"), bundledWant("review")}
	return map[string][]string{
		"rm -r .loomux/flows/example":                              example,
		"git rm .loomux/flows/review/questions/q.md":               review,
		"echo x > .loomux/flows/Example/flow.toml":                 example,
		`Remove-Item -Recurse .loomux\flows\example`:               example,
		"cp x.toml ./.loomux/flows/example/flow.toml":              example,
		`sed -i s/a/b/ ".LOOMUX/FLOWS/REVIEW/instructions/r.md"`:   review,
		`rmdir /s /q .loomux\flows\review`:                         review,
		`rd -Recurse .loomux\flows\example`:                        example,
		"[IO.Directory]::Delete('.loomux/flows/review', $true)":    review,
		"git clean -fd .loomux/flows/example":                      example,
		"rm -r .loomux/flows":                                      both,
		"rm -r .loomux/flows/":                                     both,
		"rm -r .loomux/flows/*":                                    both,
		`Remove-Item -Recurse .loomux\flows\e*`:                    example,
		"git rm -r .loomux/flows":                                  both,
		`del /s /q .loomux\flows\*\flow.toml`:                      both,
		"cp evil.md .loomux/flows/ex*/instructions/draft.md":       example,
		"cp evil.md .loomux/flows/exampl?/instructions/draft.md":   example,
	}
}

// flowShellReads leave the protected flows alone.
func flowShellReads() []string {
	return []string{
		"rm -r .loomux/flows/mine",
		"rm -r .loomux/flows/mine/*",
		"cat .loomux/flows/example/flow.toml",
		"echo x > .loomux/flows/example2/flow.toml",
		"cp -r .loomux/flows/example .loomux/flows/mine",
		"cp x .loomux/flows/mine/instructions/a.md",
		"cp -r mine .loomux/flows/",
		"ls .loomux/flows",
		"git status",
	}
}

// flowWorld is a project whose protected flows are example (the catalog)
// and review (an override), both on disk beside a flow of the agent's own.
func flowWorld(t *testing.T) string {
	t.Helper()
	catalog(t, "example")
	root := project(t)
	manifest(t, root, "[flow]\noverrides = [\"review\"]\n")
	mkfile(t, root, ".loomux/flows/example/flow.toml")
	mkfile(t, root, ".loomux/flows/example/instructions/draft.md")
	mkfile(t, root, ".loomux/flows/review/flow.toml")
	mkfile(t, root, ".loomux/flows/mine/flow.txt")
	return root
}

func TestTheShellPathCheckKeepsTheManifestBaseline(t *testing.T) {
	root := t.TempDir()
	for _, line := range manifestShellWrites() {
		if got := shellReasons(t, root, line, config.Policy{}); !slices.Equal(got, []string{manifestReason}) {
			t.Errorf("%q: reasons %q, want the manifest's", line, got)
		}
	}
	for _, line := range manifestShellReads() {
		if got := shellReasons(t, root, line, config.Policy{}); len(got) != 0 {
			t.Errorf("%q: reasons %q, want none", line, got)
		}
	}
}

func TestTheShellPathCheckKeepsTheRunFileBaseline(t *testing.T) {
	catalog(t)
	root := t.TempDir()
	mkfile(t, root, ".loomux/state/runs/0001.jsonl")
	for line, want := range runFileShellWrites() {
		if got := shellReasons(t, root, line, config.Policy{}); !slices.Equal(got, want) {
			t.Errorf("%q: reasons %q, want %q", line, got, want)
		}
	}
	for _, line := range runFileShellReads() {
		if got := shellReasons(t, root, line, config.Policy{}); len(got) != 0 {
			t.Errorf("%q: reasons %q, want none", line, got)
		}
	}
}

func TestTheShellPathCheckKeepsTheFlowBaseline(t *testing.T) {
	root := flowWorld(t)
	for line, want := range flowShellWrites() {
		if got := shellReasons(t, root, line, config.Policy{}); !slices.Equal(got, want) {
			t.Errorf("%q: reasons %q, want %q", line, got, want)
		}
	}
	for _, line := range flowShellReads() {
		if got := shellReasons(t, root, line, config.Policy{}); len(got) != 0 {
			t.Errorf("%q: reasons %q, want none", line, got)
		}
	}
}

// Every gap of the review is refused in the default mode, and the lines a
// careless reader would refuse stay open.
func TestTheShellPathCheckClosesTheGaps(t *testing.T) {
	root := flowWorld(t)
	mkfile(t, root, ".loomux/state/runs/0001.jsonl")
	mkfile(t, root, "build/a.txt")
	mkfile(t, root, "src/main.go")
	example := bundledWant("example")
	for line, want := range map[string]string{
		"ln -s evil .loomux/config.toml":                                     manifestReason,
		"tar -xf evil.tar -C .loomux/flows/example":                          example,
		"unzip evil.zip -d .loomux/flows/example":                            example,
		`Expand-Archive evil.zip -DestinationPath .loomux\flows\example`:     example,
		"New-Item -Path .loomux/config.toml -Force":                          manifestReason,
		"ni .loomux/state/runs/0002.flow":                                    runFilesWant,
		"touch .loomux/state/runs/0002.flow":                                 runFilesWant,
		`robocopy evil .loomux\flows\example /MIR`:                           example,
		`xcopy evil .loomux\flows\example /E /I`:                             example,
		"rsync -a evil/ .loomux/flows/example/":                              example,
		"find .loomux/state/runs -delete":                                    runFilesWant,
		"find .loomux/flows/example -exec rm {} +":                           example,
		"curl -sSLo .loomux/config.toml https://example.invalid/x":           manifestReason,
		"wget -O .loomux/config.toml https://example.invalid/x":              manifestReason,
		"Invoke-WebRequest https://example.invalid/x -OutFile .loomux/config.toml": manifestReason,
		"echo x > .loomux/sta*/runs/0001.jsonl":                              runFilesWant,
		"rm -rf .loomux/sta*/runs":                                           runFilesWant,
		"cp evil .loomux/flows/{example,zz}/flow.toml":                       example,
		"sudo -u root tee .loomux/config.toml":                               manifestReason,
		"xargs -n 1 rm .loomux/config.toml":                                  manifestReason,
		"timeout -s KILL 60 rm .loomux/config.toml":                          manifestReason,
		"rm -rf .loomux":                                                     manifestReason,
		"mv .loomux/flows elsewhere":                                         example,
		"rm -r .loomux/state":                                                runFilesWant,
		"echo x > ../sibling/.loomux/config.toml":                            manifestReason,
		"echo x > /repo/.loomux/config.toml":                                 manifestReason,
		`echo x > C:\repo\.loomux\config.toml`:                               manifestReason,
		"cd .loomux && rm config.toml":                                       manifestReason,
		`git commit -m "a; rm .loomux/config.toml b"`:                        manifestReason,
		// A named limit: the line is cut at its breaks, a heredoc body too.
		"cat > notes.md <<'EOF'\nrm -rf .loomux\nEOF":                        manifestReason,
		// find and git clean judge their start paths, not their tests.
		"find . -name '*.tmp' -delete":                                       manifestReason,
		"git clean -fdx":                                                     manifestReason,
	} {
		if got := shellReasons(t, root, line, config.Policy{}); !slices.Contains(got, want) {
			t.Errorf("%q: reasons %q, want %q among them", line, got, want)
		}
	}
	for _, line := range []string{
		"cp -r mine .loomux/flows/",
		"mv mine .loomux/flows/",
		"cat .loomux/config.toml",
		"ls .loomux/state/runs",
		"git status",
		"git checkout feat/x",
		"git checkout -b x",
		"git restore --staged x",
		"git clean -n",
		"git clean --dry-run -d",
		"rm -rf build/*",
		"rm -rf *",
		"rm -rf build",
		`git commit -m "fix(guard): keep .loomux/state/runs"`,
		"echo x 2>&1",
		"echo x >&2",
		"echo x > /dev/null",
		"echo x 2>nul",
		"echo x > $null",
		"find . -name x",
	} {
		if got := shellReasons(t, root, line, config.Policy{}); len(got) != 0 {
			t.Errorf("%q: reasons %q, want none", line, got)
		}
	}
}

// A glob is matched against the disk, not against a rule: rm -rf build/*
// passes beside *.pem until a key lies in build.
func TestAGlobIsJudgedByWhatItMatches(t *testing.T) {
	root := t.TempDir()
	mkfile(t, root, "build/a.txt")
	if got := shellReasons(t, root, "rm -rf build/*", config.Policy{}); len(got) != 0 {
		t.Fatalf("no key in build: %q", got)
	}
	mkfile(t, root, "build/server.pem")
	if got := shellReasons(t, root, "rm -rf build/*", config.Policy{}); !slices.Equal(got, []string{"secrets are not written by an agent"}) {
		t.Fatalf("a key in build: %q", got)
	}
}

func TestTooManyBraceVariantsRefuse(t *testing.T) {
	got := shellReasons(t, t.TempDir(), "touch x{1..65}", config.Policy{})
	if len(got) != 1 || !strings.Contains(got[0], "more than 64 brace variants") {
		t.Fatalf("reasons %q", got)
	}
}

func TestBelowFindsTheFolderAboveAKeptPath(t *testing.T) {
	for _, row := range []struct {
		rel, glob  string
		fold, want bool
	}{
		{".", ".aws/**", true, true},
		{".", "*.pem", true, false},
		{".loomux", "**/.loomux/config.toml", true, true},
		{"x/.loomux", "**/.loomux/config.toml", true, true},
		{"../sib/.loomux/state", "**/.loomux/state/runs/**", true, true},
		{".loomux/state/runs", "**/.loomux/state/runs/**", true, false},
		{"build", "**/.loomux/config.toml", true, false},
		{"build", "*.pem", true, false},
		{"docs", "docs/*.md", false, true},
		{"x/docs", "docs/*.md", false, false},
		{"x", "*/y.md", false, false},
		{".LOOMUX", "**/.loomux/config.toml", true, true},
		{".LOOMUX", ".loomux/state/**", false, false},
	} {
		if got := below(row.rel, row.glob, row.fold); got != row.want {
			t.Errorf("below(%q, %q, %v) = %v, want %v", row.rel, row.glob, row.fold, got, row.want)
		}
	}
	if literalPrefix("a/b*/c") != "a" || literalPrefix("*.pem") != "" || literalPrefix("a/b") != "a/b" {
		t.Fatal("literalPrefix")
	}
}
```

  (Import `strings` ergänzen.) Im selben Schritt in `guardflow_test.go`
  nichts ändern; die bestehenden Shell-Tests laufen weiter gegen
  `writeSource`.

- [ ] **Schritt 2: Laufen lassen.** Run: `go test ./internal/hooks/ -run 'TestTheShellPathCheck|TestAGlobIsJudged|TestTooManyBrace|TestBelow' -count=1` — Expected: FAIL (undefiniert).

- [ ] **Schritt 3: Implementieren** (`guardjudge.go`):

```go
package hooks

import (
	"slices"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/config"
)

// judge holds what the targets of one tool call are judged against. The
// protected flows are read once per call, not once per target: a brace
// expansion alone may bring 64.
type judge struct {
	root   string
	policy config.Policy
	flows  func() ([]string, error)
}

// newJudge is the judge of one call at root.
func newJudge(root string, policy config.Policy) judge {
	return judge{root: root, policy: policy, flows: sync.OnceValues(func() ([]string, error) {
		return protectedFlows(root)
	})}
}

// reasons judges every target of a call, a writing tool's and a shell
// line's alike, and answers each reason as often as it matched; the caller
// keeps one of each. A shell target is spelled first (braces, globs, a
// stream name); a writing tool's path is the file it names.
func (j judge) reasons(targets []shellTarget) []string {
	var reasons []string
	for _, target := range targets {
		rels := []string{relativePath(target.path, j.root)}
		if target.shell {
			spelled, err := spellings(j.root, target.path)
			if err != nil {
				reasons = append(reasons, err.Error())
				continue
			}
			rels = spelled
		}
		for _, rel := range rels {
			reasons = append(reasons, j.pathReasons(rel, target.removes)...)
		}
	}
	return reasons
}

// pathReasons judges one path relative to the root: loomux's own rules in any
// case -- Windows and macOS keep .LOOMUX/State/hooks and .loomux/state/hooks
// as one folder -- the project's as the project spelled them, the protected
// flow folders, and for a removal what it takes with it.
func (j judge) pathReasons(rel string, removes bool) []string {
	reasons := pathReasons(builtinPathRules, rel, true)
	reasons = append(reasons, pathReasons(j.policy.Paths, rel, false)...)
	reasons = append(reasons, flowFolderReasons(rel, j.flows)...)
	if removes {
		reasons = append(reasons, j.ancestorReasons(rel)...)
	}
	return reasons
}

// ancestorReasons judges a removal of rel by what lies below it: .loomux
// takes the manifest, the run files and the flows with it, and the root (git
// clean without a path) takes everything. Copying into such a folder stays
// open; only a removal or the source of a move asks here.
func (j judge) ancestorReasons(rel string) []string {
	var reasons []string
	for _, book := range []struct {
		rules []config.PathRule
		fold  bool
	}{{builtinPathRules, true}, {j.policy.Paths, false}} {
		for _, rule := range book.rules {
			if slices.ContainsFunc(rule.Match, func(glob string) bool { return below(rel, glob, book.fold) }) {
				reasons = append(reasons, rule.Reason)
			}
		}
	}
	protected, err := j.flows()
	if err != nil {
		if below(rel, "**/"+flowsDir+"/*", true) {
			reasons = append(reasons, unreadableFlowsReason(err))
		}
		return reasons
	}
	for _, name := range protected {
		if below(rel, "**/"+flowsDir+"/"+name, true) {
			reasons = append(reasons, bundledFlowReason(name))
		}
	}
	return reasons
}

// below says whether rel is a folder above what glob keeps. A glob's fixed
// part is its path up to the element with the first glob character; rel is
// above it when that part lies below rel, and the root is above every glob
// with a slash. A glob under any directory (**/) is also below a folder that
// ends in a leading part of it -- x/.loomux for **/.loomux/config.toml --
// and nowhere else, or every folder would be above it. A glob without a
// slash (*.pem) names no folder and has none above it.
func below(rel, glob string, fold bool) bool {
	if fold {
		rel, glob = strings.ToLower(rel), strings.ToLower(glob)
	}
	glob, anywhere := strings.CutPrefix(glob, "**/")
	if !strings.Contains(glob, "/") {
		return false
	}
	if rel == "." {
		return true
	}
	fixed := literalPrefix(glob)
	if fixed == "" {
		return false
	}
	if strings.HasPrefix(fixed, rel+"/") {
		return true
	}
	for part := fixed; anywhere; {
		cut := strings.LastIndexByte(part, '/')
		if cut < 0 {
			return false
		}
		part = part[:cut]
		if strings.HasSuffix(rel, "/"+part) {
			return true
		}
	}
	return false
}

// literalPrefix is the part of glob before the element that holds its first
// glob character, without the slash; the whole glob when it holds none.
func literalPrefix(glob string) string {
	meta := strings.IndexAny(glob, "*?[")
	if meta < 0 {
		return glob
	}
	cut := strings.LastIndexByte(glob[:meta], '/')
	if cut < 0 {
		return ""
	}
	return glob[:cut]
}

// uniqueReasons keeps the first of each reason, in order: a refusal names a
// reason once per call, however many targets or rules brought it.
func uniqueReasons(reasons []string) []string {
	var out []string
	for _, r := range reasons {
		if !slices.Contains(out, r) {
			out = append(out, r)
		}
	}
	return out
}
```

  In `guardflow.go` die Signatur von `flowFolderReasons` ändern:

```go
func flowFolderReasons(rel string, protected func() ([]string, error)) []string {
	name, under := flowFolderName(rel)
	if !under {
		return nil
	}
	names, err := protected()
	if err != nil {
		return []string{unreadableFlowsReason(err)}
	}
	if at := slices.IndexFunc(names, func(p string) bool { return strings.ToLower(p) == name }); at >= 0 {
		return []string{bundledFlowReason(names[at])}
	}
	return nil
}
```

  In `checkTool` den Write/Edit-Zweig auf den `judge` stellen und am Ende
  deduplizieren; der Shell-Zweig bleibt wie er ist:

```go
func checkTool(root, tool string, input map[string]any, policy config.Policy) []string {
	var reasons []string
	j := newJudge(root, policy)
	if guard.IsWritingTool(tool) {
		var targets []shellTarget
		for _, target := range guard.WriteTargets(input) {
			targets = append(targets, shellTarget{path: target})
		}
		reasons = append(reasons, j.reasons(targets)...)
	}
	if _, shell := commandTools[tool]; shell {
		// The shell branch as it stood before this task, word for word.
		lines, ok := commandLines(tool, input)
		if !ok && judgedOrRefused[tool] {
			reasons = append(reasons, "loomux found no command line in this "+tool+" call, so it cannot judge it and refuses")
		}
		for _, line := range lines {
			for _, rule := range append(builtinCommands(), policy.Commands...) {
				if rule.Regex.MatchString(line) {
					reasons = append(reasons, rule.Reason)
				}
			}
			if strings.Contains(strings.ToLower(line), "flows") {
				if rule, ok := flowFolderCommand(root); ok && rule.Regex.MatchString(line) {
					reasons = append(reasons, rule.Reason)
				}
			}
			if writesConfiguration(line) {
				reasons = append(reasons, "loomux init, config and area add write the configuration the guard reads, and merge-hook install and remove write executable hooks into repositories; a human runs them. An agent proposes a change with `loomux config set|unset … --propose`, which a human applies")
			}
			if answersAGate(line) {
				reasons = append(reasons, "a flow's gate asks a human; the answer is theirs. Ask the user to answer it with `flow resume <run> --answer \"…\"` themselves")
			}
		}
	}
	return uniqueReasons(reasons)
}
```

  (Das ist der Code aus `guard.go` vor der Task, nur der Kommentar „The flow
  folder rule reads the config, so only a line that could name such a folder
  pays for it.“ bleibt über dem `flows`-Test stehen.)

  In `TestABuiltinAndAConfiguredRuleBothCarryTheirReason` und
  `TestNotebookEditIsJudgedByItsOwnTargetKey` ändert sich nichts (verschiedene
  Gründe). Der Doc-Kommentar von `checkTool` bekommt den Satz „Each reason is
  named once per call.“

- [ ] **Schritt 4: Laufen lassen.** Run: `go test ./internal/hooks/ -count=1` und `go run ./cmd/loomux check coverage` — Expected: PASS, alte und neue Shell-Tests grün nebeneinander.

- [ ] **Schritt 5: Commit.**

```text
refactor(guard): judge every write target of a call through one path check

A judge per tool call reads the protected flows once and checks each target
against loomux's own rules, the project's, the flow folders and, for a
removal, the kept paths below it. Writing tools go through it now and each
reason is named once per call; shell lines still use the command rules.
```

---

### Task 4b: Shell-Ziele durch die Pfadprüfung, `writeSource` entfällt

**Files:**
- Modify: `internal/hooks/guard.go` (`checkTool`, `builtinCommands`; entfernen `writeSource`, `globName`)
- Modify: `internal/hooks/guardflow.go` (entfernen `flowFolderCommand`, Import `regexp`)
- Modify: `internal/hooks/guard_test.go`, `internal/hooks/guardflow_test.go`, `internal/hooks/guardshell_test.go`

**Interfaces:**
- Consumes: `newJudge`, `judge.reasons`, `uniqueReasons` (4a); `shellWrites` (Task 3).
- Produces: `checkTool(root, tool string, input map[string]any, policy config.Policy) []string` — Signatur unverändert; Shell-Ziele laufen durch `judge.reasons`, `[policy] paths` gilt jetzt auch für die Shell. `builtinCommands()` hält nur noch `git push`.

Kippungen im eigenen Repo (Projektregeln aus `.loomux/config.toml`, mit Probe
in Schritt 6): `rm coverage.out`, `rm -f coverage.out`, `mv bin/loomux.new.exe
bin/loomux.exe`, `rm -rf bin` (Vorfahr von `bin/*`), `rm -rf .loomux/state/x`
(`.loomux/state/**`), `rm -rf testdata` (Vorfahr von
`testdata/cases/*-source/**`), `cp x testdata/cases/1a-source/y`, `git clean
-fdx` (Wurzel). Nicht betroffen: `go build -o bin/loomux.exe ./cmd/loomux`
(`go` ist kein Verb der Tabelle), Aufnahmen unter `testdata/cases/<stufe>/`
ohne `-source`.

- [ ] **Schritt 1: Tests umstellen.** In `guardshell_test.go` die Hilfe
  `shellReasons` so ersetzen, dass sie den ganzen Wächter fragt, für beide
  Werkzeuge:

```go
// shellReasons is what the guard says about line, which Bash and
// PowerShell must say alike.
func shellReasons(t *testing.T, root, line string, policy config.Policy) []string {
	t.Helper()
	bash := checkTool(root, "Bash", map[string]any{"command": line}, policy)
	if power := checkTool(root, "PowerShell", map[string]any{"command": line}, policy); !slices.Equal(bash, power) {
		t.Errorf("%q: Bash %q, PowerShell %q", line, bash, power)
	}
	return bash
}
```

  und dazu:

```go
// A project's path rules hold for the shell as for a writing tool.
func TestAProjectPathRuleHoldsForTheShell(t *testing.T) {
	root := t.TempDir()
	policy := config.Policy{Paths: []config.PathRule{
		{Match: []string{"notes.txt"}, Reason: "notes are a human's"},
		{Match: []string{"bin/*"}, Reason: "build output"},
	}}
	for line, want := range map[string][]string{
		"echo x > notes.txt":           {"notes are a human's"},
		"rm -f notes.txt":              {"notes are a human's"},
		"mv bin/new.exe bin/loomux.exe": {"build output"},
		"rm -rf bin":                   {"build output"},
	} {
		if got := shellReasons(t, root, line, policy); !slices.Equal(got, want) {
			t.Errorf("%q: reasons %q, want %q", line, got, want)
		}
	}
	for _, line := range []string{"cat notes.txt", "go build -o bin/loomux.exe ./cmd/loomux"} {
		if got := shellReasons(t, root, line, policy); len(got) != 0 {
			t.Errorf("%q: reasons %q, want none", line, got)
		}
	}
}

// The command rules and the path rules answer one call together, each
// reason once, the command's first.
func TestACallNamesEachReasonOnce(t *testing.T) {
	root := t.TempDir()
	got := shellReasons(t, root, "rm .loomux/config.toml; tee .loomux/config.toml; git push", config.Policy{})
	want := []string{"Whether commits reach the remote is a human's decision.", manifestReason}
	if !slices.Equal(got, want) {
		t.Fatalf("reasons %q, want %q", got, want)
	}
	got = checkTool(root, "Write", map[string]any{"file_path": ".env", "notebook_path": ".env"}, config.Policy{})
	if !slices.Equal(got, []string{"secrets are not written by an agent"}) {
		t.Fatalf("a writing tool with the same target twice: %q", got)
	}
}
```

  In `guard_test.go` `TestAShellLineThatWritesTheManifestIsRefused` löschen
  (die Zeilen leben in `manifestShellWrites`/`manifestShellReads`). In
  `guardflow_test.go` `TestAShellLineThatWritesARunFileIsRefused` und
  `TestAShellLineThatWritesABundledFlowIsRefused` löschen (leben in
  `runFileShellWrites`/`flowShellWrites`); `TestNoProtectedFlowMeansNoShellRule`
  auf die `checkTool`-Prüfung kürzen:

```go
// With no flow to keep, no folder under .loomux/flows is kept.
func TestNoProtectedFlowMeansNoShellRule(t *testing.T) {
	catalog(t)
	root := project(t)
	for _, line := range []string{"rm -r .loomux/flows/example", "rm -r .loomux/flows"} {
		if got := checkTool(root, "Bash", map[string]any{"command": line}, config.Policy{}); len(got) != 0 {
			t.Fatalf("%q: reasons %q", line, got)
		}
	}
}
```

  `TestAFlowFolderUnderAnUnreadableFlowTableIsRefused`,
  `TestAFlowNamedTwiceIsNamedOnce` und
  `TestAnOverrideInCapitalsRefusesEveryFlowFolder` bleiben unverändert und
  müssen grün bleiben. Import `regexp` in `guardflow_test.go` bleibt (für
  `asGateAnswer`).

- [ ] **Schritt 2: Laufen lassen.** Run: `go test ./internal/hooks/ -count=1` — Expected: FAIL (`TestAProjectPathRuleHoldsForTheShell`, die Manifest-Grundlinie über `checkTool` mit zwei Gründen, `TestACallNamesEachReasonOnce`).

- [ ] **Schritt 3: Umschalten.** `checkTool` in `guard.go`:

```go
// checkTool judges one tool call against the built-in rules and the project's
// own, and answers every reason it found, each once: a caller that wants to
// say why it refuses needs all of them, not the first. A writing tool's
// targets and the targets of a shell line go through the same path check; a
// shell line is also read by the command rules and for loomux's own commands.
func checkTool(root, tool string, input map[string]any, policy config.Policy) []string {
	var reasons []string
	var targets []shellTarget
	if guard.IsWritingTool(tool) {
		for _, target := range guard.WriteTargets(input) {
			targets = append(targets, shellTarget{path: target})
		}
	}
	if _, shell := commandTools[tool]; shell {
		lines, ok := commandLines(tool, input)
		if !ok && judgedOrRefused[tool] {
			reasons = append(reasons, "loomux found no command line in this "+tool+" call, so it cannot judge it and refuses")
		}
		for _, line := range lines {
			for _, rule := range append(builtinCommands(), policy.Commands...) {
				if rule.Regex.MatchString(line) {
					reasons = append(reasons, rule.Reason)
				}
			}
			found, _ := shellWrites(root, line)
			targets = append(targets, found...)
			if writesConfiguration(line) {
				reasons = append(reasons, "loomux init, config and area add write the configuration the guard reads, and merge-hook install and remove write executable hooks into repositories; a human runs them. An agent proposes a change with `loomux config set|unset … --propose`, which a human applies")
			}
			if answersAGate(line) {
				reasons = append(reasons, "a flow's gate asks a human; the answer is theirs. Ask the user to answer it with `flow resume <run> --answer \"…\"` themselves")
			}
		}
	}
	reasons = append(reasons, newJudge(root, policy).reasons(targets)...)
	return uniqueReasons(reasons)
}
```

  `builtinCommands` kürzen:

```go
// builtinCommands compiles on first use rather than at load: this binary hangs
// on every tool call, and a package variable would pay for the expression in
// runs that never look at a command line. What a shell line writes is judged
// by the path rules (shellWrites), not here.
var builtinCommands = sync.OnceValue(func() []config.CommandRule {
	return []config.CommandRule{{
		Regex:  regexp.MustCompile(`(^|\s)git\s+push(\s|$)`),
		Source: `(^|\s)git\s+push(\s|$)`,
		Reason: "Whether commits reach the remote is a human's decision.",
	}}
})
```

  `globName` und `writeSource` samt Doc-Kommentar löschen; in `guardflow.go`
  `flowFolderCommand` und den Import `regexp` löschen. Danach
  `grep -rn "writeSource\|flowFolderCommand\|globName\|no shell command may write" internal docs README*.md`
  — Treffer in Code und Kommentaren beheben; Treffer in `docs/` und READMEs
  gehören Task 9 und werden dort bearbeitet (Liste in die Akte).

- [ ] **Schritt 4: Laufen lassen.** Run: `go test ./internal/... -count=1` und `go run ./cmd/loomux check coverage` — Expected: PASS.

- [ ] **Schritt 4b: Aufgezeichnete Fälle prüfen.** `grep -rl "pre-tool-use" testdata/cases`
  vollständig (ohne `head`) und je Treffer `compare` und `stdout` lesen.
  Ein Fall mit `compare` = `message` pinnt nur den Exit-Code und bleibt.
  Pinnt ein Fall den alten Shell-Grund („so no shell command may write it“)
  oder den gemeinsamen Flow-Grund (``example`, `review``) im Text: liegt er
  unter `testdata/cases/<stufe>/` (eigene Aufnahme von loomux), den
  erwarteten Text im selben Commit auf den neuen Grund ändern und in der
  Akte nennen; liegt er unter einem `*-source`-Baum, ist er Beleg des
  alten Werkzeugs, den kein Agent ändert (Projektregel) — dann anhalten und
  dem Controller melden, nicht umgehen.

- [ ] **Schritt 5: Commit.**

```text
fix(guard): judge the targets of a shell line by the path rules

A shell write or removal is now read for its targets and judged by the
same rules as a writing tool, in every spelling a shell expands: ln, tar,
unzip, Expand-Archive, New-Item, touch, robocopy, xcopy, rsync, find
-delete, curl -o, wget -O and Invoke-WebRequest -OutFile, globs in any
part of the path, brace expansion, wrapper flags with a value, and the
removal of a folder above a kept path. The expressions per path are gone.

A project's [policy] paths rules now hold for shell lines as well, and
the manifest has one reason for both roads.
```

- [ ] **Schritt 6: Kippungen im eigenen Repo nachweisen.** Nach dem Commit hat
  der Pre-commit-Hook `bin/loomux.exe` neu gebaut. Je Zeile der Liste oben
  eine Nutzlast per Write in den Scratchpad legen, etwa
  `…/scratchpad/flip-coverage.json` mit
  `{"tool_name":"Bash","tool_input":{"command":"rm coverage.out"}}`, und
  `bin/loomux.exe hook pre-tool-use --host claude --root . < …/flip-coverage.json`
  laufen lassen. Erwartet: Exit 2 mit dem Grund der Projektregel. Für
  `go build -o bin/loomux.exe ./cmd/loomux`: Exit 0. Ergebnis je Zeile in
  die Akte (für den PR-Text).

---

### Task 5: Schlüssel `[guard] mode`

**Files:**
- Create: `internal/config/guardsettings.go`
- Modify: `internal/config/policy.go` (`Policy.Strict`, `policyFile.Guard`, `ReadPolicy`)
- Modify: `internal/config/schema/schema.go`, `internal/config/schema/schema_test.go`
- Modify: `docs/en/configuration.md`, `docs/de/configuration.md`
- Test: `internal/config/policy_test.go`, `internal/hooks/pretool_test.go`

**Interfaces:**
- Consumes: `writeConfig(t, body) string` (policy_test.go), `project`, `manifest` (hooks-Tests), `policyReasons` (pretool.go).
- Produces: `config.GuardKeys() []string` (`["mode"]`), `config.GuardModes() []string` (`["default", "strict"]`), `config.Policy.Strict bool`; Schema-Schlüssel `guard.mode` (Enum, Base, Default `"default"`).

- [ ] **Schritt 1: Failing Tests schreiben.** In `internal/config/policy_test.go`:

```go
func TestReadPolicyReadsTheGuardMode(t *testing.T) {
	for body, want := range map[string]bool{
		"":                            false,
		"[guard]\n":                   false,
		"[guard]\nmode = \"default\"\n": false,
		"[guard]\nmode = \"strict\"\n":  true,
	} {
		policy, err := ReadPolicy(writeConfig(t, body))
		if err != nil || policy.Strict != want {
			t.Errorf("%q: strict %v, %v; want %v", body, policy.Strict, err, want)
		}
	}
}

// A mode the reader does not know refuses rather than fall back: a guard that
// ran default on a typo would look strict and not be.
func TestReadPolicyRefusesAGuardItCannotRead(t *testing.T) {
	for body, want := range map[string]string{
		"[guard]\nmode = \"hard\"\n":   "[guard] mode must be one of default, strict",
		"[guard]\nmode = 1\n":          "[guard] mode must be one of default, strict",
		"[guard]\nmod = \"strict\"\n":  `[guard] does not know "mod"; known: mode`,
		"guard = 3\n":                  "guard",
	} {
		_, err := ReadPolicy(writeConfig(t, body))
		if err == nil || !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), "config.toml") {
			t.Errorf("%q: err %v, want one naming the file and %q", body, err, want)
		}
	}
}
```

  (`writeConfig(t, body)` liefert die Wurzel.) In
  `internal/config/schema/schema_test.go` in
  `TestTheSchemaKnowsEveryKeyTheReadersRead`:

```go
	if got := ids("guard"); !slices.Equal(got, sorted(config.GuardKeys())) {
		t.Errorf("[guard]: schema %v, reader %v", got, config.GuardKeys())
	}
	if k, ok := Lookup("guard.mode"); !ok || k.Kind != Enum || !slices.Equal(k.Choices, config.GuardModes()) || k.Default != `"default"` {
		t.Errorf("guard.mode: %+v %v", k, ok)
	}
```

  und in `TestEveryNewKeyIsBase` `"guard.mode"` in die Liste aufnehmen. In
  `internal/hooks/pretool_test.go`:

```go
// A guard mode the reader refuses refuses every call, as a broken
// configuration does.
func TestAGuardModeTheReaderRefusesRefusesTheCall(t *testing.T) {
	root := project(t)
	manifest(t, root, "[guard]\nmode = \"hard\"\n")
	payload := map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": "git status"}}
	if _, err := policyReasons(payload, root); err == nil || !strings.Contains(err.Error(), "[guard] mode must be one of") {
		t.Fatalf("err %v", err)
	}
}
```

  (Falls `pretool_test.go` `readPolicy` in einem Test ersetzt, hier nicht
  anfassen: der Test nutzt den echten Leser.)

- [ ] **Schritt 2: Laufen lassen.** Run: `go test ./internal/config/... ./internal/hooks/ -run 'Guard|TheSchemaKnows|EveryNewKeyIsBase' -count=1` — Expected: FAIL.

- [ ] **Schritt 3: Implementieren.** `internal/config/guardsettings.go`:

```go
package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// GuardKeys are the keys [guard] knows, sorted.
func GuardKeys() []string { return []string{"mode"} }

// GuardModes are the values [guard] mode takes, the default first.
func GuardModes() []string { return []string{"default", "strict"} }

// parseGuard reads [guard] from the table ReadPolicy decoded and answers
// whether it asks for the strict mode, every finding at once. Only the two
// modes are values: a guard that fell back to default on a typo would look
// strict and not be.
func parseGuard(path string, table map[string]any) (bool, error) {
	var findings []string
	for _, key := range slices.Sorted(maps.Keys(table)) {
		if !slices.Contains(GuardKeys(), key) {
			findings = append(findings, fmt.Sprintf("[guard] does not know %q; known: %s", key, strings.Join(GuardKeys(), ", ")))
		}
	}
	mode, _ := table["mode"].(string)
	if _, present := table["mode"]; present && !slices.Contains(GuardModes(), mode) {
		findings = append(findings, "[guard] mode must be one of "+strings.Join(GuardModes(), ", "))
	}
	if len(findings) > 0 {
		return false, fmt.Errorf("%s: %s", path, strings.Join(findings, "; "))
	}
	return mode == "strict", nil
}
```

  In `policy.go`:

```go
// Policy is the [policy] table of .loomux/config.toml, and from [guard] how
// hard the guard reads a shell line.
type Policy struct {
	Paths    []PathRule
	Commands []CommandRule
	// Strict is [guard] mode = "strict": the guard also holds against an
	// agent that means to get round it. The zero value is the default mode.
	Strict bool
}
```

  `policyFile` bekommt als zweites Feld neben `Policy`:

```go
	Guard map[string]any `toml:"guard"`
```

  und `ReadPolicy` nach der Schleife über die Befehlsregeln, vor `return`:

```go
	strict, err := parseGuard(path, file.Guard)
	if err != nil {
		return Policy{}, err
	}
	policy.Strict = strict
```

  In `schema.go` hinter `policy.commands.rules`:

```go
		{Section: "guard", Name: "mode", Kind: Enum, Choices: config.GuardModes(), Default: `"default"`, Module: Base, Doc: "How hard the guard reads a shell line: default against an agent that errs, strict also against one that means to get round it."},
```

  `docs/en/configuration.md`: neuer Abschnitt nach `[policy.commands]`:

~~~markdown
### `[guard]` (How Hard the Guard Reads)

```toml
[guard]
mode = "default"   # or "strict"
```

`default` holds against an agent that errs or takes a shortcut: the common
writing and removing commands, their wrappers, globs, brace expansion and the
folders above a kept path. `strict` also holds against one that means to get
round it: every target is resolved by the file system (trailing dots and
blanks, 8.3 short names, case, junctions), loomux's own commands are known by
their arguments rather than the program's name, a program the guard does not
know is refused on a protected path, and so is a write whose path holds an
expansion (`$X`, `$(…)`, a backtick, `%X%`) that may land on one. Without the
key the mode is `default`; any other value is an error, and the guard then
refuses every call as for a broken configuration. Strict mode refuses more
than a project may like: in this repository it refuses `go build -o
bin/loomux.exe`, because `bin/*` is a protected path and `go` is no verb the
guard knows. See [Hooks](hooks.md#7-the-decision-path-of-pre-tool-use).
~~~

  `docs/de/configuration.md` an derselben Stelle:

~~~markdown
### `[guard]` (Wie genau der Wächter liest)

```toml
[guard]
mode = "default"   # oder "strict"
```

`default` hält gegen einen Agenten, der sich irrt oder bequem ist: die
gängigen Schreib- und Löschbefehle, ihre Wrapper, Globs, Brace-Expansion und
die Ordner über einem geschützten Pfad. `strict` hält zusätzlich gegen einen,
der gezielt umgehen will: jedes Ziel wird vom Dateisystem aufgelöst (Punkte
und Leerzeichen am Ende, 8.3-Kurznamen, Groß-/Kleinschreibung, Junctions),
loomux' eigene Befehle werden an ihren Argumenten erkannt statt am
Programmnamen, ein Programm, das der Wächter nicht kennt, wird auf einem
geschützten Pfad verweigert, ebenso eine Schreibung, deren Pfad eine Expansion
(`$X`, `$(…)`, Backtick, `%X%`) trägt, die dort landen kann. Ohne den
Schlüssel gilt `default`; jeder andere Wert ist ein Fehler, und der Wächter
verweigert dann jeden Aufruf wie bei kaputter Konfiguration. Der strikte Modus
verweigert mehr, als einem Projekt lieb sein mag: in diesem Repo verweigert er
`go build -o bin/loomux.exe`, weil `bin/*` geschützt ist und `go` kein Verb,
das der Wächter kennt. Siehe [Hooks](hooks.md#7-der-entscheidungsweg-von-pre-tool-use).
~~~

  (Den deutschen Anker gegen die tatsächliche Überschrift in
  `docs/de/hooks.md` prüfen und angleichen.) Im Abschnitt „Complete Annotated
  Example“ beider Sprachen einen auskommentierten Block ergänzen:
  `# [guard]` / `# mode = "default"`.

- [ ] **Schritt 4: Laufen lassen.** Run: `go test ./internal/config/... ./internal/hooks/ -count=1`, `go run ./cmd/loomux check coverage` — Expected: PASS.

- [ ] **Schritt 5: Commit.**

```text
feat(config): add [guard] mode

mode takes default or strict; without it the guard runs default, and any
other value or key in [guard] is an error that names the file, so the
guard refuses as for any broken configuration. The policy reader decodes
it with the rules it already reads, and the schema lists it for loomux
config.
```

- [ ] **Schritt 6: Übergabe an den Menschen (kein Schreiben durch den Agenten).**
  Dieses Repo bleibt im Standardmodus (E7). Will der Mensch den strikten
  Modus hier trotzdem, nennt der Agent ihm nach Task 6 die beiden Befehle,
  statt `.loomux/config.toml` anzufassen:
  `loomux config set guard.mode strict --propose` (darf der Agent) und
  `loomux config apply <id>` (nur der Mensch), mit dem Hinweis auf
  `go build -o bin/loomux.exe`.

---

### Task 6: Strikter Modus

**Files:**
- Modify: `internal/hooks/guardjudge.go` (`reasons` löst im Strikt-Modus auf; neu `resolvedSpellings`, `isDevice`, `strictReasons`, `namedPaths`, `beforeExpansion`, `mayReach`, `protectedGlobs`)
- Modify: `internal/hooks/guard.go` (`checkTool`; `writesConfiguration`, `readingWrites`, `wordsWriteConfiguration` mit `anyProgram`; neu `programArgs`, `knownProgram`, `knownTools`)
- Modify: `internal/hooks/guardflow.go` (`answersAGate`, `readingAnswers`, `wordsAnswer` mit `anyProgram`)
- Modify: `internal/hooks/shellwrites.go` (neu `otherWriteVerbs`)
- Test: `internal/hooks/guardstrict_test.go`, `guardstrict_windows_test.go`, `guardstrict_other_test.go`; Aufrufe in `guard_test.go`/`guardflow_test.go` bekommen `, false`

**Interfaces:**
- Consumes: `config.Policy.Strict` (Task 5), `guard.ResolvePath(target string) (string, error)`, `judge` (4a), `shellWrites` (Task 3).
- Produces: `writesConfiguration(line string, anyProgram bool) bool`, `answersAGate(line string, anyProgram bool) bool`, `programArgs(words []string, anyProgram bool) ([]string, bool)`, `knownProgram(word string) bool`, `(j judge) strictReasons(found []shellTarget, unknown [][]string) []string`.

- [ ] **Schritt 1: Aufrufe umstellen (mechanisch).** In `guard_test.go`
  `writesConfiguration(line)` → `writesConfiguration(line, false)` (drei
  Schleifen), in `guardflow_test.go` `answersAGate(line)` →
  `answersAGate(line, false)` (drei Stellen). Per Edit, dann
  `grep -n "writesConfiguration(\|answersAGate(" internal/hooks/*_test.go`.

- [ ] **Schritt 2: Failing Tests schreiben** (`guardstrict_test.go`):

```go
package hooks

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

var strictPolicy = config.Policy{Strict: true}

// command is a shell call's input.
func command(line string) map[string]any { return map[string]any{"command": line} }

// Strict mode knows loomux by what it is told: a copied or renamed binary
// answering a gate or writing the configuration is refused there, and passes
// the default mode, which knows loomux by its name.
func TestStrictModeKnowsLoomuxByItsArguments(t *testing.T) {
	root := project(t)
	has := func(reasons []string, part string) bool {
		return slices.ContainsFunc(reasons, func(r string) bool { return strings.Contains(r, part) })
	}
	for line, part := range map[string]string{
		"doc.exe flow resume 0001 --answer yes":        "a flow's gate asks a human",
		`.\x\doc.exe flow resume 0001 --answer=yes`:     "a flow's gate asks a human",
		"./git.exe flow resume 0001 --answer yes":      "a flow's gate asks a human",
		"doc.exe config set a b":                       "a human runs them",
		"doc.exe init":                                 "a human runs them",
		"doc.exe area add":                             "a human runs them",
		"doc.exe merge-hook install":                   "a human runs them",
	} {
		if got := checkTool(root, "Bash", command(line), strictPolicy); !has(got, part) {
			t.Errorf("strict %q: reasons %q, want %q", line, got, part)
		}
		if got := checkTool(root, "Bash", command(line), config.Policy{}); has(got, part) {
			t.Errorf("default %q: reasons %q", line, got)
		}
	}
	for _, line := range []string{
		"git config set user.name x", "gh config set editor vim", "npm init -y", "terraform init",
		"cat config", "echo config set a b", "ls init", "go mod init x",
	} {
		if got := checkTool(root, "Bash", command(line), strictPolicy); len(got) != 0 {
			t.Errorf("strict %q: reasons %q, want none", line, got)
		}
	}
}

// A program the guard does not know is refused on a protected path in strict
// mode; the read list and loomux's reading commands stay open.
func TestStrictModeRefusesAnUnknownProgramOnAProtectedPath(t *testing.T) {
	root := project(t)
	isStrict := func(r string) bool { return strings.HasPrefix(r, "in strict mode loomux refuses `") }
	for _, line := range []string{
		"frob .loomux/config.toml",
		"frob --in=.loomux/config.toml",
		`sh -c "sed -e x .loomux/config.toml > y"`,
		"git stash push .loomux/config.toml",
		"python edit.py .loomux/state/runs/0001.jsonl",
	} {
		if got := checkTool(root, "Bash", command(line), strictPolicy); !slices.ContainsFunc(got, isStrict) {
			t.Errorf("strict %q: reasons %q", line, got)
		}
		if got := checkTool(root, "Bash", command(line), config.Policy{}); len(got) != 0 {
			t.Errorf("default %q: reasons %q, want none", line, got)
		}
	}
	for _, line := range []string{
		"cat .loomux/config.toml", `type .loomux\config.toml`, "Get-Content .loomux/config.toml",
		"less .loomux/config.toml", "more .loomux/config.toml", "head -n 3 .loomux/config.toml",
		"tail .loomux/config.toml", "wc -l .loomux/config.toml", "stat .loomux/config.toml",
		"file .loomux/config.toml", "jq . .loomux/state/runs/x.json", "ls .loomux", "dir .loomux",
		"Get-ChildItem .loomux", "grep x .loomux/config.toml", "rg x .loomux",
		"Select-String x .loomux/config.toml", "diff .loomux/config.toml x",
		"git diff .loomux/config.toml", "git log -- .loomux/config.toml", "git show HEAD:.loomux/config.toml",
		"git status .loomux", "git blame .loomux/config.toml", "git add .loomux/config.toml",
		"git commit -m x .loomux/config.toml", "loomux flow list", "loomux flow show 0001",
		"loomux config get a", "loomux config list", "loomux config proposals", "loomux check precommit",
		"frob src/x", "go vet .", "echo .loomux/config.toml",
	} {
		if got := checkTool(root, "Bash", command(line), strictPolicy); len(got) != 0 {
			t.Errorf("strict %q: reasons %q, want none", line, got)
		}
	}
}

// A write whose path holds an expansion is refused in strict mode where the
// fixed part before it may be, or lie above, a protected path.
func TestStrictModeRefusesAnExpansionThatMayLandOnAProtectedPath(t *testing.T) {
	catalog(t, "example")
	root := project(t)
	for _, line := range []string{
		"echo x > .loomux/$X",
		"echo x > .loomux/con${X}",
		"echo x > .e$X",
		"cp x .loomux/sta$X",
		"echo x > .loomux/flows/ex%X%/flow.toml",
		"echo x > ../sib/.loomux/$X",
	} {
		got := checkTool(root, "Bash", command(line), strictPolicy)
		if !slices.ContainsFunc(got, func(r string) bool { return strings.Contains(r, "the expansion may land on a protected path") }) {
			t.Errorf("strict %q: reasons %q", line, got)
		}
		if got := checkTool(root, "Bash", command(line), config.Policy{}); len(got) != 0 {
			t.Errorf("default %q: reasons %q, want none", line, got)
		}
	}
	for _, line := range []string{
		"echo x > $X", "echo x > $env:TMP/x", "echo x > build/$X.txt", "echo x > src/$X",
		"echo x > $null", "echo x 2>nul", "echo x > /dev/null", "echo x 2>&1", "echo x >&2",
	} {
		if got := checkTool(root, "Bash", command(line), strictPolicy); len(got) != 0 {
			t.Errorf("strict %q: reasons %q, want none", line, got)
		}
	}
}

// Strict mode adds refusals, never removes one: every default refusal of the
// baseline holds there too.
func TestStrictModeKeepsEveryDefaultRefusal(t *testing.T) {
	root := t.TempDir()
	for _, line := range manifestShellWrites() {
		if got := checkTool(root, "Bash", command(line), strictPolicy); !slices.Contains(got, manifestReason) {
			t.Errorf("%q: reasons %q", line, got)
		}
	}
}
```

  `guardstrict_windows_test.go`:

```go
//go:build windows

package hooks

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
	"golang.org/x/sys/windows"
)

// shortName is the 8.3 alias the file system keeps for path, or "" when this
// volume keeps none; asked through GetShortPathName, not through the resolver
// the guard uses.
func shortName(t *testing.T, path string) string {
	t.Helper()
	wide, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]uint16, windows.MAX_PATH)
	length, err := windows.GetShortPathName(wide, &buffer[0], uint32(len(buffer)))
	if err != nil || length == 0 || int(length) > len(buffer) {
		t.Skipf("no short name for %q: %v", path, err)
	}
	short := windows.UTF16ToString(buffer[:length])
	if strings.EqualFold(short, path) {
		return ""
	}
	return short
}

// Windows opens example. as example; strict mode resolves it, the default
// mode does not.
func TestStrictModeResolvesATrailingDot(t *testing.T) {
	catalog(t, "example")
	root := project(t)
	mkfile(t, root, ".loomux/flows/example/flow.toml")
	line := "echo x > .loomux/flows/example./flow.toml"
	if got := checkTool(root, "Bash", command(line), config.Policy{}); len(got) != 0 {
		t.Fatalf("default: reasons %q, a named limit", got)
	}
	if got := checkTool(root, "Bash", command(line), strictPolicy); !slices.Equal(got, []string{bundledWant("example")}) {
		t.Fatalf("strict: reasons %q", got)
	}
	write := map[string]any{"file_path": ".loomux/flows/example ./flow.toml"}
	if got := checkTool(root, "Write", write, strictPolicy); !slices.Equal(got, []string{bundledWant("example")}) {
		t.Fatalf("strict write with a trailing blank: reasons %q", got)
	}
}

func TestStrictModeResolvesAShortName(t *testing.T) {
	root := project(t)
	short := shortName(t, filepath.Join(root, ".loomux"))
	if short == "" {
		t.Skip("this volume keeps no 8.3 names")
	}
	line := "echo x > " + filepath.Base(short) + "/config.toml"
	if got := checkTool(root, "Bash", command(line), config.Policy{}); len(got) != 0 {
		t.Fatalf("default: reasons %q, a named limit", got)
	}
	if got := checkTool(root, "Bash", command(line), strictPolicy); !slices.Equal(got, []string{manifestReason}) {
		t.Fatalf("strict: reasons %q", got)
	}
}
```

  `guardstrict_other_test.go`:

```go
//go:build !windows

package hooks

import "testing"

func TestStrictModeResolvesATrailingDot(t *testing.T) {
	t.Skip("a trailing dot names another file outside Windows; the strict test runs on Windows")
}

func TestStrictModeResolvesAShortName(t *testing.T) {
	t.Skip("8.3 short names are Windows'; the strict test runs on Windows")
}
```

- [ ] **Schritt 3: Laufen lassen.** Run: `go test ./internal/hooks/ -run 'TestStrictMode' -count=1` — Expected: FAIL.

- [ ] **Schritt 4: Implementieren.** In `guard.go` die Leser für loomux'
  Befehle mit `anyProgram` versehen:

```go
func writesConfiguration(line string, anyProgram bool) bool {
	// Judged on the line as written, before any rewrite: a continuation or
	// an escape the rewrites resolve is already no plain line.
	plain := plainLine(line)
	for _, variant := range lineVariants(line) {
		// The quote-blind cut also breaks inside a wrapper's quoted inner
		// command and puts its loomux at the head of a segment, while the
		// wrapper reads that string once more. Such a segment finds a call
		// but exempts nothing; only one whose bounds lie outside quotes may.
		aware := splitSegments(variant, true)
		for _, segment := range segments(variant) {
			exempt := plain && slices.Contains(aware, segment)
			for _, words := range readings(segment) {
				if readingWrites(words, exempt, anyProgram) {
					return true
				}
			}
		}
	}
	return false
}

func readingWrites(words []string, plain, anyProgram bool) bool {
	if wordsWriteConfiguration(words, plain, anyProgram) {
		return true
	}
	for i, w := range words {
		if (w == "{" || w == "}") && wordsWriteConfiguration(words[i+1:], false, anyProgram) {
			return true
		}
	}
	return false
}
```

  In `wordsWriteConfiguration(words []string, plain, anyProgram bool)` die
  Zeile `found, ok := loomuxArgs(words)` durch
  `found, ok := programArgs(words, anyProgram)` ersetzen. Dazu:

```go
// knownTools are programs whose subcommands share loomux's names (git
// config set, gh config set, npm init, terraform init). With the verb table
// and readVerbs they are the programs strict mode does not take for a renamed
// loomux, by their bare name only: a path to a file of that name may be
// anything.
var knownTools = []string{"git", "gh", "go", "npm", "npx", "pnpm", "yarn", "cargo", "uv", "uvx",
	"poetry", "pip", "pip3", "docker", "dotnet", "terraform", "kubectl", "helm", "make", "cmake"}

// knownProgram says whether word is the bare name of a program strict mode
// leaves to its name.
func knownProgram(word string) bool {
	if strings.ContainsAny(word, `/\`) {
		return false
	}
	name := verbOf(word)
	for _, list := range [][]string{knownTools, readVerbs, everyFileWrites, everyFileRemoves,
		moveVerbs, renameVerbs, copyVerbs, otherWriteVerbs} {
		if slices.Contains(list, name) {
			return true
		}
	}
	return false
}

// programArgs are the arguments loomux would get from words: loomuxArgs,
// and in strict mode the arguments of every program knownProgram does not
// name -- a copied or renamed binary (doc.exe flow resume … --answer) is
// loomux by what it is told, not by what it is called.
func programArgs(words []string, anyProgram bool) ([]string, bool) {
	if args, ok := loomuxArgs(words); ok || !anyProgram || knownProgram(words[0]) {
		return args, ok
	}
	return words[1:], true
}
```

  In `shellwrites.go` neben den Listen:

```go
	// otherWriteVerbs are the verbs verbWrites reads one by one, for
	// knownProgram; git stands in knownTools.
	otherWriteVerbs = []string{"dd", "tar", "unzip", "expand-archive", "robocopy", "xcopy", "new-item",
		"ni", "curl", "wget", "invoke-webrequest", "iwr", "find", "sed", "perl"}
```

  In `guardflow.go`:

```go
// answersAGate says whether a shell line runs `loomux flow resume` with an
// answer. A gate asks a human; an agent that answered its own gates would
// approve its own plan and its own push. The line is read the way
// writesConfiguration reads it, and has the same holes; with anyProgram, in
// strict mode, a program knownProgram does not name counts as loomux.
func answersAGate(line string, anyProgram bool) bool {
	for _, variant := range lineVariants(line) {
		for _, segment := range segments(variant) {
			for _, words := range readings(segment) {
				if readingAnswers(words, anyProgram) {
					return true
				}
			}
		}
	}
	return false
}

// readingAnswers judges one reading from its head and from every word after
// a lone { or }, for readingWrites' reason: a block opens a command.
func readingAnswers(words []string, anyProgram bool) bool {
	for i, w := range words {
		if (w == "{" || w == "}") && wordsAnswer(words[i+1:], anyProgram) {
			return true
		}
	}
	return wordsAnswer(words, anyProgram)
}

// wordsAnswer says whether one reading, past what runs in front of the
// program, is loomux flow resume with an answer, or a Start-Process of
// loomux, whose arguments the words cannot see.
func wordsAnswer(words []string, anyProgram bool) bool {
	words = dropPrefixes(words)
	if len(words) == 0 {
		return false
	}
	if startsLoomux(words) {
		return true
	}
	args, ok := programArgs(words, anyProgram)
	return ok && len(args) > 1 && args[0] == "flow" && args[1] == "resume" && namesAnswer(args[2:])
}
```

  `checkTool` ganz:

```go
func checkTool(root, tool string, input map[string]any, policy config.Policy) []string {
	var reasons []string
	var targets []shellTarget
	j := newJudge(root, policy)
	if guard.IsWritingTool(tool) {
		for _, target := range guard.WriteTargets(input) {
			targets = append(targets, shellTarget{path: target})
		}
	}
	if _, shell := commandTools[tool]; shell {
		lines, ok := commandLines(tool, input)
		if !ok && judgedOrRefused[tool] {
			reasons = append(reasons, "loomux found no command line in this "+tool+" call, so it cannot judge it and refuses")
		}
		for _, line := range lines {
			for _, rule := range append(builtinCommands(), policy.Commands...) {
				if rule.Regex.MatchString(line) {
					reasons = append(reasons, rule.Reason)
				}
			}
			found, unknown := shellWrites(root, line)
			targets = append(targets, found...)
			if policy.Strict {
				reasons = append(reasons, j.strictReasons(found, unknown)...)
			}
			if writesConfiguration(line, policy.Strict) {
				reasons = append(reasons, "loomux init, config and area add write the configuration the guard reads, and merge-hook install and remove write executable hooks into repositories; a human runs them. An agent proposes a change with `loomux config set|unset … --propose`, which a human applies")
			}
			if answersAGate(line, policy.Strict) {
				reasons = append(reasons, "a flow's gate asks a human; the answer is theirs. Ask the user to answer it with `flow resume <run> --answer \"…\"` themselves")
			}
		}
	}
	reasons = append(reasons, j.reasons(targets)...)
	return uniqueReasons(reasons)
}
```

  In `guardjudge.go` `reasons` vor der inneren Schleife:

```go
		if j.policy.Strict {
			resolved, err := resolvedSpellings(j.root, rels)
			if err != nil {
				reasons = append(reasons, err.Error())
				continue
			}
			rels = append(rels, resolved...)
		}
```

  und neu (Imports `fmt`, `path/filepath`, `github.com/xidus90/loomux/internal/brain/guard`):

```go
// resolvedSpellings are rels as the file system names them: through
// guard.ResolvePath, which folds 8.3 aliases, a trailing dot or blank, case
// and junctions over the longest existing part, then relative to the root
// resolved the same way. No second resolver. A device name (nul, con) is no
// place to resolve and is skipped.
func resolvedSpellings(root string, rels []string) ([]string, error) {
	base, err := guard.ResolvePath(root)
	if err != nil {
		return nil, fmt.Errorf("loomux cannot resolve the project root %s, so it refuses in strict mode: %v", root, err)
	}
	var out []string
	for _, rel := range rels {
		place := filepath.FromSlash(rel)
		if !filepath.IsAbs(place) {
			place = filepath.Join(root, place)
		}
		full, err := guard.ResolvePath(place)
		switch {
		case err == nil:
			out = append(out, relativePath(full, base))
		case !isDevice(rel):
			return nil, fmt.Errorf("loomux cannot resolve %s, so it refuses in strict mode: %v", rel, err)
		}
	}
	return out, nil
}

// isDevice says whether rel names a Windows device, whatever its extension.
func isDevice(rel string) bool {
	name, _, _ := strings.Cut(strings.ToLower(filepath.Base(rel)), ".")
	switch name {
	case "nul", "con", "prn", "aux", "conin$", "conout$":
		return true
	}
	return len(name) == 4 && (strings.HasPrefix(name, "com") || strings.HasPrefix(name, "lpt")) && name[3] >= '1' && name[3] <= '9'
}

// strictReasons are what strict mode adds for one line: a program the guard
// does not know that names a protected path, and a write whose path holds an
// expansion that may land on one.
func (j judge) strictReasons(found []shellTarget, unknown [][]string) []string {
	var reasons []string
	for _, args := range unknown {
		for _, word := range namedPaths(args[1:]) {
			if relativePath(word, j.root) == "." {
				continue
			}
			if len(j.reasons([]shellTarget{{path: word, removes: true, shell: true}})) > 0 {
				reasons = append(reasons, fmt.Sprintf("in strict mode loomux refuses `%s` on %s: it does not know whether the program writes there", verbOf(args[0]), word))
			}
		}
	}
	for _, target := range found {
		if fixed, expands := beforeExpansion(target.path); expands && j.mayReach(fixed) {
			reasons = append(reasons, fmt.Sprintf("in strict mode loomux refuses a write to %s: the expansion may land on a protected path", target.path))
		}
	}
	return reasons
}

// namedPaths are the paths a program's arguments may name: each word, the
// value of a flag glued with = or :, and each field of a word that holds a
// blank -- the inner command of sh -c "…" among them.
func namedPaths(words []string) []string {
	var out []string
	for _, w := range words {
		if strings.HasPrefix(w, "-") {
			if at := strings.IndexAny(w, "=:"); at >= 0 {
				out = append(out, w[at+1:])
			}
			continue
		}
		out = append(out, w)
		if strings.ContainsAny(w, " \t") {
			for _, field := range strings.Fields(w) {
				out = append(out, strings.Trim(field, `"'`))
			}
		}
	}
	return out
}

// beforeExpansion is the part of p before its first expansion -- $X, $(…),
// ${…}, a backtick, cmd's %X% -- and whether p holds one.
func beforeExpansion(p string) (string, bool) {
	at := strings.IndexAny(p, "$`")
	if pct := strings.IndexByte(p, '%'); pct >= 0 && strings.IndexByte(p[pct+1:], '%') >= 0 && (at < 0 || pct < at) {
		at = pct
	}
	if at < 0 {
		return "", false
	}
	return p[:at], true
}

// protectedGlob is one glob the guard keeps, and whether it matches in any case.
type protectedGlob struct {
	glob string
	fold bool
}

// protectedGlobs are the globs of every rule and protected flow folder.
func (j judge) protectedGlobs() []protectedGlob {
	var out []protectedGlob
	for _, rule := range builtinPathRules {
		for _, g := range rule.Match {
			out = append(out, protectedGlob{strings.ToLower(g), true})
		}
	}
	for _, rule := range j.policy.Paths {
		for _, g := range rule.Match {
			out = append(out, protectedGlob{g, false})
		}
	}
	names, err := j.flows()
	if err != nil {
		return append(out, protectedGlob{"**/" + flowsDir + "/*", true})
	}
	for _, name := range names {
		out = append(out, protectedGlob{"**/" + flowsDir + "/" + strings.ToLower(name), true})
	}
	return out
}

// mayReach says whether a path that begins with fixed may be a protected one
// or lie above one: a glob's literal part begins with fixed, or fixed begins
// with that part and a slash -- tried from the start of fixed and, for a glob
// under any directory, from every element; a glob without a slash is tried
// against fixed's last element. An empty fixed part is the barrier's
// question: the expansion may be any absolute path.
func (j judge) mayReach(fixed string) bool {
	fixed = strings.TrimPrefix(filepath.ToSlash(fixed), "./")
	if fixed == "" {
		return false
	}
	for _, g := range j.protectedGlobs() {
		candidate := fixed
		if g.fold {
			candidate = strings.ToLower(candidate)
		}
		glob, anywhere := strings.CutPrefix(g.glob, "**/")
		if !strings.Contains(glob, "/") {
			base := candidate[strings.LastIndexByte(candidate, '/')+1:]
			literal := glob
			if meta := strings.IndexAny(glob, "*?["); meta >= 0 {
				literal = glob[:meta]
			}
			if base != "" && literal != "" && strings.HasPrefix(literal, base) {
				return true
			}
			continue
		}
		literal := literalPrefix(glob)
		for start := 0; literal != ""; {
			c := candidate[start:]
			if c != "" && (strings.HasPrefix(literal, c) || strings.HasPrefix(c, literal+"/")) {
				return true
			}
			next := strings.IndexByte(c, '/')
			if !anywhere || next < 0 {
				break
			}
			start += next + 1
		}
	}
	return false
}
```

  Doc-Kommentare nachziehen: `writesConfiguration` und `answersAGate` nennen
  `anyProgram` („in strict mode every program knownProgram does not name
  counts as loomux“); `readings` behält seine Grenzen, die im
  Standardmodus gelten.

- [ ] **Schritt 5: Laufen lassen.** Run: `go test ./internal/hooks/ -count=1` und `go run ./cmd/loomux check coverage` — Expected: PASS. Fehlt `isDevice` für `com1`/`lpt1`/`conin$` ein Aufruf, einen Tabellentest ergänzen:

```go
func TestIsDeviceNamesTheWindowsDevices(t *testing.T) {
	for rel, want := range map[string]bool{"nul": true, "x/NUL.txt": true, "com1": true, "lpt9": true, "conout$": true,
		"com0": false, "comx": false, "nullx": false, "notes.txt": false} {
		if got := isDevice(rel); got != want {
			t.Errorf("isDevice(%q) = %v", rel, got)
		}
	}
}
```

  Scheitert `resolvedSpellings` bei einer Wurzel, die nicht auflösbar ist,
  nur unter Bedingungen, die ein Test nicht herstellt (ein laufwerksrelativer
  `--root` erreicht `checkTool` nie), bekommt der Zweig einen eigenen Test
  mit `resolvedSpellings("C:relative", …)` unter Windows; unter POSIX ist
  der Zweig nicht erreichbar, dann `//coverage:exempt` mit genau diesem
  Grund über `resolvedSpellings` — nur wenn die Abdeckung unter POSIX
  gemessen wird (die Coverage-Lane läuft im Tor unter Windows; vorher mit
  `go run ./cmd/loomux check precommit --show` nachsehen).

- [ ] **Schritt 6: Commit.**

```text
feat(guard): add a strict mode against deliberate workarounds

With [guard] mode = "strict" every write target is also resolved by the
file system, so a trailing dot or blank, an 8.3 short name, another case
or a junction names the kept path it opens. loomux's own commands are
known by their arguments, whatever the program is called, except the bare
names of common tools whose subcommands share them. A program the guard
does not know is refused on a protected path, and so is a write whose
path holds an expansion that may land on one.
```

---

### Task 7: `TestCases4c1` ohne festen Port

**Files:**
- Modify: `internal/cli/cases_4c1_test.go`

**Interfaces:**
- Consumes: `cases.RunCaseWith(c *Case, run RunFunc, normalize Normalizer) (*RunOutcome, error)`, `cases.NormalizeState(world, tree map[string][]byte) map[string][]byte`, `fakeollama.Load`.
- Produces: `const recordedOllama = "127.0.0.1:11435"`; `serveFakeOllama(t *testing.T, world string) (*callLog, string)`; `foldOllama(tree map[string][]byte, address string) map[string][]byte`; `replay4c1(t *testing.T, c *cases.Case)`.

Befund beim Lesen: nur die drei Welten mit `ollama-fixture.json`
(`open-area-not-asked`, `proposal-invented`, `proposal-kept`) nennen
`127.0.0.1:11435` in `world/config.toml` und `world_after/config.toml`;
`model-unreachable` nennt 11436 und `endpoint-off-loopback` 192.0.2.1, beide
ohne Fake. `dir` im Lauf-Callback ist die bereitgestellte Kopie der Welt;
der Normalisierer läuft nach dem Lauf über beide Bäume samt stdout.

- [ ] **Schritt 1: Failing Test schreiben** (am Ende der Datei):

```go
// The replay does not need the recorded port: another process holding it is
// what made the cases flaky under a parallel gate.
func TestCases4c1DoesNotNeedTheRecordedPort(t *testing.T) {
	blocker, err := net.Listen("tcp", recordedOllama)
	if err == nil {
		t.Cleanup(func() { _ = blocker.Close() })
	}
	corpus, err := filepath.Abs(filepath.Join("..", "..", "testdata", "cases", "4c1"))
	if err != nil {
		t.Fatal(err)
	}
	all, err := cases.DiscoverCases(corpus, "")
	if err != nil {
		t.Fatal(err)
	}
	at := slices.IndexFunc(all, func(c *cases.Case) bool { return c.Verb+"/"+c.Name == "reconcile/proposal-kept" })
	if at < 0 {
		t.Fatal("reconcile/proposal-kept is gone")
	}
	replay4c1(t, all[at])
}

func TestFoldOllamaPutsTheRecordedAddressBack(t *testing.T) {
	tree := map[string][]byte{"config.toml": []byte(`endpoint = "http://127.0.0.1:50123"`)}
	got := foldOllama(tree, "127.0.0.1:50123")
	if string(got["config.toml"]) != `endpoint = "http://127.0.0.1:11435"` {
		t.Fatalf("%s", got["config.toml"])
	}
	if got := foldOllama(tree, ""); len(got) != 1 {
		t.Fatal("no address folds nothing")
	}
}
```

- [ ] **Schritt 2: Laufen lassen.** Run: `go test ./internal/cli/ -run 'TestCases4c1DoesNotNeedTheRecordedPort|TestFoldOllama' -count=1` — Expected: FAIL (undefiniert; mit dem alten Code schlüge der erste Test an „the fixed port of the fake Ollama is taken“).

- [ ] **Schritt 3: Implementieren.** `serveFakeOllama` ersetzen, `foldOllama`
  und `replay4c1` ergänzen, `TestCases4c1` rufen lassen:

```go
// recordedOllama is where the recorded worlds' config.toml expects the fake.
const recordedOllama = "127.0.0.1:11435"

// serveFakeOllama puts the fake on a free port, names that port in the
// staged world's config.toml where the recording names recordedOllama, and
// hands back the lines it logged and the address. The recordings stay as
// they are: foldOllama puts the recorded address back before the
// comparison. The server closes with t.
func serveFakeOllama(t *testing.T, world string) (*callLog, string) {
	t.Helper()
	calls := &callLog{}
	path := filepath.Join(world, "ollama-fixture.json")
	if _, err := os.Stat(path); err != nil {
		return calls, ""
	}
	fixture, err := fakeollama.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	config := filepath.Join(world, "config.toml")
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, bytes.ReplaceAll(data, []byte(recordedOllama), []byte(address)), 0o644); err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: fixture.Handler(calls)}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return calls, address
}

// foldOllama puts the recorded address back wherever the run wrote the
// fake's real one, so a world_after and a stdout compare as recorded.
func foldOllama(tree map[string][]byte, address string) map[string][]byte {
	if address == "" {
		return tree
	}
	for name, data := range tree {
		tree[name] = bytes.ReplaceAll(data, []byte(address), []byte(recordedOllama))
	}
	return tree
}

// replay4c1 replays one recording with the fake Ollama answering both sides.
func replay4c1(t *testing.T, c *cases.Case) {
	t.Helper()
	name := c.Verb + "/" + c.Name
	var calls *callLog
	var address string
	outcome, err := cases.RunCaseWith(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
		t.Chdir(dir)
		t.Setenv("LOOMUX_STATE_DIR", dir)
		t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", dir)
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
		for _, entry := range cases.GitEnv(dir) {
			key, value, _ := strings.Cut(entry, "=")
			t.Setenv(key, value)
		}
		useRecordedEngine(t, dir)
		calls, address = serveFakeOllama(t, dir)
		return Run(args, stdin, stdout, stderr)
	}, func(world, tree map[string][]byte) map[string][]byte {
		return cases.NormalizeState(world, foldOllama(tree, address))
	})
	if err != nil {
		t.Fatal(err)
	}
	// stderr is loomux's wording, not the reference's (as in 3b).
	got := slices.DeleteFunc(slices.Clone(outcome.Mismatches), func(m string) bool {
		return strings.HasPrefix(m, "stderr:")
	})
	wanted := slices.Clone(expected4c1[name])
	slices.Sort(got)
	slices.Sort(wanted)
	if !slices.Equal(got, wanted) {
		t.Fatalf("mismatches differ from the expected ones\ngot:\n%s\nwant:\n%s\nstdout:\n%s",
			strings.Join(got, "\n"), strings.Join(wanted, "\n"), outcome.ActualStdout)
	}
	if n := strings.Count(calls.String(), "\n"); n != wantOllamaCalls4c1[name] {
		t.Fatalf("the fake Ollama got %d requests, the reference sent %d:\n%s",
			n, wantOllamaCalls4c1[name], calls.String())
	}
}
```

  In `TestCases4c1` den Körper von `t.Run(name, …)` durch
  `replay4c1(t, c)` ersetzen und den Doc-Kommentar anpassen: „The cases run
  one after another; each fake takes a port of its own.“ Die Namen
  `*cases.Case` und `DiscoverCases` gegen `internal/cases` prüfen (heute
  liefert `DiscoverCases` `[]*Case`).

- [ ] **Schritt 4: Laufen lassen.** Run: `go test ./internal/cli/ -run 'TestCases4c1|TestFoldOllama' -count=3` — Expected: PASS, auch mit belegtem 11435 (zweites Terminal: `go run ./cmd/loomux dev fake-ollama --fixture testdata/cases/4c1/reconcile/proposal-kept/world/ollama-fixture.json` oder den tatsächlichen Namen des Befehls aus `internal/cli/dev.go:366` laufen lassen, dann den Test).

- [ ] **Schritt 5: Commit.**

```text
test(cli): let the fake Ollama of the reconcile cases take a free port

The replay listened on the port the recorded worlds name and failed when
another process held it. It now listens on a free port, names it in the
staged world's config.toml and folds it back to the recorded address
before the comparison; the recordings are unchanged.
```

---

### Task 8: `TestGitReadsTheIndexACommitHookHandsIn` — systematisches Debuggen

**Files:**
- Modify: nach Befund `internal/code/query/git_test.go`, `internal/code/query/blast_test.go` und/oder `internal/code/query/git.go`

**Interfaces:**
- Consumes: `indexFileFor(dir string) (string, bool)`, `resolved(path string) string`, `hookIndex(t, root) string`, `gitRepo(t) string` (Tests), `GraphReady(root string) (bool, string)`.
- Produces: ein Regressionstest, der die Bedingung aus dem Hook im normalen Testlauf herstellt, und der Fix.

REQUIRED SUB-SKILL: superpowers:systematic-debugging. Kein Fix vor dem
Befund. Symptom (Spec): nur im Torlauf aus dem Pre-commit-Hook
„with the hook's index: false "nothing staged"“, in der Shell grün.

- [ ] **Schritt 1: Im Hook-Kontext reproduzieren, ohne das Tor.** Ein
  Wegwerf-Hook im Scratchpad, der nur den Test fährt und immer mit 1 endet
  (kein Commit entsteht). Per Write:
  `…/scratchpad/hookrepro/pre-commit`:

```sh
#!/bin/sh
log="$HOOKREPRO_LOG"
cd "$(git rev-parse --show-toplevel)"
env | grep -E '^(GIT_|TMP=|TEMP=|TMPDIR=|HOME=|USERPROFILE=)' | sort > "$log/env.txt"
go test -run 'TestGitReadsTheIndexACommitHookHandsIn' -count=5 -v ./internal/code/query > "$log/test.log" 2>&1
echo "exit $?" >> "$log/test.log"
exit 1
```

  Dann: `mkdir -p …/scratchpad/hookrepro/log`, und im Worktree
  `HOOKREPRO_LOG=<absoluter log-Pfad> git -c core.hooksPath=<absoluter Pfad von hookrepro> commit --allow-empty -m probe`.
  Erwartet: der Commit scheitert (Exit 1, gewollt), `test.log` und
  `env.txt` liegen im Log-Ordner. Zum Vergleich dieselben zwei Befehle
  (`env | grep …`, `go test …`) in der Shell in einen zweiten Ordner.
  Reproduziert es sich nicht mit `--allow-empty`, einmal mit einer
  gestageten Scratch-Änderung und einmal mit `git commit -a` probieren (git
  setzt dort `GIT_INDEX_FILE` auf eine temporäre Datei).

- [ ] **Schritt 2: Messpunkte setzen (nicht committen).** Vorübergehend am
  Anfang von `TestGitReadsTheIndexACommitHookHandsIn` und in
  `indexFileFor` loggen (`t.Logf` im Test; in `indexFileFor` über eine
  Testhilfe, die dieselben Schritte nachrechnet, statt Produktivcode zu
  ändern):

```go
	t.Logf("TempDir=%q TMP=%q TEMP=%q", os.TempDir(), os.Getenv("TMP"), os.Getenv("TEMP"))
	t.Logf("inherited GIT_INDEX_FILE=%q GIT_DIR=%q", os.Getenv("GIT_INDEX_FILE"), os.Getenv("GIT_DIR"))
	out, _ := exec.Command("git", "-C", root, "rev-parse", "--absolute-git-dir").Output()
	t.Logf("absolute-git-dir=%q resolved=%q", strings.TrimSpace(string(out)), resolved(strings.TrimSpace(string(out))))
	t.Logf("index=%q IsAbs=%v resolved=%q", index, filepath.IsAbs(index), resolved(index))
```

  (der `exec`-Aufruf hier mit `cmd.Env = gitenv.Environ()`, wie der
  Produktivcode). Schritt 1 wiederholen und die beiden Logs
  nebeneinanderlegen.

- [ ] **Schritt 3: Hypothesen einzeln prüfen**, je eine Beobachtung aus dem Log:
  - **H1 — `TMP`/`TEMP` im Hook ohne Laufwerk oder in POSIX-Form** (`/tmp`,
    `\Users\…`): dann ist `t.TempDir()` für `filepath.IsAbs` unter Windows
    nicht absolut, und `indexFileFor` verwirft den Index schon in der
    ersten Zeile. Beleg: `IsAbs=false` im Hook-Log, `true` in der Shell.
  - **H2 — Schreibweise:** `TMP` ist im Hook ein 8.3-Kurzname oder anders
    geschrieben als in der Shell, und `resolved()` einer der beiden Seiten
    fällt auf `filepath.Clean` zurück (Fehler von `ResolvePath`). Beleg:
    `resolved(index)` beginnt nicht mit `resolved(absolute-git-dir)`.
  - **H3 — geerbte git-Umgebung** erreicht einen git-Aufruf, der nicht
    über `gitenv.Environ()` läuft (in `gitRepo`, `copyTree`, `Build`,
    `hookIndex`, `GraphReady` → `runGit`, `GraphEnv`): Beleg: `env.txt`
    zeigt `GIT_DIR`/`GIT_INDEX_FILE`/`GIT_WORK_TREE`, und ein Aufruf ohne
    `gitenv` (`grep -n "exec.Command(\"git\"" internal/code/query/*.go`)
    liest sie.
  - **H4 — `worktrees/`-Regel:** `rel` beginnt im Hook mit `worktrees`,
    weil `--absolute-git-dir` im Hook den Haupt-git-Ordner statt des
    Test-Repos nennt (Folge von H3). Beleg: `absolute-git-dir` zeigt in den
    echten Checkout.

  Bestätigt das Log keine der vier, eine neue Hypothese aus dem Log bilden
  und genauso prüfen; nicht raten.

- [ ] **Schritt 4: Regressionstest schreiben**, der die bestätigte Bedingung
  im gewöhnlichen `go test` herstellt, und rot sehen. Je Befund:
  - H1: `t.Setenv("TMP", <laufwerksloser Pfad>)` und `t.Setenv("TEMP", …)`
    vor `gitRepo(t)` (unter Windows; sonst Skip mit Grund), Erwartung wie im
    bestehenden Test.
  - H2: `t.Setenv("TMP", shortName(t, os.TempDir()))` (Windows-Testdatei
    mit eigener `shortName`-Hilfe wie in Task 6, Skip ohne Kurznamen).
  - H3/H4: `t.Setenv("GIT_DIR", <Pfad des echten git-Ordners einer zweiten
    Scratch-Repo>)` und `t.Setenv("GIT_WORK_TREE", …)` vor `gitRepo(t)`.

  Run: `go test ./internal/code/query/ -run <neuer Test> -count=1` — Expected: FAIL mit „nothing staged“.

- [ ] **Schritt 5: Fix an der Stelle, die der Befund nennt**, und grün sehen.
  Richtung je Befund (nicht vorwegnehmen, sondern nach dem Log wählen):
  bei H1 ist die Ursache die Testumgebung (git selbst reicht dem Hook
  einen absoluten Index) — der Test legt seine Welt unter einem absoluten
  Pfad an (`filepath.Abs(t.TempDir())`, oder `gitRepo` löst die Wurzel
  auf), der Produktivcode bleibt; bei H2 gehört die Auflösung in
  `indexFileFor` (beide Seiten über `guard.ResolvePath`, Fehlerfall prüfen);
  bei H3/H4 bekommt der gefundene git-Aufruf `gitenv.Environ()`.
  Run: `go test ./internal/code/query/ -count=1`, dann Schritt 1 erneut:
  im Hook-Log `PASS` und `exit 0` für fünf Läufe.

- [ ] **Schritt 6: Messpunkte entfernen**, `git diff` darf nur Test und Fix
  zeigen. Befund, Beleg und Fix in die Akte.

- [ ] **Schritt 7: Commit.** Typ nach Befund: `fix(query): …`, wenn
  Produktivcode falsch war, sonst `test(query): …`. Die Nachricht nennt die
  Bedingung im Hook (etwa „TMP without a drive under the pre-commit hook“)
  und warum der Test sie jetzt selbst herstellt.

---

### Task 9: Messung, Doku, Changelog-Notizen

**Files:**
- Create: `testdata/bench/guard-shell-hooks.json`
- Modify: `docs/en/hooks.md`, `docs/de/hooks.md` (Abschnitt 7), `docs/en/flows.md`, `docs/de/flows.md` (Gates-Abschnitt, ca. Z. 350–356), `README.md`, `README.de.md` (Punkt „Agent-Safe Configuration“, ca. Z. 313), `docs/en/benchmarks.md`, `docs/de/benchmarks.md`

**Interfaces:**
- Consumes: alles vorher; `loomux dev bench hooks <cases> -n <N> --out <dir>`.
- Produces: Messeintrag, Doku, Changelog-Block für den PR-Text.

- [ ] **Schritt 1: Zwei Binaries bauen.** Mit dem SHA aus Schritt 0.3:
  `git archive <base> | tar -x -C …/scratchpad/before` und dort
  `go build -o C:/Users/micro/AppData/Local/Temp/loomux-guardshell-bench/before.exe ./cmd/loomux`;
  `go build -o C:/Users/micro/AppData/Local/Temp/loomux-guardshell-bench/after.exe ./cmd/loomux`
  im Worktree. `go version` und beide SHAs in die Akte.

- [ ] **Schritt 2: Stdin-Dateien anlegen** (per Write, nicht eingecheckt)
  unter `C:/Users/micro/AppData/Local/Temp/loomux-guardshell-bench/`:
  `edit-outside.json` (`{"tool_name":"Edit","tool_input":{"file_path":"internal/hooks/guard.go"}}`),
  `edit-flows.json` (`…".loomux/flows/mine/flow.toml"…`),
  `bash.json` (`{"tool_name":"Bash","tool_input":{"command":"git status"}}`),
  `session.json` und die beiden Welten `norun`/`waiting` wie in
  `docs/en/benchmarks.md` (Eintrag 2026-09-27 16:18) beschrieben, dazu der
  sechste Fall `bash-long.json`:

```json
{"tool_name":"Bash","tool_input":{"command":"cd docs && sudo -u root env X=1 timeout -s KILL 60 cp -r notes/{a,b,c}/*.md build/out/ ; tee -a build/log.txt < in.txt | xargs -n 1 echo && rm -rf build/tmp/* 2>&1 ; git status"}}
```

- [ ] **Schritt 3: Fall-Datei schreiben** `testdata/bench/guard-shell-hooks.json`:
  die fünf Fälle aus `testdata/bench/flow-hooks.json` mit `dir` und `--root`
  auf `C:/Users/micro/Documents/#GIT/loomux/.worktrees/guard-shell` (die
  beiden `session-start`-Fälle behalten ihre Welten, jetzt unter
  `loomux-guardshell-bench/`), `stdin` unter `loomux-guardshell-bench/`,
  `argv[0]` = `C:/Users/micro/AppData/Local/Temp/loomux-guardshell-bench/loomux.exe`;
  als sechster Fall:

```json
  {
    "name": "pre-tool-use Bash, a long line with wrappers, braces and a glob (this worktree)",
    "dir": "C:/Users/micro/Documents/#GIT/loomux/.worktrees/guard-shell",
    "stdin": "C:/Users/micro/AppData/Local/Temp/loomux-guardshell-bench/bash-long.json",
    "mode": "single",
    "steps": [
      {
        "argv": ["C:/Users/micro/AppData/Local/Temp/loomux-guardshell-bench/loomux.exe", "hook", "pre-tool-use", "--host", "claude", "--root", "C:/Users/micro/Documents/#GIT/loomux/.worktrees/guard-shell"]
      }
    ]
  }
```

- [ ] **Schritt 4: Messen.** Drei Durchgänge, abwechselnd: vor jedem
  Durchgang `before.exe` bzw. `after.exe` nach `…/loomux.exe` kopieren, dann
  `after.exe dev bench hooks testdata/bench/guard-shell-hooks.json -n 30 --out …/scratchpad/bench-<pass>`.
  Ausgabe erst ganz in eine Datei, dann lesen. Erwartet: jeder Lauf Exit 0
  (der lange Fall verweigert nichts), `pre-tool-use` warm ≤ 35 ms. Liegt
  ein Fall darüber: anhalten, nicht dokumentieren, dem Controller melden
  (Profil mit `go test -bench` auf `checkTool` als nächster Schritt).

- [ ] **Schritt 5: Eintrag in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`**,
  chronologisch am Ende, Form wie der Eintrag vom 2026-09-27 16:18: Datum und
  Uhrzeit, Worktree und Zweig, `before`/`after` mit SHAs, Go-Version,
  Maschine, Ziel („die Verbtabelle, Brace- und Glob-Auflösung und die
  Vorfahrenprüfung dürfen `pre-tool-use` nicht über 35 ms warm heben“),
  Methode, Tabelle kalt/warm-Median/warm-Min je Fall und Binary, Lesart. Im
  deutschen Eintrag dieselben Zahlen.

- [ ] **Schritt 6: `docs/en/hooks.md`, Abschnitt 7.** Die Regeltabelle
  ersetzen: die beiden Zeilen „Command | a shell write onto
  `.loomux/state/runs` …“ und „Command | the same forms onto the folder of a
  protected flow …“ entfallen; neu bzw. geändert:

```markdown
| Path | `**/.loomux/config.toml` | .loomux/config.toml: the manifest is where the barrier reads its own limits, so no agent may write it |
| Path | `.loomux/no-verify`, `**/.loomux/state/hooks/**` | the stop gate's own controls are not written by the party it gates |
| Path | `**/.loomux/state/runs/**`, the journals and markers of [flow runs](flows.md#2-running-a-flow) | a flow's journal and marker are written by loomux, not by the party the gates ask |
| Path | `.loomux/flows/<name>/` under any directory, for every flow of this binary's catalog and every name in `[flow] overrides` | a bundled flow's gates and instructions are a human's to change; … |
```

  Darunter neue Absätze (englisch):

  - **One check for both roads.** A shell line is read for the paths it
    writes or removes, and those go through the same path rules as a
    writing tool's target — the built-in ones, the project's `[policy]
    paths` and the flow folders. Since this change a project's path rules
    hold for the shell too: `rm coverage.out` is refused where
    `coverage.out` is protected.
  - **The verbs it reads:** die Liste aus Task 3 (jede genannte Datei;
    nur das Ziel; `find`; Umleitungen; in place; git; .NET), die Wrapper
    (`sudo`, `env`, `xargs`, `timeout` mit ihren Schaltern mit Wert,
    `nice`, `command`, `exec`, `nohup`, `time`, `cmd /c`, `VAR=x`) und
    `cd`/`Set-Location`/`pushd`/`popd`.
  - **How a path is spelled:** Braces bis 64 Varianten (darüber
    Ablehnung), Globs gegen die Platte mit bashs Punkt-Regel, ein Glob ohne
    Treffer bleibt wörtlich; `**/` in einer Regel steht für jedes
    Verzeichnis; eine Löschung oder Verschiebung eines Ordners über einem
    geschützten Pfad wird verweigert, eine Kopie hinein nicht; `find …
    -delete` und `git clean` ohne Pfad gelten als Löschung ihrer Startpfade
    bzw. der Wurzel.
  - **Modes:** Verweis auf `[guard] mode` in configuration.md, was `strict`
    zusätzlich tut.
  - **Limits in the default mode:** ein Pfad in einer Variablen, ein
    Programm, das die Datei selbst öffnet (`python -c …`, ein
    Build-Werkzeug), ein Befehl in einer Zeichenkette (`sh -c "…"`), ein
    Alias, ein umbenanntes Binary, Punkte/Leerzeichen am Pfadende,
    8.3-Kurznamen, `tar -x` ohne `-C` (Inhalt unbekannt), `git -C <dir>`
    (Pfade zählen ab dem Arbeitsordner), ein `(` in PowerShell
    (`Remove-Item (Join-Path …)`); Überverweigerungen: ein Schreibverb nach
    `;`, `|`, `&` oder `(` in Anführungszeichen, ein Heredoc-Körper, PowerShell
    faltet keine Braces, ein Glob, den PowerShell nicht auflöst.

  Der Absatz „**Paths are compared relative to the root.** … only
  slash-free patterns can still reach it“ bekommt die Ausnahme: „and the
  built-in rules under `**/`, which keep loomux's own files under any
  directory“. Unter den Überverweigerungen zusätzlich: eine Löschung mit
  Braces nach einem Ordner (`rm -rf .loomux/flows/{mine,zz}`) wird auch als
  Löschung des Ordners gelesen (E13).
  Der Absatz „The rules on loomux's own commands know the program by its file
  name …“ bekommt den Satz: „In strict mode they know it by its arguments
  instead.“ Der Absatz „The built-in path rules match in any case …“: „or a
  shell line that holds `flows`“ streichen (die Flow-Regel liest `[flow]` für
  jedes Ziel unter einem `.loomux/flows`-Ordner, einmal je Aufruf).
  `docs/de/hooks.md` entsprechend auf Deutsch.

- [ ] **Schritt 7: `flows.md` (en/de) und READMEs.**
  `grep -n "shell rule\|Shell-Regel\|by the shell\|per Shell" docs/en/flows.md docs/de/flows.md`
  und die Stellen auf „by the path rules, for a writing tool and a shell line
  alike“ umschreiben. `README.md` Z. ~313: „(`>`, `sed -i`, `tee`,
  `Set-Content`, `cp`/`mv` onto it)“ → „(every write or removal the guard
  reads from a shell line — redirections, `sed -i`, `tee`, `Set-Content`,
  `cp`/`mv`, `ln`, `tar`, `curl -o` and more — in any spelling a shell
  expands)“; `README.de.md` gleichsinnig. Danach
  `grep -rn "writeSource\|no shell command may write\|sudo -u root" docs README*.md internal`
  — kein Treffer außer in `docs/.superpowers/`.

- [ ] **Schritt 8: Migrationsplan prüfen.** `grep -n "guard\|Wächter" docs/en/migration.md docs/de/migration.md`:
  Diese Arbeit startet, beendet oder streicht keine Stufe; beide Dateien
  bleiben unverändert. Steht eine Fähigkeit „guard shell rules“ mit Status
  dort, ihren Status nachziehen und die Spec-Regel aus `AGENTS.md` beachten.
  Ergebnis in die Akte.

- [ ] **Schritt 9: Changelog-Block für den PR-Text** in die Akte (der
  `release-pr`-Skill übernimmt ihn; Label `release:minor`):

```markdown
## Changelog

### Added
- `[guard] mode = "strict"` holds the guard against deliberate workarounds: targets resolved by the file system (trailing dots and blanks, 8.3 short names, junctions), loomux's commands known by their arguments whatever the program is called, unknown programs and path expansions refused on protected paths.

### Changed
- A project's `[policy] paths` rules now hold for shell commands as well as for writing tools.
- The manifest has one refusal reason for writing tools and shell commands alike; a refusal names each reason once per call, and a bundled flow's reason names that one flow.
- loomux's own files are protected under any directory, a sibling worktree's `.loomux` included, for writing tools too.

### Fixed
- The guard reads the targets of `ln`, `tar`, `unzip`, `Expand-Archive`, `New-Item`, `touch`, `robocopy`, `xcopy`, `rsync`, `find -delete`, `curl -o`, `wget -O` and `Invoke-WebRequest -OutFile`, globs in any part of a path, brace expansion, wrapper flags with a value (`sudo -u root`, `xargs -n 1`, `timeout -s KILL 60`), a `cd` before a relative path, and the removal of `.loomux` or a folder above a protected path.
```

  Die Zeile `Release: minor — …` wie in `AGENTS.md`: „a new mode and
  stricter guard rules, compatible; a stricter rule is not listed as
  breaking“.

- [ ] **Schritt 10: Commit.**

```text
docs(guard): document the shell path rules and the strict mode

The rule table lists the paths loomux keeps under any directory and no
longer the shell expressions; the hooks page names the verbs, wrappers
and spellings the guard reads, the modes and the limits of the default
one. The benchmark case file adds a long shell line to the guard's cases.
```

---

## Selbstprüfung (erledigt beim Schreiben)

- **Spec-Abdeckung:** Lückentabelle default → Tasks 3/4a/4b; strikt →
  Task 6; Leser samt Verbtabelle, Unzerlegbar, Braces-Bruchstücke → Task 3;
  Normalisierung → Task 2, strikt `ResolvePath` → Task 6; eine Prüfung,
  `**/`, Manifest-Grund, Vorfahren → Tasks 1/4a/4b; bewusste Verschärfung
  samt Liste → Task 4b; Modus → Task 5/6; Grenzfälle (`judgedOrRefused`
  unverändert, Variablen benannt, Gründe einmal, `sh -c`) → 4b/6/9; Tests
  (Grundlinie, Lücken, strikt, Projektregeln, Schema) → 4a/4b/5/6; Messung →
  9; Doku → 5/9; flaky Tests → 7/8.
- **Typen:** `shellTarget{path, removes, shell}`, `shellWrites(root, line)
  ([]shellTarget, [][]string)`, `newJudge`, `judge.reasons`,
  `uniqueReasons`, `flowFolderReasons(rel, func() ([]string, error))` (ab
  4a), `writesConfiguration(line, anyProgram)` / `answersAGate(line,
  anyProgram)` (ab Task 6), `config.Policy.Strict` (ab Task 5) — in allen
  Tasks gleich benannt.
- **Offen gelassen, bewusst:** der Fix in Task 8 (Befund zuerst).
