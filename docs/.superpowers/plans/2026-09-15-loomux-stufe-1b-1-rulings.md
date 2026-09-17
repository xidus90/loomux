# Rulings zur Reparatur der Plan-Entwürfe 1b-1

Die Befundnummern F1–F83 stammen aus der Prüfung der Task-Entwürfe am 2026-09-15 (zwei Prüfer auf Stimmigkeit,
drei auf Spec- und Referenztreue; Arbeitsdatei, nicht eingecheckt). Jede Regel unten ist ohne sie vollständig.
Rangfolge: Spec vor Task, Task vor diesen Regeln, diese Regeln vor dem Vertrag
(`2026-09-15-loomux-stufe-1b-1-vertrag.md`). Wo ein Befund dem Vertrag widerspricht und der Entwurf der Quelle
folgt, gilt der Entwurf (Quelle schlägt Vertrag).

## Für alle Tasks

- **R1 (F1–F6): Stagen vor dem Tor.** Jeder Commit-Abschnitt lautet: `git add <genau die Dateien des Tasks>` →
  `sh .githooks/pre-commit` (Erwartet: Exit 0) → `git branch --show-current`, `git rev-parse --short HEAD`,
  `git diff --cached --stat` (Erwartet: genau die Dateien) → Nachrichtendatei schreiben → `git commit -F <datei>` →
  `git log -1 --format='%an <%ae>'` (Erwartet: die Nutzeridentität) und `git log -1 --format=%B`
  (Erwartet: keine `Co-Authored-By`-Zeile). Der Commit löst das Tor noch einmal aus; das ist gewollt.
- **R31 (leichte Platzhalter): feste Form statt Ellipse.**
  - Elidierte erwartete Ausgaben (`…`, `usw.`, `u. a.`, `z. B.`, Zeilennummern `:…`) werden entweder vollständig
    angegeben oder als ausführbares Muster (`grep -E '<muster>'`, Erwartet: Treffer) mit dem Satz „Zeilennummern
    variieren mit dem Kommentar".
  - Bedingte Coverage-Schritte heißen fest: „`go tool cover -func=…` — Erwartet: jede Funktion des Pakets 100.0%.
    Zeigt eine Zeile weniger, gilt Umzugsverfahren Schritt 6 (Test für genau den roten Block, rot sehen, grün);
    `//coverage:exempt` nur mit dem dort verlangten Grund."
  - Jeder Befehl hat eine Erwartung.
  - Jeder Task beginnt mit dem Satz: „Alle Befehle laufen in `$LOOMUX` (Task 0). Voraussetzung: Tasks 1 bis N−1
    sind committet."
- **R27 (F10):** `internal/dev/*` darf `internal/brain/pytext` benutzen (Global Constraints werden ergänzt).

## Einzelne Tasks

- **R2 (F7, F52, F81): READMEs nur in Task 16.**
  - Task 10 streicht seinen README-Schritt und `README.md`/`README.de.md` aus Files.
  - Task 16 pflegt beide READMEs vollständig: die fünf `loomux brain`-Befehle als aktive Befehle, die Statuszeile
    und die Reparatur des kaputten Codeblocks in `README.de.md`. Alles verankert am Text, nicht an Zeilennummern;
    das „vorher" wird am heutigen Text belegt.
  - Task 16 ergänzt außerdem `docs/en/cli-reference.md` und `docs/de/cli-reference.md` um `loomux brain`, im Stil
    des Abschnitts, den Task 13 für `loomux dev mutants` schreibt.
  - Der Satz „kein anderer Task fasst sie an" entfällt. Task 13 behält seinen cli-reference-Abschnitt.
- **R3 (F8, F76, F9): Task 14 übernimmt die Wortlaute von Task 13.** Das betrifft den Text von `ErrBaselineRed`,
  `ErrNoSources` mit Exit 2, die beiden Summenzeilen `N do not compile and are no mutants` und
  `N change nothing and are no mutants`, und dass es kein Urteil `not compiled` gibt.
- **R4 (F67): Task 8 entfernt das `N: `-Präfix, test-first in `translateReply`.** Umsetzung wie im Fix von F67:
  - `line` roh vor der Klammerung lesen.
  - Den Snippet mit `strings.Split(…, "\n")` teilen.
  - Das Präfix `strconv.Itoa(rawLine+i)+": "` nur entfernen, wenn **jede** Zeile ihr Präfix trägt; sonst bleibt der
    Snippet unverändert. Das gilt für jedes Profil.
  - Die vier Testfälle aus F67 kommen dazu.
  - Der Bericht nennt es als Befund zur Umzugsregel. Eine Paritätszeile entfällt.
