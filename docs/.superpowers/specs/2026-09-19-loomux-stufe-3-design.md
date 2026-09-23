# Stufe 3: Brain-Pflege — `reindex`, `reconcile`, das Prüfzentrum, die Wiki-Werkzeuge

**Datum:** 2026-09-19
**Deckt ab:** die ganze Stufe 3 der Fusions-Spec
(`2026-09-14-loomux-fusion-design.md`), geschnitten in **3a**, **3b** und
**3c**. Ein Implementierungsplan entsteht in diesem Zug nur für **3a**.
**Ort:** eigener Worktree von `master`. Stufe 3 läuft **parallel** zu 2b
(`claude/2b-planung-4d6dcc`) und 2c (`sdd-2c`); keine der drei wartet auf eine
andere.
**Referenz:** **Python**, für jeden Befehl dieser Stufe. Wo eine Go-Form aus
`ultra-brain/pkg/` existiert (`approve`, `evidence`, `case`, `index`), zieht sie
als Code um, ist aber **nicht** die Referenz: jeder Fall wird gegen die
Python-Ausgabe aufgezeichnet, und jede Abweichung des Go-Stands kommt in die
Akte.
**Aufzeichnungsgrundlage:** Tag `loomux-3-source` auf `ultra-brain`
`3cc72d2` — derselbe Commit wie `loomux-1a-source`; das Repo hat sich seit
Stufe 1a nicht bewegt. Den Tag setzt ein **Mensch**, vor dem Aufzeichnen
(Task 15 des Plans); loomux schreibt nicht in das Quellrepo.
**Abweichungsliste:** `docs/.superpowers/parity/stufe-3a.md` (je Teilstufe eine).

## Ziel

Die Pflegeschicht von `ultra-brain` zieht nach loomux: der Index, das Tor, an
dem eine Wissensänderung zum Fall wird, das Prüfzentrum, in dem ein Mensch sie
entscheidet, und die Werkzeuge, die die Wiki-Schicht in Form halten. Nach
Stufe 3 gibt es keinen Pfad mehr, auf dem loomux Wissen indiziert, ohne den
Fall vorher zu öffnen — oder laut zu sagen, dass keiner aufgehen konnte.

Was diese Stufe **nicht** baut: `loomux init` als Wirts-Installer, den
post-merge-Hook, `convert`/`fetch`, das lokale Modell und `bench` — die sind
Stufe 4.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Schnitt | Drei Teilstufen: 3a Erkennen, 3b Entscheiden, 3c Pflegen |
| Referenz | Python für alles; die Gleichheit der vorhandenen Go-Formen wird nachgewiesen, nicht vorausgesetzt |
| Bereichs-Onboarding | Eigener Befehl `loomux area add`, **ohne** Hook- und Agenten-Installation. `loomux init` (Stufe 4) ruft ihn |
| `reindex`/`embed` | Gehören in 3a, vor `reconcile`. Nachtrag #17 der Fusions-Spec (bisher keiner Stufe zugeordnet) |
| Ereignisprotokoll | Lese- und Drop-Seite in 3a, weil `reconcile` ohne sie unvollständig ist. Der Schreiber ist der Git-Hook, und der kommt mit Stufe 4 |
| Zustandsort | Ab 3a schreibt loomux nach `LOOMUX_STATE_DIR`; das Altverzeichnis wird nur noch gelesen. Die befristete Ausnahme der 1b-1-Akte läuft damit aus, wie angekündigt |
| Parität der Git-Fälle | Setzt `internal/cases/gitworld.go` aus 2c voraus; 3a baut nichts Zweites daneben |
| Freigaben | Nachträge #1 (`check`), #2 (`lint --scope all`) und #3 (`embed`) sind am 2026-09-19 für Stufe 3 freigegeben |

## Befunde, die den Entwurf formen

Gegen den Code gelesen am 2026-09-19, nicht gegen die Doku. Drei davon
widersprechen der Fusions-Spec.

- **`Locking` ist nicht offen.** `internal/lock` ist bereits die Portierung
  von `locking.py` und hält die Sperre über ein Betriebssystem-Handle, nicht
  über PID und Übernahmefrist wie `locking.claim`/`_take_over`. Das ist eine
  Abweichung mit Grund — ein sterbender Prozess gibt das Handle ohne Aufräumen
  frei —, und sie gehört in die Akte, nicht in die Aufgabenliste. **Fehlend ist
  nur `replace_text`**: das atomare Ersetzen mit Wiederholungen
  (`locking.py:219-252`), das jeder Schreiber dieser Stufe braucht.
- **`merge-events` ist kein Befehl.** `merge_events.COMMANDS`
  (`merge_events.py:605`) *ist* `brain hook install|status|remove`
  (`cli.py:625`) — und der steht als Nachtrag #5 bereits bei Stufe 4.
  Geschrieben wird das Protokoll vom Shell-Hook selbst (`_write_hook`,
  `merge_events.py:489`); `record_event` hat im Python-Code keinen Aufrufer.
  Gelesen wird es von `reconcile._land_merge` (`reconcile.py:96`). Stufe 3
  braucht also `read_events`/`drop_event`/`events_path` in 3a und keinen
  eigenen Befehl.
- **`reindex` und `embed` gibt es in loomux nicht**, und die Fusions-Spec
  ordnet sie keiner Stufe zu. Sie sind trotzdem Voraussetzung: der Satz
  „`reconcile` auch als Durchgang vor `reindex`" beschreibt eine Kopplung an
  einen Befehl, den es nicht gibt. **Portiermasse nachgezählt am 2026-09-20,
  nicht geschätzt:** `pkg/index/` ohne Tests sind 1.496 Zeilen, davon sind
  zwei schon in loomux. `identity.go` (196) ist vollständig umgezogen
  (`internal/brain/identity/identity.go`, 216 Zeilen, dieselben Funktionen plus
  `parseRevision`); von `catalog.go` (194) ist nur die **Leseseite** da
  (`internal/brain/catalog`, 69 Zeilen: `ReadAreaCatalog`,
  `RenderRootCatalog`), die Schreibseite `RenderCatalog`/`WriteCatalogs`/
  `ReadIntro` fehlt. Zu bauen bleiben also **rund 1.230 Zeilen**:
  `qmd_config` 432, `reindex` 310, `walk` 225, `document` 139 und die
  Katalog-Schreibseite.
