# Policy-Baukasten — Entwurf

Stand: 2026-08-25. Status: zur Durchsicht.

## Warum

Regeln darüber, was ein Agent nicht anfassen darf, stehen heute als Prosa in
CLAUDE.md-Dateien oder als handgeschriebene Hook-Skripte in einzelnen Projekten
(`iam_backend/.claude/hooks/guard_paths.py`). Prosa erzwingt nichts, und ein
Skript pro Repo driftet. ultraloom prüft bereits, ob ein Projekt grün ist; die
Policy beantwortet die zweite Frage derselben Art — ob ein Werkzeugaufruf
überhaupt stattfinden darf — und zwar überall mit demselben Werkzeug.

Zwei Regelverstöße sind in diesem Repo bereits eingetreten und stehen als
Warnung in CLAUDE.md: ein Subagent hat `master` nach `origin` gepusht, ohne es
zu berichten, und ein Verzeichnis unter `.claude/worktrees/` teilt Index und
HEAD mit dem Hauptcheckout, so dass `git add` schweigend nichts tut. Beide sind
mechanisch prüfbar und haben heute keine Sperre.

### Warum nicht "guard"

Der Name ist vergeben. `flows/verify_until_green.py` hat einen Knoten `guard`
und ein `make_guard`, die messen, was ein Reparateur angefasst hat, gegen das,
was er anfassen darf; dazu gehören ein eigener Spec und ein eigener Plan
(`2026-08-23-guard-basis-commit`). Ein zweites `guard` mit anderer Bedeutung
wäre in jeder Fehlermeldung und in jedem Gespräch über dieses Repo eine
Rückfrage wert.

## Was gebaut wird

Ein Regel-Baukasten mit drei Regelarten (Pfade, Kommandos, Dateiinhalte), einer
Konfigurationsfläche in `.ultraloom/config.toml`, eingebauten
sicherheitsrelevanten Voreinstellungen und einem Adapter für Claude Codes
Hook-Protokoll.

Ausdrücklich nicht gebaut: Regeln gegen Werkzeugnamen selbst. Das leisten
Claude Codes eigene `permissions`, und eine zweite Stelle mit derselben
Zuständigkeit wäre eine Quelle für Widersprüche.

## Architektur

Drei Schichten, jede für sich testbar:

**`ultraloom.policy.rules`** — die Engine. Nimmt ein `Ruleset` und ein `Subject`
(`kind`, `value`, `tool`) und liefert ein `Verdict`: erlaubt, oder verweigert
mit den Begründungen, die der Agent liest. Kennt weder Dateien noch Prozesse
noch JSON und importiert nur die Standardbibliothek.

**`ultraloom.policy.config`** — liest `[policy.*]` aus `.ultraloom/config.toml`
und baut das `Ruleset`, einschließlich der eingebauten Voreinstellungen. Die
eingebauten Regeln stehen als Konstante im Modul, nicht in einer mitgelieferten
TOML-Datei: eine Datei kann fehlen, eine Konstante nicht.

**`ultraloom.policy.hook`** — der Adapter. Liest Claude Codes Payload von stdin,
leitet aus `tool_name` und `tool_input` die zu prüfenden Subjects ab und
übersetzt das Verdikt in Exit-Codes. Die einzige Stelle im Repo, die weiß, wie
Claude Code spricht.

### Modulgrenze

`ultraloom.policy.*` darf `ultraloom.config` benutzen und sonst nichts aus dem
Harness (`graph`, `state`, `runner`, `journal`, `gate`, `model`, `discovery`)
und nichts aus `checks`. `tests/test_module_boundary.py` bekommt einen zweiten
Lauf, der `ultraloom policy hook` in einem Kindprozess ausführt und dessen
`sys.modules` zurückliest.

### CLI

    ultraloom policy hook                 # liest die Payload von stdin
    ultraloom policy check <art> <wert>   # für Hand und Skript

## Schema

    [policy.paths]
    mode = "deny"        # "allow" dreht um: nur Genanntes ist schreibbar
    defaults = true      # false wirft die eingebauten Sperren ab

    [[policy.paths.rules]]
    match  = [".ultraloom/runs/*", "uv.lock"]
    reason = "An edited journal destroys what replay exists for."

    [[policy.commands.rules]]
    regex  = "^git\\s+push\\b"
    reason = "Whether commits reach the remote is a human's decision."

    [[policy.content.rules]]
    regex  = "type:\\s*ignore(?!\\s*#)"
    tools  = ["Write", "Edit"]
    reason = "No type: ignore without a reason behind it."

