# Wächterlücken schließen — Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Der PreToolUse-Wächter verweigert loomux-Befehle und Schreibzugriffe auf geschützte
Pfade auch in Zeichenketten-Shells, bei Kopien in einen Ordner, hinter Variablen derselben
Zeile, in Patch-Dateien, bei 8.3-Namen und Endpunkten und (strict) in Code-Strings.

**Architecture:** Alle Änderungen liegen in `internal/hooks`. Eine gemeinsame Funktion
`ranLines` liefert die Zeilen, die eine Zeile in Zeichenketten ausführt; `commandReasons`
bündelt die Befehlsregeln einer Zeile. Die übrigen Fixes erweitern vorhandene Leser
(`destination`, `moved`, `verbWrites`, `gitWrites`, `lineVariants`, `strictReasons`,
`judge.rels`, `namedPaths`). Eine Differenzbatterie hält jede Lockerung fest.

**Tech Stack:** Go, `go test`, `go test -overlay` für Mutanten.

**Spec:** `docs/.superpowers/specs/2026-10-01-loomux-guard-wrapper-holes-design.md`

## Global Constraints

- `dropPrefixes` und `wrapperTakesValue` in `guard.go` bleiben unverändert (fix/guard-winpty-stdbuf).
- 100 % Coverage je Funktion; ein Ausschluss nur mit `//coverage:exempt <Grund>`.
- Kein `init()`, keine Paketvariable, die beim Laden etwas parst.
- Code, Kommentare, Meldungen englisch; Commits nach Conventional Commits, ohne Arbeitspapier im Text, ohne Modell als Mitautor.
- Zeilen, die bewachte Befehle nennen, nur per Write/Edit in Dateien, nie per Heredoc oder `echo`.
- Overlay-Pfade in der Form `C:/…`.
- Jede neue Testzeile läuft vor dem Code rot; der rote Lauf wird mit Assertion gesehen.

## Review Focus

1. Eine Lockerung durch eine neue Variante: Jede Variante darf nur Gründe hinzufügen. Die Differenzbatterie prüft, dass keine verweigerte Zeile durchgeht.
2. Fehlverweigerungen im Alltag: `cp -r src dst`, `cp a b` (b ist eine Datei, strict), `git apply` eines Patches ohne geschützte Ziele, `rsync -t a b`. Sie müssen durchgehen; die Batterie hält sie als `pass`.
3. strict mit nicht vorhandenen Pfaden: `ziel/basename(quelle)` unter einer Datei darf im strict-Modus nicht „cannot resolve“ auslösen. Daher wird es nur bei einem Ordnerziel gebildet.
4. Ein Patch, der fehlt oder nicht lesbar ist: Das wird verweigert, mit Grund. `git apply --check` eines harmlosen Patches geht durch.
5. Tiefe: `pwsh -c pwsh -c pwsh -c pwsh -c loomux init` liegt jenseits von `maxInnerDepth`. Dort wird verweigert, nicht durchgelassen (eine ausgeführte Zeile in der Tiefe 3, die noch eine Shell ruft, zählt als nicht lesbar).

---

### Task 1: Differenzbatterie, aufgenommen am Stand vor dem Fix

**Files:**
- Create: `internal/hooks/guardholes_test.go`
- Create: `internal/hooks/testdata/guard-battery-holes.tsv` (durch den Aufnahmetest)

**Interfaces:**
- Produces: `holesBattery() []string`, `holesFlips() map[string]bool` (Zeile → nach dem Fix verweigert, Standardmodus), `holesFlipsStrict() map[string]bool`, `holesVerdict(root, line string, strict bool) bool`.

