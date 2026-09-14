# Preflight-Rulings je Task (bindend für Implementierer und Reviewer)

Quelle der Befunde: preflight-findings.md (Workflow wyogq0t2w, 10 Prüfer gegen die Tag-Worktrees).
Spec schlägt Plan; wo der Plan eine Quelle falsch zitiert, gilt der tatsächliche Code.

## Task 1
- R1a: Zusätzlich `.gitattributes` mit `* text=auto eol=lf` im ersten Commit. Grund: `core.autocrlf=true` systemweit (C:/Program Files/Git/etc/gitconfig); CRLF-Checkouts brächen `gofmt -l` im Tor und `#!/bin/sh` in `.githooks/*`. Beide Quellrepos pinnen LF ebenso.
- R1b: `CLAUDE.md`-Satz nicht "predates HEAD", sondern: "If session-start warns that the binary is older than a Go source under `cmd/` or `internal/` (or `go.mod`/`go.sum`), rebuild before trusting a refusal." Grund: Task 12 und Spec §Sitzungshooks vergleichen gegen die jüngste Quelle, ausdrücklich nicht HEAD.

## Task 2
- R2a: covergate identifiziert Befunde über file:line; `Func` ist nur Anzeige (cover -func druckt Methoden ohne Empfänger, Funktionsliterale auf Paketebene gar nicht). Keine Codeänderung.

## Task 3
- R3a: `TestCleanKeepsGitsOtherVariables` (ul gitenv_test.go:26-31) verliert `GIT_AUTHOR_NAME` aus der Eingabe (bleiben: `GIT_EDITOR`, `GIT_TERMINAL_PROMPT`); die `Location`-Kommentare (gitenv.go:13-15, 38-39) werden auf die breitere Liste umgeschrieben.
- R3b: `scan_test.go` behält `package gitenv_test` aus der Quelle (Umzugsregel wortgleich); "Paket gitenv" im Plan meint das Verzeichnis.
- R3c: Die API bleibt die von ultraloom: `Clean(parent []string) []string`, `Environ() []string`. ultra-brains `Clean()` ohne Argument entspricht `Environ()`.

## Task 4
- R4a: `check` folgt den Tests des Plans, nicht `SRC_UL/cmd/init/check.go`: Usage-Fehler (kein Unterbefehl, falsche Argumentzahl bei commit-msg, unbekannter Unterbefehl) → Exit 2; ungeformte Dateien je Zeile auf **stdout**, Exit 1. Übernommen aus der Quelle: `gofmt` ohne Pfade prüft `.` (mit Test). Der Satz "gilt die Quelle" gilt nur für diesen Punkt. Paritätszeilen für Task 14 (siehe unten).
- R4b: Zusätzliche Literale: `gitwork.go:39` Fehlertext "run ultraloom in a working tree of its own" → "run loomux …" (Test mitziehen, falls er den Wortlaut prüft); Kommentare, die die umbenannten Pfade beschreiben (`hostio.go:34,40,116`, `hostio_test.go:19`, `sessions.go:3`), auf `.loomux/config.toml`, `.loomux/state/hooks/`, `loomux init` umstellen. Kommentare, die Python-Herkunft zitieren, bleiben.

- R4c (nachgetragen nach Task 3): `.githooks/pre-commit` fährt `go test` mit `-count=1` und einem Kommentar, warum: Liefert go test einen Teil der Pakete aus dem Cache, fehlen deren Zählungen im `-coverpkg`-Profil, und covergate meldet unveränderte Funktionen bei 0 % (beobachtet in Task 3).

## Task 5
- R5a: `mirrorcfg.go:38` liest `filepath.Join(root, ".loomux", "config.toml")`; Tests `mirrorcfg_test.go:13,16,111` mitziehen. Der zu ersetzende Satz steht am Typkommentar `mirrorcfg.go:17` ("Every other table in that file belongs to the Python side") → "… belongs to internal/config".
- R5b: `detect` hat außer `Detect`/`Facts` auch `edges.go` (`Runner`, `HooksPath`, `NeighbourWiki`) und `signals.go` — alles zieht mit.
- R5c: Junction-Coverage: Das Tor kennt nur Ausnahmen je Funktion. Arme, die ohne Systemfehler erreichbar sind (NUL-Byte in `UTF16FromString`/`UTF16PtrFromString` bei direktem Aufruf von `setMountPoint`/`reparseTarget`), bekommen Tests. Funktionen mit einem unerreichbaren Arm (`Create`: `filepath.Abs` nach erfolgreichem `Stat`; `setMountPoint`: `CreateFile` auf frisch angelegtes Verzeichnis; `reparseTarget`: `GetFileAttributes`/`CreateFile` nach `Lstat`, `DeviceIoControl(FSCTL_GET_REPARSE_POINT)`) bekommen `//coverage:exempt` mit genau diesen Aufrufen als Grund.
  - Korrektur nach Task 5: `GetFileAttributes` in `reparseTarget` und `CreateFile` in `setMountPoint` sind mit einem fehlenden Pfad direkt erreichbar; unerreichbar bleiben `CreateFile`/`DeviceIoControl` in `reparseTarget` und das zweite `UTF16FromString` in `setMountPoint`. Gründe einengen und Tests ergänzen gehört in die finale Fix-Welle.

