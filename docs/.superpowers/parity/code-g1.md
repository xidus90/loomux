# Paritätsliste Code-Graph G1

**Quelle:** `trailhq/Graft` @ `1e352a3` (MIT), `src/ask/graphrank.ts` und
`src/graph/traverse.ts`.
**Spec:** [2026-09-16-loomux-code-g1-delta.md](../specs/2026-09-16-loomux-code-g1-delta.md)
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers, bevor die Stufe als
fertig gilt.

**Stand 2026-09-16.** Das Tor ist grün:
`go test ./... -count=1 -covermode=set -coverpkg=./... -coverprofile=coverage.out`
gefolgt von `go run ./cmd/loomux dev covergate --profile coverage.out` meldet
nichts und endet mit 0 — keine Funktion in `internal/code/**` liegt unter
100 %, und kein `//coverage:exempt` steht in den drei Paketen.

Die Mutationsrunde ist
`go run ./cmd/loomux dev mutants internal/code/pagerank internal/code/blast`.
Sie lief zweimal. **Runde 1** zählte 97 Mutanten (77 über `pagerank`, 20 über
`blast`), davon 6 ohne Kompilat, 76 getötet und **15 überlebt**. Vier dieser
Überlebenden waren fehlende Tests; sie sind unten als Zeilen 1 bis 4 geführt
und mit drei neuen Tests in `internal/code/pagerank/rank_test.go` erledigt.
**Runde 2** zählt dieselben 97 Mutanten, 6 ohne Kompilat, **80 getötet und 11
überlebt**. Alle 11 sind äquivalente Mutanten: sie ändern den Code, aber nicht
sein beobachtbares Ergebnis, und kein Test kann sie töten. Sie stehen unten als
Zeilen 5 bis 15 mit der Rechnung, die das zeigt.

Keine Zeile ist bisher freigegeben.

## Verfügungen dieser Stufe, die keine Mutation betreffen

Drei Entscheidungen der Stufe stehen hier, weil sie die Lesart der Tabelle
tragen und nicht aus einer Mutante folgen:

- **Unaufgelöstes Importziel.** Der Lauf (`blast.Reach`) behält ein Ziel, das
  kein Knoten des Graphen ist — ein Import, der sein Modul nennt —, als Treffer
  mit `Node == nil`; er versteckt die Abhängigkeit nicht. Der Rang
  (`pagerank.Prepare`) verwirft dieselbe Kante, weil ein solcher Eintrag sonst
  Rangmasse sammelte und als Ergebnis gemeldet würde, das niemand öffnen kann.
  Die beiden Seiten weichen hier bewusst voneinander ab.
- **Selbstschleife und Parallelkanten.** Beide werden nicht zusammengefasst,
  sondern wirken als Kantenvielfachheit: eine doppelte Kante zählt in der
  Nachbarliste doppelt und teilt die Masse entsprechend. Das entspricht der
  Referenz.
- **Nicht normalisierte Eingaben.** Ein negativer `Depth`-Wert, der nicht `All`
  ist, und eine `Direction` außerhalb von `In`/`Out` werden nicht
  zurechtgebogen. Grund: Keine Nutzereingabe erreicht diese Typen bisher; die
  CLI-Verdrahtung steht aus (G2). Sobald sie kommt, gehört die Normalisierung
  dorthin, nicht in `Reach`.

## Die Runde

