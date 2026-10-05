# Loomux CLI-Referenzhandbuch

Dieses Handbuch dokumentiert Syntax, Flags, Standard-I/O-Verträge und Exit-Codes aller Loomux-Kommandozeilenbefehle.

---

## 1. Globale Konventionen

### Exit-Codes
Loomux nutzt eine strikte Exit-Code-Semantik, die exakt auf die Schnittstellen moderner Coding-Agenten abgestimmt ist:

| Exit-Code | Bedeutung | Verhalten im Harness (Claude / Antigravity) |
|---|---|---|
| **`0`** | **Erfolg / Erlaubt** | Die Werkzeugausführung wird fortgesetzt; die Runde ist erfolgreich. |
| **`1`** | **Mitteilung / Warnung** | Vom Harness als rein informativ bzw. „weitermachen“ interpretiert. |
| **`2`** | **Strikte Ablehnung / Blockiert** | Die Schreibschranke oder Policy hat die Aktion blockiert; Abbruch des Aufrufs. |

Unter `--host antigravity` (gemessen mit agy 1.2.8 und 1.2.11, 2026-09-25):
Die `2` von `pre-tool-use` verweigert den Aufruf wie in Claude Code, die `2`
von `post-tool-use` gibt stderr als Warnung an das Modell, ohne abzubrechen.
Ein gehaltener Stop wird zu `{"decision":"continue","reason":"…"}` auf
stdout mit Exit `0`, der Grund ist, was das Tor nach stderr geschrieben hat;
agy tritt dann erneut in seine Schleife ein. Jeder andere Code ungleich 0
endet mit `0`, seine Meldung auf stderr. Was `post-tool-use` bei Exit `0` auf
stdout schreibt, die übersprungenen Lanes und die Aufrufer des
Blast-Monitors, erreicht agy als `injectSteps`, die es dem Modell nach einem
PostToolUse zeigt (gemessen mit agy 1.2.12, 2026-09-28).

### Globale Flags & Umgebung
- `--root <pfad>`: Explizite Angabe der Projektwurzel. Wird dieses Flag weggelassen, wandert Loomux im Verzeichnisbaum aufwärts, bis es die erste `.loomux/config.toml` findet.
- `LOOMUX_STATE_DIR`: Überschreibt das globale Zustandsverzeichnis (Standard: `%LOCALAPPDATA%\loomux` unter Windows, `~/.local/state/loomux` unter POSIX).
- `loomux version` (auch `--version`, `-v`): gibt `loomux <version>` auf `stdout` aus und endet mit `0`; ein Entwicklungsbuild nennt `0.0.0-dev`.

---

## 2. Policy & Prüfketten (`loomux check`)