## Task 6
- R6a: `ManifestPath(root)` wird aus `manifestNames[0]` abgeleitet (eine Schreibweise im Paket); der Kommentar über `manifestNames` nennt die Kopie in `guard` (wie im Plan).
- R6b: `testlock/lock_windows.go`: Funktionen mit `t.Fatalf`-Armen, die nur ein scheiternder `icacls`/Encode erreicht, bekommen `//coverage:exempt` mit diesem Grund (doc.go: Testhilfe).
- R6c: `ReadManifest` ist nicht strikt (empirisch geprüft): `[policy]`, `[worktree]` stören nicht. Keine Änderung.

## Task 7
- R7a (kritisch): Eine `.loomux/config.toml` **ohne `[area]`-Tabelle** deklariert nichts: `declaredWikiRoot` geht an ihr vorbei, `checkInbox` behandelt sie wie ein fehlendes Manifest. Eine vorhandene `[area]` mit leerem `scope` oder ein kaputtes TOML bleibt ein Fehler (scheitert geschlossen). Grund: `.loomux/config.toml` ist laut Spec die eine Projektkonfiguration mit Sektionen je Modul; ein Projekt nur mit `[policy]` ist gültig, und ohne diese Regel verweigerte die Schranke jeden Write in einem solchen Projekt. Tests: Guard-Test (Write in Workspace mit policy-only-Config → erlaubt) und in Task 10 ein `PreToolUse`-Test dazu (→ 0). Paritätszeile.
- R7b: `legacyName` wird auch in `registry.go:213` (`manifestPath`-Rückfall) benutzt: `manifestPath` liefert den gebündelten Pfad unbedingt. Entfallen: `TestTheBundledManifestIsPreferredOverTheLegacyOne` (decide_test.go:1024), `TestABundledManifestIsPreferredWhereTheAreaCarriesBoth` (internal_test.go:517), `TestTheLegacyManifestIsRefusedEverywhere` (decide_test.go:473). Umgestellt statt entfernt, wo das Konzept für `.loomux/config.toml` gilt: Trailing-Dot-Test (:513), Schreibweisen-Tabelle (:497), Verzeichnis-statt-Datei (internal_test.go:529), internal_test.go:507/512.
- R7c: `.brain.toml` ist in ~75 Teststellen das Arbeitsmanifest (decide_test.go:746-1163, internal_test.go:366-879, mutation_test.go:45-325, run_test.go:280, decide_memory_test.go:78) — alle auf `.loomux/config.toml` umstellen, Datei für Datei; ein Test, der danach aus dem falschen Grund grün wäre, ist ein Fehler. Bericht listet entfernte und umgestellte Tests.
- R7d: Scratchpad: die eine `inMemory`-Closure (guard.go:427-429, genutzt bei 430 und 481) einmal erweitern, nicht zwei Stellen. Coverage: direkter Test `isScratchpad(x, "")`; Auflösungsfehler in `scratchpadBase` nach dem Muster memory_test.go:146-160 (Junction-Schleife, Windows).
- R7e: guards eigene Git-Säuberung `cleanEnv` (path.go:375-394) wird wie in Task 8 durch `gitenv.Environ()` ersetzt — eine Liste, Zweck von Task 3. Tests, die nur `cleanEnv` prüften, entfallen (gitenv testet die Liste); Bericht.

## Task 8
- R8a: Das Basispaket `pkg/check` besteht aus `finding.go` + `finding_test.go` (kein `check.go`); der `cp`-Glob des Plans stimmt.
- R8b: Es gibt keine `FindWikiPath`-Tests: `Root`-Tests inkl. aller Nachbar-Wiki-Arme neu schreiben (100 %).
- R8c: `lint` ohne Datei → Exit 2 (Usage-Konvention wie R4a); brain gab 1. Paritätszeile.
- R8d: `Root` nutzt `manifest.WikiLayout()` (validiert, lehnt Pfade außerhalb ab) statt des rohen `LayoutWiki`; Fehler ⇒ Manifest-Layout ignorieren, Rückfall.
- R8e: Kein Konflikt beim Ausgabekanal: brain und loomux schreiben Lint-Befunde nach stderr.
- R8f: `gate_test.go:17` (`decoyRepo`) auf `gitenv.Environ()`; `TestCleanEnvDropsTheConfigPrefixes` entfällt (gitenv testet die Präfixe).
- R8g: Opt-out `[area] wiki = false` entfällt ersatzlos. Paritätszeile.