- **R5 (F68, F69, F73): Task 11 baut den Fake echter nach.**
  - `MCPHandler` nummeriert Snippets wie qmds `addLineNumbers`. Tests nach F68: `"3: one\n4: two"`, ein leerer
    Snippet wird zu `"1: "`, und `TestLoomuxsPortsReadTheFake` erwartet weiter `"one\ntwo"`.
  - Punkt 3 des Berichts wird nach F69 berichtigt: Es unterscheidet sich nur das Präfix, nicht das Ausschnittfenster.
  - Neues Fixture-Feld `SearchError string \`json:"search_error"\``: `search|vsearch|query` enden mit Exit 1 und
    diesem stderr, das MCP-`query` antwortet mit einem JSON-RPC-Fehler. Unit-Tests für beide Wege.
  - Die MCP-Handler-Regel des Vertrags ist damit überholt.
- **R6 (F72): kaputte Registry und Manifest mit falschem Typ werden Fälle.**
  - Task 11 `registeredDirs`: Jeder Lesefehler der Registry, auch eine fehlende Datei, ergibt `return nil, nil`,
    mit dem Kommentar, dass die Wiedergabe ihn meldet. Der Test heißt
    `TestTranslateWorldLeavesAnUnreadableRegistryToTheReplay`.
  - Task 12 bekommt die Welten `broken-registry` (`registry.toml` = `[[area`) und `manifest-wrong-type`
    (`.brain.toml` mit `[area]` und `scope = 3`), mit den Fällen `brain-catalog/broken-registry` und
    `brain-catalog/manifest-wrong-type` (Exit 1).
  - Beide Paritätszeilen „kein Fall" entfallen.
- **R7 (F73):** Task 12 bekommt die Welt `engine-fails` mit gesetztem `search_error` und den Fall
  `brain-search/engine-fails` (`--profile full`, data, Exit 1). Die Paritätszeile „kein Fall" entfällt.
- **R8 (F74): Task 12 ergänzt alle in F74 genannten Fälle und die Welt `status-broken-register`.** Danach werden
  Fallzahl, Dateizahlen, Prüfsummen, die Zählungen in Step 5/9/15 und die README-Tabelle von `testdata/cases`
  nachgerechnet.
- **R9 (F75):** Task 13 `devMutants` nutzt `signal.NotifyContext(context.Background(), os.Interrupt)` wie im Fix
  von F75. Ein Seam-Test prüft: Keine `loomux-mutant-*`-Verzeichnisse bleiben übrig, Exit 1.
- **R10 (F70): `--state-dir` bleibt draußen.** Task 10 führt den Widerspruch Spec gegen Vertrag als Befund: Die
  Spec nennt `--state-dir`, loomux liest aus zwei Verzeichnissen. Der Mensch entscheidet bei der Freigabe der
  Paritätsliste; bis dahin bleibt die Paritätszeile.
- **R11 (F65): Eine `.loomux/config.toml` ohne `[area]`-Tabelle deklariert für `brain/*` nichts** (wie Guard-Regel
  R7a).
  - `ReadAreaManifestUntilStage4` geht dann zum nächsten Altnamen; gibt es keinen, folgt `ErrNoManifest`.
  - Mit `[area]` und leerem Scope kommt der Scope-Fehler.
  - Beide Fälle bekommen einen Test und eine Berichtszeile.
- **R12 (F64):** Die Helfer in Task 1 tragen den Ablauf im Namen: `legacyBrainDirEnvUntilStage3`,
  `legacyBrainDirUntilStage3For`, `manifestNamesUntilStage4`.
  - Der Kommentar an `requireScope` sagt, dass es mit Stufe 4 entfällt.
  - Ein Satz im Bericht sagt, dass Stufe 3 auch `search.ReadLastRun(legacyDir)` und jedes `legacyDir`-Argument in
    `brain/*` entfernt.
