# Ein Wiki-Standard für die ganze Flotte

**Datum:** 2026-09-10
**Stand:** entworfen, nicht umgesetzt
**Betrifft:** `ultraloom`, `ultra-brain`, und über den Standard jedes Projekt,
das `ulinit` aufsetzt — heute `space`, `ecoflow`, `iam_backend`,
`iam_frontend`, `iam_workers`, `iam_wiki`

## Ziel

Eine Definition, die in jedes Projekt kopiert werden kann und die `ulinit`
selbst aufsetzt: **Projektdokumentation bleibt im Projekt, übergreifendes
Wissen liegt in `brain-knowledge`.** Gleich für alle Projekte, und trotzdem
abwählbar — ein Entwickler, der sein Projektwiki lieber in den Vault legt oder
in ein Schwesterrepo, muss das weiter können.

Der Weg dahin ist **keine neue Struktur.** Zwei Mechanismen, die es schon gibt,
werden zusammengeführt: `.ultraloom/answers.toml` ist bereits die eine
handgepflegte Entscheidungsdatei eines Projekts, aus der `ulinit` alles andere
erzeugt; und `ulinit` weiß bereits, wie man einen brain-Hook schreibt. Was
fehlt, ist dass die Antwortdatei auch brains Entscheidungen trägt und
`.brain.toml` daraus fällt.

## Ausgangslage, vermessen am 2026-09-10

### Kein Projekt der Flotte schreibt Wikiseiten in die Pflegeschleife

| Projekt | Registry | brain-Konfig | Seiten im Repo-Wiki | Fälle |
|---|---|---|---|---|
| `ultraloom` | ✓ | `.ultra-brain/config.toml` | 0 (fünf Gerüstdateien) | 0 |
| `ultra-brain` | ✓ | `.brain.toml` | **24**, OKF-streng | **13** |
| `space` | ✓ (`readonly`) | `.brain.toml` | **204**, eigene Typen | **0** |
| `ecoflow` | ✓ | `.brain.toml` | — | 0 |
| `iam_wiki` | ✓ | `.brain.toml` | — | 0 |
| `iam_backend` | **✗** | **—** | — | — |
| `iam_frontend` | **✗** | **—** | — | — |
| `iam_workers` | **✗** | **—** | — | — |

`space` ist der Lehrfall: 204 Seiten, davon 190 mit `type:` und 187 mit
`sources:` — und **null** Fälle in der Warteschlange. Zwei Gründe, beide
gemessen:

1. **Zwei Werkzeuge schauen an zwei Orte.** `brain status` liest das Manifest
   und antwortet `space/docs/wiki` mit 204 Seiten. `reconcile._cases` liest
   `area.wiki_path` aus der **Registry**, und dort steht
   `brain-knowledge/91 Projekte/space` — ein leeres Gerüst.
2. **Die Seiten tragen keine Identitätsfelder.** `dependents()` gruppiert über
   `source.doc_id`; `space`s `sources[]` haben `id`, `resource` und `title`,
   aber kein `doc_id`, `content_hash` oder `revision`. Selbst bei gleichen
   Pfaden entstünde keine Kante.

### Fünf Defekte, die vor dem Standard stehen

**D1 — Go und Python indizieren verschieden.** `globToRegex`
(`ultra-brain/pkg/index/walk.go:90`) prüft `HasPrefix(p, "**/")` vor
`HasPrefix(p, "/**")`. Bei `docs/**/*.md` bleibt nach `docs` ein `/**/*.md`
stehen, das den ersten Zweig verfehlt und in `/**` → `/.*` fällt; der Ausdruck
wird `^docs/.*/[^/]*\.md$` und verlangt ein Zwischenverzeichnis. Python macht es
über `PurePosixPath.full_match` richtig (`src/brain/walk.py:117`).

Nachgemessen: Go-`brain reindex` ergab 68 Einträge **ohne**
`docs/benchmarks.md`, `docs/benchmarks.de.md`, `docs/hooks.md`,
`docs/hooks.de.md`; `brain-mcp reindex` danach 72 **mit** ihnen. Der Zustand von
`_identities.tsv` und `graph.json` hängt also davon ab, welches Binary zuletzt
gelaufen ist. Tragweite über dieses Repo hinaus: `brain init` schreibt genau
dieses Muster als Vorgabe (`write_manifest`, sobald `sources == "docs"`).

