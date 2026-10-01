# Wächterlücken: Zeilen in Zeichenketten, Ordnerziele, Variablen, Patches

Stand: 2026-10-01. Zweig `fix/guard-wrapper-holes` auf origin/master (e6c21eed).

## Ausgangslage

Ein Review vom 2026-10-01 (feat/lane-probation, 69e36864) meldete, dass eine Reihe von
Schreibweisen am PreToolUse-Wächter vorbeigeht, an master wie am Zweig. Die eigene Probe
gegen origin/master (Bash- und PowerShell-Werkzeug, Standard- und strict-Modus,
`.loomux/config.toml` vorhanden) bestätigt das nur zum Teil:

| Form | Standard | strict |
|---|---|---|
| Schreibzugriff in `bash -c`, `sh -c`, `pwsh -c`, `pwsh -Command`, `powershell -Command`, `eval` | verweigert | verweigert |
| `loomux config apply` in denselben Hüllen | **geht durch** | **geht durch** |
| `iex '…'`, `Invoke-Expression "…"`, Schreibzugriff oder Befehl | **geht durch** | Schreibzugriff verweigert, Befehl geht durch |
| `D=.loomux; echo x > $D/config.toml`, `$D='.loomux'; Set-Content "$D/config.toml" x` | **geht durch** | **geht durch** |
| `L=loomux; $L config apply`, `$L='loomux'; & $L config apply` | **geht durch** | verweigert |
| `cp /tmp/config.toml .loomux/`, `cp … .loomux`, `mv …`, `Copy-Item …` | **geht durch** | **geht durch** |
| `patch .loomux/config.toml < p.diff` | **geht durch** | verweigert |
| `echo x > LOOMUX~1/config.toml`, `echo x > .loomux/config.toml.` | **geht durch** | verweigert |
| `git apply p.diff`, `git am p.patch`, `patch -p1 < p.diff` | **geht durch** | **geht durch** |
| `python -c "open('.loomux/config.toml','w')"` | **geht durch** | **geht durch** |

Die Ursache bei den Hüllen: `shellWrites` liest den String hinter `-c`/`-Command`/`eval`
über `innerLine` als eigene Zeile, die Befehlsregeln `writesConfiguration` und
`answersAGate` (und auf feat/lane-probation `armsOrDisarms`) tun das nicht. `iex` und
`Invoke-Expression` kennt `innerLine` gar nicht. Bei `$D/config.toml` bekommt `mayReach`
einen leeren festen Teil und gibt auf. Kopierziele kennt `destination` nur als Ganzes,
nicht als Ordner, in dem die Quelle unter ihrem Namen landet.

Nachtrag nach dem Rebase: Während der Umsetzung landeten zehn Wächter-Commits auf
master (92d7baef, darunter fix/guard-winpty-stdbuf). Sie verweigern selbst schon
`pwsh -Command loomux …` ohne Anführungszeichen (behindUnknown), `cmd /c"loomux …"` im
strict-Modus und `xargs -I{} rm {}`. Die Differenzbatterie ist am neuen Stand neu
aufgenommen; die übrigen Zeilen der Tabelle gingen dort weiter durch.

## Entscheidungen

1. **Basis ist origin/master.** Die Rekursion entsteht in der gemeinsamen Schicht, damit
   feat/lane-probation nach dem Rebase `armsOrDisarms` mit einer Zeile anhängt.
   `dropPrefixes` und `wrapperTakesValue` bleiben unberührt: fix/guard-winpty-stdbuf ändert
   beide. Dessen benannte Lücke `script -c "…"` bleibt seine.
2. **Eine Differenzbatterie eigener Art**, nicht die von feat/lane-probation:
   `internal/hooks/guardholes_test.go` mit `testdata/guard-battery-holes.tsv`, nach
   demselben Muster (Aufnahme am Stand vor dem Fix, Vergleich danach). Die Dateinamen
   kollidieren nicht mit `guardgate_test.go`.

## Was geschlossen wird

### A. Zeilen, die eine Zeile ausführen, für alle Befehlsregeln

