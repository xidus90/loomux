# 4a nachziehen — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Was an 4a-1 und 4a-2 ohne Menschen noch offen ist, schließen:
Antigravity bekommt Hook-Einträge und Skills, der Wächter verweigert keine
Zeile mehr wegen eines `#` in einem gequoteten Pfad, `init` und `config`
überschreiben keine Datei, die sich seit dem Lesen geändert hat, und die
Akten sagen, was wirklich offen ist.

**Architecture:** Kein neues Paket. `internal/setup/hostfile` schreibt die
Form von `.agents/hooks.json` schon (Container `"loomux"` = benannte Gruppe);
die Regeln aus #21 der Fusions-Spec werden dort als Tests nachgetragen, nicht
`internal/agenthooks` umgezogen. Eine Änderung eines Plans trägt künftig das
Binary, das ihre Datei ruft (`Change.Binary`), und `Apply` lässt sie fallen,
wenn es fehlt — das ersetzt die Tabelle `callsBinary` und deckt `.mcp.json`
und die Antigravity-Einträge mit ab.

**Tech Stack:** Go 1.27, `encoding/json` als `map[string]any`, agy 1.2.11 für
die Probe.

**Spec:** `docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md`
(4a-2 „Host-Einträge“), Fusions-Spec #21, Messungen in
`docs/.superpowers/plans/2026-09-24-loomux-stufe-4a-2.md` („Messungen vor dem
Bau“). Entwurf im Chat freigegeben 2026-09-25.

**Stand 2026-09-25:** Auf Wunsch des Nutzers ohne Antigravity umgesetzt —
Tasks 1, 7, 8 und der `feat`-Teil von Task 6 sind zurückgestellt; der
`null`-Teil von Task 6 ist gebaut, weil er auch die Datei von Claude Code
betrifft.

## Global Constraints

- Coverage 100 % je Funktion; Ausnahme nur mit `//coverage:exempt <Grund>`
  direkt über `func`.
- Code, Kommentare, Meldungen, Commits englisch; dieser Plan und die Akten
  deutsch.
- Conventional Commits, Scope ist ein Codebereich (`guard`, `setup`,
  `config`), kein Arbeitspapier. Kein `Co-Authored-By` und keine
  Modellnennung. Commit-Text über eine Datei und `git commit -F`.
- Ein Commit je Thema. Jeder Fehler, der schon auf `master` stand, bekommt
  einen eigenen `fix`-Commit.
- `.loomux/config.toml` schreibt kein Agent; `loomux init` läuft nicht durch
  einen Agenten (der Wächter verweigert es).
- Kein Push durch einen Agenten. Vor jedem Commit Zweig und HEAD lesen.
- Gate: `sh ci/gate.sh` grün vor jedem Commit (der pre-commit-Hook fährt es).
- Antigravity-Befehle: `%LOCALAPPDATA%/loomux/bin/loomux.exe` **ohne**
  Anführungszeichen, **ohne** `--root` (Hook-Arbeitsverzeichnis ist
  `.agents/`, loomux sucht dann aufwärts). Gemessen 2026-09-24: agy startet
  Hooks über `cmd.exe`, `${…}` bleibt wörtlich, Anführungszeichen brechen.
