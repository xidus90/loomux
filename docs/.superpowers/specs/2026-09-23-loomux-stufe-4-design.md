# Stufe 4: Einrichten und Aufnehmen — `config`, `init`, das lokale Modell, `convert`/`fetch`

**Datum:** 2026-09-23
**Deckt ab:** die ganze Stufe 4 der Fusions-Spec
(`2026-09-14-loomux-fusion-design.md`), geschnitten in **4a-1**, **4a-2**,
**4c** und **4d**, dazu den Abschlussschritt **4e**. Ein Implementierungsplan
entsteht in diesem Zug nur für **4a-1**.
**Ort:** Zweig `feat/stage-4-config` von `master`. Spec, Plan und Bau von 4a-1
gehen in einen Pull Request.
**Referenz:** je Teilstufe verschieden, siehe „Parität“. `config` ist neu und
hat keine Referenz; `init` hat ulinit (Go, zieht um) und für den
post-merge-Hook `ultra-brain` (Python); 4c und 4d haben `ultra-brain` (Python).
**Abweichungsliste:** `docs/.superpowers/parity/stufe-4a-1.md` usw., je
Teilstufe eine.

## Ziel

Nach Stufe 4 richtet loomux ein Projekt selbst ein und nimmt Wissen selbst auf.
Ein Wirt braucht keinen Installer aus `ultraloom` mehr, keinen Bau in
`~/go/bin` von Hand und keinen Python-Befehl, um eine PDF, ein Transkript oder
ein Video in einen Bereich zu bringen. Ein `local_only`-Bereich bekommt seine
Vorschläge vom lokalen Modell statt keinen.

Was diese Stufe **nicht** baut: einen Übersetzer der alten Konfiguration
(`loomux migrate` fällt weg, siehe „Entscheidungen“), die Web-Oberfläche und
ihre `ui.json` (Folgeprojekt Web), die Flows (Folgeprojekt Flow).

## Entscheidungen

Alle am 2026-09-23 mit dem Nutzer getroffen.