- [ ] **Step 1:** `guardholes_test.go` anlegen. Die Batterie besteht aus Familien:
  - `stringShellLines`: `sh -c`, `bash -c`, `pwsh -c`, `pwsh -Command` ohne Quotes, `powershell -NoProfile -Command "…"`, `iex`, `Invoke-Expression`, `eval`, `pwsh -enc <b64 von "loomux config apply">`, `bash -o pipefail -c '…'`, `cmd /c"loomux init"`, `env -S 'loomux init'`, je mit `loomux config apply`, `loomux init`, `loomux flow resume 1 --answer y`, `loomux dev switchover prune-hooks --file x --match y` und mit `echo x > .loomux/config.toml`; dazu die Gegenzeilen `sh -c "loomux config list"`, `sh -c "cd x && cat y"`, `bash -c 'frob "$1"' _ x`, `pwsh -c pwsh -c pwsh -c rm x`.
  - `folderCopyLines`: `cp /tmp/config.toml .loomux/`, `cp /tmp/config.toml .loomux` (Ordner vorhanden), `cp a/config.toml b/config.toml .loomux`, `cp -t .loomux /tmp/config.toml`, `mv /tmp/config.toml .loomux/`, `Copy-Item C:/tmp/config.toml .loomux/`, `Copy-Item -Path x/config.toml -Destination .loomux`, `cp -r /tmp/x/.loomux .`, `cp -r /tmp/tpl/. .`, `rsync -a /tmp/tpl/ .`, `xcopy C:\tmp\config.toml .loomux\`, `robocopy C:\tmp .loomux config.toml`; Gegenzeilen `cp a b`, `cp -r src dst`, `cp x.txt docs/`, `rsync -t a b`, `install -m 644 a b`, `ln -s a b`, `cp /tmp/notes.md .loomux/`.
  - `patchLines`: `patch .loomux/config.toml < p.diff`, `patch -o .loomux/config.toml a < p.diff`, `patch -p1 < p.diff`, `patch -p1 -i p.diff`, `git apply p.diff`, `git apply --directory=.loomux q.diff`, `git am p.patch`, `git apply missing.diff`; Gegenzeilen `git apply ok.diff`, `patch -p1 < ok.diff`, `git apply --check ok.diff`.
  - `assignmentLines`: `D=.loomux; echo x > $D/config.toml`, `export D=.loomux; rm $D/config.toml`, `$D='.loomux'; Set-Content "$D/config.toml" x`, `$env:D='.loomux'; Remove-Item $env:D/config.toml`, `Set-Variable D .loomux; rm $D/config.toml`, `set D=.loomux& echo x > %D%\config.toml`, `L=loomux; $L config apply`, `$L='loomux'; & $L config apply`, `alias l=loomux; l init`, `Set-Alias l loomux; l init`, `M=loomux; $M init`; Gegenzeilen `D=docs; echo x > $D/a.md`, `X=1; loomux config list`.
  - `spellingLines`: `echo x > LOOMUX~1/config.toml`, `echo x > .loomux/CONFIG~1.TOM`, `echo x > .loomux/config.toml.`, `echo x > ".loomux/config.toml "`, `echo x > .loomux./config.toml`, `rm -rf LOOMUX~1`; Gegenzeilen `echo x > PROGRA~1/x`, `echo x > docs/a.`.
  - `codeStringLines` (strict): `python -c "open('.loomux/config.toml','w')"`, `node -e "require('fs').writeFileSync('.loomux/config.toml','')"`, `python -c "print(1)"`.
  - `tailLines` (strict): `echo x > $D/config.toml`, `echo x > ${D}fig.toml`, `echo x > $D/state/hooks/x`, `echo x > $D/notes.md`.
  - der bestehende Grundstock: `manifestShellWrites()` und `manifestShellReads()` aus `guardshell_test.go`.

  Die Welt jedes Laufs: `t.TempDir()` mit `.loomux/config.toml`, `ok.diff` (ein Patch auf `src/a.go`), `p.diff`/`p.patch` (ein Patch auf `.loomux/config.toml`, git-Form mit `a/` und `b/`), `q.diff` (ein Patch auf `config.toml`, Form ohne Präfix, für `--directory`). Die Patches schreibt der Test mit `os.WriteFile`.

  `TestRecordTheHolesBattery` (nur mit `LOOMUX_RECORD_GUARD_BATTERY=1`) schreibt je Zeile `default<TAB>strict<TAB>strconv.Quote(line)` mit `refused`/`pass`. `TestTheHolesFixOpensNothing` liest die Datei, verlangt dieselbe Zeilenmenge und prüft je Modus: keine Zeile `refused → pass`; jede Zeile `pass → refused` steht in `holesFlips()` bzw. `holesFlipsStrict()`, und jede Zeile dort ist wirklich gekippt. Zu Beginn sind beide Karten leer.
- [ ] **Step 2:** Aufnehmen: `LOOMUX_RECORD_GUARD_BATTERY=1 go test ./internal/hooks -run TestRecordTheHolesBattery -count=1`, dann `go test ./internal/hooks -run TestTheHolesFixOpensNothing -count=1` → PASS. Die Datei lesen und gegen die Probetabelle der Spec halten.
- [ ] **Step 3:** Commit `test(guard): record the guard's verdicts on strings, folders, variables and patches`.

