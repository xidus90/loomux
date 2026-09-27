# Wächter: Shell-Schreibziele als Pfade prüfen — Design

Stand 2026-09-28, nach Durchsicht durch Fable. Folgearbeit zu PR #50
(Flow-Laufzeit). Setzt dessen Wächterregeln voraus (Laufdateien, Flow-Ordner,
Torantworten); gebaut wird erst, wenn #50 auf `master` liegt, oder gestapelt auf
`feat/flow-runtime`.

## Ziel

Der Wächter verweigert einem Agenten Schreib- und Löschbefehle der Shell auf
geschützte Pfade in allen gängigen Schreibweisen, mit **einer** Regel je Pfad für
Write/Edit und Shell zugleich. Heute sucht `writeSource` (guard.go) per Regex nach
Verb plus Pfadwort; jede Lücke der Abschlussprüfung von #50 ist eine Variante
davon, einen Pfad per Regex zu erkennen.

Zwei Modi, gewählt vom Menschen:

- **`default`** hält gegen einen Agenten, der sich irrt oder bequem ist: gängige
  Verben, Wrapper, Globs, Braces, Eltern-Ordner.
- **`strict`** hält zusätzlich gegen einen Agenten, der gezielt umgehen will:
  Punkte/Leerzeichen am Pfadende, 8.3-Kurznamen, umbenanntes Binary, unbekannte
  Verben auf geschützten Pfaden, Expansionen vor einem geschützten Präfix.

Dazu zwei flaky Tests, die mit dem Wächter nichts zu tun haben, aber Torläufe
stören.

## Lücken und ihr Modus

