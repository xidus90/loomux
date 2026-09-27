# ulflow M1, Welle 3 — Befunde und Entscheidungen

**Plan:** [2026-09-14-ulflow-welle-3.md](2026-09-14-ulflow-welle-3.md) ·
**Spec:** [2026-09-11-ulflow-laufzeit-design.md](../specs/2026-09-11-ulflow-laufzeit-design.md) ·
**Vorgänger:** [2026-09-11-ulflow-welle-2-befunde.md](2026-09-11-ulflow-welle-2-befunde.md)
**Stand:** am 2026-09-14 fertig ausgeführt, `feature/agent-harness` von `5b3ebb8`
bis `ca633e0`, nicht gepusht. Geschrieben aus Ledger, Task-Berichten, den fünf
Task-Reviews, dem Abschlussreview, dem Nach-Review der Fix-Runde und der
Git-Historie.

`ulflow` ist ein Binary mit `run`, `resume`, `replay`, `show`, `list` und
`--option`. Der Beispiel-Flow der Spec läuft gegen das Fake-Modell zu einem
byte-genauen Golden-Journal. `ulguard hook session-start` meldet einen wartenden
Go-Lauf mit `ulflow resume`. Die Install-Skripte bauen `ulflow` mit, und alle
drei Binaries sind im Worktree und in `~/go/bin` neu gebaut. Das
Grün-Kriterium der Spec für Welle 3 ist mit den installierten Binaries erfüllt
(siehe „Messungen“).

## Commits

| Commit | Task | Inhalt |
|---|---|---|
| `a36e847` | 0 | `runs.Claim`, `WriteMarker` mit `O_EXCL`, `NextID` zählt Marken, `NewCatalog` lehnt nil-Prädikat ab, `Resume` lehnt unter Replay ein offenes Tor ab |
| `1707bdc` | 1 | Session-Start-Hook liest die Marke und nennt `ulflow resume` |
| `883428f` | 2 | `flowload.Params` und der Beispiel-Flow als Testdaten |
| `2ef95ec` | 3 | `cmd/flow` |
| `ad81011` | 4 | Golden-Journal |
| `3c3c53e` | 5 | Install-Skripte und Benchmark-Eintrag |
| `e8f4236` | Fix | falsche Kommentar-Begründungen, `/flow` in `.gitignore`, Benchmark-Methode |
| `879b52f` | Fix | `bool=false`-Test, `forgetUnstarted` nur bei `ErrNotExist`, `run <id>: ` vor dem Fehler eines behaltenen Laufs |
| `ca633e0` | Fix | Kommentarbreite im Golden-Test |

## Entscheidungen während der Ausführung

Die neunzehn Entscheidungen des Plans stehen dort; hier steht, was dazukam.

| Entscheidung | Grund | Kosten, falls falsch |
|---|---|---|
| Alle Subagenten (Implementierer, Reviews, Nach-Review) auf Opus, entgegen der Modellwahl der Skill | globale CLAUDE.md und Memory „Immer Opus 5“ | nur Kosten |
| `cmd/guard` bleibt bei 99,1 % statt 100 % | Der Brief von Task 1 setzt 99,1 % als Boden; alle sieben ungedeckten Blöcke liegen in `post_edit.go`, `status.go`, `worktree.go` und sind älter als der Plan; `resumer` ist voll gedeckt (vom Abschlussreview bestätigt) | ein eigener Coverage-Task |
| Das E2E-Projekt liegt unter `.superpowers/sdd/2026-09-14-ulflow-welle-3/e2e` statt im Scratchpad | Der brain-guard-Hook verweigert Schreibzugriffe ins Scratchpad der Sitzung. Das Verzeichnis ist git-ignoriert, liegt aber im Repo des Worktrees; `ask` braucht keine Baseline | eine schiefe Baseline für Flows, die eine brauchen |
| Das Abschlussreview läuft über `5b3ebb8..3c3c53e`, nicht ab der Merge-Basis | Der Zweig trägt Welle 1 und 2, beide schon geprüft; jede Nutzung ihrer APIs liegt im Bereich | ein wellenübergreifender Fehler bliebe unentdeckt |
| Das Abschlussreview vor den Befunden (Step 9) | sein Ergebnis gehört in die Befunde | — |
| Behält ein Lauf nach einem Go-Fehler seine Dateien, beginnt die Fehlerzeile auf stderr mit `run <id>: `; stdout und Exit bleiben | billigster Weg, den Lauf zu nennen, in der Form von Entscheidung 7 | ein Meldungsformat |
| `forgetUnstarted` bekommt eine `stat`-Funktion hineingereicht | Ein Stat-Fehler außer „nicht gefunden“ ist portabel nicht auszulösen; einziger Produktionsaufrufer reicht `os.Stat` | eine Testnaht |
| Der F3-Test zeigt Rot mit `return true, …` statt mit der Mutation `text != "false"` | Die vorgeschlagene Mutation ist nach der Prüfung auf `true`/`false` gleichwertig und von keinem Test zu fangen | — |
| Die kleinen Befunde, die das Abschlussreview als „kann bleiben“ triagiert hat, bleiben | Einzeln begründet in „Kleinere Befunde, bewusst offen“ | kleine Folgearbeiten |