- **Die Registry hat in loomux nur eine Leseseite.** `config.ReadRegistry`
  (`internal/config/registry.go:57`) ist da; nichts schreibt sie. Der einzige
  Schreiber dieser Stufe ist `area add` — `reconcile` liest die Registry nur
  (`reconcile.py:112`, `cli._registered`) und schreibt Fälle, Statistiken und
  `last-run.txt`.
- **`[maintenance]` liest in dieser Stufe niemand.**
  `internal/config/declaration.go:78-81` prüft `on_merge` und `branch`; die
  einzigen Leser der Referenz sind `merge_events.consenting`,
  `install` und `_drift` (`merge_events.py:275,321-433`) — also die
  Hook-Verwaltung aus Stufe 4. Weder `reconcile` noch `apply` fassen die
  Schlüssel an. 3a und 3b lassen sie unberührt.
- **Der Auffangdurchgang ist nicht optional, aber er ist kein Tor.**
  `_reindex` (`cli.py:964-1042`) begründet den Durchgang selbst: `reindex`
  schreibt für jede Quelle, deren Hash sich bewegt hat, eine neue `Identity`.
  Liefe er ohne vorherigen `reconcile`, ginge eine Wissensänderung am Prüftor
  vorbei und der nächste `reconcile` fände die Quelle identisch zu ihrem
  Register. **Gegen den Code gelesen bricht `reindex` aber nicht ab, wenn der
  Durchgang Fälle eröffnet:** es zählt sie auf stderr auf und indiziert weiter
  (`cli.py:1034-1041`). Abgebrochen wird nur, wenn der Durchgang selbst
  scheitert (`ReconcileError`, `OSError` ⇒ Exit 1, Meldung nennt den Weg).
  Erklärt kein Bereich ein Prüfzentrum, warnt es lang und indiziert ebenfalls
  weiter — mit der Begründung, dass ein Korpus ohne Prüfzentrum kein Tor hat,
  um das man laufen könnte.
- **Ein fehlendes Manifest ist kein Fehler, ein kaputtes schon.**
  `_manifests` (`reconcile.py:253-268`) überspringt einen Bereich, der noch
  keine `.brain.toml` geschrieben hat — registriert vor der Erklärung ist der
  Normalfall. Eine Datei, die existiert und nicht liest, wird zu
  `ReconcileError`.
- **`internal/brain/wiki/lint.go` verweist Regeln ins Leere.**
  `wrong-direction`, `long-planned`, `no-sources`, `log-date-form` zeigen auf
  Prüfungen, die es in loomux nicht gibt — das ist Nachtrag #1 und der Grund,
  warum `check` und `lint --scope all` zusammen in 3c stehen.
- **`tools/cases.py` kennt Welten, aber kein Git.** Es stellt `world/` her und
  vergleicht `world_after/` als Dateibäume (`tools/cases.py:36,121-159`). Ob
  ein `.git` darin unverändert durchläuft, ist **nicht geprüft**; `reconcile`,
  `vcs` und das Ereignisprotokoll brauchen echte Repos. Task 0 des Plans klärt
  das, bevor Fälle versprochen werden.

## Teilstufen