### Task 2: Zeilen in Zeichenketten für alle Befehlsregeln

> **Abweichung bei der Umsetzung:** Statt `commandReasons` in `checkTool` legt jede
> Befehlsregel ihren Körper in `anyRunLine` (siehe Spec A); `writesConfiguration` und
> `answersAGate` antworten so auch direkt gefragt für die Zeilen in Zeichenketten.
> Dazu kamen Shells, die von stdin lesen (`echo … | sh`, `bash <<< …`).

**Files:** Modify `internal/hooks/guard.go` (`checkTool`, neue `commandReasons`, `ranLines`), `internal/hooks/shellwrites.go` (`innerLine`, `stringShells`); Test `internal/hooks/guardholes_test.go`, `internal/hooks/shellwrites_test.go`, `internal/hooks/guard_test.go` (gepinnte Löcher).

**Interfaces:**
- Produces: `func ranLines(line string) (lines []string, tooDeep bool)`, `func commandReasons(line string, strict, nested bool) []string`, `func writesConfigurationNested(line string, anyProgram, nested bool) bool` (die alte `writesConfiguration` ruft sie mit `nested=false`).

- [ ] **Step 1:** Tests: `TestACommandInsideAStringIsJudgedLikeTheLine` über `stringShellLines` (jede loomux-Zeile verweigert, jede Gegenzeile frei); `TestInnerLineReadsEveryStringShell` mit Tabelle `args → line`: `iex 'a b'` → `a b`, `Invoke-Expression -Command "a b"` → `a b`, `pwsh -enc <b64("a b")>` → `a b`, `pwsh -e xyz!` → kein Treffer, `bash -o pipefail -c 'a'` → `a`, `bash +o x -c a` → `a`, `cmd /c"a b"` → `a b`, `env -S 'a b'` → `a b`, `env --split-string=a` → `a`. `TestATooDeepStringRefuses`: `pwsh -c pwsh -c pwsh -c pwsh -c loomux init` verweigert. Die gepinnten Löcher `sh -c "loomux init"`, `pwsh -c "loomux init"`, beide `prune-hooks`-Zeilen von `holes` nach `refused` in `guard_test.go` verschieben. In `holesFlips()` die Zeilen eintragen.
- [ ] **Step 2:** Rot laufen lassen: `go test ./internal/hooks -run 'TestACommandInsideAString|TestInnerLineReads|TestATooDeep|TestTheGuardRefusesCommands|TestTheHolesFix' -count=1`.
- [ ] **Step 3:** Code. `ranLines` läuft über `lineVariants(line)`, `segments`, `readings` und sammelt `innerLine(words)`; jede gefundene Zeile wird bis zur Tiefe `maxInnerDepth` weiter gelesen; findet die tiefste Ebene noch eine Zeile, ist `tooDeep` wahr. `commandReasons` enthält die bisherigen drei Prüfungen aus `checkTool` (Befehlsregeln per Regex bleiben auf der äußeren Zeile). In `checkTool`:

```go
reasons = append(reasons, commandReasons(line, policy.Strict, false)...)
inner, tooDeep := ranLines(line)
for _, ran := range inner {
	reasons = append(reasons, commandReasons(ran, policy.Strict, true)...)
}
if tooDeep {
	reasons = append(reasons, "loomux reads a command inside a string only "+strconv.Itoa(maxInnerDepth)+" shells deep; this line goes deeper, so it refuses")
}
```

  In `wordsWriteConfiguration` gilt `exempt` nur bei `!nested`. `innerLine` bekommt die Fälle `iex`/`invoke-expression` (wie `eval`, `-Command` übersprungen), `-encodedcommand`-Präfixe ab `-e` (`base64.StdEncoding`, UTF-16LE per `unicode/utf16`), `-o`/`+o` mit Wert bei den POSIX-Shells, `cmd` mit Wort `/c…`/`/k…`, `env` mit `-S`/`--split-string`. `stringShells` um `iex`, `invoke-expression`, `env` erweitern.