## Aus dem Abschlussreview

Urteil: mergefähig mit Fixes, keine kritischen Befunde. Die Verträge zwischen
den Tasks halten: Marke schreiben (`runs.Claim`) und lesen (Hook, `recorded`)
stimmen in jedem Feld überein; Entscheidung 12 hält in der echten Reihenfolge
`openFlow → modelFor → Baseline → Claim → Run`; zwei Go-Läufe können sich kein
Journal teilen (`O_EXCL` ist unter Windows `CREATE_NEW`). Der Benchmark-Eintrag
ist gegen `go list -deps` nachgerechnet.

Zwei wichtige Befunde, beide in `e8f4236` behoben:

1. **`runs.Claim` versprach zu viel gegenüber Python.** Der Kommentar sagte, ein
   Journal gehöre nie zwei Läufen. Python zählt nur `*.jsonl`
   (`cli.py:143-148`) und überschreibt die Marke (`cli.py:561`); ein
   `ultraloom run` im selben Projekt kann dieselbe Nummer nehmen, die Marke
   überschreiben (und damit `runtime="go"` löschen) und ins selbe Journal
   schreiben. Der Kommentar gilt jetzt für Go-Läufe und nennt die Lücke. Das
   Verhalten ist unverändert, siehe „Für M2“.
2. **Sechs falsche Begründungen an korrektem Code:** `errWaitsAtAGate` in
   `runner.go` und der Testkommentar in `retrace_test.go` (ein Replay meldet
   ohne die Ablehnung `node … is not in the journal`, nicht die Pause als
   Ergebnis); `params.go` (die Marke schreibt Optionen als JSON-Strings, die
   JSON-Liste ist die Kommandozeilen-Form); „a clock that repeats itself“ in
   `main.go` und `golden_test.go` (die Uhr geht 500 ms je Aufruf);
   „before a run writes anything“ in `main.go` und `session.go` (`resume`
   fragt die Fabrik, nachdem das Journal existiert); „every other fixture's
   exit omits the code“ in `blocks_test.go` (`stage2_kinds/f.toml` hat
   `code = 3`).

Dazu aus der Triage behoben: der fehlende `false`-Test in `params_test.go`,
`forgetUnstarted` bei fremden Stat-Fehlern, die fehlende Laufnummer nach einem
Go-Fehler, `/flow` in `.gitignore`, und im Benchmark-Eintrag der Satz, dass nur
warme Läufe gemessen sind und wann die Probe-Zeilen entstanden.

Das Nach-Review hat alle sieben Punkte als behoben bestätigt, jede neue
Begründung gegen den Code nachgerechnet, und keinen neuen kritischen oder
wichtigen Bruch gefunden.

## Messungen

**Hook-Kosten** (Eintrag in `docs/benchmarks.md` und `.de.md`, 2026-09-14
16:09): `ulguard hook session-start --host claude` gegen ein Projekt mit einem
wartenden Go-Lauf, 25 warme Läufe nach einem verworfenen, Binary aus `5b3ebb8`
gegen das aus `ad81011`.

