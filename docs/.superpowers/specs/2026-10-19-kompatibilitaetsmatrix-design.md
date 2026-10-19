---
status: active
---

# Kompatibilitätsmatrix: Sprachversionen × Frameworks

## Absicht

loomux soll durchgängig belegen, dass jede Kombination aus Sprach- oder
Engine-Version und Framework funktioniert, die es zu unterstützen behauptet.
Eine neue Version wird aufgenommen, indem man **eine Zeile Daten** einträgt,
ohne neuen Testcode; der nächste Lauf zeigt, ob etwas bricht. Was die
Support-Seite (`docs/*/support.md`) und die Framework-Seite
(`docs/*/stacks/gdscript.md`) als geprüft ausweisen, ist danach von der
Matrix belegt, nicht von Hand behauptet.

Anlass: Die Abnahme der Godot-3/4-Unterstützung lief einmalig von Hand an
geklonten Demo-Projekten und fand dabei zwei echte Fehler (Schlüssel in
Anführungszeichen, Windows-Rebuild-Sperre). Das soll eine Suite tun, immer
wieder.

## Ziele

1. **Eine Liste** aller unterstützten Kombinationen als Daten.
2. **Schicht 1, schnell:** in `go test`, bei jedem PR, ohne fremde Werkzeuge.
3. **Schicht 2, echt:** ein GitHub-Workflow, der die echten Werkzeuge je
   Eintrag installiert und `loomux check` gegen ein Beispielprojekt laufen
   lässt, auf **Linux und Windows**.
4. **Wochenjob (Späher):** findet neue Upstream-Versionen, lässt Schicht 2
   für sie probehalber laufen und öffnet ein Issue mit Ergebnis und fertigem
   Eintrag zum Kopieren.
5. **Rückkopplung:** Support- und Framework-Seite werden per Test gegen die
   Liste gehalten.

## Nicht-Ziele

- Die übrigen neun Stacks (typescript, vue, svelte, css, html, cpp, shell,
  sql, rust). Sie folgen im **zweiten PR** im selben Muster, ohne neuen
  Mechanismus.
- PRs durch einen Bot. Der Späher pusht nichts und öffnet nur Issues (die
  Regel „nur ein Mensch pusht“ bleibt unverändert).
- macOS.
- Jede Patch-Version einzeln. Eine Linie wird durch Einträge für ihre
  Minor-Versionen vertreten (siehe Liste).
- Fremde Projekte als Prüfstand (die Demo-Klone der Abnahme). Die Matrix
  läuft gegen eigene, kleine Beispielprojekte im Repo.

## Anti-Ziele

- Eine Matrix, die grün meldet, ohne dass ein Werkzeug wirklich lief (ein
  übersprungener, `unavailable` oder `unready` gewordener Lauf zählt als rot).
- Eine Liste und eine Doku, die auseinanderlaufen können.
- Ein Major-Release, das trotz roter Matrix rausgeht. Kleinere Releases und
  Betas prüft die Matrix nicht (Entscheidung des Bauenden, siehe
  „Festlegungen“).

## Entscheidungen des Nutzers

1. **Godot 3 bekommt eine Test-Lane mit GUT.** Der erste PR baut sie, nach
   dem Muster von `check godot-suite` für gdUnit4: erkannt an
   `plugin.cfg:name="Gut"`, headless, Urteil aus dem Bericht von GUT, nicht
   aus dem Exit-Code. GUT 7 trägt Godot 3, GUT 9 Godot 4; die Lane gilt für
   beide Linien, und die Liste bekommt Einträge für beide. Coverage für
   Godot 3 (godot-code-coverage) gehört nicht zum ersten PR.