- **R13 (F66):** Der Kommentar an `ParseAwareIsoFormat` wird nach F66 berichtigt.
- **R21 (F14):** `TestTheTablesAreTheOnesMeasured` bricht mit klarer Meldung ab, wenn `unicode.Version` nicht
  `17.0.0` ist („pytext tables are measured against Unicode 17.0.0 (Go 1.27); re-measure for …"). Der Plankopf
  nennt Go 1.27 als Torvoraussetzung.
- **R14 (F71):** Die Vorhersage in Task 15 lautet: Mit Task 8s Präfixentfernung ist die stdout von
  `search --profile full` auf beiden Wegen gleich. Dazu kommen ein eigener `diff` der `^brain://`-Zeilen und ein
  voller `diff`; jede Abweichung ist ein Befund.
- **R15 (F77, F78):**
  - Task 14 Step 3 vergleicht `N mutants over <pkg>` mit 712/706/231/200. Die Zahl für config wird nach Task 1 mit
    `_mutants_of` aus `go_mutants.py` gemessen; das Kommando steht im Plan.
  - Außerdem wird die Wandzeit einer Baseline von `internal/hooks` während der Runde gemessen; sie muss unter 60 s
    bleiben, sonst folgt eine Runde mit `--workers 4` zum Vergleich.
- **R16 (F79):** Jeder `uv run` in Task 15 lautet `uv run --no-sync --project …` mit `PYTHONDONTWRITEBYTECODE=1`.
  `qmd mcp stop` hat denselben Fallback wie in Step 7.
- **R17 (F80):** Der Rauchtest in Task 16 läuft nur auf ausdrückliche Weisung des Nutzers (Schritt: fragen); der
  Pilot-Text sagt „auf Weisung des Nutzers vom Controller".
- **R18 (F54, F81):** Task 16 zählt die Bereiche wie `grep -c` in Step 1. Die exempt-Erwartung nennt
  `mutants.GoTest` nicht.
- **R19 (F82):** `t.Chdir(dir)` entfällt aus `cases_1b1_test.go`.
- **R20 (F83):** Der Bericht von Task 12 legt dem Menschen eine Pfadregel für `testdata/cases/1b-1-source/**` in
  `.loomux/config.toml` vor, als Mensch-Schritt mit TOML-Block.
- **R23 (F20):** Task 16 unterscheidet Usage-Fehler des Unterbefehls (`loomux brain <sub>: error: …`) von Fehlern
  des obersten Parsers (`loomux brain: error: …`).
- **R24 (F22, F31, F37, F38, F42, F43, F49, F53): Die Interfaces werden vollständig.**
  - Paketqualifier, alle genutzten Felder und Optionen (`WithCLI`, `WithConnect`, `ConnectFunc`, `SearchHit` mit
    allen Feldern, `HTTPSession.URL`, `FakePort`-Felder).
  - Zuordnungen korrigiert: Die Faltung liefert Task 11, L1 liefert Task 9.
  - Nicht gerufene Namen werden gestrichen.
- **R25 (F33, F32):** Task 8 bekommt nach Step 21 einen eigenen Abschnitt „Stagen, Tor, Commit" für
  `stamp.go`/`stamp_test.go` und korrigiert den Schrittverweis auf 22–25.
- **R28 (F39):** Task 11 prüft test-first: `Record` meldet für ein `Cmd` ohne Tokens `empty command` statt zu
  paniken.
- **R29 (F40):** Task 12 Step 1 wiederholt die Python-Kette gegen den gebauten Fake als Vorbedingung, mit exakter
  Erwartung.
- **R32–R35:**
  - Task 15 fügt ein „hinter dem letzten Eintrag".
  - In Task 15 Step 7 steht eine Schwelle statt „Größenordnung". Der `sync.OnceValue`-Umbau wird ein eigener
    test-first-Schritt mit Code.
  - Backticks in Tabellen werden doppelt gesetzt.
  - Task 0: Hash-Vergleich statt Literal; `test -x bin/loomux.exe` vor der Probe.
- **R36 (F62):** Das erwartete Rot nach dem Umzug in Task 3 bleibt; das Umzugsverfahren im Plankopf lässt es für
  Task 3 zu.
- **R37 (Abnahme `barrier-worktrees`, gemessen 2026-09-17): Task 0 Step 4 entfällt.** Ein verknüpfter Worktree
  eines `workspace`-Bereichs ist seit `2026-09-15-loomux-schranke-worktrees` ohne eigenen Registry-Eintrag
  beschreibbar (`internal/brain/guard/worktree.go`). Die Messzeilen von Task 0 „Schranke heute … Exit 2" und
  „dieselbe Probe gegen eine Kopie der Registry mit dem Block unten … Exit 0" beschreiben die Schranke davor und
  sind widerlegt: Dieselbe Probe gegen eine Registry **ohne** den Block antwortete am 2026-09-17 mit Exit 0 und
  ohne Ausgabe, ein Write nach `#GIT/nicht-registriert/x.go` daneben weiterhin mit `deny`. Step 4 wird damit zur
  Abnahmeprobe ohne Eintrag, und der Mensch schreibt an der Registry nichts. Die Prüfung, dass
  `bin/loomux.exe` im Hauptcheckout da und nicht älter als `cmd/`/`internal/` ist, bleibt (R35): Ohne sie ist das
  Exit 0 der Probe kein Beleg.