- [ ] **Step 4:** Grün, dann die volle Paketsuite `go test ./internal/hooks -count=1`.
- [ ] **Step 5:** Mutanten per Overlay: `exempt`-Bedingung `!nested` entfernen, Tiefenprüfung entfernen, je Zweig von `innerLine` den Treffer abschalten; jeder muss einen Test rot machen.
- [ ] **Step 6:** Commit `fix(guard): judge the loomux commands a shell runs from a string`.

### Task 3: Kopieren und Verschieben in einen Ordner

**Files:** Modify `internal/hooks/shellwrites.go` (`verbWrites`, `moved`, `destination`, `robocopy`/`xcopy`-Fall); Test `shellwrites_test.go`, `guardholes_test.go`.

**Interfaces:** `func destination(dir string, args []string, targetFlag bool) []shellTarget`, `func moved(dir string, args []string) []shellTarget`, `func intoFolder(dir string, sources []string, dest string, flagged, recursive, rsync bool) []shellTarget`. `gitWrites` reicht `dir` an `moved` weiter.

- [ ] **Step 1:** Tests in `TestShellWritesReadsTheTargetsOfEveryVerb`-Form (Welt mit Ordner `.loomux`): `cp /tmp/config.toml .loomux/` → `w:.loomux/`, `w:.loomux/config.toml`; `cp /tmp/config.toml .loomux` → dasselbe ohne Schrägstrich; `cp a b` (b fehlt) → nur `w:b`; `cp -r /tmp/x/.loomux .` → `w:.`, `rm:./.loomux`; `cp -r /tmp/tpl/. .` → `rm:.`; `rsync -a /tmp/tpl/ .` → `rm:.`; `robocopy C:\tmp .loomux config.toml` → `w:.loomux`, `w:.loomux/config.toml`. Gegenzeilen in der Batterie bleiben `pass`.
- [ ] **Step 2:** Rot laufen lassen.
- [ ] **Step 3:** `intoFolder`: Ein Ziel gilt als Ordner, wenn es auf `/` oder `\` endet, aus `-t`/`--target-directory` stammt, es mehrere Quellen gibt oder `os.Stat(joinPath(dir, dest))` einen Ordner meldet. Dann entsteht je Quelle `path.Join(dest, path.Base(src))`. Bei einer rekursiven Kopie ist es `removes: true`, und bei einer Quelle auf `/.`, `/*` (oder `/` bei rsync) zählt das Ziel selbst als `removes: true`. Rekursiv sind `-r`, `-R`, `-a`, `--recursive`, `--archive`, ein Bündel mit r, R oder a, `-Recurse` (ab `-rec`), jedes `rsync`, `xcopy` mit `/E` oder `/S`, `robocopy` mit `/E`, `/S` oder `/MIR`.
- [ ] **Step 4:** Grün, Paketsuite.
- [ ] **Step 5:** Mutanten: jede der vier Ordnerbedingungen einzeln abschalten; `recursive` immer falsch; Inhaltsregel (`/.`) abschalten.
- [ ] **Step 6:** Commit `fix(guard): judge a copy or move into a folder by the name it lands under`.

### Task 4: `patch`, `git apply`, `git am` und Patch-Dateien

**Files:** Create `internal/hooks/patchfiles.go`, `internal/hooks/patchfiles_test.go`; Modify `shellwrites.go` (`segmentWrites` merkt sich die Eingabe-Umleitung, `verbWrites`, `gitWrites`, `otherWriteVerbs`), `guardjudge.go` (`shellTarget.refusal` melden), Test `guardholes_test.go`.

**Interfaces:** `func patchPaths(text string) []string` (die Rohpfade aus den Kopfzeilen), `func patchTargets(dir string, files []string, strip int, prefix string, inline string) []shellTarget`; neues Feld `refusal string` in `shellTarget`, das `judge.reasons` als Grund ausgibt, ohne den Pfad zu prüfen.

- [ ] **Step 1:** Tests für `patchPaths`: git-Form (`diff --git a/x b/y`, `--- a/x`, `+++ b/y`, `rename to z`, `copy to w`), Unified-Form mit Zeitstempel (`+++ y\t2026-…`), `/dev/null` ausgelassen, CRLF-Patch. Für `patchTargets`: strip 1 nimmt `a/` weg; bei `strip < 0` (kein `-p` bei `patch`) werden alle Stufen 0…Tiefe und der Basisname geliefert; `--directory=.loomux` setzt das Präfix; eine fehlende Datei liefert ein Ziel mit `refusal`. Für den Wächter die `patchLines` der Batterie.
- [ ] **Step 2:** Rot laufen lassen.
- [ ] **Step 3:** Code: `patch` in `otherWriteVerbs`; `verbWrites` liest bei `patch` das erste Positionsargument und `-o`/`--output` (`posixValues`), ohne Dateiargument die Patch-Datei aus `-i`/`--input`, dem zweiten Positionsargument oder der Eingabe-Umleitung. `gitWrites` kennt `apply` und `am`: Positionsargumente sind Patch-Dateien, Vorgabe `-p1`, `-pN` und `--directory=`; ohne Datei wird die Eingabe-Umleitung gelesen. Eine Heredoc-Zeile (`<<`) wird über `patchPaths(line)` gelesen. Die Eingabe-Umleitung reicht `segmentWrites` als zusätzliches Argument an `verbWrites`.
- [ ] **Step 4:** Grün, Paketsuite.
- [ ] **Step 5:** Mutanten: strip-Logik, `/dev/null`-Auslassung, Zeitstempelschnitt, `refusal`-Weitergabe.
- [ ] **Step 6:** Commit `fix(guard): judge the files a patch changes`.

### Task 5: Zuweisungen in derselben Zeile

**Files:** Create `internal/hooks/assignments.go`, `internal/hooks/assignments_test.go`; Modify `guard.go` (`lineVariants` hängt die Einsetz-Varianten an).

**Interfaces:** `func assignments(line string) map[string][]string` (Name → Werte, Schlüssel für PowerShell und cmd klein), `func substituted(line string) []string` (höchstens 16 Varianten).

- [ ] **Step 1:** Tests: je Form der Spec ein Fall `assignments` → Karte; `substituted` für `$D/x`, `${D}x`, `$env:D`, `%D%`, `$d` bei PowerShell-Zuweisung `$D=`, Alias als ganzes Wort (`l init` → `loomux init`, `ls` bleibt), zwei Werte → zwei Varianten, Grenze 16. Wächtertests: die `assignmentLines` der Batterie; `alias l=loomux; l init` und `M=loomux; $M init` aus `holes` in `guard_test.go` nach `refused`.
- [ ] **Step 2:** Rot laufen lassen.
- [ ] **Step 3:** Code (lexikalisch über `tolerantWords` je Segment: `NAME=wert` als erstes Wort oder hinter `export|declare|local|readonly|typeset|alias|set`; `$NAME`/`${NAME}`/`$env:NAME` vor `=` als eigenes Wort oder angeklebt; `Set-Variable|sv|New-Variable|nv` mit `-Name`/`-Value` oder den ersten beiden Positionsargumenten; `Set-Alias|sal|New-Alias|nal` ebenso). Werte ohne umschließende Anführungszeichen.
- [ ] **Step 4:** Grün, Paketsuite, Batterie.
- [ ] **Step 5:** Mutanten: Groß-/Kleinschreibung, Wortgrenze, Grenze 16.
- [ ] **Step 6:** Commit `fix(guard): read a variable or alias set on the same line`.

### Task 6: strict — eine Variable vor einem geschützten Ende

**Files:** Modify `guardjudge.go` (`strictReasons`, neue `afterExpansion`, `tailMayReach`); Test `guardstrict_test.go`.

- [ ] **Step 1:** Tests: `tailLines` der Batterie in `holesFlipsStrict()`; `afterExpansion` für `$D/x`, `${D}x`, `$(pwd)/x`, `` `pwd`/x ``, `%D%/x`, `$env:D/x`, `a/$D` → `""`.
- [ ] **Step 2:** Rot laufen lassen.
- [ ] **Step 3:** `tailMayReach(tail)`: Für jeden `protectedGlob` ohne führendes `**/` und jedes Byte-Präfix `p` von `literalPrefix(glob)` (ohne Slash im Glob: das leere Präfix) wird `matchGlob(glob, p+tail)` geprüft, mit Kleinschreibung bei `fold`. Ein leerer Rest prüft nichts.
- [ ] **Step 4:** Grün; Mutanten: leere Präfixe überspringen, `fold` ignorieren.
- [ ] **Step 5:** Commit `fix(guard): refuse in strict mode a variable in front of a protected tail`.

### Task 7: 8.3-Kurznamen und Endpunkte

**Files:** Create `internal/hooks/shortnames.go`, `shortnames_test.go`; Modify `guardjudge.go` (`rels`).

**Interfaces:** `func lexicalSpellings(rel string, elements []string) []string`, `func aliasMatches(alias, long string) bool`, `func (j judge) protectedElements() []string` (die Literal-Elemente aller geschützten Globs, klein).

- [ ] **Step 1:** Tests: `aliasMatches("LOOMUX~1", ".loomux")`, `("LOOMUX~12", ".loomux")`, `("LO~1", ".loomux")`, `("LO1A2B~1", ".loomux")` wahr; `("L~1", ".loomux")`, `("CONFIG~1", "config.toml")`, `("CONFIG~1.TXT", "config.toml")` falsch; `("CONFIG~1.TOM", "config.toml")` wahr. `lexicalSpellings(".loomux./config.toml. ")` → `.loomux/config.toml`. Wächterfälle: `spellingLines`, auch über das Write-Werkzeug (`file_path: LOOMUX~1/config.toml`).
- [ ] **Step 2:** Rot.
- [ ] **Step 3:** Code; `rels` hängt `lexicalSpellings` für jede Form an, in beiden Modi.
- [ ] **Step 4:** Grün; Mutanten an Stammlänge, Endungsregel, Hash-Form.
- [ ] **Step 5:** Commit `fix(guard): read 8.3 short names and trailing dots without the file system`.

### Task 8: strict — Pfade in Code-Strings

**Files:** Modify `guardjudge.go` (`namedPaths`); Test `guardstrict_test.go`.

- [ ] **Step 1:** Test: `codeStringLines` (die ersten beiden verweigert, `print(1)` frei); `namedPaths([]string{"open('.loomux/config.toml','w')"})` enthält `.loomux/config.toml`.
- [ ] **Step 2:** Rot. **Step 3:** `strings.FieldsFunc` an `'"(),;[]{}` und Leerraum. **Step 4:** Grün; Mutant: Trennzeichen entfernen.
- [ ] **Step 5:** Commit `fix(guard): look for protected paths inside the code an unknown program gets in strict mode`.

