# Wächterregel: kein Interpreter, der sein Programm von stdin liest

Stand 2026-10-06, Basis `origin/master` 757d600b, Zweig `feat/guard-stdin-interpreter`.

## Anlass

Implementierer-Subagenten starteten `python -` und `python3 -` mit leerem
Heredoc, obwohl ihr Brief es ausdrücklich verbot; einer hing, bis er per
`TaskStop` beendet wurde. Ein Satz im Brief wirkt nicht, die Regel gehört in
den Wächter, den loomux für seine Projekte ohnehin vor jeden Shell-Aufruf
stellt. Das im Auftrag genannte globale Skript
`~/.claude/scripts/no-stdin-interpreter.py` gibt es auf dieser Maschine nicht,
und `~/.claude/settings.json` ruft keinen solchen Hook; die Regel steht hier
deshalb vollständig, nicht als Port.

## Ziel und Erfolg

Der PreToolUse-Wächter verweigert jede Zeile, in der ein Interpreter sein
Programm von stdin lesen würde, ohne dass stdin aus einer Datei oder einer
Pipe mit echtem Inhalt kommt. Er tut es für beide Wirte (Claude Code,
Antigravity) und für jedes Werkzeug, das Shell-Zeilen trägt (`Bash`,
`PowerShell`, `run_command`, getippte Zeilen über `manage_task` und
`send_command_input`). Erfolg heißt: der Selbsttest unten hält, die
Differenzbatterie zeigt keine Zeile, die vorher verweigert war und jetzt
durchgeht, und jede neu verweigerte Zeile ist eine, die diese Regel meint.

## Entscheidungen

1. **Immer an, beide Modi, kein Schalter.** Die Regel ist eine eingebaute wie
   `git push` und die `gate`-Regel; eine Projektkonfiguration kann sie nicht
   abschalten (hooks.md, „No allow mode“). Sie hängt nicht an `[guard] mode`.
2. **Ort.** Neue Datei `internal/hooks/guardstdin.go` mit der Konstante
   `stdinReason` und `readsProgramFromStdin(line string) bool`. `checkTool`
   ruft sie je Zeile neben `armsOrDisarms` auf.
3. **Heredoc-Rümpfe nur für diese Regel entfernen.** Alle bestehenden Regeln
   lesen einen Heredoc-Rumpf als Befehle, und hooks.md nennt das als bewusste
   Grenze. Diese Regel entfernt die Rümpfe vor dem Lesen, auf einer Kopie der
   Zeile; die anderen Regeln sehen die Zeile unverändert, die Differenzbatterie
   hält das fest.
4. **Nur der anführungszeichenbewusste Schnitt.** Die Schreibregeln schneiden
   die Zeile zweimal, auch blind für Anführungszeichen, und nehmen dafür
   Fehlverweigerungen in Kauf. Diese Regel wehrt keinen Gegner ab, sie
   verhindert ein Hängen: ein Agent kann jederzeit eine Datei schreiben und
   ausführen. Ein übersehener Fall kostet einen gestoppten Task, eine
   Fehlverweigerung jede Commit-Nachricht, die den Befehl nennt
   (`git commit -m "fix; python -"`). Darum `splitSegments(variant, true)`
   allein.
5. **Zeilen in Strings.** Über `anyRunLine` liest die Regel auch die Zeilen,
   die eine Shell aus einem String laufen lässt (`sh -c "python -"`,
   `pwsh -Command …`, `cmd /c …`, `eval`), wie die übrigen Befehlsregeln.
6. **Tabellen je Interpreter statt einer flachen Flaggenliste.** Die Liste des
   Auftrags (`-c`, `-e`, `-m`, `-E`, `--eval`, `-p`, `--print` heißen
   „Programm auf der Zeile“) stimmt nicht für jeden Interpreter: Pythons `-E`
   ignoriert die Umgebung, Rubys `-E` nimmt eine Kodierung als Wert, und
   `perl -c` prüft die Syntax eines Programms, das es von stdin liest
   (gemessen: `echo 'print 1' | perl -c` meldet `- syntax OK`). `-Q` (nur
   Python 2) entfällt.
