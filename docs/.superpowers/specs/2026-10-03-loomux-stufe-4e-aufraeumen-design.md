# Stufe 4e: der Aufräum-PR — Design

Stand 2026-10-03, Zweig `refactor/stage-4e-cleanup` von `origin/master`
b31a2fad (v6.2.0). Entscheidungen mit dem Nutzer vom 2026-10-03; alles
Weitere aus dem Code an HEAD gelesen und, wo vermerkt, geprobt. Vorgänger:
Plan `2026-09-28-loomux-stufe-4e.md`, „Stück C“ (Tasks 6–10), und Plan
`2026-09-29-loomux-stufe-4e-umstellung.md`, „Stück 4“. Messakte:
`parity/stufe-4e.md` (Messungen 3, 4, 10), Auflagen: `parity/stufe-3a.md`
(„Auflagen an spätere Stufen“).

## Problem

Die Welle ist durch (2026-10-01/02). Im Code stehen noch die Lese-Rückfälle
aus der Zeit, in der ultra-brain neben loomux lief:

- `config.LegacyBrainDirUntilStage3` mit `LOOMUX_LEGACY_BRAIN_DIR` und
  `ArtifactLookup.Fallback`: brain liest Artefakte und Stempel zuerst aus
  `%LOCALAPPDATA%\loomux`, sonst aus `%LOCALAPPDATA%\brain`.
- `config.ReadAreaManifestUntilStage4` mit den Altnamen
  `.ultra-brain/config.toml` und `.brain.toml`.
- `Manifest.Lanes` (`[check] lanes`), das kein Produktcode liest.
- Ratschläge in Meldungen, die noch die Python-Befehle `brain reindex` und
  `brain reconcile` nennen.

Der alte Plan (Stück C) wollte vorher noch eine Aside-Auflösung für
read-only-Bereiche bauen (Tasks 6/7), allein für `#Obsidian/AI`. Der Bereich
wird jetzt aus der Registry genommen; damit hat dieser Teil keinen Nutzer.

Dazu kommt ein Befund, der erst beim Lesen für diese Spec auffiel
(Entscheidung 9): an HEAD bricht jeder brain-Leser ab, sobald die Registry
einen Arbeitsbereich ohne `[area]` führt — genau den Fall, den `init
--brain=none` seit v6.2.0 erzeugt.

## Entscheidungen

1. **Reihenfolge: erst die Registry, dann der PR.** Der Mensch entfernt drei
   Einträge (Abschnitt „Registry-Schritt“), bevor der PR gemergt wird. Danach
   ist kein registrierter Bereich mehr über einen Altnamen deklariert und
   keiner `readonly`. Der PR setzt diesen Stand voraus; ohne ihn fällt
   `ultra-brain` (`.brain.toml`) und `ultraloom` (`.ultra-brain/config.toml`)
   als „nicht deklariert“ heraus (Entscheidung 8).
2. **`LegacyBrainDirUntilStage3`, `LOOMUX_LEGACY_BRAIN_DIR`,
   `legacyBrainDirUntilStage3For` und `ArtifactLookup.Fallback` entfallen.**
   `ArtifactLookup` schrumpft auf `Primary`; ob die Hülle bleibt (sie trägt
   `Resolve`/`WritePath` an elf Aufrufern) oder durch einen Pfad ersetzt
   wird, entscheidet der Plan nach Diffgröße — Verhalten ist in beiden Fällen
   „nur der neue Ort“. Die durchgereichten Parameter `fallbackDir`,
   `FallbackDir`, `LegacyDir` entfallen mit (Liste unten).
3. **`ReadAreaManifestUntilStage4` und `manifestNamesUntilStage4` entfallen.**
   Jeder Aufrufer liest `ReadDeclaration(filepath.Join(dir, ".loomux",
   "config.toml"))` über eine kleine Hilfe in `internal/config` (Name im
   Plan, z. B. `ReadAreaDeclaration(dir)`), die zwei Fehler unterscheidet:
   Datei fehlt → `ErrNoManifest`, Datei ohne `[area]` → `ErrNoArea`. Die
   Sonderregel „`.loomux/config.toml` ohne `[area]` ist Policy, frag den
   nächsten Namen“ fällt mit den Altnamen weg. `legacy.go` entfällt ganz.