Eine Funktion `ranLines(line)` liefert jede Zeile, die die Zeile in einer Zeichenkette
ausführt: über `lineVariants` × `segments` × `readings` und `innerLine`, rekursiv bis
`maxInnerDepth`. Jede Befehlsregel legt ihren Körper in `anyRunLine(line, judge)`, das
`judge` für die Zeile selbst und, mit `nested`, für jede ausgeführte Zeile ruft:
`writesConfiguration` (Körper `lineWritesConfiguration`) und `answersAGate` (Körper
`lineAnswersAGate`). `checkTool` ruft die Regeln wie zuvor und fügt nur die
Tiefenverweigerung hinzu. In einer ausgeführten Zeile befreit keine Flagge
(`--dry-run`, `--propose`): Die Zeile ist nicht die, die der Wächter als Wörter liest,
und `sh -c 'loomux init --dry-run'` darf eine Fehlverweigerung kosten. Findet die
tiefste gelesene Ebene noch eine Zeichenketten-Shell, verweigert der Wächter mit
eigenem Grund, statt die Zeile ungelesen durchzulassen. feat/lane-probation legt
`armsOrDisarms` ebenso in `anyRunLine`, wie es `answersAGate` tut; ihr `knownGaps()`
führt `sh -c "loomux gate disarm --all"` als Durchlass, der danach kippt.
(Der Plan sah ein gemeinsames `commandReasons` vor; die Umsetzung hängt die
Rekursion an die Regeln selbst, damit Tests und Aufrufer, die `writesConfiguration`
direkt fragen, dieselbe Antwort bekommen wie `checkTool`.)

`innerLine` lernt dazu:
- `iex` und `Invoke-Expression` (ihre Wörter, ein `-Command` davor übersprungen), beide
  auch in `stringShells`, damit `shellWrites` sie als bekannt zählt;
- `pwsh`/`powershell -EncodedCommand` (`-e`, `-ec`, `-en`, `-enc` …): Base64 aus
  UTF-16LE wird dekodiert und als Zeile gelesen; was nicht dekodiert, liefert keine Zeile;
- `sh -o pipefail -c '…'`: Bei `sh`, `bash`, `zsh` und `dash` wird hinter `-o`/`+o` der
  Wert übersprungen, bevor `-c` gesucht wird;
- `cmd /c"…"` mit angeklebter Zeichenkette;
- `env -S '…'` und `env --split-string=…`: der Wert ist die Zeile.

Dazu kommt eine Shell, die ihr Skript von stdin liest: `sh`, `bash`, `zsh`, `dash` ohne
`-c` und ohne Skriptdatei (`-s` erlaubt), `pwsh`/`powershell -Command -`, `cmd` ohne
`/c`, sowie `iex` und `Invoke-Expression` ohne Argument, jeweils hinter einer Pipe. Als
Zeile gilt, was das Segment vor der Pipe ausgibt: die Wörter nach `echo`, `printf`
(ohne das Format) und `Write-Output`, und ein Segment, das nur aus einer Zeichenkette
besteht (`'…' | iex`). Ebenso der Here-String `bash <<< '…'`. Die Probe zeigte
`echo "loomux init" | sh` und `echo "echo x > .loomux/config.toml" | sh` als Durchlass.
`shellWrites` liest dieselben Zeilen. Was eine Datei in die Pipe gibt
(`cat script.sh | sh`), bleibt eine Grenze wie jedes eigene Skript.

Die gepinnten Löcher in `guard_test.go` (`sh -c "loomux init"`, `pwsh -c "loomux init"`,
`sh -c "loomux dev switchover prune-hooks …"`) werden zu Verweigerungen, die Doku-Stellen
dazu (`docs/*/cli-reference.md` „Known holes“, `docs/*/hooks.md` „Limits“) mit ihnen.

### B. Kopieren und Verschieben in einen Ordner

