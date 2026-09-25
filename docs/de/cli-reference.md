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

### Globale Flags & Umgebung
- `--root <pfad>`: Explizite Angabe der Projektwurzel. Wird dieses Flag weggelassen, wandert Loomux im Verzeichnisbaum aufwärts, bis es die erste `.loomux/config.toml` findet.
- `LOOMUX_STATE_DIR`: Überschreibt das globale Zustandsverzeichnis (Standard: `%LOCALAPPDATA%\loomux` unter Windows, `~/.local/state/loomux` unter POSIX).
- `LOOMUX_LEGACY_BRAIN_DIR`: Das Zustandsverzeichnis von ultra-brain, als Rückfall für brain-Artefakte gelesen und nie beschrieben (siehe Abschnitt 7).

---

## 2. Policy & Prüfketten (`loomux check`)

`loomux check` nimmt zuerst eine **Anfrage** und danach ihre Flags. Die Anfrage
ist ein Profil (`edit`, `precommit`, `stop` oder eins aus `[verify.profiles]`), `all`,
eine Komma-Liste von Arten (`lint,types`) oder einer der fünf eingebauten
Prüfbefehle unten. Was jede Art je Stack fährt, legen
[`[verify]`](configuration.md#verify-prüfketten--quality-gates) und die Presets
fest.

### `loomux check <anfrage> [--root <pfad>] [--show] [-v]`
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
- **Reihenfolge**: Eine Lane startet, sobald die Lane, auf die sie wartet
  (`after`), fertig ist, mit höchstens `max_parallel` Prozessen gleichzeitig.
  Der Bericht kommt am Ende, nie verzahnt: Arten in Anfrage-Reihenfolge, darin
  Stacks in Byte-Ordnung, dann Bereiche. Die eine Ausnahme ist `lint/wiki`, der
  Lint über das Wiki-Bündel, der in diesem Prozess läuft, wo `lint` angefragt
  ist und das Projekt ein Wiki hat: er kommt zuletzt. Er prüft nur die Struktur
  des Bündels; die Drift-Regel bleibt bei `loomux wiki-gate`.
- **Ausgabe** (alles auf `stdout`): eine Zeile je Lane,
  `<art>/<stack>[@<bereich>]: <zustand> [<herkunft>]`, dahinter die Dauer bei
  einer Lane, die gestartet ist, `by <lane>` bei einer blockierten oder der
  Grund bei einer, die nie startete. `<herkunft>` ist `preset`,
  `preset, variant <signal>`, `config` oder `in-process`. Eine rote Lane druckt
  ihre Ausgabe unter ihre Zeile, eine grüne nur mit `-v`. Eine Lane mit
  mehreren Befehlen druckt einen Block je Befehl mit der Kopfzeile `$ <argv>`,
  bei einem roten mit `(failed)`. Eine Art, die nichts zu prüfen hatte,
  schließt den Bericht mit ``nothing to check for `<art>` ``.
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
- **Exit-Codes**: `0` (keine Lane rot, und jede angefragte Art hatte eine
  Lane, die lief, oder ist irgendwo `not-applicable`), `1` (eine Lane ist rot,
  eine Art hatte nichts zu prüfen, oder `[verify]` bzw. die Anfrage lässt sich
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
- **Exit-Codes**: `0` (Korrekt formatiert), `1` (Unformatierte Dateien auf `stdout` gelistet).

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
  `--threshold <n>` (Vorgabe `3`; das Go-Preset setzt `5`);
  `--skip-test-callers` (nur Aufrufer außerhalb von `_test.go`-Dateien zählen).
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

Hook-Einstiegspunkte werden von Coding-Agenten synchron bei Werkzeugaufrufen gestartet.

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
- **Befehle, die die Konfiguration schreiben**: Eine `Bash`- oder
  `PowerShell`-Zeile, die `loomux init` (ohne befreiendes `--dry-run` oder
  `--detect-only`), jedes `loomux config` außer `config list …`,
  `config get …`, `config proposals …`, einem alleinstehenden
  `config --help` oder `config -h` und `config set …` oder `config unset …`
  mit befreiendem `--propose`, `loomux area add` oder
  `loomux merge-hook install` oder `remove` (`status` und `record` gehen
  durch) ausführt, wird verweigert mit ``loomux init, config and area add
  write the configuration the guard reads, and merge-hook install and remove
  write executable hooks into repositories; a human runs them. An agent
  proposes a change with
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

  Diese Befehle schreiben `.loomux/config.toml` aus ihrem eigenen Prozess, wo
  keine Pfadregel den Schreibvorgang sieht. Erkannt wird das Programm als
  `loomux`, `loomux.exe` oder ein Pfad, der auf eines von beiden endet (mit
  oder ohne Anführungszeichen, `\` oder `/`), und als `go run` von
  `cmd/loomux` oder `cmd/loomux/main.go` (mit oder ohne `./`, unter einem
  Modulpfad, in jeder `@version`, hinter Build-Flags). Gefunden wird es
  hinter `VAR=wert`, Umleitungen (`2>/dev/null`, `> out`, `2>&1`), den
  reservierten Wörtern `if`, `then`, `else`, `elif`, `while`, `until`, `do`,
  `!`, `{`, `coproc`, `function <name>`, `try`, `catch` und `finally`,
  hinter jedem `{` oder `}` der Zeile, allein oder an ein Wort geklebt (dem
  Rumpf eines Blocks, einer Funktion oder eines Skriptblocks: `try{`,
  `{loomux …}`), sowie hinter den Wrappern `sudo`, `command`, `exec`,
  `nohup`, `env`, `time`, `xargs`, `nice` (auch `nice -n N`),
  `timeout <dauer>` und `cmd` mit jedem Schalter bis `/c` oder `/k`, samt
  ihren Flags ohne eigenen Wert (und `--`). `Start-Process`, `start` oder
  `saps` wird verweigert, wenn loomux eines seiner Argumente ist, auch als
  Wert eines Parameters mit Doppelpunkt (`-FilePath:loomux.exe`), gleich
  welche die übrigen sind. Jeder Abschnitt der Zeile zählt
  (`;`, `|`, `&`, `&&`, `||`, Zeilenumbruch, `(`, `)`, `$(`, ein Backtick),
  und eine Zeilenfortsetzung (`\` oder ein Backtick am Zeilenende) wird
  vorher zusammengefügt; ein Backtick-Escape in einem Wort
  (``loomux con`fig``) wird gelesen, wie PowerShell ihn liest.
  - **Bekannte Lücken** — die Regel liest Wörter, keine Shell, und lässt
    darum durch: einen Alias; ein Programm in einer Variablen; ein
    Wrapper-Flag mit eigenem Wert (`sudo -u root loomux init`,
    `xargs -n 1 …`, `timeout -s KILL 60 …`); einen Befehl in einer
    Zeichenkette (`sh -c "loomux init"`, `pwsh -c …`); `go run .` in
    `cmd/loomux`; und, nach einem früheren maskierten `\"` oder `\'` auf
    derselben Zeile, einen Programmpfad in Anführungszeichen, dessen Teil
    hinter seinem letzten Trennzeichen (`(`, `)`, `&`, `;`, `|`) ein
    Leerzeichen enthält, etwa
    `echo "a \" b"; "C:\Program Files (x86)\My Tools\loomux.exe" init`.
  - **Bekannte Fehlverweigerungen** — im Zweifel verweigert sie:
    `echo "x; loomux init"`, `start loomux config list`,
    `Start-Process code -ArgumentList loomux`, `command -v loomux init` (das
    den Namen nur nachschlägt), ein loomux-Wort direkt hinter einer Klammer,
    die keinen Block öffnet (`awk '{ print }' loomux init`,
    `echo } loomux config set a b`, `echo ${X} loomux init`), `loomux init \`
    mit `--dry-run` auf der nächsten Zeile (PowerShell führte die erste Zeile
    allein aus) und `loomux config --root <verz> list` (Flags vor dem
    Unterbefehl; diese Form ist ohnehin ein Bedienfehler). Ebenso eine Zeile,
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
- **Übersprungene Lanes**: Eine Lane, deren Werkzeug nicht auf dem `PATH` liegt, ein noch nicht importiertes Godot-Projekt und jede Lane, die das Budget nicht mehr erreicht, werden übersprungen, nicht rot. Sie stehen auf `stdout` als `{"hookSpecificOutput":{"additionalContext":"loomux hook post-tool-use: lane skipped, the edit budget ran out: lint/go","hookEventName":"PostToolUse"}}`.
- **Blast-Monitor**: Nach einem Edit an einer `.go`-Datei ohne rote Lane folgen den übersprungenen Lanes im selben `additionalContext` die direkten Aufrufer in anderen Dateien jedes Symbols, das der Edit gegenüber dem Graphen auf der Platte geändert oder entfernt hat. Ohne Graph schweigt er, und ein Befund ist er nie; siehe [Hooks](hooks.md#der-blast-monitor).
- **Exit-Codes**: `0` (alle Lanes grün, übersprungen oder nichts zu fahren), `1` (fehlerhafter Aufruf, etwa ein fehlendes `--host`, oder ein `[verify]`, das sich nicht laden lässt), `2` (eine Lane ist gescheitert, abgelaufen oder blockiert; ihre Ausgabe auf `stderr`).

### `loomux hook session-start`
Hält den Commit fest, auf dem die Sitzung beginnt.

- **Flags**: `--host <h>` (Pflichtfeld; nur `claude` hat einen Adapter), `--root <r>`.
- **Verhalten**:
  - Schreibt `HEAD` als `base` in `.loomux/state/hooks/<session_id>.json`.
  - Belebt eine Sitzung wieder, die `worktree unlink` als beendet markiert hat: entfernt `<session_id>.ended` und schreibt die Datei mit zurückgesetzter Blockreihe zurück, sodass die Sitzung wieder zählt; bei einer nie als beendet markierten Sitzung wird die Datei nur verjüngt. Eine Marke, die sich nicht entfernen lässt, oder eine Datei, die sich nicht zurückschreiben lässt, steht im Kontext, mit Exit 0.
  - Warnt in `hookSpecificOutput.additionalContext`, wenn das Binary im Projekt älter ist als seine Go-Quellen.
  - Liest außerdem `<Zustandsverzeichnis>/update.json` und warnt, wenn unter Windows ein Durchlauf von `serve` ein anderes Binary als `<Zustandsverzeichnis>/bin/loomux.exe` als sein eigenes verzeichnet hat, oder wenn der letzte Self-Update-Durchlauf gescheitert ist, gleich wer ihn fuhr.
  - Legt keine Worktree-Junctions an; das tut `loomux worktree link`. Siehe [Hooks](hooks.md#8-sitzungshooks).
- **Exit-Codes**: `0` (Erfolg), `1` (fehlender oder unbekannter Host, kein Adapter für den Host, unlesbare Nutzlast, gescheitertes Schreiben).

### `loomux hook stop`
Das Tor am Rundenende: stellt zu, was Subagenten hinterlassen haben, und fährt dann das Profil `stop` über das, was sich seit dem letzten grünen Lauf geändert hat.

- **Flags**: `--host <h>` (Pflicht; nur `claude` hat einen Adapter), `--root <r>`, `--budget <dauer>` — wie lange die Lanes zusammen dauern dürfen (Go-Dauer, Vorgabe `270s`, unter den 300 s, die sein Settings-Eintrag gewährt). Jeder Befehl bekommt das Kleinere aus seinem eigenen `timeout` und dem Rest des Budgets.
- **Standard-Input (stdin)**: die `Stop`-Nutzlast des Hosts; gelesen wird nur `session_id`.
- **Verhalten**: in dieser Reihenfolge — die Befunde der Subagenten auf `stderr`, der Blockzähler (nach 3 Blockaden in Folge gibt er für eine Runde auf und lässt die Befunde für die nächste liegen), der Marker `.loomux/no-verify` (er überspringt die Kette, nicht die Befunde), der Fingerabdruck des Inhalts (nichts Neues seit dem letzten grünen Lauf oder der Basis: kein Werkzeug startet), dann die Arten des Profils `stop` (vorgegeben `lint`, `types`, `test`, `coverage`) im Check-Scope, dazu `lint/wiki`, wo `lint` angefragt ist und es ein Wiki gibt. Ein grüner Lauf rückt `base` auf `HEAD` vor und merkt sich den Baum. Siehe [Hooks](hooks.md#stop).
- **Standard-Fehler (stderr)**: zugestellte Befunde als `subagent <agent_id>: <zeile>`, danach nur die roten Lanes, im Format von `loomux check`.
- **Exit-Codes**: `0` (die Runde endet: grün, nichts Neues, der Marker, oder der Zähler hat aufgegeben), `2` (die Runde wird angehalten: eine rote Lane, ein Git-Fehler, oder zugestellte Befunde — mit Befunden wird selbst ein Exit 1 zu 2), `1` (das Tor konnte nicht urteilen: eine unlesbare Nutzlast oder eine ohne `session_id`, das Budget war aufgebraucht, eine angefragte Art hatte nichts, was lief, `[verify]` lässt sich nicht laden, der Plan scheitert, das Coverage-Verzeichnis lässt sich nicht vorbereiten (`verify.PrepareCover`), oder ein fehlerhafter Aufruf; die Runde endet).

### `loomux hook subagent-start`
Hält fest, wo `origin`, die lokalen Branches und `HEAD` stehen, bevor ein Subagent läuft.

- **Flags**: `--host <h>` (Pflicht; nur `claude` hat einen Adapter), `--root <r>`.
- **Standard-Input (stdin)**: die `SubagentStart`-Nutzlast des Hosts; gelesen werden `session_id` und `agent_id`.
- **Verhalten**: `git ls-remote origin` (Frist 10 s, `GIT_TERMINAL_PROMPT=0`), die lokalen Branches und `HEAD` kommen nach `.loomux/state/hooks/<session_id>/agents/<agent_id>.json`; ein Remote, der nicht antwortet, wird als `unavailable` festgehalten. Ein Befund, der noch in dieser Datei geparkt ist, bleibt erhalten. Siehe [Hooks](hooks.md#subagent-start-und-subagent-stop).
- **Exit-Codes**: `0` (geschrieben), `1` (fehlender oder unbekannter Host, keine `session_id` oder `agent_id`, unlesbare Nutzlast, gescheitertes Schreiben). Nie 2.

### `loomux hook subagent-stop`
Vergleicht mit dem Schnappschuss und parkt, was sich bewegt hat, für das `stop` des Hauptagenten.

- **Flags**: `--host <h>` (Pflicht; nur `claude` hat einen Adapter), `--root <r>`.
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
  - Pfad der Projektwurzel und deklarierte Bereiche.
  - Aktive Agenten-Harnesses (`.claude/`, `.agents/`, `.cursor/`).
  - Status der Schreibschranke und Anzahl der aktiven Policy-Regeln.
  - Die Lanes, die der post-edit-Hook je aktivem Stack fährt: das Profil `edit`, wie `[verify]` und die Presets es auslegen, jede mit ihrer Herkunft, und welche ihrer Werkzeuge auf dem `PATH` fehlen.
  - Den einzutragenden `Stop`-Eintrag (`loomux hook stop`, Profil `stop`, das Wiki-Bündel als `lint/wiki`), und für jedes der sechs Ereignisse `PreToolUse`, `PostToolUse`, `SessionStart`, `Stop`, `SubagentStart` und `SubagentStop`, ob `.claude/settings.json` seinen `loomux hook` ruft (`[OK]`) oder nicht (`[INFO]`), dazu Alt-Hooks, die er ersetzt.
- **Exit-Codes**: `0` (Bereit), `1` (Konfigurationsfehler).

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
> **`build`, `check`, `ask`, `callers`, `skeleton`, `grep`, `map`, `stats` und `blast` sind verdrahtet; `viz` bleibt spezifiziert.** Stufe G1 hat die Pakete gebaut, auf denen der Graph aufsetzt — `internal/code/model`, `internal/code/pagerank` und `internal/code/blast` —, Stufe G2a ergänzt Extraktor, Wiring-Schreiber, Frischesonde und die Befehle `build` und `check`, Stufe G2b ergänzt Lexik, lexikalisches Scoring, Personalized-PageRank-Verschmelzung und `graph ask`, Stufe G3 stellt `ask` und `check` hinter die MCP-Werkzeuge `graph_find_code` und `graph_check_freshness` (§8), und Stufe G4a lieferte die Navigationspalette (`callers`, `skeleton`, `grep`, `map`, `stats`) und ihre vier MCP-Werkzeuge; Stufe G4b ergänzte `blast`, das MCP-Werkzeug `graph_blast`, die Prüfbefehle `graph-fresh` und `blast-audit` (§2) und den Blast-Monitor im Post-Edit-Hook (§3).

### `loomux graph build [--root <pfad>]`
Liest und hasht jede Go-Quelldatei, die `internal/code/sourceset` unterhalb der Wurzel findet, extrahiert und löst sie zum deterministischen AST-Graphen auf und schreibt ihn nach `.loomux/state/graph/wiring.json`. Dabei schreibt er auch die Frischeakte (`.loomux/state/graph/cache/fingerprint.json`), die eine spätere Sonde liest; scheitert das Schreiben der Akte, meldet der Befehl das auf `stderr`, ohne den Bau selbst scheitern zu lassen — der Graph auf der Platte ist bereits korrekt.

- **Flags**: `--root <pfad>` — Projektwurzel; ohne Angabe das Arbeitsverzeichnis.
- **Ausgabe**: eine Zeile mit Dateien, Knoten und Kanten je Relation, dann eine Zeile mit unaufgelösten Importzielen, Dateien ohne Symbol und der benötigten Zeit. Illustrative Form, kein zu erwartender Wert — jeder Teil davon, auch die Zeit, bewegt sich mit dem eigenen Code dieses Repositories, und die letzten drei Commits haben hier jeweils eine Zahl geschrieben, die der nächste Commit widerlegt hat: `N files, N nodes, N edges (N contains, N calls, N imports)` / `N unresolved import targets, N files without a symbol, Nms`. Gemessene Zahlen mit Befehl und Rohausgabe stehen in `docs/de/benchmarks.md`.
- **Exit-Codes**: `0` bei Erfolg; `1`, wenn die Wurzel nicht auflösbar ist, eine Datei nicht gelesen oder geparst werden kann, die Modulauflösung scheitert oder der Graph nicht geschrieben werden kann; `2` bei einem Aufruffehler.
- **Kosten**: `build` liest die Frischeakte nie — es liest und hasht jede Datei, kalt wie warm, jedes Mal. Es gibt dabei nichts zu überspringen: anders als `check` erzeugt `build` gerade den Stand, gegen den eine Sonde später vergleicht, und ein veraltetes Byte darin wäre eine veraltete Antwort, keine ersparte Lesung. Gemessene Zahlen stehen in `docs/de/benchmarks.md`.

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
- **Exit-Codes**: `0` bei Erfolg (auch wenn keine Symbole matchen); `1` bei Fehlern (noch kein Graph, unlesbarer Graph, fehlerhafter Neubau); `2` bei Aufruffehlern (fehlende Anfrage, negatives Limit).

### `loomux graph callers <symbol> [--direction in|out] [-d <tiefe>] [--in <präfix>] [--json]`
Zeigt, wer ein Symbol aufruft, importiert oder referenziert (`--direction in`, Standard), oder was das Symbol selbst aufruft (`--direction out`).

- **Flags**:
  - `--direction <in|out>`: Verfolgt eingehende Aufrufer (`in`) oder ausgehende Aufrufe (`out`).
  - `-d <tiefe>`: Transitive Tiefe (Standard `1`; `-d all` oder `-d full` für die vollständige transitive Hülle).
  - `--in <präfix>`: Filtert Symbole vor der Auflösung nach Pfadpräfix.
  - Jeder direkte Treffer (Tiefe 1) trägt die erste Zeile im Span des Aufrufers, die den Aufgerufenen nennt. Bei `--direction out` liegt diese Zeile in der Datei des Startsymbols und wird mit ihrem Pfad ausgegeben.
  - `--json`: Gibt maschinenlesbares JSON aus (`query.CallersAnswer`).
- **Exit-Codes**: `0` bei Erfolg; `1` bei fehlendem/unlesbarem Graph oder unbekanntem Symbol; `2` bei Aufruffehlern.

### `loomux graph skeleton <datei> [--json]`
Gibt alle Funktions-, Typ-, Interface- und Methodensignaturen sowie Zeilenspannen einer Datei ohne Rümpfe aus (~10x Token-Ersparnis).

- **Flags**:
  - `--json`: Gibt maschinenlesbares JSON aus (`skeleton.FileSkeleton`).
- **Exit-Codes**: `0` bei Erfolg; `1` bei fehlendem Graph oder Datei nicht im Graph; `2` bei Aufruffehlern.

### `loomux graph grep <muster> [-i] [--fixed] [--in <präfix>] [--max-hits <n>] [--json]`
Regex-Suche über indizierte Dateien, gruppiert nach umschließendem Symbol und sortiert nach Kopplungsgrad (`inDegree`).

- **Flags**:
  - `-i`: Regex-Suche ohne Beachtung von Groß-/Kleinschreibung.
  - `--fixed`: Behandelt das Muster als reinen Text (ohne Regex-Syntax).
  - `--in <präfix>`: Begrenzt die Suche auf Dateien unterhalb des Pfadpräfix.
  - `--max-hits <n>`: Maximale Anzahl von Zeilentreffern (Standard `300`); weitere Treffer werden nur gezählt.
  - `--json`: Gibt maschinenlesbares JSON aus (`grep.Result`).
- **Exit-Codes**: `0` bei Erfolg (auch bei 0 Treffern); `1` bei fehlendem/unlesbarem Graph oder ungültigem Regex; `2` bei Aufruffehlern.

### `loomux graph map [--max-dirs <n>] [--hubs-per-dir <n>] [--hotspots <n>] [--json]`
Zeigt token-budgetierte Verzeichnis-Cluster, lokale Hubs und globale Codebasis-Hotspots gerankt nach Kanten-Kopplung.

- **Flags**:
  - `--max-dirs <n>`: Maximale Anzahl von Verzeichnis-Clustern (Standard `16`).
  - `--hubs-per-dir <n>`: Höchstzahl der Hubs je Verzeichnis (Standard `3`).
  - `--hotspots <n>`: Höchstzahl der Hotspots im ganzen Repository (Standard `12`).
  - `--json`: Gibt maschinenlesbares JSON aus (`repomap.RepoMap`).
- **Exit-Codes**: `0` bei Erfolg; `1` bei fehlendem oder unlesbarem Graph; `2` bei Aufruffehlern.

### `loomux graph stats [--json]`
Gibt strukturelle Codebasis-Kennzahlen aus `.loomux/state/graph/wiring.json` aus: Knotenzahl, Kantenzahl gruppiert nach Relation, indizierte Dateien, Sprachen und Dateigröße.

- **Flags**:
  - `--json`: Gibt maschinenlesbares JSON aus (`query.StatsAnswer`).
- **Exit-Codes**: `0` bei Erfolg; `1` bei fehlendem oder unlesbarem Graph; `2` bei Aufruffehlern.

### `loomux graph check [--root <pfad>] [--json]`
Extrahiert den ganzen Baum neu und vergleicht ihn, Knoten für Knoten, mit dem auf der Platte geschriebenen Graphen.

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

Die fünf Datenbefehle lesen die Bereiche der einen Registry (`registry.toml` in `LOOMUX_STATE_DIR` oder dessen Plattformvorgabe) und antworten wie `brain-mcp` von ultra-brain; ein aufgezeichneter Fallkorpus (`testdata/cases/1b-1`) hält sie daran. Die Artefakte eines schreibgeschützten Bereichs (`index.md`, `graph.json`, `_identities.tsv`) und der Reconcile-Stempel werden zuerst aus dem Zustandsverzeichnis von loomux gelesen, wohin Stufe 3a sie schreibt, und aus dem Zustandsverzeichnis von ultra-brain, solange am neuen Ort nichts liegt: `LOOMUX_LEGACY_BRAIN_DIR`, Standard `%LOCALAPPDATA%\brain` unter Windows und `$XDG_STATE_HOME/brain` oder `~/.local/state/brain` unter POSIX. Es entscheidet das ganze Bereichsverzeichnis, nie eine einzelne Datei; `loomux migrate` (Stufe 4) zieht den Rest um. Bis Stufe 4 wird ein Bereichsverzeichnis, dessen `.loomux/config.toml` fehlt oder keine `[area]`-Tabelle trägt, über `.ultra-brain/config.toml` oder `.brain.toml` gelesen.

- **Kanal**: Jeder Befehl nimmt `--channel local|cloud` (Standard `local`). Ein Bereich mit `[privacy] mode = "local_only"` existiert im Kanal `cloud` nicht; `[privacy] never`-Globs gelten in jedem Kanal.
- **Usage-Fehler** (Exit `2`): die Usage-Zeile, dann `loomux brain <befehl>: error: <grund>` bei fehlendem Argument, ungültiger Wahl oder `-n` kleiner 1, und `loomux brain: error: <grund>`, wenn der Befehl fehlt oder unbekannt ist oder Argumente übrig bleiben.
- **Laufzeitfehler** (Exit `1`): `error: <grund>` auf `stderr` und nichts auf `stdout` — ein unbekannter Scope, eine Verweigerung, ein fehlender Abschnitt, ein kaputtes `graph.json` oder Identitätsregister, ein fehlendes oder unlesbares Manifest irgendeines registrierten Bereichs, eine fehlende oder kaputte Registry, eine nicht erreichbare Suchmaschine.
- **Ratschläge**: Die Meldungen nennen weiter `brain reindex`, `brain reconcile` und `brain embed`, die Befehle von ultra-brain, weil die aufgezeichneten Fälle von 1b-1 und 1b-2 diesen Wortlaut halten; mit dem Umstieg wechseln sie auf `loomux reindex`, `loomux reconcile` und `loomux embed`.

### `loomux brain search <anfrage> [--scope <scope>] [--profile fast|full|keyword] [-n <n>] [--channel local|cloud]`
Durchsucht die sichtbaren Bereiche (Standard `--scope all`) über den qmd-MCP-Daemon unter `http://localhost:8765/mcp`.

- **Profile**: `fast` (Standard) Vektorsuche ohne Reranking und ohne Anfrageerweiterung; `keyword` BM25-Keyword-Suche; `full` die hybride Kette mit Erweiterung und Reranking. `-n` (Standard `5`) muss mindestens 1 sein.
- **Daemon**: Antwortet dort niemand, startet loomux `qmd mcp --http --daemon --port 8765` entkoppelt, schreibt `note: starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm.` auf `stderr` und wartet bis zu 60 s auf ihn. Das Backbone ist standardmäßig CUDA; ein vom Nutzer gesetztes `QMD_LLAMA_GPU` oder `QMD_FORCE_CPU` bleibt unangetastet.
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
- **Exit-Codes**: `0`; `1` bei einer Verweigerung, einem Bereich ohne `graph.json` (``<scope>: never indexed; run `brain reindex` ``) oder einem anderen Laufzeitfehler; `2` bei einem Usage-Fehler.

### `loomux brain status [--channel local|cloud]`
Gibt aus, was man wissen muss, bevor man einer Antwort traut, eine Zeile je Befund.

- **Zeilen, in dieser Reihenfolge**: immer der letzte Abgleich (``last reconcile: never; run `brain reconcile` ``, ``last reconcile: <iso>; older than 24 h, run `brain reconcile` `` oder `last reconcile: <iso>`); je sichtbarem Bereich in Registry-Reihenfolge Include-Globs, die die Suchmaschine nicht sieht, ein Pfad, der nicht existiert, ein nie indizierter Bereich, weniger als die Hälfte aufgelöster Links und indizierte Dokumente, die die Suchmaschine nicht kennt; über alle sichtbaren Bereiche derselbe Inhalt unter mehreren Pfaden; einmal Dokumente, die indiziert, aber noch nicht durchsuchbar sind.
- **Suchmaschine**: Zwei Zeilen fragen die qmd-CLI (`qmd ls <collection>`, `qmd status`); antwortet sie nicht, sagt die Zeile das, und der Befehl läuft weiter.
- **Exit-Codes**: `0`; `1` bei einem Laufzeitfehler (Registry, Manifest, Stempel, `graph.json`, Identitätsregister); `2` bei einem Usage-Fehler.

### Wiki-Pflege: `loomux brain check`, `loomux lint`, `loomux wiki`

Seit Stufe 3c. Ein aufgezeichneter Fallkorpus (`testdata/cases/3c`) hält sie an der Referenz: `brain check` am Go-Binär von ultra-brain, denn eine Python-Form gibt es nicht, die übrigen an `brain-mcp`. Registry und Erklärungen kommen wie bei den Pflegebefehlen aus `LOOMUX_STATE_DIR`; die Erklärung eines Bereichs wird bis Stufe 4 auch unter `.ultra-brain/config.toml` und `.brain.toml` gelesen. Keiner nimmt `--state-dir`.

#### `loomux brain check file <pfad> | bundle --scope <scope> | all [--notes]`
Prüft Seiten nach den Achsen `okf` (was ein fremder Leser des Open Knowledge Format verlangt) und `house` (die strengeren Hausregeln samt Föderation: `wrong-direction`, `unlisted-area`).

- **Breiten**: `file` eine Seite ohne Nachbarn; `bundle` einen registrierten Bereich, gegen die ganze Registry; `all` jeden Bereich mit Wiki. Die vierte Breite der Referenz, `code`, gibt es nicht: die Code-Lanes gehören `loomux check`.
- **Ausgabe** auf `stdout`: je Befund `[<stufe>] <achse>/<regel> <scope>/<pfad>: <meldung>`, sortiert nach Scope, Pfad, Achse und Regel. Notizen erscheinen nur mit `--notes`, das nach der Breite an beliebiger Stelle stehen darf.
- **Exit-Codes**: `0` geprüft und ohne Fehler; `1` mindestens ein Fehler-Befund; `2` der Lauf fand nicht statt — keine oder eine unbekannte Breite, ein Pfad, der fehlt oder keine Datei ist, `bundle` ohne `--scope` oder mit `--scope all` (das ist die Breite `all`), ein unbekannter Scope, eine Registry, die sich nicht lesen lässt (`error: <grund>` auf `stderr`).

#### `loomux lint [<datei> | --file <datei>] [--scope all|<scope>] [--root <pfad>]`
Ohne Datei der Lint über die registrierten Bereiche nach den zwölf Regeln von `lint.py` der Referenz; mit einer Datei (einem Pfad, der eine Datei ist, oder einem Namen auf `.md`) die Einzelseite der Stufe 1a, deren Regeln auch `loomux wiki-gate`, die Lane `lint/wiki` und der post-edit-Hook fahren.

- **Über Bereiche**: `--scope all` (Standard) jeder Bereich mit Wiki-Pfad, sonst genau einer. Je Bereich eine Kopfzeile, darunter `  <pfad>:<regel>: <meldung>` oder `  no findings`; am Ende `no findings` oder `<n> findings (<e> errors, <w> warnings)`.
- **Regeln**, in der Reihenfolge der Ausgabe: `broken-frontmatter`/`missing-type`, `no-sources`, `orphan`, `unlisted-area`, `dead-link`, `outside-area` (Warnung), `wrong-direction`, `conflict-count`, `untouched` (Warnung), `stale`, `implemented-without-commit`, `long-planned` (Warnung). Die letzten beiden nur in Bereichen der Familie `project/`; `unlisted-area` nur im Wegweiser.
- **Weigerungen** (Exit `1`, `error: <grund>`, nichts auf `stdout`): ein unbekannter Scope, ein Bereich ohne Wiki-Pfad, ein Wiki-Pfad, der kein Verzeichnis ist (``… run `loomux wiki init --scope <scope>` first``, auch im Lauf über alle), eine Registry, die sich nicht lesen lässt. Eine Erklärung, die sich nicht lesen lässt, beendet den Lauf dort, wo er steht.
- **Exit-Codes**: `0` ohne Fehler-Befund, auch mit Warnungen; `1` mit mindestens einem Fehler-Befund oder einer Weigerung; `2` bei einem Usage-Fehler.

#### `loomux wiki init --scope <scope>`
Legt das Gerüst des Wiki-Bündels eines Bereichs an (`_schema.md`, `index.md`, `log.md`, `audit.md`, `_identities.tsv`) und nennt jede geschriebene Datei. Eine vorhandene Datei bleibt, wie sie ist. Exit `1` bei einem unbekannten Scope, einem Bereich ohne Wiki-Pfad oder einem schreibgeschützten Bereich.

#### `loomux wiki types`
Zählt die Seitentypen über alle Bereiche mit Wiki: je Typ `<typ> [<rang>]: <summe> (<scope>: <n>, …)`, nach Summe absteigend, dann nach Name. Der Rang ist `core`, `catalogue`, `origin`, `declared` oder `unknown`; ein unbekannter Typ trägt das Präfix `? `, ein bekannter Altname ` -> <katalogname>`. Über Bereiche zählt der schlechteste Rang. Exit `0`, außer die Registry, eine Erklärung oder eine Seite lässt sich nicht lesen (`1`).

#### `loomux wiki retype --scope <scope> --from <alt> --to <neu>`
Benennt einen Seitentyp in einem Bündel um und nennt jede geschriebene Seite. Geändert wird nur die Zeile `type:` des Frontmatters; eine geschriebene Seite wird ganz auf LF gefaltet. Übersprungen werden Gerüstdateien, kaputtes Frontmatter (auch ein doppelter Schlüssel), Bytes, die kein UTF-8 sind, und ein gequoteter oder gefalteter Wert. Ein Zieltyp, den kein Rang kennt, ergibt eine Warnung auf `stderr`, der Lauf geht weiter. Exit `1` bei einem unbekannten Scope, einem Bereich ohne Wiki-Pfad oder einem schreibgeschützten Bereich.

### Pflege: `loomux reindex`, `loomux embed`, `loomux reconcile`, `loomux area add`

Vier Befehle der `brain`-CLI von ultra-brain, seit Stufe 3a Befehle auf oberster Ebene von loomux; ein aufgezeichneter Fallkorpus (`testdata/cases/3a`) hält sie an der Python-Referenz. Sie schreiben nur ins Zustandsverzeichnis von loomux und lesen das alte als den oben beschriebenen Rückfall.

- **Umgebung**: `LOOMUX_STATE_DIR` hält die Registry, die Artefakte schreibgeschützter Bereiche, `maintenance/` und `qmd-collections.json`; `LOOMUX_LEGACY_BRAIN_DIR` ist der Rückfall und wird nie beschrieben. qmds `index.yml` wird über `XDG_CONFIG_HOME` gefunden, sonst unter `~/.config`.
- **Kein `--state-dir`**: Die Referenz nimmt es an allen vieren an; loomux lehnt es ab wie jede unbekannte Flagge (Exit `2`). Der Zustand kommt aus der Umgebung, dem einen Zustandsmodell aller loomux-Befehle.
- **Positionale Argumente** (Exit `2`): Keiner der vier nimmt eines. Ein Wort, das nach den Flaggen übrig bleibt, wird mit `<befehl>: unrecognized arguments: <wörter>` abgelehnt, bevor Umgebung oder qmd gefragt werden.
- **Meldungen** des Abgleichs sind deutsch, wörtlich die der Referenz.

#### `loomux reindex [--registry <pfad>]`
Fährt einen Abgleich über die registrierten Bereiche, baut danach je Bereich die Verzeichniskataloge (`index.md`), den Linkgraphen (`graph.json`) und das Identitätsregister (`_identities.tsv`) neu und trägt die Bereiche als Sammlungen in qmds `index.yml` ein. Ein schreibbarer Bereich hält seine Artefakte im eigenen Baum; die eines schreibgeschützten werden über ein Staging-Verzeichnis nach `<zustand>/areas/<scope>/` geschrieben und als Ganzes eingetauscht.

- **`--registry`**: eine `registry.toml` oder das Verzeichnis, das eine hält; Standard ist `registry.toml` im Zustandsverzeichnis.
- **Aufholung**: Der Abgleich läuft zuerst, damit eine geänderte Quelle zum Fall wird, bevor der Indexlauf ihren Hash fortschreibt. Fälle, die er eröffnet, stehen auf `stderr`, und der Lauf **geht weiter**; ein Tresor ohne Prüfzentrum bekommt eine Warnung und wird indiziert; jeder andere Fehlschlag des Abgleichs beendet den Befehl, bevor etwas indiziert ist.
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
- **Stempel**: Der Abgleich schreibt `maintenance/last-run.txt` in UTC, den `brain status` und `brain search` lesen.
- **Exit-Codes**: `0` für einen Abgleich, der bis zum Ende lief; `1`, wenn sich eine Falldatei nicht lesen lässt (`unreadable case: <eintrag>` auf `stderr`), sowie bei einer Registry, die sich nicht lesen lässt, einem Tresor, der kein oder zwei Prüfzentren erklärt, oder einem anderen Fehlschlag (`error: <grund>`); `2` bei einem Usage-Fehler.

#### `loomux area add [--path P] [--scope S] [--wiki W] [--sources S] [--merge-branch B] [--privacy M] [--no-reindex] [-y|--yes]`
Meldet ein Repository als Bereich an und richtet es ein: der Registry-Eintrag (unter einer Sperre geschrieben); `.loomux/config.toml` mit `[area]`, `[layout]`, `[index]`, `[privacy]` und `[maintenance]`, wenn das Repository keine hat; die Routing-Regel, einmal an `AGENTS.md` angehängt; das Gerüst des Wiki-Bündels; danach `loomux reindex` samt Aufholung.

- **Vorgaben**: `--path` das Arbeitsverzeichnis; `--scope` `project/<verzeichnisname>`; `--sources` `docs`, wenn es ein Verzeichnis `docs/` gibt, sonst `.`; `--wiki` `<repo>/docs/wiki` oder `<repo>/wiki` (muss absolut sein); `--merge-branch` der Branch, den git nennt, ohne einen `master`; `--privacy` `manual_cloud`, einer von `automatic_cloud`, `local_only`, `manual_cloud`.
- **Registry zuerst**: Ein schon registrierter Scope wird abgelehnt, bevor das Repository berührt wird.
- **Eine behaltene Konfiguration**: Eine vorhandene `.loomux/config.toml` bleibt Byte für Byte stehen, mit einer Warnung, wenn sie kein `[area]` oder einen anderen Scope erklärt. Eine, die der Deklarationsleser ablehnt, beendet den Befehl, ohne dass etwas registriert ist.
- **Unterschiede zu `brain init`**: kein `.mcp.json` und keine Agenten-Hooks (`loomux init`, Stufe 4); der Indexlauf findet wirklich statt, außer mit `--no-reindex`; der Branch wird als `[maintenance] branch` geschrieben, nicht als `merge_branch`; `--privacy` wird geprüft; der erste Bereich einer Maschine braucht keine von Hand angelegte Registry-Datei. `-y`/`--yes` wird angenommen und ändert nichts.
- **Exit-Codes**: `0` oder der Exit-Code des Indexlaufs; `1` bei einem Pfad, der kein Verzeichnis ist, einem ungültigen Scope, einem relativen `--wiki`, einem abgelehnten Registry-Eintrag, einer unlesbaren Datei oder einem fehlgeschlagenen Schreiben; `2` bei einem Usage-Fehler, einem fehlenden oder unbekannten Unterbefehl (mit der Usage-Zeile) oder einem unbekannten `--privacy`.

### Der post-merge-Hook: `loomux merge-hook install|status|remove|record`
Der Hook, der `reconcile` einen gelandeten Merge meldet, in jedem Repository eines Bereichs, dessen Manifest `[maintenance] on_merge = true` sagt. `brain-mcp hook` von ultra-brain unter neuem Namen, weil `hook` hier der Namensraum der Host-Hooks ist; ein aufgenommener Fallkorpus (`testdata/cases/4a2`, 14 Fälle, elf ohne Unterschied) hält ihn an der Referenz. `loomux init` ruft `merge-hook install` als seinen Teil `merge-hook` (aus in einem Checkout von loomux, dessen Hookverzeichnis das eingecheckte `.githooks` ist).

- **Der Hook backt nichts ein**: Die Datei ist überall derselbe Text, ein kurzes `sh`, das `"${LOCALAPPDATA}/loomux/bin/loomux.exe" merge-hook record` ruft und dessen Ausgabe verwirft. Welches Repository und welcher Zweig zählen, entscheidet die Registry zur Merge-Zeit; nichts in der Datei kann veralten. Sie liegt, wo git Hooks sucht, `core.hooksPath` eingeschlossen. Die Einrichtungen merkt sich `maintenance/hooks.tsv` unter `LOOMUX_STATE_DIR` (drei Felder, `scope`, `repo`, `hook`; eine Zeile der Referenz mit mehr wird über ihre ersten drei gelesen).
- **`install`** schreibt den Hook in jedes einwilligende Repository und merkt ihn sich; eine Datei mit der Marke der Referenz `# brain post-merge hook` ist der Vorgänger desselben Hooks und wird ersetzt. Ein fremder `post-merge` heißt `refused` und bleibt stehen.
- **`status`** nennt den Zustand jedes Repositorys: `installed`, `missing` (gemerkt, Datei weg), `not installed`, `unrecorded` (die Datei ist unsere, der Eintrag verloren — auch ein Hook, den `brain-mcp` eingerichtet hat, dessen Einträge unter dem alten Zustandsverzeichnis nicht gelesen werden), `orphaned` (gemerkt für einen Bereich, der nicht mehr einwilligt, oder ein verschobenes Repository) und `refused`; jeder Befehl sagt `no repository` für einen einwilligenden Bereich, dessen Pfad in keinem liegt.
- **`remove`** nimmt jeden eigenen Hook zurück, den er kennt, einen ohne Eintrag eingeschlossen; einen, den jemand durch einen eigenen ersetzt hat, nennt er `refused` und behält dessen Eintrag.
- **Ausgabe** auf `stdout`: eine Zeile je Repository, `<zustand>: <scope> — <repo>`, gefolgt von ` [<hookdatei>]`, wenn es eine gibt. Ohne Zeile: `no area consents with [maintenance] on_merge = true, and no hook is installed`.
- **`record`** ruft der Hook selbst: ein Ereignis für den Merge, der eben im Arbeitsverzeichnis gelandet ist, wenn ein Bereich es will. Es läuft im `git merge` des Nutzers, schreibt darum nie etwas und endet immer mit `0` — keine Registry, eine kaputte, kein Repository, ein nicht schreibbares Zustandsverzeichnis und überzählige Argumente eingeschlossen.
- **Der Wächter** verweigert einem Agenten `install` und `remove`, die ausführbare Dateien in Repositorys schreiben; `status` und `record` gehen durch (siehe [`hook pre-tool-use`](#loomux-hook-pre-tool-use)).
- **Exit-Codes**: `0`, auch bei einem verwaisten, fehlenden oder unverzeichneten Hook, die Befunde sind; `1`, wenn eine Zeile `refused` oder `no repository` sagt, die Registry nicht liest oder eine Hookdatei nicht geschrieben werden kann (`loomux merge-hook <sub>: <grund>` auf `stderr`); `2` bei einem Usage-Fehler (`usage: loomux merge-hook install|status|remove|record`).

### Prüfzentrum: `loomux cases`, `loomux case`, `loomux approve`

Drei Befehle der `brain`-CLI von ultra-brain, seit Stufe 3b Befehle auf oberster Ebene von loomux; ein aufgezeichneter Fallkorpus (`testdata/cases/3b`) hält sie an der Python-Referenz, `approve` samt den Dateien, die es schreibt, und dem Commit, den es anlegt. Sie entscheiden die Fälle, die `loomux reconcile` ins Prüfzentrum legt. Ein Fall wird über den Namen seines Verzeichnisses im Prüfzentrum angesprochen, die erste Spalte von `loomux cases`.

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
- **Eine Freigabe** prüft zuerst, dass sich weder die Zielseite noch eine zitierte Quelle seit dem Fall geändert hat, schreibt dann die Seite mit fortgeschriebener Frontmatter (`generated`, `verified` mit dem Prüfer), schiebt die Identitätsregister vor, hängt an `log.md` und `audit.md` an, entfernt das Fallverzeichnis und committet genau diese Pfade über einen eigenen Index (`<zustand>/maintenance/index`) auf den aktuellen Ref des Tresors; der Index des Nutzers bleibt unberührt. Danach laufen eine Aufholung und ein Indexlauf, dieser ohne eigene Aufholung. Scheitert die Aufholung, sagt eine Warnung das, und es wird nicht indiziert; scheitert der Indexlauf, nennt eine Warnung `loomux reindex`. Keines von beiden ändert den Exit-Code.
- **Eine Ablehnung** hängt an `audit.md` an, entfernt das Fallverzeichnis und committet beides. Revision und Hash der Seite schiebt sie **nicht** vor, also eröffnet der nächste `loomux reconcile` denselben Fall wieder — von der Referenz geerbt.
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
  danach alle 24 Stunden, solange er läuft; nie `reindex`. Jedes
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

### `loomux self-update`

Ein Self-Update-Durchlauf von Hand; `serve` fährt denselben eine Minute nach
dem Start und danach alle 24 Stunden. Er wirkt nur auf das maschinenweite
Binary, `<Zustandsverzeichnis>/bin/loomux.exe`, wenn das das laufende Binary
ist und eine Release-Version trägt.

1. Listet die Releases über `gh release list` und nimmt die höchste Version
   im Kanal des laufenden Binarys (`beta` nimmt Prereleases, `stable` nicht).
2. Lädt das Windows-Asset und `SHA256SUMS` über `gh release download`, prüft
   die Prüfsumme und die `--version` des neuen Binarys.
3. Stempelt die Datei mit der aktuellen Zeit und tauscht sie ein; das alte
   Binary kommt nach `loomux.old.exe` oder in den ersten freien nummerierten
   Platz. Die nächste Brücke ersetzt den laufenden `serve`.

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
| 2 | ausgelassen: nicht Windows, ein Entwicklungs-Build oder nicht das maschinenweite Binary; oder ein unbekanntes Argument |

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
Mutiert die Go-Entscheidungen jedes Pakets und meldet, welche Mutanten seine Testsuite nicht bemerkt — ein Port von ultra-brains `tools/go_mutants.py`.

- **Familien**: `a1` die ganze `if`-Bedingung als `true` und als `false`; `a2` jeder Operand eines `&&` oder `||` auf oberster Ebene für sich; `a3` jeder Vergleichsoperator gekippt (`==`/`!=`, jede Ordnung gegen ihren Nachbarn), nicht in Kommentaren oder Zeichenketten; `a4` die Bedingung negiert. `for`-Bedingungen werden nie mutiert.
- **Mechanik**: Jeder Mutant erreicht `go test -overlay <json> -count=1 -failfast -timeout 60s ./<paket>/` über ein Overlay in einem temporären Verzeichnis; der Arbeitsbaum wird nie beschrieben. Jedes Overlay-Verzeichnis wird nach seinem Lauf entfernt, auch nach einem Fehler oder Strg+C. `--workers` fährt so viele Läufe gleichzeitig (Vorgabe: die Hälfte der Prozessoren, mindestens 1). `--only` behält Dateien, deren Name den Text enthält.
- **Bericht**: je Mutant eine Zeile — `killed`, `SURVIVED` oder `no mutant` (kompiliert nicht oder ändert nichts) — dann die Summen und die Überlebenden. Ein Lauf, der die Zeitgrenze reißt, gilt als getötet.
- **Exit-Codes**: `0` nach einer vollständigen Runde, auch mit Überlebenden; `2` bei einem Usage-Fehler, einem Paket ohne Quelldateien oder einer Suite, die vor dem ersten Mutanten nicht grün ist; `1`, wenn ein Lauf nicht gestartet werden kann oder die Runde mit Strg+C abgebrochen wird.

### `loomux dev swap-binary --dir <bin>`
Tauscht das laufende `loomux.exe`-Binary atomar gegen `loomux.new.exe` aus (löst Windows Dateisperren-Konflikte). Das abgelöste bleibt als `loomux.old.exe` liegen, oder als erstes freies `loomux.old.<n>.exe` daneben, wenn ein Prozess aus einem früheren Tausch — ein `loomux serve` oder eine Brücke — diesen Namen noch hält; jeder Platz, dessen Prozess beendet ist, wird beim nächsten Tausch geräumt, es bleiben also höchstens 16 Generationen liegen. Zwei Fälle scheitern weiterhin, und beide lassen die Binaries dort, wo sie waren: alle 16 Plätze gleichzeitig gehalten, und ein `loomux.exe`, das etwas so hält, dass es sich gar nicht umbenennen lässt — ein laufendes `loomux.exe` ist dieser Halter nicht, denn Windows lässt ein laufendes Abbild umbenennen.

### `loomux dev bench [--dir <dir>] [--corpus <datei>] [--languages <n>] [--tier <kategorie>] [--warm <n>] [--cache-dir <dir>] [--out <datei>] [--json-out <datei>] [--component-timeout <d>] [--save] [--report-dir <dir>]`
Führt umfassende Latenz-Benchmarks und normalisierte Lücken-Audits (Gap Analysis) für ein Einzel-Repository oder das gesamte Open-Source-Matrix-Korpus durch (1x kalt + Nx warmer Median, Min, Max).

- **Wie die Hooks gemessen werden**: Jeder Hook bekommt eine Claude-Code-Nutzlast für einen Edit an einer Beispieldatei der Hauptsprache des Repositorys, `post-tool-use` fährt also seine echten Lanes. Die Status-Spalte nennt die Exit-Codes aller Läufe.
- **Einzel-Repository-Modus** (Standard): Misst `pre-tool-use`, `post-tool-use` und `graph build` (bei Go-Projekten), vergleicht mit bestehenden Claude-Hooks (Speedup) und prüft Lücken zwischen nativen Werkzeugen und Loomux-Lanes.
- **Korpus-Modus** (`--corpus <pfad>`): Klont und benchmarkt die Top-N Open-Source-Projekte über Sprachen und Frameworks hinweg und liefert einen aggregierten Performance- und Lückenbericht.
- **Flags**:
  - `--dir <pfad>`: Ziel-Repository (Standard: `.`).
  - `--corpus <pfad>`: Pfad zur Open-Source-Matrix-Markdown-Datei.
  - `--languages <n>`: Anzahl der Sprachen im Korpus (Standard: `5`).
  - `--tier <kategorie>`: Filter für Sterne-Kategorie (Standard: `"Sehr viel"`).
  - `--warm <n>`: Anzahl warmer Messläufe für die Median-Berechnung (Standard: `3`).
  - `--component-timeout <d>`: Frist je gemessenem Befehl; ein Befehl darüber wird beendet und als `timeout` gemeldet (Standard: `60s`).
  - `--cache-dir <pfad>`: Verzeichnis für geklonte Repositories (Standard: `.cache/benchcorpus`).
  - `--out <pfad>`: Schreibt Markdown-Bericht in Datei (Standard: stdout).
  - `--json-out <pfad>`: Schreibt maschinenlesbaren JSON-Bericht in Datei.
  - `--save`: Speichert Benchmark-Berichte automatisch in Sprachunterordnern (`docs/{en,de}/benchmarks/<sprache>/<slug>.md`) und aktualisiert die zentrale Gesamt-Matrix (`docs/{en,de}/benchmarks/matrix.md`).
  - `--report-dir <pfad>`: Dokumentations-Stammverzeichnis für gespeicherte Berichte (Standard: `docs`).

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
# jede Form nimmt auch --root <verz> oder --global, hinter dem Unterbefehl
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
- **Flags** (hinter dem Unterbefehl; ein Flag davor ist ein Bedienfehler):
  - `--root <verz>` — das Projekt; leer wird es vom Arbeitsverzeichnis aus
    nach oben gesucht.
  - `--global` — stattdessen die rechnerweite `config.toml` im
    Zustandsverzeichnis. Sie kennt noch keinen Schlüssel: `list` gibt nichts
    aus (`[]` mit `--json`), `get`, `set` und `unset` weisen jeden Schlüssel als unbekannt ab. `--global`
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
- **Herkunft**: `set` (in der Datei), `default` (die Vorgabe des Lesers, als
  Wert gezeigt), `preset` (`verify.profiles`, von den Presets gefüllt; gezeigt
  wird der eingebaute Wert) und `unset` (kein Wert und keine Vorgabe).

### `loomux config list [--json]`
Eine Zeile je Schlüssel: Modul, Schlüssel, Wert, Herkunft. Eine Liste von
Tabellen (`commit.allow`, `policy.paths.rules`, `policy.commands.rules`)
zeigt statt eines Werts die Zahl ihrer Einträge. Sind Vorschläge offen,
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
fragt `write these changes? [y/N]` und schreibt nur bei `y`.

- **Werte** werden getippt, wie ein Mensch sie schreibt: eine Zeichenkette
  ohne Anführungszeichen, eine Zahl, `true`/`false`, eine Liste als `a, b`.
  Ein Komma in einer `{…}`-Gruppe gehört zum Eintrag, `docs/**/*.{md,txt}, src`
  sind also zwei Globs. Ein Aufzählungsschlüssel nimmt nur die Werte, die
  sein Leser annimmt. `verify.timeout` sind ganze Sekunden (`600`, nicht
  `10m`). Eine Zahl wird in ihrer schlichten Form geschrieben: `+600` und
  `0600` sind `600`. Das Wort `default` ist ein Wert wie jeder andere.
- **Vorgaben werden nie geschrieben**: `set` auf den Vorgabewert entfernt die
  Zeile, und ein Abschnitt, der dadurch leer wird, geht mit. Eine Datei, die eine Vorgabe wiederholt, hielte sie gegen eine
  spätere Änderung der Vorgabe fest.
- **Geprüft von den echten Lesern**: Der neue Text geht an jeden Leser, der
  im Betrieb läuft (Bereichsdeklaration, `[modules]`, Policy, `[verify]`,
  Commit-Policy, Worktree-Spiegel), und wird erst geschrieben, wenn alle ihn
  annehmen.
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
beim Namen und schreibt nur, was ein Mensch bestätigt. Es ersetzt `ulinit`
aus ultraloom, `scripts/install.ps1` und die Hook-Hälfte von `brain init`.
Gebaut mit Stufe 4a-2; der Lauf auf einem frischen Klon dieses Repositorys
und in einem Wirtsprojekt steht für einen Menschen noch aus (siehe den
[Migrationsplan](migration.md)).

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
  Antworten eines früheren Laufs.
- **`--hosts`**: kommagetrennt, `claude`, `antigravity` oder `codex`;
  vorgegeben sind die Wirte, die das Projekt hat (`.claude/` → Claude Code,
  `.agents/` → Antigravity; keines → Claude Code).
- `--dry-run=…` und `--detect-only=…` sind ein Usage-Fehler, und kein Flag
  nimmt ein folgendes `--dry-run` als Wert: Der Wächter lässt eine Zeile mit
  dem Wort `--dry-run` durch, ein Wert könnte das zurücknehmen.

### Module und Teile
Das Interview fragt je Modul `all`, `each` oder `none` (bei `each` eine
Vollbildliste seiner Teile), dann die Commit-Sprache (`en` oder `de`) und,
wenn das Projekt ein Bereich wird, seinen Scope. Die Vorgaben der Teile
folgen dem Projekt; die Antworten eines früheren Laufs gehen vor.

| Modul | Teil | Was er tut | Vorgabe |
|---|---|---|---|
| base | `binary` | das Binary, das die Einträge rufen: `binary-install` legt das neueste Release nach `${LOCALAPPDATA}/loomux/bin/loomux.exe` (über `gh`, geprüft gegen `SHA256SUMS` und sein `--version`); in einem Checkout von loomux baut `binary-build` `bin/loomux.exe` | an |
| base | `config` | `.loomux/config.toml`: `[modules]`, wo ein Modul aus ist, `[commit] language`, wo sie nicht `en` ist, und die noch fehlenden Policy-Regeln der erkannten Stacks; `[verify]` bleibt den Presets. Der Text muss die eigenen Leser der Konfiguration bestehen | an |
| base | `gitignore` | `.gitignore`: `/.loomux/state/` | an |
| base | `agents-md` | `AGENTS.md`, nur wenn das Projekt keine hat | an, aus in einem Checkout |
| base | `mcp-json` | `.mcp.json` mit dem Server `loomux` (siehe [Die `.mcp.json` eines Wirts](#die-mcpjson-eines-wirts)) | an, aus in einem Checkout |
| base | `tools` | sucht `git`, `qmd`, `pdftotext`, `yt-dlp` und `ollama` auf dem `PATH` und nennt für ein fehlendes den Installationsbefehl; installiert nichts | an |
| hooks | `host-entries` | die Hook-Einträge jedes Wirts (`.claude/settings.json`) | an |
| hooks | `git-hooks` | `pre-commit`, `pre-push` (verweigert einen Push nach `main` oder `master`) und `commit-msg` unter `.githooks`, dazu `git config core.hooksPath .githooks` | an in einem Repository |
| hooks | `verify-skill` | der Skill `verify-until-green` | an, aus in einem Checkout |
| brain | `area` | `loomux area add --scope <scope>`, ohne `--wiki`, also mit dem vorgegebenen Wiki von area add | an, aus in einem Checkout oder bei einem schon erklärten oder registrierten Bereich |
| brain | `merge-hook` | der post-merge-Hook von `loomux merge-hook install`, nur für die Bereiche an dieser Wurzel: Ein veralteter Bereich oder ein fremder Hook anderswo in der Registry hält `init` nicht auf. Geplant nur, wenn hier ein Bereich der Registry dieses Rechners steht und die Erklärung hier `[maintenance] on_merge = true` sagt, oder wenn `area` im selben Lauf in einem Projekt ohne `.loomux/config.toml` läuft (area add schreibt die Zustimmung nur in eine neue), und nur, wo `${LOCALAPPDATA}/loomux/bin/loomux.exe`, das der Hook ruft, installiert ist oder `binary-install` läuft, auch in einem Checkout; ohne eine Zeile für diese Wurzel scheitert die Handlung | an in einem Repository, aus in einem Checkout |
| brain | `brain-skills` | die Skills `brain-ingest`, `brain-land`, `brain-research`, `brain-review`, `brain-wiki-plan` | an, aus in einem Checkout |
| graph | `graph-build` | `loomux graph build` | an bei einem bekannten Stack, aus in einem Checkout |

Ein Checkout von loomux (sein `go.mod` erklärt `github.com/xidus90/loomux`)
bekommt als Vorgabe nur an, was seine eingecheckten Dateien schon haben,
damit `init --yes` auf einem frischen Klon `git status` leer lässt; ein Lauf
durch einen Menschen, der das prüft, steht noch aus.

### Was es schreibt und was es stehen lässt
- **Nie überschreiben, stattdessen melden.** Eine neue Datei entsteht
  exklusiv; eine vorhandene ändert sich nur, wo sie `init` gehört. Ein
  Host-Eintrag gehört `init`, wenn sein Befehl ein loomux-Binary ruft; ein
  fremder bleibt stehen und wird genannt. Eine Host-Datei oder `.mcp.json`,
  die kein JSON ist oder deren `hooks` oder `mcpServers` kein Objekt ist,
  wird nicht repariert: Der Plan scheitert, und der Lauf endet, ohne etwas
  zu schreiben. Eine vorhandene `AGENTS.md` oder ein vorhandener Skill
  bleibt, wie er ist. Bevor sich eine Datei zum ersten Mal ändert, geht eine
  Kopie nach `.loomux/state/backup/<pfad>.bak` — außer
  `.loomux/config.toml`, die im Ganzen ersetzt wird, sobald ihre Leser den
  neuen Text annehmen.
- **Einträge rufen das Binary an seinem Ort**: Die Einträge eines
  Wirtsprojekts rufen `"${LOCALAPPDATA}/loomux/bin/loomux.exe"`, die eines
  Checkouts `"${CLAUDE_PROJECT_DIR}/bin/loomux.exe"` (seine Git-Hooks
  `./bin/loomux.exe`). Steht dort kein Binary — der Teil `binary` ist aus,
  abgelehnt oder gescheitert —, entfallen Host-Einträge, Git-Hooks und der
  Merge-Hook, und `init` sagt es. `binary-install` scheitert, wenn
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
  `loomux self-update`), `binary-build` ohne `bin/loomux.exe`, `merge-hook`
  ohne eigenen `post-merge`, `graph-build` ohne Graph.
- **Lebende Hooks behalten ihr Verzeichnis**: Ist kein `core.hooksPath`
  gesetzt und liegt in `.git/hooks` ein lebender `pre-commit`, `pre-push`
  oder `commit-msg`, schreibt `init` nur die fehlenden Hooks dorthin. Ein
  Hookverzeichnis außerhalb der Wurzel (ein verknüpfter Worktree, ein
  Submodul) bleibt mit einer Notiz unberührt.
- **Antigravity** wird angenommen, bekommt aber noch keine Hook-Einträge und
  keine Skills: Seine Hookdatei ist gegen einen laufenden Agenten noch nicht
  gemessen, ebenso wenig, wie es `${LOCALAPPDATA}` auflöst. Der Plan sagt das
  in einer Notiz, und `.agents/hooks.json` wird weder gelesen noch
  abgelehnt. Codex hat keine Hookdatei.
- **Zustand**: `.loomux/state/answers.toml` hält die gewählten Wirte und
  Teile (alles andere steht in `.loomux/config.toml`);
  `.loomux/state/installed.toml` nennt, was der letzte Lauf geschrieben und
  ausgeführt hat, und wird zuletzt geschrieben, sodass ein abgebrochener Lauf
  seine offenen Änderungen im nächsten Plan wieder zeigt.

### Ausgabe
Der Plan auf `stdout`: jede Änderung als `--- <pfad>` mit Diff, dann
`actions:` mit einer Zeile je Handlung, dann `notes:`; `nothing to change`,
wenn nichts ansteht. Nach einem Lauf eine Zeile je Pfad oder Handlung:
`written: …`, `skipped: …`, `refused: …` (abgelehnt), `failed: …`.

### Der Wächter
Der Wächter verweigert einem Agenten `loomux init`, außer es trägt
`--dry-run` oder `--detect-only` als eigenes Wort (siehe
[`hook pre-tool-use`](#loomux-hook-pre-tool-use)). Ein Mensch ruft es.

### Exit-Codes
`0` Erfolg, ein Probelauf, `--detect-only`, ein mit `esc` oder
Eingabeende abgebrochener Lauf (`loomux init: cancelled; nothing written`)
und ein Lauf, in dem der Mensch eine Änderung oder Handlung abgelehnt oder
einen Teil abgeschaltet hat, das Binary eingeschlossen, auch wenn dadurch
Host-Einträge, Git-Hooks und Merge-Hook entfallen; `1` das Projekt lässt
sich nicht lesen (`go.mod`, git, die Registry), der Binary-Schritt, eine
Änderung oder Handlung ist gescheitert, das Terminal versagte im Interview
oder bei der Bestätigung, oder es ließ sich nicht zurücksetzen; `2`, bevor
etwas geschrieben ist: ein Usage-Fehler, eine Wurzel, die kein Verzeichnis
ist, eine `.loomux/config.toml` oder `.claude/settings.json`, die sich nicht
lesen lässt, eine `answers.toml`, die
nicht liest, ein scheiternder Plan (eine Konfiguration, die ihre Leser
ablehnen, eine Host-Datei oder `.mcp.json`, die kein JSON ist oder deren
`hooks` oder `mcpServers` kein Objekt ist, jede Datei, die der Plan nicht
lesen kann), und ein Lauf, der fragen muss und kein Terminal hat (`init asks questions; run it in a
terminal, or pass --yes or --dry-run`).


