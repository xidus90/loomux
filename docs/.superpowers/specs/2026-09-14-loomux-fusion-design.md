# loomux — ultraloom und ultra-brain in einem Go-Binary

**Datum:** 2026-09-14
**Stand:** teilweise umgesetzt (2026-09-27).
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
(2026-09-23), die lesenden Befehle samt Selbstnutzung. Stufe 4 ist offen und
am 2026-09-24 in 4a-1, 4a-2, 4c, 4d und den Abschlussschritt 4e geschnitten
(`2026-09-23-loomux-stufe-4-design.md`), 4c beim Planen am 2026-09-25 in 4c-1
(Modell) und 4c-2 (Bench); `loomux migrate` fällt weg. 4a-1, 4a-2 und 4c-2
sind fertig (2026-09-28), 4c-1 am 2026-09-29; 4d ist fertig
(2026-09-27), samt Selbstnutzung am echten Eingang; 4e ist offen. Die Stufe 4f (keine Verweise auf die Altprojekte, Nachtrag #24) ist
am 2026-09-28 freigegeben und offen. Vierundzwanzig Stellen, die keine Stufe hatten, stehen unter
„Stufen“ im Abschnitt „Nachgetragen“; alle sind freigegeben, ebenso die beiden
Nachträge #25 und #26 zur Stufe 4e und #27, die Schonfrist je Lane, auf die
ihre Welle wartet (2026-09-30).
**Säule 3 (Code-Graph), vorgezogen und parallel gebaut:** G1 (Modell, PageRank,
Blast) am 2026-09-17, G2a (Extraktor, Auflösung, Speicher, Frische, die
Befehle `graph build` und `graph check`) und G2b (die Abfrage) am 2026-09-18,
G3 (die Abfrage über MCP) am 2026-09-19, G4a (die Navigation) am 2026-09-22 und G4b (Diff-Blast,
die Art `graph`, der Edit-Monitor) am 2026-09-23 und G4c (der Stop-Hook mit Blast-Logik) am 2026-09-28 abgeschlossen; G5a abgeschlossen am 2026-09-26, G5b–d offen. Die Vorziehung war Absicht: beide Stufen
ziehen keine Abhängigkeit ein, und die Messung, die die alte Reihenfolge
begründete, gehört zu G3
(`2026-09-14-loomux-code-graph-design.md` §10, `2026-09-16-loomux-code-g1-delta.md` §1)
**Ort:** `docs/.superpowers/specs/` in `xidus90/loomux`; bis Stufe 1a lag die
Spec vorläufig in einem Worktree von `ultraloom`.
**Löst ab:** ulflow M4–M6
(`docs/.superpowers/specs/2026-09-11-ulflow-laufzeit-design.md`, liegt nur auf
`feature/agent-harness`),
[Go-Hooks für drei Hosts](../specs-ul/2026-09-10-go-hooks-drei-hosts-design.md), Stufen 2–5,
[Wiki-Flottenstandard](../specs-ul/2026-09-10-wiki-flottenstandard-design.md), Stufen 3–5.
**Messgrundlage:** `docs/de/benchmarks.md`, Eintrag 2026-09-14 16:05.

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

### Messungen (warm, Median, `docs/de/benchmarks.md`)

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
| Wiki-Flottenstandard | Stufe 2 (`internal/agenthooks`) wird fertig. Nachtrag 2026-09-24: sie wurde es nicht, Task 2 von 4 liegt ungemergt auf `claude/wiki-stufe-2` (Lückentabelle #21). Nachtrag 2026-09-27: die Regeln sind am 2026-09-25 mit 4a-2 in `internal/setup/hostfile` gebaut (#21), `agenthooks` zog nicht um; in ultraloom bleibt nichts zu tun | Stufen 3–5 gehen in `loomux init` auf; der Grundsatz „Projektdoku im Projekt, Übergreifendes in `brain-knowledge`" bleibt |

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
| *Nachtrag 2026-09-27* | Gegen `ls internal` gelesen: `cases` liegt unter `internal/cases`, nicht unter `brain/`; `brain/bench` wurde `dev/bench{corpus,hooks,report,search}`; `brain/convert` kommt mit 4d; `install` heißt `setup`; `journal` gibt es nicht, der Sitzungszustand liegt in `sessions`, der Laufzustand der Flows kommt mit dem Flow-Folgeprojekt. Dazu kamen `code/*` (Säule 3), `bridge` (stdio-Brücke), `mcptools`, `tui`, `lock`, `gitwork`, `release`, `dev`, `pathkey`, `conventional`, `detect`, `gitenv`, `shellwords`, `testlock` | |
| `worktree` | Worktree-Spiegel | `ultraloom/internal/{worktreetopo,junction,mirrorcfg,gitwork}` |
| `serve` | MCP-Dienst, Upkeep, stdio-Brücke | `src/brain/{daemon,ipc,client,mcp}`, `ultra-brain/pkg/mcp` |
| `selfupdate`, `swap` | Update des maschinenweiten Binarys aus dem Release; Tausch eines laufenden Binarys (Nachtrag 2026-09-23, `swap` war `dev/swap`) | neu |
| `install` | `loomux init` (`migrate` entfällt, Nachtrag #19; das Paket heißt nach der Stufe-4-Spec `internal/setup`) | `ultraloom/internal/{answers,interview,render,tooling,write,settings,agenthooks,detect}`, `src/brain/init.py` |
| `journal` | Sitzungs- und Laufzustand, soweit Hooks ihn brauchen | `ultraloom/internal/{journal,sessions}` |

**Entfällt ersatzlos:** `ultraloom/internal/vendoring` — es pinnt eine
Python-ultraloom-Laufzeit ins Projekt. `internal/brainpath` wird gegenstandslos,
weil brain kein fremdes Programm mehr ist.

### Abhängigkeitsregeln

- Jedes Paket darf `config` benutzen.
- `hooks` → `verify`, `brain/guard`, `brain/wiki`, `hosts`, `journal`, `worktree`.
  Die Wiki-Lane baut `hooks`: ein `verify.Job` mit einer Funktion (`Fn`), die
  `wiki.LintReport` im Prozess ruft. (Nachtrag 2026-09-27, gegen `go list -deps`
  gerechnet: `hooks` → `verify`, `brain/guard`, `brain/wiki`, `brain/check`,
  `code/*` für den Blast-Monitor aus G4b, `hosts`, `sessions`, `selfupdate`,
  `worktree`; `journal` gibt es nicht. `internal/cli/imports_test.go` verbietet
  `hooks` außerdem die Pflegeschicht von `brain` und die Oberfläche von
  `loomux config`.)
- `verify` → `child`, `detect`, `shellwords`; `child` → `gitenv`. `verify`
  importiert nie `hooks` und kein `brain/*`. (Nachtrag 2026-09-19, gegen
  `go list` gerechnet: Hier stand `verify` → `child`, `brain/check`; die
  Wiki-Lane ist aber in `hooks` gelandet, und `gitenv` erreicht `verify` nur
  über `child`.)
- `serve` → `brain/*`.
- `install` → `hosts`, `config`, `verify` (Presets).
- `serve`, `hooks` und `cli` → `selfupdate` → `swap`, `lock`. `selfupdate`
  importiert keins der drei; die laufende Version reicht der Aufrufer herein.
  (Nachtrag 2026-09-23; `lock` statt `config` berichtigt am 2026-09-27 nach
  `go list -deps ./internal/selfupdate`.)
- **`hooks` importiert nie `serve`.** Der Pfad an jedem Edit hängt nicht an
  einem laufenden Dienst.

### Startzeit-Regel

Ein Binary heißt: jedes `init()` und jede Paketvariable jedes importierten
Pakets läuft bei jedem `loomux hook pre-tool-use`. Deshalb:

- Kein `init()` und keine Paketvariable parst eingebettete Daten
  (wordfreq-Tabelle, Presets, Schemas). Geladen wird beim ersten Gebrauch.
- Nachweis mit `GODEBUG=inittrace=1 loomux --version` ab dem ersten Gerüst;
  der Befund steht in `docs/de/benchmarks.md` und `docs/en/benchmarks.md`.

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
  `[commit]`, `[worktree]`. Kommentare englisch. Seit Stufe 4 dazu
  `[modules]` (`hooks`, `brain`, `graph`: welche Module laufen, fehlend heißt
  an; der Wächter prüft immer) und `[model]` (das lokale Modell, global und je
  Bereich, aus schlägt an). Geschrieben wird die Datei von einem Menschen,
  auch über `loomux config` und `loomux init`, die jede Änderung einzeln
  bestätigen lassen und einem Agenten verweigert werden. (Nachtrag 2026-09-27,
  gegen `internal/config/schema` gelesen: `[project]` liest kein Leser; dazu
  kamen `[wiki]`, `[maintenance]` und `[privacy]` der Bereichsdeklaration.)
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

- Pre-Commit: `loomux check <profil>`. (Berichtigt am 2026-09-27: hier stand
  „Pre-Commit und Pre-Push“; `.githooks/pre-push` fährt keine Prüfkette, es
  verweigert nur einen Push nach `master`.)
- commit-msg: `loomux check commit-msg <datei>`.
- Beide lesen dieselbe `[verify]`-Tabelle wie post-edit. Die heutige
  Doppelung — Lanes fest in `post_edit.go`, Lanes aus der Konfiguration in
  Python — entfällt.
- post-merge (Stufe 4, Nachtrag #5): ein kurzes sh, das `loomux merge-hook
  record` ruft und immer mit `exit 0` endet; `record` hängt das
  Merge-Ereignis an `maintenance/merge-events.tsv`. Eingerichtet von
  `loomux merge-hook install` bzw. `loomux init`.

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
  Von Hand stößt `loomux upgrade` denselben Durchlauf an (Nachtrag
  2026-09-28: vorher `loomux self-update`, umbenannt ohne Alias, weil „self“
  ungenau war — der Befehl ersetzt das maschinenweite Binary, nicht das
  laufende).
  Siehe `2026-09-23-self-update-design.md`.

## Fehlerverhalten

Grundsatz 5 von ultra-brain („Ausfälle degradieren, sie blockieren nicht“)
gilt weiter für die Wissensschicht: Fällt qmd, der Index oder das Modell aus,
wird loomux unbequemer und sagt das laut, nie unbenutzbar und nie stumm leer.
Wächter und Tore sind ausgenommen, sie scheitern geschlossen (entschieden am
2026-09-24):

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
- **Messwerkzeug:** `loomux dev bench hooks` (bis 4c-2 `dev bench-hooks`) —
  der Harness vom 2026-09-14 als Unterbefehl, damit jede Stufe ihre Zielwerte
  mit demselben Werkzeug misst.
- **Tore von loomux selbst:** Pre-Commit fährt `loomux check precommit` —
  `gofmt`, `go vet`, `go test` mit 100 % Coverage; commit-msg prüft englisch;
  Pre-Push verweigert nur einen Push nach `master`.
- **Pilotbinary:** die Hooks im loomux-Repo rufen `bin/loomux.exe`; die
  Pre-Commit-Lane baut es neu; `session-start` warnt, wenn das Binary älter
  ist als die jüngste Go-Quelldatei (siehe „Sitzungshooks“; hier stand „älter
  als HEAD“).

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
2. **Maschinenzustand:** `registry.toml`, `areas/` (Identitätsregister werden
   umgezogen, nicht neu erzeugt), `maintenance/` nach `%LOCALAPPDATA%\loomux\`.
   Korrigiert am 2026-09-24: `loomux migrate` fällt weg (Nachtrag #19). Die
   Selbstnutzung seit 3a hat den Bestand schon ins neue Verzeichnis gebracht;
   was bleibt, ist ein einmaliger Abgleich von Hand in der Checkliste von 4e.
   `ui.json` zieht mit der Web-Migration. Das alte Verzeichnis bleibt als
   Sicherung, bis die Folgeprojekte fertig sind.
3. **qmd:** Collections werden mit den neuen Ignore-Mustern neu konfiguriert.
   Ob die Vektoren wiederverwendet werden, wird bei der Umstellung gemessen;
   wenn nicht, ist der Preis `reindex` + `embed` einmal je Bereich.
   Unabhängig davon verlangt ein Wechsel des Backbones (CUDA, Vulkan) ein neues
   `embed`: Unter Vulkan liefert ein unter CUDA eingebetteter Index 0/50
   (`parity/stufe-4c-2.md`, Nachtrag 2026-09-27).
4. **Wirtsprojekte:** Kein Übersetzer (korrigiert am 2026-09-24, Nachtrag
   #19). `loomux init` richtet jeden der vier Wirte neu ein und fragt ab, was
   `.ultraloom/*.toml` und `.brain.toml` hielten; die alten Dateien entfernt
   der Mensch nach der Checkliste von 4e. Korrigiert am 2026-09-29 (Nachtrag
   #26): Es sind acht Projekte und der Vault, nicht vier Wirte. Der Agent
   übersetzt die alten Dateien in eine vollständige `.loomux/config.toml` als
   Vorschlag, `loomux init --yes` läuft im `apply.sh`, das ein Mensch aufruft,
   und dasselbe Skript entfernt danach die alten Dateien.
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
| **3** | ➗ in drei Teilstufen zerfallen, siehe darunter | Brain-Pflege: `reconcile` (auch als Durchgang vor `reindex`), `apply`/`approve`/`cases`/`evidence`/`vcs`, Bereichs-Onboarding, das Ereignisprotokoll, `wiki types`/`retype`/`census`/`scaffold`, Upkeep in `serve`. Korrigiert am 2026-09-22 nach der Spec der Stufe 3: „Locking“ stand hier, aber `internal/lock` ist seit 1a die Portierung von `locking.py`, fehlend war nur `ReplaceText`; und „`merge-events`“ ist kein Befehl, sondern das Ereignisprotokoll, das `reconcile` liest und ablegt — der Befehl ist `hook install\|status\|remove` und steht als Nachtrag #5 bei Stufe 4; in loomux heißt er seit 4a-2 `loomux merge-hook install\|status\|remove\|record` |
| **4** | ➗ in Teilstufen zerfallen, siehe darunter | `convert`/`fetch` über `pdftotext`/`yt-dlp`, lokales Modell (Ollama über `net/http`, deutsche Zipf-Tabelle eingebettet), `bench`. `loomux init` vollständig für alle Hosts. Umstellung der Wirte. `loomux migrate` fällt weg (Nachtrag #19) |

**Die Teilstufen der 4** (Spec `2026-09-23-loomux-stufe-4-design.md`,
geschnitten am 2026-09-24), je mit eigenem Plan und eigener Paritätsakte:

| Teilstufe | Stand | Inhalt |
|---|---|---|
| **4a-1** Schema und `config` | ✅ 2026-09-28 (gebaut 2026-09-24; Arbeitsverzeichnis gemessen, drei Terminals und `config set` durch den Menschen geprüft, Mutationsrunde über den Stand vom 2026-09-28 neu gefahren, Akte `parity/stufe-4a-1.md`) | Schlüsselschema, Zeileneditor, Oberfläche auf `x/term`, `loomux config` (auch `--global`), `[modules]` mit Laufzeitwirkung, der Modulfilter in `loomux mcp`, die Wächterregel gegen schreibende `init`/`config`/`area add` |
| **4a-2** `init` | ✅ 2026-09-28; gebaut 2026-09-24 (Plan `2026-09-24-loomux-stufe-4a-2.md`, Akte `parity/stufe-4a-2.md`; 14 Fälle für `merge-hook`, elf ohne Unterschied); Messung von Task 1 am 2026-09-24 gemacht (Hookdatei und Skill-Ort von Antigravity, `${LOCALAPPDATA}` in `.mcp.json` und in einem Git-Hook), Antigravity-Einträge und -Skills gemessen und am 2026-09-25 gebaut (#21; #23 freigegeben 2026-09-28); Schritte des Menschen erledigt 2026-09-28 (`init --yes` auf einem frischen Klon, ein Wirt interaktiv, Freigabe einer Projekt-`.mcp.json`), Mutationsrunde über den Stand vom 2026-09-28 neu gefahren | Umzug von ulinit auf das Schema, Module mit alles/einzeln/nichts, Host-Einträge über `${LOCALAPPDATA}`, Git-Hooks und post-merge, Skills, `AGENTS.md`, `.gitignore`, `.mcp.json`, das neueste Release an den kanonischen Ort, `--detect-only`. Hängt an 4a-1 und an `feat/self-update` |
| **4c-1** Modell | ✅ 2026-09-29; gebaut 2026-09-26; Selbstnutzung erledigt (ein Fall mit Vorschlag in obsidian-ai; als Nachweis für Loopback gilt der Code, entschieden vom Nutzer, weil der `pktmon`-Mitschnitt auf dem Rechner nicht aussagekräftig war); die Messung gegen das echte Modell lief am 2026-09-28 (warm rund 580 ms je Vorschlag; kalt lief die erste Frage in die Frist von 30 s, seit dem Aufwärmen vor der ersten Frage (2026-09-29) kommt der Vorschlag kalt nach 17 s); geplant 2026-09-25 (Stufe-4-Spec, „Abweichungen beim Planen von 4c“, Plan `2026-09-25-loomux-stufe-4c-1.md`, Akte `parity/stufe-4c-1.md`) | Ollama-Client, Tor, Prompts, `[model]` global und je Bereich, `propose` in `reconcile`; Heilung von „Offen nach 3b“ #1 und #4 |
| **4c-2** Bench | ✅ 2026-09-28; gebaut 2026-09-26; Parität ✅ 2026-09-27 (50/50 Ränge gleich der Referenz bei `keyword`); Selbstnutzung erledigt 2026-09-27 (Korpus `fast` 40/50 gegen die Baseline 43/50, Alltagslatenz über den Dienst, `hooks` und `repos` mit `--out`), Alltagsqualität gemessen (qmd-Backbone: der Index ist unter CUDA eingebettet, das auf diesem Rechner abstürzt; entschieden 2026-09-27: ein rechnerweites `[search] backbone = "cuda" | "vulkan" | "cpu"` in `<zustand>/config.toml`, kein Schlüssel je Projekt, Vorrang Umgebungsvariable des Nutzers > Einstellung > `cuda`, gilt für den Daemon und jede qmd-Kommandozeile; neu eingebettet und gemessen am 2026-09-28, `fast` über mehrere Bereiche nach der Korrektur in `QmdMcpPort.Search` 31/50); Mutationsrunde am 2026-09-28 gefahren, jeder Überlebende getötet oder begründet; geplant 2026-09-25 (ebenda, „Abweichungen beim Planen von 4c-2“, Plan `2026-09-26-loomux-stufe-4c-2.md`, Akte `parity/stufe-4c-2.md`) | `loomux dev bench search`, die Untergruppe `dev bench hooks\|repos\|search` (bisher `dev bench-hooks` und `dev bench`, darum `release:major`) und ein Berichtsschema für alle drei. Hängt nicht an 4c-1 |
| **4d** `convert`/`fetch` | ✅ 2026-09-27 (Plan `2026-09-26-loomux-stufe-4d.md`, Akte `parity/stufe-4d.md`; 29 Fälle ohne Unterschied, die Aufnahme von Poppler 25.07.0 als Go-Golden; Selbstnutzung am echten Eingang von `knowledge` ohne Modell, die Scan-Schwelle 100 trägt an echtem Poppler); geplant 2026-09-26 (Stufe-4-Spec, „Abweichungen beim Planen von 4d“) | Eingang wandeln, Untertitel holen, die Modellrollen `describe` und `place`, die Richter samt Zipf-Tabelle (aus 4c hierher, weil nur `describe` sie braucht) |
| **4e** Umstellung | ✅ 2026-10-05 (Fertig-Bedingungen der Aufräum-Spec erfüllt, zuletzt die Selbstnutzungsprobe `brain catalog` und `brain status` an der echten Registry, `parity/stufe-4e.md` Messung 13; die Folgezeilen der Roadmap hängen nicht an 4e) | Drei Stücke (Nachtrag #25, Spec `2026-09-28-loomux-stufe-4e-design.md`): ein lesender Befehl `loomux area check <pfad>`, der vor dem Umzug der alten Manifeste je Schlüssel zeigt, was die heutigen Leser annehmen, anderswo lesen oder ignorieren, und welche Datei sie ablehnen (Nachtrag #24, Klasse a, korrigiert durch #25); statt der Checkliste des Menschen der Ablauf aus Nachtrag #26 (Spec `2026-09-29-loomux-stufe-4e-umstellung-vorbereitet-design.md`): Der Agent bereitet je Projekt eine vollständige `.loomux/config.toml`, die geänderte Registry und ein `apply.sh` vor, ein Mensch ruft es auf, und von der Checkliste bleiben Block 1 bis 3 und der Rauchtest. Zwei Piloten sind umgestellt und gemessen (`ecoflow` am 2026-09-29, `space` am 2026-09-30); die Welle lief nach der Veröffentlichung der Lane-Probation (Nachtrag #27, v6.1.0) am 2026-10-01/02 über `iam_backend`, `iam_frontend`, `iam_workers`, `iam_wiki` (nur als Wiki-Bereich) und den Vault, je vorher und nachher gemessen (`parity/stufe-4e.md`, Messung 10); **entschieden am 2026-10-01 (Nutzer):** `ultra-brain` und `ultraloom` werden nicht umgestellt, beide werden nach der Migration gelöscht; nach der Welle ein Aufräum-PR, der die Rückfälle entfernt und einen erschlagenen Tausch beim Lesen auflöst; der Remote für `brain-knowledge` steht seit dem 2026-09-28 |
| **4f** Altverweise | offen; vorgeschlagen 2026-09-27, freigegeben 2026-09-28 (Nachtrag #24) | In loomux verweist nichts mehr auf `ultraloom` und `ultra-brain`, außer in drei benannten Ausnahmen; zuerst eine Gleichheitsprüfung gegen die Ursprungsrepos, dann die Klassen unter „#24 im Einzelnen“, am Ende ein Tor-Test gegen neue Verweise. Hängt an 4e |

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
| **G4a** | ✅ 2026-09-22 | Die Navigation: `graph callers`, `skeleton`, `grep`, `map`, `stats` und die MCP-Werkzeuge `graph_file_api`, `graph_trace_calls`, `graph_find_all`, `graph_repo_map` (`2026-09-22-loomux-code-g4-delta.md`, Paritätsakte `parity/code-g4.md`) |
| **G4b** | ✅ 2026-09-23 | Der Blast-Radius eines git-Diffs: `internal/code/diff`, `blast.Radius`, `graph blast` und `graph_blast`, `check graph-fresh` und `check blast-audit`, die Art `graph` in `[verify]` (Profilvorgabe `precommit`), der Blast-Monitor im Post-Edit-Hook (`2026-09-23-loomux-code-g4b-delta.md`, Paritätsakte `parity/code-g4.md` §4) |
| **G4c** | ✅ 2026-09-28 | Der Stop-Hook mit Blast-Logik: `graph` in der Profilvorgabe `stop`, der Bereich „alles, was git nicht ignoriert, gegen HEAD“ über eine Indexkopie mit `add -A`, die die Lane als `GIT_INDEX_FILE` bekommt; ein Befund hält die Runde wie einen Commit (`2026-09-25-loomux-code-g4c-delta.md`, E9–E11) |
| **G5a** | ✅ 2026-09-26 (Spec `2026-09-26-loomux-code-g5-design.md`, Akte `parity/code-g5.md`) | Extraktor-Schnittstelle, gemeinsamer Tree-sitter-Kern auf `gotreesitter` (reines Go) statt `wazero`, Cache je Datei, CGo-Freiheitstor; dazu Python |
| **G5b** | offen | TypeScript/TSX |
| **G5c** | offen | GDScript |
| **G5d** | offen | C++, erst nach einer Recall-Prüfung gegen die C-Laufzeit |

**Nachtrag 2026-09-26:** G5 läuft nicht über `wazero` und WebAssembly. Eine Probe fand mit
`github.com/odvcencio/gotreesitter` eine Tree-sitter-Laufzeit in reinem Go, die ohne WASM-Build,
AOT-Cache und C-Werkzeugkette auskommt; sie kostet +12 MB Binary und ~2 ms Init je Aufruf (Zahlen
und Begründung in der G5-Spec, §2 und §10). Die Sprachen setzte der Nutzer neu: Python,
TypeScript/TSX, GDScript, C++, von einfach nach schwer; Go bleibt nativ.

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
lückenlos nachnummeriert — die Reihenfolge ist dieselbe. **Nachtrag 2026-09-23:** G4 zerfiel in
G4a und G4b, beide fertig; der Stop-Hook mit Blast-Logik, den der G4b-Nachtrag aus G4b herausnahm
(E4′), ist die neue Stufe G4c und erbt Prio 2. **Nachtrag 2026-09-26:** Flow
ist eine eigene Spur neben der Fusion, wie der Code-Graph: Teil A (Umzug der
Laufzeit, siehe `2026-09-26-loomux-flow-a-design.md`) zieht nichts aus 4c, 4d
oder 4e ein, und 4e wartet ohnehin auf einen Remote für `brain-knowledge`. Die
Prios der übrigen Zeilen bleiben; Flow behält 4 als Rang unter den offenen
Folgeprojekten, beginnt aber jetzt.

Wörtlich steht in dieser Spec nur „2c hängt an 2a“. Jede andere Abhängigkeit
ist aus Spec und Code abgeleitet, der Grund steht in der Zeile.

| Prio | Stufe | Hängt ab von | Warum hier |
|---|---|---|---|
| — | **2c** `stop`, `subagent-*`, Antigravity-Adapter | 2a ✅ | ✅ Fertig (2026-09-22). Die Claude-Seite bringt den Lint des Wiki-Bündels als Lane `lint/wiki` ans Rundenende, die Drift-Regel bleibt bei `loomux wiki-gate`. Antigravity-Messung durchgeführt und Adapter implementiert. Voraussetzung für `init` in Stufe 4 |
| — | **2b** commit-msg | keine genannt | ✅ Fertig (2026-09-19). Statt `migrate`, das wegfällt (#19), fragt `init` in Stufe 4 die Commit-Sprache ab und schreibt `[commit].language` (Nachtrag #13) |
| — | **3** Brain-Pflege (3a, 3b, 3c) | 1b-1 ✅, 1b-2 ✅ | ✅ Fertig (3a 2026-09-22, 3b und 3c 2026-09-23); bis dahin Prio 1. Die größte Stufe, und sie hängt weder an 2b noch an 2c — darum lief sie parallel zu beiden. 3a ist fertig (2026-09-22); 3b ist fertig (2026-09-23), samt Selbstnutzung; 3c ist fertig (2026-09-23), der Upkeep ruft `reconcile` aus 3a. Upkeep läuft in `serve`; die Brain-Skills aus Stufe 4 rufen `brain check` (Nachtrag #1, #7) |
| — | **G4a** Navigation, **G4b** Diff-Blast, Art `graph`, Edit-Monitor | G3 ✅; G4b an G4a ✅ | ✅ Fertig (G4a 2026-09-22, G4b 2026-09-23). W3 wartete auf G4a, W4 auf G4b |
| — | **G4c** Stop-Hook mit Blast-Logik | G4b ✅ | ✅ Fertig (2026-09-28). Neben der Fusion gebaut; loomux prüft sich an jedem Rundenende selbst und bekommt den Blast dort. Nichts wartet darauf |
| 3 | **4** `config`, `init`, Modell, `convert`/`fetch` | 2b ✅, 2c ✅, 3 ✅ (3a, 3b, 3c) | Ohne Stufe 4 bleiben die alten Repos im Dienst. In sich 4a-1 → 4a-2 → 4c-1 → 4d, dann 4e, dann 4f (Nachtrag #24, freigegeben 2026-09-28); 4c-2 daneben, ohne dass etwas auf sie wartet, nach Regel 2 (Selbstnutzung). 4a-1, 4a-2 und 4c-2 sind fertig (2026-09-28), 4c-1 am 2026-09-29; 4d ist fertig (2026-09-27); `feat/self-update` ist gemergt. Den Remote für `brain-knowledge`, den die Umstellung der Wirte braucht, gibt es seit dem 2026-09-28 (siehe „Umstellung der Wirte nach Stufe 4“) |
| 4 | Folgeprojekt **Flow**, eigene Spur neben der Fusion (Nachtrag 2026-09-26), zerlegt in A (Umzug der Laufzeit), A2 (Flows über MCP), B (Modellzugang), C (Bausteine), D (Entwicklungszyklus als Default-Flow) | ulflow M1 (Zweig `feature/agent-harness`, nicht gemergt) | Kein Wirt nutzt heute Flows (siehe „Befunde“); A hängt an keiner offenen Stufe und beginnt deshalb neben Stufe 4 |
| 5 | **W1–W5** (Web-OS-Spec) | W1 an 1b-2 ✅; W2 an W1; W3 an W1 und G4a ✅; W4 an G4b ✅ und 4; W5 an W1 und Flow | Folgeprojekt |
| 6 | **G5** Mehrsprachigkeit (G5a → G5b → G5c → G5d) | G4b ✅; jede Teilstufe an der vorigen | Nichts wartet darauf; der Nutzer zog G5a am 2026-09-26 vor, neben die übrigen Prios |

### Nachgetragen: was bisher keine Stufe hatte

Eine Durchsicht beider Quellrepos am 2026-09-19, am Code und nicht an der
Doku, fand sechzehn Stellen, die weder in loomux gebaut noch in dieser Spec
oder einer Paritätsakte genannt waren — gegen den Grundsatz unter „Ziel“, dass
nichts ersatzlos wegfällt, ohne in einer Liste zu stehen. Eine siebzehnte fand
am selben Tag die Spec der Stufe 3 (`2026-09-19-loomux-stufe-3-design.md`,
„Befunde“), eine achtzehnte am 2026-09-23 der Plan von 3c, die neunzehnte am
2026-09-24 die Spec der Stufe 4 (der Wegfall von `migrate`), die zwanzigste
und einundzwanzigste am 2026-09-24 eine Bestandsliste über alle lokalen
Zweige beider Repos, nicht nur `master`, die zweiundzwanzigste dieselbe
Durchsicht unter den ungetrackten Dateien, die dreiundzwanzigste am 2026-09-25
die Messung von Antigravity beim Bau von 4a-2 und die vierundzwanzigste am
2026-09-27 eine Vorgabe des Nutzers samt Inventur der Verweise auf die
Altprojekte. Die Zuordnung ist ein **Vorschlag**; die Spalte „Freigabe“ füllt
der Nutzer, erst dann gilt sie.

| # | Quelle | Stelle | Vorschlag | Begründung | Freigabe |
|---|---|---|---|---|---|
| 1 | ultra-brain | `brain check file\|bundle\|all` mit den Achsen OKF, Hausregeln, Föderation (`pkg/check/{okf,house,run}`, `cmd/brain/main.go:937`) | Stufe 3 | Nur das Basispaket `check` ist umgezogen; `internal/brain/wiki/lint.go` verweist die Regeln (`wrong-direction`, `long-planned`, `no-sources`, `log-date-form` …) an Checks, die es in loomux nicht gibt. Die Brain-Skills rufen `brain check` | freigegeben 2026-09-19, Stufe 3c; gebaut 2026-09-23 |
| 2 | ultra-brain | `lint` ohne Pfad und mit `--scope all` (`src/brain/cli.py:562`) | Stufe 3, mit #1 | `loomux lint` verlangt genau eine Datei. Den Lint über das ganze Bündel hat heute nur `wiki-gate`, und das nur zusammen mit der Driftprüfung | freigegeben 2026-09-19, Stufe 3c; gebaut 2026-09-23 |
| 3 | ultra-brain | `embed` als Befehl (`src/brain/cli.py:467`) | Stufe 3 | Gehört zu `reindex`; der Datenumzug (Punkt 3) rechnet schon mit „`reindex` + `embed` einmal je Bereich“. In loomux gibt es nur `QmdMcpPort.Embed` ohne Befehl | freigegeben 2026-09-19, Stufe 3a; gebaut 2026-09-22 mit #17 |
| 4 | ultra-brain | `brain layout` und `layout.json` (`pkg/layout`, `cmd/brain/main.go:340`) | Folgeprojekt Web-Migration | Nur die Web-App liest die Orte aus `layout.json` (`web/src/canvas/cosmos.test.ts`) | freigegeben 2026-09-28, Folgeprojekt Web; loomux schreibt keine `layout.json` (auch `parity/artefakte-nach-lebensdauer.md` #1 nennt sie nur als fehlend) |
| 5 | ultra-brain | `hook install\|status\|remove`: der post-merge-Hook in einwilligenden Repos (`src/brain/cli.py:625`) | Stufe 4 (`loomux init`) | Hooks schreibt `init`; der Hook selbst speist `merge-events` aus Stufe 3. Der Abschnitt „Git-Hooks“ kennt ihn noch nicht | freigegeben 2026-09-23, Stufe 4a-2, als `loomux merge-hook`; der Hook ruft das Binary statt eingebackener Pfade (2026-09-24); gebaut 2026-09-24 mit 4a-2 |
| 6 | ultra-brain | `daemon start\|run --backbone`, `--no-local`, `--no-cloud` (`src/brain/cli.py:673`) | Abweichung 1b-2 nachtragen | `serve` kennt nur `--foreground`, das Backbone ist kein Flag, sondern `[search] backbone` der rechnerweiten `config.toml` (Vorgabe CUDA, entschieden 2026-09-27; ein `QMD_LLAMA_GPU` oder `QMD_FORCE_CPU` des Nutzers gewinnt). `serve` hält immer beide Kanäle, `--no-local` und `--no-cloud` haben nichts zu schalten | freigegeben 2026-09-28 als Wegfall der drei Flags; die Abweichung steht in `parity/stufe-1b-2.md` |
| 7 | ultra-brain | Die Skills `brain-ingest`, `brain-land`, `brain-research`, `brain-review`, `brain-wiki-plan` (`.claude/skills/`) | Stufe 4 (`loomux init`), Befehle nach #1 umgeschrieben | Keine davon ist eine Review-Suite aus W4; `init` legt Skills in die Wirte | freigegeben 2026-09-23, Stufe 4a-2; dazu ins Englische übersetzt; gebaut 2026-09-24 mit 4a-2 |
| 8 | ultraloom | Skill `verify-until-green` (`templates/skills/…SKILL.md.tmpl`) | Stufe 4 (`loomux init`) | Ruft fest `uv run ultraloom check all`; wird zu `loomux check all`. Der Flow dahinter bleibt Folgeprojekt 1 | freigegeben 2026-09-23, Stufe 4a-2; gebaut 2026-09-24 mit 4a-2 |
| 9 | ultraloom | Skill `session-handover` | Stufe 4 (`loomux init`) | Ohne Abhängigkeit auf einen Befehl; zieht mit den übrigen Skills | Wegfall aus `init`, freigegeben 2026-09-24: die globale Fassung des Nutzers ist eine andere, eine Projektkopie würde sie verdecken |
| 10 | ultraloom | Die erzeugte `AGENTS.md` (`internal/render/render.go:109`, `templates/AGENTS.md.tmpl`) | Stufe 4 (`loomux init`) | `render`/`install` ziehen dort um; die Vorlage war nicht genannt | freigegeben 2026-09-23, Stufe 4a-2; nur wenn keine `AGENTS.md` da ist; gebaut 2026-09-24 mit 4a-2 |
| 11 | ultraloom | `.gitignore`-Einträge des Installers (`cmd/init/run.go:381`) | Stufe 4 (`loomux init`) | Mit den Pfaden von loomux: `.loomux/state/` statt `.ultraloom/hooks/` | freigegeben 2026-09-23, Stufe 4a-2; gebaut 2026-09-24 mit 4a-2 |
| 12 | ultraloom | `[relevance]` (`internal/render/templates/config.toml.tmpl:22`, `cmd/init/run.go:678`) | Wegfall | Der Installer schreibt ihn, aber kein Hook liest ihn, schon in ultraloom nicht. Die Presets je Stack decken die Absicht ab. `loomux migrate` lässt ihn fallen und sagt es | freigegeben 2026-09-23; da `migrate` wegfällt (#19), kennt `init` ihn einfach nicht |
| 13 | ultraloom | `[project].commit_language` (`cmd/init/run.go:507`) | Stufe 2b, `migrate` in Stufe 4 | Nur `ulinit` liest ihn als Antwortvorgabe; die Prüfung liest `[commit].language`. `migrate` übersetzt ihn dorthin | freigegeben 2026-09-23, Stufe 4a-2: statt `migrate` fragt `init` die Commit-Sprache ab |
| 14 | ultraloom | `[agent].settings`, `[agent].mcp_servers` (`config.py:219`) | Folgeprojekt 1 (Flow-Migration) | Sie steuern den Aufruf von `claude -p` in Flows. Die Tabelle „Wegfall aus dem alten Schema“ der 2a-Spec streicht nur `cli_path` | freigegeben 2026-09-26: `mcp_servers` mit Flow A, `settings` mit Flow B (`2026-09-26-loomux-flow-a-design.md`) |
| 15 | ultraloom, ultra-brain | `scripts/install.ps1` und `install.sh`: Bauen in `~/go/bin`; ebenso `scripts/install.{sh,ps1}` von ultra-brain (#32) | Stufe 4 (`loomux init`) | Release-Archive gibt es, aber keinen Weg, das Binary auf den `PATH` zu legen | freigegeben 2026-09-24, Stufe 4a-2: `init` holt das neueste Release an den kanonischen Ort `%LOCALAPPDATA%\loomux\bin\loomux.exe` der Self-Update-Spec; die Einträge rufen es über `${LOCALAPPDATA}`, nicht über den `PATH`; gebaut 2026-09-24 mit 4a-2 |
| 16 | ultraloom | `ulinit --detect-only` (`cmd/init/main.go:44`) | Stufe 4, als `loomux init --detect-only` | `internal/detect` ist da, nur ohne Befehl | freigegeben 2026-09-23, Stufe 4a-2; gebaut 2026-09-24 mit 4a-2 |
| 17 | ultra-brain | `reindex` und `embed` als Befehle (`ultra-brain/pkg/index`, `src/brain/cli.py:463,467`); `embed` allein ist #3 | Stufe 3a | Der Auffangdurchgang koppelt `reconcile` an `reindex` („`reconcile` auch als Durchgang vor `reindex`“ nennt einen Befehl, den es in loomux nicht gab), und `embed` ist ohne `reindex` gegenstandslos | freigegeben 2026-09-19, Stufe 3a; gebaut 2026-09-22 |
| 18 | ultra-brain | `brain check code` (`pkg/check/code`, die Lanes aus `[check].lanes`) | Wegfall | Die Prüfkette aus 2a (`[verify]`, Presets je Stack, `loomux check`) fährt dieselben Lanes samt Reihenfolge und Coverage-Tor; eine zweite Lane-Konfiguration stünde daneben. `Manifest.Lanes` bleibt geparst, bis `loomux migrate` (Stufe 4) es nach `[verify]` überträgt. Gefunden beim Planen von 3c | freigegeben 2026-09-23. Da `migrate` wegfällt (#19), wird `Manifest.Lanes` in 4e gelöscht statt übertragen; `[check].lanes` steht nur in `ultra-brain`. Erledigt mit dem Aufräum-PR (Nachtrag #28): `Manifest.Lanes` ist gelöscht |
| 19 | — | `loomux migrate`, der einmalige Übersetzer für Maschinenzustand und Wirtskonfiguration (Punkte 2 und 4 unter „Datenumzug“) | Wegfall | Die alten Werkzeuge hat nur der Nutzer benutzt, auf einem Rechner und in vier Wirten. Der Maschinenzustand liegt durch die Selbstnutzung seit 3a schon in `%LOCALAPPDATA%\loomux`; die Wirte richtet `init` neu ein. Was bleibt, ist die Checkliste von 4e | freigegeben 2026-09-24 |
| 20 | ultra-brain | Der nie gemergte Zweig `feature/artefakte-nach-lebensdauer` (44 Commits vor `master`, Stand 2026-09-12; Spec `2026-09-11-artefakte-nach-lebensdauer-design.md` auf `docs/artefakte-nach-lebensdauer`): `graph.json` und `layout.json` ins Zustandsverzeichnis, nur der Wurzelkatalog, ein Register mit Aliasen, in das ein Indexlauf nur Geburten und Umbenennungen schreibt, die Auflösung eines Bereichs aus einem verknüpften Worktree, dazu Fehlerbehebungen am Indexlauf (ein Worktree-Lauf schreibt nicht in die Dateien des Hauptcheckouts, ein Bereichspfad mit abschließendem Trenner, frühere Indexausgabe wird aus dem Baum geräumt) | Vor 4e: den Zweig Commit für Commit gegen `internal/brain/index` und die Registerschreibung lesen; was loomux nicht schon anders löst, wird ein Fix-PR oder eine eigene Zeile, der Rest fällt mit Begründung weg | Die Paritätsfälle von 3a sind gegen `master` aufgezeichnet, das Verhalten des Zweigs hat keiner gesehen. 3a hat dasselbe Problem nur mit `[index] include` und einer offenen Versionierungsfrage behandelt (`parity/stufe-3a.md`). Durchsicht am 2026-09-27 (`parity/artefakte-nach-lebensdauer.md`): zwei Themen löst loomux schon, eines fällt weg, keines braucht einen Fix-PR, sechs brauchen eine Entscheidung des Nutzers | freigegeben 2026-09-24, vor 4e |
| 21 | ultraloom | Wiki-Flottenstandard Stufe 2, nicht fertig: auf `claude/wiki-stufe-2` liegen nur `560709c` und `be0011d` (`internal/agenthooks/merge.go` mit Tests), Task 2 von 4 des Plans `2026-09-13-wiki-flottenstandard-stufe-2.md` (nur auf ultraloom-`master`, nicht gepusht). Die Regeln: `.agents/hooks.json` ist eine Map benannter Gruppen, der Name ist die Identität; das Werkzeug besitzt genau eine Gruppe und kodiert jede andere byte-treu aus dem gelesenen JSON neu; kein Besitzerfeld im Hook-Objekt, weil ungemessen ist, ob Antigravity ein unbekanntes Feld duldet; eine fremde Gruppe mit demselben Kommando wird gemeldet, nie repariert; eine Wurzel `null` wird abgelehnt wie jede Nicht-Objekt-Wurzel, mit dem Dateinamen in der Meldung. Task 3 (`ulinit` schreibt die Gruppe) und Task 4 (Nachweis) fehlen | Stufe 4a-2: die Regeln in den Abschnitt „Host-Einträge“ der Stufe-4-Spec, der bisher nur „Fremde Einträge bleiben stehen“ sagt; `merge.go` samt Tests zieht als Ausgangspunkt für den Schreiber von `.agents/hooks.json` um, der Gruppenname wird `loomux`-eigen | Die Zeile „Wiki-Flottenstandard“ unter „Laufende Vorhaben“ setzte voraus, dass die Stufe in ultraloom fertig wird; sie ruht seit dem 2026-09-13, und `init` ist der einzige Ort, an dem sie noch landen kann | freigegeben 2026-09-24, Stufe 4a-2; gebaut 2026-09-25 in hostfile (kein Umzug von agenthooks) |
| 22 | ultraloom | Die Spec `2026-08-24-multi-provider-llm-design.md`, nie committet, nur als ungetrackte Datei im Hauptcheckout von ultraloom; jetzt unverändert unter `specs-ul/`. Agenten-Flows unabhängig vom Anbieter: ein `Model`-Port mit Adaptern für Gemini (`google-genai`) und Claude (`anthropic`), ein eigener Werkzeug-Ausführer (`Read`, `Edit`, `Write`, `Glob`, `Grep`, `Bash`) mit Profilen und dem Schutz von `[verify].tests`, `[agent].provider`/`model`, `--provider`/`--model` | Folgeprojekt Flow, als Eingang seiner Spec | Geschrieben für die Python-Flows von ultraloom, nicht für ulflow. Sie überschneidet sich mit M2 (`claude -p`, `agy -p`), das dieselben zwei Anbieter über ihre CLIs und damit über die Abos anspricht; welcher Weg gilt oder ob beide, entscheidet die Flow-Spec | freigegeben 2026-09-24, Folgeprojekt Flow |
| 23 | — | Das Antwortprotokoll der Hooks gegenüber Antigravity und die Form von `.agents/hooks.json` | Stufe 4a-2: ein Adapter `hosts.Answer`, den `loomux hook` einmal für jeden Austrittspfad nach dem Lesen der Flags ruft, eine Panik eingeschlossen; ein fehlendes oder unbekanntes `--host` endet ohne Adapter mit dem Code eines kaputten Aufrufs. Für Antigravity: `pre-tool-use` verweigert mit Exit 2, `post-tool-use` warnt mit Exit 2 und schreibt bei Exit 0 seinen Kontext über `hosts.WriteContext` als `injectSteps`, den `hosts.Answer` durchreicht, ein gehaltener Stop wird `{"decision":"continue","reason":…}` auf stdout mit Exit 0, jeder andere Code ungleich 0 endet mit 0, außer bei `pre-tool-use`, das jeden Code durchreicht; ein unbekanntes Ereignis bleibt Exit 2. `session-start` auf `PreInvocation` meldet sich beim ersten `invocationNum` (Zahl oder Dezimal-String; die Zählung beginnt bei 0, ein späterer Aufruf ist `invocationNum > 0`) und belebt die Sitzung nur dann; ein späterer sagt nichts. `run_command` (Argument `CommandLine` oder `command_line`), `send_command_input` und `manage_task` (Aktion `send_input`, Argument `Input`) stehen in einer Werkzeugtabelle und laufen durch die Befehlsregeln: jede Zeile unter einem ihrer Argumentnamen, ohne Rücksicht auf die Schreibung, wird geprüft, gleich welche `Action` der Aufruf nennt; die stillen Aktionen `list`, `status` und `kill` von `manage_task` entschuldigen nur das Fehlen einer Zeile, und nur, wenn jeder Schlüssel `action` in jeder Schreibung eine von ihnen als String nennt, sonst wird ohne Zeile verweigert; was `send_command_input` und `manage_task` tippen, geht nur als ganze Zeilen ohne Steuerzeichen und ohne Zeilenfortsetzung (`\` oder `` ` `` am Zeilenende) durch. Ein Wert unter einem Zeilenschlüssel, der kein String ist, macht den Aufruf unprüfbar und wird verweigert; ein leerer String trägt keine Zeile. Ein Aufruf ohne Werkzeugnamen wird verweigert. Post-Edit liest die Ziele aus dem `toolCall` des PostToolUse; keine Ablage. `PreInvocation` und `Stop` stehen flach in `.agents/hooks.json` | Gemessen mit agy 1.2.8 und 1.2.11 am 2026-09-25 (`parity/stufe-4a-2.md`): gruppierte `Stop`/`PreInvocation` lassen agy die ganze Datei verwerfen, der Wächter lädt dann nicht; PostToolUse trägt `toolCall` entgegen dem Leitfaden; Exit 2 in PostToolUse bricht nicht ab; `continue` hält den Stop. Nachgemessen am 2026-09-25 mit agy 1.2.11: in einen offenen Befehl tippt agy über `manage_task` mit `send_input` und `Input`, beenden kommt als `manage_task` mit `kill`, nicht über `send_command_input` (`parity/stufe-4a-2.md`). Gemessen am 2026-09-27 mit agy 1.2.11 (`parity/stufe-4a-2.md`): `PreInvocation` trägt `invocationNum` als JSON-Zahl, 0, 1, 2, 3 je Modellaufruf, die Zählung beginnt also bei 0; zwei Blöcke der Gruppe `loomux` unter `PreToolUse` mit verschiedenen Matchern laden, und agy ruft den Hook für ein Werkzeug des zweiten Blocks. Nachgemessen am 2026-09-28 mit agy 1.2.12 (`parity/stufe-4a-2.md`): alles oben gilt weiter, und agy liest `injectSteps` auf PostToolUse; `Input` kam in zwei Läufen desselben Tages einmal ohne Zeilenende (`"hello"`, als Fragment verweigert) und einmal mit (die getippte Push-Zeile erreichte die Befehlsregeln), mit demselben Modell; ob ein Zeilenende mitkommt, entscheidet der einzelne Aufruf. Ungemessen: der Argumentname von `send_command_input`, die String-Form eines 64-Bit-`invocationNum` und die Breite des Felds, und ob agy an ein `Input` ohne Zeilenende selbst ein Enter hängt | vorgeschlagen 2026-09-25, nachgeführt 2026-09-27 nach dem Code-Review; freigegeben 2026-09-28 nach der Nachmessung mit agy 1.2.12, mit dem Durchreichen des Post-Edit-Kontexts; ein `Input` ohne Zeilenende bleibt verweigert |
| 24 | — | Verweise auf die Altprojekte `ultraloom` und `ultra-brain` in loomux: Inventur am 2026-09-27 über `origin/master` (v4.0.0), 3.829 Zeilen in 1.316 Dateien, davon rund 1.000 Aufzeichnungen unter `testdata/cases/*-source` | Neue Teilstufe **4f** nach 4e: jede Klasse unter „#24 im Einzelnen“ verschwindet, wird ohne die Namen neu gefasst oder bleibt nach ausdrücklicher Entscheidung | Vorgabe des Nutzers vom 2026-09-27: „Am Ende der Migration soll in loomux kein Verweis mehr auf die Altprojekte sein.“ Bis 4e tragen einige Verweise Funktion (Lese-Rückfälle, Erkennung alter Host-Einträge), danach keiner mehr. Offen ist, ob „Ende der Migration“ 4e meint oder erst das Ende der Folgeprojekte Flow und Web, die sich noch aus den Altprojekten speisen | Ziel vom Nutzer gesetzt am 2026-09-27; freigegeben 2026-09-28: 4f nach 4e, Prio 3, mit der Behandlung je Klasse unter „#24 im Einzelnen“. „Ende der Migration“ meint 4e; die Archive gehen mit dem letzten Folgeprojekt |
| 25 | — | Die Stufe 4e ist keine reine Checkliste mehr, und `area check` bekommt andere Klassen als in #24 | Drei Stücke: **A** `loomux area check` (nur lesend, meldet je Schlüssel angenommen, ignoriert oder abgelehnt; der Leser bleibt unverändert; die Klassen kommen aus `DeclarationKeys` und dem Schema), **B** die Checkliste (dazu neu `qmd-collections.json` kopieren, wenn im neuen Verzeichnis keine liegt, und `[index]` für `project/loomux`), **C** der Aufräum-PR nach der Checkliste. Erschlagener Tausch (`stufe-3a.md`, „der abwesende Bereichsordner“): `config.ResolvedAreaDir` liefert `<ziel>.loomux-aside`, wenn das Ziel fehlt und das Aside da ist; der Leser schreibt nichts, geheilt wird weiter vom Indexlauf. Ein `lock.Recover` im Lesepfad wäre ein Rennen mit einem lebenden `ReplaceDir`, das kein Lock nimmt. Restfenster bleibt: ein Leser, der den Aside-Pfad aufgelöst hat und dann ins Leere öffnet, trifft `ErrNoManifest` wie heute, nur im engeren Fenster | Am 2026-09-28 gemessen: `ReadDeclaration` gibt für ein Manifest mit `[llm.local]`, einer unbekannten Tabelle und einem unbekannten Top-Level-Schlüssel ein Manifest ohne Fehler zurück, ebenso für sechs echte Manifeste; ein Umbenennen gibt es im Code nicht. Ohne `area check` ginge ein ignorierter Schlüssel beim Umlegen von Hand still verloren. Die Aside-Variante wählte der Nutzer am 2026-09-28 (Variante 1: Heilung im Lesepfad); die Ausführung als Auflösen statt Zurückschieben folgt aus dem Rennen mit `ReplaceDir` | Entwurf 2026-09-28; freigegeben 2026-09-30 mit der Spec `2026-09-28-loomux-stufe-4e-design.md`; Stück B ersetzt und Stück C geändert durch #26. Die Aside-Auflösung ist überholt durch #28: `#Obsidian/AI` ist aus der Registry genommen, die Auflösung wird nicht gebaut |
| 26 | — | Die Umstellung der Projekte auf loomux ist vom LLM vorbereitet, und das Wiki liegt immer im Projekt; die read-only-Bereiche der Projekte entfallen bis auf `#Obsidian/AI`, das nicht umgestellt wird | Stück B der Stufe 4e (Checkliste) wird durch einen Ablauf ersetzt, in dem der Agent liest, im Scratchpad vorbereitet (Übersetzung des alten Manifests in einen Vorschlag für `.loomux/config.toml`, geänderte Registry, bereinigte `settings.json`, ein `apply.sh`) und danach nur lesend prüft; jede Schreibaktion steht im `apply.sh`, das der Nutzer mit einer Zeile aufruft. Erst ein Pilot (`ecoflow`), dann eine Welle über die übrigen Projekte, darunter `iam_backend`, `iam_frontend` und `iam_workers`, die die Registry nicht kennt (acht Projekte, ohne `#Obsidian/AI`, dazu der Vault `brain-knowledge` mit eigener Form: Hooks und Guards, kein Wiki-Umzug; zwei Piloten, `ecoflow` und `space`). Vor und nach jeder Umstellung Benchmarks (kalt, fünf Warmläufe) für alles Alte, alles Neue und alles Weggefallene, ein Bericht je Projekt und ein anonymisierter in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`. Die Aside-Auflösung aus #25 (Stück C) bleibt, weil `#Obsidian/AI` als einziger read-only-Bereich bestehen bleibt. **Entschieden am 2026-09-30:** Beide Piloten sind umgestellt und gemessen (`ecoflow` am 2026-09-29, `space` am 2026-09-30; `parity/stufe-4e.md`, Messung 8 und 9). Die Welle über die übrigen sechs Projekte und den Vault beginnt erst, wenn die Lane-Probation freigegeben ist (Spec `2026-09-30-loomux-lane-probation-design.md`, eigener Pull Request): Ein umgestelltes Projekt träfe sonst auf Lanes, die bei ihm noch nie grün waren, wie das Stop-Tor und `wiki lint` beim Piloten `space`. Der Aufräum-PR (Stück C) wartet auf die Welle | Vorgabe des Nutzers vom 2026-09-29 (Hooks, Brain und Graph in allen Projekten, alle Guards aktiv, Wiki im Projekt, das LLM richtet ein und bereitet Handschritte vor, Benchmarks alt gegen neu). Gemessen am selben Tag: der Wächter verweigert dem Agenten fünf der neun Projekte und die Registry, deshalb steht alles Schreibende im `apply.sh`; die alten Werkzeuge liegen noch in `~/go/bin`, die Baseline ist messbar. Nachtrag #19 gilt weiter für einen Befehl: das LLM erzeugt Vorschläge, kein Übersetzer schreibt | Entwurf 2026-09-29; freigegeben 2026-09-30 mit der Spec `2026-09-29-loomux-stufe-4e-umstellung-vorbereitet-design.md`. Das „bleibt, weil `#Obsidian/AI` …“ ist überholt durch #28: `#Obsidian/AI` ist aus der Registry genommen, die Auflösung wird nicht gebaut |
| 27 | — | Ein frisch eingerichtetes Tor ist vom ersten Tag an scharf über Code, den bisher kein Tor geprüft hat: Pilot `ecoflow` (2026-09-29, der Umstellungs-Commit ging nur mit `--no-verify`), Pilot `space` (2026-09-30, der Stop-Hook hielt jede Sitzung wegen 790 Befunden von `wiki lint`) | Schonfrist je Lane (Spec `2026-09-30-loomux-lane-probation-design.md`): eine versionierte `.loomux/armed.toml` nennt die scharfen Lanes; ohne Datei ist jede Lane scharf und jede Ausgabe wie bisher, mit Datei warnt jede Lane, die nicht darin steht. Scharf stellt nur `loomux check precommit --arm` nach einem insgesamt grünen Lauf oder ein Mensch mit `loomux gate arm`. Keine neue Stufe und kein eigener Rang: **Reihenfolge** erst die vorbereitete Umstellung (#26) mergen und veröffentlichen (geschehen mit v6.0.0 am 2026-10-01), dann die Schonfrist mergen und veröffentlichen, dann die Welle der Stufe 4e | Ohne sie zwingt jede Umstellung zu `--no-verify` oder zum Abschalten des Stop-Hooks, und beides lässt auch die Lanes fallen, die schon grün sind | freigegeben 2026-09-30 (im Gespräch am 2026-09-29 und 2026-09-30 entschieden) |
| 28 | — | Der Aufräum-PR der Stufe 4e: im Code stehen nach der Welle noch die Lese-Rückfälle aus der Zeit, in der ultra-brain neben loomux lief (`LegacyBrainDirUntilStage3` samt `LOOMUX_LEGACY_BRAIN_DIR`, `ReadAreaManifestUntilStage4` mit den Manifestnamen `.ultra-brain/config.toml` und `.brain.toml`, `Manifest.Lanes`), und Ratschläge, die Befehle der Python-Referenz nennen | **Eine Aside-Auflösung wird nicht gebaut** (Entscheidung 6 der Spec): `#Obsidian/AI` ist aus der Registry genommen, ebenso `ultra-brain` und `ultraloom`; die Regel „read-only → `<state>/areas/<scope>`“ bleibt wegen `dev bench search`. Ein Arbeitsbereich ohne `[area]` ist kein brain-Bereich (Entscheidung 9, PR #67); `ReadAreaDeclaration` antwortet für ihn mit `ErrNoArea`, `IsUndeclared` fasst das mit `ErrNoManifest` zusammen, außer wenn nur ein Altmanifest daliegt (`ErrOldManifest`: Meldung samt Hinweis, Entscheidung 8). **Auflagen aus `parity/stufe-3a.md`** (Entscheidung 11): die Ratschläge trägt dieser PR (sie nennen `loomux reindex\|reconcile\|embed`); das Fegen der Asides und die read-only-Deklarationen entfallen mit den read-only-Bereichen; das Mitnehmen von `merge-events.done.tsv` und `qmd-collections.json` ist **ausdrücklich fallengelassen** (Nutzer, 2026-10-03). **Die übrigen Befunde der Welle** (Entscheidung 12) werden Folgezeilen der Roadmap: Befund 1, 2, 4, 5 aus `parity/stufe-4e.md` (Messung 10), dazu der Entscheid des Nutzers (c): `maintenance`, `lint`, `convert` und `reconcile` lassen einen Arbeitsbereich ohne `[area]` aus (eigener PR), und der Re-Import von `dev import-cases` tauscht eine Welt atomar (ein Sperrfehler unter Windows lässt eine halb bestückte Welt liegen, schon vor diesem PR). Dazu zwei Befunde aus dem Abschluss-Review dieses PRs (Nutzer, 2026-10-04): `ReadAreaDeclaration` nimmt jeden `Stat`-Fehler wie eine fehlende Datei, und eine `.loomux/config.toml` ohne `[area]` neben einem Altmanifest bekommt keinen Hinweis. Die Folgezeilen tragen **keine Priorität** (`—`), bis der Nutzer eine vergibt; dieser Nachtrag vergibt keine. **Nutzerentscheid (b):** die sechs Entscheidungen aus `parity/artefakte-nach-lebensdauer.md` blockieren 4e ✅ (die Akte hielt sie bis dahin für nicht blockierend); 4e bleibt `open`, bis sie getroffen sind | Gemessen am 2026-10-03 in zwei Scratch-Worktrees: in den Lauf-Bäumen aller zwölf Fallsuiten liegen genau **zwei** Altmanifeste (ein erwartetes Ergebnis, das loomux nicht schreibt, und eine Welt, die vor dem Lesen endet), also 0 Übersetzer und 0 freigegebene Abweichungen; mit nur `.loomux/config.toml` und ohne Altverzeichnis laufen alle Fallsuiten ohne Rückfall grün; die Ratschläge stehen in **12** aufgezeichneten Fällen (zehn der Spec plus `brain-status/backlog` in 1b-1 und 1b-2 mit `brain embed`), der Re-Import reproduziert beide Korpora byte-gleich | freigegeben 2026-10-03 mit der Spec `2026-10-03-loomux-stufe-4e-aufraeumen-design.md`. Entscheid (c) gebaut am 2026-10-05: `convert`, `fetch` und `lint` lassen einen Arbeitsbereich ohne `[area]` aus; `reconcile` und die Aufholung ließen ihn über `Manifests` schon aus, ein Test hält das fest. Befund 1 gebaut am 2026-10-05 (Variante des Nutzers „nur wenn ausdrücklich verlangt“): Eine Art aus `all` oder einem eingebauten Profil ohne Lane wird ausgelassen, solange eine Lane einer anderen angefragten Art lief; eine Liste von Arten und ein vom Projekt gesetztes Profil halten jede Art weiter zu einer Lane an |
| 29 | — | Die sechs Entscheidungen aus `parity/artefakte-nach-lebensdauer.md` (#1, #2, #3, #6, #8, #9), die nach Nachtrag #28 4e ✅ blockieren | Keine wird vor 4e gebaut; alle werden Roadmap-Zeilen ohne Priorität (`—`). **Kommt:** #2 (der Indexlauf schreibt nur Neuanlagen und Umbenennungen, die Revision zählt die Prüfungen), #3 (eine `doc_id` entfällt nie: Aliase, Grabsteine, `merge=union`), #6 (`reindex` aus einem verknüpften Worktree schreibt nur das Register des Zweigs; hängt an #3) und #9 (a) (die Schreibschranke verweigert `_identities.tsv` in jedem registrierten Bereich, über Write/Edit und Shell). **Vielleicht:** #1 mit #8 (`graph.json` und die Kataloge je Verzeichnis ins Zustandsverzeichnis, im Baum nur der Wurzelkatalog; der Umzug räumt die alten) und #9 (b) (`reindex` committet Registeränderungen über `internal/brain/vcs`) | Nachgelesen am 2026-10-04 gegen `3934fb5b` (v7.0.1): die Befunde der Durchsicht vom 2026-09-27 gelten weiter; `reindex` indexiert ohne `reconcile`, wenn kein Bereich `[layout] review` erklärt (`internal/cli/index.go:110-116`), und Wissen über Worktrees steht nicht nur in `internal/brain/guard` | entschieden vom Nutzer 2026-10-04, einzeln |
| 30 | — | Der Rauchtest der umgestellten Wirte (`parity/stufe-4e.md`, Messung 12): `space` startet neben `loomux hook session-start` weiter ein eigenes `session_start.py`, das vier Dinge tut, die loomux nicht kann — eine Godot-Override-Datei je Worktree (eigenes `user://`), Warnungen bei falschem `core.hooksPath` und bei einem Repository ohne Arbeitsbaum, ein Auszug aus `docs/wiki/log.md` | `session_start.py` bleibt vorerst. Roadmap-Zeile unter „Kommt“ ohne Priorität: `hook session-start` prüft `core.hooksPath` und den Arbeitsbaum und bekommt einen Erweiterungspunkt für Projektschritte; danach kann das Skript schrumpfen. Der Commit-Text von `space` läuft seit `4cbfed6a` (2026-10-04) über `loomux check commit-msg` | Nachgelesen am 2026-10-04: `internal/hooks/hook_session_start.go` schreibt nur den Sitzungszustand, liest weder `core.hooksPath` noch `log.md`, und kein Konfigurationsschlüssel erweitert ihn | entschieden vom Nutzer 2026-10-04 |
| 31 | — | Mit dem Abschluss von 4f endet die Beta; die Zählung beginnt neu, und Betas gibt es künftig als eigene Versionen | Der Release-Neustart gehört zu 4f und bekommt keine eigene Stufe: erst eine Brücke der alten Zählung (neuer Versionsvergleich, `upgrade --beta`/`--version`), dann nach 4f PR C das Löschen aller `v*`-Tags und -Releases, die Kennung `-beta` an den alten Changelog-Einträgen und v1.0.0 als stabiles Release; Betas `X.Y.Z-beta.N` von jedem Zweig, eine Beta-Installation bekommt die nächste Beta (Spec `2026-10-05-loomux-release-neustart-design.md`; 4f selbst: `2026-10-05-loomux-stufe-4f-design.md`) | Vorgabe des Nutzers vom 2026-10-05. Ohne Brücke holt kein installiertes 7.x v1.0.0, weil `selfupdate.Newer` nur Nummern vergleicht und 1.0.0 für älter hält; v1.0.0 bis v1.3.0 sind schon vergeben | Freigegeben 2026-10-05; der Beta-Kanal ist eine Markierung im Zustandsverzeichnis (Variante b) |
| 32 | ultraloom, ultra-brain | Die Inventur der Gleichheitsprüfung (`parity/stufe-4f.md`, Abschnitt 3) fand elf Stellen ohne Zuordnung. Keine davon ist eine verlorene Fähigkeit. | **Wegfall:** `NeighbourWiki` (`internal/detect/edges.go:44`, wird samt Tests in 4f PR B gelöscht), `docs_language` aus `ulinit`, der Alias `brain-mcp wiki gate`, die Verdrahtung auf `claude/eager-engelbart-46d10e` und der Korpus von `claude/scheibe-9b` (vorher eine Probe auf Exit 2 für `reconcile` mit unerwartetem Argument). **Übernehmen:** die drei ulflow-Pläne von `feature/agent-harness` nach `plans-ul/`. **Eingang in Folgeprojekte:** `ultraloom run --no-model` in die Spec von Flow B/M3; Web 7a-2 (`edges-cross.json`) und 7b (Seitenleser, Volltextsuche, Prüfzentrum) in die Eingangsliste des Web-Folgeprojekts; die Regel aus `cdf5dfd` („reconcile schiebt den Hash einer unzitierten Zeile vor, ohne die Revision“) in die Roadmap-Zeile „Register revisions only from review“. **Ergänzen:** #15 nennt auch die Installationsskripte von ultra-brain. **Prüfen:** Die Wurzelquellen `Bauanleitung_Second-Brain.pdf` und `NoteGPT_Transcript…txt` werden vor dem Löschen am Inhalt gegen brain-knowledge gehalten. | Schritt 1 von 4f verlangt, dass jede Stelle gebaut, zugeordnet oder freigegeben weggefallen ist. Belege in der Akte | freigegeben 2026-10-05 |
| 33 | ultraloom, ultra-brain | Nur auf dieser Platte liegen: in ultraloom 3 Commits auf `master` samt Tag `loomux-1a-source`, `feature/agent-harness` (61 Commits), `claude/wiki-stufe-2`, `fix-audit-scheibe5` und die ungetrackten Dateien; in ultra-brain 19 Commits auf `master` samt beiden Aufnahme-Tags, fünf Seitenzweige, `refs/original/…` und sieben ungetrackte Dateien. | Bevor ein Ursprungsrepo gelöscht wird, entsteht je Repo ein `git bundle --all` samt einem Archiv der ungetrackten Dateien. Beides liegt dort, wo #24 (e) die Aufzeichnungen sichert (Tag oder Release-Anhang in loomux). | Gemessen am 2026-10-05: `git rev-list --count origin/master..master` ergibt 3 und 19, und von den Seitenzweigen hat nur `claude/ultra-loom-brain-fusion-a5bb17` ein Gegenstück auf dem Remote. Ohne Sicherung gingen die Tags, auf die sich jede Paritätsakte beruft, mit dem Repo verloren | freigegeben 2026-10-05 |
| 34 | — | Der Zuschnitt von 4f PR B und PR C deckt einige Stellen nicht ab, die die Suchliste aus #24 trifft (gemeldet von einer parallelen Sitzung, nachgeprüft am 2026-10-05). | **PR B:** `loomux dev switchover` samt `internal/switchover` und der Wächterregel zu `dev switchover prune-hooks` fällt weg, weil 4e ✅ ist. Die Marker `ultraloom`/`ulguard` in `gitfiles.RunsAGate` (`internal/setup/gitfiles/gitfiles.go:99`) fallen ebenso wie die Ausschlüsse `**/.brain.toml` und `**/.ultra-brain/**` in `index.AlwaysExcludes` (`internal/brain/index/walk.go:25-26`); die übersetzten Fälle mit `index.yml` ziehen mit. Die Vorlage `brain-review/SKILL.md:86` nennt kein `ultra-brain-…` mehr. **PR C:** `loomux dev record-case` fällt mit `dev import-cases`. **Ausnahmeliste des Tor-Tests:** `testdata/bench/1a-hooks.json` und `testdata/bench/search/v1/baseline/*` sind Messchronik wie `benchmarks.md`. Die Herkunftsspalte von `docs/*/migration.md` (je 40 Treffer) bleibt, bis der Plan nach dem Ende der Folgeprojekte entfernt wird. **Changelog:** Die Ausnahme „zwei Zeilen“ trifft mit der vollen Suchliste fünf, je eine in den veröffentlichten Einträgen 7.0.1, 7.0.0, 4.2.2, 2.5.0 und 2.3.0. Die Vorlage nannte zunächst drei, gezählt mit einer verkürzten Liste; nachgezählt am 2026-10-05. Die Zeilen in 7.0.1 und 7.0.0 stammen aus der Arbeit an 4e. Zwei Wege: (a) Die Ausnahme wächst auf diese fünf Zeilen, und #24 (g) bleibt, wie es ist. (b) Die jüngeren Zeilen werden ohne die Namen neu gefasst; das ändert #24 (g), „die Zeilen im Changelog bleiben als Geschichte“. Entschieden: (a). Die Ausnahme benennt die Zeilen nach ihrem Eintrag, nicht nach ihrer Nummer, weil jedes Release die Nummern verschiebt. | Ohne diese Zeilen träfe der Tor-Test aus PR C Stellen, die weder B noch C entfernt, und `dev switchover` bliebe als Werkzeug für einen Wechsel stehen, der abgeschlossen ist | freigegeben 2026-10-05 |
| 35 | — | Beim Planen von 4f PR B (Plan `2026-10-05-stufe-4f-b-code-und-doku.md`) zeigte der Code drei Stellen, an denen der Zuschnitt nicht passt, und drei offene Fragen | **Korrekturen:** (1) Klasse (b), „Schutz von `.ultraloom/vendor`“: `internal/hooks/worktree.go` hat keine Sonderregel für diesen Pfad, er steht nur in Kommentaren (`:78`, `:282-283`, `:414-416`, `:496`) und Testfixturen; `standsInside` und `leadsInto` schützen jeden gespiegelten Pfad und bleiben. Ohne sie könnte das Aufräumen am Sitzungsende über eine Junction den Haupt-Checkout treffen. Kommentare und Fixturen bekommen neutrale Beispiele. (2) Klasse (i): `loomux config` kennt nur Schlüsseloperationen (`internal/cli/config.go:59`) und kann keinen Kommentar vorschlagen; der Plan nennt dem Menschen die neuen Zeilen, er ändert sie von Hand. (3) Klasse (a), „wird mit genau diesem Fall geprüft“: Die Prüfung ist eine einmalige Probe je Aufrufer, festgehalten in `parity/stufe-4f.md`; ein bleibender Test mit `.brain.toml` als Eingabe widerspräche (d). **Annahmen:** (E2) Die `sources:`-Zeilen in `docs/wiki/**`, die in die Archive zeigen, bleiben bis (h), weil der Dateiname des Archivs selbst den Altnamen trägt; ebenso ihre Spiegel unter `internal/brain/apply/testdata/frontmatter/`. Nur die Prosa der Seiten wird neu gefasst. (E3) `docs/wiki/log.md` ist Chronik wie `benchmarks.md`: alte Einträge bleiben, neue nennen die Altprojekte nicht. (E4) Die Roadmap-Zeile „The hint beside a declaration without `[area]`“ (Folgezeile aus #28) fällt aus beiden READMEs, weil loomux nach (a) kein Altmanifest mehr erkennt. **Nachgezogen bei der Umsetzung (Controller-Entscheid, 2026-10-05):** Die „je 40 Treffer“ von #34 in `docs/*/migration.md` liegen nicht alle in der Herkunftsspalte; acht stehen in Beschreibungen, die die Migration selbst erzählen (etwa „moved from `ultra-brain/pkg/index`“). Sie bleiben mit dem Plan bis zu dessen Ende, statt umgeschrieben zu werden. Ebenso gehört ein im Wiki-Text zitierter Archivpfad (`docs/.superpowers/bench-ub/…`) zu den Archivverweisen aus (E2) | Gemessen am 2026-10-05 an `6e64a7e7` per `git grep` mit der Suchliste aus #24 | Annahmen des Plans, vom Nutzer mit dem Start der Umsetzung angenommen (2026-10-05) |
| 36 | — | Nach der Fertigstellung braucht es den Stand der Migration nicht mehr, und alle Hinweise auf die Altprojekte sollen aus dem Baum, außer Changelog und Benchmarks | Ein Abschluss-PR **vor** dem Neustart (#31), damit v1.0.0 schon ohne Altbestände erscheint. Die Papiere des Neustarts (Spec und Pläne) tragen keinen Altnamen und bleiben, bis er durch ist. Er nimmt den Migrationsplan `docs/{en,de}/migration.md` heraus, samt `internal/plancheck`, den Regeln in `AGENTS.md` und den Links aus den READMEs. Die Arbeitspapiere `docs/.superpowers/specs`, `plans` und `parity` sowie die Archive `specs-ul/`, `specs-ub/`, `plans-ul/`, `plans-ub/` und `bench-ub/` kommen als Anhang in das Archiv-Release `archive/parity-recordings` und verlassen den Baum. Die Wiki-Quellen und die Zeilen in `_identities.tsv`, die dorthin zeigen, werden umgehängt oder fallen. Die Folgeprojekte Flow und Web lesen ihren Stoff dann aus dem Archiv. Die Roadmap-Zeile der Web-App nennt `ultra-brain/web` nicht mehr. Danach führt der Tor-Test nur noch `benchmarks.md`, `testdata/bench` und die fünf Changelog-Zeilen als Ausnahmen. Das ändert #24 (h), wonach die Arbeitspapiere außer den Archiven bleiben und die Archive erst mit dem letzten Folgeprojekt gehen | Vorgabe des Nutzers vom 2026-10-05 nach PR C. Gezählt am selben Tag mit der Suchliste des Tor-Tests: Migrationsplan 80 Zeilen, Arbeitspapiere 79 Dateien mit 2.127 Zeilen, Archive 40 Dateien mit 805 Zeilen, Wiki 14 Dateien, Frontmatter-Spiegel 12 Dateien, `_identities.tsv` 29 Zeilen | Entschieden 2026-10-05: Abschluss vor dem Neustart; Migrationsplan löschen, Arbeitspapiere archivieren, Roadmap-Zeile neu fassen; Changelog und Benchmarks bleiben (#24 (f), (g)) |
| 37 | `trailhq/Graft` | Graft, Vorbild des Code-Graphen (Stufen G1–G5a, ✅), steht in 35 Code-Dateien, im Wiki, in den READMEs, in fünf Referenzseiten je Sprache und in 18 Arbeitspapieren | Graft bleibt nur als Hinweis auf die Idee: ein Satz in README und `architecture.md` (en/de) und der MIT-Hinweis für den portierten Code in `NOTICE.md` über `loomux dev notices`. Überall sonst fällt der Name in PR D von 4f; `benchmarks.md` bleibt Chronik, die Arbeitspapiere gehen nach #36 ins Archiv. Der Tor-Test führt `graft` mit genau diesen Ausnahmen | Vorgabe des Nutzers vom 2026-10-05. Die Kommentare „Ported from trailhq/Graft @ 1e352a3 (MIT)“ zeigen portierten Code; die MIT-Lizenz verlangt Copyright und Lizenztext bei jeder Kopie. Eine zentrale Angabe dafür fehlt heute, `dev notices` deckt nur Module und Grammatiken | Entschieden 2026-10-05 |

#### #24 im Einzelnen

Die Inventur vom 2026-09-27 suchte nach `ultraloom`, `ultra-brain`, `ulguard`,
`ulinit`, `ulflow`, `brain-mcp`, `brain guard`, `ultraLoomOwned`,
`ultraloom-wiki-guard`, `.ultraloom`, `.brain.toml`, `.ultra-brain`, den
Namen der Rückfälle und der Archivordner. Die Klassen und je ein Vorschlag; die
Spalte „Entscheidung“ hält fest, was der Nutzer am 2026-09-28 entschieden hat.

| Klasse | Umfang, Beispiele | Trägt Funktion bis | Vorschlag | Entscheidung |
|---|---|---|---|---|
| (a) Lese-Rückfälle | 14 Dateien: `LegacyBrainDirUntilStage3` samt `LOOMUX_LEGACY_BRAIN_DIR`, `ReadAreaManifestUntilStage4` mit den Manifestnamen `.ultra-brain/config.toml` und `.brain.toml`, `Manifest.Lanes`, der Umzug des Bestands (`internal/brain/apply/stock.go`) | 4e Punkt 2 der Stufe-4-Spec | Der Aufräum-Pull-Request, den 4e ohnehin vorsieht; er zieht die Kommentare mit, die noch `loomux migrate` als Ende nennen (`internal/config/legacy.go`, `artifacts.go`, `internal/cli/index.go`, `internal/brain/graph/read.go`, `internal/brain/index/reindex.go`) | Wer die Deklarationen der schreibgeschützten Bereiche aus `.brain.toml` nach `.loomux/config.toml` umlegt, bevor der Rückfall entfällt (Auflage in `parity/stufe-3a.md`; ihr Träger war `migrate`). **Entschieden:** der Mensch, in 4e, von Hand; vorher zeigt ein nur lesender Befehl `loomux area check <pfad>` je Bereich, was die heutigen Leser am alten Manifest annehmen, anderswo lesen oder ignorieren, und welche Datei sie ablehnen (nicht „umbenennen“: gemessen am 2026-09-28 ignoriert der Leser einen unbekannten Schlüssel wie `[llm.local]` still, siehe #25). Kein schreibender Übersetzer (#19). Betroffen sind am 2026-09-28 zehn Bereiche: neun mit `.brain.toml`, `ultraloom` mit `.ultra-brain/config.toml`. Der Befehl entfällt in 4f mit dem Rückfall. Erledigt mit dem Aufräum-PR (Nachtrag #28) |
| (b) Erkennung alter Host-Einträge | 4 Dateien: die Tabelle der abgelösten Hooks in `internal/hooks/status.go` (`ulguard`, `brain guard`, `ultraloom hook …`, `generate_index.py`), der Schutz von `.ultraloom/vendor` in `internal/hooks/worktree.go`, der Ausschluss `/.ultraloom/` in `internal/cases/gitworld.go` | bis jeder Wirt umgestellt ist (4e Punkt 3) | In 4f löschen | — |
| (c) Kommentare, Paketdoku, Meldungen | rund 40 Dateien, rund 110 Zeilen: Herkunftsangaben („moved from ultra-brain's …“), Zeilenverweise in die Python-Referenz, die Paketdoku von `internal/gitenv` (nennt `ulinit` und `ulguard` als eigene Programme), die Ausgabe „UltraBrain Wiki:“ von `loomux status` (`internal/hooks/status.go`) | sofort | Ohne die Namen neu fassen („the reference“) oder streichen, wo nur Herkunft steht; `internal/dev/importcases` fällt mit (e) | — |
| (d) Tests | rund 80 Dateien, die Altnamen als Eingabe benutzen oder die Referenz nennen | mit (a) bis (c) | Mit ihrem Code | — |
| (e) Aufzeichnungen | rund 1.000 Dateien unter `testdata/cases/*-source`, dazu die Welten und Nutzlasten; per Policy als Beweis geschützt | solange Parität nachzuweisen ist | Behalten als ausdrückliche Ausnahme, oder vor dem Löschen als Tag oder Release-Anhang sichern und die Wiedergabetests auf eigene Erwartungen umstellen | **Entschieden:** nach der Neuaufnahme (Schritt 1) die Originale als Tag oder Release-Anhang sichern, dann die Wiedergabetests auf eigene Erwartungen von loomux umstellen und die Aufzeichnungen aus dem Baum nehmen |
| (f) Nutzerdoku | 28 Dateien: die Herkunftsspalte in `docs/*/migration.md`, `cli-reference`, `configuration`, `getting-started` (die Rückfälle), `benchmarks` (Messungen gegen `ulguard` und `brain guard`), die Roadmap (`ulflow`, `ultra-brain/web`), `docs/wiki` | Referenzdoku mit (a) und (b), Plan und Roadmap mit dem Ende der Migration und der Folgeprojekte | Den Migrationsplan nach dem Ende entfernen oder archivieren; die Referenzdoku mit dem Code bereinigen | **Entschieden:** `benchmarks.md` bleibt als Chronik unverändert, benannte Ausnahme; neue Einträge nennen die Altprojekte nicht mehr |
| (g) `AGENTS.md`, `CHANGELOG.md`, `LICENSE.md` | die erste Zeile von `AGENTS.md`; Einträge im Changelog; der Required Notice in `LICENSE.md` zeigt auf `xidus90/ultra-brain` | — | `AGENTS.md` neu fassen; den Notice auf `xidus90/loomux` setzen | **Entschieden:** der Notice zieht auf `https://github.com/xidus90/loomux` um; die zwei Zeilen im Changelog bleiben als Geschichte, benannte Ausnahme |
| (h) Arbeitspapiere | rund 100 Dateien unter `docs/.superpowers/`, darunter die Archive `specs-ul/`, `specs-ub/`, `plans-ub/`, `bench-ub/` | Flow und Web speisen sich noch aus den Archiven | Die Archive nach den Folgeprojekten aus dem Repo nehmen (Tag oder externes Archiv), vorher die Quellen der Wiki-Seiten umhängen; Specs und Akten behalten | **Entschieden:** Specs, Pläne und Akten sind benannte Ausnahme; die Archive (`specs-ul/`, `specs-ub/`, `plans-ub/`, `bench-ub/`, `plans-ul/`) gehen mit dem letzten Folgeprojekt, vorher werden die Quellen der Wiki-Seiten umgehängt; `docs/wiki` gehört nicht zur Ausnahme |
| (i) Konfiguration | ein Kommentar in `.loomux/config.toml`; die Zeilen der Archive in `_identities.tsv` | mit (h) | Den Kommentar ändert ein Mensch; das Register folgt `reindex` | — |

**Zeitpunkt (entschieden 2026-09-28).** „Ende der Migration“ meint 4e. 4f
läuft danach und räumt alles, was der Fusion gehört: die Klassen (a) bis (g)
und (i). Flow und Web ziehen noch Stoff aus den Archiven unter (h); jedes
Folgeprojekt tilgt seine Verweise bei seinem Abschluss selbst und führt bis
dahin keine neuen ein.

**Schritt 1 von 4f, vor jedem Löschen (entschieden 2026-09-28): die
Gleichheit mit den Ursprungsrepos.**

1. Eine Inventur beider Repos auf ihrem letzten Stand, über alle Zweige: jede
   Stelle (Befehl, Flag, Konfigurationsschlüssel, Hook, Skill, Vorlage) ist in
   loomux gebaut, einer Stufe oder einem Folgeprojekt zugeordnet oder
   freigegeben weggefallen. Dieselbe Durchsicht wie am 2026-09-19, gegen den
   heutigen Stand; ein Fund wird ein neuer Nachtrag.
2. Die Fälle unter `testdata/cases/*-source` werden gegen die Ursprungsrepos
   auf ihrem heutigen Stand neu aufgenommen und mit den alten Aufnahmen
   verglichen. Ein Unterschied heißt, dass die Referenz sich seit der Aufnahme
   bewegt hat; er wird entschieden, bevor (e) die Aufzeichnungen aus dem Baum
   nimmt.

**Die benannten Ausnahmen:**
- `docs/{en,de}/benchmarks.md` (f),
- seit #34 die Messchronik `testdata/bench/1a-hooks.json` und
  `testdata/bench/search/v1/baseline/*`,
- die fünf Zeilen in `CHANGELOG.md` (g; #34, Weg (a)), je eine in den
  Einträgen 7.0.1, 7.0.0, 4.2.2, 2.5.0 und 2.3.0,
- `docs/*/migration.md` bis zum Ende des Plans: die Herkunftsspalte und, seit
  #35, auch die Beschreibungen der Zeilen, die die Migration selbst erzählen,
- die Arbeitspapiere unter `docs/.superpowers/` außer den Archiven (h),
- seit #35 `docs/wiki/log.md` und bis (h) die Verweise in `docs/wiki/**`, die
  in die Archive zeigen (`sources:`-Zeilen und im Text zitierte Archivpfade),
  samt ihren Spiegeln unter `internal/brain/apply/testdata/frontmatter/`.

**Fertig ist 4f, wenn** ein Grep nach der Suchliste der Inventur oben nur noch
in den benannten Ausnahmen trifft, und bis zum Abschluss des jeweiligen
Folgeprojekts auch in dem, was es noch trägt: den Archiven unter (h) und den
Zeilen der Roadmap und des Migrationsplans, die unfertige Folgeprojekte nennen
(`ulflow`, `ultra-brain/web`). Ein Tor-Test hält das fest, mit einer Liste
genau dieser Stellen, die jedes Folgeprojekt bei seinem Abschluss kürzt, damit
keine neuen Verweise hinzukommen.

### Offen nach 3b

Drei Fehler, die 3b von der Python-Referenz geerbt und absichtlich
nachgebildet hat, und eine Lücke, die 3b festgehalten hat. #1 und #4 sind
seit 4c-1 geheilt (2026-09-26); #2 und #3 sind nicht geheilt, und hier steht
kein Vorschlag: beide warten auf eine Entscheidung des Nutzers, ob und wie
sie geheilt werden. Die Belegstellen stehen in
`parity/stufe-3b.md`.

1. **Geheilt in 4c-1 (2026-09-26):** `--reject` schiebt jetzt `sources[]`
   der Seite (Revision und Hash) und das Register vor und hält an einer
   Quelle an, die sich seit der Fallbildung erneut bewegt hat
   (`fix(approve)`, Akte `parity/stufe-4c-1.md`, Abweichungsliste). Der
   Befund, wie er nach 3b stand: **`--reject` schiebt Revision und Hash nicht
   vor.** Die Ablehnung schreibt
   `audit.md`, entfernt den Fall und committet, schiebt aber weder die
   `sources[]` der Seite noch das Register vor; der nächste Abgleich eröffnet
   denselben Fall wieder. Akte, Abschnitt „Geerbt“, erster Absatz.
2. **Eine schließende Frontmatter-Zeile mit Leerraum (`--- `) verliert bei
   der Freigabe die alte Frontmatter.** Die zwei Frontmatter-Muster der
   Referenz sind sich über diese Zeile uneinig; geschrieben wird dann nur
   `generated` und `verified`. Akte, Abschnitt „Geerbt“, zweiter Absatz, und
   Zeile „Zwei Frontmatter-Muster“ der Abweichungsliste (Golden
   `s13-closing-blanks`). Am 2026-09-27 gegen den Code bestätigt: `advanceBlock`
   und `documentBlock` in `internal/brain/apply/frontmatter.go` lesen die
   schließende Zeile verschieden; `--reject` weicht seit 4c-1 über
   `AdvanceSources` aus und lässt die Seite unberührt. Die kleinste Heilung wäre,
   `documentBlock` dieselbe schließende Zeile `---[ \t]*` zu geben; das Golden
   `s13-closing-blanks` änderte sich mit.
3. **Versteckte Auszeichnung (`%%`, `<!--`, Bidi-Steuerzeichen) auf einer
   Überschriftenzeile eines Vorschlags wird keinem Abschnitt angelastet.**
   Akte, Abschnitt „Überlebende Mutanten“, Absatz „Geerbt, festgehalten und
   nicht geheilt“; `TestHiddenMarkupOnAHeadingLineIsChargedToNoSection` hält
   das Verhalten fest. Am 2026-09-27 gegen den Code bestätigt; die Prüfung
   liegt in `internal/brain/evidence/evidence.go` (`hiddenMarkup`), nicht in
   `apply`. Die kleinste Heilung wäre, `hiddenMarkup` auch auf die
   Überschriftenzeile anzuwenden und den Treffer dem Abschnitt anzulasten, den
   sie eröffnet.
4. **Geheilt in 4c-1 (2026-09-26):** `reindex` und `approve` teilen jetzt
   eine Sperre je Bereich, `<zustand>/areas/<scope>.lock` (`fix(index)`,
   Akte `parity/stufe-4c-1.md`, Abweichungsliste). Der Befund, wie er nach
   3b stand: **`reindex` und `approve` teilen keine Sperre.** Laufen beide
   zugleich über
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
2. Je Wirt: `loomux init` (`migrate` fällt weg, #19). Alte Hook-Einträge —
   Marke `ultraLoomOwned: true` (Claude), Gruppe `ultraloom-wiki-guard`
   (Antigravity), `ulguard`, `brain guard` — entfernt der Mensch nach der
   Checkliste von 4e, dann `.ultraloom/`, `.brain.toml` und `.ultra-brain/`;
   ein altes Manifest erst, nachdem sein Inhalt nach `.loomux/config.toml`
   umgezogen ist und `loomux area check` ihn bestätigt hat (Nachtrag #24,
   Klasse a, 2026-09-28).
   Vorher der Abgleich des Maschinenzustands (Punkt 2 unter „Datenumzug“).
   Korrigiert am 2026-09-29 (Nachtrag #26): Diese Schritte stehen je Projekt
   in einem `apply.sh`, das der Agent vorbereitet (`loomux dev switchover
   render`) und ein Mensch aufruft. Es ersetzt die Registry, legt die
   vorbereitete `.loomux/config.toml` an ihren Platz, fährt `loomux init
   --yes`, nimmt die alten Hook-Gruppen aus der `.claude/settings.json` des
   Projekts (`loomux dev switchover prune-hooks`) und entfernt die alten
   Dateien. Von Hand bleiben Block 1 bis 3 der Checkliste von 4e, darunter der
   Abgleich des Maschinenzustands, der Rauchtest und, was das Skript nicht
   anfasst: die Antigravity-Gruppe `ultraloom-wiki-guard` in
   `.agents/hooks.json`. `loomux merge-hook install` läuft einmal, nach dem
   letzten Projekt. Reihenfolge: zwei Piloten (`ecoflow` am 2026-09-29,
   `space` am 2026-09-30, beide umgestellt und gemessen), dann die Welle über
   `iam_backend`, `iam_frontend`, `iam_workers`, `iam_wiki` und den Vault,
   gelaufen am 2026-10-01/02 nach der Lane-Probation (Nachträge #26 und #27).
   `ultra-brain` und `ultraloom` fallen aus der Welle (Entscheidung des
   Nutzers vom 2026-10-01): beide werden nach der Migration gelöscht.
3. Rauchtest je Wirt: ein erlaubter Edit, ein verweigerter Edit, ein Commit
   durch commit-msg und Pre-Commit, eine Suche über MCP.

## Folgeprojekte

In dieser Reihenfolge, je mit eigener Spec:

1. **Flow-Migration.** Basis ist ulflow M1 (`feature/agent-harness`: Laufzeit,
   Journal, Resume, Replay, `internal/flowload`). Dazu M2 (`claude -p`,
   `agy -p`) und M3 (`verify_until_green` als Daten-Flow). Eingang ist
   außerdem die Multi-Provider-Spec
   (`specs-ul/2026-08-24-multi-provider-llm-design.md`, Nachtrag #22): ein
   `Model`-Port mit SDK-Adaptern und eigenem Werkzeug-Ausführer, der neben
   oder statt der CLI-Aufrufe aus M2 steht. Das Flow-Format ist
   heute TOML mit Go-Bausteinen und Go-Prädikaten; ob es für den Editor so
   bleibt oder um Ausdrücke erweitert wird, entscheidet dieses Folgeprojekt.
   Bis dahin bleiben `ulflow` und die Python-Flows in ihren Repos benutzbar.
2. **Web-Migration.** Die React/Vite-App aus `ultra-brain/web` zieht nach
   `loomux/web`, ihr Build wird per `go:embed` eingebettet, `serve` bekommt
   `/api/…`. Mit ihr zieht das Bundle-Tor aus `hooks/bundle-gate.sh` um.
3. **Grafischer Flow-Editor** auf der Web-App und der Bausteinregistry.

Die alten Repos werden archiviert, wenn beide Migrationen fertig sind.

## Offen und vor dem Bau zu messen

Stand am 2026-09-27: drei der vier Punkte sind beantwortet; offen bleibt nur die
Vektorwiederverwendung, die mit 4e gemessen wird.

- ~~**Wohin gehen die ~40 ms von `brain guard`** über seinem Startboden? (Stufe 1a,
  vor dem Zielwert für Stufe 2.)~~ Beantwortet in Stufe 1a (`docs/de/benchmarks.md`,
  Eintrag 1a, Lesart): Auf einem registrierten Bereich braucht dasselbe
  `brain.exe` 26,7 ms warm gegen seinen Boden von 24,3 ms; die 72 ms der
  Grundlinie stammen von einer kälteren Maschine, und vom Boden sind ~19 ms das
  Zeitzonen-`init` von BurntSushi/toml, das `third_party/toml` seither beim
  ersten Gebrauch auflöst.
- ~~**Antigravity:** ob die JSON-Hülle auf stdout gelesen wird oder nur der
  Exit-Code; ob `PreInvocation` Kontext ins Modell schreiben kann; ob Stop eine
  harte Zeitgrenze unter 300 s hat.~~ Beantwortet (`specs-ul/2026-09-10-antigravity-hook-messung.md`,
  Nachmessung 2c; `parity/stufe-4a-2.md`, Probe 2026-09-25): stdout wird gelesen,
  `{"decision":"continue"}` hält den Stop (gemessen mit agy 1.2.11);
  `PreInvocation` speist Kontext über `injectSteps` ein; die Frist eines
  Command-Hooks ist 30 s und je Hook mit `"timeout"` in `hooks.json`
  einstellbar (aus dem Binary gelesen, nicht live gemessen), `loomux init`
  schreibt für `Stop` 300 s.
- **Vektorwiederverwendung in qmd** nach geänderten Ignore-Mustern (bei der
  Umstellung, 4e).
- ~~**Go-Suche:** woher die abweichende Rangfolge gegenüber Python kommt — der
  erste Paritätsfall von Stufe 1b.~~ Beantwortet und freigegeben in 1b-1
  (`parity/stufe-1b-1.md`, Fälle `brain-search/fast` und `keyword`,
  2026-09-16): Die Referenz erweitert die Anfrage per LLM, führt per Maximum
  zusammen und schneidet unter 0,3; loomux sucht `vec` ohne Erweiterung und
  Rerank. Bei `keyword` sind die Ränge seit 4c-2 gleich (50/50).