| | Davor | Danach | Unterschied |
|---|---:|---:|---:|
| Median (Min–Max) | 56,7 ms (51,9–67,6) | 58,8 ms (55,0–80,4) | +2,1 ms, innerhalb der Streuung |
| Binary-Größe | 5.947.904 Byte | 5.962.240 Byte | +14.336 Byte |

Die Probe aus dem Plan (nur Import, kein Aufruf) hatte +3.584 Byte und +0,9 ms
ergeben; echte Nutzung linkt `internal/flow`, `flowcfg`, `model` und
`flow/tmpl` mit.

**Ende zu Ende mit den installierten Binaries** (Tor-Flow `ask`):
`ulflow run ask` → `run 0001: paused`, `Ship it?`, Exit 3; der Hook liefert
`run 0001 is waiting at confirm: Ship it?` und `answer it with: ulflow resume
0001 --answer "your answer"`; `ulflow resume 0001 --answer yes` → `run 0001:
done`, Exit 0; `show` zwei Zeilen (`paused`, `ok`); `list` → `ask project ok`.
`ulflow --version` sagt `0.1.0`.

**Coverage** nach `ca633e0`: `internal/runs`, `internal/flow`,
`internal/runner`, `internal/flowload` 100,0 %; `cmd/flow` 99,5 %, nur `main()`
ungedeckt; `cmd/guard` 99,1 % wie vorher.

## Für M2

- **Nummernkollision Go gegen Python.** Das Fenster reicht weiter, als der neue
  Kommentar in `claim.go:20-23` sagt: Python zählt bei `cli.py:319`, schreibt
  die Marke aber erst bei `cli.py:418`, nach dem Laden des Flows. Es genügt,
  dass Python vor Gos erstem Journaleintrag zählt und nach Gos `Claim` seine
  Marke schreibt. In M1 ist das Fenster winzig (die Produktion hat keinen
  Adapter, es laufen nur Tor- und Exit-Flows); in M2 dauert es den ganzen ersten
  Agentenknoten. Ein Weg: das Journal gleich nach `Claim` leer anlegen. Dann
  darf `forgetUnstarted` nicht mehr „Journal existiert“ als „Lauf gestartet“
  lesen, sondern „Journal nicht leer“. Den Kommentar in `claim.go` dabei
  mitziehen.
- **`internal/runs/marker.go:78-79`** sagt „a marker is claimed, never
  overwritten“ und dass sonst zwei Läufe ein Journal teilten. Gegen Python gilt
  beides nicht; dieselbe Einschränkung wie in `claim.go` nachtragen.
- **`modelFor` kennt nur `kind = "agent"`** (`session.go:85`). Ein späterer
  Baustein, der ein Modell ruft, fiele durch die Ein-Anbieter-Prüfung und die
  Ablehnung vor `Claim`.
- **`resume` ohne `--answer` fragt zuerst die Modellfabrik** (`resume.go:40`),
  wie `cli.py:377-390`. In M1 unerreichbar; in M2, wenn ein Adapter beim Bauen
  scheitern kann (fehlende Zugangsdaten), sieht man dann nicht einmal das Tor
  wieder.

## Abweichungen von Entscheidung 8, benannt

Entscheidung 8 sagt „Ablehnungen nach `cli.py:311–415`“. Bewusst anders:

- `ulflow replay 0001 --answer yes` ist ein Flag-Fehler mit Exit 2; Python lehnt
  mit Exit 1 und dem Hinweis auf `resume` ab (`cli.py:311-316`).
- Die Baseline-Ablehnung bei `resume` (`resume.go:98-99`) lässt Pythons „before
  the guard measured against a commit“ weg — eine Go-Marke trägt den Commit
  immer. Die Ablehnung bei `run` (`run.go:55`) lässt „its repairs“ weg.
- Python lehnt „kein offenes Tor“ vor dem Laden des Flows ab, Go nach `openFlow`.

## Kleinere Befunde, bewusst offen

- `internal/runs/marker.go:80-85`: scheitert `WriteString` oder
  `Close` nach erfolgreichem `OpenFile`, bleibt eine halbe Marke, die `NextID`
  zählt und `ReadMarker` ablehnt. Braucht eine volle Platte; kostet eine Nummer.