**D2 — `NeighbourWiki` findet das Wiki eines fremden Projekts.**
`internal/detect/edges.go:37` akzeptiert jedes Schwesterverzeichnis, das auf
`_wiki` endet und ein `.git` hat; ausgeschlossen wird nur das Projekt selbst
(`name == projectDir`), nie geprüft, dass es `<projekt>_wiki` heißt — obwohl der
Docstring die Konvention behauptet. Unter `#GIT/` gibt es genau ein solches
Verzeichnis, `iam_wiki`, und das ist ein **anderes Projekt**: brains Registry
führt es als `project/iam-wiki` mit `readonly = true`.

Folge, in der Flotte gemessen: `ultraloom/.ultraloom/answers.toml` steht auf
`mode = "neighbour_repo"`, `bundle = "iam_wiki/"`. **Das ist der Grund, warum
`brainEntry` hier nie die brain-Einträge geschrieben hat** — es schaltet auf
`Mode == "brain"` (`cmd/init/run.go:704`). Nicht „niemand hat die Mode
gesetzt", sondern „ein Detektor hat die falsche gesetzt". Hätte `ulinit`
daraufhin `brain wiki-gate` installiert, hätte das Gate gegen ein fremdes,
readonly registriertes Repo geprüft und grün gemeldet — schlechter als kein
Gate.

**Die Erkennung setzt das aber nicht durch, sie schlägt es vor.** Vier Daten,
die das zeigen und die eine erste Fassung dieses Absatzes widerlegt haben:
`NeighbourWiki` hängt seit `c28e71c` (2026-08-28) im Erkennungsweg, `iam_wiki`
existiert als Repo seit 2026-07-27, `ultraloom`s Eintrag stammt aus `e84df9e`
(2026-08-29) — und `space` wurde mit `4f272921` **zwei Tage später**
aufgesetzt, bei gleichem Code und gleichem Schwesterrepo, und trägt
`mode = "brain"`. Dazwischen stand das Interview: bei `space` hat ein Mensch
den Vorschlag überstimmt, bei `ultraloom` wurde er genommen. Der Defekt ist
damit die **Vorgabe**, nicht ein Zwang — und daher genau die Sorte, die in
einem unbeaufsichtigten `--yes`-Lauf durchgeht.

Eine erste Fassung sagte hier, `gather` weise das Ergebnis **bedingungslos**
zu. Das war falsch gelesen: `gather` (`cmd/init/main.go`) fragt den Nachbarn
nur, wenn `Detect` keine Mode gefunden hat (`if facts.WikiMode != "" {
return facts, nil }`). Die Lücke liegt eine Ebene tiefer, in `readWiki`
(`internal/detect/detect.go`): es liest nur `.brain.toml`. `ultraloom` trägt
`.ultra-brain/config.toml` — den Namen, den brain selbst **zuerst** sucht
(`manifestNames` in `ultra-brain/pkg/config/manifest.go`) — mit `wiki = true`,
und sein `docs/wiki/index.md` hat den Marker `okf_version` nicht. `Detect`
findet also nichts, meldet eine Unklarheit, und erst dann kommt
`NeighbourWiki` zum Zug.

Bemerkenswert und für den Entwurf entscheidend: für `iam_backend`,
`iam_design`, `iam_docs`, `iam_frontend` und `iam_workers` ist `iam_wiki/`
**richtig**, obwohl keines `iam_backend_wiki` heißt. Die gelebte Konvention
ist nicht `<projekt>_wiki`, sondern `<familie>_wiki`: ein Wiki gehört den
Geschwistern, die `<familie>` heißen oder mit `<familie>_` beginnen.
`ultraloom` und `space` beginnen nicht mit `iam_`. Das ist der Unterscheider,
und er braucht keine neue Datenquelle.

