# Paritätsakte Stufe 4c-2

**Quelle:** ultra-brain `loomux-3-source`, derselbe Tag wie in Stufe 3a, 3b
und 4c-1. Gelesen: `src/brain/bench/{questions,corpus,quality,latency,report}.py`
und `src/brain/cli.py:1813-2368` (`_bench` und alles, was es ruft).
**Spec:** `docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md`,
Abschnitte „4c im Einzelnen“, „Abweichungen beim Planen von 4c“ (die Teile zu
4c-2) und „Abweichungen beim Planen von 4c-2“
**Plan:** `docs/.superpowers/plans/2026-09-26-loomux-stufe-4c-2.md`

`loomux dev bench search` misst den Rang der erwarteten Quelle je Frage
(Treffer bei Rang ≤ 3) und, mit `--latency`, die Latenz der Suchkette. Die
Referenz ist `brain bench` (`_bench` in `cli.py`). Der Alltagsmodus fragt
über den Dienst (`QmdMcpPort`, `environment.port = "daemon"`), der
Korpusmodus über die qmd-CLI mit eigenem Indexnamen (`QmdPort`,
`environment.port = "cli"`).

## Stand

| Teil | Stand |
|---|---|
| Bau (`dev bench hooks\|repos\|search`, Hülle, Korpus `v1`) | gebaut 2026-09-26 |
| Paritätslauf gegen das echte qmd | ✅ 2026-09-27: 50/50 Ränge gleich, 37/37 Befunde |
| Selbstnutzung | erledigt 2026-09-27 (Korpusqualität, Alltagslatenz, `hooks` und `repos` mit `--out`), bis auf eine saubere Alltagsqualität (qmd-Backbone), siehe unten |

Beide Teile fassen den geteilten qmd-Index des Nutzers an (die Referenz
schreibt ihre Sammlung in die echte `index.yml`) und liefen deshalb von Hand,
mit Freigabe des Nutzers, in einem ruhigen Fenster. Das Verfahren steht unten
so, dass ein Mensch es ohne den Plan wiederholen kann.

## Abweichungen

| Abweichung | Art | Beleg | Begründung |
|---|---|---|---|
| Bericht, Hinweise und Fehlermeldungen englisch | freigegeben 2026-09-26 | Plan E2 | Jede Ausgabe von loomux ist englisch. Die Sortennamen `exakt`, `umschreibung`, `gemischt`, `sprachuebergreifend` und die Feldnamen des Fragensatzes bleiben, weil sie das Dateiformat sind |
| Korpusmodus in einem eigenen qmd-Index `--index loomux-bench-<zufall>` statt in der echten `index.yml` | freigegeben 2026-09-26 | Spec, Tabelle „Mit dem Nutzer entschieden“, Zeile „Isolation im Korpusmodus“ | Die Referenz isoliert gar nicht und trägt die Sammlung in die geteilte `index.yml` ein. loomux legt `<name>.yml` und `<name>.sqlite` an, teilt nur die Modelle, übernimmt den Block `models:` aus `index.yml` und räumt immer auf; eine Sperre je Name unter `<zustand>/bench/` schützt parallele Läufe |
| `--corpus` mit `--latency` wird verweigert (`--latency measures the everyday chain; drop it with --corpus`, Exit 1) | freigegeben 2026-09-26 | Plan E3 | Im Wegwerf-Zustand gibt es kein `index.md`, die Operation `catalog` scheiterte; die Latenz über die CLI (5–12 s je Aufruf) ist laut Spec nicht zu berichten |
| Die gemeinsame Hülle statt des JSON der Referenz | freigegeben 2026-09-26 | Plan E4, Spec „Was daraus für den Bau folgt“ (`benchreport`) | Alle drei Befehle schreiben dieselbe Hülle: `schema`, `command`, `stamp`, `environment`, `timings[]` in Millisekunden, `payload`. Die Fragen stehen darum unter `payload.questions[]`, nicht auf oberster Ebene wie bei der Referenz (`questions[]`); Umgebung und Latenz ebenso verschoben |

**Keine Abweichungen, zur Klarstellung:**

- **Reihenfolge der Befunde im Fragensatz.** Sie folgt der Referenz: je
  Eintrag in Dateifolge (fehlende Felder, doppelte ID, Ziel und Beleg, dann
  Sorte), zuletzt die Form je Sorte in der Folge der Sorten (Plan, Task
  „Fragensatz“).
