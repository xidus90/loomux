---
type: Topic
title: Suche, Profile und Messwerte
description: Die drei Suchprofile, die Suchleiter, die Latenzbudgets und was die Messungen an Annahmen umgeworfen haben.
open_conflicts: 1
realization: implemented
implemented_in: f740c8f
sources:
  - id: cli-referenz
    resource: brain://project/loomux/docs/de/cli-reference.md
    doc_id: 01M47W91R5FRSPN6BE643656ZE
    content_hash: "sha256:2e0f9455c00bbd2cd943d2eed65463448522d33232f032358addb56c2bae1255"
    revision: 1
  - id: benchmarks
    resource: brain://project/loomux/docs/de/benchmarks.md
    doc_id: 01M47W91R5MPENKN7N34H6QTNT
    content_hash: "sha256:73cf99840e93d14ff9d4cc7a385fb33fe7f8733867d1ee8523a43faeb943c1d8"
    revision: 1
---

## Drei Profile

| Profil | Kette |
|---|---|
| `keyword` | BM25, kein Modell |
| `fast` | rein vektoriell: Embedding, **keine** Frageerweiterung, **kein** Reranker |
| `full` | hybrid: Frageerweiterung + Embedding + Reranking |

Die drei Ketten gelten weiter; bis zum Umzug am 2026-09-16 wurden sie über
qmds Kommandozeile gefahren, loomux stellt dieselbe Wahl dem MCP-Daemon
(`searches:[{type:"vec"}]` beziehungsweise `{"type":"lex"}` mit
`rerank:false`, `full` mit `rerank:true`) und meldet auf stderr einmal je Port
den Aufwärm-Hinweis, wenn es den Daemon selbst gestartet hat.

**Vorgabe ist `fast`, überall** — und diese Vorgabe wurde zweimal gedreht, beide
Male auf zu dünner Grundlage, bevor sie gemessen wurde.

| Messung | `keyword` | `fast` | `full` |
|---|---|---|---|
| 30 Fragen, ein Bereich (Scheibe 2b) | 9/30 | **24/30**, dreimal | 22/30, einmal 23 |
| 50 Fragen, vier Bereiche (echter Vault) | 11/50 | 26/50, dreimal | **41/50**, dreimal |

Die zweite Zeile steht gegen die erste, und der Unterschied ist zu groß für
Rauschen. Geändert haben sich Fragensatz (30 → 50), Bestand (ein Bereich → vier)
und die Zusammensetzung der Sorten; die 2b-Zahl ist damit **überholt, nicht
widerlegt**. Am auffälligsten ist die sprachübergreifende Spalte: dort findet
`full` 14/14, `fast` nur 9/14.

**Gewechselt wurde die Vorgabe trotzdem nicht** — aus dem verbliebenen Grund:
95 ms gegen 5351 ms je Frage, das 57-fache, weil die Frageerweiterung je Anfrage
ein 1,7-Milliarden-Modell fährt. Aufgeteilt gemessen zeigt sich zudem, dass der
**Reranker** die Treffer holt (Vektor + Reranking: 40/50 in 3952 ms) und die
Frageerweiterung nur Zeit kostet. Geprüft und verworfen wurde, `full` auf
Vektor + Reranking umzudefinieren: roh gemessen sah das frei aus, kostete
durch die volle Kette aber fünf Fragen.

Die Abwägung dahinter — Latenz gegen Trefferqualität — steht ausführlich in
[Warum `fast` die Vorgabe bleibt, obwohl `full` mehr findet](../syntheses/warum-fast-die-vorgabe-bleibt.md).

Zwei weitere Befunde derselben Messreihe:

- **Der Reranker ist auf diesem Rechner instabil.** Vier von neun `full`-Läufen
  brachen mitten im Lauf mit einem CUDA-Fehler ab; kein einziger rein
  vektorieller Lauf stürzte ab. Kein Argument über Trefferqualität, aber eines
  über die Vorgabe.
- **Das frühere `fast` gab es nie.** Die Kette „Frageerweiterung ohne Reranker"
  existiert bei qmd 2.8.3 nicht — die Definition war eine ungeprüfte Annahme
  über ein fremdes Werkzeug. Damit fiel auch ein viertes Profil weg, das
  denselben Befehl aufrief. Der Plan der Scheibe 0
  führt diese alte Definition noch und leitet daraus **zwei** getrennte
  Nachfolgeläufe ab; unter qmd 2.8.3 sind beide derselbe Aufruf, und der eine
  verbliebene Lauf beantwortet beide Fragen.

## Die Sprachbrücke liegt im Einbettungsmodell

Ursprünglich stand in der Spec, das englischoptimierte Standardmodell werde
sprachübergreifend abfallen. Gemessen: `fast` 8/8, `full` 8/8, `keyword` 0/8.
Die reine Vektorsuche findet ohne Erweiterung und ohne Reranker jede
sprachübergreifende Frage in beiden Richtungen. Tragend ist damit das
**Einbettungsmodell**, nicht der Reranker — und ein Tausch dieses Modells
verlangt dieselbe Messung erneut.

## Die Suchleiter

1. `catalog` lesen. 2. `search` mit `layer=wiki`. 3. `search` mit `layer=raw`.
4. `read` — genau **eine** Datei, darin nur den relevanten Abschnitt.
5. Antworten, mit Quellenangabe.