| Ort | Mutation | Ausgang | Verfügung | Freigabe |
|---|---|---|---|---|
| `rank.go:26` (a2) | `if o.Alpha <= 0 \|\| o.Alpha >= 1 {` → `if o.Alpha <= 0 {` | Runde 1 überlebt, Runde 2 getötet | Fehlender Test: kein Fall reichte ein `Alpha` ≥ 1. Nachgezogen als `TestRankAlphaAtOrAboveOneFallsBackToTheDefault` — `Alpha: 1` und `Alpha: 1.5` müssen dasselbe liefern wie `Options{}` | offen |
| `rank.go:26` (a3) | `if o.Alpha <= 0 \|\| o.Alpha >= 1 {` → `if o.Alpha <= 0 \|\| o.Alpha > 1 {` | Runde 1 überlebt, Runde 2 getötet | Derselbe fehlende Test: `Alpha: 1` genau auf der Grenze. Derselbe neue Test tötet beide Zeilen | offen |
| `rank.go:33` (a1) | `if o.Iterations <= 0 {` → `if true {` | Runde 1 überlebt, Runde 2 getötet | Fehlender Test: kein Fall machte eine gesetzte Schrittzahl am Ergebnis sichtbar. Nachgezogen als `TestRankIterationsLimitHowFarTheWalkSpreads` — auf der Kette `a-b-c` erreicht `Iterations: 1` das `c` nicht, die Vorgabe schon | offen |
| `rank.go:56` (a2) | `if !ok \|\| w <= 0 {` → `if !ok {` | Runde 1 überlebt, Runde 2 getötet | Fehlender Test: ein negatives Gewicht wurde nie neben einem positiven gereicht, wo es die Summe löschte. Nachgezogen als `TestRankNegativeSeedWeightIsIgnoredNotSubtracted` — Saat `{hub: 1, a: -1}` muss `hub` mit 1 liefern, nicht nichts | offen |
| `rank.go:56` (a3) | `if !ok \|\| w <= 0 {` → `if !ok \|\| w < 0 {` | überlebt | Äquivalent: `restart` ist frisch mit Nullen belegt. Ein Gewicht von genau 0 schreibt der Mutant als 0 an eine Stelle, die schon 0 ist — kein beobachtbarer Unterschied. Kein Test kann das töten | offen |
| `rank.go:68` (a1) | `if total <= 0 {` → `if false {` | überlebt | Äquivalent: `total` ist die Summe echt positiver Gewichte, also 0 oder größer. Bei 0 teilt der Mutant 0 durch 0, jeder Rang wird `NaN`, `max` bleibt 0 und die Wache `max <= 0` liefert dasselbe `nil`. Die frühe Wache spart nur den Umweg. `TestRankSeedWeightBeyondTheFloatRange` hält diesen zweiten Weg bereits fest | offen |
| `rank.go:68` (a3) | `if total <= 0 {` → `if total < 0 {` | überlebt | Äquivalent, dieselbe Rechnung: `total` wird nie negativ, also greift die Wache nur bei 0, und der `NaN`-Weg endet ebenso in `nil` | offen |
| `rank.go:102` (a1) | `if dangling > 0 {` → `if true {` | überlebt | Äquivalent: `dangling` ist eine Summe nichtnegativer Massen. Bei 0 ist `dm` 0, und die Schleife addiert `0 * r` auf jeden Eintrag. Die Wache spart Arbeit, sie ändert nichts | offen |
| `rank.go:102` (a3) | `if dangling > 0 {` → `if dangling >= 0 {` | überlebt | Äquivalent, dieselbe Rechnung wie die Zeile darüber | offen |
| `rank.go:105` (a1) | `if r > 0 {` → `if true {` | überlebt | Äquivalent: `r` ist ein normiertes Restart-Gewicht und nie negativ. Bei `r == 0` addiert der Mutant `dm * 0` — die Wache überspringt nur die Nicht-Saatknoten | offen |
| `rank.go:105` (a3) | `if r > 0 {` → `if r >= 0 {` | überlebt | Äquivalent, dieselbe Rechnung wie die Zeile darüber | offen |
| `rank.go:115` (a3) | `if v > max {` → `if v >= max {` | überlebt | Äquivalent: Bei Gleichstand weist der Mutant `max` denselben Wert erneut zu. Das Maximum einer Folge hängt nicht davon ab, welcher der gleichen Werte es setzt | offen |
| `rank.go:131` (a3) | `return out[i].Score > out[j].Score` → `return out[i].Score >= out[j].Score` | überlebt | Äquivalent: Die Zeile wird nur erreicht, wenn die Wache in Zeile 130 (`out[i].Score != out[j].Score`) schon festgestellt hat, dass die beiden Werte verschieden sind. Für ungleiche Werte sind `>` und `>=` dasselbe | offen |
| `rank.go:133` (a3) | `return out[i].ID < out[j].ID` → `return out[i].ID <= out[j].ID` | überlebt | Äquivalent: Die Zeile wird nur bei gleichem Score erreicht, und `Prepare` vergibt jede ID genau einmal — gleich sind zwei IDs also nur, wenn `i` und `j` dasselbe Element meinen, und wie ein Element zu sich selbst steht, ist für die Reihenfolge ohne Belang | offen |
| `reach.go:20` (a1) | `if visited[id] {` → `if false {` | überlebt | Äquivalent: Eine doppelt genannte Start-ID landet zweimal in der Front, aber `visited[id]` steht vorher schon. Beim zweiten Durchgang über denselben Knoten ist jeder Nachbar bereits besucht, es entsteht kein Treffer. Gleiche Treffer, gleiche Reihenfolge — die Wache spart nur den Doppelgang | offen |

## Was G1 offen lässt

An G2 übergeben, unverändert: die CLI-Verdrahtung (`loomux graph build|check`),
der Abschnitt `[graph]` in `.loomux/config.toml`, `docs/{en,de}/cli-reference.md`,
die beiden READMEs und ein Eintrag in `docs/{en,de}/benchmarks.md` für die
Dangling-Messung — gepoolt ~9 ms gegen ~4,5 s pro Dangling-Knoten auf einem
Graphen mit 20k Knoten, gemessen 2026-09-16 auf einem AMD Ryzen 7 9800X3D, nur
warm; eine kalte Zahl steht noch aus. Diese Dateien ändert Stufe 1b-1 parallel;
sie werden nachgeholt, sobald 1b-1 nach `master` gegangen ist und `code-g1`
darauf steht.
