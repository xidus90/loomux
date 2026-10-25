# CLI-Modelladapter für Agentenknoten

Stand 2026-10-24. Entscheidung dazu:
`docs/wiki/syntheses/modelladapter-ueber-die-cli.md`.

## Absicht

Agentenknoten eines Flows laufen heute nicht: `flowProduction` in
`internal/cli/flow.go` antwortet für jeden Anbieter
`no adapter for provider <p> yet`. Ziel ist, dass ein Flow mit Agentenknoten
echt läuft — über die CLIs, die der Nutzer ohnehin benutzt und angemeldet hat.
Dadurch fasst loomux keine API-Schlüssel an, und die loomux-Hooks (Wächter,
Schreibschranke) greifen in der Sitzung der CLI.

## Ziele

- Drei Adapter hinter `model.Model` (`internal/flow/model/model.go`):
  `claude` (Claude Code), `agy` (Antigravity), `qoder` (Qoder CLI, `qodercli`).
- Strata (lokales Qwen 3.8 Next mit Anthropic-kompatibler API auf localhost)
  läuft über den `claude`-Adapter mit eigener Basis-URL; kein vierter Adapter.
- Modell, Agent und Effort sind bestimmbar: in der Config beim Modell, der
  Knoten darf überschreiben.
- Zeitgrenze je Agentenknoten: Vorgabe 30 Minuten, konfigurierbar, mit `0`
  ausschaltbar.

## Nicht-Ziele

- API-Adapter (eigene Roadmap-Zeile „API model adapters“).
- Parallele Agentenknoten und eine Grenze dafür — die Laufzeit führt Knoten
  nacheinander aus; Parallelität wird eine eigene Roadmap-Zeile (Fan-out/Join
  als Knotenart, Grenze der parallelen Agenten: Vorgabe, konfigurierbar,
  ausschaltbar).
- Die Lücke der Schreibschranke bei agy 1.4.3 (`write_file`): eigener Fix,
  eigener PR (Branch `claude/affectionate-dhawan-30fa68`).

## Anti-Ziele

- Ein Adapter, der stillschweigend mehr Werkzeuge freigibt, als das Profil des
  Knotens erlaubt — wo die CLI sperren kann, wird gesperrt.
- Ein Lauf, der ohne Hinweis läuft, obwohl die Grenze nicht hart ist.
- Echte CLI-Aufrufe in Tests.

## Rahmen

- Regeln aus `AGENTS.md`: 100 % Coverage je Funktion, kein `init()`, englische
  Bezeichner und Meldungen.
- `.loomux/config.toml` schreibt kein Agent; neue Schlüssel kommen in
  `config.example.toml`, die Doku und die Testdaten.
- Gemessene CLI-Fähigkeiten (2026-10-24):

  | | `claude` 2.1.310 | `agy` 1.4.3 | `qodercli` 1.1.78 |
  |---|---|---|---|
  | ohne Rückfragen | `-p` | `-p` (Prompt direkt am Flag; `--mode` davor) | `-p` |
  | JSON-Ausgabe | `--output-format json` | `--output-format json` (Status, Antwort, `usage`) | `-o json` |
  | Antwortschema | `--json-schema` | `--json-schema` | keins |
  | Werkzeuge sperren | `--tools`, `--allowedTools`, `--disallowedTools` | keine Option; `--mode plan` liest im Arbeitsordner nur, ist aber Modellverhalten, keine Sperre | `--tools`, `--allowed-tools`, `--disallowed-tools` |
  | Modell / Effort | `--model`, `--effort` | `--model`, `--effort` | `-m`, `--reasoning-effort` |
  | Agent | `--agent` | `--agent` | `--agents <json>` (zu prüfen, ob ein Name reicht) |
  | Zeitgrenze | – | `--print-timeout` | – |

## Entschieden

**Vom Nutzer:**

1. Nur CLI, drei Adapter `claude`, `agy`, `qoder`; API später.
2. Strata über den `claude`-Adapter mit Basis-URL.
3. Kann `agy` ein Profil nicht hart durchsetzen, läuft der Knoten **mit
   Warnung** (nicht abgelehnt).
4. `agent`, `model`, `effort` stehen beim Modell in `[agent.models.<name>]`;
   der Knoten darf `agent` und `effort` überschreiben (`model` kommt wie bisher
   über die Rolle).
5. Zeitgrenze je Agentenknoten 30 Minuten, konfigurierbar, `0` schaltet sie ab.
6. Parallelität nicht jetzt, eigene Roadmap-Zeile.

**Meine Festlegungen (prüfbar):**

- Config-Schlüssel: `[agent.models.<name>]` bekommt `agent`, `effort`,
  `base_url` (nur bei `provider = "claude"` erlaubt; sonst Ladefehler).
  `[agent] timeout` als Dauer-String (`"30m"`, `"0"` = aus); fehlt er, gilt
  `30m`.
- Neuer Anbieter `qoder` in der Config-Prüfung.
- Strata: Der `claude`-Adapter setzt bei gesetztem `base_url`
  `ANTHROPIC_BASE_URL` in der Umgebung des Kindprozesses; sonst bleibt die
  Umgebung unverändert. Ob Strata zusätzlich einen Platzhalter-Token braucht,
  klärt der Bau an der Strata-Doku.
- Antwort: `claude` und `agy` bekommen aus `Request.Reply` ein JSON-Schema
  (flaches Objekt, alle Felder Pflicht, keine weiteren). `qoder` bekommt das
  Schema als Vorgabe im Prompt; die Antwort wird als JSON gelesen, bei
  Fehlschlag genau ein neuer Versuch mit der Fehlermeldung, dann Fehler.
  Gemeinsamer Prüfer: Felder und Typen gegen `Request.Reply`, für alle drei.
