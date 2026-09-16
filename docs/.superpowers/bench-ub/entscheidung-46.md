# Entscheidung 46 gemessen: `fast` gegen `full`, gleicher Satz, gleicher Bestand

Gemessen am 2026-08-22 gegen den echten Vault: 50 Fragen aus
`98 Messung/questions.yaml`, alle vier registrierten Bereiche (276 indexierte
Dokumente), qmd 2.8.3, Rückgrat `vulkan`, warm über den Daemonpfad. Fragensatz
und Bestand blieben über alle Läufe unverändert — genau der Vorbehalt, der die
Entscheidung offen gemacht hatte.

## Durch die volle Kette (`brain bench`, drei Läufe je Profil)

| Profil | Lauf 1 | Lauf 2 | Lauf 3 | je Frage (Median) |
|---|---|---|---|---|
| `fast` | 26/50 | 26/50 | 26/50 | 93–97 ms |
| `full` | 41/50 | 41/50 | 41/50 | 5351–5429 ms |

Dreimal dieselbe Zahl auf beiden Seiten: **fünfzehn Fragen Unterschied ist
kein Rauschen.** `full` findet, was `fast` nicht findet — und braucht dafür
das **57-fache** an Zeit je Frage.

Die Sortenaufschlüsselung zeigt, wo:

| Sorte | `fast` | `full` |
|---|---|---|
| exakt | 6/13 | 11–12/13 |
| umschreibung | 6/13 | 8–9/13 |
| gemischt | 5/10 | 7–8/10 |
| sprachübergreifend | 9/14 | 13–14/14 |

## Wofür genau die fünf Sekunden gezahlt werden

`full` ist zweierlei: Frageerweiterung **und** Reranking. Dieselben 50 Fragen,
direkt an die Suchmaschine, alle drei Ketten über denselben warmen Daemon
([`measure_profile_split.py`](measure_profile_split.py)):

| Kette | Treffer | je Frage (Median) |
|---|---|---|
| `vec`, kein Reranking (= `fast`) | 27/50 | 115 ms |
| **`vec` + Reranking** | **40/50** | **3952 ms** |
| `query` (Erweiterung) + Reranking (= `full`) | 39/50 | 5364 ms |

**Der Reranker holt die Treffer. Die Frageerweiterung kostet nur Zeit.**
Dreizehn der vierzehn zusätzlichen Treffer bringt allein das Reranking; die
Erweiterung legt darüber 1412 ms und **einen Treffer weniger**.

Auf CUDA statt Vulkan gemessen ([`profile-split-cuda.md`](profile-split-cuda.md)):
`vec` + Reranking 40/50 in 3912 ms — der Reranker ist auf beiden Rückgraten
gleich teuer, das Rückgrat ist hier also keine Stellschraube.

## Was daraus folgt

Drei Punkte, von denen der dritte neu ist:

1. **`fast` verliert seine Begründung als „findet besser".** Die Zahl aus 2b
   (24/30 gegen 22/30) galt für dreißig Fragen über *einen* Bereich. Über vier
   Bereiche und fünfzig Fragen ist es umgekehrt, deutlich und wiederholbar.
2. **`fast` behält seine Begründung als „ist benutzbar".** 95 ms gegen 5,4 s
   ist der Unterschied zwischen einer Suche, die man beiläufig stellt, und
   einer, auf die man wartet.
3. **Roh gemessen sah `vec` + Reranking wie das Optimum aus** — 40 gegen 39
   Treffer bei 1,4 s weniger. Der nächste Abschnitt zeigt, warum das ein
   Messfehler war.

## Der Vorschlag wurde umgesetzt und wieder zurückgebaut

Aus der Rohmessung folgte: `full` auf `vec` + Reranking umdefinieren, ein
Treffer mehr für 1,4 s weniger. Das wurde gebaut — und dann gegen die volle
Kette gemessen:

| `full` | durch die Kette | je Frage |
|---|---|---|
| `query` (Erweiterung) + Reranking | **41/50** (dreimal) | 5351 ms |
| `vec` + Reranking | **36/50** (`cuda` und `vulkan`) | 3828–3974 ms |

**Fünf Fragen schlechter, nicht einer besser.** Auf beiden Rückgraten
dieselbe Zahl, es liegt also an der Kette. Die Änderung ist zurückgenommen;
`full` ist wieder Erweiterung plus Reranking.

**Warum die Rohmessung irreführte.** `measure_profile_split.py` vergleicht
`Path(...).name` — den Dateinamen. Die Kette vergleicht den aufgelösten Pfad
(`quality._comparable`). Ein Dateiname wie `index.md` oder `log.md` kommt in
vier Bereichen vor, und die Rohmessung zählt jeden davon als Treffer. Der
Unterschied zwischen 40/50 und 36/50 ist genau diese Großzügigkeit, und sie
trifft die reranking-lastigen Ketten stärker, weil der Reranker gleichnamige
Kandidaten nach oben zieht.

**Die Lehre gehört zum Fund:** eine Abkürzung am Messwerk vorbei misst eine
andere Frage. Die Rohmessung war schnell und hat die Aufteilung sichtbar
gemacht — als Entscheidungsgrundlage taugte sie nicht, und dass sie beide
Male dieselbe Richtung zeigte, hat den Fehler verdeckt.

**Was steht:** `fast` bleibt die Vorgabe (95 ms gegen 5351 ms), `full` bleibt
die Hybridkette und findet 41/50. Der Reranker ist der teure und der
tragende Teil; die Erweiterung kostet 1,4 s und bringt fünf Fragen.

Was diese Messung **nicht** beantwortet: ob die 24 Fragen, die `fast` verfehlt,
im Alltag überhaupt gestellt werden. Der Fragensatz ist für die Suchkette
gebaut, nicht aus Nutzung abgeleitet.