Grenzen: höchstens zwei Suchläufe je Frage; ein hoher Relevanzwert ist kein
Beweis; Scope nur bewusst weiten und das Weiten benennen; Widersprüche zitieren,
nicht auflösen. Die Zwei-Lauf-Grenze ist die Antwort auf den Kostentreiber:
Jeder weitere Suchschritt liest den bisherigen Gesprächsverlauf erneut.

**Stufe 2 heißt vorerst lesen, nicht suchen** — bei rund 290 Notizen passt das
verdichtete Wiki mit hoher Wahrscheinlichkeit ins Kontextfenster. Die Schwelle
wird gemessen, nicht geschätzt: `status` weist die Tokenmenge der Wiki-Schicht
je Bereich aus und meldet, wenn sie 100.000 überschreitet.

> [!conflict] Meldet `status` die Tokenmenge der Wiki-Schicht?
> Der Absatz oben (aus dem Architektur-Entwurf des Vorgängers verdichtet) sagt:
> `status` weist die Tokenmenge der Wiki-Schicht je Bereich aus und meldet,
> wenn sie 100.000 überschreitet.
> `docs/de/cli-reference.md`, Abschnitt `loomux brain status`, zählt die Zeilen
> des Befehls vollständig auf — letzter Abgleich, Include-Globs, fehlende
> Pfade, nie indizierte Bereiche, aufgelöste Links, unbekannte Dokumente,
> doppelte Inhalte, nicht durchsuchbare Dokumente — und nennt keine
> Tokenmenge; auch der Code von `internal/brain/status` kennt keine.
> Beide Stände bleiben stehen. Entscheidung offen.

**Die Suchleiter rechtfertigt sich heute nicht über Effizienz.** Sechs Läufe mit
und ohne Regelwerk fanden alle die richtige Datei, bei 14 gegen 13
Werkzeugaufrufen. Ihr Nutzen liegt in der Verlässlichkeit des Vorgehens und
darin, dass sie mit dem Bestand skaliert, während die Grundlast konstant bleibt.
Der Prüfpunkt für später steht: dieselbe Messung erneut, sobald der Bestand
deutlich gewachsen ist.

## Latenzbudgets

| Operation | Budget warm | gemessen warm |
|---|---|---|
| Katalog, Scope, Datei lesen | < 10 ms | 3 ms ✅ |
| Stichwortsuche | < 30 ms | 21 ms ✅ |
| Vektorsuche (`fast`) | < 20 ms | 78 ms ❌ (3,9-fach) |
| Daemonstart Stufe 1 / Stufe 2 | eigener Messwert | 1132 ms / 1219 ms warm, 9982 ms kalt |

Die verfehlte Zeile steht ausdrücklich **als Verfehlung** da und nicht als
angepasstes Budget: Ein Budget, das sich der Messung anpasst, ist keines mehr.

Die Zahlen oben sind die der Kommandozeile bis zum Umzug. Am 2026-09-16 hat
loomux Ende zu Ende nachgemessen (`docs/de/benchmarks.md`): `brain search
--profile fast` warm 260,7 ms Median gegen einen Zielwert von 150 ms, mit einer
warmen Spanne von 236,0 bis 373,7 ms, die ihn nie erreicht — die Verfehlung
bleibt also stehen, nur größer und an einem anderen Maßstab.

Zwei Zahlen haben Konstruktionsentscheidungen getragen:

- **Der Prozessstart von qmd kostet 278–310 ms** — das Neunfache des
  Stichwortsuch-Budgets, bevor ein Zeichen gesucht wurde. Ein Prozessstart je
  Anfrage ist damit ausgeschlossen; qmd wird als gehaltener Unterprozess
  gefahren ([qmd](../entities/qmd.md)).
- **Eine kalte Bedeutungssuche dauerte 27,1 s Frageerweiterung, 1,8 s
  Frage-Embedding, 9,4 s Reranking** — zwei Größenordnungen neben den
  ursprünglich angesetzten 150–400 ms. Ursache war ein Denkfehler, kein
  Messfehler: das Erweiterungsmodell war nicht als eigener Ladeposten gerechnet.
  Daraus folgt: **Der Daemon ist keine spätere Bequemlichkeit, sondern die
  Voraussetzung dafür, dass die volle Kette im Alltag benutzbar ist**
  ([Der brain-Daemon](../entities/brain-daemon.md)).

## Der Relevanzwert ist ein Abstandsmaß

Auf einer Bedeutungssuche ist „nichts gefunden" gar nicht erreichbar: Eine
Anfrage nach `zzqqxwvk-nichtvorhanden-42` lieferte auf `full` fünf Treffer, den
ersten mit **75 %**. Eine Schwelle, unterhalb derer unterdrückt wird, wurde
erwogen und verworfen — sie schwankt mit Bestand und Frageart, und ein
unterdrückter Treffer ist ein stiller Verlust.

Davon unberührt bleibt die Anforderung, **leeres Ergebnis von fehlgeschlagener
Suche** zu unterscheiden: qmd bricht gelegentlich still ab und liefert einen
leeren Treffersatz — ein Fehler, der sich als Ergebnis tarnt, und die
gefährlichste Sorte, weil „nichts gefunden" plausibel klingt und niemand sie
nachprüft.

Die Suche im Quelltext statt im Wissen beschreibt
[Der Code-Graph](code-graph.md).

Quellen: CLI-Referenz und Benchmarks der Nutzerdoku.