4. **`Manifest.Lanes`, `LaneConfig` und der Dekodierblock in
   `readManifestAmong` entfallen**, mit `TestReadManifestCheckLanes` und
   `TestReadManifestCheckLanesTable`. Das Drahtfeld `manifestFile.Check` kann
   mitgehen: `DeclarationKeys` (`internal/config/declaration.go:46`) kennt
   `check` nicht, und `area check` klassifiziert `check.lanes` über seine
   eigene Hinweistabelle (`internal/cli/areacheck.go`, `legacyHints`), nicht
   über das Feld. `TestClassifyKeysCallsALanesTableIgnoredWithAHint` bleibt
   grün und bleibt stehen.
5. **`area check` behält sein Wissen über die Altnamen bis 4f.** Es liest die
   drei Namen über seine eigene Tabelle `manifestNames`; nur die Zeile
   `chosen:` ruft heute `ReadAreaManifestUntilStage4`
   (`internal/cli/areacheck.go:44`). Sie bekommt eine eigene Auswahl in
   `areacheck.go` mit der bisherigen Regel (erster regulärer Name; eine
   `.loomux/config.toml` ohne `[area]` gibt nach), damit Ausgabe und
   Aufzeichnung von `area check` byte-gleich bleiben.
6. **Die Auflösung read-only-Bereiche ins Zustandsverzeichnis — was davon
   geht.** Gebaut ist an HEAD **keine** Aside-Auflösung: `ResolvedAreaDir`
   (`internal/config/artifacts.go:145`) kennt nur den Rückfall auf
   `fallbackDir`; Tasks 6/7 des alten Plans wurden nie umgesetzt. Was
   entfällt, ist also (a) der Plan dafür — Tasks 6/7 werden nicht gebaut —
   und (b) der Rückfallteil von `ResolvedAreaDir`; die Funktion fällt mit
   leerem Rückfall auf `ManifestDir` zusammen und wird durch `ManifestDir`
   ersetzt (18 Aufrufer, Liste unten). **Die Regel „read-only → Artefakte
   unter `<state>/areas/<scope>`“ selbst (`ManifestDir`,
   `internal/config/manifest.go:481`) bleibt**, weil sie einen
   Produktnutzer außerhalb der Registry des Nutzers hat:
   `dev bench search` legt seinen Korpus als read-only-Bereich mit
   Deklaration und Register im Zustandsverzeichnis an
   (`internal/dev/benchsearch/corpusrun.go:70,77`); er schreibt seine
   Deklaration schon unter `.loomux/config.toml` (`corpusrun.go:29,78`,
   geprobt), bleibt also ohne Änderung lauffähig. Widerspricht der
   Vorgabe „die Auflösung entfällt“; siehe „Widersprüche“.
7. **`readonly` als Registry-Schlüssel bleibt.** Gelesen: die Schranke baut
   aus ihm `forbiddenRoots` (`internal/brain/guard/guard.go:171-192`) und
   nimmt read-only-Wikis aus `writableRoots` (`guard.go:150`), sie liest die
   Deklaration read-only-Bereiche aus dem Zustandsverzeichnis
   (`guard/registry.go:51-65`), `area add` schreibt ihn
   (`internal/config/registrywrite.go:93`), und `dev bench search` setzt ihn
   (Entscheidung 6). Nicht tot. Nach dem Registry-Schritt trägt ihn kein
   Eintrag der Maschine; der Code bleibt, die Schranke ändert sich nicht.
   `apply.moveStock` (`internal/brain/apply/stock.go`) schrumpft mit leerem
   Rückfall auf `lock.Recover(target)` (`stock.go:62`) — **dieser Aufruf
   bleibt** im Schreibpfad von `approve` (`approve.go:452-460`), sonst
   schriebe `approve` das Register in ein fehlendes Ziel neben einem Aside,
   und ein späteres `Recover` löschte das Aside (Bedenken in
   `parity/stufe-4e.md`, Messung 4). Ein Test hält das fest: Aside, kein
   Ziel, `approve` → das Aside ist danach eingeräumt, nicht gelöscht.