7. **`uv run -` liest selbst von stdin.** Gemessen mit uv 0.12.16:
   `echo 'print(1)' | uv run --no-project -` druckt `1`. Ein nacktes `uv run`
   bricht mit Exit 2 ab („Provide a command or script …“) und hängt nicht, es
   geht durch.
8. **Inline-Text über eine Pipe zählt nicht als stdin** (Entscheidung des
   Nutzers vom 2026-10-06). `cat <<EOF | python -` und PowerShell
   `@'…'@ | python -` werden verweigert wie ein Heredoc direkt ins Programm;
   `echo 'print(1)' | python -` und `python - < script.py` bleiben erlaubt.
9. **Begründungstext wirtsneutral**, englisch:

   > loomux refuses an interpreter that reads its program from stdin
   > (`python -`, a bare `python`, `uv run -`, a heredoc or here-string into
   > one): with nothing piped in it waits until it is stopped, and a shell may
   > rewrite the backslashes of inline program text. Write the script to a file
   > in the scratchpad with the host's file tool (Write, write_to_file) and run
   > that file.

## Ablauf je Zeile

1. `withoutHereBodies(line)` entfernt die Rümpfe (Abschnitt unten).
2. `anyRunLine` mit einem Richter, der für jede Variante aus `lineVariants`
   (Variablen und Aliase der Zeile eingesetzt, Fortsetzungen gefügt) den
   anführungszeichenbewussten Schnitt nimmt.
3. Je Segment: ob es gespeist ist (Abschnitt „Echtes stdin“), dann jede
   Lesart aus `readings`; verweigert wird, wenn eine Lesart verweigert.
4. Je Lesart: `readPrefixes(words).program` streift `VAR=wert`, `env`,
   `exec`, `nohup`, `time` und die übrigen bekannten Hüllen ab. Dann
   `uv run`/`uvx` (Abschnitt unten), dann die Interpreter-Prüfung.

Ein Richter für eine verschachtelte Zeile entfernt ihre Heredoc-Rümpfe
ebenfalls, denn `sh -c` kann einen mehrzeiligen String tragen.

## Interpreter