| Teilstufe | Inhalt | Warum hier |
|---|---|---|
| **3a Erkennen** | `lock.ReplaceText`; `legacy.go` auf „neu zuerst, alt als Rückfall"; Registry-Schreibseite; `loomux area add`; `loomux reindex` und `loomux embed` (Umzug `pkg/index`); `loomux reconcile` (`reconcile.py` 1.043, `case.py` 253, `package.py` 163, `derive.py` 33, Leseseite `vcs.py`); die Lese-/Drop-Seite des Ereignisprotokolls; der Auffangdurchgang vor `reindex` | Alles, was schreibt, braucht zuerst Sperre und atomares Ersetzen. `reindex` ist der Befehl, an den der Durchgang gekoppelt ist. `reconcile` legt die Fälle an, ohne die 3b keinen Eingang hat |
| **3b Entscheiden** | `loomux cases`, `loomux case`, `loomux approve`; `apply.py` (1.378), `evidence.py` (677), die Schreibseite von `vcs.py` (Zweig, Commit, `RefMoved`) | Hängt an den Fällen aus 3a. Hier wird die Gleichheit der vorhandenen Go-Formen (`approve.go` 640, `evidence.go` 449, `case.go` 247) gegen Python nachgewiesen |
| **3c Pflegen** | `loomux brain check file\|bundle\|all` mit OKF, Hausregeln, Föderation (#1); `loomux lint --scope all\|<scope>` (#2); `loomux wiki init\|types\|retype`; Upkeep in `serve` (Namen berichtigt 2026-09-23, „Befunde 3c“) | `check` und `lint --scope all` gehören zusammen (#2 sagt „mit #1"). Upkeep ruft `reconcile` aus 3a und meldet den Rückstand, den 1b-2 dorthin verschoben hat |

## Pakete

| Paket | Inhalt | Kommt aus | Teilstufe |
|---|---|---|---|
| `internal/lock` | `ReplaceText` ergänzt | `locking.py:219-252` | 3a |
| `internal/config` | Registry-Schreibseite: Bereich anlegen, ändern, atomar zurückschreiben | `registry.py`, `init.py` | 3a |
| `internal/config` | `legacy.go` umgestellt: **neu zuerst, alt als Rückfall** | — | 3a |
| `internal/brain/index` | `walk`, `document`, `reindex`, `qmd_config` und die Katalog-**Schreibseite**; `identity` und die Katalog-Leseseite sind seit 1b-1 da | `ultra-brain/pkg/index` (Umzug) | 3a |
| `internal/brain/maintenance` | `reconcile.go`, `case.go`, `package.go`, `derive.go`, `events.go` | `maintenance/{reconcile,case,package,derive,merge_events}.py` (Neuschrift). `package.go` ist hier und nicht in 3b, weil `_land_case` neben jeder Fallakte ein `package.md` schreibt (`reconcile.py:783-800`) | 3a |
| `internal/brain/vcs` | Leseseite: `show_blob`, `changed_paths`, `commit_subjects`, `repository_root`, `common_directory` | `maintenance/vcs.py` | 3a |
| `internal/brain/vcs` | Schreibseite: `commit_paths`, `RefMoved`, die Ablehnungen bei laufender Git-Operation | `maintenance/vcs.py:108-283` | 3b |
| `internal/brain/apply` | Vorschlag anwenden, Segmente, Frontmatter, Format | `apply.py`, `pkg/maintenance/{patch,format,frontmatter,lookup}.go` | 3b |
| `internal/brain/evidence` | Belegbindung: jedes Zitat wörtlich aus einem Paketsegment | `evidence.py`, `pkg/maintenance/evidence.go` | 3b |
| `internal/brain/check` | **ergänzt** um OKF, Hausregeln, Föderation — das Basispaket ist seit 1a da, die drei Achsen fehlen | `ultra-brain/pkg/check/{okf,house,run}` (`code` fällt weg, Fusions-Spec #18) | 3c |
| `internal/brain/wiki` | `types`, `census`, `retype`; die zwölf Regeln von `lint.py` als eigener Regelsatz (`sweep.go`) neben der Go-Form in `lint.go`; das Anlegen (`scaffold.go`) ist seit 3a da | `wiki/{types,census,retype,lint}.py` | 3c |
| `internal/serve` | Upkeep: `reconcile` nachholen, Hinweis anhängen | 1b-2 hat den Platz freigelassen | 3c |
| `internal/cli` | `area`, `reindex`, `embed`, `reconcile` (3a); `cases`, `case`, `approve` (3b); `brain check`, `wiki`, `lint --scope` (3c) | — | je Teilstufe |

**Abhängigkeitsregeln.** `brain/maintenance` → `brain/identity` (es liest das
Identitätsregister), `brain/index`, `brain/vcs`, `config`, `lock`.
`brain/index` → `config`, `brain/identity`, `brain/catalog`, `brain/search`
(`QmdMcpPort.Embed` liegt in `internal/brain/search/mcp.go:239`).
`serve` → `brain/maintenance` (Upkeep). **`hooks` importiert nichts davon** —
der Pfad an jedem Edit bleibt frei von der Pflegeschicht.

## Zustand und Orte

Ab 3a schreibt loomux seinen Pflegezustand nach `LOOMUX_STATE_DIR`
(`%LOCALAPPDATA%\loomux`, unter POSIX `$XDG_STATE_HOME/loomux`):

| Was | Ort | Schreiber |
|---|---|---|
| `registry.toml` | `<state>/registry.toml` | `area add` allein (nie ein Agent, nie `reconcile`) |
| Identitätsregister, Kataloge, Graphen je Bereich | `<state>/areas/<scope>/` | `reindex` |
| `last-run.txt`, Statistiken je Bereich | `<state>/maintenance/` | `reconcile` |
| Ereignisprotokoll, abgelegte Ereignisse | `<state>/maintenance/` | Git-Hook (Stufe 4); loomux liest und legt ab |
| Fallakten | Prüfzentrum des Tresors, aus `[layout].review` | `reconcile`, `approve` |

**Leserückfall, und warum er eine eigene Aufgabe ist.** Fehlt eine Datei unter
`LOOMUX_STATE_DIR`, wird sie einmal aus `LOOMUX_LEGACY_BRAIN_DIR` (sonst
`%LOCALAPPDATA%\brain`) gelesen und beim nächsten Schreiben am neuen Ort
angelegt. Geschrieben wird **nie** ins Altverzeichnis.

Das ist kein Nebensatz: `config/legacy.go:33` schickt heute die Leser aus
1b-1 — `search`, `status`, `catalog`, `read`, `neighbors` — fest ins
Altverzeichnis, und zwar mit der Begründung, dass loomux keinen eigenen
Schreiber hat. Ab dem ersten `loomux reindex` stimmt die nicht mehr: Register
und Kataloge lägen neu, gelesen würde alt, und die beiden Hälften des Produkts
wären sich über den Ort der Wahrheit uneinig. Die Umstellung von `legacy.go`
auf „neu zuerst, alt als Rückfall" ist darum eine eigene Aufgabe, direkt nach
`ReplaceText` und **vor** `reindex`. `loomux migrate` (Stufe 4) zieht den Rest
um; bis dahin bleibt das Alte als Sicherung liegen.

**Das Prüfzentrum** ist der eine Ort des Tresors, den `[layout].review` eines
Bereichs erklärt (`reconcile._review_root`). Erklären zwei Bereiche
verschiedene Orte, ist das ein Fehler und kein Vorrang; erklärt keiner einen,
endet `reconcile` mit `NoReviewCentreError` und nennt den Konfigurationsschlüssel.

## 3a im Einzelnen

### `lock.ReplaceText`

Schreibt in eine temporäre Datei im Zielverzeichnis, `fsync`, dann
`os.Rename`. Unter Windows scheitert das Umbenennen, solange ein anderer
Prozess die Zieldatei offen hält; darum die Wiederholungen aus
`_replace_eventually` (fünf Versuche, 20 ms Pause). Scheitert der letzte,
schlägt der Aufruf fehl und nennt Pfad und Fehler — es bleibt nie eine halbe
Datei stehen.

### Registry-Schreibseite

`config.AddArea` und `config.WriteRegistry`: unter `lock.Acquire` auf
`<state>/registry.lock` lesen, ändern, über `ReplaceText` zurückschreiben. Ein
Bereich, dessen `scope` schon registriert ist, ist ein Fehler, kein
Überschreiben. Die Reihenfolge der Einträge ist stabil (nach `scope`
sortiert), damit ein Fall die Datei vergleichen kann.

### `loomux area add`

Was `brain init` tut, ohne seine Hook-Hälfte: Bereich in die Registry, `[area]`
und `[layout]` in die `.loomux/config.toml` des Repos, Wiki-Gerüst anlegen,
die Weichenregel („Wohin welches Wissen gehört", deutsch, weil sie in ein
fremdes Repo kopiert wird) in die Projektanweisung schreiben, dann `reindex` —
abschaltbar mit `--no-reindex`.

Flaggen: `--path`, `--scope` (Vorgabe `project/<name>`), `--wiki`, `--sources`,
`--merge-branch`, `--privacy`, `--no-reindex`, `-y`. **Nicht übernommen:**
`--agents` und `--no-hook` — die gehören `loomux init` in Stufe 4.

### `loomux reindex` und `loomux embed`

Umzug von `pkg/index` mit seinen Tests, auf 100 % gehoben. `reindex` baut
Kataloge, Graph und Identitätsregister neu; `embed` erzeugt die Vektoren, die
`reindex` offen lässt, über `QmdMcpPort.Embed` — den Port gibt es in loomux
schon, nur ohne Befehl (Nachtrag #3).

**Der Auffangdurchgang.** `reindex` fährt zuerst `reconcile` über die
Registry, die es gleich begehen wird — sonst liefe eine ungeprüfte
Wissensänderung am Tor vorbei. Danach indiziert es, auch wenn Fälle aufgegangen
sind; die Fälle stehen auf stderr, weil stdout das Ergebnis des Befehls trägt
und sie eine Nebenwirkung sind. Drei Zustände, wie in der Referenz:

| Zustand | Verhalten |
|---|---|
| Durchgang grün, Fälle eröffnet | Fälle auf stderr aufzählen, dann indizieren, Exit vom Indexlauf |
| Kein Bereich erklärt ein Prüfzentrum | Lange Warnung auf stderr (das Register rückt ohne Prüffall vor), dann indizieren |
| Durchgang scheitert (`ReconcileError`, `OSError`) | **Exit 1, nicht indiziert**; die Meldung nennt Ursache und Weg: beheben, `loomux reconcile`, dann `loomux reindex` |

### `loomux reconcile`

Neuschrift nach `reconcile.py`. Der Ablauf, gegen `reconcile.py:194-228`
gelesen:

1. Manifeste aller registrierten Bereiche laden; ein Bereich ohne Manifest
   wird übersprungen, ein unlesbares Manifest bricht laut ab.
2. Das eine Prüfzentrum bestimmen.
3. Je Bereich scannen: gezählt, gehasht, verändert.
4. **Erst** Quellfälle, **dann** Zusammenführungsfälle — ein Merge-Fall tritt
   hinter einen stehenden Quellfall derselben Seite zurück und kann nur einen
   sehen, der schon auf der Platte liegt.
5. Fälle nach `id` entdoppeln, `last-run.txt` schreiben, Bericht zurückgeben:
   gezählt, gehasht, Fälle, Zeitpunkt, unlesbare Quellen.

Ein Fall wird über `ReplaceText` geschrieben, unter der Sperre des
Prüfzentrums.

**Der Vorschlag bleibt in 3a aus, und das ist beabsichtigt.**
`reconcile._proposers` (`reconcile.py:271-310`) baut für jeden
`local_only`-Bereich einen Ollama-Proposer; das lokale Modell steht in
**Stufe 4**. Ein Korpus ohne `local_only`-Bereich fragt nie danach — der
Normalfall, auch im loomux-Repo. Trifft 3a doch einen, öffnet es den Fall mit
`manual = true` und einem `note`, statt zu scheitern: die Feldsemantik gibt
das her (`case.py:52-58`, „a closed area with no local proposal"), und ein
Fall ohne Vorschlag ist immer noch ein Fall, den ein Mensch entscheiden kann.
Der Eintrag steht in `parity/stufe-3a.md`; mit dem Modell in Stufe 4 fällt die
Abweichung weg.

### Ereignisprotokoll

`events.go` bringt `events_path`, `read_events`, `drop_event` und das Format
einer Zeile (`_render_event`/`_parse_event`). **`record_event` zieht nicht
mit:** sein einziger Schreiber ist der Shell-Hook aus Stufe 4, und die Fälle
dieser Stufe bringen die Protokolldatei fertig in ihrer `world/` mit. Eine
Schreibfunktion ohne Aufrufer wäre Code, den nur ihr eigener Test benutzt.

## 3b im Einzelnen

`loomux cases` listet, was im Prüfzentrum wartet; `loomux case <id>` zeigt
Paket und Vorschlag; `loomux approve` entscheidet einen Fall und committet die
Änderung über `vcs.commit_paths` auf den **aktuellen** Ref des Tresors.
Korrigiert am 2026-09-22: hier stand „in einen Zweig“, aber `commit_paths`
legt keinen Zweig an (`vcs.py:108-228`).

**Die Belegbindung ist das Herz.** Jede Behauptung des Vorschlags braucht ein
wörtliches Zitat aus einem Segment des Pakets (`evidence.py`, Kopfkommentar
und `check_evidence`); ohne Beleg wird nicht angewandt. Die Fallstricke stehen
dort schon benannt — eine Belegzäune, die selbst einen Segmentrumpf zitiert,
darf nicht als zweite Behauptung gelesen werden, und die Zaunerkennung folgt
CommonMark. Diese Regel wird nicht neu entworfen, sondern gegen die
Python-Fälle nachgewiesen.

**Der Gleichheitsnachweis.** Derselbe Fallsatz läuft durch `brain approve`
(Python), `brain approve` (Go, `pkg/maintenance/approve.go`) und die
loomux-Form. Jeder Unterschied zwischen den beiden alten Formen ist ein
Eintrag in `parity/stufe-3b.md` mit Entscheidung, welche Seite gilt.
**Überholt am 2026-09-22** durch die Befunde darunter: der Nachweis ist am
Code geführt und für `approve.go` negativ. Die Go-Form läuft darum nicht durch
den Fallsatz; ihre Abweichungen stehen einmal, gelesen, in der Akte, und die
Fälle laufen Python gegen loomux.

### Befunde 3b, gegen den Code gelesen am 2026-09-22

Referenz ist `ultra-brain` auf `loomux-3-source` (`3cc72d2`); an
`src/brain/maintenance/` und `cli.py` hat sich seit dem Tag nichts geändert.

- **`approve.go` (640 Zeilen) ist keine Portierung von `apply.py` (1.378).**
  Es fehlen: die Schreibschranke `_gate`/`_preflight` (Links, Junctions,
  8.3-Namen, Einschluss im Tresor, `apply.py:467-484` und `:556-633`), die
  Prüfung von `[layout].review` auf absolut und `..` (`_layout`,
  `:656-676`; dass der Schlüssel gesetzt ist, prüft Go), `_is_bundle` für
  ein Wiki außerhalb des Tresors, die Quellsuche über die Register **aller**
  registrierten Bereiche (`_resolve_sources`, `:155-186`; Go liest nur das
  Register des Tresors), die Liste der berührten Dateien für den
  Abbruchhinweis (`place.touched`; Go verwirft Schreibfehler mit `_ =`),
  `_unrecorded` bei der gescheiterten Belegprüfung (bei bewegtem Ziel und
  bewegter Quelle hat Go es). Texte weichen ab: `_safe` (200 Zeichen,
  Ersatz `·`, eigene Satzzeichenliste), die drei Vermerke, die Zeile
  `- entschieden:` der gescheiterten Belegprüfung, Zeitstempel
  (`isoformat()` gegen RFC 3339). Zeilen am 2026-09-22 nachgerechnet, die
  Einzelheiten stehen in `parity/stufe-3b.md`.
- **Die Commit-Seite ist in Go unsicher.** `approve.go:294-394` ruft die
  Plumbing wie `vcs.commit_paths` und übergibt `update-ref` den alten Wert,
  prüft aber dessen Exitcode nicht (`:391`) — ein verlorener Tausch gilt als
  Erfolg. Es fehlen die Weigerung bei laufendem Rebase oder Merge
  (`vcs.py:265-275`), `RefMoved` und Wiederholung, `created=False` bei
  unverändertem Baum und `:(literal)`; auch die übrigen Exitcodes bleiben
  ungeprüft.
- **Zwei Wege zu `ProposalRefused`.** Die gescheiterte Belegprüfung
  (`_refuse`, `apply.py:1032-1062`) schreibt `note`, bei einem eigenen
  Vorschlag `manual = true` und einen Auditblock. Ein Diff ohne Zaun, ein
  überlappender oder unpassender Hunk (`_collect`, `_hunks`, `_patch`,
  `:1065-1140`) schreibt **nichts**.
- **Nach einem geschriebenen `approve`** fährt Python erst `reconcile`, bei
  dessen Scheitern hält es an, dann `reindex` (`cli.py:1432-1516`). Die
  Go-Form fährt nur `reindex`. Der Exit bleibt in beiden 0.
- **Geerbter Fehler:** `--reject` rückt Revision und Hash der Seite nicht vor,
  also eröffnet der nächste Abgleich denselben Fall wieder
  (`OFFENE_AUFGABEN.md:213`). 3b übernimmt das Verhalten und trägt es als
  geerbt in die Akte; eine Heilung ist ein Nachtrag der Fusions-Spec.
- **Nah an Python und darum Umzug:** `evidence.go` (449), `patch.go` (109),
  `frontmatter.go` (168), `format.go` (112), `lookup.go` (96), `cases.go`
  (155), `privacy.go` (55). `case.go` zieht **nicht** um — loomux hat die
  Fallakte seit 3a. `frontmatter.go` rendert über yaml.v3, Python über PyYAML
  `safe_dump`; ob die Ausgabe gleich ist, ist nicht geprüft.
- **Die aufgezeichneten Fälle von ultra-brain** (`bench/cases/{approve,case,cases}`,
  8 + 4 + 8) vergleichen nur stdout und Exitcode: kein `approve`-Fall hat ein
  `world_after`, keine Welt ein `.git`, und `approve-good` zeigt kein
  `committet als`. Sie belegen die geschriebenen Dateien und den Commit nicht.
- **Git-Identität beim Abspielen.** `BuildGitWorld` setzt Autor und
  Committer nur als Umgebung seiner eigenen Aufrufe
  (`internal/cases/gitworld.go:195-197`), und `gitenv` entfernt
  `GIT_AUTHOR_*`/`GIT_COMMITTER_*`. Ein `commit-tree` aus `approve` fände im
  abgespielten Fall keine Identität — auf beiden Seiten.
- **Kein Modell, kein Netz** in `apply`, `evidence`, `case`, `package`,
  `vcs`. Nur der `reconcile` nach dem Schreiben könnte das lokale Modell
  rufen; das ist Stufe 4, und bis dahin gilt die 3a-Regel `manual = true`.

### Bauweise (entschieden am 2026-09-22)

**Hybrid.** Die Pipeline von `approve` und `commit_paths` werden **neu aus
Python** geschrieben, mit den 126 Tests von `test_apply.py` und den 41 von
`test_vcs.py` als Orakel. `evidence`, `patch`, `frontmatter`, `format`,
`lookup`, `cases` und `privacy` **ziehen um** und werden, wo die Akte eine
Abweichung zeigt, an Python gehoben.

### Parität 3b

- **Neue Aufnahmen**, nicht die 20 alten: je Befehl Erfolg, Ablehnung, Fehler,
  und für `approve` mit `world_after` **und** einer Git-Welt, damit Seite,
  `log.md`, `audit.md`, Register und Commit verglichen werden.
- **Der Commit wird als Datei verglichen.** Der Rekorder blendet `.git` aus
  (Akte 3a, `:23`), und `InfraPath` tut es beim Abspielen. Nach dem Lauf
  schreibt der Harness darum `git.after` in die Welt: den Betreff des
  HEAD-Commits (`git log -1 --format=%s`), eine Zeile `%an <%ae> / %cn
  <%ce>` desselben Commits, jeden Pfad aus `git ls-tree -r --name-only
  HEAD`, jede Zeile aus `git diff --name-status HEAD` und jede Zeile aus
  `git status --porcelain=v1 --untracked-files=no`. Weder Commit-SHA noch
  Tree stehen darin: der Tree trüge die gestempelten Dateien, die keine
  Normalisierung reparieren kann; ihre Inhalte vergleicht `world_after`, und
  die Zeilen aus `git diff --name-status HEAD` (Arbeitsbaum gegen HEAD; ein
  Pfad, der im alten Index fehlt, erschiene als `D` — kein 3b-Fall legt eine
  Datei an) belegen, dass HEAD genau diese Inhalte trägt. Die
  Statuszeilen allein können das nicht: `approve` committet über einen
  Scratch-Index, der Index des Nutzers bleibt auf dem alten Stand, und die
  Statuszeile lautet `MM` bei richtigem wie bei falschem Commit-Inhalt.
- **Die Git-Identität** schreibt `BuildGitWorld` als lokale Konfiguration des
  Repos der Welt (`user.name`, `user.email`), nicht nur als Umgebung.
- **Neue Normalisierungen** in `cases.NormalizeState`, auf beiden Seiten
  gleich: der Prüfer `human:<Benutzer>` wird `human:{{USER}}`; ein
  `isoformat()`-Stempel **dieses Laufs** in `audit.md` und in der
  Frontmatter (`generated.at`, `verified[].at`) wird `{{NOW}}`, der Tag in
  `log.md` `{{TODAY}}`; `committet als <sha>` auf stdout wird
  `committet als {{SHA}}`. Was sonst einen Zeitstempel trägt, bleibt stehen.
  Der Stempel dieses Laufs ist der aus der Überschrift des Auditblocks, den
  die Ausgangswelt nicht hatte; getauscht wird er in jeder Datei, die von der
  Welt abweicht, und nur, wenn die Welt ihn nirgends enthält — ältere Blöcke
  bleiben so byte-gleich.
- **`[[stdout]]` im Import** ersetzt im aufgezeichneten stdout jedes
  Vorkommen von `from` durch `to`; für 3b `brain case --package ` →
  `loomux case --package `. Jede solche Regel ist eine Abweichung, die die
  Akte nennt.
- **Vergleichsklassen:** alle drei sind Daten — stdout exakt, Exit und
  Dateiwelt exakt, bei `approve` samt `git.after`; stderr frei. Auch bei
  `approve` ist stdout das Ergebnis (`Fall …: …`, `verworfene Behauptung`,
  `committet als`), die Warnungen stehen auf stderr.

## 3c im Einzelnen

- **`loomux brain check file|bundle|all`** — die drei Achsen aus
  `pkg/check/{okf,house,run}`: OKF-Form, Hausregeln, Föderation. Unter
  `brain`, weil `loomux check all` die Prüfkette aus 2a ist. `check code`
  fällt weg (Fusions-Spec #18). Referenz ist das Go-Binary vom Tag, denn eine
  Python-Form von `check` gibt es nicht.
- **`loomux lint --scope all|<scope>`** — heute verlangt `loomux lint` genau
  eine Datei; den Lint über das ganze Bündel hat nur `wiki-gate`, und nur
  zusammen mit der Driftprüfung. Die Regeln sind die zwölf von `lint.py`,
  als eigener Regelsatz neben `lint.go` (`internal/brain/wiki/sweep.go`),
  statt die Befunde von `house` abzubilden, denn Texte, Regelnamen und
  Auslöser weichen dort ab. `lint <datei>`, `wiki-gate` und die Lane
  `lint/wiki` behalten die Go-Form von 1a (Entscheidung vom 2026-09-23).
- **`loomux wiki init|types|retype`** — ein Bündel anlegen, die Seitentypen
  über alle Bereiche zählen (in Python `types`, das `census()` druckt), einen
  Typ in einem Bündel umbenennen. `census` und `scaffold` sind keine eigenen
  Befehle.

### Befunde und Entscheidungen 3c

Gegen den Code gelesen am 2026-09-23. Die Befunde B1–B18 und die
Entscheidungen E1–E8, freigegeben am 2026-09-23 (E5 geändert), stehen in
`parity/stufe-3c.md`.
- **Upkeep in `serve`** — ist der letzte `reconcile` älter als 24 h, holt
  `serve` ihn nach und hängt den Hinweis an die Antwort. Der Platz dafür ist
  seit 1b-2 frei.

## Parität

### Aufzeichnen

In `ultra-brain` vom Tag `loomux-3-source`, über `tools/cases.py`. Je Befehl
Erfolgs-, Ablehnungs- und Fehlerfälle. Aufgezeichnet werden mindestens:

| Befehl | Fälle |
|---|---|
| `reconcile` | leerer Tresor · unveränderte Quelle · veränderte Quelle wird Fall · stehender Fall wird abgelöst · Merge-Ereignis wird Fall · Merge-Fall tritt hinter Quellfall zurück · kein Prüfzentrum · zwei Prüfzentren · unlesbare Quelle · fehlendes Manifest |
| `reindex` | Durchgang grün, nichts offen · Durchgang eröffnet Fälle, es wird trotzdem indiziert · kein Prüfzentrum: Warnung, dann indiziert · Durchgang scheitert: Exit 1, nicht indiziert · Bereich ohne Manifest |
| `embed` | offene Vektoren · nichts offen · qmd nicht erreichbar |
| `area add` | neuer Bereich · `scope` schon registriert · ohne `--yes` nicht interaktiv · `--no-reindex` |

### Übersetzen

Beim Import, einmal: `brain reconcile` → `loomux reconcile`, `brain reindex` →
`loomux reindex`, `brain embed` → `loomux embed`, `brain init -y` →
`loomux area add -y`, `.brain.toml`/`.ultra-brain/config.toml` → `.loomux/config.toml`,
`%LOCALAPPDATA%\brain` → `…\loomux`, `BRAIN_STATE_DIR` → `LOOMUX_STATE_DIR`.
Der Originalfall bleibt als Beleg daneben.

### Vergleichsklassen

- **Daten** — `cases`, `case`, `check`, `lint`, `wiki census|types`,
  **`reconcile` und `embed`**: stdout exakt.
- **Meldungen** — `reindex`, `area add`, `approve`: Exit-Code und Dateiwelt
  exakt, Text frei.
- **Die Dateiwelt** wird bei Daten und Meldungen dieser Stufe gleich
  verglichen. Sie schließt das Prüfzentrum und `<state>/maintenance/` ein.
  **`<state>/areas/` braucht eine Normalisierung**, und die 1b-1-Akte hat für
  sie keine festgelegt, weil dort nichts schrieb.

  **Warum `reconcile` und `embed` Daten sind (Ruling 2026-09-22, Fix-Runde 1
  zu Task 16).** Die Zählungen eines Abgleichs — geprüfte Quellen, gehashte
  Quellen, eröffnete Fälle mit ihrer Adresse — sind das Ergebnis des Befehls
  und kein Meldungstext; `_reconcile` druckt sie auf stdout, und
  `_report_unreadable` hält alles andere auf stderr. Als Meldung verglichen
  überlebte eine falsch gezählte Ausgabe (`Checked+1`, `Hashed+7`) die ganze
  Fallsuite, und der Befund zum leeren `[index] include` war nur von Hand an
  genau diesen Zahlen zu sehen. `embed` hat kein stdout; verglichen wird,
  dass es keines hat. `reindex` bleibt Meldung: sein stdout ist ein Satz über
  den Lauf, nicht sein Ergebnis — das steht in der Dateiwelt. `area add`
  ebenso.

  **Festgelegt in Task 16 (2026-09-22)**, `cases.NormalizeState`, auf beide
  Seiten gleich angewandt, und nur an drei Stellen:

  1. **Der Zeitpunkt des Durchgangs.** Der Stempel in
     `maintenance/last-run.txt` wird `{{NOW}}` — in dieser Datei und überall,
     wo **genau dieser** Stempel wiederkehrt (`created` einer Fallakte,
     `generated.at` eines Pakets). Der Tag, den er nennt, wird `{{TODAY}}`,
     aber nur dort, wo eine Fall-ID ihn trägt (`<segment>-<tag>-<4 hex>`), in
     Pfaden, in Dateien und **auf stdout** (drei `reconcile`-Fälle drucken
     die Fall-ID); der Tag jeder Seite kommt aus ihrem eigenen
     `last-run.txt`. Geprüft: die Aufzeichnung auf den Vortag umgeschrieben,
     alle Fälle bleiben grün. Getauscht wird nur ein Stempel in der Form, die
     beide Schreiber erzeugen: `isoformat()` eines UTC-Zeitpunkts, sechs
     Nachkommastellen oder keine, `+00:00`. Ein Stempel mit Ortszone, mit `Z`
     oder mit Millisekunden bleibt stehen und fällt durch.
  2. **Die Änderungszeit im Stat-Zwischenspeicher**
     `maintenance/<collection>/stats.tsv`, Spalte `mtime_ns`: Sie ist der
     Zeitpunkt, zu dem die Welt gestagt wurde, und unterscheidet sich zwischen
     Aufzeichnung und jedem Abspielen. Getauscht wird nur eine
     Nanosekundenzahl (`^\d{19}$`); ein Wert in Sekunden oder Millisekunden
     bleibt stehen und fällt durch — ein Python-Lauf nach einem solchen
     loomux-Lauf hasht jede Quelle neu. Pfad und Größe bleiben.
  3. **Eine vom Indexlauf geprägte `doc_id`**: eine Zeile eines
     `_identities.tsv`, deren ID in keinem Register der Ausgangswelt steht,
     bekommt `{{DOCID}}`, und die Zeilen eines solchen Registers werden
     sortiert — das Register ist nach ID geordnet, also ist die Stelle einer
     zufälligen ID so zufällig wie die ID. Ein Register ohne neue ID behält
     seine Reihenfolge byte-genau. Nur eine wohlgeformte ID (26 Zeichen
     Crockford) wird getauscht, und nur eine, die auf **genau einer** Zeile
     der Register des Baums steht: eine doppelt vergebene neue ID bliebe
     sonst unsichtbar.

  **Kataloge tragen keinen Zeitstempel** (`RenderCatalog`, `_write_catalogs`)
  und werden nicht normalisiert; der Satz „Identitätsregister und Kataloge
  tragen Zeitstempel", der hier stand, war falsch. Das Register trägt die
  Zeit nur verdeckt, im Zeitteil einer neu geprägten ID.

  **Dazu, ohne Normalisierung:** Ein Fall ohne `world_after` wird gegen seine
  `world` gehalten — der Rekorder schreibt ein `world_after` nur, wo der Lauf
  etwas geändert hat, also behauptet sein Fehlen „nichts geändert". Ohne
  diese Regel bliebe eine Ablehnung, die trotzdem schreibt, grün.
- **Die Fallsuite 1b-1 bleibt grün.** Sie ist der Nachweis, dass die
  Umstellung von `legacy.go` die Leser nicht bricht; wo eine ihrer Welten
  `LOOMUX_LEGACY_BRAIN_DIR` setzt, muss sie unter „neu zuerst, alt als
  Rückfall" dieselbe Antwort geben.
- **Git-Welten** — Fälle mit einem echten Repo laufen über
  `internal/cases/gitworld.go` aus 2c. Ist 2c beim Bau noch nicht gemergt,
  bleiben diese Fälle geparkt und stehen mit Grund in der Akte; 3a baut
  **keine** zweite Git-Welt daneben.

### Abweichungsliste

`docs/.superpowers/parity/stufe-3a.md`. Bereits bekannt:

| Fall | Alt | Neu | Begründung |
|---|---|---|---|
| Sperre | PID in der Sperrdatei, Übernahme nach Frist (`locking.claim`) | Betriebssystem-Handle, leere Sperrdatei (`internal/lock`) | Ein sterbender Prozess gibt das Handle ohne Aufräumen frei; der gehaltene Bytebereich wäre ohnehin unlesbar. Portiert in Stufe 1a, hier nur festgehalten |
| Zustandsort | Artefakte und Stempel unter `%LOCALAPPDATA%\brain` | Geschrieben nach `LOOMUX_STATE_DIR`, alt nur gelesen | Die befristete Ausnahme der 1b-1-Akte lief „bis Stufe 3"; 3a ist der erste Schreiber |
| `local_only`-Bereich in `reconcile` | Ollama-Proposer erzeugt einen Vorschlag neben dem Fall | Fall mit `manual = true` und `note`, kein Vorschlag | Das lokale Modell ist Stufe 4. Ein Fall ohne Vorschlag bleibt entscheidbar; die Feldsemantik ist dafür da (`case.py:52-58`). Fällt mit Stufe 4 weg |

## Fehlerverhalten

- **Kein stiller Fortschritt am Tor vorbei.** Nicht dadurch, dass `reindex`
  bei Fällen abbräche — das tut die Referenz nicht —, sondern dadurch, dass
  jeder Zustand, in dem das Register ohne Abgleich vorrückt, **laut** ist:
  aufgezählte Fälle, die Warnung ohne Prüfzentrum, Exit 1 bei gescheitertem
  Durchgang. Ein Bereich ohne Manifest wird still übersprungen (das ist der
  Normalfall vor der ersten Erklärung); ein unlesbares Manifest bricht ab.
- **Registry und Fallakten** werden atomar und unter Sperre geschrieben.
  Scheitert das Umbenennen endgültig, ist der Befehl rot; es bleibt keine
  halbe Datei.
- **Unlesbare Quellen** sind kein Abbruch: sie stehen als `unreadable` im
  Bericht, mit Pfad.
- **`.loomux/config.toml` bleibt für Agenten unbeschreibbar.** `area add`
  schreibt sie — als Befehl, den ein Mensch startet; die Schreibschranke des
  Wächters bleibt unberührt.
- **Fehlt `qmd`**, nennt `embed` Namen und Installationsbefehl und exitet
  ungleich 0. Nie eine leere Erfolgsmeldung.

## Tests und Tore

- TDD je Task, 100 % Coverage, jeder Ausschluss begründet. Umgezogene Pakete
  (`pkg/index`) bringen ihre Tests mit und werden beim Umzug gehoben.
- **Mutationsrunde der Stufe** über die Entscheidungspakete:
  `brain/maintenance` (die Fallbildung) und die Registry-Schreibseite.
  Überlebende stehen mit Begründung oder nachgereichtem Test in der Akte.
- **Startzeit.** `reindex`, `reconcile` und `area add` sind neue Importe in
  `cmd`. `GODEBUG=inittrace=1 loomux --version` wird nach 3a erneut gelesen;
  keine Paketvariable dieser Pakete parst eingebettete Daten.

## Selbstnutzung

Das loomux-Repo ist sein eigener Bereich. Nach 3a läuft `loomux reconcile` im
Repo selbst über `docs/wiki`, und `loomux reindex` ersetzt den Aufruf des alten
`brain`. Damit ist Bedingung 5 der Fertigstellung erfüllt, bevor 3b beginnt.

**Die Fälle von loomux landen in `brain-knowledge`.** Das Prüfzentrum ist der
eine Ort des Tresors, und das ist `95 Prüfzentrum/` in `brain-knowledge` — die
Fusions-Spec führt dasselbe Repo unter „Befunde" als Risiko: kein Remote,
letzter Commit 2026-09-07, und genau dieses Verzeichnis unversioniert. Die
Selbstnutzung von 3a schreibt also in ein Repo, dessen Sicherung offen ist.
Das ist **kein Bauhindernis** für 3a, aber es gehört vor die Umstellung der
Wirte gelöst, wie die Fusions-Spec schon sagt.

## Messen

In `docs/de/benchmarks.md` und `docs/en/benchmarks.md`, mit
`loomux dev bench-hooks` bzw. der Zeitmessung des Befehls:

| Fall | Vergleich |
|---|---|
| `reconcile` über den loomux-Bereich, warm | gegen `uv run brain reconcile` |
| `reindex` über den loomux-Bereich, kalt und warm | gegen `uv run brain reindex` |
| `area add` auf einem leeren Repo | gegen `uv run brain init -y` |

Ein Zielwert wird **nicht** vorab gesetzt: die Python-Form ist hier
Sekundenware, und der Gewinn ist der Startboden, nicht eine Feinoptimierung.
Gemessen wird trotzdem, weil die Zahl in die Akte gehört.

## Fertig, wenn

Für 3a, nach den fünf Bedingungen der Fusions-Spec:

1. alle übersetzten Fälle von 3a grün oder freigegeben in `parity/stufe-3a.md`,
2. Coverage 100 %, jeder Ausschluss begründet,
3. die Mutationsrunde über `brain/maintenance` und die Registry-Schreibseite
   gelaufen, Überlebende dokumentiert,
4. die drei Messungen eingetragen,
5. das loomux-Repo fährt `loomux reconcile` und `loomux reindex` auf sich selbst.

## Reihenfolge für den Plan (3a)

1. **Task 0** — Arbeitsort prüfen; Merge-Stand von 2c feststellen; klären, ob
   `tools/cases.py` eine Welt mit `.git` unverändert herstellt.
2. `lock.ReplaceText`.
3. `config/legacy.go` auf „neu zuerst, alt als Rückfall"; die Fallsuite 1b-1
   bleibt grün.
4. Registry-Schreibseite in `config`.
5. Umzug `pkg/index` → `internal/brain/index` (`walk`, `document`, `reindex`,
   `qmd_config`, Katalog-Schreibseite), auf 100 % gehoben.
6. `loomux reindex` und `loomux embed` in der CLI, noch ohne Auffangdurchgang.
7. `internal/brain/vcs`, Leseseite.
8. `events.go` — Protokollformat, lesen, ablegen.
9. `case.go` — Fallakte lesen, schreiben, `case_id`, `case_dir`.
10. `derive.go` — Abhängige einer Wiki-Seite.
11. `reconcile.go` — Scan und Quellfälle.
12. `reconcile.go` — Zusammenführungsfälle, Ablösung, Entdoppelung.
13. `loomux reconcile` in der CLI.
14. Der Auffangdurchgang in `reindex`.
15. `loomux area add`.
16. **Halt** — der Mensch setzt den Tag `loomux-3-source` in `ultra-brain`.
    Danach Fallsuite 3a: Aufzeichnen, Übersetzen, Abspielen.
17. **Halt** — der Mensch trägt `[area]` und `[layout]` des loomux-Repos in
    `.loomux/config.toml` ein — **ohne `[layout].review`**: das Prüfzentrum des
    Tresors ist `95 Prüfzentrum/` in `brain-knowledge`, und ein zweiter
    erklärter Ort wäre ein Fehler, kein Vorrang.
18. Selbstnutzung: loomux fährt `reconcile` und `reindex` auf sich.
19. Mutationsrunde und Messungen.

## Nachzutragen in der Fusions-Spec

Absichtlich **hier** und nicht dort: 2b hat dieselbe Datei auf seinem Zweig
schon geändert. Einzutragen, sobald 2b und 2c gemergt sind — **am 2026-09-22
eingetragen** (`b9c1a7e`), nachdem beide gemergt waren:

1. **Reihenfolgetabelle:** Stufe 3 rückt auf Prio 1 und läuft parallel zu 2b
   und 2c; der Grund ist, dass 3 an keiner der beiden hängt und die größte
   Stufe ist.
2. **Teilstufentabelle:** 3 zerfällt in 3a, 3b, 3c, wie oben.
3. **Freigabespalte:** #1, #2 und #3 freigegeben am 2026-09-19, Stufe 3.
4. **Korrektur:** „Locking" gehört nicht mehr zum Inhalt von Stufe 3 —
   `internal/lock` ist die Portierung, fehlend war nur `ReplaceText`.
5. **Korrektur:** „`merge-events`" ist das Ereignisprotokoll, kein Befehl; der
   Befehl ist `hook` und steht als Nachtrag #5 bei Stufe 4.
6. **Nachtrag #17:** `reindex` und `embed` als Befehle
   (`ultra-brain/pkg/index`, `cli.py:463,467`) — Vorschlag Stufe 3a,
   freigegeben am 2026-09-19. Begründung: der Auffangdurchgang koppelt
   `reconcile` an `reindex`, und `embed` ist ohne `reindex` gegenstandslos.

## Offen und vor dem Bau zu messen

- Ob `tools/cases.py` ein `.git` in `world/` unverändert durchreicht
  (Task 0). Wenn nicht, braucht der Rekorder eine kleine Ergänzung, und die
  gehört vor Task 7.
- Ob die qmd-Vektoren des Altbestands wiederverwendbar sind oder `reindex` +
  `embed` je Bereich einmal fällig wird. Die Fusions-Spec stellt die Frage
  beim Datenumzug; 3a kann sie zum ersten Mal beantworten.