- **`earlier` der Korpusprüfung.** Die Referenz kennt einen Parameter für
  frühere Stände; ihre CLI reicht ihn nie weiter. Er entfällt, es gibt also
  keine früheren Stände zu prüfen (Plan, Task „Korpusprüfung“).
- **Dateinamen.** `bench-<stempel>-<profil>.md` und `.json` wie bei der
  Referenz (Plan E1), weil der echte Fragensatz in `98 Messung` schon Dateien
  dieses Namens trägt. Stempel in UTC, `JJJJ-MM-TT-HHMM`.

## Paritätslauf

✅ Gefahren am 2026-09-27, 01:07–01:08 (Ergebnis unten).

Kriterium: gleiche Ränge je Frage mit `--profile keyword`. Die Referenz
sucht im geteilten Index mit Sammlungsfilter, loomux im eigenen Index; die
Wortgewichte von `keyword` hängen am ganzen Index (gemessen 2026-09-26:
dieselbe Notiz 0,76 im Index mit 10 Notizen, 0,88 im Index mit 100, Ränge in
vier Stichproben gleich). Ein abweichender Rang gilt erst als Fehler von
loomux, wenn die Scores beider Läufe ihn nicht erklären.

### Verfahren

Alle Befehle in Git Bash. `<scratch>` ist ein frisches Verzeichnis außerhalb
jedes Repos, `<lx>` die Arbeitskopie von loomux auf diesem Zweig, `<ub>` eine
eigene Arbeitskopie von ultra-brain.

1. **Ruhiges Fenster.** Keine andere Sitzung ruft `brain_*`; sonst startet
   der Dienst neu und hält den Index wieder offen.
2. **Dienste stoppen.** `loomux serve stop`. Dann den qmd-Dienst selbst über
   seine PID beenden, gezielt diese eine, nie per Namensfilter:

   ```sh
   cat ~/.cache/qmd/mcp.pid
   taskkill //PID <pid>
   ```

   Danach prüfen, dass kein `qmd … mcp` mehr läuft (in der
   Prozessliste nach der Befehlszeile sehen, nicht nur nach dem Namen).
3. **Sichern.** `index.yml`, `index.sqlite` und, falls vorhanden,
   `index.sqlite-wal` und `index.sqlite-shm`, dazu die sha256 jeder Datei:

   ```sh
   mkdir -p <scratch>/backup
   cp -p ~/.config/qmd/index.yml ~/.cache/qmd/index.sqlite* <scratch>/backup/
   (cd <scratch>/backup && sha256sum * > SHA256SUMS)
   ls ~/.cache/qmd/index.sqlite* > <scratch>/backup/present.txt
   ```

4. **Referenz.** In ultra-brain am Tag `loomux-3-source`, in einer eigenen
   Arbeitskopie:

   ```sh
   git -C <ultra-brain> worktree add <ub> loomux-3-source
   mkdir -p <scratch>/ref
   cd <ub> && uv run brain-mcp bench --corpus bench/corpus/v1 --profile keyword --out <scratch>/ref > <scratch>/ref.log 2>&1
   ```

   Der Einstiegspunkt ist `brain-mcp`, nicht `brain`: auf dem `PATH` liegt
   ein Go-Binär namens `brain`, das den Python-Einstiegspunkt verdeckt. Statt
   eines Worktrees geht auch ein Auszug des Tags in den Scratch
   (`git archive`), dann bleibt das Repo von ultra-brain ganz unberührt. Mit
   `update` und `embed` über alle 14 echten Sammlungen ist ein langer Lauf zu
   erwarten; im Hintergrund starten, Ausgabe in die Datei.
5. **loomux.** Ein Binär aus diesem Zweig, nicht das auf dem `PATH` oder unter
   `%LOCALAPPDATA%` (es kennt die Gruppe `dev bench` womöglich nicht);
   gestartet in der Arbeitskopie, weil `--corpus v1` sie braucht:

   ```sh
   cd <lx> && go build -o <scratch>/loomux.exe ./cmd/loomux
   mkdir -p <scratch>/lx
   cd <lx> && <scratch>/loomux.exe dev bench search --corpus v1 --profile keyword --out <scratch>/lx > <scratch>/lx.log 2>&1
   ```

   `--out` muss ein vorhandenes Verzeichnis sein; eine Datei gleichen Namens
   darin bricht den Lauf vor der ersten Anfrage ab.
6. **Ränge vergleichen.** Die Referenz trägt sie unter `questions[].rank`,
   loomux unter `payload.questions[].rank` (Hülle). Je `id` beide Ränge
   nebeneinander; `null` heißt nicht gefunden.
