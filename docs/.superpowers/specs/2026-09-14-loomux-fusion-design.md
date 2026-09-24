# loomux — ultraloom und ultra-brain in einem Go-Binary

**Datum:** 2026-09-14
**Stand:** teilweise umgesetzt (2026-09-23).
**Fusions-Stufen:** 1a, 1b-1, 1b-2 (`serve`, MCP, Brücke), 1b-3 (Wiki- und
Doku-Umzug) und 2a (Prüfkette `[verify]`, `loomux check <profil>`,
`check gocover`), 2b (commit-msg mit `[commit]`, `--calibrate`, `--language`) und
2c (Stop-Tor, `subagent-start`/`-stop`, Antigravity-Nachmessung und Host-Adapter,
am 2026-09-22) sind abgeschlossen. Der Eintrag der drei Hooks in
`.claude/settings.json` ist erfolgt; loomux prüft sich an jedem Rundenende selbst.
Stufe 3 ist in 3a, 3b und 3c zerfallen und lief parallel zu 2b und 2c; 3a
(`reindex`, `embed`, `reconcile`, `area add`) ist fertig (2026-09-22), samt
Selbstnutzung über die echte Registry. 3b (`cases`, `case`, `approve`) ist
fertig (2026-09-23), samt Selbstnutzung über die echte Registry. 3c (`brain check`,
`lint --scope`, `wiki init|types|retype`, die Aufholung in `serve`) ist fertig
(2026-09-23), die lesenden Befehle samt Selbstnutzung. Stufe 4 ist offen. Siebzehn
Stellen der Quellrepos, die bis 2026-09-19 keine Stufe hatten, stehen unter
„Stufen“ im Abschnitt „Nachgetragen“; #1, #2, #3 und #17 sind freigegeben.
**Säule 3 (Code-Graph), vorgezogen und parallel gebaut:** G1 (Modell, PageRank,
Blast) am 2026-09-17, G2a (Extraktor, Auflösung, Speicher, Frische, die
Befehle `graph build` und `graph check`) und G2b (die Abfrage) am 2026-09-18,
G3 (die Abfrage über MCP) am 2026-09-19 abgeschlossen; G4 ff. offen. Die Vorziehung war Absicht: beide Stufen
ziehen keine Abhängigkeit ein, und die Messung, die die alte Reihenfolge
begründete, gehört zu G3
(`2026-09-14-loomux-code-graph-design.md` §10, `2026-09-16-loomux-code-g1-delta.md` §1)
**Ort:** vorläufig im `ultraloom`-Worktree `claude/ultra-loom-brain-fusion-a5bb17`,
weil das Zielrepo `xidus90/loomux` noch nicht existiert. Zieht mit Stufe 1a um.
**Löst ab:** ulflow M4–M6
(`docs/.superpowers/specs/2026-09-11-ulflow-laufzeit-design.md`, liegt nur auf
`feature/agent-harness`),
[Go-Hooks für drei Hosts](2026-09-10-go-hooks-drei-hosts-design.md), Stufen 2–5,
[Wiki-Flottenstandard](2026-09-10-wiki-flottenstandard-design.md), Stufen 3–5.
**Messgrundlage:** `docs/benchmarks.md`, Eintrag 2026-09-14 16:05.

## Ziel

`ultraloom` und `ultra-brain` werden ein Produkt: **loomux**, ein Go-Binary in
einem neuen Repo. Kein Python bleibt im Produkt — weder als Laufzeit noch als
Hook noch als Werkzeugskript. Nichts wird ersatzlos gestrichen: Was heute eine
der beiden Seiten kann, kann loomux am Ende auch, oder die Abweichung steht mit
Begründung und Freigabe in einer Liste.

Das Produkt gliedert sich in **Hooks · Skills · Graph + Loop Engineering ·
Second Brain / LLM Wiki · LLM OS**. Diese Spec deckt Hooks, Prüfkette, Second
Brain, den MCP-Dienst und den Installer ab. Flow-Laufzeit, Web-Oberfläche und
grafischer Editor sind Folgeprojekte (siehe unten).