- Kein Feld in einem Antigravity-Hook-Objekt, das nicht gemessen ist (#21).

## Review Focus

1. `LOCALAPPDATA` mit Leerzeichen (`C:\Users\Max Muster\AppData\Local`) —
   Erwartung: keine Antigravity-Einträge, eine Notiz, die den Grund nennt;
   Skills dürfen trotzdem entstehen. Test in Task 6.
2. Kanonisches Binary fehlt und der Binary-Schritt scheitert — Erwartung: weder
   `.agents/hooks.json` noch `.mcp.json` werden geschrieben (ein scheiternder
   Hook blockiert in agy jeden Schreibvorgang). Test in Task 5 und 6.
3. `.agents/hooks.json` mit fremder Gruppe, die schon loomux ruft (etwa eine
   Handkopie) — Erwartung: gemeldet, nicht angefasst, unsere Gruppe daneben.
   Test in Task 6.
4. Eine Datei ändert sich zwischen Plan und Bestätigung (ein zweites Terminal,
   ein Editor) — Erwartung: nicht überschrieben, Meldung mit Pfad, Exit ≠ 0.
   Tests in Task 3 und 4.
5. `#` in einem gequoteten Pfad einer anderen Befehlsstelle der Zeile — und
   ein `#`, der wirklich einen Kommentar öffnet, vor `--dry-run` — Erwartung:
   das erste geht durch, das zweite bleibt verweigert. Test in Task 2.

---

### Task 1: Abweichung von #21 in die Fusions-Spec

**Files:**
- Modify: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`
  (Zeile #21 der Lückentabelle, Spalte „Wohin“)
- Modify: `docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md`
  (Abschnitt „Host-Einträge“ unter 4a-2)

- [ ] **Step 1:** In #21 die Spalte „Wohin“ ergänzen um: „Nachtrag
  2026-09-25: `merge.go` zieht nicht um. `internal/setup/hostfile` schreibt
  dieselbe Form (Container `loomux` ist die Gruppe); übernommen werden die
  Regeln als Tests: Wurzel `null` abgelehnt mit Dateinamen, eine fremde
  Gruppe mit einem loomux-Befehl gemeldet, kein ungemessenes Feld im
  Hook-Objekt. Freigegeben 2026-09-25.“
- [ ] **Step 2:** In der Stufe-4-Spec unter „Host-Einträge“ die Regeln von #21
  und die Befehlsform (Global Constraints oben) eintragen; dazu: keine
  Einträge bei Leerzeichen in `LOCALAPPDATA`; Einträge fallen, wenn das
  kanonische Binary nach dem Binary-Schritt fehlt; vorerst nur `PreToolUse`
  und `PostToolUse` (ob `Stop`/`PreInvocation` außerhalb des Print-Modus
  feuern, ist nicht belegt).
- [ ] **Step 3: Commit** `docs(spec): write Antigravity host entries from hostfile, not a moved package`

### Task 2: Wächter — `#` in doppelten Anführungszeichen

**Files:**
- Modify: `internal/hooks/guard.go` (`plainLine`, Zweig `quote == '"'`, und
  Doc-Kommentar)
- Test: `internal/hooks/guard_test.go`
  (`TestTheGuardRefusesCommandsThatWriteTheConfiguration`)

Ursache: `plainLine` lässt in `"…"` nur `plainByte` zu; `#` ist keiner, die
ganze Zeile gilt als nicht plain, und kein Flag befreit mehr. Ein `#` in
doppelten Anführungszeichen ist für bash und PowerShell ein Zeichen wie jedes
andere. `flagOn` schneidet an einem Wort, das mit `#` beginnt — das kann nur
Flags wegnehmen, also nur mehr verweigern, nie weniger.

- [ ] **Step 1: Failing test.** In `allowed` ergänzen:

```go
		// A # inside double quotes is a byte of the word to both shells.
		`cd "C:/x/#GIT/loomux" && loomux init --dry-run`,
		`Set-Location "C:/x/#GIT/loomux"; loomux init --dry-run`,
		`loomux init --dry-run --root "C:/x/#GIT/loomux"`,
```

  In `refused` ergänzen:

```go
		// A quoted # word still ends the flags flagOn reads.
		`loomux init "#" --dry-run`,
		`loomux init "# x" --dry-run`,
```

- [ ] **Step 2:** `go test ./internal/hooks -run TestTheGuardRefusesCommandsThatWriteTheConfiguration` — FAIL an den drei `allowed`-Zeilen.
- [ ] **Step 3: Implementierung.** In `plainLine`:

```go
		case quote == '"':
			if c == '"' {
				quote = 0
			} else if !plainByte(c) && c != '#' {
				return false
			}
```

  Doc-Kommentar: „Inside double quotes only plainByte and # may stand …; a
  quoted # reaches the program inside a word, and a word that begins with it
  only ends the flags flagOn reads.“
- [ ] **Step 4:** Test grün; `go test ./internal/hooks`.
- [ ] **Step 5:** Nachprüfen mit dem gebauten Binary (siehe Akte 4a-2,
  „Offen“): die Zeile `cd "C:/x/#GIT/loomux" && bin/loomux.exe init --dry-run`
  als Nutzlast an `hook pre-tool-use --host claude --root .` — Exit 0.
- [ ] **Step 6: Commit** `fix(guard): let a # inside double quotes keep a line plain`

### Task 3: `init` überschreibt nichts, was sich seit dem Plan geändert hat

**Files:**
- Modify: `internal/setup/apply.go` (`change`, `Apply`)
- Test: `internal/setup/apply_test.go`

Drei Fehler auf `master`, ein Thema (was `Apply` über einen gescheiterten
Lauf behauptet):
- (a) Eine Änderung ohne `Redo`, deren Datei nach dem Plan ein anderer
  geändert hat, wird mit dem alten `After` überschrieben; die Sicherung hält
  den Text zur Planzeit.
- (b) Ein gescheitertes Schreiben stoppt den Lauf, steht aber nicht in
  `Report.Failed`.
- (c) `installed.toml` entsteht auch, wenn ein Schritt gescheitert ist.

- [ ] **Step 1: Failing tests.**

```go
func TestAFileChangedSinceThePlanIsNotOverwritten(t *testing.T) {
	root := world(t, map[string]string{".gitignore": "a\n"})
	p := Plan{Changes: []Change{{Part: "gitignore", Path: ".gitignore", Before: "a\n", After: "a\nb\n", Exists: true}}}
	writeFile(t, root, ".gitignore", "someone else\n")
	report, err := Apply(root, p, Choice{}, all, nil, there, "v", time.Now())
	if err == nil || !strings.Contains(err.Error(), ".gitignore") {
		t.Fatalf("err = %v, want one naming .gitignore", err)
	}
	if got := read(t, root, ".gitignore"); got != "someone else\n" {
		t.Fatalf(".gitignore = %q, want the other writer's text", got)
	}
	if !slices.Contains(report.Failed, ".gitignore") {
		t.Fatalf("Failed = %v, want .gitignore", report.Failed)
	}
}

func TestAFailedStepLeavesNoInstalledRecord(t *testing.T) {
	root := world(t, nil)
	p := Plan{Actions: []Action{{Part: "binary", ID: "binary-install"}},
		Changes: []Change{{Part: "gitignore", Path: ".gitignore", After: "x\n"}}}
	fail := func(Action) error { return errors.New("offline") }
	report, err := Apply(root, p, Choice{}, all, fail, there, "v", time.Now())
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !slices.Contains(report.Failed, "binary-install") || exists(root, installedPath) {
		t.Fatalf("Failed = %v, installed.toml there = %v; want the failure and no record",
			report.Failed, exists(root, installedPath))
	}
	if !exists(root, answersPath) {
		t.Fatal("answers.toml missing; the choices of the run are kept regardless")
	}
}
```

  `TestApplyStopsAtAFailedWrite` (vorhanden, `apply_test.go:208`) um die
  Prüfung ergänzen, dass der Pfad in `report.Failed` steht.
- [ ] **Step 2:** `go test ./internal/setup -run 'TestAFileChanged|TestAFailedStep|TestApplyStopsAtAFailedWrite'` — FAIL.
- [ ] **Step 3: Implementierung.**
  - In `change`, nach `a.redo(ch)`, für eine Änderung ohne `Redo` und
    `ch.Exists`: die Datei lesen (`readOptional`), und wenn der Text nicht
    `ch.Before` ist:
    `return fmt.Errorf("%s changed since the plan was made; nothing written", ch.Path)`.
  - In `Apply` im `default`-Zweig der Änderungsschleife: bei Fehler zuerst
    `a.report.Failed = append(a.report.Failed, ch.Path)`, dann zurück.
  - `installed.toml` nur, wenn `len(a.report.Failed) == 0` und etwas
    geschrieben oder gelaufen ist. Doc-Kommentar von `Apply` und von
    `installedPath` angleichen: „written last, and only by a run in which
    nothing failed“.
- [ ] **Step 4:** Tests grün; `go test ./internal/setup ./internal/cli`.
  Tests, die `installed.toml` nach einem gescheiterten Binary-Schritt
  erwarten, sind nach (c) falsch und werden angepasst — im Commit-Text
  nennen.
- [ ] **Step 5: Commit** `fix(setup): refuse to overwrite a file that changed after the plan`
  (Rumpf nennt (a), (b), (c)).

### Task 4: `config` überschreibt nichts, was sich seit dem Lesen geändert hat

**Files:**
- Modify: `internal/cli/config.go` (`writeConfig` und seine zwei Aufrufer
  `:356`, `config_ui.go:134`; `apply` in `config_proposals.go`, falls es
  `writeConfig` nicht nutzt — vorher lesen)
- Test: `internal/cli/config_test.go`, `internal/cli/config_ui_test.go`

- [ ] **Step 1: Failing test.** `config set` mit einem stdin, dessen Leser die
  Datei beim Lesen der Antwort ändert:

```go
// changingReader writes path when the answer is read, the way a second
// terminal would between the diff and the confirmation.
type changingReader struct {
	path, text string
	done       bool
}

func (r *changingReader) Read(p []byte) (int, error) {
	if !r.done {
		r.done = true
		os.WriteFile(r.path, []byte(r.text), 0o644)
	}
	return copy(p, "y\n"), io.EOF
}

func TestConfigSetRefusesAFileChangedBeforeTheAnswer(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".loomux", "config.toml")
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte("[commit]\nthreshold = 2\n"), 0o644)
	var stderr bytes.Buffer
	in := &changingReader{path: path, text: "[commit]\nthreshold = 5\n"}
	code := runConfigWith(t, in, &stderr, "--root", root, "set", "commit.threshold", "3")
	if code == 0 || !strings.Contains(stderr.String(), "changed since it was read") {
		t.Fatalf("code %d, stderr %q; want a refusal", code, stderr.String())
	}
	if got, _ := os.ReadFile(path); string(got) != "[commit]\nthreshold = 5\n" {
		t.Fatalf("file = %q, want the other writer's text", got)
	}
}
```

  `runConfigWith` ist der vorhandene Testhelfer für `config` mit stdin —
  vorher in `config_test.go` nachsehen und dessen Namen nehmen.
  Für die interaktive Form: derselbe Fall über `tui.Scripted` in
  `config_ui_test.go`, die Datei im Skript zwischen Diff und `y` geändert;
  erwartet: Fehler, Datei unverändert.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implementierung.**

```go
// writeConfig is the one way every form puts a new text in place: set,
// unset, the interactive form and apply. before is the text next was made
// from; a file that no longer holds it was changed by someone else since,
// and writing next would take that change back without a word.
func writeConfig(t configTarget, before, next string) error {
	current, err := t.read()
	if err != nil {
		return err
	}
	if current != before {
		return fmt.Errorf("%s changed since it was read; nothing written", t.path)
	}
	err = os.MkdirAll(filepath.Dir(t.path), 0o755)
	if err == nil {
		err = lock.ReplaceText(t.path, next)
	}
	return err
}
```

  Die Aufrufer reichen `text` durch. `t.read()` gibt für eine fehlende Datei
  `""` — vorher prüfen; wenn nicht, dieselbe Semantik herstellen.
- [ ] **Step 4:** Tests grün; `go test ./internal/cli -run Config`.
- [ ] **Step 5: Commit** `fix(config): refuse to write over a file changed after it was read`

### Task 5: Eine Änderung trägt das Binary, das ihre Datei ruft

**Files:**
- Modify: `internal/setup/plan.go` (`Change` bekommt `Binary string`;
  `mcpJSON`, `hostEntries`, `gitHooks` setzen es)
- Modify: `internal/setup/apply.go` (`callsBinary` entfällt; `dropped`
  urteilt über `Change.Binary` bzw. `Action.Part`)
- Test: `internal/setup/apply_test.go`, `internal/setup/plan_test.go`

Fehler auf `master`: `.mcp.json` ruft `${LOCALAPPDATA}/loomux/bin/loomux.exe`,
entfällt aber nicht, wenn dieses Binary fehlt.

- [ ] **Step 1: Failing tests.**

```go
func TestMCPJSONIsDroppedWithoutTheInstalledBinary(t *testing.T) {
	root := world(t, nil)
	t.Setenv("LOCALAPPDATA", t.TempDir()) // no loomux.exe there
	p := Plan{Changes: []Change{{Part: "mcp-json", Path: ".mcp.json", After: "{}\n", Binary: hostfile.Canonical}}}
	report, err := Apply(root, p, Choice{}, all, nil, there, "v", time.Now())
	if err != nil || exists(root, ".mcp.json") || !slices.Contains(report.Failed, ".mcp.json") {
		t.Fatalf("err %v, written %v, failed %v; want .mcp.json dropped", err, exists(root, ".mcp.json"), report.Failed)
	}
}
```

  In `plan_test.go`: `changeOf(p, ".mcp.json").Binary == hostfile.Canonical`,
  `changeOf(p, ".claude/settings.json").Binary == f.Binary`, ein Git-Hook
  trägt `f.Binary`.
- [ ] **Step 2:** FAIL (Feld fehlt: Übersetzungsfehler zählt als FAIL).
- [ ] **Step 3: Implementierung.**
  - `Change.Binary`: „the binary the written file calls, hostfile.Canonical
    or hostfile.Checkout; "" for a file that calls none“.
  - Neuer Helfer `func (b *builder) add(c Change)`, der eine nicht leere
    Änderung anhängt; `redoable` ruft ihn. `hostEntries`, `mcpJSON` und
    `gitHooks` bauen ihre `Change` samt `Binary` selbst und rufen `b.add`.
    `mcpJSON` setzt `hostfile.Canonical` (`.mcp.json` ruft immer den
    kanonischen Ort, siehe E5), `gitHooks` und `hostEntries` setzen
    `b.f.Binary`.
  - In `Apply` ersetzt dies `callsBinary` und `dropped`:

```go
	present := binaryThere()
	canonical := isFile(BinaryPath(root, hostfile.Canonical))
	// missing says whether the binary a file or action calls is absent;
	// what calls it would call nothing.
	missing := func(binary string) bool {
		switch binary {
		case "":
			return false
		case hostfile.Canonical:
			return !canonical
		}
		return !present
	}
	// An action names no binary of its own: merge-hook's hook calls the
	// installed one in every project, hooks-path goes with the git hooks.
	actionMissing := func(part string) bool {
		switch part {
		case "merge-hook":
			return missing(hostfile.Canonical)
		case "git-hooks":
			return !present
		}
		return false
	}
```

    Die Änderungsschleife prüft `missing(ch.Binary)`, `a.actions` bekommt
    `actionMissing`.
- [ ] **Step 4:** `go test ./internal/setup ./internal/cli`; vorhandene Tests
  `TestAMissingBinaryDropsWhatCallsIt` und die Merge-Hook-Tests bleiben grün.
- [ ] **Step 5: Commit** `fix(setup): drop .mcp.json with the binary it calls`

### Task 6: Antigravity-Einträge

**Files:**
- Modify: `internal/setup/hostfile/table.go` (Konstante `Antigravity`,
  `entries` für Antigravity, Doc)
- Modify: `internal/setup/hostfile/merge.go` (`null`-Wurzel ablehnen;
  fremde Gruppen mit loomux-Befehl melden; Timeout nur, wo gemessen)
- Modify: `internal/setup/facts.go` (`Facts.LocalAppData string`, in
  `Gather` aus `os.Getenv("LOCALAPPDATA")`)
- Modify: `internal/setup/plan.go` (Zielauswahl: Einträge und Skills getrennt)
- Test: `internal/setup/hostfile/table_test.go`, `merge_test.go`,
  `internal/setup/plan_test.go`, `apply_test.go`

**Interfaces:**
- Produces: `const Antigravity = "%LOCALAPPDATA%/loomux/bin/loomux.exe"`;
  `Entries(hosts.HostAntigravity, _)` gibt — solange `antigravityMeasured`
  wahr ist — die zwei Einträge mit diesem Binary zurück, unabhängig vom
  übergebenen Binary; `BinaryFor(host, binary string) string` (Antigravity →
  `Canonical` für das Urteil in `Apply`, sonst `binary`).
- `Result.Notes` bekommt je fremder Gruppe mit loomux-Befehl eine Zeile
  `group "<name>" runs loomux too; its hooks now fire twice`.

- [ ] **Step 1: Failing tests (table).** `TestEntriesForAntigravityWaitForTheMeasurement`
  ersetzen:

```go
func TestAntigravityEntriesCallTheInstalledBinaryAsCmdReadsIt(t *testing.T) {
	writers := "write_to_file|replace_file_content|multi_replace_file_content"
	want := []Entry{
		{Event: "PreToolUse", Matcher: writers + "|run_command",
			Command: Antigravity + " hook pre-tool-use --host antigravity"},
		{Event: "PostToolUse", Matcher: writers,
			Command: Antigravity + " hook post-tool-use --host antigravity"},
	}
	for _, b := range []string{Canonical, Checkout} {
		if got := entries(hosts.HostAntigravity, b, true); !reflect.DeepEqual(got, want) {
			t.Fatalf("entries(antigravity, %s) =\n%v\nwant\n%v", b, got, want)
		}
	}
	if got := entries(hosts.HostAntigravity, Canonical, false); got != nil {
		t.Fatalf("unmeasured: %v, want nil", got)
	}
	if !Owned(want[0].Command) {
		t.Fatal("an Antigravity entry must count as ours")
	}
}
```

  (Timeout 0 bis Task 7 entscheidet.)
- [ ] **Step 2: Failing tests (merge).**

```go
func TestMergeRefusesANullRoot(t *testing.T) {
	for _, h := range []hosts.Host{hosts.HostClaude, hosts.HostAntigravity} {
		_, err := Merge(h, []byte("null"), Entries(h, Canonical))
		if err == nil || !strings.Contains(err.Error(), Path(h)) {
			t.Errorf("%s: err = %v, want one naming %s", h, err, Path(h))
		}
	}
}

func TestMergeNamesAForeignGroupThatRunsLoomux(t *testing.T) {
	existing := []byte(`{"hand-made":{"PreToolUse":[{"matcher":"run_command",
		"hooks":[{"type":"command","command":"loomux hook pre-tool-use --host antigravity"}]}]},
		"wiki-guard":{"PreToolUse":[{"matcher":"write_to_file","hooks":[{"type":"command","command":"brain guard"}]}]}}`)
	got, err := Merge(hosts.HostAntigravity, existing, entries(hosts.HostAntigravity, Canonical, true))
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(got.Added) != 2 {
		t.Fatalf("added %v, want both entries in our own group", got.Added)
	}
	if !slices.ContainsFunc(got.Notes, func(n string) bool { return strings.Contains(n, `"hand-made"`) }) ||
		slices.ContainsFunc(got.Notes, func(n string) bool { return strings.Contains(n, "wiki-guard") }) {
		t.Fatalf("notes %v, want hand-made named and wiki-guard not", got.Notes)
	}
	if !bytes.Contains(got.Merged, []byte(`"brain guard"`)) || !bytes.Contains(got.Merged, []byte(`"hand-made"`)) {
		t.Fatalf("a foreign group went missing:\n%s", got.Merged)
	}
}
```

- [ ] **Step 3: Failing tests (plan, ohne Schalter).** In `plan_test.go`,
  Fakten mit `Hosts: {antigravity}`:
  - `LocalAppData: "C:/Users/Max Muster/AppData/Local"` → keine Änderung an
    `.agents/hooks.json`, eine Notiz mit `LOCALAPPDATA` und `blank`.
  - `LocalAppData: "C:/Users/micro/AppData/Local"` → noch keine Änderung,
    die Notiz `the hook file is being measured` (bis Task 7).
  In `apply_test.go`: eine Änderung an `.agents/hooks.json` mit
  `Binary: hostfile.Canonical` ohne kanonisches Binary → `Failed`, Datei
  nicht geschrieben (Review Focus 2).
- [ ] **Step 4:** FAIL.
- [ ] **Step 5: Implementierung.**
  - `table.go`: `Antigravity`-Konstante mit Doc (Messung 2026-09-24: cmd.exe,
    `%…%`, keine Anführungszeichen, Schrägstriche gehen); Antigravity-Zweig
    von `entries` ohne `--root` und ohne Timeout. `antigravityMeasured` bleibt
    in diesem Task falsch; Task 7 legt ihn nach der Probe um. Die Plan-Tests
    aus Step 3 brauchen den umgelegten Schalter und werden darum in Task 7,
    Step 5 geschrieben; dieser Task prüft `hostfile`, `facts` und die Notiz
    für einen noch nicht gemessenen Wirt. Die Leerzeichen-Prüfung steht in
    `plan.go` vor dem Aufruf von `Entries` und ist unabhängig vom Schalter
    testbar: mit `LocalAppData` mit Leerzeichen erscheint die
    Leerzeichen-Notiz, nicht die Mess-Notiz.
  - `merge.go`: nach `json.Unmarshal` bei `root == nil && len(existing) > 0`:
    `return Result{}, fmt.Errorf("%s is not a JSON object: its root is null", file)`.
    Für Antigravity (`key != "hooks"`): jede andere oberste Ebene, deren
    Wert ein Objekt von Ereignislisten ist, auf Blöcke mit `Owned`-Befehlen
    durchsuchen und je Name eine Notiz; sortiert.
  - `facts.go`: `LocalAppData` setzen.
  - `plan.go`: `targets` zerfällt in `entryHosts` (Hookdatei und Einträge,
    bei Antigravity zusätzlich `!strings.ContainsAny(f.LocalAppData, " \t")`,
    sonst Notiz `antigravity: no hook entries; LOCALAPPDATA holds a blank,
    which the unquoted command cmd.exe runs cannot carry`) und `skillHosts`
    (jeder Host mit Skill-Ort). Die bisherige Notiz „expansion of
    ${LOCALAPPDATA} is not measured“ entfällt; solange
    `antigravityMeasured` falsch ist, heißt sie `antigravity: no hook entries
    yet; the hook file is being measured`.
  - `hostEntries` setzt `Change.Binary = hostfile.BinaryFor(h, b.f.Binary)`.
- [ ] **Step 6:** `go test ./internal/setup/...` grün.
- [ ] **Step 7: Commit (fix)** `fix(setup): refuse a hook file whose root is null`
  — nur der `null`-Teil samt Test, vorher mit `git add -p` getrennt.
- [ ] **Step 8: Commit (feat)** `feat(setup): write Antigravity hook entries in the form cmd.exe runs`

### Task 7: Probe gegen agy, dann umlegen

**Files:**
- Create: `internal/setup/hostfile/probe_test.go` (nur mit
  `LOOMUX_AGY_PROBE=<dir>`; schreibt `.agents/hooks.json` über `Merge` in
  `<dir>` — der Weg, auf dem ein Agent die Datei ohne `init` bekommt)
- Modify: `internal/setup/hostfile/table.go` (`antigravityMeasured = true`,
  ggf. Timeout)
- Modify: Akte `docs/.superpowers/parity/stufe-4a-2.md` („Messungen“)
- Test: die Plan-Tests aus Task 6, Step 3

- [ ] **Step 1: Welt.** Im Scratchpad ein Repo `agyprobe` mit
  `.loomux/config.toml` aus der Welt eines vorhandenen Tests (eine Pfadregel,
  die `blocked/**` verweigert; Vorlage in `internal/hooks/guard_test.go`
  suchen). `.loomux/config.toml` ist dort eine Datei der Wegwerf-Welt, nicht
  die des Projekts.
- [ ] **Step 2: Hookdatei.** `LOOMUX_AGY_PROBE=<agyprobe> go test ./internal/setup/hostfile -run TestWriteProbeHooks`
  mit einem Test, der `entries(HostAntigravity, Canonical, true)` in
  `<agyprobe>/.agents/hooks.json` merged und schreibt; ohne die Variable
  `t.Skip`. Variante B desselben Tests mit Timeout 15/60 in einer zweiten
  Welt `agyprobe-timeout`.
- [ ] **Step 3: Lauf.** Ordner in agy vertrauen (Befund 3 der Messung vom
  2026-09-10 — falls der Mensch das tun muss, hier anhalten und fragen), dann
  je Welt:
  `agy -p --add-dir <welt> "Create the file blocked/x.txt with the text hi, then create ok/y.txt with the text hi."`
  Erwartet: `blocked/x.txt` fehlt, agy nennt die Verweigerung des
  Hooks; `ok/y.txt` entsteht. In beiden Welten gleich → Timeout wird
  geschrieben (15/60 wie bei Claude); sonst ohne.
- [ ] **Step 4:** Ergebnis mit Datum, agy-Version und wörtlicher Meldung in
  die Akte. **Scheitert die Probe** (kein Hook feuert, oder `ok/y.txt` wird
  blockiert): `antigravityMeasured` bleibt falsch, Befund in die Akte, Task 8
  läuft trotzdem (Skills hängen nicht an Hooks), und der Mensch entscheidet.
- [ ] **Step 5:** Bei Erfolg `antigravityMeasured = true`, Doc-Kommentar mit
  Datum und Probe. Plan-Test: `LocalAppData: "C:/Users/micro/AppData/Local"`
  → Änderung an `.agents/hooks.json` mit `Binary == hostfile.Canonical` und
  den zwei Befehlen aus Task 6; die Mess-Notiz ist fort. Der Test
  `TestAntigravityEntriesCallTheInstalledBinaryAsCmdReadsIt` prüft dann auch
  `Entries(hosts.HostAntigravity, Canonical)`; grün.
- [ ] **Step 6:** Der Commit wird in den `feat`-Commit von Task 6 gefaltet
  (`git commit --fixup` und `rebase --autosquash` beim Gruppieren durch
  `release-pr`).

### Task 8: Skills für Antigravity

**Files:**
- Modify: `internal/setup/templates/templates.go` (`Skills`)
- Test: `internal/setup/templates/templates_test.go`,
  `internal/setup/plan_test.go`

`brain-research` verlangt „`model: opus` and `effort: low`“ für einen
Subagenten — eine Anweisung, die nur Claude Code befolgen kann. Die anderen
fünf Skills sind wirtsneutral (geprüft 2026-09-25 per `grep -i
"claude|opus|effort|subagent"`: nur „No subagent“-Sätze, die überall gelten).

- [ ] **Step 1: Failing test.**

```go
func TestAntigravitySkillsLiveUnderAgents(t *testing.T) {
	files, err := Skills([]string{"verify-until-green", "brain-research", "brain-land"}, hosts.HostAntigravity)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range files {
		got = append(got, f.Path)
	}
	want := []string{".agents/skills/verify-until-green/SKILL.md", ".agents/skills/brain-land/SKILL.md"}
	if !slices.Equal(got, want) {
		t.Fatalf("paths %v, want %v (brain-research is Claude's alone)", got, want)
	}
}
```

  Plan-Test: ein Antigravity-Wirt bekommt mit `brain-skills` eine Notiz
  `brain-research: not written for antigravity; it dispatches a Claude Code subagent`.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implementierung.** `case hosts.HostAntigravity: root =
  ".agents/skills"` (Kommentar: Ort aus den Hilfetexten von agy 1.2.8,
  gemessen 2026-09-24); `ClaudeOnly(name string) bool` für
  `brain-research`; `Skills` überspringt es für Antigravity; `plan.skills`
  notiert es.
- [ ] **Step 4:** `go test ./internal/setup/...` grün.
- [ ] **Step 5: Commit** `feat(setup): lay the skills for Antigravity under .agents/skills`

### Task 9: Kleinigkeiten sortieren — anhalten

**Files:**
- Modify: `docs/.superpowers/parity/stufe-4a-1.md`, `stufe-4a-2.md`
  (Abschnitte „Aufgeschobene Kleinigkeiten“)

- [ ] **Step 1:** Jeden Punkt beider Listen gegen den Code prüfen (Datei und
  Zeile öffnen, nicht dem Text glauben). Erledigte streichen mit Commit.
- [ ] **Step 2:** Die übrigen in drei Gruppen eintragen: **Fehler**
  (Verhalten falsch für einen Nutzer), **Testlücke**, **fällt weg**
  (wie die Referenz, Prozessnotiz, nicht erreichbar) — je mit einem Satz
  Begründung.
- [ ] **Step 3: Anhalten.** Die Liste dem Menschen vorlegen. Umgesetzt wird
  nur, was er auswählt, als eigene Tasks nach dem Muster oben.

### Task 10: Doku nachziehen

**Files:**
- Modify: `docs/.superpowers/parity/stufe-4a-1.md`, `stufe-4a-2.md`
- Modify: `docs/en/migration.md`, `docs/de/migration.md`
- Modify: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`
  (Stufenzeile 4a-2)
- Modify: `README.md`, `README.de.md` und die Befehlsreferenz unter
  `docs/en/`, `docs/de/`, wo `init` und Antigravity stehen (vorher
  `grep -rn -i antigravity README* docs/en docs/de`)

- [ ] **Step 1: Akte 4a-2.** „Offen“: Task 1 ist gelaufen (2026-09-24, siehe
  Plan) bis auf den Freigabeweg einer Projekt-`.mcp.json`; die Git-Hook-Probe
  vom 2026-09-25 als Nachbestätigung (`C:\Users\micro\AppData\Local`, Binary
  gefunden). Entscheidungen 1 und 2 (Antigravity ohne Einträge/Skills) als
  aufgehoben markieren, mit Verweis auf Task 6–8 und die Probe. Die
  Fehlverweigerung mit `#` als behoben.
- [ ] **Step 2: Akte 4a-1.** Menschenschritte bleiben; nichts sonst.
- [ ] **Step 3: migration.md EN/DE.** Zeile 4a-2: Antigravity-Einträge und
  Skills gebaut (oder, bei gescheiterter Probe, gemessen und offen); der Satz
  „`internal/agenthooks` moved over“ wird durch die tatsächliche Lösung
  ersetzt. Offen bleiben: MCP-Arbeitsverzeichnis, drei Terminals,
  `config set` mit `n`, frischer Klon, ein Wirt interaktiv samt Freigabe der
  `.mcp.json`. Fähigkeit „Brain and Verify Skills“: Antigravity-Ort
  gemessen.
- [ ] **Step 4: README/Referenz.** Was `init` für Antigravity schreibt, und
  warum der Befehl ohne Anführungszeichen steht.
- [ ] **Step 5: Commit** `docs: record Antigravity entries and what stays with a human`

### Abschluss

- [ ] `sh ci/gate.sh` grün im sauberen Stand.
- [ ] Mutationsrunde über die geänderten Funktionen
  (`loomux dev mutants --only` für `guard.go`, `apply.go`, `merge.go`,
  `table.go`, `templates.go`); Überlebende mit Begründung in die Akte 4a-2.
- [ ] `release-pr`-Skill: Commits gruppieren, Label `release:minor`, Body mit
  `## Changelog` (Added: Antigravity entries and skills; Fixed: guard `#`,
  null root, overwrite after plan/read, `.mcp.json` without binary). Push-Befehl
  dem Menschen nennen.