Ein Abgleich gegen brains Registry, den eine erste Fassung von Stufe 0
vorschlug, **kann nicht unterscheiden**: `iam_wiki` ist dort ein eigener
Bereich (`project/iam-wiki`), und kein Eintrag verbindet ihn mit
`iam_backend`. Die Registry sagt über den falschen Fall (`ultraloom`) und den
richtigen (`iam_backend`) dasselbe.

**D3 — `configure_agent_hooks` löscht fremde Hookeinträge.**
`ultra-brain/src/brain/init.py:234` weist zu statt anzuhängen:
`hooks_dict["PreToolUse"] = [ein Eintrag]`. In `ultraloom` hätte
`brain init --agents claude` damit `ulguard --root "${CLAUDE_PROJECT_DIR}"`
gelöscht und den Matcher von `Write|Edit|NotebookEdit|Bash|PowerShell` auf
`Edit|MultiEdit|Write|Bash` beschnitten. Der Gemini-Zweig (`:270`) macht
dasselbe unter dem Schlüssel `wiki-guard`. Nicht gefahren, durch Lesen und
Auszählen der jetzigen Liste belegt.

**D4 — `brain init`s Flags `--no-hook` und `--no-reindex` sind wirkungslos.**
`install_hook` und `run_reindex` stehen nur in der Signatur von `run_init` und
werden im Rumpf nie benutzt; einen Git-Hook-Installer gibt es in `init.py`
überhaupt nicht, obwohl das Manifest, das es schreibt,
`[maintenance] on_merge = true` setzt. `--agents` ist standardmäßig die leere
Menge, ein blankes `brain init -y` installiert also keine Schranke.

**D5 — `ultraloom sync` existiert nicht.** `installed.toml` sagt es selbst:
„init skips a file that is already here … so a second run cannot update the pin
below. Only `ultraloom sync` will, and it does not exist yet." Eine geänderte
Antwort wandert damit heute nicht in die generierten Dateien.

## Die Flottendefinition

### Routing

Zwei Sätze, und sie stehen so schon in `AGENTS.md`:

- Wissen, das nur für dieses Projekt gilt — Architektur, Entscheidungen,
  Messungen, Betriebswissen — nach `<repo>/docs/wiki`.
- Wissen, das ein zweites Projekt genauso brauchen könnte — Werkzeuge,
  Sprachen, Verfahren, Fremdprodukte — nach `brain-knowledge/92 Engineering/*`.
  Im Zweifel geteilt, und aus dem Projekt per Verweis darauf zeigen.

Der geteilte Bereich ist schon registriert (`engineering/python`,
`engineering/craft`, beide `shared = true` und `path == wiki`) und braucht von
einem Projekt-Init nichts.

Bereichsübergreifende Zitation trägt, gemessen: `reconcile` sammelt `changed`
über **alle** Bereiche (`changed.update(area_changed)`) und hält es gegen die
`dependents()` jedes Bereichs. Eine Seite in `engineering/python`, die eine
ultraloom-Quelle zitiert, bekommt ihren `source_changed`-Fall. Der Verweis ist
`brain://project/<scope>/<pfad>` plus `doc_id` aus dem `_identities.tsv` des
Quellbereichs.

### Die vier Wiki-Arten

`answers.WikiModes` kennt heute drei; die vierte fehlt und wird gebraucht, weil
`space`s Form sonst nicht ausdrückbar ist:

| Mode | Wiki liegt | Registry | Für wen |
|---|---|---|---|
| `brain` | `<repo>/docs/wiki` | `wiki = <repo>/docs/wiki`, nicht `readonly` | **Die Flottenvorgabe** |
| `neighbour_repo` | `<projekt>_wiki` daneben | eigener Bereich | iam |
| `vault` (**neu**) | `brain-knowledge/91 Projekte/<n>` | `readonly = true` | wie `space` heute registriert ist |
| `none` | — | kein Eintrag | Projekte ohne Wiki |

`vault` ist die Abwählbarkeit, die der Auftrag verlangt: ein Entwickler, der
sein Projektwiki nicht im Repo will, wählt sie, und `ManifestDir` schickt dann
auch die Katalogartefakte nach `stateDir/areas/…` statt in die Repowurzel.