Die Begründungen stehen englisch in der Konfiguration, weil sie als
Fehlermeldung beim Agenten landen und dieses Projekt seine Meldungen englisch
schreibt.

### Entscheidungen im Schema

**Modus je Regelart, nicht global.** Ein globales `mode = "allow"` würde mit dem
Umdrehen der Pfadregeln auch jedes nicht genannte Kommando verbieten. Der
Allow-Modus ist für Pfade nützlich und für Kommandos praktisch unbenutzbar.

**`match` (Glob) oder `regex`, genau eines pro Regel.** Ein Pfad ist als Glob
lesbar, ein Kommando kaum: `git push*` trifft `git pushed-branch-cleanup` und
verfehlt `git   push`. Beide Felder an einer Regel wären die Frage, ob UND oder
ODER gilt — ein `ConfigError` beim Lesen.

**Beide Felder nehmen eine Zeichenkette oder eine Liste.** Mehrere Muster teilen
sich dann eine Begründung. ODER-Semantik. Eine leere Liste ist `ConfigError` und
keine Regel, die nie greift.

**`tools` ist ein optionaler Filter**, kein eigener Regeltyp. Fehlt er, gilt die
Regel für alle Werkzeuge ihrer Art: `Write`, `Edit`, `MultiEdit` bei Pfaden und
Inhalten, `Bash` bei Kommandos.

**`reason` ist Pflicht.** Eine Sperre ohne Begründung erzeugt genau die Sorte
Meldung, gegen die ein Agent argumentiert oder die er umgeht.

### Voreinstellungen

Nur Sicherheitsrelevantes, und überschreibbar über `defaults = false`.

- Pfade: `.env`, `.env.*`, `*.pem`, `*.key`, `id_rsa*`, `*.p12`, `.npmrc`,
  `.pypirc`, `credentials.json`, `.aws/**`
- Inhalte: `-----BEGIN [A-Z ]*PRIVATE KEY-----`, `\bAKIA[0-9A-Z]{16}\b`,
  `\bsk-[A-Za-z0-9]{20,}\b` — die Muster stehen wörtlich in `DEFAULTS`, und die
  README schreibt sie von dort ab, nicht von hier
- Kommandos: keine

`git push` und `pip` statt `uv` sind Politik, nicht Sicherheit, und gehören in
die Projektkonfiguration, wo man sie sieht. Lock-Dateien und Journale ebenso.

### Pfade und ihre Muster

Claude Code liefert in der Payload einen **absoluten** Pfad. Eine Regel wie
`.env` würde darauf nie passen. Der Adapter macht den Pfad darum relativ zur
Projektwurzel, bevor er ihn der Engine übergibt, und normalisiert Trennzeichen
auf `/` — ein Muster soll unter Windows und POSIX dasselbe treffen. Liegt der
Pfad außerhalb der Projektwurzel, wird der absolute Pfad geprüft; eine Regel,
die dorthin zielt, muss ihn selbst nennen.

Gematcht wird mit `PurePosixPath.full_match` und nicht mit `fnmatch`: nur
ersteres kennt `**` über Verzeichnisgrenzen hinweg, und ohne das ist `.aws/**`
kein sinnvolles Muster. Für Kommandos und Inhalte gilt das Umgekehrte — dort
trennt ein Schrägstrich keine Ebenen, und `fnmatch` ist die richtige Wahl.

### Vorrang

Im Deny-Modus gelten Voreinstellungen und Projektregeln zusammen. Im Allow-Modus
gilt allein die Allowlist — auch gegenüber den Voreinstellungen. Wer den Modus
umdreht, übernimmt die Verantwortung ganz.

### Auswertung mehrerer Treffer

Im Deny-Modus werden **alle** treffenden Regeln ausgewertet und alle
Begründungen gemeldet. Bei "erster Treffer gewinnt" räumt der Agent den einen
genannten Grund aus, läuft in den zweiten und braucht drei Runden für eine
Entscheidung, die er beim ersten Mal vollständig hätte treffen können — dasselbe
Argument, aus dem ein Linter alle Befunde einer Datei auf einmal meldet.