2. **Werkzeugversionen werden einstellbar.** `[verify.<stack>]` bekommt einen
   Schlüssel `tools`, eine Tabelle Werkzeug → Versionsangabe, etwa
   `tools = { ruff = "0.6.*", gdtoolkit = "3.*" }`. Ohne Angabe bleibt alles
   wie heute (die jeweils neueste Version). Mit Angabe ruft das Preset genau
   diese Version auf (`uvx ruff@…`, `uvx --from 'gdtoolkit==…'`,
   `uv run --with 'mypy==…'`). Das ist eine neue, für Nutzer sichtbare
   Einstellung mit Doku in `configuration.md`; sie geht wie jede Änderung an
   `.loomux/config.toml` über `config set … --propose`. Ein Beispielprojekt
   der Matrix trägt die Versionen seines Eintrags in seiner eigenen
   `.loomux/config.toml`. Für Go gibt es keine Werkzeugversion: die Lanes
   nutzen nur die Toolchain, und deren Version ist die Laufzeit des Eintrags.

## Festlegungen des Bauenden

- **Graph in Schicht 2:** `dev compat run` baut und prüft den Graphen
  direkt (`graph build` und die erwarteten Kanten aus `expect.toml`) statt
  über die Lane `graph`, deren Sonde in CI absichtlich nicht anschlägt.
- **Werkzeuge sind keine Frameworks:** ruff, mypy, pyright, pytest,
  coverage und gdtoolkit stehen im Eintrag unter `tools` (Name → Version),
  nicht unter `frameworks`. `frameworks` nennt nur Addons, die im Projekt
  liegen (gdUnit4, NanoCoverage, GUT), mit den Namen aus `frameworks.toml`;
  nur für sie gilt die Regel „jedes Framework hat eine Zeile in
  `frameworks.toml`“. `frameworks.toml` bekommt keine Zeilen für Python oder
  Go. Die Zeilen `gdtoolkit3`/`gdtoolkit4` bleiben und gelten als belegt,
  wenn ein Eintrag gdtoolkit in der jeweiligen Hauptversion als Werkzeug
  führt.
- **Reichweite des Release-Tors:** Pflicht ist die volle Matrix nur für
  einen PR mit `release:major`. Minor-, Patch- und Beta-Releases laufen ohne
  sie; wer vor einem solchen Release prüfen will, startet den Workflow von
  Hand.

## Aufteilung (freigegeben am 2026-10-19)

Die drei Stacks kommen in drei PRs, in dieser Reihenfolge:

1. **Werkzeugversionen** (`[verify.<stack>] tools`, Entscheidung 2),
   eigenständig nützlich.
2. **GUT-Test-Lane** für Godot 3 und 4 (Entscheidung 1), eigenständig
   nützlich.
3. **Die Matrix** für gdscript, python und go: Liste, Beispielprojekte,
   Schicht 1, Schicht 2, Späher, Rückkopplung. Baut auf 1 und 2 auf.

Danach folgen die übrigen neun Stacks im selben Muster.

## Umfang der Matrix (PR 3)

Stacks **gdscript**, **python**, **go**.

### Die Liste

Neue Datei `internal/verify/compat.toml`, geladen wie `frameworks.toml`
(eingebettet, beim ersten Gebrauch geparst, unbekannte Schlüssel und doppelte
Einträge sind Ladefehler mit Zeilenangabe). Ein Eintrag:

```toml
[[entry]]
stack    = "gdscript"
runtime  = "godot"          # godot | python | go
version  = "4.3"            # Minor-Version der Linie; Schicht 2 nimmt die neueste Patch-Version
fixture  = "godot4-gdunit4" # Ordner unter internal/verify/testdata/compat/
kinds    = ["lint", "test", "coverage", "graph"]
frameworks = [
  { name = "gdUnit4",      version = "5.0.0" },
  { name = "NanoCoverage", version = "1.0.0" },
]
tools = { gdtoolkit = "4.3.*" }
```

Der Schlüssel eines Eintrags ist `stack/runtime-version/fixture`. Jedes
Framework eines Eintrags muss eine Zeile in `frameworks.toml` haben (siehe
„Festlegungen“ für die Abgrenzung zu `tools`).