„Python raus" meint den Code von loomux, nicht die Projekte, die es prüft.
Python-Wirte wie `iam_backend` behalten ihre ruff-, dmypy- und pyright-Lanes.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Form | Ein Repo, ein Go-Modul, ein Binary `loomux` |
| Name | `loomux` — Repo `xidus90/loomux`, Konfig `.loomux/`, Zustand `%LOCALAPPDATA%\loomux\` |
| Lizenz | PolyForm Noncommercial 1.0. Der Wechsel weg von AGPL-3.0 (ultraloom) ist möglich, weil beide Repos laut `git shortlog` genau einen menschlichen Urheber haben; die 733 ultra-brain-Commits unter `Brain <brain@example.invalid>` sind eine fehlkonfigurierte Test-Identität desselben Nutzers. Bereits veröffentlichte Stände bleiben AGPL |
| Historie | Neues Repo ohne Historie; die alten Repos werden archiviert |
| Vorhandener Go-Code | Wird **mit seinen Tests umgezogen, nicht neu geschrieben**. Neu geschrieben werden das Python und `ultra-brain/pkg/mcp` (1.595 Zeilen, handgerollt) |
| Übergang | Harter Schnitt, neues Konfigformat, keine Rückwärtskompatibilität zur Laufzeit |
| Pilot | Das loomux-Repo selbst, ab Stufe 1a |
| Übrige Wirte | Stellen nach Stufe 4 um |
| Projektkonfiguration | Eine handgepflegte `.loomux/config.toml` + maschinengeschriebenes `.loomux/state/` |
| Langlebiges | Ein Dienst `loomux serve` über localhost-HTTP, qmd dahinter |
| Parität | Verhalten der alten Formen aufzeichnen, dann portieren |
| Wikis der beiden Repos | Nach `loomux/docs/wiki`, ein Bereich `project/loomux`, jede Seite vorher geprüft |
| Flow, Web, Editor | Folgeprojekte; ulflow M1 ist die Flow-Basis |
| Laufende Vorhaben | Begonnene Stufen werden abgeschlossen (Wiki-Flotte Stufe 2, laufende ulflow-Welle), danach Stillstand in den alten Repos bis auf Fehlerbehebungen |
| Maschinenweites Binary (Nachtrag 2026-09-23) | Brücke und `serve` laufen aus `%LOCALAPPDATA%\loomux\bin\loomux.exe`, das aus einem Release stammt, nie aus einem Checkout; `serve` aktualisiert es täglich über `gh`. Siehe `2026-09-23-self-update-design.md` |

## Ausgangslage, vermessen am 2026-09-14

### Umfang

| Repo | Python-Code | Python-Tests | Go-Code | Go-Tests |
|---|---:|---:|---:|---:|
| `ultraloom` (`master` `9d01a60`) | 7.285 | 12.766 | 7.379 | 11.107 |
| `ultra-brain` (`master` `3cc72d2`) | 15.928 | 28.429 (+726 `tools/`) | 16.222 | 27.107 |

`feature/agent-harness` trägt zusätzlich ulflow M1: 61 Commits, 142 Dateien,
+23.213 Zeilen, nicht gemergt, nicht gepusht.

### Messungen (warm, Median, `docs/benchmarks.md`)

| Fall | Zeit |
|---|---:|
| `ulguard --root` (PreToolUse, Edit) | 33 ms |
| `brain guard` | 72 ms |
| beide Wächter parallel — so wie ein Edit sie sieht | 73 ms |
| beide Wächter sequenziell — die CPU | 114 ms |
| Go-Startboden (`brain version`, `ulinit --version`) | 32–34 ms |
| `ultraloom hook subagent-start` (Python, mit und ohne `uv run`) | 706–730 ms |
| `ulguard hook session-start` (Go) | 67 ms |
| `brain-mcp --help` (nur Python-Start) | 898 ms |
| `brain-mcp status` / `brain status` | 3.113 ms / 36 ms |
| `brain-mcp search` / `brain search` | 7.773 ms / 242 ms |

Die Suche ist **kein** Beschleunigungsfaktor: Go und Python lieferten für
dieselbe Anfrage verschiedene erste Treffer. `brain status` in Go ist ein
Stumpf (Arbeitsverzeichnis und Seitenzahl).

Binärgröße bestimmt den Start nicht: `brain.exe` 13,7 MB startet in 32 ms,
`ulinit.exe` 8,5 MB in 34 ms, `ulguard.exe` 6,0 MB in 33 ms.

### Was es in Go noch nicht gibt

Aus der Abdeckungskarte vom 2026-09-14, gegen den Code gelesen:

- **ultraloom:** keine Go-Form für `[verify]` und die Prüfkette, keine
  Prozessbaum-Tötung, keine Hooks `stop`/`subagent-start`/`subagent-stop`.
  `ulinit check commit-msg` ist eine englische Wortliste über die erste Zeile
  (62 Zeilen, `internal/commit/language.go`) und kein Ersatz für
  `commit/*.py` (1.444 Zeilen, `--language`, `--calibrate`, `[commit]`).
  Die 62 Zeilen gelten für ulinit; loomux' Kopie davon hatte 70 und prüfte
  zusätzlich den Conventional-Commits-Header.
  *(Hinweis Stufe 2b: Am 2026-09-19 vollständig durch `internal/verify/commit` abgelöst:
  Variante B, Go-Wortliste, Umlaut-Regel, RE2-`[[commit.allow]]`, `--calibrate` mit GitRunner-Naht).*
  `ulinit check coverage` misst nichts und **exitet 0 ohne `--summary`**.
- **ultra-brain:** Python-only sind Daemon und IPC, `reconcile`, `merge-events`,
  `init`, das Registry-Locking, `convert`/`fetch`, das lokale Modell, `bench`
  und `wiki types`/`retype`/`census`/`scaffold`. `reindex` in Go überspringt den
  `reconcile`-Durchgang. `maintenance/approve` hat eine Go-Form, deren
  Gleichheit nicht nachgewiesen ist.
- **Werkzeuge:** `tools/go_mutants.py` (Mutationstests für Go) und
  `hooks/coverage-check.py` sind selbst Python.

### Befunde, die den Entwurf formen

- `ulguard post-edit` ruft auf Wiki-Dateien `uv run brain lint`
  (`cmd/guard/post_edit.go:430`) — ~70 ms `uv`-Hülle vor einem Go-Binary.
- `brain guard` verweigert Schreibzugriffe **global** aus der Registry,
  unabhängig vom Projekt: in dieser Sitzung auch das eigene Scratchpad der
  Claude-Sitzung unter `%TEMP%\claude\…`.
- Der qmd-Index liegt in `~/.cache/qmd/index.sqlite`, die Collections heißen
  nach dem Scope (`project-ultraloom`), nicht nach dem Zustandsverzeichnis.
- `%LOCALAPPDATA%\brain\areas\` (706 KB) hält Kataloge, Graphen und
  **Identitätsregister** der schreibgeschützten Bereiche `space`, `iam-wiki`,
  `obsidian-ai`.
- **Konfigurierte Befehlsregeln greifen heute nie.** `.ultraloom/policy.toml`
  in `ultraloom`, `space` und `iam_backend` schreibt
  `…git\s+push(?![\w-])` und `…pip\s+install(?![\w-])`. Go-RE2 kennt kein
  Lookahead, und `cmd/guard/guard.go:165,172` verwirft den Kompilierfehler
  (`matched, _ := regexp.MatchString(…)`) — die Regel passt nie, ohne ein Wort.
  `git push` fängt die eingebaute Regel `(^|\s)git\s+push(\s|$)` trotzdem
  ab; `pip install` blockiert niemand. loomux kompiliert jede Regel beim Laden
  und verweigert bei einer ungültigen.
- Kein Wirt nutzt heute Flows: `.ultraloom/flows` ist in `space`, `iam_*`,
  `ecoflow`, `brain-knowledge` und `ultra-brain` leer, und keine
  Projektanweisung ruft `ultraloom run`.
- **Unabhängig von loomux:** `brain-knowledge` hat keinen Remote, der letzte
  Commit ist vom 2026-09-07, und `95 Prüfzentrum/` sowie
  `91 Projekte/ultraloom/` sind unversioniert.

## Verhältnis zu laufenden Vorhaben

| Vorhaben | Was bleibt | Was loomux übernimmt |
|---|---|---|
| ulflow (`feature/agent-harness`) | M1 abgeschlossen, laufende Welle wird fertig. M2 (`claude -p`/`agy -p`) und M3 (`verify_until_green` als Daten-Flow) gehören zum Flow-Folgeprojekt | M4 (Prüfkette, Hooks), M5 (Commit-Sprache), M6 (Schnitt) sind die Stufen 2 und 4 dieser Spec |
| Go-Hooks für drei Hosts | Stufe 1 (`hostio`, `session-start`) ist umgesetzt | Stufen 2–5; Hostmatrix, Exit-1-Disziplin des Stop-Tors, die Reparatur an `stop.py:141-146` und die Messpunkte aus Stufe 0 |
| Wiki-Flottenstandard | Stufe 2 (`internal/agenthooks`) wird fertig | Stufen 3–5 gehen in `loomux init` auf; der Grundsatz „Projektdoku im Projekt, Übergreifendes in `brain-knowledge`" bleibt |

## Architektur

### Repo und Binary

- `xidus90/loomux`, ein Go-Modul, **Go ≥ 1.25** (das offizielle MCP-go-sdk
  v1.8.0 verlangt `go 1.25.0`).
- Genau ein Binary `loomux`; alles sind Unterbefehle.
- **Windows zuerst.** POSIX baut; Worktree-Spiegel (Junctions) und Job Objects
  sind Windows-only, per Build-Tag getrennt und so dokumentiert.

### Pakete

Alle unter `internal/`; es gibt keine öffentliche Go-API.

| Paket | Inhalt | Kommt aus |
|---|---|---|
| `config` | `.loomux/config.toml`, `.loomux/state/`, globaler Zustand, `LOOMUX_STATE_DIR` für Tests | `ultraloom/internal/tomlstr`, `src/ultraloom/config.py`, `ultra-brain/pkg/config` |
| `hosts` | Normalisierte Nutzlast und Antwort; Adapter Claude, Antigravity, Codex-Naht | `ultraloom/internal/hostio` |
| `hooks` | Vereinter Wächter, post-edit, Sitzungshooks | `ulguard`, `brain guard`, `src/ultraloom/hooks` |
| `child` | Kindprozess mit Zeitgrenze, Prozessbaum-Tötung, Pipe-Absaugung | `process.py` |
| `verify` | Eine `[verify]`-Tabelle, Lanes, Profile, Coverage-Tor, commit-msg | `checks.py`, `config.py`, `commit/*`, `hooks/coverage-check.py`, `ultraloom/internal/{verify,coverage,commit}` |
| `brain/index`, `brain/graph`, `brain/reader`, `brain/catalog`, `brain/privacy`, `brain/search`, `brain/guard`, `brain/check`, `brain/wiki`, `brain/maintenance`, `brain/cases` | Die vorhandenen Nahtstellen von ultra-brain | `ultra-brain/pkg/*` |
| `brain/convert`, `brain/model`, `brain/bench` | Neu in Go | `src/brain/{convert,model,bench}` |
| `worktree` | Worktree-Spiegel | `ultraloom/internal/{worktreetopo,junction,mirrorcfg,gitwork}` |
| `serve` | MCP-Dienst, Upkeep, stdio-Brücke | `src/brain/{daemon,ipc,client,mcp}`, `ultra-brain/pkg/mcp` |
| `selfupdate`, `swap` | Update des maschinenweiten Binarys aus dem Release; Tausch eines laufenden Binarys (Nachtrag 2026-09-23, `swap` war `dev/swap`) | neu |
| `install` | `loomux init`, `loomux migrate` | `ultraloom/internal/{answers,interview,render,tooling,write,settings,agenthooks,detect}`, `src/brain/init.py` |
| `journal` | Sitzungs- und Laufzustand, soweit Hooks ihn brauchen | `ultraloom/internal/{journal,sessions}` |

**Entfällt ersatzlos:** `ultraloom/internal/vendoring` — es pinnt eine
Python-ultraloom-Laufzeit ins Projekt. `internal/brainpath` wird gegenstandslos,
weil brain kein fremdes Programm mehr ist.

### Abhängigkeitsregeln

- Jedes Paket darf `config` benutzen.
- `hooks` → `verify`, `brain/guard`, `brain/wiki`, `hosts`, `journal`, `worktree`.
  Die Wiki-Lane baut `hooks`: ein `verify.Job` mit einer Funktion (`Fn`), die
  `wiki.LintReport` im Prozess ruft.
- `verify` → `child`, `detect`, `shellwords`; `child` → `gitenv`. `verify`
  importiert nie `hooks` und kein `brain/*`. (Nachtrag 2026-09-19, gegen
  `go list` gerechnet: Hier stand `verify` → `child`, `brain/check`; die
  Wiki-Lane ist aber in `hooks` gelandet, und `gitenv` erreicht `verify` nur
  über `child`.)
- `serve` → `brain/*`.
- `install` → `hosts`, `config`, `verify` (Presets).
- `serve`, `hooks` und `cli` → `selfupdate` → `swap`, `config`. `selfupdate`
  importiert keins der drei; die laufende Version reicht der Aufrufer herein.
  (Nachtrag 2026-09-23.)
- **`hooks` importiert nie `serve`.** Der Pfad an jedem Edit hängt nicht an
  einem laufenden Dienst.

### Startzeit-Regel

Ein Binary heißt: jedes `init()` und jede Paketvariable jedes importierten
Pakets läuft bei jedem `loomux hook pre-tool-use`. Deshalb:

- Kein `init()` und keine Paketvariable parst eingebettete Daten
  (wordfreq-Tabelle, Presets, Schemas). Geladen wird beim ersten Gebrauch.
- Nachweis mit `GODEBUG=inittrace=1 loomux --version` ab dem ersten Gerüst;
  der Befund steht in `docs/benchmarks.md`.

### Externe Programme zur Laufzeit

`git`, `qmd` (Node, gepinnt), `pdftotext` (Poppler) und `yt-dlp` für
`convert`/`fetch`, optional Ollama. Fehlt eins, meldet der betroffene Befehl
Name und Installationsbefehl und exitet ungleich 0. `claude` und `agy` braucht
diese Spec nicht; sie kommen mit dem Flow-Folgeprojekt. Optional ist auch
`gh` (Nachtrag 2026-09-23): Ohne es fällt nur das Self-Update aus, und der
Sitzungsstart meldet das.

## Konfiguration und Zustand

- **`.loomux/config.toml`**, von Hand gepflegt, Sektionen je Modul: `[project]`
  (u. a. `agents`), `[area]`, `[layout]`, `[index]`, `[verify]`, `[policy]`,
  `[commit]`, `[worktree]`. Kommentare englisch.
- **`.loomux/state/`**, nur maschinengeschrieben: Antworten des Installers,
  installierte Hookstände, Sitzungszustand, Blockzähler. Wo es geht
  git-ignoriert.
- **Global `%LOCALAPPDATA%\loomux\`:** `registry.toml`, `areas/`,
  `maintenance/`, `ui.json`, `serve.json`, Logs. Unter POSIX
  `$XDG_STATE_HOME/loomux`. Dazu (Nachtrag 2026-09-23) `bin/loomux.exe`, das
  maschinenweite Binary, sowie `update.json` und `update.lock` des
  Self-Updates.
- **qmd:** Index bleibt in `~/.cache/qmd`. Die Ignore-Muster, die heute
  `**/.brain.toml` und `**/.ultra-brain/**` nennen, bekommen `**/.loomux/**`.

## Hosts

Übernommen aus der Go-Hooks-Spec und ihrer Antigravity-Messung:

| loomux-Hook | Claude Code | Antigravity | Codex |
|---|---|---|---|
| Wächter | `PreToolUse` | `PreToolUse` | Naht |
| post-edit | `PostToolUse` | `PostToolUse` | Naht |
| `session-start` | `SessionStart` | `PreInvocation` + Sitzungsmarke | Naht |
| `subagent-start` | `SubagentStart` | `PreToolUse`, Matcher `invoke_subagent` | Naht |
| `subagent-stop` | `SubagentStop` | `PostToolUse`, Matcher `invoke_subagent` | Naht |
| `stop` | `Stop` | `Stop`, flache Liste | Naht |

- Der Host kommt als `--host claude|antigravity|codex` im Kommando, geschrieben
  von `loomux init`; keine Erkennung an der Nutzlast.
- Auf Antigravity ist das Arbeitsverzeichnis eines Hooks `.agents/`. Ohne
  `--root` sucht loomux aufwärts bis zur ersten `.loomux/config.toml`.
- Codex hat eine dokumentierte Naht und genau einen Test: unbekannter Host
  ⇒ Exit 2.
- `loomux init` schreibt `.claude/settings.json` und `.agents/hooks.json` aus
  einer Tabelle.

## Datenfluss

### Hook-Pfad an jedem Edit

1. Der Host startet `loomux hook pre-tool-use --host <h> --root <R>`, Nutzlast
   auf stdin. **Ein** Eintrag mit **einem** Timeout statt heute 10 s + 15 s.
2. `config` lädt die Projektkonfiguration und die globale Registry.
3. Entscheidung in einem Durchlauf:
   - **Policy** (geschützte Pfade, Befehlsregeln) für **jedes** Werkzeug,
     auch Bash und PowerShell.
   - **Schreibschranke** nur für schreibende Werkzeuge, **global aus der
     Registry**, unabhängig von `--root`.
   - Erste Ablehnung ⇒ Exit 2, Begründung auf stderr.
4. `loomux hook post-tool-use` wählt die Lanes nach Dateityp aus `[verify]`
   und fährt sie parallel; die Wiki-Lane ist ein Funktionsaufruf.
5. **Zielwert Stufe 1a:** ein Prozess, höchstens 72 ms warm (Parität). Stufe 1a
   misst, wohin die ~40 ms gehen, die `brain guard` über seinem Startboden
   verbringt; erst danach wird ein strengerer Zielwert für Stufe 2 gesetzt.

### Sitzungshooks

`session-start` hält die Sitzungsbasis fest und warnt, wenn das Pilot-Binary
älter ist als die jüngste Go-Quelldatei (`cmd/**`, `internal/**`, `go.mod`,
`go.sum`) — nicht als HEAD: das Tor baut das Binary vor dem Commit, und die
Commit-Zeit läge danach immer später; fällige Pflege meldet es ab Stufe 3. Wartende Flow-Läufe
meldet es erst mit der Flow-Migration — bis dahin schreibt kein loomux-Befehl
einen Lauf. `stop` fährt die Prüfkette auf den geänderten Dateien, mit
`MAX_BLOCKS`-Zähler im Sitzungszustand; `stop_hook_active` wird absichtlich
nicht gelesen. `subagent-start`/`-stop` halten den Remote-Stand fest und melden
Drift. **Zielwert Stufe 2:** unter 100 ms Eigenzeit je Hook, ohne die Zeit der
Tore selbst.

### Git-Hooks

- Pre-Commit und Pre-Push: `loomux check <profil>`.
- commit-msg: `loomux check commit-msg <datei>`.
- Beide lesen dieselbe `[verify]`-Tabelle wie post-edit. Die heutige
  Doppelung — Lanes fest in `post_edit.go`, Lanes aus der Konfiguration in
  Python — entfällt.

### `loomux serve`

- Bindet nur an `127.0.0.1`, Port frei gewählt. Adresse und ein zufälliger
  Token stehen in `%LOCALAPPDATA%\loomux\serve.json`; jede Anfrage trägt den
  Token.
- **Eine Instanz:** Sperrdatei plus Lebendprüfung; `loomux serve status` und
  `loomux serve stop`. Zwei gleichzeitig startende Hosts erzeugen keinen
  zweiten Dienst.
- **MCP** über Streamable HTTP mit `github.com/modelcontextprotocol/go-sdk`,
  Werkzeuge wie heute: `search`, `catalog`, `read`, `neighbors`, `status`.
  **Überholt — siehe `2026-09-17-loomux-stufe-1b-2-design.md`:** die Werkzeuge
  heißen `brain_search`, `brain_catalog`, `brain_read`, `brain_neighbors`,
  `brain_status`. Das Protokoll kennt keine verschachtelten Werkzeuge, und der
  Codegraph legt `graph_*` in denselben Server; das Präfix ist die einzige
  Familientrennung, die es gibt.
- **Kanäle** `local` und `cloud` wie heute (`local_only`-Bereiche sind in
  `cloud` unsichtbar). Den Kanal wählt die Brücke per Flag; er ist Teil der
  Anfrage, keine eigene Bindung.
  **Überholt — siehe `2026-09-17-loomux-stufe-1b-2-design.md`:** der Kanal
  **ist** die Adresse. `serve` bindet zwei Listener mit je einem eigenen Token;
  `--channel` der Brücke wählt, welchen sie anspricht, und kein Feld der
  Anfrage trägt ihn. Ein Aufrufer mit dem cloud-Token erreicht die
  local-Sicht dadurch gar nicht erst.
- **qmd:** `serve` startet qmd (`qmd mcp --http --daemon`), prüft seine
  Gesundheit und hängt Antworten den Warm-Hinweis an, solange das Modell lädt.
  Es sind zwei langlebige Prozesse.
- **Upkeep:** Ist der letzte `reconcile`-Lauf älter als 24 h, holt `serve` ihn
  nach und hängt den Catch-up-Hinweis an.
- **Brücke:** `loomux mcp --channel local|cloud` spricht stdio, findet den
  Dienst über `serve.json`, startet ihn bei Bedarf und leitet nur weiter.
  `.mcp.json` der Wirte zeigt auf die Brücke.
- **Self-Update (Nachtrag 2026-09-23):** Läuft `serve` aus
  `%LOCALAPPDATA%\loomux\bin\loomux.exe` mit einer Release-Version, prüft es
  eine Minute nach dem Start und danach alle 24 h das höchste Release seines
  Kanals (`beta` nimmt auch Prereleases), lädt es
  über `gh`, prüft `SHA256SUMS` und tauscht die Datei. Aktiv wird das neue
  Binary über die vorhandene Regel „neueres Binary ersetzt `serve`“. Das
  Ergebnis steht in `update.json`; der Sitzungsstart liest nur diese Datei.
  Siehe `2026-09-23-self-update-design.md`.

## Fehlerverhalten

- **Der Wächter vor dem Edit scheitert geschlossen.** Zusammenbruch oder
  unlesbare Konfiguration ⇒ Exit 2 mit Begründung, die Datei und Zeile nennt.
- **`.loomux/config.toml` ist für Agenten nie beschreibbar** (entschieden am
  2026-09-14). Sie ist zugleich Manifest — `[area]` und `[layout]` bestimmen
  die Schreibrechte — und trägt `[policy]`; ein Agent, der sie bearbeiten
  darf, könnte sich selbst Rechte geben. Das übernimmt die Regel, die
  `ultra-brain/pkg/guard` heute für `.brain.toml` und
  `.ultra-brain/config.toml` durchsetzt. Agenten schlagen Änderungen vor, der
  Mensch schreibt sie.
- **Das Stop-Tor behält die Exit-1-Disziplin** der heutigen `stop.py`: fehlende
  `session_id`, kaputte `[verify]`-Tabelle und eine komplett unbenutzbare Kette
  enden mit 1 und Zeile auf stderr, weil eine gehaltene Runde die Datei nicht
  reparieren kann, die sie hält. Die eine Unstimmigkeit (`stop.py:141-146`:
  Kommentar verlangt „nicht wie sauber beenden", Code exitet 1) wird mit 2 und
  gezähltem Block repariert. Kein `fail_open`-Schalter.
- **Kein grünes Tor ohne Arbeit.** Eine Lane, die nicht laufen kann —
  Werkzeug fehlt, leerer Pfad, fehlende Zusammenfassung — ist rot und nennt den
  Installationsbefehl.
- **Timeouts töten den Prozessbaum** (Job Object unter Windows, `killpg` unter
  POSIX), mit `cmd.WaitDelay` für die Pipes.
- **Registry und Zustand** werden atomar geschrieben (temporäre Datei,
  Umbenennen) unter einer prozessübergreifenden Sperre (Port von
  `locking.py`).
- **Dienst:** Findet oder startet die Brücke `serve` nicht, antwortet sie mit
  einem MCP-Fehler, der `loomux serve status` nennt. Kommt qmd nicht in der
  Frist hoch, gibt es nach dem Warm-Hinweis einen Fehler — nie eine leere
  Trefferliste.
- **Binärtausch unter Windows:** gebaut wird nach `bin/loomux.new.exe`, dann
  umbenannt. Ein laufendes `.exe` lässt sich umbenennen, nicht überschreiben.

## Paritätsnachweis

### Fallkorpus

Das sprachunabhängige Format aus `ultra-brain/bench/cases`: ein Verzeichnis je
Fall mit `cmd`, optional `stdin`, `exit`, `stdout`, `notes.md` und `world/`
(Zustand und Dateien vorher); die Dateiwelt danach wird verglichen. In loomux
unter `testdata/cases/<stufe>/<befehl>/<fall>/`.

### Aufzeichnen

- Vor jeder Stufe, in den **alten** Repos, von einem **getaggten** Commit; der
  Tag steht in `notes.md` jedes Falls.
- brain über den vorhandenen Rekorder `tools/cases.py`; für ultraloom ein
  gleichwertiger mit Hook-Nutzlasten auf stdin. Python läuft dabei nur in den
  alten Repos.
- **Hook-Nutzlasten** kommen aus echten Sitzungen: ein Tee-Wrapper in
  `settings.json` schneidet sie im Aufzeichnungsfenster mit; `session_id`,
  `transcript_path` und absolute Pfade werden normalisiert.
- Aufgezeichnet werden Erfolgs-, Ablehnungs- und Fehlerfälle jedes Befehls der
  Stufe.

### Übersetzen

Jede Stufe hat eine Abbildungstabelle: alter Befehl → `loomux …`,
`.brain.toml`/`.ultra-brain/config.toml`/`.ultraloom/*.toml` →
`.loomux/config.toml`, `%LOCALAPPDATA%\brain` → `…\loomux`. Übersetzt wird beim
Import, einmal; der Originalfall bleibt als Beleg daneben.

### Externe Programme

qmd, Ollama, `pdftotext`, `yt-dlp` und Git-Remotes werden an der Prozessgrenze
durch Stubs ersetzt, deren Antworten aus einem echten, einmal aufgezeichneten
Lauf stammen. Das Muster der injizierten Läufer gibt es schon (`Lookup`,
`Runner`).

### Vergleichsklassen

- **Daten** — `search`, `catalog`, `read`, `neighbors`, `status`: stdout exakt.
- **Meldungen** — Wächter, Hooks, `check`: Exit-Code und Dateiwelt exakt,
  Text frei.

### Referenz

- **Python** ist Referenz für `search`, `status`, `catalog`, `read`,
  `neighbors`: diese Form rufen die Skills und `ultraloom/.mcp.json` heute auf,
  und sie ist die vollständigere (Go-Suche trägt keine Identitäten).
- **Go** ist Referenz für `guard`, `lint`, `wiki-gate`: nur diese Form wird
  gerufen.
- Wo keine Regel greift, entscheidet der Nutzer über die Abweichungsliste.

### Coverage aus Fällen

Jeder Befehl hat einen Einstieg `Run(args []string, stdin io.Reader, stdout,
stderr io.Writer) int`. Die Fälle laufen im Prozess, damit `go test -cover`
sie zählt. Umgebung und Arbeitsverzeichnis setzt der Test über `t.Setenv` und
`t.Chdir` — ein `env`-Parameter erreichte die Pakete nicht, die Umgebung und
Heimatverzeichnis selbst lesen (`config.StateDir`, die Gedächtnis-Ausnahme des
Wächters).

### Abweichungsliste

`docs/.superpowers/parity/stufe-<n>.md` im loomux-Repo. Je Eintrag: Fall, altes
Verhalten, neues Verhalten, Begründung, Freigabe. Bereits bekannt für Stufe 1a:
**Das Scratchpad der Claude-Sitzung (`%TEMP%\claude\…`) wird für die
Schreibschranke freigegeben.**

## Tests und Tore

- TDD je Task; 100 % Coverage; jeder Ausschluss mit Begründung im Code. Die
  plattformgebundenen Arme (Job Object, `killpg`, Junction) liegen hinter
  Build-Tags und sind je auf ihrem System Einheiten-getestet.
- Umgezogene Go-Pakete bringen ihre Tests mit; neuer Code entsteht test-first.
  **Umgezogen heißt nicht fertig:** ultraloom steht heute bei 98,6 % (u. a.
  `junction` 93,1 %, `render` 97,2 %, `cmd/init` 97,7 %, `verify` 98,5 %,
  gemessen im Pre-Commit von `eed20b5`), ultra-brain ist nicht nachgemessen.
  Jedes umgezogene Paket wird beim Umzug auf 100 % gehoben oder bekommt
  begründete Ausschlüsse; diese Arbeit gehört zu der Stufe, die das Paket
  umzieht.
- **Golden-Dateien** für die Antwortformen von Claude und Antigravity.
- **Mutationstests:** `tools/go_mutants.py` wird als `loomux dev mutants
  <paket>` nach Go portiert, mit denselben vier Familien (a1–a4). Je Stufe eine
  Runde über die Entscheidungspakete (Wächter, Lanes, Locking); überlebende
  Mutanten stehen in der Paritätsdatei, mit Begründung oder nachgereichtem
  Test.
- **Messwerkzeug:** `loomux dev bench-hooks` — der Harness vom 2026-09-14 als
  Unterbefehl, damit jede Stufe ihre Zielwerte mit demselben Werkzeug misst.
- **Tore von loomux selbst:** Pre-Commit und Pre-Push fahren
  `loomux check precommit` — `gofmt`, `go vet`, `go test` mit 100 % Coverage;
  commit-msg prüft englisch.
- **Pilotbinary:** die Hooks im loomux-Repo rufen `bin/loomux.exe`; die
  Pre-Commit-Lane baut es neu; `session-start` warnt, wenn das Binary älter
  als HEAD ist.

## Datenumzug

1. **Wikis und Doku der beiden Repos** (Stufe 1b, vor dem Kopieren):
   - Prüfliste über alle Seiten von `ultra-brain/docs/wiki` (32) und
     `ultraloom/docs/wiki` (4) sowie die `docs/`-Bäume (107 und 75 versionierte
     Dateien): **behalten**, **aktualisieren** oder **im Archiv lassen**.
   - Der Nutzer gibt die Liste frei.
   - Dann kopieren, Bereich `project/loomux`, Links
     `brain://project/ultra-brain/…` und `brain://project/ultraloom/…`
     umgeschrieben. Die Seiten ziehen ohne Identitätsregister um: erhalten
     bleiben die Identitäten ihrer Quellen (`doc_id`, `content_hash`,
     `revision` unter `sources[]`); eigene `doc_id`s der Seiten prägt erst
     `reindex` in Stufe 3. Hier stand „samt `_identities.tsv` (10 und 1)"; die
     Zahl fiel, weil beide Register beim Nachzählen am 2026-09-16 nur die
     Kopfzeile trugen — das Register zog darum nicht mit und wurde in loomux mit
     nur der Kopfzeile neu angelegt.
2. **Maschinenzustand** (Stufe 4, `loomux migrate`): `registry.toml`,
   `areas/` (Identitätsregister werden umgezogen, nicht neu erzeugt),
   `maintenance/`, `ui.json` nach `%LOCALAPPDATA%\loomux\`, übersetzt. Das alte
   Verzeichnis bleibt als Sicherung, bis die Folgeprojekte fertig sind.
3. **qmd:** Collections werden mit den neuen Ignore-Mustern neu konfiguriert.
   Ob die Vektoren wiederverwendet werden, wird bei der Umstellung gemessen;
   wenn nicht, ist der Preis `reindex` + `embed` einmal je Bereich.
4. **Wirtsprojekte** (`loomux migrate` je Wirt): `.ultraloom/*.toml` und
   `.brain.toml`/`.ultra-brain/config.toml` → `.loomux/config.toml` und
   `.loomux/state/`. Ein einmaliger Übersetzer, keine Laufzeit-Kompatibilität.
5. **`brain-knowledge`** wird nicht angefasst. Das Remote-Risiko steht
   außerhalb dieser Spec, gehört aber vor die Umstellung gelöst.

## Stufen

Jede Stufe endet grün und wird einzeln übergeben; jede bekommt ihren eigenen
Plan.

**Stand am 2026-09-19.** Zwei Dinge sind anders gekommen, als diese Tabelle
ursprünglich annahm, und beide stehen unten in der Spalte statt in einer
Fußnote. Erstens sind **1b und 2 je in drei Teilstufen zerfallen**, weil jede
ihren eigenen Plan und ihre eigene Abnahme brauchte. Zweitens läuft **Säule 3 (der
Code-Graph) parallel** und nicht nach Stufe 4 — der Grund steht in §10 der
Säule-3-Spec und in §1 des G1-Deltas.

**Nachtrag 2026-09-22.** Auch **Stufe 3 ist in drei Teilstufen zerfallen**
(`2026-09-19-loomux-stufe-3-design.md`), und sie lief **parallel zu 2b und 2c**,
nicht nach ihnen: 3 hängt an keiner der beiden.

| Stufe | Stand | Inhalt |
|---|---|---|
| **1a** | ✅ | Repo-Gerüst, Lizenz, Tore, Startzeit-Nachweis. Umzug der Go-Pakete, die 1a benutzt, mit Tests, auf 100 % gehoben; jedes übrige Paket zieht mit der Stufe um, die es zuerst braucht. `config`, `hosts`, vereinter Wächter, post-edit mit Wiki-Lane im Prozess, `session-start`, `lint`, `wiki-gate`. `dev bench-hooks`. Pilot: das loomux-Repo nutzt sich selbst |
| **1b** | ➗ in drei Teilstufen zerfallen, siehe darunter | `search`, `status`, `catalog`, `read`, `neighbors` mit Parität zur Python-Referenz — neuer Go-Code, kein Umzug (Identitäten in der Suche, `status` vollständig). `serve` mit MCP und Brücke. `dev mutants`. Wiki- und Doku-Umzug |
| **2** | ➗ in drei Teilstufen zerfallen, siehe darunter | `child`, vollständige Prüfkette `[verify]`, `loomux check <profil>`, Coverage-Tor, commit-msg mit `--language`/`--calibrate`/`[commit]`, Hooks `stop`, `subagent-start`, `subagent-stop`, Antigravity-Adapter vollständig |
| **3** | ➗ in drei Teilstufen zerfallen, siehe darunter | Brain-Pflege: `reconcile` (auch als Durchgang vor `reindex`), `apply`/`approve`/`cases`/`evidence`/`vcs`, Bereichs-Onboarding, das Ereignisprotokoll, `wiki types`/`retype`/`census`/`scaffold`, Upkeep in `serve`. Korrigiert am 2026-09-22 nach der Spec der Stufe 3: „Locking“ stand hier, aber `internal/lock` ist seit 1a die Portierung von `locking.py`, fehlend war nur `ReplaceText`; und „`merge-events`“ ist kein Befehl, sondern das Ereignisprotokoll, das `reconcile` liest und ablegt — der Befehl ist `hook install\|status\|remove` und steht als Nachtrag #5 bei Stufe 4 |
| **4** | offen | `convert`/`fetch` über `pdftotext`/`yt-dlp`, lokales Modell (Ollama über `net/http`, deutsche Zipf-Tabelle eingebettet), `bench`. `loomux init` vollständig für alle Hosts, `loomux migrate`. Umstellung der Wirte |

**Die drei Teilstufen der 1b**, jede mit eigenem Plan und eigener
Paritätsakte (`parity/stufe-1b-1.md` und `parity/stufe-1b-2.md`, je samt
geparkten Mutanten, und `parity/umzug-wiki-doku.md` — die Akte trägt den
Namen des Umzugs, nicht die Nummer):

| Teilstufe | Stand | Inhalt |
|---|---|---|
| **1b-1** | ✅ 2026-09-15 | `search`, `status`, `catalog`, `read`, `neighbors` mit Parität zur Python-Referenz, Identitäten in der Suche, `status` vollständig. `dev mutants` entsteht hier |
| **1b-2** | ✅ 2026-09-19 (Spec und Plan `2026-09-17-loomux-stufe-1b-2*.md`) | `serve` mit MCP über Streamable HTTP, die stdio-Brücke. Upkeep ist nach Stufe 3 gewandert |
| **1b-3** | ✅ 2026-09-17 | Wiki- und Doku-Umzug |

**Die drei Teilstufen der 2**, jede mit eigener Spec und eigenem Plan; die
gebauten auch mit Paritätsakte (`parity/stufe-2a.md`, `parity/stufe-2c.md`):

| Teilstufe | Stand | Inhalt |
|---|---|---|
| **2a** | ✅ 2026-09-19 (Spec `2026-09-19-loomux-stufe-2a-design.md`, Plan `2026-09-19-loomux-stufe-2a.md`) | `child` (Prozessbaum, Fristen, Absaugen), das Schema `[verify]` mit Presets je Stack, `loomux check <profil\|arten>` mit `--show`, `loomux check gocover` statt `dev covergate`, post-edit auf `[verify]`. loomux prüft sich selbst mit `check precommit` |
| **2b** | ✅ | commit-msg mit `--language`, `--calibrate`, `[commit]` (umgesetzt 2026-09-19) |
| **2c** | ✅ 2026-09-22 (Spec `2026-09-19-loomux-stufe-2c-design.md`, Plan `2026-09-19-loomux-stufe-2c.md`) | Hooks `stop`, `subagent-start`, `subagent-stop` für Claude Code und Antigravity, das Profil `stop`, das Wiki-Bündel als Lane `lint/wiki`; hängt an 2a. Antigravity nachgemessen und Host-Adapter implementiert. Der Eintrag in `.claude/settings.json` (Task 16) ist gemacht; loomux prüft sich damit an jedem Rundenende selbst |

**Für 2c vorgemerkt:** zwei Schwächen der heutigen `stop.py` (gelesen am
2026-09-18), die `stop` nicht übernimmt.

- **Fristen widersprechen sich.** `[verify].timeout` ist je Befehl 600 s
  (`config.py:33`), der Host tötet den Stop-Hook aber nach 300 s. Eine lange
  Suite stirbt am Hook-Timeout, bevor die eigene Frist greift, und wird nicht
  rot gemeldet. 2a hat die Frage für post-edit so entschieden, wie sie auch
  hier gilt: `timeout` bleibt je Befehl und ohne Obergrenze beim Laden, das
  **Budget gehört dem Scope**. `stop` bekommt ein Budget fest unter seiner
  Hook-Frist, mit Abstand für Start, Git und Ausgabe, so wie post-edit 50 s
  unter 60 s hat; jedes Kind bekommt `min(eigene Frist, Restbudget)`. Die
  ursprüngliche Fassung dieser Regel (Zweig `docs/stufe-2-stop-notes`,
  `967f3e6`) wollte einen größeren `timeout` beim Laden ablehnen; das hat 2a
  verworfen (Abweichungsliste 2a, Eintrag 8).
- **Dauerlauf bei uncommitteter Arbeit.** Die Basis rückt nur nach grünem
  Lauf auf HEAD vor. Solange Änderungen uncommittet bleiben, fährt jedes
  Turn-Ende die ganze Suite erneut, auch wenn sich seit dem letzten grünen
  Lauf nichts geändert hat. `stop` merkt sich deshalb einen Fingerabdruck des
  grün geprüften Stands (HEAD, Diff gegen die Basis, untracked Inhalte) in
  `Snapshots` des Sitzungszustands (`internal/sessions/state.go`) und läuft
  nicht, solange er gleich ist.

Umgesetzt am 2026-09-22, beide anders gefasst als hier vorgemerkt: das Budget
ist `--budget`, Vorgabe 270 s unter der Frist von 300 s; der Fingerabdruck ist
der Inhaltsbaum (`gitwork.ContentTree`) und steht als `green` im
Sitzungszustand, `Snapshots` gibt es dort nicht mehr (2c-Spec, „Nachträge“).

**Die drei Teilstufen der 3**, eine gemeinsame Spec
(`2026-09-19-loomux-stufe-3-design.md`), je Teilstufe ein Plan und eine
Paritätsakte (`parity/stufe-3a.md`, `parity/stufe-3b.md`):

| Teilstufe | Stand | Inhalt |
|---|---|---|
| **3a** Erkennen | ✅ 2026-09-22 (Plan `2026-09-20-loomux-stufe-3a.md`) | `lock.ReplaceText`, `legacy.go` auf „neu zuerst, alt als Rückfall“, die Registry-Schreibseite, `loomux area add`, `loomux reindex` und `loomux embed` (Umzug `ultra-brain/pkg/index`, Nachtrag #17), `loomux reconcile` samt Lese- und Ablageseite des Ereignisprotokolls, der Auffangdurchgang vor `reindex`. Die Selbstnutzung lief zuerst auf Entscheidung des Nutzers nur gegen eine Kopie der Registry; der Umstieg folgte am 2026-09-22 nach dem Merge: ein `[index]` in `.loomux/config.toml` (Auflage S3 der Akte), dann `reindex` und `embed` über die echte Registry |
| **3b** Entscheiden | ✅ 2026-09-23 (Plan `2026-09-22-loomux-stufe-3b.md`, Bauweise hybrid, Stufe-3-Spec, „Bauweise“) | `loomux cases`, `loomux case`, `loomux approve`; `apply`, `evidence`, die Schreibseite von `vcs`. Hängt an den Fällen aus 3a. 24 Fälle gegen die Python-Referenz, 19 ohne Unterschied nach der Normalisierung (stderr nicht verglichen), 5 freigegeben. Die Selbstnutzung lief am 2026-09-23 gegen die echte Registry: `cases` und `case` über das Prüfzentrum des Tresors gleich der Python-Referenz, `approve` auf Entscheidung des Nutzers nur mit `--defer` (Akte, „Selbstnutzung“). Was nach 3b zu entscheiden bleibt, steht unter „Offen nach 3b“ |
| **3c** Pflegen | ✅ 2026-09-23 (Plan `2026-09-23-loomux-stufe-3c.md`, Akte `parity/stufe-3c.md`; 35 Fälle, 34 ohne Unterschied, einer freigegeben) | `loomux brain check file\|bundle\|all` mit OKF, Hausregeln, Föderation (#1; `loomux check all` ist die Prüfkette, darum unter `brain`); `loomux lint --scope all\|<scope>` (#2) mit den zwölf Regeln von `lint.py` als eigenem Regelsatz neben der Go-Form, die `lint <datei>`, `wiki-gate` und die Lane behalten; `loomux wiki init\|types\|retype` — die drei Befehle der Referenz, `census` ist `types` und `scaffold` ist `wiki init`; Upkeep in `serve`, der nur `reconcile` aus 3a ruft. `brain check code` fällt weg (#18) |

**Säule 3, der Code-Graph** (`2026-09-14-loomux-code-graph-design.md`). Sie
steht hier, weil sie neben den Fusions-Stufen läuft und nicht hinter ihnen:

| Stufe | Stand | Inhalt |
|---|---|---|
| **G1** | ✅ 2026-09-17 | `internal/code/{model,pagerank,blast}` — Lesemodell, Personalized PageRank, Blast-Radius, an portierten Testvektoren belegt |
| **G2a** | ✅ 2026-09-18 | Schema 2, `sourceset`, `extract/golang`, `resolve`, `store`, `freshness`, die Befehle `graph build` und `graph check` |
| **G2b** | ✅ 2026-09-18 | Die Abfrage: `lexicon`, `ask`, die Beiakte (`2026-09-17-loomux-code-g2b.md`) |
| **G3** | ✅ 2026-09-19 | `graph_find_code` und `graph_check_freshness` am MCP-Gateway von 1b-2 |
| **G4–G5** | offen | Die übrige `graph`-Palette samt Hook-Anbindung, Mehrsprachigkeit über `wazero` |

Vorgezogen wurde absichtlich: G1 und G2a ziehen keine Abhängigkeit ein, und die
Messung, die die alte Reihenfolge begründete (§11 der Säule-3-Spec:
Inittrace mit dem MCP-SDK), gehört zu G3, wo `internal/serve` das SDK
tatsächlich holt.

### Reihenfolge der offenen Stufen

Festgelegt am 2026-09-19. Drei Regeln, der Reihe nach: zuerst, was seine
Abhängigkeiten schon zulassen; dann, was loomux an sich selbst benutzt
(Bedingung 5 unter „Eine Stufe ist fertig, wenn“); dann die Größe. Der Weg zur
Ablösung der alten Repos ist 2c und 2b, dann 3, dann 4; G4 liegt daneben und
kann parallel laufen. **Nachtrag 2026-09-22:** 3 lief tatsächlich parallel zu
2b und 2c und steht nun auf Prio 1, weil sie an keiner der beiden hängt und die
größte Stufe ist. 2b und 2c sind fertig, und die Zeilen unter Prio 3 sind
lückenlos nachnummeriert — die Reihenfolge ist dieselbe.

Wörtlich steht in dieser Spec nur „2c hängt an 2a“. Jede andere Abhängigkeit
ist aus Spec und Code abgeleitet, der Grund steht in der Zeile.

| Prio | Stufe | Hängt ab von | Warum hier |
|---|---|---|---|
| — | **2c** `stop`, `subagent-*`, Antigravity-Adapter | 2a ✅ | ✅ Fertig (2026-09-22). Die Claude-Seite bringt den Lint des Wiki-Bündels als Lane `lint/wiki` ans Rundenende, die Drift-Regel bleibt bei `loomux wiki-gate`. Antigravity-Messung durchgeführt und Adapter implementiert. Voraussetzung für `init` in Stufe 4 |
| — | **2b** commit-msg | keine genannt | ✅ Fertig (2026-09-19). `migrate` in Stufe 4 überträgt `[project].commit_language` nach `[commit].language` (Nachtrag #13) |
| 1 | **3** Brain-Pflege (3a, 3b, 3c) | 1b-1 ✅, 1b-2 ✅ | Die größte Stufe, und sie hängt weder an 2b noch an 2c — darum lief sie parallel zu beiden. 3a ist fertig (2026-09-22); 3b ist fertig (2026-09-23), samt Selbstnutzung; 3c ist fertig (2026-09-23), der Upkeep ruft `reconcile` aus 3a. Upkeep läuft in `serve`; die Brain-Skills aus Stufe 4 rufen `brain check` (Nachtrag #1, #7) |
| 2 | **G4** übrige `graph`-Palette, Blast-Monitor | G3 ✅ | Sofort baubar, neben der Fusion. W3 und W4 warten darauf |
| 3 | **4** `init`, `migrate`, `convert`/`fetch`, Modell | 2b ✅, 2c ✅, 3 ✅ (3a, 3b, 3c) | Ohne Stufe 4 bleiben die alten Repos im Dienst. Die Umstellung der Wirte braucht zudem einen Remote für `brain-knowledge` (siehe „Umstellung der Wirte nach Stufe 4“) |
| 4 | Folgeprojekt **Flow** | ulflow M1 (Zweig `feature/agent-harness`, nicht gemergt) | Kein Wirt nutzt heute Flows (siehe „Befunde“) |
| 5 | **W1–W5** (Web-OS-Spec) | W1 an 1b-2 ✅; W2 an W1; W3 an W1 und G4; W4 an G4 und 4; W5 an W1 und Flow | Folgeprojekt |
| 6 | **G5** `wazero` | G4 | Nichts wartet darauf |

### Nachgetragen: was bisher keine Stufe hatte

Eine Durchsicht beider Quellrepos am 2026-09-19, am Code und nicht an der
Doku, fand sechzehn Stellen, die weder in loomux gebaut noch in dieser Spec
oder einer Paritätsakte genannt waren — gegen den Grundsatz unter „Ziel“, dass
nichts ersatzlos wegfällt, ohne in einer Liste zu stehen. Eine siebzehnte fand
am selben Tag die Spec der Stufe 3 (`2026-09-19-loomux-stufe-3-design.md`,
„Befunde“), eine achtzehnte am 2026-09-23 der Plan von 3c. Die Zuordnung ist ein **Vorschlag**; die Spalte „Freigabe“ füllt
der Nutzer, erst dann gilt sie.

| # | Quelle | Stelle | Vorschlag | Begründung | Freigabe |
|---|---|---|---|---|---|
| 1 | ultra-brain | `brain check file\|bundle\|all` mit den Achsen OKF, Hausregeln, Föderation (`pkg/check/{okf,house,run}`, `cmd/brain/main.go:937`) | Stufe 3 | Nur das Basispaket `check` ist umgezogen; `internal/brain/wiki/lint.go` verweist die Regeln (`wrong-direction`, `long-planned`, `no-sources`, `log-date-form` …) an Checks, die es in loomux nicht gibt. Die Brain-Skills rufen `brain check` | freigegeben 2026-09-19, Stufe 3c; gebaut 2026-09-23 |
| 2 | ultra-brain | `lint` ohne Pfad und mit `--scope all` (`src/brain/cli.py:562`) | Stufe 3, mit #1 | `loomux lint` verlangt genau eine Datei. Den Lint über das ganze Bündel hat heute nur `wiki-gate`, und das nur zusammen mit der Driftprüfung | freigegeben 2026-09-19, Stufe 3c; gebaut 2026-09-23 |
| 3 | ultra-brain | `embed` als Befehl (`src/brain/cli.py:467`) | Stufe 3 | Gehört zu `reindex`; der Datenumzug (Punkt 3) rechnet schon mit „`reindex` + `embed` einmal je Bereich“. In loomux gibt es nur `QmdMcpPort.Embed` ohne Befehl | freigegeben 2026-09-19, Stufe 3a; gebaut 2026-09-22 mit #17 |
| 4 | ultra-brain | `brain layout` und `layout.json` (`pkg/layout`, `cmd/brain/main.go:340`) | Folgeprojekt Web-Migration | Nur die Web-App liest die Orte aus `layout.json` (`web/src/canvas/cosmos.test.ts`) | |
| 5 | ultra-brain | `hook install\|status\|remove`: der post-merge-Hook in einwilligenden Repos (`src/brain/cli.py:625`) | Stufe 4 (`loomux init`) | Hooks schreibt `init`; der Hook selbst speist `merge-events` aus Stufe 3. Der Abschnitt „Git-Hooks“ kennt ihn noch nicht | |
| 6 | ultra-brain | `daemon start\|run --backbone`, `--no-local`, `--no-cloud` (`src/brain/cli.py:673`) | Abweichung 1b-2 nachtragen | `serve` kennt nur `--foreground`, das Backbone ist fest CUDA (`internal/brain/search/daemon.go:21`). Die Wahl CUDA oder Vulkan gehört unter „Offen und vor dem Bau zu messen“ | |
| 7 | ultra-brain | Die Skills `brain-ingest`, `brain-land`, `brain-research`, `brain-review`, `brain-wiki-plan` (`.claude/skills/`) | Stufe 4 (`loomux init`), Befehle nach #1 umgeschrieben | Keine davon ist eine Review-Suite aus W4; `init` legt Skills in die Wirte | |
| 8 | ultraloom | Skill `verify-until-green` (`templates/skills/…SKILL.md.tmpl`) | Stufe 4 (`loomux init`) | Ruft fest `uv run ultraloom check all`; wird zu `loomux check all`. Der Flow dahinter bleibt Folgeprojekt 1 | |
| 9 | ultraloom | Skill `session-handover` | Stufe 4 (`loomux init`) | Ohne Abhängigkeit auf einen Befehl; zieht mit den übrigen Skills | |
| 10 | ultraloom | Die erzeugte `AGENTS.md` (`internal/render/render.go:109`, `templates/AGENTS.md.tmpl`) | Stufe 4 (`loomux init`) | `render`/`install` ziehen dort um; die Vorlage war nicht genannt | |
| 11 | ultraloom | `.gitignore`-Einträge des Installers (`cmd/init/run.go:381`) | Stufe 4 (`loomux init`) | Mit den Pfaden von loomux: `.loomux/state/` statt `.ultraloom/hooks/` | |
| 12 | ultraloom | `[relevance]` (`internal/render/templates/config.toml.tmpl:22`, `cmd/init/run.go:678`) | Wegfall | Der Installer schreibt ihn, aber kein Hook liest ihn, schon in ultraloom nicht. Die Presets je Stack decken die Absicht ab. `loomux migrate` lässt ihn fallen und sagt es | |
| 13 | ultraloom | `[project].commit_language` (`cmd/init/run.go:507`) | Stufe 2b, `migrate` in Stufe 4 | Nur `ulinit` liest ihn als Antwortvorgabe; die Prüfung liest `[commit].language`. `migrate` übersetzt ihn dorthin | |
| 14 | ultraloom | `[agent].settings`, `[agent].mcp_servers` (`config.py:219`) | Folgeprojekt 1 (Flow-Migration) | Sie steuern den Aufruf von `claude -p` in Flows. Die Tabelle „Wegfall aus dem alten Schema“ der 2a-Spec streicht nur `cli_path` | |
| 15 | ultraloom | `scripts/install.ps1` und `install.sh`: Bauen in `~/go/bin` | Stufe 4 (`loomux init`) | Release-Archive gibt es, aber keinen Weg, das Binary auf den `PATH` zu legen | |
| 16 | ultraloom | `ulinit --detect-only` (`cmd/init/main.go:44`) | Stufe 4, als `loomux init --detect-only` | `internal/detect` ist da, nur ohne Befehl | |
| 17 | ultra-brain | `reindex` und `embed` als Befehle (`ultra-brain/pkg/index`, `src/brain/cli.py:463,467`); `embed` allein ist #3 | Stufe 3a | Der Auffangdurchgang koppelt `reconcile` an `reindex` („`reconcile` auch als Durchgang vor `reindex`“ nennt einen Befehl, den es in loomux nicht gab), und `embed` ist ohne `reindex` gegenstandslos | freigegeben 2026-09-19, Stufe 3a; gebaut 2026-09-22 |
| 18 | ultra-brain | `brain check code` (`pkg/check/code`, die Lanes aus `[check].lanes`) | Wegfall | Die Prüfkette aus 2a (`[verify]`, Presets je Stack, `loomux check`) fährt dieselben Lanes samt Reihenfolge und Coverage-Tor; eine zweite Lane-Konfiguration stünde daneben. `Manifest.Lanes` bleibt geparst, bis `loomux migrate` (Stufe 4) es nach `[verify]` überträgt. Gefunden beim Planen von 3c | freigegeben 2026-09-23 |

### Offen nach 3b

Drei Fehler, die 3b von der Python-Referenz geerbt und absichtlich
nachgebildet hat, und eine Lücke, die 3b festgehalten hat. Keiner ist
geheilt, und hier steht kein Vorschlag: jeder wartet auf eine Entscheidung
des Nutzers, ob und wie er geheilt wird. Die Belegstellen stehen in
`parity/stufe-3b.md`.

1. **`--reject` schiebt Revision und Hash nicht vor.** Die Ablehnung schreibt
   `audit.md`, entfernt den Fall und committet, schiebt aber weder die
   `sources[]` der Seite noch das Register vor; der nächste Abgleich eröffnet
   denselben Fall wieder. Akte, Abschnitt „Geerbt“, erster Absatz.
2. **Eine schließende Frontmatter-Zeile mit Leerraum (`--- `) verliert bei
   der Freigabe die alte Frontmatter.** Die zwei Frontmatter-Muster der
   Referenz sind sich über diese Zeile uneinig; geschrieben wird dann nur
   `generated` und `verified`. Akte, Abschnitt „Geerbt“, zweiter Absatz, und
   Zeile „Zwei Frontmatter-Muster“ der Abweichungsliste (Golden
   `s13-closing-blanks`).
3. **Versteckte Auszeichnung (`%%`, `<!--`, Bidi-Steuerzeichen) auf einer
   Überschriftenzeile eines Vorschlags wird keinem Abschnitt angelastet.**
   Akte, Abschnitt „Überlebende Mutanten“, Absatz „Geerbt, festgehalten und
   nicht geheilt“; `TestHiddenMarkupOnAHeadingLineIsChargedToNoSection` hält
   das Verhalten fest.
4. **`reindex` und `approve` teilen keine Sperre.** Laufen beide zugleich über
   einen schreibgeschützten Bereich, kann `reindex` das Register zwischen
   Lesen und Tausch durch `approve` neu schreiben (eine Zeile geht verloren,
   der nächste Abgleich eröffnet den Fall neu), oder beide treffen sich beim
   ersten Schreiben unter `<zustand>/areas/<scope>`. Akte, Zeilen „Register
   vorschieben ohne Sperre“ und „Erstes Schreiben in einen schreibgeschützten
   Bereich“ der Abweichungsliste.

### Eine Stufe ist fertig, wenn

1. alle übersetzten Fälle der Stufe grün sind oder freigegeben in der
   Abweichungsliste stehen,
2. die Coverage 100 % ist, jeder Ausschluss begründet,
3. die Mutationsrunde der Stufe gelaufen ist und ihre Überlebenden
   dokumentiert sind — ab Stufe 1b, weil `loomux dev mutants` dort entsteht;
   die Runde von 1b schließt die Entscheidungspakete aus 1a ein,
4. die Zielwerte der Stufe gemessen und in `docs/en/benchmarks.md` und
   `docs/de/benchmarks.md` eingetragen sind,
5. das loomux-Repo die Funktionen der Stufe selbst benutzt.

### Umstellung der Wirte nach Stufe 4

1. `brain-knowledge` hat einen Remote und ist committet.
2. Je Wirt: `loomux migrate`, dann `loomux init`. Alte Hook-Einträge werden an
   der Marke `ultraLoomOwned: true` (Claude) und am Gruppennamen
   `ultraloom-wiki-guard` (Antigravity) erkannt und entfernt.
3. Rauchtest je Wirt: ein erlaubter Edit, ein verweigerter Edit, ein Commit
   durch commit-msg und Pre-Commit, eine Suche über MCP.

## Folgeprojekte

In dieser Reihenfolge, je mit eigener Spec:

1. **Flow-Migration.** Basis ist ulflow M1 (`feature/agent-harness`: Laufzeit,
   Journal, Resume, Replay, `internal/flowload`). Dazu M2 (`claude -p`,
   `agy -p`) und M3 (`verify_until_green` als Daten-Flow). Das Flow-Format ist
   heute TOML mit Go-Bausteinen und Go-Prädikaten; ob es für den Editor so
   bleibt oder um Ausdrücke erweitert wird, entscheidet dieses Folgeprojekt.
   Bis dahin bleiben `ulflow` und die Python-Flows in ihren Repos benutzbar.
2. **Web-Migration.** Die React/Vite-App aus `ultra-brain/web` zieht nach
   `loomux/web`, ihr Build wird per `go:embed` eingebettet, `serve` bekommt
   `/api/…`. Mit ihr zieht das Bundle-Tor aus `hooks/bundle-gate.sh` um.
3. **Grafischer Flow-Editor** auf der Web-App und der Bausteinregistry.

Die alten Repos werden archiviert, wenn beide Migrationen fertig sind.

## Offen und vor dem Bau zu messen

- **Wohin gehen die ~40 ms von `brain guard`** über seinem Startboden? (Stufe 1a,
  vor dem Zielwert für Stufe 2.)
- **Antigravity:** ob die JSON-Hülle auf stdout gelesen wird oder nur der
  Exit-Code; ob `PreInvocation` Kontext ins Modell schreiben kann; ob Stop eine
  harte Zeitgrenze unter 300 s hat. Ergebnisse der Messung vom 2026-09-10
  (`2026-09-10-antigravity-hook-messung.md`) werden in Stufe 1a gegen diese drei
  Fragen gelesen, fehlende nachgemessen.
- **Vektorwiederverwendung in qmd** nach geänderten Ignore-Mustern (bei der
  Umstellung).
- **Go-Suche:** woher die abweichende Rangfolge gegenüber Python kommt — der
  erste Paritätsfall von Stufe 1b.