**Folge für `brain`, die festgehalten sein muss:** `mode = "brain"` heißt
ausdrücklich **nicht** `readonly`. Der Preis ist, dass `index.md`, `graph.json`
und `_identities.tsv` in der Repowurzel liegen — sie gehören dorthin,
`ManifestDir` gibt für einen schreibbaren Bereich `area.Path` zurück und
`approve.go:422,427` liest `_identities.tsv` von derselben Wurzel mit
bereichswurzel-relativen Pfaden. Das ist kein Defekt und nicht zu reparieren,
sondern in die `.gitignore` zu schreiben oder einzuchecken.

## Eigentümerregel

Vom Nutzer entschieden am 2026-09-10:

- **`ulinit` besitzt und verwaltet jede Hostdatei:** `.claude/settings.json`,
  `.agents/hooks.json`, `.mcp.json`. Es darf fremde Einträge wrappen und
  **nichts löschen**.
- **`ultra-brain` besitzt seine eigenen Sachen:** Manifest, Registry, das
  OKF-Bundle, das Prüfzentrum.
- **`ulinit` installiert brain mit.**

Das Verhalten existiert schon: `internal/settings/merge.go` (460 Zeilen plus
479 Testzeilen) fasst nur Einträge mit der Marke `ultraLoomOwned` an, gibt
fremde byteweise zurück, vergleicht über `toolKey` statt über die
Kommandozeile — damit ein geändertes Kommando derselbe Eintrag bleibt — und
gibt den eigenen Eintrag auf, wenn ein fremder vor allen eigenen steht, statt
zu verdoppeln. Der Grund steht im Paketkommentar: zwei parallele Läufe hingen
am 2026-08-27 über Nacht.

**Der Aufruf braucht keinen Edit drumherum.** Gemessen: `configure_agent_hooks`
läuft nur bei gesetztem `--agents`, und die Vorgabe ist die leere Menge. Also
fasst `brain init -y` **ohne** `--agents` keine Hostdatei an. Die einzige echte
Überschneidung ist `.mcp.json`, das `configure_mcp` ungebunden immer schreibt.

Offene Entscheidung, in der Spec benannt statt überlesen: `brain init` schreibt
den Routing-Absatz „Wohin welches Wissen gehört" in `AGENTS.md`, eine kuratierte
Datei des Projekts. Einmalig und am Überschriftstest erkannt, also nicht
wiederholend — aber ein Fremdwerkzeug in einer Projektdatei.

## Entwurf

### Teil 1: `answers.toml` wird die eine Entscheidungsdatei

`answers.Answers` bekommt, was brain braucht. Alles davon steht heute in
`.brain.toml` und ist damit vermessen:

```toml
[gates.wiki]
mode   = "brain"          # brain | neighbour_repo | vault | none
bundle = "docs/wiki/"
scope  = "project/ultraloom"   # neu; heute [area] scope
sources = "docs"               # neu; heute [layout] sources
privacy = "manual_cloud"       # neu; heute [privacy] mode
```

`ulinit` **generiert** daraus `.brain.toml` — mit der Kopfzeile, die
`policy.toml` schon trägt: `# generated from .ultraloom/answers.toml -- edit
that and re-run init`. Das ist kein neues Verfahren, sondern der Grundsatz, auf
dem `ultraloom` gebaut ist; `answers`' Paketkommentar sagt ihn: „Decisions, not
output: everything else this tool writes is derived from this type."

Der Include wird beim Generieren so geschrieben, dass D1 nicht beißt:
`include = ["docs/*.md", "docs/**/*.md", "README*.md"]`. Nachgemessen ergeben
Go und Python damit auf acht Testpfaden dasselbe Ergebnis. Das ist ein
Umweg, nicht die Reparatur — D1 bleibt eigene Arbeit.

**Wer sein Manifest von Hand pflegen will, behält es.** `ulinit` überspringt
eine vorhandene Datei, wie überall sonst; `space`s `.brain.toml` mit ihren
begründeten Ausschlusslisten ist genau dieser Fall und bleibt unberührt.

### Teil 2: das Flottenpreset

Eine Vorlage in `ultraloom`, geschrieben von `ulinit --preset fleet`: die
Standardantworten mit `mode = "brain"`, `bundle = "docs/wiki/"`,
`agents = ["claude", "gemini"]` und dem Include aus Teil 1. Das ist „einmal
definieren, in alle Projekte kopieren", ohne dass ein bestehender Pfad umzieht.