Im Allow-Modus beendet die erste passende Erlaubnis die Prüfung.

Reihenfolge der Meldungen: Voreinstellungen zuerst, dann die Projektregeln in
der Reihenfolge der Datei.

## Kosten

Gemessen am 2026-08-25, fünf Läufe je Posten, Hauptcheckout. Die absoluten
Zahlen schwanken mit der Maschinenlast — derselbe nackte Interpreter maß an
diesem Tag zwischen 80 und 117 ms. Stabil sind die Differenzen, und um die geht
es:

| Posten | Anteil |
| --- | --- |
| `ultraloom.checks` (zieht `concurrent.futures`, `process`, `ctypes`) | 25 ms |
| `ultraloom.config` über dem nackten Interpreter (`tomllib`, `dataclasses`) | 30 ms |
| `ultraloom.exe --help` über dem Konfigurationsimport (Parserbau) | 34 ms |
| **`ultraloom.exe --help` gesamt an diesem Tag** | **181 ms** |

Die Policy läuft bei jedem `Write`, `Edit` und `Bash`; der Aufwand pro Aufruf
ist darum eine Anforderung. Er sitzt fast vollständig im Prozessstart, nicht in
der Regelauswertung.

Maßnahmen, in dieser Reihenfolge:

1. **Lazy Imports in `cli.py`.** `checks` wandert aus dem Modulkopf in die
   Funktionen, die es brauchen. `-X importtime` schreibt dem Modul 25 ms zu;
   an der Wanduhr wurden daraus 12 ms, weil ein Teil dessen, was es zieht,
   ohnehin geladen wird. `check` verliert nichts.
2. **Parserbau nur für den gewählten Zweig** — nur, wenn die Messung nach
   Schritt 1 es noch rechtfertigt. Vor argparse an `sys.argv` zu schneiden
   kostet Hilfetexte und gute Fehlermeldungen.
3. Nachmessen und die Zahl hier eintragen.

### Nachgemessen, 2026-08-25 (Schritt 1 erledigt)

Fünf Läufe je Posten, dieselbe Schleife für beide Posten, unmittelbar
nacheinander, Hauptcheckout. Die Grundlinie ist `python.exe -c "pass"`;
aussagekräftig ist allein die Differenz:

| Zustand | Grundlinie | `ultraloom.exe --help` | Differenz |
| --- | --- | --- | --- |
| vor den Lazy Imports | 112 ms / 117 ms | 169 ms / 178 ms | 57 ms / 61 ms |
| nach den Lazy Imports | 128 ms / 125 ms | 175 ms / 172 ms | 47 ms / 47 ms |

Gespart sind damit rund 12 ms, nicht die oben veranschlagten 25: der Anteil von
`ultraloom.checks` war an dieser Messung kleiner als die Einzelmessung der
Tabelle darüber ihn ausweist. Die verbleibende Differenz von 47 ms liegt unter
der Schwelle von 60 ms, ab der Schritt 2 gerechtfertigt wäre — der Parserbau
bleibt vorerst, wie er ist.

Kein Aufruf über `python -m`: der Sinn des Werkzeugs ist, dass ein Projekt
`ultraloom` aufruft und nicht Python. Darunter käme man nur, indem die Policy
die Konfiguration selbst parst — die Verdopplung, gegen die `worktree.py` in
seinem eigenen Docstring argumentiert.

**Kurzschluss:** Der Adapter liest `tool_name` und beendet sich mit 0, bevor eine
Konfiguration angefasst wird, wenn das Werkzeug keine Regelart berührt.

Ein zweiter Kurzschluss stand hier ursprünglich auch — reguläre Ausdrücke nur
für die betroffene Art zu kompilieren — und ist bei der Umsetzung *nicht*
entstanden: `load_ruleset` baut alle drei Gruppen, und `Rule.__post_init__`
kompiliert dabei jedes Muster, auch die der unbeteiligten Arten. Das ist der
Preis dafür, dass ein kaputter Ausdruck beim Lesen auffällt und nicht erst beim
ersten Treffer. Gemessen fällt es nicht ins Gewicht; sollte eine Konfiguration
je so viele Regeln tragen, dass es das tut, ist das die Stelle.