## Task 9
- R9a: `TestCli` (guard_test.go:181-195) entfällt (runGuard ist direkt getestet; der Platzhalter antwortet 2). `TestCliDispatchesWorktree{Unlink,Link,Remove}` (worktree_test.go:993,1007,1248) werden Dispatch-Tests in `internal/cli/hook_test.go`. Aus `main_test.go` als cli-Tests nachbilden: erfolgreiches Aufwärtssuchen der Wurzel (session-start, `--host claude`, `t.Chdir` in ein Unterverzeichnis eines Projekts mit `.loomux/config.toml` → 0) und unbekanntes Flag. Paritätszeile: `hook` ohne/mit unbekanntem Event 1 → 2.
- R9b: Nach dem Umzug steht `PostToolUse` (war runPostEdit) auf 0 % — in die Coverage-Arbeit aufnehmen. Der Test für `defaultRunnerFor`: die go-Lane fährt `go vet ./...`, nicht `go version` → Temp-Wurzel mit `go.mod` und kompilierbarer `.go`-Datei, Erwartung 0.
- R9c: Der Prosa-Test "`hook post-tool-use --root <tmp>` … → 0" übergibt `--host claude` (hookCommand verlangt `--host` für jedes Event).
- R9d: `hook_session_start_test.go:297,307` rufen git ohne Env → `command.Env = gitenv.Environ()` (sonst schlägt der Scan-Test aus Task 3).
- R9e: Umgezogene Tests mit `.ultraloom/config.toml` bzw. `.ultraloom/hooks` (hook_session_start_test.go:166,181,267-277 u. a.) folgen Task 4: `.loomux/config.toml` und `sessions.StateDir` statt Literal.
- R9f: In `worktree.go` (:49,130,201,365) heißt die lokale Variable nach `mirrorcfg.` → `mirror.` nicht `mirror` (Paket-Schatten), sondern `mirrored`.
- R9g: Mit `waiting` entfallen dessen Tests, darunter `TestHookSessionStartOnAnUnwritableOutput` (einziger Test des `WriteContext`-Fehlerarms) → Ersatztest, der den Fehlerarm ohne Läufe erreicht (z. B. Host, dessen Adapter fehlt). Importe entsprechend bereinigen.

## Task 10
- R10a: Zusätzlicher Test: Write auf `ROOT/main.go` bei einer Config nur mit `[[policy…]]` → 0 (R7a).
- Bestätigt: `.env`-Grund "secrets are not written by an agent" (ul guard.go:50); deny-Hülle einzeilig mit `"permissionDecision": "deny"`; `guard.Run` gibt 2 auf Nicht-JSON und leere Eingabe; keine Namenskollision der neuen Testhelfer.

## Task 11
- R11a: Neuer Parameter von `getCommandsForStacks` heißt `projectRoot` (nicht `root`: schattet die Hilfsfunktion `root(text string) command`, post_edit.go:32).
- R11b: Claude sendet absolute `file_path`; der Wiki-Arm verbindet nur, wenn `!filepath.IsAbs(target)`. Test mit absolutem Pfad unter der Wurzel.
- R11c: `TestResolveWikiDir` ist eine Funktion mit vier Schritten — entfällt ganz, ersetzt durch `wikiDirFor`-Tests. Die Zusicherungen `post_edit_test.go:440-448` ("brain lint …", "uv run brain lint …") werden eine: "loomux lint wiki/concept.md".
- R11d: post-tool-use ist in 1a Claude-only (`HookPayload`); eine Antigravity-Nutzlast endet still mit 0. Paritätszeile (Adapter vollständig in Stufe 2).

- R11e: Die Wiki-Lane muss in einem loomux-Projekt ohne `okf_version` feuern: `PostToolUse` (und `Status`) nehmen den Stack `wiki` hinzu, wenn `.loomux/config.toml` ein gültiges `[layout] wiki` deklariert, das `wiki.Root` über das Manifest auflöst. Test: `[layout] wiki` ohne `okf_version` und ohne `[wiki]` → Lane läuft. Grund: `detect.declaresWiki` kennt nur `[wiki]`/`wiki = true`/`okf_version`; loomux' Schema deklariert das Wiki über `[layout] wiki`.

## Task 12
- R12a: `WalkDir` als Variable `walkDir` injizieren (der Plan erlaubt es); ohne sie sind der Info()- und der Lesefehler-Arm nicht erreichbar.