- `cmd/flow/run.go:85-91`: „nur ein fehlendes Journal macht einen Lauf
  ungestartet“ stimmt nicht, wenn das erste `journal.Append` nach dem Baustein
  scheitert — dann fehlt das Journal, obwohl der Knoten lief. Älter als diese
  Welle, im Test nicht auszulösen (`internal/journal/write.go:34-35`).
- `forgetUnstarted` ignoriert einen Fehler von `os.Remove`.
- `catalog.go`: dass die Sortierung bei zwei nil-Prädikaten denselben Namen
  nennt, prüft kein Test (nur ein nil im Test).
- `Resume(ctx, nil)` liest jetzt `Pending` auch ohne Replay; kein Test zeigt,
  dass es an einem Lesefehler scheitert. Aus der CLI unerreichbar, `recorded`
  liest vorher.
- `cmd/guard/hook_session_start.go:141` „neither runtime resumes the other's
  runs“: in der Wirkung wahr (Python findet keine `.py`-Datei, Go lehnt die
  Marke ab), aber nur Go prüft es; die beiden Prüfungen zu nennen wäre klarer.
- `flowload.Params`: `["a",null]` wird still zu `["a",""]`; Listen-Vorgaben
  teilen sich die Scheibe mit `graph.Params` (niemand ändert sie); ein leerer
  Optionsname aus einer von Hand geänderten Marke ergibt `option  is no
  parameter` mit doppeltem Leerzeichen (die CLI lehnt ihn ab).
- `session.go:135`: `TrimRight(text, "\n")` schneidet alle Zeilenumbrüche ab,
  auch am `Detail`; eine CRLF-Fragedatei behält ein `\r`.
- `resume_test.go:52` prüft „nothing is written“ nur über die Zeilenzahl.
- „the same sequence every run“ (`main.go:45`, `golden_test.go:16`) meint jede
  Testausführung; innerhalb eines Harness läuft die Uhr über `run` und `resume`
  weiter.
- Der Golden-Test druckt bei Abweichung beide Dateien ganz; `-update` gibt es
  nur in `cmd/flow`: neu erzeugen mit `go test ./cmd/flow -update`.

## Betrieb

- **`ulguard post-edit` bricht bei jeder Bearbeitung** von `.gitignore`,
  `.gitattributes` und `scripts/install.sh` mit `**/*.sh: openBinaryFile:
  invalid argument` ab. Das Muster wird offenbar als Dateiname geöffnet. Nicht
  behoben, gehört in einen eigenen Fix an `cmd/guard`.
- Derselbe Hook meldet für `install.sh` SC1017 auf jeder Zeile: Die
  Arbeitskopie ist wegen `core.autocrlf=true` CRLF, der Index LF. Kein Befund
  am Skript; `.gitattributes` könnte `*.sh text eol=lf` bekommen.
- **Der brain-guard-Hook sperrt das Scratchpad der Sitzung** für Write. Commit-
  Nachrichten und das E2E-Projekt lagen deshalb im git-ignorierten
  SDD-Verzeichnis.
- `docs/benchmarks.de.md` hat die Überschrift `## Chronologisches
  Benchmark-Protokoll` zweimal (Zeile 48 und 152, schon vorher).
- Der Commit-Hook warnt bei jedem Lauf `VIRTUAL_ENV … does not match the
  project environment path .venv`; sechs Pakete außerhalb dieser Welle liegen
  unter 100 % (`interview`, `junction`, `render`, `sessions`, `settings`,
  `verify`).
- Ein Implementierer hat dreimal lesende Befehle verkettet, gegen „ein
  Shell-Befehl je Schritt“.
- `ulflow.exe`, `ulguard.exe` im Worktree und alle drei Binaries in `~/go/bin`
  sind auf dem Stand `ca633e0`. Das Vergleichsbinary `ulguard-before.exe` liegt
  im Scratchpad der Sitzung.
- Die SDD-Arbeitsdateien unter `.superpowers/sdd/2026-09-14-ulflow-welle-3/`
  kommen nach diesem Commit in den Papierkorb.