Erkannt am Basisnamen in Kleinschreibung, ohne `.exe`, auch hinter einem Pfad
(`C:/Python314/python.exe`, über die PowerShell-Lesart auch mit `\`):
`python`, `python3`, `python3.N` (`python` plus Ziffern, Punkte und das `t`
eines Free-Threading-Builds, `python3.14t`), `py`, `node`, `perl`, `ruby`.

Die Info-Spalte heißt genauer: ganze Wörter, nach denen der Interpreter kein
Programm von stdin liest — er druckt und endet (`--version`), oder er sucht
seine Dateien selbst (`node --test`).

| Interpreter | Programm auf der Zeile | nimmt einen Wert | Info, nur als ganzes Wort |
|---|---|---|---|
| python, py | `-c`, `-m` | `-W`, `-X`, `--check-hash-based-pycs` | `-V`, `-VV`, `--version`, `-h`, `-?`, `--help`, `--help-env`, `--help-xoptions`, `--help-all`, und für den `py`-Starter `-0`, `-0p`, `--list`, `--list-paths` |
| node | `-e`, `--eval`, `-p`, `--print` | `-r`, `--require`, `--import`, `--loader`, `--experimental-loader`, `-C`, `--conditions`, `--input-type` | `-v`, `--version`, `-h`, `--help`, `--v8-options`, `--test` |
| perl | `-e`, `-E` | `-I`, `-M`, `-m` | `-v`, `-V`, `-h`, `--version`, `--help` |
| ruby | `-e` | `-r`, `-I`, `-C`, `-E` | `-v`, `--version`, `-h`, `--help` |

Gemessen am 2026-10-06: Python 3.13.15 (`-X utf8 -c` läuft), Node 24.14.1
(`-r fs -e` läuft), Perl 5.42.2 (`-I /tmp` und `-M strict` nehmen den Wert
auch als eigenes Wort); `python -V`, `node -v`, `perl -v` enden ohne Skript
mit geschlossenem stdin, ebenso `node --test` in einem leeren Ordner,
`py --list`, `py -0p`, `perl --version` und `perl --help` (alle Exit 0,
gemessen im Abschlussreview und vom Controller nachgemessen).

Vor der Interpreter-Prüfung gilt: Beginnt die Lesart (nach `VAR=wert` und
den reservierten Wörtern einer Bedingung wie `if`, `then`, `!`) mit
`command` und einer Flagge, die `v` oder `V` enthält, beschreibt sie einen
Befehl und führt keinen aus → durch (`command -v python`, das übliche „ist
python installiert?“ in Skripten). `readPrefixes` streift `command -v` sonst
ab und ließe ein nacktes `python` zurück. Ruby ist nicht installiert; seine Zeile folgt der
Handbuchseite, `ruby -v` beendet sich laut ihr ohne Skript.

Lesen der Argumente, von links:

- Eine Umleitung samt Ziel (`> out.txt`, `2>&1`) ist kein Argument und wird
  übersprungen.
- `--` beendet die Flaggen; das Wort danach ist das Programm.
- Ein Info-Wort (ganzes Wort, `-V`, nicht `-V:3.12` des `py`-Starters) heißt:
  der Interpreter liest kein Programm → durch.
- Eine lange Flagge (`--x`): Programmflagge → durch; Wertflagge ohne `=` →
  das nächste Wort ist ihr Wert; sonst weiter.
- Eine kurze Flagge wird als Bündel gelesen, Buchstabe für Buchstabe: eine
  Programmflagge → durch (`-uc`, `-lne`, `-e'print 1'`); eine Wertflagge →
  der Rest des Worts ist ihr Wert, als letzter Buchstabe das nächste Wort
  (`-Wignore`, `-W ignore`); andere Buchstaben überspringen (`-3.12`, `-S`).
- Das erste Positional: `-` → das Programm kommt von stdin; jedes andere ist
  ein Skript → durch (`python script.py -` reicht `-` an das Skript weiter).
- Kein Positional → das Programm kommt von stdin (`python`, `python -X utf8`).

## `uv run` und `uvx`

`uv` und `uvx` stehen in `knownTools`, nicht unter den Hüllen von
`readPrefixes`; die Regel liest sie selbst. Nach `uv run` oder `uvx` werden
Flaggen übersprungen, und die Wertflaggen `--with`, `--with-editable`,
`--with-requirements`, `--python`, `-p`, `--project`, `--directory`,
`--package`, `--extra`, `--group`, `--env-file`, `--index`, `--from`
nehmen das nächste Wort. Danach:

- erstes Positional `-` → stdin (`uv run -`);
- erstes Positional ein Interpreter → dessen Tabelle (`uv run python -`);
- anderes Positional oder keines → durch (`uv run --script tool.py`,
  `uv run pytest`, nacktes `uv run`).

## Echtes stdin

Kommt das Programm von stdin, verweigert die Regel, außer stdin ist gespeist:

- ein Wort der Lesart ist eine Umleitung `<` oder `0<`, nicht `<<` und nicht
  `<<<`; oder
- das Segment folgt einem `|`, und die linke Seite der Pipe ist nicht leer
  und enthält kein `<<` (das deckt Heredoc, `<<<` und den Platzhalter `<<@`
  eines Here-Strings ab). Die linke Seite ist jedes Segment seit dem letzten
  `;`, `&`, Zeilenende oder `||`, das außerhalb von Klammern und
  geschweiften Klammern steht (`pipeLeft`): eine Subshell, eine Gruppe oder
  ein `$( )` vor dem Strich gehört dazu. So gehen `echo $(date) | python -`
  und `(echo x) | python -` durch, während `(cat <<'EOF'; echo x) |
  python -`, `cat <<'EOF' $(date) | python -`, `{ cat <<'EOF'; } |
  python -` und `cat <<'EOF' | grep p | python -` verweigert bleiben.
  Der zweite Strich von `||` speist nie, auch in einer Klammer oder Gruppe
  (`(a || python -)` wird verweigert). Geschweifte Klammern zählen nur, wenn
  die einzeln stehenden `{` und `}` der Zeile paarweise aufgehen
  (`bracesPair`); eine angeklebte (`{ $_.Name}`) oder zitierte (`' { '`)
  hielte sonst eine Gruppe bis zum Zeilenende offen. (Nachtrag aus Abschlussreview und
  Nachprüfung; die Fassung davor nahm nur das letzte nicht leere Segment
  und ließ die Heredoc-Fälle durch, der Nutzer entschied am 2026-10-06, die
  Lücke zu schließen.)

Enthält der Text des Segments selbst `<<`, speist nichts: `python - <<'EOF'`
und `python <<<'print(1)'` werden verweigert, auch hinter einer Pipe. Geprüft
wird der rohe Text, nicht die Wörter einer Lesart; ein `<<` in
Anführungszeichen (`echo "<<" | python -`) verweigert darum ebenfalls, ein
bewusst hingenommener Fehltreffer.

## Heredoc-Rümpfe

`withoutHereBodies(line)` liest die Zeile Zeile für Zeile:

- bash: auf einer Befehlszeile jedes `<<WORT` und `<<-WORT` außerhalb von
  Anführungszeichen, kein `<<<`; das Wort darf in `'…'`, `"…"` oder mit `\`
  davor stehen, Leerzeichen zwischen `<<` und Wort sind erlaubt. Mehrere
  Heredocs einer Zeile werden der Reihe nach abgeschlossen. Die Folgezeilen
  bis zur Endzeile (gleich dem Wort, bei `<<-` nach Entfernen führender Tabs)
  fallen samt Endzeile weg; ein nie geschlossener Rumpf läuft bis zum Ende.
  Die Kopfzeile mit `<<WORT` bleibt stehen.
- PowerShell: `@'` oder `@"` am Ende einer Zeile (Leerraum danach erlaubt)
  öffnet einen Here-String bis zu einer Zeile, die mit `'@` bzw. `"@` beginnt.
  Der Here-String wird durch das Wort `<<@` ersetzt, der Rest der Endzeile
  hängt sich daran: `@'⏎print(1)⏎'@ | python -` wird `<<@ | python -`. So
  gilt für beide Shells dieselbe Prüfung auf `<<`. Ein Wort mit
  Anführungszeichen taugt nicht als Platzhalter: die bash-Lesart macht aus
  `@''@` das Wort `@@`.

## Selbsttest

Verweigert: `python -`; `python3 -`; `python`;
`cd x && python3 - <<'EOF'⏎print(1)⏎EOF`; `python <<EOF⏎print(1)⏎EOF`;
`PYTHONIOENCODING=utf8 python -`; `C:/Python314/python.exe -`; `node`;
`uv run python -`; `python -X utf8 -`; `python - > out.txt 2>&1`;
dazu `uv run -`; `perl -c`; `python -E -`; `cat <<'EOF' | python -⏎…⏎EOF`;
`@'⏎print(1)⏎'@ | python -`; `sh -c "python -"`; `a || python -`;
`python <<<'print(1)'`; `C:\Python314\python.exe -`; `py -3.12 -`.

Erlaubt: `python -S "C:/Users/micro/.claude/scripts/x.py"`;
`python -c 'print(1)'`; `python -m pytest -q`; `python script.py arg`;
`echo 'print(1)' | python -`; `python - < script.py`;
`go test ./... && python3 tools/x.py`; `node -e 'console.log(1)'`;
`cat <<'EOF' > file.txt⏎python -⏎EOF`; `git log --format=%B`;
`echo python`; `uv run python script.py`; `uv run --script tool.py`;
dazu `python --version`; `python -V`; `node -v`; `perl -v`; `ruby -v`;
`uv run`; `perl -ne 'print' f.txt`; `git commit -m "fix; python -"`;
`Get-Content x.py | python -`; `python -W ignore script.py`.

Beide Listen gelten in `default` und `strict`, für `Bash`, `PowerShell` und
`run_command`.

## Benannte Grenzen

Nicht erkannt: `python -i script.py` (interaktiv nach dem Skript; ein
nacktes `python -i` wird verweigert); eine Zeile, die auf ` @"` oder ` @'`
endet, ohne PowerShell zu sein (`git commit -m "see @"`), weil die Regel
dort einen Here-String öffnet und die folgenden Zeilen als dessen Text
nimmt (geprobt); ein Interpreter in
einer Variable oder einem Alias, die anderswo als auf der Zeile gesetzt
werden; Interpreter außerhalb der Liste (`pypy`, `deno`, `bun`, `php`,
`Rscript`); eine Wertflagge von node oder uv, die die Tabelle nicht kennt,
lässt ihren Wert als Skript gelten; ein Interpreter in einem Heredoc, den
eine Shell ausführt (`cat <<'EOF' | sh` mit `python -` im Rumpf, geprobt):
der Rumpf ist für diese Regel Daten, und das Lesen dessen, was eine Pipe in
eine Shell füttert, kennt `cat <<EOF` nicht als Quelle; ein `python -` nach
einer Zeile mit arithmetischem Shift (`$((1<<2))`), den die Regel als
Heredoc liest, der bis zu einer Zeile `2` läuft; ein Skriptargument in
Anführungszeichen, das mit `<` beginnt (`python - "<x"`), das als
`<`-Datei zählt (beide im Task-Review geprobt); eine Gruppe mit Heredoc, die
in den Interpreter führt (`{ cat <<'EOF'; } | python -`), wenn dieselbe
Zeile auch eine zitierte, einzeln stehende geschweifte Klammer trägt
(`echo ' { '`): `bracesPair` schaltet das Zählen der Klammern dann für die
ganze Zeile ab, damit eine verwaiste Klammer keine spätere, echte Pipe
verweigert (in der Nachprüfung geprobt, bewusst hingenommen).

Zu viel verweigert: eine Pipe, die erst auf der nächsten Zeile weitergeht
(`echo x |⏎python -`); `|&`, dessen `&` keine Pipe ist, die die Regel kennt
(geprobt); `echo 'python -' | sh`, dessen innere Zeile die Regel
ohne die Pipe der äußeren liest; ein Heredoc, den ein Agent zeilenweise in
einen Antigravity-Task tippt, weil getippte Eingaben Zeile für Zeile beurteilt
werden.

## Tests und Nachweise

- Tabellentest über den Selbsttest, für `Bash` in beiden Modi, dazu je eine
  Stichprobe für `PowerShell` und `run_command`.
- Die versionierte Differenzbatterie (`battery()` und
  `testdata/guard-battery-before.tsv`) bekommt die Zeilen des Selbsttests
  und verwandte Schreibweisen (mit Hüllen, Pfaden, Umleitungen, Pipes,
  verschachtelten Shells). Die Aufnahme geschieht am Basisstand mit dem neuen
  Testcode, ohne die Regel; das ist zugleich der rote Lauf. `flips` bekommt
  die Zeilen dazu, die diese Regel verweigern soll, und der Kommentar des
  Tests sagt das. Keine vorher verweigerte Zeile darf durchgehen.
- Nachtrag aus der Planung: Der Rekorder nimmt nur die Zeilen auf, die der
  Aufnahme fehlen, und lässt die übrigen Urteile stehen. Ein vollständiges
  Neuaufnehmen setzte 282 `pass`-Urteile der Arm-/Gate-Änderung auf
  `refused` (gemessen) und entwertete deren Nachweis.
- Jede neue Testzeile läuft gegen den Stand vor ihrem Code rot (Befehl und
  vollständige `--- FAIL`-Ausgabe im Bericht).
- Eine Mutationsrunde je Zweig der neuen Funktionen per `go test -overlay`
  (Pfade in der Form `C:/…`, nie zusammen mit `-cover`), jede Teilbedingung
  einzeln gestrichen, jede Tabellenzeile einzeln geleert.
- 100 % Coverage je Funktion.

## Doku

- `docs/en/hooks.md` und `docs/de/hooks.md`: eine Zeile in der Tabelle der
  eingebauten Regeln, ein Absatz zu Interpreter-Tabellen, gespeistem stdin
  und Heredoc-Rümpfen, die Grenzen oben in den Abschnitten zu den Grenzen.
- `README.md` und `README.de.md`: die Fähigkeit in der Liste des Wächters.
- Kein Roadmap-Eintrag: die Arbeit ist mit diesem Pull Request fertig.

## Auslieferung

Ein Commit `feat(hooks): refuse an interpreter that reads its program from
stdin` mit Code, Tests, Batterie und Doku; Spec und Plan als eigener
`docs`-Commit davor. Label `release:minor`.