### Task 9: Doku, Grenzen, Abschluss

**Files:** `docs/en/hooks.md`, `docs/de/hooks.md` (Limits beider Modi), `docs/en/cli-reference.md`, `docs/de/cli-reference.md` (Known holes), Kommentar an `readings` in `guard.go`.

- [ ] **Step 1:** Doku: geschlossene Formen aus den Limits nehmen (`F=…; echo x > $F`, 8.3 und Endpunkte, `python -c` nur noch Standardmodus, `bash -o pipefail -c`, `cmd /c"…"`, `env -S'…'`, Kommentar „a program held in a variable, a command inside a string“). Neu als Grenzen: Patch aus einer Pipe, Patch, der sich nach dem Urteil ändert, Funktion in derselben Zeile, `find -exec loomux`, `xargs loomux` mit Argumenten von stdin, `[scriptblock]::Create`. Neue Fehlverweigerungen nennen: `sh -c 'loomux init --dry-run'`, ein echter Ordner `XXXXXX~N`.
- [ ] **Step 2:** `grep` nach den alten Sätzen in beiden Sprachen und im Code (`a program held in a variable`, `command inside a string`, `8.3 short names`, `trailing dots`).
- [ ] **Step 3:** `sh ci/gate.sh` (Ausgabe in eine Datei im Scratchpad).
- [ ] **Step 4:** Commit `docs(guard): name what the guard now reads inside strings, folders and patches`.