- Werkzeuge: `claude` und `qoder` bekommen genau `Request.Tools` über
  `--tools` (alles andere ist damit aus). `agy`: Profil `read_only` → `--mode
  plan`; andere Profile → `--mode accept-edits`. Für jedes Profil warnt der
  `agy`-Adapter einmal je Knoten auf stderr und im Journal, dass die
  Werkzeuggrenze nicht hart ist.
- Kein Prompt-Abfrage-Modus: kein Adapter setzt `--dangerously-skip-permissions`
  oder einen Bypass-Modus.
- Arbeitsordner des Kindprozesses: die Projektwurzel des Laufs (`--root`),
  damit die Hooks des Projekts greifen.
- Tokens und Modell aus der JSON-Ausgabe der CLI; fehlt die Zahl, `Tokens = 0`
  und `Model = ""` (der Port erlaubt das schon).
- Zeitgrenze über `context.WithTimeout` um den Kindprozess; bei Ablauf wird der
  Prozess beendet und der Knoten scheitert mit einer Meldung, die Grenze und
  Schlüssel nennt.
- Prozessstart hinter einer kleinen Funktion (`exec`-Naht), damit Tests eine
  Fake-CLI einsetzen; die Fake-CLI ist ein Testbinary bzw. Testhelfer-Prozess
  nach Go-Muster (`TestHelperProcess`), keine echte CLI.
- Paket: `internal/flow/model/cli` (ein Paket, drei kleine Dateien je
  Anbieter plus gemeinsamer Teil). `flowProduction` wählt den Adapter nach
  `provider`.
- Definition-Hash: `agent` und `base_url` gehen ein (ändern sie sich, zeigt
  `replay` eine geänderte Definition), `timeout` nicht.

**Nach der Bauprüfung festgelegt (meine Festlegungen):**

- **Naht:** Die Fabrik nimmt nicht mehr nur den Anbieter, sondern das
  aufgelöste Modell: `Models func(model.Resolved, model.Run) (model.Model,
  error)`, wobei `Run` Wurzel und Zeitgrenze trägt. Ein Adapter je Modell;
  damit kennt ein `claude`-Adapter für Strata seine `base_url`. `Resolved`
  bekommt `Agent`, `Effort`, `BaseURL` aus `ModelSpec`.
- **Request:** bekommt `Profile` (Name des Werkzeugprofils) und `Agent`. Der
  `agy`-Adapter entscheidet nach `Profile`: nur `read_only` → `--mode plan`,
  alles andere (auch `mcp` ohne Server) → `accept-edits`.
- **Werkzeugnamen:** Je CLI eine Übersetzungstabelle von den Profilnamen
  (`Read`, `Bash`, `mcp__<server>` …) auf die Namen der CLI. `claude`:
  eingebaute über `--tools`, `mcp__…` über `--allowedTools`. `qoder`: Namen
  beim Bau an `qodercli` messen; hat ein Profilwerkzeug keine Entsprechung,
  wird es weggelassen und gewarnt wie bei `agy` (nie stillschweigend
  breiter, nie stillschweigend schmaler).
- **Vorrang:** `agent` und `effort`: Knoten, sonst Modell, sonst leer. Der
  aufgelöste Wert geht in den Definition-Hash (wie heute `effort`), also
  zeigt auch eine Änderung beim Modell eine geänderte Definition. `base_url`
  geht ebenfalls ein. `Agent.Check` kennt den Knotenschlüssel `agent`.
- **Warnung „Grenze nicht hart“:** auf stderr und als Feld der Definition
  (z. B. `ToolsEnforced: false`), nicht als eigenes Ereignis. Sie folgt aus
  Config und Anbieter, ist also bei `replay` gleich; Goldens ändern sich nur
  durch das neue Feld, wo es gesetzt ist.
- **Tokens:** Eingabe plus Ausgabe, ohne Cache-Lesen und ohne Denk-Tokens,
  für alle drei gleich; liefert eine CLI die Teile nicht, `0`.

**Beim Bau geändert:** Die Warnung „Grenze nicht hart“ steht nur auf stderr
(`cli.Warnings`), nicht als Feld der Definition: Das Feld hatte keinen Leser,
und ein Journalfeld hätte Goldens verändert. Gemessen statt angenommen: agy
antwortet in `structured_output` und verweigert headless Werkzeuge ohne
Freigabe; `qodercli --tools` beschränkt hart, daher warnt qoder nur für
Werkzeuge ohne Entsprechung.

## Dem Bau überlassen

- Exakte Flag-Kombination je CLI nach `--help` der installierten Version,
  inklusive Reihenfolge-Eigenheiten (agy: `-p` nimmt das nächste Wort).
- Ob `qodercli --agents` einen Namen annimmt oder JSON braucht; braucht es
  JSON, bleibt `agent` bei `qoder` vorerst ein Ladefehler mit klarer Meldung.
- Form der Warnung im Journal (vorhandenes Ereignisformat nutzen).

## Doku im selben PR

- `docs/en/flows.md`, `docs/de/flows.md`: „What runs today“ und Abschnitt
  Rollen und Modelle (agent, effort, base_url, timeout, qoder, agy-Warnung).
- `docs/en/configuration.md`, `docs/de/configuration.md`: neue Schlüssel.
- README/README.de: Roadmap-Zeile Flow-Laufzeit (Adapter gebaut), neue Zeile
  Parallelität.
- Wiki: `syntheses/modelladapter-ueber-die-cli.md` auf umgesetzt, Log-Eintrag.
