# 4a-Kleinigkeiten — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans
> (Native, vom Nutzer so gewählt: „Nimm jeweils deine Empfehlung und leg los“).
> Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die aufgeschobenen Fehler, Schönheitsfehler und Testlücken aus den
Akten `stufe-4a-1.md` und `stufe-4a-2.md` schließen, die am 2026-09-25 per
Probetest gegen `origin/master` (v2.14.0) bestätigt wurden.

**Architecture:** Lauter kleine Eingriffe an bestehendem Code; eine neue
Eingabeform (Listenelemente in Anführungszeichen) bekommt einen gemeinsamen
Kodierer in `internal/config/schema`, weil `edit` `schema` importiert und
beide Richtungen übereinstimmen müssen.

**Tech Stack:** Go 1.27, `golang.org/x/text/width` (schon direkte Abhängigkeit).

**Spec:** keine eigene; die Befunde stehen in
`docs/.superpowers/parity/stufe-4a-1.md` („Offene Punkte“) und
`docs/.superpowers/parity/stufe-4a-2.md` („Offen“). Entscheidungen des
Menschen vom 2026-09-25 (jeweils die Empfehlung):

1. Ein Modul, das an ist, ohne dass ein Teil an ist, wird mit `each`
   angeboten; `none` heißt immer „Modul aus“.
2. `unset` entfernt einen Abschnittskopf nur, wenn der Rumpf wirklich leer ist;
   stehen dort noch Kommentare, bleibt der Kopf.
3. Ein Listenelement mit Komma steht in TOML-Anführungszeichen (`"a,b", c`);
   ein leeres Element ohne Anführungszeichen fällt weg, `""` bleibt.
4. Eine fremde Hookdatei bei `not installed` heißt `[<pfad>: another hook]`;
   Zustand und Exit-Code bleiben.

## Global Constraints

- Coverage 100 % je Funktion; Ausnahme nur mit `//coverage:exempt <Grund>`.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst.
- Code, Kommentare, Meldungen, Commits englisch; Conventional Commits, kein
  Arbeitspapier im Commit-Text, kein Modell als Mitautor.
- Jeder Punkt ist ein Fehler, der schon auf `master` stand: jeder behält
  seinen eigenen Commit. Die neue Listenform ist `feat(config)`; der PR trägt
  damit mindestens `release:minor`.
- Tests mit Backslashes nur über Write/Edit, nie per Heredoc.
- Tor: `sh ci/gate.sh`; vor einer Aussage über den Wächter `bin/loomux.exe`
  neu bauen.

## Review Focus

- `config --root DIR set k v` war bisher ein Usage-Fehler; nach Task 1 ist es
  ein Schreibbefehl. Der Wächter muss ihn einem Agenten weiter verweigern
  (Test in Task 1).
- Rundlauf der Listenform: jeder Wert, den `inputForm` anzeigt, muss durch
  `Render` wieder denselben TOML-Wert ergeben — auch für `{a,b}`, `"`, führende
  Leerzeichen, `""` (Test in Task 3).
- Ein Modul im loomux-Checkout (Graph an, nicht gebaut) bleibt nach Enter auf
  jeder Frage an (Test in Task 5).
- `unset` darf nie einen Kommentar entfernen oder unter einen fremden Kopf
  hängen (Test in Task 6).
- `samePath` darf unter Windows weiter ohne Groß-/Kleinschreibung vergleichen
  (Test in Task 9).

---

### Task 1: `config` nimmt Flags vor dem Unterbefehl und verweigert fremde

**Files:** `internal/cli/config.go`, `internal/cli/config_test.go`,
`internal/hooks/guard_test.go`, `docs/en/cli-reference.md`,
`docs/de/cli-reference.md`

- [x] Test: `configCommand([]string{"--root", dir, "list"}, …)` gibt 0 und
  listet; `get commit.language --yes` gibt 2; `list --yes`, `get --json`,
  `proposals --yes`, `apply x --json` geben 2; `set k v --yes` bleibt 0.
- [x] Test im Wächter: `loomux config --root . set commit.language en` und
  `loomux config --global unset x` werden einem Agenten verweigert.
- [x] Umsetzung: Nach `parseInterspersed`, wenn `sub == ""` und
  `len(positional) > 0`, `sub, positional = positional[0], positional[1:]`.
  Welche Flags gesetzt wurden, liest `flags.Visit`; erlaubt sind je
  Unterbefehl: überall `root`, `global`; `json` bei `list`, `proposals`;
  `yes` bei `set`, `unset`, `apply`; `propose` bei `set`, `unset`; `all` bei
  `apply`, `reject`; der bare Aufruf nur `root`, `global`.