Die verzeichnete Antwort schlägt weiterhin das Flag (`run.go:449`) — ein Preset
über ein Projekt, das schon geantwortet hat, wird als ignoriert gemeldet und
ändert nichts. Das ist richtig so und der Grund steht als datierter Befund
darüber: bis 2026-08-28 rannte `--coverage-threshold 55` über ein auf 100
verzeichnetes Projekt und meldete Erfolg gegen eine Zahl, die in der eigenen
Datei nicht stand.

### Teil 3: die Hookeinträge

`ulinit` schreibt drei brain-Einträge, wenn `mode == "brain"` und
`brainpath.Find` `brain` auf dem PATH findet:

| Datei | Ereignis | Kommando |
|---|---|---|
| `.claude/settings.json` | PreToolUse | `brain guard` |
| `.claude/settings.json` | Stop | `brain wiki-gate --root <wurzel>` |
| `.agents/hooks.json` | PreToolUse | `brain guard` |

Zwei Dinge, die aus c0s Messungen folgen und die dieser Entwurf einhalten muss:

- **In `.agents/hooks.json` steht das Kommando beim Namen.** Das
  Arbeitsverzeichnis eines Antigravity-Hooks ist das Verzeichnis der
  `hooks.json`, also `.agents/`, nicht die Projektwurzel; jeder relative Pfad
  geht ins Leere.
- **Keine `${CLAUDE_PROJECT_DIR}` auf der Antigravity-Seite.** Ob Antigravity
  die Variable setzt, ist **ungemessen** — sie ist eine von Claude Code, und
  `MIGRATION.md:157` nennt `${CLAUDE_PLUGIN_ROOT}`, nicht sie. Ein Hook, der
  die Wurzel braucht, sucht sie aufwärts bis zur ersten Konfiguration.

Antigravity hat außerdem **kein `SessionStart` und kein `SubagentStop`**
(`MIGRATION.md:133-142`), und im Print-Modus (`agy -p`) feuern `Stop` und
`PreInvocation` belegt nicht. PreToolUse ist dort ungemessen, weil der
Print-Modus jedes freigabepflichtige Werkzeug verweigert und der Hook nie einen
Aufruf zu bewachen hatte. Für `brain guard` heißt das: im `agy -p`-Pfad
gegenstandslos, im interaktiven Pfad unbelegt. Nachweisen kann das nur ein
Mensch in einer interaktiven agy-Sitzung.

**Ein Namenswiderspruch, der zu entscheiden ist:** `hookEntries` schreibt
`brain guard` heute mit `ultraLoomOwned: true`. Bildet `ulinit` den Eintrag —
wie dieser Entwurf es will —, ist die Marke korrekt und nur der Satz
„ultra-brain besitzt ihn" falsch. Die Marke bleibt, die Regel wird präzisiert:
`ulinit` besitzt den *Eintrag*, `ultra-brain` das *Werkzeug*.

## Nachweis

Der Standard ist nicht bewiesen, wenn `ulinit` durchläuft, sondern wenn die
Pflegeschleife sich schließt. Die Kette, in dieser Reihenfolge:

1. `ulinit --preset fleet` in einem Projekt, das noch nicht geantwortet hat.
   `.brain.toml` entsteht, die drei Hookeinträge stehen, `ulguard` ist
   unversehrt — geprüft durch Auszählen der `PreToolUse`-Liste, nicht durch
   Augenschein.
2. `brain reindex` mit **beiden** Binaries hintereinander liefert dasselbe
   `_identities.tsv`. Das ist der Test auf D1 beziehungsweise auf den Umweg.
3. Eine Seite unter `docs/wiki/topics/` mit vollständigem `sources[]`, dann
   `brain lint`, dann `brain check bundle --scope <scope>`.
4. Die zitierte Quelle ändern, `brain-mcp reindex`, `brain cases`: **es muss
   ein `source_changed`-Fall für die Seite erscheinen.** Erst das beweist, dass
   die Schleife zu ist — und genau das ist heute in keinem Projekt der Flotte
   der Fall.