8. **Fehlerverhalten nach der Änderung.** Ein registrierter Bereich, dessen
   Wurzel nur ein Altmanifest trägt, bekommt die Meldung des neuen
   Lesers, ergänzt um einen Hinweis — Vorschlag (Wortlaut im Plan
   festzurren):

   ```
   error: C:/…/ultra-brain: no manifest found (.loomux\config.toml); an old manifest lies there (.brain.toml) — `loomux area check C:/…/ultra-brain` shows what to carry over
   ```

   Der Hinweis entsteht nur, wenn `ErrNoManifest` zutrifft **und** einer der
   beiden Altnamen als reguläre Datei daliegt; die Namensliste dafür stammt
   aus `area check` (eine Quelle, entfällt mit ihr in 4f). An HEAD lautet die
   Meldung `…: no manifest found (.loomux\config.toml, .ultra-brain\config.toml, .brain.toml)`
   (geprobt, siehe Entscheidung 9). Ein Eintrag **ohne** `workspace`, dessen
   `.loomux/config.toml` kein `[area]` hat, bekam bisher `ErrNoManifest`
   (der Leser fiel durch die Altnamen), künftig `ErrNoArea`
   („[area] is missing“) — beide bleiben Fehler.
9. **Arbeitsbereiche ohne `[area]` sind ein Normalfall (eigener `fix`).**
   Geprobt an HEAD gegen die echte Registry mit einem frisch gebauten
   Binär (`go build ./cmd/loomux`, Scratchpad):
   `loomux brain catalog --scope all` und `loomux brain status` enden beide
   mit Exit 1 und `error: C:/Users/micro/Documents/#GIT/iam_backend: no
   manifest found (…)`. Die drei `iam_*` sind `workspace = true`, ihre
   `.loomux/config.toml` hat kein `[area]` (geprobt per `grep -c '^\[area\]'`:
   0). Der Fehler stammt aus v6.2.0 (`c4b78d4f`), liegt also schon auf
   `master`. **Er ist nicht Teil dieses Zweigs, sondern Vorbedingung:
   vorher gemergt: `fix(brain): …` als eigener PR von `master`.** Was
   hier steht, legt fest, was jener PR liefern muss.
   Regel, an einer Stelle definiert (Hilfe in `internal/config` neben dem
   neuen Leser): ein Registry-Eintrag mit `workspace = true`, dessen
   `.loomux/config.toml` fehlt oder kein `[area]` hat, ist **kein
   brain-Bereich**: die Leser überspringen ihn still — weder sichtbar noch
   verborgen, keine Notiz, kein Ratschlag. Ein Eintrag ohne `workspace`
   bleibt ein Fehler (Entscheidung 8). `iam_wiki` (deklariert, Wiki =
   Repo-Wurzel über die Registry, kein `[layout] wiki`) ist ohnehin ein
   gewöhnlicher Bereich und braucht nichts. Betroffene Leser — jeder, der
   über die Registry iteriert und eine Deklaration liest: siehe Tabelle
   „Aufrufer“, Spalte „iteriert Registry“. Die Schranke (`guard`) ist nicht
   betroffen: `checkDeclaration` nimmt eine fehlende Deklaration und eine
   ohne `[area]` schon heute hin (`guard/registry.go:67-69`).
10. **Ratschläge:** `ReconcileAdvice` (`internal/brain/search/stamp.go:23`),
    `internal/brain/status/status.go:61` und die zwölf Meldungen in
    `internal/brain/graph/read.go` (`:23,51,64,97,102,105,108,114,122,127,131,136`)
    nennen `loomux reconcile` bzw. `loomux reindex`. Die Doku nennt sie auch.
    Die Fälle halten den alten Wortlaut im **verglichenen** Teil: 1b-1
    `brain-status/{never-indexed,stamp-missing,stamp-stale,vault-cloud,vault-local}/stdout`
    und 1b-2 `brain-neighbors/{broken-graph,never-indexed}/result`,
    `brain-status/{broken-graph,never-indexed,vault-cloud}/result`
    (geprobt per `grep -rl`; `internal/cases/runner.go:243-303` vergleicht
    stdout und Baum, nicht stderr). Wie sie sich ändern: Abschnitt „Tests“.