- [x] Doku: der Satz über `config --root <dir> list` als bekannte
  Fehlverweigerung sagt, dass die Form jetzt gilt und der Wächter sie
  trotzdem verweigert (Unterbefehl zuerst nennen).
- [x] Commit `fix(config): read flags before the subcommand and refuse flags
  a subcommand does not take`.

### Task 2: `yes` bestätigt wie `y`

**Files:** `internal/cli/config.go`, `internal/cli/config_proposals.go`, Tests.

- [x] Test: `set` mit stdin `yes\n` und `YES\n` schreibt; `apply` ebenso;
  `n`, `no`, leer lehnen ab.
- [x] Umsetzung: `func confirmed(answer string) bool` (`y` oder `yes`, ohne
  Groß-/Kleinschreibung, getrimmt), an beiden Stellen.
- [x] Commit `fix(config): take yes as a confirmation`.

### Task 3: Listenform — leeres Element, Anführungszeichen

**Files:** Create `internal/config/schema/listinput.go` (+ Test); Modify
`internal/config/schema/current.go` (`inputForm`),
`internal/config/edit/value.go` (`Render`, `splitList` entfällt), Doku.

**Interfaces:** `schema.SplitList(input string) ([]string, error)`,
`schema.JoinList(items []string) string`.

- [x] 3a Test: `Render(StringList, "a,")` → `["a"]`, `" , "` → `[]`.
  Umsetzung in `splitList`: getrimmte leere Teile fallen weg. Commit
  `fix(config): drop an empty item from a typed list`.
- [x] 3b Test: `SplitList(`"a,b", c`)` → `a,b`, `c`; `"x\"y"` → `x"y`;
  `docs/*.{md,txt}` bleibt ein Element; `""` → ein leeres Element;
  unvollständiges `"a` → Fehler. `JoinList` setzt ein Element in
  Anführungszeichen (per `config.QuoteTOML`), wenn `SplitList` es nicht als
  ein Element zurückgäbe: leer, Komma außerhalb `{…}`, beginnt mit `"`,
  führende oder folgende Leerzeichen. Rundlauftest `SplitList(JoinList(x))
  == x` über diese Fälle und über jede `StringList`-Vorgabe.
- [x] Umsetzung: `SplitList` läuft über die Zeichen; `{`/`}` wie bisher;
  ein Teil, der getrimmt mit `"` beginnt, wird als TOML-Basisstring gelesen
  (`toml.Unmarshal("v = "+teil)`), Rest nach dem schließenden `"` darf nur
  Leerraum sein. `Render` nutzt `SplitList`, `inputForm` `JoinList`.
- [x] Doku (`cli-reference` en/de, „Values“): `"a,b", c` und `""`.
- [x] Commit `feat(config): quote a list item that holds a comma`.

### Task 4: Vorgabe des Scopes ohne Leerraum

**Files:** `internal/setup/parts.go`, `internal/setup/*_test.go`.

- [x] Test: `DefaultChoice(Facts{Root: …"my  project"})` → Scope
  `project/my-project`; `checkScope` nimmt ihn an.
- [x] Umsetzung: `strings.Join(strings.Fields(filepath.Base(root)), "-")`.
- [x] Commit `fix(setup): derive a default scope without spaces`.

### Task 5: `none` schaltet ein Modul immer aus

**Files:** `internal/cli/init_ui.go`, `internal/cli/init_test.go`.

- [x] Tests: `preset` gibt `each`, wenn das Modul an und kein Teil an ist;
  `none`, wenn das Modul aus ist. `interview` mit Enter auf jeder Frage lässt
  im Checkout-Fall das Graph-Modul an (die Teile-Auswahl wird mit Enter
  übernommen). Wer `none` tippt, schaltet das Modul aus, auch wenn es
  angeboten war.
- [x] Umsetzung: `preset`: `moduleOff` → `none`; `on == 0` → `each`;
  `interview`: Sonderfall `answer == offered && answer != "each"` entfällt
  für `none` (angenommenes `all` bleibt ohne Wirkung wie bisher).
- [x] Commit `fix(init): switch a module off whenever none is taken`.

### Task 6: `unset` behält Kopf und Kommentare

**Files:** `internal/config/edit/edit.go`, `edit_test.go`, Doku.

- [x] Test: `Remove("[commit]\n…\n[verify]\n# note\nbudget = 5\n", "verify",
  "budget")` behält `[verify]` und `# note`; ohne Kommentar fällt der Kopf wie
  bisher.