7. **Abweichung begründen.** Für jede Frage mit anderem Rang die Scores beider
   Seiten holen. Beide Läufe räumen ihre Sammlung am Ende weg (die Referenz
   `brain-bench-corpus` aus dem geteilten Index, loomux seinen ganzen
   benannten Index), darum beide Seiten von Hand nachbauen — **vor**
   Schritt 8, solange die Sicherung noch zurückgelegt wird. `keyword` braucht
   kein `embed`. Der Indexname `probe-4c2` trägt bewusst nicht das Präfix
   `loomux-bench-`, sonst räumte ihn ein paralleler Korpuslauf weg:

   ```sh
   # Seite der Referenz: die Sammlung im geteilten Index, gesucht mit -c
   qmd collection add "<ub>/bench/corpus/v1/notes" --name brain-bench-corpus --mask "**/*.md"
   qmd update
   qmd search "<anfrage>" -c brain-bench-corpus -n 10 --json > <scratch>/score-ref-<id>.json

   # Seite von loomux: derselbe Stand in einem eigenen benannten Index
   qmd --index probe-4c2 collection add "<lx>/testdata/bench/search/v1/notes" --name loomux-bench-corpus --mask "**/*.md"
   qmd --index probe-4c2 update
   qmd --index probe-4c2 search "<anfrage>" -n 10 --json > <scratch>/score-lx-<id>.json

   # aufräumen: der benannte Index ganz; der geteilte wird in Schritt 8 zurückgelegt
   rm -f ~/.config/qmd/probe-4c2.yml* ~/.cache/qmd/probe-4c2.sqlite*
   ```

   `qmd update` im geteilten Index frischt alle 14 Sammlungen auf und dauert
   entsprechend; es ändert nur, was Schritt 8 ohnehin zurücklegt. Liegen die
   beiden Kandidaten im Score so dicht, dass die Gewichtung des Index den
   Tausch erklärt, ist es keine Abweichung von loomux.
8. **Zurücklegen.** Prüfen, dass kein qmd-Dienst läuft (wie in Schritt 2).
   Denselben Satz Dateien zurücklegen; eine `-wal` oder `-shm`, die es vorher
   nicht gab (`present.txt`), löschen. Dann jede Datei gegen die Sicherung:

   ```sh
   cp -p <scratch>/backup/index.yml ~/.config/qmd/index.yml
   cp -p <scratch>/backup/index.sqlite* ~/.cache/qmd/
   sha256sum ~/.config/qmd/index.yml ~/.cache/qmd/index.sqlite*
   ```

   Jede Summe muss der Zeile gleichen Dateinamens in `SHA256SUMS` gleichen,
   und es liegt keine Datei mehr oder weniger da als in `present.txt`.
9. **Keine Reste.** Unter `~/.config/qmd/` und `~/.cache/qmd/` liegt keine
   Datei `loomux-bench-*` oder `probe-4c2*` mehr, und `~/.cache/qmd/models` ist unverändert.
10. **Dienst wieder an.** Der nächste Aufruf eines `brain_*`-Werkzeugs
    startet `serve` und den qmd-Dienst neu.

### Ergebnis

Gefahren am 2026-09-27, 01:07–01:08 Ortszeit (Stempel der Berichte
`2026-09-26-2307` und `-2308`, UTC), qmd 2.8.3 (`facd35e`), Profil
`keyword`, Korpus `v1`. Die Referenz lief als `brain-mcp bench` aus einem
Auszug des Tags `loomux-3-source` im Scratch; loomux als Binär dieses
Zweigs mit `QMD_LLAMA_GPU=vulkan` (für `keyword` ohne Wirkung auf die Ränge,
siehe Selbstnutzung).

| Sorte | Referenz | loomux |
|---|---|---|
| `exakt` | 12/13 | 12/13 |
| `umschreibung` | 0/13 | 0/13 |
| `gemischt` | 1/10 | 1/10 |
| `sprachuebergreifend` | 0/14 | 0/14 |
| gesamt | 13/50 | 13/50 |

- **Ränge je Frage:** 50/50 gleich, keine Abweichung; verglichen je `id`
  zwischen `questions[].rank` der Referenz und `payload.questions[].rank`
  von loomux. Schritt 7 war damit nicht nötig.
- **Befunde:** 37 gegen 37.
- **Median der Antwortzeit** (nicht Kriterium): Referenz 190 ms Treffer,
  373 ms Fehlschläge; loomux 168 ms und 332 ms.
- **Eingriff der Referenz:** Sie hat `index.yml` nur umformatiert
  (Einrückung), inhaltlich nichts geändert.