11. **Auflagen ohne Träger (`parity/stufe-3a.md`, Nachtrag 2026-09-27):**
    - Ratschläge `brain reindex`/`brain reconcile`: trägt dieser PR
      (Entscheidung 10).
    - Fegen der Asides nach einem erschlagenen Tausch und die
      read-only-Deklarationen unter `.loomux/config.toml`: entfallen mit
      dem Aside-Zweig (Entscheidung 6); es gibt nach dem Registry-Schritt
      keinen read-only-Bereich der Maschine mehr. Der Heiler
      `lock.Recover` in `index.recoverStock` und `apply.moveStock` bleibt.
    - Mitnehmen von `merge-events.done.tsv` und `qmd-collections.json`:
      **ausdrücklich fallengelassen**. Die Welle lief ohne; `reindex`
      baut die Collections neu. Die Abschnitte „Stufe 4: …“ in
      `stufe-3a.md` werden mit dieser Entscheidung abgeschlossen (Eintrag in
      ihrer Abweichungstabelle, wie die Akte es vorschreibt), nicht
      gelöscht.
12. **Nicht in diesem PR:** die übrigen Befunde der Welle
    (`parity/stufe-4e.md`, Messung 10, Befunde 1, 2, 4, 5: Profilart ohne
    Prüfgegenstand fällt mit Exit 1, `verify.profiles` nicht setzbar, Repo-
    Wurzel als Wiki abgelehnt, `dev bench cases` ohne Hook). Sie werden
    Folgezeilen der Roadmap. Befund 3 (`--brain=none` ohne Bereich) ist durch
    v6.2.0 erledigt; seine Folge trägt Entscheidung 9.

## Registry-Schritt (Mensch, vor dem Merge)

Datei: `C:/Users/micro/AppData/Local/loomux/registry.toml` (gelesen
2026-10-03, nur lesend). **Von Hand, es gibt keinen Befehl:** `loomux area`
kennt nur `add` und `check` (`internal/cli/area.go:56,59`, geprobt).
In einem Durchgang:

Ganze `[[area]]`-Blöcke entfernen:

| `scope` | Grund |
|---|---|
| `project/obsidian-ai` (`readonly = true`) | `#Obsidian/AI` ist abgekündigt und wird gelöscht |
| `project/ultra-brain` | wird nach der Migration gelöscht; deklariert über `.brain.toml` |
| `project/ultraloom` | ebenso; deklariert über `.ultra-brain/config.toml` |

In den Blöcken `project/iam-backend`, `project/iam-frontend` und
`project/iam-workers` die Zeile `wiki = "C:/Users/micro/Documents/#GIT/iam_wiki"`
entfernen. Gewollte Form ist die, die `init --brain=none` schreibt: `scope`,
`path`, `workspace = true`, **kein** `wiki` (`internal/setup/plan.go:178`,
„workspace without wiki“). Das Wiki gehört `project/iam-wiki`; drei weitere
Einträge auf dieselbe Wurzel öffnen sie der Schranke über fremde Bereiche.