## Stufen

| Stufe | Inhalt | Hängt an |
|---|---|---|
| 0 | D2 reparieren: `readWiki` liest `.ultra-brain/config.toml` vor `.brain.toml`, wie brain selbst; `NeighbourWiki` nimmt nur ein Wiki der eigenen Familie (`<familie>_wiki` für `<familie>` oder `<familie>_*`). Dazu `ultraloom`s eigene `answers.toml` auf `brain` / `docs/wiki/`. Plan: `docs/.superpowers/plans/2026-09-11-wiki-flottenstandard-stufe-0.md` | — (die Sperre durch den Go-Hooks-Zweig ist mit `6168ef2` gefallen) |
| 1 | Teil 1: `answers.Answers` erweitern, `.brain.toml` generieren, `vault` als vierte Mode — `vault` zunächst nur als gültiger Wert, ohne eigenes Manifest. Plan: `docs/.superpowers/plans/2026-09-11-wiki-flottenstandard-stufe-1.md` | Stufe 0 |
| 2 | Teil 3: die drei Hookeinträge, `.agents/hooks.json` als zweiter Schreiber | Stufe 1 |
| 3 | Teil 2: das Preset. Vorher muss `Load` `bundle` prüfen — relativ, mit Vorwärtsschrägstrichen —, denn ein Bundle mit Backslashes oder ein absolutes wird gerendert, aber brains `wiki_layout` weist es ab (Schlussreview Stufe 1) | Stufe 1 |
| 4 | D1 und D3 in `ultra-brain` — Glob-Parität und `configure_agent_hooks`, das anhängt statt zuzuweisen. Vorlage ist `internal/settings/merge.go` | eigene Runde, anderes Repo |
| 5 | Ausrollen: die drei iam-Coderepos aufnehmen, `space`s Riss zwischen Registry und Manifest entscheiden. Vorbedingung: `scope` muss aus dem Verzeichnisnamen des Hauptcheckouts kommen, nicht aus dem eines verlinkten Worktrees — `projectName` nimmt `filepath.Base` der Wurzel, ein erster Lauf in `.worktrees/<name>` hält also `project/<name>` fest (Schlussreview Stufe 1, Important 4) | Stufen 1 bis 4 |

D4 und D5 sind benannt und nicht eingeplant: D4 ist ein Befund für
`ultra-brain`, D5 (`ultraloom sync`) ist eigene Arbeit und begrenzt bis dahin
Teil 1 auf den Erstlauf.

## Außerhalb des Schnitts

- **Die 187 `space`-Seiten mit Identitätsfeldern nachrüsten.** Der größte
  Bestand der Flotte, und ohne `doc_id` für die Pflegeschleife unsichtbar. Das
  ist eine eigene Spec, und ihre erste Frage ist, ob es sich lohnt.
- **Ob `space` seine eigenen Seitentypen behält.** `type: Design Decision` ist
  keiner der vier OKF-Typen. 204 Seiten sagen, dass es trägt; eine Umstellung
  wäre teuer und ihr Nutzen ist unbelegt.
- **Der Codex-Host.** Auf dieser Maschine nicht installiert, sein Hook-Vertrag
  von hier nicht nachrechenbar. Bei c0 eine Naht mit einem Test, der belegt,
  dass ein unbekannter Host geschlossen fällt; hier genauso.
- **Der doppelte Manifestname.** `.ultra-brain/config.toml` gegen `.brain.toml`
  bleibt, wie es ist. Sobald `.brain.toml` generiert wird, entscheidet der
  Renderer, und ein dritter Name kostete sechs Stellen in zwei Sprachen.
- **`.ultra/{loom,brain}/` als gemeinsames Verzeichnis.** Am 2026-09-10
  erwogen und vertagt: 34 Codestellen in `ultraloom`, `manifestNames` plus
  `pkg/guard/manifest.go` plus `registry.manifest_path` in zwei Sprachen,
  Migration in sieben Repos, und eine Übergangszeit mit zwei Layouts. Es bringt
  Nachbarschaft, nicht geteilte Antworten — die bringt Teil 1, und danach ist
  die Umbenennung jederzeit mechanisch nachholbar.