- **Sicherung und Rückweg:** gesichert wurden `index.yml` (samt `.bak` und
  `.brain-backup`), `index.sqlite`, `-wal` und `-shm`, je mit sha256.
  Zurückgelegt wurden `index.yml`, `index.sqlite`, `-wal` und `-shm`; jede
  sha256 gleich der Sicherung. Danach lief kein qmd-Prozess.

## Selbstnutzung

Gefahren am 2026-09-27 (Ergebnis unten). Geplant war, mit dem Binär aus
Schritt 5 des Paritätslaufs, in der Arbeitskopie von loomux:

1. **Qualität.** `dev bench search --corpus v1 --profile fast --out <scratch>`
   gegen die Baseline 43/50 (13/13 `exakt`, 11/13 `umschreibung`, 8/10
   `gemischt`, 11/14 `sprachuebergreifend`; qmd 2.8.3,
   `testdata/bench/search/v1/baseline/`). Der Korpusmodus legt einen eigenen
   Index an und fasst `index.yml` nicht an; die Latenz dieses Laufs (CLI,
   5–12 s je Aufruf) wird nicht berichtet.
2. **Alltagslatenz.** Ein Lauf mit `--latency` über den Dienst, auf dem
   echten Fragensatz des Bereichs `knowledge` (Vorgabe `--out` ist
   `<bereich>/98 Messung`).
3. **`dev bench hooks`** über einen vorhandenen Fallsatz unter
   `testdata/bench/` mit `--out <scratch>`.
4. **`dev bench repos --dir . --out <scratch>`.**

Qualität und Alltagslatenz gehen in `docs/en/benchmarks.md` und
`docs/de/benchmarks.md` nach dem Muster der Chronik (Datum und Uhrzeit, was
gemessen, kalt und warm).

### Ergebnis

**1. Korpusqualität, `--corpus v1 --profile fast`.**

- **Erster Versuch, Vorgabe-Backbone (CUDA):** Exit 1 nach 42 min. `qmd
  embed` lief rund 40 min und ließ 94 % der Dokumente ohne Einbettung
  zurück, ohne Fehler zu melden; das folgende `vsearch` stürzte mit einem
  CUDA-Fehler in llama.cpp ab (`0xC0000409`). Das Aufräumen war sauber,
  `index.yml` unverändert. Zwei Befunde daraus: `QmdPort` gibt qmd kein
  Backbone mit, während die CLI der Referenz `QMD_LLAMA_GPU=vulkan` festsetzte;
  und ein erfolgreiches `embed` heißt nicht, dass alles eingebettet ist (die
  Referenz hat dieselbe Lücke).
- **Zweiter Versuch, `QMD_LLAMA_GPU=vulkan`**, 2026-09-27 00:47–00:57
  (Stempel `2026-09-26-2247`, UTC), qmd 2.8.3, 100 Dokumente: **40/50**
  gegen die Baseline 43/50.

| Sorte | loomux 2026-09-27 | Baseline 2026-08-21 |
|---|---|---|
| `exakt` | 12/13 | 13/13 |
| `umschreibung` | 11/13 | 11/13 |
| `gemischt` | 7/10 | 8/10 |
| `sprachuebergreifend` | 10/14 | 11/14 |
| gesamt | 40/50 | 43/50 |

  Fehlschläge wie in der Baseline: `c16`, `c21`, `c27`, `c40`, `c44`,
  `c49`. Neu: `c07`, `c32`, `c35` (nicht gefunden) und `c50` (Rang 4).
  `c33` trifft jetzt, in der Baseline nicht. Median je Frage rund 10,8 s
  (Treffer 10 804 ms, Fehlschläge 11 132 ms), weil die CLI die Modelle bei
  jedem Aufruf lädt; diese Latenz wird nicht berichtet. Aufräumen sauber,
  `index.yml` unverändert. Den Abstand von drei Treffern zur Baseline hat
  dieser Lauf nicht aufgeklärt.

**2. Ein Fehler, den die Selbstnutzung fand.** Der erste Alltagslauf brach
vor der ersten Frage ab: ein absolutes `expect` wurde an das Verzeichnis des
Fragensatzes gehängt (Go `filepath.Join` gegen pathlibs `/`, das bei einem
absoluten rechten Teil diesen nimmt). Behoben mit der Korrektur absoluter
`expect`-Pfade im Lader des Fragensatzes: ein absolutes `expect` gilt,
wie es steht; nur ein relatives wird angehängt.
Zurückgestellt: ein unter Windows gewurzelter Pfad ohne Laufwerk oder ein
laufwerksrelativer Pfad wird weiterhin angehängt.