**Erste Liste (Vorgabe):** je noch gepflegter Linie die älteste und die
neueste Minor-Version, die das jeweilige Framework trägt:

- **Godot:** 3.x mit GUT 7 und gdtoolkit 3 (älteste und neueste 3.x-Minor;
  Arten lint, test, graph), 4.x ohne Test-Framework (älteste 4.x-Minor: nur
  Graph und lint), 4.x mit gdUnit4 und NanoCoverage (älteste Minor, die
  gdUnit4 trägt, und neueste), 4.x mit GUT 9 (neueste Minor).
- **Python:** jede von python.org noch gepflegte 3.x-Version mit ruff, mypy,
  pytest und coverage; eine davon zusätzlich mit pyright statt mypy (die
  Variante des Presets).
- **Go:** die zwei Versionen, die das Go-Team pflegt.

Welche Versionen das am Tag des Baus konkret sind, legt der Bauende anhand
der Upstream-Quellen fest.

### Beispielprojekte

Unter `internal/verify/testdata/compat/<fixture>/` je ein kleines, **grünes**
Projekt, das jede Art seines Eintrags wirklich ausübt (mindestens ein Test,
eine gemessene Zeile, eine Graph-Kante je Sprachmerkmal, das die Linie hat;
für Godot 3 etwa `connect("s", self, "m")`, für Godot 4 `s.connect(m)`).

Daneben `expect.toml`:

- erkannte Stacks und Varianten,
- die Lanes, die `check precommit --show` plant,
- eine Auswahl Graph-Kanten, die der Build ergeben muss,
- je Art ein **Bruch**: eine kleine Änderung (Datei überschreiben oder
  anlegen), nach der genau diese Art rot werden muss. So prüft die Matrix
  beide Richtungen: grün, wo es grün sein soll, und rot, wo etwas kaputt
  ist.

Ein Beispielprojekt darf von mehreren Einträgen genutzt werden (Godot 4.3
und 4.7 teilen eines).

### Schicht 1 (`go test`, jeder PR)

Für jeden Eintrag, ohne fremde Werkzeuge:

- Erkennung und Lane-Planung auf dem Beispielprojekt stimmen mit
  `expect.toml` überein.
- Der Graph-Build des Beispielprojekts enthält die erwarteten Kanten.
- Jeder Bruch ändert die Eingaben der Art, die er brechen soll (die Lane
  plant weiter; ob sie rot wird, entscheidet erst Schicht 2).
- Konsistenz der Daten: jedes Framework eines Eintrags steht in
  `frameworks.toml`; jedes Beispielprojekt wird von einem Eintrag benutzt.

### Schicht 2 (Workflow `compat.yml`)

**Auslöser:**

1. `workflow_dispatch`, mit optionalem Filter auf Einträge.
2. Ein PR mit dem Label `release:major`: die **volle** Matrix. Der Lauf ist
   eine Pflichtprüfung, ein roter Lauf hält damit Merge und Release auf.
3. Ein PR, der `compat.toml` oder ein Beispielprojekt ändert: nur die
   **betroffenen** Einträge.
4. Jeder andere PR: der Job meldet sofort Erfolg, damit er als Pflichtprüfung
   nichts blockiert (GitHub kennt keine bedingt verpflichtenden Prüfungen).

Die Matrix erzeugt ein neuer Unterbefehl `loomux dev compat matrix` aus der
Liste (JSON für `fromJSON`), mit `--changed-since <ref>` für Auslöser 3. Jeder
Eintrag läuft auf Linux und Windows.

**Ein Matrix-Job:**

1. Laufzeit installieren: Godot als offizielles Binary von
   `godotengine/godot-builds` (Prüfsumme gegen die veröffentlichte),
   Python über `uv python install`, Go über `actions/setup-go`.