Danach bleibt
`%LOCALAPPDATA%\loomux\areas\project-obsidian-ai\` (samt `.lock`) und die
`.lock`-Dateien von `project-ultra-brain`/`project-ultraloom` als Waisen
liegen; der Mensch darf sie löschen, nichts liest sie mehr. Ebenso die
Reste `project-iam-wiki.alt` und `project-space.alt` aus der Umstellung.
Probe danach (nur lesend): `loomux brain catalog --scope all` und
`loomux brain status` — nach Entscheidung 9 Exit 0.

## Was geht, was bleibt

### Aufrufer (an HEAD, `git grep`, ohne Tests)

`ReadAreaManifestUntilStage4` — 12 Produktaufrufer:

| Stelle | iteriert Registry |
|---|---|
| `internal/brain/apply/resolve.go:90` | ja |
| `internal/brain/check/house/federation.go:372` | ja |
| `internal/brain/check/run/run.go:349`, `:465`, `:503` | ja |
| `internal/brain/convert/run.go:38` | ja |
| `internal/brain/index/reindex.go:167` | ja |
| `internal/brain/maintenance/reconcile.go:270` | ja |
| `internal/brain/privacy/channel.go:53` (über `privacy/areas.go:50`, `VisibleAreas`) | ja |
| `internal/brain/wiki/census.go:136` | Plan prüft |
| `internal/cli/areacheck.go:44` | nein (Entscheidung 5) |
| `internal/cli/lintsweep.go:93` | ja |
| `internal/setup/facts.go:213` | nein (ein Projekt) |

`LegacyBrainDirUntilStage3` — 4: `internal/cli/brain.go:62`,
`internal/cli/dev.go:736` (`FallbackDir`), `internal/cli/serve.go:110`
(`LegacyDir`), `internal/config/artifacts.go:24`. Mit ihm fallen
`serve.Options.LegacyDir` (`internal/serve/serve.go:49,136,258,268`),
`serve.NewUpkeep(…, legacyDir)` (`upkeep.go:52`) und `benchsearch`
`FallbackDir` (`internal/dev/benchsearch/run.go:36,74`).

`ArtifactLookup.Fallback`/`fallbackDir`-Parameter: `apply/resolve.go:268`,
`apply/stock.go:65`, `check/house/federation.go:373`, `check/run/run.go:350,504`,
`index/reindex.go:132`, `maintenance/events.go:70,152`,
`maintenance/reconcile.go:137,271,462`, `search/stamp.go:34`,
`cli/convert.go:56,120`, `cli/index.go:78`, `cli/lintsweep.go:93`,
`cli/wikicmd.go:181`, `serve/upkeep.go:52,120,162,212`.

`ResolvedAreaDir` → `ManifestDir`: `apply/resolve.go:268`, `apply/stock.go:65`,
`catalog/area.go:15`, `check/house/federation.go:373`, `check/run/run.go:350,504`,
`convert/run.go:38`, `graph/read.go:33`, `index/reindex.go:166`,
`maintenance/reconcile.go:271`, `maintenance/scan.go:77`,
`privacy/areas.go:50`, `search/search.go:131`, `status/status.go:60,123`,
`cli/lintsweep.go:93`, `cli/wikicmd.go:181`. Kommentare, die den Rückfall
oder `loomux migrate` als Ende nennen: `index/reindex.go:97`,
`index/staging.go:27,49`, `maintenance/reconcile.go:110,264`,
`maintenance/scan.go:69`, `maintenance/events.go:128`,
`privacy/areas.go:33`, `privacy/channel.go:45-51`, `apply/stock.go:31`,
`apply/resolve.go:82`, `check/run/run.go:328,378`, `lock/replacedir.go:35,46`,
`cli/maintenance.go:26`, `config/registry.go:36-38` (dort zudem veraltet:
`project/space` trägt nur `workspace`, `project/iam-wiki` keinen der beiden).

`Manifest.Lanes`: kein Produktleser (geprobt: `git grep -n "\.Lanes\b"` —
nur `manifest.go:217,223` als Draht `file.Check.Lanes` und `internal/verify/*`
mit anderem Typ; `git grep LaneConfig` nur in `manifest.go`). Gleicher Befund
wie Messung 4 in `parity/stufe-4e.md`.

### Bleibt

- `ManifestDir`, `Area.ReadOnly`, der Registry-Schlüssel `readonly`, die
  Schranke unverändert (Entscheidung 6, 7).
- `lock.Recover` in `index/staging.go:38` und `apply/stock.go:62`.
- `area check` mit seinen drei Namen und `legacyHints` (Entscheidung 5).
- `internal/dev/importcases` mit `TranslateWorld` (Werkzeug für
  Aufzeichnungen, nicht Leser).

## Tests

TDD je Commit, roter Lauf mit `--- FAIL` im Bericht.

- **Im vorher gemergten `fix(brain)`-PR (Entscheidung 9), nicht hier:** eine Testwelt mit einem
  gewöhnlichen Bereich und einem `workspace = true` ohne `[area]`; je Leser
  aus der Tabelle ein Test, rot an HEAD (Fehler `no manifest found`),
  grün danach. Gegenfall: derselbe Eintrag ohne `workspace` bleibt Fehler.
  Ein CLI-Test für `brain catalog --scope all` nimmt die Probe auf.
- **Fehlermeldung (Entscheidung 8):** Welt mit einem Bereich, der nur
  `.brain.toml` hat → Meldung samt Hinweis; nur `.ultra-brain/config.toml`
  → ebenso; keine Datei → ohne Hinweis.
- **`ReadAreaManifestUntilStage4`-Tests** (`apply/resolve_test.go:120-160`,
  `apply/stock_test.go`, `maintenance/world_test.go`,
  `privacy/privacy_test.go`, `config/legacy_test.go`,
  `check/run/run_test.go:322,369,384`, `answer/nesting_test.go:38,47`):
  umgestellt auf `.loomux/config.toml` oder entfernt; kein Test außer
  denen von `area check` behält einen Altnamen.
- **`LOOMUX_LEGACY_BRAIN_DIR` in Tests:** 22 Stellen (`config/legacy_test.go`,
  `config/artifacts_test.go:86`, `brain/check/run/run_test.go:26`, `cli/*_test.go`
  inkl. `cases_{1b1,3a,3b,3c,4a2,4c1,4d}_test.go`). Die Fallsuiten setzen ihn
  auf dieselbe Welt wie `LOOMUX_STATE_DIR` — das `Setenv` entfällt ohne
  Wirkung. `config_mutants_test.go:19`, `config_proposals_mutants_test.go:18`
  nehmen den Namen aus ihrer Liste.
- **Aufgezeichnete Fälle — nicht von Hand ändern.** Altnamen in Welten, die
  ohne Übersetzung laufen (gezählt mit `find`, `.brain.toml` und
  `.ultra-brain/config.toml`): `1b-1-worlds` 28, `1b-2-worlds` 25,
  `3a-worlds` 21, `3b-worlds` 18, `3c-worlds` 17, `4a2-worlds` 7,
  `4c1-worlds` 7, `4d-worlds` 30, dazu je einer in `3a/` und `4d/`; die
  `*-source`-Bäume (589) sind Belege. Nur 1a übersetzt beim Import
  (`importcases/mcp.go:49` → `TranslateWorld`). Ohne Maßnahme fällt nach
  Entscheidung 3 fast jede Fallsuite. Weg: **der Ablauf der Suiten übersetzt
  beim Bereitstellen** — `internal/cases` ruft auf der Kopie von `world`,
  `world_after` **und** der Vergleichsbasis (`runner.go:249-273`: ein Fall
  ohne `world_after` wird gegen seine Ausgangswelt verglichen) dieselbe
  Übersetzung (`importcases.TranslateWorld` oder eine nach `internal/cases`
  gezogene Form davon). Die Dateien unter `testdata/` bleiben unberührt.
  **Das ist eine Brücke mit festem Ende:** 4f stellt die
  Wiederholungstests auf loomux' eigene Erwartungen um, und der Übersetzer
  beim Bereitstellen geht mit ihnen. Wie viele Fälle zu freigegebenen
  Abweichungen werden, ist **unbekannt, bis der Plan sie zählt**.
  Zu prüfen, bevor der Plan festliegt (nicht geprobt): (a) Fälle mit
  absichtlich kaputtem Altmanifest (`missing-manifest`,
  `manifest-without-scope`, `manifest-wrong-type`, 4d `broken-manifest`)
  — `TranslateWorld` verweigert ungültiges TOML (README „An import reads
  every world's manifests“); sie kommen als freigegebene Abweichung in die
  Liste ihrer Suite (`approved…`-Muster wie 2a/2c) mit Zeile in der
  Paritätsakte; (b) Fälle, deren stdout einen Manifestpfad nennt —
  Anzahl messen und je Suite auflisten.
- **Ratschläge (Entscheidung 10):** die zehn Fälle oben ändern sich nicht von
  Hand. Der Plan wählt zwischen einer Umschreibregel des Imports (wie
  `rewritePaths`, `importcases.go:259`, reproduzierbar bei Re-Import) und
  einer freigegebenen Abweichungsliste je Suite; beide mit Zeile in
  `parity/stufe-1b-1.md`/`stufe-1b-2.md` bzw. der Abweichungstabelle von
  `stufe-3a.md`.
- **`moveStock`/`approve`:** der Test aus Entscheidung 7 (Aside, kein Ziel)
  muss rot werden, wenn `recoverDir` im Overlay entfällt.
- **Mutationsrunde** über jede geänderte Funktion: neue Hilfe(n) in
  `internal/config`, die Überspringregel je Leser, die Hinweiserzeugung,
  `areacheck.go` (neue `chosen`-Auswahl), `moveStock`. `internal/cli`
  braucht > 60 s — Handrunde per `go test -overlay` gegen gezielte Tests,
  Build-Fehler als BADMUTANT. Überlebende in `parity/stufe-4e.md`, Abschnitt
  „Überlebende Mutanten“ (`internal/plancheck` verlangt ihn für ✅).

## Messen

Der alte Task 10 maß die Aside-Auflösung, die entfällt. Stattdessen:
`loomux brain search` und `privacy.VisibleAreas` warm, je fünf Läufe,
vor (HEAD, nach dem Registry-Schritt) und nach dem PR — erwartet ist ein
`stat` weniger je Artefakt ohne Primärdatei. Eintrag in `docs/en|de/benchmarks.md`
(Datum, Uhrzeit, Basis gegen Änderung, kalt und warm). Vor dem Registry-Schritt
ist „vorher“ nicht messbar (Exit 1, Entscheidung 9); dann wird die Basis nach
dem `fix`-Commit genommen und das im Eintrag gesagt.

## Doku

Sätze, die den Rückfall beschreiben (alle Sprachen, geprobt per `grep` über
alle Markdown außer Arbeitspapieren und `testdata`):

- `docs/en/cli-reference.md:32,682,725,735` und `docs/de/cli-reference.md:32,705,748,758`
  (Umgebungsvariable, Pflege-Umgebung; `area check`-Abschnitt bleibt, nur
  die „fallback“-Umgebungszeile ändert sich).
- `docs/en/configuration.md:941,943,952-956` und `docs/de/configuration.md:971,973,982-987`.
- `docs/en/getting-started.md:203`, `docs/de/getting-started.md:205`
  (Altmanifeste werden angenommen).
- `docs/wiki/topics/datenmodell-und-bereiche.md:48-58` (Wiki: nachziehen
  im selben Task, nicht löschen).
- `docs/en|de/migration.md` Zeile 4e und die Fähigkeitszeilen; Roadmap in
  `README.md`/`README.de.md` bekommt die Folgezeilen aus Entscheidung 12.
- `docs/en|de/benchmarks.md`: historische Einträge bleiben wörtlich
  (chronologisches Protokoll), nur der neue Eintrag kommt dazu.
- Fusions-Spec: Nachtrag zur Stufe 4e (#28 o. ä.) mit Entscheidungen 6, 9,
  11 und 12, bevor `migration.md` folgt; die Zeilen 748 (#18, „`Manifest.Lanes`
  bleibt geparst, bis `loomux migrate`“) und 769 (Klasse a) bekommen den
  Vermerk „erledigt“.
- `parity/stufe-3a.md`: die drei Abschnitte „Stufe 4: …“ und „Umstieg: …“
  werden mit Verweis auf diese Spec abgeschlossen.
- `testdata/cases/README.md:29,51,310` nennt `LOOMUX_LEGACY_BRAIN_DIR` und
  die Altnamen in den Welten; nachziehen (Beschreibung des Ablaufs, keine
  Aufzeichnung).

## Fertig-Bedingungen von 4e

Fusions-Spec, „Eine Stufe ist fertig, wenn“, und die Abhängigkeiten aus
`2026-09-28-loomux-stufe-4e-design.md` (Zeilen 29–35):

| Bedingung | Stand 2026-10-03 |
|---|---|
| 1. übersetzte Fälle grün oder freigegeben | nach diesem PR (Abschnitt „Tests“) |
| 2. Coverage 100 % | nach diesem PR |
| 3. Mutationsrunde gelaufen, Überlebende dokumentiert | Stück A und neue Pakete: gelaufen (2026-09-29, 2026-10-02); dieser PR: offen |
| 4. Zielwerte gemessen und eingetragen | Welle: eingetragen; dieser PR: offen (Abschnitt „Messen“) |
| 5. Selbstnutzung | loomux nutzt die Stufe seit der Umstellung; nach dem Merge Probe `brain catalog`/`status` an der echten Registry |
| Abhängigkeit 4a-2 | ✅ 2026-09-28 (`migration.md:38`) |
| Abhängigkeit 4c-1 | ✅ 2026-09-29 (`migration.md:39`) |
| Abhängigkeit Remote `brain-knowledge` | erledigt (`origin` → `github.com/xidus90/brain-knowledge`, geprobt) |
| Abhängigkeit sechs Entscheidungen `parity/artefakte-nach-lebensdauer.md` | laut `parity/stufe-4e.md` Messung 5 offen; #1, #3, #8 berühren Zustandsdateien. Der Plan liest den Stand vor Task 1; sind sie offen, braucht 4e ✅ einen ausdrücklichen Entscheid des Nutzers, dass sie nicht an 4e hängen |
| Registry-Schritt | Mensch, vor dem Merge |

4e wird ✅ in `docs/en|de/migration.md` (mit `internal/plancheck`), wenn
dieser PR gemergt ist und die Zeilen oben erfüllt sind; 4f wird danach neu
gelesen.

## Nicht Teil davon

- Die Wellenbefunde 1, 2, 4, 5 (Entscheidung 12).
- Entfernen von `area check`, `TranslateWorld`, der Erkennung alter
  Host-Einträge (`internal/hooks/status.go`, `worktree.go`,
  `cases/gitworld.go`): 4f.
- Entfernen von `readonly` und `ManifestDir`-Regel (Entscheidung 7).
- Löschen der Waisen im Zustandsverzeichnis und der Repos `ultra-brain`,
  `ultraloom`, `#Obsidian/AI` (Mensch).
- Die sechs Entscheidungen aus `artefakte-nach-lebensdauer.md`.

## Widersprüche zwischen Vorgabe und Code

1. **Aside-Zweig:** gebaut ist keiner; „entfernen“ heißt hier „nicht bauen“
   (Entscheidung 6).
2. **„Auflösung read-only ins Zustandsverzeichnis entfällt“:** die Regel hat
   mit `dev bench search` einen Produktnutzer; sie bleibt, nur ihr Rückfall
   geht (Entscheidung 6, 7). Alternative, falls gewünscht: den Bench-Korpus
   auf einen schreibbaren Bereich umbauen — größerer Diff, eigener PR.
3. **„Arbeitsbereiche ohne `[area]` sind Normalfall“:** an HEAD sind sie ein
   harter Fehler aller brain-Leser (geprobt); ein eigener `fix(brain)`-PR
   von `master` behebt ihn vorher (Entscheidung 9, Vorbedingung).
4. **Ratschlagsfälle:** sie liegen im verglichenen stdout/`result`, nicht nur
   in Belegen; ihr Wechsel braucht eine Import-Regel oder Abweichungsliste.
5. **Fallsuiten:** acht Suiten laufen auf Welten mit Altnamen ohne
   Übersetzung; ohne Übersetzung beim Bereitstellen fallen sie.