`loomux check` nimmt zuerst eine **Anfrage** und danach ihre Flags. Die Anfrage
ist ein Profil (`edit`, `precommit`, `stop` oder eins aus `[verify.profiles]`), `all`,
eine Komma-Liste von Arten (`lint,types`) oder einer der fünf eingebauten
Prüfbefehle unten. Was jede Art je Stack fährt, legen
[`[verify]`](configuration.md#verify-prüfketten--quality-gates) und die Presets
fest.

### `loomux check <anfrage> [--root <pfad>] [--show] [-v] [--arm]`
Fährt die Lanes der angefragten Arten für jeden aktiven Stack und Bereich und
urteilt über sie.

- **Flags** (nach der Anfrage; `check --show` allein ist ein Aufruffehler):
  - `--root <pfad>` — Projektwurzel; ohne Angabe aufwärts vom
    Arbeitsverzeichnis gesucht, das Arbeitsverzeichnis, wenn nichts gefunden
    wird. Ein Projekt ohne `.loomux/config.toml` wird allein mit den Presets
    geprüft.
  - `--show` — nichts fahren; die wirksamen Lanes als `[verify]`-Tabellen
    drucken.
  - `-v` — auch die Ausgabe grüner Lanes drucken.
  - `--arm` — nur beim Profil `precommit`, sonst Exit `2`. Nach einem Lauf,
    der insgesamt grün endet, schreibt es jede Lane, die `ok` endete, in
    `.loomux/armed.toml` und stagt die Datei mit `git add` in den Index des
    Commits, der gerade läuft. Es schreibt nur, wo die Datei existiert: ohne
    sie ist jede Lane ohnehin scharf. Bei einem Teilcommit (`git commit
    <pfad>`, `--only`) schreibt und stagt es nichts und druckt `not armed:
    this commit takes only some paths; the next whole commit arms the lanes`.
    Die Zeile `armed: <schlüssel>` nennt, was es eingetragen hat. Eine Datei,
    die sich nicht schreiben oder stagen lässt, steht auf `stderr` und ändert
    den Exit-Code nicht.
- **Lanes in Probe**: Mit einer `.loomux/armed.toml` (siehe
  [Konfiguration](configuration.md#schonfrist-je-lane-loomuxarmedtoml)) läuft
  eine Lane, die die Datei nicht nennt, trotzdem. Eine rote liest sich
  `<zustand> (probation)` und lässt nichts scheitern, eine grüne oder
  übersprungene liest sich wie zuvor, und der Bericht jedes Laufs mit einer
  solchen Lane endet mit `probation: <schlüssel> (warn only until a green
  commit arms them)`. Eine Datei, die sich nicht lesen lässt, stellt jede Lane scharf, und
  `stderr` sagt warum.
- **Reihenfolge**: Eine Lane startet, sobald die Lane, auf die sie wartet
  (`after`), fertig ist, mit höchstens `max_parallel` Prozessen gleichzeitig.
  Der Bericht kommt am Ende, nie verzahnt: Arten in Anfrage-Reihenfolge, darin
  Stacks in Byte-Ordnung, dann Bereiche. Die eine Ausnahme ist `lint/wiki`, der
  Lint über das Wiki-Bündel, der in diesem Prozess läuft, wo `lint` angefragt
  ist und das Projekt ein Wiki hat: er kommt zuletzt. Er prüft nur die Struktur
  des Bündels; die Drift-Regel bleibt bei `loomux wiki-gate`.
- **Graph-Lane**: `check stop` beurteilt die `graph`-Lane wie der Stop-Hook —
  den Arbeitsbaum samt ungetrackter Dateien gegen `HEAD`, über eine Kopie des
  Index im Git-Verzeichnis. Es fährt nur die Lanes; das Tor um sie herum (der
  Marker, ein schon grün befundener Baum, die Befunde der Subagenten, der
  Blockzähler) bleibt beim Hook. `check precommit` und eine Liste von Arten
  lesen den echten Index.
- **Ausgabe** (alles auf `stdout`): eine Zeile je Lane,
  `<art>/<stack>[@<bereich>]: <zustand> [<herkunft>]`, dahinter die Dauer bei
  einer Lane, die gestartet ist, `by <lane>` bei einer blockierten oder der
  Grund bei einer, die nie startete. `<herkunft>` ist `preset`,
  `preset, variant <signal>`, `config` oder `in-process`. Eine rote Lane druckt
  ihre Ausgabe unter ihre Zeile, eine grüne nur mit `-v`. Eine Lane mit
  mehreren Befehlen druckt einen Block je Befehl mit der Kopfzeile `$ <argv>`,
  bei einem roten mit `(failed)`. Eine Art, die nichts zu prüfen hatte,
  schließt den Bericht mit ``nothing to check for `<art>` ``, wenn die
  Anfrage sie nannte -- eine Liste von Arten oder ein Profil, das das Projekt
  in `[verify.profiles]` setzt -- oder wenn gar keine Lane der Anfrage lief;
  eine Art aus `all` oder einem eingebauten Profil neben einer Lane, die lief,
  schließt ihn stattdessen mit ``no lane for `<art>` here, left out``.
  ```text
  lint/go: ok [preset] 0.2s
  types/go: not-applicable [preset] no command
  test/go: ok [preset] 0.5s
  coverage/go: failed [preset] 0.1s
  not covered: w.go:3 A 66.7%
  ```
- **Coverage-Dateien**: Messende Lanes schreiben nach `.loomux/state/cover/`,
  das `check` und der post-edit-Hook selbst anlegen.
  Ein grüner Lauf löscht seine eigenen Dateien, ein roter behält sie; Dateien
  anderer Läufe gehen, sobald sie 24 Stunden alt sind.
- **Exit-Codes**: `0` (keine scharfe Lane rot, und jede angefragte Art hatte
  eine Lane, die lief, oder ist irgendwo `not-applicable`), `1` (eine scharfe Lane ist rot,
  eine genannte Art hatte nichts zu prüfen, keine Lane der Anfrage lief, oder `[verify]` bzw. die Anfrage lässt sich
  nicht laden; ein Ladefehler ist eine Zeile auf `stderr`), `2` (fehlerhafter
  Aufruf: keine Anfrage, ein Flag vor der Anfrage, ein unbekanntes Flag, eine
  zweite Anfrage).

### `loomux check <anfrage> --show`
Druckt als TOML, was die Anfrage fahren würde, und fährt nichts.

```text
# max_parallel = 16, timeout = 600s

# go: areas ., tests found
[verify.go.lint]
commands = ["go vet ./...", "{loomux} check gofmt ."]  # preset
on_file = ["go vet ./...", "{loomux} check gofmt {file}"]  # preset
threaded = true  # preset

# [verify.go.types] not defined
```

- Die erste Zeile trägt die Grenzen des Laufs; jeder aktive Stack bekommt
  einen Kommentar mit seinen Bereichen und seinem Teststand (`tests found`,
  `no tests found`, oder `tests not detected` bei einem Stack ohne
  Testsignal).
- Jeder Schlüssel trägt die Herkunft seiner **Lane**: Eine Tabelle, die einen
  Schlüssel eines Presets ändert, markiert die ganze Lane als `config`.
- Eine Budgetzeile gibt es nicht: `loomux check` hat kein Budget. Das Budget
  von post-edit ist das Hook-Flag `--budget`.
- Die Ausgabe lädt, wie sie dasteht, und lässt sich nach
  `.loomux/config.toml` kopieren. Eine Lane ohne Befehl ist ein Kommentar,
  `# [verify.<stack>.<art>] not defined`; eine abgeschaltete Lane steht als
  `<art> = false` unter `[verify.<stack>]`, vor den übrigen Tabellen des
  Stacks. Eingefügt bleiben beide, wie sie waren.

### `loomux check gocover --profile <pfad> [--floor <n>] [--dir <verz>]`
Beurteilt ein Go-Coverage-Profil des Moduls in `--dir` über
`go tool cover -func`. Das fährt die `coverage`-Lane des Go-Presets.

- **Flags**: `--profile` (Pflicht; relativ zu `--dir`), `--floor <n>` (eine
  Gesamtgrenze in Prozent statt des Tors je Funktion), `--dir` (Verzeichnis
  mit `go.mod`; Vorgabe `.`). Der Modulpfad kommt aus `go.mod`.
- **Ohne `--floor`**: jede Funktion bei 100 %, außer
  `//coverage:exempt <grund>` ist die letzte Kommentarzeile direkt über ihrem
  `func`. Jede andere Funktion ist eine Zeile auf `stdout`:
  `not covered: <datei>:<zeile> <func> <prozent>%`. Ein Profil ohne Funktionen
  fällt durch.
- **Mit `--floor`**: druckt `coverage <gesamt>%` oder fällt durch mit
  `coverage <gesamt>% is below the floor of <n>%` auf `stderr`.
- **Exit-Codes**: `0` (bestanden), `1` (eine Funktion oder die Summe unter dem
  Tor, kein `go.mod`, ein unlesbares Profil), `2` (kein `--profile`, ein
  unbekanntes Flag).

### `loomux check commit-msg [flags] [<datei>]`
Prüft eine Git-Commit-Nachricht auf Sprache und Einhaltung der Formatregeln oder kalibriert Schwellenwerte gegen die Git-Historie.

- **Argumente**:
  - `<datei>`: Pfad zur Commit-Nachrichtendatei (`COMMIT_EDITMSG`). Kann nicht zusammen mit `--calibrate` angegeben werden.
- **Flags**:
  - `--root <pfad>`: Pfad zur Projektwurzel (Standard: sucht aufwärts nach `.loomux/config.toml` oder `.git`).
  - `--calibrate <N>`: Misst Ablehnungsraten über die letzten `N` Commits statt eine Datei zu prüfen.
  - `--language <en|de>`: Sprache zur Kalibrierung (Standard: `[commit].language` oder `"en"`). Kann bei der Prüfung einer Datei nicht verwendet werden.
- **Verhalten**:
  - **Wortschatz & Sprache**: Prüft alle Zeilen auf fremdsprachige Stopwörter (Variante B). Bei Zielsprache `en` zählen Wörter mit Umlauten als Treffer, und 82 deutsche Entwicklerwörter (`fehler`, `datei`, `behebe`, `aktualisiere` …) bilden eine vierte Wortquelle neben der deutschen, englischen und romanischen.
  - **Fremdschriften**: Erkennt Läufe von Nicht-Latein-Schriftzeichen (CJK-Ideogramme, Hiragana, Katakana, Kyrillisch etc.) und zählt jeden Lauf als Fremdwort-Treffer.
  - **Ausnahmen & Spannen**: Ignoriert Text in mehrzeiligen Backtick-Codeblöcken (``` `...` ```), einzeiligen Anführungszeichen (`"..."`; ein Apostroph begrenzt nichts), Git-Kommentarzeilen (`#`), Scherenzeilen (`# ------------------------ >8 ------------------------`), Git-Trailers (`Signed-off-by:`, `Co-authored-by:`) im Rumpf, Pfad-Token, Namenspartikel (`van`, `von`), Bezeichner mit Binde-/Unterstrichen sowie jede Zeile, auf die ein Muster aus `[[commit.allow]]` passt — sie wird ganz übersprungen.
  - **Conventional Commits**: Wenn `[commit].conventional = true` (Standard), wird geprüft, ob die Betreffzeile dem Schema `<type>[(<scope>)][!]: <description>` entspricht.
- **Exit-Codes**:
  - `0`: Gültige Commit-Nachricht bzw. Kalibrierung erfolgreich abgeschlossen.
  - `1`: Commit-Nachricht abgewiesen (Begründung und Zeilen auf `stderr`) oder Konfigurations-/Git-Fehler.
  - `2`: Syntaxfehler im Aufruf (ungültige Flags oder Argumente).

### `loomux check gofmt [pfade...]`
Überprüft Go-Quelldateien auf Formatierungskonformität, ohne sie zu verändern.

- **Argumente**: Optionale Verzeichnisse oder Dateipfade (Standard: Arbeitsverzeichnis).
- **Exit-Codes**: `0` (Korrekt formatiert), `1` (Unformatierte Dateien auf `stdout` gelistet, oder ein Pfad, der sich nicht lesen lässt, gemeldet als `loomux check gofmt: <grund>` auf `stderr`).

### `loomux check graph-fresh [--root <pfad>] [--wait <dauer>]`
Die erste Hälfte der Graph-Lane: bringt den Graphen auf der Platte auf den
Stand des Baums, bevor `blast-audit` ihn liest.

- **Flags**: `--root <pfad>` (Projektwurzel; leer: nach oben gesucht),
  `--wait <dauer>` (wie lange auf den Neubau eines anderen Laufs gewartet
  wird, bis er die Sperre freigibt; Vorgabe `30s`).
- **Verhalten**: Drift, ein fehlender Frische-Datensatz, ein Graph einer
  anderen Extraktor-Version und ein veraltetes Schema bauen den Graphen unter
  der prozessübergreifenden Sperre neu und sind grün. Eine Sperre, die älter
  als eine Stunde ist, wird übernommen. Fortschrittsnotizen gehen auf `stderr`.
- **Ausgabe**: `graph rebuilt` oder `graph is fresh` auf `stdout`.
- **Exit-Codes**: `0` (frisch oder neu gebaut), `1` (keine Projektwurzel, gar
  kein Graph — einen ersten baut er nie —, ein fehlgeschlagener Neubau, eine
  fehlgeschlagene Probe oder eine Sperre, die nach `--wait` noch gehalten
  wird, mit Pfad und Alter genannt), `2` (ein unbekanntes Flag).

### `loomux check blast-audit [--root <pfad>] [--cached | --base <ref>] [--threshold <n>] [--skip-test-callers]`
Die zweite Hälfte der Graph-Lane: rot, wenn ein geänderter Bereich mit genug
Aufrufern keinen geänderten Test hat, der ihn erreicht.

- **Flags**: `--root <pfad>`; `--cached` (der Index gegen `HEAD`, das nutzt
  die Lane); `--base <ref>` (`<ref>...HEAD`); ohne beide der Arbeitsbaum gegen
  `HEAD` oder, bei sauberem Baum, der letzte Commit, wie bei `graph blast`.
  `--threshold <n>` (Vorgabe `3`; die Presets setzen `5`);
  `--skip-test-callers` (nur Aufrufer außerhalb von Testdateien zählen:
  `_test.go` in Go; `test_*.py`, `*_test.py`, `tests.py`, `conftest.py` und
  jede Datei unter einem Verzeichnis `tests/` oder `test/` in Python).
- **Der Befund**: ein Bereich mit dem Testsignal `none` oder `stale`, der
  einen Seed mit mindestens `<n>` eingehenden Walk-Kanten hat. Der Befehl liest
  den Graphen, wie er ist, und baut ihn nie neu; das ist Sache von
  `graph-fresh`.
- **Ausgabe**: bei einem Befund `blast audit: <bereich>, threshold <n>` und
  eine Zeile je rotem Bereich, `<pfad> [<signal>]: <seed> in-degree <k>, …`;
  sonst `no area at or above <n> callers lacks a changed test (<m> areas)`.
- **Exit-Codes**: `0` (kein Befund), `1` (ein Befund oder ein Fehler auf
  `stderr`: keine Projektwurzel oder kein Graph, `--base` zusammen mit
  `--cached`, eine Basis, die mit `-` beginnt, eine Schwelle unter 1, ein
  git-Fehler), `2` (ein unbekanntes Flag). Der Exit-Code allein unterscheidet
  Befund und Fehler nicht; `stdout` tut es.

---

## 3. Agenten-Harness-Hooks (`loomux hook`)

Hook-Einstiegspunkte werden von Coding-Agenten synchron bei Werkzeugaufrufen gestartet. Jedes Ereignis außer `pre-tool-use` endet ohne Wirkung mit `0`, wenn `[modules] hooks = false` gilt, und mit `1`, wenn kein `--root` angegeben ist und aufwärts keine `.loomux/config.toml` gefunden wird. Ein unbekanntes Ereignis endet auf jedem Host mit `2`.

```bash
loomux hook <event> --host <claude|antigravity|codex> [--root <pfad>]
```

### `loomux hook pre-tool-use`
Prüft Projekt-Policy und globale Schreibschranke, bevor der Agent ein Werkzeug ausführt.

- **Standard-Input (stdin)**: JSON-Nutzlast des aufrufenden Agenten:
  ```json
  {
    "tool_name": "Write",
    "tool_input": {
      "file_path": "C:/Projekte/repo/.env"
    }
  }
  ```
- **Laufzeit-Budget**: `<35ms` Kaltstart-Boden.
- **Kein Modul**: `[modules]` wird hier nicht gelesen. Die Schreibschranke ist
  global und schützt die schreibgeschützten Bereiche anderer Repositories;
  `hooks = false` lässt den Wächter darum laufen.
- **Befehle, die ein Mensch ausführt**: Eine `Bash`- oder `PowerShell`-Zeile,
  die einen dieser Befehle ausführt, wird verweigert:
  - `loomux init`, außer mit befreiendem `--dry-run` oder `--detect-only`;
  - `loomux config`, außer `config list …`, `config get …`,
    `config proposals …`, einem alleinstehenden `config --help` oder
    `config -h` und `config set …` oder `config unset …` mit befreiendem
    `--propose`;
  - `loomux area add`;
  - `loomux merge-hook install` oder `remove` (`status` und `record` gehen
    durch);
  - `loomux gate arm` und `loomux gate disarm` (`gate status` geht durch); die
    Ablehnung sagt, dass arm und disarm entscheiden, welche Lanes das Tor
    scheitern lassen, und dass `loomux gate status` sie zeigt;
  - `loomux convert` oder `loomux fetch`, außer allein mit `--help` oder `-h`.

  Die Ablehnung lautet ``loomux init, config and
  area add write the configuration the guard reads, merge-hook install and
  remove write executable hooks into repositories, and convert and fetch
  write into an area's inbox, which the write barrier keeps from agents; a
  human runs them. An agent proposes a change with
  `loomux config set|unset … --propose`, which a human applies``.
  `config apply` und `config reject` bleiben verweigert.
  - **Wann ein Flag befreit** — eine Positivliste, geprüft an der Zeile, wie
    sie geschrieben steht, vor jeder Umformung. Das Flag befreit nur, wenn
    alle drei gelten:
    - Der Aufruf ist direkt: Das loomux-Programm (`loomux`, ein Pfad darauf
      oder `go run` von `cmd/loomux`) ist das erste Wort eines Befehls, den
      die Zeile selbst beginnt — am Zeilenanfang oder direkt hinter einem
      `;`, `|`, `&&` oder Zeilenumbruch außerhalb von Anführungszeichen. Ein
      loomux in einer gequoteten Zeichenkette
      (`cmd /c 'x; loomux init --dry-run'`, `sh -c 'true` + Zeilenumbruch +
      `loomux …'`) wird trotzdem gefunden und verweigert, ist aber nie
      befreit. Hinter jedem Vorsatz — `cmd /c`, `sudo`, `env`, `nice`, `timeout`,
      `xargs`, `exec`, `command`, `Start-Process`, `VAR=wert`, einer
      Umleitung, einem Schlüsselwort wie `then` oder `!`, einer `{` — befreit
      das Flag nichts, denn ein Wrapper kann die Wörter ein zweites Mal lesen:
      `cmd /c` löst `^`, `%X%`, `!X!` und `"` auch in dem auf, was die Shell
      als ein einfach gequotetes Wort weitergab.
    - Die ganze Zeile ist schlicht. Außerhalb von Anführungszeichen stehen
      nur Buchstaben, Ziffern, Leerraum, `. _ / : = , + -`, die Trenner `;`
      `|` `&&` und Zeilenumbrüche, die Umleitungen `<` `>` (mit `&` darin,
      `2>&1`, `&>`) und ein `#` am Wortanfang (ein Kommentar; der Rest seiner
      Zeile wird nicht geprüft). Ein alleinstehendes `&` — der Aufrufoperator
      von PowerShell oder ein Hintergrundjob — ist nicht schlicht. In
      doppelten Anführungszeichen darf nur die erste dieser Mengen stehen,
      also kein `$`, kein Backtick, kein `\`. In einfachen Anführungszeichen
      darf jedes ASCII-Zeichen stehen, weil weder bash noch PowerShell dort
      etwas expandiert, außer den Trennern `; | & ( ) < >` und `^ % !` von
      cmd, die ein Wrapper, der die Zeichenkette bekommt, erneut lesen
      könnte, und `#`, weil ein gequotetes `#x` als Wort beim Programm
      ankommt, das der Wächter als Kommentar läse; und keines jenseits von
      ASCII, weil PowerShell eine
      einfach gequotete Zeichenkette auch an einem typografischen
      Anführungszeichen beendet. Ein offenes Anführungszeichen ist nicht
      schlicht. Alles andere — `$`, ein Backtick, `\`, `( ) { } [ ]`,
      `* ? ~ ! @`, ein `#` mitten im Wort — macht die Zeile unschlicht. Auch
      `%` ist nicht schlicht: Nach dem Stop-Parsing-Zeichen `--%` von
      PowerShell geht der Rest der Zeile mit aufgelöstem `%X%` an das
      Programm.
    - Unter den Argumenten steht das Flag (`--propose` oder `-propose`;
      `--dry-run` oder `--detect-only`) als eigenes Wort vor jedem `--`,
      ohne ein `-name=…` desselben Flags daneben (das letzte gilt,
      `--dry-run --dry-run=false` wird also verweigert). Ein `#`-Wort beendet
      die Argumente. Ab der ersten Umleitung dürfen nur Umleitungen und ihre
      Ziele folgen (`> out`, `>out`, `2>&1`): `> out --propose=false` reicht
      dem Programm `--propose=false` trotzdem weiter.

    Ein Wert mit `$`, `*`, `{…}`, `?`, `[…]` oder einem Backslash muss darum
    in einfachen Anführungszeichen stehen (`loomux config set index.include
    'docs/**/*.md' --propose`), und ein befreiter Befehl muss für sich
    stehen, nicht in einem Block (`try { … }`) und nicht hinter einem
    Programmpfad, der expandiert (`${X}/loomux`).

  Diese Befehle schreiben aus ihrem eigenen Prozess — `.loomux/config.toml`,
  einen Git-Hook in einem anderen Repository, die `settings.json` mit den
  Hook-Einträgen eines Projekts (darunter die des Wächters selbst), eine Datei
  im Eingang eines Bereichs —, wo keine Pfadregel den Schreibvorgang sieht. Erkannt wird das Programm als
  `loomux`, `loomux.exe` oder ein Pfad, der auf eines von beiden endet (mit
  oder ohne Anführungszeichen, `\` oder `/`), und als `go` (oder `go.exe`)
  `run` von `cmd/loomux` oder `cmd/loomux/main.go` (mit oder ohne `./`, unter einem
  Modulpfad, in jeder `@version`, hinter Build-Flags). Gefunden wird es
  hinter `VAR=wert`, Umleitungen (`2>/dev/null`, `> out`, `2>&1`), den
  reservierten Wörtern `if`, `then`, `else`, `elif`, `while`, `until`, `do`,
  `!`, `{`, `coproc`, `function <name>`, `try`, `catch`, `finally` und
  PowerShells Dot-Source `.`
  (jedes reservierte Wort nur als das Wort selbst: eine Datei dieses Namens,
  `./do`, ist ein Programm),
  hinter jedem `{` oder `}` der Zeile, allein oder an ein Wort geklebt (dem
  Rumpf eines Blocks, einer Funktion oder eines Skriptblocks: `try{`,
  `{loomux …}`), sowie hinter den Wrappern `sudo`, `command`, `exec`,
  `nohup`, `env`, `time`, `xargs`, `nice`, `ionice`, `stdbuf`, `winpty`,
  `setsid`, `chronic`, `unbuffer`, `timeout <dauer>` und `cmd` mit
  jedem Schalter bis `/c`, `/k` oder `/r` (der Befehl darf daran geklebt
  sein, `/cloomux`), eine aufgelöste Caret-Maskierung von cmd (`con^fig`),
  jeder externe auch als `<name>.exe`,
  samt ihren Flags (und `--` und dem
  einzelnen `-` von `env`). Die Flags liest er, wie getopt sie liest: Der
  eigene Wert eines Flags von `sudo`, `env`, `xargs`, `nice`, `ionice`,
  `stdbuf`, `timeout`, `exec`, `time` oder `unbuffer`, das einen nimmt, wird
  übersprungen (`sudo -u root`, `xargs -n 1`, `nice -n 10`, `ionice -c 3`,
  `stdbuf -o 0`, `timeout -s KILL`, `exec -a NAME`, GNU `time -o DATEI`,
  `unbuffer -ignore HUP`), auch wenn das Flag das letzte eines Bündels ist
  (`sudo -Hu root`) oder eine auf einen Anfang gekürzte Langoption
  (`timeout --sig KILL`), und eine Umleitung zwischen einem Flag und seinem
  Wert ist kein Wert. `sudo run` (Sudo für Windows) wird wie `sudo`
  gelesen. `Start-Process`, `start` oder
  `saps` wird verweigert, wenn loomux eines seiner Argumente ist, auch als
  Wert eines Parameters mit Doppelpunkt (`-FilePath:loomux.exe`), gleich
  welche die übrigen sind. Jeder Abschnitt der Zeile zählt
  (`;`, `|`, `&`, `&&`, `||`, Zeilenumbruch, `(`, `)`, `$(`, ein Backtick),
  und eine Zeilenfortsetzung (`\` oder ein Backtick am Zeilenende) wird
  vorher zusammengefügt; ein Backtick-Escape in einem Wort
  (``loomux con`fig``) wird gelesen, wie PowerShell ihn liest.
  - **In Zeichenketten gelesen** — ein Befehl, den eine Shell aus einer
    Zeichenkette ausführt (`sh -c "loomux init"`, `pwsh -c …`, `pwsh
    -EncodedCommand …`, `iex '…'`, `eval`, `env -S`) oder aus einer Pipe
    oder einem Here-String liest (`echo "…" | sh`, `'…' | iex`, `bash <<<
    '…'`), wird als eigene Zeile geprüft, bis drei Shells tief (tiefer wird
    verweigert), und dort befreit kein Flag. Eine Variable oder ein Alias,
    den die Zeile selbst setzt, wird eingesetzt (`M=loomux; $M init`, `alias
    l=loomux; l init`).
  - **Bekannte Lücken** — die Regel liest Wörter, keine Shell, und lässt
    darum durch: einen Alias oder ein Programm in einer Variablen, die
    anderswo als in der Zeile gesetzt sind; eine Funktion, die die Zeile
    definiert; einen Befehl in `script -c …`; ein Programm, das ein
    bekanntes Werkzeug startet (`uv run loomux init`, `npx loomux init`,
    `find … -exec loomux …`); `xargs loomux` mit seinen Argumenten von stdin;
    `go run .` in `cmd/loomux`; und, nach einem früheren maskierten `\"` oder `\'` auf
    derselben Zeile, einen Programmpfad in Anführungszeichen, dessen Teil
    hinter seinem letzten Trennzeichen (`(`, `)`, `&`, `;`, `|`) ein
    Leerzeichen enthält, etwa
    `echo "a \" b"; "C:\Program Files (x86)\My Tools\loomux.exe" init`.
  - **Bekannte Fehlverweigerungen** — im Zweifel verweigert sie: Hinter
    einem Programm, das sie nicht kennt, zählt ein späteres loomux-Wort als
    Aufruf, gleich was das Programm damit tut (`ssh host loomux init`,
    `gdb --args loomux init`, `zip -r loomux.zip loomux config`; ein Programm,
    das nur einen Namen nachschlägt oder sein Handbuch zeigt, `man`, `tldr`,
    `which`, ist ausgenommen);
    `echo "x; loomux init"`, `start loomux config list`,
    `Start-Process code -ArgumentList loomux`, `command -v loomux init` (das
    den Namen nur nachschlägt), ein loomux-Wort direkt hinter einer Klammer,
    die keinen Block öffnet (`awk '{ print }' loomux init`,
    `echo } loomux config set a b`, `echo ${X} loomux init`), `loomux init \`
    mit `--dry-run` auf der nächsten Zeile (PowerShell führte die erste Zeile
    allein aus) und `config` mit einem anderen Flag als `--root <verz>`,
    `--root=<verz>` oder `--global` vor dem Unterbefehl
    (`loomux config --json list`) sowie `convert` oder `fetch` mit der Hilfe
    in jeder Form außer einem alleinstehenden `--help` oder `-h`
    (`convert -help`, `convert --help=true`, `convert --help x`,
    `fetch --scope x --help`), die nur die Hilfe ausgeben. Ebenso eine Zeile,
    die solchen Text nur als Daten trägt, etwa ein Heredoc mit
    `loomux config set …`.
- **Standard-Output / Fehler**:
  - Bei Ablehnung: JSON-Ablehnungs-Umschlag auf `stdout`, Begründung auf `stderr`.
- **Exit-Codes**:
  - `0`: Gestattet.
  - `2`: Verweigert (Policy-Verletzung oder Schreibzugriff außerhalb registrierter Bereiche).

### `loomux hook post-tool-use`
Wird ausgeführt, nachdem ein Agent eine Datei bearbeitet hat.

- **Standard-Input (stdin)**: Name des Werkzeugs und Eingabe-Payload; der bearbeitete Pfad kommt aus `file_path`, sonst aus `notebook_path`.
- **Flags**: `--host <h>` (Pflicht), `--root <r>`, `--budget <dauer>` — wie lange die Lanes zusammen dauern dürfen (Go-Dauer, Vorgabe `50s`, unter der Hook-Frist des Hosts von 60 s). Jeder Befehl bekommt das Kleinere aus seinem eigenen `timeout` und dem Rest des Budgets.
- **Verhalten**: Fährt die Lanes des Profils `edit` (vorgegeben `lint` und `types`) für den Stack der bearbeiteten Datei, in dem Bereich, der die Datei enthält, so wie [`[verify]`](configuration.md#verify-prüfketten--quality-gates) und die Presets sie auslegen, mit `on_file`, wo eine Lane es hat; siehe [Hooks](hooks.md#5-die-post-edit-lanes-je-sprachstack).
- **Übersprungene Lanes**: Eine Lane, deren Werkzeug nicht auf dem `PATH` liegt, ein noch nicht importiertes Godot-Projekt und jede Lane, die das Budget nicht mehr erreicht, werden übersprungen, nicht rot. Jede steht auf `stderr`, und bei Exit 0 auf `stdout` in der Form des Hosts, für Claude Code als `{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"loomux hook post-tool-use: lane skipped, the edit budget ran out: lint/go"}}`, `<`, `>` und `&` unverändert. Eine Datei eines Aufrufs, die das Budget nicht mehr erreicht, steht dort ebenso. Bei jedem anderen Exit-Code schreibt der Hook nichts auf `stdout`. Unter `--host antigravity` wird dieses `stdout` nicht weitergegeben, denn ob agy den Kontext eines PostToolUse liest, ist ungemessen: Dort erreicht ein Skip das Modell nur bei Exit 2, auf `stderr`, und die Aufrufer des Blast-Monitors gar nicht.
- **Blast-Monitor**: Nach einem Edit an einer `.go`-Datei, wenn keine Lane des Aufrufs rot ist, folgen den übersprungenen Lanes im selben `additionalContext` die direkten Aufrufer in anderen Dateien jedes Symbols, das der Edit gegenüber dem Graphen auf der Platte geändert oder entfernt hat. Ohne Graph schweigt er, und ein Befund ist er nie; siehe [Hooks](hooks.md#der-blast-monitor).
- **Exit-Codes**: `0` (alle Lanes grün, übersprungen oder nichts zu fahren), `1` (fehlerhafter Aufruf, etwa ein fehlendes oder unbekanntes `--host`, ein `[verify]`, das sich nicht laden lässt, oder `--host codex`, sobald der Aufruf eine Datei nennt: Codex hat noch keinen Adapter, der Hook verweigert also, statt in der Form eines anderen Hosts zu antworten), `2` (eine scharfe Lane ist gescheitert, abgelaufen oder blockiert; ihre Ausgabe auf `stderr`). Eine rote Lane in Probe endet mit Exit 0, und ihr Befund geht, mit `(probation)` gekennzeichnet, wie eine übersprungene Lane in den Kontext des Wirts.

### `loomux hook session-start`
Hält den Commit fest, auf dem die Sitzung beginnt, und meldet die Flow-Läufe, die auf einen Menschen warten.

- **Flags**: `--host <h>` (Pflichtfeld; `claude` und `antigravity` haben Adapter), `--root <r>`.
- **Verhalten**:
  - Schreibt `HEAD` als `base` in `.loomux/state/hooks/<session_id>.json`.
  - Belebt eine Sitzung wieder, die `worktree unlink` als beendet markiert hat: entfernt `<session_id>.ended` und schreibt die Datei mit zurückgesetzter Blockreihe zurück, sodass die Sitzung wieder zählt; bei einer nie als beendet markierten Sitzung wird die Datei nur verjüngt. Eine Marke, die sich nicht entfernen lässt, oder eine Datei, die sich nicht zurückschreiben lässt, steht im Kontext, mit Exit 0.
  - Warnt in `hookSpecificOutput.additionalContext`, wenn das Binary im Projekt älter ist als seine Go-Quellen, `go.mod`, `go.sum` oder eine Datei unter `flows/`, deren Pfad kein Glied mit `_` oder `.` vorne hat (`_test/` eines Flows zählt nicht).
  - Meldet jeden Lauf unter `.loomux/state/runs/`, der an einem Tor wartet, bei jedem Start, auch bei einem wiederholten unter Antigravity: `run <id> (<flow>, <herkunft>) is waiting at <tor>: <frage>`, dann `a human answers it with: <binary> flow resume <id> --answer "your answer"`; `<binary>` ist der Pfad, aus dem der Hook läuft, mit Schrägstrichen, in doppelten Anführungszeichen, wenn er Leerraum oder ein anderes Zeichen enthält, das eine Shell liest. Einen Laufordner, der sich nicht auflisten lässt, nennt er auf `stderr` (`.loomux/state/runs cannot be read as a folder of runs: …`), ebenso ein Journal oder eine Marke, die sich nicht lesen lässt; sie verbirgt nur ihren eigenen Lauf; sagt die Marke, dass eine andere loomux-Version den Lauf schrieb, nennt die Zeile beide (`run 0001 was written by loomux 0.0.0-dev, this is …`). Siehe [Flows](flows.md#6-tore-gehören-einem-menschen).
  - Nennt jeden Eintrag unter `.loomux/flows/`, der den Namen eines mitgelieferten Flows trägt, solange `[flow] overrides` ihn nicht nennt: `.loomux/flows/<name> is ignored: [flow] overrides does not name it`. Ein `.loomux/flows`, das sich nicht auflisten lässt, bekommt `.loomux/flows cannot be read as a folder of flows: …`, mit den Worten, mit denen `loomux flow` warnt.
  - Liest außerdem `<Zustandsverzeichnis>/update.json` und warnt, wenn unter Windows ein Durchlauf von `serve` ein anderes Binary als `<Zustandsverzeichnis>/bin/loomux.exe` als sein eigenes verzeichnet hat, oder wenn der letzte Self-Update-Durchlauf gescheitert ist, gleich wer ihn fuhr.
  - Nennt beim ersten Start einer Sitzung die Lanes in Probe aus `.loomux/armed.toml` (je eine Kontextzeile) und den Bericht, den der Stop-Hook zuletzt für diesen `HEAD` unter den jetzt scharfen Lanes gemerkt hat, auf 40 Zeilen gekürzt. Eine Datei, die sich nicht lesen lässt, ist eine Zeile `loomux: <fehler>`.
  - Legt keine Worktree-Junctions an; das tut `loomux worktree link`. Siehe [Hooks](hooks.md#8-sitzungshooks).
- **Exit-Codes**: `0` (Erfolg), `1` (fehlender oder unbekannter Host, kein Adapter für den Host, unlesbare Nutzlast, gescheitertes Schreiben).

### `loomux hook stop`
Das Tor am Rundenende: stellt zu, was Subagenten hinterlassen haben, und fährt dann das Profil `stop` über das, was sich seit dem letzten grünen Lauf geändert hat.

- **Flags**: `--host <h>` (Pflicht; `claude` und `antigravity` haben Adapter), `--root <r>`, `--budget <dauer>` — wie lange die Lanes zusammen dauern dürfen (Go-Dauer, Vorgabe `270s`, unter den 300 s, die sein Settings-Eintrag gewährt). Jeder Befehl bekommt das Kleinere aus seinem eigenen `timeout` und dem Rest des Budgets.
- **Standard-Input (stdin)**: die `Stop`-Nutzlast des Hosts; gelesen wird nur `session_id`.
- **Verhalten**: in dieser Reihenfolge — die Befunde der Subagenten auf `stderr`, der Blockzähler (nach 3 Blockaden in Folge gibt er für eine Runde auf und lässt die Befunde für die nächste liegen), der Marker `.loomux/no-verify` (er überspringt die Kette, nicht die Befunde), die Konfiguration und ihr Profil `stop` (ein `[verify]`, das sich nicht lesen lässt, beendet das Tor mit Exit 1, der nichts anhält), der Fingerabdruck des Inhalts (nichts Neues seit dem letzten grünen Lauf oder der Basis: kein Werkzeug startet; mit einer Graph-Lane nur unter demselben `HEAD`), dann die Arten des Profils `stop` (vorgegeben `lint`, `types`, `test`, `coverage`, `graph`; die Lane `graph` liest eine Kopie des Index, die den ganzen Baum trägt, und urteilt über die Runde gegen `HEAD`) im Check-Scope, dazu `lint/wiki`, wo `lint` angefragt ist und es ein Wiki gibt. Ein grüner Lauf rückt `base` auf `HEAD` vor und merkt sich den Baum. Ein Lauf, der nur in Lanes in Probe rot ist, lässt `base` stehen und merkt sich seinen Baum, `HEAD`, die scharfen Lanes und seinen Bericht; ein späteres Rundenende über denselben Baum, unter demselben `HEAD` und mit denselben scharfen Lanes startet kein Werkzeug und sagt diesen Bericht noch einmal. Siehe [Hooks](hooks.md#stop).
- **Standard-Fehler (stderr)**: zugestellte Befunde als `subagent <agent_id>: <zeile>`, danach nur die roten Lanes, im Format von `loomux check`. Bei Exit 0 mit einer roten Lane in Probe der Bericht dieser Lanes, der mit der Zeile `probation: <schlüssel>` endet; `stderr` zeigt bei Exit 0 kein Host, darum gibt ihn die nächste Sitzungseröffnung weiter.
- **Exit-Codes**: `0` (die Runde endet: grün, nur in Lanes in Probe rot, nichts Neues, der Marker, oder der Zähler hat aufgegeben), `2` (die Runde wird angehalten: eine scharfe rote Lane, ein Git-Fehler, oder zugestellte Befunde — mit Befunden wird selbst ein Exit 1 zu 2), `1` (das Tor konnte nicht urteilen: eine unlesbare Nutzlast oder eine ohne `session_id`, das Budget war aufgebraucht, eine angefragte Art hatte nichts, was lief, `[verify]` lässt sich nicht laden — die Konfiguration wird vor dem Baum gelesen, daher endet das mit 1, auch wenn der Baum schon als grün bekannt war —, der Plan scheitert, das Coverage-Verzeichnis lässt sich nicht vorbereiten (`verify.PrepareCover`), oder ein fehlerhafter Aufruf; die Runde endet).

### `loomux hook subagent-start`
Hält fest, wo `origin`, die lokalen Branches und `HEAD` stehen, bevor ein Subagent läuft.

- **Flags**: `--host <h>` (Pflicht; `claude` und `antigravity` haben Adapter), `--root <r>`.
- **Standard-Input (stdin)**: die `SubagentStart`-Nutzlast des Hosts; gelesen werden `session_id` und `agent_id`.
- **Verhalten**: `git ls-remote origin` (Frist 10 s, `GIT_TERMINAL_PROMPT=0`), die lokalen Branches und `HEAD` kommen nach `.loomux/state/hooks/<session_id>/agents/<agent_id>.json`; ein Remote, der nicht antwortet, wird als `unavailable` festgehalten. Ein Befund, der noch in dieser Datei geparkt ist, bleibt erhalten. Siehe [Hooks](hooks.md#subagent-start-und-subagent-stop).
- **Exit-Codes**: `0` (geschrieben), `1` (fehlender oder unbekannter Host, keine `session_id` oder `agent_id`, unlesbare Nutzlast, gescheitertes Schreiben). Nie 2.

### `loomux hook subagent-stop`
Vergleicht mit dem Schnappschuss und parkt, was sich bewegt hat, für das `stop` des Hauptagenten.

- **Flags**: `--host <h>` (Pflicht; `claude` und `antigravity` haben Adapter), `--root <r>`.
- **Standard-Input (stdin)**: die `SubagentStop`-Nutzlast des Hosts; gelesen werden `session_id` und `agent_id`.
- **Verhalten**: eine Zeile je Ref von `origin` oder lokalem Branch, der neu, weg oder bewegt ist, und ein `new commit <oneline>` je Commit, den `HEAD` und die bewegten Branches gewonnen haben, angehängt an die Datei des Agenten; ohne Befund wird die Datei entfernt. Ohne Datei oder ohne Schnappschuss schweigt er. Für das Modell schreibt er nichts: seine eigene Ausgabe erreichte den Subagenten, nicht den Hauptagenten.
- **Exit-Codes**: `0` (verglichen, oder nichts zu vergleichen), `1` (fehlender oder unbekannter Host, keine `session_id` oder `agent_id`, unlesbare Nutzlast, gescheitertes Schreiben). Nie 2.

---

## 4. Diagnose-Doktor (`loomux status`)

### `loomux status [--root <pfad>]`
Inspiziert den Zustand des aktuellen Repositories, der Prüfketten, erkannten Agenten und Hooks.

```bash
loomux status
```
- **Aliase**: `loomux explain`, `loomux doctor`.
- **Ausgabe-Details**:
  - Pfad der Projektwurzel, erkannte Stacks und ob das Wiki-Bündel aktiv ist (mit seinem Verzeichnis).
  - Die Lanes, die der post-edit-Hook je aktivem Stack fährt: das Profil `edit`, wie `[verify]` und die Presets es auslegen, jede mit ihrer Herkunft, und welche ihrer Werkzeuge auf dem `PATH` fehlen.
  - Den einzutragenden `Stop`-Eintrag (`loomux hook stop`, Profil `stop`, das Wiki-Bündel als `lint/wiki`), und für jedes der sechs Ereignisse `PreToolUse`, `PostToolUse`, `SessionStart`, `Stop`, `SubagentStart` und `SubagentStop`, ob `.claude/settings.json` seinen `loomux hook` ruft (`[OK]`) oder nicht (`[INFO]`). Ein Eintrag eines anderen Werkzeugs in dieser Datei wird nicht gemeldet.
  - Einen Abschnitt „Lane Probation“, wenn das Projekt `.loomux/armed.toml` hat: jede Lane als `[ARMED]` oder `[PROBATION]`, einen Eintrag, dem keine Lane antwortet, als `[ORPHAN]`, ein `[WARN]` für einen pre-commit-Hook, der keine Lanes scharf stellt, und eines, wenn es gar keinen pre-commit-Hook gibt (beide mit dem Hinweis, dort `loomux check precommit --arm` zu rufen oder von Hand mit `loomux gate arm` scharf zu stellen), und ein `[WARN]`, wenn git die Datei ignoriert. Eine Datei, die sich nicht lesen lässt, ist ein `[WARN]` mit ihrem Grund. Ohne die Datei fehlt der Abschnitt.
- **Exit-Codes**: `0` (Bericht ausgegeben, auch wenn er einen Konfigurationsfehler nennt), `1` (unbekanntes Flag).

---

## 5. Worktree-Spiegelung (`loomux worktree`)

Stellt die in `[worktree] mirror` genannten Verzeichnisse als Windows-Junctions in verknüpfte Git-Worktrees und nimmt sie wieder heraus. Junctions gibt es nur unter Windows. Der vollständige Entscheidungsweg steht unter [Hooks](hooks.md#9-worktree-spiegelung).

### `loomux worktree link [--root <pfad>]`
Legt in einem verknüpften Worktree für jeden konfigurierten Pfad, der dort fehlt, eine Junction in den Haupt-Checkout an; räumt danach, wo immer es läuft, unsere Junctions aus Verzeichnissen unter `.worktrees/` und `.claude/worktrees/`, die Git nicht mehr hält.

### `loomux worktree unlink [--root <pfad>]`
Liest `session_id` aus der Nutzlast auf `stdin`, markiert diese Sitzung mit einer Datei `<id>.ended` neben ihrem Stand unter `.loomux/state/hooks/` als beendet (der Stand bleibt für eine Fortsetzung unter derselben ID; `hook session-start` nimmt die Marke wieder weg) und entfernt die Junctions nur, wenn keine andere Sitzung übrig ist, die jünger als 24 Stunden und nicht als beendet markiert ist.

### `loomux worktree remove <worktree-pfad>`
Lehnt den Haupt-Checkout und jedes Verzeichnis ab, an dem Git keinen Worktree hält, entfernt die Junctions, fährt `git worktree remove --force`, prüft, ob das Verzeichnis weg ist, und gibt `removed <pfad>` aus.

- **Exit-Codes**: `0` (in Ordnung oder nichts zu tun), `1` (ein Fehler, auf `stderr` benannt), `2` (kein oder ein unbekannter Unterbefehl).

---

## 6. Code-Graph-Engine (`loomux graph`)

> [!NOTE]
> **`build`, `check`, `ask`, `callers`, `skeleton`, `grep`, `map`, `stats` und `blast` sind verdrahtet; `viz` bleibt spezifiziert.** Stufe G1 hat die Pakete gebaut, auf denen der Graph aufsetzt — `internal/code/model`, `internal/code/pagerank` und `internal/code/blast` —, Stufe G2a ergänzt Extraktor, Wiring-Schreiber, Frischesonde und die Befehle `build` und `check`, Stufe G2b ergänzt Lexik, lexikalisches Scoring, Personalized-PageRank-Verschmelzung und `graph ask`, Stufe G3 stellt `ask` und `check` hinter die MCP-Werkzeuge `graph_find_code` und `graph_check_freshness` (§8), und Stufe G4a lieferte die Navigationspalette (`callers`, `skeleton`, `grep`, `map`, `stats`) und ihre vier MCP-Werkzeuge; Stufe G4b ergänzte `blast`, das MCP-Werkzeug `graph_blast`, die Prüfbefehle `graph-fresh` und `blast-audit` (§2) und den Blast-Monitor im Post-Edit-Hook (§3). Stufe G5a (✅ 2026-09-26) ergänzte die Python-Extraktion auf `gotreesitter`, einer Tree-sitter-Laufzeit in reinem Go, und den Extraktions-Cache hinter `build`.

### `loomux graph build [--root <pfad>] [--no-reuse]`
Liest und hasht jede Quelldatei, die `internal/code/sourceset` unterhalb der Wurzel findet — `.go` und `.py`, genau und mit Groß- und Kleinschreibung verglichen, `.pyi`-Stubs ausgenommen —, extrahiert jede mit dem Extraktor ihrer Sprache (Go mit `go/parser`, Python auf `gotreesitter`), löst sie zum deterministischen AST-Graphen auf und schreibt ihn nach `.loomux/state/graph/wiring.json`. Jede Sprache löst in einem eigenen Namensindex auf, keine Kante überquert eine Sprachgrenze. Dabei schreibt er auch die Frischeakte (`.loomux/state/graph/cache/fingerprint.json`), die eine spätere Sonde liest, und den Extraktions-Cache (`.loomux/state/graph/cache/extract.json`), den der nächste Build liest; scheitert das Schreiben einer der beiden, meldet der Befehl das auf `stderr`, ohne den Bau selbst scheitern zu lassen — der Graph auf der Platte ist bereits korrekt.

- **Flags**: `--root <pfad>` — Projektwurzel; ohne Angabe das Arbeitsverzeichnis. `--no-reuse` — jede Datei parsen, gleich was der Extraktions-Cache hält; der Ausweg, wenn sich ein Extraktor ohne seine Version geändert hat.
- **Ausgabe**: eine Zeile mit Dateien, Knoten und Kanten je Relation (`extends`, die Basisklassen einer Python-Klasse, nur wenn es welche gibt, die Zeile eines Go-Graphen hat also keine), dann eine Zeile mit unaufgelösten Importzielen, Dateien ohne Symbol und der benötigten Zeit, dann je Sprache eine Zeile, was der Build mit ihren Dateien getan hat, und für eine Sprache mit Parsefehlern eine Zeile mit den ersten fünf dieser Dateien (danach `(+N more)`). Illustrative Form, kein zu erwartender Wert — jeder Teil davon, auch die Zeit, bewegt sich mit dem eigenen Code dieses Repositories, und die letzten drei Commits haben hier jeweils eine Zahl geschrieben, die der nächste Commit widerlegt hat: `N files, N nodes, N edges (N contains, N calls, N imports[, N extends])` / `N unresolved import targets, N files without a symbol, Nms` / `  <lang>: N files, P parsed, R reused, E parse errors` / `  <lang> parse errors in: a.py, b.py`. Gemessene Zahlen mit Befehl und Rohausgabe stehen in `docs/de/benchmarks.md`.
- **Der Extraktions-Cache** hält je Datei die Version der Sprache, den Hash der Bytes, die Extraktion und die Rumpftexte. Eine Datei, deren Hash und Sprachversion zu ihrem Eintrag passen, wird nicht neu geparst; der Eintrag einer Datei, die es nicht mehr gibt, fällt mit dem nächsten Build heraus. Ein fehlender Cache ist ein kalter Start und meldet nichts. Ein unlesbarer Cache oder einer mit anderer Formatversion lässt jede Datei parsen und sagt das auf `stderr` (`loomux graph build: extract cache ignored, parsing every file: …`); ein Cache, der nicht geschrieben werden kann, meldet `extract cache not written: …`. Keins von beiden ändert den Exit-Code.
- **Parsefehler**: eine Python-Datei mit Syntaxfehlern bleibt mit ihrem Dateiknoten im Graphen; eine Definition, die der Parser um abgelehnten Text herum neu bauen musste, bekommt keinen Knoten, und die Fehler zählt der Bericht: jeden ERROR- und MISSING-Knoten des Baums und einen mehr für ein Parsen, das vorzeitig anhielt. Die Zahl ist eine Untergrenze — ein wiederhergestellter Baum kann Text verlieren, ohne dass ein Knoten davon zeugt —, einer Datei ohne gezählten Parsefehler kann also trotzdem eine Definition fehlen. Eine Go-Datei, die nicht parst, lässt den Build weiter scheitern.
- **Exit-Codes**: `0` bei Erfolg, auch wenn Python-Dateien Parsefehler haben; `1`, wenn die Wurzel nicht auflösbar ist, eine Datei nicht gelesen werden kann, eine Go-Datei nicht parst, die Modulauflösung scheitert oder der Graph nicht geschrieben werden kann; `2` bei einem Aufruffehler.
- **Kosten**: `build` liest die Frischeakte nie — es liest und hasht jede Datei, kalt wie warm, jedes Mal. Der Extraktions-Cache erspart einer Datei nur das Parsen, nie das Lesen: ein Eintrag gilt nur gegen den Hash der eben gelesenen Bytes. Es gibt dabei nichts zu überspringen: anders als `check` erzeugt `build` gerade den Stand, gegen den eine Sonde später vergleicht, und ein veraltetes Byte darin wäre eine veraltete Antwort, keine ersparte Lesung. Gemessene Zahlen stehen in `docs/de/benchmarks.md`.

### `loomux graph ask "<anfrage>" [flags]`
Sucht Code-Symbole gerankt nach BM25-artigem lexikalischen Matching verschmolzen mit **Personalized PageRank** über den AST-Aufrufgraph.

- **Flags**:
  - `--root <pfad>` — Projektwurzel; ohne Angabe das Arbeitsverzeichnis.
  - `--limit <n>` — Maximale Anzahl auszugebender Treffer (Standard `8`).
  - `--in <präfix>` — Filtert Kandidaten vor Scoring und PageRank-Lauf nach Pfadpräfix ein; berechnet Dokumenthäufigkeiten über den Rest neu.
  - `--source` — Blendet den Quellcode-Span für jeden Treffer inline ein (gedeckelt auf 80 Zeilen, außer bei `--full`). Ohne `--source` werden nur Fundorte und Signaturen ausgegeben.
  - `--full` — Hebt bei `--source` die 80-Zeilen-Deckelung auf und blendet den vollen Span ein.
  - `--json` — Gibt maschinenlesbares JSON gemäß der `ask.Answer`-Struktur aus (`hits`, `query`, `note`, `stats`).
  - `--no-refresh` — Überspringt die Frischeprüfung und den automatischen Hintergrund-Neubau bei Abweichung.
- **Die beiden Dinge, die sonst zweimal gefragt werden**:
  - Ohne `--source` kommt kein Quelltext — nur Fundort (Pfad, Zeilenspan), Symbol-ID, Signatur sowie der Gesamtwert mit seinen lexikalischen und graphischen Komponenten.
  - Standardmäßig prüft `ask` vor der Antwort die Frische des Graphen. Ist der Graph veraltet, fehlt seine Frischeakte, oder fehlt die Beiakte beziehungsweise trägt sie eine andere Indexversion, baut `ask` Graph und Beiakte unter einem prozessübergreifenden Lock neu, bevor geantwortet wird (Statusmeldungen auf `stderr`). Die Beiakte gehört zur Prüfung, weil die Frischeakte nur Quelldateien kennt: ohne diese Prüfung würde eine gelöschte `ask-index.json` jede spätere Frage auf Namen und Pfade zurückwerfen, bis sich zufällig eine Quelldatei ändert. Um den bestehenden Stand ohne Neubau abzufragen, dient `--no-refresh` — das überspringt auch die Beiaktenprüfung, sodass die Antwort auf Namen und Pfade zurückfallen kann und das auf `stderr` sagt.
  - Ohne Graph endet `ask` mit Exit 1 und verweist auf `loomux graph build`; eine Abfrage baut nie einen ersten Graphen.
- **Ausgabe**: Rangliste der Treffer im Format:
  `N. <id>  <pfad>:<span-oder-zeile>  (<score> lex <lexical> graph <graph>)`
  gefolgt von der Signatur und bei `--source` dem mit `|` eingerückten Quelltextblock. Passt kein Symbol zur Anfrage, wird ein Hinweis ausgegeben und mit Code 0 beendet.
- **Exit-Codes**: `0` bei Erfolg (auch wenn keine Symbole matchen; ein `--limit` von null oder darunter fällt auf `8` zurück); `1` bei Fehlern (noch kein Graph, unlesbarer Graph, fehlerhafter Neubau, Syntaxfehler in einer Datei beim Neubau); `2` bei Aufruffehlern (keine oder mehr als eine Anfrage, unbekanntes Flag).

### `loomux graph callers <symbol> [--root <pfad>] [--direction in|out] [-d <tiefe>] [--in <präfix>] [--no-refresh] [--json]`
Zeigt, wer ein Symbol aufruft, importiert oder referenziert (`--direction in`, Standard), oder was das Symbol selbst aufruft (`--direction out`).

- **Flags**:
  - `--root <pfad>`: Projektwurzel; ohne Angabe das Arbeitsverzeichnis.
  - `--no-refresh`: Antwortet aus dem Graphen auf der Platte, baut nie neu.
  - `--direction <in|out>`: Verfolgt eingehende Aufrufer (`in`) oder ausgehende Aufrufe (`out`).
  - `-d`, `--depth <tiefe>`: Transitive Tiefe (Standard `1`; `-d all` oder `-d full` für die vollständige transitive Hülle).
  - `--in <präfix>`: Filtert Symbole vor der Auflösung nach Pfadpräfix.
  - Jeder direkte Treffer (Tiefe 1) trägt die erste Zeile im Span des Aufrufers, die den Aufgerufenen nennt. Bei `--direction out` liegt diese Zeile in der Datei des Startsymbols und wird mit ihrem Pfad ausgegeben.
  - `--json`: Gibt maschinenlesbares JSON aus (`query.CallersAnswer`).
- **Exit-Codes**: `0` bei Erfolg; `1` bei fehlendem/unlesbarem Graph oder unbekanntem Symbol; `2` bei Aufruffehlern.

### `loomux graph skeleton <datei> [--root <pfad>] [--no-refresh] [--json]`
Gibt alle Funktions-, Typ-, Interface- und Methodensignaturen sowie Zeilenspannen einer Datei ohne Rümpfe aus (~10x Token-Ersparnis).

- **Flags**:
  - `--root <pfad>`: Projektwurzel; ohne Angabe das Arbeitsverzeichnis.
  - `--no-refresh`: Antwortet aus dem Graphen auf der Platte, baut nie neu.
  - `--json`: Gibt maschinenlesbares JSON aus (`skeleton.FileSkeleton`).
- **Exit-Codes**: `0` bei Erfolg; `1` bei fehlendem Graph oder Datei nicht im Graph; `2` bei Aufruffehlern.

### `loomux graph grep <muster> [--root <pfad>] [-i] [--fixed] [--in <präfix>] [--max-hits <n>] [--no-refresh] [--json]`
Regex-Suche über indizierte Dateien, gruppiert nach umschließendem Symbol und sortiert nach Kopplungsgrad (`inDegree`).

- **Flags**:
  - `--root <pfad>`: Projektwurzel; ohne Angabe das Arbeitsverzeichnis.
  - `--no-refresh`: Antwortet aus dem Graphen auf der Platte, baut nie neu.
  - `-i`, `--ignore-case`: Regex-Suche ohne Beachtung von Groß-/Kleinschreibung.
  - `--fixed`: Behandelt das Muster als reinen Text (ohne Regex-Syntax).
  - `--in <präfix>`: Begrenzt die Suche auf Dateien unterhalb des Pfadpräfix.
  - `--max-hits <n>`: Maximale Anzahl von Zeilentreffern (Standard `300`); weitere Treffer werden nur gezählt.
  - `--json`: Gibt maschinenlesbares JSON aus (`grep.Result`).
- **Exit-Codes**: `0` bei Erfolg (auch bei 0 Treffern); `1` bei fehlendem/unlesbarem Graph oder ungültigem Regex; `2` bei Aufruffehlern.

### `loomux graph map [--root <pfad>] [--max-dirs <n>] [--hubs-per-dir <n>] [--hotspots <n>] [--no-refresh] [--json]`
Zeigt token-budgetierte Verzeichnis-Cluster, lokale Hubs und globale Codebasis-Hotspots gerankt nach Kanten-Kopplung.

- **Flags**:
  - `--root <pfad>`: Projektwurzel; ohne Angabe das Arbeitsverzeichnis.
  - `--no-refresh`: Antwortet aus dem Graphen auf der Platte, baut nie neu.
  - `--max-dirs <n>`: Maximale Anzahl von Verzeichnis-Clustern (Standard `16`).
  - `--hubs-per-dir <n>`: Höchstzahl der Hubs je Verzeichnis (Standard `3`).
  - `--hotspots <n>`: Höchstzahl der Hotspots im ganzen Repository (Standard `12`).
  - `--json`: Gibt maschinenlesbares JSON aus (`repomap.RepoMap`).
- **Exit-Codes**: `0` bei Erfolg; `1` bei fehlendem oder unlesbarem Graph; `2` bei Aufruffehlern.

### `loomux graph stats [--root <pfad>] [--json]`
Gibt strukturelle Codebasis-Kennzahlen aus `.loomux/state/graph/wiring.json` aus: Knotenzahl, Kantenzahl gruppiert nach Relation, indizierte Dateien, Sprachen und Dateigröße.

- **Flags**:
  - `--root <pfad>`: Projektwurzel; ohne Angabe das Arbeitsverzeichnis.
  - `--json`: Gibt maschinenlesbares JSON aus (`query.StatsAnswer`).
- **Exit-Codes**: `0` bei Erfolg; `1` bei fehlendem oder unlesbarem Graph; `2` bei Aufruffehlern.

### `loomux graph check [--root <pfad>] [--json]`
Extrahiert den ganzen Baum neu und vergleicht ihn, Knoten für Knoten, mit dem auf der Platte geschriebenen Graphen. Es liest den Extraktions-Cache wie `build`, sodass eine Datei, deren Hash und Sprachversion zu ihrem Eintrag passen, nicht neu geparst wird, schreibt ihn aber nie; ein unlesbarer Cache kostet es nur Parsezeit und bleibt still.

- **Flags**: `--root <pfad>` — Projektwurzel; ohne Angabe das Arbeitsverzeichnis. `--json` — schreibt die Abweichung als JSON (`checkResult`: `ok`, `missing`, `foreign`, `added`, `removed`, `changed`) statt des menschenlesbaren Berichts.
- **Das eine, was sonst zweimal gefragt wird**: `check` liest die Frischeakte nicht. Die Akte beantwortet „soll eine Anfrage sich die Mühe eines Neubaus machen"; `check` beantwortet „beschreibt der Graph den Code noch", und die einzig ehrliche Antwort darauf ist, neu zu extrahieren und Rumpf-Hashes zu vergleichen. Ein `touch`, das die Änderungszeit einer Datei ändert, aber keine Bytes, ist deshalb kein Befund — hier wie bei der Sonde, aber aus einem anderen Grund: die Sonde kommt gar nicht erst über ihren Stat-Vergleich hinaus, `check` kommt bis zum Hash und findet ihn unverändert.
- **Ausgabe**: `NO GRAPH`, wenn noch nichts gebaut wurde; `FOREIGN GRAPH`, wenn der Graph auf der Platte eine andere Extraktor-Version nennt als dieses Binary; `OK`, wenn nichts abgewichen ist; sonst `DRIFT` mit je einer Zeile pro hinzugefügter, entfernter oder geänderter Knoten-ID.
- **Exit-Codes**: `0` — frisch (`OK`); `1` — noch kein Graph, ein fremder Graph, gefundene Abweichung, oder ein Fehler bei der Neuextraktion; `2` — Aufruffehler.

### `loomux graph blast [--root <pfad>] [--cached | --base <ref>] [-d N|all] [--no-refresh] [--json]`
Zeigt, was eine Änderung erreicht, aus dem Git-Diff: je geänderter Datei die Symbole, die ihre Hunks berühren, was sie erreicht und ob sich ein Test, der sie erreicht, mitgeändert hat. Gelöschte Dateien schließen den Bericht ab.

- **Flags**:
  - `--root <pfad>`: Projektwurzel; das Arbeitsverzeichnis, wenn leer.
  - `--cached`: vergleicht den Index mit `HEAD`.
  - `--base <ref>`: vergleicht `<ref>...HEAD` statt des Working Tree. Nicht zusammen mit `--cached`.
  - `-d`, `--depth`: Tiefengrenze, eine positive ganze Zahl (Vorgabe `1`), oder `all`/`full` für die Hülle.
  - `--no-refresh`: antwortet aus dem Graphen auf der Platte, baut nie neu.
  - `--json`: schreibt die Antwort als JSON.
- **Bereich**: ohne `--cached` und `--base` der Arbeitsbaum gegen `HEAD`, bei sauberem Baum `HEAD~1...HEAD`. git läuft in der Projektwurzel mit `--relative`, die Pfade sind also auch in einem Unterverzeichnis-Bereich die des Graphen.
- **Ausgabe**: `blast radius: <bereich>`, dann je geänderter Datei `<pfad> [<signal>]` mit ihren Seeds (`seed <name> (<art>) <span> in-degree <n>`) und den Tests, die sie erreichen; `reached:` mit Tiefe, Relation und den geänderten Dateien jedes Treffers; `not indexed:` Dateien, die der Graph nicht kennt; `evidence:` die Diff-Zeilen in höchstens vier Seeds, je sechs Zeilen; zuletzt `deleted:`. Das Signal ist `changed` (eine Testdatei, die den Bereich erreicht, ist auch im Diff), `stale` (Tests erreichen ihn, keiner ist geändert), `none` (kein Test erreicht ihn) oder `na` (die Datei ist ein Test, oder kein Seed ist Funktion oder Methode); die Tests kommen immer aus der ganzen Hülle, gleich was `-d` sagt.
- **Exitcodes**: `0` bei Erfolg; `1` ohne Projektwurzel oder Graph, bei einer Basis, die mit `-` beginnt, oder wenn Git oder der Graph scheitert; `2` bei einem Aufruffehler (ein Argument, `--base` mit `--cached`, eine ungültige Tiefe).

### `loomux graph viz [dir]` *(spezifiziert, Stufe W3)*
Startet die lokale interaktive D3-Force / WebGL Graph-Visualisierung im Browser.
- **Flags**: `--port <p>`, `--no-open`.

---

## 7. Second Brain & Wiki (`loomux brain`)

Die fünf Datenbefehle lesen die Bereiche der einen Registry (`registry.toml` in `LOOMUX_STATE_DIR` oder dessen Plattformvorgabe) und antworten wie die Referenz; ein aufgezeichneter Fallkorpus (`testdata/cases/1b-1`) hält sie daran. Die Artefakte eines schreibgeschützten Bereichs (`index.md`, `graph.json`, `_identities.tsv`) und der Reconcile-Stempel werden aus dem Zustandsverzeichnis von loomux gelesen, unter `areas/<flacher Scope>` und `maintenance/`; ein Bereichsverzeichnis, dessen `.loomux/config.toml` fehlt oder keine `[area]`-Tabelle trägt, ist nicht deklariert.

- **Kanal**: Jeder Befehl nimmt `--channel local|cloud` (Standard `local`). Ein Bereich mit `[privacy] mode = "local_only"` existiert im Kanal `cloud` nicht; `[privacy] never`-Globs gelten in jedem Kanal. Verschachtelung hebt `local_only` nicht auf: Wo der Baum eines anderen Bereichs das Wiki oder den Quellbaum eines `local_only`-Bereichs enthält, bleibt jeder Pfad darin im Kanal `cloud` auch über diesen Scope verborgen — `brain read` beantwortet ihn wie eine fehlende Datei, und Suchtreffer, Katalogzeilen, Nachbarn und Befunde von `brain status` darunter fallen weg. Der Kanal `local` bleibt unverändert.
- **Usage-Fehler** (Exit `2`): die Usage-Zeile, dann `loomux brain <befehl>: error: <grund>` bei fehlendem Argument, ungültiger Wahl oder `-n` kleiner 1, und `loomux brain: error: <grund>`, wenn der Befehl fehlt oder unbekannt ist oder Argumente übrig bleiben.
- **Laufzeitfehler** (Exit `1`): `error: <grund>` auf `stderr` und nichts auf `stdout` — ein unbekannter Scope, eine Verweigerung, ein fehlender Abschnitt, ein kaputtes `graph.json` oder Identitätsregister, ein fehlendes oder unlesbares Manifest irgendeines registrierten Bereichs, eine fehlende oder kaputte Registry, eine nicht erreichbare Suchmaschine.
- **Ratschläge**: Die Meldungen nennen loomux' eigene Befehle, `loomux reindex`, `loomux reconcile` und `loomux embed`; die aufgezeichneten Fälle von 1b-1 und 1b-2 tragen den Wortlaut der Referenz über Umschreiberegeln ihrer Importkarten.

### `loomux brain search <anfrage> [--scope <scope>] [--profile fast|full|keyword] [-n <n>] [--channel local|cloud]`
Durchsucht die sichtbaren Bereiche (Standard `--scope all`) über den qmd-MCP-Daemon unter `http://localhost:8765/mcp`.

- **Profile**: `fast` (Standard) Vektorsuche ohne Reranking und ohne Anfrageerweiterung; `keyword` BM25-Keyword-Suche; `full` die hybride Kette mit Erweiterung und Reranking. `-n` (Standard `5`) muss mindestens 1 sein.
- **Daemon**: Antwortet dort niemand, startet loomux `qmd mcp --http --daemon --port 8765` entkoppelt, schreibt `note: starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm.` auf `stderr` und wartet bis zu 60 s auf ihn.
- **Backbone**: Worauf qmd rechnet, sagt `[search] backbone` der rechnerweiten `<zustand>/config.toml` (`cuda`, `vulkan` oder `cpu`, Vorgabe `cuda`; `loomux config --global set search.backbone vulkan`). `vulkan` gibt qmd `QMD_LLAMA_GPU=vulkan` mit, `cpu` `QMD_FORCE_CPU=1`, `cuda` nichts. Ein vom Nutzer gesetztes `QMD_LLAMA_GPU` oder `QMD_FORCE_CPU`, auch ein leeres, gewinnt und bleibt unangetastet. Die Einstellung gilt für den Daemon, den loomux startet (für `brain search`, die MCP-Werkzeuge und `serve`), und für jede qmd-Kommandozeile, die loomux aufruft (`brain status`, `reindex`, `embed`, den Korpusmodus von `dev bench search`). Ein Daemon, der schon läuft, behält das Backbone, mit dem er gestartet wurde, gleich wer ihn gestartet hat, bis sein Prozess endet; `loomux serve stop` beendet nur den Dienst von loomux, nicht den Daemon von qmd. Zum Wechseln den Prozess beenden, der auf Port 8765 lauscht — unter Windows `Get-NetTCPConnection -LocalPort 8765 -State Listen | ForEach-Object { Stop-Process -Id $_.OwningProcess }`, unter POSIX `lsof -ti tcp:8765 -sTCP:LISTEN | xargs kill` —, und die nächste Suche startet den Daemon mit dem neuen Backbone. `qmd mcp stop` ist dafür nicht verlässlich: ein einziges `qmd status`, das loomux selbst aufruft (`brain status`, `reindex`, `embed`), löscht die PID-Datei von qmd, und `qmd mcp stop` antwortet dann `Not running (no PID file).`, während der Daemon weiter antwortet. Ein `[search]`-Block, der sich nicht lesen lässt, ist die Antwort der Suchmaschine, mit Datei und Schlüssel: `brain search`, `reindex` und `embed` scheitern mit Exit `1`, `brain status` sagt es in seinen Zeilen zur Suchmaschine, und was keine Suchmaschine fragt (`catalog`, `read`, `neighbors`), bleibt unberührt.
- **Ausgabe**: je Treffer `brain://<scope>/<pfad>:<zeile>  <score>%  <titel>`, jede Snippet-Zeile um vier Leerzeichen eingerückt, dann eine Leerzeile; ohne Treffer genau `no matches`.
- **Befunde**: nach den Treffern `note: <befund>`-Zeilen auf `stderr`, in dieser Reihenfolge — die Suchmaschine antwortete zweimal leer, ein Treffer fehlt im Identitätsregister seines Bereichs, wie viele Treffer zurückgehalten wurden, der Reconcile-Stempel ist 24 Stunden alt oder älter.
- **Exit-Codes**: `0` mit Treffern oder `no matches`; `1` bei einem Laufzeitfehler, auch wenn die Suchmaschine nicht antwortet — nie eine leere Antwort stattdessen; `2` bei einem Usage-Fehler.

### `loomux brain catalog [--scope <scope>] [--channel local|cloud]`
Gibt den Katalog der sichtbaren Bereiche oder eines Bereichs aus.

- **Ausgabe**: mit `--scope all` (Standard) `# brain`, eine Leerzeile und je sichtbarem Bereich `* [<scope>](brain://<scope>/)`, nach Scope sortiert; mit einem benannten Scope das `index.md` dieses Bereichs, streng UTF-8, Zeilenenden zu `\n` gefaltet.
- **Exit-Codes**: `0`; `1` bei unbekanntem Scope, fehlendem `index.md` oder einem anderen Laufzeitfehler; `2` bei einem Usage-Fehler.

### `loomux brain read <pfad> --scope <scope> [--section <titel>] [--channel local|cloud]`
Gibt eine Datei eines Bereichs oder einen Abschnitt daraus aus.

- **Lesen**: streng UTF-8, Zeilenenden zu `\n` gefaltet. `--section` gibt ab der Überschrift mit diesem Titel bis zur nächsten Überschrift gleicher oder höherer Ebene aus.
- **Verweigerungen** (Exit `1`): `<scope>/<pfad> leaves the area`; `<scope>/<pfad> is excluded by [privacy] never`; `<scope>/<pfad> is the review centre; refused on the cloud channel`; `no section titled '<titel>'`.
- **Exit-Codes**: `0`; `1` bei einer Verweigerung oder einem anderen Laufzeitfehler; `2` bei einem Usage-Fehler.

### `loomux brain neighbors <pfad> --scope <scope> [--channel local|cloud]`
Gibt die Links in eine Seite hinein und aus ihr heraus aus, gelesen aus dem `graph.json` des Bereichs.

- **Ausgabe**: `incoming: <a>, <b>` und `outgoing: <c>`, jede Liste sortiert, `-` für eine Richtung ohne Links.
- **Exit-Codes**: `0`; `1` bei einer Verweigerung, einem Bereich ohne `graph.json` (``<scope>: never indexed; run `loomux reindex` ``) oder einem anderen Laufzeitfehler; `2` bei einem Usage-Fehler.

### `loomux brain status [--channel local|cloud]`
Gibt aus, was man wissen muss, bevor man einer Antwort traut, eine Zeile je Befund.

- **Zeilen, in dieser Reihenfolge**: immer der letzte Abgleich (``last reconcile: never; run `loomux reconcile` ``, ``last reconcile: <iso>; older than 24 h, run `loomux reconcile` `` oder `last reconcile: <iso>`); je sichtbarem Bereich in Registry-Reihenfolge Include-Globs, die die Suchmaschine nicht sieht, ein Pfad, der nicht existiert, ein nie indizierter Bereich, weniger als die Hälfte aufgelöster Links und indizierte Dokumente, die die Suchmaschine nicht kennt; über alle sichtbaren Bereiche derselbe Inhalt unter mehreren Pfaden; einmal Dokumente, die indiziert, aber noch nicht durchsuchbar sind.
- **Suchmaschine**: Zwei Zeilen fragen die qmd-CLI (`qmd ls <collection>`, `qmd status`); antwortet sie nicht, sagt die Zeile das, und der Befehl läuft weiter.
- **Exit-Codes**: `0`; `1` bei einem Laufzeitfehler (Registry, Manifest, Stempel, `graph.json`, Identitätsregister); `2` bei einem Usage-Fehler.

### Wiki-Pflege: `loomux brain check`, `loomux lint`, `loomux wiki`

Seit Stufe 3c. Ein aufgezeichneter Fallkorpus (`testdata/cases/3c`) hält sie an der Referenz. Registry und Erklärungen kommen wie bei den Pflegebefehlen aus `LOOMUX_STATE_DIR`. Keiner nimmt `--state-dir`.

#### `loomux brain check file <pfad> | bundle --scope <scope> | all [--notes]`
Prüft Seiten nach den Achsen `okf` (was ein fremder Leser des Open Knowledge Format verlangt) und `house` (die strengeren Hausregeln samt Föderation: `wrong-direction`, `unlisted-area`).

- **Breiten**: `file` eine Seite ohne Nachbarn; `bundle` einen registrierten Bereich, gegen die ganze Registry; `all` jeden Bereich mit Wiki. Die vierte Breite der Referenz, `code`, gibt es nicht: die Code-Lanes gehören `loomux check`.
- **Ausgabe** auf `stdout`: je Befund `[<stufe>] <achse>/<regel> <scope>/<pfad>: <meldung>`, sortiert nach Scope, Pfad, Achse und Regel. Notizen erscheinen nur mit `--notes`, das nach der Breite an beliebiger Stelle stehen darf.
- **Exit-Codes**: `0` geprüft und ohne Fehler; `1` mindestens ein Fehler-Befund; `2` der Lauf fand nicht statt — keine oder eine unbekannte Breite, ein Pfad, der fehlt oder keine Datei ist, `bundle` ohne `--scope` oder mit `--scope all` (das ist die Breite `all`), ein unbekannter Scope, eine Registry, die sich nicht lesen lässt (`error: <grund>` auf `stderr`).

#### `loomux lint [<datei> | --file <datei>] [--scope all|<scope>] [--root <pfad>]`
Ohne Datei der Lint über die registrierten Bereiche nach den zwölf Regeln von `lint.py` der Referenz; mit einer Datei (einem Pfad, der eine Datei ist, oder einem Namen auf `.md`) die Einzelseite der Stufe 1a, deren Regeln auch `loomux wiki-gate`, die Lane `lint/wiki` und der post-edit-Hook fahren.

- **Über Bereiche**: `--scope all` (Standard) jeder Bereich mit Wiki-Pfad außer einem Arbeitsbereich, der kein `[area]` deklariert und wortlos übergangen wird; sonst genau einer. Je Bereich eine Kopfzeile, darunter `  <pfad>:<regel>: <meldung>` oder `  no findings`; am Ende `no findings` oder `<n> findings (<e> errors, <w> warnings)`.
- **Regeln**, in der Reihenfolge der Ausgabe: `broken-frontmatter`/`missing-type`, `no-sources`, `orphan`, `unlisted-area`, `dead-link`, `outside-area` (Warnung), `wrong-direction`, `conflict-count`, `untouched` (Warnung), `stale`, `implemented-without-commit`, `long-planned` (Warnung). Die letzten beiden nur in Bereichen der Familie `project/`; `unlisted-area` nur im Wegweiser.
- **Weigerungen** (Exit `1`, `error: <grund>`, nichts auf `stdout`): ein unbekannter Scope, ein Arbeitsbereich, der kein `[area]` deklariert (`… lint sweeps only a brain area`), ein Bereich ohne Wiki-Pfad, ein Wiki-Pfad, der kein Verzeichnis ist (``… run `loomux wiki init --scope <scope>` first``, auch im Lauf über alle), eine Registry, die sich nicht lesen lässt. Eine Erklärung, die sich nicht lesen lässt, beendet den Lauf dort, wo er steht.
- **Exit-Codes**: `0` ohne Fehler-Befund, auch mit Warnungen; `1` mit mindestens einem Fehler-Befund oder einer Weigerung; `2` bei einem Usage-Fehler.

#### `loomux wiki-gate [--root <pfad>]`
Das Wiki-Tor eines Projekts mit aktivem Wiki-Bündel. Es meldet `wiki-drift`, wenn `git status` geänderten Code zeigt, im Wiki aber nichts geändert ist, und jeden Fehler-Befund des Bündel-Lints als `wiki-lint:<regel>`. Ein Projekt ohne Wiki-Bündel besteht.

- **Flags**: `--root <pfad>` — Projektwurzel; ohne Angabe das Arbeitsverzeichnis.
- **Ausgabe**: `OK: Wiki Gate passed. …` auf `stdout`, oder die Verstöße als `  • [<name>] <meldung>` auf `stderr`, gefolgt von `Found <n> violation(s).`
- **Exit-Codes**: `0` ohne Verstoß; `1` mit mindestens einem Verstoß, oder wenn sich das Arbeitsverzeichnis nicht lesen lässt; `2` bei einem unbekannten Flag.

#### `loomux wiki init --scope <scope>`
Legt das Gerüst des Wiki-Bündels eines Bereichs an (`_schema.md`, `index.md`, `log.md`, `audit.md`, `_identities.tsv`) und nennt jede geschriebene Datei. Eine vorhandene Datei bleibt, wie sie ist. Exit `1` bei einem unbekannten Scope, einem Bereich ohne Wiki-Pfad oder einem schreibgeschützten Bereich.

#### `loomux wiki types`
Zählt die Seitentypen über alle Bereiche mit Wiki: je Typ `<typ> [<rang>]: <summe> (<scope>: <n>, …)`, nach Summe absteigend, dann nach Name. Der Rang ist `core`, `catalogue`, `origin`, `declared` oder `unknown`; ein unbekannter Typ trägt das Präfix `? `, ein bekannter Altname ` -> <katalogname>`. Über Bereiche zählt der schlechteste Rang. Exit `0`, außer die Registry, eine Erklärung oder eine Seite lässt sich nicht lesen (`1`).

#### `loomux wiki retype --scope <scope> --from <alt> --to <neu>`
Benennt einen Seitentyp in einem Bündel um und nennt jede geschriebene Seite. Geändert wird nur die Zeile `type:` des Frontmatters; eine geschriebene Seite wird ganz auf LF gefaltet. Übersprungen werden Gerüstdateien, kaputtes Frontmatter (auch ein doppelter Schlüssel), Bytes, die kein UTF-8 sind, und ein gequoteter oder gefalteter Wert. Ein Zieltyp, den kein Rang kennt, ergibt eine Warnung auf `stderr`, der Lauf geht weiter. Exit `1` bei einem unbekannten Scope, einem Bereich ohne Wiki-Pfad oder einem schreibgeschützten Bereich.

### Pflege: `loomux reindex`, `loomux embed`, `loomux reconcile`, `loomux area add`

Vier Befehle auf oberster Ebene seit Stufe 3a; ein aufgezeichneter Fallkorpus (`testdata/cases/3a`) hält sie an der Python-Referenz. Sie lesen und schreiben allein im Zustandsverzeichnis von loomux.

- **Umgebung**: `LOOMUX_STATE_DIR` hält die Registry, die Artefakte schreibgeschützter Bereiche, `maintenance/` und `qmd-collections.json`. qmds `index.yml` wird über `XDG_CONFIG_HOME` gefunden, sonst unter `~/.config`.
- **Kein `--state-dir`**: Die Referenz nimmt es an allen vieren an; loomux lehnt es ab wie jede unbekannte Flagge (Exit `2`). Der Zustand kommt aus der Umgebung, dem einen Zustandsmodell aller loomux-Befehle.
- **Positionale Argumente** (Exit `2`): Keiner der vier nimmt eines. Ein Wort, das nach den Flaggen übrig bleibt, wird mit `<befehl>: unrecognized arguments: <wörter>` abgelehnt, bevor Umgebung oder qmd gefragt werden.
- **Meldungen** des Abgleichs sind deutsch, wörtlich die der Referenz.

#### `loomux reindex [--registry <pfad>]`
Fährt einen Abgleich über die registrierten Bereiche, baut danach je Bereich die Verzeichniskataloge (`index.md`), den Linkgraphen (`graph.json`) und das Identitätsregister (`_identities.tsv`) neu und trägt die Bereiche als Sammlungen in qmds `index.yml` ein. Ein schreibbarer Bereich hält seine Artefakte im eigenen Baum; die eines schreibgeschützten werden über ein Staging-Verzeichnis nach `<zustand>/areas/<scope>/` geschrieben und als Ganzes eingetauscht.

- **`--registry`**: eine `registry.toml` oder das Verzeichnis, das eine hält; Standard ist `registry.toml` im Zustandsverzeichnis.
- **Aufholung**: Der Abgleich läuft zuerst, damit eine geänderte Quelle zum Fall wird, bevor der Indexlauf ihren Hash fortschreibt. Fälle, die er eröffnet, stehen auf `stderr`, und der Lauf **geht weiter**; ein Tresor ohne Prüfzentrum bekommt eine Warnung und wird indiziert; jeder andere Fehlschlag des Abgleichs beendet den Befehl, bevor etwas indiziert ist. Für einen Fall eines `local_only`-Bereichs fragt der Abgleich auch das lokale Modell (30 s je Fall, sobald das Modell geladen ist, siehe [`loomux reconcile`](#loomux-reconcile)), und ein kaputtes `[model]`, das diesen Abgleich beendet (wann, steht dort), ist ein solcher Fehlschlag: `reindex` bricht ab.
- **Bereichssperre**: Jeder Bereich wird unter `<zustand>/areas/<scope>.lock` indiziert, der Sperre, die `loomux approve` für denselben Bereich nimmt; so schreibt keiner das Identitätsregister neu, während der andere es liest. Ein zweiter Läufer wartet, bis der erste sie freigibt. Die Datei bleibt liegen wie `registry.lock`.
- **Ausgabe**: `indexed the areas of <pfad>` auf `stdout`; auf `stderr` die aktualisierten oder entfernten Sammlungen und jede verweigerte, weil qmd schon eine gleichnamige führt, die brain nicht angelegt hat.
- **Exit-Codes**: `0` bei Erfolg, und auch dann, wenn im Zustandsverzeichnis keine Registry liegt (`no areas registered in <pfad>; nothing to index` auf `stdout`); `1` bei einer mit `--registry` benannten Registry, die es nicht gibt, einer Registry, die sich nicht lesen lässt, einer fehlgeschlagenen Aufholung, einem fehlgeschlagenen Indexlauf oder einer verweigerten Sammlung; `2` bei einem Usage-Fehler.

#### `loomux embed [--registry <pfad>]`
Lässt qmd die Vektoren erzeugen, die der Indexlauf offen lässt, für jeden registrierten Bereich.

- **qmd zuerst**: Ohne `qmd` auf dem `PATH` schreibt er `loomux embed: qmd is not on PATH; install it with: npm install -g @tobilu/qmd` und endet mit Exit `1`, bevor er die Registry liest.
- **Ausgabe**: `embedded <n> area(s)` auf `stderr`.
- **Exit-Codes**: `0` bei Erfolg, und auch dann, wenn im Zustandsverzeichnis keine Registry liegt (`no areas registered in <pfad>; nothing to embed` auf `stdout`); `1` bei fehlendem qmd, einer benannten Registry, die es nicht gibt, einer Registry, die sich nicht lesen lässt, oder einer Suchmaschine, die ablehnt; `2` bei einem Usage-Fehler.

#### `loomux reconcile`
Misst jede Quelle der registrierten Bereiche an ihrem Identitätsregister und eröffnet für jede Wiki-Seite, die aus einer geänderten Quelle abgeleitet ist, einen Fall (`case.toml` und ein Paket mit dem Diff) im Prüfzentrum, dem einen Verzeichnis, das ein Bereich unter `[layout] review` erklärt. Ein in `maintenance/merge-events.tsv` festgehaltener Merge eröffnet einen Fall mit den Belegen des Merges, nach den Quellfällen.

- **Ausgabe** auf `stdout`: `<n> Quellen geprüft, <m> davon gehasht`, je Fall eine Zeile (Verzeichnis, Bereich, Ziel, Zustand und `manuell` für einen Fall, der eine Entscheidung von Hand verlangt, durch Tabs getrennt, um zwei Leerzeichen eingerückt), dann `<k> Fälle`.
- **Ein Fall ist kein Fehlschlag**: Offene Fälle lassen den Exit-Code bei `0`.
- **Verschachtelte Wikis**: Eine Seite gehört dem tiefsten registrierten Bereich, dessen Wiki sie enthält. Ein Bereich, dessen Wiki das Wiki eines anderen Bereichs enthält, geht nicht hinein; eine geänderte Quelle eröffnet für eine solche Seite also einen Fall, im inneren Bereich, und nie einen zweiten im umschließenden — dessen Paket sonst etwa den Diff einer `local_only`-Quelle in einen `manual_cloud`-Bereich trüge. Dasselbe gilt für die Seiten, nach denen ein Merge fragt.
- **Stempel**: Der Abgleich schreibt `maintenance/last-run.txt` in UTC, den `brain status` und `brain search` lesen.
- **Lokales Modell**: Für einen Fall eines `local_only`-Bereichs fragt der Abgleich das lokale Modell (`[model]` der rechnerweiten `config.toml`, siehe [`loomux config`](#10-konfiguration-loomux-config)) nach einem Vorschlag; einer, dessen Behauptungen alle die Belegbindung bestehen, landet als `proposal.md` neben dem Fall, mit `prompt_version` in `case.toml`, alles andere hinterlässt einen manuellen Fall mit dem Vermerk `manual review: the local proposer returned no usable proposal (slice-6 spec §3)`. Jede Frage hat 30 s. Vor der ersten Frage eines Bereichs in einem Durchgang lädt loomux das Modell in Ollama mit den Einstellungen der Fragen (samt `num_ctx`, für das Ollama ein Modell lädt) und lässt es ein Token antworten; dieses Vorwärmen hat keine eigene Frist, nur ein Abbruch des Durchgangs beendet es vorzeitig, und so kann der erste Fall so lange dauern wie das Laden. Scheitert das Vorwärmen (keine Verbindung, ein Status außerhalb von 2xx), ist das ein Ausfall: Die Fälle dieses Bereichs bleiben für den Rest des Durchgangs manuell, und für sie geht keine weitere Anfrage hinaus. Ist das Modell aus, bleibt der Fall manuell wie bisher. Die Einstellungen werden nur gelesen, wenn ein `local_only`-Bereich registriert ist; dann beendet ein `[model]`-Block, der sich nicht lesen lässt, oder ein Endpunkt außerhalb des Loopbacks, während Modell und Rolle `propose` für einen solchen Bereich an sind, den Abgleich mit Exit `1`, nachdem die Quellen gemessen sind und bevor ein Fall geschrieben wird.
- **Exit-Codes**: `0` für einen Abgleich, der bis zum Ende lief; `1`, wenn sich eine Falldatei nicht lesen lässt (`unreadable case: <eintrag>` auf `stderr`), sowie bei einer Registry, die sich nicht lesen lässt, einem Tresor, der kein oder zwei Prüfzentren erklärt, oder einem anderen Fehlschlag (`error: <grund>`); `2` bei einem Usage-Fehler.

#### `loomux area add [--path P] [--scope S] [--wiki W] [--sources S] [--merge-branch B] [--privacy M] [--no-reindex] [-y|--yes]`
Meldet ein Repository als Bereich an und richtet es ein: der Registry-Eintrag (unter einer Sperre geschrieben); `.loomux/config.toml` mit `[area]`, `[layout]`, `[index]`, `[privacy]` und `[maintenance]`, wenn das Repository keine hat; die Routing-Regel, einmal an `AGENTS.md` angehängt; das Gerüst des Wiki-Bündels; danach `loomux reindex` samt Aufholung.

- **Vorgaben**: `--path` das Arbeitsverzeichnis; `--scope` `project/<verzeichnisname>`; `--sources` `docs`, wenn es ein Verzeichnis `docs/` gibt, sonst `.`; `--wiki` `<repo>/docs/wiki` oder `<repo>/wiki` (muss absolut sein); `--merge-branch` der Branch, den git nennt, ohne einen `master`; `--privacy` `manual_cloud`, einer von `automatic_cloud`, `local_only`, `manual_cloud`.
- **Registry zuerst**: Ein schon registrierter Scope wird abgelehnt, bevor das Repository berührt wird.
- **Eine behaltene Konfiguration**: Eine vorhandene `.loomux/config.toml` bleibt Byte für Byte stehen, mit einer Warnung, wenn sie kein `[area]` oder einen anderen Scope erklärt. Eine, die der Deklarationsleser ablehnt, beendet den Befehl, ohne dass etwas registriert ist.
- **Unterschiede zu `brain init`**: kein `.mcp.json` und keine Agenten-Hooks (`loomux init`, Stufe 4); der Indexlauf findet wirklich statt, außer mit `--no-reindex`; der Branch wird als `[maintenance] branch` geschrieben, nicht als `merge_branch`; `--privacy` wird geprüft; der erste Bereich einer Maschine braucht keine von Hand angelegte Registry-Datei. `-y`/`--yes` wird angenommen und ändert nichts.
- **Exit-Codes**: `0` oder der Exit-Code des Indexlaufs; `1` bei einem Pfad, der kein Verzeichnis ist, einem ungültigen Scope, einem relativen `--wiki`, einem abgelehnten Registry-Eintrag, einer unlesbaren Datei oder einem fehlgeschlagenen Schreiben; `2` bei einem Usage-Fehler, einem fehlenden oder unbekannten Unterbefehl (mit der Usage-Zeile) oder einem unbekannten `--privacy`.

### Eingang: `loomux convert`, `loomux fetch`

Zwei Befehle auf oberster Ebene seit Stufe 4d; ein aufgezeichneter Fallsatz (`testdata/cases/4d`, 29 Fälle) hält `convert` an der Python-Referenz, und eine Aufnahme der eigenen Ausgabe von Poppler hält den PDF-Weg am echten Werkzeug. Beide schreiben in den Eingang eines Bereichs, das Verzeichnis, das sein Manifest als `[layout] inbox` nennt, relativ zum Pfad des Bereichs.

- **Befehle des Menschen**: Der Wächter verweigert beide einem Agenten, denn dorthin verbietet die Schreibschranke Agenten das Schreiben (siehe [`hook pre-tool-use`](#loomux-hook-pre-tool-use)); nur ein alleinstehendes `--help` oder `-h` geht durch.
- **Modul Brain**: Mit `[modules] brain = false` im Projekt, das die Suche vom Arbeitsverzeichnis nach oben findet, geben beide `loomux <befehl>: the brain module is off in <datei> ([modules] brain = false)` aus und enden mit `1`, bevor etwas gelesen ist. Außerhalb eines Projekts ist nichts abgeschaltet.
- **Umgebung**: Registry und Bereichsdeklarationen kommen aus `LOOMUX_STATE_DIR`, wie bei der [Pflege](#pflege-loomux-reindex-loomux-embed-loomux-reconcile-loomux-area-add). `--state-dir` und `--channel`, die die Referenz annimmt und nicht nutzt, sind unbekannte Flags (Exit `2`).
- **Externe Programme**: Beide werden auf dem `PATH` gesucht und nie installiert: `pdftotext` von Poppler (`winget install --id oschwartz10612.Poppler -e`) und `yt-dlp` (`winget install --id yt-dlp.yt-dlp -e`). Ein fehlendes wird mit diesem Befehl genannt.

#### `loomux convert [<datei>]`
Geht in der Reihenfolge der Registry durch den Eingang jedes Bereichs, lässt einen Bereich aus, der schreibgeschützt ist, ein Arbeitsbereich ohne `[area]` ist, keinen Eingang nennt oder dessen Eingang kein Verzeichnis ist, und wandelt darin jede reguläre Datei außer `*.md`, nach Namen sortiert (unter Windows in Kleinschreibung, wie Python dort Pfade sortiert). Mit einer `<datei>` wandelt es nur diese, wo immer sie liegt, und fragt nie das Modell. Jedes Ergebnis wird neben seiner Quelle als `<name>.<endung>.md` geschrieben — aus `doku.pdf` und `doku.txt` werden `doku.pdf.md` und `doku.txt.md`.

- **Formate**: eine `.pdf` und eine `.txt`, deren erste 8192 Zeichen eine Transkriptmarke am Zeilenanfang tragen: die Klammerform `[mm:ss]` oder `[hh:mm:ss]` oder die Bereichsform `hh:mm:ss - hh:mm:ss` allein auf ihrer Zeile. Die Fragmente werden, ohne ein Wort zu ändern, zu Absätzen von etwa 1200 Zeichen verbunden, jeder mit der Marke seines ersten Fragments. Alles andere bleibt liegen: `skipped: <name>: no converter knows this format`.
- **PDFs**: über `pdftotext -layout -enc UTF-8 -eol unix <name> -`, im Eingang gestartet, höchstens 2 Minuten je Datei. Vor der ersten PDF eines Laufs muss `pdftotext -v` Poppler nennen; auch xpdf bringt ein `pdftotext` mit und schreibt einen anderen Text, es wird darum wie ein fehlendes Programm abgewiesen, und jede PDF des Laufs bleibt mit `skipped: <name>: <programm> is not Poppler's pdftotext (…); install Poppler with: …` liegen. Ein Lauf ohne PDF braucht das Programm nie. Jede Seite wird für sich an der Scan-Schwelle gemessen: Eine Seite mit weniger als 100 Zeichen gilt als Scan und fällt weg (`skipped: <name>: <n> page(s) skipped as scanned`, das Ziel wird trotzdem geschrieben); eine PDF nur aus Scans schreibt nichts (`no extractable text, looks like a scan`), eine ohne Seiten ebenso (`no pages to extract`), und eine, die `pdftotext` ablehnt, heißt `cannot be read as a PDF (pdftotext exited <n>: <seine erste Zeile>)`.
- **Der Herkunftskopf**: YAML-Frontmatter, dann eine Leerzeile und der Text:
  ```yaml
  ---
  source_url: https://www.youtube.com/watch?v=<id>
  retrieved: 2026-09-27
  converter: brain-pdf/2
  asr: false
  description: <ein deutscher Satz>
  ---
  ```
  `source_url` wird aus einem elf Zeichen langen Lauf in Klammern im Dateinamen gelesen (eine YouTube-ID; leer ohne sie); `retrieved` ist das UTC-Datum, an dem die Quelle zuletzt geändert wurde; `converter` ist `brain-pdf/2` oder `brain-transcript/1`; `asr` ist `true` für ein Transkript, damit sein Text nie als wörtliches Zitat zählt; `description` steht nur, wenn das Modell einen Satz gab.
- **Zweiter Lauf**: Ein Ziel wird nur neu geschrieben, wenn sich sein Text ändern würde; ein unberührtes behält seine Zeit und wird nicht ausgegeben. Ein Ziel, dessen Kopf keinen Wandler nennt, hat ein Mensch geschrieben, und es wird nie überschrieben (`skipped: <ziel>: not written by us, left untouched`). Einen Satz, den ein Kopf schon trägt, behält es und fragt nie neu; ein Kopf ohne Satz wird bei jedem Lauf gefragt.
- **Das lokale Modell** (`[model]` der rechnerweiten `config.toml`, eingeengt durch das `[model]` des Bereichs, siehe [`loomux config`](#10-konfiguration-loomux-config)): Gleich welcher Datenschutzmodus des Bereichs, mit eingeschaltetem Modell und eingeschalteter Rolle schickt `convert` die ersten 1800 Zeichen des gewandelten Texts an das lokale Modell. Die Rolle `describe` fragt nach dem einen Satz des Kopfs, behalten nur, wenn er ein deutscher Satz von höchstens 22 Wörtern ist, den kein Richter abweist (zerhackte Wörter, gemessen an einer eingebetteten deutschen Worthäufigkeitstabelle unter CC BY-SA 4.0; ein Satz, den YAML nicht zurückliest); sonst behält der Kopf vier Zeilen. Die Rolle `place` fragt für eine Datei, die dieser Lauf schrieb, in welchen Bereich sie gehört, und bietet nur Bereiche an, die nicht schreibgeschützt und nicht offener sind als der des Eingangs (`local_only` < `manual_cloud` < `automatic_cloud`); eine bekannte Antwort ist eine Zeile `suggested: <ziel>: belongs in <scope>, left in the inbox` auf `stdout`, und nichts wird verschoben. Keine Antwort, ein Ausfall oder ein abgewiesener Satz ist kein Befund. Jede Rolle jedes Eingangs wärmt das Modell vor ihrer ersten Frage vor wie [`loomux reconcile`](#loomux-reconcile) (keine Frist für das Laden, danach 30 s je Frage); ein gescheitertes Vorwärmen lässt diese Rolle für den Rest des Laufs ohne Antwort. Die Einstellungen werden vor der ersten Datei gelesen: Ein `[model]`, das sich nicht lesen lässt, oder ein Endpunkt außerhalb des Loopbacks, während eine Rolle für einen Eingang an ist, beendet den Lauf, ohne dass etwas gewandelt ist.
- **Ausgabe**: Auf `stdout` jedes geschriebene Ziel, dann die `suggested:`-Zeilen; auf `stderr` eine Zeile `skipped: <grund>` für jede Datei, die für einen Menschen liegen bleibt, und `error: <grund>` für einen Fehler, der den Lauf beendet (eine Registry oder Deklaration, die sich nicht lesen lässt, ein Eingang, der sich nicht auflisten lässt — was frühere Eingänge schrieben, steht dann schon da).
- **Exit-Codes**: `0`, wenn nichts liegen blieb; `1`, sobald eine `skipped:`-Zeile ausgegeben wurde, bei einem Fehler, der den Lauf beendete, und bei abgeschaltetem Modul Brain; `2` bei einem Usage-Fehler (ein unbekanntes Flag, mehr als eine Datei). Ein Scan, der im Eingang liegen bleibt, wiederholt seine `skipped:`-Zeile und Exit `1` bei jedem Lauf, wie in der Referenz.

#### `loomux fetch <url> [--scope <scope>]`
Lässt `yt-dlp` die Untertitel eines Videos in ein frisches temporäres Verzeichnis schreiben und legt sie im Eingang des Bereichs `--scope` (Vorgabe `knowledge`) als Transkript in Klammerform ab, ein Fragment je Absatz (`[hh:mm:ss] text`), bereit für `convert`. loomux selbst spricht nie ins Netz.

- **Der Aufruf**: `yt-dlp --ignore-config --no-playlist --no-progress --skip-download --write-subs --write-auto-subs --sub-langs de,en --sub-format json3 --write-info-json --ignore-errors -o v <url>`, höchstens 10 Minuten. `--ignore-config` hält eine Nutzerkonfiguration davon ab, Name oder Ort zu ändern, `--no-playlist` holt bei einer Adresse mit `&list=` nur das Video, `--ignore-errors` hält eine scheiternde Spur (ein 429 auf einer automatisch übersetzten) davon ab, yt-dlp zu beenden, bevor es die Info-JSON schreibt. Der Exit-Code von yt-dlp entscheidet nichts; was es schrieb, entscheidet.
- **Die Spur**: manuelle Untertitel vor automatischen, `de` vor `en` innerhalb jeder Art; innerhalb einer Sprache wählt yt-dlp die Spur. Ein Video ohne Spur endet mit `<url>: no subtitle track to fetch, and this system does no ASR`.
- **Die Datei**: `<titel> (<id>).txt` für eine Adresse mit YouTube-ID, sonst `<titel>.txt`; der Titel verliert, was Windows in einem Namen verbietet, und jedes Steuerzeichen und wird auf 150 Zeichen gekürzt, `video`, wenn nichts bleibt. Der Eingang wird angelegt, wenn er fehlt; eine gleichnamige Datei wird ersetzt.
- **Verweigert**: eine URL, die mit `-` beginnt (yt-dlp läse sie als Option, mit `--` oder ohne: `loomux fetch: a URL does not begin with '-': <url>`, Exit `2`); eine URL, die auf `youtube.com`, `www.`, `m.` oder `music.youtube.com` nur eine Playlist nennt (die Playlist-Seite `/playlist`, oder `/watch` mit `list=`, aber ohne `v=`, die yt-dlp auf die Playlist-Seite umleitet), denn `--no-playlist` grenzt sie nicht auf ein Video ein (`loomux fetch: a URL that names only a playlist is not fetched, give the URL of one video: <url>`, Exit `2`; eine Watch-URL mit `&list=` wird genommen); keine oder mehr als eine URL (Exit `2`); ein Scope, den die Registry nicht kennt, ein Arbeitsbereich, der kein `[area]` deklariert (`… fetch files only into a brain area`), ein Bereich ohne Eingang, ein schreibgeschützter Bereich (Exit `1`, `error: …`).
- **Nicht verweigert, obwohl yt-dlp sie ebenfalls als Playlist liest**: eine nackte Playlist-ID (`PL…`); ein anderer Pfad auf `youtube.com` mit `list=`, aber ohne `v=` (etwa `/embed/videoseries?list=…`); dieselben Seiten auf einer anderen Subdomain von `youtube.com` oder auf `youtubekids.com`. yt-dlp läuft dann durch jeden Eintrag der Playlist; stattdessen die URL eines Videos angeben.
- **Ausgabe**: der Pfad der geschriebenen Datei auf `stdout`; `error: <grund>` auf `stderr`.
- **Exit-Codes**: `0` für eine geschriebene Datei; `1` bei fehlendem `yt-dlp`, einem verweigerten Bereich, einem Abruf ohne Spur oder einer unlesbaren Antwort und bei abgeschaltetem Modul Brain; `2` bei einem Usage-Fehler.

### Der post-merge-Hook: `loomux merge-hook install|status|remove|record`
Der Hook, der `reconcile` einen gelandeten Merge meldet, in jedem Repository eines Bereichs, dessen Manifest `[maintenance] on_merge = true` sagt. Er heißt `merge-hook` und nicht `hook`, weil `hook` der Namensraum der Host-Hooks ist; ein aufgenommener Fallkorpus (`testdata/cases/4a2`, 14 Fälle, elf ohne Unterschied) hält ihn an der Referenz. `loomux init` ruft `merge-hook install` als seinen Teil `merge-hook` (aus in einem Checkout von loomux, dessen Hookverzeichnis das eingecheckte `.githooks` ist).

- **Der Hook backt nichts ein**: Die Datei ist überall derselbe Text, ein kurzes `sh`, das `"${LOCALAPPDATA}/loomux/bin/loomux.exe" merge-hook record` ruft und dessen Ausgabe verwirft. Welches Repository und welcher Zweig zählen, entscheidet die Registry zur Merge-Zeit; nichts in der Datei kann veralten. Sie liegt, wo git Hooks sucht, `core.hooksPath` eingeschlossen. Die Einrichtungen merkt sich `maintenance/hooks.tsv` unter `LOOMUX_STATE_DIR` (drei Felder, `scope`, `repo`, `hook`; eine Zeile der Referenz mit mehr wird über ihre ersten drei gelesen).
- **`install`** schreibt den Hook in jedes einwilligende Repository und merkt ihn sich; eine Datei mit der Marke der Referenz `# brain post-merge hook` ist der Vorgänger desselben Hooks und wird ersetzt. Ein fremder `post-merge` heißt `refused` und bleibt stehen.
- **`status`** nennt den Zustand jedes Repositorys: `installed`, `missing` (gemerkt, Datei weg), `moved` (gemerkt, die Datei steht, aber `core.hooksPath` hat sich geändert und git sucht woanders; `install` schreibt ihn dorthin, wo git jetzt sucht), `not installed` (mit `: another hook` hinter dem Pfad, wenn dort eine fremde Datei steht), `unrecorded` (die Datei ist unsere, der Eintrag verloren), `orphaned` (gemerkt für einen Bereich, der nicht mehr einwilligt, oder ein verschobenes Repository) und `refused`; jeder Befehl sagt `no repository` für einen einwilligenden Bereich, dessen Pfad in keinem liegt.
- **`remove`** nimmt jeden eigenen Hook zurück, den er kennt, einen ohne Eintrag eingeschlossen; einen, den jemand durch einen eigenen ersetzt hat, nennt er `refused` und behält dessen Eintrag.
- **Ausgabe** auf `stdout`: eine Zeile je Repository, `<zustand>: <scope> — <repo>`, gefolgt von ` [<hookdatei>]`, wenn es eine gibt. Ohne Zeile: `no area consents with [maintenance] on_merge = true, and no hook is installed`.
- **`record`** ruft der Hook selbst: ein Ereignis für den Merge, der eben im Arbeitsverzeichnis gelandet ist, wenn ein Bereich es will. Es läuft im `git merge` des Nutzers, schreibt darum nie etwas und endet immer mit `0` — keine Registry, eine kaputte, kein Repository, ein nicht schreibbares Zustandsverzeichnis und überzählige Argumente eingeschlossen.
- **Der Wächter** verweigert einem Agenten `install` und `remove`, die ausführbare Dateien in Repositorys schreiben; `status` und `record` gehen durch (siehe [`hook pre-tool-use`](#loomux-hook-pre-tool-use)).
- **Exit-Codes**: `0`, auch bei einem verwaisten, fehlenden, verschobenen (`moved`) oder unverzeichneten Hook, die Befunde sind; `1`, wenn eine Zeile `refused` oder `no repository` sagt, die Registry nicht liest oder eine Hookdatei nicht geschrieben werden kann (`loomux merge-hook <sub>: <grund>` auf `stderr`); `2` bei einem Usage-Fehler (`usage: loomux merge-hook install|status|remove|record`).

### Prüfzentrum: `loomux cases`, `loomux case`, `loomux approve`

Drei Befehle auf oberster Ebene seit Stufe 3b; ein aufgezeichneter Fallkorpus (`testdata/cases/3b`) hält sie an der Python-Referenz, `approve` samt den Dateien, die es schreibt, und dem Commit, den es anlegt. Sie entscheiden die Fälle, die `loomux reconcile` ins Prüfzentrum legt. Ein Fall wird über den Namen seines Verzeichnisses im Prüfzentrum angesprochen, die erste Spalte von `loomux cases`.

- **Umgebung**: Registry und Prüfzentrum kommen aus `LOOMUX_STATE_DIR`, wie bei den Pflegebefehlen; das Prüfzentrum ist das eine Verzeichnis, das ein Bereich unter `[layout] review` erklärt.
- **Kein `--state-dir`**: Die Referenz nimmt es an allen dreien an; loomux lehnt es ab wie jede unbekannte Flagge (Exit `2`).
- **Flaggen und ID** dürfen in beliebiger Reihenfolge stehen, wie bei argparse. Eine fehlende ID ist `<befehl>: the following arguments are required: id`, ein zweites Wort `<befehl>: unrecognized arguments: <wörter>`, beides Exit `2`.
- **Meldungen** auf `stdout` sind deutsch, wörtlich die der Referenz.
- **Laufzeitfehler** (Exit `1`): `error: <grund>` auf `stderr` — eine Registry, die sich nicht lesen lässt, ein Tresor, der kein oder zwei Prüfzentren erklärt, eine ID, auf die kein Fall antwortet (`no case named '<id>' …`) oder die zwei Verzeichnisse tragen, eine `case.toml`, die sich nicht lesen lässt.

#### `loomux cases`
Listet jeden Fall im Prüfzentrum, eine Zeile je Fall: Verzeichnis, Bereich, Ziel und Zustand, durch Tabs getrennt, und `manuell` für einen Fall, der eine Entscheidung von Hand verlangt; `keine offenen Fälle`, wenn keiner wartet.

- **Ein Fall ist kein Fehlschlag**: Wartende Fälle lassen den Exit-Code bei `0`.
- **Umbenannter Fall**: Weichen die `id` in `case.toml` und der Verzeichnisname voneinander ab, sagt eine Warnung auf `stderr` das; es zählt der Verzeichnisname.
- **Exit-Codes**: `0`; `1`, wenn sich eine Falldatei nicht lesen lässt (`unreadable case: <eintrag>` auf `stderr`, die übrigen Fälle werden trotzdem gelistet), oder bei einem Laufzeitfehler; `2` bei jedem positionalen Argument.

#### `loomux case <id> [--package]`
Gibt aus, was das menschliche Tor sehen muss, bevor es entscheidet: `Fall <id> (<zustand>, ausgelöst durch <auslöser>, Gewicht <gewicht>)`, den Bereich, das Ziel, je Quelle eine Zeile `Quelle:` mit Revision und Hash, dann `package.md`, `proposal.md` und, wo einer abgelöst wurde, `superseded-proposal.md`, jede unter einer Überschrift `===== <datei> =====`, oder `(nicht vorhanden: <pfad>)`.

- **Zurückgehalten**: Ein Fall eines `local_only`-Bereichs, oder eines Bereichs, dessen Erklärung sich nicht lesen lässt, gibt statt seiner Dateien die Haltezeile aus, mit dem Ort im Prüfzentrum und `Bewusst ausgeben: loomux case --package <id>`. Das Paket trägt den Quelldiff, und ein Vorschlag zitiert ihn wörtlich.
- **`--package`**: Gibt die zurückgehaltenen Dateien trotzdem aus. Die Haltezeile steht weiter darüber.
- **Weitere Zeilen**: `manuell: für diesen Fall wird kein Skill-Pfad angeboten` bei einem manuellen Fall, `Vermerk: <vermerk>`, wenn eine verweigerte Freigabe einen hinterlassen hat.
- **Exit-Codes**: `0`; `1` bei einem Laufzeitfehler; `2` bei einem Usage-Fehler.

#### `loomux approve <id> [--amend <datei> | --reject | --defer]`
Entscheidet einen Fall. Ohne Flagge wird der Vorschlag des Falls freigegeben; `--amend` gibt stattdessen die genannte Datei frei; `--reject` verwirft den Vorschlag; `--defer` lässt den Fall in der Warteschlange.

- **Belegbindung**: Jede Behauptung des Vorschlags braucht ein wörtliches Zitat aus einem Segment des Pakets. Eine Behauptung ohne Beleg wird verworfen und auf `stdout` genannt (`  verworfene Behauptung: <behauptung>`); bleibt keine übrig, oder tragen die übrigen keinen Diff, der passt, wird nichts an die Seite geschrieben.
- **Eine Freigabe** prüft zuerst, dass sich weder die Zielseite noch eine zitierte Quelle seit dem Fall geändert hat, schreibt dann die Seite mit fortgeschriebener Frontmatter (`generated`, `verified` mit dem Prüfer), schiebt die Identitätsregister vor, hängt an `log.md` und `audit.md` an, entfernt das Fallverzeichnis und committet genau diese Pfade über einen eigenen Index (`<zustand>/maintenance/index`) auf den aktuellen Ref des Tresors; der Index des Nutzers bleibt unberührt. Danach laufen eine Aufholung und ein Indexlauf, dieser ohne eigene Aufholung. Die Aufholung fragt für `local_only`-Fälle das lokale Modell wie `loomux reconcile` (30 s je Fall, sobald das Modell geladen ist). Scheitert die Aufholung — ein kaputtes `[model]` eingeschlossen —, sagt eine Warnung das, und es wird nicht indiziert; scheitert der Indexlauf, nennt eine Warnung `loomux reindex`. Keines von beiden ändert den Exit-Code.
- **Eine Ablehnung** nimmt die Quellen zur Kenntnis, über denen der Fall gebildet wurde, damit der nächste `loomux reconcile` denselben Fall nicht wieder eröffnet: Sie schiebt `revision` und `content_hash` jedes passenden Eintrags in `sources[]` der Seite und die Identitätsregister vor, hängt an `audit.md` an, entfernt das Fallverzeichnis und committet alles zusammen. Text, `generated` und `verified` der Seite bleiben, wie sie sind, und ihr Hash wird nicht geprüft; eine Seite ohne Frontmatter, oder eine, die seit der Fallbildung gelöscht oder umbenannt wurde, hält keine Ablehnung davon ab, den Fall zu schließen; eine Seite, deren Frontmatter sich nicht laden lässt (etwa keine Zuordnung ist), verweigert die Ablehnung mit Exit `1`, bevor etwas geschrieben ist. Eine zitierte Quelle, die sich seit der Fallbildung erneut geändert hat, hält die Ablehnung an wie eine Freigabe (Vermerk in `case.toml`, Exit `1`): Dieser neuere Stand lag nie zur Prüfung vor. Die Referenz schiebt bei einer Ablehnung nichts vor und eröffnet den Fall bei jedem Abgleich neu; loomux weicht hier bewusst ab.
- **Bereichssperre**: Während eine Freigabe oder Ablehnung ein Identitätsregister vorschiebt, hält sie `<zustand>/areas/<scope>.lock` jedes Bereichs, der dieses Register schreibt — die Sperre, die `loomux reindex` nimmt —, und wartet auf sie, solange ein Indexlauf sie hält. `--defer` nimmt keine.
- **`--defer`** schreibt nichts: `Fall <id> zurückgestellt; er bleibt unverändert in der Warteschlange.`
- **Der Prüfer** ist `human:<konto>`, das Konto, unter dem der Befehl läuft, ohne Domäne. Keine Flagge nennt ihn.
- **Ausgabe**: `Fall <id>: approve` oder `Fall <id>: reject`, dann `committet als <sha>`. Eine Entscheidung, die geschrieben, aber nicht committet ist — der Tresor ist kein Git-Repository, oder ein Rebase oder Merge läuft —, ist `geschrieben, aber nicht committet: <grund>` (bei einer Ablehnung `entschieden, …`) auf `stderr`, mit Exit `0`: Der Tresor hat sich geändert, und ein zweiter Aufruf machte es nicht besser.
- **Weigerungen** (Exit `1`): Eine bewegte Zielseite, eine bewegte Quelle oder ein Vorschlag, den die Belegprüfung abweist, schreiben einen Vermerk in `case.toml` und, für einen Ausgang, der noch nicht vermerkt war, einen Block in `audit.md`; ein abgewiesener Vorschlag des Falls selbst, keine Datei aus `--amend`, markiert den Fall zudem `manuell`. Dann `error: <grund>`. Jede schon geänderte Datei steht nach einer Zeile `Hinweis:` auf `stderr`; nichts davon ist committet.
- **`--amend=`** mit leerem Wert nennt `.`, wie Python `Path("")` liest, und wird als Verzeichnis abgewiesen; es fällt nie auf den Vorschlag des Falls zurück.
- **Exit-Codes**: `0` für eine getroffene Entscheidung, committet oder nicht, und für `--defer`; `1` bei einer Weigerung oder einem Laufzeitfehler; `2` bei einem Usage-Fehler, auch bei zwei Entscheidungen zugleich (`loomux approve: argument --<spätere>: not allowed with argument --<frühere>`) und bei `--amend` gefolgt von einem Wort, das argparse als Option liest (`argument --amend: expected one argument`).

---

## 8. MCP-Dienst & stdio-Brücke (`loomux serve` / `loomux mcp`)

Der Dienst beantwortet zwölf Werkzeuge über Streamable HTTP — die fünf
`brain_*`-Werkzeuge und die sieben `graph_*`-Werkzeuge der Stufen G3, G4a und G4b; die Brücke
ist das, was ein MCP-Wirt startet, und sie reicht nur weiter. Beides ist in
Stufe 1b-2 entstanden. Das Web OS der Stufe W1 gibt es noch nicht, die
Upstream-Proxies ebenso wenig.

**Der Kanal ist die Adresse, kein Feld der Anfrage.** `serve` bindet zwei
Loopback-Listener, einen für `local` und einen für `cloud`, jeden mit eigenem
Zufallstoken. Ein Aufrufer mit dem cloud-Token erreicht die local-Sicht gar
nicht erst. Alles, was der Dienst schreibt — `serve.json`, `serve.lock` und
`logs/serve.log` —, liegt unter dem Zustandsverzeichnis (`LOOMUX_STATE_DIR`,
standardmäßig `%LOCALAPPDATA%\loomux`), nie unter einem festen Pfad.

### `loomux serve [--foreground]`
Startet den langlebigen localhost-MCP-Dienst. Ohne `--foreground` ist der Start
entkoppelt: der Dienst wird neben dem Aufrufer erzeugt, aus dessen Job-Object
gelöst und der Befehl kehrt sofort zurück und nennt die Logdatei. Mit
`--foreground` läuft der Dienst in diesem Prozess — der einzige Weg, auf dem ein
Mensch einen fehlgeschlagenen Start sieht, und zugleich das, was das entkoppelte
Kind ausführt.

- **Eine Instanz**: `serve.lock` plus Lebendprüfung. Ein zweites `loomux serve`
  sagt das, statt einen zweiten Dienst zu starten.
- **Breakaway**: Wird das Lösen aus dem Job-Object des Wirts abgelehnt, wird der
  Start ohne diese Bitte wiederholt und der Befehl meldet `note: the breakaway
  was refused, so this service dies with its host`.
- **Tägliche Aufholung** (seit Stufe 3c): Ist der letzte `reconcile` älter als
  24 Stunden oder gab es noch keinen, fährt der Dienst ihn beim Start selbst und
  danach alle 24 Stunden, solange er läuft; nie `reindex`. Der Durchgang
  fragt für `local_only`-Fälle das lokale Modell wie `loomux reconcile` (30 s
  je Fall, sobald das Modell geladen ist); ein kaputtes `[model]` lässt den Durchgang scheitern, was
  wie jeder andere Fehlschlag gemeldet wird und nichts anhält. Jedes
  `brain_*`-Werkzeug wartet auf den ersten Durchgang und meldet das als
  Fortschritt. Was er gefunden hat, hängt an den Antworten: `brain_status` die
  geöffneten Fälle, die unlesbaren Falldateien und einen Fehlschlag, die übrigen
  vier eine Zeile mit `! ` (der Fehlschlag oder die Zahl der Fälle), alle außer
  `brain_search` dazu die Zeile zum veralteten Stempel. Der Kanal `cloud` sieht
  nur seine Bereiche und nicht die Ursache eines Fehlschlags.
- **Exit-Codes**: `0` gestartet; `1` läuft bereits, oder Erzeugung bzw. Lauf
  sind gescheitert; `2` ein unbekanntes Argument oder ein unbekannter
  Unterbefehl.

### `loomux serve status`
Fragt den local-Listener, ob er antwortet — eine Zustandsdatei ist ein Hinweis,
ein antwortender Listener ist die Wahrheit — und gibt dann aus, was `serve.json`
sagt: PID, beide Endpunkt-URLs, das Programm, seine Größe und Bauzeit, ob die
Sperre gehalten wird und ob der Breakaway gegriffen hat. Die Token bleiben aus
dem Bericht draußen. Ein Dienst, den es nicht gibt, ist ein Zustand und kein
Fehlschlag: der Bericht sagt es, der Exit-Code ist `0`.

- **Exit-Codes**: `0` berichtet; `1` der Zustand war nicht lesbar; `2` ein
  unbekanntes Argument.

### `loomux serve stop [--force]`
Beendet den Dienst über seinen eigenen Endpunkt. `--force` tötet ihn über die
PID aus `serve.json`, wenn der Endpunkt nicht mehr antwortet. Erfolg ist still.

- **Exit-Codes**: `0` beendet, und ebenso, wenn nichts lief; `1` der Stopp ist
  gescheitert; `2` ein unbekanntes Argument.

### `loomux upgrade [--beta | --stable | --version <x.y.z>]`

Ein Update-Durchlauf von Hand; `serve` fährt denselben eine Minute nach
dem Start und danach alle 24 Stunden. Er wirkt nur auf das maschinenweite
Binary, `<Zustandsverzeichnis>/bin/loomux.exe`, wenn das das laufende Binary
ist. Ein Entwicklungs-Build (`0.0.0-dev`) dort wird durch das neueste Release
ersetzt; einer an jedem anderen Ort bleibt unberührt.

1. Listet die Releases über `gh release list` und nimmt die höchste Version
   im Kanal der Maschine: mit der Markierung `<Zustandsverzeichnis>/channel`
   Betas und stabile Releases, ohne sie nur stabile. Ein Binary aus der
   Zählung vor 1.0.0 nimmt ohne die Markierung die Releases dieser Zählung
   und die stabilen, nie eine neue Beta.
2. Lädt das Windows-Asset und `SHA256SUMS` über `gh release download`, prüft
   die Prüfsumme und die `--version` des neuen Binarys.
3. Stempelt die Datei mit der aktuellen Zeit und tauscht sie ein; das alte
   Binary kommt nach `loomux.old.exe` oder in den ersten freien nummerierten
   Platz. Die nächste Brücke ersetzt den laufenden `serve`.

Die Flags wählen das Release von Hand; höchstens eines zugleich:

- `--beta` nimmt das neueste Release beider Arten und setzt die Markierung
  `<Zustandsverzeichnis>/channel` (die Zeile `beta`), damit `serve` weiter
  Betas nimmt.
- `--stable` nimmt das neueste stabile Release und entfernt die Markierung. Solange es noch kein stabiles Release der neuen
  Zählung gibt, endet es mit „no release in channel stable“ (Exit 1).
- `--version <x.y.z>` nimmt genau diese Version (Präfix `v` ist erlaubt; eine
  Beta wie `1.1.0-beta.2` geht auch). Eine stabile Version entfernt die
  Markierung, eine Beta setzt sie. Ein Downgrade auf diesem Weg hält nicht:
  `serve` hebt das Binary binnen 24 Stunden wieder an.

Ein Problem allein mit der Markierung lässt den Durchlauf nicht scheitern: das
Binary liegt an seinem Platz, das Problem geht nach stderr, und der
Sitzungsstart nennt es erneut.

Schreibt `<Zustandsverzeichnis>/update.json` (`source` = `serve` | `cli`,
`checked_at`, `executable`, `running`, `result` = `current` | `updated` |
`skipped` | `failed`, `version`, `error`). Ein Durchlauf, der `update.lock`
belegt findet, tritt zurück und schreibt nichts. Ebenso wenig schreibt ein
Durchlauf von Hand, der ausgelassen wird; so bleibt der Eintrag vom letzten
Durchlauf von `serve` für den Sitzungsstart stehen.

| Exit | Bedeutung |
|---|---|
| 0 | `already current (vX)` oder `updated to vX` |
| 1 | der Durchlauf ist gescheitert, oder ein anderer läuft |
| 2 | ausgelassen: nicht Windows oder nicht das maschinenweite Binary (auch ein Entwicklungs-Build); oder ein unbekanntes Argument, zwei Flags zugleich oder eine Version, die keine ist |

### `loomux mcp [--channel local|cloud] [--root <verz>]`
Die stdio-Brücke, die ein MCP-Wirt startet. Sie bietet die zwölf Werkzeuge selbst
an — die Beschreibungen sind statisch, also sitzt nie ein kalter Dienst im
Handschlag des Wirts — und leitet jeden `tools/call` an die Adresse des Kanals
weiter, Name zu Name und Argumente zu Argumenten.

- **Vorgabekanal**: `local`, genau wie `loomux brain` zurückfällt. Der enge
  Kanal ist die einzige sichere Vorgabe.
- **Nur die Module des Projekts**: Die Brücke bietet die `brain_*`-Werkzeuge
  nur mit eingeschaltetem `[modules] brain` an und die `graph_*`-Werkzeuge
  nur mit `graph` (siehe
  [`[modules]`](configuration.md#modules-was-in-diesem-projekt-läuft)). Das
  Projekt ist das, das `--root` nennt, sonst die erste `.loomux/config.toml`
  oberhalb des Verzeichnisses, in dem der Wirt die Brücke gestartet hat;
  außerhalb jedes Projekts wird jedes Werkzeug angeboten. Ein `[modules]`,
  das der Leser ablehnt, beendet die Brücke vor dem Handschlag mit Exit `1`.
- **Eine Änderung von `[modules]` braucht einen Neustart**: Die Brücke liest
  es einmal beim Start, und der Wirt hält `tools/list` im Cache. Die Änderung
  wirkt, wenn der Wirt die Brücke neu startet.
- **Sie startet den Dienst selbst**: im Hintergrund, neben dem Handschlag. Ein
  aufgezeichneter Bau, der älter ist als der der Brücke, wird gestoppt und
  ersetzt (neuer gewinnt); ein gleich alter oder neuerer, dessen Sperre niemand
  hält, wird erneut gestartet.
- **Eine Wiederholung**: Ein Aufruf, dessen Sitzung gestorben ist — ein Commit
  hat das Binary neu gebaut und eine andere Brücke den Dienst ersetzt —, wird
  einmal neu verhandelt. Zwei Wiederholungen verdeckten einen echten Ausfall.
- **Auf stdout wird nie geschrieben**: stdout ist die MCP-Leitung des Wirts.
  Ablehnungen und Fehlschläge gehen nach stderr.
- **Exit-Codes**: `0` der Wirt hat aufgelegt, oder Strg+C; `1` die Brücke ist
  gescheitert; `2` ein unbekanntes Argument oder ein ungültiges `--channel`.

### Die zwölf Werkzeuge

| Werkzeug | Argumente |
|---|---|
| `brain_search` | `query` (Pflicht), `scope` → `all`, `profile` ∈ {`fast`, `full`, `keyword`} → `fast`, `n` → 10 |
| `brain_catalog` | `scope` → `all` |
| `brain_read` | `scope` und `relative` (beide Pflicht), `section` |
| `brain_neighbors` | `scope` und `relative` (beide Pflicht) |
| `brain_status` | keine |
| `graph_find_code` | `scope` und `query` (beide Pflicht), `limit` → 5, `full`, `in` |
| `graph_check_freshness` | `scope` (Pflicht) |
| `graph_file_api` | `scope` und `file_path` (beide Pflicht) |
| `graph_trace_calls` | `scope` und `symbol` (beide Pflicht), `direction` ∈ {`in`, `out`} → `in`, `depth` → 1 |
| `graph_find_all` | `scope` und `pattern` (beide Pflicht), `ignore_case`, `fixed` |
| `graph_repo_map` | `scope` (Pflicht), `max_dirs` → 16 |
| `graph_blast` | `scope` (Pflicht), `base`, `depth` → 1 |

`n` ist hier 10 und auf der Kommandozeile 5; das ist Parität mit der
Python-Referenz, die es genauso hält, und keine Unstimmigkeit. `limit` ist hier
5 und bei `loomux graph ask` 8, beides Grafts Werte.

Die sieben `graph_*`-Werkzeuge operieren über das Repository eines registrierten Bereichs:

- **`graph_find_code`** fügt den Quelltext an jedem Treffer immer ein; `full`
  nimmt den ganzen Span statt des gekappten Auszugs, und `in` verengt vor dem
  Scoring auf ein Pfadpräfix. Ein Auffrisch-Hinweis steht vor der Antwort. Ohne
  Graph ist der Aufruf ein Fehler, der auf `loomux graph build` verweist — eine
  Abfrage baut nie einen ersten Graphen.
- **`graph_check_freshness`** frischt nie auf und berichtet deshalb über den
  Graphen, wie er vorgefunden wurde. Drift und ein fehlender Graph sind Text,
  keine Fehler. `isError` kennzeichnet einen abgewiesenen Aufruf — einen
  fehlenden, unbekannten oder verborgenen Scope, bei `graph_find_code` auch
  eine fehlende Anfrage — und einen echten Lesefehler.
- **`graph_file_api`** gibt Definitionen und Typen für `file_path` aus dem
  gepufferten AST-Graphen ohne Funktionsrümpfe aus.
- **`graph_trace_calls`** verfolgt eingehende Aufrufer (`direction: in`) oder ausgehende
  Aufrufe (`direction: out`) von `symbol`, entweder direkt (`depth: 1`) oder
  transitiv (`depth: "all"`).
- **`graph_find_all`** führt eine symbol-gekoppelte Regex-Suche über indizierte Dateien
  durch, gruppiert nach umschließendem Symbol und gerankt nach Kanten-Kopplung (`inDegree`).
- **`graph_repo_map`** erzeugt eine token-budgetierte strukturelle Übersicht über
  Verzeichnis-Cluster, Hubs und Hotspots.
- **`graph_blast`** ist `loomux graph blast` in der Wurzel des Bereichs: `base`
  vergleicht `base...HEAD`, leer vergleicht den Working Tree mit `HEAD` oder,
  bei sauberem Baum, den letzten Commit; `depth` wie bei `graph_trace_calls`.
  Eine geänderte Datei oder ein Treffer unter den `never`-Globs wird gezählt,
  nicht genannt, und die Auffrisch-Hinweise fallen auf dem Cloud-Kanal weg.

**Sichtbarkeit:** Ein Bereich, dessen Manifest `[privacy] mode = "local_only"`
setzt, existiert auf dem cloud-Kanal nicht (`unknown scope`, wie bei
`brain_*`), und auf beiden Kanälen fallen Pfade unter den `[privacy] never`-Globs
des Manifests vor dem Scoring, aus Dateilisten und aus dem Driftbericht heraus — der auf dem
local-Kanal nennt, wie viele er weggelassen hat, auf dem cloud-Kanal nicht.
Auf dem cloud-Kanal gilt außerdem:
- `graph_file_api` auf eine Datei unter `never`-Globs meldet `NotFound` (`isError`).
- `graph_trace_calls` filtert Aufrufer und Aufgerufene in geschützten Pfaden aus und zählt sie in `hidden`.
- `graph_find_all` verweigert über die injizierte Lesefunktion den Zugriff auf geschützte Pfade (`os.ErrPermission`), zählt sie in `unreadable_files` und gibt keine Quelltextzeilen daraus zurück.
- `graph_repo_map` filtert geschützte Verzeichnisse und Knoten vor der Erstellung der Karte heraus.
- Keine Auffrisch-Hinweise gehen hinaus, weder vor der Antwort noch als Fortschritt: ein Hinweis zählt auch Dateien unter den `never`-Globs mit.
Jeder Lese- oder Abfragefehler wird dort zu dem festen Text
„the graph could not be read on this channel; ask on the local channel for
details", weil eine Fehlermeldung eine verborgene Datei oder einen lokalen
Pfad nennen kann. Ausgenommen ist ein fehlender Graph, der seinen eigenen Text
behält. Ein interner Fehler lautet dort „internal error; ask on the local
channel for details", ohne den Wert, den er trug.

### Die `.mcp.json` eines Wirts

`loomux init` schreibt sie als seinen Teil `mcp-json` (siehe [`loomux init`](#11-projekt-einrichten-loomux-init)),
nur wenn der Nutzerbereich (`~/.claude.json`) keinen Server `loomux` nennt,
und behält jeden anderen Server einer vorhandenen Datei:

```json
{ "mcpServers": { "loomux": { "command": "${LOCALAPPDATA}/loomux/bin/loomux.exe", "args": ["mcp", "--channel", "local"] } } }
```

Sie ruft das maschinenweite Binary an seinem festen Ort, nicht `loomux` über
den `PATH`. Dass Claude Code `${LOCALAPPDATA}` dort auflöst, ist noch nicht
gemessen; bis dahin kann ein Wirt `loomux` über den `PATH` von Hand nennen.

---

## 9. Entwickler-Prüftore (`loomux dev`)

### `loomux dev mutants <paket>... [--only <name>] [--family a1|a2|a3|a4] [--workers <n>]`
Mutiert die Go-Entscheidungen jedes Pakets und meldet, welche Mutanten seine Testsuite nicht bemerkt.

- **Familien**: `a1` die ganze `if`-Bedingung als `true` und als `false`; `a2` jeder Operand eines `&&` oder `||` auf oberster Ebene für sich; `a3` jeder Vergleichsoperator gekippt (`==`/`!=`, jede Ordnung gegen ihren Nachbarn), nicht in Kommentaren oder Zeichenketten; `a4` die Bedingung negiert. `for`-Bedingungen werden nie mutiert.
- **Mechanik**: Zuerst läuft die unveränderte Suite jedes Pakets (`-timeout 10m`), und ihre Dauer wird gemessen. Danach erreicht jeder Mutant `go test -overlay <json> -count=1 -failfast -timeout <grenze> ./<paket>/` über ein Overlay in einem temporären Verzeichnis; der Arbeitsbaum wird nie beschrieben. Die Grenze ist das Dreifache der Dauer der unveränderten Suite, auf ganze Sekunden aufgerundet, und mindestens 60 s, damit ein Mutant, den die Suite nicht bemerkt, nicht abgebrochen wird, weil die Suite langsam ist oder sich mehrere Läufe den Rechner teilen. Jedes Overlay-Verzeichnis wird nach seinem Lauf entfernt, auch nach einem Fehler oder Strg+C. `--workers` fährt so viele Läufe gleichzeitig (Vorgabe: die Hälfte der Prozessoren, mindestens 1). `--only` behält Dateien, deren Name den Text enthält.
- **Bericht**: je Paket zuerst eine Zeile mit seiner Grenze (`internal/cli: each mutant run is bounded at 2m55s (3 × the unchanged suite's 58.3s)`, oder `…, the floor (…)`, wo die 60 s greifen), dann je Mutant eine Zeile — `killed`, `timed out`, `SURVIVED` oder `no mutant` (kompiliert nicht oder ändert nichts) — dann die Summen und die Überlebenden. Ein Lauf, der seine Grenze reißt (die Geduld oder das eigene `-timeout` des Testbinärs), ist `timed out` und gilt als getötet; die Summen nennen, wie viele der Getöteten in ihre Grenze liefen, denn unter Last reißt auch eine Suite, die bestanden hätte, ihre Grenze.
- **Exit-Codes**: `0` nach einer vollständigen Runde, auch mit Überlebenden; `2` bei einem Usage-Fehler, einem Paket ohne Quelldateien oder einer Suite, die vor dem ersten Mutanten nicht grün ist; `1`, wenn ein Lauf nicht gestartet werden kann oder die Runde mit Strg+C abgebrochen wird.

### `loomux dev swap-binary [--dir <verz>]`
Tauscht das laufende `loomux.exe`-Binary atomar gegen `loomux.new.exe` aus (löst Windows Dateisperren-Konflikte). Das abgelöste bleibt als `loomux.old.exe` liegen, oder als erstes freies `loomux.old.<n>.exe` daneben, wenn ein Prozess aus einem früheren Tausch — ein `loomux serve` oder eine Brücke — diesen Namen noch hält; jeder Platz, dessen Prozess beendet ist, wird beim nächsten Tausch geräumt, es bleiben also höchstens 16 Generationen liegen. Zwei Fälle scheitern weiterhin, und beide lassen die Binaries dort, wo sie waren: alle 16 Plätze gleichzeitig gehalten, und ein `loomux.exe`, das etwas so hält, dass es sich gar nicht umbenennen lässt — ein laufendes `loomux.exe` ist dieser Halter nicht, denn Windows lässt ein laufendes Abbild umbenennen.

- **Flags**: `--dir <verz>` — Verzeichnis mit `loomux.new.exe` (Standard: `bin`).
- **Exit-Codes**: `0` nach dem Tausch; `1` wenn der Tausch scheitert (`loomux dev swap-binary: <grund>` auf `stderr`); `2` bei einem unbekannten Flag.

### `loomux dev bench <hooks|repos|search|compare|cases> [flags]`
Drei Messungen und zwei Helfer unter einer Gruppe. `loomux dev bench` allein nennt die Unterbefehle und endet mit `2`; ein unbekannter Unterbefehl ebenso. Die Gruppe ersetzt `dev bench-hooks` (jetzt `dev bench hooks`) und `dev bench` (jetzt `dev bench repos`).

- **Eine Berichtsform**: Mit `--out <verzeichnis>` schreibt jeder der drei messenden Unterbefehle (`hooks`, `repos`, `search`) `bench-<stempel>-<befehl>.md` und `.json` (`dev bench search`: `bench-<stempel>-<profil>`) in dieses vorhandene Verzeichnis, beide oder keine, und nie über eine vorhandene Datei, auch nicht über eine, die ein Lauf derselben Minute inzwischen geschrieben hat. Der Stempel ist UTC, `JJJJ-MM-TT-HHMM`. Liegt eine der beiden Dateien schon da, bricht der Lauf vor der ersten Messung ab. Das Markdown von `hooks` und `repos` beginnt mit Befehl, Stempel, System, Go-Version und loomux-Version; das von `search` mit Stempel, Profil, qmd, Modellen, dem qmd-Backbone, System, loomux, Suchweg, dem gemessenen Bereich, der Zahl der indizierten Dokumente und dem Fragensatz, dann im Korpusmodus mit dem Korpus und dem Hinweis, dass seine Zahlen nur Regression messen, und bei `fast` mit dem Hinweis, dass der Lauf rein vektoriell ist. Das JSON ist eine Hülle für alle drei: `schema` (`1`), `command` (`hooks`, `repos` oder `search`), `stamp`, `environment` (`os`, `arch`, `cpu`, `go`, `loomux`; `search` ergänzt `qmd`, `models`, `profile`, `port`, das `daemon` oder `cli` ist, und `backbone`: worauf ein qmd-Prozess rechnet, den dieser Lauf startet (siehe das Backbone von `brain search`), wo der Nutzer `QMD_LLAMA_GPU` oder `QMD_FORCE_CPU` gesetzt hat, der Wert von `QMD_LLAMA_GPU`, `cpu` unter einem nicht leeren `QMD_FORCE_CPU` und `default`, wenn beide leer sind, sonst das `[search] backbone` des Rechners (`cuda`, wenn nicht gesetzt); im Korpusmodus immer, weil die qmd-Kommandozeile darauf rechnet, und bei einem Lauf über den Suchdienst nur, wenn dieser Lauf den Daemon gestartet hat (er schrieb den Aufwärmhinweis); `unknown` bei einem Daemon, den er schon laufend vorfand und der das Backbone des Prozesses behält, der ihn gestartet hat), `timings[]` (`name`, `cold_ms`, `warm_ms[]`, `median_ms`, `min_ms`, `max_ms` und, wo sie gelten, `exit_codes`, `applicable`, `timed_out`), alle Zeiten in Millisekunden, und eine eigene `payload` je Befehl (`repos`: die geprüften Repositories; `search`: `question_set`, `corpus` (null außerhalb des Korpusmodus), `documents`, `questions[]` mit `id`, `sort`, `rank` (null, wenn nicht gefunden), `hit` und `elapsed_ms`, `findings[]` und `latency` (gemessenes Dokument und Anfrage, null ohne `--latency`); `hooks`: keine). `compare` schreibt dieselben zwei Dateien, sein JSON ist aber der Vergleich selbst und nicht diese Hülle; `cases` schreibt keinen Bericht, nur eine Falldatei und ihre Nutzlasten.

#### `loomux dev bench hooks <falldatei> [-n <n>] [--out <dir>]`
Misst die Hook-Befehle einer Falldatei: jeden Fall einmal kalt, dann `-n`-mal warm, und schreibt eine Markdown-Tabelle (Fall, kalt, warmer Median, Min, Max, Exit-Codes des letzten Laufs) auf stdout; mit `--out` zusätzlich die beiden Berichtsdateien. Die Falldatei ist eine JSON-Liste von Fällen, je mit `name`, `dir`, `stdin` (eine Datei, die den Schritten zugeführt wird), `mode` (`single`, die Vorgabe, und `seq` fahren die Schritte nacheinander in einer gemessenen Spanne; `par` startet sie zugleich) und `steps[]` (`argv`). Die Falldatei darf vor oder hinter den Flags stehen. Die Falldateien der Chronik in `docs/de/benchmarks.md` liegen unter `testdata/bench/`.

- **Flags**:
  - `-n <n>`: Warme Läufe je Fall, nach einem kalten (Standard: `20`, mindestens `1`).
  - `--out <dir>`: Verzeichnis für den Markdown- und JSON-Bericht.
- **Exit-Codes**: `0` nach einer Messung, gleich, womit die gemessenen Befehle enden; `2` bei einem unbekannten Flag, ohne Falldatei, mit einem zweiten Argument oder mit `-n` unter 1; `1`, wenn die Falldatei nicht gelesen oder dekodiert werden kann, ein Fall ungültig ist, ein Schritt nicht gestartet werden kann oder eine Berichtsdatei schon da ist oder nicht geschrieben werden kann.

#### `loomux dev bench repos [--dir <dir>] [--corpus <datei>] [--languages <n>] [--tier <kategorie>] [--warm <n>] [--cache-dir <dir>] [--timeout <d>] [--component-timeout <d>] [--out <dir>] [--save] [--report-dir <dir>]`
Führt Latenz-Benchmarks und normalisierte Lücken-Audits (Gap Analysis) für ein Einzel-Repository oder das gesamte Open-Source-Matrix-Korpus durch (1x kalt + Nx warmer Median, Min, Max).

- **Wie die Hooks gemessen werden**: Jeder Hook bekommt eine Claude-Code-Nutzlast für einen Edit an einer Beispieldatei der Hauptsprache des Repositorys, `post-tool-use` fährt also seine echten Lanes. Die Status-Spalte nennt die Exit-Codes aller Läufe.
- **Einzel-Repository-Modus** (Standard): Misst `pre-tool-use`, `post-tool-use` und `graph build` (bei Go-Projekten), vergleicht mit bestehenden Claude-Hooks (Speedup) und prüft Lücken zwischen nativen Werkzeugen und Loomux-Lanes.
- **Korpus-Modus** (`--corpus <pfad>`): Klont und benchmarkt die Top-N Open-Source-Projekte über Sprachen und Frameworks hinweg und liefert einen aggregierten Performance- und Lückenbericht. Ein Repository, das nicht gemessen werden konnte, wird auf stderr als übersprungen genannt.
- **Ausgabe**: Ohne `--out` geht der Markdown-Bericht auf stdout; mit `--out` nur in die beiden Berichtsdateien.
- **Flags**:
  - `--dir <pfad>`: Ziel-Repository (Standard: `.`).
  - `--corpus <pfad>`: Pfad zur Open-Source-Matrix-Markdown-Datei.
  - `--languages <n>`: Anzahl der Sprachen im Korpus (Standard: `5`, im Korpus-Modus mindestens `1`).
  - `--tier <kategorie>`: Filter für Sterne-Kategorie (Standard: `"Sehr viel"`).
  - `--warm <n>`: Anzahl warmer Messläufe für die Median-Berechnung (Standard: `3`, mindestens `1`).
  - `--cache-dir <pfad>`: Verzeichnis für geklonte Repositories (Standard: `.cache/benchcorpus`).
  - `--timeout <d>`: Frist für die Messung eines Repositorys (Standard: `5m`).
  - `--component-timeout <d>`: Frist je gemessenem Befehl; ein Befehl darüber wird beendet und als `timeout` gemeldet (Standard: `60s`).
  - `--out <dir>`: Verzeichnis für den Markdown- und JSON-Bericht (Standard: Markdown auf stdout).
  - `--save`: Speichert Benchmark-Berichte automatisch in Sprachunterordnern (`docs/{en,de}/benchmarks/<sprache>/<slug>.md`) und aktualisiert die zentrale Gesamt-Matrix (`docs/{en,de}/benchmarks/matrix.md`).
  - `--report-dir <pfad>`: Dokumentations-Stammverzeichnis für gespeicherte Berichte (Standard: `docs`).
- **Exit-Codes**: `0` nach einer Messung; `2` bei einem Usage-Fehler (`--warm` unter 1, `--languages` unter 1 im Korpus-Modus); `1`, wenn die Korpusdatei nicht gelesen werden kann, eine Messung scheitert oder ein Bericht nicht geschrieben oder gesichert werden kann.

#### `loomux dev bench search [--scope <scope>|all] [--profile keyword|fast|full] [--channel local|cloud] [--out <dir>] [--questions <datei>] [--corpus v1|<dir>] [--latency] [--latency-query <q>] [--repeat <n>]`
Misst, wie gut die Suche eine Notiz findet: für jede Frage eines Fragensatzes den Rang der erwarteten Quelle (Treffer bei Rang ≤ 3) und die Zeit der Antwort; mit `--latency` zusätzlich die Latenz von Katalog, Lesen und den drei Profilen. Der Bericht wird immer als die beiden Dateien geschrieben; danach geht das Markdown auf stdout.

- **Alltagsmodus** (Standard): fragt den registrierten Bereich über den Suchdienst (`environment.port` ist `daemon`), so wie loomux im Betrieb sucht. Der Fragensatz ist `<out>/questions.yaml`; `--out` ist standardmäßig `<bereich>/98 Messung` und muss vorhanden sein. Wird mehr als ein Bereich gemessen (`--scope all` über ein Register mit mehreren), gibt es keine Vorgabe, `--out` ist dann Pflicht. Ohne `--scope` findet `knowledge` nur `--out` und damit den Fragensatz; gemessen werden die Bereiche, in denen seine `expect`-Pfade liegen: der eine Bereich, wenn alle in einem liegen, `all`, wenn sie in mehreren liegen. Ein `expect` in keinem registrierten Bereich bricht den Lauf mit Exit 1 ab und nennt Frage und Pfad; dann den Bereich mit `--scope` nennen. Ein angegebenes `--scope`, auch `knowledge`, wird immer so gemessen.
- **Korpusmodus** (`--corpus`): misst einen Korpusstand. `v1` meint den eingecheckten `testdata/bench/search/v1` (100 Notizen, 50 Fragen, Baseline 43/50 bei `fast`) und braucht einen loomux-Checkout; jeder andere Wert ist das Verzeichnis eines Stands. Der Stand wird zuerst geprüft, dann in einem Wegwerf-Zustand und einem eigenen qmd-Index `loomux-bench-<zufall>` angelegt und über die qmd-Befehlszeile befragt (`environment.port` ist `cli`); die geteilte `index.yml` wird nie angefasst. Der Lauf übernimmt den Block `models:` aus `index.yml` in seinen eigenen Index, hält eine Sperre je Indexname und räumt Index und Zustand auf jedem Rückweg weg, auch nach einem Fehler. Strg+C räumt nicht auf; einen `loomux-bench-*`-Index, den ein abgebrochener oder abgestürzter Lauf liegen ließ und den niemand hält, räumt der nächste Korpuslauf mit demselben Zustandsverzeichnis. Scheitert `qmd update`, bricht der Lauf vor `qmd embed` ab. Meldet qmd nach dem Einbetten noch Dokumente ohne Vektoren, bricht ein `fast`- oder `full`-Lauf mit Exit 1 ab und nennt ihre Zahl, weil der Bericht einen nicht eingebetteten Index messen würde; `keyword` liest keine Vektoren und prüft das nicht. Seine Latenz (Sekunden je Aufruf, weil die Befehlszeile die Modelle jedes Mal lädt) ist mit dem Alltagsmodus nicht vergleichbar.
- **Fragensatz**: eine YAML-Liste von Einträgen mit `id`, `sort` (`exakt`, `umschreibung`, `gemischt`, `sprachuebergreifend`), `query`, `expect` (die Notiz, relativ zur Fragendatei), `beleg` (eine Stelle dieser Notiz) und optional `hinweis`. Alle Probleme werden gesammelt vor der ersten Anfrage gemeldet.
- **Während des Laufs**: Hält die Suchmaschine kein Dokument der gemessenen Bereiche, bricht der Lauf vor der ersten Frage ab. Was die Suchkette zu einer Anfrage vermerkt (etwa zweimal hintereinander eine leere Antwort), steht als Befund im Bericht. Mit `--latency` wird die Latenzanfrage zuerst einmal ungezählt geprobt; findet sie nichts, bricht der Lauf ab.
- **Flags**:
  - `--scope <scope>`: Der zu messende Bereich, oder `all` (Standard: die Bereiche, in denen die `expect`-Pfade des Fragensatzes liegen; `knowledge` findet nur den Fragensatz).
  - `--profile <p>`: `keyword`, `fast` oder `full` (Standard: `fast`).
  - `--channel <c>`: `local` oder `cloud` (Standard: `local`).
  - `--out <dir>`: Verzeichnis für den Bericht (Standard: `<bereich>/98 Messung`).
  - `--questions <datei>`: Fragensatz (Standard: `<out>/questions.yaml`).
  - `--corpus <v1|dir>`: `v1` für den eingecheckten Korpus, oder das Verzeichnis eines Stands. Verweigert `--scope`, `--questions` und `--latency` und verlangt `--out`.
  - `--latency`: Misst zusätzlich Katalog, Lesen und die drei Profile, nach dem Qualitätsdurchgang (die Kette ist dann schon warm).
  - `--latency-query <q>`: Die Anfrage der Latenzsuchen (Standard: `latenz`).
  - `--repeat <n>`: Warme Läufe je gemessener Operation, nach einem kalten (Standard: `10`, mindestens `1`).
- **Exit-Codes**: `0` nach einer Messung; `2` bei einem unbekannten Flag, einem überzähligen Argument, einem unbekannten Profil oder Kanal oder `--repeat` unter 1; `1` mit einer Zeile `error: <problem>` je Problem auf stderr für alles andere (eine verweigerte Flag-Kombination, `--corpus v1` außerhalb eines Checkouts, ein fehlerhafter Fragensatz oder Stand, ein fehlendes Verzeichnis oder eine schon vorhandene Berichtsdatei, ein leerer Index, ein Fehler der Suchmaschine).

#### `loomux dev bench compare --before <bericht> --after <bericht> [--title <t>] [--lang de|en] [--out <dir>]`
Stellt zwei Berichte von `dev bench hooks` nebeneinander: den früheren Lauf (`--before`) gegen den späteren (`--after`), mit dem Faktor dazwischen (2 heißt doppelt so schnell, 0,5 halb so schnell). Die Fälle werden nach Namen gepaart, die beiden Falldateien müssen also dasselbe gleich benennen. Ein Fall, der auf einer Seite nicht gilt (`applicable` falsch), fällt auf dieser Seite weg. Das Markdown geht auf stdout und beginnt mit `# <titel>` und einer Zusammenfassungszeile (verglichen, schneller, langsamer, unklar, neu, weggefallen), dann drei Listen: die Fälle, die beide Läufe gemessen haben (kalt, warmer Mittelwert, Median, die Faktoren und die Exit-Codes beider), die Fälle nur des späteren Laufs (neu) und die nur des früheren (weggefallen). Mit `--out` schreibt der Befehl wie `dev bench hooks` `bench-<stempel>-compare.md` und `.json` (das JSON ist der Vergleich selbst, eingerückt), beide oder keine, nie über eine vorhandene Datei, und er prüft die beiden Namen, bevor er einen Bericht liest.

- **Flags**:
  - `--before <datei>`, `--after <datei>`: Die JSON-Berichte, die verglichen werden (Pflicht). Ein Bericht mit einem anderen `schema`, als dieses loomux schreibt, wird abgelehnt.
  - `--title <t>`: Die Überschrift, für den Namen eines Projekts oder ein anonymisiertes Beispiel (Standard: `loomux dev bench compare`).
  - `--lang <l>`: `de` oder `en`, die Sprache der Überschriften (Standard: `de`).
  - `--out <dir>`: Verzeichnis für Markdown und JSON; es muss vorhanden sein.
- **Exit-Codes**: `0` nach einem Vergleich; `2` bei einem unbekannten Flag, einem überzähligen Argument, fehlendem `--before` oder `--after` oder einer unbekannten Sprache; `1`, wenn ein Bericht nicht gelesen oder dekodiert werden kann, ein anderes Schema hat, einen Fall zweimal nennt, oder wenn eine Berichtsdatei schon da ist oder nicht geschrieben werden kann.

#### `loomux dev bench cases [--settings <datei>] --root <dir> --file <markdown> --out <dir> [--extras <datei>]`
Baut die Falldatei, die `dev bench hooks` misst, aus den Hooks der Claude-`settings.json` eines Projekts, damit das Inventar dessen, was bei einem Edit läuft, aus dem Projekt kommt. Je Ereignis mit einem Befehl, der für einen Edit an `--file` gilt, entsteht ein Fall (`SessionStart`, `PreToolUse`, `PostToolUse`, `SubagentStart`, `SubagentStop`, `Stop`, in dieser Reihenfolge; die Werkzeug-Ereignisse nur für einen Matcher, der auf `Edit` passt), benannt nach dem Ereignis (`PreToolUse (Edit on README.md)` bei den Werkzeug-Ereignissen), damit die alte und die neue Konfiguration für `dev bench compare` dieselben Namen liefern. Die Befehle eines Ereignisses laufen zusammen (`par`), ein einzelner allein. `${CLAUDE_PROJECT_DIR}` in einem Befehl wird durch `--root` ersetzt, das auch das Verzeichnis ist, in dem die Fälle laufen; jedes andere `${NAME}` durch die Umgebungsvariable dieses Namens (`${LOCALAPPDATA}` in den Hook-Befehlen, die loomux installiert). Der Befehl läuft durch keine Shell: Nur die Form mit geschweiften Klammern wird ersetzt, und zwar überall, wo sie steht, auch in einfachen Anführungszeichen, wo eine Shell sie stehen ließe; danach wird die Zeile an Leerzeichen und Anführungszeichen in Argumente geschnitten, und ein Operator, eine Umleitung oder ein Glob bleibt der Text, der er ist. Eine nicht gesetzte Variable ist ein Fehler, der sie nennt (die Shell machte daraus nichts und ließe einen kaputten Pfad zurück); eine auf die leere Zeichenkette gesetzte wird zu ihr. Der Befehl gibt nichts aus; er schreibt `cases.json` (eingerückt) und je Fall eine Nutzlastdatei `payload-<ereignis>.json` mit der Eingabe, die ein Host diesem Hook übergibt (bei den Werkzeug-Ereignissen ein Edit an `--file`). Jede Nutzlast trägt `session_id` `loomux-bench` und einen `transcript_path` auf `<root>/.loomux-bench-no-transcript.jsonl`, eine Datei, die es nicht gibt; die Subagent-Ereignisse tragen zusätzlich `agent_id` `loomux-bench-agent` und `agent_type` `general-purpose`. Die Hooks, die Zustand je Sitzung führen, weisen eine Nutzlast ohne diese Kennungen ab; mit ihnen tun sie ihre echte Arbeit und können darum unter dieser Sitzungskennung Zustand im Projekt hinterlassen. `--out` muss vorhanden sein und keine dieser Dateien darf darin liegen; die Dateien werden alle oder keine geschrieben, und ein Schreibfehler nimmt die schon geschriebenen zurück.

`--extras` nennt eine JSON-Datei mit weiteren Fällen (dieselbe Form wie `cases.json`), für ein Ziel ohne `settings.json` oder mit Messungen über seine Hooks hinaus. Im Text dieser Datei wird `{{ROOT}}` durch `--root` und `{{OUT}}` durch `--out` ersetzt (absolut, ohne abschließenden Schrägstrich, in Schrägstrichform), bevor sie gelesen wird, beides so, wie es in einer JSON-Zeichenkette steht; ein Pfad mit Backslash macht die Datei also nicht ungültig. Die Zusatzfälle folgen denen der Konfiguration; ein Fallname, der zweimal vorkommt, zwischen beiden oder unter den Zusatzfällen, ist ein Fehler. `--settings` darf fehlen, wenn `--extras` gegeben ist; dann enthält `cases.json` nur die Zusatzfälle, und es wird keine Nutzlastdatei geschrieben.

- **Flags**:
  - `--settings <datei>`: Die Claude-`settings.json`, aus der die Hooks gelesen werden.
  - `--root <dir>`: Das Projektverzeichnis (Pflicht).
  - `--file <markdown>`: Die Datei, die der Beispiel-Edit berührt (Pflicht).
  - `--out <dir>`: Das vorhandene Verzeichnis für `cases.json` und die Nutzlasten (Pflicht). Ein relatives wird gegen das Arbeitsverzeichnis aufgelöst: Die Fälle nennen ihre Nutzlasten mit absolutem Pfad in Schrägstrichform, sodass der Messlauf sie findet, wo immer er startet.
  - `--extras <datei>`: Weitere Fälle, mit `{{ROOT}}` und `{{OUT}}`.
- **Exit-Codes**: `0` nach dem Schreiben; `2` bei einem unbekannten Flag, einem überzähligen Argument, fehlendem `--root`, `--file` oder `--out` oder wenn weder `--settings` noch `--extras` gegeben ist; `1`, wenn eine Datei nicht gelesen oder dekodiert werden kann, ein Matcher kein gültiger Ausdruck ist, ein Befehl sich nicht zerlegen lässt, ein Befehl ein nicht gesetztes `${NAME}` benutzt, kein Hook eines Ereignisses für einen Edit gilt, ein Fallname zweimal vorkommt, `--out` kein Verzeichnis ist oder eine der zu schreibenden Dateien schon da ist oder nicht geschrieben werden kann.

### `loomux dev release <next-version|next-beta|parse-body|changelog-insert|build> [flags]`
Die Release-Regeln hinter `.github/workflows/release.yml` und der Prüfung von `release-pr`. Ohne Unterbefehl oder mit einem unbekannten endet es mit `2`. Jeder Fehler wird als `loomux dev release <unterbefehl>: <grund>` auf `stderr` gemeldet.

- **`next-version --bump major|minor|patch [--tags <datei>]`**: liest je Zeile einen Tag (Standard `-`, stdin) und gibt die nächste Version aus. Exit `0`; `2` bei einem unbekannten Flag, einer unlesbaren Tag-Datei oder einem ungültigen Bump.
- **`next-beta --bump major|minor|patch [--tags <datei>]`**: liest die Tags wie `next-version` und gibt `X.Y.Z-beta.N` für die Version aus, die `next-version` schneiden würde; `N` liegt eins über der höchsten Beta dieser Version (`1` ohne eine). Exit `0`; `2` bei einem unbekannten Flag, einer unlesbaren Tag-Datei oder einem ungültigen Bump.
- **`parse-body [--labels <a,b>] [--body <datei>] [--commits <datei>]`**: prüft das Release-Label und den Rumpf des Pull Requests (Standard `-`, stdin) und, mit `--commits` (ein JSON-Array der Commit-Nachrichten), dass kein Commit ein höheres Label verlangt. Gibt den gelesenen Rumpf als JSON auf `stdout` aus. Exit `0`; `1` mit einer Zeile je Problem; `2` bei einem unbekannten Flag oder einer unlesbaren Rumpf- oder Commit-Datei.
- **`changelog-insert --version <v> --date <JJJJ-MM-TT> --link <url> [--notes <datei>] [--file <pfad>]`**: fügt den Changelog-Block (Standard `-`, stdin) als Abschnitt des Releases in `--file` ein (Standard `CHANGELOG.md`, fehlt sie, wird sie angelegt). `--version` steht ohne `v`. Exit `0`; `1`, wenn die Version schon im Changelog steht; `2`, wenn ein Pflicht-Flag fehlt oder die Datei sich nicht lesen oder schreiben lässt.
- **`build --version <v> [--channel <name>] [--out <verz>]`**: baut die Release-Binaries für alle Zielplattformen nach `--out` (Standard `dist`) und nennt jede Datei auf `stdout`. Exit `0`; `1`, wenn ein Bau scheitert; `2` bei einem unbekannten Flag oder ohne `--version`.

### `loomux dev record-case --exe <altes-binary>|--argv <programm> --cmd <zeile> --world <verz> --out <verz> [flags]`
Zeichnet einen Fall eines alten Werkzeugs unter `testdata/cases/` auf: legt `--world` an, fährt die Befehlszeile `--cmd` (mit `{{WORLD}}` für das angelegte Verzeichnis) und schreibt, was es beobachtet hat, in das Fallverzeichnis `--out`.

- **Flags**:
  - `--exe <pfad>`: Das alte Binary; `--argv <programm und argumente>` setzt stattdessen ein Programm und seine führenden Argumente an die Stelle des ersten Worts des Befehls. Die beiden schließen einander aus.
  - `--env KEY=VALUE`: Umgebung des aufgezeichneten Prozesses, `{{WORLD}}` erlaubt; wiederholbar.
  - `--path-prepend <verz>`: Verzeichnis, das dem `PATH` des aufgezeichneten Prozesses vorangestellt wird.
  - `--stdin <datei>`: Datei mit der Nutzlast.
  - `--notes <text>`: Text für `notes.md`.
  - `--compare <art>`: Leer (die Daten vergleichen) oder `message`.
  - `--git-after`: Hält den Commit, den der Lauf gemacht hat, in `git.after` des Repositorys der Git-Welt fest.
- **Exit-Codes**: `0` nach der Aufzeichnung; `1`, wenn sie scheitert; `2` bei einem unbekannten Flag, `--exe` zusammen mit `--argv` oder einem fehlenden Pflicht-Flag.

### `loomux dev record-mcp-case --argv <programm> --tool <name> --world <verz> --out <verz> [flags]`
Zeichnet einen Aufruf der MCP-Front der Referenz als Fall auf: einen Werkzeugaufruf und sein `CallToolResult`, keine Befehlszeile.

- **Flags**:
  - `--argv <programm und argumente>`: Das Programm der Referenz und seine führenden Argumente.
  - `--tool <name>`: Das aufzurufende Werkzeug; `--arguments <json>` seine Argumente als JSON-Objekt, `{{WORLD}}` erlaubt.
  - `--channel <name>`: Der Kanal, den der Fall aufzeichnet.
  - `--env KEY=VALUE`, `--path-prepend <verz>`, `--notes <text>`: Wie bei `record-case`.
  - `--compare <art>`: Leer (den Text vergleichen) oder `outcome`.
- **Exit-Codes**: `0` nach der Aufzeichnung; `1`, wenn sie scheitert; `2` bei einem unbekannten Flag, einem fehlenden Pflicht-Flag oder einem anderen `--compare`.

### `loomux dev import-cases --map <datei> --from <verz> --to <verz> [--mcp] [--merge-fixture <datei>]`
Übersetzt aufgezeichnete Fälle aus `--from` nach `--to` nach den `[[command]]`-Regeln (mit `--mcp`, für Aufzeichnungen von MCP-Aufrufen, den `[[tool]]`-Regeln) der TOML-Datei `--map`. Mit `--mcp` ersetzen `[[result]]`-Regeln `{from, to}` jedes Vorkommen von `from` durch `to` in jedem aufgezeichneten `result`, auf den Bytes, wie aufgezeichnet, so wie `[[stdout]]`-Regeln in der Ausgabe eines Befehls. `--merge-fixture` nennt eine JSON-Datei mit zwei Listen und führt sie in die `faketool.json` jeder übersetzten Welt ein, hinter dem, was die Welt aufgezeichnet hat: zuerst die `answers`, wie sie dastehen, dann je Eintrag `{"prefix": …, "as": …}` von `same`, dessen `as` die Welt aufgezeichnet hat, eine Kopie, nämlich diese aufgezeichnete Antwort unter dem neuen `prefix` (von zwei Aufzeichnungen mit diesem Präfix die spätere, also die, die die Fixture gibt). Eine Befehlszeile, die nur loomux stellt, bekommt so eine feste Antwort oder die Antwort, die die Welt der alten Befehlszeile gab: Eine Welt, die ein scheiterndes `uv run pytest` aufgezeichnet hat, scheitert bei `uv run --with pytest pytest` genauso. Eine Welt, die `as` nicht aufgezeichnet hat, bekommt keine Kopie, und eine ohne Fixture bekommt eine.

- **Exit-Codes**: `0` nach dem Import; `1`, wenn sich die Zuordnung nicht dekodieren lässt oder Import oder Zusammenführung scheitern; `2` bei einem unbekannten Flag oder einem fehlenden Pflicht-Flag.

### `loomux dev fake-ollama --fixture <datei> [--addr <host:port>] [--log <datei>]`
Beantwortet jede Anfrage an einen Ollama-Endpunkt mit der einen Antwort der JSON-Datei `--fixture`, bis Strg+C. Es lauscht auf `--addr` (Standard `127.0.0.1:11435`) und hängt die Anfragezeilen an `--log` an, ohne `--log` an `stderr`. Eine Anfrage, die ihre Antwort mit `num_predict` begrenzt (das Vorwärmen von loomux, das die Referenz nie schickt), endet mit `num_predict=<n>`.

- **Exit-Codes**: `0` nach Strg+C; `1`, wenn sich Fixture oder Log nicht öffnen lassen oder die Adresse nicht bedient werden kann; `2` bei einem unbekannten Flag oder ohne `--fixture`.

### `loomux dev notices [--out <datei>]`
Schreibt `NOTICE.md` (Vorgabe `internal/notices/NOTICE.md`) aus dem, was das Binary linkt, das aus dem Checkout im Arbeitsverzeichnis gebaut wird: die Lizenz der Standardbibliothek von Go, jedes Moduls und jeder tree-sitter-Grammatik, deren Paket importiert ist, wörtlich, dazu der Hinweis der eingebetteten Worthäufigkeitstabelle. Den Build-Graphen fragt es beim `go`-Befehl ohne cgo ab, wie das Release baut, und weist eine copyleft-Grammatik ab, deren Bedingungen das ganze Binary bänden. Ein Test hält die eingecheckte Datei an dem, was dies erzeugt, damit keine neue Abhängigkeit ohne ihren Hinweis ausgeliefert wird; das Release schreibt denselben Text neben die Binaries und in `SHA256SUMS`.

- **Ausgabe**: der geschriebene Pfad auf `stdout`.
- **Exit-Codes**: `0` bei Erfolg; `1`, wenn `go` scheitert, eine Lizenzdatei sich nicht lesen lässt, eine Grammatik copyleft ist oder die Datei sich nicht schreiben lässt; `2` bei einem unbekannten Flag.

### `loomux dev record-poppler --exe <pdftotext> --dir <verz> --out <datei>`
Zeichnet Poppler für den Golden-Test von `convert` auf: startet einmal `<pdftotext> -v` und für jede `*.pdf` in `--dir` `<pdftotext> -layout -enc UTF-8 -eol unix <name> -`, aus diesem Verzeichnis und mit dem bloßen Namen, wie `convert` fragt, und schreibt die Antworten (Exit-Code und Ausgabe; bei `-v` samt `stderr` in der Ausgabe) als Fixture von `internal/dev/faketool` nach `--out`. Ein Mensch ruft es einmal mit installiertem Poppler auf; die Fixture hat keinen Platz für `stderr`, darum gehen Exit-Code, Größe und `stderr` jeder PDF für die Paritätsnotizen auf `stdout`.

- **Exit-Codes**: `0` bei Erfolg; `1`, wenn sich `--dir` nicht lesen oder die Fixture sich nicht schreiben lässt; `2` bei einem unbekannten Flag oder fehlendem `--exe`, `--dir` oder `--out`.

---

## 10. Konfiguration (`loomux config`)

Zeigt jeden Schlüssel von `.loomux/config.toml` mit der Herkunft seines
Werts und ändert einen Schlüssel nach dem anderen als Zeilenänderung, die
jeden Kommentar und jede nicht berührte Zeile erhält.

```bash
loomux config                                   # die interaktive Form
loomux config list [--json]
loomux config get <schlüssel>
loomux config set <schlüssel> <wert> [--yes | --propose]
loomux config unset <schlüssel> [--yes | --propose]
loomux config proposals [--json]
loomux config apply <id>|--all [--yes]
loomux config reject <id>|--all
# jede Form nimmt auch --root <verz> oder --global, vor oder hinter dem Unterbefehl
```

- **Der Wächter verweigert einem Agenten** jede Form außer `list`, `get`,
  `proposals`, einem alleinstehenden `--help` oder `-h` und `set` oder
  `unset` mit `--propose`: `set` und `unset` ohne es, `apply`, `reject` und
  die interaktive Form (siehe [`hook pre-tool-use`](#loomux-hook-pre-tool-use)).
  Ein Mensch führt sie aus.
- **Für Agenten**: Ein Agent ändert selbst nichts. Er schlägt eine Änderung
  mit `loomux config set <schlüssel> <wert> --propose` oder
  `loomux config unset <schlüssel> --propose` vor; sie durchläuft jede
  Prüfung von `set` und wird abgelegt statt geschrieben. Der Mensch sieht sie
  mit `loomux config proposals` durch und wendet sie mit
  `loomux config apply <id>` an (oder verwirft sie mit
  `loomux config reject <id>`). `config list` nennt, wie viele Vorschläge
  offen sind.
- **Flags** (vor oder hinter dem Unterbefehl; ein Flag, das der Unterbefehl nicht nimmt, ist ein Bedienfehler):
  - `--root <verz>` — das Projekt; leer wird es vom Arbeitsverzeichnis aus
    nach oben gesucht.
  - `--global` — stattdessen die rechnerweite `config.toml` im
    Zustandsverzeichnis (`LOOMUX_STATE_DIR`, Vorgabe `%LOCALAPPDATA%\loomux`).
    Ihre Schlüssel sind die des lokalen Modells: `model.enabled` (Vorgabe
    `false`), `model.endpoint` (Vorgabe `http://127.0.0.1:11434`, nur
    Loopback), `model.name` (das Ollama-Modell), `model.temperature` (eine
    Zahl von 0 bis 2, Vorgabe `0.0`) und `model.roles` (eine Tabelle, von
    Hand bearbeitet; einmal gesetzt, ist jede Rolle aus, die sie nicht
    nennt), dazu `search.backbone` der Suchmaschine (`cuda`, `vulkan` oder
    `cpu`, Vorgabe `cuda`; siehe `brain search`), den es nur hier gibt. Ein
    neuer Text wird vom Leser von `[model]`, von der Loopback-Wache des
    Clients und vom Leser von `[search]` geprüft: ein Endpunkt außerhalb des
    Loopbacks oder ein anderes Backbone wird verweigert (Exit `1`, mit Datei
    und Schlüssel), und die Datei bleibt, wie sie war. Nach dem Schreiben von
    `search.backbone` geben `set`, `unset` und `apply` eine Zeile mehr aus:
    ein laufender qmd-Daemon behält sein Backbone, bis sein Prozess endet;
    also den Prozess beenden, der auf Port 8765 lauscht (siehe das Backbone
    von `brain search`); die nächste Suche startet ihn mit dem neuen
    Backbone. Eine Datei,
    die kein TOML ist, wird mit ihrem eigenen Pfad genannt. Die
    `.loomux/config.toml` eines Bereichs kennt nur `model.enabled` und
    `model.roles` und kann nur abschalten oder einengen. `--global`
    zusammen mit `--root` ist ein Bedienfehler.
  - `--yes` — `set`, `unset` und `apply` schreiben, ohne zu fragen.
  - `--propose` — `set` und `unset` legen einen Vorschlag ab, statt zu
    schreiben; zusammen mit `--yes` ist es ein Bedienfehler.
  - `--all` — `apply` und `reject` wirken auf jeden offenen Vorschlag statt
    auf eine `<id>`; eine `<id>` zusammen mit `--all` ist ein Bedienfehler.
  - `--json` — `list` gibt ein JSON-Array aus `{key, module, value, origin,
    count}` aus; `proposals` ein JSON-Array der Vorschläge (`[]`, wenn keiner
    offen ist).
- **Schlüssel** heißen `<abschnitt>.<name>` (`commit.threshold`,
  `modules.graph`, `verify.timeout`), gruppiert nach Modul: `base`, `hooks`,
  `brain`. Aufgeführt ist jeder Schlüssel, den die Leser der Datei annehmen,
  und kein anderer, außer den Tabellen je Stack `[verify.<stack>.<art>]`
  (siehe `list`).
- **Benannte Schlüssel** tragen einen Namen, den das Projekt wählt:
  `agent.roles.<rolle>`, `agent.models.<name>.provider` und
  `agent.models.<name>.model` (siehe
  [`[agent]`](configuration.md#agent-modelle-für-die-rollen-eines-flows)).
  `list` zeigt eine Zeile je Name, den die Datei hält, und die Familie mit `*`
  an der Stelle des Namens als ungesetzte Zeile, solange sie keinen hält;
  `set`, `unset` und `get` nehmen den Schlüssel mit eingesetztem Namen, nie
  den `*`.
- **Herkunft**: `set` (in der Datei), `default` (die Vorgabe des Lesers, als
  Wert gezeigt), `preset` (`verify.profiles`, von den Presets gefüllt; gezeigt
  wird der eingebaute Wert) und `unset` (kein Wert und keine Vorgabe).

### `loomux config list [--json]`
Eine Zeile je Schlüssel: Modul, Schlüssel, Wert, Herkunft. Eine Liste von
Tabellen (`commit.allow`, `policy.paths.rules`, `policy.commands.rules`)
zeigt statt eines Werts die Zahl ihrer Einträge. Ein Wert über 60 Zeichen
wird dort abgeschnitten und endet auf `…`; `get` und `--json` geben ihn
ganz. Sind Vorschläge offen,
folgt eine letzte Zeile `N proposals open — loomux config proposals`
(`1 proposal open …` bei einem);
`--json` gibt nur die Zeilen aus. Die Tabellen je Stack,
`[verify.<stack>.<art>]`, sind keine Schlüssel: Sie werden von Hand geändert,
und `loomux check precommit --show` zeigt, was sie zusammen mit den Presets
ergeben.

### `loomux config get <schlüssel>`
Gibt den Wert des Schlüssels aus, die Vorgabe, wo keiner gesetzt ist. Für
eine Liste von Tabellen gibt es je Eintrag eine Zeile aus, die Tabelle so,
wie Go sie druckt (`map[reason:… regex:…]`), und eine leere Zeile, wenn es keinen gibt;
`list` zeigt die Zahl der Einträge (`list --json` als `count`).

### `loomux config set <schlüssel> <wert> [--yes]`
Berechnet die neue Datei, zeigt die Änderung als Zeilendiff auf `stderr`,
fragt `write these changes? [y/N]` und schreibt nur bei `y` oder `yes`.

- **Werte** werden getippt, wie ein Mensch sie schreibt: eine Zeichenkette
  ohne Anführungszeichen, eine Zahl, `true`/`false`, eine Liste als `a, b`.
  Ein Komma in einer `{…}`-Gruppe gehört zum Eintrag, `docs/**/*.{md,txt}, src`
  sind also zwei Globs. Ein Eintrag mit einem Komma, ein leerer Eintrag oder
  einer mit Leerraum an den Enden steht in doppelten Anführungszeichen, als
  TOML-Zeichenkette mit ihren Escapes: `"a,b", c` sind zwei Einträge, `""`
  ist ein leerer. Ein leerer Eintrag ohne Anführungszeichen ist keiner, `a,`
  ist die Liste aus `a`. Die interaktive Form zeigt eine Liste in eben dieser
  Schreibweise. Ein Aufzählungsschlüssel nimmt nur die Werte, die
  sein Leser annimmt. `verify.timeout` sind ganze Sekunden (`600`, nicht
  `10m`). Eine Zahl wird in ihrer schlichten Form geschrieben: `+600` und
  `0600` sind `600`. Das Wort `default` ist ein Wert wie jeder andere.
- **Vorgaben werden nie geschrieben**: `set` auf den Vorgabewert entfernt die
  Zeile, und ein Abschnitt, der dadurch leer wird, geht mit — außer es steht noch ein Kommentar darin. Eine Datei, die eine Vorgabe wiederholt, hielte sie gegen eine
  spätere Änderung der Vorgabe fest.
- **Geprüft von den echten Lesern**: Der neue Text geht an jeden Leser, der
  im Betrieb läuft (Bereichsdeklaration, `[modules]`, Policy, `[verify]`,
  Commit-Policy, `[agent]`, `[flow]`, Worktree-Spiegel), und wird erst
  geschrieben, wenn alle ihn annehmen. Eine Rollenbindung braucht darum zuerst
  ihr Modell: `set agent.roles.reviewer gemini` wird abgelehnt, bis
  `agent.models.gemini.provider` gesetzt ist.
- **Brain-Schlüssel brauchen `[area]`**: Ohne liest das Brain nichts; jeder
  Brain-Schlüssel außer `area.scope` wird darum mit `set area.scope first`
  abgewiesen.
- **Als Tabelle abgewiesen**: `model.roles`, `verify.profiles` und die Listen
  von Tabellen werden von Hand bearbeitet.
- **Als Raten abgewiesen**: eine Datei, die die Schlüssel des Abschnitts als
  gepunktete oder gequotete Schlüssel oder als Inline-Tabelle führt, eine
  mehrzeilige Zeichenkette, ein doppelter Schlüssel oder Abschnitt oder eine
  Zeile ohne bekannte Form. Die Meldung nennt die Zeile.
- **Wie geschrieben erhalten**: Eine ersetzte Zeile behält ihre Einrückung,
  die Abstände um `=` und ihren Kommentar am Zeilenende; eine UTF-8-
  Bytereihenfolgemarke am Dateianfang bleibt.
- **Ausgabe**: `loomux config: wrote <pfad>`, `already so; nothing written`
  (die Datei änderte sich nicht) oder `declined; nothing written`, alles auf
  `stderr`.

### `loomux config unset <schlüssel> [--yes]`
Nimmt die Zeile des Schlüssels heraus, sodass er auf seine Vorgabe
zurückfällt (oder ungesetzt ist, wo er keine hat); ein Abschnitt, der dadurch
leer wird, geht mit. Diff, Rückfrage, Leser, Abweisungen und Ausgabe sind die
von `set`; ein Brain-Schlüssel braucht hier kein `[area]`. Ein Schlüssel, der
nicht in der Datei steht, schreibt nichts (`already so`).

### `loomux config set|unset … --propose`
Berechnet die Änderung genau wie `set` oder `unset`, mit denselben
Abweisungen (Exit `1`), schreibt aber keine `config.toml`. Sie legt einen
Vorschlag als `.loomux/state/config/proposals/<id>.json` im Projekt ab (für
`--global`: `config/proposals/<id>.json` im Zustandsverzeichnis) mit `id`,
`op` (`set`/`unset`), `key`, `input` (der Wert wie getippt), `created`
(UTC) und `target` (`project`/`global`) und gibt Diff und Id auf `stdout`
aus. Ein Diff wird nicht abgelegt: Jede Form, die einen zeigt, rechnet ihn
neu gegen die Datei, wie sie dann ist. Die Id ist die UTC-Zeit und ein
Zähler (`20260924T101530Z-001`), Ids sortieren also in der Reihenfolge, in
der sie entstanden. Ein Vorschlag, der nichts ändern würde, wird nicht
abgelegt (`already so; nothing proposed`, Exit `0`).

### `loomux config proposals [--json]`
Listet die offenen Vorschläge, den ältesten zuerst: Id, Zeit, die erbetene
Änderung und den Diff, den sie an der aktuellen Datei machen würde, jetzt
gerechnet. Jeder Diff gilt für sich gegen die aktuelle Datei, nicht gegen
das, was die Vorschläge davor unter `apply --all` hinterließen. Ein
Vorschlag, der nicht mehr gilt, zeigt statt eines Diffs `refused now:
<grund>`, einer, der schon gilt, `already so; apply removes it`. Mit
`--json` trägt jeder Eintrag den neu gerechneten `diff` und, wenn er nicht
mehr gilt, einen `error`. Ist nichts offen, gibt es nichts aus (`[]` mit
`--json`). Eine Vorschlagsdatei, die sich nicht lesen lässt, wird auf
`stderr` genannt, die übrigen werden trotzdem gelistet, und der Lauf endet
mit `1`; `reject` entfernt sie trotzdem. Eine `config.toml`, die sich nicht
lesen lässt, ist `1`.

Schlüssel, Wert und Diff kommen von einem Agenten. In der Ausgabe von
`--propose`, `proposals` (Text und `--json`) und `apply` wird jedes
Steuerzeichen außer Zeilenumbruch und Tabulator (eine Escape-Sequenz, eine
Glocke, ein Wagenrücklauf, DEL, der C1-Bereich) und jedes Byte, das kein
UTF-8 ist, maskiert als `\xNN` ausgegeben; in `--json` steht diese Maske im
Wert der Zeichenkette (`"\\x1b"`). In den Kopfzeilen der Textformen und in
den Meldungen von `apply` steht ein Schlüssel oder Wert wie getippt, wenn er
ein Wort aus druckbaren Zeichen ist, und in Anführungszeichen mit
Go-Maskierung, wenn er leer ist oder ein Leerzeichen, ein `"` oder ein nicht
druckbares Zeichen enthält.

### `loomux config apply <id>|--all [--yes]`
Für jeden Vorschlag in Id-Reihenfolge: berechnet die Änderung aus `op`,
`key` und `input` neu gegen die Datei, **wie sie jetzt ist**, zeigt diesen
Diff, fragt (oder nimmt `--yes`), schreibt so, wie `set` schreibt, und
entfernt den Vorschlag. Ein Vorschlag, der schon gilt, wird mit einem
Hinweis entfernt; ein abgelehnter bleibt; einer, den ein Leser jetzt
abweist, bleibt mit seinem Fehler, die übrigen laufen weiter, und der Lauf
endet mit `1`. Ist die Änderung geschrieben, lässt sich die Vorschlagsdatei
aber nicht entfernen, sagt es genau das, und der Lauf endet mit `1`;
`reject` entfernt die Datei. Eine unbekannte Id ist `1`.

### `loomux config reject <id>|--all`
Entfernt den Vorschlag oder jeden offenen, ohne etwas zu schreiben.

### `loomux config` (interaktiv)
Eine Vollbildliste der Schlüssel, nach Modul gruppiert: `↑`/`↓` bewegen, `/`
filtert, `enter` ändert den Schlüssel unter dem Cursor, `q` oder `esc` beendet.
Ein Textschlüssel wird getippt; eine Aufzählung oder ein Wahrheitswert
wechselt mit `tab` durch die Möglichkeiten, und hat er eine Vorgabe, nimmt
eine weitere Möglichkeit, `(default)`, seine Zeile heraus wie `unset`. Ein
getippter Schlüssel kehrt mit `unset` zu seiner Vorgabe zurück. Danach folgen der Diff und eine
Bestätigung; jede Änderung wird für sich geschrieben, so wie `set` schreibt.
Eine Tabelle wird gezeigt, nicht bearbeitet; eine Liste von Tabellen zeigt
die Zahl ihrer Einträge und jeden Eintrag. Der Titel zeigt, wie viele
Vorschläge offen sind. Ohne Terminal endet die Form mit
Exit `2` und nennt `list`, `get`, `set` und `unset`.

### Exit-Codes
`0` Erfolg, auch eine abgelehnte Bestätigung, eine Änderung, die nichts
ändert, ein abgelegter Vorschlag und `apply`/`reject` ohne offenen
Vorschlag; `1` ein Leser oder der Editor lehnt ab (unbekannter Schlüssel,
ungültiger Wert, Raten, eine unlesbare Datei), das Schreiben scheitert, ein
Vorschlag lässt sich nicht ablegen oder lesen, eine unbekannte Vorschlags-Id
oder `apply` hat einen Vorschlag stehen lassen, der nicht mehr gilt; `2` ein
Bedienfehler (auch `--propose` mit `--yes`, ein `set`/`unset`, das das
Propose-Flag nennt, es am Ende aber aus hat — `--propose --propose=false`,
`-propose=0` oder `--propose` hinter `--`, wo es ein Wert ist —, mit der
Meldung `--propose given and switched off; say what you mean`, `--propose`
außerhalb von
`set`/`unset`, `--all` außerhalb von `apply`/`reject`, `apply`/`reject` ohne
genau eines von `<id>` und `--all`) und die interaktive Form ohne Terminal.

---

## 11. Projekt einrichten (`loomux init`)

Richtet ein Projekt für loomux ein: Es liest, was das Projekt ist, fragt je
Modul, was einzurichten ist, zeigt jede Änderung als Diff und jede Handlung
beim Namen und schreibt nur, was ein Mensch bestätigt. Gebaut mit Stufe 4a-2; am 2026-09-28 hat ein Mensch es auf einem frischen
Klon dieses Repositorys und interaktiv in einem Wirtsprojekt laufen lassen
(siehe den [Migrationsplan](migration.md)).

```bash
loomux init [--root DIR] [--dry-run] [--detect-only] [--yes]
            [--hooks=all|each|none] [--brain=all|each|none] [--graph=all|each|none]
            [--hosts=claude,antigravity]
```

- **`--root DIR`**: das Projekt; leer das Arbeitsverzeichnis.
- **`--dry-run`**: zeigt den Plan und schreibt nichts. Ohne Terminal behält
  es die Vorgaben, statt zu fragen.
- **`--detect-only`**: druckt als JSON, was das Projekt ist (Stacks, Wirte,
  Hookverzeichnis, Konfiguration, Binary, Merge-Hook, Graph), und endet;
  daneben nimmt es nur `--root`.
- **`--yes`**: nimmt die Vorgaben und bestätigt jede Änderung und Handlung,
  ohne Terminal.
- **`--hooks`, `--brain`, `--graph`**: `all` schaltet jeden Teil des Moduls
  an, `none` das Modul aus, `each` fragt Teil für Teil. Ein Flag schlägt die
  Antworten eines früheren Laufs. `--brain=none` registriert das Projekt
  trotzdem, als Workspace ohne Wiki (der Teil `workspace`), damit die
  Schreibschranke seinen Baum öffnet.
- **`--hosts`**: kommagetrennt, `claude`, `antigravity` oder `codex`;
  vorgegeben sind die Wirte, die das Projekt hat (`.claude/` → Claude Code;
  `.agents/hooks.json`, `.agents/skills/` oder `GEMINI.md` → Antigravity,
  ein `.agents/` allein genügt nicht; keines → Claude Code).
- `--dry-run=…` und `--detect-only=…` sind ein Usage-Fehler, und kein Flag
  nimmt ein folgendes `--dry-run` als Wert: Der Wächter lässt eine Zeile mit
  dem Wort `--dry-run` durch, ein Wert könnte das zurücknehmen.

### Module und Teile
Das Interview fragt je Modul `all`, `each` oder `none` (bei `each` eine
Vollbildliste seiner Teile), dann die Commit-Sprache (`en` oder `de`) und,
wenn das Projekt ein Bereich wird, seinen Scope. Die Vorgaben der Teile
folgen dem Projekt; die Antworten eines früheren Laufs gehen vor. `none`
schaltet das Modul immer aus; ein Modul, das läuft, ohne dass jetzt einer
seiner Teile eingerichtet wird (der Graph eines loomux-Checkouts), wird mit
`each` angeboten, sodass es an bleibt, wer jedes Angebot annimmt. Der
vorgegebene Scope ist `project/<verzeichnisname>`, Leerraum durch `-`
ersetzt, und `project/root`, wo vom Namen nichts bleibt.

| Modul | Teil | Was er tut | Vorgabe |
|---|---|---|---|
| base | `binary` | das Binary, das die Einträge rufen: `binary-install` legt das neueste Release nach `${LOCALAPPDATA}/loomux/bin/loomux.exe` (über `gh`, geprüft gegen `SHA256SUMS` und sein `--version`); in einem Checkout von loomux baut `binary-build` `bin/loomux.exe` | an |
| base | `config` | `.loomux/config.toml`: `[modules]`, wo ein Modul aus ist, `[commit] language`, wo sie nicht `en` ist, und die noch fehlenden Policy-Regeln der erkannten Stacks; `[verify]` bleibt den Presets. Der Text muss die eigenen Leser der Konfiguration bestehen. Dazu eine leere `.loomux/armed.toml`, nur wo vor dem Lauf weder sie noch eine Konfiguration noch ein pre-commit-Hook von loomux stand (siehe unten) | an |
| base | `gitignore` | `.gitignore`: `/.loomux/state/` | an |
| base | `agents-md` | `AGENTS.md`, nur wenn das Projekt keine hat | an, aus in einem Checkout |
| base | `mcp-json` | `.mcp.json` mit dem Server `loomux` (siehe [Die `.mcp.json` eines Wirts](#die-mcpjson-eines-wirts)) | an, aus in einem Checkout |
| base | `tools` | sucht `git`, `qmd`, `pdftotext`, `yt-dlp` und `ollama` auf dem `PATH` und nennt für ein fehlendes den Installationsbefehl; installiert nichts | an |
| hooks | `host-entries` | die Hook-Einträge jedes Wirts (`.claude/settings.json`, bei Antigravity `.agents/hooks.json`) | an |
| hooks | `git-hooks` | `pre-commit` (er fährt `check precommit --arm`), `pre-push` (verweigert einen Push nach `main` oder `master`) und `commit-msg` unter `.githooks`, dazu `git config core.hooksPath .githooks` | an in einem Repository |
| hooks | `verify-skill` | der Skill `verify-until-green` | an, aus in einem Checkout |
| hooks | `workspace` | nur, solange das Brain-Modul aus ist: `workspace-add` schreibt einen Registry-Eintrag mit dem Bereichsnamen, `path` und `workspace = true` und ohne `wiki`, unter der Registry-Sperre wie `area add`; nichts in `.loomux/config.toml`, kein Wiki, keine Routing-Regel, kein Merge-Hook. Die Brain-Leser überspringen einen solchen Eintrag, solange das Projekt kein `[area]` erklärt. Ohne Eintrag verweigert die Schreibschranke jeden Write im Projekt. Ein Bereichsname, den schon ein anderer Pfad trägt, beendet den Lauf mit Exit 1; die Handlung läuft vor jeder Datei, neben `area-add`, sodass eine verweigerte Registrierung nichts geschrieben hinterlässt. Ist das Hooks-Modul an und das Brain aus, vermerkt der Plan an einer registrierten Wurzel, gewählt oder nicht, `workspace: skipped; the registry has an area at this root already`, und `workspace: skipped by choice; the write barrier opens no tree in this project until a human registers one`, wenn der Teil in einem frischen Projekt abgewählt ist, das kein `[area]` erklärt und nicht registriert ist | an, aus in einem Checkout oder bei einem schon erklärten oder registrierten Bereich |
| brain | `area` | `loomux area add --scope <scope>`, ohne `--wiki`, also mit dem vorgegebenen Wiki von area add | an, aus in einem Checkout oder bei einem schon erklärten oder registrierten Bereich |
| brain | `merge-hook` | der post-merge-Hook von `loomux merge-hook install`, nur für die Bereiche an dieser Wurzel: Ein veralteter Bereich oder ein fremder Hook anderswo in der Registry hält `init` nicht auf. Geplant nur, wenn hier ein Bereich der Registry dieses Rechners steht und die Erklärung hier `[maintenance] on_merge = true` sagt, oder wenn `area` im selben Lauf in einem Projekt ohne `.loomux/config.toml` läuft (area add schreibt die Zustimmung nur in eine neue), und nur, wo `${LOCALAPPDATA}/loomux/bin/loomux.exe`, das der Hook ruft, installiert ist oder `binary-install` läuft, auch in einem Checkout; ohne eine Zeile für diese Wurzel scheitert die Handlung | an in einem Repository, aus in einem Checkout |
| brain | `brain-skills` | die Skills `brain-ingest`, `brain-land`, `brain-research`, `brain-review`, `brain-wiki-plan` | an, aus in einem Checkout |
| brain | `model` | fragt Ollama (`GET /api/tags`) nach dem Modell aus `[model] name` der rechnerweiten `config.toml` und plant, wenn es fehlt, `model-pull`: `ollama pull <name>` über `POST /api/pull`, mit dem Fortschritt auf stderr und ohne Gesamtzeitlimit; Ctrl+C beendet nur den Download. Das einzige Werkzeug, das `init` installiert statt es zu nennen; ein Fehlschlag ist ein Hinweis, `init` läuft weiter. Ist Ollama nicht erreichbar oder liegt der Endpunkt nicht auf dem Loopback, bleibt es beim Hinweis. Nur `init` lädt ein Modell, `reconcile` nie | an bei `[privacy] mode = "local_only"` im Projekt oder `[model] enabled = true` global, sonst aus |
| graph | `graph-build` | `loomux graph build` | an bei einem bekannten Stack, aus in einem Checkout |

Ein Checkout von loomux (sein `go.mod` erklärt `github.com/xidus90/loomux`)
bekommt als Vorgabe nur an, was seine eingecheckten Dateien schon haben,
damit `init --yes` auf einem frischen Klon `git status` leer lässt; ein Lauf
durch einen Menschen am 2026-09-28 baute das Binary, setzte
`core.hooksPath` und fand es so, und ein zweiter Lauf hatte nichts zu ändern.

### Was es schreibt und was es stehen lässt
- **Nie überschreiben, stattdessen melden.** Eine neue Datei entsteht
  exklusiv; eine vorhandene ändert sich nur, wo sie `init` gehört. Ein
  Host-Eintrag gehört `init`, wenn sein Befehl ein loomux-Binary ruft; ein
  fremder bleibt stehen und wird genannt. Eine Host-Datei oder `.mcp.json`,
  die kein JSON ist, deren Wurzel `null` ist oder deren `hooks` oder
  `mcpServers` kein Objekt ist,
  wird nicht repariert: Der Plan scheitert, und der Lauf endet, ohne etwas
  zu schreiben. Eine vorhandene `AGENTS.md` oder ein vorhandener Skill
  bleibt, wie er ist. Bevor sich eine Datei zum ersten Mal ändert, geht eine
  Kopie nach `.loomux/state/backup/<pfad>.bak` — außer
  `.loomux/config.toml`, die im Ganzen ersetzt wird, sobald ihre Leser den
  neuen Text annehmen.
- **Schonfrist je Lane.** `init` schreibt eine leere `.loomux/armed.toml` nur,
  wo vor dem Lauf weder diese Datei noch `.loomux/config.toml` noch ein
  pre-commit-Hook von loomux stand und der Teil `config` gewählt ist: Keine
  Lane eines Projekts, in dem loomux nie eingerichtet war, lässt das Tor
  scheitern, bevor ein grüner Commit sie scharf stellt. Ein Projekt, das schon
  eingerichtet ist, bekommt die Schonfrist nur durch `loomux gate disarm
  --all`, von einem Menschen ausgeführt; `init` legt dort nichts an, auch
  nicht, wenn es den Hook erneuert. Der pre-commit-Hook, den es schreibt, fährt
  `exec "<binary>" check precommit --arm`. Den eigenen älteren pre-commit-Hook
  (Shebang, Markerzeile und der eine Aufruf `check precommit`, sonst nichts)
  ersetzt es durch den heutigen, mit einer Kopie unter
  `.loomux/state/backup/`. Eine Datei, die `init` ersetzt, behält ihre
  Rechtebits, und ein Skript bekommt nur das Ausführungsbit dazu, damit der
  Hook ausführbar bleibt. Ein pre-commit-Hook des Projekts bleibt, wie er ist;
  hat das Projekt die Datei oder bekommt es sie in diesem Lauf und stellt
  dieser Hook nichts scharf, nennt ihn eine Notiz mit dem Hinweis, dort
  `loomux check precommit --arm` zu rufen oder von Hand mit `loomux gate arm`
  scharf zu stellen.
- **Einträge rufen das Binary an seinem Ort**: Die Einträge von Claude Code
  rufen in einem Wirtsprojekt `"${LOCALAPPDATA}/loomux/bin/loomux.exe"`, in
  einem Checkout `"${CLAUDE_PROJECT_DIR}/bin/loomux.exe"`; die Git-Hooks
  rufen dasselbe Binary, die eines Checkouts als `./bin/loomux.exe`. Steht
  dort kein Binary — der Teil `binary` ist aus, abgelehnt oder gescheitert
  —, entfallen diese Einträge und Git-Hooks, und `init` sagt es. Die
  Einträge von Antigravity rufen in jedem Projekt das installierte Binary
  über `cmd.exe`, ungequotet, als `%LOCALAPPDATA%/loomux/bin/loomux.exe`,
  und werden wie der Merge-Hook nur geplant, wo es installiert ist oder
  `binary-install` läuft (siehe unten). `binary-install` scheitert, wenn
  `LOOMUX_STATE_DIR` das Zustandsverzeichnis von `${LOCALAPPDATA}/loomux`
  wegverlegt, weil die Einträge sonst nichts riefen.
- **Reihenfolge**: das Binary, `area add`, die Dateien, `core.hooksPath`,
  der Merge-Hook, der Graph. `area add` schreibt `[area]`, `[layout]`,
  `[index]`, `[privacy]` und `[maintenance]` nur in eine Konfiguration, die
  es noch nicht gibt, und läuft deshalb zuerst; die eigene Änderung von
  `init` an `.loomux/config.toml` und eine fehlende `AGENTS.md` entstehen
  dann über dem, was es hinterlassen hat (die Vorlage vor seiner
  Routing-Regel). Der Plan zeigt den Diff gegen die Datei, wie sie steht,
  und sagt das in einer Notiz.
- **Eine Handlung wird nur geplant, solange ihr Ergebnis fehlt**:
  `binary-install` ohne installiertes Binary (Updates bleiben bei `serve` und
  `loomux upgrade`), `binary-build` ohne `bin/loomux.exe`, `merge-hook`
  ohne eigenen `post-merge`, `graph-build` ohne Graph.
- **Lebende Hooks behalten ihr Verzeichnis**: Ist kein `core.hooksPath`
  gesetzt und liegt in `.git/hooks` ein lebender `pre-commit`, `pre-push`
  oder `commit-msg`, schreibt `init` nur die fehlenden Hooks dorthin. Ein
  Hookverzeichnis außerhalb der Wurzel (ein verknüpfter Worktree, ein
  Submodul) bleibt mit einer Notiz unberührt.
- **Antigravity** bekommt vier Einträge in der Gruppe `loomux` von
  `.agents/hooks.json` — `PreInvocation` mit `session-start` (Timeout
  20 s), `PreToolUse` auf
  `write_to_file|replace_file_content|multi_replace_file_content|run_command`
  (15 s), `PostToolUse` auf die drei schreibenden Werkzeuge (60 s) und
  `Stop` mit `--budget 270s` (300 s); `PreToolUse` trifft auch
  `send_command_input` und `manage_task`. `PreInvocation` und `Stop` stehen
  als flache Liste von Handlern, die Werkzeug-Ereignisse als Block mit
  `matcher` und `hooks`: agy 1.2.11 verwirft sonst die ganze Datei. Ein
  `run_command` wird nach denselben Befehlsregeln beurteilt wie `Bash`; was
  ein `send_command_input` oder ein `manage_task` in eine Aufgabe tippt,
  ebenso, Zeile für Zeile und nur als ganze Zeilen ohne Steuerzeichen und
  ohne Zeilenfortsetzung. Ein Aufruf, in dem der Wächter keine Befehlszeile
  findet, wird verweigert; `list`, `status` und `kill` von `manage_task`
  laufen nur durch, solange sie keine Zeile tragen. Ein Eintrag von vor
  `manage_task` im Matcher bleibt stehen, und `init` hängt mit dem aktuellen
  loomux-Befehl und einer Notiz einen Block für `manage_task` daneben; ein
  eigener Eintrag von Claude Code unter einem älteren Matcher bekommt ebenso einen
  für `MultiEdit`. Ein Eintrag unter einem Matcher, der keine schlichte Liste
  von Werkzeugnamen ist, etwa `.*`, wird nur genannt, und was ihm fehlt,
  ergänzt man von Hand. Dazu
  kommen die Skills unter
  `.agents/skills/<name>/SKILL.md`, dieselben
  Texte wie Claude Code. agy führt einen Hook über `cmd.exe` aus
  `.agents/` aus: Es löst `%LOCALAPPDATA%` auf, lässt `${LOCALAPPDATA}`
  stehen und zerbricht einen gequoteten Programmpfad. Deshalb rufen die
  Einträge immer das installierte Binary, ungequotet, auch in einem
  Checkout:
  `%LOCALAPPDATA%/loomux/bin/loomux.exe hook pre-tool-use --host antigravity --root ..`
  (die drei anderen ebenso). Ohne dieses Binary und ohne
  `binary-install` im selben Lauf lässt der Plan die Einträge mit einer
  Notiz weg, wie beim Merge-Hook, und ein Lauf schreibt die Datei nur,
  solange das Binary steht; die Einträge von Claude Code behalten ihr
  eigenes Binary. Ein installiertes Binary, das älter ist als das laufende
  `init`, kennt diese Hooks womöglich nicht, und agy bricht bei einem
  scheiternden Hook ab: `init` fragt es nach seiner `--version` und plant
  die Einträge nur, wenn sie mindestens die des laufenden `init` ist, oder
  wenn `binary-install` im selben Lauf läuft. Ein Entwicklungsbuild von
  `init` (`0.0.0-dev`) hat keine Version zum Vergleichen und plant keine.
  Jeder Fall steht in einer Notiz. Der Merge-Hook wartet auf keine Version:
  Sein Aufruf ist still und endet mit 0, ein älteres Binary zeichnet also
  nichts auf, bis es aktualisiert ist. Enthält `LOCALAPPDATA`
  Leerraum oder eines von `, ; = & | < > ^ ( ) "`, würde `cmd.exe` den
  ungequoteten Pfad zerteilen: `init`
  schreibt dann keine Antigravity-Einträge, sagt es in einer Notiz und
  liest `.agents/hooks.json` nicht; die Skills kommen trotzdem. Jede andere
  Gruppe der Datei wird Token für Token übernommen (Schlüsselreihenfolge,
  Escapes und Zahlen wie vorher; nur die Einrückung wird zu zwei
  Leerzeichen), und eine Gruppe, in der ein loomux-Binary schon einen
  dieser Hooks ausführt, in welcher Form auch immer (anderer Pfad,
  gequotet, anderes `--root`), wird genannt
  (`the group X already runs loomux hook …; it now fires twice`), nie
  repariert. agy lädt die Hooks eines Projekts nur in einem Ordner, dem es
  vertraut (`trustedWorkspaces` in
  `~/.gemini/antigravity-cli/settings.json`); der Plan erinnert daran in
  einer Notiz. Codex hat keine Hookdatei.
- **Zustand**: `.loomux/state/answers.toml` hält die gewählten Wirte und
  Teile (alles andere steht in `.loomux/config.toml`);
  `.loomux/state/installed.toml` nennt, was der letzte Lauf geschrieben und
  ausgeführt hat, und wird zuletzt geschrieben, sodass ein abgebrochener Lauf
  seine offenen Änderungen im nächsten Plan wieder zeigt.

### Ausgabe
Der Plan auf `stdout`: jede Änderung als `--- <pfad>` mit Diff, dann
`actions:` mit einer Zeile je Handlung, dann `notes:`; `nothing to change`,
wenn nichts ansteht. Nach einem Lauf eine Zeile je Pfad oder Handlung:
`written: …`, `skipped: …`, `refused: …` (abgelehnt), `failed: …` und
`note: …` für einen Fehlschlag, der nichts aufhält (ein nicht beendeter
Download des Modells, mit dem Befehl, ihn von Hand nachzuholen). Während `model-pull` läuft,
geht sein Fortschritt als `model-pull: …`-Zeilen auf `stderr`: eine je
Status und eine je zehn Prozent einer Schicht im Download.

### Der Wächter
Der Wächter verweigert einem Agenten `loomux init`, außer es trägt
`--dry-run` oder `--detect-only` als eigenes Wort (siehe
[`hook pre-tool-use`](#loomux-hook-pre-tool-use)). Ein Mensch ruft es.

### Exit-Codes
`0` Erfolg, ein Probelauf, `--detect-only`, ein mit `esc` oder
Eingabeende abgebrochener Lauf (`loomux init: cancelled; nothing written`)
und ein Lauf, in dem der Mensch eine Änderung oder Handlung abgelehnt oder
einen Teil abgeschaltet hat, das Binary eingeschlossen, auch wenn dadurch
Host-Einträge, Git-Hooks und Merge-Hook entfallen, und ein Lauf, dessen
Download des Modells scheiterte oder mit Ctrl+C beendet wurde (ein `note:`,
`init` läuft weiter); `1` das Projekt lässt
sich nicht lesen (`go.mod`, git, die Registry), der Binary-Schritt, eine
Änderung oder Handlung ist gescheitert, das Terminal versagte im Interview
oder bei der Bestätigung, oder es ließ sich nicht zurücksetzen; `2`, bevor
etwas geschrieben ist: ein Usage-Fehler, eine Wurzel, die kein Verzeichnis
ist, eine `.loomux/config.toml` oder `.claude/settings.json`, die sich nicht
lesen lässt, eine `answers.toml`, die
nicht liest, ein scheiternder Plan (eine Konfiguration, die ihre Leser
ablehnen, eine Host-Datei oder `.mcp.json`, die kein JSON ist, deren Wurzel
`null` ist oder deren `hooks` oder `mcpServers` kein Objekt ist, jede Datei,
die der Plan nicht lesen kann), und ein Lauf, der fragen muss und kein
Terminal hat (`init asks questions; run it in a terminal, or pass --yes or
--dry-run`).

---

## 12. Flows (`loomux flow`)

Fährt einen Flow, hält ihn an einem Tor an, setzt fort, spielt nach, zeigt und
listet Flows: den Katalog, den das Binary mitbringt, und die eigenen des
Projekts unter `.loomux/flows/`. Format, Rollen, Overlays und Katalog stehen
in [Flows](flows.md). Agentenknoten warten auf die Modelladapter: Ein Flow mit
einem solchen Knoten lehnt den Start ab.

```bash
loomux flow run [<flow>] [--option name=wert]... [--root ordner]
loomux flow resume <lauf> [--answer text] [--root ordner]
loomux flow replay <lauf> [--root ordner]
loomux flow show <lauf|flow> [--root ordner]
loomux flow list [--root ordner]
```

- **Das Projekt** ist `--root`, sonst die nächste `.loomux/config.toml`
  oberhalb des Arbeitsverzeichnisses, sonst das Arbeitsverzeichnis selbst.
- **Erst der Name, dann die Flags**: Flow oder Laufnummer stehen vor den
  Flags. `-h` druckt die Flags und endet mit `0`.
- **Jeder Befehl, der einen Flow lädt, liest vorher `[agent]` und `[flow]`**
  und hält mit Exit `1` an einer Tabelle, die ihr Leser ablehnt (siehe
  [Konfiguration](configuration.md#agent-modelle-für-die-rollen-eines-flows)).
  `resume` und `replay` sehen zuerst Journal und Marke des Laufs an; die
  Ablehnungen, die von seinem offenen Tor abhängen, kommen, nachdem der Flow
  gefunden und geladen ist. `show <lauf>` liest nur das Journal.
- **Kein `stdin`**: Kein Befehl liest es; ein Lauf hält alles, was von ihm
  bleibt, in Journal und Marke, jeder Aufrufer fährt ihn also gleich.
- **Warnungen** gehen als `warning: …` nach `stderr` und ändern den Exit-Code
  nicht: ein Projektordner, den ein mitgelieferter Flow übergeht, Overlays,
  die sich seit dem Start eines Laufs geändert haben, ein Knoten, der auf einer
  inzwischen geänderten Definition lief.
- **Ablehnungen** gehen mit Exit `1` nach `stderr`; ein Flow, der nicht lädt,
  wird mit jedem Befund seiner Ladestufe abgelehnt, eine Zeile je Befund.
- **Der Wächter** verweigert einem Agenten `resume … --answer`; jede andere
  Form steht ihm offen (siehe [Flows](flows.md#6-tore-gehören-einem-menschen)).

### Exit-Codes

| Code | Bedeutung |
|---|---|
| `0` | der Lauf ist fertig; `show` und `list` gelingen; `-h` |
| `1` | ein Fehler oder eine Ablehnung: ein Flow, der nicht lädt, ein gescheiterter Lauf (ein Knoten scheiterte ohne Fehlerkante, keine Kante trifft zu, ein Besuchsdeckel), eine Antwort, die keine Wahl trifft, ein Replay eines Laufs, der an einem Ausgangsknoten endete |
| `2` | ein Aufruffehler: kein oder ein unbekannter Unterbefehl, ein fehlender Name oder eine fehlende Laufnummer, eine Laufnummer, die nicht nur aus Ziffern besteht, ein übriges Argument, ein unbekanntes Flag, ein `--option` ohne `=` oder doppelt |
| `3` | der Lauf ist an einem Tor pausiert |
| andere | der Code des Ausgangsknotens, an dem der Lauf endete |

### `loomux flow run [<flow>] [--option name=wert]... [--root ordner]`
Startet einen neuen Lauf und fährt ihn, bis er fertig ist, an einem Tor
pausiert oder scheitert.

- **Ohne Namen** fährt `[flow] default`. Ist es nicht gesetzt, lehnt der
  Befehl mit `no flow named and [flow] default is unset; known flows: example,
  ship` ab; ein Default, der keinen Flow nennt, mit `[flow] default names "x",
  which is no flow here`.
- **`--option name=wert`** setzt einen Parameter; wiederholbar. Der Text wird
  als Typ des Parameters gelesen: ein `int` dezimal, ein `bool` als `true` oder
  `false`, eine `list[string]` als JSON-Liste von Strings (`'["a","b"]'`), ein
  `string`, wie er steht. Jede Option, die kein Parameter ist oder sich nicht
  als ihr Typ lesen lässt, wird auf einmal genannt (`option x is no parameter
  of flow "ship"; known parameters: none`).
- **Abgelehnt, bevor es einen Lauf gibt**, Exit `1`: ein Name, der kein
  Flow-Name ist (`"Bad" is not a flow name; a flow name is [a-z][a-z0-9-]*`),
  kein Flow dieses Namens (`no flow named "nope"; known flows: example,
  ship`), ein Ladebefund, ein Anbieter ohne Adapter (`no adapter for provider
  claude yet`, heute jeder Flow mit einem Agentenknoten), Agentenknoten, die
  auf zwei Anbieter auflösen.
- **Ein Lauf** nimmt die nächste Nummer unter `.loomux/state/runs/`, schreibt
  seine Marke mit den Optionen, der Herkunft und, wenn Git antwortet, `HEAD`
  und den geänderten Dateien, und journalisiert jeden Schritt. Ein Lauf, den
  der Runner vor seinem ersten Schritt ablehnt, gibt seine Nummer zurück; einer,
  der später scheitert, behält seine Dateien, und die Meldung nennt den Lauf
  (`run 0003: …`).
- **Ausgabe** auf `stdout`: `run <id> (<flow>, <herkunft>): <status>` mit dem
  Status `done`, `paused` oder `error`, dann die Frage des Tors oder der Grund,
  aus dem der Lauf endete.

```
$ loomux flow run ship
run 0001 (ship, project): paused
Ship it?
$ loomux flow run quick        # ein Flow, dessen Start ein Ausgangsknoten mit Code 5 ist
run 0002 (quick, project): error
stopped at once
$ echo $?
5
```

### `loomux flow resume <lauf> [--answer text] [--root ordner]`
Setzt einen Lauf fort, der an einem Tor wartet.

- **Ohne `--answer`** fragt das Tor erneut, und nichts wird geschrieben;
  Exit `3`.
- **Mit `--answer`** nimmt das Tor die Antwort, und der Lauf geht weiter:
  Exit `0`, wenn er fertig ist, `3` am nächsten Tor, der Code eines
  Ausgangsknotens. Eine Antwort ist eine Wahl, oder eine Wahl, Leerraum oder
  `:` und eine Begründung. Auch `--answer ""` ist eine Antwort. Eine Antwort,
  die keine Wahl trifft, wird abgelehnt (`the answer matches none of the
  choices; the choices are yes, no`, Exit `1`), und das Tor bleibt offen.

  ```
  $ loomux flow resume 0001 --answer "no: too thin"   # das mitgelieferte example, in seinem Test
  run 0001 (example, bundled): error
  rejected after 2 rounds
  $ echo $?
  4
  ```
- **Abgelehnt als Aufruffehler**, Exit `2`, bevor etwas gelesen wird: eine
  Laufnummer, die nicht nur aus Ziffern besteht, wie `loomux flow run` sie
  vergibt (`loomux flow resume: "../x" is no run number; a run number is
  digits, as loomux flow run hands them out`). Die Nummer wird Teil eines
  Pfads; ein `../`, ein Trenner oder ein Laufwerk läse Journal und Marke
  anderswo.
- **Abgelehnt**, Exit `1`: ein Lauf, den es nicht gibt (`no run "0009" under
  …`, mit dem Laufordner in Schrägstrichen), ein Lauf ohne
  Marke (`run "0001" does not say which flow it belongs to`) oder mit einer,
  die sich nicht lesen lässt (eine leere: `…0001.flow: says nothing -- not even
  which flow it belongs to`), ein
  Lauf, der an keinem Tor wartet (``run 0002 is not waiting at a gate; there
  is nothing to answer. Use `loomux flow replay` to re-derive it, or `loomux
  flow run` to start a new one``), ein Flow, dessen `flow.toml` inzwischen aus
  der anderen Quelle kommt, und ein Anbieter ohne Adapter.
- **Die Quelle des Flows** wird mit der Marke verglichen, bevor der Flow
  lädt. Der Projektordner (`project`, `project (hides bundled)`) und der
  Katalog (`bundled`, `bundled+overlay`) sind zwei Quellen; ein Wechsel
  zwischen ihnen wird mit beiden Herkünften abgelehnt (`run 0001 started on
  project (hides bundled) and example now resolves to bundled, another
  flow.toml; start a new run with loomux flow run example`). Innerhalb einer
  Quelle läuft der Lauf weiter; eine andere Menge Overlay-Dateien ist eine
  Warnung (`warning: run 0001 started with overlays questions/approve.md and
  now has instructions/draft.md, questions/approve.md`).
- **Nur ein Mensch antwortet.** Der Wächter verweigert `--answer` einem Agenten
  in jeder Schreibweise; `resume` ohne steht offen.

### `loomux flow replay <lauf> [--root ordner]`
Leitet einen beendeten Lauf aus seinem Journal neu her. Es führt keinen
Knoten aus und fragt kein Modell, geht also auch ohne Adapter.

- **Exit** ist der des Laufs, wie er verzeichnet ist: `0` für fertig, `1` für
  einen Fehler. Das Journal hält die Meldung eines Ausgangsknotens, aber nicht
  seinen Code; ein Lauf, der an einem endete, spielt sich also mit seiner
  Meldung und Exit `1` nach.
- **Abgelehnt als Aufruffehler**, Exit `2`: eine Laufnummer, die nicht nur
  aus Ziffern besteht (wie bei `resume`).
- **Abgelehnt**, Exit `1`: ein Lauf, der an einem Tor wartet (``run 0001
  never finished: it is waiting at gate "confirm"; answer it with `loomux flow
  resume` before replaying``), ein Lauf, den es nicht gibt, ein Lauf ohne
  Marke oder mit einer, die sich nicht lesen lässt, ein Flow, dessen
  `flow.toml` inzwischen aus der anderen Quelle kommt (wie bei `resume`).

```
$ loomux flow replay 0002      # Lauf 0002 von quick endete an seinem Ausgangsknoten mit Code 5
run 0002 (quick, project): error
stopped at once
$ echo $?
1
$ loomux flow replay 0001      # Lauf 0001 von ship wartet an seinem Tor
run 0001 never finished: it is waiting at gate "confirm"; answer it with `loomux flow resume` before replaying
$ echo $?
1
```

### `loomux flow show <lauf|flow> [--root ordner]`
Eine Laufnummer besteht aus Ziffern, ein Flow-Name beginnt mit einem
Buchstaben; verwechseln lassen sich beide nicht.

- **Ein Lauf**: eine Zeile je Journaleintrag, in Spalten: Knoten, Art,
  Ausgang, Tokens, Sekunden, Werkzeugprofil (`-`, wo keines ist).

  ```
  $ loomux flow show 0001
  confirm                  gate   paused        0 tok    0.00s -
  ```
- **Ein Flow**, wie ein Lauf ihn laden würde: seine Herkunft, seine Knoten
  (ein Agentenknoten mit seiner Rolle, dem Modell, auf das sie auflöst, und
  woher) und seine Kanten mit der Bedingung, unter der jede genommen wird,
  `[on error]` für eine Fehlerkante.

  ```
  $ loomux flow show example
  example (bundled)
  nodes:
    draft    agent  writer  claude:cli-default  role writer (node), the CLI's own default
    approve  gate
    stop     exit
  edges:
    draft -> approve [verdict == "done"]
    draft -> draft
    approve -> END [answer == "yes"]
    approve -> stop
    stop -> END
  ```

### `loomux flow list [--root ordner]`
Jeder Flow, den das Projekt nennen kann, sortiert: Name, Herkunft (`project`,
`bundled`, `bundled+overlay`, `project (hides bundled)`) und `ok` oder die
Befunde, die ihn am Laden hindern, eine Zeile je Befund; `(default)` markiert
`[flow] default`. Ein Flow, der nicht lädt, steht mit seinem Grund in der
Liste, statt wegzufallen, ebenso eine Datei direkt unter `.loomux/flows/`
(`… is a file; a flow is a folder with flow.toml`) und ein Link dort,
symbolisch oder eine Junction (`… is a link; a flow is a folder with
flow.toml`).

```
$ loomux flow list
broken   project  .loomux/flows/broken/flow.toml: schema_version 7 is unknown; loomux knows version 1
    .loomux/flows/broken/flow.toml: [flow] is missing
example  bundled  ok
ship     project  ok
```

- **Warnungen** auf `stderr`: ein Projektordner, den ein mitgelieferter Flow
  übergeht (`warning: .loomux/flows/example is ignored: [flow] overrides does
  not name it`), und ein Name in `[flow] overrides`, den kein mitgelieferter
  Flow hat (`warning: [flow] overrides names "ghost", which no bundled flow
  has`).
- **Exit** `0`; `1` nach der Liste, wenn `[flow] default` keinen Flow nennt.

---

## 13. Das Tor (`loomux gate`)

Was ein Mensch dazu sagt, welche Lanes das Tor scheitern lassen: die Befehle
für [`.loomux/armed.toml`](configuration.md#schonfrist-je-lane-loomuxarmedtoml).
Eine eigene Gruppe und nicht unter `check`, wo jedes erste Wort ein Profil oder
eine Art ist. Eine Lane heißt nach ihrem Schlüssel, `<art>/<stack>@<bereich>`,
mit `/` geschrieben (`lint/go@sub/dir`, `lint/go@.` für die Wurzel); ein mit
`\` getippter Schlüssel wird mit `/` gelesen. Jeder Befehl nimmt `--root
<ordner>`, das Projekt. Ohne Angabe wird es aufwärts vom Arbeitsverzeichnis
gesucht: das Verzeichnis mit `.loomux/config.toml`, zuerst und ohne Grenze
gesucht, wie `check` es sucht; sonst das nächste mit `.loomux/armed.toml`,
eine Suche, die an der obersten Ebene des git-Repositorys endet; sonst diese
oberste Ebene. Außerhalb eines Repositorys gibt es keine Suche nach
`.loomux/armed.toml`, und ohne Konfiguration ist das Arbeitsverzeichnis das
Projekt. So findet sich ein Projekt in Probe ohne Konfiguration aus jedem
seiner Unterverzeichnisse, und die Datei wird dort nie daneben geschrieben. Ein Agent führt nur `loomux gate status` aus: Der
Wächter verweigert `arm` und `disarm` einem Agenten.

### `loomux gate status [--root <ordner>]`
Druckt jede Lane des Tors des Projekts und jeden Eintrag, dem keine Lane
antwortet, je eine Zeile `<schlüssel>: <zustand>`, nach Schlüssel sortiert; der
Zustand ist `armed`, `probation` oder `orphan` (ein Eintrag, dessen Stack
wegfiel oder dessen Bereich umbenannt wurde). Die Lanes sind die, die ein
Profil fahren kann: Eine Lane ohne etwas zu prüfen (`not-applicable`,
`unavailable`) steht nicht dort, `missing-tool` und `unready` schon.

- **Die drei Zustände der Datei.** Ohne sie druckt der Befehl `no
  .loomux/armed.toml: every lane is armed` und plant keine Lane. Mit ihr die
  Liste unten. Eine Datei, die sich nicht lesen lässt, ist ein Fehler (Exit
  `1`), der sagt, warum und dass jede Lane scharf ist.
- **Eine Datei, die git ignoriert,** steht in der Liste, wie sie ist, und
  `stderr` warnt, dass sie keinen Commit erreicht und nur auf diesem Rechner
  gilt.

```text
$ loomux gate status
coverage/go@.: probation
coverage/python@web: probation
lint/go@.: armed
lint/python@web: probation
lint/typescript@old: orphan
test/go@.: probation
test/python@web: probation
types/python@web: probation
```

### `loomux gate arm <lane>... [--root <ordner>]`
Trägt die Lanes in die Datei ein, auch eine rote: Von nun an zählen sie. Es
druckt `armed: <schlüssel>`. Ein Schlüssel, dem keine Lane antwortet, ist
zuerst ein Fehler, mit der Datei und ohne sie, und nichts wird geschrieben.
Ohne die Datei wird ebenfalls nichts geschrieben, weil ohnehin jede Lane scharf
ist: Der Befehl druckt die Zeile `no .loomux/armed.toml: every lane is armed`.

### `loomux gate disarm <lane>...|--all [--root <ordner>]`
Nimmt Einträge aus der Datei, auch einen verwaisten, und druckt `probation:
<schlüssel>`; einen Schlüssel, den die Datei nicht hielt, nennt es `<schlüssel>:
was not armed`. Ohne die Datei legt es eine an, die jede andere Lane als scharf
nennt (ein Schlüssel, der keine Lane ist, ist zuerst ein Fehler). `--all`
legt die Datei ohne Eintrag an oder leert sie und druckt `probation: every
lane`: **der Weg, einem schon eingerichteten Projekt die Schonfrist zu geben.**

- **Exit-Codes**: `0`; `1` für eine Datei oder Lanes, die sich nicht lesen oder
  schreiben lassen, und für einen unbekannten Schlüssel; `2` für einen
  falschen Aufruf (kein Unterbefehl, ein unbekannter, `arm` ohne Lane,
  `disarm` ohne Lane und ohne `--all` oder mit beidem).