2. Frameworks in der Version des Eintrags installieren: Godot-Addons aus dem
   Git-Tag ihres Repos nach `addons/`. Werkzeuge installiert der Job nicht
   selbst: `run` schreibt die `tools` des Eintrags in die
   `.loomux/config.toml` der Arbeitskopie, und die Presets rufen genau diese
   Versionen auf (Entscheidung 2).
3. Das Beispielprojekt in ein Arbeitsverzeichnis kopieren und
   `loomux dev compat run <eintrag>` aufrufen. Der Befehl ruft `loomux check`
   für jede Art des Eintrags auf und verlangt **grün und wirklich gelaufen**;
   dann wendet er jeden Bruch an und verlangt, dass genau seine Art rot wird.
4. Ergebnis als Job-Zusammenfassung und als JSON-Artefakt.

Die Runner wählt `pick-runner.yml` wie in den übrigen Workflows.

### Späher (Workflow `compat-scout.yml`)

Wöchentlich (`schedule`) und per `workflow_dispatch`, Rechte nur
`contents: read` und `issues: write`.

1. `loomux dev compat scout` fragt die Upstream-Quellen ab: Godot über die
   Releases von `godotengine/godot`, Python über die Versionsliste von
   python.org, Go über `go.dev/dl/?mode=json`, Frameworks über ihre
   GitHub-Releases oder PyPI.
2. **Neu** ist eine Minor-Version (oder Major) einer Laufzeit oder eines
   Frameworks, die neuer ist als jeder Eintrag ihrer Linie. Patch-Versionen
   lösen nichts aus.
3. Für jede neue Version erzeugt er einen Probe-Eintrag: eine Kopie des
   neuesten Eintrags derselben Linie mit der neuen Version, und lässt
   Schicht 2 für ihn auf Linux und Windows laufen.
4. Er öffnet je neue Version **ein** Issue oder aktualisiert das vorhandene
   (erkannt am Titel), mit dem Ergebnis beider Plattformen und dem fertigen
   TOML-Block zum Kopieren. Ein roter Lauf ist damit ein Fehlerbericht, bevor
   jemand die Version einträgt. Eine Version, die eingetragen wurde, schließt
   ihr Issue beim nächsten Lauf.

### Rückkopplung zur Doku

- `frameworks.toml` verliert das handgepflegte `proven`: Ein Framework gilt
  als belegt, wenn ein Eintrag der Liste es enthält. Die bestehenden Tests,
  die `proven` gegen die Presets halten, lesen das abgeleitete Feld.
- Die Support-Seite bekommt einen Abschnitt „Geprüfte Kombinationen“ (eine
  Tabellenzeile je Eintrag: Stack, Laufzeit und Version, Frameworks mit
  Versionen, Arten). Ein Test hält ihn Zeile für Zeile gegen die Liste, auf
  Englisch und Deutsch.
- Die Versionsspalte der Übersicht nennt für gdscript, python und go nur
  noch, was die Liste trägt, und verweist auf den Abschnitt.

## Randbedingungen

- Projektregeln aus `AGENTS.md`: 100 % Coverage je Funktion, kein `init()`,
  eingebettete Daten erst beim ersten Gebrauch parsen, Doku Englisch und
  Deutsch, Commits und Code Englisch.
- Die Pflichtprüfung `compat` wird in `.github/rulesets/` eingetragen; auf
  GitHub anwenden muss sie ein Mensch.
- Schicht 2 und Späher laden fremde Binaries und Pakete: nur aus den
  offiziellen Quellen, mit Prüfsumme, wo die Quelle eine veröffentlicht.

## Dem Bauenden überlassen

- Die konkreten Versionen der ersten Liste (nach den Regeln oben).
- Format und Inhalt von `expect.toml` im Einzelnen.
- Aufbau der Unterbefehle `dev compat matrix|run|scout` und wie `run` die
  Ergebnisse von `loomux check` liest.
- Ob Werkzeuge zwischen Matrix-Jobs zwischengespeichert werden.