- [x] Umsetzung: vor dem Entfernen des Kopfs prüfen, ob zwischen Kopf und
  nächstem Kopf (oder Dateiende) eine Zeile steht, die nicht leer ist.
- [x] Doku: „a section it leaves empty goes too“ → „…, unless a comment is
  left in it“.
- [x] Commit `fix(config): keep a section header that still holds comments`.

### Task 7: `tui.fit` zählt Zellen

**Files:** `internal/tui/list.go`, `list_test.go`.

- [x] Test: `fit("日本語テキスト", 6)` → `"日本語"`; `fit("ab日", 3)` → `"ab"`;
  ein kombinierendes Zeichen zählt 0.
- [x] Umsetzung: `cells(r rune) int` über `width.LookupRune(r).Kind()`
  (`EastAsianWide`, `EastAsianFullwidth` → 2), `unicode.Mn`/`Me`/`Cf` → 0.
- [x] Commit `fix(tui): fit a line by display cells`.

### Task 8: `config list` kürzt lange Werte

**Files:** `internal/cli/config.go`, `config_test.go`, Doku.

- [x] Test: ein Wert über 60 Zeichen erscheint in `list` gekürzt auf 60 mit
  `…` am Ende; `list --json` und `get` bleiben vollständig.
- [x] Umsetzung: `const listWidth = 60`, `shorten(s string) string`.
- [x] Commit `fix(config): shorten long values in the list`.

### Task 9: Pfadvergleich nur unter Windows und macOS ohne Groß-/Kleinschreibung (beim Bau um macOS erweitert: APFS ignoriert Groß-/Kleinschreibung in der Vorgabe)

**Files:** `internal/brain/maintenance/mergehook.go`, Tests.

- [x] Test: `foldsCase("windows")` true, `foldsCase("linux")` false; eine
  Variante `samePathOn(goos, a, b)` vergleicht auf Linux `A` und `a` als
  verschieden; `pathKeyOn` ebenso.
- [x] Umsetzung: `samePath(a, b) = samePathOn(runtime.GOOS, a, b)`, gleich für
  `pathKey`.
- [x] Commit `fix(maintenance): compare paths by case outside Windows`.

### Task 10: Fremde Hookdatei in `merge-hook status`

**Files:** `internal/brain/maintenance/mergehook.go`, Tests, Akte.

- [x] Test: fremde Datei am Hookpfad → `Detail == pfad+": another hook"`;
  keine Datei → `Detail == pfad` wie bisher. Die aufgezeichneten 4a2-Fälle
  bleiben grün.
- [x] Commit `fix(maintenance): name a foreign hook file in the status`.

### Task 11: Testlücken

**Files:** jeweils die Testdatei.

- [x] Jede Vorgabe in `Keys()` dekodiert als `v = <Vorgabe>` (auch ohne Namen).
- [x] `tui.Pick`-Tests prüfen `ok` und `err`.
- [x] Fixture `hostfile/testdata/loomux-settings.json` gegen die echte
  `.claude/settings.json`: der Test liest die echte Datei aus dem Repo
  (`../../../.claude/settings.json`), Fixture entfällt, wenn sie gleichwertig
  ist; sonst Befund.
- [x] `remove-installed`: der Fall prüft den Inhalt (Hook entfernt, Eintrag
  weg), nicht nur die Bytezahl.
- [x] Vorlagentest prüft Befehl und Unterbefehl (`loomux brain catalog`,
  `loomux config set`) gegen die Unterbefehle, die der Befehl kennt.
- [x] `Apply` fragt `approve` nicht für eine Änderung, deren Binary fehlt.
- [x] Common-Dir über Junction: `RecordMerge` aus einem Junction-Pfad schreibt
  ein Ereignis.
- [x] Ein roter Test ist ein Fehler: eigener `fix`-Commit. Sonst ein Commit
  `test: …`.

### Task 12: Akten und Doku

- [x] `stufe-4a-1.md`, `stufe-4a-2.md`: die erledigten Punkte als „Behoben am
  2026-09-25“ mit den Entscheidungen oben; die Menschenschritte bleiben.
- [x] README/`migration.md` nur, wenn sich ein Stand ändert (er ändert sich
  nicht: 🚧 wegen der Menschenschritte).

### Abschluss

- [ ] `sh ci/gate.sh` grün; `release-pr`-Skill für Gruppierung, Label
  (`release:minor`), Rumpf; Push-Befehl dem Menschen nennen.