## Exit-Protokoll

    0  erlaubt, oder das Werkzeug berührt keine Regel
    1  interner Fehler — blockt nie; eine defekte Policy darf keine Sitzung einsperren
    2  verweigert; alle Begründungen auf stderr

**Eine kaputte Konfiguration ist Exit 2, nicht Exit 1.** Eine Policy, die bei
fehlerhafter Konfiguration stillschweigend durchlässt, ist der Fehlermodus, den
`config.py` beim Profil-Lesen ausdrücklich vermeidet und den die README als die
eine Fehlerart beschreibt, die wirklich Schaden anrichtet. Exit 1 bleibt echten
Innenfehlern vorbehalten: Payload unlesbar, stdin leer.

Fehlt `.ultraloom/config.toml` ganz, greifen die Voreinstellungen. Ein Repo ohne
Konfiguration ist damit geschützt, ohne dass jemand etwas eingerichtet hat.

## Tests

- `tests/policy/test_rules.py` — tabellengetrieben über die Entscheidungsmatrix:
  Deny und Allow je mit und ohne Voreinstellungen, Listen, `tools`-Filter,
  mehrfache Treffer und ihre Reihenfolge.
- `tests/policy/test_config.py` — jeder `ConfigError` einzeln: unbekannte Art,
  beide Musterfelder, keines von beiden, leere Liste, fehlendes `reason`,
  unbekannter Modus. Dazu: fehlende Datei ergibt die Voreinstellungen.
- `tests/policy/test_hook.py` — Payload-Fixtures für `Write`, `Edit`,
  `MultiEdit`, `Bash` und ein unbeteiligtes Werkzeug; leeres stdin, kaputtes
  JSON. Prüft Exit-Codes, nicht Regeln.
- `tests/test_module_boundary.py` — zweiter Lauf für die Policy.

**Die Aufwandszusage wird ein Test, keine Messung.** Ein
Millisekunden-Schwellwert ist auf einer geteilten Maschine wackelig — die
Schwankung oben zeigt, warum. Deterministisch prüfbar ist die Ursache: ein
Kindprozess führt `ultraloom policy hook` aus, und `ultraloom.checks`,
`concurrent.futures` und `ctypes` dürfen hinterher nicht in seinem `sys.modules`
stehen. Das hält die Lazy Imports gegen den nächsten Beitragenden, der oben im
Modul wieder ein `import` ergänzt.

100 % Coverage wie im übrigen Repo, jeder Ausschluss mit Begründung.

## Doku

Ein README-Abschnitt "Policy" zwischen Prüfkette und Harness, mit der
vollständigen Liste der Voreinstellungen — was ein Werkzeug ohne Konfiguration
bereits sperrt, muss man nachlesen können, ohne den Quelltext zu öffnen. Die
`README.de.md` bekommt denselben Abschnitt.

Dazu ein Ablaufbild unter `docs/abläufe/` mit Mermaid: Payload, Art, Modus,
Voreinstellungen, Verdikt. Diese Verzweigung wird als Prosa unlesbar.

## Reihenfolge der Umsetzung

1. `policy.rules` samt Tests — die Engine steht allein.
2. `policy.config` samt Tests — Schema, Voreinstellungen, Fehlerfälle.
3. Lazy Imports in `cli.py`, nachmessen, Zahl eintragen.
4. `policy.hook` und das Unterkommando, samt Importtest.
5. README, README.de und Ablaufbild.
6. `.claude/settings.json` dieses Repos mit einem `PreToolUse`-Eintrag, der die
   fertige Policy ruft.

## Ausdrücklich nicht in diesem Vorhaben

**Die übrigen Hooks dieses Repos** — `post_edit` (Profil `edit`), `stop`
(`check all`, blockend mit Zähler bis 3, Marker `.claude/.no-verify`) und
`session_start` (pausierte Läufe melden). Entworfen, aber ein eigener Schritt
nach der Policy.

**Das Ausrollen nach space und iam_backend.** Beide kennen ultraloom heute nicht;
sie brauchen je einen eigenen Vorgang. Fehler im Schema findet man im eigenen
Repo billiger als in dreien.