**3. Alltagslauf über den Dienst.** Der echte Fragensatz
(`brain-knowledge/98 Messung/questions.yaml`) ist veraltet: 27 Ziele unter
`space/wiki` liegen seit dem Commit `8eb67a21` in `space` unter
`space/docs/wiki`. Die Datei des Nutzers blieb unberührt; gemessen wurde mit
einer Kopie im Scratch, in der die 27 Pfade umgeschrieben sind, mit
`--scope all --latency`, 511 indizierte Dokumente.

- **Vorgabe-Backbone (CUDA):** Der qmd-Dienst stürzt bei der ersten Anfrage
  mit einem CUDA-Fehler ab, obwohl die GPU frei ist (435 MiB belegt). Das ist
  ein Fehler von node-llama-cpp mit CUDA auf diesem Rechner, nicht von
  loomux.
- **`QMD_LLAMA_GPU=vulkan`, `fast`:** 0/50. Der Index ist unter CUDA
  eingebettet, die Anfragen werden unter Vulkan eingebettet; die Vektoren
  passen nicht zusammen, und jede Anfrage liefert dieselbe `audit.md`.
- **`QMD_LLAMA_GPU=vulkan`, `keyword`:** 8/50 (`exakt` 6/13, `umschreibung`
  1/13, `gemischt` 1/10, `sprachuebergreifend` 0/14). Das belegt, dass loomux
  Treffer richtig auf Bereich und Pfad abbildet; eine Qualitätszahl ist es
  nicht.
- **Vergleich:** Die Referenz kam im Alltag mit `fast` am 2026-08-22 auf
  26/50.
- **Latenz über den Dienst** (Vulkan, Stempel `2026-09-26-2304`, UTC;
  1 kalter und 10 warme Läufe, nach dem Qualitätsdurchgang; gelesenes
  Dokument `engineering/craft/_schema.md`, Anfrage `latenz`):

| Operation | kalt | warm Median | Min | Max |
|---|---:|---:|---:|---:|
| `catalog` | 1 ms | 1 ms | 1 ms | 2 ms |
| `read` | 1 ms | 1 ms | 1 ms | 2 ms |
| `keyword` | 14 ms | 13 ms | 12 ms | 15 ms |
| `fast` | 186 ms | 185 ms | 179 ms | 210 ms |
| `full` | 9293 ms | 707 ms | 653 ms | 731 ms |

  Den Vulkan-Dienst, den dieser Lauf startete, hat der Controller danach über
  seine PID beendet; kein qmd-Prozess blieb.

**4. `dev bench hooks` und `dev bench repos` mit `--out`**, 2026-09-27
gegen 01:14 Ortszeit (Stempel `2026-09-26-2314`, UTC), Go 1.27.0. Beide
Läufe schrieben beide Dateien.

- **`hooks`** über einen Fall im Scratch, den Wächter `pre-tool-use` bei
  einem Edit in diesem Worktree, `-n 10`: kalt 14,5 ms, warm Median 9,0 ms,
  Min 8,5 ms, Max 9,5 ms, Exit `[0]`.
- **`repos --dir . --warm 3`** auf diesem Worktree (Beispieldatei
  `cmd/loomux/main.go`):

| Komponente | kalt | warm Median | Min | Max |
|---|---:|---:|---:|---:|
| `pre-tool-use` | 11,5 ms | 10,5 ms | 9,7 ms | 10,5 ms |
| `post-tool-use` | 787,2 ms | 791,2 ms | 783,6 ms | 844,4 ms |
| `graph build` | 675,5 ms | 679,2 ms | 677,0 ms | 687,6 ms |
| gesamt | 1.474,2 ms | 1.489,3 ms | 1.472,5 ms | 1.531,9 ms |

  Die Hülle trägt `schema` 1 und `command` `repos`; die `payload` hat die
  Schlüssel `repos` und `skipped`.

**Offen:**

- **Eine saubere Qualitätszahl im Alltag.** Sie braucht entweder einen
  funktionierenden CUDA-Weg auf diesem Rechner oder einen Index, der unter
  demselben Backbone eingebettet ist, unter dem gesucht wird (also eine neue
  Einbettung).
- **Backbone in `QmdPort`.** loomux gibt qmd kein Backbone mit; die Referenz
  setzte `QMD_LLAMA_GPU=vulkan` fest. Ob loomux das übernimmt, entscheidet der
  Nutzer.