| Frage | Entscheidung |
|---|---|
| Schnitt | 4a-1 Schema und `config`, 4a-2 `init`, 4c Modell und `dev bench search`, 4d `convert`/`fetch`; 4e Umstellung der Wirte als Abschlussschritt |
| Reihenfolge | 4a-1 → 4a-2 → 4c → 4d, danach 4e. Nach Regel 2 der Fusions-Spec (Selbstnutzung): `init` benutzt loomux bei jedem Klon, das Modell ändert das Prüfzentrum, `convert` braucht loomux selbst kaum. 4a-1 vor 4a-2, weil `init` über Schema und Oberfläche aus 4a-1 fragt |
| 4e | Kein Code (Nachtrag 2026-09-28: bis auf den lesenden Befehl `loomux area check`, Fusions-Spec #24) und kein grünes Stufenende: er wartet auf einen Remote für `brain-knowledge`, den nur der Mensch anlegt. Er hält die übrigen Teilstufen nicht auf |
| `migrate` | **Fällt weg.** Die alten Werkzeuge hat nur der Nutzer benutzt, auf einem Rechner und in vier Wirten. Der Maschinenzustand ist durch die Selbstnutzung seit 3a schon im neuen Verzeichnis (Befunde). Was bleibt, ist ein Punkt der Checkliste von 4e |
| `init` | Ein interaktiver Ablauf, der **alles** einrichtet und installiert, gegliedert in Module. Jedes Modul fragt: alles, jeden Teil einzeln, nichts. `--yes` nimmt überall die Vorgabe |
| Module | Basis (immer), Hooks, Wiki (Brain), Graph; OS kommt mit W1 |
| `[modules]` | Die Wahl gilt **auch zur Laufzeit** und steht als `[modules]` in `.loomux/config.toml` |
| Umfang der Konfiguration | `init` deckt **jede** Sektion von `.loomux/config.toml` ab |
| Vorhandene `config.toml` | Fehlende Sektionen werden angehängt, soweit ihre Werte von der Vorgabe abweichen (Vorgaben schreibt `init` nie, siehe „Zeileneditor“); ein vorhandener Wert ändert sich nur gezielt in seiner Zeile, Kommentare bleiben; **jede** Änderung wird einzeln bestätigt. Regellisten und `[verify]`-Tabellen werden nur ergänzt, nie umgeschrieben |
| `loomux config` | Neuer Befehl auf demselben Unterbau wie `init`: zeigt die Einstellungen mit Herkunft und ändert sie, nach dem Vorbild von `/config` in Claude Code |
| Oberfläche | Vollbild-Liste, selbst gebaut auf `golang.org/x/term` (Raw-Modus), für `config` und für die Fragen von `init`. Kein `bubbletea` |
| Nachträge | Freigegeben 2026-09-23 (Gruppe A): #5, #7, #8, #10, #11, #16 in 4a-2; #12 und #13 — #13 wird eine gewöhnliche Frage von `init`, #12 fällt weg (`init` kennt `[relevance]` nicht) |
| #7 Brain-Skills | Befehle auf loomux umgeschrieben **und ins Englische übersetzt**: die fünf Skills sind deutsch, und AGENTS.md verlangt alles, was ein LLM anweist, englisch |
| #18 | Bleibt Wegfall; da `migrate` wegfällt, wird `Manifest.Lanes` gelöscht statt übertragen |
| `bench` | Als `loomux dev bench search` in 4c. `dev bench hooks`, `corpus` und `search` teilen **ein** Berichtsschema |
| `[modules].hooks = false` | Schaltet post-edit, `stop`, `session-start` und `subagent-*` ab, **nie** den Wächter: die Schreibschranke bleibt global, wie die Fusions-Spec es verlangt (2026-09-24) |
| Vorhandene loomux-Einträge | `init` erkennt jeden Hook-Eintrag, der ein loomux-Binary ruft (`loomux`, `bin/loomux.exe`, …), als eingerichtet und schreibt ihn nicht um. Zeigen die vorhandenen Einträge auf ein Checkout-Binary wie `bin/loomux.exe`, baut `init` es dort; sonst gilt der kanonische Ort. Git-Hooks, die schon ein Tor fahren, übergeht es ebenso; fehlt `core.hooksPath` bei vorhandenem `.githooks/`, setzt es ihn. Selbstnutzung: `go run ./cmd/loomux init --yes` richtet einen frischen Klon von loomux ein und ersetzt die zwei Handbefehle in AGENTS.md (2026-09-24) |
| Binary der Wirte (#15) | `init` legt das neueste Release über `selfupdate` an den kanonischen Ort der Self-Update-Spec, `%LOCALAPPDATA%\loomux\bin\loomux.exe`, nie eine Kopie seiner selbst. Die Einträge der Wirte rufen es dort über `${LOCALAPPDATA}`, nicht über den `PATH`, weil ein nicht gefundener Wächter jeden Aufruf durchließe (2026-09-24) |
| post-merge (#5) | Ein Weg: Der Hook ruft `loomux merge-hook record` am kanonischen Ort und endet immer mit `exit 0`; keine eingebackenen Pfade, darum auch in einem eingecheckten `.githooks/` (2026-09-24) |
| Vorschläge statt Schreiben für Agenten (2026-09-24) | Ein Agent schreibt `.loomux/config.toml` weiter nie, bedient `loomux config` aber über Vorschläge: `config set … --propose` und `config unset … --propose` rechnen die Änderung wie `set` (Schema, Editor, `Validate`) und legen sie als `<projekt>/.loomux/state/config/proposals/<id>.json` ab (für `--global` unter `<zustand>/config/proposals/`), mit Id, Operation, Schlüssel, Eingabe, Zeit und Ziel, aber ohne Diff. Der Mensch liest sie mit `config proposals`, das jeden Diff neu gegen die aktuelle Datei rechnet, und wendet sie mit `config apply <id>` (oder `--all`) an; `apply` rechnet jede Änderung aus Operation, Schlüssel und Eingabe **neu gegen die Datei, wie sie dann ist**, und schreibt nie den abgelegten Text oder Diff. `config reject` verwirft. Der Wächter erlaubt einem Agenten `list`, `get`, `proposals`, `--help`/`-h` und `set`/`unset` nur mit `--propose`; `apply`, `reject`, `set`/`unset` ohne `--propose` und die interaktive Form bleiben verweigert. `config list` und der Titel der interaktiven Form nennen die Zahl offener Vorschläge |
| Geerbte Fehler aus 3b | #1 (`--reject` schiebt Revision und Hash nicht vor) und #4 (`reindex` und `approve` ohne gemeinsame Sperre) werden in 4c geheilt, als freigegebene Abweichung von der Referenz. #2 und #3 bleiben offen |

### Vorgeschlagen und mit der Spec freigegeben (2026-09-24)

Diese Punkte kamen aus dem Entwurf, nicht aus einer Frage an den Nutzer. Er
hat sie mit der ganzen Spec am 2026-09-24 freigegeben.

| Frage | Vorschlag |
|---|---|
| #9 `session-handover` | `init` schreibt ihn nicht: die globale Fassung des Nutzers (117 Zeilen) ist eine andere als die Vorlage von ulinit (15 Zeilen), und eine Projektkopie würde sie verdecken |
| Schema prüft nicht selbst | Ein Wert wird in den Text eingesetzt und durch die heutigen Lader geschickt; ein Test hält Schema und Lader deckungsgleich |
| Editor verweigert Mehrdeutiges | Gepunktete Schlüssel, Inline-Tabellen, doppelte Sektionen ändert der Mensch |
| `loomux mcp` ohne festen Pfad | Die Brücke sucht das Projekt aufwärts vom Arbeitsverzeichnis; ein eingecheckter Pfad dieses Rechners entfällt |
| Wächterregel für `area add` | Auch `loomux area add` wird einem Agenten verweigert (siehe „Die Wächterregel“) |
| Paketorte | `internal/config/schema`, `internal/config/edit`, `internal/tui`, `internal/setup/…` |

## Befunde, die den Entwurf formen

Gegen Code und Rechner gelesen am 2026-09-23.

- **ulinit ist größer als geschätzt.** `cmd/init` und `internal/render` in
  `ultraloom` haben ~4.750 Zeilen Go ohne Tests, dazu die Pakete `answers`,
  `interview`, `settings`, `write`, `tooling`, `mirrorcfg`, `journal`. Eine
  vorhandene Datei überschreibt ulinit nicht, es meldet sie als „skipped,
  already there“. `.mcp.json` schreibt es schon.
- **Die fünf Brain-Skills rufen nur Befehle, die loomux hat:** `catalog`,
  `read`, `search`, `case`, `cases` und `check all|bundle` (in loomux
  `brain check`). Keiner ruft `convert` oder das Modell; #7 hängt also nicht an
  4c oder 4d.
- **Der post-merge-Hook schreibt selbst.** `merge_events.py` (614 Zeilen)
  legt ein reines sh-Skript mit `exit 0` ab, das Repo, Commit-Bereich, Zweig
  und Zeit in `maintenance/merge-events.tsv` anhängt. Ein Schreibbefehl in
  loomux ist nicht nötig; die Leseseite gibt es seit 3a
  (`internal/brain/maintenance/events.go`). Der Hook achtet auf
  `core.hooksPath` (`git rev-parse --git-path hooks`) und erkennt sich an der
  Marke `# brain post-merge hook`.
- **`core.hooksPath` zeigt in loomux auf das eingecheckte `.githooks/`.** Ein
  post-merge-Hook nach Art der Referenz, mit dem absoluten Ereignispfad dieses
  Rechners, landete dort im Repo. Siehe 4a-2, „post-merge“.
- **Der Maschinenzustand ist schon umgezogen.** `%LOCALAPPDATA%\loomux\areas`
  hält dieselben drei Bereiche wie `%LOCALAPPDATA%\brain\areas`
  (`project-iam-wiki`, `project-obsidian-ai`, `project-space`),
  `maintenance/` alle alten Bereiche und `project-loomux`, `registry.toml` 11
  Einträge gegen 10 (dazu `project/loomux`). `[check].lanes` steht nur in
  `ultra-brain/.brain.toml`, und `ultra-brain` wird archiviert.
- **Die Wirte:** `space`, `iam_backend`, `ecoflow`, `brain-knowledge`. Alte
  Hook-Einträge rufen `ulguard` vom `PATH` und `brain guard`.
- **`serve` ist maschinenweit, `loomux mcp` kennt kein Projekt.** Die Brücke
  nimmt nur `--channel`; `.mcp.json` gibt es in loomux nicht. Registriert ist
  der Server im **Nutzerbereich** von Claude Code (`~/.claude.json`, Name
  `loomux`), mit dem festen Pfad `%LOCALAPPDATA%\loomux\bin\loomux.exe` und
  `mcp --channel local`. Soll `[modules]` Werkzeuge verbergen, muss die Brücke
  das Projekt kennen (4a-1). Eine Projekt-`.mcp.json` mit demselben Namen
  stünde neben dieser Registrierung.
- **Die Schreibschranke ist global.** Die Fusions-Spec legt fest: „global aus
  der Registry, unabhängig von `--root`“. Der Wächter eines Projekts schützt
  also auch die schreibgeschützten Bereiche anderer Repos (`space`,
  `iam-wiki`, `obsidian-ai`).
- **Der Wächter schützt `.loomux/config.toml` vor Shell-Zeilen**
  (`internal/hooks/guard.go`, eingebaute Befehlsregel), aber nicht vor einem
  loomux-Befehl, der die Datei im eigenen Prozess schreibt. Diese Lücke gibt es
  seit 3a: `loomux area add` schreibt `[area]` und `[layout]` über
  `lock.ReplaceText`. `config` und `init` kämen dazu.
- **loomux selbst ruft ein anderes Binary als ein Wirt.** Seine Hooks rufen
  `bin/loomux.exe`, und `.githooks/pre-commit` fährt `ci/gate.sh` und baut
  danach das Pilot-Binary neu. `init` schriebe den kanonischen Ort.
- **Das Modell hat drei Rollen.** `propose` benutzt `reconcile`; `describe` und
  `place` benutzt nur `convert` (`convert/run.py:167,204`).
- **`brain bench` misst die Suche, nicht das Modell:** Rang der erwarteten
  Quelle unter den ersten drei über einen Fragensatz, dazu kalte und warme
  Latenz, über einen versionierten Korpus mit Prüfsummen (`ultra-brain/bench/`).
- **Abhängigkeiten:** `golang.org/x/sys` ist direkt im Modul, `x/term` nicht.

## Teilstufen

| Teilstufe | Inhalt | Hängt ab von |
|---|---|---|
| **4a-1** Schema und `config` | Schlüsselschema, Zeileneditor, Oberfläche auf `x/term`, `loomux config`, `[modules]` mit Laufzeitwirkung, `loomux mcp --root`, die Wächterregel | 3 ✅ |
| **4a-2** `init` | Umzug von ulinit auf das Schema, Module, Host-Einträge, Git-Hooks und post-merge (#5), Skills (#7, #8), `AGENTS.md` (#10), `.gitignore` (#11), `.mcp.json`, Binary an den kanonischen Ort (#15), `--detect-only` (#16) | 4a-1, 2b ✅, 2c ✅, `feat/self-update` gemergt (`internal/swap`, kanonischer Ort, `selfupdate`) |
| **4c-1** Modell | Ollama-Client, Tor, Prompts, `[model]` global und je Bereich, die Rolle `propose` in `reconcile`; Heilung von #1 und #4 (geschnitten beim Planen von 4c, siehe dort) | 4a-1 (`[model]` im Schema), 3 ✅ |
| **4c-2** Bench | `dev bench search`, die Untergruppe `dev bench hooks\|repos\|search` und das gemeinsame Berichtsschema | 3 ✅ |
| **4d** `convert`/`fetch` | Eingang wandeln, Untertitel holen, die Rollen `describe` und `place`, die Richter | 4c-1 |
| **4e** Umstellung | Checkliste; als Code nur `loomux area check` (Nachtrag 2026-09-28, Fusions-Spec #24) | 4a-2, 4c-1, 4d; Remote für `brain-knowledge` |

## 4a-1 im Einzelnen

### Schlüsselschema

Ein Paket `internal/config/schema` beschreibt jeden Schlüssel, den loomux in
`.loomux/config.toml` liest: Sektion, Name, Art (Zeichenkette, Zahl,
Wahrheitswert, Aufzählung, Liste von Zeichenketten, Tabelle, Tabellenliste),
Vorgabe, Modul, englische Beschreibung. Das Schema **prüft nicht selbst**: Ein
vorgeschlagener Wert wird in den Text eingesetzt, und der Text geht durch die
Lader, die heute gelten — `config.ReadDeclaration`, das Policy-Laden,
`verify` und `verify/commit`. Was die Lader ablehnen, lehnt `config` ab, mit
ihrer Meldung. Ein zweiter Prüfer neben dem, der im Betrieb gilt, entsteht so
nicht.

Ein Test hält Schema und Lader zusammen: Jeder Schlüssel, den ein Lader
kennt, steht im Schema, und umgekehrt. Dafür legen die Lader ihre Schlüssellisten
offen, so wie `verify/commit` es mit `knownCommitKeys` schon tut.

Sektionen und Module:

| Modul | Sektion | Schlüssel |
|---|---|---|
| Basis | `[modules]` | `hooks`, `brain`, `graph` |
| Basis | `[commit]` | `language`, `conventional`, `threshold`, `[[commit.allow]]` |
| Basis | `[policy]` | `[[policy.paths.rules]]`, `[[policy.commands.rules]]` |
| Basis | `[worktree]` | `mirror` |
| Hooks | `[verify]` | `max_parallel`, `timeout`, `profiles`, die Tabellen je Stack und Art |
| Brain | `[area]`, `[layout]`, `[wiki]`, `[index]`, `[privacy]`, `[maintenance]`, `[model]` | `scope`; `wiki`, `hub`, `review`, `inbox`; `types`, `untouched_days`; `include`, `exclude`, `unsearched`; `mode`, `never`; `on_merge`, `branch`; `enabled`, `roles` — die beiden letzten Sektionen liest `ReadDeclaration` schon heute (nachgetragen beim Planen von 4a-1) |
| Graph | — | heute keine |

`[check]` steht nicht im Schema; `Manifest.Lanes` wird gelöscht (#18).

### Zeileneditor

Ein Paket `internal/config/edit` ändert `.loomux/config.toml` als Text:

- **Wert ändern:** Es sucht den Kopf `[sektion]` und darunter die Zeile
  `schlüssel =`, ersetzt nur den Wert und lässt einen Kommentar am Zeilenende
  stehen. Ein mehrzeiliger Wert (eine Liste über mehrere Zeilen) wird als
  Ganzes ersetzt, begrenzt durch das Klammergleichgewicht außerhalb von
  Zeichenketten.
- **Schlüssel ergänzen:** nach der letzten Schlüsselzeile seiner Sektion.
- **Sektion ergänzen:** am Dateiende, mit Leerzeile davor und dem englischen
  Kommentar aus dem Schema.
- **Vorgaben werden nie geschrieben.** Ein Wert, der der Vorgabe des Schemas
  gleicht, kommt nicht in die Datei, ebenso wenig eine Sektion, deren Werte
  alle Vorgaben sind. Das ist die Regel, die für `[verify]` schon gilt
  (Presets werden nicht kopiert), ausgedehnt auf jede Sektion. So bleibt eine
  eingecheckte `config.toml` bei einem Lauf mit lauter Vorgaben unberührt.
- **Tabellenliste ergänzen:** ein neuer `[[…]]`-Block am Dateiende. Einträge
  werden nie entfernt oder umgeschrieben.
- **Was sich nicht eindeutig finden lässt, verweigert er** und nennt Datei und
  Schlüssel: gepunktete Schlüssel (`commit.language = …` auf oberster Ebene),
  Inline-Tabellen, eine Sektion, die zweimal vorkommt. Solche Stellen ändert
  der Mensch von Hand.
- Nach jeder Änderung liest der zuständige Lader den neuen Text; erst wenn er
  ihn annimmt, wird über `lock.ReplaceText` geschrieben.

### Oberfläche

Ein Paket `internal/tui` auf `golang.org/x/term`:

- **Bausteine:** gruppierte Liste mit Cursor und Suche (`/`), Auswahl aus
  wenigen Möglichkeiten, Ja/Nein, Texteingabe mit Prüfung beim Bestätigen,
  Mehrfachauswahl, eine Bestätigung, die den Diff der Datei zeigt.
- **Terminal als Naht:** Die Bausteine sprechen mit einer Schnittstelle
  (Taste lesen, schreiben, Größe). In den Tests spielt ein Tastenprotokoll die
  Eingabe ein, und die Ausgabe wird als Text verglichen. Nur die dünne Hülle um
  `term.MakeRaw` und die VT-Verarbeitung der Windows-Konsole
  (`ENABLE_VIRTUAL_TERMINAL_PROCESSING` über `x/sys/windows`) bleibt ohne
  Test und trägt `//coverage:exempt` mit Begründung.
- **Kein Terminal:** Ist stdin kein Terminal, verweigern die interaktiven
  Formen und nennen die nicht interaktive (`config list|get|set`, `init --yes`
  oder die Antwortflags).
- Die Oberfläche lädt nichts im Hook-Pfad; kein Hook-Paket importiert `tui`.
  Ein Tor-Test liest den Importgraphen, wie er es für `serve` schon tut.

### `loomux config`

- **`loomux config`** (interaktiv): die Liste aus dem Schema, nach Modulen
  gruppiert. Jede Zeile zeigt Schlüssel, Wert und **Herkunft**: `set` (steht in
  der Datei), `default` (Vorgabe des Schemas) oder `preset` (aus einem Preset
  von `[verify]`). Eine Zeile wählen, Wert ändern, Diff sehen, bestätigen.
  Tabellen und Tabellenlisten zeigt sie mit Anzahl und Inhalt, ändert sie aber
  nicht; eine Policy-Regel aus dem Katalog von `init` kann sie ergänzen.
- **`loomux config list [--json]`**: dasselbe ohne Terminal.
- **`loomux config get <sektion.schlüssel>`**: ein Wert, ohne Herkunft, für
  Skripte.
- **`loomux config set <sektion.schlüssel> <wert> [--yes]`**: zeigt den Diff
  und fragt; `--yes` bestätigt. Ein Wert gleich der Vorgabe entfernt die
  Zeile; ein Zauberwort `default` gibt es nicht (entschieden 2026-09-24).
- **`loomux config unset <sektion.schlüssel> [--yes]`** (entschieden
  2026-09-24): nimmt die Zeile heraus, derselbe Weg aus Diff, Frage und
  Lesern. Die interaktive Form bietet Bool- und Enum-Schlüsseln mit Vorgabe
  dafür die Wahl `(default)`.
- `--root` wie überall, sonst die erste `.loomux/config.toml` aufwärts.
- **`--global`** (entschieden 2026-09-24): dieselben Formen über
  `<zustand>/config.toml`, mit demselben Schema, Editor und denselben
  Bestätigungen. In 4a-1 kommt nur der Schalter; die globale Datei kennt dann
  noch keinen Schlüssel. `[model]` trägt 4c nach. Die Wächterregel gilt für
  `--global` ebenso.
- Exit: 0 bei Erfolg, 1 wenn ein Lader den neuen Text ablehnt oder der Editor
  die Stelle nicht findet, 2 bei falschem Aufruf.

### `[modules]` zur Laufzeit

Fehlt die Tabelle oder ein Schlüssel, ist das Modul **an**; ein Repo ohne
`[modules]` verhält sich wie heute.

| Schlüssel | `false` bewirkt |
|---|---|
| `hooks` | post-edit, `stop`, `session-start` und `subagent-*` enden sofort mit 0. Die Git-Hooks (`loomux check …`, commit-msg) bleiben unberührt: Sie schaltet man ab, indem man sie nicht einrichtet. Der Wächter (`pre-tool-use`) prüft **immer**, weil die Schreibschranke global ist und auch andere Repos schützt (entschieden 2026-09-24) |
| `brain` | Die Brücke bietet keine `brain_*`-Werkzeuge an, die Lane `lint/wiki` läuft nicht, `convert` und `fetch` verweigern mit Hinweis auf `[modules]` |
| `graph` | Die Brücke bietet keine `graph_*`-Werkzeuge an |

**Die Brücke kennt das Projekt:** `loomux mcp` sucht vom Arbeitsverzeichnis
aufwärts die erste `.loomux/config.toml`, liest ihr `[modules]`, filtert
`tools/list` und verweigert den Aufruf eines abgeschalteten Werkzeugs als
Werkzeugfehler. Findet sie keine, gilt alles als an. `--root` gibt es für den
Aufruf von Hand; in eine eingecheckte Datei schreibt `init` es nicht, weil
dort ein Pfad dieses Rechners stünde. `serve` bleibt maschinenweit und
unverändert, ebenso seine Aufholung, die an keinem Projekt hängt.

**Das trägt nur, wenn der Host den Server im Projekt startet.** Welches
Arbeitsverzeichnis Claude Code und Antigravity einem stdio-Server aus dem
Nutzerbereich geben, ist nicht gemessen; es ist die **erste Aufgabe des
Plans von 4a-1**. Gilt es nicht, filtert die Brücke nicht, bietet alle
Werkzeuge an und schreibt einmal je Start auf stderr, dass `[modules]` für
MCP nicht greift. `[modules]` wirkt dann nur auf Hooks, Lanes und Befehle.

**Der Hook-Pfad bleibt schlank:** `[modules]` liest ein kleiner Leser, der
nur diese Tabelle kennt. Ein Tor-Test liest den Importgraphen und hält
`schema`, `edit` und `tui` vom Hook-Pfad fern, wie er es für `serve` schon
tut.

### Die Wächterregel

Eine dritte eingebaute Befehlsregel neben `git push` und dem Schreiben des
Manifests: Der Wächter verweigert einem Agentenwerkzeug `loomux init` ohne
`--dry-run` oder `--detect-only`, das interaktive `loomux config`,
`loomux config set`, `config --global` in schreibender Form und
`loomux area add`. `config list` und `config get` bleiben erlaubt, dazu ein
alleinstehendes `config --help` oder `-h`; jede andere Form von `config`
wird verweigert, auch eine später hinzukommende (entschieden 2026-09-24).
Erlaubt sind seit dem Entscheid „Vorschläge statt Schreiben für Agenten“
außerdem `config proposals` und `config set|unset` mit `--propose`. Ob ein
Flag (`--propose`, `--dry-run`/`--detect-only` von `init`) befreit,
entscheidet eine Positivliste statt einer Liste verbotener Shell-Syntax, die
in jeder Prüfrunde wuchs (Klammer-Expansion, Globs, PowerShell-Klammern):
Der Aufruf ist **direkt**, loomux (oder `go run` von `cmd/loomux`) ist
also das erste Wort eines Befehls, den die Zeile selbst beginnt (die Grenze
davor liegt außerhalb von Anführungszeichen), ohne Wrapper, Zuweisung,
Umleitung, Schlüsselwort oder `{` davor, denn ein Wrapper wie `cmd /c` liest
die Wörter ein zweites Mal (`^`, `%X%`, `!X!` auch in einfachen
Anführungszeichen). Ein loomux in einer gequoteten inneren Zeichenkette
wird gefunden und verweigert, ist aber nie befreit. Die Zeile, wie geschrieben, trägt außerhalb von
Anführungszeichen nur Buchstaben, Ziffern, Leerraum, `. _ / : = , + -`, die
Trenner `;`, `|`, `&&`, `<`/`>` (mit `&` in einer Umleitung) und
Kommentare, aber kein alleinstehendes `&`; in doppelten Anführungszeichen
nur dieselben Zeichen, in einfachen jedes ASCII-Zeichen außer
`; | & ( ) < >` und `^ % !`, die ein Wrapper erneut lesen könnte, und `#`,
das als Wort ankäme, das der Wächter als Kommentar liest. `%` fehlt, weil
PowerShell nach `--%` den Rest der Zeile mit aufgelöstem `%X%` weitergibt.
Das Flag steht als eigenes Wort vor jedem `--`, ohne `-name=…` daneben, und
nach einer Umleitung folgen nur Umleitungen. Alles andere wird verweigert;
ein Wert mit `$`, `*` oder `{…}` steht darum in einfachen
Anführungszeichen. `config set|unset` selbst weist ein `--propose`, das am
Ende falsch ist, mit Exit 2 ab (2026-09-24). `area add`
zu verweigern (freigegeben 2026-09-24) schließt (die Lücke seit 3a, siehe Befunde). Das
ändert einen ausgelieferten Befehl; Anweisungen an Agenten, `area add` zu
rufen, gibt es heute nur in Plänen, Akten und der Befehlsreferenz, nicht in
AGENTS.md, `.claude/` oder den READMEs. Seit 4d verweigert der Wächter auch
`convert` und `fetch`, außer allein mit `--help` oder `-h` (Vorschlag 4 in
„Abweichungen beim Planen von 4d“). Erkannt
wird der Aufruf an jedem Befehlsbeginn einer Zeile, mit dem Programmnamen
`loomux`, `loomux.exe` oder einem Pfad darauf und mit `go run` des Pakets
`cmd/loomux` in seinen Schreibweisen, hinter Zuweisungen, Umleitungen,
reservierten Wörtern und Wrappern. Welche Formen genau erkannt werden, welche
Lücken bleiben und welche Fehlverweigerungen in Kauf genommen sind, führt die
Befehlsreferenz (`docs/de/cli-reference.md`, `hook pre-tool-use`); die Tests
in `internal/hooks/guard_test.go` halten jede dieser Listen fest. Die Grenzen
sind dieselben wie bei der Manifestregel: ein Netz mit bekannten Löchern,
kein Beweis.

### Abweichungen beim Bau von 4a-1

Beim Bau (2026-09-24) entschieden; der Text oben gilt, wo er nicht
widerspricht:

- **Keine Mehrfachauswahl.** `internal/tui` hat Liste, Eingabe mit Auswahl
  und Bestätigung, aber noch keine Mehrfachauswahl; `config` braucht keine.
  Sie kommt mit 4a-2, wo `init` die Module wählen lässt.
- **Die Tabellen je Stack sind keine Schlüssel des Schemas.**
  `[verify.<stack>.<art>]` ändert ein Mensch von Hand; `loomux check
  precommit --show` zeigt, was sie mit den Presets ergeben. `config list`
  führt sie nicht auf.
- **Ein abgeschaltetes Werkzeug ist kein Werkzeugfehler.** Die Brücke lässt
  es in `tools/list` einfach weg; ein Aufruf trifft darum den Protokollfehler
  des SDK für ein unbekanntes Werkzeug, nicht einen Werkzeugfehler mit
  Hinweis auf `[modules]`. Ein Wirt ruft nur, was gelistet ist.

### Abweichungen beim Planen von 4a-2

Beim Planen (2026-09-24) gegen den Code gelesen und mit dem Nutzer
entschieden; die Befunde stehen im Plan
(`plans/2026-09-24-loomux-stufe-4a-2.md`). Der Text unten gilt, wo er nicht
widerspricht:

- **Umzug schmal.** Mit ihren Tests ziehen nur `write` und das Zusammenführen
  aus `settings` um. `answers` und `interview` werden auf Schema und `tui` neu
  geschrieben, `render` bleibt als zwei Vorlagen (`AGENTS.md`,
  `verify-until-green`). `detect` und `gitenv` gibt es schon; `commit`,
  `coverage`, `verify`, `tomlstr`, `tooling` und `ulinit check …` fallen weg.
- **`answers.toml`** hält nur, was kein Schlüssel des Schemas ist: Hosts und je
  Modul die gewählten Teile.
- **Besitz eines Host-Eintrags** ohne Marke: Er gehört `init`, wenn sein Befehl
  ein loomux-Binary ruft. Fremde Einträge, auch `ulguard` und `brain guard`,
  bleiben stehen und werden gemeldet.
- **`merge-hook status`** kennt `stale path`, `stale branch` und `shared hook
  path` nicht, weil der Hook nichts einbackt; `record` prüft Common-Dir und
  Zweig zur Laufzeit.
- **`.mcp.json`** ruft `${LOCALAPPDATA}/loomux/bin/loomux.exe mcp --channel
  local`, dieselbe Form wie die Host-Einträge, statt `loomux` über den `PATH`.
- **Erstinstall:** `selfupdate.Run` aktualisiert nur ein Binary, das schon am
  kanonischen Ort liegt; `init` bekommt dafür `selfupdate.Install` mit
  derselben Kette aus `gh`, `SHA256SUMS`, `--version` und `swap`.

## 4a-2 im Einzelnen

### Umzug

Der Go-Code von ulinit zieht **mit seinen Tests** um, nach `internal/setup/…`:
`answers`, `interview` (auf die Bausteine aus `tui` umgestellt), `render`,
`settings` (das Zusammenführen in `.claude/settings.json` und
`.agents/hooks.json`), `write`, `tooling`, `journal`. `internal/detect` gibt
es schon. **Nicht** mit zieht `vendoring` samt `--vendor-url`/`--vendor-ref`
(die Python-Laufzeit gibt es nicht mehr) und `--install-tools`: `init`
installiert keine fremde Software, auch nicht unter `--yes`. Ebenso wenig
ziehen die Vorlagen von `render` für `brain.toml`, `policy.toml` und
`.ultraloom/config.toml` mit: Diese Formate gibt es nicht mehr. Sie fallen
samt ihren Tests weg; `.loomux/config.toml` schreibt der Editor aus 4a-1.

Die Antworten landen in `.loomux/state/answers.toml`, die installierten Stände
in `.loomux/state/installed.toml`.

### Ablauf

1. **Erkennen:** Stacks, Hosts, vorhandene Dateien. `--detect-only` gibt das
   als JSON aus und endet.
2. **Module der Reihe nach**, je mit alles/einzeln/nichts; Flags
   `--hooks=all|ask|none`, `--brain=…`, `--graph=…`; `--yes` heißt überall
   `all` mit den Vorgaben.
3. **Basis zuletzt:** `[modules]`, `[commit]`, `[policy]`, `[worktree]`,
   Binary, Projektdateien, Prüfung der externen Programme.
4. **Zusammenfassung:** Jede Datei mit Diff, jede Änderung einzeln bestätigt
   (bei `--yes` alle). `--dry-run` zeigt dasselbe und schreibt nichts.

| Modul | Teile | Vorgabe |
|---|---|---|
| **Basis** | Binary an den kanonischen Ort (neuestes Release über `selfupdate`), `.loomux/config.toml` (`[modules]`, `[commit]`, `[policy]` aus einem Regelkatalog, `[worktree]`), `.gitignore` (#11), `AGENTS.md` nur wenn keine da ist (#10), Prüfung von `git`, `qmd`, `pdftotext`, `yt-dlp`, Ollama (nur melden, mit Installationsbefehl) | — |
| **Hooks** | Wächter, post-edit, session-start, stop, subagent-start/-stop je Host; Git: pre-commit, pre-push, commit-msg; `[verify]` (nur Abweichungen vom Preset); Skill `verify-until-green` auf `loomux check all` (#8) | alle, für die erkannten Hosts, sonst `claude` |
| **Wiki (Brain)** | Bereich über `area add` samt `reindex`; `[area]`, `[layout]`, `[wiki]`, `[index]`, `[privacy]`; post-merge-Hook (#5); die fünf Brain-Skills (#7) | alle; `[area].scope = project/<name>`, `[layout].wiki = docs/wiki` |
| **Graph** | ein erster `graph build` | an |

Ein zweiter Lauf fragt, was offen ist oder sich ändern soll, und verhält sich
bei `.loomux/config.toml` wie in 4a-1 beschrieben. Eine andere vorhandene
Datei überspringt er und meldet sie. Die Commit-Sprache (#13) ist eine Frage
wie jede andere; `[relevance]` (#12) kennt `init` nicht.

Ein Agent startet `init` nur mit `--dry-run` oder `--detect-only`
(Wächterregel); schreibend läuft es nur, wenn ein Mensch es aufruft und jede
Zeile bestätigt.

**`.mcp.json`:** `init` schreibt einen Eintrag `loomux mcp` ohne Pfad, aber
nur, wenn der Nutzerbereich des Hosts keinen Server `loomux` kennt; sonst
meldet es die vorhandene Registrierung und lässt die Datei weg.

### Host-Einträge

Aus einer Tabelle je Host, wie sie die Fusions-Spec unter „Hosts“ führt:
`.claude/settings.json` für Claude Code, `.agents/hooks.json` für
Antigravity. Jeder Eintrag ruft das Binary am kanonischen Ort über eine
Umgebungsvariable, nicht über den `PATH` (entschieden 2026-09-24):
`"${LOCALAPPDATA}/loomux/bin/loomux.exe" hook <ereignis> --host <h> --root
"${CLAUDE_PROJECT_DIR}"` — dieselbe Form, in der loomux heute
`${CLAUDE_PROJECT_DIR}` benutzt. Der Eintrag löst sich auf jedem
Windows-Rechner gleich auf, ist eincheckbar und ruft genau das Binary, das
`serve` aktuell hält. Der Grund: Claude Code blockiert nur bei Exit 2; ein
Eintrag, dessen Programm die Shell nicht findet (Exit 127), ließe jeden
Werkzeugaufruf am Wächter vorbei. `init` schreibt die Einträge deshalb erst,
wenn das Binary dort liegt. Dieselbe Form gilt für die Git-Hooks eines Wirts
und den post-merge-Hook. Vor dem Bau zu messen: ob Antigravity `${…}` im
Hook-Befehl ebenso auflöst.

Vor dem ersten Umschreiben sichert `init` die Datei als `.bak`. Fremde
Einträge bleiben stehen.

Für `.agents/hooks.json` heißt das genauer (Fusions-Spec #21, aus
`internal/agenthooks` auf dem ultraloom-Zweig `claude/wiki-stufe-2`, der als
Ausgangspunkt umzieht): Die Datei ist eine Map benannter Gruppen, und der Name
ist die Identität. `init` besitzt genau eine Gruppe, `loomux`, ersetzt nur sie
und kodiert jede andere byte-treu aus dem gelesenen JSON neu. Ein
Besitzerfeld im Hook-Objekt gibt es nicht, weil ungemessen ist, ob
Antigravity ein unbekanntes Feld duldet; ein strenger Leser ließe den Hook
still fallen. Eine fremde Gruppe, die dasselbe Kommando führt, wird gemeldet,
nie repariert. Eine Wurzel, die kein Objekt ist, `null` eingeschlossen, wird
abgelehnt, und die Meldung nennt die Datei.

### Binary an den kanonischen Ort (#15)

Ziel ist der **kanonische Ort** der Self-Update-Spec
(`2026-09-23-self-update-design.md`, Zweig `feat/self-update`):
`<Zustandsverzeichnis>/bin/loomux.exe`, also
`%LOCALAPPDATA%\loomux\bin\loomux.exe`. Dort liegt schon das Binary, das die
MCP-Registrierung ruft, und `serve` hält es täglich auf dem neuesten Release.
`init` benutzt dafür `internal/swap`, das jener Zweig aus `dev/swap` zieht,
weil Windows ein laufendes `.exe` nicht überschreibt. Den `PATH` fasst `init`
nicht an und braucht ihn nicht; wer `loomux` im Terminal tippen will, bekommt
den Befehl genannt, mit dem er den Ort einträgt.

**Was dort landet (entschieden 2026-09-24):** immer das neueste Release,
geholt über `selfupdate` — nie eine Kopie des laufenden `init`, auch nicht,
wenn es selbst ein Release ist. So bleibt die Regel der Self-Update-Spec
unberührt: am kanonischen Ort liegt nie ein Checkout-Build. Fehlt `gh` oder
scheitert der Abruf, fällt nur dieser Schritt aus, und `init` nennt
`gh auth login` und `loomux upgrade`.

Ob die Warnung des Sitzungsstarts aus `update.json` ein veraltetes Binary in
Wirten schon abdeckt, wird mit dem Self-Update-Zweig abgestimmt.

### post-merge (#5)

Neuschrift von `merge_events.py`: `loomux hook install|status|remove` wird
zu `loomux merge-hook install|status|remove` (`hook` ist bei loomux der
Namensraum der Wirts-Hooks).

**Ein Weg für alle (entschieden 2026-09-24):** Der Hook ist ein kurzes sh ohne
eingebackenen Werte. Er ruft `"${LOCALAPPDATA}/loomux/bin/loomux.exe"
merge-hook record` (siehe „Host-Einträge“) und endet
immer mit `exit 0`, auch wenn das Binary fehlt oder scheitert. `record` liest
Repo, Zweig und `ORIG_HEAD..HEAD` selbst, prüft die Einwilligung des Bereichs
(Zweig aus dessen Manifest) und hängt die Zeile an
`<zustand>/maintenance/merge-events.tsv` — dieselbe Zeile wie die Referenz.
Ohne Einwilligung schreibt es nichts. Weil der Hook keinen Rechnerpfad enthält,
darf er eingecheckt werden; bei `core.hooksPath` im Arbeitsbaum (loomux:
`.githooks/`) legt `install` ihn dort ab wie in `.git/hooks`. Die Referenz
wählte reines sh wegen des Starts von `uv` und des `#` in `#GIT`; beides gilt
für ein Go-Binary nicht. Der Preis ist ein Binary-Start je Merge.

Marke, `status` mit „drifted“ und „orphaned“ und die Weigerung, eine fremde
Datei zu überschreiben oder zu löschen, bleiben wie in der Referenz. Die
Hookdatei selbst weicht ab (kein eingebackener Pfad) und steht als
freigegebene Abweichung in der Akte.

### Skills

Die fünf Brain-Skills (#7) werden übersetzt und auf loomux-Befehle
umgeschrieben (`brain check all` → `loomux brain check all` usw.), sonst
inhaltlich gleich; eine Überarbeitung gehört zu W4. `verify-until-green` (#8)
ruft `loomux check all`. Die Vorlagen liegen eingebettet und werden beim
ersten Gebrauch geladen. `init` schreibt sie nach `.claude/skills/` bzw. den
Skill-Ort von Antigravity.

## 4c im Einzelnen

- **Kern**, neu geschrieben aus `model/` (672 Zeilen):
  - Ollama-Client über `net/http`, nur Loopback. Eine Adresse außerhalb ist ein
    Konfigurationsfehler und wirft; ein Ausfall (nicht erreichbar, zu spät,
    Unsinn) ergibt „kein Vorschlag“ und führt **nie** in die Cloud.
  - Das Tor: Ein Proposer entsteht, oder er entsteht nicht; ohne Freigabe gibt
    es keine eingerichtete Adresse.
  - Die Richter: Deutsch an Funktionswörtern, nicht an Umlauten; zerhackte
    Wörter werden gezählt. Die Funktionswortliste (die „Zipf-Tabelle“ der
    Fusions-Spec) ist eingebettet und wird beim ersten Gebrauch geladen.
  - Die Prompts als eingebettete Dateien, Byte für Byte die Referenz, mit
    Versionsnummer im Fall.
- **`[model]`**: global in `<zustand>/config.toml`, je Bereich in
  `.loomux/config.toml`. Aus schlägt an, nur in diese Richtung. Die Sektion
  kommt ins Schema von 4a-1 (Modul Brain) und damit in `config` und `init`.
- **Wirkung:** `reconcile` öffnet einen `local_only`-Fall mit Vorschlag;
  `propose` prüft ihn mit `evidence` aus 3b, derselben Messlatte wie im
  Betrieb. Ein abgelehnter Vorschlag fällt auf den Fall ohne Vorschlag zurück.
- **Heilung #1:** `approve --reject` schiebt `sources[]` der Seite und das
  Register auf den geprüften Stand vor, sodass der nächste Abgleich den Fall
  nicht wieder öffnet.
- **Heilung #4:** `reindex` und `approve` nehmen dieselbe Sperre je Bereich
  unter `<zustand>/areas/<scope>`.
- **`loomux dev bench search`**: Qualität (Rang unter den ersten drei, nie der
  Score) und Latenz (kalt, dann N warm) über Korpus und Fragensatz. Korpus und
  Fragensatz ziehen nach `testdata/bench/`, mit ihren Regeln (feste Zahlen,
  Partition, Herkunft je Datei, Prüfsumme je Notiz). Der Korpus `v1` sind
  wörtliche Abschnitte aus Wikipedia (`HERKUNFT.md`), keine privaten Notizen;
  er steht unter **CC BY-SA 4.0** und behält diese Lizenz samt Quellenangabe
  in seinem Verzeichnis, getrennt von der Lizenz von loomux.
- **Ein Berichtsschema** für `dev bench hooks|corpus|search`: Kopf mit
  Umgebung, Modell und Profil, ein kalter Lauf, N warme mit Median, Minimum
  und Maximum, Ausgabe als JSON und Markdown. `benchhooks` und `benchcorpus`
  werden darauf umgestellt.

### Abweichungen beim Planen von 4c

Beim Planen (2026-09-25) gegen den Code und gegen die Referenz am Tag
`loomux-3-source` gelesen und mit dem Nutzer entschieden; der Text oben gilt,
wo er nicht widerspricht.

**Schnitt.** 4c zerfällt in **4c-1** (Modell, `propose`, Heilung #1 und #4)
und **4c-2** (`dev bench search`, das Berichtsschema). Beide hängen nicht
aneinander, jede bekommt einen eigenen Plan, eine eigene Akte
(`parity/stufe-4c-1.md`, `parity/stufe-4c-2.md`) und einen eigenen Pull
Request. 4d hängt nur an 4c-1.

**Korrekturen, die der Code erzwingt:**

- **Die Richter gehören zu 4d.** `propose` prüft nur mit `evidence`;
  `is_german`, `chopped_words` und `is_one_sentence` braucht nur `describe`.
  Die Spec vermengt zwei Dinge: eine Liste von 72 Funktionswörtern, die in
  der Referenz im Code steht (`judge.py:34-40`), und die Zipf-Frequenzen aus
  `wordfreq`, die nur `chopped_words` braucht. Größe und Lizenz einer
  eingebetteten Zipf-Tabelle sind eine Frage von 4d. Die Funktionswortliste
  bettet 4c-2 ein, weil die Korpusprüfung die Richtung einer Frage daran
  erkennt.
- **Das Tor kennt keine Freigabe und keine Umgebungsvariable.** Ein Client
  entsteht genau dann, wenn `enabled` an ist und die Rolle aktiv
  (`gate.py:17-29`). Erst dann wird der Endpoint geprüft; ein falscher
  Endpoint bei ausgeschaltetem Modell bleibt unbemerkt, wie in der Referenz.
- **`[model]` hat zwei verschiedene Schlüsselmengen.** Global
  (`<zustand>/config.toml`): `enabled` (Vorgabe `false`), `endpoint`
  (`http://127.0.0.1:11434`), `name`
  (`hf.co/unsloth/gemma-4-E4B-it-qat-GGUF:UD-Q4_K_XL`), `temperature` (`0.0`,
  0 bis 2, kein Bool) und `roles` (Vorgabe alle drei an; ist `roles`
  gesetzt, ist jede nicht genannte Rolle aus). Je Bereich nur `enabled` und
  `roles`. Die Einengung: `enabled` gilt, wenn es global an und im Bereich
  nicht `false` ist; die Rollen sind die Schnittmenge. Unbekannte Schlüssel
  in `[model]` bleiben still übergangen, eine unbekannte Rolle wird
  abgewiesen, beides wie in der Referenz.
- **Der globale Teil fehlt.** Die Bereichsschlüssel stehen seit 4a-1 im
  Schema und werden von `ReadDeclaration` geprüft, aber `Manifest` verwirft
  sie, `schema.GlobalKeys()` ist leer, und für `<zustand>/config.toml` gibt
  es keinen Leser. 4c-1 füllt `GlobalKeys`, schreibt den Leser, hängt ihn in
  die Prüfung von `config --global` und gibt `Manifest` die Felder
  `ModelEnabled` und `ModelRoles`.
- **Der Client** folgt der Referenz, nicht den Vorgaben von Go: nur `POST
  <endpoint>/api/generate` mit `model`, `prompt`, `stream:false`,
  `think:false`, `options{temperature, num_ctx:8192}`; kein Proxy aus der
  Umgebung (`trust_env=False`), keine Weiterleitung (eine 3xx ist ein
  Ausfall), 2 s für den Verbindungsaufbau, 30 s insgesamt je Frage. Vor der
  ersten Frage eines Clients lädt und wärmt loomux das Modell mit denselben
  Optionen und einer Antwort von einem Token, ohne Gesamtlimit; scheitert
  das, gilt das Modell für diesen Client als ausgefallen (Nachtrag
  2026-09-29, entschieden vom Nutzer nach der Messung vom 2026-09-28: die
  erste Frage nach dem Laden lief in die 30 s). Jeder Ausfall ist
  „keine Antwort“; ein leerer `response` ist eine Antwort. Der Request hängt
  am Kontext des Aufrufers, damit `serve` einen laufenden Aufruf beim Beenden
  abbricht. Loopback heißt: Host als Zeichenkette gleich `127.0.0.1`,
  `localhost` oder `::1`, keine Namensauflösung; die fünf Fehler der
  Referenz in ihrer Reihenfolge (`client.py:59-109`).
- **Der Prompt ist die ganze Datei** `vorschlag-v4.md`, samt dem
  Versionskommentar am Anfang; eingesetzt wird nur `{paket}`. Ein Test hält
  fest, dass die Datei sonst keine geschweiften Klammern trägt, sonst wäre
  die Ersetzung nicht mehr gleich `str.format`.
- **`propose` gibt es nur für `local_only`-Bereiche.** Ohne einen solchen
  Bereich liest `reconcile` die globale Datei gar nicht; ein
  Konfigurationsfehler wird ein Fehler von `reconcile`. Eingehängt wird in
  `landCase` zwischen `RenderPackage` und `WriteCase`. Mit Vorschlag:
  `proposal.md` mit dem Rohtext, `prompt_version = "vorschlag-v4"`,
  `local_only = true`, kein `manual`, keine `note`. Mit Proposer, aber ohne
  brauchbaren Vorschlag: `manual = true` und die zweite Notiz der Referenz,
  „manual review: the local proposer returned no usable proposal“. Ein
  stehender Fall wird nicht erneut gefragt.
- **Wo das Modell läuft:** in `reconcile`, im Abgleich vor `reindex`, nach
  `approve` und im Upkeep von `serve`; auf keinem Hook-Pfad. Im schlimmsten
  Fall kostet ein Fall 30 s, wie in der Referenz, dazu einmal je Client das
  Laden des Modells.
- **Die Sperre von #4 liegt neben dem Bereichsverzeichnis**, nicht darin:
  `<zustand>/areas/<scope>.lock` (flach geschrieben wie das Verzeichnis).
  `ReplaceDir` tauscht `<zustand>/areas/<scope>` als Ganzes, und eine offene
  Datei darin hielte unter Windows den Tausch auf; ein schreibbarer Bereich
  hat das Verzeichnis gar nicht, bekommt die Sperre aber auch. Beide nehmen
  sie blockierend: `reindex` um das Schreiben des Bestands je Bereich,
  `approve` um `moveStock` und das Vorschieben des Registers. Keiner hält
  sie über einen qmd-Aufruf.
- **Für die Aufzeichnung** gegen Python braucht es eine Ollama-Attrappe als
  eigenen Prozess auf Loopback; `fakeqmd` gibt es nur als Handler im selben
  Prozess. Neu ist `loomux dev fake-ollama --fixture <json>` auf einem
  festen Port, den die Welt in ihrer `config.toml` nennt. Im Go-Test hängt
  derselbe Handler als `httptest`-Server an einer Naht. Aufgezeichnet werden
  sieben Fälle: guter Vorschlag, erfundenes Zitat, Ollama läuft nicht,
  Modell aus, `manual_cloud` fragt nie, Endpoint außerhalb von Loopback,
  kaputtes `[model]`.
- **Bench-Namen.** `dev bench-hooks` und `dev bench` gibt es, `dev bench
  corpus` nicht, und „corpus“ heißt dort die OSS-Matrix. Siehe unten.

**Mit dem Nutzer entschieden (2026-09-25):**

| Frage | Entscheidung |
|---|---|
| Schnitt | 4c-1 und 4c-2, wie oben |
| Heilung #1 | `approve --reject` läuft erst durch `guardSources`: Hat sich eine Quelle seit der Fallbildung bewegt, hält es an wie `approve`, und der Fall bleibt. Sonst schiebt es nur `revision` und `content_hash` in `sources[]` der Seite vor, dazu das Register; `generated` und `verified` bleiben, denn die Seite ist weder neu erzeugt noch bestätigt. Ein Commit nimmt Seite, Register und `audit.md`. `TestRejectAdvancesNeitherThePageNorTheRegister` dreht sich um, der Fall `3b/approve/reject` wird freigegebene Abweichung |
| Bench-Befehle | Eine Untergruppe: `dev bench hooks` (heute `dev bench-hooks`), `dev bench repos` (heute `dev bench`), `dev bench search`. `dev bench` allein zeigt die Hilfe; die alten Namen fallen weg. Die Einträge der Chronik in `benchmarks.md` bleiben, wie sie sind, und die Fallsätze bleiben flach in `testdata/bench/` |
| Release von 4c-2 | Die Umbenennung ist ein „breaking change to a command“ nach AGENTS.md: 4c-2 geht als `release:major` |

**4c-2 im Einzelnen:**

- **`internal/dev/benchreport`**, eine Hülle für alle drei Befehle:
  `schema: 1`, `command`, `stamp`, `environment{os, arch, cpu, go, loomux,
  qmd, models, profile}`, `timings[]` aus `{name, cold_ms, warm_ms[],
  median_ms, min_ms, max_ms, exit_codes}`, dazu eine Nutzlast je Befehl
  (Trefferqualität bei `search`, Lückenaudit bei `repos`). Zeiten durchweg in
  Millisekunden als Zahl; heute schreibt `benchcorpus` Nanosekunden.
  JSON und Markdown aus demselben Wert; Median, Minimum und Maximum an einer
  Stelle statt zweimal. `docs/benchmarks.json` wird im selben Pull Request
  einmal umgerechnet, mit einem Wegwerfskript, das nicht eingecheckt wird.
- **`dev bench search`**, übertragen von `brain bench` (`cli.py:503-540`,
  `2011-2340`, `bench/*.py`): Rang der erwarteten Quelle unter den ersten
  zehn, Treffer ist Rang ≤ 3, über die volle Suchkette samt
  Privatsphärefilter; `--latency` misst `catalog`, `read`, `keyword`,
  `fast` und `full` je einmal kalt und `--repeat` mal warm, nach einem
  ungezählten Probelauf. Profile `keyword|fast|full`, Vorgabe `fast`.
  Fragensatz mit `id`, `sort`, `query`, `expect`, `beleg`, `hinweis`, Form
  13/13/10/14, `beleg` wörtlich im Ziel. Der echte Fragensatz bleibt privat,
  Vorgabe `<bereich>/98 Messung/questions.yaml`. Ausgabe: `.md` und `.json`
  beide oder keine, das Markdown auch auf stdout; eine vorhandene Zieldatei
  bricht ab; im Korpusmodus ist `--out` Pflicht. Exit 1 bei jedem Fehler des
  Korpus, des Fragensatzes oder der Suche; keine Schwelle für Qualität oder
  Latenz.
- **Korpusmodus `--corpus v1`:** erst die Prüfungen (sha256 je Notiz gegen
  `manifest.json`, `HERKUNFT.md` in beide Richtungen, Partition 100 Notizen
  in 10 Themen zu je 10, Nachbarthema vorhanden und verschieden, mindestens
  fünf Fragen in Gegenrichtung), dann ein Wegwerf-Zustand. qmd läuft dabei
  mit eigenem `XDG_CONFIG_HOME` und `XDG_CACHE_HOME`, damit weder die echte
  `index.yml` noch der echte Index berührt wird; aufgeräumt wird immer.
- **Daten:** `testdata/bench/search/v1/` mit `notes/`, `HERKUNFT.md`,
  `manifest.json`, `themes.yaml`, `questions.yaml` und `baseline/` (43/50
  bei `fast`), dazu eine Lizenznotiz für **CC BY-SA 4.0**. `.gitattributes`
  setzt dort `-text`, denn die Prüfsummen gelten den Rohbytes.
- **Parität:** Die Tests laufen gegen `fakeqmd`. Einmal von Hand und in
  der Akte festgehalten: `dev bench search --corpus v1 --profile keyword`
  gegen `brain bench` mit demselben qmd, gleiche Ränge je Frage. `fast` und
  `full` hängen an den Modellen von qmd; sie werden gemessen, nicht
  verglichen, und die Latenz nie.

**Selbstnutzung:** 4c-1: `project/obsidian-ai` ist auf diesem Rechner
`local_only`; mit laufendem Ollama und dem Referenzmodell (heruntergeladen,
2026-09-25) öffnet ein geänderter Quelltext dort einen Fall mit Vorschlag.
Keine Wikiseite des Bereichs zitiert heute eine Quelle; der Mensch legt eine
an (entschieden 2026-09-25). Dazu die offene Abnahme der Referenz
(Scheibe-6-Spec §7 und §9, `OFFENE_AUFGABEN.md` Abschnitt 3, Task 10): der
Vorschlag entsteht unter `pktmon`-Mitschnitt ohne Paket außerhalb der
Schleife, die Gegenprobe mit abgeschaltetem Modell ebenso, und ein Vorschlag
braucht unter zwei Sekunden. Den Mitschnitt nimmt der Mensch in einer
Admin-Shell.
4c-2: `dev bench search --corpus v1 --profile fast --latency` gegen die
Baseline, eingetragen in beide `benchmarks.md`.

### Nachtrag 2026-09-26: `init` lädt das lokale Modell

Mit dem Nutzer entschieden. `loomux init` bekommt im Modul brain den Teil
`model`: Fehlt das in `[model] name` der globalen `config.toml` genannte
Modell in Ollama (`GET /api/tags`, ein Name ohne Tag gilt als `:latest`),
plant init die Aktion `model-pull` (`ollama pull <name>`), bestätigt wie jede
Änderung, in `--dry-run` gezeigt und nie ausgeführt. Ausgeführt wird sie über
`POST /api/pull` auf demselben Loopback-Transport, mit dem Fortschritt auf
stderr und ohne Gesamtzeitlimit; Ctrl+C beendet nur den Download. Ein
Fehlschlag ist ein Hinweis, init läuft weiter. Ist Ollama nicht erreichbar
oder weist der Wächter den Endpunkt ab, bleibt es beim Hinweis.

Vorgabe an nur bei `[privacy] mode = "local_only"` im Projekt oder
`[model] enabled = true` global, damit `init --yes` nicht ungefragt mehrere
GB lädt. Das ist die eine Ausnahme von „Werkzeuge werden geprüft, nie
installiert“ (4a-2). Geladen wird nur in init, nie in `reconcile`.

### Abweichungen beim Planen von 4c-2

Beim Planen (2026-09-26) gegen den Code und gegen die Referenz am Tag
`loomux-3-source` gelesen, gemessen und mit dem Nutzer entschieden. Wo dieser
Abschnitt „4c-2 im Einzelnen“ oder den Abschnitt davor widerspricht, gilt er.

**Korrekturen, die die Referenz erzwingt:**

- **Richtung einer Frage.** Die Korpusprüfung erkennt eine Frage in
  Gegenrichtung (englische Frage an eine deutsche Notiz, nur Sorte
  `sprachuebergreifend`) an eigenen Listen in `corpus.py:43-81`: 14 englische
  gegen 17 deutsche Wörter, Gegenrichtung heißt mehr englische als deutsche.
  Die 72 Funktionswörter aus `judge.py:34-40` braucht 4c-2 nicht; sie bleiben
  bei 4d. Der Satz oben, 4c-2 bette die Funktionswortliste ein, entfällt.
- **Probelauf.** Er ist eine `keyword`-Suche mit n=5, ungezählt; findet sie
  nichts, bricht der Lauf ab. Es gibt keinen Aufwärmlauf je Operation. Die
  Latenz läuft nach dem Qualitätsdurchgang; der Bericht sagt, dass die Kette
  dann schon warm ist. `--latency-query` hat die Vorgabe `latenz`, `--repeat`
  die Vorgabe 10 und mindestens 1; die drei Suchen der Latenz nehmen n=5.
- **Fragensatz.** Pflicht sind `id`, `sort`, `query`, `expect`, `beleg`;
  `hinweis` ist optional. Die Sorten heißen `exakt`, `umschreibung`,
  `gemischt`, `sprachuebergreifend` (Form 13/13/10/14, im Alltag wie im
  Korpus). `beleg` wird nach dem Zusammenfassen von Leerraum gesucht, nicht
  byteweise. `expect` gilt relativ zur Fragendatei. Doppelte `id`, eine
  unbekannte Sorte und ein fehlendes Ziel sind Fehler; alle Fehler werden
  gesammelt gemeldet.
- **Vorgaben der Befehlszeile.** Bereich `knowledge`; `--out` ist
  `<bereich>/98 Messung` und muss ein vorhandenes Verzeichnis sein; der
  Fragensatz ist `<out>/questions.yaml`, folgt also `--out`. Über mehrere
  Bereiche ist `--out` Pflicht. `--corpus` verweigert `--scope` und
  `--questions`. Die Dateien heißen `bench-<stempel>-<profil>.md` und `.json`
  (Stempel in UTC, `JJJJ-MM-TT-HHMM`); liegt eine davon schon da, bricht der
  Lauf vor der ersten Anfrage ab. Scheitert das Schreiben des JSON, wird das
  Markdown gelöscht. Das Markdown geht erst nach dem Aufräumen auf stdout.
  Ein Index ohne Dokumente bricht vor der ersten Frage ab. Ein Fehlschlag ist
  `error: <problem>` auf stderr, eine Zeile je Problem, Exit 1.
- **Je Frage** stehen `id`, `sort`, `rank` (bei einem Fehlschlag leer),
  `hit` und `elapsed_ms` im Bericht; eine leere Antwort wird einmal
  wiederholt und dann ein Befund `<anfrage>: <befund>`.

**Mit dem Nutzer entschieden (2026-09-26):**

| Frage | Entscheidung |
|---|---|
| Isolation im Korpusmodus | Nicht über `XDG_CONFIG_HOME` und `XDG_CACHE_HOME`: qmd legt seine Modelle unter `~/.cache/qmd/models` ab (2,7 GB, gemessen 2026-09-26), ein eigenes `XDG_CACHE_HOME` lüde sie bei jedem Lauf neu. Stattdessen `qmd --index loomux-bench-<zufall>`: qmd 2.8.3 legt Sammlung und Index dann in `<name>.yml` und `<name>.sqlite` ab und teilt die Modelle; eine Probe mit `collection add`, `update`, `embed`, `search`, `vsearch` und `query` ließ `index.yml` bitgleich und den Modellordner unverändert. Aufgeräumt wird immer: `<name>.yml`, `<name>.sqlite` samt `-wal` und `-shm` und eine Sicherung, falls der YAML-Schreiber eine anlegt. Ein Lauf hält für die ganze Dauer eine Sperre je Name unter dem Zustandsverzeichnis von loomux; einen `loomux-bench-*`-Index ohne gehaltene Sperre, den ein abgestürzter Lauf liegen ließ, entfernt der nächste Lauf, einen gehaltenen nie, denn oft laufen mehrere Sitzungen zugleich. Die Referenz isoliert gar nicht, sie trägt die Sammlung in die echte `index.yml` ein |
| Suchweg | Im Alltagsmodus über den Dienst (`QmdMcpPort`), wie loomux im Betrieb sucht; im Korpusmodus über die CLI (`QmdPort`) mit `--index`, denn nur sie trennt den Index, ohne den geteilten Dienst anzufassen. `environment.port` sagt `daemon` oder `cli`; die Latenzen beider Modi sind nicht vergleichbar. Gemessen 2026-09-26 auf dem echten Index: `fast` über den Dienst warm 331–348 ms, über die CLI 5,3–11,6 s je Aufruf, weil die CLI die Modelle jedes Mal lädt |
| `docs/benchmarks.json` | Bleibt ein Bestand je Repo, geführt von `MergeAudits`, und wird keine Liste von Läufen. Seine Zeiten tragen dieselbe Form wie die Hülle, `timings[]` in Millisekunden, einmal umgerechnet mit einem Wegwerfskript. Eine gesammelte Ablage der Hüllen je Lauf (auch als Debug-Modus) gibt es nicht |

**Was daraus für den Bau folgt:**

- **`benchreport`** ist die Hülle eines Laufs, die alle drei Befehle mit
  `--out` als `.md` und `.json` schreiben, und die einzige Stelle, die
  Median, Minimum und Maximum rechnet. `benchhooks.median` und
  `benchcorpus.calculateMedian` fallen weg. `dev bench hooks` schreibt wie
  heute Markdown auf stdout und mit `--out` zusätzlich beide Dateien.
- **qmd mit Indexnamen.** `index.QmdConfigPath` kennt heute nur
  `index.yml`; es bekommt den Namen. Im Korpusmodus trägt **jeder**
  qmd-Aufruf `--index <name>`: der Abgleich der Sammlungen, `update` und
  `embed` im Paket `index`, `ls` (davon lebt die Latenzoperation `read`),
  `status` und die Suche über `QmdPort`. Gemessen mit qmd 2.8.3: nur die
  JSON-Ausgabe der Suche hängt `?index=<name>` an die `file`-URI
  (`qmd://colla/baustatik-04.md?index=…`), `ls` und `status` nicht; der
  Parser der Suche schneidet es ab.
- **Modelle im Korpusmodus.** qmd schreibt in eine neue `<name>.yml` seine
  eigenen Vorgabemodelle. Dass sie heute denen der echten `index.yml`
  gleichen, ist Zufall. Der Lauf übernimmt deshalb den Block `models:` aus
  `index.yml` in `<name>.yml`, bevor er indiziert, und `environment.models`
  berichtet, was in `<name>.yml` steht.
- **Zustand für den Privatsphärefilter.** Der Wegwerf-Zustand geht über die
  Verzeichnisparameter von `privacy.VisibleAreas` und der Suchkette, nicht
  über `os.Setenv("LOOMUX_STATE_DIR")` im Prozess.
- **Die Zeilen von `docs/benchmarks.json`.** Aus `cold`, `warm[]` und den
  `*_median`-, `*_min`- und `*_max`-Feldern einer Zeile wird `timings[]`:
  ein Eintrag `total`, einer je Komponente und einer je Komponente der
  Claude-Baseline als `baseline:<name>`. Ein Eintrag trägt neben
  `{name, cold_ms, warm_ms[], median_ms, min_ms, max_ms, exit_codes}` bei
  Bedarf `applicable: false` und `timed_out` (Zahl der Läufe). Als eigene
  Felder bleiben `hook_warm_median_ms`, `claude_warm_median_ms`, `speedup`
  und `baseline_error`, dazu alle Felder außerhalb der Zeiten. Millisekunden
  mit sechs Nachkommastellen: Nanosekunden durch 10⁶ haben nie mehr, so ist
  die Umrechnung verlustfrei und der Wert derselbe `float64` wie
  `float64(ns)/1e6` im alten Renderer. Das Wegwerfskript gilt als richtig, wenn
  `matrix.md` und die Seiten je Repo aus dem umgerechneten JSON bytegleich
  zu den committeten entstehen.
- **`fakeqmd`** nimmt `--index` an und bekommt einen zusätzlichen
  Fixture-Schlüssel mit Treffern je Anfrage; heute übergeht es den Text der
  Anfrage, und ein Rangtest liefe ins Leere.
- **`dev bench`** allein zeigt die Hilfe der drei Unterbefehle.
  `docs/*/cli-reference.md` nennt heute nur `dev bench`, nicht
  `dev bench-hooks`; beide Sprachen bekommen `dev bench hooks|repos|search`.
- **`--timeout`** von `dev bench` wird gelesen, aber nie benutzt. Der Fehler
  liegt schon auf master und bekommt einen eigenen `fix`-Commit.
- **Daten.** Der Korpus kommt mit `git -c core.autocrlf=false archive
  loomux-3-source bench/corpus/v1` aus ultra-brain, nicht aus dessen
  Arbeitsbaum; beide Repos stehen auf `core.autocrlf=true`, und ohne den
  Schalter wendet auch `git archive` die Umwandlung an: 106 Dateien, 330 246
  Bytes, Baseline 43/50 bei `fast` auf qmd 2.8.3 (13/13, 11/13, 8/10,
  11/14). Vor dem Commit wird jede sha256 gegen `manifest.json` geprüft.
- **Paritätsprobe.** `brain bench --corpus v1 --profile keyword` trägt die
  Sammlung in den geteilten Index ein, loomux sucht im eigenen. Das ist
  nicht dasselbe: `keyword` rechnet die Wortgewichte über den ganzen Index,
  und der Sammlungsfilter ändert sie (gemessen 2026-09-26: dieselbe Notiz
  0,76 im Index mit 10 Notizen, 0,88 im Index mit 100, Ränge in vier
  Stichproben gleich). „Gleiche Ränge je Frage“ bleibt das Kriterium; ein
  abweichender Rang wird in der Akte gegen den Abstand der Scores
  nachgerechnet, bevor er als Fehler von loomux gilt. Die Referenz isoliert
  über `XDG_CONFIG_HOME`, `XDG_CACHE_HOME` und einen verlinkten Modellordner
  laufen zu lassen, wurde probiert: `qmd embed` hing dabei über zehn
  Minuten. Vor der Probe werden `index.yml` **und** `index.sqlite` gesichert
  und danach zurückgelegt.
- **Selbstnutzung.** Die Trefferqualität kommt aus
  `dev bench search --corpus v1 --profile fast` gegen die Baseline, die
  Latenz aus einem Alltagslauf mit `--latency` über den Dienst. Beides geht
  in beide `benchmarks.md`; die Latenz des Korpusmodus (CLI, 5–12 s je
  Aufruf) nicht. Das ersetzt die Zeile zu 4c-2 unter „Selbstnutzung“ oben.
- **Im selben Pull Request:** `docs/en|de/migration.md`, beide READMEs und
  beide `cli-reference.md`.

## 4d im Einzelnen

- **`loomux convert [datei]`** geht den Eingang jedes Bereichs durch, oder die
  eine Datei. Das Format wird an der Endung erkannt und bei einer `.txt` am
  Anfang (berichtigt beim Planen von 4d, siehe dort). PDF über
  `pdftotext -layout` mit zusammengeschobenem Leerraum; Transkripte werden an
  einer festen Zeichenschwelle zu Absätzen gefügt, ohne ein Wort zu ändern.
  Jede gewandelte Datei bekommt den Herkunftskopf aus vier Zeilen, darunter
  `asr`. Kein Fehler bricht den Lauf ab; am Ende steht die Liste dessen, was
  von Hand bleibt. Ein zweiter Lauf ändert nichts.
- **`loomux fetch <url> [--scope knowledge]`** holt die Untertitel eines
  Videos über `yt-dlp` in den Eingang.
- **Rollen aus 4c:** `describe` (Beschreibung der Quelle) und `place`
  (Zielbereich einer Datei aus dem Eingang), mit den Rückfällen der Referenz.
- **Die Richter** (seit dem Planen von 4c hier, siehe dort): `is_german`,
  `chopped_words`, `is_one_sentence`, `word_count`. `chopped_words` braucht
  deutsche Zipf-Frequenzen aus `wordfreq`; Größe und Lizenz der
  eingebetteten Tabelle sind beim Planen von 4d entschieden (siehe dort).
- **Modul Brain:** Bei `brain = false` verweigern beide.
- **Externe Programme:** Fehlt `pdftotext` oder `yt-dlp`, nennt die Meldung
  Programm und Installationsbefehl, Exit ungleich 0. In den Tests vertreten
  Attrappen nach dem Muster von `internal/dev/faketool` die Programme.
- Beide stehen auf oberster Ebene wie `reindex` und `reconcile`: Sie schreiben
  in einen Bereich. Unter `brain` stehen die lesenden Befehle.

### Abweichungen beim Planen von 4d

Beim Planen (2026-09-26) gegen `master` (v3.3.0, 4c-1 gemergt) und gegen die
Referenz am Tag `loomux-3-source` gelesen: `src/brain/convert/`,
`src/brain/model/local.py` und `judge.py`, `cli.py:549-560` und
`1519-1559`. Beide Verzeichnisse sind seit dem Tag unverändert. Der Text oben
gilt, wo er nicht widerspricht.

**Korrekturen, die Code und Referenz erzwingen:**

- **Die Referenz ruft `pdftotext` nicht.** Sie liest PDFs im eigenen Prozess
  mit `pypdf` (`extraction_mode="layout"`, `pdf.py:43`), Seite für Seite, und
  die Scan-Schwelle von 100 Zeichen je Seite ist an dieser Ausgabe gemessen.
  Eine Attrappe für `pdftotext` bildet die Referenz darum nicht ab; siehe die
  Entscheidung „PDF“.
- **Das `pdftotext` auf dem Pfad von Git Bash ist xpdf 4.06**
  (`/mingw64/bin`), nicht Poppler, das `init` nennt
  (`internal/setup/tools.go`). Beide kennen `-layout` und `-enc`; welche
  Kodierung xpdf ohne `-enc` schreibt, ist ungemessen. loomux ruft darum
  immer `pdftotext -layout -enc UTF-8 -eol unix <pdf> -` und trennt die Seiten am
  Seitenvorschub `\f`. Zwei Builds auf einem Rechner brächen den zweiten
  Lauf; siehe Vorschlag 12.
- **Erkannt wird zuerst an der Endung** (`detect.py:38-53`): `.pdf` ist PDF,
  ohne Blick in die Datei; alles außer `.txt` ist nicht wandelbar; eine `.txt`
  wird an ihren ersten 8192 Zeichen als Transkript mit Klammermarke
  (`[mm:ss]`, `[hh:mm:ss]`) oder mit Bereichszeile (`hh:mm:ss - hh:mm:ss`)
  erkannt, sonst ist sie nicht wandelbar. Der Satz „am Anfang, nicht an der
  Endung“ galt nur den Transkripten und ist oben berichtigt.
- **`fetch` benutzt `yt-dlp` als Bibliothek** und holt die json3-Spur selbst
  über `urllib` (`fetch.py:92-125`): erst die manuellen Spuren, dann die
  automatischen, je `de` vor `en`, nur `ext == "json3"`. Eine leere Spur
  scheitert mit „no subtitle track to fetch, and this system does no ASR“.
  Der Dateiname kommt aus dem Titel, ohne die Zeichen, die Windows nicht
  duldet, und ohne Steuerzeichen, höchstens 150 Zeichen, sonst `video`; bei
  einer YouTube-Adresse folgt die elfstellige Id in Klammern. Die Marken
  stehen als `[hh:mm:ss]`. Siehe die Entscheidung „fetch“.
- **`fetch` prüft `readonly` nicht** (`cli.py:1548-1559`), während `convert`
  einen `readonly`-Bereich überspringt (`run.py:210-222`). Siehe Vorschlag 2.
- **Kein Befehl, der in einen Bereich schreibt, fragt heute `[modules]`.**
  `reindex` und `reconcile` laufen unabhängig davon; die Tabelle lesen nur
  `loomux mcp`, die Hooks (`config.ReadModules`) und die Prüfkette für die
  Lane `lint/wiki`. Siehe Vorschlag 3.
- **`Client.Ask` aus 4c-1 kann kein Ausgabeschema senden.** `place` braucht
  eines: Die Referenz setzt `format` auf `{scope, grund}` (`local.py:42-46`,
  `client.py:157`).
- **Die Richter lassen sich nicht wörtlich übertragen.** In RE2 sind `\w` und
  `\b` ASCII, in Python Unicode; `\b[\wÄÖÜäöüß]+(?:-…)+\b` fände in Go ein
  Wort mit Umlaut am Rand nicht. wordfreq faltet die Schreibung (`Straße`
  wird `strasse`) und behandelt Ziffern eigens; `strings.ToLower` tut beides
  nicht, `pytext.CaseFold` und `pytext.NFC` gibt es. `_reads_back` prüft mit
  PyYAML (YAML 1.1), loomux hat `yaml.v3`.
- **Die Rollen im Einzelnen** (`local.py`, `run.py`). `describe` bekommt die
  ersten 1800 Zeichen des gewandelten Textes und besteht fünf Regeln: ein
  Satz, höchstens 22 Wörter, deutsch, keine zerhackten Wörter, und der Satz
  liest sich aus dem Kopf unverändert zurück. Ein Kopf ohne Satz wird bei
  jedem Lauf neu gefragt, ein stehender Satz nie. `place` bekommt dieselben
  1800 Zeichen und die Bereiche, die nicht `readonly` sind und nicht offener
  als der Eingang (`local_only` vor `manual_cloud` vor `automatic_cloud`; ein
  Bereich ohne Deklaration zählt als `manual_cloud`). Gefragt wird nur für
  eine Datei, die dieser Lauf geschrieben hat; das Ergebnis ist eine Zeile
  `suggested:`, verschoben wird nichts.

**Mit dem Nutzer entschieden (2026-09-26):**

| Frage | Entscheidung |
|---|---|
| PDF | `pdftotext` (Poppler), wie die Fusions-Spec. Die Naht liegt bei den Seiten: Für die Test-PDFs der Referenz zeichnet ein Orakel den Seitentext auf, den `pypdf` dort liefert, und die Attrappe nach dem Muster von `internal/dev/faketool` gibt ihn mit `\f` getrennt aus. Waschen, Scan-Schwelle, Kopf, zweiter Lauf und Meldungen werden so an der Referenz gemessen; die Zeile `converter:` wird dabei als freigegebene Abweichung angeglichen (Vorschlag 8). Freigegebene Abweichungen außerdem: der Extraktor selbst und die Meldung für eine PDF, die sich nicht lesen lässt. Diese Fälle sind die Hälfte der Parität. Die andere Hälfte ist das echte Werkzeug: Je Test-PDF zeichnet ein Mensch einmal auf, was Poppler mit den Schaltern von loomux ausgibt, samt Exit-Code und dem `\f` nach der letzten Seite, und ein Go-Golden spielt es über die Attrappe ab, wie die Fusions-Spec es für externe Programme verlangt („Externe Programme“). Die Test-PDFs haben eine 5000 pt breite Seite statt A4 der Referenz: `pdftotext` schneidet Text am Seitenrand ab, auf A4 blieben 96 Zeichen der Zeile, und jede Textseite wäre unter der Schwelle ein Scan gewesen; für pypdf ändert die Breite nichts (`parity/stufe-4d.md`, „Die Test-PDFs“). Die Schwelle 100 ist an echtem Poppler 25.07.0 nachgemessen: Scan-Seiten 0 Zeichen, Textseiten mindestens 142 |
| Zipf-Tabelle | Eingebettet, mit eigener Lizenz. Die Tabelle trägt, was die zwei Signale von `chopped_words` fragen: jedes Wort mit Zipf ≥ 2,5, gemessen 85.034 Wörter, davon 39.858 ab 3,0, rund 285 KB gzip (wordfreq 3.1.1, `large_de`), dazu die rohe Frequenz der Schlüssel mit Ziffern. Gebaut am 2026-09-27 nach E3 des Plans `plans/2026-09-26-loomux-stufe-4d.md`: `de.txt.gz` hat 288.299 Bytes (entpackt 894.582), `[common]` 39.728 und `[mid]` 45.086 Schlüssel ohne Ziffernfolge, `[digits]` 1.359 Schlüssel mit Ziffernfolge (`\d[\d.,]+`) — von den 85.034 Wörtern ab 2,5 tragen 220 eine Ziffernfolge und stehen dort, und von den 3.297 Schlüsseln mit irgendeiner Ziffer werden die 1.938 ohne Ziffernfolge (`g4`, `h2o`) wie jedes Wort nach ihrer Zipf-Zahl einsortiert. Die zuerst gemessene Beschränkung auf Wörter bis fünf Zeichen (46.505 Wörter, 135 KB) trägt nicht: Der Richter zählt die Länge eines Teils, wie er geschrieben steht, die Tabelle ihren gefalteten Schlüssel, und `Grüße` (fünf Zeichen) ist dort `grüsse` (sechs) — berichtigt beim Planen, 2026-09-26. Sie liegt in einem eigenen Verzeichnis mit Hinweis auf CC BY-SA 4.0 und der Quellenangabe von wordfreq, getrennt von der Lizenz von loomux wie der Korpus von 4c-2. Ein eingechecktes PEP-723-Skript mit `wordfreq==3.1.1` erzeugt sie über `uv run --script`, ein Test hält ihre Prüfsumme fest, entpackt wird beim ersten Gebrauch. Wie der Hinweis den Empfänger eines Releases erreicht, steht in Vorschlag 13 |
| fetch | `yt-dlp` schreibt die Untertitel selbst, loomux spricht nie ins Netz: ein Aufruf mit `--ignore-config --no-playlist --no-progress --skip-download --write-subs --write-auto-subs --sub-langs de,en --sub-format json3 --write-info-json --ignore-errors -o v` in ein Wegwerfverzeichnis (beim Planen ergänzt: `--ignore-config`, damit eine Nutzerkonfiguration weder Namen noch Ort ändert, `--no-playlist`, damit eine Adresse mit `&list=` nur das Video holt, das feste `-o v`, siehe „Offen und vor dem Bau zu messen“). `--ignore-errors`, weil eine scheiternde Spur yt-dlp sonst beendet, bevor es die Info-JSON schreibt: gemessen am 2026-09-26 an `mHSOsy_usAg`, die automatisch übersetzte Spur `en` antwortete mit 429, yt-dlp endete mit 1 und hinterließ nur `v.de.json3`; mit dem Schalter wird der Fehler eine Warnung, Exit 0, die Info-JSON steht da (`parity/stufe-4d.md`). loomux liest aus der Info-JSON, welche Sprache eine manuelle Spur hat, und wählt unter den geschriebenen Dateien in der Reihenfolge der Referenz. Abweichungen: der Randfall einer manuellen Spur ohne json3, die YouTube heute nicht anbietet, und die Spur innerhalb einer Sprache (Zeile „fetch: Spur einer Sprache“) |
| fetch: Spur einer Sprache | YouTube kann unter `automatic_captions.<lang>` mehrere json3-Spuren nennen; für `mHSOsy_usAg` unter `de` zuerst eine Übersetzung (`tlang=de` aus der Spracherkennung `en-US`), dann die deutsche Spracherkennung selbst. Die Befehlszeile von yt-dlp nimmt die letzte passende (`YoutubeDL.process_subtitles`, `matches[-1]`, yt-dlp 2026.08.19), die Referenz nahm die erste. loomux behält die Wahl von yt-dlp, für dieses Video die Spracherkennung in der Originalsprache. Freigegebene Abweichung |
| Selbstnutzung | Am echten Eingang `brain-knowledge/00 Eingang` (Bereich `knowledge`, `manual_cloud`): Der Mensch legt eine PDF und ein Transkript hinein und führt `loomux convert` mit installiertem Poppler aus |
| `NOTICE.md` | Trägt jedes fremde Stück im Binary, nicht nur die Zipf-Tabelle: die Standardbibliothek von Go, jedes gelinkte Modul (am 2026-09-26 vierzehn), jede Grammatik von tree-sitter, deren Paket loomux importiert, und den Hinweis der Zipf-Tabelle. `loomux dev notices` erzeugt die Datei, ein Test hält sie aktuell. Gemessen am 2026-09-26: `gotreesitter/grammars/grammar_blobs` bettet 206 Grammatiken ein, nach dem Linken steht nur `python` (MIT) im Binary; eine copyleft-Grammatik bricht den Erzeuger ab, weil ihre Bedingungen das ganze Binary bänden. Poppler 25.07.0 ist installiert (winget, `%LOCALAPPDATA%\Microsoft\WinGet\Packages\oschwartz10612.Poppler_…\poppler-25.07.0\Library\bin`) |

**Vorgeschlagen und mit der Spec freigegeben (2026-09-26):**

| # | Frage | Vorschlag |
|---|---|---|
| 1 | Formaterkennung | Wie die Referenz, siehe „Korrekturen“ |
| 2 | `fetch` in einen `readonly`-Bereich | Verweigert, mit Meldung und Exit ungleich 0. Die geerbte Lücke wird geheilt und eine freigegebene Abweichung |
| 3 | `[modules].brain` | Maßgeblich ist das Projekt, das die Suche nach oben vom Arbeitsverzeichnis findet, wie bei `loomux mcp`; außerhalb eines Projekts laufen beide. Die Meldung nennt `[modules] brain` und die Datei |
| 4 | Wächter | Der Wächter verweigert einem Agenten `convert` und `fetch`; erlaubt bleibt ein alleinstehendes `--help` oder `-h`. Gemessen am 2026-09-26: Ein Write eines Agenten in loomux nach `brain-knowledge/00 Eingang/x.md` verweigert die Schreibschranke mit Exit 2, denn der Eingang liegt in keinem beschreibbaren Baum. Ein erlaubtes `convert` schriebe dorthin, was die Schranke dem Agenten verbietet, und `fetch` legte, was der generische Extraktor von `yt-dlp` unter einer beliebigen Adresse findet, in den Tresor. Kein Skill ruft einen der beiden, die Regel kostet also heute nichts. Sie steht neben `area add` in der Wächterregel |
| 5 | Ausgabeschema | `Client` bekommt ein optionales `format`, gesendet nur, wenn gesetzt; `propose` bleibt ohne |
| 6 | Funktionswortliste | Die 72 Wörter liegen einmal in `internal/brain/model`. Ob 4c-2 oder 4d zuerst gemergt wird: diese Stufe legt sie dort an, die andere benutzt sie. **Berichtigt 2026-09-27:** 4c-2 ist zuerst gemergt und braucht die Liste nicht; die Korpusprüfung erkennt die Richtung einer Frage an eigenen Listen (14 englische, 17 deutsche Wörter, `internal/dev/benchsearch/corpus.go`, siehe „Abweichungen beim Planen von 4c-2“). Die 72 Wörter stehen nur in `internal/brain/model/judge.go` und nur `describe` benutzt sie |
| 7 | `convert <datei>` | Ohne `describe` und `place`, wie in der Referenz: Die Datei gehört zu keinem Bereich, und der nächste Durchgang über die Eingänge füllt den Satz nach |
| 8 | Kennung des PDF-Wandlers | `converter: brain-pdf/2` statt `/1`. Die Referenz zählt hoch, wenn sich die Ausgabe ändert (`header.py:11-17`), und `pdftotext` liefert einen anderen Text als `pypdf`. `brain-transcript/1` bleibt, denn Transkripte werden gleich gewandelt. In den Fällen über die Naht wird die Zeile angeglichen, siehe „PDF“ |
| 9 | Fehlendes Programm | Fehlt `pdftotext` oder ist es kein Poppler (Vorschlag 12), wird jede PDF des Laufs eine Zeile `skipped:` mit dem gefundenen Programm und dem Installationsbefehl aus `internal/programs` (beim Planen aus `internal/setup/tools.go` dorthin verlegt, damit `init` und `convert` eine Liste lesen), und der Lauf endet mit 1; ein Eingang ohne PDF braucht das Programm nicht. Fehlt `yt-dlp`, endet `fetch` mit derselben Meldung und Exit ungleich 0 |
| 10 | Parität | Aufgezeichnet vom Tag `loomux-3-source`: `convert` mit Transkripten, nicht wandelbaren Dateien, Zieldateien von Hand, zweitem Lauf und `readonly`-Bereich; PDFs über die Naht aus „PDF“; `describe` und `place` über `loomux dev fake-ollama` aus 4c-1. `fetch` nur als Go-Golden gegen eine `yt-dlp`-Attrappe, weil die Referenz dafür das Netz braucht, dazu die Vektoren aus `tests/convert/test_fetch.py`. Die Richter an den Fällen aus `tests/model/test_judge.py` und an einer Wortliste mit Umlauten, ß, Großbuchstaben, Ziffern und Unterstrichen, deren Urteile die Referenz einmal aufzeichnet; `_reads_back` an einer Reihe von Randsätzen gegen PyYAML |
| 11 | Vor dem Bau zu messen | Erste Aufgabe des Plans, siehe „Offen und vor dem Bau zu messen“ |
| 12 | Nur Poppler | loomux liest einmal je Lauf, sobald eine PDF ansteht, `pdftotext -v` und nimmt nur Poppler an; xpdf wird wie ein fehlendes Programm behandelt (Vorschlag 9). Grund: Git Bash findet xpdf 4.06, PowerShell nach der Installation Poppler; beide schrieben unter derselben Kennung `brain-pdf/2` einen anderen Text, und jeder Lauf aus der anderen Shell schriebe jede PDF-Zieldatei neu, gegen „ein zweiter Lauf ändert nichts“ |
| 13 | Weg des Lizenzhinweises | Folge der Entscheidung „Zipf-Tabelle“: Die Tabelle geht mit jedem Release hinaus, anders als der Korpus von 4c-2, der in `testdata/` bleibt. Der Hinweis erreicht den Empfänger darum als eigene Datei des Releases: `release.Build` legt `NOTICE.md` neben die Binaries und in `SHA256SUMS`. Für die Tabelle steht darin CC BY-SA 4.0, die Quellenangabe von wordfreq und dessen ganzer Abschnitt über die Datenquellen (Google Books Ngrams, Leeds Internet Corpus, die Bedingungen der Twitter-Daten und die übrigen), wörtlich aus wordfreq 3.1.1; derselbe Hinweis liegt im Verzeichnis der Tabelle. Mit dem Nutzer erweitert (2026-09-26) auf alle fremden Teile, siehe `NOTICE.md` oben. Das ist ein Weg, kein rechtliches Urteil |

## 4e: Umstellung der Wirte

Eine Checkliste, abgehakt vom Menschen, festgehalten in
`parity/stufe-4e.md`:

1. `brain-knowledge` hat einen Remote und ist committet.
2. **Maschinenzustand:** `%LOCALAPPDATA%\brain` gegen `%LOCALAPPDATA%\loomux`
   vergleichen — `registry.toml`, die Identitätsregister unter `areas/`,
   `maintenance/`. Was fehlt, von Hand kopieren. Danach entfallen
   `LegacyBrainDirUntilStage3`, `ReadAreaManifestUntilStage4` samt den alten
   Manifestnamen und `Manifest.Lanes`, in einem eigenen Pull Request. Das alte
   Verzeichnis bleibt als Sicherung bis zum Ende der Folgeprojekte.
3. **Je Wirt** (`space`, `iam_backend`, `ecoflow`, `brain-knowledge`):
   `loomux init`; alte Einträge von Hand entfernen (Marke `ultraLoomOwned`,
   Gruppe `ultraloom-wiki-guard`, `brain guard`, `ulguard`), dann
   `.ultraloom/`, `.brain.toml` und `.ultra-brain/`.
4. **Rauchtest je Wirt:** ein erlaubter Edit, ein verweigerter Edit, ein Commit
   durch commit-msg und pre-commit, eine Suche über MCP.

## Parität

| Teilstufe | Referenz | Fälle |
|---|---|---|
| 4a-1 | keine (neu) | Golden-Tests für `config list`, `get`, `set` und das Tastenprotokoll der Oberfläche; Fälle gegen die Wächterregel |
| 4a-2 | ulinit-Go (Tests ziehen mit); `ultra-brain` für post-merge | Die umgezogenen Tests; für `merge-hook` aufgezeichnete Fälle gegen `brain hook install|status|remove` über `gitworld` |
| 4c | `ultra-brain` (Python) | Aufgezeichnete Fälle für `reconcile` mit Vorschlag gegen eine Ollama-Attrappe über HTTP; `dev bench search` gegen `brain bench` auf demselben Korpus. #1 und #4 als freigegebene Abweichungen |
| 4d | `ultra-brain` (Python) | Aufgezeichnete Fälle für `convert`, PDFs über die Naht bei den Seiten; `fetch` als Go-Golden gegen eine `yt-dlp`-Attrappe (siehe „Abweichungen beim Planen von 4d“) |

Aufzeichnungsgrundlage ist der Tag `loomux-3-source` auf `ultra-brain`; setzt
das Repo sich bis dahin fort, setzt ein Mensch einen neuen Tag.

## Fehlerverhalten

- Eine `config.toml`, die ein Lader ablehnt, wird von `config` und `init` nicht
  angefasst; die Meldung des Laders mit Datei und Schlüssel geht auf stderr.
- Der Editor schreibt nur über `lock.ReplaceText`; ein Abbruch lässt die alte
  Datei stehen.
- Eine verweigerte Bestätigung ist kein Fehler: Exit 0, und die Meldung sagt,
  was nicht geschrieben wurde.
- `init` schreibt in der Reihenfolge Zustand zuletzt: Erst wenn alle Dateien
  geschrieben sind, kommt `installed.toml`. Ein Abbruch in der Mitte wird beim
  nächsten Lauf als offen erkannt.

## Selbstnutzung

- **4a-1:** `loomux config list` zeigt die echte Konfiguration von loomux mit
  Herkunft. Die Brücke der Nutzerregistrierung filtert nach dem `[modules]`
  von loomux, sobald dort eines steht; bis dahin prüft ein Fall die Filterung.
- **4a-2:** `go run ./cmd/loomux init --yes` richtet einen frischen Klon von
  loomux ein: `core.hooksPath` auf `.githooks/`, das Binary nach
  `bin/loomux.exe`, wohin die eingecheckten Einträge zeigen. Die zwei
  Handbefehle in AGENTS.md und README werden dadurch ersetzt. Auf einem
  eingerichteten Checkout zeigt `init --dry-run` keine Änderung an
  `.claude/settings.json`, `.githooks/` und `.loomux/config.toml`.
- **4c:** Ein `local_only`-Bereich dieses Rechners öffnet einen Fall mit
  Vorschlag; `dev bench search` läuft über den Korpus.
- **4d:** Eine PDF und ein Transkript gehen durch den Eingang eines Bereichs:
  `brain-knowledge/00 Eingang`, vom Menschen, mit Poppler (entschieden
  2026-09-26).

## Messen

- `loomux hook pre-tool-use` mit `[modules]` gegen ohne: Das Lesen der Tabelle
  darf die warme Zeit nicht messbar heben (heute 7,5 ms).
- `loomux config list` warm.
- `init --dry-run` auf loomux.
- `reconcile` mit Modell gegen ohne, je Fall.

Eingetragen in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`.

## Fertig, wenn

Je Teilstufe die fünf Bedingungen der Fusions-Spec: Fälle grün oder
freigegeben, 100 % Coverage je Funktion, Mutationsrunde mit dokumentierten
Überlebenden, Messungen eingetragen, Selbstnutzung wie oben.

## Nachzutragen in der Fusions-Spec

- Stufe 4 zerfällt in 4a-1, 4a-2, 4c, 4d und den Abschlussschritt 4e; eine
  Zeile je Teilstufe unter „Stufen“ und „Reihenfolge der offenen Stufen“.
- „Datenumzug“ Punkt 2 und 4 und „Umstellung der Wirte“: `migrate` fällt weg;
  die Punkte verweisen auf die Checkliste von 4e.
- Nachträge #5, #7, #8, #10, #11, #13, #16 freigegeben 2026-09-23; #12
  freigegeben als Wegfall; #15 (kanonischer Ort) und #9 (Wegfall aus `init`)
  freigegeben 2026-09-24.
- Ein Nachtrag #19: `loomux migrate` fällt weg.
- „Git-Hooks“ bekommt die Zeile post-merge.
- „Konfiguration und Zustand“ bekommt `[modules]` und `[model]`.

## Offene Entscheidungen

1. ~~`[modules].hooks = false` und die globale Schreibschranke~~ —
   entschieden 2026-09-24: der Wächter bleibt immer an (siehe
   „Entscheidungen“).
2. ~~Selbstnutzung von `init` auf loomux~~ — entschieden 2026-09-24:
   `init` erkennt vorhandene loomux-Einträge als eigene (siehe
   „Entscheidungen“).
3. ~~post-merge~~ — entschieden 2026-09-24: ein Weg, der Hook ruft
   `loomux merge-hook record` (siehe 4a-2, „post-merge“).
4. ~~Globales `[model]`~~ — entschieden 2026-09-24: `loomux config --global`
   (siehe `loomux config`).
5. ~~Welches Binary `init` an den kanonischen Ort legt~~ — entschieden
   2026-09-24: `init` holt immer das neueste Release über `selfupdate`, auch
   wenn es selbst ein Release ist; es kopiert nie sich selbst dorthin. Fehlt
   `gh` oder scheitert der Abruf, fällt der Schritt aus, und `init` nennt
   `gh auth login` und `loomux upgrade`. Ein Checkout wie loomux, dessen
   Einträge `bin/loomux.exe` rufen, bekommt sein Binary weiter aus dem Build.
6. ~~Hook-Einträge und `PATH`~~ — entschieden 2026-09-24: fester Pfad über
   `${LOCALAPPDATA}` (siehe „Host-Einträge“).

## Offen und vor dem Bau zu messen

- Ob die VT-Verarbeitung der Windows-Konsole in Windows Terminal, conhost und
  dem Terminal der Claude-App gleich wirkt (4a-1). Die Oberfläche ist ohne diese
  Prüfung gebaut; sie bleibt ein Schritt des Menschen (`parity/stufe-4a-1.md`:
  Pfeiltasten, `/`, ESC allein, Enter; Stand 2026-09-27).
- ~~Wie Antigravity Skills eines Projekts findet (4a-2).~~ Gemessen am
  2026-09-24 mit agy 1.2.8: unter `.agents/skills/<name>/SKILL.md`
  (`parity/stufe-4a-2.md`, „Antigravity-Einträge“).
- ~~Ob Antigravity `${LOCALAPPDATA}` im Hook-Befehl auflöst wie Claude Code
  (4a-2).~~ Gemessen am 2026-09-24 mit agy 1.2.8: nein, agy führt Hooks über
  `cmd.exe` aus; `init` schreibt `%LOCALAPPDATA%/loomux/bin/loomux.exe` ohne
  Anführungszeichen und nur, wenn das installierte Binary dort liegt;
  Durchstich am 2026-09-25 (`parity/stufe-4a-2.md`, „Antigravity-Einträge“).
- Welches Arbeitsverzeichnis ein stdio-MCP-Server aus dem Nutzerbereich
  bekommt (erste Aufgabe des Plans von 4a-1).
- Ob `qmd` nach geänderten Ignore-Mustern die Vektoren wiederverwendet (4e).
- ~~Was `yt-dlp` mit den Schaltern der Entscheidung „fetch“ schreibt (4d,
  erste Aufgabe des Plans, über `uvx yt-dlp` an einem echten Video), mit
  festem `-o v` in einem kurzen Wegwerfverzeichnis (MAX_PATH) — nicht
  `-o %(id)s`, denn die Id eines fremden Extraktors ist beliebig lang:
  Dateinamen, Form der Info-JSON, was ohne json3-Spur geschieht (`--sub-format`
  fällt auf ein anderes Format zurück, darum prüft loomux die Endung `.json3`,
  bevor es liest), und Exit-Code samt übrig gebliebenen Dateien, wenn eine
  der beiden Spuren scheitert (automatisch übersetzte Spuren antworten oft
  mit 429). Gewählt wird nach Info-JSON und Dateien, nicht nach dem
  Exit-Code.~~ Gemessen am 2026-09-26 mit yt-dlp 2026.08.19
  (`parity/stufe-4d.md`, „`yt-dlp` 2026.08.19“): Ohne
  `--ignore-errors` beendet die scheiternde Spur — hier 429 auf der
  automatisch übersetzten Spur `en` — yt-dlp mit Exit 1, bevor es die
  Info-JSON schreibt; mit dem Schalter wird der Fehler eine Warnung, Exit 0,
  die Info-JSON steht da. Die Entscheidung „fetch“ trägt den Schalter darum.
- ~~Was `pdftotext -v` ausgibt, an xpdf 4.06 und an Poppler (4d, ebenda), als
  Grundlage von Vorschlag 12. Poppler misst der Mensch, sobald es installiert
  ist, und zeichnet dabei die Goldens der Test-PDFs auf (Entscheidung „PDF“):
  Seitentrenner, Exit-Codes einer kaputten und einer verschlüsselten PDF, ein
  Pfad mit Umlaut.~~ Gemessen am 2026-09-26: xpdf 4.06 endet `-v` mit 99 und
  schreibt auf stdout, Poppler 25.07.0 mit 0 auf stderr; loomux liest `-v`
  darum gleich welcher Exit-Code. Der Mensch hat die Goldens gegen Poppler
  25.07.0 aufgezeichnet (`testdata/convert/poppler/faketool.json`), samt
  Seitentrennern und den Exit-Codes 1 für die kaputte und die verschlüsselte
  PDF (`parity/stufe-4d.md`, „`pdftotext`: xpdf 4.06 und Poppler 25.07.0“ und
  „Messungen: Poppler, die Aufnahme des Menschen“).