| Lücke (belegt in der Abschlussprüfung von #50) | default | strict |
|---|---|---|
| `ln`, `tar`, `unzip`, `Expand-Archive`, `New-Item`/`ni`, `touch`, `robocopy`, `xcopy`, `rsync`, `find -delete`/`-exec rm`, `curl -o`, `wget -O`, `Invoke-WebRequest -OutFile` | ✔ | ✔ |
| Globs in festen Pfadteilen (`.loomux/sta*/runs`), Brace-Expansion (`{example,zz}`) | ✔ | ✔ |
| Wrapper-Schalter mit Wert (`xargs -n 1 …`, `sudo -u root …`, `timeout -s KILL 60 …`) | ✔ | ✔ |
| Löschen/Verschieben von `.loomux` bzw. einem Eltern-Ordner (`rm -rf .loomux`, `mv .loomux/flows x`) | ✔ | ✔ |
| Punkte/Leerzeichen am Pfadende (`example./`) | – | ✔ |
| 8.3-Kurznamen (`LOOMUX~1`) | – | ✔ |
| umbenanntes/kopiertes Binary (`doc.exe flow resume … --answer`) | – | ✔ |

Die Wrapper selbst (`env`, `xargs`, `timeout`, `nice`, `cmd /c` samt Schaltern,
`VAR=x`) kennt `dropPrefixes` (guard.go) schon; die #50-Lücke war, dass
`writeSource`s eigener Kopf nur `sudo|command|exec|nohup` kannte. Offen sind nur
die Schalter mit Wert, die heute als benannte Lücken in guard_test.go stehen.

## Aufbau

### Leser für Shell-Schreibziele (`internal/hooks/shellwrites.go`)

Die Zeile geht durch die bestehende Kette `lineVariants` → `segments` → `readings`
→ `dropPrefixes`; `dropPrefixes` lernt die Wrapper-Schalter mit Wert. Je Segment
liefert der Leser Ziele `{path string, removes bool}` nach einer Verbtabelle statt
einer Regex. Wörter, die mit `-` beginnen, und alles nach `--` sind keine Ziele
der „jede genannte Datei“-Verben; `2>&1`, `>&2` und ähnliche Umleitungen auf einen
Deskriptor ergeben kein Ziel.

- **jede genannte Datei:** `tee`, `Tee-Object`, `Set-Content`, `Add-Content`,
  `Out-File`, `Clear-Content`, `truncate`, `touch`, `rm`, `del`, `erase`,
  `Remove-Item` und Aliase (`ri`, `rd`, `rmdir`); `mv`/`move`/`Move-Item`/`mi`,
  `Rename-Item`/`ren`/`rni` (Verschieben löscht die Quelle und schreibt das Ziel);
- **nur das Ziel:** `cp`/`copy`/`Copy-Item`/`cpi`/`install` (letztes Argument oder
  `-Destination`), `rsync` (letztes Argument), `dd of=`, `ln` (letztes Argument),
  `tar` (`-C <dir>`; `-f`/`--file` beim Erzeugen), `unzip -d <dir>`,
  `Expand-Archive -DestinationPath`, `robocopy <src> <dst>`, `xcopy <src> <dst>`,
  `New-Item`/`ni` (`-Path`/erstes Argument), `curl -o|--output`, `wget -O`,
  `Invoke-WebRequest -OutFile`;
- **find:** mit `-delete` oder `-exec rm …` löscht es die Startpfade;
- **Umleitungen** `>`, `>>`, `>|`, `2>`, `&>` auf ihr Ziel;
- **in place:** `sed -i…`/`--in-place`, `perl -i…` auf jede Datei;
- **git:** `mv`, `rm` auf ihre Pfade; `checkout`/`restore` nur auf Pfade nach
  `--` oder auf Argumente, die als Pfad existieren — ohne Pfad (Zweigwechsel,
  `restore --staged`) kein Ziel; `clean` ohne Pfad löscht die Wurzel, mit Pfad
  diesen, `clean -n`/`--dry-run` ist ein Lesen;
- **.NET:** `[IO.File]::{Write*,Append*,Create,Delete,Move,Replace}` und
  `[IO.Directory]::{Delete,Move,CreateDirectory}`. Weil `splitSegments` bei `(`
  und `)` schneidet, liest der Leser für einen Kopf `[IO.…]::` den Rest der Zeile
  bis zur schließenden Klammer als Argumentliste (die Klammern selbst bleiben aus
  dem Segmentschnitt heraus).

**Unzerlegbar** heißt: der strikte Split (`shellwords.Split`) in `readings`
scheitert. Dann gilt jedes **pfadartige** Wort als mögliches Ziel — ein Wort, das
`/` oder `\` enthält oder mit `.` beginnt.

`lineVariants` macht aus `{` ein ` { `; eine Variante zerlegt
`.loomux/flows/{example,zz}/x` in `{`, `example,zz`, `}`. Diese Bruchstücke ergeben
kein Ziel; die unzerlegte Variante trägt die Brace-Expansion.

### Pfadnormalisierung (`internal/hooks/pathspell.go`)

Jedes Ziel wird relativ zur Projektwurzel aufgelöst (wie `relativePath`), mit
Schrägstrichen, ohne `./`-Segmente, ohne `:stream`-Anhang (nach dem Volume).

- **Braces** werden aufgefaltet, auch Bereiche (`{1..3}`), höchstens 64 Varianten;
  darüber wird verweigert. PowerShell faltet nicht, bash nicht in Quotes: das
  überverweigert, benannt.
- **Globs** (`*`, `?`, `[`) werden gegen die Platte aufgelöst, wie die Shell es
  tut: `filepath.Glob` relativ zur Wurzel, ein führendes `*` oder `?` trifft keinen
  Namen mit führendem `.` (bash-Standard). Trifft der Glob nichts, bleibt er als
  Literal stehen (auch wie bash) und wird als Pfad geprüft. Glob gegen Glob (ein
  Ziel `build/*` gegen eine Regel `*.pem`) wird so nie verglichen.
- **Nur strikt:** das absolute Ziel geht durch `guard.ResolvePath`
  (internal/brain/guard/path.go, `finalPath` über `GetFinalPathNameByHandle`), der
  schon 8.3-Aliase, Punkte/Leerzeichen am Ende, Groß-/Kleinschreibung und
  Junctions über den längsten existierenden Vorfahren auflöst; dann
  `filepath.Rel(root)`. Kein zweiter Resolver. Unter POSIX tut `finalPath`, was
  das System hergibt.

### Eine Prüfung für beide Wege (`checkTool`)

Die Shell-Ziele laufen durch dieselbe Schleife wie Write/Edit: eingebaute
Pfadregeln (in jeder Groß-/Kleinschreibung), `[policy] paths` des Projekts (wie
geschrieben) und `flowFolderReasons`. Die Regex-Regeln aus `writeSource` für
Manifest, Laufdateien und Flow-Ordner entfallen samt `writeSource` und
`flowFolderCommand`.

- **Unter jedem Verzeichnis:** `matchGlob` lernt ein führendes `**/`. Die
  eingebauten Regeln für Manifest, Laufdateien, Flow-Ordner und
  `.loomux/state/hooks` tragen es (`**/.loomux/config.toml` usw.), so dass eine
  Schreibung in einen `.loomux`-Ordner außerhalb der Wurzel (ein Geschwister-
  Worktree, `/repo/.loomux/config.toml`) weiter verweigert wird wie heute — für
  Shell und jetzt auch für Write/Edit.
- **Manifest:** eine eingebaute Pfadregel mit **einem** Grund für beide Wege:
  „.loomux/config.toml: the manifest is where the barrier reads its own limits, so
  no agent may write it“. Die bisherigen Wortlaute („so no shell command may write
  it“ in der Befehlsregel) weichen; die Grundlinien-Tests folgen dem neuen Text.
- **Eltern-Ordner:** ein Löschen oder Verschieben, dessen Ziel ein Vorfahr eines
  geschützten Pfads ist, wird verweigert (`.loomux`, `.loomux/state`,
  `.loomux/flows`, die Wurzel bei pfadlosem `git clean`). Kopieren in einen
  Vorfahren bleibt erlaubt (`cp -r mine .loomux/flows/`), wie in #50 entschieden.

Bleiben: die Befehlsregel `git push`, die Befehlsregeln des Projekts
(`[policy] commands`), `writesConfiguration`, `answersAGate`.

**Bewusste Verschärfung:** `[policy] paths` des Projekts gilt künftig auch für
Shell-Schreibbefehle, bisher nur für Write/Edit. Im eigenen Repo kippen dadurch
(Plan zählt jede Stelle): `rm coverage.out`, `mv bin/loomux.exe …` (Regel
`bin/*`), `rm -rf .loomux/state/…` (Regel `.loomux/state/**`), Schreibungen unter
`testdata/cases/` (Aufnahmen). In den Tests kippen bewusst: `rm -rf .loomux`,
`mv` eines Eltern-Ordners, `rm -r .loomux/state/hooks` (guardflow_test.go), die
Wrapper-Schalter-Lücken in guard_test.go. Wer im eigenen Projekt so etwas
braucht, lässt es einen Menschen tun oder passt `[policy]` an. Das Changelog
bekommt dafür einen `Changed`-Eintrag neben `Fixed`; das Label bleibt
`release:minor`, weil AGENTS.md eine strengere Wächterregel nicht unter „major“
führt (dort stehen Befehl, Flag, Hook-Protokoll, Konfigformat, Exit-Code).

### Modus

`.loomux/config.toml`, Schema-Modul Base:

```toml
[guard]
mode = "default"   # oder "strict"
```

Fehlt der Schlüssel, gilt `default`. Ein anderer Wert ist ein Konfigurationsfehler
wie jeder andere; der Wächter verweigert dann nach seinem Verhalten bei kaputter
Konfiguration und fällt nicht still auf `default`. Strikt kommt hinzu:

1. Auflösung über `guard.ResolvePath` (Punkte/Leerzeichen, 8.3, Junctions).
2. Torantworten (`flow resume … --answer`) und Konfigurationsbefehle (`init`,
   `config set|unset|apply|reject`, `area add`, `merge-hook install|remove`)
   werden an den Argumenten erkannt, unabhängig vom Programmnamen.
3. Ein Segment, das einen geschützten Pfad mit einem Verb nennt, das weder in der
   Verbtabelle noch in der Leseliste steht, wird verweigert. Leseliste: `cat`,
   `type`, `Get-Content`, `less`, `more`, `head`, `tail`, `wc`, `stat`, `file`,
   `jq`, `ls`, `dir`, `Get-ChildItem`, `grep`, `rg`, `Select-String`, `diff`,
   `git diff|log|show|status|blame|add|commit`, und loomux' lesende Befehle
   (`flow list|show`, `config get|list|proposals`, `check …`).
4. Ein Schreibziel mit einer Expansion (`$X`, `$(…)`, Backticks, `%X%`) wird
   verweigert, wenn der feste Teil davor ein geschützter Pfad oder dessen Vorfahr
   ist oder auf ihn passen könnte.

## Grenzfälle

- Ein Werkzeugaufruf ohne erkennbare Befehlszeile wird dort verweigert, wo er es
  heute ist (`judgedOrRefused`).
- Variablen im Pfad bleiben im Standardmodus eine benannte Grenze.
- Jeder Grund erscheint je Werkzeugaufruf höchstens einmal — für Shell **und**
  Write/Edit (bei Write/Edit mit mehreren Zielen neu; bewusst).
- `sh -c "…"` / `bash -c "…"`: der innere Text wird gelesen, soweit `readings` es
  heute tut; was darüber hinausgeht, bleibt benannte Grenze.

## Tests

- **Grundlinie:** Alle bestehenden Tests der Shell-Schreibregeln (der Manifest-Block
  in guard_test.go, ≈80 Zeilen, die Zeilen für Laufdateien und Flow-Ordner, die
  erlaubten Zeilen) laufen gegen den neuen Weg und bleiben grün, bevor
  `writeSource` fällt — bis auf die oben gelisteten bewussten Kippungen und den
  neuen Manifest-Grund; der Bericht nennt jede geänderte Zeile.
- **Neue Zeilen je Lücke** in beiden Richtungen: verweigert jede Zeile der
  Lückentabelle (default-Spalte) und Schreibungen in einen `.loomux` außerhalb der
  Wurzel; erlaubt `cp -r mine .loomux/flows/`, `cat`/`ls` geschützter Pfade,
  `git status`, `git checkout feat/x`, `git checkout -b x`, `git restore --staged
  x`, `git clean -n`, `rm -rf build/*` neben einer Regel `*.pem`, `rm -rf *` an der
  Wurzel (trifft `.loomux` nicht).
- **Strikt:** Punkte am Ende, `LOOMUX~1` (echter Kurzname unter `t.TempDir()`,
  unter Windows scharf, sonst Skip mit Grund), `doc.exe flow resume 0001 --answer
  yes`, unbekanntes Verb auf geschütztem Pfad, `> .loomux/$X`, und die Leseliste
  bleibt erlaubt.
- **Projektregeln:** eine `[policy] paths`-Regel verweigert jetzt `echo x > datei`.
- **Schema:** `[guard] mode` im Schema, Drift-Test der Leser, ungültiger Wert.

## Messung

`loomux dev bench hooks` vorher/nachher mit den fünf Fällen aus
`testdata/bench/flow-hooks.json` und einem sechsten: `pre-tool-use` mit einer langen
Shell-Zeile (mehrere Segmente, Wrapper, Braces, ein Glob). Die stdin-Datei des
sechsten Falls liegt neben denen der übrigen (dort nicht eingecheckt), die
Fall-Datei nennt sie. Budget: 35 ms warm für `pre-tool-use`; heute 9–13 ms. Eintrag
in `docs/{en,de}/benchmarks.md`.

## Doku

`docs/{en,de}/hooks.md`: Regeltabelle ohne Regex-Formen, Liste der erkannten Verben
und Wrapper, der Modus, die Grenzen im Standardmodus, der Hinweis auf
`[policy] paths` für die Shell und auf `**/`. `docs/{en,de}/configuration.md`:
`[guard] mode`. Changelog `Added` (Modus), `Changed` (Projektregeln auf der Shell,
ein Manifest-Grund), `Fixed` (die Lücken).

## Flaky Tests

- **`TestCases4c1`** (`internal/cli/cases_4c1_test.go:68`) lauscht fest auf
  `127.0.0.1:11435`, und dieselbe Adresse steht als `endpoint` in den
  Eingabewelten `testdata/cases/4c1/reconcile/*/world/config.toml`. Der Test
  lauscht auf `127.0.0.1:0`, schreibt die tatsächliche Adresse vor dem Lauf in die
  Arbeitskopie der Welt (Aufnahmen bleiben unverändert) und normalisiert beim
  Vergleich `world_after/config.toml` und die Ausgabe zurück auf die aufgezeichnete
  Adresse.
- **`TestGitReadsTheIndexACommitHookHandsIn`** (`internal/code/query/git_test.go:37`)
  fiel nur im Torlauf aus dem Pre-commit-Hook („with the hook's index: false nothing
  staged“), wo git selbst `GIT_INDEX_FILE`, `GIT_DIR` u. a. setzt. Ursache unbekannt;
  der Plan beginnt mit systematischem Debuggen: im Hook-Kontext reproduzieren, dann
  die Hypothesen prüfen — geerbte git-Umgebung; `indexFileFor`
  (code/query/git.go:77) verwirft einen Index unter `worktrees/` und vergleicht über
  `resolved()`, und unter dem Hook können `TMP`/`TEMP` anders geschrieben sein als in
  der Shell, so dass `t.TempDir()` und `git rev-parse --absolute-git-dir` in der
  Schreibweise auseinanderlaufen. Erst danach ein Fix.

## Nicht Teil davon

Prozesse, die Dateien selbst öffnen (`python -c …`, ein Build-Werkzeug), ein Agent
mit `--root` auf einer Projektkopie, Variablen im Pfad im Standardmodus — benannte
Grenzen wie bisher.