`destination` und `moved` bekommen den Ort `dir` und schreiben neben dem Ziel selbst
`ziel/basename(quelle)` für jede Quelle, wenn
- das Ziel auf `/` oder `\` endet, oder
- es mehr als eine Quelle gibt, oder
- das Ziel aus `-t`/`--target-directory` stammt, oder
- das Ziel unter `dir` ein vorhandener Ordner ist.

Bei einer rekursiven Kopie (`-r`, `-R`, `-a`, `--recursive`, `-Recurse`, jedes `rsync`)
zählt dieses Ziel als Ort, der überschrieben werden kann, wie das Quellziel eines
Verschiebens (`removes`): `cp -r /tmp/x/.loomux .` legt `config.toml` darunter neu.
Endet die Quelle auf `/.`, `/*` oder, bei `rsync`, auf `/`, landet ihr Inhalt im Ziel
selbst, und das Ziel zählt so. `robocopy src dst datei…` schreibt `dst/datei`,
`xcopy src\datei dst\` schreibt `dst\datei`.

### C. `patch` und Patch-Dateien

`patch` kommt in die Verbtabelle (`otherWriteVerbs`): Das erste Positionsargument ist die
geänderte Datei, `-o`/`--output` die geschriebene. Ohne Dateiargument, und bei `git apply`
und `git am`, liest der Wächter die Patch-Datei von der Platte (`-i`/`--input`, das zweite
Positionsargument von `patch`, die Positionsargumente von `git apply`/`am`, eine
Umleitung `<`) und nimmt die Ziele aus `diff --git a/X b/Y`, `--- X`, `+++ Y`,
`rename to X`, `copy to X`; `/dev/null` zählt nicht, ein Zeitstempel hinter einem
Tabulator wird abgeschnitten. `-pN` (bei `git` Vorgabe 1, bei `patch` ohne `-p` der
Basisname wie bei GNU patch: alle Varianten 0…Tiefe werden geprüft) nimmt N Elemente
vorn weg, `--directory=<d>` setzt `d/` davor. Ein Heredoc-Patch steht in der Zeile selbst;
die Zeile wird mit demselben Leser nach Kopfzeilen durchsucht. Eine Patch-Datei, die sich
nicht lesen lässt, verweigert („loomux cannot read the patch …“).

### D. Zuweisungen in derselben Zeile

`lineVariants` bekommt eine Variante, in der jede Zuweisung der Zeile eingesetzt ist:
bash `NAME=wert` (auch hinter `export`, `declare`, `local`, `readonly`, `typeset`),
`alias NAME=wert`, PowerShell `$NAME = wert`, `${NAME}=wert`, `$env:NAME = wert`,
`Set-Variable`/`sv`/`New-Variable`/`nv` (`-Name`/`-Value` oder positionsweise),
`Set-Alias`/`sal`/`New-Alias`/`nal`, cmd `set NAME=wert`. Eingesetzt wird an `$NAME`,
`${NAME}`, `$env:NAME` und `%NAME%`, für PowerShell ohne Rücksicht auf Groß- und
Kleinschreibung; ein Alias wird als ganzes Wort ersetzt. Bei mehreren Werten desselben
Namens entsteht je Wert eine Variante, höchstens 16. Eine Variante fügt nur Gründe hinzu;
der Ausnahmeweg (`plainLine`) liest die Zeile wie geschrieben, und eine Zeile mit `$` ist
nie schlicht.

### E. strict: eine Variable vor einem geschützten Ende

`strictReasons` prüft zusätzlich den Teil nach der letzten Expansion, wenn er mit einem
Schrägstrich ein neues Element beginnt: Gibt es einen echten Anfang des festen Teils
eines geschützten Globs (byteweise, kürzer als der ganze feste Teil), der vor dem Rest
einen geschützten Pfad ergibt, oder einen Pfad unter einem geschützten Ordner, wird
verweigert. So trifft `$D/config.toml` auf `**/.loomux/config.toml` und
`$D/state/hooks/x` auf `**/.loomux/state/hooks/**`. Zwei Fälle bleiben bei der
Schreibschranke: eine Expansion, die den ganzen geschützten Ordner halten könnte (sonst
wäre jedes `$D/x` verweigert), und ein Rest, der an der Expansion klebt (`${D}fig.toml`;
sonst wäre `build/$X.txt` für `X=.env` verweigert). Die Probe der Umsetzung zeigte beide
Fehlverweigerungen, bevor die Regel so eng wurde.

### F. 8.3-Kurznamen und Endpunkte, lexikalisch in jedem Modus

`judge.rels` fügt jeder Schreibweise zwei lexikalische Formen hinzu, ohne
Dateisystemzugriff:
- je Element abschließende Punkte und Leerzeichen abgestreift (nicht bei `.` und `..`);
- ein Element der Form `STAMM~N[.EXT]` durch jedes Literal-Element der geschützten Globs
  ersetzt, dessen 8.3-Alias passt: Stamm aus dem Namen ohne führende Punkte, ohne
  ungültige Zeichen, groß, bis sechs Zeichen; der Eingabestamm ist ein Präfix davon mit
  mindestens zwei Zeichen, oder die Hash-Form (zwei Zeichen, vier Hexziffern); eine
  Endung ist ein Präfix der ersten drei Zeichen der geschützten Endung, und ohne Endung
  passt nur ein Element ohne Endung.

Fehlverweigerungen sind möglich, wenn ein echter Ordner `LOOMUX~1` heißt; das ist hinnehmbar.

### G. strict: Pfade in Code-Strings

`namedPaths` schneidet jedes Wort zusätzlich an Anführungszeichen und Leerraum und
nimmt jedes Stück. `python -c "open('.loomux/config.toml','w')"` liefert so
`.loomux/config.toml`. Klammern, Kommas und Semikolons als weitere Schnitte standen im
ersten Entwurf; die Mutationsrunde zeigte, dass kein realistischer Fall sie braucht,
weil ein Pfad in Code als Zeichenkettenliteral in Anführungszeichen steht.

## Benannte Grenzen (Kommentar und Doku)

- Ein Programm, das der Agent selbst schreibt: ein Skript (`python fix.py`, `sh apply.sh`,
  `pwsh -File x.ps1`), ein Build-Werkzeug, im Standardmodus Code, der eine Datei selbst
  öffnet (`python -c "open(…)"`, `node -e`, `perl -e`). Code, der loomux als Befehl ruft
  (`python -c "os.system('loomux config apply')"`, `[scriptblock]::Create('…')`), wird
  schon verweigert: Der Schnitt ohne Rücksicht auf Anführungszeichen trennt an der
  Klammer, und das Segment dahinter beginnt mit loomux (Probe am 2026-10-01).
- Eine Patch-Datei, die sich zwischen dem Urteil und dem Lauf ändert, und ein Patch aus
  einer Pipe (`cat p.diff | git apply`).
- Eine Funktion, die in derselben Zeile definiert wird und loomux ruft
  (`function l { loomux $args }; l config apply`), eine Variable aus der Umgebung, die
  nicht in der Zeile gesetzt wird, und ein Wert, der selbst aus einer Expansion entsteht.
- Wrapper außerhalb von `dropPrefixes` (`script -c`, `taskset`) sind die Sache von
  fix/guard-winpty-stdbuf; `find -exec loomux …` und `xargs loomux` mit Argumenten von
  stdin werden hier als Lücken festgehalten, nicht geschlossen.

## Prüfung

- TDD je Abschnitt A–G: Test zuerst, roter Lauf mit der Assertion, dann der Code.
- **Differenzprobe:** `TestRecordTheHolesBattery` schreibt am Stand vor dem ersten Fix
  das Urteil (Bash, Standard und strict) jeder Zeile der Batterie;
  `TestTheHolesFixOpensNothing` prüft danach, dass keine verweigerte Zeile durchgeht und
  dass genau die erwarteten Zeilen kippen. Aufgenommen ist unter Windows, und beide
  Spalten hängen daran (strict löst über das Dateisystem auf, und ob `C:/…` oder `/tmp/…`
  absolut ist, bestimmt die Schreibweise im Standardmodus); der Vergleich läuft darum nur
  unter Windows, im Windows-Tor der CI. Die Batterie von feat/lane-probation hat dieselbe
  Abhängigkeit und vergleicht ihre Standardspalte auch unter Linux. Die Batterie enthält je geänderter Funktion
  eine Zeilenfamilie für jeden Aufrufer: `innerLine` (die `cd`-Nachführung in
  `sh -c "cd x && …"`, `$0`/`$1` hinter dem String, `pwsh -c pwsh -c …` an der
  Tiefengrenze), `destination`/`moved` (Kopie in einen Ordner ohne Schutz, `rsync -t`,
  `install`, `ln`), `lineVariants` (Fortsetzungen, Backticks, Klammern), `namedPaths`,
  `rels` (Write-Werkzeug wie Shell). Zeilen, die bewachte Befehle nennen, stehen in
  Testdateien, die per Write entstehen.
- **Mutanten:** je Regel per `go test -overlay` mit Pfaden in der Form `C:/…`, gezielte
  Tests; jeder überlebende Mutant bekommt einen Test oder eine Begründung.
- 100 % Coverage je Funktion; `sh ci/gate.sh` grün.
- Doku: `docs/en|de/hooks.md` (Limits), `docs/en|de/cli-reference.md` (Known holes),
  Kommentar an `readings` und am Lückenkommentar in `guard.go`.
