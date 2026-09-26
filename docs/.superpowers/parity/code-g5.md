# Akte G5a — Python im Code-Graphen

**Stand:** 2026-09-26. Abnahme nach §8 der Spec
[`2026-09-26-loomux-code-g5-design.md`](../specs/2026-09-26-loomux-code-g5-design.md),
Plan [`2026-09-26-loomux-code-g5a.md`](../plans/2026-09-26-loomux-code-g5a.md).
Binär vom Zweig `feat/graph-python` (`69ce9fde`, Go 1.27.0, gotreesitter
v0.55.0). Eine alte Implementierung zum Vergleichen gibt es nicht: die
Referenz ist der Quelltext der Repos selbst, per `grep` gegengezählt.

**Ort der Abnahme.** Nur an Klonen im Scratchpad der Sitzung
(`git clone --local`), nie in den Repos selbst, denn `graph build` schreibt
`.loomux/state/graph/` in die Wurzel:

| Repo | Stand des Klons | Inhalt |
|---|---|---|
| `iam_backend` | `117cba9` | Django, 399 Python-Dateien |
| `ultra-brain` | `3cc72d2` | 174 Go- und 182 Python-Dateien (gemischt) |

## 1. Build ohne Abbruch

| Repo | Bericht von `graph build` (kalt) |
|---|---|
| `iam_backend` | 399 files, 4707 nodes, 10534 edges (4308 contains, 5147 calls, 965 imports); `python: 399 files, 399 parsed, 0 reused, 0 parse errors` |
| `ultra-brain` | 356 files, 4974 nodes, 16053 edges (4618 contains, 9962 calls, 1465 imports); `go: 174 files …, 0 parse errors`, `python: 182 files …, 0 parse errors` |

Bestanden.

## 2. `callers`, `ask`, `blast` an je drei Symbolen

Gewählt nach Aufruferzahl aus dem Graphen, je eine Methode mit
`self`-Aufrufern, eine per `from … import` genutzte Funktion und ein
Konstruktor (`__init__`). Gegenprobe: dieselben Aufrufstellen per `grep`.

| Repo | Symbol | `graph callers -d 1` | `grep` | Urteil |
|---|---|---:|---:|---|
| `iam_backend` | `ProjectViewSet._is_project_lead_or_admin` | 15 | 17 | stimmt: die zwei übrigen Treffer (`projects.py:2244`, `:2318`) rufen eine gleichnamige Modulfunktion, nicht die Methode |
| `iam_backend` | `generate_api_key` (`apps/ml_api/services/key_service.py`) | 31 | 31 | stimmt, 12 Dateien, alle `extracted` |
| `iam_backend` | `ImageScannerService.__init__` | 9 | 9 | stimmt, 6 Dateien (Konstruktor über `ImageScannerService()`) |
| `ultra-brain` | `QmdPort._invoke` | 5 | 5 | stimmt |
| `ultra-brain` | `read_manifest` (`src/brain/manifest.py`) | 79 | 79 | stimmt |
| `ultra-brain` | `OllamaClient.__init__` | 24 | 24 | stimmt |

`graph ask`:

- `iam_backend`, „generate an api key for a worker“ → Platz 1
  `key_service.py#generate_api_key`, danach `WorkerRegisterView`,
  `worker.py`, `WorkerKeyViewSet`, `WorkerKeySerializer`.
- `ultra-brain`, „read the manifest of an area“ → Platz 1
  `src/brain/manifest.py#read_manifest`, danach die beiden Go-Funktionen
  `ReadManifest` und `readManifest` — über Sprachgrenzen gerankt, ohne
  Kanten über Sprachgrenzen.

`graph blast` nach einer Änderung an der Kopfzeile der Funktion im Klon:

- `generate_api_key`: `key_service.py [stale]`, Seed mit In-Grad 31, die
  zehn Testdateien unter `apps/ml_api/tests/` erkannt, die Aufrufer in den
  `setUp`-Methoden erreicht.
- `read_manifest`: `manifest.py [stale]`, Seed mit In-Grad 79, 24
  Testdateien erkannt, darunter `tests/conftest.py`.

Bestanden. Beobachtet: fast alle Aufrufkanten sind `extracted`; die Klasse
mit den meisten Aufrufern ist `UserFactory` (407, factory_boy).

## 3. Warm gleich kalt

Ein zweiter Build parst 0 Dateien neu (`iam_backend`: 399 reused,
`ultra-brain`: 174 + 182 reused). Der warme Build und ein Build mit
`--no-reuse` schreiben byte-gleiche `wiring.json` und `ask-index.json`
(`cmp`) — in beiden Repos. Bestanden.

## 4. Der Go-Graph unverändert

Das neue Binär baut über einen Worktree am Commit `b5c99cf1` eine
`wiring.json` und eine `ask-index.json`, die gegen die Grundlinie des alten
Binärs byte-gleich sind bis auf die Zeile `extractor`
(`go/1+python/1@gotreesitter/v0.55.0`); `languages` bleibt `["go"]`.
Bestanden. Dieselbe Probe lief nach jeder Aufgabe, die den Go-Pfad
berührte, jedes Mal ohne Cache.

## 5. Nachtrag nach der Korrekturrunde

Nach der Durchsicht des ganzen Zweigs kamen dazu: die Testpfade `tests.py`
und `test/`, Re-Exporte über den eigenen From-Import eines Moduls, die
Quellwurzeln über jedem Paket oberster Ebene und an jeder `manage.py`, die
engere Regel für „außerhalb“ und `site-packages` in der Dateimenge. Beide
Repos neu gebaut, auf frischen `git clone --local` der Abnahme-Klone im
Scratchpad, mit dem Binär vorher (`6599415e`) und nachher (`bb0da34`, der
Code nach der Runde):