## Task 13
- R13a: `runner_test.go` ist `package cases_test` mit vier exec/powershell-Tests über `RunCase(c, "")` — diese ersetzen, nicht erweitern; neue Tests mit `cases.`-Qualifizierung und Importen `fmt`, `io`.
- R13b: `Record` schreibt `world_after` nur, wenn der normalisierte Baum von `world` abweicht (wie `tools/cases.py:155-159`).
- R13c: `TranslateWorld` übernimmt `decoded["policy"]` aus `.ultraloom/policy.toml` (die Datei trägt `policy` schon als oberste Tabelle) — nicht die ganze Map; Test prüft den Schlüsselpfad `policy.paths.rules`. Ein `match` als einzelner String bleibt String (Task 6 `globList` nimmt beides).

## Task 14
- R14a: Quellen der Guard-Welten heißen `SRC_UB/bench/cases/guard/allow-writable` und `…/deny-readonly`; die Weltnamen behalten das Präfix `guard-`.
- R14b: Die Welt muss die alte post-edit-Lane wirklich auslösen: `docs/wiki/index.md` bekommt `okf_version` im Frontmatter (sonst `[wiki]` in `.brain.toml`), beim Aufzeichnen steht `$TEMP/loomux-old` vorn im PATH (damit `brain` der Tag-Build ist). Der aufgezeichnete Exit von `broken-wiki-page` muss eine Lint-Ablehnung zeigen; Paritätszeile heißt "`brain lint` als Kindprozess" (kein uv, kein uv.lock).
- R14c: `lint/valid-page` nutzt `docs/wiki/sample.md` (Fixture aus `lint_test.go:14-22`, von `index.md` verlinkt); `index.md` ist eine Gerüstdatei, die Lint immer übergeht. Lint-Fälle bleiben `data` (stdout leer + Exit); das Befundformat belegt `report_test.go` — Paritätszeile.
- R14d: `.ultraloom/policy.toml` der Welt wörtlich: `[[policy.paths.rules]]` / `match = "generated/*"` / `reason = "generated files are rebuilt, not edited"`.

## Task 15
- R15: `dev bench-hooks` läuft mit `LOOMUX_STATE_DIR=$TEMP/loomux-bench-state`, darin eine `registry.toml` für das loomux-Repo; keine `.loomux/config.toml` (fehlend = leere Policy). Der Benchmark-Eintrag nennt, dass die echte Pilot-Config ein TOML-Parse mehr kostet, und dass `brain guard` im Vergleich mit 2 endet (loomux steht nicht in der alten Registry). Task 16s Mensch-Schritte werden dafür nicht vorgezogen.

## Task 16
- R16: `.loomux/config.toml` und `%LOCALAPPDATA%\loomux\registry.toml` legt der Controller dem Nutzer vor, schreibt sie nicht. `.claude/settings.json` schreibt ein Agent. Finale Gesamtreview nach Task 15 + Task 16 Step 3; Freigabespalte und Rauchtest gehören dem Nutzer.
- R16b: Der Task-16-Implementierer committet diese Datei als `docs/.superpowers/plans/2026-09-14-loomux-stufe-1a-rulings.md` (Arbeitspapier, deutsch), damit die Rulings den Workspace überleben.

## Paritätszeilen, die Task 14 zusätzlich eintragen muss
- `check`: Usage-Fehler Exit 1 → 2; `gofmt`-Liste stderr (mit Kopfzeile) → stdout; stderr-Meldungen mit Präfix `loomux check …` statt `ulinit check`/`usage: ulinit …` (R4a).
- `lint` ohne Datei: Exit 1 → 2 (R8c); `lint` mit mehr als einer Datei: brain lintete `args[0]` und ignorierte den Rest, loomux → Exit 2.
- `[area] wiki = false` wirkt nicht mehr (R8g).
- `hook` ohne Event / unbekanntes Event: Exit 1 → 2 (R9a).
- Config ohne `[area]` deklariert nichts statt Manifestfehler (R7a).
- post-tool-use nur Claude in 1a (R11d).
- Lint-Befundformat nicht durch Fälle belegt, sondern durch `report_test.go` (R14c).

## Nachgetragen waehrend der Ausfuehrung
- R15b (Task 15, im Dispatch erteilt, hier nachgetragen): Der Benchmark-Eintrag geht nach `docs/en/benchmarks.md` und `docs/de/benchmarks.md`, chronologisch angehaengt. Grund: Die Doku wurde von der anderen Sitzung (Commit 7702640) auf `docs/en`/`docs/de` umgestellt; `docs/benchmarks.md`, wie Plan und Spec ihn nennen, existiert nicht mehr. Beide Sprachen sagen dasselbe.
- R15c (Task 15, nach der Messung): Die im Plan erwartete Zerlegung der "~40 ms" entfaellt, weil die Messung heute nur 2,4 ms ueber dem Startboden zeigt (getaggtes brain.exe auf registriertem Bereich). Der Eintrag nennt den Befund samt Messbedingungen statt eine Luecke zu zerlegen, die es nicht gibt; kein Zielwert fuer Stufe 2, wie die Spec verlangt.