| Repo | vorher | nachher |
|---|---|---|
| `iam_backend` | 10534 Kanten, davon 114 `extends` | Kante für Kante gleich |
| `ultra-brain` | 16053 Kanten | 16337: +26 `imports`, +220 `calls` `extracted`, +38 `calls` `inferred`; keine Kante fällt weg |

Die neuen Kanten in `ultra-brain`: `tests/` hat kein `__init__.py`, seine
Unterpakete schon. `tests/` ist damit eine Quellwurzel, so wie pytest im
Importmodus `prepend` das erste Verzeichnis ohne `__init__.py` in den
Suchpfad legt, und `from conftest import …` oder
`from maintenance.test_reconcile import …` lösen auf, vor allem auf die
Helfer in den `conftest.py`. Die `inferred`-Kanten stammen aus Bench-Skripten,
die ein Geschwistermodul über den bloßen Namen importieren
(`from richter import ist_deutsch`): früher „außerhalb“, jetzt eindeutig
geraten.

In `iam_backend` ist jede geschriebene Basis ohne Kante, die den Namen einer
Repo-Klasse trägt, ein `migrations.Migration` — Djangos eigene Klasse, zu
Recht ohne Kante. Die sechs Aufruferzahlen aus Abschnitt 2 sind nachher
dieselben (15, 31, 9, 5, 79, 24).

## 6. Nachprüfung: Quellwurzeln der importierenden Datei

Vor dem Merge kam die wurzelrelative Modulsuche dazu (Spec §7, E7): ein
absoluter Modulpfad löst zuerst über die Quellwurzeln auf, die die
importierende Datei enthalten, die tiefste zuerst, und erst danach über die
Tabelle aller Wurzeln mit ihrer Mehrdeutigkeitsregel. Vorher löschte ein
Pfad, den zwei Wurzeln verschieden abbilden, die Kante für jede Datei des
Repos. Vier nachgebaute Proben, je mit `graph build --root .` von Grund auf,
vorher (`52a783d2`) und nachher:

| Probe | Aufbau | vorher | nachher |
|---|---|---|---|
| b0 | `config/` mit `load()`, `main.py` importiert `config.settings` und `load` | 3 Kanten | dieselben 3 |
| b | b0 und eine Kopie unter `examples/demo/config/` | 0 Kanten | die 3 von b0 |
| c | `tests/` und `tools/cli/tests/` (ohne `tools/cli/__init__.py`), je `helpers.make()` und ein Test, der es importiert und ruft | 0 Kanten | 4: jeder Test importiert und ruft die Helfer neben sich |
| d | `backend/manage.py` und `main.py` an der Wurzel, beide `from apps.x import helper` | 4 Kanten auf `backend/apps/x.py` | dieselben 4 (`main.py` über die Tabelle aller Wurzeln) |

Die Abnahme-Repos neu gebaut, auf frischen `git clone --local` der
Abnahme-Klone im Scratchpad (`iam_backend` `117cba9d`, `ultra-brain`
`3cc72d23`), `.loomux/state` vor jedem Bau gelöscht:

| Repo | vorher (`52a783d2`) | nachher |
|---|---|---|
| `iam_backend` | 10534 Kanten, davon 114 `extends` | `wiring.json` byte-gleich |
| `ultra-brain` | 16337 Kanten | `wiring.json` byte-gleich |

Die Zahlen vorher sind die aus Abschnitt 5. Keines der beiden Repos hat
einen Modulpfad, den zwei Quellwurzeln verschieden abbilden; die Regel
ändert dort nichts. Die Goldens `testdata/cases/graph/python` und
`testdata/cases/graph/mixed` bleiben byte-gleich, der Go-Graph über
`b5c99cf1` ebenso (Abschnitt 4).

## Offene Lücken

- **Klassen-Seed im Blast-Audit** (Ruling 20, Spec §7): Ändert sich eine
  Python-Klasse ohne eigenes `__init__` oder ein Klassenattribut, ist der
  Seed der Klassenknoten, und das Testsignal bleibt `na` — nur Funktionen
  und Methoden zählen als Verhalten. Die Abnahme wählte darum Konstruktoren
  mit `__init__`.
- **Zeilenhinweis bei Konstruktor-Kanten:** `graph callers` zeigt für eine
  Kante, die über `Klasse()` an `__init__` geht, keine Aufrufzeile oder die
  `def __init__`-Zeile des Aufrufers, weil der Hinweis nach dem Zielnamen
  sucht.
- **Basissuche:** breit statt C3 (Spec §7). Beide weichen nur ab, wenn eine
  spätere Basis nicht von der Basis einer früheren abstammt: bei `D(B, C)`,
  `B(A)`, `A.f` und `C.f`, `C` nicht von `A` abgeleitet, zeigt die
  `extracted`-Kante auf `C.f`, Python ruft `A.f`. In einer echten Raute
  (`C(A)`) wählen beide `C.f`.
- **Subskribierte Basen:** `class C(Base[T])` gibt keine `extends`-Kante.
- **Kein Überschatten durch Geltungsbereiche:** ein Parameter, der wie ein
  importiertes Modul heißt, löst als dieses Modul auf.
- **Geerbtes `__init__`:** eine Unterklasse ohne eigenes `__init__` schickt
  Konstruktoraufrufe an den Klassenknoten, nicht an das geerbte `__init__`
  (Spec §7, Regel 5, so entschieden).
